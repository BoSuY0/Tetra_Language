# Tetra Bugs

Status: fixed bug ledger for microservice-driven Tetra language testing.

Updated: 2026-05-20. All entries below are fixed and verified by promoted
regressions plus the microservice bug ledger. The final coverage set is
recorded in the 2026-05-20 closure row under Microservice Bug-Hunt Runs.

## Confirmed Language Bugs

### TETRA-BUG-0001: Generic inference fails for direct function-call arguments

- Status: fixed, verified.
- Area: compiler / generic monomorphization.
- Found while creating: `examples/microservices/compiler_pipeline_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/generic_call_result.tetra" <<'EOF'
func id<T>(x: T) -> T:
    return x

func value() -> Int:
    return 42

func main() -> Int:
    return id(value())
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/generic_call_result.tetra"
```

- Expected: `id(value())` infers `T = Int`, matching the documented current
  surface for generic functions with inferred value arguments.
- Actual: build fails with `cannot infer generic argument for 'id' arg 1`.
- Control case: assigning the call result first works:

```tetra
let v: Int = value()
return id(v)
```

- Workaround used in microservice examples: bind function-call results to a
  typed local before passing them into generic functions.

### TETRA-BUG-0002: Same-module extension static call is unresolved inside a named module

- Status: fixed, verified.
- Area: compiler / module-aware extension method resolution.
- Found while creating:
  `examples/microservices/compiler_artifact_router_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
mkdir -p "$tmp/app"
cat > "$tmp/app/main.tetra" <<'EOF'
module app.main

struct Unit:
    value: Int

protocol Score:
    func score(self: Unit) -> Int

extension Unit:
    func score(self: Unit) -> Int:
        return self.value

impl Unit: Score

func main() -> Int:
    let unit: Unit = Unit(value: 42)
    return Unit.score(unit)
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/out" "$tmp/app/main.tetra"
```

- Expected: `Unit.score(unit)` resolves the same-module extension static method
  in `module app.main`, matching the supported static protocol conformance
  surface.
- Actual: build fails with `unknown function 'Unit.score'`.
- Control case: removing the `module app.main` declaration allows the same
  static extension call to build and run.
- Workaround used in microservice examples: keep same-file static extension
  dispatch examples outside a named module until module-aware resolution is
  fixed.

### TETRA-BUG-0003: Function-typed struct field cannot directly initialize enum payload

- Status: fixed, verified.
- Area: compiler / callable metadata propagation for enum payload constructors.
- Found while creating: `examples/microservices/callable_router_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/callable_route_field.tetra" <<'EOF'
struct Handler:
    cb: fn(Int) -> Int

enum Route:
    case direct(fn(Int) -> Int)
    case fallback

func main() -> Int:
    let offset: Int = 13
    let captured: ptr = fn(value: Int) -> Int:
        return value + offset
    let handler: Handler = Handler(cb: captured)
    let route: Route = Route.direct(handler.cb)
    match route:
    case Route.direct(route_cb):
        return route_cb(29)
    case Route.fallback:
        return 0
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/callable_route_field.tetra"
```

- Expected: immutable function-typed struct field metadata propagates directly
  into the `Route.direct(...)` enum payload constructor.
- Actual: build fails with `function-typed local 'Route.direct[1]' parameter
  count mismatch: expected 1, got 2`.
- Control case: `Route.direct(captured)` builds, and assigning `handler.cb` to
  a typed local first also builds:

```tetra
let field_cb: fn(Int) -> Int = handler.cb
let route: Route = Route.direct(field_cb)
```

- Workaround: aliasing the function-typed struct field into a typed local works
  in unformatted source, but see `TETRA-BUG-0004` for a formatter interaction.
  Current microservice examples route the struct field through a callback
  parameter and initialize the enum payload from the original captured callable.

### TETRA-BUG-0004: Formatter drops function-typed local annotations needed for callable metadata

- Status: fixed, verified.
- Area: formatter / callable metadata preservation.
- Found while creating: `examples/microservices/callable_router_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/callable_alias.tetra" <<'EOF'
struct Handler:
    cb: fn(Int) -> Int

enum Route:
    case direct(fn(Int) -> Int)
    case fallback

func main() -> Int:
    let offset: Int = 13
    let captured: ptr = fn(value: Int) -> Int:
        return value + offset
    let handler: Handler = Handler(cb: captured)
    let field_cb: fn(Int) -> Int = handler.cb
    let route: Route = Route.direct(field_cb)
    match route:
    case Route.direct(route_cb):
        return route_cb(29)
    case Route.fallback:
        return 0
EOF
go run ./cli/cmd/tetra fmt -write "$tmp/callable_alias.tetra"
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/callable_alias.tetra"
```

- Expected: formatter preserves `let field_cb: fn(Int) -> Int = handler.cb`,
  and the formatted file still builds.
- Actual: formatter rewrites the binding to `let field_cb = handler.cb`; the
  build then fails with `function-typed local 'Route.direct[1]' initializer
  must be a symbol-backed function value, target-set-backed function value,
  direct named function symbol, or closure literal for the supported fnptr ABI`.
- Control case: the unformatted source with the explicit function type builds.
- Workaround used in microservice examples: avoid relying on formatted typed
  aliases for function-typed struct fields before enum payload construction.
  The same workaround applies to formatted function-typed return aliases; pass
  the return call directly to a supported callback parameter when possible.

### TETRA-BUG-0005: Actor entrypoint strings do not resolve inside named modules

- Status: fixed, verified.
- Area: compiler / module-aware actor entrypoint resolution.
- Found while creating: `examples/microservices/actor_deadline_router_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
mkdir -p "$tmp/app"
cat > "$tmp/app/main.tetra" <<'EOF'
module app.main

actor Router:
    func run() -> Int
    uses actors:
        let _sent: Int = core.send_msg(core.sender(), 42, 7)
        return 0

func main() -> Int
uses actors:
    let router: actor = core.spawn("Router.run")
    let _request: Int = core.send_msg(router, 41, 6)
    let reply: actor.msg = core.recv_msg()
    return reply.value
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app-bin" "$tmp/app/main.tetra"
```

- Expected: `core.spawn("Router.run")` resolves a same-module actor method in
  `module app.main`, matching the non-module actor service surface.
- Actual: build fails with `unknown function 'Router.run'`.
- Control case: removing the `module app.main` declaration allows the same
  actor entrypoint string to build.
- Workaround used in microservice examples: keep same-file actor entrypoint
  string examples outside named modules until module-aware actor resolution is
  fixed.

### TETRA-BUG-0006: Formatter drops function-typed global annotations required by the compiler

- Status: fixed, verified.
- Area: formatter / function-typed global declarations.
- Found while creating:
  `examples/microservices/callable_global_router_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/global_fn_format.tetra" <<'EOF'
var callback: fn(Int) -> Int = identity

func identity(value: Int) -> Int:
    return value

func main() -> Int:
    return callback(42)
EOF
go run ./cli/cmd/tetra fmt -write "$tmp/global_fn_format.tetra"
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/global_fn_format.tetra"
```

- Expected: formatter preserves `var callback: fn(Int) -> Int = identity`
  because global `var` declarations require an explicit type annotation.
- Actual: formatter rewrites the declaration to `var callback = identity`; the
  build then fails with `global var requires an explicit type annotation`.
- Control case: the unformatted source with the explicit function type builds.
- Workaround used in microservice examples: avoid mutable function-typed global
  declarations in formatted microservice sources until the formatter preserves
  required global annotations.

### TETRA-BUG-0007: Derived pointer arithmetic loses allocation provenance

- Status: fixed, verified.
- Area: runtime memory / derived pointer provenance.
- Found while creating:
  `examples/microservices/memory_copy_window_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/derived_ptr_add.tetra" <<'EOF'
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let memory_cap = core.cap_mem()
        let base: ptr = core.alloc_bytes(16)
        let dst: ptr = core.ptr_add(base, 8, memory_cap)
        let dst_one: ptr = core.ptr_add(dst, 1, memory_cap)
        let _write: u8 = core.store_u8(dst_one, 42, memory_cap)
        return core.load_u8(dst_one, memory_cap)
    return 99
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/derived_ptr_add.tetra"
"$tmp/app"
```

- Expected: `core.ptr_add` either composes derived pointers within the original
  allocation (`base + 8 + 1`) or the checker rejects derived-pointer
  composition before runtime.
- Actual: the program builds, then exits with status `2` when the runtime guard
  treats the derived pointer as a fresh allocation base and reads the wrong
  metadata header.
- Control case: using one visible base offset works:

```tetra
let dst_one: ptr = core.ptr_add(base, 9, memory_cap)
```

- Workaround used in microservice examples: keep pointer arithmetic anchored at
  the original `core.alloc_bytes` base pointer, or pass separate allocation-base
  pointers into `lib.core.memory` helpers.

### TETRA-BUG-0008: Formatter rewrites mutable actor state fields as immutable

- Status: fixed, verified.
- Area: formatter / actor state declarations.
- Found while creating:
  `examples/microservices/actor_state_counter_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/actor_state_var_format.tetra" <<'EOF'
actor Counter:
    var count: Int = 0
    func run() -> Int
    uses actors:
        count = count + 1
        let _reply: Int = core.send_msg(core.sender(), count, 1)
        return 0

func main() -> Int
uses actors:
    let counter: actor = core.spawn("Counter.run")
    let _request: Int = core.send_msg(counter, 0, 0)
    let reply: actor.msg = core.recv_msg()
    return reply.value
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/unformatted" "$tmp/actor_state_var_format.tetra"
go run ./cli/cmd/tetra fmt -write "$tmp/actor_state_var_format.tetra"
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/formatted" "$tmp/actor_state_var_format.tetra"
```

- Expected: formatter preserves `var count: Int = 0` inside the `actor`
  declaration, so the formatted source still builds.
- Actual: formatter rewrites the actor state field to `val count: Int = 0`;
  the formatted build then fails with `cannot assign to val 'count'`.
- Control case: the unformatted source with `var count` builds.
- Workaround used in microservice examples: keep mutable actor counters as local
  variables in formatted examples, or avoid running the formatter on mutable
  actor state files until the declaration kind is preserved.

### TETRA-BUG-0009: Blocking tagged receive fails in dual actor fan-in

- Status: fixed, verified.
- Area: actor runtime / blocking tagged receive scheduler path.
- Found while creating:
  `examples/microservices/actor_dual_mailbox_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/dual_blocking_recv_msg.tetra" <<'EOF'
actor Adder:
    func run() -> Int
    uses actors:
        let request: actor.msg = core.recv_msg()
        let _reply: Int = core.send_msg(core.sender(), request.value + 1, request.tag + 10)
        return 0

actor Multiplier:
    func run() -> Int
    uses actors:
        let request: actor.msg = core.recv_msg()
        let _reply: Int = core.send_msg(core.sender(), request.value * 2, request.tag + 20)
        return 0

func main() -> Int
uses actors:
    let adder: actor = core.spawn("Adder.run")
    let multiplier: actor = core.spawn("Multiplier.run")
    let _left: Int = core.send_msg(adder, 20, 1)
    let _right: Int = core.send_msg(multiplier, 21, 2)
    let first: actor.msg = core.recv_msg()
    let second: actor.msg = core.recv_msg()
    let value_total: Int = first.value + second.value
    let tag_total: Int = first.tag + second.tag
    if tag_total != 33:
        return tag_total
    if value_total == 63:
        return 0
    return value_total
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/dual_blocking_recv_msg.tetra"
"$tmp/app"
```

- Expected: two actors can both receive tagged requests and reply to the main
  actor; blocking `core.recv_msg()` in `main` waits for both replies, matching
  the single-actor blocking receive surface.
- Actual: the program builds, then exits with status `1` on Linux-x64.
- Control cases: a single tagged actor with blocking `core.recv_msg()` exits
  `0`; replacing the two blocking receives with
  `core.recv_msg_until(core.deadline_ms(5))` also exits `0`.
- Workaround used in microservice examples: use deadline-aware tagged receives
  for multi-actor fan-in until the blocking scheduler path is fixed.

### TETRA-BUG-0010: Blocking value receive fails in dual actor fan-in

- Status: fixed, verified.
- Area: actor runtime / blocking value receive scheduler path.
- Found while creating:
  `examples/microservices/actor_dual_value_mailbox_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/dual_blocking_recv.tetra" <<'EOF'
func adder() -> Int
uses actors:
    let request: Int = core.recv()
    let _reply: Int = core.send(core.sender(), request + 1)
    return 0

func multiplier() -> Int
uses actors:
    let request: Int = core.recv()
    let _reply: Int = core.send(core.sender(), request * 2)
    return 0

func main() -> Int
uses actors:
    let adder_actor: actor = core.spawn("adder")
    let multiplier_actor: actor = core.spawn("multiplier")
    let _left: Int = core.send(adder_actor, 20)
    let _right: Int = core.send(multiplier_actor, 21)
    let first: Int = core.recv()
    let second: Int = core.recv()
    let total: Int = first + second
    if total == 63:
        return 0
    return total
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/dual_blocking_recv.tetra"
"$tmp/app"
```

- Expected: two actors can both receive value messages and reply to the main
  actor; blocking `core.recv()` in `main` waits for both replies, matching the
  single-actor value receive surface.
- Actual: the program builds, then exits with status `1` on Linux-x64.
- Control cases: a single value actor with blocking `core.recv()` exits `0`;
  replacing the two blocking receives with `core.recv_until(core.deadline_ms(5))`
  exits `0`.
- Workaround used in microservice examples: use deadline-aware value receives
  for multi-actor fan-in until the blocking scheduler path is fixed.

### TETRA-BUG-0011: Struct constructors do not wrap scalar values into optional fields

- Status: fixed, verified.
- Area: compiler / struct constructor optional coercion.
- Found while creating:
  `examples/microservices/compiler_optional_box_pack/app/main.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/optional_field_constructor.tetra" <<'EOF'
struct MaybeBox:
    value: Int?

func filled(value: Int) -> MaybeBox:
    return MaybeBox(value: value)

func main() -> Int:
    let box: MaybeBox = filled(42)
    if let value = box.value:
        return value
    return 0
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/optional_field_constructor.tetra"
```

- Expected: struct constructor field checking applies the same optional
  coercion accepted by `let maybe: Int? = value`, so `MaybeBox(value: value)`
  initializes `value: Int?`.
- Actual: build fails with `slot mismatch for field 'value'`.
- Control case: explicitly binding through an optional local builds and exits
  `42`:

```tetra
let maybe: Int? = value
return MaybeBox(value: maybe)
```

- Workaround used in microservice examples: introduce an explicitly typed
  optional local before constructing a struct field of optional type.

### TETRA-BUG-0012: Enum constructors do not wrap scalar values into optional payloads

- Status: fixed, verified.
- Area: compiler / enum constructor optional coercion.
- Found while creating:
  `examples/microservices/optional_enum_router_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/optional_enum_constructor.tetra" <<'EOF'
enum Route:
    case ready(Int?)
    case empty

func make_ready(value: Int) -> Route:
    return Route.ready(value)

func score(route: Route) -> Int:
    match route:
    case Route.ready(maybe):
        if let value = maybe:
            return value
        return 0
    case Route.empty:
        return 0

func main() -> Int:
    let route: Route = make_ready(42)
    return score(route)
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/optional_enum_constructor.tetra"
```

- Expected: enum constructor payload checking applies the same optional
  coercion accepted by `let maybe: Int? = value`, so `Route.ready(value)`
  initializes an `Int?` payload.
- Actual: build fails with `enum case 'Route.ready' payload 1 slot mismatch`.
- Control case: explicitly binding through an optional local builds:

```tetra
let maybe: Int? = value
return Route.ready(maybe)
```

- Workaround used in microservice examples: introduce an explicitly typed
  optional local before constructing an enum case payload of optional type.

### TETRA-BUG-0013: Derived pointer loop arithmetic fails after pointer parameters

