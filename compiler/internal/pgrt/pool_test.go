package pgrt

import (
	"context"
	"errors"
	"testing"
)

func TestPoolReusesReleasedConnectionAndCapsOpenConnections(t *testing.T) {
	var created int
	pool, err := NewPool(2, func(ctx context.Context) (*Conn, error) {
		created++
		return newTestConn(), nil
	})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	first, err := pool.Checkout(context.Background())
	if err != nil {
		t.Fatalf("Checkout first: %v", err)
	}
	second, err := pool.Checkout(context.Background())
	if err != nil {
		t.Fatalf("Checkout second: %v", err)
	}
	if _, err := pool.Checkout(context.Background()); !errors.Is(err, ErrPoolExhausted) {
		t.Fatalf("Checkout exhausted = %v, want ErrPoolExhausted", err)
	}
	if created != 2 {
		t.Fatalf("created = %d, want 2", created)
	}
	if err := first.Release(nil); err != nil {
		t.Fatalf("Release first: %v", err)
	}
	reused, err := pool.Checkout(context.Background())
	if err != nil {
		t.Fatalf("Checkout reused: %v", err)
	}
	if reused.Conn != first.Conn {
		t.Fatalf("pool did not reuse released connection")
	}
	if err := reused.Release(nil); err != nil {
		t.Fatalf("Release reused: %v", err)
	}
	if err := second.Release(nil); err != nil {
		t.Fatalf("Release second: %v", err)
	}
}

func TestPoolDropsBadConnectionsAndCreatesReplacement(t *testing.T) {
	var created int
	pool, err := NewPool(1, func(ctx context.Context) (*Conn, error) {
		created++
		return newTestConn(), nil
	})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	first, err := pool.Checkout(context.Background())
	if err != nil {
		t.Fatalf("Checkout first: %v", err)
	}
	firstConn := first.Conn
	firstRWC := firstConn.rwc.(*countingRWC)
	if err := first.Release(ErrBadConn); err != nil {
		t.Fatalf("Release bad: %v", err)
	}
	if !firstRWC.closed {
		t.Fatalf("bad connection was not closed")
	}

	second, err := pool.Checkout(context.Background())
	if err != nil {
		t.Fatalf("Checkout replacement: %v", err)
	}
	if second.Conn == firstConn {
		t.Fatalf("expected replacement connection")
	}
	if created != 2 {
		t.Fatalf("created = %d, want 2", created)
	}
	if err := second.Release(nil); err != nil {
		t.Fatalf("Release second: %v", err)
	}
}

func TestPoolCloseRejectsFutureCheckoutAndClosesIdle(t *testing.T) {
	pool, err := NewPool(1, func(ctx context.Context) (*Conn, error) {
		return newTestConn(), nil
	})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	checked, err := pool.Checkout(context.Background())
	if err != nil {
		t.Fatalf("Checkout: %v", err)
	}
	conn := checked.Conn
	rwc := conn.rwc.(*countingRWC)
	if err := checked.Release(nil); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if err := pool.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !rwc.closed {
		t.Fatalf("idle connection was not closed")
	}
	if _, err := pool.Checkout(context.Background()); !errors.Is(err, ErrPoolClosed) {
		t.Fatalf("Checkout after close = %v, want ErrPoolClosed", err)
	}
}

func TestPoolStatsTrackOpenIdleInUseAndClosedState(t *testing.T) {
	pool, err := NewPool(2, func(ctx context.Context) (*Conn, error) {
		return newTestConn(), nil
	})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	first, err := pool.Checkout(context.Background())
	if err != nil {
		t.Fatalf("Checkout first: %v", err)
	}
	second, err := pool.Checkout(context.Background())
	if err != nil {
		t.Fatalf("Checkout second: %v", err)
	}
	stats := pool.Stats()
	if stats.MaxOpen != 2 || stats.Open != 2 || stats.InUse != 2 || stats.Idle != 0 ||
		stats.Closed {
		t.Fatalf("checked-out stats = %#v", stats)
	}
	if err := first.Release(nil); err != nil {
		t.Fatalf("Release first: %v", err)
	}
	stats = pool.Stats()
	if stats.Open != 2 || stats.InUse != 1 || stats.Idle != 1 {
		t.Fatalf("partially released stats = %#v", stats)
	}
	if err := second.Release(ErrBadConn); err != nil {
		t.Fatalf("Release bad second: %v", err)
	}
	stats = pool.Stats()
	if stats.Open != 1 || stats.InUse != 0 || stats.Idle != 1 {
		t.Fatalf("bad-release stats = %#v", stats)
	}
	if err := pool.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	stats = pool.Stats()
	if stats.Open != 0 || stats.InUse != 0 || stats.Idle != 0 || !stats.Closed {
		t.Fatalf("closed stats = %#v", stats)
	}
}

func newTestConn() *Conn {
	return &Conn{rwc: &countingRWC{}, maxPayload: 1 << 20}
}

type countingRWC struct {
	closed bool
	writes int
}

func (rw *countingRWC) Read(p []byte) (int, error) {
	return 0, errors.New("countingRWC does not read")
}

func (rw *countingRWC) Write(p []byte) (int, error) {
	rw.writes++
	return len(p), nil
}

func (rw *countingRWC) Close() error {
	rw.closed = true
	return nil
}
