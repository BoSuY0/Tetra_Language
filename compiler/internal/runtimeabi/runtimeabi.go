package runtimeabi

import (
	"fmt"
	"strconv"
	"strings"
)

// Signature describes the slot-level ABI for one runtime object symbol.
type Signature struct {
	ParamSlots  int
	ReturnSlots int
}

func RequiredActorSymbols() []string {
	return []string{
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
	}
}

func RequiredActorStateSymbols() []string {
	return []string{
		"__tetra_actor_state_load",
		"__tetra_actor_state_store",
	}
}

func RequiredDistributedActorSymbols() []string {
	return []string{
		"__tetra_actor_node_connect",
		"__tetra_actor_spawn_remote",
		"__tetra_actor_node_status",
	}
}

func RequiredTaskSymbols() []string {
	return []string{
		"__tetra_task_spawn_i32",
		"__tetra_task_join_i32",
		"__tetra_task_join_result_i32",
		"__tetra_task_join_until_i32",
		"__tetra_task_poll_i32",
		"__tetra_task_is_canceled",
		"__tetra_task_checkpoint",
	}
}

func RequiredTaskGroupSymbols() []string {
	return []string{
		"__tetra_task_group_open",
		"__tetra_task_group_close",
		"__tetra_task_group_cancel",
		"__tetra_task_group_current",
		"__tetra_task_group_status",
		"__tetra_task_spawn_group_i32",
	}
}

func RequiredTypedTaskSymbols(maxSlots int) []string {
	if maxSlots < 2 {
		maxSlots = 2
	}
	if maxSlots > 8 {
		maxSlots = 8
	}
	symbols := []string{
		"__tetra_task_result_begin",
		"__tetra_task_result_slot",
	}
	if maxSlots > 4 {
		symbols = append(symbols, "__tetra_task_result_get")
	}
	for slots := 2; slots <= maxSlots; slots++ {
		symbols = append(symbols, fmt.Sprintf("__tetra_task_join_typed_%d", slots))
	}
	return symbols
}

func RequiredTimeSymbols() []string {
	return []string{
		"__tetra_time_now_ms",
		"__tetra_sleep_ms",
		"__tetra_sleep_until_ms",
		"__tetra_deadline_ms",
		"__tetra_timer_ready_ms",
	}
}

func RequiredFilesystemSymbols() []string {
	return []string{
		"__tetra_fs_exists",
	}
}

