# Truthful Intent Architecture

Status: v0 implementation slice.

Tetra has one safe-program truth. Build flags may select a target or emit
diagnostics, reports, and instrumentation, but they must not downgrade safe
semantics.

The compiler pipeline is staged as:

```text
Source -> AST -> Typed AST -> PLIR -> stack IR / future SSA -> backend -> reports
```

Current implementation anchors:

- Safe `[]T` and `String` metadata assignment is rejected centrally in
  `compiler/internal/semantics/resolution.go`.
- PLIR v0 lives in `compiler/internal/plir` and records provenance, lifetime,
  region, borrow, range, CFG/dominance, proof guard/use, and allocation-intent
  facts for the supported slice loop, while-loop, branch-guard, view-chain, and
  copy-loop bounds proofs.
- Allocation planning v0 lives in `compiler/internal/allocplan`; it classifies
  no-escape, returned, unknown-call, unsafe, and explicit-island allocation
  intents before storage-aware lowering exists. P2.0 reports stable site ids,
  planned storage, actual lowering storage, validation status, and lowering
  status so later storage-changing slices are reviewable. Small scalar structs
  already lower as ordinary value slots, and fixed-array headers remain checked
  memory-backed views unless a later field-sensitive slice proves otherwise.
  P2.1 enables target-aware x64 stack lowering for the fixed small no-escape
  local slice subset; P2.2 keeps borrowed local views allocation-free and
  stack-lowers fixed non-escaping local-view copies; P2.3 scalar-replaces tiny
  fixed local slices whose indexed uses are all constant and in range; P2.4
  hardens explicit island storage by requiring `actual_lowering_storage:
  ExplicitIsland` reports to match lowered `IRIslandMakeSlice*` evidence; P2.5
  actually eliminates unused copy allocation intents and keeps `copy_into(dst)`
  out of allocation reports; P2.6 validates stack-backed lowered IR for
  return/call/global escape paths and rejects allocation report mismatches.
  P4.4 expands escape analysis with explicit global, task, actor, closure, and
  aggregate classifications; metadata-only reads such as `.len` do not count as
  value escape. P4.5 extends scalar replacement to tiny fixed `copy()`-created
  owned buffers when every use is a direct constant in-range index; dynamic
  indices, aliases, raw `.ptr` exposure, unknown calls, and escaping uses fall
  back to checked stack or heap storage. Copy-source initialization still uses
  normal checked loads, so this storage optimization does not invent a bounds
  proof. P5.0 freezes the runtime allocation contract in
  `compiler/internal/runtimeabi` and
  `docs/design/runtime_allocation_contract.md`: allocator API names, alignment,
  zero-size behavior, invalid-size guards, failure behavior, debug
  instrumentation, and report hooks are now explicit. P5.1 implements the
  first `linux-x64` fast heap runtime path for safe `make_*` slices: small
  non-empty requests call a shared bump helper backed by a 64 KiB `mmap` chunk,
  large safe-slice requests use the helper's `mmap` fallback, and allocation
  rows expose runtime path, allocator class, requested bytes, and reserved
  bytes for constant sizes. P15.2 upgrades that report contract to
  `per_core_small_heap` rows with allocator scope, chunk size, and
  same-core same-size-class free-list reuse policy evidence. P5.2 hardens explicit island regions:
  island size
  guards run before host allocator entry, each island slice bump is 16-byte
  aligned before capacity commit, PLIR ties the allocation row to the island
  handle region, and reports expose region id, lifetime, and debug-mode hooks.
  P5.3 adds the first implicit region planning row for non-escaping temporary
  copies that do not already fit the fixed-small stack subset: reports can say
  `planned_storage: Region` and name the function-temp region, while also
  saying the current backend still lowers that row through heap fallback. P5.4
  upgrades allocation reports to schema v2 with an exact summary: allocation
  count, planned-storage counts, actual-lowering counts, runtime-path counts,
  allocator class/scope/reuse-policy counts, raw-pointer bounds status counts,
  raw-slice policy counts, requested bytes, reserved bytes, and per-region
  summaries are validated against the plan before emission. P15.3 records
  `core.alloc_bytes` allocation-base metadata and derived `ptr_add` roots while
  keeping arbitrary raw pointers checked external/unknown.
  Cross-stage validation checks that report `actual_lowering_storage` matches
  lowered IR.
