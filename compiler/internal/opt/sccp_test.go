package opt

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestSCCPPassFoldsConstantBranchesAndReportsDecisions(t *testing.T) {
	tests := []struct {
		name       string
		prog       *ir.IRProgram
		wantAction string
		wantPrune  bool
		wantExit   int
		forbidden  []string
		required   []string
	}{
		{
			name:       "constant_zero_takes_branch",
			prog:       constantZeroBranchProgram(),
			wantAction: "folded_const_zero_branch",
			wantPrune:  true,
			wantExit:   7,
			forbidden:  []string{"const_i32 0\n  jmp_if_zero label:1", "const_i32 99"},
			required:   []string{"jmp label:1", "const_i32 7"},
		},
		{
			name:       "constant_nonzero_falls_through",
			prog:       constantNonZeroBranchProgram(),
			wantAction: "folded_const_nonzero_fallthrough",
			wantExit:   42,
			forbidden:  []string{"const_i32 1", "jmp_if_zero label:1"},
			required:   []string{"const_i32 42", "return"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			before := cloneProgram(tc.prog)
			report, err := NewManager().Run(tc.prog, SCCPPass())
			if err != nil {
				t.Fatalf("Run SCCPPass: %v", err)
			}
			row := report.Passes[0]
			if row.Name != "sccp-constant-branch" || !row.TranslationValidated {
				t.Fatalf("pass row = %#v", row)
			}
			if !hasDecision(row.Decisions, tc.wantAction, "constant_condition") {
				t.Fatalf("decisions missing %s: %#v", tc.wantAction, row.Decisions)
			}
			if tc.wantPrune && !hasDecision(row.Decisions, "pruned_unreachable_fallthrough", "constant_branch_reachability") {
				t.Fatalf("prune decision missing: %#v", row.Decisions)
			}
			for _, want := range tc.required {
				if !strings.Contains(row.AfterDump, want) {
					t.Fatalf("after dump missing %q:\n%s", want, row.AfterDump)
				}
			}
			for _, forbidden := range tc.forbidden {
				if strings.Contains(row.AfterDump, forbidden) {
					t.Fatalf("after dump still contains %q:\n%s", forbidden, row.AfterDump)
				}
			}
			beforeExit := runOptLinuxX64(t, before.Funcs, tc.name+"-before-sccp")
			afterExit := runOptLinuxX64(t, tc.prog.Funcs, tc.name+"-after-sccp")
			if beforeExit != afterExit || afterExit != tc.wantExit {
				t.Fatalf("native exits before=%d after=%d want %d", beforeExit, afterExit, tc.wantExit)
			}
		})
	}
}

func TestSCCPPassReportsDynamicBranchesWithoutClaimingCoverage(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if !hasDecision(row.Decisions, "not_folded", "dynamic_condition") {
		t.Fatalf("dynamic branch decision missing: %#v", row.Decisions)
	}
	if !strings.Contains(row.AfterDump, "load_local local:0") || !strings.Contains(row.AfterDump, "jmp_if_zero label:1") {
		t.Fatalf("dynamic branch changed unexpectedly:\n%s", row.AfterDump)
	}
}

func TestSCCPPassPrunesOnlyUntilNextLabel(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 13},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if strings.Contains(row.AfterDump, "const_i32 99") {
		t.Fatalf("unreachable fallthrough before label was not pruned:\n%s", row.AfterDump)
	}
	for _, want := range []string{"label:1", "const_i32 13", "label:2", "const_i32 7"} {
		if !strings.Contains(row.AfterDump, want) {
			t.Fatalf("after dump missing preserved label block %q:\n%s", want, row.AfterDump)
		}
	}
	if !hasDecision(row.Decisions, "pruned_unreachable_fallthrough", "constant_branch_reachability") {
		t.Fatalf("prune decision missing: %#v", row.Decisions)
	}
}

func TestSCCPPassFoldsKnownLocalConstantBranch(t *testing.T) {
	prog := knownLocalZeroBranchProgram()
	before := cloneProgram(prog)

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if !hasDecision(row.Decisions, "folded_known_local_zero_branch", "constant_local_condition") {
		t.Fatalf("known-local zero decision missing: %#v", row.Decisions)
	}
	if !hasDecision(row.Decisions, "pruned_unreachable_fallthrough", "constant_branch_reachability") {
		t.Fatalf("prune decision missing: %#v", row.Decisions)
	}
	for _, forbidden := range []string{"load_local local:0\n  jmp_if_zero label:1", "const_i32 99"} {
		if strings.Contains(row.AfterDump, forbidden) {
			t.Fatalf("after dump still contains %q:\n%s", forbidden, row.AfterDump)
		}
	}
	for _, want := range []string{"store_local local:0", "jmp label:1", "const_i32 7"} {
		if !strings.Contains(row.AfterDump, want) {
			t.Fatalf("after dump missing %q:\n%s", want, row.AfterDump)
		}
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "known-local-zero-before-sccp")
	afterExit := runOptLinuxX64(t, prog.Funcs, "known-local-zero-after-sccp")
	if beforeExit != afterExit || afterExit != 7 {
		t.Fatalf("native exits before=%d after=%d want 7", beforeExit, afterExit)
	}
}

