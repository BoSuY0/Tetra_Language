package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckCommandUsesDefaultMainT4(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.t4"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"check"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Checked: main.t4") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestCheckCommandDiscoversCapsuleT4ProjectEntryAndSourceRoots(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    entry "src/app/main.t4"

    sources:
        src
        ui

    targets:
        linux

    allow:
        ui

    policy:
        unsafe deny
        reproducible required
`)
	writeCLIProjectFile(t, dir, "src/app/main.t4", "module app.main\nimport components.counter as counter\nfunc main() -> Int:\n    return counter.value()\n")
	writeCLIProjectFile(t, dir, "ui/components/counter.t4", "module components.counter\nfunc value() -> Int:\n    return 42\n")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"check"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(filepath.ToSlash(stdout.String()), "src/app/main.t4") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestCheckCommandExplicitProjectDirectoryUsesCapsuleEntry(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
        ui
`)
	writeCLIProjectFile(t, dir, "src/app/main.t4", "module app.main\nimport components.counter as counter\nfunc main() -> Int:\n    return counter.value()\n")
	writeCLIProjectFile(t, dir, "ui/components/counter.t4", "module components.counter\nfunc value() -> Int:\n    return 42\n")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"check", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(filepath.ToSlash(stdout.String()), "src/app/main.t4") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestCheckCommandResolvesLocalCapsuleDependencyImport(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "Math/src/math/core.t4", "module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b\n")
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../Math
`)
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(filepath.Join(dir, "App")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"check"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
func TestCheckCommandValidatesPresentTetraLockAgainstCapsuleGraph(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "Math/src/math/core.t4", "module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b\n")
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../Math
`)
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n")

	lockPath := filepath.Join(dir, "App", "Tetra.lock")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "verify", "--lock", lockPath, filepath.Join(dir, "App", "Capsule.t4"), filepath.Join(dir, "Math", "Capsule.t4")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("eco verify exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.2.0"
    sources:
        src
`)

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(filepath.Join(dir, "App")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"check"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected check failure for stale Tetra.lock, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "Tetra.lock") || !strings.Contains(stderr.String(), "version mismatch") {
		t.Fatalf("stderr = %q, want Tetra.lock version mismatch", stderr.String())
	}
	if !strings.Contains(stderr.String(), "tetra project sync") {
		t.Fatalf("stderr = %q, want project sync repair hint", stderr.String())
	}
}

func TestCheckCommandSucceedsWithoutOutputFile(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(dir, "app")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"check", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Checked: "+srcPath) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("check should not create %s, stat err=%v", outPath, err)
	}
}

func TestTargetAwareCommandsRejectInvalidTargetConsistently(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "build", args: []string{"build", "--target", "not-a-target", "examples/flow_hello.tetra"}},
		{name: "run", args: []string{"run", "--target", "not-a-target", "examples/flow_hello.tetra"}},
		{name: "test", args: []string{"test", "--target", "not-a-target", "examples/tooling_tests.tetra"}},
		{name: "smoke", args: []string{"smoke", "--target", "not-a-target", "--run=false"}},
		{name: "smoke list", args: []string{"smoke", "--list", "--target", "not-a-target"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			code := runCLI(tt.args, &bytes.Buffer{}, &stderr)
			if code != 2 {
				t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
			}
			for _, want := range []string{"unsupported target: not-a-target", "supported targets: linux-x64, windows-x64, macos-x64, wasm32-wasi, wasm32-web", "build-only targets: linux-x86, linux-x32"} {
				if !strings.Contains(stderr.String(), want) {
					t.Fatalf("stderr missing %q: %q", want, stderr.String())
				}
			}
		})
	}
}

func TestCheckCommandReportsMissingDefaultMain(t *testing.T) {
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	var stderr bytes.Buffer
	code := runCLI([]string{"check"}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "main.t4") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
