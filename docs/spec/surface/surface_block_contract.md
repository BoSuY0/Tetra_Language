# Tetra Surface Block Contract

Status: contract freeze for the experimental Block System path.

This contract freezes `Block` as the runtime primitive for the Surface Block/Morph product track.
Compatibility helpers such as Button, Card, TextBox, TextField, Sidebar, and Modal may exist as
recipes or helper APIs, but they are not core Surface primitives.

The machine-readable contract lives in `docs/spec/surface/surface_block_contract.json` and is
validated by:

```sh
go run ./tools/cmd/validate-surface-block-contract \
  --contract docs/spec/surface/surface_block_contract.json
```

Individual reports can be checked for the independent Block evidence shape with:

```sh
go run ./tools/cmd/validate-surface-block-contract \
  --report reports/surface-block/p18-budget/headless/surface-headless-block-system.json
```

## ABI Slots

`Block` is the only core primitive. Its ABI surface is:

- `Block.id`
- `Block.parent_id`
- `Block.props`

`BlockProps` is intentionally compact enough for the current 10-slot ABI limit. The frozen slots
are:

- `layout_mode`
- `paint_layers`
- `text_len`
- `visual_asset`
- `interaction_flags`
- `state_flags`
- `motion_ms`
- `accessibility_role`

## Versioned Report Schemas

The contract freezes these report schema names for downstream validators:

- `tetra.surface.block-graph.v1`
- `tetra.surface.resolved-block.v1`
- `tetra.surface.paint-command.v1`
- `tetra.surface.layout-pass.v1`
- `tetra.surface.block-accessibility-node.v1`

Reports with `block_graph` evidence must also include paint command, layout pass, and Block
accessibility tree evidence. A graph-only screenshot or metadata-only report is not enough.

## Renderer Contract

The current product baseline remains software-rendered:

- `software-rgba-headless`
- `wayland-shm-rgba`
- `browser-canvas-rgba`

GPU renderer production support remains a nonclaim until a later packet proves and gates it.

## Failure Cases

The contract validator rejects:

- `core_primitives` missing `Block`;
- `core_primitives` containing Button, Card, TextField, TextBox, Sidebar, or Modal;
- `block_graph` reports without paint command, layout pass, and accessibility evidence.
