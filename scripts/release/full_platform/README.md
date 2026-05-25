# Full-Platform UI Runtime Gate

This directory contains the post-v0.4 full-platform UI runtime promotion gate.

The stable platform runtime evidence contract is `tetra.ui.platform.v1`.
Linux-x64 keeps its existing `tetra.ui.desktop-runtime.v1` production report,
while Windows and macOS must provide target-host `tetra.ui.platform.v1`
reports collected on real Windows/macOS runners. Web evidence remains the
browser-backed `tetra.web-ui-smoke.v1alpha1` report with real WASM
instantiation, DOM mount, UI metadata load, event dispatch, state/render
changes, and runtime trace markers.

`windows-ui-runtime-smoke.sh` and `macos-ui-runtime-smoke.sh` accept
`--evidence <path>` for reports produced by target-host runners. Without that
evidence on a non-target host, they write explicit blocked reports and fail.
Target-host reports must include a fresh RFC3339 `generated_at` timestamp.
Blocked, stale, build-only, metadata-only, runtime-less, sidecar-only,
fake/mock/placeholder, docs-only, or `startup_failure` reports do not count as
production runtime evidence.

Run the full gate with:

```sh
bash scripts/release/full_platform/ui-runtime-gate.sh --report-dir reports/full-platform-ui-runtime
```
