# Tetra Surface Morph Capsule

Status: experimental evidence layer over the Surface Block System.

Morph Capsule is an authoring layer for scoped tokens, materials, affordances, state lenses, motion
presets, and recipes that expand into `lib.core.block` `Block` values. It is not a new Surface
runtime, not a core widget hierarchy, and not Surface v1 production support.

Claim tier: `EXPERIMENTAL`. Morph evidence may inform future `PROD_STABLE_SCOPED` Surface authoring
only after the MRB-11 claim gate and a final same-commit audit prove the broader scope. Until then
it remains a `NONCLAIM` for React API compatibility, CSS cascade/runtime compatibility, DOM-authored
UI, platform-native widgets, GPU rendering, Windows/macOS production, and broad desktop parity.

Stable promotion is frozen separately in
`docs/spec/surface/morph/surface_morph_stable_candidate.md`. The machine-readable contract is
`docs/spec/surface/morph/surface_morph_stable_candidate_contract.json` and is validated by
`tools/cmd/validate-surface-morph-stable-candidate`. That validator is a design freeze guard only;
it is not the Surface product gate and does not promote Morph out of the `EXPERIMENTAL` tier.

Rendered beauty proof is tracked separately by
`docs/spec/surface/morph/surface_morph_rendered_beauty.md` and
`docs/spec/surface/morph/surface_morph_rendered_beauty_contract.json`, validated through
`tools/cmd/validate-surface-morph-rendered-beauty`. That contract requires the full
Morph-to-Block-to-render-commands-to-pixels-to-golden evidence chain before any Morph beauty product
claim.

## Scope

Morph v1 is validated by `tetra.surface.morph.v1` reports inside the normal
`tetra.surface.runtime.v1` envelope. The current source slice is:

- library: `lib/core/morph/morph.tetra`
- examples:
  - `examples/surface/morph_core/surface_morph_command_palette.tetra`
  - `examples/surface/morph_core/surface_morph_project_dashboard.tetra`
  - `examples/surface/morph_core/surface_morph_settings.tetra`
  - `examples/surface/morph_core/surface_morph_editor_shell.tetra`
  - `examples/surface/morph_core/surface_morph_glass_panel.tetra`
  - `examples/surface/morph_core/surface_morph_studio_shell.tetra`
- report validator: `tools/cmd/validate-surface-morph-report`
- gate: `scripts/release/surface/morph-gate.sh`

The gate requires deterministic headless evidence, same-commit validation, a Block System evidence
dependency, local artifact hashes, and a `tetra.surface.morph.gate.v1` summary. It also runs the P07
token graph validator against `docs/spec/surface/surface_token_graph_contract.json`, requiring one
capsule source of truth, explicit imports, no global cascade, fixed override order, density/DPI
mappings, and diagnostics for missing tokens, duplicate sources, raw literals, alias cycles, and CSS
cascade/runtime admission.

## Evidence Contract

A valid Morph report records:

- `morph.schema = tetra.surface.morph.v1`
- `morph.quality_level = deterministic-headless-morph-capsule-v1`
- `morph.module = lib.core.morph`
- `morph.surface_scope = surface-morph-experimental-linux-web`
- capsule and token graph hashes
- token graph source-of-truth, fixed override order, density/DPI mappings, and diagnostics
- materials, layout modes, typography roles, local asset refs, affordances, state lenses, motion
  presets, recipes, recipe expansions, and recipe-authored reference apps
- accessibility projection derived from the Block graph
- memory-budget evidence for expanded recipes, Blocks, caches, and frame data
- negative guards for missing tokens, unresolved aliases, missing assets, unbounded caches,
  unsupported targets, and dirty-checkout production claims

Morph recipes must output `Block`. `Button`, `Card`, `TextField`, `TextBox`, `Sidebar`, and `Modal`
are forbidden as core Surface primitives in Morph evidence.

## Recipe Authoring

P08 recipe authoring is the React-like ergonomics layer for Morph, but it is still pure Block
construction. A recipe names an authoring pattern, declares slots and inputs, and reports the Block
IDs it expands into. Recipes must not allocate hidden app state, call platform widgets, introduce a
React/Electron/DOM runtime, or promote a new core primitive.

The required recipe set is:

- `control.action@1`
- `field.text@1`
- `command.item@1`
- `region.panel@1`
- `form.field@1`
- `nav.item@1`
- `metric.tile@1`
- `dialog.panel@1`
- `toast.notification@1`
- `tab.item@1`
- `list.row@1`
- `app.shell@1`
- `toolbar@1`
- `split.pane@1`
- `status.bar@1`
- `settings.form@1`
- `log.row@1`
- `empty.state@1`
- `error.panel@1`

The required affordance set is `action`, `field.text`, `toggle`, `navigation`, `region`, `overlay`,
and `status`. The Morph report also records six recipe-authored reference apps and rejects missing
app rows, hidden app state, React/Electron/DOM runtime use, platform widgets, and non-`Block` output
primitives. The user-facing cookbook is `docs/user/surface/surface_morph_recipe_cookbook.md`.

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

go run ./tools/cmd/validate-surface-token-graph \
  --contract docs/spec/surface/surface_token_graph_contract.json \
  --report reports/surface-morph/headless/surface-headless-morph.json \
  --root .
```

## Nonclaims

Morph v1 does not claim production Surface support, platform-native widgets, GPU rendering, a CSS
cascade, DOM application logic, React/Electron runtime support, a browser framework, or cross-target
desktop parity. Surface v1 release-supported apps still use the bounded `lib.core.widgets` and
related release modules documented in `docs/user/surface/surface_guide.md`.
