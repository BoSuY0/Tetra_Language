package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tetra_language/compiler"
	ctarget "tetra_language/compiler/target"
)

type surfaceDevReport struct {
	Schema                 string                 `json:"schema"`
	Model                  string                 `json:"model"`
	ReleaseScope           string                 `json:"release_scope"`
	Command                string                 `json:"command"`
	Source                 string                 `json:"source"`
	Target                 string                 `json:"target"`
	Mode                   string                 `json:"mode"`
	ReloadSemantics        string                 `json:"reload_semantics"`
	ProcessRestartRequired bool                   `json:"process_restart_required"`
	HotReloadClaim         bool                   `json:"hot_reload_claim"`
	Watch                  bool                   `json:"watch"`
	SupportedTargets       []string               `json:"supported_targets"`
	Steps                  []surfaceDevStep       `json:"steps"`
	SourceDiagnostics      []surfaceDevDiagnostic `json:"source_diagnostics"`
	NegativeGuards         surfaceDevGuards       `json:"negative_guards"`
	Pass                   bool                   `json:"pass"`
}

type surfaceDevStep struct {
	Name            string   `json:"name"`
	Kind            string   `json:"kind"`
	ChangedPath     string   `json:"changed_path"`
	OutputPath      string   `json:"output_path"`
	DurationMS      int64    `json:"duration_ms"`
	CompiledModules []string `json:"compiled_modules"`
	CacheHits       []string `json:"cache_hits"`
	Pass            bool     `json:"pass"`
	Error           string   `json:"error,omitempty"`
}

type surfaceDevDiagnostic struct {
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Pass     bool   `json:"pass"`
}

type surfaceDevGuards struct {
	NoHotReloadClaim                   bool `json:"no_hot_reload_claim"`
	FullRestartDocumentedAsFastRebuild bool `json:"full_restart_documented_as_fast_rebuild"`
	NoElectronDevServer                bool `json:"no_electron_dev_server"`
	NoReactFastRefresh                 bool `json:"no_react_fast_refresh"`
	NoDOMHotReload                     bool `json:"no_dom_hot_reload"`
}

type surfaceDevChange struct {
	kind string
	path string
}

func runSurface(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printSurfaceUsage(stderr)
		return 2
	}
	if isHelpArgs(args) {
		printSurfaceUsage(stdout)
		return 0
	}
	switch args[0] {
	case "dev":
		return runSurfaceDev(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown surface command %q\n", args[0])
		printSurfaceUsage(stderr)
		return 2
	}
}

func printSurfaceUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: tetra surface <dev> [options]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "commands:")
	fmt.Fprintln(w, "  dev    run the scoped Surface fast rebuild developer loop")
}

