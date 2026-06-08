# Tetra Surface/UI Production Goal

<goal>
Implement the full Surface/UI production implementation plan in
`/home/tetra/Downloads/surface-ui-production-implementation-plan.md`.

Completion means Tetra's supported Surface v1/UI surface is production-ready in
the plan's explicit scoped sense: release evidence is fresh, same-commit, and
validated; P0 blockers are closed; P1 gaps are closed or explicitly accepted as
non-blocking limitations; headless, linux-x64 real-window, and wasm32-web
browser-canvas claims are target-scoped to current runtime evidence; fake,
stale, starter-only, or overbroad UI claims are rejected by validators and docs.
</goal>

<context>
Primary source of scope:

- `/home/tetra/Downloads/surface-ui-production-implementation-plan.md`

Read first on each continuation:

- `AGENTS.md`
- `GOAL.md`
- `PLAN.md`
- `ATTEMPTS.md`
- `NOTES.md`
- `CONTROL.md`
- Graphify MCP context for Surface runtime smoke, release Surface scripts,
  `validate-surface-runtime`, `validate-surface-release-state`,
  `tools/validators/surface`, wasm import validation, safe-view lifetime gates,
  CI/release workflows, docs/manifest validators, and Surface examples
- `graphify-out/GRAPH_REPORT.md` or `graphify-out/wiki/index.md` when local
  graph community context is useful

Important plan facts:

- First packet is `SURFPROD-P00`.
- Existing external audit was from a reconstructed dump without `.git`; final
  completion must be proven in this live repo.
- Known blockers from the plan include text-input PLIR proof mismatch, release
  report freshness, linux real-window target-host evidence, wasm browser smoke
  timeout/cleanup, safe-view lifetime timeout, CI/package release integration,
  and broad final gate evidence.
</context>

<constraints>
- Always communicate with the user in Ukrainian.
- Use Graphify MCP first for architecture/codebase navigation, then verify
  concrete files with normal repo inspection.
- Preserve unrelated dirty worktree changes.
- Use persistent Go caches under `.cache/` or `$HOME/.cache`; never set
  `GOCACHE` to `/tmp`.
- Do not use stale reports as current evidence.
- Do not use `/tmp` as trusted current proof.
- Keep reports under `reports/surface-ui-production-*`.
- Use `GOTELEMETRY=off` for Go evidence commands.
- Do not skip, weaken, delete, or rewrite validators merely to pass.
- Do not broaden production claims without same-commit runtime evidence.
- Do not claim full cross-platform UI parity, GPU rendering, platform-native
  widgets, DOM/React/user-JS app UI, rich text editor, full AT-SPI/screen-reader
  support, mature GUI-framework parity, or target support without current
  target-host evidence.
- macOS Surface, Windows Surface, wasm32-wasi UI, GPU rendering,
  platform-native widgets, dynamic trait-object widgets, witness-table
  component dispatch, full rich text editing, full AT-SPI/screen-reader support,
  DOM/React/user-JS application UI, full cross-platform UI runtime parity, and
  broad browser app framework support remain post-v1 non-goals unless the user
  explicitly changes scope.
- If a target-host precondition is unavailable, emit or preserve a blocked
  report and fail the release gate rather than promoting starter evidence.
- Do not touch Memory/IslandKernel except for a narrow compiler proof/lowering
  fix explicitly needed by `SURFPROD-P06`.
</constraints>

<scorecard>
Primary metric: packet tasks `SURFPROD-P00` through `SURFPROD-P13` are
implemented and verified against their packet done criteria.

Passing threshold:

- All P0 gaps from the external plan are closed.
- P1 gaps are closed or explicitly accepted as non-blocking with known
  limitation text and validator/docs guards.
- Release gate, experimental regression gate, safe-view lifetime gate, API
  stability gate, validators, CI/release workflow checks, docs/manifest checks,
  broad tests, and final artifact hash checks pass in this live repo.
- Final summary records exact git head, commands, artifacts, hashes, target
  scope, and non-goals.

Regression checks:

