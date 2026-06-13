# Actor Foundation RC100 Plan Tracker

External plan:
`/home/tetra/Downloads/2026-06-10-actor-foundation-prod-ready-scoped-rc-100-implementation-plan.md`

Evidence root:
`reports/actor-foundation-prod-ready-scoped-rc-100/`

## Current Strategy

1. Treat the external Markdown file as the execution contract.
2. Execute scoped packets `ACTOR-RC100-P00` through `ACTOR-RC100-P08` in order,
   unless a packet is proven already green on current HEAD and current state.
3. Preserve the current dirty source worktree; use an isolated clean worktree
   for release-candidate proof instead of destructive cleanup.
4. Do not claim `ACTOR_FOUNDATION_PROD_READY_SCOPED_RC_100_PERC` unless clean
   local proof, remote CI proof, and package workflow proof or approved dry-run
   proof all exist.
5. Keep full actor runtime production blockers separate under the optional
   `ACTOR-RC100-FULL-*` track.
6. User changed closeout scope on 2026-06-13: GitHub Actions proof is waived
   for completion, and local-only evidence may close the goal as
   `ACTOR_FOUNDATION_RC_LOCAL_ONLY_OK`; the remote RC100 target remains
   unclaimed.

## Packet Matrix

| Packet | Status | Acceptance Evidence |
| --- | --- | --- |
| `ACTOR-RC100-P00` preflight and target freeze | completed | `reports/actor-foundation-prod-ready-scoped-rc-100/P00/summary.md`, `summary.json`, `command-status.tsv` |
| `ACTOR-RC100-P01` clean checkout / isolated worktree RC proof setup | completed | `reports/actor-foundation-prod-ready-scoped-rc-100/P01/summary.md`, `summary.json`, `command-status.tsv` |
| `ACTOR-RC100-P02` clean local actor foundation gate rerun | completed | `reports/actor-foundation-prod-ready-scoped-rc-100/P02/summary.md`, `summary.json`, `command-status.tsv` |
| `ACTOR-RC100-P03` final artifact hash and stale-evidence bundle | completed | `reports/actor-foundation-prod-ready-scoped-rc-100/P03/summary.md`, `summary.json`, `command-status.tsv`, root `artifact-hashes.json` |
| `ACTOR-RC100-P04` remote CI actor foundation proof | completed / blocked | `reports/actor-foundation-prod-ready-scoped-rc-100/P04/summary.md`, `summary.json`, `command-status.tsv`, `remote-ci/actor-runtime-foundation-run.json` |
| `ACTOR-RC100-P05` release package workflow proof or approved dry-run | completed / blocked | `reports/actor-foundation-prod-ready-scoped-rc-100/P05/summary.md`, `summary.json`, `command-status.tsv`, `package-workflow/release-packages-run.json` |
| `ACTOR-RC100-P06` CI/workflow hardening against bypass and stale uploads | completed | `reports/actor-foundation-prod-ready-scoped-rc-100/P06/summary.md`, `summary.json`, `command-status.tsv`, `workflow-hardening.log` |
| `ACTOR-RC100-P07` final audit, docs, manifest, and handoff update | completed | `reports/actor-foundation-prod-ready-scoped-rc-100/P07/summary.md`, `summary.json`, `post-commit-command-status.tsv`, `final-handoff.md`, `final-artifact-sha256.txt` |
| `ACTOR-RC100-P08` final acceptance gate and verdict lock | completed / blocked | `reports/actor-foundation-prod-ready-scoped-rc-100/P08/summary.md`, `summary.json`, root `final-verdict.json`, `final-handoff.md`, `final-artifact-hashes.json`, `final-artifact-sha256.txt` |
| `ACTOR-RC100-P09` broad validation unblock and remote CI bundle | completed / remote blocked | `reports/actor-foundation-prod-ready-scoped-rc-100/P09-broad-fix/broad-rerun2-status.tsv`, `remote-ci-instruction-bundle.md`, `remote-ci-blocker.json`, `remote-ci-preflight-status.tsv` |
| `ACTOR-RC100-P10` remote blocker recheck | completed / remote blocked | `reports/actor-foundation-prod-ready-scoped-rc-100/P10-remote-blocker-recheck/summary.md`, `summary.json`, `remote-blocker-recheck-status.tsv` |
| `ACTOR-RC100-P11` third remote blocker recheck | completed / blocked threshold reached | `reports/actor-foundation-prod-ready-scoped-rc-100/P11-remote-blocked-final/summary.md`, `summary.json`, `remote-blocked-final-status.tsv` |
| `ACTOR-RC100-P12` auth-restored candidate integration | completed | `reports/actor-foundation-prod-ready-scoped-rc-100/P12-auth-restored/auth-restored-status.tsv`, candidate patch artifacts |
| `ACTOR-RC100-P13` broad green candidate and remote CI bundle | completed / remote execution pending approval | `reports/actor-foundation-prod-ready-scoped-rc-100/P13-broad-green/broad-status.tsv`, `broad-compiler-cli-tools.log`, `candidate-selected-tracked.patch`, `candidate-newfiles.patch`, `remote-ci-instruction-bundle.md` |
| `ACTOR-RC100-P16` approval blocked final | completed / blocked threshold reached | `reports/actor-foundation-prod-ready-scoped-rc-100/P16-approval-blocked-final/status.tsv`, `summary.md` |
| `ACTOR-RC100-P17` approved commit, push, remote CI, and package proof | completed / blocked | `reports/actor-foundation-prod-ready-scoped-rc-100/P17-remote-execution-billing-blocked/summary.md`, `status.tsv`, `ci-run.json`, `ci-jobs.json`, `ci-check-annotations.jsonl`, `ci-artifacts.json` |
| `ACTOR-RC100-P18` local `act` emulation and mixed remote CI audit | completed / partial | `reports/actor-foundation-prod-ready-scoped-rc-100/P18-act-local/summary.md`, `act-clean-2482fc7-summary.tsv`, `remote-ci-8b94c80-validation-status.tsv`; local act and one remote actor job are green but on different SHAs, so they do not prove final RC100 |
| `ACTOR-RC100-P19` same-SHA remote CI and package dry-run dispatch | partial / queued blocked | `reports/actor-foundation-prod-ready-scoped-rc-100/P19-same-sha-remote-package-proof/summary.md`, `summary.json`, `actions-permissions.json`, `run-ids.tsv`; runs `27459960470` and `27459960966` target `e80d68f0` but repository Actions permissions report `enabled=false` and jobs remain unmaterialized |
| `ACTOR-RC100-P20` Actions disabled recheck | partial / queued blocked | `reports/actor-foundation-prod-ready-scoped-rc-100/P20-actions-disabled-recheck/summary.md`, `summary.json`, `actions-permissions.json`; same-SHA runs remain queued with `jobs=[]` and no artifacts because repository Actions permissions still report `enabled=false` |
| `ACTOR-RC100-P21` Actions disabled blocked final | blocked threshold reached | `reports/actor-foundation-prod-ready-scoped-rc-100/P21-actions-disabled-blocked-final/summary.md`, `summary.json`, `actions-permissions.json`; third consecutive recheck saw repository Actions permissions `enabled=false`, same-SHA runs queued with `jobs=[]`, and no artifacts |
| `ACTOR-RC100-P22` local-only final after user waiver | completed / local-only OK | `reports/actor-foundation-prod-ready-scoped-rc-100/P22-local-only-final/summary.md`, `summary.json`, `broad-final-clean-head-status.tsv`, `actor-foundation-gate-rerun-status.tsv`, `validate-artifact-hashes-status.tsv`, `validate-actor-runtime-foundation-status.tsv`; GitHub Actions proof waived by user |
| `ACTOR-RC100-FULL-*` optional full actor runtime production track | not started / follow-on | Not required for scoped RC100 target |

