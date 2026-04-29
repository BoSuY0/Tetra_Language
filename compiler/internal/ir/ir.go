package ir

import "tetra_language/compiler/internal/frontend"

type IRProgram struct {
	Funcs     []IRFunc
	MainIndex int
	MainName  string
}

type IRFunc struct {
	Name        string
	ExportName  string
	ParamSlots  int
	LocalSlots  int
	ReturnSlots int
	Instrs      []IRInstr
}

type IRInstrKind int

const (
	IRWrite IRInstrKind = iota
	IRStrLit
	IRConstI32
	IRLoadLocal
	IRStoreLocal
	IRLoadGlobal
	IRStoreGlobal
	IRAddI32
	IRSubI32
	IRNegI32
	IRCmpEqI32
	IRCmpLtI32
	IRMulI32
	IRDivI32
	IRModI32
	IRCmpGtI32
	IRCmpGeI32
	IRCmpLeI32
	IRCmpNeI32
	IRCall
	IRLabel
	IRJmp
	IRJmpIfZero
	IRReturn
	IRAllocBytes
	IRMakeSliceU8
	IRMakeSliceU16
	IRMakeSliceI32
	IRIndexLoadI32
	IRIndexStoreI32
	IRIndexLoadU8
	IRIndexStoreU8
	IRIndexLoadU16
	IRIndexStoreU16
	// Islands memory model
	IRIslandNew
	IRIslandMakeSliceU8
	IRIslandMakeSliceU16
	IRIslandMakeSliceI32
	IRIslandFree
	// Capabilities + MMIO
	IRCapIO
	IRCapMem
	IRMemReadI32
	IRMemWriteI32
	IRMemReadU8
	IRMemWriteU8
	IRMemReadPtr
	IRMemWritePtr
	IRPtrAdd
	IRMmioReadI32
	IRMmioWriteI32
	IRSymAddr
	IRCtxSwitch
)

// Stack effects:
// STRLIT -> push2
// CONST/LOAD -> push
// STORE -> pop
// WRITE -> pop2
// ADD/SUB/CMP -> pop2 push1
// MUL/DIV/MOD -> pop2 push1
// CMP_GT/GE/LE/NE -> pop2 push1
// NEG -> pop1 push1
// CALL -> popN pushRet
// JMP_IF_ZERO -> pop1
// RET -> popRet
// ALLOC -> pop1 push1
// MAKE_SLICE -> pop1 push2
// INDEX_LOAD -> pop3 push1
// INDEX_STORE -> pop4
// ISLAND_NEW -> pop1 push1
// ISLAND_MAKE_SLICE -> pop2 push2
// ISLAND_FREE -> pop1
// CAP -> push1
// MEM_READ -> pop2 push1
// MEM_WRITE -> pop3 push1
// MEM_READ_U8 -> pop2 push1
// MEM_WRITE_U8 -> pop3 push1
// MEM_READ_PTR -> pop2 push1
// MEM_WRITE_PTR -> pop3 push1
// PTR_ADD -> pop3 push1
// MMIO_READ -> pop2 push1
// MMIO_WRITE -> pop3 push1
// SYM_ADDR -> push1
// CTX_SWITCH -> pop3 push1

type IRInstr struct {
	Kind     IRInstrKind
	Imm      int32
	Local    int
	Label    int
	Name     string
	ArgSlots int
	RetSlots int
	Str      []byte
	Pos      frontend.Position
}