func TestSCCPPassDoesNotFoldStaleLocalConstantBranch(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
	before := cloneProgram(prog)

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if !hasDecision(row.Decisions, "folded_known_local_nonzero_fallthrough", "constant_local_condition") {
		t.Fatalf("known-local nonzero decision missing: %#v", row.Decisions)
	}
	if strings.Contains(row.AfterDump, "load_local local:0\n  jmp_if_zero label:1") {
		t.Fatalf("known-local branch was not folded:\n%s", row.AfterDump)
	}
	if !strings.Contains(row.AfterDump, "const_i32 42") {
		t.Fatalf("after dump missing fallthrough return:\n%s", row.AfterDump)
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "stale-local-before-sccp")
	afterExit := runOptLinuxX64(t, prog.Funcs, "stale-local-after-sccp")
	if beforeExit != afterExit || afterExit != 42 {
		t.Fatalf("native exits before=%d after=%d want 42", beforeExit, afterExit)
	}
}

func TestSCCPPassClearsKnownLocalFactsAtLabels(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if hasDecision(row.Decisions, "folded_known_local_zero_branch", "constant_local_condition") {
		t.Fatalf("known-local fact crossed a label: %#v", row.Decisions)
	}
	if !strings.Contains(row.AfterDump, "load_local local:0") || !strings.Contains(row.AfterDump, "jmp_if_zero label:2") {
		t.Fatalf("branch changed despite label boundary:\n%s", row.AfterDump)
	}
}

func TestSCCPPassPropagatesKnownLocalThroughSinglePredecessorLabel(t *testing.T) {
	prog := singlePredecessorKnownLocalBranchProgram()
	before := cloneProgram(prog)

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if !hasDecision(row.Decisions, "propagated_known_local_single_predecessor", "single_predecessor_label") {
		t.Fatalf("single-predecessor propagation decision missing: %#v", row.Decisions)
	}
	if !hasDecision(row.Decisions, "folded_known_local_zero_branch", "constant_local_condition") {
		t.Fatalf("known-local zero decision missing after label propagation: %#v", row.Decisions)
	}
	if !hasDecision(row.Decisions, "pruned_unreachable_fallthrough", "constant_branch_reachability") {
		t.Fatalf("prune decision missing after label propagation: %#v", row.Decisions)
	}
	for _, forbidden := range []string{"load_local local:0\n  jmp_if_zero label:2", "const_i32 99"} {
		if strings.Contains(row.AfterDump, forbidden) {
			t.Fatalf("after dump still contains %q:\n%s", forbidden, row.AfterDump)
		}
	}
	for _, want := range []string{"jmp label:1", "label:1", "jmp label:2", "const_i32 7"} {
		if !strings.Contains(row.AfterDump, want) {
			t.Fatalf("after dump missing %q:\n%s", want, row.AfterDump)
		}
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "single-pred-before-sccp")
	afterExit := runOptLinuxX64(t, prog.Funcs, "single-pred-after-sccp")
	if beforeExit != afterExit || afterExit != 7 {
		t.Fatalf("native exits before=%d after=%d want 7", beforeExit, afterExit)
	}
}

func TestSCCPPassDoesNotPropagateKnownLocalThroughMergeLabel(t *testing.T) {
	prog := mergeLabelKnownLocalBranchProgram()
	before := FormatProgram(prog)

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if hasDecision(row.Decisions, "propagated_known_local_single_predecessor", "single_predecessor_label") {
		t.Fatalf("known-local fact propagated through merge label: %#v", row.Decisions)
	}
	if hasDecision(row.Decisions, "folded_known_local_zero_branch", "constant_local_condition") {
		t.Fatalf("merge label branch was folded with path-sensitive ambiguity: %#v", row.Decisions)
	}
	if row.AfterDump != before {
		t.Fatalf("merge-label function changed unexpectedly:\nbefore:\n%s\nafter:\n%s", before, row.AfterDump)
	}
}

func TestSCCPPassPropagatesKnownLocalThroughForwardSinglePredecessorJump(t *testing.T) {
	prog := forwardSinglePredecessorKnownLocalBranchProgram()
	before := cloneProgram(prog)

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if !hasDecision(row.Decisions, "propagated_known_local_single_predecessor", "forward_single_predecessor_jump") {
		t.Fatalf("forward single-predecessor propagation decision missing: %#v", row.Decisions)
	}
	if !hasDecision(row.Decisions, "folded_known_local_zero_branch", "constant_local_condition") {
		t.Fatalf("known-local zero decision missing after forward propagation: %#v", row.Decisions)
	}
	if !hasDecision(row.Decisions, "pruned_unreachable_fallthrough", "constant_branch_reachability") {
		t.Fatalf("prune decision missing after forward propagation: %#v", row.Decisions)
	}
	for _, forbidden := range []string{"load_local local:0\n  jmp_if_zero label:2", "const_i32 99"} {
		if strings.Contains(row.AfterDump, forbidden) {
			t.Fatalf("after dump still contains %q:\n%s", forbidden, row.AfterDump)
		}
	}
	for _, want := range []string{"jmp label:1", "const_i32 11", "label:1", "jmp label:2", "const_i32 7"} {
		if !strings.Contains(row.AfterDump, want) {
			t.Fatalf("after dump missing %q:\n%s", want, row.AfterDump)
		}
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "forward-single-pred-before-sccp")
	afterExit := runOptLinuxX64(t, prog.Funcs, "forward-single-pred-after-sccp")
	if beforeExit != afterExit || afterExit != 7 {
		t.Fatalf("native exits before=%d after=%d want 7", beforeExit, afterExit)
	}
}

