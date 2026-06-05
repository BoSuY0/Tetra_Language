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
	Policy      IRPolicy
	Instrs      []IRInstr
}

type IRPolicy struct {
	HasBudget    bool
	Budget       int32
	BudgetLocal  int
	HasConsent   bool
	ConsentLocal int
	FailLabel    int
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
	IRStackSliceU8
	IRStackSliceU16
	IRStackSliceI32
	IRRegionEnter
	IRRegionMakeSliceU8
	IRRegionMakeSliceU16
	IRRegionMakeSliceI32
	IRRegionReset
	IRRawSliceFromParts
	IRSliceWindow
	IRSlicePrefix
	IRSliceSuffix
	IRIndexLoadI32
	IRIndexLoadI32Unchecked
	IRIndexStoreI32
	IRIndexLoadU8
	IRIndexLoadU8Unchecked
	IRIndexStoreU8
	IRIndexLoadU16
	IRIndexLoadU16Unchecked
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
	IRMemWriteArchPtr
	IRMemReadI32Offset
	IRMemWriteI32Offset
	IRMemReadU8Offset
	IRMemWriteU8Offset
	IRMemReadPtrOffset
	IRMemWritePtrOffset
	IRMemWriteArchPtrOffset
	IRPtrAdd
	IRMmioReadI32
	IRMmioWriteI32
	IRSymAddr
	IRCtxSwitch
	IRAtomicLoadPtr
	IRAtomicStorePtr
	IRAtomicExchangePtr
	IRAtomicFetchAddPtr
	IRAtomicFetchSubPtr
	IRAtomicFetchAndPtr
	IRAtomicFetchOrPtr
	IRAtomicFetchXorPtr
	IRAtomicCompareExchangePtr
	IRAtomicFenceSeqCst
	IRAtomicFenceRelaxed
	IRAtomicFenceAcquire
	IRAtomicFenceRelease
	IRAtomicFenceAcqRel
	IRAtomicLoadI32
	IRAtomicStoreI32
	IRAtomicExchangeI32
	IRAtomicCompareExchangeI32
	IRAtomicFetchAddI32
	IRAtomicFetchSubI32
	IRAtomicFetchAndI32
	IRAtomicFetchOrI32
	IRAtomicFetchXorI32
	IRAtomicLoadI64
	IRAtomicStoreI64
	IRAtomicExchangeI64
	IRAtomicCompareExchangeI64
	IRAtomicFetchAddI64
	IRAtomicFetchSubI64
	IRAtomicFetchAndI64
	IRAtomicFetchOrI64
	IRAtomicFetchXorI64
	IRAtomicLoadI8
	IRAtomicStoreI8
	IRAtomicExchangeI8
	IRAtomicCompareExchangeI8
	IRAtomicFetchAddI8
	IRAtomicFetchSubI8
	IRAtomicFetchAndI8
	IRAtomicFetchOrI8
	IRAtomicFetchXorI8
	IRAtomicLoadI16
	IRAtomicStoreI16
	IRAtomicExchangeI16
	IRAtomicCompareExchangeI16
	IRAtomicFetchAddI16
	IRAtomicFetchSubI16
	IRAtomicFetchAndI16
	IRAtomicFetchOrI16
	IRAtomicFetchXorI16
	// IRInstrKindCount is a sentinel for exhaustive IR kind coverage checks.
	IRInstrKindCount
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
// STACK_SLICE -> pop1 push2
// REGION_ENTER -> no stack effect
// REGION_MAKE_SLICE -> pop1 push2
// REGION_RESET -> no stack effect
// RAW_SLICE_FROM_PARTS -> pop3 push2
// SLICE_WINDOW -> pop4 push2 (ptr, len, start, count -> ptr+start*elem, count)
// SLICE_PREFIX/SUFFIX -> pop3 push2
// INDEX_LOAD -> pop3 push1
// INDEX_LOAD_UNCHECKED -> pop3 push1 (requires ProofID)
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
// MEM_WRITE_ARCH_PTR -> pop3 push1
// MEM_READ_I32_OFFSET -> pop3 push1
// MEM_WRITE_I32_OFFSET -> pop4 push1
// MEM_READ_U8_OFFSET -> pop3 push1
// MEM_WRITE_U8_OFFSET -> pop4 push1
// MEM_READ_PTR_OFFSET -> pop3 push1
// MEM_WRITE_PTR_OFFSET -> pop4 push1
// MEM_WRITE_ARCH_PTR_OFFSET -> pop4 push1
// PTR_ADD -> pop3 push1
// MMIO_READ -> pop2 push1
// MMIO_WRITE -> pop3 push1
// SYM_ADDR -> push1
// CTX_SWITCH -> pop3 push1
// ATOMIC_LOAD_PTR -> pop2 push1
// ATOMIC_STORE_PTR -> pop3 push1
// ATOMIC_EXCHANGE_PTR -> pop3 push1
// ATOMIC_FETCH_ADD_PTR -> pop3 push1
// ATOMIC_FETCH_SUB_PTR -> pop3 push1
// ATOMIC_FETCH_AND_PTR -> pop3 push1
// ATOMIC_FETCH_OR_PTR -> pop3 push1
// ATOMIC_FETCH_XOR_PTR -> pop3 push1
// ATOMIC_COMPARE_EXCHANGE_PTR -> pop4 push1
// ATOMIC_FENCE_SEQ_CST -> no stack effect
// ATOMIC_FENCE_RELAXED/ACQUIRE/RELEASE/ACQ_REL -> no stack effect
// ATOMIC_LOAD_I32 -> pop2 push1
// ATOMIC_STORE_I32 -> pop3 push1
// ATOMIC_EXCHANGE_I32 -> pop3 push1
// ATOMIC_COMPARE_EXCHANGE_I32 -> pop4 push1
// ATOMIC_FETCH_*_I32 -> pop3 push1
// ATOMIC_LOAD_I64 -> pop2 push1
// ATOMIC_STORE_I64 -> pop3 push1
// ATOMIC_EXCHANGE_I64 -> pop3 push1
// ATOMIC_COMPARE_EXCHANGE_I64 -> pop4 push1
// ATOMIC_FETCH_*_I64 -> pop3 push1
// ATOMIC_LOAD_I8/I16 -> pop2 push1
// ATOMIC_STORE_I8/I16 -> pop3 push1
// ATOMIC_EXCHANGE_I8/I16 -> pop3 push1
// ATOMIC_COMPARE_EXCHANGE_I8/I16 -> pop4 push1
// ATOMIC_FETCH_*_I8/I16 -> pop3 push1

type IRInstr struct {
	Kind     IRInstrKind
	Imm      int32
	Local    int
	Label    int
	Name     string
	ArgSlots int
	RetSlots int
	ProofID  string
	Str      []byte
	Pos      frontend.Position
}
