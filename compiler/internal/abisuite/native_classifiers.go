package abisuite

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/backend/x86abi"
	ctarget "tetra_language/compiler/target"
)

func CheckX86I386Classifier(tgt ctarget.Target) error {
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

func CheckX32SysVClassifier(tgt ctarget.Target) error {
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
