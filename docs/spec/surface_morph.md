# Tetra Surface Morph Capsule

Status: experimental evidence layer over the Surface Block System.

Morph Capsule is an authoring layer for scoped tokens, materials,
affordances, state lenses, motion presets, and recipes that expand into
`lib.core.block` `Block` values. It is not a new Surface runtime, not a core
widget hierarchy, and not Surface v1 production support.

## Scope

Morph v1 is validated by `tetra.surface.morph.v1` reports inside the normal
`tetra.surface.runtime.v1` envelope. The current source slice is:

- library: `lib/core/morph.tetra`
- examples:
  `examples/surface_morph_command_palette.tetra`,
  `examples/surface_morph_project_dashboard.tetra`,
  `examples/surface_morph_settings.tetra`,
  `examples/surface_morph_editor_shell.tetra`, and
  `examples/surface_morph_control_panel.tetra`
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
- materials, layout modes, typography roles, local asset refs, affordances,
  state lenses, motion presets, recipes, and recipe expansions
- accessibility projection derived from the Block graph
- memory-budget evidence for expanded recipes, Blocks, caches, and frame data
- negative guards for missing tokens, unresolved aliases, missing assets,
  unbounded caches, unsupported targets, and dirty-checkout production claims

Morph recipes must output `Block`. `Button`, `Card`, `TextField`, `TextBox`,
`Sidebar`, and `Modal` are forbidden as core Surface primitives in Morph
evidence.

## Capsule Schema

Morph Capsule v1 is a scoped namespace with an explicit version, local imports,
and no global cascade. A capsule resolves only the token graph and recipes it
imports. Duplicate token sources, duplicate recipe names, unresolved aliases,
alias cycles, and fallback-to-random-default behavior are invalid evidence.

Token graph categories are `color`, `space`, `radius`, `border`, `elevation`,
`opacity`, `typography`, `motion`, `z`, `assets`, and `density`. Every emitted
token has an id, category, kind, value, capsule source, and deterministic
`sha256:` hash. The report-level `token_graph_hash` must match the token graph
hash recorded in the evidence contract.

Materials are named paint grammars over Block paint layers: fill, border,
radius, shadow, and overlay. Unsupported blur is a diagnostic and rejection
case; translucent panels use alpha fill, border, shadow, and overlay evidence
instead of claiming backdrop blur.

Affordances are semantic bundles that project into Block input, event, state,
and accessibility metadata. Morph v1 evidence covers action, `field.text`,
toggle, navigation, region, overlay, and status affordances. Accessibility
projection records role, name, description, action, state, bounds, focus order,
reading order, `labelled_by`, and `label_for`.

Recipes are algebraic expansions into Block graphs or subtrees. They declare
slots and inputs, emit recipe expansion reports, and are invalid if they
promote Button/Card/TextField/TextBox/Sidebar/Modal to core primitives or hide
application state behind platform widgets.

State and motion lenses are deterministic transforms for hover, pressed,
focus-visible, selected, disabled, error, and loading states. Motion presets
record duration, curve, animated properties, deterministic time, and
reduced-motion behavior.

Typography and assets are resolved as semantic type roles, font fallback
diagnostics, icon/image references, asset hashes, tint tokens, and bounded
caches. Local artifact hashes, cache bounds, and negative guards are part of
the release evidence, not optional prose.

## Example Coverage

The Morph example set mirrors the Block-first beauty scenes:

- command palette;
- project dashboard shell;
- settings form;
- editor shell;
- translucent control panel.

Each example imports `lib.core.morph`, expands Morph recipes into
`lib.core.block` values, validates the resulting `BlockTree`, checks focus or
accessibility order, records recipe expansion validity, and stays within the
local memory-budget helper. These examples remain experimental Morph evidence;
they are not Surface v1 production support.

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
