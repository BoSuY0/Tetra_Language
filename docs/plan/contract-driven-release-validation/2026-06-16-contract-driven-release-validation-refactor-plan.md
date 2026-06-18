# Contract-Driven Release Validation Refactor Plan

**Status:** planning document, not implementation evidence. **Date:** 2026-06-16. **Owner:**
release/tooling/validation architecture. **Requested by:** user request to fully fix the systemic
drift between `scripts`, `tools/cmd`, validators, docs claims, and CI. **Primary recommendation:**
contract-first vertical migration, starting with the Surface release `crash_reporting` slice.

## 1. Goal

Create one durable release/validation contract layer so release claims cannot drift between shell
scripts, Go validators, docs, generated manifests, and CI.

The target end state is:

- every release gate has a machine-readable contract;
- every contract lists required steps, reports, validators, artifacts, claims, nonclaims, and host
  preconditions;
- local scripts and CI workflows execute the same contract;
- validators check contract outputs rather than relying on brittle duplicated shell/script
  assumptions;
- script tests verify contracts and dry-run plans, not long fragile string snippets;
- the current Surface `crash_reporting` inconsistency is fixed end-to-end first;
- later Surface, RAM, Memory, Actor, and package gates can migrate by repeating the same pattern.

## 2. Non-goals

This plan does not rewrite the whole repository in one batch.

Out of scope for the first implementation wave:

- changing Tetra language semantics;
- weakening any existing release gate, validator, nonclaim, or evidence requirement;
- replacing all shell scripts immediately;
- deleting historical release scripts before callers migrate;
- claiming remote CI, packaging, or production status without current evidence;
- cleaning user dirty worktree changes with destructive commands;
- moving unrelated compiler, CLI, docs, or example files only for aesthetics.

## 3. Observed Current Facts

These facts were inspected locally before writing the plan.

- `docs/plan/` exists and already contains planning documents.
- `docs/plans/` is the usual canonical plan directory used by several existing repo plans, but the
  user explicitly requested `docs/plan/`.
- `go.work` includes four modules: `.`, `./compiler`, `./cli`, and `./tools`.
- `scripts/release/shared/README.md` exists, but no shared shell helper file was found there during
  the audit.
- `scripts/ci/test-all.sh` is a large canonical summarized test runner.
- `scripts/release/surface/release-gate.sh` is the main Surface v1 release gate.
- `tools/validators/surface/surface_core.go` is a large Surface validation hub and currently
  includes release summary validation.
- `tools/cmd/validate-surface-runtime` validates Surface runtime and release summary envelopes.
- `tools/cmd/validate-surface-crash-report/main.go` exists in the current dirty tree and calls
  `surface.ValidateCrashReport`.
- `tools/validators/surface/surface_morph_release.go` and its tests exist in the dirty tree.
- A focused verification run showed `tools/validators/surface` passing while
  `tools/cmd/validate-surface-runtime` failed because release summary fixtures or generated summary
  data lacked `crash_reporting`.
- `tools/scriptstest` passed in the audit run, but took roughly 157 seconds.
- The worktree is very dirty and must be preserved.

## 4. Design Summary

Introduce a small contract layer first, then migrate one vertical slice through it.

The core idea:

```text
contract JSON
  -> dry-run plan
  -> runner executes steps
  -> required reports are produced
  -> validators check reports and contract conformance
  -> artifact hashes are written and validated
  -> CI uploads the same outputs
```

Shell scripts should become stable entrypoints around the contract runner. Go validators remain the
source of semantic validation. Docs and generated manifest checks should consume evidence named by
the same contract.

## 5. Proposed File Boundaries

Add or modify only after each path is re-verified in the implementation pass.

Likely additions:

- `tools/internal/gatecontract/`
- `tools/internal/reportdir/`
- `tools/internal/artifacts/`
- `tools/cmd/run-gate/`
- `docs/schemas/gate-contract.v1.json` or `docs/spec/policy/gate_contract_v1.md`
- `scripts/release/surface/contracts/surface-release-v1.json`

