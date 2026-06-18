//go:build linux

package surfacehost

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListenAndServeUnixRecordsPeerPID(t *testing.T) {
	socketPath := shortTestSocketPath(t)
	defer os.Remove(socketPath)
	reporting := NewReportingBackend(
		&recordingBackend{nextHandle: 11},
		"wayland",
		socketPath,
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- ListenAndServeUnix(ctx, socketPath, reporting)
	}()
	waitForSocketFile(t, socketPath)

	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		t.Fatalf("dial Surface host socket: %v", err)
	}
	if err := WriteRequest(conn, Request{
		Op:        OpOpen,
		RequestID: 1,
		Width:     320,
		Height:    200,
		Payload:   []byte("Counter"),
	}); err != nil {
		t.Fatalf("write open: %v", err)
	}
	if resp, err := ReadResponse(conn); err != nil || resp.Status != 0 {
		t.Fatalf("open response = %#v, %v", resp, err)
	}
	_ = conn.Close()
	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("ListenAndServeUnix: %v", err)
	}
	if reporting.Snapshot().AppPID != os.Getpid() {
		t.Fatalf("app_pid = %d, want current pid %d", reporting.Snapshot().AppPID, os.Getpid())
	}
}

func waitForSocketFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for socket %s", path)
}

func shortTestSocketPath(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = os.TempDir()
	}
	path := filepath.Join(dir, fmt.Sprintf("tsh-%d-%d.sock", os.Getpid(), time.Now().UnixNano()))
	if len(path) >= 100 {
		t.Fatalf("test socket path too long for Unix domain socket: %s", path)
	}
	return path
}
