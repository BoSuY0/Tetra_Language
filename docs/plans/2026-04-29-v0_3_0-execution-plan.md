# Tetra v0.3.0 Execution Plan

**Goal:** turn the current `v0.3.0` planning contract into a small,
evidence-backed next-minor release path without weakening the `v0.2.0` current
support truth.

**Context:** `README.md`, `docs/spec/current_supported_surface.md`, and
`docs/spec/v0_3_scope.md` currently agree that `v0.2.0` is the public profile,
`v0.3.0` is next-cycle planning, and `v1.0.0` remains a future label. The repo
already has substantial tests around the candidate slices, but it does not yet
have a dedicated `v0.3.0` checklist, release gate, or final support claim.

**Execution:** use `subagent-driven-development` for implementation tasks that
touch compiler/CLI behavior, or `executing-plans` for checkpointed batches. Keep
each task evidence-producing; do not promote a feature in docs or
`compiler/features.go` before focused tests and broad gates pass in the same
branch state.

## Observed Baseline

- Branch state at planning time: `main...origin/main [ahead 22]` with existing
  dirty changes from the `v0.2.0` cleanup/docs/CI pass.
- Current public profile: `v0.2.0` in `README.md`,
  `docs/spec/current_supported_surface.md`, and `docs/generated/manifest.json`.
- Next-cycle contract: `docs/spec/v0_3_scope.md` lists candidate slices but does
  not claim them as stable support.
- Existing broad commands documented by the repo:
  `bash scripts/ci/test.sh`, `bash scripts/ci/test-all.sh --quick`,
  `bash scripts/ci/test-all.sh --stabilization --keep-going`,
  `bash scripts/dev/fuzz-nightly.sh --short --out-dir <report-dir>/fuzz-short`,
  `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`, and
  `git diff --check`.

## Guardrails

- Keep `v0.2.0` as current truth until an intentional version bump and fresh
  `v0.3.0` release evidence exist.
- Do not treat any `release_v1_0_*` script as proof of `v0.3.0` readiness unless
  a `v0.3.0` checklist/gate explicitly calls it and records why.
- Promote only narrow, tested slices. Avoid claiming lifetime SSA, dynamic
  protocol dispatch, captured closures, production EcoNet/TetraHub, or WASM
  runtime parity.
- Prefer focused tests before broad gates. Broad gates prove integration; they do
  not define the feature boundary by themselves.

## Task 1: Close The Current Dirty Cleanup Pass

**Goal:** make the already-completed `v0.2.0` cleanup/CI/docs changes ready for a
clean baseline before starting `v0.3.0` implementation work.

**Files:** inspect the current `git status --short --branch` output and all
dirty files; do not revert unrelated user changes.

**Approach:** review the existing dirty diff as one stabilization batch, keep the
new docs/scripts/tests together, and decide whether to commit or otherwise
checkpoint them before feature work starts. If the branch remains dirty, record
the exact reason in the next handoff.

**Verification:**

```sh
git status --short --branch
bash scripts/ci/test.sh
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
git diff --check
```

**Done when:** the team has an explicit baseline decision, and no later `v0.3.0`
task has to infer whether a dirty file belongs to previous cleanup or new work.

**Notes:** this task is first because several untracked files are part of the
current cleanup pass, including `scripts/dev/format.sh`, docs under `docs/user`, and
same-package CLI splits.

## Task 2: Add Dedicated v0.3.0 Release Scaffolding

**Goal:** create a release boundary for `v0.3.0` that cannot be confused with
`v0.2.0` or future `v1.0.0` gates.

**Files:** add `docs/checklists/v0_3_0_release_gate.md`; add
`scripts/release/v0_3_0/gate.sh`; update `tools/scriptstest` with focused gate
contract tests; inspect `scripts/release/v0_2_0/gate.sh`,
`scripts/release/v0_1_3/gate.sh`, and `tools/scriptstest/release_v1_test.go` for
reuse boundaries.

**Approach:** start with a non-claiming checklist and gate skeleton. The gate
should preflight `./tetra version` for `v0.3.0`, emit a distinct artifact/report
contract such as `tetra.release.v0_3_0.gate-report.v1`, and run the `v0.3.0`
verification envelope from `docs/spec/v0_3_scope.md`. Keep the script blocked or
version-gated until the project intentionally bumps version metadata.

**Verification:**

```sh
go test ./tools/scriptstest -count=1
bash scripts/ci/test.sh
git diff --check
```

After version metadata is intentionally bumped:

```sh
bash scripts/release/v0_3_0/gate.sh --report-dir reports/release-v0.3.0-gate
```

**Done when:** `v0.3.0` has its own checklist, gate script, script tests, and a
clear preflight that prevents false release claims.

**Notes:** do not update README/current truth to call `v0.3.0` current in this
task.

