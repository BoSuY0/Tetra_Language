# Surface Block Release Gate Implementation Plan

**Goal:** Make `SURFBLOCK-P17` release evidence mandatory by wiring Block-system smokes into the Surface release gate and CI artifact flow.
**Context:** `SURFBLOCK-P12` through `SURFBLOCK-P16` produced headless, linux-x64, wasm32-web, example, and validator evidence. The existing Surface v1 release gate still runs only the pre-Block Surface reports.
**Execution:** Use TDD: add failing script tests first, then add the gate and workflow integration, then verify with shell syntax, targeted Go tests, local gate runs, and `graphify update .`.

## Task 1 - RED Script Tests

**Goal:** Prove the current release machinery can omit Block evidence.

**Files:** `tools/scriptstest/surface_block_release_gate_test.go`, `tools/scriptstest/ci_workflow_test.go`.

**Approach:** Add static tests that require a strict `block-system-gate.sh`, require `release-gate.sh` to run it before the summary/validators, require report-dir guard rejection for stale/symlink dirs, and require CI to expose/upload Block reports without `continue-on-error`.

**Verification:** `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surfblock-p17-red GOTMPDIR=$(pwd)/.cache/go-tmp-surfblock-p17-red go test -buildvcs=false ./tools/scriptstest -run 'SurfaceBlock|ReleaseGate|ReportDir|CI' -count=1`.

**Done when:** The new tests fail for missing Block release gate integration.

## Task 2 - Block System Gate

**Goal:** Add one strict release subgate for Block evidence.

**Files:** `scripts/release/surface/block-system-gate.sh`.

**Approach:** Parse `--report-dir`, guard it with `surface_release_require_fresh_report_dir`, run the headless, linux-x64 real-window, and wasm32-web browser-canvas Block smoke scripts into target subdirectories, validate each report with `validate-surface-block-report --same-commit "$git_head"`, write a small gate summary, and validate root artifact hashes.

**Verification:** `bash -n scripts/release/surface/block-system-gate.sh` and `bash scripts/release/surface/block-system-gate.sh --report-dir reports/surface-block/gate`.

**Done when:** The gate emits same-commit Block reports and fails honestly when platform/browser prerequisites are missing.

## Task 3 - Release Gate Integration

**Goal:** Make the existing Surface v1 release gate require Block evidence.

**Files:** `scripts/release/surface/release-gate.sh`.

**Approach:** Preserve the existing fresh report-dir guard, keep the original relative report dir for subgate paths, run `block-system-gate.sh` in `"$report_dir_arg/block-system"` before writing the release summary, add Block report paths to required report checks, and record Block gate metadata in the summary.

**Verification:** `bash -n scripts/release/surface/release-gate.sh` and `bash scripts/release/surface/release-gate.sh --report-dir reports/surface-release-v1`.

**Done when:** `release-gate.sh` cannot complete unless Block gate evidence is present.

## Task 4 - CI Artifact Hardening

**Goal:** Ensure CI keeps Block release evidence visible and non-optional.

**Files:** `.github/workflows/ci.yml`, `tools/scriptstest/ci_workflow_test.go`.

**Approach:** Add explicit Block report artifact paths to the Surface release readiness upload and test that the job section has no `continue-on-error`.

**Verification:** `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surfblock-p17-green GOTMPDIR=$(pwd)/.cache/go-tmp-surfblock-p17-green go test -buildvcs=false ./tools/scriptstest -run 'SurfaceBlock|ReleaseGate|ReportDir|CI' -count=1`.

**Done when:** CI uploads Block reports and the required readiness job remains strict.

## Task 5 - Evidence Closeout

**Goal:** Record P17 evidence without claiming broader production readiness.

**Files:** `GOAL.md`, `reports/surface-block/gate/`, `reports/surface-release-v1/`.

**Approach:** Run the GREEN commands from the source plan, validate artifact hashes, run `git diff --check`, run `graphify update .`, and update `GOAL.md` only after fresh evidence exists.

**Verification:** The P17 GREEN command list passes in the current repo state.

**Done when:** `SURFBLOCK-P17` in `GOAL.md` has concrete command/artifact evidence and P18 remains the next open packet.
