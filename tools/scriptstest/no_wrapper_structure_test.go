package scriptstest

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestNoWrapperTargetDirectoriesHaveReadmes(t *testing.T) {
	root := repoRoot(t)
	requiredReadmes := []string{
		"scripts/README.md",
		"scripts/ci/README.md",
		"scripts/dev/README.md",
		"scripts/release/README.md",
		"scripts/tools/README.md",
		"cli/internal/README.md",
	}

	for _, rel := range requiredReadmes {
		info, err := os.Stat(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("%s must exist: %v", rel, err)
		}
		if info.IsDir() {
			t.Fatalf("%s must be a README file, got directory", rel)
		}
		if info.Size() == 0 {
			t.Fatalf("%s must document the target directory boundary", rel)
		}
	}
}

func TestNoWrapperScriptstestDirectoriesHaveReadmes(t *testing.T) {
	assertChildDirectoriesHaveReadmes(t, "tools/scriptstest")
}

func TestNoWrapperReleaseDirectoriesHaveReadmes(t *testing.T) {
	assertChildDirectoriesHaveReadmes(t, "scripts/release")
}

func TestNoWrapperCompilerTestDirectoriesHaveReadmes(t *testing.T) {
	assertChildDirectoriesHaveReadmes(t, "compiler/tests")
	assertChildDirectoriesHaveReadmes(t, "compiler/testdata")
}

func TestNoWrapperToolAndExampleDirectoriesHaveReadmes(t *testing.T) {
	assertChildDirectoriesHaveReadmes(t, "tools/validators")
	assertChildDirectoriesHaveReadmes(t, "examples/smoke")
}

func assertChildDirectoriesHaveReadmes(t *testing.T, parentRel string) {
	t.Helper()

	root := repoRoot(t)
	matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(parentRel), "*"))
	if err != nil {
		t.Fatalf("glob %s child directories: %v", parentRel, err)
	}

	var missing []string
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			t.Fatalf("stat child directory %s: %v", match, err)
		}
		if !info.IsDir() {
			continue
		}

		readmePath := filepath.Join(match, "README.md")
		readmeInfo, err := os.Stat(readmePath)
		if err != nil {
			rel, relErr := filepath.Rel(root, readmePath)
			if relErr != nil {
				t.Fatalf("relativize child README %s: %v", readmePath, relErr)
			}
			missing = append(missing, rel)
			continue
		}
		if readmeInfo.IsDir() || readmeInfo.Size() == 0 {
			rel, relErr := filepath.Rel(root, readmePath)
			if relErr != nil {
				t.Fatalf("relativize child README %s: %v", readmePath, relErr)
			}
			missing = append(missing, rel)
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("every %s child directory must document its boundary: %s", parentRel, strings.Join(missing, ", "))
	}
}

func TestNoWrapperRootScriptEntryPointsAreRemoved(t *testing.T) {
	root := repoRoot(t)
	entries, err := os.ReadDir(filepath.Join(root, "scripts"))
	if err != nil {
		t.Fatalf("read scripts directory: %v", err)
	}

	var rootFiles []string
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "README.md" {
			continue
		}
		path := filepath.Join(root, "scripts", entry.Name())
		info, err := entry.Info()
		if err != nil {
			t.Fatalf("stat scripts root file %s: %v", path, err)
		}
		if strings.HasSuffix(entry.Name(), ".sh") || info.Mode().Perm()&0111 != 0 || hasShebang(t, path) {
			rootFiles = append(rootFiles, filepath.Join("scripts", entry.Name()))
		}
	}
	if len(rootFiles) > 0 {
		sort.Strings(rootFiles)
		t.Fatalf("root-level scripts entrypoint wrappers must be removed; found: %s", strings.Join(rootFiles, ", "))
	}
}

func hasShebang(t *testing.T, path string) bool {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open scripts root file %s: %v", path, err)
	}
	defer file.Close()

	prefix := make([]byte, 2)
	n, err := file.Read(prefix)
	if err != nil && n == 0 {
		return false
	}
	return n == len(prefix) && string(prefix) == "#!"
}

func TestNoWrapperCompilerRootTestsAreDocumentedExceptions(t *testing.T) {
	root := repoRoot(t)
		allowed := map[string]bool{
			"abi_suite_test.go":                  true,
			"actors_test.go":                    true,
			"atomic_suite_test.go":               true,
			"atomic_target_diagnostics_test.go":  true,
			"compiler_pipeline_stage_test.go":   true,
			"compiler_test.go":                  true,
			"distributed_actor_runtime_test.go": true,
			"ffi_target_diagnostics_test.go":     true,
			"filesystem_runtime_test.go":        true,
			"fuzz_suite_test.go":                 true,
			"link_object_contract_test.go":      true,
			"manifest_test.go":                  true,
			"net_runtime_test.go":               true,
		"runtime_override_test.go":          true,
		"task_runtime_test.go":              true,
		"tetra_bug_regression_test.go":      true,
		"wasm_policy_test.go":               true,
		"wasm_runtime_diagnostics_test.go":  true,
	}

	matches, err := filepath.Glob(filepath.Join(root, "compiler", "*_test.go"))
	if err != nil {
		t.Fatalf("glob compiler root tests: %v", err)
	}

	var unexpected []string
	for _, match := range matches {
		name := filepath.Base(match)
		if !allowed[name] {
			unexpected = append(unexpected, filepath.Join("compiler", name))
		}
	}
	if len(unexpected) > 0 {
		sort.Strings(unexpected)
		t.Fatalf("compiler root tests must move under compiler/tests or be documented in compiler/tests/README.md: %s", strings.Join(unexpected, ", "))
	}

	readmePath := filepath.Join(root, "compiler", "tests", "README.md")
	readme, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read compiler/tests README: %v", err)
	}
	for name := range allowed {
		if !strings.Contains(string(readme), "`"+name+"`") {
			t.Fatalf("compiler root test exception %s must be documented in compiler/tests/README.md", name)
		}
	}
}
