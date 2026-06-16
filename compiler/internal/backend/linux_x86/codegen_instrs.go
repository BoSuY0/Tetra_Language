package linux_x86

import (
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/runtimeabi"
)

func emitCall(e *emitter, fn ir.IRFunc, instr ir.IRInstr, pop func(int) error, push func(int), callPatches *[]callPatch) error {
	if instr.Name == "" {
		return fmt.Errorf("x86 backend: empty call target in function '%s'", fn.Name)
	}
	if instr.ArgSlots < 0 || instr.RetSlots < 0 {
		return fmt.Errorf("x86 backend: call %q has negative ABI slots args=%d rets=%d in function '%s'", instr.Name, instr.ArgSlots, instr.RetSlots, fn.Name)
	}
	if instr.RetSlots > 4 {
		return fmt.Errorf("x86 backend: unsupported call return slots %d in function '%s' call %q", instr.RetSlots, fn.Name, instr.Name)
	}
	if err := pop(instr.ArgSlots); err != nil {
		return err
	}
	argBytes := instr.ArgSlots * 4
	if argBytes > 0 {
		e.subEspImm32(int32(argBytes))
		for i := 0; i < instr.ArgSlots; i++ {
			src := int32(argBytes + 4*(instr.ArgSlots-1-i))
			dst := int32(4 * i)
			e.movEaxFromEspDisp(src)
			e.movMemEspDispEax(dst)
		}
	}
	at := e.callRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: instr.Name})
	if argBytes > 0 {
		e.addEspImm32(int32(argBytes * 2))
	}
	switch instr.RetSlots {
	case 0:
	case 1:
		e.pushEax()
		push(1)
	case 2:
		e.pushEax()
		e.pushEdx()
		push(2)
	case 3:
		e.pushEax()
		e.pushEdx()
		e.pushEcx()
		push(3)
	case 4:
		e.pushEax()
		e.pushEdx()
		e.pushEcx()
		e.pushEbx()
		push(4)
	}
	return nil
}

func emitReturn(e *emitter, fn ir.IRFunc, pop func(int) error) error {
	if err := pop(fn.ReturnSlots); err != nil {
		return err
	}
	switch fn.ReturnSlots {
	case 0:
	case 1:
		e.popEax()
	case 2:
		e.popEdx()
		e.popEax()
	case 3:
		e.popEcx()
		e.popEdx()
		e.popEax()
	case 4:
		e.popEbx()
		e.popEcx()
		e.popEdx()
		e.popEax()
	default:
		return fmt.Errorf("x86 backend: unsupported return slots %d in function '%s'", fn.ReturnSlots, fn.Name)
	}
	e.leave()
	e.ret()
	return nil
}

func emitCmp(e *emitter, kind ir.IRInstrKind, pop func(int) error) error {
	if err := pop(2); err != nil {
		return err
	}
	e.popEcx()
	e.popEax()
	e.cmpEaxEcx()
	switch kind {
	case ir.IRCmpEqI32:
		e.setccAl(0x94)
	case ir.IRCmpLtI32:
		e.setccAl(0x9C)
	case ir.IRCmpGtI32:
		e.setccAl(0x9F)
	case ir.IRCmpGeI32:
		e.setccAl(0x9D)
	case ir.IRCmpLeI32:
		e.setccAl(0x9E)
	case ir.IRCmpNeI32:
		e.setccAl(0x95)
	default:
		return fmt.Errorf("unsupported comparison kind %v", kind)
	}
	e.movzxEaxAl()
	e.pushEax()
	return nil
}

func emitWrite(e *emitter, pop func(int) error) error {
	if err := pop(2); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.movEbxImm32(1)
	e.movEaxImm32(4)
	e.int80()
	return nil
}

