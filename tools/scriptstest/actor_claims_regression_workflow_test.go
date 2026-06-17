package scriptstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestActorClaimsRegressionWorkflowRunsOnPRAndPushWithoutReleaseGate(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "actor-claims-regression.yml"))
	if err != nil {
		t.Fatalf("read actor claims regression workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"name: actor-claims-regression",
		"  pull_request:",
		"  push:",
		"  workflow_dispatch:",
		"actor-claims-regression-linux:",
		"runs-on: ubuntu-latest",
		"timeout-minutes: 15",
		"actions/checkout@v4",
		"actions/setup-go@v5",
		"go-version: \"1.20.x\"",
		"go run ./tools/cmd/validate-actor-capabilities --manifest docs/contracts/actors/actor-capability-manifest.v1.json",
		"go test ./tools/cmd/validate-actor-capabilities ./tools/validators/actorprod ./tools/validators/actordist ./tools/cmd/validate-actor-runtime-foundation ./tools/cmd/validate-distributed-actor-runtime ./tools/scriptstest -run 'ActorCapability|ActorRuntimeFoundation|DistributedActor|ActorClaimsRegression' -count=1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("actor claims regression workflow missing %q", want)
		}
	}
	section := workflowJobSection(text, "actor-claims-regression-linux:")
	for _, forbidden := range []string{"actor-runtime-foundation-linux-x64-gate.sh", "continue-on-error", "|| true", "set +e", "GOCACHE=/tmp", "GOTMPDIR=/tmp"} {
		if strings.Contains(section, forbidden) {
			t.Fatalf("actor claims regression workflow must not contain release gate, bypass, or tmpfs marker %q", forbidden)
		}
	}
}