- Machine IR v0 lives in `compiler/internal/machine`; it defines virtual
  registers, blocks, branches, calls, liveness, live intervals, and linear scan
  allocation for the future register backend. P3.0 freezes the readiness
  contract in `docs/design/machine_ir_register_backend.md`, including verifier
  checks for undefined vregs, block terminators, branch targets, call ABI
  clobbers, spill bounds, and impossible physical-register assignments. P3.1
  selects a scalar Machine IR path for eligible integer-only functions on x64
  while leaving unsupported functions on the stack backend. P3.2 extends that
  path to the canonical `while i < n` integer accumulation loop. P3.3 adds the
  first proof-gated memory hot path for `while i < xs.len` `[]i32` sums: only a
  `proof:while:` unchecked index load can select the register slice-sum path,
  while checked/raw/unknown-provenance forms stay on the stack backend with
  runtime bounds checks. P3.4 adds scalar call lowering for direct/nested calls
  and the canonical call-in-loop form, carrying target ABI and caller-saved
  clobber metadata in Machine IR; multi-slot slice/String returns remain on the
  stack backend until their representation is separately verified. Backend
  reports expose Machine IR dump, liveness, interval, allocation, spill-slot
  evidence, and P3.5 `backend_path` rows showing `register` versus `stack`
  fallback per function. Those rows are emitted only by explain reports. The
  backend has an internal test-only switch to compare register paths against
  stack fallback; it is not a user semantic mode.
- Cross-stage validation v0 lives in `compiler/internal/validation`; it names
  the verifier map, checks proof-tagged removed bounds checks against live PLIR
  proof guards, validates allocation plans, verifies stack-lowered allocation
  headers do not escape through lowered IR, and provides a conservative
  translation-validation hook. P4.7 strengthens that hook into a v1 validation
  layer: function sets, call signatures, policies, and proof-fact multisets must
  be preserved; simple straight-line algebra is checked with local symbolic
  equivalence; and supported small stack-IR functions run through deterministic
  differential samples. Allocation report generation checks the emitted report
  against the exact plan before writing diagnostics.
- Optimization pass management v0 lives in `compiler/internal/opt`. Each pass
  declares its input/output IR kind, required/preserved/invalidated facts,
  validation strategy, and report output. The manager records before/after
  stack-IR dumps, runs the IR verifier before and after every pass, checks
  bounds proof consistency, and can run one named pass for tests without
  exposing a user semantic mode. Passes that request translation validation
  also run the structural `validation.ValidateTranslation` hook. P4.1 adds the
  internal `basic-scalar` pass for straight-line stack IR: safe constant
  folding, local copy propagation, simple dead-store cleanup, and algebraic
  identities such as `x + 0` and `x * 1`. Integer arithmetic folds only when
  the result is representable as i32, so unspecified overflow behavior is never
  used as an optimization assumption. P4.2 adds `inline-small-pure`, which
  inlines only tiny scalar, straight-line, side-effect-free functions. Callees
  containing calls, unsupported effects, control flow, multi-slot returns, or
  proof-sensitive instructions are left as calls and recorded as `not_inlined`
  decisions. The pass allocates fresh caller local slots for inlined callee
  locals, preserves function definitions for translation validation, and does
  not remove or move proof-tagged bounds-check evidence. P4.6 adds the internal
  `loop-canonicalization` pass for simple stack-IR loops with live
  `proof:while:` evidence: it snapshots a stable loop length local in the
  preheader, rewrites loop-local length loads to that snapshot, and canonicalizes
  `i <= len - 1` guards to `i < len`. The pass refuses loops without a while
  bounds proof, loops that store to the length local, and loops containing calls,
  index stores, memory writes, allocation/view construction, or other unknown
  mutation surfaces, preserving proof dominance by leaving proof-tagged loads in
  the dominated loop body.
- `compiler/internal/lower` runs PLIR verification before stack IR lowering.
- `for x in xs`, supported `while i < xs.len` / `while i <= xs.len - 1`
  patterns, dominating `if` guards with a known non-negative index, safe
  prefix/suffix/window view chains, and `copy()` / `copy_into(dst)` source
  loops use proof-tagged unchecked index loads only when the compiler has a
  dominating range proof. Simple safe aliases may carry those proofs; raw,
  unknown, or statically invalid aliases remain conservative across branch
  joins and keep checked index access.
- External indexing remains checked.
- `--explain`, `--emit-plir`, `--emit-proof`, `--emit-bounds-report`, and
  `--emit-alloc-report` only emit artifacts.
- Benchmark evidence starts at `tools/cmd/truth-bench-harness`; the P8/P20
  harness validates the required Tetra/C/C++/Rust matrix rows, records compiler
  versions, target CPU, binary size, raw output artifacts, and Tetra
  proof/allocation/bounds/performance report artifacts, and rejects broad or
  official performance claims that lack matching report evidence. Compiler
  `.perf.json` schema v3 reports also list the P20.1 blocker reasons: missing
  dominance, return escape, unknown call allocation, missing noalias proof,
  code-size inlining budget, register live-range pressure, unsupported
  aggregate-return stack fallback, and borrowed data crossing actor boundaries.
