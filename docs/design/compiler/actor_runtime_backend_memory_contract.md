# Actor Runtime Backend Memory Contract

Status: design contract for future implementation packets.

This document turns the P103 actor ping-pong blocker discovery into a
worker-ready boundary for backend eligibility, actor-boundary ownership
evidence, and production actor memory evidence.

## 1. Status And Non-Claims

Accepted scope:

- This is a compiler-internal design contract only.
- It describes required evidence before actor runtime calls may be selected by
  a Machine IR/register backend path or a direct emitter path.
- It separates scalar actor ping-pong value-copy semantics from typed owned
  actor transfer semantics.
- It separates actor runtime-call backend eligibility from production actor
  memory byte-budget and backpressure readiness.

Non-claims:

- This does not claim current actor code is native/register.
- This does not claim actor zero-copy.
- This does not claim RSS reduction.
- This does not claim production actor memory completion.
- This does not authorize report relabeling. Later claims must be backed by
  fresh sidecars and validators.

## 2. Current Evidence

Fresh P101 evidence is
`reports/benchmark-vnext-memory-baseline/tier1-after-matrix-multiply-main-native/report.json`.
P101 accepted only the matrix-multiply row-level native/register slice. The
fresh aggregate still has three fallback rows, including
`actor_ping_pong_tetra`.

P102 extracted the fresh actor row from that report:

- `backend_path="fallback"`;
- `backend_blockers=["unsupported_effect_runtime_call"]`;
- `bounds_left=0`;
- `heap_allocations=0`;
- `perf_blockers=["actor_copy.borrowed_data_boundary"]`.

P103 confirmed the actor-specific blocker map:

- `actor_ping_pong_tetra.backend.json` has `backend="stack"`,
  `register_path=0`, and `stack_fallback=2`.
- `pong` first falls back with `runtime_call=__tetra_actor_recv`.
- `main` first falls back with `runtime_call=__tetra_actor_spawn`.
- The same backend sidecar already records
  `runtime_features_required=["actor_runtime"]`,
  `runtime_features_linked=["actor_runtime"]`,
  `runtime_features_initialized=["actor_runtime"]`,
  `runtime_object_linked=true`, and `runtime_object_initialized=true`.
- `actor_ping_pong_tetra.explain.txt` shows scalar actor operations:
  `core.spawn`, `core.send`, `core.recv`, and `core.sender`; the source uses
  scalar `core.send`/`core.recv`, not typed `core.send_typed`.
- `actor_ping_pong_tetra.actor-transfer.json` is empty:
  `copy=0`, `move=0`, `zero_copy_move=0`, and `bytes_copied=0`.
- `iteration-01.heap.json` contains actor domains:
  `domain:actor:000` and `domain:actor:001`, each with `peak_bytes=88` and
  `bytes_copied=88`.
- `summary.json` still reports process heap values of zero for all samples.

The current state therefore has useful runtime actor-domain telemetry, but it
does not yet have backend-call eligibility, actor-boundary transfer evidence, or
production byte-budget/backpressure evidence.

## 3. Problem Split

### Backend Runtime-Call Blocker

`compiler/internal/buildreports/backend.go` currently classifies calls whose
names look like runtime effects as `unsupported_effect_runtime_call`. The fresh
actor row reports the first observed blockers as `__tetra_actor_recv` in `pong`
and `__tetra_actor_spawn` in `main`.

The runtime object is already linked and initialized for this row. The blocker
is a backend-selection/lowering contract gap, not a missing runtime object
linkage fact.

### Actor-Boundary Ownership Blocker

`actor_copy.borrowed_data_boundary` is a conservative actor-transfer blocker.
The current scalar benchmark sends `i32` values with `core.send`, so the empty
`actor-transfer.json` does not prove copy, move, or zero-copy semantics. The
typed transfer reporter currently records rows for `core.send_typed`, not for
scalar `core.send`.

### Production Memory Blocker

P24 added production runtime byte counters, and P30/Tier 1 evidence records
runtime-measured actor domains. That still does not complete production actor
memory because byte budgets and byte-aware backpressure are not proven
end-to-end for the Tier 1 actor row.

## 4. Backend Runtime Call Contract

Actor runtime calls may be considered backend-eligible only when all of these
facts are explicit in tests and sidecars:

- The exact runtime symbol is in the accepted actor-call subset. For the scalar
  ping-pong row this must at least cover the calls that lowering emits for
  `core.spawn`, `core.send`, `core.recv`, and `core.sender`; the observed first
  blockers are `__tetra_actor_spawn` and `__tetra_actor_recv`.
- `compiler/internal/runtimeabi/runtimeabi.go` provides the slot signature for
  the symbol, and the backend plan consumes that signature instead of relying on
  name-only allow-listing.
- Machine IR or the direct emitter represents the call boundary with callee,
  target ABI, argument slots, return slots, and caller-saved clobbers.
