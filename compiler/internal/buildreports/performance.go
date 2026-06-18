package buildreports

import (
	"fmt"
	"strings"
)

func BuildPerformanceReport(target string) PerfReport {
	return PerfReport{
		ReportEnvelope: ReportEnvelope{SchemaVersion: 3, Kind: "perf", Target: target},
		MatrixScope:    p20PerformanceMatrixScope,
		MatrixReport:   p20PerformanceMatrixReport,
		Claims: []string{
			"No broad performance claim is made without benchmark evidence.",
			"Allowed claims must cite a benchmark, report artifact, target, and measured comparison row.",
			("No measured speed comparison, C++/Rust parity, official " +
				"benchmark result, official TechEmpower result, P20.2 claim tier, " +
				"optimizer behavior change, or runtime behavior change is claimed by " +
				"this report."),
		},
		Blockers:   p20PerformanceBlockers(),
		Benchmarks: p20PerformanceBenchmarkExplanations(),
	}
}

const (
	p20PerformanceMatrixScope  = "p20.0_benchmark_matrix"
	p20PerformanceMatrixReport = ("reports/benchmark-matrix-hardening-v1/benchmarks/p20-" +
		"matrix-hardening-report.json")
	p20PerformanceReportArtifact = ("reports/benchmark-matrix-hardening-v1/benchmarks/" +
		"artifacts/p20-matrix-hardening.perf.json")
	p20PerformanceProofArtifact = ("reports/benchmark-matrix-hardening-v1/benchmarks/" +
		"artifacts/p20-matrix-hardening.proof.json")
	p20PerformanceAllocArtifact = ("reports/benchmark-matrix-hardening-v1/benchmarks/" +
		"artifacts/p20-matrix-hardening.allocation.json")
	p20PerformanceBoundsArtifact = ("reports/benchmark-matrix-hardening-v1/benchmarks/" +
		"artifacts/p20-matrix-hardening.bounds.json")
	p20PerformanceBackendArtifact = ("reports/benchmark-matrix-hardening-v1/benchmarks/" +
		"artifacts/p20-matrix-hardening.backend.json")
	p20PerformanceActorArtifact = ("reports/benchmark-matrix-hardening-v1/benchmarks/" +
		"artifacts/p20-matrix-hardening.actor-transfer.json")
)

func p20PerformanceBlockers() []PerformanceBlockerRow {
	rows := []PerformanceBlockerRow{
		{
			Code:      "bounds.missing_dominance",
			Component: "bounds-check-elimination",
			Message:   "left bounds check: missing dominance",
			CostClass: "dynamic_check_required",
			Evidence: ("bounds/proof reports must show a proof_id, guard, and dominance " +
				"before the bounds report may mark the check removed"),
			NextStep: ("add or preserve a dominating guard for the indexed access, or " +
				"keep the checked bounds path"),
		},
		{
			Code:      "allocation.return_escape",
			Component: "allocation-planning",
			Message:   "heap allocation: escapes through return",
			CostClass: "conservative_fallback",
			Evidence: ("allocation reports must keep heap or region storage when escape " +
				"analysis classifies a returned value"),
			NextStep: ("return a caller-owned view/copy or make the lifetime explicit " +
				"before expecting stack lowering"),
		},
		{
			Code:      "allocation.unknown_call",
			Component: "allocation-planning",
			Message:   "heap allocation: unknown call",
			CostClass: "conservative_fallback",
			Evidence: ("allocation reports must stay conservative when a call boundary " +
				"lacks escape/lifetime facts"),
			NextStep: ("inline or summarize the callee, add lifetime/effect facts, or " +
				"keep heap/region storage"),
		},
		{
			Code:      "allocation.local_call_heap_fallback",
			Component: "allocation-planning",
			Message:   "heap allocation: local call boundary heap fallback",
			CostClass: "conservative_fallback",
			Evidence: ("allocation reports may prove NoEscape for read-only local call " +
				"arguments while keeping heap storage until call-aware stack or region " +
				"lowering is validated"),
			NextStep: ("prove call-aware stack or region lowering for the local call " +
				"boundary, or keep the explicit heap fallback"),
		},
		{
			Code:      "vector.no_noalias_proof",
			Component: "vectorization",
			Message:   "not vectorized: no noalias proof",
			CostClass: "dynamic_check_required",
			Evidence: ("vectorization reports require provenance/noalias facts before " +
				"selecting a vector path that could observe aliasing"),
			NextStep: "prove source/destination disjointness or keep the scalar path selected",
		},
		{
			Code:      "inline.code_size_budget",
			Component: "inlining",
			Message:   "not inlined: code-size budget",
			CostClass: "instrumentation_only",
			Evidence: ("inlining reports must preserve not_inlined reasons when a body " +
				"exceeds the current budget"),
			NextStep: ("reduce the callee body or accept the call boundary until the " +
				"budget changes with validation"),
		},
		{
			Code:      "register_spill.live_range_pressure",
			Component: "register-allocation",
			Message:   "register spill: live range pressure",
			CostClass: "instrumentation_only",
			Evidence: ("backend reports expose machine intervals, allocation decisions, " +
				"and spill slots for register-path functions"),
			NextStep: ("shorten live ranges, split temporaries, or inspect the machine " +
				"backend allocation row"),
		},
		{
			Code:      "stack_fallback.unsupported_aggregate_return",
			Component: "backend-selection",
			Message:   "stack fallback: unsupported aggregate return",
			CostClass: "conservative_fallback",
			Evidence: ("backend reports keep stack fallback rows when the current " +
				"register ABI cannot return the aggregate shape"),
			NextStep: ("use a supported single-slot return shape or wait for aggregate-" +
				"return register backend evidence"),
		},
		{
			Code:      "actor_copy.borrowed_data_boundary",
			Component: "actor-transfer",
			Message:   "actor copy: borrowed data crosses boundary",
			CostClass: "conservative_fallback",
			Evidence: ("actor transfer reports must keep copy rows when borrowed " +
				"payload data crosses an actor boundary"),
			NextStep: ("transfer owned data/region ownership or keep the explicit copy " +
				"at the actor boundary"),
		},
	}
	return append([]PerformanceBlockerRow(nil), rows...)
}

