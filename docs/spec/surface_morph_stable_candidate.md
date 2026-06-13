# Tetra Surface Morph Stable Candidate

Status: design freeze for a future stable candidate. Current `lib.core.morph`
remains `EXPERIMENTAL` and is not Surface v1 production support.

This document defines the promotion boundary for turning the current Morph
Capsule evidence layer into a stable authoring/style/recipe graph. It does not
promote Morph today. The machine-readable contract lives in
`docs/spec/surface_morph_stable_candidate_contract.json` and is validated by:

```sh
go run ./tools/cmd/validate-surface-morph-stable-candidate \
  --contract docs/spec/surface_morph_stable_candidate_contract.json
```

## Observed Current State

- `lib/core/morph.tetra` defines capsule tokens, materials, affordances, state
  lenses, motion presets, and recipes over `lib.core.block`.
- `examples/surface_morph_command_palette.tetra` is the current Morph evidence
  source.
- `scripts/release/surface/morph-gate.sh` writes deterministic headless
  `tetra.surface.morph.v1` evidence plus `tetra.surface.morph.gate.v1`.
- `tools/cmd/validate-surface-morph-report` and
  `tools/validators/surface` validate the experimental Morph report envelope.
- Current target evidence is deterministic headless Morph evidence. Stable
  promotion needs target evidence for the scoped production Surface targets.

## Promotion Tiers

- Current tier: `EXPERIMENTAL`.
- Target tier: `PROD_STABLE_SCOPED`.
- Scope: `surface-v1-linux-web`.
- Stable promotion validator status: disabled until P20+ evidence exists.

Morph may be described as stable only after the stable-candidate contract, target
evidence, visual regression evidence, claim scanner, release reports, and final
product gate all pass on the same commit.

## Stable Schema Set

The stable candidate must freeze these schema surfaces:

- `token_graph`
- `material`
- `affordance`
- `recipe`
- `variant`
- `state_lens`
- `motion_preset`
- `accessibility_projection`

Every schema must be versioned and additive-only after promotion. Breaking
changes require a new schema version, migration notes, and compatibility tests.
The `token_graph` schema includes P07 source-of-truth, explicit import,
no-global-cascade, fixed override order, density/DPI mapping, and diagnostics
fields before it can be promoted.

## Recipe Contract

Stable Morph recipes output `Block` only. `Button`, `Card`, `TextField`,
`TextBox`, `Sidebar`, and `Modal` remain forbidden as core Surface primitives.
Recipes must expand to Block graph evidence and must not allocate hidden app
state, use platform widgets, or promote compatibility helpers into primitives.

## Required Target Evidence

Stable Morph promotion requires same-commit evidence for:

- `headless`
- `linux-x64-real-window`
- `wasm32-web-browser-canvas`

Windows and macOS remain outside the stable production claim until separate
target-host evidence exists and the platform packets promote them.

## Nonclaims

This design freeze does not claim production Morph today, React runtime,
Electron runtime, CSS cascade runtime, platform-native widgets, GPU rendering,
all-platform desktop parity, or screen-reader production support.

## Machine-Checkable Failure Cases

The stable candidate validator must reject:

- production stable Morph without complete target evidence;
- recipes that output `Button`, `Card`, `TextField`, `TextBox`, `Sidebar`, or
  `Modal`;
- missing stable schema contracts, including `variant`;
- stable promotion validator enabled before P20+ evidence exists;
- missing nonclaims or promotion gates.
