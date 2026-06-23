# CI Context Causality Review

reviewed_commit:
- source_tree_reviewed_at: `30f1f7bd71c1972bc37ef937ee913ade3b3cbb80`
- reviewed_tree: `15731efcc5f841f7ea3ca00432f13f0ae4116814`
- ci_head_sha_reviewed: `df3256b904f08b036c5378bcc003c36c0bed5c3e`
- note: Current HEAD is the docs-only D-003/D-004 diagnostic commit on top of
  the CI head. Code/test behavior was inspected from current HEAD, while CI
  execution trees were verified from the recorded CI head and PR merge ref.

ci_run_ids:
- push:
  - run_id: `28020556066`
  - event: `push`
  - attempt: `1`
  - head_sha: `df3256b904f08b036c5378bcc003c36c0bed5c3e`
  - head_tree: `3d26e5f75a8c2f79e077060175e32fbac1449137`
  - log: `/tmp/d003-push-full.log`
  - failed_excerpt: `/tmp/mcv2-push-fanin-failed.log`
- pull_request:
  - run_id: `28020557965`
  - event: `pull_request`
  - attempt: `2`
  - head_sha: `df3256b904f08b036c5378bcc003c36c0bed5c3e`
  - head_tree_from_api: `3d26e5f75a8c2f79e077060175e32fbac1449137`
  - local_merge_ref: `refs/remotes/origin/pr-6-merge`
  - local_merge_commit: `a1cae0b93fa41cf0ab3319633dabd9c222e38287`
  - local_merge_tree: `c0afac5f9e417c65745dabbf6e3f485317ddfd93`
  - log: `/tmp/d004-pr-full.log`
  - failed_excerpt: `/tmp/mcv2-pr-fanin-failed.log`

execution_trees_identical: false

push_tree:
- verified: `git rev-parse df3256b904f08b036c5378bcc003c36c0bed5c3e^{tree}`
- value: `3d26e5f75a8c2f79e077060175e32fbac1449137`

pr_merge_tree:
- verified: `git rev-parse refs/remotes/origin/pr-6-merge^{tree}`
- value: `c0afac5f9e417c65745dabbf6e3f485317ddfd93`
- parents: `5201240a5b1f3623098146ab0b4bc834f8fc8a20`,
  `df3256b904f08b036c5378bcc003c36c0bed5c3e`
- diff_vs_push_head: only adds
  `dumps/tetra_language_dump_20260622_105404Z_part_001.md` through
  `dumps/tetra_language_dump_20260622_105404Z_part_010.md`

package_ordering:
- `scripts/release/full_platform/ui-runtime-gate.sh` runs `build-cli` first,
  then `baseline-tests` as:
  `go test ./compiler/... ./cli/... ./tools/... -count=1`.
- The broad package argument order is `compiler`, then `cli`, then `tools`.
  The CI logs show compiler package output before `tetra_language/cli/cmd/tetra`,
  then tools package output.
- `tools/scriptstest/workspace/TestWorkspaceModules` loops modules in fixed
  source order: `compiler`, `cli`, `tools`.
- Each `TestWorkspaceModules` subtest runs nested `go test ./... -count=1` from
  that module directory with module-specific `GOCACHE`, `GOTMPDIR`, and `TMPDIR`,
  plus inherited `os.Environ()`.
- `scripts/ci/test-all.sh` runs `go test all packages` as its first recorded
  step and runs `tooling summary aggregation` only near the end, after the full
  or stabilization step set.

downstream_witnesses:
- `TestWorkspaceModules` is a downstream duplicate surface. It reruns the same
  module package trees that already failed in `baseline-tests`, so it repeats:
  `cli/cmd/tetra` D-003 failures under `TestWorkspaceModules/cli`, and
  `tools/cmd/validate-v0-4-readiness` D-004 failures under
  `TestWorkspaceModules/tools`.
- `TestTestAllFullToolingSummaryFailsOnZeroByteRequiredArtifact` is not the
  first CI failure. In the push log it fails inside `TestWorkspaceModules/tools`
  because its synthetic `test-all.sh --full` summary stopped after
  `unsafe promotion blocker suite` failed and never reached the expected
  `tooling summary aggregation` failure.
