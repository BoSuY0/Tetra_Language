//go:build linux

package netrt

import (
	"fmt"
	"io"
	"net"
	"syscall"
	"testing"
	"time"
)

func TestListenTCP4AcceptsNonblockingConnections(t *testing.T) {
	listener, err := ListenTCP4(TCPListenConfig{
		Address:     [4]byte{127, 0, 0, 1},
		Port:        0,
		Backlog:     16,
		Nonblocking: true,
		ReuseAddr:   true,
		ReusePort:   true,
		NoDelay:     true,
	})
	if err != nil {
		t.Fatalf("ListenTCP4: %v", err)
	}
	defer Close(listener.FD)
	if listener.Port <= 0 {
		t.Fatalf("listener port = %d, want an ephemeral port", listener.Port)
	}
	assertNonblocking(t, listener.FD, "listener")
	assertSocketFlag(t, listener.FD, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, "SO_REUSEADDR")
	assertSocketFlag(t, listener.FD, syscall.SOL_SOCKET, soReusePort, "SO_REUSEPORT")

	poller, err := NewPoller()
	if err != nil {
		t.Fatalf("NewPoller: %v", err)
	}
	defer poller.Close()
	if err := poller.AddRead(listener.FD); err != nil {
		t.Fatalf("poller.AddRead(listener): %v", err)
	}

	client, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", listener.Port), time.Second)
	if err != nil {
		t.Fatalf("client dial: %v", err)
	}
	defer client.Close()

	events, err := poller.Wait(8, time.Second)
	if err != nil {
		t.Fatalf("poller.Wait(listener): %v", err)
	}
	if !hasReadable(events, listener.FD) {
		t.Fatalf("listener fd %d was not readable in events %#v", listener.FD, events)
	}

	connFD, err := Accept(listener.FD, AcceptConfig{Nonblocking: true, CloseOnExec: true, NoDelay: true})
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	defer Close(connFD)
	assertNonblocking(t, connFD, "accepted connection")
	assertSocketFlag(t, connFD, syscall.IPPROTO_TCP, syscall.TCP_NODELAY, "TCP_NODELAY")
}

func TestPollerSignalsReadableDataAndSyscallReadWriteRoundTrip(t *testing.T) {
	listener, err := ListenTCP4(TCPListenConfig{
		Address:     [4]byte{127, 0, 0, 1},
		Port:        0,
		Backlog:     16,
		Nonblocking: true,
		ReuseAddr:   true,
		NoDelay:     true,
	})
	if err != nil {
		t.Fatalf("ListenTCP4: %v", err)
	}
	defer Close(listener.FD)

	poller, err := NewPoller()
	if err != nil {
		t.Fatalf("NewPoller: %v", err)
	}
	defer poller.Close()
	if err := poller.AddRead(listener.FD); err != nil {
		t.Fatalf("poller.AddRead(listener): %v", err)
	}

	client, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", listener.Port), time.Second)
	if err != nil {
		t.Fatalf("client dial: %v", err)
	}
	defer client.Close()

	if _, err := poller.Wait(8, time.Second); err != nil {
		t.Fatalf("poller.Wait(listener): %v", err)
	}
	connFD, err := Accept(listener.FD, AcceptConfig{Nonblocking: true, CloseOnExec: true, NoDelay: true})
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	defer Close(connFD)
	if err := poller.AddRead(connFD); err != nil {
		t.Fatalf("poller.AddRead(conn): %v", err)
	}

	if _, err := client.Write([]byte("ping")); err != nil {
		t.Fatalf("client write: %v", err)
	}
	events, err := poller.Wait(8, time.Second)
	if err != nil {
		t.Fatalf("poller.Wait(conn readable): %v", err)
	}
	if !hasReadable(events, connFD) {
		t.Fatalf("conn fd %d was not readable in events %#v", connFD, events)
	}

	buf := make([]byte, 8)
	n, err := Read(connFD, buf)
	if err != nil {
		t.Fatalf("Read(conn): %v", err)
	}
	if got := string(buf[:n]); got != "ping" {
		t.Fatalf("Read(conn) = %q, want ping", got)
	}

	if n, err := Write(connFD, []byte("pong")); err != nil || n != len("pong") {
		t.Fatalf("Write(conn) = %d, %v; want %d, nil", n, err, len("pong"))
	}
	reply := make([]byte, 4)
	if _, err := io.ReadFull(client, reply); err != nil {
		t.Fatalf("client read reply: %v", err)
	}
	if string(reply) != "pong" {
		t.Fatalf("client reply = %q, want pong", reply)
	}
}

