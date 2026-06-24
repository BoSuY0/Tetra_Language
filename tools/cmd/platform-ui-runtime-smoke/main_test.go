package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"tetra_language/tools/validators/platformui"
)

func TestBuildPlatformUIRuntimeReportPassesOnMatchingTargetHost(t *testing.T) {
	for _, tc := range []struct {
		target string
		goos   string
		goarch string
	}{
		{target: "windows-x64", goos: "windows", goarch: "amd64"},
		{target: "macos-x64", goos: "darwin", goarch: "amd64"},
	} {
		t.Run(tc.target, func(t *testing.T) {
			withPlatformWindowProbeForTest(t, platformWindowProbeResult{
				API:     "test-window-api",
				Markers: validPlatformProbeMarkersForTest(),
			})
			runner := &recordingPlatformRuntimeRunner{
				result: platformRuntimeRun{
					Evidence:      runPlatformUIRuntimeChild(tc.target),
					BuildPath:     "go build -o platform-ui-runtime-smoke ./tools/cmd/platform-ui-runtime-smoke",
					AppPath:       "platform-ui-runtime-smoke --child-runtime --target " + tc.target,
					BuildExitCode: 0,
					AppExitCode:   0,
				},
			}
			report, exitCode := buildPlatformUIRuntimeReportWithRunner(
				tc.target,
				hostRuntime{GOOS: tc.goos, GOARCH: tc.goarch},
				runner,
			)
			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0: %#v", exitCode, report)
			}
			if report.Version == "" || report.GitHead == "" {
				t.Fatalf("target-host report missing version/git_head: %#v", report)
			}
			raw := marshalReportForTest(t, report)
			if err := platformui.ValidateReport(raw, tc.target); err != nil {
				t.Fatalf("target-host report should validate: %v\n%s", err, raw)
			}
		})
	}
}

func TestBuildPlatformUIRuntimeReportUsesExecutedChildRuntimeEvidence(t *testing.T) {
	withPlatformWindowProbeForTest(t, platformWindowProbeResult{
		API:     "test-window-api",
		Markers: validPlatformProbeMarkersForTest(),
	})
	runner := &recordingPlatformRuntimeRunner{
		result: platformRuntimeRun{
			Evidence:      runPlatformUIRuntimeChild("windows-x64"),
			BuildPath:     "go build -o platform-ui-runtime-smoke ./tools/cmd/platform-ui-runtime-smoke",
			AppPath:       "platform-ui-runtime-smoke --child-runtime --target windows-x64",
			BuildExitCode: 0,
			AppExitCode:   0,
		},
	}
	report, exitCode := buildPlatformUIRuntimeReportWithRunner(
		"windows-x64",
		hostRuntime{GOOS: "windows", GOARCH: "amd64"},
		runner,
	)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0: %#v", exitCode, report)
	}
	if !runner.called {
		t.Fatal("target-host pass report must execute the platform runtime child")
	}
	if report.Runner != "target-host-runtime-child" {
		t.Fatalf("runner = %q, want target-host-runtime-child", report.Runner)
	}
	if report.Version == "" || report.GitHead == "" {
		t.Fatalf("target-host report missing version/git_head: %#v", report)
	}
	if !strings.Contains(report.RuntimeTrace, "platform-window-api:ok") ||
		!strings.Contains(report.RuntimeTrace, "window-create:ok") ||
		!strings.Contains(report.RuntimeTrace, "error-recovery:ok") {
		t.Fatalf(
			"target-host report runtime_trace missing required markers: %q",
			report.RuntimeTrace,
		)
	}
	if got := report.Processes[1].Path; !strings.Contains(got, "--child-runtime") {
		t.Fatalf("app process path = %q, want child runtime evidence", got)
	}
	raw := marshalReportForTest(t, report)
	if err := platformui.ValidateReport(raw, "windows-x64"); err != nil {
		t.Fatalf("child-backed report should validate: %v\n%s", err, raw)
	}
}

func TestRunPlatformUIRuntimeChildUsesOSBackedWindowProbe(t *testing.T) {
	called := false
	old := platformWindowProbe
	platformWindowProbe = func(target string) (platformWindowProbeResult, error) {
		called = true
		if target != "windows-x64" {
			t.Fatalf("probe target = %q, want windows-x64", target)
		}
		return platformWindowProbeResult{
			API:     "test-window-api",
			Markers: validPlatformProbeMarkersForTest(),
		}, nil
	}
	t.Cleanup(func() { platformWindowProbe = old })

	evidence := runPlatformUIRuntimeChild("windows-x64")
	if !called {
		t.Fatal("child runtime must execute the OS-backed platform window probe")
	}
	for _, marker := range []string{"platform-window-api:ok", "window-create:ok", "window-close:ok"} {
		if !strings.Contains(evidence.RuntimeTrace, marker) {
			t.Fatalf("runtime trace missing %q: %s", marker, evidence.RuntimeTrace)
		}
	}
}

