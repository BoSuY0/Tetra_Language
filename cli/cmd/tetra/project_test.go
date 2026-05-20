package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectSyncWritesLockForProjectWithoutDependencies(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    entry "src/main.t4"
    sources:
        src
    targets:
        linux-x64
`)
	writeCLIProjectFile(t, dir, "src/main.t4", "func main() -> Int:\n    return 0\n")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"project", "sync", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("project sync exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	lockPath := filepath.Join(dir, "Tetra.lock")
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read Tetra.lock: %v", err)
	}
	if !strings.Contains(string(raw), `"tetra://demo"`) {
		t.Fatalf("Tetra.lock missing capsule id:\n%s", string(raw))
	}
	if !strings.Contains(stdout.String(), "Project synced") {
		t.Fatalf("stdout = %q, want sync message", stdout.String())
	}
}

func TestProjectSyncCheckReportsMissingLockWithoutWriting(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    entry "src/main.t4"
    sources:
        src
    targets:
        linux-x64
`)
	writeCLIProjectFile(t, dir, "src/main.t4", "func main() -> Int:\n    return 0\n")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"project", "sync", "--check", dir}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("project sync --check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "would generate lock") || !strings.Contains(combined, "Tetra.lock") {
		t.Fatalf("sync --check output = stdout=%q stderr=%q, want missing lock dry-run", stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "Tetra.lock")); err == nil {
		t.Fatalf("project sync --check unexpectedly wrote Tetra.lock")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat Tetra.lock: %v", err)
	}
}

func TestProjectSyncRejectsTargetAndAllTargetsTogether(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    entry "src/main.t4"
    sources:
        src
    targets:
        linux-x64
`)
	writeCLIProjectFile(t, dir, "src/main.t4", "func main() -> Int:\n    return 0\n")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"project", "sync", "--target", "linux-x64", "--all-targets", dir}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("project sync exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "either --target or --all-targets") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestProjectSyncGeneratesDependencyArtifactsAndLock(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	appRoot := writeArtifactBuildFixture(t, dir, target)

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"project", "sync", "--target", target, appRoot}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("project sync exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	for _, rel := range []string{
		"interfaces/math/core.t4i",
		"artifacts/math/core." + target + ".tobj",
		"seeds/app-deps.t4s",
		"Tetra.lock",
	} {
		if _, err := os.Stat(filepath.Join(appRoot, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected project sync generated %s: %v", rel, err)
		}
	}
	capsuleRaw, err := os.ReadFile(filepath.Join(appRoot, "Capsule.t4"))
	if err != nil {
		t.Fatalf("read Capsule.t4: %v", err)
	}
	if !strings.Contains(string(capsuleRaw), "interface interfaces/math/core.t4i") || !strings.Contains(string(capsuleRaw), "object "+target+" artifacts/math/core."+target+".tobj") {
		t.Fatalf("Capsule.t4 missing generated artifact declarations:\n%s", string(capsuleRaw))
	}
	if !strings.Contains(stdout.String(), "Project synced") {
		t.Fatalf("stdout = %q, want sync message", stdout.String())
	}
}

func TestProjectSyncWritesLockForBuildOnlyTargetWithoutNativeArtifacts(t *testing.T) {
	dir := t.TempDir()
	appRoot := writeArtifactBuildFixture(t, dir, "wasm32-wasi")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"project", "sync", appRoot}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("project sync exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(appRoot, "Tetra.lock")); err != nil {
		t.Fatalf("expected Tetra.lock: %v", err)
	}
	if _, err := os.Stat(filepath.Join(appRoot, filepath.FromSlash("artifacts/math/core.wasm32-wasi.tobj"))); err == nil {
		t.Fatalf("project sync unexpectedly wrote wasm object artifact")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat wasm object artifact: %v", err)
	}
}

func TestProjectDepsAddPathDiscoversMetadataAndAppendsDeps(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "func main() -> Int:\n    return 0\n")

	appRoot := filepath.Join(dir, "App")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"project", "deps", "add", "--path", "../Math", appRoot}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("project deps add exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(filepath.Join(appRoot, "Capsule.t4"))
	if err != nil {
		t.Fatal(err)
	}
	capsule := string(raw)
	if !strings.Contains(capsule, "deps:") || !strings.Contains(capsule, "tetra://math 0.1.0 ../Math") {
		t.Fatalf("Capsule.t4 missing dependency:\n%s", capsule)
	}
	if !strings.Contains(stdout.String(), "Added dependency") || !strings.Contains(stdout.String(), "run: tetra project sync") {
		t.Fatalf("stdout = %q, want add message and sync hint", stdout.String())
	}
}

func TestProjectDepsAddRejectsDuplicate(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../Math
`)
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "func main() -> Int:\n    return 0\n")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"project", "deps", "add", "--path", "../Math", filepath.Join(dir, "App")}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("project deps add exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "duplicate dependency") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestProjectDepsAddAllowsMetadataOverride(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "func main() -> Int:\n    return 0\n")

	appRoot := filepath.Join(dir, "App")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"project", "deps", "add", "--path", "../Math", "--id", "tetra://math-alt", "--version", "0.2.0", appRoot}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("project deps add exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(filepath.Join(appRoot, "Capsule.t4"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "tetra://math-alt 0.2.0 ../Math") {
		t.Fatalf("Capsule.t4 missing overridden dependency:\n%s", string(raw))
	}
}

