package opt

import (
	"strconv"
	"strings"
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestBasicScalarPassFoldsSafeConstantsAndAlgebra(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRConstI32, Imm: 3},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRMulI32},
				{Kind: ir.IRReturn},
			},
		}},
	}

	report, err := NewManager().Run(prog, BasicScalarPass())
	if err != nil {
		t.Fatalf("Run BasicScalarPass: %v", err)
	}
	if len(report.Passes) != 1 {
		t.Fatalf("passes = %d, want 1", len(report.Passes))
	}
	row := report.Passes[0]
	if row.Name != "basic-scalar" || row.InputKind != IRKindStack || row.OutputKind != IRKindStack || !row.TranslationValidated {
		t.Fatalf("metadata row = %#v", row)
	}
	for _, want := range []string{"const_i32 2", "const_i32 3", "add_i32", "mul_i32"} {
		if !strings.Contains(row.BeforeDump, want) {
			t.Fatalf("before dump missing %q:\n%s", want, row.BeforeDump)
		}
	}
	if !strings.Contains(row.AfterDump, "const_i32 5") {
		t.Fatalf("after dump missing folded const 5:\n%s", row.AfterDump)
	}
	for _, forbidden := range []string{"add_i32", "mul_i32"} {
		if strings.Contains(row.AfterDump, forbidden) {
			t.Fatalf("after dump still contains %q:\n%s", forbidden, row.AfterDump)
		}
	}
}

func TestBasicScalarPassFoldsSafeConstDenominatorDivModConstants(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 20},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRDivI32},
				{Kind: ir.IRConstI32, Imm: 23},
				{Kind: ir.IRConstI32, Imm: 6},
				{Kind: ir.IRModI32},
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
	if !strings.Contains(after, "const_i32 9") {
		t.Fatalf("after dump missing folded safe div/mod result:\n%s", after)
	}
	for _, forbidden := range []string{"div_i32", "mod_i32", "add_i32"} {
		if strings.Contains(after, forbidden) {
			t.Fatalf("after dump still contains folded safe div/mod artifact %q:\n%s", forbidden, after)
		}
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "before-safe-divmod-const-fold")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-safe-divmod-const-fold")
	if beforeExit != afterExit || afterExit != 9 {
		t.Fatalf("native exits before=%d after=%d want 9", beforeExit, afterExit)
	}
}

func TestBasicScalarPassDoesNotFoldUnsafeConstDenominatorDivModConstants(t *testing.T) {
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
					ReturnSlots: 1,
					Instrs: []ir.IRInstr{
						{Kind: ir.IRConstI32, Imm: 20},
						{Kind: ir.IRConstI32, Imm: tc.denom},
						{Kind: tc.kind},
						{Kind: ir.IRReturn},
					},
				}},
			}
			before := FormatProgram(prog)

			report, err := NewManager().Run(prog, BasicScalarPass())
			if err != nil {
				t.Fatalf("Run BasicScalarPass: %v", err)
			}
			after := report.Passes[0].AfterDump
			if after != before {
				t.Fatalf("unsafe div/mod constant fold changed unexpectedly:\nbefore:\n%s\nafter:\n%s", before, after)
			}
			if !strings.Contains(after, tc.op) {
				t.Fatalf("after dump missing preserved unsafe %s:\n%s", tc.op, after)
			}
		})
	}
}

func TestBasicScalarPassDoesNotFoldUnsafeOverflowCases(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 2147483647},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRConstI32, Imm: -2147483648},
				{Kind: ir.IRNegI32},
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
	for _, want := range []string{"const_i32 2147483647", "const_i32 1", "add_i32", "const_i32 -2147483648", "neg_i32"} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q; unsafe case may have folded:\n%s", want, after)
		}
	}
}

