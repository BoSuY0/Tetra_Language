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
	"tetra_language/compiler/internal/memoryfacts"
)

// ---- coverage_test.go ----

func TestCoreOptimizationCoverageAuditsP17PlanList(t *testing.T) {
	report := CoreOptimizationCoverage()
	if report.SchemaVersion != "tetra.optimizer.core_coverage.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	want := []CoreOptimizationID{
		CoreOptimizationConstantFolding,
		CoreOptimizationCopyPropagation,
		CoreOptimizationDCE,
		CoreOptimizationSCCP,
		CoreOptimizationCSEGvn,
		CoreOptimizationMem2Reg,
		CoreOptimizationSimpleInlining,
		CoreOptimizationLoopCanonicalization,
		CoreOptimizationLICM,
		CoreOptimizationAllocationSinking,
		CoreOptimizationScalarReplacement,
		CoreOptimizationBoundsCheckElimination,
	}
	if len(report.Rows) != len(want) {
		t.Fatalf("coverage rows = %d, want %d: %#v", len(report.Rows), len(want), report.Rows)
	}
	byID := map[CoreOptimizationID]CoreOptimizationCoverageRow{}
	for _, row := range report.Rows {
		byID[row.ID] = row
		if row.Name == "" || row.Status == "" || row.Evidence == "" || row.Boundary == "" {
			t.Fatalf("row missing required evidence: %#v", row)
		}
		if row.Status != CoreOptimizationNotYetCovered && row.PassName == "" {
			t.Fatalf("covered row missing pass/lowering owner: %#v", row)
		}
	}
	for _, id := range want {
		if _, ok := byID[id]; !ok {
			t.Fatalf("missing coverage row %s", id)
		}
	}
	if byID[CoreOptimizationConstantFolding].Status != CoreOptimizationImplementedNarrow ||
		byID[CoreOptimizationConstantFolding].PassName != "basic-scalar" {
		t.Fatalf(
			"constant folding row = %#v, want basic-scalar implemented_narrow",
			byID[CoreOptimizationConstantFolding],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationConstantFolding].Boundary,
		"safe const-denominator div_i32/mod_i32 constants",
	) {
		t.Fatalf(
			"constant folding boundary missing safe div/mod constants: %#v",
			byID[CoreOptimizationConstantFolding],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationConstantFolding].Boundary,
		"same-local comparison algebraic forms",
	) {
		t.Fatalf(
			"constant folding boundary missing same-local comparison algebra: %#v",
			byID[CoreOptimizationConstantFolding],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationConstantFolding].Boundary,
		"denominators 0 and -1 remain rejected",
	) {
		t.Fatalf(
			"constant folding boundary missing unsafe denominator rejection: %#v",
			byID[CoreOptimizationConstantFolding],
		)
	}
	if byID[CoreOptimizationCSEGvn].Status != CoreOptimizationImplementedNarrow ||
		byID[CoreOptimizationCSEGvn].PassName != "basic-scalar" {
		t.Fatalf(
			"CSE/GVN row = %#v, want basic-scalar implemented_narrow",
			byID[CoreOptimizationCSEGvn],
		)
	}
	if !strings.Contains(byID[CoreOptimizationCSEGvn].Boundary, "commutative add/mul/eq/ne") {
		t.Fatalf(
			"CSE/GVN boundary missing commutative local expression limit: %#v",
			byID[CoreOptimizationCSEGvn],
		)
	}
	if !strings.Contains(byID[CoreOptimizationCSEGvn].Boundary, "mirrored lt/gt/le/ge") {
		t.Fatalf(
			"CSE/GVN boundary missing mirrored ordered-comparison limit: %#v",
			byID[CoreOptimizationCSEGvn],
		)
	}
	if !strings.Contains(byID[CoreOptimizationCSEGvn].Boundary, "local-load/constant") {
		t.Fatalf(
			"CSE/GVN boundary missing local-constant expression limit: %#v",
			byID[CoreOptimizationCSEGvn],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationCSEGvn].Boundary,
		"safe const-denominator div_i32/mod_i32",
	) {
		t.Fatalf(
			"CSE/GVN boundary missing safe division/modulo expression limit: %#v",
			byID[CoreOptimizationCSEGvn],
		)
	}
	if !strings.Contains(byID[CoreOptimizationCSEGvn].Boundary, "unary local neg_i32") {
		t.Fatalf(
			"CSE/GVN boundary missing unary local neg expression limit: %#v",
			byID[CoreOptimizationCSEGvn],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationCSEGvn].Boundary,
		"safe known-local unary neg_i32 value expressions",
	) {
		t.Fatalf(
			"CSE/GVN boundary missing safe known-local unary value limit: %#v",
			byID[CoreOptimizationCSEGvn],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationCSEGvn].Boundary,
		"overflow-sensitive unary neg_i32 min-int",
	) {
		t.Fatalf(
			"CSE/GVN boundary missing unsafe known-local unary rejection: %#v",
			byID[CoreOptimizationCSEGvn],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationCSEGvn].Boundary,
		"safe known-local add_i32/sub_i32/mul_i32 value expressions",
	) {
		t.Fatalf(
			"CSE/GVN boundary missing safe known-local arithmetic value limit: %#v",
			byID[CoreOptimizationCSEGvn],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationCSEGvn].Boundary,
		"overflow-sensitive known-local arithmetic",
	) {
		t.Fatalf(
			"CSE/GVN boundary missing unsafe known-local arithmetic rejection: %#v",
			byID[CoreOptimizationCSEGvn],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationCSEGvn].Boundary,
		"safe known-local cmp_*_i32 value expressions",
	) {
		t.Fatalf(
			"CSE/GVN boundary missing safe known-local comparison value limit: %#v",
			byID[CoreOptimizationCSEGvn],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationCSEGvn].Boundary,
		"source-local mutations that change known values",
	) {
		t.Fatalf(
			"CSE/GVN boundary missing source mutation rejection: %#v",
			byID[CoreOptimizationCSEGvn],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationCSEGvn].Boundary,
		"safe known-local div_i32/mod_i32 value expressions",
	) {
		t.Fatalf(
			"CSE/GVN boundary missing safe known-local division/modulo value limit: %#v",
			byID[CoreOptimizationCSEGvn],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationCSEGvn].Boundary,
		"unsafe known-local division/modulo",
	) {
		t.Fatalf(
			"CSE/GVN boundary missing unsafe known-local division/modulo rejection: %#v",
			byID[CoreOptimizationCSEGvn],
		)
	}
	if byID[CoreOptimizationDCE].Status != CoreOptimizationImplementedNarrow ||
		byID[CoreOptimizationDCE].PassName != "basic-scalar" {
		t.Fatalf("DCE row = %#v, want basic-scalar implemented_narrow", byID[CoreOptimizationDCE])
	}
	if !strings.Contains(
		byID[CoreOptimizationDCE].Boundary,
		"non-trapping comparison-expression producers",
	) {
		t.Fatalf(
			"DCE boundary missing non-trapping comparison expression limit: %#v",
			byID[CoreOptimizationDCE],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationDCE].Boundary,
		"safe known-local unary neg_i32 producers",
	) {
		t.Fatalf(
			"DCE boundary missing safe unary neg producer limit: %#v",
			byID[CoreOptimizationDCE],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationDCE].Boundary,
		"overflow-sensitive unary neg_i32 min-int",
	) {
		t.Fatalf("DCE boundary missing unsafe unary neg rejection: %#v", byID[CoreOptimizationDCE])
	}
	if !strings.Contains(
		byID[CoreOptimizationDCE].Boundary,
		"safe known-local add_i32/sub_i32/mul_i32 producers",
	) {
		t.Fatalf(
			"DCE boundary missing safe known-local arithmetic producer limit: %#v",
			byID[CoreOptimizationDCE],
		)
	}
	if !strings.Contains(byID[CoreOptimizationDCE].Boundary, "overflow-sensitive arithmetic") {
		t.Fatalf("DCE boundary missing unsafe arithmetic rejection: %#v", byID[CoreOptimizationDCE])
	}
	if !strings.Contains(
		byID[CoreOptimizationDCE].Boundary,
		"safe const-denominator div_i32/mod_i32 producers",
	) {
		t.Fatalf(
			"DCE boundary missing safe division/modulo expression limit: %#v",
			byID[CoreOptimizationDCE],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationDCE].Boundary,
		"div_i32/mod_i32 denominators 0 and -1 are rejected",
	) {
		t.Fatalf(
			"DCE boundary missing unsafe denominator rejection: %#v",
			byID[CoreOptimizationDCE],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationDCE].Boundary,
		"safe known-local div_i32/mod_i32 producers",
	) {
		t.Fatalf(
			"DCE boundary missing safe known-local division/modulo producer limit: %#v",
			byID[CoreOptimizationDCE],
		)
	}
	if !strings.Contains(byID[CoreOptimizationDCE].Boundary, "unsafe division/modulo DCE") {
		t.Fatalf(
			"DCE boundary missing unsafe division/modulo DCE non-claim: %#v",
			byID[CoreOptimizationDCE],
		)
	}
	if byID[CoreOptimizationSCCP].Status != CoreOptimizationImplementedNarrow ||
		byID[CoreOptimizationSCCP].PassName != "sccp-constant-branch" {
		t.Fatalf(
			"SCCP row = %#v, want sccp-constant-branch implemented_narrow",
			byID[CoreOptimizationSCCP],
		)
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "known-local") {
		t.Fatalf("SCCP boundary missing known-local branch limit: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(
		byID[CoreOptimizationSCCP].Boundary,
		"constant binary-expression branch folding",
	) {
		t.Fatalf(
			"SCCP boundary missing constant expression branch limit: %#v",
			byID[CoreOptimizationSCCP],
		)
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "constant unary neg_i32") {
		t.Fatalf(
			"SCCP boundary missing unary neg expression branch limit: %#v",
			byID[CoreOptimizationSCCP],
		)
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "stored safe unary neg_i32") {
		t.Fatalf(
			"SCCP boundary missing stored unary neg fact limit: %#v",
			byID[CoreOptimizationSCCP],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationSCCP].Boundary,
		"overflow-sensitive unary neg_i32 min-int",
	) {
		t.Fatalf(
			"SCCP boundary missing unsafe unary neg rejection: %#v",
			byID[CoreOptimizationSCCP],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationSCCP].Boundary,
		"safe const-denominator div_i32/mod_i32",
	) {
		t.Fatalf(
			"SCCP boundary missing safe div/mod expression branch limit: %#v",
			byID[CoreOptimizationSCCP],
		)
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "denominators 0 and -1") {
		t.Fatalf(
			"SCCP boundary missing unsafe div/mod denominator rejection: %#v",
			byID[CoreOptimizationSCCP],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationSCCP].Boundary,
		"stored safe constant binary-expression facts",
	) {
		t.Fatalf(
			"SCCP boundary missing stored constant-expression fact limit: %#v",
			byID[CoreOptimizationSCCP],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationSCCP].Boundary,
		"single-predecessor label propagation",
	) {
		t.Fatalf(
			"SCCP boundary missing single-predecessor label propagation limit: %#v",
			byID[CoreOptimizationSCCP],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationSCCP].Boundary,
		"forward-terminated single-predecessor",
	) {
		t.Fatalf(
			"SCCP boundary missing forward single-predecessor propagation limit: %#v",
			byID[CoreOptimizationSCCP],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationSCCP].Boundary,
		"folded zero-branch target propagation",
	) {
		t.Fatalf(
			"SCCP boundary missing folded zero-branch target propagation limit: %#v",
			byID[CoreOptimizationSCCP],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationSCCP].Boundary,
		"folded nonzero-branch fallthrough propagation",
	) {
		t.Fatalf(
			"SCCP boundary missing folded nonzero fallthrough propagation limit: %#v",
			byID[CoreOptimizationSCCP],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationSCCP].Boundary,
		"dynamic load_local zero-target and nonzero-fallthrough path facts",
	) {
		t.Fatalf(
			"SCCP boundary missing dynamic branch path-fact limit: %#v",
			byID[CoreOptimizationSCCP],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationSCCP].Boundary,
		"dynamic zero-comparison eq/ne zero/nonzero path facts",
	) {
		t.Fatalf(
			"SCCP boundary missing dynamic zero-comparison path-fact limit: %#v",
			byID[CoreOptimizationSCCP],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationSCCP].Boundary,
		"immediate label with no explicit incoming branch/jump edges",
	) {
		t.Fatalf(
			"SCCP boundary missing folded nonzero fallthrough-only label limit: %#v",
			byID[CoreOptimizationSCCP],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationSCCP].Boundary,
		"folded nonzero fallthrough labels with explicit incoming edges",
	) {
		t.Fatalf(
			"SCCP boundary missing folded nonzero explicit-incoming rejection: %#v",
			byID[CoreOptimizationSCCP],
		)
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "multi-predecessor labels") {
		t.Fatalf(
			"SCCP boundary missing multi-predecessor label rejection: %#v",
			byID[CoreOptimizationSCCP],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationSCCP].Boundary,
		"fallthrough predecessors are rejected",
	) {
		t.Fatalf(
			"SCCP boundary missing fallthrough-predecessor rejection: %#v",
			byID[CoreOptimizationSCCP],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationSCCP].Boundary,
		"dynamic zero-target labels with fallthrough predecessors",
	) {
		t.Fatalf(
			"SCCP boundary missing dynamic zero-target fallthrough rejection: %#v",
			byID[CoreOptimizationSCCP],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationSCCP].Boundary,
		"dynamic comparison-target labels with fallthrough predecessors",
	) {
		t.Fatalf(
			"SCCP boundary missing dynamic comparison-target fallthrough rejection: %#v",
			byID[CoreOptimizationSCCP],
		)
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "dynamic stored expressions") {
		t.Fatalf(
			"SCCP boundary missing dynamic stored-expression rejection: %#v",
			byID[CoreOptimizationSCCP],
		)
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "arbitrary comparison reasoning") {
		t.Fatalf(
			"SCCP boundary missing arbitrary comparison reasoning non-claim: %#v",
			byID[CoreOptimizationSCCP],
		)
	}
	if byID[CoreOptimizationMem2Reg].Status != CoreOptimizationImplementedNarrow ||
		byID[CoreOptimizationMem2Reg].PassName != "mem2reg-single-assignment" {
		t.Fatalf(
			"mem2reg row = %#v, want mem2reg-single-assignment implemented_narrow",
			byID[CoreOptimizationMem2Reg],
		)
	}
	if !strings.Contains(byID[CoreOptimizationMem2Reg].Boundary, "stack-neutral separated") {
		t.Fatalf(
			"mem2reg boundary missing separated stack-neutral temp limit: %#v",
			byID[CoreOptimizationMem2Reg],
		)
	}
	if !strings.Contains(byID[CoreOptimizationMem2Reg].Boundary, "comparison-expression") {
		t.Fatalf(
			"mem2reg boundary missing comparison-expression producer limit: %#v",
			byID[CoreOptimizationMem2Reg],
		)
	}
	if !strings.Contains(byID[CoreOptimizationMem2Reg].Boundary, "safe const unary neg_i32") {
		t.Fatalf(
			"mem2reg boundary missing safe unary neg producer limit: %#v",
			byID[CoreOptimizationMem2Reg],
		)
	}
	if !strings.Contains(byID[CoreOptimizationMem2Reg].Boundary, "safe known-local unary neg_i32") {
		t.Fatalf(
			"mem2reg boundary missing safe known-local unary neg producer limit: %#v",
			byID[CoreOptimizationMem2Reg],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationMem2Reg].Boundary,
		"overflow-sensitive unary neg_i32 min-int",
	) {
		t.Fatalf(
			"mem2reg boundary missing unsafe unary neg rejection: %#v",
			byID[CoreOptimizationMem2Reg],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationMem2Reg].Boundary,
		"safe const add_i32/sub_i32/mul_i32 arithmetic",
	) {
		t.Fatalf(
			"mem2reg boundary missing safe const arithmetic producer limit: %#v",
			byID[CoreOptimizationMem2Reg],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationMem2Reg].Boundary,
		"safe known-local add_i32/sub_i32/mul_i32 arithmetic",
	) {
		t.Fatalf(
			"mem2reg boundary missing safe known-local arithmetic producer limit: %#v",
			byID[CoreOptimizationMem2Reg],
		)
	}
	if !strings.Contains(byID[CoreOptimizationMem2Reg].Boundary, "arithmetic overflow") {
		t.Fatalf(
			"mem2reg boundary missing unsafe arithmetic rejection: %#v",
			byID[CoreOptimizationMem2Reg],
		)
	}
	if !strings.Contains(byID[CoreOptimizationMem2Reg].Boundary, "source-local mutation") {
		t.Fatalf(
			"mem2reg boundary missing source-local mutation rejection: %#v",
			byID[CoreOptimizationMem2Reg],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationMem2Reg].Boundary,
		"safe const-denominator div_i32/mod_i32 producer",
	) {
		t.Fatalf(
			"mem2reg boundary missing safe div/mod producer limit: %#v",
			byID[CoreOptimizationMem2Reg],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationMem2Reg].Boundary,
		"safe known-local div_i32/mod_i32 producer",
	) {
		t.Fatalf(
			"mem2reg boundary missing safe known-local div/mod producer limit: %#v",
			byID[CoreOptimizationMem2Reg],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationMem2Reg].Boundary,
		"denominators 0 and -1 are rejected",
	) {
		t.Fatalf(
			"mem2reg boundary missing unsafe denominator rejection: %#v",
			byID[CoreOptimizationMem2Reg],
		)
	}
	if byID[CoreOptimizationLICM].Status != CoreOptimizationImplementedNarrow ||
		byID[CoreOptimizationLICM].PassName != "licm-pure-invariant" {
		t.Fatalf(
			"LICM row = %#v, want licm-pure-invariant implemented_narrow",
			byID[CoreOptimizationLICM],
		)
	}
	if !strings.Contains(byID[CoreOptimizationLICM].Boundary, "add/sub/mul arithmetic") {
		t.Fatalf(
			"LICM boundary missing pure invariant arithmetic limit: %#v",
			byID[CoreOptimizationLICM],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationLICM].Boundary,
		"known-local add_i32/sub_i32/mul_i32 left-or-right operand",
	) {
		t.Fatalf(
			"LICM boundary missing known-local arithmetic operand limit: %#v",
			byID[CoreOptimizationLICM],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationLICM].Boundary,
		"known-local cmp_*_i32 left-or-right operand",
	) {
		t.Fatalf(
			"LICM boundary missing known-local comparison operand limit: %#v",
			byID[CoreOptimizationLICM],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationLICM].Boundary,
		"safe const-denominator div_i32/mod_i32",
	) {
		t.Fatalf(
			"LICM boundary missing safe division/modulo limit: %#v",
			byID[CoreOptimizationLICM],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationLICM].Boundary,
		"safe known-local div_i32/mod_i32 denominator",
	) {
		t.Fatalf(
			"LICM boundary missing safe known-local division/modulo denominator limit: %#v",
			byID[CoreOptimizationLICM],
		)
	}
	if !strings.Contains(byID[CoreOptimizationLICM].Boundary, "denominators 0 and -1") {
		t.Fatalf(
			"LICM boundary missing unsafe denominator rejection: %#v",
			byID[CoreOptimizationLICM],
		)
	}
	if !strings.Contains(
		byID[CoreOptimizationLICM].Boundary,
		"loop-mutated operands are rejected",
	) {
		t.Fatalf(
			"LICM boundary missing loop-mutated operand rejection: %#v",
			byID[CoreOptimizationLICM],
		)
	}
}

func TestOptimizerCoreCoverageDocsRecordP17Closure(t *testing.T) {
	docs := []string{
		"../../../docs/audits/compiler/optimizer/optimizer-core-coverage-v1.md",
		"../../../reports/optimizer-core-coverage-v1/closure.md",
	}
	for _, path := range docs {
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			text := string(data)
			for _, stale := range []string{
				"Status: P17.1 progress",
				"P17.1 remains active",
				"This is not a full P17.1 completion claim",
				"Remaining P17.1 Work",
				"later P17.1 work",
			} {
				if strings.Contains(text, stale) {
					t.Fatalf("%s still contains stale P17.1 progress wording %q", path, stale)
				}
			}
			for _, want := range []string{
				"Status: P17.1 closed",
				"bounded evidence-backed P17.1 closure",
				"no C/Rust `-O1`/`-O2` performance parity claim",
			} {
				if !strings.Contains(text, want) {
					t.Fatalf("%s missing P17.1 closure wording %q", path, want)
				}
			}
		})
	}
}

