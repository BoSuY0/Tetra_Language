package opt

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"tetra_language/compiler/internal/backend/linux_x64"
	"tetra_language/compiler/internal/format/elf"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/linker"
)

func TestBasicScalarPassDoesNotReuseKnownLocalUnaryNegExpressionAfterSourceMutation(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  4,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: -6},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: -6},
				{Kind: ir.IRStoreLocal, Local: 3},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRNegI32},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRConstI32, Imm: -5},
				{Kind: ir.IRStoreLocal, Local: 3},
				{Kind: ir.IRLoadLocal, Local: 3},
				{Kind: ir.IRNegI32},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		}},
	}
	before := cloneProgram(prog)

	report, err := NewManager().Run(prog, BasicScalarPass())
	if err != nil {
		t.Fatalf("Run BasicScalarPass: %v", err)
	}
	after := report.Passes[0].AfterDump
	if got := countDumpOccurrences(after, "neg_i32"); got != 2 {
		t.Fatalf("optimized neg_i32 count = %d, want mutated source expression preserved; dump:\n%s", got, after)
	}
	if !strings.Contains(after, "load_local local:3\n  neg_i32") {
		t.Fatalf("mutated known-local unary expression was not preserved:\n%s", after)
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "before-mutated-known-local-unary-gvn")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-mutated-known-local-unary-gvn")
	if beforeExit != afterExit || afterExit != 11 {
		t.Fatalf("native exits before=%d after=%d want 11", beforeExit, afterExit)
	}
}

func TestBasicScalarPassDoesNotReuseMinIntKnownLocalUnaryNegExpression(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  4,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: -2147483648},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: -2147483648},
				{Kind: ir.IRStoreLocal, Local: 3},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRNegI32},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRLoadLocal, Local: 3},
				{Kind: ir.IRNegI32},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		}},
	}

	report, err := NewManager().Run(prog, BasicScalarPass())
	if err != nil {
		t.Fatalf("Run BasicScalarPass: %v", err)
	}
	after := report.Passes[0].AfterDump
	if got := countDumpOccurrences(after, "neg_i32"); got != 2 {
		t.Fatalf("optimized min-int neg_i32 count = %d, want both unsafe unary expressions preserved; dump:\n%s", got, after)
	}
	if !strings.Contains(after, "load_local local:3\n  neg_i32") {
		t.Fatalf("min-int known-local unary expression was not preserved:\n%s", after)
	}
}

func TestBasicScalarPassEliminatesMirroredComparisonExpressionWithGVN(t *testing.T) {
	tests := []struct {
		name       string
		first      ir.IRInstrKind
		firstOp    string
		mirror     ir.IRInstrKind
		mirrorOp   string
		beforeExit int
	}{
		{name: "less-than-greater-than", first: ir.IRCmpLtI32, firstOp: "cmp_lt_i32", mirror: ir.IRCmpGtI32, mirrorOp: "cmp_gt_i32", beforeExit: 2},
		{name: "less-equal-greater-equal", first: ir.IRCmpLeI32, firstOp: "cmp_le_i32", mirror: ir.IRCmpGeI32, mirrorOp: "cmp_ge_i32", beforeExit: 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog := &ir.IRProgram{
				MainIndex: 0,
				MainName:  "main",
				Funcs: []ir.IRFunc{{
					Name:        "main",
					LocalSlots:  3,
					ReturnSlots: 1,
					Instrs: []ir.IRInstr{
						{Kind: ir.IRConstI32, Imm: 1},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: 2},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: ir.IRLoadLocal, Local: 1},
						{Kind: tc.first},
						{Kind: ir.IRStoreLocal, Local: 2},
						{Kind: ir.IRLoadLocal, Local: 1},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: tc.mirror},
						{Kind: ir.IRLoadLocal, Local: 2},
						{Kind: ir.IRAddI32},
						{Kind: ir.IRReturn},
					},
				}},
			}
			before := cloneProgram(prog)

			report, err := NewManager().Run(prog, BasicScalarPass())
			if err != nil {
				t.Fatalf("Run BasicScalarPass: %v", err)
			}
			after := report.Passes[0].AfterDump
			if got := countDumpOccurrences(after, tc.firstOp); got != 1 {
				t.Fatalf("optimized %s count = %d, want original comparison only; dump:\n%s", tc.firstOp, got, after)
			}
			if strings.Contains(after, tc.mirrorOp) {
				t.Fatalf("mirrored comparison was recomputed instead of reusing cached local:\n%s", after)
			}
			if got := countDumpOccurrences(after, "load_local local:2"); got != 2 {
				t.Fatalf("cached comparison loads = %d, want mirrored comparison to reuse local 2; dump:\n%s", got, after)
			}
			beforeExit := runOptLinuxX64(t, before.Funcs, "before-mirrored-"+tc.name+"-gvn")
			afterExit := runOptLinuxX64(t, prog.Funcs, "after-mirrored-"+tc.name+"-gvn")
			if beforeExit != afterExit || afterExit != tc.beforeExit {
				t.Fatalf("native exits before=%d after=%d want %d", beforeExit, afterExit, tc.beforeExit)
			}
		})
	}
}

