# Systemic Contract Platform Refactor Master Plan

**Status:** planning document, not implementation evidence. **Date:** 2026-06-16. **Scope:**
`scripts`, `tools/cmd`, `tools/internal`, `tools/validators`, `tools/scriptstest`, docs claims,
generated manifests, CI workflows, and release artifact evidence. **Problem statement:** release
validation has grown into a hidden platform across shell scripts, Go commands, validators, docs,
generated reports, and CI without one canonical manifest/contract layer. **Primary outcome:** make
release claims contract-driven, locally repeatable, CI-repeatable, reviewable, and hard to
overclaim.

This plan is intentionally more detailed than a normal refactor checklist. The failure mode here is
systemic drift: one local script passes, one validator says something else, docs imply a third
truth, and CI runs a fourth path. The fix has to make those surfaces share one contract instead of
merely patching whichever test is currently red.

## 1. Executive Goal

Create a single release/validation contract layer that owns:

- gate identity and scope;
- host preconditions;
- required steps;
- required reports;
- validators and their expected inputs;
- artifact hash policy;
- docs claims and nonclaims;
- CI artifact upload paths;
- evidence freshness and report directory policy;
- local and remote verification commands.

When this plan is complete, the following must be true:

- every important release gate has a machine-readable contract;
- shell scripts are thin entrypoints around that contract;
- `tools/cmd` validators validate contract outputs, not undocumented script side effects;
- `tools/scriptstest` checks contract wiring and deterministic plans instead of brittle shell
  snippets;
- docs claims are verified against contract-backed manifests or reports;
- CI invokes the same contract entrypoints as local development;
- remote CI and release-package evidence can be cited without hand-waving;
- no user-facing production claim exists without a contract-backed report and validator.

## 2. Current Facts Inspected Before Writing This Plan

These are observed in the current checkout and must be re-verified at execution time before any
edit, because the tree is dirty and `main` is behind `origin/main`.

- `AGENTS.md` requires Ukrainian communication, explicit completion levels, no false `DONE`, and
  persistent Go caches outside `/tmp`.
- `graphify-out/GRAPH_REPORT.md` exists and reports a large corpus: 31612 nodes, 77124 edges, 1743
  communities, built from commit `95bfd4a8`.
- The workspace uses Go 1.20 and `go.work` includes `.`, `./compiler`, `./cli`, and `./tools`.
- `docs/plan/` already contains related plans:
  - Directory: `contract-driven-release-validation/`
    File: `2026-06-16-contract-driven-release-validation-refactor-plan.md`.
  - Directory: `contract-driven-release-validation/`
    File: `2026-06-16-contract-driven-release-validation-refactor-completion-audit.md`.
  - Directory: `contract-driven-release-validation/`
    File: `2026-06-16-contract-driven-release-validation-pr-hardening-remote-ci-plan.md`.
- `.github/workflows/ci.yml` is currently manual `workflow_dispatch` and runs: shell syntax,
  actionlint, multi-OS tests, docs manifest verification, smoke reports, quick/stabilization
  wrappers, and release readiness jobs.
- `.github/workflows/release-packages.yml` builds release archives and invokes multiple release
  gates, including Surface, Memory, RAM, Actor runtime, and package artifact upload.
- Candidate contract-related paths are present in the current dirty checkout:
  - `tools/internal/gatecontract/`;
  - `tools/internal/reportdir/`;
  - `tools/internal/artifacts/`;
  - `tools/cmd/run-gate/`;
  - `scripts/release/surface/contracts/`;
  - `scripts/release/post_v0_4/contracts/`.
- Existing important validation surfaces include:
  - `tools/cmd/gen-manifest`;
  - `tools/cmd/validate-manifest`;
  - `tools/cmd/verify-docs`;
  - `tools/cmd/validate-artifact-hashes`;
  - `tools/cmd/validate-surface-release-state`;
  - `tools/cmd/validate-surface-runtime`;
  - `tools/cmd/validate-surface-final-readiness`;
  - many domain-specific validators under `tools/cmd/validate-*`.
- Existing release script surfaces include:
  - `scripts/ci/test.sh`;
  - `scripts/ci/test-all.sh`;
  - `scripts/ci/toon-format-check.sh`;
  - `scripts/release/surface/release-gate.sh`;
  - `scripts/release/surface/product-gate.sh`;
  - `scripts/release/surface/*-smoke.sh`;
  - `scripts/release/post_v0_4/*-gate.sh`;
  - `scripts/release/post_v0_4/*-smoke.sh`;
  - `scripts/release/packages/build-release-archives.sh`.
- `tools/scriptstest` is the main shell/CI/release wiring test package and already covers many
  scripts and workflows.

## 3. Diagnosis

The repository has multiple valid-looking sources of truth:

- shell scripts define what gets executed;
- Go commands define semantic validation;
- validators define report requirements;
- docs define user-facing claims;
- generated manifests define doc state;
- CI workflows define merge/release reality;
- release-package jobs define artifact reality.

Those layers currently know about each other through duplicated paths, string checks, implicit
report layouts, and historical conventions. That makes the system fragile in four ways:

