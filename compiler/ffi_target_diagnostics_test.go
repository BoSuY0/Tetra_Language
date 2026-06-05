package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestX32PointerFFIGateCoversOnlyUnverifiedPointerLikeAndFunctionPointerSpellings(t *testing.T) {
	for _, typeName := range []string{
		"fnptr",
		"fn(Int) -> Int",
	} {
		t.Run(typeName, func(t *testing.T) {
			if !targetExportedFFIRequiresX32PointerBoundaryGate("linux-x32", typeName) {
				t.Fatalf("linux-x32 FFI gate did not cover %q", typeName)
			}
			if targetExportedFFIRequiresX32PointerBoundaryGate("linux-x64", typeName) {
				t.Fatalf("linux-x64 FFI gate unexpectedly covered %q", typeName)
			}
		})
	}

	for _, typeName := range []string{
		"ptr",
		"rawptr",
		"nullable_ptr",
		"ref",
		"i32",
		"u32",
		"c_int",
		"c_uint",
		"bool",
		"usize",
		"isize",
		"size_t",
		"ssize_t",
		"native_int",
		"native_uint",
		"c_long",
		"c_ulong",
	} {
		t.Run(typeName, func(t *testing.T) {
			if targetExportedFFIRequiresX32PointerBoundaryGate("linux-x32", typeName) {
				t.Fatalf("linux-x32 FFI gate should not cover scalar or source-level target-layout type %q", typeName)
			}
		})
	}
}

func TestNativeTargetsRejectExportedAggregateFFIParameters(t *testing.T) {
	src := `repr(C) struct Pair:
    lo: Int
    hi: Int

@export("ffi_pair_c")
func ffi_pair(p: Pair) -> Int:
    return p.lo + p.hi
`

	for _, target := range []string{"linux-x86", "linux-x64", "linux-x32", "macos-x64", "windows-x64"} {
		t.Run(target, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "ffi_pair.t4")
			outPath := filepath.Join(tmp, "ffi_pair.tobj")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			_, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Emit: EmitLibrary, Jobs: 1})
			if err == nil {
				t.Fatalf("expected aggregate FFI diagnostic")
			}
			for _, want := range []string{
				"exported function 'ffi_pair'",
				"parameter 'p'",
				"type 'Pair'",
				"aggregate C ABI is not supported on " + target,
			} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("diagnostic = %v, want substring %q", err, want)
				}
			}
			if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
				t.Fatalf("output object should not be written, stat error = %v", statErr)
			}
		})
	}
}

func TestNativeTargetsRejectExportedAggregateFFIReturns(t *testing.T) {
	src := `repr(C) struct Pair:
    lo: Int
    hi: Int

@export("ffi_make_pair_c")
func ffi_make_pair() -> Pair:
    return Pair(lo: 1, hi: 2)
`

	for _, target := range []string{"linux-x86", "linux-x64", "linux-x32", "macos-x64", "windows-x64"} {
		t.Run(target, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "ffi_make_pair.t4")
			outPath := filepath.Join(tmp, "ffi_make_pair.tobj")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			_, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Emit: EmitLibrary, Jobs: 1})
			if err == nil {
				t.Fatalf("expected aggregate FFI diagnostic")
			}
			for _, want := range []string{
				"exported function 'ffi_make_pair'",
				"return type 'Pair'",
				"aggregate C ABI is not supported on " + target,
			} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("diagnostic = %v, want substring %q", err, want)
				}
			}
			if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
				t.Fatalf("output object should not be written, stat error = %v", statErr)
			}
		})
	}
}

