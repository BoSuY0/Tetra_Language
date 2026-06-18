# Memory Ideal Vertical Slice v9 Storage Final Audit

Status: `validated_narrow` for the bounded escape-aware storage/lowering surface.

Decision: proceed for v9 evidence. This slice accepts only storage/lowering integrity around
compiler-owned escape evidence. It is not "Memory 100%", not full region inference, not
optimizer-wide allocation correctness, not target parity, not performance evidence, not production
actor runtime proof, not a full async lifetime system, and not arbitrary FFI or external pointer
safety.

## Requirement Results

| requirement_id    | status             | evidence                                                                                                                                                                                                                                                                                                                 |
| ----------------- | ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `MEM-STORAGE-001` | `rejected`         | `allocplan.VerifyPlan` rejects escaped values whose actual lowering storage, planned storage, or storage claim uses trusted stack/register/region/function-temp/island/task/actor storage; negative test: `TestVerifyPlanRejectsEscapedActualTrustedLowering`.                                                           |
| `MEM-STORAGE-002` | `validated_narrow` | Trusted stack/region-style storage validates only with a compiler-owned no-escape proof status such as `validated_no_escape`, `validated_region_scope`, `validated_function_temp_region_scope`, or `validated_explicit_island_scope`; negative test: `TestVerifyPlanRejectsTrustedStorageWithoutNoEscapeProof`.          |
| `MEM-STORAGE-003` | `validated_narrow` | Heap/conservative fallback rows preserve `source_fact_id` through report schema validation and now require a reviewable `reason` in `FromPLIRAndAllocPlan` and `ValidateReport`; negative tests: `TestFromPLIRAndAllocPlanRejectsHeapFallbackWithoutReason`, `TestValidateMemoryReportRejectsHeapFallbackWithoutReason`. |
| `MEM-STORAGE-004` | `conservative`     | Task, actor, FFI, and unknown-call storage boundaries stay heap/conservative unless a later narrow proof exists; MiniMemoryModel case coverage: `TestMiniMemoryModelV9StorageCases`.                                                                                                                                     |

## Validator Map

| validator                                 | implementation                                                                                                                                                                       |
| ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `storage_escape_validator`                | `compiler/internal/allocplan.VerifyPlan` checks planned, reported, and actual lowering storage against escape class.                                                                 |
| `storage_no_escape_proof_validator`       | `compiler/internal/allocplan.storageHasCompilerOwnedNoEscapeProof` admits only matching compiler-owned proof statuses for trusted storage.                                           |
| `heap_fallback_reason_validator`          | `compiler/internal/memoryfacts.addAllocPlanFacts` and `compiler/internal/memoryfacts.ValidateReport` reject trusted-storage heap fallbacks without `reason`.                         |
| `boundary_storage_conservative_validator` | `compiler/internal/allocplan.chooseStorage`, `VerifyPlan`, and `compiler/internal/memorymodel.evaluateStorage` keep async/task/actor/FFI/unknown-call boundary storage conservative. |
| `correlation_exact_row_validator`         | `tools/cmd/validate-memory-correlation` v9 required row set and status checks.                                                                                                       |

## RED Evidence

Focused RED was observed before implementation:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-allocplan-red go test ./compiler/internal/allocplan -run 'Storage|Escape|Region|Heap|Lower' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-memoryfacts-red go test ./compiler/internal/memoryfacts -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-mini-red go test ./compiler/internal/memorymodel -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-tools-red go test ./tools/cmd/validate-memory-correlation -count=1
```

The RED failures showed that escaped actual trusted lowering, trusted storage without no-escape
proof, heap fallback without reason, missing MiniMemoryModel storage outcomes, and unknown
`MEM-STORAGE-*` correlation rows were accepted or unsupported before v9.

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

## Current Gate Evidence

Focused GREEN has passed:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-allocplan go test ./compiler/internal/allocplan -run 'Storage|Escape|Region|Heap|Lower' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-memoryfacts go test ./compiler/internal/memoryfacts -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-mini go test ./compiler/internal/memorymodel -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-tools go test ./tools/cmd/validate-memory-correlation -count=1
```

Final broad, docs, manifest, CI, hygiene, dirty-worktree, and Graphify evidence is recorded in
`.workflow/memory-ideal-vertical-slice-v9-storage/final-report.md`.