func emitCtxSwitch(e *emitter, pop func(int) error, push func(int)) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx() // cap.mem, currently a permission token only.
	e.popEcx() // to_rsp_slot
	e.popEax() // from_rsp_slot

	callAt := e.callRel32()
	e.xorEaxEax()
	e.pushEax()
	push(1)
	jmpAt := e.jmpRel32()

	switchOff := len(e.buf)
	if err := e.patchRel32(callAt, switchOff); err != nil {
		return err
	}
	e.pushEbx()
	e.pushEbp()
	e.pushEsi()
	e.pushEdi()
	e.movMemEaxPtrEsp()
	e.movEspFromEcxPtr()
	e.popEdi()
	e.popEsi()
	e.popEbp()
	e.popEbx()
	e.ret()

	contOff := len(e.buf)
	return e.patchRel32(jmpAt, contOff)
}

func emitAllocBytes(e *emitter, pop func(int) error, push func(int)) error {
	if err := pop(1); err != nil {
		return err
	}
	e.popEcx()
	e.cmpEcxImm8(1)
	sizeOKAt := e.jgeRel32()
	emitExit(e, 2)
	sizeOKOff := len(e.buf)
	if err := e.patchRel32(sizeOKAt, sizeOKOff); err != nil {
		return err
	}

	e.pushEcx()
	e.addEcxImm32(8)
	emitMmap2Anonymous(e)
	failAt := emitMmapFailureBranch(e)
	e.popEcx()
	e.movMemEaxPtrEcx()
	e.addEaxImm8(8)
	e.pushEax()
	push(1)
	return patchExitBranch(e, failAt, 2)
}

func emitMakeSlice(e *emitter, kind ir.IRInstrKind, pop func(int) error, push func(int)) error {
	if err := pop(1); err != nil {
		return err
	}
	e.popEcx()
	e.testEcxEcx()
	negativeAt := e.jlRel32()
	e.cmpEcxImm8(0)
	emptyAt := e.jzRel32()
	overflowAt := -1
	if max, ok := x86MakeSliceMaxElements(kind); ok {
		e.cmpEcxImm32(uint32(max))
		overflowAt = e.jgRel32()
	}
	e.pushEcx()
	switch kind {
	case ir.IRMakeSliceI32:
		e.shlEcxImm8(2)
	case ir.IRMakeSliceU16:
		e.shlEcxImm8(1)
	case ir.IRMakeSliceU8:
	default:
		return fmt.Errorf("x86 backend: unsupported make_slice kind %v", kind)
	}
	emitMmap2Anonymous(e)
	failAt := emitMmapFailureBranch(e)
	e.popEcx()
	e.pushEax()
	e.pushEcx()
	push(2)
	doneAt := e.jmpRel32()
	failOff := len(e.buf)
	emitExit(e, 2)
	emptyOff := len(e.buf)
	e.movEaxImm32(0)
	e.pushEax()
	e.pushEcx()
	doneOff := len(e.buf)
	if err := e.patchRel32(emptyAt, emptyOff); err != nil {
		return err
	}
	if err := e.patchRel32(negativeAt, failOff); err != nil {
		return err
	}
	if overflowAt >= 0 {
		if err := e.patchRel32(overflowAt, failOff); err != nil {
			return err
		}
	}
	if err := e.patchRel32(failAt, failOff); err != nil {
		return err
	}
	return e.patchRel32(doneAt, doneOff)
}

func emitRawSliceFromParts(e *emitter, pop func(int) error, push func(int)) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEax()
	e.pushEax()
	e.pushEcx()
	push(2)
	return nil
}

