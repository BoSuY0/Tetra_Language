package abisuite

import (
	"testing"

	"tetra_language/compiler/internal/semantics"
	ctarget "tetra_language/compiler/target"
)

func TestWASMABIChecks(t *testing.T) {
	for _, targetName := range []string{"wasm32-wasi", "wasm32-web"} {
		t.Run(targetName, func(t *testing.T) {
			tgt, err := ctarget.Parse(targetName)
			if err != nil {
				t.Fatalf("parse target: %v", err)
			}
			for _, check := range []struct {
				name string
				run  func(ctarget.Target) error
			}{
				{name: "target model", run: CheckWASMTargetModel},
				{name: "slot ABI metadata", run: CheckWASMSlotABIMetadata},
				{name: "aggregate return layouts", run: CheckWASMAggregateReturnLayouts},
				{name: "call boundary validation", run: CheckWASMCallBoundaryValidation},
				{name: "FFI repr(C) policy", run: CheckWASMFFIReprCBoundaryPolicy},
			} {
				if err := check.run(tgt); err != nil {
					t.Fatalf("%s: %v", check.name, err)
				}
			}
		})
	}
}

func TestWASMFFIReprCBoundaryPolicyPredicates(t *testing.T) {
	for _, targetName := range []string{"wasm32-wasi", "wasm32-web"} {
		if TargetRequiresExplicitAggregateExportGate(targetName) {
			t.Fatalf("%s unexpectedly requires native aggregate C ABI gate", targetName)
		}
		if TargetRequiresExplicitPointerExportGate(targetName) {
			t.Fatalf("%s unexpectedly requires native pointer C ABI gate", targetName)
		}
	}
	for _, targetName := range []string{"linux-x86", "linux-x64", "linux-x32", "macos-x64", "windows-x64"} {
		if !TargetRequiresExplicitAggregateExportGate(targetName) {
			t.Fatalf("%s lost native aggregate C ABI gate", targetName)
		}
	}

	types := map[string]*semantics.TypeInfo{
		"Pair":   {Name: "Pair", Kind: semantics.TypeStruct},
		"Bytes":  {Name: "Bytes", Kind: semantics.TypeSlice},
		"String": {Name: "String", Kind: semantics.TypeStr},
		"Choice": {Name: "Choice", Kind: semantics.TypeEnum},
	}
	for _, typeName := range []string{"Pair", "Bytes", "String", "Choice"} {
		if !TargetExportedFFIRequiresAggregateABI(typeName, types) {
			t.Fatalf("aggregate FFI detector did not recognize %s", typeName)
		}
	}
	if TargetExportedFFIRequiresAggregateABI("i32", types) {
		t.Fatalf("aggregate FFI detector should not recognize missing scalar i32 as aggregate")
	}
}