Likely modifications:

- `scripts/release/surface/release-gate.sh`
- `tools/cmd/validate-surface-runtime/main_test.go`
- `tools/cmd/validate-surface-release-state/main.go`
- `tools/cmd/validate-surface-release-state/main_test.go`
- `tools/validators/surface/surface_core.go`
- `tools/validators/surface/surface_suite_test.go`
- `tools/validators/surface/surface_morph_release.go`
- `tools/validators/surface/surface_suite_test.go`
- `tools/scriptstest/release_surface_smoke_test.go`
- `.github/workflows/ci.yml`
- `.github/workflows/release-packages.yml`
- `docs/generated/manifest.json`
- `docs/spec/core/current_supported_surface.md`
- `docs/release/surface/surface_v1_release_contract.md`
- `docs/release/surface/surface_v1_release_notes.md`

Do not assume this list is complete. Each task below includes investigation and verification.

## 6. Contract Shape

Define `tetra.gate-contract.v1` with these required top-level fields:

- `schema`
- `id`
- `title`
- `scope`
- `producer`
- `entrypoint`
- `fresh_report_dir_policy`
- `host_preconditions`
- `steps`
- `required_reports`
- `validators`
- `artifact_hashes`
- `claims`
- `nonclaims`
- `ci_artifacts`

Each step should include:

- `id`
- `kind`
- `command`
- `working_dir`
- `required`
- `report_outputs`
- `validator_refs`
- `host_preconditions`
- `blocked_status_policy`

Each required report should include:

- `path`
- `schema`
- `validator`
- `same_commit_required`
- `artifact_hash_required`
- `claim_refs`

The first contract must be intentionally small enough to implement safely. Do not try to model every
release gate feature in v1 before the Surface vertical slice passes.

## 7. Execution Plan

### Task 0 - Baseline and Safety Snapshot

**Goal:** Record current state before touching the release/tooling code.

**Files:** inspect only.

- `GOAL.md`
- `PLAN.md`
- `ATTEMPTS.md`
- `CONTROL.md`
- `NOTES.md`
- `git status --short --branch`

**Approach:**

- Record current branch, HEAD, dirty file count, and untracked file count in a small local note or
  implementation report.
- Confirm whether the active task should be independent from the old Actor Foundation RC100
  `GOAL.md`.
- Do not reset, clean, or delete anything.

**Verification:**

```sh
git status --short --branch
git rev-parse HEAD
```

**Done when:**

- the implementation session knows the exact starting state;
- unrelated dirty worktree changes are preserved;
- any active goal mismatch is explicitly documented.

**Notes:**

- This repo has a large dirty tree. Every implementation task must avoid accidental broad formatting
  or generated-file churn.

### Task 1 - Reproduce the Current Surface Drift

**Goal:** Lock the failing behavior before fixing it.

**Files:** inspect / test.

- `tools/validators/surface/surface_core.go`
- `tools/validators/surface/surface_suite_test.go`
- `tools/validators/surface/surface_morph_release.go`
- `tools/validators/surface/surface_suite_test.go`
- `tools/cmd/validate-surface-runtime/main.go`
- `tools/cmd/validate-surface-runtime/main_test.go`
- `tools/cmd/validate-surface-crash-report/main.go`
- `scripts/release/surface/release-gate.sh`

**Approach:**

- Re-run the focused failing tests with a repo-local Go cache.
- Capture whether the failure is still `crash_reporting` missing or whether the dirty tree has moved
  since this plan was written.
- Inspect the current release summary fixture and generated summary path.

**Verification:**

```sh
cache="$PWD/.cache/go-build-contract-refactor-t01"
tmp="$PWD/.cache/go-tmp-contract-refactor-t01"
mkdir -p "$cache" "$tmp"
GOTELEMETRY=off GOCACHE="$cache" GOTMPDIR="$tmp" \
  go test ./tools/validators/surface ./tools/cmd/validate-surface-runtime ./tools/cmd/validate-surface-crash-report -count=1
GOTELEMETRY=off GOCACHE="$cache" go clean -cache
rm -rf "$tmp"
```

