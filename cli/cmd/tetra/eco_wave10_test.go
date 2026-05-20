package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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

func TestEcoUnpackRejectsArchiveSymlinkEntry(t *testing.T) {
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "symlink-entry.todex")
	writeTodexWithSymlinkEntry(t, pkgPath)

	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"eco", "unpack", pkgPath, "-C", filepath.Join(dir, "out")}, &stdout, &stderr); code == 0 {
		t.Fatalf("expected symlink archive entry rejection, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "unsupported archive entry type") {
		t.Fatalf("stderr = %q, want unsupported archive entry type", stderr.String())
	}
}

func TestEcoUnpackRejectsOutputSymlinkAncestor(t *testing.T) {
	dir := t.TempDir()
	_, capsule := writeEcoProjectFixture(t, dir)
	pkgPath := filepath.Join(dir, "demo.todex")
	outDir := filepath.Join(dir, "out")
	outside := filepath.Join(dir, "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(outDir, "src")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"eco", "pack", "--project", capsule, "-o", pkgPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco pack --project exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "unpack", pkgPath, "-C", outDir}, &stdout, &stderr); code == 0 {
		t.Fatalf("expected output symlink rejection, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "symlink") {
		t.Fatalf("stderr = %q, want symlink rejection", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(outside, "main.tetra")); !os.IsNotExist(err) {
		t.Fatalf("unpack wrote through symlink ancestor: %v", err)
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

func TestEcoTrustSnapshotRejectsUnreadableVaultStore(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, store string)
	}{
		{
			name: "missing index",
			setup: func(t *testing.T, store string) {
				if err := os.MkdirAll(store, 0o755); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "broken index",
			setup: func(t *testing.T, store string) {
				if err := os.MkdirAll(store, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(store, "records.json"), []byte("{not json\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			store := filepath.Join(dir, "vault")
			trustPath := filepath.Join(dir, "trust.json")
			tt.setup(t, store)

			var stdout, stderr bytes.Buffer
			if code := runCLI([]string{"eco", "verify", "--target", "linux-x64", "--lock", lockPath, capsule}, &stdout, &stderr); code != 0 {
				t.Fatalf("eco verify exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			stdout.Reset()
			stderr.Reset()
			if code := runCLI([]string{"eco", "trust", "snapshot", "--lock", lockPath, "--store", store, "-o", trustPath}, &stdout, &stderr); code == 0 {
				t.Fatalf("expected eco trust snapshot failure, stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), "read vault store") {
				t.Fatalf("unexpected stderr: %q", stderr.String())
			}
			if _, err := os.Stat(trustPath); err == nil {
				t.Fatalf("trust snapshot should not be written for unreadable vault store")
			} else if !os.IsNotExist(err) {
				t.Fatal(err)
			}
		})
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
	trustMeta, ok := meta["trust"].(map[string]any)
	if !ok {
		t.Fatalf("publish metadata missing trust object: %#v", meta)
	}
	if trustMeta["snapshot_file"] != "trust.snapshot.json" {
		t.Fatalf("trust snapshot file should be registry-local relative path: %#v", trustMeta)
	}
	if _, err := os.Stat(filepath.Join(registry, "packages", "tetra_demo", "0.1.0", "linux-x64", "trust.snapshot.json")); err != nil {
		t.Fatalf("published trust snapshot missing: %v", err)
	}
}

func TestEcoPublishStableChannelProducesProductionMetadata(t *testing.T) {
	dir := t.TempDir()
	project, capsule := writeEcoProjectFixture(t, dir)
	pkgPath := filepath.Join(dir, "demo.todex")
	registry := filepath.Join(dir, "registry")
	store := filepath.Join(dir, "vault")
	lockPath := filepath.Join(dir, "tetra.lock.json")
	trustPath := filepath.Join(dir, "trust.snapshot.json")
	downloadPath := filepath.Join(dir, "downloaded-stable.todex")

	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"eco", "pack", "--project", capsule, "-o", pkgPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco pack --project exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "vault", "add", "--store", store, "--kind", "source", filepath.Join(project, "src", "main.tetra")}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco vault add exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
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
	if code := runCLI([]string{"eco", "publish", "--package", pkgPath, "--registry", registry, "--target", "linux-x64", "--trust", trustPath, "--channel", "stable", "--hub", "production"}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco stable publish exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Published (stable)") {
		t.Fatalf("stable publish stdout = %q", stdout.String())
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
	if meta["schema"] != "tetra.eco.publish.v1" || meta["channel"] != "stable" || meta["hub"] != "production" {
		t.Fatalf("stable publish metadata = %#v", meta)
	}
	cmd := testCommand(t, "go", "run", "./tools/cmd/validate-eco-publish", "--registry", registry, "--id", "tetra://demo", "--version", "0.1.0", "--target", "linux-x64", "--channel", "stable")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("validate-eco-publish stable failed: %v\n%s", err, out)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "download", "--id", "tetra://demo", "--version", "0.1.0", "--target", "linux-x64", "--registry", registry, "-o", downloadPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco stable download exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(downloadPath); err != nil {
		t.Fatalf("stable downloaded package missing: %v", err)
	}
}

func TestEcoTetraHubStableChannelProducesProductionMetadata(t *testing.T) {
	dir := t.TempDir()
	project, capsule := writeEcoProjectFixture(t, dir)
	pkgPath := filepath.Join(dir, "demo.todex")
	store := filepath.Join(dir, "tetrahub")
	vaultStore := filepath.Join(dir, "vault")
	lockPath := filepath.Join(dir, "tetra.lock.json")
	trustPath := filepath.Join(dir, "trust.snapshot.json")
	downloadPath := filepath.Join(dir, "hub-downloaded-stable.todex")

	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"eco", "pack", "--project", capsule, "-o", pkgPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco pack --project exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "vault", "add", "--store", vaultStore, "--kind", "source", filepath.Join(project, "src", "main.tetra")}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco vault add exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "verify", "--target", "linux-x64", "--lock", lockPath, capsule}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco verify exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "trust", "snapshot", "--lock", lockPath, "--store", vaultStore, "-o", trustPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco trust snapshot exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "tetrahub", "publish", "--package", pkgPath, "--store", store, "--target", "linux-x64", "--trust", trustPath, "--channel", "stable"}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco tetrahub stable publish exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "TetraHub stable published") {
		t.Fatalf("stable tetrahub publish stdout = %q", stdout.String())
	}

	metaPath := filepath.Join(store, "packages", "tetra_demo", "0.1.0", "linux-x64", "metadata.json")
	rawMeta, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read tetrahub metadata: %v", err)
	}
	var meta map[string]any
	if err := json.Unmarshal(rawMeta, &meta); err != nil {
		t.Fatalf("decode tetrahub metadata: %v\n%s", err, string(rawMeta))
	}
	if meta["schema"] != "tetra.eco.publish.v1" || meta["channel"] != "stable" || meta["hub"] != "tetrahub-stable" {
		t.Fatalf("stable tetrahub metadata = %#v", meta)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "tetrahub", "download", "--id", "tetra://demo", "--version", "0.1.0", "--target", "linux-x64", "--store", store, "-o", downloadPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco tetrahub stable download exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(downloadPath); err != nil {
		t.Fatalf("stable tetrahub downloaded package missing: %v", err)
	}
}

func TestEcoTetraHubMirrorCopiesStablePackageAndWritesReport(t *testing.T) {
	dir := t.TempDir()
	project, capsule := writeEcoProjectFixture(t, dir)
	pkgPath := filepath.Join(dir, "demo.todex")
	sourceStore := filepath.Join(dir, "tetrahub-a")
	destStore := filepath.Join(dir, "tetrahub-b")
	vaultStore := filepath.Join(dir, "vault")
	lockPath := filepath.Join(dir, "tetra.lock.json")
	trustPath := filepath.Join(dir, "trust.snapshot.json")
	reportPath := filepath.Join(dir, "mirror.report.json")
	downloadPath := filepath.Join(dir, "mirrored-download.todex")

	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"eco", "pack", "--project", capsule, "-o", pkgPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco pack --project exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "vault", "add", "--store", vaultStore, "--kind", "source", filepath.Join(project, "src", "main.tetra")}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco vault add exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "verify", "--target", "linux-x64", "--lock", lockPath, capsule}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco verify exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "trust", "snapshot", "--lock", lockPath, "--store", vaultStore, "-o", trustPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco trust snapshot exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "tetrahub", "publish", "--package", pkgPath, "--store", sourceStore, "--target", "linux-x64", "--trust", trustPath, "--channel", "stable"}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco tetrahub stable publish exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "tetrahub", "mirror", "--from", sourceStore, "--to", destStore, "--id", "tetra://demo", "--version", "0.1.0", "--target", "linux-x64", "-o", reportPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco tetrahub mirror exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "TetraHub mirrored") {
		t.Fatalf("mirror stdout = %q", stdout.String())
	}
	rawReport, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read mirror report: %v", err)
	}
	var report map[string]any
	if err := json.Unmarshal(rawReport, &report); err != nil {
		t.Fatalf("decode mirror report: %v\n%s", err, string(rawReport))
	}
	if report["schema"] != "tetra.eco.mirror.v1" || report["id"] != "tetra://demo" || report["target"] != "linux-x64" {
		t.Fatalf("mirror report = %#v", report)
	}
	if report["package_sha256"] == "" || report["metadata_sha256"] == "" || report["trust_snapshot_sha256"] == "" {
		t.Fatalf("mirror report missing hashes: %#v", report)
	}
	cmd := testCommand(t, "go", "run", "./tools/cmd/validate-eco-mirror", "--mirror", reportPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("validate mirror report failed: %v\n%s", err, out)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "tetrahub", "download", "--id", "tetra://demo", "--version", "0.1.0", "--target", "linux-x64", "--store", destStore, "-o", downloadPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco tetrahub mirrored download exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	sourceMeta, err := os.ReadFile(filepath.Join(sourceStore, "packages", "tetra_demo", "0.1.0", "linux-x64", "metadata.json"))
	if err != nil {
		t.Fatalf("read source metadata: %v", err)
	}
	destMeta, err := os.ReadFile(filepath.Join(destStore, "packages", "tetra_demo", "0.1.0", "linux-x64", "metadata.json"))
	if err != nil {
		t.Fatalf("read mirrored metadata: %v", err)
	}
	if !bytes.Equal(sourceMeta, destMeta) {
		t.Fatalf("mirrored metadata changed\nsource=%s\ndest=%s", sourceMeta, destMeta)
	}
}

func TestEcoTetraHubFetchMirrorsStablePackageOverHTTP(t *testing.T) {
	dir := t.TempDir()
	project, capsule := writeEcoProjectFixture(t, dir)
	pkgPath := filepath.Join(dir, "demo.todex")
	sourceStore := filepath.Join(dir, "tetrahub-source")
	destStore := filepath.Join(dir, "tetrahub-fetched")
	vaultStore := filepath.Join(dir, "vault")
	lockPath := filepath.Join(dir, "tetra.lock.json")
	trustPath := filepath.Join(dir, "trust.snapshot.json")
	reportPath := filepath.Join(dir, "fetch.report.json")
	downloadPath := filepath.Join(dir, "fetched-download.todex")

	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"eco", "pack", "--project", capsule, "-o", pkgPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco pack --project exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "vault", "add", "--store", vaultStore, "--kind", "source", filepath.Join(project, "src", "main.tetra")}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco vault add exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "verify", "--target", "linux-x64", "--lock", lockPath, capsule}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco verify exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "trust", "snapshot", "--lock", lockPath, "--store", vaultStore, "-o", trustPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco trust snapshot exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "tetrahub", "publish", "--package", pkgPath, "--store", sourceStore, "--target", "linux-x64", "--trust", trustPath, "--channel", "stable"}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco tetrahub stable publish exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	server := httptest.NewServer(http.FileServer(http.Dir(sourceStore)))
	defer server.Close()

	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "tetrahub", "fetch", "--url", server.URL, "--to", destStore, "--id", "tetra://demo", "--version", "0.1.0", "--target", "linux-x64", "-o", reportPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco tetrahub fetch exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "TetraHub fetched") {
		t.Fatalf("fetch stdout = %q", stdout.String())
	}
	cmd := testCommand(t, "go", "run", "./tools/cmd/validate-eco-mirror", "--mirror", reportPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("validate fetch mirror report failed: %v\n%s", err, out)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "tetrahub", "download", "--id", "tetra://demo", "--version", "0.1.0", "--target", "linux-x64", "--store", destStore, "-o", downloadPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco tetrahub fetched download exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(downloadPath); err != nil {
		t.Fatalf("fetched package missing: %v", err)
	}
}

