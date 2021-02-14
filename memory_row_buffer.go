package bristle

import (
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

	maxSize int
	onFull  OnFullBehavior
	buffer  [][]interface{}
}

func NewMemoryBuffer(maxSize int, onFull OnFullBehavior) *MemoryRowBuffer {
	m := &MemoryRowBuffer{
		maxSize: maxSize,
		onFull:  onFull,
		buffer:  make([][]interface{}, 0),
	}
	return m
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
		} else {
			// TODO: clean up these to a type / config validation
			panic("unsupported on-full behavior: " + m.onFull)
		}

		log.Warn().
			Str("on-full", string(m.onFull)).
			Int("dropped", batchSize-m.maxSize).
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
		} else {
			panic("unsupported on full behavior: " + m.onFull)
		}
		log.Trace().
			Str("on-full", string(m.onFull)).
			Int("dropped", batchSize-spareRoom).
			Int("batch-size", batchSize).
			Int("spare-room", spareRoom).
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