func TestSCCPPassDoesNotPropagateKnownLocalThroughForwardJumpWithFallthroughPredecessor(t *testing.T) {
	prog := forwardFallthroughPredecessorKnownLocalBranchProgram()
	before := FormatProgram(prog)

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if hasDecision(row.Decisions, "propagated_known_local_single_predecessor", "forward_single_predecessor_jump") {
		t.Fatalf("known-local fact propagated through label with fallthrough predecessor: %#v", row.Decisions)
	}
	if hasDecision(row.Decisions, "folded_known_local_zero_branch", "constant_local_condition") {
		t.Fatalf("fallthrough-predecessor label branch was folded with path-sensitive ambiguity: %#v", row.Decisions)
	}
	if row.AfterDump != before {
		t.Fatalf("fallthrough-predecessor function changed unexpectedly:\nbefore:\n%s\nafter:\n%s", before, row.AfterDump)
	}
}

func TestSCCPPassPropagatesKnownLocalThroughFoldedZeroBranchTarget(t *testing.T) {
	prog := foldedZeroBranchSinglePredecessorKnownLocalProgram()
	before := cloneProgram(prog)

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if !hasDecision(row.Decisions, "folded_known_local_zero_branch", "constant_local_condition") {
		t.Fatalf("first known-local zero branch decision missing: %#v", row.Decisions)
	}
	if !hasDecision(row.Decisions, "propagated_known_local_folded_zero_branch", "folded_zero_branch_forward_single_predecessor_jump") {
		t.Fatalf("folded zero-branch propagation decision missing: %#v", row.Decisions)
	}
	if got := countDecisions(row.Decisions, "folded_known_local_zero_branch", "constant_local_condition"); got != 2 {
		t.Fatalf("folded known-local zero branch decisions = %d, want 2: %#v", got, row.Decisions)
	}
	for _, forbidden := range []string{
		"load_local local:0\n  jmp_if_zero label:1",
		"load_local local:0\n  jmp_if_zero label:2",
		"const_i32 99",
		"const_i32 42",
	} {
		if strings.Contains(row.AfterDump, forbidden) {
			t.Fatalf("after dump still contains %q:\n%s", forbidden, row.AfterDump)
		}
	}
	for _, want := range []string{"jmp label:1", "label:1", "jmp label:2", "label:2", "const_i32 7"} {
		if !strings.Contains(row.AfterDump, want) {
			t.Fatalf("after dump missing %q:\n%s", want, row.AfterDump)
		}
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "folded-zero-branch-target-before-sccp")
	afterExit := runOptLinuxX64(t, prog.Funcs, "folded-zero-branch-target-after-sccp")
	if beforeExit != afterExit || afterExit != 7 {
		t.Fatalf("native exits before=%d after=%d want 7", beforeExit, afterExit)
	}
}

func TestSCCPPassDoesNotPropagateFoldedZeroBranchThroughFallthroughTarget(t *testing.T) {
	prog := foldedZeroBranchFallthroughTargetKnownLocalProgram()
	before := FormatProgram(prog)

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if hasDecision(row.Decisions, "propagated_known_local_folded_zero_branch", "folded_zero_branch_single_predecessor_label") {
		t.Fatalf("known-local fact propagated through folded branch target with fallthrough predecessor: %#v", row.Decisions)
	}
	if countDecisions(row.Decisions, "folded_known_local_zero_branch", "constant_local_condition") != 1 {
		t.Fatalf("only the first known-local zero branch should fold: %#v", row.Decisions)
	}
	if !strings.Contains(row.AfterDump, "load_local local:0\n  jmp_if_zero label:2") {
		t.Fatalf("fallthrough-target branch changed despite ambiguous predecessor:\n%s", row.AfterDump)
	}
	if row.AfterDump == before {
		t.Fatalf("expected first known-local branch to fold while preserving target ambiguity")
	}
}

func TestSCCPPassPropagatesKnownLocalThroughFoldedNonzeroFallthroughLabel(t *testing.T) {
	prog := foldedNonzeroFallthroughOnlyLabelKnownLocalProgram()
	before := cloneProgram(prog)

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if !hasDecision(row.Decisions, "propagated_known_local_folded_nonzero_fallthrough", "folded_nonzero_fallthrough_label") {
		t.Fatalf("folded nonzero fallthrough propagation decision missing: %#v", row.Decisions)
	}
	if got := countDecisions(row.Decisions, "folded_known_local_nonzero_fallthrough", "constant_local_condition"); got != 2 {
		t.Fatalf("folded known-local nonzero branch decisions = %d, want 2: %#v", got, row.Decisions)
	}
	for _, forbidden := range []string{
		"load_local local:0\n  jmp_if_zero label:9",
		"load_local local:0\n  jmp_if_zero label:2",
	} {
		if strings.Contains(row.AfterDump, forbidden) {
			t.Fatalf("after dump still contains %q:\n%s", forbidden, row.AfterDump)
		}
	}
	for _, want := range []string{"label:1", "const_i32 42", "label:2", "label:9"} {
		if !strings.Contains(row.AfterDump, want) {
			t.Fatalf("after dump missing %q:\n%s", want, row.AfterDump)
		}
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "folded-nonzero-fallthrough-label-before-sccp")
	afterExit := runOptLinuxX64(t, prog.Funcs, "folded-nonzero-fallthrough-label-after-sccp")
	if beforeExit != afterExit || afterExit != 42 {
		t.Fatalf("native exits before=%d after=%d want 42", beforeExit, afterExit)
	}
}

