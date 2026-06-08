# Tetra Memory + IslandKernel Production Goal

<goal>
Implement the retained IslandKernel plan in
`/home/tetra/Downloads/tetra-memory-islands-production-plan.md`, with the
user-requested third memory-graph persistence item excluded from repository
scope.

Completion means Tetra's supported memory/island surface is production-ready in
the plan's explicit sense: IslandKernel proof decisions, IslandID/Epoch memory
identity, linear island token/free/reset rules, typed proof artifacts,
independent proof validation, sanitizer/leak/fuzz/release attestations, and
public-claim discipline are implemented and verified for `linux-x64` supported
release evidence. Conservative fallbacks and non-goals remain explicit.
</goal>

<context>
Primary source of scope:

- `/home/tetra/Downloads/tetra-memory-islands-production-plan.md`

Read first on each continuation:

- `AGENTS.md`
- `GOAL.md`
- `PLAN.md`
- `ATTEMPTS.md`
- `NOTES.md`
- `CONTROL.md`
- Graphify MCP context for IslandKernel, explicit islands, memory facts,
  proof/bounds/noalias/storage validators, actornet broker lifecycle, release
  gates, and docs overclaim validators
- `graphify-out/GRAPH_REPORT.md` when local graph community context is useful

Prior completed baseline:

- `.workflow/memory-production-ready-v1/final-report.md` proves the previous
  supported compiler-owned memory surface goal, including `MEM-D04`,
  `MEM-E02`, `MEM-E05`, `MEM-F02`, `MEM-F04`, and `MEM-G04`.
</context>

<constraints>
- Always communicate with the user in Ukrainian.
- Use Graphify MCP first for architecture/codebase navigation, then verify
  concrete files with normal repo inspection.
- Preserve unrelated dirty worktree changes. Work with the current dirty state;
  do not revert old memory-production changes.
- Use persistent Go caches under `.cache/` or `$HOME/.cache`; never set
  `GOCACHE` to `/tmp`.
- Do not claim perfect memory, `Memory 100%`, Rust-like lifetime parity, full
  target parity, arbitrary unsafe/external pointer safety, full actor runtime
  scheduler proof, global/root leak impossibility, official benchmark
  superiority, or leak-free host tooling.
- Supported production release evidence remains `linux-x64` unless
  target-specific runner evidence is added and validated.
</constraints>

<scorecard>
Primary metric: retained packet tasks `MEM-ISLAND-P00` through
`MEM-ISLAND-P13` and `MEM-ISLAND-P15` through `MEM-ISLAND-P18` are implemented
and verified. `MEM-ISLAND-P14` is removed from this repository goal.

Passing threshold:

- No unresolved P0/P1 blocker remains.
- The retained production Definition of Done items have command/file/artifact
  evidence.
- Release evidence is fresh, hashed, validated, and target-scoped.
- Public docs and reports reject overclaims.

Regression checks:

- focused package tests named by each packet;
- broad `go test ./compiler/... ./cli/... ./tools/... -count=1`;
- `bash scripts/ci/test.sh`;
- memory island release smoke/artifact validators;
- docs/manifest validators;
- `git diff --check`;
- `graphify update .` after code changes.

Stop condition:

- Stop and record a blocker if the plan requires destructive cleanup,
  dependency changes not already present, external infrastructure, broad target
  parity evidence unavailable locally, or a product decision not inferable from
  the plan/code.
- If the same focused/full gate fails twice for the same reason without new
  evidence, record the blocker before attempting a third variant.
</scorecard>

<done_when>
The goal is complete only when all are true:

- Supported surface is explicit.
- In supported safe surface, scoped memory leaks/use-after-free/double-free are
  impossible or rejected before lowering.
- Scoped islands have deterministic cleanup semantics.
- `IslandID/Epoch/provenance` exist in compiler-owned facts where memory
  identity matters.
- `free/reset` invalidates epoch; stale access is rejected or trapped in
  sanitizer.
- Borrow cannot escape island lifetime.
- Actor/task/request boundaries do not accept borrowed views.
- Unsafe/external memory remains `ExternalUnsafeIsland`/`unsafe_unknown` without
  proof.
- `unsafe_unknown` cannot produce `safe_known`, noalias, BCE, or trusted
  storage.
- Removed bounds checks have compiler-owned typed proof.
- Noalias claims have proof and invalidation rules.
- Storage claims compare planned/reported/actual lowering plus proof.
- Reports validate against `MemoryFactGraph` and are not source of truth.
- Independent verifier checks proof artifacts.
- Debug/sanitize build catches island id/epoch/proof/storage violations.
- Fuzz oracle has release-blocking proof mutation cases.
- Race/stress/soak gates exist for memory-touching areas.
- CI workflows run memory-touching gates or release gate blocks.
- Release gate collects artifacts, hashes, validators, and summaries.
- Docs/manifest contain no overclaims.
- Non-goals are explicit.
- Final audit contains exact command evidence from this live repo with `.git`.
</done_when>