func TestInliningSpecializationCoverageAuditsP17PlanList(t *testing.T) {
	report := InliningSpecializationCoverage()
	if report.SchemaVersion != "tetra.optimizer.inlining_specialization.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	want := []InliningSpecializationID{
		InliningSpecializationGenericFunctions,
		InliningSpecializationSmallPureFunctions,
		InliningSpecializationStaticProtocolConformanceCalls,
		InliningSpecializationExtensionCalls,
		InliningSpecializationEnumKnownCase,
		InliningSpecializationOptionalUnwrapProvenSome,
	}
	if len(report.Rows) != len(want) {
		t.Fatalf("coverage rows = %d, want %d: %#v", len(report.Rows), len(want), report.Rows)
	}
	byID := map[InliningSpecializationID]InliningSpecializationCoverageRow{}
	for _, row := range report.Rows {
		byID[row.ID] = row
		if row.Name == "" || row.Status == "" || row.Evidence == "" || row.Boundary == "" {
			t.Fatalf("row missing required evidence: %#v", row)
		}
		if row.Status == InliningSpecializationImplementedNarrow && row.PassName == "" {
			t.Fatalf("implemented row missing pass/lowering owner: %#v", row)
		}
	}
	for _, id := range want {
		if _, ok := byID[id]; !ok {
			t.Fatalf("missing coverage row %s", id)
		}
	}
	generic := byID[InliningSpecializationGenericFunctions]
	if generic.Status != InliningSpecializationImplementedNarrow ||
		generic.PassName != "inline-small-pure" {
		t.Fatalf("generic row = %#v, want inline-small-pure implemented_narrow", generic)
	}
	for _, want := range []string{
		"monomorphized generic identity",
		"generic wrapper",
		"small_pure_wrapper",
		"no runtime generic values",
	} {
		if !strings.Contains(generic.Boundary+" "+generic.Evidence, want) {
			t.Fatalf("generic row missing %q: %#v", want, generic)
		}
	}
	smallPure := byID[InliningSpecializationSmallPureFunctions]
	if smallPure.Status != InliningSpecializationImplementedNarrow ||
		smallPure.PassName != "inline-small-pure" {
		t.Fatalf("small-pure row = %#v, want inline-small-pure implemented_narrow", smallPure)
	}
	for _, want := range []string{
		"inlined",
		"not_inlined",
		"8",
		"proof-sensitive",
		"translation validation",
	} {
		if !strings.Contains(smallPure.Boundary+" "+smallPure.Evidence, want) {
			t.Fatalf("small-pure row missing %q: %#v", want, smallPure)
		}
	}
	enumKnown := byID[InliningSpecializationEnumKnownCase]
	if enumKnown.Status != InliningSpecializationImplementedNarrow ||
		enumKnown.PassName != "sccp-constant-branch" {
		t.Fatalf(
			"enum-known-case row = %#v, want sccp-constant-branch implemented_narrow",
			enumKnown,
		)
	}
	for _, want := range []string{
		"payload enum constructor",
		"known-case match",
		"constant_stack_store",
		"translation validation",
		"no broad enum specialization",
	} {
		if !strings.Contains(enumKnown.Boundary+" "+enumKnown.Evidence, want) {
			t.Fatalf("enum-known-case row missing %q: %#v", want, enumKnown)
		}
	}
	optionalSome := byID[InliningSpecializationOptionalUnwrapProvenSome]
	if optionalSome.Status != InliningSpecializationImplementedNarrow ||
		optionalSome.PassName != "sccp-constant-branch" {
		t.Fatalf(
			"optional-proven-some row = %#v, want sccp-constant-branch implemented_narrow",
			optionalSome,
		)
	}
	for _, want := range []string{
		"proven-some optional",
		"constant_stack_store",
		"translation validation",
		"no broad optional elimination",
	} {
		if !strings.Contains(optionalSome.Boundary+" "+optionalSome.Evidence, want) {
			t.Fatalf("optional-proven-some row missing %q: %#v", want, optionalSome)
		}
	}
	extension := byID[InliningSpecializationExtensionCalls]
	if extension.Status != InliningSpecializationImplementedNarrow ||
		extension.PassName != "inline-small-pure" {
		t.Fatalf("extension-call row = %#v, want inline-small-pure implemented_narrow", extension)
	}
	for _, want := range []string{
		"statically resolved extension method",
		"direct Stack IR function symbol",
		"translation validation",
		"no dynamic extension dispatch",
	} {
		if !strings.Contains(extension.Boundary+" "+extension.Evidence, want) {
			t.Fatalf("extension-call row missing %q: %#v", want, extension)
		}
	}
	staticProtocol := byID[InliningSpecializationStaticProtocolConformanceCalls]
	if staticProtocol.Status != InliningSpecializationImplementedNarrow ||
		staticProtocol.PassName != "inline-small-pure" {
		t.Fatalf(
			"static protocol/conformance row = %#v, want inline-small-pure implemented_narrow",
			staticProtocol,
		)
	}
	for _, want := range []string{
		"statically checked protocol impl",
		"known direct Stack IR function symbol",
		"translation validation",
		"no witness tables",
		"generic-bound requirement calls",
	} {
		if !strings.Contains(staticProtocol.Boundary+" "+staticProtocol.Evidence, want) {
			t.Fatalf("static protocol/conformance row missing %q: %#v", want, staticProtocol)
		}
	}
}

func TestP21SpecializationMachineCodeCoverageCoversPlanTargetsAndRejectsFakeClaims(t *testing.T) {
	report, err := SpecializationMachineCodeCoverage()
	if err != nil {
		t.Fatalf("SpecializationMachineCodeCoverage: %v", err)
	}
	if err := ValidateSpecializationMachineCodeCoverage(report); err != nil {
		t.Fatalf("ValidateSpecializationMachineCodeCoverage: %v", err)
	}
	if report.SchemaVersion != "tetra.optimizer.specialization_machine_code.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	if report.Scope != "p21.2_specialization_v1_v2" {
		t.Fatalf("scope = %q", report.Scope)
	}
	want := []SpecializationMachineCodeID{
		SpecializationMachineCodeGenerics,
		SpecializationMachineCodeProtocolStaticConformance,
		SpecializationMachineCodeExtensionMethods,
		SpecializationMachineCodeEnumKnownCases,
		SpecializationMachineCodeOptionals,
		SpecializationMachineCodeCollections,
	}
	if len(report.Rows) != len(want) {
		t.Fatalf("rows = %d, want %d: %#v", len(report.Rows), len(want), report.Rows)
	}
	byID := map[SpecializationMachineCodeID]SpecializationMachineCodeRow{}
	for _, row := range report.Rows {
		byID[row.ID] = row
		if row.Status != SpecializationMachineCodeImplementedNarrow {
			t.Fatalf("row %q status = %q, want implemented_narrow", row.ID, row.Status)
		}
		if row.SourceEvidence == "" || row.OptimizedIREvidence == "" ||
			row.MachineCodeEvidence == "" ||
			row.Boundary == "" {
			t.Fatalf("row %q missing required evidence: %#v", row.ID, row)
		}
		if len(row.RemovedHighLevelMarkers) == 0 {
			t.Fatalf("row %q missing removed high-level markers: %#v", row.ID, row)
		}
		if row.MachineWitnessID == "" {
			t.Fatalf("row %q missing machine witness id: %#v", row.ID, row)
		}
	}
	for _, id := range want {
		if _, ok := byID[id]; !ok {
			t.Fatalf("missing P21.2 row %q", id)
		}
	}
	for _, check := range []struct {
		id    SpecializationMachineCodeID
		wants []string
	}{
		{
			SpecializationMachineCodeGenerics,
			[]string{"monomorphized generic identity", "generic wrapper", "optimized Stack IR has no call", "Machine IR contains no OpCall", "no runtime generic values"},
		},
		{
			SpecializationMachineCodeProtocolStaticConformance,
			[]string{"statically checked protocol impl", "known direct Stack IR function symbol", "Machine IR contains no OpCall", "no witness tables", "dynamic dispatch"},
		},
		{
			SpecializationMachineCodeExtensionMethods,
			[]string{"statically resolved extension method", "direct Stack IR function symbol", "Machine IR contains no OpCall", "no dynamic extension dispatch"},
		},
		{
			SpecializationMachineCodeEnumKnownCases,
			[]string{"known-case match", "folded discriminator branch", "sccp-constant-branch", ("machine code carries no " +
				"match dispatch")},
		},
		{
			SpecializationMachineCodeOptionals,
			[]string{"proven-some optional", "folded presence branch", "constant_stack_store", ("machine code carries no " +
				"optional dispatch")},
		},
		{
			SpecializationMachineCodeCollections,
			[]string{"Vec<T>", "HashMap<K,V>", "monomorphized collection helper", "caller-owned", "Machine IR contains no OpCall", "no allocator-backed production"},
		},
	} {
		row := byID[check.id]
		haystack := row.Name + " " + row.SourceEvidence + " " + row.OptimizedIREvidence + " " + row.MachineCodeEvidence + " " + strings.Join(
			row.RemovedHighLevelMarkers,
			" ",
		) + " " + row.Boundary
		for _, want := range check.wants {
			if !strings.Contains(haystack, want) {
				t.Fatalf("row %q missing %q: %#v", check.id, want, row)
			}
		}
	}
	for _, want := range []string{
		"broad specialization is not claimed",
		"performance is not claimed",
		"safe-program semantics do not change",
		"dynamic protocol dispatch is not claimed",
		"runtime generic values are not claimed",
		"allocator-backed production generic collections are not claimed",
		"layout/ABI freedom is not claimed",
	} {
		if !containsP21String(report.NonClaims, want) {
			t.Fatalf("missing non-claim %q in %#v", want, report.NonClaims)
		}
	}

	missingMachine := cloneSpecializationMachineCodeCoverage(report)
	missingMachine.Rows[0].MachineCodeEvidence = ""
	if err := ValidateSpecializationMachineCodeCoverage(missingMachine); err == nil ||
		!strings.Contains(err.Error(), "machine") {
		t.Fatalf("missing machine evidence validation err = %v", err)
	}
	fakeWitness := cloneSpecializationMachineCodeCoverage(report)
	fakeWitness.Witnesses[0].MachineIRHasCall = true
	if err := ValidateSpecializationMachineCodeCoverage(fakeWitness); err == nil ||
		!strings.Contains(err.Error(), "witness") {
		t.Fatalf("fake witness validation err = %v", err)
	}
	placeholder := cloneSpecializationMachineCodeCoverage(report)
	placeholder.Rows[0].SourceEvidence = "TODO"
	if err := ValidateSpecializationMachineCodeCoverage(placeholder); err == nil ||
		!strings.Contains(err.Error(), "placeholder") {
		t.Fatalf("placeholder validation err = %v", err)
	}
	fakeBroad := cloneSpecializationMachineCodeCoverage(report)
	fakeBroad.BroadSpecializationClaimed = true
	if err := ValidateSpecializationMachineCodeCoverage(fakeBroad); err == nil ||
		!strings.Contains(err.Error(), "broad specialization") {
		t.Fatalf("fake broad specialization validation err = %v", err)
	}
	fakeDynamic := cloneSpecializationMachineCodeCoverage(report)
	fakeDynamic.DynamicDispatchClaimed = true
	if err := ValidateSpecializationMachineCodeCoverage(fakeDynamic); err == nil ||
		!strings.Contains(err.Error(), "dynamic dispatch") {
		t.Fatalf("fake dynamic dispatch validation err = %v", err)
	}
	fakeRuntimeGenerics := cloneSpecializationMachineCodeCoverage(report)
	fakeRuntimeGenerics.RuntimeGenericValuesClaimed = true
	if err := ValidateSpecializationMachineCodeCoverage(fakeRuntimeGenerics); err == nil ||
		!strings.Contains(err.Error(), "runtime generic") {
		t.Fatalf("fake runtime generics validation err = %v", err)
	}
	fakeCollections := cloneSpecializationMachineCodeCoverage(report)
	fakeCollections.AllocatorBackedCollectionsClaimed = true
	if err := ValidateSpecializationMachineCodeCoverage(fakeCollections); err == nil ||
		!strings.Contains(err.Error(), "allocator-backed") {
		t.Fatalf("fake collection runtime validation err = %v", err)
	}
	fakeLayout := cloneSpecializationMachineCodeCoverage(report)
	fakeLayout.LayoutABIFreedomClaimed = true
	if err := ValidateSpecializationMachineCodeCoverage(fakeLayout); err == nil ||
		!strings.Contains(err.Error(), "layout/ABI") {
		t.Fatalf("fake layout/ABI validation err = %v", err)
	}
	fakePerformance := cloneSpecializationMachineCodeCoverage(report)
	fakePerformance.PerformanceClaimed = true
	if err := ValidateSpecializationMachineCodeCoverage(fakePerformance); err == nil ||
		!strings.Contains(err.Error(), "performance") {
		t.Fatalf("fake performance validation err = %v", err)
	}
	fakeSafeSemantics := cloneSpecializationMachineCodeCoverage(report)
	fakeSafeSemantics.SafeSemanticsChanged = true
	if err := ValidateSpecializationMachineCodeCoverage(fakeSafeSemantics); err == nil ||
		!strings.Contains(err.Error(), "safe-program semantics") {
		t.Fatalf("fake safe-semantics validation err = %v", err)
	}
}

func TestP21SpecializationMachineCodeWitnessProvesDirectCallDisappearsBeforeMachineIR(
	t *testing.T,
) {
	witness, err := BuildP21SpecializationMachineCodeWitness()
	if err != nil {
		t.Fatalf("BuildP21SpecializationMachineCodeWitness: %v", err)
	}
	if witness.ID != "p21.2_known_direct_call_scalar_machine_witness" {
		t.Fatalf("witness id = %q", witness.ID)
	}
	if !witness.TranslationValidated {
		t.Fatalf("witness lacks translation validation: %#v", witness)
	}
	if !witness.StackIRHadCallBefore || witness.StackIRHasCallAfter {
		t.Fatalf("Stack IR call disappearance mismatch: %#v", witness)
	}
	if !witness.MachineIRVerified || witness.MachineIRHasCall {
		t.Fatalf("Machine IR call disappearance mismatch: %#v", witness)
	}
	if witness.MachineTarget != "scalar-int" {
		t.Fatalf("machine target = %q", witness.MachineTarget)
	}
	for _, want := range []string{"mov", "add", "return"} {
		if !containsP21String(witness.MachineOps, want) {
			t.Fatalf("machine ops missing %q: %#v", want, witness.MachineOps)
		}
	}
	if !containsP21String(witness.InlineDecisions, "main->known_i32_add:inlined:small_pure") {
		t.Fatalf("missing inline decision: %#v", witness.InlineDecisions)
	}
	for _, marker := range []string{"IRCall known_i32_add", "OpCall"} {
		if !containsP21String(witness.RemovedMarkers, marker) {
			t.Fatalf("missing removed marker %q: %#v", marker, witness.RemovedMarkers)
		}
	}
}

func cloneSpecializationMachineCodeCoverage(
	report SpecializationMachineCodeCoverageReport,
) SpecializationMachineCodeCoverageReport {
	out := report
	out.Rows = append([]SpecializationMachineCodeRow(nil), report.Rows...)
	out.Witnesses = append([]SpecializationMachineWitness(nil), report.Witnesses...)
	out.NonClaims = append([]string(nil), report.NonClaims...)
	for i := range out.Rows {
		out.Rows[i].Passes = append([]string(nil), report.Rows[i].Passes...)
		out.Rows[i].RemovedHighLevelMarkers = append(
			[]string(nil),
			report.Rows[i].RemovedHighLevelMarkers...)
	}
	return out
}

func containsP21String(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}

// ---- hotloop_test.go ----

func TestCoreHotLoopShapeEvidenceReportsRegisterRows(t *testing.T) {
	report, err := CoreHotLoopShapeEvidence()
	if err != nil {
		t.Fatalf("CoreHotLoopShapeEvidence: %v", err)
	}
	if report.SchemaVersion != "tetra.optimizer.hot_loop_shape.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	if !containsString(report.NonClaims, "no C/Rust -O1/-O2 performance parity claim") {
		t.Fatalf("non-claims = %#v, want explicit no performance parity claim", report.NonClaims)
	}

	rows := hotLoopRowsByID(report.Rows)
	for _, id := range []string{
		"scalar-sum-loop",
		"scalar-stride-sum-loop",
		"scalar-sum-squares-loop",
		"scalar-product-loop",
		"scalar-max-loop",
		"scalar-affine-sum-loop",
		"scalar-countdown-loop",
		"proof-slice-sum-loop",
		"proof-slice-stride-sum-loop",
		"call-sum-loop",
		"checked-slice-sum-fallback",
	} {
		if _, ok := rows[id]; !ok {
			t.Fatalf("missing hot-loop row %q in %#v", id, report.Rows)
		}
	}

	scalar := rows["scalar-sum-loop"]
	assertHotLoopRegisterRow(
		t,
		scalar,
		"machine-ir-loop",
		"scalar-int-loop",
		[]string{"cmp", "branch_if", "add", "inc"},
	)

	stride := rows["scalar-stride-sum-loop"]
	assertHotLoopRegisterRow(
		t,
		stride,
		"machine-ir-stride-loop",
		"scalar-int-loop",
		[]string{"mov", "cmp", "branch_if", "add"},
	)
	if containsString(stride.RequiredOps, "inc") {
		t.Fatalf("constant-stride row should use explicit stride add, not inc: %#v", stride)
	}

	squares := rows["scalar-sum-squares-loop"]
	assertHotLoopRegisterRow(
		t,
		squares,
		"machine-ir-sum-squares-loop",
		"scalar-int-sum-squares-loop",
		[]string{"cmp", "branch_if", "mul", "add", "inc"},
	)

	product := rows["scalar-product-loop"]
	assertHotLoopRegisterRow(
		t,
		product,
		"machine-ir-product-loop",
		"scalar-int-product-loop",
		[]string{"cmp", "branch_if", "add", "mul", "inc"},
	)

	max := rows["scalar-max-loop"]
	assertHotLoopRegisterRow(
		t,
		max,
		"machine-ir-max-loop",
		"scalar-int-max-loop",
		[]string{"cmp", "branch_if", "mov", "inc"},
	)

	affine := rows["scalar-affine-sum-loop"]
	assertHotLoopRegisterRow(
		t,
		affine,
		"machine-ir-affine-loop",
		"scalar-int-affine-loop",
		[]string{"cmp", "branch_if", "mul", "add", "inc"},
	)

	countdown := rows["scalar-countdown-loop"]
	assertHotLoopRegisterRow(
		t,
		countdown,
		"machine-ir-countdown-loop",
		"scalar-int-countdown-loop",
		[]string{"cmp", "branch_if", "add", "sub"},
	)

	slice := rows["proof-slice-sum-loop"]
	assertHotLoopRegisterRow(
		t,
		slice,
		"machine-ir-slice-sum",
		"scalar-i32-slice-sum",
		[]string{"cmp", "branch_if", "index_load", "add", "inc"},
	)
	if slice.ProofID == "" {
		t.Fatalf("slice row missing proof id: %#v", slice)
	}

	sliceStride := rows["proof-slice-stride-sum-loop"]
	assertHotLoopRegisterRow(
		t,
		sliceStride,
		"machine-ir-slice-stride-sum",
		"scalar-i32-slice-sum",
		[]string{"mov", "cmp", "branch_if", "index_load", "add"},
	)
	if containsString(sliceStride.RequiredOps, "inc") {
		t.Fatalf(
			"slice constant-stride row should use explicit stride add, not inc: %#v",
			sliceStride,
		)
	}
	if sliceStride.ProofID == "" {
		t.Fatalf("slice stride row missing proof id: %#v", sliceStride)
	}

	call := rows["call-sum-loop"]
	assertHotLoopRegisterRow(
		t,
		call,
		"machine-ir-call-loop",
		"scalar-int-call-loop",
		[]string{"cmp", "branch_if", "call", "add", "inc"},
	)
	if call.CallABI != "sysv" {
		t.Fatalf("call ABI = %q, want sysv in row %#v", call.CallABI, call)
	}
}

func TestCoreHotLoopShapeEvidenceReportsCheckedSliceFallback(t *testing.T) {
	report, err := CoreHotLoopShapeEvidence()
	if err != nil {
		t.Fatalf("CoreHotLoopShapeEvidence: %v", err)
	}
	row := hotLoopRowsByID(report.Rows)["checked-slice-sum-fallback"]
	if row.RegisterPath || row.SSAVerified || row.MachinePath != "stack-fallback" {
		t.Fatalf("checked slice fallback row = %#v, want explicit non-register fallback", row)
	}
	if row.Reason != "proof_tag_required_for_slice_sum_register_shape" {
		t.Fatalf(
			"fallback reason = %q, want proof_tag_required_for_slice_sum_register_shape",
			row.Reason,
		)
	}
	if row.Boundary == "" {
		t.Fatalf("fallback boundary missing: %#v", row)
	}
}

func assertHotLoopRegisterRow(
	t *testing.T,
	row HotLoopShapeRow,
	path string,
	target string,
	ops []string,
) {
	t.Helper()
	if !row.RegisterPath || !row.SSAVerified || row.MachinePath != path ||
		row.MachineTarget != target {
		t.Fatalf("row = %#v, want register path %s target %s with SSA verified", row, path, target)
	}
	if !row.SpillFree || row.StackChurnOps != 0 {
		t.Fatalf("row = %#v, want spill-free and no stack churn", row)
	}
	for _, op := range ops {
		if !containsString(row.RequiredOps, op) {
			t.Fatalf("row ops = %#v, want %q in row %#v", row.RequiredOps, op, row)
		}
	}
	if row.Boundary == "" || row.Evidence == "" {
		t.Fatalf("row missing evidence/boundary: %#v", row)
	}
}

func hotLoopRowsByID(rows []HotLoopShapeRow) map[string]HotLoopShapeRow {
	out := map[string]HotLoopShapeRow{}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out
}

// ---- inlining_test.go ----

func TestInlineSmallPurePassInlinesCallAndReportsDecision(t *testing.T) {
	prog := inlineAddProgram()

	report, err := NewManager().Run(prog, InlineSmallPurePass())
	if err != nil {
		t.Fatalf("Run InlineSmallPurePass: %v", err)
	}
	if len(report.Passes) != 1 {
		t.Fatalf("passes = %d, want 1", len(report.Passes))
	}
	row := report.Passes[0]
	if row.Name != "inline-small-pure" || row.ReportOutput != "inline-small-pure.opt.json" ||
		!row.TranslationValidated {
		t.Fatalf("metadata row = %#v", row)
	}
	if !strings.Contains(row.BeforeDump, "call add args:2 rets:1") {
		t.Fatalf("before dump missing call:\n%s", row.BeforeDump)
	}
	mainAfter := dumpFuncAfter(t, row.AfterDump, "main")
	if strings.Contains(mainAfter, "call add") {
		t.Fatalf("main after dump still contains call:\n%s", mainAfter)
	}
	for _, want := range []string{
		"store_local local:1",
		"store_local local:0",
		"load_local local:0",
		"load_local local:1",
		"add_i32",
	} {
		if !strings.Contains(mainAfter, want) {
			t.Fatalf("main after dump missing %q:\n%s", want, mainAfter)
		}
	}
	decision := requireDecision(t, row.Decisions, "inlined", "main", "add")
	if decision.Reason != "small_pure" {
		t.Fatalf("inlined reason = %q, want small_pure", decision.Reason)
	}
}

func TestInlineSmallPurePassReportsNotInlinedReasons(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{
			{
				Name:        "main",
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 1},
					{Kind: ir.IRCall, Name: "self", ArgSlots: 1, RetSlots: 1},
					{Kind: ir.IRCall, Name: "writer", ArgSlots: 0, RetSlots: 1},
					{Kind: ir.IRAddI32},
					{Kind: ir.IRConstI32, Imm: 0},
					{Kind: ir.IRConstI32, Imm: 0},
					{Kind: ir.IRConstI32, Imm: 0},
					{Kind: ir.IRCall, Name: "proofy", ArgSlots: 3, RetSlots: 1},
					{Kind: ir.IRAddI32},
					{Kind: ir.IRReturn},
				},
			},
			{
				Name:        "self",
				ParamSlots:  1,
				LocalSlots:  1,
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRLoadLocal, Local: 0},
					{Kind: ir.IRCall, Name: "self", ArgSlots: 1, RetSlots: 1},
					{Kind: ir.IRReturn},
				},
			},
			{
				Name:        "writer",
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRStrLit, Str: []byte("x")},
					{Kind: ir.IRWrite},
					{Kind: ir.IRConstI32, Imm: 0},
					{Kind: ir.IRReturn},
				},
			},
			{
				Name:        "proofy",
				ParamSlots:  3,
				LocalSlots:  3,
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRLoadLocal, Local: 0},
					{Kind: ir.IRLoadLocal, Local: 1},
					{Kind: ir.IRLoadLocal, Local: 2},
					{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:test"},
					{Kind: ir.IRReturn},
				},
			},
		},
	}

	report, err := NewManager().Run(prog, InlineSmallPurePass())
	if err != nil {
		t.Fatalf("Run InlineSmallPurePass: %v", err)
	}
	row := report.Passes[0]
	if got := requireDecision(
		t,
		row.Decisions,
		"not_inlined",
		"self",
		"self",
	).Reason; got != "recursive" {
		t.Fatalf("self recursive reason = %q, want recursive", got)
	}
	if got := requireDecision(
		t,
		row.Decisions,
		"not_inlined",
		"main",
		"self",
	).Reason; got != "callee_contains_call" {
		t.Fatalf("main->self reason = %q, want callee_contains_call", got)
	}
	if got := requireDecision(
		t,
		row.Decisions,
		"not_inlined",
		"main",
		"writer",
	).Reason; got != "unsupported_effect" {
		t.Fatalf("main->writer reason = %q, want unsupported_effect", got)
	}
	if got := requireDecision(
		t,
		row.Decisions,
		"not_inlined",
		"main",
		"proofy",
	).Reason; got != "proof_sensitive" {
		t.Fatalf("main->proofy reason = %q, want proof_sensitive", got)
	}
}

