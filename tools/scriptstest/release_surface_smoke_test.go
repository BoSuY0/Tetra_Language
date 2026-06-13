package scriptstest

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestReleaseSurfaceSmokeScriptsUseStrictReleaseValidation(t *testing.T) {
	root := repoRoot(t)
	scripts, err := filepath.Glob(filepath.Join(root, "scripts", "release", "surface", "*release*.sh"))
	if err != nil {
		t.Fatalf("glob Surface release scripts: %v", err)
	}
	if len(scripts) == 0 {
		t.Fatalf("no Surface release scripts matched")
	}
	for _, script := range scripts {
		raw, err := os.ReadFile(script)
		if err != nil {
			t.Fatalf("read Surface release script %s: %v", script, err)
		}
		text := string(raw)
		if strings.Contains(text, "validate-surface-runtime") && !strings.Contains(text, "--release ") {
			t.Fatalf("Surface release script %s validates runtime reports without a strict --release selector", script)
		}
		if strings.Contains(text, `--report ""`) {
			t.Fatalf("Surface release script %s has empty --report argument", script)
		}
	}
}

func TestReleaseSurfaceAPIStabilityGateDocumentsStableAPIChecks(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "surface", "api-stability-gate.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Surface API stability gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/api-stability-gate.sh [--report-dir DIR]",
		"lib.core.surface",
		"lib.core.draw",
		"lib.core.component",
		"lib.core.widgets",
		"lib.core.accessibility",
		"lib.core.text",
		"lib.core.style",
		"rg -n '^module lib\\.core\\.(surface|draw|component|widgets|accessibility|text|style)$' lib/core/*.tetra",
		"rg -n '^module lib\\.core\\..*(experimental|v[0-9]+|_[vV][0-9]+)' lib/core/*.tetra",
		"rg -n '^import lib\\.experimental(\\.|$)' examples/surface_release_*.tetra",
		"./tetra doc",
		"go run ./tools/cmd/validate-api-docs --docs",
		"go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json",
		"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
		"public-surface-api-summary.txt",
		`"public_api_summary": "public-surface-api-summary.txt"`,
		"surface-api-stability-summary.json",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Surface API stability gate missing %q", want)
		}
	}
}

func TestReleaseSurfaceExamplesExistAndUseStableCoreModules(t *testing.T) {
	root := repoRoot(t)
	examples := []struct {
		path string
		want []string
	}{
		{
			path: "examples/surface_release_counter.tetra",
			want: []string{
				"import lib.core.widgets as widgets",
				"import lib.core.style as style",
				"import lib.core.accessibility as accessibility",
				"examples.surface_release_counter",
				"surface.open",
				"surface.present",
				"event_mouse_up",
				"event_key_down",
				"event_resize",
				"accessibility.value_name",
			},
		},
		{
			path: "examples/surface_release_form.tetra",
			want: []string{
				"import lib.core.widgets as widgets",
				"import lib.core.style as style",
				"widgets.add_textbox",
				"widgets.add_checkbox",
				"widgets.add_scroll",
				"widgets.status_text_init",
			},
		},
		{
			path: "examples/surface_release_text_input.tetra",
			want: []string{
				"import lib.core.text as text",
				"clipboard_write_text",
				"clipboard_read_text_into",
				"poll_composition",
				"composition_clear",
			},
		},
		{
			path: "examples/surface_release_accessibility.tetra",
			want: []string{
				"import lib.core.widgets as widgets",
				"import lib.core.accessibility as accessibility",
				"add_accessible_textbox",
				"add_accessible_button",
				"validate_settings_counts",
			},
		},
	}
	for _, example := range examples {
		raw, err := os.ReadFile(filepath.Join(root, example.path))
		if err != nil {
			t.Fatalf("read Surface release example %s: %v", example.path, err)
		}
		text := string(raw)
		for _, want := range example.want {
			if !strings.Contains(text, want) {
				t.Fatalf("Surface release example %s missing %q", example.path, want)
			}
		}
		if strings.Contains(text, "import lib.experimental.") {
			t.Fatalf("Surface release example %s imports lib.experimental.*", example.path)
		}
	}
}

func TestReleaseSurfaceExamplesRejectFakePromotionSources(t *testing.T) {
	root := repoRoot(t)
	examples, err := filepath.Glob(filepath.Join(root, "examples", "surface_release_*.tetra"))
	if err != nil {
		t.Fatalf("glob Surface release examples: %v", err)
	}
	if len(examples) == 0 {
		t.Fatalf("no Surface release examples found")
	}

	localDemoWidgetStruct := regexp.MustCompile(`(?m)^struct\s+\w*(Button|TextBox|Row|Column|Panel|Scroll|Checkbox)\w*\s*:`)
	manualTreeWrite := regexp.MustCompile(`(?m)\.(id|parent_id|rect|first_child|child_count|flags)\s*=`)
	for _, path := range examples {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatalf("relpath Surface release example %s: %v", path, err)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read Surface release example %s: %v", rel, err)
		}
		text := string(raw)
		lower := strings.ToLower(text)
		for _, want := range []string{
			"import lib.core.widgets as widgets",
			"import lib.core.style as style",
		} {
			if !strings.Contains(text, want) {
				t.Fatalf("Surface release example %s missing anti-fake import %q", rel, want)
			}
		}
		if strings.Contains(rel, "text_input") && !strings.Contains(text, "import lib.core.text as text") {
			t.Fatalf("Surface text/input release example %s must import lib.core.text", rel)
		}
		if strings.Contains(text, "accessibility.") && !strings.Contains(text, "import lib.core.accessibility as accessibility") {
			t.Fatalf("Surface accessibility release example %s must import lib.core.accessibility", rel)
		}
		if match := localDemoWidgetStruct.FindString(text); match != "" {
			t.Fatalf("Surface release example %s defines local demo widget struct %q", rel, strings.TrimSpace(match))
		}
		if strings.Contains(text, "component.TreeNode(") {
			t.Fatalf("Surface release example %s manually constructs component.TreeNode structural evidence", rel)
		}
		if match := manualTreeWrite.FindString(text); match != "" {
			t.Fatalf("Surface release example %s writes TreeNode structural field %q manually", rel, match)
		}
		for _, forbidden := range []string{
			"react",
			"vue",
			"svelte",
			"dom ui",
			"user js",
			"user javascript",
			".ui.json",
			".ui.html",
			".ui.web.mjs",
		} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("Surface release example %s contains forbidden fake-promotion marker %q", rel, forbidden)
			}
		}
	}
}

