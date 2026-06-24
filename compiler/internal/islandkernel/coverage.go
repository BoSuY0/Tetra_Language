package islandkernel

import (
	"fmt"
	"strings"
)

type RouteStrategy string

const (
	RouteThroughIslandKernel RouteStrategy = "islandkernel"
	RouteNotApplicable       RouteStrategy = "not_applicable"
)

type DangerousDecisionRoute struct {
	Decision             string
	Question             string
	Strategy             RouteStrategy
	KernelFunction       string
	SourceFiles          []string
	TestFiles            []string
	EvidenceTokens       []string
	ProductionCallTokens []string
	NonApplicableReason  string
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
		"CanPlanExplicitIsland",
		"CanLowerAsExplicitIsland",
		"CanPromoteUnsafeRoot",
		"CanTrustStorage",
		"CanEraseRuntimeCheck",
	}
}

func DangerousDecisionRoutes() []DangerousDecisionRoute {
	return []DangerousDecisionRoute{
		directRoute(
			"CanBorrow",
			"May a reference borrow through this island token and epoch?",
			[]string{"compiler/internal/semantics/semantics_checker.go"},
			[]string{"compiler/internal/semantics/semantics_suite_test.go"},
			[]string{"islandkernel.CanBorrow"},
			[]string{"CanBorrow", "borrow.live_epoch", "borrow.stale_epoch"},
		),
		directRoute(
			"CanReturn",
			"May this memory reference escape through return?",
			[]string{"compiler/internal/semantics/semantics_checker.go"},
			[]string{"compiler/internal/semantics/semantics_suite_test.go"},
			[]string{"islandkernel.CanReturn"},
			[]string{"CanReturn", "escape.return_borrow"},
		),
		directRoute(
			"CanStoreGlobal",
			"May this memory reference be stored globally?",
			[]string{"compiler/internal/semantics/semantics_checker.go"},
			[]string{"compiler/internal/semantics/semantics_suite_test.go"},
			[]string{"islandkernel.CanStoreGlobal"},
			[]string{"CanStoreGlobal", "escape.global_borrow"},
		),
		directRoute(
			"CanCaptureClosure",
			"May this memory reference be captured by an escaping closure?",
			[]string{"compiler/internal/semantics/semantics_memory_resources.go"},
			[]string{"compiler/internal/semantics/semantics_suite_test.go"},
			[]string{"islandkernel.CanCaptureClosure"},
			[]string{"CanCaptureClosure", "escape.closure_borrow"},
		),
		directRoute(
			"CanSendToActor",
			"May this memory reference cross an actor boundary?",
			[]string{"compiler/internal/actorsafety/sendability.go"},
			[]string{"compiler/internal/actorsafety/sendability_test.go"},
			[]string{"islandkernel.CanSendToActor"},
			[]string{"CanSendToActor", "boundary.actor_borrow"},
		),
		directRoute(
			"CanSendToTask",
			"May this memory reference cross a task boundary?",
			[]string{"compiler/internal/actorsafety/sendability.go"},
			[]string{"compiler/internal/actorsafety/sendability_test.go"},
			[]string{"islandkernel.CanSendToTask"},
			[]string{"CanSendToTask", "boundary.task_borrow"},
		),
		directRoute(
			"CanMoveIsland",
			"May this island token move to a new owner?",
			[]string{"compiler/internal/actorsafety/sendability.go"},
			[]string{"compiler/internal/actorsafety/sendability_test.go"},
			[]string{"islandkernel.CanMoveIsland"},
			[]string{"CanMoveIsland", "token.move_consumes_source"},
		),
		directRoute(
			"CanFreeIsland",
			"May this island token be freed now?",
			[]string{"compiler/internal/validation/validation_allocation_lifetimes.go"},
			[]string{"compiler/internal/validation/validation_test.go"},
			[]string{"islandkernel.CanFreeIsland"},
			[]string{"CanFreeIsland", "token.free_consumes_source"},
		),
		directRoute(
			"CanResetIsland",
			"May this island token reset and advance epoch?",
			[]string{"compiler/internal/validation/validation_allocation_lifetimes.go"},
			[]string{"compiler/internal/validation/validation_test.go"},
			[]string{"islandkernel.CanResetIsland"},
			[]string{"CanResetIsland", "token.reset_epoch_advanced"},
		),
		directRoute(
			"CanClaimNoAlias",
			"May this operation claim noalias?",
			[]string{"compiler/internal/opt/opt_core.go"},
			[]string{"compiler/internal/opt/opt_suite_test.go"},
			[]string{"islandkernel.CanClaimNoAlias"},
			[]string{"CanClaimNoAlias", "noalias.unsafe_external"},
		),
		directRoute(
			"CanEliminateBoundsCheck",
			"May this bounds check be removed?",
			[]string{"compiler/internal/validation/validation.go"},
			[]string{"compiler/internal/validation/validation_test.go"},
			[]string{"islandkernel.CanEliminateBoundsCheck"},
			[]string{"CanEliminateBoundsCheck", "bounds.proof_verified"},
		),
		directRoute(
			"CanPlanExplicitIsland",
			"May this allocation plan explicit island storage?",
			[]string{"compiler/internal/allocplan/build.go"},
			[]string{"compiler/internal/allocplan/plan_test.go"},
			[]string{"islandkernel.CanPlanExplicitIsland"},
			[]string{"CanPlanExplicitIsland", "plan.explicit_island"},
		),
		directRoute(
			"CanLowerAsExplicitIsland",
			"May this allocation lower as ExplicitIsland storage?",
			[]string{"compiler/internal/validation/validation_allocation_lifetimes.go"},
			[]string{"compiler/internal/validation/validation_test.go"},
			[]string{"islandkernel.CanLowerAsExplicitIsland"},
			[]string{"CanLowerAsExplicitIsland", "storage.explicit_island_trusted"},
		),
		directRoute(
			"CanPromoteUnsafeRoot",
			"May an unsafe root be promoted to safe memory?",
			[]string{"compiler/internal/allocplan/build.go"},
			[]string{"compiler/internal/allocplan/plan_t05_test.go"},
			[]string{"islandkernel.CanPromoteUnsafeRoot"},
			[]string{"CanPromoteUnsafeRoot", "unsafe.unknown_promotion"},
		),
		directRoute(
			"CanTrustStorage",
			"May this storage claim be trusted?",
			[]string{"compiler/internal/validation/validation_allocation_lifetimes.go"},
			[]string{"compiler/internal/validation/validation_test.go"},
			[]string{"islandkernel.CanTrustStorage"},
			[]string{"CanTrustStorage", "storage.trusted_with_proof"},
		),
		directRoute(
			"CanEraseRuntimeCheck",
			"May this runtime check be erased?",
			[]string{"compiler/internal/validation/validation.go"},
			[]string{"compiler/internal/validation/validation_test.go"},
			[]string{"islandkernel.CanEraseRuntimeCheck"},
			[]string{"CanEraseRuntimeCheck", "runtime_check.erase_verified"},
		),
	}
}

