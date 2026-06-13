package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"tetra_language/internal/toon"
)

func TestJSONToTOONConvertsStrictJSON(t *testing.T) {
	got, err := jsonToTOON([]byte(`{"b":2,"a":[{"id":1,"name":"Ada"}]}`))
	if err != nil {
		t.Fatalf("jsonToTOON failed: %v", err)
	}
	if !bytes.Contains(got, []byte("a[1]{id,name}:")) || !bytes.Contains(got, []byte("b: 2")) {
		t.Fatalf("unexpected TOON output:\n%s", got)
	}
	roundTrip, err := toon.ConvertTOONToJSON(got, toon.Options{Strict: true})
	if err != nil {
		t.Fatalf("round trip failed: %v\n%s", err, got)
	}
	if !bytes.Contains(roundTrip, []byte(`"name":"Ada"`)) {
		t.Fatalf("unexpected round trip JSON: %s", roundTrip)
	}
}

func TestWriteOutputCreatesParentDirectory(t *testing.T) {
	out := filepath.Join(t.TempDir(), "nested", "report.toon")
	if err := writeOutput(out, []byte("status: pass")); err != nil {
		t.Fatalf("writeOutput failed: %v", err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(raw) != "status: pass\n" {
		t.Fatalf("output = %q", raw)
	}
}
