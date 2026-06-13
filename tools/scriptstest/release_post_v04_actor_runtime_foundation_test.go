package scriptstest

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReleasePostV04ActorRuntimeFoundationGateRunsStrictOrderedEvidence(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "post_v0_4", "actor-runtime-foundation-linux-x64-gate.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read actor runtime foundation gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/post_v0_4/actor-runtime-foundation-linux-x64-gate.sh [--report-dir DIR]",
		`source "$repo_root/scripts/release/surface/report-dir-guard.sh"`,
		`export GOCACHE="$repo_root/.cache/go-build-actor-runtime-foundation-gate"`,
		`export GOTMPDIR="$repo_root/.cache/go-tmp-actor-runtime-foundation-gate"`,
		`mkdir -p "$GOCACHE" "$GOTMPDIR"`,
		`surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "actor_runtime_foundation_gate:"`,
		`distributed_report_dir="$report_dir_arg/distributed-actors-linux-x64"`,
		`parallel_report_dir="$report_dir_arg/parallel-production-linux-x64"`,
		`manifest_path="$report_dir/actor-runtime-foundation-manifest.json"`,
		`bash scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh --report-dir "$distributed_report_dir"`,
		`bash "$script_dir/parallel-production-linux-x64-smoke.sh" --report-dir "$parallel_report_dir"`,
		`go test -buildvcs=false ./cli/cmd/tetra ./compiler/tests/ownership ./compiler -run 'Diagnostic|Actor|Backpressure|Invalid|Closed|Transfer' -count=1`,
		`go test -race -buildvcs=false ./compiler ./cli/internal/actornet -run 'Actor|Broker' -count=1`,
		`go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`,
		`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`,
		`tetra.actor.production_foundation.v1`,
		`parallel-production-linux-x64/parallel-production-linux-x64.json`,
		`distributed-actors-linux-x64/distributed-actors-linux-x64.json`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-actor-runtime-foundation --report-dir "$report_dir" --current-git-head "$git_head"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("actor runtime foundation gate missing %q", want)
		}
	}

	assertOrderedFragments(t, text,
		`distributed-actors-linux-x64-smoke.sh`,
		`parallel-production-linux-x64-smoke.sh`,
		`go test -buildvcs=false ./cli/cmd/tetra ./compiler/tests/ownership ./compiler`,
		`go test -race -buildvcs=false ./compiler ./cli/internal/actornet`,
		`go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`,
		`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`,
		`cat > "$manifest_path" <<MANIFEST`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-actor-runtime-foundation --report-dir "$report_dir" --current-git-head "$git_head"`,
	)
	for _, forbidden := range []string{"continue-on-error", "|| true", "set +e"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("actor runtime foundation gate must not contain bypass marker %q", forbidden)
		}
	}
	for _, forbidden := range []string{"GOCACHE=/tmp", "GOTMPDIR=/tmp", `GOCACHE="$TMPDIR`, `GOTMPDIR="$TMPDIR`} {
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
				if err := os.WriteFile(filepath.Join(reportPath, "stale.json"), []byte("{}\n"), 0o644); err != nil {
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
			want:      "actor_runtime_foundation_gate: refusing unsafe report directory: parent traversal is not accepted",
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
			src: filepath.Join(repo, "scripts", "release", "post_v0_4", "actor-runtime-foundation-linux-x64-gate.sh"),
			dst: filepath.Join(root, "scripts", "release", "post_v0_4", "actor-runtime-foundation-linux-x64-gate.sh"),
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
	return root
}

func runActorRuntimeFoundationGate(t *testing.T, root string, args ...string) ([]byte, error) {
	t.Helper()
	cmdArgs := append([]string{"scripts/release/post_v0_4/actor-runtime-foundation-linux-x64-gate.sh"}, args...)
	cmd := exec.Command("bash", cmdArgs...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GOTELEMETRY=off",
		"GOCACHE="+filepath.Join(root, ".cache", "go-build-actor-foundation-scriptstest"),
		"GOTMPDIR="+filepath.Join(root, ".cache", "go-tmp-actor-foundation-scriptstest"),
	)
	return cmd.CombinedOutput()
}
