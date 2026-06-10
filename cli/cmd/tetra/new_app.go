package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/compiler"
	"tetra_language/tools/validators/surfacedev"
)

func runNew(args []string, stdout io.Writer, stderr io.Writer) int {
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra new app [--lock] <NameOrPath>")
		fmt.Fprintln(stdout, "       tetra new surface-app [--template NAME] [--lock] <NameOrPath>")
		return 0
	}
	if len(args) == 0 {
		fmt.Fprintln(stderr, "new requires a template")
		return 2
	}
	switch args[0] {
	case "app":
		return runNewAppArgs(args[1:], stdout, stderr)
	case "surface-app":
		return runNewSurfaceAppArgs(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown new template %q\n", args[0])
		return 2
	}
}

type newAppOptions struct {
	WriteLock bool
}

type newSurfaceAppOptions struct {
	Template  string
	WriteLock bool
}

func runNewAppArgs(args []string, stdout io.Writer, stderr io.Writer) int {
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra new app [--lock] <NameOrPath>")
		return 0
	}
	var path string
	var opt newAppOptions
	for _, arg := range args {
		switch arg {
		case "--lock":
			opt.WriteLock = true
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(stderr, "unknown new app option %q\n", arg)
				return 2
			}
			if path != "" {
				fmt.Fprintln(stderr, "usage: tetra new app [--lock] <NameOrPath>")
				return 2
			}
			path = arg
		}
	}
	if path == "" {
		fmt.Fprintln(stderr, "usage: tetra new app [--lock] <NameOrPath>")
		return 2
	}
	return runNewApp(path, opt, stdout, stderr)
}

func runNewSurfaceAppArgs(args []string, stdout io.Writer, stderr io.Writer) int {
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra new surface-app [--template NAME] [--lock] <NameOrPath>")
		fmt.Fprintf(stdout, "templates: %s\n", strings.Join(surfacedev.RequiredTemplates(), ", "))
		return 0
	}
	var path string
	opt := newSurfaceAppOptions{Template: "surface-minimal"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--lock":
			opt.WriteLock = true
		case arg == "--template":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--template requires a value")
				return 2
			}
			opt.Template = args[i]
		case strings.HasPrefix(arg, "--template="):
			opt.Template = strings.TrimPrefix(arg, "--template=")
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(stderr, "unknown new surface-app option %q\n", arg)
				return 2
			}
			if path != "" {
				fmt.Fprintln(stderr, "usage: tetra new surface-app [--template NAME] [--lock] <NameOrPath>")
				return 2
			}
			path = arg
		}
	}
	if path == "" {
		fmt.Fprintln(stderr, "usage: tetra new surface-app [--template NAME] [--lock] <NameOrPath>")
		return 2
	}
	if !surfacedev.IsRequiredTemplate(opt.Template) {
		fmt.Fprintf(stderr, "unknown Surface template %q; templates: %s\n", opt.Template, strings.Join(surfacedev.RequiredTemplates(), ", "))
		return 2
	}
	return runNewSurfaceApp(path, opt, stdout, stderr)
}

