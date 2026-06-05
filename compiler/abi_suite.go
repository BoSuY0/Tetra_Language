package compiler

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"tetra_language/compiler/internal/backend/linux_x32"
	"tetra_language/compiler/internal/backend/linux_x86"
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/backend/x86abi"
	"tetra_language/compiler/internal/ir"
	ctarget "tetra_language/compiler/target"
)

type ABICheck struct {
	Name  string
	Error string
}

func RunTargetABIChecks(targetName string) ([]ABICheck, error) {
	tgt, err := ctarget.Parse(targetName)
	if err != nil {
		return nil, err
	}
	switch {
	case tgt.Arch == ctarget.ArchX86 && tgt.ABI == ctarget.ABI386SysV:
		return runX86ABIChecks(tgt), nil
	case tgt.Arch == ctarget.ArchX64 && tgt.ABI == ctarget.ABIX32SysV:
		return runX32ABIChecks(tgt), nil
	case tgt.Arch == ctarget.ArchX64:
		return runX64ABIChecks(tgt), nil
	case tgt.Arch == ctarget.ArchWASM32:
		return runWASMABIChecks(tgt), nil
	default:
		return nil, fmt.Errorf("ABI suite for target %s is not implemented", tgt.Triple)
	}
}

func runX86ABIChecks(tgt ctarget.Target) []ABICheck {
	return runABIChecks([]struct {
		name string
		run  func() error
	}{
		{name: "x86 target model", run: func() error { return checkX86TargetModel(tgt) }},
		{name: "x86 i386 SysV classifier", run: func() error { return checkX86I386Classifier(tgt) }},
		{name: "x86 varargs and sret ABI", run: func() error { return checkX86VarargsAndSRet(tgt) }},
		{name: "x86 pointer FFI object smoke", run: func() error { return checkPointerFFIObjectSmoke(tgt) }},
		{name: "x86 c_int FFI object smoke", run: func() error { return checkCIntFFIObjectSmoke(tgt) }},
		{name: "x86 c_uint FFI object smoke", run: func() error { return checkCUIntFFIObjectSmoke(tgt) }},
		{name: "x86 ILP32 native/libc FFI object smoke", run: func() error { return checkILP32NativeLibcFFIObjectSmoke(tgt) }},
		{name: "x86 ref FFI null-return diagnostics", run: checkX86RefFFINullReturnDiagnostics},
		{name: "x86 function-pointer FFI diagnostics", run: checkX86FunctionPointerFFIDiagnostics},
		{name: "x86 source native scalar diagnostics", run: func() error { return checkSourceNativeScalarDiagnostics(tgt) }},
		{name: "x86 stdout executable smoke", run: checkX86StdoutExecutableSmoke},
		{name: "x86 stderr fd runtime smoke", run: checkX86StderrFDRuntimeSmoke},
		{name: "x86 allocator executable smoke", run: checkX86AllocatorExecutableSmoke},
		{name: "x86 allocator failure executable smoke", run: checkX86AllocatorFailureExecutableSmoke},
		{name: "x86 raw memory bounds executable smoke", run: checkX86RawMemoryBoundsExecutableSmoke},
		{name: "x86 raw pointer slot executable smoke", run: checkX86RawPointerSlotExecutableSmoke},
		{name: "x86 raw pointer offset slot executable smoke", run: checkX86RawPointerOffsetSlotExecutableSmoke},
		{name: "x86 island free executable smoke", run: checkX86IslandFreeExecutableSmoke},
		{name: "x86 stdlib runtime boundary diagnostics", run: func() error { return checkStdlibRuntimeBoundaryDiagnostics(tgt) }},
		{name: "x86 filesystem runtime smoke", run: checkX86FilesystemRuntimeSmoke},
		{name: "x86 filesystem scheduler composition smoke", run: checkX86FilesystemSchedulerCompositionSmoke},
		{name: "x86 time runtime smoke", run: checkX86TimeRuntimeSmoke},
		{name: "x86 single-actor self-host runtime smoke", run: checkX86SingleActorSelfHostRuntimeSmoke},
		{name: "x86 single-task self-host runtime smoke", run: checkX86SingleTaskSelfHostRuntimeSmoke},
		{name: "x86 typed-task self-host runtime smoke", run: checkX86TypedTaskSelfHostRuntimeSmoke},
		{name: "x86 staged typed-task self-host runtime smoke", run: checkX86StagedTypedTaskSelfHostRuntimeSmoke},
		{name: "x86 task-group self-host runtime smoke", run: checkX86TaskGroupSelfHostRuntimeSmoke},
		{name: "x86 typed-task-group self-host runtime smoke", run: checkX86TypedTaskGroupSelfHostRuntimeSmoke},
		{name: "x86 actor-state self-host runtime smoke", run: checkX86ActorStateSelfHostRuntimeSmoke},
		{name: "x86 ctx_switch object smoke", run: checkX86CtxSwitchObjectSmoke},
		{name: "x86 target runtime boundary diagnostics", run: func() error { return checkTargetRuntimeBoundaryDiagnostics(tgt) }},
		{name: "x86 networking runtime boundary diagnostics", run: func() error { return checkNetworkingRuntimeBoundaryDiagnostics(tgt) }},
		{name: "x86 networking lifecycle runtime smoke", run: checkX86NetworkingLifecycleRuntimeSmoke},
		{name: "x86 surface/distributed runtime boundary diagnostics", run: func() error { return checkSurfaceDistributedRuntimeBoundaryDiagnostics(tgt) }},
		{name: "x86 pointer atomic ABI width", run: func() error { return checkAtomicPointerObjectWidth(tgt) }},
	})
}

func runX64ABIChecks(tgt ctarget.Target) []ABICheck {
	abiName := "SysV"
	if tgt.ABI == ctarget.ABIWin64 {
		abiName = "Win64"
	}
	prefix := x64ABICheckPrefix(tgt)
	checks := []struct {
		name string
		run  func() error
	}{
		{name: prefix + " target model", run: func() error { return checkX64TargetModel(tgt) }},
		{name: prefix + " " + abiName + " classifier", run: func() error { return checkX64Classifier(tgt) }},
		{name: prefix + " " + abiName + " varargs and aggregates", run: func() error { return checkX64VarargsAndAggregates(tgt) }},
	}
	if tgt.Triple == "macos-x64" || tgt.Triple == "windows-x64" {
		checks = append(checks, struct {
			name string
			run  func() error
		}{name: prefix + " object ABI smoke", run: func() error { return checkX64PlatformObjectABISmoke(tgt) }})
	}
	checks = append(checks, struct {
		name string
		run  func() error
	}{name: prefix + " source native scalar diagnostics", run: func() error { return checkSourceNativeScalarDiagnostics(tgt) }})
	if tgt.Triple == "linux-x64" {
		checks = append(checks, struct {
			name string
			run  func() error
		}{name: prefix + " pointer FFI regression smoke", run: checkX64PointerFFIRegressionSmoke})
		checks = append(checks, struct {
			name string
			run  func() error
		}{name: prefix + " c_int FFI object smoke", run: func() error { return checkCIntFFIObjectSmoke(tgt) }})
		checks = append(checks, struct {
			name string
			run  func() error
		}{name: prefix + " c_uint FFI object smoke", run: func() error { return checkCUIntFFIObjectSmoke(tgt) }})
		checks = append(checks, struct {
			name string
			run  func() error
		}{name: prefix + " filesystem scheduler composition smoke", run: checkX64FilesystemSchedulerCompositionSmoke})
		checks = append(checks, struct {
			name string
			run  func() error
		}{name: prefix + " networking runtime smoke", run: checkX64NetworkingRuntimeSmoke})
		checks = append(checks, struct {
			name string
			run  func() error
		}{name: prefix + " scheduler restriction regression smoke", run: checkX64SchedulerRestrictionRegressionSmoke})
	}
	checks = append(checks, struct {
		name string
		run  func() error
	}{name: prefix + " pointer atomic ABI width", run: func() error { return checkAtomicPointerObjectWidth(tgt) }})
	return runABIChecks(checks)
}

func x64ABICheckPrefix(tgt ctarget.Target) string {
	switch tgt.Triple {
	case "windows-x64", "macos-x64":
		return tgt.Triple
	default:
		return "x64"
	}
}

