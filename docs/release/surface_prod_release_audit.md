# Surface Production Release Audit

Status: `PROD_STABLE_SCOPED`.

This audit is the public-claim source for the scoped Surface production path.
It promotes only the evidence-backed `PROD_STABLE_SCOPED_LINUX_WEB_APP_UI`
surface: Linux real-window UI, wasm32-web browser-canvas UI, deterministic
headless release evidence, and Tetra-owned Surface app UI that does not depend
on Electron, React, DOM UI, CSS runtime, or user JavaScript app logic.

The `CURRENT_HEAD` marker is intentional. `tools/cmd/validate-surface-prod-audit`
resolves it to `git rev-parse HEAD` at validation time, which lets the committed
audit document remain truthful for the exact commit that contains it.

Run the strict promotion check in the clean final checkout after the final
evidence bundle exists:

`go run -buildvcs=false ./tools/cmd/validate-surface-prod-audit --audit docs/release/surface_prod_release_audit.md --expected-status PROD_STABLE_SCOPED`

```json surface-prod-audit
{
  "schema": "tetra.surface.prod-audit.v1",
  "verdict": "PROD_STABLE_SCOPED",
  "level": "surface-prod-final-same-commit-audit-v1",
  "scope": "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
  "release_scope": "surface-v1-linux-web",
  "git_head": "CURRENT_HEAD",
  "git_dirty": false,
  "clean_checkout": true,
  "generated_at_utc": "2026-06-10T20:00:00Z",
  "blockers": [],
  "commands": [
    {"name": "git-head", "command": "git rev-parse HEAD", "status": "pass"},
    {"name": "git-status", "command": "git status --short", "status": "pass"},
    {"name": "full-go-test", "command": "go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1", "status": "pass"},
    {"name": "ci-test", "command": "bash scripts/ci/test.sh", "status": "pass"},
    {"name": "block-system-gate", "command": "bash scripts/release/surface/block-system-gate.sh --report-dir reports/surface-prod/final/block-system", "status": "pass"},
    {"name": "morph-gate", "command": "bash scripts/release/surface/morph-gate.sh --report-dir reports/surface-prod/final/morph", "status": "pass"},
    {"name": "release-gate", "command": "bash scripts/release/surface/release-gate.sh --report-dir reports/surface-prod/final/surface-v1", "status": "pass"},
    {"name": "prod-gate", "command": "bash scripts/release/surface/prod-gate.sh --report-dir reports/surface-prod/final/prod", "status": "pass"},
    {"name": "visual-gate", "command": "bash scripts/release/surface/visual-gate.sh --report-dir reports/surface-prod/final/visual", "status": "pass"},
    {"name": "perf-gate", "command": "bash scripts/release/surface/perf-gate.sh --report-dir reports/surface-prod/final/perf", "status": "pass"},
    {"name": "security-gate", "command": "bash scripts/release/surface/security-gate.sh --report-dir reports/surface-prod/final/security", "status": "pass"},
    {"name": "package-gate", "command": "bash scripts/release/surface/package-gate.sh --report-dir reports/surface-prod/final/package", "status": "pass"},
    {"name": "safe-view-lifetime-gate", "command": "bash scripts/release/safe-view-lifetime/gate.sh --report-dir reports/surface-prod/final/safe-view-lifetime", "status": "pass"},
    {"name": "api-stability-gate", "command": "bash scripts/release/surface/api-stability-gate.sh --report-dir reports/surface-prod/final/api-stability", "status": "pass"},
    {"name": "generate-manifest", "command": "go run -buildvcs=false ./tools/cmd/gen-manifest -o docs/generated/manifest.json", "status": "pass"},
    {"name": "validate-manifest", "command": "go run -buildvcs=false ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json", "status": "pass"},
    {"name": "verify-docs", "command": "go run -buildvcs=false ./tools/cmd/verify-docs --manifest docs/generated/manifest.json", "status": "pass"},
    {"name": "validate-prod-audit", "command": "go run -buildvcs=false ./tools/cmd/validate-surface-prod-audit --audit docs/release/surface_prod_release_audit.md --expected-status PROD_STABLE_SCOPED", "status": "pass"},
    {"name": "validate-artifact-hashes", "command": "go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --manifest reports/surface-prod/final/artifact-hashes.json", "status": "pass"},
    {"name": "diff-check", "command": "git diff --check", "status": "pass"},
    {"name": "manifest-clean", "command": "git diff --exit-code -- docs/generated/manifest.json", "status": "pass"},
    {"name": "git-status-final", "command": "git status --short", "status": "pass"}
  ],
  "reports": [
    {"name": "surface-prod-summary", "path": "reports/surface-prod/final/prod/surface-prod-summary.json", "schema": "tetra.surface.prod-gate-summary.v1", "git_head": "CURRENT_HEAD", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "surface-prod-gate", "path": "reports/surface-prod/final/prod/surface-release-v1/surface-prod-gate-report.json", "schema": "tetra.surface.prod-gate-report.v1", "git_head": "CURRENT_HEAD", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "block-system", "path": "reports/surface-prod/final/block-system/surface-block-system-gate-summary.json", "schema": "tetra.surface.block-system.gate.v1", "git_head": "CURRENT_HEAD", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "morph", "path": "reports/surface-prod/final/morph/surface-morph-gate-summary.json", "schema": "tetra.surface.morph.gate.v1", "git_head": "CURRENT_HEAD", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "surface-release", "path": "reports/surface-prod/final/surface-v1/surface-release-summary.json", "schema": "tetra.surface.release.v1", "git_head": "CURRENT_HEAD", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "visual", "path": "reports/surface-prod/final/visual/surface-visual-report.json", "schema": "tetra.surface.visual-regression.v1", "git_head": "CURRENT_HEAD", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "perf", "path": "reports/surface-prod/final/perf/surface-perf-report.json", "schema": "tetra.surface.perf-report.v1", "git_head": "CURRENT_HEAD", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "security", "path": "reports/surface-prod/final/security/surface-security-report.json", "schema": "tetra.surface.security-report.v1", "git_head": "CURRENT_HEAD", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "package", "path": "reports/surface-prod/final/package/surface-package-report.json", "schema": "tetra.surface.package-report.v1", "git_head": "CURRENT_HEAD", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "safe-view-lifetime", "path": "reports/surface-prod/final/safe-view-lifetime/safe-view-lifetime-summary.json", "schema": "tetra.safe-view-lifetime.gate.v1", "git_head": "CURRENT_HEAD", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "api-stability", "path": "reports/surface-prod/final/api-stability/surface-api-stability-summary.json", "schema": "tetra.surface.api-stability.v1", "git_head": "CURRENT_HEAD", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "electron-comparison", "path": "reports/surface-prod/final/prod/surface-electron-comparison/surface-electron-comparison-report.json", "schema": "tetra.surface.electron-comparison-report.v1", "git_head": "CURRENT_HEAD", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"},
    {"name": "artifact-hashes", "path": "reports/surface-prod/final/artifact-hashes.json", "schema": "tetra.release-artifact-hashes.v1alpha1", "git_head": "CURRENT_HEAD", "same_commit": true, "required": true, "artifact_hash_manifest": "reports/surface-prod/final/artifact-hashes.json"}
  ],
  "target_host_evidence": [
    {"target": "headless", "tier": "release-evidence", "evidence": "headless deterministic release evidence", "production": false, "unsupported_nonclaim": false},
    {"target": "linux-x64", "tier": "prod", "evidence": "linux-x64 real-window release evidence", "production": true, "unsupported_nonclaim": false},
    {"target": "wasm32-web", "tier": "prod", "evidence": "wasm32-web browser-canvas release evidence", "production": true, "unsupported_nonclaim": false},
    {"target": "windows-x64", "tier": "beta", "evidence": "unsupported production target-host boundary", "production": false, "unsupported_nonclaim": true},
    {"target": "macos-x64", "tier": "beta", "evidence": "unsupported production target-host boundary", "production": false, "unsupported_nonclaim": true},
    {"target": "wasm32-wasi", "tier": "unsupported", "evidence": "unsupported UI target", "production": false, "unsupported_nonclaim": true}
  ],
  "claim_governance": {
    "public_claim_source": "docs/release/surface_prod_release_audit.md",
    "prod_claim_validator": "tools/cmd/validate-surface-prod-claim",
    "final_audit_validator": "tools/cmd/validate-surface-prod-audit",
    "fake_claim_rejections": [
      "fake electron/react/css replacement rejected",
      "fake cross-platform support rejected",
      "fake gpu production claim rejected",
      "fake full accessibility parity rejected",
      "missing target-host evidence rejected"
    ],
    "unsupported_target_nonclaims": [
      "windows-x64",
      "macos-x64",
      "wasm32-wasi",
      "GPU production",
      "full accessibility parity",
      "broad Electron replacement"
    ]
  }
}
```

## Current Public Claim

Surface is production-stable for the scoped Linux/web Surface UI release matrix:
Linux real-window app UI, wasm32-web browser-canvas app UI, deterministic
headless release evidence, Tetra-owned rendering/layout/input/text/app-shell
flows, and no Electron/React/DOM/CSS/user-JS runtime dependency inside the
supported app UI path.

Surface is competitive with Electron in this supported scope, with public
comparison methodology and negative claim fixtures. This audit does not claim
broad Electron replacement, Windows/macOS production parity, GPU production,
external benchmark superiority, platform-native widgets, or full accessibility
parity.
