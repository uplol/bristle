package bristle

import (
	"fmt"

	"github.com/rs/zerolog/log"
	v1 "github.com/uplol/bristle/proto/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type messageBindingRegistry map[string]*MessageTableBinding

func newMessageBindingRegistry() messageBindingRegistry {
	return make(messageBindingRegistry)
}

func (m messageBindingRegistry) BindFromClusters(clusters []*ClickhouseCluster, protoRegistry *ProtoRegistry) error {
	for _, cluster := range clusters {
		for _, table := range cluster.tables {
			for _, messageTypeName := range table.config.Messages {
				messageType := protoRegistry.MessageTypes[protoreflect.FullName(messageTypeName)]
				if messageType == nil {
					return fmt.Errorf("message type '%v' is not registered", messageTypeName)
				}

				err := m.bind(cluster, messageType, table.Name)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (m messageBindingRegistry) BindFromProtos(clusters []*ClickhouseCluster, protoRegistry *ProtoRegistry) error {
	for name, messageType := range protoRegistry.MessageTypes {
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
		for _, cluster := range clusters {
			err := m.bind(cluster, messageType, fullTableName)
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

func (m messageBindingRegistry) bind(cluster *ClickhouseCluster, messageType protoreflect.MessageType, fullTableName FullTableName) error {
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

	m[typeName] = binding
	log.Info().
		Str("message", string("name")).
		Str("table", string(fullTableName)).
		Msg("server: successfully bound message")

	return nil
}
