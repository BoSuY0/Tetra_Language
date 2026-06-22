# Memory Core v2

Status: current implementation contract for the canonical memory pipeline.

Memory Core v2 makes memory decisions in the normal compiler pipeline and treats
reports as read-only projections. A build must not create a separate report-only
memory model, and report flags must not change placement, domain, proof, cache,
or lowering decisions.

## Canonical State

The canonical state is `memorypipeline.State`. It owns:

- the checked program target and deterministic `program_id`;
- one PLIR program;
- one `memoryfacts.Graph`;
- immutable `memoryfacts.Snapshot` reads with deterministic digesting;
- one `allocplan.Plan`;
- optional lowering evidence after `LowerPlannedProgram`;
- module plan and lowering digests used by cache attestation.

The core fact vocabulary lives in `compiler/internal/memoryfacts`. Stage adapters
may translate PLIR, allocation plan, validation, lowering, or optimizer evidence
into facts, but policy vocabulary and proof resolution stay in the canonical
core.

## Phase Order

The required phase order is:

1. semantics checking;
2. PLIR construction;
3. canonical fact graph construction from PLIR;
4. immutable snapshot creation;
5. allocation planning from the snapshot;
6. plan digesting and cache attestation;
7. lowering from the exact allocation plan;
8. lowering evidence projection back into the graph;
9. optimizer proof consumption and invalidation deltas;
10. report projection.

Allocation planning must reach `PhasePlanned` before lowering. Lowering must not
rebuild PLIR or allocation plans. Report emission must consume the state already
used by the build.

## Proof Lifecycle

Proof IDs originate in canonical facts or PLIR proof rows. A consumer must resolve
proofs through `memoryfacts.Snapshot`, fail closed on missing or stale proof data,
and carry proof IDs into memory-sensitive rewrites or lowering evidence when it
claims a checked decision.

Proof invalidation is explicit. When an optimizer rewrite invalidates memory
evidence, it emits a canonical delta; downstream consumers either use the new
proof IDs or conservatively skip the rewrite.

## Storage Planning And Lowering

`allocplan` records intended storage. Actual storage is emitted only by the
lowering path that produced IR instructions. A row may say planned stack, region,
island, task, actor-move, register, eliminated, function-temp, or heap storage,
but validated actual storage requires lowering evidence and a lowered artifact
ID.

Heap fallback is a valid conservative outcome. Planned storage that lowers as
heap is not promoted to a storage optimization claim.

## Domain Accounting

`runtimeabi.MemoryDomainLedger` is executable evidence. It enforces parent DAGs,
lifecycle state, budgets, transfer accounting, copied bytes, current bytes, peak
bytes, and decommit/release behavior. Task, actor, request, and island domains
must be created from typed evidence, not owner-name string matching.

Cross-domain moves and copies must carry source, destination, and proof evidence.
Borrowed values cannot cross actor/task/request boundaries unless the typed
evidence proves a supported copy or owned move.

## Islands

Island memory is the first complete domain. Island decisions route through
`islandkernel` and carry owner, parent, epoch, reset/free/move semantics, and live
borrow checks. Stale epoch use, unsafe trusted promotion, external noalias
promotion, free with live borrows, and reset with live borrows are rejected.

The current route invariant is 16 direct dangerous-decision routes and 16 total
routes. A release evidence report must keep `island_routes_direct ==
island_routes_total`.

## Backend Matrix

| Target | Backend memory operations | Current support |
| --- | --- | --- |
| `linux-x64` | `reserve`, `commit`, `decommit`, `release` | supported through runtime ABI backend events |
| `wasm32-wasi` | runtime memory backend operations | unsupported nonclaim |
| `wasm32-web` | runtime memory backend operations | unsupported nonclaim |
| other targets | runtime memory backend operations | unsupported unless target evidence is added |

Supported rows require executable backend or runtime tests. Unsupported rows must
carry an `unsupported_reason`; an unsupported target cannot be marked supported
by docs or metadata alone.

## Optimizer Requirements

Memory-sensitive optimizer rewrites must run with a canonical `MemoryContext`.
Every performed memory rewrite must carry proof IDs. Missing proof, stale proof,
unsafe-origin proof, or invalidated proof means the optimizer skips the rewrite or
emits a rejection instead of inventing memory facts.

## Evidence Schema

The release evidence schema is `tetra.memory-core-v2.evidence.v1`. Required
fields are:

- `schema`
- `git_head`
- `target`
- `program_id`
- `memory_graph_digest`
- `module_plan_digests`
- `module_lowering_digests`
- `normal_build_state_built`
- `report_flag_decision_parity`
- `cache_attestation_checked`
- `island_routes_total`
- `island_routes_direct`
- `memorymodel_outcomes_total`
- `memorymodel_outcomes_real_pipeline`
- `backend_operation_support`
- `optimizer_memory_rewrites`
- `optimizer_rewrites_with_proof_ids`
- `negative_guards`
- `nonclaims`
- `final_signoff`

The validator rejects missing digests, report-only state, route-count mismatch,
optimizer rewrites without proof IDs, unsupported backend operations marked
supported, incomplete shadow-model parity, broad unsupported claims, and final
signoff while any requirement failed.

## Explicit Nonclaims

- no universal memory safety claim;
- no universal performance claim;
- no zero heap for all programs claim;
- no all-target memory support claim;
- no all-target backend runtime claim;
- no full formal proof claim;
- no claim that reports are a decision source.