func runSurfaceDev(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("surface dev", flag.ContinueOnError)
	fs.SetOutput(stderr)
	sourceFlag := fs.String("source", "", "Surface app source path; defaults to positional input or discovered project entry")
	targetFlag := fs.String("target", "", "target triple; current fast rebuild evidence is linux-x64")
	outDirFlag := fs.String("out-dir", "", "directory for dev-loop build artifacts")
	reportPath := fs.String("report", "", "write tetra.surface.dev-workflow.v1 JSON report")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text or json")
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	watch := fs.Bool("watch", false, "reserve watch-mode metadata; current command records one fast rebuild loop")
	var changeFlags multiFlag
	fs.Var(&changeFlags, "change-file", "changed Surface path as kind:path; repeat for token, recipe, and source")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	if fs.NArg() > 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "surface dev accepts at most one input path")
		return 2
	}
	source := strings.TrimSpace(*sourceFlag)
	if source != "" && fs.NArg() == 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "surface dev accepts either --source or one positional input path, not both")
		return 2
	}
	if source == "" && fs.NArg() == 1 {
		source = fs.Arg(0)
	}
	input, worldOpt, projectCtx, err := resolveCLIInput(source)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	rawTarget := strings.TrimSpace(*targetFlag)
	if rawTarget == "" {
		rawTarget = projectDefaultTarget(projectCtx)
	}
	tgt, ok := parseBuildTargetOrReport(rawTarget, *diagnostics, stderr)
	if !ok {
		return 2
	}
	if tgt.Triple != "linux-x64" {
		writeValidationDiagnostic(stderr, *diagnostics, "surface dev fast rebuild evidence is currently scoped to linux-x64; wasm32-web/headless are documented targets without hot reload promotion")
		return 2
	}
	if err := validateDiscoveredProjectLock(projectCtx, tgt.Triple); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	targetLinkObjects, err := projectLinkObjects(projectCtx, tgt.Triple, nil)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	opt, err := buildOptions("exe", "auto", false, "", targetLinkObjects, *jobs)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 2
	}
	opt.ProjectRoot = worldOpt.Root
	opt.SourceRoots = worldOpt.SourceRoots
	opt.DependencyRoots = worldOpt.DependencyRoots

	outDir, cleanup, err := surfaceDevOutputDir(*outDirFlag)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	defer cleanup()

	report := newSurfaceDevReport(input, tgt, *watch)
	changes, err := parseSurfaceDevChanges(changeFlags)
	if err != nil {
		writeValidationDiagnostic(stderr, *diagnostics, err.Error())
		return 2
	}
	report.SourceDiagnostics = append(report.SourceDiagnostics, surfaceDevInfoDiagnostic("source", input))
	for _, change := range changes {
		report.SourceDiagnostics = append(report.SourceDiagnostics, surfaceDevInfoDiagnostic(change.kind, change.path))
	}

	initial := runSurfaceDevBuild("initial build", "initial", "", input, outDir, tgt, opt)
	report.Steps = append(report.Steps, initial)
	if !initial.Pass {
		report.Pass = false
		report.SourceDiagnostics = []surfaceDevDiagnostic{surfaceDevErrorDiagnostic(input, initial.Error)}
		writeSurfaceDevReportIfRequested(*reportPath, report, stderr)
		writeDiagnostic(stderr, *diagnostics, errors.New(initial.Error))
		return 1
	}
	warm := runSurfaceDevBuild("warm rebuild", "warm-cache", "", input, outDir, tgt, opt)
	report.Steps = append(report.Steps, warm)
	if !warm.Pass {
		report.Pass = false
		report.SourceDiagnostics = []surfaceDevDiagnostic{surfaceDevErrorDiagnostic(input, warm.Error)}
		writeSurfaceDevReportIfRequested(*reportPath, report, stderr)
		writeDiagnostic(stderr, *diagnostics, errors.New(warm.Error))
		return 1
	}

	restoreFns := make([]func(), 0, len(changes))
	for _, change := range changes {
		restore, err := appendSurfaceDevChange(change)
		if err != nil {
			report.Pass = false
			report.SourceDiagnostics = []surfaceDevDiagnostic{surfaceDevErrorDiagnostic(change.path, err.Error())}
			writeSurfaceDevReportIfRequested(*reportPath, report, stderr)
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		restoreFns = append(restoreFns, restore)
		step := runSurfaceDevBuild(surfaceDevStepName(change.kind), change.kind+"-change", change.path, input, outDir, tgt, opt)
		report.Steps = append(report.Steps, step)
		if !step.Pass {
			report.Pass = false
			report.SourceDiagnostics = []surfaceDevDiagnostic{surfaceDevErrorDiagnostic(change.path, step.Error)}
			writeSurfaceDevReportIfRequested(*reportPath, report, stderr)
			writeDiagnostic(stderr, *diagnostics, errors.New(step.Error))
			for i := len(restoreFns) - 1; i >= 0; i-- {
				restoreFns[i]()
			}
			return 1
		}
	}
	for i := len(restoreFns) - 1; i >= 0; i-- {
		restoreFns[i]()
	}

	report.Pass = surfaceDevReportPass(report)
	if err := writeSurfaceDevReportIfRequested(*reportPath, report, stderr); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if report.Pass {
		fmt.Fprintf(stdout, "Surface dev fast rebuild report: %s\n", defaultReportLabel(*reportPath))
		return 0
	}
	writeDiagnostic(stderr, *diagnostics, errors.New("surface dev fast rebuild report did not satisfy required token/recipe/source evidence"))
	return 1
}

