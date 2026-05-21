# WASM And UI Guide

Status: user guide for WASM/UI boundaries. The current support boundary is
`docs/spec/current_supported_surface.md`; WASM release evidence is split into
artifact/import preflight and runner-backed runtime proof. `wasm32-wasi`
runtime support is conditional on a discoverable WASI runner, `wasm32-web`
runtime proof comes from the browser runner smoke trace, and Linux-x64 native
UI runtime proof comes from `tetra.ui.native-runtime.v1` smoke evidence.

## WASM

The target plan is documented in `docs/backend/wasm_backend_plan.md` and
`docs/backend/wasm_architecture.md`.

Required v1.0 targets:

- `wasm32-wasi`
- `wasm32-web`

Required release checks:

```sh
./tetra smoke --target wasm32-wasi --run=false --report /tmp/tetra-wasi-artifact.json
go run ./tools/cmd/validate-wasm-imports --target wasm32-wasi --report /tmp/tetra-wasi-artifact.json
./tetra smoke --target wasm32-wasi --run=true --report /tmp/tetra-wasi-runtime.json
./tetra smoke --target wasm32-web --run=false --report /tmp/tetra-web-artifact.json
go run ./tools/cmd/validate-wasm-imports --target wasm32-web --report /tmp/tetra-web-artifact.json
./tetra smoke --target wasm32-web --run=true --report /tmp/tetra-web-runtime.json
bash scripts/release/v1_0/wasi-smoke.sh --report /tmp/tetra-wasi-smoke.json
bash scripts/release/v1_0/web-smoke.sh --report /tmp/tetra-web-smoke.json
go run ./tools/cmd/validate-web-ui-smoke --report /tmp/tetra-web-smoke.json
bash scripts/release/post_v0_4/wasm-ui-gui-production-gate.sh --report-dir reports/wasm-ui-gui
```

## UI

UI syntax, web backend behavior, native shell behavior, and accessibility
metadata remain bounded by their dedicated smoke reports. A fallback browser
report is useful evidence for tooling, but it does not replace UI-specific
smoke coverage. Missing or crashing headless browser automation writes a
validated `blocked` report and remains a release blocker for web UI evidence.

For a `pass` web UI smoke report, validator-enforced evidence now includes:

- `ui_schema: "tetra.ui.v1"`
- `ui_bundle_path` ending in `.ui.json`
- `ui_module_path` ending in `.ui.web.mjs`
- `dom_snapshot` ending in `.html`
- `runtime_trace` containing DOM mount/layout/widget/event markers plus
  `ui-event-dispatch:web-command-dispatch`

Support boundary for the bounded post-v0.4 promotion:

- Web UI validates metadata, instantiates real WASM through a
  Chromium-compatible browser runner, mounts a DOM runtime shell, and dispatches
  supported events to lowered scalar state command operations.
- `event click -> increment`/`decrement` style handlers are validated,
  rendered, and dispatched when the generated command operations describe
  supported direct assignment or integer `+/-` state updates, including
  supported `+=` and `-=` compound assignments. String, boolean, and
  integer-like assignments are hydrated as runtime scalar values instead of
  displayed as raw source literals, and same-state field assignments copy the
  current source field value in command order. Supported style and
  accessibility metadata is mirrored into preview DOM attributes such as
  `data-tetra-style-*`, role, and aria-label; this is not a full layout engine
  or platform accessibility API integration.
- WASI dogfood (`examples/projects/dogfood_wasi/src/main.tetra`) remains non-UI
  and should not emit UI runtime sidecars.

Linux-x64 native UI runtime support is separate from both web UI and native
shell metadata. The native runtime smoke builds the current CLI, builds
`examples/ui_native_shell_smoke.tetra` for `linux-x64`, runs the native
executable, loads the generated native shell sidecar into the runtime smoke,
dispatches click events through lowered command operations, records before/after
state and widget updates, covers invalid widget/malformed metadata/unsupported
event/command failure negatives, closes the runtime, and writes
`native-ui-linux-x64.json` under the selected report directory.

Do not use `tetra.ui.v1` metadata, wasm/web UI reports, or
`tetra.ui.native-shell.v1` sidecars alone as native runtime proof. macOS and
Windows native UI runtime support still need their own host-native reports.

## Plan250 Smoke Snapshot

The Wave-A docs closure records these concrete artifacts from commit `b884653`.
They document observed behavior for this repository state; a release candidate
still needs fresh reports from its own report directory.

| Field | Evidence |
| --- | --- |
| Web UI report | `reports/plan250/backend/web-ui-smoke.json` |
| Web UI source | `examples/projects/dogfood_web_ui/src/main.tetra` |
| Web UI automation | `chromium --headless --dump-dom` |
| Web UI status | `status: pass`, `result: ok:0:ui=1`, `ui_scope_active: true` |
| Web UI generated files | `reports/plan250/backend/web-ui-smoke.ui.json`, `reports/plan250/backend/web-ui-smoke.ui.web.mjs`, `reports/plan250/backend/web-ui-smoke.dom.html` |
| WASI runner report | `reports/plan250/backend/wasi-smoke.json` |
| WASI runner behavior | `runner: node-wasi`, `total: 5`, `passed: 5`, `failed: 0` |
| WASM artifact/import reports | `reports/plan250/backend/wasm32-wasi-artifact-smoke.json`, `reports/plan250/backend/wasm32-web-artifact-smoke.json` |

Do not use the artifact/import reports as browser or WASI runtime proof. The web
runtime proof is the dedicated web smoke report; the WASI runtime proof is the
dedicated WASI smoke report.
