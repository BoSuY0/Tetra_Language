# NOTES

## Chronological Notes

- 2026-06-16: User approved the direction that Morph should be the beauty
  layer, not a duplicated design system.
- 2026-06-16: Canonical plan created at
  `docs/plans/2026-06-16-surface-morph-rendered-beauty-implementation-plan.md`.
- 2026-06-16: Existing root `GOAL.md`, `PLAN.md`, `ATTEMPTS.md`, `NOTES.md`,
  and `CONTROL.md` contained prior release-validation / Actor RC100 state and
  were superseded for this new active `/goal`.
- 2026-06-16: Required subagent policy from user: read-only subagents only
  `gpt-5.4-mini` / `gpt-5.3-codex-spark`; delegated editing only `gpt-5.5`,
  reasoning `xhigh`, `fork_context=true`.
- 2026-06-16: Read-only `explorer_fast` and `explorer` subagents were used to
  review the goal/memory reset. They did not edit files. The useful findings
  were folded into `GOAL.md`: `MORPH-T00` aliases and `ACCEPTANCE-FULL`.
- 2026-06-16: `MRB-00` baseline recorded at
  `reports/stabilization/surface_morph_rendered_beauty_mrb_00_baseline.md`.
  Current HEAD is `95bfd4a887bab5032437cb22494d034e82ae6d35`; worktree is
  dirty and behind origin. At MRB-00 time, no `surface_morph_rendered_beauty`,
  `morph-rendered-beauty`, `surfacerender`, or
  `validate-surface-morph-rendered-beauty` implementation artifact exists yet
  beyond the plan.
- 2026-06-16: `MRB-00` key gaps: Morph is intended beauty/authoring layer but
  still experimental; flagship visible draw path is immediate-mode `draw.*`;
  Tetra draw text is marker-based; Block-system/runtime evidence has
  self-golden/precomputed-frame paths; product-slice gate remains
  `product_claim=false` and `final_signoff=false`.
- 2026-06-16: `reports/` is ignored by `.gitignore`, so
  `reports/stabilization/surface_morph_rendered_beauty_mrb_00_baseline.md` is
  local evidence, not a tracked diff artifact.
- 2026-06-16: Write-enabled subagent policy could not be satisfied exactly:
  `spawn_agent` rejected `agent_type=worker` with full-history
  `fork_context=true`. No substitute write subagent was used; `MRB-01` edits
  were done by the parent controller with TDD.
- 2026-06-16: `MRB-01` added the experimental
  `tetra.surface.morph-rendered-beauty.contract.v1` contract, report schema
  expectation `tetra.surface.morph-rendered-beauty.v1`, CLI validator, and
  tests rejecting core Button primitives, missing self-golden guard,
  self-golden reports, metadata-only pixel evidence, precomputed product
  frames, and missing DOM guard.
- 2026-06-16: `MRB-02` audit found full visual specs already exist in
  `lib/core/block.tetra`, but `BlockProps` intentionally compresses them for
  ABI. The implemented direction is a tool-side `BlockSceneSnapshotReport`
  side channel for rich renderable scene evidence, not a new core primitive.
- 2026-06-16: `MRB-02` added typed `block_scene_snapshot` evidence with
  coverage for layout, paint, text, image, input, event, state, motion, and
  accessibility. `surface-runtime-smoke` now emits it for Morph/Block-system
  scenarios and MRB rendered-beauty reports now require typed snapshot evidence
  in addition to `block_scene_snapshot_hash`.
- 2026-06-16: `./tetra check examples/surface_morph_studio_shell.tetra`
  initially failed because `lib.core.block` did not export existing
  `block.parts` tree/text/state/a11y helpers. `lib/core/block.tetra` now folds
  those helper groups into the public `lib.core.block` module; this keeps
  `BlockProps` compact and does not add `Button`/`Card`/`TextField` core
  primitives.
- 2026-06-16: `MRB-03` added first-class
  `tetra.surface.render-command-stream.v1` evidence. The stream is generated
  from `BlockSceneSnapshotReport` in `tools/internal/surfacerender`, carried by
  `surface-runtime-smoke`, and validated as source-linked evidence with
  `source_node_id`, Morph `recipe`, `block_scene_hash`, `frame_checksum`,
  deterministic command order, and handcrafted/non-source-linked rejection.
