package compiler

import (
	"errors"
	"fmt"
	"strings"

	"tetra_language/compiler/internal/memoryfacts"
)

const (
	MemoryFuzzOracleSchemaV1   = "tetra.memory-fuzz.oracle.v1"
	MemoryFuzzOracleScopeMPC15 = "memory_production_core_v1_mpc15"
)

type MemoryFuzzOracleCategory string

const (
	MemoryFuzzOracleCheckerRejectExpected           MemoryFuzzOracleCategory = "checker_reject_expected"
	MemoryFuzzOracleRuntimeTrapExpected             MemoryFuzzOracleCategory = "runtime_trap_expected"
	MemoryFuzzOracleReferenceOutputExpected         MemoryFuzzOracleCategory = "compiled_output_equals_reference_expected"
	MemoryFuzzOracleCompilerCrashBug                MemoryFuzzOracleCategory = "compiler_crash_is_bug"
	MemoryFuzzOracleMiscompileBug                   MemoryFuzzOracleCategory = "miscompile_is_bug"
	MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug MemoryFuzzOracleCategory = "unsafe_unknown_optimized_as_safe_is_bug"
	MemoryFuzzOracleReportValidationFailureBug      MemoryFuzzOracleCategory = "report_validation_failure_is_bug"
)

type MemoryFuzzOracleResult string

const (
	MemoryFuzzOraclePass MemoryFuzzOracleResult = "pass"
	MemoryFuzzOracleFail MemoryFuzzOracleResult = "fail"
	MemoryFuzzOracleBug  MemoryFuzzOracleResult = "bug"
)

type MemoryFuzzTier string

const (
	MemoryFuzzTier1ShortCI        MemoryFuzzTier = "tier1_short_ci_smoke"
	MemoryFuzzTier2Nightly        MemoryFuzzTier = "tier2_nightly_fuzz"
	MemoryFuzzTier3ReleaseFocused MemoryFuzzTier = "tier3_release_blocking_focused_memory_fuzz"
)

type MemoryFuzzInvariantID string

const (
	MemoryFuzzInvariantNoSafeMetadataMutation          MemoryFuzzInvariantID = "no_safe_metadata_mutation"
	MemoryFuzzInvariantNoBorrowedEscape                MemoryFuzzInvariantID = "no_borrowed_escape"
	MemoryFuzzInvariantNoUnsafeUnknownToSafeKnown      MemoryFuzzInvariantID = "no_unsafe_unknown_to_safe_known"
	MemoryFuzzInvariantNoBoundsRemovalWithoutProofID   MemoryFuzzInvariantID = "no_removed_bounds_check_without_proof_id"
	MemoryFuzzInvariantNoStackRegionStorageWhenEscaped MemoryFuzzInvariantID = "no_stack_region_storage_if_escape_exists"
	MemoryFuzzInvariantReportsValidateAgainstFactGraph MemoryFuzzInvariantID = "reports_validate_against_memory_fact_graph"
	MemoryFuzzInvariantReportsPreserveMemoryCostModel  MemoryFuzzInvariantID = "reports_preserve_memory_cost_model"
)

type MemoryFuzzGeneratorSurfaceTier string

const (
	MemoryFuzzGeneratorTier1SupportedNow         MemoryFuzzGeneratorSurfaceTier = "tier1_supported_now"
	MemoryFuzzGeneratorTier2SupportedNarrow      MemoryFuzzGeneratorSurfaceTier = "tier2_supported_narrow"
	MemoryFuzzGeneratorTier3ConservativeRejected MemoryFuzzGeneratorSurfaceTier = "tier3_conservative_rejected"
	MemoryFuzzGeneratorTier4Future               MemoryFuzzGeneratorSurfaceTier = "tier4_future"
)

type MemoryFuzzObservation struct {
	CheckerRejected              bool
	RuntimeTrapped               bool
	ReferenceCompared            bool
	CompiledExitCode             int
	ReferenceExitCode            int
	CompilerCrashed              bool
	UnsafeUnknownOptimizedAsSafe bool
	ReportValidationFailed       bool
}

