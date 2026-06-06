# MEM-FUZZ-012 Final Report

Date: 2026-06-06

Decision: accepted
Status: validated_narrow
Release/worktree decision: proceed_with_blockers

`MEM-FUZZ-012` is accepted as a narrow compiler-visible memory fuzz oracle
release-evidence slice. It adds deterministic Tier 1 evidence for the v0-v11
Memory Ideal chain and keeps `MemoryFactGraph` validators as the truth source.
The generated oracle report remains an evidence artifact, not a replacement for
report, graph, or compiler validators.

## Requirement Map

| Requirement | Status | Interpretation |
| --- | ---: | --- |
| `MEM-FUZZ-001` | `validated_narrow` | Tier 1 short CI smoke records deterministic v0-v11 memory oracle cases. |
| `MEM-FUZZ-002` | `validated_narrow` | Compiler crash and miscompile categories require reducer/reproducer artifact kinds. |
| `MEM-FUZZ-003` | `release_blocking` | Unsafe promotion, missing bounds proof id, trusted storage under escape, and report validation failure block release promotion. |
| `MEM-FUZZ-004` | `boundary_recorded` | Tier 2 nightly fuzz preserves seeds, unstable triage, and minimized repro expectations as boundary evidence. |
| `MEM-FUZZ-005` | `release_blocking` | Tier 3 focused memory fuzz must pass or classify every failure before release promotion. |

## RED Evidence

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v12-fuzz-tools-red go test ./tools/cmd/validate-memory-fuzz-oracle ./tools/cmd/memory-fuzz-short -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v12-fuzz-compiler-red go test ./compiler -run 'MemoryFuzzOracle.*V12|ValidateMemoryFuzzOracleReportRejectsV12' -count=1`

Both failed before implementation because `MemoryFuzzOracleReport` did not have
v12 `Requirements` or `SliceCoverage` fields and the v12 requirement, blocking,
and tier policy types did not exist.

## GREEN Evidence

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v12-fuzz-tools go test ./tools/cmd/validate-memory-fuzz-oracle ./tools/cmd/memory-fuzz-short -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v12-fuzz-compiler go test ./compiler -run 'MemoryFuzzOracle.*V12|ValidateMemoryFuzzOracleReportRejectsV12' -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v12-fuzz-oracle go run ./tools/cmd/memory-fuzz-short --tier=1 --report-dir reports/memory-fuzz-short/v12`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v12-fuzz-validate go run ./tools/cmd/validate-memory-fuzz-oracle --report reports/memory-fuzz-short/v12/memory-fuzz-oracle.json`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v12-fuzz-regression bash -lc 'for f in docs/audits/memory-ideal-vslice-v*-correlation.md; do go run ./tools/cmd/validate-memory-correlation --file "$f"; done'`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v12-fuzz-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v12-fuzz-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v12-fuzz-broad go test ./compiler/... ./cli/... ./tools/... -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v12-fuzz-ci bash scripts/ci/test.sh`
- `git diff --check`
- `git status --short`
- `graphify update .`

Results: all gates passed. `scripts/ci/test.sh` ended `OK` with artifact
`tetra.release.v0_4_0.go-test-suite.v1`. `git diff --check` exited 0.
`git status --short` exited 0 but output was non-empty. `graphify update .`
rebuilt `21427 nodes`, `66887 edges`, and `1184 communities`.

## Artifact Evidence

- `reports/memory-fuzz-short/v12/memory-fuzz-oracle.json`
- `reports/memory-fuzz-short/v12/summary.md`

The v12 JSON contains five `MEM-FUZZ-*` requirement rows, twelve deterministic
slice coverage rows (`v0` through `v11`), blocking-case rows, Tier policy rows,
and required artifact kinds for `compiler_crash_reproducer`,
`miscompile_reproducer`, and `miscompile_reducer`.

## Nonclaims

- no exhaustive fuzz proof;
- no arbitrary unsafe safety;
- no full runtime/ABI/target parity proof;
- no performance claim;
- no clean-release claim while `git status --short` remains dirty;
- no replacement for `MemoryFactGraph` validators;
- no `Memory 100%`.

## Caveat

This packet supports `accepted` / `validated_narrow` for the bounded v12 oracle
evidence scope only. The release/worktree decision remains
`proceed_with_blockers` because the worktree is still non-empty/dirty.
