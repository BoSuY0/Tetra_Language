package release_legacy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV012GateArchivesReleaseStateWithExpectedVersion(t *testing.T) {
	root := repoRoot(t)
	assertLegacyFileRemoved(t, "scripts/release_v0_1_2_gate.sh", "scripts/release/v0_1_2/gate.sh directly")
	raw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "v0_1_2", "gate.sh"))
	if err != nil {
		t.Fatalf("read v0.1.2 release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`check_release_state()`,
		`go run ./tools/cmd/validate-release-state --expected-version "$release_version" --format=json --report-dir "$report_dir" >"$artifacts_dir/release-state.json"`,
		`go run ./tools/cmd/validate-release-state --expected-version "$release_version" --format=text --report-dir "$report_dir" >"$artifacts_dir/release-state.txt"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("v0.1.2 release gate missing release-state expected-version wiring %q", want)
		}
	}
}

func TestReleaseV012GateRejectsMissingReportDirArgument(t *testing.T) {
	root := releaseOldGateMinimalRoot(t, "scripts/release/v0_1_2/gate.sh")

	out, err := runOldReleaseGate(t, root, "scripts/release/v0_1_2/gate.sh", "--report-dir")
	if err == nil {
		t.Fatalf("expected missing report-dir argument rejection\n%s", out)
	}
	if !strings.Contains(string(out), "release_v0_1_2_gate: --report-dir requires a directory") {
		t.Fatalf("missing report-dir argument output missing controlled error:\n%s", out)
	}
}

func TestReleaseV012GateRejectsNonDirectoryReportPathBeforeSideEffects(t *testing.T) {
	root := releaseOldGateMinimalRoot(t, "scripts/release/v0_1_2/gate.sh")
	reportDir := filepath.Join(root, "report-file")
	if err := os.WriteFile(reportDir, []byte("not a directory\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	assertOldGateRejectsNonDirectoryReportPath(t, root, "scripts/release/v0_1_2/gate.sh", "release_v0_1_2_gate:", reportDir)
}

func TestReleaseV012GateRejectsDanglingReportDirSymlinkBeforeSideEffects(t *testing.T) {
	root := releaseOldGateMinimalRoot(t, "scripts/release/v0_1_2/gate.sh")
	reportDir := filepath.Join(root, "dangling-report-link")
	if err := os.Symlink(filepath.Join(root, "missing-report-target"), reportDir); err != nil {
		t.Fatal(err)
	}

	assertOldGateRejectsNonDirectoryReportPath(t, root, "scripts/release/v0_1_2/gate.sh", "release_v0_1_2_gate:", reportDir)
}

func TestReleaseV012GateRejectsNonEmptyReportDirBeforeSideEffects(t *testing.T) {
	root := releaseOldGateMinimalRoot(t, "scripts/release/v0_1_2/gate.sh")
	reportDir := filepath.Join(root, "report")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "summary.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	assertOldGateRejectsNonEmptyReportDir(t, root, "scripts/release/v0_1_2/gate.sh", "release_v0_1_2_gate:", reportDir)
}

func TestReleaseV012GateRejectsDashPrefixedNonEmptyReportDirBeforeSideEffects(t *testing.T) {
	root := releaseOldGateMinimalRoot(t, "scripts/release/v0_1_2/gate.sh")
	reportDirArg := "-stale-report"
	reportDir := filepath.Join(root, reportDirArg)
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "summary.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	assertOldGateRejectsNonEmptyReportDirWithArg(t, root, "scripts/release/v0_1_2/gate.sh", "release_v0_1_2_gate:", reportDirArg, reportDir)
}
