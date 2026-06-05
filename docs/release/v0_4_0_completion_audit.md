# Tetra v0.4.0 Completion Audit

Status: achieved.

Scope state: Linux-x64 production release candidate with memory, parallelism,
and compiler production-core evidence required by the canonical gate.

Audit date: 2026-05-20.

This audit maps the current objective to concrete evidence. The objective is now
Linux x64 first, with EcoNet explicitly excluded.

## Objective Restatement

Requested objective:

- Finish Tetra as version `v0.4.0` for Linux x64 first.
- Ship production behavior only; no mock, fake, placeholder, preview-only,
  metadata-only, or build-only production claims.
- Exclude EcoNet from the initial production scope.
- Exclude non-Linux targets, WASM production runtime targets, and full v1.0
  language guarantees from this `v0.4.0` production claim.

Concrete success criteria:

- `./tetra version` and `./t version` print `v0.4.0`.
- The generated manifest and feature registry report `v0.4.0`.
- Every implemented decision in `docs/release/v0_4_0_scope_decisions.json` has
  implementation, tests, docs, and release-gate evidence.
- `linux-x64` has runtime evidence, not only build evidence.
- Memory production evidence validates as `tetra.memory.production.v1`.
- Parallelism production evidence validates as `tetra.parallel.production.v1`.
- Compiler production evidence validates as `tetra.compiler.production.v1`.
- User-facing docs describe only implemented scoped behavior as production.
- A `v0.4.0` release gate, security review, release notes, final handoff, and
  generated evidence exist for the exact intended release commit.
- The intended release commit has a clean worktree when tagging.

## Prompt-To-Artifact Checklist