- Packet-specific RED/GREEN commands from the external plan.
- `bash -n` for changed shell scripts.
- `go test -buildvcs=false ./tools/validators/surface ./tools/cmd/validate-surface-runtime ./tools/cmd/validate-surface-release-state ./tools/cmd/validate-artifact-hashes ./tools/cmd/validate-wasm-imports -count=1`.
- `go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1`.
- focused race Surface regex gate before final completion.
- `bash scripts/ci/test.sh`.
- `go run -buildvcs=false ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`.
- `go run -buildvcs=false ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- `git diff --check`.
- `graphify update .` after code changes.

Stop condition:

- Stop and record a blocker if target-host linux/browser infrastructure is
  unavailable after the repo correctly emits blocked/failing evidence.
- Stop and ask before destructive cleanup, dependency changes, public scope
  expansion, broad cross-platform UI claim, or product decision not inferable
  from the plan/code.
- If the same focused/full gate fails twice for the same reason without new
  evidence, record the blocker before attempting a third variant.
</scorecard>

<done_when>
The goal is complete only when all are true:

- `SURFPROD-P00` truth audit script and baseline report exist and honestly
  record PASS/FAIL/BLOCKED/SKIPPED without claiming production readiness.
- `SURFPROD-P01` release gate rejects stale, non-empty, symlink, traversal, and
  starter-evidence report dirs and writes same-run metadata plus hashes.
- `SURFPROD-P02` validators reject fake, stale, copied, malformed,
  metadata-only, wrong-target, legacy-sidecar, unsupported-target, node-only,
  missing-artifact, pass-count mismatch, and runner-trace mismatch evidence.
- `SURFPROD-P03` headless release evidence is deterministic, fresh,
  hash-validated, runtime-derived, and cannot be replaced by metadata-only rows.
- `SURFPROD-P04` linux-x64 real-window release evidence passes on a valid target
  host; no-display hosts fail cleanly and cannot promote memfd starter evidence.
- `SURFPROD-P05` wasm32-web browser-canvas release evidence is bounded,
  cleanup-safe, validates wasm imports, and rejects Node-only starter evidence.
- `SURFPROD-P06` text/input/clipboard/IME release examples build and run; the
  `lib.core.text.insert_bytes` proof mismatch is fixed without weakening proof
  validation.
- `SURFPROD-P07` component tree/layout/toolkit release evidence validates
  measure/layout/draw order, component relations, bounds, dispatch, and scoped
  minimal widget coverage without platform-native widget claims.
- `SURFPROD-P08` accessibility metadata, linux bridge, and browser mirror claims
  are separately validated; full screen-reader/AT-SPI claims remain non-goals.
- `SURFPROD-P09` safe-view lifetime and Surface resource cleanup gate is
  bounded, reproducible, release-blocking, and covers close/frame/event/resize
  misuse plus browser/linux cleanup.
- `SURFPROD-P10` API stability, generated docs, manifest, and docs validators
  agree with the supported Surface scope.
- `SURFPROD-P11` CI and package release workflows cannot declare or publish
  Surface release readiness without running Surface gates, and release gates are
  not `continue-on-error`.
- `SURFPROD-P12` docs/user guides/examples index match code truth and contain
  no unsupported production overclaims.
- `SURFPROD-P13` final same-commit release candidate gate passes from a live
  repo with `.git` and fresh clean report dirs.
- All P0 gaps are closed.
- P1 gaps are closed or explicitly accepted as non-blocking with limitations.
- Release scope is explicit.
- Supported targets are only targets with current same-commit runtime evidence.
- Unsupported targets remain rejected by validators and docs.
- Release gate passes from a clean report dir.
- Experimental regression gate passes.
- Safe-view lifetime gate passes.
- API stability gate passes.
- Artifact hash manifest validates.
- Surface examples build/run for supported targets.
- Validators reject fake/stale/malformed evidence.
- CI has a non-optional Surface release readiness path.
- Package publishing cannot bypass Surface gate when Surface is shipped.
- Docs match code truth.
- Starter evidence is not promoted to release evidence.
- No stale `/tmp` evidence is used as current proof.
- No full cross-platform, GPU, native widget, DOM/React, rich text, or full
  screen-reader claim is made.
</done_when>

<feedback_loop>
Fast loop:

- For each packet, run its specified RED command first with a packet-specific
  `.cache/go-build-surface-pXX-*` cache.
- After implementation, run the packet GREEN command(s), `bash -n` for touched
  shell scripts, and `git diff --check`.
- Update `ATTEMPTS.md` and `GOAL.md ## Progress` after every meaningful RED,
  GREEN, blocker, or release-gate result.

Slower loop:

- Run focused validator/scripts package sweeps after each packet group.
- Run `graphify update .` after code changes.
- Run broad compiler/cli/tools gates only after major packet groups and before
  final completion.
</feedback_loop>

