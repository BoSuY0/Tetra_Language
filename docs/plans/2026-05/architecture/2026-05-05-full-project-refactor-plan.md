# Full Project Refactor Plan

**Goal:** Continue refactoring the Tetra Language repository in small, verified slices without
changing public behavior unintentionally.

**Context:** The repository is a Go workspace with modules at `.`, `./compiler`,
`./cli`, and `./tools`. The canonical fast gate is `bash scripts/ci/test.sh`.
Use `CACHE="$PWD/.cache/go-build-refactor"` before the verification commands
below.

**Current baseline:** `GOCACHE="$CACHE" bash scripts/ci/test.sh` exits `0`,
prints `OK`, and emits `Artifact: tetra.release.v0_3_0.go-test-suite.v1`.

## Refactor Rules

- Keep each slice behavior-preserving unless the task explicitly fixes a failing test or documented
  contract.
- Prefer same-package splits before API changes.
- Run nearby package tests before the full gate.
- Do not rewrite release/version truth while unrelated refactor work is in flight.
- Do not rely on generated docs or manifests as proof unless the verifier covers the behavior being
  changed.

## Completed Slices

### Function-Type Semantic Helpers

**Goal:** Reduce `compiler/internal/semantics/checker.go` size and isolate function-type binding
logic.

**Files:**

- `compiler/internal/semantics/function_types.go`
- `compiler/internal/semantics/closure_captures.go`
- `compiler/internal/semantics/checker.go`
- `compiler/internal/semantics/exprs.go`
- `compiler/internal/semantics/inference.go`
- `compiler/internal/lower/lower.go`

**Notes:** The split exposed fnptr metadata gaps for struct fields and enum payloads. Those were
fixed in the same slice because they blocked the canonical gate.

**Verification:**

- `GOCACHE="$CACHE" go test ./compiler/internal/semantics -count=1`
- `GOCACHE="$CACHE" go test ./compiler/internal/lower -count=1`
- `GOCACHE="$CACHE" bash scripts/ci/test.sh`

### Lowering Callable Helpers

**Goal:** Move callable lowering support out of `compiler/internal/lower/lower.go` into a focused
same-package file.

**Files:**

- `compiler/internal/lower/lower.go`
- `compiler/internal/lower/callables.go`
- Keep tests in `compiler/internal/lower/callable_test.go`

**Notes:** The extraction also preserved mutable function-typed local dispatch by routing mutable
locals through fnptr target branching instead of static symbol calls. Reassignment from direct named
symbols, supported function-typed returns, and supported closure literals remains covered by smoke
tests; generic closure literal reassignment remains rejected.

**Verification:**

- `GOCACHE="$CACHE" go test ./compiler/internal/lower -run TestLowerCallable -count=1 -v`
- Function-typed propagation check:
  ```bash
  GOCACHE="$CACHE" \
    go test ./compiler \
      -run 'FunctionTyped|ReturnedFunctionTypedValuesPropagateEffects' \
      -count=1
  ```
- `GOCACHE="$CACHE" go test ./compiler/internal/lower -count=1`
- `GOCACHE="$CACHE" bash scripts/ci/test.sh`

### CLI LSP Command Surface Split

**Goal:** Reduce `cli/cmd/tetra/main.go` by extracting the cohesive LSP command group.

**Files:**

- `cli/cmd/tetra/main.go`
- `cli/cmd/tetra/lsp.go`
- Existing nearby files: `cli/cmd/tetra/lsp_protocol.go`, `cli/cmd/tetra/lsp_wire.go`
- Tests: `cli/cmd/tetra/main_test.go`, `cli/cmd/tetra/eco_wave10_test.go`,
  `cli/cmd/tetra/lsp_wire_test.go`

**Notes:** Moved the `runLSP` command, JSON-RPC stdio loop, LSP request helpers,
symbol/hover/definition/reference/rename/completion/code-action/formatting helpers, and `maxInt` out
of `main.go`. `main.go` dropped from 4489 lines to 3518 lines.

**Verification:**

