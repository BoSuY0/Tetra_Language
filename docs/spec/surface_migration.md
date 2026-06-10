# Surface Widget To Block/Morph Migration

Surface migration evidence is experimental under `ui.surface-block-system`.
The P32 contract uses schema `tetra.surface.migration-report.v1` and level
`surface-widget-block-migration-v1`. The supported deterministic path is
`scripts/release/surface/migration-gate.sh` and
`validate-surface-migration-report`.

The goal is compatibility without freezing the old helper layer as the final
architecture. `lib.core.widgets` remains available for existing Surface v1 app
shapes, while new production UI should prefer `lib.core.block` and
`lib.core.morph` recipes.

Required mapping evidence:

- `Panel` maps to a ComponentTree panel, a Block column/layout region, and the
  Morph `region_panel` recipe.
- `Button` maps to a ComponentTree button, a Block row/action control, and the
  Morph `control_action` recipe.
- `TextBox` maps to a ComponentTree textbox, a Block text-input row, and the
  Morph `field_text` recipe.
- `StatusText` maps to ComponentTree text, a fixed Block status area, and the
  Morph `status_message` recipe.

Existing Surface v1 examples must still pass. Migration examples must show both
the compatibility helper and the Block/Morph replacement path in the same
checked source. The canonical source-level smoke is
`examples/surface_migration_widgets_to_block.tetra`.

Fake-claim rejection is part of the contract. The validator rejects:

- widgets declared as the core final architecture;
- breaking Surface v1 widget examples without migration;
- missing widget-to-Block/Morph mappings;
- deprecation before production examples and gates cover replacement;
- docs that fail to recommend Block/Morph for new production UI.

This evidence is not a deprecation announcement. It is not a claim that the
legacy widget/component tree is the final architecture, not a promise that old
examples may break, and not a platform widget compatibility claim.
