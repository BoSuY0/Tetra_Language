# Full Platform UI Runtime Implementation Plan

**Goal:** Promote the UI runtime gate surface from Linux/Web-only evidence toward a strict full-platform contract without rewriting the v0.4.0 release truth.
**Context:** The current repo has Linux native UI and browser-backed web UI smoke paths. Windows/macOS target-host UI runtime files are absent in the real worktree even though stale Graphify artifacts mention them. The final production claim must stay blocked until real Windows and macOS target-host reports exist.
**Execution:** Implement locally with TDD-style validator and script tests, then run the requested gates. Do not mark READY unless all platform evidence and clean-worktree checks pass.

## Tasks

1. **Full-platform report contract**
   - **Files:** add `tools/validators/uiplatform`, `tools/cmd/validate-windows-ui-runtime`, `tools/cmd/validate-macos-ui-runtime`, `tools/cmd/validate-cross-platform-ui-runtime`.
   - **Approach:** define `tetra.ui.platform.v1` as the platform UI runtime evidence schema for Windows/macOS target-host reports. Require `status: pass`, `host == target`, real process/window/widget/event/case evidence, and reject docs-only/build-only/metadata-only/runtime-less/fake/mock/placeholder/startup_failure evidence.
   - **Verification:** targeted `go test` for new validators, then `go test ./tools/... -count=1`.
   - **Done when:** fake or blocked reports fail, valid target-host reports pass.

2. **Full-platform smoke scripts and gate**
   - **Files:** add `scripts/release/full_platform/windows-ui-runtime-smoke.sh`, `macos-ui-runtime-smoke.sh`, `ui-runtime-gate.sh`, and README.
   - **Approach:** accept externally produced target-host evidence with validation, write explicit blocked reports on non-target hosts without claiming production, and make the gate run baseline, Linux, Windows, macOS, Web, cross-platform validation, docs/manifest checks, and artifact hashes.
   - **Verification:** script structure tests plus direct non-host smoke run should fail with a blocked report.
   - **Done when:** missing/blocked Windows/macOS evidence stops the gate.

3. **Target metadata and feature/docs truth**
   - **Files:** update `compiler/target/target.go`, `cli/cmd/tetra/metadata.go`, `compiler/manifest.go`, validators, `compiler/features.go`, `docs/spec/current_supported_surface.md`, `docs/spec/ui_v1.md`, `docs/spec/runtime_abi.md`, `docs/user/wasm_ui_guide.md`, and regenerated `docs/generated/manifest.json`.
   - **Approach:** expose truthful UI runtime status per target: Linux and Web have current runtime evidence paths; Windows/macOS require target-host evidence before production; WASI/build-only targets do not provide UI runtime dispatch.
   - **Verification:** `go run ./tools/cmd/gen-manifest`, `verify-docs`, `validate-manifest`, `validate-targets`.
   - **Done when:** docs/features/targets do not imply Windows/macOS production from build-only or remote-blocked evidence.

4. **Web UI runtime marker hardening**
   - **Files:** update `scripts/release/v1_0/web-smoke.sh` and `tools/cmd/validate-web-ui-smoke`.
   - **Approach:** require browser trace markers for real WASM instantiation, DOM mount, core widget surfaces, event dispatch, state/render updates, timer/async/redraw/error-recovery markers, and UI bundle/DOM artifacts.
   - **Verification:** validator tests and `bash scripts/release/v1_0/web-smoke.sh --report reports/full-platform-ui-runtime/web-smoke.json`.
   - **Done when:** reports without the expanded runtime trace fail validation.

5. **Final evidence run**
   - **Commands:** run the requested baseline, platform smokes, cross-platform validator, full gate, `git diff --check`, and `git status --porcelain --untracked-files=all`.
   - **Done when:** final answer says `READY` only if all commands pass and the worktree is clean; otherwise `NOT READY` with exact blockers.
