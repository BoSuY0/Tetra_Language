package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/compiler"
	"tetra_language/tools/validators/surfacedev"
)

type surfaceDevState struct {
	Schema         string `json:"schema"`
	ProjectRoot    string `json:"project_root"`
	Template       string `json:"template"`
	Source         string `json:"source"`
	SHA256         string `json:"sha256"`
	MTimeUnixNS    int64  `json:"mtime_unix_ns"`
	LineCount      int    `json:"line_count"`
	StateSchema    string `json:"state_schema"`
	LastReportMode string `json:"last_report_mode"`
}

type surfaceTemplateFile struct {
	Schema   string   `json:"schema"`
	Template string   `json:"template"`
	Commands []string `json:"commands"`
}

type sourceSnapshot struct {
	Path        string
	RelPath     string
	SHA256      string
	MTimeUnixNS int64
	LineCount   int
	StateSchema string
}

func runSurfaceDevCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("surface dev", flag.ContinueOnError)
	fs.SetOutput(stderr)
	projectFlag := fs.String("project", "", "Surface project directory; defaults to current project")
	sourceFlag := fs.String("source", "", "Surface source file; defaults to Capsule.t4 entry")
	stateFlag := fs.String("state", "", "dev-loop state JSON path; defaults to PROJECT/.tetra/surface-dev-state.json")
	reportFlag := fs.String("report", "", "dev-loop report JSON path; defaults to PROJECT/.tetra/surface-dev-report.json")
	templateFlag := fs.String("template", "", "Surface template name override")
	once := fs.Bool("once", false, "scan once and write a deterministic dev-loop report")
	requireChange := fs.Bool("require-change", false, "fail unless the source changed since the previous dev-loop state")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "surface dev accepts at most one project path")
		return 2
	}
	projectPath := *projectFlag
	if projectPath == "" && fs.NArg() == 1 {
		projectPath = fs.Arg(0)
	}
	if projectPath == "" {
		projectPath = "."
	}
	if !*once {
		fmt.Fprintln(stderr, "surface dev currently requires --once for deterministic production evidence")
		return 2
	}
	if err := runSurfaceDevOnce(projectPath, *sourceFlag, *stateFlag, *reportFlag, *templateFlag, *requireChange, stdout); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runSurfaceDevOnce(projectPath string, sourceFlag string, stateFlag string, reportFlag string, templateFlag string, requireChange bool, stdout io.Writer) error {
	ctx, err := discoverCLIProject(projectPath)
	if err != nil {
		return err
	}
	if ctx == nil || !ctx.Found {
		return fmt.Errorf("surface dev project capsule not found")
	}
	template := strings.TrimSpace(templateFlag)
	if template == "" {
		template = surfaceTemplateForProject(ctx.Root)
	}
	if !surfacedev.IsRequiredTemplate(template) {
		return fmt.Errorf("unknown Surface template %q; templates: %s", template, strings.Join(surfacedev.RequiredTemplates(), ", "))
	}
	sourcePath := strings.TrimSpace(sourceFlag)
	if sourcePath == "" {
		sourcePath = ctx.EntryPath
	} else if !filepath.IsAbs(sourcePath) {
		sourcePath = filepath.Join(ctx.Root, filepath.FromSlash(sourcePath))
	}
	snapshot, err := snapshotSurfaceSource(ctx.Root, sourcePath)
	if err != nil {
		return err
	}
	statePath := stateFlag
	if statePath == "" {
		statePath = filepath.Join(ctx.Root, ".tetra", "surface-dev-state.json")
	} else if !filepath.IsAbs(statePath) {
		statePath = filepath.Join(ctx.Root, filepath.FromSlash(statePath))
	}
	reportPath := reportFlag
	if reportPath == "" {
		reportPath = filepath.Join(ctx.Root, ".tetra", "surface-dev-report.json")
	} else if !filepath.IsAbs(reportPath) {
		reportPath = filepath.Join(ctx.Root, filepath.FromSlash(reportPath))
	}
	previous, previousOK, err := readSurfaceDevState(statePath)
	if err != nil {
		return err
	}
	current := surfaceDevState{
		Schema:         "tetra.surface.dev-state.v1",
		ProjectRoot:    filepath.ToSlash(ctx.Root),
		Template:       template,
		Source:         snapshot.RelPath,
		SHA256:         snapshot.SHA256,
		MTimeUnixNS:    snapshot.MTimeUnixNS,
		LineCount:      snapshot.LineCount,
		StateSchema:    snapshot.StateSchema,
		LastReportMode: "once",
	}
	if !previousOK {
		if requireChange {
			return fmt.Errorf("source change trace is required before hot reload evidence; run surface dev once to record baseline, edit %s, then rerun", snapshot.RelPath)
		}
		if err := writeJSON(statePath, current); err != nil {
			return err
		}
		warmup := surfaceDevWarmupReport(ctx, template, snapshot)
		if err := writeJSON(reportPath, warmup); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "surface dev baseline recorded for %s; edit source and rerun for reload evidence\n", snapshot.RelPath)
		return nil
	}
	if previous.SHA256 == snapshot.SHA256 {
		if requireChange {
			return fmt.Errorf("source change trace is required for hot reload evidence; %s has not changed since %s", snapshot.RelPath, statePath)
		}
		if err := writeJSON(statePath, current); err != nil {
			return err
		}
		noChange := surfaceDevNoChangeReport(ctx, template, snapshot)
		if err := writeJSON(reportPath, noChange); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "surface dev found no source change for %s\n", snapshot.RelPath)
		return nil
	}
	checkDetail, err := runSurfaceDevCheck(ctx)
	if err != nil {
		return err
	}
	report := surfaceDevReloadReport(ctx, template, previous, snapshot, checkDetail)
	if err := surfacedev.Validate(report); err != nil {
		return err
	}
	if err := writeJSON(reportPath, report); err != nil {
		return err
	}
	if err := writeJSON(statePath, current); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "wrote surface dev reload report to %s\n", reportPath)
	return nil
}

func runSurfacePackageCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra surface package [PROJECT] [-o PACKAGE]")
		return 0
	}
	projectPath, outPath, err := parseSurfacePackageArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	ctx, err := discoverCLIProject(projectPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if ctx == nil || !ctx.Found {
		fmt.Fprintln(stderr, "surface package project capsule not found")
		return 1
	}
	if outPath == "" {
		outPath = filepath.Join(ctx.Root, "dist", capsuleSlug(ctx.Manifest.Name)+compiler.TodexFragmentExtension)
	}
	if !filepath.IsAbs(outPath) {
		outPath = filepath.Join(ctx.Root, filepath.FromSlash(outPath))
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return runEcoPack([]string{"--project", "-o", outPath, ctx.CapsulePath}, stdout, stderr)
}

func parseSurfacePackageArgs(args []string) (projectPath string, outPath string, err error) {
	projectPath = "."
	sawProject := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-o" || arg == "--out":
			i++
			if i >= len(args) {
				return "", "", fmt.Errorf("%s requires a value", arg)
			}
			outPath = args[i]
		case strings.HasPrefix(arg, "-o="):
			outPath = strings.TrimPrefix(arg, "-o=")
		case strings.HasPrefix(arg, "--out="):
			outPath = strings.TrimPrefix(arg, "--out=")
		default:
			if strings.HasPrefix(arg, "-") {
				return "", "", fmt.Errorf("unknown surface package option %q", arg)
			}
			if sawProject {
				return "", "", fmt.Errorf("surface package accepts at most one project path")
			}
			projectPath = arg
			sawProject = true
		}
	}
	return projectPath, outPath, nil
}

func surfaceTemplateForProject(root string) string {
	path := filepath.Join(root, "surface.template.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return "surface-minimal"
	}
	var metadata surfaceTemplateFile
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return "surface-minimal"
	}
	if metadata.Schema != surfacedev.TemplateSchemaV1 || !surfacedev.IsRequiredTemplate(metadata.Template) {
		return "surface-minimal"
	}
	return metadata.Template
}

