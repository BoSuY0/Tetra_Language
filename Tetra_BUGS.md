# Tetra Bugs

Status: fixed bug ledger for microservice-driven Tetra language testing.

Updated: 2026-05-23. All entries below are fixed and verified by promoted
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
go run ./cli/cmd/tetra fmt -write "$tmp/app/main.tetra"
go run ./cli/cmd/tetra build --target linux-x64 -o "$tmp/formatted" "$tmp/app/main.tetra"
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

### TETRA-BUG-0056: Go workspace validation fails the SCRAM local benchmark package

- Status: fixed, verified.
- Area: tooling / Go workspace module configuration.
- Found while verifying backend web-stack microservice and TechEmpower-adjacent
  bug-hunt tests after adding:
  `examples/microservices/backend_http_pipeline_gateway_service.tetra` and
  `examples/microservices/backend_postgres_prepared_pipeline_service.tetra`.
- Reproduction command:

```sh
go test ./benchmarks/techempower/tetra/cmd/scram-local-bench -count=1
```

- Expected: the SCRAM local benchmark package tests run under the repository
  `go.work` workspace, matching the nearby backend runtime package checks.
- Actual: workspace-mode dependency validation fails before the package tests
  run:

```text
tetra_language/compiler@v0.0.0: malformed module path "tetra_language/compiler": missing dot in first path element
```

- Control case: running the same package with `GOWORK=off` passes, proving the
  package itself builds and the failure begins at the workspace/module boundary.
- Root cause: `cli/go.mod` required the local `tetra_language/compiler`
  workspace module without an explicit local `replace`, so newer Go workspace
  validation treated the underscore module path as a remote dependency path.
- Fix: add `replace tetra_language/compiler => ../compiler` to `cli/go.mod`.
- Verification:

```sh
go test ./benchmarks/techempower/tetra/cmd/scram-local-bench -count=1
go list -m all
```

- Workaround before the fix: run the benchmark package with `GOWORK=off`, but
  that bypasses the repository workspace and does not verify workspace
  compatibility.

### TETRA-BUG-0057: Zero-length heap slice constructors call mmap with length 0

- Status: fixed, verified.
- Area: compiler / native backend slice allocation.
- Found while creating:
  `examples/microservices/backend_time_collection_window_service.tetra`.
- Reproduction command:

```sh
go test ./compiler -run TestBuildMakeZeroLengthSlices -count=1
```

- Expected: zero-length heap slice constructors produce an empty slice that can
  be iterated without element access, matching `lib.core.collections` fallback
  helpers for empty `[]i32` values.
- Actual before the fix: the program exits with code `2` before reaching the
  empty iteration fallback.
- Control case: non-empty `make_i32` slices continued to build and run through
  `TestBuildMakeI32Slice`; zero-length stdlib memory helper examples also
  already treated length `0` as a non-accessing operation.
- Root cause: native make-slice emitters passed byte length `0` directly to
  `mmap`; Linux/macOS reject zero-length mappings, and the existing mmap
  failure guard converted that into runtime exit code `2`.
- Fix: native make-slice emission now returns `(ptr=0, len=0)` without calling
  `mmap` when the requested logical slice length is zero. The same guard was
  added to the linux-x86 and Win64 emitter paths, and the compiler cache ABI
  discriminator was bumped so stale native objects compiled with the old
  allocation path are not reused.
- Verification:

```sh
go test ./compiler -run 'TestBuildMakeZeroLengthSlices|TestMicroserviceExamplesAndBugLedger' -count=1
go test ./compiler/internal/backend/linux_x86 ./compiler/internal/backend/x64abi ./compiler/internal/backend/x64core ./compiler/internal/cache -count=1
```

- Workaround before the fix: avoid constructing zero-length heap slices with
  `make_u8(0)`, `make_u16(0)`, `make_i32(0)`, or `make_bool(0)` on native
  targets.

### TETRA-BUG-0058: Explicit source-file CLI inputs inherit capsule source roots

- Status: fixed, verified.
- Area: CLI / project discovery.
- Found while verifying:
  `examples/microservices/backend_capsule_source_root_service`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra check examples/projects/dogfood_cli/src/main.tetra
```

- Expected: an explicit source-file input is checked as the requested file,
  matching `tetra test examples/projects/dogfood_cli/src/main.tetra` and the
  direct file smoke registry behavior.
- Actual before the fix: `check`, `build`, and `run` discovered the surrounding
  capsule and forced the file through capsule source-root validation:

```text
module 'examples.projects.dogfood_cli.src.main' must be in src/examples/projects/dogfood_cli/src/main.t4
```

- Control case: `go run ./cli/cmd/tetra test --target linux-x64
  examples/projects/dogfood_cli/src/main.tetra` passed because the test command
  already treated explicit source-file inputs as standalone files.
- Root cause: `resolveCLIInput` returned project `WorldOptions` for explicit
  file paths inside a discovered capsule unless the input was outside the
  project. That made `check`, `build`, and `run` disagree with `test` and with
  explicit-file expectations.
- Fix: `resolveCLIInput` now applies capsule project options only to project
  references (empty input, project directory, or capsule manifest path). Explicit
  non-project source files are returned as standalone inputs with no project
  world options.
- Verification:

```sh
go test ./cli/cmd/tetra -run TestBuildCheckRunCommandsAcceptExplicitProjectSourceFile -count=1
go run ./cli/cmd/tetra check examples/projects/dogfood_cli/src/main.tetra
go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-dogfood-cli-file examples/projects/dogfood_cli/src/main.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/projects/dogfood_cli/src/main.tetra
```

### TETRA-BUG-0059: HTTP Connection-close detection is case-sensitive

- Status: fixed, verified.
- Area: stdlib / backend HTTP helpers.
- Found while creating:
  `examples/microservices/backend_http_status_matrix_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra
```

- Expected: `lib.core.http.request_keep_alive(...)` and
  `request_keep_alive_bytes_at(...)` treat `Connection: close` case
  insensitively for ASCII header names and values, matching HTTP header
  semantics and the existing exact-case close behavior.
- Actual before the fix: the service exited with status `22` because
  `connection: close` was not recognized as a close request, so keep-alive
  stayed enabled.
- Control cases: exact `Connection: close` returned non-keep-alive, and a
  request without the header remained keep-alive.
- Root cause: `connection_close_next_state` compared raw byte values for the
  literal `Connection: close` pattern. The state machine started with uppercase
  `C` and then required lowercase letters, so lowercase header names and mixed
  case values missed the close path.
- Fix: normalize ASCII letters inside `connection_close_next_state` before
  comparing the pattern. Both string and byte-buffer helpers share this state
  machine.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1
```

### TETRA-BUG-0060: HTTP Connection-close detection rejects optional whitespace

- Status: fixed, verified.
- Area: stdlib / backend HTTP helpers.
- Found while creating:
  `examples/microservices/backend_http_header_whitespace_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_header_whitespace_service.tetra
```

- Expected: `lib.core.http.request_keep_alive(...)` and
  `request_keep_alive_bytes_at(...)` recognize `Connection: close` when the
  colon is followed by HTTP optional whitespace such as multiple spaces or a
  horizontal tab.
- Actual before the fix: the service exited with status `3` because
  `Connection:   close` was not recognized as a close request, so keep-alive
  stayed enabled. The tab variant was behind the same state-machine boundary.
- Control cases: exact `Connection: close` returned non-keep-alive, and a
  request without the header remained keep-alive.
- Root cause: after matching `Connection:`, `connection_close_next_state`
  required exactly one ASCII space before matching `close`.
- Fix: keep the state machine in the whitespace state for ASCII space and tab,
  and transition to the `close` matcher once the first `c` arrives. The string
  and byte-buffer helpers share this state machine.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_header_whitespace_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1
```

### TETRA-BUG-0061: HTTP Connection-close detection ignores close after comma

- Status: fixed, verified.
- Area: stdlib / backend HTTP helpers.
- Found while creating:
  `examples/microservices/backend_http_connection_list_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_list_service.tetra
```

- Expected: `lib.core.http.request_keep_alive(...)` and
  `request_keep_alive_bytes_at(...)` recognize `close` as a token in a
  comma-separated `Connection` header value such as `Connection: keep-alive,
  close`.
- Actual before the fix: the service exited with status `4` because
  `Connection: keep-alive, close` was not recognized as a close request, so
  keep-alive stayed enabled.
- Control cases: exact `Connection: close` returned non-keep-alive,
  `Connection: keep-alive` remained keep-alive, and the previously fixed
  case-insensitive/optional-whitespace paths continued to pass.
- Root cause: after matching `Connection:`, `connection_close_next_state`
  required the next value token to be `close`; any other token reset the state
  instead of scanning the header value until a comma introduced the next token.
- Fix: add a value-scan state that skips non-matching `Connection` tokens until
  a comma, then re-enters the optional-whitespace/token matcher. The string and
  byte-buffer helpers share this state machine.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_list_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_header_whitespace_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1
```

### TETRA-BUG-0062: HTTP Connection-close detection matches suffix headers

- Status: fixed, verified.
- Area: stdlib / backend HTTP helpers.
- Found while creating:
  `examples/microservices/backend_http_connection_scope_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_scope_service.tetra
```

- Expected: `lib.core.http.request_keep_alive(...)` and
  `request_keep_alive_bytes_at(...)` only apply close semantics to the
  `Connection` header field, not suffix matches inside unrelated fields such as
  `X-Connection: close` or `Proxy-Connection: close`.
- Actual before the fix: the service exited with status `1` because
  `X-Connection: close` was treated as a real `Connection: close` header.
- Control cases: exact `Connection: close` still returned non-keep-alive, while
  the previously fixed case-insensitive, optional-whitespace, and token-list
  paths continued to pass.
- Root cause: `connection_close_next_state` started matching `Connection` at
  any `c` byte in the request buffer, so suffix header names could trigger the
  close path.
- Fix: add an in-line scan state so the matcher only starts at the beginning of
  the input or immediately after LF. Non-matching header lines are skipped until
  the next LF before matching can begin again.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_scope_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_list_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_header_whitespace_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1
```

### TETRA-BUG-0063: HTTP Connection-close detection accepts close token prefixes

- Status: fixed, verified.
- Area: stdlib / backend HTTP helpers.
- Found while creating:
  `examples/microservices/backend_http_connection_token_boundary_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_token_boundary_service.tetra
```

- Expected: `lib.core.http.request_keep_alive(...)` and
  `request_keep_alive_bytes_at(...)` only treat `close` as a complete
  comma-separated `Connection` token. Values such as `closex` or
  `close-upgrade` should not disable keep-alive.
- Actual before the fix: the service exited with status `1` because
  `Connection: closex` was treated as a close request.
- Control cases: `Connection: close, keep-alive` and `Connection: close   `
  still returned non-keep-alive, while `enclose` stayed keep-alive.
- Root cause: `connection_close_next_state` returned the terminal matched state
  immediately after reading the `e` in `close`, without checking the next byte
  for a token delimiter.
- Fix: add a pending-close state after `close`; it only becomes a terminal
  close match when the next byte is a valid token delimiter: space, tab, comma,
  CR, or LF. Non-delimiter bytes fall back to scanning the rest of the header
  value.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_token_boundary_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_scope_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_list_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_header_whitespace_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1
```

### TETRA-BUG-0064: HTTP/1.1 detection scans beyond the request line

- Status: fixed, verified.
- Area: stdlib / backend HTTP helpers.
- Found while creating:
  `examples/microservices/backend_http_version_scope_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_version_scope_service.tetra
```

- Expected: `lib.core.http.route_tech_empower(...)` and
  `request_keep_alive(...)` should only accept an exact `HTTP/1.1` version on
  the request line. `HTTP/1.1` inside a header value, or a request-line version
  prefix such as `HTTP/1.10`, should not make the request HTTP/1.1.
- Actual before the fix: the service exited with status `1` because
  `GET /plaintext HTTP/1.0` with `X-Debug: HTTP/1.1` was routed as plaintext.
- Control cases: real `GET /queries?queries=7 HTTP/1.1` and byte-buffer
  `GET /db HTTP/1.1` requests still route correctly and remain keep-alive by
  default.
- Root cause: `contains_http11_marker` and `contains_http11_marker_bytes_at`
  scanned the entire request buffer for ` HTTP/1.1`, returned success as soon
  as the final `1` was read, and did not require the match to be on the first
  line or followed by a line terminator.
- Fix: make both helpers stop at the first request-line CR/LF unless an exact
  ` HTTP/1.1` has just been matched, and only return success when that exact
  version is followed by CR or LF.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_version_scope_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_token_boundary_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_scope_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_list_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_header_whitespace_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1
```

### TETRA-BUG-0065: HTTP route helper classifies empty request targets as not found

- Status: fixed, verified.
- Area: stdlib / backend HTTP helpers.
- Found while creating:
  `examples/microservices/backend_http_request_target_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_target_guard_service.tetra
```

- Expected: `lib.core.http.route_tech_empower(...)` and
  `route_tech_empower_bytes_at(...)` should return `route_bad_request()` for
  malformed request lines whose request target is empty or starts with `?`.
- Actual before the fix: the service exited with status `1` because
  `GET  /json HTTP/1.1` ended the target before any path byte and returned
  `route_not_found()` instead of `route_bad_request()`.
- Control cases: `GET / HTTP/1.1` and `GET /?debug=1 HTTP/1.1` still return
  `route_not_found()`, while `/plaintext?debug=1` and `/json` continue to
  route normally.
- Root cause: both route helpers treated the first space or `?` after `GET `
  as a valid target terminator even when `path_pos == 0`, then fell through to
  the normal route-miss result.
- Fix: return `route_bad_request()` when the target terminator appears before
  any path byte in both the string and byte-buffer routing paths.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_target_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_pipeline_gateway_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_response_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_version_scope_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1
```

