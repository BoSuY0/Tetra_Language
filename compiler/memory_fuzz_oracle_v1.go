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

type MemoryFuzzRequirementID string

const (
	MemoryFuzzRequirementTier1V0V11Coverage         MemoryFuzzRequirementID = "MEM-FUZZ-001"
	MemoryFuzzRequirementCrashMiscompileArtifacts   MemoryFuzzRequirementID = "MEM-FUZZ-002"
	MemoryFuzzRequirementBlockingMemoryFailures     MemoryFuzzRequirementID = "MEM-FUZZ-003"
	MemoryFuzzRequirementTier2NightlySeedTriage     MemoryFuzzRequirementID = "MEM-FUZZ-004"
	MemoryFuzzRequirementTier3ReleasePassOrClassify MemoryFuzzRequirementID = "MEM-FUZZ-005"
)

type MemoryFuzzBlockingCaseID string

const (
	MemoryFuzzBlockingUnsafeUnknownOptimizedAsSafe MemoryFuzzBlockingCaseID = "unsafe_unknown_optimized_as_safe"
	MemoryFuzzBlockingBoundsCheckWithoutProofID    MemoryFuzzBlockingCaseID = "bounds_check_eliminated_without_proof_id"
	MemoryFuzzBlockingTrustedStorageUnderEscape    MemoryFuzzBlockingCaseID = "trusted_storage_under_escape"
	MemoryFuzzBlockingReportValidationFailure      MemoryFuzzBlockingCaseID = "report_validation_failure"
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
	Requirements                         []MemoryFuzzRequirementRow      `json:"requirements"`
	SliceCoverage                        []MemoryFuzzSliceCoverageRow    `json:"slice_coverage"`
	Rows                                 []MemoryFuzzOracleRow           `json:"rows"`
	Invariants                           []MemoryFuzzInvariantRow        `json:"invariants"`
	GeneratorSurfaces                    []MemoryFuzzGeneratorSurfaceRow `json:"generator_surfaces"`
	BlockingCases                        []MemoryFuzzBlockingCaseRow     `json:"blocking_cases"`
	TierPolicies                         []MemoryFuzzTierPolicyRow       `json:"tier_policies"`
	Artifacts                            []MemoryFuzzArtifact            `json:"artifacts"`
	NonClaims                            []string                        `json:"non_claims"`
}

type MemoryFuzzRequirementRow struct {
	ID         MemoryFuzzRequirementID `json:"id"`
	Status     string                  `json:"status"`
	Evidence   []string                `json:"evidence"`
	Tests      []string                `json:"tests"`
	Boundaries []string                `json:"boundaries"`
}

