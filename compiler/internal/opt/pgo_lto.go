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
	"tetra_language/compiler/internal/semantics"
)

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
			"compiler/internal/opt/pgo_lto.go::ValidatePGOLTOTargetCPUSafeSemanticsClosure",
			"compiler/reports_internal_test.go::TestBuildOptionsExposeNoBackendSemanticMode",
			"compiler/internal/opt/manager_test.go::TestManagerRejectsProfileGuidedRewritePolicyUntilValidationExists",
			"compiler/internal/backend/x64/target_features_test.go::TestCodegenOptionsTargetFeatureGuardIsEvidenceOnly",
			"compiler/internal/cache/lto_summary_test.go::TestIncrementalModuleSummaryV1RecordsDependencyHashContractAndRejectsConsumers",
		},
		Boundary: "final P17.4 closure is evidence-only: no PGO/profile/LTO/target-cpu public flag changes safe-program semantics, profile-guided rewrite policy is rejected, target-specific optimization evidence is rejected, and LTO/incremental summaries remain non-consumer evidence only",
	}, nil
}

func ValidatePGOLTOTargetCPUSafeSemanticsClosure(report PGOLTOTargetCPUCoverageReport) error {
	if report.SchemaVersion != "tetra.optimizer.pgo_lto_target_cpu.v1" {
		return fmt.Errorf("P17.4 safe-semantics closure: schema = %q", report.SchemaVersion)
	}
	if !hasReportRow(report.NonClaims, "no PGO, LTO, target-cpu, or profile flag changes safe-program semantics") {
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
		return fmt.Errorf("P17.4 safe-semantics closure: row count = %d, want %d", len(report.Rows), len(expected))
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
			return fmt.Errorf("P17.4 safe-semantics closure: row %q missing machine-checkable evidence", row.ID)
		}
		if row.Status != PGOLTOTargetCPUImplementedNarrow {
			return fmt.Errorf("P17.4 safe-semantics closure: row %q not complete: %s", row.ID, row.Status)
		}
		if len(row.MissingFacts) != 0 {
			return fmt.Errorf("P17.4 safe-semantics closure: row %q has missing facts: %v", row.ID, row.MissingFacts)
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
	if err := validateProfileCollectionClosureRow(byID[PGOLTOTargetCPUProfileCollectionFormat]); err != nil {
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
		return fmt.Errorf("profile collection: schema_version = %q, want %q", profile.SchemaVersion, ProfileCollectionSchemaVersion)
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
				return fmt.Errorf("profile collection: function %q counter %d missing kind", fn.ID, j)
			}
			if strings.TrimSpace(counter.Name) == "" {
				return fmt.Errorf("profile collection: function %q counter %d missing name", fn.ID, j)
			}
			key := counter.Kind + "\x00" + counter.Name
			if seenCounters[key] {
				return fmt.Errorf("profile collection: function %q duplicate counter %q/%q", fn.ID, counter.Kind, counter.Name)
			}
			seenCounters[key] = true
		}
	}
	return nil
}

