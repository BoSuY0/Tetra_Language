package compiler

import (
	"strings"
	"testing"
)

func checkProgram(src string) error {
	prog, err := Parse([]byte(src))
	if err != nil {
		return err
	}
	_, err = Check(prog)
	return err
}

func TestScopedIslandOk(t *testing.T) {
	src := "fun main(): i32 {\n  island(64) as isl {\n    var xs: []u8 = core.island_make_u8(isl, 4)\n    xs[0] = 1\n  }\n  return 0\n}\n"
	if err := checkProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestScopedIslandReturnEscape(t *testing.T) {
	src := "fun make(): []u8 {\n  island(16) as isl {\n    var xs: []u8 = core.island_make_u8(isl, 4)\n    return xs\n  }\n  return make_u8(1)\n}\n"
	if err := checkProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestScopedIslandAssignOuterEscape(t *testing.T) {
	src := "fun main(): i32 {\n  var out: []u8 = make_u8(1)\n  island(16) as isl {\n    out = core.island_make_u8(isl, 4)\n  }\n  return 0\n}\n"
	if err := checkProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestScopedIslandLocalScope(t *testing.T) {
	src := "fun main(): i32 {\n  island(16) as isl {\n    let x: i32 = 1\n  }\n  return x\n}\n"
	if err := checkProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestScopedIslandStructLiteralEscape(t *testing.T) {
	src := "struct Box { buf: []u8 }\nfun main(): i32 {\n  var box: Box = Box{ buf: make_u8(1) }\n  island(16) as isl {\n    box = Box{ buf: core.island_make_u8(isl, 4) }\n  }\n  return 0\n}\n"
	if err := checkProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestScopedIslandDanglingFieldAccess(t *testing.T) {
	src := "struct Box { buf: []u8 }\nfun main(): i32 {\n  var box: Box = Box{ buf: make_u8(1) }\n  island(16) as isl {\n    box.buf = core.island_make_u8(isl, 4)\n  }\n  return box.buf[0]\n}\n"
	if err := checkProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestUnsafeAllowsAllocBytes(t *testing.T) {
	src := "fun main(): i32 {\n  unsafe {\n    let p: ptr = core.alloc_bytes(4)\n  }\n  return 0\n}\n"
	if err := checkProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestAllocBytesRequiresUnsafe(t *testing.T) {
	src := "fun main(): i32 {\n  let p: ptr = core.alloc_bytes(4)\n  return 0\n}\n"
	if err := checkProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestIslandNewRequiresUnsafe(t *testing.T) {
	src := "fun main(): i32 {\n  let isl: island = core.island_new(16)\n  return 0\n}\n"
	if err := checkProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestManualFreeRequiresUnsafe(t *testing.T) {
	src := "fun main(): i32 {\n  island(16) as isl {\n    free(isl)\n  }\n  return 0\n}\n"
	if err := checkProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestManualFreeAllowedInUnsafe(t *testing.T) {
	src := "fun main(): i32 {\n  unsafe {\n    let isl: island = core.island_new(16)\n    free(isl)\n  }\n  return 0\n}\n"
	if err := checkProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestIslandMakeRequiresUnsafeWithoutRegion(t *testing.T) {
	src := "fun main(): i32 {\n  unsafe {\n    let isl: island = core.island_new(16)\n  }\n  var buf: []u8 = core.island_make_u8(isl, 4)\n  return 0\n}\n"
	if err := checkProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestScopedIslandHandleReturnEscape(t *testing.T) {
	src := "fun make(): island {\n  island(16) as isl {\n    return isl\n  }\n  unsafe {\n    return core.island_new(1)\n  }\n}\n"
	if err := checkProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestScopedIslandHandleStructEscape(t *testing.T) {
	src := "struct Box { isl: island }\nfun main(): i32 {\n  unsafe {\n    var box: Box = Box{ isl: core.island_new(1) }\n    island(16) as isl {\n      box = Box{ isl: isl }\n    }\n  }\n  return 0\n}\n"
	if err := checkProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestScopedIslandHelperRegionOk(t *testing.T) {
	src := "fun make_buf(isl: island, n: i32): []u8 {\n  var buf: []u8 = core.island_make_u8(isl, n)\n  return buf\n}\nfun main(): i32 {\n  island(64) as isl {\n    var out: []u8 = make_buf(isl, 4)\n    out[0] = 1\n  }\n  return 0\n}\n"
	if err := checkProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestScopedIslandHelperChainRegionOk(t *testing.T) {
	src := "fun make_buf1(isl: island, n: i32): []u8 {\n  return core.island_make_u8(isl, n)\n}\nfun make_buf2(isl: island, n: i32): []u8 {\n  return make_buf1(isl, n)\n}\nfun main(): i32 {\n  island(64) as isl {\n    var out: []u8 = make_buf2(isl, 4)\n    out[0] = 1\n  }\n  return 0\n}\n"
	if err := checkProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestScopedIslandHelperReturnsStructWithSliceRegionOk(t *testing.T) {
	src := "struct Box { buf: []u8 }\nfun make_buf(isl: island, n: i32): []u8 {\n  return core.island_make_u8(isl, n)\n}\nfun make_box(isl: island, n: i32): Box {\n  return Box{ buf: make_buf(isl, n) }\n}\nfun main(): i32 {\n  island(64) as isl {\n    let b: Box = make_box(isl, 4)\n    let v: u8 = b.buf[0]\n  }\n  return 0\n}\n"
	if err := checkProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestScopedIslandHelperEscape(t *testing.T) {
	src := "fun make_buf(isl: island, n: i32): []u8 {\n  var buf: []u8 = core.island_make_u8(isl, n)\n  return buf\n}\nfun main(): i32 {\n  var out: []u8 = make_u8(1)\n  island(64) as isl {\n    out = make_buf(isl, 4)\n  }\n  return 0\n}\n"
	if err := checkProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestCapabilitiesRequireUnsafe(t *testing.T) {
	src := "fun main(): i32 {\n  let io: cap.io = core.cap_io()\n  return 0\n}\n"
	if err := checkProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestCapabilitiesAllowMmioInUnsafe(t *testing.T) {
	src := "fun main(): i32 {\n  unsafe {\n    let io: cap.io = core.cap_io()\n    let mem: cap.mem = core.cap_mem()\n    let p: ptr = core.alloc_bytes(4)\n    let v: i32 = core.mmio_read_i32(p, io)\n    let w: i32 = core.mmio_write_i32(p, v, io)\n  }\n  return 0\n}\n"
	if err := checkProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestCapabilitiesTypeMismatch(t *testing.T) {
	src := "fun main(): i32 {\n  unsafe {\n    let p: ptr = core.alloc_bytes(4)\n    let v: i32 = core.mmio_read_i32(p, p)\n  }\n  return 0\n}\n"
	if err := checkProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestCapMemAllowsLoadStoreInUnsafe(t *testing.T) {
	src := "fun main(): i32 {\n  unsafe {\n    let mem: cap.mem = core.cap_mem()\n    let p: ptr = core.alloc_bytes(8)\n    let q: ptr = core.ptr_add(p, 4, mem)\n    let _: i32 = core.store_i32(q, 123, mem)\n    let v: i32 = core.load_i32(q, mem)\n  }\n  return 0\n}\n"
	if err := checkProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestCapMemBuiltinsRequireUnsafe(t *testing.T) {
	src := "fun main(): i32 {\n  let mem: cap.mem = core.cap_mem()\n  let p: ptr = core.alloc_bytes(8)\n  let v: i32 = core.load_i32(p, mem)\n  return v\n}\n"
	if err := checkProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestCapMemTypeMismatch(t *testing.T) {
	src := "fun main(): i32 {\n  unsafe {\n    let io: cap.io = core.cap_io()\n    let p: ptr = core.alloc_bytes(8)\n    let v: i32 = core.load_i32(p, io)\n  }\n  return 0\n}\n"
	if err := checkProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestCapMemAllowsLoadStoreU8InUnsafe(t *testing.T) {
	src := "fun main(): i32 {\n  unsafe {\n    let mem: cap.mem = core.cap_mem()\n    let p: ptr = core.alloc_bytes(1)\n    let _: u8 = core.store_u8(p, 7, mem)\n    let v: u8 = core.load_u8(p, mem)\n  }\n  return 0\n}\n"
	if err := checkProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestCapMemU8TypeMismatch(t *testing.T) {
	src := "fun main(): i32 {\n  unsafe {\n    let io: cap.io = core.cap_io()\n    let p: ptr = core.alloc_bytes(1)\n    let v: u8 = core.load_u8(p, io)\n  }\n  return 0\n}\n"
	if err := checkProgram(src); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestU8WorksInBinaryOps(t *testing.T) {
	src := "fun main(): i32 {\n  let x: u8 = 1\n  let y: i32 = x + 1\n  if (y == 2) {\n    return 0\n  }\n  return 1\n}\n"
	if err := checkProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestRegionAmbiguousControlFlowMessage(t *testing.T) {
	src := "fun pick(a: island, b: island, cond: i32): []u8 {\n  var out: []u8 = core.island_make_u8(a, 1)\n  if (cond) {\n    out = core.island_make_u8(a, 1)\n  } else {\n    out = core.island_make_u8(b, 1)\n  }\n  return out\n}\nfun main(): i32 {\n  return 0\n}\n"
	err := checkProgram(src)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "control-flow merge") {
		t.Fatalf("expected control-flow merge hint, got: %v", err)
	}
	if !strings.Contains(err.Error(), "then:") || !strings.Contains(err.Error(), "else:") {
		t.Fatalf("expected then/else region details, got: %v", err)
	}
	if !strings.Contains(err.Error(), "param#0(a)") || !strings.Contains(err.Error(), "param#1(b)") {
		t.Fatalf("expected parameter region details, got: %v", err)
	}
}
