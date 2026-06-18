package surfacehost

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

func ListenAndServeUnix(ctx context.Context, socketPath string, backend Backend) error {
	socketPath = strings.TrimSpace(socketPath)
	if socketPath == "" {
		return fmt.Errorf("surface host socket path is required")
	}
	if !filepath.IsAbs(socketPath) {
		return fmt.Errorf("surface host socket path must be absolute: %s", socketPath)
	}
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		return err
	}
	_ = os.Remove(socketPath)
	addr := net.UnixAddr{Name: socketPath, Net: "unix"}
	listener, err := net.ListenUnix("unix", &addr)
	if err != nil {
		return fmt.Errorf("listen on Surface host socket %s: %w", socketPath, err)
	}
	defer listener.Close()
	defer os.Remove(socketPath)

	errCh := make(chan error, 1)
	go func() {
		<-ctx.Done()
		errCh <- listener.Close()
	}()

	for {
		conn, err := listener.AcceptUnix()
		if err != nil {
			if errors.Is(ctx.Err(), context.Canceled) ||
				errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil
			}
			select {
			case closeErr := <-errCh:
				if closeErr == nil || errors.Is(closeErr, net.ErrClosed) {
					return nil
				}
				return closeErr
			default:
			}
			return err
		}
		if recorder, ok := backend.(AppPIDRecorder); ok {
			recorder.RecordAppPID(unixPeerPID(conn))
		}
		if err := ServeConn(ctx, conn, backend); err != nil {
			conn.Close()
			return err
		}
		_ = conn.Close()
	}
}

func NewBackend(name string) (Backend, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "wayland":
		return NewWaylandBackend()
	default:
		return nil, fmt.Errorf("unsupported Surface host backend %q", name)
	}
}
