package compiler_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/testkit"
)

func surfaceBlockBeautyExamplePaths() []string {
	return []string{
		"examples/surface_block_command_palette.tetra",
		"examples/surface_block_project_dashboard.tetra",
		"examples/surface_block_settings.tetra",
		"examples/surface_block_editor_shell.tetra",
		"examples/surface_block_glass_panel.tetra",
	}
}

type surfaceProductionExample struct {
	path  string
	shape string
}

func surfaceProductionExampleShapes() []surfaceProductionExample {
	return []surfaceProductionExample{
		{path: "examples/surface_prod_command_palette.tetra", shape: "command_palette"},
		{path: "examples/surface_prod_settings_app.tetra", shape: "settings"},
		{path: "examples/surface_prod_project_dashboard.tetra", shape: "project_dashboard"},
		{path: "examples/surface_prod_editor_shell.tetra", shape: "editor_shell"},
		{path: "examples/surface_prod_file_manager_shell.tetra", shape: "file_manager_shell"},
		{path: "examples/surface_prod_multi_window_notes.tetra", shape: "multi_window_notes"},
		{path: "examples/surface_prod_system_tray_status.tetra", shape: "system_tray_status"},
		{path: "examples/surface_prod_notification_dialog.tetra", shape: "notification_dialog"},
		{path: "examples/surface_prod_localized_form.tetra", shape: "localized_form"},
		{path: "examples/surface_prod_accessibility_heavy_form.tetra", shape: "accessibility_heavy_form"},
	}
}

func TestSurfaceCounterExampleLoadsCoreSurfaceAndDrawModules(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_counter.tetra")
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
	}

	for _, module := range []string{"lib.core.surface", "lib.core.draw"} {
		if _, ok := world.ByModule[module]; !ok {
			t.Fatalf("surface counter did not load module %s; modules=%v", module, world.ByModule)
		}
	}

	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface counter): %v", err)
	}
}

func TestSurfaceComponentCounterExampleLoadsStaticComponentAbilities(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_component_counter.tetra")
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
	}

	for _, module := range []string{"lib.core.surface", "lib.core.draw", "lib.core.component"} {
		if _, ok := world.ByModule[module]; !ok {
			t.Fatalf("surface component counter did not load module %s; modules=%v", module, world.ByModule)
		}
	}

	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface component counter): %v", err)
	}
}

func TestSurfaceTextInputExampleLoadsTextBoxComponent(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_text_input.tetra")
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
	}

	for _, module := range []string{"lib.core.surface", "lib.core.draw", "lib.core.component"} {
		if _, ok := world.ByModule[module]; !ok {
			t.Fatalf("surface text input did not load module %s; modules=%v", module, world.ByModule)
		}
	}

	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface text input): %v", err)
	}
}

func TestSurfaceTextModuleDefinesProductionTextBufferAPI(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.text as text

func main() -> Int
uses alloc, mem:
    var storage: []u8 = core.make_u8(32)
    var buf: text.TextBuffer = text.buffer_init(storage)
    var cursor: text.TextCursor = text.caret_start(buf)
    var selection: text.TextSelection = text.clear_selection()
    var composition: text.TextComposition = text.composition_clear()
    var bytes: []u8 = core.make_u8(2)
    bytes[0] = 79
    bytes[1] = 75

    let was_empty: Bool = text.is_empty(buf)
    let inserted_bytes: text.TextEditResult = text.insert_bytes(buf, cursor, bytes)
    let inserted_string: text.TextEditResult = text.insert_string(buf, cursor, "!")
    let moved_left: text.TextCursor = text.move_left(buf, cursor)
    let moved_right: text.TextCursor = text.move_right(buf, moved_left)
    let home: text.TextCursor = text.move_home(buf, moved_right)
    var end: text.TextCursor = text.move_end(buf, home)
    selection = text.select_range(buf, 0, text.len_bytes(buf))
    let replaced: text.TextEditResult = text.replace_selection(buf, end, selection, bytes)
    let backed: text.TextEditResult = text.backspace(buf, end, selection)
    let deleted: text.TextEditResult = text.delete(buf, end, selection)
    let cleared: text.TextEditResult = text.buffer_clear(buf)
    let composition_clear: Bool = !composition.active && composition.start == 0 && composition.len == 0 && composition.preview_len == 0

    if was_empty && inserted_bytes.ok && inserted_string.ok && replaced.ok && backed.error >= 0 && deleted.error >= 0 && cleared.ok && composition_clear:
        return text.error_none() + text.error_capacity() + text.error_invalid_utf8() + text.error_invalid_range() + cursor.byte_index - cursor.byte_index
    return 1
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt: %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(lib.core.text consumer): %v", err)
	}
}

func TestSurfaceI18nModuleDefinesLocalizationHooks(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.i18n as i18n

func main() -> Int:
    let en: i18n.Locale = i18n.locale_en_us()
    let es: i18n.Locale = i18n.locale_es_es()
    let ar: i18n.Locale = i18n.locale_ar_eg()
    let title: i18n.StringID = i18n.string_id(1, 10)
    let resource: i18n.LocaleResource = i18n.locale_resource(ar, 5, 4103, true, true)
    let formats: i18n.FormatPolicy = i18n.format_policy(true, true, true, false)
    let policy: i18n.LocalizationPolicy = i18n.localization_policy(3, true, true, false, false, i18n.direction_rtl(), false)
    let text_len: Int = i18n.localized_text_len(resource, title, 12)
    if i18n.locale_valid(en) &&
       i18n.locale_valid(es) &&
       i18n.locale_valid(ar) &&
       i18n.string_id_valid(title) &&
       i18n.locale_resource_valid(resource) &&
       i18n.format_policy_valid(formats) &&
       i18n.localization_policy_valid(policy) &&
       text_len > 0:
        return i18n.direction_ltr() - i18n.direction_ltr()
    return 1
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt: %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(lib.core.i18n consumer): %v", err)
	}
}

func TestSurfaceProductionToolkitDefinesStyleAndWidgetAPI(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface
import lib.core.widgets as widgets
import lib.core.style as style

func main() -> Int
uses alloc, mem:
    let rect: surface.Rect = surface.Rect(x: 0, y: 0, w: 320, h: 240)
    let min_size: surface.Size = surface.Size(w: 16, h: 16)
    let max_size: surface.Size = surface.Size(w: 640, h: 480)
    let theme: style.Theme = style.default_theme()
    let normal_colors: style.WidgetColors = style.style_for_state(theme.button, style.state_normal())
    let focused_colors: style.WidgetColors = style.style_for_state(theme.textbox, style.state_focused())
    let state_sum: Int = style.state_normal() + style.state_focused() + style.state_hovered() + style.state_pressed() + style.state_disabled() + style.state_error()

    var text: widgets.Text = widgets.Text(rect: rect, role: widgets.role_label(), text_len: 0, status_code: 0)
    var label: widgets.Label = widgets.Label(rect: rect, role: widgets.role_label(), text_len: 0, labelled_for: 7)
    var status: widgets.StatusText = widgets.StatusText(rect: rect, role: widgets.role_status(), text_len: 0, status_code: 0)
    var button: widgets.Button = widgets.Button(rect: rect, focused: false, press_count: 0, action: widgets.action_save())
    var storage: []u8 = core.make_u8(32)
    var box: widgets.TextBox = widgets.TextBox(rect: rect, focused: false, text_len: 0, caret: 0, buffer: storage)
    var checkbox: widgets.Checkbox = widgets.Checkbox(rect: rect, focused: false, checked: false, toggle_count: 0, action: widgets.action_changed())
    var row: widgets.Row = widgets.Row(rect: rect, gap: 0)
    var column: widgets.Column = widgets.Column(rect: rect, padding: 0, gap: 0)
    var panel: widgets.Panel = widgets.Panel(rect: rect, padding: 0)
    var stack: widgets.Stack = widgets.Stack(rect: rect, child_count: 0, gap: 0)
    var scroll: widgets.Scroll = widgets.Scroll(rect: rect, content_size: max_size, offset_y: 0)
    var spacer: widgets.Spacer = widgets.Spacer(rect: rect, min_size: min_size)

    let text_ok: Int = widgets.text_init(text, rect, widgets.role_label(), 5)
    let label_ok: Int = widgets.label_init(label, rect, 4, 7)
    let status_ok: Int = widgets.status_text_init(status, rect, 6, 2)
    let button_ok: Int = widgets.button_init(button, rect, widgets.action_save())
    let box_ok: Int = widgets.textbox_init(box, rect, storage)
    let checkbox_ok: Int = widgets.checkbox_init(checkbox, rect, false)
    let toggled: Int = widgets.checkbox_toggle(checkbox)
    let row_ok: Int = widgets.row_init(row, rect, 8)
    let column_ok: Int = widgets.column_init(column, rect, 12, 8)
    let panel_ok: Int = widgets.panel_init(panel, rect, 12)
    let stack_ok: Int = widgets.stack_init(stack, rect, 3, 8)
    let scroll_ok: Int = widgets.scroll_init(scroll, rect, max_size)
    let scrolled: Int = widgets.scroll_set_offset(scroll, 16)
    let spacer_ok: Int = widgets.spacer_init(spacer, rect, min_size)

    if checkbox.checked && toggled == 1 && scrolled == 16 && normal_colors.fg.a == 255 && focused_colors.border.a == 255:
        return text_ok + label_ok + status_ok + button_ok + box_ok + checkbox_ok + row_ok + column_ok + panel_ok + stack_ok + scroll_ok + spacer_ok + theme.padding.left + theme.spacing.x + state_sum - state_sum
    return 1
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt: %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(lib.core.style/widgets production toolkit consumer): %v", err)
	}
}

