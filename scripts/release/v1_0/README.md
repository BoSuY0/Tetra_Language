# scripts/release/v1_0

`gate.sh` is the v1.0 release gate entrypoint.

This directory also owns the v1.0 release evidence helpers:

- `api-diff.sh`
- `binary-size.sh`
- `reproducible-build.sh`
- `security-review.sh`
- `wasi-smoke.sh`
- `web-smoke.sh`

Keep v1.0-specific release behavior in this directory. Do not add root-level
compatibility wrappers under `scripts/`.