func emitSliceView(e *emitter, kind ir.IRInstrKind, shift byte, pop func(int) error, push func(int)) error {
	failPatches := []int{}
	switch kind {
	case ir.IRSliceWindow:
		if err := pop(4); err != nil {
			return err
		}
		e.popEbx()
		e.popEdx()
		e.popEcx()
		e.popEax()
		e.cmpEdxImm8(0)
		failPatches = append(failPatches, e.jlRel32())
		e.cmpEbxImm32(0)
		failPatches = append(failPatches, e.jlRel32())
		e.cmpEdxEcx()
		failPatches = append(failPatches, e.jgRel32())
		e.subEcxEdx()
		e.cmpEbxEcx()
		failPatches = append(failPatches, e.jgRel32())
		if shift > 0 {
			e.shlEdxImm8(shift)
		}
		e.addEaxEdx()
		e.pushEax()
		e.pushEbx()
		push(2)
		return patchExitBranches(e, failPatches, 1)
	case ir.IRSlicePrefix:
		if err := pop(3); err != nil {
			return err
		}
		e.popEbx()
		e.popEcx()
		e.popEax()
		e.cmpEbxImm32(0)
		failPatches = append(failPatches, e.jlRel32())
		e.cmpEbxEcx()
		failPatches = append(failPatches, e.jgRel32())
		e.pushEax()
		e.pushEbx()
		push(2)
		return patchExitBranches(e, failPatches, 1)
	case ir.IRSliceSuffix:
		if err := pop(3); err != nil {
			return err
		}
		e.popEdx()
		e.popEcx()
		e.popEax()
		e.cmpEdxImm8(0)
		failPatches = append(failPatches, e.jlRel32())
		e.cmpEdxEcx()
		failPatches = append(failPatches, e.jgRel32())
		e.subEcxEdx()
		if shift > 0 {
			e.shlEdxImm8(shift)
		}
		e.addEaxEdx()
		e.pushEax()
		e.pushEcx()
		push(2)
		return patchExitBranches(e, failPatches, 1)
	default:
		return fmt.Errorf("x86 backend: unsupported slice view kind %v", kind)
	}
}

func emitIslandNew(e *emitter, pop func(int) error, push func(int), opt x64.CodegenOptions) error {
	if err := pop(1); err != nil {
		return err
	}
	e.popEcx()
	cfg := runtimeabi.RuntimeRegionAllocatorConfig(opt.IslandsDebug)
	headerSize := cfg.HeaderBytes
	e.testEcxEcx()
	negativeAt := e.jlRel32()
	e.cmpEcxImm32(uint32(cfg.MaxPayloadBytes))
	overflowAt := e.jgRel32()
	if opt.IslandsDebug && headerSize != x64.IslandsDebugPageSize {
		return fmt.Errorf("internal error: island debug header size mismatch")
	}
	e.addEcxImm32(headerSize)
	e.pushEcx()
	emitMmap2Anonymous(e)
	failAt := emitMmapFailureBranch(e)
	e.popEcx()
	e.movMemEaxPtrImm32(uint32(headerSize))
	e.movMemEaxDispEcx(4)
	e.movMemEaxDispEcx(8)
	e.movMemEaxDispImm32(12, 0)
	e.pushEax()
	push(1)
	return patchExitBranches(e, []int{negativeAt, overflowAt, failAt}, 2)
}

func emitIslandMakeSlice(e *emitter, kind ir.IRInstrKind, pop func(int) error, push func(int)) error {
	if err := pop(2); err != nil {
		return err
	}
	e.popEcx()
	e.popEax()
	e.testEcxEcx()
	negativeAt := e.jlRel32()
	e.cmpEcxImm8(0)
	emptyAt := e.jzRel32()
	overflowAt := -1
	if max, ok := x86MakeSliceMaxElements(kind); ok {
		e.cmpEcxImm32(uint32(max))
		overflowAt = e.jgRel32()
	}
	e.pushEcx()
	switch kind {
	case ir.IRIslandMakeSliceI32:
		e.shlEcxImm8(2)
	case ir.IRIslandMakeSliceU16:
		e.shlEcxImm8(1)
	case ir.IRIslandMakeSliceU8:
	default:
		return fmt.Errorf("x86 backend: unsupported island_make_slice kind %v", kind)
	}
	e.movEdxFromEaxPtr()
	e.movEbxFromEaxDisp(4)
	e.movEdiEdx()
	e.addEdiEcx()
	e.addEdiImm8(byte(runtimeabi.RegionAllocatorAlignmentBytes - 1))
	e.andEdiImm32(-runtimeabi.RegionAllocatorAlignmentBytes)
	e.cmpEdiEbx()
	failAt := e.jaRel32()
	e.movMemEaxPtrEdi()
	e.addEdxEax()
	e.popEcx()
	e.pushEdx()
	e.pushEcx()
	push(2)
	doneAt := e.jmpRel32()
	lengthFailOff := len(e.buf)
	emitExit(e, 2)
	capacityFailOff := len(e.buf)
	emitExit(e, 1)
	emptyOff := len(e.buf)
	e.movEaxImm32(0)
	e.pushEax()
	e.pushEcx()
	doneOff := len(e.buf)
	if err := e.patchRel32(negativeAt, lengthFailOff); err != nil {
		return err
	}
	if overflowAt >= 0 {
		if err := e.patchRel32(overflowAt, lengthFailOff); err != nil {
			return err
		}
	}
	if err := e.patchRel32(emptyAt, emptyOff); err != nil {
		return err
	}
	if err := e.patchRel32(failAt, capacityFailOff); err != nil {
		return err
	}
	return e.patchRel32(doneAt, doneOff)
}

