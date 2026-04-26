package main

import (
	"os"
	"os/exec"
	"path/filepath"
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
	if !strings.Contains(string(out), "missing Tetra.capsule") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoUnpackRejectsMissingSources(t *testing.T) {
	root := makeUnpackedProject(t, true, false)
	out, err := runUnpackValidator(t, root)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing .tetra sources under src") {
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
	return root
}

func runUnpackValidator(t *testing.T, root string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command("go", "run", ".", "--dir", root)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}
