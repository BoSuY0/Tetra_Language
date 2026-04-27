package scriptstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFuzzNightlyWrapperDocumentsBoundedCommands(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "fuzz_nightly.sh"))
	if err != nil {
		t.Fatalf("read fuzz nightly script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"--fuzztime",
		"FuzzLexer",
		"FuzzParser",
		"FuzzParseCapsuleDoesNotPanic",
		"property-stress-regressions",
		"crasher_archive_path",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("fuzz nightly script missing %q", want)
		}
	}
}

func TestFuzzNightlyDocsNameWrapperAndCrashers(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "testing", "fuzz_property_stress.md"))
	if err != nil {
		t.Fatalf("read fuzz docs: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"bash scripts/fuzz_nightly.sh --out-dir reports/fuzz-nightly",
		"bash scripts/fuzz_nightly.sh --short",
		"<package>/testdata/fuzz/<FuzzName>/",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("fuzz docs missing %q", want)
		}
	}
}
