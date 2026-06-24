# v0.3.0 Release Gate Checklist

Status: historical release-gate checklist for the `v0.3.0` release line. The
active release profile has moved past this version; retagging this historical line
would still require a fresh passing gate
report in the same branch state.

Canonical scope: `docs/spec/flow/v0_3_scope.md`.
Current truth before promotion: `docs/spec/current_supported_surface.md`.
Artifact policy: `docs/release/artifact_policy.md`.
Security review gate: `docs/checklists/security_review_gate.md`.

## Required Gate

Run:

```sh
TETRA_MACOS_RUNTIME_SMOKE_REPORT=<macos-smoke-run-true.json> \
TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=<windows-smoke-run-true.json> \
bash scripts/release/v0_3_0/gate.sh --report-dir <report-dir>
```

The gate is blocked unless both local entrypoints report the release version:

```sh
./tetra version
./t version
```

Both commands must print `v0.3.0`. The gate artifact mapping is
`tetra.release.v0_3_0.gate-report.v1`.

## Required Evidence

The gate must collect or point to these checks in the same branch state:

```sh
go test ./compiler/... ./cli/... ./tools/... -count=1
bash scripts/ci/test-all.sh --stabilization --keep-going --report-dir <report-dir>/artifacts/test-all
bash scripts/dev/fuzz-nightly.sh --short --out-dir <report-dir>/artifacts/fuzz-short
./tetra smoke --target macos-x64 --run=true --report <macos-smoke-run-true.json> # on macOS host or CI macOS runner
./tetra smoke --target windows-x64 --run=true --report <windows-smoke-run-true.json> # on Windows host or CI Windows runner
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
bash scripts/release/v0_3_0/security-review.sh --signoff <security-review.md>
git diff --check
```

The `go test packages` step clears release input environment variables before
running package tests. `TETRA_SECURITY_REVIEW_SIGNOFF`,
`TETRA_RELEASE_GATE_CI_ALLOW_MISSING_SECURITY_SIGNOFF`,
`TETRA_MACOS_RUNTIME_SMOKE_REPORT`, `TETRA_WINDOWS_RUNTIME_SMOKE_REPORT`, and
`TETRA_RESIDUAL_RISKS_JSON` are gate inputs only and must not change package
test behavior.

The macOS and Windows runtime smoke reports are host-gated inputs, not
Linux-produced cross-target build evidence. The release gate validates both host-gated reports before archiving either runtime evidence artifact; only after
both pass does it copy them into
`<report-dir>/artifacts/macos-runtime-smoke.json` and
`<report-dir>/artifacts/windows-runtime-smoke.json`. Release-state validation
requires both archived reports to be target-matching, `build_only: false`,
same-host, fresh for the candidate version/Git head, and to contain ran/passing
actor and task smoke cases.

Security signoff evidence is mandatory for the exact `v0.3.0` candidate:
run the gate with `TETRA_SECURITY_REVIEW_SIGNOFF=<security-review.md>`. The
gate stages the reviewer signoff during the evidence steps, creates the fresh
same-run `summary.json`, `release-state.json`, and `artifact-hashes.json`
artifacts, then archives and validates the final signoff as
`<report-dir>/artifacts/security-review.md`. The source signoff's
`Report directory:` must resolve to the same `<report-dir>` used by the gate.
To keep canonical artifact hashes cycle-safe, the final canonical
`<report-dir>/artifact-hashes.json` excludes both
`artifacts/security-review.md` and its detached attestation
`artifacts/security-review.md.sha256`. The detached hash artifact must attest
the final archived `artifacts/security-review.md` from the same run.
During the pre-signoff release-state refresh, release-state records not-yet-archived signoff evidence as `deferred`; the final source of truth for signoff validity is `bash scripts/release/v0_3_0/security-review.sh --signoff <report-dir>/artifacts/security-review.md`
after the gate has written `summary.json`, `artifact-hashes.json`, and
`artifacts/release-state.json`.

CI may set `TETRA_RELEASE_GATE_CI_ALLOW_MISSING_SECURITY_SIGNOFF=1` only to
collect non-tag-ready evidence before a human signoff exists. In that mode the
gate writes `<report-dir>/artifacts/security-review.md` as an explicit
missing-signoff placeholder, keeps `summary.json` at `status: "blocked"`,
emits blocked `<report-dir>/artifacts/release-state.json` and
`<report-dir>/artifacts/release-state.txt`, and prints that the run is not a
full release evidence pass. This CI mode must not be used as evidence pass or
tag-ready release promotion.

Required artifacts:

- `<report-dir>/summary.json`
- `<report-dir>/summary.md`
- `<report-dir>/logs/*.log`
- `<report-dir>/artifacts/test-all/summary.json`
- `<report-dir>/artifacts/fuzz-short/*`
- `<report-dir>/artifacts/release-state.json`
- `<report-dir>/artifacts/release-state.txt`
- `<report-dir>/artifacts/residual-risks.json`
- `<report-dir>/artifacts/macos-runtime-smoke.json`
- `<report-dir>/artifacts/windows-runtime-smoke.json`
- `<report-dir>/artifact-hashes.json`
- `<report-dir>/artifacts/security-review.md`
- `<report-dir>/artifacts/security-review.md.sha256`

The gate writes `artifacts/residual-risks.json` during residual risk
validation. If `TETRA_RESIDUAL_RISKS_JSON` is set, the gate copies and
validates that source file; otherwise it archives a valid empty residual-risk
object for the same `v0.3.0` release version.

## Evidence Pass vs Tag-Ready Clean Pass

An evidence pass means every required evidence command completed successfully
for the recorded branch state and the report directory contains the required
artifacts, including the same-run human security signoff. A CI run with
`TETRA_RELEASE_GATE_CI_ALLOW_MISSING_SECURITY_SIGNOFF=1` is blocked evidence
collection, not an evidence pass. An evidence pass may be recorded while the
worktree is dirty only when the handoff includes a dirty waiver that names the
dirty files, explains why they are outside the candidate evidence, and records
the exact command results used for the evidence pass.

A dirty waiver never makes the release tag-ready. It only preserves the
evidence result for review while follow-up cleanup is pending.

A tag-ready clean pass requires all evidence-pass criteria plus a clean
worktree at the intended tag commit:

```sh
bash scripts/release/v0_3_0/gate.sh --report-dir <report-dir> --require-clean
git status --porcelain --untracked-files=all
```

The command must print no tracked or untracked entries, and the final handoff
must record that clean worktree result. If `git status --porcelain --untracked-files=all` prints any
entry, the release remains blocked for tagging even when the evidence commands
pass.

## Promotion Boundaries

- `v1.0.0` scripts or reports are not `v0.3.0` evidence unless this checklist
  explicitly names the reused command and records why it applies.
- Feature support claims must match `compiler/features.go`,
  `docs/generated/manifest.json`, and `docs/spec/current_supported_surface.md`.
- Preview docs cannot promote generic structs, dynamic protocol dispatch, full
  first-class callables, lifetime SSA, WASM runtime parity, distributed EcoNet,
  distributed actors, or production UI runtime behavior.

## Done When

The release has an evidence pass when `bash scripts/release/v0_3_0/gate.sh
--report-dir <report-dir>` passes with security signoff evidence and the report
directory is archived. If the worktree is dirty, the handoff must record the
dirty waiver and must not call the candidate tag-ready.

The release is tag-ready only after a tag-ready clean pass: the evidence pass is
still current for the intended tag commit, `git status --porcelain --untracked-files=all` is empty,
and
the handoff records the clean worktree result, the selected promoted slices,
and any deferred `v0.3.0` candidates.