func directRoute(
	decision string,
	question string,
	sourceFiles []string,
	testFiles []string,
	productionCallTokens []string,
	evidenceTokens []string,
) DangerousDecisionRoute {
	return DangerousDecisionRoute{
		Decision:       decision,
		Question:       question,
		Strategy:       RouteThroughIslandKernel,
		KernelFunction: decision,
		SourceFiles:    append([]string(nil), sourceFiles...),
		TestFiles: append(
			append([]string{"compiler/internal/islandkernel/kernel.go", "compiler/internal/islandkernel/kernel_test.go"}, testFiles...),
		),
		ProductionCallTokens: append([]string(nil), productionCallTokens...),
		EvidenceTokens:       append([]string(nil), evidenceTokens...),
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
				fmt.Sprintf("%s: kernel function %q must match decision", decision, route.KernelFunction),
			)
		}
		if !knownRouteStrategy(route.Strategy) {
			issues = append(issues, fmt.Sprintf("%s: unknown route strategy %q", decision, route.Strategy))
		}
		if _, ok := required[decision]; ok && route.Strategy != RouteThroughIslandKernel {
			issues = append(issues, fmt.Sprintf("%s: must route through islandkernel", decision))
		}
		if route.Strategy == RouteNotApplicable &&
			strings.TrimSpace(route.NonApplicableReason) == "" {
			issues = append(issues, fmt.Sprintf("%s: non-applicable route needs reason", decision))
		}
		if len(route.SourceFiles) == 0 {
			issues = append(issues, fmt.Sprintf("%s: missing source files", decision))
		}
		if len(route.TestFiles) == 0 {
			issues = append(issues, fmt.Sprintf("%s: missing test files", decision))
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
		validateEvidenceTokens(decision, route, fileContainsToken, &issues)
		validateProductionCallTokens(decision, route, fileContainsToken, &issues)
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

func validateEvidenceTokens(
	decision string,
	route DangerousDecisionRoute,
	fileContainsToken func(string, string) bool,
	issues *[]string,
) {
	if len(route.EvidenceTokens) == 0 {
		*issues = append(*issues, fmt.Sprintf("%s: missing evidence tokens", decision))
	}
	for _, token := range route.EvidenceTokens {
		if strings.TrimSpace(token) == "" {
			*issues = append(*issues, fmt.Sprintf("%s: empty evidence token", decision))
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
			*issues = append(
				*issues,
				fmt.Sprintf("%s: evidence token %q not found in route files", decision, token),
			)
		}
	}
}

func validateProductionCallTokens(
	decision string,
	route DangerousDecisionRoute,
	fileContainsToken func(string, string) bool,
	issues *[]string,
) {
	if route.Strategy != RouteThroughIslandKernel {
		return
	}
	if len(route.ProductionCallTokens) == 0 {
		*issues = append(*issues, fmt.Sprintf("%s: missing production call tokens", decision))
		return
	}
	for _, token := range route.ProductionCallTokens {
		if strings.TrimSpace(token) == "" {
			*issues = append(*issues, fmt.Sprintf("%s: empty production call token", decision))
			continue
		}
		if fileContainsToken == nil {
			continue
		}
		found := false
		for _, path := range route.SourceFiles {
			if strings.Contains(filepathSlash(path), "/islandkernel/") {
				continue
			}
			if fileContainsToken(path, token) {
				found = true
				break
			}
		}
		if !found {
			*issues = append(
				*issues,
				fmt.Sprintf("%s: production call token %q not found in source files", decision, token),
			)
		}
	}
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
	case RouteThroughIslandKernel, RouteNotApplicable:
		return true
	default:
		return false
	}
}

func filepathSlash(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}
