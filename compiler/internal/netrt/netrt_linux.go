//go:build linux

package netrt

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

const soReusePort = 0x0f

func ListenTCP4(cfg TCPListenConfig) (TCPListener, error) {
	if cfg.Port < 0 || cfg.Port > 65535 {
		return TCPListener{}, fmt.Errorf("tcp listen port %d outside 0..65535", cfg.Port)
	}
	backlog := cfg.Backlog
	if backlog <= 0 {
		backlog = 128
	}
	socketType := syscall.SOCK_STREAM | syscall.SOCK_CLOEXEC
	if cfg.Nonblocking {
		socketType |= syscall.SOCK_NONBLOCK
	}
	fd, err := syscall.Socket(syscall.AF_INET, socketType, syscall.IPPROTO_TCP)
	if err != nil {
		return TCPListener{}, err
	}
	keep := false
	defer func() {
		if !keep {
			_ = syscall.Close(fd)
		}
	}()

	if cfg.ReuseAddr {
		if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
			return TCPListener{}, err
		}
	}
	if cfg.ReusePort {
		if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, soReusePort, 1); err != nil {
			return TCPListener{}, err
		}
	}
	if cfg.NoDelay {
		if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_NODELAY, 1); err != nil {
			return TCPListener{}, err
		}
	}
	addr := &syscall.SockaddrInet4{Port: cfg.Port, Addr: cfg.Address}
	if err := syscall.Bind(fd, addr); err != nil {
		return TCPListener{}, err
	}
	if err := syscall.Listen(fd, backlog); err != nil {
		return TCPListener{}, err
	}
	port := cfg.Port
	if port == 0 {
		sa, err := syscall.Getsockname(fd)
		if err != nil {
			return TCPListener{}, err
		}
		inet, ok := sa.(*syscall.SockaddrInet4)
		if !ok {
			return TCPListener{}, fmt.Errorf("getsockname returned %T, want *syscall.SockaddrInet4", sa)
		}
		port = inet.Port
	}
	keep = true
	return TCPListener{FD: fd, Port: port}, nil
}

func Accept(listenerFD int, cfg AcceptConfig) (int, error) {
	flags := 0
	if cfg.Nonblocking {
		flags |= syscall.SOCK_NONBLOCK
	}
	if cfg.CloseOnExec {
		flags |= syscall.SOCK_CLOEXEC
	}
	fd, _, err := syscall.Accept4(listenerFD, flags)
	if err != nil {
		return -1, err
	}
	if cfg.NoDelay {
		if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_NODELAY, 1); err != nil {
			_ = syscall.Close(fd)
			return -1, err
		}
	}
	return fd, nil
}

func IsNonblocking(fd int) (bool, error) {
	flags, err := fcntl(fd, syscall.F_GETFL, 0)
	if err != nil {
		return false, err
	}
	return flags&syscall.O_NONBLOCK != 0, nil
}

func SetNonblocking(fd int, enabled bool) error {
	return syscall.SetNonblock(fd, enabled)
}

func Read(fd int, buf []byte) (int, error) {
	return syscall.Read(fd, buf)
}

func Recv(fd int, buf []byte) (int, error) {
	n, _, err := syscall.Recvfrom(fd, buf, 0)
	return n, err
}

func Write(fd int, buf []byte) (int, error) {
	return syscall.Write(fd, buf)
}

func Send(fd int, buf []byte) (int, error) {
	return syscall.SendmsgN(fd, buf, nil, nil, 0)
}

func Writev(fd int, chunks [][]byte) (int, error) {
	iovecs := make([]syscall.Iovec, 0, len(chunks))
	for _, chunk := range chunks {
		if len(chunk) == 0 {
			continue
		}
		iovecs = append(iovecs, syscall.Iovec{Base: &chunk[0]})
		iovecs[len(iovecs)-1].SetLen(len(chunk))
	}
	if len(iovecs) == 0 {
		return 0, nil
	}
	n, _, errno := syscall.Syscall(syscall.SYS_WRITEV, uintptr(fd), uintptr(unsafe.Pointer(&iovecs[0])), uintptr(len(iovecs)))
	if errno != 0 {
		return int(n), errno
	}
	return int(n), nil
}