func TestSCCPPassDoesNotPropagateFoldedNonzeroFallthroughThroughExplicitIncomingLabel(t *testing.T) {
	prog := foldedNonzeroFallthroughExplicitIncomingLabelKnownLocalProgram()
	before := cloneProgram(prog)

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if hasDecision(row.Decisions, "propagated_known_local_folded_nonzero_fallthrough", "folded_nonzero_fallthrough_label") {
		t.Fatalf("known-local fact propagated through explicit-incoming fallthrough label: %#v", row.Decisions)
	}
	if got := countDecisions(row.Decisions, "folded_known_local_nonzero_fallthrough", "constant_local_condition"); got != 1 {
		t.Fatalf("folded known-local nonzero branch decisions = %d, want 1: %#v", got, row.Decisions)
	}
	if !strings.Contains(row.AfterDump, "load_local local:0\n  jmp_if_zero label:2") {
		t.Fatalf("explicit-incoming label branch changed despite merge ambiguity:\n%s", row.AfterDump)
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "folded-nonzero-explicit-incoming-before-sccp")
	afterExit := runOptLinuxX64(t, prog.Funcs, "folded-nonzero-explicit-incoming-after-sccp")
	if beforeExit != afterExit || afterExit != 42 {
		t.Fatalf("native exits before=%d after=%d want 42", beforeExit, afterExit)
	}
}

func TestSCCPPassPropagatesDynamicZeroFactThroughSinglePredecessorTarget(t *testing.T) {
	prog := dynamicZeroTargetPathKnownLocalProgram()

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if !hasDecision(row.Decisions, "propagated_path_local_zero_target", "dynamic_zero_forward_single_predecessor_jump") {
		t.Fatalf("dynamic zero target fact decision missing: %#v", row.Decisions)
	}
	if !hasDecision(row.Decisions, "folded_path_local_zero_branch", "path_local_condition") {
		t.Fatalf("path-known zero branch fold missing: %#v", row.Decisions)
	}
	if !hasDecision(row.Decisions, "pruned_unreachable_fallthrough", "constant_branch_reachability") {
		t.Fatalf("path-known zero branch prune missing: %#v", row.Decisions)
	}
	for _, forbidden := range []string{
		"load_local local:0\n  jmp_if_zero label:2",
		"const_i32 99",
	} {
		if strings.Contains(row.AfterDump, forbidden) {
			t.Fatalf("after dump still contains %q:\n%s", forbidden, row.AfterDump)
		}
	}
	for _, want := range []string{
		"load_local local:0\n  jmp_if_zero label:1",
		"jmp label:2",
		"label:2",
		"const_i32 7",
	} {
		if !strings.Contains(row.AfterDump, want) {
			t.Fatalf("after dump missing %q:\n%s", want, row.AfterDump)
		}
	}
}

func TestSCCPPassUsesDynamicNonzeroFallthroughFactForRepeatedLocalBranch(t *testing.T) {
	prog := dynamicNonzeroFallthroughPathKnownLocalProgram()

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if !hasDecision(row.Decisions, "derived_path_local_nonzero_fallthrough", "dynamic_branch_fallthrough") {
		t.Fatalf("dynamic nonzero fallthrough fact decision missing: %#v", row.Decisions)
	}
	if !hasDecision(row.Decisions, "folded_path_local_nonzero_fallthrough", "path_local_condition") {
		t.Fatalf("path-known nonzero branch fold missing: %#v", row.Decisions)
	}
	if got := strings.Count(row.AfterDump, "jmp_if_zero"); got != 1 {
		t.Fatalf("after dump jmp_if_zero count = %d, want only the original dynamic branch:\n%s", got, row.AfterDump)
	}
	for _, forbidden := range []string{
		"load_local local:0\n  jmp_if_zero label:2",
	} {
		if strings.Contains(row.AfterDump, forbidden) {
			t.Fatalf("after dump still contains %q:\n%s", forbidden, row.AfterDump)
		}
	}
	for _, want := range []string{
		"load_local local:0\n  jmp_if_zero label:1",
		"const_i32 42",
		"label:1",
	} {
		if !strings.Contains(row.AfterDump, want) {
			t.Fatalf("after dump missing %q:\n%s", want, row.AfterDump)
		}
	}
}

func TestSCCPPassDoesNotPropagateDynamicZeroFactThroughFallthroughTarget(t *testing.T) {
	prog := dynamicZeroFallthroughTargetPathKnownLocalProgram()
	before := FormatProgram(prog)

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if hasDecision(row.Decisions, "folded_path_local_zero_branch", "path_local_condition") {
		t.Fatalf("fallthrough-target branch was folded with path-sensitive ambiguity: %#v", row.Decisions)
	}
	if !strings.Contains(row.AfterDump, "load_local local:0\n  jmp_if_zero label:2") {
		t.Fatalf("fallthrough-target branch changed despite ambiguity:\n%s", row.AfterDump)
	}
	if row.AfterDump != before {
		t.Fatalf("fallthrough-target function changed unexpectedly:\nbefore:\n%s\nafter:\n%s", before, row.AfterDump)
	}
}

