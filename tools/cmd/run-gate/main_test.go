package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tetra_language/tools/internal/artifacts"
	"tetra_language/tools/internal/gatecontract"
)

func TestDryRunJSONOutputIncludesOrderedPlan(t *testing.T) {
	repoRoot := t.TempDir()
	t.Chdir(repoRoot)

	contractPath := writeFixtureContract(t, repoRoot, nil)
	reportDir := filepath.Join("reports", "surface-release-v1")

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--contract", contractPath,
		"--report-dir", reportDir,
		"--dry-run",
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d, stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var plan struct {
		Mode     string `json:"mode"`
		Contract struct {
			ID         string `json:"id"`
			Scope      string `json:"scope"`
			Producer   string `json:"producer"`
			Entrypoint string `json:"entrypoint"`
		} `json:"contract"`
		ReportDir               string                           `json:"report_dir"`
		ResolvedReportDir       string                           `json:"resolved_report_dir"`
		Steps                   []gatecontract.Step              `json:"steps"`
		RequiredReports         []gatecontract.RequiredReport    `json:"required_reports"`
		Validators              []gatecontract.Validator         `json:"validators"`
		ArtifactHashes          *gatecontract.ArtifactHashPolicy `json:"artifact_hashes"`
		ArtifactHashCommandPlan struct {
			ManifestPath string `json:"manifest_path"`
			Write        struct {
				Name string   `json:"name"`
				Args []string `json:"args"`
			} `json:"write"`
			Validate struct {
				Name string   `json:"name"`
				Args []string `json:"args"`
			} `json:"validate"`
		} `json:"artifact_hash_command_plan"`
		Claims      []gatecontract.Claim      `json:"claims"`
		Nonclaims   []gatecontract.Nonclaim   `json:"nonclaims"`
		CIArtifacts []gatecontract.CIArtifact `json:"ci_artifacts"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatalf("dry-run stdout is not JSON: %v\n%s", err, stdout.String())
	}

	if plan.Mode != "dry-run" {
		t.Fatalf("mode = %q, want dry-run", plan.Mode)
	}
	if plan.Contract.ID != "fixture-surface-release-v1" {
		t.Fatalf("contract id = %q", plan.Contract.ID)
	}
	if plan.Contract.Scope != "surface-v1-linux-web" {
		t.Fatalf("contract scope = %q", plan.Contract.Scope)
	}
	if plan.Contract.Producer != "scripts/release/surface/release-gate.sh" {
		t.Fatalf("contract producer = %q", plan.Contract.Producer)
	}
	if plan.Contract.Entrypoint != "scripts/release/surface/release-gate.sh" {
		t.Fatalf("contract entrypoint = %q", plan.Contract.Entrypoint)
	}
	if plan.ReportDir != reportDir {
		t.Fatalf("report_dir = %q, want %q", plan.ReportDir, reportDir)
	}

	resolvedReportDir := filepath.Join(repoRoot, "reports", "surface-release-v1")
	if plan.ResolvedReportDir != resolvedReportDir {
		t.Fatalf("resolved_report_dir = %q, want %q", plan.ResolvedReportDir, resolvedReportDir)
	}
	assertStepIDs(
		t,
		plan.Steps,
		[]string{"surface-crash-report-smoke", "validate-surface-release-state"},
	)
	assertReportPaths(
		t,
		plan.RequiredReports,
		[]string{
			"surface-crash-report.json",
			"surface-release-summary.json",
			"artifact-hashes.json",
		},
	)
	assertValidatorIDs(
		t,
		plan.Validators,
		[]string{
			"validate-surface-crash-report",
			"validate-surface-runtime-release-summary",
			"validate-artifact-hashes",
			"validate-surface-release-state",
		},
	)
	if plan.ArtifactHashes == nil || !plan.ArtifactHashes.Enabled ||
		!plan.ArtifactHashes.Required ||
		plan.ArtifactHashes.Algorithm != "sha256" {
		t.Fatalf("artifact_hashes = %#v, want enabled required sha256", plan.ArtifactHashes)
	}

	manifestPath := filepath.Join(resolvedReportDir, artifacts.HashManifestName)
	if plan.ArtifactHashCommandPlan.ManifestPath != manifestPath {
		t.Fatalf(
			"hash manifest path = %q, want %q",
			plan.ArtifactHashCommandPlan.ManifestPath,
			manifestPath,
		)
	}
	wantWriteArgs := []string{
		"go",
		"run",
		"./tools/cmd/validate-artifact-hashes",
		"--write",
		"--root",
		resolvedReportDir,
		"--out",
		manifestPath,
	}
	if !reflect.DeepEqual(plan.ArtifactHashCommandPlan.Write.Args, wantWriteArgs) {
		t.Fatalf(
			"hash write args = %#v, want %#v",
			plan.ArtifactHashCommandPlan.Write.Args,
			wantWriteArgs,
		)
	}
	wantValidateArgs := []string{
		"go",
		"run",
		"./tools/cmd/validate-artifact-hashes",
		"--manifest",
		manifestPath,
	}
	if !reflect.DeepEqual(plan.ArtifactHashCommandPlan.Validate.Args, wantValidateArgs) {
		t.Fatalf(
			"hash validate args = %#v, want %#v",
			plan.ArtifactHashCommandPlan.Validate.Args,
			wantValidateArgs,
		)
	}
	if len(plan.Claims) == 0 || plan.Claims[0].ID != "crash_reporting" {
		t.Fatalf("claims = %#v, want first claim crash_reporting", plan.Claims)
	}
	if len(plan.Nonclaims) == 0 {
		t.Fatalf("nonclaims is empty")
	}
	if len(plan.CIArtifacts) != 3 || plan.CIArtifacts[0].Path != "surface-crash-report.json" {
		t.Fatalf("ci_artifacts = %#v", plan.CIArtifacts)
	}
}

func TestInvalidContractFailsBeforePlanOutput(t *testing.T) {
	repoRoot := t.TempDir()
	t.Chdir(repoRoot)

	contractPath := writeFixtureContract(t, repoRoot, func(doc map[string]any) {
		doc["schema"] = "tetra.gate-contract.v0"
	})

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--contract", contractPath,
		"--report-dir", filepath.Join("reports", "surface-release-v1"),
		"--dry-run",
		"--json",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("run accepted invalid contract")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want no plan output", stdout.String())
	}
	for _, want := range []string{"invalid gate contract", gatecontract.SchemaV1} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr = %q, want substring %q", stderr.String(), want)
		}
	}
}

func TestRunGateExecutionDispatchesEntrypointWithGuardEnvAndReportDir(t *testing.T) {
	repoRoot := t.TempDir()
	t.Chdir(repoRoot)
	entrypoint := filepath.Join("scripts", "fixture-entrypoint.sh")
	writeFixtureEntrypoint(t, repoRoot, entrypoint)
	contractPath := writeFixtureContract(t, repoRoot, func(doc map[string]any) {
		doc["entrypoint"] = filepath.ToSlash(entrypoint)
	})
	reportDir := filepath.Join("reports", "surface-release-v1")

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--contract", contractPath,
		"--report-dir", reportDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "fixture entrypoint stdout") {
		t.Fatalf("stdout = %q, want fixture entrypoint stdout", stdout.String())
	}
	if !strings.Contains(stderr.String(), "fixture entrypoint stderr") {
		t.Fatalf("stderr = %q, want fixture entrypoint stderr", stderr.String())
	}

	raw, err := os.ReadFile(filepath.Join(repoRoot, reportDir, "entrypoint-env.txt"))
	if err != nil {
		t.Fatalf("read entrypoint env marker: %v", err)
	}
	marker := string(raw)
	for _, want := range []string{
		"PWD=" + repoRoot,
		"ARG1=--report-dir",
		"ARG2=" + reportDir,
		"TETRA_RUN_GATE_CONTRACT_EXEC=1",
		"TETRA_RUN_GATE_CONTRACT_ID=fixture-surface-release-v1",
		"TETRA_RUN_GATE_REPORT_DIR=" + reportDir,
	} {
		if !strings.Contains(marker, want+"\n") {
			t.Fatalf("entrypoint marker = %q, want line %q", marker, want)
		}
	}
}

func TestRunGateExecutionRejectsRecursiveSameContractDispatch(t *testing.T) {
	repoRoot := t.TempDir()
	t.Chdir(repoRoot)
	contractPath := writeFixtureContract(t, repoRoot, nil)
	t.Setenv(runGateContractExecEnv, "1")
	t.Setenv(runGateContractIDEnv, "fixture-surface-release-v1")

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--contract", contractPath,
		"--report-dir", filepath.Join("reports", "surface-release-v1"),
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("run exit code = %d, want 2; stderr = %q", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(
		stderr.String(),
		`refusing recursive run-gate execution for contract "fixture-surface-release-v1"`,
	) {
		t.Fatalf("stderr = %q, want recursive dispatch diagnostic", stderr.String())
	}
}

func TestRunGateExecutionRejectsUnsafeContractEntrypoint(t *testing.T) {
	cases := []struct {
		name       string
		entrypoint string
		want       string
	}{
		{
			name:       "absolute",
			entrypoint: "/tmp/release-gate.sh",
			want:       "absolute contract entrypoint is not allowed",
		},
		{
			name:       "parent traversal",
			entrypoint: "../release-gate.sh",
			want:       "parent traversal is not allowed in contract entrypoint",
		},
		{
			name:       "dash prefixed",
			entrypoint: "-c",
			want:       "dash-prefixed contract entrypoint is not allowed",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			t.Chdir(repoRoot)
			contractPath := writeFixtureContract(t, repoRoot, func(doc map[string]any) {
				doc["entrypoint"] = tc.entrypoint
			})

			var stdout, stderr bytes.Buffer
			code := run([]string{
				"--contract", contractPath,
				"--report-dir", filepath.Join("reports", "surface-release-v1"),
			}, &stdout, &stderr)
			if code == 0 {
				t.Fatalf("run accepted unsafe entrypoint %q", tc.entrypoint)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("stderr = %q, want substring %q", stderr.String(), tc.want)
			}
		})
	}
}

func TestMissingRequiredFlagsFail(t *testing.T) {
	repoRoot := t.TempDir()
	t.Chdir(repoRoot)
	contractPath := writeFixtureContract(t, repoRoot, nil)

	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "missing contract",
			args: []string{
				"--report-dir",
				filepath.Join("reports", "surface-release-v1"),
				"--dry-run",
				"--json",
			},
			want: "--contract is required",
		},
		{
			name: "missing report dir",
			args: []string{"--contract", contractPath, "--dry-run", "--json"},
			want: "--report-dir is required",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tc.args, &stdout, &stderr)
			if code == 0 {
				t.Fatalf("run accepted args %#v", tc.args)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("stderr = %q, want substring %q", stderr.String(), tc.want)
			}
		})
	}
}

func TestReportDirValidationRejectsNonEmptyDirectory(t *testing.T) {
	repoRoot := t.TempDir()
	t.Chdir(repoRoot)
	contractPath := writeFixtureContract(t, repoRoot, nil)
	reportDir := filepath.Join("reports", "surface-release-v1")
	if err := os.MkdirAll(filepath.Join(repoRoot, reportDir), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(repoRoot, reportDir, "old.json"),
		[]byte("{}\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--contract", contractPath,
		"--report-dir", reportDir,
		"--dry-run",
		"--json",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("run accepted non-empty report directory")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "non-empty report directory") {
		t.Fatalf("stderr = %q, want non-empty report directory diagnostic", stderr.String())
	}
}

func TestRealSurfaceContractRepresentsReleaseGateRequiredReports(t *testing.T) {
	contract, err := gatecontract.Load(
		filepath.Join(
			"..",
			"..",
			"..",
			"scripts",
			"release",
			"surface",
			"contracts",
			"surface-release-v1.json",
		),
	)
	if err != nil {
		t.Fatalf("Load real Surface contract: %v", err)
	}
	if contract.Entrypoint != "scripts/release/surface/release-gate.sh" {
		t.Fatalf("entrypoint = %q", contract.Entrypoint)
	}
	if contract.ArtifactHashes == nil || !contract.ArtifactHashes.Enabled ||
		!contract.ArtifactHashes.Required ||
		contract.ArtifactHashes.Algorithm != "sha256" {
		t.Fatalf("artifact_hashes = %#v, want enabled required sha256", contract.ArtifactHashes)
	}
	wantReports := []string{
		"surface-headless-release.json",
		"surface-headless-release-text-input.json",
		"surface-headless-release-toolkit.json",
		"surface-headless-release-accessibility.json",
		"surface-headless-app-model.json",
		"surface-linux-x64-release-window.json",
		"surface-linux-x64-release-app-shell.json",
		"surface-dev-workflow.json",
		"surface-inspector.json",
		"surface-template-smoke.json",
		"surface-reference-apps.json",
		"surface-package.json",
		"surface-crash-report.json",
		"surface-i18n.json",
		"surface-widget-migration.json",
		"surface-linux-x64-release-text-input.json",
		"surface-linux-x64-release-toolkit.json",
		"surface-linux-x64-release-accessibility.json",
		"surface-macos-x64-target-host-status.json",
		"surface-windows-x64-target-host-status.json",
		"surface-wasm32-web-release-browser.json",
		"surface-wasm32-web-release-text-input.json",
		"surface-wasm32-web-release-toolkit.json",
		"surface-wasm32-web-release-accessibility.json",
		"block-system/surface-block-system-gate-summary.json",
		"block-system/headless/surface-headless-block-system.json",
		"block-system/headless/surface-block-examples.json",
		"block-system/linux-x64-real-window/surface-block-system-linux-x64.json",
		"block-system/wasm32-web-browser-canvas/surface-block-system-wasm32-web.json",
		"morph/surface-morph-gate-summary.json",
		"morph/headless/surface-headless-morph.json",
		"surface-release-summary.json",
		"artifact-hashes.json",
	}
	assertReportPaths(t, contract.RequiredReports, wantReports)
	if len(contract.RequiredReports) != 33 {
		t.Fatalf("required_reports count = %d, want 33", len(contract.RequiredReports))
	}
	for _, report := range contract.RequiredReports {
		if !report.ArtifactHashRequired {
			t.Fatalf("%s artifact_hash_required = false, want true", report.Path)
		}
		if len(report.ClaimRefs) == 0 {
			t.Fatalf("%s claim_refs is empty", report.Path)
		}
	}
	assertStepIDsContainOrdered(t, contract.Steps, []string{
		"surface-headless-release-smoke",
		"surface-crash-report-smoke",
		"block-system-gate",
		"morph-gate",
		"write-target-host-status-reports",
		"write-surface-release-summary",
		"validate-surface-runtime-release-summary",
		"artifact-hashes-write",
		"artifact-hashes-validate",
		"validate-surface-release-state",
		"validate-surface-claims",
	})
	assertValidatorIDsInclude(t, contract.Validators, []string{
		"validate-surface-runtime-release-summary",
		"validate-surface-security-report",
		"validate-surface-performance-budget",
		"validate-surface-dev-workflow",
		"validate-surface-inspector",
		"validate-surface-template-smoke",
		"validate-surface-reference-apps",
		"validate-surface-package",
		"validate-surface-crash-report",
		"validate-surface-i18n",
		"validate-surface-widget-migration",
		"validate-artifact-hashes",
		"validate-surface-release-state",
		"validate-surface-claims",
	})
	report, ok := findRequiredReport(contract.RequiredReports, "surface-crash-report.json")
	if !ok {
		t.Fatalf("surface-crash-report.json not found in required_reports")
	}
	if !contains(report.ClaimRefs, "crash_reporting") {
		t.Fatalf(
			"surface-crash-report.json claim_refs = %#v, want crash_reporting",
			report.ClaimRefs,
		)
	}
	validator, ok := findValidator(contract.Validators, "validate-surface-crash-report")
	if !ok {
		t.Fatalf("validate-surface-crash-report not found in validators")
	}
	wantCommand := `go run ./tools/cmd/validate-surface-crash-report --report "$REPORT_DIR/surface-crash-report.json"`
	if validator.Command != wantCommand {
		t.Fatalf(
			"validate-surface-crash-report command = %q, want %q",
			validator.Command,
			wantCommand,
		)
	}
	for _, nonclaim := range contract.Nonclaims {
		if nonclaim.ID == "not_full_surface_required_report_contract" {
			t.Fatalf(
				"nonclaim %q should be removed from the full required-report contract",
				nonclaim.ID,
			)
		}
	}
}

func writeFixtureContract(t *testing.T, repoRoot string, mutate func(map[string]any)) string {
	t.Helper()
	doc := map[string]any{
		"schema":                  gatecontract.SchemaV1,
		"id":                      "fixture-surface-release-v1",
		"title":                   "Fixture Surface release gate",
		"scope":                   "surface-v1-linux-web",
		"producer":                "scripts/release/surface/release-gate.sh",
		"entrypoint":              "scripts/release/surface/release-gate.sh",
		"fresh_report_dir_policy": "require-empty-or-new",
		"host_preconditions":      []any{"linux", "go", "fresh-report-dir"},
		"steps": []any{
			map[string]any{
				"id":                    "surface-crash-report-smoke",
				"kind":                  "shell",
				"command":               `bash scripts/release/surface/surface-crash-report-smoke.sh --report-dir "$REPORT_DIR"`,
				"working_dir":           ".",
				"required":              true,
				"report_outputs":        []any{"surface-crash-report.json"},
				"validator_refs":        []any{"validate-surface-crash-report"},
				"host_preconditions":    []any{"linux"},
				"blocked_status_policy": "block-release",
			},
			map[string]any{
				"id":                    "validate-surface-release-state",
				"kind":                  "go-run",
				"command":               `go run ./tools/cmd/validate-surface-release-state --report-dir "$REPORT_DIR" --expected-status current --scope surface-v1-linux-web --manifest docs/generated/manifest.json`,
				"working_dir":           ".",
				"required":              true,
				"report_outputs":        []any{},
				"validator_refs":        []any{"validate-surface-release-state"},
				"host_preconditions":    []any{"linux"},
				"blocked_status_policy": "block-release",
			},
		},
		"required_reports": []any{
			map[string]any{
				"path":                   "surface-crash-report.json",
				"schema":                 "tetra.surface.crash-report.v1",
				"validator":              "validate-surface-crash-report",
				"same_commit_required":   true,
				"artifact_hash_required": true,
				"claim_refs":             []any{"crash_reporting"},
			},
			map[string]any{
				"path":                   "surface-release-summary.json",
				"schema":                 "tetra.surface.release.v1",
				"validator":              "validate-surface-runtime-release-summary",
				"same_commit_required":   true,
				"artifact_hash_required": true,
				"claim_refs":             []any{"surface_release_summary", "release_state_current"},
			},
			map[string]any{
				"path":                   "artifact-hashes.json",
				"schema":                 artifacts.HashManifestSchema,
				"validator":              "validate-artifact-hashes",
				"same_commit_required":   true,
				"artifact_hash_required": true,
				"claim_refs":             []any{"artifact_hash_integrity"},
			},
		},
		"validators": []any{
			map[string]any{
				"id":      "validate-surface-crash-report",
				"kind":    "go-run",
				"command": `go run ./tools/cmd/validate-surface-crash-report --report "$REPORT_DIR/surface-crash-report.json"`,
			},
			map[string]any{
				"id":      "validate-surface-runtime-release-summary",
				"kind":    "go-run",
				"command": `go run ./tools/cmd/validate-surface-runtime --report "$REPORT_DIR/surface-release-summary.json" --release surface-v1`,
			},
			map[string]any{
				"id":      "validate-artifact-hashes",
				"kind":    "go-run",
				"command": `go run ./tools/cmd/validate-artifact-hashes --manifest "$REPORT_DIR/artifact-hashes.json"`,
			},
			map[string]any{
				"id":      "validate-surface-release-state",
				"kind":    "go-run",
				"command": `go run ./tools/cmd/validate-surface-release-state --report-dir "$REPORT_DIR" --expected-status current --scope surface-v1-linux-web --manifest docs/generated/manifest.json`,
			},
		},
		"artifact_hashes": map[string]any{
			"enabled":   true,
			"required":  true,
			"algorithm": "sha256",
		},
		"claims": []any{
			map[string]any{
				"id":        "crash_reporting",
				"statement": "Surface crash reporting evidence is produced and validator checked.",
			},
			map[string]any{
				"id":        "surface_release_summary",
				"statement": "Surface release summary is present and validator checked.",
			},
			map[string]any{
				"id":        "artifact_hash_integrity",
				"statement": "Release reports are covered by a sha256 artifact hash manifest.",
			},
			map[string]any{
				"id":        "release_state_current",
				"statement": "Release-state validation is part of this dry-run contract.",
			},
		},
		"nonclaims": []any{
			map[string]any{
				"id": "not_full_surface_required_report_contract",
				"statement": ("This fixture contract does not enumerate every Surface release-" +
					"gate required report."),
			},
		},
		"ci_artifacts": []any{
			map[string]any{"path": "surface-crash-report.json", "required": true},
			map[string]any{"path": "surface-release-summary.json", "required": true},
			map[string]any{"path": "artifact-hashes.json", "required": true},
		},
	}
	if mutate != nil {
		mutate(doc)
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent fixture: %v", err)
	}
	path := filepath.Join(repoRoot, "contract.json")
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile contract fixture: %v", err)
	}
	return path
}

func writeFixtureEntrypoint(t *testing.T, repoRoot, entrypoint string) {
	t.Helper()
	path := filepath.Join(repoRoot, entrypoint)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll entrypoint dir: %v", err)
	}
	script := `#!/usr/bin/env bash
set -euo pipefail
echo "fixture entrypoint stdout"
echo "fixture entrypoint stderr" >&2
if [[ "$#" -ne 2 || "${1:-}" != "--report-dir" ]]; then
  echo "unexpected args: $*" >&2
  exit 7
fi
mkdir -p "$2"
{
  printf 'PWD=%s\n' "$PWD"
  printf 'ARG1=%s\n' "$1"
  printf 'ARG2=%s\n' "$2"
  printf 'TETRA_RUN_GATE_CONTRACT_EXEC=%s\n' "${TETRA_RUN_GATE_CONTRACT_EXEC:-}"
  printf 'TETRA_RUN_GATE_CONTRACT_ID=%s\n' "${TETRA_RUN_GATE_CONTRACT_ID:-}"
  printf 'TETRA_RUN_GATE_REPORT_DIR=%s\n' "${TETRA_RUN_GATE_REPORT_DIR:-}"
} > "$2/entrypoint-env.txt"
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile entrypoint fixture: %v", err)
	}
}

