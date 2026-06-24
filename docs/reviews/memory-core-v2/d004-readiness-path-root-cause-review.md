reviewed_commit: 30f1f7bd71c1972bc37ef937ee913ade3b3cbb80

ci_run_id: push 28020556066; pull_request 28020557965

failing_test: `tetra_language/tools/cmd/validate-v0-4-readiness.TestValidateReadinessAcceptsNativeUIRuntimeEvidenceShape`

process_cwd:
- CI gate shell cwd: `/home/runner/work/Tetra_Language/Tetra_Language`
  (`actions/checkout` log working directory; `ui-runtime-gate.sh` also runs
  `cd "$repo_root"` before `baseline-tests`).
- Effective cwd at the failing path check is not the checkout root. The test
  calls `chdirReadinessEvidenceRoot`, which sets cwd to `root := t.TempDir()`
  before invoking `validateReadiness`. The exact randomized CI temp directory
  is not printed in the logs; local diagnostic fixture cwd was
  `/home/tetra/.cache/tetra-language/d004-readiness-fixture-check-2170991`.

repository_root: `/home/runner/work/Tetra_Language/Tetra_Language` in CI; `/home/tetra/.codex/worktrees/Tetra_Language-stabilize-memory-core-v2` for this review checkout.

requested_path: `docs/user/surface/wasm_ui_guide.md`

resolved_path:
- Checkout absolute path:
  `/home/runner/work/Tetra_Language/Tetra_Language/docs/user/surface/wasm_ui_guide.md`
  in CI, and
  `/home/tetra/.codex/worktrees/Tetra_Language-stabilize-memory-core-v2/docs/user/surface/wasm_ui_guide.md`
  locally.
- Failing fixture absolute path:
  `$TEST_TEMP_DIR/docs/user/surface/wasm_ui_guide.md`; local diagnostic
  reproduction resolved it to
  `/home/tetra/.cache/tetra-language/d004-readiness-fixture-check-2170991/docs/user/surface/wasm_ui_guide.md`.

file_exists_in_checkout: yes. `git ls-tree` shows `100644 blob 3d5eb105d4068797f3b01ff207b03e84a9925a76 docs/user/surface/wasm_ui_guide.md` at `HEAD`, `df3256b904f08b036c5378bcc003c36c0bed5c3e`, and PR merge `a1cae0b93fa41cf0ab3319633dabd9c222e38287`. Local filesystem stat is `mode=644 type=regular file size=7381`, and opening the checkout file succeeds.

file_exists_in_fixture: no. `TestValidateReadinessAcceptsNativeUIRuntimeEvidenceShape` creates fixture files for `tools/cmd/native-ui-runtime-smoke/main.go`, `tools/validators/nativeui/report.go`, `docs/spec/core/current_supported_surface.md`, `docs/spec/ui/ui_v0.4.0.md`, and `reports/v0.4.0/native-ui-linux-x64.json`, but not `docs/user/surface/wasm_ui_guide.md`.

file_exists_in_nested_fixture: no. Searching under `tools/cmd/validate-v0-4-readiness` finds no nested `docs/user/surface/wasm_ui_guide.md`; the only matching non-report checkout path is `./docs/user/surface/wasm_ui_guide.md`.

file_mode: checkout git mode `100644`; local filesystem mode `0644`; regular file; not a symlink.

stat_error: in the failing fixture, `os.Stat(filepath.FromSlash("docs/user/surface/wasm_ui_guide.md"))` misses with `no such file or directory`; after ancestor probing, `statFromRepoRoot` returns `os.ErrNotExist`, producing `decision ui.native-runtime evidence.docs path docs/user/surface/wasm_ui_guide.md is not readable`.

open_error: the failing docs evidence path is checked with `os.Stat`, not `os.Open`. A local diagnostic open/read against the same omitted fixture path fails with `No such file or directory`; opening the checkout file succeeds.

root_detection: not git-based. `statFromRepoRoot` first stats the cwd-relative path, then walks upward from `os.Getwd()` to `/` looking for the relative path. Because the test cwd is a temp fixture outside the repository, the walk never reaches the checkout root.

fixture_copy_behavior: fixture copy is manual and allowlist-based. `chdirReadinessEvidenceRoot` writes only the paths passed by the test using `writeReadinessEvidenceFile`; it does not copy the repository docs tree. The native UI evidence helper requires `docs/user/surface/wasm_ui_guide.md`, but the failing test omitted that path from the fixture allowlist.

validator_outside_repository_root: yes at the failing validation boundary. The gate starts at repo root, but the unit test intentionally changes cwd to a temp fixture before calling `validateReadiness`.

symlink_path_cleaning: no symlink or path-cleaning root cause found. The checkout file is not a symlink. Validator normalization uses `strings.TrimSpace`, `filepath.ToSlash`, and `filepath.FromSlash`; it does not call `filepath.Clean` or `EvalSymlinks` for this evidence path.

