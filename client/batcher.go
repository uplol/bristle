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
	// Maximum number of messages to allow being buffered in memory
	BufferSize int

	// Whether writing to the upstream will perform retries on error, or drop
	//  payloads
	Retry bool

	// Interval on which we will attempt to flush to the upstream
	FlushInterval time.Duration
}

type BristleBatcher struct {
	sync.Mutex

	client  *BristleClient
	config  BristleBatcherConfig
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

func (b *BristleBatcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(b.config.FlushInterval)
	for {
		select {
		case <-ticker.C:
			b.flush()
		case <-ctx.Done():
			return nil
		}
	}
}

func (b *BristleBatcher) flush() {
	b.Lock()
	batches := b.batches
	b.batches = make(map[string][]proto.Message)
	b.Unlock()

	retryTimes := 0
	if b.config.Retry {
		retryTimes = -1
	}

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
