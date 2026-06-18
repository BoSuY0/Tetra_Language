# Inout Writer Internal ABI Contract

Status: design contract for a future implementation packet.

This document defines a narrow internal compiler/backend contract for promoting
proven `inout []u8 -> Int` writer helpers. It is not a public language ABI and
does not widen SysV or Win64 aggregate returns.

## 1. Status And Non-Claims

Accepted scope:

- This is an internal contract between lowering, Machine IR recognition, x64
  register emission, and backend reports.
- The source-level helper remains an ordinary function with `inout` mutation
  semantics and one visible scalar `Int` return.
- Hidden `inout` writeback slots may appear in Stack IR and proof/reporting
  metadata, but a promoted register path must treat them as compiler-owned
  writeback state, not public return registers.

Non-claims:

- This does not change public SysV or Win64 function-call ABI rules.
- This does not make generic multi-slot returns register-native.
- This does not allow arbitrary aggregate returns, slice returns, or `String`
  returns to bypass the existing fallback policy.
- This does not claim JSON or HTTP rows are native today. In the current Tier 1
  sidecars, JSON and HTTP remain fallback-only.
- This does not authorize name-only benchmark promotion. A helper must satisfy
  the proof family and call-site rules below.

## 2. Problem Statement

Lowering already represents `inout` by appending hidden writeback slots to the
function ABI shape. For a source helper like:

```tetra
func write_x(dst: inout []u8) -> Int:
    ...
```

the visible return is the scalar `Int`, but a `[]u8` writeback contributes two
hidden slots. The lowered Stack IR function therefore has `ReturnSlots=3`:

- slot 0: visible scalar `Int` result;
- slots 1-2: hidden writeback copy of the updated `dst` slice header.

Current lowering emits this shape in two places:

- callee lowering computes `abiReturnSlots = effectiveReturnSlots +
  inoutReturnSlotCount(...)` and stores that in `IRFunc.ReturnSlots`;
- caller lowering computes `IRCall.RetSlots = sig.ReturnSlots +
  inoutWritebackSlotCount(...)`, then immediately writes hidden slots back to
  the caller lvalue and leaves only the visible scalar result for expression
  consumers.

That target-neutral Stack IR representation is valid, but the generic register
call ABI is still scalar-only. `machine.SysVCallABIInfo()` has
`MaxRetSlots=1`; `machine.Win64CallABIInfo()` also has `MaxRetSlots=1`.
Generic x64 call emission rejects calls whose `RetSlots` exceed that limit, and
`buildreports` correctly classifies unverified `ReturnSlots > 2` functions as
`unsupported_aggregate_return`.

The narrow problem is therefore:

- keep the existing target-neutral hidden writeback representation;
- prove a helper/caller pair where the hidden slots are compiler-internal
  writeback state;
- emit only the visible scalar result through the native return register;
- keep all generic calls and aggregate returns on the existing fallback path.

## 3. Current Evidence

PostgreSQL is the only proven row-level native path today.

Fresh P72 evidence:

- `p25.postgresql_single_multiple_update` backend sidecar:
  `function_count=6`, `register_path=6`, `stack_fallback=0`.
- `write_i32_be_at` and `write_i16_be_at` have `return_slots=3`, but report
  `multi_slot_return_policy=single_slot_register_return`,
  `value_class=single_register_slot`, and
  `boundary_status=register_return_verified`.
- The promoted details are `machine-ir-postgresql-inout-writer` for helpers and
  `machine-ir-postgresql-inout-writer-main` for the row-local caller.

JSON and HTTP are candidate proof families only.

Fresh sidecar evidence:

- `p25.json_parse_stringify.write_message_object` has source/PLIR shape
  `dst: inout []u8 -> i32` and 27 `proof:helper-summary:` proof uses, but its
  backend row is `unsupported_aggregate_return` with `return_slots=3`.
- `p25.json_parse_stringify.main` is `unsupported_call_abi` because it calls the
  writer with `ret_slots=3` while the target call ABI has `max_ret_slots=1`.
- `p25.http_plaintext_json.write_plaintext_response` has 24
  `proof:helper-summary:` proof uses and `write_json_response` has 21, but both
  helpers remain `unsupported_aggregate_return`.
