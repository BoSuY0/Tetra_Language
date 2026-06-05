# P3-docs-audit

Packet ID: P3-docs-audit

Objective: Document the proven v1 slice and wire docs/manifest validators.

Context: v1 requires exactly two new correlation rows and final audit
classification for both.

Files / sources:

- `docs/audits/memory-ideal-vslice-v1-correlation.md`
- `docs/audits/memory-ideal-vslice-v1-final.md`
- `docs/spec/memory_report_schema_v1.md`
- `docs/generated/manifest.json`
- `tools/cmd/validate-manifest`

Ownership: docs and validators needed to keep docs honest.

Do: update docs only after behavior is proven.

Do not: add unrelated release claims or target parity claims.

Expected output: docs, manifest updates, and packet result note.

Verification:

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-correlation go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory-ideal-vslice-v1-correlation.md`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`
