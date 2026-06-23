# Memory Core v2 Stabilization Findings

reviewed_commit: 8f7529505a13b5da72fbc0c34c5bb110541c020f

review_set:
- `docs/reviews/memory-core-v2/compiler-soundness-review.md`
- `docs/reviews/memory-core-v2/runtime-domain-review.md`
- `docs/reviews/memory-core-v2/optimizer-proof-review.md`
- `docs/reviews/memory-core-v2/integration-review.md`
- `docs/reviews/memory-core-v2/test-infrastructure-determinism-review.md`
- `docs/reviews/memory-core-v2/d003-linux-x32-root-cause-review.md`
- `docs/reviews/memory-core-v2/d004-readiness-path-root-cause-review.md`
- `docs/reviews/memory-core-v2/ci-context-causality-review.md`

summary:
- blocker: 0
- critical: 0
- high: 3
- medium: 3
- low: 2
- informational: 0

resolution_summary:
- resolved_high: 3
- resolved_medium: 3
- resolved_low: 1
- open_blocker: 0
- open_critical: 0
- open_high: 0
- open_medium: 0
- open_low: 1
- open_informational: 0

resolution_commits:
- A-001: `33f8609665df2997e228eb8218ba95fe1637e260`
- B-001: `b2a8df25d9bad30864d1c01aa669251474bcf732`
- C-001: `0b165d6fed08893e70932bdc50fb03d699ecc2e6`
- C-002: `0b165d6fed08893e70932bdc50fb03d699ecc2e6`
- D-002: `69f8827199583219aa1b8b97368ab692a1aa7d29`
- D-003: `7e9184aaa2c220590d67f3d369d9598b62861088`
- D-004: `f28953df325cd87cb8378b3c9b7952238b6d3e13`
- D-001: tracked as nonblocking test-infrastructure hardening; no Memory Core
  v2 code, gate, schema, or documentation fix required by the review.

## blocker

None.

## critical

None.

## high

### D-003: linux-x32 unsupported reason contract mismatch

source_review: `docs/reviews/memory-core-v2/test-infrastructure-determinism-review.md`
package: `cli/cmd/tetra`
status: resolved
severity: high
merge_blocking: false
reproduced_in_required_ci: true
ci_runs:
- 28020556066
- 28020557965
root_cause_review: `docs/reviews/memory-core-v2/d003-linux-x32-root-cause-review.md`
fix_commit: `7e9184aaa2c220590d67f3d369d9598b62861088`

finding:
The required `full-platform-ui-runtime` fan-in failed in both push and
pull_request workflows during `baseline-tests`. The first failing command is
`go test ./compiler/... ./cli/... ./tools/... -count=1` from
`scripts/release/full_platform/ui-runtime-gate.sh`. Both logs show
`cli/cmd/tetra` failures in `TestTargetMetadataCheck/wasi_runner_available` and
`TestTargetsCommandJSON` for the `linux-x32` unsupported-runner reason.

observed_failure:
- package: `tetra_language/cli/cmd/tetra`
- tests:
  `TestTargetMetadataCheck/wasi_runner_available`,
  `TestTargetsCommandJSON`
- observed value includes:
  `RunUnsupportedReason:"host linux/amd64 does not support Linux x32 ABI execution; no host fallback is allowed; probe command: tetra test --diagnostics=json --target x32 --format=json <runner-smoke.tetra>"`

root_cause:
Test expectation drift in two CLI metadata assertions. Production emits the
canonical host-qualified linux-x32 reason through
`buildOnlyNativeRunUnsupportedReason`, but
`TestTargetMetadataCheck/wasi_runner_available` and `TestTargetsCommandJSON`
still expected the older unqualified fragment
`host does not support Linux x32 ABI execution`.

fix:
Commit `7e9184aaa2c220590d67f3d369d9598b62861088` forces the affected metadata
tests through `stubLinuxX32HostSupport(false)` and compares
`run_unsupported_reason` with the canonical production constructor result.
Existing x32 run/test diagnostics now reuse the same helper instead of
duplicating the reason policy.

before_after:
- Before: CI linux/amd64 hosts without x32 execution support emitted
  `host linux/amd64 does not support Linux x32 ABI execution; no host fallback is allowed; probe command: tetra test --diagnostics=json --target x32 --format=json <runner-smoke.tetra>`
  and the stale metadata assertions failed.
