package lower

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
)

func TestVerifyProgramAcceptsLoweredControlFlow(t *testing.T) {
	src := []byte(`
func maybe(flag: Bool) -> Int?:
    if flag:
        return 42
    else:
        return none

func main() -> Int:
    var total: Int = 0
    for i in 0..<6:
        if i == 1:
            continue
        if i == 5:
            break
        total = total + i
    let value: Int? = maybe(true)
    match value:
    case some(x):
        return total + x
    case none:
        return total
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	if err := VerifyProgram(irProg); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}
}

func TestLowerImplicitOptionalCallArgumentUsesCalleeParamSlots(t *testing.T) {
	src := []byte(`
func unwrap(value: Int?) -> Int:
    if let some(x) = value:
        return x
    else:
        return 0

func main() -> Int:
    return unwrap(41)
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	mainFn := irProg.Funcs[irProg.MainIndex]
	for _, instr := range mainFn.Instrs {
		if instr.Kind == ir.IRCall && instr.Name == "unwrap" {
			if instr.ArgSlots != 2 || instr.RetSlots != 1 {
				t.Fatalf("unwrap call ABI = args %d rets %d, want args 2 rets 1", instr.ArgSlots, instr.RetSlots)
			}
			return
		}
	}
	t.Fatalf("main did not lower an unwrap call: %#v", mainFn.Instrs)
}

func TestVerifyFuncRejectsUnbalancedReturnStack(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "bad_return",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "return expects 1 stack slots") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsEmptyReturningFunction(t *testing.T) {
	fn := ir.IRFunc{Name: "empty", ReturnSlots: 1}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "empty body cannot produce 1 return slots") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyProgramRejectsInvalidMainMetadata(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 1,
		MainName:  "main",
		Funcs: []ir.IRFunc{
			{Name: "main", Instrs: []ir.IRInstr{{Kind: ir.IRReturn}}},
		},
	}
	err := VerifyProgram(prog)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "main index 1 out of bounds") {
		t.Fatalf("error = %v", err)
	}
	diag, ok := frontend.DiagnosticForError(err)
	if !ok {
		t.Fatalf("expected diagnostic error, got %T", err)
	}
	if diag.Code != DiagnosticCodeIRVerifier || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestVerifyProgramRejectsDuplicateFunctionNames(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{
			{Name: "main", Instrs: []ir.IRInstr{{Kind: ir.IRReturn}}},
			{Name: "main", Instrs: []ir.IRInstr{{Kind: ir.IRReturn}}},
		},
	}
	err := VerifyProgram(prog)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), `duplicate function name "main"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyProgramRejectsKnownFunctionCallSignatureMismatch(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{
			{
				Name:        "main",
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRCall, Name: "helper", ArgSlots: 0, RetSlots: 1},
					{Kind: ir.IRReturn},
				},
			},
			{
				Name:        "helper",
				ParamSlots:  1,
				LocalSlots:  1,
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRLoadLocal, Local: 0},
					{Kind: ir.IRReturn},
				},
			},
		},
	}
	err := VerifyProgram(prog)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), `call "helper" ABI mismatch`) {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsInvalidSlotMetadata(t *testing.T) {
	tests := []struct {
		name string
		fn   ir.IRFunc
		want string
	}{
		{
			name: "negative_param_slots",
			fn:   ir.IRFunc{Name: "bad_param", ParamSlots: -1},
			want: "negative slot metadata",
		},
		{
			name: "negative_return_slots",
			fn:   ir.IRFunc{Name: "bad_return_slots", ReturnSlots: -1},
			want: "negative slot metadata",
		},
		{
			name: "params_exceed_locals",
			fn:   ir.IRFunc{Name: "bad_locals", ParamSlots: 2, LocalSlots: 1},
			want: "param slots 2 exceed locals 1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyFunc(tt.fn)
			if err == nil {
				t.Fatalf("expected verifier error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestVerifyFuncRejectsUnknownBranchLabel(t *testing.T) {
	fn := ir.IRFunc{
		Name: "bad_label",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRJmp, Label: 99},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "unknown label 99") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsNegativeBranchLabel(t *testing.T) {
	fn := ir.IRFunc{
		Name: "bad_negative_label",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRJmp, Label: -1},
			{Kind: ir.IRLabel, Label: -1},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "negative label -1") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsLocalSlotOutOfBounds(t *testing.T) {
	pos := frontend.Position{File: "bad_lower.t4", Line: 7, Col: 5}
	fn := ir.IRFunc{
		Name:       "bad_local",
		LocalSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 1, Pos: pos},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "local slot 1 out of bounds") {
		t.Fatalf("error = %v", err)
	}
	diag, ok := frontend.DiagnosticForError(err)
	if !ok {
		t.Fatalf("expected diagnostic error, got %T", err)
	}
	if diag.Code != DiagnosticCodeIRVerifier || diag.File != "bad_lower.t4" || diag.Line != 7 || diag.Column != 5 {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestVerifyFuncRejectsNegativeGlobalSlotOperands(t *testing.T) {
	tests := []struct {
		name string
		fn   ir.IRFunc
	}{
		{
			name: "load_global",
			fn: ir.IRFunc{
				Name:        "bad_load_global",
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRLoadGlobal, Local: -1},
					{Kind: ir.IRReturn},
				},
			},
		},
		{
			name: "store_global",
			fn: ir.IRFunc{
				Name: "bad_store_global",
				Instrs: []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 7},
					{Kind: ir.IRStoreGlobal, Local: -1},
					{Kind: ir.IRReturn},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyFunc(tt.fn)
			if err == nil {
				t.Fatalf("expected verifier error")
			}
			if !strings.Contains(err.Error(), "global slot -1 out of bounds") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestLowerRunsProgramLevelVerifier(t *testing.T) {
	prog, err := frontend.Parse([]byte("func main() -> Int:\n    return 0\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	checked.MainIndex = len(checked.Funcs) + 1

	_, err = Lower(checked)
	if err == nil {
		t.Fatalf("expected program verifier error")
	}
	diag, ok := frontend.DiagnosticForError(err)
	if !ok {
		t.Fatalf("expected diagnostic error, got %T", err)
	}
	if diag.Code != DiagnosticCodeIRVerifier {
		t.Fatalf("diagnostic = %#v", diag)
	}
	if !strings.Contains(diag.Message, "main index") {
		t.Fatalf("message = %q", diag.Message)
	}
}

func TestVerifyFuncRejectsCallStackUnderflow(t *testing.T) {
	fn := ir.IRFunc{
		Name: "bad_call",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRCall, Name: "callee", ArgSlots: 2, RetSlots: 0},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "stack underflow") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsInconsistentBranchStack(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "bad_join",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRConstI32},
			{Kind: ir.IRJmp, Label: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32},
			{Kind: ir.IRConstI32},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "inconsistent stack height") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsUnknownInstructionKind(t *testing.T) {
	fn := ir.IRFunc{
		Name: "bad_kind",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRInstrKind(999)},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "unknown instruction kind 999") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsNegativeCallABISlots(t *testing.T) {
	fn := ir.IRFunc{
		Name: "bad_call_abi",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRCall, Name: "callee", ArgSlots: -1, RetSlots: 0},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), `call "callee" has negative ABI slots`) {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsKnownRuntimeCallSignatureMismatch(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "bad_runtime_call",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32},
			{Kind: ir.IRCall, Name: "__tetra_actor_spawn", ArgSlots: 0, RetSlots: 0},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), `runtime call "__tetra_actor_spawn" ABI mismatch`) {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsTypedTaskRuntimeCallSignatureMismatch(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "bad_typed_task_runtime_call",
		ReturnSlots: 5,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32},
			{Kind: ir.IRConstI32},
			{Kind: ir.IRConstI32},
			{Kind: ir.IRConstI32},
			{Kind: ir.IRConstI32},
			{Kind: ir.IRCall, Name: "__tetra_task_join_typed_5", ArgSlots: 5, RetSlots: 5},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), `runtime call "__tetra_task_join_typed_5" ABI mismatch`) {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncAcceptsKnownRuntimeCallSignature(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "runtime_call",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32},
			{Kind: ir.IRCall, Name: "__tetra_actor_spawn", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRReturn},
		},
	}
	if err := VerifyFunc(fn); err != nil {
		t.Fatalf("VerifyFunc: %v", err)
	}
}

func TestVerifyFuncRejectsMissingNamedOperands(t *testing.T) {
	tests := []struct {
		name string
		fn   ir.IRFunc
		want string
	}{
		{
			name: "call",
			fn: ir.IRFunc{
				Name: "bad_call_name",
				Instrs: []ir.IRInstr{
					{Kind: ir.IRCall, ArgSlots: 0, RetSlots: 0},
					{Kind: ir.IRReturn},
				},
			},
			want: "call is missing target name",
		},
		{
			name: "symbol_address",
			fn: ir.IRFunc{
				Name:        "bad_symbol_name",
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRSymAddr},
					{Kind: ir.IRReturn},
				},
			},
			want: "symbol address is missing name",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyFunc(tt.fn)
			if err == nil {
				t.Fatalf("expected verifier error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestVerifyFuncAcceptsPolicyGuardShape(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "guarded",
		ParamSlots:  1,
		LocalSlots:  2,
		ReturnSlots: 1,
		Policy: ir.IRPolicy{
			HasBudget:    true,
			Budget:       3,
			BudgetLocal:  1,
			HasConsent:   true,
			ConsentLocal: 0,
			FailLabel:    1,
		},
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: consentTokenRuntimeSentinel},
			{Kind: ir.IRCmpEqI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRSubI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpGeI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRCall, Name: "callee", ArgSlots: 0, RetSlots: 1},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
	if err := VerifyFunc(fn); err != nil {
		t.Fatalf("VerifyFunc: %v", err)
	}
}

func TestVerifyFuncRejectsMissingBudgetGuardBeforeChargedInstruction(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "bad_budget_guard",
		LocalSlots:  1,
		ReturnSlots: 1,
		Policy: ir.IRPolicy{
			HasBudget:   true,
			Budget:      1,
			BudgetLocal: 0,
			FailLabel:   1,
		},
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRCall, Name: "callee", ArgSlots: 0, RetSlots: 1},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "missing budget guard") {
		t.Fatalf("error = %v", err)
	}
}

func TestStackEffectCoversEveryIRInstrKind(t *testing.T) {
	for kind := ir.IRWrite; kind < ir.IRInstrKindCount; kind++ {
		_, _, known := stackEffect(ir.IRInstr{Kind: kind, ArgSlots: 1, RetSlots: 1})
		if !known {
			t.Fatalf("missing stack effect for IR kind %d", kind)
		}
	}
}

func TestBudgetChargeModelIsExplicit(t *testing.T) {
	charged := map[ir.IRInstrKind]int32{
		ir.IRWrite:              1,
		ir.IRCall:               1,
		ir.IRAllocBytes:         1,
		ir.IRMakeSliceU8:        1,
		ir.IRMakeSliceU16:       1,
		ir.IRMakeSliceI32:       1,
		ir.IRIndexLoadI32:       1,
		ir.IRIndexStoreI32:      1,
		ir.IRIndexLoadU8:        1,
		ir.IRIndexStoreU8:       1,
		ir.IRIndexLoadU16:       1,
		ir.IRIndexStoreU16:      1,
		ir.IRIslandNew:          1,
		ir.IRIslandMakeSliceU8:  1,
		ir.IRIslandMakeSliceU16: 1,
		ir.IRIslandMakeSliceI32: 1,
		ir.IRIslandFree:         1,
		ir.IRCapIO:              1,
		ir.IRCapMem:             1,
		ir.IRMemReadI32:         1,
		ir.IRMemWriteI32:        1,
		ir.IRMemReadU8:          1,
		ir.IRMemWriteU8:         1,
		ir.IRMemReadPtr:         1,
		ir.IRMemWritePtr:        1,
		ir.IRMemReadI32Offset:   1,
		ir.IRMemWriteI32Offset:  1,
		ir.IRMemReadU8Offset:    1,
		ir.IRMemWriteU8Offset:   1,
		ir.IRMemReadPtrOffset:   1,
		ir.IRMemWritePtrOffset:  1,
		ir.IRPtrAdd:             1,
		ir.IRMmioReadI32:        1,
		ir.IRMmioWriteI32:       1,
		ir.IRSymAddr:            1,
		ir.IRCtxSwitch:          1,
	}
	for kind, want := range charged {
		got, ok := budgetChargeForInstr(kind)
		if !ok {
			t.Fatalf("kind %v missing from budget charge model", kind)
		}
		if got != want {
			t.Fatalf("kind %v cost = %d, want %d", kind, got, want)
		}
	}

	uncharged := []ir.IRInstrKind{
		ir.IRStrLit,
		ir.IRConstI32,
		ir.IRLoadLocal,
		ir.IRStoreLocal,
		ir.IRLoadGlobal,
		ir.IRStoreGlobal,
		ir.IRAddI32,
		ir.IRSubI32,
		ir.IRNegI32,
		ir.IRCmpEqI32,
		ir.IRCmpLtI32,
		ir.IRMulI32,
		ir.IRDivI32,
		ir.IRModI32,
		ir.IRCmpGtI32,
		ir.IRCmpGeI32,
		ir.IRCmpLeI32,
		ir.IRCmpNeI32,
		ir.IRLabel,
		ir.IRJmp,
		ir.IRJmpIfZero,
		ir.IRReturn,
	}
	classified := make(map[ir.IRInstrKind]string, len(charged)+len(uncharged))
	for kind := range charged {
		classified[kind] = "charged"
	}
	for _, kind := range uncharged {
		if previous, exists := classified[kind]; exists {
			t.Fatalf("kind %d classified as both %s and uncharged", kind, previous)
		}
		classified[kind] = "uncharged"
		if got, ok := budgetChargeForInstr(kind); ok {
			t.Fatalf("kind %v unexpectedly charged with cost %d", kind, got)
		}
	}
	for kind := ir.IRWrite; kind < ir.IRInstrKindCount; kind++ {
		if _, ok := classified[kind]; !ok {
			t.Fatalf("missing budget charge classification for IR kind %d", kind)
		}
	}
}

func TestVerifyFuncRejectsMissingBudgetGuardBeforeIndexedAccess(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "bad_index_budget_guard",
		LocalSlots:  1,
		ReturnSlots: 1,
		Policy: ir.IRPolicy{
			HasBudget:   true,
			Budget:      1,
			BudgetLocal: 0,
			FailLabel:   1,
		},
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 100},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRIndexLoadI32},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "missing budget guard") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsMalformedConsentGuardShape(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "bad_consent_guard",
		ParamSlots:  1,
		LocalSlots:  1,
		ReturnSlots: 1,
		Policy: ir.IRPolicy{
			HasConsent:   true,
			ConsentLocal: 0,
			FailLabel:    1,
		},
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpNeI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "malformed consent guard") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsUnreachableUnknownInstructionKind(t *testing.T) {
	fn := ir.IRFunc{
		Name: "bad_unreachable_kind",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRInstrKind(999)},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "instr 1: unknown instruction kind 999") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsUnreachableLinearStackUnderflow(t *testing.T) {
	fn := ir.IRFunc{
		Name: "bad_unreachable_stack",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "linear stack underflow") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsDuplicateBranchLabelsWithStableDiagnostic(t *testing.T) {
	pos := frontend.Position{File: "duplicate_label.t4", Line: 4, Col: 9}
	fn := ir.IRFunc{
		Name: "bad_duplicate_label",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLabel, Label: 1, Pos: pos},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected duplicate label verifier error")
	}
	if !strings.Contains(err.Error(), "duplicate label 1") {
		t.Fatalf("error = %v", err)
	}
	diag, ok := frontend.DiagnosticForError(err)
	if !ok {
		t.Fatalf("expected diagnostic error, got %T", err)
	}
	if diag.Code != DiagnosticCodeIRVerifier || diag.File != "duplicate_label.t4" || diag.Line != 4 || diag.Column != 9 || diag.Hint == "" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestVerifyProgramAcceptsRepresentativeLoweringFamilies(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "typed_errors",
			src: `
enum E:
    case bad

func fail(flag: Bool) -> Int throws E:
    if flag:
        return 7
    throw E.bad

func caller(flag: Bool) -> Int throws E:
    return try fail(flag)

func main() -> Int:
    return 0
`,
		},
		{
			name: "tasks",
			src: `
func worker() -> Int:
    return 3

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
		},
		{
			name: "actors",
			src: `
func worker() -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let _: Int = core.send(peer, 1)
    return 0
`,
		},
		{
			name: "unsafe_budget_guards",
			src: `
func main() -> Int
uses alloc, budget, capability, mem
budget(32):
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let _: Int = core.store_i32(p, 9, mem)
        return core.load_i32(p, mem)
    return 0
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := frontend.Parse([]byte(tt.src))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			checked, err := semantics.Check(prog)
			if err != nil {
				t.Fatalf("Check: %v", err)
			}
			irProg, err := Lower(checked)
			if err != nil {
				t.Fatalf("Lower: %v", err)
			}
			if err := VerifyProgram(irProg); err != nil {
				t.Fatalf("VerifyProgram: %v", err)
			}
		})
	}
}
