package opt

import (
	"os"
	"strings"
	"testing"
)

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
	if byID[CoreOptimizationConstantFolding].Status != CoreOptimizationImplementedNarrow || byID[CoreOptimizationConstantFolding].PassName != "basic-scalar" {
		t.Fatalf("constant folding row = %#v, want basic-scalar implemented_narrow", byID[CoreOptimizationConstantFolding])
	}
	if !strings.Contains(byID[CoreOptimizationConstantFolding].Boundary, "safe const-denominator div_i32/mod_i32 constants") {
		t.Fatalf("constant folding boundary missing safe div/mod constants: %#v", byID[CoreOptimizationConstantFolding])
	}
	if !strings.Contains(byID[CoreOptimizationConstantFolding].Boundary, "same-local comparison algebraic forms") {
		t.Fatalf("constant folding boundary missing same-local comparison algebra: %#v", byID[CoreOptimizationConstantFolding])
	}
	if !strings.Contains(byID[CoreOptimizationConstantFolding].Boundary, "denominators 0 and -1 remain rejected") {
		t.Fatalf("constant folding boundary missing unsafe denominator rejection: %#v", byID[CoreOptimizationConstantFolding])
	}
	if byID[CoreOptimizationCSEGvn].Status != CoreOptimizationImplementedNarrow || byID[CoreOptimizationCSEGvn].PassName != "basic-scalar" {
		t.Fatalf("CSE/GVN row = %#v, want basic-scalar implemented_narrow", byID[CoreOptimizationCSEGvn])
	}
	if !strings.Contains(byID[CoreOptimizationCSEGvn].Boundary, "commutative add/mul/eq/ne") {
		t.Fatalf("CSE/GVN boundary missing commutative local expression limit: %#v", byID[CoreOptimizationCSEGvn])
	}
	if !strings.Contains(byID[CoreOptimizationCSEGvn].Boundary, "mirrored lt/gt/le/ge") {
		t.Fatalf("CSE/GVN boundary missing mirrored ordered-comparison limit: %#v", byID[CoreOptimizationCSEGvn])
	}
	if !strings.Contains(byID[CoreOptimizationCSEGvn].Boundary, "local-load/constant") {
		t.Fatalf("CSE/GVN boundary missing local-constant expression limit: %#v", byID[CoreOptimizationCSEGvn])
	}
	if !strings.Contains(byID[CoreOptimizationCSEGvn].Boundary, "safe const-denominator div_i32/mod_i32") {
		t.Fatalf("CSE/GVN boundary missing safe division/modulo expression limit: %#v", byID[CoreOptimizationCSEGvn])
	}
	if !strings.Contains(byID[CoreOptimizationCSEGvn].Boundary, "unary local neg_i32") {
		t.Fatalf("CSE/GVN boundary missing unary local neg expression limit: %#v", byID[CoreOptimizationCSEGvn])
	}
	if !strings.Contains(byID[CoreOptimizationCSEGvn].Boundary, "safe known-local unary neg_i32 value expressions") {
		t.Fatalf("CSE/GVN boundary missing safe known-local unary value limit: %#v", byID[CoreOptimizationCSEGvn])
	}
	if !strings.Contains(byID[CoreOptimizationCSEGvn].Boundary, "overflow-sensitive unary neg_i32 min-int") {
		t.Fatalf("CSE/GVN boundary missing unsafe known-local unary rejection: %#v", byID[CoreOptimizationCSEGvn])
	}
	if !strings.Contains(byID[CoreOptimizationCSEGvn].Boundary, "safe known-local add_i32/sub_i32/mul_i32 value expressions") {
		t.Fatalf("CSE/GVN boundary missing safe known-local arithmetic value limit: %#v", byID[CoreOptimizationCSEGvn])
	}
	if !strings.Contains(byID[CoreOptimizationCSEGvn].Boundary, "overflow-sensitive known-local arithmetic") {
		t.Fatalf("CSE/GVN boundary missing unsafe known-local arithmetic rejection: %#v", byID[CoreOptimizationCSEGvn])
	}
	if !strings.Contains(byID[CoreOptimizationCSEGvn].Boundary, "safe known-local cmp_*_i32 value expressions") {
		t.Fatalf("CSE/GVN boundary missing safe known-local comparison value limit: %#v", byID[CoreOptimizationCSEGvn])
	}
	if !strings.Contains(byID[CoreOptimizationCSEGvn].Boundary, "source-local mutations that change known values") {
		t.Fatalf("CSE/GVN boundary missing source mutation rejection: %#v", byID[CoreOptimizationCSEGvn])
	}
	if !strings.Contains(byID[CoreOptimizationCSEGvn].Boundary, "safe known-local div_i32/mod_i32 value expressions") {
		t.Fatalf("CSE/GVN boundary missing safe known-local division/modulo value limit: %#v", byID[CoreOptimizationCSEGvn])
	}
	if !strings.Contains(byID[CoreOptimizationCSEGvn].Boundary, "unsafe known-local division/modulo") {
		t.Fatalf("CSE/GVN boundary missing unsafe known-local division/modulo rejection: %#v", byID[CoreOptimizationCSEGvn])
	}
	if byID[CoreOptimizationDCE].Status != CoreOptimizationImplementedNarrow || byID[CoreOptimizationDCE].PassName != "basic-scalar" {
		t.Fatalf("DCE row = %#v, want basic-scalar implemented_narrow", byID[CoreOptimizationDCE])
	}
	if !strings.Contains(byID[CoreOptimizationDCE].Boundary, "non-trapping comparison-expression producers") {
		t.Fatalf("DCE boundary missing non-trapping comparison expression limit: %#v", byID[CoreOptimizationDCE])
	}
	if !strings.Contains(byID[CoreOptimizationDCE].Boundary, "safe known-local unary neg_i32 producers") {
		t.Fatalf("DCE boundary missing safe unary neg producer limit: %#v", byID[CoreOptimizationDCE])
	}
	if !strings.Contains(byID[CoreOptimizationDCE].Boundary, "overflow-sensitive unary neg_i32 min-int") {
		t.Fatalf("DCE boundary missing unsafe unary neg rejection: %#v", byID[CoreOptimizationDCE])
	}
	if !strings.Contains(byID[CoreOptimizationDCE].Boundary, "safe known-local add_i32/sub_i32/mul_i32 producers") {
		t.Fatalf("DCE boundary missing safe known-local arithmetic producer limit: %#v", byID[CoreOptimizationDCE])
	}
	if !strings.Contains(byID[CoreOptimizationDCE].Boundary, "overflow-sensitive arithmetic") {
		t.Fatalf("DCE boundary missing unsafe arithmetic rejection: %#v", byID[CoreOptimizationDCE])
	}
	if !strings.Contains(byID[CoreOptimizationDCE].Boundary, "safe const-denominator div_i32/mod_i32 producers") {
		t.Fatalf("DCE boundary missing safe division/modulo expression limit: %#v", byID[CoreOptimizationDCE])
	}
	if !strings.Contains(byID[CoreOptimizationDCE].Boundary, "div_i32/mod_i32 denominators 0 and -1 are rejected") {
		t.Fatalf("DCE boundary missing unsafe denominator rejection: %#v", byID[CoreOptimizationDCE])
	}
	if !strings.Contains(byID[CoreOptimizationDCE].Boundary, "safe known-local div_i32/mod_i32 producers") {
		t.Fatalf("DCE boundary missing safe known-local division/modulo producer limit: %#v", byID[CoreOptimizationDCE])
	}
	if !strings.Contains(byID[CoreOptimizationDCE].Boundary, "unsafe division/modulo DCE") {
		t.Fatalf("DCE boundary missing unsafe division/modulo DCE non-claim: %#v", byID[CoreOptimizationDCE])
	}
	if byID[CoreOptimizationSCCP].Status != CoreOptimizationImplementedNarrow || byID[CoreOptimizationSCCP].PassName != "sccp-constant-branch" {
		t.Fatalf("SCCP row = %#v, want sccp-constant-branch implemented_narrow", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "known-local") {
		t.Fatalf("SCCP boundary missing known-local branch limit: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "constant binary-expression branch folding") {
		t.Fatalf("SCCP boundary missing constant expression branch limit: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "constant unary neg_i32") {
		t.Fatalf("SCCP boundary missing unary neg expression branch limit: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "stored safe unary neg_i32") {
		t.Fatalf("SCCP boundary missing stored unary neg fact limit: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "overflow-sensitive unary neg_i32 min-int") {
		t.Fatalf("SCCP boundary missing unsafe unary neg rejection: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "safe const-denominator div_i32/mod_i32") {
		t.Fatalf("SCCP boundary missing safe div/mod expression branch limit: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "denominators 0 and -1") {
		t.Fatalf("SCCP boundary missing unsafe div/mod denominator rejection: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "stored safe constant binary-expression facts") {
		t.Fatalf("SCCP boundary missing stored constant-expression fact limit: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "single-predecessor label propagation") {
		t.Fatalf("SCCP boundary missing single-predecessor label propagation limit: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "forward-terminated single-predecessor") {
		t.Fatalf("SCCP boundary missing forward single-predecessor propagation limit: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "folded zero-branch target propagation") {
		t.Fatalf("SCCP boundary missing folded zero-branch target propagation limit: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "folded nonzero-branch fallthrough propagation") {
		t.Fatalf("SCCP boundary missing folded nonzero fallthrough propagation limit: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "dynamic load_local zero-target and nonzero-fallthrough path facts") {
		t.Fatalf("SCCP boundary missing dynamic branch path-fact limit: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "dynamic zero-comparison eq/ne zero/nonzero path facts") {
		t.Fatalf("SCCP boundary missing dynamic zero-comparison path-fact limit: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "immediate label with no explicit incoming branch/jump edges") {
		t.Fatalf("SCCP boundary missing folded nonzero fallthrough-only label limit: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "folded nonzero fallthrough labels with explicit incoming edges") {
		t.Fatalf("SCCP boundary missing folded nonzero explicit-incoming rejection: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "multi-predecessor labels") {
		t.Fatalf("SCCP boundary missing multi-predecessor label rejection: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "fallthrough predecessors are rejected") {
		t.Fatalf("SCCP boundary missing fallthrough-predecessor rejection: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "dynamic zero-target labels with fallthrough predecessors") {
		t.Fatalf("SCCP boundary missing dynamic zero-target fallthrough rejection: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "dynamic comparison-target labels with fallthrough predecessors") {
		t.Fatalf("SCCP boundary missing dynamic comparison-target fallthrough rejection: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "dynamic stored expressions") {
		t.Fatalf("SCCP boundary missing dynamic stored-expression rejection: %#v", byID[CoreOptimizationSCCP])
	}
	if !strings.Contains(byID[CoreOptimizationSCCP].Boundary, "arbitrary comparison reasoning") {
		t.Fatalf("SCCP boundary missing arbitrary comparison reasoning non-claim: %#v", byID[CoreOptimizationSCCP])
	}
	if byID[CoreOptimizationMem2Reg].Status != CoreOptimizationImplementedNarrow || byID[CoreOptimizationMem2Reg].PassName != "mem2reg-single-assignment" {
		t.Fatalf("mem2reg row = %#v, want mem2reg-single-assignment implemented_narrow", byID[CoreOptimizationMem2Reg])
	}
	if !strings.Contains(byID[CoreOptimizationMem2Reg].Boundary, "stack-neutral separated") {
		t.Fatalf("mem2reg boundary missing separated stack-neutral temp limit: %#v", byID[CoreOptimizationMem2Reg])
	}
	if !strings.Contains(byID[CoreOptimizationMem2Reg].Boundary, "comparison-expression") {
		t.Fatalf("mem2reg boundary missing comparison-expression producer limit: %#v", byID[CoreOptimizationMem2Reg])
	}
	if !strings.Contains(byID[CoreOptimizationMem2Reg].Boundary, "safe const unary neg_i32") {
		t.Fatalf("mem2reg boundary missing safe unary neg producer limit: %#v", byID[CoreOptimizationMem2Reg])
	}
	if !strings.Contains(byID[CoreOptimizationMem2Reg].Boundary, "safe known-local unary neg_i32") {
		t.Fatalf("mem2reg boundary missing safe known-local unary neg producer limit: %#v", byID[CoreOptimizationMem2Reg])
	}
	if !strings.Contains(byID[CoreOptimizationMem2Reg].Boundary, "overflow-sensitive unary neg_i32 min-int") {
		t.Fatalf("mem2reg boundary missing unsafe unary neg rejection: %#v", byID[CoreOptimizationMem2Reg])
	}
	if !strings.Contains(byID[CoreOptimizationMem2Reg].Boundary, "safe const add_i32/sub_i32/mul_i32 arithmetic") {
		t.Fatalf("mem2reg boundary missing safe const arithmetic producer limit: %#v", byID[CoreOptimizationMem2Reg])
	}
	if !strings.Contains(byID[CoreOptimizationMem2Reg].Boundary, "safe known-local add_i32/sub_i32/mul_i32 arithmetic") {
		t.Fatalf("mem2reg boundary missing safe known-local arithmetic producer limit: %#v", byID[CoreOptimizationMem2Reg])
	}
	if !strings.Contains(byID[CoreOptimizationMem2Reg].Boundary, "arithmetic overflow") {
		t.Fatalf("mem2reg boundary missing unsafe arithmetic rejection: %#v", byID[CoreOptimizationMem2Reg])
	}
	if !strings.Contains(byID[CoreOptimizationMem2Reg].Boundary, "source-local mutation") {
		t.Fatalf("mem2reg boundary missing source-local mutation rejection: %#v", byID[CoreOptimizationMem2Reg])
	}
	if !strings.Contains(byID[CoreOptimizationMem2Reg].Boundary, "safe const-denominator div_i32/mod_i32 producer") {
		t.Fatalf("mem2reg boundary missing safe div/mod producer limit: %#v", byID[CoreOptimizationMem2Reg])
	}
	if !strings.Contains(byID[CoreOptimizationMem2Reg].Boundary, "safe known-local div_i32/mod_i32 producer") {
		t.Fatalf("mem2reg boundary missing safe known-local div/mod producer limit: %#v", byID[CoreOptimizationMem2Reg])
	}
	if !strings.Contains(byID[CoreOptimizationMem2Reg].Boundary, "denominators 0 and -1 are rejected") {
		t.Fatalf("mem2reg boundary missing unsafe denominator rejection: %#v", byID[CoreOptimizationMem2Reg])
	}
	if byID[CoreOptimizationLICM].Status != CoreOptimizationImplementedNarrow || byID[CoreOptimizationLICM].PassName != "licm-pure-invariant" {
		t.Fatalf("LICM row = %#v, want licm-pure-invariant implemented_narrow", byID[CoreOptimizationLICM])
	}
	if !strings.Contains(byID[CoreOptimizationLICM].Boundary, "add/sub/mul arithmetic") {
		t.Fatalf("LICM boundary missing pure invariant arithmetic limit: %#v", byID[CoreOptimizationLICM])
	}
	if !strings.Contains(byID[CoreOptimizationLICM].Boundary, "known-local add_i32/sub_i32/mul_i32 left-or-right operand") {
		t.Fatalf("LICM boundary missing known-local arithmetic operand limit: %#v", byID[CoreOptimizationLICM])
	}
	if !strings.Contains(byID[CoreOptimizationLICM].Boundary, "known-local cmp_*_i32 left-or-right operand") {
		t.Fatalf("LICM boundary missing known-local comparison operand limit: %#v", byID[CoreOptimizationLICM])
	}
	if !strings.Contains(byID[CoreOptimizationLICM].Boundary, "safe const-denominator div_i32/mod_i32") {
		t.Fatalf("LICM boundary missing safe division/modulo limit: %#v", byID[CoreOptimizationLICM])
	}
	if !strings.Contains(byID[CoreOptimizationLICM].Boundary, "safe known-local div_i32/mod_i32 denominator") {
		t.Fatalf("LICM boundary missing safe known-local division/modulo denominator limit: %#v", byID[CoreOptimizationLICM])
	}
	if !strings.Contains(byID[CoreOptimizationLICM].Boundary, "denominators 0 and -1") {
		t.Fatalf("LICM boundary missing unsafe denominator rejection: %#v", byID[CoreOptimizationLICM])
	}
	if !strings.Contains(byID[CoreOptimizationLICM].Boundary, "loop-mutated operands are rejected") {
		t.Fatalf("LICM boundary missing loop-mutated operand rejection: %#v", byID[CoreOptimizationLICM])
	}
}

func TestOptimizerCoreCoverageDocsRecordP17Closure(t *testing.T) {
	docs := []string{
		"../../../docs/audits/optimizer-core-coverage-v1.md",
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
	if generic.Status != InliningSpecializationImplementedNarrow || generic.PassName != "inline-small-pure" {
		t.Fatalf("generic row = %#v, want inline-small-pure implemented_narrow", generic)
	}
	for _, want := range []string{"monomorphized generic identity", "generic wrapper", "small_pure_wrapper", "no runtime generic values"} {
		if !strings.Contains(generic.Boundary+" "+generic.Evidence, want) {
			t.Fatalf("generic row missing %q: %#v", want, generic)
		}
	}
	smallPure := byID[InliningSpecializationSmallPureFunctions]
	if smallPure.Status != InliningSpecializationImplementedNarrow || smallPure.PassName != "inline-small-pure" {
		t.Fatalf("small-pure row = %#v, want inline-small-pure implemented_narrow", smallPure)
	}
	for _, want := range []string{"inlined", "not_inlined", "8", "proof-sensitive", "translation validation"} {
		if !strings.Contains(smallPure.Boundary+" "+smallPure.Evidence, want) {
			t.Fatalf("small-pure row missing %q: %#v", want, smallPure)
		}
	}
	enumKnown := byID[InliningSpecializationEnumKnownCase]
	if enumKnown.Status != InliningSpecializationImplementedNarrow || enumKnown.PassName != "sccp-constant-branch" {
		t.Fatalf("enum-known-case row = %#v, want sccp-constant-branch implemented_narrow", enumKnown)
	}
	for _, want := range []string{"payload enum constructor", "known-case match", "constant_stack_store", "translation validation", "no broad enum specialization"} {
		if !strings.Contains(enumKnown.Boundary+" "+enumKnown.Evidence, want) {
			t.Fatalf("enum-known-case row missing %q: %#v", want, enumKnown)
		}
	}
	optionalSome := byID[InliningSpecializationOptionalUnwrapProvenSome]
	if optionalSome.Status != InliningSpecializationImplementedNarrow || optionalSome.PassName != "sccp-constant-branch" {
		t.Fatalf("optional-proven-some row = %#v, want sccp-constant-branch implemented_narrow", optionalSome)
	}
	for _, want := range []string{"proven-some optional", "constant_stack_store", "translation validation", "no broad optional elimination"} {
		if !strings.Contains(optionalSome.Boundary+" "+optionalSome.Evidence, want) {
			t.Fatalf("optional-proven-some row missing %q: %#v", want, optionalSome)
		}
	}
	extension := byID[InliningSpecializationExtensionCalls]
	if extension.Status != InliningSpecializationImplementedNarrow || extension.PassName != "inline-small-pure" {
		t.Fatalf("extension-call row = %#v, want inline-small-pure implemented_narrow", extension)
	}
	for _, want := range []string{"statically resolved extension method", "direct Stack IR function symbol", "translation validation", "no dynamic extension dispatch"} {
		if !strings.Contains(extension.Boundary+" "+extension.Evidence, want) {
			t.Fatalf("extension-call row missing %q: %#v", want, extension)
		}
	}
	staticProtocol := byID[InliningSpecializationStaticProtocolConformanceCalls]
	if staticProtocol.Status != InliningSpecializationImplementedNarrow || staticProtocol.PassName != "inline-small-pure" {
		t.Fatalf("static protocol/conformance row = %#v, want inline-small-pure implemented_narrow", staticProtocol)
	}
	for _, want := range []string{"statically checked protocol impl", "known direct Stack IR function symbol", "translation validation", "no witness tables", "generic-bound requirement calls"} {
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
		if row.SourceEvidence == "" || row.OptimizedIREvidence == "" || row.MachineCodeEvidence == "" || row.Boundary == "" {
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
		{SpecializationMachineCodeGenerics, []string{"monomorphized generic identity", "generic wrapper", "optimized Stack IR has no call", "Machine IR contains no OpCall", "no runtime generic values"}},
		{SpecializationMachineCodeProtocolStaticConformance, []string{"statically checked protocol impl", "known direct Stack IR function symbol", "Machine IR contains no OpCall", "no witness tables", "dynamic dispatch"}},
		{SpecializationMachineCodeExtensionMethods, []string{"statically resolved extension method", "direct Stack IR function symbol", "Machine IR contains no OpCall", "no dynamic extension dispatch"}},
		{SpecializationMachineCodeEnumKnownCases, []string{"known-case match", "folded discriminator branch", "sccp-constant-branch", "machine code carries no match dispatch"}},
		{SpecializationMachineCodeOptionals, []string{"proven-some optional", "folded presence branch", "constant_stack_store", "machine code carries no optional dispatch"}},
		{SpecializationMachineCodeCollections, []string{"Vec<T>", "HashMap<K,V>", "monomorphized collection helper", "caller-owned", "Machine IR contains no OpCall", "no allocator-backed production"}},
	} {
		row := byID[check.id]
		haystack := row.Name + " " + row.SourceEvidence + " " + row.OptimizedIREvidence + " " + row.MachineCodeEvidence + " " + strings.Join(row.RemovedHighLevelMarkers, " ") + " " + row.Boundary
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
	if err := ValidateSpecializationMachineCodeCoverage(missingMachine); err == nil || !strings.Contains(err.Error(), "machine") {
		t.Fatalf("missing machine evidence validation err = %v", err)
	}
	fakeWitness := cloneSpecializationMachineCodeCoverage(report)
	fakeWitness.Witnesses[0].MachineIRHasCall = true
	if err := ValidateSpecializationMachineCodeCoverage(fakeWitness); err == nil || !strings.Contains(err.Error(), "witness") {
		t.Fatalf("fake witness validation err = %v", err)
	}
	placeholder := cloneSpecializationMachineCodeCoverage(report)
	placeholder.Rows[0].SourceEvidence = "TODO"
	if err := ValidateSpecializationMachineCodeCoverage(placeholder); err == nil || !strings.Contains(err.Error(), "placeholder") {
		t.Fatalf("placeholder validation err = %v", err)
	}
	fakeBroad := cloneSpecializationMachineCodeCoverage(report)
	fakeBroad.BroadSpecializationClaimed = true
	if err := ValidateSpecializationMachineCodeCoverage(fakeBroad); err == nil || !strings.Contains(err.Error(), "broad specialization") {
		t.Fatalf("fake broad specialization validation err = %v", err)
	}
	fakeDynamic := cloneSpecializationMachineCodeCoverage(report)
	fakeDynamic.DynamicDispatchClaimed = true
	if err := ValidateSpecializationMachineCodeCoverage(fakeDynamic); err == nil || !strings.Contains(err.Error(), "dynamic dispatch") {
		t.Fatalf("fake dynamic dispatch validation err = %v", err)
	}
	fakeRuntimeGenerics := cloneSpecializationMachineCodeCoverage(report)
	fakeRuntimeGenerics.RuntimeGenericValuesClaimed = true
	if err := ValidateSpecializationMachineCodeCoverage(fakeRuntimeGenerics); err == nil || !strings.Contains(err.Error(), "runtime generic") {
		t.Fatalf("fake runtime generics validation err = %v", err)
	}
	fakeCollections := cloneSpecializationMachineCodeCoverage(report)
	fakeCollections.AllocatorBackedCollectionsClaimed = true
	if err := ValidateSpecializationMachineCodeCoverage(fakeCollections); err == nil || !strings.Contains(err.Error(), "allocator-backed") {
		t.Fatalf("fake collection runtime validation err = %v", err)
	}
	fakeLayout := cloneSpecializationMachineCodeCoverage(report)
	fakeLayout.LayoutABIFreedomClaimed = true
	if err := ValidateSpecializationMachineCodeCoverage(fakeLayout); err == nil || !strings.Contains(err.Error(), "layout/ABI") {
		t.Fatalf("fake layout/ABI validation err = %v", err)
	}
	fakePerformance := cloneSpecializationMachineCodeCoverage(report)
	fakePerformance.PerformanceClaimed = true
	if err := ValidateSpecializationMachineCodeCoverage(fakePerformance); err == nil || !strings.Contains(err.Error(), "performance") {
		t.Fatalf("fake performance validation err = %v", err)
	}
	fakeSafeSemantics := cloneSpecializationMachineCodeCoverage(report)
	fakeSafeSemantics.SafeSemanticsChanged = true
	if err := ValidateSpecializationMachineCodeCoverage(fakeSafeSemantics); err == nil || !strings.Contains(err.Error(), "safe-program semantics") {
		t.Fatalf("fake safe-semantics validation err = %v", err)
	}
}

func TestP21SpecializationMachineCodeWitnessProvesDirectCallDisappearsBeforeMachineIR(t *testing.T) {
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

func cloneSpecializationMachineCodeCoverage(report SpecializationMachineCodeCoverageReport) SpecializationMachineCodeCoverageReport {
	out := report
	out.Rows = append([]SpecializationMachineCodeRow(nil), report.Rows...)
	out.Witnesses = append([]SpecializationMachineWitness(nil), report.Witnesses...)
	out.NonClaims = append([]string(nil), report.NonClaims...)
	for i := range out.Rows {
		out.Rows[i].Passes = append([]string(nil), report.Rows[i].Passes...)
		out.Rows[i].RemovedHighLevelMarkers = append([]string(nil), report.Rows[i].RemovedHighLevelMarkers...)
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
