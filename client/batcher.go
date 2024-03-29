package client

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

type BristleBatcherConfig struct {
	// Interval between batches being flushed to the upstream client
	FlushInterval time.Duration

	// Maximum number of messages to allow being buffered in memory
	BufferSize int

	// If true (the default) batches will not be dropped on error, but will be
	//  continually retried until they can be flushed by the upstream client.
	//  When disabled messages will be dropped when a flush error is encountered.
	Retry bool
}

// Handles buffering messages in memory and flushing batches to an upstream BristleClient
//  on a configured interval.
type BristleBatcher struct {
	sync.Mutex

	config BristleBatcherConfig
	client *BristleClient

	// Mapping of proto type full-name to message buffers
	batches map[string][]proto.Message
}

func NewBristleBatcher(client *BristleClient, config BristleBatcherConfig) *BristleBatcher {
	return &BristleBatcher{
		client:  client,
		config:  config,
		batches: make(map[string][]proto.Message),
	}
}

var ErrBatchOversized = errors.New("batcher cannot handle oversized batch")
var ErrBufferFull = errors.New("batcher buffer is full")

// Run the batcher under the given context, generally this should be run in the
//  background.
func (b *BristleBatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(b.config.FlushInterval)
	for {
		select {
		case <-ticker.C:
			b.flush()
		case <-ctx.Done():
			return
		}
	}
}

func (b *BristleBatcher) flush() {
	// Swap out the underlying batch storage so we hold the lock for the shortest
	//  time possible.
	b.Lock()
	batches := b.batches
	b.batches = make(map[string][]proto.Message)
	b.Unlock()

	retryTimes := 0
	if b.config.Retry {
		retryTimes = -1
	}

	// Send a sep. batch for each message type
	for messageTypeName, batch := range batches {
		err := b.client.WriteBatchSync(messageTypeName, batch, retryTimes)
		if err != nil {
			log.Error().
				Err(err).
				Str("type", messageTypeName).
				Int("batch-size", len(batch)).
				Msg("bristle-batcher: error on writing batch to upstream")
		}
	}
}

func (b *BristleBatcher) WriteBatch(messageType string, messages []proto.Message) error {
	b.Lock()
	defer b.Unlock()

	newBatchSize := len(messages)

	if newBatchSize > b.config.BufferSize {
		return ErrBatchOversized
	}

	batch, ok := b.batches[messageType]
	if !ok {
		b.batches[messageType] = messages
		return nil
	}

	existingBatchSize := len(batch)

	if existingBatchSize+newBatchSize > b.config.BufferSize {
		return ErrBatchOversized
	}

	b.batches[messageType] = append(batch, messages...)
	return nil
}