### TETRA-BUG-0066: HTTP/1.1 detection accepts extra request-line tokens

- Status: fixed, verified.
- Area: stdlib / backend HTTP helpers.
- Found while creating:
  `examples/microservices/backend_http_request_line_token_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_line_token_guard_service.tetra
```

- Expected: `lib.core.http.route_tech_empower(...)`,
  `route_tech_empower_bytes_at(...)`, and keep-alive helpers should only treat
  `HTTP/1.1` as valid when it is the third request-line token.
- Actual before the fix: the service exited with status `1` because
  `GET /json debug HTTP/1.1` was routed as `/json` instead of being rejected as
  a bad request.
- Control cases: valid `GET /plaintext HTTP/1.1`,
  `GET /queries?queries=7 HTTP/1.1`, and byte-buffer `/db` requests still
  route normally and remain keep-alive by default.
- Root cause: `contains_http11_marker` and `contains_http11_marker_bytes_at`
  still matched ` HTTP/1.1` after any space on the request line. The route
  helpers had already stopped parsing the target at the first post-target
  space, so extra middle tokens were ignored.
- Fix: make both helpers parse the first line as method token, request-target
  token, then exact `HTTP/1.1`; if the version token is not immediately after
  the target token or is not followed by CR/LF, the helper returns false.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_line_token_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_target_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_version_scope_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_pipeline_gateway_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1
```

### TETRA-BUG-0067: HTTP keep-alive detection accepts malformed request targets

- Status: fixed, verified.
- Area: stdlib / backend HTTP helpers.
- Found while creating:
  `examples/microservices/backend_http_keep_alive_target_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_keep_alive_target_guard_service.tetra
```

- Expected: `lib.core.http.request_keep_alive(...)` and
  `request_keep_alive_bytes_at(...)` should reject malformed request-lines
  whose request target does not start with `/`.
- Actual before the fix: the service exited with status `1` because
  `GET noslash HTTP/1.1` returned keep-alive even though routing and the Go
  HTTP parser classify that target as malformed.
- Control cases: `GET / HTTP/1.1`, `GET /?debug=1 HTTP/1.1`, and
  `GET /json HTTP/1.1` still remain keep-alive by default, while
  `Connection: close` still disables keep-alive.
- Root cause: `contains_http11_marker` and `contains_http11_marker_bytes_at`
  only required a non-empty second request-line token before `HTTP/1.1`; they
  did not verify that the request target was origin-form and started with `/`.
- Fix: require the first request-target byte to be `/` in both string and
  byte-buffer HTTP/1.1 marker scanners.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_keep_alive_target_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_target_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_line_token_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_version_scope_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_pipeline_gateway_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1
```

### TETRA-BUG-0068: HTTP Connection-close detection scans request bodies

- Status: fixed, verified.
- Area: stdlib / backend HTTP helpers.
- Found while creating:
  `examples/microservices/backend_http_connection_body_scope_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_body_scope_service.tetra
```

- Expected: `lib.core.http.contains_connection_close(...)` and
  `contains_connection_close_bytes_at(...)` should only match
  `Connection: close` before the empty header terminator.
- Actual before the fix: the service exited with status `1` because
  `Connection: close` in the request body after `\r\n\r\n` disabled
  keep-alive.
- Control cases: real header `Connection: close` still disables keep-alive,
  while requests whose body merely contains that text remain keep-alive.
- Root cause: the Connection-close matcher scanned the whole request buffer and
  had no independent header-end state, so body lines could start a fresh
  `Connection` header match after the blank line.
- Fix: add a shared header-end state helper and stop both string and
  byte-buffer Connection-close scans at CRLFCRLF or LF-only blank-line
  terminators unless a real header match has already returned true.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_body_scope_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_token_boundary_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_scope_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_list_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_header_whitespace_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1
```

### TETRA-BUG-0069: HTTP keep-alive detection accepts malformed method tokens

- Status: fixed, verified.
- Area: stdlib / backend HTTP helpers.
- Found while creating:
  `examples/microservices/backend_http_keep_alive_method_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_keep_alive_method_guard_service.tetra
```

- Expected: `lib.core.http.request_keep_alive(...)` and
  `request_keep_alive_bytes_at(...)` should reject request-lines whose method
  token contains invalid HTTP token bytes such as `:` or tab.
- Actual before the fix: the service exited with status `1` because
  `GE:T /json HTTP/1.1` returned keep-alive.
- Control cases: valid `GET /json HTTP/1.1` and `POST /json HTTP/1.1` remain
  keep-alive by default, while `POST ... Connection: close` still disables
  keep-alive.
- Root cause: `contains_http11_marker` and `contains_http11_marker_bytes_at`
  counted every non-space byte before the first space as part of the method
  token, without applying HTTP token-character validation.
- Fix: add a shared HTTP token-character predicate and reject invalid method
  bytes before accepting the request-line as HTTP/1.1.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_keep_alive_method_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_keep_alive_target_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_line_token_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_version_scope_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_pipeline_gateway_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1
```

### TETRA-BUG-0070: JSON lowercase hex digit helper returns non-hex bytes out of range

- Status: fixed, verified.
- Area: stdlib / backend JSON helpers.
- Found while creating:
  `examples/microservices/backend_json_hex_digit_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_hex_digit_guard_service.tetra
```

- Expected: public `lib.core.json.hex_digit_lower(...)` should always return
  a lowercase hexadecimal ASCII byte. Out-of-range inputs should saturate to
  `0` for negatives and `f` for values above `15`.
- Actual before the fix: the service exited with status `5` because
  `hex_digit_lower(-1)` returned a byte before ASCII `0`; values above `15`
  could similarly return non-hex letters.
- Control cases: normal `0`, `9`, `10`, and `15` inputs still return
  `0`, `9`, `a`, and `f`; message-object serialization still writes expected
  escaped newline output.
- Root cause: `hex_digit_lower` assumed callers always pass a nibble and
  directly calculated `48 + value` or `87 + value`.
- Fix: saturate negative inputs to ASCII `0` and values above `15` to ASCII
  `f` before the existing digit/letter calculation.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_hex_digit_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_control_matrix_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_escape_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/jsonrt -count=1
```

### TETRA-BUG-0071: Time duration helpers wrap positive overflow

- Status: fixed, verified.
- Area: stdlib / backend time helpers.
- Found while creating:
  `examples/microservices/backend_time_overflow_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_time_overflow_guard_service.tetra
```

- Expected: duration helpers should keep non-negative duration arithmetic
  within the `Int` range: `millis_from_seconds(2147484)` and overflowing
  positive `add_duration_ms` calls should saturate to `2147483647`, while
  negative duration underflow should clamp to `0`.
- Actual before the fix: the service exited with status `2` because
  `millis_from_seconds(2147484)` multiplied past `Int` max and wrapped instead
  of saturating; positive `add_duration_ms` overflow could similarly wrap
  negative and be misclassified as a zero-duration clamp.
- Control cases: the last safe second conversion `2147483 -> 2147483000`,
  safe positive addition near max, negative delta subtraction, and
  `seconds_from_millis(2147483647) -> 2147483` remain stable.
- Root cause: `millis_from_seconds` multiplied before checking the overflow
  boundary, and `add_duration_ms` added before guarding positive overflow.
- Fix: pre-check the seconds conversion boundary, pre-check positive
  millisecond addition against `2147483647 - delta`, and keep negative
  duration results clamped to `0`.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_time_overflow_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_time_collection_window_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0072: PostgreSQL C-string scan traps on negative start

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL wire helpers.
- Found while creating:
  `examples/microservices/backend_postgres_cstring_bounds_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_cstring_bounds_guard_service.tetra
```

- Expected: `lib.core.postgres.cstring_end_at(...)` should return `-1` for
  ranges that cannot contain a bounded C-string terminator, including negative
  start indexes and reversed `limit < start` ranges.
- Actual before the fix: the service exited with runtime status `1` before it
  could return its guard code because `cstring_end_at(payload, -1, 6)` read the
  caller-owned payload at a negative index.
- Control cases: valid searches from `0..6` and `3..6` still locate the
  expected NUL bytes, and empty/reversed non-negative ranges return `-1`.
- Root cause: `cstring_end_at` initialized `pos` from `start` and entered
  `while pos < limit` without first rejecting negative `start`.
- Fix: return `-1` when `start < 0` or `limit < start` before scanning.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_cstring_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_session_state_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_result_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0073: PostgreSQL signed DataRow length leaks malformed negative values

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL wire helpers.
- Found while creating:
  `examples/microservices/backend_postgres_data_row_length_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_length_guard_service.tetra
```

- Expected: PostgreSQL signed DataRow length helpers should normalize any
  negative signed length field to the public sentinel `-1`, including malformed
  values such as `0xfffffffe`, while preserving valid positive lengths.
- Actual before the fix: the service exited with status `4` because
  `read_i32_be_signed(...)` recognized only `0xffffffff`; the malformed
  `0xfffffffe` length leaked as a distinct negative value instead of the
  stable `-1` sentinel.
- Control cases: a valid `2` length still parses `"42"`, a valid following
  column still parses `"7"`, and DataRow value start/i32 helpers keep treating
  negative length fields as missing values.
- Root cause: `read_i32_be_signed` special-cased only the PostgreSQL NULL
  marker bytes instead of normalizing the signed i32 result whenever the high
  bit is set.
- Fix: read the big-endian i32 once and return `-1` for any negative result.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_length_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_result_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_cstring_bounds_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0074: PostgreSQL ASCII i32 parser traps on negative start

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL wire helpers.
- Found while creating:
  `examples/microservices/backend_postgres_ascii_i32_bounds_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ascii_i32_bounds_guard_service.tetra
```

- Expected: `lib.core.postgres.parse_ascii_i32_at(...)` should return `0` for
  invalid bounded parse ranges such as negative `start`, just as it already
  returns `0` for empty ranges and non-digit leading bytes.
- Actual before the fix: the service exited with runtime status `1`; the
  service has no `return 1` branch, so `parse_ascii_i32_at(bytes, -1, 2)`
  trapped before it could produce the expected `0`.
- Control cases: signed `-79`, unsigned `79`, non-digit start, and zero-count
  ranges still return their existing values.
- Root cause: `parse_ascii_i32_at` assigned `pos = start` and read `src[pos]`
  for the optional sign check without first rejecting negative `start`.
- Fix: return `0` for `start < 0`, and also return `0` if `start + count`
  wraps below `start`.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ascii_i32_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_result_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_length_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0075: PostgreSQL CommandComplete parser traps on negative start

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL wire helpers.
- Found while creating:
  `examples/microservices/backend_postgres_command_tag_bounds_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_command_tag_bounds_guard_service.tetra
```

- Expected: `lib.core.postgres.command_complete_affected_rows(...)` should
  return `0` for invalid bounded parse ranges such as negative `start`,
  zero/negative `payload_len`, or wrapped `start + payload_len`.
- Actual before the fix: the service exited with runtime status `1`; the
  service has no `return 1` branch, so
  `command_complete_affected_rows(update_tag, -1, 4)` trapped before it could
  return the expected `0`.
- Control cases: `INSERT 0 3`, `UPDATE 12`, no-digit tags, empty ranges, and a
  digit-only subrange still return their existing values.
- Root cause: `command_complete_affected_rows` initialized `pos` from `start`
  and read `payload[pos]` in the loop condition without first rejecting
  negative or wrapped ranges.
- Fix: return `0` for negative starts, non-positive payload lengths, or
  wrapped limits before scanning.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_command_tag_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_result_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ascii_i32_bounds_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0076: PostgreSQL RowDescription type-OID scan traps on negative start

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL wire helpers.
- Found while creating:
  `examples/microservices/backend_postgres_row_description_bounds_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_row_description_bounds_guard_service.tetra
```

- Expected: `lib.core.postgres.row_description_type_oid_at(...)` should return
  `-1` for malformed RowDescription requests, including negative `start`,
  short payload lengths, wrapped ranges, and out-of-range column indexes.
- Actual before the fix: the service exited with runtime status `1`; the
  service has no `return 1` branch, so
  `row_description_type_oid_at(desc, -1, i, 0)` trapped before it could return
  the expected `-1`.
- Control cases: valid `id` and `message` column type OIDs still return
  `int4_oid()` and `text_oid()`, while negative/out-of-range column indexes and
  truncated metadata still return `-1`.
- Root cause: `row_description_type_oid_at` read the RowDescription column
  count at `start` before rejecting negative or malformed ranges.
- Fix: reject negative starts, too-short payload lengths, negative column
  indexes, and wrapped limits before reading the column count.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_row_description_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_result_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_command_tag_bounds_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0077: PostgreSQL DataRow value helpers trap on negative start

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL wire helpers.
- Found while creating:
  `examples/microservices/backend_postgres_data_row_bounds_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_bounds_guard_service.tetra
```

- Expected: `lib.core.postgres.data_row_value_len_at(...)` and
  `data_row_value_start_at(...)` should return `-1` for malformed DataRow
  requests such as negative `start`; `data_row_i32_at(...)` should return `0`
  for the same missing value.
- Actual before the fix: the service exited with runtime status `1`; the
  service has no `return 1` branch, so `data_row_value_len_at(row, -1, 0)`
  trapped before it could return the expected `-1`.
- Control cases: valid `42` and `7` DataRow columns still expose expected
  lengths, start offsets, and parsed integer values; negative and out-of-range
  column indexes still return missing-value sentinels.
- Root cause: DataRow value helpers read the column count at `start` before
  rejecting negative starts or negative column indexes.
- Fix: reject negative starts and negative column indexes before reading the
  DataRow column count in both value length and value start helpers.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_result_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_length_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_row_description_bounds_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0078: PostgreSQL frame header readers trap on negative start

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL wire helpers.
- Found while creating:
  `examples/microservices/backend_postgres_frame_header_bounds_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_header_bounds_guard_service.tetra
