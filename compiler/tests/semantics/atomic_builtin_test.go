package compiler_test

import (
	"testing"

	"tetra_language/compiler/internal/testkit"
)

func TestAtomicBuiltinInvalidFormsReportExplicitDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		call string
		want string
	}{
		{
			name: "load release",
			call: "core.atomic_load_i32_release(p, mem)",
			want: "atomic load does not support memory order release",
		},
		{
			name: "store acquire",
			call: "core.atomic_store_i32_acquire(p, 1, mem)",
			want: "atomic store does not support memory order acquire",
		},
		{
			name: "unknown order",
			call: "core.atomic_fetch_add_i32_consume(p, 1, mem)",
			want: "unsupported atomic memory order 'consume'",
		},
		{
			name: "unknown op",
			call: "core.atomic_nand_i32_relaxed(p, 1, mem)",
			want: "unsupported atomic operation 'nand'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireCheckErrorContains(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        return `+tt.call+`
    return 0
`, tt.want)
		})
	}
}

func TestAtomicBuiltinI64AndWeakCompareExchangeSurfaceChecks(t *testing.T) {
	testkit.RequireCheckOK(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let loaded: i64 = core.atomic_load_i64_acquire(p, mem)
        let exchanged: i64 = core.atomic_exchange_i64_seq_cst(p, loaded, mem)
        let weak: i64 = core.atomic_compare_exchange_weak_i64_seq_cst(p, loaded, exchanged, mem)
        var ignored_store: i64 = core.atomic_store_i64_release(p, weak, mem)
        return 0
    return 0
`)
}
