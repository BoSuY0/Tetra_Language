package plir_test

import (
	"strings"
	"testing"

	. "tetra_language/compiler/internal/plir"
)

func TestP41FromCheckedProgramRecordsMatrixAffineConstBLoadProof(t *testing.T) {
	checked := checkedProgram(t, matrixAffineBLoadPLIRProgram(
		"var b: []i32 = core.make_i32(9)",
		"row < 3",
		"col < 3",
		"k < 3",
		"k * 3 + col",
		"row = row + 1",
		"k = k + 1",
		"col = col + 1",
		"",
	))
	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}

	fn := findFunction(t, prog, "main")
	aTerm, ok := affineProofTermFor(fn, "a", "index_load")
	if !ok || !strings.HasPrefix(aTerm.ID, "proof:affine-const:row_k:a:") {
		t.Fatalf("P40 a load proof should remain intact, got %+v\n%s", aTerm, FormatText(prog))
	}
	cTerm, ok := affineProofTermFor(fn, "c", "index_store")
	if !ok || !strings.HasPrefix(cTerm.ID, "proof:affine-const:row_col:c:") {
		t.Fatalf("P38 c store proof should remain intact, got %+v\n%s", cTerm, FormatText(prog))
	}
	bTerms := affineProofTermsForBase(fn, "b")
	if len(bTerms) != 1 {
		t.Fatalf(
			"want exactly one affine proof term for base b, got %#v\n%s",
			bTerms,
			FormatText(prog),
		)
	}
	bTerm := bTerms[0]
	if bTerm.ID == aTerm.ID || bTerm.ID == cTerm.ID {
		t.Fatalf(
			"b proof reused another base proof id: b=%q a=%q c=%q",
			bTerm.ID,
			aTerm.ID,
			cTerm.ID,
		)
	}
	if !strings.HasPrefix(bTerm.ID, "proof:affine-const:k_col:b:") ||
		bTerm.IndexValueID != "local:k * 3 + col" ||
		bTerm.Operation != "index_load" ||
		bTerm.Range != "k * 3 + col in [0, b.len)" ||
		!containsString(bTerm.FactsUsed, "affine_const_extent") {
		t.Fatalf("b affine load proof term = %+v", bTerm)
	}

	guard, ok := proofGuardForID(fn, bTerm.ID)
	if !ok {
		t.Fatalf("missing b affine load proof guard for %q: %#v", bTerm.ID, fn.ProofGuards)
	}
	if guard.Kind != "range" ||
		!strings.Contains(guard.Condition, "k < 3") ||
		!strings.Contains(guard.Condition, "col < 3") ||
		!strings.Contains(guard.Condition, "b.len == 9") ||
		!strings.Contains(guard.Condition, "k * 3 + col") {
		t.Fatalf("b affine proof guard = %+v", guard)
	}
	use, ok := proofUseForID(fn, bTerm.ID)
	if !ok {
		t.Fatalf("missing b affine load proof use for %q: %#v", bTerm.ID, fn.ProofUses)
	}
	if !Dominates(fn, guard.Block, use.Block) {
		t.Fatalf(
			"b affine guard block %s should dominate use block %s in %+v",
			guard.Block,
			use.Block,
			fn.Dominators,
		)
	}
	op, ok := operationForID(fn, use.OpID)
	if !ok || op.Kind != OpIndexLoad || len(op.Inputs) < 2 || op.Inputs[0] != "b" ||
		op.Inputs[1] != "k * 3 + col" {
		t.Fatalf("b affine proof use should point at b index_load, use=%+v op=%+v", use, op)
	}
	rangeFact, ok := rangeFactForProofID(fn, bTerm.ID)
	if !ok {
		t.Fatalf("missing b affine load range fact for %q: %#v", bTerm.ID, fn.RangeFacts)
	}
	if rangeFact.Value != "local:k * 3 + col" ||
		rangeFact.Lower != (Bound{Kind: BoundConst, Const: 0}) ||
		rangeFact.Upper != (Bound{Kind: BoundSymbol, Symbol: "b.len"}) ||
		!rangeFact.InclusiveLower ||
		rangeFact.InclusiveUpper ||
		!containsString(rangeFact.Derivation, "affine_const_extent") {
		t.Fatalf("b affine load range fact = %+v", rangeFact)
	}
}

