package actorsrt

import "tetra_language/compiler/internal/backend/x64"

func emitFilesystemExists(e *x64.Emitter) error {
	const (
		linuxSysAccess = 21
		maxPathLen     = 4095
		pathBufSize    = 4096
	)
	var failJumps []int

	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(pathBufSize)

	// Arguments: rdi=path_ptr, rsi=path_len, rdx=cap.io token.
	e.Emit(0x48, 0x85, 0xff) // test rdi, rdi
	failJumps = append(failJumps, e.JzRel32())
	e.Emit(0x85, 0xf6) // test esi, esi
	failJumps = append(failJumps, e.JzRel32())
	e.Emit(0x81, 0xfe, 0xff, 0x0f, 0x00, 0x00) // cmp esi, 4095
	failJumps = append(failJumps, e.JaRel32())

	e.Emit(0x48, 0x89, 0xf1)       // mov rcx, rsi
	e.Emit(0x49, 0x89, 0xf8)       // mov r8, rdi
	e.Emit(0x4c, 0x8d, 0x0c, 0x24) // lea r9, [rsp]
	e.XorEaxEax()                  // rax = copy index

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
		return err
	}

	copiedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, copiedAt, copiedTo); err != nil {
		return err
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
			return err
		}
	}
	e.XorEaxEax()
	e.Leave()
	e.Ret()
	return nil
}

func emitNetSocketTCP4(e *x64.Emitter) error {
	// Arguments: rdi=cap.io token (ignored).
	e.MovEdiImm32(2)            // AF_INET
	e.Emit(0xBE, 0x01, 0, 0, 0) // mov esi, SOCK_STREAM
	e.Emit(0x31, 0xD2)          // xor edx, edx
	e.MovR10dImm32(0)
	e.MovR8dImm32(0)
	e.MovR9dImm32(0)
	e.MovEaxImm32(linuxSysSocket)
	e.Syscall()
	e.Ret()
	return nil
}

func emitNetBindTCP4Loopback(e *x64.Emitter) error {
	// Arguments: rdi=fd, rsi=port, rdx=cap.io token (ignored).
	failJumps, err := emitNetRejectInvalidTCPPort(e)
	if err != nil {
		return err
	}
	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(32)
	e.Emit(0x89, 0x7C, 0x24, 0x10) // mov [rsp+16], edi

	emitMovMem16RspDispImm16(e, 0, 2)          // AF_INET
	e.MovEaxEsi()                              // port
	e.Emit(0x86, 0xE0)                         // xchg al, ah
	emitMovMem16RspDispAx(e, 2)                // sin_port
	emitMovMem32RspDispImm32(e, 4, 0x0100007f) // 127.0.0.1 bytes
	emitMovMem32RspDispImm32(e, 8, 0)          // sin_zero
	emitMovMem32RspDispImm32(e, 12, 0)         // sin_zero
	e.Emit(0x8B, 0x7C, 0x24, 0x10)             // mov edi, [rsp+16]
	e.Emit(0x48, 0x8D, 0x34, 0x24)             // lea rsi, [rsp]
	e.MovEdxImm32(16)                          // sizeof(sockaddr_in)
	e.MovEaxImm32(linuxSysBind)
	e.Syscall()
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.MovEaxImm32(0xFFFFFFFF)
	e.Ret()
	return nil
}

func emitNetConnectTCP4Loopback(e *x64.Emitter) error {
	// Arguments: rdi=fd, rsi=port, rdx=cap.io token (ignored).
	failJumps, err := emitNetRejectInvalidTCPPort(e)
	if err != nil {
		return err
	}
	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(32)
	e.Emit(0x89, 0x7C, 0x24, 0x10) // mov [rsp+16], edi

	emitMovMem16RspDispImm16(e, 0, 2)          // AF_INET
	e.MovEaxEsi()                              // port
	e.Emit(0x86, 0xE0)                         // xchg al, ah
	emitMovMem16RspDispAx(e, 2)                // sin_port
	emitMovMem32RspDispImm32(e, 4, 0x0100007f) // 127.0.0.1 bytes
	emitMovMem32RspDispImm32(e, 8, 0)          // sin_zero
	emitMovMem32RspDispImm32(e, 12, 0)         // sin_zero
	e.Emit(0x8B, 0x7C, 0x24, 0x10)             // mov edi, [rsp+16]
	e.Emit(0x48, 0x8D, 0x34, 0x24)             // lea rsi, [rsp]
	e.MovEdxImm32(16)                          // sizeof(sockaddr_in)
	e.MovEaxImm32(linuxSysConnect)
	e.Syscall()
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.MovEaxImm32(0xFFFFFFFF)
	e.Ret()
	return nil
}

