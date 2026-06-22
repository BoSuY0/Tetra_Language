package abisuite

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"tetra_language/compiler/internal/backend/linux_x32"
	"tetra_language/compiler/internal/backend/linux_x86"
	wasm32wasi "tetra_language/compiler/internal/backend/wasm32_wasi"
	wasm32web "tetra_language/compiler/internal/backend/wasm32_web"
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/backend/x86abi"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
	ctarget "tetra_language/compiler/target"
	"time"
)

// ---- checks.go ----

type Check struct {
	Name  string
	Error string
}

type Case struct {
	Name string
	Run  func() error
}

func RunChecks(cases []Case) []Check {
	out := make([]Check, 0, len(cases))
	for _, tc := range cases {
		check := Check{Name: tc.Name}
		if err := tc.Run(); err != nil {
			check.Error = err.Error()
		}
		out = append(out, check)
	}
	return out
}

func UnsupportedTargetError(target string) error {
	return fmt.Errorf("ABI suite for target %s is not implemented", target)
}

// ---- ctx_switch.go ----

type CtxSwitchObject struct {
	Target string
	Code   []byte
}

type CtxSwitchDeps struct {
	BuildX86Object func(funcs []ir.IRFunc) (CtxSwitchObject, error)
	BuildX32Object func(funcs []ir.IRFunc) (CtxSwitchObject, error)
}

func CheckX86CtxSwitchObjectSmoke(deps CtxSwitchDeps) error {
	obj, err := buildX86CtxSwitchObject(deps, ctxSwitchSmokeIR("__tetra_x86_ctx_switch_smoke"))
	if err != nil {
		return err
	}
	if !bytes.Contains(obj.Code, ctxSwitchI386Stub()) {
		return fmt.Errorf("x86 ctx_switch object missing i386 context stub")
	}
	if !bytes.Contains(obj.Code, ctxSwitchZeroStatusContinuation()) {
		return fmt.Errorf("x86 ctx_switch object missing zero status continuation")
	}
	return nil
}

func CheckX32CtxSwitchObjectSmoke(deps CtxSwitchDeps) error {
	obj, err := buildX32CtxSwitchObject(deps, ctxSwitchSmokeIR("__tetra_x32_ctx_switch_smoke"))
	if err != nil {
		return err
	}
	if obj.Target != "linux-x32" {
		return fmt.Errorf("x32 ctx_switch object target = %q, want linux-x32", obj.Target)
	}
	if !bytes.Contains(obj.Code, ctxSwitchX32SysVStub()) {
		return fmt.Errorf("x32 ctx_switch object missing SysV x86_64 context stub")
	}
	if bytes.Contains(obj.Code, ctxSwitchX32ShadowSpaceAdjustment()) {
		return fmt.Errorf(
			"x32 ctx_switch object unexpectedly emitted Win64 shadow-space adjustment",
		)
	}
	if !bytes.Contains(obj.Code, ctxSwitchZeroStatusContinuation()) {
		return fmt.Errorf("x32 ctx_switch object missing zero status continuation")
	}
	return nil
}

func buildX86CtxSwitchObject(deps CtxSwitchDeps, funcs []ir.IRFunc) (CtxSwitchObject, error) {
	if deps.BuildX86Object != nil {
		return deps.BuildX86Object(funcs)
	}
	obj, err := linux_x86.CodegenObjectLinuxX86(funcs)
	if err != nil {
		return CtxSwitchObject{}, err
	}
	return CtxSwitchObject{Target: obj.Target, Code: obj.Code}, nil
}

func buildX32CtxSwitchObject(deps CtxSwitchDeps, funcs []ir.IRFunc) (CtxSwitchObject, error) {
	if deps.BuildX32Object != nil {
		return deps.BuildX32Object(funcs)
	}
	obj, err := linux_x32.CodegenObjectLinuxX32(funcs)
	if err != nil {
		return CtxSwitchObject{}, err
	}
	return CtxSwitchObject{Target: obj.Target, Code: obj.Code}, nil
}

func ctxSwitchSmokeIR(name string) []ir.IRFunc {
	return []ir.IRFunc{{
		Name:        name,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRCtxSwitch},
			{Kind: ir.IRReturn},
		},
	}}
}

func ctxSwitchI386Stub() []byte {
	return []byte{0x53, 0x55, 0x56, 0x57, 0x89, 0x20, 0x8B, 0x21, 0x5F, 0x5E, 0x5D, 0x5B, 0xC3}
}

func ctxSwitchX32SysVStub() []byte {
	e := &x64.Emitter{}
	e.PushRbx()
	e.PushRbp()
	e.PushR12()
	e.PushR13()
	e.PushR14()
	e.PushR15()
	e.MovMem64RdiDispRsp(0)
	e.MovRdiRsi()
	e.MovRspFromRdiDisp(0)
	e.PopR15()
	e.PopR14()
	e.PopR13()
	e.PopR12()
	e.PopRbp()
	e.PopRbx()
	e.Ret()
	return e.Buf
}

func ctxSwitchX32ShadowSpaceAdjustment() []byte {
	e := &x64.Emitter{}
	e.SubRspImm32(32)
	return e.Buf
}

func ctxSwitchZeroStatusContinuation() []byte {
	return []byte{0x31, 0xC0, 0x50}
}

// ---- ffi.go ----

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
		return fmt.Errorf(
			"pointer FFI object smoke requires linux-x86 or linux-x32 target, got %s",
			tgt.Triple,
		)
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
		return fmt.Errorf(
			"%s pointer FFI object target = %q, want %s",
			stem,
			obj.Target,
			tgt.Triple,
		)
	}
	if !ObjectHasSymbolSignature(obj, "ffi_ptr_identity_c", 1, 1) {
		return fmt.Errorf(
			"%s pointer FFI object missing exported ffi_ptr_identity_c(1)->1 symbol: %#v",
			stem,
			obj.Symbols,
		)
	}
	if !ObjectHasSymbolSignature(obj, "ffi_rawptr_identity_c", 1, 1) {
		return fmt.Errorf(
			"%s pointer FFI object missing exported ffi_rawptr_identity_c(1)->1 symbol: %#v",
			stem,
			obj.Symbols,
		)
	}
	if !ObjectHasSymbolSignature(obj, "ffi_nullable_ptr_identity_c", 1, 1) {
		return fmt.Errorf(
			"%s pointer FFI object missing exported ffi_nullable_ptr_identity_c(1)->1 symbol: %#v",
			stem,
			obj.Symbols,
		)
	}
	if !ObjectHasSymbolSignature(obj, "ffi_nullable_ptr_null_c", 0, 1) {
		return fmt.Errorf(
			"%s pointer FFI object missing exported ffi_nullable_ptr_null_c(0)->1 symbol: %#v",
			stem,
			obj.Symbols,
		)
	}
	if !ObjectHasSymbolSignature(obj, "ffi_ref_identity_c", 1, 1) {
		return fmt.Errorf(
			"%s pointer FFI object missing exported ffi_ref_identity_c(1)->1 symbol: %#v",
			stem,
			obj.Symbols,
		)
	}
	return nil
}

func CheckCIntFFIObjectSmoke(tgt ctarget.Target, deps FFICheckDeps) error {
	if tgt.Triple != "linux-x86" && tgt.Triple != "linux-x64" && tgt.Triple != "linux-x32" {
		return fmt.Errorf(
			"c_int FFI object smoke requires linux-x86/linux-x64/linux-x32 target, got %s",
			tgt.Triple,
		)
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
		return fmt.Errorf(
			"%s c_int FFI object missing exported ffi_c_int_identity_c(1)->1 symbol: %#v",
			stem,
			obj.Symbols,
		)
	}
	return nil
}

func CheckCUIntFFIObjectSmoke(tgt ctarget.Target, deps FFICheckDeps) error {
	if tgt.Triple != "linux-x86" && tgt.Triple != "linux-x64" && tgt.Triple != "linux-x32" {
		return fmt.Errorf(
			"c_uint FFI object smoke requires linux-x86/linux-x64/linux-x32 target, got %s",
			tgt.Triple,
		)
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
		return fmt.Errorf(
			"%s c_uint FFI object missing exported ffi_c_uint_identity_c(1)->1 symbol: %#v",
			stem,
			obj.Symbols,
		)
	}
	return nil
}

func CheckILP32NativeLibcFFIObjectSmoke(tgt ctarget.Target, deps FFICheckDeps) error {
	if tgt.Triple != "linux-x86" && tgt.Triple != "linux-x32" {
		return fmt.Errorf(
			"ILP32 native/libc FFI object smoke requires linux-x86 or linux-x32 target, got %s",
			tgt.Triple,
		)
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
		return fmt.Errorf(
			"%s ILP32 native/libc FFI object target = %q, want %s",
			stem,
			obj.Target,
			tgt.Triple,
		)
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
			return fmt.Errorf(
				"%s ILP32 native/libc FFI object missing exported %s(1)->1 symbol: %#v",
				stem,
				symbol,
				obj.Symbols,
			)
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
		return fmt.Errorf(
			"%s ref null-return FFI diagnostic = %q, want %q",
			stem,
			err.Error(),
			want,
		)
	}
	if strings.Contains(err.Error(), "pointer C ABI boundary") {
		return fmt.Errorf(
			"%s ref null-return FFI diagnostic = %q, should not report pointer C ABI boundary",
			stem,
			err.Error(),
		)
	}
	if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
		return fmt.Errorf(
			"%s ref null-return FFI wrote object %s (stat err=%v)",
			stem,
			outPath,
			statErr,
		)
	}
	return nil
}

func CheckFunctionPointerFFIDiagnostics(
	targetName, boundaryName, stem string,
	deps FFICheckDeps,
) error {
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
			name: "fnptr_param",
			src:  "@export(\"ffi_fnptr_param_c\")\nfunc ffi_fnptr_param(cb: fn(Int) -> Int) -> Int:\n    return 0\n",
			want: ("exported function 'ffi_fnptr_param' parameter 'cb' type 'fnptr' " +
				"requires the ") + boundaryName + " pointer C ABI boundary",
			wantBoundary: true,
		},
		{
			name: "fnptr_return",
			src:  "func identity(x: Int) -> Int:\n    return x\n\n@export(\"ffi_fnptr_return_c\")\nfunc ffi_fnptr_return() -> fn(Int) -> Int:\n    return identity\n",
			want: ("exported function 'ffi_fnptr_return' return type 'fnptr' " +
				"requires the ") + boundaryName + " pointer C ABI boundary",
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
			return fmt.Errorf(
				"%s %s pointer FFI diagnostic = %q, want %q",
				tc.name,
				stem,
				err.Error(),
				tc.want,
			)
		}
		if tc.wantBoundary {
			if !strings.Contains(
				err.Error(),
				boundaryName+" pointer C ABI boundary is not verified on "+targetName,
			) {
				return fmt.Errorf(
					"%s %s pointer FFI diagnostic = %q, want %s boundary",
					tc.name,
					stem,
					err.Error(),
					boundaryName,
				)
			}
		} else if strings.Contains(err.Error(), "pointer C ABI boundary") {
			return fmt.Errorf(
				"%s %s pointer FFI diagnostic = %q, should not report pointer C ABI boundary",
				tc.name,
				stem,
				err.Error(),
			)
		}
		if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
			return fmt.Errorf(
				"%s %s pointer FFI wrote object %s (stat err=%v)",
				tc.name,
				stem,
				outPath,
				statErr,
			)
		}
	}
	return nil
}