func TestReleaseSurfaceSmokeScriptsDocumentHeadlessAndPendingLinuxX64Gates(t *testing.T) {
	root := repoRoot(t)
	headlessPath := filepath.Join(root, "scripts", "release", "surface", "surface-headless-smoke.sh")
	headlessRaw, err := os.ReadFile(headlessPath)
	if err != nil {
		t.Fatalf("read Surface headless smoke script: %v", err)
	}
	headless := string(headlessRaw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-headless-smoke.sh [--report-dir DIR]",
		"surface-headless.json",
		"go run ./tools/cmd/surface-runtime-smoke --mode headless",
		"go run ./tools/cmd/validate-surface-runtime",
		"tetra.surface.runtime.v1",
	} {
		if !strings.Contains(headless, want) {
			t.Fatalf("Surface headless smoke script missing %q", want)
		}
	}

	linuxPath := filepath.Join(root, "scripts", "release", "surface", "surface-linux-x64-smoke.sh")
	linuxRaw, err := os.ReadFile(linuxPath)
	if err != nil {
		t.Fatalf("read Surface linux-x64 smoke script: %v", err)
	}
	linux := string(linuxRaw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-linux-x64-smoke.sh [--report-dir DIR]",
		"surface-linux-x64.json",
		"go run ./tools/cmd/surface-runtime-smoke --mode linux-x64",
		"go run ./tools/cmd/validate-surface-runtime",
		"real Linux-x64 Surface host",
		"headless/stub evidence is not accepted",
	} {
		if !strings.Contains(linux, want) {
			t.Fatalf("Surface linux-x64 smoke script missing %q", want)
		}
	}

	wasmPath := filepath.Join(root, "scripts", "release", "surface", "surface-wasm32-web-smoke.sh")
	wasmRaw, err := os.ReadFile(wasmPath)
	if err != nil {
		t.Fatalf("read Surface wasm32-web smoke script: %v", err)
	}
	wasm := string(wasmRaw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-wasm32-web-smoke.sh [--report-dir DIR]",
		"surface-wasm32-web.json",
		"go run ./tools/cmd/surface-runtime-smoke --mode wasm32-web",
		"go run ./tools/cmd/validate-surface-runtime",
		"go run ./tools/cmd/validate-wasm-imports",
		"compiler-owned wasm Surface loader",
		"legacy UI sidecars are not accepted",
	} {
		if !strings.Contains(wasm, want) {
			t.Fatalf("Surface wasm32-web smoke script missing %q", want)
		}
	}

	realWindowPath := filepath.Join(root, "scripts", "release", "surface", "surface-linux-x64-real-window-smoke.sh")
	realWindowRaw, err := os.ReadFile(realWindowPath)
	if err != nil {
		t.Fatalf("read Surface linux-x64 real-window smoke script: %v", err)
	}
	realWindow := string(realWindowRaw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-linux-x64-real-window-smoke.sh [--report-dir DIR]",
		"surface-linux-x64-real-window.json",
		"go run ./tools/cmd/surface-runtime-smoke --mode linux-x64-real-window",
		"--source examples/surface_release_counter.tetra",
		"go run ./tools/cmd/validate-surface-runtime",
		"linux-x64-real-window",
		"real Linux window",
	} {
		if !strings.Contains(realWindow, want) {
			t.Fatalf("Surface linux-x64 real-window smoke script missing %q", want)
		}
	}

	browserCanvasPath := filepath.Join(root, "scripts", "release", "surface", "surface-wasm32-web-browser-canvas-smoke.sh")
	browserCanvasRaw, err := os.ReadFile(browserCanvasPath)
	if err != nil {
		t.Fatalf("read Surface wasm32-web browser-canvas smoke script: %v", err)
	}
	browserCanvas := string(browserCanvasRaw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-wasm32-web-browser-canvas-smoke.sh [--report-dir DIR]",
		"surface-wasm32-web-browser-canvas.json",
		"go run ./tools/cmd/surface-runtime-smoke --mode wasm32-web-browser-canvas",
		"--source examples/surface_release_counter.tetra",
		"go run ./tools/cmd/validate-surface-runtime",
		"go run ./tools/cmd/validate-wasm-imports",
		"real",
		"browser canvas",
		"pointer/key/resize/text input evidence",
		"Node-only",
	} {
		if !strings.Contains(browserCanvas, want) {
			t.Fatalf("Surface wasm32-web browser-canvas smoke script missing %q", want)
		}
	}

	textScripts := []struct {
		path string
		want []string
	}{
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-headless-text-focus-input-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-headless-text-focus-input-smoke.sh [--report-dir DIR]",
				"surface-headless-text-focus-input.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode headless-text-focus-input",
				"--source examples/surface_textbox_app.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"click focus",
				"backspace/delete",
			},
		},
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-linux-x64-real-window-text-focus-input-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-linux-x64-real-window-text-focus-input-smoke.sh [--report-dir DIR]",
				"surface-linux-x64-real-window-text-focus-input.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode linux-x64-real-window-text-focus-input",
				"--source examples/surface_textbox_app.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"real-window",
				"native-input",
			},
		},
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-wasm32-web-browser-canvas-text-focus-input-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-wasm32-web-browser-canvas-text-focus-input-smoke.sh [--report-dir DIR]",
				"surface-wasm32-web-browser-canvas-text-focus-input.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode wasm32-web-browser-canvas-text-focus-input",
				"--source examples/surface_textbox_app.tetra",
				"go run ./tools/cmd/validate-wasm-imports",
				"go run ./tools/cmd/validate-surface-runtime",
				"beforeinput",
				"Node-only",
			},
		},
	}
	for _, script := range textScripts {
		raw, err := os.ReadFile(script.path)
		if err != nil {
			t.Fatalf("read Surface text-focus-input smoke script %s: %v", script.path, err)
		}
		text := string(raw)
		for _, want := range script.want {
			if !strings.Contains(text, want) {
				t.Fatalf("Surface text-focus-input smoke script %s missing %q", script.path, want)
			}
		}
	}

	releaseTextInputScripts := []struct {
		path string
		want []string
	}{
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-headless-release-text-input-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-headless-release-text-input-smoke.sh [--report-dir DIR]",
				"surface-headless-release-text-input.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode headless-release-text-input",
				"--source examples/surface_release_text_input.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"tetra.surface.text-input.v1",
				"owned UTF-8 buffer",
			},
		},
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-linux-x64-release-text-input-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-linux-x64-release-text-input-smoke.sh [--report-dir DIR]",
				"surface-linux-x64-release-text-input.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode linux-x64-release-text-input",
				"--source examples/surface_release_text_input.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"real-window",
				"platform clipboard or platform IME completion",
			},
		},
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-wasm32-web-release-text-input-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-wasm32-web-release-text-input-smoke.sh [--report-dir DIR]",
				"surface-wasm32-web-release-text-input.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode wasm32-web-release-text-input",
				"--source examples/surface_release_text_input.tetra",
				"surface-release-text-input.wasm",
				"go run ./tools/cmd/validate-wasm-imports",
				"go run ./tools/cmd/validate-surface-runtime",
				"tetra.surface.text-input.v1",
			},
		},
	}
	for _, script := range releaseTextInputScripts {
		raw, err := os.ReadFile(script.path)
		if err != nil {
			t.Fatalf("read Surface release text-input smoke script %s: %v", script.path, err)
		}
		text := string(raw)
		for _, want := range script.want {
			if !strings.Contains(text, want) {
				t.Fatalf("Surface release text-input smoke script %s missing %q", script.path, want)
			}
		}
	}

	componentTreeScripts := []struct {
		path string
		want []string
	}{
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-headless-component-tree-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-headless-component-tree-smoke.sh [--report-dir DIR]",
				"surface-headless-component-tree.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode headless-component-tree",
				"--source examples/surface_tree_app.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"root-to-leaf dispatch paths",
				"resize relayout",
			},
		},
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-linux-x64-real-window-component-tree-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-linux-x64-real-window-component-tree-smoke.sh [--report-dir DIR]",
				"surface-linux-x64-real-window-component-tree.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode linux-x64-real-window-component-tree",
				"--source examples/surface_tree_app.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"real Linux window",
				"native-input",
			},
		},
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-wasm32-web-browser-canvas-component-tree-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-wasm32-web-browser-canvas-component-tree-smoke.sh [--report-dir DIR]",
				"surface-wasm32-web-browser-canvas-component-tree.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode wasm32-web-browser-canvas-component-tree",
				"--source examples/surface_tree_app.tetra",
				"go run ./tools/cmd/validate-wasm-imports",
				"go run ./tools/cmd/validate-surface-runtime",
				"beforeinput",
				"Node-only",
			},
		},
	}
	for _, script := range componentTreeScripts {
		raw, err := os.ReadFile(script.path)
		if err != nil {
			t.Fatalf("read Surface component-tree smoke script %s: %v", script.path, err)
		}
		text := string(raw)
		for _, want := range script.want {
			if !strings.Contains(text, want) {
				t.Fatalf("Surface component-tree smoke script %s missing %q", script.path, want)
			}
		}

		componentTreeAPIScripts := []struct {
			path string
			want []string
		}{
			{
				path: filepath.Join(root, "scripts", "release", "surface", "surface-headless-component-tree-api-smoke.sh"),
				want: []string{
					"Usage: bash scripts/release/surface/surface-headless-component-tree-api-smoke.sh [--report-dir DIR]",
					"surface-headless-component-tree-api.json",
					"go run ./tools/cmd/surface-runtime-smoke --mode headless-component-tree-api",
					"--source examples/surface_tree_app.tetra",
					"go run ./tools/cmd/validate-surface-runtime",
					"component_tree_api",
					"manual_bookkeeping=false",
				},
			},
			{
				path: filepath.Join(root, "scripts", "release", "surface", "surface-linux-x64-real-window-component-tree-api-smoke.sh"),
				want: []string{
					"Usage: bash scripts/release/surface/surface-linux-x64-real-window-component-tree-api-smoke.sh [--report-dir DIR]",
					"surface-linux-x64-real-window-component-tree-api.json",
					"go run ./tools/cmd/surface-runtime-smoke --mode linux-x64-real-window-component-tree-api",
					"--source examples/surface_tree_app.tetra",
					"go run ./tools/cmd/validate-surface-runtime",
					"real Linux window",
					"component_tree_api",
				},
			},
			{
				path: filepath.Join(root, "scripts", "release", "surface", "surface-wasm32-web-browser-canvas-component-tree-api-smoke.sh"),
				want: []string{
					"Usage: bash scripts/release/surface/surface-wasm32-web-browser-canvas-component-tree-api-smoke.sh [--report-dir DIR]",
					"surface-wasm32-web-browser-canvas-component-tree-api.json",
					"go run ./tools/cmd/surface-runtime-smoke --mode wasm32-web-browser-canvas-component-tree-api",
					"--source examples/surface_tree_app.tetra",
					"go run ./tools/cmd/validate-wasm-imports",
					"go run ./tools/cmd/validate-surface-runtime",
					"browser canvas",
					"component_tree_api",
					"Node-only",
				},
			},
		}
		for _, script := range componentTreeAPIScripts {
			raw, err := os.ReadFile(script.path)
			if err != nil {
				t.Fatalf("read Surface component-tree API smoke script %s: %v", script.path, err)
			}
			text := string(raw)
			for _, want := range script.want {
				if !strings.Contains(text, want) {
					t.Fatalf("Surface component-tree API smoke script %s missing %q", script.path, want)
				}
			}
		}
	}

	blockSystemScripts := []struct {
		path string
		want []string
	}{
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-headless-block-system-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-headless-block-system-smoke.sh [--report-dir DIR]",
				"surface-headless-block-system.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode headless-block-system",
				"--source examples/surface_block_system.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"go run ./tools/cmd/validate-surface-block-examples",
				"surface-block-examples.json",
				"examples/surface_block_command_palette.tetra",
				"examples/surface_block_project_dashboard.tetra",
				"examples/surface_block_settings.tetra",
				"examples/surface_block_editor_shell.tetra",
				"examples/surface_block_glass_panel.tetra",
				"tetra.surface.block-system.v1",
				"golden/checksum",
			},
		},
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-linux-x64-real-window-block-system-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-linux-x64-real-window-block-system-smoke.sh [--report-dir DIR]",
				"surface-block-system-linux-x64.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode linux-x64-real-window-block-system",
				"--source examples/surface_block_system.tetra",
				"go run ./tools/cmd/validate-surface-block-report",
				"real Linux window",
				"tetra.surface.block-system.v1",
				"WAYLAND_DISPLAY",
				"blocked",
			},
		},
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-wasm32-web-browser-canvas-block-system-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-wasm32-web-browser-canvas-block-system-smoke.sh [--report-dir DIR]",
				"surface-block-system-wasm32-web.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode wasm32-web-browser-canvas-block-system",
				"--source examples/surface_block_system.tetra",
				"go run ./tools/cmd/validate-wasm-imports",
				"go run ./tools/cmd/validate-surface-block-report",
				"browser canvas",
				"RGBA readback",
				"compiler-owned loader",
				"no user JS",
				"no DOM UI",
				"Node-only",
			},
		},
	}
	for _, script := range blockSystemScripts {
		raw, err := os.ReadFile(script.path)
		if err != nil {
			t.Fatalf("read Surface Block-system smoke script %s: %v", script.path, err)
		}
		text := string(raw)
		for _, want := range script.want {
			if !strings.Contains(text, want) {
				t.Fatalf("Surface Block-system smoke script %s missing %q", script.path, want)
			}
		}
	}

	minimalToolkitScripts := []struct {
		path string
		want []string
	}{
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-headless-minimal-toolkit-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-headless-minimal-toolkit-smoke.sh [--report-dir DIR]",
				"surface-headless-minimal-toolkit.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode headless-minimal-toolkit",
				"--source examples/surface_toolkit_form.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"tetra.surface.toolkit.v1",
				"minimal-widgets-v1",
			},
		},
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-linux-x64-real-window-minimal-toolkit-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-linux-x64-real-window-minimal-toolkit-smoke.sh [--report-dir DIR]",
				"surface-linux-x64-real-window-minimal-toolkit.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode linux-x64-real-window-minimal-toolkit",
				"--source examples/surface_toolkit_form.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"real Linux window",
				"tetra.surface.toolkit.v1",
			},
		},
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-wasm32-web-browser-canvas-minimal-toolkit-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-wasm32-web-browser-canvas-minimal-toolkit-smoke.sh [--report-dir DIR]",
				"surface-wasm32-web-browser-canvas-minimal-toolkit.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode wasm32-web-browser-canvas-minimal-toolkit",
				"--source examples/surface_toolkit_form.tetra",
				"go run ./tools/cmd/validate-wasm-imports",
				"go run ./tools/cmd/validate-surface-runtime",
				"browser canvas",
				"tetra.surface.toolkit.v1",
				"Node-only",
			},
		},
	}
	for _, script := range minimalToolkitScripts {
		raw, err := os.ReadFile(script.path)
		if err != nil {
			t.Fatalf("read Surface minimal toolkit smoke script %s: %v", script.path, err)
		}
		text := string(raw)
		for _, want := range script.want {
			if !strings.Contains(text, want) {
				t.Fatalf("Surface minimal toolkit smoke script %s missing %q", script.path, want)
			}
		}
	}

	toolkitReuseScripts := []struct {
		path string
		want []string
	}{
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-headless-toolkit-reuse-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-headless-toolkit-reuse-smoke.sh [--report-dir DIR]",
				"surface-headless-toolkit-reuse.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode headless-toolkit-reuse",
				"--source examples/surface_toolkit_settings.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"tetra.surface.toolkit.v1",
				"toolkit-reuse-v1",
			},
		},
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-linux-x64-real-window-toolkit-reuse-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-linux-x64-real-window-toolkit-reuse-smoke.sh [--report-dir DIR]",
				"surface-linux-x64-real-window-toolkit-reuse.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode linux-x64-real-window-toolkit-reuse",
				"--source examples/surface_toolkit_settings.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"real Linux window",
				"tetra.surface.toolkit.v1",
			},
		},
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-wasm32-web-browser-canvas-toolkit-reuse-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-wasm32-web-browser-canvas-toolkit-reuse-smoke.sh [--report-dir DIR]",
				"surface-wasm32-web-browser-canvas-toolkit-reuse.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode wasm32-web-browser-canvas-toolkit-reuse",
				"--source examples/surface_toolkit_settings.tetra",
				"go run ./tools/cmd/validate-wasm-imports",
				"go run ./tools/cmd/validate-surface-runtime",
				"browser canvas",
				"tetra.surface.toolkit.v1",
				"Node-only",
			},
		},
	}
	for _, script := range toolkitReuseScripts {
		raw, err := os.ReadFile(script.path)
		if err != nil {
			t.Fatalf("read Surface toolkit reuse smoke script %s: %v", script.path, err)
		}
		text := string(raw)
		for _, want := range script.want {
			if !strings.Contains(text, want) {
				t.Fatalf("Surface toolkit reuse smoke script %s missing %q", script.path, want)
			}
		}
	}

	releaseToolkitScripts := []struct {
		path string
		want []string
	}{
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-headless-release-toolkit-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-headless-release-toolkit-smoke.sh [--report-dir DIR]",
				"surface-headless-release-toolkit.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode headless-release-toolkit",
				"--source examples/surface_release_form.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"tetra.surface.toolkit.v1",
				"production-widgets-v1",
			},
		},
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-linux-x64-release-toolkit-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-linux-x64-release-toolkit-smoke.sh [--report-dir DIR]",
				"surface-linux-x64-release-toolkit.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode linux-x64-release-toolkit",
				"--source examples/surface_release_form.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"real Linux window",
				"tetra.surface.toolkit.v1",
				"production-widgets-v1",
			},
		},
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-wasm32-web-release-toolkit-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-wasm32-web-release-toolkit-smoke.sh [--report-dir DIR]",
				"surface-wasm32-web-release-toolkit.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode wasm32-web-release-toolkit",
				"--source examples/surface_release_form.tetra",
				"go run ./tools/cmd/validate-wasm-imports",
				"go run ./tools/cmd/validate-surface-runtime",
				"browser canvas",
				"tetra.surface.toolkit.v1",
				"Node-only",
			},
		},
	}
	for _, script := range releaseToolkitScripts {
		raw, err := os.ReadFile(script.path)
		if err != nil {
			t.Fatalf("read Surface release toolkit smoke script %s: %v", script.path, err)
		}
		text := string(raw)
		for _, want := range script.want {
			if !strings.Contains(text, want) {
				t.Fatalf("Surface release toolkit smoke script %s missing %q", script.path, want)
			}
		}
	}

	releaseBrowserScripts := []struct {
		path string
		want []string
	}{
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-wasm32-web-release-browser-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-wasm32-web-release-browser-smoke.sh [--report-dir DIR]",
				"surface-wasm32-web-release-browser.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode wasm32-web-release-browser",
				"--source examples/surface_release_form.tetra",
				"surface-release-form.wasm",
				"go run ./tools/cmd/validate-wasm-imports",
				"go run ./tools/cmd/validate-surface-runtime",
				"wasm32-web-browser-canvas-release-v1",
				"browser accessibility snapshot/mirror",
				"Node-only",
			},
		},
	}
	for _, script := range releaseBrowserScripts {
		raw, err := os.ReadFile(script.path)
		if err != nil {
			t.Fatalf("read Surface release browser smoke script %s: %v", script.path, err)
		}
		text := string(raw)
		for _, want := range script.want {
			if !strings.Contains(text, want) {
				t.Fatalf("Surface release browser smoke script %s missing %q", script.path, want)
			}
		}
	}

	releaseWindowScripts := []struct {
		path string
		want []string
	}{
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-linux-x64-release-window-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-linux-x64-release-window-smoke.sh [--report-dir DIR]",
				"surface-linux-x64-release-window.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode linux-x64-release-window",
				"--source examples/surface_release_form.tetra",
				"linux-x64-release-window-v1",
				"wayland-shm-rgba-release-v1",
				"go run ./tools/cmd/validate-surface-runtime",
				"clipboard",
				"composition",
				"accessibility bridge",
				"memfd starter",
			},
		},
	}
	for _, script := range releaseWindowScripts {
		raw, err := os.ReadFile(script.path)
		if err != nil {
			t.Fatalf("read Surface linux release window smoke script %s: %v", script.path, err)
		}
		text := string(raw)
		for _, want := range script.want {
			if !strings.Contains(text, want) {
				t.Fatalf("Surface linux release window smoke script %s missing %q", script.path, want)
			}
		}
	}

	releaseAccessibilityScripts := []struct {
		path string
		want []string
	}{
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-headless-release-accessibility-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-headless-release-accessibility-smoke.sh [--report-dir DIR]",
				"surface-headless-release-accessibility.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode headless-release-accessibility",
				"--source examples/surface_release_accessibility.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"tetra.surface.accessibility-tree.v1",
				"platform-bridge-v1",
			},
		},
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-linux-x64-release-accessibility-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-linux-x64-release-accessibility-smoke.sh [--report-dir DIR]",
				"surface-linux-x64-release-accessibility.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode linux-x64-release-accessibility",
				"--source examples/surface_release_accessibility.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"linux_accessibility_host_bridge_v1",
				"linux accessibility platform probe",
			},
		},
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-wasm32-web-release-accessibility-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-wasm32-web-release-accessibility-smoke.sh [--report-dir DIR]",
				"surface-wasm32-web-release-accessibility.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode wasm32-web-release-accessibility",
				"--source examples/surface_release_accessibility.tetra",
				"surface-release-accessibility.wasm",
				"go run ./tools/cmd/validate-wasm-imports",
				"go run ./tools/cmd/validate-surface-runtime",
				"browser accessibility snapshot/mirror",
				"Node-only",
			},
		},
	}
	for _, script := range releaseAccessibilityScripts {
		raw, err := os.ReadFile(script.path)
		if err != nil {
			t.Fatalf("read Surface release accessibility smoke script %s: %v", script.path, err)
		}
		text := string(raw)
		for _, want := range script.want {
			if !strings.Contains(text, want) {
				t.Fatalf("Surface release accessibility smoke script %s missing %q", script.path, want)
			}
		}
	}

	accessibilityMetadataScripts := []struct {
		path string
		want []string
	}{
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-headless-accessibility-metadata-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-headless-accessibility-metadata-smoke.sh [--report-dir DIR]",
				"surface-headless-accessibility-metadata.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode headless-accessibility-metadata",
				"--source examples/surface_accessibility_settings.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"tetra.surface.accessibility-tree.v1",
				"platform accessibility",
			},
		},
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-linux-x64-real-window-accessibility-metadata-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-linux-x64-real-window-accessibility-metadata-smoke.sh [--report-dir DIR]",
				"surface-linux-x64-real-window-accessibility-metadata.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode linux-x64-real-window-accessibility-metadata",
				"--source examples/surface_accessibility_settings.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"native window/input",
				"tetra.surface.accessibility-tree.v1",
			},
		},
		{
			path: filepath.Join(root, "scripts", "release", "surface", "surface-wasm32-web-browser-canvas-accessibility-metadata-smoke.sh"),
			want: []string{
				"Usage: bash scripts/release/surface/surface-wasm32-web-browser-canvas-accessibility-metadata-smoke.sh [--report-dir DIR]",
				"surface-wasm32-web-browser-canvas-accessibility-metadata.json",
				"go run ./tools/cmd/surface-runtime-smoke --mode wasm32-web-browser-canvas-accessibility-metadata",
				"--source examples/surface_accessibility_settings.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"browser canvas",
				"tetra.surface.accessibility-tree.v1",
				"Node-only",
			},
		},
	}
	for _, script := range accessibilityMetadataScripts {
		raw, err := os.ReadFile(script.path)
		if err != nil {
			t.Fatalf("read Surface accessibility metadata smoke script %s: %v", script.path, err)
		}
		text := string(raw)
		for _, want := range script.want {
			if !strings.Contains(text, want) {
				t.Fatalf("Surface accessibility metadata smoke script %s missing %q", script.path, want)
			}
		}
	}
}