func TestBasicScalarPassSimplifiesSameLocalComparisonAlgebra(t *testing.T) {
	tests := []struct {
		name     string
		kind     ir.IRInstrKind
		op       string
		wantExit int
	}{
		{name: "eq", kind: ir.IRCmpEqI32, op: "cmp_eq_i32", wantExit: 1},
		{name: "le", kind: ir.IRCmpLeI32, op: "cmp_le_i32", wantExit: 1},
		{name: "ge", kind: ir.IRCmpGeI32, op: "cmp_ge_i32", wantExit: 1},
		{name: "ne", kind: ir.IRCmpNeI32, op: "cmp_ne_i32", wantExit: 0},
		{name: "lt", kind: ir.IRCmpLtI32, op: "cmp_lt_i32", wantExit: 0},
		{name: "gt", kind: ir.IRCmpGtI32, op: "cmp_gt_i32", wantExit: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog := &ir.IRProgram{
				MainIndex: 0,
				MainName:  "main",
				Funcs: []ir.IRFunc{{
					Name:        "main",
					LocalSlots:  1,
					ReturnSlots: 1,
					Instrs: []ir.IRInstr{
						{Kind: ir.IRConstI32, Imm: 7},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: tc.kind},
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
			if !strings.Contains(after, "const_i32 "+strconv.Itoa(tc.wantExit)) {
				t.Fatalf("after dump missing same-local comparison constant %d:\n%s", tc.wantExit, after)
			}
			if strings.Contains(after, tc.op) {
				t.Fatalf("same-local comparison op was not simplified:\n%s", after)
			}
			beforeExit := runOptLinuxX64(t, before.Funcs, "before-same-local-"+tc.name+"-comparison")
			afterExit := runOptLinuxX64(t, prog.Funcs, "after-same-local-"+tc.name+"-comparison")
			if beforeExit != afterExit || afterExit != tc.wantExit {
				t.Fatalf("native exits before=%d after=%d want %d", beforeExit, afterExit, tc.wantExit)
			}
		})
	}
}

func TestBasicScalarPassPropagatesCopiesAndEliminatesDeadStores(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  3,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRConstI32, Imm: 0},
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
	if !strings.Contains(after, "load_local local:0") || !strings.Contains(after, "return") {
		t.Fatalf("after dump missing load/return:\n%s", after)
	}
	for _, forbidden := range []string{"const_i32 99", "store_local local:1", "store_local local:2", "load_local local:2", "add_i32"} {
		if strings.Contains(after, forbidden) {
			t.Fatalf("after dump still contains %q:\n%s", forbidden, after)
		}
	}
	if got := len(prog.Funcs[0].Instrs); got != 2 {
		t.Fatalf("optimized instruction count = %d, want load_local + return only; dump:\n%s", got, after)
	}
}

func TestBasicScalarPassEliminatesDeadNonTrappingComparisonStore(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  3,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 4},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRCmpGtI32},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRLoadLocal, Local: 0},
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
	for _, forbidden := range []string{"cmp_gt_i32", "store_local local:2", "store_local local:1"} {
		if strings.Contains(after, forbidden) {
			t.Fatalf("after dump still contains dead comparison store artifact %q:\n%s", forbidden, after)
		}
	}
	if !strings.Contains(after, "load_local local:0") || !strings.Contains(after, "return") {
		t.Fatalf("after dump missing live return path:\n%s", after)
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "before-dead-comparison-store")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-dead-comparison-store")
	if beforeExit != afterExit || afterExit != 4 {
		t.Fatalf("native exits before=%d after=%d want 4", beforeExit, afterExit)
	}
}

func TestBasicScalarPassEliminatesDeadSafeConstDenominatorDivModStore(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  3,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 20},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRDivI32},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 6},
				{Kind: ir.IRModI32},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRConstI32, Imm: 3},
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
	for _, forbidden := range []string{"div_i32", "mod_i32", "store_local local:1", "store_local local:2"} {
		if strings.Contains(after, forbidden) {
			t.Fatalf("after dump still contains safe dead div/mod store artifact %q:\n%s", forbidden, after)
		}
	}
	if !strings.Contains(after, "const_i32 3") || !strings.Contains(after, "return") {
		t.Fatalf("after dump missing live return value:\n%s", after)
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "before-dead-safe-divmod-store")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-dead-safe-divmod-store")
	if beforeExit != afterExit || afterExit != 3 {
		t.Fatalf("native exits before=%d after=%d want 3", beforeExit, afterExit)
	}
}

