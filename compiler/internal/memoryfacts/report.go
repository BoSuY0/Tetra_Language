package memoryfacts

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"tetra_language/compiler/memoryvocab"
)

const ReportSchemaV1 = "tetra.memory-report.v1"

type Report struct {
	SchemaVersion string      `json:"schema_version"`
	Rows          []ReportRow `json:"rows"`
}

type ReportRow struct {
	ProgramID             string          `json:"program_id,omitempty"`
	FunctionID            string          `json:"function_id,omitempty"`
	ValueID               string          `json:"value_id,omitempty"`
	IslandID              string          `json:"island_id,omitempty"`
	Epoch                 int             `json:"epoch,omitempty"`
	BaseID                string          `json:"base_id,omitempty"`
	SiteID                string          `json:"site_id,omitempty"`
	SourceSpan            string          `json:"source_span,omitempty"`
	SourceFactID          FactID          `json:"source_fact_id,omitempty"`
	ParentFactID          FactID          `json:"parent_fact_id,omitempty"`
	LoweredArtifactID     string          `json:"lowered_artifact_id,omitempty"`
	SourceStage           SourceStage     `json:"source_stage,omitempty"`
	Claim                 string          `json:"claim,omitempty"`
	ClaimLevel            ClaimLevel      `json:"claim_level,omitempty"`
	ProvenanceClass       ProvenanceClass `json:"provenance_class,omitempty"`
	OwnerID               string          `json:"owner_id,omitempty"`
	ParamIndex            *int            `json:"param_index,omitempty"`
	ParamPath             string          `json:"param_path,omitempty"`
	BorrowState           BorrowState     `json:"borrow_state,omitempty"`
	EscapeState           EscapeState     `json:"escape_state,omitempty"`
	AliasState            AliasState      `json:"alias_state,omitempty"`
	UnsafeClass           UnsafeClass     `json:"unsafe_class,omitempty"`
	AllocationSiteID      string          `json:"allocation_site_id,omitempty"`
	PlannedStorage        StorageClass    `json:"planned_storage,omitempty"`
	ActualLoweringStorage StorageClass    `json:"actual_lowering_storage,omitempty"`
	ProofID               string          `json:"proof_id,omitempty"`
	ProofKind             string          `json:"proof_kind,omitempty"`
	ProofSubjectBaseID    string          `json:"proof_subject_base_id,omitempty"`
	ProofIndexValueID     string          `json:"proof_index_value_id,omitempty"`
	ProofOperation        string          `json:"proof_operation,omitempty"`
	ProofRange            string          `json:"proof_range,omitempty"`
	ValidatorName         string          `json:"validator_name,omitempty"`
	ValidatorStatus       ValidatorStatus `json:"validator_status,omitempty"`
	CostClass             CostClass       `json:"cost_class,omitempty"`
	NormalBuildCheck      bool            `json:"normal_build_check,omitempty"`
	Reason                string          `json:"reason,omitempty"`
}

func BuildReportFromGraph(graph *Graph) Report {
	report := Report{SchemaVersion: ReportSchemaV1}
	if graph == nil {
		return report
	}
	for _, fact := range graph.Facts() {
		report.Rows = append(report.Rows, rowFromFact(fact))
	}
	sortReportRows(report.Rows)
	return report
}

func ValidateReportJSON(raw []byte) error {
	var report Report
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("trailing data after memory report JSON")
		}
		return fmt.Errorf("trailing data after memory report JSON: %w", err)
	}
	return ValidateReport(report)
}

