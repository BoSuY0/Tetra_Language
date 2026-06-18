package linux_x86

import (
	"encoding/binary"
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
)

func CodegenObjectLinuxX86(funcs []ir.IRFunc) (*tobj.Object, error) {
	return CodegenObjectLinuxX86WithDataPrefix(funcs, nil)
}

func CodegenObjectLinuxX86WithOptions(
	funcs []ir.IRFunc,
	opt x64.CodegenOptions,
) (*tobj.Object, error) {
	return CodegenObjectLinuxX86WithOptionsAndDataPrefix(funcs, nil, opt)
}

func CodegenObjectLinuxX86WithDataPrefix(
	funcs []ir.IRFunc,
	dataPrefix [][]byte,
) (*tobj.Object, error) {
	return CodegenObjectLinuxX86WithOptionsAndDataPrefix(funcs, dataPrefix, x64.CodegenOptions{})
}

func CodegenObjectLinuxX86WithOptionsAndDataPrefix(
	funcs []ir.IRFunc,
	dataPrefix [][]byte,
	opt x64.CodegenOptions,
) (*tobj.Object, error) {
	obj, err := buildObject(funcs, dataPrefix, opt)
	if err != nil {
		return nil, err
	}
	obj.Target = "linux-x86"
	return obj, nil
}

type emitter struct {
	buf []byte
}

type labelPatch struct {
	at    int
	label int
}

type callPatch struct {
	at   int
	name string
}

type dataPatch struct {
	at        int
	dataIndex int
}

type funcAddrPatch struct {
	at   int
	name string
}

func (e *emitter) emit(bs ...byte) {
	e.buf = append(e.buf, bs...)
}

func (e *emitter) imm32(v uint32) {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	e.emit(buf[:]...)
}

func (e *emitter) patchRel32(at int, target int) error {
	if at < 0 || at+4 > len(e.buf) {
		return fmt.Errorf("invalid rel32 patch offset %d", at)
	}
	rel := int32(target - (at + 4))
	binary.LittleEndian.PutUint32(e.buf[at:at+4], uint32(rel))
	return nil
}

