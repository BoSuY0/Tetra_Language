package tetra_language

import (
	"os/exec"
	"testing"
)

func TestWorkspaceModules(t *testing.T) {
	cmd := exec.Command("go", "test", "./compiler/...", "./cli/...", "./tools/...", "-count=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("workspace module tests failed:\n%s", out)
	}
}
