# Safety Semantics Function Contract v1

`semantics.FuncSig` remains the compiler decision object. `FunctionContractV1` is a serialization DTO derived from that decision object; it is not checker state and must not be read back into source semantics except through the planned `.t4i` deserialization boundary.

## Schema

Schema string: `tetra.semantic.function-contract.v1`.

`FunctionContractV1` records function identity, generic/public bits, parameter contracts, result contract, optional typed-throw contract, async bit, canonical effects, semantic policy flags, mutable-global touch state, and a deterministic digest.

Parameter contracts include name, type, normalized ownership, and optional callable type metadata. Result contracts include type, normalized ownership, optional callable value metadata, region summary, resource summary, and unknown-summary flags. Throw contracts include the throws type and throw resource summary.

Callable type/value DTOs serialize the existing callable fields from `FuncSig` and `FunctionFieldInfo`: signature shape, effects, symbol/param-name metadata, captures, escape captures, mutable-global touch state, snapshot alias flags, escape kind, handle bit, function fields, and enum-payload callable maps. They do not change callable ABI constants or allow new capture behavior.

## Ownership

Internal empty ownership remains valid checker state. Projection normalizes it to explicit `owned` in the DTO. Other serialized ownership values are `borrow`, `inout`, and `consume`.

## Ordering

Before digest input is encoded:

- effects are sorted ascending;
- map keys are encoded by deterministic JSON object-key ordering;
- resource provenances are sorted by `ParamIndex`, then `ParamPath`;
- callable maps are copied by canonical key/name;
- captures preserve source declaration order.

Duplicate effects and duplicate resource provenances are rejected by `ValidateFuncSigContract`.

## Digest

`FunctionContractDigest` returns `sha256:<64 lowercase hex>`.

The digest input is canonical JSON for the DTO without the `Digest` field. The digest is then written back into the returned `FunctionContractV1`.

## Validation

`ValidateFuncSigContract` is fail-closed for malformed contract state. It rejects inconsistent parameter-array lengths, unknown ownership markers, duplicate/non-canonical effects, invalid region/resource parameter indexes, non-canonical summary paths, duplicate provenances, unsupported callable escape kinds, callable handle returns with non-handle slot counts, throw resource summaries without `ThrowsType`, negative budgets, realtime without both noalloc and noblock, and public generic export inconsistency.

## Nonclaims

This schema does not infer new effects, does not expand callable capture support, does not change `FnPtrEnvSlotCount`, `FnPtrSlotCount`, or `CallableHandleSlotCount`, and does not make the DTO mutable checker state.
