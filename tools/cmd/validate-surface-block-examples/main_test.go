package main

import (
	"strings"
	"testing"
)

func validBlockBeautySource(extra string) string {
	return `module examples.surface.block_apps.surface_block_command_palette

import lib.core.surface as surface
import lib.core.block as block

func theme_dark() -> surface.Color:
    return surface.Color(r: 14, g: 18, b: 22, a: 255)

func theme_light() -> surface.Color:
    return surface.Color(r: 248, g: 250, b: 252, a: 255)

func scene_checksum() -> Int:
    let rect: surface.Rect = surface.Rect(x: 0, y: 0, w: 320, h: 200)
    let paint: block.PaintSpec = block.paint_stack3(block.paint_layer_fill_radius(theme_dark(), 8), block.paint_layer_border(theme_light(), 1, 8), block.paint_layer_shadow(surface.Color(r: 0, g: 0, b: 0, a: 80), 14, 0, 6))
    let text: block.TextSpec = block.text_styled(12, theme_light(), block.text_family_ui(), 14, 600, 18, block.text_align_start(), block.text_wrap_none(), block.text_overflow_ellipsis(), 255)
    let asset: block.AssetRef = block.asset_icon(1, 16, 16, 101)
    let props: block.BlockProps = block.props(block.layout_row(rect, 8), paint, text, block.image_asset(asset, 16, 16), block.input_clickable(), block.event_click(block.action_primary()), block.state_interactive(), block.motion_fast(), block.accessibility_button(12), asset)
    let item: block.Block = block.make(block.id(1), block.id_none(), props)
    let hover: block.StateSelector = block.state_selector_hover()
    let focus: block.StateSelector = block.state_selector_focused()
    let pressed: block.StateSelector = block.state_selector_pressed()
    if block.id_value(item.id) == 1 && block.state_selector_matches(hover, hover) && block.state_selector_matches(focus, focus) && block.state_selector_matches(pressed, pressed):
        return block.paint_feature_flags(paint) + props.motion_ms + props.accessibility_role
    return 0
` + extra
}

func TestValidateExampleSourceRejectsCoreWidgetButtonUsage(t *testing.T) {
	err := validateExampleSource(
		"examples/surface/block_apps/surface_block_command_palette.tetra",
		validBlockBeautySource(`
import lib.core.widgets as widgets

func forbidden() -> Int:
    return widgets.action_save()
`),
	)
	if err == nil || !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("validateExampleSource widget usage err = %v, want forbidden marker error", err)
	}
}

func TestValidateExampleSourceRejectsMissingAccessibilityRoles(t *testing.T) {
	source := strings.ReplaceAll(
		validBlockBeautySource(""),
		"block.accessibility_button(12)",
		"block.accessibility_none()",
	)
	err := validateExampleSource(
		"examples/surface/block_apps/surface_block_command_palette.tetra",
		source,
	)
	if err == nil || !strings.Contains(err.Error(), "accessibility") {
		t.Fatalf(
			"validateExampleSource missing accessibility err = %v, want accessibility error",
			err,
		)
	}
}

func TestValidateExampleSourceRejectsMissingHoverFocusPressedEvidence(t *testing.T) {
	source := strings.ReplaceAll(
		validBlockBeautySource(""),
		"    let pressed: block.StateSelector = block.state_selector_pressed()\n",
		"",
	)
	err := validateExampleSource(
		"examples/surface/block_apps/surface_block_command_palette.tetra",
		source,
	)
	if err == nil || !strings.Contains(err.Error(), "state") {
		t.Fatalf("validateExampleSource missing state err = %v, want state evidence error", err)
	}
}

func TestValidateExampleSourceAcceptsBlockOnlyBeautyEvidence(t *testing.T) {
	if err := validateExampleSource(
		"examples/surface/block_apps/surface_block_command_palette.tetra",
		validBlockBeautySource(""),
	); err != nil {
		t.Fatalf("validateExampleSource valid Block-only beauty source: %v", err)
	}
}
