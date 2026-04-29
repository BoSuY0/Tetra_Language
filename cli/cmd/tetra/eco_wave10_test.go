package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"os/exec"
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

func TestEcoVerifyRejectsMissingRequiredManifestFields(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "capsule declaration",
			text: `manifest "tetra.capsule.v1"
id "tetra://app"
version "0.1.0"
target "linux-x64"
`,
			want: "missing capsule declaration",
		},
		{
			name: "id",
			text: `manifest "tetra.capsule.v1"
capsule App:
version "0.1.0"
target "linux-x64"
`,
			want: "missing capsule id",
		},
		{
			name: "version",
			text: `manifest "tetra.capsule.v1"
capsule App:
id "tetra://app"
target "linux-x64"
`,
			want: "missing capsule version",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			capsule := filepath.Join(dir, "Tetra.capsule")
			writeCapsuleFile(t, capsule, tt.text)
			var stderr bytes.Buffer
			if code := runCLI([]string{"eco", "verify", capsule}, &bytes.Buffer{}, &stderr); code == 0 {
				t.Fatalf("expected eco verify failure")
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tt.want)
			}
		})
	}
}

func TestEcoCapsuleFixtureMatrix(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	fixtureDir := filepath.Join(root, "cli", "cmd", "tetra", "testdata", "eco_capsules", "matrix")
	tests := []struct {
		name    string
		files   []string
		target  string
		wantOK  bool
		wantErr string
	}{
		{name: "valid graph", files: []string{"valid/Tetra.capsule", "valid/Core.capsule"}, target: "linux-x64", wantOK: true},
		{name: "missing dependency", files: []string{"missing_dependency/Tetra.capsule"}, target: "linux-x64", wantErr: "missing dependency tetra://fixture/missing"},
		{name: "malformed manifest", files: []string{"malformed/Tetra.capsule"}, wantErr: "expected quoted string"},
		{name: "unsupported target", files: []string{"unsupported_target/Tetra.capsule"}, wantErr: "unsupported target plan9-x64"},
		{name: "duplicate dependency", files: []string{"duplicate_dependency/Tetra.capsule", "duplicate_dependency/Core.capsule"}, target: "linux-x64", wantErr: "duplicate dependency tetra://fixture/core 0.1.0"},
		{name: "permission mismatch", files: []string{"permission_mismatch/Tetra.capsule", "permission_mismatch/Core.capsule"}, target: "linux-x64", wantErr: "missing required permission mmio for dependency tetra://fixture/core"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"eco", "verify"}
			if tt.target != "" {
				args = append(args, "--target", tt.target)
			}
			for _, file := range tt.files {
				args = append(args, filepath.Join(fixtureDir, file))
			}
			var stdout, stderr bytes.Buffer
			code := runCLI(args, &stdout, &stderr)
			if tt.wantOK {
				if code != 0 {
					t.Fatalf("eco verify exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
				}
				return
			}
			if code == 0 {
				t.Fatalf("expected eco verify failure, stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), tt.wantErr) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tt.wantErr)
			}
		})
	}
}

