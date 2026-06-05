# Final report

Goal: Tetra Surface Release Promotion v1
Status: complete
Scope: surface-v1-linux-web

## Supported

- headless evidence target
- linux-x64 real-window
- wasm32-web browser-canvas

## Unsupported

- macOS Surface
- Windows Surface
- wasm32-wasi Surface UI

## Implemented

- release contract
- release schemas
- production text/input
- clipboard
- IME/composition baseline
- production toolkit v1
- style/layout baseline
- accessibility platform bridge
- browser release Surface
- Linux release Surface
- validators
- release gate
- docs
- manifest
- Graphify

## Verification

Command list:

```sh
GOCACHE=$(pwd)/.cache/go-build-surface-release go test ./tools/scriptstest -run 'ReleaseSurfaceExamplesRejectFakePromotionSources|ReleaseSurfaceExamplesExistAndUseStableCoreModules' -count=1
GOCACHE=$(pwd)/.cache/go-build-surface-release go test ./tools/validators/surface -run 'SurfaceReleaseRejects|SurfaceReleaseNegativeFixturesRejectFakeClaims' -count=1
GOCACHE=$(pwd)/.cache/go-build-surface-release go test ./tools/cmd/verify-docs -run 'VerifySurfaceReleaseDocs' -count=1
GOCACHE=$(pwd)/.cache/go-build-surface-release go test ./compiler/tests/semantics -run 'SurfaceRelease.*Example|SurfaceReleaseTextInput|SurfaceReleaseCounter' -count=1
GOCACHE=$(pwd)/.cache/go-build-surface-release go test ./tools/scriptstest ./tools/validators/surface ./tools/cmd/verify-docs ./compiler/tests/semantics -run 'ReleaseSurfaceExamples|SurfaceReleaseRejects|SurfaceReleaseNegativeFixturesRejectFakeClaims|VerifySurfaceReleaseDocs|VerifyFeatureRegistry|SurfaceReleaseTextInput|SurfaceReleaseCounter' -count=1
GOCACHE=$(pwd)/.cache/go-build-surface-release go test ./tools/cmd/verify-docs -count=1
GOCACHE=$(pwd)/.cache/go-build-surface-release go test ./tools/validators/surface -count=1
GOCACHE=$(pwd)/.cache/go-build-surface-release go test ./tools/cmd/validate-surface-runtime ./tools/cmd/validate-surface-release-state -count=1
GOCACHE=$(pwd)/.cache/go-build-surface-release go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-surface-release go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-surface-release bash scripts/release/surface/api-stability-gate.sh --report-dir /tmp/tetra-surface-section19-api-stability
GOCACHE=$(pwd)/.cache/go-build-surface-release bash scripts/release/surface/gate.sh --report-dir /tmp/tetra-surface-section19-experimental-gate
GOCACHE=$(pwd)/.cache/go-build-surface-release bash scripts/release/surface/release-gate.sh --report-dir /tmp/tetra-surface-section19-release-gate
GOCACHE=$(pwd)/.cache/go-build-surface-release go run ./tools/cmd/validate-surface-release-state --report-dir /tmp/tetra-surface-section19-release-gate --expected-status current --scope surface-v1-linux-web --manifest docs/generated/manifest.json
git diff --check
graphify update .
```

Report paths:

- `/tmp/tetra-surface-release-v1-current/surface-release-summary.json`
- `/tmp/tetra-surface-release-v1-current/surface-headless-release.json`
- `/tmp/tetra-surface-release-v1-current/surface-headless-release-text-input.json`
- `/tmp/tetra-surface-release-v1-current/surface-headless-release-toolkit.json`
- `/tmp/tetra-surface-release-v1-current/surface-headless-release-accessibility.json`
- `/tmp/tetra-surface-release-v1-current/surface-linux-x64-release-window.json`
- `/tmp/tetra-surface-release-v1-current/surface-linux-x64-release-text-input.json`
- `/tmp/tetra-surface-release-v1-current/surface-linux-x64-release-toolkit.json`
- `/tmp/tetra-surface-release-v1-current/surface-linux-x64-release-accessibility.json`
- `/tmp/tetra-surface-release-v1-current/surface-wasm32-web-release-browser.json`
- `/tmp/tetra-surface-release-v1-current/surface-wasm32-web-release-text-input.json`
- `/tmp/tetra-surface-release-v1-current/surface-wasm32-web-release-toolkit.json`
- `/tmp/tetra-surface-release-v1-current/surface-wasm32-web-release-accessibility.json`
- `/tmp/tetra-surface-experimental-regression-current`
- `/tmp/tetra-safe-view-lifetime-surface-release-current`
- `/tmp/tetra-surface-section19-api-stability/surface-api-stability-summary.json`
- `/tmp/tetra-surface-section19-experimental-gate`
- `/tmp/tetra-surface-section19-release-gate/surface-release-summary.json`
- `/tmp/tetra-surface-section19-release-gate/surface-headless-release.json`
- `/tmp/tetra-surface-section19-release-gate/surface-headless-release-text-input.json`
- `/tmp/tetra-surface-section19-release-gate/surface-headless-release-toolkit.json`
- `/tmp/tetra-surface-section19-release-gate/surface-headless-release-accessibility.json`
- `/tmp/tetra-surface-section19-release-gate/surface-linux-x64-release-window.json`
- `/tmp/tetra-surface-section19-release-gate/surface-linux-x64-release-text-input.json`
- `/tmp/tetra-surface-section19-release-gate/surface-linux-x64-release-toolkit.json`
- `/tmp/tetra-surface-section19-release-gate/surface-linux-x64-release-accessibility.json`
- `/tmp/tetra-surface-section19-release-gate/surface-wasm32-web-release-browser.json`
- `/tmp/tetra-surface-section19-release-gate/surface-wasm32-web-release-text-input.json`
- `/tmp/tetra-surface-section19-release-gate/surface-wasm32-web-release-toolkit.json`
- `/tmp/tetra-surface-section19-release-gate/surface-wasm32-web-release-accessibility.json`

