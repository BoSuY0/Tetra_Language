package lower

import (
	"fmt"
	"hash/fnv"
	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
)

func (l *lowerer) lowerCallExpr(e *frontend.CallExpr) (int, error) {
	if slots, ok, err := l.lowerEnumCaseConstructorCall(e, nil); ok {
		return slots, err
	}
	if slots, ok, err := l.lowerStructConstructorCall(e, nil); ok {
		return slots, err
	}
	if fieldInfo, base, ok, err := l.functionFieldCallSource(e.Name, e.At); err != nil {
		return 0, err
	} else if ok {
		return l.lowerStoredFunctionCall(e, fieldInfo, base)
	}
	if local, ok := l.locals[e.Name]; ok && local.FunctionTypeValue {
		if local.FunctionHandleValue {
			return l.lowerFunctionTypedParamCall(e, local)
		}
		if local.FunctionValue != "" && !local.Mutable {
			return l.lowerStoredFunctionCall(e, semantics.FunctionFieldInfo{
				FunctionValue:          local.FunctionValue,
				FunctionParamTypes:     append([]string(nil), local.FunctionParamTypes...),
				FunctionParamOwnership: append([]string(nil), local.FunctionParamOwnership...),
				FunctionReturnType:     local.FunctionReturnType,
				FunctionThrowsType:     local.FunctionThrowsType,
			}, local.Base)
		}
		return l.lowerFunctionTypedParamCall(e, local)
	}
	if global, ok := l.globals[e.Name]; ok && global.FunctionTypeValue {
		l.emitGlobalFunctionValueInitIfNeeded(global, e.At)
		return l.lowerGlobalStoredFunctionCall(e, global)
	}
	e = lowerCallExprWithBuiltinAlias(e)
	if slots, ok, err := l.lowerRawOffsetCall(e); ok {
		return slots, err
	}
	if slots, ok, err := l.lowerPtrAddValueCall(e); ok {
		return slots, err
	}
	if slots, ok, err := l.lowerAtomicBuiltinCall(e); ok {
		return slots, err
	}
	switch e.Name {
	case "core.surface_open":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_open", 4)
	case "core.surface_close":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_close", 1)
	case "core.surface_poll_event_kind":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_poll_event_kind", 1)
	case "core.surface_poll_event_x":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_poll_event_x", 1)
	case "core.surface_poll_event_y":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_poll_event_y", 1)
	case "core.surface_poll_event_button":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_poll_event_button", 1)
	case "core.surface_poll_event_into":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_poll_event_into", 3)
	case "core.surface_poll_event_text_len":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_poll_event_text_len", 1)
	case "core.surface_poll_event_text_into":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_poll_event_text_into", 3)
	case "core.surface_clipboard_write_text":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_clipboard_write_text", 3)
	case "core.surface_clipboard_read_text_into":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_clipboard_read_text_into", 3)
	case "core.surface_poll_composition_into":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_poll_composition_into", 3)
	case "core.surface_begin_frame":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_begin_frame", 1)
	case "core.surface_present_rgba":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_present_rgba", 6)
	case "core.surface_now_ms":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_now_ms", 0)
	case "core.surface_request_redraw":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_request_redraw", 1)
	case "core.spawn":
		if len(e.Args) != 1 {
			return 0, fmt.Errorf("%s: spawn expects 1 argument", frontend.FormatPos(e.At))
		}
		lit, ok := e.Args[0].(*frontend.StringLitExpr)
		if !ok {
			return 0, fmt.Errorf("%s: spawn expects a string literal", frontend.FormatPos(e.At))
		}
		name := string(lit.Value)
		if name == "" {
			return 0, fmt.Errorf("%s: spawn expects a non-empty name", frontend.FormatPos(e.At))
		}
		h := fnv.New32a()
		_, _ = h.Write([]byte(name))
		id := int32(h.Sum32())
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: id, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_spawn", ArgSlots: 1, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.spawn_remote":
		if len(e.Args) != 2 {
			return 0, fmt.Errorf("%s: spawn_remote expects 2 arguments", frontend.FormatPos(e.At))
		}
		nodeSlots, err := l.lowerExpr(e.Args[0])
		if err != nil {
			return 0, err
		}
		if nodeSlots != 1 {
			return 0, fmt.Errorf("%s: spawn_remote expects a 1-slot node id", frontend.FormatPos(e.Args[0].Pos()))
		}
		lit, ok := e.Args[1].(*frontend.StringLitExpr)
		if !ok {
			return 0, fmt.Errorf("%s: spawn_remote expects a string literal", frontend.FormatPos(e.At))
		}
		name := string(lit.Value)
		if name == "" {
			return 0, fmt.Errorf("%s: spawn_remote expects a non-empty name", frontend.FormatPos(e.At))
		}
		h := fnv.New32a()
		_, _ = h.Write([]byte(name))
		id := int32(h.Sum32())
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: id, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_spawn_remote", ArgSlots: 2, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.task_spawn_i32":
		if len(e.Args) != 1 {
			return 0, fmt.Errorf("%s: task_spawn_i32 expects 1 argument", frontend.FormatPos(e.At))
		}
		lit, ok := e.Args[0].(*frontend.StringLitExpr)
		if !ok {
			return 0, fmt.Errorf("%s: task_spawn_i32 expects a string literal", frontend.FormatPos(e.At))
		}
		name := string(lit.Value)
		if name == "" {
			return 0, fmt.Errorf("%s: task_spawn_i32 expects a non-empty name", frontend.FormatPos(e.At))
		}
		sig, ok := l.funcs[name]
		if !ok {
			return 0, fmt.Errorf("%s: unknown task target '%s'", frontend.FormatPos(e.At), name)
		}
		if sig.ReturnSlots != 1 {
			return 0, fmt.Errorf("%s: task_spawn_i32 target must return 1 slot", frontend.FormatPos(e.At))
		}
		h := fnv.New32a()
		_, _ = h.Write([]byte(name))
		id := int32(h.Sum32())
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: id, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_spawn_i32", ArgSlots: 1, RetSlots: 2, Pos: e.At})
		return 2, nil
	case "core.task_spawn_i32_typed":
		if len(e.TypeArgs) != 1 {
			return 0, fmt.Errorf("%s: task_spawn_i32_typed expects one explicit error type argument", frontend.FormatPos(e.At))
		}
		errorType := e.TypeArgs[0].Name
		if errorType == "" {
			return 0, fmt.Errorf("%s: task_spawn_i32_typed missing resolved error type", frontend.FormatPos(e.At))
		}
		_, handleInfo, err := semantics.EnsureTypedTaskHandleType(errorType, l.types)
		if err != nil {
			return 0, fmt.Errorf("%s: %v", frontend.FormatPos(e.TypeArgs[0].At), err)
		}
		if len(e.Args) != 1 {
			return 0, fmt.Errorf("%s: task_spawn_i32_typed expects 1 argument", frontend.FormatPos(e.At))
		}
		lit, ok := e.Args[0].(*frontend.StringLitExpr)
		if !ok {
			return 0, fmt.Errorf("%s: task_spawn_i32_typed expects a string literal", frontend.FormatPos(e.At))
		}
		name := string(lit.Value)
		if name == "" {
			return 0, fmt.Errorf("%s: task_spawn_i32_typed expects a non-empty name", frontend.FormatPos(e.At))
		}
		sig, ok := l.funcs[name]
		if !ok {
			return 0, fmt.Errorf("%s: unknown task target '%s'", frontend.FormatPos(e.At), name)
		}
		if handleInfo.SlotCount <= 4 {
			if sig.ReturnSlots != handleInfo.SlotCount {
				return 0, fmt.Errorf("%s: task_spawn_i32_typed target return slot mismatch", frontend.FormatPos(e.At))
			}
		} else if sig.ReturnType != "i32" {
			return 0, fmt.Errorf("%s: task_spawn_i32_typed staged mode requires target return type i32", frontend.FormatPos(e.At))
		}
		wrapperName := typedTaskWrapperName(name, errorType)
		h := fnv.New32a()
		_, _ = h.Write([]byte(wrapperName))
		id := int32(h.Sum32())
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: id, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_spawn_i32", ArgSlots: 1, RetSlots: 2, Pos: e.At})
		if handleInfo.SlotCount > 2 {
			statusLocal := l.allocScratchSlots(1)
			handleLocal := l.allocScratchSlots(1)
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: statusLocal, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: handleLocal, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: handleLocal, Pos: e.At})
			l.emitZeroSlots(handleInfo.SlotCount-2, e.At)
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: statusLocal, Pos: e.At})
		}
		return handleInfo.SlotCount, nil
	case "core.task_spawn_group_i32_typed":
		if len(e.TypeArgs) != 1 {
			return 0, fmt.Errorf("%s: task_spawn_group_i32_typed expects one explicit error type argument", frontend.FormatPos(e.At))
		}
		errorType := e.TypeArgs[0].Name
		if errorType == "" {
			return 0, fmt.Errorf("%s: task_spawn_group_i32_typed missing resolved error type", frontend.FormatPos(e.At))
		}
		_, handleInfo, err := semantics.EnsureTypedTaskHandleType(errorType, l.types)
		if err != nil {
			return 0, fmt.Errorf("%s: %v", frontend.FormatPos(e.TypeArgs[0].At), err)
		}
		if len(e.Args) != 2 {
			return 0, fmt.Errorf("%s: task_spawn_group_i32_typed expects 2 arguments", frontend.FormatPos(e.At))
		}
		groupSlots, err := l.lowerExpr(e.Args[0])
		if err != nil {
			return 0, err
		}
		if groupSlots != 1 {
			return 0, fmt.Errorf("%s: task_spawn_group_i32_typed expects a 1-slot task.group handle", frontend.FormatPos(e.At))
		}
		groupLocal := l.allocScratchSlots(1)
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: groupLocal, Pos: e.At})
		lit, ok := e.Args[1].(*frontend.StringLitExpr)
		if !ok {
			return 0, fmt.Errorf("%s: task_spawn_group_i32_typed expects a string literal worker name", frontend.FormatPos(e.At))
		}
		name := string(lit.Value)
		if name == "" {
			return 0, fmt.Errorf("%s: task_spawn_group_i32_typed expects a non-empty name", frontend.FormatPos(e.At))
		}
		sig, ok := l.funcs[name]
		if !ok {
			return 0, fmt.Errorf("%s: unknown task target '%s'", frontend.FormatPos(e.At), name)
		}
		if handleInfo.SlotCount <= 4 {
			if sig.ReturnSlots != handleInfo.SlotCount {
				return 0, fmt.Errorf("%s: task_spawn_group_i32_typed target return slot mismatch", frontend.FormatPos(e.At))
			}
		} else if sig.ReturnType != "i32" {
			return 0, fmt.Errorf("%s: task_spawn_group_i32_typed staged mode requires target return type i32", frontend.FormatPos(e.At))
		}

		activeLabel := l.newLabel()
		endLabel := l.newLabel()
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: groupLocal, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: activeLabel, Pos: e.At})
		l.emitZeroSlots(handleInfo.SlotCount-1, e.At)
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})

		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: activeLabel, Pos: e.At})
		wrapperName := typedTaskWrapperName(name, errorType)
		h := fnv.New32a()
		_, _ = h.Write([]byte(wrapperName))
		id := int32(h.Sum32())
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: groupLocal, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: id, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_spawn_group_i32", ArgSlots: 2, RetSlots: 2, Pos: e.At})
		if handleInfo.SlotCount > 2 {
			statusLocal := l.allocScratchSlots(1)
			handleLocal := l.allocScratchSlots(1)
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: statusLocal, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: handleLocal, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: handleLocal, Pos: e.At})
			l.emitZeroSlots(handleInfo.SlotCount-2, e.At)
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: statusLocal, Pos: e.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
		return handleInfo.SlotCount, nil
	case "core.task_spawn_group_i32":
		if len(e.Args) != 2 {
			return 0, fmt.Errorf("%s: task_spawn_group_i32 expects 2 arguments", frontend.FormatPos(e.At))
		}
		groupSlots, err := l.lowerExpr(e.Args[0])
		if err != nil {
			return 0, err
		}
		if groupSlots != 1 {
			return 0, fmt.Errorf("%s: task_spawn_group_i32 expects a 1-slot task.group handle", frontend.FormatPos(e.At))
		}
		groupLocal := l.allocScratchSlots(1)
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: groupLocal, Pos: e.At})
		lit, ok := e.Args[1].(*frontend.StringLitExpr)
		if !ok {
			return 0, fmt.Errorf("%s: task_spawn_group_i32 expects a string literal worker name", frontend.FormatPos(e.At))
		}
		name := string(lit.Value)
		if name == "" {
			return 0, fmt.Errorf("%s: task_spawn_group_i32 expects a non-empty name", frontend.FormatPos(e.At))
		}
		sig, ok := l.funcs[name]
		if !ok {
			return 0, fmt.Errorf("%s: unknown task target '%s'", frontend.FormatPos(e.At), name)
		}
		if sig.ReturnSlots != 1 {
			return 0, fmt.Errorf("%s: task_spawn_group_i32 target must return 1 slot", frontend.FormatPos(e.At))
		}

		activeLabel := l.newLabel()
		endLabel := l.newLabel()
		// group == 0 => canceled handle
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: groupLocal, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: activeLabel, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})

		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: activeLabel, Pos: e.At})
		h := fnv.New32a()
		_, _ = h.Write([]byte(name))
		id := int32(h.Sum32())
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: groupLocal, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: id, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_spawn_group_i32", ArgSlots: 2, RetSlots: 2, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
		return 2, nil
	case "core.recv":
		if len(e.Args) != 0 {
			return 0, fmt.Errorf("%s: recv expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_recv", ArgSlots: 0, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.recv_msg":
		if len(e.Args) != 0 {
			return 0, fmt.Errorf("%s: recv_msg expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_recv_msg", ArgSlots: 0, RetSlots: 2, Pos: e.At})
		return 2, nil
	case "core.recv_typed":
		if len(e.Args) != 0 {
			return 0, fmt.Errorf("%s: recv_typed expects 0 arguments", frontend.FormatPos(e.At))
		}
		if len(e.TypeArgs) != 1 {
			return 0, fmt.Errorf("%s: recv_typed expects one explicit type argument", frontend.FormatPos(e.At))
		}
		msgType := e.TypeArgs[0].Name
		info, ok := l.types[msgType]
		if !ok || info.Kind != semantics.TypeEnum {
			return 0, fmt.Errorf("%s: recv_typed expects an enum type argument", frontend.FormatPos(e.At))
		}
		base := l.allocScratchSlots(info.SlotCount)
		tagBase := typedActorMessageTagBase(msgType)
		nonNegativeLabel := l.newLabel()
		mismatchLabel := l.newLabel()
		endLabel := l.newLabel()
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_recv_begin", ArgSlots: 0, RetSlots: 1, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: tagBase, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRSubI32, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: base, Pos: e.At})

		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: base, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRCmpLtI32, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nonNegativeLabel, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: mismatchLabel, Pos: e.At})

		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nonNegativeLabel, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: base, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(len(info.EnumCases)), Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRCmpLtI32, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: mismatchLabel, Pos: e.At})
		for slot := 0; slot < info.SlotCount-1; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slot), Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_recv_slot", ArgSlots: 1, RetSlots: 1, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: base + 1 + slot, Pos: e.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})

		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: mismatchLabel, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: base, Pos: e.At})
		for slot := 0; slot < info.SlotCount-1; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: -1, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: base + 1 + slot, Pos: e.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
		for slot := 0; slot < info.SlotCount; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: base + slot, Pos: e.At})
		}
		return info.SlotCount, nil
	case "core.send_typed":
		if len(e.Args) != 2 {
			return 0, fmt.Errorf("%s: send_typed expects 2 arguments", frontend.FormatPos(e.At))
		}
		targetSlots, err := l.lowerExpr(e.Args[0])
		if err != nil {
			return 0, err
		}
		if targetSlots != 1 {
			return 0, fmt.Errorf("%s: send_typed expects actor target", frontend.FormatPos(e.Args[0].Pos()))
		}
		targetLocal := l.allocScratchSlots(1)
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: targetLocal, Pos: e.At})
		msgType, err := l.inferExprType(e.Args[1])
		if err != nil {
			return 0, err
		}
		info, ok := l.types[msgType]
		if !ok || info.Kind != semantics.TypeEnum {
			return 0, fmt.Errorf("%s: send_typed expects an enum message", frontend.FormatPos(e.Args[1].Pos()))
		}
		msgBase := l.allocScratchSlots(info.SlotCount)
		msgSlots, err := l.lowerExpr(e.Args[1])
		if err != nil {
			return 0, err
		}
		if msgSlots != info.SlotCount {
			return 0, fmt.Errorf("%s: send_typed message slot mismatch", frontend.FormatPos(e.Args[1].Pos()))
		}
		for slot := info.SlotCount - 1; slot >= 0; slot-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: msgBase + slot, Pos: e.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: targetLocal, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: msgBase, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: typedActorMessageTagBase(msgType), Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRAddI32, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(info.SlotCount - 1), Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_send_begin", ArgSlots: 3, RetSlots: 1, Pos: e.At})
		beginResult := l.allocScratchSlots(1)
		beginFailedLabel := l.newLabel()
		endLabel := l.newLabel()
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: beginResult, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: beginResult, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: beginFailedLabel, Pos: e.At})
		discard := l.ensureDiscardLocal()
		for slot := 0; slot < info.SlotCount-1; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slot), Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: msgBase + 1 + slot, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_send_slot", ArgSlots: 2, RetSlots: 1, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: e.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_send_commit", ArgSlots: 0, RetSlots: 1, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: beginFailedLabel, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: beginResult, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
		return 1, nil
	case "core.self":
		if len(e.Args) != 0 {
			return 0, fmt.Errorf("%s: self expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_self", ArgSlots: 0, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.sender":
		if len(e.Args) != 0 {
			return 0, fmt.Errorf("%s: sender expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_sender", ArgSlots: 0, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.sym_addr":
		if len(e.Args) != 1 {
			return 0, fmt.Errorf("%s: sym_addr expects 1 argument", frontend.FormatPos(e.At))
		}
		lit, ok := e.Args[0].(*frontend.StringLitExpr)
		if !ok {
			return 0, fmt.Errorf("%s: sym_addr expects a string literal", frontend.FormatPos(e.At))
		}
		name := string(lit.Value)
		if name == "" {
			return 0, fmt.Errorf("%s: sym_addr expects a non-empty symbol name", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRSymAddr, Name: name, Pos: e.At})
		return 1, nil
	}
	total := 0
	callSig, hasCallSig := l.funcs[e.Name]
	for i, arg := range e.Args {
		var slots int
		var err error
		if hasCallSig && i < len(callSig.ParamFunctionTypes) && callSig.ParamFunctionTypes[i] {
			slots, err = l.lowerFunctionTypedArgument(arg)
		} else if hasCallSig && i < len(callSig.ParamTypes) {
			slots, err = l.lowerExprAs(arg, callSig.ParamTypes[i])
		} else {
			slots, err = l.lowerExpr(arg)
		}
		if err != nil {
			return 0, err
		}
		total += slots
	}
	if hasCallSig {
		l.invalidateWhileRangeProofsForInoutArgs(e.Args, callSig.ParamOwnership)
	}
	switch e.Name {
	case "core.cap_io":
		if total != 0 {
			return 0, fmt.Errorf("%s: cap_io expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCapIO, Pos: e.At})
		return 1, nil
	case "core.cap_mem":
		if total != 0 {
			return 0, fmt.Errorf("%s: cap_mem expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCapMem, Pos: e.At})
		return 1, nil
	case "core.alloc_bytes":
		if total != 1 {
			return 0, fmt.Errorf("%s: alloc_bytes expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRAllocBytes, Pos: e.At})
		return 1, nil
	case "core.make_u8":
		if total != 1 {
			return 0, fmt.Errorf("%s: make_u8 expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMakeSliceU8, Pos: e.At})
		return 2, nil
	case "core.make_u16":
		if total != 1 {
			return 0, fmt.Errorf("%s: make_u16 expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMakeSliceU16, Pos: e.At})
		return 2, nil
	case "core.make_i32":
		if total != 1 {
			return 0, fmt.Errorf("%s: make_i32 expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMakeSliceI32, Pos: e.At})
		return 2, nil
	case "core.make_bool":
		if total != 1 {
			return 0, fmt.Errorf("%s: make_bool expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMakeSliceI32, Pos: e.At})
		return 2, nil
	case "core.raw_slice_u8_from_parts", "core.raw_slice_u16_from_parts", "core.raw_slice_i32_from_parts", "core.raw_slice_bool_from_parts":
		if total != 3 {
			return 0, fmt.Errorf("%s: %s expects ptr, length, and cap.mem arguments", frontend.FormatPos(e.At), e.Name)
		}
		l.emit(ir.IRInstr{Kind: ir.IRRawSliceFromParts, Imm: rawSliceElementShift(e.Name), Pos: e.At})
		return 2, nil
	case "core.slice_borrow_u8", "core.slice_borrow_u16", "core.slice_borrow_i32", "core.slice_borrow_bool", "core.string_borrow":
		if total != 2 {
			return 0, fmt.Errorf("%s: %s expects one view source argument", frontend.FormatPos(e.At), e.Name)
		}
		return 2, nil
	case "core.slice_copy_u8", "core.slice_copy_u16", "core.slice_copy_i32", "core.slice_copy_bool", "core.string_copy":
		return l.lowerCopyBuiltinFromStack(e.Name, total, e.At)
	case "core.slice_copy_into_u8", "core.slice_copy_into_u16", "core.slice_copy_into_i32", "core.slice_copy_into_bool", "core.string_copy_into":
		return l.lowerCopyIntoBuiltinFromStack(e.Name, total, e.At)
	case "core.slice_window_u8", "core.slice_window_u16", "core.slice_window_i32", "core.slice_window_bool", "core.string_window":
		if total != 4 {
			return 0, fmt.Errorf("%s: %s expects view source, start, and count arguments", frontend.FormatPos(e.At), e.Name)
		}
		shift, ok := sliceViewElementShift(e.Name)
		if !ok {
			return 0, lowerUnsupportedError(e.At, "unsupported view window builtin '%s'", e.Name)
		}
		l.emit(ir.IRInstr{Kind: ir.IRSliceWindow, Imm: shift, Pos: e.At})
		return 2, nil
	case "core.slice_prefix_u8", "core.slice_prefix_u16", "core.slice_prefix_i32", "core.slice_prefix_bool", "core.string_prefix":
		if total != 3 {
			return 0, fmt.Errorf("%s: %s expects view source and count arguments", frontend.FormatPos(e.At), e.Name)
		}
		shift, ok := sliceViewElementShift(e.Name)
		if !ok {
			return 0, lowerUnsupportedError(e.At, "unsupported view prefix builtin '%s'", e.Name)
		}
		l.emit(ir.IRInstr{Kind: ir.IRSlicePrefix, Imm: shift, Pos: e.At})
		return 2, nil
	case "core.slice_suffix_u8", "core.slice_suffix_u16", "core.slice_suffix_i32", "core.slice_suffix_bool", "core.string_suffix":
		if total != 3 {
			return 0, fmt.Errorf("%s: %s expects view source and start argument", frontend.FormatPos(e.At), e.Name)
		}
		shift, ok := sliceViewElementShift(e.Name)
		if !ok {
			return 0, lowerUnsupportedError(e.At, "unsupported view suffix builtin '%s'", e.Name)
		}
		l.emit(ir.IRInstr{Kind: ir.IRSliceSuffix, Imm: shift, Pos: e.At})
		return 2, nil
	case "core.island_new":
		if total != 1 {
			return 0, fmt.Errorf("%s: island_new expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRIslandNew, Pos: e.At})
		return 1, nil
	case "core.island_make_u8":
		if total != 2 {
			return 0, fmt.Errorf("%s: island_make_u8 expects 2 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRIslandMakeSliceU8, Name: l.allocationNameForBuiltinCall(e.Name, e.At, allocplan.StorageExplicitIsland), Pos: e.At})
		return 2, nil
	case "core.island_make_u16":
		if total != 2 {
			return 0, fmt.Errorf("%s: island_make_u16 expects 2 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRIslandMakeSliceU16, Name: l.allocationNameForBuiltinCall(e.Name, e.At, allocplan.StorageExplicitIsland), Pos: e.At})
		return 2, nil
	case "core.island_make_i32":
		if total != 2 {
			return 0, fmt.Errorf("%s: island_make_i32 expects 2 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRIslandMakeSliceI32, Name: l.allocationNameForBuiltinCall(e.Name, e.At, allocplan.StorageExplicitIsland), Pos: e.At})
		return 2, nil
	case "core.island_make_bool":
		if total != 2 {
			return 0, fmt.Errorf("%s: island_make_bool expects 2 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRIslandMakeSliceI32, Name: l.allocationNameForBuiltinCall(e.Name, e.At, allocplan.StorageExplicitIsland), Pos: e.At})
		return 2, nil
	case "core.island_reset":
		if total != 1 {
			return 0, fmt.Errorf("%s: island_reset expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRIslandReset, Pos: e.At})
		return 1, nil
	case "core.mmio_read_i32":
		if total != 2 {
			return 0, fmt.Errorf("%s: mmio_read_i32 expects 2 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMmioReadI32, Pos: e.At})
		return 1, nil
	case "core.mmio_write_i32":
		if total != 3 {
			return 0, fmt.Errorf("%s: mmio_write_i32 expects 3 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMmioWriteI32, Pos: e.At})
		return 1, nil
	case "core.fs_exists":
		if total != 3 {
			return 0, fmt.Errorf("%s: fs_exists expects 3 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_fs_exists", ArgSlots: 3, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.net_socket_tcp4":
		if total != 1 {
			return 0, fmt.Errorf("%s: net_socket_tcp4 expects 1 argument slot", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_net_socket_tcp4", ArgSlots: 1, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.net_bind_tcp4_loopback":
		if total != 3 {
			return 0, fmt.Errorf("%s: net_bind_tcp4_loopback expects 3 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_net_bind_tcp4_loopback", ArgSlots: 3, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.net_connect_tcp4_loopback":
		if total != 3 {
			return 0, fmt.Errorf("%s: net_connect_tcp4_loopback expects 3 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_net_connect_tcp4_loopback", ArgSlots: 3, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.net_listen":
		if total != 3 {
			return 0, fmt.Errorf("%s: net_listen expects 3 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_net_listen", ArgSlots: 3, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.net_accept4":
		if total != 3 {
			return 0, fmt.Errorf("%s: net_accept4 expects 3 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_net_accept4", ArgSlots: 3, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.net_read":
		if total != 6 {
			return 0, fmt.Errorf("%s: net_read expects 6 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_net_read", ArgSlots: 6, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.net_recv":
		if total != 6 {
			return 0, fmt.Errorf("%s: net_recv expects 6 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_net_recv", ArgSlots: 6, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.net_write":
		if total != 6 {
			return 0, fmt.Errorf("%s: net_write expects 6 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_net_write", ArgSlots: 6, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.net_send":
		if total != 6 {
			return 0, fmt.Errorf("%s: net_send expects 6 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_net_send", ArgSlots: 6, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.net_epoll_create":
		if total != 1 {
			return 0, fmt.Errorf("%s: net_epoll_create expects 1 argument slot", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_net_epoll_create", ArgSlots: 1, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.net_epoll_ctl_add_read":
		if total != 3 {
			return 0, fmt.Errorf("%s: net_epoll_ctl_add_read expects 3 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_net_epoll_ctl_add_read", ArgSlots: 3, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.net_epoll_ctl_add_read_write":
		if total != 3 {
			return 0, fmt.Errorf("%s: net_epoll_ctl_add_read_write expects 3 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_net_epoll_ctl_add_read_write", ArgSlots: 3, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.net_epoll_ctl_mod_read":
		if total != 3 {
			return 0, fmt.Errorf("%s: net_epoll_ctl_mod_read expects 3 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_net_epoll_ctl_mod_read", ArgSlots: 3, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.net_epoll_ctl_mod_read_write":
		if total != 3 {
			return 0, fmt.Errorf("%s: net_epoll_ctl_mod_read_write expects 3 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_net_epoll_ctl_mod_read_write", ArgSlots: 3, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.net_epoll_ctl_delete":
		if total != 3 {
			return 0, fmt.Errorf("%s: net_epoll_ctl_delete expects 3 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_net_epoll_ctl_delete", ArgSlots: 3, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.net_epoll_wait_one":
		if total != 3 {
			return 0, fmt.Errorf("%s: net_epoll_wait_one expects 3 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_net_epoll_wait_one", ArgSlots: 3, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.net_epoll_wait_one_into":
		if total != 5 {
			return 0, fmt.Errorf("%s: net_epoll_wait_one_into expects 5 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_net_epoll_wait_one_into", ArgSlots: 5, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.net_set_nonblocking":
		if total != 2 {
			return 0, fmt.Errorf("%s: net_set_nonblocking expects 2 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_net_set_nonblocking", ArgSlots: 2, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.net_set_reuseport":
		if total != 2 {
			return 0, fmt.Errorf("%s: net_set_reuseport expects 2 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_net_set_reuseport", ArgSlots: 2, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.net_set_tcp_nodelay":
		if total != 2 {
			return 0, fmt.Errorf("%s: net_set_tcp_nodelay expects 2 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_net_set_tcp_nodelay", ArgSlots: 2, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.net_close":
		if total != 2 {
			return 0, fmt.Errorf("%s: net_close expects 2 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_net_close", ArgSlots: 2, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.load_i32":
		if total != 2 {
			return 0, fmt.Errorf("%s: load_i32 expects 2 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMemReadI32, Pos: e.At})
		return 1, nil
	case "core.store_i32":
		if total != 3 {
			return 0, fmt.Errorf("%s: store_i32 expects 3 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMemWriteI32, Pos: e.At})
		return 1, nil
	case "core.load_u8":
		if total != 2 {
			return 0, fmt.Errorf("%s: load_u8 expects 2 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMemReadU8, Pos: e.At})
		return 1, nil
	case "core.store_u8":
		if total != 3 {
			return 0, fmt.Errorf("%s: store_u8 expects 3 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMemWriteU8, Pos: e.At})
		return 1, nil
	case "core.load_ptr":
		if total != 2 {
			return 0, fmt.Errorf("%s: load_ptr expects 2 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMemReadPtr, Pos: e.At})
		return 1, nil
	case "core.store_ptr":
		if total != 3 {
			return 0, fmt.Errorf("%s: store_ptr expects 3 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMemWritePtr, Pos: e.At})
		return 1, nil
	case "core.store_arch_ptr":
		if total != 3 {
			return 0, fmt.Errorf("%s: store_arch_ptr expects 3 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMemWriteArchPtr, Pos: e.At})
		return 1, nil
	case "core.ptr_add":
		if total != 3 {
			return 0, fmt.Errorf("%s: ptr_add expects 3 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRPtrAdd, Pos: e.At})
		return 1, nil
	case "core.ctx_switch":
		if total != 3 {
			return 0, fmt.Errorf("%s: ctx_switch expects 3 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCtxSwitch, Pos: e.At})
		return 1, nil
	case "core.consent_token":
		if total != 0 {
			return 0, fmt.Errorf("%s: consent_token expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: consentTokenRuntimeSentinel, Pos: e.At})
		return 1, nil
	case "core.secret_seal_i32":
		if total != 2 {
			return 0, fmt.Errorf("%s: secret_seal_i32 expects 2 arguments", frontend.FormatPos(e.At))
		}
		// Keep the first argument (secret payload) and consume the token.
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRMulI32, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRAddI32, Pos: e.At})
		return 1, nil
	case "core.secret_unseal_i32":
		if total != 2 {
			return 0, fmt.Errorf("%s: secret_unseal_i32 expects 2 arguments", frontend.FormatPos(e.At))
		}
		// Keep the first argument (sealed payload) and consume the token.
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRMulI32, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRAddI32, Pos: e.At})
		return 1, nil
	case "core.task_group_open":
		if total != 0 {
			return 0, fmt.Errorf("%s: task_group_open expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_group_open", ArgSlots: 0, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.time_now_ms":
		if total != 0 {
			return 0, fmt.Errorf("%s: time_now_ms expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_time_now_ms", ArgSlots: 0, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.sleep_ms":
		if total != 1 {
			return 0, fmt.Errorf("%s: sleep_ms expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_sleep_ms", ArgSlots: 1, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.sleep_until":
		if total != 1 {
			return 0, fmt.Errorf("%s: sleep_until expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_sleep_until_ms", ArgSlots: 1, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.deadline_ms":
		if total != 1 {
			return 0, fmt.Errorf("%s: deadline_ms expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_deadline_ms", ArgSlots: 1, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.timer_ready":
		if total != 1 {
			return 0, fmt.Errorf("%s: timer_ready expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_timer_ready_ms", ArgSlots: 1, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.yield":
		if total != 0 {
			return 0, fmt.Errorf("%s: yield expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_yield_now", ArgSlots: 0, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.task_group_close":
		if total != 1 {
			return 0, fmt.Errorf("%s: task_group_close expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_group_close", ArgSlots: 1, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.task_group_cancel":
		if total != 1 {
			return 0, fmt.Errorf("%s: task_group_cancel expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_group_cancel", ArgSlots: 1, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.task_group_current":
		if total != 0 {
			return 0, fmt.Errorf("%s: task_group_current expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_group_current", ArgSlots: 0, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.task_group_status":
		if total != 1 {
			return 0, fmt.Errorf("%s: task_group_status expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_group_status", ArgSlots: 1, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.task_is_canceled":
		if total != 0 {
			return 0, fmt.Errorf("%s: task_is_canceled expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_is_canceled", ArgSlots: 0, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.task_checkpoint":
		if total != 0 {
			return 0, fmt.Errorf("%s: task_checkpoint expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_checkpoint", ArgSlots: 0, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.task_join_i32":
		if total != 2 {
			return 0, fmt.Errorf("%s: task_join_i32 expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_join_i32", ArgSlots: 2, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.task_join_i32_typed", "core.task_join_group_i32_typed":
		return 0, fmt.Errorf("%s: task_join_i32_typed requires try", frontend.FormatPos(e.At))
	case "core.task_join_result_i32":
		if total != 2 {
			return 0, fmt.Errorf("%s: task_join_result_i32 expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_join_result_i32", ArgSlots: 2, RetSlots: 2, Pos: e.At})
		return 2, nil
	case "core.task_join_until_i32":
		if total != 3 {
			return 0, fmt.Errorf("%s: task_join_until_i32 expects 2 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_join_until_i32", ArgSlots: 3, RetSlots: 2, Pos: e.At})
		return 2, nil
	case "core.task_poll_i32":
		if total != 2 {
			return 0, fmt.Errorf("%s: task_poll_i32 expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_poll_i32", ArgSlots: 2, RetSlots: 2, Pos: e.At})
		return 2, nil
	case "core.select2_i32":
		if total != 3 {
			return 0, fmt.Errorf("%s: select2_i32 expects 2 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_join_until_i32", ArgSlots: 3, RetSlots: 2, Pos: e.At})
		return 2, nil
	case "core.actor_dispatch":
		if total != 1 {
			return 0, fmt.Errorf("%s: actor_dispatch expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_dispatch", ArgSlots: 1, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.actor_main_entry_id":
		if total != 0 {
			return 0, fmt.Errorf("%s: actor_main_entry_id expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_main_entry_id", ArgSlots: 0, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.actor_node_connect":
		if total != 2 {
			return 0, fmt.Errorf("%s: actor_node_connect expects 2 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_node_connect", ArgSlots: 2, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.actor_node_status":
		if total != 1 {
			return 0, fmt.Errorf("%s: actor_node_status expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_node_status", ArgSlots: 1, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.send":
		if total != 2 {
			return 0, fmt.Errorf("%s: send expects 2 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_send", ArgSlots: 2, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.send_msg":
		if total != 3 {
			return 0, fmt.Errorf("%s: send_msg expects 3 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_send_msg", ArgSlots: 3, RetSlots: 1, Pos: e.At})
		return 1, nil
	case "core.recv_poll":
		if total != 0 {
			return 0, fmt.Errorf("%s: recv_poll expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_recv_poll", ArgSlots: 0, RetSlots: 2, Pos: e.At})
		return 2, nil
	case "core.recv_until":
		if total != 1 {
			return 0, fmt.Errorf("%s: recv_until expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_recv_until", ArgSlots: 1, RetSlots: 2, Pos: e.At})
		return 2, nil
	case "core.recv_msg_until":
		if total != 1 {
			return 0, fmt.Errorf("%s: recv_msg_until expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_recv_msg_until", ArgSlots: 1, RetSlots: 3, Pos: e.At})
		return 3, nil
	default:
		sig, ok := l.funcs[e.Name]
		if !ok {
			return 0, fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(e.At), e.Name)
		}
		writebacks := []inoutWriteback(nil)
		if sig.ThrowsType == "" {
			var err error
			writebacks, err = l.collectInoutWritebacks(e.Args, sig.ParamOwnership)
			if err != nil {
				return 0, err
			}
		}
		abiReturnSlots := sig.ReturnSlots + inoutWritebackSlotCount(writebacks)
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: e.Name, ArgSlots: total, RetSlots: abiReturnSlots, Pos: e.At})
		l.emitInoutWritebacks(writebacks, e.At)
		return sig.ReturnSlots, nil
	}
}
