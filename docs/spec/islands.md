# Islands Memory Model

> **Version:** v1 checked MVP
> **Status:** Specification  
> **Introduced:** Tetra v0.12

---

## 1. Overview

**Islands** are Tetra's primary memory management abstraction. An island is a contiguous region of memory that acts as an arena/bump allocator. All allocations within an island are fast (pointer bump), and the entire island is freed as a single unit.

Target boundary (current profile):
- Island runtime paths are in the current native runtime scope.
- WASM targets (`wasm32-wasi`, `wasm32-web`) support island IR in a
  compile-compatible fallback mode: `island_new` returns a handle token from the
  linear heap allocator, `island_make_*` maps to linear heap slice allocation,
  and `island_free` is currently a no-op.

### Key Properties

1. **Fast Allocation**: O(1) bump allocation within an island
2. **Bulk Deallocation**: The entire island is freed at once via `free(island)`
3. **No Fragmentation**: Memory is reclaimed only when the whole island is freed
4. **Explicit Lifetime**: The programmer controls when an island is created and destroyed

### Safety Guarantees (MVP)

- Attempting to allocate more than the island's remaining capacity causes a controlled program exit (exit code 1)
- **Double-free is undefined behavior** by default (the memory is already deallocated, so accessing the header is invalid)
- With `--islands-debug`, double-free is detected and the data pages are protected to catch use-after-free

### Release Evidence Boundary

Current Memory/Islands release claims require `validate-island-proof`,
`--islands-debug` sanitizer smoke, `island-proof-fuzz-summary` mutation
evidence, and the integrated
`memory-islands-surface-production-gate.sh` manifest/hash path documented in
`docs/release/memory_islands_surface_scope.md`. Producer-only proof rows,
missing verifier artifacts, missing sanitizer smoke, or missing proof-fuzz
evidence are not release proof.

---

## 2. Type System

The `island` type is a primitive type in Tetra with `SlotCount = 1` (8 bytes on x64).

Creating islands directly is unsafe; prefer scoped islands for safe code.

```tetra pseudocode
unsafe {
    let isl: island = core.island_new(4096)
}
```

```tetra pseudocode
island(4096) as isl {
    // ...
}
```

An `island` value is an opaque handle pointing to the island's base address.

---

## 3. Runtime Layout (ABI)

Each island is a contiguous memory region with the following header at offset 0:

```
+--------+--------+--------+--------+
| bump   | end    | total  | flags  |
| (4B)   | (4B)   | (4B)   | (4B)   |
+--------+--------+--------+--------+
|         user data...              |
+-----------------------------------+
```

### Header Fields

| Offset | Size | Name    | Description                                      |
|--------|------|---------|--------------------------------------------------|
| 0      | 4    | `bump`  | Current allocation offset from base (starts at HEADER_SIZE) |
| 4      | 4    | `end`   | Maximum offset (= total bytes allocated from OS) |
| 8      | 4    | `total` | Total bytes for deallocation (passed to munmap/VirtualFree) |
| 12     | 4    | `flags` | Reserved; bit 0 = freed (for double-free detection) |

**HEADER_SIZE = 16 bytes**

User data starts at offset 16.

---

## 4. Builtin Functions

### 4.1 `core.island_new(size: i32) -> island`

Allocates a new island with capacity for `size` bytes of user data.

**Semantics:**
- Actual allocation = `size + HEADER_SIZE`
- Uses OS primitives:
  - **Linux/macOS**: `mmap(NULL, total, PROT_READ|PROT_WRITE, MAP_PRIVATE|MAP_ANONYMOUS, -1, 0)`
  - **Windows**: `VirtualAlloc(NULL, total, MEM_COMMIT|MEM_RESERVE, PAGE_READWRITE)`
- Initializes header: `bump = 16`, `end = total`, `total = total`, `flags = 0`
- Returns base pointer as `island`
- Only allowed inside `unsafe` blocks

**Example (unsafe):**
```tetra pseudocode
unsafe {
    let isl: island = core.island_new(1024)
}
```

### 4.2 `core.island_make_u8(isl: island, len: i32) -> []u8`

Allocates a `[]u8` slice of `len` bytes within the given island.

**Semantics:**
- Reads `bump` and `end` from island header
- Required bytes = `len`
- If `bump + len > end`: exit(1) — allocation failure
- Returns slice `{ptr: base + old_bump, len: len}`
- Updates `bump = bump + len`
- In safe code, `isl` must be a tracked region (scoped island or a region-carrying parameter); otherwise this call requires `unsafe`.

**Example:**
```tetra pseudocode
var buf: []u8 = core.island_make_u8(isl, 64)
```

### 4.3 `core.island_make_i32(isl: island, len: i32) -> []i32`

Allocates a `[]i32` slice of `len` elements within the given island.

**Semantics:**
- Same as `island_make_u8`, but required bytes = `len * 4`
- In safe code, `isl` must be a tracked region (scoped island or a region-carrying parameter); otherwise this call requires `unsafe`.

**Example:**
```tetra pseudocode
var arr: []i32 = core.island_make_i32(isl, 100)
```

### 4.4 `core.island_make_u16(isl: island, len: i32) -> []u16`

Allocates a `[]u16` slice of `len` elements within the given island.

**Semantics:**
- Same as `island_make_u8`, but required bytes = `len * 2`
- In safe code, `isl` must be a tracked region (scoped island or a region-carrying parameter); otherwise this call requires `unsafe`.

