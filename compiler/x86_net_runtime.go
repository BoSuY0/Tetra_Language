package compiler

import "fmt"

var linuxX86BasicNetRuntimeSymbols = []struct {
	name        string
	code        func() []byte
	paramSlots  int
	returnSlots int
}{
	{name: "__tetra_net_socket_tcp4", code: linuxX86NetSocketTCP4Code, paramSlots: 1, returnSlots: 1},
	{name: "__tetra_net_bind_tcp4_loopback", code: func() []byte { return linuxX86NetTCP4LoopbackCode(linuxX86SocketOpBind) }, paramSlots: 3, returnSlots: 1},
	{name: "__tetra_net_connect_tcp4_loopback", code: func() []byte { return linuxX86NetTCP4LoopbackCode(linuxX86SocketOpConnect) }, paramSlots: 3, returnSlots: 1},
	{name: "__tetra_net_listen", code: linuxX86NetListenCode, paramSlots: 3, returnSlots: 1},
	{name: "__tetra_net_accept4", code: linuxX86NetAccept4Code, paramSlots: 3, returnSlots: 1},
	{name: "__tetra_net_read", code: func() []byte { return linuxX86NetReadWriteCode(linuxX86SysRead) }, paramSlots: 6, returnSlots: 1},
	{name: "__tetra_net_recv", code: func() []byte { return linuxX86NetRecvSendCode(linuxX86SocketOpRecv) }, paramSlots: 6, returnSlots: 1},
	{name: "__tetra_net_write", code: func() []byte { return linuxX86NetReadWriteCode(linuxX86SysWrite) }, paramSlots: 6, returnSlots: 1},
	{name: "__tetra_net_send", code: func() []byte { return linuxX86NetRecvSendCode(linuxX86SocketOpSend) }, paramSlots: 6, returnSlots: 1},
	{name: "__tetra_net_epoll_create", code: linuxX86NetEpollCreateCode, paramSlots: 1, returnSlots: 1},
	{name: "__tetra_net_epoll_ctl_add_read", code: func() []byte { return linuxX86NetEpollCtlCode(linuxX86EpollCtlAdd, linuxX86EpollIn) }, paramSlots: 3, returnSlots: 1},
	{name: "__tetra_net_epoll_ctl_add_read_write", code: func() []byte { return linuxX86NetEpollCtlCode(linuxX86EpollCtlAdd, linuxX86EpollIn|linuxX86EpollOut) }, paramSlots: 3, returnSlots: 1},
	{name: "__tetra_net_epoll_ctl_mod_read", code: func() []byte { return linuxX86NetEpollCtlCode(linuxX86EpollCtlMod, linuxX86EpollIn) }, paramSlots: 3, returnSlots: 1},
	{name: "__tetra_net_epoll_ctl_mod_read_write", code: func() []byte { return linuxX86NetEpollCtlCode(linuxX86EpollCtlMod, linuxX86EpollIn|linuxX86EpollOut) }, paramSlots: 3, returnSlots: 1},
	{name: "__tetra_net_epoll_ctl_delete", code: func() []byte { return linuxX86NetEpollCtlCode(linuxX86EpollCtlDel, 0) }, paramSlots: 3, returnSlots: 1},
	{name: "__tetra_net_epoll_wait_one", code: linuxX86NetEpollWaitOneCode, paramSlots: 3, returnSlots: 1},
	{name: "__tetra_net_epoll_wait_one_into", code: linuxX86NetEpollWaitOneIntoCode, paramSlots: 5, returnSlots: 1},
	{name: "__tetra_net_set_nonblocking", code: linuxX86NetSetNonblockingCode, paramSlots: 2, returnSlots: 1},
	{name: "__tetra_net_set_reuseport", code: func() []byte { return linuxX86NetSetSockOptCode(linuxX86SolSocket, linuxX86SoReusePort) }, paramSlots: 2, returnSlots: 1},
	{name: "__tetra_net_set_tcp_nodelay", code: func() []byte { return linuxX86NetSetSockOptCode(linuxX86IPProtoTCP, linuxX86TCPNoDelay) }, paramSlots: 2, returnSlots: 1},
	{name: "__tetra_net_close", code: linuxX86NetCloseCode, paramSlots: 2, returnSlots: 1},
}

