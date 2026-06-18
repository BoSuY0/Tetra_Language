# Tetra Surface Morph Stable Candidate

Status: design freeze for a future stable candidate. Current `lib.core.morph` remains `EXPERIMENTAL`
and is not Surface v1 production support.

This document defines the promotion boundary for turning the current Morph Capsule evidence layer
into a stable authoring/style/recipe graph. It does not promote Morph today. The machine-readable
contract lives in `docs/spec/surface/morph/surface_morph_stable_candidate_contract.json` and is
validated by:

```sh
go run ./tools/cmd/validate-surface-morph-stable-candidate \
  --contract docs/spec/surface/morph/surface_morph_stable_candidate_contract.json
```

## Observed Current State

- `lib/core/morph/morph.tetra` defines capsule tokens, materials, affordances, state lenses, motion
  presets, and recipes over `lib.core.block`.
- `examples/surface/morph_core/surface_morph_command_palette.tetra` is the current Morph evidence
  source.
- `scripts/release/surface/morph-gate.sh` writes deterministic headless `tetra.surface.morph.v1`
  evidence plus `tetra.surface.morph.gate.v1`.
- `tools/cmd/validate-surface-morph-report` and `tools/validators/surface` validate the experimental
  Morph report envelope.
- Current target evidence is deterministic headless Morph evidence. Stable promotion needs target
  evidence for the scoped production Surface targets.

## MRB-13 Audit, 2026-06-16

MRB-13 does not promote Morph. It records a stable-promotion denial on the current evidence set.

Evidence reviewed:

- `reports/surface/mrb13-morph-rendered-beauty-gate-audit-20260616200009/morph-rendered-beauty-gate-summary.json`
- `reports/surface/mrb13-product-slice-gate-audit-20260616200041/surface-product-slice-summary.json`
- `reports/stabilization/surface_morph_rendered_beauty_mrb_13_stable_promotion_audit.md`

Findings:

- Morph rendered beauty gate status is `validated_with_target_blockers`, not a full target promotion
  result.
- `headless` is validated, but `linux-x64-real-window` and `wasm32-web-browser-canvas` remain
  `BLOCKED` for Morph rendered beauty product claim in the integrated gate.
- Product-slice summary passes its current gate but keeps `product_claim=false` and
  `final_signoff=false`.
- The worktree is not clean, so this is not a clean-checkout promotion audit.
- MRB-12 uses `morph.render_studio_shell_frame` as an evidence bridge for Morph-authored flagship
  pixels. That helper must not be treated as a stable renderer path or a new core primitive.

Decision:

- Current tier remains `EXPERIMENTAL`.
- Target tier remains a future `PROD_STABLE_SCOPED` candidate only.
- Stable promotion requires complete target evidence, explicit product/final signoff, and
  replacement or strict nonclaim containment of the MRB-12 frame rendering bridge.

## Post-Audit Target Evidence Follow-Ups, 2026-06-16

Later same-day follow-ups removed the explicit target blockers from newly generated Morph rendered
beauty gates:

- `wasm32-web-browser-canvas-morph` now provides app-produced browser-canvas Morph frame evidence.
- `linux-x64-real-window-morph` now provides app-produced real-window Morph frame evidence through
  `wayland-shm-rgba`.
- Fresh evidence:
  `reports/surface/mrb-linux-real-window-morph-gate-final-20260616-verify/morph-rendered-beauty-gate-summary.json`
  reports `status=validated`, validates `headless`, `linux-x64-real-window`, and
  `wasm32-web-browser-canvas`, and has `target_blockers=[]`.

This still does not promote Morph. The current tier remains `EXPERIMENTAL` because the worktree is
dirty, product/final signoff remains false, and the MRB-12 frame rendering bridge is not yet
renderer-owned stable proof.

## Renderer-Owned Stable Proof Boundary

Stable promotion requires a `renderer-owned stable proof` promotion gate. Morph rendered beauty
reports may validate target artifacts, but any bridge-owned pixels from
`morph.render_studio_shell_frame` are not sufficient for stable promotion.

The promotion proof must be renderer-owned, Block-first, derived from the render command stream, and
eligible for stable promotion in the `renderer_stable_proof` section of the Morph rendered beauty
report.

Post-MRB-13 follow-up evidence now satisfies that boundary for `headless`, `linux-x64-real-window`,
and `wasm32-web-browser-canvas` through a command-stream-derived byte-for-byte frame checksum proof.
Morph remains `EXPERIMENTAL` until clean checkout audit, product claim, and final signoff are
intentionally promoted.

## Promotion Tiers

- Current tier: `EXPERIMENTAL`.
- Target tier: `PROD_STABLE_SCOPED`.
- Scope: `surface-v1-linux-web`.
- Stable promotion validator status: disabled until P20+ evidence exists.

Morph may be described as stable only after the stable-candidate contract, target evidence,
renderer-owned stable proof, visual regression evidence, claim scanner, release reports, and final
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

Every schema must be versioned and additive-only after promotion. Breaking changes require a new
schema version, migration notes, and compatibility tests. The `token_graph` schema includes P07
source-of-truth, explicit import, no-global-cascade, fixed override order, density/DPI mapping, and
diagnostics fields before it can be promoted.

## Recipe Contract

Stable Morph recipes output `Block` only. `Button`, `Card`, `TextField`, `TextBox`, `Sidebar`, and
`Modal` remain forbidden as core Surface primitives. Recipes must expand to Block graph evidence and
must not allocate hidden app state, use platform widgets, or promote compatibility helpers into
primitives.

## Required Target Evidence

Stable Morph promotion requires same-commit evidence for:

- `headless`
- `linux-x64-real-window`
- `wasm32-web-browser-canvas`

The same-commit evidence must include renderer-owned stable proof. Target artifacts that are valid
but bridge-owned remain product-slice evidence only and do not promote Morph.

Windows and macOS remain outside the stable production claim until separate target-host evidence
exists and the platform packets promote them.

## Nonclaims

This design freeze does not claim production Morph today, React runtime, Electron runtime, CSS
cascade runtime, platform-native widgets, GPU rendering, all-platform desktop parity, or
screen-reader production support.

## Machine-Checkable Failure Cases

The stable candidate validator must reject:

- production stable Morph without complete target evidence;
- recipes that output `Button`, `Card`, `TextField`, `TextBox`, `Sidebar`, or `Modal`;
- missing stable schema contracts, including `variant`;
- stable promotion validator enabled before P20+ evidence exists;
- missing `renderer-owned stable proof` promotion gate;
- missing nonclaims or promotion gates.
