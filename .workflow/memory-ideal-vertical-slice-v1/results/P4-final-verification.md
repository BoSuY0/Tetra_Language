# P4-final-verification Result

Status: accepted

## Verification Evidence

- Focused gates passed for memoryfacts, MiniMemoryModel, semantics, tools, and
  v1 correlation.
- Broad gate passed on rerun:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-broad go test ./compiler/... ./cli/... ./tools/... -count=1`.
- CI script passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-ci bash scripts/ci/test.sh`, ending with `OK` and
  `Artifact: tetra.release.v0_4_0.go-test-suite.v1`.
- Manifest and docs validators passed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json` and
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- `git diff --check` passed after Graphify update.
- `graphify update .` passed and rebuilt 21217 nodes, 66349 edges, and 1181
  communities.

## Notes

- The first broad gate run failed on stale `validate-manifest` in-test fixtures.
  That was fixed, verified directly with `go test ./tools/cmd/validate-manifest
  -count=1`, and the broad gate passed on rerun.
- The worktree remains heavily dirty with unrelated changes preserved.
