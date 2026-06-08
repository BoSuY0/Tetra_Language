# Tetra Memory + IslandKernel Production Plan Tracker

External plan: `/home/tetra/Downloads/tetra-memory-islands-production-plan.md`

## Current Strategy

1. Treat current code/tests/validators as authoritative; the external plan's
   dump-era observations must be rechecked against this live worktree.
2. Preserve the completed memory-production baseline from
   `.workflow/memory-production-ready-v1/final-report.md`; do not redo closed
   `MEM-D04`, `MEM-E02`, `MEM-E05`, `MEM-F02`, `MEM-F04`, or `MEM-G04` unless
   current evidence contradicts them.
3. Close P0 blockers first: host broker lifecycle leak, IslandKernel skeleton,
   IslandID/Epoch facts, linear token semantics, independent proof verifier,
   and leak/soak gate.
4. Keep docs conservative until code/test/validator/release evidence exists.

## Packet Matrix

| Packet | Status | Acceptance Evidence |
| --- | --- | --- |
| `MEM-ISLAND-P00` truth audit | pending | live inventory, audit doc/script, focused core/validator gates |
| `MEM-ISLAND-P01` vocabulary/claim contract | done | island claim vocab, docs overclaim RED/GREEN, report validator island proof gate, real docs/manifest validators |
| `MEM-ISLAND-P02` IslandKernel skeleton | done | `go test ./compiler/internal/islandkernel -count=1`; pure API only, no compiler integration yet |
| `MEM-ISLAND-P03` IslandID/Epoch facts | done | memoryfacts/plir/report schema fields, projection mutation tests, CLI schema guard |
| `MEM-ISLAND-P04` linear IslandToken | done_narrow | `core.island_reset` RED/GREEN, stale token/slice semantics, PLIR epoch advancement, validation/backend/report projection sweeps |
| `MEM-ISLAND-P05` typed proof IR | done_narrow | BCE `ProofTerm` RED/GREEN, PLIR/validation typed proof mismatch rejection, memoryfacts/report typed proof projection, focused P05 gates |
| `MEM-ISLAND-P06` storage/lowering truth | pending | planned/actual/report/proof storage gates |
| `MEM-ISLAND-P07` ExternalUnsafeIsland quarantine | pending | unsafe/external promotion rejection gates |
| `MEM-ISLAND-P08` actor/task/request island boundaries | pending | borrowed boundary rejection and moved-owned transfer gates |
| `MEM-ISLAND-P09` runtime island allocator metadata | pending | runtimeabi/compiler runtime island metadata tests |
| `MEM-ISLAND-P10` debug/sanitize mode | pending | sanitizer trap fixtures and smoke rows |
| `MEM-ISLAND-P11` independent island verifier | pending | `validate-island-proof` CLI and validator fixtures |
| `MEM-ISLAND-P12` adversarial proof fuzzing | pending | proof mutation oracle and short fuzz artifacts |
| `MEM-ISLAND-P13` host Go leak audit/tests | done_narrow | close-without-cancel RED/GREEN, focused/race actornet gates, quick host leak blocker summary |
| `MEM-ISLAND-P15` benchmarks/CI gates | pending | script/workflow static tests and leak/bench schema |
| `MEM-ISLAND-P16` release gate/artifact attestation | pending | linux-x64 release smoke requires island/leak artifacts and hashes |
| `MEM-ISLAND-P17` docs correction | pending | docs/manifest overclaim gates |
| `MEM-ISLAND-P18` final production audit | pending | no unresolved P0/P1 and full final gate stack |

## Current Iteration

1. Establish new goal/tracker state for IslandKernel plan. Done: `GOAL.md`,
   `PLAN.md`, `ATTEMPTS.md`, `NOTES.md`, and `CONTROL.md` now describe
   `MEM-ISLAND-P00..P18`; previous memory-production evidence remains anchored
   in `.workflow/memory-production-ready-v1/final-report.md`.
2. Live P0 audit. Done: `compiler/internal/islandkernel`,
   `tools/cmd/validate-island-proof`, and `tools/validators/islandproof` were
   missing then; `go.sum` contains `go.uber.org/goleak` but `go.mod` does not
   require it, so P13 used a local pprof-stack regression without a dependency
   change.