func TestInlineSmallPurePassDifferentialExecution(t *testing.T) {
	before := inlineAddProgram()
	after := cloneProgram(before)
	report, err := NewManager().Run(after, InlineSmallPurePass())
	if err != nil {
		t.Fatalf("Run InlineSmallPurePass: %v", err)
	}
	if len(report.Passes[0].Decisions) == 0 {
		t.Fatalf("missing inline decisions")
	}

	beforeExit := runOptLinuxX64(t, before.Funcs, "before-inline-small-pure")
	afterExit := runOptLinuxX64(t, after.Funcs, "after-inline-small-pure")
	if beforeExit != afterExit {
		t.Fatalf("exit mismatch before=%d after=%d", beforeExit, afterExit)
	}
	if afterExit != 42 {
		t.Fatalf("optimized exit = %d, want 42", afterExit)
	}
}

func inlineAddProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{
			{
				Name:        "main",
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 20},
					{Kind: ir.IRConstI32, Imm: 22},
					{Kind: ir.IRCall, Name: "add", ArgSlots: 2, RetSlots: 1},
					{Kind: ir.IRReturn},
				},
			},
			{
				Name:        "add",
				ParamSlots:  2,
				LocalSlots:  2,
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRLoadLocal, Local: 0},
					{Kind: ir.IRLoadLocal, Local: 1},
					{Kind: ir.IRAddI32},
					{Kind: ir.IRReturn},
				},
			},
		},
	}
}

func requireDecision(
	t *testing.T,
	decisions []PassDecision,
	action string,
	caller string,
	callee string,
) PassDecision {
	t.Helper()
	for _, decision := range decisions {
		if decision.Action == action && decision.Caller == caller && decision.Callee == callee {
			return decision
		}
	}
	t.Fatalf(
		"missing decision action=%s caller=%s callee=%s in %#v",
		action,
		caller,
		callee,
		decisions,
	)
	return PassDecision{}
}

func dumpFuncAfter(t *testing.T, dump string, name string) string {
	t.Helper()
	marker := "func " + name + " "
	start := strings.Index(dump, marker)
	if start < 0 {
		t.Fatalf("dump missing function %q:\n%s", name, dump)
	}
	rest := dump[start:]
	next := strings.Index(rest[len(marker):], "\nfunc ")
	if next < 0 {
		return rest
	}
	return rest[:len(marker)+next]
}

// ---- licm_test.go ----

func TestLICMPureInvariantPassHoistsPureComparisonInsideProofLoop(t *testing.T) {
	prog := licmInvariantProgram()

	report, err := NewManager().Run(prog, LICMPureInvariantPass())
	if err != nil {
		t.Fatalf("Run LICMPureInvariantPass: %v", err)
	}
	if prog.Funcs[0].LocalSlots != 6 {
		t.Fatalf("LocalSlots = %d, want new hoisted invariant local", prog.Funcs[0].LocalSlots)
	}
	after := report.Passes[0].AfterDump
	for _, want := range []string{
		"store_local local:5",
		"load_local local:5",
		"cmp_gt_i32",
		"proof:proof:while:i:xs:1:1",
	} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	if strings.Count(after, "cmp_gt_i32") != 1 {
		t.Fatalf("after dump should keep only the hoisted invariant comparison:\n%s", after)
	}
	row := report.Passes[0]
	if row.Name != "licm-pure-invariant" || !row.TranslationValidated {
		t.Fatalf("pass row = %#v", row)
	}
	if !hasDecision(row.Decisions, "hoisted", "pure_invariant_comparison") {
		t.Fatalf("decisions missing LICM evidence: %#v", row.Decisions)
	}
}

func TestLICMPureInvariantPassHoistsPureArithmeticInsideProofLoop(t *testing.T) {
	prog := licmInvariantProgram()
	prog.Funcs[0].Instrs[15].Imm = 7
	prog.Funcs[0].Instrs[16].Kind = ir.IRAddI32

	report, err := NewManager().Run(prog, LICMPureInvariantPass())
	if err != nil {
		t.Fatalf("Run LICMPureInvariantPass: %v", err)
	}
	if prog.Funcs[0].LocalSlots != 6 {
		t.Fatalf("LocalSlots = %d, want new hoisted invariant local", prog.Funcs[0].LocalSlots)
	}
	after := report.Passes[0].AfterDump
	for _, want := range []string{
		"store_local local:5",
		"load_local local:5",
		"add_i32",
		"proof:proof:while:i:xs:1:1",
	} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	if strings.Count(after, "const_i32 7") != 1 {
		t.Fatalf("after dump should keep only the hoisted arithmetic constant:\n%s", after)
	}
	if !hoistedBeforeLoopLabel(after, "store_local local:5") {
		t.Fatalf("arithmetic invariant was not hoisted before the loop label:\n%s", after)
	}
	row := report.Passes[0]
	if row.Name != "licm-pure-invariant" || !row.TranslationValidated {
		t.Fatalf("pass row = %#v", row)
	}
	if !hasDecision(row.Decisions, "hoisted", "pure_invariant_arithmetic") {
		t.Fatalf("decisions missing arithmetic LICM evidence: %#v", row.Decisions)
	}
}

func TestLICMPureInvariantPassHoistsPureSubArithmeticInsideProofLoop(t *testing.T) {
	prog := licmInvariantProgram()
	prog.Funcs[0].Instrs[15].Imm = 7
	prog.Funcs[0].Instrs[16].Kind = ir.IRSubI32

	report, err := NewManager().Run(prog, LICMPureInvariantPass())
	if err != nil {
		t.Fatalf("Run LICMPureInvariantPass: %v", err)
	}
	if prog.Funcs[0].LocalSlots != 6 {
		t.Fatalf("LocalSlots = %d, want new hoisted invariant local", prog.Funcs[0].LocalSlots)
	}
	after := report.Passes[0].AfterDump
	for _, want := range []string{
		"store_local local:5",
		"load_local local:5",
		"sub_i32",
		"proof:proof:while:i:xs:1:1",
	} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	if strings.Count(after, "sub_i32") != 1 {
		t.Fatalf("after dump should keep only the hoisted subtraction:\n%s", after)
	}
	if !hoistedBeforeLoopLabel(after, "store_local local:5") {
		t.Fatalf("subtraction invariant was not hoisted before the loop label:\n%s", after)
	}
	row := report.Passes[0]
	if row.Name != "licm-pure-invariant" || !row.TranslationValidated {
		t.Fatalf("pass row = %#v", row)
	}
	if !hasDecision(row.Decisions, "hoisted", "pure_invariant_arithmetic") {
		t.Fatalf("decisions missing subtraction LICM evidence: %#v", row.Decisions)
	}
}

func TestLICMPureInvariantPassHoistsSafeDivArithmeticInsideProofLoop(t *testing.T) {
	prog := licmInvariantProgram()
	prog.Funcs[0].Instrs[15].Imm = 3
	prog.Funcs[0].Instrs[16].Kind = ir.IRDivI32

	report, err := NewManager().Run(prog, LICMPureInvariantPass())
	if err != nil {
		t.Fatalf("Run LICMPureInvariantPass: %v", err)
	}
	if prog.Funcs[0].LocalSlots != 6 {
		t.Fatalf("LocalSlots = %d, want new hoisted invariant local", prog.Funcs[0].LocalSlots)
	}
	after := report.Passes[0].AfterDump
	for _, want := range []string{
		"store_local local:5",
		"load_local local:5",
		"div_i32",
		"proof:proof:while:i:xs:1:1",
	} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	if strings.Count(after, "div_i32") != 1 {
		t.Fatalf("after dump should keep only the hoisted division:\n%s", after)
	}
	if !hoistedBeforeLoopLabel(after, "store_local local:5") {
		t.Fatalf("division invariant was not hoisted before the loop label:\n%s", after)
	}
	row := report.Passes[0]
	if row.Name != "licm-pure-invariant" || !row.TranslationValidated {
		t.Fatalf("pass row = %#v", row)
	}
	if !hasDecision(row.Decisions, "hoisted", "pure_invariant_safe_division") {
		t.Fatalf("decisions missing safe-division LICM evidence: %#v", row.Decisions)
	}
}

func TestLICMPureInvariantPassHoistsSafeModArithmeticInsideProofLoop(t *testing.T) {
	prog := licmInvariantProgram()
	prog.Funcs[0].Instrs[15].Imm = 3
	prog.Funcs[0].Instrs[16].Kind = ir.IRModI32

	report, err := NewManager().Run(prog, LICMPureInvariantPass())
	if err != nil {
		t.Fatalf("Run LICMPureInvariantPass: %v", err)
	}
	if prog.Funcs[0].LocalSlots != 6 {
		t.Fatalf("LocalSlots = %d, want new hoisted invariant local", prog.Funcs[0].LocalSlots)
	}
	after := report.Passes[0].AfterDump
	for _, want := range []string{
		"store_local local:5",
		"load_local local:5",
		"mod_i32",
		"proof:proof:while:i:xs:1:1",
	} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	if strings.Count(after, "mod_i32") != 1 {
		t.Fatalf("after dump should keep only the hoisted modulo:\n%s", after)
	}
	if !hoistedBeforeLoopLabel(after, "store_local local:5") {
		t.Fatalf("modulo invariant was not hoisted before the loop label:\n%s", after)
	}
	row := report.Passes[0]
	if row.Name != "licm-pure-invariant" || !row.TranslationValidated {
		t.Fatalf("pass row = %#v", row)
	}
	if !hasDecision(row.Decisions, "hoisted", "pure_invariant_safe_modulo") {
		t.Fatalf("decisions missing safe-modulo LICM evidence: %#v", row.Decisions)
	}
}

func TestLICMPureInvariantPassHoistsSafeKnownLocalDivModInsideProofLoop(t *testing.T) {
	cases := []struct {
		name   string
		kind   ir.IRInstrKind
		op     string
		reason string
	}{
		{
			name:   "division",
			kind:   ir.IRDivI32,
			op:     "div_i32",
			reason: "pure_invariant_safe_known_local_division",
		},
		{
			name:   "modulo",
			kind:   ir.IRModI32,
			op:     "mod_i32",
			reason: "pure_invariant_safe_known_local_modulo",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog := licmKnownLocalDivModInvariantProgram(tc.kind, 3)

			report, err := NewManager().Run(prog, LICMPureInvariantPass())
			if err != nil {
				t.Fatalf("Run LICMPureInvariantPass: %v", err)
			}
			if prog.Funcs[0].LocalSlots != 7 {
				t.Fatalf(
					"LocalSlots = %d, want new hoisted invariant local",
					prog.Funcs[0].LocalSlots,
				)
			}
			after := report.Passes[0].AfterDump
			for _, want := range []string{
				"store_local local:6",
				"load_local local:6",
				"load_local local:5",
				tc.op,
				"proof:proof:while:i:xs:1:1",
			} {
				if !strings.Contains(after, want) {
					t.Fatalf("after dump missing %q:\n%s", want, after)
				}
			}
			if strings.Count(after, tc.op) != 1 {
				t.Fatalf(
					"after dump should keep only the hoisted known-local %s:\n%s",
					tc.op,
					after,
				)
			}
			if !hoistedBeforeLoopLabel(after, "store_local local:6") {
				t.Fatalf(
					"known-local %s invariant was not hoisted before the loop label:\n%s",
					tc.op,
					after,
				)
			}
			if !hasDecision(report.Passes[0].Decisions, "hoisted", tc.reason) {
				t.Fatalf(
					"decisions missing safe known-local %s LICM evidence: %#v",
					tc.op,
					report.Passes[0].Decisions,
				)
			}
		})
	}
}

func TestLICMPureInvariantPassHoistsKnownLocalArithmeticInsideProofLoop(t *testing.T) {
	cases := []struct {
		name   string
		kind   ir.IRInstrKind
		op     string
		reason string
	}{
		{
			name:   "addition",
			kind:   ir.IRAddI32,
			op:     "add_i32",
			reason: "pure_invariant_known_local_arithmetic",
		},
		{
			name:   "subtraction",
			kind:   ir.IRSubI32,
			op:     "sub_i32",
			reason: "pure_invariant_known_local_arithmetic",
		},
		{
			name:   "multiplication",
			kind:   ir.IRMulI32,
			op:     "mul_i32",
			reason: "pure_invariant_known_local_arithmetic",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog := licmKnownLocalArithmeticInvariantProgram(tc.kind, 7)

			report, err := NewManager().Run(prog, LICMPureInvariantPass())
			if err != nil {
				t.Fatalf("Run LICMPureInvariantPass: %v", err)
			}
			if prog.Funcs[0].LocalSlots != 7 {
				t.Fatalf(
					"LocalSlots = %d, want new hoisted invariant local",
					prog.Funcs[0].LocalSlots,
				)
			}
			after := report.Passes[0].AfterDump
			for _, want := range []string{
				"store_local local:6",
				"load_local local:6",
				"load_local local:5",
				tc.op,
				"proof:proof:while:i:xs:1:1",
			} {
				if !strings.Contains(after, want) {
					t.Fatalf("after dump missing %q:\n%s", want, after)
				}
			}
			if strings.Count(after, "load_local local:5") != 1 {
				t.Fatalf(
					("after dump should load the known-local arithmetic operand " +
						"only in the hoisted expression:\n%s"),
					after,
				)
			}
			if !hoistedBeforeLoopLabel(after, "store_local local:6") {
				t.Fatalf(
					"known-local %s invariant was not hoisted before the loop label:\n%s",
					tc.op,
					after,
				)
			}
			if !hasDecision(report.Passes[0].Decisions, "hoisted", tc.reason) {
				t.Fatalf(
					"decisions missing known-local %s LICM evidence: %#v",
					tc.op,
					report.Passes[0].Decisions,
				)
			}
		})
	}
}

func TestLICMPureInvariantPassRejectsKnownLocalArithmeticWhenOperandMutatesInLoop(t *testing.T) {
	prog := licmKnownLocalArithmeticInvariantProgram(ir.IRMulI32, 7)
	insertAt := 16
	prog.Funcs[0].Instrs = append(prog.Funcs[0].Instrs[:insertAt], append([]ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 5},
		{Kind: ir.IRStoreLocal, Local: 5},
	}, prog.Funcs[0].Instrs[insertAt:]...)...)

	report, err := NewManager().Run(prog, LICMPureInvariantPass())
	if err != nil {
		t.Fatalf("Run LICMPureInvariantPass: %v", err)
	}
	after := report.Passes[0].AfterDump
	if strings.Contains(after, "store_local local:6") {
		t.Fatalf(
			"mutating known-local arithmetic operand invariant expression was hoisted:\n%s",
			after,
		)
	}
	if !hasDecision(report.Passes[0].Decisions, "not_hoisted", "loop_stores_invariant_operand") {
		t.Fatalf(
			"decisions missing known-local arithmetic operand mutation rejection: %#v",
			report.Passes[0].Decisions,
		)
	}
}

func TestLICMPureInvariantPassHoistsKnownLocalLeftArithmeticInsideProofLoop(t *testing.T) {
	cases := []struct {
		name string
		kind ir.IRInstrKind
		op   string
	}{
		{name: "addition", kind: ir.IRAddI32, op: "add_i32"},
		{name: "subtraction", kind: ir.IRSubI32, op: "sub_i32"},
		{name: "multiplication", kind: ir.IRMulI32, op: "mul_i32"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog := licmKnownLocalLeftArithmeticInvariantProgram(tc.kind, 7)

			report, err := NewManager().Run(prog, LICMPureInvariantPass())
			if err != nil {
				t.Fatalf("Run LICMPureInvariantPass: %v", err)
			}
			if prog.Funcs[0].LocalSlots != 7 {
				t.Fatalf(
					"LocalSlots = %d, want new hoisted invariant local",
					prog.Funcs[0].LocalSlots,
				)
			}
			after := report.Passes[0].AfterDump
			for _, want := range []string{
				"store_local local:6",
				"load_local local:6",
				"load_local local:5",
				tc.op,
				"proof:proof:while:i:xs:1:1",
			} {
				if !strings.Contains(after, want) {
					t.Fatalf("after dump missing %q:\n%s", want, after)
				}
			}
			if strings.Count(after, "load_local local:5") != 1 {
				t.Fatalf(
					"after dump should load the known-local left operand only in the hoisted expression:\n%s",
					after,
				)
			}
			if !hoistedBeforeLoopLabel(after, "store_local local:6") {
				t.Fatalf(
					"known-local left %s invariant was not hoisted before the loop label:\n%s",
					tc.op,
					after,
				)
			}
			if !hasDecision(
				report.Passes[0].Decisions,
				"hoisted",
				"pure_invariant_known_local_arithmetic",
			) {
				t.Fatalf(
					"decisions missing known-local left %s LICM evidence: %#v",
					tc.op,
					report.Passes[0].Decisions,
				)
			}
		})
	}
}

func TestLICMPureInvariantPassRejectsKnownLocalLeftArithmeticWhenOperandMutatesInLoop(
	t *testing.T,
) {
	prog := licmKnownLocalLeftArithmeticInvariantProgram(ir.IRSubI32, 7)
	insertAt := 16
	prog.Funcs[0].Instrs = append(prog.Funcs[0].Instrs[:insertAt], append([]ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 5},
		{Kind: ir.IRStoreLocal, Local: 5},
	}, prog.Funcs[0].Instrs[insertAt:]...)...)

	report, err := NewManager().Run(prog, LICMPureInvariantPass())
	if err != nil {
		t.Fatalf("Run LICMPureInvariantPass: %v", err)
	}
	after := report.Passes[0].AfterDump
	if strings.Contains(after, "store_local local:6") {
		t.Fatalf(
			"mutating known-local left arithmetic operand invariant expression was hoisted:\n%s",
			after,
		)
	}
	if !hasDecision(report.Passes[0].Decisions, "not_hoisted", "loop_stores_invariant_operand") {
		t.Fatalf(
			"decisions missing known-local left arithmetic operand mutation rejection: %#v",
			report.Passes[0].Decisions,
		)
	}
}

func TestLICMPureInvariantPassHoistsKnownLocalComparisonInsideProofLoop(t *testing.T) {
	ops := []struct {
		name string
		kind ir.IRInstrKind
		op   string
	}{
		{name: "eq", kind: ir.IRCmpEqI32, op: "cmp_eq_i32"},
		{name: "lt", kind: ir.IRCmpLtI32, op: "cmp_lt_i32"},
		{name: "gt", kind: ir.IRCmpGtI32, op: "cmp_gt_i32"},
		{name: "ge", kind: ir.IRCmpGeI32, op: "cmp_ge_i32"},
		{name: "le", kind: ir.IRCmpLeI32, op: "cmp_le_i32"},
		{name: "ne", kind: ir.IRCmpNeI32, op: "cmp_ne_i32"},
	}
	positions := []struct {
		name        string
		knownOnLeft bool
	}{
		{name: "known-left", knownOnLeft: true},
		{name: "known-right", knownOnLeft: false},
	}

	for _, pos := range positions {
		for _, op := range ops {
			t.Run(pos.name+"-"+op.name, func(t *testing.T) {
				prog := licmKnownLocalComparisonInvariantProgram(op.kind, 7, pos.knownOnLeft)

				report, err := NewManager().Run(prog, LICMPureInvariantPass())
				if err != nil {
					t.Fatalf("Run LICMPureInvariantPass: %v", err)
				}
				if prog.Funcs[0].LocalSlots != 7 {
					t.Fatalf(
						"LocalSlots = %d, want new hoisted invariant local",
						prog.Funcs[0].LocalSlots,
					)
				}
				after := report.Passes[0].AfterDump
				for _, want := range []string{
					"store_local local:6",
					"load_local local:6",
					"load_local local:5",
					op.op,
					"proof:proof:while:i:xs:1:1",
				} {
					if !strings.Contains(after, want) {
						t.Fatalf("after dump missing %q:\n%s", want, after)
					}
				}
				if strings.Count(after, "load_local local:5") != 1 {
					t.Fatalf(
						("after dump should load the known-local comparison operand " +
							"only in the hoisted expression:\n%s"),
						after,
					)
				}
				if !hoistedBeforeLoopLabel(after, "store_local local:6") {
					t.Fatalf(
						"known-local %s comparison invariant was not hoisted before the loop label:\n%s",
						op.op,
						after,
					)
				}
				if !hasDecision(
					report.Passes[0].Decisions,
					"hoisted",
					"pure_invariant_known_local_comparison",
				) {
					t.Fatalf(
						"decisions missing known-local comparison LICM evidence: %#v",
						report.Passes[0].Decisions,
					)
				}
			})
		}
	}
}

func TestLICMPureInvariantPassRejectsKnownLocalComparisonWhenOperandMutatesInLoop(t *testing.T) {
	prog := licmKnownLocalComparisonInvariantProgram(ir.IRCmpLtI32, 7, false)
	insertAt := 16
	prog.Funcs[0].Instrs = append(prog.Funcs[0].Instrs[:insertAt], append([]ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 5},
		{Kind: ir.IRStoreLocal, Local: 5},
	}, prog.Funcs[0].Instrs[insertAt:]...)...)

	report, err := NewManager().Run(prog, LICMPureInvariantPass())
	if err != nil {
		t.Fatalf("Run LICMPureInvariantPass: %v", err)
	}
	after := report.Passes[0].AfterDump
	if strings.Contains(after, "store_local local:6") {
		t.Fatalf(
			"mutating known-local comparison operand invariant expression was hoisted:\n%s",
			after,
		)
	}
	if !hasDecision(report.Passes[0].Decisions, "not_hoisted", "loop_stores_invariant_operand") {
		t.Fatalf(
			"decisions missing known-local comparison operand mutation rejection: %#v",
			report.Passes[0].Decisions,
		)
	}
}

