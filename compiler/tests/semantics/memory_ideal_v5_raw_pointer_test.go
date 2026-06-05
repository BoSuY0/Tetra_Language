package compiler_test

import (
	"testing"

	"tetra_language/compiler/internal/testkit"
)

func TestMemoryIdealV5AllocBytesRootPtrAddInBoundsTypeChecks(t *testing.T) {
	testkit.RequireFileCheckOK(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let q: ptr = core.ptr_add(p, 4, mem)
        let _: Int = core.store_i32(q, 42, mem)
        return core.load_i32(q, mem)
    return 0
`)
}

func TestMemoryIdealV5RawSliceFromPartsUnsafeGatewayTypeChecks(t *testing.T) {
	testkit.RequireFileCheckOK(t, `
func main() -> Int
uses alloc, capability, mem:
    var xs: []u8 = make_u8(4)
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let ys: []u8 = core.raw_slice_u8_from_parts(xs.ptr, xs.len, mem)
        return ys.len
    return 0
`)
}

func TestMemoryIdealV5RawSliceFromPartsOutsideUnsafeRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func forge(p: ptr, n: Int, mem: cap.mem) -> []u8
uses capability, mem:
    return core.raw_slice_u8_from_parts(p, n, mem)
`, "'core.raw_slice_u8_from_parts' is only allowed in unsafe blocks")
}
