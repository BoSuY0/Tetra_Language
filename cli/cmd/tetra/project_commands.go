package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tetra_language/compiler"
	ctarget "tetra_language/compiler/target"
)

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
	_, depManifest, relPath, err := resolveProjectDependencyAddPath(ctx.Root, *pathFlag)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
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
		_, depManifests, err := projectDependencyGraph(ctx.Root, ctx.Manifest, map[string]int{ctx.Root: projectDependencyVisiting}, []string{ctx.Root})
		if err != nil {
			status = "fail"
			issues = append(issues, projectDependencyReport{Status: "fail", Detail: err.Error()})
		} else {
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
	return resolveNativeCapsuleTargets(manifest, nativeTargetResolutionOptions{
		TargetFlag: targetFlag,
		AllTargets: allTargets,
	})
}

func isWASMTargetTriple(triple string) bool {
	for _, wasmTriple := range ctarget.WASMTriples() {
		if triple == wasmTriple {
			return true
		}
	}
	return false
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