func TestLICMPureInvariantPassRejectsUnsafeKnownLocalDivModInsideProofLoop(t *testing.T) {
	cases := []struct {
		name   string
		kind   ir.IRInstrKind
		op     string
		denom  int32
		reason string
	}{
		{
			name:   "division by zero",
			kind:   ir.IRDivI32,
			op:     "div_i32",
			denom:  0,
			reason: "unsafe_known_local_division_denominator",
		},
		{
			name:   "division by minus one",
			kind:   ir.IRDivI32,
			op:     "div_i32",
			denom:  -1,
			reason: "unsafe_known_local_division_denominator",
		},
		{
			name:   "modulo by zero",
			kind:   ir.IRModI32,
			op:     "mod_i32",
			denom:  0,
			reason: "unsafe_known_local_modulo_denominator",
		},
		{
			name:   "modulo by minus one",
			kind:   ir.IRModI32,
			op:     "mod_i32",
			denom:  -1,
			reason: "unsafe_known_local_modulo_denominator",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog := licmKnownLocalDivModInvariantProgram(tc.kind, tc.denom)

			report, err := NewManager().Run(prog, LICMPureInvariantPass())
			if err != nil {
				t.Fatalf("Run LICMPureInvariantPass: %v", err)
			}
			after := report.Passes[0].AfterDump
			if strings.Contains(after, "store_local local:6") {
				t.Fatalf(
					"unsafe known-local %s invariant expression was hoisted:\n%s",
					tc.op,
					after,
				)
			}
			if strings.Count(after, tc.op) != 1 {
				t.Fatalf(
					"unsafe known-local %s expression should remain in loop:\n%s",
					tc.op,
					after,
				)
			}
			if !hasDecision(report.Passes[0].Decisions, "not_hoisted", tc.reason) {
				t.Fatalf("decisions missing %q: %#v", tc.reason, report.Passes[0].Decisions)
			}
		})
	}
}

func TestLICMPureInvariantPassRejectsSafeKnownLocalDivModWhenDenominatorMutatesInLoop(
	t *testing.T,
) {
	prog := licmKnownLocalDivModInvariantProgram(ir.IRDivI32, 3)
	insertAt := 16
	prog.Funcs[0].Instrs = append(prog.Funcs[0].Instrs[:insertAt], append([]ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 5},
		{Kind: ir.IRStoreLocal, Local: 5},
	}, prog.Funcs[0].Instrs[insertAt:]...)...)

	report, err := NewManager().Run(prog, LICMPureInvariantPass())
	if err != nil {
		t.Fatalf("Run LICMPureInvariantPass: %v", err)
	}
	after := report.Passes[0].AfterDump
	if strings.Contains(after, "store_local local:6") {
		t.Fatalf("mutating known-local denominator invariant expression was hoisted:\n%s", after)
	}
	if !hasDecision(report.Passes[0].Decisions, "not_hoisted", "loop_stores_invariant_operand") {
		t.Fatalf(
			"decisions missing denominator mutation rejection: %#v",
			report.Passes[0].Decisions,
		)
	}
}

func TestLICMPureInvariantPassRejectsVariantOrMutatedExpressions(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ir.IRProgram)
		reason string
	}{
		{
			name: "loop index operand",
			mutate: func(p *ir.IRProgram) {
				// The candidate expression becomes `i > 0`, which is variant.
				p.Funcs[0].Instrs[14].Local = 0
			},
			reason: "variant_loop_index_operand",
		},
		{
			name: "stored invariant operand",
			mutate: func(p *ir.IRProgram) {
				insertAt := 14
				p.Funcs[0].Instrs = append(p.Funcs[0].Instrs[:insertAt], append([]ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 7},
					{Kind: ir.IRStoreLocal, Local: 4},
				}, p.Funcs[0].Instrs[insertAt:]...)...)
			},
			reason: "loop_stores_invariant_operand",
		},
		{
			name: "arithmetic loop index operand",
			mutate: func(p *ir.IRProgram) {
				p.Funcs[0].Instrs[14].Local = 0
				p.Funcs[0].Instrs[15].Imm = 7
				p.Funcs[0].Instrs[16].Kind = ir.IRAddI32
			},
			reason: "variant_loop_index_operand",
		},
		{
			name: "division by zero denominator",
			mutate: func(p *ir.IRProgram) {
				p.Funcs[0].Instrs[15].Imm = 0
				p.Funcs[0].Instrs[16].Kind = ir.IRDivI32
			},
			reason: "unsafe_division_denominator",
		},
		{
			name: "division by minus one denominator",
			mutate: func(p *ir.IRProgram) {
				p.Funcs[0].Instrs[15].Imm = -1
				p.Funcs[0].Instrs[16].Kind = ir.IRDivI32
			},
			reason: "unsafe_division_denominator",
		},
		{
			name: "modulo by zero denominator",
			mutate: func(p *ir.IRProgram) {
				p.Funcs[0].Instrs[15].Imm = 0
				p.Funcs[0].Instrs[16].Kind = ir.IRModI32
			},
			reason: "unsafe_modulo_denominator",
		},
		{
			name: "modulo by minus one denominator",
			mutate: func(p *ir.IRProgram) {
				p.Funcs[0].Instrs[15].Imm = -1
				p.Funcs[0].Instrs[16].Kind = ir.IRModI32
			},
			reason: "unsafe_modulo_denominator",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog := licmInvariantProgram()
			tc.mutate(prog)
			report, err := NewManager().Run(prog, LICMPureInvariantPass())
			if err != nil {
				t.Fatalf("Run LICMPureInvariantPass: %v", err)
			}
			after := report.Passes[0].AfterDump
			if strings.Contains(after, "store_local local:5") {
				t.Fatalf("unsafe invariant expression was hoisted:\n%s", after)
			}
			if !hasDecision(report.Passes[0].Decisions, "not_hoisted", tc.reason) {
				t.Fatalf("decisions missing %q: %#v", tc.reason, report.Passes[0].Decisions)
			}
		})
	}
}

func hoistedBeforeLoopLabel(dump string, hoisted string) bool {
	hoistedIndex := strings.Index(dump, hoisted)
	labelIndex := strings.Index(dump, "label label:1")
	return hoistedIndex >= 0 && labelIndex >= 0 && hoistedIndex < labelIndex
}

func licmInvariantProgram() *ir.IRProgram {
	instrs := []ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRStoreLocal, Local: 0},
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRStoreLocal, Local: 3},
		{Kind: ir.IRLabel, Label: 1},
		{Kind: ir.IRLoadLocal, Local: 0},
		{Kind: ir.IRLoadLocal, Local: 2},
		{Kind: ir.IRCmpLtI32},
		{Kind: ir.IRJmpIfZero, Label: 2},
		{Kind: ir.IRLoadLocal, Local: 3},
		{Kind: ir.IRLoadLocal, Local: 1},
		{Kind: ir.IRLoadLocal, Local: 2},
		{Kind: ir.IRLoadLocal, Local: 0},
		{Kind: ir.IRIndexLoadI32Unchecked, ProofID: proofID(true)},
		{Kind: ir.IRLoadLocal, Local: 4},
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRCmpGtI32},
		{Kind: ir.IRAddI32},
		{Kind: ir.IRAddI32},
		{Kind: ir.IRStoreLocal, Local: 3},
		{Kind: ir.IRLoadLocal, Local: 0},
		{Kind: ir.IRConstI32, Imm: 1},
		{Kind: ir.IRAddI32},
		{Kind: ir.IRStoreLocal, Local: 0},
		{Kind: ir.IRJmp, Label: 1},
		{Kind: ir.IRLabel, Label: 2},
		{Kind: ir.IRLoadLocal, Local: 3},
		{Kind: ir.IRReturn},
	}
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  5,
			LocalSlots:  5,
			ReturnSlots: 1,
			Instrs:      instrs,
		}},
	}
}

func licmKnownLocalDivModInvariantProgram(kind ir.IRInstrKind, denominator int32) *ir.IRProgram {
	prog := licmInvariantProgram()
	prog.Funcs[0].LocalSlots = 6
	prog.Funcs[0].Instrs = append(prog.Funcs[0].Instrs[:4], append([]ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: denominator},
		{Kind: ir.IRStoreLocal, Local: 5},
	}, prog.Funcs[0].Instrs[4:]...)...)
	prog.Funcs[0].Instrs[16] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 4}
	prog.Funcs[0].Instrs[17] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 5}
	prog.Funcs[0].Instrs[18] = ir.IRInstr{Kind: kind}
	return prog
}

func licmKnownLocalArithmeticInvariantProgram(kind ir.IRInstrKind, right int32) *ir.IRProgram {
	prog := licmInvariantProgram()
	prog.Funcs[0].LocalSlots = 6
	prog.Funcs[0].Instrs = append(prog.Funcs[0].Instrs[:4], append([]ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: right},
		{Kind: ir.IRStoreLocal, Local: 5},
	}, prog.Funcs[0].Instrs[4:]...)...)
	prog.Funcs[0].Instrs[16] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 4}
	prog.Funcs[0].Instrs[17] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 5}
	prog.Funcs[0].Instrs[18] = ir.IRInstr{Kind: kind}
	return prog
}

func licmKnownLocalLeftArithmeticInvariantProgram(kind ir.IRInstrKind, left int32) *ir.IRProgram {
	prog := licmInvariantProgram()
	prog.Funcs[0].LocalSlots = 6
	prog.Funcs[0].Instrs = append(prog.Funcs[0].Instrs[:4], append([]ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: left},
		{Kind: ir.IRStoreLocal, Local: 5},
	}, prog.Funcs[0].Instrs[4:]...)...)
	prog.Funcs[0].Instrs[16] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 5}
	prog.Funcs[0].Instrs[17] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 4}
	prog.Funcs[0].Instrs[18] = ir.IRInstr{Kind: kind}
	return prog
}

func licmKnownLocalComparisonInvariantProgram(
	kind ir.IRInstrKind,
	value int32,
	knownOnLeft bool,
) *ir.IRProgram {
	prog := licmInvariantProgram()
	prog.Funcs[0].LocalSlots = 6
	prog.Funcs[0].Instrs = append(prog.Funcs[0].Instrs[:4], append([]ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: value},
		{Kind: ir.IRStoreLocal, Local: 5},
	}, prog.Funcs[0].Instrs[4:]...)...)
	if knownOnLeft {
		prog.Funcs[0].Instrs[16] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 5}
		prog.Funcs[0].Instrs[17] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 4}
	} else {
		prog.Funcs[0].Instrs[16] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 4}
		prog.Funcs[0].Instrs[17] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 5}
	}
	prog.Funcs[0].Instrs[18] = ir.IRInstr{Kind: kind}
	return prog
}

// ---- loop_test.go ----

func TestLoopCanonicalizationPassHoistsStableLenAndCanonicalizesLeMinusOne(t *testing.T) {
	prog := loopCanonicalizationProgram(ir.IRCmpLeI32, true)

	report, err := NewManager().Run(prog, LoopCanonicalizationPass())
	if err != nil {
		t.Fatalf("Run LoopCanonicalizationPass: %v", err)
	}
	if prog.Funcs[0].LocalSlots != 5 {
		t.Fatalf("LocalSlots = %d, want new hoisted len local", prog.Funcs[0].LocalSlots)
	}
	after := report.Passes[0].AfterDump
	for _, want := range []string{
		"store_local local:4",
		"load_local local:4",
		"cmp_lt_i32",
		"proof:proof:while:i:xs:1:1",
	} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	for _, forbidden := range []string{"sub_i32", "cmp_le_i32"} {
		if strings.Contains(after, forbidden) {
			t.Fatalf("after dump still contains %q:\n%s", forbidden, after)
		}
	}
	row := report.Passes[0]
	if row.Name != "loop-canonicalization" || !row.TranslationValidated {
		t.Fatalf("pass row = %#v", row)
	}
	if !hasDecision(row.Decisions, "canonicalized", "stable_len_le_minus_one_to_lt") {
		t.Fatalf("decisions missing canonicalization evidence: %#v", row.Decisions)
	}
}

func TestLoopCanonicalizationPassHoistsStableLenForLessThanLoop(t *testing.T) {
	prog := loopCanonicalizationProgram(ir.IRCmpLtI32, true)

	report, err := NewManager().Run(prog, LoopCanonicalizationPass())
	if err != nil {
		t.Fatalf("Run LoopCanonicalizationPass: %v", err)
	}
	after := report.Passes[0].AfterDump
	if !strings.Contains(after, "store_local local:4") ||
		!strings.Contains(after, "load_local local:4") {
		t.Fatalf("after dump missing hoisted len local:\n%s", after)
	}
	if strings.Contains(after, "cmp_le_i32") || strings.Contains(after, "sub_i32") {
		t.Fatalf("less-than loop unexpectedly contains <= canonicalization remnants:\n%s", after)
	}
	if !hasDecision(report.Passes[0].Decisions, "hoisted", "stable_len_load") {
		t.Fatalf("decisions missing hoist evidence: %#v", report.Passes[0].Decisions)
	}
}

func TestLoopCanonicalizationPassRejectsUnsafeLoopShapes(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ir.IRProgram)
		reason string
	}{
		{
			name: "missing proof",
			mutate: func(p *ir.IRProgram) {
				for i := range p.Funcs[0].Instrs {
					if p.Funcs[0].Instrs[i].Kind == ir.IRIndexLoadI32Unchecked {
						p.Funcs[0].Instrs[i].Kind = ir.IRIndexLoadI32
						p.Funcs[0].Instrs[i].ProofID = ""
					}
				}
			},
			reason: "missing_while_bounds_proof",
		},
		{
			name: "call in loop",
			mutate: func(p *ir.IRProgram) {
				insertAt := 9
				p.Funcs[0].Instrs = append(
					p.Funcs[0].Instrs[:insertAt],
					append(
						[]ir.IRInstr{{Kind: ir.IRCall, Name: "touch", ArgSlots: 0, RetSlots: 0}},
						p.Funcs[0].Instrs[insertAt:]...)...)
			},
			reason: "loop_has_unknown_mutation",
		},
		{
			name: "len store in loop",
			mutate: func(p *ir.IRProgram) {
				insertAt := 9
				p.Funcs[0].Instrs = append(
					p.Funcs[0].Instrs[:insertAt],
					append(
						[]ir.IRInstr{
							{Kind: ir.IRConstI32, Imm: 9},
							{Kind: ir.IRStoreLocal, Local: 2},
						},
						p.Funcs[0].Instrs[insertAt:]...)...)
			},
			reason: "loop_stores_len_local",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog := loopCanonicalizationProgram(ir.IRCmpLtI32, true)
			tc.mutate(prog)
			report, err := NewManager().Run(prog, LoopCanonicalizationPass())
			if err != nil {
				t.Fatalf("Run LoopCanonicalizationPass: %v", err)
			}
			after := report.Passes[0].AfterDump
			if strings.Contains(after, "store_local local:4") {
				t.Fatalf("unsafe loop was hoisted:\n%s", after)
			}
			if !hasDecision(report.Passes[0].Decisions, "not_hoisted", tc.reason) {
				t.Fatalf("decisions missing %q: %#v", tc.reason, report.Passes[0].Decisions)
			}
		})
	}
}

func loopCanonicalizationProgram(cmp ir.IRInstrKind, withProof bool) *ir.IRProgram {
	instrs := []ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRStoreLocal, Local: 0},
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRStoreLocal, Local: 3},
		{Kind: ir.IRLabel, Label: 1},
		{Kind: ir.IRLoadLocal, Local: 0},
		{Kind: ir.IRLoadLocal, Local: 2},
	}
	if cmp == ir.IRCmpLeI32 {
		instrs = append(
			instrs,
			ir.IRInstr{Kind: ir.IRConstI32, Imm: 1},
			ir.IRInstr{Kind: ir.IRSubI32},
		)
	}
	instrs = append(instrs,
		ir.IRInstr{Kind: cmp},
		ir.IRInstr{Kind: ir.IRJmpIfZero, Label: 2},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 2},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRIndexLoadI32Unchecked, ProofID: proofID(withProof)},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 3},
		ir.IRInstr{Kind: ir.IRAddI32},
		ir.IRInstr{Kind: ir.IRStoreLocal, Local: 3},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 1},
		ir.IRInstr{Kind: ir.IRAddI32},
		ir.IRInstr{Kind: ir.IRStoreLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRJmp, Label: 1},
		ir.IRInstr{Kind: ir.IRLabel, Label: 2},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 3},
		ir.IRInstr{Kind: ir.IRReturn},
	)
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  3,
			LocalSlots:  4,
			ReturnSlots: 1,
			Instrs:      instrs,
		}},
	}
}

func proofID(enabled bool) string {
	if !enabled {
		return ""
	}
	return "proof:while:i:xs:1:1"
}

func hasDecision(decisions []PassDecision, action string, reason string) bool {
	for _, decision := range decisions {
		if decision.Action == action && decision.Reason == reason {
			return true
		}
	}
	return false
}

// ---- manager_test.go ----

func TestManagerVerifiesBeforeAndAfterPass(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
	manager := NewManager()
	report, err := manager.Run(prog, p17ContractTestPass("noop"))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(report.Passes) != 1 || report.Passes[0].Name != "noop" ||
		!report.Passes[0].VerifiedInput ||
		!report.Passes[0].VerifiedOutput ||
		!report.Passes[0].VerifiedProofs {
		t.Fatalf("report = %#v", report)
	}
	row := report.Passes[0]
	if row.InputKind != IRKindStack || row.OutputKind != IRKindStack ||
		row.ValidationStrategy != ValidationTranslation ||
		row.ReportOutput != "noop.opt.json" {
		t.Fatalf("metadata row = %#v", row)
	}
	for _, want := range []string{"func main", "const_i32 7", "return"} {
		if !strings.Contains(row.BeforeDump, want) || !strings.Contains(row.AfterDump, want) {
			t.Fatalf(
				"before/after dumps missing %q:\nbefore:\n%s\nafter:\n%s",
				want,
				row.BeforeDump,
				row.AfterDump,
			)
		}
	}
}

func TestManagerRejectsPassThatProducesInvalidIR(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
	manager := NewManager()
	pass := p17ContractTestPass("break-return")
	pass.Run = func(ctx *PassContext) error {
		p := ctx.Program
		p.Funcs[0].Instrs = p.Funcs[0].Instrs[:1]
		return nil
	}
	_, err := manager.Run(prog, pass)
	if err == nil || !strings.Contains(err.Error(), "break-return output verification failed") {
		t.Fatalf("Run error = %v", err)
	}
}

func TestManagerCanRunOnePassByNameForTests(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRReturn},
			},
		}},
	}
	manager := NewManager()
	ran := []string{}
	first := p17ContractTestPass("first")
	first.Run = func(ctx *PassContext) error {
		ran = append(ran, "first")
		return nil
	}
	second := p17ContractTestPass("second")
	second.Run = func(ctx *PassContext) error {
		ran = append(ran, "second")
		return nil
	}
	report, err := manager.RunWithOptions(prog, Options{OnlyPass: "second"}, first, second)
	if err != nil {
		t.Fatalf("RunWithOptions: %v", err)
	}
	if strings.Join(ran, ",") != "second" {
		t.Fatalf("ran passes = %v, want only second", ran)
	}
	if len(report.Passes) != 1 || report.Passes[0].Name != "second" {
		t.Fatalf("report passes = %#v, want only second", report.Passes)
	}
}

func TestManagerRejectsMissingPassMetadata(t *testing.T) {
	prog := validTinyProgram()
	manager := NewManager()
	_, err := manager.Run(prog, Pass{
		Name: "nameless-metadata",
		Run:  func(ctx *PassContext) error { return nil },
	})
	if err == nil || !strings.Contains(err.Error(), "missing input IR kind") {
		t.Fatalf("Run error = %v, want metadata rejection", err)
	}
}

func TestManagerRunsTranslationValidationStrategy(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{
			{
				Name:        "main",
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 1},
					{Kind: ir.IRReturn},
				},
			},
			{
				Name:        "helper",
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 2},
					{Kind: ir.IRReturn},
				},
			},
		},
	}
	manager := NewManager()
	pass := p17ContractTestPass("bad-delete-helper")
	pass.Run = func(ctx *PassContext) error {
		p := ctx.Program
		p.Funcs = p.Funcs[:1]
		return nil
	}
	_, err := manager.Run(prog, pass)
	if err == nil || !strings.Contains(err.Error(), "translation validation failed") {
		t.Fatalf("Run error = %v, want translation validation failure", err)
	}
}

func TestManagerRejectsSemanticChangingTranslationPass(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRReturn},
			},
		}},
	}
	pass := p17ContractTestPass("bad-constant-fold")
	pass.Run = func(ctx *PassContext) error {
		p := ctx.Program
		p.Funcs[0].Instrs[0].Imm = 2
		return nil
	}
	_, err := NewManager().Run(prog, pass)
	if err == nil || !strings.Contains(err.Error(), "semantic local equivalence") {
		t.Fatalf("Run error = %v, want semantic translation validation failure", err)
	}
}

func TestManagerIncludesTranslationReportEvidence(t *testing.T) {
	prog := validTinyProgram()
	report, err := NewManager().Run(prog, p17ContractTestPass("noop-translation"))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	row := report.Passes[0]
	if !row.TranslationValidated || row.TranslationReport == nil {
		t.Fatalf("translation report evidence missing: %#v", row)
	}
	if row.TranslationReport.FunctionsCompared != 1 ||
		row.TranslationReport.SemanticLocalChecks != 1 {
		t.Fatalf(
			"translation report = %+v, want function and semantic evidence",
			row.TranslationReport,
		)
	}
	if row.ValidationMetadata == nil {
		t.Fatalf("validation metadata evidence missing: %#v", row)
	}
	if row.ValidationMetadata.SchemaVersion != "tetra.translation.validation.metadata.v1" ||
		row.ValidationMetadata.BeforeHash == "" ||
		row.ValidationMetadata.AfterHash == "" {
		t.Fatalf("validation metadata = %+v", row.ValidationMetadata)
	}
}