```

- Expected: `lib.core.postgres.frame_type_at(...)` and
  `frame_length_at(...)` should return `-1` for malformed typed-frame requests
  such as negative `start`; derived `frame_payload_len_at(...)` and
  `frame_total_len_at(...)` should also return `-1` when the underlying length
  is invalid or shorter than the PostgreSQL four-byte length field.
- Actual before the fix: the service exited with runtime status `1`; the
  service has no `return 1` branch, so `frame_type_at(frame, -1)` trapped
  before it could return the expected `-1`.
- Control cases: a valid Sync frame still reports type `frame_sync()`, length
  `4`, payload length `0`, total length `5`, and payload start `5`; a
  malformed length field of `3` still remains readable through
  `frame_length_at(...)` but derived payload and total lengths are rejected.
- Root cause: typed-frame header helpers read `frame[start]` and
  `read_i32_be(frame, start + 1)` before rejecting negative starts; derived
  length helpers also accepted sub-header length fields.
- Fix: reject negative starts before reading typed-frame headers, and make
  payload/total length helpers return `-1` for any frame length below `4`.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_header_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_result_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_command_tag_bounds_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0079: PostgreSQL ReadyForQuery status reader traps on negative start

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL wire helpers.
- Found while creating:
  `examples/microservices/backend_postgres_ready_status_bounds_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ready_status_bounds_guard_service.tetra
```

- Expected: `lib.core.postgres.ready_for_query_status(...)` should return
  `-1` for malformed ReadyForQuery status requests such as negative `start`,
  while preserving the PostgreSQL status bytes for idle, in-transaction, and
  failed-transaction states.
- Actual before the fix: the service exited with runtime status `1`; the
  service has no `return 1` branch, so `ready_for_query_status(states, -1)`
  trapped before it could return the expected `-1`.
- Control cases: valid ReadyForQuery bytes still report
  `ready_for_query_idle_status()`, `ready_for_query_in_transaction_status()`,
  and `ready_for_query_failed_transaction_status()`.
- Root cause: `ready_for_query_status` read `payload[start]` before rejecting
  negative starts.
- Fix: reject negative starts before reading the caller-owned payload.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ready_status_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_session_state_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_prepared_pipeline_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_header_bounds_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0080: PostgreSQL column-count readers trap on negative start

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL wire helpers.
- Found while creating:
  `examples/microservices/backend_postgres_column_count_bounds_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_column_count_bounds_guard_service.tetra
```

- Expected: `lib.core.postgres.row_description_column_count(...)` and
  `data_row_column_count(...)` should return `-1` for malformed backend payload
  requests such as negative `start`, while preserving valid two-byte count
  fields.
- Actual before the fix: the service exited with runtime status `1`; the
  service has no `return 1` branch, so `row_description_column_count(desc, -1)`
  trapped before it could return the expected `-1`.
- Control cases: valid RowDescription and DataRow payloads still report column
  count `2`, and the existing RowDescription/DataRow/result guard services
  still pass.
- Root cause: the column-count helpers read `read_i16_be(payload, start)`
  before rejecting negative starts.
- Fix: reject negative starts before reading the two-byte count field in both
  RowDescription and DataRow column-count helpers.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_column_count_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_row_description_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_result_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0081: PostgreSQL big-endian readers trap on negative start

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL wire helpers.
- Found while creating:
  `examples/microservices/backend_postgres_read_bounds_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_read_bounds_guard_service.tetra
```

- Expected: exported `lib.core.postgres.read_i32_be(...)`,
  `read_i32_be_signed(...)`, and `read_i16_be(...)` should return `-1` for
  malformed caller-owned byte-buffer reads such as negative `start`, while
  preserving valid big-endian reads.
- Actual before the fix: the service exited with runtime status `1`; the
  service has no `return 1` branch, so `read_i32_be(bytes, -1)` trapped before
  it could return the expected `-1`.
- Control cases: valid big-endian i32 and i16 fields still round-trip through
  `write_i32_be_at(...)`, `write_i16_be_at(...)`, `read_i32_be(...)`,
  `read_i32_be_signed(...)`, and `read_i16_be(...)`; column-count,
  frame-header, and prepared-pipeline PostgreSQL guard services still pass.
- Root cause: the big-endian readers read `src[start]` before rejecting
  negative starts; `read_i32_be_signed(...)` delegated to the unsafe unsigned
  reader before normalizing signed values.
- Fix: reject negative starts in `read_i32_be(...)` and `read_i16_be(...)`;
  `read_i32_be_signed(...)` inherits the same sentinel through `read_i32_be`.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_read_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_column_count_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_header_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_prepared_pipeline_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0082: PostgreSQL big-endian writers trap on negative start

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL wire helpers.
- Found while creating:
  `examples/microservices/backend_postgres_write_bounds_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_write_bounds_guard_service.tetra
```

- Expected: exported `lib.core.postgres.write_i32_be_at(...)` and
  `write_i16_be_at(...)` should return `-1` for malformed caller-owned
  byte-buffer writes such as negative `start`, while preserving valid
  big-endian writes and existing buffer contents.
- Actual before the fix: the service exited with runtime status `1`; the
  service has no `return 1` branch, so `write_i32_be_at(bytes, -1, 1)` trapped
  before it could return the expected `-1`.
- Control cases: valid big-endian i32 and i16 writes still round-trip through
  the reader helpers; failed negative-start writes leave the existing valid
  bytes readable.
- Root cause: the big-endian writers wrote `dst[start]` before rejecting
  negative starts.
- Fix: reject negative starts in `write_i32_be_at(...)` and
  `write_i16_be_at(...)` before writing to the caller-owned buffer.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_write_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_read_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_column_count_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_prepared_pipeline_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0083: PostgreSQL text writers trap on negative start

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL wire helpers.
- Found while creating:
  `examples/microservices/backend_postgres_text_write_bounds_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_text_write_bounds_guard_service.tetra
```

- Expected: exported `lib.core.postgres.write_ascii_at(...)`,
  `write_cstring_at(...)`, and `write_cstring_pair_at(...)` should return `-1`
  for malformed caller-owned byte-buffer writes such as negative `start`,
  while preserving valid ASCII and C-string writes.
- Actual before the fix: the service exited with runtime status `1`; the
  service has no `return 1` branch, so `write_ascii_at(bytes, -1, "Z")`
  trapped before it could return the expected `-1`.
- Control cases: valid ASCII, C-string, and C-string pair writes still produce
  the expected bytes; failed negative-start writes leave the existing valid
  bytes readable.
- Root cause: `write_ascii_at(...)` wrote `dst[i]` with `i = start` before
  rejecting negative starts; C-string helpers delegated into it and then kept
  writing without checking for a failed sentinel.
- Fix: reject negative starts before text writes, and propagate `-1` from
  nested C-string writer calls before touching the caller-owned buffer again.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_text_write_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_write_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_prepared_pipeline_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_result_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0084: HTTP writer helpers trap on negative start

- Status: fixed, verified.
- Area: stdlib / backend HTTP wire helpers.
- Found while creating:
  `examples/microservices/backend_http_writer_bounds_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_writer_bounds_guard_service.tetra
```

- Expected: exported `lib.core.http.write_ascii_at(...)`,
  `write_crlf_at(...)`, `write_header_at(...)`, and
  `write_decimal_i32_at(...)` should return `-1` for malformed caller-owned
  byte-buffer writes such as negative `start`, while preserving valid writes.
- Actual before the fix: the service exited with runtime status `1`; the
  service has no `return 1` branch, so `write_ascii_at(out, -1, "Z")` trapped
  before it could return the expected `-1`.
- Control cases: valid ASCII, CRLF, header, and decimal writes still produce
  the expected bytes; failed negative-start writes leave earlier valid bytes
  readable.
- Root cause: HTTP writer helpers wrote `dst[start]` or `dst[i]` before
  rejecting negative starts; `write_header_at(...)` also kept writing after a
  nested writer could have returned a failed sentinel.
- Fix: reject negative starts before low-level HTTP writes and propagate `-1`
  from nested header writer calls before touching the caller-owned buffer.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_writer_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_response_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_pipeline_gateway_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0085: JSON writers trap on negative start

- Status: fixed, verified.
- Area: stdlib / backend JSON wire helpers.
- Found while creating:
  `examples/microservices/backend_json_writer_bounds_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_writer_bounds_guard_service.tetra
```

- Expected: exported `lib.core.json.write_json_string_at(...)` and
  `write_message_object_at(...)` should return `-1` for malformed caller-owned
  byte-buffer writes such as negative `start`, while preserving valid escaped
  string and message-object writes.
- Actual before the fix: the corrected service exited with runtime status `1`;
  the service has no `return 1` branch, so `write_json_string_at(buf, -1, "Z")`
  trapped before it could return the expected `-1`.
- Control cases: valid escaped string `A\n\"` still serializes to `"A\n\""`;
  `write_message_object_at(buf, 8, "OK")` still writes `{"message":"OK"}`
  ending at index `24`; failed negative-start writes leave earlier valid bytes
  readable.
- Root cause: JSON writers wrote `dst[start]` before rejecting negative starts;
  `write_message_object_at(...)` also kept writing after the nested string
  writer could have returned a failed sentinel.
- Fix: reject negative starts before JSON writer output and propagate `-1` from
  nested string writer calls before writing the closing object byte.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_writer_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_control_matrix_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_escape_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_hex_digit_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0086: HTTP/JSON i32 digit helpers collapse minimum value

- Status: fixed, verified.
- Area: stdlib / backend HTTP and JSON decimal helpers.
- Found while creating:
  `examples/microservices/backend_http_json_i32_min_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_json_i32_min_guard_service.tetra
```

- Expected: `http.digits_i32(-2147483648)` and
  `json.digits_i32(-2147483648)` should return `11`; JSON world-object length
  sizing and HTTP decimal byte writes should preserve the full
  `-2147483648` text.
- Actual before the fix: the service exited `21` because
  `http.digits_i32(min_i32)` did not return `11`; a diagnostic variant that
  returned the helper result exited `1`.
- Control cases: the same service verifies `json.world_object_len(...)`,
  byte-for-byte HTTP decimal output for `-2147483648`, negative-start writer
  rejection, and preserved buffer contents after the rejected write.
- Root cause: HTTP and JSON `digits_i32(...)` converted negatives with
  `0 - n`; the i32 minimum value cannot be represented as a positive i32, so
  the magnitude stayed negative and the digit loop reported only the sign.
  `http.write_decimal_i32_at(...)` had the same unsafe magnitude conversion.
- Fix: special-case `-2147483648` in both digit helpers and write its decimal
  literal through the existing HTTP ASCII writer path.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_json_i32_min_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_writer_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_control_matrix_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_writer_bounds_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0087: Crypto mix_seed overflows normalizing i32 minimum

- Status: fixed, verified.
- Area: stdlib / backend crypto interface helpers.
- Found while creating:
  `examples/microservices/backend_crypto_mix_min_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_crypto_mix_min_guard_service.tetra
```

- Expected: `crypto.mix_seed(...)` should keep the negative-normalization branch
  non-negative even when `seed * 33 + value` is exactly `-2147483648`.
- Actual before the fix: the service exited `21` because
  `crypto.mix_seed(-65075262, -2)` returned a negative value.
- Control cases: existing negative and positive seed mixes still return `65` and
  `1407`, and the experimental crypto mirror keeps passing through the stable
  helper.
- Root cause: `mix_seed(...)` converted negative `mixed` values with
  `0 - mixed`; the i32 minimum value cannot be represented as a positive i32,
  so the normalization overflowed back to a negative result.
- Fix: special-case `mixed == -2147483648` and return `2147483647`, preserving
  a deterministic non-negative saturating value.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_crypto_mix_min_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_crypto_serialization_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_experimental_buffer_mirror_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0088: Networking retry backoff overflows before cap

- Status: fixed, verified.
- Area: stdlib / backend networking policy helpers.
- Found while creating:
  `examples/microservices/backend_network_backoff_overflow_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_network_backoff_overflow_guard_service.tetra
```

- Expected: `retry_backoff_ms(...)` should honor a non-negative `max_ms` cap
  before doubling can overflow, and should not return a negative backoff for
  positive inputs.
- Actual before the fix: the service exited `21`; the first capped case
  `retry_backoff_ms(1, 1073741824, 2147483647)` returned a negative value.
- Control cases: capped multi-step backoff, ordinary capped retry,
  negative-`max_ms` uncapped behavior below overflow, and negative base clamping
  remain stable; the experimental networking mirror still delegates correctly.
- Root cause: `retry_backoff_ms(...)` doubled `value` before checking `max_ms`,
  so `1073741824 * 2` overflowed to a negative i32 and escaped the later
  `value > max_ms` cap check.
- Fix: compute an effective cap (`max_ms` or `2147483647` for uncapped calls)
  before the loop and return the cap when the next doubling would exceed it.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_network_backoff_overflow_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_network_policy_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_experimental_route_policy_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0089: Epoll event extractors trap on short buffers

- Status: fixed, verified.
- Area: stdlib / backend net epoll helpers.
- Found while creating:
  `examples/microservices/backend_net_epoll_event_bounds_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_net_epoll_event_bounds_guard_service.tetra
