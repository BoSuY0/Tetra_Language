package linuxrt

import (
	"encoding/binary"
	"fmt"

	"tetra_language/compiler/internal/format/tobj"
)

func BuildLinuxX86FilesystemRuntimeObject() *tobj.Object {
	code, err := linuxX86FilesystemExistsCode()
	if err != nil {
		panic(err)
	}
	return &tobj.Object{
		Target: "linux-x86",
		Module: "__linux_x86_fsrt",
		Code:   code,
		Symbols: []tobj.Symbol{{
			Name:         "__tetra_fs_exists",
			Offset:       0,
			HasSignature: true,
			ParamSlots:   3,
			ReturnSlots:  1,
		}},
	}
}

func AppendLinuxX86FilesystemRuntimeObject(rt *tobj.Object) error {
	if rt == nil {
		return fmt.Errorf("missing linux-x86 runtime object")
	}
	for _, sym := range rt.Symbols {
		if sym.Name == "__tetra_fs_exists" {
			return nil
		}
	}
	code, err := linuxX86FilesystemExistsCode()
	if err != nil {
		return err
	}
	offset := len(rt.Code)
	rt.Code = append(rt.Code, code...)
	rt.Symbols = append(rt.Symbols, tobj.Symbol{
		Name:         "__tetra_fs_exists",
		Offset:       uint32(offset),
		HasSignature: true,
		ParamSlots:   3,
		ReturnSlots:  1,
	})
	return nil
}

type linuxX86RuntimeEmitter struct {
	buf []byte
}

func (e *linuxX86RuntimeEmitter) emit(bs ...byte) {
	e.buf = append(e.buf, bs...)
}

func (e *linuxX86RuntimeEmitter) imm32(v uint32) {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	e.emit(buf[:]...)
}

func (e *linuxX86RuntimeEmitter) jccRel32(op1 byte, op2 byte) int {
	e.emit(op1, op2, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *linuxX86RuntimeEmitter) jmpRel32() int {
	e.emit(0xe9, 0, 0, 0, 0)
	return len(e.buf) - 4
}

func (e *linuxX86RuntimeEmitter) patchRel32(at int, target int) error {
	if at < 0 || at+4 > len(e.buf) {
		return fmt.Errorf("invalid linux-x86 runtime rel32 patch offset %d", at)
	}
	rel := int32(target - (at + 4))
	binary.LittleEndian.PutUint32(e.buf[at:at+4], uint32(rel))
	return nil
}

func linuxX86FilesystemExistsCode() ([]byte, error) {
	const (
		linuxSysAccess = 33
		pathBufSize    = 4096
		maxPathLen     = pathBufSize - 1
	)
	e := &linuxX86RuntimeEmitter{}
	var failJumps []int

	e.emit(0x55)             // push ebp
	e.emit(0x89, 0xe5)       // mov ebp, esp
	e.emit(0x53, 0x56, 0x57) // push ebx; push esi; push edi
	e.emit(0x81, 0xec)       // sub esp, pathBufSize
	e.imm32(pathBufSize)
	e.emit(0x8b, 0x75, 0x08)                              // mov esi, [ebp+8] path pointer
	e.emit(0x8b, 0x4d, 0x0c)                              // mov ecx, [ebp+12] path length
	e.emit(0x85, 0xf6)                                    // test esi, esi
	failJumps = append(failJumps, e.jccRel32(0x0f, 0x84)) // jz fail
	e.emit(0x85, 0xc9)                                    // test ecx, ecx
	failJumps = append(failJumps, e.jccRel32(0x0f, 0x84)) // jz fail
	e.emit(0x81, 0xf9)                                    // cmp ecx, maxPathLen
	e.imm32(maxPathLen)
	failJumps = append(failJumps, e.jccRel32(0x0f, 0x87)) // ja fail

	e.emit(0x8d, 0x3c, 0x24) // lea edi, [esp]
	e.emit(0x31, 0xc0)       // xor eax, eax

	copyLoop := len(e.buf)
	e.emit(0x39, 0xc8)                                    // cmp eax, ecx
	copiedAt := e.jccRel32(0x0f, 0x83)                    // jae copied
	e.emit(0x8a, 0x14, 0x06)                              // mov dl, [esi+eax]
	e.emit(0x84, 0xd2)                                    // test dl, dl
	failJumps = append(failJumps, e.jccRel32(0x0f, 0x84)) // jz fail
	e.emit(0x88, 0x14, 0x07)                              // mov [edi+eax], dl
	e.emit(0x40)                                          // inc eax
	backAt := e.jmpRel32()
	if err := e.patchRel32(backAt, copyLoop); err != nil {
		return nil, err
	}

	copiedTo := len(e.buf)
	if err := e.patchRel32(copiedAt, copiedTo); err != nil {
		return nil, err
	}
	e.emit(0xc6, 0x04, 0x0f, 0x00) // mov byte ptr [edi+ecx], 0
	e.emit(0xb8)
	e.imm32(linuxSysAccess)  // mov eax, SYS_access
	e.emit(0x89, 0xfb)       // mov ebx, edi
	e.emit(0x31, 0xc9)       // xor ecx, ecx (F_OK)
	e.emit(0xcd, 0x80)       // int 0x80
	e.emit(0x85, 0xc0)       // test eax, eax
	e.emit(0x0f, 0x94, 0xc0) // sete al
	e.emit(0x0f, 0xb6, 0xc0) // movzx eax, al
	e.emit(0x81, 0xc4)       // add esp, pathBufSize
	e.imm32(pathBufSize)
	e.emit(0x5f, 0x5e, 0x5b) // pop edi; pop esi; pop ebx
	e.emit(0x5d, 0xc3)       // pop ebp; ret

	failTo := len(e.buf)
	for _, at := range failJumps {
		if err := e.patchRel32(at, failTo); err != nil {
			return nil, err
		}
	}
	e.emit(0x31, 0xc0) // xor eax, eax
	e.emit(0x81, 0xc4) // add esp, pathBufSize
	e.imm32(pathBufSize)
	e.emit(0x5f, 0x5e, 0x5b) // pop edi; pop esi; pop ebx
	e.emit(0x5d, 0xc3)       // pop ebp; ret
	return e.buf, nil
}
