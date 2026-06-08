# scripts/release/v0.4.0_0

Compatibility entrypoint for the post-v0.4 WASM/UI/GUI production promotion
gate requested by downstream automation.

`gate.sh` delegates to
`scripts/release/post_v0_4/wasm-ui-gui-production-gate.sh` and must not skip
WASI, Web, Web UI, native UI, desktop GUI, validator, or artifact-hash checks.
Keep production gate behavior in `scripts/release/post_v0_4`; this directory is
only a stable path alias.