func (e *emitter) pushEbp()            { e.emit(0x55) }
func (e *emitter) movEbpEsp()          { e.emit(0x89, 0xE5) }
func (e *emitter) leave()              { e.emit(0xC9) }
func (e *emitter) ret()                { e.emit(0xC3) }
func (e *emitter) pushEax()            { e.emit(0x50) }
func (e *emitter) pushEcx()            { e.emit(0x51) }
func (e *emitter) pushEdx()            { e.emit(0x52) }
func (e *emitter) pushEbx()            { e.emit(0x53) }
func (e *emitter) pushEsi()            { e.emit(0x56) }
func (e *emitter) pushEdi()            { e.emit(0x57) }
func (e *emitter) popEax()             { e.emit(0x58) }
func (e *emitter) popEcx()             { e.emit(0x59) }
func (e *emitter) popEdx()             { e.emit(0x5A) }
func (e *emitter) popEbx()             { e.emit(0x5B) }
func (e *emitter) popEbp()             { e.emit(0x5D) }
func (e *emitter) popEsi()             { e.emit(0x5E) }
func (e *emitter) popEdi()             { e.emit(0x5F) }
func (e *emitter) cdq()                { e.emit(0x99) }
func (e *emitter) int80()              { e.emit(0xCD, 0x80) }
func (e *emitter) mfenceCompat()       { e.emit(0xF0, 0x83, 0x0C, 0x24, 0x00) }
func (e *emitter) xorEaxEax()          { e.emit(0x31, 0xC0) }
func (e *emitter) movEaxEcx()          { e.emit(0x89, 0xC8) }
func (e *emitter) movEcxEax()          { e.emit(0x89, 0xC1) }
func (e *emitter) movEdxEax()          { e.emit(0x89, 0xC2) }
func (e *emitter) movEdiEax()          { e.emit(0x89, 0xC7) }
func (e *emitter) movEaxEdi()          { e.emit(0x89, 0xF8) }
func (e *emitter) movEdiEdx()          { e.emit(0x89, 0xD7) }
func (e *emitter) movEbxEax()          { e.emit(0x89, 0xC3) }
func (e *emitter) movEbxEcx()          { e.emit(0x89, 0xCB) }
func (e *emitter) movEaxEcxValue()     { e.emit(0x89, 0xC8) }
func (e *emitter) addEaxEcx()          { e.emit(0x01, 0xC8) }
func (e *emitter) addEaxEdx()          { e.emit(0x01, 0xD0) }
func (e *emitter) addEdxEax()          { e.emit(0x01, 0xC2) }
func (e *emitter) addEdiEcx()          { e.emit(0x01, 0xCF) }
func (e *emitter) subEaxEcx()          { e.emit(0x29, 0xC8) }
func (e *emitter) subEcxEdx()          { e.emit(0x29, 0xD1) }
func (e *emitter) subEaxEdi()          { e.emit(0x29, 0xF8) }
func (e *emitter) imulEaxEcx()         { e.emit(0x0F, 0xAF, 0xC1) }
func (e *emitter) idivEcx()            { e.emit(0xF7, 0xF9) }
func (e *emitter) negEax()             { e.emit(0xF7, 0xD8) }
func (e *emitter) negEcx()             { e.emit(0xF7, 0xD9) }
func (e *emitter) cmpEaxEcx()          { e.emit(0x39, 0xC8) }
func (e *emitter) cmpEdxEcx()          { e.emit(0x39, 0xCA) }
func (e *emitter) cmpEbxEcx()          { e.emit(0x39, 0xCB) }
func (e *emitter) cmpEdiEbx()          { e.emit(0x39, 0xDF) }
func (e *emitter) testEaxEax()         { e.emit(0x85, 0xC0) }
func (e *emitter) testEcxEcx()         { e.emit(0x85, 0xC9) }
func (e *emitter) xorEbxEbx()          { e.emit(0x31, 0xDB) }
func (e *emitter) xorEbpEbp()          { e.emit(0x31, 0xED) }
func (e *emitter) movzxEaxAl()         { e.emit(0x0F, 0xB6, 0xC0) }
func (e *emitter) movzxEaxAx()         { e.emit(0x0F, 0xB7, 0xC0) }
func (e *emitter) movEaxFromEaxPtr()   { e.emit(0x8B, 0x00) }
func (e *emitter) movEdxFromEaxPtr()   { e.emit(0x8B, 0x10) }
func (e *emitter) movEaxFromEdiPtr()   { e.emit(0x8B, 0x07) }
func (e *emitter) movEcxFromEdiPtr()   { e.emit(0x8B, 0x0F) }
func (e *emitter) movzxEaxByteEaxPtr() { e.emit(0x0F, 0xB6, 0x00) }
func (e *emitter) movzxEaxByteEdiPtr() { e.emit(0x0F, 0xB6, 0x07) }
func (e *emitter) movzxEaxWordEaxPtr() { e.emit(0x0F, 0xB7, 0x00) }
func (e *emitter) movzxEaxWordEdiPtr() { e.emit(0x0F, 0xB7, 0x07) }
func (e *emitter) movMemEaxPtrEcx()    { e.emit(0x89, 0x08) }
func (e *emitter) movMemEaxPtrEbx()    { e.emit(0x89, 0x18) }
func (e *emitter) movMemEaxPtrEsp()    { e.emit(0x89, 0x20) }
func (e *emitter) movMemEaxPtrEdi()    { e.emit(0x89, 0x38) }
func (e *emitter) movEspFromEcxPtr()   { e.emit(0x8B, 0x21) }
func (e *emitter) movMem8EaxPtrBl()    { e.emit(0x88, 0x18) }
func (e *emitter) movMem16EaxPtrBx()   { e.emit(0x66, 0x89, 0x18) }
func (e *emitter) xchgMem32EdiEcx()    { e.emit(0x87, 0x0F) }
func (e *emitter) xchgMem8EdiCl()      { e.emit(0x86, 0x0F) }
func (e *emitter) xchgMem16EdiCx()     { e.emit(0x66, 0x87, 0x0F) }
func (e *emitter) lockXaddMem32EdiEcx() {
	e.emit(0xF0, 0x0F, 0xC1, 0x0F)
}
func (e *emitter) lockXaddMem8EdiCl() {
	e.emit(0xF0, 0x0F, 0xC0, 0x0F)
}
func (e *emitter) lockXaddMem16EdiCx() {
	e.emit(0x66, 0xF0, 0x0F, 0xC1, 0x0F)
}
func (e *emitter) lockCmpxchgMem32EdiEdx() {
	e.emit(0xF0, 0x0F, 0xB1, 0x17)
}
func (e *emitter) lockCmpxchgMem32EdiEbx() {
	e.emit(0xF0, 0x0F, 0xB1, 0x1F)
}
func (e *emitter) lockCmpxchgMem8EdiDl() {
	e.emit(0xF0, 0x0F, 0xB0, 0x17)
}
func (e *emitter) lockCmpxchgMem16EdiDx() {
	e.emit(0x66, 0xF0, 0x0F, 0xB1, 0x17)
}
func (e *emitter) andEbxEcx() { e.emit(0x21, 0xCB) }
func (e *emitter) orEbxEcx()  { e.emit(0x09, 0xCB) }
func (e *emitter) xorEbxEcx() { e.emit(0x31, 0xCB) }
func (e *emitter) andDlCl()   { e.emit(0x20, 0xCA) }
func (e *emitter) orDlCl()    { e.emit(0x08, 0xCA) }
func (e *emitter) xorDlCl()   { e.emit(0x30, 0xCA) }
func (e *emitter) andDxCx()   { e.emit(0x66, 0x21, 0xCA) }
func (e *emitter) orDxCx()    { e.emit(0x66, 0x09, 0xCA) }
func (e *emitter) xorDxCx()   { e.emit(0x66, 0x31, 0xCA) }

