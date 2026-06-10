# Tetra Actor Runtime Production Foundation Control

## Active Objective

`$goal-forge $goal-loop $define-goal Реалізуй повністю весь цей план - /home/tetra/Downloads/actor-runtime-production-foundation-codex-plan.md`

## Active Packet

None. `ACTOR-P17` final same-commit actor production foundation gate is closed
for local scoped evidence, and `ACTOR-RC100-P18` local `act` proof is closed for
candidate commit `2482fc72805730b665f18c1be398aad7fcdb839b`.

## Next Actions

1. Resolve the GitHub Actions account billing lock.
2. Rerun real GitHub-hosted `ci.yml` on branch
   `actor-rc100-p12-expanded-clean` after the lock is cleared, confirm the run
   head SHA, download the actor foundation artifact, and rerun artifact
   validators locally. Latest blocked run: `27287513207` for commit
   `2482fc72805730b665f18c1be398aad7fcdb839b`.
3. Run and record release-package proof for the same candidate before any
   `ACTOR_FOUNDATION_PROD_READY_SCOPED_RC_100_PERC`, release-candidate,
   clean-checkout proof, package publication, or `PROD_READY_PROVEN` claim.
4. Keep Linux x64 as the only P18 platform with OK local evidence until other
   platforms are actually tested.
5. Keep final audit and tracker files aligned if new evidence is added.

## Stop Conditions

- The same P17 gate or broad-test failure repeats twice without new evidence.
- A required final claim would need remote CI, package publication, non-Linux
  target-host runtime smoke, cluster deployment, distributed zero-copy, official
  benchmark evidence, or full formal race proof that was not actually run.
- The worktree contains unrelated dirty changes that make final status
  ambiguous; record dirty state honestly instead of claiming release-candidate.

## Cache Discipline

- Use persistent caches such as `.cache/go-build-actor-p17`,
  `.cache/go-tmp-actor-p17`, or `${XDG_CACHE_HOME:-$HOME/.cache}/tetra-language/go-build-actor-p17`.
- Never set `GOCACHE` to `/tmp`.
- Clean with the concrete `GOCACHE` path after evidence runs when appropriate.
