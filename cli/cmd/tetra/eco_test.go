package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEcoVerifySingleCapsuleExpandsPathDependenciesIntoTetraLock(t *testing.T) {
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

	lockPath := filepath.Join(dir, "App", "Tetra.lock")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "verify", "--lock", lockPath, filepath.Join(dir, "App", "Capsule.t4")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("eco verify exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	if !strings.Contains(string(raw), `"tetra://app"`) || !strings.Contains(string(raw), `"tetra://math"`) {
		t.Fatalf("lock did not include full path dependency graph:\n%s", string(raw))
	}
}

func TestEcoArtifactsBuildGeneratesDependencyArtifactsLockAndBuildsProject(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
    targets:
        linux
`)
	writeCLIProjectFile(t, dir, "Math/src/math/core.t4", "module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b\n")
	writeCLIProjectFile(t, dir, "App/Capsule.t4", fmt.Sprintf(`capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    targets:
        %s
    deps:
        tetra://math 0.1.0 ../Math
`, target))
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n")

	appRoot := filepath.Join(dir, "App")
	lockPath := filepath.Join(appRoot, "Tetra.lock")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "artifacts", "build", "--target", target, "--lock", lockPath, filepath.Join(appRoot, "Capsule.t4")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("eco artifacts build exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	interfaceRel := "interfaces/math/core.t4i"
	objectRel := "artifacts/math/core." + target + ".tobj"
	seedRel := "seeds/app-deps.t4s"
	for _, rel := range []string{interfaceRel, objectRel, seedRel, "Tetra.lock"} {
		if _, err := os.Stat(filepath.Join(appRoot, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected generated %s: %v", rel, err)
		}
	}
	capsuleRaw, err := os.ReadFile(filepath.Join(appRoot, "Capsule.t4"))
	if err != nil {
		t.Fatalf("read Capsule.t4: %v", err)
	}
	capsuleText := string(capsuleRaw)
	for _, want := range []string{
		"artifacts:",
		"interface " + interfaceRel,
		"object " + target + " " + objectRel,
		"seed " + seedRel,
	} {
		if !strings.Contains(capsuleText, want) {
			t.Fatalf("Capsule.t4 missing %q:\n%s", want, capsuleText)
		}
	}
	lockRaw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read Tetra.lock: %v", err)
	}
	for _, want := range []string{`"kind": "object"`, `"target": "` + target + `"`, `"module": "math.core"`, `"public_api_hash": "sha256:`} {
		if !strings.Contains(string(lockRaw), want) {
			t.Fatalf("Tetra.lock missing %q:\n%s", want, string(lockRaw))
		}
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(appRoot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})
	stdout.Reset()
	stderr.Reset()
	out := filepath.Join(appRoot, "app")
	code = runCLI([]string{"build", "--target", target, "-o", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("build exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected build output %s: %v", out, err)
	}
}

func TestEcoArtifactsCheckDetectsStaleInterfaceAndSuggestsRepair(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	appRoot := writeArtifactBuildFixture(t, dir, target)
	lockPath := filepath.Join(appRoot, "Tetra.lock")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "artifacts", "build", "--target", target, "--lock", lockPath, filepath.Join(appRoot, "Capsule.t4")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("eco artifacts build exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	writeCLIProjectFile(t, dir, "Math/src/math/core.t4", "module math.core\nfunc add(a: Int, b: Int, c: Int) -> Int:\n    return a + b + c\n")

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"eco", "artifacts", "check", "--target", target, filepath.Join(appRoot, "Capsule.t4")}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected stale artifact failure, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	combined := stdout.String() + stderr.String()
	for _, want := range []string{"stale interface artifact", "math.core", "tetra eco artifacts build --target " + target} {
		if !strings.Contains(combined, want) {
			t.Fatalf("artifact check output missing %q:\nstdout=%s\nstderr=%s", want, stdout.String(), stderr.String())
		}
	}
}

func TestEcoArtifactsBuildCheckDryRunDoesNotWriteArtifacts(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	appRoot := writeArtifactBuildFixture(t, dir, target)
	lockPath := filepath.Join(appRoot, "Tetra.lock")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "artifacts", "build", "--check", "--target", target, "--lock", lockPath, filepath.Join(appRoot, "Capsule.t4")}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected dry-run to report pending artifacts")
	}
	if !strings.Contains(stdout.String()+stderr.String(), "would generate") {
		t.Fatalf("dry-run output = stdout=%q stderr=%q, want would generate", stdout.String(), stderr.String())
	}
	for _, rel := range []string{"interfaces/math/core.t4i", "artifacts/math/core." + target + ".tobj", "seeds/app-deps.t4s", "Tetra.lock"} {
		if _, err := os.Stat(filepath.Join(appRoot, filepath.FromSlash(rel))); err == nil {
			t.Fatalf("dry-run unexpectedly wrote %s", rel)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", rel, err)
		}
	}
}

