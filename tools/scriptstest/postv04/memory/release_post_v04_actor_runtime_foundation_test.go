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

func TestReleasePostV04ActorRuntimeFoundationGateRunsStrictOrderedEvidence(t *testing.T) {
	root := repoRoot(t)
	contract := loadActorRuntimeFoundationContract(t, root)
	path := filepath.Join(
		root,
		"scripts",
		"release",
		"post_v0_4",
		"actor-runtime-foundation-linux-x64-gate.sh",
	)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read actor runtime foundation gate: %v", err)
	}
	text := string(raw)

	if contract.ID != "actor-runtime-foundation-linux-x64-v1" {
		t.Fatalf("actor runtime foundation contract id = %q", contract.ID)
	}
	if contract.Producer != "scripts/release/post_v0_4/actor-runtime-foundation-linux-x64-gate.sh" ||
		contract.Entrypoint != "scripts/release/post_v0_4/actor-runtime-foundation-linux-x64-gate.sh" {
		t.Fatalf(
			"actor runtime foundation contract producer/entrypoint = %q/%q",
			contract.Producer,
			contract.Entrypoint,
		)
	}
	if contract.Scope != "actor-runtime-foundation-linux-x64" {
		t.Fatalf("actor runtime foundation contract scope = %q", contract.Scope)
	}
	if contract.FreshReportDirPolicy != "require-empty-or-new" {
		t.Fatalf(
			"actor runtime foundation contract fresh_report_dir_policy = %q",
			contract.FreshReportDirPolicy,
		)
	}
	assertEqualOrderedStrings(
		t,
		contract.HostPreconditions,
		[]string{"linux", "go", "fresh-report-dir"},
		"actor runtime foundation host_preconditions",
	)
	if contract.ArtifactHashes == nil || !contract.ArtifactHashes.Enabled ||
		!contract.ArtifactHashes.Required ||
		contract.ArtifactHashes.Algorithm != "sha256" {
		t.Fatalf(
			"actor runtime foundation contract artifact_hashes = %#v, want enabled required sha256",
			contract.ArtifactHashes,
		)
	}
	assertActorRuntimeFoundationRequiredReports(
		t,
		contract,
		actorRuntimeFoundationRequiredReports(),
	)
	assertActorRuntimeFoundationCIArtifacts(
		t,
		contract,
		append(actorRuntimeFoundationRequiredReportPaths(), "logs/*.log"),
	)
	assertEqualOrderedStrings(t, actorRuntimeFoundationValidatorIDs(contract), []string{
		"validate-distributed-actor-runtime",
		"validate-parallel-production",
		"focused-actor-tests",
		"race-actor-slice",
		"validate-manifest",
		"verify-docs",
		"validate-artifact-hashes",
		"validate-actor-runtime-foundation",
	}, "actor runtime foundation validators")
	assertEqualOrderedStrings(t, actorRuntimeFoundationStepIDs(t, contract), []string{
		"distributed-actors-smoke",
		"parallel-production-smoke",
		"focused-actor-tests",
		"race-actor-slice",
		"validate-manifest",
		"verify-docs",
		"write-actor-runtime-foundation-manifest",
		"artifact-hashes-write",
		"artifact-hashes-validate",
		"actor-foundation-validator",
	}, "actor runtime foundation steps")

	for _, want := range []string{
		("Usage: bash scripts/release/post_v0_4/actor-runtime-foundation-" +
			"linux-x64-gate.sh [--report-dir DIR]"),
		`gate_contract="scripts/release/post_v0_4/contracts/actor-runtime-foundation-linux-x64.json"`,
		`source "$repo_root/scripts/release/surface/report-dir-guard.sh"`,
		`export GOCACHE="$repo_root/.cache/go-build-actor-runtime-foundation-gate"`,
		`export GOTMPDIR="$repo_root/.cache/go-tmp-actor-runtime-foundation-gate"`,
		`mkdir -p "$GOCACHE" "$GOTMPDIR"`,
		`go run ./tools/cmd/run-gate`,
		`--contract "$gate_contract"`,
		`--report-dir "$report_dir_arg"`,
		`--dry-run`,
		`> /dev/null`,
		`surface_release_require_fresh_report_dir`,
		`"$report_dir_arg"`,
		`"$repo_root"`,
		`"actor_runtime_foundation_gate:"`,
		`distributed_report_dir="$report_dir_arg/distributed-actors-linux-x64"`,
		`parallel_report_dir="$report_dir_arg/parallel-production-linux-x64"`,
		`manifest_path="$report_dir/actor-runtime-foundation-manifest.json"`,
		`bash scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh`,
		`--report-dir "$distributed_report_dir"`,
		`bash "$script_dir/parallel-production-linux-x64-smoke.sh"`,
		`--report-dir "$parallel_report_dir"`,
		`go test -buildvcs=false`,
		`./cli/cmd/tetra`,
		`./compiler/tests/ownership`,
		`./compiler/tests/ownership/actor_task`,
		`./compiler`,
		`-run 'Diagnostic|Actor|Backpressure|Invalid|Closed|Transfer'`,
		`-count=1`,
		`go test -race -buildvcs=false`,
		`./compiler ./cli/internal/actornet`,
		`-run 'Actor|Broker'`,
		`go run ./tools/cmd/validate-manifest`,
		`--manifest docs/generated/manifest.json`,
		`go run ./tools/cmd/verify-docs`,
		`tetra.actor.production_foundation.v1`,
		`parallel-production-linux-x64/parallel-production-linux-x64.json`,
		`distributed-actors-linux-x64/distributed-actors-linux-x64.json`,
		`go run ./tools/cmd/validate-artifact-hashes`,
		`--write`,
		`--root "$report_dir"`,
		`--out "$report_dir/artifact-hashes.json"`,
		`--manifest "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-actor-runtime-foundation`,
		`--report-dir "$report_dir"`,
		`--current-git-head "$git_head"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("actor runtime foundation gate missing %q", want)
		}
	}

	assertOrderedFragments(
		t,
		text,
		`gate_contract="scripts/release/post_v0_4/contracts/actor-runtime-foundation-linux-x64.json"`,
		`go run ./tools/cmd/run-gate`,
		`--dry-run`,
		`surface_release_require_fresh_report_dir`,
		`"$report_dir_arg"`,
		`distributed-actors-linux-x64-smoke.sh`,
		`--report-dir "$distributed_report_dir"`,
		`parallel-production-linux-x64-smoke.sh`,
		`--report-dir "$parallel_report_dir"`,
		`go test -buildvcs=false`,
		`./compiler/tests/ownership/actor_task`,
		`go test -race -buildvcs=false`,
		`./compiler ./cli/internal/actornet`,
		`go run ./tools/cmd/validate-manifest`,
		`go run ./tools/cmd/verify-docs`,
		`cat > "$manifest_path" << MANIFEST`,
		`go run ./tools/cmd/validate-artifact-hashes`,
		`--write`,
		`go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-actor-runtime-foundation`,
	)
	for _, forbidden := range []string{"continue-on-error", "|| true", "set +e"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("actor runtime foundation gate must not contain bypass marker %q", forbidden)
		}
	}
	for _, forbidden := range []string{
		"GOCACHE=/tmp",
		"GOTMPDIR=/tmp",
		`GOCACHE="$TMPDIR`,
		`GOTMPDIR="$TMPDIR`,
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("actor runtime foundation gate must not use tmpfs cache marker %q", forbidden)
		}
	}
}

