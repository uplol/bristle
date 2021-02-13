package bristle

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	Bind                 string                    `json:"bind"`
	ProtoDescriptorPaths []string                  `json:"proto_descriptor_paths"`
	Clusters             []ClickhouseClusterConfig `json:"clusters"`

	// Whether messages that contain a `bristle_table` option will be automatically
	//  bound to tables. This functionality searches for tables in all clusters in
	//  the order you defined them. The first cluster to have a table will be used.
	Autobind bool `json:"autobind"`
}

type ClickhouseClusterConfig struct {
	DSN      string                           `json:"dsn"`
	Tables   map[string]ClickhouseTableConfig `json:"tables"`
	PoolSize int                              `json:"pool_size"`
}

type ClickhouseTableConfig struct {
	// If provided this represents an explicit list of messages we will bind to.
	Messages []string `json:"messages"`

	// Describes how large of a batch we will use when writing to clickhouse. This
	//  controls the behavior of the MemoryRowBuffer
	MaxBatchSize int `json:"max_batch_size"`

	// This controls the size of a buffer between the flush loop and the actual
	//  writers. During Clickhouse instability the writers will back off while
	//  attempting to write their batches. This setting will allow the specified
	//  number of batches to queue up within the buffer. When this buffer fills
	//  it "backs off" to the message row buffer.
	BatchBufferSize int `json:"batch_buffer_size"`

	// The interval at which we write to Clickhouse. The higher number the more
	//  latency and potential for your message buffer to fill, however lower numbers
	//  dramatically increase the load of Clickhouse depending on your schema and
	//  disk performance.
	FlushInterval int `json:"flush_interval"`

	// This setting dictates the behavior of the message buffer when its full.
	//   "drop-oldest" will cause the oldest (by time written) messages to be dropped
	//   "drop-newest" will cause new messages to be dropped
	OnFull string `json:"on_full"`

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
		Messages:        nil,
		MaxBatchSize:    5000,
		BatchBufferSize: 16,
		FlushInterval:   1000,
		Writers:         1,
		OnFull:          "drop-oldest",
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