## Task 3: Make Test-All Evidence Version-Aware

**Goal:** prevent `v0.3.0` evidence from being mislabeled as `v0.2.0`.

**Files:** inspect/modify `scripts/ci/test-all.sh`; update
`tools/scriptstest/test_all_test.go`; inspect `.github/workflows/ci.yml` if CI
labels consume the report contract.

**Approach:** parameterize release version/artifact naming through the existing
environment or flags instead of hardcoding `v0_2_0` everywhere. Keep the default
current-profile behavior unchanged until the release line is intentionally moved.

**Verification:**

```sh
go test ./tools/scriptstest -run 'TestAll|Release|Workflow' -count=1
bash scripts/ci/test-all.sh --quick --report-dir reports/v0.3-test-all-quick
git diff --check
```

**Done when:** test summary artifacts can be generated for `v0.3.0` without
lying about the release line, and existing `v0.2.0` tests still pass.

**Notes:** this is a release-engineering prerequisite for trustworthy promotion
evidence.

## Task 4: Promote Enum Payload Match First If Fresh Evidence Stays Green

**Goal:** decide whether the narrow same-module positional enum payload slice can
move from experimental next-cycle to `v0.3.0` support.

**Files:** inspect/modify as needed in `compiler/internal/semantics/exprs.go`,
`compiler/internal/semantics/checker.go`, `compiler/internal/lower`,
`compiler/compiler_test.go`, `compiler/typed_errors_test.go`,
`compiler/internal/lower/enum_payload_test.go`, `examples/enum_payload_smoke.tetra`,
`compiler/features.go`, `docs/spec/current_supported_surface.md`, and
`docs/spec/v0_3_scope.md`.

**Approach:** first run the focused evidence without changing code. If it passes,
fill only missing diagnostics/docs gaps. Promotion wording must stay limited to
same-module enum constructors with positional payload arguments/bindings and
tested exhaustive `match`/`catch` behavior. Do not claim nested destructuring,
richer ADTs, or runtime algebra.

**Verification:**

```sh
go test ./compiler/... -run 'Enum|Match|TypedError' -count=1
./tetra check examples/enum_payload_smoke.tetra
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
```

**Done when:** the feature registry, specs, examples, and tests all describe the
same narrow support boundary, or the slice remains explicitly experimental with
documented blockers.

**Notes:** this is the lowest-risk language promotion candidate found during
planning.

## Task 5: Keep Callable Level 1 Narrow Or Explicitly Experimental

**Goal:** avoid over-promising function values while preserving the current
callable MVP.

**Files:** inspect/modify as needed in `compiler/function_typed_callable_test.go`,
`compiler/internal/semantics/exprs.go`, `compiler/internal/lower/lower.go`,
`compiler/features.go`, `docs/spec/current_supported_surface.md`, and
`docs/user/language_tour.md` or a new preview doc.

**Approach:** document and test the accepted Level 0/limited Level 1 matrix:
symbol-backed, non-capturing, non-generic, non-throwing functions/closures and
known callback targets. Before any promotion, decide whether the
`lowerFunctionTypedParamCall` multi-target fallback behavior is intended and add
a focused regression for the decision. Keep captured closures and general
first-class function values out of scope.

**Verification:**

```sh
go test ./compiler/... -run 'Closure|Callable|FunctionType' -count=1
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
```

**Done when:** users can tell supported callable MVP behavior from experimental
Level 1 and planned Level 2 behavior, and tests prove the boundary.

**Notes:** if the fallback semantics are unclear, leave `language.callable-level1`
experimental.

## Task 6: Promote Only Static Protocol-Bound Generic Checks

**Goal:** harden generic/protocol validation without claiming runtime protocol
dispatch.

**Files:** inspect/modify as needed in `compiler/generics_test.go`,
`compiler/protocol_conformance_test.go`, `compiler/internal/semantics/generics.go`,
`compiler/internal/semantics/checker.go`, `compiler/features.go`, and
`docs/spec/current_supported_surface.md`.

**Approach:** focus on static monomorphized generic functions, protocol bounds,
signature-shape validation, cross-module visibility, and unsupported requirement
call diagnostics. Keep witness tables, trait objects, runtime protocol values,
and dynamic dispatch explicitly outside the support claim.

**Verification:**

```sh
go test ./compiler/... -run 'Generic|Protocol|Conformance|Extension' -count=1
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
```

**Done when:** the promoted wording says static conformance/checking only, and
negative tests still reject runtime/dynamic-dispatch interpretations.

**Notes:** do not promote generic structs as part of this task unless separate
implementation, tests, and docs prove that boundary.

## Task 7: Harden Ownership/Resource Safety Within The Conservative MVP

**Goal:** reduce one concrete false-negative class without implying full lifetime
SSA.

