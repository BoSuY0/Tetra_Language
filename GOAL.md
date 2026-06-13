# Actor Foundation RC100 Goal

<goal>
Implement the full plan in
`/home/tetra/Downloads/2026-06-10-actor-foundation-prod-ready-scoped-rc-100-implementation-plan.md`.

Mission: drive the scoped Actor Foundation from the previous
`PROD_STABLE_SCOPED_DIRTY_OR_STALE` baseline to the strongest honest
`ACTOR_FOUNDATION_PROD_READY_SCOPED_RC_100_PERC` state.
</goal>

<context>
Active `/goal` objective:

`$todium-superpower Реалізуй повністю весь цей план - /home/tetra/Downloads/2026-06-10-actor-foundation-prod-ready-scoped-rc-100-implementation-plan.md`

Primary source of scope:

- `/home/tetra/Downloads/2026-06-10-actor-foundation-prod-ready-scoped-rc-100-implementation-plan.md`

Working evidence root:

- `reports/actor-foundation-prod-ready-scoped-rc-100/`

Baseline:

- Previous final-production evidence under `reports/actor-final-production/P15/`
  reached `PROD_STABLE_SCOPED_DIRTY_OR_STALE`.
- Previous P15 local gate proof is supporting evidence only; it does not prove
  clean checkout, remote CI, or package workflow proof for RC100.
</context>

<constraints>
- Always communicate with the user in Ukrainian.
- Preserve unrelated dirty worktree changes.
- Never use destructive cleanup (`git reset --hard`, `git clean -fdx`,
  deleting user files) to obtain clean proof.
- Use persistent Go caches under `.cache/`, `.gocache`, or
  `${XDG_CACHE_HOME:-$HOME/.cache}/tetra-language/...`; never set `GOCACHE` to
  `/tmp`.
- Use `GOTELEMETRY=off` for Go evidence commands.
- Do not accept stale reports, docs-only evidence, build-only evidence, or
  historical actor evidence as current same-commit RC100 proof.
- Do not weaken tests, validators, release gates, nonclaims, CI workflows, or
  package workflow guards to obtain a stronger claim.
- Do not claim full actor runtime production, Erlang/OTP parity, cluster
  membership, reconnect/retry/TLS/auth production, non-Linux distributed actor
  runtime support, distributed zero-copy pointer/region transfer, formal
  race/liveness proof, official benchmark status, or performance superiority.
- Remote CI/package proof may be recorded only if actually run, inspected, and
  evidenced, or if an approved dry-run proof is explicitly allowed by the plan
  and user permissions.
- After modifying code, run `graphify update .` before closing the packet.
</constraints>

<scorecard>
Target claim:

- `ACTOR_FOUNDATION_PROD_READY_SCOPED_RC_100_PERC`

Allowed lower verdicts:

- `ACTOR_FOUNDATION_RC_LOCAL_CLEAN`
- `ACTOR_FOUNDATION_RC_LOCAL_CLEAN_REMOTE_BLOCKED`
- `ACTOR_FOUNDATION_RC_BLOCKED_CLEAN_CHECKOUT`
- `ACTOR_FOUNDATION_RC_BLOCKED_PACKAGE_PROOF`
- `BLOCKED`

The target is achieved only when clean local proof, remote CI proof, package
workflow proof or approved dry-run proof, artifact hashes, docs/manifest, final
handoff, and residual nonclaims all satisfy the implementation plan.
</scorecard>

<done_when>
The goal is complete only when every scoped RC packet is completed or an
explicit blocker/nonclaim prevents the target:

- `ACTOR-RC100-P00` records current HEAD, dirty state, implementation plan hash,
  and stale evidence classification.
- `ACTOR-RC100-P01` obtains a clean isolated checkout or records the exact safe
  blocker.
- `ACTOR-RC100-P02` proves the clean local actor foundation gate and validators.
- `ACTOR-RC100-P03` creates and validates the RC artifact hash/stale-evidence
  bundle.
- `ACTOR-RC100-P04` records remote CI actor foundation proof or a precise
  remote blocker.
