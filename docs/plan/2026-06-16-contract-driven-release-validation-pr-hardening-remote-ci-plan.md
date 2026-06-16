# Contract-Driven Release Validation PR Hardening and Remote CI Proof Plan

Date: 2026-06-16

Status: planning document. This is not completion evidence.

Owner area: release validation, tooling contracts, CI, docs verification.

## Goal

Turn the locally completed contract-driven release validation refactor into a reviewable,
mergeable pull request with a clean scope boundary, repeatable local proof, remote CI
evidence, release-package artifact proof, and an explicit follow-up backlog.

The target outcome is not just "the local gate passed once". The target outcome is a
well-scoped PR that reviewers can trust, where every claim is backed by command output,
artifact paths, or CI run links.

## Current Context

The prior implementation concluded that the core problem was systemic:

- `scripts/`, `tools/cmd/`, validators, docs claims, and CI had grown into a separate
  release-validation platform without a single manifest/contract layer.
- The local refactor introduced a contract-driven dispatch surface and moved Surface
  release readiness through that contract path.
- Local evidence exists for the final validation ladder and final Surface release gate.
- Remote GitHub Actions CI has not been claimed as passed.
- A clean worktree has not been claimed.
- The root `GOAL.md` may describe another active objective and must not be overwritten
  without explicit user direction.

Related planning/evidence files:

- `docs/plan/2026-06-16-contract-driven-release-validation-refactor-plan.md`
- `docs/plan/2026-06-16-contract-driven-release-validation-refactor-completion-audit.md`
- `reports/contract-refactor-surface-release-complete-20260616-183817/`

## Non-Goals

- Do not change Tetra language semantics.
- Do not weaken or bypass release gates to make CI green.
- Do not silently delete unrelated user changes or generated artifacts.
- Do not turn this PR-hardening phase into another broad platform rewrite.
- Do not claim remote CI, release packaging, or production readiness without current
  evidence from this phase.
- Do not merge the contract refactor with unrelated Morph, compiler, UI, memory, actor,
  or parser changes unless they are proven required for the release-validation scope.

## Operating Principles

1. Separate scope stabilization from new feature work.
2. Treat local passing commands as `LOCAL` evidence only.
3. Treat a passing affected CI workflow as `INTEGRATION` evidence.
4. Treat release package artifact upload and documented review trail as `END_TO_END`
   evidence for this PR-hardening phase.
5. Mark final completion only when scope, local proof, remote proof, and review handoff
   are all backed by concrete artifacts.

## Required Inputs

Before executing the plan, confirm these inputs or record them as blockers:

- Current branch name and base branch.
- Whether committing and pushing are authorized.
- Whether opening a draft PR is authorized.
- GitHub authentication state for `gh` or the GitHub connector.
- Which CI workflows are required for merge.
- Whether generated evidence directories should be tracked, archived, or ignored.

## Expected Evidence Directory

Use this directory for new PR-hardening evidence:

```text
reports/contract-refactor-pr-hardening/
```

Suggested files:

- `baseline.md`
- `scope-inventory.md`
- `local-validation.md`
- `remote-ci.md`
- `release-artifacts.md`
- `review-handoff.md`
- `blockers.md`

## Phase 0: Baseline and Repo State Capture

### Objective

Capture the exact repo state before any PR-hardening edits, so later claims are anchored
to concrete evidence.

### Commands

```bash
git status --short --branch
git rev-parse HEAD
git branch --show-current
git diff --stat
git diff --name-only
git ls-files --others --exclude-standard
```

### Files to Inspect

- `AGENTS.md`
- `GOAL.md`
- `docs/plan/2026-06-16-contract-driven-release-validation-refactor-plan.md`
- `docs/plan/2026-06-16-contract-driven-release-validation-refactor-completion-audit.md`
- `graphify-out/GRAPH_REPORT.md`
- `.github/workflows/ci.yml`
- `.github/workflows/release-packages.yml`

### Output

Write `reports/contract-refactor-pr-hardening/baseline.md` with:

- current branch;
- base commit;
- dirty/untracked file count;
- known unrelated dirty files;
- known contract-refactor files;
- known generated evidence directories;
- whether root `GOAL.md` is relevant to this task.