func runX32ABIChecks(tgt ctarget.Target) []ABICheck {
	return runABIChecks([]struct {
		name string
		run  func() error
	}{
		{name: "x32 target model", run: func() error { return checkX32TargetModel(tgt) }},
		{name: "x32 SysV classifier", run: func() error { return checkX32SysVClassifier(tgt) }},
		{name: "x32 SysV varargs and aggregates", run: func() error { return checkX32SysVVarargsAndAggregates(tgt) }},
		{name: "x32 pointer FFI object smoke", run: func() error { return checkPointerFFIObjectSmoke(tgt) }},
		{name: "x32 c_int FFI object smoke", run: func() error { return checkCIntFFIObjectSmoke(tgt) }},
		{name: "x32 c_uint FFI object smoke", run: func() error { return checkCUIntFFIObjectSmoke(tgt) }},
		{name: "x32 ILP32 native/libc FFI object smoke", run: func() error { return checkILP32NativeLibcFFIObjectSmoke(tgt) }},
		{name: "x32 ref FFI null-return diagnostics", run: checkX32RefFFINullReturnDiagnostics},
		{name: "x32 function-pointer FFI diagnostics", run: checkX32FunctionPointerFFIDiagnostics},
		{name: "x32 source native scalar diagnostics", run: func() error { return checkSourceNativeScalarDiagnostics(tgt) }},
		{name: "x32 stdout executable smoke", run: checkX32StdoutExecutableSmoke},
		{name: "x32 stderr fd runtime smoke", run: checkX32StderrFDRuntimeSmoke},
		{name: "x32 allocator executable smoke", run: checkX32AllocatorExecutableSmoke},
		{name: "x32 allocator failure executable smoke", run: checkX32AllocatorFailureExecutableSmoke},
		{name: "x32 raw memory bounds executable smoke", run: checkX32RawMemoryBoundsExecutableSmoke},
		{name: "x32 raw pointer slot executable smoke", run: checkX32RawPointerSlotExecutableSmoke},
		{name: "x32 raw pointer offset slot executable smoke", run: checkX32RawPointerOffsetSlotExecutableSmoke},
		{name: "x32 island free executable smoke", run: checkX32IslandFreeExecutableSmoke},
		{name: "x32 stdlib runtime boundary diagnostics", run: func() error { return checkStdlibRuntimeBoundaryDiagnostics(tgt) }},
		{name: "x32 time runtime smoke", run: checkX32TimeRuntimeSmoke},
		{name: "x32 filesystem runtime smoke", run: checkX32FilesystemRuntimeSmoke},
		{name: "x32 filesystem scheduler composition smoke", run: checkX32FilesystemSchedulerCompositionSmoke},
		{name: "x32 single-actor self-host runtime smoke", run: checkX32SingleActorSelfHostRuntimeSmoke},
		{name: "x32 single-task self-host runtime smoke", run: checkX32SingleTaskSelfHostRuntimeSmoke},
		{name: "x32 typed-task self-host runtime smoke", run: checkX32TypedTaskSelfHostRuntimeSmoke},
		{name: "x32 staged typed-task self-host runtime smoke", run: checkX32StagedTypedTaskSelfHostRuntimeSmoke},
		{name: "x32 task-group self-host runtime smoke", run: checkX32TaskGroupSelfHostRuntimeSmoke},
		{name: "x32 typed-task-group self-host runtime smoke", run: checkX32TypedTaskGroupSelfHostRuntimeSmoke},
		{name: "x32 actor-state self-host runtime smoke", run: checkX32ActorStateSelfHostRuntimeSmoke},
		{name: "x32 ctx_switch object smoke", run: checkX32CtxSwitchObjectSmoke},
		{name: "x32 target runtime boundary diagnostics", run: func() error { return checkTargetRuntimeBoundaryDiagnostics(tgt) }},
		{name: "x32 networking runtime boundary diagnostics", run: func() error { return checkNetworkingRuntimeBoundaryDiagnostics(tgt) }},
		{name: "x32 networking lifecycle runtime smoke", run: checkX32NetworkingLifecycleRuntimeSmoke},
		{name: "x32 surface/distributed runtime boundary diagnostics", run: func() error { return checkSurfaceDistributedRuntimeBoundaryDiagnostics(tgt) }},
		{name: "x32 pointer atomic ABI width", run: func() error { return checkAtomicPointerObjectWidth(tgt) }},
	})
}

func runABIChecks(cases []struct {
	name string
	run  func() error
}) []ABICheck {
	out := make([]ABICheck, 0, len(cases))
	for _, tc := range cases {
		check := ABICheck{Name: tc.name}
		if err := tc.run(); err != nil {
			check.Error = err.Error()
		}
		out = append(out, check)
	}
	return out
}