func TestLinuxX86AndX32BuildExportedPointerFFIBoundaryObjects(t *testing.T) {
	src := `@export("ffi_ptr_identity_c")
func ffi_ptr_identity(p: ptr) -> ptr:
    return p

@export("ffi_rawptr_identity_c")
func ffi_rawptr_identity(p: rawptr) -> rawptr:
    return p

@export("ffi_nullable_ptr_identity_c")
func ffi_nullable_ptr_identity(p: nullable_ptr) -> nullable_ptr:
    return p

@export("ffi_nullable_ptr_null_c")
func ffi_nullable_ptr_null() -> nullable_ptr:
    return 0

@export("ffi_ref_identity_c")
func ffi_ref_identity(p: ref) -> ref:
    return p
`

	for _, target := range []string{"linux-x86", "linux-x32"} {
		t.Run(target, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "ffi_ptr_identity.t4")
			outPath := filepath.Join(tmp, "ffi_ptr_identity.tobj")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Emit: EmitLibrary, Jobs: 1}); err != nil {
				t.Fatalf("build %s pointer FFI object: %v", target, err)
			}
			obj, err := ReadObject(outPath)
			if err != nil {
				t.Fatalf("read object: %v", err)
			}
			if obj.Target != target {
				t.Fatalf("object target = %q, want %s", obj.Target, target)
			}
			if !abiSuiteObjectHasSymbolSignature(obj, "ffi_ptr_identity_c", 1, 1) {
				t.Fatalf("%s object missing exported ffi_ptr_identity_c(1)->1 symbol: %#v", target, obj.Symbols)
			}
			if !abiSuiteObjectHasSymbolSignature(obj, "ffi_rawptr_identity_c", 1, 1) {
				t.Fatalf("%s object missing exported ffi_rawptr_identity_c(1)->1 symbol: %#v", target, obj.Symbols)
			}
			if !abiSuiteObjectHasSymbolSignature(obj, "ffi_nullable_ptr_identity_c", 1, 1) {
				t.Fatalf("%s object missing exported ffi_nullable_ptr_identity_c(1)->1 symbol: %#v", target, obj.Symbols)
			}
			if !abiSuiteObjectHasSymbolSignature(obj, "ffi_nullable_ptr_null_c", 0, 1) {
				t.Fatalf("%s object missing exported ffi_nullable_ptr_null_c(0)->1 symbol: %#v", target, obj.Symbols)
			}
			if !abiSuiteObjectHasSymbolSignature(obj, "ffi_ref_identity_c", 1, 1) {
				t.Fatalf("%s object missing exported ffi_ref_identity_c(1)->1 symbol: %#v", target, obj.Symbols)
			}
		})
	}
}

func TestLinuxX86AndX32RejectRefNullReturnWithoutObject(t *testing.T) {
	src := `@export("ffi_ref_null_c")
func ffi_ref_null() -> ref:
    return 0
`

	for _, target := range []string{"linux-x86", "linux-x32"} {
		t.Run(target, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "ffi_ref_null.t4")
			outPath := filepath.Join(tmp, "ffi_ref_null.tobj")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			_, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Emit: EmitLibrary, Jobs: 1})
			if err == nil {
				t.Fatalf("expected %s ref null-return diagnostic", target)
			}
			for _, want := range []string{
				"type mismatch",
				"expected 'ref'",
				"got 'i32'",
			} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("diagnostic = %v, want substring %q", err, want)
				}
			}
			if strings.Contains(err.Error(), "pointer C ABI boundary") {
				t.Fatalf("diagnostic = %v, should not be reported as a pointer C ABI boundary", err)
			}
			if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
				t.Fatalf("output object should not be written, stat error = %v", statErr)
			}
		})
	}
}

func TestLinuxFamilyBuildsExportedCIntFFIScalarObjects(t *testing.T) {
	src := `@export("ffi_c_int_identity_c")
func ffi_c_int_identity(n: c_int) -> c_int:
    return n
`

	for _, target := range []string{"linux-x86", "linux-x64", "linux-x32"} {
		t.Run(target, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "ffi_c_int_identity.t4")
			outPath := filepath.Join(tmp, "ffi_c_int_identity.tobj")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Emit: EmitLibrary, Jobs: 1}); err != nil {
				t.Fatalf("build %s c_int FFI object: %v", target, err)
			}
			obj, err := ReadObject(outPath)
			if err != nil {
				t.Fatalf("read object: %v", err)
			}
			if obj.Target != target {
				t.Fatalf("object target = %q, want %s", obj.Target, target)
			}
			if !abiSuiteObjectHasSymbolSignature(obj, "ffi_c_int_identity_c", 1, 1) {
				t.Fatalf("%s object missing exported ffi_c_int_identity_c(1)->1 symbol: %#v", target, obj.Symbols)
			}
		})
	}
}

