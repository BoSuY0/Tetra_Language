package opt

import (
	"strconv"
	"strings"
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestMem2RegPassPromotesSingleAssignmentTempAndReportsDecision(t *testing.T) {
	prog := singleAssignmentTempProgram()
	before := cloneProgram(prog)

	report, err := NewManager().Run(prog, Mem2RegPass())
	if err != nil {
		t.Fatalf("Run Mem2RegPass: %v", err)
	}
	if len(report.Passes) != 1 {
		t.Fatalf("passes = %d, want 1", len(report.Passes))
	}
	row := report.Passes[0]
	if row.Name != "mem2reg-single-assignment" || !row.TranslationValidated || row.ValidationMetadata == nil {
		t.Fatalf("metadata row = %#v", row)
	}
	for _, want := range []string{"add_i32", "store_local local:0", "load_local local:0", "mul_i32"} {
		if !strings.Contains(row.BeforeDump, want) {
			t.Fatalf("before dump missing %q:\n%s", want, row.BeforeDump)
		}
	}
	after := row.AfterDump
	for _, want := range []string{"const_i32 4", "const_i32 5", "add_i32", "const_i32 2", "mul_i32", "return"} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	for _, forbidden := range []string{"store_local local:0", "load_local local:0"} {
		if strings.Contains(after, forbidden) {
			t.Fatalf("after dump still contains promoted temp %q:\n%s", forbidden, after)
		}
	}
	if !hasDecision(row.Decisions, "promoted_single_assignment_temp", "single_store_single_load_adjacent") {
		t.Fatalf("decisions = %#v, want promoted single-assignment temp", row.Decisions)
	}

	beforeExit := runOptLinuxX64(t, before.Funcs, "before-mem2reg")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-mem2reg")
	if beforeExit != afterExit || afterExit != 18 {
		t.Fatalf("exit mismatch before=%d after=%d, want both 18", beforeExit, afterExit)
	}
}

func TestMem2RegPassPromotesSeparatedSingleAssignmentTempWithStackNeutralWork(t *testing.T) {
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
				{Kind: ir.IRConstI32, Imm: 9},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRMulI32},
				{Kind: ir.IRReturn},
			},
		}},
	}
	before := cloneProgram(prog)

	report, err := NewManager().Run(prog, Mem2RegPass())
	if err != nil {
		t.Fatalf("Run Mem2RegPass: %v", err)
	}
	row := report.Passes[0]
	after := row.AfterDump
	for _, want := range []string{"const_i32 4", "const_i32 9", "store_local local:2", "const_i32 2", "mul_i32", "return"} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	for _, forbidden := range []string{"store_local local:0", "load_local local:0"} {
		if strings.Contains(after, forbidden) {
			t.Fatalf("after dump still contains promoted separated temp %q:\n%s", forbidden, after)
		}
	}
	if !hasDecision(row.Decisions, "promoted_single_assignment_temp", "single_store_single_load_stack_neutral") {
		t.Fatalf("decisions = %#v, want stack-neutral separated promotion", row.Decisions)
	}

	beforeExit := runOptLinuxX64(t, before.Funcs, "before-separated-mem2reg")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-separated-mem2reg")
	if beforeExit != afterExit || afterExit != 8 {
		t.Fatalf("exit mismatch before=%d after=%d, want both 8", beforeExit, afterExit)
	}
}

func TestMem2RegPassPromotesSeparatedComparisonExpressionTempWithStackNeutralWork(t *testing.T) {
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
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRCmpLtI32},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 3},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRReturn},
			},
		}},
	}
	before := cloneProgram(prog)

	report, err := NewManager().Run(prog, Mem2RegPass())
	if err != nil {
		t.Fatalf("Run Mem2RegPass: %v", err)
	}
	row := report.Passes[0]
	after := row.AfterDump
	for _, want := range []string{"load_local local:0", "const_i32 7", "cmp_lt_i32", "const_i32 3", "store_local local:2", "const_i32 2", "add_i32", "store_local local:0", "return"} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	for _, forbidden := range []string{"store_local local:1", "load_local local:1"} {
		if strings.Contains(after, forbidden) {
			t.Fatalf("after dump still contains promoted comparison temp %q:\n%s", forbidden, after)
		}
	}
	if !hasDecision(row.Decisions, "promoted_single_assignment_temp", "single_store_single_load_stack_neutral_comparison_expression") {
		t.Fatalf("decisions = %#v, want stack-neutral comparison-expression promotion", row.Decisions)
	}

	beforeExit := runOptLinuxX64(t, before.Funcs, "before-separated-comparison-mem2reg")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-separated-comparison-mem2reg")
	if beforeExit != afterExit || afterExit != 3 {
		t.Fatalf("exit mismatch before=%d after=%d, want both 3", beforeExit, afterExit)
	}
}

