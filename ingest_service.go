package bristle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/rs/zerolog/log"
	v1 "github.com/uplol/bristle/proto/v1"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

type IngestService struct {
	listener net.Listener
	server   *Server
}

func NewIngestService(server *Server) (*IngestService, error) {
	listener, err := net.Listen("tcp", server.config.IngestService.Bind)
	if err != nil {
		return nil, err
	}

	return &IngestService{
		listener: listener,
		server:   server,
	}, nil
}

func (i *IngestService) Run(ctx context.Context) {
	opts := []grpc.ServerOption{}

	transportCredentials, err := i.server.config.IngestService.GetTransportCredentials()
	if err != nil {
		log.Error().Err(err).Msg("ingest-service: failed to load TLS transport credentials")
		panic(err)
	} else if transportCredentials != nil {
		opts = append(opts, grpc.Creds(transportCredentials))
	}

	if i.server.config.IngestService.MaxReceiveMessageSize != nil {
		opts = append(opts, grpc.MaxRecvMsgSize(*i.server.config.IngestService.MaxReceiveMessageSize))
	}

	var metrics bool
	if i.server.config.Debugging != nil {
		metrics = i.server.config.Debugging.Metrics
	}

	if metrics {
		// register the grpc server interceptors for prometheus metrics
		opts = append(opts, grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor), grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor))
	}

	go func() {
		server := grpc.NewServer(opts...)

		v1.RegisterBristleIngestServiceServer(server, i)

		// register the grpc server with grpc_prometheus once the server is created
		grpc_prometheus.Register(server)

		go server.Serve(i.listener)
		<-ctx.Done()
		i.listener.Close()
	}()
}

var ErrNoMessageBindingRegistered = errors.New("no message binding registered for that message type")
var ErrPreparingMessage = errors.New("error occurred when preparing row values for message")

func (i *IngestService) writePayload(payload *v1.Payload) v1.BatchResult {
	i.server.RLock()
	binding, ok := i.server.messageBindingRegistry[payload.Type]
	i.server.RUnlock()
	if !ok {
		return v1.BatchResult_UNK_MESSAGE
	}

	reflectMessage := binding.InstancePool.Get()
	messageInstance := reflectMessage.Interface()
	defer binding.InstancePool.Release(reflectMessage)

	batch := make([][]interface{}, len(payload.Body))
	for idx, encodedMessage := range payload.Body {
		err := proto.Unmarshal(encodedMessage, messageInstance)
		if err != nil {
			return v1.BatchResult_DECODE_ERR
		}

		row := binding.PrepareFunc(reflectMessage)
		if row == nil {
			return v1.BatchResult_TRANSCODE_ERR
		}

		batch[idx] = row
	}

	return binding.Table.WriteBatch(batch)
}

func (i *IngestService) WriteBatch(ctx context.Context, req *v1.WriteBatchRequest) (*v1.WriteBatchResponse, error) {
	for _, payload := range req.Payloads {
		result := i.writePayload(payload)
		if result != v1.BatchResult_OK {
			return nil, fmt.Errorf("WriteBatch error code %v", result)
		}
	}
	return &v1.WriteBatchResponse{
		Dropped:      0,
		Acknowledged: 0,
	}, nil
}

