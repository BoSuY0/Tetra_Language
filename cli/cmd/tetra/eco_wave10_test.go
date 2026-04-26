package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEcoVerifyManifestV1PermissionsAndLockMetadata(t *testing.T) {
	dir := t.TempDir()
	core := filepath.Join(dir, "Core.capsule")
	app := filepath.Join(dir, "Tetra.capsule")
	writeCapsuleFile(t, core, `manifest "tetra.capsule.v1"
capsule Core:
    id "tetra://core"
    version "0.1.0"
    target "linux-x64"
    permission "io"
`)
	writeCapsuleFile(t, app, `manifest "tetra.capsule.v1"
capsule App:
    id "tetra://app"
    version "0.1.0"
    target "linux-x64"
    permission "io"
    dependency "tetra://core" "0.1.0"
`)

	lockPath := filepath.Join(dir, "tetra.lock.json")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "verify", "--target", "linux-x64", "--lock", lockPath, app, core}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("eco verify exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`"schema": "tetra.eco.lock.v1"`,
		`"manifest_schema": "tetra.capsule.v1"`,
		`"permissions_model": "tetra.eco.permissions.v1"`,
		`"permissions": [`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("lock missing %q:\n%s", want, text)
		}
	}
}

func TestEcoSeedExportImportRoundTrip(t *testing.T) {
	dir := t.TempDir()
	capsule := filepath.Join(dir, "Tetra.capsule")
	writeCapsuleFile(t, capsule, `manifest "tetra.capsule.v1"
capsule App:
    id "tetra://app"
    version "0.1.0"
    target "linux-x64"
    permission "io"
`)
	seedPath := filepath.Join(dir, "tetra.seed.json")
	lockPath := filepath.Join(dir, "imported.lock.json")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "seed", "export", "--out", seedPath, capsule}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("eco seed export exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"eco", "seed", "import", "--seed", seedPath, "--lock", lockPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("eco seed import exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("expected imported lock file: %v", err)
	}
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read imported lock: %v", err)
	}
	if !strings.Contains(string(raw), `"schema": "tetra.eco.lock.v1"`) {
		t.Fatalf("imported lock missing schema: %s", string(raw))
	}
}

func TestEcoNeedMapTrustSnapshotAndMaterialize(t *testing.T) {
	dir := t.TempDir()
	capsule := filepath.Join(dir, "Tetra.capsule")
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "main.tetra"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeCapsuleFile(t, capsule, `manifest "tetra.capsule.v1"
capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    target "linux-x64"
    permission "io"
`)

	lockPath := filepath.Join(dir, "tetra.lock.json")
	needMapPath := filepath.Join(dir, "needmap.json")
	trustPath := filepath.Join(dir, "trust.snapshot.json")
	pkgPath := filepath.Join(dir, "demo.todex")
	store := filepath.Join(dir, "vault")
	outDir := filepath.Join(dir, "materialized")

	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"eco", "verify", "--target", "linux-x64", "--lock", lockPath, capsule}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco verify exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "needmap", "--lock", lockPath, "-o", needMapPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco needmap exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "vault", "add", "--store", store, "--kind", "source", filepath.Join(dir, "src", "main.tetra")}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco vault add exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "trust", "snapshot", "--lock", lockPath, "--store", store, "-o", trustPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco trust snapshot exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "pack", "--project", capsule, "-o", pkgPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco pack --project exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "materialize", pkgPath, "--target", "linux-x64", "--trust", trustPath, "-C", outDir}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco materialize exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(outDir, "tetra.materialization.json")); err != nil {
		t.Fatalf("expected materialization metadata: %v", err)
	}
	rawNeedMap, err := os.ReadFile(needMapPath)
	if err != nil {
		t.Fatalf("read needmap: %v", err)
	}
	if !strings.Contains(string(rawNeedMap), `"schema": "tetra.eco.needmap.v1"`) {
		t.Fatalf("needmap missing schema: %s", string(rawNeedMap))
	}
	rawTrust, err := os.ReadFile(trustPath)
	if err != nil {
		t.Fatalf("read trust snapshot: %v", err)
	}
	if !strings.Contains(string(rawTrust), `"schema": "tetra.eco.trust-snapshot.v1"`) {
		t.Fatalf("trust snapshot missing schema: %s", string(rawTrust))
	}
}

func TestEcoBetaPublishDownloadAndTetraHubPath(t *testing.T) {
	dir := t.TempDir()
	project := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(project, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "src", "main.tetra"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	capsule := filepath.Join(project, "Tetra.capsule")
	writeCapsuleFile(t, capsule, `manifest "tetra.capsule.v1"
capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    target "linux-x64"
    target "windows-x64"
    permission "io"
`)
	pkgPath := filepath.Join(dir, "demo.todex")
	registry := filepath.Join(dir, "registry")
	trustPath := filepath.Join(dir, "trust.snapshot.json")
	store := filepath.Join(dir, "vault")
	hubStore := filepath.Join(dir, "tetrahub-beta")
	downloadPath := filepath.Join(dir, "downloaded.todex")
	hubDownloadPath := filepath.Join(dir, "hub-downloaded.todex")

	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"eco", "pack", "--project", capsule, "-o", pkgPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco pack --project exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "vault", "add", "--store", store, "--kind", "source", filepath.Join(project, "src", "main.tetra")}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco vault add exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	lockPath := filepath.Join(dir, "tetra.lock.json")
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "verify", "--target", "linux-x64", "--lock", lockPath, capsule}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco verify exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "trust", "snapshot", "--lock", lockPath, "--store", store, "-o", trustPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco trust snapshot exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "publish", "--package", pkgPath, "--registry", registry, "--target", "linux-x64", "--trust", trustPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco publish exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Published (beta)") {
		t.Fatalf("publish stdout = %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "download", "--id", "tetra://demo", "--version", "0.1.0", "--target", "linux-x64", "--registry", registry, "-o", downloadPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco download exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(downloadPath); err != nil {
		t.Fatalf("downloaded package missing: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "tetrahub", "publish", "--package", pkgPath, "--store", hubStore, "--target", "linux-x64", "--trust", trustPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco tetrahub publish exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "tetrahub", "download", "--id", "tetra://demo", "--version", "0.1.0", "--target", "linux-x64", "--store", hubStore, "-o", hubDownloadPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco tetrahub download exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(hubDownloadPath); err != nil {
		t.Fatalf("hub downloaded package missing: %v", err)
	}

	metaPath := filepath.Join(registry, "packages", "tetra_demo", "0.1.0", "linux-x64", "metadata.json")
	rawMeta, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read publish metadata: %v", err)
	}
	var meta map[string]any
	if err := json.Unmarshal(rawMeta, &meta); err != nil {
		t.Fatalf("decode publish metadata: %v\n%s", err, string(rawMeta))
	}
	if meta["schema"] != "tetra.eco.publish.v1beta" || meta["channel"] != "beta" {
		t.Fatalf("publish metadata = %#v", meta)
	}
}

func writeCapsuleFile(t *testing.T, path string, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}