func emitNetRejectInvalidTCPPort(e *x64.Emitter) ([]int, error) {
	var failJumps []int
	e.Emit(0x85, 0xF6) // test esi, esi
	nonNegativeAt := e.JgeRel32()
	failJumps = append(failJumps, e.JmpRel32())
	nonNegativeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nonNegativeAt, nonNegativeTo); err != nil {
		return nil, err
	}
	e.Emit(0x81, 0xFE, 0xFF, 0xFF, 0x00, 0x00) // cmp esi, 65535
	failJumps = append(failJumps, e.JaRel32())
	return failJumps, nil
}

func emitNetListen(e *x64.Emitter) error {
	// Arguments: rdi=fd, rsi=backlog, rdx=cap.io token (ignored).
	e.MovEaxImm32(linuxSysListen)
	e.Syscall()
	e.Ret()
	return nil
}

func emitNetAccept4(e *x64.Emitter) error {
	// Arguments: rdi=fd, rsi=flags, rdx=cap.io token (ignored).
	e.Emit(0x41, 0x89, 0xF2) // mov r10d, esi
	e.Emit(0x31, 0xF6)       // xor esi, esi (addr=NULL)
	e.Emit(0x31, 0xD2)       // xor edx, edx (addrlen=NULL)
	e.MovEaxImm32(linuxSysAccept4)
	e.Syscall()
	e.Ret()
	return nil
}

func emitNetRead(e *x64.Emitter) error {
	return emitNetReadWrite(e, linuxSysRead)
}

func emitNetRecv(e *x64.Emitter) error {
	return emitNetRecvSend(e, linuxSysRecvfrom)
}

func emitNetWrite(e *x64.Emitter) error {
	return emitNetReadWrite(e, linuxSysWrite)
}

func emitNetSend(e *x64.Emitter) error {
	return emitNetRecvSend(e, linuxSysSendto)
}

func emitNetReadWrite(e *x64.Emitter, syscall uint32) error {
	var failJumps []int

	// Arguments: rdi=fd, rsi=slice_ptr, rdx=slice_len, rcx=start, r8=count, r9=cap.io token (ignored).
	e.Emit(0x85, 0xC9) // test ecx, ecx
	startOK := e.JgeRel32()
	failJumps = append(failJumps, e.JmpRel32())
	startOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, startOK, startOKTo); err != nil {
		return err
	}
	e.Emit(0x45, 0x85, 0xC0) // test r8d, r8d
	countOK := e.JgeRel32()
	failJumps = append(failJumps, e.JmpRel32())
	countOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, countOK, countOKTo); err != nil {
		return err
	}
	e.Emit(0x39, 0xCA) // cmp edx, ecx
	startInRange := e.JgeRel32()
	failJumps = append(failJumps, e.JmpRel32())
	startInRangeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, startInRange, startInRangeTo); err != nil {
		return err
	}

	e.Emit(0x29, 0xCA)       // sub edx, ecx (available = len - start)
	e.Emit(0x44, 0x39, 0xC2) // cmp edx, r8d
	useRequestedCount := e.JgeRel32()
	e.Emit(0x41, 0x89, 0xD0) // mov r8d, edx
	useRequestedCountTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, useRequestedCount, useRequestedCountTo); err != nil {
		return err
	}
	e.Emit(0x48, 0x63, 0xC9) // movsxd rcx, ecx
	e.Emit(0x48, 0x01, 0xCE) // add rsi, rcx
	e.MovRdxR8()
	e.MovEaxImm32(syscall)
	e.Syscall()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.MovEaxImm32(0xFFFFFFFF)
	e.Ret()
	return nil
}

