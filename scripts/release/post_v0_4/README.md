# scripts/release/post_v0_4

Post-v0.4 production release-gate entrypoints for Linux-x64 Memory,
Parallelism, Compiler, and UI evidence.

This directory owns the ordered production gates:

- `memory-production-linux-x64-smoke.sh`
- `parallel-production-linux-x64-smoke.sh`
- `compiler-production-linux-x64-smoke.sh`
- `memory-parallel-compiler-production-linux-x64-gate.sh`
- `ui-production-runtime-linux-x64-smoke.sh`
- `memory-parallel-ui-production-linux-x64-gate.sh`
- `wasm-ui-gui-production-gate.sh`
- `ui-toolkit-core-production-gate.sh`

`wasm-ui-gui-production-gate.sh` is the bounded post-v0.4 promotion gate for
WASI/Web runtime execution, browser-backed Web UI runtime evidence, and
Linux-x64 native UI/GUI runtime evidence. It emits
`tetra.release.post_v0_4.wasm_ui_gui.production-gate.v1` plus artifact hashes
under the selected report directory.

`ui-toolkit-core-production-gate.sh` is the bounded post-v0.4 promotion gate for
the platform-independent `tetra.ui.toolkit.v1` runtime core. It emits
`tetra.release.post_v0_4.ui_toolkit_core.production-gate.v1` plus artifact
hashes under the selected report directory, without claiming GTK/Qt/OS backend
production, Windows/macOS GUI, or full cross-platform UI.

Keep post-v0.4 production evidence behavior here. Do not add root-level
compatibility wrappers under `scripts/`.
