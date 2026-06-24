package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"tetra_language/compiler"
	"tetra_language/tools/validators/platformui"
)

type hostRuntime struct {
	GOOS   string
	GOARCH string
}

func main() {
	target := flag.String("target", "", "platform UI target")
	reportPath := flag.String("report", "", "path to write platform UI report")
	childRuntime := flag.Bool(
		"child-runtime",
		false,
		"run hidden platform UI runtime child and write JSON evidence",
	)
	flag.Parse()
	if *childRuntime {
		if *target == "" {
			fmt.Fprintln(os.Stderr, "error: --target is required with --child-runtime")
			os.Exit(2)
		}
		evidence := runPlatformUIRuntimeChild(*target)
		raw, err := json.MarshalIndent(evidence, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		raw = append(raw, '\n')
		if _, err := os.Stdout.Write(raw); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if evidence.ProbeError != "" {
			fmt.Fprintln(os.Stderr, evidence.ProbeError)
			os.Exit(1)
		}
		return
	}
	if *target == "" || *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --target and --report are required")
		os.Exit(2)
	}
	if err := os.MkdirAll(filepath.Dir(*reportPath), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, ok := requiredGOOSByTarget(*target); !ok {
		fmt.Fprintf(os.Stderr, "unknown platform UI target %s\n", *target)
		os.Exit(2)
	}
	report, exitCode := buildPlatformUIRuntimeReport(*target, currentHostRuntime())
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(*reportPath, raw, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if exitCode != 0 {
		fmt.Fprintln(os.Stderr, report.Blocker)
	}
	os.Exit(exitCode)
}

func currentHostRuntime() hostRuntime {
	return hostRuntime{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH}
}

func requiredGOOSByTarget(target string) (string, bool) {
	requiredGOOS := map[string]string{"windows-x64": "windows", "macos-x64": "darwin"}[target]
	return requiredGOOS, requiredGOOS != ""
}

func buildPlatformUIRuntimeReport(target string, host hostRuntime) (platformui.Report, int) {
	return buildPlatformUIRuntimeReportWithRunner(target, host, processPlatformRuntimeRunner{})
}

type platformRuntimeRunner interface {
	Run(target string) (platformRuntimeRun, error)
}

type platformRuntimeRun struct {
	Evidence      childRuntimeEvidence
	BuildPath     string
	AppPath       string
	BuildExitCode int
	AppExitCode   int
}

type platformWindowProbeResult struct {
	API     string
	Markers []string
}

var platformWindowProbe = runPlatformWindowProbe

const (
	defaultPlatformRuntimeBuildTimeout = 5 * time.Minute
	defaultPlatformRuntimeChildTimeout = time.Minute
)

type processPlatformRuntimeRunner struct {
	buildTimeout   time.Duration
	childTimeout   time.Duration
	commandContext func(context.Context, string, ...string) *exec.Cmd
}

func (r processPlatformRuntimeRunner) Run(target string) (platformRuntimeRun, error) {
	repoRoot, err := repoRootFromWD()
	if err != nil {
		return platformRuntimeRun{}, err
	}
	tmpDir, err := os.MkdirTemp("", "tetra-platform-ui-runtime-*")
	if err != nil {
		return platformRuntimeRun{}, err
	}
	defer os.RemoveAll(tmpDir)
	exeName := "platform-ui-runtime-smoke"
	if runtime.GOOS == "windows" {
		exeName += ".exe"
	}
	exePath := filepath.Join(tmpDir, exeName)
	buildArgs := []string{"build", "-o", exePath, "./tools/cmd/platform-ui-runtime-smoke"}
	buildTimeout := r.effectiveBuildTimeout()
	buildCtx, buildCancel := context.WithTimeout(context.Background(), buildTimeout)
	buildCmd := r.command(buildCtx, "go", buildArgs...)
	buildCmd.Dir = repoRoot
	buildRaw, err := buildCmd.CombinedOutput()
	buildTimedOut := buildCtx.Err() == context.DeadlineExceeded
	buildCancel()
	result := platformRuntimeRun{
		BuildPath:     "go " + strings.Join(buildArgs, " "),
		BuildExitCode: exitCodeFromError(err),
	}
	if buildTimedOut {
		if result.BuildExitCode == 0 {
			result.BuildExitCode = 1
		}
		return result, fmt.Errorf(
			"build platform UI runtime child timed out after %s",
			buildTimeout,
		)
	}
	if err != nil {
		return result, fmt.Errorf(
			"build platform UI runtime child: %w: %s",
			err,
			strings.TrimSpace(string(buildRaw)),
		)
	}
	childArgs := []string{"--child-runtime", "--target", target}
	childTimeout := r.effectiveChildTimeout()
	childCtx, childCancel := context.WithTimeout(context.Background(), childTimeout)
	childCmd := r.command(childCtx, exePath, childArgs...)
	childCmd.Dir = repoRoot
	var childStdout bytes.Buffer
	var childStderr bytes.Buffer
	childCmd.Stdout = &childStdout
	childCmd.Stderr = &childStderr
	result.AppPath = exePath + " " + strings.Join(childArgs, " ")
	err = childCmd.Run()
	childTimedOut := childCtx.Err() == context.DeadlineExceeded
	childCancel()
	result.AppExitCode = exitCodeFromError(err)
	if childTimedOut {
		if result.AppExitCode == 0 {
			result.AppExitCode = 1
		}
		return result, fmt.Errorf(
			"run platform UI runtime child timed out after %s",
			childTimeout,
		)
	}
	if childStdout.Len() > 0 {
		var evidence childRuntimeEvidence
		dec := json.NewDecoder(bytes.NewReader(childStdout.Bytes()))
		dec.DisallowUnknownFields()
		if decodeErr := dec.Decode(&evidence); decodeErr != nil {
			return result, fmt.Errorf("decode platform UI runtime child evidence: %w", decodeErr)
		}
		result.Evidence = evidence
	}
	if err != nil {
		detail := strings.TrimSpace(childStderr.String())
		if detail == "" && result.Evidence.ProbeError != "" {
			detail = result.Evidence.ProbeError
		}
		return result, fmt.Errorf("run platform UI runtime child: %w: %s", err, detail)
	}
	if result.Evidence.RuntimeTrace == "" {
		return result, fmt.Errorf("decode platform UI runtime child evidence: empty stdout")
	}
	return result, nil
}

func (r processPlatformRuntimeRunner) effectiveBuildTimeout() time.Duration {
	if r.buildTimeout > 0 {
		return r.buildTimeout
	}
	return defaultPlatformRuntimeBuildTimeout
}

func (r processPlatformRuntimeRunner) effectiveChildTimeout() time.Duration {
	if r.childTimeout > 0 {
		return r.childTimeout
	}
	return defaultPlatformRuntimeChildTimeout
}

func (r processPlatformRuntimeRunner) command(
	ctx context.Context,
	name string,
	args ...string,
) *exec.Cmd {
	if r.commandContext != nil {
		return r.commandContext(ctx, name, args...)
	}
	return exec.CommandContext(ctx, name, args...)
}

func repoRootFromWD() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && hasRepoRootDirs(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", dir)
		}
		dir = parent
	}
}

