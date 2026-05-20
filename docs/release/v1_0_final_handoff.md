# Tetra v1.0 Final Handoff

Status: evidence schema for the future `v1.0.0` release. Do not mark this
handoff complete until `scripts/release/v1_0/gate.sh` passes on the exact
release branch state and `docs/checklists/v1_0_release_gate.md` has no unchecked
mandatory rows.

Checklist source of truth: `docs/checklists/v1_0_release_gate.md`.
Scope contract: `docs/spec/v1_scope.md`.
Cut guide: `docs/release/v1_0_release_cut_guide.md`.
Gate command: `bash scripts/release/v1_0/gate.sh --report-dir REPORT_DIR`.
Expected artifact id: `tetra.release.v1_0.gate-report.v1`.

## Release State

Every final handoff must record these concrete values from the same branch
state:

| Field | Required value |
| --- | --- |
| Date | ISO release-candidate date. |
| Branch | Release branch name. |
| Commit | Output of `git rev-parse HEAD`. |
| Version | Output of `./tetra version`, expected `v1.0.0`. |
| Short alias version | Output of `./t version`, identical to `./tetra version`. |
| Release archive path | Final report directory. |
| Gate summary | `REPORT_DIR/summary.json`. |
| Security signoff | `REPORT_DIR/artifacts/security-review.md`. |
| Release-state audit | `REPORT_DIR/artifacts/release-state.txt`. |
| Artifact hash manifest | `REPORT_DIR/artifacts/artifact-hashes.json`. |

## Fresh Verification

All completed rows must cite evidence from the same report directory and commit.
Do not copy rows from older candidate runs.

| Check | Command or gate step | Evidence path | Required result |
| --- | --- | --- | --- |
| Bootstrap binaries | `bash scripts/dev/bootstrap.sh` | `REPORT_DIR/logs/01-bootstrap-tetra-binaries.log` | Pass with exit code 0. |
| Version preflight | `./tetra version`; `./t version`; gate version preflight | `REPORT_DIR/logs/version-preflight.log` | Pass and exact version `v1.0.0`. |
| Final v1 gate | `bash scripts/release/v1_0/gate.sh --report-dir REPORT_DIR` | `REPORT_DIR/summary.json`; `REPORT_DIR/summary.md` | Pass with `failed_count: 0`. |
| Go workspace tests | `go test ./compiler/... ./cli/... ./tools/... -count=1` | `REPORT_DIR/logs/go-test-all.log` | Pass with exit code 0. |
| v1 language scope | Commands from `docs/spec/v1_scope.md` mandatory language table | `REPORT_DIR/logs/scope-language.log` | Pass or explicit blocker list. |
| Formatter and Flow gates | `./tetra fmt --check examples lib __rt compiler/selfhostrt`; `go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt` | `REPORT_DIR/logs/formatter.log`; `REPORT_DIR/logs/flow-only.log` | Pass with exit code 0. |
| CLI and tooling | `go test ./cli/... -count=1`; `go test ./tools/... -count=1`; JSON report validators | `REPORT_DIR/logs/cli-tools.log`; `REPORT_DIR/artifacts/*.json` | Pass with exit code 0. |
| Docs and API docs | Docs manifest validation, `verify-docs`, tetra docs, API diff | `REPORT_DIR/logs/docs.log`; `REPORT_DIR/artifacts/api-diff/api-diff.json` | Pass with exit code 0. |
| LSP baseline | LSP stdio and smoke validator tests | `REPORT_DIR/logs/lsp.log` | Pass with exit code 0. |
| Eco lifecycle | Eco command surface and package lifecycle validator evidence | `REPORT_DIR/logs/eco.log` | Pass with exit code 0. |
| Target matrix | Linux host, macOS/Windows build-only, WASI, and web smoke | `REPORT_DIR/artifacts/*smoke*.json`; smoke logs | Pass or explicit platform blocker list. |
| Security signoff | `bash scripts/release/v1_0/security-review.sh --signoff REPORT_DIR/artifacts/security-review.md` | `REPORT_DIR/artifacts/security-review.md`; security log | Approved or blocked. |
| Performance regression | `go run ./tools/cmd/validate-performance-report --report REPORT_DIR/artifacts/performance-regression.json` | `REPORT_DIR/artifacts/performance-regression.json` | Pass with metric count. |
| Binary size thresholds | `bash scripts/release/v1_0/binary-size.sh --report REPORT_DIR/artifacts/binary-size-thresholds.json` | `REPORT_DIR/artifacts/binary-size-thresholds.json` | Pass with fail count 0. |
| Reproducible build proof | `bash scripts/release/v1_0/reproducible-build.sh --report REPORT_DIR/artifacts/reproducible-build.json` | `REPORT_DIR/artifacts/reproducible-build.json` | Pass with matched count. |
| Release state and artifact hashes | `validate-release-state`; `validate-artifact-hashes` | `REPORT_DIR/artifacts/release-state.txt`; `REPORT_DIR/artifacts/artifact-hashes.json` | Pass with no missing artifacts. |
| Diff hygiene | `git diff --check` | Terminal transcript or gate handoff note | Pass with exit code 0. |

