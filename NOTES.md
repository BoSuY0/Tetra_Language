# Tetra Actor Runtime Production Foundation Notes

## Scope And Nonclaims

- Current mission is the actor runtime plan:
  `/home/tetra/Downloads/actor-runtime-production-foundation-codex-plan.md`.
- Evidence root is `reports/actor-runtime-foundation/`.
- Target claim is `Actor Runtime Production Foundation v1:
  PROD_STABLE_SCOPED` for Linux-x64 actor/task runtime foundation only.
- Do not claim full Erlang/OTP supervision, cluster membership,
  reconnect/retry/order production deployment, non-Linux distributed actor
  support, distributed pointer/region zero-copy, official benchmarks,
  performance superiority, full formal race proof, or broad Tetra production
  readiness.

## Durable Discoveries

- `GOAL.md` had drifted to a Surface Block goal while the active thread goal is
  actor-runtime. It was repaired on 2026-06-10; keep actor-runtime as canonical.
- `reports/actor-runtime-foundation/P01/` closes the scheduler packet by
  preserving the prototype boundary as `PROTOTYPE_ONLY_NON_GOAL`.
- `reports/actor-runtime-foundation/P04/` closes typed mailbox ownership and
  actor/island transfer proof with current targeted tests.
- `tools/validators/actorprod` and
  `scripts/release/post_v0_4/actor-runtime-foundation-linux-x64-gate.sh` are
  the authoritative final validator/gate pair for P12/P17.
- P17 final evidence exists under `reports/actor-runtime-foundation/final/` and
  `reports/actor-runtime-foundation/P17/`; final audit lives at
  `docs/audits/actor-runtime-production-foundation-final.md`.

## Active Bridge

- `ACTOR-P17` is closed with `PROD_STABLE_SCOPED` local evidence. The final
  audit records dirty worktree state, so release-candidate, clean-checkout,
  remote CI, package publication, and `PROD_READY_PROVEN` remain nonclaims.