- Runtime object requirements remain explicit:
  `runtime_features_required`, `runtime_features_linked`,
  `runtime_features_initialized`, `runtime_object_linked`,
  `runtime_object_initialized`, and any lazy-init blockers.
- The call plan preserves safe runtime behavior: no elision, inlining, or
  substitution of actor runtime semantics is allowed by backend eligibility.
- Backend reports distinguish accepted actor runtime-call lowering from generic
  runtime-effect calls.

Machine IR/direct emitter requirements:

- Calls with scalar ABI shape, such as current `__tetra_actor_spawn`
  `ParamSlots=1 ReturnSlots=1` and `__tetra_actor_recv`
  `ParamSlots=0 ReturnSlots=1`, may be candidates only after clobber and
  runtime-object evidence exists.
- Actor calls with multi-slot returns, message-frame APIs, distributed actor
  symbols, actor-state symbols, or unknown signatures must stay fallback until
  a separate contract covers their ABI and memory behavior.
- Generic `__tetra_*`, `runtime.*`, or `core.*` calls must continue to fall back
  unless they match the exact accepted actor-call contract.
- A row with mixed accepted and unaccepted actor runtime calls must remain
  fallback. A future worker must prove the whole row path, not only the first
  reported blocker.

Report requirements before a later worker may claim actor backend progress:

- per-function `backend_path` rows must show the accepted actor-call detail;
- `backend_blockers` for the row must no longer include
  `unsupported_effect_runtime_call`;
- the sidecar must still show actor runtime feature/object linkage and
  initialization;
- targeted backend tests must include negative cases for unaccepted actor
  runtime calls.

## 5. Actor Boundary Ownership Contract

Scalar ping-pong and typed owned transfer are different contracts.

Scalar ping-pong:

- `actor_ping_pong_tetra` sends `i32` values with scalar `core.send`.
- A scalar actor send is a value-copy boundary. It is not a move operation.
- A scalar value-copy boundary is not zero-copy evidence.
- The current empty `actor-transfer.json` proves only that no typed transfer
  rows were emitted. It does not prove that copies are absent or optimized away.

Typed owned transfer:

- `core.send_typed` is the current reportable typed actor transfer boundary.
- Small scalar payload rows report `ownership="copy"` and
  `transfer_mode="copy"`.
- Explicit copied borrowed views report copy rows.
- Owned region payloads may report `transfer_mode="move"`.
- Owned region-backed slice payloads may report
  `transfer_mode="zero_copy_move"` only when the same typed payload carries the
  owner and the sender loses access.
- Existing typed ownership evidence is local typed-mailbox evidence and must not
  be generalized to distributed actor zero-copy or full production runtime
  behavior.

`actor_copy.borrowed_data_boundary` may be removed or replaced only when the
sidecar truth matches the benchmark contract:

- If the Tier 1 actor row remains scalar ping-pong, the report must explicitly
  prove scalar value-copy behavior at every actor boundary and must not cite
  zero-copy.
- If the row is changed or complemented by a typed transfer benchmark, the
  source must exercise `core.send_typed`, and `actor-transfer.json` must contain
  rows for each payload with `ownership`, `transfer_mode`, `runtime_path`,
  `bytes_copied`, `zero_copy`, `claim_level`, `boundary_scope`, and
  `production_runtime_validated`.
- The sidecar must distinguish copy, move, and zero-copy move. Empty totals are
  not transfer evidence.
- Borrowed views must either be rejected, explicitly copied, or represented by
  a typed owned-region transfer that consumes the sender's owner. They must not
  silently become zero-copy actor sends.

## 6. Production Actor Memory Contract

Actor-domain telemetry is necessary but not sufficient. Production actor memory
evidence must cover all of these layers:

- actor domains: per-actor `domain_bytes` with runtime-measured current, peak,
  and copied bytes;
- mailbox/message bytes: enqueue/dequeue accounting for mailbox live bytes,
  peak bytes, reclaimed bytes, copy count, and allocation-failure counters;
- actor byte budgets: configured limits, current budget usage, peak usage, and
  explicit budget policy per actor or scheduler domain;
- byte backpressure: runtime evidence that the byte budget can block, yield,
  fail, or otherwise apply the declared backpressure policy;
- Tier 1 ingestion: report and validator fields that expose the runtime
  sidecar evidence without replacing it with allocation-report estimates.

Required evidence shape:

- runtime sidecars must expose actor memory counters for the selected iteration;
- Tier 1 `memory_evidence.domain_bytes_evidence.evidence_class` must remain
  `runtime_measured`;
- at least one actor domain must have nonzero peak or copied-byte evidence for
  actor work;
- byte-budget and backpressure fields must be present before any production
  actor memory claim;
- validators must fail if required actor byte-budget/backpressure fields are
  absent for a row that claims production actor memory readiness.

P24 byte counters and P30 actor-domain telemetry are partial prerequisites. They
do not by themselves prove production actor byte-budget/backpressure.

## 7. Implementation Tracks