<workflow>
1. Re-read this file, `CONTROL.md`, and the external plan.
2. Use Graphify MCP first, then inspect concrete code.
3. Maintain the packet matrix for retained `MEM-ISLAND-*` packets, with
   `MEM-ISLAND-P14` absent from repository scope.
4. Execute the next packet with RED/GREEN tests when code changes.
5. Run focused verification after each packet.
6. Update `GOAL.md ## Progress`, `PLAN.md`, `ATTEMPTS.md`, `NOTES.md`, and
   `CONTROL.md`.
7. Run `graphify update .` after code changes.
8. Run broad gates before claiming a major packet group complete.
9. Mark the active goal complete only after a requirement-by-requirement audit
   proves every `done_when` item.
</workflow>

<working_memory>
Maintain:

- `PLAN.md`: packet matrix, current strategy, next batch, open decisions.
- `ATTEMPTS.md`: command evidence and RED/GREEN attempts.
- `NOTES.md`: durable discoveries, nonclaims, design rationale, blockers.
- `CONTROL.md`: operator control surface for the long goal.
- `.workflow/memory-islands-production-v1/`: workflow-local mirrors and final
  reports.
</working_memory>

<verification_loop>
Minimum final gate stack:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-islands-final go test ./compiler/internal/islandkernel ./compiler/internal/memoryfacts ./compiler/internal/plir ./compiler/internal/validation ./compiler/internal/allocplan ./compiler/internal/lower -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-islands-final go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation ./tools/cmd/validate-memory-production ./tools/cmd/validate-island-proof -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-islands-final go test ./compiler/... ./cli/... ./tools/... -count=1
git diff --check
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-islands-final bash scripts/ci/test.sh
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-islands-final bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh --report-dir reports/memory-islands-production/final-linux-x64
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-islands-final bash scripts/dev/fuzz-nightly.sh --short --out-dir reports/memory-islands-production/fuzz-short
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-islands-final go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-islands-final go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
graphify update .
```
</verification_loop>

## Progress

- 2026-06-08: Active thread goal received for full IslandKernel production
  plan. Previous top-level trackers described the completed
  memory-production-ready supported-surface goal, so they were reset to this
  new `MEM-ISLAND-P00..P18` contract while preserving the previous final report
  under `.workflow/memory-production-ready-v1/`. Bridge: run live P0 audit and
  begin the first packet, with plan recommendation favoring `MEM-ISLAND-P13`
  host leak fix when shipping reliability is urgent.
- 2026-06-08: Live P0 audit confirmed `compiler/internal/islandkernel`,
  `tools/cmd/validate-island-proof`, and `tools/validators/islandproof` are
  still absent; prior current-memory blockers `MEM-D04`, `MEM-E02`, `MEM-E05`,
  `MEM-F02`, `MEM-F04`, and `MEM-G04` are covered by
  `.workflow/memory-production-ready-v1/final-report.md`. `MEM-ISLAND-P13`
  host broker lifecycle core fix completed: RED
  `TestBrokerCloseWithoutCancelStopsServeWatcher` failed with one lingering
  watcher, `Broker.Close()` now closes an internal `done` channel, focused
  broker and race gates pass, quick `test-all` now includes `host leak blocker
  suite`, and fresh summary validation passed at
  `reports/memory-islands-production/test-all-quick-p13-20260608_091350Z/`.
  Final `graphify update .` rebuilt `21823` nodes, `68004` edges, `1190`
  communities. Bridge: start `MEM-ISLAND-P02` IslandKernel skeleton while P15/
  P16 remain responsible for full leak/soak/pprof release attestation.
- 2026-06-08: `MEM-ISLAND-P02` implemented as an isolated
  `compiler/internal/islandkernel` package. RED package test first failed on
  missing `MemoryRef`, `Token`, `Proof`, `Decision`, and required decision
  functions; GREEN added pure data-in/data-out API for `CanBorrow`,
  `CanReturn`, `CanStoreGlobal`, `CanCaptureClosure`, `CanSendToActor`,
  `CanSendToTask`, `CanMoveIsland`, `CanFreeIsland`, `CanResetIsland`,
  `CanClaimNoAlias`, `CanEliminateBoundsCheck`, `CanLowerAsExplicitIsland`,
  `CanPromoteUnsafeRoot`, `CanTrustStorage`, and `CanEraseRuntimeCheck`.
  Evidence: `go test ./compiler/internal/islandkernel -count=1`, focused
  `go test ./compiler/internal/islandkernel ./compiler/internal/memorymodel
  -count=1`, `git diff --check`, and `graphify update .` passed with `21866`
  nodes, `68112` edges, `1203` communities. Bridge: next packet is
  `MEM-ISLAND-P01` claim vocabulary or `MEM-ISLAND-P03` IslandID/Epoch fact
  projection; prefer P01 first to keep docs/reports from overclaiming the new
  skeleton.
- 2026-06-08: `MEM-ISLAND-P01` completed as a conservative claim-contract
  guard. RED tests first failed on missing island claim vocabulary,
  `IslandKernel complete` docs wording detection, and report-side
  `island_proof_verified` accepted/misclassified without `validate-island-proof`.
  GREEN added shared claim constants `island_kernel_model_only`,
  `island_epoch_validated`, `island_sanitize_runtime_checked`, and
  `island_proof_verified`; docs validation now rejects `Memory 100%`,
  `IslandKernel complete`, leak-free, perfect-memory, and arbitrary unsafe
  pointer-safety wording outside explicit nonclaim context; memory report
  validators require `validate-island-proof` for validated island proof claims.
  Evidence: focused P01 tests, real `verify-docs --manifest`, manifest
  validator test/run, package sweep, `git diff --check`, and `graphify update
  .` passed with `21876` nodes, `68139` edges, `1198` communities. Bridge:
  start `MEM-ISLAND-P03` IslandID/Epoch facts; P11 remains responsible for the
  actual independent verifier implementation.
- 2026-06-08: `MEM-ISLAND-P03` completed for IslandID/Epoch schema and
  projection. RED tests first failed on absent `IslandID`/`Epoch`/`BaseID`
  fields in `memoryfacts.Fact`, `ReportRow`, standalone memory-report rows, and
  `plir.Fact`, then on `FromPLIRAndAllocPlan` dropping island memory-ref
  identity. GREEN added `island_id`, positive `epoch`, and `base_id` to facts
  and reports; projection rejects identity mutation and validated rewrites of
  invalidated/stale epoch facts; PLIR builder/verifier now carries island
  memory-ref identity for `ProvenanceIsland` values; standalone
  `validate-memory-report` rejects island-backed rows without epoch/base
  evidence. Evidence: focused P03 tests, package sweep, combined P01-P03 sweep,
  `git diff --check`, and `graphify update .` passed with `21888` nodes,
  `68174` edges, `1189` communities. Bridge: start `MEM-ISLAND-P04` linear
  IslandToken/free/reset semantics; P03 does not yet implement reset/free
  invalidation or sanitizer traps.
- 2026-06-08: `MEM-ISLAND-P04` completed narrowly for linear reset/free token
  semantics. RED coverage first exposed missing `core.island_reset`,
  `FactIslandEpochAdvanced`, unsupported wasm `IRIslandReset`, unknown
  memoryfacts claim vocabulary for `island_epoch_advanced`, and PLIR reset
  allocations incorrectly carrying `island:next epoch 1`. GREEN added the
  consuming builtin/effects/alias, `IRIslandReset` lowering/stack effects,
  explicit-island validation reset state, stale token/slice semantic rejection,
  PLIR token epoch tracking, memoryfacts projection vocabulary, and linux/x64
  plus wasm build-only backend support. Evidence: P04 PLIR reset test, package
  sweep across memoryvocab/memoryfacts/semantics/PLIR/lower/validation/
  backends/opt/runtime/semantics/ownership/safety, validator tool checks,
  `git diff --check`, and final `graphify update .` with `21910` nodes,
  `68253` edges, `1202` communities. Bridge: start `MEM-ISLAND-P05` typed
  proof IR; P10/P11/P16 still own sanitizer traps, independent proof
  validation, and release attestation.
- 2026-06-08: `MEM-ISLAND-P05` completed narrowly for BCE typed proof IR and
  report projection. RED coverage first failed on absent `plir.ProofTerm`,
  missing `Function.ProofTerms`, validation fixtures without typed terms, and
  later on range-format mismatch between `i in [0, xs.len)` and legacy
  `0..xs.len`. GREEN added `ProofTerm` metadata for bounds proofs, PLIR
  verification of subject base/index/range and island epoch/base fields,
  validation propagation into `ProofReport`, memoryfacts/report/CLI structured
  typed proof fields, and a report fix that carries explicit-island allocplan
  identity from PLIR into memoryfacts rows. Evidence: focused P05 PLIR/
  validation tests, memoryfacts/report validator typed-proof RED/GREEN, full
  P05 package sweep, compiler report sweep, `git diff --check`, and
  `graphify update .` with `21934` nodes, `68314` edges, `1191` communities.
  Bridge: start `MEM-ISLAND-P06`; noalias, storage, and island-move proof
  terms remain explicit follow-up scope and are not claimed complete.
- 2026-06-08: Current user objective excised `MEM-ISLAND-P14` from
  repository-owned scope. Bridge: update top-level trackers, run no-residue
  searches, refresh Graphify, and complete only if the live repo proves the
  removed third item is no longer tracked.
