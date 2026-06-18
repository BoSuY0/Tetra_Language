package linuxrt

import (
	"encoding/binary"
	"fmt"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/format/tobj"
)

var linuxX32BasicNetRuntimeSymbols = []struct {
	name        string
	code        func() []byte
	paramSlots  int
	returnSlots int
}{
	{
		name:        "__tetra_net_socket_tcp4",
		code:        linuxX32NetSocketTCP4Code,
		paramSlots:  1,
		returnSlots: 1,
	},
	{
		name:        "__tetra_net_bind_tcp4_loopback",
		code:        func() []byte { return linuxX32NetTCP4LoopbackCode(linuxX32SysBind) },
		paramSlots:  3,
		returnSlots: 1,
	},
	{
		name:        "__tetra_net_connect_tcp4_loopback",
		code:        func() []byte { return linuxX32NetTCP4LoopbackCode(linuxX32SysConnect) },
		paramSlots:  3,
		returnSlots: 1,
	},
	{name: "__tetra_net_listen", code: linuxX32NetListenCode, paramSlots: 3, returnSlots: 1},
	{name: "__tetra_net_accept4", code: linuxX32NetAccept4Code, paramSlots: 3, returnSlots: 1},
	{
		name:        "__tetra_net_read",
		code:        func() []byte { return linuxX32NetReadWriteCode(linuxX32SysRead) },
		paramSlots:  6,
		returnSlots: 1,
	},
	{
		name:        "__tetra_net_recv",
		code:        func() []byte { return linuxX32NetRecvSendCode(linuxX32SysRecvfrom) },
		paramSlots:  6,
		returnSlots: 1,
	},
	{
		name:        "__tetra_net_write",
		code:        func() []byte { return linuxX32NetReadWriteCode(linuxX32SysWrite) },
		paramSlots:  6,
		returnSlots: 1,
	},
	{
		name:        "__tetra_net_send",
		code:        func() []byte { return linuxX32NetRecvSendCode(linuxX32SysSendto) },
		paramSlots:  6,
		returnSlots: 1,
	},
	{
		name:        "__tetra_net_epoll_create",
		code:        linuxX32NetEpollCreateCode,
		paramSlots:  1,
		returnSlots: 1,
	},
	{
		name: "__tetra_net_epoll_ctl_add_read",
		code: func() []byte {
			return linuxX32NetEpollCtlCode(
				linuxX32EpollCtlAdd,
				linuxX32EpollIn,
			)
		},
		paramSlots:  3,
		returnSlots: 1,
	},
	{
		name: "__tetra_net_epoll_ctl_add_read_write",
		code: func() []byte {
			return linuxX32NetEpollCtlCode(
				linuxX32EpollCtlAdd,
				linuxX32EpollIn|linuxX32EpollOut,
			)
		},
		paramSlots:  3,
		returnSlots: 1,
	},
	{
		name: "__tetra_net_epoll_ctl_mod_read",
		code: func() []byte {
			return linuxX32NetEpollCtlCode(
				linuxX32EpollCtlMod,
				linuxX32EpollIn,
			)
		},
		paramSlots:  3,
		returnSlots: 1,
	},
	{
		name: "__tetra_net_epoll_ctl_mod_read_write",
		code: func() []byte {
			return linuxX32NetEpollCtlCode(
				linuxX32EpollCtlMod,
				linuxX32EpollIn|linuxX32EpollOut,
			)
		},
		paramSlots:  3,
		returnSlots: 1,
	},
	{
		name:        "__tetra_net_epoll_ctl_delete",
		code:        func() []byte { return linuxX32NetEpollCtlCode(linuxX32EpollCtlDel, 0) },
		paramSlots:  3,
		returnSlots: 1,
	},
	{
		name:        "__tetra_net_epoll_wait_one",
		code:        linuxX32NetEpollWaitOneCode,
		paramSlots:  3,
		returnSlots: 1,
	},
	{
		name:        "__tetra_net_epoll_wait_one_into",
		code:        linuxX32NetEpollWaitOneIntoCode,
		paramSlots:  5,
		returnSlots: 1,
	},
	{
		name:        "__tetra_net_set_nonblocking",
		code:        linuxX32NetSetNonblockingCode,
		paramSlots:  2,
		returnSlots: 1,
	},
	{
		name: "__tetra_net_set_reuseport",
		code: func() []byte {
			return linuxX32NetSetSockOptCode(
				linuxX32SolSocket,
				linuxX32SoReusePort,
			)
		},
		paramSlots:  2,
		returnSlots: 1,
	},
	{
		name: "__tetra_net_set_tcp_nodelay",
		code: func() []byte {
			return linuxX32NetSetSockOptCode(
				linuxX32IPProtoTCP,
				linuxX32TCPNoDelay,
			)
		},
		paramSlots:  2,
		returnSlots: 1,
	},
	{name: "__tetra_net_close", code: linuxX32NetCloseCode, paramSlots: 2, returnSlots: 1},
}