## Current Iteration

1. Active packet: P22 local-only final after user waiver is complete.
2. Local broad validation is green in the clean candidate worktree:
   `go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1`
   passed on clean local SHA
   `9c4e480a4b2d288c762fecd361bbcec3c5a97a21`; see
   `P22-local-only-final/broad-final-clean-head-status.tsv`.
3. Local actor runtime foundation gate and copied artifact validators passed;
   see `P22-local-only-final/actor-foundation-gate-rerun-status.tsv`,
   `validate-artifact-hashes-status.tsv`, and
   `validate-actor-runtime-foundation-status.tsv`.
4. P22 added local commit
   `9c4e480 Tighten headless Wayland temp cleanup`; candidate branch is clean
   and ahead of `origin/actor-rc100-p12-expanded-clean` by one commit.
5. User explicitly waived GitHub Actions remote CI and release-package workflow
   proof for this closeout. Final local-only status is
   `LOCAL_ONLY_OK_GITHUB_ACTIONS_WAIVED`.
6. Target `ACTOR_FOUNDATION_PROD_READY_SCOPED_RC_100_PERC` remains unclaimed:
   no same-SHA remote CI/package artifacts were downloaded or validated.

## Open Decisions

- Repository-level GitHub Actions are disabled for `BoSuY0/Tetra_Language`;
  `gh api repos/BoSuY0/Tetra_Language/actions/permissions` returned
  `{"enabled":false,"sha_pinning_required":false}`.
- The current main worktree is not used as clean RC proof. The clean local-only
  candidate is the worktree branch `actor-rc100-p12-expanded-clean` at
  `9c4e480a4b2d288c762fecd361bbcec3c5a97a21`, ahead of origin by one local
  commit and not pushed.
- P08 broad compiler/CLI/tools validation is fixed in P22 clean candidate
  evidence. Same-SHA remote artifacts remain absent and are waived only for the
  local-only closeout.
- The root `final-verdict.json` remains the historical P08 verdict artifact and
  still records the old broad failure; current local-only final state is
  recorded by P22 evidence.