- `ACTOR-RC100-P05` records package workflow proof or an approved dry-run proof,
  or a precise permission blocker.
- `ACTOR-RC100-P06` proves CI/workflow hardening against bypass and stale/docs
  evidence.
- `ACTOR-RC100-P07` updates final audit/docs/manifest/handoff only after
  executable evidence passes.
- `ACTOR-RC100-P08` performs final acceptance and locks the exact verdict.
- Optional `ACTOR-RC100-FULL-*` packets remain separate full actor runtime
  follow-on work and must not be mixed into the scoped RC100 claim.
</done_when>

<feedback_loop>
Packet loop: run packet-specific commands with `GOTELEMETRY=off`,
repo-local `GOCACHE`, and repo-local `GOTMPDIR`; record evidence under
`reports/actor-foundation-prod-ready-scoped-rc-100/PXX/`; update tracker files
after each packet.

Final loop: run the full validation ladder from the implementation plan,
inspect current evidence, and choose the exact truthful verdict.
</feedback_loop>

<working_memory>
Maintain:

- `GOAL.md`: canonical objective, acceptance, and progress.
- `PLAN.md`: packet matrix and current strategy.
- `ATTEMPTS.md`: completed attempts and evidence links.
- `NOTES.md`: durable discoveries and blockers.
- `CONTROL.md`: active packet, next actions, stop conditions, cache rules.
</working_memory>

## Progress

- 2026-06-10: User approval unblocked the P16 gate for committing, pushing,
  and running remote evidence. Bridge: `ACTOR-RC100-P17` is in progress from
  clean candidate worktree
  `/home/tetra/.codex/worktrees/Tetra_Language/actor-rc100-p12-expanded-clean`.
  Preflight found `git diff --check` clean, valid repository access as
  `BoSuY0/Tetra_Language`, and active `ci.yml` / `release-packages.yml`
  workflows. Next evidence: same-change candidate commit SHA, pushed branch,
  remote `ci` actor runtime foundation artifact, and release package proof.
- 2026-06-10: `ACTOR-RC100-P17` completed with blocker
  `REMOTE_CI_BILLING_LOCKED`. Candidate commit
  `3480870c7ff52d211aaa63c16238e62d6165cfbd` was pushed to
  `origin/actor-rc100-p12-expanded-clean`, and `ci.yml` run `27284988190`
  targeted the same SHA. GitHub Actions did not start jobs because the account
  is locked due to a billing issue; `actor-runtime-foundation-linux` has
  `steps=0`, no runner, no artifact, and check-run annotations record the
  billing lock. Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P17-remote-execution-billing-blocked/summary.md`,
  `status.tsv`, `ci-run.json`, `ci-jobs.json`, `ci-check-annotations.jsonl`,
  and `ci-artifacts.json`. Target
  `ACTOR_FOUNDATION_PROD_READY_SCOPED_RC_100_PERC` remains `NOT_CLAIMED`.
- 2026-06-10: `ACTOR-RC100-P18` started after the user installed `act` and
  asked the agent to run the local emulation. Scope: run the
  `actor-runtime-foundation-linux` GitHub Actions job locally on a fresh
  worktree at commit `3480870c7ff52d211aaa63c16238e62d6165cfbd`, capture
  artifact/report evidence under
  `reports/actor-foundation-prod-ready-scoped-rc-100/P18-act-local/`, and keep
  the target unclaimed unless remote GitHub Actions/package proof also exists.
- 2026-06-10: Goal realigned from the completed final-production plan to the
  active RC100 implementation plan. Bridge: `ACTOR-RC100-P00` is completed and
  `ACTOR-RC100-P01` is next. Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P00/summary.md`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P00/summary.json`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P00/command-status.tsv`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P00/git-status-short.log`,
  and
  `reports/actor-foundation-prod-ready-scoped-rc-100/P00/stale-evidence-classification.tsv`.
- 2026-06-10: `ACTOR-RC100-P01` completed. Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P01/summary.md`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P01/summary.json`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P01/command-status.tsv`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P01/clean-worktree-path.txt`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P01/clean-git-status-short.log`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P01/git-diff-check.log`,
  and
  `reports/actor-foundation-prod-ready-scoped-rc-100/P01/manifest-diff-exit-code.log`.
  Clean detached worktree:
  `/home/tetra/.codex/worktrees/Tetra_Language/actor-rc100-clean-c0258`.
  Bridge: next packet is `ACTOR-RC100-P02` clean local actor foundation gate
  rerun in the isolated worktree.
- 2026-06-10: `ACTOR-RC100-P02` completed. Clean RC worktree branch
  `actor-rc100-clean-proof` reached
  `f47d0dcc0b42784f318844621d6a2ba8ce3e31fb`; final local gate and validators
  passed from a clean committed checkout. Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P02/summary.md`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P02/summary.json`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P02/command-status.tsv`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P02/foundation-gate-final-status.tsv`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P02/final-validate-actor-runtime-foundation-status.tsv`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P02/final-validate-distributed-actor-runtime-status.tsv`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P02/final-validate-parallel-production-status.tsv`,
  and final artifact hash validator status files. Bridge: next packet is
  `ACTOR-RC100-P03` final artifact hash and stale-evidence bundle.
