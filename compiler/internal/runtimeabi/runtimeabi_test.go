package runtimeabi

import (
	"reflect"
	"testing"
)

func TestRequiredRuntimeSymbolSets(t *testing.T) {
	tests := []struct {
		name string
		got  []string
		want []string
	}{
		{
			name: "actors",
			got:  RequiredActorSymbols(),
			want: []string{
				"__tetra_entry",
				"__tetra_actor_spawn",
				"__tetra_actor_send",
				"__tetra_actor_send_msg",
				"__tetra_actor_send_begin",
				"__tetra_actor_send_slot",
				"__tetra_actor_send_commit",
				"__tetra_actor_recv",
				"__tetra_actor_recv_msg",
				"__tetra_actor_recv_poll",
				"__tetra_actor_recv_until",
				"__tetra_actor_recv_msg_until",
				"__tetra_actor_recv_begin",
				"__tetra_actor_recv_slot",
				"__tetra_actor_recv_count",
				"__tetra_actor_self",
				"__tetra_actor_sender",
				"__tetra_actor_yield_now",
			},
		},
		{
			name: "task",
			got:  RequiredTaskSymbols(),
			want: []string{
				"__tetra_task_spawn_i32",
				"__tetra_task_join_i32",
				"__tetra_task_join_result_i32",
				"__tetra_task_join_until_i32",
				"__tetra_task_poll_i32",
				"__tetra_task_is_canceled",
				"__tetra_task_checkpoint",
			},
		},
		{
			name: "typed_task_clamped",
			got:  RequiredTypedTaskSymbols(99),
			want: []string{
				"__tetra_task_result_begin",
				"__tetra_task_result_slot",
				"__tetra_task_result_get",
				"__tetra_task_join_typed_2",
				"__tetra_task_join_typed_3",
				"__tetra_task_join_typed_4",
				"__tetra_task_join_typed_5",
				"__tetra_task_join_typed_6",
				"__tetra_task_join_typed_7",
				"__tetra_task_join_typed_8",
			},
		},
		{
			name: "net",
			got:  RequiredNetSymbols(),
			want: []string{
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.got, tt.want) {
				t.Fatalf("symbols = %#v, want %#v", tt.got, tt.want)
			}
		})
	}
}

func TestSignatureForSymbol(t *testing.T) {
	tests := []struct {
		name   string
		params int
		rets   int
	}{
		{name: "__tetra_entry", params: 0, rets: 1},
		{name: "__tetra_actor_state_store", params: 2, rets: 1},
		{name: "__tetra_fs_exists", params: 3, rets: 1},
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
		{name: "__tetra_task_join_typed_4", params: 4, rets: 4},
		{name: "__tetra_task_join_typed_5", params: 5, rets: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := SignatureForSymbol(tt.name)
			if !ok {
				t.Fatalf("missing signature")
			}
			if got.ParamSlots != tt.params || got.ReturnSlots != tt.rets {
				t.Fatalf("signature = params=%d returns=%d, want params=%d returns=%d", got.ParamSlots, got.ReturnSlots, tt.params, tt.rets)
			}
		})
	}
}

func TestSignatureForSymbolRejectsUnknownTypedJoinArity(t *testing.T) {
	for _, name := range []string{"__tetra_task_join_typed_1", "__tetra_task_join_typed_9", "__tetra_task_join_typed_bad"} {
		if _, ok := SignatureForSymbol(name); ok {
			t.Fatalf("unexpected signature for %q", name)
		}
	}
}