<workflow>
1. Re-read this file, `CONTROL.md`, and the external plan.
2. Use Graphify MCP first, then inspect concrete code.
3. Maintain the packet matrix for `SURFPROD-P00..P13`.
4. Execute the next packet with RED/GREEN tests when code changes.
5. Keep reports under `reports/surface-ui-production-*`.
6. Update `GOAL.md ## Progress`, `PLAN.md`, `ATTEMPTS.md`, `NOTES.md`, and
   `CONTROL.md`.
7. Run focused verification after each packet.
8. Run `graphify update .` after code changes.
9. Run broad/final gates before claiming a packet group or goal complete.
10. Mark the active goal complete only after a requirement-by-requirement audit
    proves every `done_when` item.
</workflow>

<working_memory>
Maintain:

- `PLAN.md`: packet matrix, current strategy, next batch, open decisions.
- `ATTEMPTS.md`: command evidence and RED/GREEN attempts.
- `NOTES.md`: durable discoveries, nonclaims, design rationale, blockers.
- `CONTROL.md`: operator control surface for the long goal.
- `.workflow/surface-ui-production-v1/`: workflow-local mirrors and final
  reports.
</working_memory>

<human_control_surface>
Create and maintain `CONTROL.md` as the compact human operator panel for this
goal. Before each phase change, strategic pivot, expensive step, or sidecar
ingestion, reread `CONTROL.md`. If it changed, summarize the relevant change in
`PLAN.md` and adapt before proceeding.
</human_control_surface>

<verification_loop>
Minimum final gate stack:

```bash
export GOTELEMETRY=off
export GOCACHE="$(pwd)/.cache/go-build-surface-prod-final"

bash scripts/release/surface/release-gate.sh --report-dir reports/surface-ui-production-final/surface-release-v1
bash scripts/release/surface/gate.sh --report-dir reports/surface-ui-production-final/surface-experimental-regression
bash scripts/release/safe-view-lifetime/gate.sh --report-dir reports/surface-ui-production-final/safe-view-lifetime
bash scripts/release/surface/api-stability-gate.sh --report-dir reports/surface-ui-production-final/surface-api-stability-v1

go test -buildvcs=false ./tools/validators/surface ./tools/cmd/validate-surface-runtime ./tools/cmd/validate-surface-release-state ./tools/cmd/validate-artifact-hashes ./tools/cmd/validate-wasm-imports -count=1
go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1
go test -race -buildvcs=false ./compiler/... ./cli/... ./tools/... -run 'Surface|surface|Draw|draw|Frame|Window|WASM|wasm|Accessibility|ToolKit|Toolkit|TextBox|UI|ui' -count=1
bash scripts/ci/test.sh

go run -buildvcs=false ./tools/cmd/gen-manifest -o docs/generated/manifest.json
go run -buildvcs=false ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run -buildvcs=false ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --manifest reports/surface-ui-production-final/surface-release-v1/artifact-hashes.json

git diff --check
git diff --exit-code -- docs/generated/manifest.json
git status --short
graphify update .
```
</verification_loop>

<execution_rules>
- Check git status before edits.
- Preserve unrelated user changes.
- Prefer `rg` over `grep`.
- Use `apply_patch` for manual edits.
- Read context files before implementation.
- Batch independent file reads in parallel when possible.
- Use the fastest representative feedback check while iterating.
- Run focused tests before broad tests.
- Do not paper over failures.
- Do not widen scope.
- Keep final answers concise.
</execution_rules>

<output_contract>
Final required artifacts:

- `reports/surface-ui-production-final/surface-release-v1/`
- `reports/surface-ui-production-final/surface-experimental-regression/`
- `reports/surface-ui-production-final/safe-view-lifetime/`
- `reports/surface-ui-production-final/surface-api-stability-v1/`
- `reports/surface-ui-production-final/final-summary.md`

Final response must summarize the implemented scope, exact final commands, pass
or blocked evidence, target scope, known non-goals, and goal usage after
`update_goal complete`.
</output_contract>

## Progress

- 2026-06-08: Active Surface/UI production goal received for
  `/home/tetra/Downloads/surface-ui-production-implementation-plan.md`.
  Existing top-level trackers still described the previous Memory/IslandKernel
  goal, so they were reset to this new `SURFPROD-P00..P13` contract while
  preserving old workflow artifacts under `.workflow/memory-*`. Graphify-first
  navigation identified Surface runtime smoke, release state validators,
  Surface validators, browser/linux evidence collectors, and release script
  tests as the initial codebase hubs. Bridge: start `SURFPROD-P00` truth audit
  with RED script-test coverage.
