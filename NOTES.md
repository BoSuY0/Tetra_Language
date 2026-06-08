# Tetra Surface/UI Production Notes

## Durable Discoveries

- The active goal is Surface/UI production from
  `/home/tetra/Downloads/surface-ui-production-implementation-plan.md`, not the
  previous Memory/IslandKernel contract.
- The external plan was produced from a reconstructed dump without `.git`, so
  every completion claim must be reproven in this live repo.
- `SURFPROD-P00` is first by plan design: it collects reproducible truth
  evidence without becoming a fake release gate.
- Current known blockers from the plan: release text-input example PLIR proof
  mismatch in `lib.core.text.insert_bytes`, linux real-window evidence needing
  `WAYLAND_DISPLAY`/target host, wasm32-web browser release smoke timeout,
  safe-view lifetime timeout, weak report-dir freshness, manual-only or
  bypassable CI/package release gates, and final broad test stack proof.
- Non-goals/post-v1: macOS Surface, Windows Surface, wasm32-wasi UI, GPU
  rendering, platform-native widgets, dynamic trait-object widgets,
  witness-table component dispatch, full rich text editor, full
  AT-SPI/screen-reader support, DOM/React/user-JS app UI, full cross-platform
  runtime parity, mature GUI framework parity.
- Graphify snapshot in `graphify-out/GRAPH_REPORT.md` was built from commit
  `9b244e73`, so use MCP graph context as navigation but verify files directly.
- `scripts/analysis/surface-ui-truth-audit.sh` is an evidence collector only:
  it writes `production_ready_claim: false`, records PASS/FAIL/BLOCKED/SKIPPED,
  and intentionally keeps release-gate probe outputs under `gate-runs/` inside
  the audit report.
- P00 baseline at `reports/surface-ui-production-audit/p00-baseline/` found
  current live blockers without promoting them: wasm32-web release browser
  smoke timed out as `BLOCKED` (exit `124`), and safe-view lifetime gate failed
  because docs manifest verification reported missing doc reference
  `docs/audits/memory-ideal-vslice-v0-baseline.md`.
- `SURFPROD-P01` adds `scripts/release/surface/report-dir-guard.sh` and
  requires `release-gate.sh` to use a fresh repo-relative report dir. It rejects
  empty, absolute, repo-root, parent traversal, symlink, non-directory, and
  non-empty report dirs before any Surface sub-gate runs.
- `release-gate.sh` now writes same-run metadata fields into
  `surface-release-summary.json` once sub-gates complete: producer, git head,
  dirty state, module version, host OS/arch, generated UTC time, and command
  line. Full release-gate success is still not claimed until later packets close
  runtime/browser blockers.
- `SURFPROD-P02` extends release summary validation to reject missing/copy-stale
  producer metadata. `ValidateReleaseSummary` now requires producer
  `scripts/release/surface/release-gate.sh`, a 40-character hex `git_head`,
  non-empty version/host/generated metadata, and a command line naming
  `scripts/release/surface/release-gate.sh`.
- `SURFPROD-P03` adds strict `validate-surface-runtime --release headless`
  support. It accepts only runtime schema reports with `target=headless`,
  `runtime=surface-headless`, deterministic headless software framebuffer
  evidence, and no real-window/native-input/platform-widget claim.
- P03 headless report evidence lives at
  `reports/surface-ui-production-p03/headless-release/`. The trusted artifacts
  are the component binary and runner trace under that report tree plus
  `artifact-hashes.json`; this is not linux/browser release evidence.
- `SURFPROD-P04` adds a no-display preflight to
  `surface-linux-x64-release-window-smoke.sh`: when neither `WAYLAND_DISPLAY`
  nor `DISPLAY` is present, it writes a `status=blocked`,
  `production_claim=false` report and exits non-zero before any sub-gate work.
- `validate-surface-runtime --release linux-x64-real-window` now strictly
  accepts only `target=linux-x64`, `runtime=surface-linux-x64`,
  `linux-x64-release-window-v1`, `wayland-shm-rgba-release-v1`, and true
  framebuffer/real_window/native_input/text_input/clipboard/composition/
  accessibility_bridge host evidence.
- `SURFPROD-P05` root cause: Chromium on this host can render `data:` URLs but
  did not issue localhost HTTP runner requests, so the old browser-canvas smoke
  hung before trace extraction. `runBrowserCanvasTrace` now uses a temporary
  file-backed runner with inline compiler-owned host JS and
  `--allow-file-access-from-files`; this avoids localhost and avoids data-URL
  argv overflow for release-size wasm.
