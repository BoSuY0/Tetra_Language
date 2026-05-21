package pgrt

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrPoolExhausted = errors.New("PostgreSQL pool exhausted")
	ErrPoolClosed    = errors.New("PostgreSQL pool closed")
	ErrBadConn       = errors.New("PostgreSQL connection is bad")
)

type Connector func(context.Context) (*Conn, error)

type Pool struct {
	mu        sync.Mutex
	maxOpen   int
	open      int
	idle      []*Conn
	closed    bool
	connector Connector
}

type PooledConn struct {
	Conn     *Conn
	pool     *Pool
	released bool
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