func Sendfile(outFD int, inFD int, offset *int64, count int) (int, error) {
	return syscall.Sendfile(outFD, inFD, offset, count)
}

func Close(fd int) error {
	if fd < 0 {
		return nil
	}
	return syscall.Close(fd)
}

func NewPoller() (*Poller, error) {
	fd, err := syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}
	return &Poller{fd: fd}, nil
}

func (p *Poller) Close() error {
	if p != nil {
		p.mu.Lock()
		defer p.mu.Unlock()
	}
	if p == nil || p.fd < 0 {
		return nil
	}
	fd := p.fd
	p.fd = -1
	return syscall.Close(fd)
}

func (p *Poller) Add(fd int, interest Interest) error {
	return p.ctl(syscall.EPOLL_CTL_ADD, fd, interest)
}

func (p *Poller) AddRead(fd int) error {
	return p.Add(fd, InterestRead)
}

func (p *Poller) AddReadWrite(fd int) error {
	return p.Add(fd, InterestRead|InterestWrite)
}

func (p *Poller) Mod(fd int, interest Interest) error {
	return p.ctl(syscall.EPOLL_CTL_MOD, fd, interest)
}

func (p *Poller) Remove(fd int) error {
	if p != nil {
		p.mu.Lock()
		defer p.mu.Unlock()
	}
	if p == nil || p.fd < 0 {
		return syscall.EBADF
	}
	return syscall.EpollCtl(p.fd, syscall.EPOLL_CTL_DEL, fd, nil)
}

func (p *Poller) Wait(maxEvents int, timeout time.Duration) ([]Event, error) {
	if p != nil {
		p.mu.Lock()
		defer p.mu.Unlock()
	}
	if p == nil || p.fd < 0 {
		return nil, syscall.EBADF
	}
	if maxEvents <= 0 {
		maxEvents = 1
	}
	rawEvents := make([]syscall.EpollEvent, maxEvents)
	timeoutMillis := pollTimeoutMillis(timeout)
	for {
		n, err := syscall.EpollWait(p.fd, rawEvents, timeoutMillis)
		if err == syscall.EINTR {
			continue
		}
		if err != nil {
			return nil, err
		}
		events := make([]Event, 0, n)
		for i := 0; i < n; i++ {
			raw := rawEvents[i]
			events = append(events, Event{
				FD:       int(raw.Fd),
				Readable: raw.Events&syscall.EPOLLIN != 0,
				Writable: raw.Events&syscall.EPOLLOUT != 0,
				Hup:      raw.Events&syscall.EPOLLHUP != 0,
				Err:      raw.Events&syscall.EPOLLERR != 0,
			})
		}
		return events, nil
	}
}

func (p *Poller) ctl(op int, fd int, interest Interest) error {
	if p != nil {
		p.mu.Lock()
		defer p.mu.Unlock()
	}
	if p == nil || p.fd < 0 {
		return syscall.EBADF
	}
	event := syscall.EpollEvent{
		Events: epollEvents(interest),
		Fd:     int32(fd),
	}
	return syscall.EpollCtl(p.fd, op, fd, &event)
}

func epollEvents(interest Interest) uint32 {
	events := uint32(syscall.EPOLLERR | syscall.EPOLLHUP)
	if interest&InterestRead != 0 {
		events |= syscall.EPOLLIN
	}
	if interest&InterestWrite != 0 {
		events |= syscall.EPOLLOUT
	}
	return events
}

func fcntl(fd int, cmd int, arg int) (int, error) {
	value, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), uintptr(cmd), uintptr(arg))
	if errno != 0 {
		return 0, errno
	}
	return int(value), nil
}