func TestSCCPPassDerivesEqZeroComparisonPathFacts(t *testing.T) {
	tests := []struct {
		name           string
		prog           *ir.IRProgram
		wantDecision   string
		wantReason     string
		wantFoldAction string
		forbidden      []string
		required       []string
	}{
		{
			name:           "fallthrough_zero",
			prog:           dynamicEqZeroFallthroughPathKnownLocalProgram(),
			wantDecision:   "derived_comparison_path_local_zero_fallthrough",
			wantReason:     "eq_zero_true_fallthrough",
			wantFoldAction: "folded_path_local_zero_branch",
			forbidden:      []string{"load_local local:0\n  jmp_if_zero label:2", "const_i32 42"},
			required:       []string{"load_local local:0\n  const_i32 0\n  cmp_eq_i32\n  jmp_if_zero label:1", "jmp label:2", "const_i32 7"},
		},
		{
			name:           "target_nonzero",
			prog:           dynamicEqZeroTargetNonzeroPathKnownLocalProgram(),
			wantDecision:   "propagated_comparison_path_local_nonzero_target",
			wantReason:     "eq_zero_false_forward_single_predecessor_jump",
			wantFoldAction: "folded_path_local_nonzero_fallthrough",
			forbidden:      []string{"load_local local:0\n  jmp_if_zero label:2"},
			required:       []string{"load_local local:0\n  const_i32 0\n  cmp_eq_i32\n  jmp_if_zero label:1", "const_i32 42"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report, err := NewManager().Run(tc.prog, SCCPPass())
			if err != nil {
				t.Fatalf("Run SCCPPass: %v", err)
			}
			row := report.Passes[0]
			if !hasDecision(row.Decisions, "not_folded", "dynamic_condition") {
				t.Fatalf("dynamic comparison branch decision missing: %#v", row.Decisions)
			}
			if !hasDecision(row.Decisions, tc.wantDecision, tc.wantReason) {
				t.Fatalf("comparison path fact decision missing %s/%s: %#v", tc.wantDecision, tc.wantReason, row.Decisions)
			}
			if !hasDecision(row.Decisions, tc.wantFoldAction, "path_local_condition") {
				t.Fatalf("path-known branch fold missing %s: %#v", tc.wantFoldAction, row.Decisions)
			}
			for _, forbidden := range tc.forbidden {
				if strings.Contains(row.AfterDump, forbidden) {
					t.Fatalf("after dump still contains %q:\n%s", forbidden, row.AfterDump)
				}
			}
			for _, want := range tc.required {
				if !strings.Contains(row.AfterDump, want) {
					t.Fatalf("after dump missing %q:\n%s", want, row.AfterDump)
				}
			}
		})
	}
}

func TestSCCPPassDerivesNeZeroComparisonPathFacts(t *testing.T) {
	tests := []struct {
		name           string
		prog           *ir.IRProgram
		wantDecision   string
		wantReason     string
		wantFoldAction string
		forbidden      []string
		required       []string
	}{
		{
			name:           "fallthrough_nonzero",
			prog:           dynamicNeZeroFallthroughPathKnownLocalProgram(),
			wantDecision:   "derived_comparison_path_local_nonzero_fallthrough",
			wantReason:     "ne_zero_true_fallthrough",
			wantFoldAction: "folded_path_local_nonzero_fallthrough",
			forbidden:      []string{"load_local local:0\n  jmp_if_zero label:2"},
			required:       []string{"load_local local:0\n  const_i32 0\n  cmp_ne_i32\n  jmp_if_zero label:1", "const_i32 42"},
		},
		{
			name:           "target_zero",
			prog:           dynamicNeZeroTargetZeroPathKnownLocalProgram(),
			wantDecision:   "propagated_comparison_path_local_zero_target",
			wantReason:     "ne_zero_false_forward_single_predecessor_jump",
			wantFoldAction: "folded_path_local_zero_branch",
			forbidden:      []string{"load_local local:0\n  jmp_if_zero label:2", "const_i32 99"},
			required:       []string{"load_local local:0\n  const_i32 0\n  cmp_ne_i32\n  jmp_if_zero label:1", "jmp label:2", "const_i32 7"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report, err := NewManager().Run(tc.prog, SCCPPass())
			if err != nil {
				t.Fatalf("Run SCCPPass: %v", err)
			}
			row := report.Passes[0]
			if !hasDecision(row.Decisions, "not_folded", "dynamic_condition") {
				t.Fatalf("dynamic comparison branch decision missing: %#v", row.Decisions)
			}
			if !hasDecision(row.Decisions, tc.wantDecision, tc.wantReason) {
				t.Fatalf("comparison path fact decision missing %s/%s: %#v", tc.wantDecision, tc.wantReason, row.Decisions)
			}
			if !hasDecision(row.Decisions, tc.wantFoldAction, "path_local_condition") {
				t.Fatalf("path-known branch fold missing %s: %#v", tc.wantFoldAction, row.Decisions)
			}
			for _, forbidden := range tc.forbidden {
				if strings.Contains(row.AfterDump, forbidden) {
					t.Fatalf("after dump still contains %q:\n%s", forbidden, row.AfterDump)
				}
			}
			for _, want := range tc.required {
				if !strings.Contains(row.AfterDump, want) {
					t.Fatalf("after dump missing %q:\n%s", want, row.AfterDump)
				}
			}
		})
	}
}

func TestSCCPPassDoesNotDeriveComparisonTargetFactThroughFallthroughTarget(t *testing.T) {
	prog := dynamicComparisonFallthroughTargetPathKnownLocalProgram()
	before := FormatProgram(prog)

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if hasDecision(row.Decisions, "propagated_comparison_path_local_zero_target", "ne_zero_false_single_predecessor_label") {
		t.Fatalf("fallthrough-target comparison fact propagated with ambiguity: %#v", row.Decisions)
	}
	if hasDecision(row.Decisions, "folded_path_local_zero_branch", "path_local_condition") {
		t.Fatalf("fallthrough-target branch was folded with comparison ambiguity: %#v", row.Decisions)
	}
	if !strings.Contains(row.AfterDump, "load_local local:0\n  jmp_if_zero label:2") {
		t.Fatalf("path branch unexpectedly changed:\n%s", row.AfterDump)
	}
	if before != FormatProgram(prog) {
		t.Fatalf("ambiguous comparison target program changed:\nbefore:\n%s\nafter:\n%s", before, FormatProgram(prog))
	}
}