func TestBasicScalarPassDoesNotEliminateDeadUnsafeConstDenominatorDivModStore(t *testing.T) {
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
						{Kind: ir.IRConstI32, Imm: 20},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: tc.denom},
						{Kind: tc.kind},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRConstI32, Imm: 3},
						{Kind: ir.IRReturn},
					},
				}},
			}

			report, err := NewManager().Run(prog, BasicScalarPass())
			if err != nil {
				t.Fatalf("Run BasicScalarPass: %v", err)
			}
			after := report.Passes[0].AfterDump
			if !strings.Contains(after, tc.op) {
				t.Fatalf("after dump removed unsafe %s dead store producer:\n%s", tc.op, after)
			}
			if !strings.Contains(after, "store_local local:1") {
				t.Fatalf("after dump removed unsafe dead store sink:\n%s", after)
			}
		})
	}
}

func TestBasicScalarPassEliminatesDeadSafeKnownLocalDivModStore(t *testing.T) {
	tests := []struct {
		name      string
		kind      ir.IRInstrKind
		op        string
		leftImm   int32
		rightImm  int32
		localSlot int
	}{
		{name: "division", kind: ir.IRDivI32, op: "div_i32", leftImm: 20, rightImm: 5, localSlot: 2},
		{name: "modulo", kind: ir.IRModI32, op: "mod_i32", leftImm: 23, rightImm: 5, localSlot: 2},
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
						{Kind: ir.IRConstI32, Imm: tc.leftImm},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: tc.rightImm},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: ir.IRLoadLocal, Local: 1},
						{Kind: tc.kind},
						{Kind: ir.IRStoreLocal, Local: tc.localSlot},
						{Kind: ir.IRConstI32, Imm: 3},
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
			for _, forbidden := range []string{tc.op, "store_local local:0", "store_local local:1", "store_local local:2"} {
				if strings.Contains(after, forbidden) {
					t.Fatalf("after dump still contains safe known-local div/mod dead store artifact %q:\n%s", forbidden, after)
				}
			}
			if !strings.Contains(after, "const_i32 3") || !strings.Contains(after, "return") {
				t.Fatalf("after dump missing live return value:\n%s", after)
			}
			beforeExit := runOptLinuxX64(t, before.Funcs, "before-dead-safe-known-local-divmod-"+tc.name)
			afterExit := runOptLinuxX64(t, prog.Funcs, "after-dead-safe-known-local-divmod-"+tc.name)
			if beforeExit != afterExit || afterExit != 3 {
				t.Fatalf("native exits before=%d after=%d want 3", beforeExit, afterExit)
			}
		})
	}
}

func TestBasicScalarPassDoesNotEliminateDeadUnsafeKnownLocalDivModStore(t *testing.T) {
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
					LocalSlots:  3,
					ReturnSlots: 1,
					Instrs: []ir.IRInstr{
						{Kind: ir.IRConstI32, Imm: 20},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: tc.denom},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: ir.IRLoadLocal, Local: 1},
						{Kind: tc.kind},
						{Kind: ir.IRStoreLocal, Local: 2},
						{Kind: ir.IRConstI32, Imm: 3},
						{Kind: ir.IRReturn},
					},
				}},
			}
			before := FormatProgram(prog)

			report, err := NewManager().Run(prog, BasicScalarPass())
			if err != nil {
				t.Fatalf("Run BasicScalarPass: %v", err)
			}
			after := report.Passes[0].AfterDump
			if after != before {
				t.Fatalf("unsafe known-local div/mod dead store changed unexpectedly:\nbefore:\n%s\nafter:\n%s", before, after)
			}
			for _, want := range []string{"load_local local:0", "load_local local:1", tc.op, "store_local local:2"} {
				if !strings.Contains(after, want) {
					t.Fatalf("after dump missing preserved unsafe div/mod artifact %q:\n%s", want, after)
				}
			}
		})
	}
}

func TestBasicScalarPassEliminatesDeadSafeKnownLocalUnaryNegStore(t *testing.T) {
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
				{Kind: ir.IRConstI32, Imm: 3},
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
	for _, forbidden := range []string{"load_local local:0", "neg_i32", "store_local local:1", "store_local local:0"} {
		if strings.Contains(after, forbidden) {
			t.Fatalf("after dump still contains safe dead unary neg store artifact %q:\n%s", forbidden, after)
		}
	}
	if !strings.Contains(after, "const_i32 3") || !strings.Contains(after, "return") {
		t.Fatalf("after dump missing live return value:\n%s", after)
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "before-dead-safe-unary-neg-store")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-dead-safe-unary-neg-store")
	if beforeExit != afterExit || afterExit != 3 {
		t.Fatalf("native exits before=%d after=%d want 3", beforeExit, afterExit)
	}
}