- 2026-06-08: `SURFPROD-P00` truth audit script implemented and verified.
  RED evidence:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p00-red go test -buildvcs=false ./tools/scriptstest -run 'SurfaceUITruthAudit|AnalysisScript' -count=1`
  failed before the script existed. GREEN evidence: `bash -n
  scripts/analysis/surface-ui-truth-audit.sh` and
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p00-green go test -buildvcs=false ./tools/scriptstest -run 'SurfaceUITruthAudit|AnalysisScript' -count=1`
  passed (`ok tetra_language/tools/scriptstest 10.484s`). Baseline evidence:
  `reports/surface-ui-production-audit/p00-baseline/truth-summary.md` records
  `production_ready_claim: false`, git head
  `3e489e567edc6ab7e537594313a9719a473aea38`, `12 PASS`, `1 BLOCKED`,
  `1 FAIL`, with required audit artifacts present and non-empty. Bridge:
  start `SURFPROD-P01` release gate hardening; carry P00 blockers into P05
  wasm browser and P09 safe-view lifetime work.
- 2026-06-08: `SURFPROD-P01` release-gate hardening slice implemented and
  verified. RED evidence:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p01-red go test -buildvcs=false ./tools/scriptstest -run 'SurfaceReleaseGateRejectsStale|SurfaceReleaseGateRejectsSymlink|SurfaceReleaseGateRejectsTraversal' -count=1`
  failed because invalid report dirs reached sub-gate execution and emitted
  `go: go.mod file not found`. GREEN evidence: `bash -n
  scripts/release/surface/release-gate.sh`, `bash -n
  scripts/release/surface/report-dir-guard.sh`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p01-green go test -buildvcs=false ./tools/scriptstest -run 'SurfaceReleaseGate' -count=1`,
  and
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p01-green go test -buildvcs=false ./tools/scriptstest -run 'ReleaseSurfaceFinalReleaseGate' -count=1`
  all passed. Bridge: start `SURFPROD-P02` validator fake/stale-evidence
  rejection; do not claim a full fresh release-gate pass until later packets
  close runtime/browser blockers.
- 2026-06-08: `SURFPROD-P02` validator metadata hardening slice implemented
  and verified. Initial plan RED had no failing new coverage, so new negative
  fixtures/tests were added for copied/missing producer metadata, stale
  `git_head`, and missing release command line. RED evidence then failed in
  `tools/validators/surface` because `ValidateReleaseSummary` accepted those
  stale/copy cases. GREEN evidence:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p02-green go test -buildvcs=false ./tools/validators/surface ./tools/cmd/validate-surface-runtime ./tools/cmd/validate-surface-release-state -run 'Stale|Copied|PassCount|MetadataOnly|WrongTarget|Legacy|Unsupported|NodeOnly|MissingArtifact|RunnerTrace' -count=1`
  and
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p02-green go test -buildvcs=false ./tools/validators/surface ./tools/cmd/validate-surface-runtime ./tools/cmd/validate-surface-release-state ./tools/cmd/validate-artifact-hashes -count=1`
  passed. Bridge: start `SURFPROD-P03` headless runtime evidence hardening.
- 2026-06-08: `SURFPROD-P03` headless runtime evidence slice implemented and
  verified. The plan RED initially selected no tests, so exact-name P03
  validator tests and an extra CLI RED were added; `validate-surface-runtime`
  failed on `--release headless` with `unsupported release "headless"`. GREEN
  evidence:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p03-green go test -buildvcs=false ./tools/cmd/surface-runtime-smoke ./tools/validators/surface -run 'HeadlessReleaseRequiresBuiltBinary|HeadlessRunnerTraceMatchesReport|HeadlessRejectsMetadataOnlyFrame|HeadlessNoLegacySidecars' -count=1`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p03-green bash scripts/release/surface/surface-headless-release-smoke.sh --report-dir reports/surface-ui-production-p03/headless-release`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p03-green go run ./tools/cmd/validate-surface-runtime --report reports/surface-ui-production-p03/headless-release/surface-headless-release.json --release headless`,
  and
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p03-green go run ./tools/cmd/validate-artifact-hashes --manifest reports/surface-ui-production-p03/headless-release/artifact-hashes.json`
  passed. Bridge: start `SURFPROD-P04` linux-x64 real-window/input lifecycle;
  headless evidence is not a substitute for linux/browser evidence.
- 2026-06-08: `SURFPROD-P04` linux-x64 release-window slice implemented and
  verified. The plan RED initially selected no tests, so
  `TestLinuxRealWindowRequiresWayland` was added; RED failed because a
  no-display fake repo reached `go run` instead of writing a controlled blocked
  report. GREEN evidence: `bash -n
  scripts/release/surface/surface-linux-x64-release-window-smoke.sh`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p04-green go test -buildvcs=false ./tools/validators/surface ./tools/scriptstest -run 'LinuxRealWindowRequiresWayland|LinuxMemfdCannotClaimRelease|LinuxReleaseRequiresNativeInput|LinuxReleaseRequiresCloseResizeTextClipboardAccessibility' -count=1`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p04-green bash scripts/release/surface/surface-linux-x64-release-window-smoke.sh --report-dir reports/surface-ui-production-p04/linux-window`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p04-green go run ./tools/cmd/validate-surface-runtime --report reports/surface-ui-production-p04/linux-window/surface-linux-x64-release-window.json --release linux-x64-real-window`,
  and `go run ./tools/cmd/validate-artifact-hashes --manifest
  reports/surface-ui-production-p04/linux-window/artifact-hashes.json` passed.
  Bridge: start `SURFPROD-P05` wasm32-web browser-canvas/input hardening.
