# v0.3.0 Candidate Known Issues

## Release

- Version: `v0.3.0`
- Candidate or patch branch: `main`
- Artifact archive: `reports/plan250/waveC-v0_3_gate-rerun`
- Last release gate command:
  `bash scripts/release/v0_3_0/gate.sh --report-dir reports/plan250/waveC-v0_3_gate-rerun`
- Historical Wave-C archive result: `pass`, 0 failed steps of 10
  (stale archive only; not the current evidence contract)
- Current gate semantics: Wave-24 release gates require a same-run security
  signoff archive, detached `artifacts/security-review.md.sha256` attestation,
  blocked CI missing-signoff artifacts, and
  `artifacts/residual-risks.json`; the current gate is a 14-step evidence
  contract.

## Stale Evidence Caveat

The historical Wave-C `pass with release-process caveat` and its 10-step pass
archive are stale for the current Wave-23+ gate shape. The 10-step archive
records historical Wave-C evidence only and is not the current evidence
contract. It is non-evidence for the current non-CI `v0.3.0` 14-step release
evidence pass until a fresh gate run is recorded with
`TETRA_SECURITY_REVIEW_SIGNOFF=<security-review.md>` and, for tag-ready
promotion, `--require-clean`.

## Issues

| ID | Title | Component | User impact | Workaround | Release blocker? | Owner | Evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| KI-001 | Named security signoff is required for an evidence pass | Release security | CI can collect non-tag-ready gate artifacts before the reviewer signoff exists, but that run remains `blocked` and cannot be cited as a release evidence pass or tag-ready promotion. | Run the final gate with `TETRA_SECURITY_REVIEW_SIGNOFF=<security-review.md>` so it archives same-run `artifacts/security-review.md` and writes the detached `artifacts/security-review.md.sha256`; use `TETRA_RELEASE_GATE_CI_ALLOW_MISSING_SECURITY_SIGNOFF=1` only for blocked CI evidence collection with `summary.json`, `artifacts/release-state.json`, and `artifacts/release-state.txt` marked blocked. | yes | Security reviewer / release coordinator | `docs/checklists/v0_3_0_release_gate.md`; `reports/plan250/waveC-security/security-review-dry-run.md` |
| KI-002 | Initial canonical full wrapper run hit sandbox Go cache access | Test environment | The first requested `waveB-full` run failed before repo tests completed under sandbox cache restrictions. | Use an approved non-sandbox run or an in-workspace `GOCACHE`; `waveB-full-rerun` and the Wave-C gate rerun passed. | no | Test environment owner | `reports/plan250/waveB-full/summary.md`; `reports/plan250/waveB-full-rerun/summary.md`; `reports/plan250/waveC-v0_3_gate-rerun/summary.md` |

## Wave-24 Gate Semantics

- A release evidence pass requires a human security signoff from the same gate
  run. The gate stages `TETRA_SECURITY_REVIEW_SIGNOFF=<security-review.md>`,
  refreshes the canonical hashes after `summary.json`,
  `artifacts/release-state.json`, and `artifact-hashes.json` exist, then
  archives the final signoff as `artifacts/security-review.md`.
- The canonical `artifact-hashes.json` remains cycle-safe by excluding
  `artifacts/security-review.md` and
  `artifacts/security-review.md.sha256`. The detached
  `artifacts/security-review.md.sha256` file is required and must attest the
  final archived `artifacts/security-review.md`.
- CI may set `TETRA_RELEASE_GATE_CI_ALLOW_MISSING_SECURITY_SIGNOFF=1` only for
  non-tag-ready evidence collection. That mode writes a missing-signoff
  `artifacts/security-review.md` placeholder, keeps `summary.json` at
  `status: "blocked"`, and emits blocked `artifacts/release-state.json` and
  `artifacts/release-state.txt`.
- The gate requires `artifacts/residual-risks.json`. If
  `TETRA_RESIDUAL_RISKS_JSON` is set, the source file is copied and validated;
  otherwise the gate archives a valid empty residual-risk object for `v0.3.0`.

## REL-029 Script Reuse Rationale

The `v1_0` script names reused in the `v0.3.0` evidence are accepted as
shared release tooling, not as reused `v1.0.0` artifacts. The relevant
commands are explicitly mapped in
`docs/release/v0_3_0_final_handoff.md#rel-029-shared-script-reuse-closure`.

| Command | Release impact | Issue status |
| --- | --- | --- |
| `bash scripts/release/v1_0/wasi-smoke.sh --report .../wasi-smoke.json` | Produces fresh WASI smoke evidence under the `v0.3.0` report directory. | Not a known issue. |
| `bash scripts/release/v1_0/web-smoke.sh --report .../web-ui-smoke.json` | Produces fresh web UI smoke evidence under the `v0.3.0` report directory. | Not a known issue. |
| `bash scripts/release/v1_0/api-diff.sh --report-dir .../api-diff --baseline docs/baselines/api-diff-baseline.v1alpha1.json --enforce no-change` | Enforces no undocumented public API drift for the current `v0.3.0` branch state. | Not a known issue. |
| `bash scripts/release/v0_3_0/security-review.sh --signoff <security-review.md>` | Validates the v0.3.0 signoff shape and canonical artifact hashes; the final gate must still archive the same-run `artifacts/security-review.md` and detached `artifacts/security-review.md.sha256`. | Covered by `KI-001`; the wrapper may reuse shared security-review tooling internally, but the release-facing entrypoint and artifacts are v0.3.0-scoped. |

## Triage Rules

- A blocker prevents release until fixed or explicitly descoped from the
  release with checklist updates.
- A non-blocker must have a user-facing workaround and release notes entry.
- Closed issues need the command that proved the fix.
