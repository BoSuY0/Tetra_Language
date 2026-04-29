package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"tetra_language/compiler"
	ctarget "tetra_language/compiler/target"
)

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

type smokeCaseReport struct {
	Name         string `json:"name"`
	SrcPath      string `json:"src_path"`
	OutPath      string `json:"out_path"`
	ExpectedExit int    `json:"expected_exit"`
	ActualExit   *int   `json:"actual_exit,omitempty"`
	Ran          bool   `json:"ran"`
	Pass         bool   `json:"pass"`
	Error        string `json:"error,omitempty"`
}

type smokeReport struct {
	Timestamp    string            `json:"timestamp"`
	Target       string            `json:"target"`
	BuildOnly    bool              `json:"build_only,omitempty"`
	Host         string            `json:"host"`
	Version      string            `json:"version"`
	GitHead      string            `json:"git_head,omitempty"`
	IslandsDebug bool              `json:"islands_debug"`
	Total        int               `json:"total"`
	Passed       int               `json:"passed"`
	Failed       int               `json:"failed"`
	Cases        []smokeCaseReport `json:"cases"`
}

type smokeCase struct {
	name         string
	srcPath      string
	expectedExit int
	debugOnly    bool
}

type smokeListCase struct {
	Name         string `json:"name"`
	SrcPath      string `json:"src_path"`
	TargetGroup  string `json:"target_group"`
	ExpectedExit int    `json:"expected_exit"`
	DebugOnly    bool   `json:"debug_only,omitempty"`
}

type smokeExcludedExample struct {
	SrcPath string `json:"src_path"`
	Reason  string `json:"reason"`
}

type smokeListReport struct {
	Target           string                 `json:"target"`
	BuildOnly        bool                   `json:"build_only"`
	RunSupported     bool                   `json:"run_supported"`
	Total            int                    `json:"total"`
	IslandsDebug     bool                   `json:"islands_debug"`
	Cases            []smokeListCase        `json:"cases"`
	ExcludedExamples []smokeExcludedExample `json:"excluded_examples,omitempty"`
}

const supportedTargetsHelp = "linux-x64, windows-x64, macos-x64, wasm32-wasi (build-only), wasm32-web (build-only)"

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}