- 2026-06-08: Fresh post-tracker `SURFPROD-P04` verification passed:
  `bash -n scripts/release/surface/surface-linux-x64-release-window-smoke.sh`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p04-verify go test -buildvcs=false ./tools/validators/surface ./tools/scriptstest ./tools/cmd/validate-surface-runtime -run 'LinuxRealWindowRequiresWayland|LinuxMemfdCannotClaimRelease|LinuxReleaseRequiresNativeInput|LinuxReleaseRequiresCloseResizeTextClipboardAccessibility|RealWindow|ReleaseWindow' -count=1`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p04-verify go run ./tools/cmd/validate-surface-runtime --report reports/surface-ui-production-p04/linux-window/surface-linux-x64-release-window.json --release linux-x64-real-window`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p04-verify go run ./tools/cmd/validate-artifact-hashes --manifest reports/surface-ui-production-p04/linux-window/artifact-hashes.json`,
  `git diff --check`, and `graphify update .` all passed; p04 caches were
  cleaned. Bridge: enter `SURFPROD-P05`.
- 2026-06-08: `SURFPROD-P05` wasm32-web browser-canvas/input slice
  implemented and verified. RED evidence first showed the plan regex selected
  no P05 tests, then new tests failed because the browser smoke lacked
  timeout/cleanup and `validate-surface-runtime` did not support
  `--release wasm32-web-browser`. Debugging found Chromium on this host could
  render `data:` URLs but did not issue localhost runner requests, so
  `runBrowserCanvasTrace` now uses a temporary file-backed runner with inline
  host JS and `--allow-file-access-from-files`. GREEN evidence:
  `bash -n scripts/release/surface/surface-wasm32-web-release-browser-smoke.sh`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p05-green go test -buildvcs=false ./tools/validators/surface ./tools/scriptstest ./tools/cmd/validate-wasm-imports ./tools/cmd/validate-surface-runtime ./tools/cmd/surface-runtime-smoke -run 'BrowserReleaseRequiresChromium|NodeOnlyCannotClaimBrowser|BrowserSmokeTimeoutCleansChildren|WasmImports|CanvasFrameInputTextAccessibility|BrowserCanvasRunnerDataURL|BrowserCanvasRunnerFileURL|RunBrowserCanvasTraceRetriesPendingTrace' -count=1`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p05-green bash scripts/release/surface/surface-wasm32-web-release-browser-smoke.sh --report-dir reports/surface-ui-production-p05/browser`,
  `go run ./tools/cmd/validate-surface-runtime --report reports/surface-ui-production-p05/browser/surface-wasm32-web-release-browser.json --release wasm32-web-browser`,
  `go run ./tools/cmd/validate-wasm-imports --target wasm32-web reports/surface-ui-production-p05/browser/surface-wasm32-web-release-browser-artifacts/surface-release-form.wasm`,
  `go run ./tools/cmd/validate-artifact-hashes --manifest reports/surface-ui-production-p05/browser/artifact-hashes.json`,
  `git diff --check`, and `graphify update .` passed; no leftover
  Chromium/surface processes were found. Bridge: start `SURFPROD-P06`.
- 2026-06-08: `SURFPROD-P06` RED captured. The planned build test and
  headless text-input smoke both currently pass in this repo, but the GREEN
  selector required by the external plan fails:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p06-red go run ./tools/cmd/validate-surface-runtime --report reports/surface-ui-production-p06/red-text-input/surface-headless-release-text-input.json --release text-input`
  reports `unsupported release "text-input"`. Bridge: add strict text-input
  release validation without weakening the existing `insert_bytes` proof or
  evidence validators.
