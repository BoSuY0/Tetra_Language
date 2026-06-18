# Tetra v0.3.0 Final Handoff

Status: handoff template for the `v0.3.0` release line.

## Release Truth

- Current profile: `v0.3.0`
- Current supported surface: `docs/spec/current_supported_surface.md`
- Scope contract: `docs/spec/v0_3_scope.md`
- Checklist: `docs/checklists/v0_3_0_release_gate.md`
- Gate: `scripts/release/v0_3_0/gate.sh`
- Release notes: `docs/release-notes/v0_3_0.md`

## Promoted Slices

- `language.enum-payload-match`: positional enum payload constructors/bindings for
  match/catch/if-let with exhaustive unguarded enum match/catch diagnostics.
- `language.protocol-bound-generics-static`: static protocol-bound generic
  validation during monomorphization without dynamic dispatch.

## Deferred Slices

- Callable Level 1 remains experimental.
- Ownership/resource safety remains conservative MVP hardening; lifetime SSA is
  planned.
- Capsule/Eco claims remain local-only.
- WASI/Web runtime execution remains planned; build-only and reporting evidence
  stay explicit.

## Required Final Evidence

```sh
go test ./compiler/... ./cli/... ./tools/... -count=1
bash scripts/ci/test-all.sh --stabilization --keep-going --report-dir reports/v0.3-stabilization
bash scripts/dev/fuzz-nightly.sh --short --out-dir reports/v0.3-stabilization/fuzz-short
./tetra smoke --target macos-x64 --run=true --report <macos-smoke-run-true.json> # on macOS host or CI macOS runner
./tetra smoke --target windows-x64 --run=true --report <windows-smoke-run-true.json> # on Windows host or CI Windows runner
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
git diff --check
TETRA_MACOS_RUNTIME_SMOKE_REPORT=<macos-smoke-run-true.json> \
TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=<windows-smoke-run-true.json> \
TETRA_SECURITY_REVIEW_SIGNOFF=<security-review.md> \
bash scripts/release/v0_3_0/gate.sh --report-dir reports/release-v0.3.0-gate
```

For a non-CI evidence pass, `TETRA_SECURITY_REVIEW_SIGNOFF` must point at the
named human security review signoff artifact. The CI-only placeholder mode
(`TETRA_RELEASE_GATE_CI_ALLOW_MISSING_SECURITY_SIGNOFF=1` without
`TETRA_SECURITY_REVIEW_SIGNOFF`) records incomplete evidence and is not a full
release evidence pass. For tag-ready promotion, rerun the same gate with
`--require-clean` on a clean worktree.

The release gate's required artifacts include
`reports/release-v0.3.0-gate/artifacts/residual-risks.json`; it must be the
same file produced by residual risk validation in that gate run.

The release gate also requires real host-gated runtime execution reports for
macOS and Windows:

- `reports/release-v0.3.0-gate/artifacts/macos-runtime-smoke.json`
- `reports/release-v0.3.0-gate/artifacts/windows-runtime-smoke.json`

These are not cross-target build-only reports. They must come from
`./tetra smoke --target macos-x64 --run=true` on a macOS host or CI macOS runner
and `./tetra smoke --target windows-x64 --run=true` on a Windows host or CI
Windows runner. `validate-release-state` rejects missing reports, `build_only:
true`, target/host mismatches, stale version/Git head metadata, and actor/task
smoke cases that did not run.

## Evidence Pass vs Tag-Ready Clean Pass

Record these states separately:

- Evidence pass: all required final evidence commands passed for the recorded
  branch state, and the report directory was archived.
- Dirty waiver: allowed only for an evidence pass. Name every dirty path,
  explain why it is outside the candidate evidence, and link the exact command
  results used for the evidence pass. A dirty waiver never authorizes tagging.
- Tag-ready clean pass: the evidence pass is current for the intended tag
  commit, `git status --short` prints no entries, and this handoff records that
  clean worktree result.

Current tag-readiness status: blocked until the final reviewer signoff archive
is recorded and the handoff records a tag-ready clean pass.

## 2026-05-04 Completion Audit Snapshot

Status: not release-complete; tracked by
`docs/release/v0_3_0_completion_audit.md`.
The external unblock packet is
`docs/release/v0_3_0_unblock_packet.md`.

Local-only candidate evidence:

- `reports/release-v0.3.0-local-candidate/summary.md`: repo-local evidence
  bundle for the local candidate.
- `reports/release-v0.3.0-local-candidate.tar.gz`: packaged local-candidate
  archive with detached SHA256 in
  `reports/release-v0.3.0-local-candidate.tar.gz.sha256`.
- `docs/release/v0_3_0_local_candidate_summary.json`: machine-readable
  local-candidate status; `tag_ready: false`.
- `./tetra smoke --target linux-x64 --run=true --report /tmp/tetra-v0.3-linux-runtime-smoke.json`:
  `pass`; 62 runtime cases passed on `host: linux-x64`.
- `go run ./tools/cmd/smoke-report-to-checklist --validate-only --report /tmp/tetra-v0.3-linux-runtime-smoke.json`:
  `pass`.

Fresh evidence collected during the audit:

- `bash scripts/ci/test-all.sh --stabilization --keep-going --report-dir /tmp/tetra-v0.3-stabilization-audit-rerun`:
  `pass`; 38 checks, 0 failed.