**Done when:**

- the exact current failure is reproduced or the test is confirmed already fixed by intervening
  changes;
- the next task has a concrete target.

**Notes:**

- If the failure has changed, update this plan or add an implementation note before editing code.

### Task 2 - Define the Minimal Gate Contract Schema

**Goal:** Create the smallest useful contract format for release gates.

**Files:** add / modify.

- `tools/internal/gatecontract/contract.go`
- `tools/internal/gatecontract/validate.go`
- `tools/internal/gatecontract/contract_test.go`
- `docs/spec/policy/gate_contract_v1.md`

**Approach:**

- Implement Go structs for `tetra.gate-contract.v1`.
- Add strict JSON decoding with unknown-field rejection.
- Validate required fields, unique step IDs, unique report paths, valid validator references, and
  required artifact hash settings.
- Keep the schema generic enough for Surface now and Memory/RAM/Actor later.
- Document field meanings and the nonclaim policy.

**Verification:**

```sh
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t02" \
  go test ./tools/internal/gatecontract -count=1
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t02" go clean -cache
```

**Done when:**

- valid minimal contracts pass;
- missing required fields fail with useful diagnostics;
- unknown fields fail;
- duplicate step/report IDs fail;
- docs explain how the contract prevents claim drift.

**Notes:**

- Do not add a JSON schema dependency unless there is already a repo-approved pattern. Prefer
  standard-library JSON plus Go tests.

### Task 3 - Add Report Directory and Artifact Helpers

**Goal:** Stop duplicating fresh report directory and artifact hash logic.

**Files:** add / modify.

- `tools/internal/reportdir/`
- `tools/internal/artifacts/`
- existing shell helper candidates:
  - `scripts/release/surface/report-dir-guard.sh`
  - `scripts/ci/test-all.sh`
  - release scripts that duplicate report-dir checks

**Approach:**

- Add Go helpers for safe repo-relative report directories.
- Add tests for rejecting absolute paths, `..`, symlinks, dash-prefixed paths, non-directories, and
  non-empty report directories.
- Add artifact helper functions for checking required report files and invoking existing artifact
  hash validator behavior.
- Do not migrate every script yet. First use these helpers from the new runner.

**Verification:**

```sh
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t03" \
  go test ./tools/internal/reportdir ./tools/internal/artifacts -count=1
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t03" go clean -cache
```

**Done when:**

- helper packages pass focused tests;
- behavior matches stricter existing report-dir policies;
- no existing shell behavior is weakened.

**Notes:**

- Keep existing shell guard in place until a script actually migrates.

### Task 4 - Implement a Dry-Run Gate Runner

**Goal:** Provide a contract consumer before executing real release steps.

**Files:** add / modify.

- `tools/cmd/run-gate/main.go`
- `tools/cmd/run-gate/main_test.go`
- `tools/internal/gatecontract/`
- `tools/internal/reportdir/`

**Approach:**

- Implement:
  - `--contract PATH`
  - `--report-dir DIR`
  - `--dry-run`
  - `--json`
- Dry-run should print the resolved ordered steps, reports, validators, and artifact hash plan
  without executing release commands.
- Reject invalid contracts before producing a plan.

**Verification:**

```sh
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t04" \
  go test ./tools/cmd/run-gate ./tools/internal/gatecontract ./tools/internal/reportdir -count=1
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t04" go clean -cache
```

**Done when:**

- dry-run works for a fixture contract;
- invalid contract fails before execution;
- tests assert stable JSON output for the dry-run plan.

**Notes:**

- Do not make the runner execute arbitrary shell commands until the dry-run contract is tested.

### Task 5 - Create the Surface Release Contract

**Goal:** Describe the existing Surface release gate in a machine-readable contract without changing
behavior yet.

**Files:** add / modify.

- `scripts/release/surface/contracts/surface-release-v1.json`
- `tools/cmd/run-gate/main_test.go`
- `tools/scriptstest/release_surface_smoke_test.go`

