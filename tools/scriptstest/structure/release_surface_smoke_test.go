package structure

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestReleaseSurfaceSmokeScriptsUseStrictReleaseValidation(t *testing.T) {
	root := repoRoot(t)
	scripts, err := filepath.Glob(
		filepath.Join(root, "scripts", "release", "surface", "*release*.sh"),
	)
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
		if strings.Contains(text, "validate-surface-runtime") &&
			!strings.Contains(text, "--release ") {
			t.Fatalf(
				"Surface release script %s validates runtime reports without a strict --release selector",
				script,
			)
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
		"stable_module_pattern='^module lib\\.core\\.'",
		"stable_module_pattern+='(surface|draw|component|widgets|accessibility|text|style)$'",
		`rg -n "$stable_module_pattern" lib/core`,
		"rg -n '^module lib\\.core\\..*(experimental|v[0-9]+|_[vV][0-9]+)' lib/core",
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
			path: "examples/surface/release/surface_release_counter.tetra",
			want: []string{
				"import lib.core.widgets as widgets",
				"import lib.core.style as style",
				"import lib.core.accessibility as accessibility",
				"examples.surface.release.surface_release_counter",
				"surface.open",
				"surface.present",
				"event_mouse_up",
				"event_key_down",
				"event_resize",
				"accessibility.value_name",
			},
		},
		{
			path: "examples/surface/release/surface_release_form.tetra",
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
			path: "examples/surface/release/surface_release_text_input.tetra",
			want: []string{
				"import lib.core.text as text",
				"clipboard_write_text",
				"clipboard_read_text_into",
				"poll_composition",
				"composition_clear",
			},
		},
		{
			path: "examples/surface/release/surface_release_accessibility.tetra",
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
	examples, err := filepath.Glob(
		filepath.Join(root, "examples", "surface", "release", "surface_release_*.tetra"),
	)
	if err != nil {
		t.Fatalf("glob Surface release examples: %v", err)
	}
	if len(examples) == 0 {
		t.Fatalf("no Surface release examples found")
	}

	localDemoWidgetStruct := regexp.MustCompile(
		`(?m)^struct\s+\w*(Button|TextBox|Row|Column|Panel|Scroll|Checkbox)\w*\s*:`,
	)
	manualTreeWrite := regexp.MustCompile(
		`(?m)\.(id|parent_id|rect|first_child|child_count|flags)\s*=`,
	)
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
		if strings.Contains(rel, "text_input") &&
			!strings.Contains(text, "import lib.core.text as text") {
			t.Fatalf("Surface text/input release example %s must import lib.core.text", rel)
		}
		if strings.Contains(text, "accessibility.") &&
			!strings.Contains(text, "import lib.core.accessibility as accessibility") {
			t.Fatalf(
				"Surface accessibility release example %s must import lib.core.accessibility",
				rel,
			)
		}
		if match := localDemoWidgetStruct.FindString(text); match != "" {
			t.Fatalf(
				"Surface release example %s defines local demo widget struct %q",
				rel,
				strings.TrimSpace(match),
			)
		}
		if strings.Contains(text, "component.TreeNode(") {
			t.Fatalf(
				"Surface release example %s manually constructs component.TreeNode structural evidence",
				rel,
			)
		}
		if match := manualTreeWrite.FindString(text); match != "" {
			t.Fatalf(
				"Surface release example %s writes TreeNode structural field %q manually",
				rel,
				match,
			)
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
				t.Fatalf(
					"Surface release example %s contains forbidden fake-promotion marker %q",
					rel,
					forbidden,
				)
			}
		}
	}
}

