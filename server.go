package bristle

import (
	"fmt"

	"github.com/rs/zerolog/log"
	v1 "github.com/uplol/bristle/proto/v1"
	"golang.org/x/net/context"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Server struct {
	ingestService          *IngestService
	protoRegistry          *ProtoRegistry
	messageBindingRegistry messageBindingRegistry
	clusters               []*ClickhouseCluster
	config                 *Config
}

func NewServer(config *Config) (*Server, error) {
	protoRegistry := NewProtoRegistry()
	for _, path := range config.ProtoDescriptorPaths {
		err := protoRegistry.RegisterPath(path)
		if err != nil {
			return nil, err
		}
	}

	clusters := []*ClickhouseCluster{}
	for _, clusterConfig := range config.Clusters {
		clusters = append(clusters, NewClickhouseCluster(clusterConfig))
	}

	s := &Server{
		messageBindingRegistry: make(messageBindingRegistry),
		clusters:               clusters,
		protoRegistry:          protoRegistry,
		config:                 config,
	}

	var err error
	s.ingestService, err = NewIngestService(config.Bind, s)
	if err != nil {
		return nil, err
	}

	err = s.bindMessages()
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	s.ingestService.Run(ctx)
	log.Info().Msg("server: started ingest service")

	writersDone := make(chan struct{})
	numWriters := 0

	for _, cluster := range s.clusters {
		for _, table := range cluster.tables {
			tableWriterCount := table.config.Writers
			if tableWriterCount == 0 {
				tableWriterCount = 1
			}

			log.Info().
				Int("count", tableWriterCount).
				Str("table", string(table.Name)).
				Msg("server: starting up table writers")

			for i := 0; i < tableWriterCount; i++ {
				numWriters += 1
				writer := NewClickhouseTableWriter(table)
				go func() {
					writer.Run(ctx)
					writersDone <- struct{}{}
				}()
				table.writers = append(table.writers, writer)
			}
		}
	}

	<-ctx.Done()

	log.Info().Int("writers", numWriters).Msg("server: shutdown requested, waiting for all writers to exit")
	for i := 0; i < numWriters; i++ {
		<-writersDone
	}
	log.Info().Msg("server: all writers have exited, goodbye")

	return nil
}

func (s *Server) bindMessages() error {
	if s.config.Autobind {
		err := s.autobindMessages()
		if err != nil {
			return err
		}
	}

	for _, cluster := range s.clusters {
		for _, table := range cluster.tables {
			for _, messageTypeName := range table.config.Messages {
				messageType := s.protoRegistry.MessageTypes[protoreflect.FullName(messageTypeName)]
				if messageType == nil {
					return fmt.Errorf("message type '%v' is not registered", messageTypeName)
				}

				err := s.BindMessage(cluster, messageType, table.Name)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (s *Server) autobindMessages() error {
	for name, messageType := range s.protoRegistry.MessageTypes {
		messageOpts := messageType.Descriptor().Options()
		if !proto.HasExtension(messageOpts, v1.E_BristleTable) {
			log.Trace().
				Str("name", string(name)).
				Msg("server: skipping auto-registering message missing bristle_table option")
			continue
		}

		fullTableName := ParseFullTableName(
			proto.GetExtension(messageOpts, v1.E_BristleTable).(string),
		)

		matched := false
		for _, cluster := range s.clusters {
			err := s.BindMessage(cluster, messageType, fullTableName)
			if err != nil {
				if err == ErrNoSuchTable {
					continue
				}
				return err
			}
			matched = true
			break
		}

		if !matched {
			return fmt.Errorf("failed to find table %v for message %v", fullTableName, name)
		}
	}
	return nil
}

func (s *Server) BindMessage(cluster *ClickhouseCluster, messageType protoreflect.MessageType, fullTableName FullTableName) error {
	typeName := string(messageType.Descriptor().FullName())

	table, err := cluster.GetTable(fullTableName)
	if err != nil {
		log.Error().
			Err(err).
			Str("message", typeName).
			Str("table", string(fullTableName)).
			Msg("server: failed to find table in cluster")
		return err
	}

	binding, err := table.BindMessage(messageType, table.config.MessageInstancePoolSize)
	if err != nil {
		log.Error().
			Err(err).
			Str("message", typeName).
			Str("table", string(fullTableName)).
			Msg("server: failed to bind message to table")
		return err
	}

	s.messageBindingRegistry[typeName] = binding
	log.Info().
		Str("message", string("name")).
		Str("table", string(fullTableName)).
		Msg("server: successfully bound message")

	return nil
}
