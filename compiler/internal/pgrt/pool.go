package pgrt

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

var (
	ErrMissingAddress = errors.New("PostgreSQL dial address is empty")
	ErrPoolExhausted  = errors.New("PostgreSQL pool exhausted")
	ErrPoolClosed     = errors.New("PostgreSQL pool closed")
	ErrBadConn        = errors.New("PostgreSQL connection is bad")
)

type Connector func(context.Context) (*Conn, error)

type DialConfig struct {
	Network string
	Address string
	Timeout time.Duration
	Startup StartupConfig
}

type Pool struct {
	mu        sync.Mutex
	maxOpen   int
	open      int
	idle      []*Conn
	closed    bool
	connector Connector
}

type PoolStats struct {
	MaxOpen int
	Open    int
	InUse   int
	Idle    int
	Closed  bool
}

type PooledConn struct {
	Conn     *Conn
	pool     *Pool
	released bool
}

func Dial(ctx context.Context, cfg DialConfig) (*Conn, error) {
	if cfg.Address == "" {
		return nil, ErrMissingAddress
	}
	network := cfg.Network
	if network == "" {
		network = "tcp"
	}
	dialer := net.Dialer{}
	if cfg.Timeout > 0 {
		dialer.Timeout = cfg.Timeout
	}
	rwc, err := dialer.DialContext(ctx, network, cfg.Address)
	if err != nil {
		return nil, err
	}
	conn, err := Connect(ctx, rwc, cfg.Startup)
	if err != nil {
		_ = rwc.Close()
		return nil, err
	}
	return conn, nil
}

func DialConnector(cfg DialConfig) Connector {
	return func(ctx context.Context) (*Conn, error) {
		return Dial(ctx, cfg)
	}
}

func NewDialPool(maxOpen int, cfg DialConfig) (*Pool, error) {
	return NewPool(maxOpen, DialConnector(cfg))
}

func NewPool(maxOpen int, connector Connector) (*Pool, error) {
	if maxOpen <= 0 {
		maxOpen = 1
	}
	if connector == nil {
		return nil, errors.New("PostgreSQL pool connector is nil")
	}
	return &Pool{maxOpen: maxOpen, connector: connector}, nil
}

func (p *Pool) Checkout(ctx context.Context) (*PooledConn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, ErrPoolClosed
	}
	if n := len(p.idle); n > 0 {
		conn := p.idle[n-1]
		p.idle[n-1] = nil
		p.idle = p.idle[:n-1]
		p.mu.Unlock()
		return &PooledConn{Conn: conn, pool: p}, nil
	}
	if p.open >= p.maxOpen {
		p.mu.Unlock()
		return nil, ErrPoolExhausted
	}
	p.open++
	p.mu.Unlock()

	conn, err := p.connector(ctx)
	if err != nil {
		p.mu.Lock()
		p.open--
		p.mu.Unlock()
		return nil, err
	}
	return &PooledConn{Conn: conn, pool: p}, nil
}

func (p *Pool) Stats() PoolStats {
	if p == nil {
		return PoolStats{Closed: true}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	idle := len(p.idle)
	inUse := p.open - idle
	if inUse < 0 {
		inUse = 0
	}
	return PoolStats{
		MaxOpen: p.maxOpen,
		Open:    p.open,
		InUse:   inUse,
		Idle:    idle,
		Closed:  p.closed,
	}
}

func (pc *PooledConn) Release(err error) error {
	if pc == nil || pc.pool == nil || pc.Conn == nil || pc.released {
		return nil
	}
	pc.released = true
	return pc.pool.release(pc.Conn, errors.Is(err, ErrBadConn) || err != nil)
}

func (p *Pool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	idle := p.idle
	p.idle = nil
	p.open -= len(idle)
	p.mu.Unlock()

	var firstErr error
	for _, conn := range idle {
		if err := conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (p *Pool) release(conn *Conn, bad bool) error {
	p.mu.Lock()
	closed := p.closed
	if closed || bad {
		p.open--
		p.mu.Unlock()
		return conn.Close()
	}
	p.idle = append(p.idle, conn)
	p.mu.Unlock()
	return nil
}