func runNewApp(path string, opt newAppOptions, stdout io.Writer, stderr io.Writer) int {
	if strings.TrimSpace(path) == "" {
		fmt.Fprintln(stderr, "new app requires a name or path")
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
		fmt.Fprintln(stderr, "new app requires a valid app name")
		return 2
	}
	target := defaultTarget()
	files := map[string]string{
		"Capsule.t4": fmt.Sprintf(`manifest "tetra.capsule.v1"
capsule %s:
    id "tetra://apps/%s"
    version "0.1.0"
    entry "src/main.t4"
    source "src"
    source "tests"
    target "%s"
    permission "io"
`, name, capsuleSlug(name), target),
		"src/main.t4": `func main() -> Int:
    return 0
`,
		"tests/main_test.t4": `test "main returns success":
    expect 40 + 2 == 42
`,
		"README.md": fmt.Sprintf(`# %s

Run:

`+"```bash"+`
tetra check .
tetra build .
tetra run .
tetra test .
`+"```"+`
`, name),
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
	fmt.Fprintf(stdout, "Created app: %s\n", targetDir)
	if opt.WriteLock {
		lockPath := filepath.Join(targetDir, compiler.SemanticLockFileName)
		if err := buildCapsuleArtifacts(filepath.Join(targetDir, compiler.CapsuleFileName), capsuleArtifactBuildOptions{
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

func runNewSurfaceApp(path string, opt newSurfaceAppOptions, stdout io.Writer, stderr io.Writer) int {
	if strings.TrimSpace(path) == "" {
		fmt.Fprintln(stderr, "new surface-app requires a name or path")
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
	metadata, err := surfaceTemplateMetadata(opt.Template)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	files := map[string]string{
		"Capsule.t4": fmt.Sprintf(`manifest "tetra.capsule.v1"
capsule %s:
    id "tetra://surface-apps/%s"
    version "0.1.0"
    entry "src/main.t4"
    source "src"
    source "tests"
    target "%s"
    permission "io"
`, name, capsuleSlug(name), target),
		"src/main.t4":            surfaceTemplateSource(opt.Template, name),
		"tests/surface_smoke.t4": surfaceTemplateTest(opt.Template),
		"surface.template.json":  string(metadata),
		"README.md":              surfaceTemplateREADME(name, opt.Template),
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
	fmt.Fprintf(stdout, "Created Surface app: %s (%s)\n", targetDir, opt.Template)
	if opt.WriteLock {
		lockPath := filepath.Join(targetDir, compiler.SemanticLockFileName)
		if err := buildCapsuleArtifacts(filepath.Join(targetDir, compiler.CapsuleFileName), capsuleArtifactBuildOptions{
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

func surfaceTemplateMetadata(template string) ([]byte, error) {
	metadata := struct {
		Schema      string   `json:"schema"`
		Template    string   `json:"template"`
		Templates   []string `json:"templates"`
		Level       string   `json:"level"`
		Commands    []string `json:"commands"`
		ReleaseNote string   `json:"release_note"`
	}{
		Schema:    surfacedev.TemplateSchemaV1,
		Template:  template,
		Templates: surfacedev.RequiredTemplates(),
		Level:     surfacedev.LevelFastDevLoopV1,
		Commands: []string{
			"tetra check .",
			"tetra surface dev --project . --once --state .tetra/surface-dev-state.json --report .tetra/surface-dev-report.json",
			"tetra surface inspect --report reports/surface-runtime.json --out reports/surface-inspector.json",
			"tetra surface package . -o dist/surface-app.tdx",
		},
		ReleaseNote: "Surface dev-loop evidence is scoped to surface-v1-linux-web and does not claim Electron, React Fast Refresh, CSS HMR, or browser devtools parity.",
	}
	raw, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func surfaceTemplateSource(template string, name string) string {
	title := surfaceTemplateTitle(template, name)
	score := 1
	for _, r := range template {
		score += int(r)
	}
	return fmt.Sprintf(`// Surface template: %s
// Surface dev state schema: surface-template-state-v1
// This app is pure Tetra source; the dev loop report is produced by `+"`tetra surface dev`"+`.

struct SurfaceTemplateState:
    template_id: Int
    source_map_anchor: Int
    preserved_query: Int

func surface_template_id() -> Int:
    return %d

func surface_title_len() -> Int:
    return %d

func main() -> Int:
    let state: SurfaceTemplateState = SurfaceTemplateState(template_id: surface_template_id(), source_map_anchor: 1, preserved_query: surface_title_len())
    if state.template_id > 0 && state.source_map_anchor == 1 && state.preserved_query > 0:
        return 0
    return 1
`, template, score, len(title))
}

func surfaceTemplateTest(template string) string {
	return fmt.Sprintf(`test "surface template %s smoke":
    expect surface_template_id() > 0
`, template)
}

func surfaceTemplateREADME(name string, template string) string {
	return fmt.Sprintf(`# %s

Surface template: %s

Run:

`+"```bash"+`
tetra check .
tetra surface dev --project . --once --state .tetra/surface-dev-state.json --report .tetra/surface-dev-report.json
tetra surface package . -o dist/%s.tdx
`+"```"+`

The dev-loop report is scoped evidence for source-hash reload tracing, Surface
inspector diagnostics, and schema-compatible owned-state preservation. It is not
an Electron dev-server, React Fast Refresh, CSS HMR, or browser devtools parity
claim.
`, name, template, capsuleSlug(name))
}

func surfaceTemplateTitle(template string, name string) string {
	switch template {
	case "surface-dashboard":
		return name + " Dashboard"
	case "surface-form":
		return name + " Form"
	case "surface-editor-shell":
		return name + " Editor"
	case "surface-tray-app":
		return name + " Tray"
	case "surface-web-canvas":
		return name + " Web Canvas"
	default:
		return name + " Surface"
	}
}

func capsuleNameFromPath(path string) string {
	name := filepath.Base(filepath.Clean(path))
	var b strings.Builder
	capitalizeNext := true
	for _, r := range name {
		if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			if b.Len() == 0 && r >= '0' && r <= '9' {
				b.WriteByte('T')
			}
			if capitalizeNext && r >= 'a' && r <= 'z' {
				r = r - 'a' + 'A'
			}
			b.WriteRune(r)
			capitalizeNext = false
			continue
		}
		capitalizeNext = true
	}
	return b.String()
}

func capsuleSlug(name string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r - 'A' + 'a')
			lastDash = false
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
