# Fuzz / Property / Differential Expansion v1

Status: P23.1 evidence/report closure for the Ideal Master Plan.

Schema: `tetra.fuzz.property.differential.v1`

Scope: `p23.1_fuzz_property_differential`

## Summary

P23.1 records the current fuzz, property, and differential expansion as a
machine-checkable coverage ledger. It connects deterministic generated compiler
pipeline cases, Go fuzz/nightly infrastructure, backend differential matrix
randomized samples, first-mismatch reducer metadata, runtime allocator
properties, native-backend host boundaries, and actor-transfer stress
diagnostics.

## Coverage Rows

| Row | Evidence | Boundary |
| --- | --- | --- |
| `parser_checker_generated_programs` | Generated source snippets run through `compiler.Parse` and `compiler.Check`; existing Go fuzz targets mutate frontend and lowering seeds. | Bounded generated snippets, not exhaustive parser/checker proof. |
| `plir_lowering_verifier_pipeline` | The same generated snippets run `BuildPLIR`, `Lower`, and `VerifyIRProgram`. | Supported snippets and existing PLIR verifier coverage only. |
| `backend_differential_matrix_expansion` | `differential.CheckBackendMatrix` compares source, Stack IR, optimized Stack IR, SSA, and Machine IR with deterministic randomized samples. | Current stable i32 subset only. |
| `native_backend_boundary` | Linux x64 native lane is compared when the host supports it; otherwise an explicit unavailable boundary is recorded. | Not a full native differential suite for every target. |
| `runtime_allocator_properties` | `runtimeabi.AlignRegionBytes` accepts aligned valid sizes and rejects negative/overflow sizes. | Region ABI arithmetic, not a full allocator stress campaign. |
| `actor_transfer_stress_boundary` | `actorsafety.TypedActorOwnershipTransferCoverage` validates stress diagnostics and PLIR moved facts. | No distributed pointer/region zero-copy promotion. |
| `fuzz_nightly_summary_gate` | `scripts/dev/fuzz-nightly.sh` and `tools/cmd/validate-fuzz-summary` define required summary/log/unstable-seed artifacts. | Nightly long fuzz remains a separate bounded gate. |
| `reducer_failure_artifacts` | Backend matrix mismatch records `reduced_to_single_sample` and a reproducer string. | First-mismatch metadata, not a general-purpose reducer. |

## Validator Contract

`ValidateP23FuzzPropertyDifferentialReport` rejects:

- wrong schema or scope;
- missing or duplicate rows;
- missing witness references;
- placeholder evidence;
- missing parser/checker generated cases;
- missing PLIR/lowering verifier cases;
- missing backend matrix or randomized samples;
- missing reducer evidence;
- missing native-host sample or explicit non-host boundary;
- missing runtime allocator property evidence;
- missing actor-transfer stress diagnostics;
- missing fuzz summary artifacts or nightly boundary;
- fake full program correctness claims;
- fake exhaustive fuzzing claims;
- fake full native differential claims;
- fake performance claims;
- runtime behavior changes;
- safe-program semantics changes.

## Non-claims

- No full program correctness claim is made.
- No exhaustive fuzzing is claimed.
- No full native differential suite is claimed.
- No broad random program generator beyond bounded snippets is claimed.
- No performance claim is made.
- Runtime behavior does not change.
- Safe-program semantics do not change.

## Verification

Focused evidence:

```text
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler -run 'P23FuzzPropertyDifferential|ValidateP23FuzzPropertyDifferential' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/differential -run 'CheckBackendMatrix|Reducer' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/runtimeabi -run 'RegionAllocator|AlignRegionBytes' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/actorsafety -run 'TypedActorOwnershipTransfer' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./tools/cmd/validate-fuzz-summary -count=1
```
