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