func runCLI(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}
	if isHelpArgs(args) {
		printUsage(stdout)
		return 0
	}
	switch args[0] {
	case "version":
		if len(args) > 1 {
			if isHelpArgs(args[1:]) {
				fmt.Fprintln(stdout, "usage: tetra version")
				return 0
			}
			fmt.Fprintln(stderr, "version does not accept arguments")
			return 2
		}
		fmt.Fprintln(stdout, compiler.Version())
		return 0
	case "targets":
		return runTargets(args[1:], stdout, stderr)
	case "formats":
		return runFormats(args[1:], stdout, stderr)
	case "new":
		return runNew(args[1:], stdout, stderr)
	case "project":
		return runProject(args[1:], stdout, stderr)
	case "workspace":
		return runWorkspace(args[1:], stdout, stderr)
	case "interface":
		return runInterface(args[1:], stdout, stderr)
	case "doctor":
		return runDoctor(args[1:], stdout, stderr)
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "build":
		return runBuild(args[1:], stdout, stderr)
	case "run":
		return runRun(args[1:], stdout, stderr)
	case "smoke":
		return runSmoke(args[1:], stdout, stderr)
	case "fmt":
		return runFmt(args[1:], stdout, stderr)
	case "test":
		return runTest(args[1:], stdout, stderr)
	case "doc":
		return runDoc(args[1:], stdout, stderr)
	case "clean":
		return runClean(args[1:], stdout, stderr)
	case "eco":
		return runEco(args[1:], stdout, stderr)
	case "lsp":
		return runLSP(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func isHelpArgs(args []string) bool {
	return len(args) == 1 && (args[0] == "-h" || args[0] == "--help")
}

type targetsReport struct {
	Supported []string            `json:"supported"`
	BuildOnly []string            `json:"build_only"`
	Planned   []string            `json:"planned"`
	Targets   []targetReportEntry `json:"targets"`
}

type targetReportEntry struct {
	Triple                  string `json:"triple"`
	Status                  string `json:"status"`
	OS                      string `json:"os"`
	Arch                    string `json:"arch"`
	ABI                     string `json:"abi"`
	Format                  string `json:"format"`
	ExeExt                  string `json:"exe_ext"`
	BuildOnly               bool   `json:"build_only"`
	RunSupported            bool   `json:"run_supported"`
	SupportsDebugInfo       bool   `json:"supports_debug_info"`
	SupportsReleaseOptimize bool   `json:"supports_release_optimize"`
}

type formatsReport struct {
	Formats []compiler.FormatInfo `json:"formats"`
}

type doctorReport struct {
	Status string        `json:"status"`
	Checks []doctorCheck `json:"checks"`
}

type doctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func runTargets(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("targets", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "targets does not accept positional arguments")
		return 2
	}
	report := targetsReport{
		Supported: ctarget.SupportedTriples(),
		BuildOnly: ctarget.BuildOnlyTriples(),
		Planned:   ctarget.PlannedTriples(),
		Targets:   buildTargetReportEntries(),
	}
	switch *format {
	case "text", "":
		fmt.Fprintln(stdout, "Supported targets:")
		for _, triple := range report.Supported {
			fmt.Fprintf(stdout, "  %s\n", describeTargetForText(triple))
		}
		fmt.Fprintln(stdout, "Build-only targets:")
		for _, triple := range report.BuildOnly {
			fmt.Fprintf(stdout, "  %s\n", describeTargetForText(triple))
		}
		fmt.Fprintln(stdout, "Planned targets:")
		for _, triple := range report.Planned {
			fmt.Fprintf(stdout, "  %s\n", triple)
		}
		return 0
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func runFormats(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("formats", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "formats does not accept positional arguments")
		return 2
	}
	report := formatsReport{Formats: compiler.T4Formats()}
	switch *format {
	case "text", "":
		fmt.Fprintln(stdout, "T4 formats:")
		for _, item := range report.Formats {
			suffix := item.Extension
			if suffix == "" {
				suffix = item.FileName
			}
			markers := []string{item.Role}
			if item.Primary {
				markers = append(markers, "primary")
			}
			if item.Legacy {
				markers = append(markers, "legacy")
			}
			fmt.Fprintf(stdout, "  %s - %s (%s)\n", suffix, item.Name, strings.Join(markers, ", "))
		}
		return 0
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func runNew(args []string, stdout io.Writer, stderr io.Writer) int {
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra new app [--lock] <NameOrPath>")
		return 0
	}
	if len(args) == 0 {
		fmt.Fprintln(stderr, "new requires a template")
		return 2
	}
	switch args[0] {
	case "app":
		return runNewAppArgs(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown new template %q\n", args[0])
		return 2
	}
}

type newAppOptions struct {
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

func runProject(args []string, stdout io.Writer, stderr io.Writer) int {
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra project <info|sync|deps> [options]")
		return 0
	}
	if len(args) == 0 {
		fmt.Fprintln(stderr, "project requires a subcommand")
		return 2
	}
	switch args[0] {
	case "info":
		return runProjectInfo(args[1:], stdout, stderr)
	case "sync":
		return runProjectSync(args[1:], stdout, stderr)
	case "deps":
		return runProjectDeps(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown project subcommand %q\n", args[0])
		return 2
	}
}

type projectDepsContext struct {
	Root        string
	CapsulePath string
	Manifest    capsuleManifest
}

type projectDepsReport struct {
	Status       string                    `json:"status,omitempty"`
	Root         string                    `json:"root,omitempty"`
	CapsulePath  string                    `json:"capsule_path,omitempty"`
	Dependencies []projectDependencyReport `json:"dependencies"`
}

type projectDependencyReport struct {
	ID           string `json:"id"`
	Version      string `json:"version"`
	Path         string `json:"path,omitempty"`
	ResolvedPath string `json:"resolved_path,omitempty"`
	Status       string `json:"status"`
	Detail       string `json:"detail,omitempty"`
}

func runProjectDeps(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: tetra project deps <list|add|remove|check> [options]")
		return 2
	}
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra project deps <list|add|remove|check> [options]")
		return 0
	}
	switch args[0] {
	case "list":
		return runProjectDepsList(args[1:], stdout, stderr)
	case "add":
		return runProjectDepsAdd(args[1:], stdout, stderr)
	case "remove":
		return runProjectDepsRemove(args[1:], stdout, stderr)
	case "check":
		return runProjectDepsCheck(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown project deps command %q\n", args[0])
		return 2
	}
}

func runProjectDepsList(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("project deps list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "project deps list accepts at most one path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	ctx, err := discoverProjectDepsContext(start)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	report := buildProjectDepsReport(ctx)
	switch *format {
	case "text", "":
		writeProjectDepsText(stdout, report)
		return 0
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func runProjectDepsAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("project deps add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	pathFlag := fs.String("path", "", "local dependency project path")
	idFlag := fs.String("id", "", "dependency id; defaults to dependency Capsule.t4 id")
	versionFlag := fs.String("version", "", "dependency version; defaults to dependency Capsule.t4 version")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "project deps add accepts at most one project path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	ctx, err := discoverProjectDepsContext(start)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	depRoot, depManifest, relPath, err := resolveProjectDependencyAddPath(ctx.Root, *pathFlag)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	_ = depRoot
	id := strings.TrimSpace(*idFlag)
	if id == "" {
		id = depManifest.ID
	}
	id, err = normalizeProjectDependencyID(id)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	version := strings.TrimSpace(*versionFlag)
	if version == "" {
		version = depManifest.Version
	}
	if !isCapsuleSemver(version) {
		fmt.Fprintln(stderr, "project deps add --version must use semver x.y.z")
		return 2
	}
	for _, dep := range ctx.Manifest.Dependencies {
		if dep.ID == id && dep.Version == version {
			fmt.Fprintf(stderr, "duplicate dependency %s %s\n", id, version)
			return 1
		}
	}
	dep := capsuleDependency{ID: id, Version: version, Path: relPath}
	if err := appendDependencyToCapsule(ctx.CapsulePath, dep); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Added dependency: %s %s %s\n", dep.ID, dep.Version, dep.Path)
	fmt.Fprintf(stdout, "run: %s\n", projectSyncRepairCommand(ctx.Root, "", false))
	return 0
}

func runProjectDepsRemove(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("project deps remove", flag.ContinueOnError)
	fs.SetOutput(stderr)
	idFlag := fs.String("id", "", "dependency id")
	versionFlag := fs.String("version", "", "dependency version")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "project deps remove accepts at most one project path")
		return 2
	}
	id, err := normalizeProjectDependencyID(*idFlag)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	version := strings.TrimSpace(*versionFlag)
	if version != "" && !isCapsuleSemver(version) {
		fmt.Fprintln(stderr, "project deps remove --version must use semver x.y.z")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	ctx, err := discoverProjectDepsContext(start)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	var matches []capsuleDependency
	for _, dep := range ctx.Manifest.Dependencies {
		if dep.ID == id && (version == "" || dep.Version == version) {
			matches = append(matches, dep)
		}
	}
	if len(matches) == 0 {
		fmt.Fprintf(stderr, "dependency not found: %s\n", id)
		return 1
	}
	if version == "" && len(matches) > 1 {
		fmt.Fprintf(stderr, "dependency %s has multiple versions; remove requires --version\n", id)
		return 2
	}
	removed, err := removeDependencyFromCapsule(ctx.CapsulePath, id, version)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if removed == 0 {
		fmt.Fprintf(stderr, "dependency not found: %s\n", id)
		return 1
	}
	if version == "" {
		version = matches[0].Version
	}
	fmt.Fprintf(stdout, "Removed dependency: %s %s\n", id, version)
	fmt.Fprintf(stdout, "run: %s\n", projectSyncRepairCommand(ctx.Root, "", false))
	return 0
}

func runProjectDepsCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("project deps check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "project deps check accepts at most one path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	ctx, err := discoverProjectDepsContext(start)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	report := buildProjectDepsReport(ctx)
	status := "pass"
	var issues []projectDependencyReport
	for _, dep := range report.Dependencies {
		if dep.Status != "ok" {
			status = "fail"
			issues = append(issues, dep)
		}
	}
	if status == "pass" {
		depRoots, depManifests, err := projectDependencyGraph(ctx.Root, ctx.Manifest, map[string]int{ctx.Root: projectDependencyVisiting}, []string{ctx.Root})
		if err != nil {
			status = "fail"
			issues = append(issues, projectDependencyReport{Status: "fail", Detail: err.Error()})
		} else {
			_ = depRoots
			manifests := append([]capsuleManifest{ctx.Manifest}, depManifests...)
			if err := validateCapsuleGraph(manifests, ""); err != nil {
				status = "fail"
				issues = append(issues, projectDependencyReport{Status: "fail", Detail: err.Error()})
			}
		}
	}
	report.Status = status
	switch *format {
	case "text", "":
		if status == "pass" {
			fmt.Fprintf(stdout, "Dependencies OK: %d\n", len(report.Dependencies))
			return 0
		}
		writeProjectDependencyIssues(stderr, issues)
		return 1
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if status != "pass" {
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

type projectInfoReport struct {
	Found              bool           `json:"found"`
	Root               string         `json:"root,omitempty"`
	CapsulePath        string         `json:"capsule_path,omitempty"`
	LockPath           string         `json:"lock_path,omitempty"`
	EntryPath          string         `json:"entry_path,omitempty"`
	SourceRoots        []string       `json:"source_roots,omitempty"`
	Targets            []string       `json:"targets,omitempty"`
	DependencyRoots    []string       `json:"dependency_roots,omitempty"`
	ArtifactCounts     map[string]int `json:"artifact_counts,omitempty"`
	DependencyCapsules []string       `json:"dependency_capsules,omitempty"`
}

func runProjectInfo(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("project info", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "project info accepts at most one path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	report, err := buildProjectInfoReport(start)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	switch *format {
	case "text", "":
		if !report.Found {
			fmt.Fprintln(stdout, "Project: not found")
			return 1
		}
		fmt.Fprintf(stdout, "Project root: %s\n", report.Root)
		fmt.Fprintf(stdout, "Capsule: %s\n", report.CapsulePath)
		if report.LockPath != "" {
			fmt.Fprintf(stdout, "Lock: %s\n", report.LockPath)
		}
		fmt.Fprintf(stdout, "Entry: %s\n", report.EntryPath)
		fmt.Fprintf(stdout, "Source roots: %s\n", strings.Join(report.SourceRoots, ", "))
		fmt.Fprintf(stdout, "Targets: %s\n", strings.Join(report.Targets, ", "))
		return 0
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if !report.Found {
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func runProjectSync(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("project sync", flag.ContinueOnError)
	fs.SetOutput(stderr)
	targetFlag := fs.String("target", "", "native target triple for generated .tobj artifacts")
	checkOnly := fs.Bool("check", false, "dry-run and report pending project lock/artifact changes without writing files")
	allTargets := fs.Bool("all-targets", false, "sync artifacts for every native target listed in Capsule.t4")
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *targetFlag != "" && *allTargets {
		fmt.Fprintln(stderr, "project sync accepts either --target or --all-targets, not both")
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "project sync accepts at most one path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	ctx, err := discoverCLIProject(start)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if ctx == nil || !ctx.Found {
		fmt.Fprintln(stderr, "project capsule not found")
		return 1
	}
	lockPath := filepath.Join(ctx.Root, compiler.SemanticLockFileName)
	if *checkOnly {
		issues, err := checkProjectSync(ctx, *targetFlag, lockPath, *allTargets)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		for i := range issues {
			issues[i].Repair = projectSyncRepairCommand(ctx.Root, *targetFlag, *allTargets)
		}
		if len(issues) > 0 {
			writeArtifactIssues(stdout, issues, true)
			return 1
		}
		fmt.Fprintf(stdout, "Project current: %s\n", ctx.Root)
		return 0
	}
	useArtifactBuilder, err := projectSyncUsesArtifactBuilder(ctx.Manifest, *targetFlag, *allTargets)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if useArtifactBuilder {
		if err := buildCapsuleArtifacts(ctx.CapsulePath, capsuleArtifactBuildOptions{
			Target:     *targetFlag,
			LockPath:   lockPath,
			Jobs:       *jobs,
			AllTargets: *allTargets,
		}); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	} else {
		manifests, err := parseCapsuleGraphArgs([]string{ctx.CapsulePath})
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if err := validateCapsuleGraph(manifests, *targetFlag); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if err := writeEcoLock(lockPath, manifests); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	fmt.Fprintf(stdout, "Project synced: %s\n", ctx.Root)
	return 0
}

func checkProjectSync(ctx *cliProjectContext, targetFlag string, lockPath string, allTargets bool) ([]artifactIssue, error) {
	useArtifactBuilder, err := projectSyncUsesArtifactBuilder(ctx.Manifest, targetFlag, allTargets)
	if err != nil {
		return nil, err
	}
	if useArtifactBuilder {
		return checkCapsuleArtifacts(ctx.CapsulePath, targetFlag, lockPath, allTargets)
	}
	manifests, err := parseCapsuleGraphArgs([]string{ctx.CapsulePath})
	if err != nil {
		return nil, err
	}
	if err := validateCapsuleGraph(manifests, targetFlag); err != nil {
		return nil, err
	}
	return checkProjectLockOnly(lockPath, manifests)
}

func projectSyncUsesArtifactBuilder(manifest capsuleManifest, targetFlag string, allTargets bool) (bool, error) {
	targets, err := projectSyncNativeArtifactTargets(manifest, targetFlag, allTargets)
	if err != nil {
		return false, err
	}
	return len(targets) > 0, nil
}

func projectSyncNativeArtifactTargets(manifest capsuleManifest, targetFlag string, allTargets bool) ([]string, error) {
	if targetFlag != "" {
		target, err := normalizeCapsuleTarget(targetFlag)
		if err != nil {
			return nil, err
		}
		if ctarget.IsBuildOnlyTarget(target) {
			return nil, nil
		}
		return []string{target}, nil
	}
	if allTargets {
		seen := map[string]struct{}{}
		var targets []string
		for _, raw := range manifest.Targets {
			target, err := normalizeCapsuleTarget(raw)
			if err != nil {
				return nil, err
			}
			if ctarget.IsBuildOnlyTarget(target) {
				continue
			}
			if _, ok := seen[target]; ok {
				continue
			}
			seen[target] = struct{}{}
			targets = append(targets, target)
		}
		sort.Strings(targets)
		return targets, nil
	}
	if len(manifest.Targets) > 0 {
		target, err := normalizeCapsuleTarget(manifest.Targets[0])
		if err != nil {
			return nil, err
		}
		if ctarget.IsBuildOnlyTarget(target) {
			return nil, nil
		}
		return []string{target}, nil
	}
	host, ok := hostTarget()
	if !ok {
		return nil, nil
	}
	return []string{host}, nil
}

func checkProjectLockOnly(lockPath string, manifests []capsuleManifest) ([]artifactIssue, error) {
	if _, err := os.Stat(lockPath); err != nil {
		if os.IsNotExist(err) {
			return []artifactIssue{{
				Kind:   "missing lock",
				Path:   filepath.ToSlash(lockPath),
				Detail: "Tetra.lock is required for locked project builds",
			}}, nil
		}
		return nil, err
	}
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, err
	}
	lock, err := decodeEcoLock(raw)
	if err != nil {
		return []artifactIssue{{Kind: "invalid lock", Path: filepath.ToSlash(lockPath), Detail: err.Error()}}, nil
	}
	current, err := buildEcoLockWithArtifactHashes(manifests)
	if err != nil {
		return nil, err
	}
	if lock.GraphSHA256 != current.GraphSHA256 {
		return []artifactIssue{{
			Kind:   "stale lock",
			Path:   filepath.ToSlash(lockPath),
			Detail: fmt.Sprintf("expected graph %s, lock has %s", current.GraphSHA256, lock.GraphSHA256),
		}}, nil
	}
	return nil, nil
}

func discoverProjectDepsContext(start string) (projectDepsContext, error) {
	startDir, err := cliProjectStartDir(start)
	if err != nil {
		return projectDepsContext{}, err
	}
	capsulePath, root, ok, err := findProjectCapsule(startDir)
	if err != nil {
		return projectDepsContext{}, err
	}
	if !ok {
		return projectDepsContext{}, fmt.Errorf("project capsule not found")
	}
	manifest, err := parseCapsule(capsulePath)
	if err != nil {
		return projectDepsContext{}, err
	}
	return projectDepsContext{Root: root, CapsulePath: capsulePath, Manifest: manifest}, nil
}

func buildProjectDepsReport(ctx projectDepsContext) projectDepsReport {
	report := projectDepsReport{
		Root:         ctx.Root,
		CapsulePath:  ctx.CapsulePath,
		Dependencies: []projectDependencyReport{},
	}
	for _, dep := range ctx.Manifest.Dependencies {
		report.Dependencies = append(report.Dependencies, describeProjectDependency(ctx.Root, dep))
	}
	return report
}

func describeProjectDependency(root string, dep capsuleDependency) projectDependencyReport {
	item := projectDependencyReport{
		ID:      dep.ID,
		Version: dep.Version,
		Path:    dep.Path,
		Status:  "ok",
	}
	if dep.Path == "" {
		item.Status = "missing"
		item.Detail = "dependency has no local path"
		return item
	}
	depRoot, err := resolveDependencyProjectRoot(root, dep.Path)
	if err != nil {
		item.Status = "missing"
		item.Detail = err.Error()
		return item
	}
	item.ResolvedPath = depRoot
	capsulePath, err := findCapsulePath(depRoot)
	if err != nil {
		item.Status = "missing"
		item.Detail = err.Error()
		return item
	}
	manifest, err := parseCapsule(capsulePath)
	if err != nil {
		item.Status = "invalid"
		item.Detail = err.Error()
		return item
	}
	if manifest.ID != dep.ID {
		item.Status = "mismatch"
		item.Detail = fmt.Sprintf("id mismatch: want %s, got %s", dep.ID, manifest.ID)
		return item
	}
	if manifest.Version != dep.Version {
		item.Status = "mismatch"
		item.Detail = fmt.Sprintf("version mismatch: want %s, got %s", dep.Version, manifest.Version)
		return item
	}
	return item
}

func writeProjectDepsText(w io.Writer, report projectDepsReport) {
	if len(report.Dependencies) == 0 {
		fmt.Fprintln(w, "Dependencies: none")
		return
	}
	fmt.Fprintln(w, "Dependencies:")
	for _, dep := range report.Dependencies {
		path := dep.Path
		if path == "" {
			path = "-"
		}
		fmt.Fprintf(w, "  %s %s %s %s\n", dep.ID, dep.Version, path, dep.Status)
		if dep.Detail != "" {
			fmt.Fprintf(w, "    detail: %s\n", dep.Detail)
		}
	}
}

func writeProjectDependencyIssues(w io.Writer, issues []projectDependencyReport) {
	for _, issue := range issues {
		if issue.ID == "" {
			fmt.Fprintln(w, issue.Detail)
			continue
		}
		path := issue.Path
		if path == "" {
			path = "-"
		}
		if issue.Detail != "" {
			fmt.Fprintf(w, "%s %s %s: %s\n", issue.ID, issue.Version, path, issue.Detail)
		} else {
			fmt.Fprintf(w, "%s %s %s: %s\n", issue.ID, issue.Version, path, issue.Status)
		}
	}
}

func resolveProjectDependencyAddPath(root string, depPath string) (string, capsuleManifest, string, error) {
	depPath = strings.TrimSpace(depPath)
	if depPath == "" {
		return "", capsuleManifest{}, "", fmt.Errorf("project deps add requires --path")
	}
	path := filepath.FromSlash(depPath)
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", capsuleManifest{}, "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", capsuleManifest{}, "", err
	}
	if !info.IsDir() {
		path = filepath.Dir(path)
	}
	capsulePath, err := findCapsulePath(path)
	if err != nil {
		return "", capsuleManifest{}, "", err
	}
	depRoot := filepath.Dir(capsulePath)
	if filepath.Clean(depRoot) == filepath.Clean(root) {
		return "", capsuleManifest{}, "", fmt.Errorf("project cannot depend on itself")
	}
	manifest, err := parseCapsule(capsulePath)
	if err != nil {
		return "", capsuleManifest{}, "", err
	}
	rel, err := filepath.Rel(root, depRoot)
	if err != nil {
		return "", capsuleManifest{}, "", err
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "." || rel == "" {
		return "", capsuleManifest{}, "", fmt.Errorf("project cannot depend on itself")
	}
	if strings.ContainsAny(rel, " \t\r\n") {
		return "", capsuleManifest{}, "", fmt.Errorf("dependency path %q contains whitespace, which Capsule.t4 deps do not support yet", rel)
	}
	return depRoot, manifest, rel, nil
}

func normalizeProjectDependencyID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("dependency id is required")
	}
	if strings.ContainsAny(id, " \t\r\n") {
		return "", fmt.Errorf("dependency id must not contain whitespace")
	}
	if !strings.HasPrefix(id, "tetra://") {
		id = "tetra://" + id
	}
	return id, nil
}

func appendDependencyToCapsule(path string, dep capsuleDependency) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines, finalNewline := splitCapsuleText(raw)
	line := formatCapsuleDependencyLine(dep)
	if _, end, indent, ok := findCapsuleDepsSection(lines); ok {
		depIndent := indent + "    "
		lines = append(lines[:end], append([]string{depIndent + line}, lines[end:]...)...)
		return writeCapsuleText(path, lines, finalNewline)
	}
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
		lines = append(lines, "")
	}
	lines = append(lines, "    deps:", "        "+line)
	return writeCapsuleText(path, lines, true)
}

func removeDependencyFromCapsule(path string, id string, version string) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	lines, finalNewline := splitCapsuleText(raw)
	section := ""
	var out []string
	removed := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			out = append(out, line)
			continue
		}
		if nextSection, ok := capsuleSectionHeader(trimmed); ok {
			section = nextSection
			out = append(out, line)
			continue
		}
		dep, depLine, err := parseDependencyLineForEdit(path, i+1, section, trimmed)
		if err != nil {
			return 0, err
		}
		if depLine && dep.ID == id && (version == "" || dep.Version == version) {
			removed++
			continue
		}
		out = append(out, line)
	}
	if removed == 0 {
		return 0, nil
	}
	return removed, writeCapsuleText(path, out, finalNewline)
}

func parseDependencyLineForEdit(path string, line int, section string, trimmed string) (capsuleDependency, bool, error) {
	if strings.HasPrefix(trimmed, "dependency ") {
		dep, err := parseCapsuleDependency(path, line, strings.TrimSpace(strings.TrimPrefix(trimmed, "dependency ")))
		return dep, true, err
	}
	if section == "deps" {
		dep, err := parseCapsuleDependencyFields(path, line, strings.Fields(trimmed))
		return dep, true, err
	}
	return capsuleDependency{}, false, nil
}

func splitCapsuleText(raw []byte) ([]string, bool) {
	text := string(raw)
	finalNewline := strings.HasSuffix(text, "\n")
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return nil, finalNewline
	}
	return strings.Split(text, "\n"), finalNewline
}

func writeCapsuleText(path string, lines []string, finalNewline bool) error {
	text := strings.Join(lines, "\n")
	if finalNewline {
		text += "\n"
	}
	return os.WriteFile(path, []byte(text), 0o644)
}

func findCapsuleDepsSection(lines []string) (int, int, string, bool) {
	section := ""
	start := -1
	indent := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if nextSection, ok := capsuleSectionHeader(trimmed); ok {
			if section == "deps" {
				return start, i, indent, true
			}
			section = nextSection
			if nextSection == "deps" {
				start = i
				indent = leadingWhitespace(line)
			}
			continue
		}
	}
	if section == "deps" {
		return start, len(lines), indent, true
	}
	return -1, -1, "", false
}

func leadingWhitespace(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return s[:i]
}

func formatCapsuleDependencyLine(dep capsuleDependency) string {
	if dep.Path != "" {
		return dep.ID + " " + dep.Version + " " + dep.Path
	}
	return dep.ID + " " + dep.Version
}

func buildProjectInfoReport(start string) (projectInfoReport, error) {
	ctx, err := discoverCLIProject(start)
	if err != nil {
		return projectInfoReport{}, err
	}
	if ctx == nil || !ctx.Found {
		return projectInfoReport{Found: false}, nil
	}
	report := projectInfoReport{
		Found:              true,
		Root:               ctx.Root,
		CapsulePath:        ctx.CapsulePath,
		LockPath:           ctx.LockPath,
		EntryPath:          ctx.EntryPath,
		SourceRoots:        append([]string(nil), ctx.SourceRoots...),
		Targets:            append([]string(nil), ctx.Manifest.Targets...),
		ArtifactCounts:     map[string]int{},
		DependencyCapsules: []string{},
	}
	for _, root := range ctx.DependencyRoots {
		report.DependencyRoots = append(report.DependencyRoots, root.Root)
	}
	for _, manifest := range ctx.Manifests[1:] {
		report.DependencyCapsules = append(report.DependencyCapsules, manifest.Path)
	}
	for _, artifact := range ctx.Manifest.Artifacts {
		report.ArtifactCounts[artifact.Kind]++
	}
	sort.Strings(report.DependencyRoots)
	sort.Strings(report.DependencyCapsules)
	return report, nil
}

func runInterface(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("interface", flag.ContinueOnError)
	fs.SetOutput(stderr)
	outPath := fs.String("o", "", "output .t4i path; stdout when empty")
	checkMode := fs.Bool("check", false, "check that the .t4i public API hash matches the source")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	if fs.NArg() != 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "interface requires exactly one input path")
		return 2
	}
	inputPath := fs.Arg(0)
	if *checkMode {
		path := *outPath
		if path == "" {
			path = compiler.InterfaceOutputPath(inputPath)
		}
		src, err := os.ReadFile(inputPath)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		iface, err := os.ReadFile(path)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		if err := compiler.ValidateInterfaceAgainstSource(src, iface, inputPath); err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		fmt.Fprintf(stdout, "Interface current: %s\n", path)
		return 0
	}
	docs, err := compiler.GenerateInterfaceFile(inputPath)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if *outPath == "" {
		fmt.Fprint(stdout, string(docs))
		return 0
	}
	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if err := os.WriteFile(*outPath, docs, 0o644); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	fmt.Fprintf(stdout, "Wrote interface: %s\n", *outPath)
	return 0
}

func buildTargetReportEntries() []targetReportEntry {
	host, hostOK := hostTarget()
	triples := append([]string{}, ctarget.SupportedTriples()...)
	triples = append(triples, ctarget.BuildOnlyTriples()...)
	triples = append(triples, ctarget.PlannedTriples()...)
	out := make([]targetReportEntry, 0, len(triples))
	for _, triple := range triples {
		tgt, err := ctarget.Parse(triple)
		if err != nil {
			continue
		}
		buildOnly := ctarget.IsBuildOnlyTarget(tgt.Triple)
		out = append(out, targetReportEntry{
			Triple:                  tgt.Triple,
			Status:                  tgt.Status.String(),
			OS:                      tgt.OS.String(),
			Arch:                    tgt.Arch.String(),
			ABI:                     tgt.ABI.String(),
			Format:                  tgt.Format.String(),
			ExeExt:                  tgt.ExeExt,
			BuildOnly:               buildOnly,
			RunSupported:            hostOK && host == tgt.Triple && !buildOnly,
			SupportsDebugInfo:       tgt.SupportsDebugInfo,
			SupportsReleaseOptimize: tgt.SupportsReleaseOptimize,
		})
	}
	return out
}

func describeTargetForText(triple string) string {
	tgt, err := ctarget.Parse(triple)
	if err != nil {
		return triple
	}
	parts := []string{
		triple,
		"os=" + tgt.OS.String(),
		"arch=" + tgt.Arch.String(),
		"abi=" + tgt.ABI.String(),
		"format=" + tgt.Format.String(),
	}
	if tgt.ExeExt != "" {
		parts = append(parts, "exe_ext="+tgt.ExeExt)
	}
	if ctarget.IsBuildOnlyTarget(triple) {
		parts = append(parts, "build-only")
	}
	return strings.Join(parts, " ")
}

func runDoctor(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "doctor accepts at most one path")
		return 2
	}
	report := doctorReport{}
	if fs.NArg() == 1 {
		report = buildProjectDoctorReport(fs.Arg(0))
	} else if ctx, err := discoverCLIProject("."); err == nil && ctx != nil && ctx.Found {
		report = buildProjectDoctorReport(ctx.Root)
	} else {
		report = buildDoctorReport()
	}
	switch *format {
	case "text", "":
		fmt.Fprintf(stdout, "Tetra doctor: %s\n", report.Status)
		for _, check := range report.Checks {
			if check.Detail == "" {
				fmt.Fprintf(stdout, "  %s: %s\n", check.Name, check.Status)
			} else {
				fmt.Fprintf(stdout, "  %s: %s (%s)\n", check.Name, check.Status, check.Detail)
			}
		}
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
	if report.Status != "pass" {
		return 1
	}
	return 0
}

