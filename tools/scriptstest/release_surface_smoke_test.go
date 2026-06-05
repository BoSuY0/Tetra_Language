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
		if strings.Contains(text, "validate-surface-runtime") && !strings.Contains(text, "--release surface-v1") {
			t.Fatalf("Surface release script %s validates runtime reports without --release surface-v1", script)
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
		"surface-linux-x64-release-window-smoke.sh --report-dir \"$report_dir\"",
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
		"surface-linux-x64-release-window.json",
		"surface-linux-x64-release-text-input.json",
		"surface-linux-x64-release-toolkit.json",
		"surface-linux-x64-release-accessibility.json",
		"surface-wasm32-web-release-browser.json",
		"surface-wasm32-web-release-text-input.json",
		"surface-wasm32-web-release-toolkit.json",
		"surface-wasm32-web-release-accessibility.json",
		"surface-release-summary.json",
		"artifact-hashes.json",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-release-summary.json\" --release surface-v1",
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