### P105 Candidate: Backend Runtime-Call Lowering

Goal: define and implement the exact actor runtime-call backend eligibility
boundary for scalar ping-pong calls.

Likely focus:

- target-neutral representation or direct emitter plan for accepted actor
  runtime calls;
- ABI signatures and clobbers for the accepted call subset;
- backend report rows that distinguish accepted actor-call lowering from
  generic runtime-effect fallback;
- negative tests for unaccepted actor, distributed actor, actor-state, and
  multi-slot message calls.

This should come first because the row remains fallback even if transfer
metadata improves. It can be implemented without claiming actor transfer,
production byte-budget/backpressure, zero-copy, or RSS progress.

### P106 Candidate: Actor Ownership/Transfer Report Truth

Goal: make the actor-boundary evidence match the benchmark contract.

Likely focus:

- decide whether scalar `actor_ping_pong_tetra` should keep an explicit
  scalar-copy report or whether a separate typed-transfer row is needed;
- ensure `actor-transfer.json` is non-empty when transfer claims are made;
- remove or replace `actor_copy.borrowed_data_boundary` only when sidecar rows
  prove the chosen copy/move/zero-copy semantics.

### P107 Candidate: Production Actor Budget/Backpressure

Goal: complete production actor memory byte-budget and byte-backpressure
evidence.

Likely focus:

- runtime byte-budget enforcement;
- runtime sidecar fields for budget/backpressure;
- Tier 1 ingestion and validator requirements;
- tests that prove missing production actor memory fields prevent claims.

This track should follow P105/P106 for row-claim sequencing, or run as an
explicitly separate production-memory packet that does not claim row promotion.

## 8. Required Verification

Targeted tests for a backend runtime-call packet:

- runtime ABI signature tests for every accepted actor runtime symbol;
- Machine IR verifier tests, or direct-emitter tests, proving callee, ABI,
  return slots, argument slots, and clobbers are explicit;
- x64 emitter/runtime parity tests for scalar actor ping-pong against the stack
  path;
- buildreport tests proving accepted actor runtime calls no longer report
  `unsupported_effect_runtime_call`;
- negative buildreport tests proving unaccepted actor runtime calls still fall
  back.

Targeted tests for an ownership/transfer packet:

- scalar-copy report tests if scalar `core.send` remains the benchmark contract;
- typed `core.send_typed` report tests if a typed-transfer row is added;
- borrowed-view rejection or explicit-copy tests;
- owned-region move and owned-region-slice zero-copy-move tests only for local
  typed mailbox payloads that consume sender ownership.

Targeted tests for a production memory packet:

- actor runtime byte-counter tests for mailbox live, peak, reclaimed, copied,
  budget, and backpressure fields;
- Tier 1 metadata ingestion tests for actor-domain and actor-budget sidecars;
- validator tests that reject missing byte-budget/backpressure evidence when a
  production actor memory claim is present.

Fresh Tier 1 artifacts required before any later actor-row claim:

- `reports/benchmark-vnext-memory-baseline/.../report.json`;
- `.../artifacts/bin/actor_ping_pong_tetra.backend.json`;
- `.../artifacts/bin/actor_ping_pong_tetra.perf.json`;
- `.../artifacts/bin/actor_ping_pong_tetra.actor-transfer.json`;
- `.../artifacts/bin/actor_ping_pong_tetra.explain.txt`;
- `.../artifacts/heap-telemetry/actor_ping_pong_tetra/iteration-01.heap.json`;
- `.../artifacts/heap-telemetry/actor_ping_pong_tetra/summary.json`;
- matching validator output for the fresh report.

Sidecar fields required before a later worker may claim the actor row:

- backend: row/function evidence for accepted actor runtime-call lowering and
  no remaining backend blockers for the row;
- perf/transfer: no stale `actor_copy.borrowed_data_boundary` unless the row is
  intentionally still blocked; transfer sidecar rows must prove the chosen
  copy/move semantics;
- memory: runtime-measured actor domains plus actor byte-budget/backpressure
  evidence when claiming production actor memory;
- non-claims: no native/register, zero-copy, RSS, or production-memory wording
  unless the matching fresh evidence exists.

## 9. Open Decisions

- Should the Tier 1 actor ping-pong row stay scalar-copy, with explicit
  scalar actor-boundary copy evidence, or should it remain blocked until a typed
  ownership-transfer benchmark exists?
- Is a separate typed-transfer benchmark row needed so scalar ping-pong does not
  carry zero-copy or owned-region semantics it does not exercise?
- Can backend promotion for scalar actor runtime calls happen before production
  byte-budget/backpressure is complete, if the result explicitly avoids any
  production actor memory claim?
- Should actor runtime-call eligibility live first in Machine IR, or as a narrow
  direct x64 emitter plan that later migrates into Machine IR?
- Which actor runtime symbols belong in the first accepted subset beyond the
  observed `__tetra_actor_spawn` and `__tetra_actor_recv` blockers?
