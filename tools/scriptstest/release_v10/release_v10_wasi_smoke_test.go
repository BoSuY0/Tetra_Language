package release_v10

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV10WASISmokeWorkflowLivesInVersionedReleaseScript(t *testing.T) {
	root := repoRoot(t)
	versionedPath := filepath.Join(root, "scripts", "release", "v1_0", "wasi-smoke.sh")
	assertLegacyFileRemoved(
		t,
		"scripts/release_v1_0_wasi_smoke.sh",
		"scripts/release/v1_0/wasi-smoke.sh directly",
	)
	raw, err := os.ReadFile(versionedPath)
	if err != nil {
		t.Fatalf("read versioned wasi smoke script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/v1_0/wasi-smoke.sh",
		`command -v node`,
		`runtime prerequisite unavailable: node`,
		`./tetra smoke --target wasm32-wasi --run=true --report "$report_path"`,
		`go run ./tools/cmd/validate-wasi-smoke-report --mode runtime --report "$report_path"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("scripts/release/v1_0/wasi-smoke.sh missing %q", want)
		}
	}
	assertNoLegacyMention(
		t,
		text,
		"scripts/release_v1_0_wasi_smoke.sh",
		"scripts/release/v1_0/wasi-smoke.sh",
	)
}

func TestReleaseV10WASISmokeScriptChecksNodePrerequisiteBeforeParsingSmokeList(t *testing.T) {
	raw, err := os.ReadFile(
		filepath.Join(repoRoot(t), "scripts", "release", "v1_0", "wasi-smoke.sh"),
	)
	if err != nil {
		t.Fatalf("read wasi smoke script: %v", err)
	}
	text := string(raw)
	guard := strings.Index(text, "command -v node")
	smokeList := strings.Index(
		text,
		`./tetra smoke --list --target wasm32-wasi --format=json >"$smoke_list"`,
	)
	if guard < 0 || smokeList < 0 || guard > smokeList {
		t.Fatalf("wasi smoke must check node before parsing the smoke list")
	}
}

func TestReleaseV10WASISmokeScriptRejectsUISidecarsForDogfood(t *testing.T) {
	raw, err := os.ReadFile(
		filepath.Join(repoRoot(t), "scripts", "release", "v1_0", "wasi-smoke.sh"),
	)
	if err != nil {
		t.Fatalf("read wasi smoke script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`./tetra smoke --list --target wasm32-wasi --format=json >"$smoke_list"`,
		`smoke_source_for_case "$smoke_list" "dogfood_wasi"`,
		`unexpected UI sidecar for WASI dogfood`,
		`smoke_source_for_case "$smoke_list" "ui_web_smoke"`,
		`expected WASI UI metadata sidecar`,
		`unexpected WASI runtime UI sidecar`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("wasi smoke script missing %q", want)
		}
	}
}

func TestReleaseV10WASISmokeUsesUnifiedCLIRuntimePath(t *testing.T) {
	raw, err := os.ReadFile(
		filepath.Join(repoRoot(t), "scripts", "release", "v1_0", "wasi-smoke.sh"),
	)
	if err != nil {
		t.Fatalf("read wasi smoke script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`./tetra smoke --target wasm32-wasi --run=true --report "$report_path"`,
		`go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$report_path"`,
		`go run ./tools/cmd/validate-wasi-smoke-report --mode runtime --report "$report_path"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("wasi smoke script missing unified CLI runtime contract %q", want)
		}
	}
	for _, forbidden := range []string{
		"scripts/tools/run_wasi_smoke_from_report.mjs",
		"node-wasi",
		"missing node helper for WASI smoke harness",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf(
				"wasi smoke script still contains legacy runtime wrapper contract %q",
				forbidden,
			)
		}
	}
}

func TestReleaseV10WASISmokeRejectsMissingReportArgument(t *testing.T) {
	root := releaseV10WASISmokeFakeRepo(t)

	cmd := exec.Command("bash", "scripts/release/v1_0/wasi-smoke.sh", "--report")
	cmd.Dir = root
	cmd.Env = append(
		os.Environ(),
		"PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected missing --report argument rejection\n%s", out)
	}
	if !strings.Contains(string(out), "release/v1_0/wasi-smoke: --report requires a path") {
		t.Fatalf("missing report argument output missing controlled error:\n%s", out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
}

func TestReleaseV10WASISmokeRejectsDirectoryReportBeforeSmokeSideEffects(t *testing.T) {
	root := releaseV10WASISmokeFakeRepo(t)
	reportPath := filepath.Join(root, "report-dir")
	if err := os.MkdirAll(reportPath, 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", "scripts/release/v1_0/wasi-smoke.sh", "--report", reportPath)
	cmd.Dir = root
	cmd.Env = append(
		os.Environ(),
		"PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected directory report path rejection\n%s", out)
	}
	if !strings.Contains(
		string(out),
		"release/v1_0/wasi-smoke: refusing to use directory report path: "+reportPath,
	) {
		t.Fatalf("directory report output missing controlled error:\n%s", out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
	if _, err := os.Stat(filepath.Join(root, "tetra-smoke.log")); !os.IsNotExist(err) {
		t.Fatalf("directory report path should block before smoke, stat err = %v", err)
	}
}

func TestReleaseV10WASISmokeRejectsSymlinkReportBeforeSmokeSideEffects(t *testing.T) {
	root := releaseV10WASISmokeFakeRepo(t)
	targetPath := filepath.Join(root, "wasi-smoke-target.json")
	if err := os.WriteFile(targetPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(root, "wasi-smoke-link.json")
	if err := os.Symlink(targetPath, reportPath); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", "scripts/release/v1_0/wasi-smoke.sh", "--report", reportPath)
	cmd.Dir = root
	cmd.Env = append(
		os.Environ(),
		"PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected symlink report path rejection\n%s", out)
	}
	if !strings.Contains(
		string(out),
		"release/v1_0/wasi-smoke: refusing to use directory report path: "+reportPath,
	) {
		t.Fatalf("symlink report output missing controlled error:\n%s", out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
	if _, err := os.Stat(filepath.Join(root, "tetra-smoke.log")); !os.IsNotExist(err) {
		t.Fatalf("symlink report path should block before smoke, stat err = %v", err)
	}
}

func TestReleaseV10WASISmokeAcceptsDashPrefixedReportPathAndArtifactSidecar(t *testing.T) {
	root := releaseV10WASISmokeFakeRepo(t)
	reportArg := "-wasi-smoke.json"

	cmd := exec.Command("bash", "scripts/release/v1_0/wasi-smoke.sh", "--report", reportArg)
	cmd.Dir = root
	cmd.Env = append(
		os.Environ(),
		"PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dash-prefixed report path should work: %v\n%s", err, out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
	for _, rel := range []string{reportArg, "-wasi-smoke.artifact.json"} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected dash-prefixed output %s: %v\n%s", rel, err, out)
		}
	}
}

func TestReleaseV10WASISmokeRunsUnifiedCLIAndValidatesReport(t *testing.T) {
	root := releaseV10WASISmokeFakeRepo(t)
	reportPath := filepath.Join(root, "report", "wasi-smoke.json")

	cmd := exec.Command("bash", "scripts/release/v1_0/wasi-smoke.sh", "--report", reportPath)
	cmd.Dir = root
	cmd.Env = append(
		os.Environ(),
		"PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wasi smoke should pass with fake unified CLI: %v\n%s", err, out)
	}
	rawReport, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read unified CLI report: %v", err)
	}
	if !strings.Contains(string(rawReport), `"runner":"unified-cli"`) {
		t.Fatalf(
			"final report was not produced by unified CLI runtime path:\n%s",
			string(rawReport),
		)
	}
	rawLog, err := os.ReadFile(filepath.Join(root, "tetra-smoke.log"))
	if err != nil {
		t.Fatalf("read tetra smoke log: %v", err)
	}
	log := string(rawLog)
	for _, want := range []string{
		"--target wasm32-wasi --run=false",
		"--target wasm32-wasi --run=true --report " + reportPath,
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("tetra smoke log missing %q:\n%s", want, log)
		}
	}
	if strings.Contains(log, "run_wasi_smoke_from_report") {
		t.Fatalf("tetra smoke log should not include legacy wrapper:\n%s", log)
	}
	rawValidatorLog, err := os.ReadFile(filepath.Join(root, "validator.log"))
	if err != nil {
		t.Fatalf("read validator log: %v", err)
	}
	if !strings.Contains(
		string(rawValidatorLog),
		"smoke-report-to-checklist --validate-only --report "+reportPath,
	) {
		t.Fatalf("final report was not validated:\n%s", string(rawValidatorLog))
	}
	if !strings.Contains(
		string(rawValidatorLog),
		"validate-wasi-smoke-report --mode runtime --report "+reportPath,
	) {
		t.Fatalf(
			"final WASI runtime report was not strictly validated:\n%s",
			string(rawValidatorLog),
		)
	}
}
