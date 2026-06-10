package scriptstest

import (
	"os"
	"path/filepath"
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
}
