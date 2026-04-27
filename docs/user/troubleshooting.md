# Troubleshooting

Status: user guide for common v1.0 command failures.

Use the exact command that matches the failing workflow, then apply the fix
listed for the diagnostic you see.

## Invalid Target

Command:

```sh
./tetra build --target plan9-x64 examples/flow_hello.tetra
```

Typical diagnostic:

```text
unsupported target
```

Fix: choose one of the release targets from `./tetra targets`. Native build
targets are `linux-x64`, `macos-x64`, and `windows-x64`; WASM build-only
targets are `wasm32-wasi` and `wasm32-web`.

## Missing Import

Command:

```sh
./tetra check examples/flow_hello.tetra
```

Typical diagnostic:

```text
unknown import
```

Fix: check the module path and keep stable imports under `lib.core.*`. For
standard-library examples, start with `docs/user/standard_library_guide.md`.

## Formatter Mismatch

Command:

```sh
./tetra fmt --check examples lib __rt compiler/selfhostrt
```

Typical diagnostic:

```text
TETRA_FMT002
```

Fix: run `./tetra fmt --write <path>` on the reported file, review the diff,
then rerun the `--check` command.

## Unsupported Feature

Command:

```sh
./tetra check examples/typed_errors_smoke.tetra
```

Typical diagnostic:

```text
planned feature
```

Fix: confirm the feature is in `docs/spec/v1_scope.md`. Features listed under
post-v1 scope need the promotion checklist before implementation.

## Web Smoke Blocked

Command:

```sh
bash scripts/release_v1_0_web_smoke.sh --report reports/web-ui-smoke.json
```

Typical diagnostic:

```text
browser automation unavailable
```

Fix: keep the generated blocked report as evidence, install the documented
browser automation dependency for the release environment, then rerun the same
command.

## WASI Runner Missing

Command:

```sh
bash scripts/release_v1_0_wasi_smoke.sh --report reports/wasi-smoke.json
```

Typical diagnostic:

```text
wasmtime
```

Fix: install `wasmtime` or keep the build-only report when runner evidence is
not available on the current host. Do not treat build-only output as runtime
isolation evidence.

## Eco Lock Mismatch

Command:

```sh
./tetra eco verify --target linux-x64 --lock tetra.lock.json Tetra.capsule Core.capsule
go run ./tools/cmd/validate-eco-lock --lock tetra.lock.json
```

Typical diagnostic:

```text
graph_sha256 mismatch
```

Fix: regenerate the lock from the capsule files in the same branch state, then
rerun the validator. If the mismatch came from a dependency permission or target
change, review the capsule graph before accepting the new lock.

