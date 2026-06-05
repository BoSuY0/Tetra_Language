# P4 Docs Registry Result

Updated `compiler/features.go`, `docs/spec/surface_v1.md`,
`docs/user/surface_guide.md`, `docs/spec/current_supported_surface.md`, and
`scripts/release/surface/README.md` with honest component-tree wording.
Regenerated and validated `docs/generated/manifest.json`.

Evidence:
- `go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json`
- `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`
- `go run ./tools/cmd/verify-docs`
