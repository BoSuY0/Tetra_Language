# Self-hosting Gate v1

Status: P23.3 evidence/report closure for the Ideal Master Plan.

Schema: `tetra.self_hosting.gate.v1`

Scope: `p23.3_self_hosting_gate`

## Summary

Self-hosting Gate v1 records the current promotion gate for any future self-hosting claim. It
defines the bounded evidence subset, records current backend/optimizer/allocator/runtime/stdlib
evidence, and keeps bootstrap work blocked until real Tetra compiler-component and deterministic
bootstrap artifacts exist.

The report is valid only while `SelfHostingClaimed=false` and `GateDecision.Allowed=false`.

## Coverage Rows

| Row                                | Evidence                                                                                                          | Boundary                                                                               |
| ---------------------------------- | ----------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| `self_host_subset_definition`      | Defines `p23.3_verified_subset_gate_not_self_hosted` as the current evidence gate.                                | This is not a Tetra compiler subset that compiles itself.                              |
| `small_compiler_component_compile` | Records the missing small compiler component blocker.                                                             | Go implementation tests do not count as a Tetra compiler component compile.            |
| `go_vs_tetra_output_comparison`    | Records the missing Go-vs-Tetra output comparison blocker.                                                        | No output equivalence is claimed.                                                      |
| `register_backend_stability`       | `differential.CheckBackendMatrix` covers source, Stack IR, optimized Stack IR, SSA, and Machine IR lanes.         | Internal supported-subset evidence only.                                               |
| `optimizer_validation_maturity`    | `BuildP23TranslationValidationV2` and its validator provide optimizer validation evidence.                        | Not exhaustive optimizer completeness.                                                 |
| `allocator_runtime_stability`      | `runtimeabi.RuntimeAllocationContracts`, region allocator config, and per-core small heap evidence are validated. | Not a complete self-host runtime.                                                      |
| `stdlib_sufficiency`               | `stdlibrt.RegionAwareStdlibCoverage` and its validator provide current stdlib evidence.                           | Sufficient for the gate layer only.                                                    |
| `deterministic_bootstrap_chain`    | Records the missing staged deterministic bootstrap blocker.                                                       | `scripts/dev/bootstrap.sh` is Go-built binary refresh evidence, not a self-host chain. |
| `cross_platform_bootstrap_story`   | Records the missing cross-platform bootstrap blocker.                                                             | Current native target evidence is not cross-platform self-host bootstrap evidence.     |
| `no_self_hosting_claim`            | `selfhostgate.Evaluate` returns blocked with the remaining bootstrap blockers.                                    | Future promotion must replace blocker rows with real evidence.                         |

## Validator Contract

`ValidateP23SelfHostingGateV1Report` rejects:

- wrong schema or scope;
- missing or duplicate rows;
- missing witness references;
- placeholder evidence;
- missing compiler subset evidence;
- missing register backend evidence;
- missing optimizer validation evidence;
- missing allocator/runtime evidence;
- missing stdlib evidence;
- fake self-hosting claims;
- fake small compiler component compile claims;
- fake Go-vs-Tetra output comparison claims;
- fake deterministic bootstrap claims;
- fake cross-platform bootstrap claims;
- runtime-behavior-change claims;
- safe-semantics-change claims;
- performance claims.

## Non-claims

- Tetra is not self-hosting.
- No Tetra compiler component is claimed to compile itself yet.
- No Go compiler output vs Tetra-compiled output equivalence is claimed yet.
- No deterministic bootstrap chain is claimed yet.
- No cross-platform bootstrap story is claimed yet.
- Runtime behavior does not change.
- Safe-program semantics do not change.
- No performance claim is made.

## Verification

Focused evidence:

```text
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler ./compiler/internal/selfhostgate -run 'P23SelfHostingGate|SelfHostGate|SelfHosting' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/selfhostgate ./compiler/internal/differential ./compiler/internal/runtimeabi ./compiler/internal/stdlibrt -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/tests/semantics -run 'FeatureRegistry' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
```
