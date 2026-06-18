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

### MEM-ASYNC-001

- Status: `validated_narrow`.
- Evidence: `MiniMemoryModel` and `MemoryFactGraph` project
  `pre_await_local_borrow_validated` only for a compiler-visible borrowed value
  used before suspension with local no-escape proof.
- Validator: `pre_await_local_borrow_validator`.

### MEM-ASYNC-002

- Status: `conservative`.
- Evidence: `post_await_borrow_conservative` keeps borrowed values crossing
  await/suspend conservative unless separately proven.
- Validator: `post_await_borrow_conservative_validator`.

### MEM-ASYNC-003

- Status: `rejected`.
- Evidence: `cancellation_borrow_lifetime_invalidated` rejects borrowed
  task-owned lifetime assumptions after cancellation.
- Validator: `cancellation_lifetime_invalidation_validator`.

### MEM-ASYNC-004

- Status: `conservative`.
- Evidence: `task_group_noalias_conservative` invalidates broad noalias at
  task-group / structured concurrency boundaries.
- Validator: `task_group_boundary_conservative_validator`.

### MEM-ASYNC-005

- Status: `conservative`.
- Evidence: `actor_reentrant_callback_conservative` keeps actor reentrant
  callback borrow/storage evidence conservative without a separate proof.
- Validator: `actor_reentrant_callback_boundary_validator`.

## Validator Map

- `pre_await_local_borrow_validator`:
  - Implementation: `compiler/internal/memorymodel.evaluateAsyncV10`.
  - Implementation: `compiler/internal/memoryfacts.addBorrowAggregateV0Facts`.
  - Scope: validate only pre-suspension local no-escape borrow evidence.
- `post_await_borrow_conservative_validator`:
  - Implementation: `compiler/internal/memorymodel.evaluateAsyncV10`.
  - Implementation: `compiler/internal/memoryfacts.addBorrowAggregateV0Facts`.
  - Scope: keep post-await borrow evidence conservative.
- `cancellation_lifetime_invalidation_validator`:
  - Implementation: `compiler/internal/memorymodel.evaluateAsyncV10`.
  - Implementation: `compiler/internal/memoryfacts.addBorrowAggregateV0Facts`.
  - Scope: reject cancellation-invalidated task-owned borrow lifetimes.
- `task_group_boundary_conservative_validator`:
  - Implementation: `compiler/internal/memorymodel.evaluateInout`.
  - Implementation: `compiler/internal/memoryfacts.addNoAliasMetadataFacts`.
  - Scope: keep task-group noalias evidence conservative.
- `actor_reentrant_callback_boundary_validator`:
  - Implementation: `compiler/internal/memorymodel.evaluateAsyncV10`.
  - Implementation: `compiler/internal/memoryfacts.addBorrowAggregateV0Facts`.
  - Scope: keep actor reentrant callback borrow/storage evidence conservative.
- `correlation_exact_row_validator`:
  - Implementation: `tools/cmd/validate-memory-correlation`.
  - Scope: v10 required row set and status checks.

## RED Evidence

Focused RED was observed before implementation:

```bash
TOOLS_CACHE="$(pwd)/.cache/go-build-memory-v10-async-tools-red"
MINI_CACHE="$(pwd)/.cache/go-build-memory-v10-async-mini-red"
FACTS_CACHE="$(pwd)/.cache/go-build-memory-v10-async-memoryfacts-red"
GOTELEMETRY=off GOCACHE="$TOOLS_CACHE" \
  go test ./tools/cmd/validate-memory-correlation \
  -run 'V10|AcceptsV10|RejectsV10' \
  -count=1
GOTELEMETRY=off GOCACHE="$MINI_CACHE" \
  go test ./compiler/internal/memorymodel \
  -run 'V10|AsyncCancellation' \
  -count=1
GOTELEMETRY=off GOCACHE="$FACTS_CACHE" \
  go test ./compiler/internal/memoryfacts \
  -run 'V10|AsyncCancellation' \
  -count=1
```

The RED failures showed that `MEM-ASYNC-*` rows were treated as unexpected,
MiniMemoryModel lacked v10 cancellation/task-group/reentrant actor vocabulary,
and `MemoryFactGraph` did not project v10 source/report rows.

## Current Focused Evidence

