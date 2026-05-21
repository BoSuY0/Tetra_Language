package lower

import (
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestLowerRawPtrAddDirectOffsetMemoryAccessIR(t *testing.T) {
	fn := lowerCallableFunc(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(16)
        let stored: Int = core.store_i32(core.ptr_add(p, 4, mem), 42, mem)
        let value: Int = core.load_i32(core.ptr_add(p, 4, mem), mem)
        let stored_ptr: ptr = core.store_ptr(core.ptr_add(p, 8, mem), p, mem)
        let loaded_ptr: ptr = core.load_ptr(core.ptr_add(p, 8, mem), mem)
        let stored_arch_ptr: ptr = core.store_arch_ptr(core.ptr_add(p, 8, mem), p, mem)
        return value
    return 0
`, "main")

	if got := countInstr(fn.Instrs, ir.IRMemWriteI32Offset, ""); got != 1 {
		t.Fatalf("direct ptr_add store_i32 should lower to one offset write, got %d: %#v", got, fn.Instrs)
	}
	if got := countInstr(fn.Instrs, ir.IRMemReadI32Offset, ""); got != 1 {
		t.Fatalf("direct ptr_add load_i32 should lower to one offset read, got %d: %#v", got, fn.Instrs)
	}
	if got := countInstr(fn.Instrs, ir.IRMemWritePtrOffset, ""); got != 1 {
		t.Fatalf("direct ptr_add store_ptr should lower to one offset write, got %d: %#v", got, fn.Instrs)
	}
	if got := countInstr(fn.Instrs, ir.IRMemReadPtrOffset, ""); got != 1 {
		t.Fatalf("direct ptr_add load_ptr should lower to one offset read, got %d: %#v", got, fn.Instrs)
	}
	if got := countInstr(fn.Instrs, ir.IRMemWriteArchPtrOffset, ""); got != 1 {
		t.Fatalf("direct ptr_add store_arch_ptr should lower to one offset write, got %d: %#v", got, fn.Instrs)
	}
	if got := countInstr(fn.Instrs, ir.IRPtrAdd, ""); got != 0 {
		t.Fatalf("direct ptr_add memory access should fold into offset IR, got %d ptr_add instructions: %#v", got, fn.Instrs)
	}
}

func TestLowerRawPtrAddLocalOffsetMemoryAccessIR(t *testing.T) {
	fn := lowerCallableFunc(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let q: ptr = core.ptr_add(p, 4, mem)
        let stored: Int = core.store_i32(q, 42, mem)
        return core.load_i32(q, mem)
    return 0
`, "main")

	if got := countInstr(fn.Instrs, ir.IRMemWriteI32Offset, ""); got != 1 {
		t.Fatalf("local ptr_add store_i32 should lower to one offset write, got %d: %#v", got, fn.Instrs)
	}
	if got := countInstr(fn.Instrs, ir.IRMemReadI32Offset, ""); got != 1 {
		t.Fatalf("local ptr_add load_i32 should lower to one offset read, got %d: %#v", got, fn.Instrs)
	}
	if got := countInstr(fn.Instrs, ir.IRPtrAdd, ""); got != 1 {
		t.Fatalf("local ptr_add should keep exactly one value-producing ptr_add for q initialization, got %d: %#v", got, fn.Instrs)
	}
}

func TestLowerRawPtrAddMutableLocalWithDiscardOffsetMemoryAccessIR(t *testing.T) {
	fn := lowerCallableFunc(t, `
func main() -> i32
uses alloc, capability, mem:
    var out: i32 = 1
    unsafe:
        var mem: cap.mem = core.cap_mem()
        var p: ptr = core.alloc_bytes(8)
        var q: ptr = core.ptr_add(p, 4, mem)
        var _: i32 = core.store_i32(q, 123, mem)
        var v: i32 = core.load_i32(q, mem)
        if v == 123:
            out = 77
        else:
            out = 1
    return out
`, "main")

	if got := countInstr(fn.Instrs, ir.IRMemWriteI32Offset, ""); got != 1 {
		t.Fatalf("mutable local ptr_add store_i32 should lower to one offset write, got %d: %#v", got, fn.Instrs)
	}
	if got := countInstr(fn.Instrs, ir.IRMemReadI32Offset, ""); got != 1 {
		t.Fatalf("mutable local ptr_add load_i32 should lower to one offset read, got %d: %#v", got, fn.Instrs)
	}
}