func TestEcoArtifactsBuildAllTargetsSkipsWASMObjectTargets(t *testing.T) {
	target := mustHostTarget(t)
	dir := t.TempDir()
	appRoot := writeArtifactBuildFixture(t, dir, target)
	writeCLIProjectFile(t, dir, "App/Capsule.t4", fmt.Sprintf(`capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    targets:
        %s
        wasm32-wasi
    deps:
        tetra://math 0.1.0 ../Math
`, target))

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "artifacts", "build", "--all-targets", "--lock", filepath.Join(appRoot, "Tetra.lock"), filepath.Join(appRoot, "Capsule.t4")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("eco artifacts build --all-targets exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(appRoot, filepath.FromSlash("artifacts/math/core."+target+".tobj"))); err != nil {
		t.Fatalf("expected native object artifact: %v", err)
	}
	if _, err := os.Stat(filepath.Join(appRoot, filepath.FromSlash("artifacts/math/core.wasm32-wasi.tobj"))); err == nil {
		t.Fatalf("unexpected wasm object artifact")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat wasm object: %v", err)
	}
}

func TestEcoVerifyPackAndUnpack(t *testing.T) {
	dir := t.TempDir()
	capsule := filepath.Join(dir, "Tetra.capsule")
	src := `capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    target "linux-x64"
`
	if err := os.WriteFile(capsule, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if code := runCLI([]string{"eco", "verify", capsule}, &stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("eco verify exit code = %d, stdout=%q", code, stdout.String())
	}
	pkg := filepath.Join(dir, "demo.todex")
	if code := runCLI([]string{"eco", "pack", capsule, "-o", pkg}, &stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("eco pack exit code = %d, stdout=%q", code, stdout.String())
	}
	outDir := filepath.Join(dir, "unpacked")
	if code := runCLI([]string{"eco", "unpack", pkg, "-C", outDir}, &stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("eco unpack exit code = %d, stdout=%q", code, stdout.String())
	}
	if _, err := os.Stat(filepath.Join(outDir, "Tetra.capsule")); err != nil {
		t.Fatalf("expected unpacked capsule: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "tetra.package.json")); err != nil {
		t.Fatalf("expected unpacked package metadata: %v", err)
	}
}

func TestEcoVerifyHelpExitsSuccessfully(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"eco", "verify", "--help"}, &bytes.Buffer{}, &stderr)
	if code != 0 {
		t.Fatalf("eco verify --help exit code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "Usage of eco verify:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEcoTopLevelHelpMentionsVerifyLock(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("eco --help exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "eco verify --lock") {
		t.Fatalf("stdout = %q, want verify --lock guidance", stdout.String())
	}
}

func TestEcoPackUnpackVaultHelpExitsSuccessfully(t *testing.T) {
	for _, args := range [][]string{
		{"eco", "pack", "--help"},
		{"eco", "unpack", "--help"},
		{"eco", "vault", "--help"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runCLI(args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("%v exit code = %d, stdout=%q stderr=%q", args, code, stdout.String(), stderr.String())
			}
			combined := stdout.String() + stderr.String()
			if !strings.Contains(strings.ToLower(combined), "usage:") {
				t.Fatalf("%v output missing usage text: stdout=%q stderr=%q", args, stdout.String(), stderr.String())
			}
		})
	}
}

func TestEcoPackProjectBundle(t *testing.T) {
	dir := t.TempDir()
	capsule := filepath.Join(dir, "Tetra.capsule")
	if err := os.WriteFile(capsule, []byte(`capsule Demo:
    id "tetra://demo"
    version "0.1.0"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "main.tetra"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg := filepath.Join(dir, "demo.todex")
	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"eco", "pack", "--project", capsule, "-o", pkg}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco pack --project exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	outDir := filepath.Join(dir, "unpacked")
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "unpack", pkg, "-C", outDir}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco unpack exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(outDir, "Tetra.capsule")); err != nil {
		t.Fatalf("expected unpacked capsule: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "src", "main.tetra")); err != nil {
		t.Fatalf("expected bundled source: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "tetra.package.json")); err != nil {
		t.Fatalf("expected bundled package metadata: %v", err)
	}
}

func TestEcoPackProjectBundleUsesT4CapsuleAndSource(t *testing.T) {
	dir := t.TempDir()
	capsule := filepath.Join(dir, "Capsule.t4")
	if err := os.WriteFile(capsule, []byte(`capsule Demo:
    id "tetra://demo"
    version "0.1.0"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "main.t4"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg := filepath.Join(dir, "demo.tdx")
	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"eco", "pack", "--project", capsule, "-o", pkg}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco pack --project exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	outDir := filepath.Join(dir, "unpacked")
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "unpack", pkg, "-C", outDir}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco unpack exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	for _, rel := range []string{"Capsule.t4", "src/main.t4", "tetra.package.json"} {
		if _, err := os.Stat(filepath.Join(outDir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected bundled %s: %v", rel, err)
		}
	}
}

func TestEcoVerifyStructuredCapsuleT4WritesPolicyLock(t *testing.T) {
	dir := t.TempDir()
	capsule := filepath.Join(dir, "Capsule.t4")
	if err := os.WriteFile(capsule, []byte(`capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    entry "src/main.t4"

    sources:
        src
        ui

    targets:
        linux
        web

    allow:
        ui
        fs.readWrite.userData

    policy:
        unsafe deny
        reproducible required
`), 0o644); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(dir, "Tetra.lock")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "verify", "--lock", lockPath, capsule}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("eco verify exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	for _, want := range []string{`"path": "` + capsule + `"`, `"linux-x64"`, `"wasm32-web"`, `"ui"`, `"fs.readWrite.userData"`, `"unsafe": "deny"`, `"reproducible": "required"`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("lock missing %q:\n%s", want, string(raw))
		}
	}
}

func TestEcoVerifyDependencyGraphAndLock(t *testing.T) {
	dir := t.TempDir()
	core := filepath.Join(dir, "Core.capsule")
	app := filepath.Join(dir, "App.capsule")
	if err := os.WriteFile(core, []byte(`capsule Core:
    id "tetra://core"
    version "0.1.0"
    target "linux-x64"
    effect "io"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(app, []byte(`capsule App:
    id "tetra://app"
    version "0.1.0"
    target "linux-x64"
    effect "io"
    dependency "tetra://core" "0.1.0"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(dir, "tetra.lock.json")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "verify", "--target", "linux-x64", "--lock", lock, app, core}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("eco verify exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(lock)
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	if !strings.Contains(string(raw), `"capsules"`) || !strings.Contains(string(raw), `"tetra://core"`) {
		t.Fatalf("unexpected lock: %s", string(raw))
	}
}

func TestEcoVerifyRejectsPermissionEscalationFromDependency(t *testing.T) {
	dir := t.TempDir()
	core := filepath.Join(dir, "Core.capsule")
	app := filepath.Join(dir, "App.capsule")
	if err := os.WriteFile(core, []byte(`capsule Core:
    id "tetra://core"
    version "0.1.0"
    target "linux-x64"
    effect "io"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(app, []byte(`capsule App:
    id "tetra://app"
    version "0.1.0"
    target "linux-x64"
    dependency "tetra://core" "0.1.0"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := runCLI([]string{"eco", "verify", app, core}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatalf("expected permission mismatch failure")
	}
	if !strings.Contains(stderr.String(), "missing required effect") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEcoVerifyRejectsDuplicateManifestIDField(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "App.capsule")
	if err := os.WriteFile(app, []byte(`capsule App:
    id "tetra://app"
    id "tetra://app-2"
    version "0.1.0"
    target "linux-x64"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := runCLI([]string{"eco", "verify", app}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatalf("expected duplicate id field failure")
	}
	if !strings.Contains(stderr.String(), "duplicate id field") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEcoVerifyReportsMissingDependency(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "App.capsule")
	if err := os.WriteFile(app, []byte(`capsule App:
    id "tetra://app"
    version "0.1.0"
    dependency "tetra://missing" "0.1.0"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := runCLI([]string{"eco", "verify", app}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatalf("expected missing dependency failure")
	}
	if !strings.Contains(stderr.String(), "missing dependency") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEcoVerifyReportsDuplicateIDAndTargetMismatch(t *testing.T) {
	dir := t.TempDir()
	one := filepath.Join(dir, "One.capsule")
	two := filepath.Join(dir, "Two.capsule")
	if err := os.WriteFile(one, []byte(`capsule One:
    id "tetra://dup"
    version "0.1.0"
    target "linux-x64"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(two, []byte(`capsule Two:
    id "tetra://dup"
    version "0.1.0"
    target "linux-x64"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := runCLI([]string{"eco", "verify", one, two}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatalf("expected duplicate capsule id failure")
	}
	if !strings.Contains(stderr.String(), "duplicate capsule id") {
		t.Fatalf("stderr = %q", stderr.String())
	}

	stderr.Reset()
	code = runCLI([]string{"eco", "verify", "--target", "windows-x64", one}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatalf("expected target mismatch failure")
	}
	if !strings.Contains(stderr.String(), "target mismatch") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEcoVaultAddListAndVerify(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "module.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := filepath.Join(dir, "vault")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "vault", "add", "--store", store, "--kind", "source", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("vault add exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Vault added: sha256:") {
		t.Fatalf("vault add stdout = %q", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"eco", "vault", "list", "--store", store}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("vault list exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "source") || !strings.Contains(stdout.String(), "module.tetra") {
		t.Fatalf("vault list stdout = %q", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"eco", "vault", "verify", "--store", store}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("vault verify exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Vault OK: 1 records") {
		t.Fatalf("vault verify stdout = %q", stdout.String())
	}
}

func TestEcoVaultVerifyDetectsCorruptObject(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "module.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := filepath.Join(dir, "vault")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "vault", "add", "--store", store, "--kind", "source", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("vault add exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	fields := strings.Fields(stdout.String())
	if len(fields) < 3 || !strings.HasPrefix(fields[2], "sha256:") {
		t.Fatalf("unexpected vault add stdout = %q", stdout.String())
	}
	hash := strings.TrimPrefix(fields[2], "sha256:")
	objectPath := filepath.Join(store, "objects", "sha256", hash)
	if err := os.WriteFile(objectPath, []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"eco", "vault", "verify", "--store", store}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected vault verify failure")
	}
	if !strings.Contains(stderr.String(), "vault object") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
