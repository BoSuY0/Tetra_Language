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
	"strings"
	"syscall"
	"testing"
	"time"
)

func assertELF32Machine(t *testing.T, path string, label string, wantMachine uint16) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s executable: %v", label, err)
	}
	if len(data) < 20 {
		t.Fatalf("%s executable too small: %d bytes", label, len(data))
	}
	if string(data[:4]) != "\x7fELF" {
		t.Fatalf("%s executable missing ELF magic: % x", label, data[:4])
	}
	if data[4] != 1 {
		t.Fatalf("%s executable must use ELFCLASS32, got %d", label, data[4])
	}
	if got := uint16(data[18]) | uint16(data[19])<<8; got != wantMachine {
		t.Fatalf("%s executable machine = %#x, want %#x", label, got, wantMachine)
	}
}

type targetNetworkingSmoke struct {
	target      string
	label       string
	wantMachine uint16
}

func testTargetNetworkingSocketOptions(t *testing.T, smoke targetNetworkingSmoke) {
	t.Helper()
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_options_"+smoke.label+".tetra")
	outPath := filepath.Join(tmp, "net-options-"+smoke.label)
	if err := os.WriteFile(srcPath, []byte(`
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
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, smoke.target, BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build %s net socket options runtime: %v", smoke.label, err)
	}
	assertELF32Machine(t, outPath, smoke.label+" net socket options", smoke.wantMachine)
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if stdout != "" || code != 0 {
		t.Fatalf("%s net socket options runtime stdout=%q exit=%d, want empty/0", smoke.label, stdout, code)
	}
}

func testTargetNetworkingTCPClientReadWrite(t *testing.T, smoke targetNetworkingSmoke) {
	t.Helper()
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
		if _, err := conn.Write([]byte("OK")); err != nil {
			accepted <- err
			return
		}
		accepted <- nil
	}()

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_client_"+smoke.label+".tetra")
	outPath := filepath.Join(tmp, "net-client-"+smoke.label)
	src := fmt.Sprintf(`
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
        if written != 2:
            let close_write: Int = core.net_close(fd, cap)
            return 43
        var reply: []u8 = core.make_u8(2)
        let n: Int = core.net_read(fd, reply, 0, 2, cap)
        let closed: Int = core.net_close(fd, cap)
        if n != 2:
            return 44
        if reply[0] != 79 || reply[1] != 75:
            return 45
        if closed != 0:
            return 46
    return 0
`, port)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, smoke.target, BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build %s net client read/write runtime: %v", smoke.label, err)
	}
	assertELF32Machine(t, outPath, smoke.label+" net client read/write", smoke.wantMachine)
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if stdout != "" || code != 0 {
		t.Fatalf("%s net client read/write runtime stdout=%q exit=%d, want empty/0", smoke.label, stdout, code)
	}
	select {
	case err := <-accepted:
		if err != nil {
			t.Fatalf("accept/read/write from %s Tetra client: %v", smoke.label, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("server did not receive %s Tetra client connection", smoke.label)
	}
}

func testTargetNetworkingTCPServerRecvSend(t *testing.T, smoke targetNetworkingSmoke) {
	t.Helper()
	ln, err := netListenTCP4Localhost()
	if err != nil {
		t.Fatalf("reserve local TCP port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("close port reservation listener: %v", err)
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_server_"+smoke.label+".tetra")
	outPath := filepath.Join(tmp, "net-server-"+smoke.label)
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
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, smoke.target, BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build %s net server recv/send runtime: %v", smoke.label, err)
	}
	assertELF32Machine(t, outPath, smoke.label+" net server recv/send", smoke.wantMachine)
	runTargetTCPServerRecvSendOrSkip(t, outPath, smoke.label, port)
}

func testTargetNetworkingEpollControlLifecycle(t *testing.T, smoke targetNetworkingSmoke) {
	t.Helper()
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_epoll_control_"+smoke.label+".tetra")
	outPath := filepath.Join(tmp, "net-epoll-control-"+smoke.label)
	if err := os.WriteFile(srcPath, []byte(`
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
        let add_read: Int = core.net_epoll_ctl_add_read(epfd, fd, cap)
        let mod_read: Int = core.net_epoll_ctl_mod_read(epfd, fd, cap)
        let mod_rw: Int = core.net_epoll_ctl_mod_read_write(epfd, fd, cap)
        let del_read: Int = core.net_epoll_ctl_delete(epfd, fd, cap)
        let add_rw: Int = core.net_epoll_ctl_add_read_write(epfd, fd, cap)
        let del_rw: Int = core.net_epoll_ctl_delete(epfd, fd, cap)
        let epfd_closed: Int = core.net_close(epfd, cap)
        let fd_closed: Int = core.net_close(fd, cap)
        if add_read != 0:
            return 33
        if mod_read != 0:
            return 34
        if mod_rw != 0:
            return 35
        if del_read != 0:
            return 36
        if add_rw != 0:
            return 37
        if del_rw != 0:
            return 38
        if epfd_closed != 0 || fd_closed != 0:
            return 39
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, smoke.target, BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build %s net epoll control runtime: %v", smoke.label, err)
	}
	assertELF32Machine(t, outPath, smoke.label+" net epoll control", smoke.wantMachine)
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if stdout != "" || code != 0 {
		t.Fatalf("%s net epoll control runtime stdout=%q exit=%d, want empty/0", smoke.label, stdout, code)
	}
}

func testTargetNetworkingEpollReadiness(t *testing.T, smoke targetNetworkingSmoke) {
	t.Helper()
	ln, err := netListenTCP4Localhost()
	if err != nil {
		t.Fatalf("reserve local TCP port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("close port reservation listener: %v", err)
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_epoll_readiness_"+smoke.label+".tetra")
	outPath := filepath.Join(tmp, "net-epoll-readiness-"+smoke.label)
	src := fmt.Sprintf(`
func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let cap: cap.io = core.cap_io()
        let server: Int = core.net_socket_tcp4(cap)
        if server < 0:
            return 60
        if core.net_set_nonblocking(server, cap) < 0:
            let close_nonblocking: Int = core.net_close(server, cap)
            return 61
        if core.net_bind_tcp4_loopback(server, %d, cap) < 0:
            let close_bind: Int = core.net_close(server, cap)
            return 62
        if core.net_listen(server, 8, cap) < 0:
            let close_listen: Int = core.net_close(server, cap)
            return 63
        let epfd: Int = core.net_epoll_create(cap)
        if epfd < 0:
            let close_epoll_server: Int = core.net_close(server, cap)
            return 64
        if core.net_epoll_ctl_add_read(epfd, server, cap) < 0:
            let close_ctl_epfd: Int = core.net_close(epfd, cap)
            let close_ctl_server: Int = core.net_close(server, cap)
            return 65
        let ready: Int = core.net_epoll_wait_one(epfd, 3000, cap)
        if ready != server:
            let close_ready_epfd: Int = core.net_close(epfd, cap)
            let close_ready_server: Int = core.net_close(server, cap)
            return 66
        var event: []i32 = core.make_i32(2)
        let status: Int = core.net_epoll_wait_one_into(epfd, event, 3000, cap)
        if status != 1:
            let close_status_epfd: Int = core.net_close(epfd, cap)
            let close_status_server: Int = core.net_close(server, cap)
            return 67
        if event[0] != server:
            let close_fd_epfd: Int = core.net_close(epfd, cap)
            let close_fd_server: Int = core.net_close(server, cap)
            return 68
        if event[1] %% 2 != 1:
            let close_flags_epfd: Int = core.net_close(epfd, cap)
            let close_flags_server: Int = core.net_close(server, cap)
            return 69
        let client: Int = core.net_accept4(server, 0, cap)
        if client < 0:
            let close_accept_epfd: Int = core.net_close(epfd, cap)
            let close_accept_server: Int = core.net_close(server, cap)
            return 70
        let client_closed: Int = core.net_close(client, cap)
        let epfd_closed: Int = core.net_close(epfd, cap)
        let server_closed: Int = core.net_close(server, cap)
        if client_closed != 0 || epfd_closed != 0 || server_closed != 0:
            return 71
        return 0
    return 72
`, port)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, smoke.target, BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build %s net epoll readiness runtime: %v", smoke.label, err)
	}
	assertELF32Machine(t, outPath, smoke.label+" net epoll readiness", smoke.wantMachine)
	runTargetTCPServerReadinessOrSkip(t, outPath, smoke.label, port)
}

func runTargetTCPServerReadinessOrSkip(t *testing.T, outPath string, label string, port int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		if isUnsupportedTargetExecError(err, stdout.String()+stderr.String()) {
			t.Skipf("host cannot execute %s target binary %s: %v", label, outPath, err)
		}
		t.Fatalf("start %s readiness server: %v", label, err)
	}
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()
	conn, waitResult, err := dialTCP4LocalhostOrTargetExit(ctx, port, waitCh)
	if err != nil {
		if waitResult != nil {
			handleTargetProcessExitBeforeDial(t, outPath, label+" readiness server", waitResult.err, stdout.String(), stderr.String())
		}
		_ = cmd.Process.Kill()
		<-waitCh
		t.Fatalf("dial %s readiness server: %v; stdout=%q stderr=%q", label, err, stdout.String(), stderr.String())
	}
	_ = conn.Close()
	err = <-waitCh
	if ctx.Err() != nil {
		t.Fatalf("%s readiness server timed out; stdout=%q stderr=%q", label, stdout.String(), stderr.String())
	}
	if err != nil {
		handleTargetProcessWaitError(t, outPath, label+" readiness server", err, stdout.String(), stderr.String())
	}
	if stdout.String() != "" || stderr.String() != "" {
		t.Fatalf("%s readiness server output stdout=%q stderr=%q", label, stdout.String(), stderr.String())
	}
}

func runTargetTCPServerRecvSendOrSkip(t *testing.T, outPath string, label string, port int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		if isUnsupportedTargetExecError(err, stdout.String()+stderr.String()) {
			t.Skipf("host cannot execute %s target binary %s: %v", label, outPath, err)
		}
		t.Fatalf("start %s server: %v", label, err)
	}
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()
	conn, waitResult, err := dialTCP4LocalhostOrTargetExit(ctx, port, waitCh)
	if err != nil {
		if waitResult != nil {
			handleTargetProcessExitBeforeDial(t, outPath, label+" server", waitResult.err, stdout.String(), stderr.String())
		}
		_ = cmd.Process.Kill()
		<-waitCh
		t.Fatalf("dial %s server: %v; stdout=%q stderr=%q", label, err, stdout.String(), stderr.String())
	}
	if _, err := conn.Write([]byte("POST")); err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		<-waitCh
		t.Fatalf("write %s client request: %v", label, err)
	}
	reply := make([]byte, 4)
	if _, err := io.ReadFull(conn, reply); err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		<-waitCh
		t.Fatalf("read %s client reply: %v; stdout=%q stderr=%q", label, err, stdout.String(), stderr.String())
	}
	_ = conn.Close()
	if string(reply) != "PONG" {
		_ = cmd.Process.Kill()
		<-waitCh
		t.Fatalf("%s reply = %q, want PONG", label, reply)
	}
	err = <-waitCh
	if ctx.Err() != nil {
		t.Fatalf("%s server timed out; stdout=%q stderr=%q", label, stdout.String(), stderr.String())
	}
	if err != nil {
		handleTargetProcessWaitError(t, outPath, label+" server", err, stdout.String(), stderr.String())
	}
	if stdout.String() != "" || stderr.String() != "" {
		t.Fatalf("%s server output stdout=%q stderr=%q", label, stdout.String(), stderr.String())
	}
}