**Approach:**

- Encode the current Surface release steps from `scripts/release/surface/release-gate.sh`.
- Include every required report currently checked by the gate.
- Include the new `surface-crash-report.json` requirement only after Task 6 creates the producing
  step.
- Add a script test that loads the contract and checks:
  - required release summary report exists in the contract;
  - release-state validator is listed;
  - artifact hashes are required;
  - host unsupported target reports are listed;
  - Surface v1 nonclaims are present.

**Verification:**

```sh
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t05" \
  go test ./tools/cmd/run-gate ./tools/scriptstest -run 'Surface|Contract|Release' -count=1
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t05" go clean -cache
```

**Done when:**

- the contract validates;
- dry-run prints the expected Surface release plan;
- tests can check the contract without fragile string-matching long bash snippets.

**Notes:**

- Keep `release-gate.sh` as the public entrypoint.

### Task 6 - Fix the `crash_reporting` Vertical End-to-End

**Goal:** Make `surface-crash-report-v1` a real release evidence slice.

**Files:** add / modify.

- `scripts/release/surface/surface-crash-report-smoke.sh`
- `scripts/release/surface/release-gate.sh`
- `scripts/release/surface/contracts/surface-release-v1.json`
- `tools/cmd/validate-surface-crash-report/main.go`
- `tools/validators/surface/surface_morph_release.go`
- `tools/validators/surface/surface_suite_test.go`
- `tools/validators/surface/surface_core.go`
- `tools/validators/surface/surface_suite_test.go`
- `tools/cmd/validate-surface-runtime/main_test.go`
- `tools/cmd/validate-surface-release-state/main.go`
- `tools/cmd/validate-surface-release-state/main_test.go`

**Approach:**

- Ensure the crash report validator has its own CLI and focused tests.
- Add a smoke script or runner step that writes `surface-crash-report.json` with
  `tetra.surface.crash-report.v1` evidence.
- Add `"crash_reporting": "surface-crash-report-v1"` to the release summary writer.
- Add `surface-crash-report.json` to required release reports.
- Add `validate-surface-crash-report` invocation to the release flow.
- Update release-state validation to require and validate crash report evidence.
- Update tests and fixtures so release summary validation, runtime validation, release-state
  validation, and script contract checks all agree.

**Verification:**

```sh
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t06" \
  go test ./tools/validators/surface ./tools/cmd/validate-surface-runtime ./tools/cmd/validate-surface-release-state ./tools/cmd/validate-surface-crash-report -count=1
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t06" go clean -cache
```

Then:

```sh
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t06-scripts" \
  go test ./tools/scriptstest -run 'Surface|Crash|Contract|Release' -count=1
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t06-scripts" go clean -cache
```

**Done when:**

- the focused Surface validator/runtime/release-state packages pass;
- the release summary includes `crash_reporting`;
- missing crash report evidence fails;
- wrong crash report claim fails;
- script tests prove the contract includes the crash report step.

**Notes:**

- This is the first real vertical slice. Do not migrate unrelated Surface features in the same task.

### Task 7 - Execute the Surface Release Contract Locally

**Goal:** Prove the contract is executable through the existing public entrypoint.

**Files:** modify if needed.

- `scripts/release/surface/release-gate.sh`
- `tools/cmd/run-gate/main.go`
- `scripts/release/surface/contracts/surface-release-v1.json`

**Approach:**

- Keep `bash scripts/release/surface/release-gate.sh --report-dir DIR` working.
- Either:
  - have the script call `go run ./tools/cmd/run-gate --contract ...`, or
  - have the script execute the current commands while tests compare it to the contract.
- Prefer the first option after dry-run and crash vertical tests pass.
- Preserve host precondition behavior: unavailable Wayland/Chromium should be a truthful
  blocker/failure according to existing release policy, not a silent skip.

**Verification:**

