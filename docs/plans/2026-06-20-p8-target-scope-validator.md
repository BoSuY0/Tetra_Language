# P8 Target-Scope Validator Slice

Date: 2026-06-20

## Scope

Make the P7/P8 compiler RSS bundle validator enforce target-scope non-claim invariants, so
`target-scope.json` cannot silently drift into a cross-target claim. The earlier target-scope slice
wrote and hashed the artifact; this slice validates the artifact as part of bundle acceptance.

## Acceptance

- `validator-output.txt` covers both `scenario-summary.json` and `target-scope.json` schemas.
- The validator rejects a missing required non-claim target.
- The validator rejects any `host_rss_measured` claim for non-`linux-x64` targets.
- The validator rejects `host_rss_measured` when `host_target` is not `linux/amd64`.
- The validator rejects empty non-claim reasons.

## Verification

- RED:
  `GOTELEMETRY=off GOCACHE="$(pwd)/.cache/go-build-ram-p8-target-scope-validator-red" go test -count=1 ./tools/internal/ramcompilerrss -run TestValidateBundleOutputRejectsTargetScopeClaimLeakage`
  failed because `validateBundleOutput` did not exist.
- GREEN:
  `GOTELEMETRY=off GOCACHE="$(pwd)/.cache/go-build-ram-p8-target-scope-validator-green" go test -count=1 ./tools/internal/ramcompilerrss -run 'TestValidateBundleOutputRejectsTargetScopeClaimLeakage|TestRunWritesCompilerRSSBundle' -v`
  passed after wiring target-scope validation into the bundle validator.
