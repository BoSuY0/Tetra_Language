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

Keep post-v0.4 production evidence behavior here. Do not add root-level
compatibility wrappers under `scripts/`.
