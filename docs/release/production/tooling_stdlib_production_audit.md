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

### CLI Workflow Registry

- Required artifact or command: `./tetra features --format=json`; `cli.core`.
- Current evidence: `cli.core` is `current` and lists the local workflows for
  check/build/run/fmt/test/doc/doctor/targets/features/formats/new/interface.
- Current evidence: it also lists project/workspace/smoke/eco/clean/version/lsp.
- Result: pass.

### CLI Contract

- Required artifact or command: `docs/spec/cli_contracts.md`.
- Current evidence: contract names the current `v0.4.0` tooling profile.
- Current evidence: it documents command surface, JSON reports, diagnostics,
  project/workspace commands, Eco workflows, and LSP stdio baseline.
- Result: pass.

### Formatter Gate

- Required artifact or command:
  `./tetra fmt --check examples lib __rt compiler/selfhostrt`.
- Current evidence: fresh full gate
  `reports/test-all-20260506-192335/summary.md` step 06 passed.
- Result: pass.

### Diagnostics

- Required artifact or command: JSON diagnostic gate and validator tests.
- Current evidence: fresh full gate step 11 passed.
- Current evidence: `docs/spec/cli_contracts.md` documents stable diagnostic
  fields and code families.
- Result: pass.

### LSP

- Required artifact or command: LSP smoke and JSON-RPC stdio transcript
  validators.
- Current evidence: fresh full gate steps 19 and 20 passed.
- Current evidence: rename is scoped to conservative single-file top-level edits
  with ambiguity refusal.
- Result: pass.

### Test Runner

- Required artifact or command: `./tetra test examples`; JSON report gate.
- Current evidence: fresh full gate steps 13 and 14 passed.
- Result: pass.

### Docs Generation

- Required artifact or command: `./tetra doc examples`; generated API docs gate.
- Current evidence: fresh full gate steps 21 and 22 passed.
- Result: pass.

### Project And Workspace

- Required artifact or command: CLI contract plus Go/tool validators.
- Current evidence: `docs/spec/cli_contracts.md` documents project and workspace
  commands.
- Current evidence: full gate step 01 runs all CLI/tool Go tests.
- Result: pass.

### Validators

- Required artifact or command: `bash scripts/ci/test-all.sh`.
- Current evidence: fresh full gate
  `reports/test-all-20260506-192335/summary.md` passed all 27 steps.
- Current evidence: that includes docs, smoke, target, LSP, API-docs, Eco, and
  cross-target checks.
- Result: pass.

### Stable Stdlib Modules

- Required artifact or command: `lib/core/*.tetra`.
- Current evidence: files exist for async, capability, collections, crypto,
  filesystem, io, math, memory, networking, serialization, slices, strings,
  sync, testing, and time.
- Current evidence: the named goal modules are present.
- Result: pass.

### Stable Stdlib Examples

- Required artifact or command: `examples/core_*_smoke.tetra`.
- Current evidence: smoke examples exist for every stable core module, including
  filesystem, networking, and crypto.
- Result: pass.

### Stdlib API Docs

- Required artifact or command:
  `./tetra doc lib/core lib/experimental examples/core_*_smoke.tetra`;
  `validate-api-docs`.
- Current evidence: fresh stdlib API docs validation passes.
- Current evidence: full gate step 22 also validates generated API docs.
- Result: pass.

### Stdlib Feature Registry

- Required artifact or command: `stdlib.core-current`;
  `stdlib.experimental-mirrors`.
- Current evidence: `validate-tooling-stdlib-readiness` accepts the fresh
  `./tetra features --format=json` report, the stdlib spec, and CLI contract.
- Result: pass.

### Filesystem Slice

- Required artifact or command: runtime ABI docs, runtime symbols,
  compiler/runtime tests, smoke.
- Current evidence: `lib.core.filesystem.exists` is capability-gated and
  host-backed on linux-x64 through `__tetra_fs_exists`.
- Current evidence: unsupported targets report explicit diagnostics.
- Current evidence: targeted filesystem tests and the full gate pass.
- Result: pass.

### Networking Surface

- Required artifact or command:
  `docs/spec/stdlib.md`; `docs/user/standard_library_guide.md`; smoke.
- Current evidence: networking is a stable endpoint policy-helper surface.
- Current evidence: it does not claim socket, DNS, or HTTP transport behavior.
- Result: pass.

### Crypto Surface

- Required artifact or command:
  `docs/spec/stdlib.md`; `docs/user/standard_library_guide.md`; smoke.
- Current evidence: crypto is a stable interface-helper surface with
  deterministic helper behavior.
- Current evidence: it does not claim encryption, authentication, entropy, or
  reviewed algorithm coverage.
- Result: pass.

### Compatibility Policy

- Required artifact or command:
  `docs/spec/stdlib_naming_versioning.md`; experimental mirrors.
- Current evidence: stable names and compatibility mirror expectations are
  documented.
- Current evidence: `lib.experimental.*` mirrors forward stable callers to
  `lib.core.*`.
- Result: pass.

### No Mock Or Filler Claims

- Required artifact or command: `validate-tooling-stdlib-readiness`; text scan.
- Current evidence: the readiness validator rejects `placeholder` and `mock` in
  stdlib production evidence.
- Current evidence: current stdlib docs, compiler features, stable modules, and
  core smokes pass the scan.
- Result: pass.

### Graphify Evidence

- Required artifact or command: `graphify update .`.
- Current evidence: rebuilt `graphify-out/graph.json` and `GRAPH_REPORT.md`
  after the final Go-code edit.
- Result: pass.

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