- `GOCACHE="$CACHE" go test ./cli/cmd/tetra -run 'LSP|ReadLSP' -count=1 -v`
- `GOCACHE="$CACHE" go test ./cli/cmd/tetra -count=1`
- `GOCACHE="$CACHE" bash scripts/ci/test.sh`

### CLI Metadata Command Surface Split

**Goal:** Continue reducing `cli/cmd/tetra/main.go` by extracting the cohesive metadata command
group.

**Files:**

- `cli/cmd/tetra/main.go`
- `cli/cmd/tetra/metadata.go`

**Notes:** Moved `targets`, `features`, and `formats` command handlers, their JSON report types, and
target metadata helpers into `metadata.go`. `main.go` dropped from 3518 lines to 3267 lines.

**Verification:**

- `GOCACHE="$CACHE" go test ./cli/cmd/tetra -run 'Targets|Features|Formats|Doctor' -count=1 -v`
- `GOCACHE="$CACHE" go test ./cli/cmd/tetra -count=1`
- `GOCACHE="$CACHE" bash scripts/ci/test.sh`

### CLI New App Command Surface Split

**Goal:** Continue reducing `cli/cmd/tetra/main.go` by extracting the app scaffold command group.

**Files:**

- `cli/cmd/tetra/main.go`
- `cli/cmd/tetra/new_app.go`

**Notes:** Moved `tetra new app` argument handling, scaffold writing, `--lock` handling, and
scaffold name/slug helpers into `new_app.go`. Shared target/module helpers stayed in `main.go`
because build/run/smoke/workspace paths still use them. `main.go` dropped from 3267 lines to 3099
lines.

**Verification:**

- `GOCACHE="$CACHE" go test ./cli/cmd/tetra -run 'NewApp|NewCommand' -count=1 -v`
- `GOCACHE="$CACHE" go test ./cli/cmd/tetra -count=1`
- `GOCACHE="$CACHE" bash scripts/ci/test.sh`

### CLI Project Command Surface Split

**Goal:** Continue reducing `cli/cmd/tetra/main.go` by extracting the project command group.

**Files:**

- `cli/cmd/tetra/main.go`
- `cli/cmd/tetra/project_commands.go`
- Existing nearby file: `cli/cmd/tetra/project.go`

**Notes:** Moved `tetra project`, `project deps`, `project info`, `project sync`, dependency
edit/report helpers, project sync lock checks, and project info report building into
`project_commands.go`. Discovery/context helpers stayed in `project.go`. `main.go` dropped from 3099
lines to 2188 lines.

**Verification:**

- Project command check:
  ```bash
  GOCACHE="$CACHE" \
    go test ./cli/cmd/tetra \
      -run 'Project(Sync|Deps|Info)|DoctorCommandProject' \
      -count=1 -v
  ```
- `GOCACHE="$CACHE" go test ./cli/cmd/tetra -count=1`
- `GOCACHE="$CACHE" bash scripts/ci/test.sh`

### CLI Doctor And Interface Command Surface Split

**Goal:** Continue reducing `cli/cmd/tetra/main.go` by extracting doctor checks and the interface
command into cohesive same-package files.

**Files:**

- `cli/cmd/tetra/main.go`
- `cli/cmd/tetra/doctor.go`
- `cli/cmd/tetra/interface.go`

**Notes:** Moved `runDoctor`, doctor reports/checks, runtime metadata checks, and doctor helper
functions into `doctor.go`. Moved `runInterface` into `interface.go`. `main.go` dropped from 2188
lines to 1624 lines.

**Verification:**

- `GOCACHE="$CACHE" go test ./cli/cmd/tetra -run 'Doctor|Interface|Targets|Formats' -count=1 -v`
- `GOCACHE="$CACHE" go test ./cli/cmd/tetra -count=1`
- `GOCACHE="$CACHE" bash scripts/ci/test.sh`

### CLI Source Command Surface Split

**Goal:** Continue reducing `cli/cmd/tetra/main.go` by extracting source-facing commands into a
cohesive file.

**Files:**

- `cli/cmd/tetra/main.go`
- `cli/cmd/tetra/source_commands.go`

