package abisuite

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ctarget "tetra_language/compiler/target"
)

type FFICheckDeps struct {
	BuildLibrary func(srcPath string, outPath string, target string) error
	ReadObject   func(path string) (ObjectSummary, error)
}

type ObjectSummary struct {
	Target  string
	Data    []byte
	Symbols []ObjectSymbolSummary
	Relocs  []ObjectRelocSummary
}

type ObjectSymbolSummary struct {
	Name         string
	HasSignature bool
	ParamSlots   int
	ReturnSlots  int
}

type ObjectRelocKind int

const (
	ObjectRelocIATDisp32  ObjectRelocKind = 2
	ObjectRelocDataDisp32 ObjectRelocKind = 3
)

type ObjectRelocSummary struct {
	Kind ObjectRelocKind
	Name string
}

func CheckX86RefFFINullReturnDiagnostics(deps FFICheckDeps) error {
	return CheckRefFFINullReturnDiagnostics("linux-x86", "x86", deps)
}

func CheckX32RefFFINullReturnDiagnostics(deps FFICheckDeps) error {
	return CheckRefFFINullReturnDiagnostics("linux-x32", "x32", deps)
}

func CheckX86FunctionPointerFFIDiagnostics(deps FFICheckDeps) error {
	return CheckFunctionPointerFFIDiagnostics("linux-x86", "i386", "x86", deps)
}

func CheckX32FunctionPointerFFIDiagnostics(deps FFICheckDeps) error {
	return CheckFunctionPointerFFIDiagnostics("linux-x32", "x32", "x32", deps)
}

