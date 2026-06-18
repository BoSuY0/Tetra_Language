# Tetra Flow v1 Grammar (Canonical)

This is the single canonical grammar/source-surface reference for Tetra Flow v1. Compile/check/fmt
canonical frontend paths are defined by this document.

Normalization is retained only as migration tooling (for example,
`compiler.NormalizeFlowForMigration`) and is not part of the canonical frontend path. See
`docs/frontend/flow_parser_plan.md`.

## Supported v1 Surface

Functions:

```tetra doctest
func add(a: Int, b: Int) -> Int:
    return a + b
```

Single-expression functions may use an expression body. The parser lowers this surface form to an
ordinary one-statement function body with `return`.

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

Semantic function clauses (`budget`, `noalloc`, `noblock`, `realtime`, `nothrow`) are accepted by
the v1 frontend. Current behavior is:

- `nothrow` is rejected when combined with `throws`.
- `budget(<int-constant>)` requires a non-negative integer constant.
- `noalloc`, `noblock`, and `realtime` are enforced in checker phase 1: direct calls, closure-symbol
  calls, and function-typed callback arguments are rejected when the resolved callee/callback
  violates the clause contract. Direct calls through function-typed locals, function-typed struct
  fields, and function-typed globals report clause violations against the user-visible callable
  name; captured `fnptr` locals use the visible `function-typed callback` phrase. Function-typed
  local, struct-field, and immutable/mutable global callback arguments likewise report against the
  visible argument name instead of the backing function symbol; function-typed return-call callback
  arguments report the visible call form such as `pick()`, and direct closure literal callback
  arguments report signature/effect/unsupported-throwing diagnostics as `closure literal`, including
  generic closure capture rejections. Callback parameters and target-set-backed function-typed
  values are checked against their declared function-type effects when no single concrete callback
  symbol is available.
- Function type references may include typed-error propagation as `fn(...) -> R throws E`; the
  supported runtime slice is limited to immutable local direct-try bindings to a known throwing
  function symbol, declared function-typed returns of a concrete throwing symbol followed by local
  direct-try dispatch, immutable local struct-field direct-try dispatch for concrete throwing
  symbols, enum-payload pattern-bound direct-try dispatch for concrete throwing symbols, immutable
  same-module or imported-public function-typed global direct-try dispatch, local aliases, direct
  callback arguments, and local struct-field reassignment for concrete throwing symbols, same-module
  mutable function-typed global direct-try dispatch and direct throwing callback arguments, plus
  local struct-field/enum-payload storage direct-try after compatible concrete throwing-symbol
  initialization or reassignment, and direct synchronous callback-parameter dispatch through
  `try cb(...)` when the callback parameter type declares the same throws type.
- `realtime` additionally requires both `noalloc` and `noblock`.

This is static checker enforcement only; no new runtime/scheduler semantics are introduced in this
phase.

v0.17 enforces the first MVP effect set: `alloc`, `mem`, `io`, `mmio`, `islands`, `capability`,
`link`, `control`, `runtime`, and `actors`. Aliases `cap.mem` and `cap.io` are accepted for `mem`
and `io`.

Function-call argument labels such as `add(a: 1, b: 2)` are planned for a later release. The MVP
parser keeps `TypeName(field: value)` syntax for type-like struct constructors such as
`Vec2(x: 40)`, so lowercase/function-like callees with labels currently produce an explicit
planned-feature diagnostic instead of being treated as calls.

Type inference is intentionally local and predictable. Integer literals default to `i32`, boolean
literals to `bool`, string literals to `str`, and `none` requires an expected optional type from an
annotation, return type, assignment, or call parameter. Expected types flow through annotated
`let`/`var` bindings, assignments, returns, function arguments, `try`, `await`, and implicit
optional packing. Generic calls are inferred only from value arguments; return-only type parameters
and `none` without an expected optional type are diagnostics that ask for an explicit annotation.

Closures are accepted in the current Level 0 callable MVP as function literals:

```tetra
func main() -> Int:
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x
    return f(0)
```

The current MVP lowering materializes supported non-capturing closure literals as synthetic module
functions with a symbol-backed callable value. Older `ptr` annotations may still appear in
migration-era snippets, but they are not the current callable contract.