type targetWaitResult struct {
	err error
}

func dialTCP4LocalhostOrTargetExit(ctx context.Context, port int, waitCh <-chan error) (*net.TCPConn, *targetWaitResult, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	var lastErr error
	for ctx.Err() == nil {
		select {
		case err := <-waitCh:
			return nil, &targetWaitResult{err: err}, fmt.Errorf("target process exited before accepting TCP connections")
		default:
		}

		dialer := net.Dialer{Timeout: 50 * time.Millisecond}
		conn, err := dialer.DialContext(ctx, "tcp4", addr)
		if err == nil {
			return conn.(*net.TCPConn), nil, nil
		}
		lastErr = err

		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case err := <-waitCh:
			timer.Stop()
			return nil, &targetWaitResult{err: err}, fmt.Errorf("target process exited before accepting TCP connections")
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
	if lastErr != nil {
		return nil, nil, lastErr
	}
	return nil, nil, ctx.Err()
}

func handleTargetProcessExitBeforeDial(t *testing.T, outPath string, label string, err error, stdout string, stderr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s exited before accepting TCP connections; stdout=%q stderr=%q", label, stdout, stderr)
	}
	handleTargetProcessWaitError(t, outPath, label, err, stdout, stderr)
}

func handleTargetProcessWaitError(t *testing.T, outPath string, label string, err error, stdout string, stderr string) {
	t.Helper()
	if isUnsupportedTargetSignalExit(err, syscall.SIGSYS) {
		t.Skipf("host kernel rejected %s target binary %s with SIGSYS; target execution is unsupported in this environment", label, outPath)
	}
	if exit, ok := err.(*exec.ExitError); ok {
		if status, ok := exit.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			t.Fatalf("%s exited from signal %s; stdout=%q stderr=%q", label, status.Signal(), stdout, stderr)
		}
		t.Fatalf("%s exit code %d; stdout=%q stderr=%q", label, exit.ExitCode(), stdout, stderr)
	}
	t.Fatalf("%s wait: %v; stdout=%q stderr=%q", label, err, stdout, stderr)
}