func p20PerformanceBenchmarkExplanations() []PerformanceBenchmarkExplanation {
	specs := []struct {
		benchmark string
		category  string
		reasons   []string
	}{
		{
			benchmark: "integer_loops_tetra",
			category:  "integer loops",
			reasons:   []string{"register_spill.live_range_pressure", "inline.code_size_budget"},
		},
		{
			benchmark: "slice_sum_tetra",
			category:  "slice sum",
			reasons:   []string{"bounds.missing_dominance", "vector.no_noalias_proof"},
		},
		{
			benchmark: "bounds_check_loops_tetra",
			category:  "bounds-check loops",
			reasons:   []string{"bounds.missing_dominance"},
		},
		{
			benchmark: "function_calls_tetra",
			category:  "function calls",
			reasons:   []string{"inline.code_size_budget"},
		},
		{
			benchmark: "recursion_tetra",
			category:  "recursion",
			reasons:   []string{"inline.code_size_budget", "register_spill.live_range_pressure"},
		},
		{
			benchmark: "matrix_multiply_tetra",
			category:  "matrix multiply",
			reasons:   []string{"vector.no_noalias_proof", "register_spill.live_range_pressure"},
		},
		{
			benchmark: "hash_table_tetra",
			category:  "hash table",
			reasons:   []string{"allocation.local_call_heap_fallback", "inline.code_size_budget"},
		},
		{
			benchmark: "allocation_tetra",
			category:  "allocation",
			reasons:   []string{"allocation.return_escape", "allocation.unknown_call"},
		},
		{
			benchmark: "region_island_allocation_tetra",
			category:  "region/island allocation",
			reasons:   []string{"allocation.return_escape", "allocation.unknown_call"},
		},
		{
			benchmark: "json_parse_stringify_tetra",
			category:  "JSON parse/stringify",
			reasons:   []string{"allocation.unknown_call", "bounds.missing_dominance"},
		},
		{
			benchmark: "http_plaintext_json_tetra",
			category:  "HTTP plaintext/json",
			reasons:   []string{"allocation.unknown_call", "inline.code_size_budget"},
		},
		{
			benchmark: "postgresql_single_multiple_update_tetra",
			category:  "PostgreSQL single/multiple/update",
			reasons: []string{
				"allocation.unknown_call",
				"stack_fallback.unsupported_aggregate_return",
			},
		},
		{
			benchmark: "actor_ping_pong_tetra",
			category:  "actor ping-pong",
			reasons:   []string{"actor_copy.borrowed_data_boundary"},
		},
		{
			benchmark: "parallel_map_reduce_tetra",
			category:  "parallel map/reduce",
			reasons: []string{
				"actor_copy.borrowed_data_boundary",
				"register_spill.live_range_pressure",
			},
		},
		{
			benchmark: "startup_time_tetra",
			category:  "startup time",
			reasons:   []string{"inline.code_size_budget", "allocation.unknown_call"},
		},
		{
			benchmark: "binary_size_tetra",
			category:  "binary size",
			reasons:   []string{"inline.code_size_budget"},
		},
		{
			benchmark: "compile_time_tetra",
			category:  "compile time",
			reasons: []string{
				"inline.code_size_budget",
				"stack_fallback.unsupported_aggregate_return",
			},
		},
	}
	rows := make([]PerformanceBenchmarkExplanation, 0, len(specs))
	for _, spec := range specs {
		rows = append(rows, PerformanceBenchmarkExplanation{
			Benchmark:    spec.benchmark,
			Category:     spec.category,
			MatrixScope:  p20PerformanceMatrixScope,
			MatrixReport: p20PerformanceMatrixReport,
			ReasonCodes:  append([]string(nil), spec.reasons...),
			Artifacts: []string{
				p20PerformanceMatrixReport,
				p20PerformanceReportArtifact,
				p20PerformanceProofArtifact,
				p20PerformanceAllocArtifact,
				p20PerformanceBoundsArtifact,
				p20PerformanceBackendArtifact,
				p20PerformanceActorArtifact,
			},
			Explanation: fmt.Sprintf(
				("%s uses the P20.1 blocker map for %s; inspect the cited reason " +
					"codes and report artifacts before changing source, compiler policy, or " +
					"benchmark wording."),
				spec.benchmark,
				spec.category,
			),
			NextStep: ("open the referenced proof/allocation/bounds/backend/actor-" +
				"transfer reports, fix the first applicable blocker, then rerun the " +
				"benchmark report before making any performance claim"),
		})
	}
	return rows
}

