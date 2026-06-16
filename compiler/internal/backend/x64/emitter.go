package x64

import (
	"encoding/binary"
	"fmt"
)

type CodegenOptions struct {
	IslandsDebug                bool
	DebugInfo                   bool
	ReleaseOptimize             bool
	DisableMachinePaths         bool
	EnableSmallHeap             bool
	DisableSmallHeap            bool
	PointerWidthBits            int
	NativeIntWidthBits          int
	RegisterWidthBits           int
	TargetFeatures              TargetFeatures
	EmitRuntimeHeapTelemetry    bool
	RuntimeHeapTelemetryDir     string
	RuntimeHeapTelemetryProgram string
	RuntimeHeapTelemetryMain    string
}

const IslandsDebugPageSize = 4096

func (o CodegenOptions) EffectivePointerWidthBits() int {
	if o.PointerWidthBits == 0 {
		return 64
	}
	return o.PointerWidthBits
}

func (o CodegenOptions) PointerWidthBytes() (int32, error) {
	switch bits := o.EffectivePointerWidthBits(); bits {
	case 32:
		return 4, nil
	case 64:
		return 8, nil
	default:
		return 0, fmt.Errorf("unsupported pointer width %d for x64 codegen", bits)
	}
}

func (o CodegenOptions) EffectiveRegisterWidthBits() int {
	if o.RegisterWidthBits == 0 {
		return 64
	}
	return o.RegisterWidthBits
}

func (o CodegenOptions) RegisterWidthBytes() (int32, error) {
	switch bits := o.EffectiveRegisterWidthBits(); bits {
	case 32:
		return 4, nil
	case 64:
		return 8, nil
	default:
		return 0, fmt.Errorf("unsupported register width %d for x64 codegen", bits)
	}
}

type Emitter struct {
	Buf []byte
}

func (e *Emitter) Emit(b ...byte) {
	e.Buf = append(e.Buf, b...)
}

func (e *Emitter) CallRel32() int {
	e.Emit(0xE8, 0x00, 0x00, 0x00, 0x00)
	return len(e.Buf) - 4
}

func (e *Emitter) CallRipDisp32() int {
	e.Emit(0xFF, 0x15, 0x00, 0x00, 0x00, 0x00)
	return len(e.Buf) - 4
}

func (e *Emitter) JmpRel32() int {
	e.Emit(0xE9, 0x00, 0x00, 0x00, 0x00)
	return len(e.Buf) - 4
}

func (e *Emitter) JzRel32() int {
	e.Emit(0x0F, 0x84, 0x00, 0x00, 0x00, 0x00)
	return len(e.Buf) - 4
}

func (e *Emitter) LeaRsiRipDisp() int {
	e.Emit(0x48, 0x8D, 0x35, 0x00, 0x00, 0x00, 0x00)
	return len(e.Buf) - 4
}

func (e *Emitter) LeaRaxRipDisp() int {
	e.Emit(0x48, 0x8D, 0x05, 0x00, 0x00, 0x00, 0x00)
	return len(e.Buf) - 4
}

func (e *Emitter) LeaRaxRbpDisp(disp int32) {
	e.Emit(0x48, 0x8D, 0x85)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) LeaRdxRipDisp() int {
	e.Emit(0x48, 0x8D, 0x15, 0x00, 0x00, 0x00, 0x00)
	return len(e.Buf) - 4
}