3. First implementation packet. Done narrow: `MEM-ISLAND-P13` fixed
   `Broker.Close()` without context cancellation, added the required regression
   and wired `host leak blocker suite` into quick `test-all`; fresh evidence is
   in `reports/memory-islands-production/test-all-quick-p13-20260608_091350Z/`.
4. `MEM-ISLAND-P02` IslandKernel skeleton. Done: new isolated package exposes
   pure decisions for all section 9.2 questions with table-driven tests; no
   compiler integration or production proof claim yet. Evidence:
   `go test ./compiler/internal/islandkernel -count=1`, focused
   islandkernel/memorymodel run, `git diff --check`, and `graphify update .`
   with `21866 nodes`, `68112 edges`, `1203 communities`.
5. `MEM-ISLAND-P01` claim vocabulary/docs overclaim guard. Done: shared
   vocabulary now registers `island_kernel_model_only`,
   `island_epoch_validated`, `island_sanitize_runtime_checked`, and
   `island_proof_verified`; memory report validation rejects validated island
   proof claims unless `validator_name` is `validate-island-proof`; docs
   validation rejects `Memory 100%`, `IslandKernel complete`, leak-free, and
   arbitrary unsafe pointer-safety wording outside explicit nonclaim context.
   Evidence includes RED/GREEN focused tests, real `verify-docs --manifest`,
   `validate-manifest`, `git diff --check`, and `graphify update .` with
   `21876 nodes`, `68139 edges`, `1198 communities`.
6. `MEM-ISLAND-P03` IslandID/Epoch facts. Done: `Fact`, `ReportRow`, PLIR
   `Fact`, and standalone `validate-memory-report` rows now carry
   `island_id`, `epoch`, and `base_id`; projection rejects `island_id`
   mutation, island-backed rows without positive epoch, and reports that
   rewrite invalidated/stale epoch facts as validated; PLIR island allocation
   facts and `FromPLIRAndAllocPlan` preserve memory-ref identity. Evidence:
   RED/GREEN schema, CLI, PLIR, and PLIR→memoryfacts tests; focused package
   sweep; combined P01-P03 sweep; `git diff --check`; `graphify update .` with
   `21888 nodes`, `68174 edges`, `1189 communities`.
7. `MEM-ISLAND-P04` linear IslandToken/free/reset semantics. Done narrow:
   `core.island_reset` is now a consuming builtin with alias/effects metadata,
   lowering emits `IRIslandReset`, validation treats reset as epoch/lifetime
   invalidation, PLIR records `island_epoch_advanced` and carries reset epoch
   onto later island allocations, and linux/x64 plus wasm build-only backends
   handle the IR instruction. Evidence includes RED/GREEN reset tests,
   memoryfacts vocabulary/projection coverage, focused package sweeps,
   `git diff --check`, and `graphify update .` with `21910 nodes`,
   `68253 edges`, `1202 communities`. P10/P11/P16 still own sanitizer traps,
   independent proof validation, and release attestation.
8. `MEM-ISLAND-P05` typed proof IR. Done narrow: BCE proof IDs now carry
   compiler-owned `ProofTerm` metadata with subject base, index value,
   operation, range, optional island epoch/base, and derivation source; PLIR
   verifier rejects missing/mismatched typed bounds proof terms; validation
   propagates the typed term into `ProofReport`; memoryfacts and
   `validate-memory-report` project and require structured typed proof fields
   for validated bounds-proof rows. Evidence: P05 RED compile/mismatch tests,
   focused package sweeps over PLIR/lower/validation/memoryfacts/report
   validator, compiler report sweep, `git diff --check`, and `graphify update
   .` with `21934 nodes`, `68314 edges`, `1191 communities`. Remaining P05
   broad scope: noalias, storage, and island-move proof terms/invalidation
   rules; continue through P06-P08/P11 rather than claiming full proof IR.
9. Next packet. Pending: start `MEM-ISLAND-P06` storage/lowering truth while
   keeping noalias/storage/island-move typed proof completion explicit.

## Open Decisions

- None for the current batch. `MEM-ISLAND-P14` is outside repository scope under
  the active user objective.
