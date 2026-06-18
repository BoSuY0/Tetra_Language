package workflows

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tetra_language/tools/internal/gatecontract"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func loadSurfaceReleaseContract(t *testing.T, root string) gatecontract.Contract {
	t.Helper()
	contractPath := filepath.Join(
		root,
		"scripts",
		"release",
		"surface",
		"contracts",
		"surface-release-v1.json",
	)
	contract, err := gatecontract.Load(contractPath)
	if err != nil {
		t.Fatalf("load Surface release contract: %v", err)
	}
	return contract
}

func loadMemory100Contract(t *testing.T, root string) gatecontract.Contract {
	t.Helper()
	contractPath := filepath.Join(
		root,
		"scripts",
		"release",
		"post_v0_4",
		"contracts",
		"memory-100-prod-stable-linux-x64.json",
	)
	contract, err := gatecontract.Load(contractPath)
	if err != nil {
		t.Fatalf("load Memory100 gate contract: %v", err)
	}
	return contract
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

func ciArtifactPaths(t *testing.T, contract gatecontract.Contract) []string {
	t.Helper()
	paths := make([]string, 0, len(contract.CIArtifacts))
	for _, artifact := range contract.CIArtifacts {
		if !artifact.Required {
			t.Fatalf("release contract ci_artifacts entry %q must be required", artifact.Path)
		}
		paths = append(paths, artifact.Path)
	}
	return paths
}

func assertOrderedFragments(t *testing.T, text string, fragments ...string) {
	t.Helper()
	last := -1
	for _, fragment := range fragments {
		idx := strings.Index(text, fragment)
		if idx < 0 {
			t.Fatalf("missing ordered fragment %q", fragment)
		}
		if idx < last {
			t.Fatalf("fragment %q appears out of order", fragment)
		}
		last = idx
	}
}
