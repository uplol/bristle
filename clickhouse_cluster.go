package bristle

import (
	"database/sql"
	"errors"
	"strings"
	"sync"

	_ "github.com/mailru/go-clickhouse"
)

type ClickhouseCluster struct {
	sync.Mutex

	DSN      string
	tables   map[FullTableName]*ClickhouseTable
	pool     map[*sql.DB]bool
	poolSize int

	config ClickhouseClusterConfig
}

func NewClickhouseCluster(config ClickhouseClusterConfig) *ClickhouseCluster {
	poolSize := config.PoolSize
	if poolSize == 0 {
		poolSize = 8
	}
	return &ClickhouseCluster{
		DSN:      config.DSN,
		config:   config,
		tables:   make(map[FullTableName]*ClickhouseTable),
		pool:     make(map[*sql.DB]bool),
		poolSize: poolSize,
	}
}

type FullTableName string

func (t FullTableName) Parts() []string {
	return strings.SplitN(string(t), ".", 2)
}

func (f FullTableName) Database() string {
	return f.Parts()[0]
}

func (f FullTableName) Table() string {
	return f.Parts()[1]
}

func ParseFullTableName(full string) FullTableName {
	result := FullTableName(full)
	parts := result.Parts()
	if len(parts) != 2 {
		panic("ParseFullTableName failed, wrong number of seperators: " + full)
	}
	return result
}

var ErrNoSuchTable = errors.New("no clickhouse table by that name")

func (c *ClickhouseCluster) GetTable(fullTableName FullTableName) (*ClickhouseTable, error) {
	table := c.tables[fullTableName]
	if table != nil {
		return table, nil
	}

	conn, err := c.GetConn()
	if err != nil {
		return nil, err
	}
	defer c.ReleaseConn(conn)

	columnRows, err := conn.Query(
		`SELECT name, position, type, default_expression FROM system.columns WHERE database = ? AND table = ? ORDER BY position`,
		fullTableName.Database(),
		fullTableName.Table(),
	)
	if err != nil {
		return nil, err
	}
	defer columnRows.Close()

	columns := map[string]*ClickhouseColumn{}
	for columnRows.Next() {
		var column ClickhouseColumn
		if err := columnRows.Scan(&column.Name, &column.Position, &column.Type, &column.Default); err != nil {
			return nil, err
		}
		columns[column.Name] = &column
	}

	if len(columns) == 0 {
		return nil, ErrNoSuchTable
	}

	config, ok := c.config.Tables[string(fullTableName)]
	if !ok {
		config = DefaultClickhouseTableConfig()
	}

	table, err = newClickhouseTable(c, fullTableName, columns, config)
	if err != nil {
		return nil, err
	}
	c.tables[fullTableName] = table
	return table, nil
}

var ErrNoConn = errors.New("no connection in pool")

func (c *ClickhouseCluster) ReleaseConn(conn *sql.DB) {
	c.Lock()
	defer c.Unlock()
	c.pool[conn] = false
}

// Returns a connection from the pool if one is available, otherwise an error
func (c *ClickhouseCluster) GetConn() (*sql.DB, error) {
	// This function attempts to avoid keeping a lock while doing misc network
	//  activity, like opening a new connection or pinging an existing one. This
	//  results in some messy code, so its well documented.
	c.Lock()

	// We try to find an existing connection in the pool which is free
	var selectedConn *sql.DB
	for conn, isCheckedOut := range c.pool {
		if !isCheckedOut {
			selectedConn = conn
			break
		}
	}

	// Note at this point we have the lock acquired, so its expected that we should
	//  still have it acquired when exiting this block.
	if selectedConn != nil {
		// We mark the selected connection as checked out
		c.pool[selectedConn] = true

		// Release the lock so we can check the selected connection with a ping
		c.Unlock()

		// Ping on the connection and return if its successful
		if err := selectedConn.Ping(); err == nil {
			return selectedConn, nil
		}

		// Finalize the connection
		selectedConn.Close()

		// Reacquire the lock
		c.Lock()
		delete(c.pool, selectedConn)

		// We want to keep the lock acquired now so we match the state at the
		//  entrypoint of this block.
	}

	// If we don't have room for a new connection return that we're full
	if len(c.pool) >= c.poolSize {
		// Release the lock as we exit
		c.Unlock()

		// TODO: eventually we may want to block here
		return nil, ErrNoConn
	}

	// Release the lock so we can open a new connection
	c.Unlock()

	connect, err := sql.Open("clickhouse", c.DSN)
	if err != nil {
		return nil, err
	}

	// Ok we're almost done, we can reaqcuire the lock once more and insert the
	//  newly opened connection in the pool, finally returning it to the user.
	c.Lock()
	c.pool[selectedConn] = true
	c.Unlock()
	return connect, nil
}
