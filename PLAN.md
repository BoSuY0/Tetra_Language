# PLAN

## Goal

Execute `docs/plans/2026-06-16-surface-morph-rendered-beauty-implementation-plan.md`
end to end so Morph becomes the actual Tetra Surface rendered beauty layer.

## Current Strategy

1. Treat the plan file as the canonical execution source.
2. Work in evidence-backed phases from `MRB-00` through `MRB-13`.
3. Use read-only subagents only as `explorer` / `explorer_fast`.
4. Use delegated edit subagents only as `worker` with `fork_context=true`.
5. Do not claim product beauty until real Morph-authored pixels pass true
   pixel golden evidence and release/product gates.

## Phases

- [x] `MRB-00` Freeze baseline truth and record current gaps.
- [x] `MRB-01` Define Morph rendered beauty contract.
- [x] `MRB-02` Preserve a real Block scene snapshot.
- [x] `MRB-03` Implement render command stream v1.
- [x] `MRB-04` Replace placeholder text/icon evidence for beauty path.
- [x] `MRB-05` Build true pixel golden gate.
- [x] `MRB-06` Remove precomputed frames from product visual evidence.
- [x] `MRB-07` Add Morph rendered beauty reports.
- [x] `MRB-08` Migrate flagship Surface to Morph-authored rendering.
- [x] `MRB-09` Update developer loop and inspector.
- [x] `MRB-10` Update templates and reference apps.
- [x] `MRB-11` Harden claims and documentation.
- [x] `MRB-12` Add integrated Morph rendered beauty gate.
- [x] `MRB-13` Stable promotion audit.

## Active Task

No active implementation task after the current promotion-mode verification
slice. Final verdict remains `PARTIAL`: the target matrix validates, all
supported MRB targets now have renderer-owned byte-for-byte proof, and explicit
`--product-claim --final-signoff` promotion plumbing exists, but the current
worktree is dirty and the default verified reports still have
`product_claim=false` and `final_signoff=false`.

## Open Decisions

- `MRB-08` resolved the flagship migration path by adding
  `examples/surface_morph_rendered_studio_shell.tetra` as the clean
  Morph-authored flagship source and leaving the older manual migration file
  as historical/manual evidence.
- `MRB-09` resolved developer/inspector explainability by attaching validated
  `tetra.surface.morph-rendered-beauty.v1` reports as `morph_to_pixels`
  summaries in `tetra surface dev` and `tools/cmd/surface-inspector`.
  The dev/inspector smoke scripts now generate real flagship runtime, visual,
  MRB, frame, golden, and diff evidence before writing their reports.
- `MRB-10` resolved template/reference app onboarding evidence by requiring
  `morph_to_pixels` chains in template smoke reports, requiring each product
  reference app to provide a Morph rendered beauty chain, and marking only the
  migration compatibility app as `infrastructure_only`. Evidence reports:
  `reports/surface/mrb10-template-smoke/surface-template-smoke.json` and
  `reports/surface/mrb10-reference-apps-smoke/surface-reference-apps.json`.
- `MRB-11` resolved claim/documentation discipline by requiring valid
  same-commit `tetra.surface.morph-rendered-beauty.v1` evidence for
  Surface/Morph beauty, quality, and pixel-perfect claims, requiring product
  MRB signoff for production-ready Morph wording, adding
  `surface-docs-claims-gate.sh`, and narrowing docs wording around intended
  Morph beauty evidence. Evidence: focused and full claim validator tests,
  docs claims gate with and without MRB-10 report dirs, script wiring tests,
  `git diff --check`, and `graphify update .`.
- `MRB-12` resolved integrated release gating by adding
  `scripts/release/surface/morph-rendered-beauty-gate.sh`, wiring it into
  `scripts/release/surface/surface-product-slice-gate.sh`, making the package
  story use `examples/surface_morph_rendered_studio_shell.tetra` with expected
  exit `0`, and validating that product-slice summaries consume MRB evidence
  while keeping `product_claim=false` and `final_signoff=false`.
  Evidence:
  `reports/surface/mrb12-morph-rendered-beauty-gate-verify-20260616195220/morph-rendered-beauty-gate-summary.json`,
  `reports/surface/mrb12-product-slice-gate-verify-20260616195413/surface-product-slice-summary.json`,
  `reports/surface/mrb12-morph-source-wasm-20260616194952/wasm32-web-browser-canvas-block-system.json`,
  broad touched-package tests, docs verification, `git diff --check`, and
  standalone/product-slice gate reruns.