func TestSurfaceWidgetMigrationMapsCompatibilityLayerToBlockMorphRecipes(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface
import lib.core.component as component
import lib.core.widgets as widgets
import lib.core.block as block
import lib.core.morph as morph

func main() -> Int:
    let rect: surface.Rect = surface.Rect(x: 0, y: 0, w: 320, h: 200)
    let panel: widgets.WidgetMigration = widgets.migration_panel()
    let button: widgets.WidgetMigration = widgets.migration_button()
    let textbox: widgets.WidgetMigration = widgets.migration_textbox()
    let status: widgets.WidgetMigration = widgets.migration_status_text()
    let mapped_block: block.Block = widgets.migration_expand_button_to_block(7, 1, rect, 6)
    let expansion: morph.RecipeExpansion = morph.recipe_expansion(morph.recipe_control_action(), block.id(7))

    if widgets.widgets_are_compatibility_layer() &&
       !widgets.widgets_are_core_final_architecture() &&
       widgets.migration_valid(panel) &&
       widgets.migration_valid(button) &&
       widgets.migration_valid(textbox) &&
       widgets.migration_valid(status) &&
       panel.component_kind == component.kind_panel() &&
       panel.block_layout_mode == block.layout_mode_column() &&
       button.recipe_hash == morph.recipe_control_action_hash() &&
       textbox.recipe_hash == morph.recipe_field_text_hash() &&
       status.recipe_hash == morph.recipe_status_message_hash() &&
       block.id_value(mapped_block.id) == 7 &&
       morph.expansion_valid(expansion) &&
       widgets.migration_diagnostic_for_core_final_claim() == widgets.migration_error_core_final_architecture_claim():
        return widgets.migration_level() - widgets.migration_level()
    return 1
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt: %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(lib.core.widgets migration consumer): %v", err)
	}
}

func TestSurfaceMigrationWidgetsToBlockExampleLoads(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_migration_widgets_to_block.tetra")
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
	}

	for _, module := range []string{"lib.core.surface", "lib.core.widgets", "lib.core.block", "lib.core.morph"} {
		if _, ok := world.ByModule[module]; !ok {
			t.Fatalf("surface migration widgets-to-block example did not load module %s; modules=%v", module, world.ByModule)
		}
	}

	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface migration widgets-to-block): %v", err)
	}
}

func TestSurfaceMorphDefinesProductionRecipeAuthoringAPI(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.morph as morph

func main() -> Int:
    let action: morph.Recipe = morph.recipe_control_action()
    let field: morph.Recipe = morph.recipe_field_text()
    let toggle: morph.Recipe = morph.recipe_control_toggle()
    let command: morph.Recipe = morph.recipe_command_item()
    let nav: morph.Recipe = morph.recipe_navigation_item()
    let panel: morph.Recipe = morph.recipe_region_panel()
    let dialog: morph.Recipe = morph.recipe_overlay_dialog()
    let tabs: morph.Recipe = morph.recipe_navigation_tabs()
    let list: morph.Recipe = morph.recipe_collection_list()
    let table: morph.Recipe = morph.recipe_collection_table_lite()
    let status: morph.Recipe = morph.recipe_status_message()
    let contract: morph.AuthoringContract = morph.authoring_contract_default()

    if morph.recipe_expands_to_block(action) &&
       morph.recipe_expands_to_block(field) &&
       morph.recipe_expands_to_block(toggle) &&
       morph.recipe_expands_to_block(command) &&
       morph.recipe_expands_to_block(nav) &&
       morph.recipe_expands_to_block(panel) &&
       morph.recipe_expands_to_block(dialog) &&
       morph.recipe_expands_to_block(tabs) &&
       morph.recipe_expands_to_block(list) &&
       morph.recipe_expands_to_block(table) &&
       morph.recipe_expands_to_block(status) &&
       morph.authoring_contract_valid(contract):
        return morph.production_recipe_authoring_v1() - morph.production_recipe_authoring_v1()
    return 1
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt: %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(lib.core.morph production recipe authoring consumer): %v", err)
	}
}

func TestSurfaceBlockMinimalExampleLoadsBlockModel(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_minimal.tetra")
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
	}

	for _, module := range []string{"lib.core.surface", "lib.core.block"} {
		if _, ok := world.ByModule[module]; !ok {
			t.Fatalf("surface block minimal did not load module %s; modules=%v", module, world.ByModule)
		}
	}

	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface block minimal): %v", err)
	}
}

func TestSurfaceI18nDashboardExampleLoadsLocalizationModel(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_i18n_dashboard.tetra")
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
	}

	for _, module := range []string{"lib.core.surface", "lib.core.block", "lib.core.i18n"} {
		if _, ok := world.ByModule[module]; !ok {
			t.Fatalf("surface i18n dashboard did not load module %s; modules=%v", module, world.ByModule)
		}
	}

	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface i18n dashboard): %v", err)
	}
}

func TestSurfaceBlockModuleDefinesCorePropertySchema(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface
import lib.core.block as block

func main() -> Int:
    let rect: surface.Rect = surface.Rect(x: 0, y: 0, w: 320, h: 200)
    let id: block.BlockID = block.id(7)
    let layout: block.LayoutSpec = block.layout_fixed(rect)
    let fill: block.PaintLayer = block.paint_layer_fill(surface.Color(r: 32, g: 48, b: 64, a: 255))
    let paint: block.PaintSpec = block.paint_from_layer(fill)
    let text: block.TextSpec = block.text_label(12, surface.Color(r: 240, g: 244, b: 248, a: 255))
    let image: block.ImageSpec = block.image_none()
    let input: block.InputSpec = block.input_clickable()
    let event: block.EventSpec = block.event_click(block.action_primary())
    let state: block.StateSpec = block.state_interactive()
    let motion: block.MotionSpec = block.motion_fast()
    let a11y: block.AccessibilitySpec = block.accessibility_button(12)
    let asset: block.AssetRef = block.asset_none()
    let props: block.BlockProps = block.props(layout, paint, text, image, input, event, state, motion, a11y, asset)
    let root: block.Block = block.make(id, block.id_none(), props)

    if block.id_value(root.id) == 7 && root.props.paint_layers == 1 && root.props.text_len == 12 && root.props.interaction_flags > 0 && root.props.motion_ms > 0:
        return 0
    return 1
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt: %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(lib.core.block consumer): %v", err)
	}
}

func TestSurfaceProductionExamplesLoadAndRun(t *testing.T) {
	for _, example := range surfaceProductionExampleShapes() {
		example := example
		t.Run(filepath.Base(example.path), func(t *testing.T) {
			entry := testkit.RepoPath(t, strings.Split(example.path, "/")...)
			raw, err := os.ReadFile(entry)
			if err != nil {
				t.Fatalf("read %s: %v", example.path, err)
			}
			text := string(raw)
			for _, want := range []string{
				"import lib.core.surface as surface",
				"import lib.core.block as block",
				"import lib.core.morph as morph",
				"production_example_shape",
				example.shape,
				"block.tree_validate",
				"morph.recipe_",
				"block.accessibility_",
				"block.state_",
				"block.event_",
				"block.motion_",
				"performance_budget_ok",
			} {
				if !strings.Contains(text, want) {
					t.Fatalf("%s missing production example marker %q", example.path, want)
				}
			}
			lower := strings.ToLower(text)
			for _, forbidden := range []string{
				"import lib.core.widgets",
				"widgets.",
				"react",
				"electron",
				"dom runtime",
				"platform widget",
				"external css",
			} {
				if strings.Contains(lower, forbidden) {
					t.Fatalf("%s contains forbidden production example dependency marker %q", example.path, forbidden)
				}
			}

			world, err := compiler.LoadWorld(entry)
			if err != nil {
				t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
			}
			if _, err := compiler.CheckWorld(world); err != nil {
				t.Fatalf("CheckWorld(%s): %v", filepath.ToSlash(entry), err)
			}

			out := filepath.Join(t.TempDir(), strings.TrimSuffix(filepath.Base(example.path), ".tetra"))
			if _, err := compiler.BuildFileWithStatsOpt(entry, out, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
				t.Fatalf("BuildFileWithStatsOpt(%s): %v", example.path, err)
			}
			output, err := exec.Command(out).CombinedOutput()
			if err != nil {
				t.Fatalf("%s exited with %v\n%s", example.path, err, output)
			}
		})
	}
}

func TestSurfaceBlockTreeExampleLoadsBlockGraphModel(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_tree.tetra")
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
	}

	for _, module := range []string{"lib.core.surface", "lib.core.block"} {
		if _, ok := world.ByModule[module]; !ok {
			t.Fatalf("surface block tree did not load module %s; modules=%v", module, world.ByModule)
		}
	}

	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface block tree): %v", err)
	}
}

func TestSurfaceBlockTreeExampleRunsGraphValidation(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_tree.tetra")
	out := filepath.Join(t.TempDir(), "surface-block-tree")
	if _, err := compiler.BuildFileWithStatsOpt(entry, out, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt(surface block tree): %v", err)
	}
	cmd := exec.Command(out)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("surface block tree exited with %v\n%s", err, output)
	}
}

func TestSurfaceBlockPaintLayersExampleLoadsPaintModel(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_paint_layers.tetra")
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
	}

	for _, module := range []string{"lib.core.surface", "lib.core.draw", "lib.core.block"} {
		if _, ok := world.ByModule[module]; !ok {
			t.Fatalf("surface block paint layers did not load module %s; modules=%v", module, world.ByModule)
		}
	}

	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface block paint layers): %v", err)
	}
}

func TestSurfaceBlockPaintLayersExampleRunsPaintValidation(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_paint_layers.tetra")
	out := filepath.Join(t.TempDir(), "surface-block-paint-layers")
	if _, err := compiler.BuildFileWithStatsOpt(entry, out, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt(surface block paint layers): %v", err)
	}
	cmd := exec.Command(out)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("surface block paint layers exited with %v\n%s", err, output)
	}
}

func TestSurfaceBlockTextExampleLoadsTextModel(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_text.tetra")
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
	}

	for _, module := range []string{"lib.core.surface", "lib.core.draw", "lib.core.block", "lib.core.text"} {
		if _, ok := world.ByModule[module]; !ok {
			t.Fatalf("surface block text did not load module %s; modules=%v", module, world.ByModule)
		}
	}

	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface block text): %v", err)
	}
}

func TestSurfaceBlockTextExampleRunsTextValidation(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_text.tetra")
	out := filepath.Join(t.TempDir(), "surface-block-text")
	if _, err := compiler.BuildFileWithStatsOpt(entry, out, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt(surface block text): %v", err)
	}
	cmd := exec.Command(out)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("surface block text exited with %v\n%s", err, output)
	}
}

