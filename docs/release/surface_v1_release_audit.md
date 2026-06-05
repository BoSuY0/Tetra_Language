# Tetra Surface v1 Release Audit

Status: in progress for the bounded `surface-v1-linux-web` release scope.

This audit tracks the honest current claim:

```text
Tetra Surface v1 is current/release-ready for linux-x64 real-window and
wasm32-web browser-canvas Surface scope, with headless as release evidence
target.
```

It does not claim macOS Surface, Windows Surface, wasm32-wasi Surface UI, GPU
rendering, platform-native widgets, dynamic trait-object widgets, witness-table
component dispatch, full rich text editing, full AT-SPI/screen-reader support,
or DOM/React/user-JS application UI.

## Evidence Snapshot

- Release scope: `surface-v1-linux-web`.
- Supported runtime targets: `linux-x64`, `wasm32-web`.
- Release evidence target: `headless`.
- Unsupported targets: `macos-x64`, `windows-x64`, `wasm32-wasi`.
- Section 14 release gate report:
  `/tmp/tetra-surface-section14-release-gate/surface-release-summary.json`.
- Section 14 artifact hash manifest:
  `/tmp/tetra-surface-section14-release-gate/artifact-hashes.json`.
- Section 13 safe-view lifetime report:
  `/tmp/tetra-safe-view-lifetime-surface-release-current`.
- Section 16 API stability report:
  `/tmp/tetra-surface-section16-api-stability/surface-api-stability-summary.json`.

## Checklist

| Item | Status | Evidence |
| --- | --- | --- |
| release contract created | done | `docs/release/surface_v1_release_contract.md` |
| feature registry updated | done | `compiler/features.go`; manifest lists 8 Surface current IDs and 3 unsupported target IDs |
| docs updated | done | `docs/spec/surface_v1.md`, `docs/spec/current_supported_surface.md`, `docs/user/surface_guide.md`, `docs/user/examples_index.md`, `docs/release/surface_v1_release_notes.md` |
| manifest updated | done | `docs/generated/manifest.json`; `validate-manifest` passed |
| API docs generated | done | `scripts/release/surface/api-stability-gate.sh --report-dir /tmp/tetra-surface-section16-api-stability` generated and validated `artifacts/tetra-docs.md` with 7 stable Surface modules |
| release examples added | done | `examples/surface_release_form.tetra`, `examples/surface_release_text_input.tetra`, `examples/surface_release_accessibility.tetra` |
| text/input gate passed | done | Surface v1 release gate generated text-input reports for headless, linux-x64, and wasm32-web |
| clipboard gate passed | done | Surface v1 release gate summary reports `clipboard-text-v1` |
| IME/composition gate passed | done | Surface v1 release gate summary reports `composition-baseline-v1` |
| toolkit gate passed | done | Surface v1 release gate generated toolkit reports for headless, linux-x64, and wasm32-web |
| Linux accessibility bridge gate passed | done | `surface-linux-x64-release-accessibility.json` in the Section 14 release gate report dir |
| browser accessibility snapshot gate passed | done | `surface-wasm32-web-release-accessibility.json` in the Section 14 release gate report dir |
| release gate passed | done | `bash scripts/release/surface/release-gate.sh --report-dir /tmp/tetra-surface-section14-release-gate` |
| experimental regression gate passed | pending | Section 18 will rerun and preserve the experimental regression gate |
| safe-view-lifetime gate passed | done | Section 13 safe-view lifetime gate passed with schema `tetra.safe-view-lifetime.gate.v1` |
| full tests passed | pending | Final command matrix will rerun broad Go/CI tests |
| artifact hashes validated | done | Section 14 release gate wrote and validated 57 artifact hash entries |
| reports preserved | pending | Section 19 will preserve final release artifacts and provenance |
| unsupported targets documented | done | release contract, Surface spec, supported-surface spec, guide, release notes, and feature registry |
| no fake production claims accepted | done | release-state validator rejects missing Surface feature IDs; release validators reject unsupported/fake Surface claims |

## Commands Verified In Section 14

```sh
GOCACHE=$(pwd)/.cache/go-build-surface-release \
  go test ./tools/cmd/validate-manifest ./tools/cmd/verify-docs \
  ./tools/cmd/validate-surface-release-state ./compiler/tests/semantics \
  -run 'ValidateFeatures|VerifyFeatureRegistry|SurfaceReleaseState|FeatureRegistryDeclaresSurface|FeatureRegistryCoversReleaseStatuses|ManifestBuiltins' \
  -count=1

GOCACHE=$(pwd)/.cache/go-build-surface-release \
  go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json

GOCACHE=$(pwd)/.cache/go-build-surface-release \
  go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json

GOCACHE=$(pwd)/.cache/go-build-surface-release \
  bash scripts/release/surface/release-gate.sh \
  --report-dir /tmp/tetra-surface-section14-release-gate

GOCACHE=$(pwd)/.cache/go-build-surface-release \
  go run ./tools/cmd/validate-surface-release-state \
  --report-dir /tmp/tetra-surface-section14-release-gate \
  --expected-status current \
  --scope surface-v1-linux-web \
  --manifest docs/generated/manifest.json

git diff --check
graphify update .
```

## Next Audit Updates

- Section 18 must fill the experimental regression gate row.
- Section 19 must fill final report preservation and provenance rows.
- The final completion matrix must fill full tests passed.