- After: `TestTargetMetadataCheck`, `TestTargetsCommandJSON`, and the x32
  diagnostic tests assert the same canonical reason source and pass with the
  unsupported branch forced locally.

resolution_evidence:
- `gh run view 28020556066 --job 82935734296 --log-failed`
- `gh run view 28020557965 --job 82936970947 --log-failed`
- `rg -n -- 'TestTargetMetadataCheck|TestTargetsCommandJSON|linux-x32 unsupported host-probed metadata' /tmp/mcv2-push-fanin-failed.log /tmp/mcv2-pr-fanin-failed.log`
- RED: `go test -buildvcs=false ./cli/cmd/tetra -run '^(TestTargetMetadataCheck|TestTargetsCommandJSON)$' -count=1 -v`
  failed after forcing `linuxX32HostSupport(false)` with the old assertions.
- GREEN: `go test -buildvcs=false ./cli/cmd/tetra -run '^(TestTargetMetadataCheck|TestTargetsCommandJSON|TestRunCommandJSONDiagnosticsForLinuxX32HostUnsupported|TestTestCommandJSONDiagnosticsForBuildOnlyRuntimeUnsupported)$' -count=20 -v`
  passed.

### D-004: v0.4 readiness WASM UI guide path resolution failure

source_review: `docs/reviews/memory-core-v2/test-infrastructure-determinism-review.md`
package: `tools/cmd/validate-v0-4-readiness`
status: resolved
severity: high
merge_blocking: false
reproduced_in_required_ci: true
ci_runs:
- 28020556066
- 28020557965
root_cause_review: `docs/reviews/memory-core-v2/d004-readiness-path-root-cause-review.md`
fix_commit: `f28953df325cd87cb8378b3c9b7952238b6d3e13`

finding:
The required `full-platform-ui-runtime` fan-in failed in both push and
pull_request workflows during `baseline-tests`. The first failing command is
`go test ./compiler/... ./cli/... ./tools/... -count=1` from
`scripts/release/full_platform/ui-runtime-gate.sh`. Both logs show
`tools/cmd/validate-v0-4-readiness/TestValidateReadinessAcceptsNativeUIRuntimeEvidenceShape`
failing because the required guide path is reported as not readable.

observed_failure:
- package: `tetra_language/tools/cmd/validate-v0-4-readiness`
- test: `TestValidateReadinessAcceptsNativeUIRuntimeEvidenceShape`
- observed error:
  `expected native UI runtime-shaped evidence to pass readiness: decision ui.native-runtime evidence.docs path docs/user/surface/wasm_ui_guide.md is not readable`

root_cause:
Fixture omission. `nativeUIRuntimeEvidence()` requires
`docs/user/surface/wasm_ui_guide.md`, but
`TestValidateReadinessAcceptsNativeUIRuntimeEvidenceShape` changed cwd to an
isolated `t.TempDir()` fixture and did not create that required docs file.
The checkout file exists and is readable; the fixture file was absent.

fix:
Commit `f28953df325cd87cb8378b3c9b7952238b6d3e13` adds
`docs/user/surface/wasm_ui_guide.md` to the copied readiness fixture. The
validator requirement remains mandatory and `docs/user/surface/wasm_ui_guide.md`
content is unchanged.

before_after:
- Before: with `TMPDIR` outside the repository,
  `TestValidateReadinessAcceptsNativeUIRuntimeEvidenceShape` failed with
  `decision ui.native-runtime evidence.docs path docs/user/surface/wasm_ui_guide.md is not readable`.
- After: the fixture includes the required guide path and the same exact test
  passes under an outside-repository `TMPDIR`.

resolution_evidence:
- `gh run view 28020556066 --job 82935734296 --log-failed`
- `gh run view 28020557965 --job 82936970947 --log-failed`
- `rg -n -- 'TestValidateReadinessAcceptsNativeUIRuntimeEvidenceShape|wasm_ui_guide.md is not readable' /tmp/mcv2-push-fanin-failed.log /tmp/mcv2-pr-fanin-failed.log`
- RED: `go test -buildvcs=false ./tools/cmd/validate-v0-4-readiness -run '^TestValidateReadinessAcceptsNativeUIRuntimeEvidenceShape$' -count=100 -v`
  failed when `TMPDIR` was outside the repository fixture tree.
