package netrt

import (
	"errors"
	"sync"
	"time"
)

var ErrUnsupported = errors.New("netrt is supported only on Linux")

type TCPListenConfig struct {
	Address     [4]byte
	Port        int
	Backlog     int
	Nonblocking bool
	ReuseAddr   bool
	ReusePort   bool
	NoDelay     bool
}

type TCPListener struct {
	FD   int
	Port int
}

type AcceptConfig struct {
	Nonblocking bool
	CloseOnExec bool
	NoDelay     bool
}

type Event struct {
	FD       int
	Readable bool
	Writable bool
	Hup      bool
	Err      bool
}

type Interest uint32

const (
	InterestRead Interest = 1 << iota
	InterestWrite
)

type Poller struct {
	mu sync.Mutex
	fd int
}

func pollTimeoutMillis(timeout time.Duration) int {
	if timeout < 0 {
		return -1
	}
	if timeout == 0 {
		return 0
	}
	ms := timeout / time.Millisecond
	if ms == 0 {
		return 1
	}
	maxInt := int(^uint(0) >> 1)
	if ms > time.Duration(maxInt) {
		return maxInt
	}
	return int(ms)
}