const (
	linuxX86SysRead         = 3
	linuxX86SysWrite        = 4
	linuxX86SysClose        = 6
	linuxX86SysFcntl        = 55
	linuxX86SysSocketcall   = 102
	linuxX86SysEpollCtl     = 255
	linuxX86SysEpollWait    = 256
	linuxX86SysEpollCreate1 = 329

	linuxX86SocketOpSocket     = 1
	linuxX86SocketOpBind       = 2
	linuxX86SocketOpConnect    = 3
	linuxX86SocketOpListen     = 4
	linuxX86SocketOpSend       = 9
	linuxX86SocketOpRecv       = 10
	linuxX86SocketOpSetSockOpt = 14
	linuxX86SocketOpAccept4    = 18

	linuxX86AFInet       = 2
	linuxX86SockStream   = 1
	linuxX86SolSocket    = 1
	linuxX86SoReusePort  = 15
	linuxX86IPProtoTCP   = 6
	linuxX86TCPNoDelay   = 1
	linuxX86FGetFL       = 3
	linuxX86FSetFL       = 4
	linuxX86ONonblock    = 0x800
	linuxX86MaxTCPPort   = 65535
	linuxX86LoopbackAddr = 0x0100007f
	linuxX86EpollCtlAdd  = 1
	linuxX86EpollCtlDel  = 2
	linuxX86EpollCtlMod  = 3
	linuxX86EpollIn      = 1
	linuxX86EpollOut     = 4
)

func buildLinuxX86BasicNetRuntimeObject() *Object {
	code := make([]byte, 0)
	symbols := make([]Symbol, 0, len(linuxX86BasicNetRuntimeSymbols))
	for _, spec := range linuxX86BasicNetRuntimeSymbols {
		offset := len(code)
		code = append(code, spec.code()...)
		symbols = append(symbols, Symbol{
			Name:         spec.name,
			Offset:       uint32(offset),
			HasSignature: true,
			ParamSlots:   spec.paramSlots,
			ReturnSlots:  spec.returnSlots,
		})
	}
	return &Object{
		Target:  "linux-x86",
		Module:  "__linux_x86_netrt",
		Code:    code,
		Symbols: symbols,
	}
}

func appendLinuxX86BasicNetRuntimeObject(rt *Object) error {
	if rt == nil {
		return fmt.Errorf("missing linux-x86 runtime object")
	}
	existing := make(map[string]struct{}, len(rt.Symbols))
	for _, sym := range rt.Symbols {
		existing[sym.Name] = struct{}{}
	}
	for _, spec := range linuxX86BasicNetRuntimeSymbols {
		if _, ok := existing[spec.name]; ok {
			continue
		}
		offset := len(rt.Code)
		rt.Code = append(rt.Code, spec.code()...)
		rt.Symbols = append(rt.Symbols, Symbol{
			Name:         spec.name,
			Offset:       uint32(offset),
			HasSignature: true,
			ParamSlots:   spec.paramSlots,
			ReturnSlots:  spec.returnSlots,
		})
	}
	return nil
}

func linuxX86NetSocketTCP4Code() []byte {
	e := &linuxX86RuntimeEmitter{}
	e.emit(0x55)       // push ebp
	e.emit(0x89, 0xe5) // mov ebp, esp
	e.emit(0x53)       // push ebx
	e.emit(0x83, 0xec, 0x0c)
	e.emit(0xc7, 0x04, 0x24) // mov dword ptr [esp], AF_INET
	e.imm32(linuxX86AFInet)
	e.emit(0xc7, 0x44, 0x24, 0x04) // mov dword ptr [esp+4], SOCK_STREAM
	e.imm32(linuxX86SockStream)
	e.emit(0xc7, 0x44, 0x24, 0x08) // mov dword ptr [esp+8], protocol=0
	e.imm32(0)
	e.emit(0xb8)
	e.imm32(linuxX86SysSocketcall) // mov eax, SYS_socketcall
	e.emit(0xbb)
	e.imm32(linuxX86SocketOpSocket) // mov ebx, SYS_SOCKET
	e.emit(0x89, 0xe1)              // mov ecx, esp
	e.emit(0xcd, 0x80)              // int 0x80
	e.emit(0x83, 0xc4, 0x0c)        // add esp, 12
	e.emit(0x5b)                    // pop ebx
	e.emit(0x5d, 0xc3)              // pop ebp; ret
	return e.buf
}

