package release_v040

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV040GateUsesDedicatedReadinessPreflight(t *testing.T) {
	root := repoRoot(t)
	assertLegacyFileRemoved(t, "scripts/release_v0_4_0_gate.sh", "scripts/release/v0_4_0/gate.sh directly")
	raw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "v0_4_0", "gate.sh"))
	if err != nil {
		t.Fatalf("read v0.4.0 release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/v0_4_0/gate.sh [--report-dir DIR] [--require-clean]",
		`release_version="v0.4.0"`,
		`release_artifact="tetra.release.v0_4_0.gate-report.v1"`,
		`release_gate_command="bash scripts/release/v0_4_0/gate.sh"`,
		`go run ./cli/cmd/tetra features --format=json >"$features_json"`,
		`go run ./cli/cmd/tetra targets --format=json >"$targets_json"`,
		`run_linux_host_smoke`,
		`distributed-actors-linux-x64-smoke.sh`,
		`native-ui-linux-x64-smoke.sh`,
		`memory-production-linux-x64-smoke.sh`,
		`parallel-production-linux-x64-smoke.sh`,
		`compiler-production-linux-x64-smoke.sh`,
		`TETRA_SECURITY_REVIEW_SIGNOFF`,
		`go run ./tools/cmd/validate-v0-4-readiness`,
		`go run ./tools/cmd/validate-memory-production --report "$report_dir/artifacts/memory-production-linux-x64.json"`,
		`go run ./tools/cmd/validate-parallel-production --report "$report_dir/artifacts/parallel-production-linux-x64.json"`,
		`go run ./tools/cmd/validate-compiler-production --report "$report_dir/artifacts/compiler-production-linux-x64.json"`,
		`check_techempower_reports`,
		`docs/benchmarks/techempower_local_smoke_skip_db_report.json`,
		`docs/benchmarks/techempower_scram_single_query_local_report.json`,
		`docs/benchmarks/techempower_scram_single_query_matrix_local_report.json`,
		`docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json`,
		`go run ./tools/cmd/validate-techempower-report --report "$report" --allow-skip-db`,
		`go run ./tools/cmd/validate-techempower-report --report "$report"`,
		`run_step "techempower report schemas" check_techempower_reports`,
		`go run ./tools/cmd/validate-v0-4-completion-audit --audit docs/release/v0_4_0_completion_audit.md --expected-status achieved`,
		`go run ./tools/cmd/validate-release-gate-summary`,
		`go run ./tools/cmd/validate-v0-4-readiness-blockers`,
		`go run ./tools/cmd/validate-residual-risks`,
		`go run ./tools/cmd/validate-artifact-hashes`,
		`--expected-artifact "$release_artifact"`,
		`--expected-command "$release_gate_command"`,
		`--expected-version "$release_version"`,
		`--scope-decisions docs/release/v0_4_0_scope_decisions.json`,
		`v0.4.0 readiness preflight failed`,
		`memory-production-linux-x64.json`,
		`parallel-production-linux-x64.json`,
		`compiler-production-linux-x64.json`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("v0.4.0 release gate missing %q", want)
		}
	}
	if strings.Contains(text, `go run ./tools/cmd/validate-v0-4-completion-audit --audit docs/release/v0_4_0_completion_audit.md --expected-status not-achieved`) {
		t.Fatalf("v0.4.0 release gate should validate the achieved completion audit, not the pre-completion audit")
	}
}

func TestReleaseV040GateWritesBlockedSummaryOnReadinessFailure(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scripts", "release", "v0_4_0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(filepath.Join(repoRoot(t), "scripts", "release/v0_4_0/gate.sh"), filepath.Join(root, "scripts", "release/v0_4_0/gate.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	tetra := `#!/usr/bin/env bash
set -euo pipefail
case "${1:-}" in
  features)
    printf '{"version":"v0.3.0","features":[]}\n'
    ;;
  targets)
    printf '{"targets":[]}\n'
    ;;
  *)
    echo "unexpected tetra command: $*" >&2
    exit 2
    ;;
