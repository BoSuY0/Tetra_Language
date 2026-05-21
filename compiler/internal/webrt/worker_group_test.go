package webrt

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"tetra_language/compiler/internal/httprt"
)

func TestListenWorkersServesSharedPortAcrossMultipleEventLoops(t *testing.T) {
	group, err := ListenWorkers(2, 0, func(workerID int, port int) (*Server, error) {
		srv := NewServer(Config{
			Address:    [4]byte{127, 0, 0, 1},
			Port:       port,
			ServerName: fmt.Sprintf("Tetra-Worker-%d", workerID),
			DateFunc: func() string {
				return "Wed, 20 May 2026 12:00:00 GMT"
			},
		})
		srv.Router.Handle("GET", "/plaintext", func(req httprt.Request) httprt.Response {
			return httprt.Response{
				StatusCode:  200,
				ContentType: "text/plain",
				Body:        []byte(fmt.Sprintf("worker-%d", workerID)),
			}
		})
		return srv, nil
	})
	if err != nil {
		t.Fatalf("ListenWorkers: %v", err)
	}
	defer group.Close()

	if group.Count() != 2 {
		t.Fatalf("worker count = %d, want 2", group.Count())
	}
	if group.Port() == 0 {
		t.Fatalf("shared port was not assigned")
	}
	for _, port := range group.Ports() {
		if port != group.Port() {
			t.Fatalf("worker port = %d, want shared port %d", port, group.Port())
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- group.Serve(ctx)
	}()

	conn := dialServer(t, group.Port())
	defer conn.Close()
	if _, err := conn.Write([]byte("GET /plaintext HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n")); err != nil {
		t.Fatalf("client write: %v", err)
	}
	got := readUntil(t, conn, func(s string) bool {
		return strings.Contains(s, "HTTP/1.1 200 OK") && strings.Contains(s, "worker-")
	})
	if !strings.Contains(got, "Connection: close") {
		t.Fatalf("worker response missing close header:\n%s", got)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil && err != context.Canceled {
			t.Fatalf("worker group Serve returned %v", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("worker group did not stop")
	}
}

func TestListenWorkersClosesAlreadyStartedWorkersWhenLaterWorkerFails(t *testing.T) {
	var first *Server
	_, err := ListenWorkers(2, 0, func(workerID int, port int) (*Server, error) {
		if workerID == 1 {
			return nil, fmt.Errorf("worker %d cannot start", workerID)
		}
		first = NewServer(Config{
			Address: [4]byte{127, 0, 0, 1},
			Port:    port,
		})
		return first, nil
	})
	if err == nil || !strings.Contains(err.Error(), "worker 1") {
		t.Fatalf("ListenWorkers failure = %v, want worker 1 error", err)
	}
	if first == nil {
		t.Fatalf("first worker was not constructed")
	}
	if first.Port() != 0 {
		t.Fatalf("first worker port = %d, want closed zero state", first.Port())
	}
}
