package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFmtCommandCheckAndStdout(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "func main() -> Int\nuses mem, io:\n    return 0\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"fmt", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("fmt exit code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "uses io, mem:") {
		t.Fatalf("fmt stdout = %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"fmt", "--check", srcPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("fmt --check should fail for unformatted file")
	}
}

func TestCollectTetraFilesIncludesT4AndLegacyTetra(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"main.t4":      "func main() -> Int:\n    return 0\n",
		"legacy.tetra": "func legacy() -> Int:\n    return 0\n",
		"ignore.tdx":   "not source\n",
	}
	for rel, src := range files {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := collectTetraFiles([]string{dir})
	if err != nil {
		t.Fatalf("collectTetraFiles: %v", err)
	}
	want := []string{filepath.Join(dir, "legacy.tetra"), filepath.Join(dir, "main.t4")}
	if len(got) != len(want) {
		t.Fatalf("files = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("files = %#v, want %#v", got, want)
		}
	}
}

func TestCollectTetraFilesSkipsCapsuleManifest(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `manifest "tetra.capsule.v1"
capsule Demo:
    id "tetra://demo"
    version "0.1.0"
`)
	writeCLIProjectFile(t, dir, "src/main.t4", "func main() -> Int:\n    return 0\n")
	got, err := collectTetraFiles([]string{dir})
	if err != nil {
		t.Fatalf("collectTetraFiles: %v", err)
	}
	want := []string{filepath.Join(dir, "src", "main.t4")}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("files = %#v, want %#v", got, want)
	}
}

func TestFormatCommandWriteIsIdempotentAndPreservesStandaloneComments(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := `// module docs
func main() -> Int uses mem, io:
    // return path
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"fmt", "--write", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("fmt --write exit code = %d, stderr=%q", code, stderr.String())
	}
	once, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"// module docs", "uses io, mem:", "    // return path"} {
		if !strings.Contains(string(once), want) {
			t.Fatalf("formatted file missing %q:\n%s", want, string(once))
		}
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"fmt", "--write", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("second fmt --write exit code = %d, stderr=%q", code, stderr.String())
	}
	twice, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(twice) != string(once) {
		t.Fatalf("fmt --write not idempotent:\nonce:\n%s\ntwice:\n%s", string(once), string(twice))
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"fmt", "--check", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("fmt --check after write exit code = %d, stderr=%q", code, stderr.String())
	}
}

func TestFormatCommandJSONDiagnosticsForInlineComment(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 0 // keep me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	diag := runCLIJSONDiagnostic(t, []string{"fmt", "--diagnostics=json", srcPath}, 1)
	if diag.Code != "TETRA_FMT001" || diag.File != srcPath || diag.Line != 2 || diag.Column != 14 || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
	if !strings.Contains(diag.Message, "inline comments are not supported") {
		t.Fatalf("diagnostic message = %q", diag.Message)
	}
}

func TestFmtCommandJSONDiagnosticsForInvalidModeCombination(t *testing.T) {
	diag := runCLIJSONDiagnostic(t, []string{"fmt", "--diagnostics=json", "--check", "--write", "examples/flow_hello.tetra"}, 2)
	if diag.Code != "TETRA0001" || diag.Message != "fmt accepts only one of --check or --write" || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestFmtCommandJSONDiagnosticsForMissingPath(t *testing.T) {
	diag := runCLIJSONDiagnostic(t, []string{"fmt", "--diagnostics=json"}, 2)
	if diag.Code != "TETRA0001" || diag.Message != "fmt requires at least one path" || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestFmtCommandJSONDiagnosticsForMultipleStdoutFiles(t *testing.T) {
	dir := t.TempDir()
	one := filepath.Join(dir, "one.tetra")
	two := filepath.Join(dir, "two.tetra")
	if err := os.WriteFile(one, []byte("func one() -> Int:\n    return 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(two, []byte("func two() -> Int:\n    return 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	diag := runCLIJSONDiagnostic(t, []string{"fmt", "--diagnostics=json", one, two}, 2)
	if diag.Code != "TETRA0001" || diag.Message != "fmt stdout mode accepts exactly one file" || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestFmtCheckJSONDiagnosticsForUnformattedFile(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "func main() -> Int uses io:\n    return 0\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	diag := runCLIJSONDiagnostic(t, []string{"fmt", "--check", "--diagnostics=json", srcPath}, 1)
	if diag.Code != "TETRA_FMT002" || diag.File != srcPath || diag.Message != "not formatted" || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestFmtCheckTOONDiagnosticsForUnformattedFile(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "func main() -> Int uses io:\n    return 0\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	diag := runCLITOONDiagnostic(t, []string{"fmt", "--check", "--diagnostics=toon", srcPath}, 1)
	if diag.Code != "TETRA_FMT002" || diag.File != srcPath || diag.Message != "not formatted" || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestFormatCommandCheckJSONDiagnosticsIncludesFirstDiffPosition(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int uses io:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	diag := runCLIJSONDiagnostic(t, []string{"fmt", "--check", "--diagnostics=json", srcPath}, 1)
	if diag.Code != "TETRA_FMT002" || diag.File != srcPath || diag.Line != 1 || diag.Column != 19 || diag.Message != "not formatted" || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}
