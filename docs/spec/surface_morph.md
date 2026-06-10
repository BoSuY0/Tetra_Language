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
- materials, layout modes, typography roles, local asset refs, affordances,
  state lenses, motion presets, recipes, and recipe expansions
- accessibility projection derived from the Block graph
- memory-budget evidence for expanded recipes, Blocks, caches, and frame data
- negative guards for missing tokens, unresolved aliases, missing assets,
  unbounded caches, unsupported targets, and dirty-checkout production claims

Morph recipes must output `Block`. `Button`, `Card`, `TextField`, `TextBox`,
`Sidebar`, and `Modal` are forbidden as core Surface primitives in Morph
evidence.

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
