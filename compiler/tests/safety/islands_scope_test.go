package compiler_test

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/testkit"
)

func TestScopedIslandOk(t *testing.T) {
	src := tetraSource(
		"fun main(): i32 uses alloc, islands, mem {",
		"  island(64) as isl {",
		"    var xs: []u8 = core.island_make_u8(isl, 4)",
		"    xs[0] = 1",
		"  }",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestScopedIslandReturnEscape(t *testing.T) {
	src := tetraSource(
		"fun make(): []u8 {",
		"  island(16) as isl {",
		"    var xs: []u8 = core.island_make_u8(isl, 4)",
		"    return xs",
		"  }",
		"  return make_u8(1)",
		"}",
	)
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestScopedIslandBorrowedViewReturnEscape(t *testing.T) {
	src := tetraSource(
		"fun make(): []u8 uses alloc, islands, mem {",
		"  island(16) as isl {",
		"    var xs: []u8 = core.island_make_u8(isl, 4)",
		"    return xs.window(0, 1).borrow()",
		"  }",
		"  return make_u8(1)",
		"}",
		"fun main(): i32 {",
		"  return 0",
		"}",
	)
	err := testkit.CheckProgram(src)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "borrowed slice return") {
		t.Fatalf("expected borrowed return diagnostic, got: %v", err)
	}
}

func TestScopedIslandCopyReturnEscapeAllowed(t *testing.T) {
	src := tetraSource(
		"fun make(): []u8 uses alloc, islands, mem {",
		"  island(16) as isl {",
		"    var xs: []u8 = core.island_make_u8(isl, 4)",
		"    xs[0] = 7",
		"    return xs.window(0, 1).copy()",
		"  }",
		"  return make_u8(1)",
		"}",
		"fun main(): i32 uses alloc, islands, mem {",
		"  let out: []u8 = make()",
		"  return out.len",
		"}",
	)
	if err := testkit.CheckProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestScopedIslandOptionalReturnEscape(t *testing.T) {
	src := tetraSource(
		"fun make(): []u8? {",
		"  island(16) as isl {",
		"    var xs: []u8 = core.island_make_u8(isl, 4)",
		"    var maybe: []u8? = none",
		"    maybe = xs",
		"    return maybe",
		"  }",
		"  return none",
		"}",
	)
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestScopedIslandAssignOuterEscape(t *testing.T) {
	src := tetraSource(
		"fun main(): i32 {",
		"  var out: []u8 = make_u8(1)",
		"  island(16) as isl {",
		"    out = core.island_make_u8(isl, 4)",
		"  }",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestScopedIslandOptionalAssignOuterEscape(t *testing.T) {
	src := tetraSource(
		"fun main(): i32 {",
		"  var out: []u8? = none",
		"  island(16) as isl {",
		"    var xs: []u8 = core.island_make_u8(isl, 4)",
		"    out = xs",
		"  }",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestScopedIslandLocalScope(t *testing.T) {
	src := tetraSource(
		"fun main(): i32 {",
		"  island(16) as isl {",
		"    let x: i32 = 1",
		"  }",
		"  return x",
		"}",
	)
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestScopedIslandStructLiteralEscape(t *testing.T) {
	src := tetraSource(
		"struct Box { buf: []u8 }",
		"fun main(): i32 {",
		"  var box: Box = Box{ buf: make_u8(1) }",
		"  island(16) as isl {",
		"    box = Box{ buf: core.island_make_u8(isl, 4) }",
		"  }",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestScopedIslandDanglingFieldAccess(t *testing.T) {
	src := tetraSource(
		"struct Box { buf: []u8 }",
		"fun main(): i32 {",
		"  var box: Box = Box{ buf: make_u8(1) }",
		"  island(16) as isl {",
		"    box.buf = core.island_make_u8(isl, 4)",
		"  }",
		"  return box.buf[0]",
		"}",
	)
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestUnsafeAllowsAllocBytes(t *testing.T) {
	src := tetraSource(
		"fun main(): i32 uses alloc, mem {",
		"  unsafe {",
		"    let p: ptr = core.alloc_bytes(4)",
		"  }",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestAllocBytesRequiresUnsafe(t *testing.T) {
	src := tetraSource(
		"fun main(): i32 {",
		"  let p: ptr = core.alloc_bytes(4)",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestIslandNewRequiresUnsafe(t *testing.T) {
	src := tetraSource(
		"fun main(): i32 {",
		"  let isl: island = core.island_new(16)",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestManualFreeRequiresUnsafe(t *testing.T) {
	src := tetraSource(
		"fun main(): i32 {",
		"  island(16) as isl {",
		"    free(isl)",
		"  }",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestManualFreeAllowedInUnsafe(t *testing.T) {
	src := tetraSource(
		"fun main(): i32 uses alloc, islands, mem {",
		"  unsafe {",
		"    let isl: island = core.island_new(16)",
		"    free(isl)",
		"  }",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestIslandMakeRequiresUnsafeWithoutRegion(t *testing.T) {
	src := tetraSource(
		"fun main(): i32 {",
		"  unsafe {",
		"    let isl: island = core.island_new(16)",
		"  }",
		"  var buf: []u8 = core.island_make_u8(isl, 4)",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestScopedIslandHandleReturnEscape(t *testing.T) {
	src := tetraSource(
		"fun make(): island {",
		"  island(16) as isl {",
		"    return isl",
		"  }",
		"  unsafe {",
		"    return core.island_new(1)",
		"  }",
		"}",
	)
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestScopedIslandHandleStructEscape(t *testing.T) {
	src := tetraSource(
		"struct Box { isl: island }",
		"fun main(): i32 {",
		"  unsafe {",
		"    var box: Box = Box{ isl: core.island_new(1) }",
		"    island(16) as isl {",
		"      box = Box{ isl: isl }",
		"    }",
		"  }",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestScopedIslandHelperRegionOk(t *testing.T) {
	src := tetraSource(
		"fun make_buf(isl: island, n: i32): []u8 uses alloc, islands, mem {",
		"  var buf: []u8 = core.island_make_u8(isl, n)",
		"  return buf",
		"}",
		"fun main(): i32 uses alloc, islands, mem {",
		"  island(64) as isl {",
		"    var out: []u8 = make_buf(isl, 4)",
		"    out[0] = 1",
		"  }",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestScopedIslandHelperChainRegionOk(t *testing.T) {
	src := tetraSource(
		"fun make_buf1(isl: island, n: i32): []u8 uses alloc, islands, mem {",
		"  return core.island_make_u8(isl, n)",
		"}",
		"fun make_buf2(isl: island, n: i32): []u8 uses alloc, islands, mem {",
		"  return make_buf1(isl, n)",
		"}",
		"fun main(): i32 uses alloc, islands, mem {",
		"  island(64) as isl {",
		"    var out: []u8 = make_buf2(isl, 4)",
		"    out[0] = 1",
		"  }",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestScopedIslandHelperReturnsStructWithSliceRegionOk(t *testing.T) {
	src := tetraSource(
		"struct Box { buf: []u8 }",
		"fun make_buf(isl: island, n: i32): []u8 uses alloc, islands, mem {",
		"  return core.island_make_u8(isl, n)",
		"}",
		"fun make_box(isl: island, n: i32): Box uses alloc, islands, mem {",
		"  return Box{ buf: make_buf(isl, n) }",
		"}",
		"fun main(): i32 uses alloc, islands, mem {",
		"  island(64) as isl {",
		"    let b: Box = make_box(isl, 4)",
		"    let v: u8 = b.buf[0]",
		"  }",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestScopedIslandHelperReturnsStructWithTwoSliceRegionsOk(t *testing.T) {
	src := tetraSource(
		"struct PairBuf { left: []u8, right: []u8 }",
		"fun make_pair(a: island, b: island): PairBuf uses alloc, islands, mem {",
		"  return PairBuf{",
		"    left: core.island_make_u8(a, 1),",
		"    right: core.island_make_u8(b, 1)",
		"  }",
		"}",
		"fun main(): i32 uses alloc, islands, mem {",
		"  island(64) as a {",
		"    island(64) as b {",
		"      let pair: PairBuf = make_pair(a, b)",
		"      let x: u8 = pair.left[0]",
		"      let y: u8 = pair.right[0]",
		"    }",
		"  }",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestScopedIslandHelperRejectsStructWithTwoSliceRegionsEscape(t *testing.T) {
	src := tetraSource(
		"struct PairBuf { left: []u8, right: []u8 }",
		"fun make_pair(a: island, b: island): PairBuf uses alloc, islands, mem {",
		"  return PairBuf{",
		"    left: core.island_make_u8(a, 1),",
		"    right: core.island_make_u8(b, 1)",
		"  }",
		"}",
		"fun main(): i32 uses alloc, islands, mem {",
		"  var pair: PairBuf = PairBuf{ left: make_u8(1), right: make_u8(1) }",
		"  island(64) as a {",
		"    island(64) as b {",
		"      pair = make_pair(a, b)",
		"    }",
		"  }",
		"  return pair.left[0]",
		"}",
	)
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestScopedIslandHelperEscape(t *testing.T) {
	src := tetraSource(
		"fun make_buf(isl: island, n: i32): []u8 {",
		"  var buf: []u8 = core.island_make_u8(isl, n)",
		"  return buf",
		"}",
		"fun main(): i32 {",
		"  var out: []u8 = make_u8(1)",
		"  island(64) as isl {",
		"    out = make_buf(isl, 4)",
		"  }",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestCapabilitiesRequireUnsafe(t *testing.T) {
	src := tetraSource(
		"fun main(): i32 {",
		"  let io: cap.io = core.cap_io()",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestCapabilitiesAllowMmioInUnsafe(t *testing.T) {
	src := tetraSource(
		"fun main(): i32 uses alloc, capability, io, mem, mmio {",
		"  unsafe {",
		"    let io: cap.io = core.cap_io()",
		"    let mem: cap.mem = core.cap_mem()",
		"    let p: ptr = core.alloc_bytes(4)",
		"    let v: i32 = core.mmio_read_i32(p, io)",
		"    let w: i32 = core.mmio_write_i32(p, v, io)",
		"  }",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestCapabilitiesTypeMismatch(t *testing.T) {
	src := tetraSource(
		"fun main(): i32 {",
		"  unsafe {",
		"    let p: ptr = core.alloc_bytes(4)",
		"    let v: i32 = core.mmio_read_i32(p, p)",
		"  }",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestCapMemAllowsLoadStoreInUnsafe(t *testing.T) {
	src := tetraSource(
		"fun main(): i32 uses alloc, capability, mem {",
		"  unsafe {",
		"    let mem: cap.mem = core.cap_mem()",
		"    let p: ptr = core.alloc_bytes(8)",
		"    let q: ptr = core.ptr_add(p, 4, mem)",
		"    let _: i32 = core.store_i32(q, 123, mem)",
		"    let v: i32 = core.load_i32(q, mem)",
		"  }",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestCapMemBuiltinsRequireUnsafe(t *testing.T) {
	src := tetraSource(
		"fun main(): i32 {",
		"  let mem: cap.mem = core.cap_mem()",
		"  let p: ptr = core.alloc_bytes(8)",
		"  let v: i32 = core.load_i32(p, mem)",
		"  return v",
		"}",
	)
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestCapMemTypeMismatch(t *testing.T) {
	src := tetraSource(
		"fun main(): i32 {",
		"  unsafe {",
		"    let io: cap.io = core.cap_io()",
		"    let p: ptr = core.alloc_bytes(8)",
		"    let v: i32 = core.load_i32(p, io)",
		"  }",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestCapMemAllowsLoadStoreU8InUnsafe(t *testing.T) {
	src := tetraSource(
		"fun main(): i32 uses alloc, capability, mem {",
		"  unsafe {",
		"    let mem: cap.mem = core.cap_mem()",
		"    let p: ptr = core.alloc_bytes(1)",
		"    let _: u8 = core.store_u8(p, 7, mem)",
		"    let v: u8 = core.load_u8(p, mem)",
		"  }",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestCapMemU8TypeMismatch(t *testing.T) {
	src := tetraSource(
		"fun main(): i32 {",
		"  unsafe {",
		"    let io: cap.io = core.cap_io()",
		"    let p: ptr = core.alloc_bytes(1)",
		"    let v: u8 = core.load_u8(p, io)",
		"  }",
		"  return 0",
		"}",
	)
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestU8WorksInBinaryOps(t *testing.T) {
	src := tetraSource(
		"fun main(): i32 {",
		"  let x: u8 = 1",
		"  let y: i32 = x + 1",
		"  if (y == 2) {",
		"    return 0",
		"  }",
		"  return 1",
		"}",
	)
	if err := testkit.CheckProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestRegionAmbiguousControlFlowMessage(t *testing.T) {
	src := tetraSource(
		"fun pick(a: island, b: island, cond: i32): []u8 uses alloc, islands, mem {",
		"  var out: []u8 = core.island_make_u8(a, 1)",
		"  if (cond) {",
		"    out = core.island_make_u8(a, 1)",
		"  } else {",
		"    out = core.island_make_u8(b, 1)",
		"  }",
		"  return out",
		"}",
		"fun main(): i32 {",
		"  return 0",
		"}",
	)
	err := testkit.CheckProgram(src)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "control-flow merge") {
		t.Fatalf("expected control-flow merge hint, got: %v", err)
	}
	if !strings.Contains(err.Error(), "then:") || !strings.Contains(err.Error(), "else:") {
		t.Fatalf("expected then/else region details, got: %v", err)
	}
	if !strings.Contains(err.Error(), "param#0(a)") ||
		!strings.Contains(err.Error(), "param#1(b)") {
		t.Fatalf("expected parameter region details, got: %v", err)
	}
}

func tetraSource(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}