- a new report can be generated but never validated;
- a validator can pass while docs still overclaim;
- CI can run a different path than local release scripts;
- tests can lock onto shell text instead of stable behavior.

The correct fix is not a broad rename or a one-off Surface patch. The correct fix is to introduce a
durable contract boundary and migrate gates through it in small vertical slices.

## 4. Non-Goals

This plan must not be used as permission to rewrite the whole project in one chaotic batch.

Out of scope:

- changing Tetra language semantics;
- weakening any release gate to make CI green;
- deleting historical release scripts before replacement entrypoints are proven;
- pretending local validation is remote CI proof;
- claiming a clean PR while unrelated dirty files are mixed in;
- moving compiler, parser, CLI, UI runtime, memory, or actor logic only for aesthetics;
- storing Go build caches under `/tmp`;
- pushing branches, opening PRs, or dispatching workflows without explicit user authorization.

## 5. Target Architecture

The desired architecture is:

```text
contract JSON
  -> contract loader / schema validation
  -> deterministic dry-run plan
  -> runner invokes existing gate entrypoint
  -> scripts produce reports in a fresh report dir
  -> validators validate the reports named by the contract
  -> artifact hashes bind reports to evidence
  -> docs and manifest verification consume the same truth
  -> CI uploads the same contract-named artifacts
```

### 5.1 Contract Layer

Canonical contract paths:

- `scripts/release/surface/contracts/*.json`
- `scripts/release/post_v0_4/contracts/*.json`
- future path if needed: `docs/generated/contracts/*.json`

Core Go package:

- `tools/internal/gatecontract/`

Required contract fields:

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

Contract rules:

- IDs are stable and unique.
- Every required report maps to exactly one validator.
- Every production claim maps to at least one required report.
- Every unsupported target or not-yet-supported feature is a nonclaim.
- Artifact hash policy is explicit.
- Host preconditions are explicit and machine-readable enough to display in CI failure messages.
- A missing host precondition must fail with a structured reason, not silently skip a production
  claim.

### 5.2 Runner Layer

Canonical command:

- `tools/cmd/run-gate`

Required behavior:

- `--contract PATH` loads and validates a contract.
- `--report-dir DIR` enforces the fresh report directory policy.
- `--dry-run` prints a deterministic human-readable plan.
- `--json` prints the dry-run plan as JSON.
- normal execution calls the contract `entrypoint`.
- recursion guard prevents scripts from accidentally calling `run-gate` again forever.
- runner injects environment variables that scripts can verify:
  - `TETRA_RUN_GATE_CONTRACT_EXEC`;
  - `TETRA_RUN_GATE_CONTRACT_ID`;
  - `TETRA_RUN_GATE_REPORT_DIR`.

### 5.3 Report Directory and Artifacts Layer

Core packages:

- `tools/internal/reportdir/`
- `tools/internal/artifacts/`

Rules:

- release gates must write to fresh report directories;
- report paths must be repo-relative unless a tool explicitly documents an absolute path use case;
- artifact hashes must be generated and then validated;
- generated evidence must not mix old reports with new ones;
- CI artifact upload paths must match contract `ci_artifacts`.

### 5.4 Validators Layer

Validators remain semantic authorities. The refactor should not turn shell scripts into validators.

Relevant validator packages and commands include:

- `tools/validators/surface/`
- `tools/validators/memoryprod/`
- `tools/validators/postv04prod/`
- `tools/validators/actorprod/`
- `tools/validators/actordist/`
- `tools/validators/parallelprod/`
- `tools/validators/compilerprod/`
- `tools/cmd/validate-surface-runtime`
- `tools/cmd/validate-surface-release-state`
- `tools/cmd/validate-surface-final-readiness`
- `tools/cmd/validate-memory-production`
- `tools/cmd/validate-actor-runtime-foundation`
- `tools/cmd/validate-artifact-hashes`
- `tools/cmd/validate-manifest`
- `tools/cmd/verify-docs`

The end state is not one giant validator. The end state is a shared contract shape with
domain-specific validators.

### 5.5 Docs and Manifest Layer

Canonical doc truth surfaces:

- `docs/generated/manifest.json`
- `docs/generated/v1_0/`
- `docs/spec/core/current_supported_surface.md`
- release docs under `docs/release/`
- user docs that make supported-target or production-readiness claims.

Rules:

- docs claims must be checked by `tools/cmd/verify-docs`;
- generated manifest must be deterministic;
- `git diff --exit-code -- docs/generated/manifest.json` must be part of CI;
- production claims must point to contract-backed evidence;
- nonclaims must be visible in docs where they prevent user confusion.

### 5.6 CI Layer

Relevant workflows:

- `.github/workflows/ci.yml`
- `.github/workflows/release-packages.yml`

Rules:

- CI must invoke the same contract runner or script wrapper as local execution;
- CI must upload the evidence paths named by the contract;
- release-package dry-run evidence is not a production release claim;
- workflow dispatch results are `INTEGRATION` or `END_TO_END` evidence only when the exact run URL,
  commit SHA, workflow, and artifacts are recorded.