func ValidatePerformanceBlockerReport(report PerfReport) error {
	if report.SchemaVersion != 3 {
		return fmt.Errorf(
			"performance blocker report schema_version = %d, want 3",
			report.SchemaVersion,
		)
	}
	if report.Kind != "perf" {
		return fmt.Errorf("performance blocker report kind = %q, want perf", report.Kind)
	}
	if strings.TrimSpace(report.Target) == "" {
		return fmt.Errorf("performance blocker report target is required")
	}
	if report.MatrixScope != p20PerformanceMatrixScope {
		return fmt.Errorf(
			"performance blocker report matrix_scope = %q, want %q",
			report.MatrixScope,
			p20PerformanceMatrixScope,
		)
	}
	if report.MatrixReport != p20PerformanceMatrixReport {
		return fmt.Errorf(
			"performance blocker report matrix_report = %q, want %q",
			report.MatrixReport,
			p20PerformanceMatrixReport,
		)
	}
	if err := validatePerformanceBlockerClaims(report.Claims); err != nil {
		return err
	}
	requiredBlockers := map[string]PerformanceBlockerRow{}
	for _, row := range p20PerformanceBlockers() {
		requiredBlockers[row.Code] = row
	}
	seenBlockers := map[string]bool{}
	for _, row := range report.Blockers {
		if strings.TrimSpace(row.Code) == "" {
			return fmt.Errorf("performance blocker row missing code")
		}
		if seenBlockers[row.Code] {
			return fmt.Errorf("duplicate performance blocker code %q", row.Code)
		}
		seenBlockers[row.Code] = true
		required, ok := requiredBlockers[row.Code]
		if !ok {
			return fmt.Errorf("unknown performance blocker code %q", row.Code)
		}
		if row.Message != required.Message {
			return fmt.Errorf(
				"performance blocker %s message = %q, want %q",
				row.Code,
				row.Message,
				required.Message,
			)
		}
		if !knownPerformanceCostClass(row.CostClass) {
			return fmt.Errorf(
				"performance blocker %s unknown cost_class %q",
				row.Code,
				row.CostClass,
			)
		}
		if row.CostClass != required.CostClass {
			return fmt.Errorf(
				"performance blocker %s cost_class = %q, want %q",
				row.Code,
				row.CostClass,
				required.CostClass,
			)
		}
		if isWeakPerformanceText(row.Component) || isWeakPerformanceText(row.Evidence) ||
			isWeakPerformanceText(row.NextStep) {
			return fmt.Errorf(
				"performance blocker %s has placeholder evidence or next step",
				row.Code,
			)
		}
	}
	for code := range requiredBlockers {
		if !seenBlockers[code] {
			return fmt.Errorf("performance blocker report missing required blocker %s", code)
		}
	}

	requiredBenchmarks := map[string]string{}
	for _, row := range p20PerformanceBenchmarkExplanations() {
		requiredBenchmarks[row.Benchmark] = row.Category
	}
	knownReasons := map[string]bool{}
	for code := range requiredBlockers {
		knownReasons[code] = true
	}
	seenBenchmarks := map[string]bool{}
	for _, row := range report.Benchmarks {
		if strings.TrimSpace(row.Benchmark) == "" {
			return fmt.Errorf("performance benchmark explanation missing benchmark")
		}
		if seenBenchmarks[row.Benchmark] {
			return fmt.Errorf("duplicate performance benchmark explanation %q", row.Benchmark)
		}
		seenBenchmarks[row.Benchmark] = true
		requiredCategory, ok := requiredBenchmarks[row.Benchmark]
		if !ok {
			return fmt.Errorf("unknown performance benchmark explanation %q", row.Benchmark)
		}
		if row.Category != requiredCategory {
			return fmt.Errorf(
				"performance benchmark %s category = %q, want %q",
				row.Benchmark,
				row.Category,
				requiredCategory,
			)
		}
		if row.MatrixScope != report.MatrixScope || row.MatrixReport != report.MatrixReport {
			return fmt.Errorf(
				"performance benchmark %s matrix linkage = %q/%q",
				row.Benchmark,
				row.MatrixScope,
				row.MatrixReport,
			)
		}
		if len(row.ReasonCodes) == 0 {
			return fmt.Errorf("performance benchmark %s missing reason codes", row.Benchmark)
		}
		for _, code := range row.ReasonCodes {
			if !knownReasons[code] {
				return fmt.Errorf(
					"performance benchmark %s cites unknown reason code %q",
					row.Benchmark,
					code,
				)
			}
		}
		if len(row.Artifacts) == 0 {
			return fmt.Errorf("performance benchmark %s missing report artifacts", row.Benchmark)
		}
		if isWeakPerformanceText(row.Explanation) || isWeakPerformanceText(row.NextStep) {
			return fmt.Errorf(
				"performance benchmark %s has placeholder explanation or next step",
				row.Benchmark,
			)
		}
	}
	for benchmark := range requiredBenchmarks {
		if !seenBenchmarks[benchmark] {
			return fmt.Errorf(
				"performance blocker report missing benchmark explanation %s",
				benchmark,
			)
		}
	}
	return nil
}