func TestSCCPPassFoldsKnownLocalConstantExpressionBranch(t *testing.T) {
	tests := []struct {
		name       string
		localValue int32
		compareImm int32
		wantAction string
		wantPrune  bool
		wantExit   int
		forbidden  []string
		required   []string
	}{
		{
			name:       "expression_zero_takes_branch",
			localValue: 10,
			compareImm: 5,
			wantAction: "folded_const_expr_zero_branch",
			wantPrune:  true,
			wantExit:   7,
			forbidden:  []string{"load_local local:0\n  const_i32 5\n  cmp_lt_i32\n  jmp_if_zero label:1", "const_i32 99"},
			required:   []string{"store_local local:0", "jmp label:1", "const_i32 7"},
		},
		{
			name:       "expression_nonzero_falls_through",
			localValue: 3,
			compareImm: 5,
			wantAction: "folded_const_expr_nonzero_fallthrough",
			wantExit:   42,
			forbidden:  []string{"load_local local:0\n  const_i32 5\n  cmp_lt_i32\n  jmp_if_zero label:1"},
			required:   []string{"store_local local:0", "const_i32 42", "label:1"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog := knownLocalLessThanBranchProgram(tc.localValue, tc.compareImm)
			before := cloneProgram(prog)

			report, err := NewManager().Run(prog, SCCPPass())
			if err != nil {
				t.Fatalf("Run SCCPPass: %v", err)
			}
			row := report.Passes[0]
			if !hasDecision(row.Decisions, tc.wantAction, "constant_expression_condition") {
				t.Fatalf("expression branch decision missing %s: %#v", tc.wantAction, row.Decisions)
			}
			if tc.wantPrune && !hasDecision(row.Decisions, "pruned_unreachable_fallthrough", "constant_branch_reachability") {
				t.Fatalf("prune decision missing: %#v", row.Decisions)
			}
			for _, forbidden := range tc.forbidden {
				if strings.Contains(row.AfterDump, forbidden) {
					t.Fatalf("after dump still contains %q:\n%s", forbidden, row.AfterDump)
				}
			}
			for _, want := range tc.required {
				if !strings.Contains(row.AfterDump, want) {
					t.Fatalf("after dump missing %q:\n%s", want, row.AfterDump)
				}
			}
			beforeExit := runOptLinuxX64(t, before.Funcs, tc.name+"-before-sccp")
			afterExit := runOptLinuxX64(t, prog.Funcs, tc.name+"-after-sccp")
			if beforeExit != afterExit || afterExit != tc.wantExit {
				t.Fatalf("native exits before=%d after=%d want %d", beforeExit, afterExit, tc.wantExit)
			}
		})
	}
}

func TestSCCPPassFoldsSafeUnaryNegExpressionBranch(t *testing.T) {
	tests := []struct {
		name       string
		imm        int32
		wantAction string
		wantPrune  bool
		wantExit   int
		forbidden  []string
		required   []string
	}{
		{
			name:       "zero_takes_branch",
			imm:        0,
			wantAction: "folded_const_unary_expr_zero_branch",
			wantPrune:  true,
			wantExit:   7,
			forbidden:  []string{"const_i32 0\n  neg_i32\n  jmp_if_zero label:1", "const_i32 42"},
			required:   []string{"jmp label:1", "const_i32 7"},
		},
		{
			name:       "nonzero_falls_through",
			imm:        -5,
			wantAction: "folded_const_unary_expr_nonzero_fallthrough",
			wantExit:   42,
			forbidden:  []string{"const_i32 -5\n  neg_i32\n  jmp_if_zero label:1"},
			required:   []string{"const_i32 42", "label:1"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog := unaryNegBranchProgram(tc.imm)
			before := cloneProgram(prog)

			report, err := NewManager().Run(prog, SCCPPass())
			if err != nil {
				t.Fatalf("Run SCCPPass: %v", err)
			}
			row := report.Passes[0]
			if !hasDecision(row.Decisions, tc.wantAction, "constant_unary_expression_condition") {
				t.Fatalf("unary expression branch decision missing %s: %#v", tc.wantAction, row.Decisions)
			}
			if tc.wantPrune && !hasDecision(row.Decisions, "pruned_unreachable_fallthrough", "constant_branch_reachability") {
				t.Fatalf("prune decision missing: %#v", row.Decisions)
			}
			for _, forbidden := range tc.forbidden {
				if strings.Contains(row.AfterDump, forbidden) {
					t.Fatalf("after dump still contains %q:\n%s", forbidden, row.AfterDump)
				}
			}
			for _, want := range tc.required {
				if !strings.Contains(row.AfterDump, want) {
					t.Fatalf("after dump missing %q:\n%s", want, row.AfterDump)
				}
			}
			beforeExit := runOptLinuxX64(t, before.Funcs, "before-safe-unary-neg-"+tc.name+"-sccp")
			afterExit := runOptLinuxX64(t, prog.Funcs, "after-safe-unary-neg-"+tc.name+"-sccp")
			if beforeExit != afterExit || afterExit != tc.wantExit {
				t.Fatalf("native exits before=%d after=%d want %d", beforeExit, afterExit, tc.wantExit)
			}
		})
	}
}