func TestReleaseSurfaceGateRunsAllSurfaceEvidenceSlices(t *testing.T) {
	root := repoRoot(t)
	gatePath := filepath.Join(root, "scripts", "release", "surface", "gate.sh")
	raw, err := os.ReadFile(gatePath)
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/gate.sh [--report-dir DIR]",
		"surface-headless-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-real-window-smoke.sh --report-dir \"$report_dir\"",
		"surface-wasm32-web-smoke.sh --report-dir \"$report_dir\"",
		"surface-wasm32-web-browser-canvas-smoke.sh --report-dir \"$report_dir\"",
		"surface-headless-text-focus-input-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-real-window-text-focus-input-smoke.sh --report-dir \"$report_dir\"",
		"surface-wasm32-web-browser-canvas-text-focus-input-smoke.sh --report-dir \"$report_dir\"",
		"surface-headless-component-tree-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-real-window-component-tree-smoke.sh --report-dir \"$report_dir\"",
		"surface-wasm32-web-browser-canvas-component-tree-smoke.sh --report-dir \"$report_dir\"",
		"surface-headless-component-tree-api-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-real-window-component-tree-api-smoke.sh --report-dir \"$report_dir\"",
		"surface-wasm32-web-browser-canvas-component-tree-api-smoke.sh --report-dir \"$report_dir\"",
		"surface-headless-minimal-toolkit-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-real-window-minimal-toolkit-smoke.sh --report-dir \"$report_dir\"",
		"surface-wasm32-web-browser-canvas-minimal-toolkit-smoke.sh --report-dir \"$report_dir\"",
		"surface-headless-toolkit-reuse-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-real-window-toolkit-reuse-smoke.sh --report-dir \"$report_dir\"",
		"surface-wasm32-web-browser-canvas-toolkit-reuse-smoke.sh --report-dir \"$report_dir\"",
		"surface-headless-accessibility-metadata-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-real-window-accessibility-metadata-smoke.sh --report-dir \"$report_dir\"",
		"surface-wasm32-web-browser-canvas-accessibility-metadata-smoke.sh --report-dir \"$report_dir\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-headless.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-linux-x64.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-linux-x64-real-window.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-wasm32-web.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-wasm32-web-browser-canvas.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-headless-text-focus-input.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-linux-x64-real-window-text-focus-input.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-wasm32-web-browser-canvas-text-focus-input.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-headless-component-tree.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-linux-x64-real-window-component-tree.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-wasm32-web-browser-canvas-component-tree.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-headless-component-tree-api.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-linux-x64-real-window-component-tree-api.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-wasm32-web-browser-canvas-component-tree-api.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-headless-minimal-toolkit.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-linux-x64-real-window-minimal-toolkit.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-wasm32-web-browser-canvas-minimal-toolkit.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-headless-toolkit-reuse.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-linux-x64-real-window-toolkit-reuse.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-wasm32-web-browser-canvas-toolkit-reuse.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-headless-accessibility-metadata.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-linux-x64-real-window-accessibility-metadata.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-wasm32-web-browser-canvas-accessibility-metadata.json\"",
		"go run ./tools/cmd/validate-artifact-hashes --write --root \"$report_dir\" --out \"$report_dir/artifact-hashes.json\"",
		"go run ./tools/cmd/validate-artifact-hashes --manifest \"$report_dir/artifact-hashes.json\"",
		"tetra.surface.runtime.v1",
		"no legacy UI sidecar artifacts",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate script missing %q", want)
		}
	}
}