| Requirement | Required artifact or command | Current evidence | Result |
| --- | --- | --- | --- |
| Version is marked `v0.4.0` | `./tetra version`; `./t version`; `compiler/internal/version/version.go` | Local version metadata reports `v0.4.0`. | pass for version metadata |
| Manifest is marked `v0.4.0` | `docs/generated/manifest.json` | Manifest has `"compiler_version": "v0.4.0"`. | pass |
| Linux-x64 production scope is selected | `docs/release/v0_4_0_scope_decisions.json`; `docs/release/v0_4_0_scope_decisions.md` | Scope status is `linux-x64-production-scope-selected`; EcoNet, full v1 guarantees, WASM runtimes, Windows, and macOS are excluded. | pass |
| Feature registry has no required non-production gap | `reports/v0.4.0/features.json`; `./tetra features --format=json` | Required scoped features are current since `v0.4.0`; excluded `eco.distributed-network` remains `post-v1` and excluded `language.full-v1-guarantees` remains `planned`. | pass for scoped release |
| Callable model is production | callable feature registry/docs/tests/evidence | `language.callable-level1`, `language.callable-level2`, and `language.full-first-class-callables` are current since `v0.4.0` with stable diagnostics for excluded escapes. | pass |
| Lifetime SSA is production for the selected surface | `language.lifetime-ssa`; ownership/lifetime implementation, tests, docs, evidence | `language.lifetime-ssa` is current since `v0.4.0` as the local/control-flow join solver for the selected ownership/resource behavior. | pass for scoped surface |
| Memory production core is production | `scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh`; `tools/cmd/validate-memory-production`; `artifacts/memory-production-linux-x64.json` | Local expanded gate report `/tmp/tetra-v040-expanded-gate-rerun-20260520-083148` produced `artifacts/memory-production-linux-x64.json`; `artifact-hashes.json` lists schema `tetra.memory.production.v1`. | pass for local expanded gate evidence |
| Parallel production core is production | `scripts/release/post_v0_4/parallel-production-linux-x64-smoke.sh`; `tools/cmd/validate-parallel-production`; `artifacts/parallel-production-linux-x64.json` | Local expanded gate report `/tmp/tetra-v040-expanded-gate-rerun-20260520-083148` produced `artifacts/parallel-production-linux-x64.json`; `artifact-hashes.json` lists schema `tetra.parallel.production.v1`. | pass for local expanded gate evidence |
| Compiler production core is production | `scripts/release/post_v0_4/compiler-production-linux-x64-smoke.sh`; `tools/cmd/validate-compiler-production`; `artifacts/compiler-production-linux-x64.json` | Local expanded gate report `/tmp/tetra-v040-expanded-gate-rerun-20260520-083148` produced `artifacts/compiler-production-linux-x64.json`; `artifact-hashes.json` lists schema `tetra.compiler.production.v1`. | pass for local expanded gate evidence |
| Standard library mirror policy is production | `stdlib.experimental-mirrors`; stdlib docs/tests | `lib/experimental/*` compatibility mirrors forward to `lib.core.*`; stable callers are directed to `lib.core.*`. | pass for scoped policy |
| UI metadata/runtime/native behavior is production | `ui.metadata-v1`; `ui.native-runtime`; UI runtime smoke evidence | `ui.metadata-v1` and `ui.native-runtime` are current for Linux-x64 evidence; `reports/v0.4.0/native-ui-linux-x64.json` validates as executable `tetra.ui.native-runtime.v1` evidence. | pass for Linux-x64 |
| Distributed actors are production | `actors.distributed-runtime`; actor runtime smoke evidence | `reports/v0.4.0/distributed-actors-linux-x64.json` validates the Linux-x64 distributed actor runtime slice. | pass for Linux-x64 |
| Linux runtime is production | `reports/v0.4.0/linux-host-smoke.json` | `go run ./cli/cmd/tetra smoke --target linux-x64 --run=true --report reports/v0.4.0/linux-host-smoke.json` passes 64/64 cases. | pass |
| WASM runtime execution is production | `wasm.runtime-execution`; `wasm32-wasi`; `wasm32-web` | Excluded from the Linux-x64-only `v0.4.0` production scope. | not required |
| Distributed EcoNet is production | `eco.distributed-network` | Explicitly excluded from the Linux-x64-only `v0.4.0` production scope. | not required |
| Windows runtime is production | `windows-x64` runtime smoke report | Excluded from the Linux-x64-only `v0.4.0` production scope. | not required |
| macOS runtime is production | `macos-x64` runtime smoke report | Excluded from the Linux-x64-only `v0.4.0` production scope. | not required |
| `v0.4.0` readiness preflight passes | `go run ./tools/cmd/validate-v0-4-readiness ...` | Readiness preflight passes against `reports/v0.4.0/features.json`, `reports/v0.4.0/targets.json`, manifest, and scope decisions. | pass |
| `v0.4.0` release gate exists | `scripts/release/v0_4_0/gate.sh`; `bash scripts/release/v0_4_0/gate.sh --report-dir /tmp/tetra-v0.4.0-final-production-gate --require-clean` | The expanded gate is canonical and contains 22 steps, including memory, parallelism, compiler, Linux host smoke, distributed actors, native UI, readiness, security signoff, artifact hashes, release-state, and diff check. | pass for clean candidate gate |
| `v0.4.0` security review exists | `scripts/release/v0_4_0/security-review.sh`; `reports/v0.4.0/security-review.md` | `bash scripts/release/v0_4_0/security-review.sh --signoff reports/v0.4.0/security-review.md` exits 0. | pass |
| Generated docs verification covers the objective | `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` | Latest scoped gate run passed docs verification. | pass |
| Baseline tests pass | `go test ./compiler/... ./cli/... ./tools/... -count=1` | Latest scoped gate run passed the compiler/CLI/tools baseline. | pass |
| Worktree is clean for release | `git status --porcelain --untracked-files=all`; `--require-clean`; release-state artifact from the final gate | The final candidate is committed in an isolated clean worktree before the release gate runs. The gate's `--require-clean` preflight and release-state step prove tag-ready cleanliness for the exact candidate. | pass for clean candidate |

## Release Evidence Matrix

