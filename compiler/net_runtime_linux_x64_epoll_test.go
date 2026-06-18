package compiler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestNetRuntimeEpollReadinessBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	ln, err := netListenTCP4Localhost()
	if err != nil {
		t.Fatalf("reserve local TCP port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("close port reservation listener: %v", err)
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_epoll_server.tetra")
	src := fmt.Sprintf(`
func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let cap: cap.io = core.cap_io()
        let server: Int = core.net_socket_tcp4(cap)
        if server < 0:
            return 20
        if core.net_set_nonblocking(server, cap) < 0:
            let close_nonblocking: Int = core.net_close(server, cap)
            return 21
        if core.net_bind_tcp4_loopback(server, %d, cap) < 0:
            let close_bind: Int = core.net_close(server, cap)
            return 22
        if core.net_listen(server, 8, cap) < 0:
            let close_listen: Int = core.net_close(server, cap)
            return 23
        let epfd: Int = core.net_epoll_create(cap)
        if epfd < 0:
            let close_epoll_server: Int = core.net_close(server, cap)
            return 24
        if core.net_epoll_ctl_add_read(epfd, server, cap) < 0:
            let close_ctl_epfd: Int = core.net_close(epfd, cap)
            let close_ctl_server: Int = core.net_close(server, cap)
            return 25
        let ready: Int = core.net_epoll_wait_one(epfd, 3000, cap)
        if ready != server:
            let close_wait_epfd: Int = core.net_close(epfd, cap)
            let close_wait_server: Int = core.net_close(server, cap)
            return 26
        let client: Int = core.net_accept4(server, 0, cap)
        if client < 0:
            let close_accept_epfd: Int = core.net_close(epfd, cap)
            let close_accept_server: Int = core.net_close(server, cap)
            return 27
        var req: []u8 = core.make_u8(16)
        let n: Int = core.net_read(client, req, 0, 16, cap)
        if n != 4:
            let close_short_client: Int = core.net_close(client, cap)
            let close_short_epfd: Int = core.net_close(epfd, cap)
            let close_short_server: Int = core.net_close(server, cap)
            return 28
        var resp: []u8 = core.make_u8(2)
        resp[0] = 79
        resp[1] = 75
        let written: Int = core.net_write(client, resp, 0, 2, cap)
        let client_closed: Int = core.net_close(client, cap)
        let epfd_closed: Int = core.net_close(epfd, cap)
        let server_closed: Int = core.net_close(server, cap)
        if written != 2:
            return 29
        if client_closed != 0 || epfd_closed != 0 || server_closed != 0:
            return 30
        return 0
    return 31
`, port)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "net_epoll_server")
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	conn, err := dialTCP4Localhost(ctx, port)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("dial server: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	if _, err := conn.Write([]byte("PING")); err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("write client request: %v", err)
	}
	reply := make([]byte, 2)
	if _, err := io.ReadFull(conn, reply); err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("read client reply: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	_ = conn.Close()
	if string(reply) != "OK" {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("reply = %q, want OK", string(reply))
	}
	err = cmd.Wait()
	if ctx.Err() != nil {
		t.Fatalf("server timed out; stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			t.Fatalf("server exit code %d; stdout=%q stderr=%q", exit.ExitCode(), stdout.String(), stderr.String())
		}
		t.Fatalf("server wait: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("stdout mismatch: %q", stdout.String())
	}
}

func TestNetRuntimeEpollWaitOneIntoBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	ln, err := netListenTCP4Localhost()
	if err != nil {
		t.Fatalf("reserve local TCP port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("close port reservation listener: %v", err)
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_epoll_event_server.tetra")
	src := fmt.Sprintf(`
func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let cap: cap.io = core.cap_io()
        let server: Int = core.net_socket_tcp4(cap)
        if server < 0:
            return 80
        if core.net_set_nonblocking(server, cap) < 0:
            let close_nonblocking: Int = core.net_close(server, cap)
            return 81
        if core.net_bind_tcp4_loopback(server, %d, cap) < 0:
            let close_bind: Int = core.net_close(server, cap)
            return 82
        if core.net_listen(server, 8, cap) < 0:
            let close_listen: Int = core.net_close(server, cap)
            return 83
        let epfd: Int = core.net_epoll_create(cap)
        if epfd < 0:
            let close_epoll_server: Int = core.net_close(server, cap)
            return 84
        if core.net_epoll_ctl_add_read(epfd, server, cap) < 0:
            let close_ctl_epfd: Int = core.net_close(epfd, cap)
            let close_ctl_server: Int = core.net_close(server, cap)
            return 85
        var event: []i32 = core.make_i32(2)
        let status: Int = core.net_epoll_wait_one_into(epfd, event, 3000, cap)
        if status != 1:
            let close_status_epfd: Int = core.net_close(epfd, cap)
            let close_status_server: Int = core.net_close(server, cap)
            return 86
        if event[0] != server:
            let close_fd_epfd: Int = core.net_close(epfd, cap)
            let close_fd_server: Int = core.net_close(server, cap)
            return 87
        if event[1] %% 2 != 1:
            let close_flags_epfd: Int = core.net_close(epfd, cap)
            let close_flags_server: Int = core.net_close(server, cap)
            return 88
        let client: Int = core.net_accept4(server, 0, cap)
        if client < 0:
            let close_accept_epfd: Int = core.net_close(epfd, cap)
            let close_accept_server: Int = core.net_close(server, cap)
            return 89
        let client_closed: Int = core.net_close(client, cap)
        let epfd_closed: Int = core.net_close(epfd, cap)
        let server_closed: Int = core.net_close(server, cap)
        if client_closed != 0 || epfd_closed != 0 || server_closed != 0:
            return 90
        return 0
    return 91
`, port)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "net_epoll_event_server")
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	conn, err := dialTCP4Localhost(ctx, port)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("dial server: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	_ = conn.Close()
	err = cmd.Wait()
	if ctx.Err() != nil {
		t.Fatalf("server timed out; stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			t.Fatalf("server exit code %d; stdout=%q stderr=%q", exit.ExitCode(), stdout.String(), stderr.String())
		}
		t.Fatalf("server wait: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("stdout mismatch: %q", stdout.String())
	}
}

func TestNetStdlibAcceptNonblockingAndEpollFlagHelpersBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	ln, err := netListenTCP4Localhost()
	if err != nil {
		t.Fatalf("reserve local TCP port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("close port reservation listener: %v", err)
	}

	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "test")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	srcPath := filepath.Join(srcDir, "net_stdlib_event_server.tetra")
	src := fmt.Sprintf(`
module test.net_stdlib_event_server

import lib.core.capability as capability
import lib.core.net as net

func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let io_cap: cap.io = capability.io()
        let server: Int = net.socket_tcp4(io_cap)
        if server < 0:
            return 100
        if net.set_nonblocking(server, io_cap) < 0:
            let close_nonblocking: Int = net.close(server, io_cap)
            return 101
        if net.bind_tcp4_loopback(server, %d, io_cap) < 0:
            let close_bind: Int = net.close(server, io_cap)
            return 102
        if net.listen(server, 8, io_cap) < 0:
            let close_listen: Int = net.close(server, io_cap)
            return 103
        let epfd: Int = net.epoll_create(io_cap)
        if epfd < 0:
            let close_epoll_server: Int = net.close(server, io_cap)
            return 104
        if net.epoll_ctl_add_read(epfd, server, io_cap) < 0:
            let close_ctl_epfd: Int = net.close(epfd, io_cap)
            let close_ctl_server: Int = net.close(server, io_cap)
            return 105
        var event: []i32 = core.make_i32(2)
        let status: Int = net.epoll_wait_one_into(epfd, event, 3000, io_cap)
        if status != 1:
            let close_status_epfd: Int = net.close(epfd, io_cap)
            let close_status_server: Int = net.close(server, io_cap)
            return 106
        if net.epoll_event_fd(event) != server:
            let close_fd_epfd: Int = net.close(epfd, io_cap)
            let close_fd_server: Int = net.close(server, io_cap)
            return 107
        let flags: Int = net.epoll_event_flags(event)
        if !net.epoll_event_readable(flags):
            let close_read_epfd: Int = net.close(epfd, io_cap)
            let close_read_server: Int = net.close(server, io_cap)
            return 108
        if net.epoll_event_writable(flags) || net.epoll_event_has_error(flags):
            let close_flags_epfd: Int = net.close(epfd, io_cap)
            let close_flags_server: Int = net.close(server, io_cap)
            return 109
        let client: Int = net.accept_nonblocking(server, io_cap)
        if client < 0:
            let close_accept_epfd: Int = net.close(epfd, io_cap)
            let close_accept_server: Int = net.close(server, io_cap)
            return 110
        let nodelay: Int = net.set_tcp_nodelay(client, io_cap)
        let client_closed: Int = net.close(client, io_cap)
        let epfd_closed: Int = net.close(epfd, io_cap)
        let server_closed: Int = net.close(server, io_cap)
        if nodelay != 0:
            return 111
        if client_closed != 0 || epfd_closed != 0 || server_closed != 0:
            return 112
        return 0
    return 113
`, port)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "net_stdlib_event_server")
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{
		DependencyRoots: []ModuleRoot{{Root: projectRoot(t)}},
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	conn, err := dialTCP4Localhost(ctx, port)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("dial server: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	_ = conn.Close()
	err = cmd.Wait()
	if ctx.Err() != nil {
		t.Fatalf("server timed out; stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			t.Fatalf("server exit code %d; stdout=%q stderr=%q", exit.ExitCode(), stdout.String(), stderr.String())
		}
		t.Fatalf("server wait: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("stdout mismatch: %q", stdout.String())
	}
}