func newSurfaceDevReport(input string, tgt ctarget.Target, watch bool) surfaceDevReport {
	return surfaceDevReport{
		Schema:                 "tetra.surface.dev-workflow.v1",
		Model:                  "surface-dev-workflow-v1",
		ReleaseScope:           "surface-v1-linux-web",
		Command:                "tetra surface dev",
		Source:                 filepath.ToSlash(input),
		Target:                 tgt.Triple,
		Mode:                   "fast-rebuild",
		ReloadSemantics:        "fast-rebuild",
		ProcessRestartRequired: true,
		HotReloadClaim:         false,
		Watch:                  watch,
		SupportedTargets:       []string{"headless", "linux-x64", "wasm32-web"},
		NegativeGuards: surfaceDevGuards{
			NoHotReloadClaim:                   true,
			FullRestartDocumentedAsFastRebuild: true,
			NoElectronDevServer:                true,
			NoReactFastRefresh:                 true,
			NoDOMHotReload:                     true,
		},
	}
}

func surfaceDevOutputDir(raw string) (string, func(), error) {
	if strings.TrimSpace(raw) != "" {
		if err := os.MkdirAll(raw, 0o755); err != nil {
			return "", func() {}, err
		}
		abs, err := filepath.Abs(raw)
		if err != nil {
			return "", func() {}, err
		}
		return abs, func() {}, nil
	}
	dir, err := os.MkdirTemp("", "tetra-surface-dev-*")
	if err != nil {
		return "", func() {}, err
	}
	return dir, func() { _ = os.RemoveAll(dir) }, nil
}

func parseSurfaceDevChanges(values []string) ([]surfaceDevChange, error) {
	changes := make([]surfaceDevChange, 0, len(values))
	seen := map[string]bool{}
	for _, raw := range values {
		kind, path, ok := strings.Cut(raw, ":")
		if !ok {
			return nil, fmt.Errorf("surface dev --change-file must use kind:path")
		}
		kind = strings.TrimSpace(kind)
		path = strings.TrimSpace(path)
		switch kind {
		case "token", "recipe", "source", "block", "morph":
		default:
			return nil, fmt.Errorf("surface dev --change-file kind %q is unsupported", kind)
		}
		if path == "" {
			return nil, fmt.Errorf("surface dev --change-file path is required")
		}
		if seen[kind] {
			return nil, fmt.Errorf("surface dev --change-file duplicate kind %q", kind)
		}
		seen[kind] = true
		changes = append(changes, surfaceDevChange{kind: kind, path: path})
	}
	sort.Slice(changes, func(i, j int) bool {
		order := map[string]int{"token": 0, "recipe": 1, "source": 2, "block": 3, "morph": 4}
		return order[changes[i].kind] < order[changes[j].kind]
	})
	return changes, nil
}

func runSurfaceDevBuild(name string, kind string, changedPath string, input string, outDir string, tgt ctarget.Target, opt compiler.BuildOptions) surfaceDevStep {
	outputPath := filepath.Join(outDir, kind, "app"+tgt.ExeExt)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return surfaceDevStep{Name: name, Kind: kind, ChangedPath: filepath.ToSlash(changedPath), OutputPath: filepath.ToSlash(outputPath), Pass: false, Error: err.Error()}
	}
	start := time.Now()
	stats, err := compiler.BuildFileWithStatsOpt(input, outputPath, tgt.Triple, opt)
	duration := time.Since(start).Milliseconds()
	step := surfaceDevStep{
		Name:        name,
		Kind:        kind,
		ChangedPath: filepath.ToSlash(changedPath),
		OutputPath:  filepath.ToSlash(outputPath),
		DurationMS:  duration,
		Pass:        err == nil,
	}
	if stats != nil {
		step.CompiledModules = append([]string(nil), stats.CompiledModules...)
		step.CacheHits = append([]string(nil), stats.CacheHits...)
		sort.Strings(step.CompiledModules)
		sort.Strings(step.CacheHits)
	}
	if err != nil {
		step.Error = err.Error()
	}
	return step
}

