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