Function type references are parser-accepted in type positions:

```tetra
func apply(cb: fn(Int, Bool) -> UInt8, x: Int, ok: Bool) -> UInt8:
    return cb(x, ok)
```

Callable boundary levels in this wave:

- Level 0 current MVP: `fn(...) -> ...` type references plus `let`-bound function-typed locals
  initialized with a non-capturing, non-throwing closure literal or direct function symbol,
  including generic function symbols when every type parameter is inferred from the declared
  `fn(...) -> ...` type, followed by direct local calls; and callee-side callback parameter calls
  when the caller passes either a known symbol-backed function-typed local (for example
  `apply(f, 41)`) or a direct named non-throwing function/closure symbol (for example
  `apply(add1, 41)`), including generic function symbols when the callback parameter type fully
  infers the generic arguments (for example `apply(id, 41)`); plus return of symbol-backed
  non-generic non-throwing values from function-typed return paths, target-set-backed function-typed
  parameter returns, function-typed local-to-local binding when signatures match, including snapshot
  copies from mutable function-typed locals, immutable same-module and namespace/selective imported
  public symbol-backed function-typed globals for direct calls, stable diagnostics for imported
  mutable function-typed globals that would require cross-module global-data ABI, and
  signature-compatible mutable local reassignment among supported function-typed values, including
  target-set-backed parameter-return calls such as `identity(captured)` or
  `callbacks.identity(captured)`.
- Level 1 current since `v0.4.0`: production non-capturing symbol-backed callable support covers
  function-typed locals, local aliases including target-set-backed aliases of function-typed
  parameters, callback parameters, function-typed parameter storage into local struct fields with
  direct field calls or synchronous callback arguments, function-typed parameter storage into enum
  payloads with direct payload calls, mutable local enum reassignment, returned enum propagation, or
  synchronous callback arguments, direct named function/closure symbols, symbol-backed
  function-typed returns, declared function-typed local initializers, symbol-backed same-module and
  namespace/selective imported public function-typed globals for direct calls, direct callback
  arguments, same-module mutable global reassignment with direct calls or synchronous callback
  arguments and local or nested local struct-field/enum-payload
  storage/reassignment/returned-aggregate propagation, stable diagnostics for imported mutable
  function-typed globals that would require cross-module global-data ABI, actor/task boundary
  diagnostics when a worker directly dispatches through a mutable function-typed global or passes it
  as a synchronous callback argument, passes a symbol-backed callback argument whose target touches
  mutable globals, imported functions accepting structs with function-typed fields and dispatching
  through caller-supplied local struct values or namespace/selective imported direct struct
  constructors carrying closure literals or captured `ptr` closure locals, imported functions
  accepting enums with function-typed payloads and dispatching through pattern-bound payload
  callbacks from caller-supplied local enum values, direct enum-returning calls, or direct
  namespace/selective imported enum constructor arguments, and inferable same-module/imported
  generic-symbol initializers, function-typed returns including target-set-backed function-typed
  parameter returns, mutable local or nested struct field reassignment, function-typed nested struct
  field initializers, and enum payload initializers or mutable enum-payload reassignments from
  non-capturing generic closure literals or inferable same-module/imported generic function symbols,
  mutable local or nested struct field reassignment from inferable same-module or imported generic
  function symbols, function-typed nested struct field initializers for inferable same-module or
  imported generic function symbols, enum payload initializers or mutable enum-payload reassignments
  for inferable same-module or imported generic function symbols, non-capturing closure-literal
  function-typed globals, exact parameter arity validation for closure literals assigned to declared
  function types or passed directly as callback arguments, snapshot aliasing from mutable
  function-typed locals, and stable diagnostics for unsupported movement.