```sh
report_dir="reports/contract-refactor-surface-release-$(date -u +%Y%m%d-%H%M%S)"
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t07" \
  bash scripts/release/surface/release-gate.sh --report-dir "$report_dir"
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t07" go clean -cache
```

If host preconditions are unavailable, record the precise blocked evidence and run the strongest
available dry-run / focused validator proof:

```sh
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t07-dry" \
  go run ./tools/cmd/run-gate --contract scripts/release/surface/contracts/surface-release-v1.json --report-dir reports/contract-refactor-dry-run --dry-run --json
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t07-dry" go clean -cache
```

**Done when:**

- full Surface release gate passes on a host with required preconditions, or a precise blocker is
  recorded;
- dry-run contract validation passes either way;
- generated report directory contains all contract-required reports and artifact hashes.

**Notes:**

- This task may be environment-sensitive. Do not call it DONE from a dry-run alone unless the
  implementation goal explicitly allows that lower verdict.

### Task 8 - Replace Brittle Script Assertions with Contract Assertions

**Goal:** Reduce test coupling to exact bash text while preserving safety.

**Files:** modify.

- `tools/scriptstest/release_surface_smoke_test.go`
- `tools/scriptstest/surface_token_graph_gate_test.go`
- `tools/scriptstest/surface_visual_gate_test.go`
- other `tools/scriptstest/*` files only when they directly cover the migrated Surface release
  contract.

**Approach:**

- Keep a small number of entrypoint smoke assertions:
  - script exists;
  - usage text exists;
  - script calls contract runner or references the contract.
- Move detailed step/report/validator assertions to contract parsing tests.
- Keep negative tests for stale report dirs, symlinks, docs-only claims, and unsupported target
  promotion.

**Verification:**

```sh
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t08" \
  go test ./tools/scriptstest -count=1
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t08" go clean -cache
```

**Done when:**

- `tools/scriptstest` still passes;
- the tests fail when a required contract step/report/validator is removed;
- the tests no longer require large command snippets duplicated in Go strings.

**Notes:**

- Keep this scoped to Surface until the pattern is proven.

### Task 9 - Wire CI to the Same Contract

**Goal:** Make GitHub Actions use the same release contract as local scripts.

**Files:** modify.

- `.github/workflows/ci.yml`
- `.github/workflows/release-packages.yml`
- `tools/scriptstest/ci_workflow_test.go`
- `tools/scriptstest/release_packages_workflow_test.go`

**Approach:**

- Replace duplicated Surface release command details with the public script or `run-gate` invocation
  that consumes the contract.
- Ensure uploaded artifacts include all contract-required reports, including
  `surface-crash-report.json`.
- Add workflow tests that assert CI references the contract or the stable contract-backed script.

**Verification:**

```sh
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t09" \
  go test ./tools/scriptstest -run 'CI|Workflow|ReleasePackages|Surface' -count=1
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t09" go clean -cache
```

If `actionlint` is available or allowed by the repo's existing workflow:

```sh
GOTELEMETRY=off go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7
```

**Done when:**

- CI workflow tests pass;
- workflow upload paths match contract-required artifacts;
- there is one local/CI source of truth for the Surface release gate.

**Notes:**

- If network access for `actionlint` is unavailable, record it as not verified and rely on existing
  workflow tests.

### Task 10 - Sync Docs, Manifest, and Claims

**Goal:** Ensure docs claims match executable evidence.

**Files:** modify.

- `docs/spec/core/current_supported_surface.md`
- `docs/spec/surface/surface_v1.md`
- `docs/release/surface/surface_v1_release_contract.md`
- `docs/release/surface/surface_v1_release_notes.md`
- `docs/generated/manifest.json`
- related Surface user docs if they mention crash reporting or release gates.

**Approach:**

- Update docs to describe `surface-crash-report-v1` only within its proven scope.
- Keep unsupported/nonclaim language explicit.
- Regenerate and validate docs manifest using existing repo commands.
- Avoid broad "production" language unless the release gate evidence proves it.

**Verification:**

```sh
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t10" \
  go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t10" \
  go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t10" \
  go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t10" go clean -cache
```