const (
	linuxX32SysRead         = linuxX32SyscallBit + 0
	linuxX32SysWrite        = linuxX32SyscallBit + 1
	linuxX32SysClose        = linuxX32SyscallBit + 3
	linuxX32SysSocket       = linuxX32SyscallBit + 41
	linuxX32SysConnect      = linuxX32SyscallBit + 42
	linuxX32SysSendto       = linuxX32SyscallBit + 44
	linuxX32SysBind         = linuxX32SyscallBit + 49
	linuxX32SysListen       = linuxX32SyscallBit + 50
	linuxX32SysFcntl        = linuxX32SyscallBit + 72
	linuxX32SysEpollWait    = linuxX32SyscallBit + 232
	linuxX32SysEpollCtl     = linuxX32SyscallBit + 233
	linuxX32SysAccept4      = linuxX32SyscallBit + 288
	linuxX32SysEpollCreate1 = linuxX32SyscallBit + 291
	linuxX32SysRecvfrom     = linuxX32SyscallBit + 517
	linuxX32SysSetSockOpt   = linuxX32SyscallBit + 541

	linuxX32AFInet       = 2
	linuxX32SockStream   = 1
	linuxX32SolSocket    = 1
	linuxX32SoReusePort  = 15
	linuxX32IPProtoTCP   = 6
	linuxX32TCPNoDelay   = 1
	linuxX32FGetFL       = 3
	linuxX32FSetFL       = 4
	linuxX32ONonblock    = 0x800
	linuxX32MaxTCPPort   = 65535
	linuxX32LoopbackAddr = 0x0100007f
	linuxX32EpollCtlAdd  = 1
	linuxX32EpollCtlDel  = 2
	linuxX32EpollCtlMod  = 3
	linuxX32EpollIn      = 1
	linuxX32EpollOut     = 4
)

func BuildLinuxX32BasicNetRuntimeObject() *tobj.Object {
	code := make([]byte, 0)
	symbols := make([]tobj.Symbol, 0, len(linuxX32BasicNetRuntimeSymbols))
	for _, spec := range linuxX32BasicNetRuntimeSymbols {
		offset := len(code)
		code = append(code, spec.code()...)
		symbols = append(symbols, tobj.Symbol{
			Name:         spec.name,
			Offset:       uint32(offset),
			HasSignature: true,
			ParamSlots:   spec.paramSlots,
			ReturnSlots:  spec.returnSlots,
		})
	}
	return &tobj.Object{
		Target:  "linux-x32",
		Module:  "__linux_x32_netrt",
		Code:    code,
		Symbols: symbols,
	}
}

func AppendLinuxX32BasicNetRuntimeObject(rt *tobj.Object) error {
	if rt == nil {
		return fmt.Errorf("missing linux-x32 runtime object")
	}
	existing := make(map[string]struct{}, len(rt.Symbols))
	for _, sym := range rt.Symbols {
		existing[sym.Name] = struct{}{}
	}
	for _, spec := range linuxX32BasicNetRuntimeSymbols {
		if _, ok := existing[spec.name]; ok {
			continue
		}
		offset := len(rt.Code)
		rt.Code = append(rt.Code, spec.code()...)
		rt.Symbols = append(rt.Symbols, tobj.Symbol{
			Name:         spec.name,
			Offset:       uint32(offset),
			HasSignature: true,
			ParamSlots:   spec.paramSlots,
			ReturnSlots:  spec.returnSlots,
		})
	}
	return nil
}

