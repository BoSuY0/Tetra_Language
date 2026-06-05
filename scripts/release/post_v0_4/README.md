# scripts/release/post_v0_4

Post-v0.4 production release-gate entrypoints for Linux-x64 Memory,
Parallelism, Compiler, UI, and Linux native target-family evidence.

This directory owns the ordered production gates:

- `memory-production-linux-x64-smoke.sh`
- `parallel-production-linux-x64-smoke.sh`
- `compiler-production-linux-x64-smoke.sh`
- `memory-parallel-compiler-production-linux-x64-gate.sh`
- `ui-production-runtime-linux-x64-smoke.sh`
- `memory-parallel-ui-production-linux-x64-gate.sh`
- `linux-native-targets-smoke.sh`
- `linux-x86-smoke.sh`
- `linux-x32-smoke.sh`

The Linux native target-family scripts write ABI, atomic, fuzz, runner, brutal,
and artifact-hash reports as applicable. They validate `artifact-hashes.json`
before passing it to `tools/cmd/validate-linux-native-targets`, so the Linux
native gate covers both suite contents and same-run artifact integrity. Runner
evidence must be either a real passing runtime runner test report for that
target or an explicit no-host-fallback JSON diagnostic for linux-x86/linux-x32
on hosts that cannot execute those ABIs. Passing runner reports include
`runner arithmetic`, `runner alloc memory`, `runner filesystem`,
`runner stderr fd`, `runner time`, `runner network socket`, and
`runner network options`, and `runner task join`. Blocked runner
diagnostics include the target, host identity, and exact probe command. The validator also checks
that passing runner reports line up with `run_supported: true` in `targets.json`
and no-host diagnostics line up with `run_supported: false`. Passing per-target ABI,
atomic, fuzz, and runner reports include top-level `target` identity, and
`validate-linux-native-targets` rejects wrong-target evidence.

The Linux native scripts default to `go run ./cli/cmd/tetra` so evidence follows
the current source tree instead of a possibly stale local `./tetra` binary. Set
`TETRA_CMD` explicitly when a release job must use a prebuilt binary. They also
default `GOCACHE` to repo-local `.cache/go-build-*` paths to avoid tmpfs-backed
cache pressure during repeated ABI/validator runs.

Keep post-v0.4 production evidence behavior here. Do not add root-level
compatibility wrappers under `scripts/`.