type MemoryFuzzSliceCoverageRow struct {
	SliceID          string                     `json:"slice_id"`
	Status           string                     `json:"status"`
	Surface          []string                   `json:"surface"`
	OracleCategories []MemoryFuzzOracleCategory `json:"oracle_categories"`
	Invariants       []MemoryFuzzInvariantID    `json:"invariants"`
	Evidence         []string                   `json:"evidence"`
	Tests            []string                   `json:"tests"`
	Boundaries       []string                   `json:"boundaries"`
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

type MemoryFuzzBlockingCaseRow struct {
	ID            MemoryFuzzBlockingCaseID `json:"id"`
	Status        string                   `json:"status"`
	BlocksRelease bool                     `json:"blocks_release"`
	Evidence      []string                 `json:"evidence"`
	Tests         []string                 `json:"tests"`
	Boundaries    []string                 `json:"boundaries"`
}

type MemoryFuzzTierPolicyRow struct {
	Tier                                   MemoryFuzzTier `json:"tier"`
	Status                                 string         `json:"status"`
	SeedsPreserved                         bool           `json:"seeds_preserved"`
	UnstableTriageRequired                 bool           `json:"unstable_triage_required"`
	MinimizedReproducerRequired            bool           `json:"minimized_reproducer_required"`
	ReleasePromotionBlockedUntilClassified bool           `json:"release_promotion_blocked_until_classified"`
	Evidence                               []string       `json:"evidence"`
	Tests                                  []string       `json:"tests"`
	Boundaries                             []string       `json:"boundaries"`
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
		Tier1ShortCISmokeCases:               12,
		Tier2NightlyBoundaryRecorded:         true,
		Tier3ReleaseBlockingBoundaryRecorded: true,
		Requirements: []MemoryFuzzRequirementRow{
			memoryFuzzRequirementRow(MemoryFuzzRequirementTier1V0V11Coverage, "validated_narrow",
				"Tier 1 short CI smoke covers deterministic v0-v11 memory oracle cases across the supported compiler-visible memory surfaces",
				"go test ./tools/cmd/memory-fuzz-short ./tools/cmd/validate-memory-fuzz-oracle -count=1",
				"Tier 1 is deterministic short smoke evidence, not exhaustive fuzz proof"),
			memoryFuzzRequirementRow(MemoryFuzzRequirementCrashMiscompileArtifacts, "validated_narrow",
				"compiler crash and miscompile classifications require reducer or reproducer artifact slots before evidence promotion",
				"go test ./compiler -run 'MemoryFuzzOracle.*V12|ValidateMemoryFuzzOracleReportRejectsV12' -count=1",
				"artifact discipline is release evidence only and does not claim full program correctness"),
			memoryFuzzRequirementRow(MemoryFuzzRequirementBlockingMemoryFailures, "release_blocking",
				"unsafe_unknown optimized as safe, missing bounds proof id, trusted storage under escape, and report validation failure block release promotion",
				"go test ./compiler/internal/memoryfacts ./tools/cmd/validate-memory-report -run 'Unsafe|Bounds|Storage|Validate' -count=1",
				"blocking cases preserve MemoryFactGraph truth and do not replace validators"),
			memoryFuzzRequirementRow(MemoryFuzzRequirementTier2NightlySeedTriage, "boundary_recorded",
				"Tier 2 nightly fuzz preserves seeds, unstable triage, and minimized repro expectations",
				"bash scripts/dev/fuzz-nightly.sh --short --out-dir reports/fuzz-nightly-smoke",
				"Tier 2 is longer/nightly boundary evidence, not mandatory Tier 1 evidence"),
			memoryFuzzRequirementRow(MemoryFuzzRequirementTier3ReleasePassOrClassify, "release_blocking",
				"Tier 3 release-blocking focused memory fuzz must pass or classify every failure before release promotion",
				"go run ./tools/cmd/validate-memory-fuzz-oracle --report reports/memory-fuzz-short/v12/memory-fuzz-oracle.json",
				"Tier 3 blocks release promotion on unclassified failures without claiming target parity"),
		},
		SliceCoverage: memoryFuzzSliceCoverageRows(),
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
		BlockingCases: []MemoryFuzzBlockingCaseRow{
			memoryFuzzBlockingCaseRow(MemoryFuzzBlockingUnsafeUnknownOptimizedAsSafe,
				"unsafe_unknown optimized as safe remains a release-blocking oracle bug",
				"go test ./compiler/internal/memoryfacts -run 'UnsafeUnknown|SafeKnown|Optimization' -count=1",
				"unsafe_unknown may remain checked, trapped, or conservative, but never becomes safe_known"),
			memoryFuzzBlockingCaseRow(MemoryFuzzBlockingBoundsCheckWithoutProofID,
				"bounds_check_eliminated without a compiler-owned proof id remains release-blocking",
				"go test ./compiler/internal/memoryfacts ./compiler -run 'Bounds|Proof|MemoryReport' -count=1",
				"bounds removal evidence must preserve proof ids from the compiler-owned graph"),
			memoryFuzzBlockingCaseRow(MemoryFuzzBlockingTrustedStorageUnderEscape,
				"stack, region, or trusted storage under escape remains release-blocking",
				"go test ./compiler/internal/memoryfacts ./compiler/internal/validation -run 'Storage|Escape|Region|HeapFallback' -count=1",
				"escaped values require heap, conservative, or rejected classification"),
			memoryFuzzBlockingCaseRow(MemoryFuzzBlockingReportValidationFailure,
				"memory report validation failure remains release-blocking before artifact promotion",
				"go test ./compiler/internal/memoryfacts ./tools/cmd/validate-memory-report -run 'ValidateMemoryReport|Cost|Unsafe' -count=1",
				"report rows are projections and cannot reconstruct MemoryFactGraph truth"),
		},
		TierPolicies: []MemoryFuzzTierPolicyRow{
			{
				Tier:           MemoryFuzzTier1ShortCI,
				Status:         "covered",
				SeedsPreserved: true,
				Evidence:       []string{"Tier 1 uses deterministic v0-v11 smoke cases and writes release-evidence artifacts"},
				Tests:          []string{"go run ./tools/cmd/memory-fuzz-short --tier=1 --report-dir reports/memory-fuzz-short/v12"},
				Boundaries:     []string{"Tier 1 is short deterministic smoke, not nightly fuzz or exhaustive proof"},
			},
			{
				Tier:                        MemoryFuzzTier2Nightly,
				Status:                      "boundary_recorded",
				SeedsPreserved:              true,
				UnstableTriageRequired:      true,
				MinimizedReproducerRequired: true,
				Evidence:                    []string{"Tier 2 nightly fuzz preserves seeds, unstable triage, and minimized repros using the fuzz property stress protocol"},
				Tests:                       []string{"bash scripts/dev/fuzz-nightly.sh --short --out-dir reports/fuzz-nightly-smoke"},
				Boundaries:                  []string{"Tier 2 is nightly/release-candidate evidence and is not required as deterministic Tier 1"},
			},
			{
				Tier:                                   MemoryFuzzTier3ReleaseFocused,
				Status:                                 "release_blocking",
				SeedsPreserved:                         true,
				UnstableTriageRequired:                 true,
				MinimizedReproducerRequired:            true,
				ReleasePromotionBlockedUntilClassified: true,
				Evidence:                               []string{"Tier 3 focused memory fuzz must pass or classify every failure before release promotion"},
				Tests:                                  []string{"go run ./tools/cmd/validate-memory-fuzz-oracle --report reports/memory-fuzz-short/v12/memory-fuzz-oracle.json"},
				Boundaries:                             []string{"Tier 3 release blocking is classification evidence, not target parity or runtime ABI proof"},
			},
		},
		Artifacts: []MemoryFuzzArtifact{
			{Path: "reports/memory-fuzz-short/<slice>/memory-fuzz-oracle.json", Kind: "tier1_short_ci_smoke_report", Required: true},
			{Path: "reports/memory-fuzz-short/<slice>/summary.md", Kind: "tier1_short_ci_smoke_summary", Required: true},
			{Path: "reports/memory-fuzz-short/<slice>/reproducers/compiler-crash/", Kind: "compiler_crash_reproducer", Required: true},
			{Path: "reports/memory-fuzz-short/<slice>/reproducers/miscompile/", Kind: "miscompile_reproducer", Required: true},
			{Path: "reports/memory-fuzz-short/<slice>/reducers/miscompile/", Kind: "miscompile_reducer", Required: true},
			{Path: "docs/audits/memory-fuzz-oracle-v1.md", Kind: "audit_contract", Required: true},
		},
		NonClaims: []string{
			"no exhaustive fuzzing is claimed",
			"no exhaustive fuzz proof is claimed",
			"no unsupported unsafe pointer safety is claimed",
			"no arbitrary unsafe safety is claimed",
			"no full runtime/ABI/target parity proof is claimed",
			"no full program correctness claim is made",
			"no runtime behavior change",
			"no safe-program semantics change",
			"no performance claim is made",
			"no clean-release claim under dirty worktree",
			"no replacement for MemoryFactGraph validators",
			"no Memory 100% claim is made",
		},
	}, nil
}

