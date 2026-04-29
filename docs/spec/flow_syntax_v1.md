# Tetra Flow v1 Grammar (Canonical)

This is the single canonical grammar/source-surface reference for Tetra Flow
v1. Compile/check/fmt canonical frontend paths are defined by this document.

Normalization is retained only as migration tooling (for example,
`compiler.NormalizeFlowForMigration`) and is not part of the canonical frontend
path. See `docs/frontend/flow_parser_plan.md`.

## Supported v1 Surface

Functions:

```tetra doctest
func add(a: Int, b: Int) -> Int:
    return a + b
```

Single-expression functions may use an expression body. The parser lowers this
surface form to an ordinary one-statement function body with `return`.

```tetra doctest
func add(a: Int, b: Int) -> Int = a + b
```

Functions declare observable effects with `uses`:

```tetra
func main() -> Int
uses io:
    print("hello\n")
    return 0
```

Semantic function clauses (`budget`, `noalloc`, `noblock`, `realtime`,
`nothrow`) are accepted by the v1 frontend. Current behavior is:

- `nothrow` is rejected when combined with `throws`.
- `budget(<int-constant>)` requires a non-negative integer constant.
- `noalloc`, `noblock`, and `realtime` are enforced in checker phase 1:
  direct calls, closure-symbol calls, and function-typed callback arguments are
  rejected when the resolved callee/callback violates the clause contract.
- `realtime` additionally requires both `noalloc` and `noblock`.

This is static checker enforcement only; no new runtime/scheduler semantics are
introduced in this phase.

v0.17 enforces the first MVP effect set: `alloc`, `mem`, `io`, `mmio`,
`islands`, `capability`, `link`, `control`, `runtime`, and `actors`.
Aliases `cap.mem` and `cap.io` are accepted for `mem` and `io`.

Function-call argument labels such as `add(a: 1, b: 2)` are planned for a later
release. The MVP parser keeps `TypeName(field: value)` syntax for type-like
struct constructors such as `Vec2(x: 40)`, so lowercase/function-like callees
with labels currently produce an explicit planned-feature diagnostic instead of
being treated as calls.

Type inference is intentionally local and predictable. Integer literals default
to `i32`, boolean literals to `bool`, string literals to `str`, and `none`
requires an expected optional type from an annotation, return type, assignment,
or call parameter. Expected types flow through annotated `let`/`var` bindings,
assignments, returns, function arguments, `try`, `await`, and implicit optional
packing. Generic calls are inferred only from value arguments; return-only type
parameters and `none` without an expected optional type are diagnostics that ask
for an explicit annotation.

Closures are accepted in MVP as function literals:

```tetra
func main() -> Int:
    let f: ptr = fn(x: Int) -> Int:
        return x
    return 0
```

The current MVP lowering materializes closure literals as synthetic module
functions and a `ptr` value to that symbol.

Function type references are parser-accepted in type positions:

```tetra
func apply(cb: fn(Int, Bool) -> UInt8, x: Int, ok: Bool) -> UInt8:
    return cb(x, ok)
```

Callable MVP boundaries in this wave:

- supported: `let`-bound function-typed locals initialized with a
  non-capturing, non-generic, non-throwing closure literal, followed by direct
  local calls; and callee-side callback parameter calls when the caller passes
  either a known symbol-backed function-typed local (for example
  `apply(f, 41)`) or a direct named non-generic non-throwing function/closure
  symbol (for example `apply(add1, 41)`); plus return of symbol-backed
  non-generic non-throwing values from function-typed return paths and
  immutable function-typed local-to-local binding when signatures match, with
  function-typed local reassignment rejected in this MVP-safe path;
- unsupported with explicit diagnostics: function-value escape/passing/storing,
  capturing closure binding to function type, generic/throwing callback symbols
  in this path, and signature mismatches.

Generic closure literal syntax (`fn<T>(...)`) is parser-accepted and supported
only in the existing local direct-call subset with inferable concrete arguments.
Outside that subset, semantic checking reports explicit diagnostics.

Top-level `closure` declarations are also accepted by the parser and lowered
through the same function declaration surface:

```tetra
closure add1(x: Int) -> Int {
    return x + 1
}
```

Tests:

```tetra doctest
test "math":
    expect 40 + 2 == 42
```

`test` blocks are ignored by normal app builds and run through `tetra test`.
`tetra fmt` formats supported MVP syntax in canonical Flow style with 4-space
indentation and sorted `uses` clauses.

`test` declarations require a quoted test name (`test "name":`) and each
`expect` statement requires an expression. Malformed test declarations and
`expect` expressions report structured parse diagnostics (`file`, `line`,
`column`, `code`) so release checks can assert deterministic failure locations.

## Blocks And Indentation

