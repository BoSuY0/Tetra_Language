# Actor Foundation RC100 Notes

## Scope And Nonclaims

- Current mission is the RC100 implementation plan:
  `/home/tetra/Downloads/2026-06-10-actor-foundation-prod-ready-scoped-rc-100-implementation-plan.md`.
- Evidence root is `reports/actor-foundation-prod-ready-scoped-rc-100/`.
- Target claim is `ACTOR_FOUNDATION_PROD_READY_SCOPED_RC_100_PERC`.
- The target is scoped actor foundation release-candidate readiness, not full
  actor runtime production.
- Do not claim full actor runtime production, Erlang/OTP parity, cluster
  membership, reconnect/retry/TLS/auth production, non-Linux distributed actor
  runtime support, distributed zero-copy pointer/region transfer, formal
  race/liveness proof, official benchmark status, or performance superiority.

## Durable Discoveries

- Current HEAD for P00 is
  `c0258b63a636775b114d69d31cb7832fc3991b05`.
- Implementation plan SHA-256 is
  `ef0d9a1f89ecce5b6c482cd5cc673b40d9f7b50316daea9df219a1eba5e0e48b`.
- Previous P15 final-production local evidence reached
  `PROD_STABLE_SCOPED_DIRTY_OR_STALE`, not RC100.
- Previous P15 local gate proof under
  `reports/actor-final-production/P15/foundation-gate-rerun1/` is supporting
  baseline only because clean checkout, remote CI, and package workflow proof
  were not claimed.
- Main worktree is dirty; `ACTOR-RC100-P01` must use an isolated worktree or
  record a safe blocker. Do not destructively clean the main worktree.
- `reports/` is ignored by `.gitignore`, so evidence artifacts are not shown by
  default in `git status --short`.
- `ACTOR-RC100-P01` created clean detached worktree
  `/home/tetra/.codex/worktrees/Tetra_Language/actor-rc100-clean-c0258` at
  `c0258b63a636775b114d69d31cb7832fc3991b05`. Its status, `git diff --check`,
  and `docs/generated/manifest.json` diff checks are empty. It does not include
  uncommitted tracker/evidence/doc changes from the dirty main worktree.
- `ACTOR-RC100-P02` found that committed `c0258...` could not pass the clean
  gate because `docs/generated/manifest.json` was stale/missing
  `compiler.ram-contracts`. The failure is preserved under
  `reports/actor-foundation-prod-ready-scoped-rc-100/P02/foundation-gate-initial-failed/`.
- `ACTOR-RC100-P02` created local clean RC branch `actor-rc100-clean-proof`
  in the isolated worktree with commits
  `867956f3d83a913565e55e7898678008b11370af` and
  `6ce068ba08f69f8d292c38fc455ab7dfb0334061`, then P03 test hardening commit
  `f47d0dcc0b42784f318844621d6a2ba8ce3e31fb`. Final local gate reports are
  under
  `/home/tetra/.codex/worktrees/Tetra_Language/actor-rc100-clean-c0258/reports/actor-foundation-prod-ready-scoped-rc-100/local/foundation-gate/`.
- Final P02 clean status logs before and after the gate are empty, and
  `validate-actor-runtime-foundation`, `validate-distributed-actor-runtime`
  with `--current-git-head`, `validate-parallel-production`, and all three
  artifact hash validators passed for
  `f47d0dcc0b42784f318844621d6a2ba8ce3e31fb`.
- `ACTOR-RC100-P03` copied the clean gate artifacts to the canonical main
  evidence root under
  `reports/actor-foundation-prod-ready-scoped-rc-100/local/foundation-gate/`.
  The copied report validates with `--current-git-head
  f47d0dcc0b42784f318844621d6a2ba8ce3e31fb`.
- Root `reports/actor-foundation-prod-ready-scoped-rc-100/artifact-hashes.json`
  validated at generation time with 166 artifacts; its SHA-256 is recorded in
  `reports/actor-foundation-prod-ready-scoped-rc-100/final-artifact-sha256.txt`.
  Post-validation P03 summary/status files are intentionally outside that
  validation moment and final P07/P08 may regenerate the root manifest.
