# Tetra v1.0 Final Handoff

Status: evidence schema for the future `v1.0.0` release. Do not mark this
handoff complete until `scripts/release/v1_0/gate.sh` passes on the exact
release branch state and `docs/checklists/v1_0_release_gate.md` has no
unchecked mandatory rows.

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
| Security signoff detached hash | `REPORT_DIR/artifacts/security-review.md.sha256`. |
| Release-state audit | `REPORT_DIR/artifacts/release-state.txt`. |
| Artifact hash manifest | `REPORT_DIR/artifacts/artifact-hashes.json`. |

## Fresh Verification

All completed rows must cite evidence from the same report directory and
commit. Do not copy rows from older candidate runs.

### Bootstrap binaries

- Command:

```sh
bash scripts/dev/bootstrap.sh
```

- Evidence path: `REPORT_DIR/logs/01-bootstrap-tetra-binaries.log`.
- Required result: pass with exit code 0.

### Version preflight

- Commands:

```sh
./tetra version
./t version
```

- Gate step: gate version preflight.
- Evidence paths:
  - `REPORT_DIR/logs/02-version-preflight-v1-0-0-required.log`
  - `REPORT_DIR/logs/03-short-alias-version-parity.log`
- Required result: pass and exact version `v1.0.0`.

### Final v1 gate

- Command:

```sh
bash scripts/release/v1_0/gate.sh \
  --report-dir REPORT_DIR
```

- Evidence paths:
  - `REPORT_DIR/summary.json`
  - `REPORT_DIR/summary.md`
- Required result: pass with `failed_count: 0`.

### Go workspace tests

- Command:

```sh
go test ./compiler/... ./cli/... ./tools/... -count=1
```

- Evidence path: `REPORT_DIR/logs/go-test-all.log`.
- Required result: pass with exit code 0.

### v1 language scope

- Command source:
  commands from `docs/spec/v1_scope.md` mandatory language table.
- Evidence path: `REPORT_DIR/logs/scope-language.log`.
- Required result: pass or explicit blocker list.

### Formatter and Flow gates

- Commands:

```sh
./tetra fmt --check examples lib __rt compiler/selfhostrt
go run ./tools/cmd/validate-flow-only \
  examples \
  lib \
  __rt \
  compiler/selfhostrt
```

- Evidence paths:
  - `REPORT_DIR/logs/formatter.log`
  - `REPORT_DIR/logs/flow-only.log`
- Required result: pass with exit code 0.

### CLI and tooling

- Commands:

```sh
go test ./cli/... -count=1
go test ./tools/... -count=1
```

- Additional gate step: JSON report validators.
- Evidence paths:
  - `REPORT_DIR/logs/cli-tools.log`
  - `REPORT_DIR/artifacts/*.json`
- Required result: pass with exit code 0.

### Docs and API docs

- Gate steps:
  docs manifest validation, `verify-docs`, tetra docs, API diff.
- Evidence paths:
  - `REPORT_DIR/logs/docs.log`
  - `REPORT_DIR/artifacts/api-diff/api-diff.json`
- Required result: pass with exit code 0.

### LSP baseline

- Gate steps: LSP stdio and smoke validator tests.
- Evidence path: `REPORT_DIR/logs/lsp.log`.
- Required result: pass with exit code 0.

### Eco lifecycle

- Gate steps: Eco command surface and package lifecycle validator evidence.
- Evidence path: `REPORT_DIR/logs/eco.log`.
- Required result: pass with exit code 0.

### Target matrix

- Gate steps:
  Linux host, macOS/Windows build-only, WASI, and web smoke.
- Evidence paths:
  - `REPORT_DIR/artifacts/*smoke*.json`
  - smoke logs
- Required result: pass or explicit platform blocker list.

### Security signoff

- Command:

```sh
bash scripts/release/v1_0/security-review.sh \
  --signoff REPORT_DIR/artifacts/security-review.md
```

- Evidence paths:
  - `REPORT_DIR/artifacts/security-review.md`
  - `REPORT_DIR/artifacts/security-review.md.sha256`
  - security log
- Required result: approved or blocked.

### Performance regression

- Command:

```sh
go run ./tools/cmd/validate-performance-report \
  --report REPORT_DIR/artifacts/performance-regression.json
```

- Evidence path: `REPORT_DIR/artifacts/performance-regression.json`.
- Required result: pass with metric count.

### Binary size thresholds

- Command:

```sh
bash scripts/release/v1_0/binary-size.sh \
  --report REPORT_DIR/artifacts/binary-size-thresholds.json
```

- Evidence path: `REPORT_DIR/artifacts/binary-size-thresholds.json`.
- Required result: pass with fail count 0.

### Reproducible build proof

- Command:

```sh
bash scripts/release/v1_0/reproducible-build.sh \
  --report REPORT_DIR/artifacts/reproducible-build.json
```

- Evidence path: `REPORT_DIR/artifacts/reproducible-build.json`.
- Required result: pass with matched count.

### Release state and artifact hashes

- Gate steps: `validate-release-state`; `validate-artifact-hashes`.
- Evidence paths:
  - `REPORT_DIR/artifacts/release-state.txt`
  - `REPORT_DIR/artifacts/artifact-hashes.json`
- Required result: pass with no missing artifacts.

### Diff hygiene

- Command: `git diff --check`.
- Evidence path: terminal transcript or gate handoff note.
- Required result: pass with exit code 0.

## Backend Summary

Record a backend-focused summary from the same report directory as the final
gate. The summary is incomplete unless every field has a concrete value.

### Backend summary note

- Required evidence:
  `REPORT_DIR/artifacts/backend-summary.md` or checked-in audit note path.

### Commit and version

- Required evidence:

```sh
git rev-parse HEAD
./tetra version
```

### Native target reports

- Required evidence:
  Linux run report plus macOS/Windows build-only reports.

### WASM artifact/import preflight reports

- Required evidence:
  `wasm32-wasi-artifact-smoke.json` and `wasm32-web-artifact-smoke.json`.

### WASI runtime report

- Required evidence:
  `wasi-smoke.json`, including runner name, total, passed, and failed counts.

### Web UI runtime report

- Required evidence:
  `web-ui-smoke.json`, including automation, `ui_schema`, bundle path, module
  path, and DOM snapshot path.

### UI/native boundary decision

- Required evidence:
  one sentence confirming native widgets and runtime event dispatch are post-v1
  unless separately promoted.

### Residual backend risks

- Required evidence:
  explicit list, or `None`, with owner/date for each accepted risk.

## Scope Signoff

### Flow syntax and parser diagnostics

- Evidence summary:
  commands, logs, and artifacts from the final report directory.
- Residual risk: named risk or `None`.
- Signoff: owner and date.

### Type, generic, protocol, extension, module contracts

- Evidence summary:
  commands, logs, and artifacts from the final report directory.
- Residual risk: named risk or `None`.
- Signoff: owner and date.

### Ownership, lifetimes, islands, actors, tasks

- Evidence summary:
  commands, logs, and artifacts from the final report directory.
- Residual risk: named risk or `None`.
- Signoff: owner and date.

### Effects, capabilities, unsafe, privacy, consent, budgets

- Evidence summary:
  commands, logs, and artifacts from the final report directory.
- Residual risk: named risk or `None`.
- Signoff: owner and date.

### Async/task/actor runtime MVP

- Evidence summary:
  commands, logs, and artifacts from the final report directory.
- Residual risk: named risk or `None`.
- Signoff: owner and date.

### Runtime ABI, TOBJ, and linking

- Evidence summary:
  commands, logs, and artifacts from the final report directory.
- Residual risk: named risk or `None`.
- Signoff: owner and date.

### UI metadata and accessibility surface

- Evidence summary:
  commands, logs, and artifacts from the final report directory.
- Residual risk: named risk or `None`.
- Signoff: owner and date.

### CLI, tooling, LSP, docs, and Eco lifecycle

- Evidence summary:
  commands, logs, and artifacts from the final report directory.
- Residual risk: named risk or `None`.
- Signoff: owner and date.

### Target matrix

- Evidence summary:
  commands, logs, and artifacts from the final report directory.
- Residual risk: named risk or `None`.
- Signoff: owner and date.

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
command results must all match this handoff. Approval is invalid if any
required evidence comes from a different branch state.