func TestReleasePostV04ActorRuntimeFoundationGateRejectsUnsafeReportDirs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash release script test")
	}
	for _, tc := range []struct {
		name      string
		reportRel string
		setup     func(t *testing.T, root string, reportRel string)
		want      string
	}{
		{
			name:      "non-empty",
			reportRel: filepath.ToSlash(filepath.Join("reports", "actor-foundation")),
			setup: func(t *testing.T, root string, reportRel string) {
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
			},
			want: "actor_runtime_foundation_gate: refusing to reuse non-empty report directory: ",
		},
		{
			name:      "symlink",
			reportRel: filepath.ToSlash(filepath.Join("reports", "actor-foundation")),
			setup: func(t *testing.T, root string, reportRel string) {
				target := filepath.Join(root, "target")
				if err := os.MkdirAll(target, 0o755); err != nil {
					t.Fatal(err)
				}
				linkPath := filepath.Join(root, filepath.FromSlash(reportRel))
				if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, linkPath); err != nil {
					t.Fatal(err)
				}
			},
			want: "actor_runtime_foundation_gate: refusing to use symlink report directory: ",
		},
		{
			name:      "parent traversal",
			reportRel: "reports/../escape",
			setup:     func(t *testing.T, root string, reportRel string) {},
			want: ("actor_runtime_foundation_gate: refusing unsafe report " +
				"directory: parent traversal is not accepted"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := actorRuntimeFoundationGateFakeRoot(t)
			tc.setup(t, root, tc.reportRel)
			out, err := runActorRuntimeFoundationGate(t, root, "--report-dir", tc.reportRel)
			if err == nil {
				t.Fatalf("expected report-dir guard rejection\n%s", out)
			}
			if !strings.Contains(string(out), tc.want) {
				t.Fatalf("report-dir guard output missing %q:\n%s", tc.want, out)
			}
		})
	}
}