**Files:** inspect/modify as needed in `compiler/ownership_test.go`,
`compiler/resource_finalization_test.go`, `compiler/internal/semantics/exprs.go`,
`compiler/internal/semantics/region.go`, `compiler/features.go`, and
`docs/spec/current_supported_surface.md`.

**Approach:** pick one failing or missing scenario at a time, write a regression,
then adjust the conservative checker. Do not introduce a broad SSA rewrite under
this plan. Keep diagnostics conservative for ambiguous provenance and merge
cases.

**Verification:**

```sh
go test ./compiler/... -run 'Ownership|Borrow|Consume|Inout|Lifetime|Resource|Island|Task' -count=1
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
```

**Done when:** one named false-negative class is fixed or explicitly deferred,
and `language.lifetime-ssa` remains planned unless a separate design is approved.

**Notes:** this is medium/high risk because resource logic crosses tasks, actors,
islands, and control-flow merges.

## Task 8: Improve Capsule/Eco Artifact Usability Without Network Claims

**Goal:** make local path dependency artifact generation, validation, and repair
guidance easier to trust for `v0.3.0`.

**Files:** inspect/modify as needed in `cli/cmd/tetra/main.go`,
`cli/cmd/tetra/eco.go`, `cli/cmd/tetra/eco_manifest.go`,
`cli/cmd/tetra/main_test.go`, `tools/cmd/validate-eco-lock/main.go`, and
`docs/user/eco_package_guide.md`.

**Approach:** keep changes local-only: clearer stale artifact diagnostics,
repair hints, dry-run behavior, target-aware artifact checks, and mechanical
same-package splits where they reduce risk. Do not claim distributed EcoNet,
production TetraHub, global trust scoring, or proof-carrying capsules.

**Verification:**

```sh
go test ./cli/... ./tools/... -run 'Eco|Project|Workspace|Artifact|Capsule|Lock' -count=1
bash scripts/ci/test-all.sh --full --keep-going --report-dir reports/v0.3-eco-full
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
```

**Done when:** common local artifact repair paths are tested and documented, and
the support claim remains local-only.

**Notes:** mutating project-sync commands need explicit dry-run tests before UX
wording changes.

## Task 9: Clarify WASI/Web Smoke Reporting Without Runtime Support Claims

**Goal:** distinguish build-only success, missing host runner/browser
dependencies, and real execution failures in user-facing reports.

**Files:** inspect/modify as needed in `cli/cmd/tetra/main.go`,
`tools/cmd/validate-web-ui-smoke/main.go`, `scripts/release/v1_0/wasi-smoke.sh`,
`scripts/release/v1_0/web-smoke.sh`, `.github/workflows/ci.yml`,
`docs/user/wasm_ui_guide.md`, and the new `v0.3.0` checklist/gate from Task 2.

**Approach:** treat this as reporting clarity, not WASM runtime parity. If
reusing `release_v1_0_*` smoke scripts, the `v0.3.0` checklist must clearly mark
which evidence is build-only, host-blocked, or executed. Add structured status or
reason fields only with validator/test coverage.

**Verification:**

```sh
./tetra smoke --target wasm32-wasi --run=false --report reports/v0.3-wasi-build.json
./tetra smoke --target wasm32-web --run=false --report reports/v0.3-web-build.json
go run ./tools/cmd/validate-web-ui-smoke --report reports/v0.3-web-build.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
```

If host dependencies are intentionally tested:

```sh
bash scripts/release/v1_0/wasi-smoke.sh --report reports/v0.3-wasi-runtime.json
bash scripts/release/v1_0/web-smoke.sh --report reports/v0.3-web-runtime.json
```

**Done when:** reports and docs let maintainers separate host blockers from real
compiler/runtime regressions without claiming stable WASM runtime execution.

**Notes:** current manifest target truth should also be reconciled or documented:
`docs/generated/manifest.json` lists native targets while WASM build-only support
appears as feature metadata.

## Task 10: Add A User-Facing v0.3 Preview Boundary

**Goal:** make next-cycle behavior understandable without making it current
support prematurely.

**Files:** add `docs/user/v0_3_preview.md` or extend `docs/user/status.md`;
inspect/update `docs/user/language_tour.md`, `docs/user/examples_index.md`,
`docs/user/troubleshooting.md`, `docs/spec/current_supported_surface.md`, and
`docs/generated/manifest.json` only when required by the selected approach.

**Approach:** map each candidate slice to user-visible boundaries: enum payload
match, callable levels, static protocol-bound generics, conservative
ownership/resource hardening, local Capsule/Eco artifacts, and WASI/Web smoke
clarity. Reclassify `examples/generic_struct_smoke.tetra` in docs as
experimental/excluded unless separate evidence promotes generic structs.

**Verification:**

```sh
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
git diff --check
```