- Status: fixed, verified.
- Area: runtime memory / derived pointer provenance across function calls.
- Found while creating:
  `examples/microservices/memory_derived_copy_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/derived_ptr_param_loop.tetra" <<'EOF'
func copy_loop(dst: ptr, src: ptr, n: Int) -> Int
uses capability, mem:
    unsafe:
        let memory_cap = core.cap_mem()
        var i: Int = 0
        while i < n:
            let sp: ptr = core.ptr_add(src, i, memory_cap)
            let dp: ptr = core.ptr_add(dst, i, memory_cap)
            let b: u8 = core.load_u8(sp, memory_cap)
            let _: u8 = core.store_u8(dp, b, memory_cap)
            i = i + 1
        return 0
    return 99

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let memory_cap = core.cap_mem()
        let source: ptr = core.alloc_bytes(8)
        let target: ptr = core.alloc_bytes(8)
        let source_two: ptr = core.ptr_add(source, 2, memory_cap)
        let source_three: ptr = core.ptr_add(source, 3, memory_cap)
        let target_one: ptr = core.ptr_add(target, 1, memory_cap)
        let target_two: ptr = core.ptr_add(target, 2, memory_cap)
        let _write_two: u8 = core.store_u8(source_two, 20, memory_cap)
        let _write_three: u8 = core.store_u8(source_three, 22, memory_cap)
        let _copy: Int = copy_loop(target_one, source_two, 2)
        return core.load_u8(target_one, memory_cap) + core.load_u8(target_two, memory_cap)
    return 98
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/derived_ptr_param_loop.tetra"
"$tmp/app"
```

- Expected: the helper copies `20` and `22`, then exits `42`.
- Actual: the program builds but exits `2`, showing the looped `core.ptr_add`
  over derived pointer parameters does not preserve the intended provenance /
  offset semantics.
- Control cases: direct `core.load_u8` through a derived pointer parameter works,
  and an unrolled copy using explicit derived pointers works. Allocation-base
  pointer parameters remain safe through deferred memory writes during typed
  error unwind and early return in
  `memory_defer_throw_base_store_service.tetra` and
  `memory_defer_return_base_store_service.tetra`.
- Additional failing evidence: passing a derived cell pointer into a helper that
  performs the `core.store_i32` from a `defer` cleanup while unwinding a typed
  error builds but exits `2`; the allocation-base version above exits `0`.
- Workaround used in microservice examples: keep helper loops anchored at the
  allocation base, or unroll fixed derived-pointer copies with direct
  `core.load_u8` / `core.store_u8` operations.

### TETRA-BUG-0014: Formatter drops generic protocol requirement type parameters

- Status: fixed, verified.
- Area: formatter / generic protocol requirements.
- Found while creating:
  `examples/microservices/compiler_generic_extension_pack/app/main.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/generic_protocol_fmt.tetra" <<'EOF'
struct Vec2:
    x: Int

protocol Mapper:
    func map<T>(self: Vec2, value: T) -> T
EOF
go run ./cli/cmd/tetra fmt -write "$tmp/generic_protocol_fmt.tetra"
sed -n '1,8p' "$tmp/generic_protocol_fmt.tetra"
```

- Expected: formatter preserves `func map<T>(...) -> T`.
- Actual: formatter rewrites the requirement to
  `func map(self: Vec2, value: T) -> T`, leaving `T` unbound.
- Workaround used in microservice examples: avoid formatted generic protocol
  requirement sources in microservice packs; use non-generic protocol
  requirements for extension-conformance probes until formatter preservation is
  fixed.

### TETRA-BUG-0015: Imported generic extension static calls do not monomorphize

- Status: fixed, verified.
- Area: compiler / imported generic extension monomorphization.
- Found while creating:
  `examples/microservices/compiler_generic_extension_pack/app/main.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
mkdir -p "$tmp/engine" "$tmp/app"
cat > "$tmp/engine/core.tetra" <<'EOF'
module engine.core

struct Vec2:
    x: Int
EOF
cat > "$tmp/app/ext.tetra" <<'EOF'
module app.ext

import engine.core as core

extension core.Vec2:
    func map<T>(self: core.Vec2, value: T) -> T:
        return value
EOF
cat > "$tmp/app/main.tetra" <<'EOF'
module app.main

import app.ext as ext
import engine.core as core

func main() -> Int:
    let value: core.Vec2 = core.Vec2(x: 7)
    return core.Vec2.map(value, 42)
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app-bin" "$tmp/app/main.tetra"
```

- Expected: imported `core.Vec2.map(value, 42)` monomorphizes with `T = Int`,
  matching same-file generic extension static calls.
- Actual: build fails with `generic function 'core.Vec2.map' could not be
  monomorphized; use inferable value arguments`.
- Control case: the same generic extension and static call build when declared
  in one source file without module import boundaries.
- Workaround used in microservice examples: use non-generic imported extension
  methods for protocol-backed extension dispatch probes.

### TETRA-BUG-0016: Match case payload bindings leak across sibling cases

- Status: fixed, verified.
- Area: compiler / match payload binding scope.
- Found while creating:
  `examples/microservices/actor_typed_chain_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/match_case_duplicate_payload.tetra" <<'EOF'
enum Msg:
    case left(Int)
    case right(Int)

func main() -> Int:
    let msg: Msg = Msg.left(42)
    match msg:
    case Msg.left(value):
        return value
    case Msg.right(value):
        return value
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/match_case_duplicate_payload.tetra"
```

- Expected: payload bindings are scoped to their own `case` arm, so reusing
  `value` in sibling arms is accepted.
- Actual: build fails with `duplicate local 'value'`.
- Workaround used in microservice examples: use unique payload binding names in
  every `match` / typed `catch` case arm.
- Additional failing evidence: scoped pattern names remain reserved even after
  the expression or branch exits. Two separate `match` expressions that both
  bind `value`, two separate typed `catch` expressions that both bind `code`,
  or two separate `if let value = ...` branches in one function all fail with
  `duplicate local`, even though existing scope diagnostics reject using those
  names after the case/branch.
- Additional failing evidence: cross-module optional enum resource routing hits
  the same reservation leak. Reusing `handle` in
  `case resources.ActorSlot.ready(handle)` and later
  `case resources.TaskSlot.ready(handle)` inside
  `compiler_optional_enum_resource_pack/app/main.tetra` failed with
  `duplicate local 'handle'` until the passing microservice used distinct
  payload binding names.

### TETRA-BUG-0017: Stored derived pointers lose loadable memory provenance

- Status: fixed, verified.
- Area: runtime memory / pointer table provenance.
- Found while creating:
  `examples/microservices/memory_derived_ptr_table_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/derived_ptr_table.tetra" <<'EOF'
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let memory_cap = core.cap_mem()
        let payload: ptr = core.alloc_bytes(8)
        let table: ptr = core.alloc_bytes(8)
        let payload_four: ptr = core.ptr_add(payload, 4, memory_cap)
        let _stored_value: Int = core.store_i32(payload_four, 42, memory_cap)
        let _stored_ptr: ptr = core.store_ptr(table, payload_four, memory_cap)
        let loaded: ptr = core.load_ptr(table, memory_cap)
        return core.load_i32(loaded, memory_cap)
    return 99
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/derived_ptr_table.tetra"
"$tmp/app"
```

- Expected: storing and loading a derived pointer preserves enough provenance to
  load the pointed-to cell, so the program exits `42`.
- Actual: the program builds but exits `2`, matching the runtime memory guard's
  invalid access path.
- Control cases: storing/loading an allocation-base pointer exits `42`, and
  storing the base pointer then applying `core.ptr_add(loaded, 4, mem)` exits
  `42`.
- Workaround used in microservice examples: store allocation-base pointers in
  raw pointer tables and re-derive offsets after loading the table entry.

### TETRA-BUG-0018: Struct pointer fields lose memory provenance

- Status: fixed, verified.
- Area: runtime memory / struct pointer-field provenance.
- Found while creating:
  `examples/microservices/memory_aggregate_ptr_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/struct_base_ptr.tetra" <<'EOF'
struct Box:
    base: ptr

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let memory_cap = core.cap_mem()
        let payload: ptr = core.alloc_bytes(8)
        let cell: ptr = core.ptr_add(payload, 4, memory_cap)
        let _write: Int = core.store_i32(cell, 42, memory_cap)
        let box: Box = Box(base: payload)
        let loaded_cell: ptr = core.ptr_add(box.base, 4, memory_cap)
        return core.load_i32(loaded_cell, memory_cap)
    return 99
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/struct_base_ptr.tetra"
"$tmp/app"
```

- Expected: carrying an allocation-base pointer through a struct field preserves
  enough provenance for `core.ptr_add(box.base, 4, mem)` and `load_i32` to exit
  `42`.
- Actual: the program builds but exits `2`, matching the runtime memory guard's
  invalid access path.
- Additional failing evidence: monomorphized generic `Box<ptr>`, optional
  `PtrBox?`, optional `Box<ptr>?`, and an imported pointer helper that returns
  `Box<ptr>` all build, but deriving a cell from the boxed allocation-base
  pointer exits `2`.
- Additional failing evidence: returning `PtrBox(raw: base)` through a typed
  error payload also exits `2` when deriving from `recovered_box.raw`, matching
  the same struct pointer-field provenance loss.
- Control cases: a plain local alias of the same base pointer exits `42`;
  a non-generic helper alias exits `0` in
  `memory_ptr_alias_base_service.tetra`; same-module and imported generic
  identity over an allocation-base `ptr` exit `0` in
  `memory_ptr_generic_identity_base_service.tetra` and
  `compiler_ptr_generic_pack/app/main.tetra`; enum base-pointer payloads exit
  `0` in `memory_aggregate_ptr_service.tetra`; optional typed-error
  allocation-base pointer payloads exit `0` in
  `memory_typed_error_optional_ptr_base_service.tetra`,
  `memory_typed_error_optional_ptr_dynamic_service.tetra`, and
  `compiler_typed_error_optional_ptr_pack/app/main.tetra`.
- Workaround used in microservice examples: keep raw allocation-base pointers in
  locals, or use an enum base-pointer payload when an aggregate route is needed.

### TETRA-BUG-0019: Enum derived pointer payloads lose memory provenance

- Status: fixed, verified.
- Area: runtime memory / enum pointer-payload provenance.
- Found while creating:
  `examples/microservices/memory_aggregate_ptr_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/enum_derived_ptr.tetra" <<'EOF'
enum Cell:
    case raw(ptr)
    case empty

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let memory_cap = core.cap_mem()
        let payload: ptr = core.alloc_bytes(8)
        let payload_four: ptr = core.ptr_add(payload, 4, memory_cap)
        let _stored_value: Int = core.store_i32(payload_four, 42, memory_cap)
        let cell: Cell = Cell.raw(payload_four)
        match cell:
        case Cell.raw(raw_ptr):
            return core.load_i32(raw_ptr, memory_cap)
        case Cell.empty:
            return 1
    return 99
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/enum_derived_ptr.tetra"
"$tmp/app"
```

- Expected: carrying a derived pointer through an enum payload preserves enough
  provenance for the matched `raw_ptr` load to exit `42`.
- Actual: the program builds but exits `2`, matching the runtime memory guard's
  invalid access path.
- Control case: carrying the allocation-base pointer through the enum payload,
  then applying `core.ptr_add(route_base, 4, mem)` after matching, exits `42`.
- Workaround used in microservice examples: carry allocation-base pointers
  through enum payloads and re-derive offsets after matching.

### TETRA-BUG-0020: Generic identity over function-typed locals lowers to an unknown fn type

- Status: fixed, verified.
- Area: compiler / generic monomorphization / function-typed values.
- Found while creating:
  `examples/microservices/compiler_callable_generic_route_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/generic_fn_identity.tetra" <<'EOF'
func add_one(value: Int) -> Int:
    return value + 1

func id<T>(value: T) -> T:
    return value

func apply(cb: fn(Int) -> Int, value: Int) -> Int:
    return cb(value)

func main() -> Int:
    let handler: fn(Int) -> Int = add_one
    let routed: fn(Int) -> Int = id(handler)
    return apply(routed, 41)
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/generic_fn_identity.tetra"
```

- Expected: `id(handler)` monomorphizes for the supported `fn(Int) -> Int`
  value and the program builds, matching generic identity over scalar values and
  non-generic identity over function-typed values.
- Actual: build fails with `unknown type 'fn(i32)->i32'`.
- Control cases: `id(42)` builds and exits `42`; a non-generic
  `id_fn(value: fn(Int) -> Int) -> fn(Int) -> Int` builds and exits `42`.
- Workaround used in microservice examples: route scalar work through the
  generic helper and pass function-typed values through non-generic callback
  parameters.

### TETRA-BUG-0021: Optional derived pointer payloads lose memory provenance

- Status: fixed, verified.
- Area: runtime memory / optional pointer-payload provenance.
- Found while creating:
  `examples/microservices/memory_optional_ptr_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/optional_derived_ptr.tetra" <<'EOF'
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let memory_cap = core.cap_mem()
        let payload: ptr = core.alloc_bytes(8)
        let payload_four: ptr = core.ptr_add(payload, 4, memory_cap)
        let _stored_value: Int = core.store_i32(payload_four, 42, memory_cap)
        let maybe_cell: ptr? = payload_four
        if let cell = maybe_cell:
            return core.load_i32(cell, memory_cap)
        return 98
    return 99
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/optional_derived_ptr.tetra"
"$tmp/app"
```

- Expected: carrying a derived pointer through an optional payload preserves
  enough provenance for the unwrapped `cell` load to exit `42`.
- Actual: the program builds but exits `2`, matching the runtime memory guard's
  invalid access path.
- Control case: carrying the allocation-base pointer through the optional, then
  applying `core.ptr_add(base, 4, mem)` after unwrapping, exits `42`.
- Workaround used in microservice examples: carry allocation-base pointers
  through optional payloads and re-derive offsets after unwrapping.

### TETRA-BUG-0022: Generic callback parameters do not accept compatible function symbols

- Status: fixed, verified.
- Area: compiler / generic monomorphization / callback parameters.
- Found while creating:
  `examples/microservices/compiler_callable_generic_route_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/generic_callback.tetra" <<'EOF'
func add_one(value: Int) -> Int:
    return value + 1

func apply_generic<T>(cb: fn(T) -> T, value: T) -> T:
    return cb(value)

func main() -> Int:
    return apply_generic(add_one, 41)
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/generic_callback.tetra"
```

- Expected: `T` is inferred as `Int`, and `add_one` is accepted as a compatible
  `fn(Int) -> Int` callback for `fn(T) -> T`.
- Actual: build fails with `cannot infer generic argument for 'apply_generic'
  arg 1`.
- Control cases: a non-generic `apply_int(cb: fn(Int) -> Int, value: Int)`
  builds and exits `42`; scalar generic identity over `Int` builds and exits
  `42`.
- Additional failing control: putting the value first as
  `apply_generic<T>(value: T, cb: fn(T) -> T)` still fails on the callback
  argument with `cannot infer generic argument`.
- Additional failing evidence: generic callback inference also fails for
  optional pointer callbacks. `apply_generic<T>(cb: fn(T?) -> T?, value: T?)`
  called with `keep_optional(value: ptr?) -> ptr?` and a `ptr?` argument fails
  with `cannot infer generic argument for 'apply_generic' arg 1`, while a
  generic identity over the same `ptr?` value exits `0`.
- Workaround used in microservice examples: keep callback parameters
  non-generic and route scalar values through separate generic helpers.

### TETRA-BUG-0023: Function returns of derived pointers lose memory provenance

- Status: fixed, verified.
- Area: runtime memory / function return pointer provenance.
- Found while creating:
  `examples/microservices/memory_function_ptr_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/function_derived_ptr_return.tetra" <<'EOF'
func keep_ptr(value: ptr) -> ptr:
    return value

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem = core.cap_mem()
        let payload: ptr = core.alloc_bytes(8)
        let cell: ptr = core.ptr_add(payload, 4, mem)
        let _write: Int = core.store_i32(cell, 42, mem)
        let kept: ptr = keep_ptr(cell)
        return core.load_i32(kept, mem)
    return 99
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/function_derived_ptr_return.tetra"
"$tmp/app"
```

- Expected: returning a derived pointer from a function preserves enough
  provenance for `core.load_i32(kept, mem)` to exit `42`.
- Actual: the program builds but exits `2`, matching the runtime memory guard's
  invalid access path.
- Control cases: returning the allocation-base pointer from the same helper,
  then deriving the offset after the call, exits `42`; generic identity over
  the allocation-base pointer also exits `42`.
