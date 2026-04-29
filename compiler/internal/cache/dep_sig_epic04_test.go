package cache

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/semantics"
)

func TestDepSigHashFromDepsMissingFunctionSignatureDiagnostic(t *testing.T) {
	_, err := DepSigHashFromDeps(
		[]string{"engine.math.add_one"},
		nil,
		nil,
		nil,
	)
	if err == nil {
		t.Fatalf("expected missing function signature error")
	}
	if !strings.Contains(err.Error(), "missing signature for 'engine.math.add_one'") {
		t.Fatalf("error = %v", err)
	}
}

func TestDepSigHashFromDepsMissingTypeSignatureDiagnosticText(t *testing.T) {
	_, err := DepSigHashFromDeps(
		nil,
		[]string{"engine.types.Vec"},
		nil,
		nil,
	)
	if err == nil {
		t.Fatalf("expected missing type signature error")
	}
	if !strings.Contains(err.Error(), "missing type signature for 'engine.types.Vec'") {
		t.Fatalf("error = %v", err)
	}
}

func TestDepSigHashFromDepsIncludesInterfaceHashes(t *testing.T) {
	sigs := map[string]semantics.FuncSig{
		"math.core.add": {ParamTypes: []string{"i32", "i32"}, ReturnType: "i32", Public: true},
	}
	hash1, err := DepSigHashFromDepsWithInterfaceHashes(
		[]string{"math.core.add"},
		nil,
		sigs,
		nil,
		map[string]string{"math.core": "sha256:1111"},
	)
	if err != nil {
		t.Fatalf("DepSigHashFromDepsWithInterfaceHashes hash1: %v", err)
	}
	hash2, err := DepSigHashFromDepsWithInterfaceHashes(
		[]string{"math.core.add"},
		nil,
		sigs,
		nil,
		map[string]string{"math.core": "sha256:2222"},
	)
	if err != nil {
		t.Fatalf("DepSigHashFromDepsWithInterfaceHashes hash2: %v", err)
	}
	if hash1 == hash2 {
		t.Fatalf("interface hash change did not affect dep signature")
	}
}