- `env TETRA_RELEASE_GATE_CI_ALLOW_MISSING_SECURITY_SIGNOFF=1 TETRA_MACOS_RUNTIME_SMOKE_REPORT=docs/generated/v1_0/macos-smoke.json TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=docs/generated/v1_0/windows-smoke.json bash scripts/release/v0_3_0/gate.sh --report-dir /tmp/tetra-v0.3-gate-audit-crossbuild-runtime-blocked`:
  `blocked`; gate identity is correct for `v0.3.0`, but the supplied macOS
  and Windows smoke reports are cross-target build artifacts from
  `host: linux-x64`, not native `--run=true` runtime execution evidence.

Remaining blockers:

- Native macOS and Windows `./tetra smoke --target <target> --run=true`
  reports for the current Git head.
- Same-run human security signoff and detached hash.
- Final `v0.3.0` gate pass using those inputs.
- Clean worktree and `--require-clean` pass before tagging.

## Wave-C Final Evidence Snapshot

Status: historical `pass with release-process caveat` on 2026-04-29.

This snapshot is stale for the current Wave-47+ gate shape. It records the
historical Wave-C archive only; it does not replace a fresh non-CI `v0.3.0`
release gate run with `TETRA_MACOS_RUNTIME_SMOKE_REPORT`,
`TETRA_WINDOWS_RUNTIME_SMOKE_REPORT`,
`TETRA_SECURITY_REVIEW_SIGNOFF=<security-review.md>`, and, for tag-ready
promotion, `--require-clean`.

Current branch evidence was collected from commit
`b8846534066cd9400ab1b3fc902973fc2ef7fc57` on `main`.

| Evidence | Result | Report |
| --- | --- | --- |
| `bash scripts/ci/test-all.sh --full --keep-going --report-dir reports/plan250/waveB-full` | `fail`; sandbox run could not read the Go build cache during `bash scripts/ci/test.sh`. | `reports/plan250/waveB-full/summary.md` |
| `bash scripts/ci/test-all.sh --full --keep-going --report-dir reports/plan250/waveB-full-rerun` | `pass`; 27 checks, 0 failed. | `reports/plan250/waveB-full-rerun/summary.md` |
| `bash scripts/ci/test-all.sh --stabilization --keep-going --report-dir reports/plan250/waveB-stabilization` | `pass`; 38 checks, 0 failed. | `reports/plan250/waveB-stabilization/summary.md` |
| `bash scripts/release/v0_3_0/gate.sh --report-dir reports/plan250/waveC-v0_3_gate-rerun` | `pass`; 10 checks, 0 failed. | `reports/plan250/waveC-v0_3_gate-rerun/summary.md` |
| `bash scripts/release/v1_0/security-review.sh --signoff reports/plan250/waveC-security/security-review-dry-run.md` | `pass`; validator dry-run only. | `reports/plan250/waveC-security/security-review-dry-run.md` |
| `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` | `pass`. | Direct command exit 0. |
| `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json` | `pass`. | Direct command exit 0. |
| `git diff --check` | `pass`. | Direct command exit 0. |

## REL-029 Shared Script Reuse Closure

The `v1_0` names below are historical script names, not claims that the
artifacts are `v1.0.0` release evidence. They are acceptable for `v0.3.0`
because each command validates a version-neutral release invariant and writes
fresh reports under the selected `v0.3.0` report directory.

| Reused command | v0.3 artifact location | Rationale |
| --- | --- | --- |
| `bash scripts/release/v1_0/wasi-smoke.sh --report reports/plan250/waveC-v0_3_gate-rerun/artifacts/test-all/wasi-smoke.json` | `reports/plan250/waveC-v0_3_gate-rerun/artifacts/test-all/wasi-smoke.json`; `logs/34-wasi-runner-smoke.log` | Reuses the shared WASI smoke harness to validate runtime/build evidence for the current branch state; the output path is inside the `v0.3.0` stabilization archive. |
| `bash scripts/release/v1_0/web-smoke.sh --report reports/plan250/waveC-v0_3_gate-rerun/artifacts/test-all/web-ui-smoke.json` | `reports/plan250/waveC-v0_3_gate-rerun/artifacts/test-all/web-ui-smoke.json`; `logs/35-web-ui-browser-smoke.log` | Reuses the shared web UI smoke harness to validate the browser/UI report schema and behavior; the report is regenerated for the `v0.3.0` gate run. |
| `bash scripts/release/v1_0/api-diff.sh --report-dir reports/plan250/waveC-v0_3_gate-rerun/artifacts/test-all/api-diff --baseline docs/baselines/api-diff-baseline.v1alpha1.json --enforce no-change` | `reports/plan250/waveC-v0_3_gate-rerun/artifacts/test-all/api-diff`; `logs/36-api-diff-no-change.log` | Reuses the shared API diff checker against the checked-in v1 alpha baseline because `v0.3.0` must not introduce an undocumented public surface drift. |
| `bash scripts/release/v1_0/security-review.sh --signoff reports/plan250/waveC-security/security-review-dry-run.md` | `reports/plan250/waveC-security/security-review-dry-run.md` | Reuses the shared security signoff schema validator only; the recorded result is a local dry-run and remains blocked until replaced by the named reviewer signoff for `v0.3.0`. |

Release-state passed in the fresh Wave-C archive. The archived
`reports/plan250/waveC-v0_3_gate-rerun/artifacts/release-state.txt` reports
`required artifacts: 36`, `missing artifacts: 0`, and last gate evidence
`pass`. That required artifact set includes
`reports/plan250/waveC-v0_3_gate-rerun/artifacts/residual-risks.json`.

Tagging rule: do not tag `v0.3.0` from this branch state until the named
security reviewer replaces the local dry-run signoff, this handoff points at
that final signoff archive, and the handoff records a tag-ready clean pass with
an empty `git status --short` result. Any dirty-waiver evidence remains review
evidence only and does not make the branch tag-ready.
