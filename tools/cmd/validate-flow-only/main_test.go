package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateFlowOnlyAcceptsFlowSource(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(src, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runFlowOnlyValidator(t, src)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateFlowOnlyRejectsLegacyBraceFunction(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "legacy.tetra")
	if err := os.WriteFile(src, []byte("fun main(): i32 {\n    return 0\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runFlowOnlyValidator(t, src)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	text := string(out)
	if !strings.Contains(text, "legacy.tetra:1:1") || !strings.Contains(text, "legacy function syntax") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateFlowOnlyRejectsLegacyBlockAndSemicolon(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "legacy_blocks.tetra")
	if err := os.WriteFile(src, []byte("func main() -> Int:\n    if (1) {\n        return 1;\n    }\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runFlowOnlyValidator(t, src)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	text := string(out)
	for _, want := range []string{"legacy braced if syntax", "trailing semicolon", "legacy brace token"} {
		if !strings.Contains(text, want) {
			t.Fatalf("unexpected output, missing %q:\n%s", want, out)
		}
	}
}

func TestValidateFlowOnlyRejectsTabsAndStandaloneBraces(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "bad.tetra")
	if err := os.WriteFile(src, []byte("func main() -> Int:\n\treturn 0\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runFlowOnlyValidator(t, src)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	text := string(out)
	for _, want := range []string{"tabs are not supported", "legacy brace token"} {
		if !strings.Contains(text, want) {
			t.Fatalf("unexpected output, missing %q:\n%s", want, out)
		}
	}
}

func TestValidateFlowOnlyIgnoresCommentsAndStrings(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "main.tetra")
	raw := "func main() -> Int uses io:\n    // legacy { ; } tokens in comment\n    print(\"literal with { } ; and // text\\n\")\n    return 0\n"
	if err := os.WriteFile(src, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runFlowOnlyValidator(t, src)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateFlowOnlyScansDirectories(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ok.tetra"), []byte("func ok() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.tetra"), []byte("while (1) {\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runFlowOnlyValidator(t, dir)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "bad.tetra:1:1") || strings.Contains(string(out), "ok.tetra") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func runFlowOnlyValidator(t *testing.T, args ...string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command("go", append([]string{"run", "."}, args...)...)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}