- Additional failing variant: routing the derived pointer through a
  function-typed callable (`let keep: fn(ptr) -> ptr = fn(value: ptr) -> ptr:
  return value`) or through `apply(cb: fn(ptr) -> ptr, value: ptr) -> ptr`
  also builds but exits `2`. Function-typed callable returns of derived pointers hit the same guard,
  while the allocation-base callable control exits `42`.
- Workaround used in microservice examples: return allocation-base pointers
  through function or generic helpers and derive raw-memory offsets only after
  the pointer is back in the caller.

### TETRA-BUG-0024: Global pointer variables lose memory provenance

- Status: fixed, verified.
- Area: runtime memory / global pointer provenance.
- Found while creating:
  `examples/microservices/memory_global_state_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/global_base_ptr.tetra" <<'EOF'
var saved: ptr = 0

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem = core.cap_mem()
        let payload: ptr = core.alloc_bytes(8)
        let cell: ptr = core.ptr_add(payload, 4, mem)
        let _write: Int = core.store_i32(cell, 42, mem)
        saved = payload
        let loaded_cell: ptr = core.ptr_add(saved, 4, mem)
        return core.load_i32(loaded_cell, mem)
    return 99
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/global_base_ptr.tetra"
"$tmp/app"
```

- Expected: storing an allocation-base pointer in a global `ptr` variable
  preserves enough provenance for deriving the offset after loading the global,
  so the program exits `42`.
- Actual: the program builds but exits `2`, matching the runtime memory guard's
  invalid access path.
- Control cases: a local `ptr` alias of the same allocation-base pointer exits
  `42`; a global `Int` assignment/read exits `42`.
- Workaround used in microservice examples: keep raw pointers in locals or
  function parameters, and store only scalar service state in globals.

### TETRA-BUG-0025: Global integer offsets break raw pointer arithmetic provenance

- Status: fixed, verified.
- Area: runtime memory / pointer arithmetic with global scalar operands.
- Found while creating:
  `examples/microservices/memory_global_state_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/global_offset_direct.tetra" <<'EOF'
var cell_offset: Int = 4

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem = core.cap_mem()
        let payload: ptr = core.alloc_bytes(8)
        let cell: ptr = core.ptr_add(payload, cell_offset, mem)
        let _write: Int = core.store_i32(cell, 42, mem)
        return core.load_i32(cell, mem)
    return 99
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/global_offset_direct.tetra"
"$tmp/app"
```

- Expected: using a global scalar `Int` offset with value `4` in `core.ptr_add`
  behaves like the same local or literal offset, so the program exits `42`.
- Actual: the program builds but exits `2`, matching the runtime memory guard's
  invalid access path.
- Control cases: using the literal offset `4` exits `42`; global scalar
  assignment/read outside pointer arithmetic exits `42`.
- Workaround used in microservice examples: use local or literal offsets for
  raw pointer arithmetic, and avoid global scalar operands in `core.ptr_add`.

### TETRA-BUG-0026: Mutable local derived pointer variables lose memory provenance

- Status: fixed, verified.
- Area: runtime memory / mutable local pointer provenance.
- Found while creating:
  `examples/microservices/memory_mutable_ptr_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/local_var_derived_ptr.tetra" <<'EOF'
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem = core.cap_mem()
        let payload: ptr = core.alloc_bytes(8)
        let cell: ptr = core.ptr_add(payload, 4, mem)
        let _write: Int = core.store_i32(cell, 42, mem)
        var saved: ptr = 0
        saved = cell
        return core.load_i32(saved, mem)
    return 99
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/local_var_derived_ptr.tetra"
"$tmp/app"
```

- Expected: assigning a derived pointer into a mutable local `ptr` preserves
  enough provenance for `core.load_i32(saved, mem)` to exit `42`.
- Actual: the program builds but exits `2`, matching the runtime memory guard's
  invalid access path.
- Control cases: assigning an allocation-base pointer into the same mutable
  local and deriving the offset after assignment exits `42`; initializing the
  mutable local directly from an allocation-base pointer exits `42`.
- Workaround used in microservice examples: store allocation-base pointers in
  mutable locals and derive offsets after selecting the active base pointer.

### TETRA-BUG-0027: Struct field offsets break raw pointer arithmetic provenance

- Status: fixed, verified.
- Area: runtime memory / pointer arithmetic with struct-field scalar operands.
- Found while creating:
  `examples/microservices/memory_struct_offset_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/struct_offset.tetra" <<'EOF'
struct Config:
    offset: Int

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem = core.cap_mem()
        let payload: ptr = core.alloc_bytes(8)
        let config: Config = Config(offset: 4)
        let cell: ptr = core.ptr_add(payload, config.offset, mem)
        let _write: Int = core.store_i32(cell, 42, mem)
        return core.load_i32(cell, mem)
    return 99
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/struct_offset.tetra"
"$tmp/app"
```

- Expected: using `config.offset` with value `4` in `core.ptr_add` behaves like
  an equivalent local `Int` offset, so the program exits `42`.
- Actual: the program builds but exits `2`, matching the runtime memory guard's
  invalid access path.
- Additional failing evidence: a typed actor struct payload field behaves the
  same way. Receiving `OffsetMsg.reply(OffsetBox(offset: 4))` and using
  `reply_box.offset` directly as the `core.ptr_add` offset builds but exits
  `2`.
- Control cases: assigning `config.offset` to a local `Int` before `ptr_add`
  exits `42`; enum and optional `Int` offset payloads exit `42`; direct
  non-memory reads of `config.offset` exit `42`; assigning the received typed
  actor struct payload field to a local `Int` first exits `0` in
  `memory_actor_typed_struct_payload_offset_service.tetra` and
  `compiler_typed_actor_payload_memory_pack/app/main.tetra`.
- Workaround used in microservice examples: copy struct field offsets into a
  local `Int` before calling `core.ptr_add`.

### TETRA-BUG-0028: Function-call offset operands break raw pointer arithmetic provenance

- Status: fixed, verified.
- Area: runtime memory / pointer arithmetic with call-expression operands.
- Found while creating:
  `examples/microservices/memory_function_offset_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/function_return_offset.tetra" <<'EOF'
func offset() -> Int:
    return 4

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem = core.cap_mem()
        let payload: ptr = core.alloc_bytes(8)
        let cell: ptr = core.ptr_add(payload, offset(), mem)
        let _write: Int = core.store_i32(cell, 42, mem)
        return core.load_i32(cell, mem)
    return 99
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/function_return_offset.tetra"
"$tmp/app"
```

- Expected: using a function call that returns `4` as the offset operand in
  `core.ptr_add` behaves like a local `Int` with value `4`, so the program
  exits `42`.
- Actual: the program builds but exits `2`, matching the runtime memory guard's
  invalid access path.
- Control cases: assigning `offset()` to a local `Int` before `ptr_add` exits
  `42`; direct `offset()` outside pointer arithmetic exits `42`; the same
  direct-operand failure reproduces with generic `id(4)`. Runtime builtin scalar
  calls are not currently affected by this bug: direct `core.task_group_status`
  offsets for open, canceled, and closed task groups exit `0` in
  `memory_group_status_direct_offset_service.tetra`.
- Workaround used in microservice examples: alias function-call or generic-call
  offset results into local `Int` bindings before calling `core.ptr_add`.

### TETRA-BUG-0029: Expression offset operands break raw pointer arithmetic provenance

- Status: fixed, verified.
- Area: runtime memory / pointer arithmetic with computed scalar operands.
- Found while creating:
  `examples/microservices/memory_expression_offset_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/expression_offset.tetra" <<'EOF'
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem = core.cap_mem()
        let payload: ptr = core.alloc_bytes(8)
        let cell: ptr = core.ptr_add(payload, 2 + 2, mem)
        let _write: Int = core.store_i32(cell, 42, mem)
        return core.load_i32(cell, mem)
    return 99
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/expression_offset.tetra"
"$tmp/app"
```

- Expected: using an arithmetic expression that evaluates to `4` as the offset
  operand in `core.ptr_add` behaves like a local `Int` with value `4`, so the
  program exits `42`.
- Actual: the program builds but exits `2`, matching the runtime memory guard's
  invalid access path.
- Control case: assigning `2 + 2` to a local `Int` before `ptr_add` exits `42`.
- Additional failing controls: direct `2 * 2` and parenthesized `(2 + 2)` offset
  operands also exit `2`.
- Additional failing evidence: imported typed-error payload memory code that
  recovers four local offsets from typed task `catch` expressions still exits
  `2` when the later `core.ptr_add` calls use arithmetic offset expressions
  such as `second + 4`, `third + 8`, and `fourth + 12`. The same imported
  payload workers pass after each recovered offset is used as a plain local
  `Int` against a separate allocation base in
  `compiler_typed_error_payload_memory_pack/app/main.tetra`.
- Workaround used in microservice examples: compute arithmetic offsets in local
  `Int` bindings before calling `core.ptr_add`.

### TETRA-BUG-0030: Runtime result and message fields break raw pointer arithmetic provenance

- Status: fixed, verified.
- Area: runtime memory / actor-task result field operands.
- Found while creating:
  `examples/microservices/memory_task_result_offset_service.tetra` and
  `examples/microservices/memory_actor_message_offset_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/task_result_offset.tetra" <<'EOF'
func offset_worker() -> Int:
    return 4

func main() -> Int
uses alloc, capability, mem, runtime:
    let task: task.i32 = core.task_spawn_i32("offset_worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    unsafe:
        let mem = core.cap_mem()
        let payload: ptr = core.alloc_bytes(8)
        let cell: ptr = core.ptr_add(payload, result.value, mem)
        let _write: Int = core.store_i32(cell, 42, mem)
        return core.load_i32(cell, mem)
    return 99
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/task_result_offset.tetra"
"$tmp/app"
```

- Expected: using `result.value` with value `4` in `core.ptr_add` behaves like
  an equivalent local `Int` offset, so the program exits `42`.
- Actual: the program builds but exits `2`, matching the runtime memory guard's
  invalid access path.
- Additional failing controls: grouped `task.result_i32.value`, `actor.msg.value`,
  and `actor.recv_msg_result.value` direct offset operands also exit `2`.
- Additional failing evidence: deadline/select task-result paths have the same
  memory guard boundary. Using `result.value` directly as the `core.ptr_add`
  offset after `core.task_join_until_i32(task, core.deadline_ms(5))` or
  `core.select2_i32(task, core.deadline_ms(5))` builds but exits `2`.
- Additional failing evidence: plain actor receive results and tagged receive
  tags hit the same memory guard boundary. Using `reply.value` directly as the
  `core.ptr_add` offset after `core.recv_until(core.deadline_ms(5))` or a
  completed `core.recv_poll()` builds but exits `2`; using `reply.tag` directly
  after `core.recv_msg_until(core.deadline_ms(5))` also builds but exits `2`.
- Control cases: assigning each runtime field to a local `Int` before `ptr_add`
  exits `42`; scalar typed actor enum payload bindings are already local case
  bindings and exit `0` in `memory_actor_typed_payload_offset_service.tetra`;
  local aliases of `task_join_until_i32`, completed `task_poll_i32`, and
  `select2_i32` result values exit `0` in
  `memory_join_until_result_offset_service.tetra`,
  `memory_poll_result_offset_service.tetra`,
  `memory_select_result_offset_service.tetra`, and
  `compiler_task_wait_memory_pack/app/main.tetra`. Direct timeout/pending
  `.error` fields remain a passing byte-offset control in
  `memory_join_until_error_offset_service.tetra`,
  `memory_poll_error_offset_service.tetra`,
  `memory_select_error_offset_service.tetra`, and
  `compiler_task_wait_error_memory_pack/app/main.tetra`.
  Local aliases of deadline-aware actor receive values, completed actor poll
  values, and tagged actor receive tags exit `0` in
  `memory_actor_recv_value_offset_service.tetra`,
  `memory_actor_poll_value_offset_service.tetra`,
  `memory_actor_tag_offset_service.tetra`, and
  `compiler_actor_wait_memory_pack/app/main.tetra`.
  Direct actor receive timeout, empty poll, and tagged receive timeout `.error`
  fields remain passing byte-offset controls in
  `memory_actor_recv_error_offset_service.tetra`,
  `memory_actor_poll_error_offset_service.tetra`,
  `memory_actor_recv_msg_error_offset_service.tetra`, and
  `compiler_actor_error_memory_pack/app/main.tetra`.
- Workaround used in microservice examples: copy task result and actor message
  fields into local `Int` bindings before using them as `core.ptr_add` offsets.

### TETRA-BUG-0031: Enum payloads reject generic struct instantiations

- Status: fixed, verified.
- Area: compiler / enum payload type resolution / generic structs.
- Found while creating:
  `examples/microservices/compiler_generic_struct_field_pack/app/main.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/generic_struct_enum_payload.tetra" <<'EOF'
struct Box<T>:
    value: T

enum Route:
    case ready(Box<Int>)
    case empty

func main() -> Int:
    let box: Box<Int> = Box<Int>(value: 42)
    let route: Route = Route.ready(box)
    match route:
    case Route.ready(payload):
        return payload.value
    case Route.empty:
        return 0
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/generic_struct_enum_payload.tetra"
```

- Expected: `Box<Int>` is a valid monomorphized generic struct type and can be
  used as an enum payload type, matching generic struct locals and non-generic
  struct enum payloads.
- Actual: build fails with `enum 'Route' case 'ready': unknown type 'Box'`.
- Additional failing evidence: a typed actor message enum with cases such as
  `case request(Box<Int>)` and `case reply(Box<Int>)` hits the same generic
  enum-payload type-resolution boundary in the memory+actor probe variant.
- Additional failing evidence: typed error enums hit the same boundary. A
  throwing worker with `enum OffsetErr: case boxed(Box<Int>)` fails before
  runtime with `enum 'OffsetErr' case 'boxed': unknown type 'Box'`.
- Control cases: a non-generic `Box` enum payload builds and exits `42`; a
  local `Box<Int>` value outside an enum payload builds and exits `42`; a
  `Holder` struct field of type `Box<Int>` builds and exits `42`; typed actor
  nested enum payloads with non-generic scalar or struct payloads exit `0` in
  `memory_actor_typed_enum_payload_offset_service.tetra`,
  `memory_actor_typed_enum_struct_payload_offset_service.tetra`, and
  `compiler_typed_actor_enum_payload_memory_pack/app/main.tetra`; typed task
  error non-generic struct, nested enum, optional, and guarded payloads exit
  `0` in the typed-error payload memory microservices and
  `compiler_typed_error_payload_memory_pack/app/main.tetra`.
- Workaround used in microservice examples: wrap generic struct payloads in
  non-enum struct fields, or use non-generic enum payload types.

### TETRA-BUG-0032: Indexed and metadata offsets break raw pointer arithmetic provenance

- Status: fixed, verified.
- Area: runtime memory / pointer arithmetic with indexed and metadata scalar
  operands.
- Found while creating:
  `examples/microservices/memory_indexed_metadata_offset_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/slice_offset.tetra" <<'EOF'
func main() -> Int
uses alloc, capability, islands, mem:
    unsafe:
        let mem = core.cap_mem()
        let payload: ptr = core.alloc_bytes(8)
        island(128) as isl:
            var offsets: []i32 = core.island_make_i32(isl, 2)
            offsets[0] = 1
            offsets[1] = 4
            let cell: ptr = core.ptr_add(payload, offsets[1], mem)
            let _write: Int = core.store_i32(cell, 42, mem)
            return core.load_i32(cell, mem)
    return 99
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/slice_offset.tetra"
"$tmp/app"
```

- Expected: using a slice element with value `4` as the offset operand in
  `core.ptr_add` behaves like a local `Int` with value `4`, so the program
  exits `42`.
- Actual: the program builds but exits `2`, matching the runtime memory guard's
  invalid access path.
- Additional failing control: direct `label.len` string metadata offset operands
  also build and exit `2`.
- Control cases: assigning `offsets[1]` or `label.len` to a local `Int` before
  `ptr_add` exits `42`.
- Workaround used in microservice examples: copy indexed and metadata scalar
  reads into local `Int` bindings before calling `core.ptr_add`.

