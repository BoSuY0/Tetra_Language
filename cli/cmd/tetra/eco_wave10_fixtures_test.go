package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

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