func emitNetRecvSend(e *x64.Emitter, syscall uint32) error {
	var failJumps []int

	// Arguments: rdi=fd, rsi=slice_ptr, rdx=slice_len, rcx=start, r8=count, r9=cap.io token (ignored).
	// Emits recvfrom/sendto with flags=0 and NULL address operands.
	e.Emit(0x85, 0xC9) // test ecx, ecx
	startOK := e.JgeRel32()
	failJumps = append(failJumps, e.JmpRel32())
	startOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, startOK, startOKTo); err != nil {
		return err
	}
	e.Emit(0x45, 0x85, 0xC0) // test r8d, r8d
	countOK := e.JgeRel32()
	failJumps = append(failJumps, e.JmpRel32())
	countOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, countOK, countOKTo); err != nil {
		return err
	}
	e.Emit(0x39, 0xCA) // cmp edx, ecx
	startInRange := e.JgeRel32()
	failJumps = append(failJumps, e.JmpRel32())
	startInRangeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, startInRange, startInRangeTo); err != nil {
		return err
	}

	e.Emit(0x29, 0xCA)       // sub edx, ecx (available = len - start)
	e.Emit(0x44, 0x39, 0xC2) // cmp edx, r8d
	useRequestedCount := e.JgeRel32()
	e.Emit(0x41, 0x89, 0xD0) // mov r8d, edx
	useRequestedCountTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, useRequestedCount, useRequestedCountTo); err != nil {
		return err
	}
	e.Emit(0x48, 0x63, 0xC9) // movsxd rcx, ecx
	e.Emit(0x48, 0x01, 0xCE) // add rsi, rcx
	e.MovRdxR8()
	e.MovR10dImm32(0) // flags=0
	e.MovR8dImm32(0)  // addr=NULL
	e.MovR9dImm32(0)  // addrlen=NULL
	e.MovEaxImm32(syscall)
	e.Syscall()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.MovEaxImm32(0xFFFFFFFF)
	e.Ret()
	return nil
}

func emitNetEpollCreate(e *x64.Emitter) error {
	// Arguments: rdi=cap.io token (ignored).
	e.MovEdiImm32(0) // flags=0
	e.MovEaxImm32(linuxSysEpollCreate1)
	e.Syscall()
	e.Ret()
	return nil
}

func emitNetEpollCtlAddRead(e *x64.Emitter) error {
	const (
		epollCtlAdd = 1
		epollIn     = 1
	)
	return emitNetEpollCtl(e, epollCtlAdd, epollIn)
}

func emitNetEpollCtlAddReadWrite(e *x64.Emitter) error {
	const (
		epollCtlAdd = 1
		epollIn     = 1
		epollOut    = 4
	)
	return emitNetEpollCtl(e, epollCtlAdd, epollIn|epollOut)
}

func emitNetEpollCtlModRead(e *x64.Emitter) error {
	const (
		epollCtlMod = 3
		epollIn     = 1
	)
	return emitNetEpollCtl(e, epollCtlMod, epollIn)
}

func emitNetEpollCtlModReadWrite(e *x64.Emitter) error {
	const (
		epollCtlMod = 3
		epollIn     = 1
		epollOut    = 4
	)
	return emitNetEpollCtl(e, epollCtlMod, epollIn|epollOut)
}

func emitNetEpollCtlDelete(e *x64.Emitter) error {
	const epollCtlDel = 2
	return emitNetEpollCtl(e, epollCtlDel, 0)
}

func emitNetEpollCtl(e *x64.Emitter, op uint32, events uint32) error {
	// Arguments: rdi=epfd, rsi=fd, rdx=cap.io token (ignored).
	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(16)
	emitMovMem32RspDispImm32(e, 0, events)
	e.Emit(0x48, 0x89, 0x74, 0x24, 0x04)                                                      // mov [rsp+4], rsi (event.data.u64)
	e.Emit(0x48, 0x89, 0xF2)                                                                  // mov rdx, rsi (fd)
	e.Emit(0xBE, byte(op&0xff), byte((op>>8)&0xff), byte((op>>16)&0xff), byte((op>>24)&0xff)) // mov esi, op
	e.Emit(0x49, 0x89, 0xE2)                                                                  // mov r10, rsp
	e.MovEaxImm32(linuxSysEpollCtl)
	e.Syscall()
	e.Leave()
	e.Ret()
	return nil
}