func TestBasicScalarPassDoesNotEliminateDeadUnsafeKnownLocalUnaryNegStore(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: -2147483648},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRNegI32},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 3},
				{Kind: ir.IRReturn},
			},
		}},
	}
	before := FormatProgram(prog)

	report, err := NewManager().Run(prog, BasicScalarPass())
	if err != nil {
		t.Fatalf("Run BasicScalarPass: %v", err)
	}
	after := report.Passes[0].AfterDump
	if after != before {
		t.Fatalf("unsafe known-local unary neg dead store changed unexpectedly:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	for _, want := range []string{"const_i32 -2147483648", "load_local local:0", "neg_i32", "store_local local:1"} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing preserved unsafe unary neg artifact %q:\n%s", want, after)
		}
	}
}

func TestBasicScalarPassEliminatesDeadSafeKnownLocalArithmeticStore(t *testing.T) {
	tests := []struct {
		name      string
		kind      ir.IRInstrKind
		op        string
		leftImm   int32
		rightImm  int32
		rightLoad bool
	}{
		{name: "add-local-const", kind: ir.IRAddI32, op: "add_i32", leftImm: 5, rightImm: 7},
		{name: "sub-local-const", kind: ir.IRSubI32, op: "sub_i32", leftImm: 5, rightImm: 3},
		{name: "mul-two-locals", kind: ir.IRMulI32, op: "mul_i32", leftImm: 6, rightImm: 7, rightLoad: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			instrs := []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: tc.leftImm},
				{Kind: ir.IRStoreLocal, Local: 0},
			}
			if tc.rightLoad {
				instrs = append(instrs,
					ir.IRInstr{Kind: ir.IRConstI32, Imm: tc.rightImm},
					ir.IRInstr{Kind: ir.IRStoreLocal, Local: 1},
				)
			}
			instrs = append(instrs, ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0})
			if tc.rightLoad {
				instrs = append(instrs, ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1})
			} else {
				instrs = append(instrs, ir.IRInstr{Kind: ir.IRConstI32, Imm: tc.rightImm})
			}
			instrs = append(instrs,
				ir.IRInstr{Kind: tc.kind},
				ir.IRInstr{Kind: ir.IRStoreLocal, Local: 2},
				ir.IRInstr{Kind: ir.IRConstI32, Imm: 3},
				ir.IRInstr{Kind: ir.IRReturn},
			)
			prog := &ir.IRProgram{
				MainIndex: 0,
				MainName:  "main",
				Funcs: []ir.IRFunc{{
					Name:        "main",
					LocalSlots:  3,
					ReturnSlots: 1,
					Instrs:      instrs,
				}},
			}
			before := cloneProgram(prog)

			report, err := NewManager().Run(prog, BasicScalarPass())
			if err != nil {
				t.Fatalf("Run BasicScalarPass: %v", err)
			}
			after := report.Passes[0].AfterDump
			for _, forbidden := range []string{tc.op, "store_local local:2", "store_local local:0"} {
				if strings.Contains(after, forbidden) {
					t.Fatalf("after dump still contains safe dead arithmetic store artifact %q:\n%s", forbidden, after)
				}
			}
			if tc.rightLoad && strings.Contains(after, "store_local local:1") {
				t.Fatalf("after dump still contains dead right operand local store:\n%s", after)
			}
			if !strings.Contains(after, "const_i32 3") || !strings.Contains(after, "return") {
				t.Fatalf("after dump missing live return value:\n%s", after)
			}
			beforeExit := runOptLinuxX64(t, before.Funcs, "before-dead-safe-known-local-arithmetic-"+tc.name)
			afterExit := runOptLinuxX64(t, prog.Funcs, "after-dead-safe-known-local-arithmetic-"+tc.name)
			if beforeExit != afterExit || afterExit != 3 {
				t.Fatalf("native exits before=%d after=%d want 3", beforeExit, afterExit)
			}
		})
	}
}