func ObjectHasSymbolSignature(obj ObjectSummary, name string, params, returns int) bool {
	for _, sym := range obj.Symbols {
		if sym.Name == name && sym.HasSignature && sym.ParamSlots == params &&
			sym.ReturnSlots == returns {
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

// ---- native_classifiers.go ----

func CheckX86I386Classifier(tgt ctarget.Target) error {
	classifier, err := x86abi.NewClassifier(tgt)
	if err != nil {
		return err
	}
	if classifier.Name() != "i386-sysv" || classifier.StackCleanup() != x86abi.StackCleanupCaller {
		return fmt.Errorf(
			"x86 classifier identity = %s cleanup=%s, want i386-sysv caller cleanup",
			classifier.Name(),
			classifier.StackCleanup(),
		)
	}
	plan, err := classifier.ClassifySignature(x86abi.ABISignature{
		Params: []x86abi.ABIParam{
			{Name: "p", Type: "ptr"},
			{Name: "wide", Type: "u64"},
			{Name: "f", Type: "f32"},
		},
		Return: &x86abi.ABIParam{Name: "ret", Type: "ptr"},
	})
	if err != nil {
		return err
	}
	if plan.PointerWidthBits != 32 || plan.RegisterWidthBits != 32 ||
		plan.StackCleanup != x86abi.StackCleanupCaller {
		return fmt.Errorf(
			"x86 ABI plan identity = %#v, want 32-bit pointer/register caller-cleaned stack",
			plan,
		)
	}
	if err := expectX86StackArg(plan.Params[0], "p", x86abi.ABIClassInteger, 0, 4, 4, 4); err != nil {
		return err
	}
	if err := expectX86StackArg(
		plan.Params[1],
		"wide",
		x86abi.ABIClassInteger,
		4,
		8,
		8,
		4,
	); err != nil {
		return err
	}
	if err := expectX86StackArg(plan.Params[2], "f", x86abi.ABIClassX87, 12, 4, 4, 4); err != nil {
		return err
	}
	if got := plan.Return; got.Register != "eax" || got.Class != x86abi.ABIClassInteger ||
		got.SizeBytes != 4 ||
		got.Extension != x86abi.ABIExtendNone {
		return fmt.Errorf(
			"x86 ptr return = %#v, want eax pointer return without widening extension",
			got,
		)
	}
	scalarReturns, err := classifier.ClassifySignature(
		x86abi.ABISignature{Return: &x86abi.ABIParam{Name: "ret", Type: "i64"}},
	)
	if err != nil {
		return err
	}
	if got := scalarReturns.Return; got.Register != "edx:eax" ||
		!sameStrings(got.Registers, []string{"eax", "edx"}) ||
		got.SizeBytes != 8 ||
		got.Class != x86abi.ABIClassInteger {
		return fmt.Errorf("x86 i64 return = %#v, want edx:eax", got)
	}
	floatReturns, err := classifier.ClassifySignature(
		x86abi.ABISignature{Return: &x86abi.ABIParam{Name: "ret", Type: "f64"}},
	)
	if err != nil {
		return err
	}
	if got := floatReturns.Return; got.Register != "st0" || got.Class != x86abi.ABIClassX87 ||
		got.RegisterWidthBits != 80 {
		return fmt.Errorf("x86 f64 return = %#v, want x87 st0", got)
	}
	return nil
}

func CheckX86VarargsAndSRet(tgt ctarget.Target) error {
	classifier, err := x86abi.NewClassifier(tgt)
	if err != nil {
		return err
	}
	fields := []ctarget.LayoutField{
		{Name: "tag", Type: "u8"},
		{Name: "raw", Type: "ptr"},
	}
	aggregate, err := classifier.ClassifySignature(x86abi.ABISignature{
		Params: []x86abi.ABIParam{{Name: "value", Type: "Pair", Fields: fields}},
		Return: &x86abi.ABIParam{Name: "ret", Type: "Pair", Fields: fields},
	})
	if err != nil {
		return err
	}
	if got := aggregate.Params[0]; got.Class != x86abi.ABIClassMemory ||
		got.StackOffsetBytes != 4 ||
		got.StackSlotBytes != 8 ||
		got.SizeBytes != 8 ||
		got.AlignBytes != 4 {
		return fmt.Errorf("x86 struct param = %#v, want stack copy after hidden sret pointer", got)
	}
	if got := aggregate.Return; got.Class != x86abi.ABIClassMemory || !got.Indirect ||
		got.Register != "sret@stack+0" ||
		got.StackOffsetBytes != 0 ||
		got.StackSlotBytes != 4 ||
		got.SizeBytes != 8 {
		return fmt.Errorf(
			"x86 struct return = %#v, want hidden sret pointer at first stack argument",
			got,
		)
	}
	variadic, err := classifier.ClassifySignature(x86abi.ABISignature{
		Variadic:        true,
		FixedParamCount: 1,
		Params: []x86abi.ABIParam{
			{Name: "fmt", Type: "ptr"},
			{Name: "first", Type: "f64"},
			{Name: "count", Type: "i32"},
		},
	})
	if err != nil {
		return err
	}
	if !variadic.Variadic || variadic.FixedParamCount != 1 || variadic.VarargStartIndex != 1 ||
		variadic.StackCleanup != x86abi.StackCleanupCaller {
		return fmt.Errorf(
			"x86 variadic metadata = %#v, want caller-cleaned stack varargs",
			variadic,
		)
	}
	if variadic.RegisterVarargs || variadic.VarargRegisterSaveBytes != 0 {
		return fmt.Errorf("x86 varargs unexpectedly require register save area: %#v", variadic)
	}
	if err := expectX86StackArg(
		variadic.Params[1],
		"first",
		x86abi.ABIClassX87,
		4,
		8,
		8,
		4,
	); err != nil {
		return err
	}
	if _, err := classifier.ClassifySignature(x86abi.ABISignature{
		Variadic:        true,
		FixedParamCount: 3,
		Params:          []x86abi.ABIParam{{Name: "fmt", Type: "ptr"}, {Name: "value", Type: "i32"}},
	}); err == nil || !strings.Contains(err.Error(), "invalid variadic fixed parameter count") {
		return fmt.Errorf("x86 invalid variadic fixed prefix diagnostic = %v", err)
	}
	return nil
}

func CheckX64Classifier(tgt ctarget.Target) error {
	classifier, err := x64abi.NewClassifier(tgt)
	if err != nil {
		return err
	}
	if !classifier.UsesX64Registers() {
		return fmt.Errorf("x64 classifier %s does not report x64 registers", classifier.Name())
	}
	plan, err := classifier.ClassifySignature(x64abi.ABISignature{
		Params: []x64abi.ABIParam{
			{Name: "p", Type: "ptr"},
			{Name: "n", Type: "usize"},
			{Name: "wide", Type: "u64"},
			{Name: "f", Type: "f64"},
		},
		Return: &x64abi.ABIParam{Name: "ret", Type: "ptr"},
	})
	if err != nil {
		return err
	}
	if plan.PointerWidthBits != 64 || plan.RegisterWidthBits != 64 {
		return fmt.Errorf("x64 ABI plan identity = %#v, want 64-bit pointer/registers", plan)
	}
	switch tgt.ABI {
	case ctarget.ABISysV:
		if classifier.Name() != "sysv" {
			return fmt.Errorf("x64 SysV classifier name = %s, want sysv", classifier.Name())
		}
		if err := expectX64Arg(
			plan.Params[0],
			"p",
			x64abi.ABIClassInteger,
			"rdi",
			8,
			8,
			64,
			x64abi.ABIExtendNone,
		); err != nil {
			return err
		}
		if err := expectX64Arg(
			plan.Params[1],
			"n",
			x64abi.ABIClassInteger,
			"rsi",
			8,
			8,
			64,
			x64abi.ABIExtendNone,
		); err != nil {
			return err
		}
		if err := expectX64Arg(
			plan.Params[2],
			"wide",
			x64abi.ABIClassInteger,
			"rdx",
			8,
			8,
			64,
			x64abi.ABIExtendNone,
		); err != nil {
			return err
		}
		if err := expectX64Arg(
			plan.Params[3],
			"f",
			x64abi.ABIClassSSE,
			"xmm0",
			8,
			8,
			128,
			x64abi.ABIExtendNone,
		); err != nil {
			return err
		}
	case ctarget.ABIWin64:
		if classifier.Name() != "win64" {
			return fmt.Errorf("x64 Win64 classifier name = %s, want win64", classifier.Name())
		}
		if err := expectX64Arg(
			plan.Params[0],
			"p",
			x64abi.ABIClassInteger,
			"rcx",
			8,
			8,
			64,
			x64abi.ABIExtendNone,
		); err != nil {
			return err
		}
		if err := expectX64Arg(
			plan.Params[1],
			"n",
			x64abi.ABIClassInteger,
			"rdx",
			8,
			8,
			64,
			x64abi.ABIExtendNone,
		); err != nil {
			return err
		}
		if err := expectX64Arg(
			plan.Params[2],
			"wide",
			x64abi.ABIClassInteger,
			"r8",
			8,
			8,
			64,
			x64abi.ABIExtendNone,
		); err != nil {
			return err
		}
		if err := expectX64Arg(
			plan.Params[3],
			"f",
			x64abi.ABIClassSSE,
			"xmm3",
			8,
			8,
			128,
			x64abi.ABIExtendNone,
		); err != nil {
			return err
		}
	default:
		return fmt.Errorf("x64 unsupported classifier ABI %s", tgt.ABI)
	}
	if got := plan.Return; got.Register != "rax" || got.Class != x64abi.ABIClassInteger ||
		got.SizeBytes != 8 {
		return fmt.Errorf("x64 ptr return = %#v, want rax", got)
	}
	return nil
}

func CheckX64VarargsAndAggregates(tgt ctarget.Target) error {
	classifier, err := x64abi.NewClassifier(tgt)
	if err != nil {
		return err
	}
	switch tgt.ABI {
	case ctarget.ABISysV:
		variadic, err := classifier.ClassifySignature(x64abi.ABISignature{
			Variadic:        true,
			FixedParamCount: 1,
			Params: []x64abi.ABIParam{
				{Name: "fmt", Type: "ptr"},
				{Name: "first", Type: "f64"},
				{Name: "count", Type: "i32"},
				{Name: "second", Type: "f32"},
			},
		})
		if err != nil {
			return err
		}
		if !variadic.Variadic || !variadic.SysVRequiresAL ||
			variadic.SysV_ALSSERegisterCount != 2 ||
			variadic.Win64ShadowSpaceBytes != 0 ||
			len(variadic.Win64VarargFloatMirrors) != 0 {
			return fmt.Errorf(
				"x64 SysV variadic metadata = %#v, want %%al SSE upper bound 2 and no Win64 mirrors",
				variadic,
			)
		}
		fields := []ctarget.LayoutField{{Name: "raw", Type: "ptr"}, {Name: "count", Type: "usize"}}
		aggregate, err := classifier.ClassifySignature(x64abi.ABISignature{
			Params: []x64abi.ABIParam{{Name: "view", Type: "View", Fields: fields}},
			Return: &x64abi.ABIParam{Name: "ret", Type: "View", Fields: fields},
		})
		if err != nil {
			return err
		}
		if got := aggregate.Params[0]; got.SizeBytes != 16 || got.AlignBytes != 8 ||
			got.Class != x64abi.ABIClassInteger ||
			!sameStrings(got.Registers, []string{"rdi", "rsi"}) {
			return fmt.Errorf("x64 SysV aggregate param = %#v, want two integer registers", got)
		}
		if got := aggregate.Return; got.SizeBytes != 16 || got.AlignBytes != 8 ||
			got.Class != x64abi.ABIClassInteger ||
			!sameStrings(got.Registers, []string{"rax", "rdx"}) {
			return fmt.Errorf("x64 SysV aggregate return = %#v, want rax/rdx", got)
		}
	case ctarget.ABIWin64:
		variadic, err := classifier.ClassifySignature(x64abi.ABISignature{
			Variadic:        true,
			FixedParamCount: 1,
			Params: []x64abi.ABIParam{
				{Name: "fmt", Type: "ptr"},
				{Name: "first", Type: "f64"},
				{Name: "count", Type: "i32"},
				{Name: "second", Type: "f32"},
			},
		})
		if err != nil {
			return err
		}
		if !variadic.Variadic || variadic.Win64ShadowSpaceBytes != 32 || variadic.SysVRequiresAL ||
			variadic.SysV_ALSSERegisterCount != 0 {
			return fmt.Errorf(
				"x64 Win64 variadic metadata = %#v, want shadow space and no SysV %%al",
				variadic,
			)
		}
		wantMirrors := []x64abi.VarargFloatMirror{
			{ParamIndex: 1, XMMRegister: "xmm1", GPRegister: "rdx"},
			{ParamIndex: 3, XMMRegister: "xmm3", GPRegister: "r9"},
		}
		if !sameX64Mirrors(variadic.Win64VarargFloatMirrors, wantMirrors) {
			return fmt.Errorf(
				"x64 Win64 float mirrors = %#v, want %#v",
				variadic.Win64VarargFloatMirrors,
				wantMirrors,
			)
		}
		smallFields := []ctarget.LayoutField{{Name: "lo", Type: "u32"}, {Name: "hi", Type: "u32"}}
		small, err := classifier.ClassifySignature(x64abi.ABISignature{
			Params: []x64abi.ABIParam{{Name: "pair", Type: "Pair", Fields: smallFields}},
			Return: &x64abi.ABIParam{Name: "ret", Type: "Pair", Fields: smallFields},
		})
		if err != nil {
			return err
		}
		if got := small.Params[0]; got.Class != x64abi.ABIClassInteger || got.Register != "rcx" ||
			got.SizeBytes != 8 ||
			got.Indirect {
			return fmt.Errorf("x64 Win64 small aggregate param = %#v, want rcx integer scalar", got)
		}
		largeFields := []ctarget.LayoutField{{Name: "a", Type: "ptr"}, {Name: "b", Type: "ptr"}}
		large, err := classifier.ClassifySignature(x64abi.ABISignature{
			Params: []x64abi.ABIParam{{Name: "wide", Type: "Wide", Fields: largeFields}},
			Return: &x64abi.ABIParam{Name: "ret", Type: "Wide", Fields: largeFields},
		})
		if err != nil {
			return err
		}
		if got := large.Params[0]; got.Class != x64abi.ABIClassMemory || !got.Indirect ||
			got.Register != "rcx" ||
			got.SizeBytes != 16 ||
			got.ABIBytes != 8 {
			return fmt.Errorf(
				"x64 Win64 large aggregate param = %#v, want by-reference pointer in rcx",
				got,
			)
		}
	default:
		return fmt.Errorf("x64 unsupported ABI %s", tgt.ABI)
	}
	if _, err := classifier.ClassifySignature(x64abi.ABISignature{
		Variadic:        true,
		FixedParamCount: 3,
		Params:          []x64abi.ABIParam{{Name: "fmt", Type: "ptr"}, {Name: "value", Type: "i32"}},
	}); err == nil || !strings.Contains(err.Error(), "invalid variadic fixed parameter count") {
		return fmt.Errorf("x64 invalid variadic fixed prefix diagnostic = %v", err)
	}
	return nil
}

func CheckX32SysVClassifier(tgt ctarget.Target) error {
	classifier, err := x64abi.NewClassifier(tgt)
	if err != nil {
		return err
	}
	if classifier.Name() != "x32-sysv" || !classifier.UsesX64Registers() {
		return fmt.Errorf(
			"x32 classifier identity = %s x64regs=%v, want x32-sysv with x86_64 registers",
			classifier.Name(),
			classifier.UsesX64Registers(),
		)
	}
	plan, err := classifier.ClassifySignature(x64abi.ABISignature{
		Params: []x64abi.ABIParam{
			{Name: "p", Type: "ptr"},
			{Name: "n", Type: "usize"},
			{Name: "s", Type: "isize"},
			{Name: "wide", Type: "u64"},
			{Name: "f", Type: "f64"},
		},
		Return: &x64abi.ABIParam{Name: "ret", Type: "ptr"},
	})
	if err != nil {
		return err
	}
	if plan.PointerWidthBits != 32 || plan.RegisterWidthBits != 64 {
		return fmt.Errorf(
			"x32 ABI plan identity = %#v, want 32-bit pointers and 64-bit registers",
			plan,
		)
	}
	if err := expectX64Arg(
		plan.Params[0],
		"p",
		x64abi.ABIClassInteger,
		"rdi",
		4,
		4,
		64,
		x64abi.ABIExtendZero,
	); err != nil {
		return err
	}
	if err := expectX64Arg(
		plan.Params[1],
		"n",
		x64abi.ABIClassInteger,
		"rsi",
		4,
		4,
		64,
		x64abi.ABIExtendZero,
	); err != nil {
		return err
	}
	if err := expectX64Arg(
		plan.Params[2],
		"s",
		x64abi.ABIClassInteger,
		"rdx",
		4,
		4,
		64,
		x64abi.ABIExtendSign,
	); err != nil {
		return err
	}
	if err := expectX64Arg(
		plan.Params[3],
		"wide",
		x64abi.ABIClassInteger,
		"rcx",
		8,
		8,
		64,
		x64abi.ABIExtendNone,
	); err != nil {
		return err
	}
	if err := expectX64Arg(
		plan.Params[4],
		"f",
		x64abi.ABIClassSSE,
		"xmm0",
		8,
		8,
		128,
		x64abi.ABIExtendNone,
	); err != nil {
		return err
	}
	if got := plan.Return; got.Register != "rax" || got.Class != x64abi.ABIClassInteger ||
		got.SizeBytes != 4 ||
		got.AlignBytes != 4 ||
		got.RegisterWidthBits != 64 ||
		got.Extension != x64abi.ABIExtendZero {
		return fmt.Errorf("x32 ptr return = %#v, want zero-extended 32-bit pointer in rax", got)
	}
	x86Tgt, err := ctarget.Parse("x86")
	if err != nil {
		return err
	}
	if _, err := x64abi.NewClassifier(x86Tgt); err == nil ||
		!strings.Contains(err.Error(), "x64abi classifier requires x64 ISA") {
		return fmt.Errorf("x32 classifier did not keep i386 separate: %v", err)
	}
	return nil
}

func CheckX32SysVVarargsAndAggregates(tgt ctarget.Target) error {
	classifier, err := x64abi.NewClassifier(tgt)
	if err != nil {
		return err
	}
	variadic, err := classifier.ClassifySignature(x64abi.ABISignature{
		Variadic:        true,
		FixedParamCount: 1,
		Params: []x64abi.ABIParam{
			{Name: "fmt", Type: "ptr"},
			{Name: "first", Type: "f64"},
			{Name: "count", Type: "i32"},
			{Name: "second", Type: "f32"},
		},
	})
	if err != nil {
		return err
	}
	if !variadic.Variadic || variadic.FixedParamCount != 1 || variadic.VarargStartIndex != 1 ||
		!variadic.RegisterVarargs {
		return fmt.Errorf(
			"x32 variadic metadata = %#v, want register varargs after fixed prefix",
			variadic,
		)
	}
	if !variadic.SysVRequiresAL || variadic.SysV_ALSSERegisterCount != 2 ||
		variadic.VarargRegisterSaveBytes != 176 {
		return fmt.Errorf(
			"x32 SysV vararg AL metadata = %#v, want %%al upper bound 2 and 176-byte save area",
			variadic,
		)
	}
	if variadic.Win64ShadowSpaceBytes != 0 || len(variadic.Win64VarargFloatMirrors) != 0 {
		return fmt.Errorf("x32 varargs unexpectedly used Win64 metadata: %#v", variadic)
	}
	fields := []ctarget.LayoutField{{Name: "raw", Type: "ptr"}, {Name: "count", Type: "usize"}}
	aggregate, err := classifier.ClassifySignature(x64abi.ABISignature{
		Params: []x64abi.ABIParam{{Name: "view", Type: "View", Fields: fields}},
		Return: &x64abi.ABIParam{Name: "ret", Type: "View", Fields: fields},
	})
	if err != nil {
		return err
	}
	if got := aggregate.Params[0]; got.SizeBytes != 8 || got.AlignBytes != 4 ||
		got.Class != x64abi.ABIClassInteger ||
		got.Register != "rdi" ||
		!sameStrings(got.Registers, []string{"rdi"}) {
		return fmt.Errorf(
			"x32 aggregate param = %#v, want one integer register carrying ptr32+usize32 aggregate",
			got,
		)
	}
	if got := aggregate.Return; got.SizeBytes != 8 || got.AlignBytes != 4 ||
		got.Class != x64abi.ABIClassInteger ||
		got.Register != "rax" ||
		!sameStrings(got.Registers, []string{"rax"}) {
		return fmt.Errorf(
			"x32 aggregate return = %#v, want rax carrying ptr32+usize32 aggregate",
			got,
		)
	}
	x64Tgt, err := ctarget.Parse("x64")
	if err != nil {
		return err
	}
	x64Classifier, err := x64abi.NewClassifier(x64Tgt)
	if err != nil {
		return err
	}
	x64Plan, err := x64Classifier.ClassifySignature(x64abi.ABISignature{
		Params: []x64abi.ABIParam{{Name: "view", Type: "View", Fields: fields}},
	})
	if err != nil {
		return err
	}
	if got := x64Plan.Params[0]; got.SizeBytes != 16 || got.AlignBytes != 8 ||
		!sameStrings(got.Registers, []string{"rdi", "rsi"}) {
		return fmt.Errorf(
			"x32 aggregate comparison failed: x64 aggregate = %#v, want two-register LP64 layout",
			got,
		)
	}
	largeFields := []ctarget.LayoutField{
		{Name: "a", Type: "ptr"},
		{Name: "b", Type: "ptr"},
		{Name: "c", Type: "ptr"},
		{Name: "d", Type: "ptr"},
		{Name: "e", Type: "ptr"},
	}
	large, err := classifier.ClassifySignature(x64abi.ABISignature{
		Params: []x64abi.ABIParam{{Name: "large", Type: "Large", Fields: largeFields}},
		Return: &x64abi.ABIParam{Name: "ret", Type: "Large", Fields: largeFields},
	})
	if err != nil {
		return err
	}
	if got := large.Params[0]; got.Class != x64abi.ABIClassMemory || got.Register != "" ||
		got.StackOffsetBytes != 0 ||
		got.StackSlotBytes != 24 ||
		got.SizeBytes != 20 {
		return fmt.Errorf(
			"x32 large aggregate param = %#v, want 20-byte memory aggregate in 24-byte stack slot",
			got,
		)
	}
	if got := large.Return; got.Class != x64abi.ABIClassMemory || !got.Indirect ||
		got.Register != "rdi" ||
		got.SizeBytes != 20 {
		return fmt.Errorf("x32 large aggregate return = %#v, want hidden sret pointer in rdi", got)
	}
	return nil
}

func expectX86StackArg(
	got x86abi.ABILocation,
	name string,
	class x86abi.ABIClass,
	offset int,
	slot int,
	size int,
	align int,
) error {
	if got.Name != name || got.Class != class || got.Register != "" ||
		got.StackOffsetBytes != offset ||
		got.StackSlotBytes != slot ||
		got.SizeBytes != size ||
		got.AlignBytes != align {
		return fmt.Errorf(
			"x86 %s stack arg = %#v, want class=%s offset=%d slot=%d size=%d align=%d",
			name,
			got,
			class,
			offset,
			slot,
			size,
			align,
		)
	}
	return nil
}

func expectX64Arg(
	got x64abi.ABILocation,
	name string,
	class x64abi.ABIClass,
	register string,
	size int,
	align int,
	regWidth int,
	extend x64abi.ABIExtension,
) error {
	if got.Name != name || got.Class != class || got.Register != register ||
		got.SizeBytes != size ||
		got.AlignBytes != align ||
		got.RegisterWidthBits != regWidth ||
		got.Extension != extend {
		return fmt.Errorf(
			"x64 %s arg = %#v, want class=%s register=%s size=%d align=%d regWidth=%d extend=%s",
			name,
			got,
			class,
			register,
			size,
			align,
			regWidth,
			extend,
		)
	}
	return nil
}

func sameStrings(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameX64Mirrors(a []x64abi.VarargFloatMirror, b []x64abi.VarargFloatMirror) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ---- runtime_boundaries.go ----

type RuntimeBoundaryDeps struct {
	BuildExecutable                 func(srcPath string, outPath string, target string) error
	DiagnosticFromError             func(error) DiagnosticSummary
	TargetRuntimeDiagnosticCode     string
	TargetSupportsNetRuntimeSymbols func(target string, symbols []string) bool
	RequiredNetRuntimeSymbols       func() []string
	NetRuntimeSymbolForBuiltin      func(name string) (string, bool)
}

type DiagnosticSummary struct {
	Code     string
	Message  string
	Severity string
	Hint     string
}

type RuntimeBoundaryCase struct {
	Name        string
	Source      string
	WantMessage string
}

func CheckStdlibRuntimeBoundaryDiagnostics(tgt ctarget.Target, deps RuntimeBoundaryDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-stdlib-runtime-boundary-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	cases := []struct {
		name        string
		runtimeName string
		src         string
	}{}
	if tgt.Triple != "linux-x86" && tgt.Triple != "linux-x32" {
		cases = append(cases, struct {
			name        string
			runtimeName string
			src         string
		}{
			name:        "filesystem",
			runtimeName: "filesystem",
			src: `
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        if core.fs_exists("README.md", cap):
            return 0
    return 1
`,
		})
	}
	if !targetSupportsNetRuntimeSymbols(deps, tgt.Triple, requiredNetRuntimeSymbols(deps)) {
		cases = append(cases, struct {
			name        string
			runtimeName string
			src         string
		}{
			name:        "networking",
			runtimeName: "networking",
			src: `
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        return core.net_epoll_create(cap)
    return 1
`,
		})
	}

	for _, tc := range cases {
		srcPath := filepath.Join(tmpDir, tc.name+".tetra")
		outPath := filepath.Join(tmpDir, tc.name+"-"+tgt.Triple)
		if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
			return err
		}
		err := buildRuntimeBoundaryExecutable(deps, srcPath, outPath, tgt.Triple)
		if err == nil {
			return fmt.Errorf(
				"%s accepted unsupported %s stdlib runtime boundary",
				tgt.Triple,
				tc.runtimeName,
			)
		}
		diag := runtimeBoundaryDiagnostic(deps, err)
		wantMessage := fmt.Sprintf("%s runtime not supported on %s", tc.runtimeName, tgt.Triple)
		if diag.Code != targetRuntimeDiagnosticCode(deps) || diag.Severity != "error" ||
			diag.Message != wantMessage {
			return fmt.Errorf(
				"%s %s runtime diagnostic = %#v, want code=%s severity=error message=%q",
				tgt.Triple,
				tc.runtimeName,
				diag,
				targetRuntimeDiagnosticCode(deps),
				wantMessage,
			)
		}
		if !strings.Contains(diag.Hint, "Build this source for linux-x64") {
			return fmt.Errorf(
				"%s %s runtime hint = %q, want linux-x64 guidance",
				tgt.Triple,
				tc.runtimeName,
				diag.Hint,
			)
		}
		if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
			return fmt.Errorf(
				"%s %s runtime rejection wrote output %s (stat err=%v)",
				tgt.Triple,
				tc.runtimeName,
				outPath,
				statErr,
			)
		}
	}
	return nil
}

func CheckTargetRuntimeBoundaryDiagnostics(tgt ctarget.Target, deps RuntimeBoundaryDeps) error {
	cases, err := targetRuntimeBoundaryCases(tgt)
	if err != nil {
		return err
	}
	return checkRuntimeBoundaryDiagnostics(tgt, "tetra-target-runtime-boundary-*", cases, deps)
}

func CheckSurfaceDistributedRuntimeBoundaryDiagnostics(
	tgt ctarget.Target,
	deps RuntimeBoundaryDeps,
) error {
	return checkRuntimeBoundaryDiagnostics(
		tgt,
		"tetra-surface-distributed-runtime-boundary-*",
		[]RuntimeBoundaryCase{
			{
				Name: "surface",
				Source: `
func main() -> Int
uses surface:
    return core.surface_open("demo", 10, 10)
`,
				WantMessage: "surface runtime not supported on " + tgt.Triple,
			},
			{
				Name: "distributed_actors",
				Source: `
func main() -> Int
uses actors, runtime:
    return core.actor_node_status(2)
`,
				WantMessage: "distributed actors runtime not supported on " + tgt.Triple,
			},
		},
		deps,
	)
}

func CheckNetworkingRuntimeBoundaryDiagnostics(tgt ctarget.Target, deps RuntimeBoundaryDeps) error {
	cases := []struct {
		name    string
		uses    string
		prelude string
		expr    string
	}{
		{name: "socket_tcp4", expr: "core.net_socket_tcp4(cap)"},
		{name: "bind_tcp4_loopback", expr: "core.net_bind_tcp4_loopback(3, 0, cap)"},
		{name: "connect_tcp4_loopback", expr: "core.net_connect_tcp4_loopback(3, 0, cap)"},
		{name: "listen", expr: "core.net_listen(3, 8, cap)"},
		{name: "accept4", expr: "core.net_accept4(3, 0, cap)"},
		{
			name:    "read",
			uses:    "alloc, capability, io, mem",
			prelude: "        var buf: []u8 = make_u8(4)\n",
			expr:    "core.net_read(3, buf, 0, 1, cap)",
		},
		{
			name:    "recv",
			uses:    "alloc, capability, io, mem",
			prelude: "        var buf: []u8 = make_u8(4)\n",
			expr:    "core.net_recv(3, buf, 0, 1, cap)",
		},
		{
			name:    "write",
			uses:    "alloc, capability, io, mem",
			prelude: "        var buf: []u8 = make_u8(4)\n",
			expr:    "core.net_write(3, buf, 0, 1, cap)",
		},
		{
			name:    "send",
			uses:    "alloc, capability, io, mem",
			prelude: "        var buf: []u8 = make_u8(4)\n",
			expr:    "core.net_send(3, buf, 0, 1, cap)",
		},
		{name: "epoll_create", expr: "core.net_epoll_create(cap)"},
		{name: "epoll_ctl_add_read", expr: "core.net_epoll_ctl_add_read(4, 3, cap)"},
		{name: "epoll_ctl_add_read_write", expr: "core.net_epoll_ctl_add_read_write(4, 3, cap)"},
		{name: "epoll_ctl_mod_read", expr: "core.net_epoll_ctl_mod_read(4, 3, cap)"},
		{name: "epoll_ctl_mod_read_write", expr: "core.net_epoll_ctl_mod_read_write(4, 3, cap)"},
		{name: "epoll_ctl_delete", expr: "core.net_epoll_ctl_delete(4, 3, cap)"},
		{name: "epoll_wait_one", expr: "core.net_epoll_wait_one(4, 0, cap)"},
		{
			name:    "epoll_wait_one_into",
			uses:    "alloc, capability, io, mem",
			prelude: "        var event: []i32 = make_i32(2)\n",
			expr:    "core.net_epoll_wait_one_into(4, event, 0, cap)",
		},
		{name: "set_nonblocking", expr: "core.net_set_nonblocking(3, cap)"},
		{name: "set_reuseport", expr: "core.net_set_reuseport(3, cap)"},
		{name: "set_tcp_nodelay", expr: "core.net_set_tcp_nodelay(3, cap)"},
		{name: "close", expr: "core.net_close(3, cap)"},
	}
	boundaryCases := make([]RuntimeBoundaryCase, 0, len(cases))
	for _, tc := range cases {
		builtinName := tc.expr
		if openParen := strings.IndexByte(builtinName, '('); openParen >= 0 {
			builtinName = builtinName[:openParen]
		}
		if symbol, ok := netRuntimeSymbolForBuiltin(deps, builtinName); ok &&
			targetSupportsNetRuntimeSymbols(deps, tgt.Triple, []string{symbol}) {
			continue
		}
		uses := tc.uses
		if uses == "" {
			uses = "capability, io"
		}
		boundaryCases = append(boundaryCases, RuntimeBoundaryCase{
			Name: tc.name,
			Source: "func main() -> Int\nuses " + uses + ":\n    unsafe:\n        let cap: cap.io = core.cap_io()\n" +
				tc.prelude +
				"        return " + tc.expr + "\n    return 1\n",
			WantMessage: "networking runtime not supported on " + tgt.Triple,
		})
	}
	return checkRuntimeBoundaryDiagnostics(
		tgt,
		"tetra-networking-runtime-boundary-*",
		boundaryCases,
		deps,
	)
}

func checkRuntimeBoundaryDiagnostics(
	tgt ctarget.Target,
	tmpPattern string,
	cases []RuntimeBoundaryCase,
	deps RuntimeBoundaryDeps,
) error {
	tmpDir, err := os.MkdirTemp("", tmpPattern)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	for _, tc := range cases {
		srcPath := filepath.Join(tmpDir, tc.Name+".tetra")
		outPath := filepath.Join(tmpDir, tc.Name+"-"+tgt.Triple)
		if err := os.WriteFile(srcPath, []byte(tc.Source), 0o644); err != nil {
			return err
		}
		err := buildRuntimeBoundaryExecutable(deps, srcPath, outPath, tgt.Triple)
		if err == nil {
			return fmt.Errorf(
				"%s accepted unsupported %s target runtime boundary",
				tgt.Triple,
				tc.Name,
			)
		}
		diag := runtimeBoundaryDiagnostic(deps, err)
		if diag.Code != targetRuntimeDiagnosticCode(deps) || diag.Severity != "error" ||
			diag.Message != tc.WantMessage {
			return fmt.Errorf(
				"%s %s runtime diagnostic = %#v, want code=%s severity=error message=%q",
				tgt.Triple,
				tc.Name,
				diag,
				targetRuntimeDiagnosticCode(deps),
				tc.WantMessage,
			)
		}
		if !strings.Contains(diag.Hint, "Build this source for linux-x64") {
			return fmt.Errorf(
				"%s %s runtime hint = %q, want linux-x64 guidance",
				tgt.Triple,
				tc.Name,
				diag.Hint,
			)
		}
		if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
			return fmt.Errorf(
				"%s %s runtime rejection wrote output %s (stat err=%v)",
				tgt.Triple,
				tc.Name,
				outPath,
				statErr,
			)
		}
	}
	return nil
}

func targetRuntimeBoundaryCases(tgt ctarget.Target) ([]RuntimeBoundaryCase, error) {
	switch tgt.Triple {
	case "linux-x86":
		return []RuntimeBoundaryCase{
			{
				Name: "actor_fanout_over_two_task",
				Source: `
func slow() -> Int:
    return 1

func fast() -> Int:
    return 2

func extra() -> Int:
    return 3

func main() -> Int
uses runtime:
    let _slow: task.i32 = core.task_spawn_i32("slow")
    let _fast: task.i32 = core.task_spawn_i32("fast")
    let _extra: task.i32 = core.task_spawn_i32("extra")
    return 0
`,
				WantMessage: "actor fanout above 2 runtime not supported on linux-x86",
			},
			{
				Name: "actor_fanout_over_two_actors",
				Source: `
func slow() -> Int
uses actors:
    return 1

func fast() -> Int
uses actors:
    return 2

func extra() -> Int
uses actors:
    return 3

func main() -> Int
uses actors, runtime:
    let _slow: actor = core.spawn("slow")
    let _fast: actor = core.spawn("fast")
    let _extra: actor = core.spawn("extra")
    return 0
`,
				WantMessage: "actor fanout above 2 runtime not supported on linux-x86",
			},
			{
				Name: "actor_fanout_over_two_task_group",
				Source: `
func slow() -> Int:
    return 1

func fast() -> Int:
    return 2

func extra() -> Int:
    return 3

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let _slow: task.i32 = core.task_spawn_group_i32(group, "slow")
    let _fast: task.i32 = core.task_spawn_group_i32(group, "fast")
    let _extra: task.i32 = core.task_spawn_group_i32(group, "extra")
    return 0
`,
				WantMessage: "actor fanout above 2 runtime not supported on linux-x86",
			},
		}, nil
	case "linux-x32":
		return []RuntimeBoundaryCase{
			{
				Name: "actor_fanout_over_two_actors",
				Source: `
func slow() -> Int
uses actors:
    return 1

func fast() -> Int
uses actors:
    return 2

func extra() -> Int
uses actors:
    return 3

func main() -> Int
uses actors, runtime:
    let _slow: actor = core.spawn("slow")
    let _fast: actor = core.spawn("fast")
    let _extra: actor = core.spawn("extra")
    return 0
`,
				WantMessage: "actor fanout above 2 runtime not supported on linux-x32",
			},
			{
				Name: "actor_fanout_over_two_task",
				Source: `
func slow() -> Int:
    return 1

func fast() -> Int:
    return 2

func extra() -> Int:
    return 3

func main() -> Int
uses runtime:
    let _slow: task.i32 = core.task_spawn_i32("slow")
    let _fast: task.i32 = core.task_spawn_i32("fast")
    let _extra: task.i32 = core.task_spawn_i32("extra")
    return 0
`,
				WantMessage: "actor fanout above 2 runtime not supported on linux-x32",
			},
		}, nil
	default:
		return nil, fmt.Errorf("target runtime boundary suite is not defined for %s", tgt.Triple)
	}
}

func buildRuntimeBoundaryExecutable(
	deps RuntimeBoundaryDeps,
	srcPath string,
	outPath string,
	target string,
) error {
	if deps.BuildExecutable == nil {
		return fmt.Errorf("missing runtime boundary build executable callback")
	}
	return deps.BuildExecutable(srcPath, outPath, target)
}

func runtimeBoundaryDiagnostic(deps RuntimeBoundaryDeps, err error) DiagnosticSummary {
	if deps.DiagnosticFromError == nil {
		return DiagnosticSummary{Message: err.Error()}
	}
	return deps.DiagnosticFromError(err)
}

func targetRuntimeDiagnosticCode(deps RuntimeBoundaryDeps) string {
	if deps.TargetRuntimeDiagnosticCode != "" {
		return deps.TargetRuntimeDiagnosticCode
	}
	return "TETRA3003"
}

func targetSupportsNetRuntimeSymbols(
	deps RuntimeBoundaryDeps,
	target string,
	symbols []string,
) bool {
	if deps.TargetSupportsNetRuntimeSymbols == nil {
		return false
	}
	return deps.TargetSupportsNetRuntimeSymbols(target, symbols)
}

func requiredNetRuntimeSymbols(deps RuntimeBoundaryDeps) []string {
	if deps.RequiredNetRuntimeSymbols == nil {
		return nil
	}
	return deps.RequiredNetRuntimeSymbols()
}

func netRuntimeSymbolForBuiltin(deps RuntimeBoundaryDeps, name string) (string, bool) {
	if deps.NetRuntimeSymbolForBuiltin == nil {
		return "", false
	}
	return deps.NetRuntimeSymbolForBuiltin(name)
}

// ---- runtime_build_smoke.go ----

type runtimeBuildSmokeOptions struct {
	target      string
	stem        string
	label       string
	src         string
	wantClass   byte
	wantMachine uint16
}

func CheckX86TimeRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkRuntimeBuildSmoke(runtimeBuildSmokeOptions{
		target:      "linux-x86",
		stem:        "x86-time-runtime",
		label:       "x86 time runtime",
		src:         timeRuntimeSmokeSource(),
		wantClass:   1,
		wantMachine: 0x03,
	}, deps)
}