## 6. Completion Levels

Use these levels throughout execution:

- `LOCAL`: one package, script, command, or validator passes.
- `INTEGRATION`: affected scripts, validators, docs, and CI config are wired together locally.
- `END_TO_END`: the real release flow works locally and remotely with artifacts.
- `FINAL`: all acceptance criteria pass, remote evidence exists, docs are accurate, and no known
  blocker remains.

Do not mark the overall refactor `DONE` from a single green validator, a single green release gate,
or a local dry-run.

## 7. PR and Execution Strategy

The safest implementation should be split into reviewable vertical PRs.

### PR 1: Contract Core

Scope:

- `tools/internal/gatecontract/`
- `tools/internal/reportdir/`
- `tools/internal/artifacts/`
- `tools/cmd/run-gate/`
- minimal contract fixtures;
- focused unit tests.

Exit criteria:

- contract loader rejects malformed contracts;
- dry-run plan is deterministic;
- fresh report dir policy is enforced;
- artifact hash command plan is deterministic;
- no release scripts are required to change yet except tiny fixtures if needed.

### PR 2: Surface Release Vertical Slice

Scope:

- `scripts/release/surface/contracts/surface-release-v1.json`
- `scripts/release/surface/release-gate.sh`
- Surface report validators and tests required by the contract;
- `tools/scriptstest` wiring tests for Surface contract execution.

Exit criteria:

- `surface-release-v1` can run through `run-gate`;
- dry-run plan lists required reports, validators, claims, and CI artifacts;
- final Surface release gate writes artifact hashes and validates them;
- docs claims do not exceed the contract.

### PR 3: Post-v0.4 Production Gates

Scope:

- `scripts/release/post_v0_4/contracts/*.json`
- Memory/RAM/Actor gate wrappers;
- domain validator alignment;
- script tests for contract wiring.

Exit criteria:

- Memory/RAM/Actor gates have contract-backed claims and nonclaims;
- old script entrypoints continue to work;
- `release-packages.yml` can invoke the contract-backed entrypoints.

### PR 4: Docs, Manifest, and CI Hardening

Scope:

- `tools/cmd/gen-manifest`
- `tools/cmd/validate-manifest`
- `tools/cmd/verify-docs`
- `docs/generated/manifest.json`
- `.github/workflows/ci.yml`
- `.github/workflows/release-packages.yml`

Exit criteria:

- docs claims are contract-linked;
- generated manifest is deterministic;
- CI uploads contract-named evidence;
- remote CI proof exists.

### PR 5: Cleanup and Deprecation

Scope:

- remove duplicated checks after all callers migrate;
- delete dead shell snippets only when tests prove no caller remains;
- update release docs to describe the contract workflow.

Exit criteria:

- no duplicated undocumented release truth remains;
- all public entrypoints still work or have documented replacement paths;
- stale generated evidence is not tracked as current proof.

## 8. Detailed Work Plan

### Task 0: Baseline and Dirty-Tree Inventory

**Goal:** prevent accidental mixing of unrelated changes with the contract refactor.

**Inspect:**

- `git status --short --branch`
- `git diff --name-status`
- `git ls-files --others --exclude-standard`
- `docs/plan/`
- `.github/workflows/`
- `scripts/`
- `tools/cmd/`
- `tools/internal/`
- `tools/validators/`
- `tools/scriptstest/`

**Approach:**

1. Record current branch, HEAD, upstream, and dirty count.
2. Bucket every changed/untracked path as:
   - contract core;
   - Surface vertical slice;
   - post-v0.4 gate migration;
   - docs/manifest/CI;
   - evidence artifact;
   - unrelated or ambiguous.
3. Do not delete or reset unrelated files.
4. If a clean PR is required, create an isolated worktree from `origin/main` before applying only
   the selected scope.

**Verification:**

```sh
git status --short --branch
git rev-parse HEAD
git diff --name-status
git ls-files --others --exclude-standard
```

**Done when:**

- every dirty path has an owner bucket;
- the intended first PR scope is explicit;
- unrelated changes have a safe handling plan.

### Task 1: Freeze `tetra.gate-contract.v1`

**Goal:** define the contract schema before migrating more gates.

**Modify or add:**

- `tools/internal/gatecontract/contract.go`
- `tools/internal/gatecontract/validate.go`
- `tools/internal/gatecontract/contract_test.go`
- optional: `docs/spec/policy/gate_contract_v1.md`
- optional: `docs/schemas/gate-contract.v1.json`

**Approach:**

1. Confirm existing `Contract`, `Step`, `RequiredReport`, `Validator`, `ArtifactHashPolicy`,
   `Claim`, `Nonclaim`, and `CIArtifact` fields.
2. Make required fields explicit and strictly decoded.
3. Add negative tests for:
   - empty JSON;
   - unknown fields;
   - missing top-level fields;
   - missing nested fields;
   - duplicate validator IDs;
   - duplicate claim IDs;
   - duplicate report paths;
   - report references to missing validators;
   - report claim refs to missing claims;
   - enabled artifact hashes with reports that opt out.