func x86MakeSliceMaxElements(kind ir.IRInstrKind) (int32, bool) {
	switch kind {
	case ir.IRMakeSliceU16, ir.IRIslandMakeSliceU16:
		return 2147483647 / 2, true
	case ir.IRMakeSliceI32, ir.IRIslandMakeSliceI32:
		return 2147483647 / 4, true
	default:
		return 0, false
	}
}

func emitIslandFree(e *emitter, pop func(int) error, opt x64.CodegenOptions) error {
	if err := pop(1); err != nil {
		return err
	}
	e.popEbx()
	if opt.IslandsDebug {
		e.movEaxFromEbxDisp(12)
		e.testEaxEax()
		okAt := e.jzRel32()
		emitExit(e, 2)
		okOff := len(e.buf)
		if err := e.patchRel32(okAt, okOff); err != nil {
			return err
		}
		e.movMemEbxDispImm32(12, 1)
		e.movEcxFromEbxDisp(8)
		e.subEcxImm32(x64.IslandsDebugPageSize)
		e.addEbxImm32(x64.IslandsDebugPageSize)
		e.movEdxImm32(0)
		e.movEaxImm32(125)
		e.int80()
		return nil
	}
	e.movEcxFromEbxDisp(8)
	e.movEaxImm32(91)
	e.int80()
	return nil
}

func emitIslandReset(e *emitter, pop func(int) error, push func(int), opt x64.CodegenOptions) error {
	if err := pop(1); err != nil {
		return err
	}
	e.popEbx()
	if opt.IslandsDebug {
		e.movEaxFromEbxDisp(12)
		e.testEaxEax()
		okAt := e.jzRel32()
		emitExit(e, 2)
		okOff := len(e.buf)
		if err := e.patchRel32(okAt, okOff); err != nil {
			return err
		}
	}
	cfg := runtimeabi.RuntimeRegionAllocatorConfig(opt.IslandsDebug)
	e.movMemEbxDispImm32(0, uint32(cfg.HeaderBytes))
	e.pushEbx()
	push(1)
	return nil
}

func emitIndexLoad(e *emitter, kind ir.IRInstrKind, pop func(int) error, push func(int)) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEax()
	checked := kind == ir.IRIndexLoadI32 || kind == ir.IRIndexLoadU8 || kind == ir.IRIndexLoadU16
	failAt := 0
	if checked {
		e.cmpEdxEcx()
		failAt = e.jaeRel32()
	}
	scaleIndex(e, kind)
	e.addEaxEdx()
	switch kind {
	case ir.IRIndexLoadI32, ir.IRIndexLoadI32Unchecked:
		e.movEaxFromEaxPtr()
	case ir.IRIndexLoadU16, ir.IRIndexLoadU16Unchecked:
		e.movzxEaxWordEaxPtr()
	case ir.IRIndexLoadU8, ir.IRIndexLoadU8Unchecked:
		e.movzxEaxByteEaxPtr()
	default:
		return fmt.Errorf("x86 backend: unsupported index load kind %v", kind)
	}
	e.pushEax()
	push(1)
	if checked {
		return patchExitBranch(e, failAt, 1)
	}
	return nil
}

