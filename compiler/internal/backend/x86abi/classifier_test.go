package x86abi

import (
	"strings"
	"testing"

	ctarget "tetra_language/compiler/target"
)

func TestI386SysVClassifierUsesStackArguments(t *testing.T) {
	classifier := mustClassifier(t, "x86")
	if classifier.Name() != "i386-sysv" || classifier.StackCleanup() != StackCleanupCaller {
		t.Fatalf("classifier identity = %s cleanup=%s, want i386-sysv caller cleanup", classifier.Name(), classifier.StackCleanup())
	}

	plan, err := classifier.ClassifySignature(ABISignature{
		Params: []ABIParam{
			{Name: "p", Type: "ptr"},
			{Name: "wide", Type: "u64"},
			{Name: "f", Type: "f32"},
		},
		Return: &ABIParam{Name: "ret", Type: "ptr"},
	})
	if err != nil {
		t.Fatalf("ClassifySignature: %v", err)
	}
	assertStackArg(t, plan.Params[0], "p", ABIClassInteger, 0, 4, 4, 4)
	assertStackArg(t, plan.Params[1], "wide", ABIClassInteger, 4, 8, 8, 4)
	assertStackArg(t, plan.Params[2], "f", ABIClassX87, 12, 4, 4, 4)
	if got := plan.Return; got.Register != "eax" || got.SizeBytes != 4 || got.Class != ABIClassInteger || got.Extension != ABIExtendNone {
		t.Fatalf("ptr return = %#v, want eax pointer return without widening extension", got)
	}
}

func TestI386SysVClassifierScalarReturns(t *testing.T) {
	plan, err := mustClassifier(t, "x86").ClassifySignature(ABISignature{
		Return: &ABIParam{Name: "ret", Type: "i64"},
	})
	if err != nil {
		t.Fatalf("ClassifySignature i64: %v", err)
	}
	if got := plan.Return; got.Register != "edx:eax" || !sameStrings(got.Registers, []string{"eax", "edx"}) || got.SizeBytes != 8 || got.Class != ABIClassInteger {
		t.Fatalf("i64 return = %#v, want edx:eax", got)
	}

	plan, err = mustClassifier(t, "x86").ClassifySignature(ABISignature{
		Return: &ABIParam{Name: "ret", Type: "f64"},
	})
	if err != nil {
		t.Fatalf("ClassifySignature f64: %v", err)
	}
	if got := plan.Return; got.Register != "st0" || got.Class != ABIClassX87 || got.RegisterWidthBits != 80 {
		t.Fatalf("f64 return = %#v, want x87 st0", got)
	}
}

func TestI386SysVClassifierStructReturnUsesHiddenSRet(t *testing.T) {
	fields := []ctarget.LayoutField{
		{Name: "tag", Type: "u8"},
		{Name: "raw", Type: "ptr"},
	}
	plan, err := mustClassifier(t, "x86").ClassifySignature(ABISignature{
		Params: []ABIParam{{Name: "value", Type: "Pair", Fields: fields}},
		Return: &ABIParam{Name: "ret", Type: "Pair", Fields: fields},
	})
	if err != nil {
		t.Fatalf("ClassifySignature: %v", err)
	}
	if got := plan.Params[0]; got.Class != ABIClassMemory || got.StackOffsetBytes != 4 || got.StackSlotBytes != 8 || got.SizeBytes != 8 || got.AlignBytes != 4 {
		t.Fatalf("struct param = %#v, want by-value stack memory copy after hidden sret pointer", got)
	}
	if got := plan.Return; got.Class != ABIClassMemory || !got.Indirect || got.Register != "sret@stack+0" || got.StackOffsetBytes != 0 || got.StackSlotBytes != 4 || got.SizeBytes != 8 {
		t.Fatalf("struct return = %#v, want hidden sret pointer at first stack argument", got)
	}
}

func TestI386SysVClassifierRejectsNonX86Targets(t *testing.T) {
	for _, raw := range []string{"x64", "x32"} {
		if _, err := NewClassifier(mustTarget(t, raw)); err == nil || !strings.Contains(err.Error(), "x86abi classifier requires x86 i386-sysv") {
			t.Fatalf("NewClassifier(%s) = %v, want x86 rejection", raw, err)
		}
	}
}

func TestI386SysVVarargsRemainCallerCleanedStackArguments(t *testing.T) {
	plan, err := mustClassifier(t, "x86").ClassifySignature(ABISignature{
		Variadic:        true,
		FixedParamCount: 1,
		Params: []ABIParam{
			{Name: "fmt", Type: "ptr"},
			{Name: "first", Type: "f64"},
			{Name: "count", Type: "i32"},
		},
	})
	if err != nil {
		t.Fatalf("ClassifySignature: %v", err)
	}
	if !plan.Variadic || plan.FixedParamCount != 1 || plan.VarargStartIndex != 1 || plan.StackCleanup != StackCleanupCaller {
		t.Fatalf("i386 variadic metadata = %#v", plan)
	}
	if plan.RegisterVarargs || plan.VarargRegisterSaveBytes != 0 {
		t.Fatalf("i386 varargs unexpectedly require register varargs/save area: %#v", plan)
	}
	assertStackArg(t, plan.Params[0], "fmt", ABIClassInteger, 0, 4, 4, 4)
	assertStackArg(t, plan.Params[1], "first", ABIClassX87, 4, 8, 8, 4)
	assertStackArg(t, plan.Params[2], "count", ABIClassInteger, 12, 4, 4, 4)
}

func TestI386VariadicSignatureRejectsInvalidFixedPrefix(t *testing.T) {
	for _, fixed := range []int{-1, 3} {
		_, err := mustClassifier(t, "x86").ClassifySignature(ABISignature{
			Variadic:        true,
			FixedParamCount: fixed,
			Params:          []ABIParam{{Name: "fmt", Type: "ptr"}, {Name: "value", Type: "i32"}},
		})
		if err == nil || !strings.Contains(err.Error(), "invalid variadic fixed parameter count") {
			t.Fatalf("fixed=%d err=%v, want invalid fixed-prefix diagnostic", fixed, err)
		}
	}
}

func mustClassifier(t *testing.T, raw string) Classifier {
	t.Helper()
	classifier, err := NewClassifier(mustTarget(t, raw))
	if err != nil {
		t.Fatalf("NewClassifier(%s): %v", raw, err)
	}
	return classifier
}

func mustTarget(t *testing.T, raw string) ctarget.Target {
	t.Helper()
	tgt, err := ctarget.Parse(raw)
	if err != nil {
		t.Fatalf("Parse(%s): %v", raw, err)
	}
	return tgt
}

func assertStackArg(t *testing.T, got ABILocation, name string, class ABIClass, offset int, slot int, size int, align int) {
	t.Helper()
	if got.Name != name || got.Class != class || got.Register != "" || got.StackOffsetBytes != offset || got.StackSlotBytes != slot || got.SizeBytes != size || got.AlignBytes != align {
		t.Fatalf("%s stack arg = %#v, want class=%s offset=%d slot=%d size=%d align=%d",
			name, got, class, offset, slot, size, align)
	}
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
