# v0.4.0 Release Gate Checklist

Status: Linux-x64 production gate selected; tag-ready status requires a passing
gate and clean worktree.

Canonical scope: `docs/spec/v0_4_scope.md`.
Current truth before promotion: `docs/spec/current_supported_surface.md`.
Completion audit: `docs/release/v0_4_0_completion_audit.md`.
Gap matrix: `docs/release/v0_4_0_prod_gap_matrix.json`.
Scope decisions: `docs/release/v0_4_0_scope_decisions.json`.
Artifact policy: `docs/release/artifact_policy.md`.
Security review gate: `docs/checklists/security_review_gate.md`.

## Required Gate

The final release command is:

```sh
bash scripts/release/v0_4_0/gate.sh --report-dir <report-dir> --require-clean
```

For local development evidence, the same gate may be run without
`--require-clean`; that proves gate mechanics but is not tag-ready evidence.

Both local entrypoints must report the promoted release version:

```sh
./tetra version
./t version
```

Both commands must print `v0.4.0`. The gate artifact mapping must be
`tetra.release.v0_4_0.gate-report.v1`.

## Required Preflight

The gate must fail before expensive evidence collection if any of these remain
true:

- `--report-dir` points at an existing non-empty directory.
- `go run ./tools/cmd/validate-v0-4-readiness --features <features.json>
  --targets <targets.json> --manifest docs/generated/manifest.json
  --scope-decisions docs/release/v0_4_0_scope_decisions.json` exits non-zero.
- `docs/release/v0_4_0_scope_decisions.json` contains an implemented-scope gap
  with missing implementation, tests, docs, or release-gate evidence.
- `./tetra features --format=json` reports any required scoped `v0.4.0` feature
  as `experimental`, `planned`, or `post-v1`.
- `./tetra targets --format=json` does not report `linux-x64` as runnable.
- `docs/generated/manifest.json` reports a compiler version other than
  `v0.4.0`.
- The worktree is dirty when `--require-clean` is set.

## Required Evidence

The final gate must collect or validate these checks in the same branch state:

```sh
go test ./compiler/... ./cli/... ./tools/... -count=1
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_local_smoke_skip_db_report.json --allow-skip-db
go run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_scram_single_query_local_report.json
go run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_scram_single_query_matrix_local_report.json
go run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json
go run ./cli/cmd/tetra smoke --target linux-x64 --run=true --report reports/v0.4.0/linux-host-smoke.json
bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh --report-dir reports/v0.4.0
go run ./tools/cmd/validate-memory-production --report reports/v0.4.0/memory-production-linux-x64.json
bash scripts/release/post_v0_4/parallel-production-linux-x64-smoke.sh --report-dir reports/v0.4.0
go run ./tools/cmd/validate-parallel-production --report reports/v0.4.0/parallel-production-linux-x64.json
bash scripts/release/post_v0_4/compiler-production-linux-x64-smoke.sh --report-dir reports/v0.4.0
go run ./tools/cmd/validate-compiler-production --report reports/v0.4.0/compiler-production-linux-x64.json
bash scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh --report-dir reports/v0.4.0
bash scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh --report-dir reports/v0.4.0
go run ./tools/cmd/validate-native-ui-runtime --report reports/v0.4.0/native-ui-linux-x64.json
go run ./tools/cmd/validate-distributed-actor-runtime --report reports/v0.4.0/distributed-actors-linux-x64.json
git diff --check
git status --porcelain --untracked-files=all
```

The gate must also carry release-gate evidence for every selected production
decision:

- callable Level 1, Level 2, and selected first-class callable semantics
- local/control-flow lifetime SSA behavior
- Linux-x64 Memory Production Core behavior
- Linux-x64 Parallelism Production Core behavior
- Linux-x64 Compiler Production Core behavior
- standard-library compatibility mirror policy
- UI metadata v1 for the selected Linux-x64 UI/native shell surface
- Linux-x64 distributed actor runtime behavior
- Linux-x64 native UI runtime behavior
- Linux-x64 host-native runtime smoke behavior