4. Add positive fixture tests for a small valid contract.

**Verification:**

```sh
go test ./tools/internal/gatecontract -count=1
```

**Done when:**

- invalid contracts fail before any script executes;
- valid contracts can be loaded deterministically;
- schema/version errors are clear enough for CI logs.

### Task 2: Harden Fresh Report Directory Policy

**Goal:** prevent stale report reuse and mixed evidence.

**Modify or add:**

- `tools/internal/reportdir/`
- `scripts/release/surface/report-dir-guard.sh`
- tests in `tools/internal/reportdir/`

**Approach:**

1. Define allowed report directory forms.
2. Reject empty, root, parent traversal, and already-populated directories.
3. Preserve script compatibility where existing gates pass `--report-dir`.
4. Return clear error messages with the rejected path and policy reason.
5. Ensure shell guard and Go guard enforce the same policy.

**Verification:**

```sh
go test ./tools/internal/reportdir -count=1
bash -n scripts/release/surface/report-dir-guard.sh
```

**Done when:**

- fresh report dir policy is enforced in both Go runner and shell wrappers;
- stale evidence cannot be silently reused.

### Task 3: Build Artifact Hash Policy

**Goal:** bind reports to immutable evidence.

**Modify or add:**

- `tools/internal/artifacts/`
- `tools/cmd/validate-artifact-hashes/`
- contract fixtures that require artifact hashes.

**Approach:**

1. Centralize the command plan for writing and validating artifact hashes.
2. Require a deterministic manifest path such as `<report-dir>/artifact-hashes.json`.
3. Make dry-run show the exact write and validate commands.
4. Ensure validators reject stale or missing artifact hash entries when the contract requires them.

**Verification:**

```sh
go test ./tools/internal/artifacts ./tools/cmd/validate-artifact-hashes -count=1
```

**Done when:**

- artifact hash generation and validation have one reusable command plan;
- contract-required reports cannot pass without required hash coverage.

### Task 4: Implement and Lock `tools/cmd/run-gate`

**Goal:** make local and CI execution enter through one runner.

**Modify or add:**

- `tools/cmd/run-gate/main.go`
- `tools/cmd/run-gate/main_test.go`

**Approach:**

1. Support `--contract`, `--report-dir`, `--dry-run`, and `--json`.
2. Validate the contract and report dir before execution.
3. In dry-run mode, emit:
   - contract identity;
   - resolved report dir;
   - steps;
   - required reports;
   - validators;
   - artifact hash command plan;
   - claims;
   - nonclaims;
   - CI artifacts.
4. In execution mode, call the contract entrypoint with guarded environment.
5. Refuse absolute entrypoints, parent traversal, dash-prefixed entrypoints, and recursive runner
   calls.
6. Keep `stdout` and `stderr` behavior deterministic in tests.

**Verification:**

```sh
go test ./tools/cmd/run-gate -count=1
go run ./tools/cmd/run-gate \
  --contract scripts/release/surface/contracts/surface-release-v1.json \
  --report-dir reports/run-gate-smoke \
  --dry-run \
  --json
```

**Done when:**

- dry-run works without executing gate steps;
- execution path can call a guarded script entrypoint;
- bad contracts and bad paths fail before side effects.

### Task 5: Create Contract Registry and Naming Rules

**Goal:** avoid contracts becoming another pile of JSON files.

**Modify or add:**

- `docs/spec/policy/gate_contract_v1.md` or equivalent spec doc;
- optional generated registry under `docs/generated/`;
- contract fixtures in `scripts/release/**/contracts/`.

**Approach:**

1. Define contract ID naming:
   - `surface-release-v1`;
   - `morph-gate`;
   - `memory-100-prod-stable-linux-x64`;
   - `ram-contract-linux-x64`;
   - `actor-runtime-foundation-linux-x64`.
2. Define scope naming:
   - product/release scope;
   - target scope;
   - experimental vs production status.
3. Add a validator or test that contract IDs are unique across the repository.
4. Document how a new gate adds a contract and tests.

**Verification:**

```sh
go test ./tools/internal/gatecontract ./tools/cmd/run-gate -count=1
```

If a registry tool is added:

```sh
go test ./tools/cmd/<registry-validator> -count=1
```

**Done when:**

- contract naming is documented and testable;
- adding a new gate has a clear checklist.

### Task 6: Migrate Surface Release Gate

**Goal:** prove the contract architecture on the largest drift surface.

**Modify:**

- `scripts/release/surface/contracts/surface-release-v1.json`
- `scripts/release/surface/release-gate.sh`
- `scripts/release/surface/report-dir-guard.sh`
- Surface smoke scripts only when contract wiring requires it.
- `tools/validators/surface/`
- `tools/cmd/validate-surface-runtime/`
- `tools/cmd/validate-surface-release-state/`
- `tools/cmd/validate-surface-final-readiness/`
- `tools/scriptstest/release_surface_gate_wiring_test.go`
- related `tools/scriptstest/surface_*` tests.

**Approach:**