func checkX86TargetModel(tgt ctarget.Target) error {
	if tgt.Triple != "linux-x86" || tgt.OS != ctarget.OSLinux || tgt.Arch != ctarget.ArchX86 || tgt.ABI != ctarget.ABI386SysV {
		return fmt.Errorf("x86 identity = triple=%s os=%s arch=%s abi=%s, want linux-x86/linux/x86/i386-sysv", tgt.Triple, tgt.OS, tgt.Arch, tgt.ABI)
	}
	if tgt.DataModel != ctarget.DataModelILP32 || tgt.Format != ctarget.FormatELF || tgt.Endian != ctarget.EndianLittle {
		return fmt.Errorf("x86 platform = model=%s format=%s endian=%s, want ilp32/elf/little", tgt.DataModel, tgt.Format, tgt.Endian)
	}
	if tgt.PointerWidthBits != 32 || tgt.NativeIntWidthBits != 32 || tgt.RegisterWidthBits != 32 || tgt.StackAlignmentBytes != 16 || tgt.MaxAtomicWidthBits != 32 {
		return fmt.Errorf("x86 widths = ptr=%d native=%d reg=%d stack=%d atomic=%d, want 32/32/32/16/32", tgt.PointerWidthBits, tgt.NativeIntWidthBits, tgt.RegisterWidthBits, tgt.StackAlignmentBytes, tgt.MaxAtomicWidthBits)
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

func checkX86I386Classifier(tgt ctarget.Target) error {
	classifier, err := x86abi.NewClassifier(tgt)
	if err != nil {
		return err
	}
	if classifier.Name() != "i386-sysv" || classifier.StackCleanup() != x86abi.StackCleanupCaller {
		return fmt.Errorf("x86 classifier identity = %s cleanup=%s, want i386-sysv caller cleanup", classifier.Name(), classifier.StackCleanup())
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
	if plan.PointerWidthBits != 32 || plan.RegisterWidthBits != 32 || plan.StackCleanup != x86abi.StackCleanupCaller {
		return fmt.Errorf("x86 ABI plan identity = %#v, want 32-bit pointer/register caller-cleaned stack", plan)
	}
	if err := expectX86StackArg(plan.Params[0], "p", x86abi.ABIClassInteger, 0, 4, 4, 4); err != nil {
		return err
	}
	if err := expectX86StackArg(plan.Params[1], "wide", x86abi.ABIClassInteger, 4, 8, 8, 4); err != nil {
		return err
	}
	if err := expectX86StackArg(plan.Params[2], "f", x86abi.ABIClassX87, 12, 4, 4, 4); err != nil {
		return err
	}
	if got := plan.Return; got.Register != "eax" || got.Class != x86abi.ABIClassInteger || got.SizeBytes != 4 || got.Extension != x86abi.ABIExtendNone {
		return fmt.Errorf("x86 ptr return = %#v, want eax pointer return without widening extension", got)
	}
	scalarReturns, err := classifier.ClassifySignature(x86abi.ABISignature{Return: &x86abi.ABIParam{Name: "ret", Type: "i64"}})
	if err != nil {
		return err
	}
	if got := scalarReturns.Return; got.Register != "edx:eax" || !sameStrings(got.Registers, []string{"eax", "edx"}) || got.SizeBytes != 8 || got.Class != x86abi.ABIClassInteger {
		return fmt.Errorf("x86 i64 return = %#v, want edx:eax", got)
	}
	floatReturns, err := classifier.ClassifySignature(x86abi.ABISignature{Return: &x86abi.ABIParam{Name: "ret", Type: "f64"}})
	if err != nil {
		return err
	}
	if got := floatReturns.Return; got.Register != "st0" || got.Class != x86abi.ABIClassX87 || got.RegisterWidthBits != 80 {
		return fmt.Errorf("x86 f64 return = %#v, want x87 st0", got)
	}
	return nil
}

func checkX86VarargsAndSRet(tgt ctarget.Target) error {
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
	if got := aggregate.Params[0]; got.Class != x86abi.ABIClassMemory || got.StackOffsetBytes != 4 || got.StackSlotBytes != 8 || got.SizeBytes != 8 || got.AlignBytes != 4 {
		return fmt.Errorf("x86 struct param = %#v, want stack copy after hidden sret pointer", got)
	}
	if got := aggregate.Return; got.Class != x86abi.ABIClassMemory || !got.Indirect || got.Register != "sret@stack+0" || got.StackOffsetBytes != 0 || got.StackSlotBytes != 4 || got.SizeBytes != 8 {
		return fmt.Errorf("x86 struct return = %#v, want hidden sret pointer at first stack argument", got)
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
	if !variadic.Variadic || variadic.FixedParamCount != 1 || variadic.VarargStartIndex != 1 || variadic.StackCleanup != x86abi.StackCleanupCaller {
		return fmt.Errorf("x86 variadic metadata = %#v, want caller-cleaned stack varargs", variadic)
	}
	if variadic.RegisterVarargs || variadic.VarargRegisterSaveBytes != 0 {
		return fmt.Errorf("x86 varargs unexpectedly require register save area: %#v", variadic)
	}
	if err := expectX86StackArg(variadic.Params[1], "first", x86abi.ABIClassX87, 4, 8, 8, 4); err != nil {
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

func checkX64TargetModel(tgt ctarget.Target) error {
	if tgt.Arch != ctarget.ArchX64 || tgt.PointerWidthBits != 64 || tgt.NativeIntWidthBits != 64 || tgt.RegisterWidthBits != 64 || tgt.StackAlignmentBytes != 16 || tgt.MaxAtomicWidthBits != 64 {
		return fmt.Errorf("x64 widths = arch=%s ptr=%d native=%d reg=%d stack=%d atomic=%d, want x64/64/64/64/16/64", tgt.Arch, tgt.PointerWidthBits, tgt.NativeIntWidthBits, tgt.RegisterWidthBits, tgt.StackAlignmentBytes, tgt.MaxAtomicWidthBits)
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
		if tgt.DataModel != ctarget.DataModelLP64 || tgt.Format != ctarget.FormatELF && tgt.Format != ctarget.FormatMachO {
			return fmt.Errorf("x64 SysV platform = model=%s format=%s, want lp64/elf-or-macho", tgt.DataModel, tgt.Format)
		}
		if err := expectTargetScalarLayout(tgt, "c_long", 8, 8); err != nil {
			return err
		}
	case ctarget.ABIWin64:
		if tgt.DataModel != ctarget.DataModelLLP64 || tgt.Format != ctarget.FormatPE {
			return fmt.Errorf("x64 Win64 platform = model=%s format=%s, want llp64/pe", tgt.DataModel, tgt.Format)
		}
		if err := expectTargetScalarLayout(tgt, "c_long", 4, 4); err != nil {
			return err
		}
	default:
		return fmt.Errorf("x64 unsupported ABI %s", tgt.ABI)
	}
	return nil
}

func checkX64Classifier(tgt ctarget.Target) error {
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
		if err := expectX64Arg(plan.Params[0], "p", x64abi.ABIClassInteger, "rdi", 8, 8, 64, x64abi.ABIExtendNone); err != nil {
			return err
		}
		if err := expectX64Arg(plan.Params[1], "n", x64abi.ABIClassInteger, "rsi", 8, 8, 64, x64abi.ABIExtendNone); err != nil {
			return err
		}
		if err := expectX64Arg(plan.Params[2], "wide", x64abi.ABIClassInteger, "rdx", 8, 8, 64, x64abi.ABIExtendNone); err != nil {
			return err
		}
		if err := expectX64Arg(plan.Params[3], "f", x64abi.ABIClassSSE, "xmm0", 8, 8, 128, x64abi.ABIExtendNone); err != nil {
			return err
		}
	case ctarget.ABIWin64:
		if classifier.Name() != "win64" {
			return fmt.Errorf("x64 Win64 classifier name = %s, want win64", classifier.Name())
		}
		if err := expectX64Arg(plan.Params[0], "p", x64abi.ABIClassInteger, "rcx", 8, 8, 64, x64abi.ABIExtendNone); err != nil {
			return err
		}
		if err := expectX64Arg(plan.Params[1], "n", x64abi.ABIClassInteger, "rdx", 8, 8, 64, x64abi.ABIExtendNone); err != nil {
			return err
		}
		if err := expectX64Arg(plan.Params[2], "wide", x64abi.ABIClassInteger, "r8", 8, 8, 64, x64abi.ABIExtendNone); err != nil {
			return err
		}
		if err := expectX64Arg(plan.Params[3], "f", x64abi.ABIClassSSE, "xmm3", 8, 8, 128, x64abi.ABIExtendNone); err != nil {
			return err
		}
	default:
		return fmt.Errorf("x64 unsupported classifier ABI %s", tgt.ABI)
	}
	if got := plan.Return; got.Register != "rax" || got.Class != x64abi.ABIClassInteger || got.SizeBytes != 8 {
		return fmt.Errorf("x64 ptr return = %#v, want rax", got)
	}
	return nil
}

func checkX64VarargsAndAggregates(tgt ctarget.Target) error {
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
		if !variadic.Variadic || !variadic.SysVRequiresAL || variadic.SysV_ALSSERegisterCount != 2 || variadic.Win64ShadowSpaceBytes != 0 || len(variadic.Win64VarargFloatMirrors) != 0 {
			return fmt.Errorf("x64 SysV variadic metadata = %#v, want %%al SSE upper bound 2 and no Win64 mirrors", variadic)
		}
		fields := []ctarget.LayoutField{{Name: "raw", Type: "ptr"}, {Name: "count", Type: "usize"}}
		aggregate, err := classifier.ClassifySignature(x64abi.ABISignature{
			Params: []x64abi.ABIParam{{Name: "view", Type: "View", Fields: fields}},
			Return: &x64abi.ABIParam{Name: "ret", Type: "View", Fields: fields},
		})
		if err != nil {
			return err
		}
		if got := aggregate.Params[0]; got.SizeBytes != 16 || got.AlignBytes != 8 || got.Class != x64abi.ABIClassInteger || !sameStrings(got.Registers, []string{"rdi", "rsi"}) {
			return fmt.Errorf("x64 SysV aggregate param = %#v, want two integer registers", got)
		}
		if got := aggregate.Return; got.SizeBytes != 16 || got.AlignBytes != 8 || got.Class != x64abi.ABIClassInteger || !sameStrings(got.Registers, []string{"rax", "rdx"}) {
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
		if !variadic.Variadic || variadic.Win64ShadowSpaceBytes != 32 || variadic.SysVRequiresAL || variadic.SysV_ALSSERegisterCount != 0 {
			return fmt.Errorf("x64 Win64 variadic metadata = %#v, want shadow space and no SysV %%al", variadic)
		}
		wantMirrors := []x64abi.VarargFloatMirror{
			{ParamIndex: 1, XMMRegister: "xmm1", GPRegister: "rdx"},
			{ParamIndex: 3, XMMRegister: "xmm3", GPRegister: "r9"},
		}
		if !sameX64Mirrors(variadic.Win64VarargFloatMirrors, wantMirrors) {
			return fmt.Errorf("x64 Win64 float mirrors = %#v, want %#v", variadic.Win64VarargFloatMirrors, wantMirrors)
		}
		smallFields := []ctarget.LayoutField{{Name: "lo", Type: "u32"}, {Name: "hi", Type: "u32"}}
		small, err := classifier.ClassifySignature(x64abi.ABISignature{
			Params: []x64abi.ABIParam{{Name: "pair", Type: "Pair", Fields: smallFields}},
			Return: &x64abi.ABIParam{Name: "ret", Type: "Pair", Fields: smallFields},
		})
		if err != nil {
			return err
		}
		if got := small.Params[0]; got.Class != x64abi.ABIClassInteger || got.Register != "rcx" || got.SizeBytes != 8 || got.Indirect {
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
		if got := large.Params[0]; got.Class != x64abi.ABIClassMemory || !got.Indirect || got.Register != "rcx" || got.SizeBytes != 16 || got.ABIBytes != 8 {
			return fmt.Errorf("x64 Win64 large aggregate param = %#v, want by-reference pointer in rcx", got)
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

func checkX32TargetModel(tgt ctarget.Target) error {
	if tgt.Triple != "linux-x32" || tgt.OS != ctarget.OSLinux || tgt.Arch != ctarget.ArchX64 || tgt.ABI != ctarget.ABIX32SysV {
		return fmt.Errorf("x32 identity = triple=%s os=%s arch=%s abi=%s, want linux-x32/linux/x64/x32-sysv", tgt.Triple, tgt.OS, tgt.Arch, tgt.ABI)
	}
	if tgt.DataModel != ctarget.DataModelX32 || tgt.Format != ctarget.FormatELF || tgt.Endian != ctarget.EndianLittle {
		return fmt.Errorf("x32 platform = model=%s format=%s endian=%s, want x32/elf/little", tgt.DataModel, tgt.Format, tgt.Endian)
	}
	if tgt.PointerWidthBits != 32 || tgt.NativeIntWidthBits != 32 || tgt.RegisterWidthBits != 64 || tgt.StackAlignmentBytes != 16 || tgt.MaxAtomicWidthBits != 64 {
		return fmt.Errorf("x32 widths = ptr=%d native=%d reg=%d stack=%d atomic=%d, want 32/32/64/16/64", tgt.PointerWidthBits, tgt.NativeIntWidthBits, tgt.RegisterWidthBits, tgt.StackAlignmentBytes, tgt.MaxAtomicWidthBits)
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
	if x86.Arch == tgt.Arch || x86.RegisterWidthBits == tgt.RegisterWidthBits || x86.MaxAtomicWidthBits == tgt.MaxAtomicWidthBits {
		return fmt.Errorf("x32 collapsed into x86: x86 arch=%s reg=%d atomic=%d, x32 arch=%s reg=%d atomic=%d", x86.Arch, x86.RegisterWidthBits, x86.MaxAtomicWidthBits, tgt.Arch, tgt.RegisterWidthBits, tgt.MaxAtomicWidthBits)
	}
	x64, err := ctarget.Parse("x64")
	if err != nil {
		return err
	}
	if x64.PointerWidthBits == tgt.PointerWidthBits || x64.NativeIntWidthBits == tgt.NativeIntWidthBits || x64.ABI == tgt.ABI {
		return fmt.Errorf("x32 collapsed into x64: x64 ptr=%d native=%d abi=%s, x32 ptr=%d native=%d abi=%s", x64.PointerWidthBits, x64.NativeIntWidthBits, x64.ABI, tgt.PointerWidthBits, tgt.NativeIntWidthBits, tgt.ABI)
	}
	return nil
}

func checkX32SysVClassifier(tgt ctarget.Target) error {
	classifier, err := x64abi.NewClassifier(tgt)
	if err != nil {
		return err
	}
	if classifier.Name() != "x32-sysv" || !classifier.UsesX64Registers() {
		return fmt.Errorf("x32 classifier identity = %s x64regs=%v, want x32-sysv with x86_64 registers", classifier.Name(), classifier.UsesX64Registers())
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
		return fmt.Errorf("x32 ABI plan identity = %#v, want 32-bit pointers and 64-bit registers", plan)
	}
	if err := expectX64Arg(plan.Params[0], "p", x64abi.ABIClassInteger, "rdi", 4, 4, 64, x64abi.ABIExtendZero); err != nil {
		return err
	}
	if err := expectX64Arg(plan.Params[1], "n", x64abi.ABIClassInteger, "rsi", 4, 4, 64, x64abi.ABIExtendZero); err != nil {
		return err
	}
	if err := expectX64Arg(plan.Params[2], "s", x64abi.ABIClassInteger, "rdx", 4, 4, 64, x64abi.ABIExtendSign); err != nil {
		return err
	}
	if err := expectX64Arg(plan.Params[3], "wide", x64abi.ABIClassInteger, "rcx", 8, 8, 64, x64abi.ABIExtendNone); err != nil {
		return err
	}
	if err := expectX64Arg(plan.Params[4], "f", x64abi.ABIClassSSE, "xmm0", 8, 8, 128, x64abi.ABIExtendNone); err != nil {
		return err
	}
	if got := plan.Return; got.Register != "rax" || got.Class != x64abi.ABIClassInteger || got.SizeBytes != 4 || got.AlignBytes != 4 || got.RegisterWidthBits != 64 || got.Extension != x64abi.ABIExtendZero {
		return fmt.Errorf("x32 ptr return = %#v, want zero-extended 32-bit pointer in rax", got)
	}
	x86Tgt, err := ctarget.Parse("x86")
	if err != nil {
		return err
	}
	if _, err := x64abi.NewClassifier(x86Tgt); err == nil || !strings.Contains(err.Error(), "x64abi classifier requires x64 ISA") {
		return fmt.Errorf("x32 classifier did not keep i386 separate: %v", err)
	}
	return nil
}

func checkX32SysVVarargsAndAggregates(tgt ctarget.Target) error {
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
	if !variadic.Variadic || variadic.FixedParamCount != 1 || variadic.VarargStartIndex != 1 || !variadic.RegisterVarargs {
		return fmt.Errorf("x32 variadic metadata = %#v, want register varargs after fixed prefix", variadic)
	}
	if !variadic.SysVRequiresAL || variadic.SysV_ALSSERegisterCount != 2 || variadic.VarargRegisterSaveBytes != 176 {
		return fmt.Errorf("x32 SysV vararg AL metadata = %#v, want %%al upper bound 2 and 176-byte save area", variadic)
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
	if got := aggregate.Params[0]; got.SizeBytes != 8 || got.AlignBytes != 4 || got.Class != x64abi.ABIClassInteger || got.Register != "rdi" || !sameStrings(got.Registers, []string{"rdi"}) {
		return fmt.Errorf("x32 aggregate param = %#v, want one integer register carrying ptr32+usize32 aggregate", got)
	}
	if got := aggregate.Return; got.SizeBytes != 8 || got.AlignBytes != 4 || got.Class != x64abi.ABIClassInteger || got.Register != "rax" || !sameStrings(got.Registers, []string{"rax"}) {
		return fmt.Errorf("x32 aggregate return = %#v, want rax carrying ptr32+usize32 aggregate", got)
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
	if got := x64Plan.Params[0]; got.SizeBytes != 16 || got.AlignBytes != 8 || !sameStrings(got.Registers, []string{"rdi", "rsi"}) {
		return fmt.Errorf("x32 aggregate comparison failed: x64 aggregate = %#v, want two-register LP64 layout", got)
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
	if got := large.Params[0]; got.Class != x64abi.ABIClassMemory || got.Register != "" || got.StackOffsetBytes != 0 || got.StackSlotBytes != 24 || got.SizeBytes != 20 {
		return fmt.Errorf("x32 large aggregate param = %#v, want 20-byte memory aggregate in 24-byte stack slot", got)
	}
	if got := large.Return; got.Class != x64abi.ABIClassMemory || !got.Indirect || got.Register != "rdi" || got.SizeBytes != 20 {
		return fmt.Errorf("x32 large aggregate return = %#v, want hidden sret pointer in rdi", got)
	}
	return nil
}

func checkX86RefFFINullReturnDiagnostics() error {
	return checkRefFFINullReturnDiagnostics("linux-x86", "x86")
}

func checkX32RefFFINullReturnDiagnostics() error {
	return checkRefFFINullReturnDiagnostics("linux-x32", "x32")
}

func checkX86FunctionPointerFFIDiagnostics() error {
	return checkFunctionPointerFFIDiagnostics("linux-x86", "i386", "x86")
}

func checkX32FunctionPointerFFIDiagnostics() error {
	return checkFunctionPointerFFIDiagnostics("linux-x32", "x32", "x32")
}

func checkPointerFFIObjectSmoke(tgt ctarget.Target) error {
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
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{Emit: EmitLibrary, Jobs: 1}); err != nil {
		return err
	}
	obj, err := ReadObject(outPath)
	if err != nil {
		return err
	}
	if obj.Target != tgt.Triple {
		return fmt.Errorf("%s pointer FFI object target = %q, want %s", stem, obj.Target, tgt.Triple)
	}
	if !abiSuiteObjectHasSymbolSignature(obj, "ffi_ptr_identity_c", 1, 1) {
		return fmt.Errorf("%s pointer FFI object missing exported ffi_ptr_identity_c(1)->1 symbol: %#v", stem, obj.Symbols)
	}
	if !abiSuiteObjectHasSymbolSignature(obj, "ffi_rawptr_identity_c", 1, 1) {
		return fmt.Errorf("%s pointer FFI object missing exported ffi_rawptr_identity_c(1)->1 symbol: %#v", stem, obj.Symbols)
	}
	if !abiSuiteObjectHasSymbolSignature(obj, "ffi_nullable_ptr_identity_c", 1, 1) {
		return fmt.Errorf("%s pointer FFI object missing exported ffi_nullable_ptr_identity_c(1)->1 symbol: %#v", stem, obj.Symbols)
	}
	if !abiSuiteObjectHasSymbolSignature(obj, "ffi_nullable_ptr_null_c", 0, 1) {
		return fmt.Errorf("%s pointer FFI object missing exported ffi_nullable_ptr_null_c(0)->1 symbol: %#v", stem, obj.Symbols)
	}
	if !abiSuiteObjectHasSymbolSignature(obj, "ffi_ref_identity_c", 1, 1) {
		return fmt.Errorf("%s pointer FFI object missing exported ffi_ref_identity_c(1)->1 symbol: %#v", stem, obj.Symbols)
	}
	return nil
}

func checkCIntFFIObjectSmoke(tgt ctarget.Target) error {
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
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{Emit: EmitLibrary, Jobs: 1}); err != nil {
		return err
	}
	obj, err := ReadObject(outPath)
	if err != nil {
		return err
	}
	if obj.Target != tgt.Triple {
		return fmt.Errorf("%s c_int FFI object target = %q, want %s", stem, obj.Target, tgt.Triple)
	}
	if !abiSuiteObjectHasSymbolSignature(obj, "ffi_c_int_identity_c", 1, 1) {
		return fmt.Errorf("%s c_int FFI object missing exported ffi_c_int_identity_c(1)->1 symbol: %#v", stem, obj.Symbols)
	}
	return nil
}

func checkCUIntFFIObjectSmoke(tgt ctarget.Target) error {
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
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{Emit: EmitLibrary, Jobs: 1}); err != nil {
		return err
	}
	obj, err := ReadObject(outPath)
	if err != nil {
		return err
	}
	if obj.Target != tgt.Triple {
		return fmt.Errorf("%s c_uint FFI object target = %q, want %s", stem, obj.Target, tgt.Triple)
	}
	if !abiSuiteObjectHasSymbolSignature(obj, "ffi_c_uint_identity_c", 1, 1) {
		return fmt.Errorf("%s c_uint FFI object missing exported ffi_c_uint_identity_c(1)->1 symbol: %#v", stem, obj.Symbols)
	}
	return nil
}

func checkILP32NativeLibcFFIObjectSmoke(tgt ctarget.Target) error {
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
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{Emit: EmitLibrary, Jobs: 1}); err != nil {
		return err
	}
	obj, err := ReadObject(outPath)
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
		if !abiSuiteObjectHasSymbolSignature(obj, symbol, 1, 1) {
			return fmt.Errorf("%s ILP32 native/libc FFI object missing exported %s(1)->1 symbol: %#v", stem, symbol, obj.Symbols)
		}
	}
	return nil
}

func checkRefFFINullReturnDiagnostics(targetName, stem string) error {
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
	_, err = BuildFileWithStatsOpt(srcPath, outPath, targetName, BuildOptions{Emit: EmitLibrary, Jobs: 1})
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

func checkFunctionPointerFFIDiagnostics(targetName, boundaryName, stem string) error {
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
		_, err := BuildFileWithStatsOpt(srcPath, outPath, targetName, BuildOptions{Emit: EmitLibrary, Jobs: 1})
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

func checkX86StdoutExecutableSmoke() error {
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
	})
}

func checkX32StdoutExecutableSmoke() error {
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
	})
}

func checkX86StderrFDRuntimeSmoke() error {
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
	})
}

func checkX32StderrFDRuntimeSmoke() error {
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
	})
}

func checkX86AllocatorExecutableSmoke() error {
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
	})
}