- 2026-06-16: Full affected runtime tests exposed a stale stream frame checksum
  in the wasm browser-canvas Block-system path after frame evidence was merged.
  `buildReport` now rebuilds the render command stream from the final scenario
  frames before emitting `surface.Report`.
- 2026-06-16: `MRB-04` replaced text placeholder evidence with a deterministic
  5x7 glyph mask in `lib/core/draw.tetra` and a matching Go expected-frame
  helper in `tools/cmd/surface-runtime-smoke`. This is a bounded software
  baseline for readable raster evidence, not a full font/shaping claim.
- 2026-06-16: `MRB-04` added typed raster proof fields for text/icon render
  commands and MRB reports. Validators now reject marker-only text/icon raster
  claims and require nonempty raster hashes, dimensions, coverage, and the
  expected built-in raster formats.
- 2026-06-16: During `MRB-04`, `./tetra check` showed the public
  `lib.core.block` facade still needed tree/text/state/a11y helper exports for
  Morph examples after the split `block.parts` work. The facade helpers are
  now present in `lib/core/block.tetra`; `BlockProps` remains compact and no
  new core UI primitive was added.
- 2026-06-16: `MRB-05` moved visual regression proof from string-derived
  checksums to artifact bytes. `surface-visual-diff` now reads current and
  golden `.rgba`/`.png` artifacts, computes SHA256 from file bytes, compares
  pixels, and requires separate artifact paths. `tools/validators/surface`
  rejects self-golden paths, metadata-only checksums, fixture/testdata frame
  evidence, unsupported artifact formats, and missing MRB-05 negative guards.
- 2026-06-16: `surface-runtime-smoke` can attach `artifact_path` to
  Block-system frame evidence by writing the actual RGBA bytes used for
  checksum evidence. This is current-frame evidence only; separate golden
  artifacts remain required by `surface-visual-diff`.
- 2026-06-16: `scripts/release/surface/visual-gate.sh` explicitly rejects
  `--write-golden`; golden updates must stay outside release/product gates.
  Remaining Phase 3 bridge: `MRB-06` must remove/prevent precomputed fixture
  frames from satisfying product visual evidence.
- 2026-06-16: `MRB-06` introduced explicit frame provenance. Runtime
  `FrameReport` now distinguishes `producer`, `evidence_role`, `app_source`,
  `morph_recipe_hash`, `block_scene_hash`, `render_command_stream_hash`, and
  `precomputed`. Validators allow precomputed frames only as
  `host_probe_only` infrastructure evidence; `product_visual` frames must be
  app-produced and source/hash-linked. MRB pixel evidence has matching source
  links and rejects fixture/precomputed/synthetic artifact paths even when the
  explicit `precomputed_fixture_frame` flag is false.
- 2026-06-16: Read-only MRB-06 subagent confirmed the gap was not in
  `block_scene_snapshot` or `render_command_stream`, which already validate
  source/hash linkage; the missing layer was leaf frame/pixel provenance and a
  separate Morph recipe hash in product visual evidence. Editing still stayed
  in the parent controller because the exact write-subagent policy remains
  unavailable.
- 2026-06-16: `MRB-07` moved Morph rendered beauty validation out of the
  CLI-only command and into shared `tools/validators/surface` APIs. The
  `validate-surface-morph-rendered-beauty` command now wraps the shared
  validator, and `tetra.surface.morph-rendered-beauty.v1` reports require
  `scenario_name`, Morph source identity, token coverage, recipe coverage,
  Block scene hash/evidence, render command stream hash/evidence, real
  pixel/golden evidence, target, negative guards, and nonclaims.
- 2026-06-16: `surface-runtime-smoke` can emit a first-class MRB report when
  invoked with both `--visual-report` and `--morph-rendered-beauty-report`.
  The builder links runtime Morph evidence to visual-diff targets, hashes the
  Morph source, carries app-produced frame provenance, and validates the report
  before writing it.
- 2026-06-16: MRB-07 real CLI smoke evidence lives under local
  `.cache/mrb07-e2e/`: `surface-headless-morph.json`,
  `surface-visual-regression.json`, and
  `surface-morph-rendered-beauty.json`. The first attempt failed because every
  rendered frame needs a separate golden artifact; the second failed because
  the golden drift exceeded tolerance. The passing run used separate per-frame
  goldens with a tiny within-tolerance byte drift and validated the final MRB
  report.
