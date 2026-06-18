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
| Version is marked `v0.4.0` | `./tetra version` | version is `v0.4.0` | pass |
| | `./t version`; `compiler/internal/version/version.go` | local metadata reports `v0.4.0` | |
| Manifest is marked `v0.4.0` | `docs/generated/manifest.json` | manifest says `v0.4.0` | pass |
| Linux-x64 production scope is selected | scope decisions | Linux-x64 selected | pass |
| | | EcoNet, full v1, WASM, Windows, and macOS are excluded | |
| Feature registry has no required non-production gap | feature registry | scoped current | pass |
| | `reports/v0.4.0/features.json` | scoped features are current | |
| | `./tetra features --format=json` | EcoNet stays `post-v1`; full v1 stays planned | |
| Callable model is production | callable evidence | callable levels are current | pass |
| | callable registry, docs, tests, evidence | | |
| | | excluded escape diagnostics are stable | |
| Lifetime SSA is production for the selected surface | lifetime evidence | current | pass |
| | `language.lifetime-ssa` | current since `v0.4.0` | |
| | ownership/lifetime implementation, tests, docs, evidence | selected join solver is covered | |
| Memory production core is production | memory smoke and validator | memory JSON produced | pass |
| | `artifacts/memory-production-linux-x64.json` | hash schema is `tetra.memory.production.v1` | |
| Parallel production core is production | parallel smoke | parallel JSON produced | pass |
| | parallel validator | | |
| | `artifacts/parallel-production-linux-x64.json` | schema `tetra.parallel.production.v1` | |
| Compiler production core is production | compiler smoke | compiler JSON produced | pass |
| | compiler validator | | |
| | `artifacts/compiler-production-linux-x64.json` | schema `tetra.compiler.production.v1` | |
| Standard library mirror policy is production | mirror policy | mirrors forward | pass |
| | `stdlib.experimental-mirrors` | mirrors forward to `lib.core.*` | |
| | stdlib docs/tests | stable callers are directed to `lib.core.*` | |
| UI metadata/runtime/native behavior is production | UI evidence | Linux-x64 current | pass |
| | `ui.metadata-v1`; `ui.native-runtime` | Linux-x64 evidence is current | |
| | UI runtime smoke evidence | native UI report validates executable evidence | |
| Distributed actors are production | actor runtime evidence | actor report validates | pass |
| | `actors.distributed-runtime` | Linux-x64 actor report validates | |
| | actor runtime smoke evidence | distributed actor runtime slice is covered | |
| Linux runtime is production | Linux host smoke report | Linux host smoke passes 64/64 | pass |
| | `reports/v0.4.0/linux-host-smoke.json` | | |
| WASM runtime execution is production | WASM scope decision | excluded from scope | not required |
| | | excluded from Linux-x64 scope | |
| | `wasm.runtime-execution`; `wasm32-wasi` | | |
| | `wasm32-web` | | |
| Distributed EcoNet is production | EcoNet scope decision | excluded from scope | not required |
| | `eco.distributed-network` | excluded from Linux-x64 scope | |
| Windows runtime is production | Windows scope decision | excluded from scope | not required |
| | | excluded from Linux-x64 scope | |
| | `windows-x64` runtime smoke report | | |
| macOS runtime is production | macOS scope decision | excluded from scope | not required |
| | | excluded from Linux-x64 scope | |
| | `macos-x64` runtime smoke report | | |
| `v0.4.0` readiness preflight passes | readiness validator | readiness passes | pass |
| | | features, targets, manifest, and scope decisions are checked | |
| `v0.4.0` release gate exists | release gate script | expanded gate is canonical | pass |
| | `scripts/release/v0_4_0/gate.sh` | | |
| | clean candidate gate command | gate has 22 production, readiness, and signoff steps | |
| `v0.4.0` security review exists | security review artifacts | signoff exits 0 | pass |
| | security-review script and signoff report | | |
| Generated docs verification covers the objective | verify-docs command | docs passed | pass |
| | | scoped gate passed docs verification | |
| | `go run ./tools/cmd/verify-docs ...` | scoped gate passed docs verification | |
| Baseline tests pass | baseline package tests | scoped gate passed baseline | pass |
| | `go test ./compiler/... ./cli/... ./tools/... -count=1` | | |
| Worktree is clean for release | clean worktree preflight | candidate is committed clean | pass |
| | `git status --porcelain --untracked-files=all` | final candidate is committed clean | |
| | `--require-clean`; release-state artifact | clean preflight proves tag-ready candidate | |

## Release Evidence Matrix

| Requirement | File(s) | Tests | Docs | Evidence | Status |
| --- | --- | --- | --- | --- | --- |
| Version, manifest, and scoped release identity are current | | | | | pass |
| | implementation: version, manifest, feature files | positive: version checks | | | |
| | | | docs: scope docs | report: features | |
| | `compiler/internal/version/version.go` | negative: stale fixtures reject drift | | | |
| | | | manifest: generated manifest | graphify: update | |
| | `compiler/manifest.go`; `compiler/features.go` | | | ci: release gate | |
| Compiler production core is production | | | | | pass |
| | implementation: compiler, lower, backend, validation | positive: compiler tests | | | |
| | | | docs: surface spec | report: compiler JSON | |
| | `compiler/compiler.go`; `compiler/internal/lower` | negative: validators reject fakes | | | |
| | | | manifest: generated manifest | graphify: update | |
| | `compiler/internal/backend`; `compiler/internal/validation` | | | ci: release gate | |
| Memory production core is production | | | | | pass |
| | implementation: runtimeabi, allocplan, lower | positive: memory package tests | | | |
| | | | docs: runtime ABI | report: memory JSON | |
| | `compiler/internal/runtimeabi`; `compiler/internal/allocplan` | | | | |
| | | negative: validators reject gaps | manifest: generated manifest | graphify: update | |
| | `compiler/internal/lower` | | | ci: release gate | |
| Parallel and actor production core is production | | | | | pass |
| | implementation: actorsrt, parallelrt, netrt | positive: actor/parallel tests | | | |
| | | | docs: actors docs | report: parallel JSON | |
| | `compiler/internal/actorsrt`; `compiler/internal/parallelrt` | | | | |
| | | negative: validators reject gaps | manifest: generated manifest | graphify: update | |
| | `compiler/internal/netrt` | | | ci: release gate | |
| UI/native/Linux runtime scoped behavior is production | | | | | pass |
| | implementation: webrt, htmlrt, surface, UI tools | positive: native UI smoke | | | |
| | | | docs: UI docs | report: native UI JSON | |
| | `compiler/internal/webrt`; `compiler/internal/htmlrt` | negative: validators reject gaps | | | |
| | | | manifest: generated manifest | graphify: update | |
| | `lib/core/surface.tetra`; `tools/cmd/*ui*` | | | ci: release gate | |
| Release gate, security review, docs, tests, and clean-worktree evidence are complete | | | | | |
| | | | | | pass |
| | implementation: release gate and security review | positive: audit validation | | | |
| | | | docs: release audit | report: security review | |
| | `scripts/release/v0_4_0/gate.sh` | negative: validators reject dirty pass | | | |
| | | | manifest: generated manifest | graphify: update | |
| | `scripts/release/v0_4_0/security-review.sh` | release-state/docs/baseline pass | | | |
| | | | | ci: release gate | |
| | `tools/cmd/validate-release-state` | | | | |

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