- `TestTestAllStabilizationToolingSummaryRequiresFocusedArtifacts` is also not
  the first CI failure. In the PR log it fails inside `TestWorkspaceModules/tools`
  because its synthetic `test-all.sh --stabilization` summary stopped after
  `RAM contract fuzz oracle artifact gate` failed and never reached the expected
  `tooling summary aggregation` failure.
- Focused local run passed both summary tests:
  `go test -buildvcs=false ./tools/scriptstest/test_all -run 'TestTestAllFullToolingSummaryFailsOnZeroByteRequiredArtifact|TestTestAllStabilizationToolingSummaryRequiresFocusedArtifacts' -count=1`
  returned `ok tetra_language/tools/scriptstest/test_all 2.037s`.
- Therefore the summary tests are downstream witnesses in these CI logs, not the
  primary CI root.

D001_assessment:
- D-001 can be rejected as the causal explanation for D-003/D-004 in these two
  CI runs.
- Current logs do not show D-001's recorded signatures: no missing
  `$WORK/.../_pkg_.a` import archive failure and no GitHub-shaped remote URL
  assumption failure.
- The D-003 failure is a concrete linux-x32 unsupported-runner reason contract
  mismatch in `cli/cmd/tetra`.
- The D-004 failure reproduces locally as a concrete readiness fixture/path
  completeness failure.
- D-001 should remain tracked as open nonblocking `tools/scriptstest` hardening,
  but it is not supported as the root cause for the D-003/D-004 CI split.

reproduction:
- Tree verification:
  - `git rev-parse df3256b904f08b036c5378bcc003c36c0bed5c3e^{tree}` ->
    `3d26e5f75a8c2f79e077060175e32fbac1449137`
  - `git rev-parse refs/remotes/origin/pr-6-merge^{tree}` ->
    `c0afac5f9e417c65745dabbf6e3f485317ddfd93`
  - `git diff --name-status df3256b904f08b036c5378bcc003c36c0bed5c3e refs/remotes/origin/pr-6-merge`
    lists only the ten `dumps/tetra_language_dump_20260622_105404Z_part_*.md`
    additions.
- CI log checks:
  - `/tmp/d003-push-full.log` lines around `1092-1098`, `1245-1248`, and
    `1293-1305` show D-003, D-004, then the workspace duplicate.
  - `/tmp/d004-pr-full.log` lines around `1147-1153`, `1300-1303`, and
    `1348-1360` show the same D-003, D-004, then the workspace duplicate.
  - `/tmp/d003-push-full.log` lines around `1498-1501` show the full-mode
    summary assertion after an earlier synthetic `unsafe promotion blocker suite`
    failure.
  - `/tmp/d004-pr-full.log` lines around `1553-1556` show the stabilization-mode
    summary assertion after an earlier synthetic
    `RAM contract fuzz oracle artifact gate` failure.
- Focused local diagnostics:
  - D-003 local diagnostic:
    `go test -buildvcs=false ./cli/cmd/tetra -run 'TestTargetMetadataCheck|TestTargetsCommandJSON$' -count=1`
    returned `ok tetra_language/cli/cmd/tetra 0.401s`; the CI-only D-003 value
    was not reproduced on this host.
  - D-004 local diagnostic:
    `go test -buildvcs=false ./tools/cmd/validate-v0-4-readiness -run TestValidateReadinessAcceptsNativeUIRuntimeEvidenceShape -count=1`
    failed with `decision ui.native-runtime evidence.docs path docs/user/surface/wasm_ui_guide.md is not readable`.
  - Summary-test local diagnostic:
    `go test -buildvcs=false ./tools/scriptstest/test_all -run 'TestTestAllFullToolingSummaryFailsOnZeroByteRequiredArtifact|TestTestAllStabilizationToolingSummaryRequiresFocusedArtifacts' -count=1`
    returned `ok tetra_language/tools/scriptstest/test_all 2.037s`.
- Cache hygiene:
  - Focused Go runs used caches under
    `${XDG_CACHE_HOME:-$HOME/.cache}/tetra-language/go-build-mcv2-ci-context-*`.
  - `go clean -cache` was run for those caches and the temporary directories were
    removed after the evidence runs.

