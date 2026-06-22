package wasm32_web

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"sort"

	"tetra_language/compiler/internal/ir"
)

func compileFunctionWithControlFlow(
	fn Function,
	data *dataBuilder,
	funcIndexByName map[string]uint32,
	consoleLogImport int,
	heapGlobalIndex uint32,
	tempPtr int,
	tempLen int,
	tempIdx int,
	tempVal int,
	tempByteLen int,
	pcLocal int,
	body *bytes.Buffer,
) ([]byte, error) {
	labels, heights, err := verifyControlFlowStackModel(fn)
	if err != nil {
		return nil, err
	}
	starts := map[int]struct{}{0: {}}
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRLabel {
			starts[i] = struct{}{}
		}
		if (instr.Kind == ir.IRJmp || instr.Kind == ir.IRJmpIfZero || instr.Kind == ir.IRReturn) &&
			i+1 < len(fn.Instrs) {
			starts[i+1] = struct{}{}
		}
	}
	var blockStarts []int
	for idx := range starts {
		blockStarts = append(blockStarts, idx)
	}
	sort.Ints(blockStarts)
	type blockRange struct {
		start int
		end   int
	}
	blocks := make([]blockRange, 0, len(blockStarts))
	blockByStart := make(map[int]int, len(blockStarts))
	for i, start := range blockStarts {
		end := len(fn.Instrs)
		if i+1 < len(blockStarts) {
			end = blockStarts[i+1]
		}
		blockByStart[start] = len(blocks)
		blocks = append(blocks, blockRange{start: start, end: end})
	}
	labelToBlock := make(map[int]int, len(labels))
	for label, idx := range labels {
		bid, ok := blockByStart[idx]
		if !ok {
			return nil, fmt.Errorf(
				"wasm backend: internal control-flow block mapping failure in '%s'",
				fn.Name,
			)
		}
		labelToBlock[label] = bid
		if heights[idx] != 0 {
			return nil, fmt.Errorf(
				"wasm backend: unsupported non-zero stack at label %d in function '%s'",
				label,
				fn.Name,
			)
		}
	}

	writeI32Const(body, 0)
	body.WriteByte(0x21)
	writeULEB(body, uint32(pcLocal))
	body.WriteByte(0x02)
	body.WriteByte(0x40)
	body.WriteByte(0x03)
	body.WriteByte(0x40)

	for bi, block := range blocks {
		body.WriteByte(0x20)
		writeULEB(body, uint32(pcLocal))
		writeI32Const(body, int32(bi))
		body.WriteByte(0x46)
		body.WriteByte(0x04)
		body.WriteByte(0x40)

		stackDepth := 0
		terminated := false
		for i := block.start; i < block.end; i++ {
			instr := fn.Instrs[i]
			switch instr.Kind {
			case ir.IRLabel:
				continue
			case ir.IRJmp:
				if stackDepth != 0 {
					return nil, fmt.Errorf(
						"wasm backend: unsupported non-zero stack at jump in function '%s'",
						fn.Name,
					)
				}
				target, ok := labelToBlock[instr.Label]
				if !ok {
					return nil, fmt.Errorf(
						"wasm backend: unknown jump label %d in function '%s'",
						instr.Label,
						fn.Name,
					)
				}
				writeI32Const(body, int32(target))
				body.WriteByte(0x21)
				writeULEB(body, uint32(pcLocal))
				body.WriteByte(0x0c)
				writeULEB(body, 1)
				terminated = true
			case ir.IRJmpIfZero:
				if stackDepth < 1 {
					return nil, fmt.Errorf(
						"wasm backend: stack underflow in '%s' (jmp_if_zero)",
						fn.Name,
					)
				}
				stackDepth--
				if stackDepth != 0 {
					return nil, fmt.Errorf(
						"wasm backend: unsupported non-zero stack after conditional jump in function '%s'",
						fn.Name,
					)
				}
				target, ok := labelToBlock[instr.Label]
				if !ok {
					return nil, fmt.Errorf(
						"wasm backend: unknown jump label %d in function '%s'",
						instr.Label,
						fn.Name,
					)
				}
				body.WriteByte(0x45)
				body.WriteByte(0x04)
				body.WriteByte(0x40)
				writeI32Const(body, int32(target))
				body.WriteByte(0x21)
				writeULEB(body, uint32(pcLocal))
				body.WriteByte(0x0c)
				writeULEB(body, 2)
				body.WriteByte(0x0b)
				nextBlock := bi + 1
				if nextBlock >= len(blocks) {
					return nil, fmt.Errorf(
						"wasm backend: conditional branch falls off end in function '%s'",
						fn.Name,
					)
				}
				writeI32Const(body, int32(nextBlock))
				body.WriteByte(0x21)
				writeULEB(body, uint32(pcLocal))
				body.WriteByte(0x0c)
				writeULEB(body, 1)
				terminated = true
			case ir.IRReturn:
				if stackDepth != fn.ReturnSlots {
					return nil, fmt.Errorf(
						"wasm backend: return stack mismatch in '%s': got %d want %d",
						fn.Name,
						stackDepth,
						fn.ReturnSlots,
					)
				}
				body.WriteByte(0x0f)
				stackDepth = 0
				terminated = true
			default:
				nextDepth, err := emitWebNonControlInstr(
					body,
					fn,
					instr,
					data,
					funcIndexByName,
					consoleLogImport,
					heapGlobalIndex,
					tempPtr,
					tempLen,
					tempIdx,
					tempVal,
					tempByteLen,
					stackDepth,
				)
				if err != nil {
					return nil, err
				}
				stackDepth = nextDepth
			}
			if terminated {
				break
			}
		}
		if !terminated {
			if stackDepth != 0 {
				return nil, fmt.Errorf(
					"wasm backend: unsupported non-zero stack at block fallthrough in function '%s'",
					fn.Name,
				)
			}
			if bi+1 >= len(blocks) {
				if fn.ReturnSlots == 0 {
					body.WriteByte(0x0f)
				} else {
					return nil, fmt.Errorf(
						"wasm backend: function '%s' falls off end in control-flow mode",
						fn.Name,
					)
				}
			} else {
				writeI32Const(body, int32(bi+1))
				body.WriteByte(0x21)
				writeULEB(body, uint32(pcLocal))
				body.WriteByte(0x0c)
				writeULEB(body, 1)
			}
		}
		body.WriteByte(0x0b)
	}
	body.WriteByte(0x00)
	body.WriteByte(0x0b)
	body.WriteByte(0x0b)
	body.WriteByte(0x00) // unreachable function fallthrough for multi-result funcs
	body.WriteByte(0x0b)
	return body.Bytes(), nil
}