func (e *emitter) subEspImm32(v int32) {
	e.emit(0x81, 0xEC)
	e.imm32(uint32(v))
}

func (e *emitter) addEspImm32(v int32) {
	e.emit(0x81, 0xC4)
	e.imm32(uint32(v))
}

func (e *emitter) addEaxImm8(v byte) {
	e.emit(0x83, 0xC0, v)
}

func (e *emitter) addEdxImm8(v byte) {
	e.emit(0x83, 0xC2, v)
}

func (e *emitter) subEdxImm8(v byte) {
	e.emit(0x83, 0xEA, v)
}

func (e *emitter) addEdiImm8(v byte) {
	e.emit(0x83, 0xC7, v)
}

func (e *emitter) addEcxImm32(v int32) {
	e.emit(0x81, 0xC1)
	e.imm32(uint32(v))
}

func (e *emitter) subEcxImm32(v int32) {
	e.emit(0x81, 0xE9)
	e.imm32(uint32(v))
}

func (e *emitter) addEbxImm32(v int32) {
	e.emit(0x81, 0xC3)
	e.imm32(uint32(v))
}

func (e *emitter) movEaxImm32(v uint32) {
	e.emit(0xB8)
	e.imm32(v)
}

func (e *emitter) movEbxImm32(v uint32) {
	e.emit(0xBB)
	e.imm32(v)
}

func (e *emitter) movEdxImm32(v uint32) {
	e.emit(0xBA)
	e.imm32(v)
}

func (e *emitter) movEsiImm32(v uint32) {
	e.emit(0xBE)
	e.imm32(v)
}

func (e *emitter) movEdiImm32(v uint32) {
	e.emit(0xBF)
	e.imm32(v)
}

