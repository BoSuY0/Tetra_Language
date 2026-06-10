# Surface Migration Validator

`tools/validators/surfacemigration` validates
`tetra.surface.migration-report.v1` evidence for
`surface-widget-block-migration-v1`.

The package owns widget/component-tree to Block/Morph migration evidence. It
checks source coverage, compatibility rows, rewritten app-shape evidence, and
negative guards so migration support does not imply legacy widget promotion.