- 2026-06-16: MRB-07 intentionally did not wire release/product gates to the
  new report. That remains `MRB-12`; until then the report is first-class and
  runtime-emitted, but product-slice signoff must not claim Morph rendered
  beauty from it automatically.
- 2026-06-16: MRB-08 resolved the flagship path by adding a clean Morph-first
  source, `examples/surface_morph_rendered_studio_shell.tetra`, instead of
  rewriting the older manual `examples/surface_migration_tetra_control_center.tetra`.
  The old file remains useful historical/manual migration evidence; the new
  file is the flagship evidence source for Morph-authored beauty.
- 2026-06-16: MRB-08 runtime evidence is source-aware. `headless-morph` can now
  run `runMorphScenarioForSource(...)`; the rendered studio shell source gets
  its own Block graph/accessibility tree/scene snapshot, `surface-morph-rendered-studio-shell`
  component artifact, expected app exit `0`, visual diff report, and
  `tetra.surface.morph-rendered-beauty.v1` report.
- 2026-06-16: MRB-12 resolved the earlier single-file flagship check risk.
  `go run -buildvcs=false ./cli/cmd/tetra check examples/surface_morph_rendered_studio_shell.tetra`
  now passes after the source stopped importing `lib.core.draw` and calls
  Morph-owned `morph.render_studio_shell_frame` before presenting real
  `surface.Frame` values.
- 2026-06-16: MRB-09 added `MorphToPixelsChainReport` as the compact
  developer/inspector explanation layer over the already validated
  `tetra.surface.morph-rendered-beauty.v1` report. This deliberately avoids a
  second design system or duplicate renderer: the chain summarizes source hash,
  token graph, recipes, Block scene hash, render command stream hash, frame
  artifact, golden artifact, and diff metrics from the MRB report.
- 2026-06-16: `tetra surface dev` remains a fast rebuild loop, not hot reload.
  It can attach Morph-to-pixels evidence with
  `--morph-rendered-beauty-report <path>`, and the MRB-09 smoke script now
  generates that real flagship runtime/visual/MRB evidence before writing the
  dev workflow report.
- 2026-06-16: `tools/cmd/surface-inspector` can now accept
  `--runtime-report morph-rendered-beauty:<path>` and emits sections for
  recipe expansions, Block scene nodes, render commands, frame artifacts, and
  golden diff plus the `morph_to_pixels` hash chain. General non-MRB inspector
  reports remain readable; MRB-specific smoke evidence lives under
  `reports/surface/mrb09-inspector-smoke/`.
- 2026-06-16: MRB-10 taught `surface-runtime-smoke` and Surface validators to
  treat `examples/surface_reference_*.tetra` as normal exit-zero Surface apps
  for Morph evidence, and to accept generated template sources under
  `reports/surface/.../templates/<kind>/src/main.tetra`.
- 2026-06-16: `validate-surface-morph-rendered-beauty` now has
  `--morph-to-pixels-chain-out`, which writes a validated
  `MorphToPixelsChainReport` from a first-class MRB report. Template and
  reference smoke scripts use this instead of parsing MRB JSON in shell.
- 2026-06-16: `surface-template-smoke.sh` now generates MRB runtime, visual
  diff, rendered beauty report, and Morph-to-pixels chain evidence from the
  generated `studio-shell` template source. Evidence lives under
  `reports/surface/mrb10-template-smoke/template-morph-rendered-beauty/`.
- 2026-06-16: `surface-reference-apps-smoke.sh` now generates product
  Morph-to-pixels evidence for nine reference apps. The `migration` reference
  app remains compatibility/infrastructure-only because it intentionally uses
  `lib.core.widgets` migration evidence; it must not be counted as product
  beauty evidence.
- 2026-06-16: `validate-surface-claims` now treats
  `tetra.surface.morph-rendered-beauty.v1` as the evidence boundary for
  Surface/Morph beauty, quality, and pixel-perfect claims. The report must be
  fully valid and match the current `git rev-parse HEAD`; `production-ready
  Morph` wording additionally requires `product_claim` and `final_signoff` in
  that same-commit MRB report.
