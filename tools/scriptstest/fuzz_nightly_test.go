package scriptstest

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type fuzzNightlySummary struct {
	Mode        string `json:"mode"`
	Status      string `json:"status"`
	ExitCode    int    `json:"exit_code"`
	Fuzztime    string `json:"fuzztime"`
	StepCount   int    `json:"step_count"`
	FailedCount int    `json:"failed_count"`
	Artifacts   struct {
		SummaryMD          string `json:"summary_md"`
		SummaryJSON        string `json:"summary_json"`
		LogsDir            string `json:"logs_dir"`
		UnstableSeedLog    string `json:"unstable_seed_log"`
		CrasherArchivePath string `json:"crasher_archive_path"`
	} `json:"artifacts"`
	Steps []struct {
		Name            string `json:"name"`
		Status          string `json:"status"`
		ExitCode        int    `json:"exit_code"`
		DurationSeconds int    `json:"duration_seconds"`
		Command         string `json:"command"`
		Log             string `json:"log"`
	} `json:"steps"`
}

type fuzzNightlyInventory struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	ScannedRoots  []struct {
		Root       string `json:"root"`
		Exists     bool   `json:"exists"`
		Targets    int    `json:"targets"`
		Corpus     int    `json:"corpus_files"`
		Crashers   int    `json:"crasher_files"`
		TotalFiles int    `json:"total_files"`
	} `json:"scanned_roots"`
	Counts struct {
		Roots        int `json:"roots"`
		Existing     int `json:"existing_roots"`
		Targets      int `json:"targets"`
		CorpusFiles  int `json:"corpus_files"`
		CrasherFiles int `json:"crasher_files"`
		TotalFiles   int `json:"total_files"`
	} `json:"counts"`
}

func TestFuzzNightlyWrapperDocumentsBoundedCommands(t *testing.T) {
	assertLegacyFileRemoved(t, "scripts/fuzz_nightly.sh", "scripts/dev/fuzz-nightly.sh")
	devRaw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "dev", "fuzz-nightly.sh"))
	if err != nil {
		t.Fatalf("read dev fuzz nightly script: %v", err)
	}
	devText := string(devRaw)
	assertNoLegacyMention(t, devText, "scripts/fuzz_nightly.sh", "scripts/dev/fuzz-nightly.sh help")
	if strings.Contains(devText, "bash -c") {
		t.Fatalf("fuzz nightly script must not execute fuzz commands through bash -c")
	}
	for _, want := range []string{
		"--fuzztime",
		"FuzzLexer",
		"FuzzParser",
		"compiler-linker-linkcore",
		"FuzzLinkX64ObjectsDoesNotPanic",
		"FuzzHTTPParseRequest",
		"FuzzAppendStringProducesValidJSON",
		"FuzzReadFrameDoesNotPanic",
		"FuzzParseCapsuleDoesNotPanic",
		"property-stress-regressions",
		"crasher_archive_path",
		"unstable-seeds.md",
	} {
		if !strings.Contains(devText, want) {
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
		"bash scripts/dev/fuzz-nightly.sh --out-dir reports/fuzz-nightly",
		"bash scripts/dev/fuzz-nightly.sh --short",
		"<package>/testdata/fuzz/<FuzzName>/",
		"unstable-seeds.md",
		"deterministic regression test",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("fuzz docs missing %q", want)
		}
	}
}