func TestRecvSendRoundTripOnConnectedTCP(t *testing.T) {
	listener, err := ListenTCP4(TCPListenConfig{
		Address:     [4]byte{127, 0, 0, 1},
		Port:        0,
		Backlog:     16,
		Nonblocking: true,
		ReuseAddr:   true,
		NoDelay:     true,
	})
	if err != nil {
		t.Fatalf("ListenTCP4: %v", err)
	}
	defer Close(listener.FD)

	client, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", listener.Port), time.Second)
	if err != nil {
		t.Fatalf("client dial: %v", err)
	}
	defer client.Close()

	poller, err := NewPoller()
	if err != nil {
		t.Fatalf("NewPoller: %v", err)
	}
	defer poller.Close()
	if err := poller.AddRead(listener.FD); err != nil {
		t.Fatalf("poller.AddRead(listener): %v", err)
	}
	if _, err := poller.Wait(8, time.Second); err != nil {
		t.Fatalf("poller.Wait(listener): %v", err)
	}

	connFD, err := Accept(listener.FD, AcceptConfig{Nonblocking: true, CloseOnExec: true, NoDelay: true})
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	defer Close(connFD)
	if err := poller.AddRead(connFD); err != nil {
		t.Fatalf("poller.AddRead(conn): %v", err)
	}

	if _, err := client.Write([]byte("recv")); err != nil {
		t.Fatalf("client write: %v", err)
	}
	if _, err := poller.Wait(8, time.Second); err != nil {
		t.Fatalf("poller.Wait(conn readable): %v", err)
	}
	buf := make([]byte, 8)
	n, err := Recv(connFD, buf)
	if err != nil {
		t.Fatalf("Recv(conn): %v", err)
	}
	if got := string(buf[:n]); got != "recv" {
		t.Fatalf("Recv(conn) = %q, want recv", got)
	}

	if n, err := Send(connFD, []byte("send")); err != nil || n != len("send") {
		t.Fatalf("Send(conn) = %d, %v; want %d, nil", n, err, len("send"))
	}
	reply := make([]byte, 4)
	if _, err := io.ReadFull(client, reply); err != nil {
		t.Fatalf("client read reply: %v", err)
	}
	if string(reply) != "send" {
		t.Fatalf("client reply = %q, want send", reply)
	}
}

func assertNonblocking(t *testing.T, fd int, name string) {
	t.Helper()
	enabled, err := IsNonblocking(fd)
	if err != nil {
		t.Fatalf("IsNonblocking(%s): %v", name, err)
	}
	if !enabled {
		t.Fatalf("%s fd %d is blocking, want nonblocking", name, fd)
	}
}

func assertSocketFlag(t *testing.T, fd int, level int, opt int, name string) {
	t.Helper()
	got, err := syscall.GetsockoptInt(fd, level, opt)
	if err != nil {
		t.Fatalf("GetsockoptInt(%s): %v", name, err)
	}
	if got != 1 {
		t.Fatalf("%s = %d, want 1", name, got)
	}
}

func hasReadable(events []Event, fd int) bool {
	for _, event := range events {
		if event.FD == fd && event.Readable {
			return true
		}
	}
	return false
}
