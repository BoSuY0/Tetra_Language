# Tetra 0.6.x Stabilization Roadmap

> Historical checkpoint. This roadmap belongs to the older v0.6 stabilization line and is superseded
> by `docs/spec/flow/v1_scope.md` and `docs/checklists/v1_0_release_gate.md`. Public release truth
> for this branch lives in `docs/spec/core/current_supported_surface.md` (`v0.2.0`). The v1.0 scope
> remains a future contract.

Tetra 0.6.x was a stabilization line for the then-current Usable Alpha surface. The goal is not to
add another large language feature, but to make the existing compiler, runtime, tooling, docs, and
local Eco flows repeatably testable.

## Historical Baseline

- `tetra version` reports `v0.6.0`.
- The v0.6 release gate is captured by `scripts/release/v0_6/gate.sh`.
- The 0.6.x stabilization wrapper is `scripts/ci/test-all.sh`.
- Reports from `scripts/ci/test-all.sh` contain per-step logs, `summary.md`, and `summary.json`;
  each JSON step includes `name`, `status`, `duration_seconds`, `exit_code`, `command`, and `log`.
  The JSON envelope also includes top-level `step_count` and `failed_count` fields for CI consumers.
  The summary is validated against its log artifacts before emission; passing runs require
  validation, while failing runs preserve the original failure report if the summary validator
  itself fails. Step names and log paths are expected to be unique so CI consumers can key reports
  deterministically.
- The point-release plan is tracked in `docs/roadmaps/early/roadmap_0_6_1_to_0_6_3.md`.

## Test Commands

Fast local iteration:

```sh
bash scripts/ci/test-all.sh --quick
```

Full stabilization gate:

```sh
bash scripts/ci/test-all.sh --full
```

Collect every selected failure before exiting:

```sh
bash scripts/ci/test-all.sh --full --keep-going
```

Emit only machine-readable summary JSON:

```sh
bash scripts/ci/test-all.sh --full --json-only
```

Canonical v0.6.0 release compatibility gate:

```sh
bash scripts/release/v0_6/gate.sh
```

The full wrapper currently covers:

- Go package tests for compiler, CLI, and tools.
- Repository test script and bootstrap.
- Version-prefix validation for the `v0.6.x` line.
- Formatter coverage for `examples` and `lib`.
- `tetra test` text reports and JSON reports validated through `tools/cmd/validate-test-report`.
- Native Linux smoke execution.
- Docs manifest schema validation, diff, and docs verification.
- LSP `--stdio-smoke` JSON validation and framed stdio JSON-RPC transcript validation.
- Generated API docs smoke with Markdown shape validation.
- Local Eco graph with lock JSON validation, project bundle unpack validation, and vault store
  validation.
- Native host smoke plus build-only smoke for `linux-x64`, `macos-x64`, and `windows-x64`.
- Native and cross-target smoke JSON report validation for aggregate counts and per-case metadata
  through `tools/cmd/smoke-report-to-checklist --validate-only`.

## 0.6.1 Target: Test Envelope Hardening

- Keep `scripts/ci/test-all.sh --full` green.
- Keep `scripts/ci/test-all.sh --quick --json-only` valid for machine consumers.
- Keep per-step JSON `exit_code` stable so CI can distinguish command failures without scraping
  logs.
- Preserve machine-readable failure summaries even when secondary report validation tooling is
  broken.
- Use `--keep-going` when collecting stabilization failure envelopes.
- Add focused regression tests for any failure found in current examples, formatter output, LSP
  diagnostics, Eco packing, or docs verification.
- Prefer fixing bugs under existing feature boundaries over adding new syntax.
- Do not change the public language surface unless a stabilization bug requires a diagnostic
  clarification.

## 0.6.2 Target: Negative Coverage

- Expand semantic negative tests around optionals, typed errors, ownership markers, effects,
  protocols, extensions, async/task calls, and Eco manifests.
- Add JSON diagnostic shape snapshots for common parser/semantic failures.
- Ensure each planned-feature diagnostic still names the planned feature instead of surfacing a raw
  parser error.

## 0.6.3 Target: Cross-Platform Confidence

- Keep build-only smoke green for all supported x64 targets.
- Add object-format assertions where existing test helpers already support ELF, PE, Mach-O, and
  TOBJ.
- Run native smoke only when host and target match.
- Keep self-host actors and builtin actor fallback both covered.

## Release Rules For 0.6.x

- A point release cannot be cut while `scripts/ci/test-all.sh --full` fails.
- A point release cannot be cut while `scripts/release/v0_6/gate.sh` fails, unless the version has
  intentionally advanced and the release gate was updated in the same patch.
- Generated docs manifest changes must be intentional and verified.
- Local reports under `reports/` are ignored artifacts, not source.
- New features belong in the next minor roadmap unless they directly reduce instability in the
  current 0.6 surface.