- Level 2 current since `v0.4.0`: captured closure literals may initialize function-typed locals and
  be called directly when captures are local `Int`/`Bool`/`String` values or simple structs without
  pointer/resource fields. The compiler materializes those values as a nine-slot `fnptr`, so up to
  eight by-value snapshot capture slots may be called directly, including direct calls that use
  labels for the explicit closure parameters and direct function-typed global calls that use labels
  as call-site documentation, with mixed labeled/unlabeled lists rejected against the user-visible
  callable value and explicit type arguments rejected before synthetic closure-symbol lowering,
  passed to synchronous callback parameters as function-typed locals or direct closure-literal
  arguments, including imported callback callees, returned from function-typed return paths as
  function-typed locals or direct closure-literal returns, including multiple known function
  return-path targets from direct symbols, local aliases, or captured closure literals propagated
  through direct local calls and synchronous callback arguments across local or imported module
  boundaries, alias let-bound captured `ptr` closure values into function-typed locals or reassign
  compatible mutable function-typed locals, store them in local struct fields or enum payloads,
  including direct closure-literal container initializers with module-qualified synthetic closure
  targets, return them directly from function-typed return paths, and pass them as direct
  synchronous callback arguments when their environment fits the eight-slot `fnptr` envelope,
  including through imported function-typed parameter-return helpers such as `identity(cb) -> cb`;
  imported returns that ignore a captured callback and return a concrete symbol do not inherit the
  ignored argument's captures, assigned into mutable function-typed locals and reassigned through
  supported mutable local, struct-field, or enum-payload paths, stored in immutable local struct
  fields or enum payloads, including direct closure-literal initializers, target-set-backed
  parameter-return calls such as `Holder(cb: identity(captured))`,
  `Holder(cb: callbacks.identity(captured))`, `MaybeCallback.some(identity(captured))`, or
  `MaybeCallback.some(callbacks.identity(captured))`, and multi-target function-return values, and
  reassigned on supported mutable locals, struct fields, or enum payload values, including imported
  parameter-return reassignment forms such as `cb = callbacks.identity(captured)`,
  `holder.cb = callbacks.identity(captured)`, `box.holder.cb = callbacks.identity(captured)`,
  `holder = Holder(cb: callbacks.identity(captured))`,
  `box.holder = Holder(cb: callbacks.identity(captured))`,
  `box = Box(holder: Holder(cb: callbacks.identity(captured)))`, and
  `MaybeCallback.some(callbacks.identity(captured))`, including enum payloads stored behind mutable
  local struct fields such as `box.choice = MaybeCallback.some(callbacks.identity(captured))`. Known
  struct returns preserve stable function-typed field metadata for subsequent local field calls,
  including after local field reassignment before return and through nested struct literal
  initializers. Returned-struct function-field target sets may collect multiple known return-path
  targets and propagate them through direct field calls or synchronous callback arguments. Returned
  structs whose fields contain enum payloads, such as
  `makeBox(f) -> Box(choice: MaybeCallback.some(f))`, preserve function-typed payload metadata after
  call-site substitution from imported parameter-return arguments, including after whole-struct
  local reassignment such as `box = makeBox(callbacks.identity(captured))` and through nested
  returned struct initializers such as `makeOuter(f) -> Outer(box: makeBox(f))`. They also preserve
  multiple known return-path targets for direct `match box.choice` payload calls. Known enum returns
  preserve stable function-typed payload metadata for subsequent pattern-bound payload calls,
  including direct returned-enum match scrutinees. Returned-enum payload target sets may collect
  multiple known return-path targets and propagate them through synchronous callback arguments after
  pattern binding. Whole-enum local aliases preserve that metadata before pattern binding. Mutable
  local enum values may be reassigned from supported stable-target function payload constructors,
  direct closure literals, known function-typed returns, or whole-enum aliases before a local
  `match`; multiple known branch targets dispatch through stable symbol-address target sets for
  direct calls and synchronous callback arguments. Explicitly declared function-typed locals may
  also bind captured throwing closure literals when the declared `fn(...) -> R throws E` signature
  matches the closure exactly, dispatch them through `try cb(...)` inside a compatible throwing
  function, reassign compatible mutable locals, pass them as direct callback arguments, return them
  from function-typed return paths, and store or reassign them in local struct fields or enum
  payloads for direct-try dispatch, including immutable local aliases from those struct fields or
  enum payload bindings.
- Safe first-class callable movement in the current profile uses a fixed 4-slot handle when an
  immutable by-value captured environment exceeds the bounded `fnptr` envelope. That handle path
  covers local aliases, mutable local storage, same-module mutable global snapshots, function-typed
  returns, local or cross-module returned struct fields and enum payloads, synchronous callback
  arguments, generated `.t4i` interface-only metadata, and return alias chains that return captured
  closure snapshots.