func validatePerformanceBlockerClaims(claims []string) error {
	if len(claims) == 0 {
		return fmt.Errorf("performance blocker report claim policy notes are required")
	}
	for _, claim := range claims {
		lower := strings.ToLower(claim)
		nonClaim := strings.Contains(lower, "no ") || strings.Contains(lower, "not ") ||
			strings.Contains(lower, "without") ||
			strings.Contains(lower, "does not")
		switch {
		case strings.Contains(lower, "fastest language") && !nonClaim:
			return fmt.Errorf("performance blocker report claims fastest language")
		case strings.Contains(lower, "c++/rust parity") && !nonClaim:
			return fmt.Errorf("performance blocker report claims C++/Rust parity")
		case strings.Contains(lower, "official techempower") && !nonClaim:
			return fmt.Errorf("performance blocker report claims official TechEmpower result")
		case strings.Contains(lower, "official benchmark") && !nonClaim:
			return fmt.Errorf("performance blocker report claims official benchmark result")
		case (strings.Contains(
			lower,
			"measured speed",
		) || strings.Contains(
			lower,
			"speed superiority",
		) || strings.Contains(
			lower,
			"throughput advantage",
		) || strings.Contains(
			lower,
			"latency advantage",
		)) && !nonClaim:
			return fmt.Errorf("performance blocker report claims measured speed comparison")
		case strings.Contains(lower, "runtime behavior change") && !nonClaim:
			return fmt.Errorf("performance blocker report claims runtime behavior change")
		case (strings.Contains(
			lower,
			"zero-cost",
		) || strings.Contains(
			lower,
			"zero cost",
		) || strings.Contains(
			lower,
			"zero_cost",
		)) && strings.Contains(
			lower,
			"dynamic_check_required",
		) && !nonClaim:
			return fmt.Errorf(
				"performance blocker report claims dynamic_check_required as zero-cost",
			)
		case strings.Contains(lower, "unsafe_unknown") && strings.Contains(lower, "trusted") && !nonClaim:
			return fmt.Errorf(
				"performance blocker report claims unsafe_unknown trusted optimization",
			)
		}
	}
	return nil
}

func knownPerformanceCostClass(value string) bool {
	switch value {
	case "zero_cost_proven",
		"dynamic_check_required",
		"instrumentation_only",
		"unsupported_rejected",
		"conservative_fallback":
		return true
	default:
		return false
	}
}

func isWeakPerformanceText(text string) bool {
	text = strings.TrimSpace(strings.ToLower(text))
	return text == "" || text == "todo" || text == "tbd" || strings.Contains(text, "placeholder")
}