func TestBasicScalarPassDoesNotEliminateDeadUnsafeKnownLocalArithmeticStore(t *testing.T) {
	tests := []struct {
		name     string
		kind     ir.IRInstrKind
		op       string
		leftImm  int32
		rightImm int32
	}{
		{name: "add-overflow", kind: ir.IRAddI32, op: "add_i32", leftImm: 2147483647, rightImm: 1},
		{name: "sub-overflow", kind: ir.IRSubI32, op: "sub_i32", leftImm: -2147483648, rightImm: 1},
		{name: "mul-overflow", kind: ir.IRMulI32, op: "mul_i32", leftImm: 1073741824, rightImm: 3},
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
						{Kind: ir.IRConstI32, Imm: tc.leftImm},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: tc.rightImm},
						{Kind: tc.kind},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRConstI32, Imm: 3},
						{Kind: ir.IRReturn},
					},
				}},
			}
			before := FormatProgram(prog)

			report, err := NewManager().Run(prog, BasicScalarPass())
			if err != nil {
				t.Fatalf("Run BasicScalarPass: %v", err)
			}
			after := report.Passes[0].AfterDump
			if after != before {
				t.Fatalf("unsafe known-local arithmetic dead store changed unexpectedly:\nbefore:\n%s\nafter:\n%s", before, after)
			}
			for _, want := range []string{"load_local local:0", tc.op, "store_local local:1"} {
				if !strings.Contains(after, want) {
					t.Fatalf("after dump missing preserved unsafe arithmetic artifact %q:\n%s", want, after)
				}
			}
		})
	}
}

func TestBasicScalarPassEliminatesRepeatedPureLocalExpressionWithCSE(t *testing.T) {
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
				{Kind: ir.IRAddI32},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
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
	if got := countDumpOccurrences(after, "add_i32"); got != 2 {
		t.Fatalf("optimized add count = %d, want initial expression plus final add; dump:\n%s", got, after)
	}
	if got := countDumpOccurrences(after, "load_local local:2"); got != 2 {
		t.Fatalf("optimized cached-expression loads = %d, want repeated expression to reuse local 2; dump:\n%s", got, after)
	}
	if strings.Contains(after, "load_local local:0\n  load_local local:1\n  add_i32\n  load_local local:2") {
		t.Fatalf("second common expression still recomputed before cached local load:\n%s", after)
	}
}

func TestBasicScalarPassEliminatesCommutativeLocalExpressionWithGVN(t *testing.T) {
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
				{Kind: ir.IRAddI32},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
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
	if got := countDumpOccurrences(after, "add_i32"); got != 2 {
		t.Fatalf("optimized add count = %d, want initial commutative expression plus final add; dump:\n%s", got, after)
	}
	if got := countDumpOccurrences(after, "load_local local:2"); got != 2 {
		t.Fatalf("optimized cached-expression loads = %d, want swapped expression to reuse local 2; dump:\n%s", got, after)
	}
	if strings.Contains(after, "load_local local:1\n  load_local local:0\n  add_i32\n  load_local local:2") {
		t.Fatalf("swapped common expression still recomputed before cached local load:\n%s", after)
	}
}

func TestBasicScalarPassEliminatesRepeatedLocalConstantExpressionWithCSE(t *testing.T) {
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
	if got := countDumpOccurrences(after, "const_i32 7"); got != 1 {
		t.Fatalf("optimized const count = %d, want one cached local-constant expression input; dump:\n%s", got, after)
	}
	if got := countDumpOccurrences(after, "add_i32"); got != 2 {
		t.Fatalf("optimized add count = %d, want initial expression plus final add; dump:\n%s", got, after)
	}
	if got := countDumpOccurrences(after, "load_local local:2"); got != 2 {
		t.Fatalf("optimized cached-expression loads = %d, want repeated local-constant expression to reuse local 2; dump:\n%s", got, after)
	}
}

func TestBasicScalarPassEliminatesCommutativeLocalConstantExpressionWithGVN(t *testing.T) {
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
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRLoadLocal, Local: 0},
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
	if got := countDumpOccurrences(after, "const_i32 7"); got != 1 {
		t.Fatalf("optimized const count = %d, want swapped local-constant expression to reuse cached local; dump:\n%s", got, after)
	}
	if got := countDumpOccurrences(after, "add_i32"); got != 2 {
		t.Fatalf("optimized add count = %d, want initial expression plus final add; dump:\n%s", got, after)
	}
	if got := countDumpOccurrences(after, "load_local local:2"); got != 2 {
		t.Fatalf("optimized cached-expression loads = %d, want swapped local-constant expression to reuse local 2; dump:\n%s", got, after)
	}
}