func ValidateReport(report Report) error {
	var issues []string
	if report.SchemaVersion != ReportSchemaV1 {
		issues = append(
			issues,
			fmt.Sprintf("schema_version is %q, want %q", report.SchemaVersion, ReportSchemaV1),
		)
	}
	if len(report.Rows) == 0 {
		issues = append(issues, "rows are required")
	}
	for i, row := range report.Rows {
		issues = append(issues, validateReportRow(i, row)...)
	}
	if index, previous, current, ok := firstNonDeterministicReportRow(report.Rows); ok {
		issues = append(
			issues,
			fmt.Sprintf(
				("row %d: non-deterministic memory report row order: source_fact_"+
					"id %q sorts before previous source_fact_id %q"),
				index,
				current.SourceFactID,
				previous.SourceFactID,
			),
		)
	}
	seenFactIDs := map[FactID]int{}
	for i, row := range report.Rows {
		sourceFactID := FactID(strings.TrimSpace(string(row.SourceFactID)))
		if sourceFactID == "" {
			continue
		}
		if previous, ok := seenFactIDs[sourceFactID]; ok {
			issues = append(
				issues,
				fmt.Sprintf(
					"row %d: duplicate source_fact_id %q also used by row %d",
					i,
					sourceFactID,
					previous,
				),
			)
			continue
		}
		seenFactIDs[sourceFactID] = i
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func sortReportRows(rows []ReportRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		return compareReportRows(rows[i], rows[j]) < 0
	})
}

func firstNonDeterministicReportRow(rows []ReportRow) (int, ReportRow, ReportRow, bool) {
	for i := 1; i < len(rows); i++ {
		if compareReportRows(rows[i-1], rows[i]) > 0 {
			return i, rows[i-1], rows[i], true
		}
	}
	return 0, ReportRow{}, ReportRow{}, false
}

func compareReportRows(a ReportRow, b ReportRow) int {
	for _, pair := range []struct {
		a string
		b string
	}{
		{string(a.SourceFactID), string(b.SourceFactID)},
		{a.ProgramID, b.ProgramID},
		{a.FunctionID, b.FunctionID},
		{a.IslandID, b.IslandID},
		{fmt.Sprintf("%010d", a.Epoch), fmt.Sprintf("%010d", b.Epoch)},
		{a.BaseID, b.BaseID},
		{a.ProofID, b.ProofID},
		{a.SiteID, b.SiteID},
		{a.ValueID, b.ValueID},
		{a.Claim, b.Claim},
		{string(a.SourceStage), string(b.SourceStage)},
		{string(a.ParentFactID), string(b.ParentFactID)},
		{a.LoweredArtifactID, b.LoweredArtifactID},
	} {
		if pair.a < pair.b {
			return -1
		}
		if pair.a > pair.b {
			return 1
		}
	}
	return 0
}

