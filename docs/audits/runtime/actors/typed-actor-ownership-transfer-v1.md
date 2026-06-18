# Typed Actor Ownership Transfer v1

Status: P18.1 bounded evidence slice for the Ideal Master Plan.

## Summary

`compiler/internal/actorsafety.TypedActorOwnershipTransferCoverage()` emits schema
`tetra.actors.ownership_transfer.v1`. The coverage report ties existing typed actor sendability
checks, new PLIR moved facts, typed mailbox model evidence, actor-transfer explain reports, and
stress diagnostics into one machine-checkable closure.

This slice does not implement a new actor runtime, scheduler, transport, distributed ownership
protocol, or safe raw-pointer typed actor payload.

## Rows

| Row                            | Status               | Evidence                                                                                                                                                                                                                                                                                   | Boundary                                                                                                                                                  |
| ------------------------------ | -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| borrowed view copy boundary    | `implemented_narrow` | `TestBorrowedSliceAcrossActorBoundaryRejectsUnlessCopied`, `TestBorrowedActorSendRejectedUnlessCopied`, `validateActorBoundaryPayloadExpr`                                                                                                                                                 | Borrowed views are rejected at typed actor boundaries unless the expression explicitly uses `.copy()` or an owned-region slice moves with its owner.      |
| owned region move              | `implemented_narrow` | `TestActorsTypedMessagesAllowIslandTransferCheckAndLower`, `TestActorsTypedMessagesOwnedRegionSliceMoveBuildAndRun`, `TestOwnedRegionMessageMovesZeroCopyAndBorrowedPayloadRequiresCopy`                                                                                                   | Local typed actor payloads can move an island owner and associated region-backed slice without copying bytes. Distributed zero-copy is not claimed.       |
| sender loses access after move | `implemented_narrow` | `TestActorsTypedMessagesIslandTransferConsumesSource`, `TestActorsTypedMessagesOwnedRegionSliceMoveConsumesSenderSlice`, `TestOwnedRegionMustMoveAndSenderUseAfterMoveRejects`                                                                                                             | Sender use-after-move remains a checker diagnostic, not a full race-safety proof.                                                                         |
| receiver owns moved region     | `implemented_narrow` | `TestOwnedRegionMessageMovesZeroCopyAndBorrowedPayloadRequiresCopy`, `TestActorsTypedMessagesOwnedRegionSliceMoveBuildAndRun`                                                                                                                                                              | Receiver ownership evidence is local scheduler-model plus current linux-x64 typed mailbox execution.                                                      |
| explicit copy fallback         | `implemented_narrow` | `TestBorrowedActorSendRejectedUnlessCopied`, `compiler/internal/buildreports/actor_transfer.go::actorTransferRowForPayload`, `TestOwnedRegionMessageMovesZeroCopyAndBorrowedPayloadRequiresCopy`                                                                                           | Copy fallback is explicit and report-visible; no hidden copy-elision promotion is claimed.                                                                |
| unsafe send contract           | `implemented_narrow` | `TestUnsafePointerRequiresExplicitUnsafeSendContract`, `validateTypedActorMessageType`, `TestPlan250SafetyRuntimeMatrix`                                                                                                                                                                   | Safe typed actor messages still reject pointer payloads. The unsafe-send contract is internal checker-model evidence only.                                |
| semantics transfer checker     | `implemented_narrow` | `checkTypedActorCallWithEffects`, `validateActorBoundaryPayloadExpr`, `consumeTypedActorTransferPayloads`                                                                                                                                                                                  | Source-level semantics consume moved payloads and reject borrowed escapes conservatively.                                                                 |
| PLIR moved facts               | `implemented_narrow` | `recordActorSendCall`, `TestFromCheckedProgramRecordsTypedActorMovedFacts`, `TestVerifyProgramRejectsFakeActorMovedFactClaims`                                                                                                                                                             | Direct checked `core.send_typed` enum constructor payloads now emit `OpActorSend` and `FactMoved` rows for local moved island/region-backed slice values. |
| runtime mailbox representation | `implemented_narrow` | `TypedMailbox`, `TestTypedMailboxPreservesCapacityBackpressureAndOwnershipMetadata`, `TestOwnedRegionMessageMovesZeroCopyAndBorrowedPayloadRequiresCopy`                                                                                                                                   | Runtime mailbox evidence remains bounded local model/report evidence.                                                                                     |
| actor transfer report          | `implemented_narrow` | `compiler/internal/buildreports/actor_transfer.go::BuildActorTransferReport`, `compiler/internal/buildreports/actor_transfer.go::actorTransferRowForPayload`, `TestActorsTypedMessagesOwnedRegionSliceMoveExplainReport`, `TestActorsTypedMailboxExplainReportIncludesMetadataAndCopyMove` | Reports are evidence only and do not change safe semantics or runtime behavior.                                                                           |
| stress diagnostics             | `implemented_narrow` | `actor_task_ownership_test.go`, `actor_task_stress_test.go`, typed actor consumed-source tests                                                                                                                                                                                             | Stress coverage is bounded compiler and runtime smoke evidence, not a full cross-thread race proof.                                                       |

## Validator Guards

`ValidateTypedActorOwnershipTransferCoverage()` rejects:

- distributed zero-copy claims;
- runtime-behavior-change claims;
- missing P18.1 rows, required facts, evidence, or boundaries;
- PLIR moved-fact rows that omit `FactMoved`, `OpActorSend`, or `core.send_typed`;
- missing use-after-move stress diagnostics;
- missing distributed zero-copy and runtime-behavior non-claims.

`compiler/internal/plir.VerifyFunction()` now rejects fake moved facts that lack value/source/reason
evidence or try to mark borrowed values as moved.

## Non-Claims

- Distributed pointer or region zero-copy is not claimed.
- P18.1 does not change actor runtime behavior.
- Safe typed actor raw pointer payloads remain rejected.
- Full production actor runtime remains governed by the P18.0 boundary audit.