- Unsupported with explicit diagnostics in the current profile: arbitrary heap/thread escape without
  a source-level callable transfer surface, imported mutable global-data escape, let-bound captured
  `ptr` closure locals that are not re-bound through a declared `fn(...)` type, by-reference mutable
  capture, pointer/resource capture, unstable assignment sources without target/capture metadata,
  higher-order generic callable movement and throwing callable movement outside the explicitly
  declared `fn(...) -> R throws E` direct-try slice, unsupported assignment sources, and signature
  mismatches. Workers that directly call through same-module mutable function-typed globals,
  imported immutable function-typed globals whose targets touch mutable globals, pass them as
  synchronous callback arguments, pass same-module/imported symbol-backed callback arguments whose
  targets touch mutable globals, pass same-module or imported direct function-typed return-call
  callback arguments whose returned targets or multi-return target sets touch mutable globals,
  preserve that classification through local/field alias returns and returned struct/enum aggregate
  fields or payloads across module boundaries, directly call function-typed locals/struct
  fields/enum payloads whose targets touch mutable globals, reassign them into function-typed locals
  or local struct fields/enum payloads, store them into local function-typed struct fields/enum
  payloads, return them from function-typed return helpers, or write mutable function-typed globals
  are rejected at actor/task spawn as mutable-global boundary crossings. Direct captured closure
  literals, let-bound captured `ptr` closure locals, direct same-module/imported function-typed
  return calls, immutable local aliases initialized from those return calls, mutable function-typed
  locals, local struct fields, local enum payloads, whole local or nested structs with function
  fields reassigned from struct literals containing direct closure literals or direct return calls,
  whole local enums reassigned from enum constructors containing direct closure literals or direct
  return calls, or same-module or source-imported returned enum payloads or returned struct enum
  payloads carrying direct closure literals, or return alias chains that return captured closure
  snapshots assigned into same-module mutable global function-typed values are stored as bounded
  by-value `fnptr` snapshots and may be called later through that global, passed as synchronous
  callback arguments, returned from same-module or imported functions, passed as callback arguments
  or reassigned into mutable locals after cross-module returns, stored in local struct fields or
  enum payloads, including after cross-module function-typed returns, or dispatched through
  `try cb(...)` when the global type declares the same throws type. Captured `fnptr` values reached
  through mutable function-typed whole-struct reassignments not backed by direct closure or direct
  return-call field initializers, unsupported assignment sources, or parameter escapes remain
  outside the current production claim so broader local `fnptr` environments do not escape without
  heap/lifetime evidence. Function-typed parameters also cannot be stored into mutable global
  function-typed values in the current profile and report a dedicated parameter-to-global escape
  diagnostic, including when the parameter is first routed through a local alias, mutable local
  reassignment, direct same-module or imported function-typed return call, helper return alias,
  helper struct-field return, local struct field, enum payload binding, same-module returned struct
  field, same-module or imported returned nested struct field path, same-module or imported whole
  struct-parameter return, same-module or imported whole enum-parameter return, or same-module or
  imported returned enum payload, and captured values passed through direct, inline, or imported
  source or generated `.t4i` interface-only function-typed parameter-return calls such as
  `identity(f) -> f`, through same-module, imported source, or generated `.t4i` interface-only
  struct-parameter field returns such as `pick(holder) -> holder.cb` and nested paths such as
  `pick(box) -> box.holder.cb`, through same-module, imported source, or generated `.t4i`
  interface-only whole struct-parameter returns such as `echo(box) -> box` that preserve nested
  function-field target sets, through same-module or imported enum-parameter payload returns or
  whole enum-parameter returns such as `echo(choice) -> choice`, including inline imported
  struct/enum constructors carrying captured closure literals, with those returned captured `fnptr`
  values usable for local direct calls or direct synchronous callback arguments, through direct
  function-typed returns from local struct-field aliases or reassignments, enum-payload bindings or
  reassignments, or through returned struct fields including nested paths and enum payloads built
  from function-typed parameters, local aliases of those parameters, or local struct-field aliases
  carrying those parameters, are rejected at the global assignment boundary. Direct `ptr` closure
  calls reject mutable captures with a stable diagnostic because that path would observe mutable
  locals by reference; use an explicit function-typed `fnptr` binding for the supported by-value
  snapshot model. Captured callback arguments, including direct closure-literal callback arguments,
  direct function-typed local calls, direct function-typed struct-field calls, and direct
  function-typed enum-payload calls whose environment exceeds the eight-slot `fnptr` envelope report
  the concrete environment slot count and remain unsupported. Captured closure initializers and
  reassignments for function-typed local storage, struct fields, and enum payloads use the same
  eight-slot environment limit at the semantic storage boundary.

