# P7 Semantic State Release Slice

Date: 2026-06-20

## Scope

Add compiler phase-profile visibility for retained source and semantic graph state, then release
those heavy references at the first safe native executable boundaries:

- release the loaded `World` graph after module codegen no longer needs cache root/source data;
- retain `CheckedProgram` through UI/report generation because reports still need semantic data;
- release `CheckedProgram` before the `final_cleanup` profile snapshot.

This is a P7 heavyweight compiler-lifetime slice only. It does not close the full P7 RSS gate or
any P8 target parity claim.

## Acceptance

- `tetra.compiler.phase-profile.v1` includes `source_file_count`,
  `checked_function_count`, and `checked_type_count` in the top-level latest-state object and each
  phase snapshot.
- `semantic_analysis` records positive source and checked graph counts.
- `report_generation` records positive checked graph counts when report generation needs semantic
  state.
- `final_cleanup` records zero source and checked graph counts after explicit release and Go memory
  release.
- Existing object and transient IR release regressions stay green.

## Verification

- RED:
  `GOTELEMETRY=off GOCACHE="$(pwd)/.cache/go-build-ram-p7-semantic-red" go test -count=1 ./compiler -run TestP7CompilerPhaseProfileReleasesSemanticStateAtFinalCleanup`
  failed because the retained source/checked graph fields were absent and decoded as zero.
- GREEN:
  `GOTELEMETRY=off GOCACHE="$(pwd)/.cache/go-build-ram-p7-semantic-green" go test -count=1 ./compiler -run TestP7CompilerPhaseProfileReleasesSemanticStateAtFinalCleanup`
  passed after adding the fields and release boundary.
- Focused P7 profile regressions:
  `GOTELEMETRY=off GOCACHE="$(pwd)/.cache/go-build-ram-p7-semantic-phase" go test -count=1 ./compiler -run 'Test(BuildCompilerPhaseProfileRecordsP7PhaseRSS|P7CompilerPhaseProfile(DropsObjectReferencesBeforeReports|ReleasesTransientIRBeforeModuleCodegen|ReleasesSemanticStateAtFinalCleanup)|P7CompilerPhaseProfileMemoryBudgetReducesWorkerCount|CompilerReportWriter)'`
  passed.
