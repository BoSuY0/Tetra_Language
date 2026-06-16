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

func TestNetRuntimeSocketLifecycleBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        if fd < 0:
            return 11
        let nb: Int = core.net_set_nonblocking(fd, cap)
        let closed: Int = core.net_close(fd, cap)
        if nb < 0:
            return 12
        if closed != 0:
            return 13
    return 0
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want networking socket lifecycle smoke success", exitCode)
	}
}

func TestNetRuntimeSocketOptionsBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        if fd < 0:
            return 21
        let reuse: Int = core.net_set_reuseport(fd, cap)
        let nodelay: Int = core.net_set_tcp_nodelay(fd, cap)
        let closed: Int = core.net_close(fd, cap)
        if reuse != 0:
            return 22
        if nodelay != 0:
            return 23
        if closed != 0:
            return 24
    return 0
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want networking socket options smoke success", exitCode)
	}
}

func TestNetRuntimeEpollControlLifecycleBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        if fd < 0:
            return 31
        let epfd: Int = core.net_epoll_create(cap)
        if epfd < 0:
            let close_fd: Int = core.net_close(fd, cap)
            return 32
        let add_rw: Int = core.net_epoll_ctl_add_read_write(epfd, fd, cap)
        let mod_read: Int = core.net_epoll_ctl_mod_read(epfd, fd, cap)
        let mod_rw: Int = core.net_epoll_ctl_mod_read_write(epfd, fd, cap)
        let deleted: Int = core.net_epoll_ctl_delete(epfd, fd, cap)
        let epfd_closed: Int = core.net_close(epfd, cap)
        let fd_closed: Int = core.net_close(fd, cap)
        if add_rw != 0:
            return 33
        if mod_read != 0:
            return 34
        if mod_rw != 0:
            return 35
        if deleted != 0:
            return 36
        if epfd_closed != 0 || fd_closed != 0:
            return 37
    return 0
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want epoll control lifecycle smoke success", exitCode)
	}
}

func TestNetRuntimeTCPClientConnectWriteBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	ln, err := netListenTCP4Localhost()
	if err != nil {
		t.Fatalf("listen local TCP server: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	accepted := make(chan error, 1)
	go func() {
		conn, err := ln.AcceptTCP()
		if err != nil {
			accepted <- err
			return
		}
		defer conn.Close()
		if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
			accepted <- err
			return
		}
		got := make([]byte, 2)
		if _, err := io.ReadFull(conn, got); err != nil {
			accepted <- err
			return
		}
		if string(got) != "PG" {
			accepted <- fmt.Errorf("server read %q, want PG", got)
			return
		}
		accepted <- nil
	}()

	stdout, exitCode := buildAndRunWithOptions(t, fmt.Sprintf(`
func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        if fd < 0:
            return 41
        if core.net_connect_tcp4_loopback(fd, %d, cap) != 0:
            let close_connect: Int = core.net_close(fd, cap)
            return 42
        var payload: []u8 = core.make_u8(2)
        payload[0] = 80
        payload[1] = 71
        let written: Int = core.net_write(fd, payload, 0, 2, cap)
        let closed: Int = core.net_close(fd, cap)
        if written != 2:
            return 43
        if closed != 0:
            return 44
    return 0
`, port), BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want TCP client connect/write smoke success", exitCode)
	}
	select {
	case err := <-accepted:
		if err != nil {
			t.Fatalf("accept/read from Tetra client: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("server did not receive Tetra client connection")
	}
}

func TestNetRuntimeTCPServerRecvSendBuildAndRunLinuxX64(t *testing.T) {
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
	srcPath := filepath.Join(tmp, "net_recv_send_server.tetra")
	src := fmt.Sprintf(`
func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let cap: cap.io = core.cap_io()
        let server: Int = core.net_socket_tcp4(cap)
        if server < 0:
            return 50
        if core.net_bind_tcp4_loopback(server, %d, cap) < 0:
            let close_bind: Int = core.net_close(server, cap)
            return 51
        if core.net_listen(server, 8, cap) < 0:
            let close_listen: Int = core.net_close(server, cap)
            return 52
        let client: Int = core.net_accept4(server, 0, cap)
        if client < 0:
            let close_accept: Int = core.net_close(server, cap)
            return 53
        var req: []u8 = core.make_u8(8)
        let n: Int = core.net_recv(client, req, 0, 8, cap)
        if n != 4:
            let close_short_client: Int = core.net_close(client, cap)
            let close_short_server: Int = core.net_close(server, cap)
            return 54
        if req[0] != 80 || req[1] != 79 || req[2] != 83 || req[3] != 84:
            let close_bad_client: Int = core.net_close(client, cap)
            let close_bad_server: Int = core.net_close(server, cap)
            return 55
        var resp: []u8 = core.make_u8(4)
        resp[0] = 80
        resp[1] = 79
        resp[2] = 78
        resp[3] = 71
        let sent: Int = core.net_send(client, resp, 0, 4, cap)
        let client_closed: Int = core.net_close(client, cap)
        let server_closed: Int = core.net_close(server, cap)
        if sent != 4:
            return 56
        if client_closed != 0:
            return 57
        if server_closed != 0:
            return 58
        return 0
    return 59
`, port)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "net_recv_send_server")
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
	if _, err := conn.Write([]byte("POST")); err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("write client request: %v", err)
	}
	reply := make([]byte, 4)
	if _, err := io.ReadFull(conn, reply); err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("read client reply: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	_ = conn.Close()
	if string(reply) != "PONG" {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("reply = %q, want PONG", reply)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("server exit: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	if stdout.String() != "" || stderr.String() != "" {
		t.Fatalf("server output stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestNetRuntimeTCPServerAcceptReadWriteBuildAndRunLinuxX64(t *testing.T) {
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
	srcPath := filepath.Join(tmp, "net_server.tetra")
	src := fmt.Sprintf(`
func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let cap: cap.io = core.cap_io()
        let server: Int = core.net_socket_tcp4(cap)
        if server < 0:
            return 10
        if core.net_bind_tcp4_loopback(server, %d, cap) < 0:
            let close_bind: Int = core.net_close(server, cap)
            return 11
        if core.net_listen(server, 8, cap) < 0:
            let close_listen: Int = core.net_close(server, cap)
            return 12
        let client: Int = core.net_accept4(server, 0, cap)
        if client < 0:
            let close_accept: Int = core.net_close(server, cap)
            return 13
        var req: []u8 = core.make_u8(16)
        let n: Int = core.net_read(client, req, 0, 16, cap)
        if n != 4:
            let close_short_client: Int = core.net_close(client, cap)
            let close_short_server: Int = core.net_close(server, cap)
            return 14
        if req[0] != 80 || req[1] != 73 || req[2] != 78 || req[3] != 71:
            let close_bad_client: Int = core.net_close(client, cap)
            let close_bad_server: Int = core.net_close(server, cap)
            return 15
        var resp: []u8 = core.make_u8(2)
        resp[0] = 79
        resp[1] = 75
        let written: Int = core.net_write(client, resp, 0, 2, cap)
        let client_closed: Int = core.net_close(client, cap)
        let server_closed: Int = core.net_close(server, cap)
        if written != 2:
            return 16
        if client_closed != 0:
            return 17
        if server_closed != 0:
            return 18
        return 0
    return 19
`, port)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "net_server")
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
