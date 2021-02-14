package bristle

import (
	"context"
	"errors"
	"net"

	"github.com/rs/zerolog/log"
	v1 "github.com/uplol/bristle/proto/v1"
	"google.golang.org/grpc"
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
	}

	if transportCredentials != nil {
		opts = append(opts, grpc.Creds(transportCredentials))
	}

	go func() {
		server := grpc.NewServer(opts...)

		v1.RegisterBristleIngestServiceServer(server, i)
		// ingestv1.RegisterIngestServiceServer(server, m)
		go server.Serve(i.listener)
		<-ctx.Done()
		i.listener.Close()
	}()
}

var ErrNoMessageBindingRegistered = errors.New("no message binding registered for that message type")
var ErrPreparingMessage = errors.New("error occurred when preparing row values for message")

func (i *IngestService) writePayload(payload *v1.Payload) error {
	i.server.RLock()
	binding, ok := i.server.messageBindingRegistry[payload.Type]
	i.server.RUnlock()
	if !ok {
		return ErrNoMessageBindingRegistered
	}

	reflectMessage := binding.InstancePool.Get()
	messageInstance := reflectMessage.Interface()
	defer binding.InstancePool.Release(reflectMessage)

	batch := make([][]interface{}, len(payload.Body))
	for idx, encodedMessage := range payload.Body {
		err := proto.Unmarshal(encodedMessage, messageInstance)
		if err != nil {
			return err
		}

		row := binding.PrepareFunc(reflectMessage)
		if row == nil {
			return ErrPreparingMessage
		}

		batch[idx] = row
	}

	binding.Table.WriteBatch(batch)
	return nil
}

func (i *IngestService) WriteBatch(ctx context.Context, req *v1.WriteBatchRequest) (*v1.WriteBatchResponse, error) {
	for _, payload := range req.Payloads {
		err := i.writePayload(payload)
		if err != nil {
			return nil, err
		}
	}
	return &v1.WriteBatchResponse{
		Dropped:      0,
		Acknowledged: 0,
	}, nil
}

func (i *IngestService) StreamingWriteBatch(ingestion v1.BristleIngestService_StreamingWriteBatchServer) error {
	for {
		batch, err := ingestion.Recv()
		if err != nil {
			return err
		}

		for _, payload := range batch.Payloads {
			err := i.writePayload(payload)
			if err != nil {
				log.Printf("Failed to write payload: %v", err)
				return err
			}
		}
	}
}
