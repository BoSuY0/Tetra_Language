package webrt

import (
	"context"
	"errors"
	"sync"
	"syscall"
	"time"

	"tetra_language/compiler/internal/httprt"
	"tetra_language/compiler/internal/netrt"
)

const httpDateLayout = "Mon, 02 Jan 2006 15:04:05 GMT"

type Config struct {
	Address      [4]byte
	Port         int
	Backlog      int
	ServerName   string
	MaxBodyBytes int
	DateFunc     func() string
}

type Server struct {
	Config
	Router httprt.Router
	mu     sync.Mutex

	listener netrt.TCPListener
	poller   *netrt.Poller
	conns    map[int]*connState
}

type connState struct {
	fd              int
	input           []byte
	output          []byte
	closeAfterWrite bool
}

func NewServer(cfg Config) *Server {
	if cfg.ServerName == "" {
		cfg.ServerName = "Tetra"
	}
	return &Server{
		Config: cfg,
		conns:  map[int]*connState{},
	}
}

func (s *Server) Listen() error {
	listener, err := netrt.ListenTCP4(netrt.TCPListenConfig{
		Address:     s.Address,
		Port:        s.Config.Port,
		Backlog:     s.Backlog,
		Nonblocking: true,
		ReuseAddr:   true,
		ReusePort:   true,
		NoDelay:     true,
	})
	if err != nil {
		return err
	}
	poller, err := netrt.NewPoller()
	if err != nil {
		_ = netrt.Close(listener.FD)
		return err
	}
	if err := poller.AddRead(listener.FD); err != nil {
		_ = poller.Close()
		_ = netrt.Close(listener.FD)
		return err
	}
	s.mu.Lock()
	s.listener = listener
	s.poller = poller
	s.mu.Unlock()
	return nil
}

func (s *Server) Port() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listener.Port
}

func (s *Server) Serve(ctx context.Context) error {
	s.mu.Lock()
	poller := s.poller
	s.mu.Unlock()
	if poller == nil {
		return errors.New("server is not listening")
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		events, err := poller.Wait(256, 50*time.Millisecond)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if errors.Is(err, syscall.EBADF) {
				return nil
			}
			return err
		}
		s.mu.Lock()
		if s.poller != poller {
			s.mu.Unlock()
			return nil
		}
		for _, event := range events {
			if event.FD == s.listener.FD {
				if err := s.acceptReady(); err != nil && !isWouldBlock(err) {
					s.mu.Unlock()
					return err
				}
				continue
			}
			conn := s.conns[event.FD]
			if conn == nil {
				continue
			}
			if event.Err || event.Hup {
				s.closeConn(conn)
				continue
			}
			if event.Readable {
				if err := s.readReady(conn); err != nil {
					if isWouldBlock(err) {
						continue
					}
					s.closeConn(conn)
					continue
				}
			}
			if event.Writable || len(conn.output) > 0 {
				if err := s.flush(conn); err != nil {
					if !isWouldBlock(err) {
						s.closeConn(conn)
					}
				}
			}
		}
		s.mu.Unlock()
	}
}