- 2026-06-16: `surface-docs-claims-gate.sh` is the standalone MRB-11 docs
  gate. Current docs pass without a report-dir because they describe intended
  Morph beauty as an experimental evidence contract/nonclaim; passing with
  MRB-10 report dirs verifies generated evidence does not create false
  positives. MRB-12 still needs to wire the integrated gate before any product
  or final signoff.
- 2026-06-16: MRB-12 integrated gate evidence exists under
  `reports/surface/mrb12-morph-rendered-beauty-gate-verify-20260616195220/`.
  The gate summary is `validated_with_target_blockers` and `pass=true`, but it
  intentionally keeps `product_claim=false` and `final_signoff=false`; blocked
  target entries for `linux-x64-real-window` and `wasm32-web-browser-canvas`
  do not create a product claim.
- 2026-06-16: MRB-12 product-slice evidence exists under
  `reports/surface/mrb12-product-slice-gate-verify-20260616195413/`.
  `surface-product-slice-summary.json` consumes the MRB gate, uses
  `examples/surface_morph_rendered_studio_shell.tetra` as flagship source,
  marks `morph_rendered_beauty=validated`, and keeps `product_claim=false` /
  `final_signoff=false` for MRB-13 discipline.
- 2026-06-16: Moving runtime frame rendering into `lib.core.morph` means the
  module now has `mem` effects. Keep `lib/core/morph.tetra` `// Effects: mem`
  and `docs/user/standard_library_guide.md` in sync or `verify-docs` will
  block the product-slice gate.
- 2026-06-16: Read-only MRB-12 review found no hard blocker, but flagged
  `morph.render_studio_shell_frame` as a possible second-renderer-path risk.
  Treat it as an MRB-12 evidence bridge only. MRB-13 stable promotion must
  replace it with renderer-owned Block-first proof or keep it explicitly
  experimental/nonclaim.
- 2026-06-16: MRB-13 stable-promotion audit denied promotion. Morph remains
  `EXPERIMENTAL`; see
  `reports/stabilization/surface_morph_rendered_beauty_mrb_13_stable_promotion_audit.md`.
  Fresh MRB-13 gates passed but retained `validated_with_target_blockers`,
  blocked `linux-x64-real-window` and `wasm32-web-browser-canvas` Morph rendered
  beauty product claims, and kept both `product_claim=false` and
  `final_signoff=false`.
- 2026-06-16: MRB-13 audit found that MRB reports/gate summaries did not expose
  a machine-visible `git_commit` field. Post-audit follow-up resolved this for
  newly generated MRB/product-slice evidence: `git_commit` is required to match
  `git_head` in MRB reports, Morph-to-pixels chains, MRB gate summaries,
  product-slice summaries, and same-commit claim evidence. Fresh evidence lives
  under
  `reports/surface/mrb-git-identity-morph-rendered-beauty-gate-20260616171311/`
  and
  `reports/surface/mrb-git-identity-product-slice-gate-20260616171359/`.
  This does not promote Morph: target blockers, dirty worktree,
  `product_claim=false`, `final_signoff=false`, and the renderer-owned stable
  proof gap remain.
- 2026-06-16: Post-MRB-13 wasm browser-canvas Morph target evidence resolved
  the browser target blocker for newly generated MRB gates. Runtime mode
  `wasm32-web-browser-canvas-morph` now emits app-produced browser-canvas RGBA
  frames, `browser-canvas-rgba` render command streams, product visual frame
  provenance, visual-diff evidence without synthetic `block_system` fallback,
  and MRB Morph-to-pixels reports for
  `examples/surface_morph_rendered_studio_shell.tetra`. Fresh evidence lives
  under
  `reports/surface/mrb-wasm-browser-canvas-morph-gate-final-20260616-verify/`
  and
  `reports/surface/mrb-wasm-browser-canvas-product-slice-final-20260616-verify/`.
  Current MRB gate blockers are reduced to `linux-x64-real-window`; Morph still
  stays experimental because the worktree is dirty, product/final signoff is
  false, and renderer-owned stable proof is not complete.