- `p25.http_plaintext_json.main` remains `unsupported_call_abi`.

Conclusion: P72 proves the exact PostgreSQL offset-writer slice. P77 and P78
show JSON/HTTP share the source shape and proof-family direction, but do not
yet prove the backend ABI boundary.

## 4. Internal ABI Contract

### Callee Obligations

A promoted helper must satisfy all of these:

- Visible return type is scalar `Int`/`i32` and contributes exactly one visible
  return slot.
- Hidden writeback slots arise only from `inout []u8` parameters recognized by
  lowering. For the current writer family this means `ReturnSlots=3`.
- The helper body writes bytes only through accepted `IRIndexStoreU8` writer
  stores covered by the selected proof family.
- The helper returns the exact scalar byte-count/next-offset shape required by
  its proof family.
- The helper does not rely on target public multi-return registers. Under
  register promotion, only the visible scalar result is returned in the native
  integer return register; `inout` mutation is represented by stores into the
  caller-provided buffer.
- The helper does not escape, retain, or publish the `inout` slice header or
  base pointer.

### Caller Obligations

A promoted caller must satisfy all of these:

- It is row-local or otherwise proven to be a closed caller slice for the exact
  helpers being promoted.
- Every promoted `IRCall` target is already accepted by the helper recognizer.
- `IRCall.RetSlots` may be `3` only for accepted writer helpers. Any other
  multi-slot call in the caller keeps the caller on fallback.
- The `inout` argument resolves to a writable caller-owned lvalue. Lowering's
  `collectInoutWritebacks` must be able to resolve the writeback destination.
- After the call, range facts that depend on the mutated `inout` target must be
  invalidated, matching the existing `invalidateWhileRangeProofsForInoutArgs`
  behavior.

### Hidden Writeback Representation

The target-neutral representation remains:

- callee `IRFunc.ReturnSlots = visibleReturnSlots + hiddenInoutWritebackSlots`;
- caller `IRCall.RetSlots = callee.ReturnSlots`;
- caller-side lowering emits writeback stores for hidden slots and exposes only
  `sig.ReturnSlots` visible slots to surrounding expressions.

The promoted machine/backend representation is:

- accepted helper plans reinterpret the hidden slots as internal writeback
  evidence, not as physical return values;
- x64 emission performs the proven stores directly into the caller-owned buffer;
- x64 emission returns only the scalar `Int` in `RAX`/`EAX`;
- report metadata records the promoted helper as a verified internal ABI
  exception, not as generic aggregate-return support.

### Target-Neutral Responsibilities

Lowering and Machine IR recognition must:

- preserve the existing hidden writeback slot accounting;
- verify call signatures in Stack IR before backend selection;
- identify accepted writer helpers from IR shape and proof tags;
- identify accepted row-local callers from exact call graph shape;
- reject mixed or ambiguous callers before backend emission;
- produce a target-independent plan containing parameter slots, store offsets
  or constant indexes, proof IDs, store count, and scalar return shape.

### Backend-Specific Responsibilities

x64 emission and ABI packages must:

- keep generic `SysVUnix.EmitCall` and `Win64.EmitCall` rejecting unsupported
  `RetSlots > MaxRetSlots`;
- keep scalar-call helpers rejecting generic `RetSlots > 1`;
- add direct emission only for the accepted internal writer plan;
- use the target ABI only for ordinary parameter passing/spilling and frame
  setup;
- return the visible scalar in the normal integer return register;
- never expose hidden writeback slots as public target return registers.

## 5. Proof Families

### `helper-offset`

Used for PostgreSQL-style offset writers such as `write_i32_be_at` and
`write_i16_be_at`.

Required shape:

- Store op kind: every promoted byte write is `IRIndexStoreU8`.
- Store target: base and length come from the accepted `dst` slice locals.
- Index shape: `start` or `start + const`.
- Offsets are dense and exact for the helper width:
  - i32 big-endian writer: offsets `0,1,2,3`, scalar return `start + 4`;
  - i16 big-endian writer: offsets `0,1`, scalar return `start + 2`.
- Every promoted store has a `ProofID` starting with `proof:helper-offset:`.
- The helper rejects extra calls, labels, jumps, dynamic stores, untagged
  stores, and non-final return shapes.