func CheckX86FilesystemRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkRuntimeBuildSmoke(runtimeBuildSmokeOptions{
		target:      "linux-x86",
		stem:        "x86-filesystem-runtime",
		label:       "x86 filesystem runtime",
		src:         filesystemRuntimeSmokeSource(),
		wantClass:   1,
		wantMachine: 0x03,
	}, deps)
}

func CheckX86FilesystemSchedulerCompositionSmoke(deps RuntimeSmokeDeps) error {
	return checkRuntimeBuildSmoke(runtimeBuildSmokeOptions{
		target:      "linux-x86",
		stem:        "x86-filesystem-scheduler",
		label:       "x86 filesystem scheduler",
		src:         filesystemSchedulerRuntimeSmokeSource(),
		wantClass:   1,
		wantMachine: 0x03,
	}, deps)
}

func CheckX32TimeRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkRuntimeBuildSmoke(runtimeBuildSmokeOptions{
		target:      "linux-x32",
		stem:        "x32-time-runtime",
		label:       "x32 time runtime",
		src:         timeRuntimeSmokeSource(),
		wantClass:   1,
		wantMachine: 0x3e,
	}, deps)
}

func CheckX32FilesystemRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkRuntimeBuildSmoke(runtimeBuildSmokeOptions{
		target:      "linux-x32",
		stem:        "x32-filesystem-runtime",
		label:       "x32 filesystem runtime",
		src:         filesystemRuntimeSmokeSource(),
		wantClass:   1,
		wantMachine: 0x3e,
	}, deps)
}

func CheckX32FilesystemSchedulerCompositionSmoke(deps RuntimeSmokeDeps) error {
	return checkRuntimeBuildSmoke(runtimeBuildSmokeOptions{
		target:      "linux-x32",
		stem:        "x32-filesystem-scheduler-runtime",
		label:       "x32 filesystem scheduler runtime",
		src:         filesystemSchedulerRuntimeSmokeSource(),
		wantClass:   1,
		wantMachine: 0x3e,
	}, deps)
}

func checkRuntimeBuildSmoke(opts runtimeBuildSmokeOptions, deps RuntimeSmokeDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+opts.stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, opts.stem+".tetra")
	outPath := filepath.Join(tmpDir, opts.stem)
	if err := os.WriteFile(srcPath, []byte(opts.src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, opts.target); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	return checkRuntimeSmokeELF(data, opts.label, opts.wantClass, opts.wantMachine)
}

