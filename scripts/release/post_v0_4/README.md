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

Release evidence directories must be fresh. Memory production gates refuse
symlink, non-directory, or non-empty `--report-dir` values before running
smokes, so stale JSON or hash manifests cannot be reused as same-run evidence.
The Linux-x64 memory production gate also writes `targets.json` with
`go run ./cli/cmd/tetra targets --format=json`, validates it with
`go run ./tools/cmd/validate-targets --report`, and includes that target
capability report in the same `artifact-hashes.json` manifest. It also writes
a Tier 1 deterministic memory fuzz oracle bundle under `memory-fuzz-tier1/`
with `go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir`, validates
`memory-fuzz-oracle.json`, `summary.md`, `summary.json`, and command
provenance with `go run ./tools/cmd/validate-memory-fuzz-oracle --artifact-dir`,
and includes those fuzz artifacts in the same hash manifest. Tier 2 nightly
seed triage and Tier 3 release-blocking focused fuzz remain scheduled/release
policy boundaries, not exhaustive fuzz proof.
Quick evidence is useful for local iteration only; it is not full,
stabilization, nightly, or release proof unless the corresponding full gate and
validators ran for that artifact set.

Keep post-v0.4 production evidence behavior here. Do not add root-level
compatibility wrappers under `scripts/`.