func TestMem2RegPassPromotesSeparatedSafeConstDenominatorDivModTempWithStackNeutralWork(t *testing.T) {
	cases := []struct {
		name     string
		kind     ir.IRInstrKind
		op       string
		denom    int32
		wantExit int
	}{
		{name: "division", kind: ir.IRDivI32, op: "div_i32", denom: 3, wantExit: 6},
		{name: "modulo", kind: ir.IRModI32, op: "mod_i32", denom: 5, wantExit: 4},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog := &ir.IRProgram{
				MainIndex: 0,
				MainName:  "main",
				Funcs: []ir.IRFunc{{
					Name:        "main",
					LocalSlots:  3,
					ReturnSlots: 1,
					Instrs: []ir.IRInstr{
						{Kind: ir.IRConstI32, Imm: 12},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: tc.denom},
						{Kind: tc.kind},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRConstI32, Imm: 7},
						{Kind: ir.IRStoreLocal, Local: 2},
						{Kind: ir.IRLoadLocal, Local: 1},
						{Kind: ir.IRConstI32, Imm: 2},
						{Kind: ir.IRAddI32},
						{Kind: ir.IRConstI32, Imm: 0},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRReturn},
					},
				}},
			}
			before := cloneProgram(prog)

			report, err := NewManager().Run(prog, Mem2RegPass())
			if err != nil {
				t.Fatalf("Run Mem2RegPass: %v", err)
			}
			row := report.Passes[0]
			after := row.AfterDump
			for _, want := range []string{"load_local local:0", "const_i32 " + strconv.FormatInt(int64(tc.denom), 10), tc.op, "const_i32 7", "store_local local:2", "const_i32 2", "add_i32", "store_local local:0", "return"} {
				if !strings.Contains(after, want) {
					t.Fatalf("after dump missing %q:\n%s", want, after)
				}
			}
			for _, forbidden := range []string{"store_local local:1", "load_local local:1"} {
				if strings.Contains(after, forbidden) {
					t.Fatalf("after dump still contains promoted div/mod temp %q:\n%s", forbidden, after)
				}
			}
			if !hasDecision(row.Decisions, "promoted_single_assignment_temp", "single_store_single_load_stack_neutral_safe_const_denominator_divmod_expression") {
				t.Fatalf("decisions = %#v, want stack-neutral safe div/mod expression promotion", row.Decisions)
			}

			beforeExit := runOptLinuxX64(t, before.Funcs, "before-separated-safe-divmod-mem2reg-"+tc.name)
			afterExit := runOptLinuxX64(t, prog.Funcs, "after-separated-safe-divmod-mem2reg-"+tc.name)
			if beforeExit != afterExit || afterExit != tc.wantExit {
				t.Fatalf("exit mismatch before=%d after=%d, want both %d", beforeExit, afterExit, tc.wantExit)
			}
		})
	}
}

func TestMem2RegPassPromotesSeparatedSafeKnownLocalDivModTempWithStackNeutralWork(t *testing.T) {
	cases := []struct {
		name     string
		kind     ir.IRInstrKind
		op       string
		left     int32
		right    int32
		wantExit int
	}{
		{name: "division", kind: ir.IRDivI32, op: "div_i32", left: 20, right: 5, wantExit: 13},
		{name: "modulo", kind: ir.IRModI32, op: "mod_i32", left: 23, right: 5, wantExit: 12},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog := &ir.IRProgram{
				MainIndex: 0,
				MainName:  "main",
				Funcs: []ir.IRFunc{{
					Name:        "main",
					LocalSlots:  4,
					ReturnSlots: 1,
					Instrs: []ir.IRInstr{
						{Kind: ir.IRConstI32, Imm: 0},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: tc.left},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: 0},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRConstI32, Imm: tc.right},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: ir.IRLoadLocal, Local: 1},
						{Kind: tc.kind},
						{Kind: ir.IRStoreLocal, Local: 2},
						{Kind: ir.IRConstI32, Imm: 9},
						{Kind: ir.IRStoreLocal, Local: 3},
						{Kind: ir.IRLoadLocal, Local: 2},
						{Kind: ir.IRLoadLocal, Local: 3},
						{Kind: ir.IRAddI32},
						{Kind: ir.IRReturn},
					},
				}},
			}
			before := cloneProgram(prog)

			report, err := NewManager().Run(prog, Mem2RegPass())
			if err != nil {
				t.Fatalf("Run Mem2RegPass: %v", err)
			}
			row := report.Passes[0]
			after := row.AfterDump
			for _, want := range []string{
				"load_local local:0",
				"load_local local:1",
				tc.op,
				"const_i32 9",
				"store_local local:3",
				"load_local local:3",
				"add_i32",
				"return",
			} {
				if !strings.Contains(after, want) {
					t.Fatalf("after dump missing %q:\n%s", want, after)
				}
			}
			for _, forbidden := range []string{"store_local local:2", "load_local local:2"} {
				if strings.Contains(after, forbidden) {
					t.Fatalf("after dump still contains promoted known-local div/mod temp %q:\n%s", forbidden, after)
				}
			}
			if !hasDecision(row.Decisions, "promoted_single_assignment_temp", "single_store_single_load_stack_neutral_safe_known_local_divmod_expression") {
				t.Fatalf("decisions = %#v, want stack-neutral safe known-local div/mod expression promotion", row.Decisions)
			}

			beforeExit := runOptLinuxX64(t, before.Funcs, "before-separated-safe-known-local-divmod-mem2reg-"+tc.name)
			afterExit := runOptLinuxX64(t, prog.Funcs, "after-separated-safe-known-local-divmod-mem2reg-"+tc.name)
			if beforeExit != afterExit || afterExit != tc.wantExit {
				t.Fatalf("exit mismatch before=%d after=%d, want both %d", beforeExit, afterExit, tc.wantExit)
			}
		})
	}
}