func timeRuntimeSmokeSource() string {
	return `
func main() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    let _until: Int = core.sleep_until(core.deadline_ms(2))
    return core.time_now_ms()
`
}

func filesystemRuntimeSmokeSource() string {
	return `
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        if !core.fs_exists("../README.md", cap):
            return 11
        if core.fs_exists("__tetra_missing_fs_exists_smoke__", cap):
            return 12
    return 0
`
}

func filesystemSchedulerRuntimeSmokeSource() string {
	return `
func worker() -> Int:
    return 41

func main() -> Int
uses capability, io, runtime:
    unsafe:
        let cap: cap.io = core.cap_io()
        if !core.fs_exists("../README.md", cap):
            return 11
        if core.fs_exists("__tetra_missing_fs_exists_smoke__", cap):
            return 12
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`
}

// ---- runtime_selfhost_smoke.go ----

func CheckX32SingleTaskSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke(
		"linux-x32",
		"x32-task-runtime",
		"x32 task runtime",
		singleTaskRuntimeSmokeSource(),
		0x3e,
		deps,
	)
}

func CheckX32TypedTaskSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke(
		"linux-x32",
		"x32-typed-task-runtime",
		"x32 typed-task runtime",
		typedTaskRuntimeSmokeSource(),
		0x3e,
		deps,
	)
}

func CheckX32StagedTypedTaskSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke(
		"linux-x32",
		"x32-staged-typed-task-runtime",
		"x32 staged typed-task runtime",
		stagedTypedTaskRuntimeSmokeSource(),
		0x3e,
		deps,
	)
}

func CheckX32TaskGroupSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke(
		"linux-x32",
		"x32-task-group-runtime",
		"x32 task-group runtime",
		taskGroupRuntimeSmokeSource(),
		0x3e,
		deps,
	)
}

func CheckX32TypedTaskGroupSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke(
		"linux-x32",
		"x32-typed-task-group-runtime",
		"x32 typed-task-group runtime",
		typedTaskGroupRuntimeSmokeSource(),
		0x3e,
		deps,
	)
}

func CheckX32SingleActorSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke(
		"linux-x32",
		"x32-actor-runtime",
		"x32 actor runtime",
		singleActorRuntimeSmokeSource(),
		0x3e,
		deps,
	)
}

func CheckX32ActorStateSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke(
		"linux-x32",
		"x32-actor-state-runtime",
		"x32 actor-state runtime",
		actorStateRuntimeSmokeSource(),
		0x3e,
		deps,
	)
}

func CheckX86SingleTaskSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke(
		"linux-x86",
		"x86-task-runtime",
		"x86 task runtime",
		singleTaskRuntimeSmokeSource(),
		0x03,
		deps,
	)
}

func CheckX86TypedTaskSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke(
		"linux-x86",
		"x86-typed-task-runtime",
		"x86 typed-task runtime",
		typedTaskRuntimeSmokeSource(),
		0x03,
		deps,
	)
}

func CheckX86StagedTypedTaskSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke(
		"linux-x86",
		"x86-staged-typed-task-runtime",
		"x86 staged typed-task runtime",
		stagedTypedTaskRuntimeSmokeSource(),
		0x03,
		deps,
	)
}

func CheckX86TaskGroupSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke(
		"linux-x86",
		"x86-task-group-runtime",
		"x86 task-group runtime",
		taskGroupRuntimeSmokeSource(),
		0x03,
		deps,
	)
}

func CheckX86TypedTaskGroupSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke(
		"linux-x86",
		"x86-typed-task-group-runtime",
		"x86 typed-task-group runtime",
		typedTaskGroupRuntimeSmokeSource(),
		0x03,
		deps,
	)
}

func CheckX86SingleActorSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke(
		"linux-x86",
		"x86-actor-runtime",
		"x86 actor runtime",
		singleActorRuntimeSmokeSource(),
		0x03,
		deps,
	)
}

func CheckX86ActorStateSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke(
		"linux-x86",
		"x86-actor-state-runtime",
		"x86 actor-state runtime",
		actorStateRuntimeSmokeSource(),
		0x03,
		deps,
	)
}

func checkSelfHostRuntimeSmoke(
	target string,
	stem string,
	label string,
	src string,
	wantMachine uint16,
	deps RuntimeSmokeDeps,
) error {
	return checkRuntimeBuildSmoke(runtimeBuildSmokeOptions{
		target:      target,
		stem:        stem,
		label:       label,
		src:         src,
		wantClass:   1,
		wantMachine: wantMachine,
	}, deps)
}

func singleTaskRuntimeSmokeSource() string {
	return `
func worker() -> Int:
    return 41

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`
}

func typedTaskRuntimeSmokeSource() string {
	return `
enum TaskErr:
    case boom(Int)
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(23)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        9
`
}

func stagedTypedTaskRuntimeSmokeSource() string {
	return `
enum TaskErr:
    case boom(Int, Int, Int, Int, Int)
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(1, 2, 3, 4, 5)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(a, b, c, d, e):
        a + b + c + d + e
    case TaskErr.stopped:
        99
`
}

func taskGroupRuntimeSmokeSource() string {
	return `
func worker() -> Int
uses runtime:
    let group: task.group = core.task_group_current()
    let status: Int = core.task_group_status(group)
    if status != 1:
        return 60 + status
    return 7

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let before: Int = core.task_group_status(group)
    if before != 1:
        return 30 + before
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    let closeError: Int = core.task_group_close(group)
    if closeError != 0:
        return 80 + closeError
    let after: Int = core.task_group_status(group)
    if after != 3:
        return 90 + after
    if result.error != 0:
        return 100 + result.error
    return result.value
`
}

func typedTaskGroupRuntimeSmokeSource() string {
	return `
enum TaskErr:
    case boom(Int)
    case stopped

func worker() -> Int throws TaskErr uses runtime:
    let group: task.group = core.task_group_current()
    let status: Int = core.task_group_status(group)
    if status != 1:
        throw TaskErr.boom(60 + status)
    throw TaskErr.boom(23)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let before: Int = core.task_group_status(group)
    if before != 1:
        return 30 + before
    let task = core.task_spawn_group_i32_typed<TaskErr>(group, "worker")
    let result: Int = catch core.task_join_group_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        9
    let closeError: Int = core.task_group_close(group)
    if closeError != 0:
        return 80 + closeError
    let after: Int = core.task_group_status(group)
    if after != 3:
        return 90 + after
    return result
`
}

func singleActorRuntimeSmokeSource() string {
	return `
func worker() -> Int
uses actors:
    let value: Int = core.recv()
    if value == 41:
        let _sent: Int = core.send(core.sender(), 42)
        return 0
    return 1

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let _sent: Int = core.send(peer, 41)
    let reply: Int = core.recv()
    if reply == 42:
        return 0
    return reply
`
}

func actorStateRuntimeSmokeSource() string {
	return `
actor Counter:
    var count: Int = 0
    const enabled: Bool = true
    func run() -> Int
    uses actors:
        let delta: Int = core.recv()
        if enabled:
            count = count + delta + 1
        let _sent: Int = core.send(core.sender(), count)
        return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("Counter.run")
    let _sent: Int = core.send(peer, 41)
    return core.recv()
`
}

// ---- runtime_smoke.go ----

type stdoutExecutableSmokeOptions struct {
	target      string
	stem        string
	label       string
	wantClass   byte
	wantMachine uint16
	wantLiteral string
	wantCode    [][]byte
	forbidCode  []byte
}

type stderrFDRuntimeSmokeOptions struct {
	target      string
	stem        string
	label       string
	wantClass   byte
	wantMachine uint16
	wantCode    [][]byte
	forbidCode  []byte
}

type allocatorExecutableSmokeOptions struct {
	target      string
	stem        string
	label       string
	wantClass   byte
	wantMachine uint16
	wantCode    [][]byte
	forbidCode  [][]byte
}

type allocatorFailureExecutableSmokeOptions struct {
	target      string
	stem        string
	label       string
	wantClass   byte
	wantMachine uint16
	wantCode    [][]byte
	forbidCode  [][]byte
}

type rawMemoryBoundsExecutableSmokeOptions struct {
	target      string
	stem        string
	label       string
	wantClass   byte
	wantMachine uint16
	wantCode    [][]byte
	forbidCode  [][]byte
}

type rawPointerSlotExecutableSmokeOptions struct {
	target      string
	stem        string
	label       string
	wantClass   byte
	wantMachine uint16
	wantCode    [][]byte
	forbidCode  [][]byte
}

type networkingLifecycleRuntimeSmokeOptions struct {
	target      string
	stem        string
	label       string
	wantClass   byte
	wantMachine uint16
	wantCode    [][]byte
	forbidCode  []byte
}

type islandFreeExecutableSmokeOptions struct {
	target          string
	stem            string
	label           string
	wantClass       byte
	wantMachine     uint16
	wantCode        [][]byte
	wantDebugCode   [][]byte
	forbidCode      [][]byte
	forbidDebugCode [][]byte
}

func CheckX86StdoutExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkStdoutExecutableSmoke(stdoutExecutableSmokeOptions{
		target:      "linux-x86",
		stem:        "x86_stdout",
		label:       "x86 stdout executable",
		wantClass:   1,
		wantMachine: 0x03,
		wantLiteral: "x86 stdout\n",
		wantCode: [][]byte{
			{0xB8, 0x04, 0x00, 0x00, 0x00},
			{0xCD, 0x80},
		},
		forbidCode: []byte{0x0F, 0x05},
	}, deps)
}

func CheckX32StdoutExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkStdoutExecutableSmoke(stdoutExecutableSmokeOptions{
		target:      "linux-x32",
		stem:        "x32_stdout",
		label:       "x32 stdout executable",
		wantClass:   1,
		wantMachine: 0x3e,
		wantLiteral: "x32 stdout\n",
		wantCode: [][]byte{
			{0xB8, 0x01, 0x00, 0x00, 0x40},
			{0x0F, 0x05},
		},
		forbidCode: []byte{0xCD, 0x80},
	}, deps)
}

func CheckX86StderrFDRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkStderrFDRuntimeSmoke(stderrFDRuntimeSmokeOptions{
		target:      "linux-x86",
		stem:        "x86_stderr_fd",
		label:       "x86 stderr fd runtime",
		wantClass:   1,
		wantMachine: 0x03,
		wantCode: [][]byte{
			{0xB8, 0x02, 0x00, 0x00, 0x00, 0x50},
			{0x8B, 0x5D, 0x08, 0x8B, 0x4D, 0x0C, 0x03, 0x4D, 0x14},
			{0xB8, 0x04, 0x00, 0x00, 0x00, 0xCD, 0x80},
		},
		forbidCode: []byte{0x0F, 0x05},
	}, deps)
}

func CheckX32StderrFDRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkStderrFDRuntimeSmoke(stderrFDRuntimeSmokeOptions{
		target:      "linux-x32",
		stem:        "x32_stderr_fd",
		label:       "x32 stderr fd runtime",
		wantClass:   1,
		wantMachine: 0x3e,
		wantCode: [][]byte{
			{0xB8, 0x02, 0x00, 0x00, 0x00, 0x50},
			{0x48, 0x63, 0xC9, 0x48, 0x01, 0xCE, 0x4C, 0x89, 0xC2},
			{0xB8, 0x01, 0x00, 0x00, 0x40, 0x0F, 0x05},
		},
		forbidCode: []byte{0xCD, 0x80},
	}, deps)
}

