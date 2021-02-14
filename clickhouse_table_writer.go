package bristle

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// A group of ClickhouseTableWriter's which are managed as one unit. This allows
//  for dynamic configuration reload by swapping the underlying writer group.
type writerGroup struct {
	sync.RWMutex

	done    chan struct{}
	cleanup chan struct{}
	size    int
}

type ClickhouseTableWriter struct {
	table       *ClickhouseTable
	buffer      *MemoryRowBuffer
	batchBuffer chan [][]interface{}
}

func newWriterGroup() *writerGroup {
	return &writerGroup{
		done:    make(chan struct{}),
		cleanup: make(chan struct{}),
		size:    0,
	}
}

func NewClickhouseTableWriter(table *ClickhouseTable) *ClickhouseTableWriter {
	buffer := NewMemoryBuffer(table.config.MaxBatchSize, table.config.OnFull)

	return &ClickhouseTableWriter{
		table:       table,
		buffer:      buffer,
		batchBuffer: make(chan [][]interface{}, table.config.BatchBufferSize),
	}
}

func (c *writerGroup) Add(writer *ClickhouseTableWriter) {
	c.Lock()
	defer c.Unlock()

	writer.table.writers = append(writer.table.writers, writer)
	c.size += 1
	go func() {
		writer.Run(c.done)
		c.cleanup <- struct{}{}
	}()
}

func (c *writerGroup) Close() {
	c.Lock()
	defer c.Unlock()

	close(c.done)

	log.Info().Int("writers", c.size).Msg("writer-group: waiting for all writers to shutdown")
	for i := 0; i < c.size; i++ {
		<-c.cleanup
	}
	log.Info().Msg("writer-group: all writers have shutdown, goodbye")
}

func (c *ClickhouseTableWriter) Run(done chan struct{}) {
	go c.writer()
	defer close(c.batchBuffer)

	ticker := time.NewTicker(time.Duration(c.table.config.FlushInterval) * time.Millisecond)

	running := true
	for running {
		<-ticker.C

		batch := c.buffer.FlushBatch()
		if batch != nil {
			c.batchBuffer <- batch
			continue
		}

		select {
		case <-done:
			return
		default:
			continue
		}
	}
}

func (c *ClickhouseTableWriter) writer() {
	for {
		batch, ok := <-c.batchBuffer
		if !ok {
			return
		}

		err := c.writeBatch(batch)

		// TODO: data error vs transient write error
		if err != nil {
			log.Error().Err(err).Int("batch-size", len(batch)).Msg("clickhouse-table-writer: failed to write batch")
		}
	}
}

func (c *ClickhouseTableWriter) writeBatch(batch [][]interface{}) error {
	conn, err := c.table.cluster.GetConn()
	if err != nil {
		return err
	}
	defer c.table.cluster.ReleaseConn(conn)

	tx, err := conn.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(c.table.cachedInsertQuery)
	if err != nil {
		return err
	}

	for _, row := range batch {
		_, err = stmt.Exec(
			row...,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
