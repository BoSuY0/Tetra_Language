# Tetra Surface/UI Production Plan Tracker

External plan: `/home/tetra/Downloads/surface-ui-production-implementation-plan.md`

## Current Strategy

1. Treat the external plan as the scope source, but recheck every observation
   against this live repo with `.git`.
2. Reset top-level goal-loop state from the old Memory/IslandKernel contract to
   `SURFPROD-P00..P13`.
3. Start with `SURFPROD-P00` truth audit script because current evidence is
   mixed and target-host blockers must be recorded honestly before code/gate
   fixes.
4. Work packet-by-packet with RED/GREEN evidence; do not promote starter or
   blocked evidence to production release evidence.

## Packet Matrix

| Packet | Status | Acceptance Evidence |
| --- | --- | --- |
| `SURFPROD-P00` truth audit script and baseline report | done_narrow | `bash -n scripts/analysis/surface-ui-truth-audit.sh`; `go test -buildvcs=false ./tools/scriptstest -run 'SurfaceUITruthAudit|AnalysisScript' -count=1`; baseline `reports/surface-ui-production-audit/p00-baseline/truth-summary.md` |
| `SURFPROD-P01` Surface release gate hardening | done_narrow | report-dir guard rejects stale/symlink/traversal/absolute/root/non-directory before sub-gates; release summary metadata contract covered by `SurfaceReleaseGate` script tests |
| `SURFPROD-P02` validator fake-evidence rejection | done_narrow | release summary now requires producer/git/version/host/generated/command metadata; stale/copy negative fixtures reject missing producer, stale git head, and missing command line |
| `SURFPROD-P03` headless runtime evidence hardening | done_narrow | headless release smoke under `reports/surface-ui-production-p03/headless-release`, `--release headless` validation, artifact hash manifest validation |
| `SURFPROD-P04` linux-x64 real-window/input lifecycle | done_narrow | no-display fake-root blocked report test; linux release-window smoke under `reports/surface-ui-production-p04/linux-window`; `--release linux-x64-real-window` validation |
| `SURFPROD-P05` wasm32-web browser-canvas/input | done_narrow | bounded browser smoke under `reports/surface-ui-production-p05/browser`; file-backed Chromium runner avoids localhost hang/argv overflow; `--release wasm32-web-browser`, wasm imports, and artifact hashes validate |
| `SURFPROD-P06` text/input/clipboard/IME examples | done_narrow | text release example builds/runs; strict `--release text-input` validation; `insert_bytes` source loop proof regression guards proof validation |
| `SURFPROD-P07` component tree/layout/toolkit | done_narrow | toolkit/component runtime report under `reports/surface-ui-production-p07/headless-toolkit`; validator rejects single-example production toolkit claim; scoped production-minimal widget comment |
| `SURFPROD-P08` accessibility metadata/bridge scope | done_narrow | headless/linux/wasm accessibility reports under `reports/surface-ui-production-p08/`; validator/CLI reject target-specific accessibility overclaims without platform probe or browser mirror artifacts |
| `SURFPROD-P09` safe-view lifetime/resource cleanup | done_narrow | bounded safe-view gate under `reports/surface-ui-production-p09/safe-view-lifetime`; linux cleanup report under `reports/surface-ui-production-p09/linux-window-cleanup`; race Surface/SafeView regex passed |
| `SURFPROD-P10` API stability/docs generation | done_narrow | API stability gate under `reports/surface-ui-production-p10/surface-api-stability-v1`; docs/manifest validators pass; manifest generation is idempotent and records `core.island_reset` |
| `SURFPROD-P11` CI release-readiness integration | done_narrow | CI readiness and package publishing workflows run Surface gates without `continue-on-error`; package release uploads Surface reports before publish |
| `SURFPROD-P12` docs/nonclaims/user guide correction | done_narrow | docs validator rejects Surface overclaims; user guide documents target scope, troubleshooting, starter-vs-release evidence, and release-supported examples |
| `SURFPROD-P13` final production release candidate gate | blocked_on_manifest_cleanliness | release, experimental, safe-view, API, broad tests, race gate, CI, docs/manifest/hash validators, `git diff --check`, manifest idempotence, final summary, and Graphify update pass; exact `git diff --exit-code -- docs/generated/manifest.json` remains non-zero for the intended `core.island_reset` generated-manifest update |

## Current Iteration

1. `SURFPROD-P00` done as a narrow truth-audit packet. The baseline report
   honestly records `production_ready_claim: false`, `12 PASS`, `1 BLOCKED`,
   and `1 FAIL` at git head `3e489e567edc6ab7e537594313a9719a473aea38`.
2. `SURFPROD-P01` done as a narrow release-gate hardening slice: invalid
   report dirs now fail before sub-gates, and the release summary path records
   same-run metadata fields. A full fresh release-gate pass is still deferred
   until later runtime/browser packets close target blockers.
3. `SURFPROD-P02` done as a narrow validator hardening slice: release summaries
   now reject copied/stale producer metadata and valid fixtures include same-run
   metadata. Existing fake claim fixtures still pass the full validator package
   sweep.