func emitWebNonControlInstr(
	body *bytes.Buffer,
	fn Function,
	instr ir.IRInstr,
	data *dataBuilder,
	funcIndexByName map[string]uint32,
	consoleLogImport int,
	heapGlobalIndex uint32,
	tempPtr int,
	tempLen int,
	tempIdx int,
	tempVal int,
	tempByteLen int,
	stackDepth int,
) (int, error) {
	pop := func(n int, opname string) error {
		if stackDepth < n {
			return fmt.Errorf("wasm backend: stack underflow in '%s' (%s)", fn.Name, opname)
		}
		stackDepth -= n
		return nil
	}
	push := func(n int) { stackDepth += n }

	switch instr.Kind {
	case ir.IRStrLit:
		dataOff := data.addString(instr.Str)
		writeI32Const(body, int32(dataBase+dataOff))
		writeI32Const(body, int32(len(instr.Str)))
		push(2)
	case ir.IRConstI32:
		writeI32Const(body, instr.Imm)
		push(1)
	case ir.IRLoadLocal:
		body.WriteByte(0x20)
		writeULEB(body, uint32(instr.Local))
		push(1)
	case ir.IRStoreLocal:
		if err := pop(1, "store_local"); err != nil {
			return 0, err
		}
		body.WriteByte(0x21)
		writeULEB(body, uint32(instr.Local))
	case ir.IRLoadGlobal:
		globalIndex, err := wasmDataGlobalIndex(fn.Name, heapGlobalIndex, instr.Local)
		if err != nil {
			return 0, err
		}
		body.WriteByte(0x23)
		writeULEB(body, globalIndex)
		push(1)
	case ir.IRStoreGlobal:
		if err := pop(1, "store_global"); err != nil {
			return 0, err
		}
		globalIndex, err := wasmDataGlobalIndex(fn.Name, heapGlobalIndex, instr.Local)
		if err != nil {
			return 0, err
		}
		body.WriteByte(0x24)
		writeULEB(body, globalIndex)
	case ir.IRAddI32:
		if err := pop(2, "add_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x6a)
		push(1)
	case ir.IRSubI32:
		if err := pop(2, "sub_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x6b)
		push(1)
	case ir.IRNegI32:
		if err := pop(1, "neg_i32"); err != nil {
			return 0, err
		}
		writeI32Const(body, -1)
		body.WriteByte(0x6c)
		push(1)
	case ir.IRMulI32:
		if err := pop(2, "mul_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x6c)
		push(1)
	case ir.IRDivI32:
		if err := pop(2, "div_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x6d)
		push(1)
	case ir.IRModI32:
		if err := pop(2, "mod_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x6f)
		push(1)
	case ir.IRCmpEqI32:
		if err := pop(2, "cmp_eq_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x46)
		push(1)
	case ir.IRCmpLtI32:
		if err := pop(2, "cmp_lt_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x48)
		push(1)
	case ir.IRCmpGtI32:
		if err := pop(2, "cmp_gt_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x4a)
		push(1)
	case ir.IRCmpGeI32:
		if err := pop(2, "cmp_ge_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x4e)
		push(1)
	case ir.IRCmpLeI32:
		if err := pop(2, "cmp_le_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x4c)
		push(1)
	case ir.IRCmpNeI32:
		if err := pop(2, "cmp_ne_i32"); err != nil {
			return 0, err
		}
		body.WriteByte(0x47)
		push(1)
	case ir.IRCall:
		if err := pop(instr.ArgSlots, "call"); err != nil {
			return 0, err
		}
		target, ok := funcIndexByName[instr.Name]
		if !ok {
			return 0, fmt.Errorf(
				"wasm backend: function '%s' calls unsupported symbol '%s'",
				fn.Name,
				instr.Name,
			)
		}
		body.WriteByte(0x10)
		writeULEB(body, target)
		push(instr.RetSlots)
	case ir.IRSymAddr:
		writeI32Const(body, int32(wasmSymbolToken(instr.Name)))
		push(1)
	case ir.IRWrite:
		if err := pop(2, "write"); err != nil {
			return 0, err
		}
		body.WriteByte(0x10)
		writeULEB(body, uint32(consoleLogImport))
	case ir.IRMakeSliceU8, ir.IRMakeSliceU16, ir.IRMakeSliceI32:
		if err := pop(1, "make_slice"); err != nil {
			return 0, err
		}
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempLen))
		emitWasmMakeSliceContract(
			body,
			instr.Kind,
			heapGlobalIndex,
			tempPtr,
			tempLen,
			tempByteLen,
			tempVal,
		)
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempPtr))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempLen))
		push(2)
	case ir.IRStackSliceU8, ir.IRStackSliceU16, ir.IRStackSliceI32:
		if !wasmZeroStackSliceSentinel(instr) {
			return 0, wasmUnsupportedInstrError(fn.Name, instr.Kind)
		}
		if err := pop(1, "zero_stack_slice"); err != nil {
			return 0, err
		}
		emitWasmZeroSliceSentinel(body)
		push(2)
	case ir.IRRawSliceFromParts:
		if err := pop(3, "raw_slice_from_parts"); err != nil {
			return 0, err
		}
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempByteLen))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempLen))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempPtr))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempPtr))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempLen))
		push(2)
	case ir.IRSliceWindow, ir.IRSlicePrefix, ir.IRSliceSuffix:
		popSlots := 3
		if instr.Kind == ir.IRSliceWindow {
			popSlots = 4
		}
		if err := pop(popSlots, "slice_view"); err != nil {
			return 0, err
		}
		emitWasmSliceView(body, instr.Kind, byte(instr.Imm), tempPtr, tempLen, tempIdx, tempVal)
		push(2)
	case ir.IRIslandNew:
		if err := pop(1, "island_new"); err != nil {
			return 0, err
		}
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempByteLen))
		emitHeapBumpAndGrow(body, heapGlobalIndex, tempPtr, tempByteLen, tempVal)
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempPtr))
		push(1)
	case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32:
		if err := pop(2, "island_make_slice"); err != nil {
			return 0, err
		}
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempLen))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempPtr))
		emitWasmMakeSliceContract(
			body,
			instr.Kind,
			heapGlobalIndex,
			tempPtr,
			tempLen,
			tempByteLen,
			tempVal,
		)
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempPtr))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempLen))
		push(2)
	case ir.IRIslandFree:
		if err := pop(1, "island_free"); err != nil {
			return 0, err
		}
	case ir.IRIslandReset:
		if err := pop(1, "island_reset"); err != nil {
			return 0, err
		}
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempPtr))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempPtr))
		push(1)
	case ir.IRIndexLoadI32, ir.IRIndexLoadU8, ir.IRIndexLoadU16,
		ir.IRIndexLoadI32Unchecked, ir.IRIndexLoadU8Unchecked, ir.IRIndexLoadU16Unchecked:
		if err := pop(3, "index_load"); err != nil {
			return 0, err
		}
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempIdx))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempLen))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempPtr))
		checked := instr.Kind == ir.IRIndexLoadI32 || instr.Kind == ir.IRIndexLoadU8 ||
			instr.Kind == ir.IRIndexLoadU16
		if checked {
			body.WriteByte(0x20)
			writeULEB(body, uint32(tempIdx))
			body.WriteByte(0x20)
			writeULEB(body, uint32(tempLen))
			body.WriteByte(0x4f)
			body.WriteByte(0x04)
			body.WriteByte(0x40)
			body.WriteByte(0x00)
			body.WriteByte(0x0b)
		}
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempPtr))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempIdx))
		switch instr.Kind {
		case ir.IRIndexLoadI32, ir.IRIndexLoadI32Unchecked:
			writeI32Const(body, 2)
			body.WriteByte(0x74)
		case ir.IRIndexLoadU16, ir.IRIndexLoadU16Unchecked:
			writeI32Const(body, 1)
			body.WriteByte(0x74)
		}
		body.WriteByte(0x6a)
		switch instr.Kind {
		case ir.IRIndexLoadI32, ir.IRIndexLoadI32Unchecked:
			body.WriteByte(0x28)
			writeULEB(body, 2)
			writeULEB(body, 0)
		case ir.IRIndexLoadU16, ir.IRIndexLoadU16Unchecked:
			body.WriteByte(0x2f)
			writeULEB(body, 1)
			writeULEB(body, 0)
		default:
			body.WriteByte(0x2d)
			writeULEB(body, 0)
			writeULEB(body, 0)
		}
		push(1)
	case ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16:
		if err := pop(4, "index_store"); err != nil {
			return 0, err
		}
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempVal))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempIdx))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempLen))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempPtr))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempIdx))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempLen))
		body.WriteByte(0x4f)
		body.WriteByte(0x04)
		body.WriteByte(0x40)
		body.WriteByte(0x00)
		body.WriteByte(0x0b)
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempPtr))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempIdx))
		switch instr.Kind {
		case ir.IRIndexStoreI32:
			writeI32Const(body, 2)
			body.WriteByte(0x74)
		case ir.IRIndexStoreU16:
			writeI32Const(body, 1)
			body.WriteByte(0x74)
		}
		body.WriteByte(0x6a)
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempVal))
		switch instr.Kind {
		case ir.IRIndexStoreI32:
			body.WriteByte(0x36)
			writeULEB(body, 2)
			writeULEB(body, 0)
		case ir.IRIndexStoreU16:
			body.WriteByte(0x3b)
			writeULEB(body, 1)
			writeULEB(body, 0)
		default:
			body.WriteByte(0x3a)
			writeULEB(body, 0)
			writeULEB(body, 0)
		}
	default:
		return 0, wasmUnsupportedInstrError(fn.Name, instr.Kind)
	}
	return stackDepth, nil
}