func linuxX32NetSocketTCP4Code() []byte {
	e := &x64.Emitter{}
	e.MovEdiImm32(linuxX32AFInet)
	e.Emit(0xBE, byte(linuxX32SockStream), 0, 0, 0) // mov esi, SOCK_STREAM
	e.Emit(0x31, 0xD2)                              // xor edx, edx
	e.MovEaxImm32(linuxX32SysSocket)
	e.Syscall()
	e.Ret()
	return e.Buf
}

func linuxX32NetTCP4LoopbackCode(syscall uint32) []byte {
	e := &x64.Emitter{}
	failJumps := linuxX32EmitRejectInvalidTCPPort(e)
	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(32)
	e.Emit(0x89, 0x7C, 0x24, 0x10) // mov [rsp+16], edi

	linuxX32EmitMovMem16RspDispImm16(e, 0, linuxX32AFInet)
	e.MovEaxEsi()
	e.Emit(0x86, 0xE0)                   // xchg al, ah
	e.Emit(0x66, 0x89, 0x44, 0x24, 0x02) // mov [rsp+2], ax
	linuxX32EmitMovMem32RspDispImm32(e, 4, linuxX32LoopbackAddr)
	linuxX32EmitMovMem32RspDispImm32(e, 8, 0)
	linuxX32EmitMovMem32RspDispImm32(e, 12, 0)
	e.Emit(0x8B, 0x7C, 0x24, 0x10) // mov edi, [rsp+16]
	e.Emit(0x48, 0x8D, 0x34, 0x24) // lea rsi, [rsp]
	e.MovEdxImm32(16)
	e.MovEaxImm32(syscall)
	e.Syscall()
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			panic(err)
		}
	}
	e.MovEaxImm32(0xffffffff)
	e.Ret()
	return e.Buf
}

func linuxX32EmitRejectInvalidTCPPort(e *x64.Emitter) []int {
	var failJumps []int
	e.Emit(0x85, 0xF6) // test esi, esi
	nonNegativeAt := e.JgeRel32()
	failJumps = append(failJumps, e.JmpRel32())
	if err := x64.PatchRel32(e.Buf, nonNegativeAt, len(e.Buf)); err != nil {
		panic(err)
	}
	e.Emit(0x81, 0xFE, 0xFF, 0xFF, 0x00, 0x00) // cmp esi, 65535
	failJumps = append(failJumps, e.JaRel32())
	return failJumps
}

func linuxX32NetListenCode() []byte {
	e := &x64.Emitter{}
	e.MovEaxImm32(linuxX32SysListen)
	e.Syscall()
	e.Ret()
	return e.Buf
}

func linuxX32NetAccept4Code() []byte {
	e := &x64.Emitter{}
	e.Emit(0x41, 0x89, 0xF2) // mov r10d, esi
	e.Emit(0x31, 0xF6)       // xor esi, esi
	e.Emit(0x31, 0xD2)       // xor edx, edx
	e.MovEaxImm32(linuxX32SysAccept4)
	e.Syscall()
	e.Ret()
	return e.Buf
}

func linuxX32NetReadWriteCode(syscall uint32) []byte {
	e := &x64.Emitter{}
	failJumps := linuxX32EmitNetSliceBoundsCheck(e)
	e.Emit(0x48, 0x63, 0xC9) // movsxd rcx, ecx
	e.Emit(0x48, 0x01, 0xCE) // add rsi, rcx
	e.MovRdxR8()
	e.MovEaxImm32(syscall)
	e.Syscall()
	e.Ret()
	linuxX32EmitNetSliceFailure(e, failJumps)
	return e.Buf
}

func linuxX32NetRecvSendCode(syscall uint32) []byte {
	e := &x64.Emitter{}
	failJumps := linuxX32EmitNetSliceBoundsCheck(e)
	e.Emit(0x48, 0x63, 0xC9) // movsxd rcx, ecx
	e.Emit(0x48, 0x01, 0xCE) // add rsi, rcx
	e.MovRdxR8()
	e.MovR10dImm32(0)
	e.MovR8dImm32(0)
	e.MovR9dImm32(0)
	e.MovEaxImm32(syscall)
	e.Syscall()
	e.Ret()
	linuxX32EmitNetSliceFailure(e, failJumps)
	return e.Buf
}

