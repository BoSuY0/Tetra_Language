# Tetra Surface Morph Capsule

Status: experimental evidence layer over the Surface Block System, with a
stable style/token graph candidate boundary.

Morph Capsule is an authoring layer for scoped tokens, materials,
affordances, state lenses, motion presets, and recipes that expand into
`lib.core.block` `Block` values. It is not a new Surface runtime, not a core
widget hierarchy, and not Surface v1 production support.

Under the planned `PROD_STABLE_SCOPED_LINUX_WEB_APP_UI` Surface production
contract in `docs/spec/surface_production_platform.md`, Morph now records a
stable style/token graph candidate that can serve as the CSS replacement
boundary for scoped Surface work. Morph remains experimental until target
evidence, validators, and final production gates promote it. Morph evidence
must not claim broad Electron replacement, CSS runtime parity, GPU production,
or cross-platform desktop replacement support.

## Scope

Morph v1 is validated by `tetra.surface.morph.v1` reports inside the normal
`tetra.surface.runtime.v1` envelope. The current source slice is:

- library: `lib/core/morph.tetra`
- example: `examples/surface_morph_command_palette.tetra`
- report validator: `tools/cmd/validate-surface-morph-report`
- gate: `scripts/release/surface/morph-gate.sh`

The gate requires deterministic headless evidence, same-commit validation, a
Block System evidence dependency, local artifact hashes, and a
`tetra.surface.morph.gate.v1` summary.

## Evidence Contract

A valid Morph report records:

- `morph.schema = tetra.surface.morph.v1`
- `morph.quality_level = deterministic-headless-morph-capsule-v1`
- `morph.module = lib.core.morph`
- `morph.surface_scope = surface-morph-experimental-linux-web`
- capsule and token graph hashes
- `morph.style_graph.schema = tetra.surface.morph.style-graph.v1`
- `morph.style_graph.css_replacement_level =
  typed-style-graph-candidate-v1`
- `morph.authoring.schema = tetra.surface.morph.authoring.v1`
- materials, layout modes, typography roles, local asset refs, affordances,
  state lenses, motion presets, recipes, and recipe expansions
- accessibility projection derived from the Block graph
- memory-budget evidence for expanded recipes, Blocks, caches, and frame data
- negative guards for missing tokens, unresolved aliases, missing assets,
  unbounded caches, unsupported targets, and dirty-checkout production claims

## Style Graph Contract

The Morph style graph freezes the candidate vocabulary used by this capsule
track. A valid `tetra.surface.morph.style-graph.v1` report records:

- `css_replacement_level = typed-style-graph-candidate-v1`;
- token categories for color, space/spacing, radius, border, elevation,
  opacity, typography/type, motion, z, assets, and density;
- material slots for fill, border, radius, shadow, and overlay;
- affordance roles for action, field text, toggle, navigation, region, overlay,
  and status;
- recipe output constrained to `Block`;
- state selectors for hover, pressed, focus visible, selected, disabled, error,
  and loading;
- motion properties for fill, opacity, and transform;
- override order from capsule imports through tokens, materials, affordances,
  state lenses, motion, recipes, and accessibility safety;
- conflict diagnostics for alias cycles, duplicate recipes, duplicate token
  sources, unresolved tokens, raw literals, unsupported CSS cascade imports,
  forbidden runtime imports, global style leaks, specificity ambiguity, and raw
  CSS runtime imports;
- an import allowlist containing only `lib.core.block` and `lib.core.morph`.

Morph reports must prove that CSS cascade imports, DOM runtime imports, React
runtime imports, Electron runtime imports, global style leaks, specificity-like
override ambiguity, and raw CSS runtime imports are rejected. Reports must also
prove that there is no selector engine and no specificity scoring. These
diagnostics are evidence for the style graph boundary, not a CSS-runtime
compatibility claim.

## Authoring Boundary

Morph authoring evidence uses `tetra.surface.morph.authoring.v1` with
`level = production-recipe-authoring-v1`. This is an evidence level for the
experimental Morph track, not Surface v1 production support. A valid report
proves:

- 11 stable recipe families are present: action, field, toggle, command item,
  nav item, panel, dialog overlay, tabs, list, table-lite, and status;
- every recipe declares inputs, slots, state, and accessibility projection;
- every public recipe has a reported Block-only expansion;
- all public recipes are polished authoring surfaces;
- author-facing recipe input width is bounded to 16 fields or fewer;
- raw 80-field Block authoring is rejected;
- direct Block prop editing and raw literal styles are rejected;
- designer inputs are token-driven, with generated Block props only.

Morph recipes must output `Block`. `Button`, `Card`, `TextField`, `TextBox`,
`Sidebar`, and `Modal` are forbidden as core Surface primitives in Morph
evidence. The validator also rejects hidden app state in recipes, platform
widget recipe output, unreported expansion, component bloat, and core primitive
promotion.

## Commands

```sh
bash scripts/release/surface/surface-headless-morph-smoke.sh \
  --report-dir reports/surface-morph/headless

bash scripts/release/surface/morph-gate.sh \
  --report-dir reports/surface-morph/gate
```

For one report:

```sh
go run ./tools/cmd/validate-surface-morph-report \
  --report reports/surface-morph/headless/surface-headless-morph.json
```

## Nonclaims

Morph v1 does not claim production Surface support, platform-native widgets,
GPU rendering, a CSS cascade, DOM application logic, React/Electron runtime
support, a browser framework, or cross-target desktop parity. Surface v1
release-supported apps still use the bounded `lib.core.widgets` and related
release modules documented in `docs/user/surface_guide.md`.