These levels intentionally do not claim full first-class functions.

Generic closure literal syntax (`fn<T>(...)`) is parser-accepted and supported where the current
callable surface can infer every type parameter from the declared `fn(...) -> ...` type, callback
parameter, return type, field type, or enum payload type. Outside that subset, semantic checking
reports explicit diagnostics.

Top-level `closure` declarations are also accepted by the parser and lowered through the same
function declaration surface:

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

`test` blocks are ignored by normal app builds and run through `tetra test`. `tetra fmt` formats
supported MVP syntax in canonical Flow style with 4-space indentation and sorted `uses` clauses.

`test` declarations require a quoted test name (`test "name":`) and each `expect` statement requires
an expression. Malformed test declarations and `expect` expressions report structured parse
diagnostics (`file`, `line`, `column`, `code`) so release checks can assert deterministic failure
locations.

## Flow Source-Of-Truth Examples

These snippets are the release-covered source-of-truth examples for the canonical Flow
parser/formatter surface. They are intentionally small enough to compile/check in focused frontend
and semantics regression tests.

Declarations:

```tetra doctest
module docs.flow_examples

capsule App:
    id: "tetra://docs/flow-examples"
    version: "1.0.0"

struct Vec2:
    x: Int

enum Mode:
    case fast
    case slow

protocol Drawable:
    func draw(self: Vec2) -> Int

extension Vec2:
    func draw(self: Vec2) -> Int:
        return self.x

impl Vec2: Drawable

const answer: Int = 42

func main() -> Int:
    let value: Vec2 = Vec2(x: answer)
    return Vec2.draw(value)
```

Control flow, optionals, errors, and enum payloads:

```tetra doctest
enum ReadError:
    case denied(Int)

func read(flag: Bool) -> Int throws ReadError:
    if flag:
        return 42
    throw ReadError.denied(7)

func main() -> Int:
    let maybe: Int? = none
    if let value = maybe:
        return value

    let recovered: Int = catch read(false):
    case ReadError.denied(code):
        code
    return recovered
```

Callable MVP:

```tetra doctest
func apply(cb: fn(borrow Int) -> Int, value: borrow Int) -> Int:
    return cb(value)

func main() -> Int:
    let add1: fn(borrow Int) -> Int = fn(x: borrow Int) -> Int:
        return x + 1
    return apply(add1, 41)
```

Function-type callback signatures may annotate each parameter with the same ownership markers as
named function parameters: `fn(borrow T) -> U`, `fn(inout T) -> U`, and `fn(consume T) -> U`.
Callback calls and callback symbol binding validate those markers before lowering. Direct calls
through function-typed struct fields use the same positional ownership checks, including aliasing
and mutable-target diagnostics for `borrow`/`consume`/`inout` combinations. Pattern-bound
function-typed enum payload calls use the same callback ownership checks and allow labels as
call-site documentation. Function-typed global direct calls use the declared global function type
for argument count, positional type, and positional ownership checks, with diagnostics reported
against the user-visible global callable name. Explicit type arguments on those value calls are
rejected against that same global callable name.

## Blocks And Indentation

Canonical Flow uses `:` to open indentation-sensitive blocks. A block header must be followed by at
least one line with greater indentation, except `match` headers whose immediate `case` clauses stay
aligned with the `match` line. Tabs are rejected in indentation. Blank lines and standalone comments
do not start or end blocks.

The supported block headers are:

- top-level `capsule`, `struct`, `enum`, `protocol`, `extension`, `impl`, `state`, `view`, `actor`,
  `closure`, `property`, `test`, and `func` declarations;
