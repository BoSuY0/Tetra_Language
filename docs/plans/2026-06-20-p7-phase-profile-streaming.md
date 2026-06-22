# P7 Phase Profile Streaming Slice

Date: 2026-06-20

## Scope

Extend the P7.3 streaming/atomic report-output work to the compiler phase-profile artifact itself.
`compilerPhaseProfiler.write` previously marshaled the whole profile with `json.MarshalIndent` and
then wrote the resulting byte slice with `os.WriteFile`.

## Acceptance

- The phase-profile writer reuses the same file-backed JSON encoder and temporary-file/rename path
  as compiler JSON reports.
- The writer no longer uses `json.MarshalIndent` or `os.WriteFile`.
- Existing report writer streaming and failure-cleanup tests remain green.

## Verification

- RED:
  `GOTELEMETRY=off GOCACHE="$(pwd)/.cache/go-build-ram-p7-profile-stream-red" go test -count=1 ./compiler -run TestP7CompilerPhaseProfileStreamsJSONToTemporaryFileAndRenames`
  failed because `compilerPhaseProfiler.write` still used `json.MarshalIndent` and `os.WriteFile`.
- GREEN:
  `GOTELEMETRY=off GOCACHE="$(pwd)/.cache/go-build-ram-p7-profile-stream-green" go test -count=1 ./compiler -run 'TestP7(CompilerPhaseProfileStreamsJSONToTemporaryFileAndRenames|WriteReportStreamsJSONToTemporaryFileAndRenames|WriteReportRemovesTemporaryFileAfterFailure)'`
  passed after delegating profile emission to `writeReport`.
