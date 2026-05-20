# No-Wrapper Project Structure

This document is the canonical target structure for the direct architecture
refactor mode: migrate references to domain-owned paths, then remove legacy
root-level entrypoints instead of preserving compatibility wrappers.

## Target Tree

```text
Tetra_Language/
  AGENTS.md
  README.md
  go.mod
  go.sum
  go.work

  cli/
    cmd/
      tetra/
        main.go
        *_command.go
        *_test.go
    internal/
    testkit/
    tests/
      commands/
      eco/
      lsp/
      smoke/
      fixtures/

  compiler/
    api.go
    compiler.go
    diagnostics.go
    format.go
    lsp.go
    manifest.go
    target/
    selfhostrt/
    internal/
      actorsrt/
      backend/
        native_shell/
        wasm32_wasi/
        wasm32_web/
        x64/
        x64abi/
        x64core/
        x64obj/
      deps/
      format/
      frontend/
      ir/
      lower/
      semantics/
      testkit/
    tests/
      backend/
      callables/
      frontend/
      lowering/
      ownership/
      runtime/
      safety/
      semantics/
    testdata/
      callables/
      frontend/
      lowering/
      ownership/
      safety/
      semantics/

  tools/
    cmd/
      dump-project/
      smoke-report-to-checklist/
      validate-*/
      verify-docs/
    validators/
      docs/
      eco/
      release/
      shared/
      wasm/
      workspace/
    release/
      gates/
      security/
      smoke/
    testkit/
    scriptstest/
      fixtures/
      v0_1_1/
      v0_1_2/
      v0_1_3/
      v0_2_0/
      v0_3_0/
      v0_4_0/
      v1_0_0/

  scripts/
    ci/
      test.sh
      test-all.sh
    dev/
      bootstrap.sh
      dump-project.sh
      format.sh
      fuzz-nightly.sh
    release/
      shared/
      smoke/
      v0_1_1/
        gate.sh
      v0_1_2/
        gate.sh
      v0_1_3/
        gate.sh
      v0_2_0/
        gate.sh
      v0_3_0/
        gate.sh
        security-review.sh
      v0_4_0/
        gate.sh
        security-review.sh
      v0_5/
        gate.sh
      v0_6/
        gate.sh
      v1_0/
        api-diff.sh
        binary-size.sh
        gate.sh
        reproducible-build.sh
        security-review.sh
        wasi-smoke.sh
        web-smoke.sh
    tools/
      api_diff_report.mjs

  docs/
    architecture/
      no_wrapper_project_structure.md
      project_structure.md
    audits/
    baselines/
    checklists/
    contributing/
    generated/
      v1_0/
    performance/
    plans/
    release/
    release-notes/
    schemas/
    spec/
    superpowers/
    testing/
    user/

  examples/
    smoke/
      core/
      ownership/
      tasks/
      ui/
      wasm/
    projects/
      dogfood_wasi/
      dogfood_web_ui/
      hello_t4/

  lib/
    core/
    experimental/

  reports/
    plan250/
    release-*/
    test-all-*/

  graphify-out/
    GRAPH_REPORT.md
    graph.json
    wiki/

  .github/
    workflows/
      ci.yml
```

## Entry Point Policy

- Final-state root-level `scripts/*.sh` compatibility wrappers are not allowed.
- Migrate every caller to canonical paths before deleting a legacy entrypoint.
- Use these canonical examples:
  - `scripts/ci/test.sh`
  - `scripts/ci/test-all.sh`
  - `scripts/dev/bootstrap.sh`
  - `scripts/dev/format.sh`
  - `scripts/release/v1_0/gate.sh`
  - `scripts/release/v1_0/api-diff.sh`
  - `scripts/release/v1_0/binary-size.sh`
  - `scripts/release/v1_0/reproducible-build.sh`
- A legacy path may remain only if it is explicitly documented as an intentional
  exception with a reason, owner, and verification command.

## Slice Rules

- Work in small behavior-preserving slices.
- Before removing a legacy entrypoint, update CI, docs, tests, release gates,
  generated references, and tool fixtures that still invoke it.
- Each slice must include a focused guard that proves callers use the canonical
  path or that the old path has been removed intentionally.
- Run relevant `gofmt`, focused tests, package tests, shell syntax checks,
  `git diff --check`, and `graphify update .`.
- Do not claim the migration complete until every old flat or mixed zone is
  either moved into the target tree or documented as an intentional exception.
