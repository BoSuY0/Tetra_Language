# Final Report: Memory Ideal Vertical Slice v9 Storage

Date: 2026-06-06

Decision: accepted

Status: validated_narrow

## Summary

`MEM-STORAGE-009` adds a narrow escape-aware storage/lowering integrity slice
to the existing v0-v8 Memory Ideal evidence chain. `MemoryFactGraph` remains
the truth source, `tetra.memory-report.v1` remains a projection, and the slice
does not add new optimizer, target, performance, runtime, FFI lifetime,
arbitrary external pointer, or "Memory 100%" claims.

## Integration Result

| Requirement | Status | Integration summary |
| --- | --- | --- |
| `MEM-STORAGE-001` | `rejected` | Escaped values cannot project or lower as trusted stack/register/region/function-temp/island/task/actor storage. `allocplan.VerifyPlan` checks `Storage`, `PlannedStorage`, and `ActualLoweringStorage`. |
| `MEM-STORAGE-002` | `validated_narrow` | Trusted local/region storage requires compiler-owned no-escape proof status. Missing proof rows are rejected. |
| `MEM-STORAGE-003` | `validated_narrow` | Heap/conservative fallback rows preserve `source_fact_id` through report schema validation and require a reviewable `reason` in graph/report validators. |
| `MEM-STORAGE-004` | `conservative` | Async/task/actor/FFI/unknown-call storage boundaries remain heap/conservative unless a later narrow proof exists. |

## Files

- `compiler/internal/allocplan/plan.go`
- `compiler/internal/allocplan/plan_test.go`
- `compiler/internal/memoryfacts/from_plir.go`
- `compiler/internal/memoryfacts/from_plir_test.go`
- `compiler/internal/memoryfacts/report.go`
- `compiler/internal/memoryfacts/report_test.go`
- `compiler/internal/memoryfacts/validate.go`
- `compiler/internal/memorymodel/mini.go`
- `compiler/internal/memorymodel/mini_test.go`
- `tools/cmd/validate-memory-correlation/main.go`
- `tools/cmd/validate-memory-correlation/main_test.go`
- `docs/audits/memory-ideal-vslice-v9-storage-correlation.md`
- `docs/audits/memory-ideal-vslice-v9-storage-final.md`
- `docs/spec/memory_report_schema_v1.md`
- `docs/design/memory_production_core_v1.md`
- `docs/generated/manifest.json`
- `graphify-out/GRAPH_REPORT.md`
- `graphify-out/graph.json`
- `graphify-out/manifest.json`
- `.workflow/memory-ideal-vertical-slice-v9-storage/*`
- `GOAL.md`

## RED Evidence

These gates failed before implementation for the intended reasons:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-allocplan-red go test ./compiler/internal/allocplan -run 'Storage|Escape|Region|Heap|Lower' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-memoryfacts-red go test ./compiler/internal/memoryfacts -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-mini-red go test ./compiler/internal/memorymodel -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-tools-red go test ./tools/cmd/validate-memory-correlation -count=1
```

Observed RED failures:

- escaped actual trusted lowering returned `<nil>`;
- trusted storage without no-escape proof returned `<nil>`;
- heap fallback without reason was accepted by allocplan, graph, and report
  validation;
- MiniMemoryModel lacked v9 storage fields/outcomes;
- `validate-memory-correlation` treated `MEM-STORAGE-*` rows as unexpected v0
  rows.

## GREEN Evidence

Focused gates passed:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-memoryfacts go test ./compiler/internal/memoryfacts -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-allocplan go test ./compiler/internal/allocplan -run 'Storage|Escape|Region|Heap|Lower' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-validation go test ./compiler/internal/validation -run 'Storage|Escape|Region|Heap|Lower' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-lower go test ./compiler/internal/lower -run 'Storage|Escape|Region|Heap|Lower' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-mini go test ./compiler/internal/memorymodel -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-semantics go test ./compiler/tests/semantics ./compiler/tests/ownership -run 'Memory|Borrow|Escape|Storage|Region|Heap|Actor|Task|Async|FFI|Raw|Pointer' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation ./tools/cmd/validate-memory-fuzz-oracle -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-correlation go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory-ideal-vslice-v9-storage-correlation.md
```

Regression/docs gates passed:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-regression bash -lc 'for f in docs/audits/memory-ideal-vslice-v*-correlation.md; do go run ./tools/cmd/validate-memory-correlation --file "$f"; done'
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

Full gates passed:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-broad go test ./compiler/... ./cli/... ./tools/... -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-ci bash scripts/ci/test.sh
```

`scripts/ci/test.sh` ended `OK` and emitted artifact
`tetra.release.v0_4_0.go-test-suite.v1`.

Hygiene and graph evidence:

```bash
git diff --check
git status --short
graphify update .
```

`git diff --check` exited 0. `git status --short` exited 0 with non-empty,
heavily dirty output, so this packet does not claim a clean release worktree.
`graphify update .` rebuilt `21379 nodes`, `66765 edges`, and
`1165 communities`, updating `graphify-out/graph.json` and
`graphify-out/GRAPH_REPORT.md`.

## Nonclaims

- No "Memory 100% complete" claim.
- No full region inference.
- No optimizer-wide allocation correctness proof.
- No target parity.
- No performance claim.
- No production actor runtime proof.
- No full async lifetime system.
- No arbitrary FFI lifetime proof.
- No arbitrary external pointer safety.
- No clean-release claim while `git status --short` remains dirty.