func (s *Server) Close() error {
	s.mu.Lock()
	poller := s.poller
	s.mu.Unlock()
	var firstErr error
	if poller != nil {
		if err := poller.Close(); err != nil && !errors.Is(err, syscall.EBADF) {
			firstErr = err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, conn := range s.conns {
		if err := netrt.Close(conn.fd); err != nil && !errors.Is(err, syscall.EBADF) && firstErr == nil {
			firstErr = err
		}
	}
	s.conns = map[int]*connState{}
	if s.listener.FD > 0 {
		if err := netrt.Close(s.listener.FD); err != nil && !errors.Is(err, syscall.EBADF) && firstErr == nil {
			firstErr = err
		}
		s.listener = netrt.TCPListener{}
	}
	if s.poller != nil {
		s.poller = nil
	}
	return firstErr
}

func (s *Server) acceptReady() error {
	for {
		fd, err := netrt.Accept(s.listener.FD, netrt.AcceptConfig{Nonblocking: true, CloseOnExec: true, NoDelay: true})
		if err != nil {
			return err
		}
		conn := &connState{fd: fd}
		if err := s.poller.AddRead(fd); err != nil {
			_ = netrt.Close(fd)
			return err
		}
		s.conns[fd] = conn
	}
}

func (s *Server) readReady(conn *connState) error {
	buf := make([]byte, 4096)
	for {
		n, err := netrt.Read(conn.fd, buf)
		if n > 0 {
			conn.input = append(conn.input, buf[:n]...)
			if err := s.processInput(conn); err != nil {
				return err
			}
		}
		if err != nil {
			return err
		}
		if n == 0 {
			conn.closeAfterWrite = true
			return nil
		}
	}
}

func (s *Server) processInput(conn *connState) error {
	limits := httprt.Limits{MaxHeaderBytes: 8192, MaxHeaders: 64, MaxBodyBytes: s.MaxBodyBytes}
	for len(conn.input) > 0 {
		req, consumed, err := httprt.ParseRequest(conn.input, limits)
		if errors.Is(err, httprt.ErrIncomplete) {
			return nil
		}
		if err != nil {
			statusCode := 400
			body := []byte("Bad Request")
			if errors.Is(err, httprt.ErrBodyTooLarge) {
				statusCode = 413
				body = []byte("Payload Too Large")
			}
			conn.output = httprt.AppendResponse(conn.output, s.decorateResponse(httprt.Response{
				StatusCode: statusCode,
				Body:       body,
				KeepAlive:  false,
			}))
			conn.input = conn.input[:0]
			conn.closeAfterWrite = true
			return s.updateInterest(conn)
		}
		resp, ok := s.Router.Route(req)
		if !ok {
			resp = httprt.Response{
				StatusCode: 404,
				Body:       []byte("Not Found"),
			}
		}
		resp.KeepAlive = req.KeepAlive
		conn.output = httprt.AppendResponse(conn.output, s.decorateResponse(resp))
		conn.input = conn.input[consumed:]
		if !req.KeepAlive {
			conn.closeAfterWrite = true
			conn.input = conn.input[:0]
			break
		}
	}
	return s.updateInterest(conn)
}

func (s *Server) decorateResponse(resp httprt.Response) httprt.Response {
	if resp.Server == "" {
		resp.Server = s.ServerName
	}
	if resp.Date == "" {
		resp.Date = s.date()
	}
	return resp
}

func (s *Server) date() string {
	if s.DateFunc != nil {
		return s.DateFunc()
	}
	return time.Now().UTC().Format(httpDateLayout)
}

func (s *Server) flush(conn *connState) error {
	for len(conn.output) > 0 {
		n, err := netrt.Write(conn.fd, conn.output)
		if n > 0 {
			copy(conn.output, conn.output[n:])
			conn.output = conn.output[:len(conn.output)-n]
		}
		if err != nil {
			if isWouldBlock(err) {
				return s.updateInterest(conn)
			}
			return err
		}
		if n == 0 {
			return s.updateInterest(conn)
		}
	}
	if conn.closeAfterWrite {
		s.closeConn(conn)
		return nil
	}
	return s.updateInterest(conn)
}

func (s *Server) updateInterest(conn *connState) error {
	if s.poller == nil {
		return nil
	}
	interest := netrt.InterestRead
	if len(conn.output) > 0 {
		interest |= netrt.InterestWrite
	}
	return s.poller.Mod(conn.fd, interest)
}

func (s *Server) closeConn(conn *connState) {
	if conn == nil {
		return
	}
	if s.poller != nil {
		_ = s.poller.Remove(conn.fd)
	}
	_ = netrt.Close(conn.fd)
	delete(s.conns, conn.fd)
}

func isWouldBlock(err error) bool {
	return errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EWOULDBLOCK)
}
