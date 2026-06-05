# Final Report: Memory Ideal Vertical Slice v5

## Outcome

Integration is accepted. All four v5 rows have code, report projection,
validator, MiniMemoryModel, documentation, and verification evidence.

## Accepted Results

- MemoryFactGraph projections accepted for
  `unsafe_unknown_rejected_safe_facts`,
  `unsafe_verified_root_allocation_base`,
  `unsafe_contract_runtime_checkable`, and
  `unsafe_contract_static_untrusted`.
- Validators accepted for `unsafe_unknown_fact_validator`,
  `unsafe_verified_root_bounds_validator`,
  `unsafe_runtime_contract_validator`, and
  `unsafe_static_contract_validator`.
- MiniMemoryModel v5 cases accepted for verified root bounds, runtime-checkable
  unsafe contracts, unknown pointer safe/noalias rejection, static-untrusted
  noalias/lifetime/region contracts, external unknown raw slices, and
  too-large verified-root raw slices.
- Semantics tests accepted for the current supported raw pointer unsafe gateway
  surface.

## Conservative Or Rejected Results

- `MEM-UNSAFE-001` is rejected by design: `unsafe_unknown` raw pointers cannot
  project safe-known, provenance-known, or noalias facts.
- `MEM-UNSAFE-004` is conservative by design: unsafe noalias/lifetime/region
  contracts remain `unsafe_contract_static_untrusted` unless separately proven.

## Conflicts Resolved

The previous `GOAL.md` described completed v4 while the active thread goal
described v5. `GOAL.md` was replaced with the v5 contract before code edits.

## Verification Evidence

- Focused memoryfacts passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v5-memoryfacts go test ./compiler/internal/memoryfacts -count=1`.
- Focused memorymodel passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v5-mini go test ./compiler/internal/memorymodel -count=1`.
- Focused tools passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v5-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation -count=1`.
- Focused semantics passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v5-semantics go test ./compiler/tests/semantics -run 'MemoryIdealV5|Unsafe|Raw|Pointer|Alloc|Slice|Bounds' -count=1`.
- V5 correlation validation passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v5-correlation go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory-ideal-vslice-v5-correlation.md`.
- Manifest validation passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v5-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`.
- Docs verification passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v5-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- Broad Go gate passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v5-broad go test ./compiler/... ./cli/... ./tools/... -count=1`.
- CI script gate passed and printed `OK` with artifact
  `tetra.release.v0_4_0.go-test-suite.v1`:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v5-ci bash scripts/ci/test.sh`.
- Whitespace check passed:
  `git diff --check`.
- Graphify refresh passed:
  `graphify update .`.

## Remaining Risks

No unresolved v5 blockers remain. This slice intentionally does not provide
arbitrary external pointer safety, an FFI lifetime system, broad unsafe
noalias, safe wrapper promotion, actor/task/runtime expansion, target parity,
or performance evidence.