- GREEN: `go test -buildvcs=false ./tools/cmd/validate-v0-4-readiness -run '^TestValidateReadinessAcceptsNativeUIRuntimeEvidenceShape$' -count=100 -v`
  passed with an outside-repository `TMPDIR`.
- GREEN: `go test -buildvcs=false ./tools/cmd/validate-v0-4-readiness -count=50 -v`
  passed with an outside-repository `TMPDIR`.

### D-002: Windows full-platform UI runtime CI cannot checkout committed report evidence with long paths

source_review: GitHub Actions run `28019874499`
status: resolved
fix_commit: `69f8827199583219aa1b8b97368ab692a1aa7d29`

finding:
The pushed Draft PR branch failed the `full-platform-ui-runtime` Windows target
job before test execution. `actions/checkout@v4` could not create tracked
`reports/stabilization/tetra-ram-p7-compiler-rss-b452638a8af7-full-repo-smoke-samples2/...`
files because their paths exceed Git for Windows default filename limits.

reproduction:
- `gh run view 28019874499 --log-failed`
- Observe `windows-2025` checkout errors containing `Filename too long` before
  any build, test, or artifact upload step starts.

required_fix:
Enable `git config --global core.longpaths true` before `actions/checkout@v4`
for Windows full-platform UI runtime target-host jobs in both the standalone
workflow and the mirrored `ci.yml` workflow. Add workflow regression tests that
assert this ordering remains before checkout.

## medium

### A-001: Public multi-module lowering bypasses the canonical Memory Core v2 pipeline

source_review: `docs/reviews/memory-core-v2/compiler-soundness-review.md`
status: resolved
fix_commit: `33f8609665df2997e228eb8218ba95fe1637e260`

finding:
`compiler/compiler_facade.go` exposes `LowerModules(checked []*CheckedProgram)`
as a direct call to `lower.LowerModules(checked)`, while neighboring `Lower`
and `LowerModule` build a `memorypipeline.State`, lower via
`LowerPlannedProgram`, and apply lowering evidence. The internal
`lower.LowerModules` route calls `lowerCheckedFuncWithOptions(..., Options{},
nil, ...)`, so it has no canonical `memoryfacts.Graph`, no `allocplan.Plan`,
no per-allocation lowering evidence, and no validator handoff.

reproduction:
- `rg -n "func LowerModules|LowerModules\\(" compiler/compiler_facade.go compiler/internal/lower/lower_core.go compiler/tests/runtime/linker_test.go`
- Inspect `compiler/compiler_facade.go` around the public `LowerModules` API
  and `compiler/internal/lower/lower_core.go` around the internal
  `LowerModules` helper.

required_fix:
Route public `compiler.LowerModules` through the canonical Memory Core v2
pipeline or remove/deprecate the unplanned public lowering surface. Add
regression coverage proving `LowerModules` cannot lower memory-sensitive
programs without the canonical allocation plan/evidence path.

### B-001: Release evidence marks `wasm32-wasi reserve` unsupported while runtime ABI supports WASM reserve/commit

source_review: `docs/reviews/memory-core-v2/runtime-domain-review.md`
status: resolved
fix_commit: `b2a8df25d9bad30864d1c01aa669251474bcf732`

finding:
`MemoryBackendSupportMatrix("wasm32-wasi")` marks `reserve` and `commit`
supported through `wasm_memory_grow_combined_reserve_commit`, but the Memory
Core v2 gate and positive fixture mark `wasm32-wasi reserve` unsupported. The
release validator also expects flipping that row to `supported: true` to fail,
which contradicts the runtime ABI contract.

reproduction:
- Inspect `compiler/internal/runtimeabi/memory_backend.go` for
  `MemoryBackendSupportMatrix("wasm32-wasi")`.
- Inspect `compiler/internal/runtimeabi/memory_backend_test.go` for the WASM
  reserve/commit support assertions.
- Inspect `scripts/release/memory/memory-core-v2-gate.sh`,
  `tools/validators/memorycorev2/testdata/positive.json`, and
  `tools/validators/memorycorev2/report.go` for the release evidence and
  validator policy.