### Done When

- The baseline file exists.
- It lists all currently visible dirty and untracked surfaces.
- It explicitly states that unrelated user changes were not modified.

## Phase 1: Scope Inventory and Ownership Split

### Objective

Separate files into reviewable buckets before modifying anything else.

### Buckets

1. Contract-refactor implementation:
   - contract parsing/dispatch packages;
   - gate runner command;
   - release gate script wiring;
   - validators touched by the refactor;
   - tests proving contract wiring.

2. CI and workflow integration:
   - GitHub Actions workflow edits;
   - package release workflow edits;
   - workflow-specific scripts or generated docs claims.

3. Docs and planning:
   - original plan;
   - completion audit;
   - this PR-hardening plan;
   - generated manifest/docs if intentionally updated.

4. Evidence artifacts:
   - `reports/...`;
   - any generated summaries or logs;
   - artifact hashes.

5. Unrelated or ambiguous changes:
   - root `GOAL.md` if it references another goal;
   - unrelated compiler, Morph, parser, memory, actor, UI, or test files;
   - graph outputs changed only because of unrelated prior work.

### Commands

```bash
git diff --name-status
git diff --stat
git ls-files --others --exclude-standard
git status --short --ignored
```

### Output

Write `reports/contract-refactor-pr-hardening/scope-inventory.md` with a table:

```text
Path | Status | Bucket | Include in PR? | Reason | Evidence
```

### Done When

- Every changed or untracked path has a bucket.
- Every path marked "Include in PR" has a reason tied to the contract-refactor goal.
- Every path marked "Exclude" has a safe handling plan.

## Phase 2: Normalize Planning and Audit Documents

### Objective

Make sure planning docs, completion audit, and the future PR description tell the same
story.

### Actions

- Review the original refactor plan for stale assumptions.
- Review the completion audit for command accuracy, evidence paths, and nonclaims.
- Add this PR-hardening plan to the docs index only if such an index already exists and
  is normally maintained.
- Do not rewrite root `GOAL.md` unless the user explicitly asks for a goal-state update.
- If `GOAL.md` conflicts with this task, document the conflict in
  `reports/contract-refactor-pr-hardening/baseline.md`.

### Done When

- Docs distinguish local completion from remote CI proof.
- Docs do not imply a clean tree unless a clean tree is verified.
- Docs do not imply remote pass unless remote CI evidence is captured.

## Phase 3: Validate the Intended PR File Set

### Objective

Confirm the actual implementation files that belong in the PR.

### Likely Surfaces to Review

This list is a starting point, not a claim of completeness:

- `scripts/release/surface/release-gate.sh`
- `scripts/release/surface/contracts/`
- `scripts/release/post_v0_4/contracts/`
- `tools/internal/gatecontract/`
- `tools/internal/reportdir/`
- `tools/internal/artifacts/`
- `tools/cmd/run-gate/`
- `tools/scriptstest/`
- `tools/cmd/validate-manifest/`
- `tools/cmd/verify-docs/`
- `tools/validators/surface/`
- `tools/cmd/validate-surface-runtime/`
- `tools/cmd/validate-surface-release-state/`
- `tools/cmd/validate-surface-crash-report/`
- `tools/cmd/surface-runtime-smoke/`
- `tools/cmd/surface-visual-diff/`
- `.github/workflows/ci.yml`
- `.github/workflows/release-packages.yml`
- `docs/generated/manifest.json`
- `docs/plan/`

### Review Checklist

For each included file:

- Why does this file belong to the contract-refactor scope?
- Is the behavior covered by a test, validator, script syntax check, or CI job?
- Does the file introduce a new public contract?
- If yes, where is that contract documented?
- Does the file depend on a generated artifact?
- Does the file make an unverified docs claim?

### Done When

- The intended PR file set is explicit.
- Ambiguous files are either justified or excluded.
- The PR can be reviewed without unrelated noise.

## Phase 4: Local Validation Rerun

### Objective

Reproduce the local proof from a fresh PR-hardening evidence directory.

### Cache Discipline

Use persistent repo-local caches. Do not put Go caches under `/tmp`.

```bash
mkdir -p .cache/go-build-contract-refactor-pr-hardening
mkdir -p .cache/go-tmp-contract-refactor-pr-hardening
```

