package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestX32PointerFFIGateCoversPointerAndNativeLibcSpellings(t *testing.T) {
	for _, typeName := range []string{
		"ptr",
		"fnptr",
		"fn(Int) -> Int",
		"ref",
		"nullable_ptr",
		"rawptr",
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
			if !targetExportedFFIRequiresX32PointerBoundaryGate("linux-x32", typeName) {
				t.Fatalf("linux-x32 FFI gate did not cover %q", typeName)
			}
			if targetExportedFFIRequiresX32PointerBoundaryGate("linux-x64", typeName) {
				t.Fatalf("linux-x64 FFI gate unexpectedly covered %q", typeName)
			}
		})
	}

	for _, typeName := range []string{"i32", "u32", "c_int", "c_uint", "bool"} {
		t.Run(typeName, func(t *testing.T) {
			if targetExportedFFIRequiresX32PointerBoundaryGate("linux-x32", typeName) {
				t.Fatalf("linux-x32 FFI gate should not cover scalar wrapper type %q", typeName)
			}
		})
	}
}

func TestNativeTargetsRejectExportedAggregateFFIParameters(t *testing.T) {
	src := `struct Pair:
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
	src := `struct Pair:
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

func TestLinuxX32RejectsExportedPointerFFIBoundaryUntilVerified(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "param",
			src: `@export("ffi_ptr_param_c")
func ffi_ptr_param(p: ptr) -> Int:
    return 0
`,
			want: []string{
				"exported function 'ffi_ptr_param'",
				"parameter 'p'",
				"type 'ptr'",
				"x32 pointer C ABI boundary is not verified on linux-x32",
			},
		},
		{
			name: "return",
			src: `@export("ffi_ptr_return_c")
func ffi_ptr_return() -> ptr:
    return 0
`,
			want: []string{
				"exported function 'ffi_ptr_return'",
				"return type 'ptr'",
				"x32 pointer C ABI boundary is not verified on linux-x32",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "ffi_x32_ptr_"+tc.name+".t4")
			outPath := filepath.Join(tmp, "ffi_x32_ptr_"+tc.name+".tobj")
			if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			_, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x32", BuildOptions{Emit: EmitLibrary, Jobs: 1})
			if err == nil {
				t.Fatalf("expected x32 pointer FFI diagnostic")
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

func TestLinuxX86RejectsExportedPointerNativeAndFunctionPointerFFIBoundaryUntilVerified(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "ptr_param",
			src: `@export("ffi_ptr_param_c")
func ffi_ptr_param(p: ptr) -> Int:
    return 0
`,
			want: []string{
				"exported function 'ffi_ptr_param'",
				"parameter 'p'",
				"type 'ptr'",
				"i386 pointer C ABI boundary is not verified on linux-x86",
			},
		},
		{
			name: "native_return",
			src: `@export("ffi_native_int_return_c")
func ffi_native_int_return() -> native_int:
    return 0
`,
			want: []string{
				"exported function 'ffi_native_int_return'",
				"return type 'native_int'",
				"i386 pointer C ABI boundary is not verified on linux-x86",
			},
		},
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
				t.Fatalf("expected x86 pointer/native/function-pointer FFI diagnostic")
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
