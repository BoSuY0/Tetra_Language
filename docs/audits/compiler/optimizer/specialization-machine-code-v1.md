# Specialization Machine-Code Evidence v1

Status: P21.2 closed as bounded evidence.

Schema: `tetra.optimizer.specialization_machine_code.v1`

Scope: `p21.2_specialization_v1_v2`

## Coverage Rows

| Target                      | Status               | Evidence boundary                                                                                                                              |
| --------------------------- | -------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| Generics                    | `implemented_narrow` | Monomorphized generic identity/wrapper calls become concrete Stack IR calls; accepted tiny helpers disappear through `inline-small-pure`.      |
| Protocol/static conformance | `implemented_narrow` | Statically checked protocol impl calls that lower to known direct Stack IR symbols may disappear.                                              |
| Extension methods           | `implemented_narrow` | Statically resolved extension methods that lower to direct Stack IR symbols may disappear.                                                     |
| Enum known cases            | `implemented_narrow` | Locally constructed known enum tags fold discriminator branches through `sccp-constant-branch`.                                                |
| Optionals                   | `implemented_narrow` | Locally constructed proven-some optionals fold presence branches through `sccp-constant-branch`.                                               |
| Collections                 | `implemented_narrow` | P19.1 caller-owned `Vec<T>`/`HashMap<K,V>` source helpers monomorphize; only bounded concrete helper calls are covered by the machine witness. |

## Machine Witness

`BuildP21SpecializationMachineCodeWitness` constructs a tiny known direct-call program where `main`
calls `known_i32_add`.

The witness records:

- the call exists before `inline-small-pure`;
- the call is absent from optimized Stack IR after translation-validated inlining;
- `machine.ScalarIntFunctionFromStackIR` lowers the optimized `main` to verified scalar Machine IR;
- the Machine IR contains `mov`, `add`, and `return`, and no `OpCall`.

This is a narrow machine-code evidence witness for known direct helpers. It does not claim that
every source-level abstraction in the language is erased in all machine backends.

## Validator

`ValidateSpecializationMachineCodeCoverage` rejects:

- wrong schema or scope;
- missing or duplicate target rows;
- missing source, optimized IR, or Machine IR evidence;
- placeholder evidence;
- missing removed-marker or machine-witness facts;
- fake broad specialization;
- fake dynamic dispatch;
- fake runtime generic values;
- fake allocator-backed generic collections;
- fake layout/ABI freedom;
- fake performance;
- fake safe-semantics changes.

## Non-Claims

- No public optimizer mode is added.
- No runtime behavior changes are made.
- No broad specialization completeness is claimed.
- No dynamic protocol dispatch, witness-table, trait-object, or conformance-table lookup removal is
  claimed.
- No runtime generic values are claimed.
- No allocator-backed production `Vec<T>`/`HashMap<K,V>` runtime is claimed.
- No layout/ABI freedom is claimed by this slice.
- No performance claim is made.

## Verification

RED evidence:

```sh
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/opt -run 'TestP21SpecializationMachineCode' -count=1
```

Initial RED failed because `SpecializationMachineCodeCoverage`,
`ValidateSpecializationMachineCodeCoverage`, `SpecializationMachineCodeID`, the P21.2 row constants,
and `SpecializationMachineCodeCoverageReport` did not exist.

Focused GREEN:

```sh
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/opt -run 'TestP21SpecializationMachineCode' -count=1
```

Feature/docs/manifest linkage:

```sh
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/tests/semantics -run 'TestFeatureRegistryCoversReleaseStatusesAndKeyBoundaries' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

Relevant package gate:

```sh
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/opt ./compiler/internal/machine ./compiler/internal/stdlibrt ./compiler/tests/semantics ./compiler -count=1
```
