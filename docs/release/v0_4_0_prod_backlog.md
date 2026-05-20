# Tetra v0.4.0 Production Backlog

Status: completed scoped Linux-x64 backlog derived from
`docs/release/v0_4_0_prod_gap_matrix.json` and
`docs/release/v0_4_0_scope_decisions.json`.

This backlog records the implementation themes that close the Linux-x64
`v0.4.0` production claim. EcoNet, non-Linux targets, WASM target runtimes, and
full v1.0 language guarantees are explicitly outside this release scope.

## Objective Boundary

Requested objective:

- Finish Tetra for Linux x64 first.
- Mark it as `0.4.0`.
- Keep only real production behavior in the release claim.
- Exclude EcoNet from the initial production scope.

Current scope decision:

- selected feature gaps are implemented/current for the scoped Linux-x64 claim
- `linux-x64` is the only selected production runtime target
- no selected non-current feature or target runtime gap remains in the matrix
- final tag-ready evidence is produced by the canonical clean candidate gate,
  which includes memory, parallelism, and compiler production artifacts

## Epic 1: Production Scope Decision

Goal: keep the scoped Linux-x64 `v0.4.0` contract machine-readable and aligned
with human docs.

Inputs:

- `docs/release/v0_4_0_prod_gap_audit.md`
- `docs/release/v0_4_0_prod_gap_matrix.json`
- `docs/release/v0_4_0_scope_decisions.md`
- `docs/release/v0_4_0_scope_decisions.json`
- `docs/spec/v0_4_scope.md`

Done when:

- Every selected gap is marked `implement` or `implement-production-runtime`.
- Every excluded gap is named as outside the production claim.
- User-facing docs do not imply excluded behavior is `v0.4.0` production.
- The production gap matrix reports `prod_ready_now: true` for the expanded
  memory/parallel/compiler gate from the clean candidate.

Verification:

```sh
jq empty docs/release/v0_4_0_prod_gap_matrix.json
jq empty docs/release/v0_4_0_scope_decisions.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
```

## Epic 2: Linux-x64 Release Gate

Goal: make `scripts/release/v0_4_0/gate.sh` produce final scoped release
evidence instead of stopping after readiness preflight.

Primary impact areas:

- `scripts/release/v0_4_0/gate.sh`
- `tools/scriptstest/release_v040_gate_test.go`
- `tools/cmd/validate-release-gate-summary`
- `tools/cmd/validate-artifact-hashes`

Done when:

- The gate writes `tetra.release.v0_4_0.gate-report.v1`.
- The gate records feature, target, Linux smoke, memory production, parallel
  production, compiler production, actor runtime, native UI runtime,
  release-state, security-review, and artifact-hash evidence.
- `--require-clean` blocks dirty tag-ready runs before expensive evidence.
- A non-`--require-clean` evidence run can be used for local verification.
- The clean gate report validates with `status: pass`, `failed_count: 0`, and
  includes `memory-production-linux-x64.json`,
  `parallel-production-linux-x64.json`, and
  `compiler-production-linux-x64.json` in `artifact-hashes.json`.

Verification:

```sh
bash scripts/release/v0_4_0/gate.sh --report-dir <report-dir>
go run ./tools/cmd/validate-release-gate-summary --summary <report-dir>/summary.json --report-dir <report-dir> --expected-version v0.4.0 --expected-artifact tetra.release.v0_4_0.gate-report.v1 --expected-command 'bash scripts/release/v0_4_0/gate.sh'
go run ./tools/cmd/validate-artifact-hashes --manifest <report-dir>/artifact-hashes.json
```

## Epic 3: Final Verification Pass

Goal: rerun the commands that prove the selected Linux-x64 scope after every
scope/doc/code change.

Required commands:

```sh
go test ./tools/cmd/validate-v0-4-readiness -count=1
go run ./cli/cmd/tetra smoke --target linux-x64 --run=true --report reports/v0.4.0/linux-host-smoke.json
bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh --report-dir reports/v0.4.0
go run ./tools/cmd/validate-memory-production --report reports/v0.4.0/memory-production-linux-x64.json
bash scripts/release/post_v0_4/parallel-production-linux-x64-smoke.sh --report-dir reports/v0.4.0
go run ./tools/cmd/validate-parallel-production --report reports/v0.4.0/parallel-production-linux-x64.json
bash scripts/release/post_v0_4/compiler-production-linux-x64-smoke.sh --report-dir reports/v0.4.0
go run ./tools/cmd/validate-compiler-production --report reports/v0.4.0/compiler-production-linux-x64.json
bash scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh --report-dir reports/v0.4.0
bash scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh --report-dir reports/v0.4.0
go run ./tools/cmd/validate-v0-4-readiness --expected-version v0.4.0 --features reports/v0.4.0/features.json --targets reports/v0.4.0/targets.json --manifest docs/generated/manifest.json --scope-decisions docs/release/v0_4_0_scope_decisions.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
```

Final clean release candidate evidence:

- `reports/v0.4.0/linux-host-smoke.json`
- `reports/v0.4.0/distributed-actors-linux-x64.json`
- `reports/v0.4.0/native-ui-linux-x64.json`
- `reports/v0.4.0/release-gate-clean/summary.json`
- `reports/v0.4.0/release-gate-clean/artifact-hashes.json`
- `reports/v0.4.0/release-gate-clean/artifacts/release-state.json`

## Epic 4: Security And Tag Readiness

Goal: make the release auditable for production use.

Done when:

- `scripts/release/v0_4_0/security-review.sh --signoff <path>` validates an
  approved signoff for the exact release candidate.
- The final gate report includes `security-review.md` and
  `security-review.md.sha256`.
- `git status --porcelain --untracked-files=all` is empty before tagging.
- `docs/release/v0_4_0_final_handoff.md` records the exact release commit.
- The copied clean gate release-state artifact has `git.clean: true` and zero
  git entries.

Verification:

```sh
bash scripts/release/v0_4_0/security-review.sh --signoff <security-review.md>
git status --porcelain --untracked-files=all
bash scripts/release/v0_4_0/gate.sh --report-dir <report-dir> --require-clean
```
