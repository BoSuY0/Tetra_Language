# Tetra Surface Token Graph Contract

Status: current contract for the Morph token/theme/style graph evidence slice.
This contract is a boundary guard for the Block/Morph track, not production
Morph support and not a browser CSS cascade.

The machine-readable contract is
`docs/spec/surface_token_graph_contract.json` and is validated by:

```sh
go run ./tools/cmd/validate-surface-token-graph \
  --contract docs/spec/surface_token_graph_contract.json \
  --report reports/surface-morph/gate/headless/surface-headless-morph.json \
  --root .
```

## Source Of Truth

`lib.core.morph` owns the current token graph source under the
`tetra.surface.morph.app` namespace. App code imports the capsule explicitly and
does not use a global cascade. The contract requires one canonical source named
`capsule`; multiple color sources are rejected.

Required typed categories:

- color
- space
- radius
- border
- elevation
- opacity
- typography
- motion
- z
- assets
- density

The current reference sources are the five
`examples/surface_morph_*.tetra` recipe-authored apps. They use Morph helpers
and must not contain raw color/style literals. Raw literals remain allowed only
in the canonical token source, legacy Surface v1 style compatibility module,
and experimental raw Block fixtures until recipe migration covers those
examples.

## Override Order

The graph uses a fixed override order:

```text
base -> theme -> density -> variant -> state -> local
```

The validator rejects drift from this order so style resolution cannot become an
implicit cascade.

## Density

The contract maps headless, linux-x64 real-window, and wasm32-web
browser-canvas evidence to `density.1x`, `target_dpi = 96`,
`scale_milli = 1000`, and `integer-half-up-v1` rounding. Higher-density targets
must add explicit mappings before they can be claimed.

## Diagnostics

`validate-surface-token-graph` requires diagnostics for:

- alias cycles;
- missing tokens;
- duplicate token sources;
- raw literals outside allowed scopes;
- unresolved fallbacks;
- CSS cascade/runtime admission;
- multiple color sources;
- override order drift;
- density/DPI mapping drift.

## Nonclaims

This contract records boundary evidence only. It does not claim production
Morph, a DOM style runtime, a CSS cascade, React runtime support, Electron
runtime support, platform-native widgets, GPU rendering, or broad desktop
parity.
