package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ctarget "tetra_language/compiler/target"
)

func TestWorkspaceInitAddListAndRemove(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "App/src/main.t4", "func main() -> Int:\n    return 0\n")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"workspace", "init", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("workspace init exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "Tetra.workspace")); err != nil {
		t.Fatalf("expected Tetra.workspace: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"workspace", "add", "App", "--workspace", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("workspace add exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(filepath.Join(dir, "Tetra.workspace"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `member "App"`) {
		t.Fatalf("workspace missing member:\n%s", string(raw))
	}

	var report struct {
		Root    string `json:"root"`
		Members []struct {
			Path      string `json:"path"`
			CapsuleID string `json:"capsule_id"`
			Status    string `json:"status"`
		} `json:"members"`
	}
	runCLIJSONStdout(t, []string{"workspace", "list", "--format=json", dir}, 0, &report)
	if filepath.Clean(report.Root) != filepath.Clean(dir) || len(report.Members) != 1 || report.Members[0].Path != "App" || report.Members[0].CapsuleID != "tetra://app" || report.Members[0].Status != "ok" {
		t.Fatalf("workspace list report = %#v", report)
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"workspace", "remove", "App", "--workspace", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("workspace remove exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err = os.ReadFile(filepath.Join(dir, "Tetra.workspace"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `member "App"`) {
		t.Fatalf("workspace member was not removed:\n%s", string(raw))
	}
}

func TestWorkspaceCheckGraphAndSync(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	appRoot := writeArtifactBuildFixture(t, dir, target)
	writeCLIProjectFile(t, dir, "Tetra.workspace", `workspace "tetra.workspace.v1"
member "Math"
member "App"
`)

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"workspace", "check", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("workspace check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Workspace OK") {
		t.Fatalf("stdout = %q", stdout.String())
	}

	var graph struct {
		Nodes []struct {
			Path      string `json:"path"`
			CapsuleID string `json:"capsule_id"`
		} `json:"nodes"`
		Edges []struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"edges"`
	}
	runCLIJSONStdout(t, []string{"workspace", "graph", "--format=json", dir}, 0, &graph)
	if len(graph.Nodes) != 2 || len(graph.Edges) != 1 || graph.Edges[0].From != "App" || graph.Edges[0].To != "Math" {
		t.Fatalf("workspace graph = %#v", graph)
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"workspace", "sync", "--check", "--target", target, dir}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("workspace sync --check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String()+stderr.String(), "would generate") {
		t.Fatalf("sync --check output = stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(appRoot, "Tetra.lock")); err == nil {
		t.Fatalf("workspace sync --check unexpectedly wrote App Tetra.lock")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat App Tetra.lock: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"workspace", "sync", "--target", target, dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("workspace sync exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	for _, rel := range []string{"Tetra.lock", "interfaces/math/core.t4i", "artifacts/math/core." + target + ".tobj", "seeds/app-deps.t4s"} {
		if _, err := os.Stat(filepath.Join(appRoot, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected workspace sync generated %s: %v", rel, err)
		}
	}
}

func TestWorkspaceCheckFailures(t *testing.T) {
	t.Run("missing member", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Tetra.workspace", `workspace "tetra.workspace.v1"
member "Missing"
`)
		var stdout, stderr bytes.Buffer
		code := runCLI([]string{"workspace", "check", dir}, &stdout, &stderr)
		if code != 1 {
			t.Fatalf("workspace check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
		if !strings.Contains(stderr.String(), "Missing") {
			t.Fatalf("stderr = %q", stderr.String())
		}
	})
	t.Run("duplicate capsule id", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "A/Capsule.t4", `capsule A:
    id "tetra://dup"
    version "0.1.0"
`)
		writeCLIProjectFile(t, dir, "B/Capsule.t4", `capsule B:
    id "tetra://dup"
    version "0.1.0"
`)
		writeCLIProjectFile(t, dir, "Tetra.workspace", `workspace "tetra.workspace.v1"
member "A"
member "B"
`)
		var stdout, stderr bytes.Buffer
		code := runCLI([]string{"workspace", "check", dir}, &stdout, &stderr)
		if code != 1 {
			t.Fatalf("workspace check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
		if !strings.Contains(stderr.String(), "duplicate capsule id") {
			t.Fatalf("stderr = %q", stderr.String())
		}
	})
	t.Run("dependency cycle", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    deps:
        tetra://math 0.1.0 ../Math
`)
		writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    deps:
        tetra://app 0.1.0 ../App
`)
		writeCLIProjectFile(t, dir, "Tetra.workspace", `workspace "tetra.workspace.v1"
member "App"
member "Math"
`)
		var stdout, stderr bytes.Buffer
		code := runCLI([]string{"workspace", "check", dir}, &stdout, &stderr)
		if code != 1 {
			t.Fatalf("workspace check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
		if !strings.Contains(stderr.String(), "capsule dependency cycle") {
			t.Fatalf("stderr = %q", stderr.String())
		}
	})
}

func TestWorkspaceBuildWritesPerMemberOutputsAndJSONSummary(t *testing.T) {
	target := mustHostTarget(t)
	tgt, err := ctarget.Parse(target)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writeWorkspaceMainProject(t, dir, "App", "tetra://app", target, 0)
	writeWorkspaceMainProject(t, dir, "Tool", "tetra://tool", target, 0)
	writeCLIProjectFile(t, dir, "Tetra.workspace", `workspace "tetra.workspace.v1"
member "App"
member "Tool"
`)
	outDir := filepath.Join(dir, "dist")

	var report struct {
		Command string `json:"command"`
		Total   int    `json:"total"`
		Passed  int    `json:"passed"`
		Failed  int    `json:"failed"`
		Skipped int    `json:"skipped"`
		Members []struct {
			Path   string `json:"path"`
			Status string `json:"status"`
		} `json:"members"`
	}
	runCLIJSONStdout(t, []string{"workspace", "build", "--target", target, "--format=json", "-o", outDir, dir}, 0, &report)
	if report.Command != "build" || report.Total != 2 || report.Passed != 2 || report.Failed != 0 || report.Skipped != 0 {
		t.Fatalf("workspace build report = %#v", report)
	}
	for _, rel := range []string{
		filepath.ToSlash(filepath.Join("App", defaultOutput(tgt, "exe"))),
		filepath.ToSlash(filepath.Join("Tool", defaultOutput(tgt, "exe"))),
	} {
		if _, err := os.Stat(filepath.Join(outDir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected workspace build output %s: %v", rel, err)
		}
	}
}

func TestWorkspaceBuildSkipsDependentAfterFailedDependency(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Lib/Capsule.t4", fmt.Sprintf(`capsule Lib:
    id "tetra://lib"
    version "0.1.0"
    entry "src/main.t4"
    sources:
        src
    targets:
        %s
`, target))
	writeCLIProjectFile(t, dir, "Lib/src/main.t4", "func main() -> Int:\n    return\n")
	writeCLIProjectFile(t, dir, "App/Capsule.t4", fmt.Sprintf(`capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/main.t4"
    sources:
        src
    targets:
        %s
    deps:
        tetra://lib 0.1.0 ../Lib
`, target))
	writeCLIProjectFile(t, dir, "App/src/main.t4", "func main() -> Int:\n    return 0\n")
	writeCLIProjectFile(t, dir, "Tetra.workspace", `workspace "tetra.workspace.v1"
member "Lib"
member "App"
`)

	var report struct {
		Failed  int `json:"failed"`
		Skipped int `json:"skipped"`
		Members []struct {
			Path   string `json:"path"`
			Status string `json:"status"`
			Detail string `json:"detail"`
		} `json:"members"`
	}
	runCLIJSONStdout(t, []string{"workspace", "build", "--target", target, "--format=json", "-o", filepath.Join(dir, "dist"), dir}, 1, &report)
	if report.Failed != 1 || report.Skipped != 1 || len(report.Members) != 2 {
		t.Fatalf("workspace build report = %#v", report)
	}
	if report.Members[0].Path != "Lib" || report.Members[0].Status != "fail" {
		t.Fatalf("first member = %#v", report.Members[0])
	}
	if report.Members[1].Path != "App" || report.Members[1].Status != "skipped" || !strings.Contains(report.Members[1].Detail, "Lib") {
		t.Fatalf("dependent member = %#v", report.Members[1])
	}
}

func TestWorkspaceTestFailFastJSONSummary(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	writeWorkspaceTestProject(t, dir, "Pass", "tetra://pass", target, "pass ok", "40 + 2 == 42")
	writeWorkspaceTestProject(t, dir, "Fail", "tetra://fail", target, "fail bad", "1 == 2")
	writeWorkspaceTestProject(t, dir, "Later", "tetra://later", target, "later ok", "2 + 2 == 4")
	writeCLIProjectFile(t, dir, "Tetra.workspace", `workspace "tetra.workspace.v1"
member "Pass"
member "Fail"
member "Later"
`)

	var report struct {
		Command string `json:"command"`
		Total   int    `json:"total"`
		Passed  int    `json:"passed"`
		Failed  int    `json:"failed"`
		Skipped int    `json:"skipped"`
		Members []struct {
			Path   string `json:"path"`
			Status string `json:"status"`
		} `json:"members"`
	}
	runCLIJSONStdout(t, []string{"workspace", "test", "--target", target, "--fail-fast", "--format=json", dir}, 1, &report)
	if report.Command != "test" || report.Total != 3 || report.Passed != 1 || report.Failed != 1 || report.Skipped != 1 {
		t.Fatalf("workspace test report = %#v", report)
	}
	if report.Members[2].Path != "Later" || report.Members[2].Status != "skipped" {
		t.Fatalf("fail-fast member = %#v", report.Members[2])
	}
}

func TestWorkspaceRunMemberAndUnknownMember(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	writeWorkspaceMainProject(t, dir, "App", "tetra://app", target, 7)
	writeWorkspaceMainProject(t, dir, "Tool", "tetra://tool", target, 0)
	writeCLIProjectFile(t, dir, "Tetra.workspace", `workspace "tetra.workspace.v1"
member "App"
member "Tool"
`)

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"workspace", "run", "App", "--workspace", dir, "--target", target}, &stdout, &stderr)
	if code != 7 {
		t.Fatalf("workspace run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"workspace", "run", "Missing", "--workspace", dir, "--target", target}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("unknown workspace run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "workspace member not found") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