- 2026-06-08: `SURFPROD-P06` text/input/clipboard/IME slice implemented and
  verified. The live `insert_bytes` proof mismatch from the external plan did
  not reproduce here, so the packet fix stayed narrow: `validate-surface-runtime`
  now supports strict `--release text-input`, the headless text-input smoke uses
  that selector, and `compiler/internal/lower/proof_bce_test.go` guards the
  `insert_bytes` loop shape so `bytes[i]` remains proof-tagged unchecked while
  the destination store remains checked. GREEN evidence:
  `bash -n scripts/release/surface/surface-headless-release-text-input-smoke.sh`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p06-green go test -buildvcs=false ./tools/cmd/validate-surface-runtime ./compiler/internal/lower -run 'TextInputReleaseValidatorAcceptsProductionTextInputReport|TextInsertBytesSourceLoopUsesProofTaggedUncheckedLoad' -count=1`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p06-green go test -buildvcs=false ./compiler/tests/semantics ./compiler/internal/plir ./compiler/internal/validation ./compiler/internal/lower -run 'SurfaceReleaseTextInput|TextInput|InsertBytes|Proof|Bounds' -count=1`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p06-green bash scripts/release/surface/surface-headless-release-text-input-smoke.sh --report-dir reports/surface-ui-production-p06/headless-text-input`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p06-green go run ./tools/cmd/validate-surface-runtime --report reports/surface-ui-production-p06/headless-text-input/surface-headless-release-text-input.json --release text-input`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p06-green go run ./tools/cmd/validate-artifact-hashes --manifest reports/surface-ui-production-p06/headless-text-input/artifact-hashes.json`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p06-green go test -buildvcs=false ./tools/cmd/surface-runtime-smoke ./tools/validators/surface ./tools/cmd/validate-surface-runtime -run 'TextInput|text input' -count=1`,
  `git diff --check`, and `graphify update .` passed. Bridge: start
  `SURFPROD-P07`.
- 2026-06-08: `SURFPROD-P07` RED captured. The external plan regex and the
  existing headless release toolkit smoke already pass, but a new validator
  negative for the plan's single-example production claim gap failed:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p07-red go test -buildvcs=false ./tools/validators/surface -run TestValidateSurfaceProductionToolkitRejectsSingleExampleClaim -count=1`
  accepted `example_count=1` with only `examples/surface_release_form.tetra` in
  toolkit `sources`. Bridge: require multi-example production toolkit evidence
  without broadening toolkit scope or claiming platform-native widgets.
- 2026-06-08: `SURFPROD-P07` component tree/toolkit slice implemented and
  verified. `ValidateReport` now rejects production toolkit reports with only a
  single scoped example and requires the release form, toolkit form, and toolkit
  settings examples for `production-widgets-v1`. The canonical report at
  `reports/surface-ui-production-p07/headless-toolkit/surface-headless-release-toolkit.json`
  validates with artifact hashes and records `example_count=3`, component tree,
  draw order, required widgets, and `no_platform_widgets=true`. GREEN evidence:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p07-green go test -buildvcs=false ./tools/validators/surface -run TestValidateSurfaceProductionToolkitRejectsSingleExampleClaim -count=1`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p07-green go test -buildvcs=false ./tools/validators/surface ./compiler/tests/semantics -run 'Toolkit|ComponentTree|Measure|Layout|Draw|Bounds|Dispatch|Checkbox|Scroll|TextBox|Button|Label' -count=1`,
  `bash -n scripts/release/surface/surface-headless-release-toolkit-smoke.sh`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p07-green bash scripts/release/surface/surface-headless-release-toolkit-smoke.sh --report-dir reports/surface-ui-production-p07/headless-toolkit`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p07-green go run ./tools/cmd/validate-surface-runtime --report reports/surface-ui-production-p07/headless-toolkit/surface-headless-release-toolkit.json --release surface-v1`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p07-green go run ./tools/cmd/validate-artifact-hashes --manifest reports/surface-ui-production-p07/headless-toolkit/artifact-hashes.json`,
  `git diff --check`, and `graphify update .` passed. Bridge: start
  `SURFPROD-P08`.
- 2026-06-08: `SURFPROD-P08` accessibility metadata/bridge slice implemented
  and verified. The external plan regex already passed, so a focused RED was
  added:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p08-red go test -buildvcs=false ./tools/cmd/validate-surface-runtime -run TestValidateSurfaceRuntimeReportReleaseModeRejectsAccessibilityClaimsWithoutTargetEvidence -count=1`
  failed because `validateSurfaceV1RuntimeReleaseReport` accepted
  accessibility release claims without linux/browser target evidence. A real
  wasm smoke also failed until `scripts/tools/surface_browser_canvas_host.mjs`
  emitted compiler-owned browser accessibility snapshot/mirror trace payload for
  `release-accessibility`. GREEN evidence:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p08-green bash scripts/release/surface/surface-headless-release-accessibility-smoke.sh --report-dir reports/surface-ui-production-p08/headless-accessibility`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p08-green bash scripts/release/surface/surface-linux-x64-release-accessibility-smoke.sh --report-dir reports/surface-ui-production-p08/linux-accessibility`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p08-green bash scripts/release/surface/surface-wasm32-web-release-accessibility-smoke.sh --report-dir reports/surface-ui-production-p08/wasm-accessibility`,
  explicit `validate-surface-runtime --release surface-v1` and
  `validate-artifact-hashes` for all three report dirs, and
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p08-green go test -buildvcs=false ./tools/cmd/validate-surface-runtime ./tools/cmd/surface-runtime-smoke ./tools/validators/surface -run 'Accessibility|BrowserMirror|PlatformProbe|ScreenReader|LinuxBridge|ReleaseModeAcceptsReleaseEvidenceSlices|AccessibilityClaimsWithoutTargetEvidence|ReleaseAccessibilityModesProducePlatformBridgeEvidence' -count=1`
  passed. Canonical reports are under `reports/surface-ui-production-p08/`;
  headless records `headless_platform_tree_probe`, linux records
  `linux_platform_probe=true` plus `linux-accessibility-platform-probe`, and
  wasm records browser snapshot/mirror host flags plus runner trace payload.
  Post-tracker verification: `git diff --check`, the focused P08 package test
  rerun, and `graphify update .` passed; Graphify rebuilt `22007` nodes and
  `68520` edges, and P08 caches were cleaned. Bridge: start `SURFPROD-P09`.