func ValidateReportProjection(graph *Graph, report Report) error {
	var issues []string
	if graph == nil {
		issues = append(issues, "graph is required")
	} else if err := graph.Validate(); err != nil {
		issues = append(issues, "graph validation failed: "+err.Error())
	}
	if err := ValidateReport(report); err != nil {
		issues = append(issues, err.Error())
	}
	expectedRows := map[FactID]ReportRow{}
	if graph != nil {
		for _, fact := range graph.Facts() {
			expectedRows[fact.ID] = rowFromFact(fact)
		}
	}
	seenRows := map[FactID]bool{}
	for index, row := range report.Rows {
		sourceFactID := FactID(strings.TrimSpace(string(row.SourceFactID)))
		if sourceFactID == "" {
			continue
		}
		expected, ok := expectedRows[sourceFactID]
		if !ok {
			issues = append(
				issues,
				fmt.Sprintf(
					"row %d: unknown source_fact_id %q in MemoryFactGraph",
					index,
					sourceFactID,
				),
			)
			continue
		}
		seenRows[sourceFactID] = true
		issues = append(issues, validateReportProjectionRow(index, row, expected)...)
	}
	for _, fact := range graphFactsOrNil(graph) {
		if !seenRows[fact.ID] {
			issues = append(
				issues,
				fmt.Sprintf("missing report row for source_fact_id %q", fact.ID),
			)
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func rowFromFact(fact Fact) ReportRow {
	level := ClaimEvidenceOnly
	status := ValidatorNotRun
	if isConservativeUnknownFact(fact) {
		level = ClaimConservative
		status = ValidatorNotApplicable
	}
	if fact.ValidationState == ValidationPass {
		level = ClaimValidated
		status = ValidatorPass
	}
	if fact.ValidationState == ValidationFail || fact.ValidationState == ValidationInvalidated {
		level = ClaimRejected
		status = ValidatorFail
	}
	return ReportRow{
		ProgramID:             fact.ProgramID,
		FunctionID:            fact.FunctionID,
		ValueID:               fact.ValueID,
		IslandID:              fact.IslandID,
		Epoch:                 fact.Epoch,
		BaseID:                fact.BaseID,
		SiteID:                fact.SiteID,
		SourceSpan:            fact.SourceSpan,
		SourceFactID:          fact.ID,
		ParentFactID:          fact.ParentFactID,
		LoweredArtifactID:     fact.LoweredArtifactID,
		SourceStage:           fact.SourceStage,
		Claim:                 fact.Claim,
		ClaimLevel:            level,
		ProvenanceClass:       fact.ProvenanceClass,
		OwnerID:               fact.OwnerID,
		ParamIndex:            fact.ParamIndex,
		ParamPath:             fact.ParamPath,
		BorrowState:           fact.BorrowState,
		EscapeState:           fact.EscapeState,
		AliasState:            fact.AliasState,
		UnsafeClass:           fact.UnsafeClass,
		AllocationSiteID:      fact.AllocationSiteID,
		PlannedStorage:        fact.StoragePlan,
		ActualLoweringStorage: fact.ActualLoweringStorage,
		ProofID:               fact.ProofID,
		ProofKind:             fact.ProofKind,
		ProofSubjectBaseID:    fact.ProofSubjectBaseID,
		ProofIndexValueID:     fact.ProofIndexValueID,
		ProofOperation:        fact.ProofOperation,
		ProofRange:            fact.ProofRange,
		ValidatorName:         fact.ValidatorName,
		ValidatorStatus:       status,
		CostClass:             fact.CostClass,
		NormalBuildCheck:      fact.NormalBuildCheck,
		Reason:                fact.Reason,
	}
}

func graphFactsOrNil(graph *Graph) []Fact {
	if graph == nil {
		return nil
	}
	return graph.Facts()
}

func validateReportProjectionRow(index int, row ReportRow, expected ReportRow) []string {
	var issues []string
	issues = appendProjectionMismatch(
		issues,
		index,
		"program_id",
		row.ProgramID,
		expected.ProgramID,
	)
	issues = appendProjectionMismatch(
		issues,
		index,
		"function_id",
		row.FunctionID,
		expected.FunctionID,
	)
	issues = appendProjectionMismatch(issues, index, "value_id", row.ValueID, expected.ValueID)
	issues = appendProjectionMismatch(issues, index, "island_id", row.IslandID, expected.IslandID)
	issues = appendProjectionMismatch(issues, index, "epoch", row.Epoch, expected.Epoch)
	issues = appendProjectionMismatch(issues, index, "base_id", row.BaseID, expected.BaseID)
	issues = appendProjectionMismatch(issues, index, "site_id", row.SiteID, expected.SiteID)
	issues = appendProjectionMismatch(
		issues,
		index,
		"source_span",
		row.SourceSpan,
		expected.SourceSpan,
	)
	issues = appendProjectionMismatch(
		issues,
		index,
		"source_fact_id",
		row.SourceFactID,
		expected.SourceFactID,
	)
	issues = appendProjectionMismatch(
		issues,
		index,
		"parent_fact_id",
		row.ParentFactID,
		expected.ParentFactID,
	)
	issues = appendProjectionMismatch(
		issues,
		index,
		"lowered_artifact_id",
		row.LoweredArtifactID,
		expected.LoweredArtifactID,
	)
	issues = appendProjectionMismatch(
		issues,
		index,
		"source_stage",
		row.SourceStage,
		expected.SourceStage,
	)
	issues = appendProjectionMismatch(issues, index, "claim", row.Claim, expected.Claim)
	issues = appendProjectionMismatch(
		issues,
		index,
		"claim_level",
		row.ClaimLevel,
		expected.ClaimLevel,
	)
	issues = appendProjectionMismatch(
		issues,
		index,
		"provenance_class",
		row.ProvenanceClass,
		expected.ProvenanceClass,
	)
	issues = appendProjectionMismatch(issues, index, "owner_id", row.OwnerID, expected.OwnerID)
	if !sameOptionalInt(row.ParamIndex, expected.ParamIndex) {
		issues = append(
			issues,
			fmt.Sprintf(
				"row %d: param_index projection mismatch got %v want %v",
				index,
				optionalIntValue(row.ParamIndex),
				optionalIntValue(expected.ParamIndex),
			),
		)
	}
	issues = appendProjectionMismatch(
		issues,
		index,
		"param_path",
		row.ParamPath,
		expected.ParamPath,
	)
	issues = appendProjectionMismatch(
		issues,
		index,
		"borrow_state",
		row.BorrowState,
		expected.BorrowState,
	)
	issues = appendProjectionMismatch(
		issues,
		index,
		"escape_state",
		row.EscapeState,
		expected.EscapeState,
	)
	issues = appendProjectionMismatch(
		issues,
		index,
		"alias_state",
		row.AliasState,
		expected.AliasState,
	)
	issues = appendProjectionMismatch(
		issues,
		index,
		"unsafe_class",
		row.UnsafeClass,
		expected.UnsafeClass,
	)
	issues = appendProjectionMismatch(
		issues,
		index,
		"allocation_site_id",
		row.AllocationSiteID,
		expected.AllocationSiteID,
	)
	issues = appendProjectionMismatch(
		issues,
		index,
		"planned_storage",
		row.PlannedStorage,
		expected.PlannedStorage,
	)
	issues = appendProjectionMismatch(
		issues,
		index,
		"actual_lowering_storage",
		row.ActualLoweringStorage,
		expected.ActualLoweringStorage,
	)
	issues = appendProjectionMismatch(issues, index, "proof_id", row.ProofID, expected.ProofID)
	issues = appendProjectionMismatch(
		issues,
		index,
		"proof_kind",
		row.ProofKind,
		expected.ProofKind,
	)
	issues = appendProjectionMismatch(
		issues,
		index,
		"proof_subject_base_id",
		row.ProofSubjectBaseID,
		expected.ProofSubjectBaseID,
	)
	issues = appendProjectionMismatch(
		issues,
		index,
		"proof_index_value_id",
		row.ProofIndexValueID,
		expected.ProofIndexValueID,
	)
	issues = appendProjectionMismatch(
		issues,
		index,
		"proof_operation",
		row.ProofOperation,
		expected.ProofOperation,
	)
	issues = appendProjectionMismatch(
		issues,
		index,
		"proof_range",
		row.ProofRange,
		expected.ProofRange,
	)
	issues = appendProjectionMismatch(
		issues,
		index,
		"validator_name",
		row.ValidatorName,
		expected.ValidatorName,
	)
	issues = appendProjectionMismatch(
		issues,
		index,
		"validator_status",
		row.ValidatorStatus,
		expected.ValidatorStatus,
	)
	issues = appendProjectionMismatch(
		issues,
		index,
		"cost_class",
		row.CostClass,
		expected.CostClass,
	)
	issues = appendProjectionMismatch(
		issues,
		index,
		"normal_build_check",
		row.NormalBuildCheck,
		expected.NormalBuildCheck,
	)
	issues = appendProjectionMismatch(issues, index, "reason", row.Reason, expected.Reason)
	return issues
}

func appendProjectionMismatch[T comparable](
	issues []string,
	index int,
	field string,
	got T,
	want T,
) []string {
	if got != want {
		return append(
			issues,
			fmt.Sprintf("row %d: %s projection mismatch got %v want %v", index, field, got, want),
		)
	}
	return issues
}

func sameOptionalInt(got *int, want *int) bool {
	if got == nil || want == nil {
		return got == nil && want == nil
	}
	return *got == *want
}

func optionalIntValue(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func isConservativeUnknownFact(fact Fact) bool {
	if fact.ProvenanceClass != ProvenanceUnsafeUnknown && fact.UnsafeClass != UnsafeUnknown {
		return false
	}
	return true
}

func validateReportRow(index int, row ReportRow) []string {
	prefix := fmt.Sprintf("row %d", index)
	var issues []string
	if strings.TrimSpace(string(row.SourceFactID)) == "" {
		issues = append(issues, prefix+": source_fact_id is required")
	}
	if strings.TrimSpace(row.SiteID) == "" {
		issues = append(issues, prefix+": site_id is required")
	}
	if strings.TrimSpace(row.Claim) == "" {
		issues = append(issues, prefix+": claim is required")
	} else if !knownReportClaim(row.Claim) {
		issues = append(issues, fmt.Sprintf("%s: unknown memory report claim %q", prefix, row.Claim))
	}
	if !knownSourceStage(row.SourceStage) {
		issues = append(issues, fmt.Sprintf("%s: unknown source_stage %q", prefix, row.SourceStage))
	}
	if !knownProvenanceClass(row.ProvenanceClass) {
		issues = append(
			issues,
			fmt.Sprintf("%s: unknown provenance_class %q", prefix, row.ProvenanceClass),
		)
	}
	if !knownUnsafeClass(row.UnsafeClass) {
		issues = append(issues, fmt.Sprintf("%s: unknown unsafe_class %q", prefix, row.UnsafeClass))
	}
	if !knownClaimLevel(row.ClaimLevel) {
		issues = append(issues, fmt.Sprintf("%s: unknown claim_level %q", prefix, row.ClaimLevel))
	}
	if !knownValidatorStatus(row.ValidatorStatus) {
		issues = append(
			issues,
			fmt.Sprintf("%s: unknown validator_status %q", prefix, row.ValidatorStatus),
		)
	}
	if !knownAliasState(row.AliasState) {
		issues = append(issues, fmt.Sprintf("%s: unknown alias_state %q", prefix, row.AliasState))
	}
	if !knownCostClass(row.CostClass) {
		issues = append(issues, fmt.Sprintf("%s: unknown cost_class %q", prefix, row.CostClass))
	}
	if islandBackedReportRow(row) && row.Epoch <= 0 {
		issues = append(issues, prefix+": island-backed row requires positive epoch")
	}
	if row.Epoch > 0 && strings.TrimSpace(row.IslandID) == "" {
		issues = append(issues, prefix+": epoch requires island_id")
	}
	if strings.TrimSpace(row.IslandID) != "" && strings.TrimSpace(row.BaseID) == "" {
		issues = append(issues, prefix+": island_id requires base_id")
	}
	if dynamicRawRuntimeCheckCostDisallowed(row.Claim, row.CostClass) {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s: dynamic raw check claim %q must use %s, got %s",
				prefix,
				row.Claim,
				CostDynamicCheckRequired,
				row.CostClass,
			),
		)
	}
	if memoryvocab.ZeroCostProvenClaimDisallowed(
		row.Claim,
		string(row.CostClass),
		string(row.ClaimLevel),
		string(row.PlannedStorage),
		string(row.ActualLoweringStorage),
	) {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s: zero_cost_proven requires validated compiler-owned proof for claim %q",
				prefix,
				row.Claim,
			),
		)
	}
	if row.CostClass == CostDynamicCheckRequired && !row.NormalBuildCheck {
		issues = append(issues, prefix+": dynamic_check_required requires normal_build_check")
	}
	if row.Claim == "protocol_dispatch_report_integrity" {
		if row.CostClass != CostDynamicCheckRequired {
			issues = append(
				issues,
				prefix+": protocol_dispatch_report_integrity requires cost_class dynamic_check_required",
			)
		}
		if !row.NormalBuildCheck {
			issues = append(
				issues,
				prefix+": protocol_dispatch_report_integrity requires normal_build_check",
			)
		}
	}
	if row.ClaimLevel == ClaimValidated && row.ValidatorStatus != ValidatorPass {
		issues = append(issues, prefix+": validated claim requires validator_status pass")
	}
	if row.ClaimLevel == ClaimValidated && strings.TrimSpace(string(row.SourceFactID)) == "" {
		issues = append(issues, prefix+": validated claim requires source_fact_id")
	}
	if row.ClaimLevel == ClaimValidated && strings.TrimSpace(row.ValidatorName) == "" {
		issues = append(issues, prefix+": validated claim requires validator_name")
	}
	if row.ClaimLevel == ClaimValidated && boundsTypedProofClaim(row.Claim) {
		if missing := missingTypedProofFields(
			row.ProofID,
			row.ProofKind,
			row.ProofSubjectBaseID,
			row.ProofIndexValueID,
			row.ProofOperation,
			row.ProofRange,
		); len(
			missing,
		) > 0 {
			issues = append(
				issues,
				fmt.Sprintf(
					"%s: validated bounds proof claim %q requires typed proof fields: %s",
					prefix,
					row.Claim,
					strings.Join(missing, ", "),
				),
			)
		}
	}
	if row.ClaimLevel == ClaimValidated &&
		memoryvocab.IslandKernelClaimValidatorMismatch(row.Claim, row.ValidatorName) {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s: validated island claim %q requires validator_name %q",
				prefix,
				row.Claim,
				memoryvocab.RequiredIslandKernelClaimValidator(row.Claim),
			),
		)
	}
	if hasSafeProvenanceFromUnsafeUnknown(row.ProvenanceClass, row.UnsafeClass) {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s: %s claim cannot be sourced from unsafe_unknown",
				prefix,
				row.ProvenanceClass,
			),
		)
	}
	if hasUnsafeUnknownClass(row.ProvenanceClass, row.UnsafeClass) &&
		unsafeUnknownOptimizationClaim(row.Claim, row.AliasState) {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s: unsafe_unknown cannot authorize optimization claim %q",
				prefix,
				row.Claim,
			),
		)
	}
	if broadNoAliasClaim(row.Claim) {
		issues = append(
			issues,
			fmt.Sprintf("%s: broad noalias claim %q is outside Memory Ideal v0", prefix, row.Claim),
		)
	}
	if row.ClaimLevel == ClaimValidated && conservativeNoAliasBoundaryClaim(row.Claim) {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s: conservative noalias boundary claim %q cannot be validated",
				prefix,
				row.Claim,
			),
		)
	}
	if claimRequiresParentFactID(row.Claim) && row.ParentFactID == "" {
		issues = append(
			issues,
			fmt.Sprintf("%s: derived claim %q requires parent_fact_id", prefix, row.Claim),
		)
	}
	if row.Claim == "copy_owned" && row.ProvenanceClass != ProvenanceSafeOwned {
		issues = append(
			issues,
			fmt.Sprintf("%s: copy_owned requires safe_owned provenance", prefix),
		)
	}
	if hasUnsafeUnknownClass(row.ProvenanceClass, row.UnsafeClass) &&
		row.CostClass == CostZeroCostProven {
		issues = append(
			issues,
			fmt.Sprintf("%s: unsafe_unknown cannot claim %s", prefix, row.CostClass),
		)
	}
	if row.CostClass == CostDynamicCheckRequired &&
		memoryOptimizationClaim(row.Claim, row.AliasState) &&
		!row.NormalBuildCheck {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s: dynamic_check_required optimization claim %q requires normal_build_check",
				prefix,
				row.Claim,
			),
		)
	}
	if bareBoundsCheckEliminatedClaim(row.Claim) {
		issues = append(
			issues,
			fmt.Sprintf(
				("%s: bounds_check_eliminated requires compiler-owned proof id; "+
					"use bounds_check_removed_with_proof_id"),
				prefix,
			),
		)
	}
	if unsafeVerifiedRootDisallowedClaim(row.ProvenanceClass, row.UnsafeClass, row.Claim) {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s: unsafe_verified_root cannot claim %q without bounded raw metadata",
				prefix,
				row.Claim,
			),
		)
	}
	if unsafeCheckedDisallowedClaim(row.ProvenanceClass, row.UnsafeClass, row.Claim) {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s: unsafe_checked cannot claim %q outside checked runtime/bounds evidence",
				prefix,
				row.Claim,
			),
		)
	}
	if capMemDisallowedProofClaim(row.Claim, row.ValidatorName, row.Reason) {
		issues = append(
			issues,
			fmt.Sprintf(
				("%s: cap.mem authorization cannot claim %q; cap.mem authorizes "+
					"raw operations only and does not prove pointer validity, bounds, "+
					"ownership, noalias, or safe provenance"),
				prefix,
				row.Claim,
			),
		)
	}
	if row.ClaimLevel == ClaimValidated &&
		unsafeExternalRootTrustedStorage(
			row.ProvenanceClass,
			row.UnsafeClass,
			row.PlannedStorage,
			row.ActualLoweringStorage,
		) {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s: unsafe/external root %s/%s cannot validate trusted storage lowering %q/%q",
				prefix,
				row.ProvenanceClass,
				row.UnsafeClass,
				row.PlannedStorage,
				row.ActualLoweringStorage,
			),
		)
	}
	if row.ClaimLevel == ClaimValidated && strings.Contains(row.Claim, "no_alias") &&
		!validatedNoAliasState(row.AliasState) {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s: validated no_alias requires unique or mutable_exclusive alias_state, got %q",
				prefix,
				row.AliasState,
			),
		)
	}
	if row.ParamIndex != nil && *row.ParamIndex < 0 {
		issues = append(issues, fmt.Sprintf("%s: negative param_index %d", prefix, *row.ParamIndex))
	}
	issues = append(issues, validateReportRowStorage(index, row)...)
	if reportRowRequiresArtifact(row) && row.LoweredArtifactID == "" {
		issues = append(issues, prefix+": storage/lowering claim requires lowered_artifact_id")
	}
	if row.ClaimLevel == ClaimValidated &&
		validatedTrustedStorageHeapFallback(row.PlannedStorage, row.ActualLoweringStorage) {
		issues = append(
			issues,
			fmt.Sprintf("%s: validated %s claim cannot lower as Heap", prefix, row.PlannedStorage),
		)
	}
	if row.ClaimLevel == ClaimValidated &&
		runtimeProofRequiredStorage(row.PlannedStorage, row.ActualLoweringStorage) {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s: validated runtime boundary storage %q/%q requires production runtime proof",
				prefix,
				row.PlannedStorage,
				row.ActualLoweringStorage,
			),
		)
	}
	if storageFallbackRequiresReason(
		row.PlannedStorage,
		row.ActualLoweringStorage,
		row.CostClass,
	) &&
		strings.TrimSpace(row.Reason) == "" {
		issues = append(issues, prefix+": storage/conservative fallback requires reason")
	}
	return issues
}