**Done when:**

- docs mention the new evidence only where backed by reports;
- generated manifest is current;
- `verify-docs` passes.

**Notes:**

- If docs generation changes many unrelated files, stop and inspect before accepting the churn.

### Task 11 - Split Surface Validators by Schema

**Goal:** Reduce the `tools/validators/surface/surface_core.go` monolith after the vertical slice is
stable.

**Files:** modify / add inside `tools/validators/surface/`.

Candidate files:

- `release_summary.go`
- `runtime_report.go`
- `text_input.go`
- `app_shell.go`
- `security_permission.go`
- `performance_budget.go`
- `package.go`
- `reference_apps.go`
- `crash_report.go`
- matching focused tests.

**Approach:**

- Move types and validation functions by schema, preserving package name `surface` to avoid import
  churn.
- Do not change exported API behavior.
- Keep each move mechanical and verified.
- Avoid splitting unrelated compiler or CLI packages in this task.

**Verification:**

```sh
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t11" \
  go test ./tools/validators/surface -count=1
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t11" go clean -cache
```

**Done when:**

- `tools/validators/surface` passes;
- no exported validator disappears;
- `report.go` is reduced to shared runtime report logic or compatibility facade content;
- schema-specific tests live near schema-specific validators.

**Notes:**

- This is intentionally after the crash vertical fix. Splitting first would mix architecture
  movement with behavior repair.

### Task 12 - Migrate One More Surface Gate to the Contract Pattern

**Goal:** Prove the pattern is reusable beyond `release-gate.sh`.

**Files:** choose one after inspection.

Candidates:

- `scripts/release/surface/morph-gate.sh`
- `scripts/release/surface/block-system-gate.sh`
- `scripts/release/surface/visual-gate.sh`

**Approach:**

- Add a second contract for one smaller Surface gate.
- Reuse `gatecontract`, `reportdir`, `artifacts`, and `run-gate`.
- Convert tests from detailed shell string checks to contract checks.

**Verification:**

```sh
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t12" \
  go test ./tools/cmd/run-gate ./tools/scriptstest ./tools/validators/surface -run 'Morph|Block|Visual|Contract|Surface' -count=1
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t12" go clean -cache
```

**Done when:**

- a second gate uses the same contract machinery;
- duplicated shell helpers decrease;
- tests show both contracts are valid.

**Notes:**

- Choose the smallest gate that gives meaningful proof. Do not migrate all Surface scripts in this
  task.

### Task 13 - Broaden to RAM, Memory, and Actor Gates

**Goal:** Apply the proven pattern to non-Surface release gates.

**Files:** inspect before modification.

Likely candidates:

- `scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh`
- `scripts/release/post_v0_4/memory-100-prod-stable-gate.sh`
- `scripts/release/post_v0_4/actor-runtime-foundation-linux-x64-gate.sh`
- related `tools/cmd/validate-*`
- related `tools/scriptstest/*`
- related CI workflow jobs.

**Approach:**

- Create one contract per gate.
- Keep existing scripts as compatibility entrypoints.
- Reuse the same report-dir, artifact, validator, and CI artifact patterns.
- Migrate one gate at a time.

**Verification:**

Use focused tests per gate first, then:

```sh
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t13" \
  go test ./tools/scriptstest ./tools/cmd/... ./tools/validators/... -count=1
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-t13" go clean -cache
```

If `./tools/cmd/...` or `./tools/validators/...` is too broad or unsupported by module boundaries,
replace with the exact packages discovered during the implementation pass.

**Done when:**

- at least RAM, Memory100, and Actor Foundation gates have contracts;
- CI references contract-backed entrypoints;
- validators and docs for those gates agree with the contracts.

**Notes:**

- This task may need to be split into separate implementation packets.

### Task 14 - Final Local Validation Ladder

**Goal:** Prove the refactor did not weaken the repo.

**Files:** no direct edits unless failures require fixes.

**Approach:**