func (e *emitter) movEaxImm32Patch() int {
	e.emit(0xB8, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *emitter) movEaxFromEbpDisp(disp int32) {
	e.emit(0x8B, 0x85)
	e.imm32(uint32(disp))
}

func (e *emitter) movMemEbpDispEax(disp int32) {
	e.emit(0x89, 0x85)
	e.imm32(uint32(disp))
}

func (e *emitter) movEaxFromEspDisp(disp int32) {
	e.emit(0x8B, 0x84, 0x24)
	e.imm32(uint32(disp))
}

func (e *emitter) movMemEspDispEax(disp int32) {
	e.emit(0x89, 0x84, 0x24)
	e.imm32(uint32(disp))
}

func (e *emitter) movMemEbpDispImm32(disp int32, v uint32) {
	e.emit(0xC7, 0x85)
	e.imm32(uint32(disp))
	e.imm32(v)
}

func (e *emitter) movMemEaxPtrImm32(v uint32) {
	e.emit(0xC7, 0x00)
	e.imm32(v)
}

func (e *emitter) movMemEaxDispEcx(disp byte) {
	e.emit(0x89, 0x48, disp)
}

func (e *emitter) movMemEaxDispImm32(disp byte, v uint32) {
	e.emit(0xC7, 0x40, disp)
	e.imm32(v)
}

func (e *emitter) movEbxFromEaxDisp(disp byte) {
	e.emit(0x8B, 0x58, disp)
}

func (e *emitter) movEaxFromEbxDisp(disp byte) {
	e.emit(0x8B, 0x43, disp)
}

func (e *emitter) movEcxFromEbxDisp(disp byte) {
	e.emit(0x8B, 0x4B, disp)
}

func (e *emitter) movMemEbxDispImm32(disp byte, v uint32) {
	e.emit(0xC7, 0x43, disp)
	e.imm32(v)
}

func (e *emitter) andEdiImm32(v int32) {
	e.emit(0x81, 0xE7)
	e.imm32(uint32(v))
}

func (e *emitter) cmpEcxImm8(v byte) {
	e.emit(0x83, 0xF9, v)
}

func (e *emitter) cmpEcxImm32(v uint32) {
	e.emit(0x81, 0xF9)
	e.imm32(v)
}

func (e *emitter) cmpEdxImm8(v byte) {
	e.emit(0x83, 0xFA, v)
}

func (e *emitter) cmpEbxImm32(v uint32) {
	e.emit(0x81, 0xFB)
	e.imm32(v)
}

func (e *emitter) cmpEaxImm32(v uint32) {
	e.emit(0x3D)
	e.imm32(v)
}

func (e *emitter) shlEcxImm8(v byte) {
	e.emit(0xC1, 0xE1, v)
}

func (e *emitter) shlEdxImm8(v byte) {
	e.emit(0xC1, 0xE2, v)
}

func (e *emitter) setccAl(op byte) {
	e.emit(0x0F, op, 0xC0)
}

func (e *emitter) jgeRel32() int {
	e.emit(0x0F, 0x8D, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *emitter) jaeRel32() int {
	e.emit(0x0F, 0x83, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *emitter) jaRel32() int {
	e.emit(0x0F, 0x87, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *emitter) jlRel32() int {
	e.emit(0x0F, 0x8C, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *emitter) jgRel32() int {
	e.emit(0x0F, 0x8F, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *emitter) jnzRel32() int {
	e.emit(0x0F, 0x85, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *emitter) jzRel32() int {
	e.emit(0x0F, 0x84, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *emitter) jmpRel32() int {
	e.emit(0xE9, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *emitter) callRel32() int {
	e.emit(0xE8, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *emitter) movEaxFromAbs32() int {
	e.emit(0xA1, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *emitter) movAbs32FromEax() int {
	e.emit(0xA3, 0, 0, 0, 0)
	return len(e.buf) - 4
}
