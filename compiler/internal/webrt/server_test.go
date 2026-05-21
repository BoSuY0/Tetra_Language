package webrt

import (
	"context"
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"tetra_language/compiler/internal/httprt"
)

func TestServerPlaintextKeepAliveAndPipelining(t *testing.T) {
	srv, stop := startBenchmarkServer(t)
	defer stop()

	conn := dialServer(t, srv.Port())
	defer conn.Close()
	raw := "GET /plaintext HTTP/1.1\r\nHost: localhost\r\nConnection: keep-alive\r\n\r\n" +
		"GET /plaintext HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"
	if _, err := conn.Write([]byte(raw)); err != nil {
		t.Fatalf("client write: %v", err)
	}

	got := readUntil(t, conn, func(s string) bool {
		return strings.Count(s, "HTTP/1.1 200 OK") == 2 &&
			strings.Count(s, "Hello, World!") == 2
	})
	for _, want := range []string{
		"Server: Tetra-Test",
		"Date: Wed, 20 May 2026 12:00:00 GMT",
		"Content-Type: text/plain",
		"Content-Length: 13",
		"Connection: keep-alive",
		"Connection: close",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("response missing %q:\n%s", want, got)
		}
	}
}

func TestServerJSONEndpointKeepAliveAndPipelining(t *testing.T) {
	srv, stop := startBenchmarkServer(t)
	defer stop()

	conn := dialServer(t, srv.Port())
	defer conn.Close()
	raw := "GET /json HTTP/1.1\r\nHost: localhost\r\nConnection: keep-alive\r\n\r\n" +
		"GET /json HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"
	if _, err := conn.Write([]byte(raw)); err != nil {
		t.Fatalf("client write: %v", err)
	}

	got := readUntil(t, conn, func(s string) bool {
		return strings.Count(s, "HTTP/1.1 200 OK") == 2 &&
			strings.Count(s, `{"message":"Hello, World!"}`) == 2
	})
	for _, want := range []string{
		"Content-Type: application/json",
		"Content-Length: 27",
		"Connection: keep-alive",
		"Connection: close",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("response missing %q:\n%s", want, got)
		}
	}
	body := responseBody(t, got)
	var decoded struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("json.Unmarshal body %q: %v\nfull response:\n%s", body, err, got)
	}
	if decoded.Message != "Hello, World!" {
		t.Fatalf("decoded message = %q", decoded.Message)
	}
}

func TestServerHandlesPartialRequestRead(t *testing.T) {
	srv, stop := startBenchmarkServer(t)
	defer stop()

	conn := dialServer(t, srv.Port())
	defer conn.Close()
	if _, err := conn.Write([]byte("GET /plain")); err != nil {
		t.Fatalf("partial write 1: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := conn.Write([]byte("text HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n")); err != nil {
		t.Fatalf("partial write 2: %v", err)
	}

	got := readUntil(t, conn, func(s string) bool {
		return strings.Contains(s, "HTTP/1.1 200 OK") && strings.Contains(s, "Hello, World!")
	})
	if !strings.Contains(got, "Connection: close") {
		t.Fatalf("partial response missing close header:\n%s", got)
	}
}

func TestServerRejectsMalformedRequest(t *testing.T) {
	srv, stop := startBenchmarkServer(t)
	defer stop()

	conn := dialServer(t, srv.Port())
	defer conn.Close()
	if _, err := conn.Write([]byte("GET /missing-version\r\nHost: localhost\r\n\r\n")); err != nil {
		t.Fatalf("client write malformed request: %v", err)
	}

	got := readUntil(t, conn, func(s string) bool {
		return strings.Contains(s, "HTTP/1.1 400 Bad Request")
	})
	if !strings.Contains(got, "Connection: close") {
		t.Fatalf("malformed response missing close header:\n%s", got)
	}
}

func startBenchmarkServer(t *testing.T) (*Server, func()) {
	t.Helper()
	srv := NewServer(Config{
		Address:    [4]byte{127, 0, 0, 1},
		Port:       0,
		ServerName: "Tetra-Test",
		DateFunc: func() string {
			return "Wed, 20 May 2026 12:00:00 GMT"
		},
	})
	srv.Router.Handle("GET", "/plaintext", func(req httprt.Request) httprt.Response {
		return httprt.Response{
			StatusCode:  200,
			ContentType: "text/plain",
			Body:        []byte("Hello, World!"),
		}
	})
	srv.Router.Handle("GET", "/json", JSONMessageHandler("Hello, World!"))
	if err := srv.Listen(); err != nil {
		t.Fatalf("Listen: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- srv.Serve(ctx)
	}()
	stop := func() {
		cancel()
		if err := srv.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		select {
		case err := <-done:
			if err != nil && err != context.Canceled {
				t.Fatalf("Serve returned %v", err)
			}
		case <-time.After(time.Second):
			t.Fatalf("server did not stop")
		}
	}
	return srv, stop
}

func dialServer(t *testing.T, port int) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), time.Second)
	if err != nil {
		t.Fatalf("DialTimeout: %v", err)
	}
	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetDeadline: %v", err)
	}
	return conn
}

func readUntil(t *testing.T, conn net.Conn, done func(string) bool) string {
	t.Helper()
	var b strings.Builder
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			b.Write(buf[:n])
			if done(b.String()) {
				return b.String()
			}
		}
		if err != nil {
			t.Fatalf("read response before condition was met: %v\n%s", err, b.String())
		}
	}
}

func responseBody(t *testing.T, raw string) string {
	t.Helper()
	idx := strings.Index(raw, "\r\n\r\n")
	if idx < 0 {
		t.Fatalf("response missing body separator:\n%s", raw)
	}
	end := idx + len("\r\n\r\n")
	next := strings.Index(raw[end:], "HTTP/1.1 ")
	if next >= 0 {
		return raw[end : end+next]
	}
	return raw[end:]
}