func TestBasicScalarPassEliminatesSafeConstDenominatorDivModExpressionWithCSE(t *testing.T) {
	tests := []struct {
		name      string
		kind      ir.IRInstrKind
		opName    string
		denom     int32
		wantExit  int
		finalKind ir.IRInstrKind
	}{
		{name: "division", kind: ir.IRDivI32, opName: "div_i32", denom: 3, wantExit: 8, finalKind: ir.IRAddI32},
		{name: "modulo", kind: ir.IRModI32, opName: "mod_i32", denom: 5, wantExit: 4, finalKind: ir.IRAddI32},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog := &ir.IRProgram{
				MainIndex: 0,
				MainName:  "main",
				Funcs: []ir.IRFunc{{
					Name:        "main",
					LocalSlots:  2,
					ReturnSlots: 1,
					Instrs: []ir.IRInstr{
						{Kind: ir.IRConstI32, Imm: 12},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: tc.denom},
						{Kind: tc.kind},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: tc.denom},
						{Kind: tc.kind},
						{Kind: ir.IRLoadLocal, Local: 1},
						{Kind: tc.finalKind},
						{Kind: ir.IRReturn},
					},
				}},
			}
			before := cloneProgram(prog)

			report, err := NewManager().Run(prog, BasicScalarPass())
			if err != nil {
				t.Fatalf("Run BasicScalarPass: %v", err)
			}
			after := report.Passes[0].AfterDump
			if got := countDumpOccurrences(after, tc.opName); got != 1 {
				t.Fatalf("optimized %s count = %d, want one safe cached expression; dump:\n%s", tc.opName, got, after)
			}
			if got := countDumpOccurrences(after, "const_i32 "+strconv.Itoa(int(tc.denom))); got != 1 {
				t.Fatalf("optimized denominator const count = %d, want one cached expression input; dump:\n%s", got, after)
			}
			if got := countDumpOccurrences(after, "load_local local:1"); got != 2 {
				t.Fatalf("optimized cached-expression loads = %d, want repeated safe expression to reuse local 1; dump:\n%s", got, after)
			}
			beforeExit := runOptLinuxX64(t, before.Funcs, "before-safe-"+tc.name+"-cse")
			afterExit := runOptLinuxX64(t, prog.Funcs, "after-safe-"+tc.name+"-cse")
			if beforeExit != afterExit || afterExit != tc.wantExit {
				t.Fatalf("native exits before=%d after=%d want %d", beforeExit, afterExit, tc.wantExit)
			}
		})
	}
}

func TestBasicScalarPassEliminatesRepeatedUnaryLocalNegExpressionWithCSE(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: -6},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRNegI32},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRNegI32},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		}},
	}
	before := cloneProgram(prog)

	report, err := NewManager().Run(prog, BasicScalarPass())
	if err != nil {
		t.Fatalf("Run BasicScalarPass: %v", err)
	}
	after := report.Passes[0].AfterDump
	if got := countDumpOccurrences(after, "neg_i32"); got != 1 {
		t.Fatalf("optimized neg_i32 count = %d, want one cached unary expression; dump:\n%s", got, after)
	}
	if got := countDumpOccurrences(after, "load_local local:1"); got != 2 {
		t.Fatalf("optimized cached unary-expression loads = %d, want repeated neg to reuse local 1; dump:\n%s", got, after)
	}
	if strings.Contains(after, "load_local local:0\n  neg_i32\n  load_local local:1") {
		t.Fatalf("second unary expression still recomputed before cached local load:\n%s", after)
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "before-unary-neg-cse")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-unary-neg-cse")
	if beforeExit != afterExit || afterExit != 12 {
		t.Fatalf("native exits before=%d after=%d want 12", beforeExit, afterExit)
	}
}