func emitNetEpollWaitOne(e *x64.Emitter) error {
	// Arguments: rdi=epfd, rsi=timeout_ms, rdx=cap.io token (ignored).
	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(16)
	e.Emit(0x41, 0x89, 0xF2) // mov r10d, esi
	e.Emit(0x48, 0x89, 0xE6) // mov rsi, rsp
	e.MovEdxImm32(1)         // maxevents=1
	e.MovEaxImm32(linuxSysEpollWait)
	e.Syscall()
	e.TestEaxEax()
	nonNegativeAt := e.JgeRel32()
	e.Leave()
	e.Ret()

	nonNegativeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nonNegativeAt, nonNegativeTo); err != nil {
		return err
	}
	e.TestEaxEax()
	readyAt := e.JnzRel32()
	e.Leave()
	e.Ret()

	readyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, readyAt, readyTo); err != nil {
		return err
	}
	e.Emit(0x8B, 0x44, 0x24, 0x04) // mov eax, [rsp+4] (event.data lower i32)
	e.Leave()
	e.Ret()
	return nil
}

func emitNetEpollWaitOneInto(e *x64.Emitter) error {
	// Arguments: rdi=epfd, rsi=[]i32 ptr, rdx=[]i32 len, rcx=timeout_ms, r8=cap.io token (ignored).
	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(32)
	e.Emit(0x83, 0xFA, 0x02) // cmp edx, 2
	lenOKAt := e.JgeRel32()
	e.MovEaxImm32(0xFFFFFFFF)
	e.Leave()
	e.Ret()

	lenOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, lenOKAt, lenOKTo); err != nil {
		return err
	}
	e.Emit(0x48, 0x89, 0x74, 0x24, 0x10) // mov [rsp+16], rsi (out ptr)
	e.Emit(0x41, 0x89, 0xCA)             // mov r10d, ecx (timeout_ms)
	e.Emit(0x48, 0x89, 0xE6)             // mov rsi, rsp (events)
	e.MovEdxImm32(1)                     // maxevents=1
	e.MovEaxImm32(linuxSysEpollWait)
	e.Syscall()
	e.TestEaxEax()
	nonNegativeAt := e.JgeRel32()
	e.Leave()
	e.Ret()

	nonNegativeTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nonNegativeAt, nonNegativeTo); err != nil {
		return err
	}
	e.TestEaxEax()
	readyAt := e.JnzRel32()
	e.Leave()
	e.Ret()

	readyTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, readyAt, readyTo); err != nil {
		return err
	}
	e.Emit(0x48, 0x8B, 0x54, 0x24, 0x10) // mov rdx, [rsp+16] (out ptr)
	e.MovEaxFromRspDisp(4)               // event.data lower i32
	e.Emit(0x89, 0x02)                   // mov [rdx], eax
	e.MovEaxFromRspDisp(0)               // event.events
	e.Emit(0x89, 0x42, 0x04)             // mov [rdx+4], eax
	e.MovEaxImm32(1)
	e.Leave()
	e.Ret()
	return nil
}

func emitNetSetNonblocking(e *x64.Emitter) error {
	const (
		linuxFGetFL    = 3
		linuxFSetFL    = 4
		linuxONonblock = 2048
	)

	// Arguments: rdi=fd, rsi=cap.io token (ignored).
	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(16)
	e.Emit(0x89, 0x3C, 0x24) // mov [rsp], edi

	e.Emit(0xBE, byte(linuxFGetFL), 0, 0, 0) // mov esi, F_GETFL
	e.Emit(0x31, 0xD2)                       // xor edx, edx
	e.MovEaxImm32(linuxSysFcntl)
	e.Syscall()
	e.TestEaxEax()
	okAt := e.JgeRel32()
	e.Leave()
	e.Ret()

	okTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, okAt, okTo); err != nil {
		return err
	}
	e.Emit(0x0D, byte(linuxONonblock&0xff), byte((linuxONonblock>>8)&0xff), byte((linuxONonblock>>16)&0xff), byte((linuxONonblock>>24)&0xff)) // or eax, O_NONBLOCK
	e.Emit(0x89, 0xC2)                                                                                                                        // mov edx, eax
	e.Emit(0x8B, 0x3C, 0x24)                                                                                                                  // mov edi, [rsp]
	e.Emit(0xBE, byte(linuxFSetFL), 0, 0, 0)                                                                                                  // mov esi, F_SETFL
	e.MovEaxImm32(linuxSysFcntl)
	e.Syscall()
	e.Leave()
	e.Ret()
	return nil
}

