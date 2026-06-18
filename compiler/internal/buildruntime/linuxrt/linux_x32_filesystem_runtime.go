package linuxrt

import (
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/format/tobj"
)

const linuxX32SyscallBit = 0x40000000

func BuildLinuxX32FilesystemRuntimeObject() *tobj.Object {
	code, err := linuxX32FilesystemExistsCode()
	if err != nil {
		panic(err)
	}
	return &tobj.Object{
		Target: "linux-x32",
		Module: "__linux_x32_fsrt",
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

func AppendLinuxX32FilesystemRuntimeObject(rt *tobj.Object) error {
	if rt == nil {
		return fmt.Errorf("missing linux-x32 runtime object")
	}
	for _, sym := range rt.Symbols {
		if sym.Name == "__tetra_fs_exists" {
			return nil
		}
	}
	code, err := linuxX32FilesystemExistsCode()
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

func linuxX32FilesystemExistsCode() ([]byte, error) {
	const (
		linuxSysAccess = linuxX32SyscallBit + 21
		pathBufSize    = 4096
		maxPathLen     = pathBufSize - 1
	)
	e := &x64.Emitter{}
	var failJumps []int

	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(pathBufSize)

	// x32 uses the x86_64 register ABI with 32-bit pointer/native-int facts.
	e.Emit(0x48, 0x85, 0xff) // test rdi, rdi
	failJumps = append(failJumps, e.JzRel32())
	e.Emit(0x85, 0xf6) // test esi, esi
	failJumps = append(failJumps, e.JzRel32())
	e.Emit(0x81, 0xfe, 0xff, 0x0f, 0x00, 0x00) // cmp esi, 4095
	failJumps = append(failJumps, e.JaRel32())

	e.Emit(0x48, 0x89, 0xf1)       // mov rcx, rsi
	e.Emit(0x49, 0x89, 0xf8)       // mov r8, rdi
	e.Emit(0x4c, 0x8d, 0x0c, 0x24) // lea r9, [rsp]
	e.XorEaxEax()

	copyLoop := len(e.Buf)
	e.Emit(0x48, 0x39, 0xc8) // cmp rax, rcx
	copiedAt := e.JaeRel32()
	e.Emit(0x41, 0x8a, 0x14, 0x00) // mov dl, byte ptr [r8+rax]
	e.Emit(0x84, 0xd2)             // test dl, dl
	failJumps = append(failJumps, e.JzRel32())
	e.Emit(0x41, 0x88, 0x14, 0x01) // mov byte ptr [r9+rax], dl
	e.Emit(0x48, 0xff, 0xc0)       // inc rax
	backAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backAt, copyLoop); err != nil {
		return nil, err
	}

	copiedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, copiedAt, copiedTo); err != nil {
		return nil, err
	}
	e.Emit(0x41, 0xc6, 0x04, 0x09, 0x00) // mov byte ptr [r9+rcx], 0
	e.Emit(0x4c, 0x89, 0xcf)             // mov rdi, r9
	e.Emit(0x31, 0xf6)                   // xor esi, esi (F_OK)
	e.MovEaxImm32(linuxSysAccess)
	e.Syscall()
	e.TestEaxEax()
	e.SeteAl()
	e.MovzxEaxAl()
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return nil, err
		}
	}
	e.XorEaxEax()
	e.Leave()
	e.Ret()
	return e.Buf, nil
}
