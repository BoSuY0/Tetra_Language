# Troubleshooting

Status: user guide for common command failures in the current `v0.3.0` profile
and future v1.0 preparation.

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
targets are `linux-x64`, `macos-x64`, and `windows-x64`; WASM runner-gated
runtime targets are `wasm32-wasi` and `wasm32-web`.

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
./tetra check path/to/non-release-covered-preview-feature.tetra
```

Typical diagnostic:

```text
planned feature
```

Fix: confirm the feature status in `compiler/features.go` and
`docs/spec/current_supported_surface.md`. Features listed as `experimental`,
`planned`, or `post-v1` need a promotion checklist before being treated as
current support.

## Web Smoke Blocked

Command:

```sh
bash scripts/release/v1_0/web-smoke.sh --report reports/web-ui-smoke.json
```

Typical diagnostic:

```text
browser automation unavailable
```

Fix: keep the generated blocked report as evidence, install the documented
browser automation dependency for the release environment, then rerun the same
command. These `release_v1_0_*` helpers are future-release maintainer tooling;
they are not current `v0.3.0` runtime support by themselves.

## WASI Runner Missing

Command:

```sh
bash scripts/release/v1_0/wasi-smoke.sh --report reports/wasi-smoke.json
```

Typical diagnostic:

```text
wasmtime
```

Fix: install `wasmtime` or keep the artifact/import preflight report when
runner evidence is not available on the current host. Do not treat
artifact/import output as runtime isolation evidence, and do not treat the
`release_v1_0_*` helper name as a `v0.3.0` runtime support claim.

## Eco Lock Mismatch

Command:

```sh
./tetra project sync --check .
./tetra project sync .
go run ./tools/cmd/validate-eco-lock --lock Tetra.lock
```

Typical diagnostic:

```text
graph_sha256 mismatch
```

Fix: run `tetra project sync .` from the project root to refresh `Tetra.lock`
and any generated local dependency artifacts in the same branch state, then
rerun the validator. If the mismatch came from a dependency permission or target
change, review the capsule graph before accepting the new lock. For explicit
capsule graph work, the lower-level `tetra eco verify --lock Tetra.lock
Capsule.t4` path is still available.

## Example Index Mismatch

Command:

```sh
go run ./tools/cmd/validate-example-index --smoke-list reports/smoke-list-linux-x64.json --index docs/user/examples_index.md
```

Typical diagnostic:

```text
example index missing examples/...
```

Fix: add or correct the row in `docs/user/examples_index.md` so the example path,
target group, and expected behavior match the smoke-list case or documented
exclusion.

## Smoke List Coverage Gap

Command:

```sh
go run ./tools/cmd/validate-smoke-list --report reports/smoke-list-linux-x64.json --examples-root examples
```

Typical diagnostic:

```text
example examples/... is not assigned to a smoke case or documented exclusion
```

Fix: either add the example to smoke-list generation or add it to
`excluded_examples` with a clear reason. Keep paths portable (`examples/...`,
forward slashes, no absolute paths).

## Test Report Shape Mismatch

Command:

```sh
go run ./tools/cmd/validate-test-report --report reports/examples-test-report.json
```

Typical diagnostic:

```text
report counts mismatch
```

Fix: regenerate the JSON report with `./tetra test --report=json examples`,
then verify per-file totals, per-test order (`filename` then `index`), and
failure fields (`exit_code` or `error`) are consistent.

## Dogfood Example Fails But Smoke List Passes

Command:

```sh
./tetra run --target linux-x64 examples/projects/eco_dogfood/src/main.tetra
```

Typical diagnostic:

```text
excluded from linux-x64 smoke profile
```

Fix: for known exclusions (for example `eco_dogfood`), use direct `tetra run`
evidence instead of treating smoke-list exclusion as a regression. For required
dogfood smoke entries (`dogfood_cli`, `dogfood_actor_task`), any runtime failure
is a real regression.