func TestManagerAcceptsProfileInputAsValidatedMetadataWithoutChangingIR(t *testing.T) {
	prog := validTinyProgram()
	before := FormatProgram(prog)
	profile := ProfileCollection{
		SchemaVersion: ProfileCollectionSchemaVersion,
		ProgramHash:   "sha256:managerprofile",
		TargetTriple:  "linux-x64",
		Functions: []ProfileFunction{{
			ID:         "fn:main",
			Name:       "main",
			EntryCount: 99,
			Counters: []ProfileCounter{
				{Kind: "edge", Name: "return", Count: 99},
			},
		}},
	}

	report, err := NewManager().RunWithOptions(
		prog,
		Options{ProfileInput: &profile},
		p17ContractTestPass("noop-profile-input"),
	)
	if err != nil {
		t.Fatalf("RunWithOptions: %v", err)
	}
	after := FormatProgram(prog)
	if before != after {
		t.Fatalf("profile input changed IR:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	row := report.Passes[0]
	if row.ProfileInputPolicy != ProfileInputUnused {
		t.Fatalf("profile policy = %q, want %q", row.ProfileInputPolicy, ProfileInputUnused)
	}
	if row.ProfileInput == nil {
		t.Fatalf("profile input evidence missing: %#v", row)
	}
	if row.ProfileInput.SchemaVersion != ProfileCollectionSchemaVersion ||
		row.ProfileInput.ProgramHash != "sha256:managerprofile" ||
		row.ProfileInput.Functions != 1 ||
		row.ProfileInput.TotalEntryCount != 99 {
		t.Fatalf("profile input evidence = %+v", row.ProfileInput)
	}
	if !strings.HasPrefix(row.ProfileInput.Digest, "sha256:") {
		t.Fatalf("profile digest = %q, want sha256", row.ProfileInput.Digest)
	}
	if !containsString(row.ProfileInput.CounterKinds, "edge") {
		t.Fatalf("profile counter kinds = %#v, want edge", row.ProfileInput.CounterKinds)
	}
	if row.ValidationMetadata == nil {
		t.Fatalf("validation metadata missing: %#v", row)
	}
	if row.ValidationMetadata.ProfileInputPolicy != string(ProfileInputUnused) ||
		row.ValidationMetadata.ProfileInputDigest != row.ProfileInput.Digest {
		t.Fatalf(
			"validation profile metadata mismatch: row=%#v metadata=%+v",
			row.ProfileInput,
			row.ValidationMetadata,
		)
	}
}

func TestManagerRejectsProfileGuidedRewritePolicyUntilValidationExists(t *testing.T) {
	pass := p17ContractTestPass("bad-profile-guided-rewrite")
	pass.ProfileInputPolicy = ProfileInputGuidedRewrite
	_, err := NewManager().Run(validTinyProgram(), pass)
	if err == nil ||
		!strings.Contains(
			err.Error(),
			"profile-guided optimizer decisions require dedicated validation",
		) {
		t.Fatalf("Run error = %v, want profile-guided validation rejection", err)
	}
}

func TestRegisteredOptimizerPassesExposeP17ContractEvidence(t *testing.T) {
	passes := RegisteredPasses()
	wantNames := map[string]bool{
		"basic-scalar":              false,
		"inline-small-pure":         false,
		"licm-pure-invariant":       false,
		"loop-canonicalization":     false,
		"mem2reg-single-assignment": false,
		"sccp-constant-branch":      false,
	}
	if len(passes) != len(wantNames) {
		t.Fatalf("registered passes = %d, want %d: %#v", len(passes), len(wantNames), passes)
	}
	for _, pass := range passes {
		if _, ok := wantNames[pass.Name]; !ok {
			t.Fatalf("unexpected registered pass %q", pass.Name)
		}
		wantNames[pass.Name] = true
		if err := ValidatePassContract(pass); err != nil {
			t.Fatalf("registered pass %q contract invalid: %v", pass.Name, err)
		}
	}
	for name, seen := range wantNames {
		if !seen {
			t.Fatalf("registered pass %q missing", name)
		}
	}

	report, err := NewManager().Run(validTinyProgram(), passes...)
	if err != nil {
		t.Fatalf("Run registered passes: %v", err)
	}
	if len(report.Passes) != len(passes) {
		t.Fatalf("report passes = %d, want %d", len(report.Passes), len(passes))
	}
	for _, row := range report.Passes {
		if row.InputVerifier != VerifierLowerVerifyProgram ||
			row.OutputVerifier != VerifierLowerVerifyProgram {
			t.Fatalf("%s verifier evidence missing: %#v", row.Name, row)
		}
		if row.ProofRule != ProofRulePreserveBoundsInvalidateLiveness {
			t.Fatalf(
				"%s proof rule = %q, want %q",
				row.Name,
				row.ProofRule,
				ProofRulePreserveBoundsInvalidateLiveness,
			)
		}
		if row.TranslationValidationHook != TranslationHookValidateTranslation ||
			!row.TranslationValidated {
			t.Fatalf("%s translation hook evidence missing: %#v", row.Name, row)
		}
		if row.ProfileInputPolicy != ProfileInputUnused {
			t.Fatalf(
				"%s profile input policy = %q, want %q",
				row.Name,
				row.ProfileInputPolicy,
				ProfileInputUnused,
			)
		}
		for _, want := range RequiredP17ReportRows() {
			if !containsString(row.ReportRows, want) {
				t.Fatalf("%s report rows missing %q: %#v", row.Name, want, row.ReportRows)
			}
		}
		if row.NegativeTestMarker != NegativeTestPassContractV1 {
			t.Fatalf(
				"%s negative-test marker = %q, want %q",
				row.Name,
				row.NegativeTestMarker,
				NegativeTestPassContractV1,
			)
		}
		if row.ValidationMetadata == nil {
			t.Fatalf("%s missing validation metadata", row.Name)
		}
		if row.ValidationMetadata.InputVerifier != row.InputVerifier ||
			row.ValidationMetadata.OutputVerifier != row.OutputVerifier {
			t.Fatalf(
				"%s validation metadata verifier mismatch: row=%#v metadata=%+v",
				row.Name,
				row,
				row.ValidationMetadata,
			)
		}
		if row.ValidationMetadata.ProofRule != string(row.ProofRule) ||
			row.ValidationMetadata.TranslationValidationHook != row.TranslationValidationHook {
			t.Fatalf(
				"%s validation metadata contract mismatch: row=%#v metadata=%+v",
				row.Name,
				row,
				row.ValidationMetadata,
			)
		}
		if row.ValidationMetadata.ProfileInputPolicy != string(ProfileInputUnused) {
			t.Fatalf("%s validation profile metadata = %+v", row.Name, row.ValidationMetadata)
		}
	}
}

func TestManagerRejectsIncompletePassContractEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Pass)
		want   string
	}{
		{
			name:   "missing input verifier",
			mutate: func(pass *Pass) { pass.InputVerifier = "" },
			want:   "missing input verifier",
		},
		{
			name:   "fake input verifier",
			mutate: func(pass *Pass) { pass.InputVerifier = "paper.input.Verifier" },
			want:   "unsupported input verifier",
		},
		{
			name:   "missing output verifier",
			mutate: func(pass *Pass) { pass.OutputVerifier = "" },
			want:   "missing output verifier",
		},
		{
			name:   "fake output verifier",
			mutate: func(pass *Pass) { pass.OutputVerifier = "paper.output.Verifier" },
			want:   "unsupported output verifier",
		},
		{
			name:   "missing proof rule",
			mutate: func(pass *Pass) { pass.ProofRule = "" },
			want:   "missing proof preservation or invalidation rule",
		},
		{
			name:   "fake proof rule",
			mutate: func(pass *Pass) { pass.ProofRule = "trust_me_preserved" },
			want:   "unknown proof preservation or invalidation rule",
		},
		{
			name:   "missing translation hook",
			mutate: func(pass *Pass) { pass.TranslationValidationHook = "" },
			want:   "missing translation validation hook",
		},
		{
			name:   "fake translation hook",
			mutate: func(pass *Pass) { pass.TranslationValidationHook = "paper.translation.Hook" },
			want:   "unsupported translation validation hook",
		},
		{
			name:   "missing report rows",
			mutate: func(pass *Pass) { pass.ReportRows = nil },
			want:   "missing report rows",
		},
		{
			name:   "missing required report row",
			mutate: func(pass *Pass) { pass.ReportRows = []string{"before_dump", "after_dump"} },
			want:   "missing required report row",
		},
		{
			name:   "missing profile input policy",
			mutate: func(pass *Pass) { pass.ProfileInputPolicy = "" },
			want:   "missing profile input policy",
		},
		{
			name:   "unsupported profile guided policy",
			mutate: func(pass *Pass) { pass.ProfileInputPolicy = ProfileInputGuidedRewrite },
			want:   "profile-guided optimizer decisions require dedicated validation",
		},
		{
			name:   "missing negative-test marker",
			mutate: func(pass *Pass) { pass.NegativeTestMarker = "" },
			want:   "missing negative-test marker",
		},
		{
			name:   "fake negative-test marker",
			mutate: func(pass *Pass) { pass.NegativeTestMarker = "paper-negative-tests" },
			want:   "unknown negative-test marker",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pass := p17ContractTestPass("contracted-noop")
			tc.mutate(&pass)
			_, err := NewManager().Run(validTinyProgram(), pass)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Run error = %v, want %q", err, tc.want)
			}
		})
	}
}

func p17ContractTestPass(name string) Pass {
	return Pass{
		Name:                      name,
		InputKind:                 IRKindStack,
		OutputKind:                IRKindStack,
		InputVerifier:             VerifierLowerVerifyProgram,
		OutputVerifier:            VerifierLowerVerifyProgram,
		RequiredFacts:             []Fact{FactIRVerified},
		PreservedFacts:            []Fact{FactBoundsProofs},
		InvalidatedFacts:          []Fact{FactLiveness},
		PreservedProofKinds:       []memoryfacts.ProofKind{memoryfacts.ProofBounds},
		ProofRule:                 ProofRulePreserveBoundsInvalidateLiveness,
		ValidationStrategy:        ValidationTranslation,
		TranslationValidationHook: TranslationHookValidateTranslation,
		ReportOutput:              name + ".opt.json",
		ReportRows:                RequiredP17ReportRows(),
		NegativeTestMarker:        NegativeTestPassContractV1,
		ProfileInputPolicy:        ProfileInputUnused,
		Run:                       func(ctx *PassContext) error { return nil },
	}
}

func validTinyProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// ---- mem2reg_test.go ----

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
	if row.Name != "mem2reg-single-assignment" || !row.TranslationValidated ||
		row.ValidationMetadata == nil {
		t.Fatalf("metadata row = %#v", row)
	}
	for _, want := range []string{"add_i32", "store_local local:0", "load_local local:0", "mul_i32"} {
		if !strings.Contains(row.BeforeDump, want) {
			t.Fatalf("before dump missing %q:\n%s", want, row.BeforeDump)
		}
	}
	after := row.AfterDump
	for _, want := range []string{
		"const_i32 4",
		"const_i32 5",
		"add_i32",
		"const_i32 2",
		"mul_i32",
		"return",
	} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	for _, forbidden := range []string{"store_local local:0", "load_local local:0"} {
		if strings.Contains(after, forbidden) {
			t.Fatalf("after dump still contains promoted temp %q:\n%s", forbidden, after)
		}
	}
	if !hasDecision(
		row.Decisions,
		"promoted_single_assignment_temp",
		"single_store_single_load_adjacent",
	) {
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
	for _, want := range []string{
		"const_i32 4",
		"const_i32 9",
		"store_local local:2",
		"const_i32 2",
		"mul_i32",
		"return",
	} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	for _, forbidden := range []string{"store_local local:0", "load_local local:0"} {
		if strings.Contains(after, forbidden) {
			t.Fatalf("after dump still contains promoted separated temp %q:\n%s", forbidden, after)
		}
	}
	if !hasDecision(
		row.Decisions,
		"promoted_single_assignment_temp",
		"single_store_single_load_stack_neutral",
	) {
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
	for _, want := range []string{
		"load_local local:0",
		"const_i32 7",
		"cmp_lt_i32",
		"const_i32 3",
		"store_local local:2",
		"const_i32 2",
		"add_i32",
		"store_local local:0",
		"return",
	} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	for _, forbidden := range []string{"store_local local:1", "load_local local:1"} {
		if strings.Contains(after, forbidden) {
			t.Fatalf("after dump still contains promoted comparison temp %q:\n%s", forbidden, after)
		}
	}
	if !hasDecision(
		row.Decisions,
		"promoted_single_assignment_temp",
		"single_store_single_load_stack_neutral_comparison_expression",
	) {
		t.Fatalf(
			"decisions = %#v, want stack-neutral comparison-expression promotion",
			row.Decisions,
		)
	}

	beforeExit := runOptLinuxX64(t, before.Funcs, "before-separated-comparison-mem2reg")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-separated-comparison-mem2reg")
	if beforeExit != afterExit || afterExit != 3 {
		t.Fatalf("exit mismatch before=%d after=%d, want both 3", beforeExit, afterExit)
	}
}

func TestMem2RegPassPromotesSeparatedSafeConstDenominatorDivModTempWithStackNeutralWork(
	t *testing.T,
) {
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
			for _, want := range []string{"load_local local:0", "const_i32 " + strconv.FormatInt(
				int64(tc.denom),
				10,
			), tc.op, "const_i32 7", "store_local local:2", "const_i32 2", "add_i32", "store_local local:0", "return"} {
				if !strings.Contains(after, want) {
					t.Fatalf("after dump missing %q:\n%s", want, after)
				}
			}
			for _, forbidden := range []string{"store_local local:1", "load_local local:1"} {
				if strings.Contains(after, forbidden) {
					t.Fatalf(
						"after dump still contains promoted div/mod temp %q:\n%s",
						forbidden,
						after,
					)
				}
			}
			if !hasDecision(
				row.Decisions,
				"promoted_single_assignment_temp",
				"single_store_single_load_stack_neutral_safe_const_denominator_divmod_expression",
			) {
				t.Fatalf(
					"decisions = %#v, want stack-neutral safe div/mod expression promotion",
					row.Decisions,
				)
			}

			beforeExit := runOptLinuxX64(
				t,
				before.Funcs,
				"before-separated-safe-divmod-mem2reg-"+tc.name,
			)
			afterExit := runOptLinuxX64(
				t,
				prog.Funcs,
				"after-separated-safe-divmod-mem2reg-"+tc.name,
			)
			if beforeExit != afterExit || afterExit != tc.wantExit {
				t.Fatalf(
					"exit mismatch before=%d after=%d, want both %d",
					beforeExit,
					afterExit,
					tc.wantExit,
				)
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
					t.Fatalf(
						"after dump still contains promoted known-local div/mod temp %q:\n%s",
						forbidden,
						after,
					)
				}
			}
			if !hasDecision(
				row.Decisions,
				"promoted_single_assignment_temp",
				"single_store_single_load_stack_neutral_safe_known_local_divmod_expression",
			) {
				t.Fatalf(
					"decisions = %#v, want stack-neutral safe known-local div/mod expression promotion",
					row.Decisions,
				)
			}

			beforeExit := runOptLinuxX64(
				t,
				before.Funcs,
				"before-separated-safe-known-local-divmod-mem2reg-"+tc.name,
			)
			afterExit := runOptLinuxX64(
				t,
				prog.Funcs,
				"after-separated-safe-known-local-divmod-mem2reg-"+tc.name,
			)
			if beforeExit != afterExit || afterExit != tc.wantExit {
				t.Fatalf(
					"exit mismatch before=%d after=%d, want both %d",
					beforeExit,
					afterExit,
					tc.wantExit,
				)
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
					t.Fatalf(
						"after dump still contains promoted arithmetic temp %q:\n%s",
						forbidden,
						after,
					)
				}
			}
			if !hasDecision(
				row.Decisions,
				"promoted_single_assignment_temp",
				"single_store_single_load_stack_neutral_safe_const_arithmetic_expression",
			) {
				t.Fatalf(
					"decisions = %#v, want stack-neutral safe const arithmetic expression promotion",
					row.Decisions,
				)
			}

			beforeExit := runOptLinuxX64(
				t,
				before.Funcs,
				"before-separated-safe-arithmetic-mem2reg-"+tc.name,
			)
			afterExit := runOptLinuxX64(
				t,
				prog.Funcs,
				"after-separated-safe-arithmetic-mem2reg-"+tc.name,
			)
			if beforeExit != afterExit || afterExit != tc.wantExit {
				t.Fatalf(
					"exit mismatch before=%d after=%d, want both %d",
					beforeExit,
					afterExit,
					tc.wantExit,
				)
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
		{
			name:  "subtraction overflow",
			kind:  ir.IRSubI32,
			op:    "sub_i32",
			left:  -2147483648,
			right: 1,
		},
		{
			name:  "multiplication overflow",
			kind:  ir.IRMulI32,
			op:    "mul_i32",
			left:  50000,
			right: 50000,
		},
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
				t.Fatalf(
					"unsafe const arithmetic temp changed unexpectedly:\nbefore:\n%s\nafter:\n%s",
					beforeDump,
					row.AfterDump,
				)
			}
			for _, want := range []string{"store_local local:0", "load_local local:0"} {
				if !strings.Contains(row.AfterDump, want) {
					t.Fatalf(
						"after dump missing preserved unsafe arithmetic temp %q:\n%s",
						want,
						row.AfterDump,
					)
				}
			}
			if got := countDumpOccurrences(row.AfterDump, tc.op); got != 1 {
				t.Fatalf("%s count after = %d, want 1:\n%s", tc.op, got, row.AfterDump)
			}
			if !hasDecision(row.Decisions, "not_promoted", "producer_not_available") {
				t.Fatalf(
					"decisions = %#v, want unsafe const arithmetic producer rejection",
					row.Decisions,
				)
			}
		})
	}
}

func TestMem2RegPassPromotesSeparatedSafeKnownLocalArithmeticTempWithStackNeutralWork(
	t *testing.T,
) {
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
					t.Fatalf(
						"after dump still contains promoted known-local arithmetic temp %q:\n%s",
						forbidden,
						after,
					)
				}
			}
			if !hasDecision(
				row.Decisions,
				"promoted_single_assignment_temp",
				"single_store_single_load_stack_neutral_safe_known_local_arithmetic_expression",
			) {
				t.Fatalf(
					"decisions = %#v, want stack-neutral safe known-local arithmetic expression promotion",
					row.Decisions,
				)
			}

			beforeExit := runOptLinuxX64(
				t,
				before.Funcs,
				"before-separated-safe-known-local-arithmetic-mem2reg-"+tc.name,
			)
			afterExit := runOptLinuxX64(
				t,
				prog.Funcs,
				"after-separated-safe-known-local-arithmetic-mem2reg-"+tc.name,
			)
			if beforeExit != afterExit || afterExit != tc.wantExit {
				t.Fatalf(
					"exit mismatch before=%d after=%d, want both %d",
					beforeExit,
					afterExit,
					tc.wantExit,
				)
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
		{
			name:  "subtraction overflow",
			kind:  ir.IRSubI32,
			op:    "sub_i32",
			left:  -2147483648,
			right: 1,
		},
		{
			name:  "multiplication overflow",
			kind:  ir.IRMulI32,
			op:    "mul_i32",
			left:  50000,
			right: 50000,
		},
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
				t.Fatalf(
					"unsafe known-local arithmetic temp changed unexpectedly:\nbefore:\n%s\nafter:\n%s",
					beforeDump,
					row.AfterDump,
				)
			}
			for _, want := range []string{"store_local local:2", "load_local local:2"} {
				if !strings.Contains(row.AfterDump, want) {
					t.Fatalf(
						"after dump missing preserved unsafe known-local arithmetic temp %q:\n%s",
						want,
						row.AfterDump,
					)
				}
			}
			if got := countDumpOccurrences(row.AfterDump, tc.op); got != 1 {
				t.Fatalf("%s count after = %d, want 1:\n%s", tc.op, got, row.AfterDump)
			}
			if !hasDecision(row.Decisions, "not_promoted", "producer_not_available") {
				t.Fatalf(
					"decisions = %#v, want unsafe known-local arithmetic producer rejection",
					row.Decisions,
				)
			}
		})
	}
}

