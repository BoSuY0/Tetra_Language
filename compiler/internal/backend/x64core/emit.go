package x64core

import (
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/backend/x64obj"
	"tetra_language/compiler/internal/ir"
)

type labelPatch struct {
	at    int
	label int
}

func NewEmitFunc(abi x64abi.ABI) x64obj.EmitFunc {
	return func(
		e *x64.Emitter,
		fn ir.IRFunc,
		dataBlobs *[][]byte,
		leaPatches *[]x64obj.LeaPatch,
		callPatches *[]x64obj.CallPatch,
		importPatches *[]x64obj.ImportPatch,
		opt x64.CodegenOptions,
	) error {
		if abi == nil {
			return fmt.Errorf("missing ABI")
		}
		if e == nil {
			return fmt.Errorf("missing emitter")
		}
		if dataBlobs == nil || leaPatches == nil || callPatches == nil {
			return fmt.Errorf("missing patches buffers")
		}

		labelOffsets := make(map[int]int)
		var patches []labelPatch
		stackDepth := 0
		nextInternalLabel := -1

		newInternalLabel := func() int {
			id := nextInternalLabel
			nextInternalLabel--
			return id
		}

		pop := func(n int) error {
			if stackDepth < n {
				return fmt.Errorf("stack underflow in function '%s'", fn.Name)
			}
			stackDepth -= n
			return nil
		}
		push := func(n int) { stackDepth += n }

		e.PushRbp()
		e.MovRbpRsp()
		localSize := x64.AlignStackSize(fn.LocalSlots * 8)
		if localSize > 0 {
			e.SubRspImm32(int32(localSize))
		}
		abi.SpillParams(e, fn)
		for i := fn.ParamSlots; i < fn.LocalSlots; i++ {
			off := -int32((i + 1) * 8)
			e.MovMem64RbpDispImm(off, 0)
		}

		for _, instr := range fn.Instrs {
			switch instr.Kind {
			case ir.IRWrite:
				if err := abi.EmitWriteStdout(e, &stackDepth, importPatches); err != nil {
					return err
				}
			case ir.IRStrLit:
				leaPos := e.LeaRaxRipDisp()
				e.PushRax()
				e.MovEaxImm32(uint32(len(instr.Str)))
				e.PushRax()
				push(2)
				*leaPatches = append(*leaPatches, x64obj.LeaPatch{At: leaPos, DataIndex: len(*dataBlobs)})
				*dataBlobs = append(*dataBlobs, instr.Str)
			case ir.IRConstI32:
				e.MovEaxImm32(uint32(instr.Imm))
				e.PushRax()
				push(1)
			case ir.IRLoadLocal:
				off := -int32((instr.Local + 1) * 8)
				e.MovRaxFromRbpDisp(off)
				e.PushRax()
				push(1)
			case ir.IRStoreLocal:
				if err := pop(1); err != nil {
					return err
				}
				off := -int32((instr.Local + 1) * 8)
				e.PopRax()
				e.MovMem64RbpDispRax(off)
			case ir.IRLoadGlobal:
				leaPos := e.LeaRsiRipDisp()
				e.MovRdiRsi()
				e.MovRaxFromRdiDisp(0)
				e.PushRax()
				push(1)
				*leaPatches = append(*leaPatches, x64obj.LeaPatch{At: leaPos, DataIndex: instr.Local})
			case ir.IRStoreGlobal:
				if err := pop(1); err != nil {
					return err
				}
				e.PopRax()
				leaPos := e.LeaRsiRipDisp()
				e.MovRdiRsi()
				e.MovMem64RdiDispRax(0)
				*leaPatches = append(*leaPatches, x64obj.LeaPatch{At: leaPos, DataIndex: instr.Local})
			case ir.IRAddI32:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRcx()
				e.PopRax()
				e.AddEaxEcx()
				e.PushRax()
				push(1)
			case ir.IRSubI32:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRcx()
				e.PopRax()
				e.SubEaxEcx()
				e.PushRax()
				push(1)
			case ir.IRNegI32:
				if err := pop(1); err != nil {
					return err
				}
				e.PopRax()
				e.NegEax()
				e.PushRax()
				push(1)
			case ir.IRCmpEqI32:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRcx()
				e.PopRax()
				e.CmpEaxEcx()
				e.SeteAl()
				e.MovzxEaxAl()
				e.PushRax()
				push(1)
			case ir.IRCmpLtI32:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRcx()
				e.PopRax()
				e.CmpEaxEcx()
				e.SetlAl()
				e.MovzxEaxAl()
				e.PushRax()
				push(1)
			case ir.IRCall:
				if err := abi.EmitCall(e, instr, &stackDepth, callPatches); err != nil {
					return err
				}
			case ir.IRLabel:
				labelOffsets[instr.Label] = len(e.Buf)
			case ir.IRJmp:
				at := e.JmpRel32()
				patches = append(patches, labelPatch{at: at, label: instr.Label})
			case ir.IRJmpIfZero:
				if err := pop(1); err != nil {
					return err
				}
				e.PopRax()
				e.TestEaxEax()
				at := e.JzRel32()
				patches = append(patches, labelPatch{at: at, label: instr.Label})
			case ir.IRReturn:
				if err := pop(fn.ReturnSlots); err != nil {
					return err
				}
				switch fn.ReturnSlots {
				case 1:
					e.PopRax()
				case 2:
					e.PopRdx()
					e.PopRax()
				default:
					return fmt.Errorf("unsupported return slots")
				}
				e.Leave()
				e.Ret()
			case ir.IRAllocBytes:
				if err := abi.EmitAllocBytes(e, &stackDepth, opt, importPatches); err != nil {
					return err
				}
			case ir.IRMakeSliceU8, ir.IRMakeSliceI32:
				if err := abi.EmitMakeSlice(e, instr.Kind, &stackDepth, opt, importPatches); err != nil {
					return err
				}
			case ir.IRIndexLoadI32, ir.IRIndexLoadU8:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopRcx()
				e.PopRax()
				e.CmpEdxEcx()
				failAt := e.JaeRel32()
				if instr.Kind == ir.IRIndexLoadI32 {
					e.ShlRdxImm8(2)
				}
				e.AddRaxRdx()
				if instr.Kind == ir.IRIndexLoadI32 {
					e.MovEaxFromRaxPtr()
				} else {
					e.MovzxEaxBytePtrRax()
				}
				stackBeforePush := stackDepth
				e.PushRax()
				push(1)
				doneAt := e.JmpRel32()
				failOff := len(e.Buf)
				if err := abi.EmitExit(e, 1, stackBeforePush, importPatches); err != nil {
					return err
				}
				doneOff := len(e.Buf)
				if err := x64.PatchRel32(e.Buf, failAt, failOff); err != nil {
					return err
				}
				if err := x64.PatchRel32(e.Buf, doneAt, doneOff); err != nil {
					return err
				}
			case ir.IRIndexStoreI32, ir.IRIndexStoreU8:
				if err := pop(4); err != nil {
					return err
				}
				e.PopR8()
				e.PopRdx()
				e.PopRcx()
				e.PopRax()
				e.CmpEdxEcx()
				failAt := e.JaeRel32()
				if instr.Kind == ir.IRIndexStoreI32 {
					e.ShlRdxImm8(2)
				}
				e.AddRaxRdx()
				if instr.Kind == ir.IRIndexStoreI32 {
					e.MovMem32RaxPtrR8d()
				} else {
					e.MovMem8RaxPtrR8b()
				}
				doneAt := e.JmpRel32()
				failOff := len(e.Buf)
				if err := abi.EmitExit(e, 1, stackDepth, importPatches); err != nil {
					return err
				}
				doneOff := len(e.Buf)
				if err := x64.PatchRel32(e.Buf, failAt, failOff); err != nil {
					return err
				}
				if err := x64.PatchRel32(e.Buf, doneAt, doneOff); err != nil {
					return err
				}
			case ir.IRIslandNew:
				if err := abi.EmitIslandNew(e, &stackDepth, opt, importPatches); err != nil {
					return err
				}
			case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceI32:
				if err := abi.EmitIslandMakeSlice(e, instr.Kind, &stackDepth, opt, importPatches); err != nil {
					return err
				}
			case ir.IRIslandFree:
				if err := abi.EmitIslandFree(e, &stackDepth, opt, importPatches); err != nil {
					return err
				}
			case ir.IRCapIO:
				e.MovEaxImm32(0xC10)
				e.PushRax()
				push(1)
			case ir.IRCapMem:
				e.MovEaxImm32(0xC11)
				e.PushRax()
				push(1)
			case ir.IRMemReadI32:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRdx()
				e.PopRax()
				e.MovEaxFromRaxPtr()
				e.PushRax()
				push(1)
			case ir.IRMemWriteI32:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopRcx()
				e.PopRax()
				e.MovMem32RaxPtrEcx()
				e.PushRcx()
				push(1)
			case ir.IRMemReadU8:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRdx()
				e.PopRax()
				e.MovzxEaxBytePtrRax()
				e.PushRax()
				push(1)
			case ir.IRMemWriteU8:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopRcx()
				e.PopRax()
				e.MovMem8RaxPtrCl()
				e.PushRcx()
				push(1)
			case ir.IRMemReadPtr:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRdx()
				e.PopRax()
				e.MovRdiRax()
				e.MovRaxFromRdiDisp(0)
				e.PushRax()
				push(1)
			case ir.IRMemWritePtr:
				if err := pop(3); err != nil {
					return err
				}
				e.PopR8()  // cap.mem (unused)
				e.PopRax() // value
				e.PopRcx() // addr
				e.MovRdiRcx()
				e.MovMem64RdiDispRax(0)
				e.PushRax()
				push(1)
			case ir.IRPtrAdd:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRcx()
				e.PopRdx()
				e.PopRax()
				e.MovsxdRdxEdx()
				e.AddRaxRdx()
				e.PushRax()
				push(1)
			case ir.IRMmioReadI32:
				if err := pop(2); err != nil {
					return err
				}
				e.PopRdx()
				e.PopRax()
				e.MovEaxFromRaxPtr()
				e.PushRax()
				push(1)
			case ir.IRMmioWriteI32:
				if err := pop(3); err != nil {
					return err
				}
				e.PopRdx()
				e.PopRcx()
				e.PopRax()
				e.MovMem32RaxPtrEcx()
				e.PushRcx()
				push(1)
			case ir.IRSymAddr:
				leaPos := e.LeaRaxRipDisp()
				*callPatches = append(*callPatches, x64obj.CallPatch{At: leaPos, Name: instr.Name})
				e.PushRax()
				push(1)
			case ir.IRCtxSwitch:
				if err := pop(3); err != nil {
					return err
				}

				switch abi.(type) {
				case *x64abi.SysVUnix:
					e.PopR8()  // cap.mem (unused)
					e.PopRsi() // to_rsp_slot
					e.PopRdi() // from_rsp_slot
				case *x64abi.Win64:
					e.PopR8()  // cap.mem (unused)
					e.PopRdx() // to_rsp_slot
					e.PopRcx() // from_rsp_slot
				default:
					return fmt.Errorf("ctx_switch: unsupported ABI")
				}

				switchLabel := newInternalLabel()
				contLabel := newInternalLabel()

				if _, ok := abi.(*x64abi.Win64); ok {
					e.SubRspImm32(32)
				}
				callAt := e.CallRel32()
				patches = append(patches, labelPatch{at: callAt, label: switchLabel})

				if _, ok := abi.(*x64abi.Win64); ok {
					e.AddRspImm32(32)
				}
				e.XorEaxEax()
				e.PushRax()
				push(1)
				jmpAt := e.JmpRel32()
				patches = append(patches, labelPatch{at: jmpAt, label: contLabel})

				labelOffsets[switchLabel] = len(e.Buf)
				switch abi.(type) {
				case *x64abi.SysVUnix:
					e.PushRbx()
					e.PushRbp()
					e.PushR12()
					e.PushR13()
					e.PushR14()
					e.PushR15()
					e.MovMem64RdiDispRsp(0)
					e.MovRdiRsi()
					e.MovRspFromRdiDisp(0)
					e.PopR15()
					e.PopR14()
					e.PopR13()
					e.PopR12()
					e.PopRbp()
					e.PopRbx()
					e.Ret()
				case *x64abi.Win64:
					e.PushRbx()
					e.PushRbp()
					e.PushRdi()
					e.PushRsi()
					e.PushR12()
					e.PushR13()
					e.PushR14()
					e.PushR15()
					e.MovRdiRcx()
					e.MovMem64RdiDispRsp(0)
					e.MovRdiRdx()
					e.MovRspFromRdiDisp(0)
					e.PopR15()
					e.PopR14()
					e.PopR13()
					e.PopR12()
					e.PopRsi()
					e.PopRdi()
					e.PopRbp()
					e.PopRbx()
					e.Ret()
				}

				labelOffsets[contLabel] = len(e.Buf)
			default:
				return fmt.Errorf("unsupported IR instruction")
			}
		}

		for _, patch := range patches {
			target, ok := labelOffsets[patch.label]
			if !ok {
				return fmt.Errorf("unknown label %d", patch.label)
			}
			if err := x64.PatchRel32(e.Buf, patch.at, target); err != nil {
				return err
			}
		}

		return nil
	}
}
