# Contributor Guide: Compiler, Tests, And Release Work

Status: contributor guide for the current development baseline.

## Compiler Pipeline

The compiler pipeline is organized around frontend parsing, semantic checking,
lowering, target/backend emission, and CLI/tooling wrappers. When changing a
stage, inspect the real package and tests first; do not infer behavior from old
roadmaps.

## Adding Syntax

1. Update the Flow syntax spec before promising support.
2. Add parser tests for accepted and rejected forms.
3. Add formatter coverage when the syntax is printable.
4. Add user-facing diagnostics for invalid forms.
5. Wire release evidence into the checklist only after tests pass.

## Adding A Builtin Or Stable Stdlib API

1. Document the API in the relevant `docs/spec/` file.
2. Add examples or doctests where the existing docs verifier expects them.
3. Update generated API documentation only through the generator.
4. Run docs verification and the focused package tests.

## Adding A Backend Or Target Smoke

1. Document target status and limitations.
2. Add build-only smoke first.
3. Add runner/browser automation only when the runner is real in CI.
4. Validate reports with `tools/cmd/smoke-report-to-checklist --validate-only`.

## Test Strategy

Use the narrowest test that proves the change, then run the nearby gate:

```sh
go test ./compiler/... ./cli/... ./tools/... -count=1
bash scripts/test_all.sh --full --keep-going
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

For release scripts, keep tests in `tools/scriptstest` aligned with the shell
contract.

## Generated Artifacts Policy

Generated artifacts are either tracked deliberately or written under a report
directory. The detailed policy is `docs/release/artifact_policy.md`.

## Release Process

The release process is documented in `docs/release/rc_process.md`.
`scripts/release_v1_0_gate.sh` is expected to fail on `v0.1.0`; do not bypass
the version preflight or mark checklist items complete without evidence.