- statement-level `if`, `else if`, `else`, `if let`, `else if let`, `while`, `for`, `match`, `case`,
  `unsafe`, `island`, and UI `command` blocks;
- closure literals in expression positions (capture forms are constrained by callable MVP
  diagnostics).

Legacy brace/semicolon syntax still parses as compatibility input, but it is a migration surface.
Canonical formatting prints Flow indentation.

`tools/cmd/validate-flow-only` enforces this boundary for release scans. In addition to
braces/semicolons/tabs in general, it explicitly reports legacy braced test syntax (`test "name" {`)
and records exact source positions.

## Comments

Line comments begin with `//` and run to the end of the line. Standalone line comments may appear
before declarations or statements and `tetra fmt` preserves them at the nearest formatted code
position.

Standalone block comments begin with `/*` and end with `*/`; formatter support is conservative and
preserves standalone block comments. Inline comments after code are intentionally rejected by
`tetra fmt` with a formatter diagnostic so the tool does not silently move or drop user text.

Doc comments currently use the same standalone comment syntax. They are preserved by the formatter
but are not yet a distinct AST node.

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

`const` is an immutable global. In the current MVP it uses the same one-slot global storage path as
`val`, supports `i32`, `bool`, and `ptr`, and type inference supports constant numeric and boolean
expressions. Constant expressions are intentionally conservative: literals, earlier immutable global
constants in the same file, unary `-`/`!`, arithmetic, comparisons, and `&&`/`||`. Forward
references and division/modulo by zero are reported as compile-time diagnostics.

Top-level mutable globals (`var`) support compile-time constant initializers for `i32`/`bool` when
an explicit type annotation is present, plus compatible direct named function-symbol initializers
for `fn(...) -> ...` globals:

```tetra
var base: Int = 40 + 2
var enabled: Bool = true && !false
var cb: fn(Int) -> Int = add1
```

Current MVP limits for top-level `var` initializers:

- initializer must be a compile-time constant expression;
- non-constant forms (for example function calls) are rejected;
- `ptr` initializers on mutable globals are explicitly rejected in this phase;
- `String`/`str` globals are supported only with string-literal initializers; non-literal string
  initializers remain diagnostics in this phase.
- function-typed globals are limited to compatible symbol-backed direct function initializers,
  namespace/selective imported public immutable direct calls, and same-module direct-symbol
  reassignment. Imported mutable function-typed globals are rejected with a boundary diagnostic
  until cross-module global-data ABI exists.

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

`+=`, `-=`, `*=`, `/=`, and `%=` lower through the existing assignment and binary-expression path.
They do not introduce a separate IR operation.

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

The v0.15 range form is exclusive on the upper bound and supports only integer ranges. The v0.6.x
hardening line also accepts collection iteration over `String`, `[]u8`, `[]u16`, `[]i32`, and
`[]bool`:

```tetra
func main() -> Int:
    var total: Int = 0
    let text: String = "*"
    for ch in text:
        total = total + ch
    return total
```

Loop bodies support `break` and `continue`. Both are diagnostics outside a `while`, range `for`, or
collection `for`.

The v1 frontend accepts statements that appear after `return`, `throw`, `break`, or `continue` in
the same block. Those statements are treated as ordinary unreachable code: they may still be parsed,
type-checked, formatted, and lowered, but no unreachable-code diagnostic is promised in v1. Lowering
is required to keep verifier invariants such as stack balance and valid branch targets even when
unreachable instructions are present.

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

`match` is accepted as both a statement and an expression. Patterns support same-module enum cases
with zero or more positional payload bindings, integer literals, `none` and `some(name)` for
one-slot optionals, optional guards with `if`, and `_` default. The next-cycle
`language.enum-payload-match` promotion is limited to this positional payload slice plus exhaustive
enum match/catch coverage; richer ADT constructors, nested destructuring patterns, and expanded
guard algebra remain future work. An enum match is treated as complete when every enum case is
covered; integer matches still require `_` when used as a terminal returning statement. An optional
match with both `none` and `some(name)` is treated as complete. Match expressions must be exhaustive
and all case bodies must produce the same result type.

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

