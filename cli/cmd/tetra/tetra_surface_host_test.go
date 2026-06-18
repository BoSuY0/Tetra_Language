package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSurfaceHostWaylandUsesSurfaceHostRunner(t *testing.T) {
	src := filepath.Join(t.TempDir(), "test", "surface_host_run.tetra")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(src, []byte(`module test.surface_host_run
func main() -> Int:
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	var gotPath string
	var gotOpt surfaceHostRunOptions
	restore := stubNativeSurfaceExec(
		func(path string, opt surfaceHostRunOptions, stdout io.Writer, stderr io.Writer) int {
			gotPath = path
			gotOpt = opt
			return 0
		},
	)
	defer restore()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runCLI(
		[]string{"run", "--target", "linux-x64", "--surface-host", "wayland", src},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("run exit = %d, stderr = %s", code, stderr.String())
	}
	if strings.TrimSpace(gotPath) == "" {
		t.Fatalf("surface-host runner did not receive built executable path")
	}
	if gotOpt.Backend != "wayland" {
		t.Fatalf("surface host backend = %q, want wayland", gotOpt.Backend)
	}
	if gotOpt.Protocol != "tetra.surface.host-ipc.v1" {
		t.Fatalf("surface host protocol = %q", gotOpt.Protocol)
	}
	if strings.TrimSpace(gotOpt.SocketPath) == "" {
		t.Fatalf("surface host socket path is empty")
	}
	if strings.TrimSpace(gotOpt.ReportPath) == "" {
		t.Fatalf("surface host report path is empty")
	}
	if gotOpt.RequiredEnv["TETRA_SURFACE_HOST_REQUIRED"] != "1" ||
		gotOpt.RequiredEnv["TETRA_SURFACE_HOST"] != "wayland" {
		t.Fatalf("required env = %#v", gotOpt.RequiredEnv)
	}
}

func TestRunSurfaceHostWaylandRequiresLinuxX64(t *testing.T) {
	src := filepath.Join(t.TempDir(), "test", "surface_host_reject.tetra")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(src, []byte(`module test.surface_host_reject
func main() -> Int:
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runCLI(
		[]string{"run", "--target", "wasm32-web", "--surface-host", "wayland", src},
		&stdout,
		&stderr,
	)
	if code != 2 {
		t.Fatalf("run exit = %d, want 2; stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "surface-host wayland requires linux-x64") {
		t.Fatalf("stderr = %s, want linux-x64 surface-host diagnostic", stderr.String())
	}
}

func TestRunSurfaceHostWaylandUsesExplicitHostReportPath(t *testing.T) {
	src := filepath.Join(t.TempDir(), "test", "surface_host_report.tetra")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(src, []byte(`module test.surface_host_report
func main() -> Int:
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	reportPath := filepath.Join(t.TempDir(), "host-report.json")
	var gotOpt surfaceHostRunOptions
	restore := stubNativeSurfaceExec(
		func(path string, opt surfaceHostRunOptions, stdout io.Writer, stderr io.Writer) int {
			gotOpt = opt
			return 0
		},
	)
	defer restore()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runCLI(
		[]string{
			"run",
			"--target",
			"linux-x64",
			"--surface-host",
			"wayland",
			"--surface-host-report",
			reportPath,
			src,
		},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("run exit = %d, stderr = %s", code, stderr.String())
	}
	if gotOpt.ReportPath != reportPath {
		t.Fatalf("surface host report path = %q, want %q", gotOpt.ReportPath, reportPath)
	}
}

func TestSurfaceRunUsesNativeWaylandHostPath(t *testing.T) {
	src := filepath.Join(t.TempDir(), "test", "surface_short_run.tetra")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(src, []byte(`module test.surface_short_run
func main() -> Int:
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	reportPath := filepath.Join(t.TempDir(), "surface-host-report.json")
	var gotPath string
	var gotOpt surfaceHostRunOptions
	restore := stubNativeSurfaceExec(
		func(path string, opt surfaceHostRunOptions, stdout io.Writer, stderr io.Writer) int {
			gotPath = path
			gotOpt = opt
			return 0
		},
	)
	defer restore()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runCLI(
		[]string{
			"surface",
			"run",
			"--host-report",
			reportPath,
			src,
		},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("surface run exit = %d, stderr = %s", code, stderr.String())
	}
	if strings.TrimSpace(gotPath) == "" {
		t.Fatalf("surface run did not build and pass an executable to native Surface runner")
	}
	if gotOpt.Backend != "wayland" {
		t.Fatalf("surface run backend = %q, want wayland", gotOpt.Backend)
	}
	if gotOpt.Protocol != "tetra.surface.host-ipc.v1" {
		t.Fatalf("surface run protocol = %q", gotOpt.Protocol)
	}
	if gotOpt.ReportPath != reportPath {
		t.Fatalf("surface run host report = %q, want %q", gotOpt.ReportPath, reportPath)
	}
	if gotOpt.RequiredEnv["TETRA_SURFACE_HOST_REQUIRED"] != "1" ||
		gotOpt.RequiredEnv["TETRA_SURFACE_HOST"] != "wayland" {
		t.Fatalf("surface run required env = %#v", gotOpt.RequiredEnv)
	}
}

func TestSurfaceRunForwardsRunDiagnosticsMode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runCLI(
		[]string{
			"surface",
			"run",
			"--diagnostics=json",
			"--target",
			"wasm32-web",
			"examples/surface/runtime/surface_window_counter.tetra",
		},
		&stdout,
		&stderr,
	)
	if code != 2 {
		t.Fatalf("surface run exit = %d, want 2; stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "surface-host wayland requires linux-x64") {
		t.Fatalf("stderr = %s, want linux-x64 surface-host diagnostic", stderr.String())
	}
}
