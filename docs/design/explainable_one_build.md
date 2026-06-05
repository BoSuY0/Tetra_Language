# Explainable One Build

Status: allocation report v2, P20.1 performance blocker report v3, and P21.0
layout report v2.

The default build keeps one safe-program truth. The explain/report flags add
evidence; they do not change language semantics.

Supported report flags:

```text
--explain
--emit-plir
--emit-proof
--emit-bounds-report
--emit-alloc-report
```

Current report artifacts use the output path as a prefix:

```text
app.plir.json
app.plir.txt
app.proof.json
app.bounds.json
app.alloc.json
app.alloc.txt
app.backend.json
app.layout.json
app.perf.json
app.explain.txt
```

Layout reports are evidence-only. Schema version 2 records the P21.0 layout
policy `p21.0_default_layout_freedom_v1`: default struct layout is
compiler-owned, `repr(C)` locks layout, public ABI/exported FFI requires
explicit `repr(C)`, and report rows show decisions. Rows use decisions such as
`compiler_owned_default` and `abi_locked_repr_c`, public ABI states such as
`exported_ffi_explicit_repr_c`, and transform names `field_reordering`,
`padding_removal`, `hot_cold_splitting`, `scalar_replacement`, and
`aos_to_soa`. `ValidateLayoutReport` rejects fake layout freedoms, `repr(C)`
rows that allow transforms, default rows that claim ABI locks, exported ABI
rows missing explicit `repr(C)`, and placeholder evidence. The report does not
perform layout transforms, change runtime behavior, or make a performance
claim.

ABI verification v1 is also evidence-only. The report schema
`tetra.abi.verification.v1` with scope `p21.1_abi_verification` records target
rows for linux-x64, linux-x86, linux-x32, macos-x64, windows-x64,
wasm32-wasi, and wasm32-web, plus task rows for the ABI test corpus,
struct/enum/slice/String return validation, call boundary validation, and FFI
`repr(C)` tests. Native rows point at classifier/object/FFI diagnostics; wasm
rows point at compiler-owned i32 slot ABI metadata and backend call arg/return
slot validation. This report does not claim runtime execution for build-only or
wasm targets, C ABI for default structs, native C aggregate ABI for wasm,
performance, or safe-program semantic changes.

Specialization machine-code evidence v1 is evidence-only as well. The report
schema `tetra.optimizer.specialization_machine_code.v1` with scope
`p21.2_specialization_v1_v2` records rows for generics, protocol/static
conformance, extension methods, enum known cases, optionals, and collections.
Its machine witness runs `inline-small-pure`, verifies translation, lowers the
optimized direct-helper path through `machine.ScalarIntFunctionFromStackIR`,
and records verified scalar Machine IR without `OpCall`. The report connects
that witness to P17.2 and P19.1 source evidence, but it does not claim broad
specialization, performance, dynamic dispatch removal, runtime generic values,
allocator-backed generic collections, layout/ABI freedom, runtime behavior
changes, or safe-program semantic changes.

Full feature surface audit v1 is also evidence-only. The report schema
`tetra.language.feature_surface_audit.v1` with scope
`p22.0_full_feature_surface_audit` records rows for first-class callables,
closures, protocols/trait objects, runtime generics, advanced enums/pattern
matching, async typed errors, structured concurrency, modules/packages,
macros/metaprogramming, UI/surface, and Eco/capsules. Rows copy
`FeatureRegistry()` statuses so validators can reject drift, placeholders, and
fake same-branch promotions. The report does not claim full v1 language
guarantees, runtime generic values, trait objects, macro systems, full
structured concurrency, cross-platform production UI runtime, distributed
EcoNet, proof-carrying capsules, performance, runtime behavior changes, or
safe-program semantic changes.

First-class callables v1 is evidence-only too. The report schema
`tetra.language.first_class_callables.v1` with scope
`p22.1_first_class_callables_v1` records rows for the bounded `fnptr` fast
path, fat callable handle, capture safety classifier, mutable capture escape
diagnostics, resource/thread escape diagnostics, fixed ABI width, cross-module
interface metadata, and storage/callback paths. Its witnesses parse, check,
and lower a one-capture 9-slot `fnptr` value without heap environment
allocation and a nine-capture fixed 4-slot handle value with one
`IRAllocBytes`, nine `IRMemWritePtrOffset` writes, nine `IRMemReadPtrOffset`
reads, and call arg/ret slots `10/1`; generated `.t4i` metadata is checked for
`ReturnFunctionHandleValue`, heap escape kind, capture count, target identity,
and `ReturnSlots = 4`. The report rejects placeholders and fake claims for
variable-width callable ABI, exploding return slots, mutable by-reference
capture support, pointer/resource capture support, thread-boundary callable
transfer, runtime generic callable polymorphism, dynamic callable dispatch,
unsafe lifetime relaxation, performance, runtime behavior changes, or
safe-program semantic changes.

