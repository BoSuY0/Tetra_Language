package scriptstest

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

func TestReleasePostV04Memory100ProdStableGateRunsStrictOrderedEvidence(t *testing.T) {
	root := repoRoot(t)
	contract := loadMemory100Contract(t, root)
	path := filepath.Join(root, "scripts", "release", "post_v0_4", "memory-100-prod-stable-gate.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Memory100 production stable gate: %v", err)
	}
	text := string(raw)

	if contract.ID != "memory-100-prod-stable-linux-x64-v1" {
		t.Fatalf("Memory100 contract id = %q", contract.ID)
	}
	if contract.Producer != "scripts/release/post_v0_4/memory-100-prod-stable-gate.sh" ||
		contract.Entrypoint != "scripts/release/post_v0_4/memory-100-prod-stable-gate.sh" {
		t.Fatalf("Memory100 contract producer/entrypoint = %q/%q", contract.Producer, contract.Entrypoint)
	}
	if contract.Scope != "memory-100-prod-stable-linux-x64" {
		t.Fatalf("Memory100 contract scope = %q", contract.Scope)
	}
	if contract.FreshReportDirPolicy != "require-empty-or-new" {
		t.Fatalf("Memory100 contract fresh_report_dir_policy = %q", contract.FreshReportDirPolicy)
	}
	assertEqualOrderedStrings(t, contract.HostPreconditions, []string{"linux", "go", "fresh-report-dir"}, "Memory100 contract host_preconditions")
	if contract.ArtifactHashes == nil || !contract.ArtifactHashes.Enabled || !contract.ArtifactHashes.Required || contract.ArtifactHashes.Algorithm != "sha256" {
		t.Fatalf("Memory100 contract artifact_hashes = %#v, want enabled required sha256", contract.ArtifactHashes)
	}
	assertMemory100RequiredReports(t, contract, memory100RequiredReports())
	assertMemory100CIArtifacts(t, contract, memory100RequiredReportPaths())
	assertEqualOrderedStrings(t, memory100ValidatorIDs(contract), []string{
		"validate-memory-production",
		"validate-ram-contract-release",
		"validate-memory-islands-surface-production",
		"validate-memory-fuzz-oracle",
		"validate-artifact-hashes",
		"validate-memory-100-prod-stable",
	}, "Memory100 validators")
	assertEqualOrderedStrings(t, memory100StepIDs(t, contract), []string{
		"memory-production-gate",
		"ram-contract-gate",
		"integrated-gate",
		"write-raw-memory-contract",
		"write-allocation-lowering-report",
		"copy-proof-store-summary",
		"write-proof-transition-report",
		"write-runtime-memory-contract",
		"memory-fuzz-short",
		"memory-fuzz-validator",
		"write-semantic-safety-matrix",
		"write-leak-resource-report",
		"write-docs-claim-policy",
		"write-memory-100-prod-stable-manifest",
		"artifact-hashes-write",
		"artifact-hashes-validate",
		"memory-100-validator",
	}, "Memory100 steps")

	for _, want := range []string{
		"Usage: bash scripts/release/post_v0_4/memory-100-prod-stable-gate.sh [--report-dir DIR]",
		`gate_contract="scripts/release/post_v0_4/contracts/memory-100-prod-stable-linux-x64.json"`,
		`source "$repo_root/scripts/release/surface/report-dir-guard.sh"`,
		`go run ./tools/cmd/run-gate --contract "$gate_contract" --report-dir "$report_dir_arg" --dry-run >/dev/null`,
		`surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "memory100_prod_stable_gate:"`,
		`memory_production_dir="$report_dir_arg/memory-production"`,
		`ram_contract_dir="$report_dir_arg/ram-contract"`,
		`integrated_dir="$report_dir_arg/integrated"`,
		`aggregate_manifest_path="$report_dir/memory-100-prod-stable-manifest.json"`,
		`bash "$script_dir/memory-production-linux-x64-smoke.sh" --report-dir "$memory_production_dir"`,
		`bash "$script_dir/ram-contract-linux-x64-smoke.sh" --report-dir "$ram_contract_dir"`,
		`bash "$script_dir/memory-islands-surface-production-gate.sh" --report-dir "$integrated_dir"`,
		`tetra.memory-100.prod-stable.v1`,
		`memory-production/memory-release-manifest.json`,
		`ram-contract/ram-contract-release-manifest.json`,
		`memory100_verdict="MEMORY100_SCOPED_READY_LOCAL"`,
		`memory100_verdict="MEMORY100_SCOPED_READY_DIRTY"`,
		`"verdict": "$memory100_verdict"`,
		`raw-memory-contract/raw-memory-contract.json`,
		`allocation-lowering/allocation-lowering-report.json`,
		`proof-store/proof-store-summary.json`,
		`proof-transition/proof-transition-report.json`,
		`runtime-memory/runtime-memory-contract.json`,
		`memory-fuzz/memory-fuzz-oracle.json`,
		`leak-resource/leak-resource-report.json`,
		`integrated/memory-islands-surface-production-manifest.json`,
		`docs-manifest/claim-policy.json`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-memory-100-prod-stable --report-dir "$report_dir" --current-git-head "$git_head"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Memory100 production stable gate missing %q", want)
		}
	}

	assertOrderedFragments(t, text,
		`gate_contract="scripts/release/post_v0_4/contracts/memory-100-prod-stable-linux-x64.json"`,
		`go run ./tools/cmd/run-gate --contract "$gate_contract" --report-dir "$report_dir_arg" --dry-run >/dev/null`,
		`surface_release_require_fresh_report_dir "$report_dir_arg"`,
		`memory-production-linux-x64-smoke.sh`,
		`ram-contract-linux-x64-smoke.sh`,
		`memory-islands-surface-production-gate.sh`,
		`cat > "$aggregate_manifest_path" <<MANIFEST`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-memory-100-prod-stable --report-dir "$report_dir" --current-git-head "$git_head"`,
	)
	for _, forbidden := range []string{"continue-on-error", "|| true", "set +e"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Memory100 gate must not contain bypass marker %q", forbidden)
		}
	}
}

func TestReleasePostV04Memory100ProdStableGateRejectsStaleReportDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash release script test")
	}

	root := memory100GateFakeRoot(t)
	reportRel := filepath.ToSlash(filepath.Join("reports", "memory-100", "stale"))
	reportPath := filepath.Join(root, filepath.FromSlash(reportRel))
	if err := os.MkdirAll(reportPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportPath, "stale.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runMemory100Gate(t, root, "--report-dir", reportRel)
	if err == nil {
		t.Fatalf("expected stale report-dir rejection\n%s", out)
	}
	for _, want := range []string{
		"memory100_prod_stable_gate: refusing to reuse non-empty report directory: " + reportRel,
		"memory100_prod_stable_gate: choose a fresh --report-dir so stale reports cannot be reused",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("stale report-dir output missing %q:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{
		"memory-production-linux-x64-smoke.sh",
		"ram-contract-linux-x64-smoke.sh",
		"memory-islands-surface-production-gate.sh",
		"go run",
	} {
		if strings.Contains(string(out), forbidden) {
			t.Fatalf("Memory100 gate should reject stale report-dir before sub-gates:\n%s", out)
		}
	}
}

func memory100GateFakeRoot(t *testing.T) string {
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
			src: filepath.Join(repo, "scripts", "release", "post_v0_4", "memory-100-prod-stable-gate.sh"),
			dst: filepath.Join(root, "scripts", "release", "post_v0_4", "memory-100-prod-stable-gate.sh"),
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
	writeMemory100GateStubGo(t, root)
	return root
}

func runMemory100Gate(t *testing.T, root string, args ...string) ([]byte, error) {
	t.Helper()
	cmdArgs := append([]string{"scripts/release/post_v0_4/memory-100-prod-stable-gate.sh"}, args...)
	cmd := exec.Command("bash", cmdArgs...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GOTELEMETRY=off",
		"GOCACHE="+filepath.Join(root, ".cache", "go-build-memory100-scriptstest"),
		"GOTMPDIR="+filepath.Join(root, ".cache", "go-tmp-memory100-scriptstest"),
		"PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	return cmd.CombinedOutput()
}

func loadMemory100Contract(t *testing.T, root string) gatecontract.Contract {
	t.Helper()
	contractPath := filepath.Join(root, "scripts", "release", "post_v0_4", "contracts", "memory-100-prod-stable-linux-x64.json")
	contract, err := gatecontract.Load(contractPath)
	if err != nil {
		t.Fatalf("load Memory100 gate contract: %v", err)
	}
	return contract
}

type memory100RequiredReport struct {
	path      string
	schema    string
	validator string
}

func memory100RequiredReports() []memory100RequiredReport {
	return []memory100RequiredReport{
		{path: "memory-100-prod-stable-manifest.json", schema: "tetra.memory-100.prod-stable.v1", validator: "validate-memory-100-prod-stable"},
		{path: "memory-production/memory-production-linux-x64.json", schema: "tetra.memory.production.v1", validator: "validate-memory-production"},
		{path: "memory-production/memory-release-manifest.json", schema: "tetra.memory.release-manifest.v1", validator: "validate-memory-production"},
		{path: "memory-production/artifact-hashes.json", schema: "tetra.release-artifact-hashes.v1alpha1", validator: "validate-artifact-hashes"},
		{path: "ram-contract/ram-contract-release-manifest.json", schema: "tetra.ram-contract.release-manifest.v1", validator: "validate-ram-contract-release"},
		{path: "ram-contract/ram-contract-report.json", schema: "tetra.ram-contract-report.v1", validator: "validate-ram-contract-release"},
		{path: "ram-contract/memory-grade-report.json", schema: "tetra.memory-grade-report.v1", validator: "validate-ram-contract-release"},
		{path: "ram-contract/proof-store-summary.json", schema: "tetra.proof-store-summary.v1", validator: "validate-ram-contract-release"},
		{path: "ram-contract/validation-pipeline-coverage.json", schema: "tetra.validation-pipeline-coverage.v1", validator: "validate-ram-contract-release"},
		{path: "ram-contract/heap-blockers.json", schema: "tetra.ram-blockers.v1", validator: "validate-ram-contract-release"},
		{path: "ram-contract/copy-blockers.json", schema: "tetra.ram-blockers.v1", validator: "validate-ram-contract-release"},
		{path: "ram-contract/fuzz/ram-contract-fuzz-oracle.json", schema: "tetra.ram-contract-fuzz-oracle.v1", validator: "validate-ram-contract-release"},
		{path: "ram-contract/artifact-hashes.json", schema: "tetra.release-artifact-hashes.v1alpha1", validator: "validate-artifact-hashes"},
		{path: "raw-memory-contract/raw-memory-contract.json", schema: "tetra.raw-memory-contract.v1", validator: "validate-memory-100-prod-stable"},
		{path: "allocation-lowering/allocation-lowering-report.json", schema: "tetra.allocation-lowering.v1", validator: "validate-memory-100-prod-stable"},
		{path: "proof-store/proof-store-summary.json", schema: "tetra.proof-store-summary.v1", validator: "validate-memory-100-prod-stable"},
		{path: "proof-transition/proof-transition-report.json", schema: "tetra.proof-transition-report.v1", validator: "validate-memory-100-prod-stable"},
		{path: "runtime-memory/runtime-memory-contract.json", schema: "tetra.runtime-memory-contract.v1", validator: "validate-memory-100-prod-stable"},
		{path: "memory-fuzz/memory-fuzz-oracle.json", schema: "tetra.memory-fuzz.oracle.v1", validator: "validate-memory-fuzz-oracle"},
		{path: "memory-fuzz/artifact-hashes.json", schema: "tetra.release-artifact-hashes.v1alpha1", validator: "validate-artifact-hashes"},
		{path: "semantic-safety/memory-semantic-safety-matrix.json", schema: "tetra.memory-semantic-safety-matrix.v1", validator: "validate-memory-100-prod-stable"},
		{path: "leak-resource/leak-resource-report.json", schema: "tetra.leak-resource.v1", validator: "validate-memory-100-prod-stable"},
		{path: "integrated/memory-islands-surface-production-manifest.json", schema: "tetra.memory-islands-surface.production-gate.v1", validator: "validate-memory-islands-surface-production"},
		{path: "integrated/artifact-hashes.json", schema: "tetra.release-artifact-hashes.v1alpha1", validator: "validate-artifact-hashes"},
		{path: "docs-manifest/claim-policy.json", schema: "tetra.memory-100.claim-policy.v1", validator: "validate-memory-100-prod-stable"},
		{path: "artifact-hashes.json", schema: "tetra.release-artifact-hashes.v1alpha1", validator: "validate-artifact-hashes"},
	}
}

func memory100RequiredReportPaths() []string {
	reports := memory100RequiredReports()
	paths := make([]string, 0, len(reports))
	for _, report := range reports {
		paths = append(paths, report.path)
	}
	return paths
}

func assertMemory100RequiredReports(t *testing.T, contract gatecontract.Contract, want []memory100RequiredReport) {
	t.Helper()
	got := make([]memory100RequiredReport, 0, len(contract.RequiredReports))
	for _, report := range contract.RequiredReports {
		got = append(got, memory100RequiredReport{
			path:      report.Path,
			schema:    report.Schema,
			validator: report.Validator,
		})
		if !report.SameCommitRequired {
			t.Fatalf("Memory100 required report %q same_commit_required = false, want true", report.Path)
		}
		if !report.ArtifactHashRequired {
			t.Fatalf("Memory100 required report %q artifact_hash_required = false, want true", report.Path)
		}
		if len(report.ClaimRefs) == 0 {
			t.Fatalf("Memory100 required report %q claim_refs is empty", report.Path)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Memory100 required reports = %#v, want %#v", got, want)
	}
}

func assertMemory100CIArtifacts(t *testing.T, contract gatecontract.Contract, want []string) {
	t.Helper()
	got := make([]string, 0, len(contract.CIArtifacts))
	for _, artifact := range contract.CIArtifacts {
		if !artifact.Required {
			t.Fatalf("Memory100 ci_artifacts entry %q required = false, want true", artifact.Path)
		}
		got = append(got, artifact.Path)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Memory100 ci_artifacts paths = %#v, want %#v", got, want)
	}
}

func memory100ValidatorIDs(contract gatecontract.Contract) []string {
	ids := make([]string, 0, len(contract.Validators))
	for _, validator := range contract.Validators {
		ids = append(ids, validator.ID)
	}
	return ids
}

func memory100StepIDs(t *testing.T, contract gatecontract.Contract) []string {
	t.Helper()
	ids := make([]string, 0, len(contract.Steps))
	for _, step := range contract.Steps {
		if !step.Required {
			t.Fatalf("Memory100 step %q required = false, want true", step.ID)
		}
		ids = append(ids, step.ID)
	}
	return ids
}

func writeMemory100GateStubGo(t *testing.T, root string) {
	t.Helper()
	stub := `#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "run" && "${2:-}" == "./tools/cmd/run-gate" && "$*" == *"--dry-run"* ]]; then
  exit 0
fi
echo "unexpected go invocation in Memory100 stale-dir fake root: $*" >&2
exit 1
`
	if err := os.WriteFile(filepath.Join(root, "bin", "go"), []byte(stub), 0o755); err != nil {
		t.Fatalf("write fake go stub: %v", err)
	}
}
