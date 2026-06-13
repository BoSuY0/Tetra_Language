package main

const memoryPositiveSource = `
import lib.core.memory as memory

func main() -> Int
uses alloc, capability, islands, mem:
    island(64) as isl:
        var xs: []u8 = core.island_make_u8(isl, 1)
        xs[0] = 7
        if xs[0] != 7:
            return 1
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let src: ptr = core.alloc_bytes(4)
        let dst: ptr = core.alloc_bytes(4)
        let clear_status: Int = memory.memset_u8(dst, 0, 4, mem)
        let seed_status: Int = memory.memset_u8(src, 42, 4, mem)
        let copy_status: Int = memory.memcpy_u8(dst, src, 4, mem)
        if clear_status == 0:
            if seed_status == 0:
                if copy_status == 0:
                    if core.load_u8(dst, mem) == 42:
                        return 0
        return 1
    return 1
`

const memoryStressSource = `
import lib.core.memory as memory

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let src: ptr = core.alloc_bytes(64)
        let dst: ptr = core.alloc_bytes(64)
        var i: Int = 0
        while i < 32:
            let seed_status: Int = memory.memset_u8(src, 7, 64, mem)
            let clear_status: Int = memory.memset_u8(dst, 0, 64, mem)
            let copy_status: Int = memory.memcpy_u8(dst, src, 64, mem)
            if seed_status != 0:
                return 1
            if clear_status != 0:
                return 1
            if copy_status != 0:
                return 1
            let p: ptr = core.ptr_add(dst, i, mem)
            if core.load_u8(p, mem) != 7:
                return 1
            i = i + 1
        return 0
    return 1
`

const memoryFuzzSource = `
import lib.core.memory as memory

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let src: ptr = core.alloc_bytes(64)
        let dst: ptr = core.alloc_bytes(64)
        var n: Int = 1
        while n < 33:
            let seed_status: Int = memory.memset_u8(src, 17, n, mem)
            let clear_status: Int = memory.memset_u8(dst, 0, 64, mem)
            let copy_status: Int = memory.memcpy_u8(dst, src, n, mem)
            if seed_status != 0:
                return 1
            if clear_status != 0:
                return 1
            if copy_status != 0:
                return 1
            if core.load_u8(dst, mem) != 17:
                return 1
            let last: ptr = core.ptr_add(dst, n - 1, mem)
            if core.load_u8(last, mem) != 17:
                return 1
            let sentinel: ptr = core.ptr_add(dst, n, mem)
            if core.load_u8(sentinel, mem) != 0:
                return 1
            n = n + 1
        return 0
    return 1
`

const allocInvalidSizeSource = `
func main() -> Int
uses alloc, mem:
    unsafe:
        let _: ptr = core.alloc_bytes(0)
        return 0
    return 0
`

const sliceBoundsSource = `
func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[2] = 1
    return 0
`

const rawPtrAddNegativeSource = `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let q: ptr = core.ptr_add(p, 0 - 1, mem)
        let _: UInt8 = core.store_u8(q, 7, mem)
        return 0
    return 0
`

const rawPtrAddUpperSource = `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let q: ptr = core.ptr_add(p, 4, mem)
        let _: UInt8 = core.store_u8(q, 7, mem)
        return 0
    return 0
`

const rawI32WidthSource = `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let q: ptr = core.ptr_add(p, 5, mem)
        let _: Int = core.load_i32(q, mem)
        return 0
    return 0
`

const rawPtrWidthSource = `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let q: ptr = core.ptr_add(p, 1, mem)
        let _: ptr = core.store_ptr(q, p, mem)
        return 0
    return 0
`

const rawStoreI32WidthSource = `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let q: ptr = core.ptr_add(p, 5, mem)
        let _: Int = core.store_i32(q, 123, mem)
        return 0
    return 0
`

const rawLoadPtrWidthSource = `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let q: ptr = core.ptr_add(p, 1, mem)
        let _: ptr = core.load_ptr(q, mem)
        return 0
    return 0
`