### `helper-summary`

Used for JSON/HTTP-style constant-index writers such as
`write_message_object`, `write_plaintext_response`, and `write_json_response`.

Required shape for a future implementation:

- Store op kind: every promoted byte write is `IRIndexStoreU8`.
- Store target: base and length come from the accepted `dst` slice locals.
- Index shape: compile-time constant indexes only.
- Store coverage: the recognizer records an exact store count and index set.
  Current candidates show 27 JSON stores, 24 HTTP plaintext stores, and 21 HTTP
  JSON stores.
- Every promoted store has a `ProofID` starting with `proof:helper-summary:`.
- Scalar return is an exact constant byte count for the helper.
- Any dynamic index, missing proof tag, extra write family, or uncertain return
  expression keeps the helper on fallback.

## 6. Call-Site Safety

Promotion is allowed only when the call site proves the `inout` mutation is
local and owned.

Rules:

- Row-local caller: the caller must be recognized as the exact benchmark row or
  an equivalent closed caller slice. Do not promote a helper merely because its
  signature matches.
- Caller-owned writable buffer: the `inout []u8` argument must resolve to a
  local/global lvalue that lowering can write back. Temporaries, computed
  non-lvalues, borrowed read-only data, escaped buffers, or externally owned
  data are not accepted.
- Mixed safe/unsafe rejection: if a caller has one accepted writer call and one
  unaccepted multi-slot call, the whole caller remains fallback.
- Lifetime/noescape: the helper may write into the buffer during the call, but
  must not store the buffer header/base pointer anywhere, return it publicly, or
  pass it to unverified callees.
- Proof invalidation: after an `inout` call, caller range proofs for that target
  must be invalidated. New proofs are required for later checked/unchecked
  indexed operations.

## 7. Backend And Report Integration Plan

Machine recognition should live in `compiler/internal/machine`.

Implementation direction:

- Keep the existing PostgreSQL recognizer as the first accepted instance.
- Introduce a target-neutral writer contract plan before x64 emission is
  generalized. The plan should make proof family explicit, for example
  `helper-offset` or `helper-summary`.
- The plan should carry enough data for emission and reporting:
  function name, proof family, visible return slots, hidden writeback slots,
  `dst` base/length slots, store kind, store indexes/offsets, proof IDs, return
  kind, and accepted call targets.
- A row-local caller plan should be separate from a helper plan. It should
  reject unaccepted multi-slot calls instead of asking the generic call ABI to
  handle them.

x64 emission should live in `compiler/internal/backend/x64core`.

Implementation direction:

- Do not change `compiler/internal/backend/x64abi/sysv_unix.go` or
  `compiler/internal/backend/x64abi/win64.go` to accept public multi-slot
  returns.
- Add direct x64core emission for accepted writer plans only.
- Use existing target ABI parameter spilling (`abi.SpillParams`) and normal
  frame setup.
- Emit stores from the proven plan and return the scalar result in `RAX/EAX`.
- Keep generic scalar call emission rejecting `RetSlots > 1`.

Report integration should live in `compiler/internal/buildreports/backend.go`.

Implementation direction:

- Replace the current PostgreSQL-only ABI special case with an exact internal
  writer ABI predicate only after the generic proof-family plan exists.
- Report accepted helpers as verified internal promotion, using the existing
  status vocabulary:
  - `multi_slot_return_policy=single_slot_register_return`;
  - `value_class=single_register_slot`;
  - `boundary_status=register_return_verified`.
- Preserve `unsupported_aggregate_return` for any `ReturnSlots > 2` function
  not accepted by the exact internal writer contract.
- Preserve `unsupported_call_abi` for any caller whose `IRCall.RetSlots`
  exceeds the target `MaxRetSlots` without an accepted row-local caller plan.
- Include proof family/detail in machine path names or report detail, for
  example `machine-ir-inout-writer-helper-offset` and
  `machine-ir-inout-writer-helper-summary`, only once implemented.

## 8. Negative Cases

These must stay fallback:

- Dynamic indexes in helper stores.
- Missing or malformed `proof:helper-offset:` / `proof:helper-summary:` tags.
- Any promoted store kind other than `IRIndexStoreU8`.
- Wrong `ReturnSlots`, including helpers that are not exactly one visible scalar
  return plus the expected hidden writeback slots.