- 2026-06-08: `SURFPROD-P09` safe-view lifetime/resource cleanup slice
  implemented and verified. Initial plan RED passed without P09 script
  coverage, so `tools/scriptstest/surface_safe_view_lifetime_test.go` added
  focused RED coverage for bounded gate steps and browser/linux cleanup
  contracts; it failed on missing `safe_view_lifetime_cleanup()` and
  `surface_linux_x64_release_window_cleanup()`. GREEN changes keep the
  safe-view gate Surface-focused and bounded with per-step logs and a
  `safe-view-lifetime-summary.json` containing `bounded=true` and
  `release_blocking=true`, add timeout/cleanup wrappers to linux release-window
  smoke, and keep final linux artifact-hash validation unlogged so the report
  dir is not mutated after `artifact-hashes.json` is sealed. Evidence:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p09-green bash scripts/release/safe-view-lifetime/gate.sh --report-dir reports/surface-ui-production-p09/safe-view-lifetime`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p09-green go test -race -buildvcs=false ./compiler/internal/semantics ./compiler/tests/semantics ./tools/scriptstest -run 'Surface|surface|SafeView|ResourceCleanup' -count=1`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p09-green SURFACE_LINUX_RELEASE_WINDOW_TIMEOUT_SECONDS=120 bash scripts/release/surface/surface-linux-x64-release-window-smoke.sh --report-dir reports/surface-ui-production-p09/linux-window-cleanup`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p09-verify go test -buildvcs=false ./tools/scriptstest -run 'SurfaceSafeViewLifetime|BrowserSmokeTimeoutCleansChildren|LinuxRealWindowRequiresWayland|ReleaseSurfaceSmokeScriptsUseStrictReleaseValidation' -count=1`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p09-verify go run ./tools/cmd/validate-artifact-hashes --manifest reports/surface-ui-production-p09/linux-window-cleanup/artifact-hashes.json`,
  `git diff --check`, no leftover Surface process check, and `graphify update .`
  passed; Graphify rebuilt `22010` nodes and `68524` edges. Bridge: start
  `SURFPROD-P10`.
- 2026-06-08: `SURFPROD-P10` API stability/docs generation slice implemented
  and verified. Focused RED coverage failed until docs rejected `/tmp` current
  release evidence, manifest validation required current/unsupported Surface
  feature rows, and `api-stability-gate.sh` wrote
  `public-surface-api-summary.txt`. GREEN evidence:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p10-green go test -buildvcs=false ./tools/cmd/verify-docs ./tools/cmd/validate-manifest ./tools/scriptstest -run 'SurfaceAPIStability|SurfaceDocsGenerated|SurfaceDocsOverclaim|ManifestSurface|ValidateFeaturesRequiresMemoryProductionFinalAuditDocs' -count=1`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p10-green bash scripts/release/surface/api-stability-gate.sh --report-dir reports/surface-ui-production-p10/surface-api-stability-v1`,
  `go run -buildvcs=false ./tools/cmd/gen-manifest -o docs/generated/manifest.json`,
  `go run -buildvcs=false ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`,
  `go run -buildvcs=false ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`,
  temp manifest idempotence diff, `bash -n
  scripts/release/surface/api-stability-gate.sh`, `git diff --check`, and
  `graphify update .` passed; Graphify rebuilt `22013` nodes and `68532` edges.
  Exact `git diff --exit-code -- docs/generated/manifest.json` remains non-zero
  only because P10 updates the tracked generated manifest with
  `core.island_reset`. Bridge: start `SURFPROD-P11`.
- 2026-06-08: `SURFPROD-P11` CI/package release-readiness slice implemented
  and verified. Exact-name RED tests first exposed that
  `release-packages.yml` could publish after only the Memory production gate.
  GREEN changes run Surface release, experimental regression, safe-view lifetime,
  and API stability gates before package upload, GitHub release, container, or
  Homebrew publishing paths, upload those Surface report dirs, and keep release
  gates free of `continue-on-error: true`. Evidence:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p11-green go test -buildvcs=false ./tools/scriptstest -run 'SurfaceReleaseReadinessWorkflow|ReleasePackagesRunsSurfaceGate|NoContinueOnError' -count=1`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p11-green go test -buildvcs=false ./tools/scriptstest -run 'Workflow|SurfaceRelease|ReleasePackages' -count=1`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p11-green go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7`,
  `git diff --check`, and `graphify update .` passed; Graphify rebuilt
  `22016` nodes and `68538` edges. Bridge: start `SURFPROD-P12`.