- Run formatting / shell syntax / focused contract tests / broader package tests.
- Use persistent repo-local Go caches and clean them after evidence runs.
- Record any host-dependent release gate blockers honestly.

**Verification:**

```sh
find scripts -name '*.sh' -print0 | xargs -0 -n1 bash -n
```

```sh
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-final" \
  go test ./tools/internal/... ./tools/cmd/run-gate ./tools/scriptstest ./tools/validators/surface ./tools/cmd/validate-surface-runtime ./tools/cmd/validate-surface-release-state ./tools/cmd/validate-surface-crash-report -count=1
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-final" go clean -cache
```

If runtime cost is acceptable:

```sh
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-all" \
  bash scripts/ci/test.sh
GOTELEMETRY=off GOCACHE="$PWD/.cache/go-build-contract-refactor-all" go clean -cache
```

**Done when:**

- shell syntax passes;
- focused contract and Surface validator tests pass;
- broader CI test script passes or any blocker is specific and documented;
- no known drift remains in the migrated gates.

**Notes:**

- Do not claim full repo readiness if only focused tests pass.

### Task 15 - Graphify Update and Handoff

**Goal:** Keep repository graph artifacts current after code changes.

**Files:** generated graph artifacts if the repo expects them.

**Approach:**

- Run Graphify update after modifications.
- Inspect generated changes before accepting them.
- Record what changed and what remains out of scope.

**Verification:**

```sh
graphify update .
```

**Done when:**

- Graphify update completes or a precise tool blocker is recorded;
- final handoff lists migrated gates, passing checks, and remaining unmigrated gates.

**Notes:**

- If Graphify updates large ignored artifacts only, confirm whether they should remain untracked.

## 8. Suggested Implementation Order

Recommended packet order:

1. Tasks 0-1: baseline and reproduce.
2. Tasks 2-4: contract foundation and dry-run runner.
3. Tasks 5-7: Surface release contract plus `crash_reporting` end-to-end.
4. Tasks 8-10: script tests, CI, docs/manifest sync.
5. Tasks 11-12: monolith split and second Surface gate migration.
6. Task 13: RAM/Memory/Actor contract migration.
7. Tasks 14-15: final validation and Graphify update.

Do not start Task 11 before Task 6 is green. Otherwise behavior repair and file movement will blur
together.

## 9. Acceptance Criteria

The refactor is DONE only when all of these are true:

- Surface `crash_reporting` is produced, validated, included in release summary, required by
  release-state validation, and covered by tests.
- At least one Surface release gate executes from a machine-readable contract.
- CI and local scripts use the same contract-backed entrypoint.
- Artifact hash validation remains mandatory.
- Host preconditions are explicit and cannot silently skip required release evidence.
- Script tests validate contract semantics rather than copying long bash command strings.
- Surface validator monolith has begun moving into schema-owned files without changing public
  validator behavior.
- Focused validation passes.
- Any broad validation not run is listed as not verified.

## 10. Remaining Risks

- The current dirty tree may include concurrent user or agent changes. Every task must re-check
  files before editing.
- Surface release gates depend on host features such as Wayland/display and Chromium-compatible
  browser execution. End-to-end release execution may be blocked by environment, not code.
- Full migration of RAM, Memory, and Actor gates is too large for one safe code batch. Treat each
  gate as its own follow-up packet after the Surface pattern is proven.
- CI proof may be blocked by repository permissions, account state, billing, or runner availability.
- The existing `tools/scriptstest` package is already heavy. More contract tests should reduce
  fragility, not add slow fake-repo setup unnecessarily.

## 11. Execution Mode Recommendation

Use `subagent-driven-development` only if the work is split into isolated packets with review after
each task. Otherwise use `executing-plans` with checkpointed batches:

- Batch A: Tasks 0-4.
- Batch B: Tasks 5-7.
- Batch C: Tasks 8-10.
- Batch D: Tasks 11-12.
- Batch E: Task 13 per gate.
- Batch F: Tasks 14-15.

Each batch must end with evidence, not just code changes.
