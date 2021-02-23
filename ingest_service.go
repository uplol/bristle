package bristle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/rs/zerolog/log"
	v1 "github.com/uplol/bristle/proto/v1"
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

	go func() {
		server := grpc.NewServer(opts...)

		v1.RegisterBristleIngestServiceServer(server, i)
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

func (i IngestService) writeStreamingBatch(stream v1.BristleIngestService_StreamingServer, batch *v1.StreamingClientMessageWriteBatch) {
	writeResult := func(result v1.BatchResult) {
		stream.Send(&v1.StreamingServerMessage{
			Inner: &v1.StreamingServerMessage_WriteBatchResult{
				WriteBatchResult: &v1.StreamingServerMessageWriteBatchResult{
					Id:     batch.Id,
					Result: result,
				},
			},
		})
	}

	// Grab a binding for the given payload type
	i.server.RLock()
	binding, ok := i.server.messageBindingRegistry[batch.Type]
	i.server.RUnlock()
	if !ok {
		writeResult(v1.BatchResult_UNK_MESSAGE)
		return
	}

	reflectMessage := binding.InstancePool.Get()
	messageInstance := reflectMessage.Interface()
	defer binding.InstancePool.Release(reflectMessage)

	idx := 0
	offset := 0
	messages := make([][]interface{}, batch.Size)
	for {
		if offset >= len(batch.Data) {
			break
		}

		messageData, bytesRead := protowire.ConsumeBytes(batch.Data[offset:])
		offset += bytesRead

		err := proto.Unmarshal(messageData, messageInstance)
		if err != nil {
			writeResult(v1.BatchResult_DECODE_ERR)
			return
		}

		row := binding.PrepareFunc(reflectMessage)
		if row == nil {
			writeResult(v1.BatchResult_TRANSCODE_ERR)
			return
		}

		messages[idx] = row
		idx += 1
	}

	writeResult(binding.Table.WriteBatch(messages))
}

func (i *IngestService) Streaming(stream v1.BristleIngestService_StreamingServer) error {
	for {
		message, err := stream.Recv()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		switch innerMessage := message.Inner.(type) {
		case *v1.StreamingClientMessage_WriteBatch:
			// TODO: limit concurrency
			go i.writeStreamingBatch(stream, innerMessage.WriteBatch)
		}
	}
}