root_cause_assessment:
- Primary CI failures are not the two `test_all` summary assertions.
- D-003 primary signal:
  - CI observed `RunUnsupportedReason:"host linux/amd64 does not support Linux x32 ABI execution; no host fallback is allowed; ..."`
  - The relevant tests check for the stable substring
    `host does not support Linux x32 ABI execution` plus `no host fallback`.
  - This explains the CI failure shape directly and precedes workspace/reporting
    duplicates.
- D-004 primary signal:
  - `nativeUIRuntimeEvidence()` requires
    `docs/user/surface/wasm_ui_guide.md`.
  - `TestValidateReadinessAcceptsNativeUIRuntimeEvidenceShape` builds a reduced
    readiness evidence root and does not include that file before calling
    `validateReadiness`.
  - The same error reproduces locally, so this is direct fixture/path completeness
    evidence rather than a downstream summary-test mutation.
- Summary-test behavior:
  - Both named summary tests construct a synthetic, incomplete repository fixture
    with `testAllFakeRepo`, copying only `scripts/ci/test-all.sh` and creating
    stubbed scripts, `docs/generated/manifest.json`, fake `go`, and fake `tetra`.
  - They alter only the subprocess environment passed to `runTestAll`
    (`TETRA_FAKE_ZERO_DOCTOR_REPORT=1` or
    `TETRA_FAKE_SKIP_WEB_UI_SMOKE_REPORT=1`, plus fake `PATH`); they do not mutate
    the parent test process environment.
  - In CI, an earlier synthetic step failure prevents the expected final
    `tooling summary aggregation` failure, so the assertion reports a nested
    summary shape, not a new root cause.
- One primary failure can produce multiple visible CI failures because
  `baseline-tests` reports the package failure directly, `TestWorkspaceModules`
  reruns module package trees and reports the same package failure again, and the
  fan-in gate records the failed `baseline-tests` step. The two `test_all`
  summary assertions are downstream witnesses, but their internal earlier failing
  step differs between push and PR, so a single shared summary-test root is not
  confirmed.

post_fix_resolution:
- D-003 status: resolved by
  `7e9184aaa2c220590d67f3d369d9598b62861088`.
- D-004 status: resolved by
  `f28953df325cd87cb8378b3c9b7952238b6d3e13`.
- D-001 status: still tracked as nonblocking test-infrastructure hardening; the
  D-003/D-004 logs do not reopen it as a merge blocker.
- Worktree comparison before the fixes:
  - branch HEAD `df3256b904f08b036c5378bcc003c36c0bed5c3e`, tree
    `3d26e5f75a8c2f79e077060175e32fbac1449137`, failed focused and broad
    local comparison runs.
  - PR merge `a1cae0b93fa41cf0ab3319633dabd9c222e38287`, tree
    `c0afac5f9e417c65745dabbf6e3f485317ddfd93`, failed focused and broad
    local comparison runs.
  - Both branch and PR merge comparison logs reproduced D-004 directly; local
    D-003 metadata tests required `linuxX32HostSupport(false)` to force the same
    unsupported branch seen in CI.
- Fix evidence:
  - D-003 RED/GREEN logs:
    `/tmp/mcv2-d003-d004-diagnostics/d003-red.log`,
    `/tmp/mcv2-d003-d004-diagnostics/d003-green.log`.
  - D-004 RED/GREEN logs:
    `/tmp/mcv2-d003-d004-diagnostics/d004-exact-count100-outside-tmp.log`,
    `/tmp/mcv2-d003-d004-diagnostics/d004-green-exact.log`,
    `/tmp/mcv2-d003-d004-diagnostics/d004-green-package.log`.
- Merge recommendation: not ready until the full requested local validation,
  temporary final merge validation, push fan-in, and pull_request fan-in pass on
  the final exact HEAD.

unresolved_risks:
- Full local validation and new exact-HEAD CI have not yet completed in this
  document. PR readiness still depends on those gates.
- The PR merge tree differs from the push tree by a large dumps-only addition.
  No code/test source difference was found between the execution trees, but the
  tree difference remains relevant for checkout/artifact pressure review.
- `TestWorkspaceModules` still inherits ambient environment for nested module
  tests. That is not the D-003/D-004 root here, but it remains part of D-001
  hardening.

verdict: DOWNSTREAM_ONLY_CONFIRMED
