# Tetra Current Supported Surface

Status: current for `v1.0.0`.

This document is the short release-truth layer for the current public Tetra
profile. It records what the repository may describe as supported now, and what
must still be described as future or planned.

The current major line is `v1.0.0`. The scope contract is
`docs/spec/v1_scope.md`, but this is still a bounded release profile: it does
not promote full platform guarantees, EcoNet, hosted TetraHub, non-Linux
desktop GUI runtime, or full first-class language guarantees beyond the
feature-registry entries marked current.

## Current Major Scope

The current major line is `v1.0.0`. Its release identity and verification
surface are tracked here:

- Scope contract: `docs/spec/v1_scope.md`
- Release checklist: `docs/checklists/v1_0_release_gate.md`
- Release gate script: `scripts/release/v1_0/gate.sh`
- Release notes: `docs/release-notes/v1_0.md`
- Final handoff: `docs/release/v1_0_final_handoff.md`

The version metadata is promoted to `v1.0.0`. Tagging still requires a fresh
green `scripts/release/v1_0/gate.sh` report and matching handoff evidence.

## Current Release Gate

- Current gate: `scripts/release/v1_0/gate.sh`.
- Current checklist: `docs/checklists/v1_0_release_gate.md`.
- Previous gate: `scripts/release/v0_4_0/gate.sh` remains for the immutable
  `v0.4.0` Linux-x64 release truth and must not be rewritten by this profile.