- P9 layout/specialization freedom is explicit metadata, not a hidden ABI
  promise. Plain structs keep the default Tetra representation and may receive
  future layout transforms only through `compiler/internal/layoutopt` policy;
  `repr(C)` structs are ABI-locked and deny field reordering, packing,
  hot/cold splitting, scalar replacement, and AoS-to-SoA transforms. P21.0
  promotes that boundary into `.layout.json` schema version 2 with
  `p21.0_default_layout_freedom_v1` decision rows and validator rejection for
  fake layout freedoms; exported public ABI aggregate boundaries require
  explicit `repr(C)`. P21.1 adds ABI verification report schema
  `tetra.abi.verification.v1` / `p21.1_abi_verification` for linux-x64,
  linux-x86, linux-x32, macos-x64, windows-x64, wasm32-wasi, and wasm32-web,
  with native classifier/FFI evidence and wasm compiler-owned i32 slot ABI
  call-boundary evidence; it does not claim runtime execution for build-only or
  wasm targets, default-struct C ABI, wasm native C aggregate ABI, performance,
  or a safe-semantics change. Generic calls are monomorphized first; the current
  concrete optimization evidence is the small-pure inline path that can erase
  identity/wrapper calls without erasing proof, provenance, or ABI facts. P21.2
  records that boundary as `tetra.optimizer.specialization_machine_code.v1` /
  `p21.2_specialization_v1_v2`: a translation-validated direct-helper witness
  disappears from optimized Stack IR and lowers to verified scalar Machine IR
  without `OpCall`, while rows connect the evidence to generics,
  protocol/static conformance, extension methods, enum known cases, optionals,
  and P19.1 caller-owned collection helpers. This does not claim broad
  specialization, dynamic protocol dispatch removal, runtime generic values,
  allocator-backed generic collections, layout/ABI freedom, performance, or
  safe-semantics changes. P22.0 adds
  `tetra.language.feature_surface_audit.v1` /
  `p22.0_full_feature_surface_audit`, a registry-backed audit of first-class
  callables, closures, protocols/trait objects, runtime generics, advanced
  enums/pattern matching, async typed errors, structured concurrency,
  modules/packages, macros/metaprogramming, UI/surface, and Eco/capsules. It
  copies `FeatureRegistry()` statuses and rejects fake same-branch promotions,
  but it does not promote P22.1/P22.2 work, runtime generic values, trait
  objects, macro systems, full structured concurrency, cross-platform
  production UI runtime, distributed EcoNet, proof-carrying capsules,
  performance, runtime behavior, or safe-semantics changes. P22.1 records
  first-class callable evidence as `tetra.language.first_class_callables.v1` /
  `p22.1_first_class_callables_v1`: live witnesses parse, check, and lower a
  one-capture 9-slot `fnptr` value without heap environment allocation and a
  nine-capture fixed 4-slot handle with one `IRAllocBytes`, nine
  `IRMemWritePtrOffset` writes, nine `IRMemReadPtrOffset` reads, call arg/ret
  slots `10/1`, and generated `.t4i` return metadata preserving
  `ReturnFunctionHandleValue`, heap escape kind, capture count, target
  identity, and `ReturnSlots = 4`. The report covers the bounded `fnptr` fast
  path, fat handle, capture safety classifier, mutable/resource/thread
  diagnostics, fixed ABI width, cross-module metadata, and storage/callback
  paths, while rejecting variable-width ABI, exploding return slots, mutable
  by-reference capture support, pointer/resource capture support,
  thread-boundary callable transfer, runtime generic callable polymorphism,
  dynamic callable dispatch, unsafe lifetime relaxation, performance, runtime
  behavior, and safe-semantics claims. P22.2 records the protocol /
  trait-object decision as
  `tetra.language.protocol_trait_object_decision.v1` /
  `p22.2_protocol_trait_object_decision`, with decision
  `keep_static_conformance_only`. Its witnesses parse, check, and lower a
  static protocol impl direct `Vec2.draw` `IRCall`, a protocol-bound concrete
  `id__T_Vec2` direct call, runtime protocol value rejection with
  `unknown type 'Drawable'`, generic-bound requirement-call rejection, and
  P17/P21 known-direct specialization evidence. Runtime protocol values, trait
  objects, witness tables, dynamic dispatch, conformance-table lookup, runtime
  existential ABI, broad protocol specialization, performance, runtime
  behavior, and safe-semantics claims remain rejected unless a future
  same-branch ABI/lifetime design promotes them. PLIR effect facts
  are emitted only from checked declared effects and mutable-global analysis.
