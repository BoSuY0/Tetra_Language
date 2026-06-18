# Contract-Driven Release Validation Refactor Completion Audit

**Date:** 2026-06-16.
**Status:** technical acceptance proven locally; ready for `/goal` completion
under the current subagent policy, which requires `worker` (`gpt-5.5 xhigh`)
with disjoint write scopes and no longer requires `fork_context=true`.
**Plan:** `docs/plan/2026-06-16-contract-driven-release-validation-refactor-plan.md`.

## Current Evidence

Final current Surface dispatch report:

- `reports/contract-refactor-surface-release-complete-20260616-183817/`

The release gate was executed through the contract-backed normal path:

```sh
GOTELEMETRY=off \
GOCACHE="$PWD/.cache/go-build-contract-refactor-complete" \
GOTMPDIR="$PWD/.cache/go-tmp-contract-refactor-complete" \
timeout 45m bash scripts/release/surface/release-gate.sh \
  --report-dir reports/contract-refactor-surface-release-complete-20260616-183817
```

Result: `exit=0`.

The current summary includes:

- `"schema": "tetra.surface.release.v1"`
- `"crash_reporting": "surface-crash-report-v1"`
- `"block_system_gate": "tetra.surface.block-system.gate.v1"`
- `"morph_gate": "tetra.surface.morph.gate.v1"`
- `"artifact_hashes_validated": true`

## Acceptance Matrix

| Acceptance item | Current status | Evidence |
| --- | --- | --- |
| Surface `crash_reporting` is produced, validated, included in release summary, required by release-state validation, and covered by tests. | Proven locally | `surface-crash-report.json`, `surface-release-summary.json`, `tools/cmd/validate-surface-crash-report`, `tools/validators/surface/report_release_test.go`, `tools/validators/surface/crash_report_test.go`, `tools/scriptstest/release_surface_gate_wiring_test.go`. |
| At least one Surface release gate executes from a machine-readable contract. | Proven locally | `tools/cmd/run-gate` now dispatches normal execution; `scripts/release/surface/release-gate.sh` delegates normal runs through `run-gate --contract scripts/release/surface/contracts/surface-release-v1.json`; final dispatch gate report above passed. |
| CI and local scripts use the same contract-backed entrypoint. | Proven locally for repository wiring | `.github/workflows/ci.yml`, `.github/workflows/release-packages.yml`, `scripts/release/surface/release-gate.sh`, `scripts/release/surface/morph-gate.sh`, and post-v0.4 RAM/Memory/Actor gate scripts reference the contract-backed scripts and/or `run-gate` contract preflight. Remote CI execution is not claimed. |
| Artifact hash validation remains mandatory. | Proven locally | Surface contract requires `artifact_hashes.enabled=true`, `required=true`, algorithm `sha256`; final report `artifact-hashes.json` validates with `validate-artifact-hashes`. |
| Host preconditions are explicit and cannot silently skip required release evidence. | Proven locally | Gate contracts include `host_preconditions`; Surface final gate failed on a Chromium probe when `TMPDIR` was incorrectly forced, then passed after the environment was corrected rather than silently skipping evidence. |
| Script tests validate contract semantics rather than only long bash command strings. | Improved and covered | Script tests load contracts through `tools/internal/gatecontract`, compare contract required reports/validators/CI artifacts, and verify dispatcher guard semantics. Some string/order assertions intentionally remain for shell wiring invariants. |
| Surface validator monolith has begun moving into schema-owned files without changing public validator behavior. | Proven locally | Surface validation now includes schema-owned files such as `release_summary.go`, `crash_report.go`, and focused validation files while public validator tests pass. |
| Focused validation passes. | Proven locally | Commands listed below passed in the current worktree. |
| Any broad validation not run is listed as not verified. | Explicitly listed | Remote CI and full repository-wide broad validation are not claimed here. |

## Fresh Verification Commands

These commands were re-run after the dispatcher implementation:

```sh
GOTELEMETRY=off \
GOCACHE="$PWD/.cache/go-build-contract-refactor-complete" \
GOTMPDIR="$PWD/.cache/go-tmp-contract-refactor-complete" \
go test \
  ./tools/internal/gatecontract \
  ./tools/internal/reportdir \
  ./tools/internal/artifacts \
  ./tools/cmd/run-gate \
  ./tools/scriptstest \
  ./tools/cmd/validate-manifest \
  ./tools/cmd/verify-docs \
  ./tools/validators/surface \
  ./tools/cmd/validate-surface-runtime \
  ./tools/cmd/validate-surface-release-state \
  ./tools/cmd/validate-surface-crash-report \
  ./tools/cmd/surface-runtime-smoke \
  ./tools/cmd/surface-visual-diff \
  -count=1
```

Result:

```text
ok  	tetra_language/tools/internal/gatecontract
ok  	tetra_language/tools/internal/reportdir
ok  	tetra_language/tools/internal/artifacts
ok  	tetra_language/tools/cmd/run-gate
ok  	tetra_language/tools/scriptstest
ok  	tetra_language/tools/cmd/validate-manifest
ok  	tetra_language/tools/cmd/verify-docs
ok  	tetra_language/tools/validators/surface
ok  	tetra_language/tools/cmd/validate-surface-runtime
ok  	tetra_language/tools/cmd/validate-surface-release-state
?   	tetra_language/tools/cmd/validate-surface-crash-report	[no test files]
ok  	tetra_language/tools/cmd/surface-runtime-smoke
ok  	tetra_language/tools/cmd/surface-visual-diff
```

```sh
report_dir=reports/contract-refactor-surface-release-complete-20260616-183817
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-complete" GOTMPDIR="$PWD/.cache/go-tmp-contract-refactor-complete" go run ./tools/cmd/validate-surface-crash-report --report "$report_dir/surface-crash-report.json"
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-complete" GOTMPDIR="$PWD/.cache/go-tmp-contract-refactor-complete" go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-complete" GOTMPDIR="$PWD/.cache/go-tmp-contract-refactor-complete" go run ./tools/cmd/validate-surface-runtime --report "$report_dir/surface-release-summary.json" --release surface-v1
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-complete" GOTMPDIR="$PWD/.cache/go-tmp-contract-refactor-complete" go run ./tools/cmd/validate-surface-release-state --report-dir "$report_dir" --expected-status current --scope surface-v1-linux-web --manifest docs/generated/manifest.json
```

Result: all commands exited `0`.

```sh
graphify update .
```

Result: `2083 files`, `31236 nodes`, `76481 edges`, `1719 communities`.

## Subagent Policy

The current active objective permits code-editing subagents when they are:

- `agent_type=worker`
- `gpt-5.5`
- reasoning `xhigh`
- assigned disjoint write scopes

The bounded dispatcher implementation was delegated to `agent_type=worker`,
whose tool-defined role uses `gpt-5.5` with `xhigh` reasoning. The worker write
scope was limited to:

- `tools/cmd/run-gate/main.go`
- `tools/cmd/run-gate/main_test.go`
- `scripts/release/surface/release-gate.sh`
- `tools/scriptstest/release_surface_gate_wiring_test.go`

The parent controller inspected, hardened, integrated, and re-verified that
patch. No current subagent-policy blocker remains.

## Not Claimed

- Remote CI execution is not claimed.
- A clean worktree is not claimed.
- The root `GOAL.md` is not updated by this audit because the current dirty
  file describes a different Morph rendered beauty goal.