esac
`
	if err := os.WriteFile(filepath.Join(root, "tetra"), []byte(tetra), 0o755); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	goTool := `#!/usr/bin/env bash
set -euo pipefail
if [[ "$*" == run\ ./tools/cmd/validate-v0-4-readiness-blockers* ]]; then
  exit 0
fi
if [[ "$*" == run\ ./cli/cmd/tetra\ features* ]]; then
  printf '{"version":"v0.3.0","features":[]}\n'
  exit 0
fi
if [[ "$*" == run\ ./cli/cmd/tetra\ targets* ]]; then
  printf '{"targets":[]}\n'
  exit 0
fi
if [[ "$*" == run\ ./tools/cmd/validate-residual-risks* ]]; then
  exit 0
fi
if [[ "$*" == run\ ./tools/cmd/validate-v0-4-readiness* ]]; then
  echo "fake readiness failure" >&2
  exit 1
fi
if [[ "$*" == run\ ./tools/cmd/validate-release-gate-summary* ]]; then
  exit 0
fi
if [[ "$*" == run\ ./tools/cmd/validate-release-state*--format=json* ]]; then
  printf '{"schema":"tetra.release-state.v1alpha1","status":"fail","expected_version":"v0.4.0"}\n'
  exit 0
fi
if [[ "$*" == run\ ./tools/cmd/validate-release-state*--format=text* ]]; then
  printf 'status: fail\nexpected version: v0.4.0\n'
  exit 0
fi
if [[ "$*" == run\ ./tools/cmd/validate-artifact-hashes*--write* ]]; then
  out=""
  prev=""
  for arg in "$@"; do
    if [[ "$prev" == "--out" ]]; then
      out="$arg"
      break
    fi
    prev="$arg"
  done
  if [[ -z "$out" ]]; then
    echo "missing --out" >&2
    exit 2
  fi
  printf '{"schema":"tetra.release-artifact-hashes.v1alpha1","root":".","artifacts":[{"path":"summary.json","sha256":"sha256:0000000000000000000000000000000000000000000000000000000000000000","size":0}]}\n' >"$out"
  exit 0
fi
if [[ "$*" == run\ ./tools/cmd/validate-artifact-hashes*--manifest* ]]; then
  exit 0