func emitIndexStore(e *emitter, kind ir.IRInstrKind, pop func(int) error) error {
	if err := pop(4); err != nil {
		return err
	}
	e.popEbx()
	e.popEdx()
	e.popEcx()
	e.popEax()
	e.cmpEdxEcx()
	failAt := e.jaeRel32()
	scaleIndex(e, kind)
	e.addEaxEdx()
	switch kind {
	case ir.IRIndexStoreI32:
		e.movMemEaxPtrEbx()
	case ir.IRIndexStoreU16:
		e.movMem16EaxPtrBx()
	case ir.IRIndexStoreU8:
		e.movMem8EaxPtrBl()
	default:
		return fmt.Errorf("x86 backend: unsupported index store kind %v", kind)
	}
	return patchExitBranch(e, failAt, 1)
}

func scaleIndex(e *emitter, kind ir.IRInstrKind) {
	switch kind {
	case ir.IRIndexLoadI32, ir.IRIndexLoadI32Unchecked, ir.IRIndexStoreI32:
		e.shlEdxImm8(2)
	case ir.IRIndexLoadU16, ir.IRIndexLoadU16Unchecked, ir.IRIndexStoreU16:
		e.shlEdxImm8(1)
	}
}

func emitMmap2Anonymous(e *emitter) {
	e.xorEbxEbx()
	e.movEdxImm32(3)
	e.movEsiImm32(0x22)
	e.movEdiImm32(0xFFFFFFFF)
	e.pushEbp()
	e.xorEbpEbp()
	e.movEaxImm32(192)
	e.int80()
	e.popEbp()
}

func emitMmapFailureBranch(e *emitter) int {
	e.cmpEaxImm32(0xFFFFF001)
	return e.jaeRel32()
}

func patchExitBranch(e *emitter, failAt int, code int32) error {
	doneAt := e.jmpRel32()
	failOff := len(e.buf)
	emitExit(e, code)
	doneOff := len(e.buf)
	if err := e.patchRel32(failAt, failOff); err != nil {
		return err
	}
	return e.patchRel32(doneAt, doneOff)
}

func patchExitBranches(e *emitter, failAts []int, code int32) error {
	doneAt := e.jmpRel32()
	failOff := len(e.buf)
	emitExit(e, code)
	doneOff := len(e.buf)
	for _, failAt := range failAts {
		if err := e.patchRel32(failAt, failOff); err != nil {
			return err
		}
	}
	return e.patchRel32(doneAt, doneOff)
}

func emitExit(e *emitter, code int32) {
	e.movEbxImm32(uint32(code))
	e.movEaxImm32(1)
	e.int80()
}

func emitRawMemoryRead(e *emitter, kind ir.IRInstrKind, pop func(int) error, push func(int)) error {
	if err := pop(2); err != nil {
		return err
	}
	e.popEdx()
	e.popEax()
	if err := guardAllocationBaseRawAccess(e, rawMemoryWidthBytes(kind)); err != nil {
		return err
	}
	emitRawLoad(e, kind)
	e.pushEax()
	push(1)
	return nil
}

func emitRawMemoryWrite(e *emitter, kind ir.IRInstrKind, pop func(int) error, push func(int)) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEbx()
	e.popEax()
	if err := guardAllocationBaseRawAccess(e, rawMemoryWidthBytes(kind)); err != nil {
		return err
	}
	emitRawStore(e, kind)
	e.pushEbx()
	push(1)
	return nil
}

func emitRawMemoryOffsetRead(e *emitter, kind ir.IRInstrKind, pop func(int) error, push func(int)) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEcx()
	e.popEdx()
	e.popEax()
	if err := guardAllocationOffsetRawAccess(e, rawMemoryWidthBytes(kind)); err != nil {
		return err
	}
	emitRawLoad(e, kind)
	e.pushEax()
	push(1)
	return nil
}