**Notes:** Moved `runDoc`, `runCheck`, `runFmt`, and `firstFormatterDiffPosition` into
`source_commands.go`. `main.go` dropped from 1624 lines to 1427 lines.

**Verification:**

- `GOCACHE="$CACHE" go test ./cli/cmd/tetra -run 'Doc|Check|Fmt|Interface' -count=1 -v`
- `GOCACHE="$CACHE" go test ./cli/cmd/tetra -count=1`
- `GOCACHE="$CACHE" bash scripts/ci/test.sh`

### CLI Smoke Command Surface Split

**Goal:** Continue reducing `cli/cmd/tetra/main.go` by extracting smoke report/list command handling
into a cohesive file next to the smoke registry.

**Files:**

- `cli/cmd/tetra/main.go`
- `cli/cmd/tetra/smoke.go`
- Existing nearby file: `cli/cmd/tetra/smoke_registry.go`

**Notes:** Moved smoke report/list structs, `runSmoke`, durable WASM artifact placement, smoke list
JSON/text rendering, example exclusion reporting, and target-group helpers into `smoke.go`.
`main.go` dropped from 1427 lines to 1104 lines.

**Verification:**

- `GOCACHE="$CACHE" go test ./cli/cmd/tetra -run 'Smoke|Targets|Doctor' -count=1 -v`
- `GOCACHE="$CACHE" go test ./cli/cmd/tetra -count=1`
- `GOCACHE="$CACHE" bash scripts/ci/test.sh`

### CLI Test Command And Source File Helpers Split

**Goal:** Continue reducing `cli/cmd/tetra/main.go` by extracting the `test` command and shared CLI
source-file helpers.

**Files:**

- `cli/cmd/tetra/main.go`
- `cli/cmd/tetra/test_command.go`
- `cli/cmd/tetra/source_files.go`
- Existing nearby file: `cli/cmd/tetra/source_commands.go`

**Notes:** Moved `runTest`, test runner duration handling, source file collection, module path
rewriting, and default source input helpers out of `main.go`. `collectTetraFiles` remains
package-visible for `check`/`fmt` flows in `source_commands.go`. `main.go` dropped from 1104 lines
to 782 lines.

**Verification:**

- Test/source command check:
  ```bash
  GOCACHE="$CACHE" \
    go test ./cli/cmd/tetra \
      -run 'TestCommand|CollectTetraFiles|CheckCommand|FmtCommand|DocCommand' \
      -count=1 -v
  ```
- `GOCACHE="$CACHE" go test ./cli/cmd/tetra -count=1`
- `GOCACHE="$CACHE" bash scripts/ci/test.sh`

### V0.4 Readiness Runtime Evidence Hardening

**Goal:** Preserve the canonical gate while allowing explicit cross-host runtime smoke evidence for
native targets that cannot run on the current host.

**Files:**

- `tools/cmd/validate-v0-4-readiness/main.go`
- `tools/cmd/validate-v0-4-readiness/main_test.go`

**Notes:** Added `--runtime-report target=path` inputs, validated runtime report
target/version/timestamp/git head/counts/cases, and required host-native reports to come from the
matching target host. This keeps `run_supported=false` targets blocked unless a concrete external
runtime smoke report satisfies the release decision evidence.

**Verification:**

- v0.4 readiness runtime evidence check:
  ```bash
  GOCACHE="$CACHE" \
    go test ./tools/cmd/validate-v0-4-readiness \
      -run 'RuntimeEvidence|RuntimeSmoke|WrongHost' \
      -count=1 -v
  ```
- `GOCACHE="$CACHE" go test ./tools/... -count=1`
- `GOCACHE="$CACHE" bash scripts/ci/test.sh`

### Nested Struct Callable Return Propagation

**Goal:** Preserve function-typed struct field metadata when a returned struct initializer embeds
another returned struct that contains callable fields.

**Files:**

- `compiler/internal/semantics/function_types.go`
- `compiler/internal/lower/callables.go`
- Tests: `compiler/function_typed_callable_test.go`

