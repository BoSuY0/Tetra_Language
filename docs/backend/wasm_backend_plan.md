# WASM Backend Plan

Status: planned

This document turns the v1.0 WASM blocker into an implementation contract. It does not mark full WASM runtime parity as complete. The current compiler exposes `wasm32-wasi` and `wasm32-web` as build-only targets until the phases below are complete and the release gate runs the real smoke checks end-to-end.

v0.2.0 checkpoint (current behavior in this repository):

- `wasm32-wasi` and `wasm32-web` are build-only targets with deterministic module/linker checks.
- Phase 1 minimal backend parity is in place for control-flow IR (`IRLabel`,
  `IRJmp`, `IRJmpIfZero`) and array/slice IR used by the Array MVP
  (`IRMakeSliceI32/U8/U16`, `IRIndexLoadI32/U8/U16`, `IRIndexStoreI32/U8/U16`).
- Unsupported IR remains explicit and fails with stable backend diagnostics
  instead of silent behavior changes.
- UI output is metadata-first (`tetra.ui.v1`) with preview artifacts.
- Web preview validates and mounts metadata; it does not imply runtime UI event dispatch support.
- WASI dogfood remains non-UI for this wave and must not emit web/native UI sidecars.

Exact object/runtime/package/host-binding decisions are fixed in [WASM Object and Runtime Architecture](wasm_architecture.md) and should be treated as the prerequisite contract for target metadata and backend implementation changes.

## Targets

- `wasm32-wasi`: command-line and server-side runtime target. This target must produce a WASI-compatible module and pass both build-only smoke and runner smoke.
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
- Build-only smoke can write JSON reports for both WASM targets.

Gate commands:

```sh
./tetra smoke --target wasm32-wasi --run=false
./tetra smoke --target wasm32-web --run=false
```

## Phase 2: WASI runner

Goal: make `wasm32-wasi` executable in the v1.0 gate, not only buildable.

Required work:

- Select and document the runner command, currently expected to be `wasmtime`.
- Map process exit, stdout, stderr, memory, and any allowed filesystem access through explicit WASI imports.
- Ensure effects and capabilities remain visible in diagnostics and docs before host access is allowed.
- Add runner availability diagnostics so missing `wasmtime` is reported as an environment skip or hard release-gate failure according to the gate mode.

Done when:

- `./tetra smoke --target wasm32-wasi --run=true` executes smoke programs through `wasmtime`.
- Smoke JSON distinguishes build failure, runner failure, and missing runner.
- `bash scripts/release_v1_0_gate.sh` runs the WASI smoke path as a mandatory v1.0 check.

## Phase 3: Web runtime

Goal: make `wasm32-web` usable by browser and UI smoke tests.

Required work:

- Define the browser import namespace for console/output, memory setup, event loop entry, and any UI runtime calls.
- Produce or locate the JS glue needed to instantiate the module deterministically.
- Add a tiny browser smoke page that can load the compiled module.
- Validate the smoke page through browser automation.

Done when:

- `./tetra smoke --target wasm32-web --run=false` proves the module and glue are generated.
- A browser automation smoke test loads the module and observes a deterministic result.
- UI web release checks depend on the WASM web backend instead of a placeholder.

## Phase 4: v1.0 release gate

Goal: make the final v1.0 gate fail for real implementation gaps and pass only with supported WASM.

Required work:

- Keep build-only smoke for both WASM targets:

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
bash scripts/release_v1_0_gate.sh
```

The v1.0 release gate must not be changed to skip WASM. Until all commands above are real and green, the correct state is a failing `scripts/release_v1_0_gate.sh`.

## Remaining Blockers For Full Runtime Parity

- `wasm32-wasi` still needs mandatory `--run=true` runner integration in the
  main release gate.
- `wasm32-web` still needs browser-runtime smoke automation for execute-path
  evidence, not only build-only module checks.
- Runtime parity beyond current build-only scope (for example full task/actor
  execution behavior on wasm targets) is still out of this phase.
- API diff and reproducible-build checks still need stable WASM artifact
  baselines before they can certify final v1.0 parity.
