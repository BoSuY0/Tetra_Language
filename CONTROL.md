# Tetra Actor Runtime Production Foundation Control

## Active Objective

`$goal-forge $goal-loop $define-goal Реалізуй повністю весь цей план - /home/tetra/Downloads/actor-runtime-production-foundation-codex-plan.md`

## Active Packet

None. `ACTOR-P17` final same-commit actor production foundation gate is closed
for local scoped evidence.

## Next Actions

1. Keep final audit and tracker files aligned if new evidence is added.
2. Do not claim release-candidate, clean-checkout proof, remote CI proof,
   package publication, or `PROD_READY_PROVEN` unless those checks are actually
   run and recorded.
3. If preparing a PR/commit later, preserve unrelated dirty worktree changes and
   rerun the final verification commands after the commit boundary is clear.

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