func buildProjectDoctorReport(start string) doctorReport {
	ctx, err := discoverCLIProject(start)
	if err != nil {
		return doctorReport{Status: "fail", Checks: []doctorCheck{failCheck("project capsule", err.Error())}}
	}
	if ctx == nil || !ctx.Found {
		return doctorReport{Status: "fail", Checks: []doctorCheck{failCheck("project capsule", "not found")}}
	}
	checks := []doctorCheck{
		passCheck("version", compiler.Version()),
		passCheck("project root", ctx.Root),
		passCheck("project capsule", ctx.CapsulePath),
		passCheck("project entry", ctx.EntryPath),
	}
	sourcePaths := existingProjectSourcePaths(ctx)
	if len(sourcePaths) == 0 {
		checks = append(checks, failCheck("project source roots", "no existing source roots"))
	} else {
		var rels []string
		for _, path := range sourcePaths {
			rel, err := filepath.Rel(ctx.Root, path)
			if err != nil {
				rels = append(rels, filepath.ToSlash(path))
				continue
			}
			rels = append(rels, filepath.ToSlash(rel))
		}
		checks = append(checks, passCheck("project source roots", strings.Join(rels, ", ")))
	}
	if len(ctx.DependencyRoots) == 0 {
		checks = append(checks, passCheck("project dependencies", "none"))
	} else {
		checks = append(checks, passCheck("project dependencies", fmt.Sprintf("%d root(s)", len(ctx.DependencyRoots))))
	}
	if ctx.LockPath == "" {
		checks = append(checks, passCheck("project lock", "not present; run "+projectSyncRepairCommand(ctx.Root, "", false)))
	} else if err := validateDiscoveredProjectLock(ctx, ""); err != nil {
		checks = append(checks, failCheck("project lock", err.Error()))
	} else {
		checks = append(checks, passCheck("project lock", ctx.LockPath))
	}
	return doctorReport{Status: doctorStatus(checks), Checks: checks}
}

func buildDoctorReport() doctorReport {
	checks := []doctorCheck{
		passCheck("version", compiler.Version()),
		passCheck("supported targets", strings.Join(ctarget.SupportedTriples(), ", ")),
		passCheck("build-only targets", strings.Join(ctarget.BuildOnlyTriples(), ", ")),
		passCheck("planned targets", strings.Join(ctarget.PlannedTriples(), ", ")),
	}
	root, err := findRepoRoot()
	if err != nil {
		checks = append(checks, failCheck("repo root", err.Error()))
		return doctorReport{Status: doctorStatus(checks), Checks: checks}
	}
	return buildDoctorReportForRootWithChecks(root, checks)
}

func buildDoctorReportForRoot(root string) doctorReport {
	checks := []doctorCheck{
		passCheck("version", compiler.Version()),
		passCheck("supported targets", strings.Join(ctarget.SupportedTriples(), ", ")),
		passCheck("build-only targets", strings.Join(ctarget.BuildOnlyTriples(), ", ")),
		passCheck("planned targets", strings.Join(ctarget.PlannedTriples(), ", ")),
	}
	return buildDoctorReportForRootWithChecks(root, checks)
}

func buildDoctorReportForRootWithChecks(root string, checks []doctorCheck) doctorReport {
	checks = append(checks,
		passCheck("repo root", root),
		pathCheck(root, "__rt/actors_sysv.tetra"),
		pathCheck(root, "__rt/actors_win64.tetra"),
		pathCheck(root, "compiler/selfhostrt/actors_sysv.tetra"),
		pathCheck(root, "compiler/selfhostrt/actors_win64.tetra"),
		pathCheck(root, "examples/flow_hello.tetra"),
		pathCheck(root, "docs/generated/manifest.json"),
		manifestVersionCheck(root),
		manifestSurfaceCheck(root),
		smokeSourcesCheck(root),
		runtimeExportsCheck(root),
		targetMetadataCheck(),
		toolingCommandsCheck(),
	)
	return doctorReport{Status: doctorStatus(checks), Checks: checks}
}

func manifestVersionCheck(root string) doctorCheck {
	path := filepath.Join(root, filepath.FromSlash("docs/generated/manifest.json"))
	raw, err := os.ReadFile(path)
	if err != nil {
		return failCheck("docs manifest version", err.Error())
	}
	var manifest struct {
		CompilerVersion string `json:"compiler_version"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return failCheck("docs manifest version", err.Error())
	}
	if manifest.CompilerVersion != compiler.Version() {
		return failCheck("docs manifest version", fmt.Sprintf("got %s want %s", manifest.CompilerVersion, compiler.Version()))
	}
	return passCheck("docs manifest version", manifest.CompilerVersion)
}

func manifestSurfaceCheck(root string) doctorCheck {
	path := filepath.Join(root, filepath.FromSlash("docs/generated/manifest.json"))
	raw, err := os.ReadFile(path)
	if err != nil {
		return failCheck("docs manifest surface", err.Error())
	}
	var manifest struct {
		Formats []struct {
			Extension string `json:"extension,omitempty"`
			FileName  string `json:"file_name,omitempty"`
			Role      string `json:"role"`
			Primary   bool   `json:"primary,omitempty"`
			Legacy    bool   `json:"legacy,omitempty"`
		} `json:"formats"`
		Targets []struct {
			Triple string `json:"triple"`
		} `json:"targets"`
		RuntimeABI struct {
			ActorsSupportedTargets []string `json:"actors_supported_targets"`
			ActorsRequiredSymbols  []string `json:"actors_required_symbols"`
			TimeRequiredSymbols    []string `json:"time_required_symbols"`
		} `json:"runtime_abi"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return failCheck("docs manifest surface", err.Error())
	}
	formatKeys := map[string]bool{}
	var sourcePrimary, sourceLegacy bool
	for _, format := range manifest.Formats {
		key := format.Extension
		if key == "" {
			key = format.FileName
		}
		formatKeys[key] = true
		if key == compiler.T4SourceExtension && format.Role == "source" && format.Primary {
			sourcePrimary = true
		}
		if key == compiler.LegacyTetraSourceExtension && format.Role == "source" && format.Legacy {
			sourceLegacy = true
		}
	}
	requiredFormats := []string{
		compiler.T4SourceExtension,
		compiler.TodexFragmentExtension,
		compiler.T4SeedExtension,
		compiler.T4InterfaceExtension,
		compiler.T4ProofExtension,
		compiler.T4ReplayExtension,
		compiler.T4QuestExtension,
		compiler.NeedMapExtension,
		compiler.SemanticLockFileName,
	}
	for _, key := range requiredFormats {
		if !formatKeys[key] {
			return failCheck("docs manifest surface", "missing format "+key)
		}
	}
	if !sourcePrimary || !sourceLegacy {
		return failCheck("docs manifest surface", "missing source format primary/legacy markers")
	}
	var targetTriples []string
	for _, target := range manifest.Targets {
		targetTriples = append(targetTriples, target.Triple)
	}
	if !sameStringSet(targetTriples, ctarget.SupportedTriples()) {
		return failCheck("docs manifest surface", fmt.Sprintf("targets got %s want %s", strings.Join(sortedDoctorStrings(targetTriples), ", "), strings.Join(sortedDoctorStrings(ctarget.SupportedTriples()), ", ")))
	}
	if !sameStringSet(manifest.RuntimeABI.ActorsSupportedTargets, ctarget.SupportedTriples()) {
		return failCheck("docs manifest surface", fmt.Sprintf("actors targets got %s want %s", strings.Join(sortedDoctorStrings(manifest.RuntimeABI.ActorsSupportedTargets), ", "), strings.Join(sortedDoctorStrings(ctarget.SupportedTriples()), ", ")))
	}
	if !sameStringSet(manifest.RuntimeABI.ActorsRequiredSymbols, actorRuntimeSymbols()) {
		return failCheck("docs manifest surface", fmt.Sprintf("runtime symbols got %s want %s", strings.Join(sortedDoctorStrings(manifest.RuntimeABI.ActorsRequiredSymbols), ", "), strings.Join(sortedDoctorStrings(actorRuntimeSymbols()), ", ")))
	}
	if !sameStringSet(manifest.RuntimeABI.TimeRequiredSymbols, timeRuntimeSymbols()) {
		return failCheck("docs manifest surface", fmt.Sprintf("time runtime symbols got %s want %s", strings.Join(sortedDoctorStrings(manifest.RuntimeABI.TimeRequiredSymbols), ", "), strings.Join(sortedDoctorStrings(timeRuntimeSymbols()), ", ")))
	}
	return passCheck("docs manifest surface", fmt.Sprintf("%d formats, %d targets, %d runtime symbols", len(manifest.Formats), len(targetTriples), len(actorRuntimeSymbols())+len(timeRuntimeSymbols())))
}