**Notes:** `Box(holder: makeHolder())` now carries `holder.cb` metadata through the nested returned
struct initializer. The callable target graph also keeps assignment-derived target edges, so
reassigned function-typed fields dispatch to the actual stored fnptr instead of a stale initializer
target.

**Verification:**

- Nested struct callable smoke:
  ```bash
  RUN_REGEX="$(
    printf '%s' \
      'TestBuildFunctionTypedStructFieldFromReassignedReturnedStructSmoke|' \
      'TestBuildFunctionTypedNestedStructFieldFromReturnedStructInitializerSmoke'
  )"
  GOCACHE="$CACHE" \
    go test ./compiler \
      -run "$RUN_REGEX" \
      -count=1 -v
  ```
- Function-typed propagation check:
  ```bash
  GOCACHE="$CACHE" \
    go test ./compiler \
      -run 'FunctionTyped|ReturnedFunctionTypedValuesPropagateEffects' \
      -count=1 -v
  ```
- `GOCACHE="$CACHE" go test ./compiler/internal/semantics -count=1`
- `GOCACHE="$CACHE" go test ./compiler/internal/lower -count=1`
- `GOCACHE="$CACHE" bash scripts/ci/test.sh`

### Direct Returned Enum Match Callable Propagation

**Goal:** Preserve function-typed enum payload metadata when matching directly on a function call
that returns an enum, without requiring an intermediate immutable local.

**Files:**

- `compiler/internal/semantics/function_types.go`
- `compiler/internal/semantics/checker.go`
- Tests: `compiler/function_typed_callable_test.go`

**Notes:** `match makeChoice(): case MaybeCallback.some(cb): cb(...)` now binds `cb` from the callee
return enum payload target metadata. This keeps direct returned enum matches aligned with the
already-supported `let choice = makeChoice(); match choice:` alias flow.

**Verification:**

- Direct returned enum match smoke:
  ```bash
  GOCACHE="$CACHE" \
    go test ./compiler \
      -run TestBuildFunctionTypedEnumPayloadFromDirectReturnedEnumMatchSmoke \
      -count=1 -v
  ```
- Function-typed propagation check:
  ```bash
  GOCACHE="$CACHE" \
    go test ./compiler \
      -run 'FunctionTyped|ReturnedFunctionTypedValuesPropagateEffects' \
      -count=1 -v
  ```
- `GOCACHE="$CACHE" go test ./compiler/internal/semantics -count=1`
- `GOCACHE="$CACHE" go test ./compiler/internal/lower -count=1`
- `GOCACHE="$CACHE" bash scripts/ci/test.sh`

### Mutable Enum Payload Callable Target Propagation

**Goal:** Preserve lowerer callable target metadata when a mutable enum local is assigned a
function-typed payload before matching on it.

**Files:**

- `compiler/internal/lower/callables.go`
- Tests: `compiler/function_typed_callable_test.go`

**Notes:** The callable target scan now tracks enum payload targets per local from constructor,
alias, and returned-enum flows, then binds those targets to enum pattern payload locals such as
`case MaybeCallback.some(cb)`. This lets dynamic enum payload callbacks dispatch through stored
fnptr slots after mutable reassignment.

**Verification:**

- Mutable enum payload callable target smoke:
  ```bash
  RUN_REGEX="$(
    printf '%s' \
      'TestBuildFunctionTypedMutableEnumPayloadReassignmentSmoke|' \
      'TestBuildFunctionTypedEnumPayloadAliasSmoke|' \
      'TestBuildFunctionTypedEnumPayloadFromReturnedEnumWholeEnumAliasSmoke'
  )"
  GOCACHE="$CACHE" \
    go test ./compiler \
      -run "$RUN_REGEX" \
      -count=1 -v
  ```
- Function-typed propagation check:
  ```bash
  GOCACHE="$CACHE" \
    go test ./compiler \
      -run 'FunctionTyped|ReturnedFunctionTypedValuesPropagateEffects' \
      -count=1 -v
  ```
- `GOCACHE="$CACHE" go test ./compiler/internal/lower -count=1`
- `GOCACHE="$CACHE" bash scripts/ci/test.sh`