func TestSurfaceBlockInputExampleLoadsEditableTextModel(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_input.tetra")
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
	}

	for _, module := range []string{"lib.core.surface", "lib.core.block", "lib.core.text"} {
		if _, ok := world.ByModule[module]; !ok {
			t.Fatalf("surface block input did not load module %s; modules=%v", module, world.ByModule)
		}
	}

	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface block input): %v", err)
	}
}

func TestSurfaceBlockTextSpecDefinesMeasurementFallbackAndCacheAPI(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface
import lib.core.draw as draw
import lib.core.block as block
import lib.core.text as text

func main() -> Int
uses alloc, mem:
    let fg: surface.Color = surface.Color(r: 238, g: 242, b: 247, a: 255)
    let spec: block.TextSpec = block.text_styled(28, fg, block.text_family_ui(), 16, 600, 20, block.text_align_start(), block.text_wrap_word(), block.text_overflow_ellipsis(), 255)
    let editable: block.TextSpec = block.text_editable_styled(4, 8, fg, block.text_family_ui(), 14, 400, 18, block.text_align_start(), block.text_wrap_none(), block.text_overflow_clip(), 255)
    let measured: surface.Size = block.text_measure(spec, 96)
    let lines: Int = block.text_wrap_line_count(spec, 96)
    let ellipsis_len: Int = block.text_ellipsized_len(spec, 96)
    let fallback_len: Int = block.text_font_fallback_chain_len(spec)
    let cache_status: Int = block.text_glyph_cache_validate(block.text_glyph_cache_budget_bytes(), 4096, 12)
    var pixels: []u8 = core.make_u8(128 * 64 * 4)
    let surface_ref: surface.Surface = surface.Surface(handle: 0, width: 128, height: 64)
    var frame: surface.Frame = surface.Frame(surface: surface_ref, width: 128, height: 64, stride: 128 * 4, pixels: pixels)
    var ctx: draw.DrawContext = draw.DrawContext(frame: frame)
    let render: Int = draw.text_glyph_run(ctx, surface.Rect(x: 8, y: 8, w: measured.w, h: measured.h), spec.text_len, block.text_char_width(spec), block.text_effective_line_height(spec), fg)
    var storage: []u8 = core.make_u8(16)
    var input: text.EditableText = text.editable_empty(storage, 6)

    if measured.w == 96 && measured.h >= 20 && lines >= 2 &&
       ellipsis_len > 0 && ellipsis_len < spec.text_len &&
       fallback_len >= 2 && cache_status == block.text_cache_error_ok() &&
       block.text_is_editable(editable) && text.editable_len(input) == 0 &&
       render == draw.text_command_render():
        return 0
    return 1
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt: %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(lib.core.block text consumer): %v", err)
	}
}

func TestSurfaceBlockLayoutExampleLoadsLayoutModel(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_layout.tetra")
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
	}

	for _, module := range []string{"lib.core.surface", "lib.core.block"} {
		if _, ok := world.ByModule[module]; !ok {
			t.Fatalf("surface block layout did not load module %s; modules=%v", module, world.ByModule)
		}
	}

	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface block layout): %v", err)
	}
}

func TestSurfaceBlockLayoutExampleRunsLayoutValidation(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_layout.tetra")
	out := filepath.Join(t.TempDir(), "surface-block-layout")
	if _, err := compiler.BuildFileWithStatsOpt(entry, out, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt(surface block layout): %v", err)
	}
	cmd := exec.Command(out)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("surface block layout exited with 0, want Surface smoke success exit 1\n%s", output)
	}
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 1 {
		t.Fatalf("surface block layout exited with %v, want exit 1\n%s", err, output)
	}
}

func TestSurfaceBlockLayoutSpecDefinesConstraintsAPI(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface
import lib.core.block as block

func main() -> Int:
    let root_rect: surface.Rect = surface.Rect(x: 0, y: 0, w: 320, h: 200)
    let child_rect: surface.Rect = surface.Rect(x: 0, y: 0, w: 0, h: 40)
    let root: block.LayoutSpec = block.layout_constrained(block.layout_mode_column(), root_rect, block.layout_size_fixed(), block.layout_size_fixed(), 240, 160, 480, 260, 12, 8, block.layout_align_stretch(), block.layout_justify_start(), block.layout_overflow_clip(), 0)
    let child: block.LayoutSpec = block.layout_constrained(block.layout_mode_row(), child_rect, block.layout_size_fill(), block.layout_size_fixed(), 80, 32, 280, 64, 4, 6, block.layout_align_center(), block.layout_justify_space_between(), block.layout_overflow_visible(), 1)
    let measured: surface.Size = surface.Size(w: 96, h: 20)
    let resolved: surface.Rect = block.layout_resolve_child(root, child, 0, 2, measured)
    let grid: block.LayoutSpec = block.layout_grid(root_rect, 2, 2, 6)
    let grid_cell: surface.Rect = block.layout_resolve_child(root, grid, 1, 4, measured)
    let dock: block.LayoutSpec = block.layout_dock(root_rect, block.layout_dock_top(), 32)
    let docked: surface.Rect = block.layout_resolve_child(root, dock, 0, 1, measured)
    let scroll: block.LayoutSpec = block.layout_scroll(surface.Rect(x: 0, y: 0, w: 120, h: 80), 120, 180, 32)
    let max_offset: Int = block.layout_scroll_max_offset(scroll)
    let resized: block.LayoutSpec = block.layout_resize(root, 480, 260)

    if resolved.w > 0 && grid_cell.w > 0 && docked.h == 32 && max_offset == 100 && resized.w == 480 && block.layout_validate_constraints(child) == block.layout_error_ok():
        return 0
    return 1
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt: %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(lib.core.block layout consumer): %v", err)
	}
}

func TestSurfaceBlockEventsExampleLoadsEventFocusModel(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_events.tetra")
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
	}

	for _, module := range []string{"lib.core.surface", "lib.core.block"} {
		if _, ok := world.ByModule[module]; !ok {
			t.Fatalf("surface block events did not load module %s; modules=%v", module, world.ByModule)
		}
	}

	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface block events): %v", err)
	}
}

func TestSurfaceBlockEventsExampleRunsEventFocusValidation(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_events.tetra")
	out := filepath.Join(t.TempDir(), "surface-block-events")
	if _, err := compiler.BuildFileWithStatsOpt(entry, out, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt(surface block events): %v", err)
	}
	cmd := exec.Command(out)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("surface block events exited with 0, want Surface smoke success exit 1\n%s", output)
	}
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 1 {
		t.Fatalf("surface block events exited with %v, want exit 1\n%s", err, output)
	}
}

func TestSurfaceBlockEventFocusDefinesRoutingAPI(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface
import lib.core.block as block

func main() -> Int
uses alloc, mem:
    let root_rect: surface.Rect = surface.Rect(x: 0, y: 0, w: 320, h: 200)
    let panel_rect: surface.Rect = surface.Rect(x: 16, y: 16, w: 288, h: 168)
    let input_rect: surface.Rect = surface.Rect(x: 24, y: 64, w: 120, h: 44)
    let disabled_rect: surface.Rect = surface.Rect(x: 152, y: 64, w: 120, h: 44)
    let action_rect: surface.Rect = surface.Rect(x: 24, y: 120, w: 120, h: 44)

    var tree: block.BlockTree = block.tree_init(8)
    let root: block.Block = block.make(block.id(1), block.id_none(), block.props_default(root_rect))
    let panel: block.Block = block.make(block.id(2), block.id(1), block.props_default(panel_rect))
    let input: block.Block = block.make(block.id(4), block.id(2), block.props(block.layout_fixed(input_rect), block.paint_empty(), block.text_label(4, surface.Color(r: 240, g: 244, b: 248, a: 255)), block.image_none(), block.input_text(), block.event_text(block.action_primary()), block.state_interactive(), block.motion_fast(), block.accessibility_text(4), block.asset_none()))
    let disabled: block.Block = block.make(block.id(5), block.id(2), block.props(block.layout_fixed(disabled_rect), block.paint_empty(), block.text_label(8, surface.Color(r: 160, g: 170, b: 180, a: 255)), block.image_none(), block.input_disabled(block.input_clickable()), block.event_click(block.action_primary()), block.state_interactive(), block.motion_none(), block.accessibility_button(8), block.asset_none()))
    let action: block.Block = block.make(block.id(6), block.id(2), block.props(block.layout_fixed(action_rect), block.paint_empty(), block.text_label(6, surface.Color(r: 240, g: 244, b: 248, a: 255)), block.image_none(), block.input_clickable(), block.event_click(block.action_secondary()), block.state_interactive(), block.motion_fast(), block.accessibility_button(6), block.asset_none()))

    let _root_id: Int = block.tree_add_root(tree, root, root_rect)
    let _panel_id: Int = block.tree_add_child(tree, block.id(1), panel, panel_rect)
    let _input_id: Int = block.tree_add_child(tree, block.id(2), input, input_rect)
    let _disabled_id: Int = block.tree_add_child(tree, block.id(2), disabled, disabled_rect)
    let _action_id: Int = block.tree_add_child(tree, block.id(2), action, action_rect)

    var hit_path: []i32 = core.make_i32(8)
    let hit_len: Int = block.event_hit_test_path(tree, 40, 80, hit_path)
    var dispatch_path: []i32 = core.make_i32(8)
    let dispatch_len: Int = block.event_build_dispatch_path(tree, block.id(4), dispatch_path)
    let focus0: Int = block.id_value(block.focus_order_at(tree, 0))
    let focus1: Int = block.id_value(block.focus_next(tree, block.id(4)))
    let focus_wrap: Int = block.id_value(block.focus_next(tree, block.id(6)))
    let disabled_status: Int = block.event_dispatch_status(block.input_disabled(block.input_clickable()), block.event_kind_click(), true)
    let unfocused_text: Int = block.event_dispatch_status(block.input_text(), block.event_kind_text(), false)
    let focused_text: Int = block.event_dispatch_status(block.input_text(), block.event_kind_text(), true)

    if block.event_policy_capture_bubble_direct() > 0 &&
       block.event_kind_pointer_enter() > 0 && block.event_kind_frame() > 0 &&
       hit_len == 3 && hit_path[0] == 1 && hit_path[1] == 2 && hit_path[2] == 4 &&
       dispatch_len == 3 && dispatch_path[0] == 1 && dispatch_path[1] == 2 && dispatch_path[2] == 4 &&
       focus0 == 4 && focus1 == 6 && focus_wrap == 4 &&
       disabled_status == block.event_route_rejected_disabled() &&
       unfocused_text == block.event_route_rejected_unfocused() &&
       focused_text == block.event_route_delivered():
        return 0
    return 1
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt(lib.core.block event/focus consumer): %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(lib.core.block event/focus consumer): %v", err)
	}
}