func linuxX86NetTCP4LoopbackCode(socketOp uint32) []byte {
	e := &linuxX86RuntimeEmitter{}
	var failJumps []int
	e.emit(0x55)       // push ebp
	e.emit(0x89, 0xe5) // mov ebp, esp
	e.emit(0x53)       // push ebx
	e.emit(0x83, 0xec, 0x1c)

	e.emit(0x8b, 0x45, 0x0c) // mov eax, [ebp+12] port
	e.emit(0x85, 0xc0)       // test eax, eax
	portNonNegative := e.jccRel32(0x0f, 0x8d)
	failJumps = append(failJumps, e.jmpRel32())
	if err := e.patchRel32(portNonNegative, len(e.buf)); err != nil {
		panic(err)
	}
	e.emit(0x3d) // cmp eax, 65535
	e.imm32(linuxX86MaxTCPPort)
	failJumps = append(failJumps, e.jccRel32(0x0f, 0x87)) // ja fail

	e.emit(0x66, 0xc7, 0x04, 0x24, 0x02, 0x00) // mov word ptr [esp], AF_INET
	e.emit(0x66, 0x8b, 0x45, 0x0c)             // mov ax, [ebp+12] port
	e.emit(0x86, 0xe0)                         // xchg al, ah
	e.emit(0x66, 0x89, 0x44, 0x24, 0x02)       // mov [esp+2], ax
	e.emit(0xc7, 0x44, 0x24, 0x04)             // mov dword ptr [esp+4], 127.0.0.1
	e.imm32(linuxX86LoopbackAddr)
	e.emit(0xc7, 0x44, 0x24, 0x08) // mov dword ptr [esp+8], 0
	e.imm32(0)
	e.emit(0xc7, 0x44, 0x24, 0x0c) // mov dword ptr [esp+12], 0
	e.imm32(0)

	e.emit(0x8b, 0x45, 0x08)       // mov eax, [ebp+8] fd
	e.emit(0x89, 0x44, 0x24, 0x10) // mov [esp+16], eax
	e.emit(0x8d, 0x04, 0x24)       // lea eax, [esp]
	e.emit(0x89, 0x44, 0x24, 0x14) // mov [esp+20], eax
	e.emit(0xc7, 0x44, 0x24, 0x18) // mov dword ptr [esp+24], sizeof(sockaddr_in)
	e.imm32(16)
	e.emit(0xb8)
	e.imm32(linuxX86SysSocketcall)
	e.emit(0xbb)
	e.imm32(socketOp)
	e.emit(0x8d, 0x4c, 0x24, 0x10) // lea ecx, [esp+16]
	e.emit(0xcd, 0x80)             // int 0x80
	doneAt := e.jmpRel32()

	failTo := len(e.buf)
	for _, at := range failJumps {
		if err := e.patchRel32(at, failTo); err != nil {
			panic(err)
		}
	}
	e.emit(0xb8)
	e.imm32(0xffffffff) // mov eax, -1

	doneTo := len(e.buf)
	if err := e.patchRel32(doneAt, doneTo); err != nil {
		panic(err)
	}
	e.emit(0x83, 0xc4, 0x1c) // add esp, 28
	e.emit(0x5b)             // pop ebx
	e.emit(0x5d, 0xc3)       // pop ebp; ret
	return e.buf
}

func linuxX86NetListenCode() []byte {
	e := &linuxX86RuntimeEmitter{}
	e.emit(0x55)       // push ebp
	e.emit(0x89, 0xe5) // mov ebp, esp
	e.emit(0x53)       // push ebx
	e.emit(0x83, 0xec, 0x08)
	e.emit(0x8b, 0x45, 0x08)       // mov eax, [ebp+8] fd
	e.emit(0x89, 0x04, 0x24)       // mov [esp], eax
	e.emit(0x8b, 0x45, 0x0c)       // mov eax, [ebp+12] backlog
	e.emit(0x89, 0x44, 0x24, 0x04) // mov [esp+4], eax
	e.emit(0xb8)
	e.imm32(linuxX86SysSocketcall)
	e.emit(0xbb)
	e.imm32(linuxX86SocketOpListen)
	e.emit(0x89, 0xe1)       // mov ecx, esp
	e.emit(0xcd, 0x80)       // int 0x80
	e.emit(0x83, 0xc4, 0x08) // add esp, 8
	e.emit(0x5b)             // pop ebx
	e.emit(0x5d, 0xc3)       // pop ebp; ret
	return e.buf
}

