package callables

import (
	"reflect"
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/semantics"
)

func TestCallableClosureTargetNameQualifiesModule(t *testing.T) {
	caller := semantics.CheckedFunc{Module: "app"}
	funcs := map[string]semantics.FuncSig{"app.worker": {}}

	got := ClosureTargetName(caller, &frontend.ClosureExpr{Name: "worker"}, funcs)
	if got != "app.worker" {
		t.Fatalf("closure target = %q", got)
	}
}

func TestFunctionFieldAndImportTargets(t *testing.T) {
	expr := &frontend.FieldAccessExpr{
		Base: &frontend.FieldAccessExpr{
			Base:  &frontend.IdentExpr{Name: "handler"},
			Field: "nested",
		},
		Field: "callback",
	}
	if name := FunctionTypedFieldNameFromExpr(expr); name != "handler.nested.callback" {
		t.Fatalf("field name = %q", name)
	}

	locals := map[string]semantics.LocalInfo{
		"handler": {
			FunctionFields: map[string]semantics.FunctionFieldInfo{
				"nested.callback": {FunctionValue: "app.on_event"},
			},
		},
	}
	if target, ok := FunctionFieldTargetFromExpr(expr, locals); !ok || target != "app.on_event" {
		t.Fatalf("field target = %q, %v", target, ok)
	}

	importExpr := &frontend.FieldAccessExpr{Base: &frontend.IdentExpr{Name: "math"}, Field: "add"}
	if target, ok := ImportedFunctionTargetFromExpr(
		importExpr,
		map[string]string{"math": "std.math"},
		map[string]semantics.FuncSig{"std.math.add": {}},
	); !ok ||
		target != "std.math.add" {
		t.Fatalf("import target = %q, %v", target, ok)
	}
}

func TestEnumPayloadTargetInfoClonesMetadata(t *testing.T) {
	caseInfo := semantics.EnumCaseInfo{
		Ordinal:                7,
		PayloadFunctionParams:  [][]string{{"i32", "bool"}},
		PayloadFunctionOwns:    [][]string{{"copy", "borrow"}},
		PayloadFunctionReturns: []string{"i32"},
		PayloadFunctionEffects: [][]string{{"read"}},
	}

	info := EnumPayloadTargetInfo(caseInfo, 0, "app.handle")
	if key := EnumPayloadTargetKey(caseInfo.Ordinal, 0); key != "7:0" {
		t.Fatalf("payload key = %q", key)
	}
	if info.FunctionValue != "app.handle" || info.FunctionReturnType != "i32" {
		t.Fatalf("payload info = %#v", info)
	}
	info.FunctionParamTypes[0] = "mutated"
	if reflect.DeepEqual(info.FunctionParamTypes, caseInfo.PayloadFunctionParams[0]) {
		t.Fatalf("payload metadata was not cloned")
	}
}

func TestEnumCaseConstructorInfoUsesResolvedAndShortTypeNames(t *testing.T) {
	types := map[string]*semantics.TypeInfo{
		"pkg.Result": {
			Name: "pkg.Result",
			Kind: semantics.TypeEnum,
			CaseMap: map[string]semantics.EnumCaseInfo{
				"Ok": {Ordinal: 1},
			},
		},
	}

	typeName, caseInfo, ok := EnumCaseConstructorInfoForTargets(
		&frontend.CallExpr{Name: "pkg.Result.Ok"},
		types,
	)
	if !ok || typeName != "pkg.Result" || caseInfo.Ordinal != 1 {
		t.Fatalf("constructor info = %q %#v %v", typeName, caseInfo, ok)
	}

	typeName, caseInfo, ok = EnumCaseConstructorInfoForTargets(
		&frontend.CallExpr{Name: "Result.Ok"},
		types,
	)
	if !ok || typeName != "pkg.Result" || caseInfo.Ordinal != 1 {
		t.Fatalf("short constructor info = %q %#v %v", typeName, caseInfo, ok)
	}
}

func TestTrimAndResolveFunctionFields(t *testing.T) {
	fields := map[string]semantics.FunctionFieldInfo{
		"nested.callback": {FunctionValue: "app.on_event"},
		"other":           {FunctionValue: "app.other"},
	}
	trimmed := TrimFunctionFields(fields, "nested.")
	if len(trimmed) != 1 || trimmed["callback"].FunctionValue != "app.on_event" {
		t.Fatalf("trimmed = %#v", trimmed)
	}

	field, ok, err := ResolveFunctionFieldName(
		"handler.nested.callback",
		map[string]semantics.LocalInfo{
			"handler": {FunctionFields: fields},
		},
	)
	if err != nil || !ok || field.FunctionValue != "app.on_event" {
		t.Fatalf("resolve = %#v %v %v", field, ok, err)
	}
}