func TestBasicScalarPassEliminatesSafeKnownLocalArithmeticExpressionWithGVN(t *testing.T) {
	tests := []struct {
		name        string
		kind        ir.IRInstrKind
		opName      string
		leftImm     int32
		rightImm    int32
		wantExit    int
		wantOpCount int
	}{
		{name: "add", kind: ir.IRAddI32, opName: "add_i32", leftImm: 5, rightImm: 7, wantExit: 24, wantOpCount: 2},
		{name: "sub", kind: ir.IRSubI32, opName: "sub_i32", leftImm: 11, rightImm: 4, wantExit: 14, wantOpCount: 1},
		{name: "mul", kind: ir.IRMulI32, opName: "mul_i32", leftImm: 6, rightImm: 7, wantExit: 84, wantOpCount: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog := &ir.IRProgram{
				MainIndex: 0,
				MainName:  "main",
				Funcs: []ir.IRFunc{{
					Name:        "main",
					LocalSlots:  4,
					ReturnSlots: 1,
					Instrs: []ir.IRInstr{
						{Kind: ir.IRConstI32, Imm: tc.leftImm},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: tc.rightImm},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRConstI32, Imm: tc.leftImm},
						{Kind: ir.IRStoreLocal, Local: 3},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: ir.IRLoadLocal, Local: 1},
						{Kind: tc.kind},
						{Kind: ir.IRStoreLocal, Local: 2},
						{Kind: ir.IRLoadLocal, Local: 3},
						{Kind: ir.IRLoadLocal, Local: 1},
						{Kind: tc.kind},
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
			if got := countDumpOccurrences(after, tc.opName); got != tc.wantOpCount {
				t.Fatalf("optimized %s count = %d, want %d with known-local value reuse; dump:\n%s", tc.opName, got, tc.wantOpCount, after)
			}
			if got := countDumpOccurrences(after, "load_local local:2"); got != 2 {
				t.Fatalf("cached-expression loads = %d, want repeated known-local expression to reuse local 2; dump:\n%s", got, after)
			}
			if strings.Contains(after, "load_local local:3\n  load_local local:1\n  "+tc.opName+"\n  load_local local:2") {
				t.Fatalf("known-local equivalent expression was recomputed instead of reusing cached local:\n%s", after)
			}
			beforeExit := runOptLinuxX64(t, before.Funcs, "before-safe-known-local-"+tc.name+"-gvn")
			afterExit := runOptLinuxX64(t, prog.Funcs, "after-safe-known-local-"+tc.name+"-gvn")
			if beforeExit != afterExit || afterExit != tc.wantExit {
				t.Fatalf("native exits before=%d after=%d want %d", beforeExit, afterExit, tc.wantExit)
			}
		})
	}
}

func TestBasicScalarPassEliminatesSafeKnownLocalComparisonExpressionWithGVN(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  4,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRStoreLocal, Local: 3},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRCmpLtI32},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 3},
				{Kind: ir.IRCmpGtI32},
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
	if got := countDumpOccurrences(after, "cmp_lt_i32"); got != 1 {
		t.Fatalf("optimized cmp_lt_i32 count = %d, want original comparison only; dump:\n%s", got, after)
	}
	if strings.Contains(after, "cmp_gt_i32") {
		t.Fatalf("mirrored known-local comparison was recomputed instead of reusing cached local:\n%s", after)
	}
	if got := countDumpOccurrences(after, "load_local local:2"); got != 2 {
		t.Fatalf("cached comparison loads = %d, want known-local comparison to reuse local 2; dump:\n%s", got, after)
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "before-safe-known-local-comparison-gvn")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-safe-known-local-comparison-gvn")
	if beforeExit != afterExit || afterExit != 2 {
		t.Fatalf("native exits before=%d after=%d want 2", beforeExit, afterExit)
	}
}

