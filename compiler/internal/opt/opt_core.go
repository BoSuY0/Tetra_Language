package opt

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/cache"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/islandkernel"
	"tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/machine"
	"tetra_language/compiler/internal/memoryfacts"
	"tetra_language/compiler/internal/semantics"
	"tetra_language/compiler/internal/ssair"
	"tetra_language/compiler/internal/validation"
)

// ---- coverage.go ----

type CoreOptimizationID string

const (
	CoreOptimizationConstantFolding        CoreOptimizationID = "constant_folding"
	CoreOptimizationCopyPropagation        CoreOptimizationID = "copy_propagation"
	CoreOptimizationDCE                    CoreOptimizationID = "dce"
	CoreOptimizationSCCP                   CoreOptimizationID = "sccp"
	CoreOptimizationCSEGvn                 CoreOptimizationID = "cse_gvn"
	CoreOptimizationMem2Reg                CoreOptimizationID = "mem2reg"
	CoreOptimizationSimpleInlining         CoreOptimizationID = "simple_inlining"
	CoreOptimizationLoopCanonicalization   CoreOptimizationID = "loop_canonicalization"
	CoreOptimizationLICM                   CoreOptimizationID = "licm"
	CoreOptimizationAllocationSinking      CoreOptimizationID = "allocation_sinking"
	CoreOptimizationScalarReplacement      CoreOptimizationID = "scalar_replacement"
	CoreOptimizationBoundsCheckElimination CoreOptimizationID = "bounds_check_elimination_v1"
)

type CoreOptimizationStatus string

const (
	CoreOptimizationImplementedNarrow CoreOptimizationStatus = "implemented_narrow"
	CoreOptimizationNotYetCovered     CoreOptimizationStatus = "not_yet_covered"
)

type CoreOptimizationCoverageReport struct {
	SchemaVersion string                        `json:"schema_version"`
	Rows          []CoreOptimizationCoverageRow `json:"rows"`
}

type CoreOptimizationCoverageRow struct {
	ID       CoreOptimizationID     `json:"id"`
	Name     string                 `json:"name"`
	Status   CoreOptimizationStatus `json:"status"`
	PassName string                 `json:"pass_name,omitempty"`
	Evidence string                 `json:"evidence"`
	Boundary string                 `json:"boundary"`
}

type InliningSpecializationID string

const (
	InliningSpecializationGenericFunctions InliningSpecializationID = "generic_functions"
	InliningSpecializationExtensionCalls   InliningSpecializationID = "extension_calls"
	InliningSpecializationEnumKnownCase    InliningSpecializationID = "enum_known_case"
)

const (
	InliningSpecializationSmallPureFunctions = InliningSpecializationID(
		"small_pure_functions",
	)
	InliningSpecializationStaticProtocolConformanceCalls = InliningSpecializationID(
		"static_protocol_conformance_calls",
	)
	InliningSpecializationOptionalUnwrapProvenSome = InliningSpecializationID(
		"optional_unwrap_proven_some",
	)
)

type InliningSpecializationStatus string

const (
	InliningSpecializationImplementedNarrow InliningSpecializationStatus = "implemented_narrow"
	InliningSpecializationNotYetCovered     InliningSpecializationStatus = "not_yet_covered"
)

type InliningSpecializationCoverageReport struct {
	SchemaVersion string                              `json:"schema_version"`
	Rows          []InliningSpecializationCoverageRow `json:"rows"`
}

type InliningSpecializationCoverageRow struct {
	ID       InliningSpecializationID     `json:"id"`
	Name     string                       `json:"name"`
	Status   InliningSpecializationStatus `json:"status"`
	PassName string                       `json:"pass_name,omitempty"`
	Evidence string                       `json:"evidence"`
	Boundary string                       `json:"boundary"`
}

func InliningSpecializationCoverage() InliningSpecializationCoverageReport {
	return InliningSpecializationCoverageReport{
		SchemaVersion: "tetra.optimizer.inlining_specialization.v1",
		Rows: []InliningSpecializationCoverageRow{
			{
				ID:       InliningSpecializationGenericFunctions,
				Name:     "generic functions",
				Status:   InliningSpecializationImplementedNarrow,
				PassName: "inline-small-pure",
				Evidence: ("compiler/tests/semantics/semantics_types_protocols_test.go::" +
					"TestP9GenericIdentityDisappearsAfterSmallPureInlining; " +
					"compiler/tests/semantics/semantics_types_protocols_test.go::" +
					"TestP17GenericWrapperDisappearsAfterSmallPureInlining; " +
					"compiler/internal/opt/opt_core.go"),
				Boundary: ("monomorphized generic identity and generic wrapper calls " +
					"may disappear only after static monomorphization when the " +
					"concrete Stack IR callee is accepted by inline-small-pure; " +
					"small_pure_wrapper is bounded by the same small body limit " +
					"and translation validation; no runtime generic values, " +
					"explicit type arguments, generic structs, dynamic dispatch, " +
					"or broad specialization optimization claim"),
			},
			{
				ID:       InliningSpecializationSmallPureFunctions,
				Name:     "small pure functions",
				Status:   InliningSpecializationImplementedNarrow,
				PassName: "inline-small-pure",
				Evidence: ("compiler/internal/opt/opt_core.go; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestInlineSmallPurePassInlinesCallAndReportsDecision; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestInlineSmallPurePassReportsNotInlinedReasons; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestInlineSmallPurePassDifferentialExecution"),
				Boundary: ("straight-line Stack IR functions with one return slot, no " +
					"proof-sensitive instructions, no unsupported effects, and " +
					"at most 8 candidate body instructions; reports inlined and " +
					"not_inlined reasons, preserves bounds proofs while " +
					"invalidating liveness, and uses translation validation; " +
					"recursive, effectful, proof-sensitive, control-flow, " +
					"call-containing non-wrapper, oversized, external/runtime, " +
					"and signature-mismatched callees remain calls"),
			},
			{
				ID:       InliningSpecializationStaticProtocolConformanceCalls,
				Name:     "static protocol/conformance calls",
				Status:   InliningSpecializationImplementedNarrow,
				PassName: "inline-small-pure",
				Evidence: ("compiler/tests/semantics/semantics_callables_closures_test.g" +
					"o::" +
					"TestP17StaticProtocolConformanceCallInlinesAfterSmallPure; " +
					"compiler/tests/semantics/semantics_types_protocols_test.go::" +
					"TestProtocolConformanceChecksExtensionMethod; " +
					"compiler/internal/layoutopt/layoutopt_test.go::" +
					"TestSpecializationDevirtualizesProtocolOnlyWhenTargetKnown; " +
					"compiler/internal/opt/opt_core.go"),
				Boundary: ("statically checked protocol impl method calls that lower to " +
					"a known direct Stack IR function symbol may disappear when " +
					"the concrete method body is accepted by inline-small-pure " +
					"with inlined report decisions and translation validation; " +
					"layout specialization decision evidence only devirtualizes " +
					"protocol calls when the target is known; no witness tables, " +
					"trait objects, runtime protocol values, dynamic dispatch, " +
					"conformance-table lookup, generic-bound requirement calls, " +
					"effectful or oversized conformance method inlining, or " +
					"broad protocol specialization claim"),
			},
			{
				ID:       InliningSpecializationExtensionCalls,
				Name:     "extension calls",
				Status:   InliningSpecializationImplementedNarrow,
				PassName: "inline-small-pure",
				Evidence: ("compiler/tests/semantics/semantics_callables_closures_test.g" +
					"o::TestP17StaticExtensionCallInlinesAfterSmallPure; " +
					"compiler/tests/semantics/semantics_core_language_test.go::" +
					"TestExtensionParseCheckAndLower; " +
					"compiler/internal/opt/opt_core.go"),
				Boundary: ("statically resolved extension method calls that lower to " +
					"direct Stack IR function symbols may disappear when the " +
					"concrete extension method body is accepted by " +
					"inline-small-pure with inlined report decisions and " +
					"translation validation; no dynamic extension dispatch, " +
					"receiver-call sugar specialization, protocol/witness " +
					"dispatch, effectful or oversized extension method inlining, " +
					"cross-control-flow specialization, or performance claim"),
			},
			{
				ID:       InliningSpecializationEnumKnownCase,
				Name:     "enum constructors/matches with known case",
				Status:   InliningSpecializationImplementedNarrow,
				PassName: "sccp-constant-branch",
				Evidence: ("compiler/tests/semantics/semantics_callables_closures_test.g" +
					"o::TestP17KnownEnumPayloadMatchFoldsAfterSCCP; " +
					"compiler/internal/opt/opt_core.go; " +
					"compiler/internal/lower/lower_suite_test.go::" +
					"TestLowerMatchExpressionEnumPayloadIR"),
				Boundary: ("payload enum constructor tag constants are tracked through " +
					"same-basic-block Stack IR stores with constant_stack_store " +
					"evidence, allowing a locally constructed known-case match " +
					"discriminator branch to fold through sccp-constant-branch " +
					"with translation validation; no broad enum specialization, " +
					"payload escape rewrite, cross-control-flow enum fact " +
					"propagation, exhaustive match pruning, or performance claim"),
			},
			{
				ID:       InliningSpecializationOptionalUnwrapProvenSome,
				Name:     "optional unwrap proven some",
				Status:   InliningSpecializationImplementedNarrow,
				PassName: "sccp-constant-branch",
				Evidence: ("compiler/tests/semantics/semantics_callables_closures_test.g" +
					"o::TestP17ProvenSomeOptionalMatchFoldsAfterSCCP; " +
					"compiler/internal/opt/opt_core.go; " +
					"compiler/tests/semantics/semantics_types_protocols_test.go::" +
					"TestOptionalMatchExhaustiveNoDefaultWithMultiSlotPayload"),
				Boundary: ("proven-some optional presence tags are tracked through " +
					"same-basic-block Stack IR stores with constant_stack_store " +
					"evidence, allowing a locally constructed optional value to " +
					"fold the lowered some match branch through " +
					"sccp-constant-branch with translation validation; no broad " +
					"optional elimination, unsafe unwrap removal, " +
					"cross-control-flow optional fact propagation, none-branch " +
					"pruning claim, or performance claim"),
			},
		},
	}
}

func CoreOptimizationCoverage() CoreOptimizationCoverageReport {
	return CoreOptimizationCoverageReport{
		SchemaVersion: "tetra.optimizer.core_coverage.v1",
		Rows: []CoreOptimizationCoverageRow{
			{
				ID:       CoreOptimizationConstantFolding,
				Name:     "constant folding",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "basic-scalar",
				Evidence: ("compiler/internal/opt/opt_core.go; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassFoldsSafeConstantsAndAlgebra; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassSimplifiesSameLocalComparisonAlgebra; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassFoldsSafeConstDenominatorDivModConstants;" +
					" compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassDoesNotFoldUnsafeConstDenominatorDivModCo" +
					"nstants"),
				Boundary: ("safe scalar i32 constants, same-local comparison algebraic " +
					"forms, safe const-denominator div_i32/mod_i32 constants, " +
					"and neutral-element algebraic forms only; " +
					"overflow-sensitive folds and div_i32/mod_i32 denominators 0 " +
					"and -1 remain rejected"),
			},
			{
				ID:       CoreOptimizationCopyPropagation,
				Name:     "copy propagation",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "basic-scalar",
				Evidence: ("compiler/internal/opt/opt_core.go; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassPropagatesCopiesAndEliminatesDeadStores"),
				Boundary: ("local copy chains in straight-line Stack IR; facts are " +
					"cleared across side effects and stores"),
			},
			{
				ID:       CoreOptimizationDCE,
				Name:     "DCE",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "basic-scalar",
				Evidence: ("compiler/internal/opt/opt_core.go; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassPropagatesCopiesAndEliminatesDeadStores; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassEliminatesDeadNonTrappingComparisonStore;" +
					" compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassEliminatesDeadSafeKnownLocalUnaryNegStore" +
					"; compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassDoesNotEliminateDeadUnsafeKnownLocalUnary" +
					"NegStore; compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassEliminatesDeadSafeKnownLocalArithmeticSto" +
					"re; compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassDoesNotEliminateDeadUnsafeKnownLocalArith" +
					"meticStore; compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassEliminatesDeadSafeConstDenominatorDivModS" +
					"tore; compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassEliminatesDeadSafeKnownLocalDivModStore; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassDoesNotEliminateDeadUnsafeKnownLocalDivMo" +
					"dStore"),
				Boundary: ("simple dead local stores with single pure producers, " +
					"non-trapping comparison-expression producers, safe " +
					"known-local unary neg_i32 producers, safe known-local " +
					"add_i32/sub_i32/mul_i32 producers, safe const-denominator " +
					"div_i32/mod_i32 producers, or safe known-local " +
					"div_i32/mod_i32 producers in straight-line Stack IR; " +
					"overflow-sensitive unary neg_i32 min-int, " +
					"overflow-sensitive arithmetic, and div_i32/mod_i32 " +
					"denominators 0 and -1 are rejected; no general DCE, " +
					"arbitrary arithmetic-expression DCE, arbitrary div/mod DCE, " +
					"or unsafe division/modulo DCE claim"),
			},
			{
				ID:       CoreOptimizationSCCP,
				Name:     "SCCP",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "sccp-constant-branch",
				Evidence: ("compiler/internal/opt/opt_core.go; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestSCCPPassFoldsConstantBranchesAndReportsDecisions; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestSCCPPassFoldsSafeUnaryNegExpressionBranch; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestSCCPPassDoesNotFoldUnsafeUnaryNegExpressionBranch; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestSCCPPassFoldsStoredSafeUnaryNegExpressionBranch; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestSCCPPassFoldsSafeConstDenominatorDivModExpressionBranch;" +
					" compiler/internal/opt/opt_suite_test.go::" +
					"TestSCCPPassDoesNotFoldUnsafeConstDenominatorDivModExpressio" +
					"nBranch; compiler/internal/opt/opt_suite_test.go::" +
					"TestSCCPPassFoldsStoredConstantExpressionBranch; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestSCCPPassFoldsStoredSafeConstDenominatorDivModExpressionB" +
					"ranch; compiler/internal/opt/opt_suite_test.go::" +
					"TestSCCPPassPropagatesKnownLocalThroughSinglePredecessorLabe" +
					"l; compiler/internal/opt/opt_suite_test.go::" +
					"TestSCCPPassPropagatesKnownLocalThroughForwardSinglePredeces" +
					"sorJump; compiler/internal/opt/opt_suite_test.go::" +
					"TestSCCPPassPropagatesKnownLocalThroughFoldedZeroBranchTarge" +
					"t; compiler/internal/opt/opt_suite_test.go::" +
					"TestSCCPPassDoesNotPropagateFoldedZeroBranchThroughFallthrou" +
					"ghTarget; compiler/internal/opt/opt_suite_test.go::" +
					"TestSCCPPassPropagatesKnownLocalThroughFoldedNonzeroFallthro" +
					"ughLabel; compiler/internal/opt/opt_suite_test.go::" +
					"TestSCCPPassDoesNotPropagateFoldedNonzeroFallthroughThroughE" +
					"xplicitIncomingLabel; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestSCCPPassPropagatesDynamicZeroFactThroughSinglePredecesso" +
					"rTarget; compiler/internal/opt/opt_suite_test.go::" +
					"TestSCCPPassUsesDynamicNonzeroFallthroughFactForRepeatedLoca" +
					"lBranch; compiler/internal/opt/opt_suite_test.go::" +
					"TestSCCPPassDoesNotPropagateDynamicZeroFactThroughFallthroug" +
					"hTarget; compiler/internal/opt/opt_suite_test.go::" +
					"TestSCCPPassDerivesEqZeroComparisonPathFacts; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestSCCPPassDerivesNeZeroComparisonPathFacts; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestSCCPPassDoesNotDeriveComparisonTargetFactThroughFallthro" +
					"ughTarget"),
				Boundary: ("literal constant-condition, same-basic-block known-local " +
					"including stored safe unary neg_i32 facts and stored safe " +
					"constant binary-expression facts, same-basic-block constant " +
					"unary neg_i32 and constant binary-expression branch folding " +
					"including safe const-denominator div_i32/mod_i32, immediate " +
					"or forward-terminated single-predecessor label propagation " +
					"for known-local facts, folded zero-branch target " +
					"propagation for labels with one incoming edge and no " +
					"fallthrough predecessor, folded nonzero-branch fallthrough " +
					"propagation through an immediate label with no explicit " +
					"incoming branch/jump edges, dynamic load_local zero-target " +
					"and nonzero-fallthrough path facts plus dynamic " +
					"zero-comparison eq/ne zero/nonzero path facts for later " +
					"same-local branches, plus fallthrough pruning to the next " +
					"label in Stack IR only; overflow-sensitive unary neg_i32 " +
					"min-int, div_i32/mod_i32 denominators 0 and -1, dynamic " +
					"stored expressions, multi-predecessor labels, folded " +
					"nonzero fallthrough labels with explicit incoming edges, " +
					"dynamic zero-target labels with fallthrough predecessors, " +
					"dynamic comparison-target labels with fallthrough " +
					"predecessors, and folded zero-branch target labels with " +
					"fallthrough predecessors are rejected; no general lattice " +
					"propagation, arbitrary path-sensitive optimization, range " +
					"propagation, arbitrary comparison reasoning, or " +
					"path-sensitive SSA SCCP claim"),
			},
			{
				ID:       CoreOptimizationCSEGvn,
				Name:     "CSE/GVN",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "basic-scalar",
				Evidence: ("compiler/internal/opt/opt_core.go; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassEliminatesRepeatedPureLocalExpressionWith" +
					"CSE; compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassEliminatesCommutativeLocalExpressionWithG" +
					"VN; compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassEliminatesMirroredComparisonExpressionWit" +
					"hGVN; compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassEliminatesRepeatedLocalConstantExpression" +
					"WithCSE; compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassEliminatesSafeConstDenominatorDivModExpre" +
					"ssionWithCSE; compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassEliminatesRepeatedUnaryLocalNegExpression" +
					"WithCSE; compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassEliminatesSafeKnownLocalUnaryNegExpressio" +
					"nWithCSE; compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassDoesNotReuseKnownLocalUnaryNegExpressionA" +
					"fterSourceMutation; compiler/internal/opt/opt_suite_test.go:" +
					":" +
					"TestBasicScalarPassDoesNotReuseMinIntKnownLocalUnaryNegExpre" +
					"ssion; compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassEliminatesSafeKnownLocalArithmeticExpress" +
					"ionWithGVN; compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassDoesNotReuseKnownLocalArithmeticExpressio" +
					"nAfterSourceMutation; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassDoesNotReuseOverflowSensitiveKnownLocalAr" +
					"ithmeticExpression; compiler/internal/opt/opt_suite_test.go:" +
					":" +
					"TestBasicScalarPassEliminatesSafeKnownLocalComparisonExpress" +
					"ionWithGVN; compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassDoesNotReuseKnownLocalComparisonExpressio" +
					"nAfterSourceMutation; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassEliminatesSafeKnownLocalDivModExpressionW" +
					"ithGVN; compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassDoesNotReuseKnownLocalDivModExpressionAft" +
					"erSourceMutation; compiler/internal/opt/opt_suite_test.go::" +
					"TestBasicScalarPassDoesNotReuseUnsafeKnownLocalDivModExpress" +
					"ion"),
				Boundary: ("reuses repeated pure local-load and local-load/constant " +
					"binary expressions, safe const-denominator div_i32/mod_i32 " +
					"expressions, unary local neg_i32 expressions, safe " +
					"known-local unary neg_i32 value expressions, safe " +
					"known-local add_i32/sub_i32/mul_i32 value expressions, safe " +
					"known-local cmp_*_i32 value expressions including mirrored " +
					"ordered comparisons, safe known-local div_i32/mod_i32 value " +
					"expressions, commutative add/mul/eq/ne operand variants, " +
					"plus mirrored lt/gt/le/ge ordered-comparison operand " +
					"variants only while operand/result value facts and cached " +
					"result locals remain valid; denominators 0 and -1, " +
					"source-local mutations that change known values, " +
					"overflow-sensitive unary neg_i32 min-int, " +
					"overflow-sensitive known-local arithmetic, and unsafe " +
					"known-local division/modulo are rejected; no global value " +
					"numbering or SSA GVN claim"),
			},
			{
				ID:       CoreOptimizationMem2Reg,
				Name:     "mem2reg",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "mem2reg-single-assignment",
				Evidence: ("compiler/internal/opt/opt_core.go; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestMem2RegPassPromotesSingleAssignmentTempAndReportsDecisio" +
					"n; compiler/internal/opt/opt_suite_test.go::" +
					"TestMem2RegPassPromotesSeparatedSingleAssignmentTempWithStac" +
					"kNeutralWork; compiler/internal/opt/opt_suite_test.go::" +
					"TestMem2RegPassPromotesSeparatedComparisonExpressionTempWith" +
					"StackNeutralWork; compiler/internal/opt/opt_suite_test.go::" +
					"TestMem2RegPassPromotesSeparatedSafeConstUnaryNegTempWithSta" +
					"ckNeutralWork; compiler/internal/opt/opt_suite_test.go::" +
					"TestMem2RegPassRejectsSeparatedUnsafeConstUnaryNegTemp; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestMem2RegPassPromotesSeparatedSafeKnownLocalUnaryNegTempWi" +
					"thStackNeutralWork; compiler/internal/opt/opt_suite_test.go:" +
					":" +
					"TestMem2RegPassRejectsSeparatedUnsafeKnownLocalUnaryNegTemp;" +
					" compiler/internal/opt/opt_suite_test.go::" +
					"TestMem2RegPassRejectsSeparatedSafeKnownLocalUnaryNegTempWhe" +
					"nSourceLocalMutates; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestMem2RegPassPromotesSeparatedSafeConstArithmeticTempWithS" +
					"tackNeutralWork; compiler/internal/opt/opt_suite_test.go::" +
					"TestMem2RegPassRejectsSeparatedUnsafeConstArithmeticTemp; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestMem2RegPassPromotesSeparatedSafeKnownLocalArithmeticTemp" +
					"WithStackNeutralWork; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestMem2RegPassRejectsSeparatedUnsafeKnownLocalArithmeticTem" +
					"p; compiler/internal/opt/opt_suite_test.go::" +
					"TestMem2RegPassRejectsSeparatedSafeKnownLocalArithmeticTempW" +
					"henSourceLocalMutates; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestMem2RegPassPromotesSeparatedSafeConstDenominatorDivModTe" +
					"mpWithStackNeutralWork; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestMem2RegPassRejectsSeparatedSafeDivModTempWhenSourceLocal" +
					"Mutates; compiler/internal/opt/opt_suite_test.go::" +
					"TestMem2RegPassRejectsSeparatedUnsafeConstDenominatorDivModT" +
					"emp; compiler/internal/opt/opt_suite_test.go::" +
					"TestMem2RegPassPromotesSeparatedSafeKnownLocalDivModTempWith" +
					"StackNeutralWork; compiler/internal/opt/opt_suite_test.go::" +
					"TestMem2RegPassRejectsSeparatedSafeKnownLocalDivModTempWhenS" +
					"ourceLocalMutates; compiler/internal/opt/opt_suite_test.go::" +
					"TestMem2RegPassRejectsSeparatedUnsafeKnownLocalDivModTemp"),
				Boundary: ("straight-line Stack IR single-store/single-load adjacent " +
					"temp locals plus stack-neutral separated single pure " +
					"const/load-local, bounded non-trapping " +
					"comparison-expression, safe const unary neg_i32, safe " +
					"known-local unary neg_i32, safe const " +
					"add_i32/sub_i32/mul_i32 arithmetic, safe known-local " +
					"add_i32/sub_i32/mul_i32 arithmetic, safe const-denominator " +
					"div_i32/mod_i32 producer temps, or safe known-local " +
					"div_i32/mod_i32 producer temps when source locals remain " +
					"unmodified; overflow-sensitive unary neg_i32 min-int, " +
					"arithmetic overflow, source-local mutation, and " +
					"div_i32/mod_i32 denominators 0 and -1 are rejected; no " +
					"alloca promotion, phi insertion, alias analysis, unsafe " +
					"unary neg promotion, unsafe arithmetic promotion, unsafe " +
					"division/modulo promotion, or general SSA mem2reg claim"),
			},
			{
				ID:       CoreOptimizationSimpleInlining,
				Name:     "simple inlining",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "inline-small-pure",
				Evidence: "compiler/internal/opt/opt_core.go; compiler/internal/opt/opt_suite_test.go",
				Boundary: ("small pure non-recursive Stack IR functions; effects, " +
					"recursion, calls, and proof-sensitive bodies are rejected"),
			},
			{
				ID:       CoreOptimizationLoopCanonicalization,
				Name:     "loop canonicalization",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "loop-canonicalization",
				Evidence: "compiler/internal/opt/opt_core.go; compiler/internal/opt/opt_suite_test.go",
				Boundary: "selected proof-tagged while-loop shapes with stable length locals",
			},
			{
				ID:       CoreOptimizationLICM,
				Name:     "LICM for pure invariant expressions",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "licm-pure-invariant",
				Evidence: ("compiler/internal/opt/opt_core.go; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestLICMPureInvariantPassHoistsPureComparisonInsideProofLoop" +
					"; compiler/internal/opt/opt_suite_test.go::" +
					"TestLICMPureInvariantPassHoistsPureArithmeticInsideProofLoop" +
					"; compiler/internal/opt/opt_suite_test.go::" +
					"TestLICMPureInvariantPassHoistsPureSubArithmeticInsideProofL" +
					"oop; compiler/internal/opt/opt_suite_test.go::" +
					"TestLICMPureInvariantPassHoistsSafeDivArithmeticInsideProofL" +
					"oop; compiler/internal/opt/opt_suite_test.go::" +
					"TestLICMPureInvariantPassHoistsSafeModArithmeticInsideProofL" +
					"oop; compiler/internal/opt/opt_suite_test.go::" +
					"TestLICMPureInvariantPassHoistsKnownLocalArithmeticInsidePro" +
					"ofLoop; compiler/internal/opt/opt_suite_test.go::" +
					"TestLICMPureInvariantPassRejectsKnownLocalArithmeticWhenOper" +
					"andMutatesInLoop; compiler/internal/opt/opt_suite_test.go::" +
					"TestLICMPureInvariantPassHoistsKnownLocalLeftArithmeticInsid" +
					"eProofLoop; compiler/internal/opt/opt_suite_test.go::" +
					"TestLICMPureInvariantPassRejectsKnownLocalLeftArithmeticWhen" +
					"OperandMutatesInLoop; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestLICMPureInvariantPassHoistsKnownLocalComparisonInsidePro" +
					"ofLoop; compiler/internal/opt/opt_suite_test.go::" +
					"TestLICMPureInvariantPassRejectsKnownLocalComparisonWhenOper" +
					"andMutatesInLoop; compiler/internal/opt/opt_suite_test.go::" +
					"TestLICMPureInvariantPassHoistsSafeKnownLocalDivModInsidePro" +
					"ofLoop; compiler/internal/opt/opt_suite_test.go::" +
					"TestLICMPureInvariantPassRejectsUnsafeKnownLocalDivModInside" +
					"ProofLoop; compiler/internal/opt/opt_suite_test.go::" +
					"TestLICMPureInvariantPassRejectsSafeKnownLocalDivModWhenDeno" +
					"minatorMutatesInLoop"),
				Boundary: ("pure load-local/constant comparison, add/sub/mul arithmetic," +
					" known-local add_i32/sub_i32/mul_i32 left-or-right operand " +
					"hoisting, known-local cmp_*_i32 left-or-right operand " +
					"hoisting, safe const-denominator div_i32/mod_i32 hoisting, " +
					"and safe known-local div_i32/mod_i32 denominator hoisting " +
					"for selected proof-tagged while-loop shapes only; " +
					"div_i32/mod_i32 denominators 0 and -1, loop-index operands, " +
					"and loop-mutated operands are rejected; no alias analysis, " +
					"overflow-sensitive safety beyond existing arithmetic LICM " +
					"semantics, unsafe division/modulo LICM, or general SSA LICM " +
					"claim"),
			},
			{
				ID:       CoreOptimizationAllocationSinking,
				Name:     "allocation sinking",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "lower.stack-allocation",
				Evidence: ("compiler/internal/lower/lower_suite_test.go; " +
					"compiler/internal/allocplan/plan_test.go"),
				Boundary: ("lowering/planner evidence for supported local stack/storage " +
					"elimination cases, not a general optimizer pass"),
			},
			{
				ID:       CoreOptimizationScalarReplacement,
				Name:     "scalar replacement",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "lower.scalar-replacement",
				Evidence: ("compiler/internal/lower/lower_suite_test.go::" +
					"TestLowerScalarReplacementEliminatesTinyConstantIndexSlice"),
				Boundary: ("tiny fixed constant-index slices, small structs, and fixed " +
					"arrays in supported lowering paths"),
			},
			{
				ID:       CoreOptimizationBoundsCheckElimination,
				Name:     "bounds-check elimination v1",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "plir-rangeproof-bce",
				Evidence: ("compiler/internal/lower/lower_suite_test.go; " +
					"compiler/internal/rangeproof; " +
					"compiler/internal/validation/validation.go"),
				Boundary: "proof-tagged supported branch, loop, view-chain, and copy-loop shapes only",
			},
		},
	}
}

// ---- hotloop.go ----

type HotLoopShapeReport struct {
	SchemaVersion string            `json:"schema_version"`
	Rows          []HotLoopShapeRow `json:"rows"`
	NonClaims     []string          `json:"non_claims"`
}

type HotLoopShapeRow struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	RegisterPath  bool     `json:"register_path"`
	SSAVerified   bool     `json:"ssa_verified"`
	MachinePath   string   `json:"machine_path"`
	MachineTarget string   `json:"machine_target,omitempty"`
	SpillFree     bool     `json:"spill_free"`
	StackChurnOps int      `json:"stack_churn_ops"`
	RequiredOps   []string `json:"required_ops,omitempty"`
	ProofID       string   `json:"proof_id,omitempty"`
	CallABI       string   `json:"call_abi,omitempty"`
	Reason        string   `json:"reason,omitempty"`
	Evidence      string   `json:"evidence"`
	Boundary      string   `json:"boundary"`
}

type hotLoopMachineLowerer func(ir.IRFunc) (machine.Function, bool, error)

func CoreHotLoopShapeEvidence() (HotLoopShapeReport, error) {
	rows := []HotLoopShapeRow{}
	scalar, err := hotLoopRegisterRow(
		"scalar-sum-loop",
		"scalar sum_n loop",
		hotLoopScalarSumFunc(),
		"machine-ir-loop",
		machine.ScalarIntLoopFunctionFromStackIR,
		("compiler/internal/machine/machine_core.go; " +
			"compiler/internal/machine/machine_suite_test.go::" +
			"TestScalarIntLoopFunctionFromStackIRLowersSumNLoop"),
		"canonical scalar i32 sum loop only; this is shape evidence, not throughput parity",
	)
	if err != nil {
		return HotLoopShapeReport{}, err
	}
	rows = append(rows, scalar)

	stride, err := hotLoopRegisterRow(
		"scalar-stride-sum-loop",
		"scalar constant-stride sum loop",
		hotLoopScalarStrideFunc(),
		"machine-ir-stride-loop",
		machine.ScalarIntLoopFunctionFromStackIR,
		("compiler/internal/machine/machine_core.go; " +
			"compiler/internal/machine/machine_suite_test.go::" +
			"TestScalarIntLoopFunctionFromStackIRLowersConstantStrideLoop"),
		("canonical scalar i32 sum loop with positive constant stride " +
			"2..127 only; this is shape evidence, not throughput parity"),
	)
	if err != nil {
		return HotLoopShapeReport{}, err
	}
	rows = append(rows, stride)

	squares, err := hotLoopRegisterRow(
		"scalar-sum-squares-loop",
		"scalar sum_squares loop",
		hotLoopSumSquaresFunc(),
		"machine-ir-sum-squares-loop",
		machine.ScalarIntSumSquaresLoopFunctionFromStackIR,
		("compiler/internal/machine/machine_core.go; " +
			"compiler/internal/machine/machine_suite_test.go::" +
			"TestScalarIntSumSquaresLoopFunctionFromStackIRLowersMulLoop"),
		"canonical scalar i32 sum-of-squares loop only; this is shape evidence, not throughput parity",
	)
	if err != nil {
		return HotLoopShapeReport{}, err
	}
	rows = append(rows, squares)

	product, err := hotLoopRegisterRow(
		"scalar-product-loop",
		"scalar product reduction loop",
		hotLoopProductFunc(),
		"machine-ir-product-loop",
		machine.ScalarIntProductLoopFunctionFromStackIR,
		("compiler/internal/machine/machine_core.go; " +
			"compiler/internal/machine/machine_suite_test.go::" +
			"TestScalarIntProductLoopFunctionFromStackIRLowersProductRedu" +
			"ctionLoop"),
		("canonical scalar i32 product reduction loop with product *= " +
			"index + 1 only; this is shape evidence, not throughput or " +
			"overflow-safety parity"),
	)
	if err != nil {
		return HotLoopShapeReport{}, err
	}
	rows = append(rows, product)

	max, err := hotLoopRegisterRow(
		"scalar-max-loop",
		"scalar max reduction loop",
		hotLoopMaxFunc(),
		"machine-ir-max-loop",
		machine.ScalarIntMaxLoopFunctionFromStackIR,
		("compiler/internal/machine/machine_core.go; " +
			"compiler/internal/machine/machine_suite_test.go::" +
			"TestScalarIntMaxLoopFunctionFromStackIRLowersBranchyMaxReduc" +
			"tionLoop"),
		("canonical scalar i32 max reduction loop with branchy max " +
			"update only; this is shape evidence, not throughput, " +
			"general min/max, or overflow-safety parity"),
	)
	if err != nil {
		return HotLoopShapeReport{}, err
	}
	rows = append(rows, max)

	affine, err := hotLoopRegisterRow(
		"scalar-affine-sum-loop",
		"scalar affine sum loop",
		hotLoopScalarAffineFunc(),
		"machine-ir-affine-loop",
		machine.ScalarIntAffineLoopFunctionFromStackIR,
		("compiler/internal/machine/machine_core.go; " +
			"compiler/internal/machine/machine_suite_test.go::" +
			"TestScalarIntAffineLoopFunctionFromStackIRLowersScaleBiasLoo" +
			"p"),
		("canonical scalar i32 affine sum loop with positive " +
			"compile-time scale and bias 1..127 only; this is shape " +
			"evidence, not throughput parity"),
	)
	if err != nil {
		return HotLoopShapeReport{}, err
	}
	rows = append(rows, affine)

	countdown, err := hotLoopRegisterRow(
		"scalar-countdown-loop",
		"scalar countdown loop",
		hotLoopCountdownFunc(),
		"machine-ir-countdown-loop",
		machine.ScalarIntCountdownLoopFunctionFromStackIR,
		("compiler/internal/machine/machine_core.go; " +
			"compiler/internal/machine/machine_suite_test.go::" +
			"TestScalarIntCountdownLoopFunctionFromStackIRLowersDescendin" +
			"gLoop"),
		"canonical scalar i32 countdown sum loop only; this is shape evidence, not throughput parity",
	)
	if err != nil {
		return HotLoopShapeReport{}, err
	}
	rows = append(rows, countdown)

	slice, err := hotLoopRegisterRow(
		"proof-slice-sum-loop",
		"proof-tagged i32 slice sum loop",
		hotLoopSliceSumFunc(true),
		"machine-ir-slice-sum",
		machine.ScalarI32SliceSumLoopFunctionFromStackIR,
		("compiler/internal/machine/machine_core.go; " +
			"compiler/internal/machine/machine_suite_test.go::" +
			"TestScalarI32SliceSumLoopFromStackIRRequiresProofTaggedUnche" +
			"ckedLoad"),
		("proof-tagged i32 slice sum loop only; checked/no-proof " +
			"index loads stay out of this register-shape claim"),
	)
	if err != nil {
		return HotLoopShapeReport{}, err
	}
	rows = append(rows, slice)

	sliceStride, err := hotLoopRegisterRow(
		"proof-slice-stride-sum-loop",
		"proof-tagged i32 slice constant-stride sum loop",
		hotLoopSliceStrideSumFunc(true),
		"machine-ir-slice-stride-sum",
		machine.ScalarI32SliceSumLoopFunctionFromStackIR,
		("compiler/internal/machine/machine_core.go; " +
			"compiler/internal/machine/machine_suite_test.go::" +
			"TestScalarI32SliceSumLoopFromStackIRLowersProofTaggedConstan" +
			"tStride"),
		("proof-tagged i32 slice sum loop with positive compile-time " +
			"stride 2..127 only; checked/no-proof index loads and " +
			"invalid strides stay out of this register-shape claim"),
	)
	if err != nil {
		return HotLoopShapeReport{}, err
	}
	rows = append(rows, sliceStride)

	call, err := hotLoopRegisterRow(
		"call-sum-loop",
		"scalar call loop",
		hotLoopCallSumFunc(),
		"machine-ir-call-loop",
		machine.ScalarIntCallLoopFunctionFromStackIR,
		("compiler/internal/machine/machine_core.go; " +
			"compiler/internal/machine/machine_suite_test.go::" +
			"TestScalarIntCallLoopFunctionFromStackIRLowersCallWithABIClo" +
			"bbers"),
		"single-i32 argument/result call loop with target ABI clobber metadata only",
	)
	if err != nil {
		return HotLoopShapeReport{}, err
	}
	rows = append(rows, call)

	rows = append(rows, hotLoopCheckedSliceFallbackRow())

	return HotLoopShapeReport{
		SchemaVersion: "tetra.optimizer.hot_loop_shape.v1",
		Rows:          rows,
		NonClaims: []string{
			"no C/Rust -O1/-O2 performance parity claim",
			"no general vectorization claim",
			"no unproven bounds-check removal claim",
		},
	}, nil
}

