# Final Report: Memory Ideal Vertical Slice v10 Async Cancellation

Date: 2026-06-06

Decision: accepted

Status: validated_narrow

## Summary

`MEM-ASYNC-010` adds a narrow async cancellation and structured boundary
conservatism slice to the existing v0-v9 Memory Ideal evidence chain.
`MemoryFactGraph` remains the truth source, `tetra.memory-report.v1` remains a
projection, and the slice does not add production actor runtime, distributed
actor, full async lifetime, full structured concurrency, target parity,
performance, broad noalias, arbitrary FFI/runtime lifetime, arbitrary external
pointer, clean-release, or "Memory 100%" claims.

## Integration Result

| Requirement | Status | Integration summary |
| --- | --- | --- |
| `MEM-ASYNC-001` | `validated_narrow` | Pre-await local borrowed value use validates only with compiler-visible local no-escape proof. |
| `MEM-ASYNC-002` | `conservative` | Borrowed values crossing await/suspend remain conservative unless separately proven. |
| `MEM-ASYNC-003` | `rejected` | Cancellation invalidates task-owned borrowed lifetime assumptions. |
| `MEM-ASYNC-004` | `conservative` | Task-group / structured concurrency boundary noalias evidence remains conservative. |
| `MEM-ASYNC-005` | `conservative` | Actor reentrant callback borrow/storage evidence remains conservative unless separately proven. |

## Files

- `compiler/internal/memorymodel/mini.go`
- `compiler/internal/memorymodel/mini_test.go`
- `compiler/internal/memoryfacts/from_plir.go`
- `compiler/internal/memoryfacts/from_plir_test.go`
- `compiler/internal/memoryfacts/validate.go`
- `tools/cmd/validate-memory-correlation/main.go`
- `tools/cmd/validate-memory-correlation/main_test.go`
- `docs/audits/memory-ideal-vslice-v10-async-cancel-correlation.md`
- `docs/audits/memory-ideal-vslice-v10-async-cancel-final.md`
- `docs/spec/memory_report_schema_v1.md`
- `docs/design/memory_production_core_v1.md`
- `docs/generated/manifest.json`
- `README.md`
- `graphify-out/GRAPH_REPORT.md`
- `graphify-out/graph.json`
- `graphify-out/manifest.json`
- `.workflow/memory-ideal-vertical-slice-v10-async-cancel/*`
- `GOAL.md`

## RED Evidence

These gates failed before implementation for the intended reasons:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-tools-red go test ./tools/cmd/validate-memory-correlation -run 'V10|AcceptsV10|RejectsV10' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-mini-red go test ./compiler/internal/memorymodel -run 'V10|AsyncCancellation' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v10-async-memoryfacts-red go test ./compiler/internal/memoryfacts -run 'V10|AsyncCancellation' -count=1
```

Observed RED failures:

- `validate-memory-correlation` treated `MEM-ASYNC-*` rows as unexpected v0
  rows.
- `MiniMemoryModel` lacked v10 post-await, cancellation, task-group, and actor
  reentrant callback vocabulary.
- `MemoryFactGraph` did not project the v10 report rows.

## GREEN Evidence

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

`git diff --check` exited 0. `git status --short` exited 0 with non-empty dirty
output, so this packet does not claim a clean release worktree.
`graphify update .` rebuilt `21387 nodes`, `66790 edges`, and
`1186 communities`.

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