- `surface-wasm32-web-release-browser-smoke.sh` now has hard timeout/cleanup
  and writes a controlled blocked report on smoke failure, but the canonical
  P05 evidence passed at `reports/surface-ui-production-p05/browser/`.
  `validate-surface-runtime --release wasm32-web-browser` strictly checks
  wasm32-web browser-canvas release host evidence and rejects Node-only browser
  substitution.
- `SURFPROD-P06` live RED was not the historical `insert_bytes` proof mismatch:
  the build test and headless text-input smoke already passed on this branch.
  The actual blocker was missing strict
  `validate-surface-runtime --release text-input` routing. The fix added that
  selector and a compiler regression for the `insert_bytes` loop shape so the
  source `bytes[i]` load remains proof-tagged unchecked while the destination
  `buf[idx + i]` store remains checked.
- `SURFPROD-P07` found an actual validator gap after the plan RED passed:
  production toolkit reports could still claim `example_count=1`. Production
  `production-widgets-v1` now requires the three scoped toolkit examples
  `examples/surface_release_form.tetra`, `examples/surface_toolkit_form.tetra`,
  and `examples/surface_toolkit_settings.tetra`; this hardens multi-example
  evidence without claiming platform-native widgets or a full UI framework.
- `SURFPROD-P08` separates accessibility evidence tiers: headless release
  accessibility may only name `headless_platform_tree_probe`; linux release
  accessibility requires `host_evidence.accessibility_bridge=true`, a linux
  platform-probe process, and a `linux-accessibility-platform-probe` artifact;
  wasm release accessibility requires browser snapshot/mirror host flags plus
  runner-trace `browser_accessibility` payload. The browser trace root cause was
  `scripts/tools/surface_browser_canvas_host.mjs` leaving the
  `release-accessibility` trace payload empty; it now marks only the
  compiler-owned accessibility mirror, not clipboard/composition/full
  screen-reader support.
- `SURFPROD-P09` closes the safe-view lifetime blocker in Surface scope by
  making `scripts/release/safe-view-lifetime/gate.sh` bounded, per-step logged,
  release-blocking, and focused on Surface lifetime/resource cleanup instead of
  broad `./compiler/...`, `./cli/...`, `./tools/...`, or docs manifest checks.
  The P00 docs manifest failure is now routed to `SURFPROD-P10`, not hidden in
  the safe-view gate.
- `surface-linux-x64-release-window-smoke.sh` now has a timeout/cleanup wrapper
  for live host commands. Its final artifact-hash validation is bounded but
  unlogged so it does not create a new report artifact after
  `artifact-hashes.json` has been sealed.
- `TestReleaseSurfaceSmokeScriptsUseStrictReleaseValidation` now requires a
  strict `--release` selector without forcing every release script back to the
  broad `surface-v1` selector; packet-specific selectors such as `text-input`
  remain valid strict release evidence.
- `SURFPROD-P10` closes the docs/manifest consistency blocker that P00 exposed:
  `verify-docs` rejects `/tmp` paths as current Surface release evidence,
  `validate-manifest` requires current and unsupported Surface feature rows, and
  `api-stability-gate.sh` writes `public-surface-api-summary.txt` next to the
  JSON summary. Generated manifest now includes `core.island_reset`; the exact
  `git diff --exit-code -- docs/generated/manifest.json` check reports that
  tracked generated-file update, while fresh temp generation matches the working
  manifest byte-for-byte.
- `SURFPROD-P11` makes package publishing Surface-aware: `release-packages.yml`
  now runs Surface release, experimental regression, safe-view lifetime, and API
  stability gates before package artifact upload, GitHub release, container
  publishing, or Homebrew tap update. Surface report dirs are uploaded with the
  package artifacts, and workflow tests reject `continue-on-error: true` on
  release gates.
- `SURFPROD-P12` keeps docs honest by rejecting GPU, platform/native widget,
  rich-text, screen-reader, DOM/React/user-JS, cross-platform, and unsupported
  target Surface promotion claims unless they are explicitly written as
  nonclaims. The Surface guide now includes release troubleshooting for display
  host variables, browser availability, blocked reports, fresh repo-local report
  dirs, and starter-vs-release evidence.
- `SURFPROD-P13` collected final candidate evidence in the live repo: final
  Surface release, experimental regression, safe-view lifetime, API stability,
  broad tests, race Surface/UI selector, CI script, docs/manifest/hash
  validators, `git diff --check`, manifest idempotence, and Graphify update all
  pass. The remaining blocker is the exact
  `git diff --exit-code -- docs/generated/manifest.json` check: the working
  manifest intentionally includes generated `core.island_reset`, and a fresh
  final generated copy matches it exactly. Without a commit or an explicit
  acceptance of this generated-file cleanliness caveat, the goal should remain
  active rather than complete.