func TestReleaseSurfaceFinalReleaseGateRunsCurrentSurfaceV1Evidence(t *testing.T) {
	root := repoRoot(t)
	gatePath := filepath.Join(root, "scripts", "release", "surface", "release-gate.sh")
	raw, err := os.ReadFile(gatePath)
	if err != nil {
		t.Fatalf("read Surface final release gate script: %v", err)
	}
	gate := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/release-gate.sh [--report-dir DIR]",
		"surface-headless-release-smoke.sh --report-dir \"$report_dir\"",
		"surface-headless-release-text-input-smoke.sh --report-dir \"$report_dir\"",
		"surface-headless-release-toolkit-smoke.sh --report-dir \"$report_dir\"",
		"surface-headless-release-accessibility-smoke.sh --report-dir \"$report_dir\"",
		"surface-headless-app-model-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-release-window-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-release-app-shell-smoke.sh --report-dir \"$report_dir\"",
		"surface-dev-workflow-smoke.sh --report-dir \"$report_dir\"",
		"surface-inspector-smoke.sh --report-dir \"$report_dir\"",
		"surface-template-smoke.sh --report-dir \"$report_dir\"",
		"surface-reference-apps-smoke.sh --report-dir \"$report_dir_arg\"",
		"surface-package-smoke.sh --report-dir \"$report_dir_arg\"",
		"surface-crash-report-smoke.sh --report-dir \"$report_dir_arg\"",
		"surface-i18n-smoke.sh --report-dir \"$report_dir_arg\"",
		"surface-widget-migration-smoke.sh --report-dir \"$report_dir_arg\"",
		"surface-linux-x64-release-text-input-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-release-toolkit-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-release-accessibility-smoke.sh --report-dir \"$report_dir\"",
		"surface-wasm32-web-release-browser-smoke.sh --report-dir \"$report_dir\"",
		"surface-wasm32-web-release-text-input-smoke.sh --report-dir \"$report_dir\"",
		"surface-wasm32-web-release-toolkit-smoke.sh --report-dir \"$report_dir\"",
		"surface-wasm32-web-release-accessibility-smoke.sh --report-dir \"$report_dir\"",
		"surface-headless-release.json",
		"surface-headless-release-text-input.json",
		"surface-headless-release-toolkit.json",
		"surface-headless-release-accessibility.json",
		"surface-headless-app-model.json",
		"surface-linux-x64-release-window.json",
		"surface-linux-x64-release-app-shell.json",
		"surface-dev-workflow.json",
		"surface-inspector.json",
		"surface-template-smoke.json",
		"surface-reference-apps.json",
		"surface-package.json",
		"surface-crash-report.json",
		"surface-i18n.json",
		"surface-widget-migration.json",
		"surface-linux-x64-release-text-input.json",
		"surface-linux-x64-release-toolkit.json",
		"surface-linux-x64-release-accessibility.json",
		"surface-macos-x64-target-host-status.json",
		"surface-windows-x64-target-host-status.json",
		"surface-wasm32-web-release-browser.json",
		"surface-wasm32-web-release-text-input.json",
		"surface-wasm32-web-release-toolkit.json",
		"surface-wasm32-web-release-accessibility.json",
		"morph/surface-morph-gate-summary.json",
		"morph/headless/surface-headless-morph.json",
		"surface-release-summary.json",
		"artifact-hashes.json",
		"source \"$script_dir/report-dir-guard.sh\"",
		"surface_release_require_fresh_report_dir \"$report_dir\" \"$repo_root\" \"surface_release_gate:\"",
		"morph-gate.sh --report-dir \"$morph_report_dir\"",
		"\"app_model\": \"explicit-command-reducer-v1\"",
		"\"linux_app_shell\": \"linux-app-shell-subset-v1\"",
		"\"app_shell_features\": \"electron-feature-ledger-v1\"",
		"\"security_permissions\": \"surface-security-permission-v1\"",
		"\"performance_budget\": \"surface-performance-budget-v1\"",
		"\"developer_fast_loop\": \"surface-dev-workflow-v1\"",
		"\"inspector\": \"surface-inspector-v1\"",
		"\"project_templates\": \"surface-template-smoke-v1\"",
		"\"reference_apps\": \"surface-reference-app-suite-v1\"",
		"\"surface_package\": \"surface-package-v1\"",
		"\"crash_reporting\": \"surface-crash-report-v1\"",
		"\"i18n_localization\": \"surface-i18n-v1\"",
		"\"widget_migration\": \"surface-widget-migration-v1\"",
		"tetra.surface.target-host-status.v1",
		"macos-target-host-runtime",
		"build-only",
		"\"morph\": \"morph-capsule\"",
		"\"morph_gate\": \"tetra.surface.morph.gate.v1\"",
		"\"git_head\":",
		"\"git_dirty\":",
		"\"host_os\":",
		"\"host_arch\":",
		"\"producer\": \"scripts/release/surface/release-gate.sh\"",
		"\"generated_at_utc\":",
		"\"command_line\":",
		"\"version\":",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-release-summary.json\" --release surface-v1",
		"go run ./tools/cmd/validate-surface-security-report --report \"$report_dir/surface-linux-x64-release-app-shell.json\"",
		"go run ./tools/cmd/validate-surface-performance-budget --report \"$report_dir/surface-linux-x64-release-app-shell.json\"",
		"go run ./tools/cmd/validate-surface-dev-workflow --report \"$report_dir/surface-dev-workflow.json\"",
		"go run ./tools/cmd/validate-surface-inspector --report \"$report_dir/surface-inspector.json\"",
		"go run ./tools/cmd/validate-surface-template-smoke --report \"$report_dir/surface-template-smoke.json\"",
		"go run ./tools/cmd/validate-surface-reference-apps --report \"$report_dir/surface-reference-apps.json\"",
		"go run ./tools/cmd/validate-surface-package --report \"$report_dir/surface-package.json\"",
		"go run ./tools/cmd/validate-surface-crash-report --report \"$report_dir/surface-crash-report.json\"",
		"go run ./tools/cmd/validate-surface-i18n --report \"$report_dir/surface-i18n.json\"",
		"go run ./tools/cmd/validate-surface-widget-migration --report \"$report_dir/surface-widget-migration.json\"",
		"go run ./tools/cmd/validate-artifact-hashes --write --root \"$report_dir\" --out \"$report_dir/artifact-hashes.json\"",
		"go run ./tools/cmd/validate-artifact-hashes --manifest \"$report_dir/artifact-hashes.json\"",
		"go run ./tools/cmd/validate-surface-release-state --report-dir \"$report_dir\" --expected-status current --scope surface-v1-linux-web --manifest docs/generated/manifest.json",
		"Surface v1 release gate must fail, not skip, when Chromium-compatible browser, Linux Wayland/display, accessibility probe, or clipboard harness evidence is unavailable.",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface final release gate script missing %q", want)
		}
	}

	hashWrite := strings.Index(gate, "go run ./tools/cmd/validate-artifact-hashes --write --root \"$report_dir\" --out \"$report_dir/artifact-hashes.json\"")
	stateValidate := strings.Index(gate, "go run ./tools/cmd/validate-surface-release-state --report-dir \"$report_dir\" --expected-status current --scope surface-v1-linux-web --manifest docs/generated/manifest.json")
	if hashWrite < 0 || stateValidate < 0 {
		t.Fatalf("Surface final release gate must include artifact hash write and release-state validation")
	}
	if hashWrite > stateValidate {
		t.Fatalf("Surface final release gate must write artifact-hashes.json before validate-surface-release-state reads it")
	}
}

