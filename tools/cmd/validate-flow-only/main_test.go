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

func TestValidateFlowOnlyAcceptsT4FlowSource(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "main.t4")
	if err := os.WriteFile(src, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runFlowOnlyValidator(t, src)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateFlowOnlySkipsCapsuleManifest(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "Capsule.t4")
	raw := "manifest \"tetra.capsule.v1\"\ncapsule Demo:\n    id \"tetra://demo\"\n    version \"0.1.0\"\n"
	if err := os.WriteFile(src, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runFlowOnlyValidator(t, dir)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateFlowOnlyAcceptsFlowTestBlock(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "tests.tetra")
	raw := "test \"math\":\n    expect 40 + 2 == 42\n\nfunc main() -> Int:\n    return 0\n"
	if err := os.WriteFile(src, []byte(raw), 0o644); err != nil {
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

func TestValidateFlowOnlyRejectsLegacyBracedTestSyntax(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "legacy_test_block.tetra")
	raw := "test \"math\" {\n    expect 40 + 2 == 42\n}\n"
	if err := os.WriteFile(src, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runFlowOnlyValidator(t, src)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	text := string(out)
	for _, want := range []string{
		"legacy_test_block.tetra:1:1",
		"legacy braced test syntax",
		"legacy brace token",
	} {
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

func TestValidateFlowOnlySpanColumnsForTabsAndLegacyTest(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "span.tetra")
	raw := "    test \"Привіт\" {\r\n\t    expect 1 == 1\r\n"
	if err := os.WriteFile(src, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	issues, err := validateFile(src)
	if err != nil {
		t.Fatalf("validateFile: %v", err)
	}
	if len(issues) == 0 {
		t.Fatalf("expected issues")
	}
	var foundLegacy bool
	var foundTab bool
	for _, issue := range issues {
		switch issue.Message {
		case "legacy braced test syntax; use Flow 'test \"name\":'":
			foundLegacy = true
			if issue.Line != 1 || issue.Column != 5 {
				t.Fatalf("legacy test issue = %#v, want line 1 col 5", issue)
			}
		case "tabs are not supported in Flow indentation":
			foundTab = true
			if issue.Line != 2 || issue.Column != 1 {
				t.Fatalf("tab issue = %#v, want line 2 col 1", issue)
			}
		}
	}
	if !foundLegacy || !foundTab {
		t.Fatalf("issues = %#v, want both legacy test and tab diagnostics", issues)
	}
}

func runFlowOnlyValidator(t *testing.T, args ...string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command("go", append([]string{"run", "."}, args...)...)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}
