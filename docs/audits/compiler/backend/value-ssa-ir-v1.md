# Value SSA IR v1

Status: P16.1 evidence audit for the Ideal Master Plan.

## Summary

The compiler now has a typed SSA/value IR layer for the current scalar and slice backend subset. SSA
values carry explicit types, blocks carry params for phi-style joins, branch terminators pass typed
arguments, and calls plus memory operations carry effect tokens.

## Evidence

| Check                                                                  | Result |
| ---------------------------------------------------------------------- | ------ |
| SSA verifier rejects malformed values and missing call effect tokens   | pass   |
| Stack IR scalar lowering to typed SSA                                  | pass   |
| Stack IR scalar loop lowering with block params                        | pass   |
| Stack IR proof-tagged slice-sum lowering with memory effect token      | pass   |
| PLIR call and index-load rows lower to SSA with effect chains          | pass   |
| Machine backend reports require `ssa_verified: true` before Machine IR | pass   |

## Boundaries

This is an internal compiler evidence layer. It does not claim a full SSA optimizer, full
aggregate/string lowering, public backend selection, or removal of stack IR fallback. Unsupported
functions continue to use the stack fallback path until later register-backend coverage slices
promote them with evidence.