func verifyControlFlowStackModel(fn Function) (map[int]int, []int, error) {
	labels := make(map[int]int)
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRLabel {
			labels[instr.Label] = i
		}
	}
	heights := make([]int, len(fn.Instrs))
	seen := make([]bool, len(fn.Instrs))
	type state struct {
		idx    int
		height int
	}
	work := []state{{idx: 0, height: 0}}
	for len(work) > 0 {
		cur := work[len(work)-1]
		work = work[:len(work)-1]
		if cur.idx < 0 || cur.idx >= len(fn.Instrs) {
			continue
		}
		if fn.Instrs[cur.idx].Kind == ir.IRLabel && cur.height != 0 {
			return nil, nil, fmt.Errorf(
				"wasm backend: unsupported non-zero stack at label %d in function '%s'",
				fn.Instrs[cur.idx].Label,
				fn.Name,
			)
		}
		if seen[cur.idx] {
			if heights[cur.idx] != cur.height {
				return nil, nil, fmt.Errorf(
					"wasm backend: inconsistent stack height at instr %d in '%s'",
					cur.idx,
					fn.Name,
				)
			}
			continue
		}
		seen[cur.idx] = true
		heights[cur.idx] = cur.height
		pop, push, ok := wasmStackEffect(fn.Instrs[cur.idx])
		if !ok {
			return nil, nil, wasmUnsupportedInstrError(fn.Name, fn.Instrs[cur.idx].Kind)
		}
		if cur.height < pop {
			return nil, nil, fmt.Errorf(
				"wasm backend: stack underflow at instr %d in '%s'",
				cur.idx,
				fn.Name,
			)
		}
		nextHeight := cur.height - pop + push
		switch fn.Instrs[cur.idx].Kind {
		case ir.IRReturn:
			if cur.height != fn.ReturnSlots {
				return nil, nil, fmt.Errorf(
					"wasm backend: return stack mismatch at instr %d in '%s'",
					cur.idx,
					fn.Name,
				)
			}
		case ir.IRJmp:
			work = append(work, state{idx: labels[fn.Instrs[cur.idx].Label], height: nextHeight})
		case ir.IRJmpIfZero:
			work = append(work, state{idx: labels[fn.Instrs[cur.idx].Label], height: nextHeight})
			work = append(work, state{idx: cur.idx + 1, height: nextHeight})
		default:
			work = append(work, state{idx: cur.idx + 1, height: nextHeight})
		}
	}
	return labels, heights, nil
}