func TestSurfaceBlockStatesExampleLoadsSelectorResolver(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_states.tetra")
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
	}

	for _, module := range []string{"lib.core.surface", "lib.core.block"} {
		if _, ok := world.ByModule[module]; !ok {
			t.Fatalf("surface block states did not load module %s; modules=%v", module, world.ByModule)
		}
	}

	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface block states): %v", err)
	}
}

func TestSurfaceBlockStatesExampleRunsStateValidation(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_states.tetra")
	out := filepath.Join(t.TempDir(), "surface-block-states")
	if _, err := compiler.BuildFileWithStatsOpt(entry, out, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt(surface block states): %v", err)
	}
	cmd := exec.Command(out)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("surface block states exited with 0, want Surface smoke success exit 1\n%s", output)
	}
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 1 {
		t.Fatalf("surface block states exited with %v, want exit 1\n%s", err, output)
	}
}

func TestSurfaceBlockStateResolverDefinesGenericAPI(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.block as block

func main() -> Int:
    let active: block.StateSelector = block.state_selector(true, true, true, true, true, true, true)
    let hover: block.StateSelector = block.state_selector_hover()
    let pressed: block.StateSelector = block.state_selector_pressed()
    let disabled: block.StateSelector = block.state_selector_disabled()
    let base: block.StateSpec = block.state_variant(1)
    let pressed_spec: block.StateSpec = block.state_with_selector(pressed, 2)
    let disabled_spec: block.StateSpec = block.state_with_selector(disabled, 3)
    let resolved_pressed: block.StateSpec = block.state_resolve(base, pressed_spec, active)
    let resolved_disabled: block.StateSpec = block.state_resolve(resolved_pressed, disabled_spec, active)
    let resolved_fill: Int = block.state_resolve_int(10, 20, active, hover)
    let skipped_fill: Int = block.state_resolve_int(10, 30, block.state_selector_none(), hover)

    if block.state_flags(active) == 127 &&
       block.state_selector_matches(active, hover) &&
       block.state_selector_matches(active, pressed) &&
       !block.state_selector_matches(block.state_selector_none(), hover) &&
       block.state_resolver_order_base() == 1 &&
       block.state_resolver_order_variant() == 2 &&
       block.state_resolver_order_hover() == 3 &&
       block.state_resolver_order_pressed() == 4 &&
       block.state_resolver_order_focused() == 5 &&
       block.state_resolver_order_selected() == 6 &&
       block.state_resolver_order_disabled() == 7 &&
       block.state_resolver_order_error() == 8 &&
       block.state_resolver_order_loading() == 9 &&
       block.state_resolver_order_motion() == 10 &&
       resolved_pressed.variant == 2 &&
       resolved_disabled.variant == 3 &&
       resolved_disabled.disabled &&
       resolved_fill == 20 &&
       skipped_fill == 10:
        return 0
    return 1
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt(lib.core.block state resolver consumer): %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(lib.core.block state resolver consumer): %v", err)
	}
}

func TestSurfaceBlockStateEventsAppModelDefinesCommandAPI(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.block as block

func main() -> Int:
    let palette: block.AppStateStore = block.app_state_store(1, block.app_surface_command_palette(), block.app_state_scope_overlay(), 3, 10, 20, 2)
    let dashboard: block.AppStateStore = block.app_state_store(2, block.app_surface_dashboard(), block.app_state_scope_page(), 3, 30, 40, 1)
    let settings: block.AppStateStore = block.app_state_store(3, block.app_surface_settings(), block.app_state_scope_form(), 3, 50, 60, 1)
    let editor: block.AppStateStore = block.app_state_store(4, block.app_surface_editor_shell(), block.app_state_scope_document(), 4, 70, 80, 2)

    let open_cmd: block.AppCommand = block.app_command(block.app_command_kind_command(), block.app_source_key(), block.id(4), palette, 2, true, false)
    let save_cmd: block.AppCommand = block.app_command(block.app_command_kind_command(), block.app_source_click(), block.id(5), settings, 1, true, false)
    let text_cmd: block.AppCommand = block.app_command(block.app_command_kind_text_edit(), block.app_source_text(), block.id(4), editor, 3, true, false)
    let refresh_cmd: block.AppCommand = block.app_command(block.app_command_kind_async(), block.app_source_timer(), block.id(2), dashboard, 2, true, true)

    let disabled_status: Int = block.app_command_dispatch_status(save_cmd, block.input_disabled(block.input_clickable()), block.event_kind_click(), true)
    let unfocused_status: Int = block.app_command_dispatch_status(text_cmd, block.input_text(), block.event_kind_text(), false)
    let focused_status: Int = block.app_command_dispatch_status(text_cmd, block.input_text(), block.event_kind_text(), true)
    let delivered: block.AppEventTrace = block.app_event_trace(1, text_cmd, block.event_kind_text(), focused_status, block.id(4), 2)
    let rejected_disabled: block.AppEventTrace = block.app_event_trace(2, save_cmd, block.event_kind_click(), disabled_status, block.id(4), 0)
    let rejected_unfocused: block.AppEventTrace = block.app_event_trace(3, text_cmd, block.event_kind_text(), unfocused_status, block.id(5), 2)

    let policy: block.AppModelPolicy = block.app_model_policy_production()
    let nav: block.AppNavigationStep = block.app_navigation_step(block.id(4), block.id(5), block.app_surface_command_palette(), false)
    let trap: block.AppNavigationStep = block.app_navigation_step(block.id(4), block.id(4), block.app_surface_editor_shell(), true)
    let redraw: block.AppRedrawRequest = block.app_redraw_request(editor, 1, 2, 111, 222)

    if block.app_model_policy_valid(policy) &&
       block.app_state_store_valid(palette) && block.app_state_store_valid(dashboard) && block.app_state_store_valid(settings) && block.app_state_store_valid(editor) &&
       block.app_command_safe(open_cmd) && block.app_command_safe(text_cmd) && block.app_async_boundary_safe(refresh_cmd) &&
       disabled_status == block.event_route_rejected_disabled() &&
       unfocused_status == block.event_route_rejected_unfocused() &&
       focused_status == block.event_route_delivered() &&
       block.app_event_trace_valid(delivered) && block.app_event_trace_valid(rejected_disabled) && block.app_event_trace_valid(rejected_unfocused) &&
       block.app_navigation_valid(nav) && block.app_navigation_valid(trap) && trap.focus_trap &&
       block.app_shortcut_scope_allows(block.app_surface_command_palette(), block.app_surface_command_palette(), true) &&
       !block.app_shortcut_scope_allows(block.app_surface_settings(), block.app_surface_command_palette(), true) &&
       block.app_error_propagated_handled(true, true) &&
       block.app_redraw_valid(redraw):
        return 0
    return 1
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt(lib.core.block app model consumer): %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(lib.core.block app model consumer): %v", err)
	}
}

func TestSurfaceBlockKeyboardUXDefinesFocusShortcutUndoAPI(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.block as block

func main() -> Int:
    let palette: block.AppStateStore = block.app_state_store(1, block.app_surface_command_palette(), block.app_state_scope_overlay(), 3, 10, 20, 2)
    let editor: block.AppStateStore = block.app_state_store(2, block.app_surface_editor_shell(), block.app_state_scope_document(), 4, 30, 40, 2)

    let action_cmd: block.AppCommand = block.app_command(block.app_command_kind_shortcut(), block.app_source_key(), block.id(5), palette, 1, true, false)
    let open_cmd: block.AppCommand = block.app_command(block.app_command_kind_command(), block.app_source_key(), block.id(4), palette, 2, true, false)
    let undo_cmd: block.AppCommand = block.app_command(block.app_command_kind_undo(), block.app_source_key(), block.id(4), editor, 2, true, false)
    let redo_cmd: block.AppCommand = block.app_command(block.app_command_kind_redo(), block.app_source_key(), block.id(4), editor, 2, true, false)

    let input_node: block.KeyboardFocusNode = block.keyboard_focus_node(block.id(4), block.app_surface_editor_shell(), block.accessibility_role_textbox(), 0, block.id(3))
    let action_node: block.KeyboardFocusNode = block.keyboard_focus_node(block.id(5), block.app_surface_command_palette(), block.accessibility_role_button(), 11, block.id_none())

    let enter_binding: block.KeyboardBinding = block.keyboard_binding(block.keyboard_key_enter(), block.app_surface_command_palette(), action_cmd, true)
    let palette_binding: block.KeyboardBinding = block.keyboard_binding(block.keyboard_key_ctrl_k(), block.app_surface_command_palette(), open_cmd, true)
    let undo_binding: block.KeyboardBinding = block.keyboard_binding(block.keyboard_key_ctrl_z(), block.app_surface_editor_shell(), undo_cmd, true)
    let redo_binding: block.KeyboardBinding = block.keyboard_binding(block.keyboard_key_ctrl_y(), block.app_surface_editor_shell(), redo_cmd, true)

    let conflict: block.KeyboardShortcutConflict = block.keyboard_shortcut_conflict(block.keyboard_key_ctrl_k(), block.app_surface_command_palette(), 2, true, true)
    let stack: block.KeyboardUndoRedoStack = block.keyboard_undo_redo_stack(editor, 1, 0, true, true)
    let command_script: block.KeyboardScript = block.keyboard_script(block.app_surface_command_palette(), 3, true, true)
    let settings_script: block.KeyboardScript = block.keyboard_script(block.app_surface_settings(), 3, true, true)
    let trap: block.AppNavigationStep = block.app_navigation_step(block.id(4), block.id(4), block.app_surface_command_palette(), true)

    if block.keyboard_policy_focus_order_graph() == 1 &&
       block.keyboard_policy_focus_trap_overlay() == 2 &&
       block.keyboard_policy_roving_focus() == 3 &&
       block.keyboard_policy_activation() == 4 &&
       block.keyboard_policy_shortcut_conflict() == 5 &&
       block.keyboard_policy_undo_redo_stack() == 6 &&
       block.keyboard_focus_node_valid(input_node) && block.keyboard_focus_node_valid(action_node) &&
       block.keyboard_binding_valid(enter_binding) && block.keyboard_binding_valid(palette_binding) &&
       block.keyboard_binding_valid(undo_binding) && block.keyboard_binding_valid(redo_binding) &&
       block.keyboard_shortcut_conflict_valid(conflict) &&
       block.keyboard_undo_redo_valid(stack) &&
       block.keyboard_script_valid(command_script) && block.keyboard_script_valid(settings_script) &&
       block.keyboard_focus_trap_valid(trap, true, true) &&
       block.keyboard_roving_group_valid(block.id(4), block.id(4), block.id(5), true, true, true):
        return 0
    return 1
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt(lib.core.block keyboard UX consumer): %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(lib.core.block keyboard UX consumer): %v", err)
	}
}

