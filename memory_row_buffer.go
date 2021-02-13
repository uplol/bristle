package bristle

import (
	"sync"

	"github.com/rs/zerolog/log"
)

var trueValue = true

type MemoryRowBuffer struct {
	sync.Mutex

	maxSize int
	onFull  string
	buffer  [][]interface{}
}

func NewMemoryBuffer(maxSize int, onFull string) *MemoryRowBuffer {
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

	batchSize := len(batch)
	bufferSize := len(m.buffer)
	spareRoom := m.maxSize - bufferSize

	// If we don't have enough space and we drop-oldest
	if spareRoom < batchSize {
		if m.onFull == "drop-oldest" {
			// Drop enough existing "old" items (front of the buffer) to fit
			m.buffer = m.buffer[batchSize-spareRoom:]
		} else if m.onFull == "drop-newest" {
			// Shorten the batch by enough to fit
			batch = batch[batchSize-spareRoom:]
		} else {
			panic("unsupported on full behavior: " + m.onFull)
		}
		log.Trace().
			Str("on-full", m.onFull).
			Int("dropped", batchSize-spareRoom).
			Int("batch-size", batchSize).
			Int("spare-room", spareRoom).
			Msg("memory-buffer: triggered on-full behavior, some messages where dropped")
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