func TestProjectDepsListJSONReportsResolvedPath(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../Math
`)
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "func main() -> Int:\n    return 0\n")

	var report struct {
		Dependencies []struct {
			ID           string `json:"id"`
			Version      string `json:"version"`
			Path         string `json:"path"`
			ResolvedPath string `json:"resolved_path"`
			Status       string `json:"status"`
		} `json:"dependencies"`
	}
	runCLIJSONStdout(t, []string{"project", "deps", "list", "--format=json", filepath.Join(dir, "App")}, 0, &report)
	if len(report.Dependencies) != 1 {
		t.Fatalf("dependencies = %#v", report.Dependencies)
	}
	dep := report.Dependencies[0]
	if dep.ID != "tetra://math" || dep.Version != "0.1.0" || dep.Path != "../Math" || dep.Status != "ok" || !strings.HasSuffix(filepath.ToSlash(dep.ResolvedPath), "/Math") {
		t.Fatalf("dependency report = %#v", dep)
	}
}

func TestProjectDepsRemoveByID(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../Math
`)
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "func main() -> Int:\n    return 0\n")

	appRoot := filepath.Join(dir, "App")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"project", "deps", "remove", "--id", "tetra://math", appRoot}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("project deps remove exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(filepath.Join(appRoot, "Capsule.t4"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "tetra://math") {
		t.Fatalf("dependency was not removed:\n%s", string(raw))
	}
	if !strings.Contains(stdout.String(), "Removed dependency") || !strings.Contains(stdout.String(), "run: tetra project sync") {
		t.Fatalf("stdout = %q, want remove message and sync hint", stdout.String())
	}
}

func TestProjectDepsRemoveRejectsAmbiguousID(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../MathV1
        tetra://math 0.2.0 ../MathV2
`)
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "func main() -> Int:\n    return 0\n")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"project", "deps", "remove", "--id", "tetra://math", filepath.Join(dir, "App")}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("project deps remove exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "requires --version") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestProjectDepsCheckPassesForValidDependency(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../Math
`)
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "func main() -> Int:\n    return 0\n")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"project", "deps", "check", filepath.Join(dir, "App")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("project deps check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Dependencies OK") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestProjectDepsCheckFailsForMissingPathVersionMismatchAndCycle(t *testing.T) {
	t.Run("missing path", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://missing 0.1.0 ../Missing
`)
		writeCLIProjectFile(t, dir, "App/src/app/main.t4", "func main() -> Int:\n    return 0\n")

		var stdout, stderr bytes.Buffer
		code := runCLI([]string{"project", "deps", "check", filepath.Join(dir, "App")}, &stdout, &stderr)
		if code != 1 {
			t.Fatalf("project deps check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
		if !strings.Contains(stderr.String(), "tetra://missing") || !strings.Contains(stderr.String(), "Missing") {
			t.Fatalf("stderr = %q", stderr.String())
		}
	})
	t.Run("version mismatch", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.2.0"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../Math
`)
		writeCLIProjectFile(t, dir, "App/src/app/main.t4", "func main() -> Int:\n    return 0\n")

		var stdout, stderr bytes.Buffer
		code := runCLI([]string{"project", "deps", "check", filepath.Join(dir, "App")}, &stdout, &stderr)
		if code != 1 {
			t.Fatalf("project deps check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
		if !strings.Contains(stderr.String(), "version mismatch") {
			t.Fatalf("stderr = %q", stderr.String())
		}
	})
	t.Run("cycle", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../Math
`)
		writeCLIProjectFile(t, dir, "App/src/app/main.t4", "func main() -> Int:\n    return 0\n")
		writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
    deps:
        tetra://app 0.1.0 ../App
`)

		var stdout, stderr bytes.Buffer
		code := runCLI([]string{"project", "deps", "check", filepath.Join(dir, "App")}, &stdout, &stderr)
		if code != 1 {
			t.Fatalf("project deps check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
		if !strings.Contains(stderr.String(), "capsule dependency cycle") {
			t.Fatalf("stderr = %q", stderr.String())
		}
	})
}