func TestSurfaceBlockMotionExampleLoadsTransitionModel(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_motion.tetra")
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
	}

	for _, module := range []string{"lib.core.surface", "lib.core.block"} {
		if _, ok := world.ByModule[module]; !ok {
			t.Fatalf("surface block motion did not load module %s; modules=%v", module, world.ByModule)
		}
	}

	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface block motion): %v", err)
	}
}

func TestSurfaceBlockMotionExampleRunsTransitionValidation(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_motion.tetra")
	out := filepath.Join(t.TempDir(), "surface-block-motion")
	if _, err := compiler.BuildFileWithStatsOpt(entry, out, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt(surface block motion): %v", err)
	}
	cmd := exec.Command(out)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("surface block motion exited with 0, want Surface smoke success exit 1\n%s", output)
	}
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 1 {
		t.Fatalf("surface block motion exited with %v, want exit 1\n%s", err, output)
	}
}

func TestSurfaceBlockMotionDefinesTransitionAPI(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.block as block

func main() -> Int:
    let motion: block.MotionSpec = block.motion_transition(120, 0, block.motion_ease_linear(), true, true, true, 12, 0, 108)
    let reduced: block.MotionSpec = block.motion_reduced(motion)
    let progress0: Int = block.motion_progress(motion, 0)
    let progress_mid: Int = block.motion_progress(motion, 60)
    let progress_done: Int = block.motion_progress(motion, 120)
    let opacity_mid: Int = block.motion_resolve_opacity(80, 200, motion, 60)
    let color_mid: Int = block.motion_resolve_color_channel(32, 96, motion, 60)
    let tx_mid: Int = block.motion_resolve_translate_x(motion, 60)
    let scale_mid: Int = block.motion_resolve_scale(motion, 60)
    let reduced_progress: Int = block.motion_progress(reduced, 0)
    let timing_mid: Bool = block.motion_frame_timing_ok(0, 60)
    let lifecycle_stops: Bool = block.motion_lifecycle_complete_stops(motion, 120)
    let reduced_stops: Bool = block.motion_reduced_stops_schedule(motion)

    if block.motion_duration(motion) == 120 &&
       block.motion_delay(motion) == 0 &&
       block.motion_easing(motion) == block.motion_ease_linear() &&
       block.motion_frame_interval_ms() == 16 &&
       block.motion_frame_budget_default() == 4 &&
       block.motion_max_frame_delta_ms() == 60 &&
       progress0 == 0 && progress_mid == 500 && progress_done == 1000 &&
       opacity_mid == 140 && color_mid == 64 &&
       tx_mid == 6 && scale_mid == 104 &&
       block.motion_should_schedule(motion, 60) &&
       !block.motion_should_schedule(motion, 120) &&
       block.motion_is_complete(motion, 120) &&
       reduced.reduced && reduced_progress == 1000 &&
       timing_mid && lifecycle_stops && reduced_stops &&
       !block.motion_should_schedule(reduced, 0):
        return 0
    return 1
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt(lib.core.block motion consumer): %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(lib.core.block motion consumer): %v", err)
	}
}

func TestSurfaceBlockAssetsExampleLoadsAssetPipeline(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_assets.tetra")
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
	}

	for _, module := range []string{"lib.core.surface", "lib.core.block"} {
		if _, ok := world.ByModule[module]; !ok {
			t.Fatalf("surface block assets did not load module %s; modules=%v", module, world.ByModule)
		}
	}

	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface block assets): %v", err)
	}
}

func TestSurfaceBlockAssetsExampleRunsAssetValidation(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_assets.tetra")
	out := filepath.Join(t.TempDir(), "surface-block-assets")
	if _, err := compiler.BuildFileWithStatsOpt(entry, out, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt(surface block assets): %v", err)
	}
	cmd := exec.Command(out)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("surface block assets exited with 0, want Surface smoke success exit 1\n%s", output)
	}
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 1 {
		t.Fatalf("surface block assets exited with %v, want exit 1\n%s", err, output)
	}
}

func TestSurfaceBlockAssetsDefinesManifestAPI(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface
import lib.core.block as block

func main() -> Int:
    let font: block.AssetRef = block.asset_embedded(block.asset_kind_font(), 1, 0, 0, 101)
    let icon: block.AssetRef = block.asset_icon(2, 16, 16, 202)
    let image: block.AssetRef = block.asset_image(3, 48, 32, 303)
    let vector: block.AssetRef = block.asset_vector(4, 24, 24, 404)
    let missing: block.AssetRef = block.asset_missing(block.asset_kind_image(), 9)
    let remote: block.AssetRef = block.asset_remote_url(block.asset_kind_image(), 10)
    let manifest: block.AssetManifest = block.asset_manifest(font, icon, image, vector, block.asset_cache_budget_bytes(), block.asset_cache_entry_limit())
    let tinted: block.ImageSpec = block.image_asset_tinted_scaled(icon, 32, 32, surface.Color(r: 96, g: 174, b: 244, a: 255), 1)
    let scaled: block.ImageSpec = block.image_asset_tinted_scaled(image, 96, 64, surface.Color(r: 255, g: 255, b: 255, a: 255), 2)
    let vector_image: block.ImageSpec = block.image_asset_tinted_scaled(vector, 40, 32, surface.Color(r: 96, g: 174, b: 244, a: 255), 1)

    if block.asset_manifest_validate(manifest) == block.asset_error_ok() &&
       block.asset_manifest_hash(manifest) > 0 &&
       block.asset_is_embedded(font) &&
       block.asset_is_local(icon) &&
       block.asset_is_local(vector) &&
       block.asset_hash(icon) == 202 &&
       block.asset_width(image) == 48 && block.asset_height(image) == 32 &&
       block.asset_cache_validate(block.asset_cache_budget_bytes(), 4096, 4, 6) == block.asset_error_ok() &&
       block.asset_cache_validate(0, 0, 0, 0) == block.asset_error_unbounded_cache() &&
       block.asset_resolve_status(missing) == block.asset_error_missing_fallback() &&
       block.asset_resolve_status(remote) == block.asset_error_network_rejected() &&
       block.asset_vector_sanitize_status(vector) == block.asset_error_ok() &&
       block.asset_vector_sanitize_status(remote) == block.asset_error_unsafe_svg() &&
       block.asset_diagnostic_missing_asset() > 0 &&
       block.asset_diagnostic_network_rejected() > 0 &&
       tinted.asset_kind == block.asset_kind_icon() &&
       tinted.tint_b == 244 &&
       scaled.width == 96 && scaled.height == 64 && scaled.fit == 2 &&
       vector_image.asset_kind == block.asset_kind_vector() && vector_image.width == 40:
        return 0
    return 1
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt(lib.core.block asset consumer): %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(lib.core.block asset consumer): %v", err)
	}
}

func TestSurfaceBlockAccessibilityExampleLoadsMetadataModel(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_accessibility.tetra")
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
	}

	for _, module := range []string{"lib.core.surface", "lib.core.block"} {
		if _, ok := world.ByModule[module]; !ok {
			t.Fatalf("surface block accessibility did not load module %s; modules=%v", module, world.ByModule)
		}
	}

	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface block accessibility): %v", err)
	}
}

func TestSurfaceBlockAccessibilityExampleRunsMetadataValidation(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_accessibility.tetra")
	out := filepath.Join(t.TempDir(), "surface-block-accessibility")
	if _, err := compiler.BuildFileWithStatsOpt(entry, out, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt(surface block accessibility): %v", err)
	}
	cmd := exec.Command(out)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("surface block accessibility exited with 0, want Surface smoke success exit 1\n%s", output)
	}
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 1 {
		t.Fatalf("surface block accessibility exited with %v, want exit 1\n%s", err, output)
	}
}

func TestSurfaceBlockSystemExampleLoadsHeadlessGoldenModel(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_system.tetra")
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
	}

	for _, module := range []string{"lib.core.surface", "lib.core.block"} {
		if _, ok := world.ByModule[module]; !ok {
			t.Fatalf("surface block system did not load module %s; modules=%v", module, world.ByModule)
		}
	}

	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface block system): %v", err)
	}
}

func TestSurfaceBlockSystemExampleRunsHeadlessGoldenValidation(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_block_system.tetra")
	out := filepath.Join(t.TempDir(), "surface-block-system")
	if _, err := compiler.BuildFileWithStatsOpt(entry, out, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt(surface block system): %v", err)
	}
	cmd := exec.Command(out)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("surface block system exited with 0, want Surface smoke success exit 1\n%s", output)
	}
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 1 {
		t.Fatalf("surface block system exited with %v, want exit 1\n%s", err, output)
	}
}

