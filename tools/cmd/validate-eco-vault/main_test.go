package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateEcoVaultAcceptsValidStore(t *testing.T) {
	root := makeVaultStore(t, []byte("hello tetra\n"), 0, "")
	out, err := runVaultValidator(t, root)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateEcoVaultRejectsUnknownTopLevelField(t *testing.T) {
	root := makeVaultStore(t, []byte("hello tetra\n"), 0, "")
	raw, err := os.ReadFile(filepath.Join(root, "records.json"))
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), "{\n", "{\n  \"extra\": true,\n", 1))
	if err := os.WriteFile(filepath.Join(root, "records.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runVaultValidator(t, root)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), `unknown field "extra"`) {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoVaultRejectsUnknownRecordField(t *testing.T) {
	root := makeVaultStore(t, []byte("hello tetra\n"), 0, "")
	raw, err := os.ReadFile(filepath.Join(root, "records.json"))
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `"size": `, `"extra": true, "size": `, 1))
	if err := os.WriteFile(filepath.Join(root, "records.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runVaultValidator(t, root)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), `unknown field "extra"`) {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoVaultRejectsMissingRecords(t *testing.T) {
	root := t.TempDir()
	out, err := runVaultValidator(t, root)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing records.json") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoVaultRejectsMissingObject(t *testing.T) {
	root := makeVaultStore(t, []byte("hello tetra\n"), 0, "")
	if err := os.Remove(filepath.Join(root, "objects", "sha256", onlyRecordHex(t, root))); err != nil {
		t.Fatal(err)
	}
	out, err := runVaultValidator(t, root)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing object") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoVaultRejectsHashMismatch(t *testing.T) {
	root := makeVaultStore(t, []byte("hello tetra\n"), 0, "")
	objectPath := filepath.Join(root, "objects", "sha256", onlyRecordHex(t, root))
	if err := os.WriteFile(objectPath, []byte("jello tetra\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runVaultValidator(t, root)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "hash mismatch") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoVaultRejectsSizeMismatch(t *testing.T) {
	root := makeVaultStore(t, []byte("hello tetra\n"), 99, "")
	out, err := runVaultValidator(t, root)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "size mismatch") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoVaultRejectsInvalidHash(t *testing.T) {
	root := makeVaultStore(t, []byte("hello tetra\n"), 0, "sha256:not-hex")
	out, err := runVaultValidator(t, root)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "invalid sha256 hash") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoVaultRejectsUnsupportedKind(t *testing.T) {
	root := makeVaultStore(t, []byte("hello tetra\n"), 0, "")
	raw, err := os.ReadFile(filepath.Join(root, "records.json"))
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `"kind": "source"`, `"kind": "blob"`, 1))
	if err := os.WriteFile(filepath.Join(root, "records.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runVaultValidator(t, root)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unsupported kind blob") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoVaultRejectsDuplicateRecordIdentity(t *testing.T) {
	root := makeVaultStore(t, []byte("hello tetra\n"), 0, "")
	raw, err := os.ReadFile(filepath.Join(root, "records.json"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	recordStart := strings.Index(text, `{"hash":`)
	if recordStart < 0 {
		t.Fatalf("record not found:\n%s", text)
	}
	record := strings.TrimSuffix(strings.TrimSpace(text[recordStart:strings.LastIndex(text, "\n  ]")]), "\n")
	text = strings.Replace(text, "\n  ]", ",\n    "+record+"\n  ]", 1)
	if err := os.WriteFile(filepath.Join(root, "records.json"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runVaultValidator(t, root)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "duplicate record") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoVaultAllowsSameObjectForDifferentKinds(t *testing.T) {
	root := makeVaultStore(t, []byte("hello tetra\n"), 0, "")
	raw, err := os.ReadFile(filepath.Join(root, "records.json"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	second := strings.Replace(text[strings.Index(text, `{"hash":`):strings.LastIndex(text, "\n  ]")], `"kind": "source"`, `"kind": "interface"`, 1)
	text = strings.Replace(text, "\n  ]", ",\n    "+strings.TrimSpace(second)+"\n  ]", 1)
	if err := os.WriteFile(filepath.Join(root, "records.json"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runVaultValidator(t, root)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func makeVaultStore(t *testing.T, object []byte, sizeOverride int, hashOverride string) string {
	t.Helper()
	root := t.TempDir()
	sum := sha256.Sum256(object)
	hexSum := hex.EncodeToString(sum[:])
	hash := "sha256:" + hexSum
	if hashOverride != "" {
		hash = hashOverride
	}
	if err := os.MkdirAll(filepath.Join(root, "objects", "sha256"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "objects", "sha256", hexSum), object, 0o644); err != nil {
		t.Fatal(err)
	}
	size := len(object)
	if sizeOverride != 0 {
		size = sizeOverride
	}
	records := fmt.Sprintf(`{
  "records": [
    {"hash": %q, "kind": "source", "source": "examples/flow_hello.tetra", "size": %d}
  ]
}
`, hash, size)
	if err := os.WriteFile(filepath.Join(root, "records.json"), []byte(records), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func onlyRecordHex(t *testing.T, root string) string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "objects", "sha256"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("object count = %d, want 1", len(entries))
	}
	return entries[0].Name()
}

func runVaultValidator(t *testing.T, root string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command("go", "run", ".", "--store", root)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}