Do not set `TMPDIR` for the full Surface release gate if the Chromium or browser probe
expects the system temp behavior. Use `GOTMPDIR` for Go only.

### Commands

```bash
find scripts -name '*.sh' -print0 | xargs -0 -n1 bash -n
```

```bash
GOTELEMETRY=off \
GOCACHE="$PWD/.cache/go-build-contract-refactor-pr-hardening" \
GOTMPDIR="$PWD/.cache/go-tmp-contract-refactor-pr-hardening" \
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

```bash
GOTELEMETRY=off \
GOCACHE="$PWD/.cache/go-build-contract-refactor-pr-hardening" \
GOTMPDIR="$PWD/.cache/go-tmp-contract-refactor-pr-hardening" \
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
```

```bash
GOTELEMETRY=off \
GOCACHE="$PWD/.cache/go-build-contract-refactor-pr-hardening" \
GOTMPDIR="$PWD/.cache/go-tmp-contract-refactor-pr-hardening" \
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

```bash
GOTELEMETRY=off \
GOCACHE="$PWD/.cache/go-build-contract-refactor-pr-hardening" \
GOTMPDIR="$PWD/.cache/go-tmp-contract-refactor-pr-hardening" \
timeout 45m bash scripts/release/surface/release-gate.sh \
  --report-dir reports/contract-refactor-pr-hardening/surface-release-final
```

```bash
git diff --check
```

### Cleanup

After evidence is recorded:

```bash
GOCACHE="$PWD/.cache/go-build-contract-refactor-pr-hardening" go clean -cache
rm -rf .cache/go-tmp-contract-refactor-pr-hardening
```

Do not remove evidence reports unless they are intentionally excluded and already
summarized.

### Output

Write `reports/contract-refactor-pr-hardening/local-validation.md` with:

- exact command lines;
- exit codes;
- key pass/fail lines;
- evidence report paths;
- cache cleanup status.

### Done When

- Shell syntax passes.
- Targeted Go test ladder passes.
- Manifest validation passes.
- Docs verification passes.
- Final Surface release gate exits 0.
- `git diff --check` exits 0.

## Phase 5: Branch and Commit Preparation

### Objective

Prepare the work for review without mixing unrelated changes into the PR.

### Pre-Commit Requirements

- Read `docs/lore-commit-protocol.md` if present.
- Confirm whether the user wants commits created now.
- Confirm whether the current branch should be reused or a new branch should be created.
- Verify no unrelated dirty files are staged.

### Suggested Commit Grouping

If commits are authorized, prefer reviewable slices:

1. Contract foundations:
   - `tools/internal/gatecontract/`
   - `tools/internal/reportdir/`
   - `tools/internal/artifacts/`
   - `tools/cmd/run-gate/`

2. Surface release gate contract wiring:
   - `scripts/release/surface/release-gate.sh`
   - `scripts/release/surface/contracts/`
   - Surface validator or state command updates.

3. Related release-family contract preflight:
   - RAM contract readiness;
   - Memory contract readiness;
   - Actor contract readiness;
   - shared release contract conventions.

4. Tests, docs, CI, and evidence:
   - `tools/scriptstest/`
   - `docs/generated/manifest.json`
   - `.github/workflows/`
   - planning and audit docs.

### Done When

- The staged diff exactly matches the intended PR file set.
- Commit messages follow the local protocol.
- No unrelated dirty file is staged by accident.

## Phase 6: Remote CI Execution

### Objective

Convert local proof into remote integration evidence.

### Actions

- Push the branch if authorized.
- Open a draft PR if authorized.
- Trigger or wait for relevant GitHub Actions workflows.
- Capture workflow run URLs and job names.
- Download or inspect failing logs if anything fails.

### Required Workflows to Check

At minimum inspect:

- `.github/workflows/ci.yml`
- `.github/workflows/release-packages.yml`

Expected relevant jobs may include:

- `surface-release-readiness-linux`
- `ram-contract-release-readiness-linux`
- `actor-runtime-foundation`
- `memory-100`
- package or artifact publication jobs in `release-packages.yml`

The actual job names must be verified from the workflow files and remote run output.