func wasmStackEffect(instr ir.IRInstr) (int, int, bool) {
	switch instr.Kind {
	case ir.IRWrite:
		return 2, 0, true
	case ir.IRStrLit:
		return 0, 2, true
	case ir.IRConstI32, ir.IRLoadLocal, ir.IRLoadGlobal, ir.IRSymAddr:
		return 0, 1, true
	case ir.IRStoreLocal, ir.IRStoreGlobal:
		return 1, 0, true
	case ir.IRAddI32, ir.IRSubI32, ir.IRCmpEqI32, ir.IRCmpLtI32,
		ir.IRMulI32, ir.IRDivI32, ir.IRModI32, ir.IRCmpGtI32,
		ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		return 2, 1, true
	case ir.IRNegI32:
		return 1, 1, true
	case ir.IRCall:
		return instr.ArgSlots, instr.RetSlots, true
	case ir.IRLabel, ir.IRJmp:
		return 0, 0, true
	case ir.IRJmpIfZero:
		return 1, 0, true
	case ir.IRReturn:
		return 0, 0, true
	case ir.IRMakeSliceU8, ir.IRMakeSliceU16, ir.IRMakeSliceI32,
		ir.IRStackSliceU8, ir.IRStackSliceU16, ir.IRStackSliceI32:
		return 1, 2, true
	case ir.IRRawSliceFromParts:
		return 3, 2, true
	case ir.IRSliceWindow:
		return 4, 2, true
	case ir.IRSlicePrefix, ir.IRSliceSuffix:
		return 3, 2, true
	case ir.IRIslandNew:
		return 1, 1, true
	case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32:
		return 2, 2, true
	case ir.IRIslandFree:
		return 1, 0, true
	case ir.IRIslandReset:
		return 1, 1, true
	case ir.IRIndexLoadI32, ir.IRIndexLoadU8, ir.IRIndexLoadU16,
		ir.IRIndexLoadI32Unchecked, ir.IRIndexLoadU8Unchecked, ir.IRIndexLoadU16Unchecked:
		return 3, 1, true
	case ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16:
		return 4, 0, true
	case ir.IRMemReadI32Offset, ir.IRMemReadU8Offset, ir.IRMemReadPtrOffset:
		return 3, 1, true
	case ir.IRMemWriteI32Offset, ir.IRMemWriteU8Offset, ir.IRMemWritePtrOffset:
		return 4, 1, true
	default:
		return 0, 0, false
	}
}