func TestReleaseSurfaceTemplateSmokeAndGateWiring(t *testing.T) {
	root := repoRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release", "surface", "surface-template-smoke.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Surface template smoke script: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-template-smoke.sh [--report-dir DIR]",
		"surface-template-smoke.json",
		"go run ./cli/cmd/tetra new surface-app",
		"--template command-palette",
		"--template settings",
		"--template dashboard",
		"--template editor-shell",
		"--template multi-window-notes",
		"--template web-canvas",
		"go run ./cli/cmd/tetra check",
		"go run ./cli/cmd/tetra build --target linux-x64",
		"go run ./cli/cmd/tetra run --target linux-x64",
		"go run ./tools/cmd/surface-inspector",
		"go run ./tools/cmd/surface-visual-diff",
		"tar -czf",
		"go run ./tools/cmd/validate-surface-template-smoke --report \"$report_path\"",
		"tetra.surface.template-smoke.v1",
		"surface-template-smoke-v1",
		"no React",
		"no Electron",
		"no DOM app UI tree",
		"no CSS runtime",
		"Block/Morph",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Surface template smoke script missing %q", want)
		}
	}

	gateRaw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(gateRaw)
	for _, want := range []string{
		"surface-template-smoke.sh --report-dir \"$report_dir\"",
		"surface-template-smoke.json",
		"\"project_templates\": \"surface-template-smoke-v1\"",
		"go run ./tools/cmd/validate-surface-template-smoke --report \"$report_dir/surface-template-smoke.json\"",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate missing template smoke wiring %q", want)
		}
	}
	assertOrderedFragments(t, gate,
		`surface-inspector-smoke.sh`,
		`surface-template-smoke.sh`,
		`surface-reference-apps-smoke.sh`,
		`surface-package-smoke.sh`,
		`summary_path="$report_dir/surface-release-summary.json"`,
		`validate-surface-template-smoke --report "$report_dir/surface-template-smoke.json"`,
		`validate-surface-reference-apps --report "$report_dir/surface-reference-apps.json"`,
		`validate-surface-package --report "$report_dir/surface-package.json"`,
		`validate-surface-release-state --report-dir "$report_dir"`,
	)
}