Protocol / trait-object decision v1 is also evidence-only. The report schema
`tetra.language.protocol_trait_object_decision.v1` with scope
`p22.2_protocol_trait_object_decision` records decision
`keep_static_conformance_only`. Rows cover the static conformance fast path,
static protocol-bound generics, runtime existential decision, explicit
dynamic-dispatch gate, specialization static abstraction, witness-table
boundary, trait-object boundary, and registry/docs alignment. Its witnesses
parse, check, and lower a static protocol impl direct `Vec2.draw` `IRCall`, a
protocol-bound concrete `id__T_Vec2` direct call, runtime protocol value
rejection with `unknown type 'Drawable'`, generic-bound requirement-call
rejection, and P17/P21 known-direct specialization evidence. The report rejects
runtime existential promotion, trait-object promotion, witness-table
promotion, dynamic-dispatch promotion, conformance-table lookup promotion,
runtime protocol value claims, broad protocol specialization, performance,
runtime behavior changes, and safe-program semantic changes.

Performance reports are evidence-only. Schema version 3 records the P20.1
performance blocker contract for the P20.0 matrix: `matrix_scope:
p20.0_benchmark_matrix`, the matrix report artifact path, the exact blocker
messages `left bounds check: missing dominance`, `heap allocation: escapes
through return`, `heap allocation: unknown call`, `not vectorized: no noalias
proof`, `not inlined: code-size budget`, `register spill: live range pressure`,
`stack fallback: unsupported aggregate return`, and `actor copy: borrowed data
crosses boundary`, plus one benchmark explanation row for every P20.0 Tetra
benchmark. `ValidatePerformanceBlockerReport` rejects missing reasons, missing
benchmark explanations, placeholder text, and fake fastest-language,
C++/Rust-parity, official-benchmark, measured-speed, or runtime-behavior-change
claims. The report does not remove blockers, run benchmarks, or change safe
semantics.

Bounds reports count proof-removed checks and checks left in place. Every
unchecked index load must carry a proof id, and external indexing remains
checked unless a future pass proves otherwise. P4.3 adds report reasons for
`proof:if:` branch guards and `proof:copy-loop:` source-copy loops alongside
the existing while-loop and for-collection proof ids; invalid or unknown view
aliases still appear as checks left in place.

Allocation reports are planner/lowering reports. Schema v2 keeps stable
`site_id`, `planned_storage`, `actual_lowering_storage`, validation status, and
lowering status for `Stack`, `Eliminated`, `Heap`, `Region`, and
`ExplicitIsland` decisions from `compiler/internal/allocplan`. It also adds a
`summary` section with allocation count, planned-storage counts,
actual-lowering counts, runtime-path counts, requested bytes, reserved bytes,
allocator-class counts, allocator-scope counts, allocator-reuse-policy counts,
and per-region allocation summaries. A backend-storage note is included when
the current stack backend still lowers a narrower planned storage class through
the conservative heap/runtime path.
P5.0 freezes the future runtime-allocation report hooks in
`compiler/internal/runtimeabi`: storage class, runtime path, bytes requested,
bytes reserved, region id, lifetime, and debug mode. P5.1 starts using those
hooks for constant-size `linux-x64` safe-slice heap rows. P15.2 reports show
`runtime_path: per_core_small_heap`, an `allocator_class` such as `small_32`,
`allocator_scope: core:0`,
`allocator_reuse_policy: same_core_same_size_class_free_list`,
`bytes_requested`, and `bytes_reserved`, or `runtime_path: large_mmap` for
large safe-slice fallback. P5.2 also fills those hooks for explicit island
allocations with `runtime_path: explicit_island`, `allocator_class:
region_bump_16`, 16-byte-rounded `bytes_reserved`, `region_id`, `lifetime`,
and debug-mode evidence. P5.3 fills region-planning hooks for bounded
function-local temporary copies, but keeps `actual_lowering_storage: Heap` in
the same row until implicit region lowering exists. P5.4 upgrades the concrete
allocation report envelope to schema v2 and requires the runtime/storage/byte
summary to match the exact allocation plan before the report is written.
For x64 native P2.1 stack-lowered sites, `planned_storage` and
`actual_lowering_storage` are both `Stack`.
P4.4 escape rows distinguish `EscapesGlobal`, `EscapesTask`, `EscapesActor`,
`EscapesClosure`, and `EscapesAggregate` in addition to return, unknown-call,
unsafe, and no-escape cases. Reading allocation metadata such as `.len` remains
non-escaping evidence.