func TestMem2RegPassPromotesSeparatedSafeConstArithmeticTempWithStackNeutralWork(t *testing.T) {
	cases := []struct {
		name     string
		kind     ir.IRInstrKind
		op       string
		left     int32
		right    int32
		wantExit int
	}{
		{name: "addition", kind: ir.IRAddI32, op: "add_i32", left: 7, right: 5, wantExit: 21},
		{name: "subtraction", kind: ir.IRSubI32, op: "sub_i32", left: 13, right: 5, wantExit: 17},
		{name: "multiplication", kind: ir.IRMulI32, op: "mul_i32", left: 7, right: 5, wantExit: 44},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog := &ir.IRProgram{
				MainIndex: 0,
				MainName:  "main",
				Funcs: []ir.IRFunc{{
					Name:        "main",
					LocalSlots:  2,
					ReturnSlots: 1,
					Instrs: []ir.IRInstr{
						{Kind: ir.IRConstI32, Imm: tc.left},
						{Kind: ir.IRConstI32, Imm: tc.right},
						{Kind: tc.kind},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: 9},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: ir.IRLoadLocal, Local: 1},
						{Kind: ir.IRAddI32},
						{Kind: ir.IRReturn},
					},
				}},
			}
			before := cloneProgram(prog)

			report, err := NewManager().Run(prog, Mem2RegPass())
			if err != nil {
				t.Fatalf("Run Mem2RegPass: %v", err)
			}
			row := report.Passes[0]
			after := row.AfterDump
			for _, want := range []string{
				"const_i32 " + strconv.FormatInt(int64(tc.left), 10),
				"const_i32 " + strconv.FormatInt(int64(tc.right), 10),
				tc.op,
				"const_i32 9",
				"store_local local:1",
				"load_local local:1",
				"add_i32",
				"return",
			} {
				if !strings.Contains(after, want) {
					t.Fatalf("after dump missing %q:\n%s", want, after)
				}
			}
			for _, forbidden := range []string{"store_local local:0", "load_local local:0"} {
				if strings.Contains(after, forbidden) {
					t.Fatalf("after dump still contains promoted arithmetic temp %q:\n%s", forbidden, after)
				}
			}
			if !hasDecision(row.Decisions, "promoted_single_assignment_temp", "single_store_single_load_stack_neutral_safe_const_arithmetic_expression") {
				t.Fatalf("decisions = %#v, want stack-neutral safe const arithmetic expression promotion", row.Decisions)
			}

			beforeExit := runOptLinuxX64(t, before.Funcs, "before-separated-safe-arithmetic-mem2reg-"+tc.name)
			afterExit := runOptLinuxX64(t, prog.Funcs, "after-separated-safe-arithmetic-mem2reg-"+tc.name)
			if beforeExit != afterExit || afterExit != tc.wantExit {
				t.Fatalf("exit mismatch before=%d after=%d, want both %d", beforeExit, afterExit, tc.wantExit)
			}
		})
	}
}

func TestMem2RegPassRejectsSeparatedUnsafeConstArithmeticTemp(t *testing.T) {
	cases := []struct {
		name  string
		kind  ir.IRInstrKind
		op    string
		left  int32
		right int32
	}{
		{name: "addition overflow", kind: ir.IRAddI32, op: "add_i32", left: 2147483647, right: 1},
		{name: "subtraction overflow", kind: ir.IRSubI32, op: "sub_i32", left: -2147483648, right: 1},
		{name: "multiplication overflow", kind: ir.IRMulI32, op: "mul_i32", left: 50000, right: 50000},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog := &ir.IRProgram{
				MainIndex: 0,
				MainName:  "main",
				Funcs: []ir.IRFunc{{
					Name:        "main",
					LocalSlots:  2,
					ReturnSlots: 1,
					Instrs: []ir.IRInstr{
						{Kind: ir.IRConstI32, Imm: tc.left},
						{Kind: ir.IRConstI32, Imm: tc.right},
						{Kind: tc.kind},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: 9},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: ir.IRReturn},
					},
				}},
			}
			beforeDump := FormatProgram(prog)

			report, err := NewManager().Run(prog, Mem2RegPass())
			if err != nil {
				t.Fatalf("Run Mem2RegPass: %v", err)
			}
			row := report.Passes[0]
			if row.AfterDump != beforeDump {
				t.Fatalf("unsafe const arithmetic temp changed unexpectedly:\nbefore:\n%s\nafter:\n%s", beforeDump, row.AfterDump)
			}
			for _, want := range []string{"store_local local:0", "load_local local:0"} {
				if !strings.Contains(row.AfterDump, want) {
					t.Fatalf("after dump missing preserved unsafe arithmetic temp %q:\n%s", want, row.AfterDump)
				}
			}
			if got := countDumpOccurrences(row.AfterDump, tc.op); got != 1 {
				t.Fatalf("%s count after = %d, want 1:\n%s", tc.op, got, row.AfterDump)
			}
			if !hasDecision(row.Decisions, "not_promoted", "producer_not_available") {
				t.Fatalf("decisions = %#v, want unsafe const arithmetic producer rejection", row.Decisions)
			}
		})
	}
}