func SignatureForSymbol(name string) (Signature, bool) {
	switch name {
	case "__tetra_entry":
		return Signature{ParamSlots: 0, ReturnSlots: 1}, true
	case "__tetra_actor_spawn":
		return Signature{ParamSlots: 1, ReturnSlots: 1}, true
	case "__tetra_actor_send":
		return Signature{ParamSlots: 2, ReturnSlots: 1}, true
	case "__tetra_actor_send_msg":
		return Signature{ParamSlots: 3, ReturnSlots: 1}, true
	case "__tetra_actor_send_begin":
		return Signature{ParamSlots: 3, ReturnSlots: 1}, true
	case "__tetra_actor_send_slot":
		return Signature{ParamSlots: 2, ReturnSlots: 1}, true
	case "__tetra_actor_send_commit":
		return Signature{ParamSlots: 0, ReturnSlots: 1}, true
	case "__tetra_actor_recv":
		return Signature{ParamSlots: 0, ReturnSlots: 1}, true
	case "__tetra_actor_recv_msg":
		return Signature{ParamSlots: 0, ReturnSlots: 2}, true
	case "__tetra_actor_recv_poll":
		return Signature{ParamSlots: 0, ReturnSlots: 2}, true
	case "__tetra_actor_recv_until":
		return Signature{ParamSlots: 1, ReturnSlots: 2}, true
	case "__tetra_actor_recv_msg_until":
		return Signature{ParamSlots: 1, ReturnSlots: 3}, true
	case "__tetra_actor_recv_begin":
		return Signature{ParamSlots: 0, ReturnSlots: 1}, true
	case "__tetra_actor_recv_slot":
		return Signature{ParamSlots: 1, ReturnSlots: 1}, true
	case "__tetra_actor_recv_count":
		return Signature{ParamSlots: 0, ReturnSlots: 1}, true
	case "__tetra_actor_self":
		return Signature{ParamSlots: 0, ReturnSlots: 1}, true
	case "__tetra_actor_sender":
		return Signature{ParamSlots: 0, ReturnSlots: 1}, true
	case "__tetra_actor_yield_now":
		return Signature{ParamSlots: 0, ReturnSlots: 1}, true
	case "__tetra_actor_state_load":
		return Signature{ParamSlots: 1, ReturnSlots: 1}, true
	case "__tetra_actor_state_store":
		return Signature{ParamSlots: 2, ReturnSlots: 1}, true
	case "__tetra_actor_node_connect":
		return Signature{ParamSlots: 2, ReturnSlots: 1}, true
	case "__tetra_actor_spawn_remote":
		return Signature{ParamSlots: 2, ReturnSlots: 1}, true
	case "__tetra_actor_node_status":
		return Signature{ParamSlots: 1, ReturnSlots: 1}, true
	case "__tetra_task_spawn_i32":
		return Signature{ParamSlots: 1, ReturnSlots: 2}, true
	case "__tetra_task_join_i32":
		return Signature{ParamSlots: 2, ReturnSlots: 1}, true
	case "__tetra_task_join_result_i32":
		return Signature{ParamSlots: 2, ReturnSlots: 2}, true
	case "__tetra_task_join_until_i32":
		return Signature{ParamSlots: 3, ReturnSlots: 2}, true
	case "__tetra_task_poll_i32":
		return Signature{ParamSlots: 2, ReturnSlots: 2}, true
	case "__tetra_task_is_canceled":
		return Signature{ParamSlots: 0, ReturnSlots: 1}, true
	case "__tetra_task_checkpoint":
		return Signature{ParamSlots: 0, ReturnSlots: 1}, true
	case "__tetra_task_group_open":
		return Signature{ParamSlots: 0, ReturnSlots: 1}, true
	case "__tetra_task_group_close":
		return Signature{ParamSlots: 1, ReturnSlots: 1}, true
	case "__tetra_task_group_cancel":
		return Signature{ParamSlots: 1, ReturnSlots: 1}, true
	case "__tetra_task_group_current":
		return Signature{ParamSlots: 0, ReturnSlots: 1}, true
	case "__tetra_task_group_status":
		return Signature{ParamSlots: 1, ReturnSlots: 1}, true
	case "__tetra_task_spawn_group_i32":
		return Signature{ParamSlots: 2, ReturnSlots: 2}, true
	case "__tetra_task_result_begin":
		return Signature{ParamSlots: 1, ReturnSlots: 1}, true
	case "__tetra_task_result_slot":
		return Signature{ParamSlots: 2, ReturnSlots: 1}, true
	case "__tetra_task_result_get":
		return Signature{ParamSlots: 1, ReturnSlots: 1}, true
	case "__tetra_time_now_ms":
		return Signature{ParamSlots: 0, ReturnSlots: 1}, true
	case "__tetra_sleep_ms":
		return Signature{ParamSlots: 1, ReturnSlots: 1}, true
	case "__tetra_sleep_until_ms":
		return Signature{ParamSlots: 1, ReturnSlots: 1}, true
	case "__tetra_deadline_ms":
		return Signature{ParamSlots: 1, ReturnSlots: 1}, true
	case "__tetra_timer_ready_ms":
		return Signature{ParamSlots: 1, ReturnSlots: 1}, true
	case "__tetra_fs_exists":
		return Signature{ParamSlots: 3, ReturnSlots: 1}, true
	}

	const typedJoinPrefix = "__tetra_task_join_typed_"
	if strings.HasPrefix(name, typedJoinPrefix) {
		slots, err := strconv.Atoi(strings.TrimPrefix(name, typedJoinPrefix))
		if err != nil || slots < 2 || slots > 8 {
			return Signature{}, false
		}
		if slots > 4 {
			return Signature{ParamSlots: slots, ReturnSlots: 1}, true
		}
		return Signature{ParamSlots: slots, ReturnSlots: slots}, true
	}

	return Signature{}, false
}