func emitNetSetReusePort(e *x64.Emitter) error {
	const (
		linuxSolSocket   = 1
		linuxSoReusePort = 15
	)
	return emitNetSetIntSockOpt(e, linuxSolSocket, linuxSoReusePort)
}

func emitNetSetTCPNoDelay(e *x64.Emitter) error {
	const (
		linuxIPProtoTCP = 6
		linuxTCPNoDelay = 1
	)
	return emitNetSetIntSockOpt(e, linuxIPProtoTCP, linuxTCPNoDelay)
}

func emitNetSetIntSockOpt(e *x64.Emitter, level uint32, optname uint32) error {
	// Arguments: rdi=fd, rsi=cap.io token (ignored).
	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(16)
	emitMovMem32RspDispImm32(e, 0, 1)
	e.Emit(0xBE, byte(level&0xff), byte((level>>8)&0xff), byte((level>>16)&0xff), byte((level>>24)&0xff)) // mov esi, level
	e.MovEdxImm32(optname)
	e.Emit(0x49, 0x89, 0xE2) // mov r10, rsp (optval=&one)
	e.MovR8dImm32(4)         // optlen=sizeof(i32)
	e.MovR9dImm32(0)
	e.MovEaxImm32(linuxSysSetSockOpt)
	e.Syscall()
	e.Leave()
	e.Ret()
	return nil
}

func emitNetClose(e *x64.Emitter) error {
	// Arguments: rdi=fd, rsi=cap.io token (ignored).
	e.MovEaxImm32(linuxSysClose)
	e.Syscall()
	e.Ret()
	return nil
}