```

- Expected: `net.epoll_event_fd(...)` should return `-1` for an empty event
  buffer, and `net.epoll_event_flags(...)` should return `-1` when the flags
  slot is missing.
- Actual before the fix: the service terminated with runtime `exit status 1`
  before reaching its explicit guard return codes because the helpers indexed
  short slices directly.
- Control cases: a two-slot event still returns the stored fd and decodes
  `EPOLLIN | EPOLLHUP`; a one-slot event still exposes the fd slot while
  reporting missing flags as `-1`.
- Root cause: `epoll_event_fd(...)` returned `event[0]` and
  `epoll_event_flags(...)` returned `event[1]` without proving that the
  caller-owned slice contained those slots.
- Fix: scan the event slice for the requested slot and return `-1` when the
  slot is absent.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_net_epoll_event_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_net_epoll_lifecycle_service.tetra
go test ./compiler -run 'TestBuildCoreNetSmoke|TestMicroserviceExamplesAndBugLedger' -count=1
```

### TETRA-BUG-0090: PostgreSQL frame header readers trap on short buffers

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL frame helpers.
- Found while creating:
  `examples/microservices/backend_postgres_frame_short_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_short_guard_service.tetra
```

- Expected: `frame_type_at(...)` should return `-1` when the tag slot is
  missing, and `frame_length_at(...)`, `frame_payload_len_at(...)`, and
  `frame_total_len_at(...)` should return `-1` when the four-byte length field
  is missing or truncated.
- Actual before the fix: the service terminated with runtime `exit status 1`
  before reaching its explicit guard return codes because the frame header
  helpers indexed short slices directly.
- Control cases: a valid Sync frame still reports type `S`, length `4`,
  payload length `0`, and total length `5`; a tag-only frame still exposes the
  present tag while rejecting the missing length field.
- Root cause: `frame_type_at(...)` returned `frame[start]`, and
  `frame_length_at(...)` delegated to `read_i32_be(frame, start + 1)` without
  proving that the caller-owned slice contained the requested header bytes.
- Fix: scan the frame slice for the requested tag/length byte positions and
  return `-1` when any required byte is absent or the computed offset wraps.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_header_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_read_bounds_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0091: PostgreSQL big-endian readers trap on short buffers

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL byte readers.
- Found while creating:
  `examples/microservices/backend_postgres_read_short_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_read_short_guard_service.tetra
```

- Expected: `read_i32_be(...)`, `read_i32_be_signed(...)`, and
  `read_i16_be(...)` should return `-1` when a caller-owned byte buffer is
  empty, truncated, or too short at the requested start offset.
- Actual before the fix: the service terminated with runtime `exit status 1`
  before reaching its explicit guard return codes because the readers indexed
  short slices directly.
- Control cases: valid i32 and i16 big-endian fields still decode to
  `16909060` and `258`; signed i32 reads still delegate through the same valid
  path and preserve the negative-value normalization from the earlier reader
  guard.
- Root cause: `read_i32_be(...)` and `read_i16_be(...)` rejected negative
  starts but did not prove that `start + n` remained in the caller-owned slice.
- Fix: scan for each required byte position, reject wrapped offsets, and return
  `-1` if any required byte is absent.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_read_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_read_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_short_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0092: PostgreSQL big-endian writers trap on short buffers

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL byte writers.
- Found while creating:
  `examples/microservices/backend_postgres_write_short_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_write_short_guard_service.tetra
```

- Expected: `write_i32_be_at(...)` and `write_i16_be_at(...)` should return
  `-1` when a caller-owned destination byte buffer is empty, truncated, or too
  short at the requested start offset, without partially changing the rejected
  buffer.
- Actual before the fix: the service terminated with runtime `exit status 1`
  before reaching its explicit guard return codes because the writers indexed
  short slices directly.
- Control cases: valid i32 and i16 big-endian writes still return the next
  byte indexes `4` and `6`, and the existing negative-start writer guard
  remains stable.
- Root cause: `write_i32_be_at(...)` and `write_i16_be_at(...)` rejected
  negative starts but did not prove that every required destination byte
  existed before writing.
- Fix: share byte-window probes with the big-endian readers, reject wrapped
  offsets or missing destination bytes before writing, and leave rejected
  buffers unchanged.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_write_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_write_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_read_short_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0093: PostgreSQL text writers trap on short buffers

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL text writers.
- Found while creating:
  `examples/microservices/backend_postgres_text_write_short_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_text_write_short_guard_service.tetra
```

- Expected: `write_ascii_at(...)`, `write_cstring_at(...)`, and
  `write_cstring_pair_at(...)` should return `-1` when a caller-owned
  destination byte buffer is empty, truncated, or too short at the requested
  start offset, without partially changing the rejected buffer.
- Actual before the fix: the service terminated with runtime `exit status 1`
  before reaching its explicit guard return codes because the text writers
  indexed short slices directly.
- Control cases: valid ASCII, C-string, and C-string-pair writes still return
  the next byte indexes `2`, `4`, and `8`; the existing negative-start text
  writer guard remains stable.
- Root cause: the text writers rejected negative starts but did not prove that
  the complete ASCII or NUL-terminated destination byte window existed before
  writing.
- Fix: add a shared byte-window probe and use it before ASCII, C-string, and
  C-string-pair writes; rejected writes return `-1` before mutating the
  destination.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_text_write_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_text_write_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_write_short_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0094: HTTP writer helpers trap on short buffers

- Status: fixed, verified.
- Area: stdlib / backend HTTP writers.
- Found while creating:
  `examples/microservices/backend_http_writer_short_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_writer_short_guard_service.tetra
```

- Expected: `write_ascii_at(...)`, `write_crlf_at(...)`,
  `write_header_at(...)`, and `write_decimal_i32_at(...)` should return `-1`
  when a caller-owned destination byte buffer is empty, truncated, or too short
  at the requested start offset, without partially changing the rejected
  buffer.
- Actual before the fix: the service terminated with runtime `exit status 1`
  before reaching its explicit guard return codes because the HTTP writers
  indexed short slices directly.
- Control cases: valid ASCII, CRLF, header, and decimal writes still return the
  next byte indexes `2`, `4`, `10`, and `13`; the existing negative-start HTTP
  writer guard remains stable, including the `-2147483648` decimal path.
- Root cause: the HTTP writers rejected negative starts but did not prove that
  the complete destination byte window existed before writing.
- Fix: add a shared HTTP byte-window probe and use it before ASCII, CRLF,
  header, and decimal writes; rejected writes return `-1` before mutating the
  destination.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_writer_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_writer_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0095: JSON writers trap on short buffers

- Status: fixed, verified.
- Area: stdlib / backend JSON writers.
- Found while creating:
  `examples/microservices/backend_json_writer_short_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_writer_short_guard_service.tetra
```

- Expected: `write_json_string_at(...)` and `write_message_object_at(...)`
  should return `-1` when a caller-owned destination byte buffer is empty,
  truncated, or too short at the requested start offset, without partially
  changing the rejected buffer.
- Actual before the fix: the service terminated with runtime `exit status 1`
  before reaching its explicit guard return codes because the JSON writers
  indexed short slices directly.
- Control cases: valid escaped JSON string and message-object writes still
  return the next byte indexes `7` and `24`; the existing negative-start JSON
  writer guard remains stable.
- Root cause: the JSON writers rejected negative starts but did not prove that
  the complete encoded string or message-object destination byte window existed
  before writing.
- Fix: add a shared JSON byte-window probe and use it before escaped-string and
  message-object writes; rejected writes return `-1` before mutating the
  destination.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_writer_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_writer_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_control_matrix_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0096: PostgreSQL frame writers trap on short buffers

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL frame writers.
- Found while creating:
  `examples/microservices/backend_postgres_frame_writer_short_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_writer_short_guard_service.tetra
```

- Expected: high-level PostgreSQL frontend writers such as
  `write_startup_message(...)`, `write_simple_query(...)`, `write_parse(...)`,
  `write_bind_text_1(...)`, `write_describe_portal(...)`,
  `write_execute(...)`, `write_sync(...)`, and `write_terminate(...)` should
  return `-1` when the caller-owned destination byte buffer is empty or too
  short, without partially changing the rejected buffer.
- Actual before the fix: the service terminated with runtime `exit status 1`
  before reaching its explicit guard return codes because some frame writers
  indexed `dst[0]` directly and `write_startup_message(...)` could continue to a
  final `dst[i]` write after lower-level writers had returned `-1`.
- Control cases: valid startup, Simple Query, Parse, Bind, Describe, Execute,
  Sync, and Terminate frames still return their exact frame lengths; existing
  PostgreSQL prepared pipeline and session-state examples remain stable.
- Root cause: top-level frame writers delegated to guarded byte writers but did
  not prove that the complete frame destination window existed before their own
  direct writes.
- Fix: preflight each high-level frame writer with the existing PostgreSQL
  byte-window probe and exact frame-length helpers before writing the first
  byte; startup-message writing also stops immediately after any nested `-1`.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_writer_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_prepared_pipeline_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_session_state_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0097: PostgreSQL bounded parsers trap on short buffers

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL bounded parsers.
- Found while creating:
  `examples/microservices/backend_postgres_parser_short_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_parser_short_guard_service.tetra
```

- Expected: `cstring_end_at(...)`, `parse_ascii_i32_at(...)`, and
  `command_complete_affected_rows(...)` should return their sentinel values
  when caller-provided limits or counts extend past the physical byte buffer:
  `-1` for missing C-string terminators and `0` for malformed integer or
  CommandComplete ranges.
- Actual before the fix: the service terminated with runtime `exit status 1`
  before reaching its explicit guard return codes because the parsers indexed
  `src[pos]` or `payload[pos]` while trusting the oversized caller range.
- Control cases: valid bounded C-string scans, signed/unsigned ASCII integer
  parses, and CommandComplete affected-row extraction still return `2`, `4`,
  `-42`, `42`, and `12` as expected; existing PostgreSQL parser guard examples
  remain stable.
- Root cause: the parsers validated negative, reversed, empty, and wrapped ranges
  but did not prove that the requested byte window existed before direct
  indexing.
- Fix: preflight the requested byte window with the existing PostgreSQL
  `has_u8_window` helper before scanning or parsing.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_parser_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_cstring_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ascii_i32_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_command_tag_bounds_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0098: PostgreSQL ReadyForQuery status reader traps on short buffers

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL ReadyForQuery parser.
- Found while creating:
  `examples/microservices/backend_postgres_ready_status_short_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ready_status_short_guard_service.tetra
```

- Expected: `ready_for_query_status(...)` should return `-1` for empty payloads
  or starts that point past the single status byte, while preserving valid idle,
  in-transaction, and failed-transaction status reads.
- Actual before the fix: the service terminated with runtime `exit status 1`
  before reaching its explicit guard return codes because
  `ready_for_query_status(...)` indexed `payload[start]` after only rejecting
  negative starts.
- Control cases: valid `I`, `T`, and `E` status bytes still return
  `ready_for_query_idle_status()`, `ready_for_query_in_transaction_status()`,
  and `ready_for_query_failed_transaction_status()`; the existing negative-start
  ReadyForQuery guard remains stable.
- Root cause: the status reader did not prove that the requested one-byte window
  existed before direct indexing.
- Fix: preflight `ready_for_query_status(...)` with the existing PostgreSQL
  `has_u8_window` helper for a one-byte payload window.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ready_status_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ready_status_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_session_state_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_prepared_pipeline_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0099: HTTP request scanners trap on short buffers

- Status: fixed, verified.
- Area: stdlib / backend HTTP request scanners.
- Found while creating:
  `examples/microservices/backend_http_request_short_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_short_guard_service.tetra
```

- Expected: byte-buffer HTTP request scanners should return their existing
  sentinel values when caller-provided `request_len` ranges are empty,
  offset-short, or extend past the physical request buffer: `0` for missing
  request heads, `false` for keep-alive/version/header probes, and
  `route_bad_request()` for route classification.
- Actual before the fix: the service terminated with runtime `exit status 1`
  before reaching its explicit guard return codes because the scanners indexed
  `request[start + pos]` while trusting the oversized caller range.
- Control cases: valid pipelined `/json` and `/plaintext` HTTP/1.1 requests
  still produce exact head lengths, route IDs, keep-alive decisions, and
  `Connection: close` detection.
- Root cause: `route_tech_empower_bytes_at(...)`,
  `request_keep_alive_bytes_at(...)`, `request_head_len_bytes_at(...)`,
  `contains_http11_marker_bytes_at(...)`, and
  `contains_connection_close_bytes_at(...)` validated request syntax while
  assuming that the complete caller-requested byte window existed.
- Fix: preflight each byte-buffer request scanner with the existing HTTP
  `has_u8_window` helper before any direct indexing.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_body_scope_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_line_token_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0100: HTTP response writers trap or partially write on short buffers

- Status: fixed, verified.
- Area: stdlib / backend HTTP response writers.
- Found while creating:
  `examples/microservices/backend_http_response_writer_short_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_response_writer_short_guard_service.tetra
```

- Expected: `write_response_head(...)`, `write_plaintext_response(...)`, and
  `write_json_message_response(...)` should return `-1` when a caller-owned
  destination byte buffer is empty or too short for the complete output, without
  partially changing the rejected buffer.
- Actual before the fix: the service terminated with runtime `exit status 1`
  before reaching its explicit guard return codes. Later truncated response-body
  paths could also write a complete head before returning `-1` for the missing
  body bytes.
- Control cases: valid response heads, plaintext responses, and JSON message
  responses still return their exact byte counts and preserve the expected HTTP
  prefix, CRLF terminator, plaintext body, and JSON object terminator.
- Root cause: the high-level response writers delegated to guarded lower-level
  byte writers but did not prove that the complete response destination window
  existed before starting to write. `write_response_head(...)` also used the
  lower writer result as a direct `dst[i]` index even when that result was `-1`.
- Fix: preflight the exact complete destination window for response heads,
  plaintext responses, and JSON message responses with the existing HTTP
  `has_u8_window` helper before writing the first byte.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_response_writer_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_response_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_writer_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0101: PostgreSQL big-endian writers encode negative values incorrectly

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL big-endian writers.
- Found while creating:
  `examples/microservices/backend_postgres_signed_write_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_signed_write_guard_service.tetra
```

