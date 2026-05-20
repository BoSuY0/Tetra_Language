package x64abi

import (
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64obj"
	"tetra_language/compiler/internal/ir"
)

const maxCallReturnSlots = 10

type ABI interface {
	SpillParams(e *x64.Emitter, fn ir.IRFunc)

	EmitCall(e *x64.Emitter, instr ir.IRInstr, stackDepth *int, callPatches *[]x64obj.CallPatch) error
	EmitWriteStdout(e *x64.Emitter, stackDepth *int, importPatches *[]x64obj.ImportPatch) error
	EmitExit(e *x64.Emitter, code int32, stackSlots int, importPatches *[]x64obj.ImportPatch) error

	EmitAllocBytes(e *x64.Emitter, stackDepth *int, opt x64.CodegenOptions, importPatches *[]x64obj.ImportPatch) error
	EmitMakeSlice(e *x64.Emitter, kind ir.IRInstrKind, stackDepth *int, opt x64.CodegenOptions, importPatches *[]x64obj.ImportPatch) error

	EmitIslandNew(e *x64.Emitter, stackDepth *int, opt x64.CodegenOptions, importPatches *[]x64obj.ImportPatch) error
	EmitIslandMakeSlice(e *x64.Emitter, kind ir.IRInstrKind, stackDepth *int, opt x64.CodegenOptions, importPatches *[]x64obj.ImportPatch) error
	EmitIslandFree(e *x64.Emitter, stackDepth *int, opt x64.CodegenOptions, importPatches *[]x64obj.ImportPatch) error
}
