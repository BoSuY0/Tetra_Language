package ramcontract

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strings"
)

var fullGitHeadRE = regexp.MustCompile(`^[0-9a-f]{40}$`)

var requiredPipelineEntrypoints = []string{
	"BuildFileWithStatsOpt",
	"buildObjectFileWithStatsOpt",
	"buildLibraryObjectWithStatsOpt",
	"InterfaceOnly",
	"wasm32-wasi-build",
	"wasm32-web-build",
	"explain-report-path",
}

func ValidateReport(report Report) error {
	var issues []string
	if report.SchemaVersion != ReportSchemaV1 {
		issues = append(issues, fmt.Sprintf("schema_version is %q, want %q", report.SchemaVersion, ReportSchemaV1))
	}
	if strings.TrimSpace(report.Target) == "" {
		issues = append(issues, "target is required")
	}
	if strings.TrimSpace(report.GeneratedBy) == "" {
		issues = append(issues, "generated_by is required")
	}
	if report.GitHead != "" && report.GitHead != "unknown" && !fullGitHeadRE.MatchString(report.GitHead) {
		issues = append(issues, "git_head must be a 40-character lowercase hex commit or unknown")
	}
	if len(report.Rows) == 0 {
		issues = append(issues, "rows are required; use an explicit M0 summary report for no-allocation artifacts")
	}
	proofs := map[string]ProofSummary{}
	for i, proof := range report.Proofs {
		if strings.TrimSpace(proof.ProofID) == "" {
			issues = append(issues, fmt.Sprintf("proof %d: proof_id is required", i))
			continue
		}
		if _, ok := proofs[proof.ProofID]; ok {
			issues = append(issues, fmt.Sprintf("proof %d: duplicate proof_id %q", i, proof.ProofID))
		}
		proofs[proof.ProofID] = proof
		if strings.TrimSpace(proof.Kind) == "" || strings.TrimSpace(proof.Subject) == "" || strings.TrimSpace(proof.StableHash) == "" {
			issues = append(issues, fmt.Sprintf("proof %s: kind, subject, and stable_hash are required", proof.ProofID))
		}
		switch proof.Status {
		case "proven", "conservative", "rejected", "unknown":
		default:
			issues = append(issues, fmt.Sprintf("proof %s: unknown status %q", proof.ProofID, proof.Status))
		}
	}
	for i, row := range report.Rows {
		issues = append(issues, validateRow(i, row, proofs)...)
	}
	expectedSummary := SummarizeRows(report.Rows)
	if !reflect.DeepEqual(report.Summary, expectedSummary) {
		issues = append(issues, fmt.Sprintf("summary mismatch: got %+v want %+v", report.Summary, expectedSummary))
		if report.Summary.ArtifactGrade != expectedSummary.ArtifactGrade {
			issues = append(issues, fmt.Sprintf("artifact_grade is %q, want %q", report.Summary.ArtifactGrade, expectedSummary.ArtifactGrade))
		}
	}
	if len(report.Functions) > 0 && !reflect.DeepEqual(report.Functions, SummarizeFunctions(report.Rows)) {
		issues = append(issues, "functions summary does not match rows")
	}
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func ValidateReportFile(path string) error {
	var report Report
	if err := readStrictJSONFile(path, &report); err != nil {
		return err
	}
	return ValidateReport(report)
}

func validateRow(index int, row Row, proofs map[string]ProofSummary) []string {
	var issues []string
	prefix := fmt.Sprintf("row %d", index)
	if strings.TrimSpace(row.SiteID) == "" {
		issues = append(issues, prefix+": site_id is required")
	}
	if strings.TrimSpace(row.ValueID) == "" {
		issues = append(issues, prefix+": value_id is required")
	}
	if strings.TrimSpace(row.Function) == "" {
		issues = append(issues, prefix+": function is required")
	}
	if !knownIntent(row.Intent) {
		issues = append(issues, fmt.Sprintf("%s: unknown intent %q", prefix, row.Intent))
	}
	if !knownPlacement(row.Placement) {
		issues = append(issues, fmt.Sprintf("%s: unknown placement %q", prefix, row.Placement))
	}
	if !knownEscapeStatus(row.EscapeStatus) {
		issues = append(issues, fmt.Sprintf("%s: unknown escape_status %q", prefix, row.EscapeStatus))
	}
	if !knownValidationStatus(row.ValidationStatus) {
		issues = append(issues, fmt.Sprintf("%s: unknown validation_status %q", prefix, row.ValidationStatus))
	}
	if !knownGrade(row.ContractGrade) {
		issues = append(issues, fmt.Sprintf("%s: unknown contract_grade %q", prefix, row.ContractGrade))
	} else if want := GradeForPlacement(row.Placement); row.ContractGrade != want {
		issues = append(issues, fmt.Sprintf("%s: contract_grade %q contradicts placement %q, want %q", prefix, row.ContractGrade, row.Placement, want))
	}
	if row.RequestedBytes < 0 {
		issues = append(issues, prefix+": requested_bytes must not be negative")
	}
	if strings.TrimSpace(row.Owner) == "" {
		issues = append(issues, prefix+": owner is required")
	}
	if strings.TrimSpace(row.Lifetime) == "" {
		issues = append(issues, prefix+": lifetime is required")
	}
	if trustedPlacement(row.Placement) {
		if row.EscapeStatus != EscapeNoEscape {
			issues = append(issues, fmt.Sprintf("%s: trusted placement %q requires no_escape escape_status, got %q", prefix, row.Placement, row.EscapeStatus))
		}
		if row.ValidationStatus != ValidationValidated {
			issues = append(issues, fmt.Sprintf("%s: trusted placement %q requires validated no-escape proof, got validation_status %q", prefix, row.Placement, row.ValidationStatus))
		}
		if len(row.ProofIDs) == 0 {
			issues = append(issues, prefix+": proof_ids are required for validated trusted placement")
		}
	}
	if isHeapPlacement(row.Placement) && len(row.Blockers) == 0 {
		issues = append(issues, prefix+": heap placement requires at least one blocker")
	}
	if row.Placement == PlacementHeapUnbounded && row.Bounded {
		issues = append(issues, prefix+": heap_unbounded row cannot be bounded")
	}
	if isCopyIntent(row.Intent) && strings.TrimSpace(row.CopyReason) == "" {
		issues = append(issues, prefix+": copy row requires copy_reason")
	}
	for _, proofID := range row.ProofIDs {
		proof, ok := proofs[proofID]
		if !ok {
			issues = append(issues, fmt.Sprintf("%s: proof_id %q is not present in proof summary", prefix, proofID))
			continue
		}
		if proof.Status == "rejected" || proof.Status == "unknown" {
			issues = append(issues, fmt.Sprintf("%s: proof_id %q has unusable status %s", prefix, proofID, proof.Status))
		}
		if trustedPlacement(row.Placement) && proof.Status != "proven" {
			issues = append(issues, fmt.Sprintf("%s: trusted placement proof_id %q must be proven, got %s", prefix, proofID, proof.Status))
		}
		issues = append(issues, validateScopedPlacementProof(prefix, row, proofID, proof)...)
	}
	return issues
}

func validateScopedPlacementProof(prefix string, row Row, proofID string, proof ProofSummary) []string {
	var issues []string
	wantKind := ""
	switch row.Placement {
	case PlacementRegion:
		wantKind = "region_lifetime_placement"
	case PlacementIsland:
		wantKind = "island_lifetime_placement"
	default:
		return issues
	}
	if proof.Kind != wantKind {
		issues = append(issues, fmt.Sprintf("%s: %s placement proof_id %q must be scoped proof kind %q, got %q", prefix, row.Placement, proofID, wantKind, proof.Kind))
	}
	if row.Lifetime == "" || !strings.Contains(proof.Subject, row.Lifetime) {
		issues = append(issues, fmt.Sprintf("%s: %s placement proof_id %q must bind lifetime %q in proof subject", prefix, row.Placement, proofID, row.Lifetime))
	}
	return issues
}

func ValidateGradeReport(report GradeReport) error {
	var issues []string
	if report.SchemaVersion != GradeReportSchemaV1 {
		issues = append(issues, fmt.Sprintf("schema_version is %q, want %q", report.SchemaVersion, GradeReportSchemaV1))
	}
	if !knownGrade(report.ArtifactGrade) {
		issues = append(issues, fmt.Sprintf("unknown artifact_grade %q", report.ArtifactGrade))
	}
	if report.Summary.ArtifactGrade != report.ArtifactGrade {
		issues = append(issues, fmt.Sprintf("summary artifact_grade is %q, want %q", report.Summary.ArtifactGrade, report.ArtifactGrade))
	}
	for i, fn := range report.Functions {
		if strings.TrimSpace(fn.Function) == "" || !knownGrade(fn.Grade) {
			issues = append(issues, fmt.Sprintf("function %d has invalid name or grade", i))
		}
	}
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func ValidateProofStoreSummary(report ProofStoreSummary) error {
	var issues []string
	if report.SchemaVersion != ProofStoreSummarySchemaV1 {
		issues = append(issues, fmt.Sprintf("schema_version is %q, want %q", report.SchemaVersion, ProofStoreSummarySchemaV1))
	}
	if report.Summary.ProofCount != len(report.Proofs) {
		issues = append(issues, fmt.Sprintf("proof_count is %d, want %d", report.Summary.ProofCount, len(report.Proofs)))
	}
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func ValidatePipelineCoverage(report PipelineCoverageReport) error {
	var issues []string
	if report.SchemaVersion != PipelineCoverageSchemaV1 {
		issues = append(issues, fmt.Sprintf("schema_version is %q, want %q", report.SchemaVersion, PipelineCoverageSchemaV1))
	}
	if len(report.Entries) == 0 {
		issues = append(issues, "entries are required")
	}
	seenEntrypoints := map[string]bool{}
	for i, entry := range report.Entries {
		if strings.TrimSpace(entry.Entrypoint) == "" {
			issues = append(issues, fmt.Sprintf("entry %d: entrypoint is required", i))
		}
		if seenEntrypoints[entry.Entrypoint] {
			issues = append(issues, fmt.Sprintf("entry %d: duplicate entrypoint %q", i, entry.Entrypoint))
		}
		seenEntrypoints[entry.Entrypoint] = true
		switch entry.Status {
		case "validated_by_pipeline":
			if len(entry.Validators) == 0 {
				issues = append(issues, fmt.Sprintf("entry %d: validated_by_pipeline requires validators", i))
			}
			if strings.TrimSpace(entry.ArtifactPath) == "" {
				issues = append(issues, fmt.Sprintf("entry %d: validated_by_pipeline requires artifact_path", i))
			}
		case "formal_exemption_with_reason":
			if strings.TrimSpace(entry.Exemption) == "" {
				issues = append(issues, fmt.Sprintf("entry %d: exemption reason is required", i))
			} else if !meaningfulPipelineExemption(entry.Exemption) {
				issues = append(issues, fmt.Sprintf("entry %d: exemption reason is not specific enough", i))
			}
		case "not_artifact_producing":
		case "legacy_untrusted_path_blocked":
		default:
			issues = append(issues, fmt.Sprintf("entry %d: unknown status %q", i, entry.Status))
		}
	}
	for _, required := range requiredPipelineEntrypoints {
		if !seenEntrypoints[required] {
			issues = append(issues, fmt.Sprintf("missing required pipeline entrypoint %s", required))
		}
	}
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func meaningfulPipelineExemption(reason string) bool {
	reason = strings.TrimSpace(strings.ToLower(reason))
	if len(reason) < 24 {
		return false
	}
	switch reason {
	case "todo", "tbd", "n/a", "none":
		return false
	default:
		return true
	}
}

func ValidateBlockerReport(report BlockerReport, kind string) error {
	var issues []string
	if report.SchemaVersion != BlockerReportSchemaV1 {
		issues = append(issues, fmt.Sprintf("schema_version is %q, want %q", report.SchemaVersion, BlockerReportSchemaV1))
	}
	if report.Kind != kind {
		issues = append(issues, fmt.Sprintf("kind is %q, want %q", report.Kind, kind))
	}
	for i, row := range report.Rows {
		if strings.TrimSpace(row.SiteID) == "" || strings.TrimSpace(row.Function) == "" {
			issues = append(issues, fmt.Sprintf("row %d: site_id and function are required", i))
		}
		if !knownIntent(row.Intent) {
			issues = append(issues, fmt.Sprintf("row %d: unknown intent %q", i, row.Intent))
		}
		if !knownPlacement(row.Placement) {
			issues = append(issues, fmt.Sprintf("row %d: unknown placement %q", i, row.Placement))
		}
		if !knownGrade(row.ContractGrade) {
			issues = append(issues, fmt.Sprintf("row %d: unknown contract_grade %q", i, row.ContractGrade))
		}
		if kind == "heap" && len(row.Blockers) == 0 {
			issues = append(issues, fmt.Sprintf("row %d: heap blocker row requires blockers", i))
		}
		if kind == "copy" && strings.TrimSpace(row.CopyReason) == "" {
			issues = append(issues, fmt.Sprintf("row %d: copy blocker row requires copy_reason", i))
		}
		issues = append(issues, validateActionableBlockerRow(i, row, kind)...)
	}
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateActionableBlockerRow(index int, row BlockerRow, kind string) []string {
	var issues []string
	prefix := fmt.Sprintf("row %d", index)
	switch row.SourceLocationStatus {
	case "available":
		if strings.TrimSpace(row.File) == "" || row.Line <= 0 {
			issues = append(issues, prefix+": source_location_status available requires file and positive line")
		}
	case "unavailable", "generated", "internal":
	default:
		issues = append(issues, fmt.Sprintf("%s: source_location_status %q is required and must be available, unavailable, generated, or internal", prefix, row.SourceLocationStatus))
	}
	if strings.TrimSpace(row.Symbol) == "" {
		issues = append(issues, prefix+": symbol is required")
	}
	switch row.Severity {
	case "P1", "P2", "P3":
	default:
		issues = append(issues, fmt.Sprintf("%s: severity %q must be P1, P2, or P3", prefix, row.Severity))
	}
	if strings.TrimSpace(row.Reason) == "" {
		issues = append(issues, prefix+": reason is required")
	}
	if strings.TrimSpace(row.SuggestedFix) == "" {
		issues = append(issues, prefix+": suggested_fix is required")
	}
	if strings.TrimSpace(row.ProofID) == "" && strings.TrimSpace(row.EvidenceID) == "" {
		issues = append(issues, prefix+": proof_id or evidence_id is required")
	}
	if kind == "copy" {
		if !knownCopyKind(row.CopyKind) {
			issues = append(issues, fmt.Sprintf("%s: copy_kind %q is not recognized", prefix, row.CopyKind))
		}
		if strings.TrimSpace(row.SourceValue) == "" {
			issues = append(issues, prefix+": source_value is required")
		}
		if strings.TrimSpace(row.DestinationValue) == "" {
			issues = append(issues, prefix+": destination_value is required")
		}
		if row.BytesEstimate < 0 {
			issues = append(issues, prefix+": bytes_estimate must not be negative")
		}
		if strings.TrimSpace(row.SafetyReason) == "" {
			issues = append(issues, prefix+": safety_reason is required")
		}
	}
	return issues
}

func knownCopyKind(value string) bool {
	switch value {
	case "HOT_PATH_COPY", "RELEASE_TOOL_COPY", "TEST_ONLY_COPY", "ACCEPTABLE_SMALL_COPY",
		"NEEDS_STREAMING", "NEEDS_CAPACITY_HINT", "NEEDS_ZERO_COPY_OR_BORROWED_VIEW":
		return true
	default:
		return false
	}
}

func readStrictJSONFile(path string, out any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("trailing data after JSON")
		}
		return fmt.Errorf("trailing data after JSON: %w", err)
	}
	return nil
}

func trustedPlacement(placement Placement) bool {
	switch placement {
	case PlacementRegister, PlacementStack, PlacementStatic, PlacementInterned, PlacementIsland, PlacementRegion:
		return true
	default:
		return false
	}
}

func knownGrade(grade MemoryGrade) bool {
	switch grade {
	case GradeM0, GradeM1, GradeM2, GradeM3, GradeM4, GradeM5, GradeM6:
		return true
	default:
		return false
	}
}

func knownIntent(intent Intent) bool {
	switch intent {
	case IntentAllocation, IntentCopy, IntentIntern, IntentRegionAlloc, IntentHeapFallback, IntentCopyEliminated,
		IntentCopyStackBacked, IntentCopyHeapBounded, IntentCopyHeapUnbounded, IntentCopyRequiredBoundary,
		IntentCopyRequiredMutableAlias, IntentCopyIntoNoAllocation:
		return true
	default:
		return false
	}
}

func knownPlacement(placement Placement) bool {
	switch placement {
	case PlacementEliminated, PlacementRegister, PlacementStack, PlacementStatic, PlacementInterned, PlacementIsland,
		PlacementRegion, PlacementHeapBounded, PlacementHeapUnbounded, PlacementExternal, PlacementRejected:
		return true
	default:
		return false
	}
}

func knownEscapeStatus(status EscapeStatus) bool {
	switch status {
	case EscapeNoEscape, EscapeReturn, EscapeCall, EscapeActorCrossing, EscapeTaskCrossing, EscapeFFICrossing,
		EscapeBrowserCrossing, EscapeUnsafe, EscapeUnknown:
		return true
	default:
		return false
	}
}

func knownValidationStatus(status ValidationStatus) bool {
	switch status {
	case ValidationValidated, ValidationConservative, ValidationRejected, ValidationUnknown:
		return true
	default:
		return false
	}
}

func validateNonClaims(nonClaims []string) []string {
	var issues []string
	if len(nonClaims) == 0 {
		return []string{"non_claims are required"}
	}
	for i, claim := range nonClaims {
		trimmed := strings.TrimSpace(claim)
		if trimmed == "" {
			issues = append(issues, fmt.Sprintf("non_claim %d is empty", i))
		}
		if forbiddenClaimWithoutNegation(trimmed) {
			issues = append(issues, fmt.Sprintf("non_claim %d contains forbidden broad claim: %q", i, claim))
		}
	}
	return issues
}

func forbiddenClaimWithoutNegation(text string) bool {
	lower := strings.ToLower(text)
	for _, phrase := range []string{
		"memory 100%",
		"prod_ready_proven",
		"full formal proof",
		"full target parity",
		"production actor runtime",
		"official benchmark",
		"fastest-language",
		"fastest language",
		"faster than c",
		"faster than rust",
		"c/rust parity",
		"broad zero-cost performance",
		"zero heap for all programs",
		"zero-copy for all programs",
		"all-target ram parity",
		"production object memory",
		"production persistent memory",
		"arbitrary unsafe external pointer safety",
		"all-target surface support",
	} {
		if !strings.Contains(lower, phrase) {
			continue
		}
		prefix := strings.TrimSpace(lower[:strings.Index(lower, phrase)])
		if negatedClaimPrefix(prefix) {
			continue
		}
		return true
	}
	return false
}

func negatedClaimPrefix(prefix string) bool {
	for _, allowed := range []string{
		"no",
		"not",
		"not a",
		"not an",
		"without",
		"does not claim",
		"do not claim",
		"nonclaim",
		"non-claim",
	} {
		if strings.HasSuffix(prefix, allowed) || strings.Contains(prefix, allowed+" ") {
			return true
		}
	}
	return false
}

func DefaultNonClaims() []string {
	return []string{
		"no Memory 100% claim",
		"no full formal proof claim",
		"no full target parity claim",
		"no production actor runtime claim",
		"no official benchmark or fastest-language claim",
		"no C/Rust parity or faster-than-C/Rust claim",
		"no broad zero-cost performance claim",
	}
}