1. Make `release-gate.sh` call `run-gate` unless it is already in guarded contract execution mode.
2. Make the script refuse mismatched `TETRA_RUN_GATE_CONTRACT_ID`.
3. Ensure the contract enumerates every required Surface report:
   - release summary;
   - headless release evidence;
   - linux-x64 real-window evidence;
   - wasm32-web browser evidence;
   - app shell evidence;
   - text input evidence;
   - toolkit evidence;
   - accessibility evidence;
   - template/reference/package/crash/i18n/widget evidence where claimed;
   - block system and Morph subgate reports when part of release scope;
   - artifact hashes.
4. Map every production claim to required reports.
5. Map macOS/Windows build-only or unsupported runtime status to nonclaims.
6. Make validators accept only fresh, contract-shaped evidence.
7. Keep older `bash scripts/release/surface/release-gate.sh --report-dir DIR` entrypoint working.

**Verification:**

```sh
go test \
  ./tools/validators/surface \
  ./tools/cmd/validate-surface-runtime \
  ./tools/cmd/validate-surface-release-state \
  ./tools/cmd/validate-surface-final-readiness \
  -count=1
SURFACE_RE='ReleaseSurfaceGate|SurfaceRelease|Surface.*Contract|Surface.*ReportDir'
go test ./tools/scriptstest -run "$SURFACE_RE" -count=1
bash scripts/release/surface/release-gate.sh \
  --report-dir reports/contract-platform/surface-release-v1
go run ./tools/cmd/validate-artifact-hashes \
  --manifest reports/contract-platform/surface-release-v1/artifact-hashes.json
```

**Done when:**

- Surface release gate is contract-backed end to end;
- old entrypoint still works;
- required reports and claims are enumerated in the contract;
- validators reject missing/stale evidence;
- artifact hashes validate.

### Task 7: Migrate Morph and Surface Subgates

**Goal:** avoid Surface release being contract-driven only at the top level while subgates still
drift.

**Modify:**

- `scripts/release/surface/contracts/morph-gate.json`
- `scripts/release/surface/morph-gate.sh`
- `scripts/release/surface/block-system-gate.sh`
- `scripts/release/surface/visual-gate.sh`
- `scripts/release/surface/surface-docs-claims-gate.sh`
- related validators under `tools/cmd/validate-surface-*` and `tools/validators/surface/`.

**Approach:**

1. Decide whether each subgate is a standalone contract or a step inside `surface-release-v1`.
2. If standalone, add a contract and dry-run test.
3. If embedded, ensure top-level contract names its required reports.
4. Avoid duplicated claim definitions between parent and subgate.

**Verification:**

```sh
go test \
  ./tools/cmd/validate-surface-morph-gate-summary \
  ./tools/cmd/validate-surface-morph-rendered-beauty \
  ./tools/cmd/validate-surface-block-report \
  -count=1
go test ./tools/scriptstest -run 'Surface.*Gate|Morph|Block|Visual|DocsClaims' -count=1
```

**Done when:**

- Surface subgates have no undocumented report requirements;
- parent and child contracts do not contradict each other.

### Task 8: Migrate Post-v0.4 Production Gates

**Goal:** make Memory/RAM/Actor post-v0.4 release gates use the same contract model.

**Modify:**

- `scripts/release/post_v0_4/contracts/memory-100-prod-stable-linux-x64.json`
- `scripts/release/post_v0_4/contracts/ram-contract-linux-x64.json`
- `scripts/release/post_v0_4/contracts/actor-runtime-foundation-linux-x64.json`
- `scripts/release/post_v0_4/memory-100-prod-stable-gate.sh`
- `scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh`
- `scripts/release/post_v0_4/actor-runtime-foundation-linux-x64-gate.sh`
- domain validators:
  - `tools/cmd/validate-memory-production`;
  - `tools/cmd/validate-memory-islands-surface-production`;
  - `tools/cmd/validate-actor-runtime-foundation`;
  - `tools/validators/memoryprod/`;
  - `tools/validators/postv04prod/`;
  - `tools/validators/actorprod/`.

**Approach:**

1. Start with dry-run-only contract coverage for each gate.
2. Wire scripts through `run-gate` one at a time.
3. Preserve existing script names and arguments.
4. Add nonclaims for unsupported targets or partial runtime promises.
5. Make release-package workflow consume the same script wrappers.

**Verification:**

```sh
go test \
  ./tools/cmd/validate-memory-production \
  ./tools/cmd/validate-memory-islands-surface-production \
  ./tools/cmd/validate-actor-runtime-foundation \
  -count=1
go test \
  ./tools/validators/memoryprod \
  ./tools/validators/postv04prod \
  ./tools/validators/actorprod \
  -count=1
POST_RE='PostV04|Memory100|RAM|ActorRuntimeFoundation|ReleasePackages'
go test ./tools/scriptstest -run "$POST_RE" -count=1
```

**Done when:**

- post-v0.4 gates have contract-backed dry-runs;
- selected gate scripts execute through `run-gate`;
- release-package workflow paths align with contract artifact paths.

### Task 9: Unify Docs Claims and Manifest Verification