**Example:**
```tetra pseudocode
var half: []u16 = core.island_make_u16(isl, 32)
```

### 4.5 `core.island_make_bool(isl: island, len: i32) -> []bool`

Allocates a `[]bool` slice of `len` elements within the given island.

**Semantics:**
- Same region/ownership safety contract as `island_make_u8`
- Current MVP lowering uses the same i32-width allocation layout as `[]i32`
  (required bytes = `len * 4`)
- In safe code, `isl` must be a tracked region (scoped island or a region-carrying parameter); otherwise this call requires `unsafe`.

**Example:**
```tetra pseudocode
var flags: []bool = core.island_make_bool(isl, 32)
```

---

## 5. Scoped Islands (Auto-free)

Scoped islands provide automatic cleanup for most cases.
Scoped islands remain safe when their handle is tracked by the region checker.

```tetra pseudocode
island(4096) as isl {
    var buf: []u8 = core.island_make_u8(isl, 64)
    // use buf
} // free(isl) is injected automatically
```

**Semantics:**
- `island(size) as name { ... }` allocates a new island and binds it to `name` for the block.
- `free(name)` is injected on normal block exit and on early `return`.
- Locals declared inside the block are out of scope after the block ends.

---

## 6. Region Typing (MVP for Scoped Islands)

Region typing prevents slices from scoped islands escaping their scope.

**Rules:**
- Slices created by `core.island_make_*` inside a scoped island cannot be:
  - returned from the function,
  - assigned to a variable in an outer scope.
- `island` handles bound by `island(size) as ...` also cannot escape their scope.
- Region info propagates through struct literals and field access for fields that can contain regions (slices, islands, or structs/arrays containing them).
- Region info propagates through function calls when the return is tied to a single region-carrying parameter.
- After `if`/`while`, a variable assigned from different regions becomes ambiguous and must be reassigned before use (the compiler reports an “ambiguous region after control-flow merge” error).

**Known limitations (MVP):**
- Interprocedural propagation is limited to returns derived from a single region-carrying parameter; mixing regions is rejected.
- Region tracking is still conservative across complex control flow and does not use SSA/phi-based inference.

---

## 7. The `free` Statement

```tetra pseudocode
free(<expr>)
```

Frees the entire island. The expression must evaluate to type `island`.
Manual `free` is only allowed inside `unsafe` blocks. Scoped islands inject an implicit free on exit.

**Semantics:**
1. Read `total` from header at offset 8
2. Call OS deallocation:
   - **Linux/macOS**: `munmap(base, total)`
   - **Windows**: `VirtualFree(base, 0, MEM_RELEASE)`

> **Warning:** Double-free is **undefined behavior** by default. After `free(isl)`, the island's memory is returned to the OS and any access (including another `free`) may crash or corrupt memory.
>
> **Debug mode (`--islands-debug`):** The runtime keeps the mapping, sets `flags |= 1`, and protects data pages. A second `free` triggers a controlled exit (exit code 2), and use-after-free faults on protected pages.

**Example (unsafe):**
```tetra pseudocode
unsafe {
    let isl: island = core.island_new(1024)
    var buf: []u8 = core.island_make_u8(isl, 64)
    // ... use buf ...
    free(isl)
}
```

---

## 8. Example Programs

### 6.1 Hello World with Island

```tetra
fun main(): i32 {
    island(64) as isl {
        var msg: []u8 = core.island_make_u8(isl, 6)
        msg[0] = 72   // 'H'
        msg[1] = 101  // 'e'
        msg[2] = 108  // 'l'
        msg[3] = 108  // 'l'
        msg[4] = 111  // 'o'
        msg[5] = 10   // '\n'
        print(msg)
    }
    return 0
}
```

### 6.2 Sum of Array

```tetra
fun main(): i32 {
    var sum: i32 = 0
    island(4096) as isl {
        let n: i32 = 10
        var arr: []i32 = core.island_make_i32(isl, n)
        
        var i: i32 = 0
        while (i < n) {
            arr[i] = i + 1
            i = i + 1
        }
        
        i = 0
        while (i < n) {
            sum = sum + arr[i]
            i = i + 1
        }
    }
    return sum  // Expected: 55
}
```

---

## 9. Future Extensions

1. **Nested Islands**: Islands that are themselves allocated from parent islands
2. **Actor-Owned Islands**: Each actor has a default island for its private state

---

## 10. Platform Notes

### Linux (x86-64)
- Uses syscall 9 (`mmap`) for allocation
- Uses syscall 11 (`munmap`) for deallocation

### macOS (x86-64)
- Uses syscall 0x2000000 + 197 (`mmap`) for allocation
- Uses syscall 0x2000000 + 73 (`munmap`) for deallocation

### Windows (x64)
- Uses `kernel32.VirtualAlloc` for allocation
- Uses `kernel32.VirtualFree` for deallocation
- Requires adding `VirtualFree` to PE import table

## 11. Epic 06 coverage

Island and region coverage is release-blocking in the focused safety slice:

```sh
go test ./compiler/... -run "Effect|Uses|Capability|Unsafe|Ownership|Borrow|Consume|Inout|Island|Region|Privacy|Budget" -count=1
```

The slice checks safe scoped allocation, helper functions whose return region is
tied to a single island parameter, diagnostics for scoped slices or island
handles escaping, ambiguous control-flow region merges, and runtime examples for
overflow and debug double-free behavior.
