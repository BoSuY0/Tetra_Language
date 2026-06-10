# Surface Accessibility Target v1

`tetra.surface.accessibility-target.v1` is the production evidence object for
Surface v1 accessibility target support. It is required for
`examples/surface_release_accessibility.tetra` reports when
`accessibility_tree.accessibility_level = "platform-bridge-v1"`.

This schema does not replace the accessibility tree. It binds
`tetra.surface.accessibility-tree.v1` to target-specific inspector evidence so
the release can prove roles, names, values, states, relationships, actions,
bounds, focus order, reading order, and snapshots without promoting broad
screen-reader parity.

## Required Object

Release accessibility reports must include:

```json
{
  "schema": "tetra.surface.accessibility-target.v1",
  "level": "production-accessibility-target-v1",
  "release_scope": "surface-v1-linux-web",
  "tree_schema": "tetra.surface.accessibility-tree.v1"
}
```

The object must match the surrounding report `target` and `runtime`, match the
tree `platform_bridge`, and copy the tree's exact screen-reader evidence token.
The count fields must be derived from the tree in the same report:

- `role_count`
- `named_node_count`
- `state_node_count`
- `relationship_count`
- `action_count`
- `focus_order_count`
- `reading_order_count`
- `snapshot_count`

## Target Evidence

| Target | Inspector | Required bridge evidence |
|---|---|---|
| `headless` | `deterministic-accessibility-tree-inspector-v1` | deterministic tree export only; no target-host bridge claim |
| `linux-x64` | `linux-accessibility-platform-probe-v1` | `linux_accessibility_host_bridge_v1`, host bridge evidence, platform probe artifact |
| `wasm32-web` | `browser-accessibility-snapshot-mirror-v1` | browser canvas target evidence, browser accessibility snapshot, browser accessibility mirror |

Linux and web evidence are target-specific. A browser snapshot cannot prove a
desktop Linux bridge, and Linux host evidence cannot prove the browser mirror.

## Negative Guards

Validators require these guards to be true:

- `focusable_unnamed_rejected`
- `aria_dom_desktop_bridge_rejected`
- `full_atspi_without_screen_reader_rejected`
- `metadata_platform_overclaim_rejected`
- `shuffled_focus_order_rejected`
- `shuffled_reading_order_rejected`

The required nonclaims are:

- full screen-reader parity
- desktop aria bridge
- metadata platform overclaim
- unnamed focusable block
- AT-SPI full support

## Verification

The narrow validator and generator checks are:

```sh
go test -buildvcs=false ./tools/validators/surface -run 'AccessibilityRequiresProductionTargetEvidence|ReleaseAccessibility|AccessibilityTarget' -count=1
go test -buildvcs=false ./tools/cmd/surface-runtime-smoke -run 'ReleaseAccessibility|Accessibility' -count=1
```

Release scripts for headless, Linux, and wasm32-web accessibility reports must
emit `accessibility_target` next to `accessibility_tree` and then pass
`tools/validators/surface.ValidateReport`.