- Expected: `write_i32_be_at(...)` and `write_i16_be_at(...)` should encode
  negative `Int` values as signed two's-complement big-endian bytes, preserving
  ordinary positive big-endian writes.
- Actual before the fix: the service exited `12` because
  `write_i32_be_at(bytes, 0, -1)` returned the next index but did not write
  `255,255,255,255`. The same arithmetic affected other negative i32/i16
  values such as `-2`, `-32768`, and the i32 minimum.
- Control cases: positive `16909060` and `258` still serialize as
  `01 02 03 04` and `01 02`; short-buffer and frame-writer guard services
  remain stable.
- Root cause: the writers computed bytes with division and modulo directly on
  negative `Int` values, so high bytes collapsed toward zero or the final modulo
  stayed negative instead of producing two's-complement byte values.
- Fix: add negative-value paths that complement the positive magnitude bytes,
  with explicit handling for `-2147483648`; the i16 writer emits the low
  two's-complement bytes.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_signed_write_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_write_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_read_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_writer_short_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0102: PostgreSQL frame length reader leaks malformed signed lengths

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL frame helpers.
- Found while creating:
  `examples/microservices/backend_postgres_frame_signed_length_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_signed_length_guard_service.tetra
```

- Expected: `frame_length_at(...)` should normalize malformed negative signed
  frame length fields to the public sentinel `-1`, and derived payload/total
  length helpers should continue returning `-1` for those frames.
- Actual before the fix: the service exited `11` because a frame whose length
  bytes encode `-2` was returned as a raw negative length instead of `-1`.
  The i32 minimum length encoding followed the same manual arithmetic path.
- Control cases: valid Sync frame length `4`, payload length `0`, total length
  `5`, and malformed positive short length `3` behavior remain unchanged.
- Root cause: `frame_length_at(...)` reconstructed the i32 length field by
  multiplying bytes directly, duplicating the unsigned/high-bit arithmetic path
  instead of using the signed PostgreSQL length reader that maps negative
  values to `-1`.
- Fix: delegate `frame_length_at(...)` to `read_i32_be_signed(...)` after the
  existing start/offset guard.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_signed_length_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_header_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_signed_write_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0103: PostgreSQL big-endian reader leaks high-bit i32 values

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL big-endian readers.
- Found while creating:
  `examples/microservices/backend_postgres_high_bit_read_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_high_bit_read_guard_service.tetra
```

- Expected: `read_i32_be(...)` should preserve max positive `0x7fffffff`
  as `2147483647` and return `-1` for high-bit/unrepresentable i32 values
  such as `0x80000000` and `0xfffffffe`; `read_i32_be_signed(...)` should
  keep normalizing those to `-1`.
- Actual before fix: the service exited `11` because `read_i32_be(...)`
  leaked the high-bit value instead of returning the public sentinel.
- Control cases: max positive control plus signed writer, frame signed-length,
  and read short controls remain stable.
- Root cause: `read_i32_be(...)` multiplied high-bit `b0` before checking
  whether the result was representable as a non-negative `Int`, so i32
  overflow produced arbitrary negative values.
- Fix: after proving four bytes exist, return `-1` whenever `b0 >= 128`
  before arithmetic.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_high_bit_read_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_read_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_signed_length_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_signed_write_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0104: Time duration addition clamps negative base before applying positive delta

- Status: fixed, verified.
- Area: stdlib / backend time helpers.
- Found while creating:
  `examples/microservices/backend_time_negative_base_delta_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_time_negative_base_delta_guard_service.tetra
```

- Expected: `add_duration_ms(base, delta)` should add `delta` to `base` first,
  then clamp a negative summed result to `0` or saturate positive overflow to
  `2147483647`. A negative base such as `-5` should therefore recover to `5`
  when the positive delta is `10`.
- Actual before fix: the service exited `11` because
  `add_duration_ms(-5, 10)` returned `0` instead of the summed duration `5`.
- Control cases: large negative-base recovery, still-negative sums, zero-delta
  negative bases, negative-delta subtraction, and positive overflow saturation
  remain stable.
- Root cause: `add_duration_ms(...)` clamped every negative `base` to `0`
  before checking whether a positive `delta` would make the final sum
  non-negative.
- Fix: handle the positive-delta branch before the negative-base clamp, keep
  the existing positive overflow pre-check, and clamp the actual positive-delta
  sum only if it remains negative.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_time_negative_base_delta_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_time_overflow_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_time_collection_window_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0105: HTTP request-line scanner accepts LF-only or bare-CR HTTP/1.1 terminators

- Status: fixed, verified.
- Area: stdlib / backend HTTP request-line scanners.
- Found while creating:
  `examples/microservices/backend_http_request_crlf_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_crlf_guard_service.tetra
```

- Expected: route and keep-alive helpers should accept the `HTTP/1.1`
  request-line version only when it is terminated by CRLF. LF-only
  `HTTP/1.1\n` and bare-CR `HTTP/1.1\rHost` inputs should be malformed,
  matching the documented `\r\n\r\n` request-head splitter boundary.
- Actual before fix: the service exited `3` because
  `route_tech_empower("GET /json HTTP/1.1\n...")` returned the JSON route
  instead of `route_bad_request()`. Bare-CR request lines followed the same
  early-success marker path.
- Control cases: valid CRLF request-line routing, keep-alive detection, and
  request-head length detection remain stable for string and byte-buffer
  helpers.
- Root cause: `contains_http11_marker(...)` and
  `contains_http11_marker_bytes_at(...)` returned success immediately after
  seeing either LF or CR after `HTTP/1.1`; they did not require CR to be
  followed by LF.
- Fix: add a post-CR state to both HTTP/1.1 marker scanners and return success
  only when the next byte is LF.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_crlf_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_line_token_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_version_scope_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_short_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0106: filesystem.exists accepts embedded-NUL paths and checks only the prefix

- Status: fixed, verified.
- Area: stdlib / backend filesystem runtime ABI.
- Found while creating:
  `examples/microservices/backend_filesystem_nul_exists_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_filesystem_nul_exists_guard_service.tetra
```

- Expected: `filesystem.exists(path, cap)` should return `false` when the
  Tetra `String` contains an embedded NUL byte. The host should not see only
  the prefix before that NUL.
- Actual before fix: the service exited `2` because
  `filesystem.exists("/<NUL>_suffix", cap)` returned true by checking
  the existing `/` prefix.
- Control cases: normal existing-path lookup for `/`, missing-path
  lookup, filesystem path-policy helper coverage, and the experimental
  filesystem mirror remain stable.
- Root cause: `emitFilesystemExists(...)` copied all `path_len` bytes into the
  stack path buffer and appended a trailing NUL for Linux `access(2)`, but did
  not reject NUL bytes already present inside the caller-provided string.
- Fix: reject the runtime call during the copy loop when any copied byte is
  zero, returning false before calling `access(2)`.
- Verification:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_filesystem_nul_exists_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_filesystem_path_policy_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_experimental_route_policy_service.tetra
go test ./compiler -run 'TestFilesystemRuntimeExistsBuildAndRunLinuxX64|TestMicroserviceExamplesAndBugLedger' -count=1
```

### TETRA-BUG-0107: HTTP request-target scanners accept control bytes as route misses or keep-alive requests

- Status: fixed, verified.
- Area: stdlib / backend HTTP request-line scanners.
- Found while creating:
  `examples/microservices/backend_http_request_target_char_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_target_char_guard_service.tetra
```

- Expected: route and keep-alive helpers should reject request targets
  containing tab or other control bytes, such as `GET /json\t HTTP/1.1`,
  before route classification or keep-alive policy.
- Actual before fix: the service exited `1` because
  `route_tech_empower("GET /json\t HTTP/1.1...")` returned a syntactic route
  miss instead of `route_bad_request()`. Keep-alive marker scanning also
  accepted tab/control bytes in the request target.
- Control cases: valid `/json?debug=1`, `/queries?queries=7`, and
  byte-buffer `/plaintext?ok=1` request targets still route and keep alive as
  before.
- Root cause: `contains_http11_marker(...)`,
  `contains_http11_marker_bytes_at(...)`, `route_tech_empower(...)`, and
  `route_tech_empower_bytes_at(...)` only required the target to start with
  `/`; after that, non-space/non-CRLF target bytes were accepted by the
  marker scanner and treated as ordinary path bytes by the route scanner.
- Fix: add a shared visible-ASCII request-target character guard and apply it
  in both string and byte-buffer marker and route scanners.
- Verification:

```sh
go run ./cli/cmd/tetra check examples/microservices/backend_http_request_target_char_guard_service.tetra
go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-http-request-target-char-guard examples/microservices/backend_http_request_target_char_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_target_char_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_target_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_line_token_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_keep_alive_method_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_crlf_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1
```

### TETRA-BUG-0108: PostgreSQL ASCII i32 parser wraps out-of-range values

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL ASCII integer parser.
- Found while creating:
  `examples/microservices/backend_postgres_ascii_i32_overflow_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ascii_i32_overflow_guard_service.tetra
```

- Expected: `lib.core.postgres.parse_ascii_i32_at(...)` and DataRow integer
  parsing should return `0` for out-of-range i32 text such as `2147483648` or
  `-2147483649`, while preserving `2147483647` and `-2147483648`.
- Actual before fix: the service exited `14` because
  `parse_ascii_i32_at("2147483648")` returned a wrapped negative value instead
  of `0`.
- Control cases: positive max, negative min, trailing non-digit sentinel
  behavior, and DataRow parsing of a following valid column remain stable.
- Root cause: `parse_ascii_i32_at(...)` multiplied the accumulated value by
  ten and added the next digit before checking whether the i32 boundary would
  be crossed.
- Fix: preflight each digit against the signed i32 positive and negative
  magnitude limits, preserving the special `-2147483648` boundary while
  returning `0` for overflow/underflow.
- Verification:

```sh
go run ./cli/cmd/tetra check examples/microservices/backend_postgres_ascii_i32_overflow_guard_service.tetra
go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-postgres-ascii-i32-overflow-guard examples/microservices/backend_postgres_ascii_i32_overflow_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ascii_i32_overflow_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ascii_i32_min_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ascii_i32_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_parser_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_length_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```

### TETRA-BUG-0109: PostgreSQL CommandComplete affected-row parser wraps out-of-range values

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL CommandComplete parser.
- Found while creating:
  `examples/microservices/backend_postgres_command_tag_overflow_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_command_tag_overflow_guard_service.tetra
```

- Expected: `lib.core.postgres.command_complete_affected_rows(...)` should
  return `0` for out-of-range affected-row counts such as
  `UPDATE 2147483648` or `INSERT 0 2147483648`, while preserving
  `2147483647`.
- Actual before fix: the service exited `13` because
  `command_complete_affected_rows("UPDATE 2147483648")` returned a wrapped
  negative value instead of `0`.
- Control cases: `UPDATE 2147483647`, `INSERT 0 2147483647`, short valid
  subranges, existing CommandComplete bounds controls, and bounded parser
  short-buffer controls remain stable.
- Root cause: `command_complete_affected_rows(...)` accumulated each digit run
  with `current = current * 10 + digit` before checking the i32 maximum.
- Fix: preflight each digit run against the i32 maximum, ignore overflowed
  non-trailing digit runs, and return `0` when the trailing affected-row count
  overflows.
- Verification:

```sh
go run ./cli/cmd/tetra check examples/microservices/backend_postgres_command_tag_overflow_guard_service.tetra
go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-postgres-command-tag-overflow-guard examples/microservices/backend_postgres_command_tag_overflow_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_command_tag_overflow_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_command_tag_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_parser_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ascii_i32_overflow_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/pgrt -count=1
```

### TETRA-BUG-0110: PostgreSQL CommandComplete parser returns non-trailing digit runs

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL CommandComplete parser.
- Found while creating:
  `examples/microservices/backend_postgres_command_tag_trailing_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_command_tag_trailing_guard_service.tetra
```

- Expected: `lib.core.postgres.command_complete_affected_rows(...)` should
  return an affected-row count only when the digit run actually trails the
  bounded CommandComplete tag payload. Malformed tags such as `UPDATE 12 rows`
  or `UPDATE 12 ` should return `0`.
- Actual before fix: the service exited `20` because
  `command_complete_affected_rows("UPDATE 12 rows")` returned `12` even
  though the digit run was followed by non-count text.
- Control cases: valid `UPDATE 12`, `INSERT 0 3`, digit-only subranges,
  out-of-range CommandComplete counts, and existing bounds/short-buffer
  controls remain stable.
- Root cause: `command_complete_affected_rows(...)` saved each completed digit
  run in a `last` accumulator and returned it when the payload ended without an
  active trailing digit run.
- Fix: remove the non-trailing `last` fallback and return a count only when
  the bounded payload ends while reading a valid, non-overflowed digit run.
- Verification:

```sh
go run ./cli/cmd/tetra check examples/microservices/backend_postgres_command_tag_trailing_guard_service.tetra
go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-postgres-command-tag-trailing-guard examples/microservices/backend_postgres_command_tag_trailing_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_command_tag_trailing_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_command_tag_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_command_tag_overflow_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_parser_short_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/pgrt -count=1
```

### TETRA-BUG-0111: PostgreSQL DataRow helpers accept truncated positive value windows

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL DataRow parser.
- Found while creating:
  `examples/microservices/backend_postgres_data_row_truncated_value_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_truncated_value_guard_service.tetra