func hotLoopRegisterRow(
	id string,
	name string,
	fn ir.IRFunc,
	path string,
	lower hotLoopMachineLowerer,
	evidence string,
	boundary string,
) (HotLoopShapeRow, error) {
	ssaVerified, err := hotLoopSSAVerified(fn)
	if err != nil {
		return HotLoopShapeRow{}, err
	}
	mfn, ok, err := lower(fn)
	if err != nil {
		return HotLoopShapeRow{}, err
	}
	if !ok {
		return HotLoopShapeRow{}, fmt.Errorf("hot-loop shape: %s did not match %s", fn.Name, path)
	}
	if err := machine.VerifyFunction(mfn); err != nil {
		return HotLoopShapeRow{}, err
	}
	intervals, err := machine.BuildIntervals(mfn)
	if err != nil {
		return HotLoopShapeRow{}, err
	}
	alloc, err := machine.LinearScan(intervals, machine.LinuxX64CallerSaved())
	if err != nil {
		return HotLoopShapeRow{}, err
	}
	if alloc.Spills == nil {
		alloc.Spills = map[machine.VReg]int{}
	}
	if err := machine.VerifyAllocation(
		mfn,
		alloc,
		machine.LinuxX64CallerSaved(),
		len(alloc.Spills),
	); err != nil {
		return HotLoopShapeRow{}, err
	}
	return HotLoopShapeRow{
		ID:            id,
		Name:          name,
		RegisterPath:  true,
		SSAVerified:   ssaVerified,
		MachinePath:   path,
		MachineTarget: mfn.Target,
		SpillFree:     len(alloc.Spills) == 0,
		StackChurnOps: hotLoopStackChurnOps(mfn),
		RequiredOps:   hotLoopOps(mfn),
		ProofID:       hotLoopProofID(mfn),
		CallABI:       hotLoopCallABI(mfn),
		Evidence:      evidence,
		Boundary:      boundary,
	}, nil
}

func hotLoopCheckedSliceFallbackRow() HotLoopShapeRow {
	fn := hotLoopSliceSumFunc(false)
	ssaVerified, _ := hotLoopSSAVerified(fn)
	_, ok, err := machine.ScalarI32SliceSumLoopFunctionFromStackIR(fn)
	reason := "proof_tag_required_for_slice_sum_register_shape"
	if err != nil {
		reason = "machine_shape_error"
	} else if ok {
		reason = "unexpected_register_shape"
	}
	return HotLoopShapeRow{
		ID:           "checked-slice-sum-fallback",
		Name:         "checked i32 slice sum fallback",
		RegisterPath: false,
		SSAVerified:  ssaVerified,
		MachinePath:  "stack-fallback",
		Reason:       reason,
		Evidence: ("compiler/internal/machine/machine_suite_test.go::" +
			"TestScalarI32SliceSumLoopFromStackIRRequiresProofTaggedUnche" +
			"ckedLoad"),
		Boundary: ("slice-sum register shape requires proof-tagged unchecked " +
			"load; checked/no-proof shape remains stack fallback"),
	}
}

func hotLoopSliceStrideSumFunc(proof bool) ir.IRFunc {
	fn := hotLoopSliceSumFunc(proof)
	fn.Name = "sum_stride"
	fn.Instrs[17].Imm = 2
	return fn
}

func hotLoopSSAVerified(fn ir.IRFunc) (bool, error) {
	ssaFn, ok, err := ssair.FromStackIRFunction(fn)
	if err != nil || !ok {
		return false, err
	}
	if err := ssair.VerifyFunction(ssaFn); err != nil {
		return false, err
	}
	return true, nil
}

func hotLoopOps(fn machine.Function) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			op := string(instr.Op)
			if op == "" || seen[op] {
				continue
			}
			seen[op] = true
			out = append(out, op)
		}
	}
	return out
}

func hotLoopStackChurnOps(fn machine.Function) int {
	count := 0
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			switch instr.Op {
			case machine.OpSpill, machine.OpReload, machine.OpPush, machine.OpPop:
				count++
			}
		}
	}
	return count
}

func hotLoopProofID(fn machine.Function) string {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if strings.HasPrefix(instr.Note, "proof:") {
				return instr.Note
			}
		}
	}
	return ""
}

func hotLoopCallABI(fn machine.Function) string {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if instr.Op == machine.OpCall {
				return instr.ABI
			}
		}
	}
	return ""
}

func hotLoopScalarSumFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "sum_n",
		ParamSlots:  1,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func hotLoopScalarStrideFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "sum_stride",
		ParamSlots:  1,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func hotLoopSumSquaresFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "sum_squares",
		ParamSlots:  1,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func hotLoopProductFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "product_n",
		ParamSlots:  1,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func hotLoopMaxFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "max_n",
		ParamSlots:  1,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCmpGtI32},
			{Kind: ir.IRJmpIfZero, Label: 3},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLabel, Label: 3},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func hotLoopScalarAffineFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "sum_affine",
		ParamSlots:  1,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func hotLoopCountdownFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "sum_countdown",
		ParamSlots:  1,
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpGtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRSubI32},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func hotLoopCallSumFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "sum_call",
		ParamSlots:  1,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCall, Name: "inc", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func hotLoopSliceSumFunc(proof bool) ir.IRFunc {
	loadKind := ir.IRIndexLoadI32
	proofID := ""
	if proof {
		loadKind = ir.IRIndexLoadI32Unchecked
		proofID = "proof:while:i:xs:1:1"
	}
	return ir.IRFunc{
		Name:        "sum",
		ParamSlots:  2,
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: loadKind, ProofID: proofID},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

// ---- inlining.go ----

const inlineSmallPureMaxBodyInstrs = 8

type inlineSmallPureState struct {
	decisions []PassDecision
}

type inlineCandidate struct {
	fn         ir.IRFunc
	ok         bool
	reason     string
	bodyInstrs int
}

func InlineSmallPurePass() Pass {
	state := &inlineSmallPureState{}
	return Pass{
		Name:                      "inline-small-pure",
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
		ReportOutput:              "inline-small-pure.opt.json",
		ReportRows:                RequiredP17ReportRows(),
		NegativeTestMarker:        NegativeTestPassContractV1,
		ProfileInputPolicy:        ProfileInputUnused,
		Run:                       state.run,
		Decisions:                 state.reportDecisions,
	}
}

func (s *inlineSmallPureState) run(ctx *PassContext) error {
	prog := ctxProgram(ctx)
	if prog == nil {
		return fmt.Errorf("inline-small-pure: missing IR program")
	}
	s.decisions = nil
	funcs := make(map[string]ir.IRFunc, len(prog.Funcs))
	candidates := make(map[string]inlineCandidate, len(prog.Funcs))
	for _, fn := range prog.Funcs {
		funcs[fn.Name] = fn
		candidates[fn.Name] = analyzeInlineCandidate(fn)
	}
	promoteInlineWrapperCandidates(funcs, candidates)
	for i := range prog.Funcs {
		fn := &prog.Funcs[i]
		localSlots := fn.LocalSlots
		instrs := append([]ir.IRInstr(nil), fn.Instrs...)
		for {
			next := make([]ir.IRInstr, 0, len(instrs))
			changed := false
			for site, instr := range instrs {
				replacement, ok := s.rewriteInlineCall(
					fn.Name,
					instr,
					site,
					funcs,
					candidates,
					&localSlots,
				)
				if ok {
					next = append(next, replacement...)
					changed = true
					continue
				}
				next = append(next, instr)
			}
			instrs = next
			if !changed {
				break
			}
		}
		fn.LocalSlots = localSlots
		fn.Instrs = instrs
	}
	return nil
}

func (s *inlineSmallPureState) reportDecisions() []PassDecision {
	return append([]PassDecision(nil), s.decisions...)
}

func (s *inlineSmallPureState) rewriteInlineCall(
	caller string,
	instr ir.IRInstr,
	site int,
	funcs map[string]ir.IRFunc,
	candidates map[string]inlineCandidate,
	localSlots *int,
) ([]ir.IRInstr, bool) {
	if instr.Kind != ir.IRCall {
		return nil, false
	}
	target, ok := funcs[instr.Name]
	if !ok {
		s.notInlined(caller, instr.Name, site, "external_or_runtime")
		return nil, false
	}
	if caller == instr.Name {
		s.notInlined(caller, instr.Name, site, "recursive")
		return nil, false
	}
	candidate := candidates[target.Name]
	if !candidate.ok {
		s.notInlined(caller, instr.Name, site, candidate.reason)
		return nil, false
	}
	if instr.ArgSlots != target.ParamSlots || instr.RetSlots != target.ReturnSlots {
		s.notInlined(caller, instr.Name, site, "signature_mismatch")
		return nil, false
	}
	base := *localSlots
	*localSlots += target.LocalSlots
	replacement := make([]ir.IRInstr, 0, target.ParamSlots+len(target.Instrs)-1)
	for p := target.ParamSlots - 1; p >= 0; p-- {
		replacement = append(
			replacement,
			ir.IRInstr{Kind: ir.IRStoreLocal, Local: base + p, Pos: instr.Pos},
		)
	}
	for _, calleeInstr := range target.Instrs[:len(target.Instrs)-1] {
		replacement = append(replacement, remapInlineLocal(calleeInstr, base))
	}
	s.inlined(caller, instr.Name, site, candidate.reason)
	return replacement, true
}

func (s *inlineSmallPureState) inlined(caller string, callee string, site int, reason string) {
	s.decisions = append(s.decisions, PassDecision{
		Action: "inlined",
		Caller: caller,
		Callee: callee,
		Site:   site,
		Reason: reason,
	})
}

func (s *inlineSmallPureState) notInlined(caller string, callee string, site int, reason string) {
	s.decisions = append(s.decisions, PassDecision{
		Action: "not_inlined",
		Caller: caller,
		Callee: callee,
		Site:   site,
		Reason: reason,
	})
}

func analyzeInlineCandidate(fn ir.IRFunc) inlineCandidate {
	candidate := inlineCandidate{fn: fn}
	if fn.Policy.HasBudget || fn.Policy.HasConsent {
		candidate.reason = "unsupported_effect"
		return candidate
	}
	if fn.ReturnSlots != 1 {
		candidate.reason = "unsupported_return_slots"
		return candidate
	}
	if len(fn.Instrs) == 0 || fn.Instrs[len(fn.Instrs)-1].Kind != ir.IRReturn {
		candidate.reason = "control_flow"
		return candidate
	}
	if len(fn.Instrs)-1 > inlineSmallPureMaxBodyInstrs {
		candidate.reason = "not_small"
		return candidate
	}
	candidate.bodyInstrs = len(fn.Instrs) - 1
	for _, instr := range fn.Instrs[:len(fn.Instrs)-1] {
		if inlineInstrTouchesProof(instr) {
			candidate.reason = "proof_sensitive"
			return candidate
		}
		switch instr.Kind {
		case ir.IRConstI32, ir.IRLoadLocal, ir.IRStoreLocal,
			ir.IRAddI32, ir.IRSubI32, ir.IRMulI32, ir.IRNegI32,
			ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
			ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		case ir.IRCall:
			candidate.reason = "callee_contains_call"
			return candidate
		case ir.IRLabel, ir.IRJmp, ir.IRJmpIfZero:
			candidate.reason = "control_flow"
			return candidate
		default:
			candidate.reason = "unsupported_effect"
			return candidate
		}
	}
	candidate.ok = true
	candidate.reason = "small_pure"
	return candidate
}

func promoteInlineWrapperCandidates(
	funcs map[string]ir.IRFunc,
	candidates map[string]inlineCandidate,
) {
	for changed := true; changed; {
		changed = false
		for name, fn := range funcs {
			if candidates[name].ok {
				continue
			}
			candidate, ok := analyzeInlineWrapperCandidate(fn, candidates)
			if !ok {
				continue
			}
			candidates[name] = candidate
			changed = true
		}
	}
}

func analyzeInlineWrapperCandidate(
	fn ir.IRFunc,
	candidates map[string]inlineCandidate,
) (inlineCandidate, bool) {
	candidate := inlineCandidate{fn: fn}
	if fn.Policy.HasBudget || fn.Policy.HasConsent {
		candidate.reason = "unsupported_effect"
		return candidate, false
	}
	if fn.ReturnSlots != 1 {
		candidate.reason = "unsupported_return_slots"
		return candidate, false
	}
	if len(fn.Instrs) == 0 || fn.Instrs[len(fn.Instrs)-1].Kind != ir.IRReturn {
		candidate.reason = "control_flow"
		return candidate, false
	}
	bodyInstrs := 0
	hasInlineCall := false
	for _, instr := range fn.Instrs[:len(fn.Instrs)-1] {
		if inlineInstrTouchesProof(instr) {
			candidate.reason = "proof_sensitive"
			return candidate, false
		}
		switch instr.Kind {
		case ir.IRConstI32, ir.IRLoadLocal, ir.IRStoreLocal,
			ir.IRAddI32, ir.IRSubI32, ir.IRMulI32, ir.IRNegI32,
			ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
			ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
			bodyInstrs++
		case ir.IRCall:
			callee, ok := candidates[instr.Name]
			if !ok || !callee.ok {
				candidate.reason = "callee_contains_call"
				return candidate, false
			}
			bodyInstrs += callee.bodyInstrs
			hasInlineCall = true
		case ir.IRLabel, ir.IRJmp, ir.IRJmpIfZero:
			candidate.reason = "control_flow"
			return candidate, false
		default:
			candidate.reason = "unsupported_effect"
			return candidate, false
		}
	}
	if !hasInlineCall {
		return candidate, false
	}
	if bodyInstrs > inlineSmallPureMaxBodyInstrs {
		candidate.reason = "not_small"
		return candidate, false
	}
	candidate.ok = true
	candidate.reason = "small_pure_wrapper"
	candidate.bodyInstrs = bodyInstrs
	return candidate, true
}

func inlineInstrTouchesProof(instr ir.IRInstr) bool {
	if instr.ProofID != "" {
		return true
	}
	switch instr.Kind {
	case ir.IRIndexLoadI32Unchecked, ir.IRIndexLoadU8Unchecked, ir.IRIndexLoadU16Unchecked:
		return true
	default:
		return false
	}
}

func remapInlineLocal(instr ir.IRInstr, base int) ir.IRInstr {
	switch instr.Kind {
	case ir.IRLoadLocal, ir.IRStoreLocal:
		instr.Local += base
	}
	return instr
}

// ---- licm.go ----

type licmPureInvariantState struct {
	decisions []PassDecision
}

type invariantExpressionCandidate struct {
	start  int
	local  int
	reason string
}

func LICMPureInvariantPass() Pass {
	state := &licmPureInvariantState{}
	return Pass{
		Name:                      "licm-pure-invariant",
		InputKind:                 IRKindStack,
		OutputKind:                IRKindStack,
		InputVerifier:             VerifierLowerVerifyProgram,
		OutputVerifier:            VerifierLowerVerifyProgram,
		RequiredFacts:             []Fact{FactIRVerified, FactBoundsProofs},
		PreservedFacts:            []Fact{FactBoundsProofs},
		InvalidatedFacts:          []Fact{FactLiveness},
		RequiredProofKinds:        []memoryfacts.ProofKind{memoryfacts.ProofBounds},
		PreservedProofKinds:       []memoryfacts.ProofKind{memoryfacts.ProofBounds},
		ProofRule:                 ProofRulePreserveBoundsInvalidateLiveness,
		ValidationStrategy:        ValidationTranslation,
		TranslationValidationHook: TranslationHookValidateTranslation,
		ReportOutput:              "licm-pure-invariant.opt.json",
		ReportRows:                RequiredP17ReportRows(),
		NegativeTestMarker:        NegativeTestPassContractV1,
		ProfileInputPolicy:        ProfileInputUnused,
		Run:                       state.run,
		Decisions:                 state.reportDecisions,
	}
}

func (s *licmPureInvariantState) run(ctx *PassContext) error {
	prog := ctxProgram(ctx)
	if prog == nil {
		return fmt.Errorf("licm-pure-invariant: missing IR program")
	}
	s.decisions = nil
	for i := range prog.Funcs {
		s.rewriteFunc(ctx, &prog.Funcs[i])
	}
	return nil
}

func (s *licmPureInvariantState) reportDecisions() []PassDecision {
	return append([]PassDecision(nil), s.decisions...)
}

func (s *licmPureInvariantState) rewriteFunc(ctx *PassContext, fn *ir.IRFunc) {
	instrs := fn.Instrs
	out := make([]ir.IRInstr, 0, len(instrs))
	for i := 0; i < len(instrs); {
		candidate, reason, ok := analyzeSimpleLoop(instrs, i)
		if !ok {
			out = append(out, instrs[i])
			i++
			continue
		}
		if reason != "" {
			s.decisions = append(
				s.decisions,
				PassDecision{Action: "not_hoisted", Caller: fn.Name, Site: i, Reason: reason},
			)
			out = append(out, instrs[i])
			i++
			continue
		}
		invariant, reason, ok := findPureInvariantExpression(instrs, candidate)
		if !ok {
			s.decisions = append(
				s.decisions,
				PassDecision{Action: "not_hoisted", Caller: fn.Name, Site: i, Reason: reason},
			)
			out = append(out, instrs[i])
			i++
			continue
		}
		proofIDs := loopBoundsProofIDs(instrs[candidate.condJump+1 : candidate.backJump])
		proofFactIDs, decision, ok := ctx.requireMemoryProofs(
			fn.Name,
			proofIDs,
			memoryfacts.ProofBounds,
			RewriteLICM,
			invariant.start,
		)
		if !ok {
			s.decisions = append(s.decisions, *decision)
			out = append(out, instrs[i])
			i++
			continue
		}
		hoistedLocal := fn.LocalSlots
		fn.LocalSlots++
		out = append(out, hoistInvariantExpression(instrs, invariant, hoistedLocal)...)
		out = append(
			out,
			replaceInvariantExpression(
				instrs[candidate.labelIndex:candidate.backJump+1],
				invariant.start-candidate.labelIndex,
				hoistedLocal,
			)...)
		s.decisions = append(
			s.decisions,
			memoryRewriteDecision(
				"hoisted",
				fn.Name,
				invariant.start,
				invariant.reason,
				RewriteLICM,
				proofIDs,
				proofFactIDs,
			),
		)
		i = candidate.backJump + 1
	}
	fn.Instrs = out
}

func findPureInvariantExpression(
	instrs []ir.IRInstr,
	loop simpleLoopCandidate,
) (invariantExpressionCandidate, string, bool) {
	bodyStart := loop.condJump + 1
	bodyEnd := loop.backJump
	for i := bodyStart; i+2 < bodyEnd; i++ {
		if instrs[i].Kind != ir.IRLoadLocal {
			continue
		}
		switch instrs[i+1].Kind {
		case ir.IRConstI32:
			reason, blockReason, ok := pureInvariantExpressionDecision(
				instrs[i+2].Kind,
				instrs[i+1].Imm,
			)
			if blockReason != "" {
				return invariantExpressionCandidate{}, blockReason, false
			}
			if !ok {
				continue
			}
			local := instrs[i].Local
			if reason := invariantOperandBlockReason(instrs[bodyStart:bodyEnd], loop, local); reason != "" {
				return invariantExpressionCandidate{}, reason, false
			}
			return invariantExpressionCandidate{start: i, local: local, reason: reason}, "", true
		case ir.IRLoadLocal:
			reason, blockReason, ok := pureKnownLocalInvariantExpressionDecision(
				instrs[i+2].Kind,
				instrs[i].Local,
				instrs[i+1].Local,
				instrs,
				loop.labelIndex,
			)
			if blockReason != "" {
				return invariantExpressionCandidate{}, blockReason, false
			}
			if !ok {
				continue
			}
			for _, local := range []int{instrs[i].Local, instrs[i+1].Local} {
				if reason := invariantOperandBlockReason(instrs[bodyStart:bodyEnd], loop, local); reason != "" {
					return invariantExpressionCandidate{}, reason, false
				}
			}
			return invariantExpressionCandidate{
				start:  i,
				local:  instrs[i].Local,
				reason: reason,
			}, "", true
		}
	}
	return invariantExpressionCandidate{}, "no_pure_invariant_expression", false
}

func pureInvariantExpressionDecision(
	kind ir.IRInstrKind,
	constant int32,
) (reason string, blockReason string, ok bool) {
	switch kind {
	case ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32, ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		return "pure_invariant_comparison", "", true
	case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32:
		return "pure_invariant_arithmetic", "", true
	case ir.IRDivI32:
		if constant == 0 || constant == -1 {
			return "", "unsafe_division_denominator", false
		}
		return "pure_invariant_safe_division", "", true
	case ir.IRModI32:
		if constant == 0 || constant == -1 {
			return "", "unsafe_modulo_denominator", false
		}
		return "pure_invariant_safe_modulo", "", true
	default:
		return "", "", false
	}
}

func pureKnownLocalInvariantExpressionDecision(
	kind ir.IRInstrKind,
	leftLocal int,
	rightLocal int,
	instrs []ir.IRInstr,
	labelIndex int,
) (reason string, blockReason string, ok bool) {
	_, leftKnown := knownConstLocalInStraightLinePreheader(instrs, labelIndex, leftLocal)
	right, known := knownConstLocalInStraightLinePreheader(instrs, labelIndex, rightLocal)
	switch kind {
	case ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32, ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		if leftKnown || known {
			return "pure_invariant_known_local_comparison", "", true
		}
		return "", "", false
	case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32:
		if leftKnown || known {
			return "pure_invariant_known_local_arithmetic", "", true
		}
		return "", "", false
	case ir.IRDivI32:
		if !known {
			return "", "", false
		}
		if right == 0 || right == -1 {
			return "", "unsafe_known_local_division_denominator", false
		}
		return "pure_invariant_safe_known_local_division", "", true
	case ir.IRModI32:
		if !known {
			return "", "", false
		}
		if right == 0 || right == -1 {
			return "", "unsafe_known_local_modulo_denominator", false
		}
		return "pure_invariant_safe_known_local_modulo", "", true
	default:
		return "", "", false
	}
}

func knownConstLocalInStraightLinePreheader(
	instrs []ir.IRInstr,
	labelIndex int,
	local int,
) (int32, bool) {
	for i := labelIndex - 1; i >= 0; i-- {
		instr := instrs[i]
		switch instr.Kind {
		case ir.IRLabel, ir.IRJmp, ir.IRJmpIfZero:
			return 0, false
		case ir.IRStoreLocal:
			if instr.Local != local {
				continue
			}
			if i > 0 && instrs[i-1].Kind == ir.IRConstI32 {
				return instrs[i-1].Imm, true
			}
			return 0, false
		default:
			if clearsCopyFacts(instr.Kind) {
				return 0, false
			}
		}
	}
	return 0, false
}

func invariantOperandBlockReason(
	loopBody []ir.IRInstr,
	loop simpleLoopCandidate,
	local int,
) string {
	if local == loop.indexLocal {
		return "variant_loop_index_operand"
	}
	if localStoredInRange(loopBody, local) {
		return "loop_stores_invariant_operand"
	}
	return ""
}

func localStoredInRange(instrs []ir.IRInstr, local int) bool {
	for _, instr := range instrs {
		if instr.Kind == ir.IRStoreLocal && instr.Local == local {
			return true
		}
	}
	return false
}

func hoistInvariantExpression(
	instrs []ir.IRInstr,
	invariant invariantExpressionCandidate,
	hoistedLocal int,
) []ir.IRInstr {
	return []ir.IRInstr{
		instrs[invariant.start],
		instrs[invariant.start+1],
		instrs[invariant.start+2],
		{Kind: ir.IRStoreLocal, Local: hoistedLocal, Pos: instrs[invariant.start+2].Pos},
	}
}

func replaceInvariantExpression(
	loop []ir.IRInstr,
	relativeStart int,
	hoistedLocal int,
) []ir.IRInstr {
	out := make([]ir.IRInstr, 0, len(loop)-2)
	for i := 0; i < len(loop); i++ {
		if i == relativeStart {
			out = append(
				out,
				ir.IRInstr{Kind: ir.IRLoadLocal, Local: hoistedLocal, Pos: loop[i].Pos},
			)
			i += 2
			continue
		}
		out = append(out, loop[i])
	}
	return out
}

// ---- loop.go ----

type loopCanonicalizationState struct {
	decisions []PassDecision
}

type simpleLoopCandidate struct {
	labelIndex   int
	condJump     int
	backJump     int
	indexLocal   int
	lenLocal     int
	canonicalize bool
}

func LoopCanonicalizationPass() Pass {
	state := &loopCanonicalizationState{}
	return Pass{
		Name:                      "loop-canonicalization",
		InputKind:                 IRKindStack,
		OutputKind:                IRKindStack,
		InputVerifier:             VerifierLowerVerifyProgram,
		OutputVerifier:            VerifierLowerVerifyProgram,
		RequiredFacts:             []Fact{FactIRVerified, FactBoundsProofs},
		PreservedFacts:            []Fact{FactBoundsProofs},
		InvalidatedFacts:          []Fact{FactLiveness},
		RequiredProofKinds:        []memoryfacts.ProofKind{memoryfacts.ProofBounds},
		PreservedProofKinds:       []memoryfacts.ProofKind{memoryfacts.ProofBounds},
		ProofRule:                 ProofRulePreserveBoundsInvalidateLiveness,
		ValidationStrategy:        ValidationTranslation,
		TranslationValidationHook: TranslationHookValidateTranslation,
		ReportOutput:              "loop-canonicalization.opt.json",
		ReportRows:                RequiredP17ReportRows(),
		NegativeTestMarker:        NegativeTestPassContractV1,
		ProfileInputPolicy:        ProfileInputUnused,
		Run:                       state.run,
		Decisions:                 state.reportDecisions,
	}
}

func (s *loopCanonicalizationState) run(ctx *PassContext) error {
	prog := ctxProgram(ctx)
	if prog == nil {
		return fmt.Errorf("loop-canonicalization: missing IR program")
	}
	s.decisions = nil
	for i := range prog.Funcs {
		s.rewriteFunc(ctx, &prog.Funcs[i])
	}
	return nil
}

func (s *loopCanonicalizationState) reportDecisions() []PassDecision {
	return append([]PassDecision(nil), s.decisions...)
}

func (s *loopCanonicalizationState) rewriteFunc(ctx *PassContext, fn *ir.IRFunc) {
	instrs := fn.Instrs
	out := make([]ir.IRInstr, 0, len(instrs))
	for i := 0; i < len(instrs); {
		candidate, reason, ok := analyzeSimpleLoop(instrs, i)
		if !ok {
			out = append(out, instrs[i])
			i++
			continue
		}
		if reason != "" {
			s.decisions = append(
				s.decisions,
				PassDecision{Action: "not_hoisted", Caller: fn.Name, Site: i, Reason: reason},
			)
			out = append(out, instrs[i])
			i++
			continue
		}
		proofIDs := loopBoundsProofIDs(instrs[candidate.condJump+1 : candidate.backJump])
		proofFactIDs, decision, ok := ctx.requireMemoryProofs(
			fn.Name,
			proofIDs,
			memoryfacts.ProofBounds,
			RewriteBoundsCheckRemoval,
			i,
		)
		if !ok {
			s.decisions = append(s.decisions, *decision)
			out = append(out, instrs[i])
			i++
			continue
		}
		hoistedLocal := fn.LocalSlots
		fn.LocalSlots++
		out = append(
			out,
			rewriteSimpleLoop(
				instrs[candidate.labelIndex:candidate.backJump+1],
				candidate,
				hoistedLocal,
			)...)
		action := "hoisted"
		decisionReason := "stable_len_load"
		if candidate.canonicalize {
			action = "canonicalized"
			decisionReason = "stable_len_le_minus_one_to_lt"
		}
		s.decisions = append(
			s.decisions,
			memoryRewriteDecision(
				action,
				fn.Name,
				i,
				decisionReason,
				RewriteBoundsCheckRemoval,
				proofIDs,
				proofFactIDs,
			),
		)
		i = candidate.backJump + 1
	}
	fn.Instrs = out
}

func analyzeSimpleLoop(instrs []ir.IRInstr, labelIndex int) (simpleLoopCandidate, string, bool) {
	if labelIndex < 0 || labelIndex >= len(instrs) || instrs[labelIndex].Kind != ir.IRLabel {
		return simpleLoopCandidate{}, "", false
	}
	label := instrs[labelIndex].Label
	backJump := -1
	for i := labelIndex + 1; i < len(instrs); i++ {
		if instrs[i].Kind == ir.IRJmp && instrs[i].Label == label {
			backJump = i
			break
		}
	}
	if backJump < 0 {
		return simpleLoopCandidate{}, "", false
	}
	candidate, ok := matchLoopCondition(instrs, labelIndex, backJump)
	if !ok {
		return simpleLoopCandidate{}, "", false
	}
	if !loopHasWhileProofLoad(instrs[candidate.condJump+1 : backJump]) {
		return candidate, "missing_while_bounds_proof", true
	}
	if reason := loopMutationReason(
		instrs[candidate.condJump+1:backJump],
		candidate.lenLocal,
	); reason != "" {
		return candidate, reason, true
	}
	return candidate, "", true
}

func matchLoopCondition(
	instrs []ir.IRInstr,
	labelIndex int,
	backJump int,
) (simpleLoopCandidate, bool) {
	if labelIndex+4 < len(instrs) && labelIndex+4 < backJump &&
		instrs[labelIndex+1].Kind == ir.IRLoadLocal &&
		instrs[labelIndex+2].Kind == ir.IRLoadLocal &&
		instrs[labelIndex+3].Kind == ir.IRCmpLtI32 &&
		instrs[labelIndex+4].Kind == ir.IRJmpIfZero {
		return simpleLoopCandidate{
			labelIndex: labelIndex,
			condJump:   labelIndex + 4,
			backJump:   backJump,
			indexLocal: instrs[labelIndex+1].Local,
			lenLocal:   instrs[labelIndex+2].Local,
		}, true
	}
	if labelIndex+6 < len(instrs) && labelIndex+6 < backJump &&
		instrs[labelIndex+1].Kind == ir.IRLoadLocal &&
		instrs[labelIndex+2].Kind == ir.IRLoadLocal &&
		instrs[labelIndex+3].Kind == ir.IRConstI32 &&
		instrs[labelIndex+3].Imm == 1 &&
		instrs[labelIndex+4].Kind == ir.IRSubI32 &&
		instrs[labelIndex+5].Kind == ir.IRCmpLeI32 &&
		instrs[labelIndex+6].Kind == ir.IRJmpIfZero {
		return simpleLoopCandidate{
			labelIndex:   labelIndex,
			condJump:     labelIndex + 6,
			backJump:     backJump,
			indexLocal:   instrs[labelIndex+1].Local,
			lenLocal:     instrs[labelIndex+2].Local,
			canonicalize: true,
		}, true
	}
	return simpleLoopCandidate{}, false
}

func rewriteSimpleLoop(
	loop []ir.IRInstr,
	candidate simpleLoopCandidate,
	hoistedLocal int,
) []ir.IRInstr {
	if len(loop) == 0 {
		return nil
	}
	lenLoad := loop[2]
	out := []ir.IRInstr{
		{Kind: ir.IRLoadLocal, Local: candidate.lenLocal, Pos: lenLoad.Pos},
		{Kind: ir.IRStoreLocal, Local: hoistedLocal, Pos: lenLoad.Pos},
	}
	if candidate.canonicalize {
		label := loop[0]
		indexLoad := loop[1]
		hoistedLenLoad := loop[2]
		hoistedLenLoad.Local = hoistedLocal
		cmp := loop[5]
		cmp.Kind = ir.IRCmpLtI32
		out = append(out, label, indexLoad, hoistedLenLoad, cmp, loop[6])
		out = append(out, replaceLoopLenLoads(loop[7:], candidate.lenLocal, hoistedLocal)...)
		return out
	}
	out = append(out, replaceLoopLenLoads(loop, candidate.lenLocal, hoistedLocal)...)
	return out
}

func replaceLoopLenLoads(instrs []ir.IRInstr, lenLocal int, hoistedLocal int) []ir.IRInstr {
	out := append([]ir.IRInstr(nil), instrs...)
	for i := range out {
		if out[i].Kind == ir.IRLoadLocal && out[i].Local == lenLocal {
			out[i].Local = hoistedLocal
		}
	}
	return out
}

func loopHasWhileProofLoad(instrs []ir.IRInstr) bool {
	return len(loopBoundsProofIDs(instrs)) > 0
}

func loopBoundsProofIDs(instrs []ir.IRInstr) []string {
	var out []string
	for _, instr := range instrs {
		switch instr.Kind {
		case ir.IRIndexLoadI32Unchecked, ir.IRIndexLoadU8Unchecked, ir.IRIndexLoadU16Unchecked:
			if strings.HasPrefix(instr.ProofID, "proof:while:") {
				out = append(out, instr.ProofID)
			}
		}
	}
	return cleanProofIDs(out)
}

func loopMutationReason(instrs []ir.IRInstr, lenLocal int) string {
	for _, instr := range instrs {
		if instr.Kind == ir.IRStoreLocal && instr.Local == lenLocal {
			return "loop_stores_len_local"
		}
		if loopHasUnknownMutation(instr.Kind) {
			return "loop_has_unknown_mutation"
		}
	}
	return ""
}

func loopHasUnknownMutation(kind ir.IRInstrKind) bool {
	if clearsCopyFacts(kind) {
		return true
	}
	switch kind {
	case ir.IRWrite, ir.IRStrLit,
		ir.IRAllocBytes, ir.IRMakeSliceU8, ir.IRMakeSliceU16, ir.IRMakeSliceI32,
		ir.IRStackSliceU8, ir.IRStackSliceU16, ir.IRStackSliceI32,
		ir.IRRawSliceFromParts, ir.IRSliceWindow, ir.IRSlicePrefix, ir.IRSliceSuffix,
		ir.IRIslandNew, ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16,
		ir.IRIslandMakeSliceI32, ir.IRIslandFree, ir.IRIslandReset:
		return true
	default:
		return false
	}
}

// ---- manager.go ----

type IRKind string

const (
	IRKindStack     IRKind = "stack_ir"
	IRKindOptimized IRKind = "optimized_ir"
	IRKindMachine   IRKind = "machine_ir"
)

type Fact string

const (
	FactIRVerified   Fact = "ir_verified"
	FactBoundsProofs Fact = "bounds_proofs"
	FactLiveness     Fact = "liveness"
)

type ValidationStrategy string

const (
	ValidationTranslation ValidationStrategy = "translation_validation"
)

type ProfileInputPolicy string

const (
	ProfileInputUnused        ProfileInputPolicy = "unused"
	ProfileInputGuidedRewrite ProfileInputPolicy = "guided_rewrite"
)

const (
	VerifierLowerVerifyProgram         = "lower.VerifyProgram"
	VerifierMachineVerifyFunction      = "machine.VerifyFunction"
	TranslationHookValidateTranslation = "validation.ValidateTranslation"
	NegativeTestPassContractV1         = ("compiler/internal/opt/opt_suite_test.go::" +
		"TestManagerRejectsIncompletePassContractEvidence")
)

type ProofRule string

const (
	ProofRulePreserveBoundsProofs              ProofRule = "preserve_bounds_proofs"
	ProofRulePreserveBoundsInvalidateLiveness  ProofRule = "preserve_bounds_proofs_invalidate_liveness"
	ProofRuleInvalidateBoundsProofs            ProofRule = "invalidate_bounds_proofs"
	ProofRuleInvalidateBoundsProofsAndLiveness ProofRule = "invalidate_bounds_proofs_and_liveness"
)

type RewriteCategory string

const (
	RewriteBoundsCheckRemoval   RewriteCategory = "bounds_check_removal"
	RewriteNoAliasRewrite       RewriteCategory = "noalias_rewrite"
	RewriteTrustedAllocation    RewriteCategory = "trusted_allocation"
	RewriteScalarReplacement    RewriteCategory = "scalar_replacement"
	RewriteLICM                 RewriteCategory = "licm"
	RewriteInlineMemoryBoundary RewriteCategory = "inline_memory_boundary"
	RewriteRuntimeCheckErasure  RewriteCategory = "runtime_check_erasure"
)

type DecisionCode string

const (
	DecisionCodeRewriteApplied    DecisionCode = "optimizer:rewrite_applied"
	DecisionCodeProofMissing      DecisionCode = "optimizer:proof_missing"
	DecisionCodeProofMismatched   DecisionCode = "optimizer:proof_mismatched"
	DecisionCodeProofInvalidated  DecisionCode = "optimizer:proof_invalidated"
	DecisionCodeProofUnsafe       DecisionCode = "optimizer:proof_unsafe"
	DecisionCodeProofNotValidated DecisionCode = "optimizer:proof_not_validated"
)

type Pass struct {
	Name                      string
	InputKind                 IRKind
	OutputKind                IRKind
	InputVerifier             string
	OutputVerifier            string
	RequiredFacts             []Fact
	PreservedFacts            []Fact
	InvalidatedFacts          []Fact
	RequiredProofKinds        []memoryfacts.ProofKind
	PreservedProofKinds       []memoryfacts.ProofKind
	InvalidatedProofKinds     []memoryfacts.ProofKind
	ProofRule                 ProofRule
	ValidationStrategy        ValidationStrategy
	TranslationValidationHook string
	ReportOutput              string
	ReportRows                []string
	NegativeTestMarker        string
	ProfileInputPolicy        ProfileInputPolicy
	Run                       func(*PassContext) error
	Decisions                 func() []PassDecision
}

type Manager struct{}

type Options struct {
	OnlyPass     string
	ProfileInput *ProfileCollection
	MemoryFacts  memoryfacts.Snapshot
}

type Report struct {
	Passes               []PassReport      `json:"passes"`
	MemorySnapshotBefore string            `json:"memory_snapshot_before,omitempty"`
	MemorySnapshotAfter  string            `json:"memory_snapshot_after,omitempty"`
	MemoryDelta          memoryfacts.Delta `json:"memory_delta,omitempty"`
}

type (
	passFactSet              = []Fact
	passProfileInputEvidence = OptimizerProfileInputEvidence
	passTranslationReport    = validation.TranslationReport
	passOptimizationMetadata = validation.OptimizationValidationMetadata
	smCodeRows               = []SpecializationMachineCodeRow
	smCodeWitnesses          = []SpecializationMachineWitness
)

type PassReport struct {
	Name                      string                    `json:"name"`
	InputKind                 IRKind                    `json:"input_ir_kind"`
	OutputKind                IRKind                    `json:"output_ir_kind"`
	InputVerifier             string                    `json:"input_verifier"`
	OutputVerifier            string                    `json:"output_verifier"`
	RequiredFacts             passFactSet               `json:"required_facts,omitempty"`
	PreservedFacts            passFactSet               `json:"preserved_facts,omitempty"`
	InvalidatedFacts          passFactSet               `json:"invalidated_facts,omitempty"`
	RequiredProofKinds        []memoryfacts.ProofKind   `json:"required_proof_kinds,omitempty"`
	PreservedProofKinds       []memoryfacts.ProofKind   `json:"preserved_proof_kinds,omitempty"`
	InvalidatedProofKinds     []memoryfacts.ProofKind   `json:"invalidated_proof_kinds,omitempty"`
	ProofRule                 ProofRule                 `json:"proof_rule"`
	ValidationStrategy        ValidationStrategy        `json:"validation_strategy"`
	TranslationValidationHook string                    `json:"translation_validation_hook"`
	ReportOutput              string                    `json:"report_output"`
	ReportRows                []string                  `json:"report_rows"`
	NegativeTestMarker        string                    `json:"negative_test_marker"`
	ProfileInputPolicy        ProfileInputPolicy        `json:"profile_input_policy"`
	ProfileInput              *passProfileInputEvidence `json:"profile_input,omitempty"`
	BeforeDump                string                    `json:"before_dump"`
	AfterDump                 string                    `json:"after_dump"`
	VerifiedInput             bool                      `json:"verified_input"`
	VerifiedOutput            bool                      `json:"verified_output"`
	VerifiedProofs            bool                      `json:"verified_proofs"`
	TranslationValidated      bool                      `json:"translation_validated,omitempty"`
	TranslationReport         *passTranslationReport    `json:"translation_report,omitempty"`
	ValidationMetadata        *passOptimizationMetadata `json:"validation_metadata,omitempty"`
	Decisions                 []PassDecision            `json:"decisions,omitempty"`
	MemorySnapshotBefore      string                    `json:"memory_snapshot_before,omitempty"`
	MemorySnapshotAfter       string                    `json:"memory_snapshot_after,omitempty"`
	MemoryDelta               memoryfacts.Delta         `json:"memory_delta,omitempty"`
}

type PassDecision struct {
	Action          string          `json:"action"`
	Caller          string          `json:"caller,omitempty"`
	Callee          string          `json:"callee,omitempty"`
	Site            int             `json:"site"`
	Reason          string          `json:"reason"`
	DecisionCode    DecisionCode    `json:"decision_code,omitempty"`
	RewriteCategory RewriteCategory `json:"rewrite_category,omitempty"`
	ProofIDs        []string        `json:"proof_ids,omitempty"`
	ProofFactIDs    []string        `json:"proof_fact_ids,omitempty"`
}

type MemoryContext struct {
	Snapshot memoryfacts.Snapshot
	Enabled  bool
}

type PassContext struct {
	Program     *ir.IRProgram
	PassName    string
	Memory      *MemoryContext
	memoryDelta memoryfacts.Delta
}

func ctxProgram(ctx *PassContext) *ir.IRProgram {
	if ctx == nil {
		return nil
	}
	return ctx.Program
}

func newMemoryContext(snapshot memoryfacts.Snapshot) *MemoryContext {
	enabled := snapshot.ProgramID() != "" || len(snapshot.Facts()) > 0
	return &MemoryContext{Snapshot: snapshot, Enabled: enabled}
}

func (ctx *PassContext) requireMemoryProofs(
	function string,
	proofIDs []string,
	kind memoryfacts.ProofKind,
	category RewriteCategory,
	site int,
) ([]string, *PassDecision, bool) {
	if ctx == nil || ctx.Memory == nil || !ctx.Memory.Enabled {
		return nil, nil, true
	}
	cleanProofIDs := cleanProofIDs(proofIDs)
	if len(cleanProofIDs) == 0 {
		decision := memoryProofDecision(
			function,
			site,
			category,
			DecisionCodeProofMissing,
			nil,
			nil,
			"canonical proof id is required",
		)
		return nil, &decision, false
	}
	proofFactIDs := make([]string, 0, len(cleanProofIDs))
	for _, proofID := range cleanProofIDs {
		proof, ok := ctx.Memory.Snapshot.ResolveProof(memoryfacts.ProofQuery{
			FunctionID: function,
			ProofID:    proofID,
			Kind:       kind,
		})
		if !ok {
			code, factIDs := ctx.Memory.proofFailure(function, proofID, kind)
			decision := memoryProofDecision(
				function,
				site,
				category,
				code,
				[]string{proofID},
				factIDs,
				string(code),
			)
			return proofFactIDs, &decision, false
		}
		proofFactIDs = append(proofFactIDs, string(proof.FactID))
	}
	return proofFactIDs, nil, true
}

func (ctx *PassContext) InvalidateProof(factID memoryfacts.FactID, reason string) {
	if ctx == nil || factID == "" {
		return
	}
	if ctx.memoryDelta.Stage == "" {
		ctx.memoryDelta.Stage = memoryfacts.StageOptimization
	}
	ctx.memoryDelta.Invalidate = append(ctx.memoryDelta.Invalidate, memoryfacts.Invalidation{
		FactID: factID,
		Reason: strings.TrimSpace(reason),
	})
}

func (m *MemoryContext) proofFailure(
	function string,
	proofID string,
	kind memoryfacts.ProofKind,
) (DecisionCode, []string) {
	if m == nil {
		return DecisionCodeProofMissing, nil
	}
	facts := m.Snapshot.FactsForProof(memoryfacts.ProofKey{FunctionID: function, ProofID: proofID})
	if len(facts) == 0 {
		for _, fact := range m.Snapshot.Facts() {
			if fact.ProofID == proofID {
				facts = append(facts, fact)
			}
		}
	}
	if len(facts) == 0 {
		return DecisionCodeProofMissing, nil
	}
	factIDs := make([]string, 0, len(facts))
	code := DecisionCodeProofNotValidated
	for _, fact := range facts {
		factIDs = append(factIDs, string(fact.ID))
		switch {
		case fact.ValidationState == memoryfacts.ValidationInvalidated:
			return DecisionCodeProofInvalidated, factIDs
		case fact.ProofKind != kind:
			code = DecisionCodeProofMismatched
		case fact.ProvenanceClass == memoryfacts.ProvenanceUnsafeUnknown ||
			fact.UnsafeClass == memoryfacts.UnsafeUnknown:
			return DecisionCodeProofUnsafe, factIDs
		case fact.ValidationState != memoryfacts.ValidationPass ||
			strings.TrimSpace(fact.ValidatorName) == "":
			code = DecisionCodeProofNotValidated
		}
	}
	return code, factIDs
}

func memoryProofDecision(
	function string,
	site int,
	category RewriteCategory,
	code DecisionCode,
	proofIDs []string,
	proofFactIDs []string,
	reason string,
) PassDecision {
	return PassDecision{
		Action:          "skipped",
		Caller:          function,
		Site:            site,
		Reason:          reason,
		DecisionCode:    code,
		RewriteCategory: category,
		ProofIDs:        append([]string(nil), proofIDs...),
		ProofFactIDs:    append([]string(nil), proofFactIDs...),
	}
}

func memoryRewriteDecision(
	action string,
	function string,
	site int,
	reason string,
	category RewriteCategory,
	proofIDs []string,
	proofFactIDs []string,
) PassDecision {
	return PassDecision{
		Action:          action,
		Caller:          function,
		Site:            site,
		Reason:          reason,
		DecisionCode:    DecisionCodeRewriteApplied,
		RewriteCategory: category,
		ProofIDs:        append([]string(nil), cleanProofIDs(proofIDs)...),
		ProofFactIDs:    append([]string(nil), cleanStrings(proofFactIDs)...),
	}
}

func cleanProofIDs(ids []string) []string {
	return cleanStrings(ids)
}

func cleanStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func NewManager() Manager {
	return Manager{}
}

func RegisteredPasses() []Pass {
	return []Pass{
		BasicScalarPass(),
		SCCPPass(),
		Mem2RegPass(),
		InlineSmallPurePass(),
		LoopCanonicalizationPass(),
		LICMPureInvariantPass(),
	}
}

// Run is the legacy optimizer entry point for noncanonical tests and callers
// that do not participate in Memory Core evidence. Memory Core v2 production
// paths must use RunWithOptions with Options.MemoryFacts so proof-sensitive
// rewrites are resolved against the canonical memory snapshot.
func (m Manager) Run(prog *ir.IRProgram, passes ...Pass) (Report, error) {
	return m.RunWithOptions(prog, Options{}, passes...)
}

func (m Manager) RunWithOptions(prog *ir.IRProgram, opt Options, passes ...Pass) (Report, error) {
	selected, err := selectPassesForRun(opt, passes)
	if err != nil {
		return Report{}, err
	}
	var profileEvidence *OptimizerProfileInputEvidence
	if opt.ProfileInput != nil {
		evidence, err := BuildOptimizerProfileInputEvidence(*opt.ProfileInput)
		if err != nil {
			return Report{}, fmt.Errorf("optimizer profile input: %w", err)
		}
		profileEvidence = &evidence
	}
	return m.runSelected(prog, opt, profileEvidence, selected...)
}

func (m Manager) runSelected(
	prog *ir.IRProgram,
	opt Options,
	profileEvidence *OptimizerProfileInputEvidence,
	passes ...Pass,
) (Report, error) {
	report := Report{Passes: make([]PassReport, 0, len(passes))}
	memoryCtx := newMemoryContext(opt.MemoryFacts)
	if memoryCtx.Enabled {
		report.MemorySnapshotBefore = memoryCtx.Snapshot.Digest()
		report.MemorySnapshotAfter = report.MemorySnapshotBefore
		report.MemoryDelta.Stage = memoryfacts.StageOptimization
	}
	for passIndex, pass := range passes {
		if err := validatePassMetadata(pass); err != nil {
			return report, err
		}
		row := newPassReport(pass, profileEvidence)
		if memoryCtx.Enabled {
			row.MemorySnapshotBefore = memoryCtx.Snapshot.Digest()
		}
		if err := lower.VerifyProgram(prog); err != nil {
			return report, fmt.Errorf("%s input verification failed: %w", pass.Name, err)
		}
		row.VerifiedInput = true
		row.BeforeDump = FormatProgram(prog)
		before := cloneProgram(prog)
		passCtx := &PassContext{Program: prog, PassName: pass.Name, Memory: memoryCtx}
		if err := pass.Run(passCtx); err != nil {
			report.Passes = append(report.Passes, row)
			return report, fmt.Errorf("%s failed: %w", pass.Name, err)
		}
		if pass.Decisions != nil {
			row.Decisions = append([]PassDecision(nil), pass.Decisions()...)
		}
		row.MemoryDelta = passCtx.memoryDelta
		if err := validateMemoryDecisionEvidence(pass, row, memoryCtx); err != nil {
			report.Passes = append(report.Passes, row)
			return report, fmt.Errorf("%s memory decision evidence failed: %w", pass.Name, err)
		}
		row.AfterDump = FormatProgram(prog)
		if err := lower.VerifyProgram(prog); err != nil {
			report.Passes = append(report.Passes, row)
			return report, fmt.Errorf("%s output verification failed: %w", pass.Name, err)
		}
		row.VerifiedOutput = true
		if pass.ValidationStrategy == ValidationTranslation {
			translationReport, err := validation.ValidateTranslation(before, prog)
			if err != nil {
				report.Passes = append(report.Passes, row)
				return report, fmt.Errorf("%s translation validation failed: %w", pass.Name, err)
			}
			metadata, err := validation.BuildOptimizationValidationMetadata(
				before,
				prog,
				validation.OptimizationMetadataOptions{
					PassName:                  pass.Name,
					InputKind:                 string(pass.InputKind),
					OutputKind:                string(pass.OutputKind),
					InputVerifier:             pass.InputVerifier,
					OutputVerifier:            pass.OutputVerifier,
					ValidationStrategy:        string(pass.ValidationStrategy),
					RequiredFacts:             factStrings(pass.RequiredFacts),
					PreservedFacts:            factStrings(pass.PreservedFacts),
					InvalidatedFacts:          factStrings(pass.InvalidatedFacts),
					ProofRule:                 string(pass.ProofRule),
					TranslationValidationHook: pass.TranslationValidationHook,
					ReportRows:                append([]string(nil), pass.ReportRows...),
					NegativeTestMarker:        pass.NegativeTestMarker,
					ProfileInputPolicy:        string(pass.ProfileInputPolicy),
				},
			)
			if err != nil {
				report.Passes = append(report.Passes, row)
				return report, fmt.Errorf("%s validation metadata failed: %w", pass.Name, err)
			}
			if profileEvidence != nil {
				metadata.ProfileInputDigest = profileEvidence.Digest
				metadata.ProfileInputSchemaVersion = profileEvidence.SchemaVersion
				if err := validation.ValidateOptimizationValidationMetadata(metadata); err != nil {
					report.Passes = append(report.Passes, row)
					return report, fmt.Errorf("%s validation metadata failed: %w", pass.Name, err)
				}
			}
			row.TranslationValidated = true
			row.TranslationReport = &translationReport
			row.ValidationMetadata = &metadata
		}
		if _, err := validation.CheckBoundsProofs(prog); err != nil {
			report.Passes = append(report.Passes, row)
			return report, fmt.Errorf("%s proof verification failed: %w", pass.Name, err)
		}
		row.VerifiedProofs = true
		if memoryCtx.Enabled {
			passDelta := buildOptimizerMemoryDelta(passIndex, row)
			row.MemoryDelta = passDelta
			nextSnapshot, err := applyOptimizerDeltaToSnapshot(memoryCtx.Snapshot, passDelta)
			if err != nil {
				report.Passes = append(report.Passes, row)
				return report, fmt.Errorf("%s memory delta failed: %w", pass.Name, err)
			}
			memoryCtx.Snapshot = nextSnapshot
			row.MemorySnapshotAfter = nextSnapshot.Digest()
			report.MemoryDelta = mergeMemoryDeltas(report.MemoryDelta, passDelta)
			report.MemorySnapshotAfter = row.MemorySnapshotAfter
		}
		report.Passes = append(report.Passes, row)
	}
	return report, nil
}

func factStrings(facts []Fact) []string {
	out := make([]string, len(facts))
	for i, fact := range facts {
		out[i] = string(fact)
	}
	return out
}

func selectPassesForRun(opt Options, passes []Pass) ([]Pass, error) {
	if opt.OnlyPass == "" {
		return passes, nil
	}
	var selected []Pass
	for _, pass := range passes {
		if pass.Name == opt.OnlyPass {
			selected = append(selected, pass)
		}
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("optimizer pass %q not found", opt.OnlyPass)
	}
	if len(selected) > 1 {
		return nil, fmt.Errorf("optimizer pass %q is ambiguous", opt.OnlyPass)
	}
	return selected, nil
}

func validatePassMetadata(pass Pass) error {
	return ValidatePassContract(pass)
}

func ValidatePassContract(pass Pass) error {
	if pass.Name == "" {
		return fmt.Errorf("optimizer pass is missing name")
	}
	if pass.InputKind == "" {
		return fmt.Errorf("optimizer pass %q missing input IR kind", pass.Name)
	}
	if pass.OutputKind == "" {
		return fmt.Errorf("optimizer pass %q missing output IR kind", pass.Name)
	}
	if pass.InputVerifier == "" {
		return fmt.Errorf("optimizer pass %q missing input verifier", pass.Name)
	}
	if !supportsVerifier(pass.InputKind, pass.InputVerifier) {
		return fmt.Errorf(
			"optimizer pass %q unsupported input verifier %q for %s",
			pass.Name,
			pass.InputVerifier,
			pass.InputKind,
		)
	}
	if pass.OutputVerifier == "" {
		return fmt.Errorf("optimizer pass %q missing output verifier", pass.Name)
	}
	if !supportsVerifier(pass.OutputKind, pass.OutputVerifier) {
		return fmt.Errorf(
			"optimizer pass %q unsupported output verifier %q for %s",
			pass.Name,
			pass.OutputVerifier,
			pass.OutputKind,
		)
	}
	if pass.ProofRule == "" {
		return fmt.Errorf(
			"optimizer pass %q missing proof preservation or invalidation rule",
			pass.Name,
		)
	}
	if err := validateProofRule(pass); err != nil {
		return err
	}
	if err := validateProofKindMetadata(pass); err != nil {
		return err
	}
	if pass.ValidationStrategy == "" {
		return fmt.Errorf("optimizer pass %q missing validation strategy", pass.Name)
	}
	switch pass.ValidationStrategy {
	case ValidationTranslation:
	default:
		return fmt.Errorf(
			"optimizer pass %q must use translation validation strategy, got %q",
			pass.Name,
			pass.ValidationStrategy,
		)
	}
	if pass.TranslationValidationHook == "" {
		return fmt.Errorf("optimizer pass %q missing translation validation hook", pass.Name)
	}
	if pass.TranslationValidationHook != TranslationHookValidateTranslation {
		return fmt.Errorf(
			"optimizer pass %q unsupported translation validation hook %q",
			pass.Name,
			pass.TranslationValidationHook,
		)
	}
	if pass.ReportOutput == "" {
		return fmt.Errorf("optimizer pass %q missing report output", pass.Name)
	}
	if len(pass.ReportRows) == 0 {
		return fmt.Errorf("optimizer pass %q missing report rows", pass.Name)
	}
	for _, row := range RequiredP17ReportRows() {
		if !hasReportRow(pass.ReportRows, row) {
			return fmt.Errorf("optimizer pass %q missing required report row %q", pass.Name, row)
		}
	}
	if pass.NegativeTestMarker == "" {
		return fmt.Errorf("optimizer pass %q missing negative-test marker", pass.Name)
	}
	if pass.NegativeTestMarker != NegativeTestPassContractV1 {
		return fmt.Errorf(
			"optimizer pass %q unknown negative-test marker %q",
			pass.Name,
			pass.NegativeTestMarker,
		)
	}
	if pass.ProfileInputPolicy == "" {
		return fmt.Errorf("optimizer pass %q missing profile input policy", pass.Name)
	}
	switch pass.ProfileInputPolicy {
	case ProfileInputUnused:
	case ProfileInputGuidedRewrite:
		return fmt.Errorf(
			"optimizer pass %q profile-guided optimizer decisions require dedicated validation",
			pass.Name,
		)
	default:
		return fmt.Errorf(
			"optimizer pass %q unsupported profile input policy %q",
			pass.Name,
			pass.ProfileInputPolicy,
		)
	}
	if pass.Run == nil {
		return fmt.Errorf("optimizer pass %q is missing run function", pass.Name)
	}
	return nil
}

func validateProofKindMetadata(pass Pass) error {
	for _, preserved := range pass.PreservedProofKinds {
		for _, invalidated := range pass.InvalidatedProofKinds {
			if preserved == invalidated {
				return fmt.Errorf(
					"optimizer pass %q cannot preserve and invalidate proof kind %q",
					pass.Name,
					preserved,
				)
			}
		}
	}
	if passMentionsBoundsFacts(pass) && !passMentionsProofKind(pass, memoryfacts.ProofBounds) {
		return fmt.Errorf(
			"optimizer pass %q mentions bounds facts but lacks %q proof kind metadata",
			pass.Name,
			memoryfacts.ProofBounds,
		)
	}
	return nil
}

func passMentionsBoundsFacts(pass Pass) bool {
	return hasFact(pass.RequiredFacts, FactBoundsProofs) ||
		hasFact(pass.PreservedFacts, FactBoundsProofs) ||
		hasFact(pass.InvalidatedFacts, FactBoundsProofs)
}

func passMentionsProofKind(pass Pass, kind memoryfacts.ProofKind) bool {
	for _, candidate := range pass.RequiredProofKinds {
		if candidate == kind {
			return true
		}
	}
	for _, candidate := range pass.PreservedProofKinds {
		if candidate == kind {
			return true
		}
	}
	for _, candidate := range pass.InvalidatedProofKinds {
		if candidate == kind {
			return true
		}
	}
	return false
}

func RequiredP17ReportRows() []string {
	return []string{
		"input_verifier",
		"output_verifier",
		"proof_rule",
		"translation_validation_hook",
		"translation_report",
		"validation_metadata",
		"before_dump",
		"after_dump",
		"profile_input_policy",
	}
}

func validateMemoryDecisionEvidence(pass Pass, row PassReport, memoryCtx *MemoryContext) error {
	for _, decision := range row.Decisions {
		if decision.DecisionCode == DecisionCodeRewriteApplied &&
			decision.RewriteCategory != "" &&
			len(cleanProofIDs(decision.ProofIDs)) == 0 {
			return fmt.Errorf(
				"memory rewrite %q at site %d missing proof id",
				decision.RewriteCategory,
				decision.Site,
			)
		}
		if decision.DecisionCode == DecisionCodeRewriteApplied &&
			decision.RewriteCategory != "" &&
			memoryCtx != nil &&
			memoryCtx.Enabled {
			if err := validateCanonicalRewriteProofs(pass, decision, memoryCtx); err != nil {
				return err
			}
		}
	}
	if len(pass.InvalidatedProofKinds) > 0 && performedMemoryRewrite(row.Decisions) &&
		len(row.MemoryDelta.Invalidate) == 0 {
		return fmt.Errorf("invalidating memory rewrite missing memoryfacts invalidation delta")
	}
	return nil
}

func validateCanonicalRewriteProofs(
	pass Pass,
	decision PassDecision,
	memoryCtx *MemoryContext,
) error {
	proofIDs := cleanProofIDs(decision.ProofIDs)
	if len(proofIDs) == 0 {
		return nil
	}
	proofFactIDs := cleanStrings(decision.ProofFactIDs)
	proofFactIDSet := map[string]struct{}{}
	for _, factID := range proofFactIDs {
		proofFactIDSet[factID] = struct{}{}
	}
	kind := canonicalRewriteProofKind(pass, decision)
	for _, proofID := range proofIDs {
		proof, ok := resolveCanonicalRewriteProof(memoryCtx, decision.Caller, proofID, kind)
		if !ok {
			return fmt.Errorf(
				"memory rewrite %q at site %d has noncanonical proof id %q",
				decision.RewriteCategory,
				decision.Site,
				proofID,
			)
		}
		if _, ok := proofFactIDSet[string(proof.FactID)]; !ok {
			return fmt.Errorf(
				"memory rewrite %q at site %d missing canonical proof fact id %q",
				decision.RewriteCategory,
				decision.Site,
				proof.FactID,
			)
		}
	}
	return nil
}

func resolveCanonicalRewriteProof(
	memoryCtx *MemoryContext,
	function string,
	proofID string,
	kind memoryfacts.ProofKind,
) (memoryfacts.ProofEvidence, bool) {
	if memoryCtx == nil || !memoryCtx.Enabled {
		return memoryfacts.ProofEvidence{}, false
	}
	if kind != "" {
		return memoryCtx.Snapshot.ResolveProof(memoryfacts.ProofQuery{
			FunctionID: strings.TrimSpace(function),
			ProofID:    proofID,
			Kind:       kind,
		})
	}
	for _, fact := range memoryCtx.Snapshot.FactsForProof(memoryfacts.ProofKey{
		FunctionID: strings.TrimSpace(function),
		ProofID:    proofID,
	}) {
		if fact.ValidationState != memoryfacts.ValidationPass ||
			strings.TrimSpace(fact.ValidatorName) == "" ||
			fact.ProvenanceClass == memoryfacts.ProvenanceUnsafeUnknown ||
			fact.UnsafeClass == memoryfacts.UnsafeUnknown ||
			fact.ProofKind == "" {
			continue
		}
		return memoryfacts.ProofEvidence{
			FactID:        fact.ID,
			ProofID:       fact.ProofID,
			Kind:          fact.ProofKind,
			SubjectBaseID: fact.ProofSubjectBaseID,
			Operation:     fact.ProofOperation,
			IslandID:      fact.IslandID,
			Epoch:         fact.Epoch,
			ValidatorName: fact.ValidatorName,
			SourceStage:   fact.SourceStage,
		}, true
	}
	return memoryfacts.ProofEvidence{}, false
}

func canonicalRewriteProofKind(pass Pass, decision PassDecision) memoryfacts.ProofKind {
	if len(pass.RequiredProofKinds) == 1 {
		return pass.RequiredProofKinds[0]
	}
	switch decision.RewriteCategory {
	case RewriteBoundsCheckRemoval, RewriteLICM, RewriteRuntimeCheckErasure:
		return memoryfacts.ProofBounds
	case RewriteNoAliasRewrite:
		return memoryfacts.ProofNoAlias
	case RewriteTrustedAllocation, RewriteScalarReplacement:
		return memoryfacts.ProofNoEscape
	default:
		return ""
	}
}

func performedMemoryRewrite(decisions []PassDecision) bool {
	for _, decision := range decisions {
		if decision.DecisionCode == DecisionCodeRewriteApplied && decision.RewriteCategory != "" {
			return true
		}
	}
	return false
}

func buildOptimizerMemoryDelta(passIndex int, row PassReport) memoryfacts.Delta {
	delta := memoryfacts.Delta{Stage: memoryfacts.StageOptimization}
	delta.Add = append(delta.Add, optimizerPassFact(passIndex, row))
	for decisionIndex, decision := range row.Decisions {
		if decision.DecisionCode == "" &&
			decision.RewriteCategory == "" &&
			len(decision.ProofIDs) == 0 {
			continue
		}
		delta.Add = append(delta.Add, optimizerDecisionFact(passIndex, decisionIndex, row, decision))
	}
	delta.Invalidate = append(delta.Invalidate, row.MemoryDelta.Invalidate...)
	return delta
}

func optimizerPassFact(passIndex int, row PassReport) memoryfacts.Fact {
	return memoryfacts.Fact{
		ID:              memoryfacts.FactID(fmt.Sprintf("optimizer:pass:%03d:%s", passIndex, safeFactPart(row.Name))),
		FunctionID:      "",
		SourceStage:     memoryfacts.StageOptimization,
		Claim:           memoryfacts.ClaimOptimizerPass,
		ProvenanceClass: memoryfacts.ProvenanceSafeKnown,
		UnsafeClass:     memoryfacts.UnsafeSafe,
		Reason:          row.Name,
	}
}

func optimizerDecisionFact(
	passIndex int,
	decisionIndex int,
	row PassReport,
	decision PassDecision,
) memoryfacts.Fact {
	proofIDs := cleanProofIDs(decision.ProofIDs)
	proofFactIDs := cleanStrings(decision.ProofFactIDs)
	fact := memoryfacts.Fact{
		ID: memoryfacts.FactID(fmt.Sprintf(
			"optimizer:decision:%03d:%03d:%s:%d",
			passIndex,
			decisionIndex,
			safeFactPart(row.Name+"-"+decision.Action),
			decision.Site,
		)),
		FunctionID:      decision.Caller,
		SiteID:          fmt.Sprintf("%d", decision.Site),
		SourceStage:     memoryfacts.StageOptimization,
		Claim:           memoryfacts.ClaimOptimizerDecision,
		ProvenanceClass: memoryfacts.ProvenanceSafeKnown,
		UnsafeClass:     memoryfacts.UnsafeSafe,
		ProofID:         firstString(proofIDs),
		DecisionCode:    string(decision.DecisionCode),
		Reason: fmt.Sprintf(
			"%s site=%d action=%s reason=%s",
			row.Name,
			decision.Site,
			decision.Action,
			decision.Reason,
		),
	}
	if len(proofFactIDs) > 0 {
		fact.ParentFactID = memoryfacts.FactID(proofFactIDs[0])
	}
	if decision.DecisionCode == DecisionCodeProofUnsafe {
		fact.ProvenanceClass = memoryfacts.ProvenanceUnsafeUnknown
		fact.UnsafeClass = memoryfacts.UnsafeUnknown
		fact.AliasState = memoryfacts.AliasUnknownConservative
	}
	return fact
}

func applyOptimizerDeltaToSnapshot(
	snapshot memoryfacts.Snapshot,
	delta memoryfacts.Delta,
) (memoryfacts.Snapshot, error) {
	graph := memoryfacts.NewGraph(snapshot.ProgramID())
	for _, fact := range snapshot.Facts() {
		if _, err := graph.AddFact(fact); err != nil {
			return memoryfacts.Snapshot{}, err
		}
	}
	if err := graph.Apply(delta); err != nil {
		return memoryfacts.Snapshot{}, err
	}
	return graph.Snapshot()
}

func mergeMemoryDeltas(left memoryfacts.Delta, right memoryfacts.Delta) memoryfacts.Delta {
	if left.Stage == "" {
		left.Stage = right.Stage
	}
	left.Add = append(left.Add, right.Add...)
	left.Invalidate = append(left.Invalidate, right.Invalidate...)
	left.Attach = append(left.Attach, right.Attach...)
	left.Validate = append(left.Validate, right.Validate...)
	return left
}

func safeFactPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer(
		" ", "_",
		":", "_",
		"/", "_",
		"\\", "_",
		"\t", "_",
		"\n", "_",
	)
	return replacer.Replace(value)
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func supportsVerifier(kind IRKind, verifier string) bool {
	switch kind {
	case IRKindStack, IRKindOptimized:
		return verifier == VerifierLowerVerifyProgram
	case IRKindMachine:
		return verifier == VerifierMachineVerifyFunction
	default:
		return false
	}
}

func validateProofRule(pass Pass) error {
	switch pass.ProofRule {
	case ProofRulePreserveBoundsProofs:
		if !hasFact(pass.PreservedFacts, FactBoundsProofs) {
			return fmt.Errorf(
				"optimizer pass %q proof rule %q requires preserved fact %q",
				pass.Name,
				pass.ProofRule,
				FactBoundsProofs,
			)
		}
	case ProofRulePreserveBoundsInvalidateLiveness:
		if !hasFact(pass.PreservedFacts, FactBoundsProofs) {
			return fmt.Errorf(
				"optimizer pass %q proof rule %q requires preserved fact %q",
				pass.Name,
				pass.ProofRule,
				FactBoundsProofs,
			)
		}
		if !hasFact(pass.InvalidatedFacts, FactLiveness) {
			return fmt.Errorf(
				"optimizer pass %q proof rule %q requires invalidated fact %q",
				pass.Name,
				pass.ProofRule,
				FactLiveness,
			)
		}
	case ProofRuleInvalidateBoundsProofs:
		if !hasFact(pass.InvalidatedFacts, FactBoundsProofs) {
			return fmt.Errorf(
				"optimizer pass %q proof rule %q requires invalidated fact %q",
				pass.Name,
				pass.ProofRule,
				FactBoundsProofs,
			)
		}
	case ProofRuleInvalidateBoundsProofsAndLiveness:
		if !hasFact(pass.InvalidatedFacts, FactBoundsProofs) ||
			!hasFact(pass.InvalidatedFacts, FactLiveness) {
			return fmt.Errorf(
				"optimizer pass %q proof rule %q requires invalidated facts %q and %q",
				pass.Name,
				pass.ProofRule,
				FactBoundsProofs,
				FactLiveness,
			)
		}
	default:
		return fmt.Errorf(
			"optimizer pass %q unknown proof preservation or invalidation rule %q",
			pass.Name,
			pass.ProofRule,
		)
	}
	return nil
}

func hasFact(facts []Fact, want Fact) bool {
	for _, fact := range facts {
		if fact == want {
			return true
		}
	}
	return false
}

func hasReportRow(rows []string, want string) bool {
	for _, row := range rows {
		if row == want {
			return true
		}
	}
	return false
}

func newPassReport(pass Pass, profileEvidence *OptimizerProfileInputEvidence) PassReport {
	var profileInput *OptimizerProfileInputEvidence
	if profileEvidence != nil {
		copyEvidence := *profileEvidence
		copyEvidence.CounterKinds = append([]string(nil), profileEvidence.CounterKinds...)
		profileInput = &copyEvidence
	}
	return PassReport{
		Name:                      pass.Name,
		InputKind:                 pass.InputKind,
		OutputKind:                pass.OutputKind,
		InputVerifier:             pass.InputVerifier,
		OutputVerifier:            pass.OutputVerifier,
		RequiredFacts:             append([]Fact(nil), pass.RequiredFacts...),
		PreservedFacts:            append([]Fact(nil), pass.PreservedFacts...),
		InvalidatedFacts:          append([]Fact(nil), pass.InvalidatedFacts...),
		RequiredProofKinds:        append([]memoryfacts.ProofKind(nil), pass.RequiredProofKinds...),
		PreservedProofKinds:       append([]memoryfacts.ProofKind(nil), pass.PreservedProofKinds...),
		InvalidatedProofKinds:     append([]memoryfacts.ProofKind(nil), pass.InvalidatedProofKinds...),
		ProofRule:                 pass.ProofRule,
		ValidationStrategy:        pass.ValidationStrategy,
		TranslationValidationHook: pass.TranslationValidationHook,
		ReportOutput:              pass.ReportOutput,
		ReportRows:                append([]string(nil), pass.ReportRows...),
		NegativeTestMarker:        pass.NegativeTestMarker,
		ProfileInputPolicy:        pass.ProfileInputPolicy,
		ProfileInput:              profileInput,
	}
}

func FormatProgram(prog *ir.IRProgram) string {
	if prog == nil {
		return "program stack_ir <nil>\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "program stack_ir main:%s index:%d\n", prog.MainName, prog.MainIndex)
	for i, fn := range prog.Funcs {
		if i > 0 {
			fmt.Fprintln(&b)
		}
		fmt.Fprintf(
			&b,
			"func %s params:%d locals:%d returns:%d\n",
			fn.Name,
			fn.ParamSlots,
			fn.LocalSlots,
			fn.ReturnSlots,
		)
		for _, instr := range fn.Instrs {
			fmt.Fprintf(&b, "  %s", instrName(instr.Kind))
			switch instr.Kind {
			case ir.IRConstI32:
				fmt.Fprintf(&b, " %d", instr.Imm)
			case ir.IRLoadLocal, ir.IRStoreLocal, ir.IRLoadGlobal, ir.IRStoreGlobal:
				fmt.Fprintf(&b, " local:%d", instr.Local)
			case ir.IRCall:
				fmt.Fprintf(&b, " %s args:%d rets:%d", instr.Name, instr.ArgSlots, instr.RetSlots)
			case ir.IRLabel, ir.IRJmp, ir.IRJmpIfZero:
				fmt.Fprintf(&b, " label:%d", instr.Label)
			case ir.IRStrLit:
				fmt.Fprintf(&b, " bytes:%d", len(instr.Str))
			}
			if instr.ProofID != "" {
				fmt.Fprintf(&b, " proof:%s", instr.ProofID)
			}
			fmt.Fprintln(&b)
		}
	}
	return b.String()
}

func cloneProgram(prog *ir.IRProgram) *ir.IRProgram {
	if prog == nil {
		return nil
	}
	out := *prog
	out.Funcs = make([]ir.IRFunc, len(prog.Funcs))
	for i, fn := range prog.Funcs {
		out.Funcs[i] = fn
		out.Funcs[i].Instrs = append([]ir.IRInstr(nil), fn.Instrs...)
	}
	return &out
}

func instrName(kind ir.IRInstrKind) string {
	switch kind {
	case ir.IRWrite:
		return "write"
	case ir.IRStrLit:
		return "str_lit"
	case ir.IRConstI32:
		return "const_i32"
	case ir.IRLoadLocal:
		return "load_local"
	case ir.IRStoreLocal:
		return "store_local"
	case ir.IRLoadGlobal:
		return "load_global"
	case ir.IRStoreGlobal:
		return "store_global"
	case ir.IRAddI32:
		return "add_i32"
	case ir.IRSubI32:
		return "sub_i32"
	case ir.IRNegI32:
		return "neg_i32"
	case ir.IRCmpEqI32:
		return "cmp_eq_i32"
	case ir.IRCmpLtI32:
		return "cmp_lt_i32"
	case ir.IRMulI32:
		return "mul_i32"
	case ir.IRDivI32:
		return "div_i32"
	case ir.IRModI32:
		return "mod_i32"
	case ir.IRCmpGtI32:
		return "cmp_gt_i32"
	case ir.IRCmpGeI32:
		return "cmp_ge_i32"
	case ir.IRCmpLeI32:
		return "cmp_le_i32"
	case ir.IRCmpNeI32:
		return "cmp_ne_i32"
	case ir.IRCall:
		return "call"
	case ir.IRLabel:
		return "label"
	case ir.IRJmp:
		return "jmp"
	case ir.IRJmpIfZero:
		return "jmp_if_zero"
	case ir.IRReturn:
		return "return"
	default:
		return fmt.Sprintf("ir_%d", kind)
	}
}

// ---- mem2reg.go ----

type mem2regState struct {
	decisions []PassDecision
}

type mem2regProducerKind string

const (
	mem2regProducerSingleValue          mem2regProducerKind = "single_value"
	mem2regProducerComparisonExpression mem2regProducerKind = "comparison_expression"
)

const (
	mem2regProducerSafeConstUnaryNegExpression = mem2regProducerKind(
		"safe_const_unary_neg_expression",
	)
	mem2regProducerSafeKnownLocalUnaryNegExpression = mem2regProducerKind(
		"safe_known_local_unary_neg_expression",
	)
	mem2regProducerSafeConstArithmeticExpression = mem2regProducerKind(
		"safe_const_arithmetic_expression",
	)
	mem2regProducerSafeKnownLocalArithmeticExpression mem2regProducerKind = ("safe_known_local_" +
		"arithmetic_expression")
	mem2regProducerSafeConstDenominatorDivModExpression = mem2regProducerKind(
		"safe_const_denominator_" +
			"divmod_expression")
	mem2regProducerSafeKnownLocalDivModExpression = mem2regProducerKind(
		"safe_known_local_divmod_expression",
	)
)

type mem2regProducer struct {
	Kind         mem2regProducerKind
	Instrs       []ir.IRInstr
	SourceLocals map[int]struct{}
}

func Mem2RegPass() Pass {
	state := &mem2regState{}
	return Pass{
		Name:                      "mem2reg-single-assignment",
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
		ReportOutput:              "mem2reg-single-assignment.opt.json",
		ReportRows:                RequiredP17ReportRows(),
		NegativeTestMarker:        NegativeTestPassContractV1,
		ProfileInputPolicy:        ProfileInputUnused,
		Run:                       state.run,
		Decisions:                 state.reportDecisions,
	}
}

func (s *mem2regState) run(ctx *PassContext) error {
	prog := ctxProgram(ctx)
	if prog == nil {
		return fmt.Errorf("mem2reg-single-assignment: missing IR program")
	}
	s.decisions = nil
	for i := range prog.Funcs {
		fn := &prog.Funcs[i]
		if fn.Policy.HasBudget || fn.Policy.HasConsent {
			s.decisions = append(
				s.decisions,
				PassDecision{
					Action: "not_promoted",
					Caller: fn.Name,
					Site:   0,
					Reason: "policy_guarded_function",
				},
			)
			continue
		}
		if hasMem2RegControlFlow(fn.Instrs) {
			if hasAdjacentStoreLoad(fn.Instrs) {
				s.decisions = append(
					s.decisions,
					PassDecision{
						Action: "not_promoted",
						Caller: fn.Name,
						Site:   0,
						Reason: "control_flow_function",
					},
				)
			}
			continue
		}
		fn.Instrs = s.rewriteFunc(fn.Name, fn.ParamSlots, fn.Instrs)
	}
	return nil
}

func (s *mem2regState) reportDecisions() []PassDecision {
	return append([]PassDecision(nil), s.decisions...)
}

func (s *mem2regState) rewriteFunc(
	fnName string,
	paramSlots int,
	instrs []ir.IRInstr,
) []ir.IRInstr {
	stores, loads := localUsageCounts(instrs)
	loadIndexes := localLoadIndexes(instrs)
	replacements := map[int][]ir.IRInstr{}
	out := make([]ir.IRInstr, 0, len(instrs))
	for i := 0; i < len(instrs); i++ {
		if replacement, ok := replacements[i]; ok {
			out = append(out, replacement...)
			continue
		}
		instr := instrs[i]
		if i+1 < len(instrs) && instr.Kind == ir.IRStoreLocal &&
			instrs[i+1].Kind == ir.IRLoadLocal &&
			instr.Local == instrs[i+1].Local {
			local := instr.Local
			if reason := mem2regPromotionBlockReason(local, paramSlots, stores, loads); reason != "" {
				s.decisions = append(
					s.decisions,
					PassDecision{Action: "not_promoted", Caller: fnName, Site: i, Reason: reason},
				)
			} else {
				s.decisions = append(
					s.decisions,
					PassDecision{
						Action: "promoted_single_assignment_temp",
						Caller: fnName,
						Site:   i,
						Reason: "single_store_single_load_adjacent",
					},
				)
				i++
				continue
			}
		}
		if instr.Kind == ir.IRStoreLocal && i > 0 && !isAdjacentMem2RegStoreLoad(instrs, i) {
			local := instr.Local
			loadIndex, hasLoad := loadIndexes[local]
			if hasLoad && loadIndex > i+1 {
				if reason := mem2regPromotionBlockReason(local, paramSlots, stores, loads); reason != "" {
					s.decisions = append(
						s.decisions,
						PassDecision{
							Action: "not_promoted",
							Caller: fnName,
							Site:   i,
							Reason: reason,
						},
					)
				} else if producer, ok := mem2regProducerBeforeStore(instrs, i); !ok {
					s.decisions = append(
						s.decisions,
						PassDecision{
							Action: "not_promoted",
							Caller: fnName,
							Site:   i,
							Reason: "producer_not_available",
						},
					)
				} else {
					if len(
						out,
					) < len(
						producer.Instrs,
					) || !sameMem2RegProducerSpan(
						out[len(out)-len(producer.Instrs):],
						producer.Instrs,
					) {
						s.decisions = append(
							s.decisions,
							PassDecision{
								Action: "not_promoted",
								Caller: fnName,
								Site:   i,
								Reason: "producer_not_available",
							},
						)
					} else if reason := separatedMem2RegBlockReason(
						instrs[i+1:loadIndex],
						producer.SourceLocals,
						local,
					); reason != "" {
						s.decisions = append(
							s.decisions,
							PassDecision{Action: "not_promoted", Caller: fnName, Site: i, Reason: reason},
						)
					} else {
						out = out[:len(out)-len(producer.Instrs)]
						replacements[loadIndex] = append([]ir.IRInstr(nil), producer.Instrs...)
						reason := "single_store_single_load_stack_neutral"
						switch producer.Kind {
						case mem2regProducerComparisonExpression:
							reason = "single_store_single_load_stack_neutral_comparison_expression"
						case mem2regProducerSafeConstUnaryNegExpression:
							reason = "single_store_single_load_stack_neutral_safe_const_unary_neg_expression"
						case mem2regProducerSafeKnownLocalUnaryNegExpression:
							reason = "single_store_single_load_stack_neutral_safe_known_local_unary_neg_expression"
						case mem2regProducerSafeConstArithmeticExpression:
							reason = "single_store_single_load_stack_neutral_safe_const_arithmetic_expression"
						case mem2regProducerSafeKnownLocalArithmeticExpression:
							reason = "single_store_single_load_stack_neutral_safe_known_local_arithmetic_expression"
						case mem2regProducerSafeConstDenominatorDivModExpression:
							reason = "single_store_single_load_stack_neutral_safe_const_denominator_divmod_expression"
						case mem2regProducerSafeKnownLocalDivModExpression:
							reason = "single_store_single_load_stack_neutral_safe_known_local_divmod_expression"
						}
						s.decisions = append(
							s.decisions,
							PassDecision{
								Action: "promoted_single_assignment_temp",
								Caller: fnName,
								Site:   i,
								Reason: reason,
							},
						)
						continue
					}
				}
			}
		}
		out = append(out, instr)
	}
	return out
}

func localUsageCounts(instrs []ir.IRInstr) (map[int]int, map[int]int) {
	stores := map[int]int{}
	loads := map[int]int{}
	for _, instr := range instrs {
		switch instr.Kind {
		case ir.IRStoreLocal:
			stores[instr.Local]++
		case ir.IRLoadLocal:
			loads[instr.Local]++
		}
	}
	return stores, loads
}

func localLoadIndexes(instrs []ir.IRInstr) map[int]int {
	indexes := map[int]int{}
	for i, instr := range instrs {
		if instr.Kind == ir.IRLoadLocal {
			indexes[instr.Local] = i
		}
	}
	return indexes
}

func mem2regPromotionBlockReason(
	local int,
	paramSlots int,
	stores map[int]int,
	loads map[int]int,
) string {
	if local < paramSlots {
		return "param_slot"
	}
	if stores[local] != 1 {
		return "local_not_single_store"
	}
	if loads[local] != 1 {
		return "local_not_single_load"
	}
	return ""
}

func isAdjacentMem2RegStoreLoad(instrs []ir.IRInstr, index int) bool {
	return index+1 < len(instrs) &&
		instrs[index].Kind == ir.IRStoreLocal &&
		instrs[index+1].Kind == ir.IRLoadLocal &&
		instrs[index].Local == instrs[index+1].Local
}

func sameMem2RegProducerSpan(left, right []ir.IRInstr) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if !sameMem2RegProducerInstr(left[i], right[i]) {
			return false
		}
	}
	return true
}

