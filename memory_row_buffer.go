package bristle

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
)

type OnFullBehavior string

const (
	DropOldest OnFullBehavior = "drop-oldest"
	DropNewest OnFullBehavior = "drop-newest"
)

var trueValue = true

type MemoryRowBuffer struct {
	sync.Mutex

	name    string
	maxSize int
	onFull  OnFullBehavior
	buffer  [][]interface{}
}

func NewMemoryBuffer(name string, maxSize int, onFull OnFullBehavior) (*MemoryRowBuffer, error) {
	if onFull != DropNewest && onFull != DropOldest {
		return nil, fmt.Errorf("invalid on-full behavior '%v'", onFull)
	}

	return &MemoryRowBuffer{
		name:    name,
		maxSize: maxSize,
		onFull:  onFull,
		buffer:  make([][]interface{}, 0),
	}, nil
}

func (m *MemoryRowBuffer) WriteBatch(batch [][]interface{}) {
	m.Lock()
	defer m.Unlock()

	// We can't possibly flush a batch greater than our internal size, we'll need
	//  to adjust the batch depending on the on-full behavior.
	batchSize := len(batch)
	if batchSize > m.maxSize {
		// Drop the oldest entries in our batch
		if m.onFull == DropOldest {
			batch = batch[:m.maxSize]
		} else if m.onFull == DropNewest {
			batch = batch[batchSize-m.maxSize:]
		}

		log.Warn().
			Str("name", m.name).
			Str("on-full", string(m.onFull)).
			Int("dropped", batchSize-m.maxSize).
			Int("max-size", m.maxSize).
			Msg("memory-buffer: batch exceeds max batch size, triggering on-full to drop extras")
		batchSize = len(batch)
	}

	bufferSize := len(m.buffer)
	spareRoom := m.maxSize - bufferSize

	// If we don't have enough space and we drop-oldest
	if spareRoom < batchSize {
		if m.onFull == DropOldest {
			// Drop enough existing "old" items (front of the buffer) to fit
			m.buffer = m.buffer[batchSize-spareRoom:]
		} else if m.onFull == DropNewest {
			// Shorten the batch by enough to fit
			batch = batch[batchSize-spareRoom:]
		}
		log.Trace().
			Str("name", m.name).
			Str("on-full", string(m.onFull)).
			Int("dropped", batchSize-spareRoom).
			Int("batch-size", batchSize).
			Int("spare-room", spareRoom).
			Int("max-size", m.maxSize).
			Msg("memory-buffer: batch exceeds buffer room, triggering on-full and dropping")
	}

	m.buffer = append(m.buffer, batch...)
}

func (m *MemoryRowBuffer) FlushBatch() [][]interface{} {
	m.Lock()
	defer m.Unlock()

	if len(m.buffer) == 0 {
		return nil
	}

	result := m.buffer[:len(m.buffer)]
	m.buffer = [][]interface{}{}

	return result
}
