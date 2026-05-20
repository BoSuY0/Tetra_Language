package scriptstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV020GateDelegatesWithV020Boundary(t *testing.T) {
	root := repoRoot(t)
	assertLegacyFileRemoved(t, "scripts/release_v0_2_0_gate.sh", "scripts/release/v0_2_0/gate.sh directly")
	raw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "v0_2_0", "gate.sh"))
	if err != nil {
		t.Fatalf("read v0.2.0 release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`repo_root="$(cd "$script_dir/../../.." && pwd)"`,
		"Usage: bash scripts/release/v0_2_0/gate.sh [--report-dir DIR]",
		`bash scripts/dev/bootstrap.sh`,
		`if [[ "$version" != "v0.2.0" ]]`,
		`TETRA_RELEASE_GATE_VERSION=v0.2.0`,
		`TETRA_RELEASE_GATE_ARTIFACT="$release_artifact"`,
		`TETRA_RELEASE_GATE_COMMAND="bash scripts/release/v0_2_0/gate.sh"`,
		`TETRA_RELEASE_GATE_ACTOR_DIAGNOSTIC_CONTAINS="actor declarations currently support state fields and func methods only"`,
		`exec env`,
		`"$repo_root/scripts/release/v0_1_3/gate.sh"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("v0.2.0 release gate missing %q", want)
		}
	}
}

func TestReleaseV020GateRejectsNonDirectoryReportPathBeforeBootstrap(t *testing.T) {
	root := releaseOldGateMinimalRoot(t, "scripts/release/v0_2_0/gate.sh")
	reportDir := filepath.Join(root, "report-file")
	if err := os.WriteFile(reportDir, []byte("not a directory\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	assertOldGateRejectsNonDirectoryReportPath(t, root, "scripts/release/v0_2_0/gate.sh", "release_v0_2_0_gate:", reportDir)
}

func TestReleaseV020GateRejectsDanglingReportDirSymlinkBeforeBootstrap(t *testing.T) {
	root := releaseOldGateMinimalRoot(t, "scripts/release/v0_2_0/gate.sh")
	reportDir := filepath.Join(root, "dangling-report-link")
	if err := os.Symlink(filepath.Join(root, "missing-report-target"), reportDir); err != nil {
		t.Fatal(err)
	}

	assertOldGateRejectsNonDirectoryReportPath(t, root, "scripts/release/v0_2_0/gate.sh", "release_v0_2_0_gate:", reportDir)
}

func TestReleaseV020GateRejectsNonEmptyReportDirBeforeBootstrap(t *testing.T) {
	root := releaseOldGateMinimalRoot(t, "scripts/release/v0_2_0/gate.sh")
	reportDir := filepath.Join(root, "report")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "summary.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	assertOldGateRejectsNonEmptyReportDir(t, root, "scripts/release/v0_2_0/gate.sh", "release_v0_2_0_gate:", reportDir)
}

func TestReleaseV020GateRejectsDashPrefixedNonEmptyReportDirBeforeBootstrap(t *testing.T) {
	root := releaseOldGateMinimalRoot(t, "scripts/release/v0_2_0/gate.sh")
	reportDirArg := "-stale-report"
	reportDir := filepath.Join(root, reportDirArg)
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "summary.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	assertOldGateRejectsNonEmptyReportDirWithArg(t, root, "scripts/release/v0_2_0/gate.sh", "release_v0_2_0_gate:", reportDirArg, reportDir)
}