func wasmUnsupportedInstrError(fnName string, kind ir.IRInstrKind) error {
	return fmt.Errorf("wasm backend: unsupported IR instruction %d in function '%s'", kind, fnName)
}

func wasmZeroStackSliceSentinel(instr ir.IRInstr) bool {
	switch instr.Kind {
	case ir.IRStackSliceU8, ir.IRStackSliceU16, ir.IRStackSliceI32:
		return instr.Local < 0 && instr.ArgSlots == 0 && instr.Imm == 0
	default:
		return false
	}
}

func emitWasmZeroSliceSentinel(body *bytes.Buffer) {
	body.WriteByte(0x1a) // drop length operand
	writeI32Const(body, 0)
	writeI32Const(body, 0)
}

func wasmDataGlobalIndex(fnName string, heapGlobalIndex uint32, slot int) (uint32, error) {
	if slot < 0 {
		return 0, wasmNegativeGlobalSlotError(fnName, slot)
	}
	return heapGlobalIndex + 1 + uint32(slot), nil
}

func wasmGlobalInitializers(globalSlots int, dataPrefix [][]byte) ([]int32, error) {
	inits := make([]int32, globalSlots)
	for i := 0; i < globalSlots && i < len(dataPrefix); i++ {
		if len(dataPrefix[i]) == 0 {
			continue
		}
		if len(dataPrefix[i]) < 4 {
			return nil, fmt.Errorf(
				"wasm backend: global data slot %d is %d bytes, want at least 4",
				i,
				len(dataPrefix[i]),
			)
		}
		inits[i] = int32(binary.LittleEndian.Uint32(dataPrefix[i][:4]))
	}
	return inits, nil
}