- v1 safety evidence closure is documented in `docs/spec/v1_scope.md`
  and `docs/checklists/v1_0_release_gate.md`. It requires the same-branch
  aggregate compiler command
  `go test ./compiler/... -run 'Ownership|Borrow|Consume|Inout|Lifetime|Resource|Island|Actor|Task|Unsafe|Capability|Effect|Privacy|Consent|Budget|MMIO|Mem' -count=1`
  plus `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- Previous gate: `scripts/release/v0_2_0/gate.sh` remains for the immutable
  `v0.2.0` tag.
- Historical gate: `scripts/release/v0_1_3/gate.sh` remains for the immutable
  `v0.1.3` tag.
- Historical gate: `scripts/release/v0_1_1/gate.sh` remains for the immutable
  `v0.1.1` tag.
- Separately gated post-v0.4 promotion evidence for WASM/Web UI/Linux-x64 GUI
  lives under `scripts/release/post_v0_4/wasm-ui-gui-production-gate.sh` and
  writes fresh reports under `reports/wasm-ui-gui` when requested. The v1 gate
  archives fresh WASM and web UI smoke artifacts rather than reusing stale
  post-v0.4 evidence.

## Supported Now

- Flow indentation syntax for the examples, standard library sources, runtime
  sources, and self-hosted runtime snippets covered by the release gate.
- Local compiler and CLI workflows: `check`, `build`, `run`, `fmt`, `test`,
  `doc`, `doctor`, `targets`, `features`, `formats`, `new`, `interface`,
  `project`, `workspace`, `smoke`, `eco`, `clean`, `version`, and `lsp`.
- The current `lsp --stdio` surface is a minimal JSON-RPC `"2.0"` server for
  editor smoke coverage. It accepts request `id` values as JSON numbers or
  strings and echoes the same value and type in success or error responses.
  Notifications omit `id`. Current request coverage includes initialize,
  shutdown, exit, didOpen/didChange/didClose diagnostics, document symbols,
  hover, completion, definition, references, rename, formatting, and code
  actions. Unknown request methods return JSON-RPC `-32601`; malformed params
  return `-32602`; invalid request envelopes return `-32600`; parse failures
  return `-32700`. For unopened documents, document symbols, completion,
  formatting, code actions, and references return empty arrays, while hover,
  definition, and rename return `null`.
- LSP rename is a conservative single-file top-level symbol operation. It uses
  the open document's top-level LSP symbol table, skips common line comments and
  string literals, validates `newName` as a Tetra identifier with JSON-RPC
  `-32602` for invalid names, and returns `null` when same-named local bindings
  or parameters would make the edit ambiguous. Project-wide and cross-module
  rename are outside the current contract, so public API renames should still
  be reviewed through the resulting diff.
- Native build/smoke coverage for `linux-x64`, plus build-only coverage for
  `macos-x64` and `windows-x64`.
- WASM artifact/import preflight for `wasm32-wasi` and `wasm32-web` through
  `smoke --run=false`, with runtime proof coming from the dedicated WASI and
  browser-backed web runner smoke reports validated by the gate.
- Local Eco package lifecycle validation for verify, lock generation/validation
  through `--lock` workflows, pack/unpack, vault, stable local publish
  metadata, beta publish metadata, target-aware downloads, and stable/beta
  TetraHub store fixtures, including local mirror reports and single-origin
  HTTP(S) fetch into a verified local store.
- JSON reports and validators for diagnostics, tests, smoke lists, targets,
  doctor output, web UI smoke, actor transport evidence, artifact hashes, and
  release state.
- Target-neutral IR verification before lowering results reach public codegen:
  main metadata, function slot metadata, branch labels, stack heights, local
  slots, returns, calls, unknown instructions, and unsupported lowering paths
  are reported with structured diagnostics.
- Static monomorphized generic functions: generic functions with inferred value
  arguments are parsed, checked, formatted, documented, and specialized with
  deterministic names across modules. The current truth boundary excludes
  runtime generic values, explicit type arguments, generic structs,
  higher-ranked generics, full protocol-bound generic dispatch, specialization
  optimization, and any dynamic dispatch claim.
- Static protocol conformance: protocol declarations and `impl Type: Protocol`
  are checked against extension/static methods, including compatible effects,
  async, throws, parameter ownership markers, params, return types, and MVP
  generic requirement signature shape (`func req<T>(...)`). This is static conformance only: no witness
  tables, trait objects, runtime protocol values, or dynamic dispatch model are
  introduced.
- Generic protocol requirement parsing/checking in MVP form (`func req<T>(...)`)
  with signature-shape conformance checks and no new runtime dispatch model.
- Static protocol-bound generics: generic function type parameters with protocol
  bounds are validated during monomorphization, including same-module and
  cross-module impl conformance with parameter ownership markers, requirement
  signature shape, and visibility diagnostics. This does not introduce calls through generic protocol bounds,
  witness tables, trait objects, runtime protocol values, or dynamic dispatch.
- Enum payload constructors and exhaustive enum match/catch coverage: positional
  enum payload constructors and payload bindings are supported for
  match/catch/if-let, with exhaustive unguarded enum match/catch checks and
  stable diagnostics for arity, type, duplicate, default-order, and payload
  syntax errors. Cross-module enum constructor/match paths are checked and
  lowered; advanced ADT constructors, nested destructuring patterns, richer
  payload algebra, and guard expansion remain future/post-v1.
- Function type references in type positions (`fn(T1, T2) -> R`) plus the
  current Level 0 callable MVP for direct local calls of let-bound
  non-capturing closure values, plus callback-parameter calls in callees when
  the call-site passes a known symbol-backed function-typed local or a direct
  named non-generic non-throwing function/closure symbol. The current safe
  subset also allows returning direct named or otherwise symbol-backed
  non-generic non-throwing function values from functions with function-typed
  returns, function-typed local-to-local binding (`let g: fn(...) -> ... = f`)
  when signatures match, including snapshot copies from mutable function-typed
  locals, symbol-backed same-module and namespace/selective imported public
  function-typed globals for direct calls plus local
  function-typed initialization/reassignment/direct callback arguments,
  non-capturing closure-literal function-typed globals,
  same-module mutable global reassignment with direct calls or synchronous
  callback arguments, stable diagnostics for imported mutable
  function-typed globals that would require cross-module global-data ABI, and
  actor/task boundary diagnostics when a worker directly dispatches through a
  same-module mutable function-typed global, an imported immutable
  function-typed global whose target touches mutable globals, or passes it as a
  synchronous callback argument, passes a same-module or imported
  symbol-backed callback argument whose target touches mutable globals, passes
  a same-module or imported direct function-typed return-call callback argument
  whose returned target or multi-return target set touches mutable globals,
  preserves that classification through local/field alias returns and returned
  struct/enum aggregate fields or payloads across module boundaries, and
  inferable same-module/imported generic-symbol
  initializers, and
  signature-compatible mutable local reassignment among supported
  function-typed values, including target-set-backed parameter-return calls such
  as `identity(captured)` or `callbacks.identity(captured)`. Closure literals assigned to a declared function type
  or passed directly as callback arguments must match the declared parameter
  arity exactly before lowering. Capturing closure literals carry conservative
  lifetime/ABI evidence through the Level 2 `fnptr` fast path and the
  full-callable handle path described below. `fnptr` remains the compact
  direct-call ABI for up to eight captured environment slots; larger safe
  by-value captures use a fixed 4-slot callable handle for local storage,
  mutable local reassignment, function-typed returns, same-module global
  snapshots, struct fields, enum payloads, synchronous callback arguments,
  cross-module returned values, aliases, and generated `.t4i` metadata.
  Mutable by-reference captures, pointer/resource captures, and thread-boundary
  callable escape keep stable diagnostics until an explicit
  ownership/synchronization transfer model exists.
  Function type references may declare a typed-error edge with
  `fn(...) -> R throws E`; the current runtime support for throwing callables is
  limited to explicitly declared immutable local direct-try bindings to
  concrete symbols or captured closure literals, such as
  `let cb: fn(Int) -> Int throws Boom = risky` or
  `let cb: fn(Int) -> Int throws Boom = fn(x: Int) -> Int throws Boom: ...`
  followed by `try cb(41)`. Captured throwing closure literals are also
  covered for mutable local reassignment, direct callback arguments,
  function-typed returns, and immutable local struct-field or enum-payload
  direct-try dispatch or aliases, plus mutable local struct-field/enum-payload
  reassignment direct-try dispatch when the declared `fn(...) -> R throws E`
  signature matches exactly. The concrete-symbol slice additionally supports
  declared function-typed returns of a concrete throwing symbol such as
  `func pick() -> fn(Int) -> Int throws Boom: return risky` followed by local
  direct-try dispatch, plus immutable local struct-field direct-try dispatch
  and enum-payload pattern-bound direct-try dispatch for concrete throwing
  symbols, plus immutable same-module or imported-public function-typed global
  direct-try dispatch, local aliases, mutable local reassignment, direct callback arguments, local
  struct-field initializer/reassignment, and enum-payload reassignment for concrete throwing symbols, plus same-module
  mutable function-typed global direct-try dispatch and direct throwing callback
  arguments, plus local struct-field/enum-payload storage direct-try after
  compatible concrete throwing-symbol initialization or reassignment, plus
  direct synchronous callback-parameter dispatch through `try cb(...)` when the
  callback parameter type declares the same throws type.
  Direct calls through function-typed locals report unsupported explicit type
  arguments, arity mismatches, type mismatches, and mixed labeled/unlabeled
  argument lists against the visible callback name; captured `fnptr` local
  semantic-clause violations use the same visible `function-typed callback`
  phrase.
  Direct calls through function-typed struct fields report unsupported explicit
  type arguments and arity mismatches against the visible field path.
  Pattern-bound function-typed enum payload calls report the same call-shape
  ownership, and semantic-clause diagnostics against the visible payload
  binding.
- Semantic-clause checker phase 1 for `noalloc`/`noblock`/`realtime`:
  resolved direct calls, closure-symbol calls, and function-typed callback
  arguments are validated against clause contracts. Callback parameters and
  target-set-backed function-typed values use their declared function-type
  effects when a single concrete symbol is not available; direct calls through
  function-typed locals, struct fields, and globals report violations against
  the user-visible callable name, with captured `fnptr` locals using the
  visible `function-typed callback` phrase; function-typed local,
  struct-field, and immutable/mutable global callback arguments report
  violations against the visible argument name, and function-typed return-call
  callback arguments use the visible call form such as `pick()`; direct closure
  literal callback
  arguments report signature/effect/unsupported-throwing diagnostics as
  `closure literal`, including generic closure capture rejections;
  `realtime` requires `noalloc` and `noblock`.
- Effects and `uses` checker MVP: stable effect names and groups are checked,
  function calls propagate callee effects transitively across resolved direct,
  generic, protocol, and supported callable paths, and missing `uses`
  declarations are diagnostics. This is a static MVP; it does not infer effects
  or claim proof-level effect-system guarantees.
- Capabilities and unsafe boundary MVP: `cap.io` and `cap.mem` are opaque tokens
  obtained only inside `unsafe` blocks; raw memory/MMIO operations require the
  matching `uses` effects, an `unsafe` boundary, the required capability
  argument, and capsule permissions for attenuated capability groups. This is
  compile-time gating with minimal current backend lowering, not a broad
  safe-code capability construction model. On `wasm32-wasi` and `wasm32-web`,
  raw unsafe allocation, capability-token construction, raw memory access,
  MMIO, pointer arithmetic, and context switching are blocked by compile-time
  target diagnostics before WASM backend emission; safe slices and the current
  compile-compatible scoped island path remain available.
- Privacy and consent checker MVP: `uses privacy` requires a `privacy` semantic
  clause, recursive signature detection (parameter/return/throws) unwraps `?`
  and `[]` layers and treats `secret.*` as secret-bearing, such signatures
  require `consent(<token>)`, the consent parameter must have `consent.token`
  type, and privacy builtins require the privacy effect plus consent token.
  Lowering currently uses a minimal local contract (`consent_token` lowers to
  an opaque runtime sentinel, consent clauses validate exact sentinel equality,
  and `secret_seal_i32`/`secret_unseal_i32` preserve payload value while
  evaluating token arguments). This is static auditing and
  call-shape/lowering-shape enforcement, not cryptographic isolation or
  distributed consent enforcement.
- Budget clause lowering MVP: `budget(<non-negative integer constant>)`
  requires `uses budget`, and lowering emits deterministic budget guard
  instructions with stable local-slot metadata. The checker also applies a
  conservative static cross-edge guardrail: direct calls, `core.spawn`, and
  `core.task_spawn_*` edges into `budget(N)` functions/workers require a caller
  budget context of at least `N`; the edge call charge remains covered by the
  caller's local lowering guard. `budget(0)` remains a deterministic local
  failure-before-call path. Budget exhaustion has a stable local ABI:
  non-throwing functions return zero/default result slots, while throwing
  functions return zero/default error payload slots with trap status `1`. This
  is deterministic local lowering plus static edge validation, not aggregate
  runtime-wide accounting, process abort semantics, or distributed budget
  enforcement.
- Safety production core is current for the `v0.4.0` local profile. The
  release-covered core combines ownership/lifetime/borrow/consume/inout checks,
  resource finalization with stable `TETRA2101` JSON diagnostics for resource
  use-after-free, double-join, ambiguous-provenance, and same-module/cross-module
  struct-field and enum-payload alias use-after-free, including
  same-module/cross-module struct-field and enum-payload alias use-after-free
  with stable `TETRA2101` JSON diagnostic evidence, plus island transfer
  non-local-payload cases,
  callable escape diagnostics, effects/capabilities/
  privacy/consent/budget policy, unsafe boundaries, actor/task transfer safety,
  and pointer/MMIO/memory capability gates. Unsupported distributed,
  cryptographic, formal-proof, runtime-wide, and broader synchronization claims
  remain explicit boundaries rather than hidden promises.
- Top-level globals (`var`/`val`/`property`) in the current global pipeline:
  compile-time constant initializers for scalar MVP types plus `String`/`str`
  when the initializer is a string literal. Function-typed globals may be
  initialized with a same-module or imported direct named function symbol,
  called directly, assigned into local function-typed values, used as mutable
  local reassignment sources, and passed as synchronous callback arguments.
  Direct calls through function-typed globals enforce the declared global
  function type's argument count, positional type checks, and positional
  ownership markers, effect/semantic-clause checks, and report diagnostics
  against the user-visible global callable name; explicit type arguments on
  those value calls are rejected against the same user-visible global callable.
  Same-module mutable function-typed globals may be reassigned to compatible
  direct named function symbols and then called directly or passed through
  synchronous callback arguments, returned from function-typed return paths, or
  snapshotted into local or nested local struct fields or enum payloads for supported direct
  calls or synchronous callback arguments, including through known returned
  struct fields or enum payloads. Imported mutable function-typed globals are
  rejected with a stable boundary diagnostic until cross-module global-data ABI
  exists.
  Imported functions may accept structs with function-typed fields and call
  those fields when the caller passes a known local struct value or direct
  namespace/selective imported struct constructor carrying a closure literal or
  captured `ptr` closure local within the Level 2 `fnptr` envelope.
  Imported functions may also accept enums with function-typed payloads and
  call pattern-bound payload callbacks when the caller passes a known local enum
  value, direct enum-returning call, or direct namespace/selective imported
  enum constructor argument carrying a supported callable target.
  Mutable non-function globals reject direct assignment from borrowed `ptr`
  parameters, and the same global-assignment escape diagnostic is used for
  region-backed borrowed values when those values are present in a supported
  assignment source.
  Other non-constant/non-literal and unsupported-type initializers remain
  rejected.
- Top-level `property` declarations mapped onto the current global pipeline.
- Top-level language `capsule` declarations accepted as compile-time metadata
  only (duplicate-key/key-shape/value-shape checks; no runtime/codegen impact).
- Native-first `[]u16` slice support including `make_u16` and
  `core.island_make_u16`.
- `[]bool` slice support including `make_bool` and `core.island_make_bool`.
  In the current MVP lowering path, bool-slice allocation reuses the existing
  i32-width slice layout.
  `make_bool` is available on native and WASM targets, while
  `core.island_make_bool` follows the current island runtime boundary (native
  runtime scope); WASM targets provide compile-compatible island IR
  fallback (`island_new` handle token, `island_make_*` mapped to linear heap
  slice allocation by element width, `island_free` no-op).
- Ownership markers MVP for `borrow`, `inout`, and `consume` call-site
  contracts. The current checker covers local-call marker validation,
  ownership-path alias rejection, same-module/cross-module struct-field and enum-payload partial consume
  with sibling-path reuse and whole-value call/let/return rejection with stable
  `TETRA2101` CLI JSON diagnostics, including stable CLI JSON evidence for
  same-module/cross-module whole-copy rejection after partial struct/enum consume,
  same-module/cross-module enum wrapper-constructor rejection after partial consume
  with stable `TETRA2101` CLI JSON evidence,
  mutable struct-field, whole-struct, or whole-enum
  reinitialization after partial field/payload consume,
  same-module/cross-module optional-payload whole-value rejection after payload
  consume/free with stable TETRA2101 JSON diagnostic evidence, use-after-`consume`,
  same-module and cross-module interprocedural enum-payload and if-let/match
  optional-payload return resource alias double-free, including nested
  struct-field and enum-payload optional resource wrappers with stable
  same-module/cross-module `TETRA2101` CLI JSON evidence,
  same-module/cross-module task-handle/task-group if-let/match
  optional-payload join/close aliases with stable TETRA2101 CLI JSON evidence,
  and borrow escape diagnostics for returns, owned/inout calls, and supported
  mutable global assignment boundaries, including borrowed `ptr` parameters,
  same-module/cross-module scalar `ptr` `consume` and `inout` assignment,
  same-module/cross-module borrowed ptr-containing aggregate parameters including
  nested inout/global assignment paths, including same-module/cross-module
  whole-aggregate global assignment with stable `TETRA2102` JSON diagnostic
  evidence, same-module/cross-module ptr-containing enum whole-value global
  assignment with stable `TETRA2102` JSON diagnostic evidence, and stable
  same-module/cross-module global field target assignment with stable
  `TETRA2102` JSON diagnostic evidence, same-module/cross-module aggregate and
  nested-aggregate global field escapes with stable `TETRA2102` JSON diagnostic
  evidence, same-module/cross-module
  pattern-bound enum payload aliases and if-let/match optional payload aliases
  including scalar return, owned/consume/inout call, inout-assignment, and
  global-assignment escapes, with stable TETRA2102 JSON diagnostic evidence for
  same-module/cross-module ptr enum-payload return/global/inout assignment
  escapes, same-module/cross-module ptr optional-payload return/global/inout assignment escapes with stable TETRA2102 JSON diagnostic evidence plus
  same-module/cross-module ptr-containing/nested aggregate owned/consume/inout
  call rejections with stable TETRA2101 JSON diagnostic evidence,
  same-module/cross-module ptr enum-payload owned/consume/inout call
  rejections with stable TETRA2101 JSON diagnostic evidence,
  same-module/cross-module ptr optional-payload owned/consume/inout call
  rejections with stable TETRA2101 JSON diagnostic evidence, and
  same-module/cross-module slice optional-payload owned/consume/inout call
  rejections with stable TETRA2101 JSON diagnostic evidence, and
  same-module/cross-module borrowed scalar `ptr` escapes through ptr-containing
  struct `inout` assignment, same-module/cross-module fixed-array alias return
  plus direct global assignment, optional global
  assignment, and inout-assignment escapes with stable `TETRA2102` diagnostic evidence and borrowed
  string alias return/global assignment escapes with stable `TETRA2102` CLI JSON
  evidence, slice-containing struct literal/alias/nested
  struct/enum-payload return,
  struct `inout` assignment, and enum direct/alias return escapes with stable
  same-module/cross-module `TETRA2102` CLI JSON evidence, slice-containing
  struct/enum owned/consume/inout call escapes with stable
  same-module/cross-module and imported direct `TETRA2101` CLI JSON evidence,
  function-typed value/struct-field/enum-payload callback slice-containing
  struct/enum owned/consume/inout call rejections with stable TETRA2101 JSON
  diagnostic evidence,
  ptr/slice optional assignment
  return/owned/consume/inout escape, including stable same-module/cross-module
  `TETRA2101`/`TETRA2102` CLI JSON evidence for slice optional assignment
  owned/consume/inout and return escapes, same-module/cross-module slice
  optional payload binding owned/consume/inout call, `inout` assignment, and
  global assignment escapes with stable `TETRA2101`/`TETRA2102` CLI JSON evidence,
  same-module/cross-module direct slice global assignment with stable
  `TETRA2102` JSON diagnostic evidence, same-module/cross-module optional ptr
  global assignment with stable `TETRA2102` JSON diagnostic evidence, and
  same-module/cross-module optional aggregate global assignment with stable
  TETRA2102 JSON diagnostic evidence, same-module/cross-module ptr optional
  assignment if-let/match global escape with stable TETRA2102 JSON diagnostic
  evidence, same-module/cross-module slice optional-payload inout/global assignment
  escapes with stable TETRA2102 JSON diagnostic evidence,
  same-module/cross-module nested slice enum-payload return/inout/global
  assignment escapes with stable TETRA2102 JSON diagnostic evidence,
  same-module/cross-module nested slice struct return/inout/global assignment
  escapes with stable TETRA2102 JSON diagnostic evidence,
  same-module/cross-module ptr enum alias return escape with stable TETRA2102
  JSON diagnostic evidence,
  and same-module/cross-module ptr-containing aggregate
  whole/field/alias/nested-field return escapes with stable TETRA2102 JSON
  diagnostic evidence,
  same-module/cross-module whole-aggregate global assignment with stable
  `TETRA2102` JSON diagnostic evidence,
  same-module/cross-module generic slice-containing struct/enum aggregate
  owned/consume/inout instantiations with stable `TETRA2101` CLI JSON evidence,
  local aliases returned
  directly, inside ptr-containing aggregate literals, or through ptr-containing
  struct-field or enum-payload aggregate aliases, passed as ptr-containing struct/enum aggregate arguments to
  direct
  owned/consume/inout parameters, including same-module/cross-module
  monomorphized generic aggregate parameters and optional `ptr?` generic
  owned/consume/inout instantiations with stable `TETRA2101` CLI JSON
  evidence plus same-module/cross-module generic borrow-aggregate/optional-ptr
  return diagnostics with stable `TETRA2102` CLI JSON evidence and imported direct owned/consume/inout
  call boundaries for optional ptr, struct, enum-payload, and nested
  ptr-containing aggregate arguments, including imported direct ptr-containing/nested
  aggregate owned/consume/inout call rejections with stable TETRA2101 JSON
  diagnostic evidence,
  same-module/cross-module protocol impl parameter ownership matching plus
  same-module/cross-module protocol impl parameter ownership mismatch
  diagnostics with stable TETRA2001 CLI JSON evidence, and
  same-module/cross-module generic protocol requirement parameter ownership mismatch diagnostics with stable TETRA2001 JSON diagnostic evidence,
  or function-typed callback value/struct-field/enum-payload
  owned/consume/inout parameters including same-module/cross-module
  function-typed value/struct-field/enum-payload optional `ptr?`
  owned/consume/inout arguments with stable `TETRA2101` CLI JSON evidence,
  assigned into ptr-containing `inout` aggregate
  parameters, or assigned to globals.
- Resource lifetime MVP for task handles, task groups, island handles,
  region-backed slices, optional region wrappers, and structs containing those resources. Common local
  scopes and control-flow merges are checked conservatively; branch/match/loop
  task-handle maybe-joined, task-group maybe-closed, and island maybe-freed
  merge diagnostics, branch/match/loop resource finalization merge diagnostics
  with stable `TETRA2101` JSON evidence, stable `TETRA2101` task-group
  use-after-close CLI JSON diagnostics, same-module/cross-module struct-field
  and enum-payload alias use-after-free CLI JSON diagnostics, double-use,
  same-module/cross-module nested struct-field and enum-payload optional
  resource-wrapper alias use-after-free CLI JSON diagnostics,
  same-module/cross-module task-handle/task-group struct-field/enum-payload
  join/close aliases including same-module/cross-module task-handle
  struct-field/enum-payload alias join diagnostics with stable TETRA2101 JSON
  diagnostic evidence and same-module/cross-module task-group
  struct-field/enum-payload alias close diagnostics with stable TETRA2101 JSON
  diagnostic evidence, same-module/cross-module enum-constructor return resource
  aliases with stable TETRA2101 CLI JSON evidence,
  same-module/cross-module monomorphized generic struct
  task-handle/task-group/island resource aliases with stable TETRA2101 CLI JSON evidence,
  same-module/cross-module transitive interprocedural task-handle/task-group/island
  resource aliases with stable TETRA2101 CLI JSON evidence,
  same-module/cross-module task-handle/task-group optional-payload join/close
  aliases with stable TETRA2101 CLI JSON evidence, ambiguous provenance, and ambiguous lifetime merges are diagnostics
  rather than proof obligations solved by a full SSA analysis.
- Lifetime SSA local join solver is current since `v0.4.0` for branch, match,
  and loop flow snapshots over ownership consume state, resource finalization
  state, optional region-wrapper escapes with stable `TETRA2102` diagnostics,
  same-module and interface-only cross-module per-field interprocedural region summaries for aggregate returns
  from multiple island parameters, including optional aggregate wrappers,
  enum payload wrappers, branch aggregate wrappers, match aggregate wrappers,
  if-let aggregate wrappers, mixed safe/provenance aggregate branch and match
  returns, and optional mixed safe/provenance aggregate branch merges, and
  maybe-consumed diagnostics. Broader
  interprocedural lifetime proofs,
  broad alias modeling, race proofs, and full formal lifetime guarantees remain
  under full-v1 scope.
- Mutable by-reference captures, including callable mutable-capture
  global-escape and heap-escape, callable pointer/resource capture escape,
  function-typed storage/return unsupported capture rejection, captured callable
  or function-typed parameter global-storage escape, unsupported function-value
  escape outside the fnptr ABI, capturing closure raw-ptr escape, captured
  closure explicit type-arg rejection, function-typed explicit type-arg
  rejection, unsupported function-value call, generic closure capture and
  generic callback-closure capture rejection, generic closure
  pointer/direct-call rejection, imported mutable function-typed global
  boundary diagnostics, and
  thread-boundary callable escape keep
  `stable JSON diagnostics` until a separate synchronization and
  ownership-transfer model is release-gated.
- Actor/task transfer safety MVP for local worker entrypoints, sendable scalar
  and supported structural results, handle transfer, actor/task
  use-after-transfer diagnostics with stable `TETRA2101` CLI JSON evidence,
  island transfer non-local-payload rejection with stable `TETRA2101` CLI JSON
  evidence,
  branch/match/loop actor consume reuse diagnostics with stable `TETRA2101`
  CLI JSON evidence,
  same-module/cross-module transitive actor consume alias diagnostics with stable
  `TETRA2101` CLI JSON evidence,
  same-module/cross-module monomorphized generic struct actor consume alias
  diagnostics with stable `TETRA2101` CLI JSON evidence,
  same-module/cross-module actor/task if-let/match optional-payload alias
  transfer diagnostics with stable TETRA2101 JSON diagnostic evidence,
  same-module/cross-module actor if-let/match optional-payload, struct-field, and
  enum-payload consume alias diagnostics including same-module/cross-module
  actor struct-field/enum-payload alias transfer diagnostics with stable
  TETRA2101 JSON diagnostic evidence,
  same-module/cross-module task-handle struct-field/enum-payload alias
  transfer diagnostics with stable TETRA2101 JSON diagnostic evidence plus
  same-module/cross-module task-handle struct-field/enum-payload alias join
  diagnostics with stable TETRA2101 JSON diagnostic evidence,
  release-covered cooperative `core.task_group_cancel` wake/join behavior,
  same-module/cross-module task_group_cancel return provenance diagnostics with
  stable TETRA2101 CLI JSON evidence, and task group lifecycle status/close smokes. Worker entrypoints
  are additionally checked at the
  declared effect boundary: actor/runtime scheduling effects remain allowed for
  the MVP scheduler surface, while raw memory allocation/access, capability,
  MMIO, islands, linker/control, and privacy effects are conservative
  diagnostics when present on the worker effect surface. IO and budget remain
  covered by their existing effect and budget-context checkers. For typed actor
  messages, checked ownership transfer currently applies to `island` payload
  paths; actor/task handles in typed message payloads remain outside this
  transfer contract and are rejected by the current value-only payload rule.
  This is a conservative local MVP; it does not claim distributed actor safety,
  full race-safety proofs, full cancellation semantics, or structured
  concurrency.
- Typed task handle wrappers support slot counts `2..8` on the builtin runtime
  path (`2..4` direct, `5..8` staged). `--runtime=auto` selects builtin for this
  surface, while `--runtime=selfhost` currently rejects typed task handles.
  Layouts above `8` are rejected.

## Future Or Limited

- Full `v1.0.0` language guarantees remain future work.
- The `v0.4.0` scope contract lives in `docs/spec/v0_4_scope.md`. It records the
  promoted slices and the candidates that remain experimental or reporting-only.
- Distributed EcoNet, hosted production TetraHub publishing, global trust
  scoring, hub federation, and proof-carrying capsules remain post-v1 unless
  explicitly promoted.
- `actors.distributed-runtime` is current for the Linux-x64 runtime path. The
  production claim covers the builtin Linux-x64 lowering/runtime integration
  with the `actornet` loopback TCP broker, distributed node identity, remote
  actor handles, network mailbox send/receive for i32, tagged, and typed frames,
  missing-node failure/status propagation, and compatibility with the existing
  cooperative task cancel/join handles.
- Distributed actor evidence must be executable evidence:
  `scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh` builds a fresh
  CLI, starts `tetra actor-net`, compiles and runs Linux-x64 actor nodes, writes a
  `tetra.actors.distributed-runtime.v1` report, and validates it with
  `go run ./tools/cmd/validate-distributed-actor-runtime`. A
  `tetra.actors.transport.v1` report or `validate-actor-transport` run remains
  transport-only evidence and is insufficient by itself.
- Non-Linux-x64 distributed actor targets, multi-threaded scheduling, and
  broader structured-concurrency guarantees beyond the documented cooperative
  task group handles remain outside this claim unless separately promoted.
- A full TechEmpower-compatible web stack is still broader than the current
  stable Tetra source surface: no production HTTP server, full HTTP header/body
  parser, full event-loop abstraction, io_uring path, per-core worker runtime,
  broad socket-option API, or PostgreSQL socket/database runtime is supported by the current
  `v0.4.0` profile. `lib.core.net` now provides executable linux-x64 TCP socket
  open/bind/connect/listen/accept/read/recv/write/send/nonblocking/close helpers,
  `SO_REUSEPORT` and `TCP_NODELAY` helpers, plus epoll
  create/add-read/add-read-write/mod-read/mod-read-write/delete/wait-one
  and wait-one-into readiness flag helpers, `SOCK_NONBLOCK`/`SOCK_CLOEXEC`
  accept helpers, and `EPOLLIN`/`EPOLLOUT`/`EPOLLERR`/`EPOLLHUP` predicates for
  local client/server slices.
  `lib.core.http` now provides executable HTTP/1.1 String and byte-buffer
  request-line routing, byte-buffer request-head framing for pipelined local
  server slices, and response byte-buffer helpers.
  `lib.core.json` provides executable JSON byte-buffer helpers for response
  body construction. `lib.core.postgres` now provides executable PostgreSQL
  wire-frame byte-buffer helpers for startup, simple query, extended-query
  Parse/Bind/Describe/Execute/Sync, RowDescription/DataRow/CommandComplete/ReadyForQuery
  inspection, and terminate messages. Real transport and full database APIs remain separate from
  `lib.core.networking` policy helpers and still require broader
  `lib.core.net` event-loop/socket-option expansion and `lib.core.postgres`
  driver/pool layers.
- Callable Level 1 is current since `v0.4.0`: the production claim covers
  non-capturing, symbol-backed function-typed locals, immutable aliases,
  target-set-backed aliases of function-typed parameters, callback parameters,
  function-typed parameter storage into local struct fields with direct field
  calls or synchronous callback arguments,
  function-typed parameter storage into enum payloads with direct payload
  calls, mutable local enum reassignment, returned enum propagation, or
  synchronous callback arguments,
  direct named function/closure symbols, symbol-backed
  function-typed returns, declared function-typed local initializers,
  symbol-backed same-module and namespace/selective imported public
  function-typed globals for direct calls, local function-typed
  initialization/reassignment, direct callback arguments, same-module mutable
  global initializers from non-capturing closure literals, same-module mutable
  global reassignment with direct calls or synchronous callback arguments and
  local or nested local struct-field/enum-payload storage/reassignment, and inferable
  same-module/imported generic-symbol initializers, function-typed
  returns including target-set-backed function-typed parameter returns, mutable
  local or nested struct
  field reassignment, function-typed nested struct field initializers, and enum
  payload initializers or mutable enum-payload reassignments from
  non-capturing generic closure literals or
  same-module/imported generic
  function symbols whose type parameters are fully inferred from the declared
  `fn(...) -> ...` type, callback parameter type, global type, return type,
  target type, field type, or payload type,
  mutable local or nested struct field
  reassignment from same-module or imported generic function symbols whose type
  parameters are fully inferred from the target's declared function type,
  function-typed nested struct field initializers for same-module or imported
  generic function symbols whose type parameters are fully inferred from the
  field type, enum payload initializers or mutable enum-payload reassignments
  for same-module or imported generic function symbols whose type parameters
  are fully inferred from the payload type, optional argument labels on
  function-typed value and field calls with mixed labeled/unlabeled lists
  rejected, and stable
  diagnostics for unsupported
  movement.
  Callable Level 2 is current since `v0.4.0` for captured closure
  function-typed locals backed by the nine-slot `fnptr` value. Local
  `Int`/`Bool`/`String` values and simple structs without pointer or resource
  fields may be snapshotted by value into up to eight environment slots, called
  directly, including direct calls that use labels for the explicit closure
  parameters and direct function-typed global calls that use labels as
  call-site documentation, with mixed labeled/unlabeled lists rejected against
  the user-visible callable value and explicit type arguments rejected before
  synthetic closure-symbol lowering, passed to synchronous function-typed
  callback parameters as
  function-typed locals or direct closure-literal arguments, including imported
  callback callees, returned through
  function-typed return paths as function-typed locals or direct
  closure-literal returns, including multiple known function return-path
  targets from direct symbols, local aliases, captured closure literals, or
  function-typed parameters propagated through direct local calls and
  synchronous callback arguments, including across imported module boundaries,
  alias let-bound captured `ptr` closure values into function-typed locals
  or reassign compatible mutable function-typed locals, store them in local
  struct fields or enum payloads, including direct closure-literal container
  initializers with module-qualified synthetic closure targets, return them
  directly from function-typed return paths, and pass them as direct
  synchronous callback arguments when
  their environment fits the eight-slot `fnptr` envelope, including through
  imported function-typed parameter-return helpers such as `identity(cb) -> cb`;
  imported returns that ignore a captured callback and return a concrete symbol
  do not inherit the ignored argument's captures,
  assigned into mutable
  function-typed locals, and reassigned through supported mutable local,
  struct-field, or enum-payload paths, and
  stored in immutable local struct fields or enum payloads, including direct
  closure-literal initializers. Larger safe immutable environments are handled
  by the v0.4.0 full-callable 4-slot handle path for aliases, callback
  passing, function-typed returns, local or same-module global storage, struct
  fields, enum payloads, and cross-module returned values, with generated
  `.t4i` interface-only stubs preserving the corresponding returned direct enum
  or aggregate payload metadata for API-only validation. Heap/global escape for
  safe by-value callable captures is now covered by the full first-class
  callable handle model. By-reference mutable capture, pointer/resource
  capture, unsupported dynamic callable movement, imported mutable global-data
  escape, and thread escape without a synchronization/ownership transfer model
  are rejected with stable diagnostics. Throwing function values are supported
  in the explicitly declared direct-try, callback, return, local/global, and
  aggregate paths described above; broader throwing callable movement remains a
  diagnostic boundary. Imported mutable
  function-typed globals
  that require cross-module global-data ABI are rejected with stable
  diagnostics, and actor/task workers that dispatch through same-module mutable
  function-typed globals, imported immutable function-typed globals whose
  targets touch mutable globals, imported direct function-typed return-call
  callback arguments, or imported returned struct/enum aggregate fields or
  payloads whose known targets touch mutable globals are treated as touching
  mutable global state before they can cross the boundary. Direct captured
  closure literals, let-bound captured `ptr` closure locals, direct
  same-module/imported function-typed return calls, immutable local aliases
  initialized from those return calls, mutable function-typed locals, local
  struct fields, local enum payloads, whole local or nested structs with
  function fields reassigned from struct literals containing direct closure
  literals or direct return calls, whole local enums reassigned from enum
  constructors containing direct closure literals or direct return calls, or
  same-module or source-imported returned enum payloads or returned struct enum
  payloads carrying direct closure literals, or
  generated `.t4i` interface-only stubs preserving the corresponding returned
  direct enum or aggregate payload metadata for API-only validation, or
  return alias chains that return captured closure snapshots assigned into same-module mutable global
  function-typed values are stored as bounded
  by-value `fnptr` snapshots and may be called later through that global, passed as synchronous
  callback arguments, returned from same-module or imported functions, passed as callback arguments or reassigned into mutable locals after cross-module returns, stored
  in local struct fields or enum payloads, or dispatched through `try cb(...)`
  when the global type declares the same throws type. Captured `fnptr`
  values reached through mutable function-typed whole-struct reassignments not
  backed by direct closure or direct return-call field initializers,
  unsupported assignment sources, or parameter escapes
  remain outside the production claim until broader heap/lifetime evidence is
  available. Function-typed parameters also cannot be stored into
  mutable global function-typed values in the current profile and report a
  dedicated parameter-to-global escape diagnostic, including when the parameter
  is first routed through a local alias, mutable local reassignment, direct
  same-module or imported function-typed return call, helper return alias,
  helper struct-field return, local struct field, enum payload binding,
  same-module returned struct field, same-module or imported returned nested
  struct field path, same-module or imported whole struct-parameter return, or
  same-module or imported whole enum-parameter return, or same-module or
  imported returned enum payload, and captured
  values passed through direct, inline, imported source, or generated `.t4i`
  interface-only function-typed parameter-return calls such as
  `identity(f) -> f`, through direct
  same-module, imported source, or generated `.t4i` interface-only
  struct-parameter field returns such as `pick(holder) -> holder.cb` and nested paths such as
  `pick(box) -> box.holder.cb`, same-module, imported source, or generated
  `.t4i` interface-only whole struct-parameter returns such as
  `echo(box) -> box` that preserve nested function-field target sets, same-module or imported enum-parameter payload
  returns or whole enum-parameter returns such as `echo(choice) -> choice`,
  returns, including inline imported struct/enum constructors carrying captured
  closure literals, with those returned captured `fnptr` values usable for
  local direct calls or direct synchronous callback arguments, function-typed returns from local struct-field aliases or reassignments,
  enum-payload bindings or reassignments, or through returned struct fields
  including nested paths and enum payloads built from
  function-typed parameters, local aliases of those parameters, or local
  struct-field aliases carrying those parameters, including returned structs
  such as `makeBox(f) -> Box(choice: MaybeCallback.some(f))`, are rejected at
  the global assignment boundary.
  Direct `ptr` closure calls reject mutable captures with a stable diagnostic
  because that path would observe mutable locals by reference; use an explicit
  function-typed `fnptr` binding for the supported by-value snapshot model.
  Captured callback arguments, including direct closure-literal callback
  arguments and captured `ptr` aliases, use the 4-slot handle path when the
  environment exceeds the eight-slot `fnptr` envelope and the captures are safe
  by-value captures. Generated `.t4i` stubs for direct returned function values
  preserve capture count, heap escape kind, handle flag, function target
  identity, and 4-slot return handle metadata for that handle slice. Mutable
  captures and pointer/resource captures that would escape through heap/global
  callable handles report stable diagnostics; thread-boundary mutable/resource
  escape diagnostics are fixed at the classifier boundary until a source-level
  function-value-to-thread transfer surface exists.
  Captured closure initializers and reassignments for function-typed local
  storage, struct fields, and enum payloads use the `fnptr` fast path for
  bounded environments and the 4-slot handle path for larger safe immutable
  by-value environments.
- Function-typed struct fields support the current safe callable model: local
  struct values may store non-capturing symbol-backed function values, captured
  `fnptr` values with up to eight environment slots, or handle-backed larger
  immutable by-value captured values, and call them directly through
  `value.field(...)`, alias them into function-typed locals, or pass them as
  supported callback arguments. These field values may also be
  initialized from function-typed parameters when call-site target sets are
  known, with subsequent direct field calls or synchronous callback arguments
  dispatching over those propagated targets; cache dependency collection treats
  those field calls as callable storage, not external function symbols. Direct
  field calls enforce positional function-type ownership markers with the same
  borrow/consume/inout aliasing and mutability diagnostics as local callback
  calls. They
  may also be initialized from direct closure literals, other immutable symbol-backed struct
  fields, symbol-backed enum payload bindings, or from known function-typed
  returns with stable targets or target-set-backed function-typed
  parameter-return calls such as `Holder(cb: identity(captured))` or
  `Holder(cb: callbacks.identity(captured))`, including multi-target return target sets with mutable-global-target classification,
  returned from function-typed return paths,
  preserved through known struct returns that carry stable function-field
  metadata, including after local struct field reassignment before return, and
  through nested struct literal initializers such as `Box(holder: makeHolder())`.
  Known struct returns may collect multiple function-field targets across return
  paths and preserve them for subsequent direct field calls or synchronous
  callback arguments.
  They may be reassigned on mutable local structs from supported named
  functions, closure literals, known function-typed returns, or
  target-set-backed parameter-return calls such as `holder.cb = identity(captured)`,
  including imported forms such as `holder.cb = callbacks.identity(captured)`,
  and nested local field paths such as
  `box.holder.cb = callbacks.identity(captured)`,
  with dynamic dispatch over known target sets, including subsequent local struct-field
  snapshots, whole-struct local aliases that
  preserve function-field metadata, whole-struct local reassignments such as
  `holder = Holder(cb: callbacks.identity(captured))`, struct-valued field
  reassignments such as `box.holder = Holder(cb: callbacks.identity(captured))`,
  whole nested-struct reassignments such as
  `box = Box(holder: Holder(cb: callbacks.identity(captured)))`,
  local function aliases, and synchronous
  callback arguments. Direct calls, reassignment, and callback arguments may use
  nested local struct field paths such as `box.holder.cb`.
  Semantic-clause diagnostics for direct field calls use the visible field path.
- Function-typed enum payloads support the current safe callable model:
  immutable local enum values constructed with non-capturing symbol-backed
  function values, captured `fnptr` values with up to eight environment slots,
  or handle-backed larger immutable by-value captured values may bind the
  payload in `match`, call it directly, alias it into function-typed locals, or
  pass it as a supported callback argument. Whole-enum local aliases preserve
  function-payload metadata before pattern binding. These payloads may also be
  initialized from function-typed parameters when call-site target sets are
  known; those targets propagate through local constructor bindings, mutable
  local enum reassignment, returned enum values, direct payload calls, and
  synchronous callback arguments. They may also be
  initialized from direct closure literals, immutable symbol-backed struct
  fields or symbol-backed enum payload bindings, or known function-typed
  returns with stable targets or target-set-backed function-typed
  parameter-return calls such as `MaybeCallback.some(identity(captured))` or
  `MaybeCallback.some(callbacks.identity(captured))`, including multi-target return target sets with mutable-global-target classification,
  preserved through known enum returns carrying
  stable function-payload metadata for local bindings and direct
  `match makeChoice()` scrutinees, including multiple known targets collected
  across return paths and later passed through synchronous callback arguments,
  or reassigned through same-module or imported parameter-return calls while
  preserving captured metadata for direct `match` calls and global-escape
  diagnostics. That reassignment metadata is preserved for mutable enum locals
  and for enum values stored behind mutable local struct fields, including
  `box.choice = MaybeCallback.some(callbacks.identity(captured))`. Returned
  structs whose fields contain enum payloads, such as
  `makeBox(f) -> Box(choice: MaybeCallback.some(f))`, preserve payload metadata
  after call-site substitution from imported parameter-return arguments,
  including after whole-struct local reassignment such as
  `box = makeBox(callbacks.identity(captured))` and through nested returned
  struct initializers such as `makeOuter(f) -> Outer(box: makeBox(f))`.
  Returned-struct enum-payload target sets may collect multiple known
  return-path targets and dispatch direct `match box.choice` payload calls
  through the runtime-selected branch target. Payload
  bindings may be returned from function-typed return paths.
  Mutable local enum values may be reassigned from
  supported enum constructors carrying direct named functions, direct closure
  literals, known function-typed returns with stable targets including
  multi-target return target sets with mutable-global-target classification, or whole-enum aliases before a local
  `match`; multiple known branch targets dispatch
  through the same stable symbol-address target-set path used by callback
  values, including when the pattern-bound payload is passed to a synchronous
  function-typed callback parameter. Direct calls through pattern-bound payloads
  use the same positional ownership checks as callback calls and allow labels
  as call-site documentation.
  Non-symbol-backed, heap/global/thread-escaped function values in enum payloads
  remain outside the current claim.
- Supported symbol-backed struct-field and enum-payload callback paths are
  checked for cross-module callback calls; same-module enum constructors are
  treated as type construction, not external callable dependencies, for cache
  dependency hashing.
- Arbitrary callable/function-pointer semantics remain outside the current
  support claim when they require mutable by-reference capture, pointer or
  resource capture, thread-boundary escape, unsupported dynamic/generic
  callable movement, or ABI behavior beyond the v0.4.0 safe by-value callable
  handle model.
- Generic structs, explicit type arguments, higher-ranked generics, runtime
  generic values, full protocol-bound generic dispatch, calls through generic
  requirement bounds, specialization optimization, witness tables, trait objects,
  runtime protocol values, and protocol dynamic dispatch remain outside the
  current `v0.4.0` support claim unless separately promoted by a later gate.
- Advanced ADT constructors, nested destructuring patterns, richer enum payload
  algebra, and guard expansion remain future/post-v1 unless separately promoted.
- Broad formal lifetime proofs, distributed race-safety proofs, and
  synchronization-aware heap/global/thread escape analysis remain future work
  beyond the current local lifetime SSA solver.
- Effect inference, proof-level effect guarantees, broad safe-code capability
  construction, cryptographic privacy isolation, distributed consent
  enforcement, aggregate runtime-wide budget accounting beyond the static
  cross-edge guardrail, and distributed budget enforcement remain outside the
  current `v0.4.0` support claim.
- Non-Linux-x64 distributed actor targets, full async cancellation/structured
  concurrency, GTK/Qt/OS UI toolkit backends, broad native UI input/change/focus
  behavior, and platform accessibility integration remain outside the current
  `v0.4.0` support claim.
- UI metadata v1 (`ui.metadata-v1`) is promoted for the `v0.4.0` metadata
  contract: checked state/view declarations, deterministic `tetra.ui.v1` JSON,
  wasm32-web command-dispatch preview sidecars for lowered scalar state
  operations, and native shell command-dispatch text plus
  `tetra.ui.native-shell.v1` JSON trace sidecars for lowered scalar state
  operations and deterministic native shell widget-tree artifacts, including
  direct assignment and integer increment/decrement updates, including
  supported `+=`/`-=` compound assignments, with scalar assignment hydration
  and same-state field-copy assignment in command order.
  The web preview mirrors supported style and accessibility metadata into DOM
  preview attributes, but this is not platform-native widgets, a full
  styling/layout engine, platform accessibility API integration, or `v1.0.0`
  readiness without the full release gate.
- UI native runtime (`ui.native-runtime`) is promoted only for the Linux-x64
  production slice. The release gate runs
  `bash scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh`, which builds the
  current `tetra` CLI, builds `examples/ui_native_shell_smoke.tetra` for
  `linux-x64`, runs the resulting native executable, loads the checked
  `tetra.ui.native-shell.v1` sidecar as runtime input, creates native runtime
  widget instances with stable IDs, hierarchy, bounds, text/value, enabled, and
  visible state, dispatches click events through lowered command operations,
  records before/after state and widget updates, covers runtime close, and
  writes `reports/v0.4.0/native-ui-linux-x64.json` using the strict
  `tetra.ui.native-runtime.v1` schema. The validator rejects metadata-only,
  web-only, native-shell sidecar-only, fake/mock/placeholder, missing event
  execution, and missing state-transition evidence.
- WASM runtime execution is supported when the required host runner is
  discoverable. `wasm32-wasi` uses `run_mode: "wasi_runner"` and runs through
  `wasmtime` or the Node WASI fallback. `wasm32-web` uses
  `run_mode: "web_runner"` and runs through a discovered Chromium-compatible
  browser runner. Browser automation is also the required Web UI runtime smoke
  evidence when it instantiates real WASM, mounts DOM from `tetra.ui.v1`, and
  passes `go run ./tools/cmd/validate-web-ui-smoke`. Missing runners are explicit environment blockers in
  `targets --format=json`; Linux-x64 native UI runtime evidence remains a
  separate `ui.native-runtime` release artifact and does not promote
  macOS/Windows native UI claims.
- Build-only Linux x86/x32 target metadata uses `run_mode: "host_probed"`:
  `run_supported` is true only when the current host can execute that exact
  ABI (`i386` compatibility for x86, Linux x32 ABI support for x32), and
  false results must carry an explicit no-host-fallback
  `run_unsupported_reason`. Their broader runtime/stdlib/FFI limitations remain
  in `unsupported_reason`.
- Any feature labeled `planned`, `beta`, `deferred-post-v1`, or
  `blocked-by-prerequisite` in release docs must not be marketed as stable.

Language note:
- Source-language `capsule ...` declarations are not Eco package manifests.
  Eco packaging still uses project manifest files (`Capsule.t4`,
  `Tetra.capsule`) and corresponding `tetra eco` workflows.

## Patch-Line Rule

`v0.4.x` releases are allowed to clean, stabilize, document, and harden the
current profile. Breaking language or project compatibility changes belong in a
later `x.0.0` line, and large feature updates belong in a later `0.x.0` line.