type MemoryFuzzOracleReport struct {
	SchemaVersion                        string                          `json:"schema_version"`
	Scope                                string                          `json:"scope"`
	Tier1ShortCISmokeCases               int                             `json:"tier1_short_ci_smoke_cases"`
	Tier2NightlyBoundaryRecorded         bool                            `json:"tier2_nightly_boundary_recorded"`
	Tier3ReleaseBlockingBoundaryRecorded bool                            `json:"tier3_release_blocking_boundary_recorded"`
	Rows                                 []MemoryFuzzOracleRow           `json:"rows"`
	Invariants                           []MemoryFuzzInvariantRow        `json:"invariants"`
	GeneratorSurfaces                    []MemoryFuzzGeneratorSurfaceRow `json:"generator_surfaces"`
	Artifacts                            []MemoryFuzzArtifact            `json:"artifacts"`
	NonClaims                            []string                        `json:"non_claims"`
}

type MemoryFuzzOracleRow struct {
	Category       MemoryFuzzOracleCategory `json:"oracle_category"`
	Name           string                   `json:"name"`
	Tier           MemoryFuzzTier           `json:"tier"`
	ExpectedResult MemoryFuzzOracleResult   `json:"expected_result"`
	Status         string                   `json:"status"`
	Evidence       []string                 `json:"evidence"`
	Tests          []string                 `json:"tests"`
	Boundaries     []string                 `json:"boundaries"`
}

type MemoryFuzzInvariantRow struct {
	ID         MemoryFuzzInvariantID `json:"id"`
	Status     string                `json:"status"`
	Evidence   []string              `json:"evidence"`
	Tests      []string              `json:"tests"`
	Boundaries []string              `json:"boundaries"`
}

type MemoryFuzzGeneratorSurfaceRow struct {
	Tier       MemoryFuzzGeneratorSurfaceTier `json:"tier"`
	Status     string                         `json:"status"`
	Surface    []string                       `json:"surface"`
	Boundaries []string                       `json:"boundaries"`
}

type MemoryFuzzArtifact struct {
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	Required bool   `json:"required"`
}

func ClassifyMemoryFuzzOracleObservation(category MemoryFuzzOracleCategory, obs MemoryFuzzObservation) MemoryFuzzOracleResult {
	switch category {
	case MemoryFuzzOracleCheckerRejectExpected:
		if obs.CompilerCrashed {
			return MemoryFuzzOracleBug
		}
		if obs.CheckerRejected {
			return MemoryFuzzOraclePass
		}
		return MemoryFuzzOracleFail
	case MemoryFuzzOracleRuntimeTrapExpected:
		if obs.CompilerCrashed {
			return MemoryFuzzOracleBug
		}
		if obs.RuntimeTrapped {
			return MemoryFuzzOraclePass
		}
		return MemoryFuzzOracleFail
	case MemoryFuzzOracleReferenceOutputExpected:
		if obs.CompilerCrashed {
			return MemoryFuzzOracleBug
		}
		if !obs.ReferenceCompared {
			return MemoryFuzzOracleFail
		}
		if obs.CompiledExitCode == obs.ReferenceExitCode {
			return MemoryFuzzOraclePass
		}
		return MemoryFuzzOracleBug
	case MemoryFuzzOracleCompilerCrashBug:
		if obs.CompilerCrashed {
			return MemoryFuzzOracleBug
		}
		return MemoryFuzzOraclePass
	case MemoryFuzzOracleMiscompileBug:
		if obs.CompilerCrashed {
			return MemoryFuzzOracleBug
		}
		if obs.ReferenceCompared && obs.CompiledExitCode != obs.ReferenceExitCode {
			return MemoryFuzzOracleBug
		}
		return MemoryFuzzOraclePass
	case MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug:
		if obs.UnsafeUnknownOptimizedAsSafe {
			return MemoryFuzzOracleBug
		}
		return MemoryFuzzOraclePass
	case MemoryFuzzOracleReportValidationFailureBug:
		if obs.ReportValidationFailed {
			return MemoryFuzzOracleBug
		}
		return MemoryFuzzOraclePass
	default:
		return MemoryFuzzOracleFail
	}
}

