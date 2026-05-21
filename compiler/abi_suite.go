package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/backend/x86abi"
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
		{name: "x86 pointer/native-libc FFI diagnostics", run: checkX86PointerNativeLibcFFIDiagnostics},
		{name: "x86 source native scalar diagnostics", run: func() error { return checkSourceNativeScalarDiagnostics(tgt) }},
		{name: "x86 stdlib runtime boundary diagnostics", run: func() error { return checkStdlibRuntimeBoundaryDiagnostics(tgt) }},
		{name: "x86 target runtime boundary diagnostics", run: func() error { return checkTargetRuntimeBoundaryDiagnostics(tgt) }},
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
		{name: "x32 pointer/native-libc FFI diagnostics", run: checkX32PointerNativeLibcFFIDiagnostics},
		{name: "x32 source native scalar diagnostics", run: func() error { return checkSourceNativeScalarDiagnostics(tgt) }},
		{name: "x32 stdlib runtime boundary diagnostics", run: func() error { return checkStdlibRuntimeBoundaryDiagnostics(tgt) }},
		{name: "x32 target runtime boundary diagnostics", run: func() error { return checkTargetRuntimeBoundaryDiagnostics(tgt) }},
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

func checkX86PointerNativeLibcFFIDiagnostics() error {
	return checkPointerNativeLibcFFIDiagnostics("linux-x86", "i386", "x86")
}

func checkX32PointerNativeLibcFFIDiagnostics() error {
	return checkPointerNativeLibcFFIDiagnostics("linux-x32", "x32", "x32")
}

func checkPointerNativeLibcFFIDiagnostics(targetName, boundaryName, stem string) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+stem+"-ffi-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "param",
			src:  "@export(\"ffi_ptr_param_c\")\nfunc ffi_ptr_param(p: ptr) -> Int:\n    return 0\n",
			want: "exported function 'ffi_ptr_param' parameter 'p' type 'ptr' requires the " + boundaryName + " pointer C ABI boundary",
		},
		{
			name: "return",
			src:  "@export(\"ffi_ptr_return_c\")\nfunc ffi_ptr_return() -> ptr:\n    return 0\n",
			want: "exported function 'ffi_ptr_return' return type 'ptr' requires the " + boundaryName + " pointer C ABI boundary",
		},
		{
			name: "fnptr_param",
			src:  "@export(\"ffi_fnptr_param_c\")\nfunc ffi_fnptr_param(cb: fn(Int) -> Int) -> Int:\n    return 0\n",
			want: "exported function 'ffi_fnptr_param' parameter 'cb' type 'fnptr' requires the " + boundaryName + " pointer C ABI boundary",
		},
		{
			name: "fnptr_return",
			src:  "func identity(x: Int) -> Int:\n    return x\n\n@export(\"ffi_fnptr_return_c\")\nfunc ffi_fnptr_return() -> fn(Int) -> Int:\n    return identity\n",
			want: "exported function 'ffi_fnptr_return' return type 'fnptr' requires the " + boundaryName + " pointer C ABI boundary",
		},
		{
			name: "usize_param",
			src:  "@export(\"ffi_usize_param_c\")\nfunc ffi_usize_param(n: usize) -> Int:\n    return 0\n",
			want: "exported function 'ffi_usize_param' parameter 'n' type 'usize' requires the " + boundaryName + " pointer C ABI boundary",
		},
		{
			name: "size_t_param",
			src:  "@export(\"ffi_size_t_param_c\")\nfunc ffi_size_t_param(n: size_t) -> Int:\n    return 0\n",
			want: "exported function 'ffi_size_t_param' parameter 'n' type 'size_t' requires the " + boundaryName + " pointer C ABI boundary",
		},
		{
			name: "native_int_return",
			src:  "@export(\"ffi_native_int_return_c\")\nfunc ffi_native_int_return() -> native_int:\n    return 0\n",
			want: "exported function 'ffi_native_int_return' return type 'native_int' requires the " + boundaryName + " pointer C ABI boundary",
		},
		{
			name: "c_long_return",
			src:  "@export(\"ffi_c_long_return_c\")\nfunc ffi_c_long_return() -> c_long:\n    return 0\n",
			want: "exported function 'ffi_c_long_return' return type 'c_long' requires the " + boundaryName + " pointer C ABI boundary",
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
		if !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), boundaryName+" pointer C ABI boundary is not verified on "+targetName) {
			return fmt.Errorf("%s %s pointer FFI diagnostic = %q, want %q", tc.name, stem, err.Error(), tc.want)
		}
		if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
			return fmt.Errorf("%s %s pointer FFI wrote object %s (stat err=%v)", tc.name, stem, outPath, statErr)
		}
	}
	return nil
}

func checkSourceNativeScalarDiagnostics(tgt ctarget.Target) error {
	tmpDir, err := os.MkdirTemp("", "tetra-source-native-scalar-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	cases := []struct {
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
	}{
		{
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
		},
		{
			name:        "networking",
			runtimeName: "networking",
			src: `
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        return core.net_close(fd, cap)
    return 1
`,
		},
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

func checkTargetRuntimeBoundaryDiagnostics(tgt ctarget.Target) error {
	tmpDir, err := os.MkdirTemp("", "tetra-target-runtime-boundary-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	cases, err := targetRuntimeBoundaryCases(tgt)
	if err != nil {
		return err
	}
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
				name: "time",
				src: `
func main() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(1)
    return 0
`,
				wantMessage: "time runtime not supported on linux-x86",
			},
			{
				name: "task",
				src: `
func worker() -> Int:
    return 41

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
				wantMessage: "task runtime not supported on linux-x86",
			},
			{
				name: "actors",
				src: `
func worker() -> Int
uses actors:
    return core.recv()

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let _sent: Int = core.send(peer, 41)
    return core.recv()
`,
				wantMessage: "actors runtime not supported on linux-x86",
			},
			{
				name: "actor_state",
				src: `
actor Counter:
    var count: Int = 0
    func run() -> Int
    uses actors:
        count = count + 1
        return count

func main() -> Int:
    return 0
`,
				wantMessage: "actors runtime not supported on linux-x86",
			},
		}, nil
	case "linux-x32":
		return []targetRuntimeBoundaryCase{
			{
				name: "multi_spawn_actors",
				src: `
func slow() -> Int
uses actors:
    return 1

func fast() -> Int
uses actors:
    return 2

func main() -> Int
uses actors, runtime:
    let _slow: actor = core.spawn("slow")
    let _fast: actor = core.spawn("fast")
    return 0
`,
				wantMessage: "multi-spawn actors runtime not supported on linux-x32",
			},
			{
				name: "multi_spawn_task",
				src: `
func slow() -> Int:
    return 1

func fast() -> Int:
    return 2

func main() -> Int
uses runtime:
    let _slow: task.i32 = core.task_spawn_i32("slow")
    let _fast: task.i32 = core.task_spawn_i32("fast")
    return 0
`,
				wantMessage: "multi-spawn actors runtime not supported on linux-x32",
			},
			{
				name: "task_group",
				src: `
func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    return core.task_group_close(group)
`,
				wantMessage: "task group runtime not supported on linux-x32",
			},
			{
				name: "typed_task",
				src: `
enum TaskErr:
    case stopped

func worker() -> Int throws TaskErr:
    return 42

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.stopped:
        7
`,
				wantMessage: "typed task runtime not supported on linux-x32",
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
