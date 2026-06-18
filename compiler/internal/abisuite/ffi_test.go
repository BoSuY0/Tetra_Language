package abisuite

import (
	"fmt"
	"strings"
	"testing"

	ctarget "tetra_language/compiler/target"
)

func TestFFIChecksUseBuildAndObjectCallbacks(t *testing.T) {
	var lastTarget string
	deps := FFICheckDeps{
		BuildLibrary: func(srcPath string, outPath string, target string) error {
			lastTarget = target
			switch {
			case strings.Contains(srcPath, "ffi_ref_null_return"):
				return fmt.Errorf("type mismatch: expected 'ref', got 'i32'")
			case strings.Contains(srcPath, "fnptr_param"):
				boundary := "i386"
				if target == "linux-x32" {
					boundary = "x32"
				}
				return fmt.Errorf("exported function 'ffi_fnptr_param' parameter 'cb' type 'fnptr' requires the %s pointer C ABI boundary; %s pointer C ABI boundary is not verified on %s", boundary, boundary, target)
			case strings.Contains(srcPath, "fnptr_return"):
				boundary := "i386"
				if target == "linux-x32" {
					boundary = "x32"
				}
				return fmt.Errorf("exported function 'ffi_fnptr_return' return type 'fnptr' requires the %s pointer C ABI boundary; %s pointer C ABI boundary is not verified on %s", boundary, boundary, target)
			default:
				return nil
			}
		},
		ReadObject: func(path string) (ObjectSummary, error) {
			return ObjectSummary{
				Target: lastTarget,
				Symbols: []ObjectSymbolSummary{
					{Name: "ffi_ptr_identity_c", HasSignature: true, ParamSlots: 1, ReturnSlots: 1},
					{Name: "ffi_rawptr_identity_c", HasSignature: true, ParamSlots: 1, ReturnSlots: 1},
					{Name: "ffi_nullable_ptr_identity_c", HasSignature: true, ParamSlots: 1, ReturnSlots: 1},
					{Name: "ffi_nullable_ptr_null_c", HasSignature: true, ParamSlots: 0, ReturnSlots: 1},
					{Name: "ffi_ref_identity_c", HasSignature: true, ParamSlots: 1, ReturnSlots: 1},
					{Name: "ffi_c_int_identity_c", HasSignature: true, ParamSlots: 1, ReturnSlots: 1},
					{Name: "ffi_c_uint_identity_c", HasSignature: true, ParamSlots: 1, ReturnSlots: 1},
					{Name: "ffi_usize_identity_c", HasSignature: true, ParamSlots: 1, ReturnSlots: 1},
					{Name: "ffi_isize_identity_c", HasSignature: true, ParamSlots: 1, ReturnSlots: 1},
					{Name: "ffi_size_t_identity_c", HasSignature: true, ParamSlots: 1, ReturnSlots: 1},
					{Name: "ffi_ssize_t_identity_c", HasSignature: true, ParamSlots: 1, ReturnSlots: 1},
					{Name: "ffi_native_int_identity_c", HasSignature: true, ParamSlots: 1, ReturnSlots: 1},
					{Name: "ffi_native_uint_identity_c", HasSignature: true, ParamSlots: 1, ReturnSlots: 1},
					{Name: "ffi_c_long_identity_c", HasSignature: true, ParamSlots: 1, ReturnSlots: 1},
					{Name: "ffi_c_ulong_identity_c", HasSignature: true, ParamSlots: 1, ReturnSlots: 1},
				},
			}, nil
		},
	}

	x86, err := ctarget.Parse("linux-x86")
	if err != nil {
		t.Fatalf("parse x86 target: %v", err)
	}
	x64, err := ctarget.Parse("linux-x64")
	if err != nil {
		t.Fatalf("parse x64 target: %v", err)
	}
	x32, err := ctarget.Parse("linux-x32")
	if err != nil {
		t.Fatalf("parse x32 target: %v", err)
	}

	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{name: "x86 pointer object", run: func() error { return CheckPointerFFIObjectSmoke(x86, deps) }},
		{name: "x32 pointer object", run: func() error { return CheckPointerFFIObjectSmoke(x32, deps) }},
		{name: "x64 c_int object", run: func() error { return CheckCIntFFIObjectSmoke(x64, deps) }},
		{name: "x86 c_uint object", run: func() error { return CheckCUIntFFIObjectSmoke(x86, deps) }},
		{name: "x32 native libc object", run: func() error { return CheckILP32NativeLibcFFIObjectSmoke(x32, deps) }},
		{name: "x86 ref null diagnostic", run: func() error { return CheckRefFFINullReturnDiagnostics("linux-x86", "x86", deps) }},
		{name: "x32 fnptr diagnostic", run: func() error { return CheckFunctionPointerFFIDiagnostics("linux-x32", "x32", "x32", deps) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err != nil {
				t.Fatalf("FFI check: %v", err)
			}
		})
	}
}