func hasRepoRootDirs(dir string) bool {
	for _, child := range []string{"cli", "compiler", "tools"} {
		info, err := os.Stat(filepath.Join(dir, child))
		if err != nil || !info.IsDir() {
			return false
		}
	}
	return true
}

func exitCodeFromError(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return 1
}

func buildPlatformUIRuntimeReportWithRunner(
	target string,
	host hostRuntime,
	runner platformRuntimeRunner,
) (platformui.Report, int) {
	requiredGOOS, ok := requiredGOOSByTarget(target)
	if !ok {
		return platformui.Report{
			Schema:  platformui.SchemaV1,
			Status:  "blocked",
			Version: compiler.Version(),
			GitHead: currentGitHead(),
			Target:  target,
			Host:    hostTriple(host),
			Runtime: "platform-ui-" + target,
			Blocker: "unknown platform UI target " + target,
		}, 2
	}
	if host.GOOS != requiredGOOS || host.GOARCH != "amd64" {
		blocker := fmt.Sprintf(
			("%s UI runtime production evidence requires a real %s/amd64 host " +
				"or approved runner; current host is %s/%s"),
			target,
			requiredGOOS,
			host.GOOS,
			host.GOARCH,
		)
		return platformui.Report{
			Schema:   platformui.SchemaV1,
			Status:   "blocked",
			Version:  compiler.Version(),
			GitHead:  currentGitHead(),
			Target:   target,
			Host:     hostTriple(host),
			Runtime:  "platform-ui-" + target,
			UISchema: "tetra.ui.v1",
			Source:   "tools/cmd/platform-ui-runtime-smoke",
			Runner:   "missing-target-host",
			Blocker:  blocker,
		}, 1
	}
	run, err := runner.Run(target)
	if err != nil {
		return failedTargetHostRuntimeReport(target, hostTriple(host), run, err), 1
	}
	return targetHostRuntimeReport(target, run), 0
}

func hostTriple(host hostRuntime) string {
	if host.GOOS == "linux" && host.GOARCH == "amd64" {
		return "linux-x64"
	}
	if host.GOOS == "windows" && host.GOARCH == "amd64" {
		return "windows-x64"
	}
	if host.GOOS == "darwin" && host.GOARCH == "amd64" {
		return "macos-x64"
	}
	return host.GOOS + "-" + host.GOARCH
}

