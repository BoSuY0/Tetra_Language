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
- `noalloc`, `noblock`, and `realtime` are accepted as marker clauses.

These clauses are currently syntax/semantic metadata only (no codegen or
scheduler/runtime enforcement yet).

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

Closures are accepted in MVP as non-capturing function literals:

```tetra
func main() -> Int:
    let f: ptr = fn(x: Int) -> Int:
        return x
    return 0
```

The current MVP lowering materializes closure literals as synthetic module
functions and a `ptr` value to that symbol. Capturing outer locals and invoking
function pointers are planned for later releases.

Tests:

```tetra doctest
test "math":
    expect 40 + 2 == 42
```

`test` blocks are ignored by normal app builds and run through `tetra test`.
`tetra fmt` formats supported MVP syntax in canonical Flow style with 4-space
indentation and sorted `uses` clauses.

## Blocks And Indentation

Canonical Flow uses `:` to open indentation-sensitive blocks. A block header
must be followed by at least one line with greater indentation, except `match`
headers whose immediate `case` clauses stay aligned with the `match` line.
Tabs are rejected in indentation. Blank lines and standalone comments do not
start or end blocks.

The supported block headers are:

- top-level `struct`, `enum`, `protocol`, `extension`, `impl`, `state`, `view`,
  `test`, and `func` declarations;
- statement-level `if`, `else if`, `else`, `if let`, `else if let`, `while`,
  `for`, `match`, `case`, `unsafe`, `island`, and UI `command` blocks;
- non-capturing closure literals in expression positions.

Legacy brace/semicolon syntax still parses as compatibility input, but it is a
migration surface. Canonical formatting prints Flow indentation.

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
`String`, `[]u8`, and `[]i32`:

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

No-payload enums and statement-level `match`:

```tetra
enum Color:
    case red
    case green

func main() -> Int:
    let color: Color = Color.green
    match color:
    case Color.red:
        return 1
    case Color.green:
        return 42
    case _:
        return 0
```

`match` is a statement, not an expression. Patterns support same-module enum
cases, integer literals, `none` and `some(name)` for one-slot optionals, and
`_` default. A no-payload enum match is treated as complete when every enum
case is covered; integer matches still require `_` when used as a terminal
returning statement. An optional match with both `none` and `some(name)` is
treated as complete. Enum payload patterns currently produce planned-feature
diagnostics.

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
payloads. Throwing functions return a success tag plus success slots and error
slots. `throw` must match the declared error type, `try` is only valid inside a
throwing function with a compatible error type, and bare calls to throwing
functions are rejected. Catching/recovery syntax and throwing `main` wrappers
remain post-v1.

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
another async function. The cooperative task API adds
`core.task_spawn_i32("worker")`, `core.task_spawn_group_i32(group, "worker")`,
`core.task_join_i32(task)`, and `core.task_join_result_i32(task)` for
zero-argument synchronous `i32` workers; these APIs require `uses runtime`.
Task groups expose typed handles and cancellation state, while structured
concurrency remains post-v1.

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
Protocol-bound generics and dynamic dispatch remain post-v1.

Generic signatures:

```tetra
func id<T>(x: T) -> T:
    return x
```

The v1 generics MVP parses, validates, formats, documents, and monomorphizes
generic function calls with inferred value arguments across modules. Generated
specialization names are deterministic and encode fully qualified type names to
avoid collisions. Higher-ranked generics, explicit type arguments,
protocol-bound generics, and specialization optimization remain post-v1.

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
`column`. UTF-8 input must be valid; invalid byte sequences are reported before
tokenization with an exact byte position.

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

The compiler intentionally reports planned-feature diagnostics for `actor`,
`property`, and `capsule` language declarations. The actor runtime is available
through `core.spawn`, `core.send`, `core.recv`, `core.self`, and `core.sender`;
the declaration syntax is post-v1.

Enum payloads, richer payload match patterns, exhaustive integer match checking,
collection `for` exhaustiveness improvements, closure captures/function-pointer
invocation, catch/recovery syntax, effect polymorphism/inference,
protocol-bound generics, implicit receiver-call syntax, distributed actors, and
structured concurrency are planned for later releases.
