package surface

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSurfaceTreeAppUsesHardenedComponentTreeAPI(t *testing.T) {
	root := repoRoot(t)
	componentRaw, err := os.ReadFile(
		filepath.Join(root, "lib", "core", "surface", "component.tetra"),
	)
	if err != nil {
		t.Fatalf("read component helper module: %v", err)
	}
	componentModule := string(componentRaw)
	for _, want := range []string{
		"func tree_add_root(",
		"func tree_add_child(",
		"func tree_layout_column(",
		"func tree_layout_row(",
		"func tree_hit_test(",
		"func tree_build_dispatch_path(",
	} {
		if !strings.Contains(componentModule, want) {
			t.Fatalf("lib/core/surface/component.tetra must expose hardened helper %q", want)
		}
	}

	appRaw, err := os.ReadFile(
		filepath.Join(root, "examples", "surface", "toolkit", "surface_tree_app.tetra"),
	)
	if err != nil {
		t.Fatalf("read Surface tree example: %v", err)
	}
	app := string(appRaw)
	for _, want := range []string{
		"component.tree_add_root(",
		"component.tree_add_child(",
		"component.tree_layout_column(",
		"component.tree_layout_row(",
		"component.tree_hit_test(",
		"component.tree_build_dispatch_path(",
	} {
		if !strings.Contains(app, want) {
			t.Fatalf("surface_tree_app.tetra must use hardened helper %q", want)
		}
	}
	for _, forbidden := range []string{
		".first_child =",
		".child_count =",
		".child_index =",
		".parent_id =",
		".id =",
		".tree.len = 7",
		"component.TreeNode(id:",
		"component.tree_hit_test_static(",
		"component.contains(reset.rect, event.x, event.y)",
		"component.contains(submit.rect, event.x, event.y)",
		"component.contains(box.rect, event.x, event.y)",
		"component.contains(label.rect, event.x, event.y)",
		"component.contains(row.rect, event.x, event.y)",
		"component.contains(column.rect, event.x, event.y)",
	} {
		if strings.Contains(app, forbidden) {
			t.Fatalf(
				"surface_tree_app.tetra still has hardcoded pointer hit-test branch %q",
				forbidden,
			)
		}
	}
}

func TestSurfaceToolkitFormUsesReusableWidgetModule(t *testing.T) {
	root := repoRoot(t)
	widgetsRaw, err := os.ReadFile(filepath.Join(root, "lib", "core", "widgets", "widgets.tetra"))
	if err != nil {
		t.Fatalf("read widgets helper module: %v", err)
	}
	widgetsModule := string(widgetsRaw)
	for _, want := range []string{
		"module lib.core.widgets",
		"struct Text:",
		"struct Button:",
		"struct TextBox:",
		"struct Row:",
		"struct Column:",
		"struct Panel:",
		"func add_button(",
		"func add_textbox(",
		"func add_text(",
		"func add_row(",
		"func add_column(",
		"func add_panel(",
	} {
		if !strings.Contains(widgetsModule, want) {
			t.Fatalf("lib/core/widgets/widgets.tetra must expose minimal toolkit API %q", want)
		}
	}

	appRaw, err := os.ReadFile(
		filepath.Join(root, "examples", "surface", "toolkit", "surface_toolkit_form.tetra"),
	)
	if err != nil {
		t.Fatalf("read Surface toolkit form example: %v", err)
	}
	app := string(appRaw)
	for _, want := range []string{
		"import lib.core.widgets as widgets",
		"widgets.add_panel(",
		"widgets.add_column(",
		"widgets.add_text(",
		"widgets.add_textbox(",
		"widgets.add_row(",
		"widgets.add_button(",
		"widgets.textbox_text_input(",
		"widgets.button_key_event(",
		"component.tree_validate(",
		"component.tree_build_dispatch_path(",
	} {
		if !strings.Contains(app, want) {
			t.Fatalf("surface_toolkit_form.tetra must use toolkit/helper API %q", want)
		}
	}
	for _, forbidden := range []string{
		"struct Button:",
		"struct TextBox:",
		"struct Text:",
		"struct Row:",
		"struct Column:",
		"struct Panel:",
		".first_child =",
		".child_count =",
		".child_index =",
		".parent_id =",
		".id =",
		"component.TreeNode(id:",
		"tetra.ui.v1",
		".ui.html",
		".ui.web.mjs",
		".ui.json",
		"React",
		"user JS",
	} {
		if strings.Contains(app, forbidden) {
			t.Fatalf(
				"surface_toolkit_form.tetra must not contain demo/fake toolkit marker %q",
				forbidden,
			)
		}
	}
}