```bash
TOOLS_CACHE="$(pwd)/.cache/go-build-memory-v10-async-tools"
MINI_CACHE="$(pwd)/.cache/go-build-memory-v10-async-mini"
FACTS_CACHE="$(pwd)/.cache/go-build-memory-v10-async-memoryfacts"
GOTELEMETRY=off GOCACHE="$TOOLS_CACHE" \
  go test ./tools/cmd/validate-memory-correlation \
  -run 'V10|AcceptsV10|RejectsV10' \
  -count=1
GOTELEMETRY=off GOCACHE="$MINI_CACHE" \
  go test ./compiler/internal/memorymodel \
  -run 'V10|AsyncCancellation' \
  -count=1
GOTELEMETRY=off GOCACHE="$FACTS_CACHE" \
  go test ./compiler/internal/memoryfacts \
  -run 'V10|AsyncCancellation' \
  -count=1
```

All three focused commands passed.

## Final Gate Evidence

Focused gates passed:

```bash
FACTS_CACHE="$(pwd)/.cache/go-build-memory-v10-async-memoryfacts"
MINI_CACHE="$(pwd)/.cache/go-build-memory-v10-async-mini"
ALLOC_CACHE="$(pwd)/.cache/go-build-memory-v10-async-allocplan"
VALID_CACHE="$(pwd)/.cache/go-build-memory-v10-async-validation"
LOWER_CACHE="$(pwd)/.cache/go-build-memory-v10-async-lower"
SEM_CACHE="$(pwd)/.cache/go-build-memory-v10-async-semantics"
TOOLS_CACHE="$(pwd)/.cache/go-build-memory-v10-async-tools"
ASYNC_RE='Async|Task|Actor|Cancel|Storage|Escape'
GOTELEMETRY=off GOCACHE="$FACTS_CACHE" \
  go test ./compiler/internal/memoryfacts -count=1
GOTELEMETRY=off GOCACHE="$MINI_CACHE" \
  go test ./compiler/internal/memorymodel -count=1
GOTELEMETRY=off GOCACHE="$ALLOC_CACHE" \
  go test ./compiler/internal/allocplan -run "$ASYNC_RE" -count=1
GOTELEMETRY=off GOCACHE="$VALID_CACHE" \
  go test ./compiler/internal/validation -run "$ASYNC_RE" -count=1
GOTELEMETRY=off GOCACHE="$LOWER_CACHE" \
  go test ./compiler/internal/lower -run "$ASYNC_RE" -count=1
GOTELEMETRY=off GOCACHE="$SEM_CACHE" \
  go test \
  ./compiler/tests/semantics \
  ./compiler/tests/ownership \
  -run 'Memory|Borrow|Escape|Async|Await|Task|Actor|Cancel|Callback|Alias' \
  -count=1
GOTELEMETRY=off GOCACHE="$TOOLS_CACHE" \
  go test \
  ./tools/cmd/validate-memory-report \
  ./tools/cmd/validate-memory-correlation \
  ./tools/cmd/validate-memory-fuzz-oracle \
  -count=1
```

Correlation/docs gates passed:

```bash
CORR_CACHE="$(pwd)/.cache/go-build-memory-v10-async-correlation"
REG_CACHE="$(pwd)/.cache/go-build-memory-v10-async-regression"
MANIFEST_CACHE="$(pwd)/.cache/go-build-memory-v10-async-manifest"
DOCS_CACHE="$(pwd)/.cache/go-build-memory-v10-async-docs"
CORR="docs/audits/memory-ideal-vslice-v10-async-cancel-correlation.md"
GOTELEMETRY=off GOCACHE="$CORR_CACHE" \
  go run ./tools/cmd/validate-memory-correlation --file "$CORR"
GOTELEMETRY=off GOCACHE="$REG_CACHE" \
  bash -lc '
    for f in docs/audits/memory-ideal-vslice-v*-correlation.md; do
      go run ./tools/cmd/validate-memory-correlation --file "$f"
    done
  '
GOTELEMETRY=off GOCACHE="$MANIFEST_CACHE" \
  go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
GOTELEMETRY=off GOCACHE="$DOCS_CACHE" \
  go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

Full gates passed:

```bash
BROAD_CACHE="$(pwd)/.cache/go-build-memory-v10-async-broad"
CI_CACHE="$(pwd)/.cache/go-build-memory-v10-async-ci"
GOTELEMETRY=off GOCACHE="$BROAD_CACHE" \
  go test ./compiler/... ./cli/... ./tools/... -count=1
GOTELEMETRY=off GOCACHE="$CI_CACHE" bash scripts/ci/test.sh
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
