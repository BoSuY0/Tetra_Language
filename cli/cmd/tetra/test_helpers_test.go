package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler"
)

func stubLookPath(fn func(string) (string, error)) func() {
	old := commandLookPath
	oldWebRunnerProbe := webRunnerProbe
	commandLookPath = fn
	webRunnerProbe = func(string) error { return nil }
	return func() {
		commandLookPath = old
		webRunnerProbe = oldWebRunnerProbe
	}
}

func stubLinuxX32HostSupport(supported bool) func() {
	old := linuxX32HostSupport
	linuxX32HostSupport = func() bool { return supported }
	return func() {
		linuxX32HostSupport = old
	}
}

func stubNativeExec(fn func(string, io.Writer, io.Writer) int) func() {
	old := execNativeProgram
	execNativeProgram = fn
	return func() {
		execNativeProgram = old
	}
}

type cliJSONDiagnostic struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Hint     string `json:"hint"`
	Severity string `json:"severity"`
}

func runCLIJSONDiagnostic(t *testing.T, args []string, wantExit int) cliJSONDiagnostic {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := runCLI(args, &stdout, &stderr)
	if code != wantExit {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("expected empty stdout for JSON diagnostic, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	var diag cliJSONDiagnostic
	if err := json.Unmarshal(stderr.Bytes(), &diag); err != nil {
		t.Fatalf("json diagnostic: %v\n%s", err, stderr.String())
	}
	return diag
}

func runCLIJSONStdout(t *testing.T, args []string, wantExit int, out any) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := runCLI(args, &stdout, &stderr)
	if code != wantExit {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if err := json.Unmarshal(stdout.Bytes(), out); err != nil {
		t.Fatalf("json stdout: %v\n%s", err, stdout.String())
	}
	return stdout.String()
}

func assertCLIJSONOwnershipDiagnostic(t *testing.T, srcPath string, wantText string) {
	t.Helper()
	assertCLIJSONOwnershipDiagnosticForPath(t, srcPath, srcPath, wantText)
}

func assertCLIJSONOwnershipDiagnosticForPath(t *testing.T, checkPath string, diagPath string, wantText string) {
	t.Helper()
	assertCLIJSONDiagnosticForPath(t, checkPath, diagPath, compiler.DiagnosticCodeSafetyOwnership, wantText)
}

func assertCLIJSONLifetimeDiagnostic(t *testing.T, srcPath string, wantText string) {
	t.Helper()
	assertCLIJSONLifetimeDiagnosticForPath(t, srcPath, srcPath, wantText)
}

func assertCLIJSONLifetimeDiagnosticForPath(t *testing.T, checkPath string, diagPath string, wantText string) {
	t.Helper()
	assertCLIJSONDiagnosticForPath(t, checkPath, diagPath, compiler.DiagnosticCodeSafetyLifetime, wantText)
}

func assertCLIJSONSemanticDiagnostic(t *testing.T, checkPath string, diagPath string, wantText string) {
	t.Helper()
	assertCLIJSONDiagnosticForPath(t, checkPath, diagPath, compiler.DiagnosticCodeSemantic, wantText)
}

func assertCLIJSONDiagnosticForPath(t *testing.T, checkPath string, diagPath string, wantCode string, wantText string) {
	t.Helper()
	diag := runCLIJSONDiagnostic(t, []string{"check", "--diagnostics=json", checkPath}, 1)
	if diag.Code != wantCode || filepath.Clean(diag.File) != filepath.Clean(diagPath) || diag.Line <= 0 || diag.Column <= 0 || diag.Severity != "error" || !strings.Contains(diag.Message, wantText) {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func mustHostTarget(t *testing.T) string {
	t.Helper()
	target, ok := hostTarget()
	if !ok {
		t.Skip("host target unsupported")
	}
	return target
}

func writeCLIProjectFile(t *testing.T, root string, rel string, src string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

func nonHostTarget(t *testing.T) string {
	t.Helper()
	host := mustHostTarget(t)
	for _, target := range []string{"linux-x64", "macos-x64", "windows-x64"} {
		if target != host {
			return target
		}
	}
	t.Fatal("no non-host target found")
	return ""
}

func writeArtifactBuildFixture(t *testing.T, dir string, target string) string {
	t.Helper()
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", fmt.Sprintf(`capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
    targets:
        %s
`, target))
	writeCLIProjectFile(t, dir, "Math/src/math/core.t4", "module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b\n")
	writeCLIProjectFile(t, dir, "App/Capsule.t4", fmt.Sprintf(`capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    targets:
        %s
    deps:
        tetra://math 0.1.0 ../Math
`, target))
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n")
	return filepath.Join(dir, "App")
}

func writeWorkspaceMainProject(t *testing.T, root string, name string, id string, target string, exitCode int) {
	t.Helper()
	writeCLIProjectFile(t, root, filepath.ToSlash(filepath.Join(name, "Capsule.t4")), fmt.Sprintf(`capsule %s:
    id "%s"
    version "0.1.0"
    entry "src/main.t4"
    sources:
        src
    targets:
        %s
`, name, id, target))
	writeCLIProjectFile(t, root, filepath.ToSlash(filepath.Join(name, "src/main.t4")), fmt.Sprintf("func main() -> Int:\n    return %d\n", exitCode))
}

func writeWorkspaceTestProject(t *testing.T, root string, name string, id string, target string, testName string, condition string) {
	t.Helper()
	writeCLIProjectFile(t, root, filepath.ToSlash(filepath.Join(name, "Capsule.t4")), fmt.Sprintf(`capsule %s:
    id "%s"
    version "0.1.0"
    entry "src/main.t4"
    sources:
        src
    targets:
        %s
`, name, id, target))
	writeCLIProjectFile(t, root, filepath.ToSlash(filepath.Join(name, "src/main.t4")), "func main() -> Int:\n    return 0\n")
	writeCLIProjectFile(t, root, filepath.ToSlash(filepath.Join(name, "src/tests.t4")), fmt.Sprintf("test %q:\n    expect %s\n", testName, condition))
}