- 2026-06-10: `ACTOR-RC100-P03` completed. The canonical RC evidence root
  has a validated artifact hash manifest and `final-artifact-sha256.txt`;
  historical `reports/actor-runtime-foundation/final` evidence is rejected as
  current proof for clean HEAD
  `f47d0dcc0b42784f318844621d6a2ba8ce3e31fb`. Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P03/summary.md`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P03/summary.json`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P03/command-status.tsv`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P03/stale-evidence-bundle.tsv`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/artifact-hashes.json`,
  and `reports/actor-foundation-prod-ready-scoped-rc-100/final-artifact-sha256.txt`.
  Bridge: next packet is `ACTOR-RC100-P04` remote CI actor foundation proof or
  precise remote blocker.
- 2026-06-10: `ACTOR-RC100-P04` completed with blocker `REMOTE_CI_BLOCKED`.
  Static CI workflow proof passed, but exact clean commit
  `f47d0dcc0b42784f318844621d6a2ba8ce3e31fb` is not present on `origin` and
  `gh auth status` reports invalid tokens, so no remote run/artifact proof can
  be honestly recorded in this session. Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P04/summary.md`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P04/summary.json`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P04/command-status.tsv`,
  and
  `reports/actor-foundation-prod-ready-scoped-rc-100/remote-ci/actor-runtime-foundation-run.json`.
  Bridge: next packet is `ACTOR-RC100-P05` release package workflow proof or
  approved dry-run / permission blocker.
- 2026-06-10: `ACTOR-RC100-P05` completed with blocker
  `PACKAGE_WORKFLOW_PROOF_BLOCKED`. Static release-package workflow proof
  passed and actor gate ordering/artifact inclusion are covered, but no real
  package workflow run or approved dry-run was executed. Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P05/summary.md`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P05/summary.json`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P05/command-status.tsv`,
  and
  `reports/actor-foundation-prod-ready-scoped-rc-100/package-workflow/release-packages-run.json`.
  Bridge: next packet is `ACTOR-RC100-P06` CI/workflow hardening.
- 2026-06-10: `ACTOR-RC100-P06` completed. Workflow hardening tests passed;
  raw bypass-marker findings were classified as legacy/other scripts or
  cleanup/diagnostic fallbacks, and targeted actor RC workflows/gate scan found
  no bypass markers affecting actor RC gates. Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P06/summary.md`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P06/summary.json`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P06/command-status.tsv`,
  and
  `reports/actor-foundation-prod-ready-scoped-rc-100/P06/workflow-hardening.log`.
  Bridge: next packet is `ACTOR-RC100-P07` final audit/docs/handoff update.
- 2026-06-10: `ACTOR-RC100-P07` completed. Clean RC branch
  `actor-rc100-clean-proof` reached
  `f417202a7fd611b4da0059b57c54a63a9e86f81e`; final audit/docs/manifest were
  updated without claiming the target, and `verify-docs` now requires the
  post-scope actor blocker ledger in the distributed actor feature manifest.
  Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P07/summary.md`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P07/summary.json`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P07/post-commit-command-status.tsv`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P07/final-handoff.md`,
  and
  `reports/actor-foundation-prod-ready-scoped-rc-100/P07/final-artifact-sha256.txt`.
  Bridge: next packet is `ACTOR-RC100-P08` final acceptance gate and verdict
  lock for clean HEAD `f417202a7fd611b4da0059b57c54a63a9e86f81e`.