func linuxX32EmitNetSliceBoundsCheck(e *x64.Emitter) []int {
	var failJumps []int
	e.Emit(0x85, 0xC9) // test ecx, ecx
	startOK := e.JgeRel32()
	failJumps = append(failJumps, e.JmpRel32())
	if err := x64.PatchRel32(e.Buf, startOK, len(e.Buf)); err != nil {
		panic(err)
	}
	e.Emit(0x45, 0x85, 0xC0) // test r8d, r8d
	countOK := e.JgeRel32()
	failJumps = append(failJumps, e.JmpRel32())
	if err := x64.PatchRel32(e.Buf, countOK, len(e.Buf)); err != nil {
		panic(err)
	}
	e.Emit(0x39, 0xCA) // cmp edx, ecx
	startInRange := e.JgeRel32()
	failJumps = append(failJumps, e.JmpRel32())
	if err := x64.PatchRel32(e.Buf, startInRange, len(e.Buf)); err != nil {
		panic(err)
	}
	e.Emit(0x29, 0xCA)       // sub edx, ecx
	e.Emit(0x44, 0x39, 0xC2) // cmp edx, r8d
	useRequestedCount := e.JgeRel32()
	e.Emit(0x41, 0x89, 0xD0) // mov r8d, edx
	if err := x64.PatchRel32(e.Buf, useRequestedCount, len(e.Buf)); err != nil {
		panic(err)
	}
	return failJumps
}

func linuxX32EmitNetSliceFailure(e *x64.Emitter, failJumps []int) {
	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			panic(err)
		}
	}
	e.MovEaxImm32(0xffffffff)
	e.Ret()
}

func linuxX32NetEpollCreateCode() []byte {
	e := &x64.Emitter{}
	e.MovEdiImm32(0)
	e.MovEaxImm32(linuxX32SysEpollCreate1)
	e.Syscall()
	e.Ret()
	return e.Buf
}

func linuxX32NetEpollCtlCode(op uint32, events uint32) []byte {
	e := &x64.Emitter{}
	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(16)
	linuxX32EmitMovMem32RspDispImm32(e, 0, events)
	e.Emit(0x89, 0x74, 0x24, 0x04) // mov [rsp+4], esi
	linuxX32EmitMovMem32RspDispImm32(e, 8, 0)
	e.Emit(0x89, 0xF2) // mov edx, esi
	e.Emit(0xBE, byte(op&0xff), byte((op>>8)&0xff), byte((op>>16)&0xff), byte((op>>24)&0xff))
	e.Emit(0x49, 0x89, 0xE2) // mov r10, rsp
	e.MovEaxImm32(linuxX32SysEpollCtl)
	e.Syscall()
	e.Leave()
	e.Ret()
	return e.Buf
}

func linuxX32NetEpollWaitOneCode() []byte {
	e := &x64.Emitter{}
	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(16)
	e.Emit(0x41, 0x89, 0xF2) // mov r10d, esi
	e.Emit(0x48, 0x89, 0xE6) // mov rsi, rsp
	e.MovEdxImm32(1)
	e.MovEaxImm32(linuxX32SysEpollWait)
	e.Syscall()
	e.TestEaxEax()
	nonNegativeAt := e.JgeRel32()
	e.Leave()
	e.Ret()

	if err := x64.PatchRel32(e.Buf, nonNegativeAt, len(e.Buf)); err != nil {
		panic(err)
	}
	e.TestEaxEax()
	readyAt := e.JnzRel32()
	e.Leave()
	e.Ret()

	if err := x64.PatchRel32(e.Buf, readyAt, len(e.Buf)); err != nil {
		panic(err)
	}
	e.MovEaxFromRspDisp(4)
	e.Leave()
	e.Ret()
	return e.Buf
}