func TestFuzzNightlyWritesDeterministicCrasherInventory(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create fake bin dir: %v", err)
	}

	fakeGo := filepath.Join(binDir, "go")
	if err := os.WriteFile(fakeGo, []byte(`#!/usr/bin/env bash
set -euo pipefail
printf 'fake go %s\n' "$*" >&2
`), 0o755); err != nil {
		t.Fatalf("write fake go: %v", err)
	}

	workDir := filepath.Join(dir, "work")
	for _, path := range []string{
		filepath.Join(workDir, "compiler/internal/frontend/testdata/fuzz/FuzzLexer/corpus-a"),
		filepath.Join(workDir, "compiler/internal/frontend/testdata/fuzz/FuzzLexer/crashers"),
		filepath.Join(workDir, "compiler/internal/frontend/testdata/fuzz/FuzzParser/corpus-b"),
		filepath.Join(workDir, "cli/cmd/tetra/testdata/fuzz/FuzzParseCapsuleDoesNotPanic/corpus"),
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("create fuzz fixture dir %s: %v", path, err)
		}
	}
	fixtures := map[string]string{
		"compiler/internal/frontend/testdata/fuzz/FuzzLexer/corpus-a/seed-2":                    "go test fuzz v1\n[]byte(\"b\")\n",
		"compiler/internal/frontend/testdata/fuzz/FuzzLexer/crashers/crasher-1":                 "go test fuzz v1\n[]byte(\"boom\")\n",
		"compiler/internal/frontend/testdata/fuzz/FuzzParser/corpus-b/seed-1":                   "go test fuzz v1\n[]byte(\"parse\")\n",
		"cli/cmd/tetra/testdata/fuzz/FuzzParseCapsuleDoesNotPanic/corpus/seed-1":                "go test fuzz v1\n[]byte(\"capsule\")\n",
		"cli/cmd/tetra/testdata/fuzz/FuzzParseCapsuleDoesNotPanic/corpus/not-counted.tmp~":      "scratch\n",
		"cli/cmd/tetra/testdata/fuzz/FuzzParseCapsuleDoesNotPanic/corpus/.not-counted-hidden":   "scratch\n",
		"compiler/internal/frontend/testdata/fuzz/FuzzLexer/crashers/.not-counted-hidden-crash": "scratch\n",
		"compiler/internal/frontend/testdata/fuzz/FuzzLexer/crashers/not-counted-crash.tmp~":    "scratch\n",
	}
	for rel, contents := range fixtures {
		path := filepath.Join(workDir, filepath.FromSlash(rel))
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("write fixture %s: %v", rel, err)
		}
	}

	outDir := filepath.Join(dir, "out")
	cmd := exec.Command("bash", filepath.Join(repoRoot(t), "scripts", "dev", "fuzz-nightly.sh"), "--short", "--out-dir", outDir)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fuzz nightly failed: %v\n%s", err, out)
	}

	raw, err := os.ReadFile(filepath.Join(outDir, "crasher-inventory.json"))
	if err != nil {
		t.Fatalf("read crasher-inventory.json: %v\noutput:\n%s", err, out)
	}
	var inventory fuzzNightlyInventory
	if err := json.Unmarshal(raw, &inventory); err != nil {
		t.Fatalf("unmarshal crasher-inventory.json: %v\n%s", err, raw)
	}
	if inventory.SchemaVersion != 1 || inventory.Kind != "go-testdata-fuzz-inventory" {
		t.Fatalf("unexpected inventory identity: %+v", inventory)
	}
	if len(inventory.ScannedRoots) != 7 || inventory.Counts.Roots != 7 || inventory.Counts.Existing != 2 {
		t.Fatalf("unexpected root counts: counts=%+v roots=%+v", inventory.Counts, inventory.ScannedRoots)
	}
	if inventory.Counts.Targets != 3 || inventory.Counts.CorpusFiles != 3 || inventory.Counts.CrasherFiles != 1 || inventory.Counts.TotalFiles != 4 {
		t.Fatalf("unexpected inventory totals: %+v", inventory.Counts)
	}
	wantRoots := []string{
		"cli/cmd/tetra/testdata/fuzz",
		"compiler/internal/frontend/testdata/fuzz",
		"compiler/internal/httprt/testdata/fuzz",
		"compiler/internal/jsonrt/testdata/fuzz",
		"compiler/internal/linker/linkcore/testdata/fuzz",
		"compiler/internal/pgrt/testdata/fuzz",
		"tools/cmd/validate-manifest/testdata/fuzz",
	}
	for i, want := range wantRoots {
		if inventory.ScannedRoots[i].Root != want {
			t.Fatalf("root order[%d] = %q want %q", i, inventory.ScannedRoots[i].Root, want)
		}
	}
	again, err := os.ReadFile(filepath.Join(outDir, "crasher-inventory.json"))
	if err != nil {
		t.Fatalf("read crasher-inventory.json again: %v", err)
	}
	if string(raw) != string(again) || !strings.Contains(string(raw), `"crasher_files": 1`) {
		t.Fatalf("inventory should be stable and include crasher count:\n%s", raw)
	}
}