func wasmNegativeGlobalSlotError(fnName string, slot int) error {
	return fmt.Errorf("wasm backend: negative global slot %d in function '%s'", slot, fnName)
}

func validateWasmObjectFunctions(funcs []Function) error {
	functionNames := make(map[string]struct{}, len(funcs))
	for _, fn := range funcs {
		if err := validateWasmFunctionMetadata(
			functionNames,
			fn.Name,
			fn.ParamSlots,
			fn.LocalSlots,
			fn.ReturnSlots,
		); err != nil {
			return err
		}
	}
	return nil
}

func validateWasmFunctionMetadata(
	seen map[string]struct{},
	name string,
	paramSlots int,
	localSlots int,
	returnSlots int,
) error {
	if name == "" {
		return fmt.Errorf("wasm backend: function name is empty")
	}
	if _, ok := seen[name]; ok {
		return fmt.Errorf("wasm backend: duplicate function '%s'", name)
	}
	seen[name] = struct{}{}
	if paramSlots < 0 || localSlots < 0 || returnSlots < 0 || localSlots < paramSlots {
		return fmt.Errorf(
			"wasm backend: function '%s' has invalid slots params=%d locals=%d returns=%d",
			name,
			paramSlots,
			localSlots,
			returnSlots,
		)
	}
	return nil
}

func validateWasmObjectGlobalSlots(obj *Object) error {
	if obj.GlobalSlots < 0 {
		return fmt.Errorf("wasm backend: invalid global slot count %d", obj.GlobalSlots)
	}
	for _, fn := range obj.Functions {
		for _, instr := range fn.Instrs {
			if instr.Kind != ir.IRLoadGlobal && instr.Kind != ir.IRStoreGlobal {
				continue
			}
			if instr.Local < 0 {
				return wasmNegativeGlobalSlotError(fn.Name, instr.Local)
			}
			if instr.Local >= obj.GlobalSlots {
				return fmt.Errorf(
					"wasm backend: global slot %d in function '%s' exceeds object global slot count %d",
					instr.Local,
					fn.Name,
					obj.GlobalSlots,
				)
			}
		}
	}
	return nil
}

func validateWasmObjectLabels(funcs []Function) error {
	for _, fn := range funcs {
		if err := validateWasmLabelMetadata(fn.Name, fn.Instrs); err != nil {
			return err
		}
	}
	return nil
}

func validateWasmLabelMetadata(fnName string, instrs []ir.IRInstr) error {
	labels := make(map[int]struct{})
	for _, instr := range instrs {
		if instr.Kind != ir.IRLabel {
			continue
		}
		if instr.Label < 0 {
			return fmt.Errorf("wasm backend: function '%s' negative label %d", fnName, instr.Label)
		}
		if _, exists := labels[instr.Label]; exists {
			return fmt.Errorf("wasm backend: function '%s' duplicate label %d", fnName, instr.Label)
		}
		labels[instr.Label] = struct{}{}
	}
	for _, instr := range instrs {
		if instr.Kind != ir.IRJmp && instr.Kind != ir.IRJmpIfZero {
			continue
		}
		if instr.Label < 0 {
			return fmt.Errorf("wasm backend: function '%s' negative label %d", fnName, instr.Label)
		}
		if _, ok := labels[instr.Label]; !ok {
			return fmt.Errorf("wasm backend: function '%s' unknown label %d", fnName, instr.Label)
		}
	}
	return nil
}

