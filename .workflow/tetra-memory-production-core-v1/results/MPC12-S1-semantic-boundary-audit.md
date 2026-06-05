# MPC12-S1 Semantic Boundary Audit Result

Status: integrated.

Raman completed the read-only semantic audit. Key accepted findings:

- `checkTypedActorBuiltin` validates `core.send_typed` payload expressions with
  `validateActorBoundaryPayloadExpr`, `checkBorrowedEscape`, and
  `consumeTypedActorTransferPayloads`.
- `validateActorBoundaryPayloadExpr` permits slice/String payloads only for
  explicit copy expressions or the narrow owned-region slice plus owner move.
- `checkTypedTaskBuiltin` validates worker signatures, effects, and typed error
  sendability. The current typed task spawn API has no payload expression, so
  task String/slice transfer remains conservative instead of a validated
  cross-task copy path.
- Existing actor tests cover borrowed slice/String rejection, copied actor send
  success, owned-region move, and sender consume diagnostics. MPC-12 added
  focused task boundary coverage for String/slice typed-error rejection and
  copied local work before typed task spawn.

Commands were read-only `sed`/`rg` inspections over `compiler/internal/semantics`,
`compiler/tests`, `compiler/actors_test.go`, examples, and
`docs/design/actor_region_transfer.md`.