func TestMem2RegPassPromotesSeparatedSafeKnownLocalArithmeticTempWithStackNeutralWork(t *testing.T) {
	cases := []struct {
		name     string
		kind     ir.IRInstrKind
		op       string
		left     int32
		right    int32
		wantExit int
	}{
		{name: "addition", kind: ir.IRAddI32, op: "add_i32", left: 7, right: 5, wantExit: 21},
		{name: "subtraction", kind: ir.IRSubI32, op: "sub_i32", left: 13, right: 5, wantExit: 17},
		{name: "multiplication", kind: ir.IRMulI32, op: "mul_i32", left: 7, right: 5, wantExit: 44},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog := &ir.IRProgram{
				MainIndex: 0,
				MainName:  "main",
				Funcs: []ir.IRFunc{{
					Name:        "main",
					LocalSlots:  4,
					ReturnSlots: 1,
					Instrs: []ir.IRInstr{
						{Kind: ir.IRConstI32, Imm: 0},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: tc.left},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: 0},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRConstI32, Imm: tc.right},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: ir.IRLoadLocal, Local: 1},
						{Kind: tc.kind},
						{Kind: ir.IRStoreLocal, Local: 2},
						{Kind: ir.IRConstI32, Imm: 9},
						{Kind: ir.IRStoreLocal, Local: 3},
						{Kind: ir.IRLoadLocal, Local: 2},
						{Kind: ir.IRLoadLocal, Local: 3},
						{Kind: ir.IRAddI32},
						{Kind: ir.IRReturn},
					},
				}},
			}
			before := cloneProgram(prog)

			report, err := NewManager().Run(prog, Mem2RegPass())
			if err != nil {
				t.Fatalf("Run Mem2RegPass: %v", err)
			}
			row := report.Passes[0]
			after := row.AfterDump
			for _, want := range []string{
				"load_local local:0",
				"load_local local:1",
				tc.op,
				"const_i32 9",
				"store_local local:3",
				"load_local local:3",
				"add_i32",
				"return",
			} {
				if !strings.Contains(after, want) {
					t.Fatalf("after dump missing %q:\n%s", want, after)
				}
			}
			for _, forbidden := range []string{"store_local local:2", "load_local local:2"} {
				if strings.Contains(after, forbidden) {
					t.Fatalf("after dump still contains promoted known-local arithmetic temp %q:\n%s", forbidden, after)
				}
			}
			if !hasDecision(row.Decisions, "promoted_single_assignment_temp", "single_store_single_load_stack_neutral_safe_known_local_arithmetic_expression") {
				t.Fatalf("decisions = %#v, want stack-neutral safe known-local arithmetic expression promotion", row.Decisions)
			}

			beforeExit := runOptLinuxX64(t, before.Funcs, "before-separated-safe-known-local-arithmetic-mem2reg-"+tc.name)
			afterExit := runOptLinuxX64(t, prog.Funcs, "after-separated-safe-known-local-arithmetic-mem2reg-"+tc.name)
			if beforeExit != afterExit || afterExit != tc.wantExit {
				t.Fatalf("exit mismatch before=%d after=%d, want both %d", beforeExit, afterExit, tc.wantExit)
			}
		})
	}
}

