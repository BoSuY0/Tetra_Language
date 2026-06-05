# P3 Docs, Audit, And Manifest

Objective: document v2 rows and synchronize schema/manifest validation.

Ownership:
- `docs/audits/memory-ideal-vslice-v2-correlation.md`
- `docs/audits/memory-ideal-vslice-v2-final.md`
- `docs/spec/memory_report_schema_v1.md`
- `docs/design/memory_production_core_v1.md` if needed
- `docs/generated/manifest.json`
- `tools/cmd/validate-memory-correlation`
- `tools/cmd/validate-manifest` if fixtures require sync

Do:
- Add exact three-row v2 correlation matrix.
- Add final audit with all rows `validated_narrow` or `conservative`.
- Update report schema and manifest.
- Keep nonclaims explicit.

Do not:
- Claim full callable memory model.
- Claim target parity or performance.

Expected output: `.workflow/memory-ideal-vertical-slice-v2/results/P3-docs-audit-manifest.md`.

Verification:
- `go test ./tools/cmd/validate-memory-correlation -count=1`
- `go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory-ideal-vslice-v2-correlation.md`
- `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`
- `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`
