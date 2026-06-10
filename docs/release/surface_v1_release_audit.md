# Tetra Surface v1 Release Audit

Status: P20 final audit complete with `NEAR_READY_WITH_BLOCKERS` verdict for
the bounded `surface-v1-linux-web` release scope.

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
- Experimental Block-system gate scope:
  `tetra.surface.block-system.gate.v1`.
- Experimental Block-system evidence path:
  `reports/surface-block/p18-budget/surface-block-system-gate-summary.json`.
- Experimental Block-system memory budget evidence:
  `block_system.memory_budget` in the headless, linux-x64 real-window, and
  wasm32-web browser-canvas reports under `reports/surface-block/p18-budget`.
- Block-system nonclaim: P18 proves scoped same-commit experimental Block
  reports, not production Block support.
- Experimental Morph Capsule gate scope:
  `tetra.surface.morph.gate.v1`.
- Experimental Morph Capsule evidence path:
  `reports/surface-morph/gate/surface-morph-gate-summary.json`.
- Morph nonclaim: Morph proves deterministic headless recipe expansion into
  Block evidence, not production Morph support.
- P09 safe-view lifetime report:
  `reports/surface-ui-production-p09/safe-view-lifetime/safe-view-lifetime-summary.json`.
- P10 API stability report:
  `reports/surface-ui-production-p10/surface-api-stability-v1/surface-api-stability-summary.json`.
- Final release gate target path:
  `reports/surface-ui-production-final/surface-release-v1/surface-release-summary.json`.
- Final artifact hash manifest target path:
  `reports/surface-ui-production-final/surface-release-v1/artifact-hashes.json`.

## Checklist

| Item | Status | Evidence |
| --- | --- | --- |
| release contract created | done | `docs/release/surface_v1_release_contract.md` |
| feature registry updated | done | `compiler/features.go`; manifest lists 8 Surface current IDs and 3 unsupported target IDs |
| docs updated | done | `docs/spec/surface_v1.md`, `docs/spec/current_supported_surface.md`, `docs/user/surface_guide.md`, `docs/user/examples_index.md`, `docs/release/surface_v1_release_notes.md` |
| manifest updated | done | `docs/generated/manifest.json`; `validate-manifest` passed |
| experimental Block-system gate evidence recorded | done | `reports/surface-block/p18-budget/surface-block-system-gate-summary.json`; `block_system.memory_budget`; no production Block claim |
| experimental Morph Capsule gate evidence recorded | done | `reports/surface-morph/gate/surface-morph-gate-summary.json`; `tetra.surface.morph.v1`; no production Morph claim |
| API docs generated | done | `scripts/release/surface/api-stability-gate.sh --report-dir reports/surface-ui-production-p10/surface-api-stability-v1` generated and validated `artifacts/tetra-docs.md` with 7 stable Surface modules |
| release examples added | done | `examples/surface_release_form.tetra`, `examples/surface_release_text_input.tetra`, `examples/surface_release_accessibility.tetra` |
| text/input gate passed | done | Surface v1 release gate generated text-input reports for headless, linux-x64, and wasm32-web |
| clipboard gate passed | done | Surface v1 release gate summary reports `clipboard-text-v1` |
| IME/composition gate passed | done | Surface v1 release gate summary reports `composition-baseline-v1` |
| toolkit gate passed | done | Surface v1 release gate generated toolkit reports for headless, linux-x64, and wasm32-web |
| Linux accessibility bridge gate passed | done | `reports/surface-ui-production-p08/linux-accessibility/surface-linux-x64-release-accessibility.json` |
| browser accessibility snapshot gate passed | done | `reports/surface-ui-production-p08/wasm-accessibility/surface-wasm32-web-release-accessibility.json` |
| release gate passed | done | `reports/surface-block/final/surface-release-v1/surface-release-summary.json` |
| experimental regression gate passed | done | `reports/surface-block/final/surface-experimental-regression/artifact-hashes.json` |
| safe-view-lifetime gate passed | done | `reports/surface-ui-production-p09/safe-view-lifetime/safe-view-lifetime-summary.json` has schema `tetra.safe-view-lifetime.gate.v1` |
| full tests passed | done | P20 ran `go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1`, `go test -buildvcs=false ./... ./compiler/... ./cli/... ./tools/... -count=1`, and `bash scripts/ci/test.sh` |
| artifact hashes validated | done | P20 validated every `artifact-hashes.json` under `reports/surface-block/final` |
| reports preserved | done with blocker | `reports/surface-block/final/final-readiness-audit.md`; verdict is `NEAR_READY_WITH_BLOCKERS` because final summaries record `git_dirty: true` |
| unsupported targets documented | done | release contract, Surface spec, supported-surface spec, guide, release notes, and feature registry |
| no fake production claims accepted | done | release-state validator rejects missing Surface feature IDs; release validators reject unsupported/fake Surface claims |

## P20 Final Readiness Verdict

P20 final audit artifact:
`reports/surface-block/final/final-readiness-audit.md`.

Verdict: `NEAR_READY_WITH_BLOCKERS`.

The P20 matrix passed broad Go tests, `scripts/ci/test.sh`, Block System gate,
Surface v1 release gate, experimental Surface regression gate, safe-view
lifetime gate, Surface API stability gate, docs/manifest validators, final
artifact-hash validation, release-state validation, and same-commit Block
report validation. It is not promoted to `PROD_READY_SCOPED` because the final
Surface release and Block System summaries both record `git_dirty: true`, and
`reports/surface-block/final/git-status-short.txt` records a dirty checkout.
Rerun the same P20 matrix from a clean committed checkout before making a final
production-ready scoped signoff.

## Final Commands To Rerun

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
  bash scripts/release/surface/block-system-gate.sh \
  --report-dir reports/surface-block/p18-budget

GOCACHE=$(pwd)/.cache/go-build-surface-release \
  bash scripts/release/surface/release-gate.sh \
  --report-dir reports/surface-ui-production-final/surface-release-v1

GOCACHE=$(pwd)/.cache/go-build-surface-release \
  go run ./tools/cmd/validate-surface-release-state \
  --report-dir reports/surface-ui-production-final/surface-release-v1 \
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