### Script/Test Artifact Version Cleanup

**Goal:** Make local test artifact identity follow the selected release version
instead of carrying historical `v0.3.0` defaults into the current `v0.4.0`
profile.

**Files:**

- `scripts/ci/test.sh`
- `scripts/ci/test-all.sh`
- `tools/scriptstest/test_script_test.go`
- `tools/scriptstest/test_all_test.go`

**Notes:** `scripts/ci/test.sh` now derives
`tetra.release.<version>.go-test-suite.v1` from `./tetra version` when the
binary is present, with `TETRA_TEST_RELEASE_VERSION` and
`TETRA_TEST_RELEASE_ARTIFACT` overrides for test harnesses. `scripts/ci/test-all.sh`
now derives its default `release_version` from `./tetra version` while
preserving the existing `TETRA_TEST_ALL_RELEASE_VERSION` and
`TETRA_TEST_ALL_RELEASE_ARTIFACT` overrides. This fixes local summary identity
for the current `v0.4.0` profile without claiming the blocked release is
complete.

**Verification:**

- Artifact identity check:
  ```bash
  RUN_REGEX="$(
    printf '%s' \
      'TestCanonicalTestScriptArtifactFollowsTetraVersion|' \
      'TestTestAllQuickJSONIncludesStepExitCodes'
  )"
  GOCACHE="$CACHE" \
    go test ./tools/scriptstest \
      -run "$RUN_REGEX" \
      -count=1 -v
  ```
- Test-all summary check:
  ```bash
  RUN_REGEX="$(
    printf '%s' \
      'TestCanonicalTestScript|' \
      'TestTestAll(Quick|ReleaseArtifact|ChecksShortAliasVersion|' \
      'ValidatesSummaryArtifacts|TopLevelGoTestBypassesCache)'
  )"
  GOCACHE="$CACHE" \
    go test ./tools/scriptstest \
      -run "$RUN_REGEX" \
      -count=1
  ```
- Diff hygiene check:
  ```bash
  git diff --check -- \
    scripts/ci/test.sh \
    scripts/ci/test-all.sh \
    tools/scriptstest/test_script_test.go \
    tools/scriptstest/test_all_test.go
  ```

## Next Slices

### 1. CLI Command Surface Split

**Goal:** Continue reducing `cli/cmd/tetra/main.go` by extracting cohesive non-LSP command groups.

**Files:**

- Inspect/modify `cli/cmd/tetra/main.go`
- Existing nearby files: `cli/cmd/tetra/eco.go`, `cli/cmd/tetra/workspace.go`,
  `cli/cmd/tetra/smoke_registry.go`
- Tests: `cli/cmd/tetra/main_test.go`, `cli/cmd/tetra/eco_wave10_test.go`

**Approach:**

- First extract pure helpers and command-specific option structs.
- Keep command output strings stable.
- Avoid changing flag parsing behavior in the same slice as file moves.

**Verification:**

- `GOCACHE="$CACHE" go test ./cli/cmd/tetra -count=1`
- `GOCACHE="$CACHE" bash scripts/ci/test.sh`

### 2. Tool Validator Shared Library

**Goal:** Reduce repeated schema/report parsing patterns across `tools/cmd/validate-*`.

**Files:**

- Inspect `tools/cmd/validate-*`
- Add a shared package only if at least three validators share the same parsing/report contract.

**Approach:**

- Start with read-only duplication inventory.
- Extract only pure helpers with table-driven tests.

**Verification:**

- Targeted validator package tests for modified tools.
- `GOCACHE="$CACHE" go test ./tools/... -count=1`
- `GOCACHE="$CACHE" bash scripts/ci/test.sh`

## Completion Criteria

The full project refactor is not complete until:

- The canonical gate passes after every planned slice.
- Hotspot files are reduced or documented with intentional boundaries.
- Public behavior changes, if any, have tests and docs.
- Dirty worktree scope is reviewed so unrelated pre-existing changes are not accidentally claimed or
  reverted.
- A final completion audit maps the original request to concrete evidence.