func TestMem2RegPassRejectsSeparatedUnsafeKnownLocalArithmeticTemp(t *testing.T) {
	cases := []struct {
		name  string
		kind  ir.IRInstrKind
		op    string
		left  int32
		right int32
	}{
		{name: "addition overflow", kind: ir.IRAddI32, op: "add_i32", left: 2147483647, right: 1},
		{name: "subtraction overflow", kind: ir.IRSubI32, op: "sub_i32", left: -2147483648, right: 1},
		{name: "multiplication overflow", kind: ir.IRMulI32, op: "mul_i32", left: 50000, right: 50000},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog := &ir.IRProgram{
				MainIndex: 0,
				MainName:  "main",
				Funcs: []ir.IRFunc{{
					Name:        "main",
					LocalSlots:  4,
					ReturnSlots: 1,
					Instrs: []ir.IRInstr{
						{Kind: ir.IRConstI32, Imm: 0},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: tc.left},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: 0},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRConstI32, Imm: tc.right},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: ir.IRLoadLocal, Local: 1},
						{Kind: tc.kind},
						{Kind: ir.IRStoreLocal, Local: 2},
						{Kind: ir.IRConstI32, Imm: 9},
						{Kind: ir.IRStoreLocal, Local: 3},
						{Kind: ir.IRLoadLocal, Local: 2},
						{Kind: ir.IRReturn},
					},
				}},
			}
			beforeDump := FormatProgram(prog)

			report, err := NewManager().Run(prog, Mem2RegPass())
			if err != nil {
				t.Fatalf("Run Mem2RegPass: %v", err)
			}
			row := report.Passes[0]
			if row.AfterDump != beforeDump {
				t.Fatalf("unsafe known-local arithmetic temp changed unexpectedly:\nbefore:\n%s\nafter:\n%s", beforeDump, row.AfterDump)
			}
			for _, want := range []string{"store_local local:2", "load_local local:2"} {
				if !strings.Contains(row.AfterDump, want) {
					t.Fatalf("after dump missing preserved unsafe known-local arithmetic temp %q:\n%s", want, row.AfterDump)
				}
			}
			if got := countDumpOccurrences(row.AfterDump, tc.op); got != 1 {
				t.Fatalf("%s count after = %d, want 1:\n%s", tc.op, got, row.AfterDump)
			}
			if !hasDecision(row.Decisions, "not_promoted", "producer_not_available") {
				t.Fatalf("decisions = %#v, want unsafe known-local arithmetic producer rejection", row.Decisions)
			}
		})
	}
}

func TestMem2RegPassRejectsSeparatedSafeKnownLocalArithmeticTempWhenSourceLocalMutates(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  3,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRConstI32, Imm: 100},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRReturn},
			},
		}},
	}
	before := cloneProgram(prog)
	beforeDump := FormatProgram(prog)

	report, err := NewManager().Run(prog, Mem2RegPass())
	if err != nil {
		t.Fatalf("Run Mem2RegPass: %v", err)
	}
	row := report.Passes[0]
	if row.AfterDump != beforeDump {
		t.Fatalf("mutating safe known-local arithmetic source local changed unexpectedly:\nbefore:\n%s\nafter:\n%s", beforeDump, row.AfterDump)
	}
	if !hasDecision(row.Decisions, "not_promoted", "source_local_modified_before_load") {
		t.Fatalf("decisions = %#v, want explicit safe known-local arithmetic source-local mutation rejection", row.Decisions)
	}

	beforeExit := runOptLinuxX64(t, before.Funcs, "before-safe-known-local-arithmetic-source-mutates-mem2reg")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-safe-known-local-arithmetic-source-mutates-mem2reg")
	if beforeExit != afterExit || afterExit != 12 {
		t.Fatalf("exit mismatch before=%d after=%d, want both 12", beforeExit, afterExit)
	}
}

func TestMem2RegPassRejectsSeparatedUnsafeConstDenominatorDivModTemp(t *testing.T) {
	cases := []struct {
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

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog := &ir.IRProgram{
				MainIndex: 0,
				MainName:  "main",
				Funcs: []ir.IRFunc{{
					Name:        "main",
					LocalSlots:  3,
					ReturnSlots: 1,
					Instrs: []ir.IRInstr{
						{Kind: ir.IRConstI32, Imm: 12},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: tc.denom},
						{Kind: tc.kind},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRConstI32, Imm: 7},
						{Kind: ir.IRStoreLocal, Local: 2},
						{Kind: ir.IRLoadLocal, Local: 1},
						{Kind: ir.IRConstI32, Imm: 0},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRReturn},
					},
				}},
			}
			beforeDump := FormatProgram(prog)

			report, err := NewManager().Run(prog, Mem2RegPass())
			if err != nil {
				t.Fatalf("Run Mem2RegPass: %v", err)
			}
			row := report.Passes[0]
			if row.AfterDump != beforeDump {
				t.Fatalf("unsafe denominator div/mod temp changed unexpectedly:\nbefore:\n%s\nafter:\n%s", beforeDump, row.AfterDump)
			}
			for _, want := range []string{"store_local local:1", "load_local local:1"} {
				if !strings.Contains(row.AfterDump, want) {
					t.Fatalf("after dump missing preserved temp %q:\n%s", want, row.AfterDump)
				}
			}
			if got := countDumpOccurrences(row.AfterDump, tc.op); got != 1 {
				t.Fatalf("%s count after = %d, want 1:\n%s", tc.op, got, row.AfterDump)
			}
			if !hasDecision(row.Decisions, "not_promoted", "producer_not_available") {
				t.Fatalf("decisions = %#v, want unsafe denominator producer rejection", row.Decisions)
			}
		})
	}
}

