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
	"strings"
	"testing"
	"time"

	"tetra_language/compiler/target"
)

func TestNetRuntimeRequiredSymbolsAndSignatures(t *testing.T) {
	got := requiredNetRuntimeSymbols()
	want := []string{
		"__tetra_net_socket_tcp4",
		"__tetra_net_bind_tcp4_loopback",
		"__tetra_net_connect_tcp4_loopback",
		"__tetra_net_listen",
		"__tetra_net_accept4",
		"__tetra_net_read",
		"__tetra_net_recv",
		"__tetra_net_write",
		"__tetra_net_send",
		"__tetra_net_epoll_create",
		"__tetra_net_epoll_ctl_add_read",
		"__tetra_net_epoll_ctl_add_read_write",
		"__tetra_net_epoll_ctl_mod_read",
		"__tetra_net_epoll_ctl_mod_read_write",
		"__tetra_net_epoll_ctl_delete",
		"__tetra_net_epoll_wait_one",
		"__tetra_net_epoll_wait_one_into",
		"__tetra_net_set_nonblocking",
		"__tetra_net_set_reuseport",
		"__tetra_net_set_tcp_nodelay",
		"__tetra_net_close",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("networking runtime symbols = %#v, want %#v", got, want)
	}
	tests := []struct {
		name   string
		params int
		rets   int
	}{
		{name: "__tetra_net_socket_tcp4", params: 1, rets: 1},
		{name: "__tetra_net_bind_tcp4_loopback", params: 3, rets: 1},
		{name: "__tetra_net_connect_tcp4_loopback", params: 3, rets: 1},
		{name: "__tetra_net_listen", params: 3, rets: 1},
		{name: "__tetra_net_accept4", params: 3, rets: 1},
		{name: "__tetra_net_read", params: 6, rets: 1},
		{name: "__tetra_net_recv", params: 6, rets: 1},
		{name: "__tetra_net_write", params: 6, rets: 1},
		{name: "__tetra_net_send", params: 6, rets: 1},
		{name: "__tetra_net_epoll_create", params: 1, rets: 1},
		{name: "__tetra_net_epoll_ctl_add_read", params: 3, rets: 1},
		{name: "__tetra_net_epoll_ctl_add_read_write", params: 3, rets: 1},
		{name: "__tetra_net_epoll_ctl_mod_read", params: 3, rets: 1},
		{name: "__tetra_net_epoll_ctl_mod_read_write", params: 3, rets: 1},
		{name: "__tetra_net_epoll_ctl_delete", params: 3, rets: 1},
		{name: "__tetra_net_epoll_wait_one", params: 3, rets: 1},
		{name: "__tetra_net_epoll_wait_one_into", params: 5, rets: 1},
		{name: "__tetra_net_set_nonblocking", params: 2, rets: 1},
		{name: "__tetra_net_set_reuseport", params: 2, rets: 1},
		{name: "__tetra_net_set_tcp_nodelay", params: 2, rets: 1},
		{name: "__tetra_net_close", params: 2, rets: 1},
	}
	for _, tt := range tests {
		sig, ok := runtimeObjectSignature(tt.name)
		if !ok {
			t.Fatalf("missing runtime signature for %s", tt.name)
		}
		if sig.paramSlots != tt.params || sig.returnSlots != tt.rets {
			t.Fatalf("%s signature = params %d returns %d, want params %d returns %d", tt.name, sig.paramSlots, sig.returnSlots, tt.params, tt.rets)
		}
	}
}

