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
| `ACTOR-RC100-P18` local `act` emulation of actor runtime foundation job | in progress | User installed `act`; running local emulation on fresh worktree for commit `3480870c7ff52d211aaa63c16238e62d6165cfbd` |
| `ACTOR-RC100-FULL-*` optional full actor runtime production track | not started / follow-on | Not required for scoped RC100 target |

## Current Iteration

1. Active packet: P18 local `act` emulation is in progress.
2. Local broad validation is green in the clean candidate worktree:
   `go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1`
   passed; see `P13-broad-green/broad-status.tsv` and
   `P13-broad-green/broad-compiler-cli-tools.log`.
3. User approval on 2026-06-10 cleared the P16 approval blocker for
   commit/push/remote evidence.
4. `gh repo view` confirms access to `BoSuY0/Tetra_Language` as `ADMIN`, and
   both `ci.yml` and `release-packages.yml` are active.
5. Target `ACTOR_FOUNDATION_PROD_READY_SCOPED_RC_100_PERC` remains unclaimed
   because GitHub Actions did not start jobs and produced no remote CI/package
   artifacts. A local `act` pass can strengthen local/dry-run evidence only.
6. Remote CI instruction bundle is prepared at
   `P13-broad-green/remote-ci-instruction-bundle.md`.

## Open Decisions

- GitHub account billing lock must be resolved before same-change remote CI or
  package workflow proof can be produced.
- The current dirty main worktree cannot itself prove clean RC status.
- P08 broad compiler/CLI/tools validation is fixed in P13 clean candidate
  evidence, but the broad-green state is not yet a pushed same-change commit.
  `gh auth status` is valid; see
  `reports/actor-foundation-prod-ready-scoped-rc-100/P13-broad-green/remote-ci-instruction-bundle.md`.
- The root `final-verdict.json` remains the historical P08 verdict artifact and
  still records the old broad failure; current broad-green remote-blocked state
  is recorded by P09 through P11 evidence.
