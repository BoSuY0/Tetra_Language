# Final Report: Memory Ideal Vertical Slice v8 Report Integrity

Decision: accepted.
Status: validated_narrow.
Date: 2026-06-06.

## Summary

`MEM-REPORT-008` adds a narrow graph/report projection and claim-drift
integrity slice. `MemoryFactGraph` remains the truth source, and
`tetra.memory-report.v1` remains a projection. The slice adds no new memory
semantics, optimizer behavior, target parity, performance evidence,
FFI/runtime proof, arbitrary external pointer safety, or "Memory 100%" claim.

## Implemented Scope

- `compiler/internal/memoryfacts.ValidateReportProjection` validates a report
  against the graph that produced it.
- Projection validation rejects unknown `source_fact_id`, missing graph facts,
  and altered projected fields, including `parent_fact_id`, validator fields,
  `claim_level`, `cost_class`, and `normal_build_check`.
- `tools/cmd/validate-memory-correlation` recognizes the exact five v8
  `MEM-REPORT-*` rows and rejects missing, extra, or widened v8 rows.
- `memory_claim_drift_validator` rejects broad safety wording such as
  "Memory 100%" or broad safety proven from conservative/rejected evidence.
- v8 audit docs, schema/design notes, manifest references, and Graphify
  artifacts were updated.

## Requirement Results

| Requirement | Status | Evidence |
| --- | --- | --- |
| `MEM-REPORT-001` | `validated_narrow` | `ValidateReportProjection`; `TestValidateReportProjectionRejectsUnknownSourceFactID`. |
| `MEM-REPORT-002` | `validated_narrow` | `ValidateReportProjection`; `TestValidateReportProjectionRejectsMissingProjectedGraphFact`. |
| `MEM-REPORT-003` | `validated_narrow` | `ValidateReportProjection`; `TestValidateReportProjectionRejectsAlteredCostClass`; `TestValidateReportProjectionRejectsDroppedNormalBuildCheck`. |
| `MEM-REPORT-004` | `validated_narrow` | v8 required row set and status checks; `TestValidateMemoryCorrelationRejectsV8MissingClaimDriftRow`; `TestValidateMemoryCorrelationRejectsV8ExtraRow`. |
| `MEM-REPORT-005` | `rejected` | `memory_claim_drift_validator`; `TestValidateMemoryCorrelationRejectsV8BroadSafetyClaimDrift`. |

## RED Evidence

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v8-report-memoryfacts-red go test ./compiler/internal/memoryfacts -count=1
```

Result: failed with `undefined: ValidateReportProjection`.

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v8-report-tools-red go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation -count=1
```

Result: failed because `MEM-REPORT-*` rows were treated as v0 rows and claim
drift was not detected.

## GREEN And Final Gates

All commands exited 0 unless noted.

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v8-report-memoryfacts go test ./compiler/internal/memoryfacts -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v8-report-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation ./tools/cmd/validate-memory-fuzz-oracle -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v8-report-correlation go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory-ideal-vslice-v8-report-correlation.md
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v8-report-regression bash -lc 'for f in docs/audits/memory-ideal-vslice-v*-correlation.md; do go run ./tools/cmd/validate-memory-correlation --file "$f"; done'
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v8-report-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v8-report-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v8-report-broad go test ./compiler/... ./cli/... ./tools/... -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v8-report-ci bash scripts/ci/test.sh
git diff --check
graphify update .
```

Canonical CI result:

```text
OK
Artifact: tetra.release.v0_4_0.go-test-suite.v1
```

Graphify result:

```text
Rebuilt: 21358 nodes, 66713 edges, 1186 communities
```

`git diff --check` exited 0 before and after Graphify.

## Dirty Worktree Caveat

`git status --short` remains heavily dirty with many pre-existing modified and
untracked files. Scoped v8 files are present, but this final report does not
claim a clean release worktree.

## Nonclaims

- No "Memory 100% complete".
- No new memory semantics.
- No optimizer rewrite or broad optimizer correctness proof.
- No arbitrary external pointer safety.
- No FFI/runtime lifetime proof.
- No target parity.
- No performance claim.
- No production runtime/ABI proof.
- No clean-release claim while the worktree remains dirty.