func TestSCCPPassDoesNotFoldUnsafeUnaryNegExpressionBranch(t *testing.T) {
	prog := unaryNegBranchProgram(-2147483648)
	before := FormatProgram(prog)

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if hasDecision(row.Decisions, "folded_const_unary_expr_zero_branch", "constant_unary_expression_condition") ||
		hasDecision(row.Decisions, "folded_const_unary_expr_nonzero_fallthrough", "constant_unary_expression_condition") {
		t.Fatalf("unsafe unary neg expression was folded: %#v", row.Decisions)
	}
	if row.AfterDump != before {
		t.Fatalf("unsafe unary neg expression changed unexpectedly:\nbefore:\n%s\nafter:\n%s", before, row.AfterDump)
	}
}

func TestSCCPPassFoldsStoredSafeUnaryNegExpressionBranch(t *testing.T) {
	prog := storedUnaryNegBranchProgram()
	before := cloneProgram(prog)

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if !hasDecision(row.Decisions, "folded_known_local_zero_branch", "constant_local_condition") {
		t.Fatalf("stored unary neg known-local zero decision missing: %#v", row.Decisions)
	}
	if !hasDecision(row.Decisions, "pruned_unreachable_fallthrough", "constant_branch_reachability") {
		t.Fatalf("prune decision missing after stored unary neg fold: %#v", row.Decisions)
	}
	for _, forbidden := range []string{"load_local local:0\n  jmp_if_zero label:1", "const_i32 42"} {
		if strings.Contains(row.AfterDump, forbidden) {
			t.Fatalf("after dump still contains %q:\n%s", forbidden, row.AfterDump)
		}
	}
	for _, want := range []string{"neg_i32", "store_local local:0", "jmp label:1", "const_i32 7"} {
		if !strings.Contains(row.AfterDump, want) {
			t.Fatalf("after dump missing %q:\n%s", want, row.AfterDump)
		}
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "before-stored-safe-unary-neg-sccp")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-stored-safe-unary-neg-sccp")
	if beforeExit != afterExit || afterExit != 7 {
		t.Fatalf("native exits before=%d after=%d want 7", beforeExit, afterExit)
	}
}

func TestSCCPPassFoldsSafeConstDenominatorDivModExpressionBranch(t *testing.T) {
	tests := []struct {
		name       string
		kind       ir.IRInstrKind
		op         string
		left       int32
		right      int32
		wantAction string
		wantPrune  bool
		wantExit   int
		forbidden  []string
		required   []string
	}{
		{
			name:       "division_nonzero_falls_through",
			kind:       ir.IRDivI32,
			op:         "div_i32",
			left:       20,
			right:      5,
			wantAction: "folded_const_expr_nonzero_fallthrough",
			wantExit:   42,
			forbidden:  []string{"const_i32 20\n  const_i32 5\n  div_i32\n  jmp_if_zero label:1"},
			required:   []string{"const_i32 42", "label:1"},
		},
		{
			name:       "modulo_zero_takes_branch",
			kind:       ir.IRModI32,
			op:         "mod_i32",
			left:       20,
			right:      5,
			wantAction: "folded_const_expr_zero_branch",
			wantPrune:  true,
			wantExit:   7,
			forbidden:  []string{"const_i32 20\n  const_i32 5\n  mod_i32\n  jmp_if_zero label:1", "const_i32 99"},
			required:   []string{"jmp label:1", "const_i32 7"},
		},
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
						{Kind: ir.IRConstI32, Imm: tc.left},
						{Kind: ir.IRConstI32, Imm: tc.right},
						{Kind: tc.kind},
						{Kind: ir.IRJmpIfZero, Label: 1},
						{Kind: ir.IRConstI32, Imm: 42},
						{Kind: ir.IRReturn},
						{Kind: ir.IRLabel, Label: 1},
						{Kind: ir.IRConstI32, Imm: 7},
						{Kind: ir.IRReturn},
					},
				}},
			}
			before := cloneProgram(prog)

			report, err := NewManager().Run(prog, SCCPPass())
			if err != nil {
				t.Fatalf("Run SCCPPass: %v", err)
			}
			row := report.Passes[0]
			if !hasDecision(row.Decisions, tc.wantAction, "constant_expression_condition") {
				t.Fatalf("safe %s branch decision missing %s: %#v", tc.op, tc.wantAction, row.Decisions)
			}
			if tc.wantPrune && !hasDecision(row.Decisions, "pruned_unreachable_fallthrough", "constant_branch_reachability") {
				t.Fatalf("prune decision missing: %#v", row.Decisions)
			}
			for _, forbidden := range tc.forbidden {
				if strings.Contains(row.AfterDump, forbidden) {
					t.Fatalf("after dump still contains %q:\n%s", forbidden, row.AfterDump)
				}
			}
			for _, want := range tc.required {
				if !strings.Contains(row.AfterDump, want) {
					t.Fatalf("after dump missing %q:\n%s", want, row.AfterDump)
				}
			}
			beforeExit := runOptLinuxX64(t, before.Funcs, tc.name+"-before-sccp")
			afterExit := runOptLinuxX64(t, prog.Funcs, tc.name+"-after-sccp")
			if beforeExit != afterExit || afterExit != tc.wantExit {
				t.Fatalf("native exits before=%d after=%d want %d", beforeExit, afterExit, tc.wantExit)
			}
		})
	}
}