func (i IngestService) writeStreamingBatch(session *StreamingClientSession, batch *v1.StreamingClientMessageWriteBatch) {
	var typeName string
	var ok bool
	switch innerMessageType := batch.MessageType.(type) {
	case *v1.StreamingClientMessageWriteBatch_TypeId:
		session.Lock()
		typeName, ok = session.identifiedMessageTypes[innerMessageType.TypeId]
		session.Unlock()
		if !ok {
			session.WriteBatchResult(batch.Id, v1.BatchResult_UNK_MESSAGE)
			return
		}
	case *v1.StreamingClientMessageWriteBatch_TypeName:
		typeName = innerMessageType.TypeName
	}

	// Grab a binding for the given payload type
	i.server.RLock()
	binding, ok := i.server.messageBindingRegistry[typeName]
	i.server.RUnlock()
	if !ok {
		session.WriteBatchResult(batch.Id, v1.BatchResult_UNK_MESSAGE)
		return
	}

	// Fetch an existing instance of the message type from the bindings pool
	reflectMessage := binding.InstancePool.Get()
	messageInstance := reflectMessage.Interface()
	defer binding.InstancePool.Release(reflectMessage)

	idx := 0
	offset := 0
	messages := make([][]interface{}, batch.Length)
	for {
		if offset >= len(batch.Data) {
			break
		}

		messageData, bytesRead := protowire.ConsumeBytes(batch.Data[offset:])
		offset += bytesRead

		err := proto.Unmarshal(messageData, messageInstance)
		if err != nil {
			session.WriteBatchResult(batch.Id, v1.BatchResult_DECODE_ERR)
			return
		}

		row := binding.PrepareFunc(reflectMessage)
		if row == nil {
			session.WriteBatchResult(batch.Id, v1.BatchResult_TRANSCODE_ERR)
			return
		}

		messages[idx] = row
		idx += 1
	}

	session.WriteBatchResult(batch.Id, binding.Table.WriteBatch(messages))
}

var ErrUnsupported = errors.New("unsupported")

func (i *IngestService) Streaming(stream v1.BristleIngestService_StreamingServer) error {
	session := NewStreamingClientSession(stream, 12)

	for {
		message, err := stream.Recv()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		switch innerMessage := message.Inner.(type) {
		case *v1.StreamingClientMessage_RegisterMessageType:
			descriptor := innerMessage.RegisterMessageType.GetDescriptor_()

			// TODO: support dynamically registering and binding messages
			if len(descriptor) > 0 {
				return ErrUnsupported
			}

			session.Lock()
			session.identifiedMessageTypeIdx += 1
			messageTypeId := session.identifiedMessageTypeIdx
			session.identifiedMessageTypes[session.identifiedMessageTypeIdx] = innerMessage.RegisterMessageType.Type
			session.Unlock()

			session.stream.Send(&v1.StreamingServerMessage{
				Inner: &v1.StreamingServerMessage_IdentifyMessageType{
					IdentifyMessageType: &v1.StreamingServerMessageIdentifyMessageType{
						Type: innerMessage.RegisterMessageType.Type,
						Id:   messageTypeId,
					},
				},
			})
		case *v1.StreamingClientMessage_WriteBatch:
			if acquired := session.batchWriteSem.TryAcquire(1); !acquired {
				session.WriteBatchResult(innerMessage.WriteBatch.Id, v1.BatchResult_TOO_MANY_IN_FLIGHT_BATCHES)
				continue
			}

			go func() {
				defer session.batchWriteSem.Release(1)
				i.writeStreamingBatch(session, innerMessage.WriteBatch)
			}()
		case *v1.StreamingClientMessage_UpdateDefault:
			// TODO: support setting a message default per-client session
			return ErrUnsupported
		}

	}
}

type StreamingClientSession struct {
	sync.Mutex

	stream        v1.BristleIngestService_StreamingServer
	batchWriteSem *semaphore.Weighted

	identifiedMessageTypeIdx uint32
	identifiedMessageTypes   map[uint32]string
}

func NewStreamingClientSession(stream v1.BristleIngestService_StreamingServer, maxConcurrentBatchWrites int64) *StreamingClientSession {
	return &StreamingClientSession{
		stream:        stream,
		batchWriteSem: semaphore.NewWeighted(maxConcurrentBatchWrites),
	}
}

func (s *StreamingClientSession) WriteBatchResult(batchId uint32, result v1.BatchResult) error {
	s.Lock()
	defer s.Unlock()
	return s.stream.Send(&v1.StreamingServerMessage{
		Inner: &v1.StreamingServerMessage_WriteBatchResult{
			WriteBatchResult: &v1.StreamingServerMessageWriteBatchResult{
				Id:     batchId,
				Result: result,
			},
		},
	})
}
