package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const surfaceProjectTemplateModel = "surface-project-template-v1"

var surfaceAppTemplateKinds = []string{
	"command-palette",
	"settings",
	"dashboard",
	"editor-shell",
	"multi-window-notes",
	"web-canvas",
}

type newSurfaceAppOptions struct {
	Template  string
	WriteLock bool
}

type surfaceAppTemplateSpec struct {
	Kind        string
	TitleHash   int
	AccentAlpha int
	Recipes     []string
	Actions     []int
	AppShell    bool
	WebCanvas   bool
}

func runNewSurfaceAppArgs(args []string, stdout io.Writer, stderr io.Writer) int {
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra new surface-app [--template KIND] [--lock] <NameOrPath>")
		fmt.Fprintf(stdout, "templates: %s\n", strings.Join(surfaceAppTemplateKinds, ", "))
		return 0
	}
	var path string
	opt := newSurfaceAppOptions{Template: "command-palette"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--lock":
			opt.WriteLock = true
		case "--template":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "new surface-app --template requires a value")
				return 2
			}
			opt.Template = args[i+1]
			i++
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(stderr, "unknown new surface-app option %q\n", arg)
				return 2
			}
			if path != "" {
				fmt.Fprintln(stderr, "usage: tetra new surface-app [--template KIND] [--lock] <NameOrPath>")
				return 2
			}
			path = arg
		}
	}
	if path == "" {
		fmt.Fprintln(stderr, "usage: tetra new surface-app [--template KIND] [--lock] <NameOrPath>")
		return 2
	}
	return runNewSurfaceApp(path, opt, stdout, stderr)
}