- Historical `reports/actor-runtime-foundation/final` is rejected as current
  proof because its `git_head` is `e2c19b8...`, not the clean RC head.
- `ACTOR-RC100-P04` static CI workflow proof passed. Remote CI proof is blocked:
  `actor-rc100-clean-proof` is not present on `origin`, exact SHA
  `f47d0dcc0b42784f318844621d6a2ba8ce3e31fb` is not present in
  `git ls-remote origin`, and `gh auth status` reports invalid GitHub tokens.
- `ACTOR-RC100-P05` static release-package proof passed. The actor runtime
  foundation release gate is ordered before artifact upload, GitHub Release,
  container, and Homebrew publishing steps. Real package workflow proof remains
  blocked without explicit run/dry-run approval and valid GitHub auth.
- `ACTOR-RC100-P06` static workflow hardening passed. Raw bypass-marker audit
  found 76 markers in legacy/other release scripts or cleanup/diagnostic
  fallbacks, while targeted actor RC workflow/gate scan found no
  `continue-on-error`, `|| true`, or `set +e` markers.
- `ACTOR-RC100-P07` committed clean branch
  `f417202a7fd611b4da0059b57c54a63a9e86f81e`, updating the final actor
  foundation audit, adding
  `docs/plans/2026-06-10-actor-runtime-post-scope-blockers.md`, regenerating
  `docs/generated/manifest.json`, and hardening `verify-docs` to require that
  post-scope ledger in `actors.distributed-runtime` docs. Post-commit docs,
  manifest, `git diff --check`, and clean status checks passed.
- `ACTOR-RC100-P08` reran same-head actor evidence on
  `f417202a7fd611b4da0059b57c54a63a9e86f81e`. Actor core packages,
  validators, focused actor checks, race slice, distributed smoke, parallel
  smoke, actor foundation gate, copied report validation, docs/manifest,
  graphify update, final hash manifest, and clean status checks passed.
- P08 full broad compiler/CLI/tools validation is reproducibly failing outside
  the actor RC slice: `tools/cmd/validate-surface-runtime` morph release-summary
  tests, `tools/scriptstest` structural/example-index tests, and
  `tools/validators/postv04prod` fixture artifact checks. Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P08/broad-failure-debug-status.tsv`.
- P08 final verdict is `BLOCKED`; strongest actor-specific state is
  `ACTOR_FOUNDATION_RC_LOCAL_CLEAN_REMOTE_BLOCKED`. Target remains unclaimed
  until broad validation passes and same-commit remote CI/package workflow proof
  exists.
- P09 broad validation unblock passed in the current worktree. The original P08
  focused broad failures now pass, `tools/scriptstest/workspace_modules_test.go`
  isolates nested module `GOTMPDIR` under `.cache/go-tmp-workspace-modules-*`,
  and
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-actor-rc100 GOTMPDIR=$(pwd)/.cache/go-tmp-actor-rc100 go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1`
  passed. Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P09-broad-fix/broad-rerun2-status.tsv`.
- P09 remote CI remains blocked with
  `BLOCKED_REMOTE_CI_USER_ACTION_REQUIRED`: local branch `main` is at
  `c0258b63a636775b114d69d31cb7832fc3991b05`, `origin/main` is at
  `3e489e567edc6ab7e537594313a9719a473aea38`, the broad-green worktree is dirty,
  and `gh auth status` reports invalid tokens. Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P09-broad-fix/remote-ci-instruction-bundle.md`.
- P09 closeout checks passed: `graphify update .`, `git diff --check`, and
  cleanup of the actor-rc100 Go caches/temp dirs. Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P09-broad-fix/graphify-update-status.tsv`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P09-broad-fix/git-diff-check-status.tsv`,
  and
  `reports/actor-foundation-prod-ready-scoped-rc-100/P09-broad-fix/cache-clean-status.tsv`.