func appendSurfaceDevChange(change surfaceDevChange) (func(), error) {
	raw, err := os.ReadFile(change.path)
	if err != nil {
		return func() {}, err
	}
	next := append([]byte(nil), raw...)
	next = append(next, []byte(fmt.Sprintf("\n// tetra surface dev %s fast-rebuild change %d\n", change.kind, time.Now().UnixNano()))...)
	if err := os.WriteFile(change.path, next, 0o644); err != nil {
		return func() {}, err
	}
	return func() { _ = os.WriteFile(change.path, raw, 0o644) }, nil
}

func surfaceDevInfoDiagnostic(kind string, path string) surfaceDevDiagnostic {
	if kind == "" {
		kind = classifySurfaceDevPath(path, nil)
	}
	codeKind := strings.ToUpper(strings.ReplaceAll(kind, "-", "_"))
	return surfaceDevDiagnostic{
		Kind:     kind,
		Path:     filepath.ToSlash(path),
		Line:     1,
		Column:   1,
		Code:     "SURFACE_DEV_" + codeKind + "_PATH",
		Message:  kind + " file participates in Surface fast rebuild",
		Severity: "info",
		Pass:     true,
	}
}

func surfaceDevErrorDiagnostic(path string, message string) surfaceDevDiagnostic {
	diag := compiler.DiagnosticFromError(errors.New(message))
	kind := classifySurfaceDevPath(path, nil)
	line := diag.Line
	column := diag.Column
	if line <= 0 {
		line = 1
	}
	if column <= 0 {
		column = 1
	}
	diagPath := diag.File
	if diagPath == "" {
		diagPath = path
	}
	return surfaceDevDiagnostic{
		Kind:     kind,
		Path:     filepath.ToSlash(diagPath),
		Line:     line,
		Column:   column,
		Code:     diag.Code,
		Message:  diag.Message,
		Severity: "error",
		Pass:     false,
	}
}

func classifySurfaceDevPath(path string, content []byte) string {
	lower := strings.ToLower(filepath.ToSlash(path))
	if len(content) == 0 {
		if raw, err := os.ReadFile(path); err == nil {
			content = raw
		}
	}
	text := strings.ToLower(string(content))
	switch {
	case strings.Contains(lower, "token") || strings.Contains(text, "token"):
		return "token"
	case strings.Contains(lower, "recipe") || strings.Contains(text, "recipe"):
		return "recipe"
	case strings.Contains(lower, "morph") || strings.Contains(text, "morph"):
		return "morph"
	case strings.Contains(lower, "block") || strings.Contains(text, "block"):
		return "block"
	default:
		return "source"
	}
}

func surfaceDevStepName(kind string) string {
	switch kind {
	case "token":
		return "token rebuild"
	case "recipe":
		return "recipe rebuild"
	case "source":
		return "source rebuild"
	case "block":
		return "block rebuild"
	case "morph":
		return "morph rebuild"
	default:
		return kind + " rebuild"
	}
}

func surfaceDevReportPass(report surfaceDevReport) bool {
	kinds := map[string]surfaceDevStep{}
	for _, step := range report.Steps {
		if !step.Pass {
			return false
		}
		kinds[step.Kind] = step
	}
	warm, ok := kinds["warm-cache"]
	if !ok || len(warm.CompiledModules) != 0 || len(warm.CacheHits) == 0 {
		return false
	}
	for _, kind := range []string{"initial", "token-change", "recipe-change", "source-change"} {
		step, ok := kinds[kind]
		if !ok {
			return false
		}
		if kind != "initial" && len(step.CompiledModules) == 0 {
			return false
		}
	}
	diagKinds := map[string]bool{}
	for _, diag := range report.SourceDiagnostics {
		diagKinds[diag.Kind] = diag.Pass
	}
	return diagKinds["token"] && diagKinds["recipe"] && diagKinds["source"]
}

func writeSurfaceDevReportIfRequested(path string, report surfaceDevReport, stderr io.Writer) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := writeJSON(path, report); err != nil {
		fmt.Fprintf(stderr, "surface dev report write failed: %v\n", err)
		return err
	}
	return nil
}

func defaultReportLabel(path string) string {
	if strings.TrimSpace(path) == "" {
		return "(not written)"
	}
	return path
}