func sameMem2RegProducerInstr(left, right ir.IRInstr) bool {
	if left.Kind != right.Kind {
		return false
	}
	switch left.Kind {
	case ir.IRConstI32:
		return left.Imm == right.Imm
	case ir.IRLoadLocal:
		return left.Local == right.Local
	case ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
		ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32,
		ir.IRNegI32, ir.IRAddI32, ir.IRSubI32, ir.IRMulI32,
		ir.IRDivI32, ir.IRModI32:
		return true
	default:
		return false
	}
}

func mem2regProducerBeforeStore(instrs []ir.IRInstr, storeIndex int) (mem2regProducer, bool) {
	if storeIndex <= 0 {
		return mem2regProducer{}, false
	}
	if isSingleMem2RegProducerValue(instrs[storeIndex-1]) {
		span := instrs[storeIndex-1 : storeIndex]
		return mem2regProducer{
			Kind:         mem2regProducerSingleValue,
			Instrs:       append([]ir.IRInstr(nil), span...),
			SourceLocals: mem2regProducerSourceLocals(span),
		}, true
	}
	if storeIndex >= 2 &&
		isMem2RegSafeConstUnaryNegProducer(instrs[storeIndex-1], instrs[storeIndex-2]) {
		span := instrs[storeIndex-2 : storeIndex]
		return mem2regProducer{
			Kind:         mem2regProducerSafeConstUnaryNegExpression,
			Instrs:       append([]ir.IRInstr(nil), span...),
			SourceLocals: mem2regProducerSourceLocals(span),
		}, true
	}
	if storeIndex >= 2 &&
		isMem2RegSafeKnownLocalUnaryNegProducer(
			instrs[storeIndex-1],
			instrs[storeIndex-2],
			instrs,
			storeIndex-2,
		) {
		span := instrs[storeIndex-2 : storeIndex]
		return mem2regProducer{
			Kind:         mem2regProducerSafeKnownLocalUnaryNegExpression,
			Instrs:       append([]ir.IRInstr(nil), span...),
			SourceLocals: mem2regProducerSourceLocals(span),
		}, true
	}
	if storeIndex >= 3 &&
		isMem2RegComparisonProducer(instrs[storeIndex-1]) &&
		isSingleMem2RegProducerValue(instrs[storeIndex-2]) &&
		isSingleMem2RegProducerValue(instrs[storeIndex-3]) {
		span := instrs[storeIndex-3 : storeIndex]
		return mem2regProducer{
			Kind:         mem2regProducerComparisonExpression,
			Instrs:       append([]ir.IRInstr(nil), span...),
			SourceLocals: mem2regProducerSourceLocals(span),
		}, true
	}
	if storeIndex >= 3 &&
		isMem2RegSafeConstArithmeticProducer(
			instrs[storeIndex-1],
			instrs[storeIndex-3],
			instrs[storeIndex-2],
		) {
		span := instrs[storeIndex-3 : storeIndex]
		return mem2regProducer{
			Kind:         mem2regProducerSafeConstArithmeticExpression,
			Instrs:       append([]ir.IRInstr(nil), span...),
			SourceLocals: mem2regProducerSourceLocals(span),
		}, true
	}
	if storeIndex >= 3 &&
		isMem2RegSafeKnownLocalArithmeticProducer(
			instrs[storeIndex-1],
			instrs[storeIndex-3],
			instrs[storeIndex-2],
			instrs,
			storeIndex-3,
		) {
		span := instrs[storeIndex-3 : storeIndex]
		return mem2regProducer{
			Kind:         mem2regProducerSafeKnownLocalArithmeticExpression,
			Instrs:       append([]ir.IRInstr(nil), span...),
			SourceLocals: mem2regProducerSourceLocals(span),
		}, true
	}
	if storeIndex >= 3 &&
		isMem2RegSafeConstDenominatorDivModProducer(
			instrs[storeIndex-1],
			instrs[storeIndex-3],
			instrs[storeIndex-2],
		) {
		span := instrs[storeIndex-3 : storeIndex]
		return mem2regProducer{
			Kind:         mem2regProducerSafeConstDenominatorDivModExpression,
			Instrs:       append([]ir.IRInstr(nil), span...),
			SourceLocals: mem2regProducerSourceLocals(span),
		}, true
	}
	if storeIndex >= 3 &&
		isMem2RegSafeKnownLocalDivModProducer(
			instrs[storeIndex-1],
			instrs[storeIndex-3],
			instrs[storeIndex-2],
			instrs,
			storeIndex-3,
		) {
		span := instrs[storeIndex-3 : storeIndex]
		return mem2regProducer{
			Kind:         mem2regProducerSafeKnownLocalDivModExpression,
			Instrs:       append([]ir.IRInstr(nil), span...),
			SourceLocals: mem2regProducerSourceLocals(span),
		}, true
	}
	return mem2regProducer{}, false
}

func isSingleMem2RegProducerValue(instr ir.IRInstr) bool {
	return instr.Kind == ir.IRConstI32 || instr.Kind == ir.IRLoadLocal
}

func isMem2RegComparisonProducer(instr ir.IRInstr) bool {
	switch instr.Kind {
	case ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
		ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		return true
	default:
		return false
	}
}

func isMem2RegSafeConstUnaryNegProducer(op ir.IRInstr, operand ir.IRInstr) bool {
	if op.Kind != ir.IRNegI32 || operand.Kind != ir.IRConstI32 {
		return false
	}
	_, ok := checkedNegI32(operand.Imm)
	return ok
}

func isMem2RegSafeKnownLocalUnaryNegProducer(
	op ir.IRInstr,
	operand ir.IRInstr,
	instrs []ir.IRInstr,
	beforeIndex int,
) bool {
	if op.Kind != ir.IRNegI32 || operand.Kind != ir.IRLoadLocal {
		return false
	}
	remove := make([]bool, len(instrs))
	imm, ok := knownConstLocalBefore(instrs, remove, beforeIndex, operand.Local)
	if !ok {
		return false
	}
	_, ok = checkedNegI32(imm)
	return ok
}

func isMem2RegSafeConstArithmeticProducer(op ir.IRInstr, left ir.IRInstr, right ir.IRInstr) bool {
	switch op.Kind {
	case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32:
		if left.Kind != ir.IRConstI32 || right.Kind != ir.IRConstI32 {
			return false
		}
		_, ok := foldConstBinaryI32(op.Kind, left.Imm, right.Imm)
		return ok
	default:
		return false
	}
}

func isMem2RegSafeKnownLocalArithmeticProducer(
	op ir.IRInstr,
	left ir.IRInstr,
	right ir.IRInstr,
	instrs []ir.IRInstr,
	beforeIndex int,
) bool {
	switch op.Kind {
	case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32:
	default:
		return false
	}
	if left.Kind != ir.IRLoadLocal && right.Kind != ir.IRLoadLocal {
		return false
	}
	remove := make([]bool, len(instrs))
	leftImm, ok := knownConstOperandBefore(left, instrs, remove, beforeIndex)
	if !ok {
		return false
	}
	rightImm, ok := knownConstOperandBefore(right, instrs, remove, beforeIndex)
	if !ok {
		return false
	}
	_, ok = foldConstBinaryI32(op.Kind, leftImm, rightImm)
	return ok
}

func isMem2RegSafeConstDenominatorDivModProducer(
	op ir.IRInstr,
	left ir.IRInstr,
	right ir.IRInstr,
) bool {
	switch op.Kind {
	case ir.IRDivI32, ir.IRModI32:
		return left.Kind == ir.IRLoadLocal && right.Kind == ir.IRConstI32 && right.Imm != 0 &&
			right.Imm != -1
	default:
		return false
	}
}

func isMem2RegSafeKnownLocalDivModProducer(
	op ir.IRInstr,
	left ir.IRInstr,
	right ir.IRInstr,
	instrs []ir.IRInstr,
	beforeIndex int,
) bool {
	switch op.Kind {
	case ir.IRDivI32, ir.IRModI32:
	default:
		return false
	}
	if left.Kind != ir.IRLoadLocal && right.Kind != ir.IRLoadLocal {
		return false
	}
	remove := make([]bool, len(instrs))
	leftImm, ok := knownConstOperandBefore(left, instrs, remove, beforeIndex)
	if !ok {
		return false
	}
	rightImm, ok := knownConstOperandBefore(right, instrs, remove, beforeIndex)
	if !ok {
		return false
	}
	_, ok = foldConstBinaryI32(op.Kind, leftImm, rightImm)
	return ok
}

func mem2regProducerSourceLocals(instrs []ir.IRInstr) map[int]struct{} {
	locals := map[int]struct{}{}
	for _, instr := range instrs {
		if instr.Kind == ir.IRLoadLocal {
			locals[instr.Local] = struct{}{}
		}
	}
	return locals
}

func separatedMem2RegBlockReason(
	instrs []ir.IRInstr,
	sourceLocals map[int]struct{},
	tempLocal int,
) string {
	depth := 0
	for _, instr := range instrs {
		if instr.Kind == ir.IRStoreLocal {
			if instr.Local == tempLocal {
				return "temp_local_modified_before_load"
			}
			if _, ok := sourceLocals[instr.Local]; ok {
				return "source_local_modified_before_load"
			}
		}
		pop, push, ok := mem2regStackEffect(instr)
		if !ok || depth < pop {
			return "intervening_not_stack_neutral"
		}
		depth += push - pop
	}
	if depth != 0 {
		return "intervening_not_stack_neutral"
	}
	return ""
}

func mem2regStackEffect(instr ir.IRInstr) (pop int, push int, ok bool) {
	switch instr.Kind {
	case ir.IRConstI32, ir.IRLoadLocal:
		return 0, 1, true
	case ir.IRStoreLocal:
		return 1, 0, true
	case ir.IRNegI32:
		return 1, 1, true
	case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32,
		ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
		ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		return 2, 1, true
	default:
		return 0, 0, false
	}
}

func hasMem2RegControlFlow(instrs []ir.IRInstr) bool {
	for _, instr := range instrs {
		switch instr.Kind {
		case ir.IRLabel, ir.IRJmp, ir.IRJmpIfZero:
			return true
		}
	}
	return false
}

func hasAdjacentStoreLoad(instrs []ir.IRInstr) bool {
	for i := 0; i+1 < len(instrs); i++ {
		if instrs[i].Kind == ir.IRStoreLocal && instrs[i+1].Kind == ir.IRLoadLocal &&
			instrs[i].Local == instrs[i+1].Local {
			return true
		}
	}
	return false
}

// ---- pgo_lto.go ----

const ProfileCollectionSchemaVersion = "tetra.optimizer.profile.v1"

type ProfileCollection struct {
	SchemaVersion string            `json:"schema_version"`
	ProgramHash   string            `json:"program_hash"`
	TargetTriple  string            `json:"target_triple"`
	Functions     []ProfileFunction `json:"functions"`
}

type ProfileFunction struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	EntryCount uint64           `json:"entry_count"`
	Counters   []ProfileCounter `json:"counters,omitempty"`
}

type ProfileCounter struct {
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	Count uint64 `json:"count"`
}

type OptimizerProfileInputEvidence struct {
	SchemaVersion   string   `json:"schema_version"`
	ProgramHash     string   `json:"program_hash"`
	TargetTriple    string   `json:"target_triple"`
	Functions       int      `json:"functions"`
	TotalEntryCount uint64   `json:"total_entry_count"`
	CounterKinds    []string `json:"counter_kinds,omitempty"`
	Digest          string   `json:"digest"`
}

type PGOLTOTargetCPUID string

const (
	PGOLTOTargetCPUProfileCollectionFormat     PGOLTOTargetCPUID = "profile_collection_format"
	PGOLTOTargetCPUPGOOptimizerInput           PGOLTOTargetCPUID = "pgo_optimizer_input"
	PGOLTOTargetCPUTargetCPUFeatureDetection   PGOLTOTargetCPUID = "target_cpu_feature_detection"
	PGOLTOTargetCPULTOIncrementalModuleSummary PGOLTOTargetCPUID = "lto_incremental_module_summary"
	PGOLTOTargetCPUSafeSemanticsFlags          PGOLTOTargetCPUID = "safe_semantics_flags"
)

type PGOLTOTargetCPUStatus string

const (
	PGOLTOTargetCPUImplementedNarrow PGOLTOTargetCPUStatus = "implemented_narrow"
	PGOLTOTargetCPUNotYetCovered     PGOLTOTargetCPUStatus = "not_yet_covered"
)

type PGOLTOTargetCPUCoverageReport struct {
	SchemaVersion string                       `json:"schema_version"`
	Rows          []PGOLTOTargetCPUCoverageRow `json:"rows"`
	NonClaims     []string                     `json:"non_claims"`
}

type PGOLTOTargetCPUSafeSemanticsClosureEvidence struct {
	SchemaVersion           string                `json:"schema_version"`
	Status                  PGOLTOTargetCPUStatus `json:"status"`
	CompletedRows           []PGOLTOTargetCPUID   `json:"completed_rows"`
	RejectedUnsafeClaims    []string              `json:"rejected_unsafe_claims"`
	PublicSemanticFlagCount int                   `json:"public_semantic_flag_count"`
	ChangesSafeSemantics    bool                  `json:"changes_safe_semantics"`
	Evidence                []string              `json:"evidence"`
	Boundary                string                `json:"boundary"`
}

type PGOLTOTargetCPUCoverageRow struct {
	ID                   PGOLTOTargetCPUID     `json:"id"`
	Name                 string                `json:"name"`
	Status               PGOLTOTargetCPUStatus `json:"status"`
	OptimizerInput       bool                  `json:"optimizer_input"`
	ChangesSafeSemantics bool                  `json:"changes_safe_semantics"`
	RequiredFacts        []string              `json:"required_facts,omitempty"`
	MissingFacts         []string              `json:"missing_facts,omitempty"`
	Reason               string                `json:"reason"`
	Evidence             string                `json:"evidence"`
	Boundary             string                `json:"boundary"`
}

func PGOLTOTargetCPUSafeSemanticsClosure() (PGOLTOTargetCPUSafeSemanticsClosureEvidence, error) {
	report, err := PGOLTOTargetCPUCoverage()
	if err != nil {
		return PGOLTOTargetCPUSafeSemanticsClosureEvidence{}, err
	}
	if err := ValidatePGOLTOTargetCPUSafeSemanticsClosure(report); err != nil {
		return PGOLTOTargetCPUSafeSemanticsClosureEvidence{}, err
	}
	rejected, err := p17SafeSemanticsRejectedUnsafeClaims()
	if err != nil {
		return PGOLTOTargetCPUSafeSemanticsClosureEvidence{}, err
	}
	completed := make([]PGOLTOTargetCPUID, 0, len(report.Rows))
	for _, row := range report.Rows {
		completed = append(completed, row.ID)
	}
	return PGOLTOTargetCPUSafeSemanticsClosureEvidence{
		SchemaVersion:           "tetra.optimizer.pgo_lto_target_cpu.safe_semantics_closure.v1",
		Status:                  PGOLTOTargetCPUImplementedNarrow,
		CompletedRows:           completed,
		RejectedUnsafeClaims:    rejected,
		PublicSemanticFlagCount: 0,
		ChangesSafeSemantics:    false,
		Evidence: []string{
			"compiler/internal/opt/opt_core.go::ValidatePGOLTOTargetCPUSafeSemanticsClosure",
			"compiler/compiler_suite_test.go::TestBuildOptionsExposeNoBackendSemanticMode",
			("compiler/internal/opt/opt_suite_test.go::" +
				"TestManagerRejectsProfileGuidedRewritePolicyUntilValidationE" +
				"xists"),
			("compiler/internal/backend/x64/target_features_test.go::" +
				"TestCodegenOptionsTargetFeatureGuardIsEvidenceOnly"),
			("compiler/internal/cache/lto_summary_test.go::" +
				"TestIncrementalModuleSummaryV1RecordsDependencyHashContractA" +
				"ndRejectsConsumers"),
		},
		Boundary: ("final P17.4 closure is evidence-only: no " +
			"PGO/profile/LTO/target-cpu public flag changes safe-program " +
			"semantics, profile-guided rewrite policy is rejected, " +
			"target-specific optimization evidence is rejected, and " +
			"LTO/incremental summaries remain non-consumer evidence only"),
	}, nil
}

func ValidatePGOLTOTargetCPUSafeSemanticsClosure(report PGOLTOTargetCPUCoverageReport) error {
	if report.SchemaVersion != "tetra.optimizer.pgo_lto_target_cpu.v1" {
		return fmt.Errorf("P17.4 safe-semantics closure: schema = %q", report.SchemaVersion)
	}
	if !hasReportRow(
		report.NonClaims,
		"no PGO, LTO, target-cpu, or profile flag changes safe-program semantics",
	) {
		return fmt.Errorf("P17.4 safe-semantics closure: missing safe-semantics non-claim")
	}
	expected := map[PGOLTOTargetCPUID]bool{
		PGOLTOTargetCPUProfileCollectionFormat:     true,
		PGOLTOTargetCPUPGOOptimizerInput:           true,
		PGOLTOTargetCPUTargetCPUFeatureDetection:   true,
		PGOLTOTargetCPULTOIncrementalModuleSummary: true,
		PGOLTOTargetCPUSafeSemanticsFlags:          true,
	}
	if len(report.Rows) != len(expected) {
		return fmt.Errorf(
			"P17.4 safe-semantics closure: row count = %d, want %d",
			len(report.Rows),
			len(expected),
		)
	}
	byID := map[PGOLTOTargetCPUID]PGOLTOTargetCPUCoverageRow{}
	for _, row := range report.Rows {
		if row.ID == "" {
			return fmt.Errorf("P17.4 safe-semantics closure: row missing id")
		}
		if !expected[row.ID] {
			return fmt.Errorf("P17.4 safe-semantics closure: unexpected row %q", row.ID)
		}
		if _, exists := byID[row.ID]; exists {
			return fmt.Errorf("P17.4 safe-semantics closure: duplicate row %q", row.ID)
		}
		if row.Name == "" || row.Reason == "" || row.Evidence == "" || row.Boundary == "" {
			return fmt.Errorf(
				"P17.4 safe-semantics closure: row %q missing machine-checkable evidence",
				row.ID,
			)
		}
		if row.Status != PGOLTOTargetCPUImplementedNarrow {
			return fmt.Errorf(
				"P17.4 safe-semantics closure: row %q not complete: %s",
				row.ID,
				row.Status,
			)
		}
		if len(row.MissingFacts) != 0 {
			return fmt.Errorf(
				"P17.4 safe-semantics closure: row %q has missing facts: %v",
				row.ID,
				row.MissingFacts,
			)
		}
		if row.ChangesSafeSemantics {
			return fmt.Errorf("P17.4 safe-semantics closure: row %q changes safe semantics", row.ID)
		}
		byID[row.ID] = row
	}
	for id := range expected {
		if _, ok := byID[id]; !ok {
			return fmt.Errorf("P17.4 safe-semantics closure: missing row %q", id)
		}
	}
	if err := validateProfileCollectionClosureRow(
		byID[PGOLTOTargetCPUProfileCollectionFormat],
	); err != nil {
		return err
	}
	if err := validatePGOInputClosureRow(byID[PGOLTOTargetCPUPGOOptimizerInput]); err != nil {
		return err
	}
	if err := validateTargetCPUClosureRow(byID[PGOLTOTargetCPUTargetCPUFeatureDetection]); err != nil {
		return err
	}
	if err := validateLTOClosureRow(byID[PGOLTOTargetCPULTOIncrementalModuleSummary]); err != nil {
		return err
	}
	if err := validateSafeSemanticsClosureRow(byID[PGOLTOTargetCPUSafeSemanticsFlags]); err != nil {
		return err
	}
	return nil
}

func MarshalProfileCollection(profile ProfileCollection) ([]byte, error) {
	if err := ValidateProfileCollection(profile); err != nil {
		return nil, err
	}
	canonical := canonicalProfileCollection(profile)
	out, err := json.Marshal(canonical)
	if err != nil {
		return nil, fmt.Errorf("profile collection: marshal: %w", err)
	}
	return out, nil
}

func ParseProfileCollection(raw []byte) (ProfileCollection, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var profile ProfileCollection
	if err := dec.Decode(&profile); err != nil {
		return ProfileCollection{}, fmt.Errorf("profile collection: decode: %w", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return ProfileCollection{}, fmt.Errorf("profile collection: trailing JSON value")
		}
		return ProfileCollection{}, fmt.Errorf("profile collection: trailing JSON: %w", err)
	}
	if err := ValidateProfileCollection(profile); err != nil {
		return ProfileCollection{}, err
	}
	return canonicalProfileCollection(profile), nil
}

func ValidateProfileCollection(profile ProfileCollection) error {
	if profile.SchemaVersion != ProfileCollectionSchemaVersion {
		return fmt.Errorf(
			"profile collection: schema_version = %q, want %q",
			profile.SchemaVersion,
			ProfileCollectionSchemaVersion,
		)
	}
	if strings.TrimSpace(profile.ProgramHash) == "" {
		return fmt.Errorf("profile collection: missing program_hash")
	}
	if !strings.HasPrefix(profile.ProgramHash, "sha256:") {
		return fmt.Errorf("profile collection: program_hash must use sha256: prefix")
	}
	if strings.TrimSpace(profile.TargetTriple) == "" {
		return fmt.Errorf("profile collection: missing target_triple")
	}
	if len(profile.Functions) == 0 {
		return fmt.Errorf("profile collection: at least one function row is required")
	}
	seenIDs := map[string]bool{}
	seenNames := map[string]bool{}
	for i, fn := range profile.Functions {
		if strings.TrimSpace(fn.ID) == "" {
			return fmt.Errorf("profile collection: function %d missing id", i)
		}
		if strings.TrimSpace(fn.Name) == "" {
			return fmt.Errorf("profile collection: function %q missing name", fn.ID)
		}
		if seenIDs[fn.ID] {
			return fmt.Errorf("profile collection: duplicate function id %q", fn.ID)
		}
		seenIDs[fn.ID] = true
		if seenNames[fn.Name] {
			return fmt.Errorf("profile collection: duplicate function name %q", fn.Name)
		}
		seenNames[fn.Name] = true
		seenCounters := map[string]bool{}
		for j, counter := range fn.Counters {
			if strings.TrimSpace(counter.Kind) == "" {
				return fmt.Errorf(
					"profile collection: function %q counter %d missing kind",
					fn.ID,
					j,
				)
			}
			if strings.TrimSpace(counter.Name) == "" {
				return fmt.Errorf(
					"profile collection: function %q counter %d missing name",
					fn.ID,
					j,
				)
			}
			key := counter.Kind + "\x00" + counter.Name
			if seenCounters[key] {
				return fmt.Errorf(
					"profile collection: function %q duplicate counter %q/%q",
					fn.ID,
					counter.Kind,
					counter.Name,
				)
			}
			seenCounters[key] = true
		}
	}
	return nil
}

func BuildOptimizerProfileInputEvidence(
	profile ProfileCollection,
) (OptimizerProfileInputEvidence, error) {
	encoded, err := MarshalProfileCollection(profile)
	if err != nil {
		return OptimizerProfileInputEvidence{}, err
	}
	canonical := canonicalProfileCollection(profile)
	sum := sha256.Sum256(encoded)
	kindSet := map[string]bool{}
	var totalEntryCount uint64
	for _, fn := range canonical.Functions {
		totalEntryCount += fn.EntryCount
		for _, counter := range fn.Counters {
			kindSet[counter.Kind] = true
		}
	}
	counterKinds := make([]string, 0, len(kindSet))
	for kind := range kindSet {
		counterKinds = append(counterKinds, kind)
	}
	sort.Strings(counterKinds)
	return OptimizerProfileInputEvidence{
		SchemaVersion:   canonical.SchemaVersion,
		ProgramHash:     canonical.ProgramHash,
		TargetTriple:    canonical.TargetTriple,
		Functions:       len(canonical.Functions),
		TotalEntryCount: totalEntryCount,
		CounterKinds:    counterKinds,
		Digest:          fmt.Sprintf("sha256:%x", sum),
	}, nil
}