func checkX86AllocatorFailureExecutableSmoke() error {
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
	})
}

func checkX32AllocatorExecutableSmoke() error {
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
	})
}

func checkX32AllocatorFailureExecutableSmoke() error {
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
	})
}

func checkX86RawMemoryBoundsExecutableSmoke() error {
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
			{0xBA, 0x00, 0x00, 0x00, 0x00, 0x83, 0xFA, 0x00, 0x0F, 0x8D},
			{0x81, 0xE7, 0x00, 0xF0, 0xFF, 0xFF},
			{0x8B, 0x0F, 0x83, 0xC7, 0x08},
			{0x01, 0xC2, 0x83, 0xC2, 0x04, 0x39, 0xCA},
			{0x01, 0xC2, 0x83, 0xC2, 0x01, 0x39, 0xCA},
			{0x88, 0x18, 0x53},
			{0x0F, 0xB6, 0x00, 0x50},
		},
		forbidCode: [][]byte{{0x0F, 0x05}},
	})
}

func checkX32RawMemoryBoundsExecutableSmoke() error {
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
			{0xBA, 0x00, 0x00, 0x00, 0x00, 0x81, 0xFA, 0x00, 0x00, 0x00, 0x00, 0x0F, 0x8D},
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
	})
}

func checkX86RawPointerSlotExecutableSmoke() error {
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
			{0xBA, 0x00, 0x00, 0x00, 0x00, 0x83, 0xFA, 0x00, 0x0F, 0x8D},
			{0x81, 0xE7, 0x00, 0xF0, 0xFF, 0xFF},
			{0x8B, 0x0F, 0x83, 0xC7, 0x08},
			{0x01, 0xC2, 0x83, 0xC2, 0x04, 0x39, 0xCA},
			{0x89, 0x18, 0x53},
			{0x8B, 0x00, 0x50},
		},
		forbidCode: [][]byte{{0x0F, 0x05}},
	})
}

