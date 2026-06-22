# P7 PLIR-Only Report Fast Path Slice

Date: 2026-06-20

## Scope

Make the `EmitPLIR`-only report mode avoid compiler report intermediates that are not needed for
PLIR output. Before this slice, `emitExplainReports` built the allocation plan, lowered full IR,
validated allocation lowering, and built bounds reports before writing `.plir.json` and
`.plir.txt`, even when no other report flag was requested.

## Acceptance

- `EmitPLIR` by itself writes `.plir.json` and `.plir.txt` after PLIR verification.
- The `EmitPLIR`-only path returns before `allocplan.FromPLIRWithOptions` and
  `lower.LowerWithOptions`.
- `Explain`, proof, bounds, allocation, memory, and RAM-contract modes keep the existing
  validation/report intermediates.

## Verification

- RED:
  `GOTELEMETRY=off GOCACHE="$(pwd)/.cache/go-build-ram-p7-plir-only-red" go test -count=1 ./compiler -run TestP7EmitPLIROnlyReturnsBeforeAllocPlanAndIRReportIntermediates`
  failed because `emitExplainReports` had no `plirOnly` fast path before heavy report
  intermediates.
- GREEN:
  `GOTELEMETRY=off GOCACHE="$(pwd)/.cache/go-build-ram-p7-plir-only-green" go test -count=1 ./compiler -run 'TestP7EmitPLIROnlyReturnsBeforeAllocPlanAndIRReportIntermediates|TestReportFlagsDoNotChangeBorrowedReturnFailure|TestBuildExplainReportsTruthProofAndAllocationArtifacts|TestP7WriteReportStreamsJSONToTemporaryFileAndRenames|TestP7CompilerPhaseProfileStreamsJSONToTemporaryFileAndRenames'`
  passed after adding the PLIR-only early return.