func smokeSourcesCheck(root string) doctorCheck {
	seenNames := map[string]bool{}
	seenSources := map[string]bool{}
	var missing []string
	var duplicates []string
	cases := smokeCases(true)
	for _, c := range cases {
		if seenNames[c.name] {
			duplicates = append(duplicates, "name:"+c.name)
		}
		seenNames[c.name] = true
		if seenSources[c.srcPath] {
			duplicates = append(duplicates, "src:"+c.srcPath)
		}
		seenSources[c.srcPath] = true
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(c.srcPath))); err != nil {
			missing = append(missing, c.srcPath)
		}
	}
	if len(missing) > 0 || len(duplicates) > 0 {
		sort.Strings(missing)
		sort.Strings(duplicates)
		parts := []string{}
		if len(missing) > 0 {
			parts = append(parts, "missing "+strings.Join(missing, ", "))
		}
		if len(duplicates) > 0 {
			parts = append(parts, "duplicate "+strings.Join(duplicates, ", "))
		}
		return failCheck("smoke sources", strings.Join(parts, "; "))
	}
	return passCheck("smoke sources", fmt.Sprintf("%d sources", len(cases)))
}

func runtimeExportsCheck(root string) doctorCheck {
	paths := []string{
		"__rt/actors_sysv.tetra",
		"__rt/actors_win64.tetra",
		"compiler/selfhostrt/actors_sysv.tetra",
		"compiler/selfhostrt/actors_win64.tetra",
	}
	required := append(actorRuntimeSymbols(), timeRuntimeSymbols()...)
	var missing []string
	for _, rel := range paths {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			missing = append(missing, rel+": "+err.Error())
			continue
		}
		text := string(raw)
		for _, symbol := range required {
			if !strings.Contains(text, "@export("+strconv.Quote(symbol)+")") {
				missing = append(missing, rel+":"+symbol)
			}
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return failCheck("runtime exports", strings.Join(missing, ", "))
	}
	return passCheck("runtime exports", fmt.Sprintf("%d files, %d symbols", len(paths), len(required)))
}

func targetMetadataCheck() doctorCheck {
	entries := buildTargetReportEntries()
	seen := map[string]bool{}
	buildOnlyCount := 0
	for _, entry := range entries {
		if seen[entry.Triple] {
			return failCheck("target metadata", "duplicate target "+entry.Triple)
		}
		seen[entry.Triple] = true
		tgt, err := ctarget.Parse(entry.Triple)
		if err != nil {
			return failCheck("target metadata", err.Error())
		}
		if entry.OS != tgt.OS.String() || entry.Arch != tgt.Arch.String() || entry.ABI != tgt.ABI.String() || entry.Format != tgt.Format.String() {
			return failCheck("target metadata", fmt.Sprintf("%s metadata mismatch", entry.Triple))
		}
		if entry.ExeExt != tgt.ExeExt {
			return failCheck("target metadata", fmt.Sprintf("%s exe_ext got %q want %q", entry.Triple, entry.ExeExt, tgt.ExeExt))
		}
		buildOnly := ctarget.IsBuildOnlyTarget(entry.Triple)
		if entry.BuildOnly != buildOnly {
			return failCheck("target metadata", fmt.Sprintf("%s build_only got %v want %v", entry.Triple, entry.BuildOnly, buildOnly))
		}
		if buildOnly {
			buildOnlyCount++
			if entry.RunSupported {
				return failCheck("target metadata", entry.Triple+" must not be run-supported")
			}
		}
	}
	wantCount := len(ctarget.SupportedTriples()) + len(ctarget.BuildOnlyTriples()) + len(ctarget.PlannedTriples())
	if len(entries) != wantCount {
		return failCheck("target metadata", fmt.Sprintf("got %d targets want %d", len(entries), wantCount))
	}
	return passCheck("target metadata", fmt.Sprintf("%d targets, %d build-only", len(entries), buildOnlyCount))
}

func toolingCommandsCheck() doctorCheck {
	commands := []string{"check", "build", "run", "fmt", "test", "doc", "interface", "smoke", "targets", "formats", "doctor", "project", "new", "lsp", "eco", "clean", "version"}
	if len(commands) == 0 {
		return failCheck("tooling commands", "no commands registered")
	}
	return passCheck("tooling commands", strings.Join(commands, ", "))
}

func actorRuntimeSymbols() []string {
	return []string{
		"__tetra_entry",
		"__tetra_actor_spawn",
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
		"__tetra_actor_send_slot",
		"__tetra_actor_send_commit",
		"__tetra_actor_recv",
		"__tetra_actor_recv_msg",
		"__tetra_actor_recv_poll",
		"__tetra_actor_recv_until",
		"__tetra_actor_recv_msg_until",
		"__tetra_actor_recv_begin",
		"__tetra_actor_recv_slot",
		"__tetra_actor_recv_count",
		"__tetra_actor_self",
		"__tetra_actor_sender",
		"__tetra_actor_yield_now",
	}
}

func timeRuntimeSymbols() []string {
	return []string{
		"__tetra_time_now_ms",
		"__tetra_sleep_ms",
		"__tetra_sleep_until_ms",
		"__tetra_deadline_ms",
		"__tetra_timer_ready_ms",
	}
}

func sameStringSet(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[string]int{}
	for _, s := range a {
		seen[s]++
	}
	for _, s := range b {
		seen[s]--
		if seen[s] < 0 {
			return false
		}
	}
	for _, n := range seen {
		if n != 0 {
			return false
		}
	}
	return true
}

func sortedDoctorStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func pathCheck(root string, rel string) doctorCheck {
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
		return failCheck(rel, err.Error())
	}
	return passCheck(rel, "found")
}

func passCheck(name string, detail string) doctorCheck {
	return doctorCheck{Name: name, Status: "pass", Detail: detail}
}

func failCheck(name string, detail string) doctorCheck {
	return doctorCheck{Name: name, Status: "fail", Detail: detail}
}

func doctorStatus(checks []doctorCheck) string {
	for _, check := range checks {
		if check.Status != "pass" {
			return "fail"
		}
	}
	return "pass"
}

func runDoc(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("doc", flag.ContinueOnError)
	fs.SetOutput(stderr)
	outPath := fs.String("o", "", "output markdown path; stdout when empty")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	paths := fs.Args()
	if len(paths) == 0 {
		ctx, err := discoverCLIProject(".")
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		if ctx != nil && ctx.Found {
			paths = existingProjectSourcePaths(ctx)
		}
	}
	docs, err := compiler.GenerateAPIDocs(paths)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if *outPath == "" {
		fmt.Fprint(stdout, string(docs))
		return 0
	}
	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if err := os.WriteFile(*outPath, docs, 0o644); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	fmt.Fprintf(stdout, "Wrote docs: %s\n", *outPath)
	return 0
}

func runCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	interfaceOnly := fs.Bool("interface-only", false, "check interface/API surface without requiring executable output")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	input := ""
	if fs.NArg() > 0 {
		input = fs.Arg(0)
	}
	if fs.NArg() > 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "check accepts at most one input path")
		return 2
	}
	input, worldOpt, projectCtx, err := resolveCLIInput(input)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if err := validateDiscoveredProjectLock(projectCtx, ""); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	world, err := compiler.LoadWorldOpt(input, worldOpt)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	checkOpt := compiler.CheckOptions{RequireMain: true}
	if *interfaceOnly {
		checkOpt.RequireMain = false
	}
	if _, err := compiler.CheckWorldOpt(world, checkOpt); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	fmt.Fprintf(stdout, "Checked: %s\n", input)
	return 0
}