```

- Expected: `lib.core.postgres.data_row_value_len_at(...)` and
  `data_row_value_start_at(...)` should return `-1` when a DataRow column
  advertises a positive value length but the caller-owned buffer does not
  physically contain that many value bytes; `data_row_i32_at(...)` should
  return `0` for the same malformed value.
- Actual before fix: the service exited `22` because a row with advertised
  length `4` and only two physical value bytes made
  `data_row_value_len_at(row, 0, 0)` return length `4` instead of `-1`.
- Control cases: valid `"42"` DataRow values, empty values, negative/malformed
  length sentinel handling, and existing DataRow bounds controls remain
  stable.
- Root cause: DataRow value length/start helpers trusted the advertised
  positive value length after reading the 4-byte length field and did not
  preflight the corresponding physical value byte window.
- Fix: check positive DataRow value windows with the existing `has_u8_window`
  helper before returning the target length/start or skipping a previous value
  to reach a later column.
- Verification:

```sh
go run ./cli/cmd/tetra check examples/microservices/backend_postgres_data_row_truncated_value_guard_service.tetra
go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-postgres-data-row-truncated-value-guard examples/microservices/backend_postgres_data_row_truncated_value_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_truncated_value_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_length_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_result_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/pgrt -count=1
```

### TETRA-BUG-0112: PostgreSQL frame total length overflows at max signed length

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL frame header parser.
- Found while creating:
  `examples/microservices/backend_postgres_frame_total_overflow_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_total_overflow_guard_service.tetra
```

- Expected: `lib.core.postgres.frame_total_len_at(...)` should return `-1`
  when the typed frame length is `2147483647`, because adding the one-byte
  frame tag would overflow the signed `Int` total length. Length `2147483646`
  should remain the maximum valid total length.
- Actual before fix: the service exited `23` because
  `frame_total_len_at(frame, 0)` wrapped after `2147483647 + 1` instead of
  returning `-1`.
- Control cases: frame length and payload length readers still accept the
  maximum positive signed length, total length `2147483647` from length
  `2147483646` remains valid, and existing short/negative frame guards remain
  stable.
- Root cause: `frame_total_len_at(...)` only rejected lengths below the
  PostgreSQL four-byte minimum and then returned `length + 1` without checking
  the signed `Int` boundary.
- Fix: reject `length == 2147483647` before adding the typed-frame tag byte.
- Verification:

```sh
go run ./cli/cmd/tetra check examples/microservices/backend_postgres_frame_total_overflow_guard_service.tetra
go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-postgres-frame-total-overflow-guard examples/microservices/backend_postgres_frame_total_overflow_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_total_overflow_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_header_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_signed_length_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_short_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/pgrt -count=1
```

### TETRA-BUG-0113: HTTP response writer accepts negative Content-Length

- Status: fixed, verified.
- Area: stdlib / backend HTTP response writer.
- Found while creating:
  `examples/microservices/backend_http_negative_content_length_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_negative_content_length_guard_service.tetra
```

- Expected: `lib.core.http.response_head_len(...)` and
  `write_response_head(...)` should return `-1` for negative
  `Content-Length` values and leave the caller-owned destination buffer
  untouched.
- Actual before fix: the service exited `11` because
  `response_head_len(200, "OK", "Tetra", date, "text/plain", -1, true)`
  returned a normal serialized head length, allowing `write_response_head(...)`
  to emit `Content-Length: -1`.
- Control cases: zero and positive Content-Length response heads still size and
  write exactly, generic decimal writing still supports negative values for
  non-HTTP-header call sites, and existing HTTP response short-buffer controls
  remain stable.
- Root cause: response-head sizing reused the generic decimal digit helper
  without first applying the HTTP Content-Length non-negative invariant; the
  writer trusted that size and serialized the malformed negative value.
- Fix: return `-1` from `response_head_len(...)` when `content_len < 0` and
  have `write_response_head(...)` consume that sentinel before touching the
  destination buffer.
- Verification:

```sh
go run ./cli/cmd/tetra check examples/microservices/backend_http_negative_content_length_guard_service.tetra
go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-http-negative-content-length-guard examples/microservices/backend_http_negative_content_length_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_negative_content_length_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_response_writer_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_json_i32_min_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1
```

### TETRA-BUG-0114: HTTP response writer accepts non-three-digit status codes

- Status: fixed, verified.
- Area: stdlib / backend HTTP response writer.
- Found while creating:
  `examples/microservices/backend_http_status_code_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_code_guard_service.tetra
```

- Expected: `lib.core.http.response_head_len(...)` and
  `write_response_head(...)` should return `-1` for response status codes
  outside the three-digit HTTP status-code range (`100..999`) and leave the
  caller-owned destination buffer untouched.
- Actual before fix: the service exited `11` because
  `response_head_len(99, "Bad", "Tetra", date, "text/plain", 0, true)`
  returned a normal serialized head length, allowing malformed response lines
  such as `HTTP/1.1 99 Bad` or `HTTP/1.1 1000 Bad`.
- Control cases: boundary status codes `100` and `999`, ordinary `200`
  response heads, negative Content-Length rejection, generic decimal writing,
  and existing HTTP response matrix controls remain stable.
- Root cause: response-head sizing reused the generic decimal digit helper for
  the status field without first enforcing the HTTP status-code width
  invariant; the writer trusted that size and serialized malformed status text.
- Fix: return `-1` from `response_head_len(...)` when `status < 100` or
  `status > 999`; `write_response_head(...)` already consumes that sentinel
  before touching the destination buffer.
- Verification:

```sh
go run ./cli/cmd/tetra check examples/microservices/backend_http_status_code_guard_service.tetra
go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-http-status-code-guard examples/microservices/backend_http_status_code_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_code_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_negative_content_length_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_response_writer_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1
```

### TETRA-BUG-0115: HTTP header writers accept CR/LF header injection

- Status: fixed, verified.
- Area: stdlib / backend HTTP response writer.
- Found while creating:
  `examples/microservices/backend_http_header_injection_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_header_injection_guard_service.tetra
```

- Expected: `lib.core.http.header_line_len(...)`, `write_header_at(...)`,
  `response_head_len(...)`, and `write_response_head(...)` should return `-1`
  for CR/LF header injection attempts in header names, header values, response
  reason text, and response header values, leaving caller-owned destination
  buffers untouched.
- Actual before fix: the service exited `11` because
  `header_line_len("X\rBad", "ok")` returned a normal serialized header length,
  and calls such as `write_header_at(header, 0, "X\rBad", "ok")` could
  serialize malformed names. Values such as `"ok\r\nX-Injected: yes"` could
  become additional response header lines.
- Control cases: ordinary `X-Test: ok` header writing, valid `200 OK` response
  heads, status-code bounds, negative Content-Length rejection, and existing
  HTTP writer short-buffer controls remain stable.
- Root cause: header and response-head sizing reused raw ASCII lengths for
  header names, header values, and reason text without enforcing HTTP token
  syntax for field names or rejecting embedded CR/LF in line-bounded fields.
- Fix: validate header names with the existing HTTP token character policy,
  reject CR/LF inside header values and response reason/header fields, and
  preserve `-1` sentinels through full plaintext/JSON response length helpers.
- Verification:

```sh
go run ./cli/cmd/tetra check examples/microservices/backend_http_header_injection_guard_service.tetra
go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-http-header-injection-guard examples/microservices/backend_http_header_injection_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_header_injection_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_code_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_negative_content_length_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_response_writer_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1
```

### TETRA-BUG-0116: HTTP header writers accept non-HTAB control bytes

- Status: fixed, verified.
- Area: stdlib / backend HTTP response writer.
- Found while creating:
  `examples/microservices/backend_http_header_control_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_header_control_guard_service.tetra
```

- Expected: `lib.core.http.header_line_len(...)`, `write_header_at(...)`,
  `response_head_len(...)`, `write_response_head(...)`, and full
  plaintext/JSON response helpers should return `-1` for header values and
  response reason/header fields containing non-HTAB control bytes or DEL,
  leaving caller-owned destination buffers untouched. HTAB remains valid inside
  header values.
- Actual before fix: the service exited `11` because a header value containing
  ASCII 0x1f between `ok` and `bad` returned a normal serialized header length
  instead of `-1`.
- Control cases: ordinary header writing, HTAB inside a header value, CR/LF
  injection rejection, status-code bounds, negative Content-Length rejection,
  and HTTP response short-buffer controls remain stable.
- Root cause: `http_header_value_valid` only rejected CR/LF after the CR/LF
  injection fix, leaving other C0 control bytes and DEL accepted by response
  header serializers.
- Fix: reject every header value byte below 32 except HTAB (`9`) and reject
  DEL (`127`) before any length or writer helper serializes the response.
- Verification:

```sh
go run ./cli/cmd/tetra check examples/microservices/backend_http_header_control_guard_service.tetra
go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-http-header-control-guard examples/microservices/backend_http_header_control_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_header_control_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_header_injection_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_code_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_negative_content_length_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_response_writer_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1
```

### TETRA-BUG-0117: PostgreSQL Parse writer wraps parameter counts above signed i16 range

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL wire-frame writers.
- Found while creating:
  `examples/microservices/backend_postgres_parse_count_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_parse_count_guard_service.tetra
```

- Expected: `lib.core.postgres.parse_payload_len(...)`,
  `parse_frame_len(...)`, and `write_parse(...)` should return `-1` when the
  parameter type OID count exceeds the signed i16 protocol range, leaving
  caller-owned destination buffers untouched.
- Actual before fix: the service exited `11` because
  `parse_payload_len("", "SELECT 1", many)` returned a normal positive length
  for `32768` OIDs. `write_parse(...)` could then encode the count through the
  low two bytes as `0x8000` instead of rejecting the malformed Parse frame.
- Control cases: small valid Parse frames still serialize with the correct
  count and OID bytes; PostgreSQL prepared pipeline, parser short-buffer, frame
  writer short-buffer, and signed writer controls remain stable.
- Root cause: `parse_payload_len(...)` accumulated one four-byte OID slot per
  slice element but never checked whether the element count fit the signed i16
  PostgreSQL count field that `write_parse(...)` later serializes.
- Fix: make Parse payload/frame length helpers return `-1` once the OID count
  would exceed `32767`; `write_parse(...)` already preflights the frame length,
  so it now returns `-1` before writing.
- Verification:

```sh
go run ./cli/cmd/tetra check examples/microservices/backend_postgres_parse_count_guard_service.tetra
go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-postgres-parse-count-guard examples/microservices/backend_postgres_parse_count_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_parse_count_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_prepared_pipeline_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_parser_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_writer_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_signed_write_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/pgrt -count=1
```

### TETRA-BUG-0118: TCP loopback bind accepts out-of-range ports

- Status: fixed, verified.
- Area: runtime / backend net TCP helpers.
- Found while creating:
  `examples/microservices/backend_net_port_bounds_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_net_port_bounds_guard_service.tetra
```

- Expected: `lib.core.net.bind_tcp4_loopback(...)` should return a negative
  result for ports outside `0..65535`, while preserving `0` as the
  kernel-selected ephemeral-port sentinel.
- Actual before fix: the service exited `13` because
  `bind_tcp4_loopback(fd, 65536, io_cap)` succeeded by serializing only the low
  16 bits of the port, effectively binding port `0` and letting the kernel pick
  an unintended ephemeral port. Negative values with low 16 bits equal to `0`
  had the same boundary leak.
- Control cases: valid ephemeral bind, epoll lifecycle helpers, epoll event
  extractors, and core net smoke remain stable.
- Root cause: the linux-x64 TCP bind/connect emitter moved the caller port into
  the sockaddr `sin_port` field after a byte swap without validating that the
  signed Tetra `Int` fit the TCP port range.
- Fix: reject negative ports and ports above `65535` in the bind/connect
  runtime emitters before constructing the sockaddr.
- Verification:

```sh
go run ./cli/cmd/tetra check examples/microservices/backend_net_port_bounds_guard_service.tetra
go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-net-port-bounds-guard examples/microservices/backend_net_port_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_net_port_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_net_epoll_lifecycle_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_net_epoll_event_bounds_guard_service.tetra
go test ./compiler -run 'TestBuildCoreNetSmoke|TestMicroserviceExamplesAndBugLedger|TestNetRuntimeSocketLifecycleBuildAndRunLinuxX64|TestNetRuntimeEpollWaitOneIntoBuildAndRunLinuxX64' -count=1
go test ./compiler/internal/actorsrt -count=1
```

### TETRA-BUG-0119: PostgreSQL column-count readers accept high-bit signed counts

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL payload readers.
- Found while creating:
  `examples/microservices/backend_postgres_column_count_signed_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_column_count_signed_guard_service.tetra