### TETRA-BUG-0033: Payload-typed task handles reject explicit task.i32 annotations

- Status: fixed, verified.
- Area: compiler / typed task handle type checking / payload error enums.
- Found while creating:
  `examples/microservices/parallel_typed_task_payload_handle_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/typed_payload_annotated.tetra" <<'EOF'
enum TaskErr:
    case boom(Int)
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(42)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        7
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/typed_payload_annotated.tetra"
```

- Expected: the typed task spawn result is usable with an explicit `task.i32`
  local annotation, matching no-payload typed task examples and the public task
  handle surface.
- Actual: build fails with
  `type mismatch: expected 'task.i32', got 'task.i32.throws.TaskErr'`.
- Additional failing control: grouped payload typed tasks fail the same way with
  `task_spawn_group_i32_typed`.
- Additional failing evidence: the precise inferred typed-handle name is not
  available as source syntax for APIs or containers. Struct fields, optional
  locals, and function parameters written as `task.i32.throws.TaskErr` all fail
  in the parser with `expected identifier, got throws`.
- Control cases: omitting the local type annotation for payload typed task
  handles builds and exits `42`; explicitly annotating a no-payload typed task
  handle as `task.i32` builds and exits `42`; keeping payload typed task
  handles on local inferred bindings exits `0` in
  `parallel_typed_task_match_catch_service.tetra`,
  `memory_typed_task_error_offset_service.tetra`, and
  `compiler_typed_task_match_pack/app/main.tetra`.
- Workaround used in microservice examples: let payload typed task handles infer
  their precise typed-handle type instead of annotating them as `task.i32` or
  moving them through typed APIs/containers.

### TETRA-BUG-0034: Direct pointer base expressions break raw pointer arithmetic provenance

- Status: fixed, verified.
- Area: runtime memory / pointer arithmetic with non-local base operands.
- Found while creating:
  `examples/microservices/memory_direct_base_offset_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/direct_alloc_base_offset.tetra" <<'EOF'
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem = core.cap_mem()
        let cell: ptr = core.ptr_add(core.alloc_bytes(8), 4, mem)
        let _write: Int = core.store_i32(cell, 42, mem)
        return core.load_i32(cell, mem)
    return 99
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/direct_alloc_base_offset.tetra"
"$tmp/app"
```

- Expected: using a direct allocation expression as the base pointer operand in
  `core.ptr_add` behaves like first storing that allocation in a local `ptr`, so
  the program exits `42`.
- Actual: the program builds but exits `2`, matching the runtime memory guard's
  invalid access path.
- Additional failing controls: direct function-returned base pointers
  (`core.ptr_add(make(), 4, mem)`) and direct loaded base pointers
  (`core.ptr_add(core.load_ptr(table, mem), 4, mem)`) also build and exit `2`.
- Control cases: assigning each base pointer expression to a local `ptr` before
  `ptr_add` exits `42`.
- Workaround used in microservice examples: materialize allocation,
  function-returned, and loaded pointer bases into local `ptr` bindings before
  raw pointer arithmetic.

### TETRA-BUG-0035: Typed actor receives silently reinterpret mismatched enum message types

- Status: fixed, verified.
- Area: actor runtime / typed actor mailbox / enum message type safety.
- Found while creating:
  `examples/microservices/actor_typed_envelope_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/typed_actor_mismatch.tetra" <<'EOF'
enum Request:
    case value(Int)
    case stop

enum Reply:
    case failed(Int)
    case value(Int)

func worker() -> Int
uses actors:
    let msg: Reply = core.recv_typed<Reply>()
    match msg:
    case Reply.failed(code):
        let _failed: Int = core.send_typed(core.sender(), Reply.failed(code))
        return 0
    case Reply.value(value):
        let _reply: Int = core.send_typed(core.sender(), Reply.value(value + 1))
        return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let _sent: Int = core.send_typed(peer, Request.value(41))
    let reply: Reply = core.recv_typed<Reply>()
    match reply:
    case Reply.failed(code):
        return code
    case Reply.value(value):
        if value == 42:
            return 0
        return value
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/typed_actor_mismatch.tetra"
"$tmp/app"
```

- Expected: typed actor mailboxes reject or otherwise protect a receive of
  `Reply` when the queued message was sent as `Request`, so enum tags and
  payload slots from unrelated types cannot be reinterpreted silently.
- Actual: the program builds and exits `41`; `Request.value(41)` is read as
  `Reply.failed(41)` because only the raw enum tag/payload slots cross the
  mailbox.
- Additional failing control: when `Request.value` and `Reply.value` have the
  same case order, the mismatch builds and exits `0`, hiding the type mismatch
  by treating `Request.value(41)` as `Reply.value(41)`.
- Control case: sending and receiving one shared `Message` enum builds and exits
  `0`.
- Workaround used in microservice examples: use one shared typed envelope enum
  for every actor participating in a typed mailbox protocol.

### TETRA-BUG-0036: Typed error payloads of derived pointers lose memory provenance

- Status: fixed, verified.
- Area: typed errors / catch lowering / raw pointer provenance.
- Found while creating:
  `examples/microservices/memory_typed_error_ptr_base_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/typed_error_derived_ptr.tetra" <<'EOF'
enum PtrErr:
    case raw(ptr)

func fail(raw: ptr) -> ptr throws PtrErr:
    throw PtrErr.raw(raw)

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let payload: ptr = core.alloc_bytes(8)
        let cell: ptr = core.ptr_add(payload, 4, mem)
        let _write: Int = core.store_i32(cell, 42, mem)
        let loaded_cell: ptr = catch fail(cell):
        case PtrErr.raw(raw):
            raw
        return core.load_i32(loaded_cell, mem)
    return 99
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/typed_error_derived_ptr.tetra"
"$tmp/app"
```

- Expected: a synchronous `throw`/`catch` inside one thread preserves enough
  derived pointer provenance for the later `core.load_i32` to exit `42`.
- Actual: the program builds but exits `2`, matching the runtime memory guard's
  invalid access path.
- Additional failing evidence: wrapping the same derived pointer in an optional
  typed-error payload (`case maybe(ptr?)`) also builds but exits `2` after
  unwrapping and loading from the recovered derived pointer.
- Control case: throwing and catching the allocation-base pointer, then
  deriving `ptr_add` after the catch, exits `42`; optional allocation-base
  pointer payloads exit `0` in
  `memory_typed_error_optional_ptr_base_service.tetra`,
  `memory_typed_error_optional_ptr_dynamic_service.tetra`, and
  `compiler_typed_error_optional_ptr_pack/app/main.tetra`.
- Workaround used in microservice examples: carry allocation-base pointers
  through direct or optional typed-error payloads and derive raw-memory offsets
  only after catch.

### TETRA-BUG-0037: Global fixed-array element writes do not round-trip at runtime

- Status: fixed, verified.
- Area: fixed arrays / global lowering / runtime value storage.
- Found while probing compiler fixed-array service candidates.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/global_fixed_array_write.tetra" <<'EOF'
var seed: [3]Int

func main() -> Int:
    seed[0] = 42
    return seed[0]
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/global_fixed_array_write.tetra"
"$tmp/app"
```

- Expected: `seed[0]` reads back the value written to the global fixed-array
  element and exits `42`.
- Actual: the program builds but exits `1`. The same symptom appears for
  `box.items[0] = 42; return box.items[0]` on a global struct with a fixed-array
  field and for copying a mutated global fixed-array into a struct.
- Additional failing evidence: reading a zero-initialized global fixed-array
  element (`var seed: [3]Int; return seed[0]`), a zero-initialized global
  struct fixed-array field, or an optional fixed-array assigned from a global
  fixed array all build but exit `1` instead of `0`.
- Control case: a scalar global write/read (`var saved: Int; saved = 42; return
  saved`) builds and exits `42`; an optional fixed-array global initialized to
  `none` branches through the `else` path and exits `0`.
- Workaround: avoid global fixed-array mutation in runtime-sensitive services;
  use slices or scalar globals until global fixed-array lowering is fixed.

### TETRA-BUG-0038: Scalar inout writes do not propagate back to caller locals

- Status: fixed, verified.
- Area: ownership/inout lowering / caller local writeback.
- Found while probing compiler ownership microservice candidates.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/inout_writeback.tetra" <<'EOF'
func bump(value: inout Int) -> Int:
    value = value + 1
    return value

func main() -> Int:
    var score: Int = 41
    let result: Int = bump(score)
    if result == 42 && score == 42:
        return 0
    return result + score
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/inout_writeback.tetra"
"$tmp/app"
```

- Expected: `inout Int` gives exclusive mutable access, so `score` is updated
  and the program exits `0`.
- Actual: the program builds but exits `83`, showing `result == 42` while the
  caller local remains `41`.
- Additional failing evidence: routing the same `bump` function through a
  `fn(inout Int) -> Int` callback also builds but exits `83`, so callback
  dispatch does not compensate for the missing caller-local writeback.
- Additional failing evidence: non-generic and generic `inout ptr?` replacement
  helpers also fail to write back to the caller local. A program that sets
  `var maybe_base: ptr? = none`, calls `replace(maybe_base, next_base)`, then
  checks `maybe_base` builds but exits `95`, showing the local stayed `none`.
- Control case: using only the returned value (`return bump(score)`) builds and
  exits `42`. For optional pointers, consuming the explicit returned `ptr?`
  from `replace(slot, next_base)` and deriving the raw-memory cell from that
  returned value exits `0`.
- Workaround used in microservice examples: consume the explicit return value
  from scalar or optional-pointer `inout` helpers instead of relying on
  caller-local writeback.

### TETRA-BUG-0039: Dynamic ptr_add offsets from derived pointer locals lose memory provenance

- Status: fixed, verified.
- Area: runtime memory / derived pointer provenance with looped dynamic
  offsets.
- Found while creating:
  `examples/microservices/memory_dynamic_base_offset_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/derived_dynamic_offset.tetra" <<'EOF'
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let base: ptr = core.alloc_bytes(8)
        let two: ptr = core.ptr_add(base, 2, mem)
        let three: ptr = core.ptr_add(base, 3, mem)
        let _w2: UInt8 = core.store_u8(two, 20, mem)
        let _w3: UInt8 = core.store_u8(three, 22, mem)
        var i: Int = 0
        var total: Int = 0
        while i < 2:
            let p: ptr = core.ptr_add(two, i, mem)
            total = total + core.load_u8(p, mem)
            i = i + 1
        return total
    return 99
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/derived_dynamic_offset.tetra"
"$tmp/app"
```

- Expected: dynamic offsets from a derived pointer local preserve the same
  allocation provenance as constant offsets, so the loop reads `20 + 22` and
  exits `42`.
- Actual: the program builds but exits `2`, matching the runtime memory guard's
  invalid access path.
- Control cases: looping over the allocation base with a mutable local offset
  (`core.ptr_add(base, offset, mem)`) exits `42`; using a constant offset from
  the derived pointer local (`core.ptr_add(two, 1, mem)`) also exits `42`.
- Additional failing evidence: using an immutable local offset
  (`let delta: Int = 1`) from a derived pointer local also builds but exits `2`;
  the same helper shape anchored at an allocation-base pointer parameter exits
  `42`.
- Workaround used in microservice examples: keep dynamic offset loops anchored
  at the allocation base and carry the absolute offset in an `Int` local.

### TETRA-BUG-0040: Explicit selfhost task-group builds fail with raw missing ABI symbol

- Status: fixed, verified.
- Area: compiler / runtime selection diagnostics / task groups.
- Found while probing explicit runtime modes for task-group microservices.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/selfhost_task_group.tetra" <<'EOF'
func worker() -> Int:
    return 42

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let value: Int = core.task_join_i32(task)
    let close_error: Int = core.task_group_close(group)
    if close_error == 0:
        return value
    return close_error
EOF
go run ./cli/cmd/tetra build --runtime=selfhost --target linux-x64 -o "$tmp/app" "$tmp/selfhost_task_group.tetra"
```

- Expected: because `--runtime=auto` selects builtin for task-group programs and
  explicit `--runtime=selfhost` already has targeted diagnostics for unsupported
  typed tasks and multi-spawn actor shapes, explicit selfhost task-group builds
  should fail early with a stable diagnostic such as "self-host runtime does not
  support task groups; use runtime=auto or runtime=builtin".
- Actual: build fails with the raw ABI/link validation message
  `runtime object missing required symbol '__tetra_task_group_open'`.
- Control cases: the same source builds and exits `42` with `--runtime=auto` or
  `--runtime=builtin`; a selfhost-supported untyped deadline task service builds
  and exits `0` under explicit `--runtime=selfhost`.
- Workaround used in microservice examples: keep task-group services on
  `runtime=auto`/builtin and reserve explicit selfhost smoke coverage for the
  untyped task/deadline surface.

### TETRA-BUG-0041: Scoped if-let and catch payload bindings remain reserved after scope exit

- Status: fixed, verified.
- Area: compiler / pattern binding scopes / local-name tracking.
- Found while creating:
  `examples/microservices/compiler_pattern_binding_unique_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/iflet_binding_reuse.tetra" <<'EOF'
func maybe(flag: Bool, value: Int) -> Int?:
    if flag:
        return value
    return none

func main() -> Int:
    let a: Int? = maybe(true, 20)
    if let value = a:
        let left: Int = value
    let b: Int? = maybe(true, 22)
    if let value = b:
        return value + 20
    return 99
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/iflet_binding_reuse.tetra"
```

- Expected: the first `if let value = ...` binding is scoped to its branch and
  cannot be used after the branch, so a later independent `if let value = ...`
  should be accepted and the program should exit `42`.
- Actual: build fails with `duplicate local 'value'`.
- Additional failing evidence: two separate typed `catch` expressions that both
  bind `code` and two separate `match` expressions that both bind `value` fail
  the same way.
- Control case: using distinct payload binding names in the two `if let` or
  `catch` expressions builds and exits `42`.
- Workaround used in microservice examples: keep payload binding names unique
  across the whole function, even when the pattern branch/expression scope has
  ended.

### TETRA-BUG-0042: Stdlib byte helpers fail on valid derived memory windows

- Status: fixed, verified.
- Area: stdlib memory helpers / runtime memory / derived pointer provenance.
- Found while creating:
  `examples/microservices/memory_base_dynamic_copy_service.tetra`.
- Reproduction command, run from the project root:

```sh
mkdir -p probes
cat > probes/tetra_bug_0042_memcpy.tetra <<'EOF'
module probes.tetra_bug_0042_memcpy

