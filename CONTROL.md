# Tetra Actor Foundation RC100 Control

## Active Objective

`$todium-superpower Реалізуй повністю весь цей план - /home/tetra/Downloads/2026-06-10-actor-foundation-prod-ready-scoped-rc-100-implementation-plan.md`

## Active Packet

`ACTOR-RC100-P22`: local-only final after user waiver.

User changed closeout scope on 2026-06-13: mark the work OK without GitHub
Actions, using local tests only.

Current local-only candidate:
`/home/tetra/.codex/worktrees/Tetra_Language/actor-rc100-p12-expanded-clean`
at `9c4e480a4b2d288c762fecd361bbcec3c5a97a21`.

Candidate branch is clean locally and ahead of
`origin/actor-rc100-p12-expanded-clean` by one commit. It was not pushed.

P22 local evidence passed:

- broad compiler/CLI/tools gate;
- actor runtime foundation local gate;
- copied artifact hash validation;
- copied actor foundation report validation.

## Next Actions

1. Mark `/goal` complete as local-only if final hygiene checks pass.
2. Do not claim `ACTOR_FOUNDATION_PROD_READY_SCOPED_RC_100_PERC`.
3. If remote proof is later required, push local commit `9c4e480`, enable
   repository-level GitHub Actions, rerun same-SHA CI/package workflows,
   download artifacts, and validate them.

## Stop Conditions

- Clean checkout cannot be created without destructive cleanup or overwriting
  user-owned changes.
- Remote CI/package artifacts are not part of the P22 local-only closeout.
- A stronger claim would require full actor runtime production features outside
  scoped RC100.

## Cache Discipline

- Use:
  `GOTELEMETRY=off`
  `GOCACHE=$(pwd)/.cache/go-build-actor-rc100`
  `GOTMPDIR=$(pwd)/.cache/go-tmp-actor-rc100`
- Never set `GOCACHE` to `/tmp`.
- After evidence runs, clean with the concrete `GOCACHE` path when appropriate.