func TestMem2RegPassRejectsSeparatedSafeKnownLocalArithmeticTempWhenSourceLocalMutates(
	t *testing.T,
) {
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
		t.Fatalf(
			("mutating safe known-local arithmetic source local changed " +
				"unexpectedly:\nbefore:\n%s\nafter:\n%s"),
			beforeDump,
			row.AfterDump,
		)
	}
	if !hasDecision(row.Decisions, "not_promoted", "source_local_modified_before_load") {
		t.Fatalf(
			"decisions = %#v, want explicit safe known-local arithmetic source-local mutation rejection",
			row.Decisions,
		)
	}

	beforeExit := runOptLinuxX64(
		t,
		before.Funcs,
		"before-safe-known-local-arithmetic-source-mutates-mem2reg",
	)
	afterExit := runOptLinuxX64(
		t,
		prog.Funcs,
		"after-safe-known-local-arithmetic-source-mutates-mem2reg",
	)
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
				t.Fatalf(
					"unsafe denominator div/mod temp changed unexpectedly:\nbefore:\n%s\nafter:\n%s",
					beforeDump,
					row.AfterDump,
				)
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
				t.Fatalf(
					"decisions = %#v, want unsafe denominator producer rejection",
					row.Decisions,
				)
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
				t.Fatalf(
					"unsafe known-local div/mod temp changed unexpectedly:\nbefore:\n%s\nafter:\n%s",
					beforeDump,
					row.AfterDump,
				)
			}
			for _, want := range []string{"store_local local:2", "load_local local:2"} {
				if !strings.Contains(row.AfterDump, want) {
					t.Fatalf(
						"after dump missing preserved unsafe known-local div/mod temp %q:\n%s",
						want,
						row.AfterDump,
					)
				}
			}
			if got := countDumpOccurrences(row.AfterDump, tc.op); got != 1 {
				t.Fatalf("%s count after = %d, want 1:\n%s", tc.op, got, row.AfterDump)
			}
			if !hasDecision(row.Decisions, "not_promoted", "producer_not_available") {
				t.Fatalf(
					"decisions = %#v, want unsafe known-local div/mod producer rejection",
					row.Decisions,
				)
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
	for _, want := range []string{
		"const_i32 -6",
		"neg_i32",
		"const_i32 7",
		"store_local local:1",
		"const_i32 2",
		"add_i32",
		"return",
	} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	for _, forbidden := range []string{"store_local local:0", "load_local local:0"} {
		if strings.Contains(after, forbidden) {
			t.Fatalf("after dump still contains promoted unary neg temp %q:\n%s", forbidden, after)
		}
	}
	if !hasDecision(
		row.Decisions,
		"promoted_single_assignment_temp",
		"single_store_single_load_stack_neutral_safe_const_unary_neg_expression",
	) {
		t.Fatalf(
			"decisions = %#v, want stack-neutral safe const unary neg expression promotion",
			row.Decisions,
		)
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
		t.Fatalf(
			"unsafe unary neg temp changed unexpectedly:\nbefore:\n%s\nafter:\n%s",
			beforeDump,
			row.AfterDump,
		)
	}
	for _, want := range []string{
		"const_i32 -2147483648",
		"neg_i32",
		"store_local local:0",
		"load_local local:0",
	} {
		if !strings.Contains(row.AfterDump, want) {
			t.Fatalf(
				"after dump missing preserved unsafe unary neg temp %q:\n%s",
				want,
				row.AfterDump,
			)
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
	for _, want := range []string{
		"load_local local:0",
		"neg_i32",
		"const_i32 7",
		"store_local local:2",
		"const_i32 2",
		"add_i32",
		"return",
	} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	for _, forbidden := range []string{"store_local local:1", "load_local local:1"} {
		if strings.Contains(after, forbidden) {
			t.Fatalf(
				"after dump still contains promoted known-local unary neg temp %q:\n%s",
				forbidden,
				after,
			)
		}
	}
	if !hasDecision(
		row.Decisions,
		"promoted_single_assignment_temp",
		"single_store_single_load_stack_neutral_safe_known_local_unary_neg_expression",
	) {
		t.Fatalf(
			"decisions = %#v, want stack-neutral safe known-local unary neg expression promotion",
			row.Decisions,
		)
	}

	beforeExit := runOptLinuxX64(
		t,
		before.Funcs,
		"before-separated-safe-known-local-unary-neg-mem2reg",
	)
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
		t.Fatalf(
			"unsafe known-local unary neg temp changed unexpectedly:\nbefore:\n%s\nafter:\n%s",
			beforeDump,
			row.AfterDump,
		)
	}
	for _, want := range []string{
		"const_i32 -2147483648",
		"load_local local:0",
		"neg_i32",
		"store_local local:1",
		"load_local local:1",
	} {
		if !strings.Contains(row.AfterDump, want) {
			t.Fatalf(
				"after dump missing preserved unsafe known-local unary neg temp %q:\n%s",
				want,
				row.AfterDump,
			)
		}
	}
	if !hasDecision(row.Decisions, "not_promoted", "producer_not_available") {
		t.Fatalf(
			"decisions = %#v, want unsafe known-local unary neg producer rejection",
			row.Decisions,
		)
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
		t.Fatalf(
			("mutating safe known-local unary neg source local changed " +
				"unexpectedly:\nbefore:\n%s\nafter:\n%s"),
			beforeDump,
			row.AfterDump,
		)
	}
	if !hasDecision(row.Decisions, "not_promoted", "source_local_modified_before_load") {
		t.Fatalf(
			"decisions = %#v, want explicit safe known-local unary neg source-local mutation rejection",
			row.Decisions,
		)
	}

	beforeExit := runOptLinuxX64(
		t,
		before.Funcs,
		"before-safe-known-local-unary-neg-source-mutates-mem2reg",
	)
	afterExit := runOptLinuxX64(
		t,
		prog.Funcs,
		"after-safe-known-local-unary-neg-source-mutates-mem2reg",
	)
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
		t.Fatalf(
			"mutating source local changed unexpectedly:\nbefore:\n%s\nafter:\n%s",
			beforeDump,
			row.AfterDump,
		)
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
		t.Fatalf(
			"mutating safe div/mod source local changed unexpectedly:\nbefore:\n%s\nafter:\n%s",
			beforeDump,
			row.AfterDump,
		)
	}
	if !hasDecision(row.Decisions, "not_promoted", "source_local_modified_before_load") {
		t.Fatalf(
			"decisions = %#v, want explicit safe div/mod source-local mutation rejection",
			row.Decisions,
		)
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
		t.Fatalf(
			"mutating safe known-local div/mod source local changed unexpectedly:\nbefore:\n%s\nafter:\n%s",
			beforeDump,
			row.AfterDump,
		)
	}
	if !hasDecision(row.Decisions, "not_promoted", "source_local_modified_before_load") {
		t.Fatalf(
			"decisions = %#v, want explicit safe known-local div/mod source-local mutation rejection",
			row.Decisions,
		)
	}

	beforeExit := runOptLinuxX64(
		t,
		before.Funcs,
		"before-safe-known-local-divmod-source-mutates-mem2reg",
	)
	afterExit := runOptLinuxX64(
		t,
		prog.Funcs,
		"after-safe-known-local-divmod-source-mutates-mem2reg",
	)
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
		t.Fatalf(
			"mutating comparison source local changed unexpectedly:\nbefore:\n%s\nafter:\n%s",
			beforeDump,
			row.AfterDump,
		)
	}
	if !hasDecision(row.Decisions, "not_promoted", "source_local_modified_before_load") {
		t.Fatalf(
			"decisions = %#v, want explicit comparison source-local mutation rejection",
			row.Decisions,
		)
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
		t.Fatalf(
			"multi-load local changed unexpectedly:\nbefore:\n%s\nafter:\n%s",
			before,
			row.AfterDump,
		)
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

// ---- pgo_lto_test.go ----

func TestPGOLTOTargetCPUCoverageAuditsP17PlanList(t *testing.T) {
	report, err := PGOLTOTargetCPUCoverage()
	if err != nil {
		t.Fatalf("PGOLTOTargetCPUCoverage: %v", err)
	}
	if report.SchemaVersion != "tetra.optimizer.pgo_lto_target_cpu.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	if !containsString(
		report.NonClaims,
		"no PGO, LTO, target-cpu, or profile flag changes safe-program semantics",
	) {
		t.Fatalf("non-claims = %#v, want explicit safe-semantics non-claim", report.NonClaims)
	}

	want := []PGOLTOTargetCPUID{
		PGOLTOTargetCPUProfileCollectionFormat,
		PGOLTOTargetCPUPGOOptimizerInput,
		PGOLTOTargetCPUTargetCPUFeatureDetection,
		PGOLTOTargetCPULTOIncrementalModuleSummary,
		PGOLTOTargetCPUSafeSemanticsFlags,
	}
	if len(report.Rows) != len(want) {
		t.Fatalf("coverage rows = %d, want %d: %#v", len(report.Rows), len(want), report.Rows)
	}
	byID := map[PGOLTOTargetCPUID]PGOLTOTargetCPUCoverageRow{}
	for _, row := range report.Rows {
		byID[row.ID] = row
		if row.Name == "" || row.Status == "" || row.Reason == "" || row.Evidence == "" ||
			row.Boundary == "" {
			t.Fatalf("row missing required P17.4 evidence: %#v", row)
		}
		if row.ChangesSafeSemantics {
			t.Fatalf("P17.4 row changes safe semantics: %#v", row)
		}
	}
	for _, id := range want {
		if _, ok := byID[id]; !ok {
			t.Fatalf("missing P17.4 row %s", id)
		}
	}

	profile := byID[PGOLTOTargetCPUProfileCollectionFormat]
	if profile.Status != PGOLTOTargetCPUImplementedNarrow {
		t.Fatalf("profile format row = %#v, want implemented_narrow", profile)
	}
	if profile.OptimizerInput {
		t.Fatalf("profile format row must be inert evidence, not optimizer input: %#v", profile)
	}
	for _, want := range []string{
		"tetra.optimizer.profile.v1",
		"canonical JSON",
		"duplicate",
		"negative counter",
		"inert",
	} {
		if !strings.Contains(profile.Reason+" "+profile.Evidence+" "+profile.Boundary, want) {
			t.Fatalf("profile format row missing %q: %#v", want, profile)
		}
	}
	for _, want := range []string{
		"schema_validation",
		"canonical_json",
		"duplicate_rejection",
		"negative_counter_rejection",
	} {
		if !containsString(profile.RequiredFacts, want) {
			t.Fatalf("profile format row missing required fact %q: %#v", want, profile)
		}
	}

	pgo := byID[PGOLTOTargetCPUPGOOptimizerInput]
	if pgo.Status != PGOLTOTargetCPUImplementedNarrow {
		t.Fatalf("pgo_optimizer_input row = %#v, want implemented_narrow", pgo)
	}
	if !pgo.OptimizerInput {
		t.Fatalf("pgo_optimizer_input row should record optimizer input evidence: %#v", pgo)
	}
	if len(pgo.MissingFacts) != 0 {
		t.Fatalf("pgo_optimizer_input row has missing facts after foundation evidence: %#v", pgo)
	}
	for _, want := range []string{
		"Options.ProfileInput",
		"profile_input_policy",
		"validation metadata",
		"translation validation",
		"profile-guided rewrite policy rejected",
		"no profile-guided rewrite",
	} {
		if !strings.Contains(pgo.Reason+" "+pgo.Evidence+" "+pgo.Boundary, want) {
			t.Fatalf("pgo_optimizer_input row missing %q: %#v", want, pgo)
		}
	}
	for _, want := range []string{
		"optimizer_profile_input_api",
		"pass_contract_profile_metadata",
		"translation_validation_for_profile_guided_decisions",
		"negative_safe_semantics_tests",
	} {
		if !containsString(pgo.RequiredFacts, want) {
			t.Fatalf("pgo_optimizer_input row missing required fact %q: %#v", want, pgo)
		}
	}

	targetCPU := byID[PGOLTOTargetCPUTargetCPUFeatureDetection]
	if targetCPU.Status != PGOLTOTargetCPUImplementedNarrow {
		t.Fatalf("target_cpu_feature_detection row = %#v, want implemented_narrow", targetCPU)
	}
	if targetCPU.OptimizerInput || targetCPU.ChangesSafeSemantics {
		t.Fatalf(
			"target_cpu_feature_detection must not enable optimizer input or semantic change: %#v",
			targetCPU,
		)
	}
	if len(targetCPU.MissingFacts) != 0 {
		t.Fatalf(
			"target_cpu_feature_detection row has missing facts after foundation evidence: %#v",
			targetCPU,
		)
	}
	for _, want := range []string{
		"target feature model",
		"portable baseline fallback",
		"guarded codegen contract",
		"negative safe-semantics",
		"no target-specific rewrite",
	} {
		if !strings.Contains(targetCPU.Reason+" "+targetCPU.Evidence+" "+targetCPU.Boundary, want) {
			t.Fatalf("target_cpu_feature_detection row missing %q: %#v", want, targetCPU)
		}
	}
	for _, want := range []string{
		"target_feature_model",
		"portable_baseline_fallback",
		"guarded_codegen_contract",
		"negative_safe_semantics_tests",
	} {
		if !containsString(targetCPU.RequiredFacts, want) {
			t.Fatalf(
				"target_cpu_feature_detection row missing required fact %q: %#v",
				want,
				targetCPU,
			)
		}
	}

	lto := byID[PGOLTOTargetCPULTOIncrementalModuleSummary]
	if lto.Status != PGOLTOTargetCPUImplementedNarrow {
		t.Fatalf("lto_incremental_module_summary row = %#v, want implemented_narrow", lto)
	}
	if lto.OptimizerInput || lto.ChangesSafeSemantics {
		t.Fatalf(
			"lto_incremental_module_summary must not enable optimizer input or semantic change: %#v",
			lto,
		)
	}
	if len(lto.MissingFacts) != 0 {
		t.Fatalf(
			"lto_incremental_module_summary row has missing facts after foundation evidence: %#v",
			lto,
		)
	}
	for _, want := range []string{
		"tetra.incremental.module_summary.v1",
		"dependency hash contract",
		"cross-module validation",
		"non-consumer boundary",
		"no LTO optimizer",
	} {
		if !strings.Contains(lto.Reason+" "+lto.Evidence+" "+lto.Boundary, want) {
			t.Fatalf("lto_incremental_module_summary row missing %q: %#v", want, lto)
		}
	}
	for _, want := range []string{
		"module_summary_schema",
		"dependency_hash_contract",
		"cross_module_validation_row",
		"incremental_cache_negative_tests",
		"non_consumer_boundary",
	} {
		if !containsString(lto.RequiredFacts, want) {
			t.Fatalf("lto_incremental_module_summary row missing required fact %q: %#v", want, lto)
		}
	}

	safe := byID[PGOLTOTargetCPUSafeSemanticsFlags]
	if safe.Status != PGOLTOTargetCPUImplementedNarrow {
		t.Fatalf("safe semantics row = %#v, want implemented_narrow guard", safe)
	}
	if safe.OptimizerInput || safe.ChangesSafeSemantics {
		t.Fatalf(
			"safe semantics guard must not enable optimizer input or semantic change: %#v",
			safe,
		)
	}
	if !containsString(safe.RequiredFacts, "validators_reject_fake_claims") {
		t.Fatalf("safe semantics row missing validators_reject_fake_claims fact: %#v", safe)
	}
	for _, want := range []string{
		"no public BuildOptions flag",
		"profile parsing is evidence-only",
		"no optimizer pass consumes profile",
		"safe-program semantics unchanged",
	} {
		if !strings.Contains(safe.Reason+" "+safe.Evidence+" "+safe.Boundary, want) {
			t.Fatalf("safe semantics row missing %q: %#v", want, safe)
		}
	}
}

func TestPGOLTOTargetCPUSafeSemanticsClosureProvesFinalP17Row(t *testing.T) {
	closure, err := PGOLTOTargetCPUSafeSemanticsClosure()
	if err != nil {
		t.Fatalf("PGOLTOTargetCPUSafeSemanticsClosure: %v", err)
	}
	if closure.SchemaVersion != "tetra.optimizer.pgo_lto_target_cpu.safe_semantics_closure.v1" {
		t.Fatalf("closure schema = %q", closure.SchemaVersion)
	}
	if closure.Status != PGOLTOTargetCPUImplementedNarrow {
		t.Fatalf("closure status = %q, want %q", closure.Status, PGOLTOTargetCPUImplementedNarrow)
	}
	if closure.ChangesSafeSemantics {
		t.Fatalf("closure must not change safe semantics: %#v", closure)
	}
	if closure.PublicSemanticFlagCount != 0 {
		t.Fatalf("public semantic flag count = %d, want 0", closure.PublicSemanticFlagCount)
	}
	for _, want := range []PGOLTOTargetCPUID{
		PGOLTOTargetCPUProfileCollectionFormat,
		PGOLTOTargetCPUPGOOptimizerInput,
		PGOLTOTargetCPUTargetCPUFeatureDetection,
		PGOLTOTargetCPULTOIncrementalModuleSummary,
		PGOLTOTargetCPUSafeSemanticsFlags,
	} {
		if !containsP17RowID(closure.CompletedRows, want) {
			t.Fatalf("closure missing completed row %s: %#v", want, closure.CompletedRows)
		}
	}
	for _, want := range []string{
		"public_build_options_semantic_flag_rejected",
		"profile_guided_rewrite_policy_rejected",
		"target_specific_optimization_evidence_rejected",
		"lto_codegen_consumer_rejected",
		"lto_linker_consumer_rejected",
		"coverage_validator_rejects_fake_claims",
	} {
		if !containsString(closure.RejectedUnsafeClaims, want) {
			t.Fatalf(
				"closure missing rejected unsafe claim %q: %#v",
				want,
				closure.RejectedUnsafeClaims,
			)
		}
	}
	for _, want := range []string{
		"compiler/internal/opt/opt_core.go::ValidatePGOLTOTargetCPUSafeSemanticsClosure",
		"compiler/compiler_suite_test.go::TestBuildOptionsExposeNoBackendSemanticMode",
		("compiler/internal/opt/opt_suite_test.go::" +
			"TestManagerRejectsProfileGuidedRewritePolicyUntilValidationE" +
			"xists"),
		("compiler/internal/cache/lto_summary_test.go::" +
			"TestIncrementalModuleSummaryV1RecordsDependencyHashContractA" +
			"ndRejectsConsumers"),
	} {
		if !containsString(closure.Evidence, want) {
			t.Fatalf("closure missing evidence %q: %#v", want, closure.Evidence)
		}
	}
	if !strings.Contains(
		closure.Boundary,
		"no PGO/profile/LTO/target-cpu public flag changes safe-program semantics",
	) {
		t.Fatalf("closure boundary missing safe-semantics non-claim: %q", closure.Boundary)
	}
}

func TestPGOLTOTargetCPUSafeSemanticsClosureRejectsFakeClaims(t *testing.T) {
	report, err := PGOLTOTargetCPUCoverage()
	if err != nil {
		t.Fatalf("PGOLTOTargetCPUCoverage: %v", err)
	}
	if err := ValidatePGOLTOTargetCPUSafeSemanticsClosure(report); err != nil {
		t.Fatalf("valid P17.4 coverage rejected: %v", err)
	}

	for name, tc := range map[string]struct {
		mutate func(PGOLTOTargetCPUCoverageReport) PGOLTOTargetCPUCoverageReport
		want   string
	}{
		"semantic change": {
			mutate: func(r PGOLTOTargetCPUCoverageReport) PGOLTOTargetCPUCoverageReport {
				return mutateP17Row(r, PGOLTOTargetCPUPGOOptimizerInput, func(row *PGOLTOTargetCPUCoverageRow) {
					row.ChangesSafeSemantics = true
				})
			},
			want: "changes safe semantics",
		},
		"incomplete row": {
			mutate: func(r PGOLTOTargetCPUCoverageReport) PGOLTOTargetCPUCoverageReport {
				return mutateP17Row(r, PGOLTOTargetCPUTargetCPUFeatureDetection, func(row *PGOLTOTargetCPUCoverageRow) {
					row.Status = PGOLTOTargetCPUNotYetCovered
				})
			},
			want: "not complete",
		},
		"missing fact": {
			mutate: func(r PGOLTOTargetCPUCoverageReport) PGOLTOTargetCPUCoverageReport {
				return mutateP17Row(r, PGOLTOTargetCPULTOIncrementalModuleSummary, func(row *PGOLTOTargetCPUCoverageRow) {
					row.MissingFacts = []string{"non_consumer_boundary"}
				})
			},
			want: "missing facts",
		},
		"profile format optimizer input": {
			mutate: func(r PGOLTOTargetCPUCoverageReport) PGOLTOTargetCPUCoverageReport {
				return mutateP17Row(r, PGOLTOTargetCPUProfileCollectionFormat, func(row *PGOLTOTargetCPUCoverageRow) {
					row.OptimizerInput = true
				})
			},
			want: "profile collection format",
		},
		"lto optimizer input": {
			mutate: func(r PGOLTOTargetCPUCoverageReport) PGOLTOTargetCPUCoverageReport {
				return mutateP17Row(r, PGOLTOTargetCPULTOIncrementalModuleSummary, func(row *PGOLTOTargetCPUCoverageRow) {
					row.OptimizerInput = true
				})
			},
			want: "LTO/incremental module summary",
		},
		"safe truth fact missing": {
			mutate: func(r PGOLTOTargetCPUCoverageReport) PGOLTOTargetCPUCoverageReport {
				return mutateP17Row(r, PGOLTOTargetCPUSafeSemanticsFlags, func(row *PGOLTOTargetCPUCoverageRow) {
					row.RequiredFacts = removeString(row.RequiredFacts, "safe_program_truth_preserved")
				})
			},
			want: "safe_program_truth_preserved",
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := ValidatePGOLTOTargetCPUSafeSemanticsClosure(tc.mutate(report))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf(
					"ValidatePGOLTOTargetCPUSafeSemanticsClosure error = %v, want %q",
					err,
					tc.want,
				)
			}
		})
	}
}

func TestProfileCollectionFormatV1RoundTripsAndRejectsUnsafeDrift(t *testing.T) {
	profile := ProfileCollection{
		SchemaVersion: ProfileCollectionSchemaVersion,
		ProgramHash:   "sha256:abc123",
		TargetTriple:  "linux-x64",
		Functions: []ProfileFunction{
			{
				ID:         "fn:z",
				Name:       "main",
				EntryCount: 8,
				Counters: []ProfileCounter{
					{Kind: "edge", Name: "return", Count: 1},
					{Kind: "edge", Name: "loop", Count: 5},
				},
			},
			{
				ID:         "fn:a",
				Name:       "helper",
				EntryCount: 3,
				Counters: []ProfileCounter{
					{Kind: "block", Name: "entry", Count: 3},
				},
			},
		},
	}

	encoded, err := MarshalProfileCollection(profile)
	if err != nil {
		t.Fatalf("MarshalProfileCollection: %v", err)
	}
	const wantJSON = ("{\"schema_version\":\"tetra.optimizer.profile.v1\"," +
		"\"program_hash\":\"sha256:abc123\",\"target_triple\":\"linux-x64\"," +
		"\"functions\":[{\"id\":\"fn:a\",\"name\":\"helper\",\"entry_count\":3," +
		"\"counters\":[{\"kind\":\"block\",\"name\":\"entry\",\"count\":3}]}," +
		"{\"id\":\"fn:z\",\"name\":\"main\",\"entry_count\":8,\"counters\":" +
		"[{\"kind\":\"edge\",\"name\":\"loop\",\"count\":5},{\"kind\":\"edge\"," +
		"\"name\":\"return\",\"count\":1}]}]}")
	if string(encoded) != wantJSON {
		t.Fatalf("canonical profile JSON:\n got %s\nwant %s", string(encoded), wantJSON)
	}
	decoded, err := ParseProfileCollection(encoded)
	if err != nil {
		t.Fatalf("ParseProfileCollection: %v", err)
	}
	reencoded, err := MarshalProfileCollection(decoded)
	if err != nil {
		t.Fatalf("MarshalProfileCollection(decoded): %v", err)
	}
	if string(reencoded) != wantJSON {
		t.Fatalf("round-trip profile JSON:\n got %s\nwant %s", string(reencoded), wantJSON)
	}

	for name, raw := range map[string][]byte{
		"wrong schema": []byte(("{\"schema_version\":\"tetra.optimizer.profile.v2\"," +
			"\"program_hash\":\"sha256:abc123\",\"target_triple\":\"linux-x64\"," +
			"\"functions\":[{\"id\":\"fn:a\",\"name\":\"helper\",\"entry_count\":1}]}")),
		"duplicate id": []byte(("{\"schema_version\":\"tetra.optimizer.profile.v1\"," +
			"\"program_hash\":\"sha256:abc123\",\"target_triple\":\"linux-x64\"," +
			"\"functions\":[{\"id\":\"fn:a\",\"name\":\"helper\",\"entry_count\":1}," +
			"{\"id\":\"fn:a\",\"name\":\"other\",\"entry_count\":1}]}")),
		"duplicate name": []byte(("{\"schema_version\":\"tetra.optimizer.profile.v1\"," +
			"\"program_hash\":\"sha256:abc123\",\"target_triple\":\"linux-x64\"," +
			"\"functions\":[{\"id\":\"fn:a\",\"name\":\"helper\",\"entry_count\":1}," +
			"{\"id\":\"fn:b\",\"name\":\"helper\",\"entry_count\":1}]}")),
		"negative counter": []byte(("{\"schema_version\":\"tetra.optimizer.profile.v1\"," +
			"\"program_hash\":\"sha256:abc123\",\"target_triple\":\"linux-x64\"," +
			"\"functions\":[{\"id\":\"fn:a\",\"name\":\"helper\",\"entry_count\":1," +
			"\"counters\":[{\"kind\":\"edge\",\"name\":\"loop\",\"count\":-1}]}]}")),
	} {
		if _, err := ParseProfileCollection(raw); err == nil {
			t.Fatalf("%s: ParseProfileCollection succeeded, want rejection", name)
		}
	}
}

func containsP17RowID(values []PGOLTOTargetCPUID, want PGOLTOTargetCPUID) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func mutateP17Row(
	report PGOLTOTargetCPUCoverageReport,
	id PGOLTOTargetCPUID,
	mutate func(*PGOLTOTargetCPUCoverageRow),
) PGOLTOTargetCPUCoverageReport {
	out := PGOLTOTargetCPUCoverageReport{
		SchemaVersion: report.SchemaVersion,
		Rows:          append([]PGOLTOTargetCPUCoverageRow(nil), report.Rows...),
		NonClaims:     append([]string(nil), report.NonClaims...),
	}
	for i := range out.Rows {
		out.Rows[i].RequiredFacts = append([]string(nil), out.Rows[i].RequiredFacts...)
		out.Rows[i].MissingFacts = append([]string(nil), out.Rows[i].MissingFacts...)
		if out.Rows[i].ID == id {
			mutate(&out.Rows[i])
		}
	}
	return out
}

