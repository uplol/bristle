package bristle

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/credentials"
)

type Config struct {
	IngestService        *IngestServiceConfig      `json:"ingest"`
	ProtoDescriptorPaths []string                  `json:"proto_descriptor_paths"`
	Clusters             []ClickhouseClusterConfig `json:"clusters"`
	Debugging            *DebuggingConfig          `json:"debugging"`
	Metrics              bool                      `json:"metrics"`
	LogLevel             string                    `json:"log_level"`

	// Whether messages that contain a `bristle_table` option will be automatically
	//  bound to tables. This functionality searches for tables in all clusters in
	//  the order you defined them. The first cluster to have a table will be used.
	Autobind bool `json:"autobind"`
}

type DebuggingConfig struct {
	Bind                 string `json:"bind"`
	BlockProfileRate     *int   `json:"block_profile_rate"`
	MutexProfileFraction *int   `json:"mutex_profile_fraction"`
	Metrics              bool   `json:"metrics"`
}

type TlsConfig struct {
	Certificate string `json:"certificate"`
	Key         string `json:"key"`
}

type IngestServiceConfig struct {
	Bind                  string     `json:"bind"`
	Tls                   *TlsConfig `json:"tls"`
	MaxReceiveMessageSize *int       `json:"max_receive_message_size"`
}

type ClickhouseClusterConfig struct {
	DSN      string                           `json:"dsn"`
	Tables   map[string]ClickhouseTableConfig `json:"tables"`
	PoolSize int                              `json:"pool_size"`
}

type ClickhouseTableConfig struct {
	// If provided this represents an explicit list of messages we will bind to.
	Messages []string `json:"messages"`

	// Configures the maximum possible size of a batch we will try to commit to
	//  clickhouse.
	MaxBatchSize int `json:"max_batch_size"`

	// Configures the maximum buffer size. This controls the number of messages
	//  that can sit in memory waiting to be written.
	MaxBufferSize int `json:"max_buffer_size"`

	// The interval at which we write to Clickhouse. The higher number the more
	//  latency and potential for your message buffer to fill, however lower numbers
	//  dramatically increase the load of Clickhouse depending on your schema and
	//  disk performance.
	FlushInterval int `json:"flush_interval"`

	// This setting dictates the behavior of the message buffer when its full.
	//   "drop-oldest" will cause the oldest (by time written) messages to be dropped
	//   "drop-newest" will cause new messages to be dropped
	OnFull OnFullBehavior `json:"on_full"`

	// The number of writers to run. Each writer has its own set of buffers and
	//  will back-off messages independently. Writes are distributed amongst writers
	//  via a round-robin strategy.
	Writers int `json:"writers"`

	// Controls the size of the message instance pool, which contains instances
	//  of the proto message interface, for use by the ingest server. This value
	//  is effectively how many concurrent message decodes for each message in this
	//  table
	MessageInstancePoolSize int `json:"message_instance_pool_size"`
}

func DefaultClickhouseTableConfig() ClickhouseTableConfig {
	return ClickhouseTableConfig{
		Messages:      nil,
		MaxBatchSize:  100000,
		MaxBufferSize: 500000,
		FlushInterval: 1000,
		Writers:       1,
		OnFull:        "block",
	}
}

func LoadConfig(path string) (*Config, error) {
	var config Config

	rawData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(rawData, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (i *IngestServiceConfig) GetTransportCredentials() (credentials.TransportCredentials, error) {
	if i.Tls == nil {
		return nil, nil
	}

	serverCert, err := tls.LoadX509KeyPair(i.Tls.Certificate, i.Tls.Key)
	if err != nil {
		return nil, err
	}

	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.NoClientCert,
	}), nil
}

func setLogLevel(logLevel string) {
	switch logLevel {
	case "error":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	}
}
