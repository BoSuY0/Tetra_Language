# Memory Ideal Vertical Slice v0 Final Workflow Report

Status: complete

## Accepted

- P0 baseline/docs: accepted. A0-lite baseline is
  `validated_with_gaps`, not blocked.
- P1 semantics registry: accepted. Representation metadata is centralized in
  `compiler/internal/semantics/representation_metadata.go` and assignment
  rejection routes through the registry.
- P2 memoryfacts/report: accepted. MemoryFactGraph remains truth, reports are
  projections, and v0 validators reject fake derived/noalias/unsafe rows.
- P3 borrow/inout surface: accepted. B2a is limited to struct fields and
  optional payloads; B3a is limited to unique local and sequential inout.
- P4 final review: accepted. Full verification passed in the current checkout.

## Rejected

- Broad noalias wording and broad mutable alias claims.
- Enum/generic/interface/callable borrow closure for this slice.
- Actor/task, async, callback/reentrant, raw pointer, target parity, and
  performance claims.
- Any claim that reports are truth rather than MemoryFactGraph projections.

## Conflicts

- No unresolved packet conflicts. P1 warned about reserved metadata names and
  user fields; the implementation keeps the reservation scoped to
  representation metadata handling. P3 warned about broader existing tests; the
  final audit records those areas as nonclaims.

## Decisions

- `MEM-REP-001` is `validated`.
- `MEM-BORROW-001` is `validated_narrow`.
- `MEM-ALIAS-001` is `validated_narrow`.
- The correlation matrix uses the plan's limited status vocabulary, so both
  narrow rows are recorded as `validated` there and bounded by the final audit.
- The copied optional payload acceptance gap was fixed narrowly by treating
  explicit `.copy()` calls as owned before borrowed aggregate escape checks.

## Final Changes

- Added `tools/cmd/validate-memory-correlation`.
- Added `compiler/internal/semantics/representation_metadata.go`.
- Added `compiler/internal/memorymodel`.
- Extended `compiler/internal/memoryfacts` projection and validation for
  `aggregate_contains_borrow`, `optional_contains_borrow`,
  `no_alias_validated_narrow_unique_local`, and
  `no_alias_validated_narrow_sequential_inout`.
- Synchronized `tools/cmd/validate-memory-report` with the in-process report
  validator rules.
- Added targeted acceptance tests for v0 borrow/inout behavior.
- Added v0 audit docs and manifest references.
- Updated Graphify artifacts with `graphify update .`.

## Verification

- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-memoryfacts go test ./compiler/internal/memoryfacts -count=1`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-semantics go test ./compiler/internal/semantics -run 'Representation|Borrow|Lifetime|Inout|Alias|MemoryIdeal' -count=1`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-plir-validation go test ./compiler/internal/plir ./compiler/internal/validation -run 'Borrow|Alias|MemoryIdeal|Report' -count=1`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-compiler go test ./compiler -run 'Memory|Borrow|Lifetime|Alias|Unsafe|Report' -count=1`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation -count=1`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-mini go test ./compiler/internal/memorymodel -count=1`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-correlation go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory-ideal-vslice-v0-correlation.md`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-broad go test ./compiler/... ./cli/... ./tools/... -count=1`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-ci bash scripts/ci/test.sh`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- Passed: `git diff --check`.
- Passed: `graphify update .`.

## Remaining Risks

- The repository still has substantial unrelated dirty state. This workflow did
  not attempt to revert or classify unrelated work.
- v0 intentionally leaves future borrow/alias slices for enum payloads,
  generics, function/interface metadata, disjoint aliasing, raw pointers,
  actor/task boundaries, target parity, and performance evidence.