import lib.core.capability as cap
import lib.core.memory as memory

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let memory_cap: cap.mem = core.cap_mem()
        let source: ptr = core.alloc_bytes(8)
        let target: ptr = core.alloc_bytes(8)
        let source_two: ptr = core.ptr_add(source, 2, memory_cap)
        let source_three: ptr = core.ptr_add(source, 3, memory_cap)
        let source_four: ptr = core.ptr_add(source, 4, memory_cap)
        let source_five: ptr = core.ptr_add(source, 5, memory_cap)
        let target_one: ptr = core.ptr_add(target, 1, memory_cap)
        let target_two: ptr = core.ptr_add(target, 2, memory_cap)
        let target_three: ptr = core.ptr_add(target, 3, memory_cap)
        let target_four: ptr = core.ptr_add(target, 4, memory_cap)
        let _seed_two: UInt8 = core.store_u8(source_two, 9, memory_cap)
        let _seed_three: UInt8 = core.store_u8(source_three, 10, memory_cap)
        let _seed_four: UInt8 = core.store_u8(source_four, 11, memory_cap)
        let _seed_five: UInt8 = core.store_u8(source_five, 12, memory_cap)
        let _clear: Int = memory.memset_u8(target, 0, 8, memory_cap)
        let _copy: Int = memory.memcpy_u8(target_one, source_two, 4, memory_cap)
        let total: Int = core.load_u8(target_one, memory_cap) + core.load_u8(target_two, memory_cap) + core.load_u8(target_three, memory_cap) + core.load_u8(target_four, memory_cap)
        if total == 42:
            return 0
        return total
    return 99
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-bug-0042 probes/tetra_bug_0042_memcpy.tetra
/tmp/tetra-bug-0042
rm -f probes/tetra_bug_0042_memcpy.tetra /tmp/tetra-bug-0042
rmdir probes 2>/dev/null || true
```

- Expected: `source_two..source_five` and `target_one..target_four` are valid
  windows inside their allocations, so `memory.memcpy_u8(target_one,
  source_two, 4, mem)` should copy `9 + 10 + 11 + 12` and the program should
  exit `0`.
- Actual: the program builds but exits `2`, matching the runtime memory guard's
  invalid access path before the valid derived window can be copied.
- Additional failing evidence: `memory.memset_u8(target_one, 21, 2, mem)` on a
  valid derived target window also builds but exits `2` instead of writing two
  bytes and exiting `0`.
- Control cases: allocation-base `memory.memcpy_u8(target, source, n, mem)`
  builds and exits `0` in `memory_copy_window_service.tetra`; manually copying
  derived windows with unrolled direct loads/stores exits `0` in
  `memory_derived_copy_service.tetra`; dynamically copying the same subwindow
  while anchoring every `core.ptr_add` at the allocation bases exits `0` in
  `memory_base_dynamic_copy_service.tetra`; zero-length `memory.memcpy_u8` and
  `memory.memset_u8` calls over derived windows exit `0` in
  `memory_zero_length_derived_helper_service.tetra`.
- Workaround used in microservice examples: do not pass derived source or
  destination pointers to stdlib byte helpers; keep helper calls on allocation
  bases or perform the subwindow loop with base-anchored dynamic offsets.

### TETRA-BUG-0043: Formatter drops public visibility modifiers from declarations

- Status: fixed, verified.
- Area: formatter / module public API surface.
- Found while creating:
  `examples/microservices/compiler_interface_control_pack`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/pub_format_probe.tetra" <<'EOF'
module probes.pub_format_probe

pub enum Route:
    case ready(Int)

pub func score(route: Route) -> Int:
    return 42
EOF
go run ./cli/cmd/tetra fmt "$tmp/pub_format_probe.tetra"
```

- Expected: `tetra fmt` preserves the public visibility modifiers because
  `pub enum` and `pub func` are part of the source module's exported API
  surface.
- Actual: `tetra fmt` exits `0` but prints:

```tetra
module probes.pub_format_probe

enum Route:
    case ready(Int)

func score(route: Route) -> Int:
    return 42
```

- Control case: the parser accepts public declarations and downstream interface
  tests assert public API text such as `pub struct Point:` and
  `pub func add(a: i32, b: i32) -> i32:`, so this is formatter loss, not a
  syntax limitation.
- Additional failing evidence: a formatter probe containing `pub import`,
  `pub struct`, `pub protocol`, `pub extension`, `pub const`, and `pub func`
  prints every declaration without its public modifier.
- Workaround used in microservice examples: keep formatter-gated smoke packs on
  non-public declarations until formatter public-modifier preservation is fixed;
  avoid running `tetra fmt -write` on source files where `pub` is required for
  interface/API artifacts.

### TETRA-BUG-0044: Formatter corrupts selective import declarations

- Status: fixed, verified.
- Area: formatter / module imports.
- Found while probing compiler import microservices after `TETRA-BUG-0043`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
mkdir -p "$tmp/app" "$tmp/lib"
cat > "$tmp/lib/math.tetra" <<'EOF'
module lib.math

func add(a: Int, b: Int) -> Int:
    return a + b
EOF
cat > "$tmp/app/main.tetra" <<'EOF'
module app.main

import lib.math.{add}

func main() -> Int:
    return add(40, 2)
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/original" "$tmp/app/main.tetra"
go run ./cli/cmd/tetra fmt "$tmp/app/main.tetra" > "$tmp/app/formatted_main.tetra"
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/formatted" "$tmp/app/formatted_main.tetra"
```

- Expected: `tetra fmt` preserves the selective import or rewrites it to a
  valid equivalent import while keeping `add` resolvable.
- Actual: the original selective-import source builds, but `tetra fmt` prints:

```tetra
module app.main

import lib.math as

func main() -> Int:
    return add(40, 2)
```

  Building the formatted source fails with `expected identifier, got fun`.
- Control cases: ordinary aliased module imports build and exit `0` in
  `compiler_import_alias_pack/app/main.tetra`; parser tests already cover
  selective imports such as `import engine.math.{add, Vec}`.
- Workaround used in microservice examples: use `import module.path as alias`
  plus qualified calls for formatter-gated sources; avoid running `tetra fmt`
  on selective-import files until import item preservation is fixed.

### TETRA-BUG-0045: Spawning through an optional task-group payload returns the wrong worker value

- Status: fixed, verified.
- Area: runtime / task groups / optional resource payloads.
- Found while probing optional task-group microservices after Graphify pointed
  at task-group optional-payload provenance tests.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/group_optional_spawn.tetra" <<'EOF'
func worker() -> Int:
    return 42

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let maybe: task.group? = group
    if let handle = maybe:
        let task: task.i32 = core.task_spawn_group_i32(handle, "worker")
        let result: task.result_i32 = core.task_join_result_i32(task)
        let close_error: Int = core.task_group_close(handle)
        if result.error != 0:
            return 10 + result.error
        if close_error != 0:
            return 20 + close_error
        return result.value
    return 99
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/group_optional_spawn.tetra"
"$tmp/app"
```

- Expected: the unwrapped `task.group` payload should behave like the original
  task group handle. Spawning `worker`, joining it, and closing the group should
  return the worker value `42`.
- Actual: the program builds, `task_join_result_i32(task).error` is `0`, and
  `task_group_close(handle)` returns `0`, but the joined `result.value` is `1`
  instead of `42`.
- Additional failing evidence: `core.task_spawn_group_i32_typed<GroupErr>` on
  the same unwrapped `task.group?` payload builds, but the typed join catch path
  returns `0` instead of the direct-control worker value `42`.
- Additional failing evidence: wrapping the group in an optional aggregate also
  loses the worker value. Both `GroupBox?` with a `task.group` field and
  `Box<task.group>?` with a generic `task.group` field build, report
  `task_join_result_i32(task).error == 0`, close with `0`, and return joined
  `result.value == 1` instead of `42`.
- Additional failing evidence: unwrapping `task.group?`, passing the handle
  through a non-generic `alias_group(group: task.group) -> task.group` helper,
  and then calling `core.task_spawn_group_i32` on the returned handle also
  builds but exits `1` instead of the worker value `42`.
- Additional failing evidence: wrapping a `task.group` inside
  `GroupSlot.ready(group)`, then wrapping that enum in `GroupSlot?`, unwrapping
  with `if let`, matching `GroupSlot.ready(handle)`, and calling
  `core.task_spawn_group_i32(handle, "worker")` builds but the joined worker
  value is still `1` instead of `42`.
- Additional failing evidence: the typed grouped-spawn variant on the same
  optional enum wrapper also loses the success value. A direct typed enum route
  that returns `core.task_join_group_i32_typed<GroupErr>(task)` exits `42`,
  while the optional-enum wrapped route exits `0`, showing the catch success
  payload is dropped before the worker value reaches the caller.
- Control cases: direct `core.task_spawn_group_i32(group, "worker")` with the
  original group handle exits `42`; direct typed grouped spawn exits `42`;
  optional group status/close exits `0` in
  `parallel_group_optional_close_service.tetra`; optional group cancel/status
  exits `0` in `parallel_group_optional_cancel_service.tetra`; optional group
  match/status/close exits `0` in
  `parallel_group_optional_match_close_service.tetra`; struct and enum
  task-group payload spawns exit `0` in
  `parallel_group_struct_spawn_service.tetra` and
  `parallel_group_enum_spawn_service.tetra`; typed struct and enum task-group
  payload spawns exit `0` in
  `parallel_group_typed_struct_spawn_service.tetra` and
  `parallel_group_typed_enum_spawn_service.tetra`; non-optional generic
  task-group boxes exit `0` in `parallel_group_generic_box_spawn_service.tetra`;
  non-generic task-group aliases exit `0` in
  `parallel_group_alias_spawn_service.tetra`; optional alias status/close exits
  `0` in `parallel_group_optional_alias_close_service.tetra`; optional actor
  alias send and optional task alias join exit `0` in
  `parallel_actor_optional_alias_send_service.tetra` and
  `parallel_task_optional_alias_join_service.tetra`; optional enum status/close
  exits `0` in `parallel_group_optional_enum_close_service.tetra`; optional
  enum actor send and optional enum task join exit `0` in
  `parallel_actor_optional_enum_send_service.tetra` and
  `parallel_task_optional_enum_join_service.tetra`; typed optional task-group
  alias-spawn exits `0` in
  `parallel_typed_group_optional_alias_spawn_service.tetra`.
- Workaround used in microservice examples: do not spawn grouped tasks through
  optional payloads that contain `task.group`; keep
  `core.task_spawn_group_i32` on a direct, struct-field, enum-payload, generic
  box, non-generic alias, or optional status/close-only `task.group` handle
  until optional task-group runtime payloads preserve the worker result
  correctly.

### TETRA-BUG-0046: Generic identity over actor/task/island resources loses usable provenance

- Status: fixed, verified.
- Area: compiler / generics / resource provenance / actor, task, and island resources.
- Found while probing generic actor/task/island resource microservices.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/group_generic_identity.tetra" <<'EOF'
func worker() -> Int:
    return 42

