package x64core

import (
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/backend/x64obj"
	"tetra_language/compiler/internal/ir"
)

func emitSliceView(
	e *x64.Emitter,
	kind ir.IRInstrKind,
	shift byte,
	pop func(int) error,
	push func(int),
	stackDepth *int,
	abi x64abi.ABI,
	importPatches *[]x64obj.ImportPatch,
) error {
	failPatches := []int{}
	switch kind {
	case ir.IRSliceWindow:
		if err := pop(4); err != nil {
			return err
		}
		failStackDepth := *stackDepth
		e.PopRbx() // count
		e.PopRdx() // start
		e.PopRcx() // source len
		e.PopRax() // source ptr
		e.CmpEdxImm32(0)
		failPatches = append(failPatches, e.JlRel32())
		e.CmpEbxImm32(0)
		failPatches = append(failPatches, e.JlRel32())
		e.CmpEdxEcx()
		failPatches = append(failPatches, e.JgRel32())
		e.SubEcxEdx()
		e.CmpEbxEcx()
		failPatches = append(failPatches, e.JgRel32())
		if shift > 0 {
			e.ShlRdxImm8(shift)
		}
		e.AddRaxRdx()
		e.PushRax()
		e.PushRbx()
		push(2)
		return patchSliceViewFailure(e, failPatches, failStackDepth, abi, importPatches)
	case ir.IRSlicePrefix:
		if err := pop(3); err != nil {
			return err
		}
		failStackDepth := *stackDepth
		e.PopRbx() // count
		e.PopRcx() // source len
		e.PopRax() // source ptr
		e.CmpEbxImm32(0)
		failPatches = append(failPatches, e.JlRel32())
		e.CmpEbxEcx()
		failPatches = append(failPatches, e.JgRel32())
		e.PushRax()
		e.PushRbx()
		push(2)
		return patchSliceViewFailure(e, failPatches, failStackDepth, abi, importPatches)
	case ir.IRSliceSuffix:
		if err := pop(3); err != nil {
			return err
		}
		failStackDepth := *stackDepth
		e.PopRdx() // start
		e.PopRcx() // source len
		e.PopRax() // source ptr
		e.CmpEdxImm32(0)
		failPatches = append(failPatches, e.JlRel32())
		e.CmpEdxEcx()
		failPatches = append(failPatches, e.JgRel32())
		e.SubEcxEdx()
		if shift > 0 {
			e.ShlRdxImm8(shift)
		}
		e.AddRaxRdx()
		e.PushRax()
		e.PushRcx()
		push(2)
		return patchSliceViewFailure(e, failPatches, failStackDepth, abi, importPatches)
	default:
		return fmt.Errorf("x64 backend: unsupported slice view kind %v", kind)
	}
}

func patchSliceViewFailure(e *x64.Emitter, failPatches []int, failStackDepth int, abi x64abi.ABI, importPatches *[]x64obj.ImportPatch) error {
	doneAt := e.JmpRel32()
	failOff := len(e.Buf)
	if err := abi.EmitExit(e, 1, failStackDepth, importPatches); err != nil {
		return err
	}
	doneOff := len(e.Buf)
	for _, at := range failPatches {
		if err := x64.PatchRel32(e.Buf, at, failOff); err != nil {
			return err
		}
	}
	return x64.PatchRel32(e.Buf, doneAt, doneOff)
}