func BuildMemoryFuzzOracleReport() (MemoryFuzzOracleReport, error) {
	if err := memoryFuzzReportValidationFailureWitness(); err != nil {
		return MemoryFuzzOracleReport{}, err
	}
	return MemoryFuzzOracleReport{
		SchemaVersion:                        MemoryFuzzOracleSchemaV1,
		Scope:                                MemoryFuzzOracleScopeMPC15,
		Tier1ShortCISmokeCases:               7,
		Tier2NightlyBoundaryRecorded:         true,
		Tier3ReleaseBlockingBoundaryRecorded: true,
		Rows: []MemoryFuzzOracleRow{
			memoryFuzzOracleRow(MemoryFuzzOracleCheckerRejectExpected, "Checker reject expected", MemoryFuzzTier1ShortCI, MemoryFuzzOraclePass,
				[]string{"checker reject expected cases cover generated borrow escape, safe metadata mutation, and unsupported unsafe surface diagnostics"},
				[]string{"go test ./compiler/tests/safety ./compiler/tests/semantics -run 'Borrow|Escape|Metadata|Unsafe' -count=1"},
				[]string{"checker reject expected is a passing oracle only when the compiler rejects the generated program with the expected diagnostic"}),
			memoryFuzzOracleRow(MemoryFuzzOracleRuntimeTrapExpected, "Runtime trap expected", MemoryFuzzTier1ShortCI, MemoryFuzzOraclePass,
				[]string{"runtime trap expected cases reuse memory-production-smoke bounds diagnostics for slice bounds, ptr_add bounds, and raw-slice length overflow"},
				[]string{"go test ./tools/cmd/memory-production-smoke -run 'RuntimeDiagnostic|Raw|Bounds' -count=1"},
				[]string{"runtime trap expected is limited to normal-build checks that remain in the generated executable"}),
			memoryFuzzOracleRow(MemoryFuzzOracleReferenceOutputExpected, "Compiled output equals interpreter/reference expected", MemoryFuzzTier1ShortCI, MemoryFuzzOraclePass,
				[]string{"compiled output equals interpreter/reference expected is backed by differential.CheckBackendMatrix source interpreter lanes for deterministic samples"},
				[]string{"go test ./compiler/internal/differential -run 'CheckBackendMatrix' -count=1"},
				[]string{"reference equality is bounded to supported deterministic samples and is not a full source interpreter claim"}),
			memoryFuzzOracleRow(MemoryFuzzOracleCompilerCrashBug, "Compiler crash is bug", MemoryFuzzTier1ShortCI, MemoryFuzzOracleBug,
				[]string{"compiler crash is bug: generated parser/checker/lowering fuzz entries must return diagnostics or valid artifacts; panic/crash is never a passing oracle"},
				[]string{"go test ./compiler/tests/fuzz -run 'FuzzLoweringPipelineVerifiesIR|FuzzFormatSourceIdempotent' -count=1"},
				[]string{"crash classification records a bug and does not promote the generated program as passing evidence"}),
			memoryFuzzOracleRow(MemoryFuzzOracleMiscompileBug, "Miscompile is bug", MemoryFuzzTier2Nightly, MemoryFuzzOracleBug,
				[]string{"miscompile is bug: differential mismatch between compiled output and source/interpreter reference is reduced to a reproducer"},
				[]string{"go test ./compiler/internal/differential -run 'Reducer|CheckBackendMatrix' -count=1"},
				[]string{"miscompile classification is a failure artifact, not performance or correctness proof"}),
			memoryFuzzOracleRow(MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug, "unsafe_unknown optimized as safe is bug", MemoryFuzzTier1ShortCI, MemoryFuzzOracleBug,
				[]string{"unsafe_unknown optimized as safe is bug: memoryfacts rejects unsafe_unknown -> safe_known, no_alias, bounds_check_eliminated, and trusted storage claims"},
				[]string{"go test ./compiler/internal/memoryfacts -run 'UnsafeUnknown|SafeKnown|Optimization|TrustedStorage' -count=1"},
				[]string{"unsafe_unknown may stay checked, trapped, or conservative, but never becomes safe_known"}),
			memoryFuzzOracleRow(MemoryFuzzOracleReportValidationFailureBug, "Report validation failure is bug", MemoryFuzzTier1ShortCI, MemoryFuzzOracleBug,
				[]string{"report validation failure is bug: MemoryFactGraph validation rejects invalid memory reports before artifact emission"},
				[]string{"go test ./compiler/internal/memoryfacts ./tools/cmd/validate-memory-report -run 'ValidateMemoryReport|Cost|Unsafe' -count=1"},
				[]string{"reports validate against MemoryFactGraph and the MPC-14 cost model rather than report-reconstructed truth"}),
		},
		Invariants: []MemoryFuzzInvariantRow{
			memoryFuzzInvariantRow(MemoryFuzzInvariantNoSafeMetadataMutation, "safe representation metadata is not user-assignable", "go test ./compiler/tests/semantics -run 'Metadata' -count=1"),
			memoryFuzzInvariantRow(MemoryFuzzInvariantNoBorrowedEscape, "borrowed values cannot escape return/actor/task boundaries without checked copy/transfer", "go test ./compiler/tests/safety ./compiler/tests/ownership -run 'Borrow|Escape|Actor|Task' -count=1"),
			memoryFuzzInvariantRow(MemoryFuzzInvariantNoUnsafeUnknownToSafeKnown, "unsafe_unknown rows cannot become safe_known or safe_borrowed proof rows", "go test ./compiler/internal/memoryfacts -run 'UnsafeUnknown|SafeKnown|SafeBorrowed' -count=1"),
			memoryFuzzInvariantRow(MemoryFuzzInvariantNoBoundsRemovalWithoutProofID, "bounds check removal requires compiler-owned proof id and validated report evidence", "go test ./compiler/internal/memoryfacts ./compiler -run 'Bounds|Proof|MemoryReport' -count=1"),
			memoryFuzzInvariantRow(MemoryFuzzInvariantNoStackRegionStorageWhenEscaped, "stack/region storage claims are rejected when escape evidence forces heap or conservative fallback", "go test ./compiler/internal/memoryfacts ./compiler/internal/validation -run 'Storage|Escape|Region|HeapFallback' -count=1"),
			memoryFuzzInvariantRow(MemoryFuzzInvariantReportsValidateAgainstFactGraph, "memory reports validate against MemoryFactGraph during compiler report emission and CLI validation", "go test ./compiler/internal/memoryfacts ./tools/cmd/validate-memory-report -count=1"),
			memoryFuzzInvariantRow(MemoryFuzzInvariantReportsPreserveMemoryCostModel, "memory report rows preserve cost_class and normal_build_check rules from the MPC-14 cost model", "go test ./compiler/internal/memoryfacts ./tools/cmd/validate-memory-report -run 'Cost|Dynamic|Unsafe' -count=1"),
		},
		GeneratorSurfaces: []MemoryFuzzGeneratorSurfaceRow{
			{Tier: MemoryFuzzGeneratorTier1SupportedNow, Status: "covered", Surface: []string{"slices", "Strings", "borrow/copy", "simple structs/enums/optionals", "safe views", "make_*", "explicit islands"}, Boundaries: []string{"Tier 1 short CI smoke uses deterministic generated samples only"}},
			{Tier: MemoryFuzzGeneratorTier2SupportedNarrow, Status: "boundary_recorded", Surface: []string{"generics", "function-typed borrowed returns", "async/task boundary smoke", "raw verified roots"}, Boundaries: []string{"Tier 2 nightly fuzz may expand these narrow supported surfaces with bounded seeds"}},
			{Tier: MemoryFuzzGeneratorTier3ConservativeRejected, Status: "boundary_recorded", Surface: []string{"arbitrary unsafe pointers", "unknown external calls", "unsupported target behavior"}, Boundaries: []string{"Tier 3 records conservative/rejected outcomes instead of upgrading safety claims"}},
			{Tier: MemoryFuzzGeneratorTier4Future, Status: "future", Surface: []string{"full FFI lifetime", "full actor zero-copy runtime", "generic lifetimes"}, Boundaries: []string{"future-only scope is not a current production claim"}},
		},
		Artifacts: []MemoryFuzzArtifact{
			{Path: "reports/memory-fuzz-short/<slice>/memory-fuzz-oracle.json", Kind: "tier1_short_ci_smoke_report", Required: true},
			{Path: "reports/memory-fuzz-short/<slice>/summary.md", Kind: "tier1_short_ci_smoke_summary", Required: true},
			{Path: "docs/audits/memory-fuzz-oracle-v1.md", Kind: "audit_contract", Required: true},
		},
		NonClaims: []string{
			"no exhaustive fuzzing is claimed",
			"no unsupported unsafe pointer safety is claimed",
			"no full program correctness claim is made",
			"no runtime behavior change",
			"no safe-program semantics change",
			"no performance claim is made",
		},
	}, nil
}

