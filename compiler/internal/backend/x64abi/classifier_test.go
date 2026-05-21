package x64abi

import (
	"strings"
	"testing"

	ctarget "tetra_language/compiler/target"
)

func TestX32SysVClassifierKeepsX64RegistersButILP32PointerSlots(t *testing.T) {
	classifier := mustClassifier(t, "x32")
	if classifier.Name() != "x32-sysv" || !classifier.UsesX64Registers() {
		t.Fatalf("x32 classifier identity = %s x64regs=%v", classifier.Name(), classifier.UsesX64Registers())
	}

	plan, err := classifier.ClassifySignature(ABISignature{
		Params: []ABIParam{
			{Name: "p", Type: "ptr"},
			{Name: "n", Type: "usize"},
			{Name: "wide", Type: "u64"},
			{Name: "f", Type: "f64"},
		},
		Return: &ABIParam{Name: "ret", Type: "ptr"},
	})
	if err != nil {
		t.Fatalf("ClassifySignature: %v", err)
	}

	assertArg(t, plan.Params[0], "p", ABIClassInteger, "rdi", 4, 4, 64, ABIExtendZero)
	assertArg(t, plan.Params[1], "n", ABIClassInteger, "rsi", 4, 4, 64, ABIExtendZero)
	assertArg(t, plan.Params[2], "wide", ABIClassInteger, "rdx", 8, 8, 64, ABIExtendNone)
	assertArg(t, plan.Params[3], "f", ABIClassSSE, "xmm0", 8, 8, 128, ABIExtendNone)
	assertArg(t, plan.Return, "ret", ABIClassInteger, "rax", 4, 4, 64, ABIExtendZero)
}

func TestX32ClassifierDiffersFromX64AndRejectsX86(t *testing.T) {
	x64Plan, err := mustClassifier(t, "x64").ClassifySignature(ABISignature{
		Params: []ABIParam{{Name: "p", Type: "ptr"}},
	})
	if err != nil {
		t.Fatalf("x64 ClassifySignature: %v", err)
	}
	x32Plan, err := mustClassifier(t, "x32").ClassifySignature(ABISignature{
		Params: []ABIParam{{Name: "p", Type: "ptr"}},
	})
	if err != nil {
		t.Fatalf("x32 ClassifySignature: %v", err)
	}
	if x64Plan.Params[0].SizeBytes != 8 || x32Plan.Params[0].SizeBytes != 4 {
		t.Fatalf("pointer param sizes x64=%#v x32=%#v", x64Plan.Params[0], x32Plan.Params[0])
	}
	if x64Plan.Params[0].Register != x32Plan.Params[0].Register {
		t.Fatalf("x32 should keep AMD64 integer arg registers: x64=%s x32=%s", x64Plan.Params[0].Register, x32Plan.Params[0].Register)
	}

	if _, err := NewClassifier(mustTarget(t, "x86")); err == nil || !strings.Contains(err.Error(), "x64abi classifier requires x64 ISA") {
		t.Fatalf("NewClassifier(x86) = %v, want x64 ISA rejection", err)
	}
}

func TestX32SysVClassifierUsesEightByteStackSlotsBeyondRegisters(t *testing.T) {
	classifier := mustClassifier(t, "x32")
	params := make([]ABIParam, 7)
	for i := range params {
		params[i] = ABIParam{Name: "p", Type: "ptr"}
	}
	plan, err := classifier.ClassifySignature(ABISignature{Params: params})
	if err != nil {
		t.Fatalf("ClassifySignature: %v", err)
	}
	if got := plan.Params[6]; got.Register != "" || got.StackOffsetBytes != 0 || got.StackSlotBytes != 8 || got.SizeBytes != 4 {
		t.Fatalf("x32 seventh integer arg = %#v, want first 8-byte stack slot carrying 4-byte pointer", got)
	}
}