func linuxX86NetAccept4Code() []byte {
	e := &linuxX86RuntimeEmitter{}
	e.emit(0x55)       // push ebp
	e.emit(0x89, 0xe5) // mov ebp, esp
	e.emit(0x53)       // push ebx
	e.emit(0x83, 0xec, 0x10)
	e.emit(0x8b, 0x45, 0x08)       // mov eax, [ebp+8] fd
	e.emit(0x89, 0x04, 0x24)       // mov [esp], eax
	e.emit(0xc7, 0x44, 0x24, 0x04) // mov dword ptr [esp+4], NULL addr
	e.imm32(0)
	e.emit(0xc7, 0x44, 0x24, 0x08) // mov dword ptr [esp+8], NULL addrlen
	e.imm32(0)
	e.emit(0x8b, 0x45, 0x0c)       // mov eax, [ebp+12] flags
	e.emit(0x89, 0x44, 0x24, 0x0c) // mov [esp+12], eax
	e.emit(0xb8)
	e.imm32(linuxX86SysSocketcall)
	e.emit(0xbb)
	e.imm32(linuxX86SocketOpAccept4)
	e.emit(0x89, 0xe1)       // mov ecx, esp
	e.emit(0xcd, 0x80)       // int 0x80
	e.emit(0x83, 0xc4, 0x10) // add esp, 16
	e.emit(0x5b)             // pop ebx
	e.emit(0x5d, 0xc3)       // pop ebp; ret
	return e.buf
}

func linuxX86NetReadWriteCode(syscall uint32) []byte {
	e := &linuxX86RuntimeEmitter{}
	failJumps := linuxX86EmitNetSliceBoundsCheck(e)
	e.emit(0x8b, 0x5d, 0x08) // mov ebx, [ebp+8] fd
	e.emit(0x8b, 0x4d, 0x0c) // mov ecx, [ebp+12] slice ptr
	e.emit(0x03, 0x4d, 0x14) // add ecx, [ebp+20] start
	e.emit(0xb8)
	e.imm32(syscall)
	e.emit(0xcd, 0x80) // int 0x80
	doneAt := e.jmpRel32()
	linuxX86EmitNetSliceFailure(e, failJumps)
	if err := e.patchRel32(doneAt, len(e.buf)); err != nil {
		panic(err)
	}
	e.emit(0x5b)       // pop ebx
	e.emit(0x5d, 0xc3) // pop ebp; ret
	return e.buf
}

func linuxX86NetRecvSendCode(socketOp uint32) []byte {
	e := &linuxX86RuntimeEmitter{}
	failJumps := linuxX86EmitNetSliceBoundsCheck(e)
	e.emit(0x83, 0xec, 0x10)
	e.emit(0x8b, 0x45, 0x08) // mov eax, [ebp+8] fd
	e.emit(0x89, 0x04, 0x24) // mov [esp], eax
	e.emit(0x8b, 0x4d, 0x0c) // mov ecx, [ebp+12] slice ptr
	e.emit(0x03, 0x4d, 0x14) // add ecx, [ebp+20] start
	e.emit(0x89, 0x4c, 0x24, 0x04)
	e.emit(0x89, 0x54, 0x24, 0x08)
	e.emit(0xc7, 0x44, 0x24, 0x0c) // flags=0
	e.imm32(0)
	e.emit(0xb8)
	e.imm32(linuxX86SysSocketcall)
	e.emit(0xbb)
	e.imm32(socketOp)
	e.emit(0x89, 0xe1)       // mov ecx, esp
	e.emit(0xcd, 0x80)       // int 0x80
	e.emit(0x83, 0xc4, 0x10) // add esp, 16
	doneAt := e.jmpRel32()
	linuxX86EmitNetSliceFailure(e, failJumps)
	if err := e.patchRel32(doneAt, len(e.buf)); err != nil {
		panic(err)
	}
	e.emit(0x5b)       // pop ebx
	e.emit(0x5d, 0xc3) // pop ebp; ret
	return e.buf
}

