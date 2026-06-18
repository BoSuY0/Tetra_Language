# WASM And UI Guide

Status: user guide for WASM/UI boundaries. The current support boundary is
`docs/spec/core/current_supported_surface.md`; WASM release evidence is split into artifact/import
preflight and runner-backed runtime proof. `wasm32-wasi` runtime support is conditional on a
discoverable WASI runner, `wasm32-web` runtime proof comes from the browser runner smoke trace, and
Linux-x64 native UI runtime proof comes from `tetra.ui.native-runtime.v1` smoke evidence. The
post-v0.4 full-platform promotion contract is `tetra.ui.platform.v1`; Windows/macOS require real
target-host reports before production can be claimed.

## WASM

The target plan is documented in `docs/backend/wasm_backend_plan.md` and
`docs/backend/wasm_architecture.md`.

Required v1.0 targets:

- `wasm32-wasi`
- `wasm32-web`

Required release checks:

```sh
./tetra smoke --target wasm32-wasi --run=false --report /tmp/tetra-wasi-artifact.json
./tetra smoke --target wasm32-web --run=false --report /tmp/tetra-web-artifact.json
bash scripts/release/v1_0/wasi-smoke.sh --report /tmp/tetra-wasi-smoke.json
bash scripts/release/v1_0/web-smoke.sh --report /tmp/tetra-web-smoke.json
go run ./tools/cmd/validate-web-ui-smoke --report /tmp/tetra-web-smoke.json
bash scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh --report-dir reports/v0.4.0
go run ./tools/cmd/validate-native-ui-runtime --report reports/v0.4.0/native-ui-linux-x64.json
```

## UI

UI syntax, web backend behavior, native shell behavior, and accessibility metadata remain bounded by
their dedicated smoke reports. A fallback browser report is useful evidence for tooling, but it does
not replace UI-specific smoke coverage. Missing or crashing headless browser automation writes a
validated `blocked` report and remains a release blocker for web UI evidence.

For a `pass` web UI smoke report, validator-enforced evidence now includes:

- a fresh RFC3339 `generated_at` timestamp
- `ui_schema: "tetra.ui.v1"`
- `ui_bundle_path` ending in `.ui.json`
- `ui_module_path` ending in `.ui.web.mjs`
- `dom_snapshot` ending in `.html`
- `runtime_trace` containing `window-mount:ok`, `root-mount:ok`, layout, text, button, input, list,
  panel, focus, input-event, change, select, click, timer, async-command, redraw-update,
  error-recovery, and `ui-event-dispatch:web-command-dispatch` markers

Current support boundary:

- Web UI validates metadata, mounts a DOM preview shell with panel, text, button, input, and
  list/select controls, and dispatches supported events to lowered scalar state command operations.
- `event click -> increment`/`decrement` style handlers are validated, rendered, and dispatched when
  the generated command operations describe supported direct assignment or integer `+/-` state
  updates, including supported `+=` and `-=` compound assignments. String, boolean, and integer-like
  assignments are hydrated as runtime scalar values instead of displayed as raw source literals, and
  same-state field assignments copy the current source field value in command order. Supported style
  and accessibility metadata is mirrored into preview DOM attributes such as `data-tetra-style-*`,
  role, and aria-label; this is not a full layout engine or platform accessibility API integration.
- The production browser smoke exercises focus, input, change, select, click, timer, async,
  redraw/update, and error-recovery paths and records those markers in `runtime_trace`.
- WASI dogfood (`examples/projects/dogfood_wasi/src/main.tetra`) remains non-UI and should not emit
  UI runtime sidecars.

Linux-x64 native UI runtime support is separate from both web UI and native shell metadata. The
v0.4.0 native runtime smoke builds the current CLI, builds `examples/ui/ui_native_shell_smoke.tetra`
for `linux-x64`, runs the native executable, loads the generated native shell sidecar into the
runtime smoke, dispatches click events through lowered command operations, records before/after
state and widget updates, covers invalid widget/malformed metadata/unsupported event/command failure
negatives, closes the runtime, and writes `reports/v0.4.0/native-ui-linux-x64.json`.

Do not use `tetra.ui.v1` metadata, wasm/web UI reports, or `tetra.ui.native-shell.v1` sidecars alone
as native runtime proof. macOS and Windows native UI runtime support still need their own
host-native reports. The full-platform scripts under `scripts/release/full_platform/` accept
target-host evidence via `--evidence`; blocked reports document the missing runner path but do not
count as production runtime evidence.

The full-platform UI runtime gate is `scripts/release/full_platform/ui-runtime-gate.sh`. It writes
fresh evidence in one report directory and requires Linux, Windows, macOS, and Web reports plus
artifact hashes. On a Linux-only host, the Windows and macOS smoke wrappers write blocked
`tetra.ui.platform-runtime.v1` reports and fail; that is the expected production blocker until real
target-host runner evidence exists. CI fan-in can pass real runner reports back to the Linux
aggregation gate with `TETRA_WINDOWS_UI_RUNTIME_REPORT` and `TETRA_MACOS_UI_RUNTIME_REPORT`.

## Plan250 Smoke Snapshot

The Wave-A docs closure records these concrete artifacts from commit `b884653`. They document
observed behavior for this repository state; a release candidate still needs fresh reports from its
own report directory.

| Field                        | Evidence                                                                                                                                           |
| ---------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| Web UI report                | `reports/plan250/backend/web-ui-smoke.json`                                                                                                        |
| Web UI source                | `examples/projects/dogfood_web_ui/src/main.tetra`                                                                                                  |
| Web UI automation            | `chromium --headless --dump-dom`                                                                                                                   |
| Web UI status                | `status: pass`, `result: ok:0:ui=1`, `ui_scope_active: true`                                                                                       |
| Web UI generated files       | `reports/plan250/backend/web-ui-smoke.ui.json`, `reports/plan250/backend/web-ui-smoke.ui.web.mjs`, `reports/plan250/backend/web-ui-smoke.dom.html` |
| WASI runner report           | `reports/plan250/backend/wasi-smoke.json`                                                                                                          |
| WASI runner behavior         | `runner: node-wasi`, `total: 5`, `passed: 5`, `failed: 0`                                                                                          |
| WASM artifact/import reports | `reports/plan250/backend/wasm32-wasi-artifact-smoke.json`, `reports/plan250/backend/wasm32-web-artifact-smoke.json`                                |

Do not use the artifact/import reports as browser or WASI runtime proof. The web runtime proof is
the dedicated web smoke report; the WASI runtime proof is the dedicated WASI smoke report.