func TestSurfaceToolkitSettingsUsesReusableWidgetModule(t *testing.T) {
	root := repoRoot(t)
	appRaw, err := os.ReadFile(
		filepath.Join(root, "examples", "surface", "toolkit", "surface_toolkit_settings.tetra"),
	)
	if err != nil {
		t.Fatalf("read Surface toolkit settings example: %v", err)
	}
	app := string(appRaw)
	for _, want := range []string{
		"import lib.core.widgets as widgets",
		"import lib.core.component as component",
		"widgets.add_panel(",
		"widgets.add_column(",
		"widgets.add_text(",
		"widgets.add_textbox(",
		"widgets.add_row(",
		"widgets.add_button(",
		"widgets.hit_test(",
		"widgets.textbox_text_input(",
		"widgets.button_key_event(",
		"component.tree_validate(",
		"component.tree_build_dispatch_path(",
	} {
		if !strings.Contains(app, want) {
			t.Fatalf("surface_toolkit_settings.tetra must use toolkit/helper API %q", want)
		}
	}
	for _, want := range []string{
		"struct ToolkitSettingsApp:",
		"NameTextBox",
		"EmailTextBox",
		"SaveButton",
		"ResetButton",
		"StatusText",
	} {
		if !strings.Contains(app, want) {
			t.Fatalf("surface_toolkit_settings.tetra missing reuse fixture marker %q", want)
		}
	}
	for _, forbidden := range []string{
		"struct Button:",
		"struct TextBox:",
		"struct Text:",
		"struct Row:",
		"struct Column:",
		"struct Panel:",
		".first_child =",
		".child_count =",
		".child_index =",
		".parent_id =",
		".id =",
		"tree.nodes[",
		"component.TreeNode(id:",
		"tetra.ui.v1",
		".ui.html",
		".ui.web.mjs",
		".ui.json",
		"React",
		"user JS",
	} {
		if strings.Contains(app, forbidden) {
			t.Fatalf(
				"surface_toolkit_settings.tetra must not contain demo/fake toolkit marker %q",
				forbidden,
			)
		}
	}
	if strings.Count(app, "widgets.add_textbox(") < 2 {
		t.Fatalf(
			("surface_toolkit_settings.tetra must construct at least two " +
				"TextBoxes through widgets.add_textbox"),
		)
	}
}

func TestSurfaceAccessibilitySettingsUsesMetadataTreeHelpers(t *testing.T) {
	root := repoRoot(t)
	accessibilityRaw, err := os.ReadFile(
		filepath.Join(root, "lib", "core", "surface", "accessibility.tetra"),
	)
	if err != nil {
		t.Fatalf("read accessibility helper module: %v", err)
	}
	accessibilityModule := string(accessibilityRaw)
	for _, want := range []string{
		"module lib.core.accessibility",
		"struct NodeMetadata:",
		"struct Snapshot:",
		"func role_textbox()",
		"func role_button()",
		"func textbox_metadata(",
		"func button_metadata(",
		"func validate_settings_counts(",
	} {
		if !strings.Contains(accessibilityModule, want) {
			t.Fatalf("lib/core/surface/accessibility.tetra must expose metadata helper %q", want)
		}
	}

	widgetsRaw, err := os.ReadFile(filepath.Join(root, "lib", "core", "widgets", "widgets.tetra"))
	if err != nil {
		t.Fatalf("read widgets helper module: %v", err)
	}
	widgetsModule := string(widgetsRaw)
	for _, want := range []string{
		"import lib.core.accessibility as accessibility",
		"func add_accessible_textbox(",
		"func add_accessible_button(",
		"func add_accessible_status(",
		"func hit_test_accessibility_settings(",
	} {
		if !strings.Contains(widgetsModule, want) {
			t.Fatalf("lib/core/widgets/widgets.tetra must expose accessibility helper %q", want)
		}
	}

	appRaw, err := os.ReadFile(
		filepath.Join(
			root,
			"examples",
			"surface",
			"toolkit",
			"surface_accessibility_settings.tetra",
		),
	)
	if err != nil {
		t.Fatalf("read Surface accessibility settings example: %v", err)
	}
	app := string(appRaw)
	for _, want := range []string{
		"import lib.core.accessibility as accessibility",
		"struct AccessibilitySettingsApp:",
		"widgets.add_accessible_textbox(",
		"widgets.add_accessible_button(",
		"widgets.add_accessible_status(",
		"accessibility.validate_settings_counts(",
		"component.tree_build_dispatch_path(",
		"component.tree_build_draw_order(",
		"surface.open(\"Surface Accessibility Settings\"",
	} {
		if !strings.Contains(app, want) {
			t.Fatalf(
				"surface_accessibility_settings.tetra must use accessibility metadata helper %q",
				want,
			)
		}
	}
	for _, forbidden := range []string{
		"tetra.ui.v1",
		".ui.html",
		".ui.web.mjs",
		".ui.json",
		"React",
		"DOM",
		"ARIA",
		"screen reader",
		"platform accessibility host",
		"user JS",
	} {
		if strings.Contains(app, forbidden) {
			t.Fatalf(
				("surface_accessibility_settings.tetra must not contain " +
					"production accessibility/legacy marker %q"),
				forbidden,
			)
		}
	}
}