func TestFuzzNightlyWritesMachineReadableSummary(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create fake bin dir: %v", err)
	}

	fakeGo := filepath.Join(binDir, "go")
	if err := os.WriteFile(fakeGo, []byte(`#!/usr/bin/env bash
set -euo pipefail
printf 'fake go %s\n' "$*" >&2
`), 0o755); err != nil {
		t.Fatalf("write fake go: %v", err)
	}

	outDir := filepath.Join(dir, "out")
	cmd := exec.Command("bash", filepath.Join(repoRoot(t), "scripts", "dev", "fuzz-nightly.sh"), "--short", "--out-dir", outDir)
	cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fuzz nightly failed: %v\n%s", err, out)
	}

	raw, err := os.ReadFile(filepath.Join(outDir, "summary.json"))
	if err != nil {
		t.Fatalf("read summary.json: %v\noutput:\n%s", err, out)
	}
	var summary fuzzNightlySummary
	if err := json.Unmarshal(raw, &summary); err != nil {
		t.Fatalf("unmarshal summary.json: %v\n%s", err, raw)
	}
	if summary.Mode != "short" || summary.Status != "pass" || summary.ExitCode != 0 || summary.Fuzztime != "2s" {
		t.Fatalf("unexpected top-level summary fields: %+v", summary)
	}
	if summary.StepCount != 9 || summary.FailedCount != 0 || len(summary.Steps) != summary.StepCount {
		t.Fatalf("unexpected summary counts: step_count=%d failed_count=%d len=%d", summary.StepCount, summary.FailedCount, len(summary.Steps))
	}
	if summary.Artifacts.SummaryMD != filepath.Join(outDir, "summary.md") ||
		summary.Artifacts.SummaryJSON != filepath.Join(outDir, "summary.json") ||
		summary.Artifacts.LogsDir != filepath.Join(outDir, "logs") ||
		summary.Artifacts.UnstableSeedLog != filepath.Join(outDir, "unstable-seeds.md") ||
		summary.Artifacts.CrasherArchivePath != "<package>/testdata/fuzz/<FuzzName>/" {
		t.Fatalf("unexpected artifact paths: %+v", summary.Artifacts)
	}
	for _, step := range summary.Steps {
		if step.Status != "pass" || step.ExitCode != 0 {
			t.Fatalf("unexpected step status: %+v", step)
		}
		if step.Name == "" || step.Command == "" || !strings.HasPrefix(step.Log, "logs/") {
			t.Fatalf("step missing stable metadata: %+v", step)
		}
		if step.DurationSeconds < 0 {
			t.Fatalf("step has negative duration: %+v", step)
		}
	}
}

func TestFuzzNightlySummaryEscapesControlCharacters(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create fake bin dir: %v", err)
	}

	fakeGo := filepath.Join(binDir, "go")
	if err := os.WriteFile(fakeGo, []byte(`#!/usr/bin/env bash
set -euo pipefail
printf 'fake go %s\n' "$*" >&2
`), 0o755); err != nil {
		t.Fatalf("write fake go: %v", err)
	}

	fuzztime := "2s"
	outDir := filepath.Join(dir, "out\twith\rcontrol\x01")
	cmd := exec.Command("bash", filepath.Join(repoRoot(t), "scripts", "dev", "fuzz-nightly.sh"), "--fuzztime", fuzztime, "--out-dir", outDir)
	cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fuzz nightly failed: %v\n%s", err, out)
	}

	raw, err := os.ReadFile(filepath.Join(outDir, "summary.json"))
	if err != nil {
		t.Fatalf("read summary.json: %v\noutput:\n%s", err, out)
	}
	var summary fuzzNightlySummary
	if err := json.Unmarshal(raw, &summary); err != nil {
		t.Fatalf("summary.json with tab/CR paths and strings must remain valid JSON: %v\n%s", err, raw)
	}
	if summary.Fuzztime != fuzztime {
		t.Fatalf("fuzztime did not round-trip: got %q want %q", summary.Fuzztime, fuzztime)
	}
	if summary.Artifacts.SummaryJSON != filepath.Join(outDir, "summary.json") {
		t.Fatalf("summary_json path did not round-trip control characters: got %q want %q", summary.Artifacts.SummaryJSON, filepath.Join(outDir, "summary.json"))
	}
	if len(summary.Steps) != summary.StepCount {
		t.Fatalf("unexpected summary step count: step_count=%d len=%d", summary.StepCount, len(summary.Steps))
	}
}