func TestLinuxFamilyBuildsExportedCUIntFFIScalarObjects(t *testing.T) {
	src := `@export("ffi_c_uint_identity_c")
func ffi_c_uint_identity(n: c_uint) -> c_uint:
    return n
`

	for _, target := range []string{"linux-x86", "linux-x64", "linux-x32"} {
		t.Run(target, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "ffi_c_uint_identity.t4")
			outPath := filepath.Join(tmp, "ffi_c_uint_identity.tobj")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Emit: EmitLibrary, Jobs: 1}); err != nil {
				t.Fatalf("build %s c_uint FFI object: %v", target, err)
			}
			obj, err := ReadObject(outPath)
			if err != nil {
				t.Fatalf("read object: %v", err)
			}
			if obj.Target != target {
				t.Fatalf("object target = %q, want %s", obj.Target, target)
			}
			if !abiSuiteObjectHasSymbolSignature(obj, "ffi_c_uint_identity_c", 1, 1) {
				t.Fatalf("%s object missing exported ffi_c_uint_identity_c(1)->1 symbol: %#v", target, obj.Symbols)
			}
		})
	}
}

func TestLinuxX86AndX32BuildExportedILP32NativeLibcFFIScalarObjects(t *testing.T) {
	src := `@export("ffi_usize_identity_c")
func ffi_usize_identity(n: usize) -> usize:
    return n

@export("ffi_isize_identity_c")
func ffi_isize_identity(n: isize) -> isize:
    return n

@export("ffi_size_t_identity_c")
func ffi_size_t_identity(n: size_t) -> size_t:
    return n

@export("ffi_ssize_t_identity_c")
func ffi_ssize_t_identity(n: ssize_t) -> ssize_t:
    return n

@export("ffi_native_int_identity_c")
func ffi_native_int_identity(n: native_int) -> native_int:
    return n

@export("ffi_native_uint_identity_c")
func ffi_native_uint_identity(n: native_uint) -> native_uint:
    return n

@export("ffi_c_long_identity_c")
func ffi_c_long_identity(n: c_long) -> c_long:
    return n

@export("ffi_c_ulong_identity_c")
func ffi_c_ulong_identity(n: c_ulong) -> c_ulong:
    return n
`

	for _, target := range []string{"linux-x86", "linux-x32"} {
		t.Run(target, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "ffi_ilp32_native_libc_scalars.t4")
			outPath := filepath.Join(tmp, "ffi_ilp32_native_libc_scalars.tobj")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Emit: EmitLibrary, Jobs: 1}); err != nil {
				t.Fatalf("build %s ILP32 native/libc scalar FFI object: %v", target, err)
			}
			obj, err := ReadObject(outPath)
			if err != nil {
				t.Fatalf("read object: %v", err)
			}
			if obj.Target != target {
				t.Fatalf("object target = %q, want %s", obj.Target, target)
			}
			for _, symbol := range []string{
				"ffi_usize_identity_c",
				"ffi_isize_identity_c",
				"ffi_size_t_identity_c",
				"ffi_ssize_t_identity_c",
				"ffi_native_int_identity_c",
				"ffi_native_uint_identity_c",
				"ffi_c_long_identity_c",
				"ffi_c_ulong_identity_c",
			} {
				if !abiSuiteObjectHasSymbolSignature(obj, symbol, 1, 1) {
					t.Fatalf("%s object missing exported %s(1)->1 symbol: %#v", target, symbol, obj.Symbols)
				}
			}
		})
	}
}

func TestLinuxX86RejectsExportedPointerAndFunctionPointerFFIBoundaryUntilVerified(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "fnptr_param",
			src: `@export("ffi_callback_param_c")
func ffi_callback_param(cb: fn(Int) -> Int) -> Int:
    return 0
`,
			want: []string{
				"exported function 'ffi_callback_param'",
				"parameter 'cb'",
				"type 'fnptr'",
				"i386 pointer C ABI boundary is not verified on linux-x86",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "ffi_x86_pointer_"+tc.name+".t4")
			outPath := filepath.Join(tmp, "ffi_x86_pointer_"+tc.name+".tobj")
			if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			_, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x86", BuildOptions{Emit: EmitLibrary, Jobs: 1})
			if err == nil {
				t.Fatalf("expected x86 pointer/function-pointer FFI diagnostic")
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("diagnostic = %v, want substring %q", err, want)
				}
			}
			if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
				t.Fatalf("output object should not be written, stat error = %v", statErr)
			}
		})
	}
}

