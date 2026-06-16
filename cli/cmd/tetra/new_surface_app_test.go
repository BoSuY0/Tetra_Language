package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewSurfaceAppScaffoldCreatesRunnableBlockMorphProject(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	appDir := filepath.Join(dir, "PaletteDesk")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"new", "surface-app", "--template", "command-palette", appDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("new surface-app exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	for _, rel := range []string{"Capsule.t4", "src/main.tetra", "surface-template.json", "README.md"} {
		if _, err := os.Stat(filepath.Join(appDir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected scaffold file %s: %v", rel, err)
		}
	}
	capsuleRaw, err := os.ReadFile(filepath.Join(appDir, "Capsule.t4"))
	if err != nil {
		t.Fatal(err)
	}
	capsuleText := string(capsuleRaw)
	for _, want := range []string{`capsule PaletteDesk:`, `id "tetra://surface-apps/palettedesk"`, `entry "src/main.tetra"`, `source "src"`, `target "` + mustHostTarget(t) + `"`, `target "wasm32-web"`} {
		if !strings.Contains(capsuleText, want) {
			t.Fatalf("Capsule.t4 missing %q:\n%s", want, capsuleText)
		}
	}
	sourceRaw, err := os.ReadFile(filepath.Join(appDir, "src", "main.tetra"))
	if err != nil {
		t.Fatal(err)
	}
	sourceText := string(sourceRaw)
	for _, want := range []string{"import lib.core.surface as surface", "import lib.core.block as block", "import lib.core.morph as morph", "morph.expand_"} {
		if !strings.Contains(sourceText, want) {
			t.Fatalf("src/main.tetra missing %q:\n%s", want, sourceText)
		}
	}
	assertSurfaceTemplateSourceHasNoForbiddenRuntime(t, sourceText)

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"check", appDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("surface scaffold check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	outPath := filepath.Join(dir, "palette-desk")
	code = runCLI([]string{"build", "--target", mustHostTarget(t), "-o", outPath, appDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("surface scaffold build exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"run", "--target", mustHostTarget(t), appDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("surface scaffold run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestNewSurfaceAppGeneratesAllP21TemplateKinds(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	for _, kind := range []string{"command-palette", "settings", "dashboard", "editor-shell", "studio-shell", "multi-window-notes", "web-canvas"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			appDir := filepath.Join(dir, "Surface"+strings.ReplaceAll(kind, "-", ""))
			var stdout, stderr bytes.Buffer
			code := runCLI([]string{"new", "surface-app", "--template", kind, appDir}, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("new surface-app %s exit code = %d, stdout=%q stderr=%q", kind, code, stdout.String(), stderr.String())
			}
			metaRaw, err := os.ReadFile(filepath.Join(appDir, "surface-template.json"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(metaRaw), `"template": "`+kind+`"`) || !strings.Contains(string(metaRaw), `"model": "surface-project-template-v1"`) {
				t.Fatalf("surface-template.json missing template metadata for %s:\n%s", kind, string(metaRaw))
			}
			sourceRaw, err := os.ReadFile(filepath.Join(appDir, "src", "main.tetra"))
			if err != nil {
				t.Fatal(err)
			}
			sourceText := string(sourceRaw)
			for _, want := range []string{"import lib.core.surface as surface", "import lib.core.block as block", "import lib.core.morph as morph"} {
				if !strings.Contains(sourceText, want) {
					t.Fatalf("%s source missing %q:\n%s", kind, want, sourceText)
				}
			}
			if (kind == "multi-window-notes" || kind == "studio-shell") && !strings.Contains(sourceText, "import lib.core.surface_app_shell as shell") {
				t.Fatalf("%s source missing app shell import:\n%s", kind, sourceText)
			}
			if kind == "studio-shell" {
				for _, want := range []string{"morph.recipe_app_shell()", "morph.recipe_toolbar()", "morph.recipe_split_pane()", "morph.recipe_status_bar()"} {
					if !strings.Contains(sourceText, want) {
						t.Fatalf("studio-shell source missing %q:\n%s", want, sourceText)
					}
				}
			}
			if kind == "web-canvas" {
				capsuleRaw, err := os.ReadFile(filepath.Join(appDir, "Capsule.t4"))
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(capsuleRaw), `target "wasm32-web"`) {
					t.Fatalf("web-canvas capsule missing wasm32-web target:\n%s", string(capsuleRaw))
				}
			}
			assertSurfaceTemplateSourceHasNoForbiddenRuntime(t, sourceText)
		})
	}
}

func TestNewSurfaceAppRejectsUnknownTemplateKind(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"new", "surface-app", "--template", "react-dashboard", filepath.Join(dir, "BadApp")}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("new surface-app exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown surface app template") {
		t.Fatalf("stderr = %q, want unknown template diagnostic", stderr.String())
	}
}

func assertSurfaceTemplateSourceHasNoForbiddenRuntime(t *testing.T, source string) {
	t.Helper()
	for _, forbidden := range []string{
		"React",
		"Electron",
		"Chromium",
		"DOM",
		"CSS",
		"JavaScript",
		"lib.core.widgets",
		"lib.core.component",
		"Button",
		"Card",
		"TextField",
		"TextBox",
		"platform widget",
		"native widget",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("generated Surface template source contains forbidden runtime/core-widget token %q:\n%s", forbidden, source)
		}
	}
}
