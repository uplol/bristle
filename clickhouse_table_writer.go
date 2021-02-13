package bristle

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

type ClickhouseTableWriter struct {
	table       *ClickhouseTable
	buffer      *MemoryRowBuffer
	batchBuffer chan [][]interface{}
}

func NewClickhouseTableWriter(table *ClickhouseTable) *ClickhouseTableWriter {
	buffer := NewMemoryBuffer(table.config.MaxBatchSize, table.config.OnFull)

	return &ClickhouseTableWriter{
		table:       table,
		buffer:      buffer,
		batchBuffer: make(chan [][]interface{}, table.config.BatchBufferSize),
	}
}

func (c *ClickhouseTableWriter) Run(ctx context.Context) {
	go c.writer()

	ticker := time.NewTicker(time.Duration(c.table.config.FlushInterval) * time.Millisecond)

	running := true
	for running {
		<-ticker.C

		batch := c.buffer.FlushBatch()
		c.batchBuffer <- batch

		select {
		case <-ctx.Done():
			return
		default:
			continue
		}
	}

	close(c.batchBuffer)
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
