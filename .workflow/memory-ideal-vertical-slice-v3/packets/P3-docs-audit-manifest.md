# P3 Docs Audit Manifest Packet

## Scope

Add v3 audit artifacts and synchronize schema, manifest, and validators.

## Required Artifacts

- `docs/audits/memory-ideal-vslice-v3-correlation.md`
- `docs/audits/memory-ideal-vslice-v3-final.md`
- `docs/spec/memory_report_schema_v1.md`
- `docs/design/memory_production_core_v1.md` if production-core behavior changes
- `docs/generated/manifest.json`
- `tools/cmd/validate-manifest` fixtures if required

## Acceptance

Accepted only after tool tests, v3 correlation validation, `validate-manifest`,
and `verify-docs` pass with recorded command evidence.