func removeString(values []string, remove string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != remove {
			out = append(out, value)
		}
	}
	return out
}

// ---- scalar_expression_test.go ----

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
		t.Fatalf(
			"optimized neg_i32 count = %d, want mutated source expression preserved; dump:\n%s",
			got,
			after,
		)
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
		t.Fatalf(
			"optimized min-int neg_i32 count = %d, want both unsafe unary expressions preserved; dump:\n%s",
			got,
			after,
		)
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
		{
			name:       "less-than-greater-than",
			first:      ir.IRCmpLtI32,
			firstOp:    "cmp_lt_i32",
			mirror:     ir.IRCmpGtI32,
			mirrorOp:   "cmp_gt_i32",
			beforeExit: 2,
		},
		{
			name:       "less-equal-greater-equal",
			first:      ir.IRCmpLeI32,
			firstOp:    "cmp_le_i32",
			mirror:     ir.IRCmpGeI32,
			mirrorOp:   "cmp_ge_i32",
			beforeExit: 2,
		},
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
				t.Fatalf(
					"optimized %s count = %d, want original comparison only; dump:\n%s",
					tc.firstOp,
					got,
					after,
				)
			}
			if strings.Contains(after, tc.mirrorOp) {
				t.Fatalf(
					"mirrored comparison was recomputed instead of reusing cached local:\n%s",
					after,
				)
			}
			if got := countDumpOccurrences(after, "load_local local:2"); got != 2 {
				t.Fatalf(
					"cached comparison loads = %d, want mirrored comparison to reuse local 2; dump:\n%s",
					got,
					after,
				)
			}
			beforeExit := runOptLinuxX64(t, before.Funcs, "before-mirrored-"+tc.name+"-gvn")
			afterExit := runOptLinuxX64(t, prog.Funcs, "after-mirrored-"+tc.name+"-gvn")
			if beforeExit != afterExit || afterExit != tc.beforeExit {
				t.Fatalf(
					"native exits before=%d after=%d want %d",
					beforeExit,
					afterExit,
					tc.beforeExit,
				)
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
		{
			name:      "division",
			kind:      ir.IRDivI32,
			opName:    "div_i32",
			denom:     3,
			wantExit:  8,
			finalKind: ir.IRAddI32,
		},
		{
			name:      "modulo",
			kind:      ir.IRModI32,
			opName:    "mod_i32",
			denom:     5,
			wantExit:  4,
			finalKind: ir.IRAddI32,
		},
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
				t.Fatalf(
					"optimized %s count = %d, want one safe cached expression; dump:\n%s",
					tc.opName,
					got,
					after,
				)
			}
			if got := countDumpOccurrences(after, "const_i32 "+strconv.Itoa(int(tc.denom))); got != 1 {
				t.Fatalf(
					"optimized denominator const count = %d, want one cached expression input; dump:\n%s",
					got,
					after,
				)
			}
			if got := countDumpOccurrences(after, "load_local local:1"); got != 2 {
				t.Fatalf(
					("optimized cached-expression loads = %d, want repeated safe " +
						"expression to reuse local 1; dump:\n%s"),
					got,
					after,
				)
			}
			beforeExit := runOptLinuxX64(t, before.Funcs, "before-safe-"+tc.name+"-cse")
			afterExit := runOptLinuxX64(t, prog.Funcs, "after-safe-"+tc.name+"-cse")
			if beforeExit != afterExit || afterExit != tc.wantExit {
				t.Fatalf(
					"native exits before=%d after=%d want %d",
					beforeExit,
					afterExit,
					tc.wantExit,
				)
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
		t.Fatalf(
			"optimized neg_i32 count = %d, want one cached unary expression; dump:\n%s",
			got,
			after,
		)
	}
	if got := countDumpOccurrences(after, "load_local local:1"); got != 2 {
		t.Fatalf(
			"optimized cached unary-expression loads = %d, want repeated neg to reuse local 1; dump:\n%s",
			got,
			after,
		)
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
				t.Fatalf(
					"optimized unsafe %s count = %d, want both expressions preserved; dump:\n%s",
					tc.op,
					got,
					after,
				)
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
		t.Fatalf(
			("optimized const count = %d, want both local-constant " +
				"expressions preserved after operand mutation; dump:\n%s"),
			got,
			after,
		)
	}
	if got := countDumpOccurrences(after, "add_i32"); got != 3 {
		t.Fatalf(
			"optimized add count = %d, want first expression, second expression, and final add; dump:\n%s",
			got,
			after,
		)
	}
	if got := countDumpOccurrences(after, "load_local local:2"); got != 1 {
		t.Fatalf(
			"cached local loads = %d, want only final use of local 2 after operand mutation; dump:\n%s",
			got,
			after,
		)
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
		t.Fatalf(
			"optimized sub count = %d, want both ordered sub expressions preserved; dump:\n%s",
			got,
			after,
		)
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

// ---- scalar_test.go ----

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
	if row.Name != "basic-scalar" || row.InputKind != IRKindStack ||
		row.OutputKind != IRKindStack ||
		!row.TranslationValidated {
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
			t.Fatalf(
				"after dump still contains folded safe div/mod artifact %q:\n%s",
				forbidden,
				after,
			)
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
				t.Fatalf(
					"unsafe div/mod constant fold changed unexpectedly:\nbefore:\n%s\nafter:\n%s",
					before,
					after,
				)
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
	for _, want := range []string{
		"const_i32 2147483647",
		"const_i32 1",
		"add_i32",
		"const_i32 -2147483648",
		"neg_i32",
	} {
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
				t.Fatalf(
					"after dump missing same-local comparison constant %d:\n%s",
					tc.wantExit,
					after,
				)
			}
			if strings.Contains(after, tc.op) {
				t.Fatalf("same-local comparison op was not simplified:\n%s", after)
			}
			beforeExit := runOptLinuxX64(
				t,
				before.Funcs,
				"before-same-local-"+tc.name+"-comparison",
			)
			afterExit := runOptLinuxX64(t, prog.Funcs, "after-same-local-"+tc.name+"-comparison")
			if beforeExit != afterExit || afterExit != tc.wantExit {
				t.Fatalf(
					"native exits before=%d after=%d want %d",
					beforeExit,
					afterExit,
					tc.wantExit,
				)
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
	for _, forbidden := range []string{
		"const_i32 99",
		"store_local local:1",
		"store_local local:2",
		"load_local local:2",
		"add_i32",
	} {
		if strings.Contains(after, forbidden) {
			t.Fatalf("after dump still contains %q:\n%s", forbidden, after)
		}
	}
	if got := len(prog.Funcs[0].Instrs); got != 2 {
		t.Fatalf(
			"optimized instruction count = %d, want load_local + return only; dump:\n%s",
			got,
			after,
		)
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
			t.Fatalf(
				"after dump still contains dead comparison store artifact %q:\n%s",
				forbidden,
				after,
			)
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
	for _, forbidden := range []string{
		"div_i32",
		"mod_i32",
		"store_local local:1",
		"store_local local:2",
	} {
		if strings.Contains(after, forbidden) {
			t.Fatalf(
				"after dump still contains safe dead div/mod store artifact %q:\n%s",
				forbidden,
				after,
			)
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
		{
			name:      "division",
			kind:      ir.IRDivI32,
			op:        "div_i32",
			leftImm:   20,
			rightImm:  5,
			localSlot: 2,
		},
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
			for _, forbidden := range []string{
				tc.op,
				"store_local local:0",
				"store_local local:1",
				"store_local local:2",
			} {
				if strings.Contains(after, forbidden) {
					t.Fatalf(
						"after dump still contains safe known-local div/mod dead store artifact %q:\n%s",
						forbidden,
						after,
					)
				}
			}
			if !strings.Contains(after, "const_i32 3") || !strings.Contains(after, "return") {
				t.Fatalf("after dump missing live return value:\n%s", after)
			}
			beforeExit := runOptLinuxX64(
				t,
				before.Funcs,
				"before-dead-safe-known-local-divmod-"+tc.name,
			)
			afterExit := runOptLinuxX64(
				t,
				prog.Funcs,
				"after-dead-safe-known-local-divmod-"+tc.name,
			)
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
				t.Fatalf(
					"unsafe known-local div/mod dead store changed unexpectedly:\nbefore:\n%s\nafter:\n%s",
					before,
					after,
				)
			}
			for _, want := range []string{
				"load_local local:0",
				"load_local local:1",
				tc.op,
				"store_local local:2",
			} {
				if !strings.Contains(after, want) {
					t.Fatalf(
						"after dump missing preserved unsafe div/mod artifact %q:\n%s",
						want,
						after,
					)
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
	for _, forbidden := range []string{
		"load_local local:0",
		"neg_i32",
		"store_local local:1",
		"store_local local:0",
	} {
		if strings.Contains(after, forbidden) {
			t.Fatalf(
				"after dump still contains safe dead unary neg store artifact %q:\n%s",
				forbidden,
				after,
			)
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
		t.Fatalf(
			"unsafe known-local unary neg dead store changed unexpectedly:\nbefore:\n%s\nafter:\n%s",
			before,
			after,
		)
	}
	for _, want := range []string{
		"const_i32 -2147483648",
		"load_local local:0",
		"neg_i32",
		"store_local local:1",
	} {
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
		{
			name:      "mul-two-locals",
			kind:      ir.IRMulI32,
			op:        "mul_i32",
			leftImm:   6,
			rightImm:  7,
			rightLoad: true,
		},
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
					t.Fatalf(
						"after dump still contains safe dead arithmetic store artifact %q:\n%s",
						forbidden,
						after,
					)
				}
			}
			if tc.rightLoad && strings.Contains(after, "store_local local:1") {
				t.Fatalf("after dump still contains dead right operand local store:\n%s", after)
			}
			if !strings.Contains(after, "const_i32 3") || !strings.Contains(after, "return") {
				t.Fatalf("after dump missing live return value:\n%s", after)
			}
			beforeExit := runOptLinuxX64(
				t,
				before.Funcs,
				"before-dead-safe-known-local-arithmetic-"+tc.name,
			)
			afterExit := runOptLinuxX64(
				t,
				prog.Funcs,
				"after-dead-safe-known-local-arithmetic-"+tc.name,
			)
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
				t.Fatalf(
					"unsafe known-local arithmetic dead store changed unexpectedly:\nbefore:\n%s\nafter:\n%s",
					before,
					after,
				)
			}
			for _, want := range []string{"load_local local:0", tc.op, "store_local local:1"} {
				if !strings.Contains(after, want) {
					t.Fatalf(
						"after dump missing preserved unsafe arithmetic artifact %q:\n%s",
						want,
						after,
					)
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
		t.Fatalf(
			"optimized add count = %d, want initial expression plus final add; dump:\n%s",
			got,
			after,
		)
	}
	if got := countDumpOccurrences(after, "load_local local:2"); got != 2 {
		t.Fatalf(
			"optimized cached-expression loads = %d, want repeated expression to reuse local 2; dump:\n%s",
			got,
			after,
		)
	}
	if strings.Contains(
		after,
		"load_local local:0\n  load_local local:1\n  add_i32\n  load_local local:2",
	) {
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
		t.Fatalf(
			"optimized add count = %d, want initial commutative expression plus final add; dump:\n%s",
			got,
			after,
		)
	}
	if got := countDumpOccurrences(after, "load_local local:2"); got != 2 {
		t.Fatalf(
			"optimized cached-expression loads = %d, want swapped expression to reuse local 2; dump:\n%s",
			got,
			after,
		)
	}
	if strings.Contains(
		after,
		"load_local local:1\n  load_local local:0\n  add_i32\n  load_local local:2",
	) {
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
		t.Fatalf(
			"optimized const count = %d, want one cached local-constant expression input; dump:\n%s",
			got,
			after,
		)
	}
	if got := countDumpOccurrences(after, "add_i32"); got != 2 {
		t.Fatalf(
			"optimized add count = %d, want initial expression plus final add; dump:\n%s",
			got,
			after,
		)
	}
	if got := countDumpOccurrences(after, "load_local local:2"); got != 2 {
		t.Fatalf(
			("optimized cached-expression loads = %d, want repeated " +
				"local-constant expression to reuse local 2; dump:\n%s"),
			got,
			after,
		)
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
		t.Fatalf(
			("optimized const count = %d, want swapped local-constant " +
				"expression to reuse cached local; dump:\n%s"),
			got,
			after,
		)
	}
	if got := countDumpOccurrences(after, "add_i32"); got != 2 {
		t.Fatalf(
			"optimized add count = %d, want initial expression plus final add; dump:\n%s",
			got,
			after,
		)
	}
	if got := countDumpOccurrences(after, "load_local local:2"); got != 2 {
		t.Fatalf(
			("optimized cached-expression loads = %d, want swapped " +
				"local-constant expression to reuse local 2; dump:\n%s"),
			got,
			after,
		)
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
		{
			name:        "add",
			kind:        ir.IRAddI32,
			opName:      "add_i32",
			leftImm:     5,
			rightImm:    7,
			wantExit:    24,
			wantOpCount: 2,
		},
		{
			name:        "sub",
			kind:        ir.IRSubI32,
			opName:      "sub_i32",
			leftImm:     11,
			rightImm:    4,
			wantExit:    14,
			wantOpCount: 1,
		},
		{
			name:        "mul",
			kind:        ir.IRMulI32,
			opName:      "mul_i32",
			leftImm:     6,
			rightImm:    7,
			wantExit:    84,
			wantOpCount: 1,
		},
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
				t.Fatalf(
					"optimized %s count = %d, want %d with known-local value reuse; dump:\n%s",
					tc.opName,
					got,
					tc.wantOpCount,
					after,
				)
			}
			if got := countDumpOccurrences(after, "load_local local:2"); got != 2 {
				t.Fatalf(
					("cached-expression loads = %d, want repeated known-local " +
						"expression to reuse local 2; dump:\n%s"),
					got,
					after,
				)
			}
			if strings.Contains(
				after,
				"load_local local:3\n  load_local local:1\n  "+tc.opName+"\n  load_local local:2",
			) {
				t.Fatalf(
					"known-local equivalent expression was recomputed instead of reusing cached local:\n%s",
					after,
				)
			}
			beforeExit := runOptLinuxX64(t, before.Funcs, "before-safe-known-local-"+tc.name+"-gvn")
			afterExit := runOptLinuxX64(t, prog.Funcs, "after-safe-known-local-"+tc.name+"-gvn")
			if beforeExit != afterExit || afterExit != tc.wantExit {
				t.Fatalf(
					"native exits before=%d after=%d want %d",
					beforeExit,
					afterExit,
					tc.wantExit,
				)
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
		t.Fatalf(
			"optimized cmp_lt_i32 count = %d, want original comparison only; dump:\n%s",
			got,
			after,
		)
	}
	if strings.Contains(after, "cmp_gt_i32") {
		t.Fatalf(
			"mirrored known-local comparison was recomputed instead of reusing cached local:\n%s",
			after,
		)
	}
	if got := countDumpOccurrences(after, "load_local local:2"); got != 2 {
		t.Fatalf(
			"cached comparison loads = %d, want known-local comparison to reuse local 2; dump:\n%s",
			got,
			after,
		)
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "before-safe-known-local-comparison-gvn")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-safe-known-local-comparison-gvn")
	if beforeExit != afterExit || afterExit != 2 {
		t.Fatalf("native exits before=%d after=%d want 2", beforeExit, afterExit)
	}
}

func TestBasicScalarPassDoesNotReuseKnownLocalComparisonExpressionAfterSourceMutation(
	t *testing.T,
) {
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
		t.Fatalf(
			"optimized cmp_lt_i32 count = %d, want original comparison preserved; dump:\n%s",
			got,
			after,
		)
	}
	if got := countDumpOccurrences(after, "cmp_gt_i32"); got != 1 {
		t.Fatalf(
			"optimized cmp_gt_i32 count = %d, want mutated comparison preserved; dump:\n%s",
			got,
			after,
		)
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

func TestBasicScalarPassDoesNotReuseKnownLocalArithmeticExpressionAfterSourceMutation(
	t *testing.T,
) {
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
		t.Fatalf(
			"optimized add count = %d, want mutated source expression preserved plus final add; dump:\n%s",
			got,
			after,
		)
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
		t.Fatalf(
			("optimized overflow-sensitive add count = %d, want both " +
				"unsafe expressions preserved plus final add; dump:\n%s"),
			got,
			after,
		)
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
		{
			name:     "division",
			kind:     ir.IRDivI32,
			opName:   "div_i32",
			leftImm:  20,
			rightImm: 5,
			wantExit: 8,
		},
		{
			name:     "modulo",
			kind:     ir.IRModI32,
			opName:   "mod_i32",
			leftImm:  23,
			rightImm: 5,
			wantExit: 6,
		},
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
				t.Fatalf(
					"optimized %s count = %d, want one safe known-local value expression; dump:\n%s",
					tc.opName,
					got,
					after,
				)
			}
			if got := countDumpOccurrences(after, "load_local local:2"); got != 2 {
				t.Fatalf(
					"cached-expression loads = %d, want repeated known-local %s to reuse local 2; dump:\n%s",
					got,
					tc.opName,
					after,
				)
			}
			if strings.Contains(
				after,
				"load_local local:3\n  load_local local:1\n  "+tc.opName+"\n  load_local local:2",
			) {
				t.Fatalf(
					"known-local div/mod expression was recomputed instead of reusing cached local:\n%s",
					after,
				)
			}
			beforeExit := runOptLinuxX64(t, before.Funcs, "before-safe-known-local-"+tc.name+"-gvn")
			afterExit := runOptLinuxX64(t, prog.Funcs, "after-safe-known-local-"+tc.name+"-gvn")
			if beforeExit != afterExit || afterExit != tc.wantExit {
				t.Fatalf(
					"native exits before=%d after=%d want %d",
					beforeExit,
					afterExit,
					tc.wantExit,
				)
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
		t.Fatalf(
			("optimized div_i32 count = %d, want mutated source " +
				"expression preserved plus final add; dump:\n%s"),
			got,
			after,
		)
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
				t.Fatalf(
					"optimized unsafe %s count = %d, want both expressions preserved; dump:\n%s",
					tc.op,
					got,
					after,
				)
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
		t.Fatalf(
			"optimized neg_i32 count = %d, want one safe known-local unary expression; dump:\n%s",
			got,
			after,
		)
	}
	if got := countDumpOccurrences(after, "load_local local:2"); got != 2 {
		t.Fatalf(
			"cached unary-expression loads = %d, want repeated known-local neg to reuse local 2; dump:\n%s",
			got,
			after,
		)
	}
	if strings.Contains(after, "load_local local:3\n  neg_i32\n  load_local local:2") {
		t.Fatalf(
			"known-local unary expression was recomputed instead of reusing cached local:\n%s",
			after,
		)
	}
	beforeExit := runOptLinuxX64(t, before.Funcs, "before-safe-known-local-unary-gvn")
	afterExit := runOptLinuxX64(t, prog.Funcs, "after-safe-known-local-unary-gvn")
	if beforeExit != afterExit || afterExit != 12 {
		t.Fatalf("native exits before=%d after=%d want 12", beforeExit, afterExit)
	}
}

// ---- sccp_programs_test.go ----

func constantZeroBranchProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func singlePredecessorKnownLocalBranchProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRJmp, Label: 1},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func mergeLabelKnownLocalBranchProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRJmp, Label: 1},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func forwardSinglePredecessorKnownLocalBranchProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRJmp, Label: 1},
				{Kind: ir.IRConstI32, Imm: 11},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func forwardFallthroughPredecessorKnownLocalBranchProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRJmp, Label: 2},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRJmpIfZero, Label: 3},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 3},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func foldedZeroBranchSinglePredecessorKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
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
}

func foldedZeroBranchFallthroughTargetKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
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
}

func foldedNonzeroFallthroughOnlyLabelKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 9},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 9},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func foldedNonzeroFallthroughExplicitIncomingLabelKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 9},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 9},
				{Kind: ir.IRJmp, Label: 1},
			},
		}},
	}
}

func dynamicZeroTargetPathKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
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
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func dynamicNonzeroFallthroughPathKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
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
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 13},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func dynamicZeroFallthroughTargetPathKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
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
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func dynamicEqZeroFallthroughPathKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRCmpEqI32},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 13},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func dynamicEqZeroTargetNonzeroPathKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRCmpEqI32},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func dynamicNeZeroFallthroughPathKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRCmpNeI32},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 13},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func dynamicNeZeroTargetZeroPathKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRCmpNeI32},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func dynamicComparisonFallthroughTargetPathKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRCmpNeI32},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func knownLocalLessThanBranchProgram(localValue int32, compareImm int32) *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: localValue},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: compareImm},
				{Kind: ir.IRCmpLtI32},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func unaryNegBranchProgram(imm int32) *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: imm},
				{Kind: ir.IRNegI32},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func storedUnaryNegBranchProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRNegI32},
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
}

func storedConstantExpressionBranchProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  2,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 3},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 3},
				{Kind: ir.IRSubI32},
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
}

func storedDynamicExpressionBranchProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 3},
				{Kind: ir.IRSubI32},
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
}

func knownLocalZeroBranchProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func constantNonZeroBranchProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func countDecisions(decisions []PassDecision, action string, reason string) int {
	count := 0
	for _, decision := range decisions {
		if decision.Action == action && decision.Reason == reason {
			count++
		}
	}
	return count
}

