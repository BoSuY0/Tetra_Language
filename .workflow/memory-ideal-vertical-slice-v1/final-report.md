# Memory Ideal Vertical Slice v1 Final Workflow Report

Status: complete

## Accepted

- P0 discovery: accepted. v0 patterns and v1 integration points were grounded in
  concrete files before edits.
- P1 tests/model/report: accepted. MiniMemoryModel, MemoryFactGraph projection,
  report validators, and correlation validator cover enum payload and generic
  wrapper carriers.
- P2 semantics: accepted. Explicit `MemoryIdealV1` tests cover local use,
  global-storage rejection, owned-return rejection through existing aggregate
  tests, and `.copy()` owned escape. No checker code change was needed because
  the current checker already recurses through enum payloads and monomorphized
  generic struct wrappers.
- P3 docs/audit: accepted. v1 correlation, final audit, schema docs, and
  manifest hooks were added.
- P4 final verification: accepted. Focused gates, full gates, docs/manifest
  gates, diff check, and Graphify update passed.

## Rejected

- Broad noalias wording and target parity claims.
- Interface, function-typed value, callback, async, actor/task, and raw pointer
  borrow closure.
- Treating reports as truth instead of MemoryFactGraph projections.

## Conflicts

- No unresolved packet conflicts.
- The first broad gate failed because `tools/cmd/validate-manifest` in-test
  feature fixtures still listed only v0 memory-ideal docs. The fixture was
  updated to match the new validator requirement, and the broad gate passed on
  rerun.

## Decisions

- `MEM-BORROW-002` is `validated_narrow`.
- `MEM-BORROW-003` is `validated_narrow`.
- The v1 correlation validator accepts exactly the v1 row set when v1 IDs are
  present while preserving v0 validation for the v0 matrix.
- `unsafe_unknown` remains rejected or conservative; it does not produce trusted
  borrowed source facts.

## Final Changes

- Added `docs/audits/memory-ideal-vslice-v1-correlation.md`.
- Added `docs/audits/memory-ideal-vslice-v1-final.md`.
- Updated `docs/spec/memory_report_schema_v1.md` with v1 projections.
- Added MiniMemoryModel wrapper kinds and tests for enum payload/generic wrapper
  cases.
- Added MemoryFactGraph PLIR-derived projections:
  `enum_payload_contains_borrow` and `generic_wrapper_contains_borrow`.
- Updated in-process and CLI memory report validators to require parent facts
  for v1 derived borrow rows.
- Updated `validate-memory-correlation` for v0/v1 exact row sets and
  `validated_narrow`.
- Added explicit `MemoryIdealV1` semantics tests for local/global/copy behavior.
- Updated `docs/generated/manifest.json`,
  `tools/cmd/validate-manifest/main.go`, and manifest tests.
- Refreshed `graphify-out/` with `graphify update .`.

## Verification

- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-memoryfacts go test ./compiler/internal/memoryfacts -count=1`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-mini go test ./compiler/internal/memorymodel -count=1`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-semantics go test ./compiler/tests/semantics -run 'MemoryIdealV1|BorrowedAggregate|Borrow' -count=1`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation -count=1`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-correlation go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory-ideal-vslice-v1-correlation.md`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-correlation go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory-ideal-vslice-v0-correlation.md`.
- Passed after fixture fix: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-broad go test ./compiler/... ./cli/... ./tools/... -count=1`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-ci bash scripts/ci/test.sh`; final output included `OK` and `Artifact: tetra.release.v0_4_0.go-test-suite.v1`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- Passed: `git diff --check` after `graphify update .`.
- Passed: `graphify update .`; rebuilt 21217 nodes, 66349 edges, and 1181
  communities.
- Passed: dynamic workflow helpers; `collect_results.py` generated the
  integration checklist and `verify_workflow.py` reported workflow verification
  passed.

## Remaining Risks

- The repository remains heavily dirty with substantial unrelated tracked and
  untracked state. This workflow preserved that state and did not attempt to
  classify unrelated files.
- v1 intentionally remains narrow. Interfaces, function-typed values,
  callbacks, async, actor/task boundaries, raw pointer semantics, target parity,
  and broad noalias remain nonclaims.

## Reusable Follow-up

- Future memory carrier slices can reuse the v1 pattern: add a narrow wrapper
  kind/model case, derive a MemoryFactGraph projection with parent fact
  requirement, add exact-row correlation validation, and close with a final
  audit that names nonclaims.