- `MRB-13` resolved final signoff discipline by running a stable-promotion
  audit and denying promotion. Evidence:
  `reports/stabilization/surface_morph_rendered_beauty_mrb_13_stable_promotion_audit.md`,
  `reports/surface/mrb13-morph-rendered-beauty-gate-audit-20260616200009/morph-rendered-beauty-gate-summary.json`,
  and
  `reports/surface/mrb13-product-slice-gate-audit-20260616200041/surface-product-slice-summary.json`.
  Morph remains `EXPERIMENTAL`; target blockers, dirty worktree, missing
  machine-visible `git_commit` fields, `product_claim=false`,
  `final_signoff=false`, and the MRB-12 frame-render bridge prevent stable
  promotion.
- Post-MRB-13 same-commit identity follow-up resolved the missing
  machine-visible `git_commit` blocker for new MRB/product-slice reports.
  `git_commit` is now required to match `git_head` in MRB reports,
  Morph-to-pixels chains, MRB gate summaries, product-slice summaries, and
  same-commit claim evidence. Fresh evidence:
  `reports/surface/mrb-git-identity-morph-rendered-beauty-gate-20260616171311/morph-rendered-beauty-gate-summary.json`
  and
  `reports/surface/mrb-git-identity-product-slice-gate-20260616171359/surface-product-slice-summary.json`.
  Remaining promotion blockers after that follow-up were dirty worktree,
  blocked `linux-x64-real-window` and `wasm32-web-browser-canvas` Morph
  rendered beauty modes, `product_claim=false`, `final_signoff=false`, and the
  MRB-12 frame-render bridge not yet being renderer-owned stable proof.
- Post-MRB-13 wasm browser-canvas target follow-up resolved the
  `wasm32-web-browser-canvas` Morph rendered beauty target blocker for newly
  generated MRB/product-slice evidence. Evidence:
  `reports/surface/mrb-wasm-browser-canvas-morph-gate-final-20260616-verify/morph-rendered-beauty-gate-summary.json`
  and
  `reports/surface/mrb-wasm-browser-canvas-product-slice-final-20260616-verify/surface-product-slice-summary.json`.
  At that point the remaining promotion blockers were dirty worktree,
  blocked `linux-x64-real-window`, `product_claim=false`,
  `final_signoff=false`, and renderer-owned stable proof.
- Post-MRB-13 linux real-window target follow-up resolved the
  `linux-x64-real-window` Morph rendered beauty target blocker for newly
  generated MRB/product-slice evidence. Runtime mode
  `linux-x64-real-window-morph` emits app-produced, source-linked
  `product_visual` RGBA frames from the Morph flagship through
  `wayland-shm-rgba`; the MRB gate now reports `status=validated`,
  `target_blockers=[]`, and validated `headless`, `linux-x64-real-window`, and
  `wasm32-web-browser-canvas` target matrix entries. Evidence:
  `reports/surface/mrb-linux-real-window-morph-gate-final-20260616-verify/morph-rendered-beauty-gate-summary.json`
  and
  `reports/surface/mrb-linux-real-window-product-slice-final-20260616-verify/surface-product-slice-summary.json`.
  Current remaining promotion blockers: dirty worktree,
  `product_claim=false`, `final_signoff=false`, and renderer-owned stable
  proof.
- Post-MRB-13 renderer-owned stable proof guard made that final blocker
  machine-visible instead of narrative-only. `tetra.surface.morph-rendered-beauty.v1`
  reports now include `renderer_stable_proof`; generated reports explicitly
  mark current pixels as `pixel_owner=morph-evidence-bridge`,
  `renderer_owned=false`, and `stable_promotion_eligible=false`. Product/final
  claims require renderer-owned stable proof, MRB gate summaries expose
  `stable_promotion_blockers`, and the stable-candidate contract now requires a
  `renderer-owned stable proof` promotion gate. Fresh evidence:
  `reports/surface/mrb-renderer-proof-guard-gate-20260616-verify/morph-rendered-beauty-gate-summary.json`
  and
  `reports/surface/mrb-renderer-proof-guard-product-slice-20260616-verify/surface-product-slice-summary.json`.
  Current remaining promotion blockers: dirty worktree,
  `product_claim=false`, `final_signoff=false`, and actual renderer-owned
  stable proof implementation/signoff.
