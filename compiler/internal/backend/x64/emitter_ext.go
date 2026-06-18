package x64

import (
	"encoding/binary"
	"fmt"
)

func (e *Emitter) JaeRel32() int {
	e.Emit(0x0F, 0x83, 0x00, 0x00, 0x00, 0x00)
	return len(e.Buf) - 4
}

func (e *Emitter) JlRel32() int {
	e.Emit(0x0F, 0x8C, 0x00, 0x00, 0x00, 0x00)
	return len(e.Buf) - 4
}

func (e *Emitter) JgRel32() int {
	e.Emit(0x0F, 0x8F, 0x00, 0x00, 0x00, 0x00)
	return len(e.Buf) - 4
}

// ===================== ISLANDS EMITTER METHODS =====================

func (e *Emitter) AddEaxImm32(v int32) {
	e.Emit(0x05)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	e.Emit(buf[:]...)
}

func (e *Emitter) AddEcxImm8(v byte) {
	e.Emit(0x83, 0xC1, v)
}

func (e *Emitter) SubEaxImm32(v int32) {
	e.Emit(0x2D)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	e.Emit(buf[:]...)
}

func (e *Emitter) AddRaxImm32(v int32) {
	e.Emit(0x48, 0x05)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	e.Emit(buf[:]...)
}

func (e *Emitter) AndRdiImm32(v int32) {
	e.Emit(0x48, 0x81, 0xE7)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	e.Emit(buf[:]...)
}

func (e *Emitter) XorEaxEax() {
	e.Emit(0x31, 0xC0)
}

func (e *Emitter) XorEdxEdx() {
	e.Emit(0x31, 0xD2)
}