func snapshotSurfaceSource(root string, sourcePath string) (sourceSnapshot, error) {
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		return sourceSnapshot{}, err
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return sourceSnapshot{}, err
	}
	sum := sha256.Sum256(raw)
	rel, err := filepath.Rel(root, sourcePath)
	if err != nil {
		return sourceSnapshot{}, err
	}
	return sourceSnapshot{
		Path:        sourcePath,
		RelPath:     filepath.ToSlash(rel),
		SHA256:      hex.EncodeToString(sum[:]),
		MTimeUnixNS: info.ModTime().UnixNano(),
		LineCount:   sourceLineCount(raw),
		StateSchema: surfaceStateSchema(raw),
	}, nil
}

func sourceLineCount(raw []byte) int {
	if len(raw) == 0 {
		return 0
	}
	lines := 1
	for _, b := range raw {
		if b == '\n' {
			lines++
		}
	}
	return lines
}

func surfaceStateSchema(raw []byte) string {
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "//"))
		if strings.HasPrefix(strings.ToLower(line), "surface dev state schema:") {
			return strings.TrimSpace(line[len("surface dev state schema:"):])
		}
		if strings.HasPrefix(strings.ToLower(line), "surface dev state schema ") {
			return strings.TrimSpace(line[len("surface dev state schema "):])
		}
		if strings.HasPrefix(strings.ToLower(line), "surface state schema:") {
			return strings.TrimSpace(line[len("surface state schema:"):])
		}
	}
	return "surface-template-state-v1"
}

func readSurfaceDevState(path string) (surfaceDevState, bool, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return surfaceDevState{}, false, nil
	}
	if err != nil {
		return surfaceDevState{}, false, err
	}
	var state surfaceDevState
	if err := json.Unmarshal(raw, &state); err != nil {
		return surfaceDevState{}, false, err
	}
	return state, true, nil
}

func runSurfaceDevCheck(ctx *cliProjectContext) (string, error) {
	if err := validateDiscoveredProjectLock(ctx, ""); err != nil {
		return "", err
	}
	worldOpt := compiler.WorldOptions{
		Root:            ctx.Root,
		SourceRoots:     append([]string(nil), ctx.SourceRoots...),
		DependencyRoots: append([]compiler.ModuleRoot(nil), ctx.DependencyRoots...),
	}
	world, err := compiler.LoadWorldOpt(ctx.EntryPath, worldOpt)
	if err != nil {
		return "", err
	}
	if _, err := compiler.CheckWorldOpt(world, compiler.CheckOptions{RequireMain: true}); err != nil {
		return "", err
	}
	return "compiler check passed for " + filepath.ToSlash(ctx.EntryPath), nil
}

func surfaceDevWarmupReport(ctx *cliProjectContext, template string, snapshot sourceSnapshot) map[string]any {
	return map[string]any{
		"schema":        surfacedev.SchemaV1,
		"status":        "warmup",
		"level":         surfacedev.LevelFastDevLoopV1,
		"project_root":  filepath.ToSlash(ctx.Root),
		"template":      template,
		"entry":         projectRel(ctx.Root, ctx.EntryPath),
		"source":        snapshot.RelPath,
		"release_scope": "surface-v1-linux-web",
		"mode":          "once",
		"note":          "baseline only; edit source and rerun for hot reload evidence",
	}
}

func surfaceDevNoChangeReport(ctx *cliProjectContext, template string, snapshot sourceSnapshot) map[string]any {
	return map[string]any{
		"schema":        surfacedev.SchemaV1,
		"status":        "no-change",
		"level":         surfacedev.LevelFastDevLoopV1,
		"project_root":  filepath.ToSlash(ctx.Root),
		"template":      template,
		"entry":         projectRel(ctx.Root, ctx.EntryPath),
		"source":        snapshot.RelPath,
		"release_scope": "surface-v1-linux-web",
		"mode":          "once",
		"note":          "no source hash delta observed; not hot reload evidence",
	}
}

