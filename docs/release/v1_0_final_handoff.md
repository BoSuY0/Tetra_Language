# Tetra v1.0 Final Handoff

Status: evidence template for the future `v1.0.0` release. Do not mark this
handoff complete until `scripts/release_v1_0_gate.sh` passes on the exact
release branch state and `docs/checklists/v1_0_release_gate.md` has no unchecked
mandatory rows.

Checklist source of truth: `docs/checklists/v1_0_release_gate.md`.
Scope contract: `docs/spec/v1_scope.md`.
Cut guide: `docs/release/v1_0_release_cut_guide.md`.
Gate command: `bash scripts/release_v1_0_gate.sh --report-dir <report-dir>`.
Expected artifact id: `tetra.release.v1_0.gate-report.v1`.

## Release State

- Date: `<YYYY-MM-DD>`.
- Branch: `<release branch>`.
- Commit: `<git rev-parse HEAD>`.
- Version: `<./tetra version>`; expected `v1.0.0`.
- Short alias version: `<./t version>`; must match `./tetra version`.
- Release archive path: `<report-dir>`.
- Gate summary: `<report-dir>/summary.json`.
- Security signoff: `<report-dir>/artifacts/security-review.md`.
- Release-state audit: `<report-dir>/artifacts/release-state.txt`.
- Artifact hash manifest: `<report-dir>/artifacts/artifact-hashes.json`.

## Fresh Verification

All completed rows must cite evidence from the same `<report-dir>` and commit.
Do not copy rows from older candidate runs.

| Check | Command or gate step | Evidence path | Result |
| --- | --- | --- | --- |
| Bootstrap binaries | `bash scripts/bootstrap.sh` | `<report-dir>/logs/01-bootstrap-tetra-binaries.log` | `<pass/fail, exit code>` |
| Version preflight | `./tetra version`; `./t version`; gate version preflight | `<report-dir>/logs/*version*.log` | `<pass/fail, exact version>` |
| Final v1 gate | `bash scripts/release_v1_0_gate.sh --report-dir <report-dir>` | `<report-dir>/summary.json`; `<report-dir>/summary.md` | `<pass/fail, failed_count, step_count>` |
| Go workspace tests | `go test ./compiler/... ./cli/... ./tools/... -count=1` | `<report-dir>/logs/*go-test*.log` | `<pass/fail, exit code>` |
| v1 language scope | Commands from `docs/spec/v1_scope.md` mandatory language table | `<report-dir>/logs/<scope logs>` | `<pass/fail, blockers>` |
| Formatter and Flow gates | `./tetra fmt --check examples lib __rt compiler/selfhostrt`; `go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt` | `<report-dir>/logs/*formatter*`; `<report-dir>/logs/*flow-only*` | `<pass/fail, exit code>` |
| CLI and tooling | `go test ./cli/... -count=1`; `go test ./tools/... -count=1`; JSON report validators | `<report-dir>/logs/<cli-tools logs>`; `<report-dir>/artifacts/*.json` | `<pass/fail, exit code>` |
| Docs and API docs | docs manifest validation, `verify-docs`, tetra docs, API diff | `<report-dir>/logs/*docs*`; `<report-dir>/artifacts/api-diff/api-diff.json` | `<pass/fail, exit code>` |
| LSP baseline | LSP stdio and smoke validator tests | `<report-dir>/logs/<lsp logs>` | `<pass/fail, exit code>` |
| Eco lifecycle | Eco command surface and package lifecycle validator evidence | `<report-dir>/logs/<eco logs>` | `<pass/fail, exit code>` |
| Target matrix | linux host, macOS/Windows build-only, WASI, and web smoke | `<report-dir>/artifacts/*smoke*.json`; smoke logs | `<pass/fail, blockers>` |
| Security signoff | `bash scripts/release_v1_0_security_review.sh --signoff <report-dir>/artifacts/security-review.md` | `<report-dir>/artifacts/security-review.md`; security log | `<approved/blocked>` |
| Performance regression | `go run ./tools/cmd/validate-performance-report --report <report-dir>/artifacts/performance-regression.json` | `<report-dir>/artifacts/performance-regression.json` | `<pass/fail, metric_count>` |
| Binary size thresholds | `bash scripts/release_v1_0_binary_size.sh --report <report-dir>/artifacts/binary-size-thresholds.json` | `<report-dir>/artifacts/binary-size-thresholds.json` | `<pass/fail, fail_count>` |
| Reproducible build proof | `bash scripts/release_v1_0_repro.sh --report <report-dir>/artifacts/reproducible-build.json` | `<report-dir>/artifacts/reproducible-build.json` | `<pass/fail, matched_count>` |
| Release state and artifact hashes | `validate-release-state`; `validate-artifact-hashes` | `<report-dir>/artifacts/release-state.txt`; `<report-dir>/artifacts/artifact-hashes.json` | `<pass/fail, missing artifacts>` |
| Diff hygiene | `git diff --check` | `<terminal transcript or gate handoff note>` | `<pass/fail, exit code>` |

## Scope Signoff

| v1 scope area | Evidence summary | Residual risk | Signoff |
| --- | --- | --- | --- |
| Flow syntax and parser diagnostics | `<commands, logs, artifacts>` | `<risk or None>` | `<owner/date>` |
| Type, generic, protocol, extension, module contracts | `<commands, logs, artifacts>` | `<risk or None>` | `<owner/date>` |
| Ownership, lifetimes, islands, actors, tasks | `<commands, logs, artifacts>` | `<risk or None>` | `<owner/date>` |
| Effects, capabilities, unsafe, privacy, consent, budgets | `<commands, logs, artifacts>` | `<risk or None>` | `<owner/date>` |
| Async/task/actor runtime MVP | `<commands, logs, artifacts>` | `<risk or None>` | `<owner/date>` |
| Runtime ABI, TOBJ, and linking | `<commands, logs, artifacts>` | `<risk or None>` | `<owner/date>` |
| UI metadata and accessibility surface | `<commands, logs, artifacts>` | `<risk or None>` | `<owner/date>` |
| CLI, tooling, LSP, docs, and Eco lifecycle | `<commands, logs, artifacts>` | `<risk or None>` | `<owner/date>` |
| Target matrix | `<commands, logs, artifacts>` | `<risk or None>` | `<owner/date>` |

## Known Issues And Residual Risks

- Known issues artifact: `<report-dir>/artifacts/known_issues.md`.
- Accepted non-blockers: `<list or None>`.
- Release blockers: `<list or None; must be None before final approval>`.
- Post-v1 deferred features checked against `docs/spec/v1_scope.md`: `<yes/no>`.

## Release Decision

Decision: `<approved for v1.0.0 release | blocked>`.

Approver: `<name/contact>`.
Date: `<YYYY-MM-DD>`.

Approval is valid only for the commit, version, report directory, artifacts,
and command results cited in this handoff.