- P10 rechecked the remote blocker: `gh auth status` still reports invalid
  tokens, no same-change remote branch/commit is available, and the current
  dirty `main` base HEAD `c0258b63a636775b114d69d31cb7832fc3991b05` is not a
  remote proof SHA. Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P10-remote-blocker-recheck/summary.md`.
- P10 hygiene checks passed: `git diff --check` and JSON validity for
  `P09-broad-fix/remote-ci-blocker.json` and
  `P10-remote-blocker-recheck/summary.json`.
- Blocked audit count after P10: same remote blocker has repeated for the second
  consecutive resumed goal turn after prior blocked state. If the next resumed
  goal turn rechecks and sees the same `gh`/same-change commit blocker, the
  strict blocked threshold is satisfied.
- P11 rechecked the same blocker for the third consecutive resumed goal turn:
  P09 broad is still PASS, `gh auth status` still fails, and no same-change
  remote branch/commit exists. Strict blocked threshold is satisfied. Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P11-remote-blocked-final/summary.md`.
- P11 hygiene checks passed: `git diff --check`, JSON validity for blocker
  summaries, and `graphify update .`.
- P13 resumed after user login: `gh auth status` is valid for `BoSuY0` with
  `repo` and `workflow` scopes. Clean candidate branch
  `actor-rc100-p12-expanded-clean` passed the required broad compiler/CLI/tools
  command; evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P13-broad-green/broad-status.tsv`
  and
  `reports/actor-foundation-prod-ready-scoped-rc-100/P13-broad-green/broad-compiler-cli-tools.log`.
- P13 added candidate safeguards for PLIR borrowed return ownership,
  explicit-island cross-module summary validation, and monomorphized generic
  cache invalidation. Candidate patch artifacts:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P13-broad-green/candidate-selected-tracked.patch`
  and
  `reports/actor-foundation-prod-ready-scoped-rc-100/P13-broad-green/candidate-newfiles.patch`.
- P13 remote CI bundle is prepared at
  `reports/actor-foundation-prod-ready-scoped-rc-100/P13-broad-green/remote-ci-instruction-bundle.md`.
  The agent did not commit, push, or run GitHub Actions without explicit user
  approval.
- P15 remote proof audit repeated the same approval gate: broad remains PASS,
  `gh auth` remains valid, but `origin` has no
  `actor-rc100-p12-expanded-clean` branch, the candidate diff is uncommitted,
  and remote CI/package workflows are not run. Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P15-remote-proof-still-pending/status.tsv`.
- P16 repeated the same approval gate for the third consecutive resumed turn.
  Strict blocked threshold is reached for commit/push/remote CI approval.
  Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P16-approval-blocked-final/status.tsv`.
- User approval on 2026-06-10 unblocked commit, push, and remote evidence
  execution. P17 preflight confirmed the clean candidate worktree still has a
  whitespace-clean diff, `BoSuY0/Tetra_Language` repository access is available,
  and `ci.yml` / `release-packages.yml` are active. `gh auth status` still
  returns non-zero because old inactive accounts have invalid tokens, but the
  active `BoSuY0` account and repository-scoped commands work.
- P17 pushed candidate commit `3480870c7ff52d211aaa63c16238e62d6165cfbd` to
  `origin/actor-rc100-p12-expanded-clean` and dispatched `ci.yml` run
  `27284988190` on the same SHA. All jobs failed before steps started
  (`steps=0`, empty runner name), and check-run annotations say:
  `The job was not started because your account is locked due to a billing issue.`
  `ci-artifacts.json` has `total_count=0`, and downloading
  `tetra-actor-runtime-foundation-3480870c7ff52d211aaa63c16238e62d6165cfbd-linux-x64`
  failed with `no valid artifacts found to download`.

## Active Bridge

- Scoped RC packets plus P13 clean broad-green candidate are complete. P17
  proved same-change commit/push, but same-change remote CI/package proof is
  blocked by GitHub account billing lock. The target remains unclaimed until
  billing is resolved and the workflows produce downloadable artifacts.
