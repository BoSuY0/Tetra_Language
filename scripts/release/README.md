# scripts/release

Release scripts are versioned by directory.

Examples:

- `v1_0/gate.sh`
- `v1_0/api-diff.sh`
- `v1_0/binary-size.sh`
- `v1_0/reproducible-build.sh`
- `v1_0/security-review.sh`
- `v1_0/wasi-smoke.sh`
- `v1_0/web-smoke.sh`

Release entrypoints must live under a versioned release directory or a focused
shared directory such as `shared/` or `smoke/`. Do not add root-level release
compatibility wrappers.

`scripts/release/v1_0/api-diff.sh` is the canonical API diff workflow.

`scripts/release/v1_0/binary-size.sh` is the canonical binary-size evidence
workflow.

`scripts/release/v1_0/reproducible-build.sh` is the canonical reproducible
build proof workflow.
