# Tetra Language (v0.13)

A systems programming language with region-based memory management (Islands).

## Build

```
bash scripts/bootstrap.sh
```

## Tests

```
bash scripts/test.sh
```

## Smoke

```
./tetra smoke
```

## Project Dump (for agents)

Creates a single text file with a curated view of the repository (by default excludes caches/artifacts and focuses on source+docs):

```
go run ./tools/cmd/dump-project
```

Useful flags:
- `--all` (dump everything, still excluding common caches/artifacts)
- `--only <prefix>` (repeatable)
- `--exclude-prefix <prefix>` (repeatable)

## Usage

```
./tetra build --target linux-x64 -o app examples/hello.tetra
./app
```

Run (build + execute):

```
./tetra run --target linux-x64 examples/hello.tetra
```

Windows:

```
./tetra build --target windows-x64 -o app.exe examples/hello.tetra
```

macOS:

```
./tetra build --target macos-x64 -o app examples/hello.tetra
```

If no input file is provided, `tetra` looks for `main.tetra` in the current directory.

## Language Features

### Basic Syntax

- Multiple functions: `fun main(): i32 { ... }` (legacy `fn main() -> i32` is still accepted)
- `main` must exist and has no parameters
- Statements:
  - `print("...");` (string literal or `[]u8`)
  - `var x: i32 = <expr>` (mutable)
  - `val x: i32 = <expr>` (immutable)
  - `let x: i32 = <expr>` (legacy alias of `var`)
  - `x = <expr>`
  - `if (<expr>) { ... } else { ... }`
  - `while (<expr>) { ... }`
  - `island(<size>) as <name> { ... }` (scoped island, auto-free)
  - `unsafe { ... }` (enables unsafe operations)
  - `free(<island>)` (allowed only in `unsafe` blocks)
  - `;` (empty statement / no-op)
  - `return <expr>`
- Expressions: integer literals, identifiers, calls `foo(1, 2)`, `+`, `-`, `==`, `<`, unary `-`, and parentheses
- Semicolons are optional after statements
- Every function must end with `return`
- Calls support `i32` arguments (first 6 in registers, 7+ on the stack)

### Types

- `i32` - 32-bit signed integer
- `u8` - 8-bit unsigned byte (currently treated as “int32-like” in expressions; implicitly compatible with `i32`)
- `ptr` - raw pointer
- `str` - string literal (ptr + len)
- `[]u8`, `[]i32` - slices
- `island` - (**NEW**) region memory handle
- User-defined structs

### Islands Memory Model

Islands are Tetra's region-based memory management system. An island is a contiguous memory region with bump allocation.

**Builtin functions (unsafe):**

```tetra
unsafe {
    // Create a new island with capacity for `size` bytes
    let isl: island = core.island_new(1024)

    // Allocate a []u8 slice from the island
    var buf: []u8 = core.island_make_u8(isl, 64)

    // Allocate a []i32 slice from the island
    var arr: []i32 = core.island_make_i32(isl, 100)

    // Free the entire island
    free(isl)
}
```

**Scoped islands (auto-free):**

```tetra
island(1024) as isl {
    var buf: []u8 = core.island_make_u8(isl, 64)
    // use buf
}
```

**Properties:**
- O(1) bump allocation
- Bulk deallocation (entire island freed at once)
- Overflow protection (exit code 1 if allocation exceeds capacity)

See `docs/spec/islands.md` for the full specification.

## Notes

- Targets supported: `linux-x64`, `windows-x64`, `macos-x64`.
- Unsafe/capability model: see `docs/spec/unsafe.md` and `docs/spec/capabilities.md`.
- Build flag: `--islands-debug` (double-free detection and UAF traps for islands).
- Linux output is a native ELF file without a custom extension.
  - Default output name is `app` (use `-o` to override).
- Windows output is a PE32+ `.exe`.
  - Default output name is `app.exe`.
- macOS output is a Mach-O 64-bit executable.
- Windows PE uses `.text`, `.rdata`, `.idata`, `.reloc` sections.
- WinAPI imports are referenced as `dll.Symbol` (for example: `kernel32.ExitProcess`).
- Linux calls follow System V x86_64: first 6 params in registers, remaining params on the stack.

## Verification (Linux)

```
go test ./compiler/...
go test ./cli/...
go test ./tools/...
```

Quick smoke (builds examples; runs them when target matches host):

```
./tetra smoke --target linux-x64
```

Version:

```
./tetra version
```

Manual checks:

```
./tetra build --target linux-x64 -o app examples/hello.tetra
file ./app
./app
echo $?
readelf -h ./app
```

Maintenance:

```
./tetra clean
```

## Examples

### Hello with Islands

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

### Sum of Array

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
    return sum  // Returns 55
}
```
