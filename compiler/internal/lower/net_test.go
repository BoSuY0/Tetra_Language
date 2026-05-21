package lower

import "testing"

func TestLowerNetBuiltinsUseRuntimeCalls(t *testing.T) {
	prog := lowerCallableProgram(t, `
func probe(cap: cap.io) -> Int
uses alloc, io, mem:
    let fd: Int = core.net_socket_tcp4(cap)
    let bind_status: Int = core.net_bind_tcp4_loopback(fd, 18080, cap)
    let connect_status: Int = core.net_connect_tcp4_loopback(fd, 18080, cap)
    let listen_status: Int = core.net_listen(fd, 8, cap)
    let client: Int = core.net_accept4(fd, 0, cap)
    var buf: []u8 = core.make_u8(8)
    let read_status: Int = core.net_read(client, buf, 0, 8, cap)
    let recv_status: Int = core.net_recv(client, buf, 0, 8, cap)
    let write_status: Int = core.net_write(client, buf, 0, 2, cap)
    let send_status: Int = core.net_send(client, buf, 0, 2, cap)
    let epfd: Int = core.net_epoll_create(cap)
    let epoll_add: Int = core.net_epoll_ctl_add_read(epfd, fd, cap)
    let epoll_add_rw: Int = core.net_epoll_ctl_add_read_write(epfd, fd, cap)
    let epoll_mod_read: Int = core.net_epoll_ctl_mod_read(epfd, fd, cap)
    let epoll_mod_rw: Int = core.net_epoll_ctl_mod_read_write(epfd, fd, cap)
    let epoll_delete: Int = core.net_epoll_ctl_delete(epfd, fd, cap)
    let epoll_ready: Int = core.net_epoll_wait_one(epfd, 0, cap)
    var event: []i32 = core.make_i32(2)
    let epoll_event_ready: Int = core.net_epoll_wait_one_into(epfd, event, 0, cap)
    let nb: Int = core.net_set_nonblocking(fd, cap)
    let reuse: Int = core.net_set_reuseport(fd, cap)
    let nodelay: Int = core.net_set_tcp_nodelay(fd, cap)
    let closed: Int = core.net_close(fd, cap)
    return fd + bind_status + connect_status + listen_status + client + read_status + recv_status + write_status + send_status + epfd + epoll_add + epoll_add_rw + epoll_mod_read + epoll_mod_rw + epoll_delete + epoll_ready + epoll_event_ready + nb + reuse + nodelay + closed

func main() -> Int:
    return 0
`)
	probe := requireCallableFunc(t, prog, "probe")
	if countCall(probe.Instrs, "__tetra_net_socket_tcp4", 1, 1) != 1 {
		t.Fatalf("probe did not lower core.net_socket_tcp4 to __tetra_net_socket_tcp4(1 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_bind_tcp4_loopback", 3, 1) != 1 {
		t.Fatalf("probe did not lower core.net_bind_tcp4_loopback to __tetra_net_bind_tcp4_loopback(3 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_connect_tcp4_loopback", 3, 1) != 1 {
		t.Fatalf("probe did not lower core.net_connect_tcp4_loopback to __tetra_net_connect_tcp4_loopback(3 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_listen", 3, 1) != 1 {
		t.Fatalf("probe did not lower core.net_listen to __tetra_net_listen(3 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_accept4", 3, 1) != 1 {
		t.Fatalf("probe did not lower core.net_accept4 to __tetra_net_accept4(3 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_read", 6, 1) != 1 {
		t.Fatalf("probe did not lower core.net_read to __tetra_net_read(6 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_recv", 6, 1) != 1 {
		t.Fatalf("probe did not lower core.net_recv to __tetra_net_recv(6 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_write", 6, 1) != 1 {
		t.Fatalf("probe did not lower core.net_write to __tetra_net_write(6 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_send", 6, 1) != 1 {
		t.Fatalf("probe did not lower core.net_send to __tetra_net_send(6 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_epoll_create", 1, 1) != 1 {
		t.Fatalf("probe did not lower core.net_epoll_create to __tetra_net_epoll_create(1 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_epoll_ctl_add_read", 3, 1) != 1 {
		t.Fatalf("probe did not lower core.net_epoll_ctl_add_read to __tetra_net_epoll_ctl_add_read(3 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_epoll_ctl_add_read_write", 3, 1) != 1 {
		t.Fatalf("probe did not lower core.net_epoll_ctl_add_read_write to __tetra_net_epoll_ctl_add_read_write(3 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_epoll_ctl_mod_read", 3, 1) != 1 {
		t.Fatalf("probe did not lower core.net_epoll_ctl_mod_read to __tetra_net_epoll_ctl_mod_read(3 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_epoll_ctl_mod_read_write", 3, 1) != 1 {
		t.Fatalf("probe did not lower core.net_epoll_ctl_mod_read_write to __tetra_net_epoll_ctl_mod_read_write(3 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_epoll_ctl_delete", 3, 1) != 1 {
		t.Fatalf("probe did not lower core.net_epoll_ctl_delete to __tetra_net_epoll_ctl_delete(3 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_epoll_wait_one", 3, 1) != 1 {
		t.Fatalf("probe did not lower core.net_epoll_wait_one to __tetra_net_epoll_wait_one(3 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_epoll_wait_one_into", 5, 1) != 1 {
		t.Fatalf("probe did not lower core.net_epoll_wait_one_into to __tetra_net_epoll_wait_one_into(5 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_set_nonblocking", 2, 1) != 1 {
		t.Fatalf("probe did not lower core.net_set_nonblocking to __tetra_net_set_nonblocking(2 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_set_reuseport", 2, 1) != 1 {
		t.Fatalf("probe did not lower core.net_set_reuseport to __tetra_net_set_reuseport(2 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_set_tcp_nodelay", 2, 1) != 1 {
		t.Fatalf("probe did not lower core.net_set_tcp_nodelay to __tetra_net_set_tcp_nodelay(2 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_close", 2, 1) != 1 {
		t.Fatalf("probe did not lower core.net_close to __tetra_net_close(2 -> 1): %#v", probe.Instrs)
	}
}