fi
echo "unexpected go command: $*" >&2
exit 2
`
	if err := os.WriteFile(filepath.Join(binDir, "go"), []byte(goTool), 0o755); err != nil {
		t.Fatal(err)
	}

	reportDir := filepath.Join(root, "report")
	cmd := exec.Command("bash", "scripts/release/v0_4_0/gate.sh", "--report-dir", reportDir)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected readiness failure to block v0.4.0 gate\n%s", out)
	}
	if !strings.Contains(string(out), "v0.4.0 readiness preflight failed") {
		t.Fatalf("blocked gate output missing readiness failure:\n%s", out)
	}

	raw, err := os.ReadFile(filepath.Join(reportDir, "summary.json"))
	if err != nil {
		t.Fatalf("blocked gate should write summary.json: %v\n%s", err, out)
	}
	var summary struct {
		Status             string `json:"status"`
		ReleaseVersion     string `json:"release_version"`
		ReleaseArtifact    string `json:"release_artifact"`
		ReleaseGateCommand string `json:"release_gate_command"`
		StartedAt          string `json:"started_at"`
		EndedAt            string `json:"ended_at"`
		StepCount          int    `json:"step_count"`
		FailedCount        int    `json:"failed_count"`
		Steps              []struct {
			Name            string `json:"name"`
			Status          string `json:"status"`
			DurationSeconds int    `json:"duration_seconds"`
			ExitCode        int    `json:"exit_code"`
			Command         string `json:"command"`
			Log             string `json:"log"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(raw, &summary); err != nil {
		t.Fatalf("summary.json is invalid JSON: %v\n%s", err, raw)
	}
	if summary.Status != "blocked" || summary.ReleaseVersion != "v0.4.0" || summary.ReleaseArtifact != "tetra.release.v0_4_0.gate-report.v1" || summary.ReleaseGateCommand != "bash scripts/release/v0_4_0/gate.sh" || summary.StepCount != 1 || summary.FailedCount != 1 || summary.StartedAt == "" || summary.EndedAt == "" {
		t.Fatalf("unexpected blocked summary: %#v\n%s", summary, raw)
	}
	if len(summary.Steps) != 1 {
		t.Fatalf("summary steps = %d, want 1\n%s", len(summary.Steps), raw)
	}
	step := summary.Steps[0]
	if step.Name != "readiness preflight" || step.Status != "fail" || step.ExitCode == 0 || step.Log != "logs/01-readiness-preflight.log" {
		t.Fatalf("unexpected blocked step: %#v\n%s", step, raw)
	}
	logRaw, err := os.ReadFile(filepath.Join(reportDir, filepath.FromSlash(step.Log)))
	if err != nil {
		t.Fatalf("blocked gate should write readiness log: %v", err)
	}
	if !strings.Contains(string(logRaw), "fake readiness failure") {
		t.Fatalf("readiness log missing validator output:\n%s", logRaw)
	}
	for _, path := range []string{
		filepath.Join(reportDir, "artifacts", "release-state.json"),
		filepath.Join(reportDir, "artifacts", "release-state.txt"),
		filepath.Join(reportDir, "artifacts", "readiness-blockers.json"),
		filepath.Join(reportDir, "artifacts", "residual-risks.json"),
		filepath.Join(reportDir, "artifact-hashes.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("blocked gate should write %s: %v", path, err)
		}
	}
	riskRaw, err := os.ReadFile(filepath.Join(reportDir, "artifacts", "residual-risks.json"))
	if err != nil {
		t.Fatalf("read blocked residual-risks.json: %v", err)
	}
	var risks struct {
		Schema         string `json:"schema"`
		ReleaseVersion string `json:"release_version"`
		Artifact       string `json:"artifact"`
		Risks          []struct {
			ID       string `json:"id"`
			Severity string `json:"severity"`
			Owner    string `json:"owner"`
			Status   string `json:"status"`
			Summary  string `json:"summary"`
			Evidence string `json:"evidence"`
		} `json:"risks"`
	}
	if err := json.Unmarshal(riskRaw, &risks); err != nil {
		t.Fatalf("blocked residual-risks.json is invalid JSON: %v\n%s", err, riskRaw)
	}
	if risks.Schema != "tetra.release.residual-risks.v1" || risks.ReleaseVersion != "v0.4.0" || risks.Artifact != "residual-risks.json" {
		t.Fatalf("unexpected blocked residual risk identity: %#v\n%s", risks, riskRaw)
	}
	if len(risks.Risks) == 0 {
		t.Fatalf("blocked residual-risks.json should name readiness blockers:\n%s", riskRaw)
	}
	firstRisk := risks.Risks[0]
	if firstRisk.ID != "v0.4.0-readiness-preflight" || firstRisk.Severity != "critical" || firstRisk.Owner == "" || firstRisk.Status != "blocked" || !strings.Contains(firstRisk.Summary, "readiness preflight") || firstRisk.Evidence != "logs/01-readiness-preflight.log" {
		t.Fatalf("unexpected blocked readiness risk: %#v\n%s", firstRisk, riskRaw)
	}
	blockersRaw, err := os.ReadFile(filepath.Join(reportDir, "artifacts", "readiness-blockers.json"))
	if err != nil {
		t.Fatalf("read readiness-blockers.json: %v", err)
	}
	var blockers struct {
		Schema         string `json:"schema"`
		ReleaseVersion string `json:"release_version"`
		Artifact       string `json:"artifact"`
		SourceLog      string `json:"source_log"`
		Blockers       []struct {
			ID      string `json:"id"`
			Status  string `json:"status"`
			Summary string `json:"summary"`
			Detail  string `json:"detail"`
		} `json:"blockers"`
	}
	if err := json.Unmarshal(blockersRaw, &blockers); err != nil {
		t.Fatalf("readiness-blockers.json is invalid JSON: %v\n%s", err, blockersRaw)
	}
	if blockers.Schema != "tetra.release.v0_4_0.readiness-blockers.v1" || blockers.ReleaseVersion != "v0.4.0" || blockers.Artifact != "readiness-blockers.json" || blockers.SourceLog != "logs/01-readiness-preflight.log" {
		t.Fatalf("unexpected readiness blockers identity: %#v\n%s", blockers, blockersRaw)
	}
	if len(blockers.Blockers) != 1 || blockers.Blockers[0].ID != "readiness-preflight" || blockers.Blockers[0].Status != "blocked" || !strings.Contains(blockers.Blockers[0].Detail, "fake readiness failure") {
		t.Fatalf("unexpected readiness blockers payload: %#v\n%s", blockers.Blockers, blockersRaw)
	}
	summaryMD, err := os.ReadFile(filepath.Join(reportDir, "summary.md"))
	if err != nil {
		t.Fatalf("blocked gate should write summary.md: %v", err)
	}
	if !strings.Contains(string(summaryMD), "readiness preflight") || !strings.Contains(string(summaryMD), "blocked") {
		t.Fatalf("summary.md missing blocked readiness context:\n%s", summaryMD)
	}
}