func emitRawMemoryOffsetWrite(e *emitter, kind ir.IRInstrKind, pop func(int) error, push func(int)) error {
	if err := pop(4); err != nil {
		return err
	}
	e.popEcx()
	e.popEbx()
	e.popEdx()
	e.popEax()
	if err := guardAllocationOffsetRawAccess(e, rawMemoryWidthBytes(kind)); err != nil {
		return err
	}
	emitRawStore(e, kind)
	e.pushEbx()
	push(1)
	return nil
}

func emitPtrAdd(e *emitter, pop func(int) error, push func(int)) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEcx()
	e.popEdx()
	e.popEax()
	if err := guardAllocationOffsetRawAccess(e, 1); err != nil {
		return err
	}
	e.pushEax()
	push(1)
	return nil
}

func emitMMIOReadI32(e *emitter, pop func(int) error, push func(int)) error {
	if err := pop(2); err != nil {
		return err
	}
	e.popEdx()
	e.popEax()
	e.movEaxFromEaxPtr()
	e.pushEax()
	push(1)
	return nil
}

func emitMMIOWriteI32(e *emitter, pop func(int) error, push func(int)) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEax()
	e.movMemEaxPtrEcx()
	e.pushEcx()
	push(1)
	return nil
}

func rawMemoryWidthBytes(kind ir.IRInstrKind) byte {
	switch kind {
	case ir.IRMemReadU8, ir.IRMemWriteU8, ir.IRMemReadU8Offset, ir.IRMemWriteU8Offset:
		return 1
	default:
		return 4
	}
}

func emitRawLoad(e *emitter, kind ir.IRInstrKind) {
	switch kind {
	case ir.IRMemReadU8, ir.IRMemReadU8Offset:
		e.movzxEaxByteEaxPtr()
	default:
		e.movEaxFromEaxPtr()
	}
}

func emitRawStore(e *emitter, kind ir.IRInstrKind) {
	switch kind {
	case ir.IRMemWriteU8, ir.IRMemWriteU8Offset:
		e.movMem8EaxPtrBl()
	default:
		e.movMemEaxPtrEbx()
	}
}

func guardAllocationBaseRawAccess(e *emitter, width byte) error {
	e.movEdxImm32(0)
	return guardAllocationOffsetRawAccess(e, width)
}

func guardAllocationOffsetRawAccess(e *emitter, width byte) error {
	e.cmpEdxImm8(0)
	offsetOKAt := e.jgeRel32()
	emitExit(e, 2)
	offsetOKOff := len(e.buf)
	if err := e.patchRel32(offsetOKAt, offsetOKOff); err != nil {
		return err
	}
	e.movEdiEax()
	e.andEdiImm32(-4096)
	e.movEcxFromEdiPtr()
	e.addEdiImm8(8)
	e.subEaxEdi()
	e.addEdxEax()
	e.addEdxImm8(width)
	e.cmpEdxEcx()
	failAt := e.jaRel32()
	e.subEdxImm8(width)
	e.movEaxEdi()
	e.addEaxEdx()
	return patchExitBranch(e, failAt, 2)
}

func atomicLoad32(e *emitter, pop func(int) error) error {
	if err := pop(2); err != nil {
		return err
	}
	e.popEdx()
	e.popEdi()
	e.movEaxFromEdiPtr()
	e.pushEax()
	return nil
}

func atomicLoad8(e *emitter, pop func(int) error) error {
	if err := pop(2); err != nil {
		return err
	}
	e.popEdx()
	e.popEdi()
	e.movzxEaxByteEdiPtr()
	e.pushEax()
	return nil
}

func atomicLoad16(e *emitter, pop func(int) error) error {
	if err := pop(2); err != nil {
		return err
	}
	e.popEdx()
	e.popEdi()
	e.movzxEaxWordEdiPtr()
	e.pushEax()
	return nil
}

func atomicStore32(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.movEbxEcx()
	e.xchgMem32EdiEcx()
	e.pushEbx()
	return nil
}

func atomicStore8(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.movEaxEcx()
	e.movzxEaxAl()
	e.xchgMem8EdiCl()
	e.pushEax()
	return nil
}

