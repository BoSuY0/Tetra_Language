package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateFuzzSummaryAcceptsShortReport(t *testing.T) {
	dir := makeFuzzReport(t, nil)
	out, err := runFuzzValidator(t, dir)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateFuzzSummaryRejectsMissingRequiredLog(t *testing.T) {
	dir := makeFuzzReport(t, nil)
	if err := os.Remove(filepath.Join(dir, "logs", "validate-manifest.log")); err != nil {
		t.Fatal(err)
	}
	out, err := runFuzzValidator(t, dir)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing required artifact logs/validate-manifest.log") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateFuzzSummaryRejectsMissingUnstableSeedArtifact(t *testing.T) {
	dir := makeFuzzReport(t, nil)
	if err := os.Remove(filepath.Join(dir, "unstable-seeds.md")); err != nil {
		t.Fatal(err)
	}
	out, err := runFuzzValidator(t, dir)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing required artifact unstable-seeds.md") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateFuzzSummaryRejectsMissingSummaryJSON(t *testing.T) {
	dir := makeFuzzReport(t, nil)
	if err := os.Remove(filepath.Join(dir, "summary.json")); err != nil {
		t.Fatal(err)
	}
	out, err := runFuzzValidator(t, dir)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing required artifact summary.json") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateFuzzSummaryRejectsMalformedSummaryJSON(t *testing.T) {
	dir := makeFuzzReport(t, nil)
	if err := os.WriteFile(filepath.Join(dir, "summary.json"), []byte("{not json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runFuzzValidator(t, dir)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "summary.json is malformed") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateFuzzSummaryRejectsMalformedMetadata(t *testing.T) {
	dir := makeFuzzReport(t, map[string]string{
		"mode": "- mode: `turbo`",
	})
	out, err := runFuzzValidator(t, dir)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), `invalid mode "turbo"`) {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateFuzzSummaryRejectsFailingStep(t *testing.T) {
	dir := makeFuzzReport(t, map[string]string{
		"step-validate-manifest": "- `validate-manifest`: fail exit `1`, command `go test ./tools/cmd/validate-manifest -run \\^\\$ -fuzz=. -fuzztime=2s -parallel=1`, log `" + filepath.ToSlash(filepath.Join("OUT_DIR", "logs", "validate-manifest.log")) + "`",
	})
	out, err := runFuzzValidator(t, dir)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "step validate-manifest has invalid or failing status") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateFuzzSummaryRejectsUnknownStep(t *testing.T) {
	dir := makeFuzzReport(t, map[string]string{
		"step-validate-manifest": "- `surprise`: pass, command `go test ./tools/cmd/validate-manifest -run \\^\\$ -fuzz=. -fuzztime=2s -parallel=1`, log `" + filepath.ToSlash(filepath.Join("OUT_DIR", "logs", "validate-manifest.log")) + "`",
	})
	out, err := runFuzzValidator(t, dir)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "step 7 name = \"surprise\", want \"validate-manifest\"") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateFuzzSummaryRejectsMalformedUnstableSeedLog(t *testing.T) {
	dir := makeFuzzReport(t, nil)
	if err := os.WriteFile(filepath.Join(dir, "unstable-seeds.md"), []byte("# Unstable Fuzz Seeds\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runFuzzValidator(t, dir)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unstable-seeds.md missing table header") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func makeFuzzReport(t *testing.T, overrides map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	logsDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range fuzzStepNames {
		if err := os.WriteFile(filepath.Join(logsDir, name+".log"), []byte("ok\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "unstable-seeds.md"), []byte(`# Unstable Fuzz Seeds

Record any flaky, timeout-sensitive, or non-deterministic fuzz seed observed
during this run.

| package | fuzz target | seed/crasher path | status | owner | next command |
| --- | --- | --- | --- | --- | --- |
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "crasher-inventory.json"), []byte(`{
  "schema_version": 1,
  "kind": "go-testdata-fuzz-inventory",
  "scanned_roots": [],
  "counts": {
    "roots": 0,
    "existing_roots": 0,
    "targets": 0,
    "corpus_files": 0,
    "crasher_files": 0,
    "total_files": 0
  }
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	lines := map[string]string{
		"mode":                             "- mode: `short`",
		"fuzztime":                         "- fuzztime: `2s`",
		"output_dir":                       "- output_dir: `" + filepath.ToSlash(dir) + "`",
		"crasher_archive_path":             "- crasher_archive_path: `<package>/testdata/fuzz/<FuzzName>/`",
		"crasher_inventory_json":           "- crasher_inventory_json: `" + filepath.ToSlash(filepath.Join(dir, "crasher-inventory.json")) + "`",
		"unstable_seed_log":                "- unstable_seed_log: `" + filepath.ToSlash(filepath.Join(dir, "unstable-seeds.md")) + "`",
		"step-compiler-frontend-lexer":     "- `compiler-frontend-lexer`: pass, command `go test ./compiler/internal/frontend -run \\^\\$ -fuzz=FuzzLexer -fuzztime=2s -parallel=1`, log `" + filepath.ToSlash(filepath.Join(dir, "logs", "compiler-frontend-lexer.log")) + "`",
		"step-compiler-frontend-parser":    "- `compiler-frontend-parser`: pass, command `go test ./compiler/internal/frontend -run \\^\\$ -fuzz=FuzzParser -fuzztime=2s -parallel=1`, log `" + filepath.ToSlash(filepath.Join(dir, "logs", "compiler-frontend-parser.log")) + "`",
		"step-compiler-linker-linkcore":    "- `compiler-linker-linkcore`: pass, command `go test ./compiler/internal/linker/linkcore -run \\^\\$ -fuzz=FuzzLinkX64ObjectsDoesNotPanic -fuzztime=2s -parallel=1`, log `" + filepath.ToSlash(filepath.Join(dir, "logs", "compiler-linker-linkcore.log")) + "`",
		"step-http-runtime":                "- `http-runtime`: pass, command `go test ./compiler/internal/httprt -run \\^\\$ -fuzz=FuzzHTTPParseRequest -fuzztime=2s -parallel=1`, log `" + filepath.ToSlash(filepath.Join(dir, "logs", "http-runtime.log")) + "`",
		"step-json-runtime":                "- `json-runtime`: pass, command `go test ./compiler/internal/jsonrt -run \\^\\$ -fuzz=FuzzAppendStringProducesValidJSON -fuzztime=2s -parallel=1`, log `" + filepath.ToSlash(filepath.Join(dir, "logs", "json-runtime.log")) + "`",
		"step-postgres-wire":               "- `postgres-wire`: pass, command `go test ./compiler/internal/pgrt -run \\^\\$ -fuzz=FuzzReadFrameDoesNotPanic -fuzztime=2s -parallel=1`, log `" + filepath.ToSlash(filepath.Join(dir, "logs", "postgres-wire.log")) + "`",
		"step-validate-manifest":           "- `validate-manifest`: pass, command `go test ./tools/cmd/validate-manifest -run \\^\\$ -fuzz=. -fuzztime=2s -parallel=1`, log `" + filepath.ToSlash(filepath.Join(dir, "logs", "validate-manifest.log")) + "`",
		"step-eco-capsule":                 "- `eco-capsule`: pass, command `go test ./cli/cmd/tetra -run \\^\\$ -fuzz=FuzzParseCapsuleDoesNotPanic -fuzztime=2s -parallel=1`, log `" + filepath.ToSlash(filepath.Join(dir, "logs", "eco-capsule.log")) + "`",
		"step-property-stress-regressions": "- `property-stress-regressions`: pass, command `go test ./compiler/... ./cli/... ./tools/cmd/validate-manifest -run Fuzz\\|Property\\|Stress -count=1`, log `" + filepath.ToSlash(filepath.Join(dir, "logs", "property-stress-regressions.log")) + "`",
	}
	for key, value := range overrides {
		lines[key] = strings.ReplaceAll(value, "OUT_DIR", filepath.ToSlash(dir))
	}
	summary := strings.Join([]string{
		"# Fuzz Nightly Summary",
		"",
		lines["mode"],
		lines["fuzztime"],
		lines["output_dir"],
		lines["crasher_archive_path"],
		lines["crasher_inventory_json"],
		lines["unstable_seed_log"],
		"",
		"## Steps",
		lines["step-compiler-frontend-lexer"],
		lines["step-compiler-frontend-parser"],
		lines["step-compiler-linker-linkcore"],
		lines["step-http-runtime"],
		lines["step-json-runtime"],
		lines["step-postgres-wire"],
		lines["step-validate-manifest"],
		lines["step-eco-capsule"],
		lines["step-property-stress-regressions"],
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(dir, "summary.md"), []byte(summary), 0o644); err != nil {
		t.Fatal(err)
	}
	summaryJSON := `{
  "mode": "short",
  "status": "pass",
  "exit_code": 0,
  "duration_seconds": 0,
  "started_at": "2026-04-29T00:00:00Z",
  "ended_at": "2026-04-29T00:00:01Z",
  "fuzztime": "2s",
  "step_count": 9,
  "failed_count": 0,
	"artifacts": {
	    "summary_md": "` + filepath.ToSlash(filepath.Join(dir, "summary.md")) + `",
	    "summary_json": "` + filepath.ToSlash(filepath.Join(dir, "summary.json")) + `",
	    "crasher_inventory_json": "` + filepath.ToSlash(filepath.Join(dir, "crasher-inventory.json")) + `",
	    "logs_dir": "` + filepath.ToSlash(filepath.Join(dir, "logs")) + `",
    "unstable_seed_log": "` + filepath.ToSlash(filepath.Join(dir, "unstable-seeds.md")) + `",
    "crasher_archive_path": "<package>/testdata/fuzz/<FuzzName>/"
  },
  "steps": [
    {"name":"compiler-frontend-lexer","status":"pass","duration_seconds":0,"exit_code":0,"command":"go test ./compiler/internal/frontend -run \\^\\$ -fuzz=FuzzLexer -fuzztime=2s -parallel=1","log":"logs/compiler-frontend-lexer.log"},
    {"name":"compiler-frontend-parser","status":"pass","duration_seconds":0,"exit_code":0,"command":"go test ./compiler/internal/frontend -run \\^\\$ -fuzz=FuzzParser -fuzztime=2s -parallel=1","log":"logs/compiler-frontend-parser.log"},
    {"name":"compiler-linker-linkcore","status":"pass","duration_seconds":0,"exit_code":0,"command":"go test ./compiler/internal/linker/linkcore -run \\^\\$ -fuzz=FuzzLinkX64ObjectsDoesNotPanic -fuzztime=2s -parallel=1","log":"logs/compiler-linker-linkcore.log"},
    {"name":"http-runtime","status":"pass","duration_seconds":0,"exit_code":0,"command":"go test ./compiler/internal/httprt -run \\^\\$ -fuzz=FuzzHTTPParseRequest -fuzztime=2s -parallel=1","log":"logs/http-runtime.log"},
    {"name":"json-runtime","status":"pass","duration_seconds":0,"exit_code":0,"command":"go test ./compiler/internal/jsonrt -run \\^\\$ -fuzz=FuzzAppendStringProducesValidJSON -fuzztime=2s -parallel=1","log":"logs/json-runtime.log"},
    {"name":"postgres-wire","status":"pass","duration_seconds":0,"exit_code":0,"command":"go test ./compiler/internal/pgrt -run \\^\\$ -fuzz=FuzzReadFrameDoesNotPanic -fuzztime=2s -parallel=1","log":"logs/postgres-wire.log"},
    {"name":"validate-manifest","status":"pass","duration_seconds":0,"exit_code":0,"command":"go test ./tools/cmd/validate-manifest -run \\^\\$ -fuzz=. -fuzztime=2s -parallel=1","log":"logs/validate-manifest.log"},
    {"name":"eco-capsule","status":"pass","duration_seconds":0,"exit_code":0,"command":"go test ./cli/cmd/tetra -run \\^\\$ -fuzz=FuzzParseCapsuleDoesNotPanic -fuzztime=2s -parallel=1","log":"logs/eco-capsule.log"},
    {"name":"property-stress-regressions","status":"pass","duration_seconds":0,"exit_code":0,"command":"go test ./compiler/... ./cli/... ./tools/cmd/validate-manifest -run Fuzz\\|Property\\|Stress -count=1","log":"logs/property-stress-regressions.log"}
  ]
}
`
	if err := os.WriteFile(filepath.Join(dir, "summary.json"), []byte(summaryJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func runFuzzValidator(t *testing.T, reportDir string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command("go", "run", ".", "--report-dir", reportDir)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}
