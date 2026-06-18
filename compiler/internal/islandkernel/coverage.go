package islandkernel

import (
	"fmt"
	"strings"
)

type RouteStrategy string

const (
	RouteThroughIslandKernel RouteStrategy = "islandkernel"
	RouteValidatedEquivalent RouteStrategy = "validated_equivalent"
	RouteNotApplicable       RouteStrategy = "not_applicable"
)

type DangerousDecisionRoute struct {
	Decision            string
	Question            string
	Strategy            RouteStrategy
	KernelFunction      string
	SourceFiles         []string
	TestFiles           []string
	EvidenceTokens      []string
	Equivalent          string
	NonApplicableReason string
}

func RequiredDangerousDecisions() []string {
	return []string{
		"CanBorrow",
		"CanReturn",
		"CanStoreGlobal",
		"CanCaptureClosure",
		"CanSendToActor",
		"CanSendToTask",
		"CanMoveIsland",
		"CanFreeIsland",
		"CanResetIsland",
		"CanClaimNoAlias",
		"CanEliminateBoundsCheck",
		"CanLowerAsExplicitIsland",
		"CanPromoteUnsafeRoot",
		"CanTrustStorage",
		"CanEraseRuntimeCheck",
	}
}

func DangerousDecisionRoutes() []DangerousDecisionRoute {
	return []DangerousDecisionRoute{
		{
			Decision:       "CanBorrow",
			Question:       "May a reference borrow through this island token and epoch?",
			Strategy:       RouteValidatedEquivalent,
			KernelFunction: "CanBorrow",
			SourceFiles: []string{
				"compiler/internal/islandkernel/kernel.go",
				"compiler/internal/semantics/semantics_memory_resources.go",
				"compiler/internal/memoryfacts/fromplir/from_plir.go",
			},
			TestFiles: []string{
				"compiler/internal/islandkernel/kernel_test.go",
				"compiler/internal/memoryfacts_test/from_plir_test.go",
				"compiler/internal/memoryfacts_test/graph_test.go",
			},
			EvidenceTokens: []string{
				"CanBorrow",
				"bindExplicitBorrow",
				"borrowed_imm",
				"TestMemoryFactsAcceptsSafeBorrowedView",
			},
			Equivalent: ("regionState assigns borrowed regions and MemoryFactGraph " +
				"projects borrowed facts with parent/source validation."),
		},
		{
			Decision:       "CanReturn",
			Question:       "May this memory reference escape through return?",
			Strategy:       RouteValidatedEquivalent,
			KernelFunction: "CanReturn",
			SourceFiles: []string{
				"compiler/internal/islandkernel/kernel.go",
				"compiler/internal/allocplan/plan.go",
				"compiler/internal/validation/validation.go",
			},
			TestFiles: []string{
				"compiler/internal/islandkernel/kernel_test.go",
				"compiler/internal/allocplan/plan_test.go",
				"compiler/internal/validation/validation_test.go",
			},
			EvidenceTokens: []string{
				"CanReturn",
				"EscapeReturn",
				"allocation is returned",
				"TestValidateAllocationLoweringRejectsReturnedExplicitIslandAllocation",
			},
			Equivalent: ("allocation escape classification and validation reject trusted " +
				"stack/island storage when the allocation escapes by return."),
		},
		{
			Decision:       "CanStoreGlobal",
			Question:       "May this memory reference be stored globally?",
			Strategy:       RouteValidatedEquivalent,
			KernelFunction: "CanStoreGlobal",
			SourceFiles: []string{
				"compiler/internal/islandkernel/kernel.go",
				"compiler/internal/allocplan/plan.go",
				"compiler/internal/validation/validation.go",
			},
			TestFiles: []string{
				"compiler/internal/islandkernel/kernel_test.go",
				"compiler/internal/allocplan/plan_test.go",
				"compiler/internal/validation/validation_test.go",
			},
			EvidenceTokens: []string{
				"CanStoreGlobal",
				"EscapeGlobal",
				"stored in global state",
				"TestValidateAllocationLoweringRejectsGlobalStoredStackAllocation",
			},
			Equivalent: ("global-store escape classification forces conservative heap " +
				"storage and validation rejects trusted storage for global escapes."),
		},
		{
			Decision:       "CanCaptureClosure",
			Question:       "May this memory reference be captured by an escaping closure?",
			Strategy:       RouteValidatedEquivalent,
			KernelFunction: "CanCaptureClosure",
			SourceFiles: []string{
				"compiler/internal/islandkernel/kernel.go",
				"compiler/internal/semantics/semantics_memory_resources.go",
				"compiler/internal/allocplan/plan.go",
			},
			TestFiles: []string{
				"compiler/internal/islandkernel/kernel_test.go",
				"compiler/internal/semantics/semantics_suite_test.go",
				"compiler/internal/allocplan/plan_test.go",
			},
			EvidenceTokens: []string{
				"CanCaptureClosure",
				"classifyCallableEscape",
				"EscapeClosure",
				"TestClassifyCallableEscapeUsesHandleForOversizedReturn",
			},
			Equivalent: ("callable escape classification rejects unsupported captures and " +
				"allocation planning treats closure captures as heap/conservative " +
				"escapes."),
		},
		{
			Decision:       "CanSendToActor",
			Question:       "May this memory reference cross an actor boundary?",
			Strategy:       RouteValidatedEquivalent,
			KernelFunction: "CanSendToActor",
			SourceFiles: []string{
				"compiler/internal/islandkernel/kernel.go",
				"compiler/internal/memoryfacts/fromplir/from_plir.go",
				"compiler/internal/allocplan/plan.go",
			},
			TestFiles: []string{
				"compiler/internal/islandkernel/kernel_test.go",
				"compiler/internal/memoryfacts_test/from_plir_test.go",
				"compiler/internal/allocplan/plan_test.go",
			},
			EvidenceTokens: []string{
				"CanSendToActor",
				"actor_boundary_borrow_rejected",
				"EscapeActor",
				"TestMemoryIdealV4ProjectsAsyncTaskActorBoundaryFacts",
			},
			Equivalent: ("actor boundary projection rejects borrowed rows and allocation " +
				"planning avoids trusted storage for actor escapes."),
		},
		{
			Decision:       "CanSendToTask",
			Question:       "May this memory reference cross a task boundary?",
			Strategy:       RouteValidatedEquivalent,
			KernelFunction: "CanSendToTask",
			SourceFiles: []string{
				"compiler/internal/islandkernel/kernel.go",
				"compiler/internal/memoryfacts/fromplir/from_plir.go",
				"compiler/internal/allocplan/plan.go",
			},
			TestFiles: []string{
				"compiler/internal/islandkernel/kernel_test.go",
				"compiler/internal/memoryfacts_test/from_plir_test.go",
				"compiler/internal/allocplan/plan_test.go",
			},
			EvidenceTokens: []string{
				"CanSendToTask",
				"task_boundary_borrow_rejected",
				"EscapeTask",
				"TestMemoryIdealV4ProjectsAsyncTaskActorBoundaryFacts",
			},
			Equivalent: ("task boundary projection rejects borrowed rows and allocation " +
				"planning avoids trusted storage for task escapes."),
		},
		{
			Decision:       "CanMoveIsland",
			Question:       "May this island token move to a new owner?",
			Strategy:       RouteThroughIslandKernel,
			KernelFunction: "CanMoveIsland",
			SourceFiles: []string{
				"compiler/internal/islandkernel/kernel.go",
			},
			TestFiles: []string{
				"compiler/internal/islandkernel/kernel_test.go",
			},
			EvidenceTokens: []string{
				"CanMoveIsland",
				"ConsumesToken",
				"token.move_consumes_source",
			},
			Equivalent: ("current checked route is the kernel decision API; broader token " +
				"lifetime expansion remains MEMISL-P05 scope."),
		},
		{
			Decision:       "CanFreeIsland",
			Question:       "May this island token be freed now?",
			Strategy:       RouteValidatedEquivalent,
			KernelFunction: "CanFreeIsland",
			SourceFiles: []string{
				"compiler/internal/islandkernel/kernel.go",
				"compiler/internal/validation/validation.go",
				"compiler/internal/runtimeabi/allocation_contract.go",
			},
			TestFiles: []string{
				"compiler/internal/islandkernel/kernel_test.go",
				"compiler/internal/validation/validation_test.go",
			},
			EvidenceTokens: []string{
				"CanFreeIsland",
				"IRIslandFree",
				"AllocationDebugDoubleFree",
				"TestValidateAllocationLoweringRejectsExplicitIslandDoubleFree",
			},
			Equivalent: ("explicit island validation tracks free operations and runtime " +
				"contracts name double-free instrumentation; MEMISL-P05 expands token " +
				"misuse cases."),
		},
		{
			Decision:       "CanResetIsland",
			Question:       "May this island token reset and advance epoch?",
			Strategy:       RouteValidatedEquivalent,
			KernelFunction: "CanResetIsland",
			SourceFiles: []string{
				"compiler/internal/islandkernel/kernel.go",
				"compiler/internal/validation/validation.go",
				"compiler/internal/runtimeabi/allocation_contract.go",
			},
			TestFiles: []string{
				"compiler/internal/islandkernel/kernel_test.go",
				"compiler/internal/validation/validation_test.go",
			},
			EvidenceTokens: []string{
				"CanResetIsland",
				"IRIslandReset",
				"AllocationDebugRegionReset",
				"TestValidateAllocationLoweringRejectsExplicitIslandUseAfterReset",
			},
			Equivalent: ("explicit island validation tracks reset/use-after-reset and " +
				"runtime contracts name reset instrumentation; MEMISL-P05 expands epoch " +
				"matrix coverage."),
		},
		{
			Decision:       "CanClaimNoAlias",
			Question:       "May this operation claim noalias?",
			Strategy:       RouteValidatedEquivalent,
			KernelFunction: "CanClaimNoAlias",
			SourceFiles: []string{
				"compiler/internal/islandkernel/kernel.go",
				"compiler/internal/memoryfacts/fromplir/from_plir_copy.go",
				"compiler/internal/memoryfacts/graph.go",
				"compiler/internal/memoryfacts/report.go",
			},
			TestFiles: []string{
				"compiler/internal/islandkernel/kernel_test.go",
				"compiler/internal/memoryfacts_test/from_plir_boundaries_test.go",
				"compiler/internal/memoryfacts_test/graph_test.go",
				"compiler/internal/memoryfacts_test/report_test.go",
			},
			EvidenceTokens: []string{
				"CanClaimNoAlias",
				"no_alias_validated_narrow_unique_local",
				"validated no_alias",
				"TestMemoryIdealV0ProjectsNarrowInoutNoAliasFacts",
			},
			Equivalent: ("MemoryFactGraph/report validation accepts only narrow compiler-" +
				"owned noalias evidence and rejects broad or unsafe_unknown noalias " +
				"claims."),
		},
		{
			Decision:       "CanEliminateBoundsCheck",
			Question:       "May this bounds check be removed?",
			Strategy:       RouteValidatedEquivalent,
			KernelFunction: "CanEliminateBoundsCheck",
			SourceFiles: []string{
				"compiler/internal/islandkernel/kernel.go",
				"compiler/internal/validation/validation.go",
				"compiler/internal/memoryfacts/from_validation.go",
				"compiler/internal/memoryfacts/report.go",
			},
			TestFiles: []string{
				"compiler/internal/islandkernel/kernel_test.go",
				"compiler/internal/validation/validation_test.go",
				"compiler/internal/memoryfacts_test/from_validation_test.go",
				"compiler/internal/memoryfacts_test/report_test.go",
			},
			EvidenceTokens: []string{
				"CanEliminateBoundsCheck",
				"bounds_check_removed_with_proof_id",
				"bounds_proof_id_validator",
				"TestCheckBoundsProofsWithPLIRRejectsTypedProofBaseMismatch",
			},
			Equivalent: ("validation emits typed proof terms and memory report validation " +
				"rejects bare bounds-check elimination without compiler-owned proof ids."),
		},
		{
			Decision:       "CanLowerAsExplicitIsland",
			Question:       "May this allocation lower as ExplicitIsland storage?",
			Strategy:       RouteValidatedEquivalent,
			KernelFunction: "CanLowerAsExplicitIsland",
			SourceFiles: []string{
				"compiler/internal/islandkernel/kernel.go",
				"compiler/internal/allocplan/plan.go",
				"compiler/internal/lower/lower_expressions.go",
				"compiler/internal/validation/validation.go",
				"compiler/internal/memoryfacts/report.go",
			},
			TestFiles: []string{
				"compiler/internal/islandkernel/kernel_test.go",
				"compiler/internal/allocplan/plan_test.go",
				"compiler/internal/lower/lower_suite_test.go",
				"compiler/internal/validation/validation_test.go",
				"compiler/internal/memoryfacts_test/report_test.go",
			},
			EvidenceTokens: []string{
				"CanLowerAsExplicitIsland",
				"validated_explicit_island_scope",
				"lowerExplicitIslandAllocationLet",
				"TestValidateAllocationLoweringRejectsMissingExplicitIslandIR",
			},
			Equivalent: ("allocation planning, lowering, validation, and memory report " +
				"checks require explicit island storage/lowering agreement and reject " +
				"escape/fallback mismatches."),
		},
		{
			Decision:       "CanPromoteUnsafeRoot",
			Question:       "May an unsafe root be promoted to safe memory?",
			Strategy:       RouteValidatedEquivalent,
			KernelFunction: "CanPromoteUnsafeRoot",
			SourceFiles: []string{
				"compiler/internal/islandkernel/kernel.go",
				"compiler/internal/memoryfacts/fromplir/from_plir_allocplan.go",
				"compiler/internal/memoryfacts/fromplir/from_plir_unsafe.go",
				"compiler/internal/memoryfacts/graph.go",
				"compiler/internal/memoryfacts/report.go",
			},
			TestFiles: []string{
				"compiler/internal/islandkernel/kernel_test.go",
				"compiler/internal/memoryfacts_test/from_plir_boundaries_test.go",
				"compiler/internal/memoryfacts_test/graph_test.go",
				"compiler/internal/memoryfacts_test/report_test.go",
			},
			EvidenceTokens: []string{
				"CanPromoteUnsafeRoot",
				"unsafe_unknown_rejected_safe_facts",
				"unsafe_verified_root_allocation_base",
				"TestMemoryIdealV5ProjectsRawPointerUnsafeContractFacts",
			},
			Equivalent: ("unsafe_unknown remains rejected/conservative and unsafe_" +
				"verified_root is limited to bounded allocation-base metadata without " +
				"safe/noalias promotion."),
		},
		{
			Decision:       "CanTrustStorage",
			Question:       "May this storage claim be trusted?",
			Strategy:       RouteValidatedEquivalent,
			KernelFunction: "CanTrustStorage",
			SourceFiles: []string{
				"compiler/internal/islandkernel/kernel.go",
				"compiler/internal/allocplan/plan.go",
				"compiler/internal/memoryfacts/report.go",
				"compiler/internal/memoryfacts/graph.go",
			},
			TestFiles: []string{
				"compiler/internal/islandkernel/kernel_test.go",
				"compiler/internal/allocplan/plan_test.go",
				"compiler/internal/memoryfacts_test/report_test.go",
				"compiler/internal/memoryfacts_test/graph_test.go",
			},
			EvidenceTokens: []string{
				"CanTrustStorage",
				"validatedTrustedStorageHeapFallback",
				"runtimeProofRequiredStorage",
				"TestValidateMemoryReportRejectsValidatedTaskActorRegionStorageWithoutRuntimeProof",
			},
			Equivalent: ("allocation and memory report validators reject heap fallback, " +
				"unsafe_unknown, and ungated runtime-boundary storage as trusted storage."),
		},
		{
			Decision:       "CanEraseRuntimeCheck",
			Question:       "May this runtime check be erased?",
			Strategy:       RouteValidatedEquivalent,
			KernelFunction: "CanEraseRuntimeCheck",
			SourceFiles: []string{
				"compiler/internal/islandkernel/kernel.go",
				"compiler/internal/memoryfacts/from_validation.go",
				"compiler/internal/memoryfacts/fromplir/from_plir.go",
				"compiler/internal/memoryfacts/report.go",
			},
			TestFiles: []string{
				"compiler/internal/islandkernel/kernel_test.go",
				"compiler/internal/memoryfacts_test/from_validation_test.go",
				"compiler/internal/memoryfacts_test/from_plir_test.go",
				"compiler/internal/memoryfacts_test/report_test.go",
			},
			EvidenceTokens: []string{
				"CanEraseRuntimeCheck",
				"bounds_check_retained_dynamic",
				"raw_bounds_runtime_check_normal_build",
				"TestMemoryIdealV6ProjectsBoundsProofFacts",
			},
			Equivalent: ("dynamic raw/bounds uncertainty stays as normal-build checks " +
				"unless compiler-owned proof facts authorize a narrow removal."),
		},
	}
}