func TestMem2RegPassRejectsSeparatedUnsafeKnownLocalDivModTemp(t *testing.T) {
	cases := []struct {
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

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog := &ir.IRProgram{
				MainIndex: 0,
				MainName:  "main",
				Funcs: []ir.IRFunc{{
					Name:        "main",
					LocalSlots:  4,
					ReturnSlots: 1,
					Instrs: []ir.IRInstr{
						{Kind: ir.IRConstI32, Imm: 0},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: 12},
						{Kind: ir.IRStoreLocal, Local: 0},
						{Kind: ir.IRConstI32, Imm: 0},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRConstI32, Imm: tc.denom},
						{Kind: ir.IRStoreLocal, Local: 1},
						{Kind: ir.IRLoadLocal, Local: 0},
						{Kind: ir.IRLoadLocal, Local: 1},
						{Kind: tc.kind},
						{Kind: ir.IRStoreLocal, Local: 2},
						{Kind: ir.IRConstI32, Imm: 7},
						{Kind: ir.IRStoreLocal, Local: 3},
						{Kind: ir.IRLoadLocal, Local: 2},
						{Kind: ir.IRReturn},
					},
				}},
			}
			beforeDump := FormatProgram(prog)

			report, err := NewManager().Run(prog, Mem2RegPass())
			if err != nil {
				t.Fatalf("Run Mem2RegPass: %v", err)
			}
			row := report.Passes[0]
			if row.AfterDump != beforeDump {
				t.Fatalf("unsafe known-local div/mod temp changed unexpectedly:\nbefore:\n%s\nafter:\n%s", beforeDump, row.AfterDump)
			}
			for _, want := range []string{"store_local local:2", "load_local local:2"} {
				if !strings.Contains(row.AfterDump, want) {
					t.Fatalf("after dump missing preserved unsafe known-local div/mod temp %q:\n%s", want, row.AfterDump)
				}
			}
			if got := countDumpOccurrences(row.AfterDump, tc.op); got != 1 {
				t.Fatalf("%s count after = %d, want 1:\n%s", tc.op, got, row.AfterDump)
			}
			if !hasDecision(row.Decisions, "not_promoted", "producer_not_available") {
				t.Fatalf("decisions = %#v, want unsafe known-local div/mod producer rejection", row.Decisions)
			}
		})
	}
}

func TestMem2RegPassPromotesSeparatedSafeConstUnaryNegTempWithStackNeutralWork(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: -6},
				{Kind: ir.IRNegI32},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		}},
	}
	before := cloneProgram(prog)

	report, err := NewManager().Run(prog, Mem2RegPass())
	if err != nil {
		t.Fatalf("Run Mem2RegPass: %v", err)
	}
	row := report.Passes[0]
	after := row.AfterDump
	for _, want := range []string{"const_i32 -6", "neg_i32", "const_i32 7", "store_local local:1", "const_i32 2", "add_i32", "return"} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	for _, forbidden := range []string{"store_local local:0", "load_local local:0"} {
		if strings.Contains(after, forbidden) {
			t.Fatalf("after dump still contains promoted unary neg temp %q:\n%s", forbidden, after)
		}
	}
	if !hasDecision(row.Decisions, "promoted_single_assignment_temp", "single_store_single_load_stack_neutral_safe_const_unary_neg_expression") {
		t.Fatalf("decisions = %#v, want stack-neutral safe const unary neg expression promotion", row.Decisions)
	}

	beforeExit := runOptLinuxX64(t, before.Funcs, "before-separated-safe-unary-neg-mem2reg")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-separated-safe-unary-neg-mem2reg")
	if beforeExit != afterExit || afterExit != 8 {
		t.Fatalf("exit mismatch before=%d after=%d, want both 8", beforeExit, afterExit)
	}
}

func TestMem2RegPassRejectsSeparatedUnsafeConstUnaryNegTemp(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: -2147483648},
				{Kind: ir.IRNegI32},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRReturn},
			},
		}},
	}
	beforeDump := FormatProgram(prog)

	report, err := NewManager().Run(prog, Mem2RegPass())
	if err != nil {
		t.Fatalf("Run Mem2RegPass: %v", err)
	}
	row := report.Passes[0]
	if row.AfterDump != beforeDump {
		t.Fatalf("unsafe unary neg temp changed unexpectedly:\nbefore:\n%s\nafter:\n%s", beforeDump, row.AfterDump)
	}
	for _, want := range []string{"const_i32 -2147483648", "neg_i32", "store_local local:0", "load_local local:0"} {
		if !strings.Contains(row.AfterDump, want) {
			t.Fatalf("after dump missing preserved unsafe unary neg temp %q:\n%s", want, row.AfterDump)
		}
	}
	if !hasDecision(row.Decisions, "not_promoted", "producer_not_available") {
		t.Fatalf("decisions = %#v, want unsafe unary neg producer rejection", row.Decisions)
	}
}

