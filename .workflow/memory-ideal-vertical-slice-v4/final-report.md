# Final Report: Memory Ideal Vertical Slice v4

## Outcome

Accepted. The integrated lower-layer, semantics, docs, validator, and
verification slice covers exactly the async/task/actor boundary rows requested:
`MEM-BORROW-008`, `MEM-BORROW-009`, `MEM-BORROW-010`, and `MEM-ALIAS-004`.

## Accepted Results

- MemoryFactGraph projections accepted for
  `async_boundary_borrow_conservative`, `task_boundary_borrow_rejected`,
  `actor_boundary_borrow_rejected`, and `boundary_noalias_conservative`.
- Validators accepted for `async_boundary_borrow_validator`,
  `task_boundary_borrow_validator`, `actor_boundary_borrow_validator`, and
  `boundary_alias_conservative_validator`.
- MiniMemoryModel v4 cases accepted for local async use before suspension,
  await crossing conservatism, task reject, actor reject, copied task/actor
  accept, owned actor accept, and boundary noalias conservatism.
- Semantics tests accepted for local async borrowed use, async result escape
  rejection, actor `.copy()` acceptance, actor borrowed/wrapper rejection,
  current typed-task boundary rejection, unknown task target rejection, and
  task/actor boundary conservative noalias proxy diagnostics.

## Rejected Results

No packet is rejected at this stage. The current typed-task API does not expose
a general task payload send expression, so task payload-copy semantics are
modeled in MiniMemoryModel and kept conservative in final audit wording rather
than promoted as a production task runtime claim.

## Conflicts Resolved

The previous `GOAL.md` described v3 while the active thread goal described v4.
`GOAL.md` was replaced with the v4 contract before implementation.

The actor optional-wrapper test is rejected earlier as an unsupported optional
wrapper payload. That remains acceptable narrow evidence because it prevents a
borrowed optional wrapper from crossing the actor boundary.

## Verification Evidence

- Focused memoryfacts passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v4-memoryfacts go test ./compiler/internal/memoryfacts -count=1`.
- Focused memorymodel passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v4-mini go test ./compiler/internal/memorymodel -count=1`.
- Focused tools passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v4-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation -count=1`.
- Focused semantics passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v4-semantics go test ./compiler/tests/semantics -run 'MemoryIdealV4|Async|Await|Task|Actor|Borrow|Alias|NoAlias' -count=1`.
- v4 correlation validation passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v4-correlation go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory-ideal-vslice-v4-correlation.md`.
- Manifest validation passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v4-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`.
- Docs verification passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v4-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- Broad Go test passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v4-broad go test ./compiler/... ./cli/... ./tools/... -count=1`.
- CI script passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v4-ci bash scripts/ci/test.sh`.
  Output ended with `OK` and artifact
  `tetra.release.v0_4_0.go-test-suite.v1`.
- Diff whitespace check passed:
  `git diff --check`.
- Graphify update passed:
  `graphify update .`, rebuilding `graphify-out` with 21271 nodes, 66470
  edges, and 1168 communities.

After adding the final positive task/actor local-use semantics test, the
current-state focused semantics, broad Go test, CI script, and Graphify update
were repeated with the same commands above. The repeated CI run ended with
`OK` and artifact `tetra.release.v0_4_0.go-test-suite.v1`.

## Remaining Risks

Async suspension remains conservative unless proven local and non-escaping.
Task boundary payload transfer is constrained by the current task APIs.
Task/actor noalias remains conservative. This slice does not provide full
production actor runtime, full async lifetime system, structured concurrency,
cancellation model, distributed actor memory model, broad noalias, target
parity, or performance evidence.