func BuildOptimizerProfileInputEvidence(profile ProfileCollection) (OptimizerProfileInputEvidence, error) {
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
				Reason:   "guarded:no public BuildOptions flag applies PGO/LTO/target-cpu/profile data; profile input is internal evidence with unused pass policy, profile-guided rewrite rejection, and final safe-semantics closure validation",
				Evidence: "compiler/compiler.go::BuildOptions; compiler/reports_internal_test.go::TestBuildOptionsExposeNoBackendSemanticMode; compiler/internal/opt/pgo_lto.go::PGOLTOTargetCPUCoverage; compiler/internal/opt/pgo_lto.go::ValidatePGOLTOTargetCPUSafeSemanticsClosure; compiler/internal/opt/pgo_lto_test.go::TestPGOLTOTargetCPUSafeSemanticsClosureRejectsFakeClaims",
				Boundary: "no public BuildOptions flag, no optimizer pass consumes profile counts, all registered optimizer passes declare profile_input_policy unused, profile-guided rewrite policy is rejected, target-cpu feature data is internal evidence only with no target-specific rewrite, and LTO/incremental summaries are non-consumer evidence only; no LTO setting reaches codegen or linker from this slice; profile parsing is evidence-only, profile input is report/metadata evidence only, the final coverage validator rejects fake semantic-changing coverage rows, incomplete rows, fake profile-format optimizer input, fake target-cpu/LTO optimizer input, and missing safe-program truth facts, and safe-program semantics unchanged",
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
		return fmt.Errorf("P17.4 safe-semantics closure: profile collection format must remain inert evidence, not optimizer input")
	}
	for _, fact := range []string{"schema_validation", "canonical_json", "duplicate_rejection", "negative_counter_rejection"} {
		if !hasReportRow(row.RequiredFacts, fact) {
			return fmt.Errorf("P17.4 safe-semantics closure: profile collection format missing required fact %q", fact)
		}
	}
	if !strings.Contains(row.Boundary, "inert evidence") || !strings.Contains(row.Boundary, "does not feed optimizer decisions") {
		return fmt.Errorf("P17.4 safe-semantics closure: profile collection format boundary no longer proves inert evidence")
	}
	return nil
}

func validatePGOInputClosureRow(row PGOLTOTargetCPUCoverageRow) error {
	if !row.OptimizerInput {
		return fmt.Errorf("P17.4 safe-semantics closure: PGO optimizer input row must record optimizer input evidence")
	}
	for _, fact := range []string{"optimizer_profile_input_api", "pass_contract_profile_metadata", "translation_validation_for_profile_guided_decisions", "negative_safe_semantics_tests"} {
		if !hasReportRow(row.RequiredFacts, fact) {
			return fmt.Errorf("P17.4 safe-semantics closure: PGO optimizer input missing required fact %q", fact)
		}
	}
	if !strings.Contains(row.Boundary, "all registered passes keep profile_input_policy unused") ||
		!strings.Contains(row.Boundary, "no profile-guided rewrite is selected") {
		return fmt.Errorf("P17.4 safe-semantics closure: PGO optimizer input boundary no longer rejects profile-guided rewrite")
	}
	return nil
}

func validateTargetCPUClosureRow(row PGOLTOTargetCPUCoverageRow) error {
	if row.OptimizerInput {
		return fmt.Errorf("P17.4 safe-semantics closure: target-cpu feature detection must not be optimizer input")
	}
	for _, fact := range []string{"target_feature_model", "portable_baseline_fallback", "guarded_codegen_contract", "negative_safe_semantics_tests"} {
		if !hasReportRow(row.RequiredFacts, fact) {
			return fmt.Errorf("P17.4 safe-semantics closure: target-cpu feature detection missing required fact %q", fact)
		}
	}
	if !strings.Contains(row.Boundary, "no public target-cpu BuildOptions field") ||
		!strings.Contains(row.Boundary, "no target-specific rewrite") ||
		!strings.Contains(row.Boundary, "safe-program semantics unchanged") {
		return fmt.Errorf("P17.4 safe-semantics closure: target-cpu boundary no longer proves evidence-only semantics")
	}
	return nil
}

func validateLTOClosureRow(row PGOLTOTargetCPUCoverageRow) error {
	if row.OptimizerInput {
		return fmt.Errorf("P17.4 safe-semantics closure: LTO/incremental module summary must not be optimizer input")
	}
	for _, fact := range []string{"module_summary_schema", "dependency_hash_contract", "cross_module_validation_row", "incremental_cache_negative_tests", "non_consumer_boundary"} {
		if !hasReportRow(row.RequiredFacts, fact) {
			return fmt.Errorf("P17.4 safe-semantics closure: LTO/incremental module summary missing required fact %q", fact)
		}
	}
	for _, want := range []string{"no LTO optimizer", "cross-module inlining", "linker consumer", "codegen consumer", "safe-program semantics change"} {
		if !strings.Contains(row.Boundary, want) {
			return fmt.Errorf("P17.4 safe-semantics closure: LTO/incremental module summary boundary missing %q", want)
		}
	}
	return nil
}