func PGOLTOTargetCPUCoverage() (PGOLTOTargetCPUCoverageReport, error) {
	profileRow, err := pgoProfileCollectionFormatRow()
	if err != nil {
		return PGOLTOTargetCPUCoverageReport{}, err
	}
	pgoInputRow, err := pgoOptimizerInputRow()
	if err != nil {
		return PGOLTOTargetCPUCoverageReport{}, err
	}
	return PGOLTOTargetCPUCoverageReport{
		SchemaVersion: "tetra.optimizer.pgo_lto_target_cpu.v1",
		Rows: []PGOLTOTargetCPUCoverageRow{
			profileRow,
			pgoInputRow,
			targetCPUFeatureDetectionRow(),
			ltoIncrementalModuleSummaryRow(),
			{
				ID:                   PGOLTOTargetCPUSafeSemanticsFlags,
				Name:                 "safe semantics for PGO/LTO/target-cpu flags",
				Status:               PGOLTOTargetCPUImplementedNarrow,
				OptimizerInput:       false,
				ChangesSafeSemantics: false,
				RequiredFacts: []string{
					"no_public_semantic_flag",
					"profile_format_validated",
					"profile_input_policy_unused",
					"profile_guided_rewrite_rejected",
					"target_feature_evidence_only",
					"lto_summary_non_consumer",
					"validators_reject_fake_claims",
					"safe_program_truth_preserved",
				},
				Reason: ("guarded:no public BuildOptions flag applies " +
					"PGO/LTO/target-cpu/profile data; profile input is internal " +
					"evidence with unused pass policy, profile-guided rewrite " +
					"rejection, and final safe-semantics closure validation"),
				Evidence: ("compiler/compiler_facade.go::BuildOptions; " +
					"compiler/compiler_suite_test.go::" +
					"TestBuildOptionsExposeNoBackendSemanticMode; " +
					"compiler/internal/opt/opt_core.go::PGOLTOTargetCPUCoverage; " +
					"compiler/internal/opt/opt_core.go::" +
					"ValidatePGOLTOTargetCPUSafeSemanticsClosure; " +
					"compiler/internal/opt/opt_suite_test.go::" +
					"TestPGOLTOTargetCPUSafeSemanticsClosureRejectsFakeClaims"),
				Boundary: ("no public BuildOptions flag, no optimizer pass consumes " +
					"profile counts, all registered optimizer passes declare " +
					"profile_input_policy unused, profile-guided rewrite policy " +
					"is rejected, target-cpu feature data is internal evidence " +
					"only with no target-specific rewrite, and LTO/incremental " +
					"summaries are non-consumer evidence only; no LTO setting " +
					"reaches codegen or linker from this slice; profile parsing " +
					"is evidence-only, profile input is report/metadata evidence " +
					"only, the final coverage validator rejects fake " +
					"semantic-changing coverage rows, incomplete rows, fake " +
					"profile-format optimizer input, fake target-cpu/LTO " +
					"optimizer input, and missing safe-program truth facts, and " +
					"safe-program semantics unchanged"),
			},
		},
		NonClaims: []string{
			"no PGO, LTO, target-cpu, or profile flag changes safe-program semantics",
			"no profile-guided optimizer rewrite claim",
			"no target-specific rewrite or CPU-tuned codegen claim",
			"no LTO optimizer, linker consumer, codegen consumer, or incremental speedup claim",
			"no LTO, incremental compilation speedup, or C/Rust performance parity claim",
		},
	}, nil
}

func validateProfileCollectionClosureRow(row PGOLTOTargetCPUCoverageRow) error {
	if row.OptimizerInput {
		return fmt.Errorf(
			("P17.4 safe-semantics closure: profile collection format " +
				"must remain inert evidence, not optimizer input"),
		)
	}
	for _, fact := range []string{
		"schema_validation",
		"canonical_json",
		"duplicate_rejection",
		"negative_counter_rejection",
	} {
		if !hasReportRow(row.RequiredFacts, fact) {
			return fmt.Errorf(
				"P17.4 safe-semantics closure: profile collection format missing required fact %q",
				fact,
			)
		}
	}
	if !strings.Contains(row.Boundary, "inert evidence") ||
		!strings.Contains(row.Boundary, "does not feed optimizer decisions") {
		return fmt.Errorf(
			("P17.4 safe-semantics closure: profile collection format " +
				"boundary no longer proves inert evidence"),
		)
	}
	return nil
}

func validatePGOInputClosureRow(row PGOLTOTargetCPUCoverageRow) error {
	if !row.OptimizerInput {
		return fmt.Errorf(
			"P17.4 safe-semantics closure: PGO optimizer input row must record optimizer input evidence",
		)
	}
	for _, fact := range []string{
		"optimizer_profile_input_api",
		"pass_contract_profile_metadata",
		"translation_validation_for_profile_guided_decisions",
		"negative_safe_semantics_tests",
	} {
		if !hasReportRow(row.RequiredFacts, fact) {
			return fmt.Errorf(
				"P17.4 safe-semantics closure: PGO optimizer input missing required fact %q",
				fact,
			)
		}
	}
	if !strings.Contains(row.Boundary, "all registered passes keep profile_input_policy unused") ||
		!strings.Contains(row.Boundary, "no profile-guided rewrite is selected") {
		return fmt.Errorf(
			("P17.4 safe-semantics closure: PGO optimizer input boundary " +
				"no longer rejects profile-guided rewrite"),
		)
	}
	return nil
}

func validateTargetCPUClosureRow(row PGOLTOTargetCPUCoverageRow) error {
	if row.OptimizerInput {
		return fmt.Errorf(
			"P17.4 safe-semantics closure: target-cpu feature detection must not be optimizer input",
		)
	}
	for _, fact := range []string{
		"target_feature_model",
		"portable_baseline_fallback",
		"guarded_codegen_contract",
		"negative_safe_semantics_tests",
	} {
		if !hasReportRow(row.RequiredFacts, fact) {
			return fmt.Errorf(
				"P17.4 safe-semantics closure: target-cpu feature detection missing required fact %q",
				fact,
			)
		}
	}
	if !strings.Contains(row.Boundary, "no public target-cpu BuildOptions field") ||
		!strings.Contains(row.Boundary, "no target-specific rewrite") ||
		!strings.Contains(row.Boundary, "safe-program semantics unchanged") {
		return fmt.Errorf(
			"P17.4 safe-semantics closure: target-cpu boundary no longer proves evidence-only semantics",
		)
	}
	return nil
}

func validateLTOClosureRow(row PGOLTOTargetCPUCoverageRow) error {
	if row.OptimizerInput {
		return fmt.Errorf(
			"P17.4 safe-semantics closure: LTO/incremental module summary must not be optimizer input",
		)
	}
	for _, fact := range []string{
		"module_summary_schema",
		"dependency_hash_contract",
		"cross_module_validation_row",
		"incremental_cache_negative_tests",
		"non_consumer_boundary",
	} {
		if !hasReportRow(row.RequiredFacts, fact) {
			return fmt.Errorf(
				"P17.4 safe-semantics closure: LTO/incremental module summary missing required fact %q",
				fact,
			)
		}
	}
	for _, want := range []string{
		"no LTO optimizer",
		"cross-module inlining",
		"linker consumer",
		"codegen consumer",
		"safe-program semantics change",
	} {
		if !strings.Contains(row.Boundary, want) {
			return fmt.Errorf(
				"P17.4 safe-semantics closure: LTO/incremental module summary boundary missing %q",
				want,
			)
		}
	}
	return nil
}

func validateSafeSemanticsClosureRow(row PGOLTOTargetCPUCoverageRow) error {
	if row.OptimizerInput {
		return fmt.Errorf(
			"P17.4 safe-semantics closure: safe semantics row must not be optimizer input",
		)
	}
	for _, fact := range []string{
		"no_public_semantic_flag",
		"profile_format_validated",
		"profile_input_policy_unused",
		"profile_guided_rewrite_rejected",
		"target_feature_evidence_only",
		"lto_summary_non_consumer",
		"validators_reject_fake_claims",
		"safe_program_truth_preserved",
	} {
		if !hasReportRow(row.RequiredFacts, fact) {
			return fmt.Errorf(
				"P17.4 safe-semantics closure: safe semantics row missing required fact %q",
				fact,
			)
		}
	}
	for _, want := range []string{
		"no public BuildOptions flag",
		"no optimizer pass consumes profile counts",
		"profile-guided rewrite policy is rejected",
		"target-cpu feature data is internal evidence only",
		"LTO/incremental summaries are non-consumer evidence only",
		"safe-program semantics unchanged",
	} {
		if !strings.Contains(row.Boundary, want) {
			return fmt.Errorf(
				"P17.4 safe-semantics closure: safe semantics row boundary missing %q",
				want,
			)
		}
	}
	return nil
}

func p17SafeSemanticsRejectedUnsafeClaims() ([]string, error) {
	report, err := PGOLTOTargetCPUCoverage()
	if err != nil {
		return nil, err
	}
	mutated := clonePGOLTOCoverageReport(report)
	for i := range mutated.Rows {
		if mutated.Rows[i].ID == PGOLTOTargetCPUSafeSemanticsFlags {
			mutated.Rows[i].ChangesSafeSemantics = true
		}
	}
	if err := ValidatePGOLTOTargetCPUSafeSemanticsClosure(mutated); err == nil {
		return nil, fmt.Errorf(
			"P17.4 safe-semantics closure: fake semantic-changing coverage was accepted",
		)
	}
	rejected := []string{
		"coverage_validator_rejects_fake_claims",
		"public_build_options_semantic_flag_rejected",
	}

	guided := pgoInputEvidencePass()
	guided.ProfileInputPolicy = ProfileInputGuidedRewrite
	if err := ValidatePassContract(guided); err == nil {
		return nil, fmt.Errorf(
			"P17.4 safe-semantics closure: profile-guided rewrite policy was accepted",
		)
	}
	rejected = append(rejected, "profile_guided_rewrite_policy_rejected")

	if err := validateTargetFeatureClosureEvidence(x64.TargetFeatureEvidence{
		Source: string(x64.TargetFeatureSourceExplicit),
		Features: []string{string(
			x64.TargetFeatureSSE2,
		), string(
			x64.TargetFeatureAVX2,
		)},
		PortableBaselineFallback:          false,
		ChangesSafeSemantics:              false,
		EnablesTargetSpecificOptimization: true,
	}); err == nil {
		return nil, fmt.Errorf(
			"P17.4 safe-semantics closure: target-specific optimization evidence was accepted",
		)
	}
	rejected = append(rejected, "target_specific_optimization_evidence_rejected")

	summary, err := p17ClosureModuleSummaryFixture()
	if err != nil {
		return nil, err
	}
	codegenConsumer := summary
	codegenConsumer.CodegenConsumer = true
	if err := cache.ValidateIncrementalModuleSummary(codegenConsumer); err == nil {
		return nil, fmt.Errorf("P17.4 safe-semantics closure: LTO codegen consumer was accepted")
	}
	rejected = append(rejected, "lto_codegen_consumer_rejected")

	linkerConsumer := summary
	linkerConsumer.LinkerConsumer = true
	if err := cache.ValidateIncrementalModuleSummary(linkerConsumer); err == nil {
		return nil, fmt.Errorf("P17.4 safe-semantics closure: LTO linker consumer was accepted")
	}
	rejected = append(rejected, "lto_linker_consumer_rejected")

	sort.Strings(rejected)
	return rejected, nil
}

func validateTargetFeatureClosureEvidence(evidence x64.TargetFeatureEvidence) error {
	if strings.TrimSpace(evidence.Source) == "" {
		return fmt.Errorf("P17.4 target-feature evidence: missing source")
	}
	if evidence.ChangesSafeSemantics {
		return fmt.Errorf("P17.4 target-feature evidence: changes safe semantics")
	}
	if evidence.EnablesTargetSpecificOptimization {
		return fmt.Errorf("P17.4 target-feature evidence: enables target-specific optimization")
	}
	return nil
}

func p17ClosureModuleSummaryFixture() (cache.IncrementalModuleSummary, error) {
	depHash, err := cache.DepSigHashFromDepsWithInterfaceHashes(
		[]string{"math.core.add"},
		[]string{"math.core.Vec"},
		map[string]semantics.FuncSig{
			"math.core.add": {ParamTypes: []string{"i32", "i32"}, ReturnType: "i32", Public: true},
		},
		map[string]string{"math.core.Vec": "struct{x:i32,y:i32}"},
		map[string]string{"math.core": "sha256:p17closureapi"},
	)
	if err != nil {
		return cache.IncrementalModuleSummary{}, err
	}
	return cache.BuildIncrementalModuleSummary(cache.IncrementalModuleSummaryInput{
		Module:           "app.main",
		Target:           "linux-x64",
		BuildTag:         "p17-safe-semantics-closure",
		Source:           []byte("module app.main\n"),
		DependencyHash:   depHash,
		PublicAPIHash:    "sha256:p17closureapp",
		ExternalCallees:  []string{"math.core.add"},
		ExternalTypeDeps: []string{"math.core.Vec"},
	})
}

func clonePGOLTOCoverageReport(report PGOLTOTargetCPUCoverageReport) PGOLTOTargetCPUCoverageReport {
	out := PGOLTOTargetCPUCoverageReport{
		SchemaVersion: report.SchemaVersion,
		Rows:          append([]PGOLTOTargetCPUCoverageRow(nil), report.Rows...),
		NonClaims:     append([]string(nil), report.NonClaims...),
	}
	for i := range out.Rows {
		out.Rows[i].RequiredFacts = append([]string(nil), out.Rows[i].RequiredFacts...)
		out.Rows[i].MissingFacts = append([]string(nil), out.Rows[i].MissingFacts...)
	}
	return out
}

func canonicalProfileCollection(profile ProfileCollection) ProfileCollection {
	out := ProfileCollection{
		SchemaVersion: profile.SchemaVersion,
		ProgramHash:   profile.ProgramHash,
		TargetTriple:  profile.TargetTriple,
		Functions:     append([]ProfileFunction(nil), profile.Functions...),
	}
	sort.SliceStable(out.Functions, func(i, j int) bool {
		if out.Functions[i].ID == out.Functions[j].ID {
			return out.Functions[i].Name < out.Functions[j].Name
		}
		return out.Functions[i].ID < out.Functions[j].ID
	})
	for i := range out.Functions {
		out.Functions[i].Counters = append([]ProfileCounter(nil), out.Functions[i].Counters...)
		sort.SliceStable(out.Functions[i].Counters, func(a, b int) bool {
			if out.Functions[i].Counters[a].Kind == out.Functions[i].Counters[b].Kind {
				return out.Functions[i].Counters[a].Name < out.Functions[i].Counters[b].Name
			}
			return out.Functions[i].Counters[a].Kind < out.Functions[i].Counters[b].Kind
		})
	}
	return out
}

func pgoProfileCollectionFormatRow() (PGOLTOTargetCPUCoverageRow, error) {
	fixture := ProfileCollection{
		SchemaVersion: ProfileCollectionSchemaVersion,
		ProgramHash:   "sha256:p17profilefixture",
		TargetTriple:  "linux-x64",
		Functions: []ProfileFunction{
			{
				ID:         "fn:main",
				Name:       "main",
				EntryCount: 1,
				Counters: []ProfileCounter{
					{Kind: "edge", Name: "return", Count: 1},
				},
			},
		},
	}
	encoded, err := MarshalProfileCollection(fixture)
	if err != nil {
		return PGOLTOTargetCPUCoverageRow{}, err
	}
	parsed, err := ParseProfileCollection(encoded)
	if err != nil {
		return PGOLTOTargetCPUCoverageRow{}, err
	}
	reencoded, err := MarshalProfileCollection(parsed)
	if err != nil {
		return PGOLTOTargetCPUCoverageRow{}, err
	}
	if !bytes.Equal(encoded, reencoded) {
		return PGOLTOTargetCPUCoverageRow{}, fmt.Errorf(
			"profile collection: canonical round trip drifted: %s vs %s",
			string(encoded),
			string(reencoded),
		)
	}
	for name, raw := range map[string][]byte{
		"duplicate": []byte(("{\"schema_version\":\"tetra.optimizer.profile.v1\"," +
			"\"program_hash\":\"sha256:p17profilefixture\",\"target_triple\":" +
			"\"linux-x64\",\"functions\":[{\"id\":\"fn:main\",\"name\":\"main\"," +
			"\"entry_count\":1},{\"id\":\"fn:main\",\"name\":\"other\"," +
			"\"entry_count\":1}]}")),
		"negative counter": []byte(("{\"schema_version\":\"tetra.optimizer.profile.v1\"," +
			"\"program_hash\":\"sha256:p17profilefixture\",\"target_triple\":" +
			"\"linux-x64\",\"functions\":[{\"id\":\"fn:main\",\"name\":\"main\"," +
			"\"entry_count\":1,\"counters\":[{\"kind\":\"edge\",\"name\":\"return\"," +
			"\"count\":-1}]}]}")),
	} {
		if _, err := ParseProfileCollection(raw); err == nil {
			return PGOLTOTargetCPUCoverageRow{}, fmt.Errorf(
				"profile collection: %s fixture unexpectedly accepted",
				name,
			)
		}
	}
	return PGOLTOTargetCPUCoverageRow{
		ID:                   PGOLTOTargetCPUProfileCollectionFormat,
		Name:                 "profile collection format",
		Status:               PGOLTOTargetCPUImplementedNarrow,
		OptimizerInput:       false,
		ChangesSafeSemantics: false,
		RequiredFacts: []string{
			"schema_validation",
			"canonical_json",
			"duplicate_rejection",
			"negative_counter_rejection",
		},
		Reason: ("implemented_narrow:tetra.optimizer.profile.v1 canonical " +
			"JSON profile collection format with duplicate and negative " +
			"counter rejection; inert until a separate optimizer-input " +
			"slice consumes it"),
		Evidence: ("compiler/internal/opt/opt_core.go::ProfileCollection; " +
			"compiler/internal/opt/opt_suite_test.go::" +
			"TestProfileCollectionFormatV1RoundTripsAndRejectsUnsafeDrift"),
		Boundary: ("tetra.optimizer.profile.v1 is an inert evidence format only:" +
			" it records canonical JSON function entry counts and named " +
			"counters, rejects duplicate function/counter identity and " +
			"negative counter JSON, and does not feed optimizer " +
			"decisions, codegen, target-cpu selection, LTO, incremental " +
			"compilation, or safe-program semantics"),
	}, nil
}

func pgoOptimizerInputRow() (PGOLTOTargetCPUCoverageRow, error) {
	profile := ProfileCollection{
		SchemaVersion: ProfileCollectionSchemaVersion,
		ProgramHash:   "sha256:p17pgoinputfixture",
		TargetTriple:  "linux-x64",
		Functions: []ProfileFunction{{
			ID:         "fn:main",
			Name:       "main",
			EntryCount: 7,
			Counters: []ProfileCounter{
				{Kind: "edge", Name: "return", Count: 7},
			},
		}},
	}
	evidence, err := BuildOptimizerProfileInputEvidence(profile)
	if err != nil {
		return PGOLTOTargetCPUCoverageRow{}, err
	}
	prog := pgoInputEvidenceProgram()
	before := FormatProgram(prog)
	report, err := NewManager().RunWithOptions(
		prog,
		Options{ProfileInput: &profile},
		pgoInputEvidencePass(),
	)
	if err != nil {
		return PGOLTOTargetCPUCoverageRow{}, err
	}
	if FormatProgram(prog) != before {
		return PGOLTOTargetCPUCoverageRow{}, fmt.Errorf(
			"pgo optimizer input: profile input changed IR",
		)
	}
	if len(report.Passes) != 1 {
		return PGOLTOTargetCPUCoverageRow{}, fmt.Errorf(
			"pgo optimizer input: expected one pass report, got %d",
			len(report.Passes),
		)
	}
	row := report.Passes[0]
	if row.ProfileInputPolicy != ProfileInputUnused || row.ProfileInput == nil ||
		row.ProfileInput.Digest != evidence.Digest {
		return PGOLTOTargetCPUCoverageRow{}, fmt.Errorf(
			"pgo optimizer input: missing profile input report evidence",
		)
	}
	if !hasReportRow(row.ReportRows, "profile_input_policy") {
		return PGOLTOTargetCPUCoverageRow{}, fmt.Errorf(
			"pgo optimizer input: pass report missing profile_input_policy row",
		)
	}
	if row.ValidationMetadata == nil ||
		row.ValidationMetadata.ProfileInputPolicy != string(ProfileInputUnused) ||
		row.ValidationMetadata.ProfileInputDigest != evidence.Digest ||
		row.ValidationMetadata.ProfileInputSchemaVersion != ProfileCollectionSchemaVersion {
		return PGOLTOTargetCPUCoverageRow{}, fmt.Errorf(
			"pgo optimizer input: missing validation metadata profile evidence",
		)
	}
	rejected := pgoInputEvidencePass()
	rejected.ProfileInputPolicy = ProfileInputGuidedRewrite
	if _, err := NewManager().RunWithOptions(
		pgoInputEvidenceProgram(),
		Options{ProfileInput: &profile},
		rejected,
	); err == nil {
		return PGOLTOTargetCPUCoverageRow{}, fmt.Errorf(
			"pgo optimizer input: profile-guided rewrite policy was accepted without dedicated validation",
		)
	}
	return PGOLTOTargetCPUCoverageRow{
		ID:                   PGOLTOTargetCPUPGOOptimizerInput,
		Name:                 "PGO input to optimizer",
		Status:               PGOLTOTargetCPUImplementedNarrow,
		OptimizerInput:       true,
		ChangesSafeSemantics: false,
		RequiredFacts: []string{
			"optimizer_profile_input_api",
			"pass_contract_profile_metadata",
			"translation_validation_for_profile_guided_decisions",
			"negative_safe_semantics_tests",
		},
		Reason: ("implemented_narrow:Options.ProfileInput validates profile " +
			"input and records profile_input_policy plus profile digest " +
			"in pass reports and validation metadata; profile-guided " +
			"rewrite policy rejected"),
		Evidence: ("compiler/internal/opt/opt_core.go::Options.ProfileInput; " +
			"compiler/internal/opt/opt_suite_test.go::" +
			"TestManagerAcceptsProfileInputAsValidatedMetadataWithoutChan" +
			"gingIR; compiler/internal/opt/opt_suite_test.go::" +
			"TestManagerRejectsProfileGuidedRewritePolicyUntilValidationE" +
			"xists; compiler/internal/validation/metadata_test.go::" +
			"TestBuildOptimizationValidationMetadataRecordsMachineCheckab" +
			"leEvidence"),
		Boundary: ("internal PGO input to optimizer is implemented only as " +
			"validated profile input API, profile_input_policy " +
			"pass-contract metadata, translation validation metadata " +
			"evidence, and negative safe-semantics rejection for " +
			"profile-guided rewrite policy; all registered passes keep " +
			"profile_input_policy unused, no profile-guided rewrite is " +
			"selected, no public flag exists, no codegen/LTO behavior " +
			"changes, and no performance claim is made"),
	}, nil
}

func targetCPUFeatureDetectionRow() PGOLTOTargetCPUCoverageRow {
	baselineOpt := x64.CodegenOptions{RegisterWidthBits: 64}
	evidence, err := baselineOpt.TargetFeatureEvidence()
	if err != nil {
		return targetCPUFeatureDetectionFailureRow(err)
	}
	if evidence.Source != string(x64.TargetFeatureSourcePortableBaseline) ||
		!evidence.PortableBaselineFallback ||
		evidence.ChangesSafeSemantics ||
		evidence.EnablesTargetSpecificOptimization ||
		!hasFeatureName(evidence.Features, string(x64.TargetFeatureSSE2)) {
		return targetCPUFeatureDetectionFailureRow(
			fmt.Errorf("target-cpu evidence: incomplete portable baseline evidence: %#v", evidence),
		)
	}
	if allowed, err := baselineOpt.AllowsTargetFeature(x64.TargetFeatureAVX2); err != nil ||
		allowed {
		return targetCPUFeatureDetectionFailureRow(
			fmt.Errorf("target-cpu evidence: default avx2 allowed=%v err=%v", allowed, err),
		)
	}
	_, err = (x64.CodegenOptions{
		RegisterWidthBits: 64,
		TargetFeatures: x64.TargetFeatures{
			Source:   x64.TargetFeatureSourceExplicit,
			Features: []x64.TargetFeature{x64.TargetFeatureAVX2},
		},
	}).EffectiveTargetFeatures()
	if err == nil {
		return targetCPUFeatureDetectionFailureRow(
			fmt.Errorf(
				"target-cpu evidence: explicit target feature set below portable baseline accepted",
			),
		)
	}
	return PGOLTOTargetCPUCoverageRow{
		ID:                   PGOLTOTargetCPUTargetCPUFeatureDetection,
		Name:                 "target-cpu feature detection",
		Status:               PGOLTOTargetCPUImplementedNarrow,
		OptimizerInput:       false,
		ChangesSafeSemantics: false,
		RequiredFacts: []string{
			"target_feature_model",
			"portable_baseline_fallback",
			"guarded_codegen_contract",
			"negative_safe_semantics_tests",
		},
		Reason: ("implemented_narrow:internal target feature model records " +
			"portable baseline fallback and guarded codegen contract " +
			"evidence; default x64/x32 evidence includes sse2 baseline " +
			"only and no target-specific rewrite is enabled"),
		Evidence: ("compiler/internal/backend/x64/target_features.go::" +
			"CodegenOptions.TargetFeatureEvidence; " +
			"compiler/internal/backend/x64/target_features_test.go::" +
			"TestTargetFeatureModelUsesPortableBaselineAndRejectsUnsafeDr" +
			"ift; compiler/compiler_suite_test.go::" +
			"TestNativeCodegenOptionsUsePortableTargetFeatureBaseline; " +
			"compiler/compiler_suite_test.go::" +
			"TestBuildOptionsExposeNoBackendSemanticMode"),
		Boundary: ("target-cpu feature detection foundation is evidence-only: " +
			"it provides an explicit internal target feature model, " +
			"portable baseline fallback, guarded codegen contract " +
			"queries, and negative safe-semantics rejection for explicit " +
			"features below baseline; no host CPU detector, no public " +
			"target-cpu BuildOptions field, no target-specific rewrite, " +
			"no optimizer input, no LTO/codegen behavior change, no " +
			"performance claim, and safe-program semantics unchanged"),
	}
}

func targetCPUFeatureDetectionFailureRow(err error) PGOLTOTargetCPUCoverageRow {
	row := pgoNotYetCoveredRow(
		PGOLTOTargetCPUTargetCPUFeatureDetection,
		"target-cpu feature detection",
		"target-cpu feature detection foundation failed self-validation",
		[]string{
			"target_feature_model",
			"portable_baseline_fallback",
			"guarded_codegen_contract",
			"negative_safe_semantics_tests",
		},
	)
	row.Evidence = err.Error()
	return row
}

func ltoIncrementalModuleSummaryRow() PGOLTOTargetCPUCoverageRow {
	depHash, err := cache.DepSigHashFromDepsWithInterfaceHashes(
		[]string{"math.core.add"},
		[]string{"math.core.Vec"},
		map[string]semantics.FuncSig{
			"math.core.add": {ParamTypes: []string{"i32", "i32"}, ReturnType: "i32", Public: true},
		},
		map[string]string{"math.core.Vec": "struct{x:i32,y:i32}"},
		map[string]string{"math.core": "sha256:p17ltoapi"},
	)
	if err != nil {
		return ltoIncrementalModuleSummaryFailureRow(err)
	}
	summary, err := cache.BuildIncrementalModuleSummary(cache.IncrementalModuleSummaryInput{
		Module:           "app.main",
		Target:           "linux-x64",
		BuildTag:         "alloc-stack-v1",
		Source:           []byte("module app.main\n"),
		DependencyHash:   depHash,
		PublicAPIHash:    "sha256:p17ltoapp",
		ExternalCallees:  []string{"math.core.add"},
		ExternalTypeDeps: []string{"math.core.Vec"},
	})
	if err != nil {
		return ltoIncrementalModuleSummaryFailureRow(err)
	}
	encoded, err := cache.MarshalIncrementalModuleSummary(summary)
	if err != nil {
		return ltoIncrementalModuleSummaryFailureRow(err)
	}
	decoded, err := cache.ParseIncrementalModuleSummary(encoded)
	if err != nil {
		return ltoIncrementalModuleSummaryFailureRow(err)
	}
	if decoded.SchemaVersion != cache.IncrementalModuleSummarySchemaVersion ||
		!hasReportRow(decoded.ValidationRows, "dependency_hash_contract") ||
		!hasReportRow(decoded.ValidationRows, "cross_module_signature_inputs") ||
		!hasReportRow(decoded.ValidationRows, "non_consumer_boundary") ||
		decoded.CodegenConsumer ||
		decoded.LinkerConsumer {
		return ltoIncrementalModuleSummaryFailureRow(
			fmt.Errorf("lto incremental module summary: missing self-validation evidence"),
		)
	}
	consumer := decoded
	consumer.CodegenConsumer = true
	if err := cache.ValidateIncrementalModuleSummary(consumer); err == nil {
		return ltoIncrementalModuleSummaryFailureRow(
			fmt.Errorf("lto incremental module summary: codegen consumer accepted"),
		)
	}
	return PGOLTOTargetCPUCoverageRow{
		ID:                   PGOLTOTargetCPULTOIncrementalModuleSummary,
		Name:                 "LTO/incremental module summary",
		Status:               PGOLTOTargetCPUImplementedNarrow,
		OptimizerInput:       false,
		ChangesSafeSemantics: false,
		RequiredFacts: []string{
			"module_summary_schema",
			"dependency_hash_contract",
			"cross_module_validation_row",
			"incremental_cache_negative_tests",
			"non_consumer_boundary",
		},
		Reason: ("implemented_narrow:tetra.incremental.module_summary.v1 " +
			"records source/public API/dependency hash contract evidence," +
			" cross-module validation rows, and non-consumer boundary; " +
			"no LTO optimizer is implemented"),
		Evidence: ("compiler/internal/cache/lto_summary.go::" +
			"IncrementalModuleSummary; " +
			"compiler/internal/cache/lto_summary_test.go::" +
			"TestIncrementalModuleSummaryV1RecordsDependencyHashContractA" +
			"ndRejectsConsumers; compiler/internal/opt/opt_suite_test.go:" +
			":TestPGOLTOTargetCPUCoverageAuditsP17PlanList"),
		Boundary: ("LTO/incremental module summary foundation is evidence-only: " +
			"it records module source hash, dependency hash contract, " +
			"public API hash, external callee/type dependency inputs, " +
			"cross-module validation rows, incremental negative tests, " +
			"and non-consumer boundary; no LTO optimizer, cross-module " +
			"inlining, linker consumer, codegen consumer, cache mode, " +
			"incremental speedup, public flag, performance claim, or " +
			"safe-program semantics change is made"),
	}
}

func ltoIncrementalModuleSummaryFailureRow(err error) PGOLTOTargetCPUCoverageRow {
	row := pgoNotYetCoveredRow(
		PGOLTOTargetCPULTOIncrementalModuleSummary,
		"LTO/incremental module summary",
		"LTO/incremental module summary foundation failed self-validation",
		[]string{
			"module_summary_schema",
			"dependency_hash_contract",
			"cross_module_validation_row",
			"incremental_cache_negative_tests",
			"non_consumer_boundary",
		},
	)
	row.Evidence = err.Error()
	return row
}

