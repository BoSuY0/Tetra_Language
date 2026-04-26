# v1.0 Release Prep Artifacts (2026-04-26)

This directory contains generated artifacts used to document the v1.0 release
prep state for the current branch.

- `release_gate_summary.json` / `release_gate_summary.md`: output from
  `bash scripts/release_v1_0_gate.sh`.
- `test_all_full_summary.json` / `test_all_full_summary.md`: output from
  `bash scripts/test_all.sh --full --keep-going --report-dir /tmp/release-prep-full`.
- `api-diff/`: generated API docs candidate and API diff report.
- `wasi-smoke.json`: validated WASI smoke run report.
- `web-ui-smoke.json`: browser automation report (blocked without UI-specific
  smoke source, fallback evidence included).
- `reproducible-build.json`: native + WASM reproducibility proof.
