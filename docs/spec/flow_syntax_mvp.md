# Tetra Flow/Core Syntax MVP (v0.14-v0.18)

This document describes the Flow syntax accepted by the MVP compiler.
Flow syntax is normalized into the existing AST/IR path, so it is a frontend
compatibility layer rather than a backend redesign.

## Supported

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

v0.17 enforces the first MVP effect set: `alloc`, `mem`, `io`, `mmio`,
`islands`, `capability`, `link`, `control`, `runtime`, and `actors`.
Aliases `cap.mem` and `cap.io` are accepted for `mem` and `io`.

Function-call argument labels such as `add(a: 1, b: 2)` are planned for a later
release. The MVP parser keeps `TypeName(field: value)` syntax for type-like
struct constructors such as `Vec2(x: 40)`, so lowercase/function-like callees
with labels currently produce an explicit planned-feature diagnostic instead of
being treated as calls.

Tests:

```tetra doctest
test "math":
    expect 40 + 2 == 42
```

`test` blocks are ignored by normal app builds and run through `tetra test`.
`tetra fmt` formats supported MVP syntax in canonical Flow style with 4-space
indentation and sorted `uses` clauses.

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

The v0.5 optionals MVP supports one-slot payloads, `none`, implicit `some`
packing for compatible values, equality/inequality with `none`, and Flow
`if let` unwrapping. Multi-slot optional payloads such as `String?` are planned
for a later layout pass.

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

The v0.5 marker MVP records ownership in function signatures, keeps `borrow`
parameters immutable, allows local mutation through `inout`, and reports a
diagnostic when a local value is reused after being passed to a `consume`
parameter. It is not a full lifetime solver.

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

The v0.5 typed-errors MVP supports one-slot success values and one-slot error
values. `try` is only valid inside another throwing function, and `main` remains
non-throwing. Catching/recovery syntax, typed error unions, multi-slot error
payloads, and throwing `main` wrappers are planned for later releases.

Async syntax:

```tetra
async func answer() -> Int:
    return 42

async func caller() -> Int:
    let value: Int = await answer()
    return value
```

The v0.5 async MVP is syntax and semantic checking over the existing
synchronous call path. Calls to async functions require `await`, and `await` is
only valid inside another async function. The cooperative task MVP adds
`core.task_spawn_i32("worker")` and `core.task_join_i32(task)` for zero-argument
`i32` worker functions; these APIs require `uses runtime`. Cancellation and
structured concurrency remain later runtime work.

Extensions:

```tetra
struct Vec2:
    x: Int
    y: Int

extension Vec2:
    func sum(self: Vec2) -> Int:
        return self.x + self.y
```

The v0.5 extensions MVP lowers methods to namespaced static functions. Call
extension methods as `Vec2.sum(value)`. Extension protocol conformance clauses,
implicit receiver-call syntax, and cross-module extension lookup are planned for
later releases.

Protocols:

```tetra
protocol Renderable:
    func draw(self: Vec2) -> Int

impl Vec2: Renderable
```

The v0.5 protocols MVP accepts declarations with typed required function
signatures, validates referenced types, and exposes them to formatter, generated
docs, and LSP symbols. `impl Type: Protocol` checks that matching
extension/static methods exist with compatible signatures. Protocol-bound
generics and dynamic dispatch are planned for later releases.

Generic signatures:

```tetra
func id<T>(x: T) -> T:
    return x
```

The v0.5 generics MVP parses, validates, formats, documents, and monomorphizes
simple same-module generic function calls with inferred type parameters.
Higher-ranked generics, protocol-bound generics, and specialization optimization
are planned for later releases.

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

## MVP aliases

- `Int` maps to `i32`
- `String` maps to `str`
- `UInt8` and `Byte` map to `u8`
- `Bool` and `bool` are real boolean types in v0.15
- `T?` stores a one-slot optional payload plus a presence tag in v0.5
- `throws E` uses a one-slot success payload plus a one-slot error/tag result in v0.5
- `async func` currently lowers through the normal synchronous ABI in v0.5
- `core.task_spawn_i32` runs zero-argument `i32` workers through the current cooperative task MVP
- Simple same-module generic functions are monomorphized at call sites in v0.5

## Not yet implemented

The compiler intentionally reports planned-feature diagnostics for `actor`,
`view`, `state`, `property`, and `capsule` language declarations.

Enum payloads, richer payload match patterns, exhaustive integer match checking,
collection `for`, catch/recovery syntax, effect polymorphism/inference, full
ownership/lifetime solving, protocol-bound generics, extension conformance
clauses, task cancellation, and structured concurrency are planned for later
releases.