func checkX32RawPointerSlotExecutableSmoke() error {
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
			{0xBA, 0x00, 0x00, 0x00, 0x00, 0x81, 0xFA, 0x00, 0x00, 0x00, 0x00, 0x0F, 0x8D},
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
	})
}

func checkX86RawPointerOffsetSlotExecutableSmoke() error {
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
	})
}

func checkX32RawPointerOffsetSlotExecutableSmoke() error {
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
	})
}

func checkX86IslandFreeExecutableSmoke() error {
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
		forbidCode:      [][]byte{{0x0F, 0x05}},
		forbidDebugCode: [][]byte{{0x8B, 0x4B, 0x08, 0xB8, 0x5B, 0x00, 0x00, 0x00, 0xCD, 0x80}, {0x0F, 0x05}},
	})
}

func checkX32IslandFreeExecutableSmoke() error {
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
	})
}

func checkX86NetworkingLifecycleRuntimeSmoke() error {
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
	})
}

func checkX32NetworkingLifecycleRuntimeSmoke() error {
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
	})
}

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

type networkingLifecycleRuntimeSmokeOptions struct {
	target      string
	stem        string
	label       string
	wantClass   byte
	wantMachine uint16
	wantCode    [][]byte
	forbidCode  []byte
}

func checkNetworkingLifecycleRuntimeSmoke(opts networkingLifecycleRuntimeSmokeOptions) error {
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
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, opts.target, BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("%s output is not an ELF executable", opts.label)
	}
	if data[4] != opts.wantClass {
		return fmt.Errorf("%s ELF class = %d, want %d", opts.label, data[4], opts.wantClass)
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != opts.wantMachine {
		return fmt.Errorf("%s ELF machine = %#x, want %#x", opts.label, machine, opts.wantMachine)
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing target networking syscall sequence % x", opts.label, wantCode)
		}
	}
	if len(opts.forbidCode) > 0 && bytes.Contains(data, opts.forbidCode) {
		return fmt.Errorf("%s contains forbidden syscall sequence % x", opts.label, opts.forbidCode)
	}
	return nil
}

func checkIslandFreeExecutableSmoke(opts islandFreeExecutableSmokeOptions) error {
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
	if _, err := BuildFileWithStatsOpt(srcPath, normalPath, opts.target, BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	if err := checkIslandFreeExecutableBytes(normalPath, opts.label, opts, opts.wantCode, opts.forbidCode); err != nil {
		return err
	}

	debugPath := filepath.Join(tmpDir, opts.stem+"_debug")
	if _, err := BuildFileWithStatsOpt(srcPath, debugPath, opts.target, BuildOptions{Jobs: 1, IslandsDebug: true}); err != nil {
		return err
	}
	return checkIslandFreeExecutableBytes(debugPath, opts.label+" debug", opts, opts.wantDebugCode, opts.forbidDebugCode)
}

func checkIslandFreeExecutableBytes(path string, label string, opts islandFreeExecutableSmokeOptions, wantCodes [][]byte, forbidCodes [][]byte) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("%s output is not an ELF executable", label)
	}
	if data[4] != opts.wantClass {
		return fmt.Errorf("%s ELF class = %d, want %d", label, data[4], opts.wantClass)
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != opts.wantMachine {
		return fmt.Errorf("%s ELF machine = %#x, want %#x", label, machine, opts.wantMachine)
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

func checkRawMemoryBoundsExecutableSmoke(opts rawMemoryBoundsExecutableSmokeOptions) error {
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
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, opts.target, BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("%s output is not an ELF executable", opts.label)
	}
	if data[4] != opts.wantClass {
		return fmt.Errorf("%s ELF class = %d, want %d", opts.label, data[4], opts.wantClass)
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != opts.wantMachine {
		return fmt.Errorf("%s ELF machine = %#x, want %#x", opts.label, machine, opts.wantMachine)
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing raw memory bounds sequence % x", opts.label, wantCode)
		}
	}
	for _, forbidCode := range opts.forbidCode {
		if len(forbidCode) > 0 && bytes.Contains(data, forbidCode) {
			return fmt.Errorf("%s contains forbidden raw memory bounds sequence % x", opts.label, forbidCode)
		}
	}
	return nil
}

func checkRawPointerSlotExecutableSmoke(opts rawPointerSlotExecutableSmokeOptions) error {
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
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, opts.target, BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("%s output is not an ELF executable", opts.label)
	}
	if data[4] != opts.wantClass {
		return fmt.Errorf("%s ELF class = %d, want %d", opts.label, data[4], opts.wantClass)
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != opts.wantMachine {
		return fmt.Errorf("%s ELF machine = %#x, want %#x", opts.label, machine, opts.wantMachine)
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing raw pointer slot sequence % x", opts.label, wantCode)
		}
	}
	for _, forbidCode := range opts.forbidCode {
		if len(forbidCode) > 0 && bytes.Contains(data, forbidCode) {
			return fmt.Errorf("%s contains forbidden raw pointer slot sequence % x", opts.label, forbidCode)
		}
	}
	return nil
}

func checkRawPointerOffsetSlotExecutableSmoke(opts rawPointerSlotExecutableSmokeOptions) error {
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
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, opts.target, BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("%s output is not an ELF executable", opts.label)
	}
	if data[4] != opts.wantClass {
		return fmt.Errorf("%s ELF class = %d, want %d", opts.label, data[4], opts.wantClass)
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != opts.wantMachine {
		return fmt.Errorf("%s ELF machine = %#x, want %#x", opts.label, machine, opts.wantMachine)
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing raw pointer offset slot sequence % x", opts.label, wantCode)
		}
	}
	for _, forbidCode := range opts.forbidCode {
		if len(forbidCode) > 0 && bytes.Contains(data, forbidCode) {
			return fmt.Errorf("%s contains forbidden raw pointer offset slot sequence % x", opts.label, forbidCode)
		}
	}
	return nil
}

func checkAllocatorExecutableSmoke(opts allocatorExecutableSmokeOptions) error {
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
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, opts.target, BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("%s output is not an ELF executable", opts.label)
	}
	if data[4] != opts.wantClass {
		return fmt.Errorf("%s ELF class = %d, want %d", opts.label, data[4], opts.wantClass)
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != opts.wantMachine {
		return fmt.Errorf("%s ELF machine = %#x, want %#x", opts.label, machine, opts.wantMachine)
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing target allocator sequence % x", opts.label, wantCode)
		}
	}
	for _, forbidCode := range opts.forbidCode {
		if len(forbidCode) > 0 && bytes.Contains(data, forbidCode) {
			return fmt.Errorf("%s contains forbidden allocator sequence % x", opts.label, forbidCode)
		}
	}
	return nil
}

func checkAllocatorFailureExecutableSmoke(opts allocatorFailureExecutableSmokeOptions) error {
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
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, opts.target, BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("%s output is not an ELF executable", opts.label)
	}
	if data[4] != opts.wantClass {
		return fmt.Errorf("%s ELF class = %d, want %d", opts.label, data[4], opts.wantClass)
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != opts.wantMachine {
		return fmt.Errorf("%s ELF machine = %#x, want %#x", opts.label, machine, opts.wantMachine)
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing allocator failure sequence % x", opts.label, wantCode)
		}
	}
	for _, forbidCode := range opts.forbidCode {
		if len(forbidCode) > 0 && bytes.Contains(data, forbidCode) {
			return fmt.Errorf("%s contains forbidden allocator failure sequence % x", opts.label, forbidCode)
		}
	}
	return nil
}

