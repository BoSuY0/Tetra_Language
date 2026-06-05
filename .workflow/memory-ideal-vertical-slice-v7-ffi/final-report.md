# Final Report: Memory Ideal Vertical Slice v7 FFI

Date: 2026-06-05

Decision: accepted

Status: `validated_narrow`

## Integration Summary

`MEM-FFI-007` is accepted as a narrow external pointer and FFI lifetime
quarantine evidence slice. It extends the v0-v6 Memory Ideal correlation
discipline with exactly four v7 requirement rows and keeps
`MemoryFactGraph` as the truth source. Reports remain projections.

No conflict was found with v6 bounds evidence. v7 does not claim "Memory 100%",
arbitrary external pointer safety, C/FFI lifetime safety, safe wrapper
promotion, broad unsafe noalias, target parity, performance, arbitrary external
allocator provenance, or full runtime/ABI proof.

## Row Classifications

| requirement_id | classification | report row | validator |
| --- | --- | --- | --- |
| `MEM-FFI-001` | conservative | `ffi_pointer_external_unknown` | `external_pointer_provenance_validator` |
| `MEM-FFI-002` | conservative | `ffi_call_may_retain_borrow` | `ffi_lifetime_conservative_validator` |
| `MEM-FFI-003` | rejected | `safe_wrapper_promotion_rejected_without_contract` | `safe_wrapper_promotion_validator` |
| `MEM-FFI-004` | conservative | `ffi_noalias_invalidated_by_external_call` | `ffi_noalias_conservative_validator` |

Supporting rejected evidence:
`external_pointer_provenance_rejected`.

## Accepted Packets

- Source facts/projections:
  `compiler/internal/memoryfacts/from_plir.go`,
  `compiler/internal/memoryfacts/validate.go`.
- Source tests:
  `compiler/internal/memoryfacts/from_plir_test.go`,
  `compiler/internal/memoryfacts/report_test.go`.
- MiniMemoryModel:
  `compiler/internal/memorymodel/mini.go`,
  `compiler/internal/memorymodel/mini_test.go`.
- Report/correlation tooling:
  `tools/cmd/validate-memory-report`,
  `tools/cmd/validate-memory-correlation`.
- Docs/schema:
  `docs/audits/memory-ideal-vslice-v7-ffi-correlation.md`,
  `docs/audits/memory-ideal-vslice-v7-ffi-final.md`,
  `docs/spec/memory_report_schema_v1.md`,
  `docs/design/memory_production_core_v1.md`,
  `docs/spec/unsafe.md`,
  `docs/generated/manifest.json`.

## Verification Evidence

RED evidence was observed first:

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v7-ffi-memoryfacts-red go test ./compiler/internal/memoryfacts -count=1` failed on missing v7 projections/parent discipline.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v7-ffi-mini-red go test ./compiler/internal/memorymodel -count=1` failed on missing v7 MiniMemoryModel symbols.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v7-ffi-tools-red go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation -count=1` failed on missing v7 parent discipline and correlation row set.

Focused gates passed:

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v7-ffi-memoryfacts go test ./compiler/internal/memoryfacts -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v7-ffi-mini go test ./compiler/internal/memorymodel -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v7-ffi-semantics go test ./compiler/tests/semantics -run 'Memory|Borrow|Alias|Unsafe|Raw|Pointer|Bounds|FFI|Extern|Callback|Interface|Protocol' -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v7-ffi-ownership go test ./compiler/tests/ownership -run 'Memory|Borrow|Alias|Unsafe|Raw|Pointer|FFI|Extern' -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v7-ffi-runtimeabi go test ./compiler/internal/runtimeabi -run 'Raw|Pointer|Bounds|External|FFI|Unknown' -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v7-ffi-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation ./tools/cmd/validate-memory-fuzz-oracle -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v7-ffi-correlation go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory-ideal-vslice-v7-ffi-correlation.md`

Docs and regression gates passed:

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v7-ffi-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v7-ffi-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`
- v0-v7 correlation regression passed by running
  `go run ./tools/cmd/validate-memory-correlation --file <correlation-doc>`
  for every `docs/audits/memory-ideal-vslice-v{0,1,2,3,4,5,6-bounds,7-ffi}-correlation.md`.

Full gates passed:

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v7-ffi-broad go test ./compiler/... ./cli/... ./tools/... -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v7-ffi-ci bash scripts/ci/test.sh`
  ended with `OK` and `Artifact: tetra.release.v0_4_0.go-test-suite.v1`.

Hygiene and graph gates passed:

- `graphify update .` rebuilt code graph: 21338 nodes, 66653 edges,
  1178 communities.
- `git diff --check` exited 0.
- `git status --short` was run and was non-empty. The worktree remains heavily
  dirty with many pre-existing modified and untracked files; therefore
  whitespace cleanliness is not a clean-worktree claim. Scoped v7 status showed
  modified `docs/generated/manifest.json`, `docs/spec/unsafe.md`,
  `graphify-out/*`, and untracked `GOAL.md`, v7 workflow docs,
  `compiler/internal/memoryfacts/`, `compiler/internal/memorymodel/`,
  `tools/cmd/validate-memory-report/`, `tools/cmd/validate-memory-correlation/`,
  `docs/spec/memory_report_schema_v1.md`,
  `docs/design/memory_production_core_v1.md`, and v7 audit docs.

## Caveats

- This is `validated_narrow`, not a broad memory-safety completion claim.
- Dirty worktree state remains a release checklist item if a clean release
  packet is required.
- v7 keeps FFI/external behavior conservative or rejected; it does not prove
  arbitrary C-side lifetime behavior.