Each implemented decision in `docs/release/v0_4_0_scope_decisions.json` must
carry non-empty evidence buckets:

- `implementation`
- `tests`
- `docs`
- `release_gate_evidence`

`implementation` and `docs` entries must be readable repository-relative paths;
`docs` entries must live under `docs/`. Test commands and release-gate report
paths remain validated by the gate steps that execute or produce them.

## Required Artifacts

The report directory must contain at least:

- `<report-dir>/summary.json`
- `<report-dir>/summary.md`
- `<report-dir>/logs/*.log`
- `<report-dir>/artifacts/features.json`
- `<report-dir>/artifacts/targets.json`
- `<report-dir>/artifacts/linux-host-smoke.json`
- `<report-dir>/artifacts/memory-production-linux-x64.json`
- `<report-dir>/artifacts/parallel-production-linux-x64.json`
- `<report-dir>/artifacts/compiler-production-linux-x64.json`
- `<report-dir>/artifacts/distributed-actors-linux-x64.json`
- `<report-dir>/artifacts/native-ui-linux-x64.json`
- `<report-dir>/artifacts/release-state.json`
- `<report-dir>/artifacts/release-state.txt`
- `<report-dir>/artifacts/security-review.md`
- `<report-dir>/artifacts/security-review.md.sha256`
- `<report-dir>/artifact-hashes.json`

The summary must validate with:

```sh
go run ./tools/cmd/validate-release-gate-summary \
  --summary <report-dir>/summary.json \
  --report-dir <report-dir> \
  --expected-version v0.4.0 \
  --expected-artifact tetra.release.v0_4_0.gate-report.v1 \
  --expected-command 'bash scripts/release/v0_4_0/gate.sh'
```

The hash manifest must validate with:

```sh
go run ./tools/cmd/validate-artifact-hashes \
  --manifest <report-dir>/artifact-hashes.json
```

For a passing `v0.4.0` summary, `<report-dir>/artifact-hashes.json` must list
the required release-gate artifacts, including:

- `summary.json`
- `summary.md`
- `artifacts/features.json`: `tetra.features.v1`
- `artifacts/targets.json`
- `artifacts/linux-host-smoke.json`
- `artifacts/memory-production-linux-x64.json`:
  `tetra.memory.production.v1`
- `artifacts/parallel-production-linux-x64.json`:
  `tetra.parallel.production.v1`
- `artifacts/compiler-production-linux-x64.json`:
  `tetra.compiler.production.v1`
- `artifacts/distributed-actors-linux-x64.json`:
  `tetra.actors.distributed-runtime.v1`
- `artifacts/native-ui-linux-x64.json`:
  `tetra.ui.native-runtime.v1`
- `artifacts/release-state.json`:
  `tetra.release.v0_4_0.release-state.v1`
- `artifacts/release-state.txt`
- `artifacts/security-review.md`
- `artifacts/security-review.md.sha256`

## Security Review

Security signoff is mandatory for the exact `v0.4.0` candidate. The signoff
must cover:

- filesystem, networking, crypto, and capability effects for the scoped Linux
  surface
- Linux-x64 runtime execution and native process boundaries
- UI event dispatch and command execution
- distributed actors, scheduling, cancellation, and failure modes
- artifact hashes and release-state integrity

EcoNet, WASI/Web runtime boundaries, Windows, and macOS are explicitly outside
the scoped `v0.4.0` production claim.

## Promotion Boundaries

- `v0.3.0` gate reports are historical evidence only; they cannot prove
  `v0.4.0` readiness.
- `v1.0.0` future docs or scripts cannot be used as `v0.4.0` evidence unless
  the `v0.4.0` gate explicitly names the reused command and validates the
  version boundary.
- Build-only, metadata-only, preview-only, experimental, planned, or post-v1
  evidence cannot satisfy a selected production gap.

## Done When

The release is tag-ready only after:

- the Linux-x64 production scope in `docs/spec/v0_4_scope.md` is implemented
- `bash scripts/release/v0_4_0/gate.sh --report-dir <report-dir> --require-clean`
  passes
- the report directory contains the required artifacts
- the final handoff records the clean worktree result and the exact release
  commit
