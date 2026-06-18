package buildruntime

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/format/tobj"
)

func TestRuntimeObjectSignatureAndAnnotation(t *testing.T) {
	sig, ok := RuntimeObjectSignature("__tetra_fs_exists")
	if !ok {
		t.Fatalf("RuntimeObjectSignature did not find filesystem symbol")
	}
	if sig.ParamSlots != 3 || sig.ReturnSlots != 1 {
		t.Fatalf("RuntimeObjectSignature slots = (%d, %d), want (3, 1)", sig.ParamSlots, sig.ReturnSlots)
	}

	obj := &tobj.Object{Symbols: []tobj.Symbol{
		{Name: "__tetra_fs_exists"},
		{Name: "__tetra_custom_runtime_hook"},
	}}
	AnnotateRuntimeObjectSignatures(obj)

	if !obj.Symbols[0].HasSignature {
		t.Fatalf("AnnotateRuntimeObjectSignatures did not mark known runtime symbol")
	}
	if obj.Symbols[0].ParamSlots != 3 || obj.Symbols[0].ReturnSlots != 1 {
		t.Fatalf("annotated slots = (%d, %d), want (3, 1)", obj.Symbols[0].ParamSlots, obj.Symbols[0].ReturnSlots)
	}
	if obj.Symbols[1].HasSignature {
		t.Fatalf("AnnotateRuntimeObjectSignatures marked unknown runtime symbol")
	}
}

func TestValidateRuntimeObjectSymbols(t *testing.T) {
	obj := &tobj.Object{Symbols: []tobj.Symbol{
		{Name: "__tetra_fs_exists", HasSignature: true, ParamSlots: 3, ReturnSlots: 1},
	}}
	if err := ValidateRuntimeObjectSymbols(obj, "missing filesystem runtime object", RequiredFilesystemRuntimeSymbols()); err != nil {
		t.Fatalf("ValidateRuntimeObjectSymbols error = %v", err)
	}

	err := ValidateRuntimeObjectSymbols(nil, "missing filesystem runtime object", RequiredFilesystemRuntimeSymbols())
	if err == nil || !strings.Contains(err.Error(), "missing filesystem runtime object") {
		t.Fatalf("nil object error = %v, want missing-object diagnostic", err)
	}

	bad := &tobj.Object{Symbols: []tobj.Symbol{
		{Name: "__tetra_fs_exists", HasSignature: true, ParamSlots: 2, ReturnSlots: 1},
	}}
	err = ValidateRuntimeObjectSymbols(bad, "missing filesystem runtime object", RequiredFilesystemRuntimeSymbols())
	if err == nil || !strings.Contains(err.Error(), "signature mismatch") {
		t.Fatalf("signature mismatch error = %v, want mismatch diagnostic", err)
	}
}