P4.5 scalar-replacement rows use `planned_storage: Eliminated`,
`actual_lowering_storage: Eliminated`, and `lowering_status:
scalar_replacement` for tiny fixed local slices and tiny fixed `copy()`-created
owned buffers whose uses are only direct constant in-range indices. Copy-buffer
rows include the `scalar_replacement_copy_fixed_constant_indices` reason. If a
copy result is used through a dynamic index, a local alias, raw `.ptr` exposure,
an unknown call, or an escape path, the allocation report keeps a checked stack
or heap fallback instead of reporting scalar elimination. Source-copy loads are
not reported as proof-removed bounds checks unless they carry their own proof
id; P4.5's scalar copy initialization deliberately keeps checked source loads.

Validation reports and optimizer pass reports must keep removed bounds checks
attached to proof ids. A report flag may make evidence more visible, but it may
not change safe semantics or silently select a different optimizer contract.

P4.0 optimizer pass reports are produced by the internal pass manager rather
than a user backend mode. Each row records pass metadata, before/after stack IR
dumps, verifier status, proof verification status, and the declared validation
strategy. Test-only single-pass execution is available through the internal
manager options so individual passes can be exercised without creating a CLI
semantic switch.

P4.1's `basic-scalar` pass uses that same report surface. Its v0 scope is
straight-line stack IR: safe i32 constant folding, local copy propagation,
simple dead-store cleanup, and conservative algebraic simplification. Arithmetic
folding is skipped whenever the folded result would overflow i32; those cases
remain visible in the after dump and continue to execute through the ordinary
runtime/backend path.

P4.2 extends pass rows with optional `decisions` entries for optimization
choices. The first user is `inline-small-pure`, whose rows record
`action: "inlined"` with `reason: "small_pure"` for accepted call sites and
`action: "not_inlined"` with concrete reasons such as `recursive`,
`callee_contains_call`, `unsupported_effect`, or `proof_sensitive` for rejected
sites. These decisions explain the internal optimizer result; they are not a
build-mode selector.

P4.6 reuses the same decision channel for the internal
`loop-canonicalization` pass. Accepted loops record `action: "hoisted"` with
`reason: "stable_len_load"` or `action: "canonicalized"` with `reason:
"stable_len_le_minus_one_to_lt"`. Conservative loops record `action:
"not_hoisted"` with reasons such as `missing_while_bounds_proof`,
`loop_has_unknown_mutation`, or `loop_stores_len_local`. The pass report keeps
before/after stack-IR dumps and translation-validation status; it does not
remove proof-tagged bounds evidence or create a user-selectable optimization
mode.

P4.7 expands `translation_validation` rows with a nested `translation_report`.
That report records the compared function set, the number of preserved
proof-facts, how many simple local semantic checks ran, and how many
differential samples were executed. A pass that changes function signatures,
introduces or drops proof ids, rewrites straight-line algebra incorrectly, or
changes the result of a supported concrete sample fails before its report can be
treated as validated evidence.

P11 adds machine-checkable validation metadata beside that report. For every
translation-validated optimization pass, `compiler/internal/validation` can
build a `tetra.translation.validation.metadata.v1` record with the pass name,
input/output IR kinds, declared facts, sha256 hashes of the before/after IR,
the compared function set, and the translation-validation counters. The opt
manager stores this metadata in pass reports for translation-validation passes.
The companion `compiler/internal/differential` library interprets the stable
scalar-i32 Stack IR/Machine IR subset and compares source interpreter, stack
backend, register backend, and optimized backend results. These are internal
evidence paths; they do not create a user-visible backend selector or source
interpreter mode.