func CheckPointerFFIObjectSmoke(tgt ctarget.Target, deps FFICheckDeps) error {
	if tgt.Triple != "linux-x86" && tgt.Triple != "linux-x32" {
		return fmt.Errorf("pointer FFI object smoke requires linux-x86 or linux-x32 target, got %s", tgt.Triple)
	}
	stem := strings.TrimPrefix(tgt.Triple, "linux-")
	tmpDir, err := os.MkdirTemp("", "tetra-"+stem+"-pointer-ffi-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, stem+"_pointer_ffi.tetra")
	outPath := filepath.Join(tmpDir, stem+"_pointer_ffi.tobj")
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
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildLibrary(deps, srcPath, outPath, tgt.Triple); err != nil {
		return err
	}
	obj, err := readObject(deps, outPath)
	if err != nil {
		return err
	}
	if obj.Target != tgt.Triple {
		return fmt.Errorf("%s pointer FFI object target = %q, want %s", stem, obj.Target, tgt.Triple)
	}
	if !ObjectHasSymbolSignature(obj, "ffi_ptr_identity_c", 1, 1) {
		return fmt.Errorf("%s pointer FFI object missing exported ffi_ptr_identity_c(1)->1 symbol: %#v", stem, obj.Symbols)
	}
	if !ObjectHasSymbolSignature(obj, "ffi_rawptr_identity_c", 1, 1) {
		return fmt.Errorf("%s pointer FFI object missing exported ffi_rawptr_identity_c(1)->1 symbol: %#v", stem, obj.Symbols)
	}
	if !ObjectHasSymbolSignature(obj, "ffi_nullable_ptr_identity_c", 1, 1) {
		return fmt.Errorf("%s pointer FFI object missing exported ffi_nullable_ptr_identity_c(1)->1 symbol: %#v", stem, obj.Symbols)
	}
	if !ObjectHasSymbolSignature(obj, "ffi_nullable_ptr_null_c", 0, 1) {
		return fmt.Errorf("%s pointer FFI object missing exported ffi_nullable_ptr_null_c(0)->1 symbol: %#v", stem, obj.Symbols)
	}
	if !ObjectHasSymbolSignature(obj, "ffi_ref_identity_c", 1, 1) {
		return fmt.Errorf("%s pointer FFI object missing exported ffi_ref_identity_c(1)->1 symbol: %#v", stem, obj.Symbols)
	}
	return nil
}

func CheckCIntFFIObjectSmoke(tgt ctarget.Target, deps FFICheckDeps) error {
	if tgt.Triple != "linux-x86" && tgt.Triple != "linux-x64" && tgt.Triple != "linux-x32" {
		return fmt.Errorf("c_int FFI object smoke requires linux-x86/linux-x64/linux-x32 target, got %s", tgt.Triple)
	}
	stem := strings.TrimPrefix(tgt.Triple, "linux-")
	tmpDir, err := os.MkdirTemp("", "tetra-"+stem+"-c-int-ffi-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, stem+"_c_int_ffi.tetra")
	outPath := filepath.Join(tmpDir, stem+"_c_int_ffi.tobj")
	src := `@export("ffi_c_int_identity_c")
func ffi_c_int_identity(n: c_int) -> c_int:
    return n
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildLibrary(deps, srcPath, outPath, tgt.Triple); err != nil {
		return err
	}
	obj, err := readObject(deps, outPath)
	if err != nil {
		return err
	}
	if obj.Target != tgt.Triple {
		return fmt.Errorf("%s c_int FFI object target = %q, want %s", stem, obj.Target, tgt.Triple)
	}
	if !ObjectHasSymbolSignature(obj, "ffi_c_int_identity_c", 1, 1) {
		return fmt.Errorf("%s c_int FFI object missing exported ffi_c_int_identity_c(1)->1 symbol: %#v", stem, obj.Symbols)
	}
	return nil
}

func CheckCUIntFFIObjectSmoke(tgt ctarget.Target, deps FFICheckDeps) error {
	if tgt.Triple != "linux-x86" && tgt.Triple != "linux-x64" && tgt.Triple != "linux-x32" {
		return fmt.Errorf("c_uint FFI object smoke requires linux-x86/linux-x64/linux-x32 target, got %s", tgt.Triple)
	}
	stem := strings.TrimPrefix(tgt.Triple, "linux-")
	tmpDir, err := os.MkdirTemp("", "tetra-"+stem+"-c-uint-ffi-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, stem+"_c_uint_ffi.tetra")
	outPath := filepath.Join(tmpDir, stem+"_c_uint_ffi.tobj")
	src := `@export("ffi_c_uint_identity_c")
func ffi_c_uint_identity(n: c_uint) -> c_uint:
    return n
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildLibrary(deps, srcPath, outPath, tgt.Triple); err != nil {
		return err
	}
	obj, err := readObject(deps, outPath)
	if err != nil {
		return err
	}
	if obj.Target != tgt.Triple {
		return fmt.Errorf("%s c_uint FFI object target = %q, want %s", stem, obj.Target, tgt.Triple)
	}
	if !ObjectHasSymbolSignature(obj, "ffi_c_uint_identity_c", 1, 1) {
		return fmt.Errorf("%s c_uint FFI object missing exported ffi_c_uint_identity_c(1)->1 symbol: %#v", stem, obj.Symbols)
	}
	return nil
}

func CheckILP32NativeLibcFFIObjectSmoke(tgt ctarget.Target, deps FFICheckDeps) error {
	if tgt.Triple != "linux-x86" && tgt.Triple != "linux-x32" {
		return fmt.Errorf("ILP32 native/libc FFI object smoke requires linux-x86 or linux-x32 target, got %s", tgt.Triple)
	}
	stem := strings.TrimPrefix(tgt.Triple, "linux-")
	tmpDir, err := os.MkdirTemp("", "tetra-"+stem+"-ilp32-native-libc-ffi-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, stem+"_ilp32_native_libc_ffi.tetra")
	outPath := filepath.Join(tmpDir, stem+"_ilp32_native_libc_ffi.tobj")
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
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildLibrary(deps, srcPath, outPath, tgt.Triple); err != nil {
		return err
	}
	obj, err := readObject(deps, outPath)
	if err != nil {
		return err
	}
	if obj.Target != tgt.Triple {
		return fmt.Errorf("%s ILP32 native/libc FFI object target = %q, want %s", stem, obj.Target, tgt.Triple)
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
		if !ObjectHasSymbolSignature(obj, symbol, 1, 1) {
			return fmt.Errorf("%s ILP32 native/libc FFI object missing exported %s(1)->1 symbol: %#v", stem, symbol, obj.Symbols)
		}
	}
	return nil
}

func CheckRefFFINullReturnDiagnostics(targetName, stem string, deps FFICheckDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+stem+"-ffi-ref-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, stem+"_ffi_ref_null_return.tetra")
	outPath := filepath.Join(tmpDir, stem+"_ffi_ref_null_return.tobj")
	src := "@export(\"ffi_ref_null_c\")\nfunc ffi_ref_null() -> ref:\n    return 0\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	err = buildLibrary(deps, srcPath, outPath, targetName)
	if err == nil {
		return fmt.Errorf("%s ref null-return FFI export was accepted", stem)
	}
	if want := "type mismatch: expected 'ref', got 'i32'"; !strings.Contains(err.Error(), want) {
		return fmt.Errorf("%s ref null-return FFI diagnostic = %q, want %q", stem, err.Error(), want)
	}
	if strings.Contains(err.Error(), "pointer C ABI boundary") {
		return fmt.Errorf("%s ref null-return FFI diagnostic = %q, should not report pointer C ABI boundary", stem, err.Error())
	}
	if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
		return fmt.Errorf("%s ref null-return FFI wrote object %s (stat err=%v)", stem, outPath, statErr)
	}
	return nil
}

func CheckFunctionPointerFFIDiagnostics(targetName, boundaryName, stem string, deps FFICheckDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+stem+"-ffi-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	cases := []struct {
		name         string
		src          string
		want         string
		wantBoundary bool
	}{
		{
			name:         "fnptr_param",
			src:          "@export(\"ffi_fnptr_param_c\")\nfunc ffi_fnptr_param(cb: fn(Int) -> Int) -> Int:\n    return 0\n",
			want:         "exported function 'ffi_fnptr_param' parameter 'cb' type 'fnptr' requires the " + boundaryName + " pointer C ABI boundary",
			wantBoundary: true,
		},
		{
			name:         "fnptr_return",
			src:          "func identity(x: Int) -> Int:\n    return x\n\n@export(\"ffi_fnptr_return_c\")\nfunc ffi_fnptr_return() -> fn(Int) -> Int:\n    return identity\n",
			want:         "exported function 'ffi_fnptr_return' return type 'fnptr' requires the " + boundaryName + " pointer C ABI boundary",
			wantBoundary: true,
		},
	}
	for _, tc := range cases {
		srcPath := filepath.Join(tmpDir, stem+"_ffi_"+tc.name+".tetra")
		outPath := filepath.Join(tmpDir, stem+"_ffi_"+tc.name+".tobj")
		if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
			return err
		}
		err := buildLibrary(deps, srcPath, outPath, targetName)
		if err == nil {
			return fmt.Errorf("%s %s pointer FFI export was accepted", tc.name, stem)
		}
		if !strings.Contains(err.Error(), tc.want) {
			return fmt.Errorf("%s %s pointer FFI diagnostic = %q, want %q", tc.name, stem, err.Error(), tc.want)
		}
		if tc.wantBoundary {
			if !strings.Contains(err.Error(), boundaryName+" pointer C ABI boundary is not verified on "+targetName) {
				return fmt.Errorf("%s %s pointer FFI diagnostic = %q, want %s boundary", tc.name, stem, err.Error(), boundaryName)
			}
		} else if strings.Contains(err.Error(), "pointer C ABI boundary") {
			return fmt.Errorf("%s %s pointer FFI diagnostic = %q, should not report pointer C ABI boundary", tc.name, stem, err.Error())
		}
		if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
			return fmt.Errorf("%s %s pointer FFI wrote object %s (stat err=%v)", tc.name, stem, outPath, statErr)
		}
	}
	return nil
}

func ObjectHasSymbolSignature(obj ObjectSummary, name string, params, returns int) bool {
	for _, sym := range obj.Symbols {
		if sym.Name == name && sym.HasSignature && sym.ParamSlots == params && sym.ReturnSlots == returns {
			return true
		}
	}
	return false
}

func ObjectHasRelocKind(obj ObjectSummary, kind ObjectRelocKind) bool {
	for _, reloc := range obj.Relocs {
		if reloc.Kind == kind {
			return true
		}
	}
	return false
}

func ObjectHasReloc(obj ObjectSummary, kind ObjectRelocKind, name string) bool {
	for _, reloc := range obj.Relocs {
		if reloc.Kind == kind && reloc.Name == name {
			return true
		}
	}
	return false
}

func buildLibrary(deps FFICheckDeps, srcPath string, outPath string, target string) error {
	if deps.BuildLibrary == nil {
		return fmt.Errorf("missing FFI build library callback")
	}
	return deps.BuildLibrary(srcPath, outPath, target)
}

func readObject(deps FFICheckDeps, path string) (ObjectSummary, error) {
	if deps.ReadObject == nil {
		return ObjectSummary{}, fmt.Errorf("missing FFI read object callback")
	}
	return deps.ReadObject(path)
}