// ---- sccp_test.go ----

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
			if tc.wantPrune &&
				!hasDecision(
					row.Decisions,
					"pruned_unreachable_fallthrough",
					"constant_branch_reachability",
				) {
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
				t.Fatalf(
					"native exits before=%d after=%d want %d",
					beforeExit,
					afterExit,
					tc.wantExit,
				)
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
	if !strings.Contains(row.AfterDump, "load_local local:0") ||
		!strings.Contains(row.AfterDump, "jmp_if_zero label:1") {
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
	if !hasDecision(
		row.Decisions,
		"pruned_unreachable_fallthrough",
		"constant_branch_reachability",
	) {
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
	if !hasDecision(
		row.Decisions,
		"pruned_unreachable_fallthrough",
		"constant_branch_reachability",
	) {
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
	if !hasDecision(
		row.Decisions,
		"folded_known_local_nonzero_fallthrough",
		"constant_local_condition",
	) {
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
	if !strings.Contains(row.AfterDump, "load_local local:0") ||
		!strings.Contains(row.AfterDump, "jmp_if_zero label:2") {
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
	if !hasDecision(
		row.Decisions,
		"propagated_known_local_single_predecessor",
		"single_predecessor_label",
	) {
		t.Fatalf("single-predecessor propagation decision missing: %#v", row.Decisions)
	}
	if !hasDecision(row.Decisions, "folded_known_local_zero_branch", "constant_local_condition") {
		t.Fatalf("known-local zero decision missing after label propagation: %#v", row.Decisions)
	}
	if !hasDecision(
		row.Decisions,
		"pruned_unreachable_fallthrough",
		"constant_branch_reachability",
	) {
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
	if hasDecision(
		row.Decisions,
		"propagated_known_local_single_predecessor",
		"single_predecessor_label",
	) {
		t.Fatalf("known-local fact propagated through merge label: %#v", row.Decisions)
	}
	if hasDecision(row.Decisions, "folded_known_local_zero_branch", "constant_local_condition") {
		t.Fatalf("merge label branch was folded with path-sensitive ambiguity: %#v", row.Decisions)
	}
	if row.AfterDump != before {
		t.Fatalf(
			"merge-label function changed unexpectedly:\nbefore:\n%s\nafter:\n%s",
			before,
			row.AfterDump,
		)
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
	if !hasDecision(
		row.Decisions,
		"propagated_known_local_single_predecessor",
		"forward_single_predecessor_jump",
	) {
		t.Fatalf("forward single-predecessor propagation decision missing: %#v", row.Decisions)
	}
	if !hasDecision(row.Decisions, "folded_known_local_zero_branch", "constant_local_condition") {
		t.Fatalf("known-local zero decision missing after forward propagation: %#v", row.Decisions)
	}
	if !hasDecision(
		row.Decisions,
		"pruned_unreachable_fallthrough",
		"constant_branch_reachability",
	) {
		t.Fatalf("prune decision missing after forward propagation: %#v", row.Decisions)
	}
	for _, forbidden := range []string{"load_local local:0\n  jmp_if_zero label:2", "const_i32 99"} {
		if strings.Contains(row.AfterDump, forbidden) {
			t.Fatalf("after dump still contains %q:\n%s", forbidden, row.AfterDump)
		}
	}
	for _, want := range []string{
		"jmp label:1",
		"const_i32 11",
		"label:1",
		"jmp label:2",
		"const_i32 7",
	} {
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

func TestSCCPPassDoesNotPropagateKnownLocalThroughForwardJumpWithFallthroughPredecessor(
	t *testing.T,
) {
	prog := forwardFallthroughPredecessorKnownLocalBranchProgram()
	before := FormatProgram(prog)

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if hasDecision(
		row.Decisions,
		"propagated_known_local_single_predecessor",
		"forward_single_predecessor_jump",
	) {
		t.Fatalf(
			"known-local fact propagated through label with fallthrough predecessor: %#v",
			row.Decisions,
		)
	}
	if hasDecision(row.Decisions, "folded_known_local_zero_branch", "constant_local_condition") {
		t.Fatalf(
			"fallthrough-predecessor label branch was folded with path-sensitive ambiguity: %#v",
			row.Decisions,
		)
	}
	if row.AfterDump != before {
		t.Fatalf(
			"fallthrough-predecessor function changed unexpectedly:\nbefore:\n%s\nafter:\n%s",
			before,
			row.AfterDump,
		)
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
	if !hasDecision(
		row.Decisions,
		"propagated_known_local_folded_zero_branch",
		"folded_zero_branch_forward_single_predecessor_jump",
	) {
		t.Fatalf("folded zero-branch propagation decision missing: %#v", row.Decisions)
	}
	if got := countDecisions(
		row.Decisions,
		"folded_known_local_zero_branch",
		"constant_local_condition",
	); got != 2 {
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
	if hasDecision(
		row.Decisions,
		"propagated_known_local_folded_zero_branch",
		"folded_zero_branch_single_predecessor_label",
	) {
		t.Fatalf(
			"known-local fact propagated through folded branch target with fallthrough predecessor: %#v",
			row.Decisions,
		)
	}
	if countDecisions(
		row.Decisions,
		"folded_known_local_zero_branch",
		"constant_local_condition",
	) != 1 {
		t.Fatalf("only the first known-local zero branch should fold: %#v", row.Decisions)
	}
	if !strings.Contains(row.AfterDump, "load_local local:0\n  jmp_if_zero label:2") {
		t.Fatalf(
			"fallthrough-target branch changed despite ambiguous predecessor:\n%s",
			row.AfterDump,
		)
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
	if !hasDecision(
		row.Decisions,
		"propagated_known_local_folded_nonzero_fallthrough",
		"folded_nonzero_fallthrough_label",
	) {
		t.Fatalf("folded nonzero fallthrough propagation decision missing: %#v", row.Decisions)
	}
	if got := countDecisions(
		row.Decisions,
		"folded_known_local_nonzero_fallthrough",
		"constant_local_condition",
	); got != 2 {
		t.Fatalf(
			"folded known-local nonzero branch decisions = %d, want 2: %#v",
			got,
			row.Decisions,
		)
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

func TestSCCPPassDoesNotPropagateFoldedNonzeroFallthroughThroughExplicitIncomingLabel(
	t *testing.T,
) {
	prog := foldedNonzeroFallthroughExplicitIncomingLabelKnownLocalProgram()
	before := cloneProgram(prog)

	report, err := NewManager().Run(prog, SCCPPass())
	if err != nil {
		t.Fatalf("Run SCCPPass: %v", err)
	}
	row := report.Passes[0]
	if hasDecision(
		row.Decisions,
		"propagated_known_local_folded_nonzero_fallthrough",
		"folded_nonzero_fallthrough_label",
	) {
		t.Fatalf(
			"known-local fact propagated through explicit-incoming fallthrough label: %#v",
			row.Decisions,
		)
	}
	if got := countDecisions(
		row.Decisions,
		"folded_known_local_nonzero_fallthrough",
		"constant_local_condition",
	); got != 1 {
		t.Fatalf(
			"folded known-local nonzero branch decisions = %d, want 1: %#v",
			got,
			row.Decisions,
		)
	}
	if !strings.Contains(row.AfterDump, "load_local local:0\n  jmp_if_zero label:2") {
		t.Fatalf(
			"explicit-incoming label branch changed despite merge ambiguity:\n%s",
			row.AfterDump,
		)
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
	if !hasDecision(
		row.Decisions,
		"propagated_path_local_zero_target",
		"dynamic_zero_forward_single_predecessor_jump",
	) {
		t.Fatalf("dynamic zero target fact decision missing: %#v", row.Decisions)
	}
	if !hasDecision(row.Decisions, "folded_path_local_zero_branch", "path_local_condition") {
		t.Fatalf("path-known zero branch fold missing: %#v", row.Decisions)
	}
	if !hasDecision(
		row.Decisions,
		"pruned_unreachable_fallthrough",
		"constant_branch_reachability",
	) {
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
	if !hasDecision(
		row.Decisions,
		"derived_path_local_nonzero_fallthrough",
		"dynamic_branch_fallthrough",
	) {
		t.Fatalf("dynamic nonzero fallthrough fact decision missing: %#v", row.Decisions)
	}
	if !hasDecision(
		row.Decisions,
		"folded_path_local_nonzero_fallthrough",
		"path_local_condition",
	) {
		t.Fatalf("path-known nonzero branch fold missing: %#v", row.Decisions)
	}
	if got := strings.Count(row.AfterDump, "jmp_if_zero"); got != 1 {
		t.Fatalf(
			"after dump jmp_if_zero count = %d, want only the original dynamic branch:\n%s",
			got,
			row.AfterDump,
		)
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
		t.Fatalf(
			"fallthrough-target branch was folded with path-sensitive ambiguity: %#v",
			row.Decisions,
		)
	}
	if !strings.Contains(row.AfterDump, "load_local local:0\n  jmp_if_zero label:2") {
		t.Fatalf("fallthrough-target branch changed despite ambiguity:\n%s", row.AfterDump)
	}
	if row.AfterDump != before {
		t.Fatalf(
			"fallthrough-target function changed unexpectedly:\nbefore:\n%s\nafter:\n%s",
			before,
			row.AfterDump,
		)
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
			required: []string{
				"load_local local:0\n  const_i32 0\n  cmp_eq_i32\n  jmp_if_zero label:1",
				"jmp label:2",
				"const_i32 7",
			},
		},
		{
			name:           "target_nonzero",
			prog:           dynamicEqZeroTargetNonzeroPathKnownLocalProgram(),
			wantDecision:   "propagated_comparison_path_local_nonzero_target",
			wantReason:     "eq_zero_false_forward_single_predecessor_jump",
			wantFoldAction: "folded_path_local_nonzero_fallthrough",
			forbidden:      []string{"load_local local:0\n  jmp_if_zero label:2"},
			required: []string{
				"load_local local:0\n  const_i32 0\n  cmp_eq_i32\n  jmp_if_zero label:1",
				"const_i32 42",
			},
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
				t.Fatalf(
					"comparison path fact decision missing %s/%s: %#v",
					tc.wantDecision,
					tc.wantReason,
					row.Decisions,
				)
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
			required: []string{
				"load_local local:0\n  const_i32 0\n  cmp_ne_i32\n  jmp_if_zero label:1",
				"const_i32 42",
			},
		},
		{
			name:           "target_zero",
			prog:           dynamicNeZeroTargetZeroPathKnownLocalProgram(),
			wantDecision:   "propagated_comparison_path_local_zero_target",
			wantReason:     "ne_zero_false_forward_single_predecessor_jump",
			wantFoldAction: "folded_path_local_zero_branch",
			forbidden:      []string{"load_local local:0\n  jmp_if_zero label:2", "const_i32 99"},
			required: []string{
				"load_local local:0\n  const_i32 0\n  cmp_ne_i32\n  jmp_if_zero label:1",
				"jmp label:2",
				"const_i32 7",
			},
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
				t.Fatalf(
					"comparison path fact decision missing %s/%s: %#v",
					tc.wantDecision,
					tc.wantReason,
					row.Decisions,
				)
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
	if hasDecision(
		row.Decisions,
		"propagated_comparison_path_local_zero_target",
		"ne_zero_false_single_predecessor_label",
	) {
		t.Fatalf("fallthrough-target comparison fact propagated with ambiguity: %#v", row.Decisions)
	}
	if hasDecision(row.Decisions, "folded_path_local_zero_branch", "path_local_condition") {
		t.Fatalf(
			"fallthrough-target branch was folded with comparison ambiguity: %#v",
			row.Decisions,
		)
	}
	if !strings.Contains(row.AfterDump, "load_local local:0\n  jmp_if_zero label:2") {
		t.Fatalf("path branch unexpectedly changed:\n%s", row.AfterDump)
	}
	if before != FormatProgram(prog) {
		t.Fatalf(
			"ambiguous comparison target program changed:\nbefore:\n%s\nafter:\n%s",
			before,
			FormatProgram(prog),
		)
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
			forbidden: []string{
				"load_local local:0\n  const_i32 5\n  cmp_lt_i32\n  jmp_if_zero label:1",
				"const_i32 99",
			},
			required: []string{"store_local local:0", "jmp label:1", "const_i32 7"},
		},
		{
			name:       "expression_nonzero_falls_through",
			localValue: 3,
			compareImm: 5,
			wantAction: "folded_const_expr_nonzero_fallthrough",
			wantExit:   42,
			forbidden: []string{
				"load_local local:0\n  const_i32 5\n  cmp_lt_i32\n  jmp_if_zero label:1",
			},
			required: []string{"store_local local:0", "const_i32 42", "label:1"},
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
			if tc.wantPrune &&
				!hasDecision(
					row.Decisions,
					"pruned_unreachable_fallthrough",
					"constant_branch_reachability",
				) {
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
				t.Fatalf(
					"native exits before=%d after=%d want %d",
					beforeExit,
					afterExit,
					tc.wantExit,
				)
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
				t.Fatalf(
					"unary expression branch decision missing %s: %#v",
					tc.wantAction,
					row.Decisions,
				)
			}
			if tc.wantPrune &&
				!hasDecision(
					row.Decisions,
					"pruned_unreachable_fallthrough",
					"constant_branch_reachability",
				) {
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
				t.Fatalf(
					"native exits before=%d after=%d want %d",
					beforeExit,
					afterExit,
					tc.wantExit,
				)
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
	if hasDecision(
		row.Decisions,
		"folded_const_unary_expr_zero_branch",
		"constant_unary_expression_condition",
	) ||
		hasDecision(
			row.Decisions,
			"folded_const_unary_expr_nonzero_fallthrough",
			"constant_unary_expression_condition",
		) {
		t.Fatalf("unsafe unary neg expression was folded: %#v", row.Decisions)
	}
	if row.AfterDump != before {
		t.Fatalf(
			"unsafe unary neg expression changed unexpectedly:\nbefore:\n%s\nafter:\n%s",
			before,
			row.AfterDump,
		)
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
	if !hasDecision(
		row.Decisions,
		"pruned_unreachable_fallthrough",
		"constant_branch_reachability",
	) {
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
			forbidden: []string{
				"const_i32 20\n  const_i32 5\n  mod_i32\n  jmp_if_zero label:1",
				"const_i32 99",
			},
			required: []string{"jmp label:1", "const_i32 7"},
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
				t.Fatalf(
					"safe %s branch decision missing %s: %#v",
					tc.op,
					tc.wantAction,
					row.Decisions,
				)
			}
			if tc.wantPrune &&
				!hasDecision(
					row.Decisions,
					"pruned_unreachable_fallthrough",
					"constant_branch_reachability",
				) {
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
				t.Fatalf(
					"native exits before=%d after=%d want %d",
					beforeExit,
					afterExit,
					tc.wantExit,
				)
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
				t.Fatalf(
					"unsafe div/mod branch changed unexpectedly:\nbefore:\n%s\nafter:\n%s",
					before,
					row.AfterDump,
				)
			}
			if hasDecision(
				row.Decisions,
				"folded_const_expr_zero_branch",
				"constant_expression_condition",
			) ||
				hasDecision(
					row.Decisions,
					"folded_const_expr_nonzero_fallthrough",
					"constant_expression_condition",
				) {
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
	if !hasDecision(
		row.Decisions,
		"pruned_unreachable_fallthrough",
		"constant_branch_reachability",
	) {
		t.Fatalf("prune decision missing after stored-expression fold: %#v", row.Decisions)
	}
	for _, forbidden := range []string{"load_local local:1\n  jmp_if_zero label:1", "const_i32 42"} {
		if strings.Contains(row.AfterDump, forbidden) {
			t.Fatalf("after dump still contains %q:\n%s", forbidden, row.AfterDump)
		}
	}
	for _, want := range []string{
		"load_local local:0",
		"const_i32 3",
		"sub_i32",
		"store_local local:1",
		"jmp label:1",
		"const_i32 7",
	} {
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
	if !hasDecision(
		row.Decisions,
		"pruned_unreachable_fallthrough",
		"constant_branch_reachability",
	) {
		t.Fatalf("prune decision missing after stored safe div/mod fold: %#v", row.Decisions)
	}
	for _, forbidden := range []string{"load_local local:1\n  jmp_if_zero label:1", "const_i32 42"} {
		if strings.Contains(row.AfterDump, forbidden) {
			t.Fatalf("after dump still contains %q:\n%s", forbidden, row.AfterDump)
		}
	}
	for _, want := range []string{
		"load_local local:0",
		"const_i32 5",
		"mod_i32",
		"store_local local:1",
		"jmp label:1",
		"const_i32 7",
	} {
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
		t.Fatalf(
			"dynamic stored-expression function changed unexpectedly:\nbefore:\n%s\nafter:\n%s",
			before,
			row.AfterDump,
		)
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
	if hasDecision(
		row.Decisions,
		"folded_const_expr_zero_branch",
		"constant_expression_condition",
	) {
		t.Fatalf("constant expression crossed a label: %#v", row.Decisions)
	}
	if !strings.Contains(row.AfterDump, "cmp_lt_i32") ||
		!strings.Contains(row.AfterDump, "jmp_if_zero label:1") {
		t.Fatalf("expression branch changed despite label boundary:\n%s", row.AfterDump)
	}
}

// ---- vectorization_test.go ----

func TestVectorizationCoverageAuditsP17PlanList(t *testing.T) {
	report, err := VectorizationCoverage()
	if err != nil {
		t.Fatalf("VectorizationCoverage: %v", err)
	}
	if report.SchemaVersion != "tetra.optimizer.vectorization.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	if !containsString(report.NonClaims, "no broad SIMD or auto-vectorization claim") {
		t.Fatalf("non-claims = %#v, want explicit broad-SIMD non-claim", report.NonClaims)
	}

	want := []VectorizationID{
		VectorizationSumI32,
		VectorizationCopyU8,
		VectorizationMemsetMemcpy,
		VectorizationMapI32,
	}
	if len(report.Rows) != len(want) {
		t.Fatalf("coverage rows = %d, want %d: %#v", len(report.Rows), len(want), report.Rows)
	}
	byID := map[VectorizationID]VectorizationCoverageRow{}
	for _, row := range report.Rows {
		byID[row.ID] = row
		if row.Name == "" || row.Status == "" || row.Decision == "" || row.Reason == "" ||
			row.Evidence == "" ||
			row.Boundary == "" {
			t.Fatalf("row missing required vectorization evidence: %#v", row)
		}
	}
	for _, id := range want {
		if _, ok := byID[id]; !ok {
			t.Fatalf("missing vectorization row %s", id)
		}
	}

	sum := byID[VectorizationSumI32]
	if sum.Status != VectorizationImplementedNarrow || sum.Decision != VectorizationVectorized {
		t.Fatalf("sum_i32 row = %#v, want implemented_narrow vectorized candidate", sum)
	}
	if !sum.Candidate || !sum.RangeProof || sum.ProofID == "" {
		t.Fatalf("sum_i32 row missing candidate/range-proof evidence: %#v", sum)
	}
	for _, want := range []string{
		"proof-tagged",
		"noalias not required",
		"safe unaligned",
		"vector backend lowering",
		"scalar tail",
		"scalar-i32-slice-sum",
		"vector-i32x4-slice-sum-plan",
		"native SIMD",
		"linux-x64",
		"translation/differential",
	} {
		if !strings.Contains(sum.Reason+" "+sum.Evidence+" "+sum.Boundary, want) {
			t.Fatalf("sum_i32 row missing %q: %#v", want, sum)
		}
	}
	if len(sum.MissingFacts) != 0 {
		t.Fatalf("sum_i32 row reports missing facts after native SIMD validation: %#v", sum)
	}
	for _, want := range []string{"native_simd_codegen", "translation_differential_validation"} {
		if !containsString(sum.RequiredFacts, want) {
			t.Fatalf("sum_i32 row missing required fact %q: %#v", want, sum)
		}
	}

	copyU8 := byID[VectorizationCopyU8]
	if copyU8.Status != VectorizationImplementedNarrow ||
		copyU8.Decision != VectorizationVectorized {
		t.Fatalf("copy_u8 row = %#v, want implemented_narrow vectorized candidate", copyU8)
	}
	if !copyU8.Candidate || !copyU8.RangeProof || copyU8.ProofID == "" {
		t.Fatalf("copy_u8 row missing candidate/range-proof evidence: %#v", copyU8)
	}
	for _, want := range []string{
		"copy-loop",
		"noalias required",
		"source/dest disjoint",
		"safe unaligned",
		"vector backend lowering",
		"native SIMD",
		"linux-x64",
		"translation/differential",
		"scalar tail",
		"scalar-u8-copy",
		"vector-u8x16-copy-plan",
	} {
		if !strings.Contains(copyU8.Reason+" "+copyU8.Evidence+" "+copyU8.Boundary, want) {
			t.Fatalf("copy_u8 row missing %q: %#v", want, copyU8)
		}
	}
	if len(copyU8.MissingFacts) != 0 {
		t.Fatalf("copy_u8 row reports missing facts after native SIMD validation: %#v", copyU8)
	}
	for _, want := range []string{"native_simd_codegen", "translation_differential_validation"} {
		if !containsString(copyU8.RequiredFacts, want) {
			t.Fatalf("copy_u8 row missing required fact %q: %#v", want, copyU8)
		}
	}

	mapI32 := byID[VectorizationMapI32]
	if mapI32.Status != VectorizationImplementedNarrow ||
		mapI32.Decision != VectorizationVectorized {
		t.Fatalf("map_i32 row = %#v, want implemented_narrow vectorized candidate", mapI32)
	}
	if !mapI32.Candidate || !mapI32.RangeProof || mapI32.ProofID == "" {
		t.Fatalf("map_i32 row missing candidate/range-proof evidence: %#v", mapI32)
	}
	for _, want := range []string{
		"map-loop",
		"single mutable slice in-place",
		"safe unaligned",
		"vector backend lowering",
		"native SIMD",
		"linux-x64",
		"translation/differential",
		"scalar tail",
		"scalar-i32-map",
		"vector-i32x4-map-add-const-plan",
	} {
		if !strings.Contains(mapI32.Reason+" "+mapI32.Evidence+" "+mapI32.Boundary, want) {
			t.Fatalf("map_i32 row missing %q: %#v", want, mapI32)
		}
	}
	if len(mapI32.MissingFacts) != 0 {
		t.Fatalf("map_i32 row reports missing facts after native SIMD validation: %#v", mapI32)
	}
	for _, want := range []string{"native_simd_codegen", "translation_differential_validation"} {
		if !containsString(mapI32.RequiredFacts, want) {
			t.Fatalf("map_i32 row missing required fact %q: %#v", want, mapI32)
		}
	}

	memsetMemcpy := byID[VectorizationMemsetMemcpy]
	if memsetMemcpy.Status != VectorizationImplementedNarrow ||
		memsetMemcpy.Decision != VectorizationVectorized {
		t.Fatalf(
			"memset_memcpy row = %#v, want implemented_narrow vectorized candidate",
			memsetMemcpy,
		)
	}
	if !memsetMemcpy.Candidate || !memsetMemcpy.RangeProof || memsetMemcpy.ProofID == "" {
		t.Fatalf("memset_memcpy row missing candidate/range-proof evidence: %#v", memsetMemcpy)
	}
	for _, want := range []string{
		"memset-loop",
		"memcpy helper via copy []u8",
		"zero-fill helper",
		"single mutable slice zero-fill",
		"safe unaligned",
		"vector backend lowering",
		"native SIMD",
		"linux-x64",
		"translation/differential",
		"scalar tail",
		"scalar-u8-memset-zero",
		"vector-u8x16-memset-zero-plan",
	} {
		if !strings.Contains(
			memsetMemcpy.Reason+" "+memsetMemcpy.Evidence+" "+memsetMemcpy.Boundary,
			want,
		) {
			t.Fatalf("memset_memcpy row missing %q: %#v", want, memsetMemcpy)
		}
	}
	if len(memsetMemcpy.MissingFacts) != 0 {
		t.Fatalf(
			"memset_memcpy row reports missing facts after native SIMD validation: %#v",
			memsetMemcpy,
		)
	}
	for _, want := range []string{"native_simd_codegen", "translation_differential_validation"} {
		if !containsString(memsetMemcpy.RequiredFacts, want) {
			t.Fatalf("memset_memcpy row missing required fact %q: %#v", want, memsetMemcpy)
		}
	}
}