func TestEcoTetraHubFetchRejectsTamperedHTTPPackage(t *testing.T) {
	dir := t.TempDir()
	_, capsule := writeEcoProjectFixture(t, dir)
	pkgPath := filepath.Join(dir, "demo.todex")
	sourceStore := filepath.Join(dir, "tetrahub-source")
	destStore := filepath.Join(dir, "tetrahub-fetched")
	reportPath := filepath.Join(dir, "fetch.report.json")

	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"eco", "pack", "--project", capsule, "-o", pkgPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco pack --project exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "tetrahub", "publish", "--package", pkgPath, "--store", sourceStore, "--target", "linux-x64", "--channel", "stable"}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco tetrahub stable publish exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	publishedPackage := filepath.Join(sourceStore, "packages", "tetra_demo", "0.1.0", "linux-x64", "package.todex")
	raw, err := os.ReadFile(publishedPackage)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(publishedPackage, append(raw, []byte("tampered")...), 0o644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.FileServer(http.Dir(sourceStore)))
	defer server.Close()

	stdout.Reset()
	stderr.Reset()
	code := runCLI([]string{"eco", "tetrahub", "fetch", "--url", server.URL, "--to", destStore, "--id", "tetra://demo", "--version", "0.1.0", "--target", "linux-x64", "-o", reportPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected fetch failure for tampered package, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "package size mismatch") && !strings.Contains(stderr.String(), "package hash mismatch") {
		t.Fatalf("stderr = %q, want package integrity failure", stderr.String())
	}
	if _, err := os.Stat(reportPath); !os.IsNotExist(err) {
		t.Fatalf("fetch report should not be written after integrity failure: %v", err)
	}
}

