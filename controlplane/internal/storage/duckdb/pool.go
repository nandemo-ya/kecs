package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// ConnectionPool manages a pool of DuckDB connections
type ConnectionPool struct {
	mu          sync.Mutex
	connections chan *pooledConnection
	dsn         string
	maxConns    int
	maxIdleTime time.Duration
	maxLifetime time.Duration
	
	// Statistics
	totalConns   int
	activeConns  int
	waitCount    int64
	waitDuration time.Duration
}

type pooledConnection struct {
	conn       *sql.DB
	lastUsed   time.Time
	createdAt  time.Time
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(dsn string, maxConns, maxIdleConns int) (*ConnectionPool, error) {
	if maxIdleConns > maxConns {
		maxIdleConns = maxConns
	}
	
	pool := &ConnectionPool{
		connections: make(chan *pooledConnection, maxIdleConns),
		dsn:         dsn,
		maxConns:    maxConns,
		maxIdleTime: 15 * time.Minute,
		maxLifetime: 1 * time.Hour,
	}
	
	// Pre-create idle connections
	for i := 0; i < maxIdleConns; i++ {
		conn, err := pool.createConnection()
		if err != nil {
			// Clean up already created connections
			pool.Close()
			return nil, fmt.Errorf("failed to create initial connection: %w", err)
		}
		pool.connections <- conn
	}
	
	// Start cleanup goroutine
	go pool.cleanupLoop()
	
	return pool, nil
}

// Get acquires a connection from the pool
func (p *ConnectionPool) Get(ctx context.Context) (*sql.DB, error) {
	p.mu.Lock()
	waitStart := time.Now()
	p.waitCount++
	p.mu.Unlock()
	
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case conn := <-p.connections:
		p.mu.Lock()
		p.waitDuration += time.Since(waitStart)
		p.activeConns++
		p.mu.Unlock()
		
		// Check if connection is still valid
		if time.Since(conn.createdAt) > p.maxLifetime {
			conn.conn.Close()
			newConn, err := p.createConnection()
			if err != nil {
				p.mu.Lock()
				p.activeConns--
				p.mu.Unlock()
				return nil, err
			}
			return newConn.conn, nil
		}
		
		// Ping to ensure connection is alive
		if err := conn.conn.PingContext(ctx); err != nil {
			conn.conn.Close()
			newConn, err := p.createConnection()
			if err != nil {
				p.mu.Lock()
				p.activeConns--
				p.mu.Unlock()
				return nil, err
			}
			return newConn.conn, nil
		}
		
		conn.lastUsed = time.Now()
		return conn.conn, nil
		
	default:
		// No idle connection available, create new if under limit
		p.mu.Lock()
		if p.totalConns < p.maxConns {
			p.mu.Unlock()
			conn, err := p.createConnection()
			if err != nil {
				return nil, err
			}
			p.mu.Lock()
			p.activeConns++
			p.waitDuration += time.Since(waitStart)
			p.mu.Unlock()
			return conn.conn, nil
		}
		p.mu.Unlock()
		
		// Wait for a connection to become available
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case conn := <-p.connections:
			p.mu.Lock()
			p.waitDuration += time.Since(waitStart)
			p.activeConns++
			p.mu.Unlock()
			conn.lastUsed = time.Now()
			return conn.conn, nil
		}
	}
}

// Put returns a connection to the pool
func (p *ConnectionPool) Put(conn *sql.DB) {
	if conn == nil {
		return
	}
	
	p.mu.Lock()
	p.activeConns--
	p.mu.Unlock()
	
	pc := &pooledConnection{
		conn:     conn,
		lastUsed: time.Now(),
	}
	
	select {
	case p.connections <- pc:
		// Connection returned to pool
	default:
		// Pool is full, close the connection
		conn.Close()
		p.mu.Lock()
		p.totalConns--
		p.mu.Unlock()
	}
}

// createConnection creates a new database connection
func (p *ConnectionPool) createConnection() (*pooledConnection, error) {
	conn, err := sql.Open("duckdb", p.dsn)
	if err != nil {
		return nil, err
	}
	
	// Configure connection
	conn.SetMaxOpenConns(1) // DuckDB is single-threaded per connection
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(p.maxLifetime)
	
	p.mu.Lock()
	p.totalConns++
	p.mu.Unlock()
	
	return &pooledConnection{
		conn:      conn,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
	}, nil
}

// cleanupLoop periodically removes idle connections
func (p *ConnectionPool) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		p.cleanup()
	}
}