func TestX32SysVClassifierUsesX32AggregateLayout(t *testing.T) {
	fields := []ctarget.LayoutField{
		{Name: "raw", Type: "ptr"},
		{Name: "count", Type: "usize"},
	}
	x32Plan, err := mustClassifier(t, "x32").ClassifySignature(ABISignature{
		Params: []ABIParam{{Name: "view", Type: "View", Fields: fields}},
		Return: &ABIParam{Name: "ret", Type: "View", Fields: fields},
	})
	if err != nil {
		t.Fatalf("x32 ClassifySignature: %v", err)
	}
	if got := x32Plan.Params[0]; got.SizeBytes != 8 || got.AlignBytes != 4 || got.Class != ABIClassInteger || got.Register != "rdi" || !sameStrings(got.Registers, []string{"rdi"}) {
		t.Fatalf("x32 aggregate param = %#v, want one integer register carrying 8-byte ILP32 aggregate", got)
	}
	if got := x32Plan.Return; got.SizeBytes != 8 || got.AlignBytes != 4 || got.Class != ABIClassInteger || got.Register != "rax" || !sameStrings(got.Registers, []string{"rax"}) {
		t.Fatalf("x32 aggregate return = %#v, want rax", got)
	}

	x64Plan, err := mustClassifier(t, "x64").ClassifySignature(ABISignature{
		Params: []ABIParam{{Name: "view", Type: "View", Fields: fields}},
	})
	if err != nil {
		t.Fatalf("x64 ClassifySignature: %v", err)
	}
	if got := x64Plan.Params[0]; got.SizeBytes != 16 || got.AlignBytes != 8 || !sameStrings(got.Registers, []string{"rdi", "rsi"}) {
		t.Fatalf("x64 aggregate param = %#v, want two integer registers from LP64 layout", got)
	}
}

func TestX32SysVClassifierClassifiesMixedAggregateEightbytes(t *testing.T) {
	plan, err := mustClassifier(t, "x32").ClassifySignature(ABISignature{
		Params: []ABIParam{{
			Name: "mixed",
			Type: "Mixed",
			Fields: []ctarget.LayoutField{
				{Name: "score", Type: "f64"},
				{Name: "raw", Type: "ptr"},
			},
		}},
	})
	if err != nil {
		t.Fatalf("ClassifySignature: %v", err)
	}
	got := plan.Params[0]
	if got.SizeBytes != 16 || got.AlignBytes != 8 || !sameClasses(got.Classes, []ABIClass{ABIClassSSE, ABIClassInteger}) || !sameStrings(got.Registers, []string{"xmm0", "rdi"}) {
		t.Fatalf("mixed aggregate = %#v, want SSE then integer eightbytes in x32 layout", got)
	}
}

func TestX32SysVClassifierUsesMemoryForLargeAggregates(t *testing.T) {
	fields := []ctarget.LayoutField{
		{Name: "a", Type: "ptr"},
		{Name: "b", Type: "ptr"},
		{Name: "c", Type: "ptr"},
		{Name: "d", Type: "ptr"},
		{Name: "e", Type: "ptr"},
	}
	plan, err := mustClassifier(t, "x32").ClassifySignature(ABISignature{
		Params: []ABIParam{{Name: "large", Type: "Large", Fields: fields}},
		Return: &ABIParam{Name: "ret", Type: "Large", Fields: fields},
	})
	if err != nil {
		t.Fatalf("ClassifySignature: %v", err)
	}
	if got := plan.Params[0]; got.Class != ABIClassMemory || got.Register != "" || len(got.Registers) != 0 || got.StackOffsetBytes != 0 || got.StackSlotBytes != 24 || got.SizeBytes != 20 {
		t.Fatalf("large aggregate param = %#v, want stack memory slot rounded to 24 bytes", got)
	}
	if got := plan.Return; got.Class != ABIClassMemory || !got.Indirect || got.Register != "rdi" || got.StackSlotBytes != 0 || got.SizeBytes != 20 {
		t.Fatalf("large aggregate return = %#v, want hidden sret pointer in rdi", got)
	}
}