func TestBuildPlatformUIRuntimeReportFailsWhenOSBackedWindowProbeFails(t *testing.T) {
	report, exitCode := buildPlatformUIRuntimeReportWithRunner(
		"windows-x64",
		hostRuntime{GOOS: "windows", GOARCH: "amd64"},
		processPlatformRuntimeRunner{},
	)
	if exitCode == 0 {
		t.Fatalf("expected nonzero exit when OS-backed window probe fails: %#v", report)
	}
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	if !strings.Contains(report.Blocker, "platform window probe") {
		t.Fatalf("blocker = %q, want probe failure", report.Blocker)
	}
	if !strings.Contains(report.RuntimeTrace, "platform-window-api:error") {
		t.Fatalf("runtime_trace = %q, want platform-window-api:error", report.RuntimeTrace)
	}
}

func TestPlatformRuntimeChildTimeoutIsBounded(t *testing.T) {
	runner := processPlatformRuntimeRunner{
		buildTimeout: 5 * time.Second,
		childTimeout: 100 * time.Millisecond,
		commandContext: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			mode := "child"
			if name == "go" {
				mode = "build"
			}
			cmd := exec.CommandContext(
				ctx,
				os.Args[0],
				"-test.run=^TestPlatformRuntimeTimeoutHelper$",
			)
			cmd.Env = append(
				os.Environ(),
				"TETRA_PLATFORM_RUNTIME_TIMEOUT_HELPER=1",
				"TETRA_PLATFORM_RUNTIME_TIMEOUT_HELPER_MODE="+mode,
			)
			return cmd
		},
	}
	start := time.Now()
	report, exitCode := buildPlatformUIRuntimeReportWithRunner(
		"windows-x64",
		hostRuntime{GOOS: "windows", GOARCH: "amd64"},
		runner,
	)
	elapsed := time.Since(start)
	if elapsed > 5*time.Second {
		t.Fatalf("child timeout returned after %s, want under 5s", elapsed)
	}
	if exitCode == 0 {
		t.Fatalf("exit code = 0, want nonzero: %#v", report)
	}
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	if !strings.Contains(report.Blocker, "timed out") ||
		!strings.Contains(report.Blocker, "run platform UI runtime child timed out after 100ms") {
		t.Fatalf("blocker = %q, want child timeout", report.Blocker)
	}
	if len(report.Processes) < 2 {
		t.Fatalf("processes = %#v, want build and app processes", report.Processes)
	}
	app := report.Processes[1]
	if app.Path == "" {
		t.Fatal("app process path is empty, want child AppPath")
	}
	if app.ExitCode == nil || *app.ExitCode == 0 {
		t.Fatalf("app exit code = %v, want nonzero", app.ExitCode)
	}
	if app.Pass {
		t.Fatal("app process pass = true, want false")
	}
	raw := marshalReportForTest(t, report)
	if !json.Valid(raw) {
		t.Fatalf("timeout report is not JSON: %s", raw)
	}
}

func TestPlatformRuntimeTimeoutHelper(t *testing.T) {
	if os.Getenv("TETRA_PLATFORM_RUNTIME_TIMEOUT_HELPER") != "1" {
		return
	}
	switch os.Getenv("TETRA_PLATFORM_RUNTIME_TIMEOUT_HELPER_MODE") {
	case "build":
		return
	case "child":
		time.Sleep(10 * time.Second)
	default:
		t.Fatalf(
			"unknown TETRA_PLATFORM_RUNTIME_TIMEOUT_HELPER_MODE %q",
			os.Getenv("TETRA_PLATFORM_RUNTIME_TIMEOUT_HELPER_MODE"),
		)
	}
}

func withPlatformWindowProbeForTest(t *testing.T, result platformWindowProbeResult) {
	t.Helper()
	old := platformWindowProbe
	platformWindowProbe = func(target string) (platformWindowProbeResult, error) {
		return result, nil
	}
	t.Cleanup(func() { platformWindowProbe = old })
}

func validPlatformProbeMarkersForTest() []string {
	return []string{
		"platform-widget-tree:ok",
		"platform-event-dispatch:ok",
		"platform-timer:ok",
		"platform-redraw:ok",
	}
}

type recordingPlatformRuntimeRunner struct {
	called bool
	result platformRuntimeRun
}

func (r *recordingPlatformRuntimeRunner) Run(target string) (platformRuntimeRun, error) {
	r.called = true
	return r.result, nil
}

func marshalReportForTest(t *testing.T, report platformui.Report) []byte {
	t.Helper()
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	return raw
}

func TestBuildPlatformUIRuntimeReportBlocksOnWrongHost(t *testing.T) {
	report, exitCode := buildPlatformUIRuntimeReport(
		"windows-x64",
		hostRuntime{GOOS: "linux", GOARCH: "amd64"},
	)
	if exitCode == 0 {
		t.Fatalf("expected blocked exit code, report=%#v", report)
	}
	raw := marshalReportForTest(t, report)
	err := platformui.ValidateReport(raw, "windows-x64")
	if err == nil {
		t.Fatalf("blocked report must not validate as production evidence")
	}
	for _, want := range []string{"status", "host", "target host"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q: %v", want, err)
		}
	}
}
