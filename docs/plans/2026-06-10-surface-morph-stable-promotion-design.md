# Surface Morph Stable Promotion Design

## Decision

Use a stable-candidate design freeze before any production Morph promotion.

Chosen route: keep `lib.core.morph` experimental, add a machine-readable
stable-candidate contract, and validate that contract with a disabled
promotion validator until P20+ product evidence exists.

## Observed Facts

- `docs/spec/surface_morph.md` states Morph is experimental and not Surface v1
  production support.
- `lib/core/morph.tetra` defines tokens, materials, affordances, state lenses,
  motion presets, and recipes over `lib.core.block`.
- `scripts/release/surface/morph-gate.sh` produces deterministic headless
  evidence only.
- `tools/validators/surface/report.go` already validates rich experimental
  Morph evidence, recipe expansion into Block, negative guards, nonclaims, and
  dirty-checkout production rejection.

## Rejected Alternatives

- Docs-only criteria: easy to write but not enough for the plan's
  machine-checkable acceptance.
- Immediate stable promotion: premature API freeze and unsupported production
  claim because Morph currently has deterministic headless evidence only.

## Design

- Add `docs/spec/surface_morph_stable_candidate.md` for human-readable
  promotion criteria.
- Add `docs/spec/surface_morph_stable_candidate_contract.json` for
  machine-readable criteria.
- Add `tools/cmd/validate-surface-morph-stable-candidate` to reject missing
  stable schema fields, production Morph claims without complete target
  evidence, recipe output beyond `Block`, and premature validator enablement.
- Do not add this validator to production release gates yet. P20+ must provide
  visual/product evidence before promotion gates may consume it as release
  evidence.

## Verification

- Targeted Go tests for the stable-candidate validator.
- Direct validation of the repo contract JSON.
- Existing docs/manifest checks must continue to pass.

## Rollout

P03-P08 can build on the contract. P20+ may enable stable-promotion gate
integration only after target, visual, accessibility, perf, and claim evidence
are present on the same commit.