func TestReleaseSurfaceReferenceAppsSmokeAndGateWiring(t *testing.T) {
	root := repoRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release", "surface", "surface-reference-apps-smoke.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Surface reference apps smoke script: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-reference-apps-smoke.sh [--report-dir DIR]",
		"surface-reference-apps.json",
		"surface_reference_command_palette.tetra",
		"surface_reference_settings.tetra",
		"surface_reference_dashboard.tetra",
		"surface_reference_editor_shell.tetra",
		"surface_reference_file_manager.tetra",
		"surface_reference_dialog_notification.tetra",
		"surface_reference_localized_form.tetra",
		"surface_reference_accessibility_form.tetra",
		"surface_reference_multi_window_notes.tetra",
		"surface_reference_migration.tetra",
		"go run ./cli/cmd/tetra check \"$source\"",
		"go run ./cli/cmd/tetra build --target linux-x64 -o \"$build_path\" \"$source\"",
		"go run ./cli/cmd/tetra run --target linux-x64 \"$source\"",
		"go run ./tools/cmd/validate-surface-reference-apps --report \"$report_path\"",
		"go run ./tools/cmd/validate-surface-visual-report --report \"$visual_report\"",
		"tetra.surface.reference-app-suite.v1",
		"surface-reference-app-suite-v1",
		"headless",
		"linux-x64-real-window",
		"wasm32-web-browser-canvas",
		"React",
		"Electron",
		"DOM app UI",
		"CSS runtime",
		"migration",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Surface reference apps smoke script missing %q", want)
		}
	}

	gateRaw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(gateRaw)
	for _, want := range []string{
		"surface-reference-apps-smoke.sh --report-dir \"$report_dir_arg\"",
		"surface-reference-apps.json",
		"\"reference_apps\": \"surface-reference-app-suite-v1\"",
		"go run ./tools/cmd/validate-surface-reference-apps --report \"$report_dir/surface-reference-apps.json\"",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate missing reference apps wiring %q", want)
		}
	}
	assertOrderedFragments(t, gate,
		`surface-template-smoke.sh`,
		`surface-reference-apps-smoke.sh`,
		`surface-package-smoke.sh`,
		`summary_path="$report_dir/surface-release-summary.json"`,
		`validate-surface-reference-apps --report "$report_dir/surface-reference-apps.json"`,
		`validate-surface-package --report "$report_dir/surface-package.json"`,
		`validate-surface-release-state --report-dir "$report_dir"`,
	)
}

func TestReleaseSurfacePackageSmokeAndGateWiring(t *testing.T) {
	root := repoRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release", "surface", "surface-package-smoke.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Surface package smoke script: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-package-smoke.sh [--report-dir DIR]",
		"surface-package.json",
		"surface_reference_command_palette.tetra",
		"go run ./cli/cmd/tetra check \"$source_path\"",
		"go run ./cli/cmd/tetra build --target linux-x64 -o \"$linux_binary\" \"$source_path\"",
		"go run ./cli/cmd/tetra build --target wasm32-web -o \"$wasm_binary\" \"$source_path\"",
		"surface-command-palette-linux-x64.tar.gz",
		"surface-command-palette-wasm32-web.tar.gz",
		"surface-command-palette.mjs",
		"tetra.surface.package.v1",
		"surface-package-v1",
		"surface-app-package-v1",
		"tetra.surface.update-channel.v1",
		"hash-pinned-channel-manifest-v1",
		"auto_update_runtime_claim",
		"network_update_claim",
		"no_unsigned_signing_claim",
		"go run ./tools/cmd/validate-surface-package --report \"$report_path\"",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Surface package smoke script missing %q", want)
		}
	}

	gateRaw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(gateRaw)
	for _, want := range []string{
		"surface-package-smoke.sh --report-dir \"$report_dir_arg\"",
		"surface-package.json",
		"\"surface_package\": \"surface-package-v1\"",
		"go run ./tools/cmd/validate-surface-package --report \"$report_dir/surface-package.json\"",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate missing package wiring %q", want)
		}
	}
	assertOrderedFragments(t, gate,
		`surface-reference-apps-smoke.sh`,
		`surface-package-smoke.sh`,
		`summary_path="$report_dir/surface-release-summary.json"`,
		`validate-surface-package --report "$report_dir/surface-package.json"`,
		`validate-surface-release-state --report-dir "$report_dir"`,
	)
}