func TestSysVClassifierUsesMemoryForPackedUnalignedAggregates(t *testing.T) {
	for _, raw := range []string{"x64", "x32"} {
		t.Run(raw, func(t *testing.T) {
			plan, err := mustClassifier(t, raw).ClassifySignature(ABISignature{
				Params: []ABIParam{{
					Name:   "packed",
					Type:   "Packed",
					Packed: true,
					Fields: []ctarget.LayoutField{
						{Name: "tag", Type: "u8"},
						{Name: "raw", Type: "ptr"},
					},
				}},
				Return: &ABIParam{
					Name:   "ret",
					Type:   "Packed",
					Packed: true,
					Fields: []ctarget.LayoutField{
						{Name: "tag", Type: "u8"},
						{Name: "raw", Type: "ptr"},
					},
				},
			})
			if err != nil {
				t.Fatalf("ClassifySignature: %v", err)
			}
			wantSize := 9
			wantSlot := 16
			if raw == "x32" {
				wantSize = 5
				wantSlot = 8
			}
			if got := plan.Params[0]; got.Class != ABIClassMemory || got.Register != "" || len(got.Registers) != 0 || got.StackOffsetBytes != 0 || got.StackSlotBytes != wantSlot || got.SizeBytes != wantSize || got.AlignBytes != 1 {
				t.Fatalf("%s packed param = %#v, want MEMORY stack slot=%d size=%d align=1", raw, got, wantSlot, wantSize)
			}
			if got := plan.Return; got.Class != ABIClassMemory || !got.Indirect || got.Register != "rdi" || got.StackSlotBytes != 0 || got.SizeBytes != wantSize || got.AlignBytes != 1 {
				t.Fatalf("%s packed return = %#v, want hidden sret pointer in rdi size=%d align=1", raw, got, wantSize)
			}
		})
	}
}

func TestSysVVarargsReportALUpperBound(t *testing.T) {
	for _, raw := range []string{"x64", "x32"} {
		t.Run(raw, func(t *testing.T) {
			plan, err := mustClassifier(t, raw).ClassifySignature(ABISignature{
				Variadic:        true,
				FixedParamCount: 1,
				Params: []ABIParam{
					{Name: "fmt", Type: "ptr"},
					{Name: "first", Type: "f64"},
					{Name: "count", Type: "i32"},
					{Name: "second", Type: "f32"},
				},
			})
			if err != nil {
				t.Fatalf("ClassifySignature: %v", err)
			}
			if !plan.Variadic || plan.FixedParamCount != 1 || plan.VarargStartIndex != 1 {
				t.Fatalf("%s variadic metadata = %#v", raw, plan)
			}
			if !plan.SysVRequiresAL || plan.SysV_ALSSERegisterCount != 2 {
				t.Fatalf("%s SysV AL metadata = requires=%v count=%d, want requires=true count=2", raw, plan.SysVRequiresAL, plan.SysV_ALSSERegisterCount)
			}
			if plan.Win64ShadowSpaceBytes != 0 || len(plan.Win64VarargFloatMirrors) != 0 {
				t.Fatalf("%s unexpected Win64 vararg metadata: %#v", raw, plan)
			}
		})
	}
}

func TestWin64VarargsReportShadowSpaceAndFloatMirrors(t *testing.T) {
	plan, err := mustClassifier(t, "windows-x64").ClassifySignature(ABISignature{
		Variadic:        true,
		FixedParamCount: 1,
		Params: []ABIParam{
			{Name: "fmt", Type: "ptr"},
			{Name: "first", Type: "f64"},
			{Name: "count", Type: "i32"},
			{Name: "second", Type: "f32"},
		},
	})
	if err != nil {
		t.Fatalf("ClassifySignature: %v", err)
	}
	if !plan.Variadic || plan.FixedParamCount != 1 || plan.VarargStartIndex != 1 {
		t.Fatalf("Win64 variadic metadata = %#v", plan)
	}
	if plan.Win64ShadowSpaceBytes != 32 {
		t.Fatalf("Win64 shadow space = %d, want 32", plan.Win64ShadowSpaceBytes)
	}
	wantMirrors := []VarargFloatMirror{
		{ParamIndex: 1, XMMRegister: "xmm1", GPRegister: "rdx"},
		{ParamIndex: 3, XMMRegister: "xmm3", GPRegister: "r9"},
	}
	if !sameMirrors(plan.Win64VarargFloatMirrors, wantMirrors) {
		t.Fatalf("Win64 float mirrors = %#v, want %#v", plan.Win64VarargFloatMirrors, wantMirrors)
	}
	if plan.SysVRequiresAL || plan.SysV_ALSSERegisterCount != 0 {
		t.Fatalf("unexpected SysV AL metadata for Win64: %#v", plan)
	}
}