generated_manifest_path: not the D-004 cause. `docs/generated/manifest.json` contains `docs/user/surface/wasm_ui_guide.md`. `docs/generated/v1_0/manifest.json` and `docs/release/v0_4/data/v0_4_0_scope_decisions.json` contain legacy `docs/user/wasm_ui_guide.md` entries, but the failing test uses inline `nativeUIRuntimeEvidence()` and fails on `docs/user/surface/wasm_ui_guide.md`.

pull_request_merge_checkout: no root-detection change found. The PR run checks out merge commit `a1cae0b93fa41cf0ab3319633dabd9c222e38287`; that merge has the same fixture omission and the same `100644` docs file as the push commit. The failure mechanism is the test temp cwd, not PR merge checkout shape.

reproduction:
- CI evidence: both fan-in logs show `baseline-tests` running from
  `scripts/release/full_platform/ui-runtime-gate.sh`, then failing
  `TestValidateReadinessAcceptsNativeUIRuntimeEvidenceShape` with the exact
  unreadable docs path. The same failure is repeated through
  `TestWorkspaceModules/tools`.
- Source evidence: the omission is identical at reviewed `HEAD`, push commit
  `df3256b904f08b036c5378bcc003c36c0bed5c3e`, and PR merge
  `a1cae0b93fa41cf0ab3319633dabd9c222e38287`.
- Local diagnostic fixture evidence: creating exactly the five paths passed to
  `chdirReadinessEvidenceRoot` and then checking
  `docs/user/surface/wasm_ui_guide.md` yields `stat: cannot statx ... No such
  file or directory`; no matching fixture file is found.
- Local `go test` was attempted with persistent `GOCACHE`, but the host Go
  toolchain failed during setup because its configured `/usr/lib/go` stdlib is
  incomplete. That failure did not reach the D-004 test and is not used as D-004
  evidence.

root_cause: fixture omission. The repository checkout contains the requested docs file, but the passing-shape unit test changes cwd to an isolated temp fixture and does not create the docs file required by `nativeUIRuntimeEvidence().Docs`.

minimal_fix: add `docs/user/surface/wasm_ui_guide.md` to the `chdirReadinessEvidenceRoot` path list in `TestValidateReadinessAcceptsNativeUIRuntimeEvidenceShape`.

regression_test: run `go test ./tools/cmd/validate-v0-4-readiness -run '^TestValidateReadinessAcceptsNativeUIRuntimeEvidenceShape$' -count=1`, then rerun the broader `go test ./tools/cmd/validate-v0-4-readiness -count=1`. CI coverage should also include the existing `baseline-tests` command from `scripts/release/full_platform/ui-runtime-gate.sh`.

resolution_status: resolved

fix_commit: `f28953df325cd87cb8378b3c9b7952238b6d3e13`

before_behavior:
- With `TMPDIR` outside the repository, the test cwd was an isolated fixture
  that contained the native UI runtime implementation, validator, specs, and
  report files, but omitted `docs/user/surface/wasm_ui_guide.md`.
- `validateEvidencePaths` therefore reported
  `decision ui.native-runtime evidence.docs path docs/user/surface/wasm_ui_guide.md is not readable`.

after_behavior:
- `TestValidateReadinessAcceptsNativeUIRuntimeEvidenceShape` copies
  `docs/user/surface/wasm_ui_guide.md` into the same fixture as the other
  required native UI runtime evidence paths.
- `docs/user/surface/wasm_ui_guide.md` content was not changed, and the
  readiness validator still treats the guide as required evidence.

regression_tests:
- RED:
  `go test -buildvcs=false ./tools/cmd/validate-v0-4-readiness -run '^TestValidateReadinessAcceptsNativeUIRuntimeEvidenceShape$' -count=100 -v`
  failed with an outside-repository `TMPDIR`.
- GREEN:
  `go test -buildvcs=false ./tools/cmd/validate-v0-4-readiness -run '^TestValidateReadinessAcceptsNativeUIRuntimeEvidenceShape$' -count=100 -v`
  passed with an outside-repository `TMPDIR`.
- GREEN:
  `go test -buildvcs=false ./tools/cmd/validate-v0-4-readiness -count=50 -v`
  passed with an outside-repository `TMPDIR`.

merge_recommendation: pending full local validation and new exact-HEAD CI; do
not mark PR ready until all requested gates pass.

forbidden_fixes:
- Do not edit `docs/user/surface/wasm_ui_guide.md`.
- Do not weaken `validateEvidencePaths` readability checks.
- Do not remove the docs evidence requirement from `nativeUIRuntimeEvidence`.
- Do not change generated manifests to mask the fixture omission.
- Do not add symlinks or repository-root fallbacks that let unit fixtures silently read unrelated checkout files.

verdict: ROOT_CAUSE_CONFIRMED
