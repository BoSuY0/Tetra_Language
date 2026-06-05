# P3 Docs Audit Manifest Result

Status: accepted

## GREEN Evidence

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v2-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation ./tools/cmd/validate-manifest -count=1`
  passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v2-correlation go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory-ideal-vslice-v2-correlation.md`
  passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v2-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`
  passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v2-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`
  passed.

## Accepted Changes

- `docs/audits/memory-ideal-vslice-v2-correlation.md` adds exactly three rows:
  `MEM-BORROW-004`, `MEM-BORROW-005`, and `MEM-ALIAS-002`.
- `docs/audits/memory-ideal-vslice-v2-final.md` classifies the borrow rows as
  `validated_narrow` and the callback/reentrant inout row as `conservative`.
- `docs/spec/memory_report_schema_v1.md` documents v2 projections,
  parent-fact requirements, and conservative callback inout projection.
- `docs/design/memory_production_core_v1.md` documents the narrow production
  projection boundary.
- `docs/generated/manifest.json` and `tools/cmd/validate-manifest` now require
  the v2 audit docs for `safety.production-core`.

## Nonclaims Preserved

- No full callable ABI.
- No captured or escaping closure support.
- No trusted unknown callback target facts.
- No broad noalias.
