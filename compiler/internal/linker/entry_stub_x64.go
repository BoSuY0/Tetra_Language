package linker

import "tetra_language/compiler/internal/backend/x64"

func emitEntryStubSysVUnixX64(sysExit uint32) ([]byte, int) {
	e := &x64.Emitter{}
	callAt := e.CallRel32()
	e.MovRdiRax()
	e.MovEaxImm32(sysExit)
	e.Syscall()
	return e.Buf, callAt
}

func emitEntryStubWin64X64() ([]byte, int, int) {
	e := &x64.Emitter{}
	e.SubRspImm32(32)
	callMainAt := e.CallRel32()
	e.MovEcxEax()
	callExitAt := e.CallRipDisp32()
	return e.Buf, callMainAt, callExitAt
}