func memoryFuzzRequirementRow(id MemoryFuzzRequirementID, status string, evidence string, test string, boundary string) MemoryFuzzRequirementRow {
	return MemoryFuzzRequirementRow{
		ID:         id,
		Status:     status,
		Evidence:   []string{evidence},
		Tests:      []string{test},
		Boundaries: []string{boundary},
	}
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

func memoryFuzzSliceCoverageRows() []MemoryFuzzSliceCoverageRow {
	return []MemoryFuzzSliceCoverageRow{
		memoryFuzzSliceCoverageRow("v0", []string{"metadata", "borrow", "narrow noalias"},
			[]MemoryFuzzOracleCategory{MemoryFuzzOracleCheckerRejectExpected, MemoryFuzzOracleReferenceOutputExpected},
			[]MemoryFuzzInvariantID{MemoryFuzzInvariantNoSafeMetadataMutation, MemoryFuzzInvariantNoBorrowedEscape, MemoryFuzzInvariantReportsValidateAgainstFactGraph},
			"go test ./compiler/internal/memoryfacts -run 'MemoryIdealV0|NoAlias|Metadata|Borrow' -count=1"),
		memoryFuzzSliceCoverageRow("v1", []string{"enum payload borrow carriers", "generic wrapper borrow carriers"},
			[]MemoryFuzzOracleCategory{MemoryFuzzOracleCheckerRejectExpected, MemoryFuzzOracleReferenceOutputExpected},
			[]MemoryFuzzInvariantID{MemoryFuzzInvariantNoBorrowedEscape, MemoryFuzzInvariantReportsValidateAgainstFactGraph},
			"go test ./compiler/internal/memoryfacts -run 'MemoryIdealV1|Enum|Generic|Borrow' -count=1"),
		memoryFuzzSliceCoverageRow("v2", []string{"callbacks", "function values", "borrowed callable returns"},
			[]MemoryFuzzOracleCategory{MemoryFuzzOracleCheckerRejectExpected, MemoryFuzzOracleReferenceOutputExpected},
			[]MemoryFuzzInvariantID{MemoryFuzzInvariantNoBorrowedEscape, MemoryFuzzInvariantReportsValidateAgainstFactGraph},
			"go test ./compiler/internal/memoryfacts -run 'MemoryIdealV2|Callback|Function' -count=1"),
		memoryFuzzSliceCoverageRow("v3", []string{"protocol dispatch", "interface borrow carriers", "dynamic dispatch conservatism"},
			[]MemoryFuzzOracleCategory{MemoryFuzzOracleCheckerRejectExpected, MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug},
			[]MemoryFuzzInvariantID{MemoryFuzzInvariantNoBorrowedEscape, MemoryFuzzInvariantNoUnsafeUnknownToSafeKnown},
			"go test ./compiler/internal/memoryfacts -run 'MemoryIdealV3|Protocol|Interface|Dynamic' -count=1"),
		memoryFuzzSliceCoverageRow("v4", []string{"async boundary", "task boundary", "actor boundary"},
			[]MemoryFuzzOracleCategory{MemoryFuzzOracleCheckerRejectExpected, MemoryFuzzOracleReportValidationFailureBug},
			[]MemoryFuzzInvariantID{MemoryFuzzInvariantNoBorrowedEscape, MemoryFuzzInvariantReportsValidateAgainstFactGraph},
			"go test ./compiler/internal/memoryfacts -run 'MemoryIdealV4|Async|Task|Actor' -count=1"),
		memoryFuzzSliceCoverageRow("v5", []string{"unsafe gateway", "raw pointer conservatism", "unsafe_unknown provenance"},
			[]MemoryFuzzOracleCategory{MemoryFuzzOracleCheckerRejectExpected, MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug},
			[]MemoryFuzzInvariantID{MemoryFuzzInvariantNoUnsafeUnknownToSafeKnown, MemoryFuzzInvariantReportsValidateAgainstFactGraph},
			"go test ./compiler/internal/memoryfacts -run 'MemoryIdealV5|Unsafe|Raw|Pointer' -count=1"),
		memoryFuzzSliceCoverageRow("v6", []string{"bounds proof ids", "bounds check elimination rejection", "proof source ids"},
			[]MemoryFuzzOracleCategory{MemoryFuzzOracleRuntimeTrapExpected, MemoryFuzzOracleReportValidationFailureBug},
			[]MemoryFuzzInvariantID{MemoryFuzzInvariantNoBoundsRemovalWithoutProofID, MemoryFuzzInvariantReportsValidateAgainstFactGraph},
			"go test ./compiler/internal/memoryfacts ./compiler -run 'MemoryIdealV6|Bounds|Proof' -count=1"),
		memoryFuzzSliceCoverageRow("v7", []string{"FFI quarantine", "external call provenance", "raw verified roots"},
			[]MemoryFuzzOracleCategory{MemoryFuzzOracleCheckerRejectExpected, MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug},
			[]MemoryFuzzInvariantID{MemoryFuzzInvariantNoUnsafeUnknownToSafeKnown, MemoryFuzzInvariantReportsValidateAgainstFactGraph},
			"go test ./compiler/internal/memoryfacts -run 'MemoryIdealV7|FFI|External|Raw' -count=1"),
		memoryFuzzSliceCoverageRow("v8", []string{"memory report integrity", "cost model projection", "normal build checks"},
			[]MemoryFuzzOracleCategory{MemoryFuzzOracleReportValidationFailureBug},
			[]MemoryFuzzInvariantID{MemoryFuzzInvariantReportsValidateAgainstFactGraph, MemoryFuzzInvariantReportsPreserveMemoryCostModel},
			"go test ./compiler/internal/memoryfacts ./tools/cmd/validate-memory-report -run 'Report|Cost|NormalBuild' -count=1"),
		memoryFuzzSliceCoverageRow("v9", []string{"storage lowering", "escape-aware heap fallback", "trusted storage rejection"},
			[]MemoryFuzzOracleCategory{MemoryFuzzOracleCheckerRejectExpected, MemoryFuzzOracleReportValidationFailureBug},
			[]MemoryFuzzInvariantID{MemoryFuzzInvariantNoStackRegionStorageWhenEscaped, MemoryFuzzInvariantReportsValidateAgainstFactGraph},
			"go test ./compiler/internal/memoryfacts ./compiler/internal/validation -run 'Storage|Escape|Lower|HeapFallback' -count=1"),
		memoryFuzzSliceCoverageRow("v10", []string{"async cancellation", "task group boundary", "actor reentrant callback boundary"},
			[]MemoryFuzzOracleCategory{MemoryFuzzOracleCheckerRejectExpected, MemoryFuzzOracleReportValidationFailureBug},
			[]MemoryFuzzInvariantID{MemoryFuzzInvariantNoBorrowedEscape, MemoryFuzzInvariantNoStackRegionStorageWhenEscaped},
			"go test ./compiler/internal/memoryfacts ./compiler/internal/memorymodel -run 'Async|Task|Actor|Cancel' -count=1"),
		memoryFuzzSliceCoverageRow("v11", []string{"dynamic protocol", "existential borrow carrier", "witness/conformance table conservatism"},
			[]MemoryFuzzOracleCategory{MemoryFuzzOracleCheckerRejectExpected, MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug},
			[]MemoryFuzzInvariantID{MemoryFuzzInvariantNoBorrowedEscape, MemoryFuzzInvariantNoUnsafeUnknownToSafeKnown},
			"go test ./compiler/internal/memoryfacts ./compiler/internal/memorymodel -run 'Dynamic|Protocol|Witness|Conformance' -count=1"),
	}
}

func memoryFuzzSliceCoverageRow(sliceID string, surface []string, categories []MemoryFuzzOracleCategory, invariants []MemoryFuzzInvariantID, test string) MemoryFuzzSliceCoverageRow {
	return MemoryFuzzSliceCoverageRow{
		SliceID:          sliceID,
		Status:           "covered",
		Surface:          surface,
		OracleCategories: categories,
		Invariants:       invariants,
		Evidence:         []string{"deterministic Tier 1 memory fuzz oracle coverage is recorded for " + sliceID},
		Tests:            []string{test},
		Boundaries:       []string{"coverage is limited to supported compiler-visible " + sliceID + " memory evidence and is not exhaustive fuzz proof"},
	}
}

func memoryFuzzBlockingCaseRow(id MemoryFuzzBlockingCaseID, evidence string, test string, boundary string) MemoryFuzzBlockingCaseRow {
	return MemoryFuzzBlockingCaseRow{
		ID:            id,
		Status:        "blocks_release",
		BlocksRelease: true,
		Evidence:      []string{evidence},
		Tests:         []string{test},
		Boundaries:    []string{boundary},
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
	issues = append(issues, validateMemoryFuzzRequirements(report.Requirements)...)
	issues = append(issues, validateMemoryFuzzSliceCoverage(report.SliceCoverage)...)
	issues = append(issues, validateMemoryFuzzOracleRows(report.Rows)...)
	issues = append(issues, validateMemoryFuzzInvariants(report.Invariants)...)
	issues = append(issues, validateMemoryFuzzGeneratorSurfaces(report.GeneratorSurfaces)...)
	issues = append(issues, validateMemoryFuzzBlockingCases(report.BlockingCases)...)
	issues = append(issues, validateMemoryFuzzTierPolicies(report.TierPolicies)...)
	if len(report.Artifacts) == 0 {
		issues = append(issues, "memory fuzz oracle artifacts are required")
	}
	issues = append(issues, validateMemoryFuzzRequiredArtifacts(report.Artifacts)...)
	for _, want := range []string{
		"no exhaustive fuzzing is claimed",
		"no exhaustive fuzz proof is claimed",
		"no unsupported unsafe pointer safety is claimed",
		"no arbitrary unsafe safety is claimed",
		"no full runtime/ABI/target parity proof is claimed",
		"no runtime behavior change",
		"no safe-program semantics change",
		"no performance claim is made",
		"no clean-release claim under dirty worktree",
		"no replacement for MemoryFactGraph validators",
		"no Memory 100% claim is made",
	} {
		if !memoryFuzzHasString(report.NonClaims, want) {
			issues = append(issues, fmt.Sprintf("missing non-claim %q", want))
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateMemoryFuzzRequirements(rows []MemoryFuzzRequirementRow) []string {
	seen := map[MemoryFuzzRequirementID]bool{}
	var issues []string
	expected := map[MemoryFuzzRequirementID]string{
		MemoryFuzzRequirementTier1V0V11Coverage:         "validated_narrow",
		MemoryFuzzRequirementCrashMiscompileArtifacts:   "validated_narrow",
		MemoryFuzzRequirementBlockingMemoryFailures:     "release_blocking",
		MemoryFuzzRequirementTier2NightlySeedTriage:     "boundary_recorded",
		MemoryFuzzRequirementTier3ReleasePassOrClassify: "release_blocking",
	}
	for _, row := range rows {
		if !knownMemoryFuzzRequirementID(row.ID) {
			issues = append(issues, fmt.Sprintf("unknown requirement %q", row.ID))
		}
		if seen[row.ID] {
			issues = append(issues, fmt.Sprintf("duplicate requirement %s", row.ID))
		}
		seen[row.ID] = true
		if want := expected[row.ID]; row.Status != want {
			issues = append(issues, fmt.Sprintf("requirement %s status = %q, want %q", row.ID, row.Status, want))
		}
		issues = append(issues, validateMemoryFuzzTextList("requirement "+string(row.ID)+" evidence", row.Evidence)...)
		issues = append(issues, validateMemoryFuzzTextList("requirement "+string(row.ID)+" tests", row.Tests)...)
		issues = append(issues, validateMemoryFuzzTextList("requirement "+string(row.ID)+" boundaries", row.Boundaries)...)
	}
	for _, id := range memoryFuzzRequirementIDs() {
		if !seen[id] {
			issues = append(issues, fmt.Sprintf("missing requirement %s", id))
		}
	}
	return issues
}

func validateMemoryFuzzSliceCoverage(rows []MemoryFuzzSliceCoverageRow) []string {
	seen := map[string]bool{}
	var issues []string
	for _, row := range rows {
		if !knownMemoryFuzzSliceID(row.SliceID) {
			issues = append(issues, fmt.Sprintf("unknown slice coverage %q", row.SliceID))
		}
		if seen[row.SliceID] {
			issues = append(issues, fmt.Sprintf("duplicate slice coverage %s", row.SliceID))
		}
		seen[row.SliceID] = true
		if row.Status != "covered" {
			issues = append(issues, fmt.Sprintf("slice coverage %s status = %q, want covered", row.SliceID, row.Status))
		}
		issues = append(issues, validateMemoryFuzzTextList("slice coverage "+row.SliceID+" surface", row.Surface)...)
		if len(row.OracleCategories) == 0 {
			issues = append(issues, fmt.Sprintf("slice coverage %s oracle_categories are required", row.SliceID))
		}
		for _, category := range row.OracleCategories {
			if !knownMemoryFuzzOracleCategory(category) {
				issues = append(issues, fmt.Sprintf("slice coverage %s unknown oracle_category %q", row.SliceID, category))
			}
		}
		if len(row.Invariants) == 0 {
			issues = append(issues, fmt.Sprintf("slice coverage %s invariants are required", row.SliceID))
		}
		for _, id := range row.Invariants {
			if !knownMemoryFuzzInvariantID(id) {
				issues = append(issues, fmt.Sprintf("slice coverage %s unknown invariant %q", row.SliceID, id))
			}
		}
		issues = append(issues, validateMemoryFuzzTextList("slice coverage "+row.SliceID+" evidence", row.Evidence)...)
		issues = append(issues, validateMemoryFuzzTextList("slice coverage "+row.SliceID+" tests", row.Tests)...)
		issues = append(issues, validateMemoryFuzzTextList("slice coverage "+row.SliceID+" boundaries", row.Boundaries)...)
	}
	for _, id := range memoryFuzzSliceIDs() {
		if !seen[id] {
			issues = append(issues, fmt.Sprintf("missing slice coverage %s", id))
		}
	}
	return issues
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

func validateMemoryFuzzBlockingCases(rows []MemoryFuzzBlockingCaseRow) []string {
	seen := map[MemoryFuzzBlockingCaseID]bool{}
	var issues []string
	for _, row := range rows {
		if !knownMemoryFuzzBlockingCaseID(row.ID) {
			issues = append(issues, fmt.Sprintf("unknown blocking case %q", row.ID))
		}
		if seen[row.ID] {
			issues = append(issues, fmt.Sprintf("duplicate blocking case %s", row.ID))
		}
		seen[row.ID] = true
		if row.Status != "blocks_release" {
			issues = append(issues, fmt.Sprintf("blocking case %s status = %q, want blocks_release", row.ID, row.Status))
		}
		if !row.BlocksRelease {
			issues = append(issues, fmt.Sprintf("blocking case %s must set blocks_release", row.ID))
		}
		issues = append(issues, validateMemoryFuzzTextList("blocking case "+string(row.ID)+" evidence", row.Evidence)...)
		issues = append(issues, validateMemoryFuzzTextList("blocking case "+string(row.ID)+" tests", row.Tests)...)
		issues = append(issues, validateMemoryFuzzTextList("blocking case "+string(row.ID)+" boundaries", row.Boundaries)...)
	}
	for _, id := range memoryFuzzBlockingCaseIDs() {
		if !seen[id] {
			issues = append(issues, fmt.Sprintf("missing blocking case %s", id))
		}
	}
	return issues
}

func validateMemoryFuzzTierPolicies(rows []MemoryFuzzTierPolicyRow) []string {
	seen := map[MemoryFuzzTier]bool{}
	var issues []string
	for _, row := range rows {
		if !knownMemoryFuzzTier(row.Tier) {
			issues = append(issues, fmt.Sprintf("unknown tier policy %q", row.Tier))
		}
		if seen[row.Tier] {
			issues = append(issues, fmt.Sprintf("duplicate tier policy %s", row.Tier))
		}
		seen[row.Tier] = true
		if strings.TrimSpace(row.Status) == "" {
			issues = append(issues, fmt.Sprintf("tier policy %s status is required", row.Tier))
		}
		switch row.Tier {
		case MemoryFuzzTier1ShortCI:
			if row.Status != "covered" {
				issues = append(issues, fmt.Sprintf("Tier 1 short CI smoke status = %q, want covered", row.Status))
			}
		case MemoryFuzzTier2Nightly:
			if row.Status != "boundary_recorded" {
				issues = append(issues, fmt.Sprintf("Tier 2 nightly fuzz status = %q, want boundary_recorded", row.Status))
			}
			if !row.SeedsPreserved {
				issues = append(issues, "Tier 2 nightly fuzz seed preservation is required")
			}
			if !row.UnstableTriageRequired {
				issues = append(issues, "Tier 2 nightly fuzz unstable triage is required")
			}
			if !row.MinimizedReproducerRequired {
				issues = append(issues, "Tier 2 nightly fuzz minimized repro is required")
			}
		case MemoryFuzzTier3ReleaseFocused:
			if row.Status != "release_blocking" {
				issues = append(issues, fmt.Sprintf("Tier 3 release-blocking memory fuzz status = %q, want release_blocking", row.Status))
			}
			if !row.ReleasePromotionBlockedUntilClassified {
				issues = append(issues, "Tier 3 release-blocking memory fuzz must block promotion until every failure is classified")
			}
			if !row.MinimizedReproducerRequired {
				issues = append(issues, "Tier 3 release-blocking memory fuzz minimized repro is required")
			}
		}
		issues = append(issues, validateMemoryFuzzTextList("tier policy "+string(row.Tier)+" evidence", row.Evidence)...)
		issues = append(issues, validateMemoryFuzzTextList("tier policy "+string(row.Tier)+" tests", row.Tests)...)
		issues = append(issues, validateMemoryFuzzTextList("tier policy "+string(row.Tier)+" boundaries", row.Boundaries)...)
	}
	for _, tier := range []MemoryFuzzTier{MemoryFuzzTier1ShortCI, MemoryFuzzTier2Nightly, MemoryFuzzTier3ReleaseFocused} {
		if !seen[tier] {
			issues = append(issues, fmt.Sprintf("missing tier policy %s", tier))
		}
	}
	return issues
}

func validateMemoryFuzzRequiredArtifacts(artifacts []MemoryFuzzArtifact) []string {
	seen := map[string]MemoryFuzzArtifact{}
	var issues []string
	for _, artifact := range artifacts {
		if strings.TrimSpace(artifact.Kind) == "" {
			issues = append(issues, "artifact kind is required")
			continue
		}
		seen[artifact.Kind] = artifact
		issues = append(issues, validateMemoryFuzzTextList("artifact "+artifact.Kind+" path", []string{artifact.Path})...)
	}
	for _, kind := range memoryFuzzRequiredArtifactKinds() {
		artifact, ok := seen[kind]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing required artifact kind %s", kind))
			continue
		}
		if !artifact.Required {
			issues = append(issues, fmt.Sprintf("artifact kind %s must be required", kind))
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

func memoryFuzzRequirementIDs() []MemoryFuzzRequirementID {
	return []MemoryFuzzRequirementID{
		MemoryFuzzRequirementTier1V0V11Coverage,
		MemoryFuzzRequirementCrashMiscompileArtifacts,
		MemoryFuzzRequirementBlockingMemoryFailures,
		MemoryFuzzRequirementTier2NightlySeedTriage,
		MemoryFuzzRequirementTier3ReleasePassOrClassify,
	}
}

func memoryFuzzSliceIDs() []string {
	return []string{"v0", "v1", "v2", "v3", "v4", "v5", "v6", "v7", "v8", "v9", "v10", "v11"}
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

func memoryFuzzBlockingCaseIDs() []MemoryFuzzBlockingCaseID {
	return []MemoryFuzzBlockingCaseID{
		MemoryFuzzBlockingUnsafeUnknownOptimizedAsSafe,
		MemoryFuzzBlockingBoundsCheckWithoutProofID,
		MemoryFuzzBlockingTrustedStorageUnderEscape,
		MemoryFuzzBlockingReportValidationFailure,
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

func memoryFuzzRequiredArtifactKinds() []string {
	return []string{
		"tier1_short_ci_smoke_report",
		"tier1_short_ci_smoke_summary",
		"compiler_crash_reproducer",
		"miscompile_reproducer",
		"miscompile_reducer",
		"audit_contract",
	}
}

func knownMemoryFuzzRequirementID(id MemoryFuzzRequirementID) bool {
	for _, known := range memoryFuzzRequirementIDs() {
		if id == known {
			return true
		}
	}
	return false
}

func knownMemoryFuzzSliceID(id string) bool {
	for _, known := range memoryFuzzSliceIDs() {
		if id == known {
			return true
		}
	}
	return false
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

func knownMemoryFuzzBlockingCaseID(id MemoryFuzzBlockingCaseID) bool {
	for _, known := range memoryFuzzBlockingCaseIDs() {
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

func (r *MemoryFuzzOracleReport) BlockingCase(id MemoryFuzzBlockingCaseID) *MemoryFuzzBlockingCaseRow {
	for i := range r.BlockingCases {
		if r.BlockingCases[i].ID == id {
			return &r.BlockingCases[i]
		}
	}
	return &MemoryFuzzBlockingCaseRow{}
}

func (r *MemoryFuzzOracleReport) TierPolicy(tier MemoryFuzzTier) *MemoryFuzzTierPolicyRow {
	for i := range r.TierPolicies {
		if r.TierPolicies[i].Tier == tier {
			return &r.TierPolicies[i]
		}
	}
	return &MemoryFuzzTierPolicyRow{}
}

func cloneMemoryFuzzOracleReport(in MemoryFuzzOracleReport) MemoryFuzzOracleReport {
	out := in
	out.Requirements = append([]MemoryFuzzRequirementRow(nil), in.Requirements...)
	out.SliceCoverage = append([]MemoryFuzzSliceCoverageRow(nil), in.SliceCoverage...)
	out.Rows = append([]MemoryFuzzOracleRow(nil), in.Rows...)
	out.Invariants = append([]MemoryFuzzInvariantRow(nil), in.Invariants...)
	out.GeneratorSurfaces = append([]MemoryFuzzGeneratorSurfaceRow(nil), in.GeneratorSurfaces...)
	out.BlockingCases = append([]MemoryFuzzBlockingCaseRow(nil), in.BlockingCases...)
	out.TierPolicies = append([]MemoryFuzzTierPolicyRow(nil), in.TierPolicies...)
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
