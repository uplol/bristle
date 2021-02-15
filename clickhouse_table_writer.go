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

	writers []*ClickhouseTableWriter
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
		writers: make([]*ClickhouseTableWriter, 0),
		done:    make(chan struct{}),
		cleanup: make(chan struct{}),
		size:    0,
	}
}

func NewClickhouseTableWriter(table *ClickhouseTable) (*ClickhouseTableWriter, error) {
	buffer, err := NewMemoryBuffer(
		string(table.Name),
		table.config.MaxBatchSize,
		table.config.OnFull,
	)
	if err != nil {
		return nil, err
	}

	return &ClickhouseTableWriter{
		table:       table,
		buffer:      buffer,
		batchBuffer: make(chan [][]interface{}, table.config.BatchBufferSize),
	}, nil
}

func (c *writerGroup) Add(writer *ClickhouseTableWriter) {
	c.Lock()
	defer c.Unlock()

	writer.table.writers = append(writer.table.writers, writer)
	c.writers = append(c.writers, writer)
}

func (c *writerGroup) Close() {
	c.Lock()
	defer c.Unlock()

	close(c.done)

	go func() {
		log.Info().Int("writers", c.size).Msg("writer-group: waiting for all writers to shutdown")
		for i := 0; i < c.size; i++ {
			<-c.cleanup
		}
		log.Info().Msg("writer-group: all writers have shutdown, goodbye")
	}()
}

func (c *writerGroup) Start() {
	c.Lock()
	defer c.Unlock()

	for _, writer := range c.writers {
		writer.Start(c.done, c.cleanup)
		c.size += 1
	}
}

func (c *ClickhouseTableWriter) Start(done chan struct{}, cleanup chan struct{}) {
	go func() {
		c.run(done)
		cleanup <- struct{}{}
	}()
}

func (c *ClickhouseTableWriter) run(done chan struct{}) {
	writerDone := make(chan struct{})
	go c.writer(writerDone)

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
			c.batchBuffer <- nil
			<-writerDone
			return
		default:
			continue
		}
	}
}

func (c *ClickhouseTableWriter) writer(done chan struct{}) {
	for {
		batch := <-c.batchBuffer
		if batch == nil {
			log.Trace().Msg("clickhouse-table-writer: close requested, goodbye!")
			close(done)
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