func linuxX32NetEpollWaitOneIntoCode() []byte {
	e := &x64.Emitter{}
	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(32)
	e.Emit(0x83, 0xFA, 0x02) // cmp edx, 2
	lenOKAt := e.JgeRel32()
	e.MovEaxImm32(0xffffffff)
	e.Leave()
	e.Ret()

	if err := x64.PatchRel32(e.Buf, lenOKAt, len(e.Buf)); err != nil {
		panic(err)
	}
	e.Emit(0x48, 0x89, 0x74, 0x24, 0x10) // mov [rsp+16], rsi
	e.Emit(0x41, 0x89, 0xCA)             // mov r10d, ecx
	e.Emit(0x48, 0x89, 0xE6)             // mov rsi, rsp
	e.MovEdxImm32(1)
	e.MovEaxImm32(linuxX32SysEpollWait)
	e.Syscall()
	e.TestEaxEax()
	nonNegativeAt := e.JgeRel32()
	e.Leave()
	e.Ret()

	if err := x64.PatchRel32(e.Buf, nonNegativeAt, len(e.Buf)); err != nil {
		panic(err)
	}
	e.TestEaxEax()
	readyAt := e.JnzRel32()
	e.Leave()
	e.Ret()

	if err := x64.PatchRel32(e.Buf, readyAt, len(e.Buf)); err != nil {
		panic(err)
	}
	e.Emit(0x48, 0x8B, 0x54, 0x24, 0x10) // mov rdx, [rsp+16]
	e.MovEaxFromRspDisp(4)
	e.Emit(0x89, 0x02) // mov [rdx], eax
	e.MovEaxFromRspDisp(0)
	e.Emit(0x89, 0x42, 0x04) // mov [rdx+4], eax
	e.MovEaxImm32(1)
	e.Leave()
	e.Ret()
	return e.Buf
}

func linuxX32NetSetNonblockingCode() []byte {
	e := &x64.Emitter{}
	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(16)
	e.Emit(0x89, 0x3C, 0x24) // mov [rsp], edi
	e.Emit(0xBE, byte(linuxX32FGetFL), 0, 0, 0)
	e.Emit(0x31, 0xD2)
	e.MovEaxImm32(linuxX32SysFcntl)
	e.Syscall()
	e.TestEaxEax()
	okAt := e.JgeRel32()
	e.Leave()
	e.Ret()

	if err := x64.PatchRel32(e.Buf, okAt, len(e.Buf)); err != nil {
		panic(err)
	}
	e.Emit(
		0x0D,
		byte(linuxX32ONonblock&0xff),
		byte((linuxX32ONonblock>>8)&0xff),
		byte((linuxX32ONonblock>>16)&0xff),
		byte((linuxX32ONonblock>>24)&0xff),
	)
	e.Emit(0x89, 0xC2)       // mov edx, eax
	e.Emit(0x8B, 0x3C, 0x24) // mov edi, [rsp]
	e.Emit(0xBE, byte(linuxX32FSetFL), 0, 0, 0)
	e.MovEaxImm32(linuxX32SysFcntl)
	e.Syscall()
	e.Leave()
	e.Ret()
	return e.Buf
}

func linuxX32NetSetSockOptCode(level uint32, optname uint32) []byte {
	e := &x64.Emitter{}
	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(16)
	linuxX32EmitMovMem32RspDispImm32(e, 0, 1)
	e.Emit(
		0xBE,
		byte(level&0xff),
		byte((level>>8)&0xff),
		byte((level>>16)&0xff),
		byte((level>>24)&0xff),
	)
	e.MovEdxImm32(optname)
	e.Emit(0x49, 0x89, 0xE2) // mov r10, rsp
	e.MovR8dImm32(4)
	e.MovR9dImm32(0)
	e.MovEaxImm32(linuxX32SysSetSockOpt)
	e.Syscall()
	e.Leave()
	e.Ret()
	return e.Buf
}

func linuxX32NetCloseCode() []byte {
	e := &x64.Emitter{}
	e.MovEaxImm32(linuxX32SysClose)
	e.Syscall()
	e.Ret()
	return e.Buf
}

func linuxX32EmitMovMem16RspDispImm16(e *x64.Emitter, disp byte, val uint16) {
	e.Emit(0x66, 0xC7, 0x44, 0x24, disp)
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], val)
	e.Emit(buf[:]...)
}

func linuxX32EmitMovMem32RspDispImm32(e *x64.Emitter, disp byte, val uint32) {
	e.Emit(0xC7, 0x44, 0x24, disp)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], val)
	e.Emit(buf[:]...)
}
