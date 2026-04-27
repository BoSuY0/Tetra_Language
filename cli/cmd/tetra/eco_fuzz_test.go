package main

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzParseCapsuleDoesNotPanic(f *testing.F) {
	f.Add(`manifest "tetra.capsule.v1"
capsule App:
    id "tetra://app"
    version "0.1.0"
    target "linux-x64"
`)
	f.Add("capsule Broken:\n")
	f.Add("")
	f.Add("manifest \"wrong\"\n")

	f.Fuzz(func(t *testing.T, text string) {
		dir := t.TempDir()
		path := filepath.Join(dir, "Tetra.capsule")
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		_, _ = parseCapsule(path)
	})
}
