# P7 Failed Build Final Cleanup Slice

Date: 2026-06-20

## Scope

Make the profiled native build failure path match the successful build cleanup standard. Before
this slice, `BuildFileWithStatsOpt` wrote a `final_cleanup` phase profile from the deferred error
path, but did not invoke `compilerProcessMemoryRelease` first and could carry stale retained-state
counts from earlier phases.

## Acceptance

- Failed profiled native builds call `compilerProcessMemoryRelease` before the deferred
  `final_cleanup` snapshot.
- The deferred failure cleanup clears retained `World`, module plan, linked-object, and object
  references before releasing memory.
- The failed `final_cleanup` phase records zero source, checked, object, transient IR, and
  allocation-plan retained counts.
- The profile preserves a failure note so compile-error samples remain distinguishable from
  successful builds.

## Verification

- RED:
  `GOTELEMETRY=off GOCACHE="$(pwd)/.cache/go-build-ram-p7-failed-cleanup-red" go test -count=1 ./compiler -run TestP7FailedProfiledBuildReleasesMemoryBeforeFinalCleanupSnapshot`
  failed because failed profiled builds made zero memory-release calls before `final_cleanup`.
- GREEN:
  `GOTELEMETRY=off GOCACHE="$(pwd)/.cache/go-build-ram-p7-failed-cleanup-green2" go test -count=1 ./compiler -run 'TestP7(FailedProfiledBuildReleasesMemoryBeforeFinalCleanupSnapshot|ReportBuildReleasesTransientMemoryBeforeProfileSnapshot)'`
  passed.
- Evidence bundle:
  `reports/stabilization/tetra-ram-p7-compiler-rss-b452638a8af7-failed-cleanup-samples1/`
  contains a compile-error sample whose `compiler-profile.json` has `source_loading_parsing` and
  `final_cleanup` phases, a failure note, and zero retained counts in `final_cleanup`.