func checkStderrFDRuntimeSmoke(opts stderrFDRuntimeSmokeOptions) error {
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
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, opts.target, BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("%s output is not an ELF executable", opts.label)
	}
	if data[4] != opts.wantClass {
		return fmt.Errorf("%s ELF class = %d, want %d", opts.label, data[4], opts.wantClass)
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != opts.wantMachine {
		return fmt.Errorf("%s ELF machine = %#x, want %#x", opts.label, machine, opts.wantMachine)
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

func checkStdoutExecutableSmoke(opts stdoutExecutableSmokeOptions) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+opts.stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, opts.stem+".tetra")
	outPath := filepath.Join(tmpDir, opts.stem)
	src := fmt.Sprintf("func main() -> Int\nuses io:\n    print(%q)\n    return 0\n", opts.wantLiteral)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, opts.target, BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("%s output is not an ELF executable", opts.label)
	}
	if data[4] != opts.wantClass {
		return fmt.Errorf("%s ELF class = %d, want %d", opts.label, data[4], opts.wantClass)
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != opts.wantMachine {
		return fmt.Errorf("%s ELF machine = %#x, want %#x", opts.label, machine, opts.wantMachine)
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

func checkSourceNativeScalarDiagnostics(tgt ctarget.Target) error {
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
		_, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{Emit: EmitLibrary, Jobs: 1})
		if err == nil {
			return fmt.Errorf("%s accepted source-level target-layout scalar in %s", tgt.Triple, tc.name)
		}
		for _, want := range []string{
			"target-layout scalar type",
			"not supported in source-level Tetra yet",
			"native-int/codegen support",
		} {
			if !strings.Contains(err.Error(), want) {
				return fmt.Errorf("%s source native scalar diagnostic for %s = %q, want %q", tgt.Triple, tc.name, err.Error(), want)
			}
		}
		if strings.Contains(err.Error(), "unknown type") {
			return fmt.Errorf("%s source native scalar diagnostic for %s fell back to unknown type: %q", tgt.Triple, tc.name, err.Error())
		}
		if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
			return fmt.Errorf("%s source native scalar wrote object %s (stat err=%v)", tgt.Triple, outPath, statErr)
		}
	}
	return nil
}

func checkX64PlatformObjectABISmoke(tgt ctarget.Target) error {
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
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{Emit: EmitLibrary, Jobs: 1}); err != nil {
		return err
	}
	obj, err := ReadObject(outPath)
	if err != nil {
		return err
	}
	if obj.Target != tgt.Triple {
		return fmt.Errorf("target mismatch: got %q want %s", obj.Target, tgt.Triple)
	}
	if !strings.Contains(string(obj.Data), tgt.Triple+" abi\n") {
		return fmt.Errorf("%s object data missing ABI smoke literal: %q", tgt.Triple, string(obj.Data))
	}
	if !abiSuiteObjectHasSymbolSignature(obj, "ffi_say_i32", 0, 1) {
		return fmt.Errorf("%s object missing scalar exported ffi_say_i32 symbol: %#v", tgt.Triple, obj.Symbols)
	}
	if !abiSuiteObjectHasRelocKind(obj, RelocDataDisp32) {
		return fmt.Errorf("%s object missing data displacement relocation: %#v", tgt.Triple, obj.Relocs)
	}
	switch tgt.Triple {
	case "macos-x64":
		if abiSuiteObjectHasRelocKind(obj, RelocIATDisp32) {
			return fmt.Errorf("macos-x64 object unexpectedly has Windows IAT reloc: %#v", obj.Relocs)
		}
	case "windows-x64":
		for _, name := range []string{"kernel32.GetStdHandle", "kernel32.WriteFile"} {
			if !abiSuiteObjectHasReloc(obj, RelocIATDisp32, name) {
				return fmt.Errorf("windows-x64 object missing IAT relocation %q: %#v", name, obj.Relocs)
			}
		}
	default:
		return fmt.Errorf("x64 platform object ABI smoke does not cover %s", tgt.Triple)
	}
	return nil
}

func checkX64PointerFFIRegressionSmoke() error {
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
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{Emit: EmitLibrary, Jobs: 1}); err != nil {
		return err
	}
	obj, err := ReadObject(outPath)
	if err != nil {
		return err
	}
	if obj.Target != "linux-x64" {
		return fmt.Errorf("x64 pointer FFI object target = %q, want linux-x64", obj.Target)
	}
	if !abiSuiteObjectHasSymbolSignature(obj, "ffi_ptr_param_c", 1, 1) {
		return fmt.Errorf("x64 pointer FFI object missing exported ffi_ptr_param_c(1)->1 symbol: %#v", obj.Symbols)
	}
	return nil
}

func checkX64FilesystemSchedulerCompositionSmoke() error {
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
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x64 filesystem scheduler output is not an ELF executable")
	}
	if data[4] != 2 {
		return fmt.Errorf("x64 filesystem scheduler ELF class = %d, want ELFCLASS64", data[4])
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != 0x3e {
		return fmt.Errorf("x64 filesystem scheduler ELF machine = %#x, want EM_X86_64", machine)
	}
	return nil
}

func checkX64NetworkingRuntimeSmoke() error {
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
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x64 networking runtime output is not an ELF executable")
	}
	if data[4] != 2 {
		return fmt.Errorf("x64 networking runtime ELF class = %d, want ELFCLASS64", data[4])
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != 0x3e {
		return fmt.Errorf("x64 networking runtime ELF machine = %#x, want EM_X86_64", machine)
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("x64 networking runtime executable timed out: %q", out.String())
	}
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			return fmt.Errorf("run x64 networking runtime: %w output=%q", err, out.String())
		}
		return fmt.Errorf("x64 networking runtime exit=%d output=%q, want 0", exitErr.ProcessState.ExitCode(), out.String())
	}
	if out.Len() != 0 {
		return fmt.Errorf("x64 networking runtime output=%q, want empty", out.String())
	}
	return nil
}

func checkX64SchedulerRestrictionRegressionSmoke() error {
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
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x64 scheduler regression output is not an ELF executable")
	}
	if data[4] != 2 {
		return fmt.Errorf("x64 scheduler regression ELF class = %d, want ELFCLASS64", data[4])
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != 0x3e {
		return fmt.Errorf("x64 scheduler regression ELF machine = %#x, want EM_X86_64", machine)
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("x64 scheduler regression executable timed out: %q", out.String())
	}
	code := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			return fmt.Errorf("run x64 scheduler regression: %w output=%q", err, out.String())
		}
		code = exitErr.ProcessState.ExitCode()
	}
	if code != 42 {
		return fmt.Errorf("x64 scheduler regression exit=%d output=%q, want 42", code, out.String())
	}
	return nil
}

func abiSuiteObjectHasSymbolSignature(obj *Object, name string, params, returns int) bool {
	for _, sym := range obj.Symbols {
		if sym.Name == name && sym.HasSignature && sym.ParamSlots == params && sym.ReturnSlots == returns {
			return true
		}
	}
	return false
}

func abiSuiteObjectHasRelocKind(obj *Object, kind RelocKind) bool {
	for _, reloc := range obj.Relocs {
		if reloc.Kind == kind {
			return true
		}
	}
	return false
}

func abiSuiteObjectHasReloc(obj *Object, kind RelocKind, name string) bool {
	for _, reloc := range obj.Relocs {
		if reloc.Kind == kind && reloc.Name == name {
			return true
		}
	}
	return false
}

func checkStdlibRuntimeBoundaryDiagnostics(tgt ctarget.Target) error {
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
	if !targetSupportsNetRuntimeSymbols(tgt.Triple, requiredNetRuntimeSymbols()) {
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
		_, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{Jobs: 1})
		if err == nil {
			return fmt.Errorf("%s accepted unsupported %s stdlib runtime boundary", tgt.Triple, tc.runtimeName)
		}
		diag := DiagnosticFromError(err)
		wantMessage := fmt.Sprintf("%s runtime not supported on %s", tc.runtimeName, tgt.Triple)
		if diag.Code != DiagnosticCodeTargetRuntime || diag.Severity != "error" || diag.Message != wantMessage {
			return fmt.Errorf("%s %s runtime diagnostic = %#v, want code=%s severity=error message=%q", tgt.Triple, tc.runtimeName, diag, DiagnosticCodeTargetRuntime, wantMessage)
		}
		if !strings.Contains(diag.Hint, "Build this source for linux-x64") {
			return fmt.Errorf("%s %s runtime hint = %q, want linux-x64 guidance", tgt.Triple, tc.runtimeName, diag.Hint)
		}
		if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
			return fmt.Errorf("%s %s runtime rejection wrote output %s (stat err=%v)", tgt.Triple, tc.runtimeName, outPath, statErr)
		}
	}
	return nil
}

func checkX86TimeRuntimeSmoke() error {
	tmpDir, err := os.MkdirTemp("", "tetra-x86-time-runtime-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, "x86_time_runtime.tetra")
	outPath := filepath.Join(tmpDir, "x86-time-runtime")
	src := `
func main() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    let _until: Int = core.sleep_until(core.deadline_ms(2))
    return core.time_now_ms()
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x86", BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x86 time runtime output is not an ELF executable")
	}
	if data[4] != 1 {
		return fmt.Errorf("x86 time runtime ELF class = %d, want ELFCLASS32", data[4])
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != 0x03 {
		return fmt.Errorf("x86 time runtime ELF machine = %#x, want EM_386", machine)
	}
	return nil
}

func checkX86FilesystemRuntimeSmoke() error {
	tmpDir, err := os.MkdirTemp("", "tetra-x86-filesystem-runtime-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, "x86_filesystem_runtime.tetra")
	outPath := filepath.Join(tmpDir, "x86-filesystem-runtime")
	src := `
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
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x86", BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x86 filesystem runtime output is not an ELF executable")
	}
	if data[4] != 1 {
		return fmt.Errorf("x86 filesystem runtime ELF class = %d, want ELFCLASS32", data[4])
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != 0x03 {
		return fmt.Errorf("x86 filesystem runtime ELF machine = %#x, want EM_386", machine)
	}
	return nil
}

func checkX86FilesystemSchedulerCompositionSmoke() error {
	tmpDir, err := os.MkdirTemp("", "tetra-x86-filesystem-scheduler-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, "x86_filesystem_scheduler.tetra")
	outPath := filepath.Join(tmpDir, "x86-filesystem-scheduler")
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
    return core.task_join_i32(task)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x86", BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x86 filesystem scheduler output is not an ELF executable")
	}
	if data[4] != 1 {
		return fmt.Errorf("x86 filesystem scheduler ELF class = %d, want ELFCLASS32", data[4])
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != 0x03 {
		return fmt.Errorf("x86 filesystem scheduler ELF machine = %#x, want EM_386", machine)
	}
	return nil
}