func TestSurfaceBlockExamplesAreBlockOnlyBeautifulScenes(t *testing.T) {
	for _, rel := range surfaceBlockBeautyExamplePaths() {
		rel := rel
		t.Run(filepath.Base(rel), func(t *testing.T) {
			entry := testkit.RepoPath(t, strings.Split(rel, "/")...)
			raw, err := os.ReadFile(entry)
			if err != nil {
				t.Fatalf("read %s: %v", rel, err)
			}
			text := string(raw)
			lower := strings.ToLower(text)
			for _, want := range []string{
				"import lib.core.surface as surface",
				"import lib.core.block as block",
				"theme_dark",
				"theme_light",
				"block.layout_",
				"block.paint_stack",
				"block.text_",
				"block.asset_",
				"block.accessibility_",
				"block.state_selector_hover()",
				"block.state_selector_focused()",
				"block.state_selector_pressed()",
				"block.motion_",
				"scene_checksum",
			} {
				if !strings.Contains(text, want) {
					t.Fatalf("%s missing Block beauty evidence marker %q", rel, want)
				}
			}
			for _, forbidden := range []string{
				"import lib.core.widgets",
				"widgets.",
				"widgets.Button",
				"widgets.TextBox",
				"Button(",
				"TextBox(",
				"Card(",
				"Modal(",
				"react",
				"electron",
				"dom ui",
				"user js",
			} {
				if strings.Contains(text, forbidden) || strings.Contains(lower, forbidden) {
					t.Fatalf("%s contains forbidden non-Block marker %q", rel, forbidden)
				}
			}

			world, err := compiler.LoadWorld(entry)
			if err != nil {
				t.Fatalf("LoadWorld(%s): %v", rel, err)
			}
			for _, module := range []string{"lib.core.surface", "lib.core.block"} {
				if _, ok := world.ByModule[module]; !ok {
					t.Fatalf("%s did not load module %s; modules=%v", rel, module, world.ByModule)
				}
			}
			if _, err := compiler.CheckWorld(world); err != nil {
				t.Fatalf("CheckWorld(%s): %v", rel, err)
			}

			out := filepath.Join(t.TempDir(), strings.TrimSuffix(filepath.Base(rel), ".tetra"))
			if _, err := compiler.BuildFileWithStatsOpt(entry, out, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
				t.Fatalf("BuildFileWithStatsOpt(%s): %v", rel, err)
			}
			cmd := exec.Command(out)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%s exited with %v\n%s", rel, err, output)
			}
		})
	}
}

func TestSurfaceBlockAccessibilityDefinesMetadataAPI(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.block as block

func main() -> Int:
    let label_id: block.BlockID = block.id(3)
    let submit_id: block.BlockID = block.id(4)
    let label: block.AccessibilitySpec = block.accessibility_label_for(4, submit_id)
    let submit: block.AccessibilitySpec = block.accessibility_button_labelled_by(6, label_id, block.action_primary())
    let unnamed: block.AccessibilitySpec = block.accessibility_button(0)

    if block.accessibility_focusable_has_name(submit) &&
       !block.accessibility_focusable_has_name(unnamed) &&
       block.accessibility_relationship_matches(label, submit, label_id, submit_id) &&
       block.accessibility_reading_order_matches(0, 1) &&
       block.accessibility_metadata_claim_scoped(false, false, false) &&
       !block.accessibility_metadata_claim_scoped(true, false, false) &&
       !block.accessibility_screen_reader_claim_allowed(false, true) &&
       block.accessibility_screen_reader_claim_allowed(true, true):
        return 0
    return 1
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt(lib.core.block accessibility consumer): %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(lib.core.block accessibility consumer): %v", err)
	}
}

func TestSurfaceBlockEditableTextRejectsBorrowedStorage(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.text as text

func bad_editable(xs: borrow []u8) -> text.EditableText:
    return text.editable_empty(xs.window(0, 2).borrow(), 6)

func main() -> Int:
    return 0
`,
	}, "borrowed value derived from 'xs' cannot be passed to non-borrow parameter 1 of 'lib.core.text.editable_empty'")
}

func TestSurfaceBlockModuleDefinesPaintLayerGrammar(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface
import lib.core.draw as draw
import lib.core.block as block

func main() -> Int
uses alloc, mem:
    let rect: surface.Rect = surface.Rect(x: 4, y: 4, w: 32, h: 20)
    let bg: surface.Color = surface.Color(r: 20, g: 28, b: 36, a: 255)
    let fill_color: surface.Color = surface.Color(r: 42, g: 108, b: 214, a: 255)
    let hi_color: surface.Color = surface.Color(r: 88, g: 180, b: 132, a: 255)
    let border_color: surface.Color = surface.Color(r: 225, g: 232, b: 240, a: 255)
    let shadow_color: surface.Color = surface.Color(r: 0, g: 0, b: 0, a: 96)
    let outline_color: surface.Color = surface.Color(r: 246, g: 205, b: 92, a: 255)

    let fill: block.PaintLayer = block.paint_layer_fill_radius(fill_color, 8)
    let gradient: block.PaintLayer = block.paint_layer_linear_gradient(fill_color, hi_color, 8)
    let border: block.PaintLayer = block.paint_layer_border(border_color, 1, 8)
    let shadow: block.PaintLayer = block.paint_layer_shadow(shadow_color, 12, 0, 4)
    let outline: block.PaintLayer = block.paint_layer_outline(outline_color, 2, 10)
    let paint: block.PaintSpec = block.paint_stack5(fill, gradient, border, shadow, outline)
    let flags: Int = block.paint_feature_flags(paint)
    let command0: Int = block.paint_resolve_command(paint, 0)
    let command1: Int = block.paint_resolve_command(paint, 1)
    let command2: Int = block.paint_resolve_command(paint, 2)
    let command3: Int = block.paint_resolve_command(paint, 3)
    let command4: Int = block.paint_resolve_command(paint, 4)
    let valid: Int = block.paint_validate_visual_grammar(paint)

    var pixels: []u8 = core.make_u8(32 * 24 * 4)
    let surface_ref: surface.Surface = surface.Surface(handle: 0, width: 32, height: 24)
    var frame: surface.Frame = surface.Frame(surface: surface_ref, width: 32, height: 24, stride: 128, pixels: pixels)
    var ctx: draw.DrawContext = draw.DrawContext(frame: frame)
    let clear_ok: Int = draw.clear(ctx, bg)
    let shadow_ok: Int = draw.box_shadow_approx(ctx, rect, 8, 12, 0, 4, shadow_color)
    let gradient_ok: Int = draw.linear_gradient_rect(ctx, rect, fill_color, hi_color)
    let fill_ok: Int = draw.rounded_rect(ctx, rect, 8, fill_color)
    let border_ok: Int = draw.rounded_rect_outline(ctx, rect, 8, 1, border_color)

    let blur: block.PaintLayer = block.paint_layer_blur(16)
    let invalid_blur: block.PaintSpec = block.paint_from_layer(blur)
    let blur_status: Int = block.paint_validate(invalid_blur)

    if valid == block.paint_error_ok() &&
       blur_status == block.paint_error_unsupported_blur() &&
       block.paint_has_feature(flags, block.paint_feature_fill()) &&
       block.paint_has_feature(flags, block.paint_feature_border()) &&
       block.paint_has_feature(flags, block.paint_feature_radius()) &&
       block.paint_has_feature(flags, block.paint_feature_shadow()) &&
       block.paint_has_feature(flags, block.paint_feature_outline()) &&
       block.paint_has_feature(flags, block.paint_feature_gradient()) &&
       command0 == block.paint_command_fill() &&
       command1 == block.paint_command_gradient() &&
       command2 == block.paint_command_border() &&
       command3 == block.paint_command_shadow() &&
       command4 == block.paint_command_outline() &&
       clear_ok == 0 && shadow_ok == draw.paint_command_shadow() && gradient_ok == draw.paint_command_gradient() &&
       fill_ok == draw.paint_command_fill() && border_ok == draw.paint_command_border():
        return 0
    return 1
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt: %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(lib.core.block paint consumer): %v", err)
	}
}

func TestSurfaceReleaseTextInputExampleLoadsCoreTextModule(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_release_text_input.tetra")
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
	}

	for _, module := range []string{"lib.core.surface", "lib.core.draw", "lib.core.text"} {
		if _, ok := world.ByModule[module]; !ok {
			t.Fatalf("surface release text input did not load module %s; modules=%v", module, world.ByModule)
		}
	}

	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface release text input): %v", err)
	}
}

func TestSurfaceReleaseTextInputExampleBuildsLinuxX64(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_release_text_input.tetra")
	out := filepath.Join(t.TempDir(), "surface-release-text-input")

	if _, err := compiler.BuildFileWithStatsOpt(entry, out, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt(surface release text input): %v", err)
	}
}

func TestSurfaceReleaseCounterExampleLoadsStableWidgetAccessibilityModules(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_release_counter.tetra")
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
	}

	for _, module := range []string{
		"lib.core.surface",
		"lib.core.draw",
		"lib.core.component",
		"lib.core.widgets",
		"lib.core.style",
		"lib.core.accessibility",
	} {
		if _, ok := world.ByModule[module]; !ok {
			t.Fatalf("surface release counter did not load module %s; modules=%v", module, world.ByModule)
		}
	}

	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface release counter): %v", err)
	}
}

func TestSurfaceModuleDefinesClipboardAndCompositionABIWrappers(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses alloc, mem, surface:
    var win: surface.Surface = surface.open("clipboard-ime", 160, 80)
    var text: []u8 = core.make_u8(3)
    text[0] = 84
    text[1] = 101
    text[2] = 116
    var out: []u8 = core.make_u8(3)
    var slots: []i32 = core.make_i32(4)
    let wrote: Int = surface.clipboard_write_text(win, text)
    let read: Int = surface.clipboard_read_text_into(win, out)
    let copied: Int = surface.poll_composition_into(win, slots)
    let trace: surface.CompositionTrace = surface.poll_composition(win)
    let closed: Int = surface.close(win)
    if wrote == 3 && read == 3 && copied == 4 && trace.start && trace.update && trace.commit && trace.cancel && surface.event_composition_start() == 10 && closed == 0:
        return 0
    return 1
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	if _, err := compiler.BuildFileWithStatsOpt(entry, filepath.Join(t.TempDir(), "surface-clipboard-ime"), "linux-x64", compiler.BuildOptions{
		DependencyRoots: []compiler.ModuleRoot{{Root: testkit.RepoRoot(t)}},
		Jobs:            1,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt(lib.core.surface clipboard/IME consumer): %v", err)
	}
}

func TestSurfaceModuleDefinesAppShellHostABI(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int:
    let window: surface.ShellWindowSpec = surface.shell_window_spec(20, 400, 240, 320, 200, surface.shell_window_resizable())
    let cap: surface.ShellCapability = surface.shell_capability(surface.shell_cap_menu(), true, true, false)
    let trace: surface.ShellActionTrace = surface.shell_action_trace(surface.shell_cap_menu(), 1, true, false, 0)
    let denied: surface.ShellPermission = surface.shell_permission(surface.shell_cap_permission(), surface.shell_permission_scoped(), false, true, 1)
    let diagnostic: surface.ShellDiagnostic = surface.shell_unsupported_diagnostic(surface.shell_cap_platform_widget_shell(), 1)
    let dpi: surface.ShellDPI = surface.shell_dpi(2000, 800, 480)
    if surface.shell_window_spec_valid(window) && surface.shell_capability_valid(cap) && surface.shell_action_trace_valid(trace) && surface.shell_permission_valid(denied) && surface.shell_diagnostic_valid(diagnostic) && surface.shell_dpi_valid(dpi):
        return 0
    return 1
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	if _, err := compiler.BuildFileWithStatsOpt(entry, filepath.Join(t.TempDir(), "surface-app-shell-abi"), "linux-x64", compiler.BuildOptions{
		DependencyRoots: []compiler.ModuleRoot{{Root: testkit.RepoRoot(t)}},
		Jobs:            1,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt(lib.core.surface app shell ABI consumer): %v", err)
	}
}

func TestSurfaceClipboardRejectsBorrowedTextBoundaryWithoutCopy(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "surface wrapper",
			body: `
    let copied: Int = surface.clipboard_write_text(win, borrowed)
`,
			want: "borrowed value derived from 'xs' cannot be passed to non-borrow parameter 2 of 'lib.core.surface.clipboard_write_text'",
		},
		{
			name: "raw host abi",
			body: `
    let copied: Int = core.surface_clipboard_write_text(win.handle, borrowed)
`,
			want: "borrowed value derived from 'xs' cannot be passed to non-borrow parameter 2 of 'core.surface_clipboard_write_text'",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			requireSurfaceCheckErrorContains(t, map[string]string{
				"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses alloc, mem, surface:
    var win: surface.Surface = surface.open("clipboard-borrow", 160, 80)
    var xs: []u8 = core.make_u8(4)
    let borrowed: []u8 = xs.window(0, 3).borrow()
` + tc.body + `
    let closed: Int = surface.close(win)
    return copied + closed
`,
			}, tc.want)
		})
	}
}