func TestFuzzNightlyFailureSummaryRecordsExitCodes(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create fake bin dir: %v", err)
	}

	fakeGo := filepath.Join(binDir, "go")
	if err := os.WriteFile(fakeGo, []byte(`#!/usr/bin/env bash
set -euo pipefail
case "$*" in
  *FuzzParser*) exit 23 ;;
esac
`), 0o755); err != nil {
		t.Fatalf("write fake go: %v", err)
	}

	outDir := filepath.Join(dir, "out")
	cmd := exec.Command("bash", filepath.Join(repoRoot(t), "scripts", "dev", "fuzz-nightly.sh"), "--short", "--out-dir", outDir)
	cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected failing fuzz nightly run\n%s", out)
	}

	raw, readErr := os.ReadFile(filepath.Join(outDir, "summary.json"))
	if readErr != nil {
		t.Fatalf("read failure summary.json: %v\noutput:\n%s", readErr, out)
	}
	var summary fuzzNightlySummary
	if unmarshalErr := json.Unmarshal(raw, &summary); unmarshalErr != nil {
		t.Fatalf("unmarshal failure summary.json: %v\n%s", unmarshalErr, raw)
	}
	if summary.Status != "fail" || summary.ExitCode != 1 || summary.FailedCount != 1 {
		t.Fatalf("unexpected failure summary: %+v", summary)
	}
	var sawParser bool
	var sawLaterStep bool
	for _, step := range summary.Steps {
		if step.Name == "compiler-frontend-parser" {
			sawParser = true
			if step.Status != "fail" || step.ExitCode != 23 {
				t.Fatalf("parser failure step = %+v", step)
			}
		}
		if step.Name == "property-stress-regressions" && step.Status == "pass" && step.ExitCode == 0 {
			sawLaterStep = true
		}
	}
	if !sawParser || !sawLaterStep {
		t.Fatalf("failure summary did not record failing and later steps: %+v", summary.Steps)
	}
}

func TestFuzzNightlyRejectsInvalidFuzztimeBeforeExecution(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create fake bin dir: %v", err)
	}

	goRan := filepath.Join(dir, "go-ran")
	fakeGo := filepath.Join(binDir, "go")
	if err := os.WriteFile(fakeGo, []byte(`#!/usr/bin/env bash
set -euo pipefail
touch "$FUZZ_NIGHTLY_FAKE_GO_RAN"
`), 0o755); err != nil {
		t.Fatalf("write fake go: %v", err)
	}

	sentinel := filepath.Join(dir, "fuzztime-injection-created")
	tests := []struct {
		name     string
		fuzztime string
	}{
		{name: "empty", fuzztime: ""},
		{name: "nonsense", fuzztime: "tomorrow"},
		{name: "bad decimal", fuzztime: "1.5.0s"},
		{name: "sub second below min", fuzztime: "999ms"},
		{name: "zero", fuzztime: "0s"},
		{name: "negative", fuzztime: "-1s"},
		{name: "control characters", fuzztime: "1s\twith\rcontrol\x01"},
		{name: "shell metacharacters", fuzztime: "1s; touch " + sentinel + " #"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.Remove(goRan); err != nil && !os.IsNotExist(err) {
				t.Fatalf("remove fake go marker: %v", err)
			}
			outDir := filepath.Join(dir, "out-"+strings.ReplaceAll(tt.name, " ", "-"))
			cmd := exec.Command("bash", filepath.Join(repoRoot(t), "scripts", "dev", "fuzz-nightly.sh"), "--fuzztime", tt.fuzztime, "--out-dir", outDir)
			cmd.Env = append(os.Environ(),
				"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
				"FUZZ_NIGHTLY_FAKE_GO_RAN="+goRan,
			)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("expected invalid fuzztime rejection\n%s", out)
			}
			exitErr, ok := err.(*exec.ExitError)
			if !ok || exitErr.ExitCode() != 2 {
				t.Fatalf("expected invalid fuzztime exit 2, got %v\n%s", err, out)
			}
			if !strings.Contains(string(out), "invalid --fuzztime") {
				t.Fatalf("expected invalid fuzztime message, got:\n%s", out)
			}
			if _, err := os.Stat(goRan); err == nil {
				t.Fatalf("fake go ran for invalid fuzztime %q\noutput:\n%s", tt.fuzztime, out)
			} else if !os.IsNotExist(err) {
				t.Fatalf("stat fake go marker: %v", err)
			}
			if _, err := os.Stat(sentinel); err == nil {
				t.Fatalf("fuzztime was evaluated by a shell and created %s\noutput:\n%s", sentinel, out)
			} else if !os.IsNotExist(err) {
				t.Fatalf("stat sentinel: %v", err)
			}
		})
	}
}