func actorRuntimeFoundationGateFakeRoot(t *testing.T) string {
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
			src: filepath.Join(
				repo,
				"scripts",
				"release",
				"post_v0_4",
				"actor-runtime-foundation-linux-x64-gate.sh",
			),
			dst: filepath.Join(
				root,
				"scripts",
				"release",
				"post_v0_4",
				"actor-runtime-foundation-linux-x64-gate.sh",
			),
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
	writeActorRuntimeFoundationGateStubGo(t, root)
	return root
}

func runActorRuntimeFoundationGate(t *testing.T, root string, args ...string) ([]byte, error) {
	t.Helper()
	cmdArgs := append(
		[]string{"scripts/release/post_v0_4/actor-runtime-foundation-linux-x64-gate.sh"},
		args...)
	cmd := exec.Command("bash", cmdArgs...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GOTELEMETRY=off",
		"GOCACHE="+filepath.Join(root, ".cache", "go-build-actor-foundation-scriptstest"),
		"GOTMPDIR="+filepath.Join(root, ".cache", "go-tmp-actor-foundation-scriptstest"),
		"PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	return cmd.CombinedOutput()
}

func loadActorRuntimeFoundationContract(t *testing.T, root string) gatecontract.Contract {
	t.Helper()
	contractPath := filepath.Join(
		root,
		"scripts",
		"release",
		"post_v0_4",
		"contracts",
		"actor-runtime-foundation-linux-x64.json",
	)
	contract, err := gatecontract.Load(contractPath)
	if err != nil {
		t.Fatalf("load actor runtime foundation gate contract: %v", err)
	}
	return contract
}

type actorRuntimeFoundationRequiredReport struct {
	path      string
	schema    string
	validator string
}

func actorRuntimeFoundationRequiredReports() []actorRuntimeFoundationRequiredReport {
	return []actorRuntimeFoundationRequiredReport{
		{
			path:      "actor-runtime-foundation-manifest.json",
			schema:    "tetra.actor.production_foundation.v1",
			validator: "validate-actor-runtime-foundation",
		},
		{
			path:      "parallel-production-linux-x64/parallel-production-linux-x64.json",
			schema:    "tetra.parallel.production.v1",
			validator: "validate-parallel-production",
		},
		{
			path:      "parallel-production-linux-x64/artifact-hashes.json",
			schema:    "tetra.release-artifact-hashes.v1alpha1",
			validator: "validate-artifact-hashes",
		},
		{
			path:      "distributed-actors-linux-x64/distributed-actors-linux-x64.json",
			schema:    "tetra.actors.distributed-runtime.v1",
			validator: "validate-distributed-actor-runtime",
		},
		{
			path:      "distributed-actors-linux-x64/artifact-hashes.json",
			schema:    "tetra.release-artifact-hashes.v1alpha1",
			validator: "validate-artifact-hashes",
		},
		{
			path:      "artifact-hashes.json",
			schema:    "tetra.release-artifact-hashes.v1alpha1",
			validator: "validate-artifact-hashes",
		},
	}
}

func actorRuntimeFoundationRequiredReportPaths() []string {
	reports := actorRuntimeFoundationRequiredReports()
	paths := make([]string, 0, len(reports))
	for _, report := range reports {
		paths = append(paths, report.path)
	}
	return paths
}

func assertActorRuntimeFoundationRequiredReports(
	t *testing.T,
	contract gatecontract.Contract,
	want []actorRuntimeFoundationRequiredReport,
) {
	t.Helper()
	got := make([]actorRuntimeFoundationRequiredReport, 0, len(contract.RequiredReports))
	for _, report := range contract.RequiredReports {
		got = append(got, actorRuntimeFoundationRequiredReport{
			path:      report.Path,
			schema:    report.Schema,
			validator: report.Validator,
		})
		if !report.SameCommitRequired {
			t.Fatalf(
				"actor runtime foundation required report %q same_commit_required = false, want true",
				report.Path,
			)
		}
		if !report.ArtifactHashRequired {
			t.Fatalf(
				"actor runtime foundation required report %q artifact_hash_required = false, want true",
				report.Path,
			)
		}
		if len(report.ClaimRefs) == 0 {
			t.Fatalf("actor runtime foundation required report %q claim_refs is empty", report.Path)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("actor runtime foundation required reports = %#v, want %#v", got, want)
	}
}

func assertActorRuntimeFoundationCIArtifacts(
	t *testing.T,
	contract gatecontract.Contract,
	want []string,
) {
	t.Helper()
	got := make([]string, 0, len(contract.CIArtifacts))
	for _, artifact := range contract.CIArtifacts {
		if !artifact.Required {
			t.Fatalf(
				"actor runtime foundation ci_artifacts entry %q required = false, want true",
				artifact.Path,
			)
		}
		got = append(got, artifact.Path)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("actor runtime foundation ci_artifacts paths = %#v, want %#v", got, want)
	}
}

func actorRuntimeFoundationValidatorIDs(contract gatecontract.Contract) []string {
	ids := make([]string, 0, len(contract.Validators))
	for _, validator := range contract.Validators {
		ids = append(ids, validator.ID)
	}
	return ids
}

func actorRuntimeFoundationStepIDs(t *testing.T, contract gatecontract.Contract) []string {
	t.Helper()
	ids := make([]string, 0, len(contract.Steps))
	for _, step := range contract.Steps {
		if !step.Required {
			t.Fatalf("actor runtime foundation step %q required = false, want true", step.ID)
		}
		ids = append(ids, step.ID)
	}
	return ids
}

func writeActorRuntimeFoundationGateStubGo(t *testing.T, root string) {
	t.Helper()
	stub := `#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "run" && "${2:-}" == "./tools/cmd/run-gate" && "$*" == *"--dry-run"* ]]; then
  exit 0
fi
echo "unexpected go invocation in actor foundation fake root: $*" >&2
exit 1
`
	if err := os.WriteFile(filepath.Join(root, "bin", "go"), []byte(stub), 0o755); err != nil {
		t.Fatalf("write fake go stub: %v", err)
	}
}
