# Memory Ideal Vertical Slice v1

## Goal

Extend the v0 memory correlation pattern to exactly two new borrow carrier
forms:

- `MEM-BORROW-002`: borrowed view through enum payload cannot escape owner.
- `MEM-BORROW-003`: borrowed view through monomorphized generic wrapper cannot
  escape owner.

## Success Criteria

- Correlation matrix has exactly two v1 rows and validates.
- MemoryFactGraph source facts and report projections cover
  `enum_payload_contains_borrow` and `generic_wrapper_contains_borrow`.
- Validators reject enum/generic wrapper escapes, global storage, mixed branch
  owners, and `unsafe_unknown`.
- Positive tests allow local use inside the owner lifetime and `.copy()` owned
  escape.
- MiniMemoryModel includes v1 enum/generic wrapper cases.
- Final audit classifies both rows as `validated_narrow` or `conservative`.
- Focused and full gates pass, with unrelated dirty-state failures classified
  if any occur.

## Current Context

- v0 final report is complete and explicitly rejected enum/generic carrier
  closure for v0.
- Relevant paths observed so far:
  - `compiler/internal/memoryfacts`
  - `compiler/internal/memorymodel`
  - `compiler/internal/semantics`
  - `compiler/tests/semantics/borrow_copy_test.go`
  - `tools/cmd/validate-memory-report`
  - `tools/cmd/validate-memory-correlation`
  - `docs/audits/memory-ideal-vslice-v0-correlation.md`
  - `docs/audits/memory-ideal-vslice-v0-final.md`
  - `docs/spec/memory_report_schema_v1.md`
  - `docs/generated/manifest.json`
- Graphify MCP found memoryfacts/report/semantics nodes, but no literal
  `MemoryFactGraph` node; concrete files remain authoritative.
- Worktree is heavily dirty. Preserve unrelated changes.

## Constraints

- Do not broaden to interfaces, function-typed values, callbacks, async,
  actor/task boundaries, raw pointer semantics, target parity, or broad noalias.
- Reports are projections; MemoryFactGraph remains truth.
- Use persistent Go caches, never `/tmp`.
- Run `graphify update .` after code changes before completion.

## Risks

- Generic wrappers may share parser/semantics paths with broader generic
  features; keep tests narrow to monomorphized wrapper carriers.
- Enum payload borrow propagation could accidentally include actor/task or
  function-typed payloads; reject/conserve excluded domains.
- Existing dirty worktree may make full gates fail for unrelated reasons; record
  exact failure evidence instead of reverting unrelated files.

## Approval Required

No approval gate is required for local, non-destructive source/docs/tests edits.
Ask before destructive git operations, broad codemods, external writes, or
write-enabled delegation.

## Work Packets

- P0-discovery: read-only v0 pattern and integration point discovery.
- P1-tests-model: RED/GREEN tests for MiniMemoryModel and memoryfacts/report
  validators.
- P2-semantics: RED/GREEN compiler semantics tests for enum/generic wrapper
  borrowed escape and `.copy()` behavior.
- P3-docs-audit: correlation matrix, schema/manifest, final audit.
- P4-final-verification: focused/full gates, graphify update, final report.

## Integration Policy

Integrate only against direct repo inspection. If packet notes disagree with
source, source wins. Keep final changes limited to v1.

## Verification

Run focused gates after each implementation cluster, then full gates before
completion:

- `go test ./compiler/internal/memoryfacts -count=1`
- `go test ./compiler/internal/memorymodel -count=1`
- `go test ./compiler/tests/semantics -run 'MemoryIdealV1|BorrowedAggregate|Borrow' -count=1`
- `go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation -count=1`
- `go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory-ideal-vslice-v1-correlation.md`
- `go test ./compiler/... ./cli/... ./tools/... -count=1`
- `bash scripts/ci/test.sh`
- `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`
- `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`
- `git diff --check`
- `graphify update .`

## Reusable Artifacts

The v1 workflow can become a recipe for future narrow memory-carrier closure
slices if final verification passes.
