# WASM Backend Plan

Status: current for v0.4.0 runner-backed WASM execution plus the separately
gated post-v0.4 WASM/UI/GUI production promotion path; broader runtime parity
work remains tracked by the v1 release gate.

This document records the WASM implementation contract. The current compiler
supports `wasm32-wasi` through `wasmtime` or the Node WASI fallback and supports
`wasm32-web` through a discovered Chromium-compatible browser runner. Both
targets still keep explicit missing-runner diagnostics. Browser UI event
dispatch is production evidence only when the dedicated Web UI smoke report
loads real WASM, mounts DOM from `tetra.ui.v1`, dispatches events, and passes
`go run ./tools/cmd/validate-web-ui-smoke`.

v0.4.0 checkpoint (current behavior in this repository):

- `wasm32-wasi` and `wasm32-web` are supported runtime targets with
  deterministic module/linker checks.
- `wasm32-wasi` target metadata uses `run_mode: "wasi_runner"` and reports
  `run_supported` according to runner discovery.
- `wasm32-web` target metadata uses `run_mode: "web_runner"` and reports
  `run_supported` according to Chromium-compatible browser discovery.
- Phase 1 minimal backend parity is in place for control-flow IR (`IRLabel`,
  `IRJmp`, `IRJmpIfZero`) and array/slice IR used by the Array MVP
  (`IRMakeSliceI32/U8/U16`, `IRIndexLoadI32/U8/U16`, `IRIndexStoreI32/U8/U16`).
- Unsupported IR remains explicit and fails with stable backend diagnostics
  instead of silent behavior changes.
- UI output is metadata-first (`tetra.ui.v1`) with preview artifacts.
- Web UI production smoke validates metadata, instantiates real WASM in a
  Chromium-compatible browser runner, mounts DOM, dispatches lowered scalar
  state command operations for supported events, verifies state/render changes,
  and records the validator-required production runtime trace markers.
- WASI dogfood remains non-UI for this wave and must not emit web/native UI sidecars.

Exact object/runtime/package/host-binding decisions are fixed in [WASM Object and Runtime Architecture](wasm_architecture.md) and should be treated as the prerequisite contract for target metadata and backend implementation changes.

## Targets

- `wasm32-wasi`: command-line and server-side runtime target. This target must produce a WASI-compatible module, pass artifact/import preflight, and pass runner smoke.
- `wasm32-web`: browser runtime target. This target must produce a web-loadable module plus the minimal JS/runtime glue needed by UI smoke tests.

## Shared Backend Shape

The native backend already has an hourglass split for x64 codegen, ABI rules, object building, and executable linking. WASM should use the same boundary in spirit, but not reuse x64 object/link layers:

- frontend, semantics, lowering, diagnostics, effects, ownership, and dependency analysis stay shared;
- IR-to-WASM emission is new target-specific code;
- module assembly replaces TOBJ plus ELF/Mach-O/PE linking for these targets;
- runtime imports are explicit and target-specific;
- smoke reports use the same JSON report shape as native target smoke.

The first supported WASM value surface should be deliberately small: `i32`, bool-like conditions, calls, returns, locals, string data where already lowered, slices only after the runtime layout is specified, and no implicit host access outside effect-gated imports.

## Phase 0: Target contract

Goal: replace planned-target diagnostics with a real target descriptor only when the backend has a minimal module writer.

Required work:

- Extend target metadata so `wasm32-wasi` and `wasm32-web` have explicit OS, arch, ABI/runtime kind, artifact extension, and import policy.
- Keep unsupported-feature diagnostics precise while each WASM slice is incomplete.
- Add target-list tests that distinguish supported WASM from unknown targets.
- Keep `tetra targets --format=json` and `go run ./tools/cmd/validate-targets` in the verification path.

Done when:

- `go run ./tools/cmd/validate-targets` accepts the updated target JSON.
- Unknown targets still fail as unsupported, not planned.
- WASM targets no longer become supported by metadata alone; the backend smoke must be real.

