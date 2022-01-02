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
	table  *ClickhouseTable
	buffer *MemoryRowBuffer
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
		table.config.MaxBufferSize,
		table.config.OnFull,
	)
	if err != nil {
		return nil, err
	}

	return &ClickhouseTableWriter{
		table:  table,
		buffer: buffer,
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
	ticker := time.NewTicker(time.Duration(c.table.config.FlushInterval) * time.Millisecond)

	running := true
	for running {
		<-ticker.C

		batch := c.buffer.FlushBatch(c.table.config.MaxBatchSize)
		if batch != nil {
			err := c.writeBatch(batch)
			if err != nil {
				log.Error().Err(err).Str("table", string(c.table.Name)).Int("batch-size", len(batch)).Msg("clickhouse-table-writer: failed to write batch")
			}
		}

		select {
		case <-done:
			return
		default:
			continue
		}
	}
}

func (c *ClickhouseTableWriter) writeBatch(batch [][]interface{}) error {
	conn, err := c.table.cluster.GetConn()
	defer c.table.cluster.ReleaseConn(conn)
	if err != nil {
		return err
	}

	tx, err := conn.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(c.table.cachedInsertQuery)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, row := range batch {
		_, err = stmt.Exec(
			row...,
		)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		// NB: for whatever reason clickhouse-go does not handle this well and
		//  leaks connections
		conn.Close()
	}
	return err
}