## Backend Summary

Record a backend-focused summary from the same report directory as the final
gate. The summary is incomplete unless every field has a concrete value:

| Field | Required evidence |
| --- | --- |
| Backend summary note | `REPORT_DIR/artifacts/backend-summary.md` or checked-in audit note path. |
| Commit and version | `git rev-parse HEAD`; `./tetra version`. |
| Native target reports | Linux run report plus macOS/Windows build-only reports. |
| WASM artifact/import preflight reports | `wasm32-wasi-artifact-smoke.json`; `wasm32-web-artifact-smoke.json`. |
| WASI runtime report | `wasi-smoke.json`, including runner name, total, passed, and failed counts. |
| Web UI runtime report | `web-ui-smoke.json`, including automation, `ui_schema`, bundle path, module path, and DOM snapshot path. |
| UI/native boundary decision | One sentence confirming native widgets and runtime event dispatch are post-v1 unless separately promoted. |
| Residual backend risks | Explicit list, or `None`, with owner/date for each accepted risk. |

## Scope Signoff

| v1 scope area | Evidence summary | Residual risk | Signoff |
| --- | --- | --- | --- |
| Flow syntax and parser diagnostics | Commands, logs, and artifacts from the final report directory. | Named risk or `None`. | Owner and date. |
| Type, generic, protocol, extension, module contracts | Commands, logs, and artifacts from the final report directory. | Named risk or `None`. | Owner and date. |
| Ownership, lifetimes, islands, actors, tasks | Commands, logs, and artifacts from the final report directory. | Named risk or `None`. | Owner and date. |
| Effects, capabilities, unsafe, privacy, consent, budgets | Commands, logs, and artifacts from the final report directory. | Named risk or `None`. | Owner and date. |
| Async/task/actor runtime MVP | Commands, logs, and artifacts from the final report directory. | Named risk or `None`. | Owner and date. |
| Runtime ABI, TOBJ, and linking | Commands, logs, and artifacts from the final report directory. | Named risk or `None`. | Owner and date. |
| UI metadata and accessibility surface | Commands, logs, and artifacts from the final report directory. | Named risk or `None`. | Owner and date. |
| CLI, tooling, LSP, docs, and Eco lifecycle | Commands, logs, and artifacts from the final report directory. | Named risk or `None`. | Owner and date. |
| Target matrix | Commands, logs, and artifacts from the final report directory. | Named risk or `None`. | Owner and date. |

## Known Issues And Residual Risks

- Known issues artifact: `REPORT_DIR/artifacts/known_issues.md`.
- Accepted non-blockers: explicit list, or `None`.
- Release blockers: explicit list, or `None` before final approval.
- Post-v1 deferred features checked against `docs/spec/v1_scope.md`: `yes` or
  `no` with blocker explanation.

## Release Decision

The final decision must be one of:

- `approved for v1.0.0 release`
- `blocked`

The approver, approval date, commit, version, report directory, artifacts, and
command results must all match this handoff. Approval is invalid if any required
evidence comes from a different branch state.