func emitActorNodeConnect(e *x64.Emitter) error {
	var failReturnJumps []int
	var failCloseJumps []int

	e.PushRbp()
	e.MovRbpRsp()
	e.SubRspImm32(128)
	e.Emit(0x89, 0x7C, 0x24, 0x70) // node id spill
	e.Emit(0x89, 0x74, 0x24, 0x74) // port spill

	e.CmpEdiImm32(1)
	nodeLowOK := e.JgeRel32()
	failReturnJumps = append(failReturnJumps, e.JmpRel32())
	nodeLowTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nodeLowOK, nodeLowTo); err != nil {
		return err
	}
	e.CmpEdiImm32(maxActors - 1)
	failReturnJumps = append(failReturnJumps, e.JaRel32())

	e.MovEdiImm32(2)
	e.Emit(0xBE, 0x01, 0, 0, 0) // mov esi, SOCK_STREAM
	e.Emit(0x31, 0xD2)          // xor edx, edx
	e.MovEaxImm32(linuxSysSocket)
	e.Syscall()
	e.TestEaxEax()
	socketOK := e.JgeRel32()
	failReturnJumps = append(failReturnJumps, e.JmpRel32())
	socketOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, socketOK, socketOKTo); err != nil {
		return err
	}
	e.Emit(0x89, 0x44, 0x24, 0x78) // fd spill

	emitMovMem16RspDispImm16(e, 0, 2)
	e.MovEaxFromRspDisp(0x74)
	e.Emit(0x86, 0xE0) // xchg al, ah
	emitMovMem16RspDispAx(e, 2)
	emitMovMem32RspDispImm32(e, 4, 0x0100007f)
	e.Emit(0x48, 0xC7, 0x44, 0x24, 0x08, 0, 0, 0, 0)

	e.Emit(0x8B, 0x7C, 0x24, 0x78) // fd
	emitLeaRsiRspDisp(e, 0)
	e.MovEdxImm32(16)
	e.MovEaxImm32(linuxSysConnect)
	e.Syscall()
	e.TestEaxEax()
	connectOK := e.JgeRel32()
	failCloseJumps = append(failCloseJumps, e.JmpRel32())
	connectOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, connectOK, connectOKTo); err != nil {
		return err
	}

	emitActorWireControlFrame(e, 0x20, actorWireFrameHello)
	e.MovEaxFromRspDisp(0x70)
	emitMovMem16RspDispAx(e, 0x20+actorWireOffsetSrc)
	emitMovMem16RspDispAx(e, 0x20+actorWireOffsetDest)
	e.Emit(0x8B, 0x7C, 0x24, 0x78)
	emitLeaRsiRspDisp(e, 0x20)
	e.MovEdxImm32(actorWireFrameSize)
	e.MovEaxImm32(linuxSysWrite)
	e.Syscall()
	e.CmpEaxImm32(actorWireFrameSize)
	writeOK := e.JzRel32()
	failCloseJumps = append(failCloseJumps, e.JmpRel32())
	writeOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, writeOK, writeOKTo); err != nil {
		return err
	}

	e.Emit(0x8B, 0x7C, 0x24, 0x78)
	emitLeaRsiRspDisp(e, 0x20)
	e.MovEdxImm32(actorWireFrameSize)
	e.MovEaxImm32(linuxSysRead)
	e.Syscall()
	e.CmpEaxImm32(actorWireFrameSize)
	readOK := e.JzRel32()
	failCloseJumps = append(failCloseJumps, e.JmpRel32())
	readOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, readOK, readOKTo); err != nil {
		return err
	}
	emitMovEaxRspDisp(e, 0x20+actorWireOffsetMagic)
	e.CmpEaxImm32(actorWireMagic)
	ackMagicOK := e.JzRel32()
	failCloseJumps = append(failCloseJumps, e.JmpRel32())
	ackMagicOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, ackMagicOK, ackMagicOKTo); err != nil {
		return err
	}
	emitMovzxEaxWordRspDisp(e, 0x20+actorWireOffsetType)
	e.CmpEaxImm32(actorWireFrameHelloAck)
	ackTypeOK := e.JzRel32()
	failCloseJumps = append(failCloseJumps, e.JmpRel32())
	ackTypeOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, ackTypeOK, ackTypeOKTo); err != nil {
		return err
	}
	emitMovEaxRspDisp(e, 0x20+actorWireOffsetStatus)
	e.TestEaxEax()
	ackStatusOK := e.JzRel32()
	failCloseJumps = append(failCloseJumps, e.JmpRel32())
	ackStatusOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, ackStatusOK, ackStatusOKTo); err != nil {
		return err
	}

	e.MovRdiR15()
	e.MovEaxFromRspDisp(0x78)
	e.MovMem32RdiDispEax(schedNetFDOff)
	e.MovEaxFromRspDisp(0x70)
	e.MovMem32RdiDispEax(schedNodeIDOff)
	e.MovMem32RdiDispImm32(schedNetStatusOff, 0)
	e.XorEaxEax()
	e.Leave()
	e.Ret()

	failCloseTo := len(e.Buf)
	for _, at := range failCloseJumps {
		if err := x64.PatchRel32(e.Buf, at, failCloseTo); err != nil {
			return err
		}
	}
	e.Emit(0x8B, 0x7C, 0x24, 0x78)
	e.MovEaxImm32(linuxSysClose)
	e.Syscall()

	failReturnTo := len(e.Buf)
	for _, at := range failReturnJumps {
		if err := x64.PatchRel32(e.Buf, at, failReturnTo); err != nil {
			return err
		}
	}
	e.MovRdiR15()
	e.MovMem32RdiDispImm32(schedNetStatusOff, 1)
	e.MovEaxImm32(1)
	e.Leave()
	e.Ret()
	return nil
}