func memoryFuzzOracleRow(category MemoryFuzzOracleCategory, name string, tier MemoryFuzzTier, result MemoryFuzzOracleResult, evidence []string, tests []string, boundaries []string) MemoryFuzzOracleRow {
	return MemoryFuzzOracleRow{
		Category:       category,
		Name:           name,
		Tier:           tier,
		ExpectedResult: result,
		Status:         "covered",
		Evidence:       evidence,
		Tests:          tests,
		Boundaries:     boundaries,
	}
}

func memoryFuzzInvariantRow(id MemoryFuzzInvariantID, evidence string, test string) MemoryFuzzInvariantRow {
	return MemoryFuzzInvariantRow{
		ID:         id,
		Status:     "covered",
		Evidence:   []string{evidence},
		Tests:      []string{test},
		Boundaries: []string{"checked for generated programs before a fuzz result is promoted as passing evidence"},
	}
}

func ValidateMemoryFuzzOracleReport(report MemoryFuzzOracleReport) error {
	var issues []string
	if report.SchemaVersion != MemoryFuzzOracleSchemaV1 {
		issues = append(issues, fmt.Sprintf("schema_version = %q, want %q", report.SchemaVersion, MemoryFuzzOracleSchemaV1))
	}
	if report.Scope != MemoryFuzzOracleScopeMPC15 {
		issues = append(issues, fmt.Sprintf("scope = %q, want %q", report.Scope, MemoryFuzzOracleScopeMPC15))
	}
	if report.Tier1ShortCISmokeCases <= 0 {
		issues = append(issues, "Tier 1 short CI smoke cases are required")
	}
	if !report.Tier2NightlyBoundaryRecorded {
		issues = append(issues, "Tier 2 nightly fuzz boundary is required")
	}
	if !report.Tier3ReleaseBlockingBoundaryRecorded {
		issues = append(issues, "Tier 3 release-blocking focused memory fuzz boundary is required")
	}
	issues = append(issues, validateMemoryFuzzOracleRows(report.Rows)...)
	issues = append(issues, validateMemoryFuzzInvariants(report.Invariants)...)
	issues = append(issues, validateMemoryFuzzGeneratorSurfaces(report.GeneratorSurfaces)...)
	if len(report.Artifacts) == 0 {
		issues = append(issues, "memory fuzz oracle artifacts are required")
	}
	for _, want := range []string{"no exhaustive fuzzing is claimed", "no unsupported unsafe pointer safety is claimed", "no runtime behavior change", "no safe-program semantics change"} {
		if !memoryFuzzHasString(report.NonClaims, want) {
			issues = append(issues, fmt.Sprintf("missing non-claim %q", want))
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateMemoryFuzzOracleRows(rows []MemoryFuzzOracleRow) []string {
	seen := map[MemoryFuzzOracleCategory]bool{}
	var issues []string
	for _, row := range rows {
		if row.Category == "" {
			issues = append(issues, "oracle_category is required")
			continue
		}
		if !knownMemoryFuzzOracleCategory(row.Category) {
			issues = append(issues, fmt.Sprintf("unknown oracle_category %q", row.Category))
		}
		if seen[row.Category] {
			issues = append(issues, fmt.Sprintf("duplicate oracle_category %s", row.Category))
		}
		seen[row.Category] = true
		if strings.TrimSpace(row.Name) == "" {
			issues = append(issues, fmt.Sprintf("oracle_category %s name is required", row.Category))
		}
		if !knownMemoryFuzzTier(row.Tier) {
			issues = append(issues, fmt.Sprintf("oracle_category %s unknown tier %q", row.Category, row.Tier))
		}
		if row.Status != "covered" {
			issues = append(issues, fmt.Sprintf("oracle_category %s status = %q, want covered", row.Category, row.Status))
		}
		if row.ExpectedResult != expectedMemoryFuzzOracleResult(row.Category) {
			issues = append(issues, fmt.Sprintf("oracle_category %s expected_result = %q, want %q", row.Category, row.ExpectedResult, expectedMemoryFuzzOracleResult(row.Category)))
		}
		issues = append(issues, validateMemoryFuzzTextList("oracle_category "+string(row.Category)+" evidence", row.Evidence)...)
		issues = append(issues, validateMemoryFuzzTextList("oracle_category "+string(row.Category)+" tests", row.Tests)...)
		issues = append(issues, validateMemoryFuzzTextList("oracle_category "+string(row.Category)+" boundaries", row.Boundaries)...)
	}
	for _, category := range memoryFuzzOracleCategories() {
		if !seen[category] {
			issues = append(issues, fmt.Sprintf("missing oracle_category %s", category))
		}
	}
	return issues
}

func validateMemoryFuzzInvariants(rows []MemoryFuzzInvariantRow) []string {
	seen := map[MemoryFuzzInvariantID]bool{}
	var issues []string
	for _, row := range rows {
		if !knownMemoryFuzzInvariantID(row.ID) {
			issues = append(issues, fmt.Sprintf("unknown invariant %q", row.ID))
		}
		if seen[row.ID] {
			issues = append(issues, fmt.Sprintf("duplicate invariant %s", row.ID))
		}
		seen[row.ID] = true
		if row.Status != "covered" {
			issues = append(issues, fmt.Sprintf("invariant %s status = %q, want covered", row.ID, row.Status))
		}
		issues = append(issues, validateMemoryFuzzTextList("invariant "+string(row.ID)+" evidence", row.Evidence)...)
		issues = append(issues, validateMemoryFuzzTextList("invariant "+string(row.ID)+" tests", row.Tests)...)
		issues = append(issues, validateMemoryFuzzTextList("invariant "+string(row.ID)+" boundaries", row.Boundaries)...)
	}
	for _, id := range memoryFuzzInvariantIDs() {
		if !seen[id] {
			issues = append(issues, fmt.Sprintf("missing invariant %s", id))
		}
	}
	return issues
}

func validateMemoryFuzzGeneratorSurfaces(rows []MemoryFuzzGeneratorSurfaceRow) []string {
	seen := map[MemoryFuzzGeneratorSurfaceTier]bool{}
	var issues []string
	for _, row := range rows {
		if !knownMemoryFuzzGeneratorSurfaceTier(row.Tier) {
			issues = append(issues, fmt.Sprintf("unknown generator surface tier %q", row.Tier))
		}
		if seen[row.Tier] {
			issues = append(issues, fmt.Sprintf("duplicate generator surface tier %s", row.Tier))
		}
		seen[row.Tier] = true
		if strings.TrimSpace(row.Status) == "" {
			issues = append(issues, fmt.Sprintf("generator surface tier %s status is required", row.Tier))
		}
		issues = append(issues, validateMemoryFuzzTextList("generator surface tier "+string(row.Tier)+" surface", row.Surface)...)
		issues = append(issues, validateMemoryFuzzTextList("generator surface tier "+string(row.Tier)+" boundaries", row.Boundaries)...)
	}
	for _, tier := range memoryFuzzGeneratorSurfaceTiers() {
		if !seen[tier] {
			issues = append(issues, fmt.Sprintf("missing generator surface tier %s", tier))
		}
	}
	return issues
}

func validateMemoryFuzzTextList(label string, values []string) []string {
	if len(values) == 0 {
		return []string{label + " is required"}
	}
	var issues []string
	for _, value := range values {
		text := strings.TrimSpace(value)
		if text == "" {
			issues = append(issues, label+" contains empty text")
			continue
		}
		lower := strings.ToLower(text)
		for _, forbidden := range []string{"todo", "placeholder", " fake", " mock"} {
			if strings.Contains(lower, forbidden) {
				issues = append(issues, fmt.Sprintf("%s contains forbidden placeholder marker %q", label, strings.TrimSpace(forbidden)))
			}
		}
	}
	return issues
}

func memoryFuzzReportValidationFailureWitness() error {
	report := memoryfacts.Report{
		SchemaVersion: memoryfacts.ReportSchemaV1,
		Rows: []memoryfacts.ReportRow{{
			ProgramID:       "memory-fuzz-oracle",
			FunctionID:      "main",
			SiteID:          "unsafe:oracle",
			SourceFactID:    "memory-fuzz:unsafe-unknown",
			SourceStage:     memoryfacts.StagePLIR,
			Claim:           "unsafe_unknown became safe_known",
			ClaimLevel:      memoryfacts.ClaimConservative,
			ProvenanceClass: memoryfacts.ProvenanceSafeKnown,
			UnsafeClass:     memoryfacts.UnsafeUnknown,
			ValidatorStatus: memoryfacts.ValidatorNotApplicable,
			CostClass:       memoryfacts.CostConservativeFallback,
			Reason:          "fixture must be rejected so report validation failure remains a bug oracle",
		}},
	}
	err := memoryfacts.ValidateReport(report)
	if err == nil {
		return fmt.Errorf("memory fuzz oracle witness expected MemoryFactGraph validation failure for unsafe_unknown -> safe_known")
	}
	return nil
}

func memoryFuzzOracleCategories() []MemoryFuzzOracleCategory {
	return []MemoryFuzzOracleCategory{
		MemoryFuzzOracleCheckerRejectExpected,
		MemoryFuzzOracleRuntimeTrapExpected,
		MemoryFuzzOracleReferenceOutputExpected,
		MemoryFuzzOracleCompilerCrashBug,
		MemoryFuzzOracleMiscompileBug,
		MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug,
		MemoryFuzzOracleReportValidationFailureBug,
	}
}

func memoryFuzzInvariantIDs() []MemoryFuzzInvariantID {
	return []MemoryFuzzInvariantID{
		MemoryFuzzInvariantNoSafeMetadataMutation,
		MemoryFuzzInvariantNoBorrowedEscape,
		MemoryFuzzInvariantNoUnsafeUnknownToSafeKnown,
		MemoryFuzzInvariantNoBoundsRemovalWithoutProofID,
		MemoryFuzzInvariantNoStackRegionStorageWhenEscaped,
		MemoryFuzzInvariantReportsValidateAgainstFactGraph,
		MemoryFuzzInvariantReportsPreserveMemoryCostModel,
	}
}

func memoryFuzzGeneratorSurfaceTiers() []MemoryFuzzGeneratorSurfaceTier {
	return []MemoryFuzzGeneratorSurfaceTier{
		MemoryFuzzGeneratorTier1SupportedNow,
		MemoryFuzzGeneratorTier2SupportedNarrow,
		MemoryFuzzGeneratorTier3ConservativeRejected,
		MemoryFuzzGeneratorTier4Future,
	}
}

func knownMemoryFuzzOracleCategory(category MemoryFuzzOracleCategory) bool {
	for _, known := range memoryFuzzOracleCategories() {
		if category == known {
			return true
		}
	}
	return false
}

func knownMemoryFuzzTier(tier MemoryFuzzTier) bool {
	switch tier {
	case MemoryFuzzTier1ShortCI, MemoryFuzzTier2Nightly, MemoryFuzzTier3ReleaseFocused:
		return true
	default:
		return false
	}
}

func knownMemoryFuzzInvariantID(id MemoryFuzzInvariantID) bool {
	for _, known := range memoryFuzzInvariantIDs() {
		if id == known {
			return true
		}
	}
	return false
}

func knownMemoryFuzzGeneratorSurfaceTier(tier MemoryFuzzGeneratorSurfaceTier) bool {
	for _, known := range memoryFuzzGeneratorSurfaceTiers() {
		if tier == known {
			return true
		}
	}
	return false
}

func expectedMemoryFuzzOracleResult(category MemoryFuzzOracleCategory) MemoryFuzzOracleResult {
	switch category {
	case MemoryFuzzOracleCheckerRejectExpected, MemoryFuzzOracleRuntimeTrapExpected, MemoryFuzzOracleReferenceOutputExpected:
		return MemoryFuzzOraclePass
	case MemoryFuzzOracleCompilerCrashBug, MemoryFuzzOracleMiscompileBug, MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug, MemoryFuzzOracleReportValidationFailureBug:
		return MemoryFuzzOracleBug
	default:
		return ""
	}
}

func (r *MemoryFuzzOracleReport) RowsByCategory(category MemoryFuzzOracleCategory) *MemoryFuzzOracleRow {
	for i := range r.Rows {
		if r.Rows[i].Category == category {
			return &r.Rows[i]
		}
	}
	return &MemoryFuzzOracleRow{}
}

func cloneMemoryFuzzOracleReport(in MemoryFuzzOracleReport) MemoryFuzzOracleReport {
	out := in
	out.Rows = append([]MemoryFuzzOracleRow(nil), in.Rows...)
	out.Invariants = append([]MemoryFuzzInvariantRow(nil), in.Invariants...)
	out.GeneratorSurfaces = append([]MemoryFuzzGeneratorSurfaceRow(nil), in.GeneratorSurfaces...)
	out.Artifacts = append([]MemoryFuzzArtifact(nil), in.Artifacts...)
	out.NonClaims = append([]string(nil), in.NonClaims...)
	return out
}

func memoryFuzzHasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
