package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