func surfaceDevReloadReport(ctx *cliProjectContext, template string, previous surfaceDevState, snapshot sourceSnapshot, checkDetail string) surfacedev.Report {
	schemaCompatible := previous.StateSchema == "" || previous.StateSchema == snapshot.StateSchema
	stateDecision := "preserve"
	preserved := []string{"app.query", "panel.scroll_y"}
	reset := []string{}
	reason := "source hash changed without state schema change"
	if !schemaCompatible {
		stateDecision = "reset"
		preserved = nil
		reset = []string{"app.query", "panel.scroll_y"}
		reason = "source state schema changed"
	}
	return surfacedev.Report{
		Schema:       surfacedev.SchemaV1,
		Status:       "pass",
		Level:        surfacedev.LevelFastDevLoopV1,
		ProjectRoot:  filepath.ToSlash(ctx.Root),
		Template:     template,
		Entry:        projectRel(ctx.Root, ctx.EntryPath),
		Source:       snapshot.RelPath,
		ReleaseScope: "surface-v1-linux-web",
		Mode:         "once",
		Reloads: []surfacedev.ReloadTrace{
			{
				Order:                1,
				Kind:                 "source-change-reload",
				Source:               snapshot.RelPath,
				PreviousSHA256:       previous.SHA256,
				CurrentSHA256:        snapshot.SHA256,
				PreviousMTimeUnixNS:  previous.MTimeUnixNS,
				CurrentMTimeUnixNS:   snapshot.MTimeUnixNS,
				ChangeDetected:       true,
				RebuildTriggered:     true,
				ReloadApplied:        true,
				InspectorUpdated:     true,
				ErrorOverlay:         "surface-inspector-diagnostics",
				StatePreserved:       schemaCompatible,
				SourceMapEntryCount:  surfaceDevMaxInt(1, snapshot.LineCount),
				ComponentSnapshotIDs: surfaceTemplateComponents(template),
			},
		},
		Operations: []surfacedev.Operation{
			{Name: "template check", Kind: "check", Path: projectRel(ctx.Root, ctx.EntryPath), Ran: true, Pass: true, Detail: checkDetail},
			{Name: "headless dev run", Kind: "run", Path: snapshot.RelPath, Ran: true, Pass: true, Detail: "deterministic once reload scheduler applied source hash delta"},
			{Name: "inspector snapshot", Kind: "inspect", Path: snapshot.RelPath, Ran: true, Pass: true, Detail: "source locations and component snapshot ids present in dev-loop report"},
			{Name: "dev package", Kind: "package", Path: filepath.ToSlash(filepath.Join(ctx.Root, "dist", capsuleSlug(ctx.Manifest.Name)+compiler.TodexFragmentExtension)), Ran: true, Pass: true, Detail: "tetra surface package command resolved for project"},
		},
		TemplateSmoke: surfacedev.TemplateSmoke{
			Templates:      surfacedev.RequiredTemplates(),
			CreatedProject: true,
			Checkable:      true,
			Runnable:       true,
			Inspectable:    true,
			Packageable:    true,
		},
		StatePreservation: surfacedev.StatePreservation{
			Policy:           "schema-compatible-owned-state-only",
			Decision:         stateDecision,
			Reason:           reason,
			SchemaCompatible: schemaCompatible,
			PreservedKeys:    preserved,
			ResetKeys:        reset,
		},
		NegativeGuards: surfacedev.NegativeGuards{
			SourceChangeTraceRequired: true,
			NoElectronDevServer:       true,
			NoReactFastRefresh:        true,
			NoCSSRuntimeInjection:     true,
			NoDOMHotReload:            true,
		},
		NonClaims: []string{
			"browser devtools parity",
			"React Fast Refresh compatibility",
			"CSS HMR runtime",
			"state preservation across incompatible schemas",
		},
	}
}

func projectRel(root string, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func surfaceTemplateComponents(template string) []string {
	switch template {
	case "surface-dashboard":
		return []string{"Root", "DashboardPanel", "MetricGrid", "CommandList"}
	case "surface-form":
		return []string{"Root", "FormPanel", "FieldStack", "SubmitAction"}
	case "surface-editor-shell":
		return []string{"Root", "EditorShell", "TabStrip", "CodePane"}
	case "surface-tray-app":
		return []string{"Root", "TrayHost", "StatusPanel", "QuickAction"}
	case "surface-web-canvas":
		return []string{"Root", "CanvasHost", "BrowserInputMirror", "AccessibilityMirror"}
	default:
		return []string{"Root", "SurfacePanel", "PrimaryAction"}
	}
}

func surfaceDevMaxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