const rawPointerBoundsMetadataSource = `
func external(raw: ptr, n: Int) -> Int
uses capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let q: ptr = core.ptr_add(raw, 0 - 1, mem)
        let xs: []u8 = core.raw_slice_u8_from_parts(q, n, mem)
        return xs.len
    return 0

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(24)
        let q: ptr = core.ptr_add(p, 8, mem)
        let _: UInt8 = core.store_u8(q, 7, mem)
        let value: UInt8 = core.load_u8(q, mem)
        let neg_base: ptr = core.alloc_bytes(8)
        let neg: ptr = core.ptr_add(neg_base, 0 - 1, mem)
        let upper_base: ptr = core.alloc_bytes(8)
        let upper: ptr = core.ptr_add(upper_base, 8, mem)
        let i32_base: ptr = core.alloc_bytes(8)
        let i32_ptr: ptr = core.ptr_add(i32_base, 5, mem)
        let i32_value: Int = core.load_i32(i32_ptr, mem)
        let ptr_base: ptr = core.alloc_bytes(4)
        let ptr_ptr: ptr = core.ptr_add(ptr_base, 1, mem)
        let ptr_value: ptr = core.store_ptr(ptr_ptr, ptr_base, mem)
        let store_i32_base: ptr = core.alloc_bytes(8)
        let store_i32_ptr: ptr = core.ptr_add(store_i32_base, 5, mem)
        let store_i32_status: Int = core.store_i32(store_i32_ptr, 123, mem)
        let load_ptr_base: ptr = core.alloc_bytes(4)
        let load_ptr_ptr: ptr = core.ptr_add(load_ptr_base, 1, mem)
        let load_ptr_value: ptr = core.load_ptr(load_ptr_ptr, mem)
        let raw_slice_base: ptr = core.alloc_bytes(16)
        let raw_slice_view: []u8 = core.raw_slice_u8_from_parts(raw_slice_base, 8, mem)
        let raw_slice_negative: []u8 = core.raw_slice_u8_from_parts(raw_slice_base, 0 - 1, mem)
        let raw_slice_overflow: []i32 = core.raw_slice_i32_from_parts(raw_slice_base, 536870912, mem)
        return i32_value + store_i32_status + raw_slice_view.len + raw_slice_negative.len + raw_slice_overflow.len
    return 0
`

const allocationLengthContractReportSource = `
func main() -> Int
uses alloc, islands, mem:
    unsafe:
        let raw_zero: ptr = core.alloc_bytes(0)
    var make_zero: []u8 = make_u8(0)
    var make_negative: []u16 = make_u16(0 - 1)
    var make_overflow: []i32 = make_i32(536870912)
    island(64) as isl:
        var island_zero: []u8 = core.island_make_u8(isl, 0)
        var island_negative: []u16 = core.island_make_u16(isl, 0 - 1)
        var island_overflow: []i32 = core.island_make_i32(isl, 536870912)
        return make_zero.len + make_negative.len + make_overflow.len + island_zero.len + island_negative.len + island_overflow.len
    return 0
`

const rawSliceNegativeLengthSource = `
func main() -> Int
uses capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = 0
        let xs: []u8 = core.raw_slice_u8_from_parts(p, 0 - 1, mem)
        return xs.len + 98
    return 0
`

const rawSliceI32LengthOverflowSource = `
func main() -> Int
uses capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = 0
        let xs: []i32 = core.raw_slice_i32_from_parts(p, 536870912, mem)
        return xs.len + 98
    return 0
`

const memoryNegativeLengthSource = `
import lib.core.memory as memory

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let src: ptr = core.alloc_bytes(4)
        let dst: ptr = core.alloc_bytes(4)
        let memset_status: Int = memory.memset_u8(dst, 0, 0 - 1, mem)
        let memcpy_status: Int = memory.memcpy_u8(dst, src, 0 - 1, mem)
        if memset_status == 2:
            if memcpy_status == 2:
                return 2
        return 1
    return 1
`