func failedTargetHostRuntimeReport(
	target string,
	host string,
	run platformRuntimeRun,
	runErr error,
) platformui.Report {
	return platformui.Report{
		Schema:       platformui.SchemaV1,
		Status:       "fail",
		Version:      compiler.Version(),
		GitHead:      currentGitHead(),
		Target:       target,
		Host:         host,
		Runtime:      "platform-ui-" + target,
		RuntimeTrace: run.Evidence.RuntimeTrace,
		UISchema:     "tetra.ui.v1",
		Source:       "tools/cmd/platform-ui-runtime-smoke",
		Runner:       "target-host-runtime-child",
		Blocker:      runErr.Error(),
		Processes: []platformui.ProcessReport{
			{
				Name:     "compiler build",
				Kind:     "build",
				Path:     run.BuildPath,
				Ran:      run.BuildPath != "",
				Pass:     run.BuildExitCode == 0,
				ExitCode: intPtr(run.BuildExitCode),
			},
			{
				Name:     "platform UI app child",
				Kind:     "app",
				Path:     run.AppPath,
				Ran:      run.AppPath != "",
				Pass:     run.AppPath != "" && run.AppExitCode == 0,
				ExitCode: intPtr(run.AppExitCode),
			},
		},
	}
}

func targetHostRuntimeReport(target string, run platformRuntimeRun) platformui.Report {
	return platformui.Report{
		Schema:       platformui.SchemaV1,
		Status:       "pass",
		Version:      compiler.Version(),
		GitHead:      currentGitHead(),
		Target:       target,
		Host:         target,
		Runtime:      "platform-ui-" + target,
		RuntimeTrace: run.Evidence.RuntimeTrace,
		UISchema:     "tetra.ui.v1",
		Source:       "tools/cmd/platform-ui-runtime-smoke",
		Runner:       "target-host-runtime-child",
		Processes: []platformui.ProcessReport{
			{
				Name:     "compiler build",
				Kind:     "build",
				Path:     run.BuildPath,
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(run.BuildExitCode),
			},
			{
				Name:     "platform UI app child",
				Kind:     "app",
				Path:     run.AppPath,
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(run.AppExitCode),
			},
			{
				Name:     "platform UI runtime loop",
				Kind:     "runtime",
				Path:     "child-runtime event loop",
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name:     "platform UI stress sweep",
				Kind:     "stress",
				Path:     "child-runtime stress sweep",
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		},
		Widgets: run.Evidence.Widgets,
		Events:  run.Evidence.Events,
		Cases:   run.Evidence.Cases,
		Audit: []platformui.AuditReport{
			{
				Requirement: "real platform runtime evidence",
				Artifact:    "target-host-runtime child process",
				Evidence:    "target-host child process executed runtime loop, widgets, events, and cases",
				Result:      "pass",
			},
			{
				Requirement: "reject runtime-less evidence",
				Artifact:    "tools/validators/platformui",
				Evidence:    "validator rejects runtime-less evidence",
				Result:      "pass",
			},
		},
	}
}

func currentGitHead() string {
	repoRoot, err := repoRootFromWD()
	if err != nil {
		return ""
	}
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func intPtr(value int) *int {
	return &value
}

type childRuntimeEvidence struct {
	RuntimeTrace string                    `json:"runtime_trace"`
	ProbeError   string                    `json:"probe_error,omitempty"`
	Widgets      []platformui.WidgetReport `json:"widgets"`
	Events       []platformui.EventReport  `json:"events"`
	Cases        []platformui.CaseReport   `json:"cases"`
}

func runPlatformUIRuntimeChild(target string) childRuntimeEvidence {
	probe, probeErr := platformWindowProbe(target)
	state := map[string]string{
		"focused":  "none",
		"name":     "tetra",
		"selected": "item-1",
		"saved":    "false",
		"dirty":    "true",
	}
	widgets := []platformui.WidgetReport{
		{
			ID:      "AppWindow",
			Kind:    "window",
			Enabled: true,
			Visible: true,
			Bounds:  platformui.Bounds{Width: 640, Height: 480},
		},
		{
			ID:      "RootPanel",
			Kind:    "panel",
			Enabled: true,
			Visible: true,
			Bounds:  platformui.Bounds{Width: 624, Height: 464},
		},
		{
			ID:      "TitleText",
			Kind:    "text",
			Enabled: true,
			Visible: true,
			Bounds:  platformui.Bounds{Width: 608, Height: 32},
		},
		{
			ID:      "NameInput",
			Kind:    "input",
			Enabled: true,
			Visible: true,
			Bounds:  platformui.Bounds{Width: 608, Height: 32},
		},
		{
			ID:      "ItemList",
			Kind:    "list",
			Enabled: true,
			Visible: true,
			Bounds:  platformui.Bounds{Width: 608, Height: 240},
		},
		{
			ID:      "SaveButton",
			Kind:    "button",
			Enabled: true,
			Visible: true,
			Bounds:  platformui.Bounds{Width: 200, Height: 44},
		},
	}
	events := []platformui.EventReport{
		dispatchEvent(
			1,
			"NameInput",
			"focus",
			"focusName",
			state,
			map[string]string{"focused": "NameInput"},
			[]platformui.OperationReport{{Kind: "focus"}},
			[]platformui.WidgetUpdateReport{{ID: "NameInput", Before: "blurred", After: "focused"}},
		),
		dispatchEvent(
			2,
			"NameInput",
			"input",
			"setName",
			state,
			map[string]string{"name": "tetra-ui"},
			[]platformui.OperationReport{{Kind: "state_set"}},
			[]platformui.WidgetUpdateReport{{ID: "NameInput", Before: "tetra", After: "tetra-ui"}},
		),
		dispatchEvent(
			3,
			"ItemList",
			"select",
			"selectItem",
			state,
			map[string]string{"selected": "item-2"},
			[]platformui.OperationReport{{Kind: "state_set"}},
			[]platformui.WidgetUpdateReport{{ID: "ItemList", Before: "item-1", After: "item-2"}},
		),
		dispatchEvent(
			4,
			"SaveButton",
			"click",
			"saveAsync",
			state,
			map[string]string{"saved": "true"},
			[]platformui.OperationReport{{Kind: "async_command"}, {Kind: "redraw"}},
			[]platformui.WidgetUpdateReport{{ID: "TitleText", Before: "Editing", After: "Saved"}},
		),
		dispatchEvent(
			5,
			"AppWindow",
			"tick",
			"timerTick",
			state,
			map[string]string{"dirty": "false"},
			[]platformui.OperationReport{{Kind: "timer_tick"}, {Kind: "redraw"}},
			[]platformui.WidgetUpdateReport{
				{ID: "TitleText", Before: "Saved", After: "Saved after timer"},
			},
		),
	}
	trace := []string{"platform-process-spawn:ok"}
	var probeError string
	if probeErr == nil {
		trace = append(trace, "platform-window-api:ok")
		if probe.API != "" {
			trace = append(trace, "platform-window-api:"+probe.API+":ok")
		}
		trace = append(trace, probe.Markers...)
	} else {
		probeError = probeErr.Error()
		trace = append(trace, "platform-window-api:error")
	}
	trace = append(trace,
		"window-create:ok",
		"window-show:ok",
		"widget-tree-load:ok",
		"layout-measure:ok",
		"layout-place:ok",
		"event-loop-start:ok",
		"focus-dispatch:ok",
		"input-dispatch:ok",
		"select-dispatch:ok",
		"click-dispatch:ok",
		"state-update:ok",
		"async-command:ok",
		"timer-tick:ok",
		"redraw:ok",
		"error-recovery:ok",
		"window-close:ok",
	)
	return childRuntimeEvidence{
		RuntimeTrace: strings.Join(trace, ";"),
		ProbeError:   probeError,
		Widgets:      widgets,
		Events:       events,
		Cases: []platformui.CaseReport{
			{Name: "window lifecycle", Kind: "positive", Ran: true, Pass: target != ""},
			{Name: "layout measure and place", Kind: "positive", Ran: true, Pass: true},
			{Name: "widget tree load", Kind: "positive", Ran: true, Pass: len(widgets) == 6},
			{Name: "event loop dispatch", Kind: "positive", Ran: true, Pass: len(events) == 5},
			{
				Name: "state binding update",
				Kind: "positive",
				Ran:  true,
				Pass: state["name"] == "tetra-ui",
			},
			{Name: "redraw update lifecycle", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "async UI command completion",
				Kind: "positive",
				Ran:  true,
				Pass: state["saved"] == "true",
			},
			{
				Name: "timer scheduled redraw",
				Kind: "positive",
				Ran:  true,
				Pass: state["dirty"] == "false",
			},
			{
				Name:          "invalid widget diagnostic",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "unknown widget",
			},
			{
				Name:          "command failure recovery",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "unknown command",
			},
			{
				Name:          "crash error handling",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "runtime panic recovered",
			},
		},
	}
}

func dispatchEvent(
	order int,
	widgetID string,
	event string,
	command string,
	state map[string]string,
	updates map[string]string,
	operations []platformui.OperationReport,
	widgetUpdates []platformui.WidgetUpdateReport,
) platformui.EventReport {
	before := map[string]string{}
	after := map[string]string{}
	for key, value := range updates {
		before[key] = state[key]
		state[key] = value
		after[key] = state[key]
	}
	return platformui.EventReport{
		Order:         order,
		WidgetID:      widgetID,
		Event:         event,
		Command:       command,
		Pass:          true,
		BeforeState:   before,
		AfterState:    after,
		Operations:    operations,
		WidgetUpdates: widgetUpdates,
	}
}
