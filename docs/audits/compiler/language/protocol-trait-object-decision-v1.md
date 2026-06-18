# Protocol / Trait Object Decision v1

Status: P22.2 evidence/report decision.

Schema: `tetra.language.protocol_trait_object_decision.v1` Scope:
`p22.2_protocol_trait_object_decision` Decision: `keep_static_conformance_only`

P22.2 answers the master-plan question by keeping the current static protocol/conformance model in
this branch. Runtime protocol values, trait objects, witness tables, dynamic dispatch, and
conformance-table lookup are not promoted without a later same-branch ABI/lifetime/ownership design.

## Decision Rows

| Row                                 | Decision                  | Evidence                                                                                                                           | Boundary                                                                                                         |
| ----------------------------------- | ------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `static_conformance_fast_path`      | keep static               | `compareProtocolRequirement` validates `impl Type: Protocol`; the witness lowers a known direct `IRCall` to `Vec2.draw`.           | Static conformance remains the fast path; no runtime protocol values or dynamic dispatch are promoted.           |
| `static_protocol_bound_generics`    | keep static               | `validateGenericFuncDecl` checks protocol bounds during monomorphization; the witness records concrete `id__T_Vec2`.               | Protocol-bound generics do not add runtime generic values or requirement dispatch.                               |
| `runtime_existential_decision`      | not promoted              | Runtime protocol value witness rejects `let value: Drawable = ...` with `unknown type 'Drawable'`.                                 | Runtime existential ABI is not designed in this slice.                                                           |
| `explicit_dynamic_dispatch_gate`    | not promoted              | Generic-bound requirement calls such as `T.echo(x)` remain diagnostics.                                                            | Dynamic dispatch must be explicit and report-visible before promotion.                                           |
| `specialization_static_abstraction` | bounded existing evidence | P17.2 and P21.2 rows cover known direct static protocol/conformance calls and Machine IR without `OpCall` for the bounded witness. | No broad protocol specialization, witness-table removal, dynamic dispatch removal, or performance claim is made. |
| `witness_table_boundary`            | not promoted              | Current lowering emits no witness tables; existing optimizer reports mention them only as non-claims.                              | Witness tables require future ABI evidence.                                                                      |
| `trait_object_boundary`             | not promoted              | Protocols are not value types in the current checker.                                                                              | Trait objects require future ABI, lifetime, ownership, and report evidence.                                      |
| `registry_docs_alignment`           | keep static               | `FeatureRegistry()` keeps `language.protocol-conformance-mvp` and `language.protocol-bound-generics-static` static-only.           | Registry, docs, and manifest must preserve the same non-claims.                                                  |

## Validator Contract

`ValidateP22ProtocolTraitObjectDecision` rejects:

- missing or duplicate rows;
- missing witness references;
- placeholder evidence;
- any decision other than `keep_static_conformance_only`;
- runtime existential promotion claims;
- trait-object promotion claims;
- witness-table promotion claims;
- dynamic-dispatch promotion claims;
- conformance-table lookup promotion claims;
- runtime protocol value claims;
- broad protocol specialization claims;
- performance claims;
- runtime behavior changes;
- safe-program semantic changes.

## Non-claims

- Runtime protocol values are not promoted.
- Trait objects are not promoted.
- Witness tables are not promoted.
- Dynamic dispatch is not promoted.
- Conformance-table lookup is not promoted.
- Runtime existential ABI is not designed in this slice.
- Broad protocol specialization is not claimed.
- Performance is not claimed.
- Runtime behavior does not change.
- Safe-program semantics do not change.

## Verification

Focused evidence:

```text
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler -run 'P22ProtocolTrait|ValidateP22ProtocolTrait' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/tests/semantics -run 'ProtocolConformance|GenericFunctionProtocolBound|Plan250ProtocolConformance|FeatureRegistry' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/opt -run 'InliningSpecialization|SpecializationMachineCode' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
```