func assertStepIDs(t *testing.T, steps []gatecontract.Step, want []string) {
	t.Helper()
	got := make([]string, 0, len(steps))
	for _, step := range steps {
		got = append(got, step.ID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("step IDs = %#v, want %#v", got, want)
	}
}

func assertReportPaths(t *testing.T, reports []gatecontract.RequiredReport, want []string) {
	t.Helper()
	got := make([]string, 0, len(reports))
	for _, report := range reports {
		got = append(got, report.Path)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("required report paths = %#v, want %#v", got, want)
	}
}

func assertValidatorIDs(t *testing.T, validators []gatecontract.Validator, want []string) {
	t.Helper()
	got := make([]string, 0, len(validators))
	for _, validator := range validators {
		got = append(got, validator.ID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("validator IDs = %#v, want %#v", got, want)
	}
}

func assertStepIDsContainOrdered(t *testing.T, steps []gatecontract.Step, want []string) {
	t.Helper()
	got := make([]string, 0, len(steps))
	for _, step := range steps {
		got = append(got, step.ID)
	}
	next := 0
	for _, id := range got {
		if next < len(want) && id == want[next] {
			next++
		}
	}
	if next != len(want) {
		t.Fatalf("step IDs = %#v, want ordered subsequence %#v", got, want)
	}
}

func assertValidatorIDsInclude(t *testing.T, validators []gatecontract.Validator, want []string) {
	t.Helper()
	got := make(map[string]struct{}, len(validators))
	for _, validator := range validators {
		got[validator.ID] = struct{}{}
	}
	for _, id := range want {
		if _, ok := got[id]; !ok {
			t.Fatalf("validator IDs missing %q from %#v", id, validators)
		}
	}
}

func findRequiredReport(
	reports []gatecontract.RequiredReport,
	path string,
) (gatecontract.RequiredReport, bool) {
	for _, report := range reports {
		if report.Path == path {
			return report, true
		}
	}
	return gatecontract.RequiredReport{}, false
}

func findValidator(validators []gatecontract.Validator, id string) (gatecontract.Validator, bool) {
	for _, validator := range validators {
		if validator.ID == id {
			return validator, true
		}
	}
	return gatecontract.Validator{}, false
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