**Goal:** prevent docs from claiming more than contracts prove.

**Modify:**

- `tools/cmd/gen-manifest/main.go`
- `tools/cmd/validate-manifest/`
- `tools/cmd/verify-docs/`
- `docs/generated/manifest.json`
- relevant docs under:
  - `docs/spec/`;
  - `docs/release/`;
  - `docs/user/`;
  - `README.md` if claims change.

**Approach:**

1. Inventory all docs that state current production support, target support, or release readiness.
2. Add manifest fields or docs verification rules that point claims to contract IDs or evidence
   paths.
3. Ensure unsupported or build-only targets are explicit nonclaims.
4. Regenerate manifest with `gen-manifest`.
5. Run `verify-docs`.
6. Keep generated diff narrow and explain every changed claim.

**Verification:**

```sh
go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --exit-code -- docs/generated/manifest.json
go test ./tools/cmd/verify-docs ./tools/cmd/validate-manifest -count=1
```

**Done when:**

- docs verification fails on unsupported production overclaims;
- manifest generation is deterministic;
- public docs and contracts tell the same story.

### Task 10: Rebuild `tools/scriptstest` Around Behavior

**Goal:** make script tests prove behavior and contract wiring, not fragile string coincidences.

**Modify:**

- `tools/scriptstest/ci_workflow_test.go`
- `tools/scriptstest/release_packages_workflow_test.go`
- `tools/scriptstest/release_surface_gate_wiring_test.go`
- `tools/scriptstest/test_all_release_gates_test.go`
- `tools/scriptstest/workspace_modules_test.go`
- related focused tests as needed.

**Approach:**

1. Replace broad shell text assertions with:
   - contract file existence;
   - dry-run JSON shape;
   - validator command references;
   - artifact upload path matching;
   - report-dir freshness checks;
   - old entrypoint compatibility checks.
2. Keep shell syntax tests.
3. Ensure nested `go test` commands use persistent cache directories, not `/tmp`.
4. Add focused failure fixtures for missing contracts and mismatched CI artifacts.
5. Split slow or host-dependent checks from fast deterministic checks.

**Verification:**

```sh
CI_RE='CIWorkflow|ReleasePackages|ReleaseSurfaceGate'
CI_RE="${CI_RE}|WorkspaceModules|TestAllReleaseGates"
go test ./tools/scriptstest -run "$CI_RE" -count=1
go test ./tools/scriptstest -count=1
```

**Done when:**

- script tests fail when contract wiring is broken;
- tests no longer require brittle script implementation details;
- workspace module tests do not exhaust tmpfs.

### Task 11: Refactor CI Workflows to Contract-Backed Jobs

**Goal:** make CI local-equivalent and artifact-complete.

**Modify:**

- `.github/workflows/ci.yml`
- `.github/workflows/release-packages.yml`
- maybe `scripts/ci/test-all.sh` only if CI needs a stable contract mode.

**Approach:**

1. Keep setup steps simple:
   - checkout;
   - setup Go 1.20.x unless the repo intentionally changes version;
   - bootstrap.
2. Add or update jobs that run contract dry-runs.
3. Ensure release-package workflow invokes existing script entrypoints that now call `run-gate`.
4. Upload contract-named report directories.
5. Keep dry-run package proof clearly separate from real publishing.
6. Do not change trigger semantics without explicit approval.

**Verification:**

Local:

```sh
go test ./tools/scriptstest -run 'CIWorkflow|ReleasePackages' -count=1
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7
```

Remote, only after user approval:

```sh
gh workflow run ci.yml --ref <branch>
gh run list --workflow ci.yml --limit 5
gh run watch <run-id>
gh workflow run release-packages.yml \
  --ref <branch> \
  -f version=v0.4.0 \
  -f dry_run=true \
  -f update_homebrew_tap=false
```

**Done when:**

- workflow tests pass locally;
- actionlint passes;
- remote workflow runs are recorded with run URLs and artifact names;
- no production publishing occurs during dry-run proof.

### Task 12: Establish Verification Ladder

**Goal:** make validation repeatable and cheap enough to run during review.

**Required local ladder:**

```sh
export GOTELEMETRY=off
export GOCACHE="$PWD/.cache/go-build-contract-platform"
export GOTMPDIR="$PWD/.cache/go-tmp-contract-platform"
mkdir -p "$GOCACHE" "$GOTMPDIR"

bash -n scripts/ci/test.sh
bash -n scripts/ci/test-all.sh
find scripts -name '*.sh' -print0 | xargs -0 -n1 bash -n

go test \
  ./tools/internal/gatecontract \
  ./tools/internal/reportdir \
  ./tools/internal/artifacts \
  ./tools/cmd/run-gate \
  -count=1
go test \
  ./tools/cmd/validate-artifact-hashes \
  ./tools/cmd/validate-manifest \
  ./tools/cmd/verify-docs \
  -count=1
CI_RE='CIWorkflow|ReleasePackages|ReleaseSurfaceGate'
CI_RE="${CI_RE}|WorkspaceModules|TestAllReleaseGates"
go test ./tools/scriptstest -run "$CI_RE" -count=1

go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --exit-code -- docs/generated/manifest.json
git diff --check

GOCACHE="$PWD/.cache/go-build-contract-platform" go clean -cache
rm -rf "$PWD/.cache/go-tmp-contract-platform"
```

