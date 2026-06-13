# Tetra Actor Foundation RC100 Control

## Active Objective

`$todium-superpower Реалізуй повністю весь цей план - /home/tetra/Downloads/2026-06-10-actor-foundation-prod-ready-scoped-rc-100-implementation-plan.md`

## Active Packet

`ACTOR-RC100-P18`: local `act` emulation of the GitHub Actions
`actor-runtime-foundation-linux` job. `ACTOR-RC100-P17` completed with blocker
`REMOTE_CI_BILLING_LOCKED`; candidate commit
`3480870c7ff52d211aaa63c16238e62d6165cfbd` is pushed to
`origin/actor-rc100-p12-expanded-clean`.

## Next Actions

1. Run local `act` emulation of `ci.yml` job
   `actor-runtime-foundation-linux` on a fresh worktree for commit
   `3480870c7ff52d211aaa63c16238e62d6165cfbd`, with artifact server output
   captured under the P18 evidence directory.
2. Record local report/artifact evidence and classify it as local/dry-run, not
   remote proof.
3. Resolve the GitHub account billing lock that prevents Actions jobs from
   starting.
4. Rerun `ci` workflow job `actor-runtime-foundation-linux` for commit
   `3480870c7ff52d211aaa63c16238e62d6165cfbd`, download the
   `tetra-actor-runtime-foundation-${SHA}-linux-x64` artifact, and validate it.
5. Rerun the release package workflow with publish-side effects disabled where
   supported, download the package artifact, and validate expected actor
   runtime package paths.
6. Rerun final acceptance before claiming
   `ACTOR_FOUNDATION_PROD_READY_SCOPED_RC_100_PERC`.
7. Do not claim RC100 until same-change remote CI/package proof exists.

## Stop Conditions

- Clean checkout cannot be created without destructive cleanup or overwriting
  user-owned changes.
- GitHub Actions account remains locked due to a billing issue.
- Remote CI/package artifacts are missing, fail validation, or correspond to a
  different SHA.
- A stronger claim would require full actor runtime production features outside
  scoped RC100.

## Cache Discipline

- Use:
  `GOTELEMETRY=off`
  `GOCACHE=$(pwd)/.cache/go-build-actor-rc100`
  `GOTMPDIR=$(pwd)/.cache/go-tmp-actor-rc100`
- Never set `GOCACHE` to `/tmp`.
- After evidence runs, clean with the concrete `GOCACHE` path when appropriate.