func TestMem2RegPassPromotesSeparatedSafeKnownLocalUnaryNegTempWithStackNeutralWork(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  3,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: -6},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRNegI32},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		}},
	}
	before := cloneProgram(prog)

	report, err := NewManager().Run(prog, Mem2RegPass())
	if err != nil {
		t.Fatalf("Run Mem2RegPass: %v", err)
	}
	row := report.Passes[0]
	after := row.AfterDump
	for _, want := range []string{"load_local local:0", "neg_i32", "const_i32 7", "store_local local:2", "const_i32 2", "add_i32", "return"} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	for _, forbidden := range []string{"store_local local:1", "load_local local:1"} {
		if strings.Contains(after, forbidden) {
			t.Fatalf("after dump still contains promoted known-local unary neg temp %q:\n%s", forbidden, after)
		}
	}
	if !hasDecision(row.Decisions, "promoted_single_assignment_temp", "single_store_single_load_stack_neutral_safe_known_local_unary_neg_expression") {
		t.Fatalf("decisions = %#v, want stack-neutral safe known-local unary neg expression promotion", row.Decisions)
	}

	beforeExit := runOptLinuxX64(t, before.Funcs, "before-separated-safe-known-local-unary-neg-mem2reg")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-separated-safe-known-local-unary-neg-mem2reg")
	if beforeExit != afterExit || afterExit != 8 {
		t.Fatalf("exit mismatch before=%d after=%d, want both 8", beforeExit, afterExit)
	}
}

func TestMem2RegPassRejectsSeparatedUnsafeKnownLocalUnaryNegTemp(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  3,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: -2147483648},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRNegI32},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRReturn},
			},
		}},
	}
	beforeDump := FormatProgram(prog)

	report, err := NewManager().Run(prog, Mem2RegPass())
	if err != nil {
		t.Fatalf("Run Mem2RegPass: %v", err)
	}
	row := report.Passes[0]
	if row.AfterDump != beforeDump {
		t.Fatalf("unsafe known-local unary neg temp changed unexpectedly:\nbefore:\n%s\nafter:\n%s", beforeDump, row.AfterDump)
	}
	for _, want := range []string{"const_i32 -2147483648", "load_local local:0", "neg_i32", "store_local local:1", "load_local local:1"} {
		if !strings.Contains(row.AfterDump, want) {
			t.Fatalf("after dump missing preserved unsafe known-local unary neg temp %q:\n%s", want, row.AfterDump)
		}
	}
	if !hasDecision(row.Decisions, "not_promoted", "producer_not_available") {
		t.Fatalf("decisions = %#v, want unsafe known-local unary neg producer rejection", row.Decisions)
	}
}

func TestMem2RegPassRejectsSeparatedSafeKnownLocalUnaryNegTempWhenSourceLocalMutates(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: -6},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRNegI32},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 100},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRReturn},
			},
		}},
	}
	before := cloneProgram(prog)
	beforeDump := FormatProgram(prog)

	report, err := NewManager().Run(prog, Mem2RegPass())
	if err != nil {
		t.Fatalf("Run Mem2RegPass: %v", err)
	}
	row := report.Passes[0]
	if row.AfterDump != beforeDump {
		t.Fatalf("mutating safe known-local unary neg source local changed unexpectedly:\nbefore:\n%s\nafter:\n%s", beforeDump, row.AfterDump)
	}
	if !hasDecision(row.Decisions, "not_promoted", "source_local_modified_before_load") {
		t.Fatalf("decisions = %#v, want explicit safe known-local unary neg source-local mutation rejection", row.Decisions)
	}

	beforeExit := runOptLinuxX64(t, before.Funcs, "before-safe-known-local-unary-neg-source-mutates-mem2reg")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-safe-known-local-unary-neg-source-mutates-mem2reg")
	if beforeExit != afterExit || afterExit != 6 {
		t.Fatalf("exit mismatch before=%d after=%d, want both 6", beforeExit, afterExit)
	}
}

func TestMem2RegPassRejectsSeparatedTempWhenSourceLocalMutates(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		}},
	}
	before := cloneProgram(prog)
	beforeDump := FormatProgram(prog)

	report, err := NewManager().Run(prog, Mem2RegPass())
	if err != nil {
		t.Fatalf("Run Mem2RegPass: %v", err)
	}
	row := report.Passes[0]
	if row.AfterDump != beforeDump {
		t.Fatalf("mutating source local changed unexpectedly:\nbefore:\n%s\nafter:\n%s", beforeDump, row.AfterDump)
	}
	if !hasDecision(row.Decisions, "not_promoted", "source_local_modified_before_load") {
		t.Fatalf("decisions = %#v, want explicit source-local mutation rejection", row.Decisions)
	}

	beforeExit := runOptLinuxX64(t, before.Funcs, "before-source-mutates-mem2reg")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-source-mutates-mem2reg")
	if beforeExit != afterExit || afterExit != 7 {
		t.Fatalf("exit mismatch before=%d after=%d, want both 7", beforeExit, afterExit)
	}
}

