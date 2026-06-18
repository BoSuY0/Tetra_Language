# Surface Cookbook

Status: current for `surface-v1-linux-web` project onboarding evidence.

This cookbook is a design-system guide for the bounded Surface Linux/web scope.
It does not promote Morph to production support, does not add a CSS runtime,
and does not make a broad Electron/React/CSS replacement claim.

Use `tetra new surface-app` to create a small Surface project from the
Block/Morph recipe layer:

```sh
tetra new surface-app --template command-palette my-palette
tetra new surface-app --template settings my-settings
tetra new surface-app --template dashboard my-dashboard
tetra new surface-app --template editor-shell my-editor
tetra new surface-app --template studio-shell my-studio
tetra new surface-app --template multi-window-notes my-notes
tetra new surface-app --template web-canvas my-web-canvas
```

Each generated project contains `Capsule.t4`, `src/main.tetra`,
`surface-template.json`, `design/tokens.tetra`, `design/recipes.tetra`, and a
README. The main source imports `lib.core.surface`, `lib.core.block`, and
`lib.core.morph`; the notes template also imports `lib.core.surface_app_shell`
for the scoped app-shell window model.

## Claim Tiers

Surface docs use these tiers:

| Tier | Cookbook use |
| --- | --- |
| `PROD_STABLE_SCOPED` | guarded vocabulary for the named `surface-v1-linux-web` product scope after product-gate evidence and final audit; this cookbook is not the final verdict |
| `BETA_TARGET_HOST` | target-host path with evidence but not current production support |
| `EXPERIMENTAL` | Block/Morph/visual recipe evidence that can guide authors without becoming production support |
| `UNSUPPORTED` | target or feature with no current release support |
| `NONCLAIM` | explicit boundary for Electron APIs, React APIs, CSS cascade/runtime compatibility, DOM-authored UI, platform widgets, Windows/macOS production, GPU rendering, full rich text, full bidi, and full screen-reader support |

The scoped product evidence gate is:

```sh
bash scripts/release/surface/product-gate.sh \
  --report-dir reports/surface-product-v1
```

That gate runs the release gate, claim scanner, manifest validator, and docs
verifier. It is not the final P29 `PROD_STABLE_SCOPED` verdict.

## Design System Stack

Surface design-system authoring is layered:

| Layer | Current role | Evidence |
| --- | --- | --- |
| `Block` primitives | visual and interaction data model: layout, paint, text, image/assets, input/events, state selectors, motion, accessibility, and asset refs | `tetra.surface.block-system.gate.v1`, Block contract, renderer/layout/visual reports |
| Token/theme/style graph | scoped source of truth for color, space, radius, border, elevation, opacity, typography, motion, z, assets, and density | `tetra.surface.token-graph.contract.v1` and `validate-surface-token-graph` |
| Morph recipes | authoring names such as `control.action@1`, `field.text@1`, and `region.panel@1` that expand to `Block` | `tetra.surface.morph.v1` and `tetra.surface.morph.gate.v1` |
| Reference apps | product-shape evidence for command palette, settings, dashboard, editor shell, file manager, dialogs, localized forms, accessibility-heavy forms, multi-window notes, and migration | `surface-reference-app-suite-v1` plus visual, interaction, accessibility, performance, token/theme, layout, and artifact-hash rows |

The stack is Block-first. A button-like or card-like shape is a configured
Block or a Morph recipe expansion, not a required core widget primitive and not
a platform-native widget. `lib.core.widgets` remains a Surface v1 compatibility
layer for the release subset and the migration example.

## Block Primitives

Use `lib.core.block` for the primitive model:

- `LayoutSpec` controls fixed layout, constraints, aspect sizing, overflow and
  clip policy, scroll bounds, target density, and stable pixel snapping.
- `PaintSpec` and `PaintLayer` describe deterministic software RGBA paint:
  fill, gradient, image fill, border, radius clip, shadow, overlay, outline,
  text, and icon command order.
- `TextSpec`, `ImageSpec`, `InputSpec`, `EventSpec`, `StateSpec`,
  `MotionSpec`, `AccessibilitySpec`, and `AssetRef` keep text, assets, input,
  state, motion, accessibility metadata, and local assets in the Block graph.