### Output

Write `reports/contract-refactor-pr-hardening/remote-ci.md` with:

- PR URL;
- branch;
- commit SHA;
- workflow names;
- job names;
- run URLs;
- pass/fail state;
- rerun history if any.

### Done When

- Required CI workflows pass remotely, or a specific blocker is recorded.
- No remote success is claimed without a run URL and commit SHA.

## Phase 7: Remote CI Triage Loop

### Objective

Fix remote-only failures with evidence instead of guessing.

### Failure Classification

For each failure, classify it as one of:

- real code defect;
- contract mismatch;
- missing generated artifact;
- CI environment dependency;
- timeout or resource limit;
- flaky external dependency;
- unrelated pre-existing failure.

### Triage Process

1. Capture the failing command and log excerpt.
2. Reproduce locally if practical.
3. Make the smallest scoped fix.
4. Rerun the targeted local validator.
5. Push only the scoped fix.
6. Rerun the failed CI job or workflow.
7. Update `remote-ci.md` with the result.

### Stop Rule

If the same failure survives two scoped fixes, stop and write the blocker in
`reports/contract-refactor-pr-hardening/blockers.md` before trying a third approach.

### Done When

- All CI failures are either resolved or documented as blockers with evidence.

## Phase 8: Release Package Artifact Proof

### Objective

Verify that the release-package path preserves the reports and artifacts needed for
review and release readiness.

### Actions

- Inspect the package workflow outputs.
- Confirm expected report directories are uploaded or summarized.
- Confirm artifact hashes are generated if the workflow expects them.
- Confirm no local-only report is required for the remote release claim.

### Output

Write `reports/contract-refactor-pr-hardening/release-artifacts.md` with:

- workflow run URL;
- artifact names;
- artifact paths;
- hash or manifest references;
- missing artifacts, if any.

### Done When

- Release-package workflow evidence is linked.
- Artifact availability is proven from CI, not only from local disk.

## Phase 9: Review Handoff

### Objective

Prepare a PR that a reviewer can evaluate quickly and safely.

### PR Description Template

```markdown
## Summary

- Introduces a contract-driven release gate layer.
- Wires Surface release readiness through the contract runner.
- Adds focused tests and docs validation for the new contract path.

## Scope

- Contract runner and internal contract packages.
- Surface release gate wiring.
- Related release-family contract preflight surfaces.
- CI/docs/test updates needed to prove the path.

## Evidence

- Local validation: reports/contract-refactor-pr-hardening/local-validation.md
- Remote CI: reports/contract-refactor-pr-hardening/remote-ci.md
- Release artifacts: reports/contract-refactor-pr-hardening/release-artifacts.md

## Nonclaims

- This PR does not claim unrelated compiler, Morph, UI, or language semantics changes.
- This PR does not claim production release completion.
- This PR does not claim remote CI pass beyond the linked commit SHA and runs.

## Risks

- Release tooling now depends on contract files being kept in sync with script behavior.
- Existing scripts still contain logic that should be migrated into typed tools later.

## Follow-ups

- Expand `run-gate` from contract dispatch into full step execution where appropriate.
- Add schema validation for gate contract files if not already present.
- Continue migrating script-only claims into contract-backed validators.
```

### Done When

- PR description links local and remote evidence.
- Reviewers can see what changed, why, and how it was verified.
- Risks and nonclaims are explicit.

## Phase 10: Post-Merge Cleanup Plan

### Objective

Prevent evidence and generated outputs from becoming permanent clutter unless they are
intentionally part of the repo history.

### Actions

- Decide whether `reports/contract-refactor-pr-hardening/` should be tracked, archived,
  ignored, or summarized only.
- If report dirs are not tracked, keep summaries in docs and leave raw CI artifacts in
  GitHub Actions.
- Update `.gitignore` only if the repo already uses that pattern for evidence dirs.
- Update docs to point to durable CI artifacts rather than machine-local paths when
  appropriate.

### Done When

- Local scratch/cache files are removed.
- Evidence policy is explicit.
- No accidental temporary files remain staged.

## Phase 11: Follow-Up Backlog After PR Hardening

These items should not block the PR-hardening phase unless they are required by CI.

