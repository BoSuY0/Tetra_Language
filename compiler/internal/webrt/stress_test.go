package webrt

import (
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestServerStressManyConcurrentKeepAliveClients(t *testing.T) {
	srv, stop := startBenchmarkServer(t)
	defer stop()

	const clients = 32
	const requestsPerClient = 16
	var wg sync.WaitGroup
	errc := make(chan error, clients)
	for clientID := 0; clientID < clients; clientID++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			conn := dialServer(t, srv.Port())
			defer conn.Close()
			var raw strings.Builder
			for i := 0; i < requestsPerClient; i++ {
				connection := "keep-alive"
				if i == requestsPerClient-1 {
					connection = "close"
				}
				raw.WriteString("GET /plaintext HTTP/1.1\r\nHost: localhost\r\nConnection: ")
				raw.WriteString(connection)
				raw.WriteString("\r\n\r\n")
			}
			if _, err := io.WriteString(conn, raw.String()); err != nil {
				errc <- fmt.Errorf("client %d write: %w", clientID, err)
				return
			}
			got := readUntil(t, conn, func(s string) bool {
				return strings.Count(s, "HTTP/1.1 200 OK") == requestsPerClient &&
					strings.Count(s, "Hello, World!") == requestsPerClient &&
					strings.Contains(s, "Connection: close")
			})
			if strings.Count(got, "Connection: keep-alive") != requestsPerClient-1 {
				errc <- fmt.Errorf("client %d keep-alive response count mismatch", clientID)
			}
		}(clientID)
	}
	wg.Wait()
	close(errc)
	for err := range errc {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestServerStressPipeliningBurst(t *testing.T) {
	srv, stop := startBenchmarkServer(t)
	defer stop()

	conn := dialServer(t, srv.Port())
	defer conn.Close()
	const requests = 128
	var raw strings.Builder
	for i := 0; i < requests; i++ {
		connection := "keep-alive"
		if i == requests-1 {
			connection = "close"
		}
		raw.WriteString("GET /json HTTP/1.1\r\nHost: localhost\r\nConnection: ")
		raw.WriteString(connection)
		raw.WriteString("\r\n\r\n")
	}
	if _, err := io.WriteString(conn, raw.String()); err != nil {
		t.Fatalf("client write pipelining burst: %v", err)
	}
	got := readUntil(t, conn, func(s string) bool {
		return strings.Count(s, "HTTP/1.1 200 OK") == requests &&
			strings.Count(s, `{"message":"Hello, World!"}`) == requests &&
			strings.Contains(s, "Connection: close")
	})
	if strings.Count(got, "Content-Type: application/json") != requests {
		t.Fatalf("pipelining burst json content-type count mismatch")
	}
}

func TestServerHandlesSlowHeaderDripClient(t *testing.T) {
	srv, stop := startBenchmarkServer(t)
	defer stop()

	conn := dialServer(t, srv.Port())
	defer conn.Close()
	req := "GET /plaintext HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"
	for i := 0; i < len(req); i++ {
		if _, err := conn.Write([]byte{req[i]}); err != nil {
			t.Fatalf("slow client byte %d write: %v", i, err)
		}
		time.Sleep(500 * time.Microsecond)
	}
	got := readUntil(t, conn, func(s string) bool {
		return strings.Contains(s, "HTTP/1.1 200 OK") && strings.Contains(s, "Hello, World!")
	})
	if !strings.Contains(got, "Connection: close") {
		t.Fatalf("slow header drip response missing close:\n%s", got)
	}
}

func TestServerStressClosedClientsDoNotBreakOtherConnections(t *testing.T) {
	srv, stop := startBenchmarkServer(t)
	defer stop()

	const closedClients = 32
	for i := 0; i < closedClients; i++ {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", fmt.Sprint(srv.Port())), time.Second)
		if err != nil {
			t.Fatalf("closed client dial %d: %v", i, err)
		}
		_, _ = io.WriteString(conn, "GET /plaintext HTTP/1.1\r\nHost: localhost\r\n")
		_ = conn.Close()
	}

	conn := dialServer(t, srv.Port())
	defer conn.Close()
	if _, err := io.WriteString(conn, "GET /plaintext HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatalf("healthy client write: %v", err)
	}
	got := readUntil(t, conn, func(s string) bool {
		return strings.Contains(s, "HTTP/1.1 200 OK") && strings.Contains(s, "Hello, World!")
	})
	if !strings.Contains(got, "Connection: close") {
		t.Fatalf("healthy response missing close:\n%s", got)
	}
}