func validateWasmObjectLocalSlots(funcs []Function) error {
	for _, fn := range funcs {
		for _, instr := range fn.Instrs {
			if instr.Kind == ir.IRLoadLocal || instr.Kind == ir.IRStoreLocal {
				if err := validateWasmLocalSlot(fn.Name, fn.LocalSlots, instr.Local); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateWasmLocalSlot(fnName string, localSlots int, slot int) error {
	if slot < 0 || slot >= localSlots {
		return fmt.Errorf(
			"wasm backend: function '%s' local slot %d out of bounds (locals=%d)",
			fnName,
			slot,
			localSlots,
		)
	}
	return nil
}

func validateWasmObjectCalls(funcs []Function) error {
	functionSigs := make(map[string]wasmFunctionSignature, len(funcs))
	for _, fn := range funcs {
		functionSigs[fn.Name] = wasmFunctionSignature{
			ParamSlots:  fn.ParamSlots,
			ReturnSlots: fn.ReturnSlots,
		}
	}
	for _, fn := range funcs {
		for _, instr := range fn.Instrs {
			if instr.Kind == ir.IRCall {
				if err := validateWasmCallMetadata(fn.Name, instr, functionSigs); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateWasmCallMetadata(
	fnName string,
	instr ir.IRInstr,
	functionSigs map[string]wasmFunctionSignature,
) error {
	if instr.Name == "" {
		return fmt.Errorf("wasm backend: function '%s' call is missing target name", fnName)
	}
	if instr.ArgSlots < 0 || instr.RetSlots < 0 {
		return fmt.Errorf(
			"wasm backend: function '%s' call %q has negative ABI slots args=%d rets=%d",
			fnName,
			instr.Name,
			instr.ArgSlots,
			instr.RetSlots,
		)
	}
	sig, ok := functionSigs[instr.Name]
	if !ok {
		sig, ok = wasmSurfaceImportSignature(instr.Name)
		if !ok {
			return fmt.Errorf(
				"wasm backend: function '%s' calls unsupported symbol '%s'",
				fnName,
				instr.Name,
			)
		}
	}
	if instr.ArgSlots != sig.ParamSlots || instr.RetSlots != sig.ReturnSlots {
		return fmt.Errorf(
			"wasm backend: function '%s' call %q ABI mismatch args=%d rets=%d want args=%d rets=%d",
			fnName,
			instr.Name,
			instr.ArgSlots,
			instr.RetSlots,
			sig.ParamSlots,
			sig.ReturnSlots,
		)
	}
	return nil
}

func validateWasmObjectSymbolTokens(funcs []Function) error {
	symbolTokens := make(map[uint32]string)
	for _, fn := range funcs {
		for _, instr := range fn.Instrs {
			if instr.Kind == ir.IRSymAddr {
				if err := validateWasmSymbolToken(symbolTokens, instr.Name); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateWasmSymbolToken(seen map[uint32]string, name string) error {
	if name == "" {
		return fmt.Errorf("wasm backend: symbol address is missing name")
	}
	token := wasmSymbolToken(name)
	if previous, ok := seen[token]; ok {
		if previous != name {
			return fmt.Errorf(
				"wasm backend: symbol address token collision: %q and %q both map to 0x%08x",
				previous,
				name,
				token,
			)
		}
		return nil
	}
	seen[token] = name
	return nil
}

var wasmSymbolTokenHash = func(name string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return h.Sum32()
}

func wasmSymbolToken(name string) uint32 {
	return wasmSymbolTokenHash(name)
}

func emitWasmMakeSliceContract(
	body *bytes.Buffer,
	kind ir.IRInstrKind,
	heapGlobalIndex uint32,
	tempPtr int,
	tempLen int,
	tempByteLen int,
	tempVal int,
) {
	emitWasmLocalNonNegativeCheck(body, tempLen)
	body.WriteByte(0x20) // local.get tempLen
	writeULEB(body, uint32(tempLen))
	body.WriteByte(0x45) // i32.eqz
	body.WriteByte(0x04) // if
	body.WriteByte(0x40)
	writeI32Const(body, 0)
	body.WriteByte(0x21) // local.set tempPtr
	writeULEB(body, uint32(tempPtr))
	body.WriteByte(0x05) // else
	if max, ok := wasmMakeSliceMaxElements(kind); ok {
		body.WriteByte(0x20) // local.get tempLen
		writeULEB(body, uint32(tempLen))
		writeI32Const(body, max)
		body.WriteByte(0x4a) // i32.gt_s
		emitWasmTrapIf(body)
	}
	body.WriteByte(0x20) // local.get tempLen
	writeULEB(body, uint32(tempLen))
	if shift := wasmMakeSliceShift(kind); shift > 0 {
		writeI32Const(body, int32(shift))
		body.WriteByte(0x74) // i32.shl
	}
	body.WriteByte(0x21) // local.set tempByteLen
	writeULEB(body, uint32(tempByteLen))
	emitHeapBumpAndGrow(body, heapGlobalIndex, tempPtr, tempByteLen, tempVal)
	body.WriteByte(0x0b) // end if
}

func wasmMakeSliceShift(kind ir.IRInstrKind) byte {
	switch kind {
	case ir.IRMakeSliceU16, ir.IRIslandMakeSliceU16:
		return 1
	case ir.IRMakeSliceI32, ir.IRIslandMakeSliceI32:
		return 2
	default:
		return 0
	}
}

func wasmMakeSliceMaxElements(kind ir.IRInstrKind) (int32, bool) {
	switch kind {
	case ir.IRMakeSliceU16, ir.IRIslandMakeSliceU16:
		return 2147483647 / 2, true
	case ir.IRMakeSliceI32, ir.IRIslandMakeSliceI32:
		return 2147483647 / 4, true
	default:
		return 0, false
	}
}

func emitWasmSliceView(
	body *bytes.Buffer,
	kind ir.IRInstrKind,
	shift byte,
	tempPtr int,
	tempLen int,
	tempIdx int,
	tempVal int,
) {
	switch kind {
	case ir.IRSliceWindow:
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempVal))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempIdx))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempLen))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempPtr))
		emitWasmLocalNonNegativeCheck(body, tempIdx)
		emitWasmLocalNonNegativeCheck(body, tempVal)
		emitWasmGreaterThanTrap(body, tempIdx, tempLen)
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempVal))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempLen))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempIdx))
		body.WriteByte(0x6b)
		body.WriteByte(0x4a)
		emitWasmTrapIf(body)
		emitWasmWindowResult(body, shift, tempPtr, tempIdx)
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempVal))
	case ir.IRSlicePrefix:
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempVal))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempLen))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempPtr))
		emitWasmLocalNonNegativeCheck(body, tempVal)
		emitWasmGreaterThanTrap(body, tempVal, tempLen)
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempPtr))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempVal))
	case ir.IRSliceSuffix:
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempIdx))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempLen))
		body.WriteByte(0x21)
		writeULEB(body, uint32(tempPtr))
		emitWasmLocalNonNegativeCheck(body, tempIdx)
		emitWasmGreaterThanTrap(body, tempIdx, tempLen)
		emitWasmWindowResult(body, shift, tempPtr, tempIdx)
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempLen))
		body.WriteByte(0x20)
		writeULEB(body, uint32(tempIdx))
		body.WriteByte(0x6b)
	}
}