**Surface ladder, when Surface slice is in scope:**

```sh
export GOTELEMETRY=off
export GOCACHE="$PWD/.cache/go-build-surface-contract-platform"
export GOTMPDIR="$PWD/.cache/go-tmp-surface-contract-platform"
mkdir -p "$GOCACHE" "$GOTMPDIR"

go test \
  ./tools/validators/surface \
  ./tools/cmd/validate-surface-runtime \
  ./tools/cmd/validate-surface-release-state \
  ./tools/cmd/validate-surface-final-readiness \
  -count=1
go run ./tools/cmd/run-gate \
  --contract scripts/release/surface/contracts/surface-release-v1.json \
  --report-dir reports/contract-platform/surface-release-dry-run \
  --dry-run \
  --json
bash scripts/release/surface/release-gate.sh \
  --report-dir reports/contract-platform/surface-release-v1
go run ./tools/cmd/validate-artifact-hashes \
  --manifest reports/contract-platform/surface-release-v1/artifact-hashes.json

GOCACHE="$PWD/.cache/go-build-surface-contract-platform" go clean -cache
rm -rf "$PWD/.cache/go-tmp-surface-contract-platform"
```

**Broad local ladder before PR:**

```sh
bash scripts/dev/bootstrap.sh
bash scripts/ci/test.sh
bash scripts/ci/test-all.sh \
  --quick \
  --keep-going \
  --report-dir reports/contract-platform/test-all-quick
go test ./tools/... -count=1
```

**Done when:**

- each ladder has logs or report paths;
- failures are categorized as contract-core, Surface, docs, CI, host precondition, or unrelated repo
  failure.

### Task 13: Capture Evidence and Review Handoff

**Goal:** make reviewers able to verify claims without reconstructing the whole session.

**Add or update reports under:**

- `reports/contract-platform-refactor/`

Suggested files:

- `baseline.md`
- `scope-inventory.md`
- `local-validation.md`
- `remote-ci.md`
- `release-artifacts.md`
- `review-handoff.md`
- `blockers.md`

**Approach:**

1. Record command, exit code, date, commit SHA, and relevant artifact path.
2. Keep logs out of tracked docs unless the repo policy says otherwise.
3. Summarize failures honestly.
4. State which evidence is `LOCAL`, `INTEGRATION`, `END_TO_END`, or `FINAL`.

**Verification:**

```sh
test -f reports/contract-platform-refactor/local-validation.md
git status --short reports/contract-platform-refactor docs/plan
```

**Done when:**

- a reviewer can see what passed, what failed, and what remains;
- no remote CI claim exists without a run URL.

### Task 14: Remote CI and Release-Package Proof

**Goal:** prove the contract refactor on GitHub Actions and release-package dry run after the branch
is reviewable.

**Prerequisites:**

- explicit user approval for commit/push/PR/workflow dispatch;
- clean PR scope;
- local validation ladder passed or known failures are documented.

**Approach:**

1. Commit only intended files.
2. Push branch.
3. Open draft PR.
4. Dispatch `ci.yml`.
5. Dispatch `release-packages.yml` with `dry_run=true` and `update_homebrew_tap=false`.
6. Download or list artifact names and compare them to contract `ci_artifacts`.
7. Record run URLs and statuses in `reports/contract-platform-refactor/remote-ci.md`.

**Verification:**

```sh
gh status
gh workflow run ci.yml --ref <branch>
gh run list --workflow ci.yml --limit 5
gh workflow run release-packages.yml \
  --ref <branch> \
  -f version=v0.4.0 \
  -f dry_run=true \
  -f update_homebrew_tap=false
gh run list --workflow release-packages.yml --limit 5
```

**Done when:**

- remote CI proof exists for the branch;
- release-package dry-run artifacts exist;
- failures are either fixed or documented as blockers with exact run links.

### Task 15: Cleanup, Deprecation, and Guardrails

**Goal:** remove duplicated release truth only after contract-backed replacements are proven.

**Modify only after migration is proven:**

- obsolete shell duplicate checks;
- stale docs claims;
- old fixtures that no longer represent supported behavior;
- redundant script tests that now duplicate contract tests.

**Approach:**

1. Use `rg` to find old duplicated paths and claim strings.
2. Delete only dead code with test proof.
3. Keep compatibility wrappers for public scripts.
4. Add tests that prevent reintroducing undocumented release claims.

**Verification:**

```sh
CLAIM_RE='production_claim|experimental|supported_targets|artifact-hashes'
CLAIM_RE="${CLAIM_RE}|surface-release-v1|memory-100-prod-stable"
CLAIM_RE="${CLAIM_RE}|actor-runtime-foundation"
rg "$CLAIM_RE" scripts tools docs .github
go test ./tools/... -count=1
bash scripts/ci/test-all.sh \
  --quick \
  --keep-going \
  --report-dir reports/contract-platform/cleanup-quick
```

