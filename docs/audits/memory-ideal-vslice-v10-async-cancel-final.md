# Memory Ideal Vertical Slice v10 Async Cancellation Final Audit

Status: `validated_narrow` for the bounded async cancellation and structured
boundary conservatism surface.

Decision: proceed for v10 evidence. Current focused, docs/manifest, broad,
CI, hygiene, dirty-worktree, and Graphify evidence supports accepting the
narrow `MEM-ASYNC-010` model/correlation/report-projection spine.

This slice is not "Memory 100%", not production actor runtime proof, not a
distributed actor memory model, not a full async lifetime system, not complete
structured concurrency proof, not target parity, not performance evidence, not
broad noalias, and not arbitrary FFI/runtime or external pointer safety.

## Requirement Results

| requirement_id | status | evidence |
| --- | --- | --- |
| `MEM-ASYNC-001` | `validated_narrow` | `MiniMemoryModel` and `MemoryFactGraph` project `pre_await_local_borrow_validated` only for a compiler-visible borrowed value used before suspension with local no-escape proof; validator: `pre_await_local_borrow_validator`. |
| `MEM-ASYNC-002` | `conservative` | `post_await_borrow_conservative` keeps borrowed values crossing await/suspend conservative unless separately proven; validator: `post_await_borrow_conservative_validator`. |
| `MEM-ASYNC-003` | `rejected` | `cancellation_borrow_lifetime_invalidated` rejects borrowed task-owned lifetime assumptions after cancellation; validator: `cancellation_lifetime_invalidation_validator`. |
| `MEM-ASYNC-004` | `conservative` | `task_group_noalias_conservative` invalidates broad noalias at task-group / structured concurrency boundaries; validator: `task_group_boundary_conservative_validator`. |
| `MEM-ASYNC-005` | `conservative` | `actor_reentrant_callback_conservative` keeps actor reentrant callback borrow/storage evidence conservative without a separate proof; validator: `actor_reentrant_callback_boundary_validator`. |

## Validator Map

| validator | implementation |
| --- | --- |
| `pre_await_local_borrow_validator` | `compiler/internal/memorymodel.evaluateAsyncV10` and `compiler/internal/memoryfacts.addBorrowAggregateV0Facts` validate only pre-suspension local no-escape borrow evidence. |
| `post_await_borrow_conservative_validator` | `compiler/internal/memorymodel.evaluateAsyncV10` and `compiler/internal/memoryfacts.addBorrowAggregateV0Facts` keep post-await borrow evidence conservative. |
| `cancellation_lifetime_invalidation_validator` | `compiler/internal/memorymodel.evaluateAsyncV10` and `compiler/internal/memoryfacts.addBorrowAggregateV0Facts` reject cancellation-invalidated task-owned borrow lifetimes. |
| `task_group_boundary_conservative_validator` | `compiler/internal/memorymodel.evaluateInout` and `compiler/internal/memoryfacts.addNoAliasMetadataFacts` keep task-group noalias evidence conservative. |
| `actor_reentrant_callback_boundary_validator` | `compiler/internal/memorymodel.evaluateAsyncV10` and `compiler/internal/memoryfacts.addBorrowAggregateV0Facts` keep actor reentrant callback borrow/storage evidence conservative. |
| `correlation_exact_row_validator` | `tools/cmd/validate-memory-correlation` v10 required row set and status checks. |

## RED Evidence

Focused RED was observed before implementation:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-tools-red go test ./tools/cmd/validate-memory-correlation -run 'V10|AcceptsV10|RejectsV10' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-mini-red go test ./compiler/internal/memorymodel -run 'V10|AsyncCancellation' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-memoryfacts-red go test ./compiler/internal/memoryfacts -run 'V10|AsyncCancellation' -count=1
```

The RED failures showed that `MEM-ASYNC-*` rows were treated as unexpected,
MiniMemoryModel lacked v10 cancellation/task-group/reentrant actor vocabulary,
and `MemoryFactGraph` did not project v10 source/report rows.

## Current Focused Evidence

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-tools go test ./tools/cmd/validate-memory-correlation -run 'V10|AcceptsV10|RejectsV10' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-mini go test ./compiler/internal/memorymodel -run 'V10|AsyncCancellation' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-memoryfacts go test ./compiler/internal/memoryfacts -run 'V10|AsyncCancellation' -count=1
```

All three focused commands passed.

## Final Gate Evidence

Focused gates passed:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-memoryfacts go test ./compiler/internal/memoryfacts -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-mini go test ./compiler/internal/memorymodel -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-allocplan go test ./compiler/internal/allocplan -run 'Async|Task|Actor|Cancel|Storage|Escape' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-validation go test ./compiler/internal/validation -run 'Async|Task|Actor|Cancel|Storage|Escape' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-lower go test ./compiler/internal/lower -run 'Async|Task|Actor|Cancel|Storage|Escape' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-semantics go test ./compiler/tests/semantics ./compiler/tests/ownership -run 'Memory|Borrow|Escape|Async|Await|Task|Actor|Cancel|Callback|Alias|Storage' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation ./tools/cmd/validate-memory-fuzz-oracle -count=1
```

Correlation/docs gates passed:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-correlation go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory-ideal-vslice-v10-async-cancel-correlation.md
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-regression bash -lc 'for f in docs/audits/memory-ideal-vslice-v*-correlation.md; do go run ./tools/cmd/validate-memory-correlation --file "$f"; done'
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

Full gates passed:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-broad go test ./compiler/... ./cli/... ./tools/... -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-ci bash scripts/ci/test.sh
```

`scripts/ci/test.sh` ended `OK` and emitted artifact
`tetra.release.v0_4_0.go-test-suite.v1`.

Hygiene and graph evidence:

```bash
git diff --check
git status --short
graphify update .
```

`git diff --check` exited 0. `git status --short` exited 0 with non-empty
dirty output, so this packet does not claim a clean release worktree.
`graphify update .` rebuilt `21387 nodes`, `66790 edges`, and
`1186 communities`, updating `graphify-out/graph.json` and
`graphify-out/GRAPH_REPORT.md`.

## Nonclaims

- No "Memory 100% complete" claim.
- No production actor runtime proof.
- No distributed actor memory model.
- No full async lifetime system.
- No complete structured concurrency proof.
- No target parity.
- No performance claim.
- No broad noalias.
- No arbitrary FFI/runtime lifetime proof.
- No arbitrary external pointer safety.
- No clean-release claim while `git status --short` remains dirty.