The v1 optional contract supports `none`, implicit `some` packing for compatible values,
equality/inequality with `none`, Flow `if let` unwrapping, and statement `match` with `none`,
`some(name)`, and `_`. Optional layout is a presence tag followed by the payload slots, so `Int?`
uses two slots and `String?` uses three slots.

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

The v1 ownership marker contract records ownership in function signatures, keeps `borrow` parameters
immutable, allows mutation only through `inout` arguments backed by mutable locals, and reports
diagnostics for use after `consume`, consuming the same value twice in one call, and aliasing an
`inout` argument with a borrow or consume argument. Region-backed values derived from a borrow
cannot escape through returns, owned parameters, or `inout` assignment. The same ownership contract
is encoded in function-type and callback signatures, so `fn(borrow T) -> U` accepts borrowed
forwarding, `fn(inout T) -> U` requires mutable local arguments, and `fn(consume T) -> U` consumes
the local passed to the callback.

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

The v1 typed-errors contract supports one-slot and multi-slot success/error payloads, including enum
payload error values. Throwing functions return a success tag plus success slots and error slots.
`throw` must match the declared error type, `try` is only valid inside a throwing function with a
compatible error type, and bare calls to throwing functions are rejected. A `catch <throwing-call>:`
expression is supported for exhaustive local recovery over optional or enum error cases, including
enum payload bindings and guards. Throwing `main` wrappers remain post-v1.

Async syntax:

```tetra
async func answer() -> Int:
    return 42

async func caller() -> Int:
    let value: Int = await answer()
    return value
```

The v1 async MVP is a checked surface over the current synchronous lowering path. Calls to async
functions require `await`, and `await` is only valid inside another async function. For async
throwing calls, `try await <call>()` is the supported typed-error propagation boundary through the
synchronous lowering path. The alternate spelling `await try <call>()` is rejected with a stable
diagnostic that points back to `try await`; full async/error runtime ABI semantics are not claimed
beyond this tested boundary. The cooperative task API adds `core.task_spawn_i32("worker")`,
`core.task_spawn_group_i32(group, "worker")`, `core.task_join_i32(task)`, and
`core.task_join_result_i32(task)` for zero-argument synchronous `i32` workers; these APIs require
`uses runtime`. Task groups expose typed handles and cancellation state, while structured
concurrency remains post-v1.

Typed task handles are currently bounded to native runtime wrappers for slot counts `2..8`. Slot
counts `2..4` use the direct runtime join ABI, and slot counts `5..8` use a staged runtime join
path. One-slot task handles continue to use the existing `task.i32` path, and typed layouts above
`8` are rejected by semantics. For staged `5..8`, worker targets stay zero-arg synchronous `i32`
functions and may be either non-throwing or throwing the same typed task error enum
(`func worker() -> Int` or `func worker() -> Int throws E`).

Extensions:

```tetra
struct Vec2:
    x: Int
    y: Int

extension Vec2:
    func sum(self: Vec2) -> Int:
        return self.x + self.y
```

The v1 extensions contract lowers methods to deterministic namespaced static functions. Call
extension methods as `Vec2.sum(value)` or through an imported type namespace such as
`core.Vec2.draw(value)`. Duplicate generated method names are rejected. Implicit receiver-call
syntax and constrained extensions remain post-v1.

Protocols:

```tetra
protocol Renderable:
    func draw(self: Vec2) -> Int

impl Vec2: Renderable
```

The v1 protocols contract accepts declarations with typed required function signatures, validates
referenced types, and exposes them to formatter, generated docs, and LSP symbols.
`impl Type: Protocol` checks that matching extension/static methods exist with compatible
signatures, including effects, async, throws, parameter ownership markers, params, and return type.
Duplicate impls are rejected. Protocol conformance is a static compile-time contract in this MVP: no
witness tables, trait objects, runtime protocol values, existential containers, or dynamic dispatch
model are introduced. Minimal generic protocol requirements are supported using syntax such as
`func map<T>(self: Vec2, value: T) -> T`. Current MVP limits:

- requirement generic parameters are name-only (`<T>`); bounds in requirements are not supported;
- conformance checks require matching generic parameter count/order and compatible signature shape;
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