func linuxX86EmitNetSliceBoundsCheck(e *linuxX86RuntimeEmitter) []int {
	var failJumps []int
	e.emit(0x55)             // push ebp
	e.emit(0x89, 0xe5)       // mov ebp, esp
	e.emit(0x53)             // push ebx
	e.emit(0x8b, 0x4d, 0x14) // mov ecx, [ebp+20] start
	e.emit(0x85, 0xc9)       // test ecx, ecx
	startOK := e.jccRel32(0x0f, 0x8d)
	failJumps = append(failJumps, e.jmpRel32())
	if err := e.patchRel32(startOK, len(e.buf)); err != nil {
		panic(err)
	}
	e.emit(0x8b, 0x55, 0x18) // mov edx, [ebp+24] count
	e.emit(0x85, 0xd2)       // test edx, edx
	countOK := e.jccRel32(0x0f, 0x8d)
	failJumps = append(failJumps, e.jmpRel32())
	if err := e.patchRel32(countOK, len(e.buf)); err != nil {
		panic(err)
	}
	e.emit(0x8b, 0x45, 0x10) // mov eax, [ebp+16] slice len
	e.emit(0x39, 0xc8)       // cmp eax, ecx
	startInRange := e.jccRel32(0x0f, 0x8d)
	failJumps = append(failJumps, e.jmpRel32())
	if err := e.patchRel32(startInRange, len(e.buf)); err != nil {
		panic(err)
	}
	e.emit(0x29, 0xc8) // sub eax, ecx
	e.emit(0x39, 0xd0) // cmp eax, edx
	useRequested := e.jccRel32(0x0f, 0x8d)
	e.emit(0x89, 0xc2) // mov edx, eax
	if err := e.patchRel32(useRequested, len(e.buf)); err != nil {
		panic(err)
	}
	return failJumps
}

func linuxX86EmitNetSliceFailure(e *linuxX86RuntimeEmitter, failJumps []int) {
	failTo := len(e.buf)
	for _, at := range failJumps {
		if err := e.patchRel32(at, failTo); err != nil {
			panic(err)
		}
	}
	e.emit(0xb8)
	e.imm32(0xffffffff) // mov eax, -1
}

func linuxX86NetEpollCreateCode() []byte {
	e := &linuxX86RuntimeEmitter{}
	e.emit(0x55)       // push ebp
	e.emit(0x89, 0xe5) // mov ebp, esp
	e.emit(0x53)       // push ebx
	e.emit(0x31, 0xdb) // xor ebx, ebx
	e.emit(0xb8)
	e.imm32(linuxX86SysEpollCreate1)
	e.emit(0xcd, 0x80) // int 0x80
	e.emit(0x5b)       // pop ebx
	e.emit(0x5d, 0xc3) // pop ebp; ret
	return e.buf
}

func linuxX86NetEpollCtlCode(op uint32, events uint32) []byte {
	e := &linuxX86RuntimeEmitter{}
	e.emit(0x55)       // push ebp
	e.emit(0x89, 0xe5) // mov ebp, esp
	e.emit(0x53)       // push ebx
	e.emit(0x56)       // push esi
	e.emit(0x83, 0xec, 0x10)
	e.emit(0xc7, 0x04, 0x24)
	e.imm32(events)                // event.events
	e.emit(0x8b, 0x45, 0x0c)       // mov eax, [ebp+12] fd
	e.emit(0x89, 0x44, 0x24, 0x04) // mov [esp+4], eax
	e.emit(0xc7, 0x44, 0x24, 0x08) // zero high event.data
	e.imm32(0)
	e.emit(0xc7, 0x44, 0x24, 0x0c)
	e.imm32(0)
	e.emit(0xb8)
	e.imm32(linuxX86SysEpollCtl)
	e.emit(0x8b, 0x5d, 0x08) // mov ebx, [ebp+8] epfd
	e.emit(0xb9)
	e.imm32(op)              // mov ecx, op
	e.emit(0x8b, 0x55, 0x0c) // mov edx, [ebp+12] fd
	e.emit(0x8d, 0x34, 0x24) // lea esi, [esp]
	e.emit(0xcd, 0x80)       // int 0x80
	e.emit(0x83, 0xc4, 0x10) // add esp, 16
	e.emit(0x5e)             // pop esi
	e.emit(0x5b)             // pop ebx
	e.emit(0x5d, 0xc3)       // pop ebp; ret
	return e.buf
}