func runNewSurfaceApp(path string, opt newSurfaceAppOptions, stdout io.Writer, stderr io.Writer) int {
	if strings.TrimSpace(path) == "" {
		fmt.Fprintln(stderr, "new surface-app requires a name or path")
		return 2
	}
	spec, ok := surfaceAppTemplateByKind(opt.Template)
	if !ok {
		fmt.Fprintf(stderr, "unknown surface app template %q\n", opt.Template)
		return 2
	}
	targetDir := filepath.Clean(filepath.FromSlash(path))
	if _, err := os.Stat(targetDir); err == nil {
		fmt.Fprintf(stderr, "%s already exists\n", targetDir)
		return 2
	} else if !os.IsNotExist(err) {
		fmt.Fprintln(stderr, err)
		return 1
	}
	name := capsuleNameFromPath(targetDir)
	if name == "" {
		fmt.Fprintln(stderr, "new surface-app requires a valid app name")
		return 2
	}
	target := defaultTarget()
	files := map[string]string{
		"Capsule.t4":            surfaceAppCapsule(name, target),
		"src/main.tetra":        surfaceAppSource(spec),
		"surface-template.json": surfaceAppTemplateMetadata(name, spec, target),
		"README.md":             surfaceAppReadme(name, spec),
		"tests/main_test.tetra": surfaceAppTemplateTestSource(),
		"design/recipes.tetra":  surfaceAppDesignRecipesSource(spec),
		"design/tokens.tetra":   surfaceAppDesignTokensSource(),
	}
	for rel, content := range files {
		full := filepath.Join(targetDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	fmt.Fprintf(stdout, "Created Surface app: %s\n", targetDir)
	fmt.Fprintf(stdout, "Surface template: %s\n", spec.Kind)
	if opt.WriteLock {
		lockPath := filepath.Join(targetDir, "Tetra.lock")
		if err := buildCapsuleArtifacts(filepath.Join(targetDir, "Capsule.t4"), capsuleArtifactBuildOptions{
			LockPath: lockPath,
			Jobs:     1,
		}); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "Created lock: %s\n", lockPath)
	}
	return 0
}

func surfaceAppTemplateByKind(kind string) (surfaceAppTemplateSpec, bool) {
	switch strings.TrimSpace(kind) {
	case "command-palette":
		return surfaceAppTemplateSpec{Kind: "command-palette", TitleHash: 16, AccentAlpha: 44, Recipes: []string{"recipe_region_panel", "recipe_field_text", "recipe_command_item", "recipe_control_action"}, Actions: []int{301, 302, 303}}, true
	case "settings":
		return surfaceAppTemplateSpec{Kind: "settings", TitleHash: 8, AccentAlpha: 30, Recipes: []string{"recipe_form_field", "recipe_field_text", "recipe_tab_item", "recipe_control_action"}, Actions: []int{501, 502, 503}}, true
	case "dashboard":
		return surfaceAppTemplateSpec{Kind: "dashboard", TitleHash: 17, AccentAlpha: 36, Recipes: []string{"recipe_region_panel", "recipe_metric_tile", "recipe_list_row", "recipe_toast_notification"}, Actions: []int{401, 402, 403}}, true
	case "editor-shell":
		return surfaceAppTemplateSpec{Kind: "editor-shell", TitleHash: 12, AccentAlpha: 34, Recipes: []string{"recipe_nav_item", "recipe_tab_item", "recipe_command_item", "recipe_region_panel"}, Actions: []int{601, 602, 603}}, true
	case "multi-window-notes":
		return surfaceAppTemplateSpec{Kind: "multi-window-notes", TitleHash: 9, AccentAlpha: 38, Recipes: []string{"recipe_region_panel", "recipe_list_row", "recipe_field_text", "recipe_control_action"}, Actions: []int{701, 702, 703}, AppShell: true}, true
	case "web-canvas":
		return surfaceAppTemplateSpec{Kind: "web-canvas", TitleHash: 21, AccentAlpha: 40, Recipes: []string{"recipe_region_panel", "recipe_metric_tile", "recipe_command_item", "recipe_field_text"}, Actions: []int{801, 802, 803}, WebCanvas: true}, true
	default:
		return surfaceAppTemplateSpec{}, false
	}
}

func surfaceAppCapsule(name string, target string) string {
	return fmt.Sprintf(`manifest "tetra.capsule.v1"
capsule %s:
    id "tetra://surface-apps/%s"
    version "0.1.0"
    entry "src/main.tetra"
    source "src"
    source "design"
    target "%s"
    target "wasm32-web"
    permission "io"
`, name, capsuleSlug(name), target)
}

func surfaceAppSource(spec surfaceAppTemplateSpec) string {
	imports := `import lib.core.surface as surface
import lib.core.block as block
import lib.core.morph as morph
`
	if spec.AppShell {
		imports += "import lib.core.surface_app_shell as shell\n"
	}
	shellFunc := `func template_shell_score() -> Int:
    return 1
`
	if spec.AppShell {
		shellFunc = `func template_shell_score() -> Int:
    var main_win: shell.ShellWindow = shell.window(1, 5, 560, 420, 1000)
    var detail_win: shell.ShellWindow = shell.window(2, 9, 320, 240, 1000)
    let open_main: Int = shell.open_window(main_win)
    let open_detail: Int = shell.open_window(detail_win)
    let resize_main: Int = shell.resize_window(main_win, 720, 540)
    let cursor: Int = shell.set_cursor(main_win, shell.cursor_text())
    let menu: shell.ShellFeature = shell.scoped_adapter_feature(shell.feature_app_menu())
    let lifecycle: shell.ShellFeature = shell.target_evidenced_feature(shell.feature_window_lifecycle())
    let multi: shell.ShellFeature = shell.target_evidenced_feature(shell.feature_multi_window())
    let blocked: shell.ShellFeature = shell.blocked_pass_feature(shell.feature_notification())
    if open_main == 1 && open_detail == 1 && resize_main == 1 && cursor == 1 && shell.multi_window_ready(main_win, detail_win) && shell.feature_is_honest(menu) && shell.feature_is_honest(lifecycle) && shell.feature_is_honest(multi) && shell.feature_is_honest(blocked):
        return 1
    return 0
`
	}
	targetFunc := `func template_target_score() -> Int:
    return 1
`
	if spec.WebCanvas {
		targetFunc = `func template_target_score() -> Int:
    let size: surface.Size = surface.Size(w: 640, h: 360)
    if size.w == 640 && size.h == 360:
        return 1
    return 0
`
	}
	return fmt.Sprintf(`// Surface %s template authored through Block/Morph recipes.
module main

%s
struct SurfaceTemplateApp:
    template_hash: Int
    recipe_count: Int
    block_count: Int
    shell_score: Int
    target_score: Int

func root_block(rect: surface.Rect) -> block.Block:
    let paint: block.PaintSpec = block.paint_stack2(block.paint_layer_fill_radius(morph.theme_dark(), 0), block.paint_layer_overlay(morph.accent(), %d, 0))
    let props: block.BlockProps = block.props(block.layout_overlay(rect, 0), paint, block.text_none(), block.image_none(), block.input_none(), block.event_none(), block.state_base(), block.motion_none(), block.accessibility_region(%d), block.asset_none())
    return block.make(block.id(1), block.id_none(), props)

%s
%s
func template_recipe_checksum() -> Int:
    let capsule: morph.Capsule = morph.capsule_default()
    let first_recipe: morph.Recipe = morph.%s()
    let second_recipe: morph.Recipe = morph.%s()
    let third_recipe: morph.Recipe = morph.%s()
    let fourth_recipe: morph.Recipe = morph.%s()
    let first_expansion: morph.RecipeExpansion = morph.recipe_expansion(first_recipe, block.id(2))
    let second_expansion: morph.RecipeExpansion = morph.recipe_expansion(second_recipe, block.id(4))
    let third_expansion: morph.RecipeExpansion = morph.recipe_expansion(third_recipe, block.id(5))
    let fourth_expansion: morph.RecipeExpansion = morph.recipe_expansion(fourth_recipe, block.id(6))
    if morph.capsule_valid(capsule) && morph.expansion_valid(first_expansion) && morph.expansion_valid(second_expansion) && morph.expansion_valid(third_expansion) && morph.expansion_valid(fourth_expansion) && morph.recipe_expands_to_block(first_recipe) && morph.recipe_expands_to_block(second_recipe) && morph.recipe_expands_to_block(third_recipe) && morph.recipe_expands_to_block(fourth_recipe):
        return capsule.token_graph_hash + first_recipe.name_hash + second_recipe.name_hash + third_recipe.name_hash + fourth_recipe.name_hash
    return 0

func main() -> Int
uses alloc, mem:
    let root_rect: surface.Rect = surface.Rect(x: 0, y: 0, w: 420, h: 280)
    let panel_rect: surface.Rect = surface.Rect(x: 20, y: 20, w: 380, h: 240)
    let label_rect: surface.Rect = surface.Rect(x: 32, y: 24, w: 160, h: 18)
    let field_rect: surface.Rect = surface.Rect(x: 32, y: 56, w: 344, h: 42)
    let primary_rect: surface.Rect = surface.Rect(x: 32, y: 112, w: 344, h: 44)
    let secondary_rect: surface.Rect = surface.Rect(x: 32, y: 164, w: 344, h: 44)
    let tertiary_rect: surface.Rect = surface.Rect(x: 32, y: 216, w: 344, h: 28)
    var tree: block.BlockTree = block.tree_init(8)
    let root: Int = block.tree_add_root(tree, root_block(root_rect), root_rect)
    let panel: Int = block.tree_add_child(tree, block.id(1), morph.expand_region_panel(2, 1, panel_rect, %d), panel_rect)
    let label: Int = block.tree_add_child(tree, block.id(2), morph.expand_label(3, 2, 4, label_rect, 6), label_rect)
    let field: Int = block.tree_add_child(tree, block.id(2), morph.expand_field_text(4, 2, 3, field_rect, 18, 8), field_rect)
    let primary: Int = block.tree_add_child(tree, block.id(2), morph.expand_control_action(5, 2, primary_rect, 13, %d, true), primary_rect)
    let secondary: Int = block.tree_add_child(tree, block.id(2), morph.expand_control_action(6, 2, secondary_rect, 17, %d, false), secondary_rect)
    let tertiary: Int = block.tree_add_child(tree, block.id(2), morph.expand_control_action(7, 2, tertiary_rect, 12, %d, false), tertiary_rect)
    let app: SurfaceTemplateApp = SurfaceTemplateApp(template_hash: template_recipe_checksum(), recipe_count: 4, block_count: block.tree_len(tree), shell_score: template_shell_score(), target_score: template_target_score())
    let valid: Int = block.tree_validate(tree)
    let focus0: Int = block.id_value(block.focus_order_at(tree, 0))
    let focus1: Int = block.id_value(block.focus_order_at(tree, 1))
    let a11y0: Int = block.id_value(block.tree_accessibility_order_at(tree, 0))
    if root == 1 && panel == 2 && label == 3 && field == 4 && primary == 5 && secondary == 6 && tertiary == 7 && valid == block.tree_error_ok() && focus0 == 4 && focus1 == 5 && a11y0 == 1 && app.template_hash > 0 && app.recipe_count == 4 && app.block_count == 7 && app.shell_score == 1 && app.target_score == 1 && morph.capsule_self_check() && morph.accessibility_projection_ok(3, 4, 6) && morph.memory_budget_ok(app.block_count, 3, 64 * 64 * 4):
        return 0
    return 1
`, spec.Kind, imports, spec.AccentAlpha, spec.TitleHash, shellFunc, targetFunc, spec.Recipes[0], spec.Recipes[1], spec.Recipes[2], spec.Recipes[3], spec.TitleHash, spec.Actions[0], spec.Actions[1], spec.Actions[2])
}

func surfaceAppTemplateMetadata(name string, spec surfaceAppTemplateSpec, target string) string {
	imports := []string{"lib.core.surface", "lib.core.block", "lib.core.morph"}
	if spec.AppShell {
		imports = append(imports, "lib.core.surface_app_shell")
	}
	return fmt.Sprintf(`{
  "schema": "tetra.surface.project-template.v1",
  "model": "%s",
  "template": "%s",
  "app": "%s",
  "release_scope": "surface-v1-linux-web",
  "entry": "src/main.tetra",
  "targets": ["%s", "wasm32-web"],
  "imports": [%s],
  "block_morph_only": true,
  "uses_app_shell": %t,
  "web_canvas": %t,
  "negative_guards": {
    "no_react_import": true,
    "no_electron_import": true,
    "no_dom_app_ui_tree": true,
    "no_css_runtime": true,
    "no_core_widgets": true,
    "no_platform_widgets": true,
    "no_user_js_app_logic": true
  }
}
`, surfaceProjectTemplateModel, spec.Kind, name, target, quotedJSONList(imports), spec.AppShell, spec.WebCanvas)
}

func surfaceAppReadme(name string, spec surfaceAppTemplateSpec) string {
	return fmt.Sprintf(`# %s

Surface template: %s

This project is authored with `+"`Block`"+` and `+"`Morph`"+` recipes.

`+"```bash"+`
tetra check .
tetra build --target linux-x64 .
tetra run --target linux-x64 .
tetra build --target wasm32-web .
`+"```"+`
`, name, spec.Kind)
}

func surfaceAppTemplateTestSource() string {
	return `test "surface template math sanity":
    expect 40 + 2 == 42
`
}

func surfaceAppDesignRecipesSource(spec surfaceAppTemplateSpec) string {
	return fmt.Sprintf(`// Recipe catalog marker for the %s Surface template.
func template_recipe_count() -> Int:
    return 4
`, spec.Kind)
}

func surfaceAppDesignTokensSource() string {
	return `// Token catalog marker consumed by Surface template smoke evidence.
func template_token_count() -> Int:
    return 3
`
}

func quotedJSONList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	return strings.Join(quoted, ", ")
}
