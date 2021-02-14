package bristle

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"
)

type Server struct {
	sync.RWMutex

	configPath    string
	ingestService *IngestService

	// The following can get reloaded
	config                 *Config
	protoRegistry          *ProtoRegistry
	messageBindingRegistry messageBindingRegistry
	clusters               []*ClickhouseCluster
	writerGroup            *writerGroup
}

func NewServer(configPath string) (*Server, error) {
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	s := &Server{
		configPath: configPath,
	}

	err = s.reloadConfig(config)
	if err != nil {
		return nil, err
	}

	s.ingestService, err = NewIngestService(s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) reloadConfig(newConfig *Config) error {
	protoRegistry := NewProtoRegistry()
	for _, path := range newConfig.ProtoDescriptorPaths {
		err := protoRegistry.RegisterPath(path)
		if err != nil {
			return err
		}
	}

	clusters := []*ClickhouseCluster{}
	for _, clusterConfig := range newConfig.Clusters {
		clusters = append(clusters, NewClickhouseCluster(clusterConfig))
	}

	messageBindingRegistry := make(messageBindingRegistry)
	messageBindingRegistry.BindFromClusters(clusters, protoRegistry)

	if newConfig.Autobind {
		messageBindingRegistry.BindFromProtos(clusters, protoRegistry)
	}

	writerGroup := newWriterGroup()
	for _, cluster := range clusters {
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
				writerGroup.Add(NewClickhouseTableWriter(table))
			}
		}
	}

	s.Lock()
	s.config = newConfig
	s.protoRegistry = protoRegistry
	s.clusters = clusters
	s.messageBindingRegistry = messageBindingRegistry

	if s.writerGroup != nil {
		go func() {
			s.writerGroup.Close()
		}()
	}
	s.writerGroup = writerGroup
	defer s.Unlock()

	return nil
}

func (s *Server) Run() error {
	ctx, cancel := context.WithCancel(context.Background())

	s.ingestService.Run(ctx)
	log.Info().Msg("server: started ingest service")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		for {
			sig := <-sigs
			if sig == syscall.SIGINT || sig == syscall.SIGTERM {
				log.Info().Int("signal", int(syscall.SIGINT)).Msg("server: received shutdown signal")
				cancel()
			} else if sig == syscall.SIGHUP {
				log.Info().Msg("server: received SIGHUP, reloading configuration...")

				newConfig, err := LoadConfig(s.configPath)
				if err != nil {
					log.Error().Err(err).Msg("server: configuration reload encountered error on load, no action taken")
					continue
				}

				err = s.reloadConfig(newConfig)
				if err != nil {
					log.Error().Err(err).Msg("server: configuration reload encountered error applying, no action taken")
					continue
				}

				log.Info().Msg("server: configuration reload completed")
			}
		}
	}()

	<-ctx.Done()
	log.Info().Msg("server: exit requested, goodbye")

	return nil
}