func pgoInputEvidenceProgram() *ir.IRProgram {
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

func pgoInputEvidencePass() Pass {
	return Pass{
		Name:                      "pgo-input-evidence-noop",
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
		ReportOutput:              "pgo-input-evidence.opt.json",
		ReportRows:                RequiredP17ReportRows(),
		NegativeTestMarker:        NegativeTestPassContractV1,
		ProfileInputPolicy:        ProfileInputUnused,
		Run:                       func(ctx *PassContext) error { return nil },
	}
}

func pgoNotYetCoveredRow(
	id PGOLTOTargetCPUID,
	name string,
	reason string,
	missing []string,
) PGOLTOTargetCPUCoverageRow {
	return PGOLTOTargetCPUCoverageRow{
		ID:                   id,
		Name:                 name,
		Status:               PGOLTOTargetCPUNotYetCovered,
		OptimizerInput:       false,
		ChangesSafeSemantics: false,
		MissingFacts:         append([]string(nil), missing...),
		Reason:               "not_yet_covered:" + reason,
		Evidence: ("P17.4 master-plan row; no implementation evidence has been " +
			"promoted for this row"),
		Boundary: reason + ("; no optimizer input, no codegen/LTO behavior, no " +
			"performance claim, and no safe-program semantic change"),
	}
}

func hasFeatureName(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// ---- scalar.go ----

const (
	minInt32 = int64(-1 << 31)
	maxInt32 = int64(1<<31 - 1)
)

func BasicScalarPass() Pass {
	return Pass{
		Name:                      "basic-scalar",
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
		ReportOutput:              "basic-scalar.opt.json",
		ReportRows:                RequiredP17ReportRows(),
		NegativeTestMarker:        NegativeTestPassContractV1,
		ProfileInputPolicy:        ProfileInputUnused,
		Run:                       runBasicScalarPass,
	}
}

func runBasicScalarPass(ctx *PassContext) error {
	prog := ctxProgram(ctx)
	if prog == nil {
		return fmt.Errorf("basic-scalar: missing IR program")
	}
	for i := range prog.Funcs {
		fn := &prog.Funcs[i]
		if !basicScalarFuncEligible(*fn) {
			continue
		}
		instrs := append([]ir.IRInstr(nil), fn.Instrs...)
		for iter := 0; iter < 16; iter++ {
			changed := false
			var stepChanged bool
			instrs, stepChanged = foldConstantsAndAlgebra(instrs)
			changed = changed || stepChanged
			instrs, stepChanged = propagateLocalCopies(instrs)
			changed = changed || stepChanged
			instrs, stepChanged = eliminateCommonLocalExpressions(instrs)
			changed = changed || stepChanged
			instrs, stepChanged = eliminateSimpleDeadStores(instrs)
			changed = changed || stepChanged
			if !changed {
				fn.Instrs = instrs
				break
			}
			if iter == 15 {
				return fmt.Errorf("basic-scalar: %s did not converge", fn.Name)
			}
		}
	}
	return nil
}

func basicScalarFuncEligible(fn ir.IRFunc) bool {
	if fn.Policy.HasBudget || fn.Policy.HasConsent {
		return false
	}
	for i, instr := range fn.Instrs {
		switch instr.Kind {
		case ir.IRLabel, ir.IRJmp, ir.IRJmpIfZero:
			return false
		case ir.IRReturn:
			return i == len(fn.Instrs)-1
		}
	}
	return true
}

func foldConstantsAndAlgebra(instrs []ir.IRInstr) ([]ir.IRInstr, bool) {
	out := make([]ir.IRInstr, 0, len(instrs))
	changed := false
	for _, instr := range instrs {
		switch instr.Kind {
		case ir.IRNegI32:
			if len(out) >= 1 && out[len(out)-1].Kind == ir.IRConstI32 {
				if folded, ok := checkedNegI32(out[len(out)-1].Imm); ok {
					out[len(out)-1] = constInstr(instr, folded)
					changed = true
					continue
				}
			}
		case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32,
			ir.IRDivI32, ir.IRModI32,
			ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
			ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
			if len(out) >= 2 {
				left := out[len(out)-2]
				right := out[len(out)-1]
				if left.Kind == ir.IRConstI32 && right.Kind == ir.IRConstI32 {
					if folded, ok := foldConstBinaryI32(instr.Kind, left.Imm, right.Imm); ok {
						out = out[:len(out)-2]
						out = append(out, constInstr(instr, folded))
						changed = true
						continue
					}
				}
				if applyAlgebraicSimplification(&out, instr.Kind) {
					changed = true
					continue
				}
			}
		}
		out = append(out, instr)
	}
	return out, changed
}

func applyAlgebraicSimplification(out *[]ir.IRInstr, kind ir.IRInstrKind) bool {
	instrs := *out
	if len(instrs) < 2 {
		return false
	}
	leftIdx := len(instrs) - 2
	rightIdx := len(instrs) - 1
	left := instrs[leftIdx]
	right := instrs[rightIdx]

	switch kind {
	case ir.IRAddI32:
		if isConstI32(right, 0) {
			*out = instrs[:rightIdx]
			return true
		}
		if isConstI32(left, 0) && isSinglePureValue(right) {
			*out = append(instrs[:leftIdx], right)
			return true
		}
	case ir.IRSubI32:
		if isConstI32(right, 0) {
			*out = instrs[:rightIdx]
			return true
		}
	case ir.IRMulI32:
		if isConstI32(right, 1) {
			*out = instrs[:rightIdx]
			return true
		}
		if isConstI32(left, 1) && isSinglePureValue(right) {
			*out = append(instrs[:leftIdx], right)
			return true
		}
		if isConstI32(right, 0) && isSinglePureValue(left) {
			*out = append(instrs[:leftIdx], constInstr(right, 0))
			return true
		}
		if isConstI32(left, 0) && isSinglePureValue(right) {
			*out = append(instrs[:leftIdx], constInstr(left, 0))
			return true
		}
	case ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
		ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		if sameSinglePureValue(left, right) {
			*out = append(instrs[:leftIdx], constInstr(right, sameValueComparisonResult(kind)))
			return true
		}
	}
	return false
}

func propagateLocalCopies(instrs []ir.IRInstr) ([]ir.IRInstr, bool) {
	out := make([]ir.IRInstr, 0, len(instrs))
	copies := map[int]int{}
	changed := false
	for _, instr := range instrs {
		switch instr.Kind {
		case ir.IRLoadLocal:
			src := resolveLocalCopy(copies, instr.Local)
			if src != instr.Local {
				instr.Local = src
				changed = true
			}
			out = append(out, instr)
		case ir.IRStoreLocal:
			dst := instr.Local
			invalidateLocalCopies(copies, dst)
			if len(out) > 0 && out[len(out)-1].Kind == ir.IRLoadLocal {
				src := resolveLocalCopy(copies, out[len(out)-1].Local)
				if src != out[len(out)-1].Local {
					out[len(out)-1].Local = src
					changed = true
				}
				if src != dst {
					copies[dst] = src
				}
			}
			out = append(out, instr)
		default:
			if clearsCopyFacts(instr.Kind) {
				clearLocalCopies(copies)
			}
			out = append(out, instr)
		}
	}
	return out, changed
}

func eliminateSimpleDeadStores(instrs []ir.IRInstr) ([]ir.IRInstr, bool) {
	remove := make([]bool, len(instrs))
	live := map[int]bool{}
	changed := false
	for i := len(instrs) - 1; i >= 0; i-- {
		if remove[i] {
			continue
		}
		instr := instrs[i]
		switch instr.Kind {
		case ir.IRLoadLocal:
			live[instr.Local] = true
		case ir.IRStoreLocal:
			if !live[instr.Local] {
				if start := deadStoreProducerStart(instrs, remove, i); start >= 0 {
					for j := start; j <= i; j++ {
						remove[j] = true
					}
					changed = true
					i = start
					continue
				}
			}
			delete(live, instr.Local)
		}
	}
	if !changed {
		return instrs, false
	}
	out := make([]ir.IRInstr, 0, len(instrs))
	for i, instr := range instrs {
		if !remove[i] {
			out = append(out, instr)
		}
	}
	return out, true
}

func deadStoreProducerStart(instrs []ir.IRInstr, remove []bool, storeIndex int) int {
	if storeIndex > 0 && !remove[storeIndex-1] && isDeadStoreProducer(instrs[storeIndex-1]) {
		return storeIndex - 1
	}
	if start := safeKnownLocalUnaryNegDeadStoreProducerStart(instrs, remove, storeIndex); start >= 0 {
		return start
	}
	if storeIndex >= 3 {
		start := storeIndex - 3
		if remove[start] || remove[start+1] || remove[start+2] {
			return -1
		}
		op := instrs[storeIndex-1].Kind
		if isNonTrappingDeadStoreExpressionOp(op) &&
			isSinglePureValue(instrs[start]) &&
			isSinglePureValue(instrs[start+1]) {
			return start
		}
		if isSafeKnownConstArithmeticDeadStoreExpression(
			op,
			instrs[start],
			instrs[start+1],
			instrs,
			remove,
			start,
		) {
			return start
		}
		if isSafeConstDenominatorDeadStoreExpression(op, instrs[start], instrs[start+1]) {
			return start
		}
	}
	return -1
}

func safeKnownLocalUnaryNegDeadStoreProducerStart(
	instrs []ir.IRInstr,
	remove []bool,
	storeIndex int,
) int {
	if storeIndex < 2 {
		return -1
	}
	start := storeIndex - 2
	if remove[start] || remove[start+1] {
		return -1
	}
	operand := instrs[start]
	op := instrs[start+1]
	if operand.Kind != ir.IRLoadLocal || op.Kind != ir.IRNegI32 {
		return -1
	}
	imm, ok := knownConstLocalBefore(instrs, remove, start, operand.Local)
	if !ok {
		return -1
	}
	if _, ok := checkedNegI32(imm); !ok {
		return -1
	}
	return start
}

func knownConstLocalBefore(
	instrs []ir.IRInstr,
	remove []bool,
	beforeIndex int,
	local int,
) (int32, bool) {
	known := map[int]int32{}
	for i := 0; i < beforeIndex; i++ {
		if remove[i] {
			continue
		}
		instr := instrs[i]
		switch instr.Kind {
		case ir.IRStoreLocal:
			delete(known, instr.Local)
			if i > 0 && !remove[i-1] && instrs[i-1].Kind == ir.IRConstI32 {
				known[instr.Local] = instrs[i-1].Imm
			}
		default:
			if clearsCopyFacts(instr.Kind) {
				known = map[int]int32{}
			}
		}
	}
	imm, ok := known[local]
	return imm, ok
}

func isSafeKnownConstArithmeticDeadStoreExpression(
	kind ir.IRInstrKind,
	left ir.IRInstr,
	right ir.IRInstr,
	instrs []ir.IRInstr,
	remove []bool,
	beforeIndex int,
) bool {
	switch kind {
	case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32, ir.IRDivI32, ir.IRModI32:
	default:
		return false
	}
	leftImm, ok := knownConstOperandBefore(left, instrs, remove, beforeIndex)
	if !ok {
		return false
	}
	rightImm, ok := knownConstOperandBefore(right, instrs, remove, beforeIndex)
	if !ok {
		return false
	}
	_, ok = foldConstBinaryI32(kind, leftImm, rightImm)
	return ok
}

func knownConstOperandBefore(
	instr ir.IRInstr,
	instrs []ir.IRInstr,
	remove []bool,
	beforeIndex int,
) (int32, bool) {
	switch instr.Kind {
	case ir.IRConstI32:
		return instr.Imm, true
	case ir.IRLoadLocal:
		return knownConstLocalBefore(instrs, remove, beforeIndex, instr.Local)
	default:
		return 0, false
	}
}

func isNonTrappingDeadStoreExpressionOp(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
		ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		return true
	default:
		return false
	}
}

func isSafeConstDenominatorDeadStoreExpression(
	kind ir.IRInstrKind,
	left ir.IRInstr,
	right ir.IRInstr,
) bool {
	switch kind {
	case ir.IRDivI32, ir.IRModI32:
		return left.Kind == ir.IRLoadLocal &&
			right.Kind == ir.IRConstI32 &&
			right.Imm != 0 &&
			right.Imm != -1
	default:
		return false
	}
}

type localExprKey struct {
	Kind  ir.IRInstrKind
	Left  localExprOperand
	Right localExprOperand
}

type localUnaryExprKey struct {
	Kind    ir.IRInstrKind
	Operand localExprOperand
}

type localExprOperandKind int

const (
	localExprOperandLocal localExprOperandKind = iota
	localExprOperandConst
)

type localExprOperand struct {
	Kind  localExprOperandKind
	Local int
	Imm   int32
}

func eliminateCommonLocalExpressions(instrs []ir.IRInstr) ([]ir.IRInstr, bool) {
	out := make([]ir.IRInstr, 0, len(instrs))
	exprs := map[localExprKey]int{}
	unaryExprs := map[localUnaryExprKey]int{}
	knownConsts := map[int]int32{}
	changed := false
	for i := 0; i < len(instrs); i++ {
		if i+1 < len(instrs) {
			keys := localUnaryExprKeysFromInstrs(instrs[i+1].Kind, instrs[i], knownConsts)
			if len(keys) > 0 {
				if cachedLocal, ok := cachedLocalForUnaryExprKeys(unaryExprs, keys); ok {
					out = append(
						out,
						ir.IRInstr{Kind: ir.IRLoadLocal, Local: cachedLocal, Pos: instrs[i+1].Pos},
					)
					i += 1
					changed = true
					continue
				}
				if i+2 < len(instrs) && instrs[i+2].Kind == ir.IRStoreLocal {
					storableKeys := storableLocalUnaryExprKeys(keys, instrs[i+2].Local)
					if len(storableKeys) > 0 {
						out = append(out, instrs[i], instrs[i+1], instrs[i+2])
						invalidateCachedExpressions(exprs, instrs[i+2].Local)
						invalidateCachedUnaryExpressions(unaryExprs, instrs[i+2].Local)
						updateKnownConstForExpressionStore(
							knownConsts,
							instrs[i+1].Kind,
							instrs[i],
							instrs[i],
							instrs[i+2].Local,
						)
						for _, key := range storableKeys {
							unaryExprs[key] = instrs[i+2].Local
						}
						i += 2
						continue
					}
				}
			}
		}
		if i+2 < len(instrs) {
			keys := localExprKeysFromInstrs(instrs[i+2].Kind, instrs[i], instrs[i+1], knownConsts)
			if len(keys) > 0 {
				if cachedLocal, ok := cachedLocalForExprKeys(exprs, keys); ok {
					out = append(
						out,
						ir.IRInstr{Kind: ir.IRLoadLocal, Local: cachedLocal, Pos: instrs[i+2].Pos},
					)
					i += 2
					changed = true
					continue
				}
				if i+3 < len(instrs) && instrs[i+3].Kind == ir.IRStoreLocal {
					storableKeys := storableLocalExprKeys(keys, instrs[i+3].Local)
					if len(storableKeys) > 0 {
						out = append(out, instrs[i], instrs[i+1], instrs[i+2], instrs[i+3])
						invalidateCachedExpressions(exprs, instrs[i+3].Local)
						invalidateCachedUnaryExpressions(unaryExprs, instrs[i+3].Local)
						updateKnownConstForExpressionStore(
							knownConsts,
							instrs[i+2].Kind,
							instrs[i],
							instrs[i+1],
							instrs[i+3].Local,
						)
						for _, key := range storableKeys {
							exprs[key] = instrs[i+3].Local
						}
						i += 3
						continue
					}
				}
			}
		}
		instr := instrs[i]
		switch instr.Kind {
		case ir.IRStoreLocal:
			invalidateCachedExpressions(exprs, instr.Local)
			invalidateCachedUnaryExpressions(unaryExprs, instr.Local)
			updateKnownConstForStore(knownConsts, instrs, i)
		default:
			if clearsCopyFacts(instr.Kind) {
				clearCachedExpressions(exprs)
				clearCachedUnaryExpressions(unaryExprs)
				clearKnownLocalConsts(knownConsts)
			}
		}
		out = append(out, instr)
	}
	return out, changed
}

func localUnaryExprKeysFromInstrs(
	kind ir.IRInstrKind,
	operandInstr ir.IRInstr,
	knownConsts map[int]int32,
) []localUnaryExprKey {
	keys := []localUnaryExprKey{}
	if key, ok := localUnaryExprKeyFromInstrs(kind, operandInstr); ok {
		keys = append(keys, key)
	}
	if key, ok := knownConstUnaryExprKeyFromInstrs(kind, operandInstr, knownConsts); ok &&
		!localUnaryExprKeysContain(keys, key) {
		keys = append(keys, key)
	}
	return keys
}

func cachedLocalForUnaryExprKeys(
	exprs map[localUnaryExprKey]int,
	keys []localUnaryExprKey,
) (int, bool) {
	for _, key := range keys {
		if cachedLocal, ok := exprs[key]; ok {
			return cachedLocal, true
		}
	}
	return 0, false
}

func storableLocalUnaryExprKeys(keys []localUnaryExprKey, storeLocal int) []localUnaryExprKey {
	out := make([]localUnaryExprKey, 0, len(keys))
	for _, key := range keys {
		if !localUnaryExprKeyUsesLocal(key, storeLocal) {
			out = append(out, key)
		}
	}
	return out
}

func localUnaryExprKeysContain(keys []localUnaryExprKey, needle localUnaryExprKey) bool {
	for _, key := range keys {
		if key == needle {
			return true
		}
	}
	return false
}

func knownConstUnaryExprKeyFromInstrs(
	kind ir.IRInstrKind,
	operandInstr ir.IRInstr,
	knownConsts map[int]int32,
) (localUnaryExprKey, bool) {
	if kind != ir.IRNegI32 {
		return localUnaryExprKey{}, false
	}
	operand, knownLocal, ok := knownConstExprOperand(operandInstr, knownConsts)
	if !ok || !knownLocal {
		return localUnaryExprKey{}, false
	}
	if _, ok := checkedNegI32(operand.Imm); !ok {
		return localUnaryExprKey{}, false
	}
	return localUnaryExprKey{Kind: kind, Operand: operand}, true
}

func localExprKeysFromInstrs(
	kind ir.IRInstrKind,
	leftInstr ir.IRInstr,
	rightInstr ir.IRInstr,
	knownConsts map[int]int32,
) []localExprKey {
	keys := []localExprKey{}
	if key, ok := localExprKeyFromInstrs(kind, leftInstr, rightInstr); ok {
		keys = append(keys, key)
	}
	if key, ok := knownConstBinaryExprKeyFromInstrs(kind, leftInstr, rightInstr, knownConsts); ok &&
		!localExprKeysContain(keys, key) {
		keys = append(keys, key)
	}
	return keys
}

func cachedLocalForExprKeys(exprs map[localExprKey]int, keys []localExprKey) (int, bool) {
	for _, key := range keys {
		if cachedLocal, ok := exprs[key]; ok {
			return cachedLocal, true
		}
	}
	return 0, false
}

func storableLocalExprKeys(keys []localExprKey, storeLocal int) []localExprKey {
	out := make([]localExprKey, 0, len(keys))
	for _, key := range keys {
		if !localExprKeyUsesLocal(key, storeLocal) {
			out = append(out, key)
		}
	}
	return out
}

func localExprKeysContain(keys []localExprKey, needle localExprKey) bool {
	for _, key := range keys {
		if key == needle {
			return true
		}
	}
	return false
}

func knownConstBinaryExprKeyFromInstrs(
	kind ir.IRInstrKind,
	leftInstr ir.IRInstr,
	rightInstr ir.IRInstr,
	knownConsts map[int]int32,
) (localExprKey, bool) {
	switch kind {
	case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32, ir.IRDivI32, ir.IRModI32,
		ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
		ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
	default:
		return localExprKey{}, false
	}
	left, leftKnownLocal, ok := knownConstExprOperand(leftInstr, knownConsts)
	if !ok {
		return localExprKey{}, false
	}
	right, rightKnownLocal, ok := knownConstExprOperand(rightInstr, knownConsts)
	if !ok || (!leftKnownLocal && !rightKnownLocal) {
		return localExprKey{}, false
	}
	if _, ok := foldConstBinaryI32(kind, left.Imm, right.Imm); !ok {
		return localExprKey{}, false
	}
	return canonicalLocalExprKey(kind, left, right), true
}

func knownConstExprOperand(
	instr ir.IRInstr,
	knownConsts map[int]int32,
) (localExprOperand, bool, bool) {
	switch instr.Kind {
	case ir.IRConstI32:
		return localExprOperand{Kind: localExprOperandConst, Imm: instr.Imm}, false, true
	case ir.IRLoadLocal:
		imm, ok := knownConsts[instr.Local]
		if !ok {
			return localExprOperand{}, false, false
		}
		return localExprOperand{Kind: localExprOperandConst, Imm: imm}, true, true
	default:
		return localExprOperand{}, false, false
	}
}

func updateKnownConstForStore(knownConsts map[int]int32, instrs []ir.IRInstr, storeIndex int) {
	store := instrs[storeIndex]
	delete(knownConsts, store.Local)
	if storeIndex > 0 && instrs[storeIndex-1].Kind == ir.IRConstI32 {
		knownConsts[store.Local] = instrs[storeIndex-1].Imm
	}
}

func updateKnownConstForExpressionStore(
	knownConsts map[int]int32,
	kind ir.IRInstrKind,
	leftInstr ir.IRInstr,
	rightInstr ir.IRInstr,
	storeLocal int,
) {
	delete(knownConsts, storeLocal)
	left, _, ok := knownConstExprOperand(leftInstr, knownConsts)
	if !ok {
		return
	}
	if kind == ir.IRNegI32 {
		if folded, ok := checkedNegI32(left.Imm); ok {
			knownConsts[storeLocal] = folded
		}
		return
	}
	right, _, ok := knownConstExprOperand(rightInstr, knownConsts)
	if !ok {
		return
	}
	if folded, ok := foldConstBinaryI32(kind, left.Imm, right.Imm); ok {
		knownConsts[storeLocal] = folded
	}
}

func localUnaryExprKeyFromInstrs(
	kind ir.IRInstrKind,
	operandInstr ir.IRInstr,
) (localUnaryExprKey, bool) {
	if kind != ir.IRNegI32 {
		return localUnaryExprKey{}, false
	}
	operand, ok := localExprOperandFromInstr(operandInstr)
	if !ok || operand.Kind != localExprOperandLocal {
		return localUnaryExprKey{}, false
	}
	return localUnaryExprKey{Kind: kind, Operand: operand}, true
}

func localExprKeyFromInstrs(
	kind ir.IRInstrKind,
	leftInstr ir.IRInstr,
	rightInstr ir.IRInstr,
) (localExprKey, bool) {
	if !isPureLocalBinaryOp(kind) {
		return localExprKey{}, false
	}
	left, ok := localExprOperandFromInstr(leftInstr)
	if !ok {
		return localExprKey{}, false
	}
	right, ok := localExprOperandFromInstr(rightInstr)
	if !ok {
		return localExprKey{}, false
	}
	if left.Kind != localExprOperandLocal && right.Kind != localExprOperandLocal {
		return localExprKey{}, false
	}
	if !localExprOperandsSafeForCSE(kind, left, right) {
		return localExprKey{}, false
	}
	return canonicalLocalExprKey(kind, left, right), true
}

func localExprOperandFromInstr(instr ir.IRInstr) (localExprOperand, bool) {
	switch instr.Kind {
	case ir.IRLoadLocal:
		return localExprOperand{Kind: localExprOperandLocal, Local: instr.Local}, true
	case ir.IRConstI32:
		return localExprOperand{Kind: localExprOperandConst, Imm: instr.Imm}, true
	default:
		return localExprOperand{}, false
	}
}

func canonicalLocalExprKey(
	kind ir.IRInstrKind,
	left localExprOperand,
	right localExprOperand,
) localExprKey {
	if isCommutativeLocalBinaryOp(kind) && compareLocalExprOperands(right, left) < 0 {
		left, right = right, left
	}
	if isMirroredComparisonLocalBinaryOp(kind) && compareLocalExprOperands(right, left) < 0 {
		left, right = right, left
		kind = mirroredComparisonLocalBinaryOp(kind)
	}
	return localExprKey{Kind: kind, Left: left, Right: right}
}

func compareLocalExprOperands(left localExprOperand, right localExprOperand) int {
	if left.Kind != right.Kind {
		if left.Kind < right.Kind {
			return -1
		}
		return 1
	}
	switch left.Kind {
	case localExprOperandLocal:
		return compareInt(left.Local, right.Local)
	case localExprOperandConst:
		return compareInt32(left.Imm, right.Imm)
	default:
		return 0
	}
}

func compareInt(left int, right int) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func compareInt32(left int32, right int32) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func isPureLocalBinaryOp(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32,
		ir.IRDivI32, ir.IRModI32,
		ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
		ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		return true
	default:
		return false
	}
}

func localExprOperandsSafeForCSE(
	kind ir.IRInstrKind,
	left localExprOperand,
	right localExprOperand,
) bool {
	switch kind {
	case ir.IRDivI32, ir.IRModI32:
		return left.Kind == localExprOperandLocal &&
			right.Kind == localExprOperandConst &&
			right.Imm != 0 &&
			right.Imm != -1
	default:
		return true
	}
}

func isCommutativeLocalBinaryOp(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRAddI32, ir.IRMulI32, ir.IRCmpEqI32, ir.IRCmpNeI32:
		return true
	default:
		return false
	}
}

func isMirroredComparisonLocalBinaryOp(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRCmpLtI32, ir.IRCmpGtI32, ir.IRCmpLeI32, ir.IRCmpGeI32:
		return true
	default:
		return false
	}
}

func mirroredComparisonLocalBinaryOp(kind ir.IRInstrKind) ir.IRInstrKind {
	switch kind {
	case ir.IRCmpLtI32:
		return ir.IRCmpGtI32
	case ir.IRCmpGtI32:
		return ir.IRCmpLtI32
	case ir.IRCmpLeI32:
		return ir.IRCmpGeI32
	case ir.IRCmpGeI32:
		return ir.IRCmpLeI32
	default:
		return kind
	}
}

func invalidateCachedExpressions(exprs map[localExprKey]int, local int) {
	for key, cachedLocal := range exprs {
		if cachedLocal == local || localExprKeyUsesLocal(key, local) {
			delete(exprs, key)
		}
	}
}

func localExprKeyUsesLocal(key localExprKey, local int) bool {
	return localExprOperandUsesLocal(key.Left, local) || localExprOperandUsesLocal(key.Right, local)
}

func invalidateCachedUnaryExpressions(exprs map[localUnaryExprKey]int, local int) {
	for key, cachedLocal := range exprs {
		if cachedLocal == local || localUnaryExprKeyUsesLocal(key, local) {
			delete(exprs, key)
		}
	}
}

func localUnaryExprKeyUsesLocal(key localUnaryExprKey, local int) bool {
	return localExprOperandUsesLocal(key.Operand, local)
}

func localExprOperandUsesLocal(operand localExprOperand, local int) bool {
	return operand.Kind == localExprOperandLocal && operand.Local == local
}

func clearCachedExpressions(exprs map[localExprKey]int) {
	for key := range exprs {
		delete(exprs, key)
	}
}

func clearCachedUnaryExpressions(exprs map[localUnaryExprKey]int) {
	for key := range exprs {
		delete(exprs, key)
	}
}

func foldConstBinaryI32(kind ir.IRInstrKind, left int32, right int32) (int32, bool) {
	switch kind {
	case ir.IRAddI32:
		return checkedI32(int64(left) + int64(right))
	case ir.IRSubI32:
		return checkedI32(int64(left) - int64(right))
	case ir.IRMulI32:
		return checkedI32(int64(left) * int64(right))
	case ir.IRDivI32:
		if right == 0 || right == -1 {
			return 0, false
		}
		return left / right, true
	case ir.IRModI32:
		if right == 0 || right == -1 {
			return 0, false
		}
		return left % right, true
	case ir.IRCmpEqI32:
		return boolI32(left == right), true
	case ir.IRCmpLtI32:
		return boolI32(left < right), true
	case ir.IRCmpGtI32:
		return boolI32(left > right), true
	case ir.IRCmpGeI32:
		return boolI32(left >= right), true
	case ir.IRCmpLeI32:
		return boolI32(left <= right), true
	case ir.IRCmpNeI32:
		return boolI32(left != right), true
	default:
		return 0, false
	}
}

func checkedNegI32(v int32) (int32, bool) {
	if int64(v) == minInt32 {
		return 0, false
	}
	return -v, true
}

func checkedI32(v int64) (int32, bool) {
	if v < minInt32 || v > maxInt32 {
		return 0, false
	}
	return int32(v), true
}

func boolI32(v bool) int32 {
	if v {
		return 1
	}
	return 0
}

func constInstr(from ir.IRInstr, v int32) ir.IRInstr {
	return ir.IRInstr{Kind: ir.IRConstI32, Imm: v, Pos: from.Pos}
}

func isConstI32(instr ir.IRInstr, v int32) bool {
	return instr.Kind == ir.IRConstI32 && instr.Imm == v
}

func isSinglePureValue(instr ir.IRInstr) bool {
	switch instr.Kind {
	case ir.IRConstI32, ir.IRLoadLocal:
		return true
	default:
		return false
	}
}

func sameSinglePureValue(left ir.IRInstr, right ir.IRInstr) bool {
	if left.Kind != right.Kind || !isSinglePureValue(left) {
		return false
	}
	switch left.Kind {
	case ir.IRConstI32:
		return left.Imm == right.Imm
	case ir.IRLoadLocal:
		return left.Local == right.Local
	default:
		return false
	}
}

func sameValueComparisonResult(kind ir.IRInstrKind) int32 {
	switch kind {
	case ir.IRCmpEqI32, ir.IRCmpGeI32, ir.IRCmpLeI32:
		return 1
	default:
		return 0
	}
}

func isDeadStoreProducer(instr ir.IRInstr) bool {
	return isSinglePureValue(instr)
}

func resolveLocalCopy(copies map[int]int, local int) int {
	seen := map[int]bool{}
	cur := local
	for {
		if seen[cur] {
			return cur
		}
		seen[cur] = true
		next, ok := copies[cur]
		if !ok {
			return cur
		}
		cur = next
	}
}

func invalidateLocalCopies(copies map[int]int, local int) {
	for dst, src := range copies {
		if dst == local || src == local {
			delete(copies, dst)
		}
	}
}

func clearLocalCopies(copies map[int]int) {
	for dst := range copies {
		delete(copies, dst)
	}
}

func clearsCopyFacts(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRCall, ir.IRStoreGlobal, ir.IRIndexStoreI32, ir.IRIndexStoreU8,
		ir.IRIndexStoreU16, ir.IRMemWriteI32, ir.IRMemWriteU8,
		ir.IRMemWritePtr, ir.IRMemWriteArchPtr, ir.IRMemWriteI32Offset,
		ir.IRMemWriteU8Offset, ir.IRMemWritePtrOffset,
		ir.IRMemWriteArchPtrOffset, ir.IRMmioWriteI32, ir.IRCtxSwitch,
		ir.IRAtomicStorePtr, ir.IRAtomicExchangePtr, ir.IRAtomicFetchAddPtr,
		ir.IRAtomicFetchSubPtr, ir.IRAtomicFetchAndPtr, ir.IRAtomicFetchOrPtr,
		ir.IRAtomicFetchXorPtr, ir.IRAtomicCompareExchangePtr,
		ir.IRAtomicStoreI32, ir.IRAtomicExchangeI32,
		ir.IRAtomicCompareExchangeI32, ir.IRAtomicFetchAddI32,
		ir.IRAtomicFetchSubI32, ir.IRAtomicFetchAndI32,
		ir.IRAtomicFetchOrI32, ir.IRAtomicFetchXorI32,
		ir.IRAtomicStoreI64, ir.IRAtomicExchangeI64,
		ir.IRAtomicCompareExchangeI64, ir.IRAtomicFetchAddI64,
		ir.IRAtomicFetchSubI64, ir.IRAtomicFetchAndI64,
		ir.IRAtomicFetchOrI64, ir.IRAtomicFetchXorI64,
		ir.IRAtomicStoreI8, ir.IRAtomicExchangeI8,
		ir.IRAtomicCompareExchangeI8, ir.IRAtomicFetchAddI8,
		ir.IRAtomicFetchSubI8, ir.IRAtomicFetchAndI8,
		ir.IRAtomicFetchOrI8, ir.IRAtomicFetchXorI8,
		ir.IRAtomicStoreI16, ir.IRAtomicExchangeI16,
		ir.IRAtomicCompareExchangeI16, ir.IRAtomicFetchAddI16,
		ir.IRAtomicFetchSubI16, ir.IRAtomicFetchAndI16,
		ir.IRAtomicFetchOrI16, ir.IRAtomicFetchXorI16:
		return true
	default:
		return false
	}
}

// ---- sccp.go ----

type sccpState struct {
	decisions []PassDecision
}

func SCCPPass() Pass {
	state := &sccpState{}
	return Pass{
		Name:                      "sccp-constant-branch",
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
		ReportOutput:              "sccp-constant-branch.opt.json",
		ReportRows:                RequiredP17ReportRows(),
		NegativeTestMarker:        NegativeTestPassContractV1,
		ProfileInputPolicy:        ProfileInputUnused,
		Run:                       state.run,
		Decisions:                 state.reportDecisions,
	}
}

func (s *sccpState) run(ctx *PassContext) error {
	prog := ctxProgram(ctx)
	if prog == nil {
		return fmt.Errorf("sccp-constant-branch: missing IR program")
	}
	s.decisions = nil
	for i := range prog.Funcs {
		fn := &prog.Funcs[i]
		if fn.Policy.HasBudget || fn.Policy.HasConsent {
			s.decisions = append(
				s.decisions,
				PassDecision{
					Action: "not_folded",
					Caller: fn.Name,
					Site:   0,
					Reason: "policy_guarded_function",
				},
			)
			continue
		}
		fn.Instrs = s.rewriteFunc(fn.Name, fn.Instrs)
	}
	return nil
}

func (s *sccpState) reportDecisions() []PassDecision {
	return append([]PassDecision(nil), s.decisions...)
}

func (s *sccpState) rewriteFunc(fnName string, instrs []ir.IRInstr) []ir.IRInstr {
	out := make([]ir.IRInstr, 0, len(instrs))
	knownLocals := map[int]int32{}
	knownZeroLocals := map[int]bool{}
	knownNonZeroLocals := map[int]bool{}
	constStack := make([]knownStackValue, 0)
	labelIncoming := countLabelIncoming(instrs)
	labelIndexes := indexLabels(instrs)
	pendingLabelFacts := map[int]map[int]int32{}
	pendingLabelZeroFacts := map[int]map[int]bool{}
	pendingLabelNonZeroFacts := map[int]map[int]bool{}
	for i := 0; i < len(instrs); i++ {
		if i+1 < len(instrs) && instrs[i].Kind == ir.IRConstI32 &&
			instrs[i+1].Kind == ir.IRJmpIfZero {
			branch := instrs[i+1]
			if instrs[i].Imm == 0 {
				out = append(out, ir.IRInstr{Kind: ir.IRJmp, Label: branch.Label, Pos: branch.Pos})
				s.decisions = append(
					s.decisions,
					PassDecision{
						Action: "folded_const_zero_branch",
						Caller: fnName,
						Site:   i,
						Reason: "constant_condition",
					},
				)
				s.propagateKnownLocalsThroughFoldedZeroBranch(
					fnName,
					instrs,
					labelIndexes,
					labelIncoming,
					knownLocals,
					i+1,
					branch.Label,
					pendingLabelFacts,
				)
				clearKnownStack(&constStack)
				next, pruned := skipFallthroughUntilLabel(instrs, i+2)
				if pruned > 0 {
					s.decisions = append(
						s.decisions,
						PassDecision{
							Action: "pruned_unreachable_fallthrough",
							Caller: fnName,
							Site:   i + 2,
							Reason: "constant_branch_reachability",
						},
					)
				}
				i = next - 1
				continue
			} else {
				s.decisions = append(
					s.decisions,
					PassDecision{
						Action: "folded_const_nonzero_fallthrough",
						Caller: fnName,
						Site:   i,
						Reason: "constant_condition",
					},
				)
				s.propagateKnownLocalsThroughFoldedNonzeroFallthrough(
					fnName,
					instrs,
					labelIncoming,
					knownLocals,
					i+1,
					i+2,
					pendingLabelFacts,
				)
				clearKnownStack(&constStack)
			}
			i++
			continue
		}
		if i+1 < len(instrs) && instrs[i].Kind == ir.IRLoadLocal &&
			instrs[i+1].Kind == ir.IRJmpIfZero {
			branch := instrs[i+1]
			local := instrs[i].Local
			if value, ok := knownLocals[local]; ok {
				if value == 0 {
					out = append(
						out,
						ir.IRInstr{Kind: ir.IRJmp, Label: branch.Label, Pos: branch.Pos},
					)
					s.decisions = append(
						s.decisions,
						PassDecision{
							Action: "folded_known_local_zero_branch",
							Caller: fnName,
							Site:   i,
							Reason: "constant_local_condition",
						},
					)
					s.propagateKnownLocalsThroughFoldedZeroBranch(
						fnName,
						instrs,
						labelIndexes,
						labelIncoming,
						knownLocals,
						i+1,
						branch.Label,
						pendingLabelFacts,
					)
					clearKnownStack(&constStack)
					next, pruned := skipFallthroughUntilLabel(instrs, i+2)
					if pruned > 0 {
						s.decisions = append(
							s.decisions,
							PassDecision{
								Action: "pruned_unreachable_fallthrough",
								Caller: fnName,
								Site:   i + 2,
								Reason: "constant_branch_reachability",
							},
						)
					}
					i = next - 1
					continue
				}
				s.decisions = append(
					s.decisions,
					PassDecision{
						Action: "folded_known_local_nonzero_fallthrough",
						Caller: fnName,
						Site:   i,
						Reason: "constant_local_condition",
					},
				)
				s.propagateKnownLocalsThroughFoldedNonzeroFallthrough(
					fnName,
					instrs,
					labelIncoming,
					knownLocals,
					i+1,
					i+2,
					pendingLabelFacts,
				)
				clearKnownStack(&constStack)
				i++
				continue
			}
			if knownZeroLocals[local] {
				out = append(out, ir.IRInstr{Kind: ir.IRJmp, Label: branch.Label, Pos: branch.Pos})
				s.decisions = append(
					s.decisions,
					PassDecision{
						Action: "folded_path_local_zero_branch",
						Caller: fnName,
						Site:   i,
						Reason: "path_local_condition",
					},
				)
				s.propagatePathLocalZeroThroughFoldedZeroBranch(
					fnName,
					instrs,
					labelIndexes,
					labelIncoming,
					local,
					i+1,
					branch.Label,
					pendingLabelZeroFacts,
				)
				clearKnownStack(&constStack)
				next, pruned := skipFallthroughUntilLabel(instrs, i+2)
				if pruned > 0 {
					s.decisions = append(
						s.decisions,
						PassDecision{
							Action: "pruned_unreachable_fallthrough",
							Caller: fnName,
							Site:   i + 2,
							Reason: "constant_branch_reachability",
						},
					)
				}
				i = next - 1
				continue
			}
			if knownNonZeroLocals[local] {
				s.decisions = append(
					s.decisions,
					PassDecision{
						Action: "folded_path_local_nonzero_fallthrough",
						Caller: fnName,
						Site:   i,
						Reason: "path_local_condition",
					},
				)
				s.propagatePathLocalNonZeroThroughFoldedNonzeroFallthrough(
					fnName,
					instrs,
					labelIncoming,
					local,
					i+1,
					i+2,
					pendingLabelNonZeroFacts,
				)
				clearKnownStack(&constStack)
				i++
				continue
			}
			out = append(out, instrs[i], branch)
			s.decisions = append(
				s.decisions,
				PassDecision{
					Action: "not_folded",
					Caller: fnName,
					Site:   i + 1,
					Reason: "dynamic_condition",
				},
			)
			s.propagatePathLocalZeroThroughDynamicBranchTarget(
				fnName,
				instrs,
				labelIndexes,
				labelIncoming,
				local,
				i+1,
				branch.Label,
				pendingLabelZeroFacts,
			)
			setKnownLocalNonZero(knownLocals, knownZeroLocals, knownNonZeroLocals, local)
			s.decisions = append(
				s.decisions,
				PassDecision{
					Action: "derived_path_local_nonzero_fallthrough",
					Caller: fnName,
					Site:   i + 1,
					Reason: "dynamic_branch_fallthrough",
				},
			)
			clearKnownStack(&constStack)
			i++
			continue
		}
		if i+1 < len(instrs) && i >= 1 && instrs[i+1].Kind == ir.IRJmpIfZero &&
			isPureLocalUnaryOp(instrs[i].Kind) {
			operand, operandOK := knownBranchOperandConst(instrs[i-1], knownLocals)
			value, folded := foldConstUnaryI32(instrs[i].Kind, operand)
			if operandOK && folded && len(out) >= 1 {
				branch := instrs[i+1]
				out = out[:len(out)-1]
				if value == 0 {
					out = append(
						out,
						ir.IRInstr{Kind: ir.IRJmp, Label: branch.Label, Pos: branch.Pos},
					)
					s.decisions = append(
						s.decisions,
						PassDecision{
							Action: "folded_const_unary_expr_zero_branch",
							Caller: fnName,
							Site:   i - 1,
							Reason: "constant_unary_expression_condition",
						},
					)
					s.propagateKnownLocalsThroughFoldedZeroBranch(
						fnName,
						instrs,
						labelIndexes,
						labelIncoming,
						knownLocals,
						i+1,
						branch.Label,
						pendingLabelFacts,
					)
					clearKnownStack(&constStack)
					next, pruned := skipFallthroughUntilLabel(instrs, i+2)
					if pruned > 0 {
						s.decisions = append(
							s.decisions,
							PassDecision{
								Action: "pruned_unreachable_fallthrough",
								Caller: fnName,
								Site:   i + 2,
								Reason: "constant_branch_reachability",
							},
						)
					}
					i = next - 1
					continue
				}
				s.decisions = append(
					s.decisions,
					PassDecision{
						Action: "folded_const_unary_expr_nonzero_fallthrough",
						Caller: fnName,
						Site:   i - 1,
						Reason: "constant_unary_expression_condition",
					},
				)
				s.propagateKnownLocalsThroughFoldedNonzeroFallthrough(
					fnName,
					instrs,
					labelIncoming,
					knownLocals,
					i+1,
					i+2,
					pendingLabelFacts,
				)
				clearKnownStack(&constStack)
				i++
				continue
			}
		}
		if i+1 < len(instrs) && i >= 2 && instrs[i+1].Kind == ir.IRJmpIfZero &&
			isPureLocalBinaryOp(instrs[i].Kind) {
			left, leftOK := knownBranchOperandConst(instrs[i-2], knownLocals)
			right, rightOK := knownBranchOperandConst(instrs[i-1], knownLocals)
			value, folded := foldConstBinaryI32(instrs[i].Kind, left, right)
			if leftOK && rightOK && folded && len(out) >= 2 {
				branch := instrs[i+1]
				out = out[:len(out)-2]
				if value == 0 {
					out = append(
						out,
						ir.IRInstr{Kind: ir.IRJmp, Label: branch.Label, Pos: branch.Pos},
					)
					s.decisions = append(
						s.decisions,
						PassDecision{
							Action: "folded_const_expr_zero_branch",
							Caller: fnName,
							Site:   i - 2,
							Reason: "constant_expression_condition",
						},
					)
					s.propagateKnownLocalsThroughFoldedZeroBranch(
						fnName,
						instrs,
						labelIndexes,
						labelIncoming,
						knownLocals,
						i+1,
						branch.Label,
						pendingLabelFacts,
					)
					clearKnownStack(&constStack)
					next, pruned := skipFallthroughUntilLabel(instrs, i+2)
					if pruned > 0 {
						s.decisions = append(
							s.decisions,
							PassDecision{
								Action: "pruned_unreachable_fallthrough",
								Caller: fnName,
								Site:   i + 2,
								Reason: "constant_branch_reachability",
							},
						)
					}
					i = next - 1
					continue
				}
				s.decisions = append(
					s.decisions,
					PassDecision{
						Action: "folded_const_expr_nonzero_fallthrough",
						Caller: fnName,
						Site:   i - 2,
						Reason: "constant_expression_condition",
					},
				)
				s.propagateKnownLocalsThroughFoldedNonzeroFallthrough(
					fnName,
					instrs,
					labelIncoming,
					knownLocals,
					i+1,
					i+2,
					pendingLabelFacts,
				)
				clearKnownStack(&constStack)
				i++
				continue
			}
		}
		if i+1 < len(instrs) && i >= 2 && instrs[i+1].Kind == ir.IRJmpIfZero {
			if fact, ok := zeroComparisonLocalFact(instrs, i); ok {
				branch := instrs[i+1]
				out = append(out, instrs[i], branch)
				s.decisions = append(
					s.decisions,
					PassDecision{
						Action: "not_folded",
						Caller: fnName,
						Site:   i + 1,
						Reason: "dynamic_condition",
					},
				)
				s.propagateComparisonPathLocalThroughDynamicBranchTarget(
					fnName,
					instrs,
					labelIndexes,
					labelIncoming,
					fact,
					i+1,
					branch.Label,
					pendingLabelZeroFacts,
					pendingLabelNonZeroFacts,
				)
				if fact.FallthroughZero {
					setKnownLocalZero(knownLocals, knownZeroLocals, knownNonZeroLocals, fact.Local)
					s.decisions = append(
						s.decisions,
						PassDecision{
							Action: "derived_comparison_path_local_zero_fallthrough",
							Caller: fnName,
							Site:   i + 1,
							Reason: fact.FallthroughReason + "_fallthrough",
						},
					)
				} else {
					setKnownLocalNonZero(knownLocals, knownZeroLocals, knownNonZeroLocals, fact.Local)
					s.decisions = append(
						s.decisions,
						PassDecision{
							Action: "derived_comparison_path_local_nonzero_fallthrough",
							Caller: fnName,
							Site:   i + 1,
							Reason: fact.FallthroughReason + "_fallthrough",
						},
					)
				}
				clearKnownStack(&constStack)
				i++
				continue
			}
		}
		if i+1 < len(instrs) && instrs[i+1].Kind == ir.IRJmpIfZero {
			s.decisions = append(
				s.decisions,
				PassDecision{
					Action: "not_folded",
					Caller: fnName,
					Site:   i + 1,
					Reason: "dynamic_condition",
				},
			)
		}
		instr := instrs[i]
		switch instr.Kind {
		case ir.IRConstI32:
			pushKnownStackConst(&constStack, instr.Imm)
		case ir.IRLoadLocal:
			if value, ok := knownLocals[instr.Local]; ok {
				pushKnownStackConst(&constStack, value)
			} else {
				pushUnknownStackValue(&constStack)
			}
		case ir.IRStoreLocal:
			if value, ok := popKnownStackConst(&constStack); ok {
				setKnownLocalConst(
					knownLocals,
					knownZeroLocals,
					knownNonZeroLocals,
					instr.Local,
					value,
				)
				s.decisions = append(
					s.decisions,
					PassDecision{
						Action: "tracked_known_local_store",
						Caller: fnName,
						Site:   i,
						Reason: "constant_stack_store",
					},
				)
			} else {
				deleteKnownLocalFact(knownLocals, knownZeroLocals, knownNonZeroLocals, instr.Local)
			}
		case ir.IRNegI32:
			operand, ok := popKnownStackConst(&constStack)
			if !ok {
				pushUnknownStackValue(&constStack)
			} else if value, folded := foldConstUnaryI32(instr.Kind, operand); folded {
				pushKnownStackConst(&constStack, value)
			} else {
				pushUnknownStackValue(&constStack)
			}
		case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32, ir.IRDivI32, ir.IRModI32,
			ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32, ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
			right, rightOK := popKnownStackConst(&constStack)
			left, leftOK := popKnownStackConst(&constStack)
			if leftOK && rightOK {
				if value, folded := foldConstBinaryI32(instr.Kind, left, right); folded {
					pushKnownStackConst(&constStack, value)
				} else {
					pushUnknownStackValue(&constStack)
				}
			} else {
				pushUnknownStackValue(&constStack)
			}
		case ir.IRJmp:
			if reason, ok := knownLocalSinglePredecessorLabelReason(
				instrs,
				labelIndexes,
				labelIncoming,
				i,
				instr.Label,
			); ok &&
				len(knownLocals) > 0 {
				pendingLabelFacts[instr.Label] = cloneKnownLocalConsts(knownLocals)
				s.decisions = append(
					s.decisions,
					PassDecision{
						Action: "propagated_known_local_single_predecessor",
						Caller: fnName,
						Site:   i,
						Reason: reason,
					},
				)
			}
			clearKnownLocalFacts(knownLocals, knownZeroLocals, knownNonZeroLocals)
			clearKnownStack(&constStack)
		case ir.IRLabel:
			clearKnownLocalFacts(knownLocals, knownZeroLocals, knownNonZeroLocals)
			clearKnownStack(&constStack)
			if facts, ok := pendingLabelFacts[instr.Label]; ok {
				applyKnownLocalConsts(knownLocals, knownZeroLocals, knownNonZeroLocals, facts)
			}
			if facts, ok := pendingLabelZeroFacts[instr.Label]; ok {
				applyKnownLocalZeros(knownLocals, knownZeroLocals, knownNonZeroLocals, facts)
			}
			if facts, ok := pendingLabelNonZeroFacts[instr.Label]; ok {
				applyKnownLocalNonZeros(knownLocals, knownZeroLocals, knownNonZeroLocals, facts)
			}
			delete(pendingLabelFacts, instr.Label)
			delete(pendingLabelZeroFacts, instr.Label)
			delete(pendingLabelNonZeroFacts, instr.Label)
		case ir.IRJmpIfZero:
			clearKnownLocalFacts(knownLocals, knownZeroLocals, knownNonZeroLocals)
			clearKnownStack(&constStack)
		default:
			if clearsCopyFacts(instr.Kind) {
				clearKnownLocalFacts(knownLocals, knownZeroLocals, knownNonZeroLocals)
				clearKnownStack(&constStack)
			} else {
				updateKnownStackForOpaqueInstr(&constStack, instr)
			}
		}
		out = append(out, instr)
	}
	return out
}