- P11 verified-track evidence makes the long-term proof path concrete without
  changing user semantics. `compiler/internal/differential` compares source
  interpreter, stack backend, register backend, and optimized backend results
  for the stable scalar-i32 subset. `compiler/internal/validation` emits
  machine-checkable translation-validation metadata with sha256 before/after IR
  hashes. `compiler/internal/selfhostgate` blocks self-hosting claims until the
  register backend, optimizer, allocator, and stdlib evidence are present.
  `compiler/internal/formalcore` keeps the formal core intentionally small:
  values, provenance, regions, borrow/copy, bounds proofs, allocation length
  contracts, allocation intent, raw pointer bounds metadata, and
  check-elimination validity. None of these paths introduce a public semantic
  backend/source mode or a full formal-language claim.
- P23.0 translation validation v2 records
  `tetra.translation.validation.v2` / `p23.0_translation_validation_v2` rows
  for registered optimizer pass coverage, symbolic scalar equivalence,
  supported i32 slice memory equivalence, bounds proof preservation,
  allocation plan preservation, and machine-checkable sha256 before/after
  optimization metadata. The witnesses run the real optimizer manager,
  validation, differential, allocation-lowering, and metadata builders. This
  remains supported-subset evidence, not a full formal proof, broad memory
  model, broad loop theorem prover, performance claim, runtime behavior change,
  or safe-semantics change.
- P23.1 fuzz/property/differential expansion records
  `tetra.fuzz.property.differential.v1` /
  `p23.1_fuzz_property_differential` rows for generated parser/checker cases,
  PLIR/lowering verifier cases, backend matrix randomized samples,
  host-supported Linux x64 native differential evidence or an explicit
  unavailable boundary, runtime allocator properties, actor-transfer stress
  diagnostics, fuzz nightly summary artifacts, and reduced mismatch
  reproducers. It is robustness evidence, not exhaustive fuzzing, a full
  program-correctness proof, a full native differential suite, a performance
  claim, runtime behavior change, or safe-semantics change.
- P23.2 formal core v1 records `tetra.formal_core.v1` /
  `p23.2_formal_core_v1` rows for values, borrows and owned/copy, provenance
  and regions, bounds proof id semantics, allocation length contracts,
  allocation intent lowering, raw pointer bounds metadata, and
  check-elimination validity. The witnesses run `formalcore.ValidateSpec`,
  `differential.CheckBackendMatrix`, `plir.VerifyProgram`,
  `validation.CheckBoundsProofsWithPLIR`, `validation.ValidateAllocationLowering`,
  `allocplan.FromPLIR`, and `runtimeabi` raw-bounds helpers. This remains small
  internal evidence, not a full formal proof, broad language theorem prover,
  unsafe-policy change, runtime behavior change, safe-semantics change, or
  performance claim.
- P23.3 self-hosting gate v1 records `tetra.self_hosting.gate.v1` /
  `p23.3_self_hosting_gate` rows for self-host subset definition, small compiler
  component compile boundary, Go compiler output vs Tetra-compiled output
  comparison boundary, register backend stability, optimizer validation
  maturity, allocator/runtime stability, stdlib sufficiency, deterministic
  bootstrap chain, cross-platform bootstrap story, and no self-hosting claim.
  It reuses `selfhostgate.Evaluate`, `differential.CheckBackendMatrix`,
  `BuildP23TranslationValidationV2`, `runtimeabi`, and
  `stdlibrt.RegionAwareStdlibCoverage`. The current report requires
  `SelfHostingClaimed=false` and `GateDecision.Allowed=false`; it is not a
  self-hosting claim, deterministic bootstrap chain, cross-platform bootstrap
  story, runtime behavior change, safe-semantics change, or performance claim.
- P24.0 security review gate v1 records `tetra.security.review_gate.v1` /
  `p24.0_security_review_gate` rows for unsafe API surface, capability surface,
  memory allocator, network runtime, actor runtime, DB protocol, package/Eco
  system, build scripts, supply chain, and required security review artifacts.
  It reuses `runtimeabi.RuntimeAllocationContracts`,
  `runtimeabi.RuntimeRawPointerBoundsABI`, `netrt.IOReactorCoverage`,
  `actorsrt.ActorRuntimeProductionBoundaryAudit`,
  `pgrt.ProductionPostgresCoverage`, Eco validator path checks, release
  security-review script checks, and artifact presence checks. This is
  current-branch audit evidence, not security certification, external
  penetration testing, CVE-free status, release signoff, runtime behavior
  change, safe-semantics change, or performance claim.
