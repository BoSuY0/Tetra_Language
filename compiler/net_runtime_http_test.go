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
)

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
