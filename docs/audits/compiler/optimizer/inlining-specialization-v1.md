# Inlining / Specialization v1

Status: P17.2 closed as bounded evidence-backed coverage for the Ideal Master Plan.

## Summary

P17.2 has a machine-readable inlining/specialization coverage matrix:
`compiler/internal/opt.InliningSpecializationCoverage()` emits schema
`tetra.optimizer.inlining_specialization.v1` and lists every P17.2 target.

The implemented slice is intentionally narrow:

- `inline-small-pure` inlines small straight-line Stack IR callees with one return slot, no
  unsupported effects, no proof-sensitive instructions, and at most 8 candidate body instructions.
- Report rows include `inlined` and `not_inlined` decisions. Accepted generic wrapper call sites
  report `reason: "small_pure_wrapper"`; nested leaf calls still report `reason: "small_pure"`.
- Translation validation remains required through the P17.0 pass contract.
- Generic function support is limited to statically monomorphized concrete calls. The new P17.2
  behavior removes a tiny monomorphized generic wrapper that forwards to an already small-pure
  monomorphized callee.
- Static protocol/conformance call support is limited to statically checked `impl` methods whose
  concrete call has already lowered to a known direct Stack IR function symbol and whose method body
  is accepted by `inline-small-pure`.
- Extension call support is limited to statically resolved extension method calls that lower to
  direct Stack IR function symbols and whose concrete extension method body is accepted by
  `inline-small-pure`.
- Known-case enum support is limited to a locally constructed payload enum case whose lowered Stack
  IR tag constant is tracked through same-basic-block stores and then used to fold the lowered match
  discriminator branch through `sccp-constant-branch`.
- Proven-some optional support is limited to a locally constructed optional value whose lowered
  presence tag constant is tracked through same-basic-block stores and then used to fold the lowered
  `some` match branch through `sccp-constant-branch`.

## Coverage Rows

| Target                                    | Status             | Evidence                                                                                                                   |
| ----------------------------------------- | ------------------ | -------------------------------------------------------------------------------------------------------------------------- |
| Generic functions                         | implemented narrow | `compiler/tests/semantics/semantics_types_protocols_test.go::TestP17GenericWrapperDisappearsAfterSmallPureInlining`        |
| Small pure functions                      | implemented narrow | `compiler/internal/opt/opt_suite_test.go::TestInlineSmallPurePassInlinesCallAndReportsDecision`                            |
| Static protocol/conformance calls         | implemented narrow | `compiler/tests/semantics/semantics_callables_closures_test.go::TestP17StaticProtocolConformanceCallInlinesAfterSmallPure` |
| Extension calls                           | implemented narrow | `compiler/tests/semantics/semantics_callables_closures_test.go::TestP17StaticExtensionCallInlinesAfterSmallPure`           |
| Enum constructors/matches with known case | implemented narrow | `compiler/tests/semantics/semantics_callables_closures_test.go::TestP17KnownEnumPayloadMatchFoldsAfterSCCP`                |
| Optional unwrap proven some               | implemented narrow | `compiler/tests/semantics/semantics_callables_closures_test.go::TestP17ProvenSomeOptionalMatchFoldsAfterSCCP`              |

## Boundaries

This is not a broad specialization optimization claim. It does not claim runtime generic values,
explicit type arguments, generic structs, protocol-bound requirement calls, witness tables, trait
objects, runtime protocol values, dynamic dispatch, conformance-table lookup, dynamic extension
dispatch, receiver-call sugar specialization, enum payload escape rewrites, exhaustive match
pruning, broad optional elimination, unsafe unwrap removal, cross-control-flow optional fact
propagation, none-branch pruning, or code-size growth beyond the existing small-pure body cap.
