# P4 Verification Result

Status: accepted

## Focused Gates

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v3-memoryfacts go test ./compiler/internal/memoryfacts -count=1`
  passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v3-mini go test ./compiler/internal/memorymodel -count=1`
  passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v3-semantics go test ./compiler/tests/semantics -run 'MemoryIdealV3|Interface|Protocol|Existential|DynamicDispatch|Borrow|NoAlias|Alias' -count=1`
  passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v3-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation -count=1`
  passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v3-correlation go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory-ideal-vslice-v3-correlation.md`
  passed.

## Broad Gates

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v3-broad go test ./compiler/... ./cli/... ./tools/... -count=1`
  passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v3-ci bash scripts/ci/test.sh`
  passed and printed `OK` with artifact
  `tetra.release.v0_4_0.go-test-suite.v1`.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v3-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`
  passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v3-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`
  passed.
- `git diff --check` passed.
- `graphify update .` passed and rebuilt `graphify-out` with 21254 nodes,
  66433 edges, and 1167 communities.

## Classification

All required v3 rows are represented, documented, and verified:

- `MEM-BORROW-006`: `validated_narrow`.
- `MEM-BORROW-007`: `conservative`.
- `MEM-ALIAS-003`: `conservative`.

No unrelated dirty-worktree failure required classification; all required gates
passed.