func TestReleaseSurfaceI18nSmokeAndGateWiring(t *testing.T) {
	root := repoRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release", "surface", "surface-i18n-smoke.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Surface i18n smoke script: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-i18n-smoke.sh [--report-dir DIR]",
		"surface-i18n.json",
		"surface_reference_localized_form.tetra",
		"go run ./cli/cmd/tetra check \"$source_path\"",
		"go run ./cli/cmd/tetra build --target linux-x64 -o \"$linux_binary\" \"$source_path\"",
		"tetra.surface.i18n.v1",
		"surface-i18n-v1",
		"fallback_locale",
		"missing_key_diagnostic",
		"format_hooks",
		"rtl-placeholder-without-full-bidi-shaping-v1",
		"no_full_bidi_claim",
		"go run ./tools/cmd/validate-surface-i18n --report \"$report_path\"",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Surface i18n smoke script missing %q", want)
		}
	}

	gateRaw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(gateRaw)
	for _, want := range []string{
		"surface-i18n-smoke.sh --report-dir \"$report_dir_arg\"",
		"surface-i18n.json",
		"\"i18n_localization\": \"surface-i18n-v1\"",
		"go run ./tools/cmd/validate-surface-i18n --report \"$report_dir/surface-i18n.json\"",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate missing i18n wiring %q", want)
		}
	}
	assertOrderedFragments(t, gate,
		`surface-crash-report-smoke.sh`,
		`surface-i18n-smoke.sh`,
		`surface-widget-migration-smoke.sh`,
		`summary_path="$report_dir/surface-release-summary.json"`,
		`validate-surface-crash-report --report "$report_dir/surface-crash-report.json"`,
		`validate-surface-i18n --report "$report_dir/surface-i18n.json"`,
		`validate-surface-widget-migration --report "$report_dir/surface-widget-migration.json"`,
		`validate-surface-release-state --report-dir "$report_dir"`,
	)
}

func TestReleaseSurfaceWidgetMigrationSmokeAndGateWiring(t *testing.T) {
	root := repoRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release", "surface", "surface-widget-migration-smoke.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Surface widget migration smoke script: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-widget-migration-smoke.sh [--report-dir DIR]",
		"surface-widget-migration.json",
		"surface_reference_migration.tetra",
		"core_widgets_smoke.tetra",
		"go run ./cli/cmd/tetra check \"$source_path\"",
		"go run ./cli/cmd/tetra build --target linux-x64 -o \"$linux_binary\" \"$source_path\"",
		"tetra.surface.widget-migration.v1",
		"surface-widget-migration-v1",
		"lib.core.widgets",
		"Panel",
		"Button",
		"TextBox",
		"recipe_region_panel",
		"recipe_control_action",
		"recipe_field_text",
		"block_only_core_primitive",
		"no_future_core_primitive_promotion",
		"go run ./tools/cmd/validate-surface-widget-migration --report \"$report_path\"",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Surface widget migration smoke script missing %q", want)
		}
	}

	gateRaw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(gateRaw)
	for _, want := range []string{
		"surface-widget-migration-smoke.sh --report-dir \"$report_dir_arg\"",
		"surface-widget-migration.json",
		"\"widget_migration\": \"surface-widget-migration-v1\"",
		"go run ./tools/cmd/validate-surface-widget-migration --report \"$report_dir/surface-widget-migration.json\"",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate missing widget migration wiring %q", want)
		}
	}
	assertOrderedFragments(t, gate,
		`surface-i18n-smoke.sh`,
		`surface-widget-migration-smoke.sh`,
		`summary_path="$report_dir/surface-release-summary.json"`,
		`validate-surface-i18n --report "$report_dir/surface-i18n.json"`,
		`validate-surface-widget-migration --report "$report_dir/surface-widget-migration.json"`,
		`validate-surface-release-state --report-dir "$report_dir"`,
	)
}

func TestReleaseSurfaceCrashReportSmokeAndGateWiring(t *testing.T) {
	root := repoRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release", "surface", "surface-crash-report-smoke.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Surface crash report smoke script: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-crash-report-smoke.sh [--report-dir DIR]",
		"surface-crash-report.json",
		"surface_reference_command_palette.tetra",
		"go run ./cli/cmd/tetra build --target linux-x64 -o \"$linux_binary\" \"$source_path\"",
		"command_failure",
		"host_crash",
		"restart_recovery",
		"tetra.surface.diagnostic.v1",
		"surface-non-user-data-diagnostics-v1",
		"surface-diagnostic-redaction-v1",
		"scoped-linux-x64-process-restart-v1",
		"no_restart_claim_without_evidence",
		"no_electron_crash_reporter_dependency",
		"go run ./tools/cmd/validate-surface-crash-report --report \"$report_path\"",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Surface crash report smoke script missing %q", want)
		}
	}

	gateRaw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(gateRaw)
	for _, want := range []string{
		"surface-crash-report-smoke.sh --report-dir \"$report_dir_arg\"",
		"surface-crash-report.json",
		"\"crash_reporting\": \"surface-crash-report-v1\"",
		"go run ./tools/cmd/validate-surface-crash-report --report \"$report_dir/surface-crash-report.json\"",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate missing crash report wiring %q", want)
		}
	}
	assertOrderedFragments(t, gate,
		`surface-package-smoke.sh`,
		`surface-crash-report-smoke.sh`,
		`summary_path="$report_dir/surface-release-summary.json"`,
		`validate-surface-package --report "$report_dir/surface-package.json"`,
		`validate-surface-crash-report --report "$report_dir/surface-crash-report.json"`,
		`validate-surface-release-state --report-dir "$report_dir"`,
	)
}

func TestReleaseSurfaceDevWorkflowSmokeAndGateWiring(t *testing.T) {
	root := repoRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release", "surface", "surface-dev-workflow-smoke.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Surface dev workflow smoke script: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-dev-workflow-smoke.sh [--report-dir DIR]",
		"surface-dev-workflow.json",
		"go run ./cli/cmd/tetra surface dev",
		"--change-file \"token:$tokens_path\"",
		"--change-file \"recipe:$recipes_path\"",
		"--change-file \"source:$source_path\"",
		"go run ./tools/cmd/validate-surface-dev-workflow --report \"$report_path\"",
		"tetra.surface.dev-workflow.v1",
		"surface-dev-workflow-v1",
		"fast rebuild",
		"token/recipe/source",
		"no hot reload claim",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Surface dev workflow smoke script missing %q", want)
		}
	}

	gateRaw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(gateRaw)
	for _, want := range []string{
		"surface-dev-workflow-smoke.sh --report-dir \"$report_dir\"",
		"surface-dev-workflow.json",
		"\"developer_fast_loop\": \"surface-dev-workflow-v1\"",
		"go run ./tools/cmd/validate-surface-dev-workflow --report \"$report_dir/surface-dev-workflow.json\"",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate missing dev workflow wiring %q", want)
		}
	}
	assertOrderedFragments(t, gate,
		`surface-linux-x64-release-app-shell-smoke.sh`,
		`surface-dev-workflow-smoke.sh`,
		`summary_path="$report_dir/surface-release-summary.json"`,
		`validate-surface-dev-workflow --report "$report_dir/surface-dev-workflow.json"`,
		`validate-surface-release-state --report-dir "$report_dir"`,
	)
}

func TestReleaseSurfaceInspectorSmokeAndGateWiring(t *testing.T) {
	root := repoRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release", "surface", "surface-inspector-smoke.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Surface inspector smoke script: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-inspector-smoke.sh [--report-dir DIR]",
		"surface-inspector.json",
		"surface-inspector.html",
		"go run ./tools/cmd/surface-runtime-smoke --mode headless-block-system",
		"go run ./tools/cmd/surface-runtime-smoke --mode headless-morph",
		"go run ./tools/cmd/surface-runtime-smoke --mode headless-app-model",
		"go run ./tools/cmd/surface-runtime-smoke --mode headless-release-accessibility",
		"go run ./tools/cmd/surface-runtime-smoke --mode headless-block-events",
		"go run ./tools/cmd/surface-inspector",
		"go run ./tools/cmd/validate-surface-inspector --report \"$report_path\"",
		"tetra.surface.inspector.v1",
		"surface-inspector-v1",
		"static tool report",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Surface inspector smoke script missing %q", want)
		}
	}

	gateRaw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(gateRaw)
	for _, want := range []string{
		"surface-inspector-smoke.sh --report-dir \"$report_dir\"",
		"surface-inspector.json",
		"\"inspector\": \"surface-inspector-v1\"",
		"go run ./tools/cmd/validate-surface-inspector --report \"$report_dir/surface-inspector.json\"",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate missing inspector wiring %q", want)
		}
	}
	assertOrderedFragments(t, gate,
		`surface-dev-workflow-smoke.sh`,
		`surface-inspector-smoke.sh`,
		`summary_path="$report_dir/surface-release-summary.json"`,
		`validate-surface-inspector --report "$report_dir/surface-inspector.json"`,
		`validate-surface-release-state --report-dir "$report_dir"`,
	)
}

