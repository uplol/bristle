package bristle

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
	v1 "github.com/uplol/bristle/proto/v1"
)

type OnFullBehavior string

const (
	DropOldest OnFullBehavior = "drop-oldest"
	DropNewest OnFullBehavior = "drop-newest"
	Block      OnFullBehavior = "block"
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
	if onFull != DropNewest && onFull != DropOldest && onFull != Block {
		return nil, fmt.Errorf("invalid on-full behavior '%v'", onFull)
	}

	if maxSize == 0 {
		maxSize = 1000
	}

	return &MemoryRowBuffer{
		name:    name,
		maxSize: maxSize,
		onFull:  onFull,
		buffer:  make([][]interface{}, 0),
	}, nil
}

func (m *MemoryRowBuffer) WriteBatch(batch [][]interface{}) v1.BatchResult {
	m.Lock()
	defer m.Unlock()

	// We can't possibly flush a batch greater than our internal size
	batchSize := len(batch)
	if batchSize > m.maxSize {
		log.Trace().
			Str("name", m.name).
			Int("max-size", m.maxSize).
			Int("batch-size", batchSize).
			Msg("memory-buffer: batch is simply too big bud!")

		return v1.BatchResult_TOO_BIG
	}

	bufferSize := len(m.buffer)
	spareRoom := m.maxSize - bufferSize

	if spareRoom < batchSize {
		log.Trace().
			Str("name", m.name).
			Str("on-full", string(m.onFull)).
			Int("dropped", batchSize-spareRoom).
			Int("batch-size", batchSize).
			Int("spare-room", spareRoom).
			Int("max-size", m.maxSize).
			Msg("memory-buffer: batch exceeds buffer room, triggering on-full and dropping")
		if m.onFull == DropOldest {
			// Drop enough existing "old" items (front of the buffer) to fit
			m.buffer = m.buffer[batchSize-spareRoom:]
		} else if m.onFull == DropNewest {
			// Shorten the batch by enough to fit
			batch = batch[batchSize-spareRoom:]
		} else if m.onFull == Block {
			// Block the write with an error
			return v1.BatchResult_FULL
		}
	}

	m.buffer = append(m.buffer, batch...)
	return v1.BatchResult_OK
}

func (m *MemoryRowBuffer) FlushBatch(batchSize int) [][]interface{} {
	m.Lock()
	defer m.Unlock()

	var result [][]interface{}
	if len(m.buffer) == 0 {
		return nil
	} else if len(m.buffer) <= batchSize {
		result = m.buffer[:len(m.buffer)]
		m.buffer = [][]interface{}{}
	} else {
		result = m.buffer[:batchSize]
		m.buffer = m.buffer[batchSize:]
	}

	return result
}