func TestWin64ClassifierPassesSmallAggregatesAsIntegerScalars(t *testing.T) {
	fields := []ctarget.LayoutField{
		{Name: "lo", Type: "u32"},
		{Name: "hi", Type: "u32"},
	}
	plan, err := mustClassifier(t, "windows-x64").ClassifySignature(ABISignature{
		Params: []ABIParam{{Name: "pair", Type: "Pair", Fields: fields}},
		Return: &ABIParam{Name: "ret", Type: "Pair", Fields: fields},
	})
	if err != nil {
		t.Fatalf("ClassifySignature: %v", err)
	}
	if got := plan.Params[0]; got.Class != ABIClassInteger || got.Register != "rcx" || !sameStrings(got.Registers, []string{"rcx"}) || len(got.Classes) != 0 || got.SizeBytes != 8 || got.ABIBytes != 8 || got.Indirect {
		t.Fatalf("Win64 small aggregate param = %#v, want single integer rcx slot", got)
	}
	if got := plan.Return; got.Class != ABIClassInteger || got.Register != "rax" || !sameStrings(got.Registers, []string{"rax"}) || len(got.Classes) != 0 || got.SizeBytes != 8 || got.ABIBytes != 8 || got.Indirect {
		t.Fatalf("Win64 small aggregate return = %#v, want rax", got)
	}
}

func TestWin64ClassifierPassesLargeAggregatesByReference(t *testing.T) {
	fields := []ctarget.LayoutField{
		{Name: "a", Type: "ptr"},
		{Name: "b", Type: "ptr"},
	}
	plan, err := mustClassifier(t, "windows-x64").ClassifySignature(ABISignature{
		Params: []ABIParam{{Name: "wide", Type: "Wide", Fields: fields}},
		Return: &ABIParam{Name: "ret", Type: "Wide", Fields: fields},
	})
	if err != nil {
		t.Fatalf("ClassifySignature: %v", err)
	}
	if got := plan.Params[0]; got.Class != ABIClassMemory || !got.Indirect || got.Register != "rcx" || !sameStrings(got.Registers, []string{"rcx"}) || len(got.Classes) != 0 || got.StackSlotBytes != 0 || got.SizeBytes != 16 || got.ABIBytes != 8 {
		t.Fatalf("Win64 large aggregate param = %#v, want by-reference pointer in rcx", got)
	}
	if got := plan.Return; got.Class != ABIClassMemory || !got.Indirect || got.Register != "rcx" || !sameStrings(got.Registers, []string{"rcx"}) || len(got.Classes) != 0 || got.StackSlotBytes != 0 || got.SizeBytes != 16 || got.ABIBytes != 8 {
		t.Fatalf("Win64 large aggregate return = %#v, want hidden sret pointer in rcx", got)
	}
}

func TestVariadicSignatureRejectsInvalidFixedPrefix(t *testing.T) {
	for _, fixed := range []int{-1, 3} {
		_, err := mustClassifier(t, "x64").ClassifySignature(ABISignature{
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

func assertArg(t *testing.T, got ABILocation, name string, class ABIClass, register string, size int, align int, regWidth int, extend ABIExtension) {
	t.Helper()
	if got.Name != name || got.Class != class || got.Register != register || got.SizeBytes != size || got.AlignBytes != align || got.RegisterWidthBits != regWidth || got.Extension != extend {
		t.Fatalf("%s location = %#v, want class=%s register=%s size=%d align=%d regWidth=%d extend=%s",
			name, got, class, register, size, align, regWidth, extend)
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

func sameClasses(a []ABIClass, b []ABIClass) bool {
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

func sameMirrors(a []VarargFloatMirror, b []VarargFloatMirror) bool {
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