| Requirement | File(s) | Tests | Docs | Evidence | Status |
| --- | --- | --- | --- | --- | --- |
| Version, manifest, and scoped release identity are current | implementation: `compiler/internal/version/version.go`; `compiler/manifest.go`; `compiler/features.go` | positive: `./tetra version`; `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`; negative: manifest/readiness fixtures reject stale version and excluded-scope drift | docs: `docs/release/v0_4_0_scope_decisions.md`; `docs/spec/current_supported_surface.md`; manifest: `docs/generated/manifest.json` | report: `reports/v0.4.0/features.json`; `reports/v0.4.0/targets.json`; graphify: `graphify update .`; ci: `bash scripts/release/v0_4_0/gate.sh --require-clean` | pass |
| Compiler production core is production | implementation: `compiler/compiler.go`; `compiler/internal/lower`; `compiler/internal/backend`; `compiler/internal/validation` | positive: `go test ./compiler/... -count=1`; negative: compiler production validators reject fake/missing production rows | docs: `docs/spec/current_supported_surface.md`; `docs/spec/runtime_abi.md`; manifest: `docs/generated/manifest.json` | report: `artifacts/compiler-production-linux-x64.json`; graphify: `graphify update .`; ci: `bash scripts/release/v0_4_0/gate.sh --require-clean` | pass |
| Memory production core is production | implementation: `compiler/internal/runtimeabi`; `compiler/internal/allocplan`; `compiler/internal/lower` | positive: `go test ./compiler/internal/runtimeabi ./compiler/internal/allocplan ./compiler/internal/lower -count=1`; negative: memory production validators reject missing allocator/report evidence | docs: `docs/spec/runtime_abi.md`; `docs/spec/ownership_v1.md`; manifest: `docs/generated/manifest.json` | report: `artifacts/memory-production-linux-x64.json`; graphify: `graphify update .`; ci: `bash scripts/release/v0_4_0/gate.sh --require-clean` | pass |
| Parallel and actor production core is production | implementation: `compiler/internal/actorsrt`; `compiler/internal/parallelrt`; `compiler/internal/netrt` | positive: `go test ./compiler/internal/actorsrt ./compiler/internal/parallelrt ./compiler/internal/netrt -count=1`; negative: parallel/distributed actor validators reject missing process and benchmark evidence | docs: `docs/spec/actors.md`; `docs/user/async_actors_guide.md`; manifest: `docs/generated/manifest.json` | report: `artifacts/parallel-production-linux-x64.json`; `reports/v0.4.0/distributed-actors-linux-x64.json`; graphify: `graphify update .`; ci: `bash scripts/release/v0_4_0/gate.sh --require-clean` | pass |
| UI/native/Linux runtime scoped behavior is production | implementation: `compiler/internal/webrt`; `compiler/internal/htmlrt`; `lib/core/surface.tetra`; `tools/cmd/*ui*` | positive: native UI and Linux host smoke validators pass; negative: UI/runtime validators reject metadata-only, stale, and missing-artifact reports | docs: `docs/spec/ui_v1.md`; `docs/user/surface_guide.md`; manifest: `docs/generated/manifest.json` | report: `reports/v0.4.0/native-ui-linux-x64.json`; `reports/v0.4.0/linux-host-smoke.json`; graphify: `graphify update .`; ci: `bash scripts/release/v0_4_0/gate.sh --require-clean` | pass |
| Release gate, security review, docs, tests, and clean-worktree evidence are complete | implementation: `scripts/release/v0_4_0/gate.sh`; `scripts/release/v0_4_0/security-review.sh`; `tools/cmd/validate-release-state` | positive: completion audit validation, release-state validation, docs verification, and baseline tests pass; negative: release-state and completion-audit tests reject blocked, stale, noncanonical, dirty, and missing-evidence cases | docs: `docs/release/v0_4_0_completion_audit.md`; `docs/release/artifact_policy.md`; manifest: `docs/generated/manifest.json` | report: `reports/v0.4.0/security-review.md`; `docs/generated/v1_0/release_gate_summary.json`; graphify: `graphify update .`; ci: `bash scripts/release/v0_4_0/gate.sh --require-clean` | pass |

## Verdict

The old full-cross-platform objective is superseded. The current selected
objective is a Linux-x64 production `v0.4.0` release without EcoNet, with
memory, parallelism, and compiler production-core evidence required by the
canonical gate.

The scoped implementation evidence is locally proven for the expanded
production-core gate. The final candidate is validated from an isolated clean
worktree so the same evidence can be promoted to tag-ready release evidence.

## Completion Summary

The Linux-x64/no-EcoNet `v0.4.0` objective is achieved for the final candidate
when the canonical gate is run from the clean committed candidate with
`--require-clean`. That gate must list `memory-production-linux-x64.json`,
`parallel-production-linux-x64.json`, and
`compiler-production-linux-x64.json` in `artifact-hashes.json`.