func TestSurfaceClipboardAcceptsCopiedTextBoundary(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses alloc, mem, surface:
    var win: surface.Surface = surface.open("clipboard-copy", 160, 80)
    var xs: []u8 = core.make_u8(4)
    let copied_text: []u8 = xs.window(0, 3).borrow().copy()
    let copied: Int = surface.clipboard_write_text(win, copied_text)
    let closed: Int = surface.close(win)
    return copied + closed
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt: %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(copied clipboard boundary): %v", err)
	}
}

func TestSurfaceSafeViewLifetimeRejectsBorrowedTextBoxBuffer(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface
import lib.core.widgets as widgets

func bad_textbox_init(xs: borrow []u8) -> widgets.TextBox
uses alloc, mem:
    let rect: surface.Rect = surface.Rect(x: 0, y: 0, w: 160, h: 48)
    var storage: []u8 = core.make_u8(8)
    var box: widgets.TextBox = widgets.TextBox(rect: rect, focused: false, text_len: 0, caret: 0, buffer: storage)
    let ok: Int = widgets.textbox_init(box, rect, xs.window(0, 2).borrow())
    return box

func main() -> Int:
    return 0
`,
	}, "borrowed value derived from 'xs' cannot be passed to non-borrow parameter 3 of 'lib.core.widgets.textbox_init'")
}

func TestSurfaceSafeViewLifetimeRejectsBorrowedWidgetStateLabel(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main

struct WidgetState:
    label: String

func bad_widget_label(text: borrow String) -> WidgetState:
    return WidgetState(label: text.window(0, 2).borrow())

func main() -> Int:
    return 0
`,
	}, "aggregate 'WidgetState' contains borrowed String field 'label' that cannot escape through owned return")
}

func TestSurfaceSafeViewLifetimeRejectsBorrowedAccessibilityLabel(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.accessibility as accessibility

struct AccessibilityLabelState:
    label: String
    metadata: accessibility.NodeMetadata

func bad_accessibility_label(text: borrow String) -> AccessibilityLabelState:
    let metadata: accessibility.NodeMetadata = accessibility.label_metadata(1, accessibility.value_name(), 1)
    return AccessibilityLabelState(label: text.window(0, 2).borrow(), metadata: metadata)

func main() -> Int:
    return 0
`,
	}, "aggregate 'AccessibilityLabelState' contains borrowed String field 'label' that cannot escape through owned return")
}

func TestSurfaceSafeViewLifetimeAcceptsOwnedCopyState(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.accessibility as accessibility
import lib.core.surface as surface
import lib.core.widgets as widgets

struct AccessibilityLabelState:
    label: String
    metadata: accessibility.NodeMetadata

struct WidgetState:
    label: String
    box: widgets.TextBox
    accessibility_label: AccessibilityLabelState

func good_state(text: borrow String, bytes: borrow []u8) -> WidgetState
uses alloc, mem:
    let rect: surface.Rect = surface.Rect(x: 0, y: 0, w: 160, h: 48)
    let copied_label: String = text.window(0, 2).copy()
    let copied_buffer: []u8 = bytes.window(0, 2).copy()
    let metadata: accessibility.NodeMetadata = accessibility.label_metadata(1, accessibility.value_name(), 1)
    let label_state: AccessibilityLabelState = AccessibilityLabelState(label: copied_label.copy(), metadata: metadata)
    let box: widgets.TextBox = widgets.TextBox(rect: rect, focused: false, text_len: 0, caret: 0, buffer: copied_buffer)
    return WidgetState(label: copied_label, box: box, accessibility_label: label_state)

func main() -> Int:
    return 0
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt: %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface safe-view owned-copy state): %v", err)
	}
}

func TestSurfaceMigrationExamplesCheck(t *testing.T) {
	examples := []string{
		filepath.Join("examples", "surface_migration_ui_web_smoke.tetra"),
		filepath.Join("examples", "surface_migration_ui_native_shell_smoke.tetra"),
		filepath.Join("examples", "surface_migration_dogfood_web_ui.tetra"),
		filepath.Join("examples", "surface_migration_tetra_control_center.tetra"),
	}

	for _, rel := range examples {
		rel := rel
		t.Run(filepath.ToSlash(rel), func(t *testing.T) {
			entry := testkit.RepoPath(t, rel)
			world, err := compiler.LoadWorld(entry)
			if err != nil {
				t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
			}
			if _, err := compiler.CheckWorld(world); err != nil {
				t.Fatalf("CheckWorld(%s): %v", filepath.ToSlash(entry), err)
			}
		})
	}
}

func TestSurfaceFrameCannotEscapeViaReturn(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func leak(win: borrow surface.Surface) -> surface.Frame
uses alloc, mem, surface:
    let frame: surface.Frame = surface.begin_frame(win)
    return frame

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.surface.Frame' cannot escape via return")
}

func TestSurfaceFramePixelsCannotEscapeViaReturn(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func leak(win: borrow surface.Surface) -> []u8
uses alloc, mem, surface:
    let frame: surface.Frame = surface.begin_frame(win)
    return frame.pixels

func main() -> Int:
    return 0
`,
	}, "surface frame pixels cannot escape via return")
}

func TestSurfaceFramePixelsAliasCannotEscapeViaReturn(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func leak(win: borrow surface.Surface) -> []u8
uses alloc, mem, surface:
    let frame: surface.Frame = surface.begin_frame(win)
    var pixels: []u8 = frame.pixels
    return pixels

func main() -> Int:
    return 0
`,
	}, "surface frame pixels cannot escape via return")
}

func TestSurfaceFramePixelsCannotEscapeViaStructConstructorReturn(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

struct PixelBox:
    pixels: []u8

func leak(win: borrow surface.Surface) -> PixelBox
uses alloc, mem, surface:
    let frame: surface.Frame = surface.begin_frame(win)
    return PixelBox(pixels: frame.pixels)

func main() -> Int:
    return 0
`,
	}, "surface frame pixels cannot escape via return")
}

func TestSurfaceFramePixelsCannotEscapeViaInoutAssignment(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func leak(win: borrow surface.Surface, out: inout []u8) -> Int
uses alloc, mem, surface:
    let frame: surface.Frame = surface.begin_frame(win)
    out = frame.pixels
    return 0

func main() -> Int:
    return 0
`,
	}, "surface frame pixels cannot escape via inout assignment to 'out'")
}

func TestSurfaceEventCannotBeStoredInGlobalState(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

var leaked: surface.Event

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.surface.Event' cannot be stored in global 'leaked'")
}

func TestSurfaceEventCannotBeStoredInUserStructField(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

struct EventBox:
    event: surface.Event

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.surface.Event' cannot be stored in struct field 'event'")
}

func TestSurfaceDrawContextCannotBeStoredInUserStructField(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.draw as draw

struct ContextBox:
    ctx: draw.DrawContext

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.draw.DrawContext' cannot be stored in struct field 'ctx'")
}

func TestSurfaceEventCannotBeStoredInUserEnumPayload(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

enum EventSlot:
    case event(surface.Event)

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.surface.Event' cannot be stored in enum payload 'event'")
}

func TestSurfaceDrawContextCannotBeStoredInUserEnumPayload(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.draw as draw

enum ContextSlot:
    case ctx(draw.DrawContext)

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.draw.DrawContext' cannot be stored in enum payload 'ctx'")
}

func TestSurfaceEventCannotEscapeViaThrow(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func leak(win: borrow surface.Surface) -> Int throws surface.Event
uses alloc, mem, surface:
    let event: surface.Event = surface.poll_event(win)
    throw event

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.surface.Event' cannot escape via throw")
}

func TestSurfaceEventCannotEscapeViaFunctionTypedReturnCapture(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func pick() -> fn(Int) -> Int:
    let event: surface.Event = surface.Event(kind: 5, x: 0, y: 0, button: 0, key: 0, width: 1, height: 1, timestamp_ms: 0, text_len: 0)
    let cb: fn(Int) -> Int = fn(x: Int) -> Int:
        return event.kind + x
    return cb

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.surface.Event' cannot escape via function capture")
}

func TestSurfaceFramePixelsCannotEscapeViaFunctionTypedReturnCapture(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func pick(win: borrow surface.Surface) -> fn(Int) -> Int
uses alloc, mem, surface:
    let frame: surface.Frame = surface.begin_frame(win)
    let pixels: []u8 = frame.pixels
    let cb: fn(Int) -> Int = fn(x: Int) -> Int:
        return pixels[0] + x
    return cb

func main() -> Int:
    return 0
`,
	}, "surface frame pixels cannot escape via function capture")
}