- 2026-06-16: Post-MRB-13 linux real-window Morph target evidence resolved the
  remaining target blocker for newly generated MRB gates. Runtime mode
  `linux-x64-real-window-morph` now emits app-produced, source/hash-linked
  `product_visual` RGBA frames, `wayland-shm-rgba` render command streams,
  real-window/native-input host evidence, visual-diff evidence with separate
  goldens, and MRB Morph-to-pixels reports for
  `examples/surface_morph_rendered_studio_shell.tetra`. Fresh evidence lives
  under
  `reports/surface/mrb-linux-real-window-morph-gate-final-20260616-verify/`
  and
  `reports/surface/mrb-linux-real-window-product-slice-final-20260616-verify/`.
  Current MRB gate target blockers are empty, but Morph stays experimental
  because the worktree is dirty, product/final signoff is false, and
  renderer-owned stable proof is not complete.
- 2026-06-16: Post-MRB-13 renderer-owned stable proof guard made the remaining
  proof gap machine-checkable. New MRB reports carry `renderer_stable_proof`;
  current generated evidence is intentionally `pixel_owner=morph-evidence-bridge`,
  `renderer_owned=false`, `bridge_owned_pixels=true`, and
  `stable_promotion_eligible=false`. Product/final claims now require
  renderer-owned stable proof, MRB gate summaries expose
  `stable_promotion_blockers`, and the stable-candidate contract requires the
  `renderer-owned stable proof` promotion gate. Fresh evidence lives under
  `reports/surface/mrb-renderer-proof-guard-gate-20260616-verify/` and
  `reports/surface/mrb-renderer-proof-guard-product-slice-20260616-verify/`.
  This still does not promote Morph: target blockers are empty, but dirty
  worktree, product/final signoff, and actual renderer-owned stable output
  remain open.
- 2026-06-16: Headless Morph now has actual renderer-owned stable proof.
  `tools/internal/surfacerender.RenderCommandStreamRGBA` renders deterministic
  RGBA bytes from `RenderCommandStreamReport`; headless Morph artifact attach
  uses those bytes for frame order 1 and rebinds the command-stream checksum.
  `buildMorphRenderedBeautyReport` independently rerenders the command stream
  and sets `renderer_stable_proof` to `pixel_owner=surface-renderer` only when
  the renderer checksum matches pixel-golden evidence. Fresh evidence:
  `reports/surface/mrb-renderer-owned-headless-gate-final-20260616-verify/`
  and
  `reports/surface/mrb-renderer-owned-headless-product-slice-final-20260616-verify/`.
  This historical headless-only state was superseded by the all-supported-target
  renderer-owned proof follow-up below; product/final signoff is still false.
- 2026-06-16: All supported Morph rendered beauty targets now have a
  renderer-owned byte-for-byte proof path. The flagship Morph command stream
  emits source-linked commands that reproduce the target initial frame bytes;
  MRB reports set `pixel_owner=surface-renderer` only after
  `RenderCommandStreamRGBA` matches pixel-golden frame checksum evidence.
  The MRB gate summary now derives `renderer_owned_stable_targets` and
  `bridge_owned_stable_targets` from the actual reports. Morph still remains
  unpromoted because dirty checkout audit, product claim, and final signoff are
  still false/open. Fresh evidence:
  `reports/surface/mrb-target-renderer-owned-gate-final-20260616-verify/` and
  `reports/surface/mrb-target-renderer-owned-product-slice-final-20260616-verify/`.
- 2026-06-16: Product/final signoff is now an explicit promotion mode instead
  of a hardcoded impossible state. `surface-runtime-smoke` can mark MRB reports
  with `product_claim=true` and `final_signoff=true` only through explicit flags
  and only when `git_dirty=false` plus renderer-owned stable proof are present.
  `morph-rendered-beauty-gate.sh` and `surface-product-slice-gate.sh` forward
  this through `--product-claim --final-signoff`, but both reject promotion mode
  in the current dirty checkout before heavy evidence generation. Default gates
  still pass as validated nonclaims. Fresh evidence:
  `reports/stabilization/surface_morph_rendered_beauty_promotion_mode_audit.md`,
  `reports/surface/mrb-promotion-aware-gate-default-20260616-verify/`, and
  `reports/surface/mrb-promotion-aware-product-default-20260616-verify/`.
