package lower

import (
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestLowerCoreAtomicI32BuiltinsToIR(t *testing.T) {
	fn := lowerCallableFunc(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(16)
        var ignored_store: i32 = core.atomic_store_i32_release(p, 1, mem)
        let loaded: i32 = core.atomic_load_i32_acquire(p, mem)
        let exchanged: i32 = core.atomic_exchange_i32_seq_cst(p, 2, mem)
        let cas: i32 = core.atomic_compare_exchange_i32_acq_rel(p, loaded, exchanged, mem)
        let add: i32 = core.atomic_fetch_add_i32_relaxed(p, 3, mem)
        let sub: i32 = core.atomic_fetch_sub_i32_seq_cst(p, 1, mem)
        let anded: i32 = core.atomic_fetch_and_i32_acquire(p, 7, mem)
        let ored: i32 = core.atomic_fetch_or_i32_release(p, 8, mem)
        let xored: i32 = core.atomic_fetch_xor_i32_acq_rel(p, 9, mem)
        var ignored_fence: i32 = core.atomic_fence_seq_cst(mem)
        return loaded + exchanged + cas + add + sub + anded + ored + xored
    return 0
`, "main")

	for _, tc := range []struct {
		name string
		kind ir.IRInstrKind
	}{
		{"load", ir.IRAtomicLoadI32},
		{"store", ir.IRAtomicStoreI32},
		{"exchange", ir.IRAtomicExchangeI32},
		{"compare_exchange", ir.IRAtomicCompareExchangeI32},
		{"fetch_add", ir.IRAtomicFetchAddI32},
		{"fetch_sub", ir.IRAtomicFetchSubI32},
		{"fetch_and", ir.IRAtomicFetchAndI32},
		{"fetch_or", ir.IRAtomicFetchOrI32},
		{"fetch_xor", ir.IRAtomicFetchXorI32},
		{"fence", ir.IRAtomicFenceSeqCst},
	} {
		if got := countInstr(fn.Instrs, tc.kind, ""); got != 1 {
			t.Fatalf("atomic %s should lower to one %v, got %d: %#v", tc.name, tc.kind, got, fn.Instrs)
		}
	}
}

func TestLowerCoreAtomicSmallAndPointerBuiltinsToIR(t *testing.T) {
	fn := lowerCallableFunc(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(16)
        let byte: u8 = 1
        let word: u16 = 2
        let old_byte: u8 = core.atomic_exchange_u8_seq_cst(p, byte, mem)
        let old_word: u16 = core.atomic_exchange_u16_seq_cst(p, word, mem)
        let loaded: ptr = core.atomic_load_ptr_acquire(p, mem)
        var ignored_store: ptr = core.atomic_store_ptr_release(p, loaded, mem)
        let swapped: ptr = core.atomic_exchange_ptr_seq_cst(p, loaded, mem)
        let cas: ptr = core.atomic_compare_exchange_ptr_acq_rel(p, loaded, swapped, mem)
        let add: ptr = core.atomic_fetch_add_ptr_relaxed(p, loaded, mem)
        let sub: ptr = core.atomic_fetch_sub_ptr_seq_cst(p, loaded, mem)
        let anded: ptr = core.atomic_fetch_and_ptr_acquire(p, loaded, mem)
        let ored: ptr = core.atomic_fetch_or_ptr_release(p, loaded, mem)
        let xored: ptr = core.atomic_fetch_xor_ptr_acq_rel(p, loaded, mem)
        return old_byte + old_word
    return 0
`, "main")

	for _, tc := range []struct {
		name string
		kind ir.IRInstrKind
	}{
		{"u8 exchange", ir.IRAtomicExchangeI8},
		{"u16 exchange", ir.IRAtomicExchangeI16},
		{"ptr load", ir.IRAtomicLoadPtr},
		{"ptr store", ir.IRAtomicStorePtr},
		{"ptr exchange", ir.IRAtomicExchangePtr},
		{"ptr compare_exchange", ir.IRAtomicCompareExchangePtr},
		{"ptr fetch_add", ir.IRAtomicFetchAddPtr},
		{"ptr fetch_sub", ir.IRAtomicFetchSubPtr},
		{"ptr fetch_and", ir.IRAtomicFetchAndPtr},
		{"ptr fetch_or", ir.IRAtomicFetchOrPtr},
		{"ptr fetch_xor", ir.IRAtomicFetchXorPtr},
	} {
		if got := countInstr(fn.Instrs, tc.kind, ""); got != 1 {
			t.Fatalf("atomic %s should lower to one %v, got %d: %#v", tc.name, tc.kind, got, fn.Instrs)
		}
	}
}

func TestLowerCoreAtomicI64AndWeakCompareExchangeBuiltinsToIR(t *testing.T) {
	fn := lowerCallableFunc(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let loaded: i64 = core.atomic_load_i64_acquire(p, mem)
        let exchanged: i64 = core.atomic_exchange_i64_seq_cst(p, loaded, mem)
        let weak_i64: i64 = core.atomic_compare_exchange_weak_i64_seq_cst(p, loaded, exchanged, mem)
        let weak_i32: i32 = core.atomic_compare_exchange_weak_i32_seq_cst(p, 0, 1, mem)
        var ignored_store: i64 = core.atomic_store_i64_release(p, weak_i64, mem)
        return weak_i32
    return 0
`, "main")

	for _, tc := range []struct {
		name string
		kind ir.IRInstrKind
	}{
		{"i64 load", ir.IRAtomicLoadI64},
		{"i64 exchange", ir.IRAtomicExchangeI64},
		{"i64 weak compare_exchange", ir.IRAtomicCompareExchangeI64},
		{"i64 store", ir.IRAtomicStoreI64},
		{"i32 weak compare_exchange", ir.IRAtomicCompareExchangeI32},
	} {
		if got := countInstr(fn.Instrs, tc.kind, ""); got != 1 {
			t.Fatalf("atomic %s should lower to one %v, got %d: %#v", tc.name, tc.kind, got, fn.Instrs)
		}
	}
}
