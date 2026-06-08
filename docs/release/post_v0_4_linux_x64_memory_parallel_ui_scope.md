# Post-v0.4 Linux-x64 Memory, Parallelism, And UI Scope

Status: active production scope for the current `/goal`.

Machine-readable source:
`docs/release/post_v0_4_linux_x64_memory_parallel_ui_scope.json`.

## Scope

This line promotes Tetra beyond the scoped `v0.4.0` Linux-x64 release in three
ordered runtime layers, with an additional compiler production evidence layer
for the final language toolchain claim:

1. Memory Production Core.
2. Parallelism Production Core.
3. UI Production Runtime.

For the active final-language objective, the ordered compiler gate is:

1. Memory Production Core.
2. Parallelism Production Core.
3. Compiler Production Core.

The order is required. Parallelism depends on the memory model, and UI depends
on both memory and scheduler safety.

## Evidence Artifacts

| Layer | Required artifact |
| --- | --- |
| Memory Production Core | `tetra.memory.production.v1` |
| Parallelism Production Core | `tetra.parallel.production.v1` |
| Compiler Production Core | `tetra.compiler.production.v1` |
| UI Production Runtime | `tetra.ui.desktop-runtime.v1` |

The current Memory Production Core release-gate entrypoint is
`bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh --report-dir <dir>`.
It writes `memory-production-linux-x64.json` and runs
`go run ./tools/cmd/memory-production-smoke --report <path>`, followed by
`go run ./tools/cmd/validate-memory-production --report <path>`,
`go run ./cli/cmd/tetra targets --format=json > <dir>/targets.json`,
`go run ./tools/cmd/validate-targets --report <dir>/targets.json`,
`go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir <dir>/memory-fuzz-tier1`,
`go run ./tools/cmd/validate-memory-fuzz-oracle --report <dir>/memory-fuzz-tier1/memory-fuzz-oracle.json --artifact-dir <dir>/memory-fuzz-tier1`,
and
`go run ./tools/cmd/validate-artifact-hashes --manifest <dir>/artifact-hashes.json`,
then
`go run ./tools/cmd/validate-memory-production --report <dir>/memory-production-linux-x64.json --manifest <dir>/memory-release-manifest.json --report-dir <dir>`.
The `<dir>` value must be a fresh --report-dir: the gate refuses symlink,
non-directory, and non-empty report directories before it writes memory
evidence, so stale artifacts cannot be promoted as same-run proof. Local triage
keeps the boundary text exact:
`quick evidence is not full, stabilization, nightly, or release proof` unless
the matching full gate and validators ran for that artifact set.
The report directory must also contain `memory-release-manifest.json` linking
the memory production report, target report, Tier 1 fuzz reports, artifact hash
manifest, command provenance, target, git head, and report schemas. The Tier 1
fuzz oracle subdirectory must contain `memory-fuzz-oracle.json`, `summary.md`,
and `summary.json`; the validator checks those generated artifacts and command
provenance in addition to the oracle JSON. Tier 2 nightly seed triage and Tier
3 release-blocking focused fuzz remain policy boundaries for scheduled/release
evidence, not an exhaustive fuzz proof.
Its required memory evidence includes a deterministic `memcpy_u8`/`memset_u8`
fuzz-like length sweep in addition to positive, negative, and stress cases. The
gate also builds and runs checked-in memory examples for core memory,
ownership/borrow/consume behavior, and unsafe `cap.mem` usage. The
same artifact requires explicit unsafe-boundary, heap-closure handle,
callable mutable-capture heap escape, slice/struct borrow-escape, and
function-typed slice aggregate borrow-escape coverage cases backed by compiler
test execution, not only audit-text references. The
JSON artifact also embeds a completion audit that maps each Memory Production
Core requirement to the concrete artifact, command, or test evidence used by
the validator.

The current Parallelism Production Core release-gate entrypoint is
`bash scripts/release/post_v0_4/parallel-production-linux-x64-smoke.sh --report-dir <dir>`.
It writes `parallel-production-linux-x64.json` and runs
`go run ./tools/cmd/parallel-production-smoke --report <path>`, followed by
`go run ./tools/cmd/validate-parallel-production --report <path>` and
`go run ./tools/cmd/validate-artifact-hashes --manifest <dir>/artifact-hashes.json`.
The report must contain scheduler lifecycle, actor mailbox/backpressure,
transfer/race-safety, stable diagnostics for negative parallel cases, stress,
and completion-audit evidence. The required lifecycle and diagnostics evidence
includes cancel-wakes-deadline-join, nested cancellation propagation, task actor
mailbox handoff, double join rejection, and task-group use-after-close
rejection. It also requires explicit safe/unsafe/forbidden boundary coverage
backed by compiler tests for allowed immutable task targets, missing
runtime/actors effects, unsafe-only operations, and forbidden mutable actor/task
targets.

