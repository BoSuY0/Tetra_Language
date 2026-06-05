# P4 Docs Manifest Gates Result

Status: integrated.

Read-only audit summary from Carson:

- Docs are validated through
  `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- `docs/generated/manifest.json` is generated from `compiler.GetManifest()`,
  which is sourced from `FeatureRegistry`.
- New release-visible docs should be attached to the registry and regenerated,
  not hand-edited in the JSON.

Integrated artifacts:

- `compiler/features.go`
- `docs/generated/manifest.json`
- `docs/spec/unsafe.md`
- `docs/spec/memory_report_schema_v1.md`

Verification:

- `GOCACHE=$(pwd)/.cache/go-build-verify-docs-memory go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` passed.
- `GOCACHE=$(pwd)/.cache/go-build-validate-manifest-memory go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json` passed.