// cleanup removes expired connections
func (p *ConnectionPool) cleanup() {
	var toClose []*pooledConnection
	
	// Temporarily drain the pool to check connections
	conns := make([]*pooledConnection, 0)
	timeout := time.After(100 * time.Millisecond)
	
drainLoop:
	for {
		select {
		case conn := <-p.connections:
			conns = append(conns, conn)
		case <-timeout:
			break drainLoop
		default:
			break drainLoop
		}
	}
	
	// Check each connection
	now := time.Now()
	for _, conn := range conns {
		if now.Sub(conn.lastUsed) > p.maxIdleTime || now.Sub(conn.createdAt) > p.maxLifetime {
			toClose = append(toClose, conn)
		} else {
			// Put back valid connections
			select {
			case p.connections <- conn:
			default:
				toClose = append(toClose, conn)
			}
		}
	}
	
	// Close expired connections
	for _, conn := range toClose {
		conn.conn.Close()
		p.mu.Lock()
		p.totalConns--
		p.mu.Unlock()
	}
}

// Stats returns pool statistics
func (p *ConnectionPool) Stats() ConnectionPoolStats {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	return ConnectionPoolStats{
		TotalConnections:   p.totalConns,
		ActiveConnections:  p.activeConns,
		IdleConnections:    len(p.connections),
		WaitCount:          p.waitCount,
		WaitDuration:       p.waitDuration,
		MaxConnections:     p.maxConns,
	}
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() error {
	close(p.connections)
	
	var lastErr error
	for conn := range p.connections {
		if err := conn.conn.Close(); err != nil {
			lastErr = err
		}
	}
	
	return lastErr
}

// ConnectionPoolStats contains pool statistics
type ConnectionPoolStats struct {
	TotalConnections   int
	ActiveConnections  int
	IdleConnections    int
	WaitCount          int64
	WaitDuration       time.Duration
	MaxConnections     int
}

// PreparedStatementCache caches prepared statements per connection
type PreparedStatementCache struct {
	mu         sync.RWMutex
	statements map[string]*sql.Stmt
	conn       *sql.DB
}

// NewPreparedStatementCache creates a new prepared statement cache
func NewPreparedStatementCache(conn *sql.DB) *PreparedStatementCache {
	return &PreparedStatementCache{
		statements: make(map[string]*sql.Stmt),
		conn:       conn,
	}
}

// Get retrieves or creates a prepared statement
func (c *PreparedStatementCache) Get(ctx context.Context, query string) (*sql.Stmt, error) {
	c.mu.RLock()
	stmt, exists := c.statements[query]
	c.mu.RUnlock()
	
	if exists {
		return stmt, nil
	}
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Double-check after acquiring write lock
	if stmt, exists := c.statements[query]; exists {
		return stmt, nil
	}
	
	// Prepare new statement
	stmt, err := c.conn.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	
	c.statements[query] = stmt
	return stmt, nil
}

// Close closes all prepared statements
func (c *PreparedStatementCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	var lastErr error
	for _, stmt := range c.statements {
		if err := stmt.Close(); err != nil {
			lastErr = err
		}
	}
	
	c.statements = make(map[string]*sql.Stmt)
	return lastErr
}

// DB returns a single connection for initialization purposes
func (p *ConnectionPool) DB() *sql.DB {
	// Get a connection from the pool
	ctx := context.Background()
	conn, err := p.Get(ctx)
	if err != nil {
		// Fallback to creating a new connection
		db, _ := sql.Open("duckdb", p.dsn)
		return db
	}
	return conn
}

// InitializeCommonStatements prepares commonly used statements
func (p *ConnectionPool) InitializeCommonStatements(ctx context.Context) error {
	// Common queries that benefit from preparation
	commonQueries := []string{
		"SELECT * FROM clusters WHERE name = ?",
		"SELECT * FROM clusters WHERE region = ? AND account_id = ?",
		"SELECT * FROM services WHERE cluster_arn = ?",
		"SELECT * FROM tasks WHERE cluster_arn = ? AND status = ?",
		"SELECT * FROM task_definitions WHERE family = ? ORDER BY revision DESC LIMIT 1",
	}
	
	// Get a connection to prepare statements
	conn, err := p.Get(ctx)
	if err != nil {
		return err
	}
	defer p.Put(conn)
	
	cache := NewPreparedStatementCache(conn)
	for _, query := range commonQueries {
		if _, err := cache.Get(ctx, query); err != nil {
			return fmt.Errorf("failed to prepare statement: %w", err)
		}
	}
	
	return nil
}