- 2026-06-08: `SURFPROD-P12` docs/nonclaims/user guide slice implemented and
  verified. Exact-name RED tests first showed `verify-docs` did not reject
  GPU/native-widget/cross-platform/rich-text/screen-reader/React Surface
  promotion claims. GREEN changes add those guards, tighten line-local nonclaim
  wording in Surface docs, document `WAYLAND_DISPLAY`/`DISPLAY`, browser
  dependency, blocked-report, and starter-vs-release troubleshooting, and list
  `examples/surface_release_counter.tetra` with release-supported Surface
  examples. Evidence:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p12-green go test -buildvcs=false ./tools/cmd/verify-docs -run 'SurfaceOverclaim|UnsupportedSurfaceTargets|GPU|NativeWidgets|RichText|ScreenReader|DOM|React|UserJS|CrossPlatform' -count=1`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p12-green go run -buildvcs=false ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`,
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-surface-p12-green go run -buildvcs=false ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`,
  `git diff --check`, and `graphify update .` passed; Graphify rebuilt
  `22020` nodes and `68550` edges. Bridge: start `SURFPROD-P13`.
- 2026-06-08: `SURFPROD-P13` final candidate evidence collected, but final
  goal completion is blocked by one exact cleanliness check. Focused P13
  failures were fixed without weakening validators: `compiler/formal_core_v1.go`
  now models the P23 proof witness with `local:i` plus a typed `ProofTerm`,
  `reports/optimizer-core-coverage-v1/closure.md` provides the required P17.1
  closure artifact, and `scripts/release/surface/report-dir-guard.sh` has the
  portable strict shell header. Evidence passed:
  `go test -buildvcs=false ./compiler -run 'P23FormalCoreV1' -count=1`,
  `go test -buildvcs=false ./compiler/internal/opt -run TestOptimizerCoreCoverageDocsRecordP17Closure -count=1`,
  `go test -buildvcs=false ./tools/scriptstest -run TestShellScriptsUsePortableBashSafetyHeader -count=1`,
  the fresh final Surface release/experimental/safe-view/API gates under
  `reports/surface-ui-production-final/`, focused validator packages, broad
  `go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1`,
  focused race regex, `bash scripts/ci/test.sh`, docs/manifest/hash validators,
  `git diff --check`, manifest idempotence diff, and `graphify update .`
  (`22080` nodes, `68654` edges). `reports/surface-ui-production-final/final-summary.md`
  records artifacts, hashes, target scope, and non-goals. Blocker:
  `git diff --exit-code -- docs/generated/manifest.json` is non-zero only for
  the intended generated `core.island_reset` row; do not call `update_goal
  complete` until this final exact check is resolved or explicitly accepted.