func validateReportRowStorage(index int, row ReportRow) []string {
	prefix := fmt.Sprintf("row %d", index)
	var issues []string
	if (row.PlannedStorage == "") != (row.ActualLoweringStorage == "") {
		issues = append(
			issues,
			prefix+": planned_storage and actual_lowering_storage must be present together",
		)
	}
	if row.PlannedStorage != "" && !knownStorageClass(row.PlannedStorage) {
		issues = append(
			issues,
			fmt.Sprintf("%s: unknown planned_storage %q", prefix, row.PlannedStorage),
		)
	}
	if row.ActualLoweringStorage != "" && !knownStorageClass(row.ActualLoweringStorage) {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s: unknown actual_lowering_storage %q",
				prefix,
				row.ActualLoweringStorage,
			),
		)
	}
	return issues
}

func islandBackedReportRow(row ReportRow) bool {
	return strings.TrimSpace(row.IslandID) != "" || row.PlannedStorage == StorageExplicitIsland ||
		row.ActualLoweringStorage == StorageExplicitIsland
}

func reportRowRequiresArtifact(row ReportRow) bool {
	return memoryvocab.RowRequiresArtifact(
		string(row.PlannedStorage),
		string(row.ActualLoweringStorage),
		row.Claim,
	)
}

func boundsTypedProofClaim(claim string) bool {
	return claim == "bounds_proof_id" || claim == "bounds_check_removed_with_proof_id"
}

func missingTypedProofFields(
	proofID string,
	proofKind string,
	subjectBaseID string,
	indexValueID string,
	operation string,
	proofRange string,
) []string {
	var missing []string
	if strings.TrimSpace(proofID) == "" {
		missing = append(missing, "proof_id")
	}
	if strings.TrimSpace(proofKind) == "" {
		missing = append(missing, "proof_kind")
	}
	if strings.TrimSpace(subjectBaseID) == "" {
		missing = append(missing, "proof_subject_base_id")
	}
	if strings.TrimSpace(indexValueID) == "" {
		missing = append(missing, "proof_index_value_id")
	}
	if strings.TrimSpace(operation) == "" {
		missing = append(missing, "proof_operation")
	}
	if strings.TrimSpace(proofRange) == "" {
		missing = append(missing, "proof_range")
	}
	return missing
}
