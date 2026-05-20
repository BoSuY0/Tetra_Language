package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestValidateEcoUnpackAcceptsProjectBundle(t *testing.T) {
	root := makeUnpackedProject(t, true, true)
	out, err := runUnpackValidator(t, root)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateEcoUnpackAcceptsT4CapsuleProjectBundle(t *testing.T) {
	root := makeUnpackedProjectWithFiles(t, "Capsule.t4", "src/main.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    target "linux-x64"
`, "func main() -> Int:\n    return 0\n")
	out, err := runUnpackValidator(t, root)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateEcoUnpackAcceptsManifestV1SourcesTargetsProjectBundle(t *testing.T) {
	root := makeUnpackedProjectWithFiles(t, "Capsule.t4", "app/main.t4", `manifest "tetra.capsule.v1"
capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "app/main.t4"
    sources:
        app
    targets:
        linux-x64
        wasm32-wasi
`, "func main() -> Int:\n    return 0\n")
	out, err := runUnpackValidator(t, root)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateEcoUnpackAcceptsFormatterStyleIndentedManifest(t *testing.T) {
	root := makeUnpackedProjectWithManifest(t, `capsule App:
    id "tetra://app"
    version "0.1.0"
    target "linux-x64"
`, true)
	out, err := runUnpackValidator(t, root)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateEcoUnpackRejectsMissingCapsuleManifest(t *testing.T) {
	root := makeUnpackedProject(t, false, true)
	out, err := runUnpackValidator(t, root)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing Capsule.t4") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoUnpackRejectsMissingSources(t *testing.T) {
	root := makeUnpackedProject(t, true, false)
	out, err := runUnpackValidator(t, root)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing T4 sources under src") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoUnpackRejectsIncompleteManifest(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Tetra.capsule"), []byte("capsule App:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.tetra"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runUnpackValidator(t, root)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "manifest missing id") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoUnpackRejectsInvalidSource(t *testing.T) {
	root := t.TempDir()
	raw := `capsule App:
  id "tetra://app"
  version "0.1.0"
  target "linux-x64"
`
	if err := os.WriteFile(filepath.Join(root, "Tetra.capsule"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.tetra"), []byte("func main() -> Int:\n\treturn 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runUnpackValidator(t, root)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "parse failed") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoUnpackRejectsMissingPackageMetadata(t *testing.T) {
	root := makeUnpackedProject(t, true, true)
	if err := os.Remove(filepath.Join(root, "tetra.package.json")); err != nil {
		t.Fatal(err)
	}
	out, err := runUnpackValidator(t, root)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing tetra.package.json") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoUnpackRejectsMetadataHashMismatch(t *testing.T) {
	root := makeUnpackedProject(t, true, true)
	if err := os.WriteFile(filepath.Join(root, "src", "main.tetra"), []byte("func main() -> Int:\n    return 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runUnpackValidator(t, root)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "metadata hash mismatch") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoUnpackRejectsSymlinkSourceFile(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.tetra")
	if err := os.WriteFile(outside, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Capsule.t4"), []byte(`capsule App:
    id "tetra://app"
    version "0.1.0"
    target "linux-x64"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "src", "main.tetra")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	writeUnpackMetadataFixture(t, root)

	out, err := runUnpackValidator(t, root)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "symlink") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoUnpackAcceptsReproducibleMetadataFields(t *testing.T) {
	root := makeUnpackedProject(t, true, true)
	raw, err := os.ReadFile(filepath.Join(root, "tetra.package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var meta unpackPackageMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatal(err)
	}
	meta.Reproducible = true
	meta.BuildInputsSHA = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	meta.ManifestSchema = "tetra.capsule.v1"
	meta.PermissionsModel = "tetra.eco.permissions.v1"
	raw, err = json.MarshalIndent(meta, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(root, "tetra.package.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runUnpackValidator(t, root)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func makeUnpackedProject(t *testing.T, manifest bool, source bool) string {
	t.Helper()
	if manifest {
		return makeUnpackedProjectWithManifest(t, `capsule App:
  id "tetra://app"
  version "0.1.0"
  target "linux-x64"
`, source)
	}
	return makeUnpackedProjectWithManifest(t, "", source)
}

func makeUnpackedProjectWithManifest(t *testing.T, manifest string, source bool) string {
	t.Helper()
	root := t.TempDir()
	if manifest != "" {
		if err := os.WriteFile(filepath.Join(root, "Tetra.capsule"), []byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if source {
		if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "src", "main.tetra"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeUnpackMetadataFixture(t, root)
	return root
}

func makeUnpackedProjectWithFiles(t *testing.T, manifestRel string, sourceRel string, manifest string, source string) string {
	t.Helper()
	root := t.TempDir()
	if manifestRel != "" {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(manifestRel)), []byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if sourceRel != "" {
		sourcePath := filepath.Join(root, filepath.FromSlash(sourceRel))
		if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeUnpackMetadataFixture(t, root)
	return root
}

type unpackPackageMetadata struct {
	Schema           string                  `json:"schema"`
	Compression      string                  `json:"compression"`
	MTimeUnix        int64                   `json:"mtime_unix"`
	Reproducible     bool                    `json:"reproducible,omitempty"`
	BuildInputsSHA   string                  `json:"build_inputs_sha256,omitempty"`
	ManifestSchema   string                  `json:"manifest_schema,omitempty"`
	PermissionsModel string                  `json:"permissions_model,omitempty"`
	FileCount        int                     `json:"file_count"`
	Files            []unpackPackageFileMeta `json:"files"`
}

type unpackPackageFileMeta struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

func writeUnpackMetadataFixture(t *testing.T, root string) {
	t.Helper()
	var files []unpackPackageFileMeta
	if err := filepath.WalkDir(root, func(abs string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, abs)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "tetra.package.json" {
			return nil
		}
		raw, err := os.ReadFile(abs)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(raw)
		files = append(files, unpackPackageFileMeta{
			Path:   rel,
			SHA256: "sha256:" + hex.EncodeToString(sum[:]),
			Size:   int64(len(raw)),
		})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	meta := unpackPackageMetadata{
		Schema:      "tetra.eco.package.v1",
		Compression: "gzip",
		MTimeUnix:   0,
		FileCount:   len(files),
		Files:       files,
	}
	raw, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(root, "tetra.package.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func runUnpackValidator(t *testing.T, root string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command("go", "run", ".", "--dir", root)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}