func id<T>(value: T) -> T:
    return value

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let returned: task.group = id(group)
    let task: task.i32 = core.task_spawn_group_i32(returned, "worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    let close_error: Int = core.task_group_close(returned)
    if result.error != 0:
        return 10 + result.error
    if close_error != 0:
        return 20 + close_error
    return result.value
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/group_generic_identity.tetra"
```

- Expected: generic identity over `task.group` should preserve the same usable
  resource provenance as the non-generic identity helper. The program should
  build, spawn `worker`, join it, close the group, and exit `42`.
- Actual: build fails at the grouped spawn with
  `ambiguous resource provenance for 'returned' after control-flow merge`.
- Additional failing evidence: the imported generic helper route
  `groups.id(boxed.value)` in a cross-module generic task-group pack fails with
  `cannot infer generic argument for 'groups.id' arg 1`.
- Additional failing evidence: the same same-module failure appears for plain
  task handles. Passing a `task.i32` through `id<T>(value: T) -> T`, then
  joining the returned handle, fails with
  `ambiguous resource provenance for 'returned' after control-flow merge`.
- Additional failing evidence: the imported task-handle route
  `tasks.id(box.value)` in a cross-module generic task pack fails with
  `cannot infer generic argument for 'tasks.id' arg 1`.
- Additional failing evidence: the same generic inference boundary appears for
  typed task handles. Passing the result of
  `core.task_spawn_i32_typed<TaskErr>("worker")` through `id<T>(value: T) -> T`
  fails at the generic call with
  `cannot infer generic argument for 'id' arg 1` before the typed join can run.
- Additional failing evidence: the same same-module failure appears for actor
  handles. Passing an `actor` through `id<T>(value: T) -> T`, then sending
  through the returned handle, fails with
  `ambiguous resource provenance for 'returned' after control-flow merge`.
- Additional failing evidence: the imported actor route
  `actors.id(box.value)` in a cross-module generic actor pack fails with
  `cannot infer generic argument for 'actors.id' arg 1`.
- Additional failing evidence: the same same-module failure appears for island
  handles. Passing an `island` through `id<T>(value: T) -> T`, then allocating a
  region-backed slice from the returned handle, fails with
  `ambiguous resource provenance for 'returned' after control-flow merge`.
- Control cases: the non-generic alias helper exits `0` in
  `parallel_group_alias_spawn_service.tetra`; monomorphized generic
  `Box<task.group>` pass-through exits `0` in
  `parallel_group_generic_box_spawn_service.tetra`; imported generic
  `Box<task.group>` plus a non-generic alias builds and exits `0` in
  `compiler_group_generic_pack/app/main.tetra`; the task-handle equivalents
  exit `0` in `parallel_task_alias_join_service.tetra`,
  `parallel_task_generic_box_join_service.tetra`, and
  `compiler_task_generic_pack/app/main.tetra`. Optional struct/generic box
  task-handle joins also exit `0` in
  `parallel_task_optional_struct_box_join_service.tetra` and
  `parallel_task_optional_generic_box_join_service.tetra`. The actor-handle
  equivalents exit `0` in `parallel_actor_alias_send_service.tetra`,
  `parallel_actor_generic_box_send_service.tetra`, and
  `compiler_actor_generic_pack/app/main.tetra`; optional struct/generic actor
  boxes exit `0` in `parallel_actor_optional_struct_box_send_service.tetra` and
  `parallel_actor_optional_generic_box_send_service.tetra`; typed optional actor
  alias send exits `0` in
  `parallel_actor_typed_optional_alias_send_service.tetra`; typed optional
  grouped-task alias spawn exits `0` in
  `parallel_typed_group_optional_alias_spawn_service.tetra`; the imported typed
  optional alias pack exits `0` and builds interface-only in
  `compiler_typed_optional_alias_resource_pack/app/main.tetra`. The island-handle
  equivalents exit `0` in `memory_island_alias_region_service.tetra`,
  `memory_island_generic_box_region_service.tetra`, and
  `compiler_island_generic_pack/app/main.tetra`; optional struct/generic island
  boxes exit `0` in `memory_island_optional_struct_box_service.tetra` and
  `memory_island_optional_generic_box_service.tetra`.
- Workaround used in microservice examples: avoid generic identity functions for
  actor/task/island resources; use a non-generic resource-specific helper or a
  monomorphized generic struct pass-through instead.

### TETRA-BUG-0047: Island parameters cannot be returned inside aggregate constructors

- Status: fixed, verified.
- Area: compiler / resource provenance / island aggregate constructors.
- Found while probing generic island resource microservices.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/island_wrap_struct.tetra" <<'EOF'
struct IslandBox:
    region: island

func wrap_region(region: island) -> IslandBox:
    return IslandBox(region: region)

func main() -> Int
uses alloc, islands, mem:
    unsafe:
        let region: island = core.island_new(128)
        let boxed: IslandBox = wrap_region(region)
        var bytes: []u8 = core.island_make_u8(boxed.region, 3)
        bytes[0] = 12
        bytes[1] = 13
        bytes[2] = 17
        let total: Int = bytes[0] + bytes[1] + bytes[2]
        free(boxed.region)
        return total
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/island_wrap_struct.tetra"
```

- Expected: returning an island parameter inside a struct constructor should
  preserve the same usable provenance as actor/task aggregate wrappers and as
  direct call-site island aggregate construction. The program should build and
  run with exit `42`.
- Actual: build fails at `IslandBox(region: region)` with
  `cannot use consumed value 'region' (consumed at ...)`.
- Additional failing evidence: returning the same parameter inside
  `Box<island>(value: region)` fails with the same consumed-value diagnostic.
- Control cases: non-generic island alias helpers build and exit `0` in
  `memory_island_alias_region_service.tetra`; direct call-site
  `Box<island>(value: region)` construction builds and exits `0` in
  `memory_island_generic_box_region_service.tetra`; imported generic box
  pass-through plus a non-generic alias exits `0` in
  `compiler_island_generic_pack/app/main.tetra`; actor and task aggregate
  wrappers with parameters build in their corresponding compiler packs.
- Workaround used in microservice examples: construct island aggregate wrappers
  at the call site or pass island handles through non-generic alias helpers;
  avoid returning an island parameter directly inside a struct or generic-box
  constructor.

### TETRA-BUG-0048: Function-typed local, field, and payload calls returning optionals fail as unknown functions

- Status: fixed, verified.
- Area: compiler / callable lowering / optional returns.
- Found while creating:
  `examples/microservices/memory_callable_optional_ptr_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/function_typed_optional_return.tetra" <<'EOF'
func some_int() -> Int?:
    return 42

func main() -> Int:
    let cb: fn() -> Int? = some_int
    let result: Int? = cb()
    if let value = result:
        return value
    return 0
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/function_typed_optional_return.tetra"
```

- Expected: a function-typed local with an optional return type should dispatch
  like the same local returning a non-optional value, then bind the optional
  result normally. The program should build and exit `42`.
- Actual: build fails with `unknown function 'cb'`.
- Additional failing evidence: direct calls through function-typed struct
  fields and enum payload bindings with signature `fn(ptr?) -> ptr?` fail with
  `unknown function 'handler.cb'` and `unknown function 'route_cb'`.
- Control cases: a function-typed local returning `Int` builds and exits `0`;
  a direct named optional-return call builds; routing `fn(ptr?) -> ptr?`
  through a synchronous callback parameter builds and exits `0` in
  `memory_callable_optional_ptr_service.tetra`; the imported helper equivalent
  builds under executable and `--interface-only` `Jobs: 4` modes in
  `compiler_callable_optional_ptr_pack/app/main.tetra`.
- Workaround used in microservice examples: invoke optional-return
  function-typed fields and enum payloads through a non-generic synchronous
  callback parameter such as `apply_optional(cb, value)` instead of calling the
  local, field, or payload binding directly.

### TETRA-BUG-0049: Generic inference fails on generic struct field selections

- Status: fixed, verified.
- Area: compiler / generic inference / field selections.
- Found while creating:
  `examples/microservices/compiler_task_result_generic_pack/app/main.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/generic_field_inference.tetra" <<'EOF'
struct Box<T>:
    value: T

func id<T>(value: T) -> T:
    return value

func main() -> Int:
    let box: Box<Int> = Box<Int>{value: 42}
    let value: Int = id(box.value)
    if value == 42:
        return 0
    return value
EOF
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/app" "$tmp/generic_field_inference.tetra"
```

- Expected: `box.value` has known type `Int`, so generic inference should bind
  `T = Int` for `id<T>(value: T) -> T`; the program should build and exit `0`.
- Actual: build fails with `cannot infer generic argument for 'id' arg 1`.
- Additional failing evidence: the same pattern appears for
  `Box<task.result_i32>`. Calling imported `results.id(boxed.value)` fails with
  `cannot infer generic argument for 'results.id' arg 1`.
- Additional failing evidence: generic optional field selections hit the same
  inference gap. Calling `is_some<T>(value: T?)` as `is_some(box.value)` for
  `MaybeBox<Int>` fails with `cannot infer generic argument for 'is_some' arg
  1`.
- Additional failing evidence: after a typed actor mailbox returns a generic
  `Box<Int>` payload far enough for the payload binding to exist, calling
  `id(reply_box.value)` fails with
  `cannot infer generic argument for 'id' arg 1`.
- Control cases: aliasing the selected field first (`let selected: Int =
  box.value; id(selected)`) builds and exits `0`; direct generic identity over a
  local scalar or local `task.result_i32` also builds. Imported non-generic
  task-result pass-through plus imported generic identity after a local field
  alias exits `0` in `compiler_task_result_generic_pack/app/main.tetra`.
  Unwrapping a generic optional field first, then passing the payload binding to
  a generic function, builds and exits `0`; typed actor payload fields aliased
  to local `Int` offsets exit `0` in the typed actor payload memory examples.
- Workaround used in microservice examples: assign generic struct field
  selections into explicitly typed locals, or unwrap optional fields into local
  payload bindings, before passing them to generic functions.

### TETRA-BUG-0050: Task spawns inside match expressions miss required runtime symbols

- Status: fixed, verified.
- Area: compiler / runtime symbol collection / match expressions / typed tasks.
- Found while creating:
  `examples/microservices/parallel_typed_task_match_catch_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/match_typed_task_spawn.tetra" <<'EOF'
enum Choice:
    case left
    case right

enum TaskErr:
    case stopped
    case code(Int)

func worker() -> Int throws TaskErr:
    throw TaskErr.code(20)

func main() -> Int
uses runtime:
    let choice: Choice = Choice.left
    return match choice:
    case Choice.left:
        catch core.task_join_i32_typed<TaskErr>(core.task_spawn_i32_typed<TaskErr>("worker")):
        case TaskErr.stopped:
            70
        case TaskErr.code(code):
            code
    case Choice.right:
        99
EOF
go run ./cli/cmd/tetra run "$tmp/match_typed_task_spawn.tetra"
```

- Expected: task spawn and typed join inside a `match` expression arm should
  collect the same runtime symbols as the same expression in straight-line code;
  the program should exit `20`.
- Actual: the program lowers far enough to link, but native execution fails with
  `unresolved symbol '__tetra_task_spawn_i32'`. The same pattern with a staged
  five-slot typed error payload fails with
  `unresolved symbol '__tetra_task_result_begin'`.
- Control cases: `typed_task_error_service.tetra` and
  `parallel_typed_task_wide_payload_service.tetra` keep typed task spawn/join in
  straight-line code and exit `0`; a `match` expression without task spawn runs
  and returns its selected arm value.
- Workaround used in microservice examples: route with `match` first, then
  spawn and join typed tasks in straight-line code outside the `match` arm.

### TETRA-BUG-0051: Formatter corrupts nested catch cases inside match expression arms

- Status: fixed, verified.
- Area: formatter / nested expressions / catch inside match.
- Found while creating:
  `examples/microservices/compiler_typed_task_match_pack/app/main.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/match_catch_format.tetra" <<'EOF'
enum Choice:
    case left
    case right

enum TaskErr:
    case stopped
    case code(Int)

func worker() -> Int throws TaskErr:
    throw TaskErr.code(20)

func main() -> Int
uses runtime:
    let choice: Choice = Choice.left
    return match choice:
    case Choice.left:
        catch core.task_join_i32_typed<TaskErr>(core.task_spawn_i32_typed<TaskErr>("worker")):
        case TaskErr.stopped:
            70
        case TaskErr.code(code):
            code
    case Choice.right:
        99
EOF
go run ./cli/cmd/tetra fmt -write "$tmp/match_catch_format.tetra"
go run ./cli/cmd/tetra run "$tmp/match_catch_format.tetra"
```

- Expected: formatter should preserve the nested `catch` cases under the
  `catch ...:` expression inside the `match` arm.
- Actual: formatter dedents `case TaskErr...` to column 1 and leaves the
  sibling `case Choice.right` nested under the corrupted catch body. The next
  parse/build fails with `expected indented block after ':'`.
- Control cases: a `match` expression without nested `catch` remains
  formatter-check clean; moving the typed task `catch` out of the `match` arm is
  formatter-stable in `parallel_typed_task_match_catch_service.tetra` and
  `compiler_typed_task_match_pack/app/main.tetra`.
- Workaround used in microservice examples: avoid `catch` directly nested as a
  `match` expression arm until formatter preserves nested catch indentation.

### TETRA-BUG-0052: Formatter corrupts nested match cases inside catch expression arms

- Status: fixed, verified.
- Area: formatter / nested expressions / match inside catch.
- Found while creating:
  `examples/microservices/memory_typed_task_error_nested_enum_offset_service.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/nested_match_in_catch_format.tetra" <<'EOF'
enum Kind:
    case value(Int)
    case empty

enum Err:
    case nested(Kind)

func fail() -> Int throws Err:
    throw Err.nested(Kind.value(4))

func main() -> Int:
    return catch fail():
    case Err.nested(kind):
        match kind:
        case Kind.value(code):
            code
        case Kind.empty:
            0
EOF
go run ./cli/cmd/tetra run "$tmp/nested_match_in_catch_format.tetra"
go run ./cli/cmd/tetra fmt -write "$tmp/nested_match_in_catch_format.tetra"
go run ./cli/cmd/tetra run "$tmp/nested_match_in_catch_format.tetra"
```

- Expected: formatter should preserve the nested `match` cases under
  `match kind:` inside the `catch` payload arm. The pre-format source runs and
  exits `4`.
- Actual: formatter dedents `case Kind.value(code):` and
  `case Kind.empty:` to column 1. The next parse/build fails with
  `expected indented block after ':'`.
- Control cases: moving the nested `match` into a helper function is
  formatter-stable and exits `0` in
  `memory_typed_task_error_nested_enum_offset_service.tetra` and
  `compiler_typed_error_payload_memory_pack/app/main.tetra`.
- Workaround used in microservice examples: keep nested `match` expressions out
  of `catch` case arms; resolve nested enum payloads through helper functions.

### TETRA-BUG-0053: Awaited optional resource locals lose provenance

- Status: fixed, verified.
- Area: compiler / async resource provenance / optional resource locals.
- Found while creating:
  `examples/microservices/compiler_async_resource_pack/app/main.tetra`, then
  extended while creating
  `examples/microservices/compiler_async_throw_resource_pack/app/main.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/async_optional_resource_local.tetra" <<'EOF'
async func maybe_task(handle: task.i32) -> task.i32?:
    let out: task.i32? = handle
    return out

async func relay_task(handle: task.i32) -> task.i32?:
    let out: task.i32? = await maybe_task(handle)
    return out

func main() -> Int:
    return 42
EOF
go run ./cli/cmd/tetra build --interface-only --jobs 4 -o "$tmp/out" "$tmp/async_optional_resource_local.tetra"
```

- Expected: assigning an awaited optional resource result to a typed local and
  returning that local should preserve the resource provenance, matching scalar
  async optional locals and synchronous optional resource locals.
- Actual: build fails with
  `ambiguous resource provenance for 'out.$elem' after control-flow merge`.
- Control cases:
  - Direct async return builds:

```tetra
async func relay_task(handle: task.i32) -> task.i32?:
    return await maybe_task(handle)
```

  - The same local-binding shape without `async`/`await` builds for
    `task.i32?`.
  - The same local-binding shape with scalar `Int?` after `try await` builds,
    and direct `return try await maybe_task(...)` builds for `task.i32?`.
  - The local-binding failure reproduces for awaited `actor?` and
    `task.group?` values with the same provenance diagnostic. It also
    reproduces for `try await` typed-error propagation over `task.i32?`.
- Workaround used in microservice examples: return awaited optional resource
  values directly from async helpers, including `try await` returns; avoid
  storing awaited optional resources in locals until provenance is preserved
  across that control-flow merge.

### TETRA-BUG-0054: Awaited resource aggregate locals lose provenance

- Status: fixed, verified.
- Area: compiler / async resource provenance / aggregate resource locals.
- Found while creating:
  `examples/microservices/compiler_async_resource_wrapper_pack/app/main.tetra`,
  then extended while creating
  `examples/microservices/compiler_async_throw_resource_wrapper_pack/app/main.tetra`
  and `examples/microservices/compiler_async_generic_resource_pack/app/main.tetra`,
  then extended again with
  `examples/microservices/compiler_async_throw_generic_resource_pack/app/main.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/async_resource_aggregate_local.tetra" <<'EOF'
struct TaskBox:
    handle: task.i32

async func box_task(handle: task.i32) -> TaskBox:
    return TaskBox(handle: handle)

async func local_task_box(handle: task.i32) -> TaskBox:
    let box: TaskBox = await box_task(handle)
    return box

func main() -> Int:
    return 42
EOF
go run ./cli/cmd/tetra build --interface-only --jobs 4 -o "$tmp/out" "$tmp/async_resource_aggregate_local.tetra"
```

- Expected: assigning an awaited resource-bearing aggregate result to a typed
  local and returning that local should preserve resource provenance, matching
  direct awaited aggregate returns, synchronous resource aggregate locals, and
  scalar async aggregate locals.
- Actual: build fails with
  `ambiguous resource provenance for 'box.handle' after control-flow merge`.
- Control cases:
  - Direct async aggregate return builds:

```tetra
async func relay_task_box(handle: task.i32) -> TaskBox:
    return await box_task(handle)
```

  - The same local-binding shape without `async`/`await` builds for
    `TaskBox`.
  - The same local-binding shape with a scalar `IntBox` after `await` builds.
  - The same local-binding shape with a scalar `IntBox` after `try await`
    builds, and direct `return try await box_task(...)` builds for `TaskBox`.
  - The same local-binding shape with a generic scalar `Box<Int>` after
    `await` and `try await` builds, and direct `return await wrap(handle)` /
    `return try await wrap(handle, false)` builds for `Box<task.i32>`,
    `Box<actor>`, and `Box<task.group>`.
  - The local-binding failure reproduces for enum payloads with
    `ambiguous resource provenance for 'slot.$case0.payload0'` and for
    `actor`/`task.group` struct fields with the same aggregate-local pattern.
    It also reproduces for `try await` typed-error propagation over `TaskBox`
    and generic `Box<task.i32>` after `await` or `try await` with
    `ambiguous resource provenance for 'box.value'`.
- Workaround used in microservice examples: return awaited resource aggregate
  values directly from async helpers, including `try await` returns; avoid
  storing awaited resource-bearing aggregates in locals until provenance is
  preserved across that control-flow merge.

### TETRA-BUG-0055: Direct awaited pointer returns ignore await and try

- Status: fixed, verified.
- Area: compiler / async pointer returns / return expression checking.
- Found while creating:
  `examples/microservices/compiler_async_throw_memory_ptr_pack/app/main.tetra`,
  then extended while creating
  `examples/microservices/compiler_async_optional_memory_ptr_pack/app/main.tetra`
  and `examples/microservices/compiler_async_generic_memory_ptr_pack/app/main.tetra`,
  then extended again with
  `examples/microservices/compiler_async_enum_memory_ptr_pack/app/main.tetra`
  and `examples/microservices/compiler_async_slice_memory_pack/app/main.tetra`,
  then extended with
  `examples/microservices/compiler_async_slice_lane_memory_pack/app/main.tetra`,
  then extended with
  `examples/microservices/compiler_async_slice_shape_pack/app/main.tetra`,
  then extended with
  `examples/microservices/compiler_async_string_memory_pack/app/main.tetra`
  and
  `examples/microservices/compiler_async_optional_generic_string_pack/app/main.tetra`.
- Reproduction command:

```sh
tmp=$(mktemp -d)
cat > "$tmp/async_ptr_direct_return.tetra" <<'EOF'
async func derive(base: ptr, offset: Int, memory_cap: cap.mem) -> ptr
uses mem:
    unsafe:
        return core.ptr_add(base, offset, memory_cap)
    return base

async func relay(base: ptr, memory_cap: cap.mem) -> ptr
uses mem:
    return await derive(base, 4, memory_cap)

func main() -> Int:
    return 42
EOF
go run ./cli/cmd/tetra build --interface-only --jobs 4 -o "$tmp/out" "$tmp/async_ptr_direct_return.tetra"
```

- Expected: a direct `return await derive(...)` from an async function returning
  `ptr` should consume the async result, matching scalar direct awaited returns
  and pointer locals populated from awaited calls.
- Actual: build fails with
  `call to async function 'derive' requires await`, even though the call is
  already preceded by `await`. Parenthesizing the awaited expression does not
  change the diagnostic.
- Additional repro:
  - `return try await derive(...)` from a throwing async function returning
    `ptr` fails with `call to throwing function 'derive' requires try`.
  - Direct awaited returns of `struct PtrBox: cell: ptr` and direct
    `try await` returns of the same pointer-bearing aggregate fail with the
    same false missing-`await` / missing-`try` diagnostics.
  - Direct awaited returns of `ptr?` fail with
    `call to async function 'maybe_base' requires await`, and direct
    `try await` returns of `ptr?` fail with
    `call to throwing function 'maybe_base' requires try`.
  - Direct awaited generic `ptr` returns fail with
    `call to async function 'id__T_ptr' requires await`, and direct
    `try await` generic `ptr` returns fail with
    `call to throwing function 'id__T_ptr' requires try`.
  - Direct awaited generic `Box<ptr>` returns fail with
    `call to async function 'wrap__T_ptr' requires await`.
  - Direct awaited enum payload returns of `case raw(ptr)` fail with
    `call to async function 'make_route' requires await`, and direct
    `try await` enum payload returns of the same shape fail with
    `call to throwing function 'make_route' requires try`.
  - Direct awaited `[]u8` returns fail with
    `call to async function 'pass' requires await`, and direct `try await`
    `[]u8` returns fail with `call to throwing function 'pass' requires try`.
  - Direct awaited `SliceBox { bytes: []u8 }` returns fail with
    `call to async function 'box' requires await`.
  - Direct awaited `[]i32`, `[]bool`, and `[]u16` returns fail with
    `call to async function 'pass' requires await`.
  - Direct `try await` `[]i32` returns fail with
    `call to throwing function 'pass' requires try`.
  - Direct awaited `I32SliceBox { values: []i32 }` returns fail with
    `call to async function 'box' requires await`.
  - Direct awaited `[]i32?` returns fail with
    `call to async function 'maybe_values' requires await`, and direct
    `try await` `[]i32?` returns fail with
    `call to throwing function 'maybe_values' requires try`.
  - Direct awaited generic `[]i32` returns fail with
    `call to async function 'id__T__5b__5d_i32' requires await`, and direct
    `try await` generic `[]i32` returns fail with
    `call to throwing function 'id_throw__T__5b__5d_i32' requires try`.
  - Direct awaited generic `Box<[]i32>` returns fail with
    `call to async function 'wrap__T__5b__5d_i32' requires await`.
  - Direct `try await` generic `Box<[]i32>` returns fail with
    `call to throwing function 'wrap_throw__T__5b__5d_i32' requires try`.
  - Direct awaited enum payload returns of `case raw([]i32)` fail with
    `call to async function 'make_route' requires await`.
  - Direct `try await` enum payload returns of `case raw([]i32)` fail with
    `call to throwing function 'make_route_throw' requires try`.
  - Direct awaited `String` returns fail with
    `call to async function 'pass' requires await`, and direct `try await`
    `String` returns fail with `call to throwing function 'pass' requires try`.
  - Direct awaited `StringBox { text: String }` returns fail with
    `call to async function 'box' requires await`.
  - Direct awaited `String?` returns fail with
    `call to async function 'maybe_text' requires await`, and direct
    `try await` `String?` returns fail with
    `call to throwing function 'maybe_text' requires try`.
  - Direct awaited generic `String` returns fail with
    `call to async function 'id__T_str' requires await`, and direct
    `try await` generic `String` returns fail with
    `call to throwing function 'id_throw__T_str' requires try`.
  - Direct awaited generic `Box<String>` returns fail with
    `call to async function 'wrap__T_str' requires await`.
- Control cases:
  - Scalar direct `return try await calc(...)` builds.
  - Scalar optional direct `return await maybe_value(...)` builds.
  - Scalar generic direct `return await id(value)` builds.
  - Scalar enum direct `return await make_route(value)` builds.
  - Scalar aggregate direct `return await box_int(...)` builds.
  - Scalar direct `return await pass(value)` builds.
  - Pointer local usage, generic pointer local usage, and optional-pointer local
    unwrapping after `await` or `try await` build; awaited enum payload locals
    and synchronous enum payload locals also build. Awaited `[]u8`, `[]i32`,
    `[]i32?`, generic `[]i32`, `[]bool`, `[]u16`, enum `[]i32` payloads,
    `String`, `String?`, generic `String`, `StringBox`, `I32SliceBox`, generic
    `Box<[]i32>`, and generic `Box<String>` locals also build after `await` or
    `try await`:

```tetra
async func load_local(base: ptr, memory_cap: cap.mem) -> Int throws MemoryAsyncErr
uses mem:
    let cell: ptr = try await derive(base, 4, memory_cap, false)
    unsafe:
        return core.load_i32(cell, memory_cap)
    return 0
```

- Workaround used in microservice examples: store awaited pointer results in
  locals before using them; unwrap awaited optional pointers from locals; avoid
  direct awaited pointer, optional-pointer, generic-pointer, or
  pointer-aggregate/enum-payload/slice-lane/slice-shape/string,
  optional-string, or generic-string returns until the return-expression
  checker preserves the `await`/`try` wrappers for those result types.

## Microservice Bug-Hunt Runs

| Date | Scope | Evidence | Confirmed bugs |
| --- | --- | --- | --- |
| 2026-05-20 | Added actor/task-based microservice examples for inventory, payments, and orders gateway. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | none |
| 2026-05-20 | Added memory, parallel fanout, and compiler-pipeline microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; direct `id(value())` reproduction command | `TETRA-BUG-0001` |
| 2026-05-20 | Added island cache pool, repeated task pool, and compiler artifact router microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; module-scoped `Unit.score(unit)` control | `TETRA-BUG-0002` |
| 2026-05-20 | Added capability memory journal and grouped task worker microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | none |
| 2026-05-20 | Added typed task errors, task group cancel, wait/select, memory bounds probe, callable router, and modular compiler gateway microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; direct `Route.direct(handler.cb)` reproduction command; formatted typed-alias reproduction command | `TETRA-BUG-0003`, `TETRA-BUG-0004` |
| 2026-05-20 | Added island slice matrix and generic optional router microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | none |
| 2026-05-20 | Added actor deadline router, typed task success, byte memory window, callable return router, and compiler callable pack microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; module-scoped `core.spawn("Router.run")` reproduction command; formatted function-typed global reproduction command | `TETRA-BUG-0005`, `TETRA-BUG-0006` |
| 2026-05-20 | Added tagged actor loop, task group lifecycle, negative memory guard, callable identity router, and throwing callable pack microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | none |
| 2026-05-20 | Added actor poll timeout, task timeout recovery, u16 memory lane, generic struct router, and cross-module generic box microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | none |
| 2026-05-20 | Added typed task-group payload, actor sender snapshot, memory copy window, protocol-bound generic, and cross-module protocol pack microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; derived `core.ptr_add` reproduction command | `TETRA-BUG-0007` |
| 2026-05-20 | Added actor state counter, task-group self-cancel, generic typed-error, and cross-module generic typed-error microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; formatted mutable actor-state reproduction command | `TETRA-BUG-0008` |
| 2026-05-20 | Added task-group current-status, dual actor mailbox, memory memset stride, island bool flags, and cross-module generic pair microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; dual blocking `core.recv_msg()` reproduction command | `TETRA-BUG-0009` |
| 2026-05-20 | Added dual actor value mailbox, dual task deadline, zero-copy memory, and cross-module generic optional box microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; dual blocking `core.recv()` reproduction command; optional field constructor reproduction command | `TETRA-BUG-0010`, `TETRA-BUG-0011` |
| 2026-05-20 | Added actor timeout retry, task poll/deadline matrix, memory pointer table, and optional enum router microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; optional enum constructor reproduction command | `TETRA-BUG-0012` |
| 2026-05-20 | Added optional field update, actor chain reply, grouped task poll, and i32 memory stride microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; direct optional field assignment control command | none |
| 2026-05-20 | Added actor value-chain, typed task-group success, chained pointer stride, and cross-module optional enum microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | none |
| 2026-05-20 | Added typed actor payload, task select timeout, mixed-width memory, and imported extension microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | none |
| 2026-05-20 | Added actor self-mailbox, task-group cancel-after-spawn, derived memory copy, and protocol extension microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; derived pointer parameter loop reproduction command; formatted generic protocol requirement reproduction command; imported generic extension reproduction command | `TETRA-BUG-0013`, `TETRA-BUG-0014`, `TETRA-BUG-0015` |
| 2026-05-20 | Added typed actor chain, multi-cancel task group, derived pointer table, and imported generic function microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; duplicate match payload binding reproduction command; stored derived pointer table reproduction command | `TETRA-BUG-0016`, `TETRA-BUG-0017` |
| 2026-05-20 | Added self typed-mailbox actor, actor/task bridge, aggregate pointer memory, and local generic extension microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; struct pointer-field reproduction command; enum derived-pointer payload reproduction command | `TETRA-BUG-0018`, `TETRA-BUG-0019` |
| 2026-05-20 | Added typed actor/task bridge, task-group actor fanout, optional pointer memory, and callable generic-route microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; generic function-typed identity reproduction command; scalar generic and non-generic callable controls | `TETRA-BUG-0020` |
| 2026-05-20 | Added task actor-roundtrip and actor typed task-group microservice examples; extended optional-derived pointer and generic callback probes. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; optional derived-pointer reproduction command; generic callback reproduction command | `TETRA-BUG-0021`, `TETRA-BUG-0022` |
| 2026-05-20 | Added function/generic pointer memory, task typed-actor roundtrip, actor task-select, and generic optional-route microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; derived pointer function-return reproduction command; base pointer function/generic controls | `TETRA-BUG-0023` |
| 2026-05-20 | Added global memory-state, actor typed task-error bridge, actor cancel/select, and imported generic optional-route microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; global base-pointer reproduction command; global pointer-offset reproduction command | `TETRA-BUG-0024`, `TETRA-BUG-0025` |
| 2026-05-20 | Added mutable pointer memory, struct-offset memory, actor task-recovery, and imported generic nested optional microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; mutable derived-pointer local reproduction command; struct-field pointer-offset reproduction command | `TETRA-BUG-0026`, `TETRA-BUG-0027` |
| 2026-05-20 | Added function offset memory, expression offset memory, actor timer/task matrix, and imported generic enum microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; function-call offset reproduction command; arithmetic-expression offset reproduction command | `TETRA-BUG-0028`, `TETRA-BUG-0029` |
| 2026-05-20 | Added task-result offset memory, actor-message offset memory, actor typed group recovery, and imported generic struct-field microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; runtime field offset reproduction commands; generic struct enum-payload reproduction command | `TETRA-BUG-0030`, `TETRA-BUG-0031` |
| 2026-05-20 | Added indexed/metadata offset memory, payload typed-task handle, typed actor dual mailbox, nested task-group, and imported generic optional-struct microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; indexed and metadata offset reproduction commands; annotated payload typed-task reproduction commands | `TETRA-BUG-0032`, `TETRA-BUG-0033` |
| 2026-05-20 | Added direct-base pointer memory, wide typed-task payload, wide typed-actor payload, and cross-module runtime worker microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; direct allocation/function/load pointer base reproduction commands; local pointer base controls | `TETRA-BUG-0034` |
| 2026-05-20 | Added shared typed actor envelope, logical time-window, actor state status, and inline ptr_add memory microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; mismatched typed actor enum reproduction commands; shared enum control command | `TETRA-BUG-0035` |
| 2026-05-20 | Added structured typed-task payload, structured typed-actor payload, callable pointer-base, and match-expression pointer-base microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; function-typed callable derived-pointer repro/control commands; task/actor sendability controls | `TETRA-BUG-0023` |
| 2026-05-20 | Added typed-error pointer-base, join-until rejoin, actor task-result window, and scalar inout return-value microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; typed-error derived-pointer repro/control; global fixed-array repro/control; scalar inout writeback repro/control | `TETRA-BUG-0036`, `TETRA-BUG-0037`, `TETRA-BUG-0038` |
| 2026-05-20 | Added dynamic base-offset memory, task-group close/cancel lifecycle, and parallel-jobs compiler pack microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; derived dynamic `ptr_add` repro/control; global fixed-array default-read evidence; function-typed `inout` callback evidence | `TETRA-BUG-0039`; extended `TETRA-BUG-0037`, `TETRA-BUG-0038` |
| 2026-05-20 | Added cross-module typed-task pack and explicit selfhost deadline microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; explicit selfhost task-group diagnostic repro/control; derived local-offset evidence for `TETRA-BUG-0039` | `TETRA-BUG-0040`; extended `TETRA-BUG-0039` |
| 2026-05-20 | Added struct/enum/typed-error base dynamic memory, select recovery, and unique pattern-binding compiler microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; if-let/catch/match scoped binding reuse repro/control commands | `TETRA-BUG-0041`; extended `TETRA-BUG-0016` |
| 2026-05-20 | Added base-anchored dynamic memory copy, select rejoin, group cancel/select, and interface-only parallel compiler microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; derived `memory.memcpy_u8` and `memory.memset_u8` repro/control commands; `--jobs 4 --interface-only` compiler pack check | `TETRA-BUG-0042` |
| 2026-05-20 | Added zero-length derived memory-helper, canceled-group spawn, join-until poll, and interface-control compiler microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; task-group cancel-after-close diagnostic probe; derived offset snapshot and width-guard memory probes; formatter `pub enum`/`pub func` repro; `--jobs 4 --interface-only` compiler pack check | `TETRA-BUG-0043`; extended `TETRA-BUG-0042` controls |
| 2026-05-20 | Added base zero-length memory-helper, yield/join window, group status roundtrip, and import-alias compiler microservice examples. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; direct runs for the four new services; selective-import formatter repro/control; public-modifier formatter breadth probe; `--jobs 4 --interface-only` compiler pack check | `TETRA-BUG-0044`; extended `TETRA-BUG-0043` evidence |
| 2026-05-20 | Added heap u16/bool memory slices, actor yield-mailbox, current-group cancel/status, and cross-module actor compiler microservice examples. | Direct runs for the five new services; cross-module actor `--jobs 4 --interface-only` compiler pack check; actor/throws/ownership formatter round-trip controls; task-group cancel/status old-handle and returned-handle probes | none |
| 2026-05-20 | Added heap i32/bool memory slice, task-to-actor deadline, and imported actor-resource struct/enum compiler microservice examples. | Direct runs for the three new services; imported actor-resource `--jobs 4 --interface-only` compiler pack check; cross-module actor struct/enum payload probes; memory i32/bool slice probe; task-to-actor deadline probe | none |
| 2026-05-20 | Added heap u8 memory slice, typed group cancel/status, and imported callable-return compiler microservice examples. | Direct runs for the three new services; imported callable-return `--jobs 4 --interface-only` compiler pack check; `@export` formatter round-trip control; heap u8 slice probe; typed grouped-task cancel/status probe | none |
| 2026-05-20 | Added async-interface compiler pack, optional/enum heap-slice memory, and struct/enum task-result wrapper microservice examples. | Direct runs for the five new services; async-interface `--jobs 4 --interface-only` compiler pack check; async `main` diagnostic probe; async formatter round-trip control; optional/enum `[]u8` payload probes; struct/enum `task.result_i32` wrapper probes | none |
| 2026-05-20 | Added compiler test-command, parallel task test-command, generic typed-result payload, and looped slice-struct memory microservice examples. | `tetra test` runs for both test-command services; direct runs for the four new services; typed-result generic payload probe; looped heap-slice struct-field probe | none |
| 2026-05-20 | Added generic slice-box memory, optional task-result, nested task-spawn, and cross-module generic slice compiler microservice examples. | Direct runs for the four new services; cross-module generic slice `--jobs 4 --interface-only` compiler pack check; explicit generic type-argument boundary probe corrected to inference path | none |
| 2026-05-20 | Added heap-slice for-loop, slice `inout` mutation, optional task-handle join, and cross-module optional task compiler microservice examples. | Direct runs for the four new services; cross-module optional task `--jobs 4 --interface-only` compiler pack check; formatter public-modifier behavior observed as existing `TETRA-BUG-0043` boundary | none |
| 2026-05-20 | Added heap bool/i32 for-loop memory, optional actor-handle send, and cross-module optional actor compiler microservice examples. | Direct runs for the four new services; cross-module optional actor `--jobs 4 --interface-only` compiler pack check; missing `runtime` effect marker probe corrected as expected effect enforcement | none |
| 2026-05-20 | Added heap u16 for-loop memory, optional task-group close/cancel, and cross-module optional task-group compiler microservice examples. | Direct runs for the four new services; cross-module optional group `--jobs 4 --interface-only` compiler pack check; optional task-group spawn reproduction and direct-spawn control commands | `TETRA-BUG-0045` |
| 2026-05-20 | Added bool/i32 slice `inout` memory, optional task-group match-close, and cross-module optional task-group match compiler microservice examples. | Direct runs for the four new services; cross-module optional group match `--jobs 4 --interface-only` compiler pack check; typed optional task-group spawn reproduction and direct typed-spawn control commands | extended `TETRA-BUG-0045` |
| 2026-05-20 | Added struct/enum task-group spawn, typed struct/enum task-group spawn, `[]u16` slice `inout` stride memory, and imported task-group aggregate compiler microservice examples. | Direct runs for the six new services; imported group aggregate `--jobs 4 --interface-only` compiler pack check; aggregate task-group spawn controls kept `TETRA-BUG-0045` scoped to optional payloads | none |
| 2026-05-20 | Added non-generic task-group alias spawn, generic task-group box spawn, optional generic `[]u16` box memory, and imported generic task-group compiler microservice examples. | Direct runs for the four new services; imported group generic `--jobs 4 --interface-only` compiler pack check; optional aggregate task-group spawn probes; generic task-group identity compiler repro/control commands | `TETRA-BUG-0046`; extended `TETRA-BUG-0045` |
| 2026-05-20 | Added non-generic task-handle alias join, generic task-handle box join, optional struct/generic task-handle box joins, optional generic `[]bool` box memory, and imported generic task-handle compiler microservice examples. | Direct runs for the six new services; imported task generic `--jobs 4 --interface-only` compiler pack check; same-module and cross-module generic task-handle identity repro/control commands | extended `TETRA-BUG-0046` |
| 2026-05-20 | Added non-generic actor alias send, generic actor box send, optional struct/generic actor box sends, optional generic `[]i32` box memory, and imported generic actor compiler microservice examples. | Direct runs for the six new services; imported actor generic `--jobs 4 --interface-only` compiler pack check; same-module and cross-module generic actor identity repro/control commands | extended `TETRA-BUG-0046` |
| 2026-05-20 | Added non-generic island alias memory, generic island box memory, optional struct/generic island box memory, and imported generic island compiler microservice examples. | Direct runs for the five new services; imported island generic `--jobs 4 --interface-only` compiler pack check; same-module generic island identity repro; island parameter aggregate-constructor repro/control commands | `TETRA-BUG-0047`; extended `TETRA-BUG-0046` |
| 2026-05-20 | Added non-generic pointer alias base memory, generic pointer identity base memory, and imported generic pointer identity compiler microservice examples. | Direct runs for the three new services; imported pointer generic `--jobs 4 --interface-only` compiler pack check; generic/optional pointer aggregate repro/control commands | extended `TETRA-BUG-0018` |
| 2026-05-20 | Added optional-pointer callable memory and imported optional-pointer callable compiler microservice examples. | Direct runs for the two new services; imported optional-pointer callable `--jobs 4 --interface-only` compiler pack check; function-typed optional-return repro/control commands; generic optional-pointer callback repro/control commands | `TETRA-BUG-0048`; extended `TETRA-BUG-0022` |
| 2026-05-20 | Added optional task-result memory offset, generic task-result box, imported task-result generic compiler, and async optional compiler microservice examples. | Direct runs for the four new services; imported task-result generic and async optional `--jobs 4 --interface-only` compiler pack checks; generic struct field-selection inference repro/control commands | `TETRA-BUG-0049` |
| 2026-05-20 | Added generic optional pointer-field memory, generic optional task-result field, and imported generic optional-field compiler microservice examples. | Direct runs for the three new services; imported generic optional-field `--jobs 4 --interface-only` compiler pack check; generic optional field-selection inference repro/control commands | extended `TETRA-BUG-0049` |
| 2026-05-20 | Added generic optional-pointer call memory, optional-pointer inout return-value memory, and imported optional-pointer generic call compiler microservice examples. | Direct runs for the three new services; imported optional-pointer generic call `--jobs 4 --interface-only` compiler pack check; non-generic/generic optional-pointer inout writeback repro/control commands | extended `TETRA-BUG-0038` |
| 2026-05-20 | Added optional actor alias-send, optional task alias-join, optional task-group alias-close, and imported optional alias resource compiler microservice examples. | Direct runs for the four new services; imported optional alias resource `--jobs 4 --interface-only` compiler pack check; optional task-group alias-spawn repro plus non-optional alias-spawn and optional alias-close controls | extended `TETRA-BUG-0045` |
| 2026-05-20 | Added typed actor optional alias-send, typed task-group optional alias-spawn, and imported typed optional alias resource compiler microservice examples. | Direct runs for the three new services; imported typed optional alias resource `--jobs 4 --interface-only` compiler pack check; typed task generic identity repro plus direct typed-task and typed group optional-alias controls | extended `TETRA-BUG-0046`; narrowed `TETRA-BUG-0045` controls |
| 2026-05-20 | Added typed task match-routing, typed task error-payload memory offsets, and imported typed task match compiler microservice examples. | Direct runs for the three new services; imported typed task match `--jobs 4 --interface-only` compiler pack check; precise typed-handle field/optional/parameter repro commands; typed task spawn-in-match linker repro/control; nested catch-in-match formatter repro/control | extended `TETRA-BUG-0033`; `TETRA-BUG-0050`; `TETRA-BUG-0051` |
| 2026-05-20 | Added typed-error optional pointer-base memory, optional pointer dynamic memory, and imported optional pointer typed-error compiler microservice examples. | Direct runs for the three new services; imported typed-error optional pointer `--jobs 4 --interface-only` compiler pack check; optional derived-pointer typed-error repro/control; struct pointer typed-error repro/control | extended `TETRA-BUG-0036`; extended `TETRA-BUG-0018` controls |
| 2026-05-20 | Added typed actor scalar payload memory, typed actor struct-payload memory, and imported typed actor payload compiler microservice examples. | Direct runs for the three new services; imported typed actor payload memory `--jobs 4 --interface-only` compiler pack check; typed actor scalar-payload offset control; typed actor struct-field direct-offset repro and local-alias control | extended `TETRA-BUG-0027`; clarified `TETRA-BUG-0030` controls |
| 2026-05-20 | Added typed actor nested enum payload memory, typed actor nested enum-struct payload memory, and imported nested enum payload compiler microservice examples. | Direct runs for the three new services; imported typed actor enum payload memory `--jobs 4 --interface-only` compiler pack check; typed actor optional payload value-only diagnostic probe; typed actor generic enum-payload repro; typed actor generic field-selection inference repro/control | extended `TETRA-BUG-0031`; extended `TETRA-BUG-0049` |
| 2026-05-20 | Added optional enum actor-send, task-join, task-group close, task-result memory, and imported optional enum resource compiler microservice examples. | Direct runs for the five new services; imported optional enum resource `--jobs 4 --interface-only` compiler pack check; optional enum task-group spawn repro; typed optional enum task-group spawn repro/control; cross-module repeated payload binding repro/control | extended `TETRA-BUG-0045`; extended `TETRA-BUG-0016` |
| 2026-05-20 | Added typed-task error struct, nested enum, optional, and guarded payload memory plus imported typed-error payload compiler microservice examples. | Direct runs for the five new services; imported typed-error payload memory `--jobs 4 --interface-only` compiler pack check; typed-error generic enum-payload repro; nested match-in-catch formatter repro/control; expression-offset runtime guard repro in imported payload pack before base-split workaround | extended `TETRA-BUG-0031`; `TETRA-BUG-0052`; extended `TETRA-BUG-0029` |
| 2026-05-20 | Added deferred task-group close, deferred memory store, deferred cancel/checkpoint, deferred task-result memory, and imported deferred cleanup compiler microservice examples. | Direct runs for the five new services; imported defer cleanup `--jobs 4 --interface-only` compiler pack check; defer task-group close, nested memory write, cancel/checkpoint, and task-result offset probes | none |
| 2026-05-20 | Added join-until, poll, and select task-result memory offset microservice examples plus imported task-wait memory compiler pack. | Direct runs for the four new services; imported task-wait memory `--jobs 4 --interface-only` compiler pack check; direct `task_join_until_i32(...).value` and `select2_i32(...).value` ptr_add repro/control probes | extended `TETRA-BUG-0030` |
| 2026-05-20 | Added join-until, poll, and select task-result error byte-offset microservice examples plus imported task-wait error memory compiler pack. | Direct runs for the four new services; imported task-wait error memory `--jobs 4 --interface-only` compiler pack check; direct and local-alias `.error` ptr_add probes for join-until timeout, pending poll, and select timeout paths | none; clarified `TETRA-BUG-0030` controls |
| 2026-05-20 | Added actor receive, actor poll, and tagged receive memory offset microservice examples plus imported actor-wait memory compiler pack. | Direct runs for the four new services; imported actor-wait memory `--jobs 4 --interface-only` compiler pack check; direct `recv_until(...).value`, completed `recv_poll().value`, and `recv_msg_until(...).tag` ptr_add repro/control probes | extended `TETRA-BUG-0030` |
| 2026-05-20 | Added actor receive, actor poll, and tagged receive error byte-offset microservice examples plus imported actor-error memory compiler pack. | Direct runs for the four new services; imported actor-error memory `--jobs 4 --interface-only` compiler pack check; direct and local-alias `.error` ptr_add probes for receive timeout, empty poll, and tagged receive timeout paths | none; clarified `TETRA-BUG-0030` controls |
| 2026-05-20 | Added task-group status direct-offset memory, current-status memory, direct cancel-close lifecycle, and imported group-status memory compiler examples. | Direct runs for the four new services; imported group-status memory `--jobs 4 --interface-only` compiler pack check; direct/local `task_group_status` ptr_add probes for open, canceled, and closed groups; nested `task_group_cancel` into status/close probes | none; clarified `TETRA-BUG-0028` controls |
| 2026-05-20 | Added deferred typed-error memory store, deferred return memory store, typed-task deferred actor reply, and imported defer-unwind compiler examples. | Direct runs for the four new services; imported defer-unwind `--jobs 4 --interface-only` compiler pack check; deferred derived-cell typed-error repro plus allocation-base unwind/return controls; typed-task defer actor reply probe | extended `TETRA-BUG-0013`; controls for defer unwind |
| 2026-05-20 | Added imported async-memory compiler examples with allocation-base `ptr`, `cap.mem`, and scalar `inout` async helper signatures. | Direct run for the new async-memory fallback; imported async-memory `--jobs 4 --interface-only` compiler pack check; borrowed async pointer/slice probes rejected with expected ownership diagnostics | none |
| 2026-05-20 | Added imported async-resource compiler examples with optional task, actor, and task-group handles plus task/group memory fallback. | Direct run for the new async-resource fallback; imported async-resource `--jobs 4 --interface-only` compiler pack check; awaited optional resource local repro plus direct-return and sync local controls | `TETRA-BUG-0053` |
| 2026-05-20 | Added imported throwing async-resource compiler examples with optional task, actor, and task-group handles plus task/group memory fallback. | Direct run for the new throwing async-resource fallback; imported throwing async-resource `--jobs 4 --interface-only` compiler pack check; `try await` optional resource local repro plus direct `try await` and scalar local controls | extended `TETRA-BUG-0053` |
| 2026-05-20 | Added imported async resource-wrapper compiler examples with task, actor, and task-group handles in struct and enum aggregates. | Direct run for the new async resource-wrapper fallback; imported async resource-wrapper `--jobs 4 --interface-only` compiler pack check; awaited aggregate-local repro plus direct-return, sync-local, and scalar-aggregate controls | `TETRA-BUG-0054` |
| 2026-05-20 | Added imported throwing async resource-wrapper compiler examples with task, actor, and task-group handles in struct and enum aggregates. | Direct run for the new throwing async resource-wrapper fallback; imported throwing async resource-wrapper `--jobs 4 --interface-only` compiler pack check; `try await` aggregate-local repro plus direct `try await` and scalar-aggregate controls | extended `TETRA-BUG-0054` |
| 2026-05-20 | Added imported async generic-resource compiler examples with scalar, task, actor, and task-group values in generic boxes. | Direct run for the new async generic-resource fallback; imported async generic-resource `--jobs 4 --interface-only` compiler pack check; generic awaited aggregate-local repro plus direct resource-return and scalar generic-local controls | extended `TETRA-BUG-0054` |
| 2026-05-20 | Added imported throwing async generic-resource compiler examples with scalar, task, actor, and task-group values in generic boxes. | Direct run for the new throwing async generic-resource fallback; imported throwing async generic-resource `--jobs 4 --interface-only` compiler pack check; generic `try await` aggregate-local repro plus direct `try await` resource-return and scalar generic-local controls | extended `TETRA-BUG-0054` |
| 2026-05-20 | Added imported throwing async memory-pointer compiler examples with local pointer usage and scalar aggregate direct-return controls. | Direct run for the new throwing async memory-pointer fallback; imported throwing async memory-pointer `--jobs 4 --interface-only` compiler pack check; direct awaited pointer-return repro plus throwing pointer-return, pointer-aggregate direct-return, local pointer, scalar direct, and scalar aggregate controls | `TETRA-BUG-0055` |
| 2026-05-20 | Added imported async optional memory-pointer compiler examples with awaited local unwrapping and scalar optional direct-return controls. | Direct run for the new async optional memory-pointer fallback; imported async optional memory-pointer `--jobs 4 --interface-only` compiler pack check; direct awaited `ptr?` repro plus throwing `ptr?` direct-return, awaited optional-pointer local, sync optional-pointer local, and scalar optional direct controls | extended `TETRA-BUG-0055` |
| 2026-05-20 | Added imported async generic memory-pointer compiler examples with awaited generic local pointer usage and scalar generic direct-return controls. | Direct run for the new async generic memory-pointer fallback; imported async generic memory-pointer `--jobs 4 --interface-only` compiler pack check; direct awaited generic `ptr` repro plus throwing generic `ptr`, generic `Box<ptr>` direct-return, generic pointer local, and scalar generic direct controls | extended `TETRA-BUG-0055` |
| 2026-05-20 | Added imported async enum memory-pointer compiler examples with awaited local enum unwrapping and scalar enum direct-return controls. | Direct run for the new async enum memory-pointer fallback; imported async enum memory-pointer `--jobs 4 --interface-only` compiler pack check; direct awaited enum `ptr` payload repro plus throwing enum direct-return, awaited enum local, sync enum local, and scalar enum direct controls | extended `TETRA-BUG-0055` |
| 2026-05-20 | Added imported async slice-memory compiler examples with awaited local `[]u8` usage and scalar direct-return controls. | Direct run for the new async slice-memory fallback; imported async slice-memory `--jobs 4 --interface-only` compiler pack check; direct awaited `[]u8` repro plus throwing `[]u8`, `SliceBox`, awaited slice local, and scalar direct controls | extended `TETRA-BUG-0055` |
| 2026-05-20 | Added imported async slice-lane memory compiler examples with awaited local `[]i32`, `[]bool`, `[]u16`, and `I32SliceBox` usage. | Direct run for the new async slice-lane fallback; imported async slice-lane `--jobs 4 --interface-only` compiler pack check; direct awaited `[]i32`, throwing `[]i32`, `[]bool`, `[]u16`, `I32SliceBox`, awaited slice-lane locals, and scalar direct controls | extended `TETRA-BUG-0055` |
| 2026-05-20 | Added imported async slice-shape compiler examples with awaited local `[]i32?`, generic `[]i32`, generic `Box<[]i32>`, and enum `[]i32` payload usage. | Direct run for the new async slice-shape fallback; imported async slice-shape `--jobs 4 --interface-only` compiler pack check; direct awaited `[]i32?`, generic `[]i32`, throwing generic `[]i32`, generic `Box<[]i32>`, throwing generic `Box<[]i32>`, enum `[]i32`, throwing enum `[]i32`, and awaited local controls | extended `TETRA-BUG-0055` |
| 2026-05-20 | Added imported async string-memory compiler examples with awaited local `String` and `StringBox` usage plus scalar direct-return controls. | Direct run for the new async string-memory fallback; imported async string-memory `--jobs 4 --interface-only` compiler pack check; direct awaited `String` repro plus throwing `String`, `StringBox`, awaited string local, awaited `StringBox` local, and scalar direct controls | extended `TETRA-BUG-0055` |
| 2026-05-20 | Added imported async optional/generic string compiler examples with awaited local `String?`, generic `String`, and generic `Box<String>` usage. | Direct run for the new async optional/generic string fallback; imported async optional/generic string `--jobs 4 --interface-only` compiler pack check; direct awaited `String?`, generic `String`, throwing generic `String`, generic `Box<String>`, and awaited local controls | extended `TETRA-BUG-0055` |
| 2026-05-20 | Closed all active bug-ledger entries and promoted focused regressions for formatter output, callable/generic lowering, scoped binding reuse, resource provenance, task/actor wrappers, and async resource returns. | `go test ./compiler -run 'TestTetraBug|TestMicroserviceExamplesAndBugLedger' -count=1`; `go test ./compiler/tests/ownership ./compiler/internal/lower ./compiler/tests/callables -count=1` | all listed `TETRA-BUG` entries fixed and verified |

## Notes

- The current v0.4.0 production service surface is actor/task based on
  Linux-x64. A stable Tetra HTTP server API was not found in the supported
  surface; that is recorded as a scope boundary, not a confirmed language bug.
- Future confirmed bugs should include the failing source, command, expected
  behavior, actual behavior, and whether the issue is parser, checker,
  lowering, runtime, or tooling.
