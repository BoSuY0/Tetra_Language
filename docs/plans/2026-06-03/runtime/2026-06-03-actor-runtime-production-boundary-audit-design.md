# P18.0 Actor Runtime Production Boundary Audit Design

## Scope

P18.0 is an audit-only slice for the Ideal Master Plan. It separates concrete
actor runtime evidence from scheduler prototype evidence and preserves the
non-claim that Tetra does not yet have a full production actor runtime.

No runtime scheduler behavior, message transport behavior, or production claim
is implemented in this slice.

## Evidence Inputs

- `compiler/internal/actorsrt` owns the built-in x64 actor runtime object and
  its fixed capacities.
- `docs/spec/actors.md` documents the current single-thread cooperative actor
  runtime, typed mailbox limits, distributed Linux-x64 promotion surface, and
  non-goals.
- `compiler/internal/parallelrt` owns the checked per-core scheduler prototype
  model and zero-copy region-transfer benchmark rows.
- `tools/validators/actordist` and `tools/validators/parallelprod` validate
  executable distributed actor and parallel production evidence, while keeping
  platform and scheduler boundaries explicit.

## Audit Shape

Add an internal `compiler/internal/actorsrt` report:

- schema: `tetra.runtime.actor.production_boundary.v1`
- required rows:
  - `current_actor_runtime_limits`
  - `scheduler_prototype_features`
  - `production_runtime_acceptance`
  - `full_claim_blockers`
- required non-claim:
  - full production actor runtime is not claimed

The validator rejects:

- a report with `FullProductionClaimed=true`;
- missing or duplicate master-plan rows;
- rows without evidence and boundary text;
- a scheduler prototype row marked as production-ready;
- a blocker row without machine-readable missing facts;
- a report missing the full-production-runtime non-claim.

## Acceptance For This Slice

- Focused RED/GREEN tests prove the audit API and fake-claim rejection.
- Docs and report artifacts state current limits, prototype features,
  production runtime acceptance, and blockers.
- Feature/docs/manifest metadata points to the new P18.0 audit without
  promoting a full production actor runtime.
- Graphify is updated after code changes.