func TestReleaseSurfaceSmokeScriptsDocumentHeadlessAndPendingLinuxX64Gates(t *testing.T) {
	root := repoRoot(t)
	headlessPath := filepath.Join(
		root,
		"scripts",
		"release",
		"surface",
		"surface-headless-smoke.sh",
	)
	headlessRaw, err := os.ReadFile(headlessPath)
	if err != nil {
		t.Fatalf("read Surface headless smoke script: %v", err)
	}
	headless := string(headlessRaw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-headless-smoke.sh [--report-dir DIR]",
		"surface-headless.json",
		"go run ./tools/cmd/surface-runtime-smoke",
		"--mode headless",
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
		"go run ./tools/cmd/surface-runtime-smoke",
		"--mode linux-x64",
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
		"go run ./tools/cmd/surface-runtime-smoke",
		"--mode wasm32-web",
		"go run ./tools/cmd/validate-surface-runtime",
		"go run ./tools/cmd/validate-wasm-imports",
		"compiler-owned wasm Surface loader",
		"legacy UI sidecars are not accepted",
	} {
		if !strings.Contains(wasm, want) {
			t.Fatalf("Surface wasm32-web smoke script missing %q", want)
		}
	}

	realWindowPath := filepath.Join(
		root,
		"scripts",
		"release",
		"surface",
		"surface-linux-x64-real-window-smoke.sh",
	)
	realWindowRaw, err := os.ReadFile(realWindowPath)
	if err != nil {
		t.Fatalf("read Surface linux-x64 real-window smoke script: %v", err)
	}
	realWindow := string(realWindowRaw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-linux-x64-real-window-smoke.sh [--report-dir DIR]",
		"surface-linux-x64-real-window.json",
		"go run ./tools/cmd/surface-runtime-smoke",
		"--mode linux-x64-real-window",
		"--source examples/surface/release/surface_release_counter.tetra",
		"go run ./tools/cmd/validate-surface-runtime",
		"linux-x64-real-window",
		"real Linux window",
	} {
		if !strings.Contains(realWindow, want) {
			t.Fatalf("Surface linux-x64 real-window smoke script missing %q", want)
		}
	}

	browserCanvasPath := filepath.Join(
		root,
		"scripts",
		"release",
		"surface",
		"surface-wasm32-web-browser-canvas-smoke.sh",
	)
	browserCanvasRaw, err := os.ReadFile(browserCanvasPath)
	if err != nil {
		t.Fatalf("read Surface wasm32-web browser-canvas smoke script: %v", err)
	}
	browserCanvas := string(browserCanvasRaw)
	for _, want := range []string{
		("Usage: bash scripts/release/surface/surface-wasm32-web-browser-" +
			"canvas-smoke.sh [--report-dir DIR]"),
		"surface-wasm32-web-browser-canvas.json",
		"go run ./tools/cmd/surface-runtime-smoke",
		"--mode wasm32-web-browser-canvas",
		"--source examples/surface/release/surface_release_counter.tetra",
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
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-headless-text-focus-input-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-headless-text-focus-" +
					"input-smoke.sh [--report-dir DIR]"),
				"surface-headless-text-focus-input.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode headless-text-focus-input",
				"--source examples/surface/runtime/surface_textbox_app.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"click focus",
				"backspace/delete",
			},
		},
		{
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-linux-x64-real-window-text-focus-input-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-linux-x64-real-" +
					"window-text-focus-input-smoke.sh [--report-dir DIR]"),
				"surface-linux-x64-real-window-text-focus-input.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode linux-x64-real-window-text-focus-input",
				"--source examples/surface/runtime/surface_textbox_app.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"real-window",
				"native-input",
			},
		},
		{
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-wasm32-web-browser-canvas-text-focus-input-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-wasm32-web-browser-" +
					"canvas-text-focus-input-smoke.sh [--report-dir DIR]"),
				"surface-wasm32-web-browser-canvas-text-focus-input.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode wasm32-web-browser-canvas-text-focus-input",
				"--source examples/surface/runtime/surface_textbox_app.tetra",
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
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-headless-release-text-input-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-headless-release-" +
					"text-input-smoke.sh [--report-dir DIR]"),
				"surface-headless-release-text-input.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode headless-release-text-input",
				"--source examples/surface/release/surface_release_text_input.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"tetra.surface.text-input.v1",
				"owned UTF-8 buffer",
			},
		},
		{
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-linux-x64-release-text-input-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-linux-x64-release-" +
					"text-input-smoke.sh [--report-dir DIR]"),
				"surface-linux-x64-release-text-input.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode linux-x64-release-text-input",
				"--source examples/surface/release/surface_release_text_input.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"real-window",
				"platform clipboard or platform IME completion",
			},
		},
		{
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-wasm32-web-release-text-input-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-wasm32-web-release-" +
					"text-input-smoke.sh [--report-dir DIR]"),
				"surface-wasm32-web-release-text-input.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode wasm32-web-release-text-input",
				"--source examples/surface/release/surface_release_text_input.tetra",
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
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-headless-component-tree-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-headless-component-" +
					"tree-smoke.sh [--report-dir DIR]"),
				"surface-headless-component-tree.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode headless-component-tree",
				"--source examples/surface/toolkit/surface_tree_app.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"root-to-leaf dispatch paths",
				"resize relayout",
			},
		},
		{
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-linux-x64-real-window-component-tree-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-linux-x64-real-" +
					"window-component-tree-smoke.sh [--report-dir DIR]"),
				"surface-linux-x64-real-window-component-tree.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode linux-x64-real-window-component-tree",
				"--source examples/surface/toolkit/surface_tree_app.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"real Linux window",
				"native-input",
			},
		},
		{
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-wasm32-web-browser-canvas-component-tree-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-wasm32-web-browser-" +
					"canvas-component-tree-smoke.sh [--report-dir DIR]"),
				"surface-wasm32-web-browser-canvas-component-tree.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode wasm32-web-browser-canvas-component-tree",
				"--source examples/surface/toolkit/surface_tree_app.tetra",
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
				path: filepath.Join(
					root,
					"scripts",
					"release",
					"surface",
					"surface-headless-component-tree-api-smoke.sh",
				),
				want: []string{
					("Usage: bash scripts/release/surface/surface-headless-component-" +
						"tree-api-smoke.sh [--report-dir DIR]"),
					"surface-headless-component-tree-api.json",
					"go run ./tools/cmd/surface-runtime-smoke",
					"--mode headless-component-tree-api",
					"--source examples/surface/toolkit/surface_tree_app.tetra",
					"go run ./tools/cmd/validate-surface-runtime",
					"component_tree_api",
					"manual_bookkeeping=false",
				},
			},
			{
				path: filepath.Join(
					root,
					"scripts",
					"release",
					"surface",
					"surface-linux-x64-real-window-component-tree-api-smoke.sh",
				),
				want: []string{
					("Usage: bash scripts/release/surface/surface-linux-x64-real-" +
						"window-component-tree-api-smoke.sh [--report-dir DIR]"),
					"surface-linux-x64-real-window-component-tree-api.json",
					"go run ./tools/cmd/surface-runtime-smoke",
					"--mode linux-x64-real-window-component-tree-api",
					"--source examples/surface/toolkit/surface_tree_app.tetra",
					"go run ./tools/cmd/validate-surface-runtime",
					"real Linux window",
					"component_tree_api",
				},
			},
			{
				path: filepath.Join(
					root,
					"scripts",
					"release",
					"surface",
					"surface-wasm32-web-browser-canvas-component-tree-api-smoke.sh",
				),
				want: []string{
					("Usage: bash scripts/release/surface/surface-wasm32-web-browser-" +
						"canvas-component-tree-api-smoke.sh [--report-dir DIR]"),
					"surface-wasm32-web-browser-canvas-component-tree-api.json",
					"go run ./tools/cmd/surface-runtime-smoke",
					"--mode wasm32-web-browser-canvas-component-tree-api",
					"--source examples/surface/toolkit/surface_tree_app.tetra",
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
					t.Fatalf(
						"Surface component-tree API smoke script %s missing %q",
						script.path,
						want,
					)
				}
			}
		}
	}

	blockSystemScripts := []struct {
		path string
		want []string
	}{
		{
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-headless-block-system-smoke.sh",
			),
			want: []string{
				"Usage: bash scripts/release/surface/surface-headless-block-system-smoke.sh [--report-dir DIR]",
				"surface-headless-block-system.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode headless-block-system",
				"--source examples/surface/block_core/surface_block_system.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"go run ./tools/cmd/validate-surface-block-examples",
				"surface-block-examples.json",
				"examples/surface/block_apps/surface_block_command_palette.tetra",
				"examples/surface/block_apps/surface_block_project_dashboard.tetra",
				"examples/surface/block_apps/surface_block_settings.tetra",
				"examples/surface/block_apps/surface_block_editor_shell.tetra",
				"examples/surface/block_apps/surface_block_glass_panel.tetra",
				"tetra.surface.block-system.v1",
				"golden/checksum",
			},
		},
		{
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-linux-x64-real-window-block-system-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-linux-x64-real-" +
					"window-block-system-smoke.sh [--report-dir DIR]"),
				"surface-block-system-linux-x64.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode linux-x64-real-window-block-system",
				"--source examples/surface/block_core/surface_block_system.tetra",
				"go run ./tools/cmd/validate-surface-block-report",
				"real Linux window",
				"tetra.surface.block-system.v1",
				"WAYLAND_DISPLAY",
				"blocked",
			},
		},
		{
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-wasm32-web-browser-canvas-block-system-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-wasm32-web-browser-" +
					"canvas-block-system-smoke.sh [--report-dir DIR]"),
				"surface-block-system-wasm32-web.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode wasm32-web-browser-canvas-block-system",
				"--source examples/surface/block_core/surface_block_system.tetra",
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
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-headless-minimal-toolkit-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-headless-minimal-" +
					"toolkit-smoke.sh [--report-dir DIR]"),
				"surface-headless-minimal-toolkit.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode headless-minimal-toolkit",
				"--source examples/surface/toolkit/surface_toolkit_form.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"tetra.surface.toolkit.v1",
				"minimal-widgets-v1",
			},
		},
		{
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-linux-x64-real-window-minimal-toolkit-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-linux-x64-real-" +
					"window-minimal-toolkit-smoke.sh [--report-dir DIR]"),
				"surface-linux-x64-real-window-minimal-toolkit.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode linux-x64-real-window-minimal-toolkit",
				"--source examples/surface/toolkit/surface_toolkit_form.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"real Linux window",
				"tetra.surface.toolkit.v1",
			},
		},
		{
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-wasm32-web-browser-canvas-minimal-toolkit-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-wasm32-web-browser-" +
					"canvas-minimal-toolkit-smoke.sh [--report-dir DIR]"),
				"surface-wasm32-web-browser-canvas-minimal-toolkit.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode wasm32-web-browser-canvas-minimal-toolkit",
				"--source examples/surface/toolkit/surface_toolkit_form.tetra",
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
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-headless-toolkit-reuse-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-headless-toolkit-" +
					"reuse-smoke.sh [--report-dir DIR]"),
				"surface-headless-toolkit-reuse.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode headless-toolkit-reuse",
				"--source examples/surface/toolkit/surface_toolkit_settings.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"tetra.surface.toolkit.v1",
				"toolkit-reuse-v1",
			},
		},
		{
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-linux-x64-real-window-toolkit-reuse-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-linux-x64-real-" +
					"window-toolkit-reuse-smoke.sh [--report-dir DIR]"),
				"surface-linux-x64-real-window-toolkit-reuse.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode linux-x64-real-window-toolkit-reuse",
				"--source examples/surface/toolkit/surface_toolkit_settings.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"real Linux window",
				"tetra.surface.toolkit.v1",
			},
		},
		{
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-wasm32-web-browser-canvas-toolkit-reuse-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-wasm32-web-browser-" +
					"canvas-toolkit-reuse-smoke.sh [--report-dir DIR]"),
				"surface-wasm32-web-browser-canvas-toolkit-reuse.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode wasm32-web-browser-canvas-toolkit-reuse",
				"--source examples/surface/toolkit/surface_toolkit_settings.tetra",
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
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-headless-release-toolkit-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-headless-release-" +
					"toolkit-smoke.sh [--report-dir DIR]"),
				"surface-headless-release-toolkit.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode headless-release-toolkit",
				"--source examples/surface/release/surface_release_form.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"tetra.surface.toolkit.v1",
				"production-widgets-v1",
			},
		},
		{
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-linux-x64-release-toolkit-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-linux-x64-release-" +
					"toolkit-smoke.sh [--report-dir DIR]"),
				"surface-linux-x64-release-toolkit.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode linux-x64-release-toolkit",
				"--source examples/surface/release/surface_release_form.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"real Linux window",
				"tetra.surface.toolkit.v1",
				"production-widgets-v1",
			},
		},
		{
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-wasm32-web-release-toolkit-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-wasm32-web-release-" +
					"toolkit-smoke.sh [--report-dir DIR]"),
				"surface-wasm32-web-release-toolkit.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode wasm32-web-release-toolkit",
				"--source examples/surface/release/surface_release_form.tetra",
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
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-wasm32-web-release-browser-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-wasm32-web-release-" +
					"browser-smoke.sh [--report-dir DIR]"),
				"surface-wasm32-web-release-browser.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode wasm32-web-release-browser",
				"--source examples/surface/release/surface_release_form.tetra",
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
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-linux-x64-release-window-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-linux-x64-release-" +
					"window-smoke.sh [--report-dir DIR]"),
				"surface-linux-x64-release-window.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode linux-x64-release-window",
				"--source examples/surface/release/surface_release_form.tetra",
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
				t.Fatalf(
					"Surface linux release window smoke script %s missing %q",
					script.path,
					want,
				)
			}
		}
	}

	releaseAccessibilityScripts := []struct {
		path string
		want []string
	}{
		{
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-headless-release-accessibility-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-headless-release-" +
					"accessibility-smoke.sh [--report-dir DIR]"),
				"surface-headless-release-accessibility.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode headless-release-accessibility",
				"--source examples/surface/release/surface_release_accessibility.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"tetra.surface.accessibility-tree.v1",
				"platform-bridge-v1",
			},
		},
		{
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-linux-x64-release-accessibility-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-linux-x64-release-" +
					"accessibility-smoke.sh [--report-dir DIR]"),
				"surface-linux-x64-release-accessibility.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode linux-x64-release-accessibility",
				"--source examples/surface/release/surface_release_accessibility.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"linux_accessibility_host_bridge_v1",
				"linux accessibility platform probe",
			},
		},
		{
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-wasm32-web-release-accessibility-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-wasm32-web-release-" +
					"accessibility-smoke.sh [--report-dir DIR]"),
				"surface-wasm32-web-release-accessibility.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode wasm32-web-release-accessibility",
				"--source examples/surface/release/surface_release_accessibility.tetra",
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
				t.Fatalf(
					"Surface release accessibility smoke script %s missing %q",
					script.path,
					want,
				)
			}
		}
	}

	accessibilityMetadataScripts := []struct {
		path string
		want []string
	}{
		{
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-headless-accessibility-metadata-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-headless-" +
					"accessibility-metadata-smoke.sh [--report-dir DIR]"),
				"surface-headless-accessibility-metadata.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode headless-accessibility-metadata",
				"--source examples/surface/toolkit/surface_accessibility_settings.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"tetra.surface.accessibility-tree.v1",
				"platform accessibility",
			},
		},
		{
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-linux-x64-real-window-accessibility-metadata-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-linux-x64-real-" +
					"window-accessibility-metadata-smoke.sh [--report-dir DIR]"),
				"surface-linux-x64-real-window-accessibility-metadata.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode linux-x64-real-window-accessibility-metadata",
				"--source examples/surface/toolkit/surface_accessibility_settings.tetra",
				"go run ./tools/cmd/validate-surface-runtime",
				"native window/input",
				"tetra.surface.accessibility-tree.v1",
			},
		},
		{
			path: filepath.Join(
				root,
				"scripts",
				"release",
				"surface",
				"surface-wasm32-web-browser-canvas-accessibility-metadata-smoke.sh",
			),
			want: []string{
				("Usage: bash scripts/release/surface/surface-wasm32-web-browser-" +
					"canvas-accessibility-metadata-smoke.sh [--report-dir DIR]"),
				"surface-wasm32-web-browser-canvas-accessibility-metadata.json",
				"go run ./tools/cmd/surface-runtime-smoke",
				"--mode wasm32-web-browser-canvas-accessibility-metadata",
				"--source examples/surface/toolkit/surface_accessibility_settings.tetra",
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
				t.Fatalf(
					"Surface accessibility metadata smoke script %s missing %q",
					script.path,
					want,
				)
			}
		}
	}
}
