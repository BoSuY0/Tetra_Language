package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"tetra_language/compiler"
	ctarget "tetra_language/compiler/target"
)

func runWorkspace(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: tetra workspace <init|add|remove|list|check|graph|sync|build|test|run> [options]")
		return 2
	}
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra workspace <init|add|remove|list|check|graph|sync|build|test|run> [options]")
		return 0
	}
	switch args[0] {
	case "init":
		return runWorkspaceInit(args[1:], stdout, stderr)
	case "add":
		return runWorkspaceAdd(args[1:], stdout, stderr)
	case "remove":
		return runWorkspaceRemove(args[1:], stdout, stderr)
	case "list":
		return runWorkspaceList(args[1:], stdout, stderr)
	case "check":
		return runWorkspaceCheck(args[1:], stdout, stderr)
	case "graph":
		return runWorkspaceGraph(args[1:], stdout, stderr)
	case "sync":
		return runWorkspaceSync(args[1:], stdout, stderr)
	case "build":
		return runWorkspaceBuild(args[1:], stdout, stderr)
	case "test":
		return runWorkspaceTest(args[1:], stdout, stderr)
	case "run":
		return runWorkspaceRun(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown workspace command %q\n", args[0])
		return 2
	}
}

func runWorkspaceInit(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) > 1 {
		fmt.Fprintln(stderr, "workspace init accepts at most one path")
		return 2
	}
	root := "."
	if len(args) == 1 {
		root = args[0]
	}
	absRoot, err := filepath.Abs(filepath.FromSlash(root))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.MkdirAll(absRoot, 0o755); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	path := filepath.Join(absRoot, workspaceFileName)
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(stderr, "%s already exists\n", path)
		return 2
	} else if !os.IsNotExist(err) {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.WriteFile(path, []byte(fmt.Sprintf("workspace %q\n", workspaceSchemaV1)), 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Created workspace: %s\n", path)
	return 0
}

func runWorkspaceAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	memberPath, workspaceStart, code, err := parseWorkspaceMemberMutationArgs("workspace add", args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return code
	}
	workspace, err := loadWorkspace(workspaceStart)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	memberRel, err := normalizeWorkspaceMemberArg(workspace.Root, memberPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	memberRoot := filepath.Join(workspace.Root, filepath.FromSlash(memberRel))
	if _, err := findCapsulePath(memberRoot); err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", memberRel, err)
		return 1
	}
	for _, existing := range workspace.Members {
		if existing == memberRel {
			fmt.Fprintf(stderr, "duplicate workspace member %s\n", memberRel)
			return 1
		}
	}
	workspace.Members = append(workspace.Members, memberRel)
	if err := writeWorkspaceManifest(workspace); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Added workspace member: %s\n", memberRel)
	return 0
}

