package scriptstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV040DistributedActorSmokeScriptRunsExecutableValidator(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "v0_4_0", "distributed-actors-linux-x64-smoke.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read distributed actor smoke script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh [--report-dir DIR]",
		`distributed-actors-linux-x64.json`,
		`go run ./tools/cmd/distributed-actor-runtime-smoke`,
		`go run ./tools/cmd/validate-distributed-actor-runtime`,
		`tetra.actors.distributed-runtime.v1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("distributed actor smoke script missing %q", want)
		}
	}
}
