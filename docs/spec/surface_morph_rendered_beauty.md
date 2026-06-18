# Tetra Surface Morph Rendered Beauty

Status: experimental contract for the Morph-to-pixels proof chain.

This experimental contract defines the evidence required for Morph's intended
beauty layer. It does not grant product or final signoff, promote Morph to
stable, or claim Electron, React, DOM, CSS, GPU, macOS, or Windows parity.

## Boundary

Morph remains the authoring and beauty layer for tokens, materials, recipes,
state lenses, motion, and scene-level composition. Block remains the only core
Surface primitive.

The required proof chain is:

```text
Morph source/capsule -> token graph -> recipe expansions -> resolved Morph scene
-> Block scene snapshot -> render command stream -> real RGBA/PNG frame artifact
-> separate pixel golden artifact -> MRB-11 claim gate
```

`Button`, `Card`, `TextField`, `TextBox`, `Sidebar`, and `Modal` may exist only
as recipes, helper APIs, or compatibility facades that expand to Block. They are
not core primitives.

## Machine Contract

The machine-readable contract is:

- `docs/spec/surface_morph_rendered_beauty_contract.json`

The validator is:

```sh
go run ./tools/cmd/validate-surface-morph-rendered-beauty \
  --contract docs/spec/surface_morph_rendered_beauty_contract.json
```

The report schema required by this contract is:

- `tetra.surface.morph-rendered-beauty.v1`

## Required Evidence

A valid report must include:

- Morph source identity and source SHA-256;
- capsule hash;
- token graph hash;
- token coverage, including token count and covered token categories;
- recipe coverage, including recipe count, recipe names, and recipe expansion
  count;
- resolved Morph scene hash;
- Block scene snapshot hash;
- typed Block scene snapshot evidence with rich layout, paint, text, image,
  input, event, state, motion, and accessibility specs;
- typed source-linked render command stream, including source node IDs, Morph
  recipe IDs, paint payload, Block scene hash, command stream hash, and
  command count;
- non-marker text and icon raster evidence for the beauty path;
- app-produced frame artifact path and SHA-256;
- Morph recipe hash, Block scene hash, and render command stream hash linked
  from pixel evidence back to the app source;
- separate golden artifact path and SHA-256;
- pixel diff metrics;
- renderer stable-proof evidence naming whether pixels are owned by the Surface
  renderer or by an experimental evidence bridge;
- target, scenario name, git head, and matching git commit alias;
- product claim and final signoff booleans;
- negative guards.

The supported evidence targets are:

- `headless`
- `linux-x64-real-window`
- `wasm32-web-browser-canvas`

## Renderer-Owned Stable Proof

Renderer-owned stable proof is stricter than target frame evidence. The proof
requires the Surface renderer to derive RGBA bytes from the Block-first render
command stream and match the pixel-golden frame checksum.

Current post-MRB-13 evidence proves this for `headless`,
`linux-x64-real-window`, and `wasm32-web-browser-canvas`. The proof remains
claim-gated: target reports may be renderer-owned and stable-promotion
eligible, but product/final claims remain false until clean checkout audit and
explicit release signoff are complete.

Promotion mode is explicit. `surface-runtime-smoke` can set
`product_claim=true` / `final_signoff=true` only when requested with the Morph
rendered beauty signoff flags and only after the report is clean
`git_dirty=false` with renderer-owned stable proof. The integrated MRB and
product-slice gates forward this through `--product-claim --final-signoff`, but
they reject that mode before evidence generation if the checkout is dirty. The
default gate path remains a validated nonclaim.

Unsupported targets remain nonclaims for Morph rendered beauty:

- `macos`
- `windows`
- `wasm32-wasi`

## Negative Guards

The contract requires guards for:

- metadata-only evidence;
- self-golden evidence;
- precomputed or fixture-only product frames;
- missing frame artifacts;
- DOM-authored UI;
- CSS runtime;
- React runtime;
- Electron runtime;
- native widgets;
- hidden app state;
- non-Block output;
- dirty-checkout production claims;
- unsupported target claims;
- product/final claims without renderer-owned stable proof.

## Nonclaims

This contract is not:

- a Morph stable promotion;
- an Electron runtime claim;
- a React runtime claim;
- a CSS runtime claim;
- a DOM-authored UI claim;
- a GPU renderer production claim;
- a macOS production claim;
- a Windows production claim.