func linuxX86NetEpollWaitOneCode() []byte {
	e := &linuxX86RuntimeEmitter{}
	emitCleanup := func() {
		e.emit(0x83, 0xc4, 0x10) // add esp, 16
		e.emit(0x5e)             // pop esi
		e.emit(0x5b)             // pop ebx
		e.emit(0x5d, 0xc3)       // pop ebp; ret
	}
	e.emit(0x55)       // push ebp
	e.emit(0x89, 0xe5) // mov ebp, esp
	e.emit(0x53)       // push ebx
	e.emit(0x56)       // push esi
	e.emit(0x83, 0xec, 0x10)
	e.emit(0xb8)
	e.imm32(linuxX86SysEpollWait)
	e.emit(0x8b, 0x5d, 0x08) // mov ebx, [ebp+8] epfd
	e.emit(0x8d, 0x0c, 0x24) // lea ecx, [esp]
	e.emit(0xba)
	e.imm32(1)               // mov edx, 1
	e.emit(0x8b, 0x75, 0x0c) // mov esi, [ebp+12] timeout_ms
	e.emit(0xcd, 0x80)       // int 0x80
	e.emit(0x85, 0xc0)       // test eax, eax
	nonNegativeAt := e.jccRel32(0x0f, 0x8d)
	emitCleanup()

	if err := e.patchRel32(nonNegativeAt, len(e.buf)); err != nil {
		panic(err)
	}
	e.emit(0x85, 0xc0) // test eax, eax
	readyAt := e.jccRel32(0x0f, 0x85)
	emitCleanup()

	if err := e.patchRel32(readyAt, len(e.buf)); err != nil {
		panic(err)
	}
	e.emit(0x8b, 0x44, 0x24, 0x04) // mov eax, [esp+4]
	emitCleanup()
	return e.buf
}

func linuxX86NetEpollWaitOneIntoCode() []byte {
	e := &linuxX86RuntimeEmitter{}
	emitCleanup := func() {
		e.emit(0x83, 0xc4, 0x20) // add esp, 32
		e.emit(0x5e)             // pop esi
		e.emit(0x5b)             // pop ebx
		e.emit(0x5d, 0xc3)       // pop ebp; ret
	}
	e.emit(0x55)       // push ebp
	e.emit(0x89, 0xe5) // mov ebp, esp
	e.emit(0x53)       // push ebx
	e.emit(0x56)       // push esi
	e.emit(0x83, 0xec, 0x20)
	e.emit(0x83, 0x7d, 0x10, 0x02) // cmp dword ptr [ebp+16], 2
	lenOKAt := e.jccRel32(0x0f, 0x8d)
	e.emit(0xb8)
	e.imm32(0xffffffff)
	emitCleanup()

	if err := e.patchRel32(lenOKAt, len(e.buf)); err != nil {
		panic(err)
	}
	e.emit(0x8b, 0x45, 0x0c)       // mov eax, [ebp+12] out ptr
	e.emit(0x89, 0x44, 0x24, 0x10) // mov [esp+16], eax
	e.emit(0xb8)
	e.imm32(linuxX86SysEpollWait)
	e.emit(0x8b, 0x5d, 0x08) // mov ebx, [ebp+8] epfd
	e.emit(0x8d, 0x0c, 0x24) // lea ecx, [esp]
	e.emit(0xba)
	e.imm32(1)               // mov edx, 1
	e.emit(0x8b, 0x75, 0x14) // mov esi, [ebp+20] timeout_ms
	e.emit(0xcd, 0x80)       // int 0x80
	e.emit(0x85, 0xc0)       // test eax, eax
	nonNegativeAt := e.jccRel32(0x0f, 0x8d)
	emitCleanup()

	if err := e.patchRel32(nonNegativeAt, len(e.buf)); err != nil {
		panic(err)
	}
	e.emit(0x85, 0xc0) // test eax, eax
	readyAt := e.jccRel32(0x0f, 0x85)
	emitCleanup()

	if err := e.patchRel32(readyAt, len(e.buf)); err != nil {
		panic(err)
	}
	e.emit(0x8b, 0x54, 0x24, 0x10) // mov edx, [esp+16]
	e.emit(0x8b, 0x44, 0x24, 0x04) // mov eax, [esp+4]
	e.emit(0x89, 0x02)             // mov [edx], eax
	e.emit(0x8b, 0x04, 0x24)       // mov eax, [esp]
	e.emit(0x89, 0x42, 0x04)       // mov [edx+4], eax
	e.emit(0xb8)
	e.imm32(1)
	emitCleanup()
	return e.buf
}

