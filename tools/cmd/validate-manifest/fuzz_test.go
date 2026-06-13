package main

import "testing"

const maxValidateManifestFuzzInputBytes = 512

func shouldSkipValidateManifestFuzzInput(raw []byte) bool {
	return len(raw) > maxValidateManifestFuzzInputBytes
}

func TestValidateManifestFuzzSkipsOversizedInputs(t *testing.T) {
	if shouldSkipValidateManifestFuzzInput(make([]byte, maxValidateManifestFuzzInputBytes)) {
		t.Fatalf("max-sized fuzz input should still run")
	}
	if !shouldSkipValidateManifestFuzzInput(make([]byte, maxValidateManifestFuzzInputBytes+1)) {
		t.Fatalf("oversized fuzz input should be skipped")
	}
}

func FuzzValidateManifestDoesNotPanic(f *testing.F) {
	f.Add([]byte(`{
  "compiler_version": "v0.1.0",
  "targets": [],
  "builtins": [],
  "runtime_abi": {}
}`))
	f.Add([]byte(`{"targets":null,"builtins":[]}`))
	f.Add([]byte(""))
	f.Add([]byte{0x00, 0xff, '{', '}'})

	f.Fuzz(func(t *testing.T, raw []byte) {
		if shouldSkipValidateManifestFuzzInput(raw) {
			t.Skip("validate-manifest fuzz input exceeds bounded manifest-shape size")
		}
		_ = validateManifest(raw)
	})
}

func TestManifestNegativePropertySuite(t *testing.T) {
	tests := map[string][]byte{
		"missing compiler": []byte(`{"targets":[],"builtins":[],"runtime_abi":{}}`),
		"targets null":     []byte(`{"compiler_version":"v0.1.0","targets":null,"builtins":[],"runtime_abi":{}}`),
		"builtins null":    []byte(`{"compiler_version":"v0.1.0","targets":[],"builtins":null,"runtime_abi":{}}`),
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			if err := validateManifest(raw); err == nil {
				t.Fatalf("expected validation error")
			}
		})
	}
}