func checkX32TimeRuntimeSmoke() error {
	src := `
func main() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    let _until: Int = core.sleep_until(core.deadline_ms(2))
    return core.time_now_ms()
`
	return checkX32BuildOnlyRuntimeSmoke("time-runtime", "x32 time runtime", src)
}

func checkX32FilesystemRuntimeSmoke() error {
	tmpDir, err := os.MkdirTemp("", "tetra-x32-filesystem-runtime-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, "x32_filesystem_runtime.tetra")
	outPath := filepath.Join(tmpDir, "x32-filesystem-runtime")
	src := `
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
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x32", BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x32 filesystem runtime output is not an ELF executable")
	}
	if data[4] != 1 {
		return fmt.Errorf("x32 filesystem runtime ELF class = %d, want ELFCLASS32", data[4])
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != 0x3e {
		return fmt.Errorf("x32 filesystem runtime ELF machine = %#x, want EM_X86_64", machine)
	}
	return nil
}

func checkX32SingleTaskSelfHostRuntimeSmoke() error {
	src := `
func worker() -> Int:
    return 41

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`
	return checkX32BuildOnlyRuntimeSmoke("task-runtime", "x32 task runtime", src)
}

func checkX32TypedTaskSelfHostRuntimeSmoke() error {
	src := `
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
	return checkX32BuildOnlyRuntimeSmoke("typed-task-runtime", "x32 typed-task runtime", src)
}

func checkX32StagedTypedTaskSelfHostRuntimeSmoke() error {
	src := `
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
	return checkX32BuildOnlyRuntimeSmoke("staged-typed-task-runtime", "x32 staged typed-task runtime", src)
}

func checkX32TaskGroupSelfHostRuntimeSmoke() error {
	src := `
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
	return checkX32BuildOnlyRuntimeSmoke("task-group-runtime", "x32 task-group runtime", src)
}

func checkX32TypedTaskGroupSelfHostRuntimeSmoke() error {
	src := `
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
	return checkX32BuildOnlyRuntimeSmoke("typed-task-group-runtime", "x32 typed-task-group runtime", src)
}

func checkX32FilesystemSchedulerCompositionSmoke() error {
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
    return core.task_join_i32(task)
`
	return checkX32BuildOnlyRuntimeSmoke("filesystem-scheduler-runtime", "x32 filesystem scheduler runtime", src)
}

func checkX32SingleActorSelfHostRuntimeSmoke() error {
	src := `
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
	return checkX32BuildOnlyRuntimeSmoke("actor-runtime", "x32 actor runtime", src)
}

func checkX32ActorStateSelfHostRuntimeSmoke() error {
	src := `
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
	return checkX32BuildOnlyRuntimeSmoke("actor-state-runtime", "x32 actor-state runtime", src)
}

func checkX32BuildOnlyRuntimeSmoke(stem string, label string, src string) error {
	tmpDir, err := os.MkdirTemp("", "tetra-x32-"+stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, "x32_"+strings.ReplaceAll(stem, "-", "_")+".tetra")
	outPath := filepath.Join(tmpDir, "x32-"+stem)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x32", BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("%s output is not an ELF executable", label)
	}
	if data[4] != 1 {
		return fmt.Errorf("%s ELF class = %d, want ELFCLASS32", label, data[4])
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != 0x3e {
		return fmt.Errorf("%s ELF machine = %#x, want EM_X86_64", label, machine)
	}
	return nil
}

func checkX86SingleTaskSelfHostRuntimeSmoke() error {
	tmpDir, err := os.MkdirTemp("", "tetra-x86-task-runtime-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, "x86_task_runtime.tetra")
	outPath := filepath.Join(tmpDir, "x86-task-runtime")
	src := `
func worker() -> Int:
    return 41

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x86", BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x86 task runtime output is not an ELF executable")
	}
	if data[4] != 1 {
		return fmt.Errorf("x86 task runtime ELF class = %d, want ELFCLASS32", data[4])
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != 0x03 {
		return fmt.Errorf("x86 task runtime ELF machine = %#x, want EM_386", machine)
	}
	return nil
}

func checkX86TypedTaskSelfHostRuntimeSmoke() error {
	tmpDir, err := os.MkdirTemp("", "tetra-x86-typed-task-runtime-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, "x86_typed_task_runtime.tetra")
	outPath := filepath.Join(tmpDir, "x86-typed-task-runtime")
	src := `
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
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x86", BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x86 typed-task runtime output is not an ELF executable")
	}
	if data[4] != 1 {
		return fmt.Errorf("x86 typed-task runtime ELF class = %d, want ELFCLASS32", data[4])
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != 0x03 {
		return fmt.Errorf("x86 typed-task runtime ELF machine = %#x, want EM_386", machine)
	}
	return nil
}

func checkX86StagedTypedTaskSelfHostRuntimeSmoke() error {
	tmpDir, err := os.MkdirTemp("", "tetra-x86-staged-typed-task-runtime-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, "x86_staged_typed_task_runtime.tetra")
	outPath := filepath.Join(tmpDir, "x86-staged-typed-task-runtime")
	src := `
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
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x86", BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x86 staged typed-task runtime output is not an ELF executable")
	}
	if data[4] != 1 {
		return fmt.Errorf("x86 staged typed-task runtime ELF class = %d, want ELFCLASS32", data[4])
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != 0x03 {
		return fmt.Errorf("x86 staged typed-task runtime ELF machine = %#x, want EM_386", machine)
	}
	return nil
}

func checkX86TaskGroupSelfHostRuntimeSmoke() error {
	tmpDir, err := os.MkdirTemp("", "tetra-x86-task-group-runtime-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, "x86_task_group_runtime.tetra")
	outPath := filepath.Join(tmpDir, "x86-task-group-runtime")
	src := `
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
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x86", BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x86 task-group runtime output is not an ELF executable")
	}
	if data[4] != 1 {
		return fmt.Errorf("x86 task-group runtime ELF class = %d, want ELFCLASS32", data[4])
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != 0x03 {
		return fmt.Errorf("x86 task-group runtime ELF machine = %#x, want EM_386", machine)
	}
	return nil
}

func checkX86TypedTaskGroupSelfHostRuntimeSmoke() error {
	tmpDir, err := os.MkdirTemp("", "tetra-x86-typed-task-group-runtime-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, "x86_typed_task_group_runtime.tetra")
	outPath := filepath.Join(tmpDir, "x86-typed-task-group-runtime")
	src := `
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
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x86", BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x86 typed-task-group runtime output is not an ELF executable")
	}
	if data[4] != 1 {
		return fmt.Errorf("x86 typed-task-group runtime ELF class = %d, want ELFCLASS32", data[4])
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != 0x03 {
		return fmt.Errorf("x86 typed-task-group runtime ELF machine = %#x, want EM_386", machine)
	}
	return nil
}

func checkX86SingleActorSelfHostRuntimeSmoke() error {
	tmpDir, err := os.MkdirTemp("", "tetra-x86-actor-runtime-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, "x86_actor_runtime.tetra")
	outPath := filepath.Join(tmpDir, "x86-actor-runtime")
	src := `
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
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x86", BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x86 actor runtime output is not an ELF executable")
	}
	if data[4] != 1 {
		return fmt.Errorf("x86 actor runtime ELF class = %d, want ELFCLASS32", data[4])
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != 0x03 {
		return fmt.Errorf("x86 actor runtime ELF machine = %#x, want EM_386", machine)
	}
	return nil
}

func checkX86ActorStateSelfHostRuntimeSmoke() error {
	tmpDir, err := os.MkdirTemp("", "tetra-x86-actor-state-runtime-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, "x86_actor_state_runtime.tetra")
	outPath := filepath.Join(tmpDir, "x86-actor-state-runtime")
	src := `
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
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x86", BuildOptions{Jobs: 1}); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x86 actor-state runtime output is not an ELF executable")
	}
	if data[4] != 1 {
		return fmt.Errorf("x86 actor-state runtime ELF class = %d, want ELFCLASS32", data[4])
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != 0x03 {
		return fmt.Errorf("x86 actor-state runtime ELF machine = %#x, want EM_386", machine)
	}
	return nil
}

func checkX86CtxSwitchObjectSmoke() error {
	obj, err := linux_x86.CodegenObjectLinuxX86([]ir.IRFunc{{
		Name:        "__tetra_x86_ctx_switch_smoke",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRCtxSwitch},
			{Kind: ir.IRReturn},
		},
	}})
	if err != nil {
		return err
	}
	stub := []byte{0x53, 0x55, 0x56, 0x57, 0x89, 0x20, 0x8B, 0x21, 0x5F, 0x5E, 0x5D, 0x5B, 0xC3}
	if !bytes.Contains(obj.Code, stub) {
		return fmt.Errorf("x86 ctx_switch object missing i386 context stub")
	}
	if !bytes.Contains(obj.Code, []byte{0x31, 0xC0, 0x50}) {
		return fmt.Errorf("x86 ctx_switch object missing zero status continuation")
	}
	return nil
}

func checkX32CtxSwitchObjectSmoke() error {
	obj, err := linux_x32.CodegenObjectLinuxX32([]ir.IRFunc{{
		Name:        "__tetra_x32_ctx_switch_smoke",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRCtxSwitch},
			{Kind: ir.IRReturn},
		},
	}})
	if err != nil {
		return err
	}
	if obj.Target != "linux-x32" {
		return fmt.Errorf("x32 ctx_switch object target = %q, want linux-x32", obj.Target)
	}
	stub := expectedX32CtxSwitchSysVStub()
	if !bytes.Contains(obj.Code, stub) {
		return fmt.Errorf("x32 ctx_switch object missing SysV x86_64 context stub")
	}
	shadow := &x64.Emitter{}
	shadow.SubRspImm32(32)
	if bytes.Contains(obj.Code, shadow.Buf) {
		return fmt.Errorf("x32 ctx_switch object unexpectedly emitted Win64 shadow-space adjustment")
	}
	if !bytes.Contains(obj.Code, []byte{0x31, 0xC0, 0x50}) {
		return fmt.Errorf("x32 ctx_switch object missing zero status continuation")
	}
	return nil
}

func expectedX32CtxSwitchSysVStub() []byte {
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

func checkTargetRuntimeBoundaryDiagnostics(tgt ctarget.Target) error {
	cases, err := targetRuntimeBoundaryCases(tgt)
	if err != nil {
		return err
	}
	return checkRuntimeBoundaryDiagnostics(tgt, "tetra-target-runtime-boundary-*", cases)
}

func checkSurfaceDistributedRuntimeBoundaryDiagnostics(tgt ctarget.Target) error {
	return checkRuntimeBoundaryDiagnostics(tgt, "tetra-surface-distributed-runtime-boundary-*", []targetRuntimeBoundaryCase{
		{
			name: "surface",
			src: `
func main() -> Int
uses surface:
    return core.surface_open("demo", 10, 10)
`,
			wantMessage: "surface runtime not supported on " + tgt.Triple,
		},
		{
			name: "distributed_actors",
			src: `
func main() -> Int
uses actors, runtime:
    return core.actor_node_status(2)
`,
			wantMessage: "distributed actors runtime not supported on " + tgt.Triple,
		},
	})
}

func checkNetworkingRuntimeBoundaryDiagnostics(tgt ctarget.Target) error {
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
		{name: "read", uses: "alloc, capability, io, mem", prelude: "        var buf: []u8 = make_u8(4)\n", expr: "core.net_read(3, buf, 0, 1, cap)"},
		{name: "recv", uses: "alloc, capability, io, mem", prelude: "        var buf: []u8 = make_u8(4)\n", expr: "core.net_recv(3, buf, 0, 1, cap)"},
		{name: "write", uses: "alloc, capability, io, mem", prelude: "        var buf: []u8 = make_u8(4)\n", expr: "core.net_write(3, buf, 0, 1, cap)"},
		{name: "send", uses: "alloc, capability, io, mem", prelude: "        var buf: []u8 = make_u8(4)\n", expr: "core.net_send(3, buf, 0, 1, cap)"},
		{name: "epoll_create", expr: "core.net_epoll_create(cap)"},
		{name: "epoll_ctl_add_read", expr: "core.net_epoll_ctl_add_read(4, 3, cap)"},
		{name: "epoll_ctl_add_read_write", expr: "core.net_epoll_ctl_add_read_write(4, 3, cap)"},
		{name: "epoll_ctl_mod_read", expr: "core.net_epoll_ctl_mod_read(4, 3, cap)"},
		{name: "epoll_ctl_mod_read_write", expr: "core.net_epoll_ctl_mod_read_write(4, 3, cap)"},
		{name: "epoll_ctl_delete", expr: "core.net_epoll_ctl_delete(4, 3, cap)"},
		{name: "epoll_wait_one", expr: "core.net_epoll_wait_one(4, 0, cap)"},
		{name: "epoll_wait_one_into", uses: "alloc, capability, io, mem", prelude: "        var event: []i32 = make_i32(2)\n", expr: "core.net_epoll_wait_one_into(4, event, 0, cap)"},
		{name: "set_nonblocking", expr: "core.net_set_nonblocking(3, cap)"},
		{name: "set_reuseport", expr: "core.net_set_reuseport(3, cap)"},
		{name: "set_tcp_nodelay", expr: "core.net_set_tcp_nodelay(3, cap)"},
		{name: "close", expr: "core.net_close(3, cap)"},
	}
	boundaryCases := make([]targetRuntimeBoundaryCase, 0, len(cases))
	for _, tc := range cases {
		builtinName := tc.expr
		if openParen := strings.IndexByte(builtinName, '('); openParen >= 0 {
			builtinName = builtinName[:openParen]
		}
		if symbol, ok := netRuntimeSymbolForBuiltin(builtinName); ok && targetSupportsNetRuntimeSymbols(tgt.Triple, []string{symbol}) {
			continue
		}
		uses := tc.uses
		if uses == "" {
			uses = "capability, io"
		}
		boundaryCases = append(boundaryCases, targetRuntimeBoundaryCase{
			name: tc.name,
			src: "func main() -> Int\nuses " + uses + ":\n    unsafe:\n        let cap: cap.io = core.cap_io()\n" +
				tc.prelude +
				"        return " + tc.expr + "\n    return 1\n",
			wantMessage: "networking runtime not supported on " + tgt.Triple,
		})
	}
	return checkRuntimeBoundaryDiagnostics(tgt, "tetra-networking-runtime-boundary-*", boundaryCases)
}

func checkRuntimeBoundaryDiagnostics(tgt ctarget.Target, tmpPattern string, cases []targetRuntimeBoundaryCase) error {
	tmpDir, err := os.MkdirTemp("", tmpPattern)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	for _, tc := range cases {
		srcPath := filepath.Join(tmpDir, tc.name+".tetra")
		outPath := filepath.Join(tmpDir, tc.name+"-"+tgt.Triple)
		if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
			return err
		}
		_, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{Jobs: 1})
		if err == nil {
			return fmt.Errorf("%s accepted unsupported %s target runtime boundary", tgt.Triple, tc.name)
		}
		diag := DiagnosticFromError(err)
		if diag.Code != DiagnosticCodeTargetRuntime || diag.Severity != "error" || diag.Message != tc.wantMessage {
			return fmt.Errorf("%s %s runtime diagnostic = %#v, want code=%s severity=error message=%q", tgt.Triple, tc.name, diag, DiagnosticCodeTargetRuntime, tc.wantMessage)
		}
		if !strings.Contains(diag.Hint, "Build this source for linux-x64") {
			return fmt.Errorf("%s %s runtime hint = %q, want linux-x64 guidance", tgt.Triple, tc.name, diag.Hint)
		}
		if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
			return fmt.Errorf("%s %s runtime rejection wrote output %s (stat err=%v)", tgt.Triple, tc.name, outPath, statErr)
		}
	}
	return nil
}

type targetRuntimeBoundaryCase struct {
	name        string
	src         string
	wantMessage string
}

func targetRuntimeBoundaryCases(tgt ctarget.Target) ([]targetRuntimeBoundaryCase, error) {
	switch tgt.Triple {
	case "linux-x86":
		return []targetRuntimeBoundaryCase{
			{
				name: "actor_fanout_over_two_task",
				src: `
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
				wantMessage: "actor fanout above 2 runtime not supported on linux-x86",
			},
			{
				name: "actor_fanout_over_two_actors",
				src: `
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
				wantMessage: "actor fanout above 2 runtime not supported on linux-x86",
			},
			{
				name: "actor_fanout_over_two_task_group",
				src: `
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
				wantMessage: "actor fanout above 2 runtime not supported on linux-x86",
			},
		}, nil
	case "linux-x32":
		return []targetRuntimeBoundaryCase{
			{
				name: "actor_fanout_over_two_actors",
				src: `
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
				wantMessage: "actor fanout above 2 runtime not supported on linux-x32",
			},
			{
				name: "actor_fanout_over_two_task",
				src: `
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
				wantMessage: "actor fanout above 2 runtime not supported on linux-x32",
			},
		}, nil
	default:
		return nil, fmt.Errorf("target runtime boundary suite is not defined for %s", tgt.Triple)
	}
}