func TestSCCPPassDoesNotFoldUnsafeConstDenominatorDivModExpressionBranch(t *testing.T) {
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
						{Kind: ir.IRJmpIfZero, Label: 1},
						{Kind: ir.IRConstI32, Imm: 42},
						{Kind: ir.IRReturn},
						{Kind: ir.IRLabel, Label: 1},
						{Kind: ir.IRConstI32, Imm: 7},
						{Kind: ir.IRReturn},
					},
				}},
			}
			before := FormatProgram(prog)

			report, err := NewManager().Run(prog, SCCPPass())
			if err != nil {
				t.Fatalf("Run SCCPPass: %v", err)
			}
			row := report.Passes[0]
			if row.AfterDump != before {
				t.Fatalf("unsafe div/mod branch changed unexpectedly:\nbefore:\n%s\nafter:\n%s", before, row.AfterDump)
			}
			if hasDecision(row.Decisions, "folded_const_expr_zero_branch", "constant_expression_condition") ||
				hasDecision(row.Decisions, "folded_const_expr_nonzero_fallthrough", "constant_expression_condition") {
				t.Fatalf("unsafe %s branch was folded: %#v", tc.op, row.Decisions)
			}
			if !hasDecision(row.Decisions, "not_folded", "dynamic_condition") {
				t.Fatalf("dynamic branch decision missing for unsafe %s: %#v", tc.op, row.Decisions)
			}
		})
	}
}

func TestSCCPPassFoldsStoredConstantExpressionBranch(t *testing.T) {
	prog := storedConstantExpressionBranchProgram()
	before := cloneProgram(prog)

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if !hasDecision(row.Decisions, "folded_known_local_zero_branch", "constant_local_condition") {
		t.Fatalf("stored-expression known-local zero decision missing: %#v", row.Decisions)
	}
	if !hasDecision(row.Decisions, "pruned_unreachable_fallthrough", "constant_branch_reachability") {
		t.Fatalf("prune decision missing after stored-expression fold: %#v", row.Decisions)
	}
	for _, forbidden := range []string{"load_local local:1\n  jmp_if_zero label:1", "const_i32 42"} {
		if strings.Contains(row.AfterDump, forbidden) {
			t.Fatalf("after dump still contains %q:\n%s", forbidden, row.AfterDump)
		}
	}
	for _, want := range []string{"load_local local:0", "const_i32 3", "sub_i32", "store_local local:1", "jmp label:1", "const_i32 7"} {
		if !strings.Contains(row.AfterDump, want) {
			t.Fatalf("after dump missing %q:\n%s", want, row.AfterDump)
		}
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "stored-expression-before-sccp")
	afterExit := runOptLinuxX64(t, prog.Funcs, "stored-expression-after-sccp")
	if beforeExit != afterExit || afterExit != 7 {
		t.Fatalf("native exits before=%d after=%d want 7", beforeExit, afterExit)
	}
}

func TestSCCPPassFoldsStoredSafeConstDenominatorDivModExpressionBranch(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  2,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 20},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRModI32},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
	before := cloneProgram(prog)

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if !hasDecision(row.Decisions, "folded_known_local_zero_branch", "constant_local_condition") {
		t.Fatalf("stored safe div/mod known-local zero decision missing: %#v", row.Decisions)
	}
	if !hasDecision(row.Decisions, "pruned_unreachable_fallthrough", "constant_branch_reachability") {
		t.Fatalf("prune decision missing after stored safe div/mod fold: %#v", row.Decisions)
	}
	for _, forbidden := range []string{"load_local local:1\n  jmp_if_zero label:1", "const_i32 42"} {
		if strings.Contains(row.AfterDump, forbidden) {
			t.Fatalf("after dump still contains %q:\n%s", forbidden, row.AfterDump)
		}
	}
	for _, want := range []string{"load_local local:0", "const_i32 5", "mod_i32", "store_local local:1", "jmp label:1", "const_i32 7"} {
		if !strings.Contains(row.AfterDump, want) {
			t.Fatalf("after dump missing %q:\n%s", want, row.AfterDump)
		}
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "stored-safe-divmod-before-sccp")
	afterExit := runOptLinuxX64(t, prog.Funcs, "stored-safe-divmod-after-sccp")
	if beforeExit != afterExit || afterExit != 7 {
		t.Fatalf("native exits before=%d after=%d want 7", beforeExit, afterExit)
	}
}

func TestSCCPPassDoesNotFoldStoredDynamicExpressionBranch(t *testing.T) {
	prog := storedDynamicExpressionBranchProgram()
	before := FormatProgram(prog)

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if hasDecision(row.Decisions, "folded_known_local_zero_branch", "constant_local_condition") {
		t.Fatalf("dynamic stored-expression branch was folded: %#v", row.Decisions)
	}
	if row.AfterDump != before {
		t.Fatalf("dynamic stored-expression function changed unexpectedly:\nbefore:\n%s\nafter:\n%s", before, row.AfterDump)
	}
	if !hasDecision(row.Decisions, "not_folded", "dynamic_condition") {
		t.Fatalf("dynamic branch decision missing: %#v", row.Decisions)
	}
}

func TestSCCPPassDoesNotFoldExpressionAcrossLabel(t *testing.T) {
	prog := knownLocalLessThanBranchProgram(10, 5)
	prog.Funcs[0].Instrs = append(
		prog.Funcs[0].Instrs[:2],
		append([]ir.IRInstr{{Kind: ir.IRLabel, Label: 9}}, prog.Funcs[0].Instrs[2:]...)...,
	)

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if hasDecision(row.Decisions, "folded_const_expr_zero_branch", "constant_expression_condition") {
		t.Fatalf("constant expression crossed a label: %#v", row.Decisions)
	}
	if !strings.Contains(row.AfterDump, "cmp_lt_i32") || !strings.Contains(row.AfterDump, "jmp_if_zero label:1") {
		t.Fatalf("expression branch changed despite label boundary:\n%s", row.AfterDump)
	}
}
