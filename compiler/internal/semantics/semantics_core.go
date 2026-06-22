package semantics

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
	"tetra_language/compiler/internal/runtimeabi"
	semanticsexpressions "tetra_language/compiler/internal/semantics/expressions"
	"tetra_language/compiler/internal/semantics/model"
	semanticsworld "tetra_language/compiler/internal/semantics/world"
)

// ---- builtins.go ----

func builtinFuncSigs(types map[string]*TypeInfo) (map[string]FuncSig, error) {
	_, err := ensureTypeInfo("[]u8", types)
	if err != nil {
		return nil, err
	}
	_, err = ensureTypeInfo("[]i32", types)
	if err != nil {
		return nil, err
	}
	_, err = ensureTypeInfo("[]u16", types)
	if err != nil {
		return nil, err
	}
	_, err = ensureTypeInfo("[]bool", types)
	if err != nil {
		return nil, err
	}
	strInfo, err := ensureTypeInfo("str", types)
	if err != nil {
		return nil, err
	}
	actorInfo, err := ensureTypeInfo("actor", types)
	if err != nil {
		return nil, err
	}
	actorMsgInfo, err := ensureTypeInfo("actor.msg", types)
	if err != nil {
		return nil, err
	}
	actorRecvResultI32, err := ensureTypeInfo("actor.recv_result_i32", types)
	if err != nil {
		return nil, err
	}
	actorRecvMsgResult, err := ensureTypeInfo("actor.recv_msg_result", types)
	if err != nil {
		return nil, err
	}
	actorStatusInfo, err := ensureTypeInfo("actor.status", types)
	if err != nil {
		return nil, err
	}
	actorStatusRawInfo, err := ensureTypeInfo("actor.status_result_raw", types)
	if err != nil {
		return nil, err
	}
	actorExitReasonInfo, err := ensureTypeInfo("actor.exit_reason", types)
	if err != nil {
		return nil, err
	}
	actorWaitResultInfo, err := ensureTypeInfo("actor.wait_result", types)
	if err != nil {
		return nil, err
	}
	actorMonitorInfo, err := ensureTypeInfo("actor.monitor", types)
	if err != nil {
		return nil, err
	}
	actorSystemRecvRawInfo, err := ensureTypeInfo("actor.system_recv_raw", types)
	if err != nil {
		return nil, err
	}
	ptrInfo, err := ensureTypeInfo("ptr", types)
	if err != nil {
		return nil, err
	}
	taskGroupInfo, err := ensureTypeInfo("task.group", types)
	if err != nil {
		return nil, err
	}
	taskErrorInfo, err := ensureTypeInfo("task.error", types)
	if err != nil {
		return nil, err
	}
	taskHandleI32, err := ensureTypeInfo("task.i32", types)
	if err != nil {
		return nil, err
	}
	taskResultI32, err := ensureTypeInfo("task.result_i32", types)
	if err != nil {
		return nil, err
	}
	sliceU8, err := ensureTypeInfo("[]u8", types)
	if err != nil {
		return nil, err
	}
	sliceI32, err := ensureTypeInfo("[]i32", types)
	if err != nil {
		return nil, err
	}
	sliceU16, err := ensureTypeInfo("[]u16", types)
	if err != nil {
		return nil, err
	}
	sliceBool, err := ensureTypeInfo("[]bool", types)
	if err != nil {
		return nil, err
	}

	islandInfo, err := ensureTypeInfo("island", types)
	if err != nil {
		return nil, err
	}
	capIO, err := ensureTypeInfo("cap.io", types)
	if err != nil {
		return nil, err
	}
	capMem, err := ensureTypeInfo("cap.mem", types)
	if err != nil {
		return nil, err
	}
	consentToken, err := ensureTypeInfo("consent.token", types)
	if err != nil {
		return nil, err
	}
	secretI32, err := ensureTypeInfo("secret.i32", types)
	if err != nil {
		return nil, err
	}

	sigs := map[string]FuncSig{
		"core.alloc_bytes": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        "ptr",
			ReturnSlots:       ptrInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.make_u8": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        sliceU8.Name,
			ReturnSlots:       sliceU8.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.make_u16": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        sliceU16.Name,
			ReturnSlots:       sliceU16.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.make_i32": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        sliceI32.Name,
			ReturnSlots:       sliceI32.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.make_bool": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        sliceBool.Name,
			ReturnSlots:       sliceBool.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.raw_slice_u8_from_parts": {
			ParamTypes:        []string{"ptr", "i32", capMem.Name},
			ParamSlots:        3,
			ReturnType:        sliceU8.Name,
			ReturnSlots:       sliceU8.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.raw_slice_u16_from_parts": {
			ParamTypes:        []string{"ptr", "i32", capMem.Name},
			ParamSlots:        3,
			ReturnType:        sliceU16.Name,
			ReturnSlots:       sliceU16.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.raw_slice_i32_from_parts": {
			ParamTypes:        []string{"ptr", "i32", capMem.Name},
			ParamSlots:        3,
			ReturnType:        sliceI32.Name,
			ReturnSlots:       sliceI32.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.raw_slice_bool_from_parts": {
			ParamTypes:        []string{"ptr", "i32", capMem.Name},
			ParamSlots:        3,
			ReturnType:        sliceBool.Name,
			ReturnSlots:       sliceBool.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.slice_window_u8": {
			ParamTypes:        []string{sliceU8.Name, "i32", "i32"},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        sliceU8.SlotCount + 2,
			ReturnType:        sliceU8.Name,
			ReturnSlots:       sliceU8.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.slice_window_u16": {
			ParamTypes:        []string{sliceU16.Name, "i32", "i32"},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        sliceU16.SlotCount + 2,
			ReturnType:        sliceU16.Name,
			ReturnSlots:       sliceU16.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.slice_window_i32": {
			ParamTypes:        []string{sliceI32.Name, "i32", "i32"},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        sliceI32.SlotCount + 2,
			ReturnType:        sliceI32.Name,
			ReturnSlots:       sliceI32.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.slice_window_bool": {
			ParamTypes:        []string{sliceBool.Name, "i32", "i32"},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        sliceBool.SlotCount + 2,
			ReturnType:        sliceBool.Name,
			ReturnSlots:       sliceBool.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.slice_prefix_u8": {
			ParamTypes:        []string{sliceU8.Name, "i32"},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        sliceU8.SlotCount + 1,
			ReturnType:        sliceU8.Name,
			ReturnSlots:       sliceU8.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.slice_prefix_u16": {
			ParamTypes:        []string{sliceU16.Name, "i32"},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        sliceU16.SlotCount + 1,
			ReturnType:        sliceU16.Name,
			ReturnSlots:       sliceU16.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.slice_prefix_i32": {
			ParamTypes:        []string{sliceI32.Name, "i32"},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        sliceI32.SlotCount + 1,
			ReturnType:        sliceI32.Name,
			ReturnSlots:       sliceI32.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.slice_prefix_bool": {
			ParamTypes:        []string{sliceBool.Name, "i32"},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        sliceBool.SlotCount + 1,
			ReturnType:        sliceBool.Name,
			ReturnSlots:       sliceBool.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.slice_suffix_u8": {
			ParamTypes:        []string{sliceU8.Name, "i32"},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        sliceU8.SlotCount + 1,
			ReturnType:        sliceU8.Name,
			ReturnSlots:       sliceU8.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.slice_suffix_u16": {
			ParamTypes:        []string{sliceU16.Name, "i32"},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        sliceU16.SlotCount + 1,
			ReturnType:        sliceU16.Name,
			ReturnSlots:       sliceU16.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.slice_suffix_i32": {
			ParamTypes:        []string{sliceI32.Name, "i32"},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        sliceI32.SlotCount + 1,
			ReturnType:        sliceI32.Name,
			ReturnSlots:       sliceI32.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.slice_suffix_bool": {
			ParamTypes:        []string{sliceBool.Name, "i32"},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        sliceBool.SlotCount + 1,
			ReturnType:        sliceBool.Name,
			ReturnSlots:       sliceBool.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.slice_borrow_u8": {
			ParamTypes:        []string{sliceU8.Name},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        sliceU8.SlotCount,
			ReturnType:        sliceU8.Name,
			ReturnSlots:       sliceU8.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.slice_borrow_u16": {
			ParamTypes:        []string{sliceU16.Name},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        sliceU16.SlotCount,
			ReturnType:        sliceU16.Name,
			ReturnSlots:       sliceU16.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.slice_borrow_i32": {
			ParamTypes:        []string{sliceI32.Name},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        sliceI32.SlotCount,
			ReturnType:        sliceI32.Name,
			ReturnSlots:       sliceI32.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.slice_borrow_bool": {
			ParamTypes:        []string{sliceBool.Name},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        sliceBool.SlotCount,
			ReturnType:        sliceBool.Name,
			ReturnSlots:       sliceBool.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.slice_copy_u8": {
			ParamTypes:        []string{sliceU8.Name},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        sliceU8.SlotCount,
			ReturnType:        sliceU8.Name,
			ReturnSlots:       sliceU8.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.slice_copy_u16": {
			ParamTypes:        []string{sliceU16.Name},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        sliceU16.SlotCount,
			ReturnType:        sliceU16.Name,
			ReturnSlots:       sliceU16.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.slice_copy_i32": {
			ParamTypes:        []string{sliceI32.Name},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        sliceI32.SlotCount,
			ReturnType:        sliceI32.Name,
			ReturnSlots:       sliceI32.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.slice_copy_bool": {
			ParamTypes:        []string{sliceBool.Name},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        sliceBool.SlotCount,
			ReturnType:        sliceBool.Name,
			ReturnSlots:       sliceBool.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.slice_copy_into_u8": {
			ParamTypes:        []string{sliceU8.Name, sliceU8.Name},
			ParamOwnership:    []string{"borrow", "inout"},
			ParamSlots:        sliceU8.SlotCount * 2,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.slice_copy_into_u16": {
			ParamTypes:        []string{sliceU16.Name, sliceU16.Name},
			ParamOwnership:    []string{"borrow", "inout"},
			ParamSlots:        sliceU16.SlotCount * 2,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.slice_copy_into_i32": {
			ParamTypes:        []string{sliceI32.Name, sliceI32.Name},
			ParamOwnership:    []string{"borrow", "inout"},
			ParamSlots:        sliceI32.SlotCount * 2,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.slice_copy_into_bool": {
			ParamTypes:        []string{sliceBool.Name, sliceBool.Name},
			ParamOwnership:    []string{"borrow", "inout"},
			ParamSlots:        sliceBool.SlotCount * 2,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.string_window": {
			ParamTypes:        []string{strInfo.Name, "i32", "i32"},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        strInfo.SlotCount + 2,
			ReturnType:        strInfo.Name,
			ReturnSlots:       strInfo.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.string_prefix": {
			ParamTypes:        []string{strInfo.Name, "i32"},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        strInfo.SlotCount + 1,
			ReturnType:        strInfo.Name,
			ReturnSlots:       strInfo.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.string_suffix": {
			ParamTypes:        []string{strInfo.Name, "i32"},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        strInfo.SlotCount + 1,
			ReturnType:        strInfo.Name,
			ReturnSlots:       strInfo.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.string_borrow": {
			ParamTypes:        []string{strInfo.Name},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        strInfo.SlotCount,
			ReturnType:        strInfo.Name,
			ReturnSlots:       strInfo.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.string_copy": {
			ParamTypes:        []string{strInfo.Name},
			ParamOwnership:    []string{"borrow"},
			ParamSlots:        strInfo.SlotCount,
			ReturnType:        strInfo.Name,
			ReturnSlots:       strInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.string_copy_into": {
			ParamTypes:        []string{strInfo.Name, sliceU8.Name},
			ParamOwnership:    []string{"borrow", "inout"},
			ParamSlots:        strInfo.SlotCount + sliceU8.SlotCount,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.island_new": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        "island",
			ReturnSlots:       islandInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.island_make_u8": {
			ParamTypes:        []string{"island", "i32"},
			ParamSlots:        2,
			ReturnType:        sliceU8.Name,
			ReturnSlots:       sliceU8.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.island_make_u16": {
			ParamTypes:        []string{"island", "i32"},
			ParamSlots:        2,
			ReturnType:        sliceU16.Name,
			ReturnSlots:       sliceU16.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.island_make_i32": {
			ParamTypes:        []string{"island", "i32"},
			ParamSlots:        2,
			ReturnType:        sliceI32.Name,
			ReturnSlots:       sliceI32.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.island_make_bool": {
			ParamTypes:        []string{"island", "i32"},
			ParamSlots:        2,
			ReturnType:        sliceBool.Name,
			ReturnSlots:       sliceBool.SlotCount,
			ReturnRegionParam: 0,
		},
		"core.island_reset": {
			ParamTypes:        []string{"island"},
			ParamOwnership:    []string{"consume"},
			ParamSlots:        1,
			ReturnType:        "island",
			ReturnSlots:       islandInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.cap_io": {
			ParamTypes:        nil,
			ParamSlots:        0,
			ReturnType:        capIO.Name,
			ReturnSlots:       capIO.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.cap_mem": {
			ParamTypes:        nil,
			ParamSlots:        0,
			ReturnType:        capMem.Name,
			ReturnSlots:       capMem.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.load_i32": {
			ParamTypes:        []string{"ptr", capMem.Name},
			ParamSlots:        2,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.store_i32": {
			ParamTypes:        []string{"ptr", "i32", capMem.Name},
			ParamSlots:        3,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.load_u8": {
			ParamTypes:        []string{"ptr", capMem.Name},
			ParamSlots:        2,
			ReturnType:        "u8",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.store_u8": {
			ParamTypes:        []string{"ptr", "u8", capMem.Name},
			ParamSlots:        3,
			ReturnType:        "u8",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.load_ptr": {
			ParamTypes:        []string{"ptr", capMem.Name},
			ParamSlots:        2,
			ReturnType:        "ptr",
			ReturnSlots:       ptrInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.store_ptr": {
			ParamTypes:        []string{"ptr", "ptr", capMem.Name},
			ParamSlots:        3,
			ReturnType:        "ptr",
			ReturnSlots:       ptrInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.store_arch_ptr": {
			ParamTypes:        []string{"ptr", "ptr", capMem.Name},
			ParamSlots:        3,
			ReturnType:        "ptr",
			ReturnSlots:       ptrInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.ptr_add": {
			ParamTypes:        []string{"ptr", "i32", capMem.Name},
			ParamSlots:        3,
			ReturnType:        "ptr",
			ReturnSlots:       ptrInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.mmio_read_i32": {
			ParamTypes:        []string{"ptr", capIO.Name},
			ParamSlots:        2,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.mmio_write_i32": {
			ParamTypes:        []string{"ptr", "i32", capIO.Name},
			ParamSlots:        3,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.fs_exists": {
			ParamTypes:        []string{"str", capIO.Name},
			ParamSlots:        3,
			ReturnType:        "bool",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.net_socket_tcp4": {
			ParamTypes:        []string{capIO.Name},
			ParamSlots:        1,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.net_bind_tcp4_loopback": {
			ParamTypes:        []string{"i32", "i32", capIO.Name},
			ParamSlots:        3,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.net_connect_tcp4_loopback": {
			ParamTypes:        []string{"i32", "i32", capIO.Name},
			ParamSlots:        3,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.net_listen": {
			ParamTypes:        []string{"i32", "i32", capIO.Name},
			ParamSlots:        3,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.net_accept4": {
			ParamTypes:        []string{"i32", "i32", capIO.Name},
			ParamSlots:        3,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.net_read": {
			ParamTypes:        []string{"i32", sliceU8.Name, "i32", "i32", capIO.Name},
			ParamSlots:        6,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.net_recv": {
			ParamTypes:        []string{"i32", sliceU8.Name, "i32", "i32", capIO.Name},
			ParamSlots:        6,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.net_write": {
			ParamTypes:        []string{"i32", sliceU8.Name, "i32", "i32", capIO.Name},
			ParamSlots:        6,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.net_send": {
			ParamTypes:        []string{"i32", sliceU8.Name, "i32", "i32", capIO.Name},
			ParamSlots:        6,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.net_epoll_create": {
			ParamTypes:        []string{capIO.Name},
			ParamSlots:        1,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.net_epoll_ctl_add_read": {
			ParamTypes:        []string{"i32", "i32", capIO.Name},
			ParamSlots:        3,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.net_epoll_ctl_add_read_write": {
			ParamTypes:        []string{"i32", "i32", capIO.Name},
			ParamSlots:        3,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.net_epoll_ctl_mod_read": {
			ParamTypes:        []string{"i32", "i32", capIO.Name},
			ParamSlots:        3,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.net_epoll_ctl_mod_read_write": {
			ParamTypes:        []string{"i32", "i32", capIO.Name},
			ParamSlots:        3,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.net_epoll_ctl_delete": {
			ParamTypes:        []string{"i32", "i32", capIO.Name},
			ParamSlots:        3,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.net_epoll_wait_one": {
			ParamTypes:        []string{"i32", "i32", capIO.Name},
			ParamSlots:        3,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.net_epoll_wait_one_into": {
			ParamTypes:        []string{"i32", sliceI32.Name, "i32", capIO.Name},
			ParamSlots:        5,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.net_set_nonblocking": {
			ParamTypes:        []string{"i32", capIO.Name},
			ParamSlots:        2,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.net_set_reuseport": {
			ParamTypes:        []string{"i32", capIO.Name},
			ParamSlots:        2,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.net_set_tcp_nodelay": {
			ParamTypes:        []string{"i32", capIO.Name},
			ParamSlots:        2,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.net_close": {
			ParamTypes:        []string{"i32", capIO.Name},
			ParamSlots:        2,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.sym_addr": {
			ParamTypes:        []string{"str"},
			ParamSlots:        2,
			ReturnType:        "ptr",
			ReturnSlots:       ptrInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.ctx_switch": {
			ParamTypes:        []string{"ptr", "ptr", capMem.Name},
			ParamSlots:        3,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.surface_open": {
			ParamTypes:        []string{"str", "i32", "i32"},
			ParamSlots:        4,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.surface_close": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.surface_poll_event_kind": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.surface_poll_event_x": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.surface_poll_event_y": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.surface_poll_event_button": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.surface_poll_event_into": {
			ParamTypes:        []string{"i32", sliceI32.Name},
			ParamSlots:        1 + sliceI32.SlotCount,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.surface_poll_event_text_len": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.surface_poll_event_text_into": {
			ParamTypes:        []string{"i32", sliceU8.Name},
			ParamSlots:        1 + sliceU8.SlotCount,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.surface_clipboard_write_text": {
			ParamTypes:        []string{"i32", sliceU8.Name},
			ParamSlots:        1 + sliceU8.SlotCount,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.surface_clipboard_read_text_into": {
			ParamTypes:        []string{"i32", sliceU8.Name},
			ParamSlots:        1 + sliceU8.SlotCount,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.surface_poll_composition_into": {
			ParamTypes:        []string{"i32", sliceI32.Name},
			ParamSlots:        1 + sliceI32.SlotCount,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.surface_begin_frame": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.surface_present_rgba": {
			ParamTypes:        []string{"i32", sliceU8.Name, "i32", "i32", "i32"},
			ParamSlots:        1 + sliceU8.SlotCount + 3,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.surface_now_ms": {
			ParamTypes:        nil,
			ParamSlots:        0,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.surface_request_redraw": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.time_now_ms": {
			ParamTypes:        nil,
			ParamSlots:        0,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.sleep_ms": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.sleep_until": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.deadline_ms": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.timer_ready": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        "bool",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.yield": {
			ParamTypes:        nil,
			ParamSlots:        0,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.task_group_open": {
			ParamTypes:        nil,
			ParamSlots:        0,
			ReturnType:        taskGroupInfo.Name,
			ReturnSlots:       taskGroupInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.task_group_close": {
			ParamTypes:        []string{"task.group"},
			ParamSlots:        taskGroupInfo.SlotCount,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.task_group_cancel": {
			ParamTypes:        []string{"task.group"},
			ParamSlots:        taskGroupInfo.SlotCount,
			ReturnType:        "task.group",
			ReturnSlots:       taskGroupInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.task_group_current": {
			ParamTypes:        nil,
			ParamSlots:        0,
			ReturnType:        taskGroupInfo.Name,
			ReturnSlots:       taskGroupInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.task_group_status": {
			ParamTypes:        []string{"task.group"},
			ParamSlots:        taskGroupInfo.SlotCount,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.task_is_canceled": {
			ParamTypes:        nil,
			ParamSlots:        0,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.task_checkpoint": {
			ParamTypes:        nil,
			ParamSlots:        0,
			ReturnType:        taskErrorInfo.Name,
			ReturnSlots:       taskErrorInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.task_spawn_i32": {
			ParamTypes:        []string{"str"},
			ParamSlots:        2,
			ReturnType:        taskHandleI32.Name,
			ReturnSlots:       taskHandleI32.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.task_spawn_i32_typed": {
			ParamTypes:        []string{"str"},
			ParamSlots:        2,
			ReturnType:        taskHandleI32.Name,
			ReturnSlots:       taskHandleI32.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.task_spawn_group_i32": {
			ParamTypes:        []string{"task.group", "str"},
			ParamSlots:        taskGroupInfo.SlotCount + 2,
			ReturnType:        taskHandleI32.Name,
			ReturnSlots:       taskHandleI32.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.task_spawn_group_i32_typed": {
			ParamTypes:        []string{"task.group", "str"},
			ParamSlots:        taskGroupInfo.SlotCount + 2,
			ReturnType:        taskHandleI32.Name,
			ReturnSlots:       taskHandleI32.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.task_join_i32": {
			ParamTypes:        []string{"task.i32"},
			ParamSlots:        taskHandleI32.SlotCount,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.task_join_i32_typed": {
			ParamTypes:        []string{"task.i32"},
			ParamSlots:        taskHandleI32.SlotCount,
			ReturnType:        "i32",
			ThrowsType:        "enum",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.task_join_group_i32_typed": {
			ParamTypes:        []string{"task.i32"},
			ParamSlots:        taskHandleI32.SlotCount,
			ReturnType:        "i32",
			ThrowsType:        "enum",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.task_join_result_i32": {
			ParamTypes:        []string{"task.i32"},
			ParamSlots:        taskHandleI32.SlotCount,
			ReturnType:        taskResultI32.Name,
			ReturnSlots:       taskResultI32.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.task_join_until_i32": {
			ParamTypes:        []string{"task.i32", "i32"},
			ParamSlots:        taskHandleI32.SlotCount + 1,
			ReturnType:        taskResultI32.Name,
			ReturnSlots:       taskResultI32.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.task_poll_i32": {
			ParamTypes:        []string{"task.i32"},
			ParamSlots:        taskHandleI32.SlotCount,
			ReturnType:        taskResultI32.Name,
			ReturnSlots:       taskResultI32.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.select2_i32": {
			ParamTypes:        []string{"task.i32", "i32"},
			ParamSlots:        taskHandleI32.SlotCount + 1,
			ReturnType:        taskResultI32.Name,
			ReturnSlots:       taskResultI32.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.actor_dispatch": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.actor_main_entry_id": {
			ParamTypes:        nil,
			ParamSlots:        0,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.actor_node_connect": {
			ParamTypes:        []string{"i32", "i32"},
			ParamSlots:        2,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.actor_node_status": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.actor_ref_local": {
			ParamTypes:        []string{"i32", "i32"},
			ParamSlots:        2,
			ReturnType:        actorInfo.Name,
			ReturnSlots:       actorInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.actor_ref_slot": {
			ParamTypes:        []string{"actor"},
			ParamSlots:        actorInfo.SlotCount,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.spawn": {
			ParamTypes:        []string{"str"},
			ParamSlots:        2,
			ReturnType:        actorInfo.Name,
			ReturnSlots:       actorInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.spawn_remote": {
			ParamTypes:        []string{"i32", "str"},
			ParamSlots:        3,
			ReturnType:        actorInfo.Name,
			ReturnSlots:       actorInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.send": {
			ParamTypes:        []string{"actor", "i32"},
			ParamSlots:        actorInfo.SlotCount + 1,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.send_msg": {
			ParamTypes:        []string{"actor", "i32", "i32"},
			ParamSlots:        actorInfo.SlotCount + 2,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.send_typed": {
			ParamTypes:        []string{"actor", "enum"},
			ParamSlots:        actorInfo.SlotCount + 1,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.recv": {
			ParamTypes:        nil,
			ParamSlots:        0,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.recv_msg": {
			ParamTypes:        nil,
			ParamSlots:        0,
			ReturnType:        actorMsgInfo.Name,
			ReturnSlots:       actorMsgInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.recv_poll": {
			ParamTypes:        nil,
			ParamSlots:        0,
			ReturnType:        actorRecvResultI32.Name,
			ReturnSlots:       actorRecvResultI32.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.recv_until": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        actorRecvResultI32.Name,
			ReturnSlots:       actorRecvResultI32.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.recv_msg_until": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        actorRecvMsgResult.Name,
			ReturnSlots:       actorRecvMsgResult.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.recv_typed": {
			ParamTypes:        nil,
			ParamSlots:        0,
			ReturnType:        "enum",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.self": {
			ParamTypes:        nil,
			ParamSlots:        0,
			ReturnType:        actorInfo.Name,
			ReturnSlots:       actorInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.sender": {
			ParamTypes:        nil,
			ParamSlots:        0,
			ReturnType:        actorInfo.Name,
			ReturnSlots:       actorInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.actor_status": {
			ParamTypes:        []string{"actor"},
			ParamSlots:        actorInfo.SlotCount,
			ReturnType:        actorStatusInfo.Name,
			ReturnSlots:       actorStatusInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.actor_status_raw": {
			ParamTypes:        []string{"actor"},
			ParamSlots:        actorInfo.SlotCount,
			ReturnType:        actorStatusRawInfo.Name,
			ReturnSlots:       actorStatusRawInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.actor_wait": {
			ParamTypes:        []string{"actor"},
			ParamSlots:        actorInfo.SlotCount,
			ReturnType:        actorWaitResultInfo.Name,
			ReturnSlots:       actorWaitResultInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.actor_wait_until": {
			ParamTypes:        []string{"actor", "i32"},
			ParamSlots:        actorInfo.SlotCount + 1,
			ReturnType:        actorWaitResultInfo.Name,
			ReturnSlots:       actorWaitResultInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.actor_stop": {
			ParamTypes:        []string{"actor", actorExitReasonInfo.Name},
			ParamSlots:        actorInfo.SlotCount + actorExitReasonInfo.SlotCount,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.actor_exit_reason": {
			ParamTypes:        []string{"actor"},
			ParamSlots:        actorInfo.SlotCount,
			ReturnType:        actorExitReasonInfo.Name,
			ReturnSlots:       actorExitReasonInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.actor_link": {
			ParamTypes:        []string{"actor"},
			ParamSlots:        actorInfo.SlotCount,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.actor_unlink": {
			ParamTypes:        []string{"actor"},
			ParamSlots:        actorInfo.SlotCount,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.actor_monitor": {
			ParamTypes:        []string{"actor"},
			ParamSlots:        actorInfo.SlotCount,
			ReturnType:        actorMonitorInfo.Name,
			ReturnSlots:       actorMonitorInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.actor_demonitor": {
			ParamTypes:        []string{actorMonitorInfo.Name},
			ParamSlots:        actorMonitorInfo.SlotCount,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.actor_set_trap_exit": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
		"core.actor_recv_system": {
			ParamTypes:        nil,
			ParamSlots:        0,
			ReturnType:        actorSystemRecvRawInfo.Name,
			ReturnSlots:       actorSystemRecvRawInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.actor_recv_system_poll": {
			ParamTypes:        nil,
			ParamSlots:        0,
			ReturnType:        actorSystemRecvRawInfo.Name,
			ReturnSlots:       actorSystemRecvRawInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.actor_recv_system_until": {
			ParamTypes:        []string{"i32"},
			ParamSlots:        1,
			ReturnType:        actorSystemRecvRawInfo.Name,
			ReturnSlots:       actorSystemRecvRawInfo.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.consent_token": {
			ParamTypes:        nil,
			ParamSlots:        0,
			ReturnType:        consentToken.Name,
			ReturnSlots:       consentToken.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.secret_seal_i32": {
			ParamTypes:        []string{"i32", consentToken.Name},
			ParamSlots:        2,
			ReturnType:        secretI32.Name,
			ReturnSlots:       secretI32.SlotCount,
			ReturnRegionParam: regionNone,
		},
		"core.secret_unseal_i32": {
			ParamTypes:        []string{secretI32.Name, consentToken.Name},
			ParamSlots:        2,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		},
	}
	addAtomicBuiltinSigs(sigs, capMem.Name)
	for name, sig := range sigs {
		sig.ReturnResourceParam = regionNone
		if name == "core.task_group_cancel" {
			sig.ReturnResourceParam = 0
		}
		sig.Effects = builtinEffects(name)
		sigs[name] = sig
	}
	return sigs, nil
}

type atomicBuiltinValueType struct {
	Suffix   string
	TypeName string
}

var atomicBuiltinValueTypes = []atomicBuiltinValueType{
	{Suffix: "u8", TypeName: "u8"},
	{Suffix: "u16", TypeName: "u16"},
	{Suffix: "i32", TypeName: "i32"},
	{Suffix: "i64", TypeName: "i64"},
	{Suffix: "ptr", TypeName: "ptr"},
}

var atomicBuiltinLoadOrders = []string{"relaxed", "acquire", "seq_cst"}
var atomicBuiltinStoreOrders = []string{"relaxed", "release", "seq_cst"}

var atomicBuiltinReadModifyWriteOrders = []string{
	"relaxed",
	"acquire",
	"release",
	"acq_rel",
	"seq_cst",
}
var atomicBuiltinFenceOrders = []string{"relaxed", "acquire", "release", "acq_rel", "seq_cst"}

func addAtomicBuiltinSigs(sigs map[string]FuncSig, capMem string) {
	for _, valueType := range atomicBuiltinValueTypes {
		for _, order := range atomicBuiltinLoadOrders {
			name := "core.atomic_load_" + valueType.Suffix + "_" + order
			sigs[name] = FuncSig{
				ParamTypes:        []string{"ptr", capMem},
				ParamSlots:        2,
				ReturnType:        valueType.TypeName,
				ReturnSlots:       1,
				ReturnRegionParam: regionNone,
			}
		}
		for _, order := range atomicBuiltinStoreOrders {
			name := "core.atomic_store_" + valueType.Suffix + "_" + order
			sigs[name] = FuncSig{
				ParamTypes:        []string{"ptr", valueType.TypeName, capMem},
				ParamSlots:        3,
				ReturnType:        valueType.TypeName,
				ReturnSlots:       1,
				ReturnRegionParam: regionNone,
			}
		}
		for _, op := range []string{
			"exchange",
			"fetch_add",
			"fetch_sub",
			"fetch_and",
			"fetch_or",
			"fetch_xor",
		} {
			for _, order := range atomicBuiltinReadModifyWriteOrders {
				name := "core.atomic_" + op + "_" + valueType.Suffix + "_" + order
				sigs[name] = FuncSig{
					ParamTypes:        []string{"ptr", valueType.TypeName, capMem},
					ParamSlots:        3,
					ReturnType:        valueType.TypeName,
					ReturnSlots:       1,
					ReturnRegionParam: regionNone,
				}
			}
		}
		for _, order := range atomicBuiltinReadModifyWriteOrders {
			name := "core.atomic_compare_exchange_" + valueType.Suffix + "_" + order
			sigs[name] = FuncSig{
				ParamTypes:        []string{"ptr", valueType.TypeName, valueType.TypeName, capMem},
				ParamSlots:        4,
				ReturnType:        valueType.TypeName,
				ReturnSlots:       1,
				ReturnRegionParam: regionNone,
			}
		}
		for _, order := range atomicBuiltinReadModifyWriteOrders {
			name := "core.atomic_compare_exchange_weak_" + valueType.Suffix + "_" + order
			sigs[name] = FuncSig{
				ParamTypes:        []string{"ptr", valueType.TypeName, valueType.TypeName, capMem},
				ParamSlots:        4,
				ReturnType:        valueType.TypeName,
				ReturnSlots:       1,
				ReturnRegionParam: regionNone,
			}
		}
	}
	for _, order := range atomicBuiltinFenceOrders {
		name := "core.atomic_fence_" + order
		sigs[name] = FuncSig{
			ParamTypes:        []string{capMem},
			ParamSlots:        1,
			ReturnType:        "i32",
			ReturnSlots:       1,
			ReturnRegionParam: regionNone,
		}
	}
}

func isCoreAtomicBuiltin(name string) bool {
	return strings.HasPrefix(name, "core.atomic_")
}

func atomicBuiltinDiagnostic(name string) (string, bool) {
	const prefix = "core.atomic_"
	if !strings.HasPrefix(name, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(name, prefix)
	if strings.HasPrefix(rest, "fence_") {
		order := strings.TrimPrefix(rest, "fence_")
		if !atomicBuiltinOrderKnown(order) {
			return "unsupported atomic memory order '" + order + "'", true
		}
		return "", false
	}
	for _, op := range atomicBuiltinOpPrefixes() {
		if !strings.HasPrefix(rest, op.Prefix) {
			continue
		}
		tail := strings.TrimPrefix(rest, op.Prefix)
		width, order, ok := splitAtomicBuiltinWidthOrder(tail)
		if !ok {
			width, order, ok = splitAtomicBuiltinKnownWidthUnknownOrder(tail)
			if ok {
				return "unsupported atomic memory order '" + order + "'", true
			}
			return "unsupported atomic value width '" + atomicBuiltinDiagnosticWidth(
				tail,
			) + "'", true
		}
		if !atomicBuiltinWidthKnown(width) {
			return "unsupported atomic value width '" + width + "'", true
		}
		if !atomicBuiltinOrderAllowed(op.Name, order) {
			return "atomic " + op.Display + " does not support memory order " + order, true
		}
		return "", false
	}
	if op, ok := atomicBuiltinUnknownOp(rest); ok {
		return "unsupported atomic operation '" + op + "'", true
	}
	return "unsupported atomic builtin '" + name + "'", true
}

type atomicBuiltinOpPrefix struct {
	Prefix  string
	Name    string
	Display string
}

func atomicBuiltinOpPrefixes() []atomicBuiltinOpPrefix {
	return []atomicBuiltinOpPrefix{
		{
			Prefix:  "compare_exchange_weak_",
			Name:    "compare_exchange_weak",
			Display: "compare_exchange_weak",
		},
		{Prefix: "compare_exchange_", Name: "compare_exchange", Display: "compare_exchange"},
		{Prefix: "fetch_add_", Name: "fetch_add", Display: "fetch_add"},
		{Prefix: "fetch_sub_", Name: "fetch_sub", Display: "fetch_sub"},
		{Prefix: "fetch_and_", Name: "fetch_and", Display: "fetch_and"},
		{Prefix: "fetch_or_", Name: "fetch_or", Display: "fetch_or"},
		{Prefix: "fetch_xor_", Name: "fetch_xor", Display: "fetch_xor"},
		{Prefix: "exchange_", Name: "exchange", Display: "exchange"},
		{Prefix: "store_", Name: "store", Display: "store"},
		{Prefix: "load_", Name: "load", Display: "load"},
	}
}

func splitAtomicBuiltinWidthOrder(tail string) (width string, order string, ok bool) {
	for _, candidate := range atomicBuiltinAllOrders() {
		suffix := "_" + candidate
		if strings.HasSuffix(tail, suffix) {
			return strings.TrimSuffix(tail, suffix), candidate, true
		}
	}
	return "", "", false
}

func splitAtomicBuiltinKnownWidthUnknownOrder(tail string) (width string, order string, ok bool) {
	for _, valueType := range atomicBuiltinValueTypes {
		prefix := valueType.Suffix + "_"
		if strings.HasPrefix(tail, prefix) {
			return valueType.Suffix, strings.TrimPrefix(tail, prefix), true
		}
	}
	return "", "", false
}

func atomicBuiltinDiagnosticWidth(tail string) string {
	if before, _, ok := strings.Cut(tail, "_"); ok {
		return before
	}
	return tail
}

func atomicBuiltinUnknownOp(rest string) (string, bool) {
	for _, order := range atomicBuiltinAllOrders() {
		beforeOrder := strings.TrimSuffix(rest, "_"+order)
		if beforeOrder == rest {
			continue
		}
		for _, valueType := range atomicBuiltinValueTypes {
			widthSuffix := "_" + valueType.Suffix
			if strings.HasSuffix(beforeOrder, widthSuffix) {
				op := strings.TrimSuffix(beforeOrder, widthSuffix)
				if op != "" {
					return op, true
				}
			}
		}
	}
	return "", false
}

func atomicBuiltinAllOrders() []string {
	return []string{"relaxed", "acquire", "release", "acq_rel", "seq_cst"}
}

func atomicBuiltinWidthKnown(width string) bool {
	for _, valueType := range atomicBuiltinValueTypes {
		if width == valueType.Suffix {
			return true
		}
	}
	return false
}

func atomicBuiltinOrderKnown(order string) bool {
	for _, candidate := range atomicBuiltinAllOrders() {
		if order == candidate {
			return true
		}
	}
	return false
}

func atomicBuiltinOrderAllowed(op string, order string) bool {
	switch op {
	case "load":
		return order == "relaxed" || order == "acquire" || order == "seq_cst"
	case "store":
		return order == "relaxed" || order == "release" || order == "seq_cst"
	case "exchange", "compare_exchange", "compare_exchange_weak",
		"fetch_add", "fetch_sub", "fetch_and", "fetch_or", "fetch_xor":
		return atomicBuiltinOrderKnown(order)
	default:
		return false
	}
}

func builtinNeedsUnsafe(name string, argRegions []int) bool {
	if isCoreAtomicBuiltin(name) {
		return true
	}
	switch name {
	case "core.alloc_bytes", "core.island_new", "core.island_reset", "core.cap_io", "core.cap_mem",
		"core.raw_slice_u8_from_parts", "core.raw_slice_u16_from_parts",
		"core.raw_slice_i32_from_parts", "core.raw_slice_bool_from_parts",
		"core.load_i32", "core.store_i32",
		"core.load_u8", "core.store_u8",
		"core.load_ptr", "core.store_ptr", "core.store_arch_ptr",
		"core.ptr_add",
		"core.mmio_read_i32", "core.mmio_write_i32",
		"core.sym_addr", "core.ctx_switch",
		"core.actor_ref_local", "core.actor_ref_slot":
		return true
	case "core.island_make_u8",
		"core.island_make_u16",
		"core.island_make_i32",
		"core.island_make_bool":
		if len(argRegions) == 0 {
			return true
		}
		return argRegions[0] == regionNone
	default:
		return false
	}
}

func builtinCapsulePermission(name string) (permission string, attenuatedEffect string) {
	if isCoreAtomicBuiltin(name) {
		return "capsule.mem", "mem"
	}
	switch name {
	case "core.cap_io", "core.mmio_read_i32", "core.mmio_write_i32":
		return "capsule.io", "io"
	case "core.cap_mem",
		"core.raw_slice_u8_from_parts", "core.raw_slice_u16_from_parts",
		"core.raw_slice_i32_from_parts", "core.raw_slice_bool_from_parts",
		"core.load_i32", "core.store_i32",
		"core.load_u8", "core.store_u8",
		"core.load_ptr", "core.store_ptr", "core.store_arch_ptr",
		"core.ptr_add", "core.ctx_switch":
		return "capsule.mem", "mem"
	default:
		return "", ""
	}
}

func ResolveBuiltinAlias(name string) (string, bool) {
	switch name {
	case "alloc_bytes":
		return "core.alloc_bytes", true
	case "make_u8":
		return "core.make_u8", true
	case "make_u16":
		return "core.make_u16", true
	case "make_i32":
		return "core.make_i32", true
	case "make_bool":
		return "core.make_bool", true
	case "island_new":
		return "core.island_new", true
	case "island_make_u8":
		return "core.island_make_u8", true
	case "island_make_u16":
		return "core.island_make_u16", true
	case "island_make_i32":
		return "core.island_make_i32", true
	case "island_make_bool":
		return "core.island_make_bool", true
	case "island_reset":
		return "core.island_reset", true
	case "load_ptr":
		return "core.load_ptr", true
	case "store_ptr":
		return "core.store_ptr", true
	case "store_arch_ptr":
		return "core.store_arch_ptr", true
	case "sym_addr":
		return "core.sym_addr", true
	case "ctx_switch":
		return "core.ctx_switch", true
	case "surface_open":
		return "core.surface_open", true
	case "surface_close":
		return "core.surface_close", true
	case "surface_poll_event_kind":
		return "core.surface_poll_event_kind", true
	case "surface_poll_event_x":
		return "core.surface_poll_event_x", true
	case "surface_poll_event_y":
		return "core.surface_poll_event_y", true
	case "surface_poll_event_button":
		return "core.surface_poll_event_button", true
	case "surface_poll_event_into":
		return "core.surface_poll_event_into", true
	case "surface_poll_event_text_len":
		return "core.surface_poll_event_text_len", true
	case "surface_poll_event_text_into":
		return "core.surface_poll_event_text_into", true
	case "surface_clipboard_write_text":
		return "core.surface_clipboard_write_text", true
	case "surface_clipboard_read_text_into":
		return "core.surface_clipboard_read_text_into", true
	case "surface_poll_composition_into":
		return "core.surface_poll_composition_into", true
	case "surface_begin_frame":
		return "core.surface_begin_frame", true
	case "surface_present_rgba":
		return "core.surface_present_rgba", true
	case "surface_now_ms":
		return "core.surface_now_ms", true
	case "surface_request_redraw":
		return "core.surface_request_redraw", true
	case "time_now_ms":
		return "core.time_now_ms", true
	case "sleep_ms":
		return "core.sleep_ms", true
	case "deadline_ms":
		return "core.deadline_ms", true
	case "sleep_until":
		return "core.sleep_until", true
	case "timer_ready":
		return "core.timer_ready", true
	case "yield":
		return "core.yield", true
	case "task_spawn_i32":
		return "core.task_spawn_i32", true
	case "task_spawn_i32_typed":
		return "core.task_spawn_i32_typed", true
	case "task_spawn_group_i32":
		return "core.task_spawn_group_i32", true
	case "task_spawn_group_i32_typed":
		return "core.task_spawn_group_i32_typed", true
	case "task_join_i32":
		return "core.task_join_i32", true
	case "task_join_i32_typed":
		return "core.task_join_i32_typed", true
	case "task_join_group_i32_typed":
		return "core.task_join_group_i32_typed", true
	case "task_join_result_i32":
		return "core.task_join_result_i32", true
	case "task_join_until_i32":
		return "core.task_join_until_i32", true
	case "task_poll_i32":
		return "core.task_poll_i32", true
	case "select2_i32":
		return "core.select2_i32", true
	case "task_group_open":
		return "core.task_group_open", true
	case "task_group_close":
		return "core.task_group_close", true
	case "task_group_cancel":
		return "core.task_group_cancel", true
	case "task_group_current":
		return "core.task_group_current", true
	case "task_group_status":
		return "core.task_group_status", true
	case "task_is_canceled":
		return "core.task_is_canceled", true
	case "task_checkpoint":
		return "core.task_checkpoint", true
	case "send_msg":
		return "core.send_msg", true
	case "recv_msg":
		return "core.recv_msg", true
	case "recv_poll":
		return "core.recv_poll", true
	case "recv_until":
		return "core.recv_until", true
	case "recv_msg_until":
		return "core.recv_msg_until", true
	case "send_typed":
		return "core.send_typed", true
	case "recv_typed":
		return "core.recv_typed", true
	case "actor_dispatch":
		return "core.actor_dispatch", true
	case "actor_main_entry_id":
		return "core.actor_main_entry_id", true
	case "actor_status":
		return "core.actor_status", true
	case "actor_status_raw":
		return "core.actor_status_raw", true
	case "actor_wait":
		return "core.actor_wait", true
	case "actor_wait_until":
		return "core.actor_wait_until", true
	case "actor_stop":
		return "core.actor_stop", true
	case "actor_exit_reason":
		return "core.actor_exit_reason", true
	case "actor_link":
		return "core.actor_link", true
	case "actor_unlink":
		return "core.actor_unlink", true
	case "actor_monitor":
		return "core.actor_monitor", true
	case "actor_demonitor":
		return "core.actor_demonitor", true
	case "actor_set_trap_exit":
		return "core.actor_set_trap_exit", true
	case "consent_token":
		return "core.consent_token", true
	case "secret_seal_i32":
		return "core.secret_seal_i32", true
	case "secret_unseal_i32":
		return "core.secret_unseal_i32", true
	case "core.alloc_bytes", "core.make_u8", "core.make_u16", "core.make_i32", "core.make_bool",
		"core.raw_slice_u8_from_parts", "core.raw_slice_u16_from_parts",
		"core.raw_slice_i32_from_parts", "core.raw_slice_bool_from_parts",
		"core.island_new", "core.island_make_u8", "core.island_make_u16", "core.island_make_i32", "core.island_make_bool", "core.island_reset",
		"core.load_ptr", "core.store_ptr", "core.sym_addr", "core.ctx_switch",
		"core.surface_open", "core.surface_close", "core.surface_poll_event_kind", "core.surface_poll_event_x",
		"core.surface_poll_event_y", "core.surface_poll_event_button", "core.surface_poll_event_into", "core.surface_poll_event_text_len", "core.surface_poll_event_text_into",
		"core.surface_clipboard_write_text", "core.surface_clipboard_read_text_into", "core.surface_poll_composition_into", "core.surface_begin_frame",
		"core.surface_present_rgba", "core.surface_now_ms", "core.surface_request_redraw",
		"core.time_now_ms", "core.sleep_ms", "core.sleep_until", "core.deadline_ms", "core.timer_ready", "core.yield",
		"core.task_group_open", "core.task_group_close", "core.task_group_cancel", "core.task_group_current", "core.task_group_status",
		"core.task_is_canceled", "core.task_checkpoint",
		"core.task_spawn_i32", "core.task_spawn_i32_typed", "core.task_spawn_group_i32", "core.task_spawn_group_i32_typed",
		"core.task_join_i32", "core.task_join_i32_typed", "core.task_join_group_i32_typed", "core.task_join_result_i32", "core.task_join_until_i32",
		"core.task_poll_i32", "core.select2_i32",
		"core.send_msg", "core.recv_msg", "core.recv_poll", "core.recv_until", "core.recv_msg_until", "core.send_typed", "core.recv_typed",
		"core.actor_dispatch", "core.actor_main_entry_id", "core.actor_node_connect", "core.actor_node_status", "core.spawn_remote",
		"core.actor_status", "core.actor_status_raw", "core.actor_wait", "core.actor_wait_until", "core.actor_stop", "core.actor_exit_reason",
		"core.actor_link", "core.actor_unlink", "core.actor_monitor", "core.actor_demonitor", "core.actor_set_trap_exit",
		"core.actor_recv_system", "core.actor_recv_system_poll", "core.actor_recv_system_until",
		"core.consent_token", "core.secret_seal_i32", "core.secret_unseal_i32":
		return name, true
	default:
		return "", false
	}
}

// ---- diagnostics.go ----

const (
	DiagnosticCodeSafetyOwnership = "TETRA2101"
	DiagnosticCodeSafetyLifetime  = "TETRA2102"
	DiagnosticCodeSafetyEffect    = "TETRA2103"
	DiagnosticCodeSafetyPrivacy   = "TETRA2104"
	DiagnosticCodeSafetyBudget    = "TETRA2105"
)

type diagnosticError struct {
	code    string
	pos     frontend.Position
	message string
	hint    string
}

func (e *diagnosticError) Error() string {
	if e.pos.Line > 0 && e.pos.Col > 0 {
		return frontend.FormatPos(e.pos) + ": " + e.message
	}
	return e.message
}

func (e *diagnosticError) Diagnostic() frontend.Diagnostic {
	return frontend.Diagnostic{
		Code:     e.code,
		Message:  e.message,
		File:     e.pos.File,
		Line:     e.pos.Line,
		Column:   e.pos.Col,
		Severity: "error",
		Hint:     e.hint,
	}
}

func ownershipDiagnosticf(pos frontend.Position, format string, args ...interface{}) error {
	return safetyDiagnosticf(pos, DiagnosticCodeSafetyOwnership, format, args...)
}

func lifetimeDiagnosticf(pos frontend.Position, format string, args ...interface{}) error {
	return safetyDiagnosticf(pos, DiagnosticCodeSafetyLifetime, format, args...)
}

func effectDiagnosticf(pos frontend.Position, format string, args ...interface{}) error {
	return safetyDiagnosticf(pos, DiagnosticCodeSafetyEffect, format, args...)
}

func privacyDiagnosticf(pos frontend.Position, format string, args ...interface{}) error {
	return safetyDiagnosticf(pos, DiagnosticCodeSafetyPrivacy, format, args...)
}

func budgetDiagnosticf(pos frontend.Position, format string, args ...interface{}) error {
	return safetyDiagnosticf(pos, DiagnosticCodeSafetyBudget, format, args...)
}

func unsupportedFunctionValueEscapeError(pos frontend.Position, name string) error {
	return lifetimeDiagnosticf(
		pos,
		("function value '%s' cannot escape outside the supported " +
			"fnptr ABI; use a declared fn(...) parameter, function-typed " +
			"return, local, struct field, enum payload, or supported " +
			"same-module global snapshot"),
		name,
	)
}

func unsupportedCallableMutableCaptureEscapeError(
	pos frontend.Position,
	kind CallableEscapeKind,
	name string,
) error {
	return lifetimeDiagnosticf(
		pos,
		("%s-escaped function value captures mutable local '%s'; " +
			"mutable by-reference captures require a proven lifetime and " +
			"synchronization model"),
		kind,
		name,
	)
}

func unsupportedCallableResourceCaptureEscapeError(
	pos frontend.Position,
	name, typeName string,
) error {
	return lifetimeDiagnosticf(
		pos,
		("escaped function value captures local '%s' of type '%s'; " +
			"pointer or resource captures require an explicit ownership " +
			"transfer model"),
		name,
		typeName,
	)
}

func unsupportedCapturingClosurePointerEscapeError(pos frontend.Position, name string) error {
	return lifetimeDiagnosticf(
		pos,
		("capturing closure '%s' cannot escape as raw ptr; bind it to " +
			"a declared fn(...) value for the supported by-value fnptr " +
			"snapshot ABI"),
		name,
	)
}

func unsupportedFunctionTypedExplicitTypeArgsError(pos frontend.Position, phrase string) error {
	return lifetimeDiagnosticf(
		pos,
		("explicit type arguments are not supported for %s; " +
			"function-typed dispatch uses a monomorphic fnptr ABI, so " +
			"remove explicit type arguments"),
		phrase,
	)
}

func unsupportedFunctionValueCallMessage(name string) string {
	return fmt.Sprintf(
		("function value '%s' cannot be called through the supported " +
			"fnptr ABI; use a let-bound closure, function-typed " +
			"local/global/struct field, enum payload, callback parameter," +
			" or direct named function symbol"),
		name,
	)
}

func unsupportedFunctionValueCallError(pos frontend.Position, name string) error {
	return lifetimeDiagnosticf(pos, "%s", unsupportedFunctionValueCallMessage(name))
}

func unsupportedCallbackUnknownSemanticTargetError(
	pos frontend.Position,
	calleeName, clause string,
) error {
	return fmt.Errorf(
		("%s: callback argument for '%s' has no known fnptr target " +
			"under semantic clause '%s'; pass a direct named " +
			"function/closure symbol or a function-typed value with a " +
			"stable target set"),
		frontend.FormatPos(pos),
		calleeName,
		clause,
	)
}

func unsupportedGenericClosureCaptureError(pos frontend.Position, name string) error {
	return lifetimeDiagnosticf(
		pos,
		("generic closure literal captures local '%s'; generic " +
			"closure captures are not supported by the production fnptr " +
			"ABI; use a non-generic closure or pass captured state " +
			"explicitly"),
		name,
	)
}

func unsupportedGenericClosureCallbackCaptureError(pos frontend.Position, name string) error {
	return lifetimeDiagnosticf(
		pos,
		("callback argument 'closure literal' captures local '%s'; " +
			"generic closure captures are not supported by the " +
			"production fnptr ABI; use a non-generic closure or pass " +
			"captured state explicitly"),
		name,
	)
}

func unsupportedGenericClosurePointerEscapeError(pos frontend.Position, name string) error {
	return lifetimeDiagnosticf(pos, "%s", genericClosurePointerEscapeMessage(name))
}

func unsupportedGenericClosureDirectCallError(pos frontend.Position, name string) error {
	return lifetimeDiagnosticf(pos, "%s", genericClosureDirectCallRequirementMessage(name))
}

func genericClosurePointerEscapeMessage(name string) string {
	return fmt.Sprintf(
		("generic closure '%s' cannot be used as a pointer value; " +
			"generic closure ABI support is limited to let-bound direct " +
			"local calls with inferable concrete arguments"),
		name,
	)
}

func genericClosureDirectCallRequirementMessage(name string) string {
	return fmt.Sprintf(
		("generic closure '%s' requires the generic direct-call " +
			"closure ABI: let-bound direct local call with inferable " +
			"concrete arguments"),
		name,
	)
}

func unsupportedGenericCallbackSymbolError(pos frontend.Position, name string) error {
	return fmt.Errorf(
		("%s: generic function symbol '%s' cannot be used as callback " +
			"argument; callback fnptr ABI requires a monomorphic target " +
			"at the call site"),
		frontend.FormatPos(pos),
		name,
	)
}

func unsupportedThrowingCallbackSymbolError(pos frontend.Position, name string) error {
	return fmt.Errorf(
		("%s: throwing function symbol '%s' cannot be used as " +
			"callback argument; callback fnptr ABI requires the " +
			"parameter's declared throws type to match"),
		frontend.FormatPos(pos),
		name,
	)
}

func unsupportedImportedMutableFunctionTypedGlobalCallError(
	pos frontend.Position,
	name string,
) error {
	return lifetimeDiagnosticf(
		pos,
		("imported mutable function-typed global '%s' cannot be " +
			"called directly across module boundary; cross-module " +
			"mutable global-data ABI is not available, expose a " +
			"module-local function wrapper or immutable public " +
			"function-typed global"),
		name,
	)
}

func unsupportedImportedMutableFunctionTypedGlobalUseError(
	pos frontend.Position,
	name string,
) error {
	return lifetimeDiagnosticf(
		pos,
		("imported mutable function-typed global '%s' cannot be used " +
			"across module boundary; cross-module mutable global-data " +
			"ABI is not available, expose a module-local function " +
			"wrapper or immutable public function-typed global"),
		name,
	)
}

func unsupportedFunctionTypedGlobalTargetError(pos frontend.Position, name string) error {
	return fmt.Errorf(
		("%s: function-typed global '%s' requires a symbol-backed " +
			"function value for the supported fnptr ABI"),
		frontend.FormatPos(pos),
		name,
	)
}

func unsupportedFunctionTypedGlobalSameModuleInitializerError(
	pos frontend.Position,
	name string,
) error {
	return fmt.Errorf(
		("%s: function-typed global '%s' initializer must be a " +
			"same-module named function symbol for the supported fnptr " +
			"ABI"),
		frontend.FormatPos(pos),
		name,
	)
}

func unsupportedFunctionTypedGlobalImportedInitializerError(
	pos frontend.Position,
	name string,
) error {
	return fmt.Errorf(
		("%s: function-typed global '%s' initializer must be an " +
			"imported public function symbol for the supported fnptr ABI"),
		frontend.FormatPos(pos),
		name,
	)
}

func unsupportedGenericFunctionTypedGlobalInitializerError(
	pos frontend.Position,
	symbol, name string,
) error {
	return fmt.Errorf(
		("%s: generic function symbol '%s' cannot initialize " +
			"function-typed global '%s'; global fnptr ABI requires a " +
			"monomorphic target"),
		frontend.FormatPos(pos),
		symbol,
		name,
	)
}

func unsupportedFunctionTypedGlobalInitializerSourceError(
	pos frontend.Position,
	name string,
) error {
	return fmt.Errorf(
		("%s: function-typed global '%s' must be initialized with a " +
			"direct named function symbol or closure literal for the " +
			"supported fnptr ABI"),
		frontend.FormatPos(pos),
		name,
	)
}

func unsupportedFunctionTypedLocalInitializerSourceError(pos frontend.Position, name string) error {
	return fmt.Errorf(
		("%s: function-typed local '%s' initializer must be a " +
			"symbol-backed function value, target-set-backed function " +
			"value, direct named function symbol, or closure literal for " +
			"the supported fnptr ABI"),
		frontend.FormatPos(pos),
		name,
	)
}

func unsupportedFunctionTypedLocalInitializerReturnCallSourceError(
	pos frontend.Position,
	name, callName string,
) error {
	return fmt.Errorf(
		("%s: function-typed local '%s' initializer call '%s' must " +
			"resolve to a function-typed return for the supported fnptr " +
			"ABI"),
		frontend.FormatPos(pos),
		name,
		callName,
	)
}

func unsupportedGenericFunctionTypedLocalInitializerError(
	pos frontend.Position,
	symbol, name string,
) error {
	return fmt.Errorf(
		("%s: generic function symbol '%s' cannot initialize " +
			"function-typed local '%s'; local fnptr ABI requires a " +
			"monomorphic target"),
		frontend.FormatPos(pos),
		symbol,
		name,
	)
}

func unsupportedGenericFunctionTypedStructFieldInitializerError(
	pos frontend.Position,
	symbol, name string,
) error {
	return fmt.Errorf(
		("%s: generic function symbol '%s' cannot initialize " +
			"function-typed struct field '%s'; struct-field fnptr ABI " +
			"requires a monomorphic target"),
		frontend.FormatPos(pos),
		symbol,
		name,
	)
}

func unsupportedGenericFunctionTypedEnumPayloadInitializerError(
	pos frontend.Position,
	symbol, name string,
) error {
	return fmt.Errorf(
		("%s: generic function symbol '%s' cannot initialize " +
			"function-typed enum payload '%s'; enum-payload fnptr ABI " +
			"requires a monomorphic target"),
		frontend.FormatPos(pos),
		symbol,
		name,
	)
}

func unsupportedThrowingFunctionTypedLocalInitializerError(
	pos frontend.Position,
	symbol, name string,
) error {
	return fmt.Errorf(
		("%s: throwing function symbol '%s' cannot initialize " +
			"function-typed local '%s'; local fnptr ABI requires the " +
			"declared throws type to match"),
		frontend.FormatPos(pos),
		symbol,
		name,
	)
}

func unsupportedGenericFunctionTypedAssignmentError(
	pos frontend.Position,
	symbol, targetName string,
) error {
	return fmt.Errorf(
		("%s: generic function symbol '%s' cannot be assigned to " +
			"function-typed target '%s'; assignment fnptr ABI requires a " +
			"monomorphic target"),
		frontend.FormatPos(pos),
		symbol,
		targetName,
	)
}

func unsupportedThrowingFunctionTypedAssignmentError(
	pos frontend.Position,
	symbol, targetName string,
) error {
	return fmt.Errorf(
		("%s: throwing function symbol '%s' cannot be assigned to " +
			"function-typed target '%s'; assignment fnptr ABI requires " +
			"the target's declared throws type to match"),
		frontend.FormatPos(pos),
		symbol,
		targetName,
	)
}

func unsupportedFunctionTypedAssignmentSourceError(pos frontend.Position, targetName string) error {
	return fmt.Errorf(
		("%s: function-typed assignment to '%s' must use a supported " +
			"fnptr source: closure literal, function-typed " +
			"local/global/struct field, direct named function/closure " +
			"symbol, or function-typed return call"),
		frontend.FormatPos(pos),
		targetName,
	)
}

func unsupportedFunctionTypedAssignmentReturnCallSourceError(
	pos frontend.Position,
	targetName, callName string,
) error {
	return fmt.Errorf(
		("%s: function-typed assignment to '%s' initializer call '%s' " +
			"must resolve to a function-typed return for the supported " +
			"fnptr ABI"),
		frontend.FormatPos(pos),
		targetName,
		callName,
	)
}

func unsupportedGenericFunctionTypedReturnError(pos frontend.Position, symbol string) error {
	return fmt.Errorf(
		("%s: generic function symbol '%s' cannot be returned as " +
			"function-typed value; return fnptr ABI requires a " +
			"monomorphic target"),
		frontend.FormatPos(pos),
		symbol,
	)
}

func unsupportedFunctionTypedReturnSourceError(pos frontend.Position) error {
	return fmt.Errorf(
		("%s: function-typed return must use a supported fnptr source:" +
			" closure literal, function-typed local/global/struct field, " +
			"direct named function/closure symbol, or function-typed " +
			"return call"),
		frontend.FormatPos(pos),
	)
}

func safetyDiagnosticf(
	pos frontend.Position,
	code string,
	format string,
	args ...interface{},
) error {
	return &diagnosticError{
		code:    code,
		pos:     pos,
		message: fmt.Sprintf(format, args...),
	}
}

// ---- inference.go ----

func inferExprTypeForDecl(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (string, error) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return "i32", nil
	case *frontend.BoolLitExpr:
		return "bool", nil
	case *frontend.StringLitExpr:
		return "str", nil
	case *frontend.NoneLitExpr:
		return "", fmt.Errorf("cannot infer type from 'none'; add an optional type annotation")
	case *frontend.MatchExpr:
		return inferMatchExprType(e, locals, globals, funcs, types, module, imports)
	case *frontend.CatchExpr:
		return inferCatchExprType(e, locals, globals, funcs, types, module, imports)
	case *frontend.IdentExpr:
		if info, ok := locals[e.Name]; ok {
			if info.TypeName == "" {
				return "", fmt.Errorf("depends on '%s' which has no type annotation", e.Name)
			}
			return info.TypeName, nil
		}
		if g, ok := globals[e.Name]; ok {
			return g.TypeName, nil
		}
		return "", fmt.Errorf("unknown identifier '%s'", e.Name)
	case *frontend.UnaryExpr:
		switch e.Op {
		case frontend.TokenMinus:
			return "i32", nil
		case frontend.TokenBang:
			return "bool", nil
		default:
			return "", fmt.Errorf("unsupported unary operator")
		}
	case *frontend.BinaryExpr:
		switch e.Op {
		case frontend.TokenPlus, frontend.TokenMinus, frontend.TokenStar, frontend.TokenSlash, frontend.TokenPercent:
			return "i32", nil
		case frontend.TokenEqEq, frontend.TokenBangEq, frontend.TokenLess, frontend.TokenGreater, frontend.TokenGreaterEq, frontend.TokenLessEq,
			frontend.TokenAmpAmp, frontend.TokenPipePipe:
			return "bool", nil
		default:
			return "", fmt.Errorf("unsupported binary operator")
		}
	case *frontend.FieldAccessExpr:
		if typeName, _, ok, err := resolveEnumCaseExpr(
			e,
			locals,
			globals,
			types,
			module,
			imports,
		); ok || err != nil {
			if err != nil {
				return "", err
			}
			return typeName, nil
		}
		_, targetType, err := ResolveFieldAccessType(e, locals, globals, types)
		if err != nil {
			return "", err
		}
		return targetType, nil
	case *frontend.IndexExpr:
		baseType, err := inferExprTypeForDecl(e.Base, locals, globals, funcs, types, module, imports)
		if err != nil {
			return "", err
		}
		info, err := ensureTypeInfo(baseType, types)
		if err != nil {
			return "", err
		}
		switch info.Kind {
		case TypeStr:
			return "u8", nil
		case TypeSlice:
			return info.ElemType, nil
		case TypeArray:
			return info.ElemType, nil
		default:
			return "", fmt.Errorf("cannot index '%s'", baseType)
		}
	case *frontend.StructLitExpr:
		resolved, err := resolveTypeName(&e.Type, module, imports)
		if err != nil {
			return "", err
		}
		return resolved, nil
	case *frontend.CallExpr:
		if enumType, _, ok, err := resolveEnumCaseConstructorCall(
			e,
			types,
			module,
			imports,
		); ok || err != nil {
			if err != nil {
				return "", err
			}
			return enumType, nil
		}
		if rewritten, err := rewriteSliceViewMethodCall(
			e,
			locals,
			globals,
			types,
		); rewritten || err != nil {
			if err != nil {
				return "", err
			}
			sig, ok := funcs[e.Name]
			if !ok {
				return "", fmt.Errorf("unknown function '%s'", e.Name)
			}
			return sig.ReturnType, nil
		}
		if builtin, ok := ResolveBuiltinAlias(e.Name); ok && builtin == "core.recv_typed" {
			if len(e.TypeArgs) != 1 {
				return "", fmt.Errorf("recv_typed expects one explicit type argument")
			}
			typeName, err := resolveTypeName(&e.TypeArgs[0], module, imports)
			if err != nil {
				return "", err
			}
			e.TypeArgs[0].Name = typeName
			return typeName, nil
		}
		if builtin, ok := ResolveBuiltinAlias(e.Name); ok && builtin == "core.send_typed" {
			return "i32", nil
		}
		if builtin, ok := ResolveBuiltinAlias(
			e.Name,
		); ok && (builtin == "core.task_spawn_i32_typed" || builtin == "core.task_spawn_group_i32_typed") {
			if len(e.TypeArgs) != 1 {
				return "", fmt.Errorf("%s expects one explicit error type argument", builtin)
			}
			errorType, err := resolveTypeName(&e.TypeArgs[0], module, imports)
			if err != nil {
				return "", err
			}
			if err := validateTypedTaskErrorType(errorType, types, e.TypeArgs[0].At); err != nil {
				return "", err
			}
			e.TypeArgs[0].Name = errorType
			handleType, _, err := EnsureTypedTaskHandleType(errorType, types)
			if err != nil {
				return "", err
			}
			return handleType, nil
		}
		if builtin, ok := ResolveBuiltinAlias(
			e.Name,
		); ok && (builtin == "core.task_join_i32_typed" || builtin == "core.task_join_group_i32_typed") {
			return "i32", nil
		}
		if ctorType, ok, err := resolveStructConstructorCallType(e, types, module, imports); ok {
			return ctorType, err
		}
		resolved := ""
		if local, ok := locals[e.Name]; ok {
			if local.FunctionValue == "" || (local.FunctionTypeValue && len(
				local.FunctionCaptures,
			) == 0 && local.SlotCount == FnPtrSlotCount) {
				if !local.FunctionTypeValue {
					return "", fmt.Errorf("%s", unsupportedFunctionValueCallMessage(e.Name))
				}
				if len(local.FunctionCaptures) > 0 {
					return "", fmt.Errorf(("function-typed callback '%s' captures local values; " +
						"captured function values cannot be called through function " +
						"type in this MVP"), e.Name)
				}
				if len(e.Args) != len(local.FunctionParamTypes) {
					return "", fmt.Errorf("wrong argument count for callback '%s'", e.Name)
				}
				return local.FunctionReturnType, nil
			}
			if local.GenericFunctionValue {
				return "", fmt.Errorf("%s", genericClosureDirectCallRequirementMessage(e.Name))
			}
			if err := appendClosureCaptureArgs(e, local); err != nil {
				return "", err
			}
			resolved = local.FunctionValue
			e.Name = resolved
		} else if builtin, ok := ResolveBuiltinAlias(e.Name); ok {
			resolved = builtin
		} else if _, ok := funcs[e.Name]; ok {
			resolved = e.Name
		} else {
			name, err := resolveCallName(e.Name, module, imports, e.At)
			if err != nil {
				return "", err
			}
			resolved = name
		}
		sig, ok := funcs[resolved]
		if !ok {
			return "", fmt.Errorf("unknown function '%s'", resolved)
		}
		if sig.Generic {
			return "", fmt.Errorf(("generic function '%s' could not be monomorphized; use " +
				"inferable value arguments"), e.Name)
		}
		return sig.ReturnType, nil
	case *frontend.ClosureExpr:
		return "ptr", nil
	case *frontend.TryExpr:
		call, ok := e.X.(*frontend.CallExpr)
		if !ok {
			if await, awaitOK := e.X.(*frontend.AwaitExpr); awaitOK {
				call, ok = await.X.(*frontend.CallExpr)
			}
		}
		if !ok {
			return "", fmt.Errorf("try expects a throwing function call")
		}
		resolved := ""
		if builtin, ok := ResolveBuiltinAlias(call.Name); ok {
			resolved = builtin
		} else if _, ok := funcs[call.Name]; ok {
			resolved = call.Name
		} else {
			name, err := resolveCallName(call.Name, module, imports, call.At)
			if err != nil {
				return "", err
			}
			resolved = name
		}
		sig, ok := funcs[resolved]
		if !ok {
			return "", fmt.Errorf("unknown function '%s'", resolved)
		}
		if sig.ThrowsType == "" {
			return "", fmt.Errorf("try expects a throwing function call")
		}
		return sig.ReturnType, nil
	case *frontend.AwaitExpr:
		call, ok := e.X.(*frontend.CallExpr)
		if !ok {
			return "", fmt.Errorf("await expects an async function call")
		}
		resolved := ""
		if builtin, ok := ResolveBuiltinAlias(call.Name); ok {
			resolved = builtin
		} else if _, ok := funcs[call.Name]; ok {
			resolved = call.Name
		} else {
			name, err := resolveCallName(call.Name, module, imports, call.At)
			if err != nil {
				return "", err
			}
			resolved = name
		}
		sig, ok := funcs[resolved]
		if !ok {
			return "", fmt.Errorf("unknown function '%s'", resolved)
		}
		if !sig.Async {
			return "", fmt.Errorf("await expects an async function call")
		}
		return sig.ReturnType, nil
	default:
		return "", fmt.Errorf("unsupported expression for type inference")
	}
}

func resolveStructConstructorCallType(
	e *frontend.CallExpr,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (string, bool, error) {
	if len(e.Args) == 0 || len(e.ArgLabels) != len(e.Args) {
		return "", false, nil
	}
	for _, label := range e.ArgLabels {
		if label == "" {
			return "", false, nil
		}
	}

	ref := frontend.TypeRef{At: e.At, Kind: frontend.TypeRefNamed, Name: e.Name}
	resolved, err := resolveTypeName(&ref, module, imports)
	if err != nil {
		return "", false, nil
	}
	info, ok := types[resolved]
	if !ok || info.Kind != TypeStruct {
		return "", false, nil
	}
	if len(e.Args) != len(info.Fields) {
		return "", true, fmt.Errorf("wrong field count for '%s'", resolved)
	}

	seen := make(map[string]struct{}, len(e.ArgLabels))
	for _, label := range e.ArgLabels {
		if _, exists := seen[label]; exists {
			return "", true, fmt.Errorf("duplicate field '%s'", label)
		}
		seen[label] = struct{}{}
		if _, ok := info.FieldMap[label]; !ok {
			return "", true, fmt.Errorf("unknown field '%s'", label)
		}
	}
	for _, field := range info.Fields {
		if _, ok := seen[field.Name]; !ok {
			return "", true, fmt.Errorf("missing field '%s'", field.Name)
		}
	}
	return resolved, true, nil
}

// ---- manifest.go ----

type BuiltinManifest struct {
	Name          string   `json:"name"`
	Aliases       []string `json:"aliases,omitempty"`
	ParamTypes    []string `json:"param_types,omitempty"`
	ReturnType    string   `json:"return_type"`
	Effects       []string `json:"effects,omitempty"`
	UnsafePolicy  string   `json:"unsafe_policy"`            // never | always | conditional
	UnsafeDetails string   `json:"unsafe_details,omitempty"` // human-readable condition
}

// DescribeBuiltins returns a stable, sorted snapshot of builtin signatures and safety policies.
func DescribeBuiltins() ([]BuiltinManifest, error) {
	types := baseTypes()
	sigs, err := builtinFuncSigs(types)
	if err != nil {
		return nil, err
	}

	aliasesByTarget := make(map[string][]string)
	for _, alias := range []string{
		"alloc_bytes",
		"make_u8",
		"make_u16",
		"make_i32",
		"make_bool",
		"island_new",
		"island_make_u8",
		"island_make_u16",
		"island_make_i32",
		"island_make_bool",
		"island_reset",
		"load_ptr",
		"store_ptr",
		"store_arch_ptr",
		"sym_addr",
		"ctx_switch",
		"surface_open",
		"surface_close",
		"surface_poll_event_kind",
		"surface_poll_event_x",
		"surface_poll_event_y",
		"surface_poll_event_button",
		"surface_poll_event_into",
		"surface_poll_event_text_len",
		"surface_poll_event_text_into",
		"surface_clipboard_write_text",
		"surface_clipboard_read_text_into",
		"surface_poll_composition_into",
		"surface_begin_frame",
		"surface_present_rgba",
		"surface_now_ms",
		"surface_request_redraw",
		"time_now_ms",
		"sleep_ms",
		"sleep_until",
		"deadline_ms",
		"timer_ready",
		"yield",
		"task_group_open",
		"task_group_close",
		"task_group_cancel",
		"task_group_current",
		"task_group_status",
		"task_is_canceled",
		"task_checkpoint",
		"task_spawn_i32",
		"task_spawn_i32_typed",
		"task_spawn_group_i32",
		"task_spawn_group_i32_typed",
		"task_join_i32",
		"task_join_i32_typed",
		"task_join_group_i32_typed",
		"task_join_result_i32",
		"task_join_until_i32",
		"task_poll_i32",
		"select2_i32",
		"send_msg",
		"recv_msg",
		"recv_poll",
		"recv_until",
		"recv_msg_until",
		"send_typed",
		"recv_typed",
		"actor_dispatch",
		"actor_main_entry_id",
		"actor_status",
		"actor_status_raw",
		"actor_wait",
		"actor_wait_until",
		"actor_stop",
		"actor_exit_reason",
		"actor_link",
		"actor_unlink",
		"actor_monitor",
		"actor_demonitor",
		"actor_set_trap_exit",
		"consent_token",
		"secret_seal_i32",
		"secret_unseal_i32",
	} {
		if target, ok := ResolveBuiltinAlias(alias); ok {
			aliasesByTarget[target] = append(aliasesByTarget[target], alias)
			continue
		}
		return nil, fmt.Errorf("builtin alias '%s' has no target", alias)
	}
	for name, list := range aliasesByTarget {
		sort.Strings(list)
		aliasesByTarget[name] = list
	}

	out := make([]BuiltinManifest, 0, len(sigs))
	for name, sig := range sigs {
		effects := builtinEffects(name)
		unsafePolicy := "never"
		unsafeDetails := ""
		switch name {
		case "core.island_make_u8",
			"core.island_make_u16",
			"core.island_make_i32",
			"core.island_make_bool":
			unsafePolicy = "conditional"
			unsafeDetails = "requires unsafe when the island argument is not a scoped island variable"
		default:
			if builtinNeedsUnsafe(name, nil) {
				unsafePolicy = "always"
			}
		}
		out = append(out, BuiltinManifest{
			Name:          name,
			Aliases:       aliasesByTarget[name],
			ParamTypes:    append([]string(nil), sig.ParamTypes...),
			ReturnType:    sig.ReturnType,
			Effects:       effects,
			UnsafePolicy:  unsafePolicy,
			UnsafeDetails: unsafeDetails,
		})
		if err := validateBuiltinManifestEntry(out[len(out)-1]); err != nil {
			return nil, err
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func validateBuiltinManifestEntry(entry BuiltinManifest) error {
	if entry.Name == "" {
		return fmt.Errorf("manifest entry: missing name")
	}
	if entry.ReturnType == "" {
		return fmt.Errorf("manifest entry '%s': missing return type", entry.Name)
	}
	switch entry.UnsafePolicy {
	case "never", "always", "conditional":
		// valid
	default:
		return fmt.Errorf(
			"manifest entry '%s': unknown unsafe policy '%s'",
			entry.Name,
			entry.UnsafePolicy,
		)
	}
	for i := 1; i < len(entry.Aliases); i++ {
		if entry.Aliases[i-1] >= entry.Aliases[i] {
			return fmt.Errorf("manifest entry '%s': aliases must be sorted and unique", entry.Name)
		}
	}
	for i := 1; i < len(entry.Effects); i++ {
		if entry.Effects[i-1] >= entry.Effects[i] {
			return fmt.Errorf("manifest entry '%s': effects must be sorted and unique", entry.Name)
		}
	}
	return nil
}

func builtinEffects(name string) []string {
	if isCoreAtomicBuiltin(name) {
		return []string{"mem"}
	}
	var effects []string
	switch name {
	case "core.alloc_bytes":
		effects = []string{"alloc", "mem"}
	case "core.make_u8", "core.make_u16", "core.make_i32", "core.make_bool":
		effects = []string{"alloc", "mem"}
	case "core.slice_copy_u8",
		"core.slice_copy_u16",
		"core.slice_copy_i32",
		"core.slice_copy_bool",
		"core.string_copy":
		effects = []string{"alloc", "mem"}
	case "core.slice_copy_into_u8",
		"core.slice_copy_into_u16",
		"core.slice_copy_into_i32",
		"core.slice_copy_into_bool",
		"core.string_copy_into":
		effects = []string{"mem"}
	case "core.raw_slice_u8_from_parts", "core.raw_slice_u16_from_parts",
		"core.raw_slice_i32_from_parts", "core.raw_slice_bool_from_parts":
		effects = []string{"mem"}
	case "core.island_new":
		effects = []string{"alloc", "islands", "mem"}
	case "core.island_make_u8",
		"core.island_make_u16",
		"core.island_make_i32",
		"core.island_make_bool":
		effects = []string{"alloc", "islands", "mem"}
	case "core.island_reset":
		effects = []string{"islands", "mem"}
	case "core.cap_io":
		effects = []string{"capability", "io"}
	case "core.cap_mem":
		effects = []string{"capability", "mem"}
	case "core.load_i32", "core.store_i32",
		"core.load_u8", "core.store_u8",
		"core.load_ptr", "core.store_ptr", "core.store_arch_ptr",
		"core.ptr_add":
		effects = []string{"mem"}
	case "core.mmio_read_i32", "core.mmio_write_i32":
		effects = []string{"io", "mmio"}
	case "core.fs_exists":
		effects = []string{"io"}
	case "core.net_socket_tcp4",
		"core.net_bind_tcp4_loopback",
		"core.net_connect_tcp4_loopback",
		"core.net_listen",
		"core.net_accept4",
		"core.net_epoll_create",
		"core.net_epoll_ctl_add_read",
		"core.net_epoll_wait_one",
		"core.net_epoll_ctl_add_read_write",
		"core.net_epoll_ctl_mod_read",
		"core.net_epoll_ctl_mod_read_write",
		"core.net_epoll_ctl_delete",
		"core.net_set_nonblocking",
		"core.net_set_reuseport",
		"core.net_set_tcp_nodelay",
		"core.net_close":
		effects = []string{"io"}
	case "core.net_read",
		"core.net_recv",
		"core.net_write",
		"core.net_send",
		"core.net_epoll_wait_one_into":
		effects = []string{"io", "mem"}
	case "core.sym_addr":
		effects = []string{"link"}
	case "core.ctx_switch":
		effects = []string{"control", "runtime"}
	case "core.surface_open",
		"core.surface_close",
		"core.surface_poll_event_kind",
		"core.surface_poll_event_x",
		"core.surface_poll_event_y",
		"core.surface_poll_event_button",
		"core.surface_poll_event_text_len",
		"core.surface_begin_frame",
		"core.surface_now_ms",
		"core.surface_request_redraw":
		effects = []string{"surface"}
	case "core.surface_present_rgba",
		"core.surface_poll_event_into",
		"core.surface_poll_event_text_into",
		"core.surface_clipboard_write_text",
		"core.surface_clipboard_read_text_into",
		"core.surface_poll_composition_into":
		effects = []string{"mem", "surface"}
	case "core.time_now_ms",
		"core.sleep_ms",
		"core.sleep_until",
		"core.deadline_ms",
		"core.timer_ready":
		effects = []string{"runtime"}
	case "core.yield":
		effects = []string{"actors", "runtime"}
	case "core.task_group_open",
		"core.task_group_close",
		"core.task_group_cancel",
		"core.task_group_current",
		"core.task_group_status",
		"core.task_is_canceled",
		"core.task_checkpoint",
		"core.task_spawn_i32",
		"core.task_spawn_i32_typed",
		"core.task_spawn_group_i32",
		"core.task_spawn_group_i32_typed",
		"core.task_join_i32",
		"core.task_join_i32_typed",
		"core.task_join_group_i32_typed",
		"core.task_join_result_i32",
		"core.task_join_until_i32",
		"core.task_poll_i32",
		"core.select2_i32":
		effects = []string{"runtime"}
	case "core.recv_until", "core.recv_msg_until", "core.actor_recv_system", "core.actor_recv_system_until":
		effects = []string{"actors", "runtime"}
	case "core.recv_poll", "core.actor_recv_system_poll":
		effects = []string{"actors"}
	case "core.actor_dispatch", "core.actor_main_entry_id",
		"core.actor_ref_local", "core.actor_ref_slot",
		"core.spawn", "core.send", "core.send_msg", "core.recv", "core.recv_msg", "core.send_typed", "core.recv_typed", "core.self", "core.sender",
		"core.actor_status", "core.actor_status_raw", "core.actor_wait", "core.actor_wait_until", "core.actor_stop", "core.actor_exit_reason",
		"core.actor_link", "core.actor_unlink", "core.actor_monitor", "core.actor_demonitor", "core.actor_set_trap_exit":
		effects = []string{"actors"}
	case "core.actor_node_connect", "core.actor_node_status", "core.spawn_remote":
		effects = []string{"actors", "runtime"}
	case "core.consent_token", "core.secret_seal_i32", "core.secret_unseal_i32":
		effects = []string{"privacy"}
	}
	sort.Strings(effects)
	return effects
}

// ---- representation_metadata.go ----

type RepresentationMetadataField struct {
	Name                 string
	AppliesToTypes       []TypeKind
	ReadableInSafeCode   bool
	AssignableInSafeCode bool
	SourceFactKind       string
}

const representationMetadataSourceFactKind = "safe_representation_metadata:not_user_assignable"

var representationMetadataRegistry = []RepresentationMetadataField{
	{
		Name:               "ptr",
		AppliesToTypes:     []TypeKind{TypeSlice, TypeArray, TypeStr},
		ReadableInSafeCode: true,
		SourceFactKind:     representationMetadataSourceFactKind,
	},
	{
		Name:               "len",
		AppliesToTypes:     []TypeKind{TypeSlice, TypeArray, TypeStr},
		ReadableInSafeCode: true,
		SourceFactKind:     representationMetadataSourceFactKind,
	},
	{
		Name:           "owner_id",
		AppliesToTypes: []TypeKind{TypeSlice, TypeArray, TypeStr},
		SourceFactKind: representationMetadataSourceFactKind,
	},
	{
		Name:           "region_id",
		AppliesToTypes: []TypeKind{TypeSlice, TypeArray, TypeStr},
		SourceFactKind: representationMetadataSourceFactKind,
	},
	{
		Name:           "provenance_id",
		AppliesToTypes: []TypeKind{TypeSlice, TypeArray, TypeStr},
		SourceFactKind: representationMetadataSourceFactKind,
	},
	{
		Name:           "borrow_source",
		AppliesToTypes: []TypeKind{TypeSlice, TypeArray, TypeStr},
		SourceFactKind: representationMetadataSourceFactKind,
	},
	{
		Name:           "storage_class",
		AppliesToTypes: []TypeKind{TypeSlice, TypeArray, TypeStr},
		SourceFactKind: representationMetadataSourceFactKind,
	},
	{
		Name:           "unsafe_class",
		AppliesToTypes: []TypeKind{TypeSlice, TypeArray, TypeStr},
		SourceFactKind: representationMetadataSourceFactKind,
	},
}

func representationMetadataByName(name string) (RepresentationMetadataField, bool) {
	for _, field := range representationMetadataRegistry {
		if field.Name == name {
			return field, true
		}
	}
	return RepresentationMetadataField{}, false
}

func isReservedRepresentationMetadataField(field string) bool {
	_, ok := representationMetadataByName(field)
	return ok
}

// ---- resolution.go ----

const importSymbolPrefix = semanticsworld.ImportSymbolPrefix

func collectImportAliases(file *frontend.FileAST) (map[string]string, error) {
	return semanticsworld.CollectImportAliases(file)
}

func importSymbolTarget(target string) (string, bool) {
	return semanticsworld.ImportSymbolTarget(target)
}

func topLevelDeclarationNames(file *frontend.FileAST) map[string]struct{} {
	return semanticsworld.TopLevelDeclarationNames(file)
}

func qualifyName(module, name string) string {
	return semanticsworld.QualifyName(module, name)
}

func resolveTypeName(
	ref *frontend.TypeRef,
	module string,
	imports map[string]string,
) (string, error) {
	if ref == nil {
		return "", fmt.Errorf("missing type")
	}
	switch ref.Kind {
	case frontend.TypeRefSlice:
		if ref.Elem == nil {
			return "", fmt.Errorf("%s: missing slice element type", frontend.FormatPos(ref.At))
		}
		elem, err := resolveTypeName(ref.Elem, module, imports)
		if err != nil {
			return "", err
		}
		return "[]" + elem, nil
	case frontend.TypeRefArray:
		if ref.Elem == nil {
			return "", fmt.Errorf("%s: missing array element type", frontend.FormatPos(ref.At))
		}
		if ref.Len <= 0 {
			return "", fmt.Errorf(
				"%s: array size must be positive constant",
				frontend.FormatPos(ref.At),
			)
		}
		elem, err := resolveTypeName(ref.Elem, module, imports)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("[%d]%s", ref.Len, elem), nil
	case frontend.TypeRefOptional:
		if ref.Elem == nil {
			return "", fmt.Errorf("%s: missing optional payload type", frontend.FormatPos(ref.At))
		}
		elem, err := resolveTypeName(ref.Elem, module, imports)
		if err != nil {
			return "", err
		}
		return optionalTypeName(elem), nil
	case frontend.TypeRefNamed:
		if ref.Name == "" {
			return "", fmt.Errorf("%s: missing type name", frontend.FormatPos(ref.At))
		}
		if canonical, ok := canonicalBuiltinType(ref.Name); ok {
			return canonical, nil
		}
		parts := strings.Split(ref.Name, ".")
		if len(parts) == 1 {
			if target, ok := imports[ref.Name]; ok {
				if symbol, isSymbol := importSymbolTarget(target); isSymbol {
					return symbol, nil
				}
			}
			return qualifyName(module, ref.Name), nil
		}
		if target, ok := imports[parts[0]]; ok {
			if _, isSymbol := importSymbolTarget(target); isSymbol {
				return "", fmt.Errorf(
					"%s: selective import '%s' cannot be used as a namespace",
					frontend.FormatPos(ref.At),
					parts[0],
				)
			}
			if len(parts) != 2 {
				return "", fmt.Errorf(
					"%s: expected '%s.<type>'",
					frontend.FormatPos(ref.At),
					parts[0],
				)
			}
			return target + "." + parts[1], nil
		}
		return ref.Name, nil
	case frontend.TypeRefFunction:
		for i := range ref.Params {
			paramName, err := resolveTypeName(&ref.Params[i], module, imports)
			if err != nil {
				return "", err
			}
			ref.Params[i].Name = paramName
		}
		if ref.Return == nil {
			return "", fmt.Errorf("%s: missing function return type", frontend.FormatPos(ref.At))
		}
		retName, err := resolveTypeName(ref.Return, module, imports)
		if err != nil {
			return "", err
		}
		ref.Return.Name = retName
		if ref.Throws != nil {
			throwsName, err := resolveTypeName(ref.Throws, module, imports)
			if err != nil {
				return "", err
			}
			ref.Throws.Name = throwsName
		}
		if _, err := normalizeEffects(ref.Uses, ref.At); err != nil {
			return "", err
		}
		return "fnptr", nil
	default:
		return "", fmt.Errorf(
			"%s: unsupported type reference kind %d",
			frontend.FormatPos(ref.At),
			ref.Kind,
		)
	}
}

func canonicalBuiltinType(name string) (string, bool) {
	switch name {
	case "i32", "Int":
		return "i32", true
	case "i64", "Int64":
		return "i64", true
	case "u8", "UInt8", "Byte":
		return "u8", true
	case "u16", "UInt16":
		return "u16", true
	case "c_int":
		return "c_int", true
	case "c_uint":
		return "c_uint", true
	case "usize", "isize", "size_t", "ssize_t", "native_int", "native_uint", "c_long", "c_ulong":
		return name, true
	case "str", "String":
		return "str", true
	case "bool", "Bool":
		return "bool", true
	case "ptr",
		"rawptr",
		"nullable_ptr",
		"ref",
		"island",
		"cap.io",
		"cap.mem",
		"actor",
		"consent.token",
		"secret.i32":
		return name, true
	case "ConsentToken":
		return "consent.token", true
	case "SecretInt":
		return "secret.i32", true
	default:
		return "", false
	}
}

func resolveEnumCaseExpr(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (string, EnumCaseInfo, bool, error) {
	field, ok := expr.(*frontend.FieldAccessExpr)
	if !ok {
		return "", EnumCaseInfo{}, false, nil
	}
	baseName, fields, pos, ok := splitFieldPath(field.Base)
	if !ok {
		return "", EnumCaseInfo{}, false, nil
	}
	if _, exists := locals[baseName]; exists {
		return "", EnumCaseInfo{}, false, nil
	}
	if _, exists := globals[baseName]; exists {
		return "", EnumCaseInfo{}, false, nil
	}
	parts := append([]string{baseName}, fields...)
	ref := frontend.TypeRef{At: pos, Kind: frontend.TypeRefNamed, Name: strings.Join(parts, ".")}
	typeName, err := resolveTypeName(&ref, module, imports)
	if err != nil {
		return "", EnumCaseInfo{}, false, err
	}
	info, ok := types[typeName]
	if !ok || info.Kind != TypeEnum {
		if altName, altInfo, found := findUniqueEnumByShortName(ref.Name, types); found {
			typeName = altName
			info = altInfo
		} else {
			return "", EnumCaseInfo{}, false, nil
		}
	}
	caseInfo, ok := info.CaseMap[field.Field]
	if !ok {
		return "", EnumCaseInfo{}, true, fmt.Errorf(
			"%s: unknown enum case '%s' for '%s'",
			frontend.FormatPos(field.At),
			field.Field,
			displayTypeName(typeName, module),
		)
	}
	if len(caseInfo.PayloadTypes) > 0 {
		return "", EnumCaseInfo{}, true, fmt.Errorf(
			"%s: enum case '%s.%s' requires payload arguments",
			frontend.FormatPos(field.At),
			displayTypeName(typeName, module),
			field.Field,
		)
	}
	if len(caseInfo.PayloadTypes) == 0 && field.Field == "" {
		return "", EnumCaseInfo{}, true, fmt.Errorf(
			"%s: malformed enum case reference",
			frontend.FormatPos(field.At),
		)
	}
	field.EnumType = typeName
	field.EnumOrdinal = caseInfo.Ordinal
	return typeName, caseInfo, true, nil
}

func resolveEnumCasePattern(
	pattern *frontend.EnumCasePatternExpr,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (string, EnumCaseInfo, bool, error) {
	ref := frontend.TypeRef{At: pattern.At, Kind: frontend.TypeRefNamed, Name: pattern.TypeName}
	typeName, err := resolveTypeName(&ref, module, imports)
	if err != nil {
		return "", EnumCaseInfo{}, false, err
	}
	info, ok := types[typeName]
	if !ok || info.Kind != TypeEnum {
		if altName, altInfo, found := findUniqueEnumByShortName(pattern.TypeName, types); found {
			typeName = altName
			info = altInfo
		} else {
			return "", EnumCaseInfo{}, false, nil
		}
	}
	caseInfo, ok := info.CaseMap[pattern.CaseName]
	if !ok {
		return "", EnumCaseInfo{}, true, fmt.Errorf(
			"%s: unknown enum case '%s' for '%s'",
			frontend.FormatPos(pattern.At),
			pattern.CaseName,
			displayTypeName(typeName, module),
		)
	}
	pattern.EnumType = typeName
	pattern.EnumOrdinal = caseInfo.Ordinal
	pattern.PayloadSlots = append(pattern.PayloadSlots[:0], caseInfo.PayloadSlots...)
	return typeName, caseInfo, true, nil
}

func resolveEnumCaseConstructorCall(
	e *frontend.CallExpr,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (string, EnumCaseInfo, bool, error) {
	parts := strings.Split(e.Name, ".")
	if len(parts) < 2 {
		return "", EnumCaseInfo{}, false, nil
	}
	caseName := parts[len(parts)-1]
	typeRef := frontend.TypeRef{
		At:   e.At,
		Kind: frontend.TypeRefNamed,
		Name: strings.Join(parts[:len(parts)-1], "."),
	}
	typeName, err := resolveTypeName(&typeRef, module, imports)
	if err != nil {
		return "", EnumCaseInfo{}, false, err
	}
	info, ok := types[typeName]
	if !ok || info.Kind != TypeEnum {
		shortName := strings.Join(parts[:len(parts)-1], ".")
		if altName, altInfo, found := findUniqueEnumByShortName(shortName, types); found {
			typeName = altName
			info = altInfo
		} else {
			return "", EnumCaseInfo{}, false, nil
		}
	}
	caseInfo, ok := info.CaseMap[caseName]
	if !ok {
		return "", EnumCaseInfo{}, true, fmt.Errorf(
			"%s: unknown enum case '%s' for '%s'",
			frontend.FormatPos(e.At),
			caseName,
			displayTypeName(typeName, module),
		)
	}
	return typeName, caseInfo, true, nil
}

func findUniqueEnumByShortName(
	shortName string,
	types map[string]*TypeInfo,
) (string, *TypeInfo, bool) {
	var foundName string
	var foundInfo *TypeInfo
	for name, info := range types {
		if info == nil || info.Kind != TypeEnum {
			continue
		}
		if name != shortName && !strings.HasSuffix(name, "."+shortName) {
			continue
		}
		if foundInfo != nil && foundName != name {
			return "", nil, false
		}
		foundName = name
		foundInfo = info
	}
	return foundName, foundInfo, foundInfo != nil
}

func displayTypeName(name, module string) string {
	prefix := module + "."
	if module != "" && strings.HasPrefix(name, prefix) {
		return strings.TrimPrefix(name, prefix)
	}
	return name
}

func symbolBelongsToModule(name, module string) bool {
	if module == "" {
		return !strings.Contains(name, ".")
	}
	return name == module || strings.HasPrefix(name, module+".")
}

func ensureFuncVisible(name string, sig FuncSig, module string, pos frontend.Position) error {
	if symbolBelongsToModule(name, module) || sig.Public || strings.HasPrefix(name, "core.") {
		return nil
	}
	return fmt.Errorf(
		"%s: private function '%s' is not visible from module '%s'",
		frontend.FormatPos(pos),
		name,
		module,
	)
}

func ensureTypeVisible(name string, info *TypeInfo, module string, pos frontend.Position) error {
	if info == nil || symbolBelongsToModule(name, module) || info.Public {
		return nil
	}
	return fmt.Errorf(
		"%s: private type '%s' is not visible from module '%s'",
		frontend.FormatPos(pos),
		name,
		module,
	)
}

func resolveCallName(
	name string,
	module string,
	imports map[string]string,
	pos frontend.Position,
) (string, error) {
	parts := strings.Split(name, ".")
	if len(parts) == 1 {
		if target, ok := imports[name]; ok {
			if symbol, isSymbol := importSymbolTarget(target); isSymbol {
				return symbol, nil
			}
		}
		return qualifyName(module, name), nil
	}
	if target, ok := imports[parts[0]]; ok {
		if _, isSymbol := importSymbolTarget(target); isSymbol {
			return "", fmt.Errorf(
				"%s: selective import '%s' cannot be used as a namespace",
				frontend.FormatPos(pos),
				parts[0],
			)
		}
		if len(parts) < 2 {
			return "", fmt.Errorf("%s: expected '%s.<func>'", frontend.FormatPos(pos), parts[0])
		}
		suffix := strings.Join(parts[1:], ".")
		if suffix == "" {
			return "", fmt.Errorf("%s: expected '%s.<func>'", frontend.FormatPos(pos), parts[0])
		}
		return target + "." + suffix, nil
	}
	modPath := strings.Join(parts[:len(parts)-1], ".")
	return modPath + "." + parts[len(parts)-1], nil
}

func resolveKnownCallName(
	name string,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
	pos frontend.Position,
) (string, error) {
	if _, ok := funcs[name]; ok {
		return name, nil
	}
	resolved, err := resolveCallName(name, module, imports, pos)
	if err != nil {
		return "", err
	}
	if _, ok := funcs[resolved]; ok {
		return resolved, nil
	}
	if module != "" && strings.Contains(name, ".") {
		moduleLocal := qualifyName(module, name)
		if _, ok := funcs[moduleLocal]; ok {
			return moduleLocal, nil
		}
	}
	return resolved, nil
}

type assignTargetInfo struct {
	Name           string
	Mutable        bool
	Const          bool
	TypeName       string
	Offset         int
	Global         bool
	ActorField     bool
	ActorFieldSlot int
}

func resolveAssignTarget(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	types map[string]*TypeInfo,
) (assignTargetInfo, string, error) {
	if idx, ok := expr.(*frontend.IndexExpr); ok {
		baseName, fields, pos, ok := splitFieldPath(idx.Base)
		if !ok {
			return assignTargetInfo{}, "", fmt.Errorf(
				"%s: invalid assignment target",
				frontend.FormatPos(pos),
			)
		}
		baseType := ""
		baseOffset := 0
		mutable := false
		constant := false
		global := false
		if baseInfo, ok := locals[baseName]; ok {
			baseType = baseInfo.TypeName
			baseOffset = baseInfo.Base
			mutable = baseInfo.Mutable
			constant = baseInfo.Const
		} else if globalInfo, ok := globals[baseName]; ok {
			baseType = globalInfo.TypeName
			baseOffset = globalInfo.DataIndex
			mutable = globalInfo.Mutable
			constant = globalInfo.Const
			global = true
		} else {
			return assignTargetInfo{}, "", fmt.Errorf(
				"%s: unknown identifier '%s'",
				frontend.FormatPos(pos),
				baseName,
			)
		}
		if _, err := ensureTypeInfo(baseType, types); err != nil {
			return assignTargetInfo{}, "", err
		}
		if err := rejectCollectionInternalAssignment(baseType, fields, types, pos); err != nil {
			return assignTargetInfo{}, "", err
		}
		baseType, _, _, err := resolveFieldChain(baseType, baseOffset, fields, types, pos)
		if err != nil {
			return assignTargetInfo{}, "", err
		}
		info, err := ensureTypeInfo(baseType, types)
		if err != nil {
			return assignTargetInfo{}, "", err
		}
		if info.Kind == TypeStr {
			return assignTargetInfo{}, "", fmt.Errorf(
				"%s: cannot assign into str",
				frontend.FormatPos(pos),
			)
		}
		if info.Kind != TypeSlice && info.Kind != TypeArray {
			return assignTargetInfo{}, "", fmt.Errorf(
				"%s: cannot index '%s'",
				frontend.FormatPos(pos),
				baseType,
			)
		}
		return assignTargetInfo{
			Name:     baseName,
			Mutable:  mutable,
			Const:    constant,
			TypeName: info.ElemType,
			Global:   global,
		}, info.ElemType, nil
	}

	baseName, fields, pos, ok := splitFieldPath(expr)
	if !ok {
		return assignTargetInfo{}, "", fmt.Errorf(
			"%s: invalid assignment target",
			frontend.FormatPos(pos),
		)
	}
	info, ok := locals[baseName]
	if !ok {
		if globalInfo, ok := globals[baseName]; ok {
			if _, err := ensureTypeInfo(globalInfo.TypeName, types); err != nil {
				return assignTargetInfo{}, "", err
			}
			if err := rejectCollectionInternalAssignment(
				globalInfo.TypeName,
				fields,
				types,
				pos,
			); err != nil {
				return assignTargetInfo{}, "", err
			}
			targetType, _, offset, err := resolveFieldChain(
				globalInfo.TypeName,
				globalInfo.DataIndex,
				fields,
				types,
				pos,
			)
			if err != nil {
				return assignTargetInfo{}, "", err
			}
			return assignTargetInfo{
				Name:     baseName,
				Mutable:  globalInfo.Mutable,
				Const:    globalInfo.Const,
				TypeName: targetType,
				Offset:   offset,
				Global:   true,
			}, targetType, nil
		}
		return assignTargetInfo{}, "", fmt.Errorf(
			"%s: unknown identifier '%s'",
			frontend.FormatPos(pos),
			baseName,
		)
	}
	if _, err := ensureTypeInfo(info.TypeName, types); err != nil {
		return assignTargetInfo{}, "", err
	}
	if err := rejectCollectionInternalAssignment(info.TypeName, fields, types, pos); err != nil {
		return assignTargetInfo{}, "", err
	}
	if info.ActorField {
		if len(fields) > 0 {
			return assignTargetInfo{}, "", fmt.Errorf(
				"%s: '%s' is not a struct",
				frontend.FormatPos(pos),
				info.TypeName,
			)
		}
		return assignTargetInfo{
			Name:           baseName,
			Mutable:        info.Mutable,
			Const:          info.Const,
			TypeName:       info.TypeName,
			ActorField:     true,
			ActorFieldSlot: info.ActorFieldSlot,
		}, info.TypeName, nil
	}
	targetType, _, offset, err := resolveFieldChain(info.TypeName, info.Base, fields, types, pos)
	if err != nil {
		return assignTargetInfo{}, "", err
	}
	return assignTargetInfo{
		Name:     baseName,
		Mutable:  info.Mutable,
		Const:    info.Const,
		TypeName: targetType,
		Offset:   offset,
	}, targetType, nil
}

func rejectCollectionInternalAssignment(
	typeName string,
	fields []string,
	types map[string]*TypeInfo,
	pos frontend.Position,
) error {
	current := typeName
	for _, field := range fields {
		info, ok := types[current]
		if !ok {
			return fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), current)
		}
		if err := rejectRepresentationMetadataAssignment(info, field, pos); err != nil {
			return err
		}
		fieldInfo, ok := info.FieldMap[field]
		if !ok {
			return fmt.Errorf("%s: unknown field '%s'", frontend.FormatPos(pos), field)
		}
		current = fieldInfo.TypeName
	}
	return nil
}

func rejectRepresentationMetadataIndexBaseAssignment(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	types map[string]*TypeInfo,
) error {
	baseName, fields, pos, ok := splitFieldPath(expr)
	if !ok || len(fields) == 0 {
		return nil
	}
	if info, ok := locals[baseName]; ok {
		if _, err := ensureTypeInfo(info.TypeName, types); err != nil {
			return err
		}
		return rejectCollectionInternalAssignment(info.TypeName, fields, types, pos)
	}
	if info, ok := globals[baseName]; ok {
		if _, err := ensureTypeInfo(info.TypeName, types); err != nil {
			return err
		}
		return rejectCollectionInternalAssignment(info.TypeName, fields, types, pos)
	}
	return nil
}

func rejectRepresentationMetadataExprAssignment(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	types map[string]*TypeInfo,
) error {
	switch e := expr.(type) {
	case *frontend.FieldAccessExpr:
		if err := rejectRepresentationMetadataIndexBaseAssignment(e, locals, globals, types); err != nil {
			return err
		}
		return rejectRepresentationMetadataExprAssignment(e.Base, locals, globals, types)
	case *frontend.IndexExpr:
		return rejectRepresentationMetadataExprAssignment(e.Base, locals, globals, types)
	default:
		return nil
	}
}

func rejectRepresentationMetadataAssignment(
	info *TypeInfo,
	field string,
	pos frontend.Position,
) error {
	if info == nil {
		return nil
	}
	fieldInfo, fieldKnown := info.FieldMap[field]
	if fieldKnown && fieldInfo.UserAssignable {
		return nil
	}
	if fieldKnown || isReservedRepresentationMetadataField(field) {
		switch info.Kind {
		case TypeArray:
			return fmt.Errorf(
				("%s: cannot assign to fixed-array internals ('ptr'/'len'); " +
					"assign elements via index instead; representation metadata " +
					"field '%s' is not user-assignable in safe code"),
				frontend.FormatPos(pos),
				field,
			)
		case TypeSlice:
			return fmt.Errorf(
				("%s: cannot assign to slice internals ('ptr'/'len'); assign " +
					"elements via index instead; representation metadata field " +
					"'%s' is not user-assignable in safe code"),
				frontend.FormatPos(pos),
				field,
			)
		case TypeStr:
			return fmt.Errorf(
				("%s: cannot assign to string internals ('ptr'/'len'); " +
					"representation metadata field '%s' is not user-assignable " +
					"in safe code"),
				frontend.FormatPos(pos),
				field,
			)
		}
	}
	return nil
}

func ResolveFieldAccessType(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	types map[string]*TypeInfo,
) (assignTargetInfo, string, error) {
	baseName, fields, pos, ok := splitFieldPath(expr)
	if !ok {
		return assignTargetInfo{}, "", fmt.Errorf(
			"%s: invalid field access",
			frontend.FormatPos(pos),
		)
	}
	if info, ok := locals[baseName]; ok {
		if _, err := ensureTypeInfo(info.TypeName, types); err != nil {
			return assignTargetInfo{}, "", err
		}
		targetType, _, offset, err := resolveFieldChain(
			info.TypeName,
			info.Base,
			fields,
			types,
			pos,
		)
		if err != nil {
			return assignTargetInfo{}, "", err
		}
		return assignTargetInfo{
			Name:     baseName,
			Mutable:  info.Mutable,
			Const:    info.Const,
			TypeName: targetType,
			Offset:   offset,
		}, targetType, nil
	}
	if info, ok := globals[baseName]; ok {
		if _, err := ensureTypeInfo(info.TypeName, types); err != nil {
			return assignTargetInfo{}, "", err
		}
		targetType, _, offset, err := resolveFieldChain(
			info.TypeName,
			info.DataIndex,
			fields,
			types,
			pos,
		)
		if err != nil {
			return assignTargetInfo{}, "", err
		}
		return assignTargetInfo{
			Name:     baseName,
			Mutable:  info.Mutable,
			Const:    info.Const,
			TypeName: targetType,
			Offset:   offset,
			Global:   true,
		}, targetType, nil
	}
	return assignTargetInfo{}, "", fmt.Errorf(
		"%s: unknown identifier '%s'",
		frontend.FormatPos(pos),
		baseName,
	)
}

func splitFieldPath(expr frontend.Expr) (string, []string, frontend.Position, bool) {
	return semanticsexpressions.SplitFieldPath(expr)
}

func resolveFieldChain(
	typeName string,
	baseOffset int,
	fields []string,
	types map[string]*TypeInfo,
	pos frontend.Position,
) (string, int, int, error) {
	offset := baseOffset
	current := typeName
	for _, field := range fields {
		info, ok := types[current]
		if !ok {
			return "", 0, 0, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), current)
		}
		if info.Kind != TypeStruct && info.Kind != TypeSlice && info.Kind != TypeArray &&
			info.Kind != TypeStr {
			return "", 0, 0, fmt.Errorf(
				"%s: '%s' is not a struct",
				frontend.FormatPos(pos),
				current,
			)
		}
		fieldInfo, ok := info.FieldMap[field]
		if !ok {
			return "", 0, 0, fmt.Errorf("%s: unknown field '%s'", frontend.FormatPos(pos), field)
		}
		offset += fieldInfo.Offset
		current = fieldInfo.TypeName
	}
	info, ok := types[current]
	if !ok {
		return "", 0, 0, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), current)
	}
	return current, info.SlotCount, offset, nil
}

// ---- slice_views.go ----

func rewriteSliceViewMethodCall(
	e *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	types map[string]*TypeInfo,
) (bool, error) {
	if method, ok := syntheticViewMethodName(e.Name); ok {
		return rewriteSyntheticViewMethodCall(e, method, locals, globals, types)
	}
	receiverParts, method, ok := sliceViewMethodParts(e.Name)
	if !ok {
		return false, nil
	}
	if len(receiverParts) == 0 {
		return false, nil
	}
	root := receiverParts[0]
	if _, ok := locals[root]; !ok {
		if _, ok := globals[root]; !ok {
			return false, nil
		}
	}
	if len(e.TypeArgs) > 0 {
		return true, fmt.Errorf(
			"%s: slice view method '%s' does not accept explicit type arguments",
			frontend.FormatPos(e.At),
			method,
		)
	}
	wantArgs := viewMethodArgCount(method)
	if len(e.Args) != wantArgs {
		return true, fmt.Errorf(
			"%s: slice view method '%s' expects %d argument(s)",
			frontend.FormatPos(e.At),
			method,
			wantArgs,
		)
	}
	receiverType, err := sliceViewReceiverType(receiverParts, locals, globals, types, e.At)
	if err != nil {
		return true, err
	}
	builtin, ok := sliceViewBuiltin(receiverType, method)
	if !ok {
		return true, unsupportedViewReceiverError(e.At, method, receiverType)
	}
	receiver := exprFromPathParts(receiverParts, e.At)
	args := make([]frontend.Expr, 0, len(e.Args)+1)
	args = append(args, receiver)
	args = append(args, e.Args...)
	e.Name = builtin
	e.Args = args
	e.ArgLabels = nil
	return true, nil
}

func syntheticViewMethodName(name string) (string, bool) {
	const prefix = "__method."
	if !strings.HasPrefix(name, prefix) {
		return "", false
	}
	method := strings.TrimPrefix(name, prefix)
	if isViewMethod(method) {
		return method, true
	}
	return "", false
}

func rewriteSyntheticViewMethodCall(
	e *frontend.CallExpr,
	method string,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	types map[string]*TypeInfo,
) (bool, error) {
	if len(e.Args) == 0 {
		return true, fmt.Errorf(
			"%s: view method '%s' is missing receiver",
			frontend.FormatPos(e.At),
			method,
		)
	}
	if len(e.TypeArgs) > 0 {
		return true, fmt.Errorf(
			"%s: view method '%s' does not accept explicit type arguments",
			frontend.FormatPos(e.At),
			method,
		)
	}
	wantArgs := viewMethodArgCount(method)
	if len(e.Args)-1 != wantArgs {
		return true, fmt.Errorf(
			"%s: view method '%s' expects %d argument(s)",
			frontend.FormatPos(e.At),
			method,
			wantArgs,
		)
	}
	receiverType, err := viewReceiverTypeFromExpr(e.Args[0], locals, globals, types, e.At)
	if err != nil {
		return true, err
	}
	builtin, ok := sliceViewBuiltin(receiverType, method)
	if !ok {
		return true, unsupportedViewReceiverError(e.At, method, receiverType)
	}
	e.Name = builtin
	e.ArgLabels = nil
	return true, nil
}

func viewReceiverTypeFromExpr(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	types map[string]*TypeInfo,
	pos frontend.Position,
) (string, error) {
	switch receiver := expr.(type) {
	case *frontend.StringLitExpr:
		return "str", nil
	case *frontend.IdentExpr:
		if local, ok := locals[receiver.Name]; ok {
			return local.TypeName, nil
		}
		if global, ok := globals[receiver.Name]; ok {
			return global.TypeName, nil
		}
		return "", fmt.Errorf(
			"%s: unknown identifier '%s'",
			frontend.FormatPos(receiver.At),
			receiver.Name,
		)
	case *frontend.FieldAccessExpr:
		_, targetType, err := ResolveFieldAccessType(receiver, locals, globals, types)
		return targetType, err
	case *frontend.CallExpr:
		if _, err := rewriteSliceViewMethodCall(receiver, locals, globals, types); err != nil {
			return "", err
		}
		if elem, _, ok := sliceViewElemFromBuiltin(receiver.Name); ok {
			if elem == "str" {
				return "str", nil
			}
			return "[]" + elem, nil
		}
		sigs, err := builtinFuncSigs(types)
		if err != nil {
			return "", err
		}
		if sig, ok := sigs[receiver.Name]; ok {
			return sig.ReturnType, nil
		}
		return "", fmt.Errorf("%s: invalid view method receiver", frontend.FormatPos(pos))
	default:
		return "", fmt.Errorf("%s: invalid view method receiver", frontend.FormatPos(pos))
	}
}

func unsupportedViewReceiverError(pos frontend.Position, method string, receiverType string) error {
	return fmt.Errorf(
		"%s: view method '%s' expects []u8, []u16, []i32, []bool, or String receiver, got '%s'",
		frontend.FormatPos(pos),
		method,
		receiverType,
	)
}

func sliceViewMethodParts(name string) ([]string, string, bool) {
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return nil, "", false
	}
	method := parts[len(parts)-1]
	if isViewMethod(method) {
		return parts[:len(parts)-1], method, true
	}
	return nil, "", false
}

func isViewMethod(method string) bool {
	switch method {
	case "window", "prefix", "suffix", "borrow", "copy", "copy_into":
		return true
	default:
		return false
	}
}

func viewMethodArgCount(method string) int {
	switch method {
	case "window":
		return 2
	case "prefix", "suffix", "copy_into":
		return 1
	default:
		return 0
	}
}

func sliceViewReceiverType(
	parts []string,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	types map[string]*TypeInfo,
	pos frontend.Position,
) (string, error) {
	if len(parts) == 1 {
		if local, ok := locals[parts[0]]; ok {
			return local.TypeName, nil
		}
		if global, ok := globals[parts[0]]; ok {
			return global.TypeName, nil
		}
		return "", fmt.Errorf("%s: unknown identifier '%s'", frontend.FormatPos(pos), parts[0])
	}
	expr := exprFromPathParts(parts, pos)
	field, ok := expr.(*frontend.FieldAccessExpr)
	if !ok {
		return "", fmt.Errorf("%s: invalid slice view receiver", frontend.FormatPos(pos))
	}
	_, targetType, err := ResolveFieldAccessType(field, locals, globals, types)
	return targetType, err
}

func exprFromPathParts(parts []string, pos frontend.Position) frontend.Expr {
	if len(parts) == 0 {
		return &frontend.IdentExpr{At: pos, Name: ""}
	}
	expr := frontend.Expr(&frontend.IdentExpr{At: pos, Name: parts[0]})
	for _, field := range parts[1:] {
		expr = &frontend.FieldAccessExpr{At: pos, Base: expr, Field: field}
	}
	return expr
}

func sliceViewBuiltin(typeName, method string) (string, bool) {
	suffix := ""
	switch typeName {
	case "[]u8":
		suffix = "u8"
	case "[]u16":
		suffix = "u16"
	case "[]i32":
		suffix = "i32"
	case "[]bool":
		suffix = "bool"
	case "str", "String":
		return "core.string_" + method, true
	default:
		return "", false
	}
	return "core.slice_" + method + "_" + suffix, true
}

func sliceViewElemFromBuiltin(name string) (elem string, method string, ok bool) {
	if !strings.HasPrefix(name, "core.slice_") {
		if strings.HasPrefix(name, "core.string_") {
			method := strings.TrimPrefix(name, "core.string_")
			switch method {
			case "window", "prefix", "suffix", "borrow", "copy", "copy_into":
				return "str", method, true
			}
		}
		return "", "", false
	}
	rest := strings.TrimPrefix(name, "core.slice_")
	for _, candidate := range []string{"window", "prefix", "suffix", "borrow", "copy", "copy_into"} {
		prefix := candidate + "_"
		if strings.HasPrefix(rest, prefix) {
			elem = strings.TrimPrefix(rest, prefix)
			switch elem {
			case "u8", "u16", "i32", "bool":
				return elem, candidate, true
			}
			return "", "", false
		}
	}
	return "", "", false
}

// ---- surface_lifetime.go ----

const (
	surfaceSurfaceTypeName     = "lib.core.surface.Surface"
	surfaceFrameTypeName       = "lib.core.surface.Frame"
	surfaceEventTypeName       = "lib.core.surface.Event"
	surfaceDrawContextTypeName = "lib.core.draw.DrawContext"
)

func surfaceEphemeralValueType(typeName string, types map[string]*TypeInfo) (string, bool) {
	return surfaceEphemeralValueTypeVisiting(typeName, types, map[string]bool{})
}

func surfaceEphemeralValueTypeVisiting(
	typeName string,
	types map[string]*TypeInfo,
	visiting map[string]bool,
) (string, bool) {
	switch typeName {
	case surfaceFrameTypeName, surfaceEventTypeName, surfaceDrawContextTypeName:
		return typeName, true
	}
	if visiting[typeName] {
		return "", false
	}
	info, ok := types[typeName]
	if !ok {
		return "", false
	}
	visiting[typeName] = true
	defer delete(visiting, typeName)

	switch info.Kind {
	case TypeStruct:
		for _, field := range info.Fields {
			if surfaceType, ok := surfaceEphemeralValueTypeVisiting(field.TypeName, types, visiting); ok {
				return surfaceType, true
			}
		}
	case TypeEnum:
		for _, c := range info.EnumCases {
			for _, payload := range c.PayloadTypes {
				if surfaceType, ok := surfaceEphemeralValueTypeVisiting(payload, types, visiting); ok {
					return surfaceType, true
				}
			}
		}
	case TypeArray, TypeOptional, TypeSlice:
		return surfaceEphemeralValueTypeVisiting(info.ElemType, types, visiting)
	}
	return "", false
}

func surfaceActorTaskBoundaryValueType(typeName string, types map[string]*TypeInfo) (string, bool) {
	return surfaceActorTaskBoundaryValueTypeVisiting(typeName, types, map[string]bool{})
}

func surfaceAggregateFieldStorageAllowed(
	containerType string,
	fieldName string,
	fieldType string,
) bool {
	return containerType == surfaceDrawContextTypeName &&
		fieldName == "frame" &&
		fieldType == surfaceFrameTypeName
}

func surfaceActorTaskBoundaryValueTypeVisiting(
	typeName string,
	types map[string]*TypeInfo,
	visiting map[string]bool,
) (string, bool) {
	switch typeName {
	case surfaceSurfaceTypeName,
		surfaceFrameTypeName,
		surfaceEventTypeName,
		surfaceDrawContextTypeName:
		return typeName, true
	}
	if visiting[typeName] {
		return "", false
	}
	info, ok := types[typeName]
	if !ok {
		return "", false
	}
	visiting[typeName] = true
	defer delete(visiting, typeName)

	switch info.Kind {
	case TypeStruct:
		for _, field := range info.Fields {
			if surfaceType, ok := surfaceActorTaskBoundaryValueTypeVisiting(
				field.TypeName,
				types,
				visiting,
			); ok {
				return surfaceType, true
			}
		}
	case TypeEnum:
		for _, c := range info.EnumCases {
			for _, payload := range c.PayloadTypes {
				if surfaceType, ok := surfaceActorTaskBoundaryValueTypeVisiting(payload, types, visiting); ok {
					return surfaceType, true
				}
			}
		}
	case TypeArray, TypeOptional, TypeSlice:
		return surfaceActorTaskBoundaryValueTypeVisiting(info.ElemType, types, visiting)
	}
	return "", false
}

func surfaceEphemeralReturnAllowed(analysis *functionAnalysisState, surfaceType string) bool {
	if analysis == nil {
		return false
	}
	switch analysis.currentFuncName {
	case "lib.core.surface.begin_frame":
		return surfaceType == surfaceFrameTypeName
	case "lib.core.surface.poll_event":
		return surfaceType == surfaceEventTypeName
	default:
		return false
	}
}

func surfaceFramePixelsEscapeExpr(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	types map[string]*TypeInfo,
	analysis *functionAnalysisState,
) bool {
	_, ok := surfaceFramePixelsSourceExpr(expr, locals, globals, types, analysis)
	return ok
}

func surfaceFramePixelsSourceExpr(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	types map[string]*TypeInfo,
	analysis *functionAnalysisState,
) (string, bool) {
	switch e := expr.(type) {
	case *frontend.FieldAccessExpr:
		if e.Field != "pixels" {
			return "", false
		}
		_, baseType, err := ResolveFieldAccessType(e.Base, locals, globals, types)
		if err != nil || baseType != surfaceFrameTypeName {
			return "", false
		}
		if path, ok := canonicalOwnershipAccessPath(e.Base); ok {
			return path, true
		}
		return "", true
	case *frontend.IdentExpr:
		if source, ok := analysis.localSurfaceFramePixelsSource(e.Name); ok {
			return source, true
		}
		if local, ok := locals[e.Name]; ok && local.SurfaceFramePixelsSource != "" {
			return local.SurfaceFramePixelsSource, true
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if source, ok := surfaceFramePixelsSourceExpr(
				field.Value,
				locals,
				globals,
				types,
				analysis,
			); ok {
				return source, true
			}
		}
	}
	return "", false
}

func surfaceFrameOwnerSourceExpr(
	expr frontend.Expr,
	analysis *functionAnalysisState,
) (string, bool) {
	if call, ok := expr.(*frontend.CallExpr); ok && call.Name == "lib.core.surface.begin_frame" &&
		len(call.Args) > 0 {
		return canonicalOwnershipAccessPath(call.Args[0])
	}
	if owner, ok := surfaceManualFrameOwnerSourceExpr(expr); ok {
		return owner, true
	}
	if path, ok := canonicalOwnershipAccessPath(expr); ok {
		return analysis.localSurfaceFrameOwner(path)
	}
	return "", false
}

func surfaceManualFrameOwnerSourceExpr(expr frontend.Expr) (string, bool) {
	var surfaceExpr frontend.Expr
	switch e := expr.(type) {
	case *frontend.StructLitExpr:
		if e.Type.Name != surfaceFrameTypeName {
			return "", false
		}
		for _, field := range e.Fields {
			if field.Name == "surface" {
				surfaceExpr = field.Value
				break
			}
		}
	case *frontend.CallExpr:
		if e.ResolvedType != surfaceFrameTypeName && e.Name != surfaceFrameTypeName {
			return "", false
		}
		for i, label := range e.ArgLabels {
			if label == "surface" && i < len(e.Args) {
				surfaceExpr = e.Args[i]
				break
			}
		}
		if surfaceExpr == nil && len(e.Args) > 0 {
			surfaceExpr = e.Args[0]
		}
	default:
		return "", false
	}
	if surfaceExpr == nil {
		return "", false
	}
	if owner, ok := canonicalOwnershipAccessPath(surfaceExpr); ok {
		return owner, true
	}
	return surfaceConstructedHandleOwnerPathExpr(surfaceExpr)
}

func surfaceHandleOwnerPathExpr(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	types map[string]*TypeInfo,
) (string, bool) {
	return surfaceHandleOwnerPathExprWithAnalysis(expr, locals, globals, types, nil)
}

func surfaceHandleOwnerPathExprWithAnalysis(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	types map[string]*TypeInfo,
	analysis *functionAnalysisState,
) (string, bool) {
	if id, ok := expr.(*frontend.IdentExpr); ok {
		return analysis.localSurfaceHandleOwner(id.Name)
	}
	field, ok := expr.(*frontend.FieldAccessExpr)
	if !ok || field.Field != "handle" {
		return "", false
	}
	_, baseType, err := ResolveFieldAccessType(field.Base, locals, globals, types)
	if err != nil || baseType != surfaceSurfaceTypeName {
		return "", false
	}
	return canonicalOwnershipAccessPath(field.Base)
}

func surfaceHostABIHandleArgIndex(name string) (int, bool) {
	switch name {
	case "core.surface_close",
		"core.surface_poll_event_kind",
		"core.surface_poll_event_x",
		"core.surface_poll_event_y",
		"core.surface_poll_event_button",
		"core.surface_poll_event_into",
		"core.surface_poll_event_text_len",
		"core.surface_poll_event_text_into",
		"core.surface_clipboard_write_text",
		"core.surface_clipboard_read_text_into",
		"core.surface_poll_composition_into",
		"core.surface_begin_frame",
		"core.surface_present_rgba",
		"core.surface_request_redraw":
		return 0, true
	default:
		return 0, false
	}
}

func surfaceConstructedHandleOwnerPathExpr(expr frontend.Expr) (string, bool) {
	var handle frontend.Expr
	switch e := expr.(type) {
	case *frontend.StructLitExpr:
		if e.Type.Name != surfaceSurfaceTypeName {
			return "", false
		}
		for _, field := range e.Fields {
			if field.Name == "handle" {
				handle = field.Value
				break
			}
		}
	case *frontend.CallExpr:
		if e.ResolvedType != surfaceSurfaceTypeName && e.Name != surfaceSurfaceTypeName {
			return "", false
		}
		for i, label := range e.ArgLabels {
			if label == "handle" && i < len(e.Args) {
				handle = e.Args[i]
				break
			}
		}
		if handle == nil && len(e.Args) > 0 {
			handle = e.Args[0]
		}
	default:
		return "", false
	}
	if handle == nil {
		return "", false
	}
	field, ok := handle.(*frontend.FieldAccessExpr)
	if !ok || field.Field != "handle" {
		return "", false
	}
	return canonicalOwnershipAccessPath(field.Base)
}

func bindSurfaceFrameOwnerForLocal(
	name string,
	typeName string,
	expr frontend.Expr,
	analysis *functionAnalysisState,
) {
	if analysis == nil || name == "" {
		return
	}
	switch typeName {
	case surfaceFrameTypeName:
		if owner, ok := surfaceFrameOwnerSourceExpr(expr, analysis); ok {
			analysis.setLocalSurfaceFrameOwner(name, owner)
		} else {
			analysis.setLocalSurfaceFrameOwner(name, "")
		}
	case surfaceDrawContextTypeName:
		if owner, ok := surfaceDrawContextFrameOwnerSourceExpr(expr, analysis); ok {
			analysis.setLocalSurfaceFrameOwner(resourceFieldPath(name, "frame"), owner)
		} else {
			analysis.setLocalSurfaceFrameOwner(resourceFieldPath(name, "frame"), "")
		}
	default:
		analysis.setLocalSurfaceFrameOwner(name, "")
		analysis.setLocalSurfaceFrameOwner(resourceFieldPath(name, "frame"), "")
	}
}

func surfaceDrawContextFrameOwnerSourceExpr(
	expr frontend.Expr,
	analysis *functionAnalysisState,
) (string, bool) {
	switch e := expr.(type) {
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if field.Name == "frame" {
				return surfaceFrameOwnerSourceExpr(field.Value, analysis)
			}
		}
	case *frontend.CallExpr:
		if e.Name == surfaceDrawContextTypeName && len(e.Args) > 0 {
			return surfaceFrameOwnerSourceExpr(e.Args[0], analysis)
		}
	default:
		if path, ok := canonicalOwnershipAccessPath(expr); ok {
			return analysis.localSurfaceFrameOwner(resourceFieldPath(path, "frame"))
		}
	}
	return "", false
}

func surfacePresentedFrameArg(expr frontend.Expr) (string, bool) {
	return canonicalOwnershipAccessPath(expr)
}

func checkSurfacePresentFrameOwner(
	expr frontend.Expr,
	analysis *functionAnalysisState,
	state *regionState,
	pos frontend.Position,
) error {
	frameName, ok := surfacePresentedFrameArg(expr)
	if !ok {
		return nil
	}
	return checkSurfacePresentFrameOwnerPath(frameName, analysis, state, pos)
}

func checkSurfacePresentFrameOwnerPath(
	frameName string,
	analysis *functionAnalysisState,
	state *regionState,
	pos frontend.Position,
) error {
	if frameName == "" {
		return nil
	}
	owner, ok := analysis.localSurfaceFrameOwner(frameName)
	if !ok || owner == "" {
		return nil
	}
	return state.checkNotConsumed(owner, pos)
}

func isSurfacePresentCallName(name string) bool {
	return name == "lib.core.surface.present"
}

// ---- types.go ----

type CheckedProgram = model.CheckedProgram
type CheckedFunc = model.CheckedFunc
type ActorStateField = model.ActorStateField
type LocalInfo = model.LocalInfo
type FunctionFieldInfo = model.FunctionFieldInfo
type CallableEscapeKind = model.CallableEscapeKind
type GlobalInfo = model.GlobalInfo
type GlobalArrayBackingInfo = model.GlobalArrayBackingInfo
type FuncSig = model.FuncSig
type ResourceProvenance = model.ResourceProvenance
type ReturnRegionSummary = model.ReturnRegionSummary
type ReturnResourceSummary = model.ReturnResourceSummary
type CheckedStruct = model.CheckedStruct
type CheckedEnum = model.CheckedEnum
type CheckedProtocol = model.CheckedProtocol
type CheckedUIState = model.CheckedUIState
type CheckedUIView = model.CheckedUIView
type TypeKind = model.TypeKind
type FieldInfo = model.FieldInfo
type TypeInfo = model.TypeInfo
type EnumCaseInfo = model.EnumCaseInfo

const (
	FnPtrEnvSlotCount       = model.FnPtrEnvSlotCount
	FnPtrSlotCount          = model.FnPtrSlotCount
	CallableHandleSlotCount = model.CallableHandleSlotCount

	CallableEscapeLocalSnapshot = model.CallableEscapeLocalSnapshot
	CallableEscapeHeap          = model.CallableEscapeHeap
	CallableEscapeGlobal        = model.CallableEscapeGlobal
	CallableEscapeThread        = model.CallableEscapeThread

	TypeI32      = model.TypeI32
	TypeI64      = model.TypeI64
	TypeU8       = model.TypeU8
	TypeBool     = model.TypeBool
	TypePtr      = model.TypePtr
	TypeSlice    = model.TypeSlice
	TypeStr      = model.TypeStr
	TypeStruct   = model.TypeStruct
	TypeArray    = model.TypeArray
	TypeIsland   = model.TypeIsland
	TypeCap      = model.TypeCap
	TypeActor    = model.TypeActor
	TypeEnum     = model.TypeEnum
	TypeOptional = model.TypeOptional

	MaxActorStateSlots = model.MaxActorStateSlots
)

func makeSliceTypeInfo(name, elem string) *TypeInfo {
	fieldMap := map[string]FieldInfo{
		"ptr": {Name: "ptr", TypeName: "ptr", Offset: 0, SlotCount: 1},
		"len": {Name: "len", TypeName: "i32", Offset: 1, SlotCount: 1},
	}
	fields := []FieldInfo{fieldMap["ptr"], fieldMap["len"]}
	return &TypeInfo{
		Name:      name,
		Kind:      TypeSlice,
		Public:    true,
		Fields:    fields,
		FieldMap:  fieldMap,
		SlotCount: 2,
		ElemType:  elem,
	}
}

func makeArrayTypeInfo(name, elem string, n int) *TypeInfo {
	fieldMap := map[string]FieldInfo{
		"ptr": {Name: "ptr", TypeName: "ptr", Offset: 0, SlotCount: 1},
		"len": {Name: "len", TypeName: "i32", Offset: 1, SlotCount: 1},
	}
	fields := []FieldInfo{fieldMap["ptr"], fieldMap["len"]}
	return &TypeInfo{
		Name:      name,
		Kind:      TypeArray,
		Public:    true,
		Fields:    fields,
		FieldMap:  fieldMap,
		SlotCount: 2,
		ElemType:  elem,
		ArrayLen:  n,
	}
}

func makeStrTypeInfo() *TypeInfo {
	info := makeSliceTypeInfo("str", "u8")
	info.Kind = TypeStr
	return info
}

func makeStructTypeInfo(name string, fields []FieldInfo) *TypeInfo {
	fieldMap := make(map[string]FieldInfo, len(fields))
	offset := 0
	structFields := make([]FieldInfo, 0, len(fields))
	for _, field := range fields {
		slotCount := field.SlotCount
		if slotCount <= 0 {
			slotCount = 1
		}
		resolved := FieldInfo{
			Name:           field.Name,
			TypeName:       field.TypeName,
			Offset:         offset,
			SlotCount:      slotCount,
			UserAssignable: true,
		}
		offset += slotCount
		structFields = append(structFields, resolved)
		fieldMap[resolved.Name] = resolved
	}
	return &TypeInfo{
		Name:      name,
		Kind:      TypeStruct,
		Public:    true,
		Repr:      frontend.StructReprDefault,
		Fields:    structFields,
		FieldMap:  fieldMap,
		SlotCount: offset,
	}
}

func makeBuiltinEnumTypeInfo(name string, caseNames []string) *TypeInfo {
	caseMap := make(map[string]EnumCaseInfo, len(caseNames))
	cases := make([]EnumCaseInfo, 0, len(caseNames))
	for i, caseName := range caseNames {
		info := EnumCaseInfo{Name: caseName, Ordinal: int32(i), SlotCount: 0}
		caseMap[caseName] = info
		cases = append(cases, info)
	}
	return &TypeInfo{
		Name:      name,
		Kind:      TypeEnum,
		Public:    true,
		SlotCount: 1,
		EnumCases: cases,
		CaseMap:   caseMap,
	}
}

func baseTypes() map[string]*TypeInfo {
	actorRefSlots := runtimeabi.ActorHandleABI().RefSlots
	types := map[string]*TypeInfo{
		"i32":          {Name: "i32", Kind: TypeI32, SlotCount: 1},
		"i64":          {Name: "i64", Kind: TypeI64, SlotCount: 1, Public: true},
		"u8":           {Name: "u8", Kind: TypeU8, SlotCount: 1, Public: true},
		"u16":          {Name: "u16", Kind: TypeU8, SlotCount: 1, Public: true},
		"c_int":        {Name: "c_int", Kind: TypeI32, SlotCount: 1, Public: true},
		"c_uint":       {Name: "c_uint", Kind: TypeI32, SlotCount: 1, Public: true},
		"bool":         {Name: "bool", Kind: TypeBool, SlotCount: 1, Public: true},
		"ptr":          {Name: "ptr", Kind: TypePtr, SlotCount: 1, Public: true},
		"fnptr":        {Name: "fnptr", Kind: TypePtr, SlotCount: FnPtrSlotCount, Public: true},
		"str":          makeStrTypeInfo(),
		"actor":        {Name: "actor", Kind: TypeActor, SlotCount: actorRefSlots, Public: true},
		"actor.status": makeBuiltinEnumTypeInfo("actor.status", runtimeabi.ActorLifecycleStatusNames()),
		"actor.exit_reason": {
			Name:      "actor.exit_reason",
			Kind:      TypeI32,
			SlotCount: 1,
			Public:    true,
		},
		"actor.monitor": {
			Name:              "actor.monitor",
			Kind:              TypeI32,
			SlotCount:         1,
			Public:            true,
			RuntimeOwned:      true,
			UserConstructible: false,
			UserAssignable:    false,
			ActorSendable:     false,
		},
		"actor.spawn_options": {Name: "actor.spawn_options", Kind: TypeI32, SlotCount: 1, Public: true},
		"task.error":          {Name: "task.error", Kind: TypeI32, SlotCount: 1, Public: true},
		"task.group":          {Name: "task.group", Kind: TypeI32, SlotCount: 1, Public: true},
		"island":              {Name: "island", Kind: TypeIsland, SlotCount: 1, Public: true},
		"cap.io":              {Name: "cap.io", Kind: TypeCap, SlotCount: 1, Public: true},
		"cap.mem":             {Name: "cap.mem", Kind: TypeCap, SlotCount: 1, Public: true},
		"consent.token":       {Name: "consent.token", Kind: TypeCap, SlotCount: 1, Public: true},
		"secret.i32":          {Name: "secret.i32", Kind: TypeStruct, SlotCount: 1, Public: true},
	}
	types["i32"].Public = true
	types["task.i32"] = makeStructTypeInfo("task.i32", []FieldInfo{
		{Name: "value", TypeName: "i32"},
		{Name: "error", TypeName: "task.error"},
	})
	types["task.result_i32"] = makeStructTypeInfo("task.result_i32", []FieldInfo{
		{Name: "value", TypeName: "i32"},
		{Name: "error", TypeName: "task.error"},
	})
	types["actor.msg"] = makeStructTypeInfo("actor.msg", []FieldInfo{
		{Name: "value", TypeName: "i32"},
		{Name: "tag", TypeName: "i32"},
	})
	types["actor.recv_result_i32"] = makeStructTypeInfo("actor.recv_result_i32", []FieldInfo{
		{Name: "value", TypeName: "i32"},
		{Name: "error", TypeName: "task.error"},
	})
	types["actor.recv_msg_result"] = makeStructTypeInfo("actor.recv_msg_result", []FieldInfo{
		{Name: "value", TypeName: "i32"},
		{Name: "tag", TypeName: "i32"},
		{Name: "error", TypeName: "task.error"},
	})
	types["actor.wait_result"] = makeStructTypeInfo("actor.wait_result", []FieldInfo{
		{Name: "reason", TypeName: "actor.exit_reason"},
		{Name: "status", TypeName: "actor.status"},
	})
	types["actor.status_result_raw"] = makeStructTypeInfo("actor.status_result_raw", []FieldInfo{
		{Name: "status_code", TypeName: "i32"},
		{Name: "result", TypeName: "i32"},
	})
	types["actor.status_result_raw"].RuntimeOwned = true
	types["actor.status_result_raw"].UserConstructible = false
	types["actor.status_result_raw"].UserAssignable = false
	types["actor.status_result_raw"].ActorSendable = false
	types["actor.exit"] = makeStructTypeInfo("actor.exit", []FieldInfo{
		{Name: "target", TypeName: "actor", SlotCount: actorRefSlots},
		{Name: "reason", TypeName: "actor.exit_reason"},
	})
	types["actor.node"] = makeStructTypeInfo("actor.node", []FieldInfo{
		{Name: "id", TypeName: "i32"},
		{Name: "epoch", TypeName: "i32"},
	})
	types["actor.node"].RuntimeOwned = true
	types["actor.node"].UserConstructible = false
	types["actor.node"].UserAssignable = false
	types["actor.node"].ActorSendable = false
	types["actor.system_recv_raw"] = makeStructTypeInfo("actor.system_recv_raw", []FieldInfo{
		{Name: "status", TypeName: "i32"},
		{Name: "kind", TypeName: "i32"},
		{Name: "subject", TypeName: "actor", SlotCount: 1},
		{Name: "monitor", TypeName: "actor.monitor"},
		{Name: "node", TypeName: "actor.node", SlotCount: 2},
		{Name: "reason_kind", TypeName: "i32"},
		{Name: "reason_code", TypeName: "i32"},
	})
	types["actor.system_recv_raw"].RuntimeOwned = true
	types["actor.system_recv_raw"].UserConstructible = false
	types["actor.system_recv_raw"].UserAssignable = false
	types["actor.system_recv_raw"].ActorSendable = false
	for _, name := range []string{
		"task.i32",
		"task.result_i32",
		"actor.msg",
		"actor.recv_result_i32",
		"actor.recv_msg_result",
		"actor.wait_result",
		"actor.status_result_raw",
		"actor.exit",
		"actor.node",
		"actor.system_recv_raw",
	} {
		types[name].Repr = frontend.StructReprC
	}
	return types
}

func addILP32NativeScalarTypes(types map[string]*TypeInfo) {
	for _, name := range []string{
		"usize",
		"isize",
		"size_t",
		"ssize_t",
		"native_int",
		"native_uint",
		"c_long",
		"c_ulong",
	} {
		types[name] = &TypeInfo{Name: name, Kind: TypeI32, SlotCount: 1, Public: true}
	}
	types["rawptr"] = &TypeInfo{Name: "rawptr", Kind: TypePtr, SlotCount: 1, Public: true}
	types["nullable_ptr"] = &TypeInfo{
		Name:      "nullable_ptr",
		Kind:      TypePtr,
		SlotCount: 1,
		Public:    true,
	}
	types["ref"] = &TypeInfo{Name: "ref", Kind: TypePtr, SlotCount: 1, Public: true}
}

func IsILP32NativeScalarType(name string) bool {
	switch strings.TrimSpace(name) {
	case "usize", "isize", "size_t", "ssize_t", "native_int", "native_uint", "c_long", "c_ulong":
		return true
	default:
		return false
	}
}

func IsILP32UnsignedNativeScalarType(name string) bool {
	switch strings.TrimSpace(name) {
	case "usize", "size_t", "native_uint", "c_ulong":
		return true
	default:
		return false
	}
}

func TypedTaskHandleTypeName(errorType string, types map[string]*TypeInfo) string {
	if info, ok := types[errorType]; ok && info.SlotCount == 1 {
		return "task.i32"
	}
	return "task.i32.throws." + errorType
}

func IsTypedTaskHandleTypeName(typeName string) bool {
	return strings.HasPrefix(typeName, "task.i32.throws.")
}

func TypedTaskHandleTypesCompatible(expected, actual string) bool {
	if expected == "task.i32" && IsTypedTaskHandleTypeName(actual) {
		return true
	}
	if IsTypedTaskHandleTypeName(expected) && actual == "task.i32" {
		return true
	}
	return false
}

func EnsureTypedTaskHandleType(
	errorType string,
	types map[string]*TypeInfo,
) (string, *TypeInfo, error) {
	errorInfo, ok := types[errorType]
	if !ok {
		return "", nil, fmt.Errorf("unknown type '%s'", errorType)
	}
	if errorInfo.Kind != TypeEnum {
		return "", nil, fmt.Errorf("typed task error argument must be an enum")
	}
	if errorInfo.SlotCount == 1 {
		info, ok := types["task.i32"]
		if !ok {
			return "", nil, fmt.Errorf("unknown type 'task.i32'")
		}
		return "task.i32", info, nil
	}
	handleSlots := errorInfo.SlotCount + 2
	if handleSlots > 8 {
		return "", nil, fmt.Errorf(
			"typed task supports at most 8 slots, got %d for error type '%s'",
			handleSlots,
			errorType,
		)
	}
	name := TypedTaskHandleTypeName(errorType, types)
	if info, ok := types[name]; ok {
		return name, info, nil
	}
	info := makeStructTypeInfo(name, []FieldInfo{
		{Name: "value", TypeName: "i32"},
		{Name: "error", TypeName: errorType, SlotCount: errorInfo.SlotCount},
		{Name: "status", TypeName: "task.error"},
	})
	info.Public = true
	types[name] = info
	return name, info, nil
}

func ensureTypeInfo(name string, types map[string]*TypeInfo) (*TypeInfo, error) {
	if info, ok := types[name]; ok {
		return info, nil
	}
	if elem, ok := optionalElemName(name); ok {
		elemInfo, err := ensureTypeInfo(elem, types)
		if err != nil {
			return nil, err
		}
		info := &TypeInfo{
			Name:      name,
			Kind:      TypeOptional,
			Public:    true,
			SlotCount: elemInfo.SlotCount + 1,
			ElemType:  elem,
		}
		types[name] = info
		return info, nil
	}
	if elem, ok := sliceElemName(name); ok {
		elemInfo, ok := types[elem]
		if !ok || !isSupportedCollectionElemType(elemInfo) {
			return nil, fmt.Errorf("slice element type '%s' is not supported", elem)
		}
		info := makeSliceTypeInfo(name, elem)
		types[name] = info
		return info, nil
	}
	if n, elem, ok := parseArrayTypeName(name); ok {
		if n <= 0 {
			return nil, fmt.Errorf("array size must be positive constant")
		}
		elemInfo, ok := types[elem]
		if !ok || !isSupportedCollectionElemType(elemInfo) {
			return nil, fmt.Errorf("array element type '%s' is not supported", elem)
		}
		info := makeArrayTypeInfo(name, elem, n)
		types[name] = info
		return info, nil
	}
	if isArrayTypeName(name) {
		return nil, fmt.Errorf("invalid array type '%s'", name)
	}
	if isTargetLayoutOnlyScalar(name) {
		return nil, targetLayoutOnlyScalarError(name)
	}
	return nil, fmt.Errorf("unknown type '%s'", name)
}

func targetLayoutOnlyScalarError(name string) error {
	return fmt.Errorf(
		("target-layout scalar type '%s' is not supported in " +
			"source-level Tetra yet; it is reserved for compiler target " +
			"layout/ABI classifiers until native-int/codegen support is " +
			"implemented"),
		name,
	)
}

func isTargetLayoutOnlyScalar(name string) bool {
	switch strings.TrimSpace(name) {
	case "i8", "i16", "u32", "u64", "uint",
		"usize", "isize", "size_t", "ssize_t", "native_int", "native_uint",
		"c_long", "c_ulong", "f32", "f64", "ref", "nullable_ptr", "rawptr":
		return true
	default:
		return false
	}
}

func typesCompatible(expected, actual string) bool {
	if expected == actual {
		return true
	}
	if TypedTaskHandleTypesCompatible(expected, actual) {
		return true
	}
	if expected == "none" || actual == "none" {
		if _, ok := optionalElemName(expected); ok && actual == "none" {
			return true
		}
		if _, ok := optionalElemName(actual); ok && expected == "none" {
			return true
		}
		return false
	}
	if elem, ok := optionalElemName(expected); ok && typesCompatible(elem, actual) {
		return true
	}
	if isInt32Like(expected) && isInt32Like(actual) {
		return true
	}
	return false
}

func typesCompatibleWithNullPtr(expected, actual string, expr frontend.Expr) bool {
	if !smallIntLiteralFits(expected, actual, expr) {
		return false
	}
	if typesCompatible(expected, actual) {
		return true
	}
	if expected == "ptr" && actual == "fnptr" {
		_, ok := expr.(*frontend.ClosureExpr)
		return ok
	}
	if isNullablePointerScalarType(expected) && actual == "i32" && isNullPtrLiteral(expr) {
		return true
	}
	return false
}

func isNullablePointerScalarType(name string) bool {
	switch strings.TrimSpace(name) {
	case "ptr", "rawptr", "nullable_ptr":
		return true
	default:
		return false
	}
}

func smallIntLiteralFits(expected, actual string, expr frontend.Expr) bool {
	if actual != "i32" {
		return true
	}
	rangeType := expected
	for {
		elem, ok := optionalElemName(rangeType)
		if !ok {
			break
		}
		rangeType = elem
	}
	if rangeType != "u8" && rangeType != "u16" && rangeType != "c_uint" &&
		!IsILP32UnsignedNativeScalarType(rangeType) {
		return true
	}
	v, ok, overflow := evalConstI32(expr)
	if !ok {
		return true
	}
	if overflow {
		return false
	}
	switch rangeType {
	case "u8":
		return v >= 0 && v <= 255
	case "u16":
		return v >= 0 && v <= 65535
	case "c_uint":
		return v >= 0
	case "usize", "size_t", "native_uint", "c_ulong":
		return v >= 0
	default:
		return true
	}
}

func isNullPtrLiteral(expr frontend.Expr) bool {
	n, ok := expr.(*frontend.NumberExpr)
	return ok && n.Value == 0
}

func constI32(expr frontend.Expr) (int32, bool) {
	v, ok, overflow := evalConstI32(expr)
	if !ok || overflow {
		return 0, false
	}
	return int32(v), true
}

const (
	minConstI32 int64 = -1 << 31
	maxConstI32 int64 = 1<<31 - 1
)

func evalConstI32(expr frontend.Expr) (int64, bool, bool) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return int64(e.Value), true, false
	case *frontend.UnaryExpr:
		if e.Op != frontend.TokenMinus {
			return 0, false, false
		}
		v, ok, overflow := evalConstI32(e.X)
		if !ok || overflow {
			return 0, ok, overflow
		}
		return checkedConstI32(-v)
	case *frontend.BinaryExpr:
		left, ok, overflow := evalConstI32(e.Left)
		if !ok || overflow {
			return 0, ok, overflow
		}
		right, ok, overflow := evalConstI32(e.Right)
		if !ok || overflow {
			return 0, ok, overflow
		}
		switch e.Op {
		case frontend.TokenPlus:
			return checkedConstI32(left + right)
		case frontend.TokenMinus:
			return checkedConstI32(left - right)
		case frontend.TokenStar:
			return checkedConstI32(left * right)
		case frontend.TokenSlash:
			if right == 0 {
				return 0, false, false
			}
			return checkedConstI32(left / right)
		case frontend.TokenPercent:
			if right == 0 {
				return 0, false, false
			}
			return checkedConstI32(left % right)
		default:
			return 0, false, false
		}
	default:
		return 0, false, false
	}
}

func checkedConstI32(v int64) (int64, bool, bool) {
	if v < minConstI32 || v > maxConstI32 {
		return 0, true, true
	}
	return v, true, false
}

func isInt32Like(name string) bool {
	return name == "i32" || name == "u8" || name == "u16" || name == "c_int" || name == "c_uint" ||
		name == "task.error" ||
		name == "actor.exit_reason" ||
		IsILP32NativeScalarType(name)
}

func isConditionType(name string) bool {
	return name == "bool" || isInt32Like(name)
}

func isReservedTypeName(name string) bool {
	switch name {
	case "i32",
		"i64",
		"Int64",
		"u8",
		"u16",
		"c_int",
		"c_uint",
		"bool",
		"Bool",
		"ptr",
		"fnptr",
		"rawptr",
		"nullable_ptr",
		"ref",
		"str",
		"String",
		"usize",
		"isize",
		"size_t",
		"ssize_t",
		"native_int",
		"native_uint",
		"c_long",
		"c_ulong",
		"actor",
		"actor.msg",
		"actor.recv_result_i32",
		"actor.recv_msg_result",
		"task.error",
		"task.group",
		"task.i32",
		"task.result_i32",
		"island",
		"cap.io",
		"cap.mem",
		"consent.token",
		"secret.i32":
		return true
	default:
		return false
	}
}

func optionalElemName(name string) (string, bool) {
	if strings.HasSuffix(name, "?") {
		return strings.TrimSuffix(name, "?"), true
	}
	return "", false
}

func optionalTypeName(elem string) string {
	return elem + "?"
}

func isPrintableType(name string, types map[string]*TypeInfo) bool {
	info, err := ensureTypeInfo(name, types)
	if err != nil {
		return false
	}
	if info.Kind == TypeStr {
		return true
	}
	if info.Kind == TypeSlice && info.ElemType == "u8" {
		return true
	}
	return false
}

func sliceElemName(name string) (string, bool) {
	if strings.HasPrefix(name, "[]") {
		return name[2:], true
	}
	return "", false
}

func isArrayTypeName(name string) bool {
	return strings.HasPrefix(name, "[") && strings.Contains(name, "]")
}

func parseArrayTypeName(name string) (int, string, bool) {
	if !strings.HasPrefix(name, "[") {
		return 0, "", false
	}
	end := strings.Index(name, "]")
	if end <= 1 || end+1 > len(name) {
		return 0, "", false
	}
	n, err := strconv.Atoi(name[1:end])
	if err != nil {
		return 0, "", false
	}
	elem := name[end+1:]
	if elem == "" {
		return 0, "", false
	}
	return n, elem, true
}

func isSupportedArrayElemType(name string, types map[string]*TypeInfo) bool {
	info, ok := types[name]
	if !ok {
		return false
	}
	return isSupportedCollectionElemType(info)
}

func isSupportedCollectionElemType(info *TypeInfo) bool {
	if info == nil {
		return false
	}
	switch info.Name {
	case "i32", "u8", "u16", "c_int", "c_uint", "bool",
		"usize", "isize", "size_t", "ssize_t", "native_int", "native_uint", "c_long", "c_ulong":
		return true
	}
	return info.Kind == TypeStruct && info.Name != "secret.i32" && info.SlotCount == 1
}

func isSupportedActorStateScalarType(name string) bool {
	return name == "i32" || name == "bool" || name == "u8" || name == "u16" || name == "c_int" ||
		name == "c_uint" ||
		name == "task.error" ||
		IsILP32NativeScalarType(name)
}

func funcSigActorTaskTransferSafe(sig FuncSig, types map[string]*TypeInfo) bool {
	return funcSigActorTaskTransferUnsafeReason(sig, types) == ""
}

func funcSigActorTaskTransferUnsafeReason(sig FuncSig, types map[string]*TypeInfo) string {
	for i, typeName := range sig.ParamTypes {
		ownership := ""
		if i < len(sig.ParamOwnership) {
			ownership = sig.ParamOwnership[i]
		}
		if ownership == "borrow" || ownership == "inout" {
			return fmt.Sprintf("parameter %d uses %s ownership", i+1, ownership)
		}
		if !typeActorTaskSendable(typeName, types, map[string]bool{}) {
			return fmt.Sprintf("parameter %d type '%s' is not sendable", i+1, typeName)
		}
	}
	if sig.ReturnType == "" {
		return ""
	}
	if !typeActorTaskSendable(sig.ReturnType, types, map[string]bool{}) {
		return fmt.Sprintf("return type '%s' is not sendable", sig.ReturnType)
	}
	return ""
}

func typeActorTaskSendable(typeName string, types map[string]*TypeInfo, seen map[string]bool) bool {
	if _, ok := surfaceActorTaskBoundaryValueType(typeName, types); ok {
		return false
	}
	if seen[typeName] {
		return true
	}
	seen[typeName] = true
	info, ok := types[typeName]
	if !ok {
		return false
	}
	if info.RuntimeOwned && !info.ActorSendable {
		return false
	}
	switch info.Kind {
	case TypeI32, TypeI64, TypeU8, TypeBool, TypeActor:
		return true
	case TypeEnum:
		for _, c := range info.EnumCases {
			for _, payload := range c.PayloadTypes {
				if !typeActorTaskSendable(payload, types, seen) {
					return false
				}
			}
		}
		return true
	case TypeOptional:
		return typeActorTaskSendable(info.ElemType, types, seen)
	case TypeStruct:
		for _, field := range info.Fields {
			if !typeActorTaskSendable(field.TypeName, types, seen) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func typeActorTaskSendabilityUnsafeReason(
	typeName string,
	types map[string]*TypeInfo,
	seen map[string]bool,
) string {
	if surfaceType, ok := surfaceActorTaskBoundaryValueType(typeName, types); ok {
		return fmt.Sprintf("surface value '%s' cannot cross actor/task boundary", surfaceType)
	}
	if seen[typeName] {
		return ""
	}
	seen[typeName] = true
	info, ok := types[typeName]
	if !ok {
		return fmt.Sprintf("unknown type '%s'", typeName)
	}
	if info.RuntimeOwned && !info.ActorSendable {
		return fmt.Sprintf("runtime-owned type '%s' cannot cross actor/task boundary", typeName)
	}
	switch info.Kind {
	case TypeI32, TypeI64, TypeU8, TypeBool, TypeActor:
		return ""
	case TypeEnum:
		for _, c := range info.EnumCases {
			for _, payload := range c.PayloadTypes {
				if reason := typeActorTaskSendabilityUnsafeReason(payload, types, seen); reason != "" {
					return reason
				}
			}
		}
		return ""
	case TypeOptional:
		return typeActorTaskSendabilityUnsafeReason(info.ElemType, types, seen)
	case TypeStruct:
		for _, field := range info.Fields {
			if reason := typeActorTaskSendabilityUnsafeReason(field.TypeName, types, seen); reason != "" {
				return reason
			}
		}
		return ""
	default:
		return fmt.Sprintf("type '%s' is not sendable", typeName)
	}
}

// ---- ui.go ----

func stateAsStructDecl(state *frontend.StateDecl) *frontend.StructDecl {
	if state == nil {
		return &frontend.StructDecl{}
	}
	fields := make([]frontend.FieldDecl, 0, len(state.Fields))
	for _, field := range state.Fields {
		fields = append(fields, frontend.FieldDecl{
			At:   field.At,
			Name: field.Name,
			Type: field.Type,
		})
	}
	return &frontend.StructDecl{
		At:     state.At,
		Name:   state.Name,
		Fields: fields,
	}
}

func checkUIDecls(world *module.World, checked *CheckedProgram, types map[string]*TypeInfo) error {
	if world == nil || checked == nil {
		return nil
	}

	importsByModule := make(map[string]map[string]string, len(world.Files))
	for _, file := range world.Files {
		imports, err := collectImportAliases(file)
		if err != nil {
			return err
		}
		importsByModule[file.Module] = imports
	}

	stateByName := make(map[string]CheckedUIState, len(checked.UIStates))
	stateConstFields := make(map[string]map[string]bool, len(checked.UIStates))
	for i := range checked.UIStates {
		state := checked.UIStates[i]
		stateByName[state.Name] = state
		stateConstFields[state.Name] = make(map[string]bool)
	}

	emptyGlobals := map[string]GlobalInfo{}
	for i := range checked.UIStates {
		state := &checked.UIStates[i]
		imports := importsByModule[state.Module]
		initLocals := make(map[string]LocalInfo, len(state.Decl.Fields))
		slot := 0
		for j := range state.Decl.Fields {
			field := &state.Decl.Fields[j]
			resolved, err := resolveTypeName(&field.Type, state.Module, imports)
			if err != nil {
				return err
			}
			field.Type.Name = resolved
			info, err := ensureTypeInfo(resolved, types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(field.At), err)
			}
			if field.Init == nil {
				return fmt.Errorf(
					"%s: state field '%s' requires an initializer",
					frontend.FormatPos(field.At),
					field.Name,
				)
			}
			exprType, _, err := checkExprWithEffects(
				field.Init,
				initLocals,
				emptyGlobals,
				checked.FuncSigs,
				types,
				state.Module,
				imports,
				newRegionState(nil),
				newEffectContext(state.Name, nil, nil, true),
				nil,
			)
			if err != nil {
				return fmt.Errorf(
					"%s: state '%s' field '%s': %v",
					frontend.FormatPos(field.At),
					state.Name,
					field.Name,
					err,
				)
			}
			if !typesCompatibleWithNullPtr(resolved, exprType, field.Init) {
				return fmt.Errorf(
					"%s: state '%s' field '%s' type mismatch: expected '%s', got '%s'",
					frontend.FormatPos(field.At),
					state.Name,
					field.Name,
					resolved,
					exprType,
				)
			}
			initLocals[field.Name] = LocalInfo{
				Base:      slot,
				SlotCount: info.SlotCount,
				TypeName:  resolved,
				Mutable:   field.Mutable,
				Const:     field.Const,
			}
			slot += info.SlotCount
			stateConstFields[state.Name][field.Name] = field.Const || !field.Mutable
		}
	}

	seenViews := make(map[string]struct{})
	for _, file := range world.Files {
		imports := importsByModule[file.Module]
		for i := range file.Views {
			view := file.Views[i]
			fullName := qualifyName(file.Module, view.Name)
			if _, exists := seenViews[fullName]; exists {
				return fmt.Errorf("duplicate view '%s'", fullName)
			}
			seenViews[fullName] = struct{}{}

			stateType, err := resolveTypeName(&view.StateName, file.Module, imports)
			if err != nil {
				return err
			}
			view.StateName.Name = stateType
			_, ok := stateByName[stateType]
			if !ok {
				return fmt.Errorf(
					"%s: view '%s' references unknown state '%s'",
					frontend.FormatPos(view.At),
					fullName,
					stateType,
				)
			}
			stateInfo, err := ensureTypeInfo(stateType, types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(view.At), err)
			}

			bindingNames := map[string]struct{}{}
			eventNames := map[string]struct{}{}
			styleNames := map[string]struct{}{}
			a11yNames := map[string]struct{}{}
			commandNames := map[string]struct{}{}

			baseLocals := map[string]LocalInfo{
				"state": {
					Base:      0,
					SlotCount: stateInfo.SlotCount,
					TypeName:  stateType,
					Mutable:   true,
				},
			}
			baseSlot := stateInfo.SlotCount
			baseState := newRegionState(nil)
			baseEffects := newEffectContext(fullName, nil, nil, true)
			for j := range view.Bindings {
				binding := &view.Bindings[j]
				if _, exists := bindingNames[binding.Name]; exists {
					return fmt.Errorf(
						"%s: duplicate binding '%s'",
						frontend.FormatPos(binding.At),
						binding.Name,
					)
				}
				bindingNames[binding.Name] = struct{}{}
				resolved, err := resolveTypeName(&binding.Type, file.Module, imports)
				if err != nil {
					return err
				}
				binding.Type.Name = resolved
				info, err := ensureTypeInfo(resolved, types)
				if err != nil {
					return fmt.Errorf("%s: %v", frontend.FormatPos(binding.At), err)
				}
				exprType, _, err := checkExprWithEffects(
					binding.Value,
					baseLocals,
					emptyGlobals,
					checked.FuncSigs,
					types,
					file.Module,
					imports,
					baseState,
					baseEffects,
					nil,
				)
				if err != nil {
					return fmt.Errorf(
						"%s: binding '%s': %v",
						frontend.FormatPos(binding.At),
						binding.Name,
						err,
					)
				}
				if !typesCompatibleWithNullPtr(resolved, exprType, binding.Value) {
					return fmt.Errorf(
						"%s: binding '%s' type mismatch: expected '%s', got '%s'",
						frontend.FormatPos(binding.At),
						binding.Name,
						resolved,
						exprType,
					)
				}
				baseLocals[binding.Name] = LocalInfo{
					Base:      baseSlot,
					SlotCount: info.SlotCount,
					TypeName:  resolved,
					Mutable:   false,
					Const:     true,
				}
				baseSlot += info.SlotCount
			}

			for j := range view.Commands {
				cmd := &view.Commands[j]
				if _, exists := commandNames[cmd.Name]; exists {
					return fmt.Errorf(
						"%s: duplicate command '%s'",
						frontend.FormatPos(cmd.At),
						cmd.Name,
					)
				}
				commandNames[cmd.Name] = struct{}{}
			}
			if len(view.Commands) == 0 {
				return fmt.Errorf(
					"%s: view '%s' must declare at least one command",
					frontend.FormatPos(view.At),
					fullName,
				)
			}
			for j := range view.Events {
				event := &view.Events[j]
				if _, exists := eventNames[event.Name]; exists {
					return fmt.Errorf(
						"%s: duplicate event '%s'",
						frontend.FormatPos(event.At),
						event.Name,
					)
				}
				eventNames[event.Name] = struct{}{}
				if _, exists := commandNames[event.Command]; !exists {
					return fmt.Errorf(
						"%s: event '%s' references unknown command '%s'",
						frontend.FormatPos(event.At),
						event.Name,
						event.Command,
					)
				}
			}

			for j := range view.Styles {
				style := &view.Styles[j]
				if _, exists := styleNames[style.Name]; exists {
					return fmt.Errorf(
						"%s: duplicate style '%s'",
						frontend.FormatPos(style.At),
						style.Name,
					)
				}
				styleNames[style.Name] = struct{}{}
				resolved, err := resolveTypeName(&style.Type, file.Module, imports)
				if err != nil {
					return err
				}
				style.Type.Name = resolved
				if !isUIScalarType(resolved) {
					return fmt.Errorf(
						"%s: style '%s' uses unsupported type '%s' (allowed: i32, bool, str)",
						frontend.FormatPos(style.At),
						style.Name,
						resolved,
					)
				}
				exprType, _, err := checkExprWithEffects(
					style.Value,
					baseLocals,
					emptyGlobals,
					checked.FuncSigs,
					types,
					file.Module,
					imports,
					newRegionState(nil),
					baseEffects,
					nil,
				)
				if err != nil {
					return fmt.Errorf(
						"%s: style '%s': %v",
						frontend.FormatPos(style.At),
						style.Name,
						err,
					)
				}
				if !typesCompatibleWithNullPtr(resolved, exprType, style.Value) {
					return fmt.Errorf(
						"%s: style '%s' type mismatch: expected '%s', got '%s'",
						frontend.FormatPos(style.At),
						style.Name,
						resolved,
						exprType,
					)
				}
			}

			for j := range view.Accessibility {
				entry := &view.Accessibility[j]
				if _, exists := a11yNames[entry.Name]; exists {
					return fmt.Errorf(
						"%s: duplicate accessibility key '%s'",
						frontend.FormatPos(entry.At),
						entry.Name,
					)
				}
				a11yNames[entry.Name] = struct{}{}
				resolved, err := resolveTypeName(&entry.Type, file.Module, imports)
				if err != nil {
					return err
				}
				entry.Type.Name = resolved
				if !isUIScalarType(resolved) {
					return fmt.Errorf(
						"%s: accessibility '%s' uses unsupported type '%s' (allowed: i32, bool, str)",
						frontend.FormatPos(entry.At),
						entry.Name,
						resolved,
					)
				}
				exprType, _, err := checkExprWithEffects(
					entry.Value,
					baseLocals,
					emptyGlobals,
					checked.FuncSigs,
					types,
					file.Module,
					imports,
					newRegionState(nil),
					baseEffects,
					nil,
				)
				if err != nil {
					return fmt.Errorf(
						"%s: accessibility '%s': %v",
						frontend.FormatPos(entry.At),
						entry.Name,
						err,
					)
				}
				if !typesCompatibleWithNullPtr(resolved, exprType, entry.Value) {
					return fmt.Errorf(
						"%s: accessibility '%s' type mismatch: expected '%s', got '%s'",
						frontend.FormatPos(entry.At),
						entry.Name,
						resolved,
						exprType,
					)
				}
			}

			for j := range view.Commands {
				cmd := &view.Commands[j]
				if err := validateViewCommandStmts(cmd.Body, stateConstFields[stateType]); err != nil {
					return fmt.Errorf(
						"%s: command '%s': %v",
						frontend.FormatPos(cmd.At),
						cmd.Name,
						err,
					)
				}
				cmdLocals := cloneUILocals(baseLocals)
				slotIndex := baseSlot
				scopes := newScopeInfo()
				if err := collectLocals(
					cmd.Body,
					cmdLocals,
					&slotIndex,
					checked.FuncSigs,
					types,
					file.Module,
					imports,
					scopes,
					emptyGlobals,
				); err != nil {
					return fmt.Errorf(
						"%s: command '%s': %v",
						frontend.FormatPos(cmd.At),
						cmd.Name,
						err,
					)
				}
				cmdState := newRegionState(scopes)
				cmdEffects := newEffectContext(fullName+".command."+cmd.Name, nil, nil, false)
				if err := checkStmts(
					cmd.Body,
					cmdLocals,
					emptyGlobals,
					checked.FuncSigs,
					types,
					file.Module,
					imports,
					"i32",
					nil,
					nil,
					cmdState,
					cmdEffects,
					&functionAnalysisState{},
				); err != nil {
					return fmt.Errorf(
						"%s: command '%s': %v",
						frontend.FormatPos(cmd.At),
						cmd.Name,
						err,
					)
				}
			}

			checked.UIViews = append(checked.UIViews, CheckedUIView{
				Name:   fullName,
				Module: file.Module,
				Decl:   view,
			})
		}
	}

	return nil
}

func isUIScalarType(typeName string) bool {
	switch typeName {
	case "i32", "bool", "str":
		return true
	default:
		return false
	}
}

func cloneUILocals(src map[string]LocalInfo) map[string]LocalInfo {
	out := make(map[string]LocalInfo, len(src))
	for name, info := range src {
		out[name] = info
	}
	return out
}

func validateViewCommandStmts(stmts []frontend.Stmt, stateConstFields map[string]bool) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.ReturnStmt:
			return fmt.Errorf("%s: return is not allowed inside view commands", frontend.FormatPos(s.At))
		case *frontend.ThrowStmt:
			return fmt.Errorf("%s: throw is not allowed inside view commands", frontend.FormatPos(s.At))
		case *frontend.DeferStmt:
			if err := validateViewCommandStmts(s.Body, stateConstFields); err != nil {
				return err
			}
		case *frontend.AssignStmt:
			if field, ok := assignedStateField(s.Target); ok {
				if field == "" {
					return fmt.Errorf(("%s: assigning to 'state' directly is not allowed in view " +
						"commands"), frontend.FormatPos(s.At))
				}
				if stateConstFields[field] {
					return fmt.Errorf(
						"%s: cannot assign to immutable state field '%s'",
						frontend.FormatPos(s.At),
						field,
					)
				}
			}
		case *frontend.IfStmt:
			if err := validateViewCommandStmts(s.Then, stateConstFields); err != nil {
				return err
			}
			if err := validateViewCommandStmts(s.Else, stateConstFields); err != nil {
				return err
			}
		case *frontend.IfLetStmt:
			if err := validateViewCommandStmts(s.Then, stateConstFields); err != nil {
				return err
			}
			if err := validateViewCommandStmts(s.Else, stateConstFields); err != nil {
				return err
			}
		case *frontend.WhileStmt:
			if err := validateViewCommandStmts(s.Body, stateConstFields); err != nil {
				return err
			}
		case *frontend.ForRangeStmt:
			if err := validateViewCommandStmts(s.Body, stateConstFields); err != nil {
				return err
			}
		case *frontend.MatchStmt:
			for _, c := range s.Cases {
				if err := validateViewCommandStmts(c.Body, stateConstFields); err != nil {
					return err
				}
			}
		case *frontend.UnsafeStmt:
			if err := validateViewCommandStmts(s.Body, stateConstFields); err != nil {
				return err
			}
		case *frontend.IslandStmt:
			if err := validateViewCommandStmts(s.Body, stateConstFields); err != nil {
				return err
			}
		}
	}
	return nil
}

func assignedStateField(expr frontend.Expr) (string, bool) {
	target := expr
	if idx, ok := expr.(*frontend.IndexExpr); ok {
		target = idx.Base
	}
	base, fields, _, ok := splitFieldPath(target)
	if !ok || base != "state" {
		return "", false
	}
	if len(fields) == 0 {
		return "", true
	}
	return fields[0], true
}