- Non-scalar visible returns, including slice, `String`, tuple, or aggregate
  results.
- Return expressions that do not match the proof family:
  `start + N` for offset writers, exact constant `N` for summary writers.
- Helpers that call unverified helpers, allocate, branch in unrecognized ways,
  or contain extra stores outside the proof family.
- Public helper assumptions: a helper being source-visible or marked public in
  PLIR does not make the internal ABI public or externally callable.
- Generic aggregate returns such as unrelated `ReturnSlots > 2` functions.
- Generic multi-slot call ABI attempts through `SysVUnix.EmitCall`,
  `Win64.EmitCall`, or scalar-call emission.
- Mixed safe/unsafe callers where any multi-slot call is not accepted by the
  internal writer caller plan.

## 9. Test Matrix

PostgreSQL stability:

- Keep P72 machine tests covering `PostgreSQLInoutWriterPlanFromStackIR` and
  row-local `main` recognition.
- Keep x64core tests proving `write_i32_be_at`, `write_i16_be_at`, and
  PostgreSQL `main` emit through register paths.
- Keep compiler/backend report tests proving:
  `register_path=6`, `stack_fallback=0`, helper details are
  `machine-ir-postgresql-inout-writer`, and generic aggregate-return rows remain
  fallback.
- Keep fresh Tier 1 validation proving the PostgreSQL row remains measured with
  zero heap and zero bounds while the backend sidecar is register-native.

Future JSON/HTTP:

- Add RED recognizer tests for `helper-summary` before routing backend
  promotion. They should prove store count, constant indexes, proof tags, and
  scalar constant return shape.
- Add GREEN helper recognizer tests without changing public ABI limits.
- Add row-local caller tests proving accepted JSON/HTTP callers reject mixed
  safe/unsafe multi-slot calls.
- Only after recognizer and caller tests pass, add x64core emission tests for
  `helper-summary` plans.
- Only after x64core tests pass, update buildreport tests so JSON/HTTP move
  from fallback to verified internal ABI promotion.

Negative tests:

- Dynamic index store: fallback.
- Missing proof tag: fallback.
- Wrong proof family prefix: fallback.
- Wrong return slot count: fallback.
- Non-scalar return: fallback.
- Extra helper call or unknown side effect: fallback.
- Public/generic aggregate-return sample: fallback.
- Caller with one accepted and one unaccepted `RetSlots > 1` call: fallback.

Fresh Tier 1 and validator expectations:

- A packet that changes code must regenerate a fresh Tier 1 report before
  claiming row-level native behavior.
- The validator must pass against the fresh report.
- Backend sidecars must be the source of truth for native/fallback claims.
- JSON/HTTP must not be called native until their sidecars show register paths
  and no `unsupported_aggregate_return` / `unsupported_call_abi` blockers for
  the promoted functions.

## 10. Implementation Sequence

Use narrow packets. Do not land a broad generic ABI patch.

1. P80 should add target-neutral RED tests for the reusable internal writer
   contract in `compiler/internal/machine`. Safest first packet:
   `P80-inout-writer-helper-summary-recognizer-red-tests`.
2. Implement the target-neutral helper plan for `helper-summary` only after the
   RED tests exist. This packet should not route x64 emission or buildreport
   promotion.
3. Add caller-plan tests for row-local JSON/HTTP callers, including mixed
   safe/unsafe rejection.
4. Implement the row-local caller plan. Keep generic `IRCall.RetSlots > 1`
   fallback outside accepted plans.
5. Add x64core emission tests for the accepted plan.
6. Implement x64core direct emission for accepted `helper-summary` helpers and
   callers. Do not modify public SysV/Win64 aggregate-return limits.
7. Update `buildreports` to classify accepted internal writer promotions as
   verified while preserving generic aggregate/call fallback classifications.
8. Regenerate a fresh Tier 1 report and run the validator. Only then may a
   packet claim JSON/HTTP native promotion.

The key implementation invariant is simple: `RetSlots=3` is safe only when the
exact internal writer contract proves it is scalar return plus hidden `inout`
writeback state. Everywhere else, `RetSlots=3` remains an unsupported aggregate
or unsupported call ABI boundary.
