# Final Report: Memory Ideal Vertical Slice v6 Bounds

## Outcome

Integration is accepted. All four v6 bounds-proof rows have code, report
projection, validator, MiniMemoryModel, documentation, and verification
evidence.

## Accepted Results

- MemoryFactGraph projections accepted for
  `bounds_check_retained_dynamic`,
  `bounds_check_removed_with_proof_id`,
  `bounds_check_removal_rejected_missing_proof_id`, and
  `raw_bounds_runtime_check_normal_build`.
- Validators accepted for `bounds_proof_id_validator`,
  `raw_bounds_width_validator`, and
  `normal_build_bounds_check_validator`.
- MiniMemoryModel v6 cases accepted for proof-tagged removal, missing proof,
  mismatched proof, unsafe_unknown elimination rejection, retained dynamic
  checks, and raw overflow/width check-or-trap conservatism.
- Correlation validator accepts the exact four-row `MEM-BOUNDS-*` row set and
  v0-v5 regression correlation files.

## Conservative Or Rejected Results

- `MEM-BOUNDS-003` is rejected by design: `unsafe_unknown` cannot authorize
  eliminated bounds checks, `index_in_range`, or zero-cost bounds removal.
- `MEM-BOUNDS-004` is conservative by design: raw target-width and overflow
  uncertainty remains a normal-build check/trap or rejected/conservative row.

## Conflicts Resolved

The previous `GOAL.md` described completed v5. It was replaced with the v6
contract before implementation. Existing global `PLAN.md`, `ATTEMPTS.md`,
`NOTES.md`, and `CONTROL.md` belonged to Surface Release Promotion v1 and were
left untouched; v6 state lives under this workflow directory.

## Verification Evidence

- Focused memoryfacts passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v6-bounds-memoryfacts go test ./compiler/internal/memoryfacts -count=1`.
- Focused memorymodel passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v6-bounds-mini go test ./compiler/internal/memorymodel -count=1`.
- Focused validation passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v6-bounds-validation go test ./compiler/internal/validation -run 'Bounds|Proof|Memory' -count=1`.
- Focused lower passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v6-bounds-lower go test ./compiler/internal/lower -run 'Bounds|Proof|BCE|Unchecked' -count=1`.
- Focused semantics passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v6-bounds-semantics go test ./compiler/tests/semantics -run 'Memory|Borrow|Alias|Unsafe|Raw|Pointer|Bounds|Actor|Task|Async|Callback|Interface|Protocol|Fuzz' -count=1`.
- Focused tools passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v6-bounds-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation ./tools/cmd/validate-memory-fuzz-oracle -count=1`.
- V6 correlation validation passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v6-bounds-correlation go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory-ideal-vslice-v6-bounds-correlation.md`.
- V0-v6 correlation regression validation passed for
  `docs/audits/memory-ideal-vslice-v0-correlation.md` through
  `docs/audits/memory-ideal-vslice-v6-bounds-correlation.md`.
- Manifest validation passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v6-bounds-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`.
- Docs verification passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v6-bounds-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- Broad Go gate passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v6-bounds-broad go test ./compiler/... ./cli/... ./tools/... -count=1`.
- CI script gate passed and printed `OK` with artifact
  `tetra.release.v0_4_0.go-test-suite.v1`:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v6-bounds-ci bash scripts/ci/test.sh`.
- Whitespace check passed:
  `git diff --check`.
- Graphify refresh passed:
  `graphify update .`.

## Remaining Risks

No unresolved v6 blockers remain. This slice intentionally does not provide
"Memory 100% complete", broad optimizer correctness, target parity,
performance evidence, arbitrary unsafe pointer arithmetic proof, arbitrary
external pointer safety, an FFI lifetime model, or a full theorem prover.
