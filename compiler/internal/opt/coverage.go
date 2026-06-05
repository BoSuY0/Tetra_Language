package opt

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
	InliningSpecializationGenericFunctions               InliningSpecializationID = "generic_functions"
	InliningSpecializationSmallPureFunctions             InliningSpecializationID = "small_pure_functions"
	InliningSpecializationStaticProtocolConformanceCalls InliningSpecializationID = "static_protocol_conformance_calls"
	InliningSpecializationExtensionCalls                 InliningSpecializationID = "extension_calls"
	InliningSpecializationEnumKnownCase                  InliningSpecializationID = "enum_known_case"
	InliningSpecializationOptionalUnwrapProvenSome       InliningSpecializationID = "optional_unwrap_proven_some"
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
				Evidence: "compiler/tests/semantics/generics_test.go::TestP9GenericIdentityDisappearsAfterSmallPureInlining; compiler/tests/semantics/generics_test.go::TestP17GenericWrapperDisappearsAfterSmallPureInlining; compiler/internal/opt/inlining.go",
				Boundary: "monomorphized generic identity and generic wrapper calls may disappear only after static monomorphization when the concrete Stack IR callee is accepted by inline-small-pure; small_pure_wrapper is bounded by the same small body limit and translation validation; no runtime generic values, explicit type arguments, generic structs, dynamic dispatch, or broad specialization optimization claim",
			},
			{
				ID:       InliningSpecializationSmallPureFunctions,
				Name:     "small pure functions",
				Status:   InliningSpecializationImplementedNarrow,
				PassName: "inline-small-pure",
				Evidence: "compiler/internal/opt/inlining.go; compiler/internal/opt/inlining_test.go::TestInlineSmallPurePassInlinesCallAndReportsDecision; compiler/internal/opt/inlining_test.go::TestInlineSmallPurePassReportsNotInlinedReasons; compiler/internal/opt/inlining_test.go::TestInlineSmallPurePassDifferentialExecution",
				Boundary: "straight-line Stack IR functions with one return slot, no proof-sensitive instructions, no unsupported effects, and at most 8 candidate body instructions; reports inlined and not_inlined reasons, preserves bounds proofs while invalidating liveness, and uses translation validation; recursive, effectful, proof-sensitive, control-flow, call-containing non-wrapper, oversized, external/runtime, and signature-mismatched callees remain calls",
			},
			{
				ID:       InliningSpecializationStaticProtocolConformanceCalls,
				Name:     "static protocol/conformance calls",
				Status:   InliningSpecializationImplementedNarrow,
				PassName: "inline-small-pure",
				Evidence: "compiler/tests/semantics/inlining_specialization_test.go::TestP17StaticProtocolConformanceCallInlinesAfterSmallPure; compiler/tests/semantics/protocol_conformance_test.go::TestProtocolConformanceChecksExtensionMethod; compiler/internal/layoutopt/layoutopt_test.go::TestSpecializationDevirtualizesProtocolOnlyWhenTargetKnown; compiler/internal/opt/inlining.go",
				Boundary: "statically checked protocol impl method calls that lower to a known direct Stack IR function symbol may disappear when the concrete method body is accepted by inline-small-pure with inlined report decisions and translation validation; layout specialization decision evidence only devirtualizes protocol calls when the target is known; no witness tables, trait objects, runtime protocol values, dynamic dispatch, conformance-table lookup, generic-bound requirement calls, effectful or oversized conformance method inlining, or broad protocol specialization claim",
			},
			{
				ID:       InliningSpecializationExtensionCalls,
				Name:     "extension calls",
				Status:   InliningSpecializationImplementedNarrow,
				PassName: "inline-small-pure",
				Evidence: "compiler/tests/semantics/inlining_specialization_test.go::TestP17StaticExtensionCallInlinesAfterSmallPure; compiler/tests/semantics/extensions_test.go::TestExtensionParseCheckAndLower; compiler/internal/opt/inlining.go",
				Boundary: "statically resolved extension method calls that lower to direct Stack IR function symbols may disappear when the concrete extension method body is accepted by inline-small-pure with inlined report decisions and translation validation; no dynamic extension dispatch, receiver-call sugar specialization, protocol/witness dispatch, effectful or oversized extension method inlining, cross-control-flow specialization, or performance claim",
			},
			{
				ID:       InliningSpecializationEnumKnownCase,
				Name:     "enum constructors/matches with known case",
				Status:   InliningSpecializationImplementedNarrow,
				PassName: "sccp-constant-branch",
				Evidence: "compiler/tests/semantics/inlining_specialization_test.go::TestP17KnownEnumPayloadMatchFoldsAfterSCCP; compiler/internal/opt/sccp.go; compiler/internal/lower/enum_payload_test.go::TestLowerMatchExpressionEnumPayloadIR",
				Boundary: "payload enum constructor tag constants are tracked through same-basic-block Stack IR stores with constant_stack_store evidence, allowing a locally constructed known-case match discriminator branch to fold through sccp-constant-branch with translation validation; no broad enum specialization, payload escape rewrite, cross-control-flow enum fact propagation, exhaustive match pruning, or performance claim",
			},
			{
				ID:       InliningSpecializationOptionalUnwrapProvenSome,
				Name:     "optional unwrap proven some",
				Status:   InliningSpecializationImplementedNarrow,
				PassName: "sccp-constant-branch",
				Evidence: "compiler/tests/semantics/inlining_specialization_test.go::TestP17ProvenSomeOptionalMatchFoldsAfterSCCP; compiler/internal/opt/sccp.go; compiler/tests/semantics/optionals_test.go::TestOptionalMatchExhaustiveNoDefaultWithMultiSlotPayload",
				Boundary: "proven-some optional presence tags are tracked through same-basic-block Stack IR stores with constant_stack_store evidence, allowing a locally constructed optional value to fold the lowered some match branch through sccp-constant-branch with translation validation; no broad optional elimination, unsafe unwrap removal, cross-control-flow optional fact propagation, none-branch pruning claim, or performance claim",
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
				Evidence: "compiler/internal/opt/scalar.go; compiler/internal/opt/scalar_test.go::TestBasicScalarPassFoldsSafeConstantsAndAlgebra; compiler/internal/opt/scalar_test.go::TestBasicScalarPassSimplifiesSameLocalComparisonAlgebra; compiler/internal/opt/scalar_test.go::TestBasicScalarPassFoldsSafeConstDenominatorDivModConstants; compiler/internal/opt/scalar_test.go::TestBasicScalarPassDoesNotFoldUnsafeConstDenominatorDivModConstants",
				Boundary: "safe scalar i32 constants, same-local comparison algebraic forms, safe const-denominator div_i32/mod_i32 constants, and neutral-element algebraic forms only; overflow-sensitive folds and div_i32/mod_i32 denominators 0 and -1 remain rejected",
			},
			{
				ID:       CoreOptimizationCopyPropagation,
				Name:     "copy propagation",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "basic-scalar",
				Evidence: "compiler/internal/opt/scalar.go; compiler/internal/opt/scalar_test.go::TestBasicScalarPassPropagatesCopiesAndEliminatesDeadStores",
				Boundary: "local copy chains in straight-line Stack IR; facts are cleared across side effects and stores",
			},
			{
				ID:       CoreOptimizationDCE,
				Name:     "DCE",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "basic-scalar",
				Evidence: "compiler/internal/opt/scalar.go; compiler/internal/opt/scalar_test.go::TestBasicScalarPassPropagatesCopiesAndEliminatesDeadStores; compiler/internal/opt/scalar_test.go::TestBasicScalarPassEliminatesDeadNonTrappingComparisonStore; compiler/internal/opt/scalar_test.go::TestBasicScalarPassEliminatesDeadSafeKnownLocalUnaryNegStore; compiler/internal/opt/scalar_test.go::TestBasicScalarPassDoesNotEliminateDeadUnsafeKnownLocalUnaryNegStore; compiler/internal/opt/scalar_test.go::TestBasicScalarPassEliminatesDeadSafeKnownLocalArithmeticStore; compiler/internal/opt/scalar_test.go::TestBasicScalarPassDoesNotEliminateDeadUnsafeKnownLocalArithmeticStore; compiler/internal/opt/scalar_test.go::TestBasicScalarPassEliminatesDeadSafeConstDenominatorDivModStore; compiler/internal/opt/scalar_test.go::TestBasicScalarPassEliminatesDeadSafeKnownLocalDivModStore; compiler/internal/opt/scalar_test.go::TestBasicScalarPassDoesNotEliminateDeadUnsafeKnownLocalDivModStore",
				Boundary: "simple dead local stores with single pure producers, non-trapping comparison-expression producers, safe known-local unary neg_i32 producers, safe known-local add_i32/sub_i32/mul_i32 producers, safe const-denominator div_i32/mod_i32 producers, or safe known-local div_i32/mod_i32 producers in straight-line Stack IR; overflow-sensitive unary neg_i32 min-int, overflow-sensitive arithmetic, and div_i32/mod_i32 denominators 0 and -1 are rejected; no general DCE, arbitrary arithmetic-expression DCE, arbitrary div/mod DCE, or unsafe division/modulo DCE claim",
			},
			{
				ID:       CoreOptimizationSCCP,
				Name:     "SCCP",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "sccp-constant-branch",
				Evidence: "compiler/internal/opt/sccp.go; compiler/internal/opt/sccp_test.go::TestSCCPPassFoldsConstantBranchesAndReportsDecisions; compiler/internal/opt/sccp_test.go::TestSCCPPassFoldsSafeUnaryNegExpressionBranch; compiler/internal/opt/sccp_test.go::TestSCCPPassDoesNotFoldUnsafeUnaryNegExpressionBranch; compiler/internal/opt/sccp_test.go::TestSCCPPassFoldsStoredSafeUnaryNegExpressionBranch; compiler/internal/opt/sccp_test.go::TestSCCPPassFoldsSafeConstDenominatorDivModExpressionBranch; compiler/internal/opt/sccp_test.go::TestSCCPPassDoesNotFoldUnsafeConstDenominatorDivModExpressionBranch; compiler/internal/opt/sccp_test.go::TestSCCPPassFoldsStoredConstantExpressionBranch; compiler/internal/opt/sccp_test.go::TestSCCPPassFoldsStoredSafeConstDenominatorDivModExpressionBranch; compiler/internal/opt/sccp_test.go::TestSCCPPassPropagatesKnownLocalThroughSinglePredecessorLabel; compiler/internal/opt/sccp_test.go::TestSCCPPassPropagatesKnownLocalThroughForwardSinglePredecessorJump; compiler/internal/opt/sccp_test.go::TestSCCPPassPropagatesKnownLocalThroughFoldedZeroBranchTarget; compiler/internal/opt/sccp_test.go::TestSCCPPassDoesNotPropagateFoldedZeroBranchThroughFallthroughTarget; compiler/internal/opt/sccp_test.go::TestSCCPPassPropagatesKnownLocalThroughFoldedNonzeroFallthroughLabel; compiler/internal/opt/sccp_test.go::TestSCCPPassDoesNotPropagateFoldedNonzeroFallthroughThroughExplicitIncomingLabel; compiler/internal/opt/sccp_test.go::TestSCCPPassPropagatesDynamicZeroFactThroughSinglePredecessorTarget; compiler/internal/opt/sccp_test.go::TestSCCPPassUsesDynamicNonzeroFallthroughFactForRepeatedLocalBranch; compiler/internal/opt/sccp_test.go::TestSCCPPassDoesNotPropagateDynamicZeroFactThroughFallthroughTarget; compiler/internal/opt/sccp_test.go::TestSCCPPassDerivesEqZeroComparisonPathFacts; compiler/internal/opt/sccp_test.go::TestSCCPPassDerivesNeZeroComparisonPathFacts; compiler/internal/opt/sccp_test.go::TestSCCPPassDoesNotDeriveComparisonTargetFactThroughFallthroughTarget",
				Boundary: "literal constant-condition, same-basic-block known-local including stored safe unary neg_i32 facts and stored safe constant binary-expression facts, same-basic-block constant unary neg_i32 and constant binary-expression branch folding including safe const-denominator div_i32/mod_i32, immediate or forward-terminated single-predecessor label propagation for known-local facts, folded zero-branch target propagation for labels with one incoming edge and no fallthrough predecessor, folded nonzero-branch fallthrough propagation through an immediate label with no explicit incoming branch/jump edges, dynamic load_local zero-target and nonzero-fallthrough path facts plus dynamic zero-comparison eq/ne zero/nonzero path facts for later same-local branches, plus fallthrough pruning to the next label in Stack IR only; overflow-sensitive unary neg_i32 min-int, div_i32/mod_i32 denominators 0 and -1, dynamic stored expressions, multi-predecessor labels, folded nonzero fallthrough labels with explicit incoming edges, dynamic zero-target labels with fallthrough predecessors, dynamic comparison-target labels with fallthrough predecessors, and folded zero-branch target labels with fallthrough predecessors are rejected; no general lattice propagation, arbitrary path-sensitive optimization, range propagation, arbitrary comparison reasoning, or path-sensitive SSA SCCP claim",
			},
			{
				ID:       CoreOptimizationCSEGvn,
				Name:     "CSE/GVN",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "basic-scalar",
				Evidence: "compiler/internal/opt/scalar.go; compiler/internal/opt/scalar_test.go::TestBasicScalarPassEliminatesRepeatedPureLocalExpressionWithCSE; compiler/internal/opt/scalar_test.go::TestBasicScalarPassEliminatesCommutativeLocalExpressionWithGVN; compiler/internal/opt/scalar_test.go::TestBasicScalarPassEliminatesMirroredComparisonExpressionWithGVN; compiler/internal/opt/scalar_test.go::TestBasicScalarPassEliminatesRepeatedLocalConstantExpressionWithCSE; compiler/internal/opt/scalar_test.go::TestBasicScalarPassEliminatesSafeConstDenominatorDivModExpressionWithCSE; compiler/internal/opt/scalar_test.go::TestBasicScalarPassEliminatesRepeatedUnaryLocalNegExpressionWithCSE; compiler/internal/opt/scalar_test.go::TestBasicScalarPassEliminatesSafeKnownLocalUnaryNegExpressionWithCSE; compiler/internal/opt/scalar_test.go::TestBasicScalarPassDoesNotReuseKnownLocalUnaryNegExpressionAfterSourceMutation; compiler/internal/opt/scalar_test.go::TestBasicScalarPassDoesNotReuseMinIntKnownLocalUnaryNegExpression; compiler/internal/opt/scalar_test.go::TestBasicScalarPassEliminatesSafeKnownLocalArithmeticExpressionWithGVN; compiler/internal/opt/scalar_test.go::TestBasicScalarPassDoesNotReuseKnownLocalArithmeticExpressionAfterSourceMutation; compiler/internal/opt/scalar_test.go::TestBasicScalarPassDoesNotReuseOverflowSensitiveKnownLocalArithmeticExpression; compiler/internal/opt/scalar_test.go::TestBasicScalarPassEliminatesSafeKnownLocalComparisonExpressionWithGVN; compiler/internal/opt/scalar_test.go::TestBasicScalarPassDoesNotReuseKnownLocalComparisonExpressionAfterSourceMutation; compiler/internal/opt/scalar_test.go::TestBasicScalarPassEliminatesSafeKnownLocalDivModExpressionWithGVN; compiler/internal/opt/scalar_test.go::TestBasicScalarPassDoesNotReuseKnownLocalDivModExpressionAfterSourceMutation; compiler/internal/opt/scalar_test.go::TestBasicScalarPassDoesNotReuseUnsafeKnownLocalDivModExpression",
				Boundary: "reuses repeated pure local-load and local-load/constant binary expressions, safe const-denominator div_i32/mod_i32 expressions, unary local neg_i32 expressions, safe known-local unary neg_i32 value expressions, safe known-local add_i32/sub_i32/mul_i32 value expressions, safe known-local cmp_*_i32 value expressions including mirrored ordered comparisons, safe known-local div_i32/mod_i32 value expressions, commutative add/mul/eq/ne operand variants, plus mirrored lt/gt/le/ge ordered-comparison operand variants only while operand/result value facts and cached result locals remain valid; denominators 0 and -1, source-local mutations that change known values, overflow-sensitive unary neg_i32 min-int, overflow-sensitive known-local arithmetic, and unsafe known-local division/modulo are rejected; no global value numbering or SSA GVN claim",
			},
			{
				ID:       CoreOptimizationMem2Reg,
				Name:     "mem2reg",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "mem2reg-single-assignment",
				Evidence: "compiler/internal/opt/mem2reg.go; compiler/internal/opt/mem2reg_test.go::TestMem2RegPassPromotesSingleAssignmentTempAndReportsDecision; compiler/internal/opt/mem2reg_test.go::TestMem2RegPassPromotesSeparatedSingleAssignmentTempWithStackNeutralWork; compiler/internal/opt/mem2reg_test.go::TestMem2RegPassPromotesSeparatedComparisonExpressionTempWithStackNeutralWork; compiler/internal/opt/mem2reg_test.go::TestMem2RegPassPromotesSeparatedSafeConstUnaryNegTempWithStackNeutralWork; compiler/internal/opt/mem2reg_test.go::TestMem2RegPassRejectsSeparatedUnsafeConstUnaryNegTemp; compiler/internal/opt/mem2reg_test.go::TestMem2RegPassPromotesSeparatedSafeKnownLocalUnaryNegTempWithStackNeutralWork; compiler/internal/opt/mem2reg_test.go::TestMem2RegPassRejectsSeparatedUnsafeKnownLocalUnaryNegTemp; compiler/internal/opt/mem2reg_test.go::TestMem2RegPassRejectsSeparatedSafeKnownLocalUnaryNegTempWhenSourceLocalMutates; compiler/internal/opt/mem2reg_test.go::TestMem2RegPassPromotesSeparatedSafeConstArithmeticTempWithStackNeutralWork; compiler/internal/opt/mem2reg_test.go::TestMem2RegPassRejectsSeparatedUnsafeConstArithmeticTemp; compiler/internal/opt/mem2reg_test.go::TestMem2RegPassPromotesSeparatedSafeKnownLocalArithmeticTempWithStackNeutralWork; compiler/internal/opt/mem2reg_test.go::TestMem2RegPassRejectsSeparatedUnsafeKnownLocalArithmeticTemp; compiler/internal/opt/mem2reg_test.go::TestMem2RegPassRejectsSeparatedSafeKnownLocalArithmeticTempWhenSourceLocalMutates; compiler/internal/opt/mem2reg_test.go::TestMem2RegPassPromotesSeparatedSafeConstDenominatorDivModTempWithStackNeutralWork; compiler/internal/opt/mem2reg_test.go::TestMem2RegPassRejectsSeparatedSafeDivModTempWhenSourceLocalMutates; compiler/internal/opt/mem2reg_test.go::TestMem2RegPassRejectsSeparatedUnsafeConstDenominatorDivModTemp; compiler/internal/opt/mem2reg_test.go::TestMem2RegPassPromotesSeparatedSafeKnownLocalDivModTempWithStackNeutralWork; compiler/internal/opt/mem2reg_test.go::TestMem2RegPassRejectsSeparatedSafeKnownLocalDivModTempWhenSourceLocalMutates; compiler/internal/opt/mem2reg_test.go::TestMem2RegPassRejectsSeparatedUnsafeKnownLocalDivModTemp",
				Boundary: "straight-line Stack IR single-store/single-load adjacent temp locals plus stack-neutral separated single pure const/load-local, bounded non-trapping comparison-expression, safe const unary neg_i32, safe known-local unary neg_i32, safe const add_i32/sub_i32/mul_i32 arithmetic, safe known-local add_i32/sub_i32/mul_i32 arithmetic, safe const-denominator div_i32/mod_i32 producer temps, or safe known-local div_i32/mod_i32 producer temps when source locals remain unmodified; overflow-sensitive unary neg_i32 min-int, arithmetic overflow, source-local mutation, and div_i32/mod_i32 denominators 0 and -1 are rejected; no alloca promotion, phi insertion, alias analysis, unsafe unary neg promotion, unsafe arithmetic promotion, unsafe division/modulo promotion, or general SSA mem2reg claim",
			},
			{
				ID:       CoreOptimizationSimpleInlining,
				Name:     "simple inlining",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "inline-small-pure",
				Evidence: "compiler/internal/opt/inlining.go; compiler/internal/opt/inlining_test.go",
				Boundary: "small pure non-recursive Stack IR functions; effects, recursion, calls, and proof-sensitive bodies are rejected",
			},
			{
				ID:       CoreOptimizationLoopCanonicalization,
				Name:     "loop canonicalization",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "loop-canonicalization",
				Evidence: "compiler/internal/opt/loop.go; compiler/internal/opt/loop_test.go",
				Boundary: "selected proof-tagged while-loop shapes with stable length locals",
			},
			{
				ID:       CoreOptimizationLICM,
				Name:     "LICM for pure invariant expressions",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "licm-pure-invariant",
				Evidence: "compiler/internal/opt/licm.go; compiler/internal/opt/licm_test.go::TestLICMPureInvariantPassHoistsPureComparisonInsideProofLoop; compiler/internal/opt/licm_test.go::TestLICMPureInvariantPassHoistsPureArithmeticInsideProofLoop; compiler/internal/opt/licm_test.go::TestLICMPureInvariantPassHoistsPureSubArithmeticInsideProofLoop; compiler/internal/opt/licm_test.go::TestLICMPureInvariantPassHoistsSafeDivArithmeticInsideProofLoop; compiler/internal/opt/licm_test.go::TestLICMPureInvariantPassHoistsSafeModArithmeticInsideProofLoop; compiler/internal/opt/licm_test.go::TestLICMPureInvariantPassHoistsKnownLocalArithmeticInsideProofLoop; compiler/internal/opt/licm_test.go::TestLICMPureInvariantPassRejectsKnownLocalArithmeticWhenOperandMutatesInLoop; compiler/internal/opt/licm_test.go::TestLICMPureInvariantPassHoistsKnownLocalLeftArithmeticInsideProofLoop; compiler/internal/opt/licm_test.go::TestLICMPureInvariantPassRejectsKnownLocalLeftArithmeticWhenOperandMutatesInLoop; compiler/internal/opt/licm_test.go::TestLICMPureInvariantPassHoistsKnownLocalComparisonInsideProofLoop; compiler/internal/opt/licm_test.go::TestLICMPureInvariantPassRejectsKnownLocalComparisonWhenOperandMutatesInLoop; compiler/internal/opt/licm_test.go::TestLICMPureInvariantPassHoistsSafeKnownLocalDivModInsideProofLoop; compiler/internal/opt/licm_test.go::TestLICMPureInvariantPassRejectsUnsafeKnownLocalDivModInsideProofLoop; compiler/internal/opt/licm_test.go::TestLICMPureInvariantPassRejectsSafeKnownLocalDivModWhenDenominatorMutatesInLoop",
				Boundary: "pure load-local/constant comparison, add/sub/mul arithmetic, known-local add_i32/sub_i32/mul_i32 left-or-right operand hoisting, known-local cmp_*_i32 left-or-right operand hoisting, safe const-denominator div_i32/mod_i32 hoisting, and safe known-local div_i32/mod_i32 denominator hoisting for selected proof-tagged while-loop shapes only; div_i32/mod_i32 denominators 0 and -1, loop-index operands, and loop-mutated operands are rejected; no alias analysis, overflow-sensitive safety beyond existing arithmetic LICM semantics, unsafe division/modulo LICM, or general SSA LICM claim",
			},
			{
				ID:       CoreOptimizationAllocationSinking,
				Name:     "allocation sinking",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "lower.stack-allocation",
				Evidence: "compiler/internal/lower/allocation_stack_test.go; compiler/internal/allocplan/plan_test.go",
				Boundary: "lowering/planner evidence for supported local stack/storage elimination cases, not a general optimizer pass",
			},
			{
				ID:       CoreOptimizationScalarReplacement,
				Name:     "scalar replacement",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "lower.scalar-replacement",
				Evidence: "compiler/internal/lower/allocation_stack_test.go::TestLowerScalarReplacementEliminatesTinyConstantIndexSlice",
				Boundary: "tiny fixed constant-index slices, small structs, and fixed arrays in supported lowering paths",
			},
			{
				ID:       CoreOptimizationBoundsCheckElimination,
				Name:     "bounds-check elimination v1",
				Status:   CoreOptimizationImplementedNarrow,
				PassName: "plir-rangeproof-bce",
				Evidence: "compiler/internal/lower/proof_bce_test.go; compiler/internal/rangeproof; compiler/internal/validation/validation.go",
				Boundary: "proof-tagged supported branch, loop, view-chain, and copy-loop shapes only",
			},
		},
	}
}