func TestReleaseSurfaceAppModelSmokeAndGateWiring(t *testing.T) {
	root := repoRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release", "surface", "surface-headless-app-model-smoke.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Surface app-model smoke script: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-headless-app-model-smoke.sh [--report-dir DIR]",
		"surface-headless-app-model.json",
		"go run ./tools/cmd/surface-runtime-smoke --mode headless-app-model",
		"--source examples/surface_app_model.tetra",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_path\" --release app-model",
		"tetra.surface.app-model.v1",
		"explicit command/reducer",
		"React hooks",
		"DOM event model",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Surface app-model smoke script missing %q", want)
		}
	}

	gateRaw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(gateRaw)
	for _, want := range []string{
		"surface-headless-app-model-smoke.sh --report-dir \"$report_dir\"",
		"surface-headless-app-model.json",
		"\"app_model\": \"explicit-command-reducer-v1\"",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate missing app-model wiring %q", want)
		}
	}
	assertOrderedFragments(t, gate,
		`surface-headless-release-accessibility-smoke.sh`,
		`surface-headless-app-model-smoke.sh`,
		`summary_path="$report_dir/surface-release-summary.json"`,
		`validate-surface-runtime --report "$report_dir/surface-release-summary.json" --release surface-v1`,
	)
}

func TestReleaseSurfaceLinuxAppShellSmokeAndGateWiring(t *testing.T) {
	root := repoRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release", "surface", "surface-linux-x64-release-app-shell-smoke.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Surface linux app-shell smoke script: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-linux-x64-release-app-shell-smoke.sh [--report-dir DIR]",
		"surface-linux-x64-release-app-shell.json",
		"go run ./tools/cmd/surface-runtime-smoke --mode linux-x64-release-app-shell",
		"--source examples/surface_linux_app_shell_notes.tetra",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_path\" --release linux-app-shell",
		"go run ./tools/cmd/validate-surface-security-report --report \"$report_path\"",
		"go run ./tools/cmd/validate-surface-performance-budget --report \"$report_path\"",
		"tetra.surface.linux-app-shell.v1",
		"linux-app-shell-subset-v1",
		"electron feature ledger",
		"surface-security-permission-v1",
		"surface-performance-budget-v1",
		"startup/frame/memory/cache/framebuffer",
		"no faster-than-Electron claim",
		"capability-checked IPC/process boundaries",
		"local hashed asset/font/image",
		"multi-window notes",
		"lifecycle open/close/reopen",
		"resize/DPI/cursors",
		"file dialog",
		"file picker",
		"notification",
		"tray",
		"crash/error",
		"blocked-pass",
		"GTK/Qt/native widget UI",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Surface linux app-shell smoke script missing %q", want)
		}
	}

	gateRaw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(gateRaw)
	for _, want := range []string{
		"surface-linux-x64-release-app-shell-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-release-app-shell.json",
		"\"linux_app_shell\": \"linux-app-shell-subset-v1\"",
		"\"app_shell_features\": \"electron-feature-ledger-v1\"",
		"\"security_permissions\": \"surface-security-permission-v1\"",
		"\"performance_budget\": \"surface-performance-budget-v1\"",
		"go run ./tools/cmd/validate-surface-security-report --report \"$report_dir/surface-linux-x64-release-app-shell.json\"",
		"go run ./tools/cmd/validate-surface-performance-budget --report \"$report_dir/surface-linux-x64-release-app-shell.json\"",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate missing linux app-shell wiring %q", want)
		}
	}
	assertOrderedFragments(t, gate,
		`surface-linux-x64-release-window-smoke.sh`,
		`surface-linux-x64-release-app-shell-smoke.sh`,
		`summary_path="$report_dir/surface-release-summary.json"`,
		`validate-surface-runtime --report "$report_dir/surface-release-summary.json" --release surface-v1`,
	)
}

func TestSurfaceTreeAppUsesHardenedComponentTreeAPI(t *testing.T) {
	root := repoRoot(t)
	componentRaw, err := os.ReadFile(filepath.Join(root, "lib", "core", "component.tetra"))
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
			t.Fatalf("lib/core/component.tetra must expose hardened helper %q", want)
		}
	}

	appRaw, err := os.ReadFile(filepath.Join(root, "examples", "surface_tree_app.tetra"))
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
			t.Fatalf("surface_tree_app.tetra still has hardcoded pointer hit-test branch %q", forbidden)
		}
	}
}

func TestSurfaceToolkitFormUsesReusableWidgetModule(t *testing.T) {
	root := repoRoot(t)
	widgetsRaw, err := os.ReadFile(filepath.Join(root, "lib", "core", "widgets.tetra"))
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
			t.Fatalf("lib/core/widgets.tetra must expose minimal toolkit API %q", want)
		}
	}

	appRaw, err := os.ReadFile(filepath.Join(root, "examples", "surface_toolkit_form.tetra"))
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
			t.Fatalf("surface_toolkit_form.tetra must not contain demo/fake toolkit marker %q", forbidden)
		}
	}
}

func TestSurfaceToolkitSettingsUsesReusableWidgetModule(t *testing.T) {
	root := repoRoot(t)
	appRaw, err := os.ReadFile(filepath.Join(root, "examples", "surface_toolkit_settings.tetra"))
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
			t.Fatalf("surface_toolkit_settings.tetra must not contain demo/fake toolkit marker %q", forbidden)
		}
	}
	if strings.Count(app, "widgets.add_textbox(") < 2 {
		t.Fatalf("surface_toolkit_settings.tetra must construct at least two TextBoxes through widgets.add_textbox")
	}
}

func TestSurfaceAccessibilitySettingsUsesMetadataTreeHelpers(t *testing.T) {
	root := repoRoot(t)
	accessibilityRaw, err := os.ReadFile(filepath.Join(root, "lib", "core", "accessibility.tetra"))
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
			t.Fatalf("lib/core/accessibility.tetra must expose metadata helper %q", want)
		}
	}

	widgetsRaw, err := os.ReadFile(filepath.Join(root, "lib", "core", "widgets.tetra"))
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
			t.Fatalf("lib/core/widgets.tetra must expose accessibility helper %q", want)
		}
	}

	appRaw, err := os.ReadFile(filepath.Join(root, "examples", "surface_accessibility_settings.tetra"))
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
			t.Fatalf("surface_accessibility_settings.tetra must use accessibility metadata helper %q", want)
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
			t.Fatalf("surface_accessibility_settings.tetra must not contain production accessibility/legacy marker %q", forbidden)
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
		filepath.Join("scripts", "release", "surface", "surface-linux-x64-real-window-text-focus-input-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-wasm32-web-browser-canvas-text-focus-input-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-headless-component-tree-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-linux-x64-real-window-component-tree-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-wasm32-web-browser-canvas-component-tree-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-headless-component-tree-api-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-linux-x64-real-window-component-tree-api-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-wasm32-web-browser-canvas-component-tree-api-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-headless-block-system-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-linux-x64-real-window-block-system-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-wasm32-web-browser-canvas-block-system-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-headless-minimal-toolkit-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-linux-x64-real-window-minimal-toolkit-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-wasm32-web-browser-canvas-minimal-toolkit-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-headless-toolkit-reuse-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-linux-x64-real-window-toolkit-reuse-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-wasm32-web-browser-canvas-toolkit-reuse-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-headless-accessibility-metadata-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-linux-x64-real-window-accessibility-metadata-smoke.sh"),
		filepath.Join("scripts", "release", "surface", "surface-wasm32-web-browser-canvas-accessibility-metadata-smoke.sh"),
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