### Contract System Follow-Ups

- Add or formalize `docs/schemas/gate-contract.v1.json` if the contract format is
  intended to be stable across gates.
- Add typed validation for every contract field used by release scripts.
- Add a contract registry or index so CI can discover gates without hard-coded lists.
- Add contract version compatibility checks.

### Runner Follow-Ups

- Expand `tools/cmd/run-gate` from dispatch validation into full step execution for gates
  that are ready to move out of shell.
- Add structured output from `run-gate` for CI summaries.
- Add machine-readable failure categories.
- Add a dry-run mode that validates inputs, report dirs, expected artifacts, and external
  dependencies without running expensive steps.

### Script Follow-Ups

- Move repeated report-directory logic out of shell and into shared Go tooling.
- Replace string-heavy shell behavior tests with structured fixtures where practical.
- Reduce duplicated artifact-hash handling.
- Keep shell scripts as orchestration only after behavior moves into typed tools.

### CI Follow-Ups

- Reduce duplicated workflow blocks between release readiness jobs.
- Add a contract matrix generated from the registry.
- Make release readiness jobs upload consistent evidence artifacts.
- Add a required docs-claims job if docs verification is part of merge policy.

### Docs Follow-Ups

- Keep generated manifest ownership explicit.
- Add a release validation architecture note.
- Document how a new gate is added through the contract layer.
- Document how local evidence maps to remote CI evidence.

## Risk Register

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Dirty worktree includes unrelated changes | PR becomes unreviewable | Complete scope inventory before staging |
| Remote CI fails due to environment differences | Local proof is insufficient | Capture logs, reproduce locally, patch minimally |
| Contract files drift from script behavior | Gates silently diverge | Add tests that compare script wiring to contract expectations |
| Evidence dirs become noisy repo artifacts | Repo clutter and unclear history | Decide tracking policy before commit |
| Docs overclaim readiness | Reviewers get false confidence | Keep nonclaims and evidence links explicit |
| Release package workflow misses reports | End-to-end proof is incomplete | Verify uploaded artifacts from CI run |

## Decision Gates

### Gate A: Scope Freeze

Proceed only when:

- `scope-inventory.md` is complete;
- unrelated changes are excluded or explicitly justified;
- the intended PR file set is clear.

### Gate B: Local Proof

Proceed only when:

- shell syntax passes;
- targeted Go test ladder passes;
- manifest/docs verification passes;
- final Surface release gate passes;
- `git diff --check` passes.

### Gate C: Commit/Push Authorization

Proceed only when:

- the user has authorized commits and remote push;
- commit grouping is chosen;
- no unrelated files are staged.

### Gate D: Remote Proof

Proceed only when:

- required workflows pass for the pushed commit SHA;
- release artifacts are available or a blocker is documented.

### Gate E: Review Ready

Proceed only when:

- PR description includes summary, scope, evidence, nonclaims, risks, and follow-ups;
- local and remote evidence files are current;
- remaining risks are acceptable and explicit.

## Acceptance Criteria

This plan is complete when all of the following are true:

- A full scope inventory exists for all dirty and untracked files.
- The intended PR file set excludes unrelated work.
- Local validation has been rerun and recorded under
  `reports/contract-refactor-pr-hardening/`.
- Remote CI evidence exists for the relevant workflows and commit SHA, or a blocker is
  documented.
- Release-package artifact evidence exists, or a blocker is documented.
- The PR or draft PR description links the evidence and states nonclaims.
- Cache/temp cleanup has been performed.
- No destructive cleanup was used.
- Final status is reported as `DONE`, `PARTIAL`, or `BLOCKED` with evidence.

## First Execution Checklist

1. Read current `git status --short --branch`.
2. Read the prior refactor plan and completion audit.
3. Create `reports/contract-refactor-pr-hardening/baseline.md`.
4. Build `scope-inventory.md`.
5. Decide which files belong in the PR.
6. Rerun local validation into the new evidence directory.
7. Clean Go cache and Go temp directory.
8. Prepare staged diff only after scope is frozen.
9. Ask for commit/push authorization if not already granted.
10. Push/open draft PR only after local proof passes.
11. Capture remote CI and release artifact evidence.
12. Prepare review handoff.