func TestNativeTargetsRejectUnsupportedTargetLayoutScalarsWithSourceNativeDiagnostic(t *testing.T) {
	tests := []struct {
		target   string
		name     string
		typeName string
		src      string
	}{
		{
			target:   "linux-x64",
			name:     "x64_ref_param",
			typeName: "ref",
			src: `@export("ffi_ref_param_c")
func ffi_ref_param(n: ref) -> Int:
    return 0
`,
		},
		{
			target:   "linux-x64",
			name:     "x64_usize_param",
			typeName: "usize",
			src: `@export("ffi_usize_param_c")
func ffi_usize_param(n: usize) -> Int:
    return 0
`,
		},
		{
			target:   "linux-x64",
			name:     "x64_native_int_return",
			typeName: "native_int",
			src: `@export("ffi_native_int_return_c")
func ffi_native_int_return() -> native_int:
    return 0
`,
		},
		{
			target:   "linux-x64",
			name:     "x64_rawptr_param",
			typeName: "rawptr",
			src: `@export("ffi_rawptr_param_c")
func ffi_rawptr_param(n: rawptr) -> Int:
    return 0
`,
		},
		{
			target:   "linux-x64",
			name:     "x64_nullable_ptr_param",
			typeName: "nullable_ptr",
			src: `@export("ffi_nullable_ptr_param_c")
func ffi_nullable_ptr_param(n: nullable_ptr) -> Int:
    return 0
`,
		},
		{
			target:   "linux-x86",
			name:     "x86_u32_param",
			typeName: "u32",
			src: `@export("ffi_u32_param_c")
func ffi_u32_param(n: u32) -> Int:
    return 0
`,
		},
		{
			target:   "linux-x86",
			name:     "x86_f64_return",
			typeName: "f64",
			src: `@export("ffi_f64_return_c")
func ffi_f64_return() -> f64:
    return 0
`,
		},
		{
			target:   "linux-x32",
			name:     "x32_u32_param",
			typeName: "u32",
			src: `@export("ffi_u32_param_c")
func ffi_u32_param(n: u32) -> Int:
    return 0
`,
		},
		{
			target:   "linux-x32",
			name:     "x32_f64_return",
			typeName: "f64",
			src: `@export("ffi_f64_return_c")
func ffi_f64_return() -> f64:
    return 0
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.target+"/"+tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "ffi_target_layout_"+tc.name+".t4")
			outPath := filepath.Join(tmp, "ffi_target_layout_"+tc.name+".tobj")
			if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			_, err := BuildFileWithStatsOpt(srcPath, outPath, tc.target, BuildOptions{Emit: EmitLibrary, Jobs: 1})
			if err == nil {
				t.Fatalf("expected source-level target-layout scalar diagnostic")
			}
			for _, want := range []string{
				"target-layout scalar type '" + tc.typeName + "'",
				"not supported in source-level Tetra yet",
				"native-int/codegen support",
			} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("diagnostic = %v, want substring %q", err, want)
				}
			}
			if strings.Contains(err.Error(), "pointer C ABI boundary") {
				t.Fatalf("diagnostic = %v, should not be reported as a pointer C ABI boundary", err)
			}
			if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
				t.Fatalf("output object should not be written, stat error = %v", statErr)
			}
		})
	}
}

func TestLinuxX32RejectsExportedFunctionPointerFFIBoundaryUntilVerified(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "param",
			src: `@export("ffi_callback_param_c")
func ffi_callback_param(cb: fn(Int) -> Int) -> Int:
    return 0
`,
			want: []string{
				"exported function 'ffi_callback_param'",
				"parameter 'cb'",
				"x32 pointer C ABI boundary is not verified on linux-x32",
			},
		},
		{
			name: "return",
			src: `func identity(x: Int) -> Int:
    return x

@export("ffi_callback_return_c")
func ffi_callback_return() -> fn(Int) -> Int:
    return identity
`,
			want: []string{
				"exported function 'ffi_callback_return'",
				"return type",
				"x32 pointer C ABI boundary is not verified on linux-x32",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "ffi_x32_fnptr_"+tc.name+".t4")
			outPath := filepath.Join(tmp, "ffi_x32_fnptr_"+tc.name+".tobj")
			if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			_, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x32", BuildOptions{Emit: EmitLibrary, Jobs: 1})
			if err == nil {
				t.Fatalf("expected x32 function-pointer FFI diagnostic")
			}
			diag := DiagnosticFromError(err)
			if diag.Code != DiagnosticCodeTargetRuntime || diag.Severity != "error" {
				t.Fatalf("diagnostic identity = %#v", diag)
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("diagnostic = %v, want substring %q", err, want)
				}
			}
			if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
				t.Fatalf("output object should not be written, stat error = %v", statErr)
			}
		})
	}
}