func runLSP(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("lsp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	smokePath := fs.String("stdio-smoke", "", "analyze one .t4/.tetra file and print LSP-basic JSON")
	stdio := fs.Bool("stdio", false, "run LSP-basic JSON-RPC over stdio")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "lsp does not accept positional arguments")
		return 2
	}
	if *stdio {
		return runLSPStdio(os.Stdin, stdout, stderr)
	}
	if *smokePath == "" {
		fmt.Fprintln(stderr, "lsp requires --stdio or --stdio-smoke <file>")
		return 2
	}
	analysis, err := compiler.AnalyzeLSPFile(*smokePath)
	if err != nil {
		writeDiagnostic(stderr, "json", err)
		return 1
	}
	enc := json.NewEncoder(stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(analysis); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

type lspRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type lspTextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type lspDidOpenParams struct {
	TextDocument struct {
		URI  string `json:"uri"`
		Text string `json:"text"`
	} `json:"textDocument"`
}

type lspDidChangeParams struct {
	TextDocument   lspTextDocumentIdentifier `json:"textDocument"`
	ContentChanges []struct {
		Text string `json:"text"`
	} `json:"contentChanges"`
}

type lspTextDocumentParams struct {
	TextDocument lspTextDocumentIdentifier `json:"textDocument"`
}

type lspDidCloseParams struct {
	TextDocument lspTextDocumentIdentifier `json:"textDocument"`
}

type lspHoverParams struct {
	TextDocument lspTextDocumentIdentifier `json:"textDocument"`
	Position     struct {
		Line      int `json:"line"`
		Character int `json:"character"`
	} `json:"position"`
}

type lspDefinitionParams struct {
	TextDocument lspTextDocumentIdentifier `json:"textDocument"`
	Position     struct {
		Line      int `json:"line"`
		Character int `json:"character"`
	} `json:"position"`
}

type lspReferencesParams struct {
	TextDocument lspTextDocumentIdentifier `json:"textDocument"`
	Position     struct {
		Line      int `json:"line"`
		Character int `json:"character"`
	} `json:"position"`
	Context struct {
		IncludeDeclaration bool `json:"includeDeclaration"`
	} `json:"context"`
}

type lspRenameParams struct {
	TextDocument lspTextDocumentIdentifier `json:"textDocument"`
	Position     struct {
		Line      int `json:"line"`
		Character int `json:"character"`
	} `json:"position"`
	NewName string `json:"newName"`
}

type lspCodeActionParams struct {
	TextDocument lspTextDocumentIdentifier `json:"textDocument"`
	Context      struct {
		Diagnostics []lspCodeActionDiagnostic `json:"diagnostics"`
	} `json:"context"`
}

type lspCodeActionDiagnostic struct {
	Code    json.RawMessage `json:"code,omitempty"`
	Message string          `json:"message"`
}

type lspOpenDocument struct {
	Text     string
	Analysis compiler.LSPAnalysis
}

var lspMissingEffectDiagnosticRE = regexp.MustCompile(`^function '([^']+)' uses effect '([^']+)' but does not declare it$`)

func runLSPStdio(stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	reader := bufio.NewReader(stdin)
	openDocs := map[string]lspOpenDocument{}
	shutdown := false
	for {
		body, err := readLSPMessage(reader)
		if err == io.EOF {
			return 0
		}
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		var req lspRequest
		if err := json.Unmarshal(body, &req); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		switch req.Method {
		case "initialize":
			if req.ID != nil {
				result := map[string]any{
					"capabilities": map[string]any{
						"textDocumentSync":           1,
						"documentSymbolProvider":     true,
						"hoverProvider":              true,
						"definitionProvider":         true,
						"referencesProvider":         true,
						"renameProvider":             true,
						"documentFormattingProvider": true,
						"codeActionProvider":         true,
						"completionProvider": map[string]any{
							"resolveProvider": false,
						},
					},
				}
				if err := writeLSPResponse(stdout, *req.ID, result); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		case "shutdown":
			shutdown = true
			if req.ID != nil {
				if err := writeLSPResponse(stdout, *req.ID, nil); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		case "exit":
			if shutdown {
				return 0
			}
			return 1
		case "textDocument/didOpen":
			var params lspDidOpenParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			analysis := compiler.AnalyzeLSPSource([]byte(params.TextDocument.Text), params.TextDocument.URI)
			openDocs[params.TextDocument.URI] = lspOpenDocument{Text: params.TextDocument.Text, Analysis: analysis}
			if err := writeLSPNotification(stdout, "textDocument/publishDiagnostics", map[string]any{
				"uri":         params.TextDocument.URI,
				"diagnostics": lspDiagnostics(analysis.Diagnostics),
			}); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
		case "textDocument/didChange":
			var params lspDidChangeParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			if len(params.ContentChanges) == 0 {
				continue
			}
			text := params.ContentChanges[len(params.ContentChanges)-1].Text
			analysis := compiler.AnalyzeLSPSource([]byte(text), params.TextDocument.URI)
			openDocs[params.TextDocument.URI] = lspOpenDocument{Text: text, Analysis: analysis}
			if err := writeLSPNotification(stdout, "textDocument/publishDiagnostics", map[string]any{
				"uri":         params.TextDocument.URI,
				"diagnostics": lspDiagnostics(analysis.Diagnostics),
			}); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
		case "textDocument/didClose":
			var params lspDidCloseParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			delete(openDocs, params.TextDocument.URI)
			if err := writeLSPNotification(stdout, "textDocument/publishDiagnostics", map[string]any{
				"uri":         params.TextDocument.URI,
				"diagnostics": []any{},
			}); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
		case "textDocument/documentSymbol":
			var params lspTextDocumentParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			if req.ID != nil {
				if err := writeLSPResponse(stdout, *req.ID, lspDocumentSymbols(openDocs[params.TextDocument.URI].Analysis)); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		case "textDocument/hover":
			var params lspHoverParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			if req.ID != nil {
				if err := writeLSPResponse(stdout, *req.ID, lspHoverAt(openDocs[params.TextDocument.URI].Analysis, params.Position.Line)); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		case "textDocument/definition":
			var params lspDefinitionParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			if req.ID != nil {
				var result any
				if doc, ok := openDocs[params.TextDocument.URI]; ok {
					result = lspDefinitionLocations(doc, params.TextDocument.URI, params.Position.Line, params.Position.Character)
				}
				if err := writeLSPResponse(stdout, *req.ID, result); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		case "textDocument/references":
			var params lspReferencesParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			if req.ID != nil {
				var result any
				if doc, ok := openDocs[params.TextDocument.URI]; ok {
					result = lspReferenceLocations(doc, params.TextDocument.URI, params.Position.Line, params.Position.Character, params.Context.IncludeDeclaration)
				}
				if err := writeLSPResponse(stdout, *req.ID, result); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		case "textDocument/rename":
			var params lspRenameParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			if req.ID != nil {
				var result any
				if doc, ok := openDocs[params.TextDocument.URI]; ok {
					result = lspRenameWorkspaceEdit(doc, params.TextDocument.URI, params.Position.Line, params.Position.Character, params.NewName)
				}
				if err := writeLSPResponse(stdout, *req.ID, result); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		case "textDocument/completion":
			var params lspHoverParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			if req.ID != nil {
				if err := writeLSPResponse(stdout, *req.ID, lspCompletionItems(openDocs[params.TextDocument.URI].Analysis)); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		case "textDocument/formatting":
			var params lspTextDocumentParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			if req.ID != nil {
				edits := []map[string]any{}
				if doc, ok := openDocs[params.TextDocument.URI]; ok {
					var err error
					edits, err = lspFormattingEdits(doc.Text, params.TextDocument.URI)
					if err != nil {
						edits = []map[string]any{}
					}
				}
				if err := writeLSPResponse(stdout, *req.ID, edits); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		case "textDocument/codeAction":
			var params lspCodeActionParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			if req.ID != nil {
				actions := []map[string]any{}
				if doc, ok := openDocs[params.TextDocument.URI]; ok {
					actions = lspCodeActions(doc.Text, params.TextDocument.URI, params.Context.Diagnostics, doc.Analysis.Diagnostics)
				}
				if err := writeLSPResponse(stdout, *req.ID, actions); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		default:
			if req.ID != nil {
				if err := writeLSPResponse(stdout, *req.ID, nil); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		}
	}
}

func readLSPMessage(reader *bufio.Reader) ([]byte, error) {
	length := -1
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("invalid LSP header %q", line)
		}
		if strings.EqualFold(strings.TrimSpace(key), "Content-Length") {
			parsed, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length")
			}
			length = parsed
		}
	}
	if length < 0 {
		return nil, fmt.Errorf("missing Content-Length")
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, err
	}
	return body, nil
}

func writeLSPResponse(w io.Writer, id int, result any) error {
	return writeLSPMessage(w, map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

func writeLSPNotification(w io.Writer, method string, params any) error {
	return writeLSPMessage(w, map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

func writeLSPMessage(w io.Writer, msg any) error {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(msg); err != nil {
		return err
	}
	raw := bytes.TrimRight(b.Bytes(), "\n")
	_, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n%s", len(raw), raw)
	return err
}

func lspDiagnostics(diags []compiler.Diagnostic) []map[string]any {
	out := make([]map[string]any, 0, len(diags))
	for _, diag := range diags {
		line := diag.Line
		if line > 0 {
			line--
		}
		col := diag.Column
		if col > 0 {
			col--
		}
		out = append(out, map[string]any{
			"range": map[string]any{
				"start": map[string]int{"line": line, "character": col},
				"end":   map[string]int{"line": line, "character": col + 1},
			},
			"severity": 1,
			"code":     diag.Code,
			"source":   "tetra",
			"message":  diag.Message,
		})
	}
	return out
}

func lspDocumentSymbols(analysis compiler.LSPAnalysis) []map[string]any {
	out := make([]map[string]any, 0, len(analysis.Symbols))
	for _, sym := range analysis.Symbols {
		line := maxInt(sym.Line-1, 0)
		col := maxInt(sym.Column-1, 0)
		item := map[string]any{
			"name": sym.Name,
			"kind": lspSymbolKind(sym.Kind),
			"range": map[string]any{
				"start": map[string]int{"line": line, "character": col},
				"end":   map[string]int{"line": line, "character": col + 1},
			},
			"selectionRange": map[string]any{
				"start": map[string]int{"line": line, "character": col},
				"end":   map[string]int{"line": line, "character": col + len(sym.Name)},
			},
		}
		if sym.Detail != "" {
			item["detail"] = sym.Detail
		}
		out = append(out, item)
	}
	return out
}

func lspHoverAt(analysis compiler.LSPAnalysis, zeroBasedLine int) any {
	line := zeroBasedLine + 1
	for _, hover := range analysis.Hovers {
		if hover.Line == line {
			return map[string]any{"contents": map[string]string{"kind": "markdown", "value": hover.Contents}}
		}
	}
	return nil
}

func lspDefinitionLocations(doc lspOpenDocument, uri string, zeroBasedLine int, zeroBasedCharacter int) any {
	name := lspIdentifierAt(doc.Text, zeroBasedLine, zeroBasedCharacter)
	if name == "" {
		return nil
	}
	line, col, ok := lspDefinitionPosition(doc, name)
	if !ok {
		return nil
	}
	return []map[string]any{{
		"uri": uri,
		"range": map[string]any{
			"start": map[string]int{"line": line, "character": col},
			"end":   map[string]int{"line": line, "character": col + len(name)},
		},
	}}
}

func lspIdentifierAt(text string, zeroBasedLine int, zeroBasedCharacter int) string {
	if zeroBasedLine < 0 || zeroBasedCharacter < 0 {
		return ""
	}
	lines := strings.Split(text, "\n")
	if zeroBasedLine >= len(lines) {
		return ""
	}
	line := lines[zeroBasedLine]
	if len(line) == 0 {
		return ""
	}
	idx := zeroBasedCharacter
	if idx >= len(line) {
		idx = len(line) - 1
	}
	if !isLSPIdentifierChar(line[idx]) {
		if idx > 0 && isLSPIdentifierChar(line[idx-1]) {
			idx--
		} else {
			return ""
		}
	}
	start := idx
	for start > 0 && isLSPIdentifierChar(line[start-1]) {
		start--
	}
	end := idx + 1
	for end < len(line) && isLSPIdentifierChar(line[end]) {
		end++
	}
	return line[start:end]
}

func isLSPIdentifierChar(ch byte) bool {
	return ch == '_' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9'
}

func lspDefinitionColumn(text string, sym compiler.LSPSymbol) int {
	line := maxInt(sym.Line-1, 0)
	col := maxInt(sym.Column-1, 0)
	lines := strings.Split(text, "\n")
	if line < 0 || line >= len(lines) {
		return col
	}
	lineText := lines[line]
	if col >= 0 && col+len(sym.Name) <= len(lineText) && lineText[col:col+len(sym.Name)] == sym.Name {
		return col
	}
	if idx := strings.Index(lineText, sym.Name); idx >= 0 {
		return idx
	}
	return col
}

func lspDefinitionPosition(doc lspOpenDocument, name string) (int, int, bool) {
	for _, sym := range doc.Analysis.Symbols {
		if sym.Name != name {
			continue
		}
		line := maxInt(sym.Line-1, 0)
		col := lspDefinitionColumn(doc.Text, sym)
		return line, col, true
	}
	return 0, 0, false
}

func lspReferenceLocations(doc lspOpenDocument, uri string, zeroBasedLine int, zeroBasedCharacter int, includeDeclaration bool) any {
	name := lspIdentifierAt(doc.Text, zeroBasedLine, zeroBasedCharacter)
	if name == "" {
		return nil
	}
	defLine, defCol, hasDefinition := lspDefinitionPosition(doc, name)
	locations := []map[string]any{}
	for _, ref := range lspIdentifierReferences(doc.Text, name) {
		if !includeDeclaration && hasDefinition && ref.Line == defLine && ref.Column == defCol {
			continue
		}
		locations = append(locations, map[string]any{
			"uri": uri,
			"range": map[string]any{
				"start": map[string]int{"line": ref.Line, "character": ref.Column},
				"end":   map[string]int{"line": ref.Line, "character": ref.Column + len(name)},
			},
		})
	}
	return locations
}

type lspReference struct {
	Line   int
	Column int
}

func lspIdentifierReferences(text string, name string) []lspReference {
	if name == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	refs := []lspReference{}
	for line, content := range lines {
		searchFrom := 0
		for searchFrom <= len(content)-len(name) {
			offset := strings.Index(content[searchFrom:], name)
			if offset < 0 {
				break
			}
			col := searchFrom + offset
			startOk := col == 0 || !isLSPIdentifierChar(content[col-1])
			end := col + len(name)
			endOk := end == len(content) || !isLSPIdentifierChar(content[end])
			searchFrom = end
			if !startOk || !endOk {
				continue
			}
			refs = append(refs, lspReference{Line: line, Column: col})
		}
	}
	return refs
}

func lspRenameWorkspaceEdit(doc lspOpenDocument, uri string, zeroBasedLine int, zeroBasedCharacter int, newName string) any {
	if strings.TrimSpace(newName) == "" {
		return nil
	}
	oldName := lspIdentifierAt(doc.Text, zeroBasedLine, zeroBasedCharacter)
	if oldName == "" {
		return nil
	}
	refs := lspIdentifierReferences(doc.Text, oldName)
	if len(refs) == 0 {
		return nil
	}
	edits := make([]map[string]any, 0, len(refs))
	for _, ref := range refs {
		edits = append(edits, map[string]any{
			"range": map[string]any{
				"start": map[string]int{"line": ref.Line, "character": ref.Column},
				"end":   map[string]int{"line": ref.Line, "character": ref.Column + len(oldName)},
			},
			"newText": newName,
		})
	}
	return map[string]any{
		"changes": map[string]any{
			uri: edits,
		},
	}
}

func lspCompletionItems(analysis compiler.LSPAnalysis) []map[string]any {
	out := make([]map[string]any, 0, len(analysis.Symbols))
	for _, sym := range analysis.Symbols {
		item := map[string]any{
			"label": sym.Name,
			"kind":  lspCompletionKind(sym.Kind),
		}
		if sym.Detail != "" {
			item["detail"] = sym.Detail
		}
		out = append(out, item)
	}
	return out
}

func lspCodeActions(text string, uri string, requestDiagnostics []lspCodeActionDiagnostic, analysisDiagnostics []compiler.Diagnostic) []map[string]any {
	diagnostics := requestDiagnostics
	if len(diagnostics) == 0 {
		diagnostics = lspCodeActionDiagnosticsFromCompiler(analysisDiagnostics)
	}
	actions := []map[string]any{}
	for _, diag := range diagnostics {
		action, ok := lspMissingEffectCodeAction(text, uri, diag)
		if ok {
			actions = append(actions, action)
		}
	}
	return actions
}

func lspCodeActionDiagnosticsFromCompiler(diags []compiler.Diagnostic) []lspCodeActionDiagnostic {
	out := make([]lspCodeActionDiagnostic, 0, len(diags))
	for _, diag := range diags {
		code, err := json.Marshal(diag.Code)
		if err != nil {
			continue
		}
		out = append(out, lspCodeActionDiagnostic{
			Code:    code,
			Message: diag.Message,
		})
	}
	return out
}

func lspMissingEffectCodeAction(text string, uri string, diag lspCodeActionDiagnostic) (map[string]any, bool) {
	code := lspDiagnosticCodeString(diag.Code)
	if code != "" && code != "TETRA2001" {
		return nil, false
	}
	match := lspMissingEffectDiagnosticRE.FindStringSubmatch(diag.Message)
	if len(match) != 3 {
		return nil, false
	}
	funcName := match[1]
	effect := match[2]
	line, character, newText, ok := lspFindUsesInsertion(text, funcName, effect)
	if !ok {
		return nil, false
	}
	edit := map[string]any{
		"range": map[string]any{
			"start": map[string]int{"line": line, "character": character},
			"end":   map[string]int{"line": line, "character": character},
		},
		"newText": newText,
	}
	return map[string]any{
		"title": fmt.Sprintf("Add uses %s to function %s", effect, funcName),
		"kind":  "quickfix",
		"edit": map[string]any{
			"changes": map[string]any{
				uri: []map[string]any{edit},
			},
		},
	}, true
}

func lspDiagnosticCodeString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var codeString string
	if err := json.Unmarshal(raw, &codeString); err == nil {
		return codeString
	}
	return ""
}

func lspFindUsesInsertion(text string, funcName string, effect string) (int, int, string, bool) {
	lines := strings.Split(text, "\n")
	prefix := "func " + funcName + "("
	for lineIdx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		if usesIdx := strings.Index(trimmed, " uses "); usesIdx >= 0 {
			colonIdx := strings.LastIndex(trimmed, ":")
			if colonIdx > usesIdx {
				existing := strings.TrimSpace(trimmed[usesIdx+len(" uses ") : colonIdx])
				if lspUsesContainsEffect(existing, effect) {
					return 0, 0, "", false
				}
				return lineIdx, lspLineIndent(line) + colonIdx, ", " + effect, true
			}
		}
		if colonIdx := strings.LastIndex(trimmed, ":"); colonIdx >= 0 {
			return lineIdx, lspLineIndent(line) + colonIdx, " uses " + effect, true
		}
		if lineIdx+1 >= len(lines) {
			return 0, 0, "", false
		}
		nextLine := lines[lineIdx+1]
		nextTrimmed := strings.TrimSpace(nextLine)
		if !strings.HasPrefix(nextTrimmed, "uses ") {
			return 0, 0, "", false
		}
		colonIdx := strings.LastIndex(nextTrimmed, ":")
		if colonIdx < 0 {
			return 0, 0, "", false
		}
		existing := strings.TrimSpace(nextTrimmed[len("uses "):colonIdx])
		if lspUsesContainsEffect(existing, effect) {
			return 0, 0, "", false
		}
		return lineIdx + 1, lspLineIndent(nextLine) + colonIdx, ", " + effect, true
	}
	return 0, 0, "", false
}

func lspLineIndent(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}

func lspUsesContainsEffect(existing string, effect string) bool {
	for _, item := range strings.Split(existing, ",") {
		if strings.TrimSpace(item) == effect {
			return true
		}
	}
	return false
}

func lspCompletionKind(kind string) int {
	switch kind {
	case "function", "extension-method":
		return 3
	case "const":
		return 21
	case "enum":
		return 13
	case "protocol":
		return 8
	case "struct":
		return 7
	default:
		return 6
	}
}

func lspFormattingEdits(text string, uri string) ([]map[string]any, error) {
	formatted, err := compiler.FormatSource([]byte(text), uri)
	if err != nil {
		return nil, err
	}
	if string(formatted) == text {
		return []map[string]any{}, nil
	}
	line, character := lspFullDocumentEnd(text)
	return []map[string]any{{
		"range": map[string]any{
			"start": map[string]int{"line": 0, "character": 0},
			"end":   map[string]int{"line": line, "character": character},
		},
		"newText": string(formatted),
	}}, nil
}

func lspFullDocumentEnd(text string) (int, int) {
	parts := strings.Split(text, "\n")
	return len(parts) - 1, len(parts[len(parts)-1])
}

func lspSymbolKind(kind string) int {
	switch kind {
	case "function":
		return 12
	case "extension-method":
		return 6
	case "const":
		return 14
	case "val", "var":
		return 13
	case "enum":
		return 10
	case "protocol":
		return 11
	case "struct":
		return 23
	default:
		return 13
	}
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func runBuild(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", "", "target triple ("+supportedTargetsHelp+"); defaults to Capsule.t4 first target, then host")
	out := fs.String("o", "", "output path")
	allTargets := fs.Bool("all-targets", false, "build every target listed in Capsule.t4")
	interfaceOnly := fs.Bool("interface-only", false, "type-check interface/API graph without emitting executable code")
	islandsDebug := fs.Bool("islands-debug", false, "enable islands debug runtime checks")
	emit := fs.String("emit", "exe", "emit mode: exe, object, or library")
	runtimeMode := fs.String("runtime", "auto", "actors runtime: auto, selfhost, or builtin")
	runtimeObject := fs.String("runtime-object", "", "actors runtime object override")
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	artifactsMode := fs.String("artifacts", "strict", "artifact handling: strict or auto")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text or json")
	var linkObjects multiFlag
	fs.Var(&linkObjects, "link-object", "extra TOBJ object to link")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	if *artifactsMode != "strict" && *artifactsMode != "auto" {
		writeValidationDiagnostic(stderr, *diagnostics, "build --artifacts must be strict or auto")
		return 2
	}

	input := ""
	if fs.NArg() > 0 {
		input = fs.Arg(0)
	}
	if fs.NArg() > 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "build accepts at most one input path")
		return 2
	}

	input, worldOpt, projectCtx, err := resolveCLIInput(input)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if *artifactsMode == "auto" && projectCtx != nil && projectCtx.Found {
		if err := buildCapsuleArtifacts(projectCtx.CapsulePath, capsuleArtifactBuildOptions{
			Target:     *target,
			LockPath:   projectCtx.LockPath,
			Jobs:       *jobs,
			AllTargets: *allTargets,
		}); err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		fmt.Fprintf(stdout, "Artifacts repaired: %s\n", projectCtx.CapsulePath)
		input, worldOpt, projectCtx, err = resolveCLIInput(input)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
	}

	if *allTargets {
		targets := projectBuildTargets(projectCtx)
		if len(targets) == 0 {
			writeValidationDiagnostic(stderr, *diagnostics, "build --all-targets requires targets in Capsule.t4")
			return 2
		}
		targetLinkObjects, err := projectLinkObjects(projectCtx, "", []string(linkObjects))
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		opt, err := buildOptions(*emit, *runtimeMode, *islandsDebug, *runtimeObject, targetLinkObjects, *jobs)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 2
		}
		opt.ProjectRoot = worldOpt.Root
		opt.SourceRoots = worldOpt.SourceRoots
		opt.DependencyRoots = worldOpt.DependencyRoots
		opt.InterfaceOnly = *interfaceOnly
		for _, rawTarget := range targets {
			tgt, ok := parseBuildTargetOrReport(rawTarget, *diagnostics, stderr)
			if !ok {
				return 2
			}
			if err := validateDiscoveredProjectLock(projectCtx, tgt.Triple); err != nil {
				writeDiagnostic(stderr, *diagnostics, err)
				return 1
			}
			targetLinkObjects, err := projectLinkObjects(projectCtx, tgt.Triple, []string(linkObjects))
			if err != nil {
				writeDiagnostic(stderr, *diagnostics, err)
				return 1
			}
			opt.LinkObjectPaths = targetLinkObjects
			output := allTargetsOutput(*out, tgt, *emit)
			if _, err := compiler.BuildFileWithStatsOpt(input, output, tgt.Triple, opt); err != nil {
				writeDiagnostic(stderr, *diagnostics, err)
				return 1
			}
			if *interfaceOnly {
				fmt.Fprintf(stdout, "Interface-only build checked: %s (%s)\n", input, tgt.Triple)
			} else {
				fmt.Fprintf(stdout, "Built: %s\n", output)
			}
		}
		return 0
	}

	rawTarget := *target
	if rawTarget == "" {
		rawTarget = projectDefaultTarget(projectCtx)
	}
	tgt, ok := parseBuildTargetOrReport(rawTarget, *diagnostics, stderr)
	if !ok {
		return 2
	}
	output := *out
	if output == "" {
		output = defaultOutput(tgt, *emit)
	}

	if err := validateDiscoveredProjectLock(projectCtx, tgt.Triple); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}

	targetLinkObjects, err := projectLinkObjects(projectCtx, tgt.Triple, []string(linkObjects))
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	opt, err := buildOptions(*emit, *runtimeMode, *islandsDebug, *runtimeObject, targetLinkObjects, *jobs)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 2
	}
	opt.ProjectRoot = worldOpt.Root
	opt.SourceRoots = worldOpt.SourceRoots
	opt.DependencyRoots = worldOpt.DependencyRoots
	opt.InterfaceOnly = *interfaceOnly
	if _, err := compiler.BuildFileWithStatsOpt(input, output, tgt.Triple, opt); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if *interfaceOnly {
		fmt.Fprintf(stdout, "Interface-only build checked: %s\n", input)
	} else {
		fmt.Fprintf(stdout, "Built: %s\n", output)
	}
	return 0
}

func runRun(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", "", "target triple ("+supportedTargetsHelp+"); defaults to Capsule.t4 first target, then host")
	out := fs.String("o", "", "output path")
	islandsDebug := fs.Bool("islands-debug", false, "enable islands debug runtime checks")
	runtimeMode := fs.String("runtime", "auto", "actors runtime: auto, selfhost, or builtin")
	runtimeObject := fs.String("runtime-object", "", "actors runtime object override")
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text or json")
	var linkObjects multiFlag
	fs.Var(&linkObjects, "link-object", "extra TOBJ object to link")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	input := ""
	if fs.NArg() > 0 {
		input = fs.Arg(0)
	}
	if fs.NArg() > 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "run accepts at most one input path")
		return 2
	}
	input, worldOpt, projectCtx, err := resolveCLIInput(input)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	rawTarget := *target
	if rawTarget == "" {
		rawTarget = projectDefaultTarget(projectCtx)
	}
	tgt, ok := parseBuildTargetOrReport(rawTarget, *diagnostics, stderr)
	if !ok {
		return 2
	}
	isWASI := tgt.Triple == "wasm32-wasi"
	if !isWASI {
		host, ok := hostTarget()
		if !ok || host != tgt.Triple {
			writeDiagnostic(stderr, *diagnostics, fmt.Errorf("cannot run target %s on host %s/%s", tgt.Triple, runtime.GOOS, runtime.GOARCH))
			return 2
		}
	}
	output := *out
	tmpDir := ""
	if output == "" {
		var err error
		tmpDir, err = os.MkdirTemp("", "tetra-run-*")
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		defer os.RemoveAll(tmpDir)
		output = filepath.Join(tmpDir, defaultOutput(tgt, "exe"))
	}
	if err := validateDiscoveredProjectLock(projectCtx, tgt.Triple); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	targetLinkObjects, err := projectLinkObjects(projectCtx, tgt.Triple, []string(linkObjects))
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	opt, err := buildOptions("exe", *runtimeMode, *islandsDebug, *runtimeObject, targetLinkObjects, *jobs)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 2
	}
	opt.ProjectRoot = worldOpt.Root
	opt.SourceRoots = worldOpt.SourceRoots
	opt.DependencyRoots = worldOpt.DependencyRoots
	if _, err := compiler.BuildFileWithStatsOpt(input, output, tgt.Triple, opt); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if isWASI {
		exit, err := execWASMProgram(output, stdout, stderr)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		return exit
	}
	return execProgram(output, stdout, stderr)
}

func runFmt(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("fmt", flag.ContinueOnError)
	fs.SetOutput(stderr)
	check := fs.Bool("check", false, "check whether files are formatted")
	write := fs.Bool("write", false, "rewrite files in place")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	if *check && *write {
		writeValidationDiagnostic(stderr, *diagnostics, "fmt accepts only one of --check or --write")
		return 2
	}
	paths := fs.Args()
	if len(paths) == 0 {
		writeValidationDiagnostic(stderr, *diagnostics, "fmt requires at least one path")
		return 2
	}
	files, err := collectTetraFiles(paths)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if !*check && !*write && len(files) != 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "fmt stdout mode accepts exactly one file")
		return 2
	}
	dirty := false
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		formatted, err := compiler.FormatSource(raw, path)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		if *check {
			if string(raw) != string(formatted) {
				dirty = true
				if *diagnostics == "json" {
					line, column := firstFormatterDiffPosition(raw, formatted)
					writeDiagnosticObject(stderr, compiler.Diagnostic{
						Code:     compiler.DiagnosticCodeFormatterCheck,
						Message:  "not formatted",
						File:     path,
						Line:     line,
						Column:   column,
						Severity: "error",
						Hint:     "Run tetra fmt --write to update the file.",
					})
				} else {
					fmt.Fprintf(stderr, "%s: not formatted\n", path)
				}
			}
			continue
		}
		if *write {
			if string(raw) != string(formatted) {
				if err := os.WriteFile(path, formatted, 0o644); err != nil {
					writeDiagnostic(stderr, *diagnostics, err)
					return 1
				}
			}
			continue
		}
		fmt.Fprint(stdout, string(formatted))
	}
	if dirty {
		return 1
	}
	return 0
}

func firstFormatterDiffPosition(raw []byte, formatted []byte) (int, int) {
	line := 1
	column := 1
	limit := len(raw)
	if len(formatted) < limit {
		limit = len(formatted)
	}
	for i := 0; i < limit; i++ {
		if raw[i] != formatted[i] {
			return line, column
		}
		if raw[i] == '\n' {
			line++
			column = 1
			continue
		}
		column++
	}
	return line, column
}

func runTest(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", defaultTarget(), "target triple ("+supportedTargetsHelp+")")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text or json")
	reportFormat := fs.String("report", "text", "report format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	if *reportFormat != "text" && *reportFormat != "json" {
		writeValidationDiagnostic(stderr, *diagnostics, "unsupported --report format")
		return 2
	}
	paths := fs.Args()
	var projectCtx *cliProjectContext
	var worldOpt compiler.WorldOptions
	if len(paths) == 0 {
		ctx, err := discoverCLIProject(".")
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		if ctx != nil && ctx.Found {
			projectCtx = ctx
			worldOpt = compiler.WorldOptions{
				Root:            ctx.Root,
				SourceRoots:     append([]string(nil), ctx.SourceRoots...),
				DependencyRoots: append([]compiler.ModuleRoot(nil), ctx.DependencyRoots...),
			}
			paths = existingProjectSourcePaths(ctx)
		}
		if len(paths) == 0 {
			paths = []string{"."}
		}
	} else if len(paths) == 1 {
		resolved, resolvedWorldOpt, ctx, err := resolveCLIInput(paths[0])
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		if ctx != nil && ctx.Found && isProjectReference(paths[0], ctx) {
			projectCtx = ctx
			worldOpt = resolvedWorldOpt
			paths = existingProjectSourcePaths(ctx)
			if len(paths) == 0 {
				paths = []string{resolved}
			}
		}
	}
	tgt, ok := parseBuildTargetOrReport(*target, *diagnostics, stderr)
	if !ok {
		return 2
	}
	host, ok := hostTarget()
	if !ok || host != tgt.Triple {
		writeDiagnostic(stderr, *diagnostics, fmt.Errorf("cannot run tests for target %s on host %s/%s", tgt.Triple, runtime.GOOS, runtime.GOARCH))
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
	files, err := collectTetraFiles(paths)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	tmpDir, err := os.MkdirTemp("", "tetra-test-*")
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	defer os.RemoveAll(tmpDir)
	total := 0
	passed := 0
	var results []compiler.TestRunnerResult
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		runners, err := compiler.TestRunnerSources(raw, file)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		for i, runner := range runners {
			total++
			start := time.Now()
			srcPath := filepath.Join(tmpDir, fmt.Sprintf("test_%d.t4", total))
			runnerSource := runner.Source
			if modulePath := modulePathFromSource(runner.Source); modulePath != "" {
				var err error
				srcPath, runnerSource, err = runnerSourcePathForModuleFile(file, runner.Source, total)
				if err != nil {
					writeDiagnostic(stderr, *diagnostics, err)
					return 1
				}
				defer os.Remove(srcPath)
			}
			outPath := filepath.Join(tmpDir, fmt.Sprintf("test_%d%s", total, tgt.ExeExt))
			if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
				writeDiagnostic(stderr, *diagnostics, err)
				return 1
			}
			if err := os.WriteFile(srcPath, runnerSource, 0o644); err != nil {
				writeDiagnostic(stderr, *diagnostics, err)
				return 1
			}
			if modulePathFromSource(runnerSource) != "" && projectCtx != nil && projectCtx.Found {
				opt := compiler.BuildOptions{
					Jobs:            1,
					ProjectRoot:     worldOpt.Root,
					SourceRoots:     append([]string(nil), worldOpt.SourceRoots...),
					DependencyRoots: append([]compiler.ModuleRoot(nil), worldOpt.DependencyRoots...),
					LinkObjectPaths: append([]string(nil), targetLinkObjects...),
				}
				if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, opt); err != nil {
					writeDiagnostic(stderr, *diagnostics, err)
					return 1
				}
			} else {
				if err := compiler.BuildFile(srcPath, outPath, tgt.Triple); err != nil {
					writeDiagnostic(stderr, *diagnostics, err)
					return 1
				}
			}
			code := execProgram(outPath, io.Discard, io.Discard)
			name := runner.Name
			if name == "" {
				name = fmt.Sprintf("%s#%d", file, i+1)
			}
			result := runner.ResultWithDuration(code, nil, elapsedMillis(time.Since(start)))
			results = append(results, result)
			if code == 0 {
				passed++
				if *reportFormat == "text" {
					fmt.Fprintf(stdout, "PASS %s\n", name)
				}
			} else {
				if *reportFormat == "text" {
					if result.Error != "" {
						fmt.Fprintf(stdout, "FAIL %s (%s)\n", name, result.Error)
					} else {
						fmt.Fprintf(stdout, "FAIL %s\n", name)
					}
				}
			}
		}
	}
	if *reportFormat == "json" {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(compiler.NewTestRunnerReport(results)); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	} else {
		fmt.Fprintf(stdout, "Tetra tests: %d/%d passed\n", passed, total)
	}
	if passed != total {
		return 1
	}
	return 0
}

