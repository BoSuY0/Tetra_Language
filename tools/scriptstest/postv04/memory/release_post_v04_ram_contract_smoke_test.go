package postv04_memory

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"tetra_language/tools/internal/gatecontract"
)

func TestReleasePostV04RAMContractSmokeScriptRunsStrictValidators(t *testing.T) {
	root := repoRoot(t)
	contract := loadRAMContract(t, root)
	path := filepath.Join(
		root,
		"scripts",
		"release",
		"post_v0_4",
		"ram-contract-linux-x64-smoke.sh",
	)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read RAM contract smoke script: %v", err)
	}
	text := string(raw)

	if contract.ID != "ram-contract-linux-x64-v1" {
		t.Fatalf("RAM contract id = %q", contract.ID)
	}
	if contract.Producer != "scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh" ||
		contract.Entrypoint != "scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh" {
		t.Fatalf("RAM contract producer/entrypoint = %q/%q", contract.Producer, contract.Entrypoint)
	}
	if contract.Scope != "ram-contract-linux-x64" {
		t.Fatalf("RAM contract scope = %q", contract.Scope)
	}
	if contract.FreshReportDirPolicy != "require-empty-or-new" {
		t.Fatalf("RAM contract fresh_report_dir_policy = %q", contract.FreshReportDirPolicy)
	}
	assertEqualOrderedStrings(
		t,
		contract.HostPreconditions,
		[]string{"linux", "go", "fresh-report-dir"},
		"RAM contract host_preconditions",
	)
	if contract.ArtifactHashes == nil || !contract.ArtifactHashes.Enabled ||
		!contract.ArtifactHashes.Required ||
		contract.ArtifactHashes.Algorithm != "sha256" {
		t.Fatalf(
			"RAM contract artifact_hashes = %#v, want enabled required sha256",
			contract.ArtifactHashes,
		)
	}
	assertRAMContractRequiredReports(t, contract, ramContractRequiredReports())
	assertRAMContractCIArtifacts(t, contract, ramContractRequiredReportPaths())
	assertEqualOrderedStrings(t, ramContractValidatorIDs(contract), []string{
		"validate-ram-contract-report",
		"validate-memory-grade-report",
		"validate-proof-store-summary",
		"validate-validation-pipeline-coverage",
		"validate-heap-blockers",
		"validate-copy-blockers",
		"validate-ram-contract-fuzz-oracle",
		"validate-artifact-hashes",
		"validate-ram-contract-release",
	}, "RAM contract validators")
	assertEqualOrderedStrings(t, ramContractStepIDs(t, contract), []string{
		"ram-contract-build",
		"validate-ram-contract-report",
		"validate-memory-grade-report",
		"validate-proof-store-summary",
		"validate-validation-pipeline-coverage",
		"validate-heap-blockers",
		"validate-copy-blockers",
		"ram-contract-fuzz-short",
		"validate-ram-contract-fuzz-oracle",
		"write-ram-contract-release-manifest",
		"artifact-hashes-write",
		"artifact-hashes-validate",
		"validate-ram-contract-release",
	}, "RAM contract steps")

	for _, want := range []string{
		"Usage: bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh [--report-dir DIR]",
		`gate_contract="scripts/release/post_v0_4/contracts/ram-contract-linux-x64.json"`,
		`source "$repo_root/scripts/release/surface/report-dir-guard.sh"`,
		`go run ./tools/cmd/run-gate`,
		`--contract "$gate_contract"`,
		`--report-dir "$report_dir_arg"`,
		`--dry-run`,
		`> /dev/null`,
		`surface_release_require_fresh_report_dir`,
		`"$report_dir_arg"`,
		`"$repo_root"`,
		`"ram_contract_gate:"`,
		`go run ./cli/cmd/tetra build`,
		`--target linux-x64`,
		`--emit-ram-contract-report`,
		`--emit-memory-report`,
		`--emit-alloc-report`,
		`ram-contract-report.json`,
		`memory-grade-report.json`,
		`proof-store-summary.json`,
		`validation-pipeline-coverage.json`,
		`heap-blockers.json`,
		`copy-blockers.json`,
		`go run ./tools/cmd/validate-ram-contract-report`,
		`--report "$report_dir/ram-contract-report.json"`,
		`go run ./tools/cmd/validate-memory-grade-report`,
		`--report "$report_dir/memory-grade-report.json"`,
		`go run ./tools/cmd/validate-proof-store-summary`,
		`--report "$report_dir/proof-store-summary.json"`,
		`go run ./tools/cmd/validate-validation-pipeline-coverage`,
		`--report "$report_dir/validation-pipeline-coverage.json"`,
		`go run ./tools/cmd/validate-heap-blockers`,
		`--report "$report_dir/heap-blockers.json"`,
		`go run ./tools/cmd/validate-copy-blockers`,
		`--report "$report_dir/copy-blockers.json"`,
		`go run ./tools/cmd/ram-contract-fuzz-short`,
		`--report-dir "$report_dir/fuzz"`,
		`--git-head "$git_head"`,
		`go run ./tools/cmd/validate-ram-contract-fuzz-oracle`,
		`--report "$report_dir/fuzz/ram-contract-fuzz-oracle.json"`,
		`--current-git-head "$git_head"`,
		`go run ./tools/cmd/validate-artifact-hashes`,
		`--write`,
		`--root "$report_dir"`,
		`--out "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-ram-contract-release`,
		`--report-dir "$report_dir"`,
		`tetra.ram-contract.release-manifest.v1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("RAM contract smoke script missing %q", want)
		}
	}
	assertOrderedFragments(
		t,
		text,
		`gate_contract="scripts/release/post_v0_4/contracts/ram-contract-linux-x64.json"`,
		`go run ./tools/cmd/run-gate`,
		`--dry-run`,
		`surface_release_require_fresh_report_dir`,
		`"$report_dir_arg"`,
		`go run ./cli/cmd/tetra build`,
		`--target linux-x64`,
		`--emit-ram-contract-report`,
		`go run ./tools/cmd/validate-ram-contract-report`,
		`go run ./tools/cmd/validate-memory-grade-report`,
		`go run ./tools/cmd/validate-proof-store-summary`,
		`go run ./tools/cmd/validate-validation-pipeline-coverage`,
		`go run ./tools/cmd/validate-heap-blockers`,
		`go run ./tools/cmd/validate-copy-blockers`,
		`go run ./tools/cmd/ram-contract-fuzz-short`,
		`go run ./tools/cmd/validate-ram-contract-fuzz-oracle`,
		`cat > "$manifest_path" << MANIFEST`,
		`go run ./tools/cmd/validate-artifact-hashes`,
		`--write`,
		`go run ./tools/cmd/validate-artifact-hashes --manifest`,
		`go run ./tools/cmd/validate-ram-contract-release`,
	)
}

func TestReleasePostV04RAMContractSmokeRejectsStaleReportDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash release script test")
	}
	root := ramContractGateFakeRoot(t)
	reportRel := filepath.ToSlash(filepath.Join("reports", "ram-contract"))
	reportPath := filepath.Join(root, filepath.FromSlash(reportRel))
	if err := os.MkdirAll(reportPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(reportPath, "stale.json"),
		[]byte("{}\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	out, err := runRAMContractGate(t, root, "--report-dir", reportRel)
	if err == nil {
		t.Fatalf("expected stale report-dir rejection\n%s", out)
	}
	if !strings.Contains(
		string(out),
		"ram_contract_gate: refusing to reuse non-empty report directory: "+reportRel,
	) {
		t.Fatalf("stale report-dir output missing guard:\n%s", out)
	}
}

func ramContractGateFakeRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	repo := repoRoot(t)
	for _, dir := range []string{
		"bin",
		filepath.Join("scripts", "release", "post_v0_4"),
		filepath.Join("scripts", "release", "surface"),
	} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, copy := range []struct {
		src string
		dst string
	}{
		{
			src: filepath.Join(repo, "scripts", "release", "post_v0_4", "ram-contract-linux-x64-smoke.sh"),
			dst: filepath.Join(root, "scripts", "release", "post_v0_4", "ram-contract-linux-x64-smoke.sh"),
		},
		{
			src: filepath.Join(repo, "scripts", "release", "surface", "report-dir-guard.sh"),
			dst: filepath.Join(root, "scripts", "release", "surface", "report-dir-guard.sh"),
		},
	} {
		if err := copyFile(copy.src, copy.dst, 0o755); err != nil {
			t.Fatalf("copy %s: %v", filepath.Base(copy.src), err)
		}
	}
	writeRAMContractGateStubGo(t, root)
	return root
}

func runRAMContractGate(t *testing.T, root string, args ...string) ([]byte, error) {
	t.Helper()
	cmdArgs := append(
		[]string{"scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh"},
		args...)
	cmd := exec.Command("bash", cmdArgs...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GOTELEMETRY=off",
		"GOCACHE="+filepath.Join(root, ".cache", "go-build-ram-scriptstest"),
		"GOTMPDIR="+filepath.Join(root, ".cache", "go-tmp-ram-scriptstest"),
		"PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	return cmd.CombinedOutput()
}

func loadRAMContract(t *testing.T, root string) gatecontract.Contract {
	t.Helper()
	contractPath := filepath.Join(
		root,
		"scripts",
		"release",
		"post_v0_4",
		"contracts",
		"ram-contract-linux-x64.json",
	)
	contract, err := gatecontract.Load(contractPath)
	if err != nil {
		t.Fatalf("load RAM contract gate contract: %v", err)
	}
	return contract
}

type ramContractRequiredReport struct {
	path      string
	schema    string
	validator string
}

func ramContractRequiredReports() []ramContractRequiredReport {
	return []ramContractRequiredReport{
		{
			path:      "ram-contract-release-manifest.json",
			schema:    "tetra.ram-contract.release-manifest.v1",
			validator: "validate-ram-contract-release",
		},
		{
			path:      "ram-contract-report.json",
			schema:    "tetra.ram-contract-report.v1",
			validator: "validate-ram-contract-report",
		},
		{
			path:      "memory-grade-report.json",
			schema:    "tetra.memory-grade-report.v1",
			validator: "validate-memory-grade-report",
		},
		{
			path:      "proof-store-summary.json",
			schema:    "tetra.proof-store-summary.v1",
			validator: "validate-proof-store-summary",
		},
		{
			path:      "validation-pipeline-coverage.json",
			schema:    "tetra.validation-pipeline-coverage.v1",
			validator: "validate-validation-pipeline-coverage",
		},
		{
			path:      "heap-blockers.json",
			schema:    "tetra.ram-blockers.v1",
			validator: "validate-heap-blockers",
		},
		{
			path:      "copy-blockers.json",
			schema:    "tetra.ram-blockers.v1",
			validator: "validate-copy-blockers",
		},
		{
			path:      "fuzz/ram-contract-fuzz-oracle.json",
			schema:    "tetra.ram-contract-fuzz-oracle.v1",
			validator: "validate-ram-contract-fuzz-oracle",
		},
		{
			path:      "artifact-hashes.json",
			schema:    "tetra.release-artifact-hashes.v1alpha1",
			validator: "validate-artifact-hashes",
		},
	}
}

func ramContractRequiredReportPaths() []string {
	reports := ramContractRequiredReports()
	paths := make([]string, 0, len(reports))
	for _, report := range reports {
		paths = append(paths, report.path)
	}
	return paths
}

func assertRAMContractRequiredReports(
	t *testing.T,
	contract gatecontract.Contract,
	want []ramContractRequiredReport,
) {
	t.Helper()
	got := make([]ramContractRequiredReport, 0, len(contract.RequiredReports))
	for _, report := range contract.RequiredReports {
		got = append(got, ramContractRequiredReport{
			path:      report.Path,
			schema:    report.Schema,
			validator: report.Validator,
		})
		if !report.SameCommitRequired {
			t.Fatalf(
				"RAM contract required report %q same_commit_required = false, want true",
				report.Path,
			)
		}
		if !report.ArtifactHashRequired {
			t.Fatalf(
				"RAM contract required report %q artifact_hash_required = false, want true",
				report.Path,
			)
		}
		if len(report.ClaimRefs) == 0 {
			t.Fatalf("RAM contract required report %q claim_refs is empty", report.Path)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RAM contract required reports = %#v, want %#v", got, want)
	}
}

func assertRAMContractCIArtifacts(t *testing.T, contract gatecontract.Contract, want []string) {
	t.Helper()
	got := make([]string, 0, len(contract.CIArtifacts))
	for _, artifact := range contract.CIArtifacts {
		if !artifact.Required {
			t.Fatalf(
				"RAM contract ci_artifacts entry %q required = false, want true",
				artifact.Path,
			)
		}
		got = append(got, artifact.Path)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RAM contract ci_artifacts paths = %#v, want %#v", got, want)
	}
}

func ramContractValidatorIDs(contract gatecontract.Contract) []string {
	ids := make([]string, 0, len(contract.Validators))
	for _, validator := range contract.Validators {
		ids = append(ids, validator.ID)
	}
	return ids
}

func ramContractStepIDs(t *testing.T, contract gatecontract.Contract) []string {
	t.Helper()
	ids := make([]string, 0, len(contract.Steps))
	for _, step := range contract.Steps {
		if !step.Required {
			t.Fatalf("RAM contract step %q required = false, want true", step.ID)
		}
		ids = append(ids, step.ID)
	}
	return ids
}

func writeRAMContractGateStubGo(t *testing.T, root string) {
	t.Helper()
	stub := `#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "run" && "${2:-}" == "./tools/cmd/run-gate" && "$*" == *"--dry-run"* ]]; then
  exit 0
fi
echo "unexpected go invocation in RAM stale-dir fake root: $*" >&2
exit 1
`
	if err := os.WriteFile(filepath.Join(root, "bin", "go"), []byte(stub), 0o755); err != nil {
		t.Fatalf("write fake go stub: %v", err)
	}
}
