# WASM And UI Guide

Status: v1.0 user guide for planned release behavior. WASM and UI support must
not be called complete until the release gate has real smoke evidence.

## WASM

The target plan is documented in `docs/backend/wasm_backend_plan.md` and
`docs/backend/wasm_architecture.md`.

Required v1.0 targets:

- `wasm32-wasi`
- `wasm32-web`

Required release checks:

```sh
./tetra smoke --target wasm32-wasi --run=false --report /tmp/tetra-wasi-build.json
./tetra smoke --target wasm32-web --run=false --report /tmp/tetra-web-build.json
bash scripts/release_v1_0_wasi_smoke.sh --report /tmp/tetra-wasi-smoke.json
bash scripts/release_v1_0_web_smoke.sh --report /tmp/tetra-web-smoke.json
go run ./tools/cmd/validate-web-ui-smoke --report /tmp/tetra-web-smoke.json
```

## UI

UI syntax, web backend behavior, native shell behavior, and accessibility
metadata remain release blockers until their examples and smoke automation are
real. A fallback browser report is useful evidence for tooling, but it does not
replace UI-specific smoke coverage. Missing or crashing headless browser
automation writes a validated `blocked` report and remains a release blocker.

For a `pass` web UI smoke report, validator-enforced evidence now includes:

- `ui_schema: "tetra.ui.v1"`
- `ui_bundle_path` ending in `.ui.json`
- `ui_module_path` ending in `.ui.web.mjs`
- `dom_snapshot` ending in `.html`

Support boundary for v0.2.0:

- Web UI is a metadata preview shell and does not claim runtime event dispatch.
- WASI dogfood (`examples/projects/dogfood_wasi/src/main.tetra`) remains non-UI
  and should not emit UI runtime sidecars.