func (e *Emitter) AddEdxImm32(v int32) {
	e.Emit(0x81, 0xC2)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovRbxRcx() {
	e.Emit(0x48, 0x89, 0xCB)
}

func (e *Emitter) MovRbxR12() {
	e.Emit(0x4C, 0x89, 0xE3)
}

func (e *Emitter) MovRbxRdx() {
	e.Emit(0x48, 0x89, 0xD3)
}

func (e *Emitter) MovR12Rcx() {
	e.Emit(0x49, 0x89, 0xCC)
}

func (e *Emitter) MovR13Rax() {
	e.Emit(0x49, 0x89, 0xC5)
}

func (e *Emitter) MovR13Rcx() {
	e.Emit(0x49, 0x89, 0xCD)
}

func (e *Emitter) MovR13Rdx() {
	e.Emit(0x49, 0x89, 0xD5)
}

func (e *Emitter) MovEaxR13d() {
	e.Emit(0x44, 0x89, 0xE8)
}

func (e *Emitter) MovEcxR13d() {
	e.Emit(0x44, 0x89, 0xE9)
}

func (e *Emitter) ShlRbxImm8(v byte) {
	e.Emit(0x48, 0xC1, 0xE3, v)
}

func (e *Emitter) MovR15Rax() {
	e.Emit(0x49, 0x89, 0xC7)
}

func (e *Emitter) MovRdiR15() {
	e.Emit(0x4C, 0x89, 0xFF)
}

func (e *Emitter) MovRdiRdx() {
	e.Emit(0x48, 0x89, 0xD7)
}

func (e *Emitter) MovRsiRdx() {
	e.Emit(0x48, 0x89, 0xD6)
}

func (e *Emitter) MovR8Rdi() {
	e.Emit(0x49, 0x89, 0xF8)
}

func (e *Emitter) MovEcxEdi() {
	e.Emit(0x89, 0xF9)
}

func (e *Emitter) MovEdiEdx() {
	e.Emit(0x89, 0xD7)
}

func (e *Emitter) MovEdiEax() {
	e.Emit(0x89, 0xC7)
}

func (e *Emitter) MovEcxFromRdiDisp(disp int32) {
	e.Emit(0x8B, 0x8F)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovRaxFromRdiDisp(disp int32) {
	e.Emit(0x48, 0x8B, 0x87)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovRdxFromRdiDisp(disp int32) {
	e.Emit(0x48, 0x8B, 0x97)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem64RdiDispRax(disp int32) {
	e.Emit(0x48, 0x89, 0x87)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem64RdiDispR8(disp int32) {
	if disp == 0 {
		e.Emit(0x4C, 0x89, 0x07)
	} else if disp >= -128 && disp <= 127 {
		e.Emit(0x4C, 0x89, 0x47, byte(disp))
	} else {
		e.Emit(0x4C, 0x89, 0x87)
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(disp))
		e.Emit(buf[:]...)
	}
}

func (e *Emitter) MovMem64RdiDispRsp(disp int32) {
	e.Emit(0x48, 0x89, 0xA7)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovRspFromRdiDisp(disp int32) {
	e.Emit(0x48, 0x8B, 0xA7)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem32RdiDispImm32(disp int32, imm int32) {
	e.Emit(0xC7, 0x87)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
	binary.LittleEndian.PutUint32(buf[:], uint32(imm))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem32RdiDispEdi(disp int32) {
	e.Emit(0x89, 0xBF)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem32RdiDispEsi(disp int32) {
	e.Emit(0x89, 0xB7)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem32RdiDispEax(disp int32) {
	e.Emit(0x89, 0x87)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem32R15DispEax(disp int32) {
	e.Emit(0x41, 0x89, 0x87)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem32RdiDispR8d(disp int32) {
	if disp == 0 {
		e.Emit(0x44, 0x89, 0x07)
	} else if disp >= -128 && disp <= 127 {
		e.Emit(0x44, 0x89, 0x47, byte(disp))
	} else {
		e.Emit(0x44, 0x89, 0x87)
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(disp))
		e.Emit(buf[:]...)
	}
}

func (e *Emitter) XchgMem64RdiPtrR8() {
	e.Emit(0x4C, 0x87, 0x07)
}

func (e *Emitter) XchgMem32RdiPtrR8d() {
	e.Emit(0x44, 0x87, 0x07)
}

func (e *Emitter) XchgMem16RdiPtrR8w() {
	e.Emit(0x66, 0x44, 0x87, 0x07)
}

func (e *Emitter) XchgMem8RdiPtrR8b() {
	e.Emit(0x44, 0x86, 0x07)
}

func (e *Emitter) LockCmpxchgMem64RdiPtrR8() {
	e.Emit(0xF0, 0x4C, 0x0F, 0xB1, 0x07)
}

func (e *Emitter) LockCmpxchgMem32RdiPtrR8d() {
	e.Emit(0xF0, 0x44, 0x0F, 0xB1, 0x07)
}

func (e *Emitter) LockCmpxchgMem16RdiPtrR8w() {
	e.Emit(0xF0, 0x66, 0x44, 0x0F, 0xB1, 0x07)
}

func (e *Emitter) LockCmpxchgMem8RdiPtrR8b() {
	e.Emit(0xF0, 0x44, 0x0F, 0xB0, 0x07)
}

func (e *Emitter) LockCmpxchgMem64RdiPtrR10() {
	e.Emit(0xF0, 0x4C, 0x0F, 0xB1, 0x17)
}

func (e *Emitter) LockCmpxchgMem32RdiPtrR10d() {
	e.Emit(0xF0, 0x44, 0x0F, 0xB1, 0x17)
}

func (e *Emitter) LockCmpxchgMem16RdiPtrR10w() {
	e.Emit(0xF0, 0x66, 0x44, 0x0F, 0xB1, 0x17)
}

func (e *Emitter) LockCmpxchgMem8RdiPtrR10b() {
	e.Emit(0xF0, 0x44, 0x0F, 0xB0, 0x17)
}

func (e *Emitter) LockXaddMem64RdiPtrR8() {
	e.Emit(0xF0, 0x4C, 0x0F, 0xC1, 0x07)
}

func (e *Emitter) LockXaddMem32RdiPtrR8d() {
	e.Emit(0xF0, 0x44, 0x0F, 0xC1, 0x07)
}

func (e *Emitter) LockXaddMem16RdiPtrR8w() {
	e.Emit(0xF0, 0x66, 0x44, 0x0F, 0xC1, 0x07)
}

func (e *Emitter) LockXaddMem8RdiPtrR8b() {
	e.Emit(0xF0, 0x44, 0x0F, 0xC0, 0x07)
}

func (e *Emitter) Mfence() {
	e.Emit(0x0F, 0xAE, 0xF0)
}

func (e *Emitter) CmpEaxImm32(v int32) {
	e.Emit(0x3D)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	e.Emit(buf[:]...)
}

func (e *Emitter) CmpRaxImm32(v int32) {
	e.Emit(0x48, 0x3D)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	e.Emit(buf[:]...)
}

func (e *Emitter) CmpRcxImm32(v int32) {
	e.Emit(0x48, 0x81, 0xF9)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	e.Emit(buf[:]...)
}

func (e *Emitter) CmpEdiImm32(v int32) {
	e.Emit(0x81, 0xFF)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	e.Emit(buf[:]...)
}

func (e *Emitter) CmpEdxImm32(v int32) {
	e.Emit(0x81, 0xFA)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	e.Emit(buf[:]...)
}

func (e *Emitter) CmpEbxImm32(v int32) {
	e.Emit(0x81, 0xFB)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	e.Emit(buf[:]...)
}

func (e *Emitter) JnzRel32() int {
	e.Emit(0x0F, 0x85, 0x00, 0x00, 0x00, 0x00)
	return len(e.Buf) - 4
}

func (e *Emitter) JgeRel32() int {
	e.Emit(0x0F, 0x8D, 0x00, 0x00, 0x00, 0x00)
	return len(e.Buf) - 4
}

func (e *Emitter) MovR11Rsp() {
	e.Emit(0x49, 0x89, 0xE3)
}

func (e *Emitter) MovRspR11() {
	e.Emit(0x4C, 0x89, 0xDC)
}

func (e *Emitter) MovEdxFromRaxDisp(disp int32) {
	// mov edx, [rax+disp]
	if disp == 0 {
		e.Emit(0x8B, 0x10)
	} else {
		e.Emit(0x8B, 0x90)
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(disp))
		e.Emit(buf[:]...)
	}
}

func (e *Emitter) MovR8dFromRaxDisp(disp int32) {
	// mov r8d, [rax+disp]
	if disp == 0 {
		e.Emit(0x44, 0x8B, 0x00)
	} else {
		e.Emit(0x44, 0x8B, 0x80)
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(disp))
		e.Emit(buf[:]...)
	}
}

func (e *Emitter) MovRsiRcx() {
	// mov rsi, rcx
	e.Emit(0x48, 0x89, 0xCE)
}

func (e *Emitter) ShlRsiImm8(v byte) {
	// shl rsi, imm8
	e.Emit(0x48, 0xC1, 0xE6, v)
}

func (e *Emitter) MovEdxFromRaxPtrDisp0() {
	// mov edx, [rax]
	e.Emit(0x8B, 0x10)
}

func (e *Emitter) MovR8dFromRaxPtrDisp4() {
	// mov r8d, [rax+4]
	e.Emit(0x44, 0x8B, 0x40, 0x04)
}

func (e *Emitter) MovR9Rdx() {
	// mov r9, rdx (zero-extends for 32-bit values)
	e.Emit(0x49, 0x89, 0xD1)
}

func (e *Emitter) MovR9Rcx() {
	e.Emit(0x49, 0x89, 0xC9)
}

func (e *Emitter) MovR9R8() {
	e.Emit(0x4D, 0x89, 0xC1)
}

func (e *Emitter) MovR9Rax() {
	e.Emit(0x49, 0x89, 0xC1)
}

func (e *Emitter) MovR8dR8d() {
	e.Emit(0x45, 0x89, 0xC0)
}

func (e *Emitter) MovR9dR8d() {
	e.Emit(0x45, 0x89, 0xC1)
}

func (e *Emitter) MovR10Rax() {
	e.Emit(0x49, 0x89, 0xC2)
}

func (e *Emitter) MovR10dEax() {
	e.Emit(0x41, 0x89, 0xC2)
}

func (e *Emitter) MovRcxR9() {
	e.Emit(0x4C, 0x89, 0xC9)
}

func (e *Emitter) AddR9Rsi() {
	// add r9, rsi
	e.Emit(0x49, 0x01, 0xF1)
}

func (e *Emitter) AddR9Imm32(v int32) {
	e.Emit(0x49, 0x81, 0xC1)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	e.Emit(buf[:]...)
}

func (e *Emitter) AddR8Imm32(v int32) {
	e.Emit(0x49, 0x81, 0xC0)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	e.Emit(buf[:]...)
}

func (e *Emitter) AndR9Imm32(v int32) {
	e.Emit(0x49, 0x81, 0xE1)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	e.Emit(buf[:]...)
}

func (e *Emitter) CmpR9R8() {
	// cmp r9, r8
	e.Emit(0x4D, 0x39, 0xC1)
}

func (e *Emitter) CmpRaxR8() {
	// cmp rax, r8
	e.Emit(0x4C, 0x39, 0xC0)
}

func (e *Emitter) MovMem32RaxPtrFromR9d() {
	// mov [rax], r9d
	e.Emit(0x44, 0x89, 0x08)
}

func (e *Emitter) MovR8dFromRdiRcxScale4() {
	e.Emit(0x44, 0x8B, 0x04, 0x8F)
}

func (e *Emitter) MovR8dFromR9RcxScale4() {
	e.Emit(0x45, 0x8B, 0x04, 0x89)
}

func (e *Emitter) PxorXmm1Xmm1() {
	e.Emit(0x66, 0x0F, 0xEF, 0xC9)
}

func (e *Emitter) PxorXmm0Xmm0() {
	e.Emit(0x66, 0x0F, 0xEF, 0xC0)
}

func (e *Emitter) MovdquXmm0FromR9RcxScale4() {
	e.Emit(0xF3, 0x41, 0x0F, 0x6F, 0x04, 0x89)
}

func (e *Emitter) MovdquR9RcxScale4FromXmm0() {
	e.Emit(0xF3, 0x41, 0x0F, 0x7F, 0x04, 0x89)
}

func (e *Emitter) MovdquXmm0FromR9Rcx() {
	e.Emit(0xF3, 0x41, 0x0F, 0x6F, 0x04, 0x09)
}

func (e *Emitter) MovdquRdiRcxFromXmm0() {
	e.Emit(0xF3, 0x0F, 0x7F, 0x04, 0x0F)
}

func (e *Emitter) PadddXmm1Xmm0() {
	e.Emit(0x66, 0x0F, 0xFE, 0xC8)
}

func (e *Emitter) PadddXmm0Xmm1() {
	e.Emit(0x66, 0x0F, 0xFE, 0xC1)
}

func (e *Emitter) PshufdXmm0Xmm1Imm8(v byte) {
	e.Emit(0x66, 0x0F, 0x70, 0xC1, v)
}

func (e *Emitter) PshufdXmm1Xmm1Imm8(v byte) {
	e.Emit(0x66, 0x0F, 0x70, 0xC9, v)
}

func (e *Emitter) MovdXmm1Eax() {
	e.Emit(0x66, 0x0F, 0x6E, 0xC8)
}

func (e *Emitter) MovdEaxXmm1() {
	e.Emit(0x66, 0x0F, 0x7E, 0xC8)
}

func (e *Emitter) AddMem32R9RcxScale4Imm8(v byte) {
	e.Emit(0x41, 0x83, 0x04, 0x89, v)
}

func (e *Emitter) JaRel32() int {
	// ja rel32 (jump if above, unsigned)
	e.Emit(0x0F, 0x87, 0x00, 0x00, 0x00, 0x00)
	return len(e.Buf) - 4
}

func (e *Emitter) AddRdxRax() {
	// add rdx, rax
	e.Emit(0x48, 0x01, 0xC2)
}

func (e *Emitter) AddRdxRsi() {
	e.Emit(0x48, 0x01, 0xF2)
}

func (e *Emitter) CmpRdxR8() {
	e.Emit(0x4C, 0x39, 0xC2)
}

func (e *Emitter) AndRsiImm32(v int32) {
	e.Emit(0x48, 0x81, 0xE6)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovR8FromRdiDisp(disp int32) {
	e.Emit(0x4C, 0x8B, 0x87)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem64RdiDispRdx(disp int32) {
	e.Emit(0x48, 0x89, 0x97)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem32RaxPtrImm32(disp int32, val int32) {
	// mov dword [rax+disp], imm32
	if disp == 0 {
		e.Emit(0xC7, 0x00)
	} else if disp >= -128 && disp <= 127 {
		e.Emit(0xC7, 0x40, byte(disp))
	} else {
		e.Emit(0xC7, 0x80)
		var dispBuf [4]byte
		binary.LittleEndian.PutUint32(dispBuf[:], uint32(disp))
		e.Emit(dispBuf[:]...)
	}
	var valBuf [4]byte
	binary.LittleEndian.PutUint32(valBuf[:], uint32(val))
	e.Emit(valBuf[:]...)
}

func (e *Emitter) MovMem32Disp32RaxPtrEcx(disp int32) {
	// mov [rax+disp], ecx
	if disp == 0 {
		e.Emit(0x89, 0x08)
	} else if disp >= -128 && disp <= 127 {
		e.Emit(0x89, 0x48, byte(disp))
	} else {
		e.Emit(0x89, 0x88)
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(disp))
		e.Emit(buf[:]...)
	}
}

func (e *Emitter) MovMem8RdiDispDl(disp int32) {
	e.Emit(0x88, 0x97)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(disp))
	e.Emit(buf[:]...)
}

func (e *Emitter) MovMem8RdiRcxMinus1Imm8(v byte) {
	e.Emit(0xC6, 0x44, 0x0F, 0xFF, v)
}

func (e *Emitter) MovMem8RdiRcxDl() {
	e.Emit(0x88, 0x14, 0x0F)
}

func (e *Emitter) DecEcx() {
	e.Emit(0xFF, 0xC9)
}

func (e *Emitter) DivR9() {
	e.Emit(0x49, 0xF7, 0xF1)
}

func (e *Emitter) AddDlImm8(v byte) {
	e.Emit(0x80, 0xC2, v)
}

func (e *Emitter) MovEsiFromRdiDisp(disp int32) {
	// mov esi, [rdi+disp]
	if disp == 0 {
		e.Emit(0x8B, 0x37)
	} else if disp >= -128 && disp <= 127 {
		e.Emit(0x8B, 0x77, byte(disp))
	} else {
		e.Emit(0x8B, 0xB7)
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(disp))
		e.Emit(buf[:]...)
	}
}

func (e *Emitter) MovR8dFromRdiDisp(disp int32) {
	if disp == 0 {
		e.Emit(0x44, 0x8B, 0x07)
	} else if disp >= -128 && disp <= 127 {
		e.Emit(0x44, 0x8B, 0x47, byte(disp))
	} else {
		e.Emit(0x44, 0x8B, 0x87)
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(disp))
		e.Emit(buf[:]...)
	}
}

func (e *Emitter) MovR9dFromRdiDisp(disp int32) {
	if disp == 0 {
		e.Emit(0x44, 0x8B, 0x0F)
	} else if disp >= -128 && disp <= 127 {
		e.Emit(0x44, 0x8B, 0x4F, byte(disp))
	} else {
		e.Emit(0x44, 0x8B, 0x8F)
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(disp))
		e.Emit(buf[:]...)
	}
}

func PatchRel32(code []byte, at int, target int) error {
	if at < 0 || at > len(code)-4 {
		return fmt.Errorf("rel32 patch offset out of range")
	}
	next := at + 4
	disp := target - next
	if disp < -2147483648 || disp > 2147483647 {
		return fmt.Errorf("rel32 target out of range")
	}
	binary.LittleEndian.PutUint32(code[at:at+4], uint32(int32(disp)))
	return nil
}

func AlignStackSize(size int) int {
	if size <= 0 {
		return 0
	}
	rem := size % 16
	if rem == 0 {
		return size
	}
	return size + (16 - rem)
}