func TestEcoTetraHubMirrorRejectsTamperedSourcePackage(t *testing.T) {
	dir := t.TempDir()
	_, capsule := writeEcoProjectFixture(t, dir)
	pkgPath := filepath.Join(dir, "demo.todex")
	sourceStore := filepath.Join(dir, "tetrahub-a")
	destStore := filepath.Join(dir, "tetrahub-b")
	reportPath := filepath.Join(dir, "mirror.report.json")

	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"eco", "pack", "--project", capsule, "-o", pkgPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco pack --project exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "tetrahub", "publish", "--package", pkgPath, "--store", sourceStore, "--target", "linux-x64", "--channel", "stable"}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco tetrahub stable publish exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	publishedPackage := filepath.Join(sourceStore, "packages", "tetra_demo", "0.1.0", "linux-x64", "package.todex")
	raw, err := os.ReadFile(publishedPackage)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(publishedPackage, append(raw, []byte("tampered")...), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	code := runCLI([]string{"eco", "tetrahub", "mirror", "--from", sourceStore, "--to", destStore, "--id", "tetra://demo", "--version", "0.1.0", "--target", "linux-x64", "-o", reportPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected mirror failure for tampered package, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "package size mismatch") && !strings.Contains(stderr.String(), "package hash mismatch") {
		t.Fatalf("stderr = %q, want package integrity failure", stderr.String())
	}
	if _, err := os.Stat(reportPath); !os.IsNotExist(err) {
		t.Fatalf("mirror report should not be written after integrity failure: %v", err)
	}
}

func TestEcoTetraHubMirrorRejectsDestinationSymlinkTraversal(t *testing.T) {
	dir := t.TempDir()
	_, capsule := writeEcoProjectFixture(t, dir)
	pkgPath := filepath.Join(dir, "demo.todex")
	sourceStore := filepath.Join(dir, "tetrahub-a")
	destStore := filepath.Join(dir, "tetrahub-b")
	outside := filepath.Join(dir, "outside")
	reportPath := filepath.Join(dir, "mirror.report.json")

	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"eco", "pack", "--project", capsule, "-o", pkgPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco pack --project exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"eco", "tetrahub", "publish", "--package", pkgPath, "--store", sourceStore, "--target", "linux-x64", "--channel", "stable"}, &stdout, &stderr); code != 0 {
		t.Fatalf("eco tetrahub stable publish exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	targetParent := filepath.Join(destStore, "packages", "tetra_demo", "0.1.0")
	if err := os.MkdirAll(targetParent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(targetParent, "linux-x64")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	code := runCLI([]string{"eco", "tetrahub", "mirror", "--from", sourceStore, "--to", destStore, "--id", "tetra://demo", "--version", "0.1.0", "--target", "linux-x64", "-o", reportPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected mirror symlink destination failure, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "symlink") {
		t.Fatalf("stderr = %q, want symlink rejection", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(outside, "package.todex")); !os.IsNotExist(err) {
		t.Fatalf("mirror wrote through destination symlink: %v", err)
	}
	if _, err := os.Stat(reportPath); !os.IsNotExist(err) {
		t.Fatalf("mirror report should not be written after symlink failure: %v", err)
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

func TestEcoDownloadRejectsPublishMetadataUnknownFieldsAndKeyMismatches(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(map[string]any)
		rawSuffix string
		want      string
	}{
		{
			name: "unknown field",
			mutate: func(meta map[string]any) {
				meta["unexpected"] = true
			},
			want: "unknown field",
		},
		{
			name: "extra field",
			mutate: func(meta map[string]any) {
				meta["extra"] = map[string]any{"note": "not in publish contract"}
			},
			want: "unknown field",
		},
		{
			name: "capsule id mismatch",
			mutate: func(meta map[string]any) {
				capsule := meta["capsule"].(map[string]any)
				capsule["id"] = "tetra://other"
			},
			want: "capsule id mismatch",
		},
		{
			name: "download target mismatch",
			mutate: func(meta map[string]any) {
				downloads := meta["downloads"].([]any)
				download := downloads[0].(map[string]any)
				download["target"] = "windows-x64"
			},
			want: "download target mismatch",
		},
		{
			name:      "trailing JSON payload",
			rawSuffix: "\n{}",
			want:      "trailing",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			_, capsule := writeEcoProjectFixture(t, dir)
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

			metaPath := filepath.Join(registry, "packages", "tetra_demo", "0.1.0", "linux-x64", "metadata.json")
			rawMeta, err := os.ReadFile(metaPath)
			if err != nil {
				t.Fatalf("read publish metadata: %v", err)
			}
			var meta map[string]any
			if err := json.Unmarshal(rawMeta, &meta); err != nil {
				t.Fatalf("decode publish metadata: %v\n%s", err, string(rawMeta))
			}
			if tt.mutate != nil {
				tt.mutate(meta)
			}
			rawMeta, err = json.MarshalIndent(meta, "", "  ")
			if err != nil {
				t.Fatalf("encode publish metadata: %v", err)
			}
			rawMeta = append(rawMeta, tt.rawSuffix...)
			if err := os.WriteFile(metaPath, append(rawMeta, '\n'), 0o644); err != nil {
				t.Fatal(err)
			}

			stdout.Reset()
			stderr.Reset()
			if code := runCLI([]string{"eco", "download", "--id", "tetra://demo", "--version", "0.1.0", "--target", "linux-x64", "--registry", registry, "-o", downloadPath}, &stdout, &stderr); code == 0 {
				t.Fatalf("expected eco download failure, stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("unexpected stderr: got %q, want %q", stderr.String(), tt.want)
			}
		})
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

func writeTodexWithSymlinkEntry(t *testing.T, path string) {
	t.Helper()
	capsule := []byte(`manifest "tetra.capsule.v1"
capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    target "linux-x64"
`)
	emptySum := sha256.Sum256(nil)
	capsuleSum := sha256.Sum256(capsule)
	files := []ecoPackageMetadataFile{
		{Path: "Capsule.t4", SHA256: "sha256:" + hex.EncodeToString(capsuleSum[:]), Size: int64(len(capsule))},
		{Path: "src/main.tetra", SHA256: "sha256:" + hex.EncodeToString(emptySum[:]), Size: 0},
	}
	metadata := ecoPackageMetadata{
		Schema:           ecoPackageSchemaV1,
		Compression:      "gzip",
		MTimeUnix:        0,
		Reproducible:     true,
		ManifestSchema:   capsuleManifestSchemaV1,
		PermissionsModel: ecoPermissionsModelV1,
		FileCount:        len(files),
		Files:            files,
	}
	fingerprintSum := sha256.Sum256([]byte(packageMetadataFingerprint(files)))
	metadata.BuildInputsSHA = "sha256:" + hex.EncodeToString(fingerprintSum[:])
	rawMetadata, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	rawMetadata = append(rawMetadata, '\n')

	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(out)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "Capsule.t4", Mode: 0o644, Size: int64(len(capsule))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(capsule); err != nil {
		t.Fatal(err)
	}
	if err := tw.WriteHeader(&tar.Header{Name: "src/main.tetra", Typeflag: tar.TypeSymlink, Linkname: "../outside.tetra", Mode: 0o777}); err != nil {
		t.Fatal(err)
	}
	if err := tw.WriteHeader(&tar.Header{Name: ecoPackageMetadataPath, Mode: 0o644, Size: int64(len(rawMetadata))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(rawMetadata); err != nil {
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