func isUnsupportedTargetSignalExit(err error, signal syscall.Signal) bool {
	if err == nil {
		return false
	}
	exit, ok := err.(*exec.ExitError)
	if !ok {
		return false
	}
	status, ok := exit.Sys().(syscall.WaitStatus)
	return ok && status.Signaled() && status.Signal() == signal
}

func isUnsupportedTargetExecError(err error, output string) bool {
	if err == nil {
		return false
	}
	text := err.Error() + " " + output
	return strings.Contains(text, "exec format error") || strings.Contains(text, "no such file or directory")
}

func runtimeObjectWithNetRuntimeSignatures() *Object {
	obj := &Object{}
	for _, name := range requiredNetRuntimeSymbols() {
		sig, ok := runtimeObjectSignature(name)
		if !ok {
			panic("missing networking runtime signature for " + name)
		}
		obj.Symbols = append(obj.Symbols, Symbol{
			Name:         name,
			HasSignature: true,
			ParamSlots:   sig.paramSlots,
			ReturnSlots:  sig.returnSlots,
		})
	}
	return obj
}

func netListenTCP4Localhost() (*net.TCPListener, error) {
	addr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	return net.ListenTCP("tcp4", addr)
}

func dialTCP4Localhost(ctx context.Context, port int) (*net.TCPConn, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	var lastErr error
	for ctx.Err() == nil {
		dialer := net.Dialer{Timeout: 50 * time.Millisecond}
		conn, err := dialer.DialContext(ctx, "tcp4", addr)
		if err == nil {
			return conn.(*net.TCPConn), nil
		}
		lastErr = err
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, ctx.Err()
}