func linuxX86NetSetNonblockingCode() []byte {
	e := &linuxX86RuntimeEmitter{}
	e.emit(0x55)       // push ebp
	e.emit(0x89, 0xe5) // mov ebp, esp
	e.emit(0x53)       // push ebx
	e.emit(0xb8)
	e.imm32(linuxX86SysFcntl) // mov eax, SYS_fcntl
	e.emit(0x8b, 0x5d, 0x08)  // mov ebx, [ebp+8] fd
	e.emit(0xb9)
	e.imm32(linuxX86FGetFL) // mov ecx, F_GETFL
	e.emit(0x31, 0xd2)      // xor edx, edx
	e.emit(0xcd, 0x80)      // int 0x80
	e.emit(0x85, 0xc0)      // test eax, eax
	failAt := e.jccRel32(0x0f, 0x88)
	e.emit(0x0d)
	e.imm32(linuxX86ONonblock) // or eax, O_NONBLOCK
	e.emit(0x89, 0xc2)         // mov edx, eax
	e.emit(0xb8)
	e.imm32(linuxX86SysFcntl) // mov eax, SYS_fcntl
	e.emit(0x8b, 0x5d, 0x08)  // mov ebx, [ebp+8] fd
	e.emit(0xb9)
	e.imm32(linuxX86FSetFL) // mov ecx, F_SETFL
	e.emit(0xcd, 0x80)      // int 0x80
	done := len(e.buf)
	if err := e.patchRel32(failAt, done); err != nil {
		panic(err)
	}
	e.emit(0x5b)       // pop ebx
	e.emit(0x5d, 0xc3) // pop ebp; ret
	return e.buf
}

func linuxX86NetSetSockOptCode(level uint32, optname uint32) []byte {
	e := &linuxX86RuntimeEmitter{}
	e.emit(0x55)       // push ebp
	e.emit(0x89, 0xe5) // mov ebp, esp
	e.emit(0x53)       // push ebx
	e.emit(0x83, 0xec, 0x18)
	e.emit(0xc7, 0x04, 0x24)
	e.imm32(1) // optval = 1
	e.emit(0x8b, 0x45, 0x08)
	e.emit(0x89, 0x44, 0x24, 0x04) // fd
	e.emit(0xc7, 0x44, 0x24, 0x08)
	e.imm32(level)
	e.emit(0xc7, 0x44, 0x24, 0x0c)
	e.imm32(optname)
	e.emit(0x8d, 0x04, 0x24)
	e.emit(0x89, 0x44, 0x24, 0x10) // optval pointer
	e.emit(0xc7, 0x44, 0x24, 0x14)
	e.imm32(4) // optlen
	e.emit(0xb8)
	e.imm32(linuxX86SysSocketcall)
	e.emit(0xbb)
	e.imm32(linuxX86SocketOpSetSockOpt)
	e.emit(0x8d, 0x4c, 0x24, 0x04) // lea ecx, [esp+4]
	e.emit(0xcd, 0x80)             // int 0x80
	e.emit(0x83, 0xc4, 0x18)       // add esp, 24
	e.emit(0x5b)                   // pop ebx
	e.emit(0x5d, 0xc3)             // pop ebp; ret
	return e.buf
}

func linuxX86NetCloseCode() []byte {
	e := &linuxX86RuntimeEmitter{}
	e.emit(0x55)       // push ebp
	e.emit(0x89, 0xe5) // mov ebp, esp
	e.emit(0x53)       // push ebx
	e.emit(0xb8)
	e.imm32(linuxX86SysClose) // mov eax, SYS_close
	e.emit(0x8b, 0x5d, 0x08)  // mov ebx, [ebp+8] fd
	e.emit(0xcd, 0x80)        // int 0x80
	e.emit(0x5b)              // pop ebx
	e.emit(0x5d, 0xc3)        // pop ebp; ret
	return e.buf
}