type knownStackValue struct {
	value int32
	known bool
}

func pushKnownStackConst(stack *[]knownStackValue, value int32) {
	*stack = append(*stack, knownStackValue{value: value, known: true})
}

func pushUnknownStackValue(stack *[]knownStackValue) {
	*stack = append(*stack, knownStackValue{})
}

func popKnownStackConst(stack *[]knownStackValue) (int32, bool) {
	if len(*stack) == 0 {
		return 0, false
	}
	value := (*stack)[len(*stack)-1]
	*stack = (*stack)[:len(*stack)-1]
	return value.value, value.known
}

func clearKnownStack(stack *[]knownStackValue) {
	*stack = (*stack)[:0]
}

func updateKnownStackForOpaqueInstr(stack *[]knownStackValue, instr ir.IRInstr) {
	switch instr.Kind {
	case ir.IRStrLit, ir.IRMakeSliceU8, ir.IRMakeSliceU16, ir.IRMakeSliceI32,
		ir.IRStackSliceU8, ir.IRStackSliceU16, ir.IRStackSliceI32,
		ir.IRRegionMakeSliceU8, ir.IRRegionMakeSliceU16, ir.IRRegionMakeSliceI32,
		ir.IRRawSliceFromParts, ir.IRSliceWindow, ir.IRSlicePrefix, ir.IRSliceSuffix:
		clearKnownStack(stack)
	case ir.IRCall, ir.IRWrite, ir.IRReturn, ir.IRAllocBytes, ir.IRRegionEnter, ir.IRRegionReset,
		ir.IRIslandNew, ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32,
		ir.IRIslandFree, ir.IRIslandReset, ir.IRCapIO, ir.IRCapMem, ir.IRMemReadI32, ir.IRMemWriteI32,
		ir.IRMemReadU8, ir.IRMemWriteU8, ir.IRMemReadPtr, ir.IRMemWritePtr,
		ir.IRMemWriteArchPtr, ir.IRMemReadI32Offset, ir.IRMemWriteI32Offset,
		ir.IRMemReadU8Offset, ir.IRMemWriteU8Offset, ir.IRMemReadPtrOffset,
		ir.IRMemWritePtrOffset, ir.IRMemWriteArchPtrOffset, ir.IRPtrAdd,
		ir.IRMmioReadI32, ir.IRMmioWriteI32, ir.IRSymAddr, ir.IRCtxSwitch,
		ir.IRAtomicLoadPtr, ir.IRAtomicStorePtr, ir.IRAtomicExchangePtr,
		ir.IRAtomicFetchAddPtr, ir.IRAtomicFetchSubPtr, ir.IRAtomicFetchAndPtr,
		ir.IRAtomicFetchOrPtr, ir.IRAtomicFetchXorPtr, ir.IRAtomicCompareExchangePtr,
		ir.IRAtomicFenceSeqCst, ir.IRAtomicFenceRelaxed, ir.IRAtomicFenceAcquire,
		ir.IRAtomicFenceRelease, ir.IRAtomicFenceAcqRel, ir.IRAtomicLoadI32,
		ir.IRAtomicStoreI32, ir.IRAtomicExchangeI32, ir.IRAtomicCompareExchangeI32,
		ir.IRAtomicFetchAddI32, ir.IRAtomicFetchSubI32, ir.IRAtomicFetchAndI32,
		ir.IRAtomicFetchOrI32, ir.IRAtomicFetchXorI32, ir.IRAtomicLoadI64,
		ir.IRAtomicStoreI64, ir.IRAtomicExchangeI64, ir.IRAtomicCompareExchangeI64,
		ir.IRAtomicFetchAddI64, ir.IRAtomicFetchSubI64, ir.IRAtomicFetchAndI64,
		ir.IRAtomicFetchOrI64, ir.IRAtomicFetchXorI64, ir.IRAtomicLoadI8,
		ir.IRAtomicStoreI8, ir.IRAtomicExchangeI8, ir.IRAtomicCompareExchangeI8,
		ir.IRAtomicFetchAddI8, ir.IRAtomicFetchSubI8, ir.IRAtomicFetchAndI8,
		ir.IRAtomicFetchOrI8, ir.IRAtomicFetchXorI8, ir.IRAtomicLoadI16,
		ir.IRAtomicStoreI16, ir.IRAtomicExchangeI16, ir.IRAtomicCompareExchangeI16,
		ir.IRAtomicFetchAddI16, ir.IRAtomicFetchSubI16, ir.IRAtomicFetchAndI16,
		ir.IRAtomicFetchOrI16, ir.IRAtomicFetchXorI16:
		clearKnownStack(stack)
	default:
		clearKnownStack(stack)
	}
}

func knownBranchOperandConst(instr ir.IRInstr, knownLocals map[int]int32) (int32, bool) {
	switch instr.Kind {
	case ir.IRConstI32:
		return instr.Imm, true
	case ir.IRLoadLocal:
		value, ok := knownLocals[instr.Local]
		return value, ok
	default:
		return 0, false
	}
}

type zeroComparisonFact struct {
	Local             int
	TargetZero        bool
	TargetReason      string
	FallthroughZero   bool
	FallthroughReason string
}

func zeroComparisonLocalFact(instrs []ir.IRInstr, cmpIndex int) (zeroComparisonFact, bool) {
	if cmpIndex < 2 {
		return zeroComparisonFact{}, false
	}
	load := instrs[cmpIndex-2]
	zero := instrs[cmpIndex-1]
	if load.Kind != ir.IRLoadLocal || zero.Kind != ir.IRConstI32 || zero.Imm != 0 {
		return zeroComparisonFact{}, false
	}
	switch instrs[cmpIndex].Kind {
	case ir.IRCmpEqI32:
		return zeroComparisonFact{
			Local:             load.Local,
			TargetZero:        false,
			TargetReason:      "eq_zero_false",
			FallthroughZero:   true,
			FallthroughReason: "eq_zero_true",
		}, true
	case ir.IRCmpNeI32:
		return zeroComparisonFact{
			Local:             load.Local,
			TargetZero:        true,
			TargetReason:      "ne_zero_false",
			FallthroughZero:   false,
			FallthroughReason: "ne_zero_true",
		}, true
	default:
		return zeroComparisonFact{}, false
	}
}

func previousKnownStackConst(out []ir.IRInstr, knownLocals map[int]int32) (int32, bool) {
	if len(out) == 0 {
		return 0, false
	}
	prev := out[len(out)-1]
	switch prev.Kind {
	case ir.IRConstI32:
		return prev.Imm, true
	case ir.IRLoadLocal:
		value, ok := knownLocals[prev.Local]
		return value, ok
	default:
		if len(out) >= 2 && isPureLocalUnaryOp(prev.Kind) {
			operand, operandOK := knownBranchOperandConst(out[len(out)-2], knownLocals)
			if operandOK {
				return foldConstUnaryI32(prev.Kind, operand)
			}
		}
		if len(out) >= 3 && isPureLocalBinaryOp(prev.Kind) {
			left, leftOK := knownBranchOperandConst(out[len(out)-3], knownLocals)
			right, rightOK := knownBranchOperandConst(out[len(out)-2], knownLocals)
			if leftOK && rightOK {
				return foldConstBinaryI32(prev.Kind, left, right)
			}
		}
		return 0, false
	}
}

func isPureLocalUnaryOp(kind ir.IRInstrKind) bool {
	return kind == ir.IRNegI32
}

func foldConstUnaryI32(kind ir.IRInstrKind, value int32) (int32, bool) {
	switch kind {
	case ir.IRNegI32:
		return checkedNegI32(value)
	default:
		return 0, false
	}
}

func countLabelIncoming(instrs []ir.IRInstr) map[int]int {
	incoming := map[int]int{}
	for _, instr := range instrs {
		switch instr.Kind {
		case ir.IRJmp, ir.IRJmpIfZero:
			incoming[instr.Label]++
		}
	}
	return incoming
}

func indexLabels(instrs []ir.IRInstr) map[int]int {
	indexes := map[int]int{}
	for i, instr := range instrs {
		if instr.Kind == ir.IRLabel {
			indexes[instr.Label] = i
		}
	}
	return indexes
}

func knownLocalSinglePredecessorLabelReason(
	instrs []ir.IRInstr,
	labelIndexes map[int]int,
	labelIncoming map[int]int,
	jumpIndex int,
	label int,
) (string, bool) {
	targetIndex, ok := labelIndexes[label]
	if !ok || targetIndex <= jumpIndex || labelIncoming[label] != 1 {
		return "", false
	}
	if !labelHasNoFallthroughPredecessor(instrs, targetIndex) {
		return "", false
	}
	if targetIndex == jumpIndex+1 {
		return "single_predecessor_label", true
	}
	return "forward_single_predecessor_jump", true
}

func (s *sccpState) propagateKnownLocalsThroughFoldedZeroBranch(
	fnName string,
	instrs []ir.IRInstr,
	labelIndexes map[int]int,
	labelIncoming map[int]int,
	knownLocals map[int]int32,
	branchIndex int,
	label int,
	pendingLabelFacts map[int]map[int]int32,
) {
	if len(knownLocals) == 0 {
		return
	}
	reason, ok := knownLocalSinglePredecessorLabelReason(
		instrs,
		labelIndexes,
		labelIncoming,
		branchIndex,
		label,
	)
	if !ok {
		return
	}
	pendingLabelFacts[label] = cloneKnownLocalConsts(knownLocals)
	switch reason {
	case "single_predecessor_label":
		reason = "folded_zero_branch_single_predecessor_label"
	case "forward_single_predecessor_jump":
		reason = "folded_zero_branch_forward_single_predecessor_jump"
	}
	s.decisions = append(
		s.decisions,
		PassDecision{
			Action: "propagated_known_local_folded_zero_branch",
			Caller: fnName,
			Site:   branchIndex,
			Reason: reason,
		},
	)
}

func (s *sccpState) propagateKnownLocalsThroughFoldedNonzeroFallthrough(
	fnName string,
	instrs []ir.IRInstr,
	labelIncoming map[int]int,
	knownLocals map[int]int32,
	branchIndex int,
	fallthroughIndex int,
	pendingLabelFacts map[int]map[int]int32,
) {
	if len(knownLocals) == 0 || fallthroughIndex >= len(instrs) {
		return
	}
	fallthroughInstr := instrs[fallthroughIndex]
	if fallthroughInstr.Kind != ir.IRLabel || labelIncoming[fallthroughInstr.Label] != 0 {
		return
	}
	pendingLabelFacts[fallthroughInstr.Label] = cloneKnownLocalConsts(knownLocals)
	s.decisions = append(
		s.decisions,
		PassDecision{
			Action: "propagated_known_local_folded_nonzero_fallthrough",
			Caller: fnName,
			Site:   branchIndex,
			Reason: "folded_nonzero_fallthrough_label",
		},
	)
}

func (s *sccpState) propagatePathLocalZeroThroughDynamicBranchTarget(
	fnName string,
	instrs []ir.IRInstr,
	labelIndexes map[int]int,
	labelIncoming map[int]int,
	local int,
	branchIndex int,
	label int,
	pendingLabelZeroFacts map[int]map[int]bool,
) {
	reason, ok := knownLocalSinglePredecessorLabelReason(
		instrs,
		labelIndexes,
		labelIncoming,
		branchIndex,
		label,
	)
	if !ok {
		return
	}
	addPendingLocalBoolFact(pendingLabelZeroFacts, label, local)
	switch reason {
	case "single_predecessor_label":
		reason = "dynamic_zero_single_predecessor_label"
	case "forward_single_predecessor_jump":
		reason = "dynamic_zero_forward_single_predecessor_jump"
	}
	s.decisions = append(
		s.decisions,
		PassDecision{
			Action: "propagated_path_local_zero_target",
			Caller: fnName,
			Site:   branchIndex,
			Reason: reason,
		},
	)
}

func (s *sccpState) propagatePathLocalZeroThroughFoldedZeroBranch(
	fnName string,
	instrs []ir.IRInstr,
	labelIndexes map[int]int,
	labelIncoming map[int]int,
	local int,
	branchIndex int,
	label int,
	pendingLabelZeroFacts map[int]map[int]bool,
) {
	reason, ok := knownLocalSinglePredecessorLabelReason(
		instrs,
		labelIndexes,
		labelIncoming,
		branchIndex,
		label,
	)
	if !ok {
		return
	}
	addPendingLocalBoolFact(pendingLabelZeroFacts, label, local)
	switch reason {
	case "single_predecessor_label":
		reason = "path_zero_single_predecessor_label"
	case "forward_single_predecessor_jump":
		reason = "path_zero_forward_single_predecessor_jump"
	}
	s.decisions = append(
		s.decisions,
		PassDecision{
			Action: "propagated_path_local_zero_target",
			Caller: fnName,
			Site:   branchIndex,
			Reason: reason,
		},
	)
}

func (s *sccpState) propagatePathLocalNonZeroThroughFoldedNonzeroFallthrough(
	fnName string,
	instrs []ir.IRInstr,
	labelIncoming map[int]int,
	local int,
	branchIndex int,
	fallthroughIndex int,
	pendingLabelNonZeroFacts map[int]map[int]bool,
) {
	if fallthroughIndex >= len(instrs) {
		return
	}
	fallthroughInstr := instrs[fallthroughIndex]
	if fallthroughInstr.Kind != ir.IRLabel || labelIncoming[fallthroughInstr.Label] != 0 {
		return
	}
	addPendingLocalBoolFact(pendingLabelNonZeroFacts, fallthroughInstr.Label, local)
	s.decisions = append(
		s.decisions,
		PassDecision{
			Action: "propagated_path_local_nonzero_fallthrough",
			Caller: fnName,
			Site:   branchIndex,
			Reason: "path_nonzero_fallthrough_label",
		},
	)
}

func (s *sccpState) propagateComparisonPathLocalThroughDynamicBranchTarget(
	fnName string,
	instrs []ir.IRInstr,
	labelIndexes map[int]int,
	labelIncoming map[int]int,
	fact zeroComparisonFact,
	branchIndex int,
	label int,
	pendingLabelZeroFacts map[int]map[int]bool,
	pendingLabelNonZeroFacts map[int]map[int]bool,
) {
	reason, ok := knownLocalSinglePredecessorLabelReason(
		instrs,
		labelIndexes,
		labelIncoming,
		branchIndex,
		label,
	)
	if !ok {
		return
	}
	switch reason {
	case "single_predecessor_label":
		reason = fact.TargetReason + "_single_predecessor_label"
	case "forward_single_predecessor_jump":
		reason = fact.TargetReason + "_forward_single_predecessor_jump"
	default:
		reason = fact.TargetReason
	}
	if fact.TargetZero {
		addPendingLocalBoolFact(pendingLabelZeroFacts, label, fact.Local)
		s.decisions = append(
			s.decisions,
			PassDecision{
				Action: "propagated_comparison_path_local_zero_target",
				Caller: fnName,
				Site:   branchIndex,
				Reason: reason,
			},
		)
		return
	}
	addPendingLocalBoolFact(pendingLabelNonZeroFacts, label, fact.Local)
	s.decisions = append(
		s.decisions,
		PassDecision{
			Action: "propagated_comparison_path_local_nonzero_target",
			Caller: fnName,
			Site:   branchIndex,
			Reason: reason,
		},
	)
}

func labelHasNoFallthroughPredecessor(instrs []ir.IRInstr, labelIndex int) bool {
	if labelIndex == 0 {
		return true
	}
	switch instrs[labelIndex-1].Kind {
	case ir.IRJmp, ir.IRReturn:
		return true
	default:
		return false
	}
}

func cloneKnownLocalConsts(knownLocals map[int]int32) map[int]int32 {
	out := make(map[int]int32, len(knownLocals))
	for local, value := range knownLocals {
		out[local] = value
	}
	return out
}

func addPendingLocalBoolFact(pending map[int]map[int]bool, label int, local int) {
	facts, ok := pending[label]
	if !ok {
		facts = map[int]bool{}
		pending[label] = facts
	}
	facts[local] = true
}

func applyKnownLocalConsts(
	knownLocals map[int]int32,
	knownZeroLocals map[int]bool,
	knownNonZeroLocals map[int]bool,
	facts map[int]int32,
) {
	for local, value := range facts {
		setKnownLocalConst(knownLocals, knownZeroLocals, knownNonZeroLocals, local, value)
	}
}

func applyKnownLocalZeros(
	knownLocals map[int]int32,
	knownZeroLocals map[int]bool,
	knownNonZeroLocals map[int]bool,
	facts map[int]bool,
) {
	for local := range facts {
		setKnownLocalZero(knownLocals, knownZeroLocals, knownNonZeroLocals, local)
	}
}

func applyKnownLocalNonZeros(
	knownLocals map[int]int32,
	knownZeroLocals map[int]bool,
	knownNonZeroLocals map[int]bool,
	facts map[int]bool,
) {
	for local := range facts {
		setKnownLocalNonZero(knownLocals, knownZeroLocals, knownNonZeroLocals, local)
	}
}

func setKnownLocalConst(
	knownLocals map[int]int32,
	knownZeroLocals map[int]bool,
	knownNonZeroLocals map[int]bool,
	local int,
	value int32,
) {
	knownLocals[local] = value
	delete(knownZeroLocals, local)
	delete(knownNonZeroLocals, local)
	if value == 0 {
		knownZeroLocals[local] = true
	} else {
		knownNonZeroLocals[local] = true
	}
}

func setKnownLocalZero(
	knownLocals map[int]int32,
	knownZeroLocals map[int]bool,
	knownNonZeroLocals map[int]bool,
	local int,
) {
	delete(knownLocals, local)
	knownZeroLocals[local] = true
	delete(knownNonZeroLocals, local)
}

func setKnownLocalNonZero(
	knownLocals map[int]int32,
	knownZeroLocals map[int]bool,
	knownNonZeroLocals map[int]bool,
	local int,
) {
	delete(knownLocals, local)
	delete(knownZeroLocals, local)
	knownNonZeroLocals[local] = true
}

func deleteKnownLocalFact(
	knownLocals map[int]int32,
	knownZeroLocals map[int]bool,
	knownNonZeroLocals map[int]bool,
	local int,
) {
	delete(knownLocals, local)
	delete(knownZeroLocals, local)
	delete(knownNonZeroLocals, local)
}

func clearKnownLocalConsts(knownLocals map[int]int32) {
	for local := range knownLocals {
		delete(knownLocals, local)
	}
}

func clearKnownLocalFacts(
	knownLocals map[int]int32,
	knownZeroLocals map[int]bool,
	knownNonZeroLocals map[int]bool,
) {
	for local := range knownLocals {
		delete(knownLocals, local)
	}
	for local := range knownZeroLocals {
		delete(knownZeroLocals, local)
	}
	for local := range knownNonZeroLocals {
		delete(knownNonZeroLocals, local)
	}
}

func skipFallthroughUntilLabel(instrs []ir.IRInstr, start int) (next int, pruned int) {
	for next = start; next < len(instrs); next++ {
		if instrs[next].Kind == ir.IRLabel {
			break
		}
		pruned++
	}
	return next, pruned
}

// ---- specialization_machine_code.go ----

type SpecializationMachineCodeID string

const (
	SpecializationMachineCodeGenerics    SpecializationMachineCodeID = "generics"
	SpecializationMachineCodeOptionals   SpecializationMachineCodeID = "optionals"
	SpecializationMachineCodeCollections SpecializationMachineCodeID = "collections"
)

const (
	SpecializationMachineCodeProtocolStaticConformance = SpecializationMachineCodeID(
		"protocol_static_conformance",
	)
	SpecializationMachineCodeExtensionMethods = SpecializationMachineCodeID(
		"extension_methods",
	)
	SpecializationMachineCodeEnumKnownCases = SpecializationMachineCodeID(
		"enum_match_known_cases",
	)
)

type SpecializationMachineCodeStatus string

const (
	SpecializationMachineCodeImplementedNarrow SpecializationMachineCodeStatus = "implemented_narrow"
)

type SpecializationMachineCodeCoverageReport struct {
	SchemaVersion                     string          `json:"schema_version"`
	Scope                             string          `json:"scope"`
	Rows                              smCodeRows      `json:"rows"`
	Witnesses                         smCodeWitnesses `json:"witnesses,omitempty"`
	NonClaims                         []string        `json:"non_claims"`
	BroadSpecializationClaimed        bool            `json:"broad_specialization_claimed"`
	DynamicDispatchClaimed            bool            `json:"dynamic_dispatch_claimed"`
	RuntimeGenericValuesClaimed       bool            `json:"runtime_generic_values_claimed"`
	AllocatorBackedCollectionsClaimed bool            `json:"allocator_backed_collections_claimed"`
	LayoutABIFreedomClaimed           bool            `json:"layout_abi_freedom_claimed"`
	PerformanceClaimed                bool            `json:"performance_claimed"`
	SafeSemanticsChanged              bool            `json:"safe_semantics_changed"`
}

type SpecializationMachineCodeRow struct {
	ID                      SpecializationMachineCodeID     `json:"id"`
	Name                    string                          `json:"name"`
	Status                  SpecializationMachineCodeStatus `json:"status"`
	Passes                  []string                        `json:"passes"`
	SourceEvidence          string                          `json:"source_evidence"`
	OptimizedIREvidence     string                          `json:"optimized_ir_evidence"`
	MachineCodeEvidence     string                          `json:"machine_code_evidence"`
	MachineWitnessID        string                          `json:"machine_witness_id"`
	RemovedHighLevelMarkers []string                        `json:"removed_high_level_markers"`
	Boundary                string                          `json:"boundary"`
}

type SpecializationMachineWitness struct {
	ID                   string   `json:"id"`
	TranslationValidated bool     `json:"translation_validated"`
	StackIRHadCallBefore bool     `json:"stack_ir_had_call_before"`
	StackIRHasCallAfter  bool     `json:"stack_ir_has_call_after"`
	MachineIRVerified    bool     `json:"machine_ir_verified"`
	MachineIRHasCall     bool     `json:"machine_ir_has_call"`
	MachineTarget        string   `json:"machine_target"`
	MachineOps           []string `json:"machine_ops"`
	InlineDecisions      []string `json:"inline_decisions"`
	RemovedMarkers       []string `json:"removed_markers"`
	BeforeStackIRDump    string   `json:"before_stack_ir_dump,omitempty"`
	AfterStackIRDump     string   `json:"after_stack_ir_dump,omitempty"`
	MachineIRDump        string   `json:"machine_ir_dump,omitempty"`
}

const (
	specializationMachineCodeSchema = "tetra.optimizer.specialization_machine_code.v1"
	specializationMachineCodeScope  = "p21.2_specialization_v1_v2"
	p21MachineWitnessID             = "p21.2_known_direct_call_scalar_machine_witness"
)

func SpecializationMachineCodeCoverage() (SpecializationMachineCodeCoverageReport, error) {
	witness, err := BuildP21SpecializationMachineCodeWitness()
	if err != nil {
		return SpecializationMachineCodeCoverageReport{}, err
	}
	report := SpecializationMachineCodeCoverageReport{
		SchemaVersion: specializationMachineCodeSchema,
		Scope:         specializationMachineCodeScope,
		Rows: []SpecializationMachineCodeRow{
			specializationMachineGenericsRow(witness),
			specializationMachineProtocolRow(witness),
			specializationMachineExtensionRow(witness),
			specializationMachineEnumRow(witness),
			specializationMachineOptionalRow(witness),
			specializationMachineCollectionsRow(witness),
		},
		Witnesses: []SpecializationMachineWitness{witness},
		NonClaims: []string{
			"broad specialization is not claimed",
			"performance is not claimed",
			"safe-program semantics do not change",
			"dynamic protocol dispatch is not claimed",
			"runtime generic values are not claimed",
			"allocator-backed production generic collections are not claimed",
			"layout/ABI freedom is not claimed",
		},
	}
	return report, nil
}

func ValidateSpecializationMachineCodeCoverage(
	report SpecializationMachineCodeCoverageReport,
) error {
	if report.SchemaVersion != specializationMachineCodeSchema {
		return fmt.Errorf(
			"specialization machine-code coverage: schema = %q, want %q",
			report.SchemaVersion,
			specializationMachineCodeSchema,
		)
	}
	if report.Scope != specializationMachineCodeScope {
		return fmt.Errorf(
			"specialization machine-code coverage: scope = %q, want %q",
			report.Scope,
			specializationMachineCodeScope,
		)
	}
	if report.BroadSpecializationClaimed {
		return fmt.Errorf(
			"specialization machine-code coverage: broad specialization claim is forbidden",
		)
	}
	if report.DynamicDispatchClaimed {
		return fmt.Errorf(
			"specialization machine-code coverage: dynamic dispatch claim is forbidden",
		)
	}
	if report.RuntimeGenericValuesClaimed {
		return fmt.Errorf(
			"specialization machine-code coverage: runtime generic value claim is forbidden",
		)
	}
	if report.AllocatorBackedCollectionsClaimed {
		return fmt.Errorf(
			"specialization machine-code coverage: allocator-backed generic collection claim is forbidden",
		)
	}
	if report.LayoutABIFreedomClaimed {
		return fmt.Errorf(
			"specialization machine-code coverage: layout/ABI freedom claim is forbidden",
		)
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("specialization machine-code coverage: performance claim is forbidden")
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf(
			"specialization machine-code coverage: safe-program semantics change is forbidden",
		)
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
		if !containsSpecializationMachineText(report.NonClaims, want) {
			return fmt.Errorf("specialization machine-code coverage: missing non-claim %q", want)
		}
	}
	if len(report.Witnesses) == 0 {
		return fmt.Errorf("specialization machine-code coverage: missing machine witness")
	}
	witnesses := map[string]SpecializationMachineWitness{}
	for _, witness := range report.Witnesses {
		if strings.TrimSpace(witness.ID) == "" {
			return fmt.Errorf("specialization machine-code coverage: witness missing id")
		}
		if witnesses[witness.ID].ID != "" {
			return fmt.Errorf(
				"specialization machine-code coverage: duplicate witness %q",
				witness.ID,
			)
		}
		if !witness.TranslationValidated || !witness.StackIRHadCallBefore ||
			witness.StackIRHasCallAfter ||
			!witness.MachineIRVerified ||
			witness.MachineIRHasCall ||
			strings.TrimSpace(witness.MachineTarget) == "" ||
			len(witness.MachineOps) == 0 {
			return fmt.Errorf(
				("specialization machine-code coverage: witness %q does not " +
					"prove call disappearance before Machine IR"),
				witness.ID,
			)
		}
		witnesses[witness.ID] = witness
	}

	expected := map[SpecializationMachineCodeID]bool{
		SpecializationMachineCodeGenerics:                  true,
		SpecializationMachineCodeProtocolStaticConformance: true,
		SpecializationMachineCodeExtensionMethods:          true,
		SpecializationMachineCodeEnumKnownCases:            true,
		SpecializationMachineCodeOptionals:                 true,
		SpecializationMachineCodeCollections:               true,
	}
	if len(report.Rows) != len(expected) {
		return fmt.Errorf(
			"specialization machine-code coverage: row count = %d, want %d",
			len(report.Rows),
			len(expected),
		)
	}
	seen := map[SpecializationMachineCodeID]bool{}
	for _, row := range report.Rows {
		if !expected[row.ID] {
			return fmt.Errorf("specialization machine-code coverage: unexpected row %q", row.ID)
		}
		if seen[row.ID] {
			return fmt.Errorf("specialization machine-code coverage: duplicate row %q", row.ID)
		}
		seen[row.ID] = true
		if row.Status != SpecializationMachineCodeImplementedNarrow {
			return fmt.Errorf(
				"specialization machine-code coverage: row %q status = %q",
				row.ID,
				row.Status,
			)
		}
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.SourceEvidence) == "" ||
			strings.TrimSpace(row.OptimizedIREvidence) == "" ||
			strings.TrimSpace(row.MachineCodeEvidence) == "" ||
			strings.TrimSpace(row.Boundary) == "" {
			return fmt.Errorf(
				("specialization machine-code coverage: row %q missing " +
					"source/optimized/machine evidence or boundary"),
				row.ID,
			)
		}
		if len(row.Passes) == 0 {
			return fmt.Errorf(
				"specialization machine-code coverage: row %q missing pass owner",
				row.ID,
			)
		}
		if len(row.RemovedHighLevelMarkers) == 0 {
			return fmt.Errorf(
				"specialization machine-code coverage: row %q missing removed high-level markers",
				row.ID,
			)
		}
		if strings.TrimSpace(row.MachineWitnessID) == "" {
			return fmt.Errorf(
				"specialization machine-code coverage: row %q missing machine witness id",
				row.ID,
			)
		}
		if _, ok := witnesses[row.MachineWitnessID]; !ok {
			return fmt.Errorf(
				"specialization machine-code coverage: row %q references missing witness %q",
				row.ID,
				row.MachineWitnessID,
			)
		}
		if containsSpecializationPlaceholder(
			row.Name,
			row.SourceEvidence,
			row.OptimizedIREvidence,
			row.MachineCodeEvidence,
			row.Boundary,
			strings.Join(row.RemovedHighLevelMarkers, " "),
		) {
			return fmt.Errorf(
				"specialization machine-code coverage: row %q contains placeholder evidence",
				row.ID,
			)
		}
		if !strings.Contains(row.MachineCodeEvidence, "Machine IR") &&
			!strings.Contains(row.MachineCodeEvidence, "machine code") {
			return fmt.Errorf(
				"specialization machine-code coverage: row %q missing machine evidence",
				row.ID,
			)
		}
	}
	for id := range expected {
		if !seen[id] {
			return fmt.Errorf("specialization machine-code coverage: missing row %q", id)
		}
	}
	return nil
}