func runWorkspaceRemove(args []string, stdout io.Writer, stderr io.Writer) int {
	memberPath, workspaceStart, code, err := parseWorkspaceMemberMutationArgs("workspace remove", args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return code
	}
	workspace, err := loadWorkspace(workspaceStart)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	memberRel, err := normalizeWorkspaceMemberArg(workspace.Root, memberPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	var out []string
	removed := false
	for _, member := range workspace.Members {
		if member == memberRel {
			removed = true
			continue
		}
		out = append(out, member)
	}
	if !removed {
		fmt.Fprintf(stderr, "workspace member not found: %s\n", memberRel)
		return 1
	}
	workspace.Members = out
	if err := writeWorkspaceManifest(workspace); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Removed workspace member: %s\n", memberRel)
	return 0
}

func runWorkspaceList(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("workspace list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "workspace list accepts at most one path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	workspace, err := loadWorkspace(start)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	report := workspaceReport{
		Root:          workspace.Root,
		WorkspacePath: workspace.Path,
		Members:       describeWorkspaceMembers(workspace),
	}
	switch *format {
	case "text", "":
		writeWorkspaceListText(stdout, report)
		return 0
	case "json":
		return encodeWorkspaceJSON(stdout, stderr, report, false)
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func runWorkspaceCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("workspace check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "workspace check accepts at most one path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	graph, err := buildWorkspaceGraph(start)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	report := workspaceReport{
		Root:          graph.Workspace.Root,
		WorkspacePath: graph.Workspace.Path,
		Members:       graph.Nodes,
		Status:        "pass",
	}
	if len(graph.Issues) > 0 {
		report.Status = "fail"
	}
	switch *format {
	case "text", "":
		if report.Status == "pass" {
			fmt.Fprintf(stdout, "Workspace OK: %d member(s)\n", len(report.Members))
			return 0
		}
		writeWorkspaceIssues(stderr, graph.Issues)
		return 1
	case "json":
		code := 0
		if report.Status != "pass" {
			code = 1
		}
		return encodeWorkspaceJSON(stdout, stderr, report, code != 0)
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func runWorkspaceGraph(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("workspace graph", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "workspace graph accepts at most one path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	graph, err := buildWorkspaceGraph(start)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	status := "pass"
	if len(graph.Issues) > 0 {
		status = "fail"
	}
	report := workspaceGraphReport{
		Status:        status,
		Root:          graph.Workspace.Root,
		WorkspacePath: graph.Workspace.Path,
		Nodes:         graph.Nodes,
		Edges:         graph.Edges,
	}
	switch *format {
	case "text", "":
		writeWorkspaceGraphText(stdout, report)
		if status != "pass" {
			return 1
		}
		return 0
	case "json":
		code := 0
		if status != "pass" {
			code = 1
		}
		return encodeWorkspaceJSON(stdout, stderr, report, code != 0)
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func runWorkspaceSync(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("workspace sync", flag.ContinueOnError)
	fs.SetOutput(stderr)
	targetFlag := fs.String("target", "", "native target triple for generated .tobj artifacts")
	checkOnly := fs.Bool("check", false, "dry-run and report pending project lock/artifact changes without writing files")
	allTargets := fs.Bool("all-targets", false, "sync artifacts for every native target listed in member Capsule.t4")
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *targetFlag != "" && *allTargets {
		fmt.Fprintln(stderr, "workspace sync accepts either --target or --all-targets, not both")
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "workspace sync accepts at most one path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	graph, err := buildWorkspaceGraph(start)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if len(graph.Issues) > 0 {
		writeWorkspaceIssues(stderr, graph.Issues)
		return 1
	}
	order, err := workspaceSyncOrder(graph)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	for _, member := range order {
		ctx := workspaceProjectContext(member)
		lockPath := filepath.Join(ctx.Root, compiler.SemanticLockFileName)
		if *checkOnly {
			issues, err := checkProjectSync(ctx, *targetFlag, lockPath, *allTargets)
			if err != nil {
				fmt.Fprintf(stderr, "%s: %v\n", member.Path, err)
				return 1
			}
			for i := range issues {
				issues[i].Repair = "tetra workspace sync " + filepath.ToSlash(graph.Workspace.Root)
			}
			if len(issues) > 0 {
				writeArtifactIssues(stdout, issues, true)
				return 1
			}
			continue
		}
		if err := syncWorkspaceProject(ctx, *targetFlag, *allTargets, *jobs); err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", member.Path, err)
			return 1
		}
		fmt.Fprintf(stdout, "Workspace synced: %s\n", member.Path)
	}
	if *checkOnly {
		fmt.Fprintf(stdout, "Workspace current: %s\n", graph.Workspace.Root)
	}
	return 0
}

func runWorkspaceBuild(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("workspace build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	targetFlag := fs.String("target", "", "target triple ("+supportedTargetsHelp+"); defaults to each member Capsule.t4 first target, then host")
	allTargets := fs.Bool("all-targets", false, "build every target listed in each member Capsule.t4")
	interfaceOnly := fs.Bool("interface-only", false, "type-check interface/API graph without emitting executable code")
	islandsDebug := fs.Bool("islands-debug", false, "enable islands debug runtime checks")
	emit := fs.String("emit", "exe", "emit mode: exe, object, or library")
	runtimeMode := fs.String("runtime", "auto", "actors runtime: auto, selfhost, or builtin")
	runtimeObject := fs.String("runtime-object", "", "actors runtime object override")
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	artifactsMode := fs.String("artifacts", "strict", "artifact handling: strict or auto")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text, json, or toon")
	format := fs.String("format", "text", "workspace report format: text or json")
	outDir := ""
	fs.StringVar(&outDir, "o", "", "workspace output directory")
	fs.StringVar(&outDir, "out-dir", "", "workspace output directory")
	var linkObjects multiFlag
	fs.Var(&linkObjects, "link-object", "extra TOBJ object to link")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateWorkspaceReportFormat(stderr, *format) {
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	if *targetFlag != "" && *allTargets {
		writeValidationDiagnostic(stderr, *diagnostics, "workspace build accepts either --target or --all-targets, not both")
		return 2
	}
	if *artifactsMode != "strict" && *artifactsMode != "auto" {
		writeValidationDiagnostic(stderr, *diagnostics, "workspace build --artifacts must be strict or auto")
		return 2
	}
	if fs.NArg() > 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "workspace build accepts at most one path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	graph, err := buildWorkspaceGraph(start)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if len(graph.Issues) > 0 {
		writeWorkspaceIssues(stderr, graph.Issues)
		return 1
	}
	order, err := workspaceSyncOrder(graph)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	report := workspaceExecutionReport{
		WorkspaceRoot: graph.Workspace.Root,
		Command:       "build",
		Target:        workspaceExecutionTargetLabel(*targetFlag, *allTargets),
		Total:         len(order),
	}
	statusByPath := map[string]string{}
	for _, member := range order {
		if dep, blocked := workspaceBlockedDependency(member.Path, graph, statusByPath); blocked {
			item := workspaceExecutionItem(member, "skipped", fmt.Sprintf("blocked by failed dependency %s", dep), 0, false)
			appendWorkspaceExecutionMember(&report, item)
			statusByPath[member.Path] = item.Status
			continue
		}
		memberArgs, err := workspaceBuildMemberArgs(member, workspaceBuildOptions{
			Target:        *targetFlag,
			AllTargets:    *allTargets,
			InterfaceOnly: *interfaceOnly,
			IslandsDebug:  *islandsDebug,
			Emit:          *emit,
			RuntimeMode:   *runtimeMode,
			RuntimeObject: *runtimeObject,
			Jobs:          *jobs,
			ArtifactsMode: *artifactsMode,
			Diagnostics:   *diagnostics,
			OutDir:        outDir,
			LinkObjects:   []string(linkObjects),
		})
		if err != nil {
			item := workspaceExecutionItem(member, "fail", err.Error(), 2, true)
			appendWorkspaceExecutionMember(&report, item)
			statusByPath[member.Path] = item.Status
			continue
		}
		var memberStdout, memberStderr bytes.Buffer
		code := runBuild(memberArgs, &memberStdout, &memberStderr)
		status := "pass"
		if code != 0 {
			status = "fail"
		}
		item := workspaceExecutionItem(member, status, workspaceCommandDetail(memberStdout, memberStderr), code, true)
		appendWorkspaceExecutionMember(&report, item)
		statusByPath[member.Path] = item.Status
	}
	if err := writeWorkspaceExecutionReport(stdout, report, *format); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return workspaceExecutionExitCode(report)
}

type workspaceBuildOptions struct {
	Target        string
	AllTargets    bool
	InterfaceOnly bool
	IslandsDebug  bool
	Emit          string
	RuntimeMode   string
	RuntimeObject string
	Jobs          int
	ArtifactsMode string
	Diagnostics   string
	OutDir        string
	LinkObjects   []string
}

func workspaceBuildMemberArgs(member workspaceMemberReport, opt workspaceBuildOptions) ([]string, error) {
	args := []string{}
	if opt.Target != "" {
		args = append(args, "--target", opt.Target)
	}
	if opt.AllTargets {
		args = append(args, "--all-targets")
	}
	if opt.InterfaceOnly {
		args = append(args, "--interface-only")
	}
	if opt.IslandsDebug {
		args = append(args, "--islands-debug")
	}
	if opt.Emit != "" {
		args = append(args, "--emit", opt.Emit)
	}
	if opt.RuntimeMode != "" {
		args = append(args, "--runtime", opt.RuntimeMode)
	}
	if opt.RuntimeObject != "" {
		args = append(args, "--runtime-object", opt.RuntimeObject)
	}
	args = append(args, "--jobs", strconv.Itoa(opt.Jobs))
	if opt.ArtifactsMode != "" {
		args = append(args, "--artifacts", opt.ArtifactsMode)
	}
	if opt.Diagnostics != "" {
		args = append(args, "--diagnostics", opt.Diagnostics)
	}
	for _, path := range opt.LinkObjects {
		args = append(args, "--link-object", path)
	}
	output, err := workspaceBuildOutputPath(member, opt)
	if err != nil {
		return nil, err
	}
	if output != "" {
		args = append(args, "-o", output)
	}
	args = append(args, member.ResolvedPath)
	return args, nil
}

func workspaceBuildOutputPath(member workspaceMemberReport, opt workspaceBuildOptions) (string, error) {
	base := member.ResolvedPath
	if opt.OutDir != "" {
		base = filepath.Join(opt.OutDir, filepath.FromSlash(member.Path))
	}
	if opt.AllTargets {
		if err := os.MkdirAll(base, 0o755); err != nil {
			return "", err
		}
		return base, nil
	}
	rawTarget := opt.Target
	if rawTarget == "" {
		rawTarget = projectDefaultTarget(workspaceProjectContext(member))
	}
	tgt, err := ctarget.Parse(rawTarget)
	if err != nil {
		return "", err
	}
	output := filepath.Join(base, defaultOutput(tgt, opt.Emit))
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return "", err
	}
	return output, nil
}

func runWorkspaceTest(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("workspace test", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", defaultTarget(), "target triple ("+supportedTargetsHelp+")")
	failFast := fs.Bool("fail-fast", false, "stop after the first failed member")
	format := fs.String("format", "text", "workspace report format: text or json")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text, json, or toon")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateWorkspaceReportFormat(stderr, *format) {
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	if fs.NArg() > 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "workspace test accepts at most one path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	graph, err := buildWorkspaceGraph(start)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if len(graph.Issues) > 0 {
		writeWorkspaceIssues(stderr, graph.Issues)
		return 1
	}
	order, err := workspaceSyncOrder(graph)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	report := workspaceExecutionReport{
		WorkspaceRoot: graph.Workspace.Root,
		Command:       "test",
		Target:        *target,
		Total:         len(order),
	}
	statusByPath := map[string]string{}
	failFastAfter := ""
	for _, member := range order {
		if failFastAfter != "" {
			item := workspaceExecutionItem(member, "skipped", fmt.Sprintf("fail-fast after %s", failFastAfter), 0, false)
			appendWorkspaceExecutionMember(&report, item)
			statusByPath[member.Path] = item.Status
			continue
		}
		if dep, blocked := workspaceBlockedDependency(member.Path, graph, statusByPath); blocked {
			item := workspaceExecutionItem(member, "skipped", fmt.Sprintf("blocked by failed dependency %s", dep), 0, false)
			appendWorkspaceExecutionMember(&report, item)
			statusByPath[member.Path] = item.Status
			continue
		}
		memberArgs := []string{"--target", *target, "--diagnostics", *diagnostics, "--report", "text", member.ResolvedPath}
		var memberStdout, memberStderr bytes.Buffer
		code := runTest(memberArgs, &memberStdout, &memberStderr)
		status := "pass"
		if code != 0 {
			status = "fail"
		}
		item := workspaceExecutionItem(member, status, workspaceCommandDetail(memberStdout, memberStderr), code, true)
		appendWorkspaceExecutionMember(&report, item)
		statusByPath[member.Path] = item.Status
		if status == "fail" && *failFast {
			failFastAfter = member.Path
		}
	}
	if err := writeWorkspaceExecutionReport(stdout, report, *format); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return workspaceExecutionExitCode(report)
}

type workspaceRunOptions struct {
	Member         string
	WorkspaceStart string
	Target         string
	Output         string
	IslandsDebug   bool
	RuntimeMode    string
	RuntimeObject  string
	Jobs           int
	ArtifactsMode  string
	Diagnostics    string
}

func runWorkspaceRun(args []string, stdout io.Writer, stderr io.Writer) int {
	opt, code, err := parseWorkspaceRunArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return code
	}
	if !validateDiagnosticsMode(stderr, opt.Diagnostics) {
		return 2
	}
	if opt.ArtifactsMode != "strict" && opt.ArtifactsMode != "auto" {
		writeValidationDiagnostic(stderr, opt.Diagnostics, "workspace run --artifacts must be strict or auto")
		return 2
	}
	graph, err := buildWorkspaceGraph(opt.WorkspaceStart)
	if err != nil {
		writeDiagnostic(stderr, opt.Diagnostics, err)
		return 1
	}
	if len(graph.Issues) > 0 {
		writeWorkspaceIssues(stderr, graph.Issues)
		return 1
	}
	memberRel, err := normalizeWorkspaceMemberArg(graph.Workspace.Root, opt.Member)
	if err != nil {
		writeValidationDiagnostic(stderr, opt.Diagnostics, err.Error())
		return 2
	}
	member, ok := workspaceFindMember(graph, memberRel)
	if !ok {
		writeValidationDiagnostic(stderr, opt.Diagnostics, "workspace member not found: "+memberRel)
		return 2
	}
	if member.Status != "ok" {
		writeDiagnostic(stderr, opt.Diagnostics, fmt.Errorf("%s: %s", member.Path, member.Detail))
		return 1
	}
	if opt.ArtifactsMode == "auto" {
		if err := syncWorkspaceProject(workspaceProjectContext(member), opt.Target, false, opt.Jobs); err != nil {
			writeDiagnostic(stderr, opt.Diagnostics, err)
			return 1
		}
	}
	runArgs := workspaceRunMemberArgs(member, opt)
	return runRun(runArgs, stdout, stderr)
}

func parseWorkspaceRunArgs(args []string) (workspaceRunOptions, int, error) {
	opt := workspaceRunOptions{
		WorkspaceStart: ".",
		RuntimeMode:    "auto",
		Jobs:           1,
		ArtifactsMode:  "strict",
		Diagnostics:    "text",
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--workspace":
			value, next, err := workspaceRunValue(args, i, arg)
			if err != nil {
				return opt, 2, err
			}
			opt.WorkspaceStart = value
			i = next
		case strings.HasPrefix(arg, "--workspace="):
			opt.WorkspaceStart = strings.TrimPrefix(arg, "--workspace=")
		case arg == "--target":
			value, next, err := workspaceRunValue(args, i, arg)
			if err != nil {
				return opt, 2, err
			}
			opt.Target = value
			i = next
		case strings.HasPrefix(arg, "--target="):
			opt.Target = strings.TrimPrefix(arg, "--target=")
		case arg == "-o":
			value, next, err := workspaceRunValue(args, i, arg)
			if err != nil {
				return opt, 2, err
			}
			opt.Output = value
			i = next
		case strings.HasPrefix(arg, "-o="):
			opt.Output = strings.TrimPrefix(arg, "-o=")
		case arg == "--islands-debug":
			opt.IslandsDebug = true
		case arg == "--runtime":
			value, next, err := workspaceRunValue(args, i, arg)
			if err != nil {
				return opt, 2, err
			}
			opt.RuntimeMode = value
			i = next
		case strings.HasPrefix(arg, "--runtime="):
			opt.RuntimeMode = strings.TrimPrefix(arg, "--runtime=")
		case arg == "--runtime-object":
			value, next, err := workspaceRunValue(args, i, arg)
			if err != nil {
				return opt, 2, err
			}
			opt.RuntimeObject = value
			i = next
		case strings.HasPrefix(arg, "--runtime-object="):
			opt.RuntimeObject = strings.TrimPrefix(arg, "--runtime-object=")
		case arg == "--jobs":
			value, next, err := workspaceRunValue(args, i, arg)
			if err != nil {
				return opt, 2, err
			}
			jobs, err := strconv.Atoi(value)
			if err != nil {
				return opt, 2, fmt.Errorf("workspace run --jobs must be an integer")
			}
			opt.Jobs = jobs
			i = next
		case strings.HasPrefix(arg, "--jobs="):
			jobs, err := strconv.Atoi(strings.TrimPrefix(arg, "--jobs="))
			if err != nil {
				return opt, 2, fmt.Errorf("workspace run --jobs must be an integer")
			}
			opt.Jobs = jobs
		case arg == "--artifacts":
			value, next, err := workspaceRunValue(args, i, arg)
			if err != nil {
				return opt, 2, err
			}
			opt.ArtifactsMode = value
			i = next
		case strings.HasPrefix(arg, "--artifacts="):
			opt.ArtifactsMode = strings.TrimPrefix(arg, "--artifacts=")
		case arg == "--diagnostics":
			value, next, err := workspaceRunValue(args, i, arg)
			if err != nil {
				return opt, 2, err
			}
			opt.Diagnostics = value
			i = next
		case strings.HasPrefix(arg, "--diagnostics="):
			opt.Diagnostics = strings.TrimPrefix(arg, "--diagnostics=")
		case strings.HasPrefix(arg, "-"):
			return opt, 2, fmt.Errorf("unknown workspace run option %q", arg)
		default:
			if opt.Member != "" {
				return opt, 2, fmt.Errorf("workspace run accepts exactly one member path")
			}
			opt.Member = arg
		}
	}
	if opt.Member == "" {
		return opt, 2, fmt.Errorf("workspace run requires a member path")
	}
	return opt, 0, nil
}

func workspaceRunValue(args []string, index int, flagName string) (string, int, error) {
	if index+1 >= len(args) {
		return "", index, fmt.Errorf("workspace run requires %s value", flagName)
	}
	return args[index+1], index + 1, nil
}

func workspaceRunMemberArgs(member workspaceMemberReport, opt workspaceRunOptions) []string {
	args := []string{}
	if opt.Target != "" {
		args = append(args, "--target", opt.Target)
	}
	if opt.Output != "" {
		args = append(args, "-o", opt.Output)
	}
	if opt.IslandsDebug {
		args = append(args, "--islands-debug")
	}
	if opt.RuntimeMode != "" {
		args = append(args, "--runtime", opt.RuntimeMode)
	}
	if opt.RuntimeObject != "" {
		args = append(args, "--runtime-object", opt.RuntimeObject)
	}
	args = append(args, "--jobs", strconv.Itoa(opt.Jobs), "--diagnostics", opt.Diagnostics, member.ResolvedPath)
	return args
}

func validateWorkspaceReportFormat(stderr io.Writer, format string) bool {
	if format == "text" || format == "json" {
		return true
	}
	fmt.Fprintln(stderr, "workspace --format must be text or json")
	return false
}

func workspaceExecutionTargetLabel(target string, allTargets bool) string {
	if allTargets {
		return "all-targets"
	}
	return target
}

func workspaceBlockedDependency(path string, graph workspaceGraph, statusByPath map[string]string) (string, bool) {
	for _, edge := range graph.Edges {
		if edge.From != path {
			continue
		}
		status := statusByPath[edge.To]
		if status == "fail" || status == "skipped" {
			return edge.To, true
		}
	}
	return "", false
}

func workspaceFindMember(graph workspaceGraph, memberPath string) (workspaceMemberReport, bool) {
	for _, member := range graph.Nodes {
		if member.Path == memberPath {
			return member, true
		}
	}
	return workspaceMemberReport{}, false
}

func workspaceExecutionItem(member workspaceMemberReport, status string, detail string, exitCode int, includeExitCode bool) workspaceExecutionMemberReport {
	item := workspaceExecutionMemberReport{
		Path:      member.Path,
		CapsuleID: member.CapsuleID,
		Status:    status,
		Detail:    strings.TrimSpace(detail),
	}
	if includeExitCode {
		code := exitCode
		item.ExitCode = &code
	}
	return item
}

func appendWorkspaceExecutionMember(report *workspaceExecutionReport, item workspaceExecutionMemberReport) {
	report.Members = append(report.Members, item)
	switch item.Status {
	case "pass":
		report.Passed++
	case "fail":
		report.Failed++
	case "skipped":
		report.Skipped++
	}
}

func workspaceCommandDetail(stdout bytes.Buffer, stderr bytes.Buffer) string {
	if detail := strings.TrimSpace(stderr.String()); detail != "" {
		return detail
	}
	return strings.TrimSpace(stdout.String())
}

func writeWorkspaceExecutionReport(w io.Writer, report workspaceExecutionReport, format string) error {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	default:
		for _, member := range report.Members {
			label := strings.ToUpper(member.Status)
			if member.Detail != "" && member.Status != "pass" {
				fmt.Fprintf(w, "%s %s: %s\n", label, member.Path, firstWorkspaceDetailLine(member.Detail))
			} else {
				fmt.Fprintf(w, "%s %s\n", label, member.Path)
			}
		}
		fmt.Fprintf(w, "Workspace %s: %d/%d passed, %d failed, %d skipped\n", report.Command, report.Passed, report.Total, report.Failed, report.Skipped)
		return nil
	}
}

func firstWorkspaceDetailLine(detail string) string {
	for _, line := range strings.Split(detail, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func workspaceExecutionExitCode(report workspaceExecutionReport) int {
	if report.Failed == 0 && report.Skipped == 0 {
		return 0
	}
	for _, member := range report.Members {
		if member.Status == "fail" && member.ExitCode != nil && *member.ExitCode == 2 {
			return 2
		}
	}
	return 1
}

func parseWorkspaceMemberMutationArgs(command string, args []string) (string, string, int, error) {
	workspaceStart := "."
	var member string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--workspace" {
			if i+1 >= len(args) {
				return "", "", 2, fmt.Errorf("%s requires --workspace value", command)
			}
			workspaceStart = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, "--workspace=") {
			workspaceStart = strings.TrimPrefix(arg, "--workspace=")
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return "", "", 2, fmt.Errorf("unknown %s option %q", command, arg)
		}
		if member != "" {
			return "", "", 2, fmt.Errorf("%s accepts exactly one member path", command)
		}
		member = arg
	}
	if member == "" {
		return "", "", 2, fmt.Errorf("%s requires a member path", command)
	}
	return member, workspaceStart, 0, nil
}

func loadWorkspace(start string) (workspaceManifest, error) {
	path, root, err := findWorkspacePath(start)
	if err != nil {
		return workspaceManifest{}, err
	}
	return parseWorkspace(path, root)
}

func findWorkspacePath(start string) (string, string, error) {
	if strings.TrimSpace(start) == "" {
		start = "."
	}
	abs, err := filepath.Abs(filepath.FromSlash(start))
	if err != nil {
		return "", "", err
	}
	info, err := os.Stat(abs)
	if err == nil && !info.IsDir() && filepath.Base(abs) == workspaceFileName {
		return abs, filepath.Dir(abs), nil
	}
	if err != nil && !os.IsNotExist(err) {
		return "", "", err
	}
	dir := abs
	if err == nil && !info.IsDir() {
		dir = filepath.Dir(abs)
	}
	for {
		path := filepath.Join(dir, workspaceFileName)
		if _, err := os.Stat(path); err == nil {
			return path, dir, nil
		} else if !os.IsNotExist(err) {
			return "", "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", fmt.Errorf("%s not found", workspaceFileName)
		}
		dir = parent
	}
}

func parseWorkspace(path string, root string) (workspaceManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return workspaceManifest{}, err
	}
	workspace := workspaceManifest{Path: path, Root: root, Schema: workspaceSchemaV1}
	seenMembers := map[string]struct{}{}
	sawWorkspace := false
	for i, line := range strings.Split(string(raw), "\n") {
		content := strings.TrimSpace(line)
		if content == "" || strings.HasPrefix(content, "#") || strings.HasPrefix(content, "//") {
			continue
		}
		if strings.HasPrefix(content, "workspace ") {
			if sawWorkspace {
				return workspaceManifest{}, fmt.Errorf("%s:%d: duplicate workspace declaration", path, i+1)
			}
			value, err := parseCapsuleString(path, i+1, strings.TrimSpace(strings.TrimPrefix(content, "workspace ")))
			if err != nil {
				return workspaceManifest{}, err
			}
			if value != workspaceSchemaV1 {
				return workspaceManifest{}, fmt.Errorf("%s:%d: unsupported workspace schema %s", path, i+1, value)
			}
			workspace.Schema = value
			sawWorkspace = true
			continue
		}
		if strings.HasPrefix(content, "member ") {
			value, err := parseCapsuleBareOrQuoted(path, i+1, strings.TrimSpace(strings.TrimPrefix(content, "member ")))
			if err != nil {
				return workspaceManifest{}, err
			}
			member, err := cleanWorkspaceMemberPath(value)
			if err != nil {
				return workspaceManifest{}, fmt.Errorf("%s:%d: %v", path, i+1, err)
			}
			if _, ok := seenMembers[member]; ok {
				return workspaceManifest{}, fmt.Errorf("%s:%d: duplicate workspace member %s", path, i+1, member)
			}
			seenMembers[member] = struct{}{}
			workspace.Members = append(workspace.Members, member)
			continue
		}
		return workspaceManifest{}, fmt.Errorf("%s:%d: unknown workspace field", path, i+1)
	}
	if !sawWorkspace {
		return workspaceManifest{}, fmt.Errorf("%s: missing workspace declaration", path)
	}
	return workspace, nil
}

func writeWorkspaceManifest(workspace workspaceManifest) error {
	lines := []string{fmt.Sprintf("workspace %q", workspace.Schema)}
	for _, member := range workspace.Members {
		lines = append(lines, fmt.Sprintf("member %q", member))
	}
	lines = append(lines, "")
	return os.WriteFile(workspace.Path, []byte(strings.Join(lines, "\n")), 0o644)
}

func normalizeWorkspaceMemberArg(root string, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("workspace member path is required")
	}
	path := filepath.FromSlash(value)
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", err
	}
	return cleanWorkspaceMemberPath(filepath.ToSlash(rel))
}

func cleanWorkspaceMemberPath(value string) (string, error) {
	if strings.Contains(value, "\\") {
		return "", fmt.Errorf("member path must use forward slashes")
	}
	if filepath.IsAbs(value) {
		return "", fmt.Errorf("member path must be relative")
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if clean == "." || clean == "" {
		return "", fmt.Errorf("member path must not be empty")
	}
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("member path must stay inside workspace root")
	}
	if strings.ContainsAny(clean, "\r\n\t") {
		return "", fmt.Errorf("member path must not contain control whitespace")
	}
	return clean, nil
}

func buildWorkspaceGraph(start string) (workspaceGraph, error) {
	workspace, err := loadWorkspace(start)
	if err != nil {
		return workspaceGraph{}, err
	}
	graph := workspaceGraph{Workspace: workspace, ByRoot: map[string]workspaceMemberReport{}}
	graph.Nodes = describeWorkspaceMembers(workspace)
	var issues []workspaceMemberReport
	byID := map[string]workspaceMemberReport{}
	for _, node := range graph.Nodes {
		if node.Status != "ok" {
			issues = append(issues, node)
			continue
		}
		rootKey := filepath.Clean(node.ResolvedPath)
		graph.ByRoot[rootKey] = node
		if prev, ok := byID[node.CapsuleID]; ok {
			issues = append(issues, workspaceMemberReport{
				Path:   node.Path,
				Status: "fail",
				Detail: fmt.Sprintf("duplicate capsule id %s (also %s)", node.CapsuleID, prev.Path),
			})
		} else {
			byID[node.CapsuleID] = node
		}
	}
	for _, node := range graph.Nodes {
		if node.Status != "ok" {
			continue
		}
		manifest, err := parseCapsule(node.CapsulePath)
		if err != nil {
			issues = append(issues, workspaceMemberReport{Path: node.Path, Status: "invalid", Detail: err.Error()})
			continue
		}
		for _, dep := range manifest.Dependencies {
			if dep.Path == "" {
				continue
			}
			depRoot, err := resolveDependencyProjectRoot(node.ResolvedPath, dep.Path)
			if err != nil {
				issues = append(issues, workspaceMemberReport{Path: node.Path, Status: "fail", Detail: fmt.Sprintf("%s: %v", dep.ID, err)})
				continue
			}
			depNode, ok := graph.ByRoot[filepath.Clean(depRoot)]
			if !ok {
				issues = append(issues, workspaceMemberReport{Path: node.Path, Status: "fail", Detail: fmt.Sprintf("dependency %s path %s is not a workspace member", dep.ID, dep.Path)})
				continue
			}
			graph.Edges = append(graph.Edges, workspaceGraphEdge{From: node.Path, To: depNode.Path, ID: dep.ID})
		}
		_, depManifests, err := projectDependencyGraph(node.ResolvedPath, manifest, map[string]int{node.ResolvedPath: projectDependencyVisiting}, []string{node.ResolvedPath})
		if err != nil {
			issues = append(issues, workspaceMemberReport{Path: node.Path, Status: "fail", Detail: err.Error()})
			continue
		}
		manifests := append([]capsuleManifest{manifest}, depManifests...)
		if err := validateCapsuleGraph(manifests, ""); err != nil {
			issues = append(issues, workspaceMemberReport{Path: node.Path, Status: "fail", Detail: err.Error()})
		}
	}
	sort.Slice(graph.Edges, func(i, j int) bool {
		if graph.Edges[i].From == graph.Edges[j].From {
			return graph.Edges[i].To < graph.Edges[j].To
		}
		return graph.Edges[i].From < graph.Edges[j].From
	})
	graph.Issues = dedupeWorkspaceIssues(issues)
	return graph, nil
}

func describeWorkspaceMembers(workspace workspaceManifest) []workspaceMemberReport {
	var out []workspaceMemberReport
	for _, member := range workspace.Members {
		out = append(out, describeWorkspaceMember(workspace.Root, member))
	}
	return out
}

func describeWorkspaceMember(root string, member string) workspaceMemberReport {
	item := workspaceMemberReport{
		Path:         member,
		ResolvedPath: filepath.Join(root, filepath.FromSlash(member)),
		Status:       "ok",
	}
	info, err := os.Stat(item.ResolvedPath)
	if err != nil {
		item.Status = "missing"
		item.Detail = err.Error()
		return item
	}
	if !info.IsDir() {
		item.Status = "invalid"
		item.Detail = "member path is not a directory"
		return item
	}
	capsulePath, err := findCapsulePath(item.ResolvedPath)
	if err != nil {
		item.Status = "invalid"
		item.Detail = err.Error()
		return item
	}
	item.CapsulePath = capsulePath
	manifest, err := parseCapsule(capsulePath)
	if err != nil {
		item.Status = "invalid"
		item.Detail = err.Error()
		return item
	}
	item.CapsuleID = manifest.ID
	item.Version = manifest.Version
	return item
}

func workspaceSyncOrder(graph workspaceGraph) ([]workspaceMemberReport, error) {
	byPath := map[string]workspaceMemberReport{}
	deps := map[string][]string{}
	for _, node := range graph.Nodes {
		if node.Status == "ok" {
			byPath[node.Path] = node
		}
	}
	for _, edge := range graph.Edges {
		deps[edge.From] = append(deps[edge.From], edge.To)
	}
	for path := range deps {
		sort.Strings(deps[path])
	}
	state := map[string]int{}
	var order []workspaceMemberReport
	var visit func(string) error
	visit = func(path string) error {
		switch state[path] {
		case projectDependencyVisiting:
			return fmt.Errorf("workspace dependency cycle at %s", path)
		case projectDependencyDone:
			return nil
		}
		node, ok := byPath[path]
		if !ok {
			return fmt.Errorf("workspace member %s not found", path)
		}
		state[path] = projectDependencyVisiting
		for _, dep := range deps[path] {
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[path] = projectDependencyDone
		order = append(order, node)
		return nil
	}
	for _, member := range graph.Workspace.Members {
		if err := visit(member); err != nil {
			return nil, err
		}
	}
	return order, nil
}

func workspaceProjectContext(member workspaceMemberReport) *cliProjectContext {
	manifest, _ := parseCapsule(member.CapsulePath)
	return &cliProjectContext{
		Found:       true,
		Root:        member.ResolvedPath,
		CapsulePath: member.CapsulePath,
		LockPath:    findProjectLock(member.ResolvedPath),
		Manifest:    manifest,
	}
}

func syncWorkspaceProject(ctx *cliProjectContext, targetFlag string, allTargets bool, jobs int) error {
	lockPath := filepath.Join(ctx.Root, compiler.SemanticLockFileName)
	useArtifactBuilder, err := projectSyncUsesArtifactBuilder(ctx.Manifest, targetFlag, allTargets)
	if err != nil {
		return err
	}
	if useArtifactBuilder {
		return buildCapsuleArtifacts(ctx.CapsulePath, capsuleArtifactBuildOptions{
			Target:     targetFlag,
			LockPath:   lockPath,
			Jobs:       jobs,
			AllTargets: allTargets,
		})
	}
	manifests, err := parseCapsuleGraphArgs([]string{ctx.CapsulePath})
	if err != nil {
		return err
	}
	if err := validateCapsuleGraph(manifests, targetFlag); err != nil {
		return err
	}
	return writeEcoLock(lockPath, manifests)
}

func writeWorkspaceListText(w io.Writer, report workspaceReport) {
	if len(report.Members) == 0 {
		fmt.Fprintln(w, "Workspace members: none")
		return
	}
	fmt.Fprintln(w, "Workspace members:")
	for _, member := range report.Members {
		fmt.Fprintf(w, "  %s %s", member.Path, member.Status)
		if member.CapsuleID != "" {
			fmt.Fprintf(w, " %s %s", member.CapsuleID, member.Version)
		}
		if member.Detail != "" {
			fmt.Fprintf(w, ": %s", member.Detail)
		}
		fmt.Fprintln(w)
	}
}

func writeWorkspaceGraphText(w io.Writer, report workspaceGraphReport) {
	fmt.Fprintf(w, "Workspace graph: %d node(s), %d edge(s)\n", len(report.Nodes), len(report.Edges))
	for _, edge := range report.Edges {
		fmt.Fprintf(w, "  %s -> %s (%s)\n", edge.From, edge.To, edge.ID)
	}
}

func writeWorkspaceIssues(w io.Writer, issues []workspaceMemberReport) {
	for _, issue := range issues {
		if issue.Detail != "" {
			fmt.Fprintf(w, "%s: %s\n", issue.Path, issue.Detail)
		} else {
			fmt.Fprintf(w, "%s: %s\n", issue.Path, issue.Status)
		}
	}
}

func encodeWorkspaceJSON(stdout io.Writer, stderr io.Writer, value any, fail bool) int {
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if fail {
		return 1
	}
	return 0
}

func dedupeWorkspaceIssues(issues []workspaceMemberReport) []workspaceMemberReport {
	seen := map[string]struct{}{}
	var out []workspaceMemberReport
	for _, issue := range issues {
		key := issue.Path + "\x00" + issue.Status + "\x00" + issue.Detail
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, issue)
	}
	return out
}
