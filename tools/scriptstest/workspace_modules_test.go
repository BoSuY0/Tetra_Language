package scriptstest

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestWorkspaceModules(t *testing.T) {
	if os.Getenv("TETRA_WORKSPACE_MODULES_SUBPROCESS") == "1" {
		t.Skip("skip recursive workspace module check in subprocess")
	}

	root := repoRoot(t)
	modules := []string{"compiler", "cli", "tools"}

	for _, module := range modules {
		module := module
		t.Run(module, func(t *testing.T) {
			cacheDir := filepath.Join(root, ".cache", "go-build-workspace-modules-"+module)
			if err := os.RemoveAll(cacheDir); err != nil {
				t.Fatalf("clean module cache %s: %v", cacheDir, err)
			}
			t.Cleanup(func() {
				if err := os.RemoveAll(cacheDir); err != nil {
					t.Logf("clean module cache %s: %v", cacheDir, err)
				}
			})
			cmd := exec.Command("go", "test", "./...", "-count=1")
			cmd.Dir = filepath.Join(root, module)
			cmd.Env = append(os.Environ(),
				"TETRA_WORKSPACE_MODULES_SUBPROCESS=1",
				"GOCACHE="+cacheDir,
			)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("module %s tests failed:\n%s", module, out)
			}
		})
	}
}