- 2026-06-10: `ACTOR-RC100-P08` completed with lower truthful verdict
  `BLOCKED`; strongest actor-specific state reached is
  `ACTOR_FOUNDATION_RC_LOCAL_CLEAN_REMOTE_BLOCKED`, and target
  `ACTOR_FOUNDATION_PROD_READY_SCOPED_RC_100_PERC` remains `NOT_CLAIMED`.
  Same-head actor foundation gate, current-head validator, copied report
  validation, docs/manifest, graphify update, and final artifact hash manifest
  passed for clean HEAD `f417202a7fd611b4da0059b57c54a63a9e86f81e`.
- 2026-06-10: P09 continuation fixed the local broad validation blocker in the
  current worktree. Original P08 focused failures now pass, `TestWorkspaceModules`
  has repo-local `GOTMPDIR` isolation, and the required broad command
  `go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1`
  passed with `GOTELEMETRY=off`, repo-local `GOCACHE`, and repo-local
  `GOTMPDIR`. Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P09-broad-fix/repro-status.tsv`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P09-broad-fix/post-patch-targeted-status.tsv`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P09-broad-fix/scriptstest-repeat-status.tsv`,
  and
  `reports/actor-foundation-prod-ready-scoped-rc-100/P09-broad-fix/broad-rerun2-status.tsv`.
  Remote CI/package proof remains blocked with
  `BLOCKED_REMOTE_CI_USER_ACTION_REQUIRED` because `gh auth status` reports
  invalid tokens and the broad-green worktree is dirty/unpushed. Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P09-broad-fix/remote-ci-instruction-bundle.md`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P09-broad-fix/remote-ci-blocker.json`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P09-broad-fix/gh-auth-status.log`,
  and
  `reports/actor-foundation-prod-ready-scoped-rc-100/P09-broad-fix/remote-ci-preflight-status.tsv`.
  Closeout checks passed: `graphify update .`, `git diff --check`, and Go cache
  cleanup; see `P09-broad-fix/graphify-update-status.tsv`,
  `P09-broad-fix/git-diff-check-status.tsv`, and
  `P09-broad-fix/cache-clean-status.tsv`.
  Bridge: next action is user GitHub re-auth plus a pushed same-change commit,
  then real `ci`/`actor-runtime-foundation-linux` and `release-packages` proof.
  Remaining target blockers are `REMOTE_CI_BLOCKED` and
  `PACKAGE_WORKFLOW_PROOF_BLOCKED`; `BROAD_VALIDATION_FAILED` is cleared by P09
  local evidence.
- 2026-06-10: P10 remote blocker recheck repeated the current external blocker:
  P09 broad status remains `PASS`, but `gh auth status` still fails with invalid
  GitHub tokens, no same-change remote branch/commit is available, and the
  broad-green worktree is still dirty. The remote CI bundle now explicitly marks
  `c0258b63a636775b114d69d31cb7832fc3991b05` as the local dirty base HEAD, not
  a remote proof SHA. Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P10-remote-blocker-recheck/summary.md`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P10-remote-blocker-recheck/summary.json`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P10-remote-blocker-recheck/remote-blocker-recheck-status.tsv`,
  and
  `reports/actor-foundation-prod-ready-scoped-rc-100/P09-broad-fix/remote-ci-instruction-bundle.md`.
  Hygiene checks passed: `P10-remote-blocker-recheck/git-diff-check-status.tsv`
  and `P10-remote-blocker-recheck/json-validity-status.tsv`.
  Bridge: a fresh continuation should recheck `gh auth status`; if the same
  blocker repeats for the third consecutive resumed goal turn, the strict
  blocked-audit threshold is satisfied.
- 2026-06-10: P11 remote blocked final rechecked the same external blocker for
  the third consecutive resumed goal turn after the prior blocked state. P09
  broad validation remains `PASS`, but `gh auth status` still fails with invalid
  tokens, no same-change remote branch/commit exists, and the worktree remains
  dirty. Strict blocked-audit threshold is satisfied. Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P11-remote-blocked-final/summary.md`,
  `reports/actor-foundation-prod-ready-scoped-rc-100/P11-remote-blocked-final/summary.json`,
  and
  `reports/actor-foundation-prod-ready-scoped-rc-100/P11-remote-blocked-final/remote-blocked-final-status.tsv`.
  Hygiene checks passed: `P11-remote-blocked-final/git-diff-check-status.tsv`,
  `P11-remote-blocked-final/json-validity-status.tsv`, and
  `P11-remote-blocked-final/graphify-update-status.tsv`.
  Current target blockers are `REMOTE_CI_BLOCKED` and
  `PACKAGE_WORKFLOW_PROOF_BLOCKED`; the historical P08 root final verdict still
  records the old P08 broad failure, while P09/P11 evidence is the current
  broad-green remote-blocked continuation state.