func TestCollectNetRuntimeUsage(t *testing.T) {
	prog, err := Parse([]byte(`
func probe(cap: cap.io) -> Int
uses io:
    let fd: Int = core.net_socket_tcp4(cap)
    return core.net_close(fd, cap)

func main() -> Int:
    return 0
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !collectNetRuntimeUsage(checked) {
		t.Fatalf("networking runtime usage was not collected")
	}
}

func TestValidateNetRuntimeObjectChecksSignatureMetadata(t *testing.T) {
	obj := runtimeObjectWithNetRuntimeSignatures()
	if err := validateNetRuntimeObject(obj); err != nil {
		t.Fatalf("validate networking runtime object: %v", err)
	}

	replaceRuntimeSymbolSignature(obj, "__tetra_net_set_nonblocking", 1, 1)
	err := validateNetRuntimeObject(obj)
	if err == nil {
		t.Fatalf("expected networking runtime signature mismatch")
	}
	if !strings.Contains(err.Error(), "runtime object symbol '__tetra_net_set_nonblocking' signature mismatch") ||
		!strings.Contains(err.Error(), "params=1 want=2") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeObjectOverrideRejectsMissingNetSymbols(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if tgt.Triple != "linux-x64" {
		t.Skipf("networking runtime is linux-x64 only, host is %s", tgt.Triple)
	}

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime_missing_net.tobj")
	if err := WriteObject(rtPath, &Object{
		Target:  tgt.Triple,
		Module:  "__runtime_missing_net",
		Code:    []byte{0xC3},
		Symbols: runtimeObjectSymbols(requiredActorRuntimeSymbols()),
	}); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	srcPath := filepath.Join(tmp, "net_main.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        return core.net_close(fd, cap)
    return 1
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	outPath := filepath.Join(tmp, "net_main"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{RuntimeObjectPath: rtPath})
	if err == nil {
		t.Fatalf("expected missing networking runtime symbol failure")
	}
	if !strings.Contains(err.Error(), "runtime object missing required symbol '__tetra_net_socket_tcp4'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

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

func TestNetRuntimeHTTPPlaintextServerBuildAndRunLinuxX64(t *testing.T) {
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
	srcPath := filepath.Join(srcDir, "http_plaintext_server.tetra")
	src := fmt.Sprintf(`
module test.http_plaintext_server

import lib.core.capability as capability
import lib.core.http as http
import lib.core.net as net

func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let io_cap: cap.io = capability.io()
        let server: Int = net.socket_tcp4(io_cap)
        if server < 0:
            return 40
        if net.set_nonblocking(server, io_cap) < 0:
            let close_nonblocking: Int = net.close(server, io_cap)
            return 41
        if net.bind_tcp4_loopback(server, %d, io_cap) < 0:
            let close_bind: Int = net.close(server, io_cap)
            return 42
        if net.listen(server, 8, io_cap) < 0:
            let close_listen: Int = net.close(server, io_cap)
            return 43
        let epfd: Int = net.epoll_create(io_cap)
        if epfd < 0:
            let close_epoll_server: Int = net.close(server, io_cap)
            return 44
        if net.epoll_ctl_add_read(epfd, server, io_cap) < 0:
            let close_ctl_epfd: Int = net.close(epfd, io_cap)
            let close_ctl_server: Int = net.close(server, io_cap)
            return 45
        let ready: Int = net.epoll_wait_one(epfd, 3000, io_cap)
        if ready != server:
            let close_wait_epfd: Int = net.close(epfd, io_cap)
            let close_wait_server: Int = net.close(server, io_cap)
            return 46
        let client: Int = net.accept(server, io_cap)
        if client < 0:
            let close_accept_epfd: Int = net.close(epfd, io_cap)
            let close_accept_server: Int = net.close(server, io_cap)
            return 47
        var req: []u8 = core.make_u8(512)
        let n: Int = net.read(client, req, 0, 512, io_cap)
        if n <= 0:
            let close_empty_client: Int = net.close(client, io_cap)
            let close_empty_epfd: Int = net.close(epfd, io_cap)
            let close_empty_server: Int = net.close(server, io_cap)
            return 48
        let route: Int = http.route_tech_empower_bytes(req, n)
        if route != http.route_plaintext():
            let close_bad_client: Int = net.close(client, io_cap)
            let close_bad_epfd: Int = net.close(epfd, io_cap)
            let close_bad_server: Int = net.close(server, io_cap)
            return 49
        var resp: []u8 = core.make_u8(192)
        let written: Int = http.write_plaintext_response(resp, "Tetra", "Mon, 01 Jan 2024 00:00:00 GMT", false)
        let sent: Int = net.write(client, resp, 0, written, io_cap)
        let client_closed: Int = net.close(client, io_cap)
        let epfd_closed: Int = net.close(epfd, io_cap)
        let server_closed: Int = net.close(server, io_cap)
        if sent != written:
            return 50
        if client_closed != 0 || epfd_closed != 0 || server_closed != 0:
            return 51
        return 0
    return 52
`, port)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "http_plaintext_server")
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
	request := "GET /plaintext HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"
	if _, err := conn.Write([]byte(request)); err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("write client request: %v", err)
	}
	response, err := io.ReadAll(conn)
	if err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("read client response: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	_ = conn.Close()
	got := string(response)
	for _, want := range []string{
		"HTTP/1.1 200 OK\r\n",
		"Server: Tetra\r\n",
		"Date: Mon, 01 Jan 2024 00:00:00 GMT\r\n",
		"Content-Type: text/plain\r\n",
		"Content-Length: 13\r\n",
		"Connection: close\r\n",
		"\r\nHello, World!",
	} {
		if !strings.Contains(got, want) {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			t.Fatalf("response missing %q:\n%s", want, got)
		}
	}
	err = cmd.Wait()
	if ctx.Err() != nil {
		t.Fatalf("server timed out; stdout=%q stderr=%q response=%q", stdout.String(), stderr.String(), got)
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			t.Fatalf("server exit code %d; stdout=%q stderr=%q response=%q", exit.ExitCode(), stdout.String(), stderr.String(), got)
		}
		t.Fatalf("server wait: %v; stdout=%q stderr=%q response=%q", err, stdout.String(), stderr.String(), got)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout mismatch: %q", stdout.String())
	}
}

func TestNetRuntimeHTTPPipelinedPlaintextJSONBuildAndRunLinuxX64(t *testing.T) {
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
	srcPath := filepath.Join(srcDir, "http_pipeline_server.tetra")
	src := fmt.Sprintf(`
module test.http_pipeline_server

import lib.core.capability as capability
import lib.core.http as http
import lib.core.net as net

func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let io_cap: cap.io = capability.io()
        let server: Int = net.socket_tcp4(io_cap)
        if server < 0:
            return 60
        if net.set_nonblocking(server, io_cap) < 0:
            let close_nonblocking: Int = net.close(server, io_cap)
            return 61
        if net.bind_tcp4_loopback(server, %d, io_cap) < 0:
            let close_bind: Int = net.close(server, io_cap)
            return 62
        if net.listen(server, 8, io_cap) < 0:
            let close_listen: Int = net.close(server, io_cap)
            return 63
        let epfd: Int = net.epoll_create(io_cap)
        if epfd < 0:
            let close_epoll_server: Int = net.close(server, io_cap)
            return 64
        if net.epoll_ctl_add_read(epfd, server, io_cap) < 0:
            let close_ctl_epfd: Int = net.close(epfd, io_cap)
            let close_ctl_server: Int = net.close(server, io_cap)
            return 65
        let ready: Int = net.epoll_wait_one(epfd, 3000, io_cap)
        if ready != server:
            let close_wait_epfd: Int = net.close(epfd, io_cap)
            let close_wait_server: Int = net.close(server, io_cap)
            return 66
        let client: Int = net.accept(server, io_cap)
        if client < 0:
            let close_accept_epfd: Int = net.close(epfd, io_cap)
            let close_accept_server: Int = net.close(server, io_cap)
            return 67
        var req: []u8 = core.make_u8(768)
        let n: Int = net.read(client, req, 0, 768, io_cap)
        if n <= 0:
            let close_empty_client: Int = net.close(client, io_cap)
            let close_empty_epfd: Int = net.close(epfd, io_cap)
            let close_empty_server: Int = net.close(server, io_cap)
            return 68
        let first_len: Int = http.request_head_len_bytes(req, n)
        if first_len <= 0:
            let close_first_client: Int = net.close(client, io_cap)
            let close_first_epfd: Int = net.close(epfd, io_cap)
            let close_first_server: Int = net.close(server, io_cap)
            return 69
        let second_len: Int = http.request_head_len_bytes_at(req, first_len, n - first_len)
        if second_len <= 0:
            let close_second_client: Int = net.close(client, io_cap)
            let close_second_epfd: Int = net.close(epfd, io_cap)
            let close_second_server: Int = net.close(server, io_cap)
            return 70
        let first_route: Int = http.route_tech_empower_bytes_at(req, 0, first_len)
        let second_route: Int = http.route_tech_empower_bytes_at(req, first_len, second_len)
        if first_route != http.route_plaintext() || second_route != http.route_json():
            let close_route_client: Int = net.close(client, io_cap)
            let close_route_epfd: Int = net.close(epfd, io_cap)
            let close_route_server: Int = net.close(server, io_cap)
            return 71
        let first_keep_alive: Bool = http.request_keep_alive_bytes_at(req, 0, first_len)
        let second_keep_alive: Bool = http.request_keep_alive_bytes_at(req, first_len, second_len)
        if !first_keep_alive || second_keep_alive:
            let close_keep_client: Int = net.close(client, io_cap)
            let close_keep_epfd: Int = net.close(epfd, io_cap)
            let close_keep_server: Int = net.close(server, io_cap)
            return 72
        var plain: []u8 = core.make_u8(192)
        var json: []u8 = core.make_u8(192)
        let plain_len: Int = http.write_plaintext_response(plain, "Tetra", "Mon, 01 Jan 2024 00:00:00 GMT", true)
        let json_len: Int = http.write_json_message_response(json, "Tetra", "Mon, 01 Jan 2024 00:00:00 GMT", "Hello, World!", false)
        let plain_sent: Int = net.write(client, plain, 0, plain_len, io_cap)
        let json_sent: Int = net.write(client, json, 0, json_len, io_cap)
        let client_closed: Int = net.close(client, io_cap)
        let epfd_closed: Int = net.close(epfd, io_cap)
        let server_closed: Int = net.close(server, io_cap)
        if plain_sent != plain_len || json_sent != json_len:
            return 73
        if client_closed != 0 || epfd_closed != 0 || server_closed != 0:
            return 74
        return 0
    return 75
`, port)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "http_pipeline_server")
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
	request := "GET /plaintext HTTP/1.1\r\nHost: localhost\r\n\r\n" +
		"GET /json HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"
	if _, err := conn.Write([]byte(request)); err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("write client request: %v", err)
	}
	response, err := io.ReadAll(conn)
	if err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("read client response: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	_ = conn.Close()
	got := string(response)
	for _, want := range []string{
		"HTTP/1.1 200 OK\r\nServer: Tetra\r\nDate: Mon, 01 Jan 2024 00:00:00 GMT\r\nContent-Type: text/plain\r\nContent-Length: 13\r\nConnection: keep-alive\r\n\r\nHello, World!",
		"HTTP/1.1 200 OK\r\nServer: Tetra\r\nDate: Mon, 01 Jan 2024 00:00:00 GMT\r\nContent-Type: application/json\r\nContent-Length: 27\r\nConnection: close\r\n\r\n{\"message\":\"Hello, World!\"}",
	} {
		if !strings.Contains(got, want) {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			t.Fatalf("response missing %q:\n%s", want, got)
		}
	}
	err = cmd.Wait()
	if ctx.Err() != nil {
		t.Fatalf("server timed out; stdout=%q stderr=%q response=%q", stdout.String(), stderr.String(), got)
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			t.Fatalf("server exit code %d; stdout=%q stderr=%q response=%q", exit.ExitCode(), stdout.String(), stderr.String(), got)
		}
		t.Fatalf("server wait: %v; stdout=%q stderr=%q response=%q", err, stdout.String(), stderr.String(), got)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout mismatch: %q", stdout.String())
	}
}

func TestNetRuntimeRejectsUnsupportedNativeTargets(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_main.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        return core.net_close(fd, cap)
    return 1
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	for _, tc := range []struct {
		target string
		want   string
	}{
		{target: "macos-x64", want: "macos-x64"},
		{target: "windows-x64", want: "windows-x64"},
		{target: "x32", want: "linux-x32"},
	} {
		t.Run(tc.target, func(t *testing.T) {
			outPath := filepath.Join(tmp, "net-"+tc.target)
			_, err := BuildFileWithStatsOpt(srcPath, outPath, tc.target, BuildOptions{Jobs: 1})
			if err == nil {
				t.Fatalf("expected unsupported networking runtime diagnostic")
			}
			want := "networking runtime not supported on " + tc.want
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want %q", err, want)
			}
		})
	}
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
