# Surface Asset Pipeline Evidence

`tetra.surface.asset-pipeline.v1` is the production evidence schema for the
Surface Block asset pipeline inside the scoped Block System implementation
track. It covers local fonts, icons, raster images, and static vector assets for
the supported `surface-v1-linux-web` release scope.

This evidence does not promote the full `ui.surface-block-system` feature to
current production support by itself. It proves that the asset slice used by
Block reports is hash-addressed, local-only, bounded, decoder-scoped, and
security checked.

## Report Shape

Runtime reports that use `examples/surface_block_assets.tetra` must include:

- `asset_pipeline.schema = tetra.surface.asset-pipeline.v1`
- `asset_pipeline.level = production-asset-pipeline-v1`
- `asset_pipeline.release_scope = surface-v1-linux-web`
- `asset_pipeline.manifest_schema = tetra.surface.block-assets.v1`
- matching `manifest_hash` and `hash_algorithm = sha256`
- `font_count`, `icon_count`, `image_count`, `vector_count`, and `asset_count`
  matching `block_asset_manifest.assets`
- `render_command_count` matching `block_asset_render_commands`
- `diagnostic_count` matching `block_asset_diagnostics`

The paired `block_asset_manifest` remains the source of concrete asset rows.
Every asset must be local or embedded and must carry a valid `sha256:` digest
before any decoder evidence is accepted.

## Decoders

The production slice is intentionally narrow:

- `decoder_policy = safe-local-asset-decoders-v1`
- `font_decoder = font-table-hash-verified-v1`
- `icon_decoder = icon-mask-tint-rgba-v1`
- `image_decoder = png-rgba-bounds-checked-v1`
- `vector_decoder = svg-tiny-static-sanitized-v1`

The vector decoder is a static SVG Tiny subset for known local assets. It is
not full SVG/CSS/SMIL support.

## Cache And Bounds

The asset pipeline must mirror `block_asset_cache`:

- `cache_strategy = bounded-lru`
- positive `cache_budget_bytes`
- `cache_used_bytes <= cache_budget_bytes`
- matching `cache_entry_count`
- `cache_bounded = true`

Raster decode evidence must include bounds checks, and vector evidence must
include sanitized static parsing. Oversized raster decode attempts are rejected.

## Required Guards

`asset_pipeline.negative_guards` must prove:

- missing asset hashes are rejected
- decoder execution before hash validation is rejected
- remote fonts are rejected
- network asset fetches are rejected
- unbounded caches are rejected
- missing assets require fallback diagnostics
- unsafe SVG payloads are rejected
- oversized raster decode is rejected

Required runtime cases include:

- `block asset vector safe decode`
- `block asset unsafe SVG rejected`
- `block asset remote font rejected`

## Nonclaims

The asset pipeline does not claim:

- network assets
- remote fonts
- untrusted SVG scripting
- full SVG/CSS/SMIL
- arbitrary image codecs

These nonclaims are validator-enforced in production asset evidence.