```

- Expected: `lib.core.postgres.row_description_column_count(...)` and
  `data_row_column_count(...)` should return `-1` for malformed backend count
  fields whose signed i16 high bit is set, and dependent RowDescription/DataRow
  helpers should propagate sentinel values instead of treating the count as a
  huge positive column count.
- Actual before fix: the service exited `20` because
  `row_description_column_count(malformed, 0)` returned `32768` for bytes
  `0x80 0x00`. `data_row_column_count(...)` shared the same reader path.
- Control cases: valid one-column RowDescription and DataRow payloads still
  decode correctly; negative-start, RowDescription bounds, and DataRow bounds
  guard examples remain stable.
- Root cause: the count helpers reused the generic `read_i16_be(...)` helper,
  which intentionally exposes raw two-byte values up to `65535`; PostgreSQL
  RowDescription/DataRow column-count fields are signed i16 protocol fields.
- Fix: add a count-specific `read_i16_count_be(...)` guard that maps missing,
  negative-start, and `>32767` values to `-1`, then route only the backend count
  helpers through it.
- Verification:

```sh
go run ./cli/cmd/tetra check examples/microservices/backend_postgres_column_count_signed_guard_service.tetra
go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-postgres-column-count-signed-guard examples/microservices/backend_postgres_column_count_signed_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_column_count_signed_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_column_count_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_row_description_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_bounds_guard_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/pgrt -count=1
```

### TETRA-BUG-0120: PostgreSQL C-string writers accept embedded NUL fields

- Status: fixed, verified.
- Area: stdlib / backend PostgreSQL wire-frame writers.
- Found while creating:
  `examples/microservices/backend_postgres_cstring_nul_guard_service.tetra`.
- Reproduction command:

```sh
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_cstring_nul_guard_service.tetra
```

- Expected: PostgreSQL C-string length and writer helpers should return `-1`
  for embedded NUL bytes in startup, query, statement, or portal fields before
  writing into caller-owned frame buffers. Low-level `write_cstring_at(...)` and
  `write_cstring_pair_at(...)` should also reject embedded NUL values without
  modifying existing bytes.
- Actual before fix: the service exited `12` because
  `write_cstring_at(bytes, 0, bad)` accepted a `bad<NUL>field` string and wrote
  bytes that PostgreSQL would parse as the shorter `bad` C string plus trailing
  payload data. Higher-level frame length helpers also returned positive lengths
  for embedded-NUL C-string fields.
- Control cases: valid C-string writing, startup/session-state frames, Parse
  parameter-count rejection, and frame writer short-buffer guards remain stable.
- Root cause: `cstring_len(...)` counted every byte in the Tetra `String` and
  appended one final terminator, while `write_cstring_at(...)` delegated to the
  raw ASCII writer without checking whether the value already contained `0`.
  Frame length helpers composed those lengths directly, so writers could begin
  serializing a frame before discovering a malformed C-string field.
- Fix: make `cstring_len(...)` and `cstring_pair_len(...)` return `-1` for
  embedded NUL bytes, propagate that sentinel through startup/query/Parse/Bind/
  Describe/Execute length helpers, and let high-level writers reject malformed
  C-string inputs before touching the destination buffer.
- Verification:

```sh
go run ./cli/cmd/tetra check examples/microservices/backend_postgres_cstring_nul_guard_service.tetra
go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-postgres-cstring-nul-guard examples/microservices/backend_postgres_cstring_nul_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_cstring_nul_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_text_write_bounds_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_writer_short_guard_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_session_state_service.tetra
go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_prepared_pipeline_service.tetra
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
go test ./compiler/internal/pgrt -count=1
```

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
| 2026-05-22 | Added backend HTTP/JSON pipeline and PostgreSQL prepared wire-frame microservice examples; verified adjacent TechEmpower SCRAM local bench package under `go.work`. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./benchmarks/techempower/tetra/cmd/scram-local-bench -count=1`; `go list -m all` | `TETRA-BUG-0056` |
| 2026-05-22 | Added backend net epoll lifecycle and PostgreSQL result guard microservice examples; probed nonblocking accept-without-client, epoll flag helpers, row-description guard paths, NULL data-row values, and command tag parsing. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | none |
| 2026-05-22 | Added backend HTTP response guard and JSON escape guard microservice examples; probed offset request-head framing, keep-alive detection, query route classification, custom response heads, JSON string/object escaping, and integer length helpers. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | none |
| 2026-05-22 | Added backend networking policy and crypto/serialization guard microservice examples; probed port clamping, fallback port selection, retry backoff caps, constant-time byte equality, checksum parity, seed mixing, and u8 pair packing clamps. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | none |
| 2026-05-22 | Added backend filesystem path-policy and time/collection window microservice examples; probed ASCII route metrics, repeated-slash path depth, root detection, missing-file checks, zero-length i32 collections, fallback values, negative durations, and timeout clamps. | `go test ./compiler -run TestBuildMakeZeroLengthSlices -count=1`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0057` |
| 2026-05-22 | Added backend math/testing decision and slice/capability window microservice examples; probed clamp/min/max/add decision chains, testing status combination, island and heap slice sums, zero-length slice fallbacks, capability wrapper tokens, heap stores, and MMIO round trips. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | none |
| 2026-05-22 | Added backend memory/io buffer and sync/async status microservice examples; probed zero-length and negative-length memory helpers, byte copy/fill round trips, IO capability wrappers, MMIO read/write, sync status merging, readiness gates, countdown clamps, barrier targets, and async helper lowering with sync fallback selection. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | none |
| 2026-05-22 | Added backend experimental route-policy and buffer mirror microservice examples; probed `lib.experimental.*` compatibility forwarding for route strings, filesystem path policy, networking/time/testing/sync decisions, heap slices, crypto/serialization checksums, memory copies, MMIO wrappers, and async helper lowering. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | none |
| 2026-05-22 | Added backend modular web-stack pack microservice example; probed local-module imports over HTTP routing/JSON response helpers and PostgreSQL prepared-frame/data-row helpers under direct execution and `--jobs 4 --interface-only`. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | none |
| 2026-05-23 | Added backend capsule source-root microservice project; probed `Capsule.t4` entry discovery, `src`/`tests` source roots, cross-source local imports, `tetra check`, `tetra build`, `tetra run`, `tetra test`, and explicit dogfood source-file CLI handling near capsule projects. | `go test ./cli/cmd/tetra -run 'TestTestCommandRunsMicroserviceCapsuleSourceRootExample|TestBuildCheckRunCommandsAcceptExplicitProjectSourceFile' -count=1`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0058` |
| 2026-05-23 | Added backend PostgreSQL session-state wire microservice example; probed startup-message layout, simple-query frames, describe portal/statement frames, execute max-row encoding, sync/terminate frames, and ready-for-query state bytes. | `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | none |
| 2026-05-23 | Added backend HTTP status/header matrix microservice example; probed response status serialization, decimal writes, route path chars, string keep-alive detection, and byte-buffer mixed-case `Connection: close` handling. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1` | `TETRA-BUG-0059` |
| 2026-05-23 | Added backend JSON control-character matrix microservice example; probed tab, carriage-return, newline, empty-message object, signed integer digit, negative world-object length, and lowercase hex digit helper paths. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_control_matrix_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | none |
| 2026-05-23 | Added backend HTTP header-whitespace microservice example; probed string and byte-buffer keep-alive handling for exact, multi-space, and tabbed `Connection: close` headers plus request-head length controls. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_header_whitespace_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1` | `TETRA-BUG-0060` |
| 2026-05-23 | Added backend HTTP Connection token-list microservice example; probed string and byte-buffer keep-alive handling for exact close, keep-alive-only, comma-separated `keep-alive, close`, and `upgrade, Close` header values. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_list_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_header_whitespace_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1` | `TETRA-BUG-0061` |
| 2026-05-23 | Added backend HTTP Connection header-scope microservice example; probed `X-Connection`, `Proxy-Connection`, `Connection-Mode`, and exact `Connection` semantics through string and byte-buffer keep-alive helpers. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_scope_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_list_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_header_whitespace_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1` | `TETRA-BUG-0062` |
| 2026-05-23 | Added backend HTTP Connection token-boundary microservice example; probed `closex`, `close-upgrade`, `enclose`, `close, keep-alive`, and trailing-whitespace `close` values through string and byte-buffer keep-alive helpers. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_token_boundary_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_scope_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_list_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_header_whitespace_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1` | `TETRA-BUG-0063` |
| 2026-05-23 | Added backend HTTP request-version scope microservice example; probed HTTP/1.0 requests with header-only `HTTP/1.1` markers, `HTTP/1.10` request-line prefixes, query routes, and byte-buffer keep-alive routing. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_version_scope_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_token_boundary_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_scope_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_list_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_header_whitespace_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1` | `TETRA-BUG-0064` |
| 2026-05-23 | Added backend HTTP request-target guard microservice example; probed double-space empty targets, query-only targets, root and root-query not-found controls, and byte-buffer plaintext/missing query routing. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_target_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_pipeline_gateway_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_response_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_version_scope_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1` | `TETRA-BUG-0065` |
| 2026-05-23 | Added backend HTTP request-line token guard microservice example; probed extra middle tokens before `HTTP/1.1`, HTTP/1.0 followed by a later `HTTP/1.1`, and valid string/byte-buffer route controls. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_line_token_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_target_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_version_scope_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_pipeline_gateway_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1` | `TETRA-BUG-0066` |
| 2026-05-23 | Added backend HTTP keep-alive request-target guard microservice example; probed malformed `noslash` and query-only targets in string/byte helpers plus `/`, `/?query`, `/json`, and `Connection: close` controls. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_keep_alive_target_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_target_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_line_token_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_version_scope_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_pipeline_gateway_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1` | `TETRA-BUG-0067` |
| 2026-05-23 | Added backend HTTP Connection header/body scope microservice example; probed body-only `Connection: close`, real header `Connection: close`, CRLF header terminators, and byte-buffer offset keep-alive helpers. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_body_scope_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_token_boundary_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_scope_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_list_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_header_whitespace_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1` | `TETRA-BUG-0068` |
| 2026-05-23 | Added backend HTTP keep-alive method-token guard microservice example; probed malformed `GE:T`, tabbed method, and `GET@` method tokens in string/byte helpers plus valid `GET`, `POST`, and `Connection: close` controls. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_keep_alive_method_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_keep_alive_target_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_line_token_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_version_scope_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_pipeline_gateway_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1` | `TETRA-BUG-0069` |
| 2026-05-23 | Added backend JSON hex-digit guard microservice example; probed lowercase hex helper bounds for `-1`, `16`, and `99` plus normal nibble inputs and message-object serialization controls. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_hex_digit_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_control_matrix_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_escape_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/jsonrt -count=1` | `TETRA-BUG-0070` |
| 2026-05-23 | Added backend time overflow guard microservice example; probed `millis_from_seconds(2147483)`, `millis_from_seconds(2147484)`, positive `add_duration_ms` near `Int` max, and negative duration underflow controls. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_time_overflow_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_time_collection_window_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0071` |
| 2026-05-23 | Added backend PostgreSQL C-string bounds guard microservice example; probed bounded NUL scans for valid subranges, empty/reversed ranges, and negative start indexes. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_cstring_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_session_state_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_result_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0072` |
| 2026-05-23 | Added backend PostgreSQL DataRow signed-length guard microservice example; probed valid DataRow columns, NULL-style negative sentinel handling, malformed `0xfffffffe` length normalization, and following-column recovery. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_length_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_result_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_cstring_bounds_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0073` |
| 2026-05-23 | Added backend PostgreSQL ASCII i32 bounds guard microservice example; probed signed and unsigned ASCII integer parsing, non-digit stops, zero-count ranges, and negative-start parser bounds. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ascii_i32_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_result_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_length_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0074` |
| 2026-05-23 | Added backend PostgreSQL CommandComplete bounds guard microservice example; probed `INSERT 0 3`, `UPDATE 12`, no-digit tags, digit-only subranges, empty ranges, negative payload lengths, and negative-start parser bounds. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_command_tag_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_result_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ascii_i32_bounds_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0075` |
| 2026-05-23 | Added backend PostgreSQL RowDescription bounds guard microservice example; probed valid type OID scans, negative/out-of-range column indexes, truncated metadata, empty payload lengths, negative payload lengths, and negative-start parser bounds. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_row_description_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_result_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_command_tag_bounds_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0076` |
| 2026-05-23 | Added backend PostgreSQL DataRow bounds guard microservice example; probed valid DataRow integer columns, negative/out-of-range column indexes, negative-start length/start helpers, and `data_row_i32_at` missing-value controls. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_result_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_length_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0077` |
| 2026-05-23 | Added backend PostgreSQL frame-header bounds guard microservice example; probed valid Sync frame header reads, negative-start header reads, and malformed length fields below the PostgreSQL four-byte frame-length minimum. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_header_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_result_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_bounds_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0078` |
| 2026-05-23 | Added backend PostgreSQL ReadyForQuery status bounds guard microservice example; probed valid idle, in-transaction, and failed-transaction status bytes plus negative-start status reads. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ready_status_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_session_state_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_prepared_pipeline_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0079` |
| 2026-05-23 | Added backend PostgreSQL column-count bounds guard microservice example; probed valid RowDescription and DataRow count fields plus negative-start count readers. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_column_count_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_row_description_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_bounds_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0080` |
| 2026-05-23 | Added backend PostgreSQL big-endian reader bounds guard microservice example; probed valid i32/i16 big-endian reads, signed i32 normalization, and negative-start reader sentinels. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_read_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_column_count_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_header_bounds_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0081` |
| 2026-05-23 | Added backend PostgreSQL big-endian writer bounds guard microservice example; probed valid i32/i16 big-endian writes, negative-start writer sentinels, and preserved buffer contents after rejected writes. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_write_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_read_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_column_count_bounds_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0082` |
| 2026-05-23 | Added backend PostgreSQL text writer bounds guard microservice example; probed valid ASCII/C-string writes, negative-start text writer sentinels, and preserved buffer contents after rejected writes. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_text_write_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_write_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_prepared_pipeline_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0083` |
| 2026-05-23 | Added backend HTTP writer bounds guard microservice example; probed valid ASCII, CRLF, header, and decimal writes plus negative-start writer sentinels and preserved buffer contents after rejected writes. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_writer_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_response_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0084` |
| 2026-05-23 | Added backend JSON writer bounds guard microservice example; probed valid escaped JSON string and message-object writes plus negative-start writer sentinels and preserved buffer contents after rejected writes. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_writer_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_control_matrix_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_escape_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0085` |
| 2026-05-23 | Added backend HTTP/JSON i32 minimum guard microservice example; probed HTTP and JSON decimal digit helpers, JSON world-object sizing, HTTP decimal byte writes, and rejected negative-start writes for `-2147483648`. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_json_i32_min_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_control_matrix_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0086` |
| 2026-05-23 | Added backend crypto mix i32 minimum guard microservice example; probed stable crypto seed mixing when `seed * 33 + value` reaches `-2147483648` plus existing positive/negative controls and the experimental mirror. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_crypto_mix_min_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_crypto_serialization_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_experimental_buffer_mirror_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0087` |
| 2026-05-23 | Added backend networking backoff overflow guard microservice example; probed capped retry backoff before i32 doubling overflow, ordinary capped backoff, uncapped below-overflow behavior, negative base clamping, and the experimental networking mirror. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_network_backoff_overflow_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_network_policy_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_experimental_route_policy_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0088` |
| 2026-05-23 | Added backend net epoll event bounds guard microservice example; probed valid fd/flag extraction plus empty and one-slot event buffers. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_net_epoll_event_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_net_epoll_lifecycle_service.tetra`; `go test ./compiler -run 'TestBuildCoreNetSmoke|TestMicroserviceExamplesAndBugLedger' -count=1` | `TETRA-BUG-0089` |
| 2026-05-23 | Added backend PostgreSQL short frame-header guard microservice example; probed valid Sync frames plus empty, tag-only, and truncated typed-frame buffers. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_header_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_read_bounds_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0090` |
| 2026-05-23 | Added backend PostgreSQL big-endian reader short-buffer guard microservice example; probed valid i32/i16 reads plus empty, truncated, and offset-truncated buffers. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_read_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_read_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_short_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0091` |
| 2026-05-23 | Added backend PostgreSQL big-endian writer short-buffer guard microservice example; probed valid i32/i16 writes plus empty, truncated, and offset-truncated destination buffers. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_write_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_write_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_read_short_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0092` |
| 2026-05-23 | Added backend PostgreSQL text writer short-buffer guard microservice example; probed valid ASCII/C-string writes plus empty and truncated destination buffers. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_text_write_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_text_write_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_write_short_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0093` |
| 2026-05-23 | Added backend HTTP writer short-buffer guard microservice example; probed valid ASCII, CRLF, header, and decimal writes plus empty and truncated destination buffers. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_writer_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_writer_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0094` |
| 2026-05-23 | Added backend JSON writer short-buffer guard microservice example; probed valid escaped strings and message objects plus empty and truncated destination buffers. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_writer_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_writer_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_json_control_matrix_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0095` |
| 2026-05-23 | Added backend PostgreSQL frame writer short-buffer guard microservice example; probed valid startup, Simple Query, Parse, Bind, Describe, Execute, Sync, and Terminate frames plus empty and truncated destination buffers. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_writer_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_prepared_pipeline_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_session_state_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0096` |
| 2026-05-23 | Added backend PostgreSQL bounded parser short-buffer guard microservice example; probed valid bounded C-string, ASCII integer, and CommandComplete parsing plus overstated limits/counts for short physical buffers. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_parser_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_cstring_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ascii_i32_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_command_tag_bounds_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0097` |
| 2026-05-23 | Added backend PostgreSQL ReadyForQuery status short-buffer guard microservice example; probed valid idle/in-transaction/failed-transaction status bytes plus empty and offset-short payloads. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ready_status_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ready_status_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_session_state_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_prepared_pipeline_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0098` |
| 2026-05-23 | Added backend HTTP request short-buffer guard microservice example; probed valid pipelined requests plus empty, offset-short, and overstated request windows. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_connection_body_scope_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_line_token_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0099` |
| 2026-05-23 | Added backend HTTP response writer short-buffer guard microservice example; probed valid response heads, plaintext responses, and JSON responses plus empty, prefix-short, and body-short destination buffers. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_response_writer_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_response_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_writer_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0100` |
| 2026-05-23 | Added backend PostgreSQL signed big-endian writer guard microservice example; probed negative i32/i16 two's-complement bytes for `-1`, `-2`, `-32768`, and `-2147483648` while preserving positive controls. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_signed_write_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_write_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_read_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_writer_short_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0101` |
| 2026-05-23 | Added backend PostgreSQL signed frame-length guard microservice example; probed malformed negative signed frame lengths for `-2` and `-2147483648` while preserving valid Sync and positive short-length controls. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_signed_length_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_header_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_signed_write_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0102` |
| 2026-05-23 | Added backend PostgreSQL high-bit big-endian reader guard microservice example; probed `0x7fffffff` positive control plus `0x80000000` and `0xfffffffe` high-bit sentinels. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_high_bit_read_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_read_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_signed_length_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_signed_write_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0103` |
| 2026-05-23 | Added backend PostgreSQL ASCII i32 minimum guard microservice example; probed direct bounded parser and DataRow integer parsing for `-2147483648` plus `2147483647` positive max control. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ascii_i32_min_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ascii_i32_bounds_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | none |
| 2026-05-23 | Added backend time negative-base duration guard microservice example; probed negative-base positive-delta recovery, still-negative sums, zero-delta negative bases, negative-delta subtraction, and positive overflow saturation. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_time_negative_base_delta_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_time_overflow_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_time_collection_window_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0104` |
| 2026-05-23 | Added backend HTTP request-line CRLF guard microservice example; probed valid CRLF request routing plus LF-only and bare-CR `HTTP/1.1` terminator rejection for string and byte-buffer helpers. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_crlf_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_line_token_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_version_scope_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_short_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0105` |
| 2026-05-23 | Added backend filesystem embedded-NUL exists guard microservice example; probed normal existing/missing paths plus `/<NUL>_suffix` prefix-truncation rejection. | `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_filesystem_nul_exists_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_filesystem_path_policy_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_experimental_route_policy_service.tetra`; `go test ./compiler -run 'TestFilesystemRuntimeExistsBuildAndRunLinuxX64|TestMicroserviceExamplesAndBugLedger' -count=1` | `TETRA-BUG-0106` |
| 2026-05-23 | Added backend HTTP request-target character guard microservice example; probed tab and raw control-byte target rejection in string/byte helpers plus valid query-string target controls. | `go run ./cli/cmd/tetra check examples/microservices/backend_http_request_target_char_guard_service.tetra`; `go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-http-request-target-char-guard examples/microservices/backend_http_request_target_char_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_target_char_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_target_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_line_token_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_keep_alive_method_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_request_crlf_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1` | `TETRA-BUG-0107` |
| 2026-05-23 | Added backend PostgreSQL ASCII i32 overflow guard microservice example; probed out-of-range positive/negative ASCII integers in direct and DataRow parsing while preserving max/min i32 boundary controls. | `go run ./cli/cmd/tetra check examples/microservices/backend_postgres_ascii_i32_overflow_guard_service.tetra`; `go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-postgres-ascii-i32-overflow-guard examples/microservices/backend_postgres_ascii_i32_overflow_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ascii_i32_overflow_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ascii_i32_min_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ascii_i32_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_parser_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_length_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1` | `TETRA-BUG-0108` |
| 2026-05-23 | Added backend PostgreSQL CommandComplete affected-row overflow guard microservice example; probed out-of-range `UPDATE` and `INSERT` affected-row counts while preserving `2147483647` and valid subrange controls. | `go run ./cli/cmd/tetra check examples/microservices/backend_postgres_command_tag_overflow_guard_service.tetra`; `go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-postgres-command-tag-overflow-guard examples/microservices/backend_postgres_command_tag_overflow_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_command_tag_overflow_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_command_tag_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_parser_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_ascii_i32_overflow_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/pgrt -count=1` | `TETRA-BUG-0109` |
| 2026-05-23 | Added backend PostgreSQL CommandComplete trailing-count guard microservice example; probed malformed non-trailing digit runs such as `UPDATE 12 rows` while preserving valid trailing `UPDATE`, `INSERT`, and subrange controls. | `go run ./cli/cmd/tetra check examples/microservices/backend_postgres_command_tag_trailing_guard_service.tetra`; `go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-postgres-command-tag-trailing-guard examples/microservices/backend_postgres_command_tag_trailing_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_command_tag_trailing_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_command_tag_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_command_tag_overflow_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_parser_short_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/pgrt -count=1` | `TETRA-BUG-0110` |
| 2026-05-23 | Added backend PostgreSQL DataRow truncated-value guard microservice example; probed advertised positive value lengths whose physical bytes are missing while preserving valid and empty DataRow value controls. | `go run ./cli/cmd/tetra check examples/microservices/backend_postgres_data_row_truncated_value_guard_service.tetra`; `go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-postgres-data-row-truncated-value-guard examples/microservices/backend_postgres_data_row_truncated_value_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_truncated_value_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_length_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_result_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/pgrt -count=1` | `TETRA-BUG-0111` |
| 2026-05-23 | Added backend PostgreSQL frame total-length overflow guard microservice example; probed maximum valid typed-frame total length plus signed maximum length whose tag-inclusive total would overflow. | `go run ./cli/cmd/tetra check examples/microservices/backend_postgres_frame_total_overflow_guard_service.tetra`; `go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-postgres-frame-total-overflow-guard examples/microservices/backend_postgres_frame_total_overflow_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_total_overflow_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_header_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_signed_length_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_short_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/pgrt -count=1` | `TETRA-BUG-0112` |
| 2026-05-23 | Added backend HTTP negative Content-Length guard microservice example; probed response-head sizing and writing for negative Content-Length rejection while preserving zero and positive length controls. | `go run ./cli/cmd/tetra check examples/microservices/backend_http_negative_content_length_guard_service.tetra`; `go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-http-negative-content-length-guard examples/microservices/backend_http_negative_content_length_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_negative_content_length_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_response_writer_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_json_i32_min_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1` | `TETRA-BUG-0113` |
| 2026-05-23 | Added backend HTTP status-code guard microservice example; probed response-head sizing and writing for non-three-digit status-code rejection while preserving `100`, `200`, and `999` controls. | `go run ./cli/cmd/tetra check examples/microservices/backend_http_status_code_guard_service.tetra`; `go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-http-status-code-guard examples/microservices/backend_http_status_code_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_code_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_negative_content_length_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_response_writer_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1` | `TETRA-BUG-0114` |
| 2026-05-23 | Added backend HTTP header injection guard microservice example; probed CR/LF injection rejection for header names, header values, response reason text, and response header fields while preserving valid header/response controls. | `go run ./cli/cmd/tetra check examples/microservices/backend_http_header_injection_guard_service.tetra`; `go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-http-header-injection-guard examples/microservices/backend_http_header_injection_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_header_injection_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_code_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_negative_content_length_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_response_writer_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1` | `TETRA-BUG-0115` |
| 2026-05-23 | Added backend HTTP header control-byte guard microservice example; probed non-HTAB control-byte rejection for header values, response reason text, and response header fields while preserving valid HTAB values. | `go run ./cli/cmd/tetra check examples/microservices/backend_http_header_control_guard_service.tetra`; `go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-http-header-control-guard examples/microservices/backend_http_header_control_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_header_control_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_header_injection_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_code_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_negative_content_length_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_response_writer_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_http_status_matrix_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/httprt ./compiler/internal/webrt -count=1` | `TETRA-BUG-0116` |
| 2026-05-23 | Added backend PostgreSQL Parse parameter-count guard microservice example; probed signed i16 OID-count overflow rejection for Parse frame sizing and writing while preserving small valid Parse frames. | `go run ./cli/cmd/tetra check examples/microservices/backend_postgres_parse_count_guard_service.tetra`; `go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-postgres-parse-count-guard examples/microservices/backend_postgres_parse_count_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_parse_count_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_prepared_pipeline_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_parser_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_writer_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_signed_write_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/pgrt -count=1` | `TETRA-BUG-0117` |
| 2026-05-23 | Added backend net TCP port bounds guard microservice example; probed rejection of negative and above-range loopback bind ports while preserving ephemeral port `0`. | `go run ./cli/cmd/tetra check examples/microservices/backend_net_port_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-net-port-bounds-guard examples/microservices/backend_net_port_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_net_port_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_net_epoll_lifecycle_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_net_epoll_event_bounds_guard_service.tetra`; `go test ./compiler -run 'TestBuildCoreNetSmoke|TestMicroserviceExamplesAndBugLedger|TestNetRuntimeSocketLifecycleBuildAndRunLinuxX64|TestNetRuntimeEpollWaitOneIntoBuildAndRunLinuxX64' -count=1`; `go test ./compiler/internal/actorsrt -count=1` | `TETRA-BUG-0118` |
| 2026-05-23 | Added backend PostgreSQL signed column-count guard microservice example; probed high-bit RowDescription/DataRow count rejection while preserving one-column payload controls. | `go run ./cli/cmd/tetra check examples/microservices/backend_postgres_column_count_signed_guard_service.tetra`; `go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-postgres-column-count-signed-guard examples/microservices/backend_postgres_column_count_signed_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_column_count_signed_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_column_count_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_row_description_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_data_row_bounds_guard_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/pgrt -count=1` | `TETRA-BUG-0119` |
| 2026-05-23 | Added backend PostgreSQL embedded-NUL C-string guard microservice example; probed C-string length/writer rejection for startup, query, statement, and portal fields while preserving valid frame controls. | `go run ./cli/cmd/tetra check examples/microservices/backend_postgres_cstring_nul_guard_service.tetra`; `go run ./cli/cmd/tetra build --target linux-x64 -o /tmp/tetra-postgres-cstring-nul-guard examples/microservices/backend_postgres_cstring_nul_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_cstring_nul_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_text_write_bounds_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_frame_writer_short_guard_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_session_state_service.tetra`; `go run ./cli/cmd/tetra run --target linux-x64 examples/microservices/backend_postgres_prepared_pipeline_service.tetra`; `go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1`; `go test ./compiler/internal/pgrt -count=1` | `TETRA-BUG-0120` |

## Notes

- The current v0.4.0 production service surface is actor/task based on
  Linux-x64. A stable Tetra HTTP server API was not found in the supported
  surface; that is recorded as a scope boundary, not a confirmed language bug.
- Future confirmed bugs should include the failing source, command, expected
  behavior, actual behavior, and whether the issue is parser, checker,
  lowering, runtime, or tooling.