func TestFuzzNightlyRejectsFuzztimeAbovePolicyBoundBeforeExecution(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create fake bin dir: %v", err)
	}

	goRan := filepath.Join(dir, "go-ran")
	fakeGo := filepath.Join(binDir, "go")
	if err := os.WriteFile(fakeGo, []byte(`#!/usr/bin/env bash
set -euo pipefail
touch "$FUZZ_NIGHTLY_FAKE_GO_RAN"
`), 0o755); err != nil {
		t.Fatalf("write fake go: %v", err)
	}

	outDir := filepath.Join(dir, "out")
	cmd := exec.Command("bash", filepath.Join(repoRoot(t), "scripts", "dev", "fuzz-nightly.sh"), "--fuzztime", "10m1s", "--out-dir", outDir)
	cmd.Env = append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FUZZ_NIGHTLY_FAKE_GO_RAN="+goRan,
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected out-of-policy fuzztime rejection\n%s", out)
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 2 {
		t.Fatalf("expected out-of-policy fuzztime exit 2, got %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "invalid --fuzztime") || !strings.Contains(string(out), "max 10m") {
		t.Fatalf("expected max bound message, got:\n%s", out)
	}
	if _, err := os.Stat(goRan); err == nil {
		t.Fatalf("fake go ran for out-of-policy fuzztime\noutput:\n%s", out)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat fake go marker: %v", err)
	}
}