- P24.1 runtime hardening v1 records `tetra.runtime.hardening.v1` /
  `p24.1_runtime_hardening` rows for deterministic traps, OOM policy, stack
  overflow guard boundary, integer overflow semantics audit, allocator
  corruption detection instrumentation, region double-free/use-after-free
  instrumentation, actor mailbox overflow policy, and network parser limits.
  It reuses `runtimeabi.RuntimeAllocationContracts`,
  `runtimeabi.RuntimeRegionAllocatorConfig`,
  `runtimeabi.RuntimePerCoreSmallHeapABI`,
  `runtimeabi.NewPerCoreSmallHeapAllocator`, `parallelrt.NewTypedMailbox`,
  `actorsrt.ActorRuntimeProductionBoundaryAudit`, `httprt.ParseRequest`,
  `httprt.ParseRequestView`, `pgrt.ReadFrame`, backend trap/stack-depth file
  checks, and optimizer overflow-semantics file checks. This is current-branch
  runtime-hardening audit evidence, not a full runtime-hardening proof, full
  stack-overflow protection, OOM recovery guarantee, production actor-mailbox
  promotion, runtime behavior change, safe-semantics change, or performance
  claim.
- P24.2 compatibility/stability v1 records
  `tetra.compatibility.stability.v1` /
  `p24.2_compatibility_stability` rows for stable diagnostic codes,
  versioned report schemas, manifest compatibility checks, breaking-change
  migration guide, and deprecation policy. It reuses
  `DiagnosticCodeRegistry`, `tools/cmd/validate-diagnostic`, P21-P24 schema
  constants, `tools/cmd/validate-manifest`, `docs/generated/manifest.json`,
  `docs/spec/api_diff_policy.md`,
  `docs/release/breaking-change-migration-guide.md`,
  `docs/release/deprecation_policy.md`,
  `docs/release/v1_0_x_maintenance_policy.md`, and
  `docs/spec/stdlib_naming_versioning.md`. This is current-branch
  compatibility evidence, not a full backward compatibility guarantee,
  diagnostic-message freeze, automatic migration promise, manifest/runtime ABI
  stability promise, runtime behavior change, safe-semantics change, or
  performance claim.

## Safe View Lifetime Contracts v1

The safe view lifetime layer keeps the same architecture rule: diagnostics and
reports reveal compiler facts, but never loosen safe semantics. Borrowed
returns (`-> borrow []u8`, `-> borrow []u16`, `-> borrow []i32`,
`-> borrow []bool`, and `-> borrow String`) are parsed into function
signatures, preserved through generated interfaces, and checked before PLIR
lowering. A borrowed return must come from one safe source; ambiguous branch
sources, local owned allocation, and unsafe unknown provenance are diagnostics.

The escape checker treats direct borrowed values and aggregates containing
borrowed views uniformly. Owned returns, global storage, actor sends, task
typed-transfer surfaces, closure escape, consume parameters, and unknown
escaping call positions all require an explicit copy. The current task runtime
does not expose a general task payload/capture API; v1 therefore enforces the
available typed task result/error transfer surface and documents that future
payload syntax will reuse the same boundary checker.

Proof and allocation reports expose the distinction between no-allocation
borrowed views, borrowed returns, owned copies, and copy-into transfers. The
same invalid program must fail whether or not report flags are enabled. Stack
lowering follows those same facts: borrowed views over local stack-backed slices
do not allocate, returning a local borrow is still rejected, and escaping copies
keep owned heap storage unless a later validated storage class replaces it.
Scalar replacement follows the same discipline: it removes backing storage only
for constant in-range element accesses and falls back to checked indexed memory
for dynamic indices, alias exposure, raw pointer exposure, or unknown use paths
instead of fabricating a proof. The P4.5 copy-buffer slice copies source
elements through checked loads before storing them in scalar locals, so copied
owned buffers gain storage removal without weakening source bounds semantics.
Explicit island lowering uses the same cross-stage discipline: island-backed
views preserve island provenance and cannot escape as borrows, while `copy()`
creates a separate owned allocation that may escape without extending the
island lifetime.
`copy_into(dst)` follows the no-fresh-allocation side of the same rule: reports
show the operation in PLIR/proof evidence, not as a new allocation site, and
lowering checks the destination length before writing through the destination
view.

Forbidden directions:

- no `--unsafe-fast`
- no `--no-bounds-checks`
- no release/debug semantic split
- no removal of runtime checks without a proof id or validation path