func TestSurfaceFramePixelsCannotEscapeViaThrow(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func leak(win: borrow surface.Surface) -> Int throws []u8
uses alloc, mem, surface:
    let frame: surface.Frame = surface.begin_frame(win)
    throw frame.pixels

func main() -> Int:
    return 0
`,
	}, "surface frame pixels cannot escape via throw")
}

func TestSurfaceEventCannotCrossTypedTaskErrorBoundary(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

enum TaskErr:
    case event(surface.Event)

func worker() -> Int throws TaskErr:
    return 42

func caller() -> Int throws TaskErr
uses runtime:
    let task: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker")
    return try core.task_join_i32_typed<TaskErr>(task)

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.surface.Event' cannot be stored in enum payload 'event'")
}

func TestSurfaceEventCannotCrossTypedActorMessageBoundary(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

enum SurfaceMsg:
    case event(surface.Event)

func main() -> Int
uses actors:
    let msg: SurfaceMsg = core.recv_typed<SurfaceMsg>()
    return 0
`,
	}, "surface value 'lib.core.surface.Event' cannot be stored in enum payload 'event'")
}

func TestSurfaceHandleCannotCrossTypedTaskErrorBoundary(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

enum TaskErr:
    case window(surface.Surface)

func worker() -> Int throws TaskErr:
    return 42

func caller() -> Int throws TaskErr
uses runtime:
    let task: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker")
    return try core.task_join_i32_typed<TaskErr>(task)

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.surface.Surface' cannot cross actor/task boundary")
}

func TestSurfaceHandleCannotCrossTypedActorMessageBoundary(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

enum SurfaceMsg:
    case window(surface.Surface)

func main() -> Int
uses actors:
    let msg: SurfaceMsg = core.recv_typed<SurfaceMsg>()
    return 0
`,
	}, "surface value 'lib.core.surface.Surface' cannot cross actor/task boundary")
}

func TestSurfaceDrawContextCannotEscapeViaReturn(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface
import lib.core.draw as draw

func leak(win: borrow surface.Surface) -> draw.DrawContext
uses alloc, mem, surface:
    var frame: surface.Frame = surface.begin_frame(win)
    var ctx: draw.DrawContext = draw.DrawContext(frame: frame)
    return ctx

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.draw.DrawContext' cannot escape via return")
}

func TestSurfaceFrameCannotEscapeViaInoutAssignment(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func leak(win: borrow surface.Surface, out: inout surface.Frame) -> Int
uses alloc, mem, surface:
    out = surface.begin_frame(win)
    return 0

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.surface.Frame' cannot escape via inout assignment to 'out'")
}

func TestSurfaceDrawContextCannotUseFrameAfterPresent(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface
import lib.core.draw as draw

func main() -> Int
uses alloc, mem, surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    var frame: surface.Frame = surface.begin_frame(win)
    var ctx: draw.DrawContext = draw.DrawContext(frame: frame)
    let color: surface.Color = surface.Color(r: 0, g: 0, b: 0, a: 255)
    let presented: Int = surface.present(ctx.frame)
    let draw_status: Int = draw.clear(ctx, color)
    let closed: Int = surface.close(win)
    return presented + draw_status + closed
`,
	}, "cannot use consumed value 'ctx.frame'")
}

func TestSurfaceFramePixelsAliasCannotBeUsedAfterPresent(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses alloc, mem, surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    var frame: surface.Frame = surface.begin_frame(win)
    var pixels: []u8 = frame.pixels
    let presented: Int = surface.present(frame)
    pixels[0] = 255
    let closed: Int = surface.close(win)
    return presented + closed
`,
	}, "surface frame pixels alias 'pixels' cannot be used after frame 'frame' was presented")
}

func TestSurfaceDrawContextFramePixelsAliasCannotBeUsedAfterPresent(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface
import lib.core.draw as draw

func main() -> Int
uses alloc, mem, surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    var frame: surface.Frame = surface.begin_frame(win)
    var ctx: draw.DrawContext = draw.DrawContext(frame: frame)
    var pixels: []u8 = ctx.frame.pixels
    let presented: Int = surface.present(ctx.frame)
    pixels[0] = 255
    let closed: Int = surface.close(win)
    return presented + closed
`,
	}, "surface frame pixels alias 'pixels' cannot be used after frame 'ctx.frame' was presented")
}

func TestSurfaceDirectHostPresentMarksFramePixelsPresented(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses alloc, mem, surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    var frame: surface.Frame = surface.begin_frame(win)
    var pixels: []u8 = frame.pixels
    let raw_present: Int = core.surface_present_rgba(win.handle, frame.pixels, frame.width, frame.height, frame.stride)
    pixels[0] = 255
    let closed: Int = surface.close(win)
    return raw_present + closed
`,
	}, "surface frame pixels alias 'pixels' cannot be used after frame 'frame' was presented")
}

func TestSurfaceDirectHostPresentChecksFrameSurfaceOwner(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses alloc, mem, surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    var frame: surface.Frame = surface.begin_frame(win)
    let closed: Int = surface.close(win)
    let raw_present: Int = core.surface_present_rgba(frame.surface.handle, frame.pixels, frame.width, frame.height, frame.stride)
    return raw_present + closed
`,
	}, "cannot use consumed value 'win'")
}

func TestSurfaceAliasCannotBeClosedAfterOwnerClose(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    let alias: surface.Surface = win
    let closed: Int = surface.close(win)
    let double_closed: Int = surface.close(alias)
    return closed + double_closed
`,
	}, "cannot use consumed value 'alias'")
}

func TestSurfaceAliasCannotBeUsedAfterOwnerClose(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    let alias: surface.Surface = win
    let closed: Int = surface.close(win)
    let redraw: Int = surface.request_redraw(alias)
    return closed + redraw
`,
	}, "cannot use consumed value 'alias'")
}

func TestSurfaceStructLiteralHandleAliasCannotBeUsedAfterOwnerClose(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    let forged: surface.Surface = surface.Surface(handle: win.handle, width: win.width, height: win.height)
    let closed: Int = surface.close(win)
    let redraw: Int = surface.request_redraw(forged)
    return closed + redraw
`,
	}, "cannot use consumed value 'forged'")
}

func TestSurfaceFrameCannotBePresentedAfterSurfaceClose(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses alloc, mem, surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    let frame: surface.Frame = surface.begin_frame(win)
    let closed: Int = surface.close(win)
    let presented: Int = surface.present(frame)
    return closed + presented
`,
	}, "cannot use consumed value 'win'")
}

func TestSurfaceManualFrameCannotBePresentedAfterSurfaceClose(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses alloc, mem, surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    let pixels: []u8 = core.make_u8(16)
    let frame: surface.Frame = surface.Frame(surface: win, width: 2, height: 2, stride: 8, pixels: pixels)
    let closed: Int = surface.close(win)
    let presented: Int = surface.present(frame)
    return closed + presented
`,
	}, "cannot use consumed value 'win'")
}

func TestSurfaceDrawContextFrameCannotBePresentedAfterSurfaceClose(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface
import lib.core.draw as draw

func main() -> Int
uses alloc, mem, surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    let frame: surface.Frame = surface.begin_frame(win)
    let ctx: draw.DrawContext = draw.DrawContext(frame: frame)
    let closed: Int = surface.close(win)
    let presented: Int = surface.present(ctx.frame)
    return closed + presented
`,
	}, "cannot use consumed value 'win'")
}

func TestSurfaceDrawContextFrameAssignmentTracksNewSurfaceOwner(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface
import lib.core.draw as draw

func main() -> Int
uses alloc, mem, surface:
    let win1: surface.Surface = surface.open("one", 2, 2)
    let frame1: surface.Frame = surface.begin_frame(win1)
    var ctx: draw.DrawContext = draw.DrawContext(frame: frame1)
    let win2: surface.Surface = surface.open("two", 2, 2)
    let frame2: surface.Frame = surface.begin_frame(win2)
    ctx.frame = frame2
    let closed: Int = surface.close(win2)
    let presented: Int = surface.present(ctx.frame)
    let closed1: Int = surface.close(win1)
    return closed + presented + closed1
`,
	}, "cannot use consumed value 'win2'")
}

func TestSurfaceDirectHostCloseConsumesSurfaceHandleOwner(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    let raw_close: Int = core.surface_close(win.handle)
    let redraw: Int = surface.request_redraw(win)
    return raw_close + redraw
`,
	}, "cannot use consumed value 'win'")
}

func TestSurfaceDirectHostCloseConsumesSurfaceHandleIntAliasOwner(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    let handle: Int = win.handle
    let raw_close: Int = core.surface_close(handle)
    let redraw: Int = surface.request_redraw(win)
    return raw_close + redraw
`,
	}, "cannot use consumed value 'win'")
}

func TestSurfaceDirectHostHandleUseAfterCloseRejected(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    let handle: Int = win.handle
    let closed: Int = surface.close(win)
    let redraw: Int = core.surface_request_redraw(handle)
    return closed + redraw
`,
	}, "cannot use consumed value 'win'")
}

func requireSurfaceCheckErrorContains(t *testing.T, files map[string]string, want string) {
	t.Helper()

	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, files)

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt: %v", err)
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got: %v", want, err)
	}
}