func validateSafeSemanticsClosureRow(row PGOLTOTargetCPUCoverageRow) error {
	if row.OptimizerInput {
		return fmt.Errorf("P17.4 safe-semantics closure: safe semantics row must not be optimizer input")
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
			return fmt.Errorf("P17.4 safe-semantics closure: safe semantics row missing required fact %q", fact)
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
			return fmt.Errorf("P17.4 safe-semantics closure: safe semantics row boundary missing %q", want)
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
		return nil, fmt.Errorf("P17.4 safe-semantics closure: fake semantic-changing coverage was accepted")
	}
	rejected := []string{
		"coverage_validator_rejects_fake_claims",
		"public_build_options_semantic_flag_rejected",
	}

	guided := pgoInputEvidencePass()
	guided.ProfileInputPolicy = ProfileInputGuidedRewrite
	if err := ValidatePassContract(guided); err == nil {
		return nil, fmt.Errorf("P17.4 safe-semantics closure: profile-guided rewrite policy was accepted")
	}
	rejected = append(rejected, "profile_guided_rewrite_policy_rejected")

	if err := validateTargetFeatureClosureEvidence(x64.TargetFeatureEvidence{
		Source:                            string(x64.TargetFeatureSourceExplicit),
		Features:                          []string{string(x64.TargetFeatureSSE2), string(x64.TargetFeatureAVX2)},
		PortableBaselineFallback:          false,
		ChangesSafeSemantics:              false,
		EnablesTargetSpecificOptimization: true,
	}); err == nil {
		return nil, fmt.Errorf("P17.4 safe-semantics closure: target-specific optimization evidence was accepted")
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
		return PGOLTOTargetCPUCoverageRow{}, fmt.Errorf("profile collection: canonical round trip drifted: %s vs %s", string(encoded), string(reencoded))
	}
	for name, raw := range map[string][]byte{
		"duplicate":        []byte(`{"schema_version":"tetra.optimizer.profile.v1","program_hash":"sha256:p17profilefixture","target_triple":"linux-x64","functions":[{"id":"fn:main","name":"main","entry_count":1},{"id":"fn:main","name":"other","entry_count":1}]}`),
		"negative counter": []byte(`{"schema_version":"tetra.optimizer.profile.v1","program_hash":"sha256:p17profilefixture","target_triple":"linux-x64","functions":[{"id":"fn:main","name":"main","entry_count":1,"counters":[{"kind":"edge","name":"return","count":-1}]}]}`),
	} {
		if _, err := ParseProfileCollection(raw); err == nil {
			return PGOLTOTargetCPUCoverageRow{}, fmt.Errorf("profile collection: %s fixture unexpectedly accepted", name)
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
		Reason:   "implemented_narrow:tetra.optimizer.profile.v1 canonical JSON profile collection format with duplicate and negative counter rejection; inert until a separate optimizer-input slice consumes it",
		Evidence: "compiler/internal/opt/pgo_lto.go::ProfileCollection; compiler/internal/opt/pgo_lto_test.go::TestProfileCollectionFormatV1RoundTripsAndRejectsUnsafeDrift",
		Boundary: "tetra.optimizer.profile.v1 is an inert evidence format only: it records canonical JSON function entry counts and named counters, rejects duplicate function/counter identity and negative counter JSON, and does not feed optimizer decisions, codegen, target-cpu selection, LTO, incremental compilation, or safe-program semantics",
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
	report, err := NewManager().RunWithOptions(prog, Options{ProfileInput: &profile}, pgoInputEvidencePass())
	if err != nil {
		return PGOLTOTargetCPUCoverageRow{}, err
	}
	if FormatProgram(prog) != before {
		return PGOLTOTargetCPUCoverageRow{}, fmt.Errorf("pgo optimizer input: profile input changed IR")
	}
	if len(report.Passes) != 1 {
		return PGOLTOTargetCPUCoverageRow{}, fmt.Errorf("pgo optimizer input: expected one pass report, got %d", len(report.Passes))
	}
	row := report.Passes[0]
	if row.ProfileInputPolicy != ProfileInputUnused || row.ProfileInput == nil || row.ProfileInput.Digest != evidence.Digest {
		return PGOLTOTargetCPUCoverageRow{}, fmt.Errorf("pgo optimizer input: missing profile input report evidence")
	}
	if !hasReportRow(row.ReportRows, "profile_input_policy") {
		return PGOLTOTargetCPUCoverageRow{}, fmt.Errorf("pgo optimizer input: pass report missing profile_input_policy row")
	}
	if row.ValidationMetadata == nil ||
		row.ValidationMetadata.ProfileInputPolicy != string(ProfileInputUnused) ||
		row.ValidationMetadata.ProfileInputDigest != evidence.Digest ||
		row.ValidationMetadata.ProfileInputSchemaVersion != ProfileCollectionSchemaVersion {
		return PGOLTOTargetCPUCoverageRow{}, fmt.Errorf("pgo optimizer input: missing validation metadata profile evidence")
	}
	rejected := pgoInputEvidencePass()
	rejected.ProfileInputPolicy = ProfileInputGuidedRewrite
	if _, err := NewManager().RunWithOptions(pgoInputEvidenceProgram(), Options{ProfileInput: &profile}, rejected); err == nil {
		return PGOLTOTargetCPUCoverageRow{}, fmt.Errorf("pgo optimizer input: profile-guided rewrite policy was accepted without dedicated validation")
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
		Reason:   "implemented_narrow:Options.ProfileInput validates profile input and records profile_input_policy plus profile digest in pass reports and validation metadata; profile-guided rewrite policy rejected",
		Evidence: "compiler/internal/opt/manager.go::Options.ProfileInput; compiler/internal/opt/manager_test.go::TestManagerAcceptsProfileInputAsValidatedMetadataWithoutChangingIR; compiler/internal/opt/manager_test.go::TestManagerRejectsProfileGuidedRewritePolicyUntilValidationExists; compiler/internal/validation/metadata_test.go::TestBuildOptimizationValidationMetadataRecordsMachineCheckableEvidence",
		Boundary: "internal PGO input to optimizer is implemented only as validated profile input API, profile_input_policy pass-contract metadata, translation validation metadata evidence, and negative safe-semantics rejection for profile-guided rewrite policy; all registered passes keep profile_input_policy unused, no profile-guided rewrite is selected, no public flag exists, no codegen/LTO behavior changes, and no performance claim is made",
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
		return targetCPUFeatureDetectionFailureRow(fmt.Errorf("target-cpu evidence: incomplete portable baseline evidence: %#v", evidence))
	}
	if allowed, err := baselineOpt.AllowsTargetFeature(x64.TargetFeatureAVX2); err != nil || allowed {
		return targetCPUFeatureDetectionFailureRow(fmt.Errorf("target-cpu evidence: default avx2 allowed=%v err=%v", allowed, err))
	}
	_, err = (x64.CodegenOptions{
		RegisterWidthBits: 64,
		TargetFeatures: x64.TargetFeatures{
			Source:   x64.TargetFeatureSourceExplicit,
			Features: []x64.TargetFeature{x64.TargetFeatureAVX2},
		},
	}).EffectiveTargetFeatures()
	if err == nil {
		return targetCPUFeatureDetectionFailureRow(fmt.Errorf("target-cpu evidence: explicit target feature set below portable baseline accepted"))
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
		Reason:   "implemented_narrow:internal target feature model records portable baseline fallback and guarded codegen contract evidence; default x64/x32 evidence includes sse2 baseline only and no target-specific rewrite is enabled",
		Evidence: "compiler/internal/backend/x64/target_features.go::CodegenOptions.TargetFeatureEvidence; compiler/internal/backend/x64/target_features_test.go::TestTargetFeatureModelUsesPortableBaselineAndRejectsUnsafeDrift; compiler/compiler_pipeline_stage_test.go::TestNativeCodegenOptionsUsePortableTargetFeatureBaseline; compiler/reports_internal_test.go::TestBuildOptionsExposeNoBackendSemanticMode",
		Boundary: "target-cpu feature detection foundation is evidence-only: it provides an explicit internal target feature model, portable baseline fallback, guarded codegen contract queries, and negative safe-semantics rejection for explicit features below baseline; no host CPU detector, no public target-cpu BuildOptions field, no target-specific rewrite, no optimizer input, no LTO/codegen behavior change, no performance claim, and safe-program semantics unchanged",
	}
}

func targetCPUFeatureDetectionFailureRow(err error) PGOLTOTargetCPUCoverageRow {
	row := pgoNotYetCoveredRow(
		PGOLTOTargetCPUTargetCPUFeatureDetection,
		"target-cpu feature detection",
		"target-cpu feature detection foundation failed self-validation",
		[]string{"target_feature_model", "portable_baseline_fallback", "guarded_codegen_contract", "negative_safe_semantics_tests"},
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
		return ltoIncrementalModuleSummaryFailureRow(fmt.Errorf("lto incremental module summary: missing self-validation evidence"))
	}
	consumer := decoded
	consumer.CodegenConsumer = true
	if err := cache.ValidateIncrementalModuleSummary(consumer); err == nil {
		return ltoIncrementalModuleSummaryFailureRow(fmt.Errorf("lto incremental module summary: codegen consumer accepted"))
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
		Reason:   "implemented_narrow:tetra.incremental.module_summary.v1 records source/public API/dependency hash contract evidence, cross-module validation rows, and non-consumer boundary; no LTO optimizer is implemented",
		Evidence: "compiler/internal/cache/lto_summary.go::IncrementalModuleSummary; compiler/internal/cache/lto_summary_test.go::TestIncrementalModuleSummaryV1RecordsDependencyHashContractAndRejectsConsumers; compiler/internal/opt/pgo_lto_test.go::TestPGOLTOTargetCPUCoverageAuditsP17PlanList",
		Boundary: "LTO/incremental module summary foundation is evidence-only: it records module source hash, dependency hash contract, public API hash, external callee/type dependency inputs, cross-module validation rows, incremental negative tests, and non-consumer boundary; no LTO optimizer, cross-module inlining, linker consumer, codegen consumer, cache mode, incremental speedup, public flag, performance claim, or safe-program semantics change is made",
	}
}

func ltoIncrementalModuleSummaryFailureRow(err error) PGOLTOTargetCPUCoverageRow {
	row := pgoNotYetCoveredRow(
		PGOLTOTargetCPULTOIncrementalModuleSummary,
		"LTO/incremental module summary",
		"LTO/incremental module summary foundation failed self-validation",
		[]string{"module_summary_schema", "dependency_hash_contract", "cross_module_validation_row", "incremental_cache_negative_tests", "non_consumer_boundary"},
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
		ProofRule:                 ProofRulePreserveBoundsInvalidateLiveness,
		ValidationStrategy:        ValidationTranslation,
		TranslationValidationHook: TranslationHookValidateTranslation,
		ReportOutput:              "pgo-input-evidence.opt.json",
		ReportRows:                RequiredP17ReportRows(),
		NegativeTestMarker:        NegativeTestPassContractV1,
		ProfileInputPolicy:        ProfileInputUnused,
		Run:                       func(p *ir.IRProgram) error { return nil },
	}
}

func pgoNotYetCoveredRow(id PGOLTOTargetCPUID, name string, reason string, missing []string) PGOLTOTargetCPUCoverageRow {
	return PGOLTOTargetCPUCoverageRow{
		ID:                   id,
		Name:                 name,
		Status:               PGOLTOTargetCPUNotYetCovered,
		OptimizerInput:       false,
		ChangesSafeSemantics: false,
		MissingFacts:         append([]string(nil), missing...),
		Reason:               "not_yet_covered:" + reason,
		Evidence:             "P17.4 master-plan row; no implementation evidence has been promoted for this row",
		Boundary:             reason + "; no optimizer input, no codegen/LTO behavior, no performance claim, and no safe-program semantic change",
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
