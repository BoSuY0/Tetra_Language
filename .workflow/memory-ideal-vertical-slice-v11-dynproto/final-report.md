# Final Report: Memory Ideal Vertical Slice v11 Dynamic Protocol

Date: 2026-06-06

Decision: accepted

Status: validated_narrow

## Summary

`MEM-DYNPROTO-011` adds a narrow dynamic protocol / witness-table memory
conservatism slice to the existing v0-v10 Memory Ideal evidence chain.
`MemoryFactGraph` remains the truth source, `tetra.memory-report.v1` remains a
projection, and the slice does not add full trait-object/existential runtime
proof, complete witness-table ABI proof, production dynamic dispatch runtime
safety, target parity, performance, broad noalias, arbitrary unsafe/external
pointer promotion, clean-release, or "Memory 100%" claims.

## Integration Result

| Requirement | Status | Integration summary |
| --- | --- | --- |
| `MEM-DYNPROTO-001` | `conservative` | Dynamic existential/protocol borrow carriers remain conservative unless statically resolved. |
| `MEM-DYNPROTO-002` | `validated_narrow` | Static witness/conformance proof may carry borrow facts only with compiler-owned parent evidence. |
| `MEM-DYNPROTO-003` | `rejected` | Dynamic protocol dispatch cannot validate broad noalias. |
| `MEM-DYNPROTO-004` | `rejected` | Witness/conformance lookup cannot promote unsafe/dynamic/unknown provenance to `safe_known`. |
| `MEM-DYNPROTO-005` | `validated_narrow` | Protocol/existential dispatch report rows preserve `source_fact_id`, `cost_class`, and `normal_build_check`. |

## Files

- `compiler/internal/memorymodel/mini.go`
- `compiler/internal/memorymodel/mini_test.go`
- `compiler/internal/memoryfacts/from_plir.go`
- `compiler/internal/memoryfacts/from_plir_test.go`
- `compiler/internal/memoryfacts/report.go`
- `compiler/internal/memoryfacts/report_test.go`
- `compiler/internal/memoryfacts/validate.go`
- `tools/cmd/validate-memory-correlation/main.go`
- `tools/cmd/validate-memory-correlation/main_test.go`
- `docs/audits/memory-ideal-vslice-v11-dynproto-correlation.md`
- `docs/audits/memory-ideal-vslice-v11-dynproto-final.md`
- `docs/spec/memory_report_schema_v1.md`
- `docs/design/memory_production_core_v1.md`
- `docs/generated/manifest.json`
- `graphify-out/GRAPH_REPORT.md`
- `graphify-out/graph.json`
- `graphify-out/manifest.json`
- `.workflow/memory-ideal-vertical-slice-v11-dynproto/*`
- `GOAL.md`

## RED Evidence

These gates failed before implementation for the intended reasons:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v11-dynproto-tools-red go test ./tools/cmd/validate-memory-correlation -run 'V11|DynProto|Dynamic|Protocol|Witness|Conformance' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v11-dynproto-mini-red go test ./compiler/internal/memorymodel -run 'V11|DynamicProtocolWitness' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v11-dynproto-memoryfacts-red go test ./compiler/internal/memoryfacts -run 'V11|DynamicProtocolWitness' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v11-dynproto-report-red go test ./compiler/internal/memoryfacts -run 'V11.*Report|V11Derived|ProtocolDispatchIntegrity' -count=1
```

Observed RED failures:

- `validate-memory-correlation` treated `MEM-DYNPROTO-*` rows as unexpected v0
  rows.
- `MiniMemoryModel` lacked v11 dynamic protocol / witness vocabulary.
- `MemoryFactGraph` did not project v11 dynamic protocol report rows.
- Standalone `ValidateReport` allowed `protocol_dispatch_report_integrity`
  without the required `cost_class` / `normal_build_check` fields.

## GREEN Evidence

Focused gates passed:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v11-dynproto-memoryfacts go test ./compiler/internal/memoryfacts -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v11-dynproto-mini go test ./compiler/internal/memorymodel -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v11-dynproto-semantics go test ./compiler/tests/semantics ./compiler/tests/ownership -run 'Memory|Borrow|Alias|Interface|Protocol|Witness|Conformance|Dynamic|Existential|Unsafe|Raw|Pointer' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v11-dynproto-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation ./tools/cmd/validate-memory-fuzz-oracle -count=1
```

Correlation/docs gates passed:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v11-dynproto-correlation go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory-ideal-vslice-v11-dynproto-correlation.md
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v11-dynproto-regression bash -lc 'for f in docs/audits/memory-ideal-vslice-v*-correlation.md; do go run ./tools/cmd/validate-memory-correlation --file "$f"; done'
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v11-dynproto-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v11-dynproto-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

Full gates passed:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v11-dynproto-broad go test ./compiler/... ./cli/... ./tools/... -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v11-dynproto-ci bash scripts/ci/test.sh
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
`graphify update .` rebuilt `21397 nodes`, `66817 edges`, and
`1189 communities`.

## Nonclaims

- No "Memory 100% complete" claim.
- No full trait-object/existential runtime proof.
- No complete witness-table ABI safety proof.
- No production dynamic dispatch runtime safety claim.
- No target parity.
- No performance claim.
- No broad noalias.
- No arbitrary unsafe/external pointer promotion.
- No clean-release claim while `git status --short` remains dirty.