func TestEcoLockFixtureRejectsGraphHashMismatch(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(root, "cli", "cmd", "tetra", "testdata", "eco_capsules", "matrix", "lock_mismatch", "tetra.lock.json")
	cmd := testCommand(t, "go", "run", "./tools/cmd/validate-eco-lock", "--lock", lock)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected validate-eco-lock failure\n%s", out)
	}
	if !strings.Contains(string(out), "graph_sha256 mismatch") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestEcoVerifyManifestV1RejectsDependencyPermissionEscalation(t *testing.T) {
	dir := t.TempDir()
	core := filepath.Join(dir, "Core.capsule")
	app := filepath.Join(dir, "Tetra.capsule")
	writeCapsuleFile(t, core, `manifest "tetra.capsule.v1"
capsule Core:
    id "tetra://core"
    version "0.1.0"
    target "linux-x64"
    permission "mmio"
`)
	writeCapsuleFile(t, app, `manifest "tetra.capsule.v1"
capsule App:
    id "tetra://app"
    version "0.1.0"
    target "linux-x64"
    dependency "tetra://core" "0.1.0"
`)

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"eco", "verify", "--target", "linux-x64", app, core}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected eco verify permission failure, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "missing required permission mmio for dependency tetra://core") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestEcoVerifyRejectsDependencyVersionMismatch(t *testing.T) {
	dir := t.TempDir()
	core := filepath.Join(dir, "Core.capsule")
	app := filepath.Join(dir, "Tetra.capsule")
	writeCapsuleFile(t, core, `manifest "tetra.capsule.v1"
capsule Core:
    id "tetra://core"
    version "0.2.0"
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
	var stderr bytes.Buffer
	if code := runCLI([]string{"eco", "verify", "--target", "linux-x64", app, core}, &bytes.Buffer{}, &stderr); code == 0 {
		t.Fatalf("expected eco verify failure")
	}
	if !strings.Contains(stderr.String(), "version mismatch") {
		t.Fatalf("stderr = %q", stderr.String())
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

func TestEcoSeedImportRejectsUnsupportedPermissionsModel(t *testing.T) {
	dir := t.TempDir()
	seedPath := filepath.Join(dir, "seed.json")
	lockPath := filepath.Join(dir, "lock.json")
	if err := os.WriteFile(seedPath, []byte(`{
  "schema": "tetra.eco.seed.v1",
  "generated_at_unix": 0,
  "lock": {
    "schema": "tetra.eco.lock.v1",
    "manifest_schema": "tetra.capsule.v1",
    "permissions_model": "tetra.eco.permissions.v2",
    "capsules": [
      {
        "id": "tetra://app",
        "name": "App",
        "version": "0.1.0",
        "path": "Tetra.capsule",
        "targets": ["linux-x64"],
        "permissions": ["io"]
      }
    ]
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	if code := runCLI([]string{"eco", "seed", "import", "--seed", seedPath, "--lock", lockPath}, &bytes.Buffer{}, &stderr); code == 0 {
		t.Fatalf("expected eco seed import failure")
	}
	if !strings.Contains(stderr.String(), "unsupported lock permissions model") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestEcoPackProjectBundleIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	project := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(project, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	capsule := filepath.Join(project, "Tetra.capsule")
	writeCapsuleFile(t, capsule, `manifest "tetra.capsule.v1"
capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    target "linux-x64"
    permission "io"
`)
	if err := os.WriteFile(filepath.Join(project, "src", "main.tetra"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(dir, "first.todex")
	second := filepath.Join(dir, "second.todex")
	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"eco", "pack", "--project", capsule, "-o", first}, &stdout, &stderr); code != 0 {
		t.Fatalf("first eco pack exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "pack", "--project", capsule, "-o", second}, &stdout, &stderr); code != 0 {
		t.Fatalf("second eco pack exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	firstRaw, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	secondRaw, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstRaw, secondRaw) {
		t.Fatalf("project bundle output is not deterministic")
	}
}

func TestEcoUnpackRejectsUnsafeArchivePath(t *testing.T) {
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "unsafe.todex")
	writeTarGzFixture(t, pkgPath, "../evil.tetra", []byte("func main() -> Int:\n    return 0\n"))
	var stderr bytes.Buffer
	if code := runCLI([]string{"eco", "unpack", pkgPath, "-C", filepath.Join(dir, "out")}, &bytes.Buffer{}, &stderr); code == 0 {
		t.Fatalf("expected eco unpack failure")
	}
	if !strings.Contains(stderr.String(), "unsafe archive path") {
		t.Fatalf("stderr = %q", stderr.String())
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

func TestEcoNeedMapRejectsUnsupportedLockSchema(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "tetra.lock.json")
	if err := os.WriteFile(lockPath, []byte(`{
  "schema": "tetra.eco.lock.v2",
  "manifest_schema": "tetra.capsule.v1",
  "permissions_model": "tetra.eco.permissions.v1",
  "capsules": [
    {
      "id": "tetra://app",
      "name": "App",
      "version": "0.1.0",
      "path": "Tetra.capsule",
      "targets": ["linux-x64"],
      "permissions": ["io"]
    }
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	if code := runCLI([]string{"eco", "needmap", "--lock", lockPath, "-o", filepath.Join(dir, "needmap.json")}, &bytes.Buffer{}, &stderr); code == 0 {
		t.Fatalf("expected eco needmap failure")
	}
	if !strings.Contains(stderr.String(), "unsupported lock schema") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestEcoTrustSnapshotRejectsLockGraphHashMismatch(t *testing.T) {
	dir := t.TempDir()
	capsule := filepath.Join(dir, "Tetra.capsule")
	writeCapsuleFile(t, capsule, `manifest "tetra.capsule.v1"
capsule App:
    id "tetra://app"
    version "0.1.0"
    target "linux-x64"
    permission "io"
`)
	lockPath := filepath.Join(dir, "tetra.lock.json")
	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"eco", "verify", "--target", "linux-x64", "--lock", lockPath, capsule}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco verify exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	var lock map[string]any
	if err := json.Unmarshal(raw, &lock); err != nil {
		t.Fatal(err)
	}
	lock["graph_sha256"] = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	raw, err = json.MarshalIndent(lock, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(lockPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "trust", "snapshot", "--lock", lockPath, "--store", filepath.Join(dir, "vault"), "-o", filepath.Join(dir, "trust.json")}, &stdout, &stderr); code == 0 {
		t.Fatalf("expected eco trust snapshot failure, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "lock graph_sha256 mismatch") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
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
	cmd := testCommand(t, "go", "run", "./tools/cmd/validate-eco-publish", "--registry", registry, "--id", "tetra://demo", "--version", "0.1.0", "--target", "linux-x64")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("validate-eco-publish failed: %v\n%s", err, out)
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

func TestEcoDownloadRejectsTamperedPublishedPackage(t *testing.T) {
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
    permission "io"
`)
	pkgPath := filepath.Join(dir, "demo.todex")
	registry := filepath.Join(dir, "registry")
	downloadPath := filepath.Join(dir, "downloaded.todex")

	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"eco", "pack", "--project", capsule, "-o", pkgPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco pack --project exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "publish", "--package", pkgPath, "--registry", registry, "--target", "linux-x64"}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco publish exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	publishedPackage := filepath.Join(registry, "packages", "tetra_demo", "0.1.0", "linux-x64", "package.todex")
	raw, err := os.ReadFile(publishedPackage)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Fatal("published package is empty")
	}
	raw[0] ^= 0xff
	if err := os.WriteFile(publishedPackage, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "download", "--id", "tetra://demo", "--version", "0.1.0", "--target", "linux-x64", "--registry", registry, "-o", downloadPath}, &stdout, &stderr); code == 0 {
		t.Fatalf("expected eco download failure, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "package hash mismatch") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestEcoUnpackRejectsTamperedPackageContent(t *testing.T) {
	dir := t.TempDir()
	project, capsule := writeEcoProjectFixture(t, dir)
	pkgPath := filepath.Join(dir, "demo.todex")
	tamperedPath := filepath.Join(dir, "demo-tampered.todex")

	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"eco", "pack", "--project", capsule, "-o", pkgPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco pack --project exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	tamperTodexEntry(t, pkgPath, tamperedPath, "src/main.tetra", []byte("func main() -> Int:\n    return 9\n"))

	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "unpack", tamperedPath, "-C", filepath.Join(project, "out")}, &stdout, &stderr); code == 0 {
		t.Fatalf("expected tampered unpack failure, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "package metadata hash mismatch for src/main.tetra") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestEcoUnpackRejectsTamperedPackageMetadata(t *testing.T) {
	dir := t.TempDir()
	_, capsule := writeEcoProjectFixture(t, dir)
	pkgPath := filepath.Join(dir, "demo.todex")
	tamperedPath := filepath.Join(dir, "demo-metadata-tampered.todex")

	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"eco", "pack", "--project", capsule, "-o", pkgPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco pack --project exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	tamperTodexEntry(t, pkgPath, tamperedPath, "tetra.package.json", []byte(`{"schema":"tetra.eco.package.v1","compression":"gzip","mtime_unix":0,"file_count":1,"files":[]}`+"\n"))

	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "unpack", tamperedPath, "-C", filepath.Join(dir, "out")}, &stdout, &stderr); code == 0 {
		t.Fatalf("expected tampered metadata failure, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "package metadata file_count mismatch") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestEcoVaultVerifyRejectsTamperedObject(t *testing.T) {
	dir := t.TempDir()
	store := filepath.Join(dir, "vault")
	source := filepath.Join(dir, "src", "main.tetra")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"eco", "vault", "add", "--store", store, "--kind", "source", source}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco vault add exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	fields := strings.Fields(stdout.String())
	if len(fields) < 3 {
		t.Fatalf("unexpected vault add stdout: %q", stdout.String())
	}
	object := filepath.Join(store, "objects", "sha256", strings.TrimPrefix(fields[2], "sha256:"))
	if err := os.WriteFile(object, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "vault", "verify", "--store", store}, &stdout, &stderr); code == 0 {
		t.Fatalf("expected vault verify failure, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "vault object mismatch") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestEcoDogfoodFixtureLocalLifecycle(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, "examples", "projects", "eco_dogfood")
	app := filepath.Join(project, "Tetra.capsule")
	core := filepath.Join(project, "Core.capsule")
	source := filepath.Join(project, "src", "main.tetra")
	for _, path := range []string{app, core, source} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("dogfood fixture missing %s: %v", path, err)
		}
	}

	dir := t.TempDir()
	lockPath := filepath.Join(dir, "tetra.lock.json")
	pkgPath := filepath.Join(dir, "eco-dogfood.todex")
	unpackDir := filepath.Join(dir, "unpacked")
	store := filepath.Join(dir, "vault")
	registry := filepath.Join(dir, "registry")
	trustPath := filepath.Join(dir, "trust.snapshot.json")

	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"eco", "verify", "--target", "linux-x64", "--lock", lockPath, app, core}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco verify dogfood exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if out, err := testCommand(t, "go", "run", "./tools/cmd/validate-eco-lock", "--lock", lockPath).CombinedOutput(); err != nil {
		t.Fatalf("validate-eco-lock failed: %v\n%s", err, out)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "pack", "--project", app, "-o", pkgPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco pack dogfood exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "unpack", pkgPath, "-C", unpackDir}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco unpack dogfood exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if out, err := testCommand(t, "go", "run", "./tools/cmd/validate-eco-unpack", "--dir", unpackDir).CombinedOutput(); err != nil {
		t.Fatalf("validate-eco-unpack failed: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(unpackDir, "Tetra.capsule")); err != nil {
		t.Fatalf("unpacked dogfood capsule missing: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "vault", "add", "--store", store, "--kind", "source", source}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco vault add dogfood exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "vault", "verify", "--store", store}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco vault verify dogfood exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if out, err := testCommand(t, "go", "run", "./tools/cmd/validate-eco-vault", "--store", store).CombinedOutput(); err != nil {
		t.Fatalf("validate-eco-vault failed: %v\n%s", err, out)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "trust", "snapshot", "--lock", lockPath, "--store", store, "-o", trustPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco trust snapshot dogfood exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "publish", "--package", pkgPath, "--registry", registry, "--target", "linux-x64", "--trust", trustPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco publish dogfood exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if out, err := testCommand(t, "go", "run", "./tools/cmd/validate-eco-publish", "--registry", registry, "--id", "tetra://examples/eco-dogfood", "--version", "0.1.0", "--target", "linux-x64").CombinedOutput(); err != nil {
		t.Fatalf("validate-eco-publish failed: %v\n%s", err, out)
	}
	metaPath := filepath.Join(registry, "packages", "tetra_examples_eco_dogfood", "0.1.0", "linux-x64", "metadata.json")
	if _, err := os.Stat(metaPath); err != nil {
		t.Fatalf("dogfood publish metadata missing: %v", err)
	}
}

func TestEcoDocsDeclareLocalOnlyBetaScope(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	specPath := filepath.Join(root, "docs", "spec", "eco_publishing_v1.md")
	userPath := filepath.Join(root, "docs", "user", "eco_package_guide.md")
	for _, path := range []string{specPath, userPath} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(raw)
		if !strings.Contains(text, "local") {
			t.Fatalf("%s should declare local scope", path)
		}
		if !strings.Contains(text, "beta") {
			t.Fatalf("%s should declare beta boundary", path)
		}
		if !strings.Contains(text, "TetraHub") {
			t.Fatalf("%s should mention TetraHub boundary", path)
		}
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

func writeEcoProjectFixture(t *testing.T, dir string) (string, string) {
	t.Helper()
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
    permission "io"
`)
	return project, capsule
}

func writeTarGzFixture(t *testing.T, path string, name string, body []byte) {
	t.Helper()
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(out)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}

func tamperTodexEntry(t *testing.T, src string, dst string, entryName string, body []byte) {
	t.Helper()
	in, err := os.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	gz, err := gzip.NewReader(in)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	out, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	gzw := gzip.NewWriter(out)
	tw := tar.NewWriter(gzw)
	found := false
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		raw, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		next := *header
		if header.Name == entryName {
			raw = body
			next.Size = int64(len(raw))
			found = true
		}
		if err := tw.WriteHeader(&next); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(raw); err != nil {
			t.Fatal(err)
		}
	}
	if !found {
		t.Fatalf("entry %s not found in %s", entryName, src)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatal(err)
	}
}

func testCommand(t *testing.T, name string, args ...string) *exec.Cmd {
	t.Helper()
	root, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(name, args...)
	cmd.Dir = root
	return cmd
}