func ValidateDangerousDecisionRoutes(
	routes []DangerousDecisionRoute,
	fileExists func(string) bool,
	fileContainsToken func(string, string) bool,
) error {
	if len(routes) == 0 {
		return fmt.Errorf("islandkernel route coverage: no routes")
	}
	required := requiredDangerousDecisionSet()
	seen := map[string]struct{}{}
	var issues []string
	for _, route := range routes {
		decision := strings.TrimSpace(route.Decision)
		if decision == "" {
			issues = append(issues, "route with empty decision")
			continue
		}
		if _, ok := required[decision]; !ok {
			issues = append(
				issues,
				fmt.Sprintf("%s: decision is not in required dangerous decision set", decision),
			)
		}
		if _, ok := seen[decision]; ok {
			issues = append(issues, fmt.Sprintf("%s: duplicate route", decision))
		}
		seen[decision] = struct{}{}
		if strings.TrimSpace(route.Question) == "" {
			issues = append(issues, fmt.Sprintf("%s: missing review question", decision))
		}
		if route.KernelFunction != decision {
			issues = append(
				issues,
				fmt.Sprintf(
					"%s: kernel function %q must match decision",
					decision,
					route.KernelFunction,
				),
			)
		}
		if !knownRouteStrategy(route.Strategy) {
			issues = append(
				issues,
				fmt.Sprintf("%s: unknown route strategy %q", decision, route.Strategy),
			)
		}
		if len(route.SourceFiles) == 0 {
			issues = append(issues, fmt.Sprintf("%s: missing source files", decision))
		}
		for _, path := range append(append([]string{}, route.SourceFiles...), route.TestFiles...) {
			if strings.TrimSpace(path) == "" {
				issues = append(issues, fmt.Sprintf("%s: empty source/test path", decision))
				continue
			}
			if fileExists != nil && !fileExists(path) {
				issues = append(
					issues,
					fmt.Sprintf("%s: source/test file %q does not exist", decision, path),
				)
			}
		}
		if route.Strategy == RouteValidatedEquivalent && strings.TrimSpace(route.Equivalent) == "" {
			issues = append(
				issues,
				fmt.Sprintf("%s: validated equivalent route needs rationale", decision),
			)
		}
		if route.Strategy == RouteNotApplicable &&
			strings.TrimSpace(route.NonApplicableReason) == "" {
			issues = append(issues, fmt.Sprintf("%s: non-applicable route needs reason", decision))
		}
		if len(route.EvidenceTokens) == 0 {
			issues = append(issues, fmt.Sprintf("%s: missing evidence tokens", decision))
		}
		for _, token := range route.EvidenceTokens {
			if strings.TrimSpace(token) == "" {
				issues = append(issues, fmt.Sprintf("%s: empty evidence token", decision))
				continue
			}
			if fileContainsToken == nil {
				continue
			}
			found := false
			for _, path := range append(append([]string{}, route.SourceFiles...), route.TestFiles...) {
				if fileContainsToken(path, token) {
					found = true
					break
				}
			}
			if !found {
				issues = append(
					issues,
					fmt.Sprintf("%s: evidence token %q not found in route files", decision, token),
				)
			}
		}
	}
	for decision := range required {
		if _, ok := seen[decision]; !ok {
			issues = append(issues, fmt.Sprintf("%s: missing route", decision))
		}
	}
	if len(issues) > 0 {
		return fmt.Errorf("islandkernel route coverage failed: %s", strings.Join(issues, "; "))
	}
	return nil
}

func requiredDangerousDecisionSet() map[string]struct{} {
	out := make(map[string]struct{}, len(RequiredDangerousDecisions()))
	for _, decision := range RequiredDangerousDecisions() {
		out[decision] = struct{}{}
	}
	return out
}

func knownRouteStrategy(strategy RouteStrategy) bool {
	switch strategy {
	case RouteThroughIslandKernel, RouteValidatedEquivalent, RouteNotApplicable:
		return true
	default:
		return false
	}
}