If examples index changes:

```sh
./tetra smoke --list --format=json > reports/v0.3-smoke-list-linux-x64.json
go run ./tools/cmd/validate-example-index --smoke-list reports/v0.3-smoke-list-linux-x64.json --index docs/user/examples_index.md
go run ./tools/cmd/validate-smoke-list --report reports/v0.3-smoke-list-linux-x64.json --examples-root examples
```

**Done when:** a new user can see what `v0.3.0` is preparing, what remains
experimental, and which examples are safe to treat as supported.

**Notes:** de-emphasize current-user references to `release_v1_0_*` scripts unless
they are clearly labeled as future-release maintainer tooling.

## Task 11: Update CI For v0.3 Evidence Collection

**Goal:** make daily/PR automation collect the same kind of evidence the
`v0.3.0` plan requires.

**Files:** inspect/modify `.github/workflows/ci.yml`, `scripts/ci/test-all.sh`,
`scripts/dev/fuzz-nightly.sh`, and `tools/scriptstest/ci_workflow_test.go`.

**Approach:** keep fast PR jobs affordable, but add or clearly schedule the
`--stabilization` envelope and `v0.3.0` gate once the gate is version-ready.
Decide whether scheduled fuzz should remain short smoke or run a longer bounded
nightly duration; encode that decision in workflow tests/docs.

**Verification:**

```sh
go test ./tools/scriptstest -run 'CI|Workflow|Fuzz|TestAll|Release' -count=1
bash scripts/dev/fuzz-nightly.sh --short --out-dir reports/v0.3-fuzz-short
git diff --check
```

**Done when:** CI tests fail if the workflow drops required `v0.3.0`
stabilization/gate evidence, and scheduled fuzz behavior is intentional.

**Notes:** avoid requiring host-only browser/WASI dependencies in normal PR jobs
unless the workflow marks blocked/missing dependency outcomes clearly.

## Task 12: Final v0.3.0 Promotion And Handoff

**Goal:** make the final current-truth switch only after all selected slices have
fresh evidence.

**Files:** inspect/modify version metadata, `README.md`,
`docs/spec/current_supported_surface.md`, `docs/spec/v0_3_scope.md`,
`docs/checklists/v0_3_0_release_gate.md`, `docs/release-notes`, `docs/release`,
`compiler/features.go`, and `docs/generated/manifest.json`.

**Approach:** update release truth in one controlled batch after gates pass. The
handoff should record selected slices, rejected/deferred slices, exact report
paths, command summaries, and residual risks.

**Verification:**

```sh
go test ./compiler/... ./cli/... ./tools/... -count=1
bash scripts/ci/test-all.sh --stabilization --keep-going --report-dir reports/v0.3-stabilization
bash scripts/dev/fuzz-nightly.sh --short --out-dir reports/v0.3-stabilization/fuzz-short
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
git diff --check
```

After the version bump and gate are ready:

```sh
bash scripts/release/v0_3_0/gate.sh --report-dir reports/release-v0.3.0-gate
```

**Done when:** `v0.3.0` is either truthfully promoted with a release handoff and
fresh reports, or explicitly left as next-cycle planning with blockers listed.

**Notes:** do not mix the final truth switch with broad feature implementation;
that makes review and rollback unnecessarily risky.

## Recommended Execution Order

1. Task 1: close/checkpoint current cleanup pass.
2. Task 2 and Task 3: release scaffolding and version-aware evidence.
3. Task 4: enum payload match promotion decision.
4. Task 6: static protocol-bound generic checks.
5. Task 8: Capsule/Eco local artifact UX.
6. Task 10: user-facing preview boundary.
7. Task 11: CI evidence alignment.
8. Task 5, Task 7, and Task 9: only after the lower-risk slices are stable, or
   keep them experimental/reporting-only.
9. Task 12: final promotion/handoff.

## Full Verification Envelope

Run this before any public `v0.3.0` support claim:

```sh
go test ./compiler/... ./cli/... ./tools/... -count=1
bash scripts/ci/test-all.sh --stabilization --keep-going --report-dir reports/v0.3-stabilization
bash scripts/dev/fuzz-nightly.sh --short --out-dir reports/v0.3-stabilization/fuzz-short
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
git diff --check
```

Focused candidate commands:

```sh
go test ./compiler/... -run 'Enum|Match|TypedError' -count=1
go test ./compiler/... -run 'Closure|Callable|FunctionType' -count=1
go test ./compiler/... -run 'Generic|Protocol|Conformance|Extension' -count=1
go test ./compiler/... -run 'Ownership|Borrow|Consume|Inout|Lifetime|Resource|Island|Task' -count=1
go test ./cli/... ./tools/... -run 'Eco|Project|Workspace|Artifact|Capsule|Lock' -count=1
```