Canonical Flow uses `:` to open indentation-sensitive blocks. A block header
must be followed by at least one line with greater indentation, except `match`
headers whose immediate `case` clauses stay aligned with the `match` line.
Tabs are rejected in indentation. Blank lines and standalone comments do not
start or end blocks.

The supported block headers are:

- top-level `capsule`, `struct`, `enum`, `protocol`, `extension`, `impl`,
  `state`, `view`, `actor`, `closure`, `property`, `test`, and `func`
  declarations;
- statement-level `if`, `else if`, `else`, `if let`, `else if let`, `while`,
  `for`, `match`, `case`, `unsafe`, `island`, and UI `command` blocks;
- closure literals in expression positions (capture forms are constrained by
  callable MVP diagnostics).

Legacy brace/semicolon syntax still parses as compatibility input, but it is a
migration surface. Canonical formatting prints Flow indentation.

`tools/cmd/validate-flow-only` enforces this boundary for release scans. In
addition to braces/semicolons/tabs in general, it explicitly reports legacy
braced test syntax (`test "name" {`) and records exact source positions.

## Comments

Line comments begin with `//` and run to the end of the line. Standalone line
comments may appear before declarations or statements and `tetra fmt` preserves
them at the nearest formatted code position.

Standalone block comments begin with `/*` and end with `*/`; formatter support
is conservative and preserves standalone block comments. Inline comments after
code are intentionally rejected by `tetra fmt` with a formatter diagnostic so
the tool does not silently move or drop user text.

Doc comments currently use the same standalone comment syntax. They are
preserved by the formatter but are not yet a distinct AST node.

Structs:

```tetra
struct Vec2:
    x: Int
    y: Int
```

Blocks:

```tetra
func main() -> Int
uses alloc, capability, islands, mem:
    var out: Int = 0
    if out == 0:
        out = 42
    else if out == 1:
        out = 41
    else:
        out = 1
    while out < 42:
        out = out + 1
    return out
```

Top-level constants:

```tetra
const base: i32 = (20 + 2) * 2
const delta = 100 % 3
const enabled: Bool = base + delta == 45

func main() -> Int:
    if enabled:
        return base + delta - 3
    return 1
```

`const` is an immutable global. In the current MVP it uses the same one-slot
global storage path as `val`, supports `i32`, `bool`, and `ptr`, and type
inference supports constant numeric and boolean expressions. Constant
expressions are intentionally conservative: literals, earlier immutable global
constants in the same file, unary `-`/`!`, arithmetic, comparisons, and
`&&`/`||`. Forward references and division/modulo by zero are reported as
compile-time diagnostics.

Top-level mutable globals (`var`) support compile-time constant initializers
for `i32`/`bool` when an explicit type annotation is present:

```tetra
var base: Int = 40 + 2
var enabled: Bool = true && !false
```

Current MVP limits for top-level `var` initializers:

- initializer must be a compile-time constant expression;
- non-constant forms (for example function calls) are rejected;
- `ptr` initializers on mutable globals are explicitly rejected in this phase;
- `String`/`str` globals are supported only with string-literal initializers;
  non-literal string initializers remain diagnostics in this phase.

Function bodies also accept local `const` bindings:

```tetra
func main() -> Int:
    const answer: Int = 42
    return answer
```

Assignments support the normal `=` form and the MVP arithmetic compound forms:

```tetra
func main() -> Int:
    var x: Int = 4
    x += 3
    x *= 6
    return x
```

`+=`, `-=`, `*=`, `/=`, and `%=` lower through the existing assignment and
binary-expression path. They do not introduce a separate IR operation.

Booleans:

```tetra
func main() -> Int:
    let ok: Bool = true
    if ok && !(3 < 2):
        return 42
    return 1
```

Range `for` loops:

```tetra
func main() -> Int:
    var total: Int = 0
    for i in 0..<11:
        total = total + i
    return total
```

The v0.15 range form is exclusive on the upper bound and supports only integer
ranges. The v0.6.x hardening line also accepts collection iteration over
`String`, `[]u8`, `[]u16`, `[]i32`, and `[]bool`:

```tetra
func main() -> Int:
    var total: Int = 0
    let text: String = "*"
    for ch in text:
        total = total + ch
    return total
```

Loop bodies support `break` and `continue`. Both are diagnostics outside a
`while`, range `for`, or collection `for`.

The v1 frontend accepts statements that appear after `return`, `throw`,
`break`, or `continue` in the same block. Those statements are treated as
ordinary unreachable code: they may still be parsed, type-checked, formatted,
and lowered, but no unreachable-code diagnostic is promised in v1. Lowering is
required to keep verifier invariants such as stack balance and valid branch
targets even when unreachable instructions are present.

Enums, payloads, and `match`:

```tetra
enum Result:
    case ok(Int)
    case err(Int, Int)
    case empty

func main() -> Int:
    let result: Result = Result.ok(42)
    let score: Int = match result:
    case Result.ok(value):
        value
    case Result.err(code, detail):
        code + detail
    case Result.empty:
        0
    return score
```

`match` is accepted as both a statement and an expression. Patterns support
same-module enum cases with zero or more payload bindings, integer literals,
`none` and `some(name)` for one-slot optionals, optional guards with `if`, and
`_` default. An enum match is treated as complete when every enum case is
covered; integer matches still require `_` when used as a terminal returning
statement. An optional match with both `none` and `some(name)` is treated as
complete. Match expressions must be exhaustive and all case bodies must produce
the same result type.

Optionals:

```tetra
func maybe(flag: Bool) -> Int?:
    if flag:
        return 42
    else:
        return none

func unwrap(value: Int?) -> Int:
    if let x = value:
        return x
    else:
        return 0
```

The v1 optional contract supports `none`, implicit `some` packing for
compatible values, equality/inequality with `none`, Flow `if let` unwrapping,
and statement `match` with `none`, `some(name)`, and `_`. Optional layout is a
presence tag followed by the payload slots, so `Int?` uses two slots and
`String?` uses three slots.

Ownership markers:

```tetra
func add_one(x: borrow Int) -> Int:
    return x + 1

func bump(x: inout Int) -> Int:
    x = x + 1
    return x

func take(x: consume Int) -> Int:
    return x
```

The v1 ownership marker contract records ownership in function signatures,
keeps `borrow` parameters immutable, allows mutation only through `inout`
arguments backed by mutable locals, and reports diagnostics for use after
`consume`, consuming the same value twice in one call, and aliasing an `inout`
argument with a borrow or consume argument. Region-backed values derived from a
borrow cannot escape through returns, owned parameters, or `inout` assignment.

Typed errors:

```tetra
enum ReadError:
    case eof

func read(flag: Bool) -> Int throws ReadError:
    if flag:
        return 42
    else:
        throw ReadError.eof

func caller() -> Int throws ReadError:
    let value: Int = try read(true)
    return value
```

The v1 typed-errors contract supports one-slot and multi-slot success/error
payloads, including enum payload error values. Throwing functions return a
success tag plus success slots and error slots. `throw` must match the declared
error type, `try` is only valid inside a throwing function with a compatible
error type, and bare calls to throwing functions are rejected. A `catch
<throwing-call>:` expression is supported for exhaustive local recovery over
optional or enum error cases, including enum payload bindings and guards.
Throwing `main` wrappers remain post-v1.

Async syntax:

```tetra
async func answer() -> Int:
    return 42

async func caller() -> Int:
    let value: Int = await answer()
    return value
```

The v1 async MVP is a checked surface over the current synchronous lowering
path. Calls to async functions require `await`, and `await` is only valid inside
another async function. For async throwing calls, `try await <call>()` is the
supported typed-error propagation boundary through the synchronous lowering
path. The alternate spelling `await try <call>()` is rejected with a stable
diagnostic that points back to `try await`; full async/error runtime ABI
semantics are not claimed beyond this tested boundary. The cooperative task API
adds
`core.task_spawn_i32("worker")`, `core.task_spawn_group_i32(group, "worker")`,
`core.task_join_i32(task)`, and `core.task_join_result_i32(task)` for
zero-argument synchronous `i32` workers; these APIs require `uses runtime`.
Task groups expose typed handles and cancellation state, while structured
concurrency remains post-v1.

Typed task handles are currently bounded to native runtime wrappers for slot
counts `2..8`. Slot counts `2..4` use the direct runtime join ABI, and slot
counts `5..8` use a staged runtime join path. One-slot task handles continue
to use the existing `task.i32` path, and typed layouts above `8` are rejected
by semantics. For staged `5..8`, worker targets stay zero-arg synchronous
`i32` functions and may be either non-throwing or throwing the same typed task
error enum (`func worker() -> Int` or `func worker() -> Int throws E`).

Extensions:

```tetra
struct Vec2:
    x: Int
    y: Int

extension Vec2:
    func sum(self: Vec2) -> Int:
        return self.x + self.y
```

The v1 extensions contract lowers methods to deterministic namespaced static
functions. Call extension methods as `Vec2.sum(value)` or through an imported
type namespace such as `core.Vec2.draw(value)`. Duplicate generated method names
are rejected. Implicit receiver-call syntax and constrained extensions remain
post-v1.

Protocols:

```tetra
protocol Renderable:
    func draw(self: Vec2) -> Int

impl Vec2: Renderable
```

The v1 protocols contract accepts declarations with typed required function
signatures, validates referenced types, and exposes them to formatter,
generated docs, and LSP symbols. `impl Type: Protocol` checks that matching
extension/static methods exist with compatible signatures, including effects,
async, throws, params, and return type. Duplicate impls are rejected.
Minimal generic protocol requirements are supported using syntax such as
`func map<T>(self: Vec2, value: T) -> T`. Current MVP limits:

- requirement generic parameters are name-only (`<T>`); bounds in requirements
  are not supported;
- conformance checks require matching generic parameter count/order and
  compatible signature shape;
- no new runtime or dynamic dispatch model is introduced.

Actor declarations (subset):

```tetra
actor Counter {
    var count: Int = 0
    val step: UInt8 = 1
    const boost: UInt16 = 2
    var err: task.error = 0

    func run() -> Int {
        count = count + step
        return count + boost + err
    }
}
```

The current actor declaration subset supports `var`/`val`/`const` fields with
state types `Int`, `Bool`, `UInt8`, `UInt16`, and `task.error`. Field
initializers must be compile-time constants. Unsupported pointer/resource or
aggregate actor-state field types are rejected by semantics.

Generic signatures:

```tetra
func id<T>(x: T) -> T:
    return x
```

The v1 generics MVP parses, validates, formats, documents, and monomorphizes
generic function calls with inferred value arguments across modules. Generated
specialization names are deterministic and encode fully qualified type names to
avoid collisions. Higher-ranked generics, explicit type arguments,
full protocol-bound generic dispatch, and specialization optimization remain
post-v1.

Unsafe and scoped islands:

```tetra
func main() -> Int:
    island(64) as isl:
        var msg: []UInt8 = core.island_make_u8(isl, 1)
        msg[0] = 10

    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let _: Int = core.store_i32(p, 42, mem)
        return core.load_i32(p, mem)
    return 0
```

## Surface Aliases

- `Int` maps to `i32`
- `String` maps to `str`
- `UInt8` and `Byte` map to `u8`
- `UInt16` maps to `u16`
- `Bool` maps to `bool`
- `ConsentToken` maps to `consent.token`
- `SecretInt` maps to `secret.i32`
- `T?` stores a presence tag plus the payload slots
- `throws E` uses a success tag plus success and error payload slots
- `async func` currently lowers through the normal synchronous ABI
- `core.task_spawn_i32` runs zero-argument `i32` workers through the cooperative task MVP
- Generic functions are monomorphized at call sites using deterministic specialization names

## Expression Precedence

From tightest to loosest:

| Level | Forms | Associativity |
| --- | --- | --- |
| 1 | calls, field access, indexing, literals, identifiers, parenthesized expressions | left |
| 2 | prefix `!`, prefix `-`, `try`, `await` | prefix |
| 3 | `*`, `/`, `%` | left |
| 4 | `+`, `-` | left |
| 5 | `<`, `<=`, `>`, `>=` | non-chaining |
| 6 | `==`, `!=` | non-chaining |
| 7 | `&&` | left |
| 8 | `||` | left |

Relational and equality chains such as `a < b < c` and `a == b == c` are
diagnostics rather than implicitly associated expressions.

## Diagnostics And Formatting Contracts

Frontend diagnostics are machine-readable. Parse/frontend diagnostics use
`TETRA0001`; semantic/compiler diagnostics rendered from positioned errors use
`TETRA2001`; formatter preservation diagnostics use `TETRA_FMT001`; formatter
check mismatches use `TETRA_FMT002`. Diagnostics exposed by CLI JSON include
`code`, `severity`, `message`, and, when available, `file`, `line`, and
`column`. Formatter check JSON diagnostics report the first differing source
position when formatted output would change the file. UTF-8 input must be valid;
invalid byte sequences are reported before tokenization with an exact byte
position.

The v1 parser recovery contract is deliberately first-error based: malformed
source returns one structured diagnostic at the earliest reliable location.
Multi-error recovery and partial AST production after syntax errors are post-v1
work, so v1 tools must not depend on multiple diagnostics from one parse.

Formatter guarantees for the supported Flow surface:

- formatting is idempotent;
- indentation is four spaces per block level;
- `uses` clauses are sorted deterministically;
- standalone line comments and block comments are preserved;
- inline comments after code are rejected with a formatter diagnostic instead
  of being moved or discarded;
- malformed files report diagnostics rather than partially formatted output.

## Not In Canonical v1 Surface

`property` and top-level language `capsule` declarations are parser/semantic
accepted in the current profile. `capsule` is metadata-only in this wave (no
runtime/ABI integration).

Language `capsule` declarations are distinct from Eco project packaging files
(`Capsule.t4` / `Tetra.capsule`). The former is source-language metadata;
the latter is project/package manifest metadata.

Payload pattern forms beyond the currently tested enum/optional cases,
exhaustive integer match checking, collection `for` exhaustiveness
improvements, full first-class function-value/callable matrix, effect
polymorphism/inference, protocol-bound generic dispatch, implicit receiver-call syntax,
distributed actors, and structured concurrency are planned for later releases.