func TestBasicScalarPassDoesNotCSEUnsafeConstDenominatorDivModExpression(t *testing.T) {
	tests := []struct {
		name  string
		kind  ir.IRInstrKind
		op    string
		denom int32
	}{
		{name: "division by zero", kind: ir.IRDivI32, op: "div_i32", denom: 0},
		{name: "division by minus one", kind: ir.IRDivI32, op: "div_i32", denom: -1},
		{name: "modulo by zero", kind: ir.IRModI32, op: "mod_i32", denom: 0},
		{name: "modulo by minus one", kind: ir.IRModI32, op: "mod_i32", denom: -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog := &ir.IRProgram{
				MainIndex: 0,
				MainName:  "main",
				Funcs: []ir.IRFunc{{
					Name:        "main",
					LocalSlots:  2,
					ReturnSlots: 1,
					Instrs: []ir.IRInstr{
						{Kind: ir.IRConstI32, Imm: 12},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: tc.denom},
						{Kind: tc.kind},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: tc.denom},
						{Kind: tc.kind},
						{Kind: ir.IRLoadLocal, Local: 1},
						{Kind: ir.IRAddI32},
						{Kind: ir.IRReturn},
					},
				}},
			}

			report, err := NewManager().Run(prog, BasicScalarPass())
			if err != nil {
				t.Fatalf("Run BasicScalarPass: %v", err)
			}
			after := report.Passes[0].AfterDump
			if got := countDumpOccurrences(after, tc.op); got != 2 {
				t.Fatalf("optimized unsafe %s count = %d, want both expressions preserved; dump:\n%s", tc.op, got, after)
			}
		})
	}
}

func TestBasicScalarPassDoesNotReuseStaleLocalConstantExpression(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  3,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRConstI32, Imm: 3},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		}},
	}

	report, err := NewManager().Run(prog, BasicScalarPass())
	if err != nil {
		t.Fatalf("Run BasicScalarPass: %v", err)
	}
	after := report.Passes[0].AfterDump
	if got := countDumpOccurrences(after, "const_i32 7"); got != 2 {
		t.Fatalf("optimized const count = %d, want both local-constant expressions preserved after operand mutation; dump:\n%s", got, after)
	}
	if got := countDumpOccurrences(after, "add_i32"); got != 3 {
		t.Fatalf("optimized add count = %d, want first expression, second expression, and final add; dump:\n%s", got, after)
	}
	if got := countDumpOccurrences(after, "load_local local:2"); got != 1 {
		t.Fatalf("cached local loads = %d, want only final use of local 2 after operand mutation; dump:\n%s", got, after)
	}
}

func TestBasicScalarPassDoesNotTreatNonCommutativeExpressionAsGVN(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  2,
			LocalSlots:  3,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRSubI32},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRSubI32},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		}},
	}

	report, err := NewManager().Run(prog, BasicScalarPass())
	if err != nil {
		t.Fatalf("Run BasicScalarPass: %v", err)
	}
	after := report.Passes[0].AfterDump
	if got := countDumpOccurrences(after, "sub_i32"); got != 2 {
		t.Fatalf("optimized sub count = %d, want both ordered sub expressions preserved; dump:\n%s", got, after)
	}
	if !strings.Contains(after, "load_local local:1\n  load_local local:0\n  sub_i32") {
		t.Fatalf("swapped non-commutative expression was not preserved:\n%s", after)
	}
}

func TestBasicScalarPassDifferentialExecution(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	before := scalarDifferentialProgram()
	after := cloneProgram(before)
	if _, err := NewManager().Run(after, BasicScalarPass()); err != nil {
		t.Fatalf("Run BasicScalarPass: %v", err)
	}

	beforeExit := runOptLinuxX64(t, before.Funcs, "before-basic-scalar")
	afterExit := runOptLinuxX64(t, after.Funcs, "after-basic-scalar")
	if beforeExit != afterExit {
		t.Fatalf("exit mismatch before=%d after=%d", beforeExit, afterExit)
	}
	if afterExit != 15 {
		t.Fatalf("optimized exit = %d, want 15", afterExit)
	}
}

func countDumpOccurrences(dump string, needle string) int {
	count := 0
	for _, line := range strings.Split(dump, "\n") {
		if strings.Contains(line, needle) {
			count++
		}
	}
	return count
}

func scalarDifferentialProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 4},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRMulI32},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 6},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func runOptLinuxX64(t *testing.T, funcs []ir.IRFunc, name string) int {
	t.Helper()
	obj, err := linux_x64.CodegenObjectLinuxX64(funcs)
	if err != nil {
		t.Fatalf("%s CodegenObjectLinuxX64: %v", name, err)
	}
	img, err := linker.LinkLinuxX64([]*tobj.Object{obj}, "main")
	if err != nil {
		t.Fatalf("%s LinkLinuxX64: %v", name, err)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := elf.WriteELF64LinuxX64(path, img); err != nil {
		t.Fatalf("%s WriteELF64LinuxX64: %v", name, err)
	}
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("%s chmod: %v", name, err)
	}
	out, err := exec.Command(path).CombinedOutput()
	if len(out) != 0 {
		t.Fatalf("%s stdout/stderr = %q, want empty", name, out)
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return exit.ExitCode()
	}
	if err != nil {
		t.Fatalf("%s run: %v", name, err)
	}
	return 0
}