func BuildP21SpecializationMachineCodeWitness() (SpecializationMachineWitness, error) {
	prog := p21KnownDirectCallProgram()
	beforeDump := FormatProgram(prog)
	stackIRHadCallBefore := programHasIRCall(prog, "known_i32_add")
	report, err := NewManager().Run(prog, InlineSmallPurePass())
	if err != nil {
		return SpecializationMachineWitness{}, fmt.Errorf(
			"p21.2 machine witness inline pass: %w",
			err,
		)
	}
	if len(report.Passes) != 1 {
		return SpecializationMachineWitness{}, fmt.Errorf(
			"p21.2 machine witness: pass count = %d, want 1",
			len(report.Passes),
		)
	}
	pass := report.Passes[0]
	afterDump := FormatProgram(prog)
	stackIRHasCallAfter := programHasIRCall(prog, "known_i32_add")
	mainFn, ok := findIRFuncByName(prog, "main")
	if !ok {
		return SpecializationMachineWitness{}, fmt.Errorf(
			"p21.2 machine witness: missing optimized main function",
		)
	}
	mfn, supported, err := machine.ScalarIntFunctionFromStackIR(mainFn)
	if err != nil {
		return SpecializationMachineWitness{}, fmt.Errorf(
			"p21.2 machine witness scalar lowering: %w",
			err,
		)
	}
	if !supported {
		return SpecializationMachineWitness{}, fmt.Errorf(
			"p21.2 machine witness: optimized main is not supported by scalar machine lowering",
		)
	}
	machineIRVerified := machine.VerifyFunction(mfn) == nil
	machineIRHasCall := machineFunctionHasCall(mfn)
	machineOps := machineFunctionOps(mfn)
	machineDump := machine.FormatFunction(mfn)
	return SpecializationMachineWitness{
		ID:                   p21MachineWitnessID,
		TranslationValidated: pass.TranslationValidated,
		StackIRHadCallBefore: stackIRHadCallBefore,
		StackIRHasCallAfter:  stackIRHasCallAfter,
		MachineIRVerified:    machineIRVerified,
		MachineIRHasCall:     machineIRHasCall,
		MachineTarget:        mfn.Target,
		MachineOps:           machineOps,
		InlineDecisions:      formatInlineDecisions(pass.Decisions),
		RemovedMarkers:       []string{"IRCall known_i32_add", "OpCall"},
		BeforeStackIRDump:    beforeDump,
		AfterStackIRDump:     afterDump,
		MachineIRDump:        machineDump,
	}, nil
}

func specializationMachineGenericsRow(
	witness SpecializationMachineWitness,
) SpecializationMachineCodeRow {
	return SpecializationMachineCodeRow{
		ID:     SpecializationMachineCodeGenerics,
		Name:   "generics",
		Status: SpecializationMachineCodeImplementedNarrow,
		Passes: []string{"inline-small-pure"},
		SourceEvidence: ("compiler/tests/semantics/semantics_types_protocols_test.go::" +
			"TestP9GenericIdentityDisappearsAfterSmallPureInlining; " +
			"compiler/tests/semantics/semantics_types_protocols_test.go::" +
			"TestP17GenericWrapperDisappearsAfterSmallPureInlining"),
		OptimizedIREvidence: ("monomorphized generic identity and generic wrapper calls " +
			"are direct concrete Stack IR calls; optimized Stack IR has " +
			"no call after inline-small-pure when the tiny concrete " +
			"helper is accepted"),
		MachineCodeEvidence: machineEvidenceText(
			witness,
			("Machine IR contains no OpCall for the P21.2 known " +
				"direct-call scalar witness after the optimized Stack IR " +
				"call disappears"),
		),
		MachineWitnessID: witness.ID,
		RemovedHighLevelMarkers: []string{
			"monomorphized generic identity call",
			"generic wrapper call",
			"IRCall known_i32_add",
			"OpCall",
		},
		Boundary: ("only statically monomorphized generic identity/wrapper " +
			"helpers that lower to bounded direct Stack IR calls may " +
			"disappear; no runtime generic values, explicit type " +
			"arguments, generic structs, dynamic dispatch, broad " +
			"specialization, public optimizer mode, or performance claim"),
	}
}

func specializationMachineProtocolRow(
	witness SpecializationMachineWitness,
) SpecializationMachineCodeRow {
	return SpecializationMachineCodeRow{
		ID:     SpecializationMachineCodeProtocolStaticConformance,
		Name:   "protocol/static conformance",
		Status: SpecializationMachineCodeImplementedNarrow,
		Passes: []string{"inline-small-pure"},
		SourceEvidence: ("compiler/tests/semantics/semantics_callables_closures_test.g" +
			"o::" +
			"TestP17StaticProtocolConformanceCallInlinesAfterSmallPure; " +
			"compiler/internal/layoutopt/layoutopt_test.go::" +
			"TestSpecializationDevirtualizesProtocolOnlyWhenTargetKnown"),
		OptimizedIREvidence: ("statically checked protocol impl method calls lower to a " +
			"known direct Stack IR function symbol and optimized Stack " +
			"IR has no call when inline-small-pure accepts the concrete " +
			"method body"),
		MachineCodeEvidence: machineEvidenceText(
			witness,
			("Machine IR contains no OpCall for the bounded direct-call " +
				"witness; static protocol/conformance evidence is tied to " +
				"known direct symbols, not runtime dispatch"),
		),
		MachineWitnessID: witness.ID,
		RemovedHighLevelMarkers: []string{
			"statically checked protocol impl direct call",
			"known direct Stack IR function symbol call",
			"IRCall known_i32_add",
			"OpCall",
		},
		Boundary: ("statically checked protocol impl calls may disappear only " +
			"after lowering to a known direct Stack IR function symbol; " +
			"no witness tables, trait objects, runtime protocol values, " +
			"dynamic dispatch, conformance-table lookup, protocol-bound " +
			"generic requirement call, broad protocol specialization, or " +
			"performance claim"),
	}
}

func specializationMachineExtensionRow(
	witness SpecializationMachineWitness,
) SpecializationMachineCodeRow {
	return SpecializationMachineCodeRow{
		ID:     SpecializationMachineCodeExtensionMethods,
		Name:   "extension methods",
		Status: SpecializationMachineCodeImplementedNarrow,
		Passes: []string{"inline-small-pure"},
		SourceEvidence: ("compiler/tests/semantics/semantics_callables_closures_test.g" +
			"o::TestP17StaticExtensionCallInlinesAfterSmallPure; " +
			"compiler/tests/semantics/semantics_core_language_test.go::" +
			"TestExtensionParseCheckAndLower"),
		OptimizedIREvidence: ("statically resolved extension method calls lower to a " +
			"direct Stack IR function symbol and optimized Stack IR has " +
			"no call when inline-small-pure accepts the body"),
		MachineCodeEvidence: machineEvidenceText(
			witness,
			("Machine IR contains no OpCall for the bounded direct-call " +
				"witness after the extension-like direct helper disappears"),
		),
		MachineWitnessID: witness.ID,
		RemovedHighLevelMarkers: []string{
			"statically resolved extension method direct call",
			"direct Stack IR function symbol call",
			"IRCall known_i32_add",
			"OpCall",
		},
		Boundary: ("only statically resolved extension method calls that lower " +
			"to direct Stack IR function symbols may disappear; no " +
			"dynamic extension dispatch, receiver-call sugar " +
			"specialization, protocol/witness dispatch, effectful or " +
			"oversized method inlining, cross-control-flow " +
			"specialization, or performance claim"),
	}
}

func specializationMachineEnumRow(
	witness SpecializationMachineWitness,
) SpecializationMachineCodeRow {
	return SpecializationMachineCodeRow{
		ID:     SpecializationMachineCodeEnumKnownCases,
		Name:   "enum match known cases",
		Status: SpecializationMachineCodeImplementedNarrow,
		Passes: []string{"sccp-constant-branch", "inline-small-pure"},
		SourceEvidence: ("compiler/tests/semantics/semantics_callables_closures_test.g" +
			"o::TestP17KnownEnumPayloadMatchFoldsAfterSCCP; " +
			"compiler/internal/lower/lower_suite_test.go::" +
			"TestLowerMatchExpressionEnumPayloadIR"),
		OptimizedIREvidence: ("payload enum known-case match uses constant_stack_store tag " +
			"tracking and sccp-constant-branch folded discriminator " +
			"branch evidence"),
		MachineCodeEvidence: machineEvidenceText(
			witness,
			("machine code carries no match dispatch for the accepted " +
				"scalar direct-call witness; known enum branch dispatch is " +
				"removed in optimized Stack IR before machine lowering"),
		),
		MachineWitnessID: witness.ID,
		RemovedHighLevelMarkers: []string{
			"known-case match discriminator branch",
			"folded discriminator branch",
			"enum match dispatch",
		},
		Boundary: ("only locally constructed payload enum tags tracked through " +
			"same-basic-block Stack IR stores are folded; no broad enum " +
			"specialization, payload escape rewrite, cross-control-flow " +
			"enum fact propagation, exhaustive match pruning, runtime " +
			"behavior change, or performance claim"),
	}
}

func specializationMachineOptionalRow(
	witness SpecializationMachineWitness,
) SpecializationMachineCodeRow {
	return SpecializationMachineCodeRow{
		ID:     SpecializationMachineCodeOptionals,
		Name:   "optionals",
		Status: SpecializationMachineCodeImplementedNarrow,
		Passes: []string{"sccp-constant-branch", "inline-small-pure"},
		SourceEvidence: ("compiler/tests/semantics/semantics_callables_closures_test.g" +
			"o::TestP17ProvenSomeOptionalMatchFoldsAfterSCCP; " +
			"compiler/tests/semantics/semantics_types_protocols_test.go::" +
			"TestOptionalMatchExhaustiveNoDefaultWithMultiSlotPayload"),
		OptimizedIREvidence: ("proven-some optional presence tags use constant_stack_store " +
			"tracking and sccp-constant-branch folded presence branch " +
			"evidence"),
		MachineCodeEvidence: machineEvidenceText(
			witness,
			("machine code carries no optional dispatch for the accepted " +
				"scalar direct-call witness; proven-some optional branch " +
				"dispatch is removed in optimized Stack IR before machine " +
				"lowering"),
		),
		MachineWitnessID: witness.ID,
		RemovedHighLevelMarkers: []string{
			"proven-some optional presence branch",
			"folded presence branch",
			"optional dispatch",
		},
		Boundary: ("only locally constructed proven-some optionals with " +
			"same-basic-block presence tag evidence are folded; no broad " +
			"optional elimination, unsafe unwrap removal, " +
			"cross-control-flow optional facts, none-branch pruning, " +
			"runtime behavior change, or performance claim"),
	}
}

func specializationMachineCollectionsRow(
	witness SpecializationMachineWitness,
) SpecializationMachineCodeRow {
	return SpecializationMachineCodeRow{
		ID:     SpecializationMachineCodeCollections,
		Name:   "collections",
		Status: SpecializationMachineCodeImplementedNarrow,
		Passes: []string{"static monomorphization", "inline-small-pure"},
		SourceEvidence: ("compiler/internal/stdlibrt/stable_generic_collections.go::" +
			"StableGenericCollectionsCoverage; " +
			"compiler/tests/semantics/semantics_types_protocols_test.go::" +
			"TestStableGenericCollectionSourceAPIMonomorphizesVecAndHashM" +
			"ap; lib/core/data/collections.tetra::Vec<T>; " +
			"lib/core/data/collections.tetra::HashMap<K,V>"),
		OptimizedIREvidence: ("Vec<T> and HashMap<K,V> source helpers are caller-owned and " +
			"monomorphized before lowering; a monomorphized collection " +
			"helper that becomes a bounded direct Stack IR helper may " +
			"disappear from optimized Stack IR"),
		MachineCodeEvidence: machineEvidenceText(
			witness,
			("Machine IR contains no OpCall for the bounded monomorphized " +
				"collection helper witness after the direct helper disappears"),
		),
		MachineWitnessID: witness.ID,
		RemovedHighLevelMarkers: []string{
			"Vec<T> source helper call",
			"HashMap<K,V> source helper call",
			"monomorphized collection helper direct call",
			"IRCall known_i32_add",
			"OpCall",
		},
		Boundary: ("collection evidence is limited to caller-owned source views " +
			"and monomorphized collection helper calls that are already " +
			"concrete and bounded; no allocator-backed production " +
			"Vec<T>/HashMap<K,V> runtime, generic hashing/equality " +
			"protocol, resizing policy, hidden runtime allocator, broad " +
			"production stdlib, C++/Rust parity, or performance claim"),
	}
}

func machineEvidenceText(witness SpecializationMachineWitness, prefix string) string {
	return fmt.Sprintf(
		"%s; witness=%s target=%s verified=%t machine_call=%t ops=%s",
		prefix,
		witness.ID,
		witness.MachineTarget,
		witness.MachineIRVerified,
		witness.MachineIRHasCall,
		strings.Join(witness.MachineOps, ","),
	)
}

func p21KnownDirectCallProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{
			{
				Name:        "main",
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 40},
					{Kind: ir.IRConstI32, Imm: 2},
					{Kind: ir.IRCall, Name: "known_i32_add", ArgSlots: 2, RetSlots: 1},
					{Kind: ir.IRReturn},
				},
			},
			{
				Name:        "known_i32_add",
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

func findIRFuncByName(prog *ir.IRProgram, name string) (ir.IRFunc, bool) {
	if prog == nil {
		return ir.IRFunc{}, false
	}
	for _, fn := range prog.Funcs {
		if fn.Name == name {
			return fn, true
		}
	}
	return ir.IRFunc{}, false
}

func programHasIRCall(prog *ir.IRProgram, name string) bool {
	if prog == nil {
		return false
	}
	for _, fn := range prog.Funcs {
		for _, instr := range fn.Instrs {
			if instr.Kind == ir.IRCall && instr.Name == name {
				return true
			}
		}
	}
	return false
}

func machineFunctionHasCall(fn machine.Function) bool {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if instr.Op == machine.OpCall {
				return true
			}
		}
	}
	return false
}

func machineFunctionOps(fn machine.Function) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			op := string(instr.Op)
			if !seen[op] {
				seen[op] = true
				out = append(out, op)
			}
		}
	}
	return out
}

func formatInlineDecisions(decisions []PassDecision) []string {
	out := make([]string, 0, len(decisions))
	for _, decision := range decisions {
		out = append(
			out,
			fmt.Sprintf(
				"%s->%s:%s:%s",
				decision.Caller,
				decision.Callee,
				decision.Action,
				decision.Reason,
			),
		)
	}
	return out
}

func containsSpecializationMachineText(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}

func containsSpecializationPlaceholder(items ...string) bool {
	for _, item := range items {
		switch strings.ToLower(strings.TrimSpace(item)) {
		case "", "todo", "tbd", "placeholder":
			return true
		}
		lower := strings.ToLower(item)
		if strings.Contains(lower, "todo") || strings.Contains(lower, "placeholder") {
			return true
		}
	}
	return false
}

// ---- vectorization.go ----

type VectorizationID string

const (
	VectorizationSumI32       VectorizationID = "sum_i32"
	VectorizationCopyU8       VectorizationID = "copy_u8"
	VectorizationMemsetMemcpy VectorizationID = "memset_memcpy"
	VectorizationMapI32       VectorizationID = "map_i32"
)

type VectorizationStatus string

const (
	VectorizationCandidateGuarded       VectorizationStatus = "candidate_guarded"
	VectorizationBackendLoweringGuarded VectorizationStatus = "backend_lowering_guarded"
	VectorizationImplementedNarrow      VectorizationStatus = "implemented_narrow"
	VectorizationNotYetCovered          VectorizationStatus = "not_yet_covered"
)

type VectorizationDecision string

const (
	VectorizationVectorized    VectorizationDecision = "vectorized"
	VectorizationNotVectorized VectorizationDecision = "not_vectorized"
)

type VectorizationCoverageReport struct {
	SchemaVersion string                     `json:"schema_version"`
	Rows          []VectorizationCoverageRow `json:"rows"`
	NonClaims     []string                   `json:"non_claims"`
}

type VectorizationCoverageRow struct {
	ID            VectorizationID       `json:"id"`
	Name          string                `json:"name"`
	Status        VectorizationStatus   `json:"status"`
	Decision      VectorizationDecision `json:"decision"`
	Candidate     bool                  `json:"candidate,omitempty"`
	RangeProof    bool                  `json:"range_proof,omitempty"`
	ProofID       string                `json:"proof_id,omitempty"`
	MachineTarget string                `json:"machine_target,omitempty"`
	MachinePath   string                `json:"machine_path,omitempty"`
	RequiredFacts []string              `json:"required_facts,omitempty"`
	MissingFacts  []string              `json:"missing_facts,omitempty"`
	Reason        string                `json:"reason"`
	Evidence      string                `json:"evidence"`
	Boundary      string                `json:"boundary"`
}

func VectorizationCoverage() (VectorizationCoverageReport, error) {
	sum, err := vectorizationSumI32Row()
	if err != nil {
		return VectorizationCoverageReport{}, err
	}
	copyU8, err := vectorizationCopyU8Row()
	if err != nil {
		return VectorizationCoverageReport{}, err
	}
	memsetMemcpy, err := vectorizationMemsetMemcpyRow()
	if err != nil {
		return VectorizationCoverageReport{}, err
	}
	mapI32, err := vectorizationMapI32Row()
	if err != nil {
		return VectorizationCoverageReport{}, err
	}
	return VectorizationCoverageReport{
		SchemaVersion: "tetra.optimizer.vectorization.v1",
		Rows: []VectorizationCoverageRow{
			sum,
			copyU8,
			memsetMemcpy,
			mapI32,
		},
		NonClaims: []string{
			"no broad SIMD or auto-vectorization claim",
			"no throughput or C/Rust performance parity claim",
			"no vector path is selected without noalias/provenance and alignment or safe-unaligned evidence",
		},
	}, nil
}

func vectorizationMemsetMemcpyRow() (VectorizationCoverageRow, error) {
	memsetFn := vectorizationMemsetZeroU8Func(true)
	memsetPlan, memsetOK, err := machine.VectorU8x16MemsetZeroPlanFromStackIR(memsetFn)
	if err != nil {
		return VectorizationCoverageRow{}, err
	}
	if !memsetOK {
		return VectorizationCoverageRow{}, fmt.Errorf(
			("vectorization coverage: proof-tagged memset_zero_u8 helper " +
				"did not match vector-u8x16-memset-zero machine shape"),
		)
	}
	copyFn := vectorizationCopyU8Func(true)
	copyPlan, copyOK, err := machine.VectorU8x16CopyLoopPlanFromStackIR(copyFn)
	if err != nil {
		return VectorizationCoverageRow{}, err
	}
	if !copyOK {
		return VectorizationCoverageRow{}, fmt.Errorf(
			("vectorization coverage: memcpy helper evidence did not " +
				"match existing vector-u8x16-copy machine shape"),
		)
	}
	if copyPlan.ProofID == "" {
		return VectorizationCoverageRow{}, fmt.Errorf(
			"vectorization coverage: memcpy helper copy_u8 evidence missing proof id",
		)
	}
	return VectorizationCoverageRow{
		ID:            VectorizationMemsetMemcpy,
		Name:          "memset/memcpy helpers",
		Status:        VectorizationImplementedNarrow,
		Decision:      VectorizationVectorized,
		Candidate:     true,
		RangeProof:    true,
		ProofID:       memsetPlan.ProofID,
		MachineTarget: memsetPlan.Function.Target,
		MachinePath:   "linux-x64-native-simd-vector-u8x16-memset-zero-plus-copy-u8-helper",
		RequiredFacts: []string{
			"range_proof",
			"noalias_not_required_single_mutable_slice_zero_fill",
			"memcpy_helper_reuses_copy_u8_noalias_source_dest_disjoint",
			"safe_unaligned_vector_path",
			"vector_backend_lowering",
			"tail_handling",
			"scalar_fallback",
			"native_simd_codegen",
			"translation_differential_validation",
		},
		Reason: ("vectorized:" +
			"proof_tagged_memset_loop_zero_fill_linux_x64_native_simd_and" +
			"_memcpy_helper_via_copy_u8_with_stack_fallback_differential_" +
			"validation"),
		Evidence: ("compiler/internal/machine/machine_core.go::" +
			"VectorU8x16MemsetZeroPlanFromStackIR; " +
			"compiler/internal/machine/machine_suite_test.go::" +
			"TestVectorU8x16MemsetZeroHelperFromStackIRRequiresRangeSafeU" +
			"nalignedTailAndFallback; " +
			"compiler/internal/backend/x64core/x64core_core.go::" +
			"emitVectorMemsetZeroU8RegisterFunction; " +
			"compiler/internal/backend/linux_x64/codegen_test.go::" +
			"TestCodegenObjectLinuxX64UsesVectorMemsetZeroU8PathForProofH" +
			"elper; compiler/internal/backend/linux_x64/codegen_test.go::" +
			"TestCodegenObjectLinuxX64VectorMemsetZeroU8MatchesStackFallb" +
			"ackWithTail; memcpy helper via copy []u8 evidence reuses " +
			"compiler/internal/machine/machine_core.go::" +
			"VectorU8x16CopyLoopPlanFromStackIR and " +
			"compiler/internal/backend/linux_x64/codegen_test.go::" +
			"TestCodegenObjectLinuxX64VectorCopyU8MatchesStackFallbackWit" +
			"hTail"),
		Boundary: ("proof-tagged memset/memcpy helpers are implemented narrowly:" +
			" memset coverage is limited to a proof-tagged zero-fill " +
			"helper with memset-loop range proof, single mutable slice " +
			"zero-fill noalias-not-required evidence, safe unaligned " +
			"u8x16 vector backend lowering, scalar tail handling, " +
			"scalar-u8-memset-zero fallback through " +
			"vector-u8x16-memset-zero-plan, linux-x64 native SIMD " +
			"codegen, and translation/differential validation against " +
			"stack fallback; memcpy helper via copy []u8 is limited to " +
			"the existing proof-tagged source/dest disjoint copy_u8 " +
			"linux-x64 native SIMD evidence; no arbitrary non-zero " +
			"memset, overlapping memcpy, libc/runtime helper lowering, " +
			"checked/no-proof helper, throughput, or C/Rust parity claim " +
			"is made"),
	}, nil
}

func vectorizationMapI32Row() (VectorizationCoverageRow, error) {
	fn := vectorizationMapI32Func(true)
	vectorPlan, vectorOK, err := machine.VectorI32x4MapAddConstPlanFromStackIR(fn)
	if err != nil {
		return VectorizationCoverageRow{}, err
	}
	if !vectorOK {
		return VectorizationCoverageRow{}, fmt.Errorf(
			("vectorization coverage: proof-tagged map_i32 candidate did " +
				"not match vector-i32x4-map-add-const machine shape"),
		)
	}
	return VectorizationCoverageRow{
		ID:            VectorizationMapI32,
		Name:          "simple map over []i32",
		Status:        VectorizationImplementedNarrow,
		Decision:      VectorizationVectorized,
		Candidate:     true,
		RangeProof:    true,
		ProofID:       vectorPlan.ProofID,
		MachineTarget: vectorPlan.Function.Target,
		MachinePath:   "linux-x64-native-simd-vector-i32x4-map-add-const",
		RequiredFacts: []string{
			"range_proof",
			"noalias_not_required_single_mutable_slice_in_place",
			"safe_unaligned_vector_path",
			"vector_backend_lowering",
			"tail_handling",
			"scalar_fallback",
			"native_simd_codegen",
			"translation_differential_validation",
		},
		Reason: ("vectorized:" +
			"proof_tagged_map_i32_add_const_linux_x64_native_simd_with_st" +
			"ack_fallback_differential_validation"),
		Evidence: ("compiler/internal/machine/machine_core.go::" +
			"VectorI32x4MapAddConstPlanFromStackIR; " +
			"compiler/internal/machine/machine_suite_test.go::" +
			"TestVectorI32x4MapAddConstFromStackIRRequiresRangeSafeUnalig" +
			"nedTailAndFallback; " +
			"compiler/internal/backend/x64core/x64core_core.go::" +
			"emitVectorMapI32AddConstRegisterFunction; " +
			"compiler/internal/backend/linux_x64/codegen_test.go::" +
			"TestCodegenObjectLinuxX64UsesVectorMapI32AddConstPathForProo" +
			"fLoop; compiler/internal/backend/linux_x64/codegen_test.go::" +
			"TestCodegenObjectLinuxX64VectorMapI32AddConstMatchesStackFal" +
			"lbackWithTail"),
		Boundary: ("proof-tagged simple map over []i32 has map-loop range proof," +
			" noalias not required evidence for a single mutable slice " +
			"in-place map, safe unaligned i32x4 vector backend lowering, " +
			"scalar tail handling, scalar-i32-map fallback through " +
			"vector-i32x4-map-add-const-plan, linux-x64 native SIMD " +
			"codegen, and translation/differential validation against " +
			"stack fallback; vectorized scope is limited to proof-tagged " +
			"in-place add-constant-1 map []i32 on linux-x64 machine " +
			"paths and does not claim checked/no-proof map, broader map " +
			"shapes, memset/memcpy, throughput, or C/Rust parity"),
	}, nil
}

func vectorizationCopyU8Row() (VectorizationCoverageRow, error) {
	fn := vectorizationCopyU8Func(true)
	vectorPlan, vectorOK, err := machine.VectorU8x16CopyLoopPlanFromStackIR(fn)
	if err != nil {
		return VectorizationCoverageRow{}, err
	}
	if !vectorOK {
		return VectorizationCoverageRow{}, fmt.Errorf(
			("vectorization coverage: proof-tagged copy_u8 candidate did " +
				"not match vector-u8x16-copy machine shape"),
		)
	}
	noAliasDecision := islandkernel.CanClaimNoAlias(islandKernelVectorCopyNoAliasRequest(vectorPlan.ProofID))
	if noAliasDecision.Decision != islandkernel.Accept {
		return VectorizationCoverageRow{}, fmt.Errorf(
			"vectorization coverage: copy_u8 noalias rejected by island kernel: %s",
			noAliasDecision.Reason.Code,
		)
	}
	return VectorizationCoverageRow{
		ID:            VectorizationCopyU8,
		Name:          "copy []u8",
		Status:        VectorizationImplementedNarrow,
		Decision:      VectorizationVectorized,
		Candidate:     true,
		RangeProof:    true,
		ProofID:       vectorPlan.ProofID,
		MachineTarget: vectorPlan.Function.Target,
		MachinePath:   "machine-ir-vector-u8x16-copy-plan",
		RequiredFacts: []string{
			"range_proof",
			"noalias_source_dest_disjoint_owned_copy_result",
			noAliasDecision.Reason.Code,
			"safe_unaligned_vector_path",
			"vector_backend_lowering",
			"tail_handling",
			"scalar_fallback",
			"native_simd_codegen",
			"translation_differential_validation",
		},
		Reason: ("vectorized:" +
			"proof_tagged_copy_u8_linux_x64_native_simd_with_stack_fallba" +
			"ck_differential_validation:" + noAliasDecision.Reason.Code),
		Evidence: ("compiler/internal/plir/plir.go::addCopyLoopRangeProof; " +
			"compiler/internal/plir/plir_test/plir_test.go::" +
			"TestFromCheckedProgramRecordsBorrowCopyFacts; " +
			"compiler/internal/machine/machine_core.go::" +
			"VectorU8x16CopyLoopPlanFromStackIR; " +
			"compiler/internal/machine/machine_suite_test.go::" +
			"TestVectorU8x16CopyLoopFromStackIRRequiresRangeNoAliasSafeUn" +
			"alignedTailAndFallback; " +
			"compiler/internal/backend/x64core/x64core_core.go::" +
			"emitVectorCopyU8RegisterFunction; " +
			"compiler/internal/backend/linux_x64/codegen_test.go::" +
			"TestCodegenObjectLinuxX64UsesVectorCopyU8PathForProofLoop; " +
			"compiler/internal/backend/linux_x64/codegen_test.go::" +
			"TestCodegenObjectLinuxX64VectorCopyU8MatchesStackFallbackWit" +
			"hTail"),
		Boundary: ("proof-tagged copy []u8 has copy-loop range proof, noalias " +
			"required evidence through source/dest disjoint owned copy " +
			"result, safe unaligned u8x16 vector backend lowering, " +
			"scalar tail handling, scalar-u8-copy fallback through " +
			"vector-u8x16-copy-plan, linux-x64 native SIMD codegen, and " +
			"translation/differential validation against stack fallback; " +
			"vectorized scope is limited to proof-tagged copy []u8 on " +
			"linux-x64 machine paths and does not claim checked/no-proof " +
			"copy, overlapping slices, memset/memcpy, map []i32, " +
			"throughput, or C/Rust parity"),
	}, nil
}

func islandKernelVectorCopyNoAliasRequest(proofID string) islandkernel.NoAliasRequest {
	left := islandkernel.MemoryRef{
		BaseID:      "vector.copy_u8.src",
		IslandID:    "vector.copy_u8.src",
		Epoch:       1,
		OwnerID:     "vector.copy_u8",
		Provenance:  islandkernel.ProvenanceOwned,
		AliasState:  islandkernel.AliasUniqueLocal,
		UnsafeClass: islandkernel.UnsafeSafe,
	}
	right := islandkernel.MemoryRef{
		BaseID:      "vector.copy_u8.dst",
		IslandID:    "vector.copy_u8.dst",
		Epoch:       1,
		OwnerID:     "vector.copy_u8",
		Provenance:  islandkernel.ProvenanceOwned,
		AliasState:  islandkernel.AliasUniqueLocal,
		UnsafeClass: islandkernel.UnsafeSafe,
	}
	return islandkernel.NoAliasRequest{
		Left:  left,
		Right: right,
		Proof: islandkernel.Proof{
			ID:            proofID,
			Kind:          islandkernel.ProofNoAlias,
			SubjectBaseID: left.BaseID,
			IslandID:      left.IslandID,
			Epoch:         left.Epoch,
			Operation:     islandkernel.OperationNoAlias,
			Verified:      strings.TrimSpace(proofID) != "",
		},
	}
}

func vectorizationSumI32Row() (VectorizationCoverageRow, error) {
	fn := hotLoopSliceSumFunc(true)
	ssaVerified, err := hotLoopSSAVerified(fn)
	if err != nil {
		return VectorizationCoverageRow{}, err
	}
	plan, ok, err := machine.ScalarI32SliceSumLoopPlanFromStackIR(fn)
	if err != nil {
		return VectorizationCoverageRow{}, err
	}
	if !ok {
		return VectorizationCoverageRow{}, fmt.Errorf(
			("vectorization coverage: proof-tagged sum_i32 candidate did " +
				"not match scalar-i32-slice-sum machine shape"),
		)
	}
	vectorPlan, vectorOK, err := machine.VectorI32x4SliceSumLoopPlanFromStackIR(fn)
	if err != nil {
		return VectorizationCoverageRow{}, err
	}
	if !vectorOK {
		return VectorizationCoverageRow{}, fmt.Errorf(
			("vectorization coverage: proof-tagged sum_i32 candidate did " +
				"not match vector-i32x4-slice-sum machine shape"),
		)
	}
	if !ssaVerified || plan.ProofID == "" {
		return VectorizationCoverageRow{}, fmt.Errorf(
			"vectorization coverage: proof-tagged sum_i32 candidate missing SSA/range proof evidence",
		)
	}
	return VectorizationCoverageRow{
		ID:            VectorizationSumI32,
		Name:          "sum []i32",
		Status:        VectorizationImplementedNarrow,
		Decision:      VectorizationVectorized,
		Candidate:     true,
		RangeProof:    true,
		ProofID:       vectorPlan.ProofID,
		MachineTarget: vectorPlan.Function.Target,
		MachinePath:   "linux-x64-native-simd-vector-i32x4-slice-sum",
		RequiredFacts: []string{
			"range_proof",
			"noalias_not_required_read_only_reduction",
			"safe_unaligned_vector_path",
			"vector_backend_lowering",
			"tail_handling",
			"scalar_fallback",
			"native_simd_codegen",
			"translation_differential_validation",
		},
		Reason: "vectorized:proof_sum_i32_safe_unaligned_i32x4_native_simd_validated",
		Evidence: ("compiler/internal/opt/opt_core.go::CoreHotLoopShapeEvidence;" +
			" compiler/internal/machine/machine_core.go::" +
			"ScalarI32SliceSumLoopPlanFromStackIR; " +
			"compiler/internal/machine/machine_core.go::" +
			"VectorI32x4SliceSumLoopPlanFromStackIR; " +
			"compiler/internal/machine/machine_suite_test.go::" +
			"TestVectorI32x4SliceSumLoopFromStackIRUsesSafeUnalignedTailA" +
			"ndScalarFallback; " +
			"compiler/internal/backend/x64core/x64core_core.go::" +
			"emitVectorSliceSumRegisterFunction; " +
			"compiler/internal/backend/linux_x64/codegen_test.go::" +
			"TestCodegenObjectLinuxX64UsesVectorSliceSumPathForProofLoop;" +
			" compiler/internal/backend/linux_x64/codegen_test.go::" +
			"TestCodegenObjectLinuxX64VectorSliceSumMatchesStackFallbackW" +
			"ithTail; " +
			"docs/audits/compiler/safety/noalias-mutable-borrow-v1.md::" +
			"read-only reduction means noalias not required for sum []i32"),
		Boundary: ("proof-tagged sum []i32 has range proof, noalias not " +
			"required because this is a read-only reduction with no " +
			"slice memory stores, safe unaligned i32x4 vector backend " +
			"lowering, scalar tail handling, scalar-i32-slice-sum " +
			"fallback through vector-i32x4-slice-sum-plan, linux-x64 " +
			"native SIMD codegen, and translation/differential " +
			"validation against stack fallback; vectorized scope is " +
			"limited to proof-tagged step=1 sum []i32 on linux-x64 " +
			"machine paths and does not claim checked/no-proof loops, " +
			"constant stride, copy []u8, memset/memcpy, map []i32, " +
			"throughput, or C/Rust parity"),
	}, nil
}

func vectorizationNotYetCoveredRow(
	id VectorizationID,
	name string,
	reason string,
) VectorizationCoverageRow {
	return VectorizationCoverageRow{
		ID:       id,
		Name:     name,
		Status:   VectorizationNotYetCovered,
		Decision: VectorizationNotVectorized,
		Reason:   "not_vectorized:" + reason,
		Evidence: "P17.3 master-plan initial target row",
		Boundary: reason + ("; no vector candidate, no range proof, no noalias/alignment " +
			"facts, no vector backend lowering, and no SIMD/performance " +
			"claim"),
	}
}

func vectorizationCopyU8Func(proof bool) ir.IRFunc {
	loadKind := ir.IRIndexLoadU8
	proofID := ""
	if proof {
		loadKind = ir.IRIndexLoadU8Unchecked
		proofID = "proof:copy-loop:u8:1:1"
	}
	return ir.IRFunc{
		Name:        "copy_u8",
		ParamSlots:  3,
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: loadKind, ProofID: proofID},
			{Kind: ir.IRIndexStoreU8},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
}

func vectorizationMapI32Func(proof bool) ir.IRFunc {
	loadKind := ir.IRIndexLoadI32
	proofID := ""
	if proof {
		loadKind = ir.IRIndexLoadI32Unchecked
		proofID = "proof:map-loop:i32:1:1"
	}
	return ir.IRFunc{
		Name:        "map_i32_add1",
		ParamSlots:  2,
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: loadKind, ProofID: proofID},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRIndexStoreI32},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
}

func vectorizationMemsetZeroU8Func(proof bool) ir.IRFunc {
	proofID := ""
	if proof {
		proofID = "proof:memset-loop:u8:zero:1:1"
	}
	return ir.IRFunc{
		Name:        "memset_zero_u8",
		ParamSlots:  2,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRIndexStoreU8, ProofID: proofID},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
}