func TestP41FromCheckedProgramRejectsInvalidMatrixAffineConstBLoadProofs(t *testing.T) {
	tests := []struct {
		name       string
		bDecl      string
		colGuard   string
		kGuard     string
		bLoadIndex string
		kInc       string
		colInc     string
		beforeLoad string
	}{
		{
			name:       "wrong_stride",
			bDecl:      "var b: []i32 = core.make_i32(9)",
			colGuard:   "col < 3",
			kGuard:     "k < 3",
			bLoadIndex: "k * 4 + col",
			kInc:       "k = k + 1",
			colInc:     "col = col + 1",
		},
		{
			name:       "mutable_allocation_length",
			bDecl:      "var n: Int = 9\n    var b: []i32 = core.make_i32(n)",
			colGuard:   "col < 3",
			kGuard:     "k < 3",
			bLoadIndex: "k * 3 + col",
			kInc:       "k = k + 1",
			colInc:     "col = col + 1",
		},
		{
			name:       "non_unit_k_increment",
			bDecl:      "var b: []i32 = core.make_i32(9)",
			colGuard:   "col < 3",
			kGuard:     "k < 3",
			bLoadIndex: "k * 3 + col",
			kInc:       "k = k + 2",
			colInc:     "col = col + 1",
		},
		{
			name:       "non_unit_col_increment",
			bDecl:      "var b: []i32 = core.make_i32(9)",
			colGuard:   "col < 3",
			kGuard:     "k < 3",
			bLoadIndex: "k * 3 + col",
			kInc:       "k = k + 1",
			colInc:     "col = col + 2",
		},
		{
			name:       "non_strict_col_guard",
			bDecl:      "var b: []i32 = core.make_i32(9)",
			colGuard:   "col <= 2",
			kGuard:     "k < 3",
			bLoadIndex: "k * 3 + col",
			kInc:       "k = k + 1",
			colInc:     "col = col + 1",
		},
		{
			name:       "base_reassignment_before_load",
			bDecl:      "var b: []i32 = core.make_i32(9)",
			colGuard:   "col < 3",
			kGuard:     "k < 3",
			bLoadIndex: "k * 3 + col",
			kInc:       "k = k + 1",
			colInc:     "col = col + 1",
			beforeLoad: "b = core.make_i32(9)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checked := checkedProgram(
				t,
				matrixAffineBLoadPLIRProgram(
					tt.bDecl,
					"row < 3",
					tt.colGuard,
					tt.kGuard,
					tt.bLoadIndex,
					"row = row + 1",
					tt.kInc,
					tt.colInc,
					tt.beforeLoad,
				),
			)
			prog, err := FromCheckedProgram(checked)
			if err != nil {
				t.Fatalf("FromCheckedProgram: %v", err)
			}
			if err := VerifyProgram(prog); err != nil {
				t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
			}
			fn := findFunction(t, prog, "main")
			if got := affineProofTermsForBase(fn, "b"); len(got) != 0 {
				t.Fatalf(
					"%s: invalid b load shape received affine load proof terms: %#v\n%s",
					tt.name,
					got,
					FormatText(prog),
				)
			}
			for _, guard := range fn.ProofGuards {
				if strings.HasPrefix(guard.ID, "proof:affine-const:k_col:b:") {
					t.Fatalf(
						"%s: invalid b load shape received affine proof guard: %+v\n%s",
						tt.name,
						guard,
						FormatText(prog),
					)
				}
			}
			for _, use := range fn.ProofUses {
				if strings.HasPrefix(use.ProofID, "proof:affine-const:k_col:b:") {
					t.Fatalf(
						"%s: invalid b load shape received affine proof use: %+v\n%s",
						tt.name,
						use,
						FormatText(prog),
					)
				}
			}
			for _, candidate := range fn.ProofTerms {
				if strings.HasPrefix(candidate.ID, "proof:affine-const:") &&
					!strings.Contains(candidate.ID, ":"+candidate.SubjectBaseID+":") {
					t.Fatalf("%s: affine proof id is not base-specific: %+v", tt.name, candidate)
				}
			}
		})
	}
}

func matrixAffineBLoadPLIRProgram(
	bDecl string,
	rowGuard string,
	colGuard string,
	kGuard string,
	bLoadIndex string,
	rowInc string,
	kInc string,
	colInc string,
	beforeLoad string,
) string {
	if beforeLoad != "" {
		beforeLoad = "\n                " + beforeLoad
	}
	return strings.NewReplacer(
		"$B_DECL", bDecl,
		"$ROW_GUARD", rowGuard,
		"$COL_GUARD", colGuard,
		"$K_GUARD", kGuard,
		"$B_LOAD_INDEX", bLoadIndex,
		"$ROW_INC", rowInc,
		"$K_INC", kInc,
		"$COL_INC", colInc,
		"$BEFORE_LOAD", beforeLoad,
	).Replace(`
func main() -> Int
uses alloc, mem:
    var a: []i32 = core.make_i32(9)
    $B_DECL
    var c: []i32 = core.make_i32(9)
    var row: Int = 0
    while $ROW_GUARD:
        var col: Int = 0
        while $COL_GUARD:
            var k: Int = 0
            var total: Int = 0
            while $K_GUARD:$BEFORE_LOAD
                total = total + a[row * 3 + k] * b[$B_LOAD_INDEX]
                $K_INC
            c[row * 3 + col] = total
            $COL_INC
        $ROW_INC
    return 0
`)
}