func TestReleaseSurfaceScriptsNormalizeReportDirBeforeHashValidation(t *testing.T) {
	root := repoRoot(t)
	for _, rel := range []string{
		filepath.Join("scripts", "release", "surface", "gate.sh"),
		filepath.Join("scripts", "release", "surface", "surface-headless-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-linux-x64-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-linux-x64-real-window-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-wasm32-web-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-wasm32-web-browser-canvas-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-headless-text-focus-input-smoke.sh"),
		filepath.Join(
			"scripts",
			"release",
			"surface",
			"surface-linux-x64-real-window-text-focus-input-smoke.sh",
		),
		filepath.Join(
			"scripts",
			"release",
			"surface",
			"surface-wasm32-web-browser-canvas-text-focus-input-smoke.sh",
		),
		filepath.Join("scripts", "release", "surface", "surface-headless-component-tree-smoke.sh"),
		filepath.Join(
			"scripts",
			"release",
			"surface",
			"surface-linux-x64-real-window-component-tree-smoke.sh",
		),
		filepath.Join(
			"scripts",
			"release",
			"surface",
			"surface-wasm32-web-browser-canvas-component-tree-smoke.sh",
		),
		filepath.Join("scripts", "release", "surface", "surface-headless-component-tree-api-smoke.sh"),
		filepath.Join(
			"scripts",
			"release",
			"surface",
			"surface-linux-x64-real-window-component-tree-api-smoke.sh",
		),
		filepath.Join(
			"scripts",
			"release",
			"surface",
			"surface-wasm32-web-browser-canvas-component-tree-api-smoke.sh",
		),
		filepath.Join("scripts", "release", "surface", "surface-headless-block-system-smoke.sh"),
		filepath.Join(
			"scripts",
			"release",
			"surface",
			"surface-linux-x64-real-window-block-system-smoke.sh",
		),
		filepath.Join(
			"scripts",
			"release",
			"surface",
			"surface-wasm32-web-browser-canvas-block-system-smoke.sh",
		),
		filepath.Join("scripts", "release", "surface", "surface-headless-minimal-toolkit-smoke.sh"),
		filepath.Join(
			"scripts",
			"release",
			"surface",
			"surface-linux-x64-real-window-minimal-toolkit-smoke.sh",
		),
		filepath.Join(
			"scripts",
			"release",
			"surface",
			"surface-wasm32-web-browser-canvas-minimal-toolkit-smoke.sh",
		),
		filepath.Join("scripts", "release", "surface", "surface-headless-toolkit-reuse-smoke.sh"),
		filepath.Join(
			"scripts",
			"release",
			"surface",
			"surface-linux-x64-real-window-toolkit-reuse-smoke.sh",
		),
		filepath.Join(
			"scripts",
			"release",
			"surface",
			"surface-wasm32-web-browser-canvas-toolkit-reuse-smoke.sh",
		),
		filepath.Join(
			"scripts",
			"release",
			"surface",
			"surface-headless-accessibility-metadata-smoke.sh",
		),
		filepath.Join(
			"scripts",
			"release",
			"surface",
			"surface-linux-x64-real-window-accessibility-metadata-smoke.sh",
		),
		filepath.Join(
			"scripts",
			"release",
			"surface",
			"surface-wasm32-web-browser-canvas-accessibility-metadata-smoke.sh",
		),
	} {
		rel := rel
		t.Run(filepath.ToSlash(rel), func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(root, rel))
			if err != nil {
				t.Fatalf("read %s: %v", rel, err)
			}
			text := string(raw)
			want := `report_dir="$(cd "$report_dir" && pwd)"`
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing report_dir absolute normalization %q", rel, want)
			}
			mkdirIdx := strings.Index(text, `mkdir -p "$report_dir"`)
			normalizeIdx := strings.Index(text, want)
			if mkdirIdx < 0 || normalizeIdx < 0 || normalizeIdx < mkdirIdx {
				t.Fatalf("%s must create report_dir before normalizing it", rel)
			}
		})
	}
}