func emitActorSpawnRemote(e *x64.Emitter) error {
	var failJumps []int

	e.PushRbp()
	e.MovRbpRsp()
	e.Emit(0x48, 0x83, 0xEC, 0x70) // sub rsp, 112
	e.Emit(0x89, 0x7C, 0x24, 0x60) // remote node
	e.Emit(0x89, 0x74, 0x24, 0x64) // entry id

	e.CmpEdiImm32(1)
	nodeLowOK := e.JgeRel32()
	failJumps = append(failJumps, e.JmpRel32())
	nodeLowTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nodeLowOK, nodeLowTo); err != nil {
		return err
	}
	e.CmpEdiImm32(maxActors - 1)
	failJumps = append(failJumps, e.JaRel32())

	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedNetFDOff)
	e.TestEaxEax()
	fdOK := e.JnzRel32()
	failJumps = append(failJumps, e.JmpRel32())
	fdOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, fdOK, fdOKTo); err != nil {
		return err
	}
	e.Emit(0x89, 0x44, 0x24, 0x68) // fd

	emitActorWireControlFrame(e, 0, actorWireFrameSpawn)
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedNodeIDOff)
	emitMovMem16RspDispAx(e, actorWireOffsetSrc)
	e.MovEaxFromRspDisp(0x60)
	emitMovMem16RspDispAx(e, actorWireOffsetDest)
	e.MovEaxFromRspDisp(0x64)
	e.Emit(0x89, 0x44, 0x24, actorWireOffsetTag)

	e.Emit(0x8B, 0x7C, 0x24, 0x68)
	emitLeaRsiRspDisp(e, 0)
	e.MovEdxImm32(actorWireFrameSize)
	e.MovEaxImm32(linuxSysWrite)
	e.Syscall()
	e.CmpEaxImm32(actorWireFrameSize)
	writeOK := e.JzRel32()
	failJumps = append(failJumps, e.JmpRel32())
	writeOKTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, writeOK, writeOKTo); err != nil {
		return err
	}

	e.MovEaxFromRspDisp(0x60)
	e.Emit(0xC1, 0xE0, 0x10)
	e.Emit(0x0D, 0x00, 0x00, 0x00, 0x80)
	e.MovEdxFromRspDisp(0x64)
	e.Emit(0x81, 0xE2, 0xFF, 0xFF, 0x00, 0x00)
	e.Emit(0x09, 0xD0)
	e.Leave()
	e.Ret()

	failTo := len(e.Buf)
	for _, at := range failJumps {
		if err := x64.PatchRel32(e.Buf, at, failTo); err != nil {
			return err
		}
	}
	e.MovEaxImm32(0xFFFFFFFF)
	e.Leave()
	e.Ret()
	return nil
}

func emitActorNodeStatus(e *x64.Emitter) error {
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedNetFDOff)
	e.TestEaxEax()
	connectedAt := e.JnzRel32()
	e.MovEaxImm32(1)
	e.Ret()
	connectedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, connectedAt, connectedTo); err != nil {
		return err
	}
	e.MovEaxFromRdiDisp(schedNetStatusOff)
	e.Ret()
	return nil
}

func emitActorWireControlFrame(e *x64.Emitter, base byte, frameType uint16) {
	emitMovMem32RspDispImm32(e, base+actorWireOffsetMagic, actorWireMagic)
	emitMovMem16RspDispImm16(e, base+actorWireOffsetVer, actorWireVersion)
	emitMovMem16RspDispImm16(e, base+actorWireOffsetType, frameType)
	emitMovMem32RspDispImm32(e, base+actorWireOffsetSeq, 0)
	emitMovMem16RspDispImm16(e, base+actorWireOffsetActor, 0)
	emitMovMem16RspDispImm16(e, base+actorWireOffsetSlots, 0)
	emitMovMem32RspDispImm32(e, base+actorWireOffsetTag, 0)
}