P23.0 adds `tetra.translation.validation.v2` as an explicit evidence/report
layer for the same internal validation path. The report records registered
optimizer pass coverage, symbolic scalar equivalence, supported i32 slice
memory samples, loop and call/inlining differential samples, bounds proof
preservation, allocation plan preservation, and sha256 before/after optimization
metadata. Its validator rejects fake full-formal-proof, exhaustive optimizer,
broad memory/loop, performance, runtime-behavior, and safe-semantics claims.

P23.1 adds `tetra.fuzz.property.differential.v1` as the companion robustness
coverage ledger. It records parser/checker generated programs, PLIR/lowering
verifier cases, backend differential matrix randomized samples, host-supported
Linux x64 native differential evidence or an explicit unavailable boundary,
runtime allocator properties, actor-transfer stress diagnostics, fuzz nightly
summary artifacts, and reduced single-sample mismatch reproducers. It remains
bounded evidence, not exhaustive fuzzing, a full program-correctness proof, a
full native differential suite, a performance claim, or a semantic mode.

P23.2 adds `tetra.formal_core.v1` as the formal-core evidence ledger for the
same bounded internal track. It records values, borrows and owned/copy,
provenance and regions, bounds proof id semantics, allocation length contracts,
allocation intent lowering, raw pointer bounds metadata, and check-elimination
validity through existing machine checks. It remains small evidence, not a full
formal proof, broad language theorem prover, unsafe-policy change, runtime
behavior change, safe-semantics change, performance claim, or public semantic
mode.

P23.3 adds `tetra.self_hosting.gate.v1` as the self-hosting promotion gate. It
defines the current verified subset boundary, reuses register backend,
translation-validation, allocator/runtime, and region-aware stdlib witnesses,
and keeps `SelfHostingClaimed=false` with `GateDecision.Allowed=false`.
Small compiler component compile, Go-vs-Tetra output comparison, deterministic
bootstrap chain, and cross-platform bootstrap story remain explicit blockers.
This is not a self-hosting claim, runtime behavior change, safe-semantics
change, or performance claim.

P24.0 adds `tetra.security.review_gate.v1` as a security review gate for the
current evidence surface. It records rows for unsafe APIs, capabilities, memory
allocator, network runtime, actor runtime, DB protocol, package/Eco system,
build scripts, supply chain, and the required artifacts
`docs/audits/security-review.md`, `docs/audits/threat-model.md`,
`docs/audits/unsafe-surface-map.md`, and
`docs/audits/capability-surface-map.md`. The report reuses current validators
for allocation contracts, raw-pointer bounds metadata, IO reactor coverage,
actor production boundaries, PostgreSQL protocol coverage, Eco validator paths,
release security-review scripts, and artifact presence. It is not a security
certification, external penetration test, CVE-free claim, release security
signoff, runtime behavior change, safe-semantics change, or performance claim.

P24.1 adds `tetra.runtime.hardening.v1` as a bounded runtime-hardening evidence
report. It records deterministic traps, OOM policy, stack overflow guard
boundary, integer overflow semantics, allocator corruption instrumentation,
region double-free/use-after-free instrumentation, actor mailbox overflow
policy, and network parser limits. The report reuses allocation contracts,
region/small-heap runtime ABI evidence, typed mailbox overflow evidence, actor
production-boundary audit, HTTP/PostgreSQL parser limit checks, backend
trap/stack-depth checks, and optimizer overflow-semantics checks. It is not a
full runtime-hardening proof, full stack-overflow protection, OOM recovery
guarantee, production actor-mailbox promotion, runtime behavior change,
safe-semantics change, or performance claim.

P24.2 adds `tetra.compatibility.stability.v1` as a bounded compatibility and
stability evidence report. It records stable diagnostic codes, versioned report
schemas, manifest compatibility checks, a breaking-change migration guide, and
a deprecation policy. The report reuses `DiagnosticCodeRegistry`,
`tools/cmd/validate-diagnostic`, P21-P24 schema constants,
`tools/cmd/validate-manifest`, `docs/generated/manifest.json`,
`docs/spec/api_diff_policy.md`,
`docs/release/breaking-change-migration-guide.md`,
`docs/release/deprecation_policy.md`,
`docs/release/v1_0_x_maintenance_policy.md`, and
`docs/spec/stdlib_naming_versioning.md`. It is not a full backward
compatibility guarantee for all future versions, diagnostic-message freeze,
automatic migration promise, manifest/runtime ABI stability promise, runtime
behavior change, safe-semantics change, or performance claim.