func atomicStore16(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.movEaxEcx()
	e.movzxEaxAx()
	e.xchgMem16EdiCx()
	e.pushEax()
	return nil
}

func atomicExchange32(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.xchgMem32EdiEcx()
	e.pushEcx()
	return nil
}

func atomicExchange8(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.xchgMem8EdiCl()
	e.movzxEaxAl()
	e.pushEax()
	return nil
}

func atomicExchange16(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.xchgMem16EdiCx()
	e.movzxEaxAx()
	e.pushEax()
	return nil
}

func atomicFetchAdd32(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.lockXaddMem32EdiEcx()
	e.pushEcx()
	return nil
}

func atomicFetchAdd8(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.lockXaddMem8EdiCl()
	e.movzxEaxAl()
	e.pushEax()
	return nil
}

func atomicFetchAdd16(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.lockXaddMem16EdiCx()
	e.movzxEaxAx()
	e.pushEax()
	return nil
}

func atomicFetchSub32(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.negEcx()
	e.lockXaddMem32EdiEcx()
	e.pushEcx()
	return nil
}

func atomicFetchSub8(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.negEcx()
	e.lockXaddMem8EdiCl()
	e.movzxEaxAl()
	e.pushEax()
	return nil
}

func atomicFetchSub16(e *emitter, pop func(int) error) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.negEcx()
	e.lockXaddMem16EdiCx()
	e.movzxEaxAx()
	e.pushEax()
	return nil
}

func atomicCompareExchange32(e *emitter, pop func(int) error) error {
	if err := pop(4); err != nil {
		return err
	}
	e.popEbx()
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.movEaxEcxValue()
	e.lockCmpxchgMem32EdiEdx()
	e.pushEax()
	return nil
}

func atomicCompareExchange8(e *emitter, pop func(int) error) error {
	if err := pop(4); err != nil {
		return err
	}
	e.popEbx()
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.movEaxEcxValue()
	e.lockCmpxchgMem8EdiDl()
	e.movzxEaxAl()
	e.pushEax()
	return nil
}

func atomicCompareExchange16(e *emitter, pop func(int) error) error {
	if err := pop(4); err != nil {
		return err
	}
	e.popEbx()
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.movEaxEcxValue()
	e.lockCmpxchgMem16EdiDx()
	e.movzxEaxAx()
	e.pushEax()
	return nil
}

func atomicFetchCASLoop32(e *emitter, pop func(int) error, op func()) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.movEaxFromEdiPtr()
	retry := len(e.buf)
	e.movEbxEax()
	op()
	e.lockCmpxchgMem32EdiEbx()
	jnz := e.jnzRel32()
	if err := e.patchRel32(jnz, retry); err != nil {
		return err
	}
	e.pushEax()
	return nil
}

func atomicFetchCASLoop8(e *emitter, pop func(int) error, op func()) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.movzxEaxByteEdiPtr()
	retry := len(e.buf)
	e.movEdxEax()
	op()
	e.lockCmpxchgMem8EdiDl()
	jnz := e.jnzRel32()
	if err := e.patchRel32(jnz, retry); err != nil {
		return err
	}
	e.movzxEaxAl()
	e.pushEax()
	return nil
}

func atomicFetchCASLoop16(e *emitter, pop func(int) error, op func()) error {
	if err := pop(3); err != nil {
		return err
	}
	e.popEdx()
	e.popEcx()
	e.popEdi()
	e.movzxEaxWordEdiPtr()
	retry := len(e.buf)
	e.movEdxEax()
	op()
	e.lockCmpxchgMem16EdiDx()
	jnz := e.jnzRel32()
	if err := e.patchRel32(jnz, retry); err != nil {
		return err
	}
	e.movzxEaxAx()
	e.pushEax()
	return nil
}

func alignUp(value int, align int) int {
	if align <= 1 {
		return value
	}
	remainder := value % align
	if remainder == 0 {
		return value
	}
	return value + align - remainder
}