func (e *Emitter) LeaR9RspDisp(disp int32) {
	e.Emit(0x4C, 0x8D, 0x8C, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovEaxImm32(v uint32) {
	e.Emit(0xB8)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	e.Emit(buf[:]...)
}

func (e *Emitter) MovEcxImm32(v uint32) {
	e.Emit(0xB9)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	e.Emit(buf[:]...)
}

func (e *Emitter) MovEdiImm32(v uint32) {
	e.Emit(0xBF)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	e.Emit(buf[:]...)
}

func (e *Emitter) MovEdxImm32(v uint32) {
	e.Emit(0xBA)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	e.Emit(buf[:]...)
}

func (e *Emitter) MovR8dImm32(v uint32) {
	e.Emit(0x41, 0xB8)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	e.Emit(buf[:]...)
}

func (e *Emitter) MovR9dImm32(v uint32) {
	e.Emit(0x41, 0xB9)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	e.Emit(buf[:]...)
}

func (e *Emitter) MovRdiRax() {
	e.Emit(0x48, 0x89, 0xC7)
}

func (e *Emitter) MovRdiRcx() {
	e.Emit(0x48, 0x89, 0xCF)
}

func (e *Emitter) MovRcxRax() {
	e.Emit(0x48, 0x89, 0xC1)
}

func (e *Emitter) MovRcxRdi() {
	e.Emit(0x48, 0x89, 0xF9)
}

func (e *Emitter) MovRsiRax() {
	e.Emit(0x48, 0x89, 0xC6)
}

func (e *Emitter) MovRdiRsi() {
	e.Emit(0x48, 0x89, 0xF7)
}

func (e *Emitter) MovRsiRdi() {
	e.Emit(0x48, 0x89, 0xFE)
}

func (e *Emitter) MovRdxRax() {
	e.Emit(0x48, 0x89, 0xC2)
}

func (e *Emitter) MovR8Rdx() {
	e.Emit(0x49, 0x89, 0xD0)
}

func (e *Emitter) MovR8Rax() {
	e.Emit(0x49, 0x89, 0xC0)
}

func (e *Emitter) MovRdxR8() {
	e.Emit(0x4C, 0x89, 0xC2)
}

func (e *Emitter) MovRdxR9() {
	e.Emit(0x4C, 0x89, 0xCA)
}

func (e *Emitter) MovRsiR8() {
	e.Emit(0x4C, 0x89, 0xC6)
}

func (e *Emitter) MovRaxRdx() {
	e.Emit(0x48, 0x89, 0xD0)
}

func (e *Emitter) MovRaxRdi() {
	e.Emit(0x48, 0x89, 0xF8)
}

func (e *Emitter) MovRaxRsi() {
	e.Emit(0x48, 0x89, 0xF0)
}

func (e *Emitter) MovRaxRcx() {
	e.Emit(0x48, 0x89, 0xC8)
}

func (e *Emitter) MovRaxR9() {
	e.Emit(0x4C, 0x89, 0xC8)
}

func (e *Emitter) MovRaxR15() {
	e.Emit(0x4C, 0x89, 0xF8)
}

func (e *Emitter) MovEcxEax() {
	e.Emit(0x89, 0xC1)
}

func (e *Emitter) MovEaxEcx() {
	e.Emit(0x89, 0xC8)
}

func (e *Emitter) XorEcxEcx() {
	e.Emit(0x31, 0xC9)
}

func (e *Emitter) MovEaxEsi() {
	e.Emit(0x89, 0xF0)
}

func (e *Emitter) MovEaxEdi() {
	e.Emit(0x89, 0xF8)
}

func (e *Emitter) MovEaxR8d() {
	e.Emit(0x44, 0x89, 0xC0)
}

func (e *Emitter) MovEaxR9d() {
	e.Emit(0x44, 0x89, 0xC8)
}

func (e *Emitter) MovEaxR10d() {
	e.Emit(0x44, 0x89, 0xD0)
}

func (e *Emitter) MovEaxR14d() {
	e.Emit(0x41, 0x8B, 0xC6)
}

func (e *Emitter) MovEdxR14d() {
	e.Emit(0x41, 0x8B, 0xD6)
}

func (e *Emitter) MovEdxEcx() {
	e.Emit(0x89, 0xCA)
}

func (e *Emitter) MovEdxEax() {
	e.Emit(0x89, 0xC2)
}

func (e *Emitter) MovR8dEdx() {
	e.Emit(0x41, 0x89, 0xD0)
}

func (e *Emitter) MovEdxEdi() {
	e.Emit(0x89, 0xFA)
}

func (e *Emitter) MovEdxEsi() {
	e.Emit(0x89, 0xF2)
}

func (e *Emitter) MovEaxEdx() {
	e.Emit(0x89, 0xD0)
}

func (e *Emitter) MovEaxR12d() {
	e.Emit(0x44, 0x89, 0xE0)
}

func (e *Emitter) MovsxdRdxEdx() {
	e.Emit(0x48, 0x63, 0xD2)
}

func (e *Emitter) PushRbp() {
	e.Emit(0x55)
}

func (e *Emitter) PopRbp() {
	e.Emit(0x5D)
}

func (e *Emitter) MovRbpRsp() {
	e.Emit(0x48, 0x89, 0xE5)
}

func (e *Emitter) SubRspImm32(v int32) {
	e.Emit(0x48, 0x81, 0xEC)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	e.Emit(buf[:]...)
}

func (e *Emitter) AddRspImm32(v int32) {
	e.Emit(0x48, 0x81, 0xC4)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	e.Emit(buf[:]...)
}

func (e *Emitter) AddRdiImm32(v int32) {
	e.Emit(0x48, 0x81, 0xC7)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	e.Emit(buf[:]...)
}

func (e *Emitter) AddRsiImm32(v int32) {
	e.Emit(0x48, 0x81, 0xC6)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	e.Emit(buf[:]...)
}

func (e *Emitter) AddRdxImm32(v int32) {
	e.Emit(0x48, 0x81, 0xC2)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovEaxFromRbpDisp(disp int32) {
	e.Emit(0x8B, 0x85)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovRaxFromRbpDisp(disp int32) {
	e.Emit(0x48, 0x8B, 0x85)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovEaxFromRspDisp(disp int32) {
	e.Emit(0x8B, 0x84, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovEaxFromRdiDisp(disp int32) {
	if disp == 0 {
		e.Emit(0x8B, 0x07)
	} else if disp >= -128 && disp <= 127 {
		e.Emit(0x8B, 0x47, byte(disp))
	} else {
		e.Emit(0x8B, 0x87)
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(disp))
		e.Emit(buf[:]...)
	}
}

func (e *Emitter) MovEdxFromRdiDisp(disp int32) {
	if disp == 0 {
		e.Emit(0x8B, 0x17)
	} else if disp >= -128 && disp <= 127 {
		e.Emit(0x8B, 0x57, byte(disp))
	} else {
		e.Emit(0x8B, 0x97)
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(disp))
		e.Emit(buf[:]...)
	}
}

func (e *Emitter) MovRaxFromRspDisp(disp int32) {
	e.Emit(0x48, 0x8B, 0x84, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovEcxFromRspDisp(disp int32) {
	e.Emit(0x8B, 0x8C, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovEdxFromRspDisp(disp int32) {
	e.Emit(0x8B, 0x94, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovR8dFromRspDisp(disp int32) {
	e.Emit(0x44, 0x8B, 0x84, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovR9dFromRspDisp(disp int32) {
	e.Emit(0x44, 0x8B, 0x8C, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovRcxFromRspDisp(disp int32) {
	e.Emit(0x48, 0x8B, 0x8C, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovRdxFromRspDisp(disp int32) {
	e.Emit(0x48, 0x8B, 0x94, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovR8FromRspDisp(disp int32) {
	e.Emit(0x4C, 0x8B, 0x84, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovR9FromRspDisp(disp int32) {
	e.Emit(0x4C, 0x8B, 0x8C, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem32RbpDispEax(disp int32) {
	e.Emit(0x89, 0x85)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem64RbpDispRax(disp int32) {
	e.Emit(0x48, 0x89, 0x85)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem32RspDispEax(disp int32) {
	e.Emit(0x89, 0x84, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem64RspDispRax(disp int32) {
	e.Emit(0x48, 0x89, 0x84, 0x24)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem32RbpDispImm(disp int32, imm int32) {
	e.Emit(0xC7, 0x85)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
	binary.LittleEndian.PutUint32(buf[:], uint32(imm))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem64RbpDispImm(disp int32, imm int32) {
	e.Emit(0x48, 0xC7, 0x85)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
	binary.LittleEndian.PutUint32(buf[:], uint32(imm))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem32RbpDispEdi(disp int32) {
	e.Emit(0x89, 0xBD)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem64RbpDispRdi(disp int32) {
	e.Emit(0x48, 0x89, 0xBD)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem32RbpDispEsi(disp int32) {
	e.Emit(0x89, 0xB5)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem64RbpDispRsi(disp int32) {
	e.Emit(0x48, 0x89, 0xB5)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem32RbpDispEdx(disp int32) {
	e.Emit(0x89, 0x95)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem64RbpDispRdx(disp int32) {
	e.Emit(0x48, 0x89, 0x95)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem32RbpDispEcx(disp int32) {
	e.Emit(0x89, 0x8D)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem64RbpDispRcx(disp int32) {
	e.Emit(0x48, 0x89, 0x8D)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem32RbpDispR8d(disp int32) {
	e.Emit(0x44, 0x89, 0x85)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem64RbpDispR8(disp int32) {
	e.Emit(0x4C, 0x89, 0x85)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem32RbpDispR9d(disp int32) {
	e.Emit(0x44, 0x89, 0x8D)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem64RbpDispR9(disp int32) {
	e.Emit(0x4C, 0x89, 0x8D)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) PushRax() {
	e.Emit(0x50)
}

func (e *Emitter) PushRbx() {
	e.Emit(0x53)
}

func (e *Emitter) PushRcx() {
	e.Emit(0x51)
}

func (e *Emitter) PushRdx() {
	e.Emit(0x52)
}

func (e *Emitter) PushR8() {
	e.Emit(0x41, 0x50)
}

func (e *Emitter) PushR9() {
	e.Emit(0x41, 0x51)
}

func (e *Emitter) PushR10() {
	e.Emit(0x41, 0x52)
}

func (e *Emitter) PushR11() {
	e.Emit(0x41, 0x53)
}

func (e *Emitter) PushRsi() {
	e.Emit(0x56)
}

func (e *Emitter) PushRdi() {
	e.Emit(0x57)
}

func (e *Emitter) PushR12() {
	e.Emit(0x41, 0x54)
}

func (e *Emitter) PushR13() {
	e.Emit(0x41, 0x55)
}

func (e *Emitter) PushR14() {
	e.Emit(0x41, 0x56)
}

func (e *Emitter) PushR15() {
	e.Emit(0x41, 0x57)
}

func (e *Emitter) PopRax() {
	e.Emit(0x58)
}

func (e *Emitter) PopRbx() {
	e.Emit(0x5B)
}

func (e *Emitter) PopRcx() {
	e.Emit(0x59)
}

func (e *Emitter) PopRdx() {
	e.Emit(0x5A)
}

func (e *Emitter) PopRsi() {
	e.Emit(0x5E)
}

func (e *Emitter) PopRdi() {
	e.Emit(0x5F)
}

func (e *Emitter) PopR8() {
	e.Emit(0x41, 0x58)
}

func (e *Emitter) PopR9() {
	e.Emit(0x41, 0x59)
}

func (e *Emitter) PopR10() {
	e.Emit(0x41, 0x5A)
}

func (e *Emitter) PopR11() {
	e.Emit(0x41, 0x5B)
}

func (e *Emitter) PopR12() {
	e.Emit(0x41, 0x5C)
}

func (e *Emitter) PopR13() {
	e.Emit(0x41, 0x5D)
}

func (e *Emitter) PopR14() {
	e.Emit(0x41, 0x5E)
}

func (e *Emitter) PopR15() {
	e.Emit(0x41, 0x5F)
}

func (e *Emitter) MovR10dImm32(v uint32) {
	e.Emit(0x41, 0xBA)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	e.Emit(buf[:]...)
}

func (e *Emitter) AddEaxEcx() {
	e.Emit(0x01, 0xC8)
}

func (e *Emitter) AddRaxRdx() {
	e.Emit(0x48, 0x01, 0xD0)
}

func (e *Emitter) AddRaxRsi() {
	e.Emit(0x48, 0x01, 0xF0)
}

func (e *Emitter) AddRaxRbx() {
	e.Emit(0x48, 0x01, 0xD8)
}

func (e *Emitter) AddRdxRbx() {
	e.Emit(0x48, 0x01, 0xDA)
}

func (e *Emitter) SubEaxEcx() {
	e.Emit(0x29, 0xC8)
}

func (e *Emitter) SubEcxEdx() {
	e.Emit(0x29, 0xD1)
}

func (e *Emitter) SubR8dEcx() {
	e.Emit(0x41, 0x29, 0xC8)
}

func (e *Emitter) SubRaxRdi() {
	e.Emit(0x48, 0x29, 0xF8)
}

func (e *Emitter) CmpRdxRax() {
	e.Emit(0x48, 0x39, 0xC2)
}

func (e *Emitter) AddR10R8() {
	e.Emit(0x4D, 0x01, 0xC2)
}

func (e *Emitter) AddR10dR8d() {
	e.Emit(0x45, 0x01, 0xC2)
}

func (e *Emitter) SubR10R8() {
	e.Emit(0x4D, 0x29, 0xC2)
}

func (e *Emitter) SubR10dR8d() {
	e.Emit(0x45, 0x29, 0xC2)
}

func (e *Emitter) AndR10R8() {
	e.Emit(0x4D, 0x21, 0xC2)
}

func (e *Emitter) AndR10dR8d() {
	e.Emit(0x45, 0x21, 0xC2)
}

func (e *Emitter) OrR10R8() {
	e.Emit(0x4D, 0x09, 0xC2)
}

func (e *Emitter) OrR10dR8d() {
	e.Emit(0x45, 0x09, 0xC2)
}

func (e *Emitter) XorR10R8() {
	e.Emit(0x4D, 0x31, 0xC2)
}

func (e *Emitter) XorR10dR8d() {
	e.Emit(0x45, 0x31, 0xC2)
}

func (e *Emitter) NegEax() {
	e.Emit(0xF7, 0xD8)
}

func (e *Emitter) NegR8() {
	e.Emit(0x49, 0xF7, 0xD8)
}

func (e *Emitter) NegR8d() {
	e.Emit(0x41, 0xF7, 0xD8)
}

func (e *Emitter) NegR8w() {
	e.Emit(0x66, 0x41, 0xF7, 0xD8)
}

func (e *Emitter) NegR8b() {
	e.Emit(0x41, 0xF6, 0xD8)
}

func (e *Emitter) CmpEaxEcx() {
	e.Emit(0x39, 0xC8)
}

func (e *Emitter) CmpEaxEdx() {
	e.Emit(0x39, 0xD0)
}

func (e *Emitter) CmpEdxEcx() {
	e.Emit(0x39, 0xCA)
}

func (e *Emitter) CmpEbxEcx() {
	e.Emit(0x39, 0xCB)
}

func (e *Emitter) CmpR8dImm8(v byte) {
	e.Emit(0x41, 0x83, 0xF8, v)
}

func (e *Emitter) SeteAl() {
	e.Emit(0x0F, 0x94, 0xC0)
}

func (e *Emitter) SetlAl() {
	e.Emit(0x0F, 0x9C, 0xC0)
}

func (e *Emitter) SetneAl() {
	e.Emit(0x0F, 0x95, 0xC0)
}

func (e *Emitter) SetgAl() {
	e.Emit(0x0F, 0x9F, 0xC0)
}

func (e *Emitter) SetgeAl() {
	e.Emit(0x0F, 0x9D, 0xC0)
}

func (e *Emitter) SetleAl() {
	e.Emit(0x0F, 0x9E, 0xC0)
}

func (e *Emitter) ImulEaxEcx() {
	e.Emit(0x0F, 0xAF, 0xC1)
}

func (e *Emitter) Cdq() {
	e.Emit(0x99)
}

func (e *Emitter) IdivEcx() {
	e.Emit(0xF7, 0xF9)
}

func (e *Emitter) ShlRaxImm8(v byte) {
	e.Emit(0x48, 0xC1, 0xE0, v)
}

func (e *Emitter) ShlRdxImm8(v byte) {
	e.Emit(0x48, 0xC1, 0xE2, v)
}

func (e *Emitter) MovzxEaxAl() {
	e.Emit(0x0F, 0xB6, 0xC0)
}

func (e *Emitter) MovzxEaxAx() {
	e.Emit(0x0F, 0xB7, 0xC0)
}

func (e *Emitter) MovzxR8dR8b() {
	e.Emit(0x45, 0x0F, 0xB6, 0xC0)
}

func (e *Emitter) MovzxR8dR8w() {
	e.Emit(0x45, 0x0F, 0xB7, 0xC0)
}

func (e *Emitter) MovEaxFromRaxPtr() {
	e.Emit(0x8B, 0x00)
}

func (e *Emitter) MovzxEaxBytePtrRax() {
	e.Emit(0x0F, 0xB6, 0x00)
}

func (e *Emitter) MovzxEaxWordPtrRax() {
	e.Emit(0x0F, 0xB7, 0x00)
}

func (e *Emitter) MovzxEaxBytePtrRdi() {
	e.Emit(0x0F, 0xB6, 0x07)
}

func (e *Emitter) MovzxEaxWordPtrRdi() {
	e.Emit(0x0F, 0xB7, 0x07)
}

func (e *Emitter) MovzxEaxBytePtrRsiRcx() {
	e.Emit(0x0F, 0xB6, 0x04, 0x0E)
}

func (e *Emitter) MovMem32RaxPtrR8d() {
	e.Emit(0x44, 0x89, 0x00)
}

func (e *Emitter) MovMem32RaxPtrEcx() {
	e.Emit(0x89, 0x08)
}

func (e *Emitter) MovMem32RaxPtrEsi() {
	e.Emit(0x89, 0x30)
}

func (e *Emitter) MovMem8RaxPtrR8b() {
	e.Emit(0x44, 0x88, 0x00)
}

func (e *Emitter) MovMem16RaxPtrR8w() {
	e.Emit(0x66, 0x44, 0x89, 0x00)
}

func (e *Emitter) MovMem8RdiRcxPtrAl() {
	e.Emit(0x88, 0x04, 0x0F)
}

func (e *Emitter) MovMem8RaxPtrCl() {
	e.Emit(0x88, 0x08)
}

func (e *Emitter) TestEaxEax() {
	e.Emit(0x85, 0xC0)
}

func (e *Emitter) TestEdxEdx() {
	e.Emit(0x85, 0xD2)
}

func (e *Emitter) TestEcxEcx() {
	e.Emit(0x85, 0xC9)
}

func (e *Emitter) TestRaxRax() {
	e.Emit(0x48, 0x85, 0xC0)
}

func (e *Emitter) TestRcxRcx() {
	e.Emit(0x48, 0x85, 0xC9)
}

func (e *Emitter) Leave() {
	e.Emit(0xC9)
}

func (e *Emitter) Syscall() {
	e.Emit(0x0F, 0x05)
}

func (e *Emitter) Ret() {
	e.Emit(0xC3)
}