Block reports must carry layout, paint, accessibility, frame checksum, bounded
cache, and artifact-hash evidence. No GPU rendering, no blur/backdrop-blur
production support, no CSS/browser compositor parity, no DOM UI, no React
runtime, and no platform widgets are claimed.

## Tokens, Themes, Density, And Style Graph

The token graph is the Surface replacement boundary for CSS runtime dependency,
but it is not CSS cascade compatibility. A valid graph has one capsule source of
truth, explicit imports, and fixed resolution order:

```text
base -> theme -> density -> variant -> state -> local
```

Use typed tokens for color, space, radius, border, elevation, opacity,
typography, motion, z, assets, and density. Keep dark/light theme values,
compact/comfortable density rows, DPI mappings, and state overrides in the
token graph rather than in ad hoc app literals. The validator rejects raw
literals, duplicate sources, missing tokens, unresolved aliases, alias cycles,
CSS cascade/runtime admission, multiple color sources, override-order drift,
and density/DPI drift.

CSS nonclaims are deliberate: no selector cascade, no pseudo-class engine, no
CSS layout engine, no CSS animation runtime, no browser style inheritance, and
no backdrop-filter/backdrop-blur production support are claimed by Surface v1.

## Morph Recipes

Morph recipes are authoring conveniences over Block. A recipe declares inputs
and slots, records a `RecipeExpansion`, and outputs `Block`. It must not create
hidden app state, must not call platform widgets, must not depend on
React/Electron/DOM runtime behavior, and must not promote any core primitive:
no Button, Card, TextField, Sidebar, or Modal core primitive is claimed.

Use `docs/user/surface_morph_recipe_cookbook.md` for the exact recipe list and
`scripts/release/surface/morph-gate.sh --report-dir reports/surface-morph/gate`
for experimental Morph evidence.

## Visual Evidence

Visual evidence is evidence-backed, not screenshot-only:

```sh
bash scripts/release/surface/visual-gate.sh \
  --report-dir reports/surface-visual/gate
```

The visual gate records deterministic frame/golden/diff rows, token/theme
evidence, layout evidence, accessibility evidence, performance evidence,
same-commit golden heads, and negative guards for screenshot-only, stale-golden,
major-drift, missing Block graph, missing layout, missing accessibility, and
missing performance evidence. This is visual infrastructure evidence and not a
production beauty claim by itself.

## Recipes

Command palette:

- `morph.recipe_region_panel`
- `morph.recipe_field_text`
- `morph.recipe_command_item`
- `morph.recipe_control_action`

Settings:

- `morph.recipe_form_field`
- `morph.recipe_field_text`
- `morph.recipe_tab_item`
- `morph.recipe_control_action`

Dashboard:

- `morph.recipe_region_panel`
- `morph.recipe_metric_tile`
- `morph.recipe_list_row`
- `morph.recipe_toast_notification`

Editor shell:

- `morph.recipe_nav_item`
- `morph.recipe_tab_item`
- `morph.recipe_command_item`
- `morph.recipe_region_panel`

Multi-window notes:

- `morph.recipe_region_panel`
- `morph.recipe_list_row`
- `morph.recipe_field_text`
- `morph.recipe_control_action`
- `lib.core.surface_app_shell` window lifecycle helpers

Web-canvas:

- `morph.recipe_region_panel`
- `morph.recipe_metric_tile`
- `morph.recipe_command_item`
- `morph.recipe_field_text`
- `target "wasm32-web"` in the generated capsule

## Template Smoke

The release smoke for templates is:

```sh
bash scripts/release/surface/surface-template-smoke.sh \
  --report-dir reports/surface-templates/gate
```

It generates all seven templates, checks, builds, runs, inspects, visually tests,
packages them as tar archives, and validates
`tetra.surface.template-smoke.v1` / `surface-template-smoke-v1` evidence.

Template recipes must stay Block/Morph-authored and must not depend on
`lib.core.widgets`, platform widgets, user JavaScript app logic, React,
Electron, or a CSS runtime.