**Done when:**

- duplicate truth has been removed or justified;
- compatibility entrypoints still work;
- docs and validators are aligned.

## 9. Dependency Map

Recommended order:

1. Task 0: baseline and scope inventory.
2. Task 1: contract spec.
3. Task 2: report directory policy.
4. Task 3: artifact hash policy.
5. Task 4: runner.
6. Task 5: registry/naming rules.
7. Task 6: Surface vertical slice.
8. Task 7: Surface subgates.
9. Task 8: post-v0.4 gates.
10. Task 9: docs and manifest.
11. Task 10: `tools/scriptstest`.
12. Task 11: CI workflows.
13. Task 12: verification ladder.
14. Task 13: evidence handoff.
15. Task 14: remote CI and release-package proof.
16. Task 15: cleanup and deprecation.

Parallelizable read-only audits:

- contract inventory across `scripts/release/**/contracts`;
- docs claim inventory across `docs/` and `README.md`;
- CI artifact path inventory across `.github/workflows/`;
- validator/report schema inventory across `tools/cmd/validate-*` and `tools/validators/**`;
- script test brittleness inventory across `tools/scriptstest`.

Do not parallelize edits to the same files.

## 10. Acceptance Criteria

The full refactor is complete only when all of these pass:

- contract schema and runner tests pass;
- Surface release vertical slice passes through `run-gate`;
- post-v0.4 selected gates have contracts and dry-run coverage;
- artifact hashes are generated and validated for contract-required reports;
- docs claims are checked against contract-backed manifest/report evidence;
- `tools/scriptstest` proves CI and release script wiring;
- actionlint passes;
- local verification ladder passes;
- remote `ci.yml` evidence is recorded;
- remote `release-packages.yml` dry-run evidence is recorded;
- PR scope excludes unrelated dirty work;
- review handoff lists exact commands, artifacts, run URLs, and known risks.

Minimum `FINAL` evidence:

- local command transcript summary with exit codes;
- report directories;
- artifact hash manifests;
- generated manifest diff status;
- GitHub Actions run URLs;
- release-package dry-run artifact names;
- final `git status --short --branch`.

## 11. Known Risks

- The current checkout is dirty and `main` is behind `origin/main`; clean PR preparation may require
  an isolated worktree.
- Surface release gates are host-dependent; Wayland, browser, accessibility, and clipboard evidence
  may fail because of host preconditions rather than code.
- Broad `go test ./tools/...` can expose unrelated failures; failures must be triaged rather than
  hidden.
- Generated docs and manifests can cause noisy diffs if regeneration is not deterministic.
- CI workflow changes can look correct locally but fail remotely because of runner permissions or
  missing packages.
- Release-package dry-run must not be confused with external publishing.

## 12. Stop Conditions

Stop and record a blocker instead of improvising if:

- a gate needs a host capability that is unavailable;
- the same fix fails twice;
- unrelated dirty files prevent a clean PR boundary;
- remote workflow dispatch requires authorization not yet granted;
- a validator and docs contract disagree in a way that would weaken claims;
- implementation requires changing Tetra language semantics.

## 13. Recommended Execution Mode

For a `/goal` run, use packetized execution:

- read-only agents audit inventory and risks;
- editing agents implement one PR slice at a time;
- coordinator integrates, runs verification, and writes evidence.

Suggested packet sequence:

1. `P01-contract-inventory-readonly`
2. `P02-docs-claims-inventory-readonly`
3. `P03-ci-artifacts-inventory-readonly`
4. `P04-contract-core-implementation`
5. `P05-run-gate-hardening`
6. `P06-surface-release-contract-slice`
7. `P07-post-v04-contract-slice`
8. `P08-docs-manifest-alignment`
9. `P09-ci-workflow-alignment`
10. `P10-pr-hardening-and-remote-proof`

Each packet must return:

- files inspected;
- files changed;
- commands run;
- exit codes;
- evidence paths;
- remaining risks.

## 14. First Implementation Cut

The first implementation cut should be deliberately conservative:

1. Freeze `tools/internal/gatecontract`, `tools/internal/reportdir`, and `tools/internal/artifacts`.
2. Prove `tools/cmd/run-gate` with unit tests and dry-run JSON.
3. Wire only `scripts/release/surface/release-gate.sh` through `surface-release-v1.json`.
4. Add focused `tools/scriptstest` coverage for the new contract wiring.
5. Run the core local ladder.
6. Only then decide whether to include broader Surface validators in the same PR or split them.

This avoids turning a necessary architecture repair into an unreviewable mega-diff.

## 15. Final Reporting Template

When execution completes, report in this format:

```text
Status: DONE | PARTIAL | BLOCKED
Completed:
- ...

Scope covered:
- ...

Validation:
- command -> exit code -> evidence path

Evidence:
- reports/...
- GitHub Actions run URL, if any

Not verified / risks:
- ...
```

`DONE` is allowed only when all acceptance criteria in this plan pass. Otherwise the correct final
status is `PARTIAL` or `BLOCKED`.
