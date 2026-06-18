# Tetra Surface v1 Release Audit

Status: P20 final audit complete with `NEAR_READY_WITH_BLOCKERS` verdict for
the bounded `surface-v1-linux-web` release scope.

This audit tracks the honest current claim:

```text
Tetra Surface v1 is current/release-ready for linux-x64 real-window and
wasm32-web browser-canvas Surface scope, with headless as release evidence
target.
```

This is not a Native Surface Host v1 completion claim. `linux-x64-real-window`
remains release/probe evidence for `surface-v1-linux-web`; it must not be used
as `tetra.surface.native-host.v1` proof.

It does not claim macOS Surface, Windows Surface, wasm32-wasi Surface UI, GPU
rendering, platform-native widgets, dynamic trait-object widgets, witness-table
component dispatch, full rich text editing, full AT-SPI/screen-reader support,
or DOM/React/user-JS application UI.

## Evidence Snapshot

- Release scope: `surface-v1-linux-web`.
- Supported runtime targets: `linux-x64`, `wasm32-web`.
- Release evidence target: `headless`.
- Native Surface Host v1 evidence is separate and currently requires the strict
  `linux-x64-native-surface-host-v1` gate; old real-window probe evidence is a
  nonclaim for that stronger proof.
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
- Developer fast loop evidence:
  `surface-dev-workflow-v1` reports from `tetra surface dev`, with fast rebuild
  token/recipe/source changed rebuilds and no hot reload claim.
- Surface inspector evidence:
  `surface-inspector-v1` reports from `tools/cmd/surface-inspector`, with Block
  tree, Morph tokens, layout, paint, accessibility, event route, focus,
  perf-counter, source-location, hidden-state scan, JSON, and static HTML tool
  report evidence.
- P28 docs-governance evidence:
  `PROD_STABLE_SCOPED`, `BETA_TARGET_HOST`, `EXPERIMENTAL`, `UNSUPPORTED`, and
  `NONCLAIM` tier vocabulary stays present across the Surface release docs,
  generated manifest references, user guide, examples index, and cookbook docs.
- Product evidence gate:
  `bash scripts/release/surface/product-gate.sh --report-dir reports/surface-product-v1`
  runs the release gate, artifact hash validation, claim scanner, manifest
  validator, and docs verifier. It is not the final `PROD_STABLE_SCOPED`
  verdict; P29 owns the final same-commit product readiness audit.

## Checklist

### Release Contract Created

- Status: done.
- Evidence: `docs/release/surface_v1_release_contract.md`.

### Feature Registry Updated

- Status: done.
- Evidence: `compiler/features.go`; manifest lists current Surface IDs,
  including `ui.surface-inspector-v1` and unsupported target IDs.

### Docs Updated

- Status: done.
- Evidence: Surface spec, supported-surface spec, user guide, examples index,
  and release notes were updated.

### Manifest Updated

- Status: done.
- Evidence: `docs/generated/manifest.json`; `validate-manifest` passed.

### P28 Claim-Tier Governance Documented

- Status: done.
- Evidence: `PROD_STABLE_SCOPED`, `BETA_TARGET_HOST`, `EXPERIMENTAL`,
  `UNSUPPORTED`, and `NONCLAIM` vocabulary plus product-gate/docs/claim scanner
  commands.

### Experimental Block-System Gate Evidence Recorded

- Status: done.
- Evidence: `reports/surface-block/p18-budget/surface-block-system-gate-summary.json`.
- Evidence detail: `block_system.memory_budget`; no production Block claim.

### Experimental Morph Capsule Gate Evidence Recorded

- Status: done.
- Evidence: `reports/surface-morph/gate/surface-morph-gate-summary.json`.
- Evidence detail: `tetra.surface.morph.v1`; no production Morph claim.

### API Docs Generated

- Status: done.
- Evidence: API stability gate generated and validated `artifacts/tetra-docs.md`
  with 7 stable Surface modules.

### Release Examples Added

- Status: done.
- Evidence: release form, text input, and accessibility examples were added.

### Text/Input Gate Passed

- Status: done.
- Evidence: Surface v1 release gate generated text-input reports for headless,
  linux-x64, and wasm32-web.

### Clipboard Gate Passed

- Status: done.
- Evidence: Surface v1 release gate summary reports `clipboard-text-v1`.

### IME/Composition Gate Passed

- Status: done.
- Evidence: Surface v1 release gate summary reports `composition-baseline-v1`.

### Toolkit Gate Passed

- Status: done.
- Evidence: Surface v1 release gate generated toolkit reports for headless,
  linux-x64, and wasm32-web.

### Developer Fast Loop Gate Passed

- Status: done.
- Evidence: Surface v1 release gate generated `surface-dev-workflow.json` with
  `surface-dev-workflow-v1` fast rebuild evidence.

### Surface Inspector Gate Passed

- Status: done.
- Evidence: Surface v1 release gate generated `surface-inspector.json` with
  `surface-inspector-v1` static tool evidence.

### Linux Accessibility Bridge Gate Passed

- Status: done.
- Evidence: `reports/surface-ui-production-p08/linux-accessibility`
  release-accessibility JSON.

### Browser Accessibility Snapshot Gate Passed

- Status: done.
- Evidence: `reports/surface-ui-production-p08/wasm-accessibility`
  release-accessibility JSON.

### Release Gate Passed

- Status: done.
- Evidence: `reports/surface-block/final/surface-release-v1`
  release-summary JSON.

### Experimental Regression Gate Passed

- Status: done.
- Evidence: `reports/surface-block/final/surface-experimental-regression`
  artifact hashes.

### Safe-View-Lifetime Gate Passed

- Status: done.
- Evidence: `reports/surface-ui-production-p09/safe-view-lifetime` summary has
  schema `tetra.safe-view-lifetime.gate.v1`.

### Full Tests Passed

- Status: done.
- Evidence: P20 ran compiler/CLI/tools package tests, broad package tests, and
  `bash scripts/ci/test.sh`.

### Artifact Hashes Validated

- Status: done.
- Evidence: P20 validated every `artifact-hashes.json` under
  `reports/surface-block/final`.

### Reports Preserved

- Status: done with blocker.
- Evidence: `reports/surface-block/final/final-readiness-audit.md`.
- Evidence detail: verdict is `NEAR_READY_WITH_BLOCKERS` because final
  summaries record `git_dirty: true`.

### Unsupported Targets Documented

- Status: done.
- Evidence: release contract, Surface spec, supported-surface spec, guide,
  release notes, and feature registry.

### No Fake Production Claims Accepted

- Status: done.
- Evidence: release-state validator rejects missing Surface feature IDs, and
  release validators reject unsupported/fake Surface claims.

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
surface_release_tests='ValidateFeatures|VerifyFeatureRegistry|SurfaceReleaseState'
surface_release_tests="$surface_release_tests|FeatureRegistryDeclaresSurface"
surface_release_tests="$surface_release_tests|FeatureRegistryCoversReleaseStatuses"
surface_release_tests="$surface_release_tests|ManifestBuiltins"

GOCACHE=$(pwd)/.cache/go-build-surface-release \
  go test ./tools/cmd/validate-manifest ./tools/cmd/verify-docs \
  ./tools/cmd/validate-surface-release-state ./compiler/tests/semantics \
  -run "$surface_release_tests" \
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