func TestReleaseV040GateRejectsNonEmptyReportDirBeforeSideEffects(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scripts", "release", "v0_4_0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(filepath.Join(repoRoot(t), "scripts", "release/v0_4_0/gate.sh"), filepath.Join(root, "scripts", "release/v0_4_0/gate.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	reportDir := filepath.Join(root, "report")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "stale-summary.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tetra := `#!/usr/bin/env bash
set -euo pipefail
echo "tetra should not run when report dir is stale" >&2
exit 2
`
	if err := os.WriteFile(filepath.Join(root, "tetra"), []byte(tetra), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", "scripts/release/v0_4_0/gate.sh", "--report-dir", reportDir)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-empty report dir to block gate\n%s", out)
	}
	for _, want := range []string{
		"refusing to reuse non-empty report directory",
		"choose a fresh --report-dir",
		reportDir,
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("non-empty report dir output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(string(out), "tetra should not run") {
		t.Fatalf("gate should reject stale report dir before running tetra:\n%s", out)
	}
}

func TestReleaseV040GateRejectsNonDirectoryReportPathBeforeSideEffects(t *testing.T) {
	root := releaseV040MinimalGateRoot(t)
	reportDir := filepath.Join(root, "report-file")
	if err := os.WriteFile(reportDir, []byte("not a directory\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	assertReleaseV040RejectsNonDirectoryReportPath(t, root, reportDir)
}

func TestReleaseV040GateRejectsDanglingReportDirSymlinkBeforeSideEffects(t *testing.T) {
	root := releaseV040MinimalGateRoot(t)
	reportDir := filepath.Join(root, "dangling-report-link")
	if err := os.Symlink(filepath.Join(root, "missing-report-target"), reportDir); err != nil {
		t.Fatal(err)
	}

	assertReleaseV040RejectsNonDirectoryReportPath(t, root, reportDir)
}

func releaseV040MinimalGateRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scripts", "release", "v0_4_0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(filepath.Join(repoRoot(t), "scripts", "release/v0_4_0/gate.sh"), filepath.Join(root, "scripts", "release/v0_4_0/gate.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	tetra := `#!/usr/bin/env bash
set -euo pipefail
echo "tetra should not run when report dir is invalid" >&2
exit 2
`
	if err := os.WriteFile(filepath.Join(root, "tetra"), []byte(tetra), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func assertReleaseV040RejectsNonDirectoryReportPath(t *testing.T, root, reportDir string) {
	t.Helper()

	cmd := exec.Command("bash", "scripts/release/v0_4_0/gate.sh", "--report-dir", reportDir)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-directory report dir to block gate\n%s", out)
	}
	for _, want := range []string{
		"release_v0_4_0_gate: refusing to use non-directory report path: " + reportDir,
		"release_v0_4_0_gate: choose a fresh --report-dir directory",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("non-directory report dir output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(string(out), "tetra should not run") || strings.Contains(string(out), "mkdir:") {
		t.Fatalf("gate should reject invalid report dir before raw shell or tetra side effects:\n%s", out)
	}
}