The current actor declaration subset supports `var`/`val`/`const` fields with state types `Int`,
`Bool`, `UInt8`, `UInt16`, and `task.error`. Field initializers must be compile-time constants.
Unsupported pointer/resource or aggregate actor-state field types are rejected by semantics.

Generic signatures:

```tetra
func id<T>(x: T) -> T:
    return x
```

The v1 generics MVP parses, validates, formats, documents, and statically monomorphizes generic
function calls with inferred value arguments across modules. Generated specialization names are
deterministic and encode fully qualified type names to avoid collisions. No runtime generic value
model, metadata-driven specialization, or dynamic dispatch behavior is introduced. Higher-ranked
generics, explicit type arguments, generic structs, full protocol-bound generic dispatch, and
specialization optimization remain post-v1.

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

| Level | Forms                                                                           | Associativity |
| ----- | ------------------------------------------------------------------------------- | ------------- | --- | ---- |
| 1     | calls, field access, indexing, literals, identifiers, parenthesized expressions | left          |
| 2     | prefix `!`, prefix `-`, `try`, `await`                                          | prefix        |
| 3     | `*`, `/`, `%`                                                                   | left          |
| 4     | `+`, `-`                                                                        | left          |
| 5     | `<`, `<=`, `>`, `>=`                                                            | non-chaining  |
| 6     | `==`, `!=`                                                                      | non-chaining  |
| 7     | `&&`                                                                            | left          |
| 8     | `                                                                               |               | `   | left |

Relational and equality chains such as `a < b < c` and `a == b == c` are diagnostics rather than
implicitly associated expressions.

## Diagnostics And Formatting Contracts

Frontend diagnostics are machine-readable. Parse/frontend diagnostics use `TETRA0001`;
semantic/compiler diagnostics rendered from positioned errors use `TETRA2001`; formatter
preservation diagnostics use `TETRA_FMT001`; formatter check mismatches use `TETRA_FMT002`.
Diagnostics exposed by CLI JSON include `code`, `severity`, `message`, and, when available, `file`,
`line`, and `column`. Formatter check JSON diagnostics report the first differing source position
when formatted output would change the file. UTF-8 input must be valid; invalid byte sequences are
reported before tokenization with an exact byte position.

The default parser entry point remains first-error based: malformed source returns one structured
diagnostic at the earliest reliable location. The diagnostics entry point may perform conservative
top-level declaration recovery for Flow syntax: independent malformed declarations can produce
multiple structured diagnostics while a safe partial AST is returned for declarations that still
parse. Recovery is deterministic, preserves source line numbering, does not reinterpret unsupported
grammar as valid syntax, and stops at unrecoverable file-level errors such as invalid UTF-8.

Formatter guarantees for the supported Flow surface:

- formatting is idempotent;
- indentation is four spaces per block level;
- `uses` clauses are sorted deterministically;
- standalone line comments and block comments are preserved;
- inline comments after code are rejected with a formatter diagnostic instead of being moved or
  discarded;
- malformed files report diagnostics rather than partially formatted output.

## Not In Canonical v1 Surface

`property` and top-level language `capsule` declarations are parser/semantic accepted in the current
profile. `capsule` is metadata-only in this wave (no runtime/ABI integration).

Language `capsule` declarations are distinct from Eco project packaging files (`Capsule.t4` /
`Tetra.capsule`). The former is source-language metadata; the latter is project/package manifest
metadata.

Advanced payload pattern forms beyond the promoted positional enum payload slice, richer ADT
constructors/destructuring, exhaustive integer match checking, collection `for` exhaustiveness
improvements, Callable Level 2 captured-closure semantics, full first-class function-value/callable
matrix, effect polymorphism/inference, protocol-bound generic dispatch, implicit receiver-call
syntax, distributed actors, and structured concurrency are planned for later releases.

These boundaries are negative release guarantees, not undocumented extension points. In v1.0
planning, richer ADT constructors/destructuring, nested payload patterns, protocol runtime values,
trait objects/witness tables, dynamic dispatch through protocols, captured closure movement, and
broad first-class function-value storage must continue to fail with explicit diagnostics until a
future plan promotes them with code, docs, tests, and release evidence.