The current Compiler Production Core release-gate entrypoint is
`bash scripts/release/post_v0_4/compiler-production-linux-x64-smoke.sh --report-dir <dir>`.
It writes `compiler-production-linux-x64.json` and runs
`go run ./tools/cmd/compiler-production-smoke --report <path>`, followed by
`go run ./tools/cmd/validate-compiler-production --report <path>` and
`go run ./tools/cmd/validate-artifact-hashes --manifest <dir>/artifact-hashes.json`.
The `tetra.compiler.production.v1` evidence must prove a fresh CLI compiler
build, `v0.4.0` version identity, Linux-x64 native compile/run behavior,
Linux-x64 TOBJ object emission, interface-only compilation, `wasm32-wasi` and
`wasm32-web` module emission, frontend parser fixtures, semantic diagnostics,
IR verifier diagnostics, backend format emission, CLI build diagnostics,
compiler cache separation, deterministic backend output, and a smoke-profile
compilation matrix. It rejects docs-only, report-only, fake, mock, placeholder,
metadata-only, and sidecar-only compiler production claims.

The current UI Production Runtime release-gate entrypoint is
`bash scripts/release/post_v0_4/ui-production-runtime-linux-x64-smoke.sh --report-dir <dir>`.
It writes `ui-production-runtime-linux-x64.json` and runs
`go run ./tools/cmd/ui-production-runtime-smoke --report <path>`, which also
writes `native-ui-runtime-linux-x64.integration.json` by running
`go run ./tools/cmd/native-ui-runtime-smoke --report <native-path>`, followed
by `go run ./tools/cmd/validate-ui-production-runtime --report <path>` and
`go run ./tools/cmd/validate-artifact-hashes --manifest <dir>/artifact-hashes.json`.
The `tetra.ui.desktop-runtime.v1` validator requires Linux-x64 desktop runtime
process evidence, window lifecycle, layout, button/text/input/list/panel
widgets, input focus/input/change transitions, state binding, event loop
dispatch, async UI command completion, timer tick event evidence,
redraw/update lifecycle, error/crash handling, stable UI diagnostics, dogfood
application smoke, compiler-emitted UI bundle/native-shell trace load evidence,
sidecar-driven native UI runtime integration with a validated
`tetra.ui.native-runtime.v1` consistency case, widget stress, and an embedded
completion audit. It rejects runtime-less,
metadata-only, docs-only, build-only, web-only, sidecar-only, fake, mock, and
placeholder production evidence.

The combined ordered production gate is
`bash scripts/release/post_v0_4/memory-parallel-ui-production-linux-x64-gate.sh --report-dir <dir>`.
It runs the Memory, Parallelism, and UI release-gate entrypoints in order, then
re-validates `memory-production-linux-x64.json`,
`parallel-production-linux-x64.json`, `ui-production-runtime-linux-x64.json`,
`native-ui-runtime-linux-x64.integration.json`, and the final
`artifact-hashes.json` manifest for the report directory.

The compiler-focused combined ordered production gate is
`bash scripts/release/post_v0_4/memory-parallel-compiler-production-linux-x64-gate.sh --report-dir <dir>`.
It runs the Memory, Parallelism, and Compiler release-gate entrypoints in order,
then re-validates `memory-production-linux-x64.json`,
`parallel-production-linux-x64.json`, `compiler-production-linux-x64.json`, and
the final `artifact-hashes.json` manifest for the report directory.

Each artifact must be executable Linux-x64 evidence with positive cases,
negative cases, stress/fuzz or fuzz-like deterministic coverage, docs, examples,
strict validators, artifact hashes, and release-gate integration.

## Non-Goals

The following are outside this production line unless a future scope decision
promotes them:

- non-Linux production runtime promotion;
- WASM/browser production UI;
- EcoNet;
- full v1.0 language guarantees;
- metadata-only, docs-only, build-only, fake, mock, or placeholder production
  evidence.

## Completion Rule

This scope is complete only when all three layers have real Linux-x64
implementation, stable diagnostics, documentation, examples, smoke/stress/fuzz
evidence, strict validators, clean release-gate evidence, and a completion audit
that maps every requirement to concrete artifacts.