func expectTargetScalarLayout(tgt ctarget.Target, name string, size int, align int) error {
	layout, ok := tgt.ScalarLayout(name)
	if !ok {
		return fmt.Errorf("%s missing scalar layout %s", tgt.Triple, name)
	}
	if layout.SizeBytes != size || layout.AlignBytes != align || layout.ABIBytes != size {
		return fmt.Errorf("%s scalar %s layout = size=%d align=%d abi=%d, want %d/%d/%d", tgt.Triple, name, layout.SizeBytes, layout.AlignBytes, layout.ABIBytes, size, align, size)
	}
	return nil
}

func expectX86StackArg(got x86abi.ABILocation, name string, class x86abi.ABIClass, offset int, slot int, size int, align int) error {
	if got.Name != name || got.Class != class || got.Register != "" || got.StackOffsetBytes != offset || got.StackSlotBytes != slot || got.SizeBytes != size || got.AlignBytes != align {
		return fmt.Errorf("x86 %s stack arg = %#v, want class=%s offset=%d slot=%d size=%d align=%d", name, got, class, offset, slot, size, align)
	}
	return nil
}

func expectX64Arg(got x64abi.ABILocation, name string, class x64abi.ABIClass, register string, size int, align int, regWidth int, extend x64abi.ABIExtension) error {
	if got.Name != name || got.Class != class || got.Register != register || got.SizeBytes != size || got.AlignBytes != align || got.RegisterWidthBits != regWidth || got.Extension != extend {
		return fmt.Errorf("x64 %s arg = %#v, want class=%s register=%s size=%d align=%d regWidth=%d extend=%s", name, got, class, register, size, align, regWidth, extend)
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