func TestBasicScalarPassDoesNotReuseKnownLocalComparisonExpressionAfterSourceMutation(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  4,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRStoreLocal, Local: 3},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRCmpLtI32},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRConstI32, Imm: 8},
				{Kind: ir.IRStoreLocal, Local: 3},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 3},
				{Kind: ir.IRCmpGtI32},
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
	if got := countDumpOccurrences(after, "cmp_lt_i32"); got != 1 {
		t.Fatalf("optimized cmp_lt_i32 count = %d, want original comparison preserved; dump:\n%s", got, after)
	}
	if got := countDumpOccurrences(after, "cmp_gt_i32"); got != 1 {
		t.Fatalf("optimized cmp_gt_i32 count = %d, want mutated comparison preserved; dump:\n%s", got, after)
	}
	if !strings.Contains(after, "load_local local:1\n  load_local local:3\n  cmp_gt_i32") {
		t.Fatalf("mutated known-local comparison was not preserved:\n%s", after)
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "before-mutated-known-local-comparison-gvn")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-mutated-known-local-comparison-gvn")
	if beforeExit != afterExit || afterExit != 1 {
		t.Fatalf("native exits before=%d after=%d want 1", beforeExit, afterExit)
	}
}

func TestBasicScalarPassDoesNotReuseKnownLocalArithmeticExpressionAfterSourceMutation(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  4,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRStoreLocal, Local: 3},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRConstI32, Imm: 6},
				{Kind: ir.IRStoreLocal, Local: 3},
				{Kind: ir.IRLoadLocal, Local: 3},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRAddI32},
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
	if got := countDumpOccurrences(after, "add_i32"); got != 3 {
		t.Fatalf("optimized add count = %d, want mutated source expression preserved plus final add; dump:\n%s", got, after)
	}
	if !strings.Contains(after, "load_local local:3\n  load_local local:1\n  add_i32") {
		t.Fatalf("mutated source expression was not preserved:\n%s", after)
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "before-mutated-known-local-gvn")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-mutated-known-local-gvn")
	if beforeExit != afterExit || afterExit != 25 {
		t.Fatalf("native exits before=%d after=%d want 25", beforeExit, afterExit)
	}
}

func TestBasicScalarPassDoesNotReuseOverflowSensitiveKnownLocalArithmeticExpression(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  4,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 2147483647},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 2147483647},
				{Kind: ir.IRStoreLocal, Local: 3},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRLoadLocal, Local: 3},
				{Kind: ir.IRLoadLocal, Local: 1},
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
	if got := countDumpOccurrences(after, "add_i32"); got != 3 {
		t.Fatalf("optimized overflow-sensitive add count = %d, want both unsafe expressions preserved plus final add; dump:\n%s", got, after)
	}
	if !strings.Contains(after, "load_local local:3\n  load_local local:1\n  add_i32") {
		t.Fatalf("overflow-sensitive known-local expression was not preserved:\n%s", after)
	}
}

func TestBasicScalarPassEliminatesSafeKnownLocalDivModExpressionWithGVN(t *testing.T) {
	tests := []struct {
		name     string
		kind     ir.IRInstrKind
		opName   string
		leftImm  int32
		rightImm int32
		wantExit int
	}{
		{name: "division", kind: ir.IRDivI32, opName: "div_i32", leftImm: 20, rightImm: 5, wantExit: 8},
		{name: "modulo", kind: ir.IRModI32, opName: "mod_i32", leftImm: 23, rightImm: 5, wantExit: 6},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog := &ir.IRProgram{
				MainIndex: 0,
				MainName:  "main",
				Funcs: []ir.IRFunc{{
					Name:        "main",
					LocalSlots:  4,
					ReturnSlots: 1,
					Instrs: []ir.IRInstr{
						{Kind: ir.IRConstI32, Imm: tc.leftImm},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: tc.rightImm},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRConstI32, Imm: tc.leftImm},
						{Kind: ir.IRStoreLocal, Local: 3},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: ir.IRLoadLocal, Local: 1},
						{Kind: tc.kind},
						{Kind: ir.IRStoreLocal, Local: 2},
						{Kind: ir.IRLoadLocal, Local: 3},
						{Kind: ir.IRLoadLocal, Local: 1},
						{Kind: tc.kind},
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
			if got := countDumpOccurrences(after, tc.opName); got != 1 {
				t.Fatalf("optimized %s count = %d, want one safe known-local value expression; dump:\n%s", tc.opName, got, after)
			}
			if got := countDumpOccurrences(after, "load_local local:2"); got != 2 {
				t.Fatalf("cached-expression loads = %d, want repeated known-local %s to reuse local 2; dump:\n%s", got, tc.opName, after)
			}
			if strings.Contains(after, "load_local local:3\n  load_local local:1\n  "+tc.opName+"\n  load_local local:2") {
				t.Fatalf("known-local div/mod expression was recomputed instead of reusing cached local:\n%s", after)
			}
			beforeExit := runOptLinuxX64(t, before.Funcs, "before-safe-known-local-"+tc.name+"-gvn")
			afterExit := runOptLinuxX64(t, prog.Funcs, "after-safe-known-local-"+tc.name+"-gvn")
			if beforeExit != afterExit || afterExit != tc.wantExit {
				t.Fatalf("native exits before=%d after=%d want %d", beforeExit, afterExit, tc.wantExit)
			}
		})
	}
}

