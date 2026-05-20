# Tetra Tooling and Stdlib Production Audit

Status: achieved for the current `v0.4.0` local daily-development profile.

Audit date: 2026-05-06.

This audit maps the daily-development tooling and standard-library production
goal to concrete evidence. It is stricter than a docs-only consistency check:
generated docs, feature registry entries, release gates, and validators are
accepted as evidence only where they cover a named requirement below.

## Objective Restatement

Production-ready daily development for the current profile requires:

- stable CLI workflows for `check`, `build`, `run`, `fmt`, `test`, `doc`,
  `doctor`, `project`, `workspace`, `smoke`, `eco`, `version`, and `lsp`
- stable formatter, diagnostics, test runner, docs generation, examples, and
  validators
- stable `lib.core` modules for collections, strings, slices, math, IO,
  filesystem, networking, async, sync, testing, serialization, time, and crypto
  interfaces
- complete docs, examples, tests, compatibility policy, and release-gate
  evidence
- no mock or filler production claims for the stdlib/tooling surface

## Prompt-To-Artifact Checklist

| Requirement | Required artifact or command | Current evidence | Result |
| --- | --- | --- | --- |
| CLI workflow registry is current | `./tetra features --format=json`; `cli.core` | `cli.core` is `current` and lists `check/build/run/fmt/test/doc/doctor/targets/features/formats/new/interface/project/workspace/smoke/eco/clean/version/lsp` local workflows. | pass |
| CLI contract describes the active release | `docs/spec/cli_contracts.md` | Contract names the current `v0.4.0` tooling profile and documents the command surface, JSON reports, diagnostics, project/workspace commands, Eco workflows, and LSP stdio baseline. | pass |
| Formatter is release-gated | `./tetra fmt --check examples lib __rt compiler/selfhostrt` | Fresh full gate `reports/test-all-20260506-192335/summary.md` step 06 passed. | pass |
| Diagnostics are stable and validated | JSON diagnostic gate and diagnostic validator tests | Fresh full gate step 11 passed; `docs/spec/cli_contracts.md` documents stable diagnostic fields and code families. | pass |
| LSP is validated | LSP smoke and JSON-RPC stdio transcript validators | Fresh full gate steps 19 and 20 passed. The contract intentionally scopes rename to conservative single-file top-level edits with ambiguity refusal. | pass |
| Test runner is validated | `./tetra test examples`; JSON report gate | Fresh full gate steps 13 and 14 passed. | pass |
| Docs generation is validated | `./tetra doc examples`; generated API docs gate | Fresh full gate steps 21 and 22 passed. | pass |
| Project/workspace tooling is covered | CLI contract plus Go/tool validators | `docs/spec/cli_contracts.md` documents project and workspace commands; full gate step 01 runs all CLI/tool Go tests. | pass |
| Validators are wired into release evidence | `bash scripts/ci/test-all.sh` | Fresh full gate `reports/test-all-20260506-192335/summary.md` passed all 27 steps, including docs, smoke, target, LSP, API-docs, Eco, and cross-target checks. | pass |
| Stable stdlib module files exist | `lib/core/*.tetra` | Files exist for async, capability, collections, crypto, filesystem, io, math, memory, networking, serialization, slices, strings, sync, testing, and time. The named goal modules are present. | pass |
| Stable stdlib examples exist | `examples/core_*_smoke.tetra` | Smoke examples exist for every stable core module, including filesystem, networking, and crypto. | pass |
| Stdlib API docs render and validate | `./tetra doc lib/core lib/experimental examples/core_*_smoke.tetra`; `validate-api-docs` | Fresh stdlib API docs validation passes; full gate step 22 also validates generated API docs. | pass |
| Stdlib feature registry is current | `stdlib.core-current`; `stdlib.experimental-mirrors` | `validate-tooling-stdlib-readiness` accepts the fresh `./tetra features --format=json` report, the stdlib spec, and CLI contract. | pass |
| Filesystem has a real current slice | Runtime ABI docs, runtime symbols, compiler/runtime tests, smoke | `lib.core.filesystem.exists` is capability-gated and host-backed on linux-x64 through `__tetra_fs_exists`; unsupported targets report explicit diagnostics. Targeted filesystem tests and the full gate pass. | pass |
| Networking surface is honest | `docs/spec/stdlib.md`; `docs/user/standard_library_guide.md`; smoke | Networking is a stable endpoint policy-helper surface. It does not claim socket, DNS, or HTTP transport behavior. | pass |
| Crypto surface is honest | `docs/spec/stdlib.md`; `docs/user/standard_library_guide.md`; smoke | Crypto is a stable interface-helper surface with deterministic helper behavior. It does not claim encryption, authentication, entropy, or reviewed algorithm coverage. | pass |
| Compatibility policy exists | `docs/spec/stdlib_naming_versioning.md`; experimental mirrors | Stable names and compatibility mirror expectations are documented; `lib.experimental.*` mirrors forward stable callers to `lib.core.*`. | pass |
| No mock/filler stdlib production claims | `validate-tooling-stdlib-readiness`; text scan | The readiness validator rejects `placeholder` and `mock` in stdlib production evidence; current `docs/spec/stdlib.md`, `docs/user/standard_library_guide.md`, `compiler/features.go`, stable modules, and core smokes pass the scan. | pass |
| Graphify evidence is current after code edits | `graphify update .` | Rebuilt `graphify-out/graph.json` and `GRAPH_REPORT.md` after the final Go-code edit. | pass |

## Fresh Verification

Latest full release-style gate:

```sh
bash scripts/ci/test-all.sh
```

Result: pass. Evidence: `reports/test-all-20260506-192335/summary.md`, 27/27
steps passed, 0 failed.

Focused readiness gate:

```sh
./tetra features --format=json > /tmp/tetra-tooling-stdlib-features.json
go run ./tools/cmd/validate-tooling-stdlib-readiness \
  --features /tmp/tetra-tooling-stdlib-features.json \
  --stdlib-docs docs/spec/stdlib.md \
  --cli-contracts docs/spec/cli_contracts.md
```

Result: pass.

Docs and manifest gates:

```sh
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

Result: pass.

Workspace hygiene:

```sh
git diff --check
```

Result: pass.

## Scope Boundaries

This audit does not claim a complete project-wide IDE server, generic container
stdlib, host networking transport, reviewed cryptographic algorithms, or broad
multi-platform filesystem implementation. Those remain separate future
expansions. The achieved claim is the current release-covered daily local
development profile with honest module contracts, executable examples,
validators, and release-gate evidence.