func CheckX86AllocatorExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkAllocatorExecutableSmoke(allocatorExecutableSmokeOptions{
		target:      "linux-x86",
		stem:        "x86_allocator",
		label:       "x86 allocator executable",
		wantClass:   1,
		wantMachine: 0x03,
		wantCode: [][]byte{
			{0xB8, 0xC0, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{0x3D, 0x01, 0xF0, 0xFF, 0xFF, 0x0F, 0x83},
			{0xBB, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x01, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{0x89, 0x08, 0x83, 0xC0, 0x08},
		},
		forbidCode: [][]byte{{0x0F, 0x05}},
	}, deps)
}

func CheckX86AllocatorFailureExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkAllocatorFailureExecutableSmoke(allocatorFailureExecutableSmokeOptions{
		target:      "linux-x86",
		stem:        "x86_allocator_failure",
		label:       "x86 allocator failure executable",
		wantClass:   1,
		wantMachine: 0x03,
		wantCode: [][]byte{
			{0x83, 0xF9, 0x01, 0x0F, 0x8D},
			{0xBB, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x01, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{0xB8, 0xC0, 0x00, 0x00, 0x00, 0xCD, 0x80},
		},
		forbidCode: [][]byte{{0x0F, 0x05}},
	}, deps)
}

func CheckX32AllocatorExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkAllocatorExecutableSmoke(allocatorExecutableSmokeOptions{
		target:      "linux-x32",
		stem:        "x32_allocator",
		label:       "x32 allocator executable",
		wantClass:   1,
		wantMachine: 0x3e,
		wantCode: [][]byte{
			{0xB8, 0x09, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{0x48, 0x3D, 0x01, 0xF0, 0xFF, 0xFF, 0x0F, 0x83},
			{0xBF, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x3C, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{0x89, 0x30, 0x48, 0x05, 0x08, 0x00, 0x00, 0x00},
		},
		forbidCode: [][]byte{
			{0xCD, 0x80},
			{0xB8, 0x09, 0x00, 0x00, 0x00, 0x0F, 0x05},
			{0xB8, 0x3C, 0x00, 0x00, 0x00, 0x0F, 0x05},
		},
	}, deps)
}

func CheckX32AllocatorFailureExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkAllocatorFailureExecutableSmoke(allocatorFailureExecutableSmokeOptions{
		target:      "linux-x32",
		stem:        "x32_allocator_failure",
		label:       "x32 allocator failure executable",
		wantClass:   1,
		wantMachine: 0x3e,
		wantCode: [][]byte{
			{0x89, 0xF0, 0x3D, 0x01, 0x00, 0x00, 0x00, 0x0F, 0x8D},
			{0xBF, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x3C, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{0xB8, 0x09, 0x00, 0x00, 0x40, 0x0F, 0x05},
		},
		forbidCode: [][]byte{
			{0xCD, 0x80},
			{0xB8, 0x3C, 0x00, 0x00, 0x00, 0x0F, 0x05},
			{0xB8, 0x09, 0x00, 0x00, 0x00, 0x0F, 0x05},
		},
	}, deps)
}

func CheckX86RawMemoryBoundsExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkRawMemoryBoundsExecutableSmoke(rawMemoryBoundsExecutableSmokeOptions{
		target:      "linux-x86",
		stem:        "x86_raw_memory_bounds",
		label:       "x86 raw memory bounds executable",
		wantClass:   1,
		wantMachine: 0x03,
		wantCode: [][]byte{
			{0xB8, 0xC0, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{0x3D, 0x01, 0xF0, 0xFF, 0xFF, 0x0F, 0x83},
			{0xBB, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x01, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{
				0xB8, 0x00, 0x00, 0x00, 0x00, 0x50, 0x8B, 0x85,
				0xF8, 0xFF, 0xFF, 0xFF, 0x50, 0x59, 0x5A, 0x58,
				0x83, 0xFA, 0x00, 0x0F, 0x8D,
			},
			{0x81, 0xE7, 0x00, 0xF0, 0xFF, 0xFF},
			{0x8B, 0x0F, 0x83, 0xC7, 0x08},
			{0x01, 0xC2, 0x83, 0xC2, 0x04, 0x39, 0xCA},
			{0x01, 0xC2, 0x83, 0xC2, 0x01, 0x39, 0xCA},
			{0x88, 0x18, 0x53},
			{0x0F, 0xB6, 0x00, 0x50},
		},
		forbidCode: [][]byte{{0x0F, 0x05}},
	}, deps)
}

func CheckX32RawMemoryBoundsExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkRawMemoryBoundsExecutableSmoke(rawMemoryBoundsExecutableSmokeOptions{
		target:      "linux-x32",
		stem:        "x32_raw_memory_bounds",
		label:       "x32 raw memory bounds executable",
		wantClass:   1,
		wantMachine: 0x3e,
		wantCode: [][]byte{
			{0xB8, 0x09, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{0x48, 0x3D, 0x01, 0xF0, 0xFF, 0xFF, 0x0F, 0x83},
			{0xBF, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x3C, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{
				0xB8, 0x00, 0x00, 0x00, 0x00, 0x50, 0x48, 0x8B,
				0x85, 0xF8, 0xFF, 0xFF, 0xFF, 0x50, 0x59, 0x5A,
				0x58, 0x81, 0xFA, 0x00, 0x00, 0x00, 0x00, 0x0F,
				0x8D,
			},
			{0x48, 0x89, 0xC7, 0x48, 0x81, 0xE7, 0x00, 0xF0, 0xFF, 0xFF},
			{0x8B, 0x8F, 0x00, 0x00, 0x00, 0x00, 0x48, 0x81, 0xC7, 0x08, 0x00, 0x00, 0x00},
			{0x48, 0x29, 0xF8, 0x48, 0x01, 0xC2, 0x81, 0xC2, 0x04, 0x00, 0x00, 0x00, 0x39, 0xCA},
			{0x48, 0x29, 0xF8, 0x48, 0x01, 0xC2, 0x81, 0xC2, 0x01, 0x00, 0x00, 0x00, 0x39, 0xCA},
			{0x48, 0x63, 0xD2, 0x48, 0x89, 0xF8, 0x48, 0x01, 0xD0},
			{0x44, 0x88, 0x00, 0x41, 0x50},
			{0x0F, 0xB6, 0x00, 0x50},
		},
		forbidCode: [][]byte{
			{0xCD, 0x80},
			{0xB8, 0x09, 0x00, 0x00, 0x00, 0x0F, 0x05},
			{0xB8, 0x3C, 0x00, 0x00, 0x00, 0x0F, 0x05},
		},
	}, deps)
}

func CheckX86RawPointerSlotExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkRawPointerSlotExecutableSmoke(rawPointerSlotExecutableSmokeOptions{
		target:      "linux-x86",
		stem:        "x86_raw_pointer_slot",
		label:       "x86 raw pointer slot executable",
		wantClass:   1,
		wantMachine: 0x03,
		wantCode: [][]byte{
			{0xB8, 0xC0, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{0x3D, 0x01, 0xF0, 0xFF, 0xFF, 0x0F, 0x83},
			{0xBB, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x01, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{
				0xB8, 0x00, 0x00, 0x00, 0x00, 0x50, 0x8B, 0x85,
				0xF8, 0xFF, 0xFF, 0xFF, 0x50, 0x59, 0x5A, 0x58,
				0x83, 0xFA, 0x00, 0x0F, 0x8D,
			},
			{0x81, 0xE7, 0x00, 0xF0, 0xFF, 0xFF},
			{0x8B, 0x0F, 0x83, 0xC7, 0x08},
			{0x01, 0xC2, 0x83, 0xC2, 0x04, 0x39, 0xCA},
			{0x89, 0x18, 0x53},
			{0x8B, 0x00, 0x50},
		},
		forbidCode: [][]byte{{0x0F, 0x05}},
	}, deps)
}

func CheckX32RawPointerSlotExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkRawPointerSlotExecutableSmoke(rawPointerSlotExecutableSmokeOptions{
		target:      "linux-x32",
		stem:        "x32_raw_pointer_slot",
		label:       "x32 raw pointer slot executable",
		wantClass:   1,
		wantMachine: 0x3e,
		wantCode: [][]byte{
			{0xB8, 0x09, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{0x48, 0x3D, 0x01, 0xF0, 0xFF, 0xFF, 0x0F, 0x83},
			{0xBF, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x3C, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{
				0xB8, 0x00, 0x00, 0x00, 0x00, 0x50, 0x48, 0x8B,
				0x85, 0xF8, 0xFF, 0xFF, 0xFF, 0x50, 0x59, 0x5A,
				0x58, 0x81, 0xFA, 0x00, 0x00, 0x00, 0x00, 0x0F,
				0x8D,
			},
			{0x48, 0x89, 0xC7, 0x48, 0x81, 0xE7, 0x00, 0xF0, 0xFF, 0xFF},
			{0x8B, 0x8F, 0x00, 0x00, 0x00, 0x00, 0x48, 0x81, 0xC7, 0x08, 0x00, 0x00, 0x00},
			{0x48, 0x29, 0xF8, 0x48, 0x01, 0xC2, 0x81, 0xC2, 0x04, 0x00, 0x00, 0x00, 0x39, 0xCA},
			{0x48, 0x63, 0xD2, 0x48, 0x89, 0xF8, 0x48, 0x01, 0xD0},
			{0x48, 0x89, 0xC7, 0x45, 0x89, 0xC0, 0x44, 0x89, 0x07, 0x41, 0x50},
			{0x8B, 0x00, 0x50},
		},
		forbidCode: [][]byte{
			{0xCD, 0x80},
			{0xB8, 0x09, 0x00, 0x00, 0x00, 0x0F, 0x05},
			{0xB8, 0x3C, 0x00, 0x00, 0x00, 0x0F, 0x05},
		},
	}, deps)
}

func CheckX86RawPointerOffsetSlotExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkRawPointerOffsetSlotExecutableSmoke(rawPointerSlotExecutableSmokeOptions{
		target:      "linux-x86",
		stem:        "x86_raw_pointer_offset_slot",
		label:       "x86 raw pointer offset slot executable",
		wantClass:   1,
		wantMachine: 0x03,
		wantCode: [][]byte{
			{0xB8, 0xC0, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{0x3D, 0x01, 0xF0, 0xFF, 0xFF, 0x0F, 0x83},
			{0xBB, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x01, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{0x81, 0xE7, 0x00, 0xF0, 0xFF, 0xFF},
			{0x8B, 0x0F, 0x83, 0xC7, 0x08},
			{0x01, 0xC2, 0x83, 0xC2, 0x04, 0x39, 0xCA},
			{0x89, 0x18, 0x53},
			{0x8B, 0x00, 0x50},
		},
		forbidCode: [][]byte{
			{0x0F, 0x05},
			{0x01, 0xC2, 0x83, 0xC2, 0x01, 0x39, 0xCA},
		},
	}, deps)
}

func CheckX32RawPointerOffsetSlotExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkRawPointerOffsetSlotExecutableSmoke(rawPointerSlotExecutableSmokeOptions{
		target:      "linux-x32",
		stem:        "x32_raw_pointer_offset_slot",
		label:       "x32 raw pointer offset slot executable",
		wantClass:   1,
		wantMachine: 0x3e,
		wantCode: [][]byte{
			{0xB8, 0x09, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{0x48, 0x3D, 0x01, 0xF0, 0xFF, 0xFF, 0x0F, 0x83},
			{0xBF, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x3C, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{0x48, 0x89, 0xC7, 0x48, 0x81, 0xE7, 0x00, 0xF0, 0xFF, 0xFF},
			{0x8B, 0x8F, 0x00, 0x00, 0x00, 0x00, 0x48, 0x81, 0xC7, 0x08, 0x00, 0x00, 0x00},
			{0x48, 0x29, 0xF8, 0x48, 0x01, 0xC2, 0x81, 0xC2, 0x04, 0x00, 0x00, 0x00, 0x39, 0xCA},
			{0x48, 0x63, 0xD2, 0x48, 0x89, 0xF8, 0x48, 0x01, 0xD0},
			{0x48, 0x89, 0xC7, 0x45, 0x89, 0xC0, 0x44, 0x89, 0x07, 0x41, 0x50},
			{0x8B, 0x00, 0x50},
		},
		forbidCode: [][]byte{
			{0xCD, 0x80},
			{0xB8, 0x09, 0x00, 0x00, 0x00, 0x0F, 0x05},
			{0xB8, 0x3C, 0x00, 0x00, 0x00, 0x0F, 0x05},
			{0x48, 0x29, 0xF8, 0x48, 0x01, 0xC2, 0x81, 0xC2, 0x01, 0x00, 0x00, 0x00, 0x39, 0xCA},
		},
	}, deps)
}

func CheckX86NetworkingLifecycleRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkNetworkingLifecycleRuntimeSmoke(networkingLifecycleRuntimeSmokeOptions{
		target:      "linux-x86",
		stem:        "x86_networking_lifecycle",
		label:       "x86 networking lifecycle runtime",
		wantClass:   1,
		wantMachine: 0x03,
		wantCode: [][]byte{
			{0xB8, 0x66, 0x00, 0x00, 0x00},
			{0xBB, 0x01, 0x00, 0x00, 0x00},
			{0xBB, 0x02, 0x00, 0x00, 0x00},
			{0xBB, 0x03, 0x00, 0x00, 0x00},
			{0xBB, 0x04, 0x00, 0x00, 0x00},
			{0xBB, 0x09, 0x00, 0x00, 0x00},
			{0xBB, 0x0A, 0x00, 0x00, 0x00},
			{0xBB, 0x0E, 0x00, 0x00, 0x00},
			{0xBB, 0x12, 0x00, 0x00, 0x00},
			{0xB8, 0x03, 0x00, 0x00, 0x00},
			{0xB8, 0x04, 0x00, 0x00, 0x00},
			{0xB8, 0x49, 0x01, 0x00, 0x00},
			{0xB8, 0xFF, 0x00, 0x00, 0x00},
			{0xB8, 0x00, 0x01, 0x00, 0x00},
			{0xB8, 0x37, 0x00, 0x00, 0x00},
			{0x0D, 0x00, 0x08, 0x00, 0x00},
			{0xB8, 0x06, 0x00, 0x00, 0x00},
			{0xCD, 0x80},
		},
		forbidCode: []byte{0xB8, 0x03, 0x00, 0x00, 0x40},
	}, deps)
}

func CheckX32NetworkingLifecycleRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkNetworkingLifecycleRuntimeSmoke(networkingLifecycleRuntimeSmokeOptions{
		target:      "linux-x32",
		stem:        "x32_networking_lifecycle",
		label:       "x32 networking lifecycle runtime",
		wantClass:   1,
		wantMachine: 0x3e,
		wantCode: [][]byte{
			{0xB8, 0x29, 0x00, 0x00, 0x40},
			{0xB8, 0x31, 0x00, 0x00, 0x40},
			{0xB8, 0x2A, 0x00, 0x00, 0x40},
			{0xB8, 0x32, 0x00, 0x00, 0x40},
			{0xB8, 0x20, 0x01, 0x00, 0x40},
			{0xB8, 0x00, 0x00, 0x00, 0x40},
			{0xB8, 0x01, 0x00, 0x00, 0x40},
			{0xB8, 0x2C, 0x00, 0x00, 0x40},
			{0xB8, 0x05, 0x02, 0x00, 0x40},
			{0xB8, 0x1D, 0x02, 0x00, 0x40},
			{0xB8, 0xE8, 0x00, 0x00, 0x40},
			{0xB8, 0xE9, 0x00, 0x00, 0x40},
			{0xB8, 0x23, 0x01, 0x00, 0x40},
			{0xB8, 0x48, 0x00, 0x00, 0x40},
			{0x0D, 0x00, 0x08, 0x00, 0x00},
			{0xB8, 0x03, 0x00, 0x00, 0x40},
			{0x0F, 0x05},
		},
		forbidCode: []byte{0xCD, 0x80},
	}, deps)
}

func CheckX86IslandFreeExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkIslandFreeExecutableSmoke(islandFreeExecutableSmokeOptions{
		target:      "linux-x86",
		stem:        "x86_island_free",
		label:       "x86 island free executable",
		wantClass:   1,
		wantMachine: 0x03,
		wantCode: [][]byte{
			{0xB8, 0xC0, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{0xC7, 0x00, 0x10, 0x00, 0x00, 0x00},
			{0x8B, 0x4B, 0x08, 0xB8, 0x5B, 0x00, 0x00, 0x00, 0xCD, 0x80},
		},
		wantDebugCode: [][]byte{
			{0xC7, 0x00, 0x00, 0x10, 0x00, 0x00},
			{0x8B, 0x43, 0x0C, 0x85, 0xC0, 0x0F, 0x84},
			{0xBB, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x01, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{0xC7, 0x43, 0x0C, 0x01, 0x00, 0x00, 0x00},
			{0xB8, 0x7D, 0x00, 0x00, 0x00, 0xCD, 0x80},
		},
		forbidCode: [][]byte{{0x0F, 0x05}},
		forbidDebugCode: [][]byte{
			{0x8B, 0x4B, 0x08, 0xB8, 0x5B, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{0x0F, 0x05},
		},
	}, deps)
}

func CheckX32IslandFreeExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkIslandFreeExecutableSmoke(islandFreeExecutableSmokeOptions{
		target:      "linux-x32",
		stem:        "x32_island_free",
		label:       "x32 island free executable",
		wantClass:   1,
		wantMachine: 0x3e,
		wantCode: [][]byte{
			{0xB8, 0x09, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{0xC7, 0x00, 0x10, 0x00, 0x00, 0x00},
			{0x8B, 0x77, 0x08, 0xB8, 0x0B, 0x00, 0x00, 0x40, 0x0F, 0x05},
		},
		wantDebugCode: [][]byte{
			{0xC7, 0x00, 0x00, 0x10, 0x00, 0x00},
			{0x8B, 0x47, 0x0C, 0x85, 0xC0, 0x0F, 0x84},
			{0xBF, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x3C, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{0x48, 0x89, 0xF8, 0xC7, 0x40, 0x0C, 0x01, 0x00, 0x00, 0x00},
			{0x8B, 0x47, 0x08, 0x2D, 0x00, 0x10, 0x00, 0x00, 0x48, 0x89, 0xC6},
			{0xB8, 0x0A, 0x00, 0x00, 0x40, 0x0F, 0x05},
		},
		forbidCode: [][]byte{
			{0xCD, 0x80},
			{0xB8, 0x0B, 0x00, 0x00, 0x00, 0x0F, 0x05},
		},
		forbidDebugCode: [][]byte{
			{0xCD, 0x80},
			{0xB8, 0x0A, 0x00, 0x00, 0x00, 0x0F, 0x05},
			{0xB8, 0x0B, 0x00, 0x00, 0x40, 0x0F, 0x05},
		},
	}, deps)
}

func checkStderrFDRuntimeSmoke(opts stderrFDRuntimeSmokeOptions, deps RuntimeSmokeDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+opts.stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, opts.stem+".tetra")
	outPath := filepath.Join(tmpDir, opts.stem)
	src := `
func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let cap: cap.io = core.cap_io()
        var buf: []u8 = core.make_u8(1)
        buf[0] = 69
        let written: Int = core.net_write(2, buf, 0, 1, cap)
        if written == 999:
            return 7
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, opts.target); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if err := checkRuntimeSmokeELF(data, opts.label, opts.wantClass, opts.wantMachine); err != nil {
		return err
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing stderr fd/write sequence % x", opts.label, wantCode)
		}
	}
	if len(opts.forbidCode) > 0 && bytes.Contains(data, opts.forbidCode) {
		return fmt.Errorf("%s contains forbidden syscall sequence % x", opts.label, opts.forbidCode)
	}
	return nil
}

func checkStdoutExecutableSmoke(opts stdoutExecutableSmokeOptions, deps RuntimeSmokeDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+opts.stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, opts.stem+".tetra")
	outPath := filepath.Join(tmpDir, opts.stem)
	src := fmt.Sprintf(
		"func main() -> Int\nuses io:\n    print(%q)\n    return 0\n",
		opts.wantLiteral,
	)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, opts.target); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if err := checkRuntimeSmokeELF(data, opts.label, opts.wantClass, opts.wantMachine); err != nil {
		return err
	}
	if !bytes.Contains(data, []byte(opts.wantLiteral)) {
		return fmt.Errorf("%s missing stdout string literal %q", opts.label, opts.wantLiteral)
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing target write syscall sequence % x", opts.label, wantCode)
		}
	}
	if len(opts.forbidCode) > 0 && bytes.Contains(data, opts.forbidCode) {
		return fmt.Errorf("%s contains forbidden syscall sequence % x", opts.label, opts.forbidCode)
	}
	return nil
}

func checkAllocatorExecutableSmoke(
	opts allocatorExecutableSmokeOptions,
	deps RuntimeSmokeDeps,
) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+opts.stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, opts.stem+".tetra")
	outPath := filepath.Join(tmpDir, opts.stem)
	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let _: Int = core.store_i32(p, 42, mem)
        return core.load_i32(p, mem)
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, opts.target); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if err := checkRuntimeSmokeELF(data, opts.label, opts.wantClass, opts.wantMachine); err != nil {
		return err
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing target allocator sequence % x", opts.label, wantCode)
		}
	}
	for _, forbidCode := range opts.forbidCode {
		if len(forbidCode) > 0 && bytes.Contains(data, forbidCode) {
			return fmt.Errorf(
				"%s contains forbidden allocator sequence % x",
				opts.label,
				forbidCode,
			)
		}
	}
	return nil
}

func checkAllocatorFailureExecutableSmoke(
	opts allocatorFailureExecutableSmokeOptions,
	deps RuntimeSmokeDeps,
) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+opts.stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, opts.stem+".tetra")
	outPath := filepath.Join(tmpDir, opts.stem)
	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let _: ptr = core.alloc_bytes(0)
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, opts.target); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if err := checkRuntimeSmokeELF(data, opts.label, opts.wantClass, opts.wantMachine); err != nil {
		return err
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing allocator failure sequence % x", opts.label, wantCode)
		}
	}
	for _, forbidCode := range opts.forbidCode {
		if len(forbidCode) > 0 && bytes.Contains(data, forbidCode) {
			return fmt.Errorf(
				"%s contains forbidden allocator failure sequence % x",
				opts.label,
				forbidCode,
			)
		}
	}
	return nil
}

func checkRawMemoryBoundsExecutableSmoke(
	opts rawMemoryBoundsExecutableSmokeOptions,
	deps RuntimeSmokeDeps,
) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+opts.stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, opts.stem+".tetra")
	outPath := filepath.Join(tmpDir, opts.stem)
	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let stored_i32: Int = core.store_i32(p, 42, mem)
        let q: ptr = core.ptr_add(p, 1, mem)
        let stored_u8: u8 = core.store_u8(q, 7, mem)
        let direct: Int = core.load_i32(p, mem)
        let loaded_u8: u8 = core.load_u8(q, mem)
        return direct
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, opts.target); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if err := checkRuntimeSmokeELF(data, opts.label, opts.wantClass, opts.wantMachine); err != nil {
		return err
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing raw memory bounds sequence % x", opts.label, wantCode)
		}
	}
	for _, forbidCode := range opts.forbidCode {
		if len(forbidCode) > 0 && bytes.Contains(data, forbidCode) {
			return fmt.Errorf(
				"%s contains forbidden raw memory bounds sequence % x",
				opts.label,
				forbidCode,
			)
		}
	}
	return nil
}

func checkRawPointerSlotExecutableSmoke(
	opts rawPointerSlotExecutableSmokeOptions,
	deps RuntimeSmokeDeps,
) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+opts.stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, opts.stem+".tetra")
	outPath := filepath.Join(tmpDir, opts.stem)
	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let slot: ptr = core.alloc_bytes(4)
        let payload: ptr = core.alloc_bytes(4)
        let stored: ptr = core.store_ptr(slot, payload, mem)
        let loaded: ptr = core.load_ptr(slot, mem)
        return 0
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, opts.target); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if err := checkRuntimeSmokeELF(data, opts.label, opts.wantClass, opts.wantMachine); err != nil {
		return err
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing raw pointer slot sequence % x", opts.label, wantCode)
		}
	}
	for _, forbidCode := range opts.forbidCode {
		if len(forbidCode) > 0 && bytes.Contains(data, forbidCode) {
			return fmt.Errorf(
				"%s contains forbidden raw pointer slot sequence % x",
				opts.label,
				forbidCode,
			)
		}
	}
	return nil
}

func checkRawPointerOffsetSlotExecutableSmoke(
	opts rawPointerSlotExecutableSmokeOptions,
	deps RuntimeSmokeDeps,
) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+opts.stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, opts.stem+".tetra")
	outPath := filepath.Join(tmpDir, opts.stem)
	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let slot: ptr = core.alloc_bytes(8)
        let payload: ptr = core.alloc_bytes(4)
        let stored: ptr = core.store_ptr(core.ptr_add(slot, 3, mem), payload, mem)
        let loaded: ptr = core.load_ptr(core.ptr_add(slot, 3, mem), mem)
        return 0
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, opts.target); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if err := checkRuntimeSmokeELF(data, opts.label, opts.wantClass, opts.wantMachine); err != nil {
		return err
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf(
				"%s missing raw pointer offset slot sequence % x",
				opts.label,
				wantCode,
			)
		}
	}
	for _, forbidCode := range opts.forbidCode {
		if len(forbidCode) > 0 && bytes.Contains(data, forbidCode) {
			return fmt.Errorf(
				"%s contains forbidden raw pointer offset slot sequence % x",
				opts.label,
				forbidCode,
			)
		}
	}
	return nil
}

func checkNetworkingLifecycleRuntimeSmoke(
	opts networkingLifecycleRuntimeSmokeOptions,
	deps RuntimeSmokeDeps,
) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+opts.stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, opts.stem+".tetra")
	outPath := filepath.Join(tmpDir, opts.stem)
	src := `
func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let cap: cap.io = core.cap_io()
        let server: Int = core.net_socket_tcp4(cap)
        let client: Int = core.net_socket_tcp4(cap)
        if server < 0 || client < 0:
            return 11
        var buf: []u8 = core.make_u8(4)
        buf[0] = 80
        buf[1] = 73
        buf[2] = 78
        buf[3] = 71
        let bind_status: Int = core.net_bind_tcp4_loopback(server, 0, cap)
        let listen_status: Int = core.net_listen(server, 8, cap)
        let connect_status: Int = core.net_connect_tcp4_loopback(client, 0, cap)
        let accepted: Int = core.net_accept4(server, 0, cap)
        let written: Int = core.net_write(client, buf, 0, 1, cap)
        let read_status: Int = core.net_read(client, buf, 0, 1, cap)
        let sent: Int = core.net_send(client, buf, 0, 1, cap)
        let recv_status: Int = core.net_recv(client, buf, 0, 1, cap)
        let nb: Int = core.net_set_nonblocking(server, cap)
        let reuse: Int = core.net_set_reuseport(server, cap)
        let nodelay: Int = core.net_set_tcp_nodelay(client, cap)
        let epfd: Int = core.net_epoll_create(cap)
        let add_read: Int = core.net_epoll_ctl_add_read(epfd, server, cap)
        let mod_read: Int = core.net_epoll_ctl_mod_read(epfd, server, cap)
        let mod_rw: Int = core.net_epoll_ctl_mod_read_write(epfd, server, cap)
        let del_read: Int = core.net_epoll_ctl_delete(epfd, server, cap)
        let add_rw: Int = core.net_epoll_ctl_add_read_write(epfd, server, cap)
        let del_rw: Int = core.net_epoll_ctl_delete(epfd, server, cap)
        let wait_one: Int = core.net_epoll_wait_one(epfd, 0, cap)
        var event: []i32 = core.make_i32(2)
        let wait_into: Int = core.net_epoll_wait_one_into(epfd, event, 0, cap)
        let epfd_closed: Int = core.net_close(epfd, cap)
        let client_closed: Int = core.net_close(client, cap)
        let server_closed: Int = core.net_close(server, cap)
        if bind_status == 999 || listen_status == 999 || connect_status == 999 || accepted == 999:
            return 12
        if written == 999 || read_status == 999 || sent == 999 || recv_status == 999:
            return 13
        if nb < 0:
            return 14
        if reuse == 999 || nodelay == 999:
            return 15
        if epfd == 999 || add_read == 999 || mod_read == 999 || mod_rw == 999:
            return 16
        if del_read == 999 || add_rw == 999 || del_rw == 999:
            return 17
        if wait_one == 999 || wait_into == 999 || epfd_closed == 999:
            return 18
        if client_closed == 999 || server_closed == 999:
            return 19
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, opts.target); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if err := checkRuntimeSmokeELF(data, opts.label, opts.wantClass, opts.wantMachine); err != nil {
		return err
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf(
				"%s missing target networking syscall sequence % x",
				opts.label,
				wantCode,
			)
		}
	}
	if len(opts.forbidCode) > 0 && bytes.Contains(data, opts.forbidCode) {
		return fmt.Errorf("%s contains forbidden syscall sequence % x", opts.label, opts.forbidCode)
	}
	return nil
}

func checkIslandFreeExecutableSmoke(
	opts islandFreeExecutableSmokeOptions,
	deps RuntimeSmokeDeps,
) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+opts.stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, opts.stem+".tetra")
	src := `fun main(): i32 uses alloc, islands, mem {
  var out: i32 = 0
  island(64) as isl {
    var xs: []u16 = core.island_make_u16(isl, 2)
    xs[0] = 40
    xs[1] = 2
    out = xs[0] + xs[1]
  }
  return out
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	normalPath := filepath.Join(tmpDir, opts.stem)
	if err := buildExecutableWithOptions(
		deps,
		srcPath,
		normalPath,
		opts.target,
		RuntimeBuildOptions{},
	); err != nil {
		return err
	}
	if err := checkIslandFreeExecutableBytes(
		normalPath,
		opts.label,
		opts,
		opts.wantCode,
		opts.forbidCode,
	); err != nil {
		return err
	}

	debugPath := filepath.Join(tmpDir, opts.stem+"_debug")
	if err := buildExecutableWithOptions(
		deps,
		srcPath,
		debugPath,
		opts.target,
		RuntimeBuildOptions{IslandsDebug: true},
	); err != nil {
		return err
	}
	return checkIslandFreeExecutableBytes(
		debugPath,
		opts.label+" debug",
		opts,
		opts.wantDebugCode,
		opts.forbidDebugCode,
	)
}

func checkIslandFreeExecutableBytes(
	path string,
	label string,
	opts islandFreeExecutableSmokeOptions,
	wantCodes [][]byte,
	forbidCodes [][]byte,
) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := checkRuntimeSmokeELF(data, label, opts.wantClass, opts.wantMachine); err != nil {
		return err
	}
	for _, wantCode := range wantCodes {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing target island/free sequence % x", label, wantCode)
		}
	}
	for _, forbidCode := range forbidCodes {
		if len(forbidCode) > 0 && bytes.Contains(data, forbidCode) {
			return fmt.Errorf("%s contains forbidden island/free sequence % x", label, forbidCode)
		}
	}
	return nil
}

func checkRuntimeSmokeELF(data []byte, label string, wantClass byte, wantMachine uint16) error {
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("%s output is not an ELF executable", label)
	}
	if data[4] != wantClass {
		return fmt.Errorf("%s ELF class = %d, want %d", label, data[4], wantClass)
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != wantMachine {
		return fmt.Errorf("%s ELF machine = %#x, want %#x", label, machine, wantMachine)
	}
	return nil
}

// ---- target_models.go ----

func CheckX86TargetModel(tgt ctarget.Target) error {
	if tgt.Triple != "linux-x86" || tgt.OS != ctarget.OSLinux || tgt.Arch != ctarget.ArchX86 ||
		tgt.ABI != ctarget.ABI386SysV {
		return fmt.Errorf(
			"x86 identity = triple=%s os=%s arch=%s abi=%s, want linux-x86/linux/x86/i386-sysv",
			tgt.Triple,
			tgt.OS,
			tgt.Arch,
			tgt.ABI,
		)
	}
	if tgt.DataModel != ctarget.DataModelILP32 || tgt.Format != ctarget.FormatELF ||
		tgt.Endian != ctarget.EndianLittle {
		return fmt.Errorf(
			"x86 platform = model=%s format=%s endian=%s, want ilp32/elf/little",
			tgt.DataModel,
			tgt.Format,
			tgt.Endian,
		)
	}
	if tgt.PointerWidthBits != 32 || tgt.NativeIntWidthBits != 32 || tgt.RegisterWidthBits != 32 ||
		tgt.StackAlignmentBytes != 16 ||
		tgt.MaxAtomicWidthBits != 32 {
		return fmt.Errorf(
			"x86 widths = ptr=%d native=%d reg=%d stack=%d atomic=%d, want 32/32/32/16/32",
			tgt.PointerWidthBits,
			tgt.NativeIntWidthBits,
			tgt.RegisterWidthBits,
			tgt.StackAlignmentBytes,
			tgt.MaxAtomicWidthBits,
		)
	}
	for _, scalar := range []struct {
		name  string
		size  int
		align int
	}{
		{name: "ptr", size: 4, align: 4},
		{name: "usize", size: 4, align: 4},
		{name: "c_long", size: 4, align: 4},
		{name: "i64", size: 8, align: 4},
	} {
		if err := expectTargetScalarLayout(tgt, scalar.name, scalar.size, scalar.align); err != nil {
			return err
		}
	}
	if _, err := tgt.AtomicLayout(64); err == nil {
		return fmt.Errorf("x86 accepted 64-bit lock-free atomic without a CPU feature model")
	}
	return nil
}

func CheckX64TargetModel(tgt ctarget.Target) error {
	if tgt.Arch != ctarget.ArchX64 || tgt.PointerWidthBits != 64 || tgt.NativeIntWidthBits != 64 ||
		tgt.RegisterWidthBits != 64 ||
		tgt.StackAlignmentBytes != 16 ||
		tgt.MaxAtomicWidthBits != 64 {
		return fmt.Errorf(
			"x64 widths = arch=%s ptr=%d native=%d reg=%d stack=%d atomic=%d, want x64/64/64/64/16/64",
			tgt.Arch,
			tgt.PointerWidthBits,
			tgt.NativeIntWidthBits,
			tgt.RegisterWidthBits,
			tgt.StackAlignmentBytes,
			tgt.MaxAtomicWidthBits,
		)
	}
	if tgt.Endian != ctarget.EndianLittle {
		return fmt.Errorf("x64 endian = %s, want little", tgt.Endian)
	}
	if err := expectTargetScalarLayout(tgt, "ptr", 8, 8); err != nil {
		return err
	}
	if err := expectTargetScalarLayout(tgt, "usize", 8, 8); err != nil {
		return err
	}
	switch tgt.ABI {
	case ctarget.ABISysV:
		if tgt.DataModel != ctarget.DataModelLP64 ||
			tgt.Format != ctarget.FormatELF && tgt.Format != ctarget.FormatMachO {
			return fmt.Errorf(
				"x64 SysV platform = model=%s format=%s, want lp64/elf-or-macho",
				tgt.DataModel,
				tgt.Format,
			)
		}
		if err := expectTargetScalarLayout(tgt, "c_long", 8, 8); err != nil {
			return err
		}
	case ctarget.ABIWin64:
		if tgt.DataModel != ctarget.DataModelLLP64 || tgt.Format != ctarget.FormatPE {
			return fmt.Errorf(
				"x64 Win64 platform = model=%s format=%s, want llp64/pe",
				tgt.DataModel,
				tgt.Format,
			)
		}
		if err := expectTargetScalarLayout(tgt, "c_long", 4, 4); err != nil {
			return err
		}
	default:
		return fmt.Errorf("x64 unsupported ABI %s", tgt.ABI)
	}
	return nil
}

func CheckX32TargetModel(tgt ctarget.Target) error {
	if tgt.Triple != "linux-x32" || tgt.OS != ctarget.OSLinux || tgt.Arch != ctarget.ArchX64 ||
		tgt.ABI != ctarget.ABIX32SysV {
		return fmt.Errorf(
			"x32 identity = triple=%s os=%s arch=%s abi=%s, want linux-x32/linux/x64/x32-sysv",
			tgt.Triple,
			tgt.OS,
			tgt.Arch,
			tgt.ABI,
		)
	}
	if tgt.DataModel != ctarget.DataModelX32 || tgt.Format != ctarget.FormatELF ||
		tgt.Endian != ctarget.EndianLittle {
		return fmt.Errorf(
			"x32 platform = model=%s format=%s endian=%s, want x32/elf/little",
			tgt.DataModel,
			tgt.Format,
			tgt.Endian,
		)
	}
	if tgt.PointerWidthBits != 32 || tgt.NativeIntWidthBits != 32 || tgt.RegisterWidthBits != 64 ||
		tgt.StackAlignmentBytes != 16 ||
		tgt.MaxAtomicWidthBits != 64 {
		return fmt.Errorf(
			"x32 widths = ptr=%d native=%d reg=%d stack=%d atomic=%d, want 32/32/64/16/64",
			tgt.PointerWidthBits,
			tgt.NativeIntWidthBits,
			tgt.RegisterWidthBits,
			tgt.StackAlignmentBytes,
			tgt.MaxAtomicWidthBits,
		)
	}
	if err := expectTargetScalarLayout(tgt, "ptr", 4, 4); err != nil {
		return err
	}
	if err := expectTargetScalarLayout(tgt, "usize", 4, 4); err != nil {
		return err
	}
	if err := expectTargetScalarLayout(tgt, "isize", 4, 4); err != nil {
		return err
	}
	if err := expectTargetScalarLayout(tgt, "size_t", 4, 4); err != nil {
		return err
	}
	if err := expectTargetScalarLayout(tgt, "i64", 8, 8); err != nil {
		return err
	}
	x86, err := ctarget.Parse("x86")
	if err != nil {
		return err
	}
	if x86.Arch == tgt.Arch || x86.RegisterWidthBits == tgt.RegisterWidthBits ||
		x86.MaxAtomicWidthBits == tgt.MaxAtomicWidthBits {
		return fmt.Errorf(
			"x32 collapsed into x86: x86 arch=%s reg=%d atomic=%d, x32 arch=%s reg=%d atomic=%d",
			x86.Arch,
			x86.RegisterWidthBits,
			x86.MaxAtomicWidthBits,
			tgt.Arch,
			tgt.RegisterWidthBits,
			tgt.MaxAtomicWidthBits,
		)
	}
	x64, err := ctarget.Parse("x64")
	if err != nil {
		return err
	}
	if x64.PointerWidthBits == tgt.PointerWidthBits ||
		x64.NativeIntWidthBits == tgt.NativeIntWidthBits ||
		x64.ABI == tgt.ABI {
		return fmt.Errorf(
			"x32 collapsed into x64: x64 ptr=%d native=%d abi=%s, x32 ptr=%d native=%d abi=%s",
			x64.PointerWidthBits,
			x64.NativeIntWidthBits,
			x64.ABI,
			tgt.PointerWidthBits,
			tgt.NativeIntWidthBits,
			tgt.ABI,
		)
	}
	return nil
}

func ExpectTargetScalarLayout(tgt ctarget.Target, name string, size int, align int) error {
	return expectTargetScalarLayout(tgt, name, size, align)
}

func expectTargetScalarLayout(tgt ctarget.Target, name string, size int, align int) error {
	layout, ok := tgt.ScalarLayout(name)
	if !ok {
		return fmt.Errorf("%s missing scalar layout %s", tgt.Triple, name)
	}
	if layout.SizeBytes != size || layout.AlignBytes != align || layout.ABIBytes != size {
		return fmt.Errorf(
			"%s scalar %s layout = size=%d align=%d abi=%d, want %d/%d/%d",
			tgt.Triple,
			name,
			layout.SizeBytes,
			layout.AlignBytes,
			layout.ABIBytes,
			size,
			align,
			size,
		)
	}
	return nil
}

// ---- target_suite.go ----

type TargetCheckRunner func(tgt ctarget.Target) []Check

type TargetCheckRunners struct {
	X86  TargetCheckRunner
	X32  TargetCheckRunner
	X64  TargetCheckRunner
	WASM TargetCheckRunner
}

func RunTargetChecks(targetName string, runners TargetCheckRunners) ([]Check, error) {
	tgt, err := ctarget.Parse(targetName)
	if err != nil {
		return nil, err
	}
	switch {
	case tgt.Arch == ctarget.ArchX86 && tgt.ABI == ctarget.ABI386SysV:
		return runTargetCheckRunner("x86", runners.X86, tgt)
	case tgt.Arch == ctarget.ArchX64 && tgt.ABI == ctarget.ABIX32SysV:
		return runTargetCheckRunner("x32", runners.X32, tgt)
	case tgt.Arch == ctarget.ArchX64:
		return runTargetCheckRunner("x64", runners.X64, tgt)
	case tgt.Arch == ctarget.ArchWASM32:
		return runTargetCheckRunner("wasm", runners.WASM, tgt)
	default:
		return nil, UnsupportedTargetError(tgt.Triple)
	}
}

func X64CheckPrefix(tgt ctarget.Target) string {
	switch tgt.Triple {
	case "windows-x64", "macos-x64":
		return tgt.Triple
	default:
		return "x64"
	}
}

func runTargetCheckRunner(
	name string,
	runner TargetCheckRunner,
	tgt ctarget.Target,
) ([]Check, error) {
	if runner == nil {
		return nil, fmt.Errorf("missing ABI suite runner for %s", name)
	}
	return runner(tgt), nil
}

// ---- verification.go ----

const (
	VerificationSchemaV1  = "tetra.abi.verification.v1"
	VerificationScopeP211 = "p21.1_abi_verification"
)

const (
	VerificationTaskCorpus           = "abi_test_corpus"
	VerificationTaskAggregateReturns = "struct_enum_slice_string_return_validation"
	VerificationTaskCallBoundary     = "call_boundary_validation"
	VerificationTaskFFIReprC         = "ffi_repr_c_tests"
)

type VerificationReport struct {
	Schema    string
	Scope     string
	Claims    []string
	NonClaims []string
	Targets   []VerificationTargetRow
	Tasks     []VerificationTaskRow
}

type VerificationTargetRow struct {
	Target       string
	ABI          string
	Status       string
	TaskCoverage []string
	Evidence     []string
	Claims       []string
}

type VerificationTaskRow struct {
	ID       string
	Name     string
	Targets  []string
	Evidence []string
}

func BuildP21VerificationReport() VerificationReport {
	targets := P21VerificationTargets()
	tasks := P21VerificationTaskIDs()
	return VerificationReport{
		Schema: VerificationSchemaV1,
		Scope:  VerificationScopeP211,
		Claims: []string{
			("ABI verification v1 covers declared target metadata, classifier/" +
				"layout rows, backend call-boundary metadata, and repr(C) aggregate " +
				"export gates"),
			"wasm32 targets use compiler-owned i32 slot ABI metadata validation",
			"native exported aggregate FFI boundaries require explicit repr(C)",
		},
		NonClaims: []string{
			"no runtime execution claim for build-only or wasm targets",
			"no C ABI claim for default structs",
			"no native C aggregate ABI claim for wasm targets",
			"no performance claim",
			"no safe-program semantics change",
		},
		Targets: []VerificationTargetRow{
			{
				Target:       "linux-x64",
				ABI:          "SysV x86_64",
				Status:       "supported_native",
				TaskCoverage: append([]string{}, tasks...),
				Evidence: []string{
					("compiler/compiler_evidence_gates.go: x64 target model, SysV " +
						"classifier, varargs and aggregates, c_int/c_uint FFI object smokes"),
					"compiler/compiler_suite_test.go: native exported aggregate repr(C) diagnostics",
				},
				Claims: []string{
					"linux-x64 SysV classifier/layout/call-boundary evidence is covered by the ABI suite",
				},
			},
			{
				Target:       "linux-x86",
				ABI:          "i386 SysV",
				Status:       "build_only",
				TaskCoverage: append([]string{}, tasks...),
				Evidence: []string{
					("compiler/compiler_evidence_gates.go: x86 target model, i386 " +
						"SysV classifier, varargs/sret, pointer/c_int/c_uint/ILP32 FFI object " +
						"smokes"),
					"compiler/compiler_suite_test.go: x86 pointer and repr(C) aggregate diagnostics",
				},
				Claims: []string{
					("linux-x86 i386 SysV classifier/layout/call-boundary evidence is " +
						"covered by compile and object checks"),
				},
			},
			{
				Target:       "linux-x32",
				ABI:          "x32 SysV",
				Status:       "build_only",
				TaskCoverage: append([]string{}, tasks...),
				Evidence: []string{
					("compiler/compiler_evidence_gates.go: x32 target model, x32 SysV " +
						"classifier, varargs/aggregates, pointer/c_int/c_uint/ILP32 FFI object " +
						"smokes"),
					"compiler/compiler_suite_test.go: x32 pointer and repr(C) aggregate diagnostics",
				},
				Claims: []string{
					("linux-x32 SysV classifier/layout/call-boundary evidence is " +
						"covered by compile and object checks"),
				},
			},
			{
				Target:       "macos-x64",
				ABI:          "SysV x86_64 Mach-O",
				Status:       "supported_native",
				TaskCoverage: append([]string{}, tasks...),
				Evidence: []string{
					("compiler/compiler_evidence_gates.go: macos-x64 target model, " +
						"SysV classifier, varargs/aggregates, object ABI smoke"),
					"compiler/compiler_suite_test.go: native exported aggregate repr(C) diagnostics",
				},
				Claims: []string{
					("macos-x64 SysV classifier/layout/call-boundary evidence is " +
						"covered by object ABI smoke and classifier rows"),
				},
			},
			{
				Target:       "windows-x64",
				ABI:          "Win64",
				Status:       "supported_native",
				TaskCoverage: append([]string{}, tasks...),
				Evidence: []string{
					("compiler/compiler_evidence_gates.go: windows-x64 target model, " +
						"Win64 classifier, varargs/aggregates, object ABI smoke"),
					"compiler/compiler_suite_test.go: native exported aggregate repr(C) diagnostics",
				},
				Claims: []string{
					("windows-x64 Win64 classifier/layout/call-boundary evidence is " +
						"covered by object ABI smoke and classifier rows"),
				},
			},
			{
				Target:       "wasm32-wasi",
				ABI:          "WASI i32 slot ABI",
				Status:       "supported_wasm_artifact",
				TaskCoverage: append([]string{}, tasks...),
				Evidence: []string{
					("compiler/abi_wasm.go: wasm32-wasi target model, slot ABI " +
						"metadata, aggregate return layout, and call boundary validation"),
					"compiler/internal/backend/wasm32_wasi/codegen.go: IRCall arg/return slot metadata validation",
				},
				Claims: []string{
					("wasm32-wasi uses compiler-owned i32 slot ABI metadata for " +
						"aggregate returns and call boundaries"),
				},
			},
			{
				Target:       "wasm32-web",
				ABI:          "Web i32 slot ABI",
				Status:       "supported_wasm_artifact",
				TaskCoverage: append([]string{}, tasks...),
				Evidence: []string{
					("compiler/abi_wasm.go: wasm32-web target model, slot ABI " +
						"metadata, aggregate return layout, and call boundary validation"),
					("compiler/internal/backend/wasm32_web/codegen.go: IRCall arg/" +
						"return slot metadata validation including Surface imports"),
				},
				Claims: []string{
					("wasm32-web uses compiler-owned i32 slot ABI metadata for " +
						"aggregate returns and call boundaries"),
				},
			},
		},
		Tasks: []VerificationTaskRow{
			{
				ID:      VerificationTaskCorpus,
				Name:    "ABI test corpus",
				Targets: append([]string{}, targets...),
				Evidence: []string{
					"compiler/compiler_evidence_gates.go",
					"compiler/compiler_suite_test.go",
				},
			},
			{
				ID:      VerificationTaskAggregateReturns,
				Name:    "Struct/enum/slice/String return validation",
				Targets: append([]string{}, targets...),
				Evidence: []string{
					"compiler/compiler_evidence_gates.go native aggregate classifier checks",
					"compiler/abi_wasm.go wasm aggregate return layout checks",
				},
			},
			{
				ID:      VerificationTaskCallBoundary,
				Name:    "Call boundary validation",
				Targets: append([]string{}, targets...),
				Evidence: []string{
					"compiler/internal/backend/wasm32_wasi/codegen.go",
					"compiler/internal/backend/wasm32_web/codegen.go",
					"compiler/internal/backend/x64abi/classifier.go",
					"compiler/internal/backend/x86abi/classifier.go",
				},
			},
			{
				ID:      VerificationTaskFFIReprC,
				Name:    "FFI repr(C) tests",
				Targets: append([]string{}, targets...),
				Evidence: []string{
					"compiler/compiler_build_runtime.go",
					"compiler/compiler_suite_test.go",
					"compiler/internal/semantics/semantics_suite_test.go",
				},
			},
		},
	}
}

func ValidateP21VerificationReport(report VerificationReport) error {
	if report.Schema != VerificationSchemaV1 {
		return fmt.Errorf(
			"ABI verification report schema = %q, want %q",
			report.Schema,
			VerificationSchemaV1,
		)
	}
	if report.Scope != VerificationScopeP211 {
		return fmt.Errorf(
			"ABI verification report scope = %q, want %q",
			report.Scope,
			VerificationScopeP211,
		)
	}
	if err := validateStrings("claim", report.Claims, true); err != nil {
		return err
	}
	targetRows := map[string]VerificationTargetRow{}
	for _, row := range report.Targets {
		if strings.TrimSpace(row.Target) == "" || strings.TrimSpace(row.ABI) == "" ||
			strings.TrimSpace(row.Status) == "" {
			return fmt.Errorf("ABI verification target row missing required metadata: %#v", row)
		}
		if _, exists := targetRows[row.Target]; exists {
			return fmt.Errorf("duplicate ABI verification target %s", row.Target)
		}
		if err := validateStrings("target "+row.Target+" evidence", row.Evidence, false); err != nil {
			return err
		}
		if err := validateStrings("target "+row.Target+" claim", row.Claims, true); err != nil {
			return err
		}
		for _, task := range P21VerificationTaskIDs() {
			if !stringSliceHas(row.TaskCoverage, task) {
				return fmt.Errorf("target %s missing task %s coverage", row.Target, task)
			}
		}
		targetRows[row.Target] = row
	}
	for _, target := range P21VerificationTargets() {
		if _, ok := targetRows[target]; !ok {
			return fmt.Errorf("missing target %s in ABI verification report", target)
		}
	}
	taskRows := map[string]VerificationTaskRow{}
	for _, row := range report.Tasks {
		if strings.TrimSpace(row.ID) == "" || strings.TrimSpace(row.Name) == "" {
			return fmt.Errorf("ABI verification task row missing required metadata: %#v", row)
		}
		if _, exists := taskRows[row.ID]; exists {
			return fmt.Errorf("duplicate ABI verification task %s", row.ID)
		}
		if err := validateStrings("task "+row.ID+" evidence", row.Evidence, false); err != nil {
			return err
		}
		for _, target := range P21VerificationTargets() {
			if !stringSliceHas(row.Targets, target) {
				return fmt.Errorf("task %s missing target %s coverage", row.ID, target)
			}
		}
		taskRows[row.ID] = row
	}
	for _, task := range P21VerificationTaskIDs() {
		if _, ok := taskRows[task]; !ok {
			return fmt.Errorf("missing task %s in ABI verification report", task)
		}
	}
	for _, nonClaim := range P21VerificationNonClaims() {
		if !stringSliceHas(report.NonClaims, nonClaim) {
			return fmt.Errorf("ABI verification report missing non-claim %q", nonClaim)
		}
	}
	if err := validateStrings("non-claim", report.NonClaims, false); err != nil {
		return err
	}
	return nil
}

func P21VerificationTargets() []string {
	return []string{
		"linux-x64",
		"linux-x86",
		"linux-x32",
		"macos-x64",
		"windows-x64",
		"wasm32-wasi",
		"wasm32-web",
	}
}

func P21VerificationTaskIDs() []string {
	return []string{
		VerificationTaskCorpus,
		VerificationTaskAggregateReturns,
		VerificationTaskCallBoundary,
		VerificationTaskFFIReprC,
	}
}

func P21VerificationNonClaims() []string {
	return []string{
		"no runtime execution claim for build-only or wasm targets",
		"no C ABI claim for default structs",
		"no native C aggregate ABI claim for wasm targets",
		"no performance claim",
		"no safe-program semantics change",
	}
}

func validateStrings(label string, values []string, rejectBroadClaims bool) error {
	if len(values) == 0 {
		return fmt.Errorf("%s list is empty", label)
	}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		lower := strings.ToLower(trimmed)
		if trimmed == "" {
			return fmt.Errorf("%s contains an empty entry", label)
		}
		for _, forbidden := range []string{"placeholder", "todo", "mock"} {
			if strings.Contains(lower, forbidden) {
				return fmt.Errorf("%s contains %s evidence/claim: %q", label, forbidden, value)
			}
		}
		if !rejectBroadClaims {
			continue
		}
		if strings.Contains(lower, "full runtime") ||
			strings.Contains(lower, "runtime execution verified") {
			return fmt.Errorf("%s contains unsupported runtime execution claim: %q", label, value)
		}
		if strings.Contains(lower, "performance") {
			return fmt.Errorf("%s contains unsupported performance claim: %q", label, value)
		}
		if strings.Contains(lower, "default structs") && strings.Contains(lower, "c abi") {
			return fmt.Errorf(
				"%s contains unsupported default structs C ABI claim: %q",
				label,
				value,
			)
		}
		if strings.Contains(lower, "wasm") && strings.Contains(lower, "native c aggregate abi") {
			return fmt.Errorf(
				"%s contains unsupported wasm native C aggregate ABI claim: %q",
				label,
				value,
			)
		}
	}
	return nil
}

func stringSliceHas(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

// ---- wasm.go ----

func CheckWASMTargetModel(tgt ctarget.Target) error {
	if tgt.Arch != ctarget.ArchWASM32 || tgt.Format != ctarget.FormatWASM ||
		tgt.DataModel != ctarget.DataModelILP32 ||
		tgt.Endian != ctarget.EndianLittle {
		return fmt.Errorf(
			"%s target model = arch=%s format=%s model=%s endian=%s, want wasm32/wasm/ilp32/little",
			tgt.Triple,
			tgt.Arch,
			tgt.Format,
			tgt.DataModel,
			tgt.Endian,
		)
	}
	if tgt.PointerWidthBits != 32 || tgt.NativeIntWidthBits != 32 || tgt.RegisterWidthBits != 32 ||
		tgt.StackAlignmentBytes != 16 ||
		tgt.MaxAtomicWidthBits != 64 {
		return fmt.Errorf(
			"%s widths = ptr=%d native=%d reg=%d stack=%d atomic=%d, want 32/32/32/16/64",
			tgt.Triple,
			tgt.PointerWidthBits,
			tgt.NativeIntWidthBits,
			tgt.RegisterWidthBits,
			tgt.StackAlignmentBytes,
			tgt.MaxAtomicWidthBits,
		)
	}
	switch tgt.Triple {
	case "wasm32-wasi":
		if tgt.OS != ctarget.OSWASI || tgt.ABI != ctarget.ABIWASI ||
			tgt.RunMode != ctarget.RunModeWASIRunner ||
			tgt.ExeExt != ".wasm" {
			return fmt.Errorf(
				"%s identity = os=%s abi=%s run=%s ext=%q, want wasi/wasi/wasi_runner/.wasm",
				tgt.Triple,
				tgt.OS,
				tgt.ABI,
				tgt.RunMode,
				tgt.ExeExt,
			)
		}
	case "wasm32-web":
		if tgt.OS != ctarget.OSWeb || tgt.ABI != ctarget.ABIWeb ||
			tgt.RunMode != ctarget.RunModeWebRunner ||
			tgt.ExeExt != ".wasm" {
			return fmt.Errorf(
				"%s identity = os=%s abi=%s run=%s ext=%q, want web/web/web_runner/.wasm",
				tgt.Triple,
				tgt.OS,
				tgt.ABI,
				tgt.RunMode,
				tgt.ExeExt,
			)
		}
	default:
		return fmt.Errorf("unsupported wasm ABI target %s", tgt.Triple)
	}
	for _, scalar := range []struct {
		name  string
		size  int
		align int
	}{
		{name: "ptr", size: 4, align: 4},
		{name: "fnptr", size: 4, align: 4},
		{name: "usize", size: 4, align: 4},
		{name: "isize", size: 4, align: 4},
		{name: "c_long", size: 4, align: 4},
	} {
		if err := ExpectTargetScalarLayout(tgt, scalar.name, scalar.size, scalar.align); err != nil {
			return err
		}
	}
	return nil
}

func CheckWASMSlotABIMetadata(tgt ctarget.Target) error {
	for _, layoutCase := range []struct {
		name      string
		size      int
		alignment int
	}{
		{name: "ptr", size: 4, alignment: 4},
		{name: "usize", size: 4, alignment: 4},
		{name: "fnptr", size: 4, alignment: 4},
		{name: "string", size: 8, alignment: 4},
	} {
		layout, ok := tgt.ScalarLayout(layoutCase.name)
		if layoutCase.name == "string" {
			str, err := tgt.StringLayout()
			if err != nil {
				return err
			}
			if str.SizeBytes != layoutCase.size || str.AlignBytes != layoutCase.alignment {
				return fmt.Errorf(
					"%s string slot layout = size=%d align=%d, want %d/%d",
					tgt.Triple,
					str.SizeBytes,
					str.AlignBytes,
					layoutCase.size,
					layoutCase.alignment,
				)
			}
			continue
		}
		if !ok {
			return fmt.Errorf("%s missing scalar slot layout %s", tgt.Triple, layoutCase.name)
		}
		if layout.SizeBytes != layoutCase.size || layout.AlignBytes != layoutCase.alignment ||
			layout.ABIBytes != layoutCase.size {
			return fmt.Errorf(
				"%s %s slot layout = %#v, want size/align/abi %d/%d/%d",
				tgt.Triple,
				layoutCase.name,
				layout,
				layoutCase.size,
				layoutCase.alignment,
				layoutCase.size,
			)
		}
	}

	atomic, err := tgt.AtomicPointerLayout()
	if err != nil {
		return err
	}
	if atomic.WidthBits != 32 || atomic.RegisterWidthBits != 32 || !atomic.PointerSized {
		return fmt.Errorf(
			"%s pointer atomic slot layout = %#v, want 32-bit pointer-sized wasm slot",
			tgt.Triple,
			atomic,
		)
	}
	return nil
}

func CheckWASMAggregateReturnLayouts(tgt ctarget.Target) error {
	pair, err := tgt.StructLayout([]ctarget.LayoutField{
		{Name: "raw", Type: "ptr"},
		{Name: "count", Type: "usize"},
	})
	if err != nil {
		return err
	}
	if pair.SizeBytes != 8 || pair.AlignBytes != 4 || len(pair.Fields) != 2 ||
		pair.Fields[1].OffsetBytes != 4 {
		return fmt.Errorf(
			"%s struct return layout = %#v, want 8-byte two-slot pair",
			tgt.Triple,
			pair,
		)
	}
	slice, err := tgt.SliceLayout("u8")
	if err != nil {
		return err
	}
	if slice.SizeBytes != 8 || slice.AlignBytes != 4 || len(slice.Fields) != 2 ||
		slice.Fields[0].Type != "ptr" ||
		slice.Fields[1].OffsetBytes != 4 {
		return fmt.Errorf(
			"%s slice return layout = %#v, want ptr/i32 two-slot view",
			tgt.Triple,
			slice,
		)
	}
	str, err := tgt.StringLayout()
	if err != nil {
		return err
	}
	if str.SizeBytes != slice.SizeBytes || str.AlignBytes != slice.AlignBytes {
		return fmt.Errorf(
			"%s String return layout = %#v, want same layout as []u8 %#v",
			tgt.Triple,
			str,
			slice,
		)
	}
	enum, err := tgt.EnumLayout([]ctarget.EnumCaseLayout{
		{Name: "Empty"},
		{Name: "Text", Payload: []ctarget.LayoutField{{Name: "value", Type: "string"}}},
	})
	if err != nil {
		return err
	}
	if enum.SizeBytes != 12 || enum.AlignBytes != 4 || enum.PayloadOffsetBytes != 4 ||
		enum.PayloadSizeBytes != 8 {
		return fmt.Errorf(
			"%s enum return layout = %#v, want tag plus 8-byte String payload",
			tgt.Triple,
			enum,
		)
	}
	return nil
}

func CheckWASMCallBoundaryValidation(tgt ctarget.Target) error {
	switch tgt.Triple {
	case "wasm32-wasi":
		obj, err := wasm32wasi.CodegenObject(wasmABIValidCallFuncs(), "main")
		if err != nil {
			return err
		}
		if _, err := wasm32wasi.LinkObject(obj); err != nil {
			return err
		}
		_, err = wasm32wasi.CodegenObject(wasmABIMismatchedCallFuncs(), "main")
		if err == nil || !strings.Contains(err.Error(), `call "helper" ABI mismatch`) {
			return fmt.Errorf(
				"%s mismatched call metadata diagnostic = %v, want ABI mismatch",
				tgt.Triple,
				err,
			)
		}
	case "wasm32-web":
		obj, err := wasm32web.CodegenObject(wasmABIValidCallFuncs(), "main")
		if err != nil {
			return err
		}
		if _, err := wasm32web.LinkObject(obj); err != nil {
			return err
		}
		_, err = wasm32web.CodegenObject(wasmABIMismatchedCallFuncs(), "main")
		if err == nil || !strings.Contains(err.Error(), `call "helper" ABI mismatch`) {
			return fmt.Errorf(
				"%s mismatched call metadata diagnostic = %v, want ABI mismatch",
				tgt.Triple,
				err,
			)
		}
	default:
		return fmt.Errorf("unsupported wasm call-boundary target %s", tgt.Triple)
	}
	return nil
}

func CheckWASMFFIReprCBoundaryPolicy(tgt ctarget.Target) error {
	if TargetRequiresExplicitAggregateExportGate(tgt.Triple) {
		return fmt.Errorf(
			"%s unexpectedly requires the native aggregate C ABI export gate",
			tgt.Triple,
		)
	}
	if TargetRequiresExplicitPointerExportGate(tgt.Triple) {
		return fmt.Errorf(
			"%s unexpectedly requires the native pointer C ABI export gate",
			tgt.Triple,
		)
	}
	for _, native := range []string{
		"linux-x86",
		"linux-x64",
		"linux-x32",
		"macos-x64",
		"windows-x64",
	} {
		if !TargetRequiresExplicitAggregateExportGate(native) {
			return fmt.Errorf(
				"native target %s lost explicit repr(C) aggregate export gate",
				native,
			)
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
			return fmt.Errorf("aggregate FFI detector did not recognize %s", typeName)
		}
	}
	return nil
}

func TargetRequiresExplicitAggregateExportGate(target string) bool {
	switch target {
	case "linux-x86", "linux-x64", "linux-x32", "macos-x64", "windows-x64":
		return true
	default:
		return false
	}
}

func TargetRequiresExplicitPointerExportGate(target string) bool {
	switch target {
	case "linux-x86", "linux-x32":
		return true
	default:
		return false
	}
}

func TargetExportedFFIRequiresX32PointerBoundaryGate(target, typeName string) bool {
	return target == "linux-x32" && TargetExportedFFIRequiresPointerBoundaryGate(target, typeName)
}

func TargetExportedFFIRequiresPointerBoundaryGate(target, typeName string) bool {
	if !TargetRequiresExplicitPointerExportGate(target) {
		return false
	}
	normalized := strings.TrimSpace(typeName)
	switch normalized {
	case "fnptr":
		return true
	default:
		return strings.HasPrefix(normalized, "fn(")
	}
}

func TargetExportedFFIRequiresAggregateABI(
	typeName string,
	types map[string]*semantics.TypeInfo,
) bool {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" || typeName == "none" {
		return false
	}
	info, ok := types[typeName]
	if !ok {
		return false
	}
	switch info.Kind {
	case semantics.TypeStruct,
		semantics.TypeArray,
		semantics.TypeSlice,
		semantics.TypeStr,
		semantics.TypeEnum,
		semantics.TypeOptional:
		return true
	default:
		return false
	}
}

func wasmABIValidCallFuncs() []ir.IRFunc {
	return []ir.IRFunc{
		{
			Name:        "helper",
			ParamSlots:  2,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRCall, Name: "helper", ArgSlots: 2, RetSlots: 1},
				{Kind: ir.IRReturn},
			},
		},
	}
}

func wasmABIMismatchedCallFuncs() []ir.IRFunc {
	funcs := wasmABIValidCallFuncs()
	funcs[1].Instrs[2].ArgSlots = 1
	return funcs
}

// ---- x64_runtime.go ----

type RuntimeSmokeDeps struct {
	BuildExecutable            func(srcPath string, outPath string, target string) error
	BuildExecutableWithOptions func(srcPath string, outPath string, target string, opts RuntimeBuildOptions) error
	RunExecutable              func(path string) RuntimeRunResult
	HostGOOS                   string
	HostGOARCH                 string
}

type RuntimeBuildOptions struct {
	IslandsDebug bool
}

type RuntimeRunResult struct {
	ExitCode int
	Output   string
	TimedOut bool
	Err      error
}

func CheckSourceNativeScalarDiagnostics(tgt ctarget.Target, deps FFICheckDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-source-native-scalar-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	var cases []struct {
		name string
		src  string
	}
	if tgt.Triple == "linux-x86" || tgt.Triple == "linux-x32" {
		cases = []struct {
			name string
			src  string
		}{
			{
				name: "u32_param",
				src:  "func native_probe(n: u32) -> Int:\n    return 0\n",
			},
			{
				name: "u64_param",
				src:  "func native_probe(n: u64) -> Int:\n    return 0\n",
			},
			{
				name: "f64_return",
				src:  "func native_probe() -> f64:\n    return 0\n",
			},
		}
	} else {
		cases = []struct {
			name string
			src  string
		}{
			{
				name: "usize_param",
				src:  "func native_probe(n: usize) -> Int:\n    return 0\n",
			},
			{
				name: "size_t_param",
				src:  "func native_probe(n: size_t) -> Int:\n    return 0\n",
			},
			{
				name: "native_int_return",
				src:  "func native_probe() -> native_int:\n    return 0\n",
			},
			{
				name: "c_long_return",
				src:  "func native_probe() -> c_long:\n    return 0\n",
			},
		}
	}
	for _, tc := range cases {
		srcPath := filepath.Join(tmpDir, tgt.Triple+"_"+tc.name+".tetra")
		outPath := filepath.Join(tmpDir, tgt.Triple+"_"+tc.name+".tobj")
		if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
			return err
		}
		err := buildLibrary(deps, srcPath, outPath, tgt.Triple)
		if err == nil {
			return fmt.Errorf(
				"%s accepted source-level target-layout scalar in %s",
				tgt.Triple,
				tc.name,
			)
		}
		for _, want := range []string{
			"target-layout scalar type",
			"not supported in source-level Tetra yet",
			"native-int/codegen support",
		} {
			if !strings.Contains(err.Error(), want) {
				return fmt.Errorf(
					"%s source native scalar diagnostic for %s = %q, want %q",
					tgt.Triple,
					tc.name,
					err.Error(),
					want,
				)
			}
		}
		if strings.Contains(err.Error(), "unknown type") {
			return fmt.Errorf(
				"%s source native scalar diagnostic for %s fell back to unknown type: %q",
				tgt.Triple,
				tc.name,
				err.Error(),
			)
		}
		if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
			return fmt.Errorf(
				"%s source native scalar wrote object %s (stat err=%v)",
				tgt.Triple,
				outPath,
				statErr,
			)
		}
	}
	return nil
}

func CheckX64PlatformObjectABISmoke(tgt ctarget.Target, deps FFICheckDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-x64-platform-object-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	stem := strings.ReplaceAll(tgt.Triple, "-", "_")
	srcPath := filepath.Join(tmpDir, stem+"_abi_smoke.tetra")
	outPath := filepath.Join(tmpDir, stem+"_abi_smoke.tobj")
	src := "@export(\"ffi_say_i32\")\nfun say(): i32 uses io {\n  print(\"" + tgt.Triple + " abi\\n\")\n  return 0\n}\n"
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
		return fmt.Errorf("target mismatch: got %q want %s", obj.Target, tgt.Triple)
	}
	if !strings.Contains(string(obj.Data), tgt.Triple+" abi\n") {
		return fmt.Errorf(
			"%s object data missing ABI smoke literal: %q",
			tgt.Triple,
			string(obj.Data),
		)
	}
	if !ObjectHasSymbolSignature(obj, "ffi_say_i32", 0, 1) {
		return fmt.Errorf(
			"%s object missing scalar exported ffi_say_i32 symbol: %#v",
			tgt.Triple,
			obj.Symbols,
		)
	}
	if !ObjectHasRelocKind(obj, ObjectRelocDataDisp32) {
		return fmt.Errorf(
			"%s object missing data displacement relocation: %#v",
			tgt.Triple,
			obj.Relocs,
		)
	}
	switch tgt.Triple {
	case "macos-x64":
		if ObjectHasRelocKind(obj, ObjectRelocIATDisp32) {
			return fmt.Errorf(
				"macos-x64 object unexpectedly has Windows IAT reloc: %#v",
				obj.Relocs,
			)
		}
	case "windows-x64":
		for _, name := range []string{"kernel32.GetStdHandle", "kernel32.WriteFile"} {
			if !ObjectHasReloc(obj, ObjectRelocIATDisp32, name) {
				return fmt.Errorf(
					"windows-x64 object missing IAT relocation %q: %#v",
					name,
					obj.Relocs,
				)
			}
		}
	default:
		return fmt.Errorf("x64 platform object ABI smoke does not cover %s", tgt.Triple)
	}
	return nil
}

func CheckX64PointerFFIRegressionSmoke(deps FFICheckDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-x64-pointer-ffi-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "x64_pointer_ffi.tetra")
	outPath := filepath.Join(tmpDir, "x64_pointer_ffi.tobj")
	src := `@export("ffi_ptr_param_c")
func ffi_ptr_param(p: ptr) -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildLibrary(deps, srcPath, outPath, "linux-x64"); err != nil {
		return err
	}
	obj, err := readObject(deps, outPath)
	if err != nil {
		return err
	}
	if obj.Target != "linux-x64" {
		return fmt.Errorf("x64 pointer FFI object target = %q, want linux-x64", obj.Target)
	}
	if !ObjectHasSymbolSignature(obj, "ffi_ptr_param_c", 1, 1) {
		return fmt.Errorf(
			"x64 pointer FFI object missing exported ffi_ptr_param_c(1)->1 symbol: %#v",
			obj.Symbols,
		)
	}
	return nil
}

func CheckX64FilesystemSchedulerCompositionSmoke(deps RuntimeSmokeDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-x64-filesystem-scheduler-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "x64_filesystem_scheduler.tetra")
	outPath := filepath.Join(tmpDir, "x64-filesystem-scheduler")
	src := `
func worker() -> Int:
    return 41

func main() -> Int
uses capability, io, runtime:
    unsafe:
        let cap: cap.io = core.cap_io()
        if !core.fs_exists("../README.md", cap):
            return 11
        if core.fs_exists("__tetra_missing_fs_exists_smoke__", cap):
            return 12
    let task: task.i32 = core.task_spawn_i32("worker")
    let value: Int = core.task_join_i32(task)
    if value != 41:
        return value
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, "linux-x64"); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	return checkX64ELFExecutable(data, "x64 filesystem scheduler")
}

func CheckX64NetworkingRuntimeSmoke(deps RuntimeSmokeDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-x64-networking-runtime-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "x64_networking_runtime.tetra")
	outPath := filepath.Join(tmpDir, "x64-networking-runtime")
	src := `
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        if fd < 0:
            return 11
        let nb: Int = core.net_set_nonblocking(fd, cap)
        let closed: Int = core.net_close(fd, cap)
        if nb < 0:
            return 12
        if closed != 0:
            return 13
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, "linux-x64"); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if err := checkX64ELFExecutable(data, "x64 networking runtime"); err != nil {
		return err
	}
	if hostGOOS(deps) != "linux" || hostGOARCH(deps) != "amd64" {
		return nil
	}
	result := runExecutable(deps, outPath)
	if result.TimedOut {
		return fmt.Errorf("x64 networking runtime executable timed out: %q", result.Output)
	}
	if result.Err != nil {
		return fmt.Errorf("run x64 networking runtime: %w output=%q", result.Err, result.Output)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf(
			"x64 networking runtime exit=%d output=%q, want 0",
			result.ExitCode,
			result.Output,
		)
	}
	if result.Output != "" {
		return fmt.Errorf("x64 networking runtime output=%q, want empty", result.Output)
	}
	return nil
}

func CheckX64SchedulerRestrictionRegressionSmoke(deps RuntimeSmokeDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-x64-scheduler-regression-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "x64_scheduler_regression.tetra")
	outPath := filepath.Join(tmpDir, "x64-scheduler-regression")
	src := `
enum TaskErr:
    case boom(Int, Int)
    case stopped

func left() -> Int:
    return 7

func right() -> Int:
    return 8

func typed_worker() -> Int throws TaskErr:
    throw TaskErr.boom(10, 17)

func main() -> Int
uses runtime:
    let left_task: task.i32 = core.task_spawn_i32("left")
    let right_task: task.i32 = core.task_spawn_i32("right")
    let typed_task = core.task_spawn_i32_typed<TaskErr>("typed_worker")
    let left_value: Int = core.task_join_i32(left_task)
    let right_value: Int = core.task_join_i32(right_task)
    let typed_value: Int = catch core.task_join_i32_typed<TaskErr>(typed_task):
    case TaskErr.boom(first, second):
        first + second
    case TaskErr.stopped:
        99
    return left_value + right_value + typed_value
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, "linux-x64"); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if err := checkX64ELFExecutable(data, "x64 scheduler regression"); err != nil {
		return err
	}
	if hostGOOS(deps) != "linux" || hostGOARCH(deps) != "amd64" {
		return nil
	}
	result := runExecutable(deps, outPath)
	if result.TimedOut {
		return fmt.Errorf("x64 scheduler regression executable timed out: %q", result.Output)
	}
	if result.Err != nil {
		return fmt.Errorf("run x64 scheduler regression: %w output=%q", result.Err, result.Output)
	}
	if result.ExitCode != 42 {
		return fmt.Errorf(
			"x64 scheduler regression exit=%d output=%q, want 42",
			result.ExitCode,
			result.Output,
		)
	}
	return nil
}

func buildExecutable(deps RuntimeSmokeDeps, srcPath string, outPath string, target string) error {
	return buildExecutableWithOptions(deps, srcPath, outPath, target, RuntimeBuildOptions{})
}

func buildExecutableWithOptions(
	deps RuntimeSmokeDeps,
	srcPath string,
	outPath string,
	target string,
	opts RuntimeBuildOptions,
) error {
	if deps.BuildExecutableWithOptions != nil {
		return deps.BuildExecutableWithOptions(srcPath, outPath, target, opts)
	}
	if opts.IslandsDebug {
		return fmt.Errorf("missing runtime smoke build executable-with-options callback")
	}
	if deps.BuildExecutable != nil {
		return deps.BuildExecutable(srcPath, outPath, target)
	}
	return fmt.Errorf("missing runtime smoke build executable callback")
}

func checkX64ELFExecutable(data []byte, label string) error {
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("%s output is not an ELF executable", label)
	}
	if data[4] != 2 {
		return fmt.Errorf("%s ELF class = %d, want ELFCLASS64", label, data[4])
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != 0x3e {
		return fmt.Errorf("%s ELF machine = %#x, want EM_X86_64", label, machine)
	}
	return nil
}

func hostGOOS(deps RuntimeSmokeDeps) string {
	if deps.HostGOOS != "" {
		return deps.HostGOOS
	}
	return runtime.GOOS
}

func hostGOARCH(deps RuntimeSmokeDeps) string {
	if deps.HostGOARCH != "" {
		return deps.HostGOARCH
	}
	return runtime.GOARCH
}

func runExecutable(deps RuntimeSmokeDeps, path string) RuntimeRunResult {
	if deps.RunExecutable != nil {
		return deps.RunExecutable(path)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	output := out.String()
	if ctx.Err() == context.DeadlineExceeded {
		return RuntimeRunResult{Output: output, TimedOut: true}
	}
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			return RuntimeRunResult{Output: output, Err: err}
		}
		return RuntimeRunResult{ExitCode: exitErr.ProcessState.ExitCode(), Output: output}
	}
	return RuntimeRunResult{Output: output}
}
