# Tetra Actor Runtime Production Foundation Notes

## Scope And Nonclaims

- Current mission is the actor runtime plan:
  `/home/tetra/Downloads/actor-runtime-production-foundation-codex-plan.md`.
- Evidence root is `reports/actor-runtime-foundation/`.
- Target claim is `Actor Runtime Production Foundation v1:
  PROD_STABLE_SCOPED` for Linux-x64 actor/task runtime foundation only.
- P18 platform statement: Linux x64 local actor foundation evidence is OK;
  other platforms were not tested and remain nonclaims.
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
- P18 local `act` evidence exists under
  `reports/actor-foundation-prod-ready-scoped-rc-100/P18-act-local/`.
  Candidate commit `2482fc72805730b665f18c1be398aad7fcdb839b` passed the
  unmodified `ci.yml` job `actor-runtime-foundation-linux` from a fresh clone,
  and the unzipped local artifact passes `validate-artifact-hashes` plus
  `validate-actor-runtime-foundation`.
- Remote GitHub Actions run `27287513207` was dispatched for the same P18
  candidate commit. It failed before job steps; annotations say the jobs were
  not started because the account is locked due to a billing issue.
- P18 fixed two local CI blockers before the green `act` proof: the RAM
  readiness audit no longer pins an impossible stale SHA, and
  `.github/workflows/ci.yml` uploads
  `parallel-production-linux-x64/parallelrt-evidence.raw.json` in the actor
  foundation artifact.

## Active Bridge

- `ACTOR-P17` is closed with `PROD_STABLE_SCOPED` local evidence, and
  `ACTOR-RC100-P18` is closed for local `act` emulation. The RC100 target is
  still not claimed because real GitHub-hosted Actions are account
  billing-blocked for run `27287513207` and release-package proof has not been
  recorded. Linux x64 is the only P18 platform with OK local evidence.