func elapsedMillis(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	ms := d.Milliseconds()
	if ms == 0 {
		return 1
	}
	return ms
}

func runSmoke(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("smoke", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", defaultTarget(), "target triple ("+supportedTargetsHelp+")")
	runBuilt := fs.Bool("run", true, "run built binaries when host matches target")
	reportPath := fs.String("report", "", "write JSON smoke report")
	listCases := fs.Bool("list", false, "list smoke cases without building")
	listFormat := fs.String("format", "text", "smoke list format: text or json")
	islandsDebug := fs.Bool("islands-debug", false, "enable islands debug runtime checks")
	runtimeMode := fs.String("runtime", "auto", "actors runtime: auto, selfhost, or builtin")
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "smoke does not accept positional arguments")
		return 2
	}
	if *listCases {
		tgt, ok := parseBuildTargetOrReport(*target, "text", stderr)
		if !ok {
			return 2
		}
		return writeSmokeList(stdout, stderr, smokeCasesForTarget(*islandsDebug, tgt), *islandsDebug, *listFormat, tgt)
	}
	if *listFormat != "text" {
		fmt.Fprintln(stderr, "--format is only supported with --list")
		return 2
	}
	tgt, ok := parseBuildTargetOrReport(*target, "text", stderr)
	if !ok {
		return 2
	}
	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	tmpDir, err := os.MkdirTemp("", "tetra-smoke-*")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer os.RemoveAll(tmpDir)

	host := ""
	hostTriple, hostOK := hostTarget()
	if hostOK {
		host = hostTriple
	}
	cases := smokeCasesForTarget(*islandsDebug, tgt)
	shouldRun := *runBuilt && hostOK && hostTriple == tgt.Triple
	runWASI := false
	if *runBuilt && tgt.Triple == "wasm32-wasi" {
		runWASI = true
		shouldRun = true
		if _, err := exec.LookPath("wasmtime"); err != nil {
			fmt.Fprintln(stderr, "cannot run target wasm32-wasi: missing 'wasmtime' in PATH")
			return 2
		}
	}
	opt, err := buildOptions("exe", *runtimeMode, *islandsDebug, "", nil, *jobs)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	report := smokeReport{
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Target:       tgt.Triple,
		BuildOnly:    ctarget.IsBuildOnlyTarget(tgt.Triple),
		Host:         host,
		Version:      compiler.Version(),
		GitHead:      gitHead(repoRoot),
		IslandsDebug: *islandsDebug,
	}
	for _, c := range cases {
		outPath := filepath.Join(tmpDir, c.name+tgt.ExeExt)
		srcAbs := filepath.Join(repoRoot, filepath.FromSlash(c.srcPath))
		caseReport := smokeCaseReport{
			Name:         c.name,
			SrcPath:      c.srcPath,
			OutPath:      outPath,
			ExpectedExit: c.expectedExit,
		}
		if _, err := compiler.BuildFileWithStatsOpt(srcAbs, outPath, tgt.Triple, opt); err != nil {
			caseReport.Error = "build: " + err.Error()
			report.Cases = append(report.Cases, caseReport)
			continue
		}
		if shouldRun {
			caseReport.Ran = true
			var actual int
			if runWASI {
				actual, err = execWASMProgram(outPath, io.Discard, io.Discard)
				if err != nil {
					caseReport.Error = "run: " + err.Error()
					caseReport.Pass = false
					report.Cases = append(report.Cases, caseReport)
					continue
				}
			} else {
				actual = execProgram(outPath, io.Discard, io.Discard)
			}
			caseReport.ActualExit = &actual
			caseReport.Pass = actual == c.expectedExit
		} else {
			caseReport.Pass = true
		}
		report.Cases = append(report.Cases, caseReport)
	}

	passed := 0
	for _, c := range report.Cases {
		if c.Pass {
			passed++
		}
	}
	report.Total = len(report.Cases)
	report.Passed = passed
	report.Failed = report.Total - report.Passed
	fmt.Fprintf(stdout, "Smoke %s: %d/%d passed\n", tgt.Triple, passed, len(report.Cases))
	if *reportPath != "" {
		if err := writeJSON(*reportPath, report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	if passed != len(report.Cases) {
		return 1
	}
	return 0
}

func writeSmokeList(stdout io.Writer, stderr io.Writer, cases []smokeCase, islandsDebug bool, format string, tgt ctarget.Target) int {
	host, hostOK := hostTarget()
	report := smokeListReport{
		Target:       tgt.Triple,
		BuildOnly:    ctarget.IsBuildOnlyTarget(tgt.Triple),
		RunSupported: hostOK && host == tgt.Triple && !ctarget.IsBuildOnlyTarget(tgt.Triple),
		Total:        len(cases),
		IslandsDebug: islandsDebug,
		Cases:        make([]smokeListCase, 0, len(cases)),
	}
	for _, c := range cases {
		report.Cases = append(report.Cases, smokeListCase{
			Name:         c.name,
			SrcPath:      c.srcPath,
			TargetGroup:  smokeTargetGroup(tgt.Triple),
			ExpectedExit: c.expectedExit,
			DebugOnly:    c.debugOnly,
		})
	}
	if repoRoot, err := findRepoRoot(); err == nil {
		report.ExcludedExamples = smokeExampleExclusions(repoRoot, cases, tgt)
	}
	switch format {
	case "", "text":
		for _, c := range report.Cases {
			if c.DebugOnly {
				fmt.Fprintf(stdout, "%s %s exit=%d debug-only\n", c.Name, c.SrcPath, c.ExpectedExit)
			} else {
				fmt.Fprintf(stdout, "%s %s exit=%d\n", c.Name, c.SrcPath, c.ExpectedExit)
			}
		}
		return 0
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func smokeExampleExclusions(repoRoot string, cases []smokeCase, tgt ctarget.Target) []smokeExcludedExample {
	covered := map[string]bool{}
	for _, c := range cases {
		covered[filepath.ToSlash(filepath.Clean(c.srcPath))] = true
	}

	examplesRoot := filepath.Join(repoRoot, "examples")
	var out []smokeExcludedExample
	_ = filepath.WalkDir(examplesRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !compiler.IsSourceFile(path) {
			return nil
		}
		rel, err := filepath.Rel(examplesRoot, path)
		if err != nil {
			return nil
		}
		srcPath := "examples/" + filepath.ToSlash(rel)
		if covered[srcPath] {
			return nil
		}
		out = append(out, smokeExcludedExample{
			SrcPath: srcPath,
			Reason:  fmt.Sprintf("not part of %s smoke profile", tgt.Triple),
		})
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].SrcPath < out[j].SrcPath })
	return out
}

func runClean(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("clean", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", "", "remove cache entries for one target")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "clean does not accept positional arguments")
		return 2
	}
	if *target != "" {
		tgt, ok := parseBuildTargetOrReport(*target, "text", stderr)
		if !ok {
			return 2
		}
		for _, path := range []string{
			filepath.Join(".tetra_cache", tgt.Triple),
			filepath.Join("tetra_cache", tgt.Triple),
		} {
			if err := os.RemoveAll(path); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
		}
		fmt.Fprintf(stdout, "Cleaned Tetra cache for %s\n", tgt.Triple)
		return 0
	}
	for _, path := range []string{".tetra_cache", "tetra_cache"} {
		if err := os.RemoveAll(path); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	fmt.Fprintln(stdout, "Cleaned Tetra cache")
	return 0
}

func buildOptions(emit string, runtimeMode string, islandsDebug bool, runtimeObject string, linkObjects []string, jobs int) (compiler.BuildOptions, error) {
	opt := compiler.BuildOptions{
		Jobs:              jobs,
		IslandsDebug:      islandsDebug,
		RuntimeObjectPath: runtimeObject,
		LinkObjectPaths:   linkObjects,
	}
	switch emit {
	case "", "exe":
		opt.Emit = compiler.EmitExe
	case "object":
		opt.Emit = compiler.EmitObject
	case "library":
		opt.Emit = compiler.EmitLibrary
	default:
		return opt, fmt.Errorf("unsupported --emit %q", emit)
	}
	switch runtimeMode {
	case "", "auto":
		opt.Runtime = compiler.RuntimeAuto
	case "selfhost":
		opt.Runtime = compiler.RuntimeSelfHost
	case "builtin":
		opt.Runtime = compiler.RuntimeBuiltin
	default:
		return opt, fmt.Errorf("unsupported --runtime %q", runtimeMode)
	}
	return opt, nil
}

func smokeCases(islandsDebug bool) []smokeCase {
	cases := []smokeCase{
		{name: "islands_hello", srcPath: "examples/islands_hello.tetra", expectedExit: 0},
		{name: "islands_i32", srcPath: "examples/islands_i32.tetra", expectedExit: 55},
		{name: "islands_overflow", srcPath: "examples/islands_overflow.tetra", expectedExit: 1},
		{name: "mmio_smoke", srcPath: "examples/mmio_smoke.tetra", expectedExit: 123},
		{name: "cap_mem_smoke", srcPath: "examples/cap_mem_smoke.tetra", expectedExit: 77},
		{name: "memset_smoke", srcPath: "examples/memset_smoke.tetra", expectedExit: 88},
		{name: "actors_pingpong", srcPath: "examples/actors_pingpong.tetra", expectedExit: 0},
		{name: "actor_sleep_pingpong", srcPath: "examples/actor_sleep_pingpong.tetra", expectedExit: 0},
		{name: "flow_hello", srcPath: "examples/flow_hello.tetra", expectedExit: 0},
		{name: "flow_struct_smoke", srcPath: "examples/flow_struct_smoke.tetra", expectedExit: 42},
		{name: "flow_islands_smoke", srcPath: "examples/flow_islands_smoke.tetra", expectedExit: 0},
		{name: "flow_unsafe_cap_mem_smoke", srcPath: "examples/flow_unsafe_cap_mem_smoke.tetra", expectedExit: 42},
		{name: "ui_native_shell_smoke", srcPath: "examples/ui_native_shell_smoke.tetra", expectedExit: 0},
		{name: "bool_smoke", srcPath: "examples/bool_smoke.tetra", expectedExit: 42},
		{name: "for_range_smoke", srcPath: "examples/for_range_smoke.tetra", expectedExit: 55},
		{name: "for_collection_smoke", srcPath: "examples/for_collection_smoke.tetra", expectedExit: 42},
		{name: "for_collection_u8_smoke", srcPath: "examples/for_collection_u8_smoke.tetra", expectedExit: 42},
		{name: "loop_control_smoke", srcPath: "examples/loop_control_smoke.tetra", expectedExit: 42},
		{name: "complex_control_flow_smoke", srcPath: "examples/complex_control_flow_smoke.tetra", expectedExit: 42},
		{name: "unary_not_smoke", srcPath: "examples/unary_not_smoke.tetra", expectedExit: 42},
		{name: "const_smoke", srcPath: "examples/const_smoke.tetra", expectedExit: 42},
		{name: "const_bool_smoke", srcPath: "examples/const_bool_smoke.tetra", expectedExit: 42},
		{name: "local_const_smoke", srcPath: "examples/local_const_smoke.tetra", expectedExit: 42},
		{name: "compound_assignment_smoke", srcPath: "examples/compound_assignment_smoke.tetra", expectedExit: 42},
		{name: "else_if_smoke", srcPath: "examples/else_if_smoke.tetra", expectedExit: 42},
		{name: "enum_match_smoke", srcPath: "examples/enum_match_smoke.tetra", expectedExit: 42},
		{name: "enum_exhaustive_match_smoke", srcPath: "examples/enum_exhaustive_match_smoke.tetra", expectedExit: 42},
		{name: "effects_io_smoke", srcPath: "examples/effects_io_smoke.tetra", expectedExit: 0},
		{name: "effects_mem_smoke", srcPath: "examples/effects_mem_smoke.tetra", expectedExit: 17},
		{name: "effects_actors_smoke", srcPath: "examples/effects_actors_smoke.tetra", expectedExit: 0},
		{name: "optional_smoke", srcPath: "examples/optional_smoke.tetra", expectedExit: 42},
		{name: "optional_match_smoke", srcPath: "examples/optional_match_smoke.tetra", expectedExit: 42},
		{name: "optional_match_some_smoke", srcPath: "examples/optional_match_some_smoke.tetra", expectedExit: 42},
		{name: "ownership_smoke", srcPath: "examples/ownership_smoke.tetra", expectedExit: 42},
		{name: "typed_errors_smoke", srcPath: "examples/typed_errors_smoke.tetra", expectedExit: 42},
		{name: "async_smoke", srcPath: "examples/async_smoke.tetra", expectedExit: 42},
		{name: "task_smoke", srcPath: "examples/task_smoke.tetra", expectedExit: 42},
		{name: "time_sleep_smoke", srcPath: "examples/time_sleep_smoke.tetra", expectedExit: 0},
		{name: "task_sleep_deadline_smoke", srcPath: "examples/task_sleep_deadline_smoke.tetra", expectedExit: 0},
		{name: "task_join_wait_smoke", srcPath: "examples/task_join_wait_smoke.tetra", expectedExit: 5},
		{name: "deadline_aware_waits_smoke", srcPath: "examples/deadline_aware_waits_smoke.tetra", expectedExit: 0},
		{name: "wait_composition_smoke", srcPath: "examples/wait_composition_smoke.tetra", expectedExit: 0},
		{name: "core_math_smoke", srcPath: "examples/core_math_smoke.tetra", expectedExit: 42},
		{name: "core_memory_smoke", srcPath: "examples/core_memory_smoke.tetra", expectedExit: 42},
		{name: "extension_smoke", srcPath: "examples/extension_smoke.tetra", expectedExit: 42},
		{name: "generic_smoke", srcPath: "examples/generic_smoke.tetra", expectedExit: 42},
		{name: "protocol_impl_smoke", srcPath: "examples/protocol_impl_smoke.tetra", expectedExit: 42},
		{name: "dogfood_cli", srcPath: "examples/projects/dogfood_cli/src/main.tetra", expectedExit: 0},
		{name: "dogfood_actor_task", srcPath: "examples/projects/dogfood_actor_task/src/main.tetra", expectedExit: 0},
	}
	return cases
}

func smokeCasesForTarget(islandsDebug bool, tgt ctarget.Target) []smokeCase {
	if tgt.Triple == "wasm32-wasi" || tgt.Triple == "wasm32-web" {
		return []smokeCase{
			{name: "legacy_hello", srcPath: "examples/hello.tetra", expectedExit: 0},
			{name: "effects_io_smoke", srcPath: "examples/effects_io_smoke.tetra", expectedExit: 0},
			{name: "ui_web_smoke", srcPath: "examples/ui_web_smoke.tetra", expectedExit: 0},
			{name: "dogfood_wasi", srcPath: "examples/projects/dogfood_wasi/src/main.tetra", expectedExit: 0},
			{name: "dogfood_web_ui", srcPath: "examples/projects/dogfood_web_ui/src/main.tetra", expectedExit: 0},
		}
	}
	return smokeCases(islandsDebug)
}

func smokeTargetGroup(target string) string {
	if target == "wasm32-wasi" || target == "wasm32-web" {
		return "wasm"
	}
	return "native"
}

func defaultTarget() string {
	if target, ok := hostTarget(); ok {
		return target
	}
	return "linux-x64"
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

func hostTarget() (string, bool) {
	tgt, ok := ctarget.Host()
	if !ok {
		return "", false
	}
	return tgt.Triple, true
}

func defaultOutput(tgt ctarget.Target, emit string) string {
	switch emit {
	case "object", "library":
		return "app.tobj"
	default:
		return "app" + tgt.ExeExt
	}
}

func projectDefaultTarget(ctx *cliProjectContext) string {
	targets := projectBuildTargets(ctx)
	if len(targets) > 0 {
		return targets[0]
	}
	return defaultTarget()
}

func projectBuildTargets(ctx *cliProjectContext) []string {
	if ctx == nil || !ctx.Found || len(ctx.Manifest.Targets) == 0 {
		return nil
	}
	return append([]string(nil), ctx.Manifest.Targets...)
}

func allTargetsOutput(out string, tgt ctarget.Target, emit string) string {
	name := "app-" + tgt.Triple
	if emit == "object" || emit == "library" {
		name += ".tobj"
	} else {
		name += tgt.ExeExt
	}
	if out == "" {
		return name
	}
	if strings.HasSuffix(out, string(os.PathSeparator)) {
		return filepath.Join(out, name)
	}
	if info, err := os.Stat(out); err == nil && info.IsDir() {
		return filepath.Join(out, name)
	}
	return filepath.Join(out, name)
}

func execProgram(path string, stdout io.Writer, stderr io.Writer) int {
	cmd := exec.Command(path)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode()
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func execWASMProgram(path string, stdout io.Writer, stderr io.Writer) (int, error) {
	runner, err := exec.LookPath("wasmtime")
	if err != nil {
		return 0, fmt.Errorf("cannot run target wasm32-wasi: missing 'wasmtime' in PATH")
	}
	cmd := exec.Command(runner, "run", path)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode(), nil
		}
		return 0, err
	}
	if cmd.ProcessState == nil {
		return 0, nil
	}
	return cmd.ProcessState.ExitCode(), nil
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if fileExists(filepath.Join(dir, "go.work")) && fileExists(filepath.Join(dir, "examples")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repository root")
		}
		dir = parent
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func gitHead(root string) string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func collectTetraFiles(paths []string) ([]string, error) {
	seen := map[string]struct{}{}
	var files []string
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if isCLIManagedSourceFile(path) {
				if _, ok := seen[path]; !ok {
					seen[path] = struct{}{}
					files = append(files, path)
				}
			}
			continue
		}
		err = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if strings.HasPrefix(d.Name(), ".") && p != path {
					return filepath.SkipDir
				}
				return nil
			}
			if isCLIManagedSourceFile(p) {
				if _, ok := seen[p]; !ok {
					seen[p] = struct{}{}
					files = append(files, p)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(files)
	return files, nil
}

func isCLIManagedSourceFile(path string) bool {
	if !compiler.IsSourceFile(path) {
		return false
	}
	base := filepath.Base(path)
	return base != compiler.CapsuleFileName && base != compiler.LegacyCapsuleFileName
}

var moduleDeclRE = regexp.MustCompile(`(?m)^\s*module\s+([A-Za-z0-9_.]+)\s*$`)

func modulePathFromSource(src []byte) string {
	m := moduleDeclRE.FindSubmatch(src)
	if len(m) != 2 {
		return ""
	}
	return string(m[1])
}

func moduleRelPath(module string) string {
	return moduleRelPathWithExtension(module, compiler.T4SourceExtension)
}

func moduleRootFromEntry(entryPath string, module string) (string, error) {
	absEntry, err := filepath.Abs(entryPath)
	if err != nil {
		return "", fmt.Errorf("resolve module root: %w", err)
	}
	root := absEntry
	for range strings.Split(module, ".") {
		root = filepath.Dir(root)
	}
	rel, err := filepath.Rel(root, absEntry)
	if err != nil {
		return "", fmt.Errorf("resolve module root: %w", err)
	}
	if !cliModuleRelPathMatches(module, rel) {
		return "", fmt.Errorf("%s: module '%s' must be in %s (or legacy %s)", absEntry, module, moduleRelPathWithExtension(module, compiler.T4SourceExtension), moduleRelPathWithExtension(module, compiler.LegacyTetraSourceExtension))
	}
	return root, nil
}

func defaultInputPath() string {
	if fileExists(compiler.DefaultSourceFileName) {
		return compiler.DefaultSourceFileName
	}
	if fileExists(compiler.LegacySourceFileName) {
		return compiler.LegacySourceFileName
	}
	return compiler.DefaultSourceFileName
}

func moduleRelPathWithExtension(module string, extension string) string {
	return filepath.FromSlash(strings.ReplaceAll(module, ".", "/") + extension)
}

func cliModuleRelPathMatches(module, rel string) bool {
	cleanRel := filepath.Clean(rel)
	for _, ext := range compiler.SourceExtensions() {
		if cleanRel == filepath.Clean(moduleRelPathWithExtension(module, ext)) {
			return true
		}
	}
	return false
}

func rewriteModuleDecl(src []byte, module string) []byte {
	return []byte(moduleDeclRE.ReplaceAllString(string(src), "module "+module))
}

func runnerSourcePathForModuleFile(entryPath string, src []byte, runnerIndex int) (string, []byte, error) {
	module := modulePathFromSource(src)
	if module == "" {
		return "", nil, fmt.Errorf("runner source has imports but no module declaration")
	}
	root, err := moduleRootFromEntry(entryPath, module)
	if err != nil {
		return "", nil, err
	}
	parts := strings.Split(module, ".")
	parts[len(parts)-1] = fmt.Sprintf("__tetra_test_runner_%d", runnerIndex)
	runnerModule := strings.Join(parts, ".")
	runnerPath := filepath.Join(root, moduleRelPath(runnerModule))
	return runnerPath, rewriteModuleDecl(src, runnerModule), nil
}

func writeDiagnostic(w io.Writer, mode string, err error) {
	if mode == "json" {
		writeDiagnosticObject(w, compiler.DiagnosticFromError(err))
		return
	}
	fmt.Fprintln(w, err)
}

func writeDiagnosticWithHint(w io.Writer, mode string, message string, hint string) {
	if mode == "json" {
		writeDiagnosticObject(w, compiler.Diagnostic{
			Code:     compiler.DiagnosticCodeParse,
			Message:  message,
			Severity: "error",
			Hint:     hint,
		})
		return
	}
	fmt.Fprintln(w, message)
}

func parseBuildTargetOrReport(rawTarget string, diagnosticsMode string, stderr io.Writer) (ctarget.Target, bool) {
	tgt, err := ctarget.Parse(rawTarget)
	if err == nil {
		return tgt, true
	}
	if targetErr, ok := err.(ctarget.UnsupportedTargetError); ok {
		msg := fmt.Sprintf(
			"unsupported target: %s; supported targets: %s; build-only targets: %s",
			targetErr.Triple,
			strings.Join(ctarget.SupportedTriples(), ", "),
			strings.Join(ctarget.BuildOnlyTriples(), ", "),
		)
		writeDiagnosticWithHint(stderr, diagnosticsMode, msg, "run `tetra targets` to list valid targets")
		return ctarget.Target{}, false
	}
	writeDiagnostic(stderr, diagnosticsMode, err)
	return ctarget.Target{}, false
}

func writeValidationDiagnostic(w io.Writer, mode string, message string) {
	writeDiagnostic(w, mode, fmt.Errorf("%s", message))
}

func validateDiagnosticsMode(w io.Writer, mode string) bool {
	if mode == "text" || mode == "json" {
		return true
	}
	fmt.Fprintln(w, "unsupported --diagnostics format")
	return false
}

func writeDiagnosticObject(w io.Writer, diag compiler.Diagnostic) {
	raw, err := json.Marshal(diag)
	if err == nil {
		fmt.Fprintln(w, string(raw))
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: tetra <version|targets|formats|doctor|project|workspace|new|check|build|run|smoke|fmt|test|doc|interface|clean|eco|lsp> [options]")
}