- Post-MRB-13 headless renderer-owned stable proof follow-up replaced the
  headless bridge proof with actual command-stream-derived renderer output.
  `tools/internal/surfacerender.RenderCommandStreamRGBA` produces deterministic
  RGBA bytes from `RenderCommandStreamReport`; headless Morph frame order 1 is
  now written from that renderer and `buildMorphRenderedBeautyReport` rerenders
  before setting `pixel_owner=surface-renderer`. Fresh evidence:
  `reports/surface/mrb-renderer-owned-headless-gate-final-20260616-verify/morph-rendered-beauty-gate-summary.json`
  and
  `reports/surface/mrb-renderer-owned-headless-product-slice-final-20260616-verify/surface-product-slice-summary.json`.
  This historical state was superseded by the all-supported-target follow-up
  below. Current remaining promotion blockers are dirty worktree audit,
  `product_claim=false`, and `final_signoff=false`.
- Post-MRB-13 all-supported-target renderer-owned stable proof follow-up added
  source-linked flagship Morph render commands that reproduce the target frame
  bytes and removed the headless-only proof restriction. `buildMorphRenderedBeautyReport`
  now grants renderer-owned proof for any supported target only after
  `RenderCommandStreamRGBA` byte-for-byte matches the pixel-golden frame
  checksum. Fresh evidence:
  `reports/surface/mrb-target-renderer-owned-gate-final-20260616-verify/` and
  `reports/surface/mrb-target-renderer-owned-product-slice-final-20260616-verify/`.
  Current remaining promotion blockers: dirty worktree audit,
  `product_claim=false`, and `final_signoff=false`.
- Post-MRB-13 promotion-mode signoff follow-up converted product/final signoff
  from a hardcoded false-only state into an explicit guarded promotion path.
  `surface-runtime-smoke` can set MRB report `product_claim=true` and
  `final_signoff=true` only with explicit flags and only for clean
  renderer-owned stable proof. The integrated MRB and product-slice gates now
  accept `--product-claim --final-signoff`, reject that mode on a dirty
  checkout before heavy evidence generation, and default to the existing
  validated nonclaim state. Fresh evidence:
  `reports/stabilization/surface_morph_rendered_beauty_promotion_mode_audit.md`,
  `reports/surface/mrb-promotion-aware-gate-default-20260616-verify/`, and
  `reports/surface/mrb-promotion-aware-product-default-20260616-verify/`.
  Current remaining promotion blocker: run promotion mode from a clean checkout
  or clean isolated worktree and produce `git_dirty=false`,
  `product_claim=true`, and `final_signoff=true`.
- `MRB-02` resolved rich scene evidence as tool-side
  `BlockSceneSnapshotReport`, emitted by `surface-runtime-smoke`, with
  `lib/core/block.tetra` exporting the existing Block tree/text/state/a11y
  helper API needed by Morph examples.
- `MRB-03` resolved command-stream evidence as tool-side
  `RenderCommandStreamReport`, emitted from the resolved Block scene snapshot
  via `tools/internal/surfacerender`, with source node IDs, Morph recipe IDs,
  Block scene hash, frame checksum, and validator rejection for handcrafted or
  non-source-linked streams.
- `MRB-04` resolved text/icon placeholder evidence with deterministic
  non-marker raster proofs in render commands, validators, MRB reports, and
  runtime smoke frame expectations. Text raster claims remain intentionally
  narrow: built-in 5x7 alpha mask evidence, not full typography parity.
- `MRB-05` resolved true visual-regression evidence at the visual-diff layer:
  frame checksums now come from real `.rgba`/`.png` artifact bytes, visual
  reports carry separate current/golden artifact paths and SHA256 values,
  validators reject self-golden/metadata-only/fixture-only/missing artifact
  evidence, runtime Block-system frames can emit `artifact_path`, and release
  visual gate rejects `--write-golden`.
- `MRB-06` resolved frame provenance at the runtime/product evidence boundary:
  precomputed frames may pass only as `host_probe_only` infrastructure evidence,
  while `product_visual` frames must be app-produced and linked to app source,
  Morph recipe hash, Block scene hash, and render command stream hash. MRB
  pixel evidence now rejects synthetic/fixture frame artifacts and mismatched
  source/hash links.
- `MRB-07` resolved the first-class Morph rendered beauty report boundary:
  `tetra.surface.morph-rendered-beauty.v1` is now a shared validator/report
  type under `tools/validators/surface`, `surface-runtime-smoke` can emit it
  from runtime plus visual-diff evidence, and the report requires scenario,
  token coverage, recipe coverage, scene, command stream, pixel/golden, and
  negative-guard evidence. Release/product gate consumption remains the
  planned `MRB-12` integration task.
- If a write-enabled subagent cannot be spawned as `worker` with
  `fork_context=true`, ask the user before substituting any delegated editing
  strategy.

## Evidence Root

Use task-specific report directories under `reports/` or
`reports/stabilization/` when implementation begins. Do not create product
claim artifacts from synthetic-only evidence.