4. `SURFPROD-P03` done as a narrow headless release evidence slice: the
   headless release smoke writes built binary and runner-trace artifacts under
   `reports/surface-ui-production-p03/headless-release`, validates with
   `--release headless`, and validates artifact hashes.
5. `SURFPROD-P04` done as a narrow linux release-window slice on this host:
   no-display fake-root execution writes a blocked report, and the live host
   produced validated `linux-x64-release-window-v1` evidence under
   `reports/surface-ui-production-p04/linux-window`.
6. `SURFPROD-P05` done as a narrow wasm32-web browser release slice: the
   browser smoke has timeout/cleanup, uses a file-backed runner instead of a
   localhost mini-server, rejects Node-only release substitution, validates wasm
   imports, and writes hash-validated browser-canvas release evidence under
   `reports/surface-ui-production-p05/browser`.
7. `SURFPROD-P06` done as a narrow text/input/clipboard/IME slice: the
   release text-input example builds, the canonical headless text-input report
   validates with `--release text-input`, validators still require UTF-8,
   clipboard owned-copy, composition, and safe-view evidence, and
   `compiler/internal/lower` now has an `insert_bytes` loop-shape regression
   for the proof-tagged `bytes[i]` source load.
8. `SURFPROD-P07` done as a narrow component tree/toolkit slice: production
   toolkit evidence is canonical under
   `reports/surface-ui-production-p07/headless-toolkit`, validator coverage now
   rejects single-example production toolkit claims, and `lib/core/widgets.tetra`
   is labelled as production-minimal scoped rather than broad experimental UI
   framework parity.
9. `SURFPROD-P08` done as a narrow accessibility scope slice: headless
   accessibility keeps only `headless_platform_tree_probe`, linux accessibility
   requires `accessibility_bridge=true` plus `linux-accessibility-platform-probe`
   artifact/process evidence, and wasm accessibility requires browser
   snapshot/mirror host flags plus runner-trace payload evidence. Full
   screen-reader/AT-SPI support remains a non-goal.
10. `SURFPROD-P09` done as a narrow safe-view/resource cleanup slice:
    `scripts/release/safe-view-lifetime/gate.sh` now runs bounded per-step
    Surface lifetime/resource cleanup checks with logs and summary markers, the
    linux release-window script has timeout/cleanup process guards, browser and
    linux cleanup contracts are covered by script tests, and live evidence
    exists under `reports/surface-ui-production-p09/`.
11. `SURFPROD-P10` done as a narrow API stability/docs generation slice:
    Surface docs now reject `/tmp` current release evidence, generated manifest
    validation requires current/unsupported Surface feature rows, API stability
    gate writes a public Surface API summary, `docs/spec/unsafe.md` lists
    `core.island_reset`, and generated manifest output is idempotent. The exact
    `git diff --exit-code -- docs/generated/manifest.json` command remains
    non-zero only because this packet updates the tracked generated manifest
    with `core.island_reset`; a fresh temp generation diff is clean.
12. `SURFPROD-P11` done as a narrow CI/package release-readiness slice:
    exact-name workflow tests now cover Surface readiness, package Surface gates,
    and no `continue-on-error`; `release-packages.yml` runs Surface release,
    experimental regression, safe-view lifetime, and API stability gates before
    any package upload/GitHub release/container/Homebrew publishing path and
    uploads those report dirs.
13. `SURFPROD-P12` done as a narrow docs/nonclaims slice: `verify-docs`
    rejects GPU/native-widget/rich-text/screen-reader/DOM/React/user-JS/
    cross-platform Surface overclaims, Surface docs use line-local nonclaim
    wording, `surface_guide.md` documents display/browser blocked-report
    troubleshooting and starter-vs-release evidence, and examples index includes
    `surface_release_counter.tetra` with the release-supported examples.
14. `SURFPROD-P13` final candidate evidence is collected but not fully
    complete: focused P13 regressions for formal-core typed proof terms,
    optimizer closure artifact, and shell safety headers are fixed; final
    Surface release, experimental regression, safe-view lifetime, API
    stability, focused validator packages, broad compiler/cli/tools tests,
    focused race regex, CI script, docs/manifest/hash validators,
    `git diff --check`, manifest idempotence, final summary, and Graphify
    update all pass. The remaining blocker is the exact
    `git diff --exit-code -- docs/generated/manifest.json` final check, which
    is non-zero only because the tracked generated manifest intentionally adds
    `core.island_reset`.

## Open Decisions

- Linux real-window production evidence may require target host infrastructure.
  Browser-canvas release evidence is now proven on this host through Chromium
  file-runner evidence rather than localhost mini-server evidence.
- The P00 safe-view blocker is closed for Surface P09 scope by making the
  safe-view lifetime gate bounded and Surface-focused instead of coupling it to
  unrelated docs manifest verification. P10 closed the docs/manifest consistency
  gap with validator coverage and generated manifest idempotence evidence.
- `SURFPROD-P13` cannot be marked complete without resolving how the generated
  manifest cleanliness check should be handled in an uncommitted worktree. The
  current manifest is generator-idempotent, but it differs from `HEAD` by the
  planned `core.island_reset` row.