func TestBasicScalarPassDoesNotReuseKnownLocalDivModExpressionAfterSourceMutation(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  4,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 20},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 20},
				{Kind: ir.IRStoreLocal, Local: 3},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRDivI32},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRConstI32, Imm: 25},
				{Kind: ir.IRStoreLocal, Local: 3},
				{Kind: ir.IRLoadLocal, Local: 3},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRDivI32},
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
	if got := countDumpOccurrences(after, "div_i32"); got != 2 {
		t.Fatalf("optimized div_i32 count = %d, want mutated source expression preserved plus final add; dump:\n%s", got, after)
	}
	if !strings.Contains(after, "load_local local:3\n  load_local local:1\n  div_i32") {
		t.Fatalf("mutated source div expression was not preserved:\n%s", after)
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "before-mutated-known-local-divmod-gvn")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-mutated-known-local-divmod-gvn")
	if beforeExit != afterExit || afterExit != 9 {
		t.Fatalf("native exits before=%d after=%d want 9", beforeExit, afterExit)
	}
}

func TestBasicScalarPassDoesNotReuseUnsafeKnownLocalDivModExpression(t *testing.T) {
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
					LocalSlots:  4,
					ReturnSlots: 1,
					Instrs: []ir.IRInstr{
						{Kind: ir.IRConstI32, Imm: 20},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: tc.denom},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRConstI32, Imm: 20},
						{Kind: ir.IRStoreLocal, Local: 3},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: ir.IRLoadLocal, Local: 1},
						{Kind: tc.kind},
						{Kind: ir.IRStoreLocal, Local: 2},
						{Kind: ir.IRLoadLocal, Local: 3},
						{Kind: ir.IRLoadLocal, Local: 1},
						{Kind: tc.kind},
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
			if got := countDumpOccurrences(after, tc.op); got != 2 {
				t.Fatalf("optimized unsafe %s count = %d, want both expressions preserved; dump:\n%s", tc.op, got, after)
			}
			if !strings.Contains(after, "load_local local:3\n  load_local local:1\n  "+tc.op) {
				t.Fatalf("unsafe known-local div/mod expression was not preserved:\n%s", after)
			}
		})
	}
}

func TestBasicScalarPassEliminatesSafeKnownLocalUnaryNegExpressionWithCSE(t *testing.T) {
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
	if got := countDumpOccurrences(after, "neg_i32"); got != 1 {
		t.Fatalf("optimized neg_i32 count = %d, want one safe known-local unary expression; dump:\n%s", got, after)
	}
	if got := countDumpOccurrences(after, "load_local local:2"); got != 2 {
		t.Fatalf("cached unary-expression loads = %d, want repeated known-local neg to reuse local 2; dump:\n%s", got, after)
	}
	if strings.Contains(after, "load_local local:3\n  neg_i32\n  load_local local:2") {
		t.Fatalf("known-local unary expression was recomputed instead of reusing cached local:\n%s", after)
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "before-safe-known-local-unary-gvn")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-safe-known-local-unary-gvn")
	if beforeExit != afterExit || afterExit != 12 {
		t.Fatalf("native exits before=%d after=%d want 12", beforeExit, afterExit)
	}
}