func TestFuzzNightlyRejectsStaleNonEmptyOutDir(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create fake bin dir: %v", err)
	}
	fakeGo := filepath.Join(binDir, "go")
	if err := os.WriteFile(fakeGo, []byte(`#!/usr/bin/env bash
set -euo pipefail
printf 'fake go should not run for stale out-dir\n' >&2
`), 0o755); err != nil {
		t.Fatalf("write fake go: %v", err)
	}

	tests := []struct {
		name   string
		outDir string
		work   string
	}{
		{
			name:   "existing non-empty out-dir",
			outDir: filepath.Join(dir, "stale-out"),
			work:   dir,
		},
		{
			name:   "symlink out-dir to non-empty target",
			outDir: filepath.Join(dir, "stale-link"),
			work:   dir,
		},
		{
			name:   "dash-prefixed out-dir",
			outDir: "-stale-out",
			work:   filepath.Join(dir, "dash-work"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.MkdirAll(tt.work, 0o755); err != nil {
				t.Fatalf("create work dir: %v", err)
			}
			switch tt.name {
			case "symlink out-dir to non-empty target":
				targetDir := filepath.Join(dir, "stale-target")
				if err := os.MkdirAll(targetDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(targetDir, "summary.json"), []byte("{}\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(targetDir, tt.outDir); err != nil {
					t.Fatalf("create out-dir symlink: %v", err)
				}
			default:
				staleDir := tt.outDir
				if !filepath.IsAbs(staleDir) {
					staleDir = filepath.Join(tt.work, staleDir)
				}
				if err := os.MkdirAll(staleDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(staleDir, "summary.json"), []byte("{}\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			cmd := exec.Command("bash", filepath.Join(repoRoot(t), "scripts", "dev", "fuzz-nightly.sh"), "--short", "--out-dir", tt.outDir)
			cmd.Dir = tt.work
			cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("expected stale out-dir rejection\n%s", out)
			}
			exitErr, ok := err.(*exec.ExitError)
			if !ok || exitErr.ExitCode() != 2 {
				t.Fatalf("expected stale out-dir exit 2, got %v\n%s", err, out)
			}
			if !strings.Contains(string(out), "refusing to reuse non-empty out-dir: "+tt.outDir) {
				t.Fatalf("unexpected stale out-dir output:\n%s", out)
			}
			if strings.Contains(string(out), "find:") {
				t.Fatalf("dash-prefixed out-dir should not be parsed as a find option:\n%s", out)
			}
			assertOutputAvoidsRawPathUtilityErrors(t, out)
		})
	}
}

func TestFuzzNightlyAcceptsDashPrefixedFreshOutDir(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create fake bin dir: %v", err)
	}
	goRan := filepath.Join(dir, "go-ran")
	fakeGo := filepath.Join(binDir, "go")
	if err := os.WriteFile(fakeGo, []byte(`#!/usr/bin/env bash
set -euo pipefail
printf 'fake go %s\n' "$*" >&2
printf 'ran\n' >>"`+goRan+`"
`), 0o755); err != nil {
		t.Fatalf("write fake go: %v", err)
	}

	work := filepath.Join(dir, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatalf("create work dir: %v", err)
	}
	outDir := "-fresh-out"
	cmd := exec.Command("bash", filepath.Join(repoRoot(t), "scripts", "dev", "fuzz-nightly.sh"), "--short", "--out-dir", outDir)
	cmd.Dir = work
	cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fuzz nightly should accept dash-prefixed fresh out-dir: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "invalid option") || strings.Contains(string(out), "find:") {
		t.Fatalf("dash-prefixed out-dir should not be parsed as a command option:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(work, outDir, "summary.json")); err != nil {
		t.Fatalf("expected summary.json in dash-prefixed out-dir: %v\noutput:\n%s", err, out)
	}
	if _, err := os.Stat(goRan); err != nil {
		t.Fatalf("expected fake go to run for fresh out-dir: %v\noutput:\n%s", err, out)
	}
}

func TestFuzzNightlyRejectsUnsafeOutDirBeforeExecution(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create fake bin dir: %v", err)
	}
	goRan := filepath.Join(dir, "go-ran")
	fakeGo := filepath.Join(binDir, "go")
	if err := os.WriteFile(fakeGo, []byte(`#!/usr/bin/env bash
set -euo pipefail
printf 'fake go should not run for unsafe out-dir\n' >&2
printf 'ran\n' >>"`+goRan+`"
`), 0o755); err != nil {
		t.Fatalf("write fake go: %v", err)
	}

	tests := []struct {
		name        string
		outDir      string
		setup       func(t *testing.T, work string, outDir string)
		wantMessage string
	}{
		{
			name:   "existing file",
			outDir: filepath.Join(dir, "out-file"),
			setup: func(t *testing.T, work string, outDir string) {
				t.Helper()
				if err := os.WriteFile(outDir, []byte("not a directory\n"), 0o644); err != nil {
					t.Fatalf("write out-dir file: %v", err)
				}
			},
			wantMessage: "refusing to use non-directory out-dir:",
		},
		{
			name:   "symlink to empty directory",
			outDir: filepath.Join(dir, "out-link"),
			setup: func(t *testing.T, work string, outDir string) {
				t.Helper()
				targetDir := filepath.Join(dir, "empty-target")
				if err := os.MkdirAll(targetDir, 0o755); err != nil {
					t.Fatalf("create symlink target: %v", err)
				}
				if err := os.Symlink(targetDir, outDir); err != nil {
					t.Fatalf("create out-dir symlink: %v", err)
				}
			},
			wantMessage: "refusing to use symlink out-dir:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			work := filepath.Join(dir, "work-"+strings.ReplaceAll(tt.name, " ", "-"))
			if err := os.MkdirAll(work, 0o755); err != nil {
				t.Fatalf("create work dir: %v", err)
			}
			tt.setup(t, work, tt.outDir)

			cmd := exec.Command("bash", filepath.Join(repoRoot(t), "scripts", "dev", "fuzz-nightly.sh"), "--short", "--out-dir", tt.outDir)
			cmd.Dir = work
			cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("expected unsafe out-dir rejection\n%s", out)
			}
			exitErr, ok := err.(*exec.ExitError)
			if !ok || exitErr.ExitCode() != 2 {
				t.Fatalf("expected unsafe out-dir exit 2, got %v\n%s", err, out)
			}
			if !strings.Contains(string(out), tt.wantMessage) || !strings.Contains(string(out), tt.outDir) {
				t.Fatalf("unexpected unsafe out-dir output:\n%s", out)
			}
			assertOutputAvoidsRawPathUtilityErrors(t, out)
			if _, err := os.Stat(goRan); err == nil {
				t.Fatalf("fake go ran for unsafe out-dir\noutput:\n%s", out)
			} else if !os.IsNotExist(err) {
				t.Fatalf("stat fake go marker: %v", err)
			}
		})
	}
}