func emitTaskGroupClose(e *x64.Emitter, callPatches *[]callPatch) error {
	// Argument: rdi=task.group handle. Returns 0 on close, 1 for an invalid group.
	e.MovEaxEdi()
	e.TestEaxEax()
	nonzeroAt := e.JnzRel32()
	e.MovEaxImm32(1)
	e.Ret()

	nonzeroTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nonzeroAt, nonzeroTo); err != nil {
		return err
	}
	e.MovRdiR15()
	e.MovMem32RdiDispEax(schedCloseGroupOff)
	e.MovEdiEax()
	groupStatePtrFromEdi(e)
	e.MovEaxFromRdiDisp(0)
	e.CmpEaxImm32(taskGroupClosed)
	notClosedAt := e.JnzRel32()
	e.XorEaxEax()
	e.Ret()

	notClosedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notClosedAt, notClosedTo); err != nil {
		return err
	}
	loopStart := len(e.Buf)
	e.MovEaxImm32(1)

	scan := len(e.Buf)
	e.MovRdiR15()
	e.MovEcxFromRdiDisp(schedCountOff)
	e.CmpEaxEcx()
	doneAt := e.JaeRel32()
	e.PushRax()

	e.MovEcxEax()
	e.MovRbxRcx()
	e.ShlRbxImm8(actorSizeShift)
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.AddRaxRbx()
	e.MovRdiRax()

	e.PushRdi()
	e.MovEaxFromRdiDisp(actorTaskGroupOff)
	e.MovRdiR15()
	e.MovEdxFromRdiDisp(schedCloseGroupOff)
	e.CmpEaxEdx()
	e.PopRdi()
	notGroupAt := e.JnzRel32()
	e.MovEaxFromRdiDisp(actorStatusOff)
	e.CmpEaxImm32(statusDone)
	doneActorAt := e.JzRel32()

	e.PopRax()
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: "__tetra_actor_yield"})
	backToLoopAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, backToLoopAt, loopStart); err != nil {
		return err
	}

	continueTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, notGroupAt, continueTo); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, doneActorAt, continueTo); err != nil {
		return err
	}
	e.PopRax()
	e.AddEaxImm32(1)
	nextAt := e.JmpRel32()
	if err := x64.PatchRel32(e.Buf, nextAt, scan); err != nil {
		return err
	}

	doneTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, doneAt, doneTo); err != nil {
		return err
	}
	e.MovRdiR15()
	e.MovEaxFromRdiDisp(schedCloseGroupOff)
	e.MovEdiEax()
	groupStatePtrFromEdi(e)
	e.MovEaxFromRdiDisp(0)
	e.CmpEaxImm32(taskGroupCanceled)
	keepCanceledAt := e.JzRel32()
	e.MovMem32RdiDispImm32(0, taskGroupClosed)
	keepCanceledTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, keepCanceledAt, keepCanceledTo); err != nil {
		return err
	}
	e.XorEaxEax()
	e.Ret()
	return nil
}

func emitTaskSpawnGroupI32(e *x64.Emitter, actorSpawn string, callPatches *[]callPatch) error {
	// Arguments: rdi=task.group handle, rsi=entryID.
	// Returns task.i32 layout: rax=actor handle, rdx=error status.
	e.MovEaxEdi()
	e.TestEaxEax()
	nonzeroAt := e.JnzRel32()
	e.XorEaxEax()
	e.MovEdxImm32(1)
	e.Ret()

	nonzeroTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nonzeroAt, nonzeroTo); err != nil {
		return err
	}
	e.PushRdi()
	e.PushRsi()
	groupStatePtrFromEdi(e)
	e.MovEaxFromRdiDisp(0)
	e.CmpEaxImm32(taskGroupOpen)
	openAt := e.JzRel32()
	e.PopRsi()
	e.PopRdi()
	e.XorEaxEax()
	e.MovEdxImm32(1)
	e.Ret()

	openTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, openAt, openTo); err != nil {
		return err
	}
	e.PopRsi()
	e.PopRdi()
	e.PushRdi()
	e.MovEdxEdi()
	setPendingSpawnGroupFromEdx(e)
	e.MovRdiRsi()
	at := e.CallRel32()
	*callPatches = append(*callPatches, callPatch{at: at, name: actorSpawn})

	e.CmpEaxImm32(-1)
	spawnedAt := e.JnzRel32()
	e.PopRdx()
	e.XorEaxEax()
	e.MovEdxImm32(1)
	e.Ret()

	spawnedTo := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, spawnedAt, spawnedTo); err != nil {
		return err
	}
	e.PopRdx()
	storeActorGroupForHandleInRaxGroupInRdx(e)
	e.PushRax()
	e.MovRcxRax()
	e.MovRbxRcx()
	e.ShlRbxImm8(actorSizeShift)
	e.MovRdiR15()
	e.MovRaxFromRdiDisp(schedActorsPtrOff)
	e.AddRaxRbx()
	e.MovRdiRax()
	storeActorSavedGroupForActorPtrInRdiGroupInRdx(e)
	e.PopRax()
	e.MovEdxImm32(0)
	e.Ret()
	return nil
}