func TestMem2RegPassRejectsSeparatedSafeDivModTempWhenSourceLocalMutates(t *testing.T) {
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
				{Kind: ir.IRConstI32, Imm: 3},
				{Kind: ir.IRDivI32},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 18},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRReturn},
			},
		}},
	}
	before := cloneProgram(prog)
	beforeDump := FormatProgram(prog)

	report, err := NewManager().Run(prog, Mem2RegPass())
	if err != nil {
		t.Fatalf("Run Mem2RegPass: %v", err)
	}
	row := report.Passes[0]
	if row.AfterDump != beforeDump {
		t.Fatalf("mutating safe div/mod source local changed unexpectedly:\nbefore:\n%s\nafter:\n%s", beforeDump, row.AfterDump)
	}
	if !hasDecision(row.Decisions, "not_promoted", "source_local_modified_before_load") {
		t.Fatalf("decisions = %#v, want explicit safe div/mod source-local mutation rejection", row.Decisions)
	}

	beforeExit := runOptLinuxX64(t, before.Funcs, "before-safe-divmod-source-mutates-mem2reg")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-safe-divmod-source-mutates-mem2reg")
	if beforeExit != afterExit || afterExit != 4 {
		t.Fatalf("exit mismatch before=%d after=%d, want both 4", beforeExit, afterExit)
	}
}

func TestMem2RegPassRejectsSeparatedSafeKnownLocalDivModTempWhenSourceLocalMutates(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  3,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 20},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRDivI32},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRReturn},
			},
		}},
	}
	before := cloneProgram(prog)
	beforeDump := FormatProgram(prog)

	report, err := NewManager().Run(prog, Mem2RegPass())
	if err != nil {
		t.Fatalf("Run Mem2RegPass: %v", err)
	}
	row := report.Passes[0]
	if row.AfterDump != beforeDump {
		t.Fatalf("mutating safe known-local div/mod source local changed unexpectedly:\nbefore:\n%s\nafter:\n%s", beforeDump, row.AfterDump)
	}
	if !hasDecision(row.Decisions, "not_promoted", "source_local_modified_before_load") {
		t.Fatalf("decisions = %#v, want explicit safe known-local div/mod source-local mutation rejection", row.Decisions)
	}

	beforeExit := runOptLinuxX64(t, before.Funcs, "before-safe-known-local-divmod-source-mutates-mem2reg")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-safe-known-local-divmod-source-mutates-mem2reg")
	if beforeExit != afterExit || afterExit != 4 {
		t.Fatalf("exit mismatch before=%d after=%d, want both 4", beforeExit, afterExit)
	}
}

func TestMem2RegPassRejectsSeparatedComparisonTempWhenSourceLocalMutates(t *testing.T) {
	prog := &ir.IRProgram{
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
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRCmpLtI32},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 8},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRReturn},
			},
		}},
	}
	before := cloneProgram(prog)
	beforeDump := FormatProgram(prog)

	report, err := NewManager().Run(prog, Mem2RegPass())
	if err != nil {
		t.Fatalf("Run Mem2RegPass: %v", err)
	}
	row := report.Passes[0]
	if row.AfterDump != beforeDump {
		t.Fatalf("mutating comparison source local changed unexpectedly:\nbefore:\n%s\nafter:\n%s", beforeDump, row.AfterDump)
	}
	if !hasDecision(row.Decisions, "not_promoted", "source_local_modified_before_load") {
		t.Fatalf("decisions = %#v, want explicit comparison source-local mutation rejection", row.Decisions)
	}

	beforeExit := runOptLinuxX64(t, before.Funcs, "before-comparison-source-mutates-mem2reg")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-comparison-source-mutates-mem2reg")
	if beforeExit != afterExit || afterExit != 1 {
		t.Fatalf("exit mismatch before=%d after=%d, want both 1", beforeExit, afterExit)
	}
}

func TestMem2RegPassReportsMultiLoadTempWithoutClaimingPromotion(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 4},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		}},
	}

	before := FormatProgram(prog)
	report, err := NewManager().Run(prog, Mem2RegPass())
	if err != nil {
		t.Fatalf("Run Mem2RegPass: %v", err)
	}
	row := report.Passes[0]
	if row.AfterDump != before {
		t.Fatalf("multi-load local changed unexpectedly:\nbefore:\n%s\nafter:\n%s", before, row.AfterDump)
	}
	if !hasDecision(row.Decisions, "not_promoted", "local_not_single_load") {
		t.Fatalf("decisions = %#v, want explicit multi-load non-promotion", row.Decisions)
	}
}

func singleAssignmentTempProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 4},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRMulI32},
				{Kind: ir.IRReturn},
			},
		}},
	}
}
