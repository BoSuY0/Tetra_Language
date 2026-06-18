package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPassesShortFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, filepath.Join("docs", "guide.md"), "short line\n")

	var stdout, stderr bytes.Buffer
	exitCode := runLineLength(
		[]string{"--root", "docs", "--max", "100"},
		&stdout,
		&stderr,
		dir,
	)
	if exitCode != 0 {
		t.Fatalf(
			"exit = %d, want 0; stdout=%q stderr=%q",
			exitCode,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stdout.String(), "line length OK") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunReportsLongLine(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, filepath.Join("docs", "guide.md"), strings.Repeat("x", 101)+"\n")

	var stdout, stderr bytes.Buffer
	exitCode := runLineLength(
		[]string{"--root", "docs", "--max", "100"},
		&stdout,
		&stderr,
		dir,
	)
	if exitCode != 1 {
		t.Fatalf(
			"exit = %d, want 1; stdout=%q stderr=%q",
			exitCode,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "docs/guide.md:1: line is 101 chars, max 100") {
		t.Fatalf("stderr missing diagnostic:\n%s", stderr.String())
	}
}

func TestRunSkipsGeneratedAndCachePaths(t *testing.T) {
	dir := t.TempDir()
	longLine := strings.Repeat("x", 140) + "\n"
	writeFile(t, dir, filepath.Join("docs", "generated", "manifest.json"), longLine)
	writeFile(t, dir, filepath.Join("docs", "baselines", "api.json"), longLine)
	writeFile(t, dir, filepath.Join("docs", "benchmarks", "local_report.json"), longLine)
	writeFile(t, dir, filepath.Join("docs", "plans", "historical-prompt.md"), longLine)
	writeFile(t, dir, filepath.Join("docs", "release", "v0_4", "data", "scope.json"), longLine)
	writeFile(t, dir, filepath.Join(".cache", "generated.md"), longLine)
	writeFile(
		t,
		dir,
		filepath.Join("tools", "cmd", "validate", "line-length", "baseline.json"),
		longLine,
	)
	writeFile(t, dir, filepath.Join("docs", "guide.md"), "short\n")

	var stdout, stderr bytes.Buffer
	exitCode := runLineLength(
		[]string{"--root", "docs", "--root", ".cache", "--root", "tools", "--max", "100"},
		&stdout,
		&stderr,
		dir,
	)
	if exitCode != 0 {
		t.Fatalf(
			"exit = %d, want 0; stdout=%q stderr=%q",
			exitCode,
			stdout.String(),
			stderr.String(),
		)
	}
}

func TestRunAllowsURLAndChecksumExceptions(t *testing.T) {
	dir := t.TempDir()
	longURL := "see https://example.com/" + strings.Repeat("a", 110) + "\n"
	longHash := "checksum sha256:" + strings.Repeat("a", 64) + strings.Repeat("b", 50) + "\n"
	writeFile(t, dir, filepath.Join("docs", "links.md"), longURL+longHash)

	var stdout, stderr bytes.Buffer
	exitCode := runLineLength(
		[]string{"--root", "docs", "--max", "100"},
		&stdout,
		&stderr,
		dir,
	)
	if exitCode != 0 {
		t.Fatalf(
			"exit = %d, want 0; stdout=%q stderr=%q",
			exitCode,
			stdout.String(),
			stderr.String(),
		)
	}
}

func TestRunAllowsManualIgnoreAndReportsCount(t *testing.T) {
	dir := t.TempDir()
	line := strings.Repeat("x", 110) + " // line-length: ignore\n"
	writeFile(t, dir, filepath.Join("tools", "fixture.go"), "package tools\n"+line)

	var stdout, stderr bytes.Buffer
	exitCode := runLineLength(
		[]string{"--root", "tools", "--max", "100"},
		&stdout,
		&stderr,
		dir,
	)
	if exitCode != 0 {
		t.Fatalf(
			"exit = %d, want 0; stdout=%q stderr=%q",
			exitCode,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stdout.String(), "manual ignores: 1") {
		t.Fatalf("stdout missing manual ignore count:\n%s", stdout.String())
	}
}

func TestBaselineAllowsKnownLongLineButRejectsNewDebt(t *testing.T) {
	dir := t.TempDir()
	knownLine := strings.Repeat("k", 101)
	newLine := strings.Repeat("n", 102)
	writeFile(t, dir, filepath.Join("docs", "known.md"), knownLine+"\n")
	writeFile(t, dir, filepath.Join("docs", "new.md"), newLine+"\n")
	baselinePath := filepath.Join(dir, "baseline.json")
	writeLineBaseline(t, baselinePath, lineBaselineFile{
		Schema: baselineSchema,
		Max:    100,
		Allowances: []baselineAllowance{
			{
				Path:     "docs/known.md",
				LineHash: hashLine(knownLine),
				Length:   101,
				Reason:   "existing debt",
			},
		},
	})

	var stdout, stderr bytes.Buffer
	exitCode := runLineLength(
		[]string{"--root", "docs", "--max", "100", "--baseline", baselinePath},
		&stdout,
		&stderr,
		dir,
	)
	if exitCode != 1 {
		t.Fatalf(
			"exit = %d, want 1; stdout=%q stderr=%q",
			exitCode,
			stdout.String(),
			stderr.String(),
		)
	}
	if strings.Contains(stderr.String(), "docs/known.md") {
		t.Fatalf("known baseline debt should not be reported:\n%s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "docs/new.md:1: line is 102 chars, max 100") {
		t.Fatalf("new debt should be reported:\n%s", stderr.String())
	}
}

func TestRunWritesDeterministicBaseline(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, filepath.Join("docs", "a.md"), strings.Repeat("a", 101)+"\n")
	writeFile(t, dir, filepath.Join("docs", "b.md"), strings.Repeat("b", 102)+"\n")
	baselinePath := filepath.Join(dir, "baseline.json")

	var stdout, stderr bytes.Buffer
	exitCode := runLineLength(
		[]string{"--root", "docs", "--max", "100", "--write-baseline", baselinePath},
		&stdout,
		&stderr,
		dir,
	)
	if exitCode != 0 {
		t.Fatalf(
			"exit = %d, want 0; stdout=%q stderr=%q",
			exitCode,
			stdout.String(),
			stderr.String(),
		)
	}
	loaded := readLineBaselineFixture(t, baselinePath)
	if len(loaded.Allowances) != 2 {
		t.Fatalf("allowances = %d, want 2", len(loaded.Allowances))
	}
	if loaded.Allowances[0].Path != "docs/a.md" || loaded.Allowances[1].Path != "docs/b.md" {
		t.Fatalf("baseline order is not deterministic: %#v", loaded.Allowances)
	}
}

func TestStrictRejectsNonEmptyBaseline(t *testing.T) {
	dir := t.TempDir()
	baselinePath := filepath.Join(dir, "baseline.json")
	writeLineBaseline(t, baselinePath, lineBaselineFile{
		Schema: baselineSchema,
		Max:    100,
		Allowances: []baselineAllowance{
			{
				Path:     "docs/known.md",
				LineHash: hashLine(strings.Repeat("x", 101)),
				Length:   101,
				Reason:   "existing debt",
			},
		},
	})

	var stdout, stderr bytes.Buffer
	exitCode := runLineLength(
		[]string{"--root", "docs", "--max", "100", "--strict", "--baseline", baselinePath},
		&stdout,
		&stderr,
		dir,
	)
	if exitCode != 1 {
		t.Fatalf(
			"exit = %d, want 1; stdout=%q stderr=%q",
			exitCode,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "strict mode does not allow baseline allowances") {
		t.Fatalf("stderr missing strict baseline error:\n%s", stderr.String())
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func writeLineBaseline(t *testing.T, path string, baseline lineBaselineFile) {
	t.Helper()
	raw, err := json.Marshal(baseline)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func readLineBaselineFixture(t *testing.T, path string) lineBaselineFile {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var baseline lineBaselineFile
	if err := json.Unmarshal(raw, &baseline); err != nil {
		t.Fatal(err)
	}
	return baseline
}
