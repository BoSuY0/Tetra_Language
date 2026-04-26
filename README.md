# Tetra Language (v0.6 Usable Alpha)

A systems programming language with region-based memory management (Islands).

This repository is the working Tetra compiler/toolchain. It is not yet the full
future Tetra platform described by the final language concept; instead it is a
staged profile that grows through small, verifiable slices.

The current profile is **v0.6 Usable Alpha**: the v0.5 Integrated Alpha surface
with additional hardening for formatter coverage, LSP stdio, local Eco project
bundles, release gates, and docs. See `docs/roadmap_0_5_to_0_6.md`,
`docs/roadmap_0_6_x_stabilization.md`, `docs/release_notes_v0_6.md`, and
`docs/checklists/v0_6_release_gate.md` for the supported surface and gates.

The active long-range production plan is `docs/roadmap_0_6_to_1_0.md`.
`docs/checklists/v1_0_release_gate.md` and `scripts/release_v1_0_gate.sh`
track the eventual v1.0 release bar. The v1.0 gate intentionally fails on the
current v0.6 compiler until the Flow-only, ownership-safe, x64+WASM, UI, and
Eco requirements are implemented.

## Build

```
bash scripts/bootstrap.sh
```

Bootstrap writes both local entrypoints: `./tetra` and the short alias `./t`.

## Tests

```
bash scripts/test.sh
bash scripts/test_all.sh --quick
bash scripts/test_all.sh --full
bash scripts/test_all.sh --full --keep-going
bash scripts/test_all.sh --full --json-only
```

`scripts/test_all.sh` is the v0.6.x stabilization wrapper. It runs the quick or
full gate, writes per-step logs, and emits both `summary.md` and `summary.json`
under `reports/` by default. Each JSON step records its command, exit code,
status, duration, and log path. `--keep-going` records all selected failures
before exiting, and `--json-only` prints the summary JSON for CI/editor tooling.

## Smoke

```
./tetra smoke
```

With `--report <path>`, smoke writes JSON containing target/version metadata,
aggregate `total`/`passed`/`failed` counts, and per-case build/run results.

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

The short alias is equivalent after bootstrap:

```
./t check examples/flow_hello.tetra
```

Check without emitting an executable:

```
./tetra check examples/flow_hello.tetra
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

Useful flags:

- `--target linux-x64|windows-x64|macos-x64`
- `-o <path>`
- `--islands-debug`
- `--runtime auto|selfhost|builtin`
- `--runtime-object <path.tobj>`
- `--emit exe|object|library`
- `--link-object <path.tobj>` (repeatable)
- `--jobs <n>`
- `--diagnostics text|json` on build/run/fmt/test paths

Runtime and library object notes:

```
./tetra build --runtime=selfhost -o actors examples/actors_pingpong.tetra
./tetra build --runtime=builtin -o actors_builtin examples/actors_pingpong.tetra
./tetra build --emit=library -o lib.tobj lib.tetra
./tetra build --link-object lib.tobj -o app app.tetra
```

`--runtime=auto` selects the embedded self-host actors runtime when actor
builtins are used. `--runtime-object` must point to a target-matching runtime
object that exports the required `__tetra_*` actor symbols. `--link-object` may
be repeated for additional target-matching TOBJ libraries.

Developer tooling alpha:

```
./tetra fmt examples/flow_hello.tetra
./tetra fmt --check examples/flow_hello.tetra
./tetra fmt --write examples/flow_hello.tetra
./tetra fmt --check examples lib __rt compiler/selfhostrt
go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt
./tetra targets
./tetra targets --format=json
./tetra doctor
./tetra doctor --format=json
./t version
./tetra check examples/flow_hello.tetra
./tetra smoke --list
./tetra smoke --list --format=json
./tetra test examples
./tetra test --report=json examples
./tetra build --diagnostics=json examples/hello.tetra
./tetra doc examples
./tetra doc -o docs/api.md examples
./tetra lsp --stdio-smoke examples/flow_hello.tetra
./tetra lsp --stdio
go run ./tools/cmd/gen-docs examples
```

`tetra fmt` emits canonical Flow-style formatting for the supported profile.
In v0.6, all `examples`, `lib`, `__rt`, and `compiler/selfhostrt` sources are
part of formatter release coverage, and the stabilization gates also scan those
trees for accidental legacy syntax drift. `tetra test` discovers top-level
`test "name":` blocks and runs them on the matching host target.
`--report=json` emits a structured per-test report for editor/CI tooling with
aggregate totals, per-file totals, per-test results, and `duration_ms` timing
fields.

`--diagnostics` strictly accepts `text` or `json`. `--diagnostics=json` emits
one diagnostic object per failing command path with `code`, `message`, `file`,
`line`, `column`, `severity`, and optional `hint`. Parser/frontend diagnostics
use `TETRA0001`; positioned semantic/compiler diagnostics use `TETRA2001`;
formatter check mismatches use `TETRA_FMT002`. Text diagnostics keep the
traditional `file:line:column: message` shape.

`tetra targets` prints the current target surface. In v0.6 the supported build
targets are `linux-x64`, `windows-x64`, and `macos-x64`; `wasm32-wasi` and
`wasm32-web` are reported as planned targets with clear diagnostics until the
real v1.0 WASM backend lands.
`tetra smoke --list --format=json` emits the canonical smoke matrix without
building, so CI can validate smoke coverage before the slower build/run stage.

`tetra lsp --stdio-smoke <file>` emits a one-shot LSP-basic analysis.
`tetra lsp --stdio` runs the v0.6 minimal JSON-RPC loop for initialize,
didOpen/didChange/didClose diagnostics, document symbols, hover, shutdown, and
exit. LSP diagnostics include parser/frontend errors and single-file semantic
diagnostics; library-style files are checked without requiring `main`. Unsaved
stdio documents with imports currently keep parser/symbol/hover coverage but
skip semantic checks to avoid false unresolved-import noise. On-disk
`--stdio-smoke <file>` analysis loads the module graph and reports
imported-module semantic diagnostics.
`tools/cmd/gen-docs` generates Markdown API docs from modules, functions,
structs, enums, effects, and tests.

Stable core stdlib modules:

```
import lib.core.math as math
import lib.core.capability as cap
import lib.core.memory as mem
import lib.core.io as io
import lib.core.testing as testing
```

`lib.core.math` provides small `i32` helpers. `lib.core.capability` and
`lib.core.memory` wrap the current explicit capability/memory builtins; callers
still declare matching `uses` effects and raw memory remains unsafe.
`lib.core.io` wraps capability-gated MMIO helpers and `lib.core.testing` offers
small assertion/status combinators for smoke/test-style flows.

Local Eco/Capsule alpha commands:

```
./tetra eco verify Tetra.capsule
./tetra eco verify --target linux-x64 --lock tetra.lock.json App.capsule Core.capsule
./tetra eco pack Tetra.capsule -o app.todex
./tetra eco pack --project Tetra.capsule -o app.todex
./tetra eco unpack app.todex -C out
./tetra eco vault add --store .tetra/todex-vault --kind source examples/flow_hello.tetra
./tetra eco vault list --store .tetra/todex-vault
./tetra eco vault verify --store .tetra/todex-vault
```

`eco verify` accepts local dependency graphs in alpha manifests with
`target`, `effect`, and `dependency "<id>" "<version>"` entries. It validates
duplicate capsule IDs, missing dependencies, dependency version mismatches, and
optional target compatibility; `--lock` writes a local JSON provenance file.
`eco pack --project` creates a local project bundle rooted at the capsule file's
directory while preserving the single-manifest `eco pack` behavior.
`eco vault` is a local-only Todex prototype: it stores source/interface/build/test
records by SHA-256 content address and verifies the local object store.

## Language Features

### Staged Profile Surface

Tetra currently exposes a staged alpha surface. Older brace syntax, the Flow
bridge, Core MVP constructs, checked `uses` effects, runtime/object-linking
workflows, and developer tooling are the v0.14-v0.18 baseline. v0.5 adds
optionals, typed errors, ownership markers, simple generics, protocol
conformance checks, extensions, async/task MVP, stable `lib/core` helpers,
local Capsule/Todex graphing, docs generation, and LSP basics. v0.6 hardens
that surface for daily local use.

- Multiple functions: `fun main(): i32 { ... }` (legacy `fn main() -> i32` is still accepted)
- v0.14 Flow bridge: `func main() -> Int:` with indentation blocks is accepted and lowered into the existing AST/IR path
- Flow structs: `struct Vec2:` with indented fields
- v0.15 Core MVP: real `bool`, `true`/`false`, range `for`, no-payload `enum`, and statement-level `match`
- v0.16 runtime/toolchain stabilization: runtime mode selection, self-host actors runtime embedding, `--emit=library`, and repeatable `--link-object`
- v0.17 effects MVP: checked `uses` declarations for observable effects
- v0.18 tooling alpha: `tetra fmt`, `tetra test`, JSON diagnostics, and docs doctests
- `main` must exist and has no parameters
- Statements:
  - `print("...");` (string literal or `[]u8`)
  - `var x: i32 = <expr>` (mutable)
  - `val x: i32 = <expr>` (immutable)
  - Top-level `const name: i32 = <constant-expr>` / `const name = <constant-expr>` immutable globals
  - Local `const name: Type = <expr>` immutable bindings
  - `let x: i32 = <expr>` (legacy alias of `var`)
  - In Flow syntax, `let` is immutable and is normalized to MVP `val`
  - `x = <expr>`
  - `x += <expr>`, `x -= <expr>`, `x *= <expr>`, `x /= <expr>`, `x %= <expr>` as assignment sugar
  - `if (<expr>) { ... } else { ... }`
  - Flow form: `if expr:` / `else if expr:` / `else:`
  - `while (<expr>) { ... }`
  - Flow form: `while expr:`
  - Flow form: `for i in start..<end:` (exclusive upper bound, integer ranges only)
  - Flow form: `for value in collection:` for `String`, `[]u8`, and `[]i32`
  - `break` and `continue` inside `while`, range `for`, and collection `for`
  - Flow form: `match value:` with `case Enum.value:` / `case 1:` / `case none:` / `case some(x):` / `case _:`
  - `island(<size>) as <name> { ... }` (scoped island, auto-free)
  - Flow form: `island(<size>) as <name>:`
  - `unsafe { ... }` (enables unsafe operations)
  - Flow form: `unsafe:`
  - `free(<island>)` (allowed only in `unsafe` blocks)
  - `;` (empty statement / no-op)
  - `return <expr>`
- v0.17 effects: function signatures use `uses ...` to declare observable effects such as `io`, `mem`, `alloc`, `capability`, `islands`, `mmio`, `link`, `control`, `runtime`, and `actors`
- v0.18 tests: top-level `test "name":` blocks support `expect <bool>` and are ignored by normal app builds
- v0.5/v0.7 optionals MVP: `T?`, `none`, implicit one-slot `some` values, `value == none`, Flow `if let name = value:`, and statement `match` with `case none:` / `case some(x):`
- v0.5 typed errors MVP: `throws ErrorType`, `throw value`, and `try callee()` for one-slot success/error values inside throwing functions
- v0.5 ownership markers MVP: `borrow`, `inout`, and `consume` parameter markers with local mutation/consume diagnostics
- v0.5 async/task MVP: `async func` and `await callee()` are checked and lowered through the current synchronous call path; `core.task_spawn_i32("fn")` and `core.task_join_i32(task)` provide a cooperative single-slot task API gated by `uses runtime`
- v0.5 protocols MVP: `protocol Name:` declarations plus `impl Type: Protocol` conformance checks against extension/static methods
- v0.5 extensions MVP: `extension Type:` methods lower to namespaced static functions such as `Type.method(value)`
- v0.5 generics MVP: simple same-module generic functions such as `func id<T>(x: T) -> T` are monomorphized at call sites
- v0.6 tooling hardening: formatter coverage for examples/libs, LSP stdio MVP, and Eco project bundle mode
- Expressions: integer literals, `true`/`false`, string literals, identifiers, enum cases (`Color.red`), calls `foo(1, 2)`, field/index access, `+`, `-`, `*`, `/`, `%`, comparisons, `&&`, `||`, unary `-`, unary `!`, and parentheses
- Semicolons are optional after statements
- Every function must end with `return`
- Calls support `i32` arguments (first 6 in registers, 7+ on the stack)

See `docs/spec/flow_syntax_mvp.md` for the supported Flow/Core profile surface.

Planned beyond v0.7 hardening: enum payloads, general iterator protocols,
closures/comprehensions, full Rust-grade ownership, full structured concurrency,
UI DSL and UI backends, richer effect polymorphism/inference, package
publishing, proof-carrying capsules, and the complete EcoNet/Todex ecosystem.

### Types

- `i32` - 32-bit signed integer
- `u8` - 8-bit unsigned byte (currently treated as “int32-like” in expressions; implicitly compatible with `i32`)
- `Int` - alias of `i32`
- `UInt8` / `Byte` - aliases of `u8`
- `Bool` / `bool` - real boolean type; `true`/`false` lower to the existing single-slot backend representation
- `ptr` - raw pointer
- `str` - string literal (ptr + len)
- `String` - alias of `str`
- `[]u8`, `[]i32` - slices
- `island` - (**NEW**) region memory handle
- User-defined structs
- User-defined no-payload enums
- One-slot optionals: `Int?`, `Bool?`, enum optionals, and `none`

### Islands Memory Model

Islands are Tetra's region-based memory management system. An island is a contiguous memory region with bump allocation.

**Builtin functions (unsafe):**

```tetra
fun main(): i32 uses alloc, capability, islands, mem {
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
  return 0
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
- Actors alpha uses `i32` messages and a single-thread cooperative scheduler; the default CLI runtime mode is `--runtime=auto`, which selects the embedded self-host runtime when actors are used.
- v0.6 Usable Alpha is a coherent local compiler/tooling profile. It does not imply the full future language, package ecosystem, UI stack, or distributed runtime is complete.
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
fun main(): i32 uses alloc, islands, io, mem {
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

### Flow Hello

```tetra
func main() -> Int
uses io:
    let msg: String = "Hello from Flow!\n"
    print(msg)
    return 0
```

### Sum of Array

```tetra
fun main(): i32 uses alloc, islands, mem {
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