func emitWasmLocalNonNegativeCheck(body *bytes.Buffer, local int) {
	body.WriteByte(0x20)
	writeULEB(body, uint32(local))
	writeI32Const(body, 0)
	body.WriteByte(0x48)
	emitWasmTrapIf(body)
}

func emitWasmGreaterThanTrap(body *bytes.Buffer, left int, right int) {
	body.WriteByte(0x20)
	writeULEB(body, uint32(left))
	body.WriteByte(0x20)
	writeULEB(body, uint32(right))
	body.WriteByte(0x4a)
	emitWasmTrapIf(body)
}

func emitWasmWindowResult(body *bytes.Buffer, shift byte, tempPtr int, tempIdx int) {
	body.WriteByte(0x20)
	writeULEB(body, uint32(tempPtr))
	body.WriteByte(0x20)
	writeULEB(body, uint32(tempIdx))
	if shift > 0 {
		writeI32Const(body, int32(shift))
		body.WriteByte(0x74)
	}
	body.WriteByte(0x6a)
}

func emitWasmTrapIf(body *bytes.Buffer) {
	body.WriteByte(0x04)
	body.WriteByte(0x40)
	body.WriteByte(0x00)
	body.WriteByte(0x0b)
}

func writeSection(dst *bytes.Buffer, id byte, fn func(*bytes.Buffer)) {
	var sec bytes.Buffer
	fn(&sec)
	dst.WriteByte(id)
	writeULEB(dst, uint32(sec.Len()))
	dst.Write(sec.Bytes())
}

func writeName(dst *bytes.Buffer, s string) {
	writeULEB(dst, uint32(len(s)))
	dst.WriteString(s)
}

func writeI32Const(dst *bytes.Buffer, v int32) {
	dst.WriteByte(0x41)
	writeSLEB32(dst, v)
}

func writeULEB(dst *bytes.Buffer, v uint32) {
	var tmp [binary.MaxVarintLen32]byte
	n := binary.PutUvarint(tmp[:], uint64(v))
	dst.Write(tmp[:n])
}

func writeSLEB32(dst *bytes.Buffer, v int32) {
	x := int64(v)
	for {
		b := byte(x & 0x7f)
		x >>= 7
		signSet := (b & 0x40) != 0
		done := (x == 0 && !signSet) || (x == -1 && signSet)
		if !done {
			b |= 0x80
		}
		dst.WriteByte(b)
		if done {
			return
		}
	}
}