Artifact hash path:

- `/tmp/tetra-surface-release-v1-current/artifact-hashes.json`
- `/tmp/tetra-surface-section19-release-gate/artifact-hashes.json`

Final dump paths:

- `.workflow/tetra-surface-release-promotion-v1/final-report.md`
- `.workflow/tetra-surface-release-promotion-v1/baseline-report.md`
- `.workflow/tetra-surface-release-promotion-v1/baseline-reports/`
- `docs/generated/manifest.json`
- `docs/release/surface_v1_release_audit.md`
- `docs/release/surface_v1_release_contract.md`
- `docs/release/surface_v1_release_notes.md`
- `docs/spec/surface_v1.md`
- `graphify-out/GRAPH_REPORT.md`
- `graphify-out/graph.json`
- `dumps/tetra_language_dump_20260603_201342Z_part_001.md`
- `dumps/tetra_language_dump_20260603_201342Z_part_002.md`

## Final Verification

Section 21-22 commands passed on 2026-06-03:

```sh
GOCACHE=$(pwd)/.cache/go-build-surface-release bash scripts/release/surface/release-gate.sh --report-dir /tmp/tetra-surface-release-v1-current
GOCACHE=$(pwd)/.cache/go-build-surface-release bash scripts/release/surface/gate.sh --report-dir /tmp/tetra-surface-experimental-regression-current
GOCACHE=$(pwd)/.cache/go-build-surface-release bash scripts/release/safe-view-lifetime/gate.sh --report-dir /tmp/tetra-safe-view-lifetime-surface-release-current
GOCACHE=$(pwd)/.cache/go-build-surface-release go test ./compiler/... ./cli/... ./tools/... -count=1
GOCACHE=$(pwd)/.cache/go-build-surface-release go test ./... ./compiler/... ./cli/... ./tools/... -count=1
GOCACHE=$(pwd)/.cache/go-build-surface-release bash scripts/ci/test.sh
GOCACHE=$(pwd)/.cache/go-build-surface-release go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-surface-release go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-surface-release go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
./tetra doc examples > /tmp/tetra-surface-release-v1-current/artifacts/tetra-docs.md
GOCACHE=$(pwd)/.cache/go-build-surface-release go run ./tools/cmd/validate-api-docs --docs /tmp/tetra-surface-release-v1-current/artifacts/tetra-docs.md
GOCACHE=$(pwd)/.cache/go-build-surface-release go run ./tools/cmd/validate-artifact-hashes -write -root /tmp/tetra-surface-release-v1-current -out /tmp/tetra-surface-release-v1-current/artifact-hashes.json
GOCACHE=$(pwd)/.cache/go-build-surface-release go run ./tools/cmd/validate-artifact-hashes -manifest /tmp/tetra-surface-release-v1-current/artifact-hashes.json
GOCACHE=$(pwd)/.cache/go-build-surface-release go run ./tools/cmd/validate-surface-release-state --report-dir /tmp/tetra-surface-release-v1-current --expected-status current --scope surface-v1-linux-web --manifest docs/generated/manifest.json
git diff --check
graphify update .
GOCACHE=$(pwd)/.cache/go-build-surface-release go run ./create_dumps.go
```

Release summary inspection from
`/tmp/tetra-surface-release-v1-current/surface-release-summary.json`:

- schema: `tetra.surface.release.v1`
- status: `current`
- release scope: `surface-v1-linux-web`
- supported targets: `headless`, `linux-x64`, `wasm32-web`
- unsupported targets: `macos-x64`, `windows-x64`, `wasm32-wasi`
- artifact hashes validated: true
- current `artifact-hashes.json` was refreshed after API docs were generated
  into the current report dir and then validated successfully

Dirty worktree status:

- The repository remains intentionally dirty with broad pre-existing compiler,
  docs, examples, scripts, tools, Graphify, and sidecar changes.
- No unrelated dirty work was reverted.
- The Section 22 broad-test blocker repair changed only safety/no-wrapper test
  fixtures and root-test exception documentation.

Release summary evidence:

- schema: `tetra.surface.release.v1`
- status: `current`
- release scope: `surface-v1-linux-web`
- supported targets: `headless`, `linux-x64`, `wasm32-web`
- unsupported targets: `macos-x64`, `windows-x64`, `wasm32-wasi`
- artifact hashes validated: true

## Known Limitations

- no GPU renderer
- no macOS/Windows Surface
- no dynamic trait-object widget ABI
- no witness-table dispatch
- no full rich text editor
- no full grapheme-cluster editing
- no arbitrary native platform widgets
- no React/DOM/user-JS application UI