## Phase 1: WASM IR emitter

Goal: lower the existing compiler IR into deterministic WebAssembly modules.

Required work:

- Add an IR-to-WASM emitter with stable function ordering, local allocation, labels, branches, calls, returns, integer arithmetic, comparisons, and deterministic data segments.
- Add a module writer for type, function, export, code, memory, data, and name/custom sections as needed.
- Define the initial runtime import namespace separately for `wasm32-wasi` and `wasm32-web`.
- Add golden or structural tests that reject nondeterministic module output.

Done when:

- A minimal `examples/flow_hello.tetra` style program builds to a valid `.wasm` artifact.
- Deterministic build checks pass twice for the same input.
- Artifact/import smoke can write JSON reports for both WASM targets.

Gate commands:

```sh
./tetra smoke --target wasm32-wasi --run=false
./tetra smoke --target wasm32-web --run=false
```

## Phase 2: WASI runner

Goal: make `wasm32-wasi` executable in the v1.0 gate, not only artifact-buildable.

Required work:

- Select and document the runner command, currently expected to be `wasmtime`.
- Map process exit, stdout, stderr, memory, and any allowed filesystem access through explicit WASI imports.
- Ensure effects and capabilities remain visible in diagnostics and docs before host access is allowed.
- Add runner availability diagnostics so missing `wasmtime` is reported as an environment skip or hard release-gate failure according to the gate mode.

Done when:

- `./tetra smoke --target wasm32-wasi --run=true` executes smoke programs through `wasmtime`.
- Smoke JSON distinguishes artifact build failure, runner failure, and missing runner.
- `bash scripts/release/v1_0/gate.sh` runs the WASI smoke path as a mandatory v1.0 check.

## Phase 3: Web runtime

Goal: make `wasm32-web` usable by browser and UI smoke tests.

Required work:

- Define the browser import namespace for console/output, memory setup, event loop entry, and any UI runtime calls.
- Produce or locate the JS glue needed to instantiate the module deterministically.
- Add a tiny web runner page that can load the compiled module.
- Validate the smoke page through browser automation.

Done when:

- `./tetra smoke --target wasm32-web --run=false` proves the module and glue are generated.
- A browser automation smoke test loads the module and observes a deterministic result.
- UI web release checks depend on the WASM web backend instead of a placeholder.

## Phase 4: v1.0 release gate

Goal: make the final v1.0 gate fail for real implementation gaps and pass only with supported WASM.

Required work:

- Keep artifact/import preflight smoke for both WASM targets. This is not the
  runtime production claim; it exists to validate artifact shape and imports
  before the mandatory runner-backed checks:

```sh
./tetra smoke --target wasm32-wasi --run=false
./tetra smoke --target wasm32-web --run=false
```

- Add mandatory WASI runner smoke:

```sh
./tetra smoke --target wasm32-wasi --run=true
```

- Add mandatory web smoke through browser automation after `wasm32-web` module generation exists.
- Keep docs and target metadata validation in the same gate:

```sh
go run ./tools/cmd/validate-targets
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
bash scripts/release/v1_0/gate.sh
```

For the bounded post-v0.4 promotion wave, run the dedicated gate without
rewriting v0.4.0 release truth:

```sh
bash scripts/release/post_v0_4/wasm-ui-gui-production-gate.sh --report-dir reports/wasm-ui-gui
```

The v1.0 release gate must not be changed to skip WASM. It remains the full
future `v1.0.0` release gate and may still fail on version/scope preflights
while separately gated post-v0.4 evidence is being collected.

## Remaining Limits For Full Runtime Parity

- Runtime parity beyond current scalar/control-flow smoke scope (for example
  full task/actor execution behavior on wasm targets) is still out of this
  phase and remains explicitly unsupported by diagnostics.
- API diff and reproducible-build checks still need stable WASM artifact
  baselines before they can certify final v1.0 parity.