required_fix:
Align release evidence and validator policy with the runtime ABI contract. If
runtime ABI is correct, represent WASM reserve/commit as supported where
included and use an actually unsupported WASM operation such as `release`,
`decommit`, `trim`, or `footprint` for unsupported-target evidence.

### C-001: Manager proof-decision validation does not itself prove that nonempty proof IDs are canonical

source_review: `docs/reviews/memory-core-v2/optimizer-proof-review.md`
status: resolved
fix_commit: `0b165d6fed08893e70932bdc50fb03d699ecc2e6`

finding:
`validateMemoryDecisionEvidence` rejects `rewrite_applied` memory decisions
when `ProofIDs` is empty, but it does not resolve nonempty proof IDs through the
current `memoryfacts.Snapshot`. Current proof-consuming pass bodies call
`PassContext.requireMemoryProofs`, but the optimizer manager does not enforce
canonical proof resolution for every memory rewrite decision.

reproduction:
- Inspect `compiler/internal/opt/opt_core.go` around
  `validateMemoryDecisionEvidence`.
- Inspect `PassContext.requireMemoryProofs` and the current
  `loop-canonicalization` / `licm-pure-invariant` pass bodies.
- Add or run a contract pass that emits `DecisionCodeRewriteApplied` with
  `ProofIDs: []string{"proof:bogus"}` under `RunWithOptions` with
  `MemoryFacts` enabled; manager-level validation should reject it.

required_fix:
Add manager-level validation for memory rewrite decisions when canonical memory
facts are enabled: every proof ID on a memory rewrite decision must resolve
through the current `memoryfacts.Snapshot`, must be validated, must be non-unsafe
for proof-gated rewrites, and should populate canonical proof fact IDs or fail.
Add a regression test for a pass that emits a bogus nonempty proof ID.

## low

### C-002: Standalone `opt.Manager.Run` disables canonical memory proof resolution for proof-sensitive passes

source_review: `docs/reviews/memory-core-v2/optimizer-proof-review.md`
status: resolved
fix_commit: `0b165d6fed08893e70932bdc50fb03d699ecc2e6`

finding:
`Manager.Run` delegates to `RunWithOptions` with empty options, creating a
disabled memory context. `requireMemoryProofs` currently returns success when
memory context is disabled, so standalone callers can run proof-sensitive
passes using only IR proof IDs plus translation validation instead of canonical
snapshot resolution. Production build wiring supplies canonical `MemoryFacts`,
so this is a contract/API hardening risk rather than a known production-route
failure.

reproduction:
- Inspect `compiler/internal/opt/opt_core.go` around `Manager.Run`,
  memory context construction, and `requireMemoryProofs`.
- Inspect `compiler/compiler_facade.go` release optimization wiring, which
  supplies `Options{MemoryFacts: snapshot}`.

required_fix:
Either make `Run` reject proof-sensitive passes unless `Options.MemoryFacts` is
supplied, or clearly mark `Run` as noncanonical/test-only and add a negative
test that proof-sensitive passes skip or fail when canonical memory facts are
absent.

### D-001: Broad workspace/scriptstest runs showed transient environment coupling outside the Memory Core v2 gate path

source_review: `docs/reviews/memory-core-v2/integration-review.md`
status: open_nonblocking
tracking: test-infrastructure hardening outside the Memory Core v2 release gate

finding:
Broad `go test` runs intermittently failed in `tools/scriptstest` fake-repo or
workspace scenarios, including missing `$WORK/.../_pkg_.a` import archives and
tests that assumed a GitHub-shaped remote URL. Focused reruns of the failing
packages/tests passed, final broad reruns passed, and clean-clone Memory Core
v2 gates were deterministic.

reproduction:
- See the failed broad commands and passing focused/final reruns in
  `docs/reviews/memory-core-v2/integration-review.md`.

required_fix:
No Memory Core v2 code, gate, schema, or documentation change is required by
this review. Track as test-infrastructure hardening: make `tools/scriptstest`
fake repos and nested Go test isolation independent of broad package execution
order, shared temporary/cache state, and GitHub-shaped remotes.

## informational

None.