- 2026-06-10: P13 resumed after user GitHub login. `gh auth status` now passes
  for `BoSuY0` with `repo` and `workflow` scopes, the scoped candidate was
  integrated in clean worktree
  `/home/tetra/.codex/worktrees/Tetra_Language/actor-rc100-p12-expanded-clean`,
  and the required broad command passed with evidence logged to
  `reports/actor-foundation-prod-ready-scoped-rc-100/P13-broad-green/broad-status.tsv`
  and
  `reports/actor-foundation-prod-ready-scoped-rc-100/P13-broad-green/broad-compiler-cli-tools.log`.
  Additional fixes since P11 cover PLIR borrowed region summary ownership,
  explicit island return-summary validation across module cache boundaries, and
  generic-collections cache invalidation for monomorphized module-local
  functions/types. Candidate artifacts:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P13-broad-green/candidate-selected-tracked.patch`
  and
  `reports/actor-foundation-prod-ready-scoped-rc-100/P13-broad-green/candidate-newfiles.patch`.
  Remote CI instruction bundle is prepared at
  `reports/actor-foundation-prod-ready-scoped-rc-100/P13-broad-green/remote-ci-instruction-bundle.md`.
  Bridge: `BROAD_VALIDATION_FAILED` and `gh auth invalid` are cleared; remaining
  proof gap is a user-approved commit/push plus real `ci` job
  `actor-runtime-foundation-linux` and release package workflow evidence.
- 2026-06-10: P15 repeated the post-P13 remote proof audit without approval.
  P13 broad remains `PASS`, `gh auth status` remains valid for `BoSuY0`, but
  `git ls-remote --heads origin actor-rc100-p12-expanded-clean` returns no
  remote branch, the clean candidate worktree still has uncommitted scoped diff,
  and neither `ci` job `actor-runtime-foundation-linux` nor `release-packages`
  has been run by the agent. Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P15-remote-proof-still-pending/status.tsv`
  and
  `reports/actor-foundation-prod-ready-scoped-rc-100/P15-remote-proof-still-pending/summary.md`.
  Bridge: waiting for explicit approval to commit/push and run remote CI; RC100
  remains `NOT_CLAIMED`.
- 2026-06-10: P16 repeated the same post-P13 approval gate for the third
  consecutive resumed turn (`P14`/`P15`/`P16`). P13 broad remains `PASS`, `gh`
  auth remains valid, but no remote branch/commit exists for
  `actor-rc100-p12-expanded-clean`, the candidate diff remains uncommitted, and
  remote `ci`/`release-packages` proof cannot be created without explicit
  approval for external writes/execution. Evidence:
  `reports/actor-foundation-prod-ready-scoped-rc-100/P16-approval-blocked-final/status.tsv`
  and
  `reports/actor-foundation-prod-ready-scoped-rc-100/P16-approval-blocked-final/summary.md`.
  Strict blocked-audit threshold is reached. Goal status should be `blocked`;
  RC100 remains `NOT_CLAIMED`.
