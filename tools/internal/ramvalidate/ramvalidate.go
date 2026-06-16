package ramvalidate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

const ReportSchemaV1 = "tetra.ram-contract-report.v1"
const GradeReportSchemaV1 = "tetra.memory-grade-report.v1"
const ProofStoreSummarySchemaV1 = "tetra.proof-store-summary.v1"
const PipelineCoverageSchemaV1 = "tetra.validation-pipeline-coverage.v1"
const BlockerReportSchemaV1 = "tetra.ram-blockers.v1"

type Report struct {
	SchemaVersion string         `json:"schema_version"`
	GitHead       string         `json:"git_head,omitempty"`
	Target        string         `json:"target"`
	GeneratedBy   string         `json:"generated_by"`
	GeneratedAt   string         `json:"generated_at,omitempty"`
	Functions     []FunctionRow  `json:"functions,omitempty"`
	Rows          []Row          `json:"rows"`
	Proofs        []ProofSummary `json:"proofs,omitempty"`
	Summary       Summary        `json:"summary"`
	NonClaims     []string       `json:"non_claims"`
}

type Row struct {
	SiteID           string        `json:"site_id"`
	ValueID          string        `json:"value_id"`
	Function         string        `json:"function"`
	SourceSpan       string        `json:"source_span,omitempty"`
	Intent           string        `json:"intent"`
	RequestedBytes   int64         `json:"requested_bytes"`
	Bounded          bool          `json:"bounded"`
	Owner            string        `json:"owner"`
	Lifetime         string        `json:"lifetime"`
	EscapeStatus     string        `json:"escape_status"`
	Placement        string        `json:"placement"`
	ProofIDs         []string      `json:"proof_ids"`
	Blockers         []string      `json:"blockers"`
	CopyReason       string        `json:"copy_reason,omitempty"`
	FreePoint        string        `json:"free_point,omitempty"`
	ContractGrade    string        `json:"contract_grade"`
	ValidationStatus string        `json:"validation_status"`
	SourceFactID     string        `json:"source_fact_id,omitempty"`
	Domain           *MemoryDomain `json:"domain,omitempty"`
}

type MemoryDomain struct {
	DomainID       string `json:"domain_id"`
	ParentDomainID string `json:"parent_domain_id,omitempty"`
	Kind           string `json:"kind"`
	OwnerKind      string `json:"owner_kind"`
	OwnerID        string `json:"owner_id"`
	Lifetime       string `json:"lifetime"`
	BudgetBytes    int64  `json:"budget_bytes,omitempty"`
	RequestedBytes int64  `json:"requested_bytes,omitempty"`
	ReservedBytes  int64  `json:"reserved_bytes,omitempty"`
	CommittedBytes int64  `json:"committed_bytes,omitempty"`
	ReleasedBytes  int64  `json:"released_bytes,omitempty"`
	CurrentBytes   int64  `json:"current_bytes,omitempty"`
	PeakBytes      int64  `json:"peak_bytes,omitempty"`
	CopyCount      int    `json:"copy_count,omitempty"`
	BytesCopied    int64  `json:"bytes_copied,omitempty"`
}

type ProofSummary struct {
	ProofID    string `json:"proof_id"`
	Kind       string `json:"kind"`
	Subject    string `json:"subject"`
	StableHash string `json:"stable_hash"`
	Status     string `json:"status"`
}

type FunctionRow struct {
	Function    string `json:"function"`
	Grade       string `json:"grade"`
	RowCount    int    `json:"row_count"`
	HeapRows    int    `json:"heap_rows"`
	CopyRows    int    `json:"copy_rows"`
	BudgetBytes int64  `json:"budget_bytes"`
}

type Summary struct {
	RowCount      int                   `json:"row_count"`
	ArtifactGrade string                `json:"artifact_grade"`
	HeapRows      int                   `json:"heap_rows"`
	CopyRows      int                   `json:"copy_rows"`
	UnboundedRows int                   `json:"unbounded_rows"`
	BudgetBytes   int64                 `json:"budget_bytes"`
	Domains       []MemoryDomainSummary `json:"domains,omitempty"`
}

type MemoryDomainSummary struct {
	DomainID       string `json:"domain_id"`
	ParentDomainID string `json:"parent_domain_id,omitempty"`
	Kind           string `json:"kind"`
	OwnerKind      string `json:"owner_kind"`
	OwnerID        string `json:"owner_id"`
	Lifetime       string `json:"lifetime"`
	RowCount       int    `json:"row_count"`
	BudgetBytes    int64  `json:"budget_bytes,omitempty"`
	RequestedBytes int64  `json:"requested_bytes,omitempty"`
	ReservedBytes  int64  `json:"reserved_bytes,omitempty"`
	CommittedBytes int64  `json:"committed_bytes,omitempty"`
	ReleasedBytes  int64  `json:"released_bytes,omitempty"`
	CurrentBytes   int64  `json:"current_bytes,omitempty"`
	PeakBytes      int64  `json:"peak_bytes,omitempty"`
	CopyCount      int    `json:"copy_count,omitempty"`
	BytesCopied    int64  `json:"bytes_copied,omitempty"`
}

type GradeReport struct {
	SchemaVersion string        `json:"schema_version"`
	GitHead       string        `json:"git_head,omitempty"`
	Target        string        `json:"target"`
	GeneratedBy   string        `json:"generated_by"`
	ArtifactGrade string        `json:"artifact_grade"`
	Functions     []FunctionRow `json:"functions"`
	Summary       Summary       `json:"summary"`
	NonClaims     []string      `json:"non_claims"`
}

type ProofStoreSummary struct {
	SchemaVersion string         `json:"schema_version"`
	GitHead       string         `json:"git_head,omitempty"`
	Target        string         `json:"target"`
	GeneratedBy   string         `json:"generated_by"`
	Proofs        []ProofSummary `json:"proofs"`
	Summary       struct {
		ProofCount   int `json:"proof_count"`
		Proven       int `json:"proven"`
		Conservative int `json:"conservative"`
		Rejected     int `json:"rejected"`
		Unknown      int `json:"unknown"`
	} `json:"summary"`
	NonClaims []string `json:"non_claims"`
}

type PipelineCoverageReport struct {
	SchemaVersion string          `json:"schema_version"`
	GitHead       string          `json:"git_head,omitempty"`
	Target        string          `json:"target"`
	GeneratedBy   string          `json:"generated_by"`
	Entries       []PipelineEntry `json:"entries"`
	NonClaims     []string        `json:"non_claims"`
}

type PipelineEntry struct {
	Entrypoint   string   `json:"entrypoint"`
	ArtifactPath string   `json:"artifact_path,omitempty"`
	Status       string   `json:"status"`
	Validators   []string `json:"validators,omitempty"`
	Exemption    string   `json:"exemption,omitempty"`
}

type BlockerReport struct {
	SchemaVersion string       `json:"schema_version"`
	Kind          string       `json:"kind"`
	GitHead       string       `json:"git_head,omitempty"`
	Target        string       `json:"target"`
	GeneratedBy   string       `json:"generated_by"`
	Rows          []BlockerRow `json:"rows"`
	NonClaims     []string     `json:"non_claims"`
}

type BlockerRow struct {
	SiteID               string   `json:"site_id"`
	Function             string   `json:"function"`
	Intent               string   `json:"intent"`
	Placement            string   `json:"placement"`
	Blockers             []string `json:"blockers,omitempty"`
	CopyReason           string   `json:"copy_reason,omitempty"`
	ContractGrade        string   `json:"contract_grade"`
	File                 string   `json:"file,omitempty"`
	Line                 int      `json:"line,omitempty"`
	Symbol               string   `json:"symbol,omitempty"`
	SourceLocationStatus string   `json:"source_location_status"`
	Severity             string   `json:"severity"`
	Reason               string   `json:"reason"`
	SuggestedFix         string   `json:"suggested_fix"`
	ProofID              string   `json:"proof_id,omitempty"`
	EvidenceID           string   `json:"evidence_id"`
	SafeToOptimize       *bool    `json:"safe_to_optimize"`
	CopyKind             string   `json:"copy_kind,omitempty"`
	SourceValue          string   `json:"source_value,omitempty"`
	DestinationValue     string   `json:"destination_value,omitempty"`
	BytesEstimate        int64    `json:"bytes_estimate,omitempty"`
	SafetyReason         string   `json:"safety_reason,omitempty"`
}

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

func ReadStrictJSONFile(path string, out any) error {
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

func ValidateReportFile(path string) error {
	var report Report
	if err := ReadStrictJSONFile(path, &report); err != nil {
		return err
	}
	return ValidateReport(report)
}

func ValidateReport(report Report) error {
	var issues []string
	if report.SchemaVersion != ReportSchemaV1 {
		issues = append(issues, fmt.Sprintf("schema_version is %q, want %q", report.SchemaVersion, ReportSchemaV1))
	}
	if report.GitHead != "" && report.GitHead != "unknown" && !fullGitHeadRE.MatchString(report.GitHead) {
		issues = append(issues, "git_head must be a 40-character lowercase hex commit or unknown")
	}
	if strings.TrimSpace(report.Target) == "" || strings.TrimSpace(report.GeneratedBy) == "" {
		issues = append(issues, "target and generated_by are required")
	}
	if len(report.Rows) == 0 {
		issues = append(issues, "rows are required")
	}
	proofs := map[string]ProofSummary{}
	for _, proof := range report.Proofs {
		proofs[proof.ProofID] = proof
	}
	for i, row := range report.Rows {
		issues = append(issues, validateRow(i, row, proofs)...)
	}
	expected := SummarizeRows(report.Rows)
	if !reflect.DeepEqual(report.Summary, expected) {
		issues = append(issues, fmt.Sprintf("summary mismatch: got %+v want %+v", report.Summary, expected))
		if report.Summary.ArtifactGrade != expected.ArtifactGrade {
			issues = append(issues, fmt.Sprintf("artifact_grade is %q, want %q", report.Summary.ArtifactGrade, expected.ArtifactGrade))
		}
	}
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func ValidateGradeReportFile(path string) error {
	var report GradeReport
	if err := ReadStrictJSONFile(path, &report); err != nil {
		return err
	}
	var issues []string
	if report.SchemaVersion != GradeReportSchemaV1 {
		issues = append(issues, fmt.Sprintf("schema_version is %q, want %q", report.SchemaVersion, GradeReportSchemaV1))
	}
	if !knownGrade(report.ArtifactGrade) {
		issues = append(issues, fmt.Sprintf("unknown artifact_grade %q", report.ArtifactGrade))
	}
	if report.Summary.ArtifactGrade != report.ArtifactGrade {
		issues = append(issues, "summary artifact_grade must match artifact_grade")
	}
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func ValidateProofStoreSummaryFile(path string) error {
	var report ProofStoreSummary
	if err := ReadStrictJSONFile(path, &report); err != nil {
		return err
	}
	var issues []string
	if report.SchemaVersion != ProofStoreSummarySchemaV1 {
		issues = append(issues, fmt.Sprintf("schema_version is %q, want %q", report.SchemaVersion, ProofStoreSummarySchemaV1))
	}
	if report.Summary.ProofCount != len(report.Proofs) {
		issues = append(issues, "proof_count must match proofs length")
	}
	seen := map[string]bool{}
	counts := map[string]int{}
	for i, proof := range report.Proofs {
		if strings.TrimSpace(proof.ProofID) == "" {
			issues = append(issues, fmt.Sprintf("proof %d: proof_id is required", i))
			continue
		}
		if seen[proof.ProofID] {
			issues = append(issues, fmt.Sprintf("proof %d: duplicate proof_id %q", i, proof.ProofID))
		}
		seen[proof.ProofID] = true
		if strings.TrimSpace(proof.Kind) == "" || strings.TrimSpace(proof.Subject) == "" || strings.TrimSpace(proof.StableHash) == "" {
			issues = append(issues, fmt.Sprintf("proof %s: kind, subject, and stable_hash are required", proof.ProofID))
		}
		switch proof.Status {
		case "proven", "conservative", "rejected", "unknown":
			counts[proof.Status]++
		default:
			issues = append(issues, fmt.Sprintf("proof %s: unknown status %q", proof.ProofID, proof.Status))
		}
		if proof.Status == "proven" && (strings.Contains(strings.ToLower(proof.Kind), "unsafe_unknown") || strings.Contains(strings.ToLower(proof.Subject), "unsafe_unknown")) {
			issues = append(issues, fmt.Sprintf("proof %s: unsafe_unknown cannot be promoted to proven", proof.ProofID))
		}
	}
	if report.Summary.Proven != counts["proven"] {
		issues = append(issues, fmt.Sprintf("proven count is %d, want %d", report.Summary.Proven, counts["proven"]))
	}
	if report.Summary.Conservative != counts["conservative"] {
		issues = append(issues, fmt.Sprintf("conservative count is %d, want %d", report.Summary.Conservative, counts["conservative"]))
	}
	if report.Summary.Rejected != counts["rejected"] {
		issues = append(issues, fmt.Sprintf("rejected count is %d, want %d", report.Summary.Rejected, counts["rejected"]))
	}
	if report.Summary.Unknown != counts["unknown"] {
		issues = append(issues, fmt.Sprintf("unknown count is %d, want %d", report.Summary.Unknown, counts["unknown"]))
	}
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func ValidatePipelineCoverageFile(path string) error {
	var report PipelineCoverageReport
	if err := ReadStrictJSONFile(path, &report); err != nil {
		return err
	}
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
			issues = append(issues, fmt.Sprintf("entry %d missing entrypoint", i))
		}
		if seenEntrypoints[entry.Entrypoint] {
			issues = append(issues, fmt.Sprintf("entry %d duplicate entrypoint %q", i, entry.Entrypoint))
		}
		seenEntrypoints[entry.Entrypoint] = true
		switch entry.Status {
		case "validated_by_pipeline":
			if len(entry.Validators) == 0 {
				issues = append(issues, fmt.Sprintf("entry %d validated_by_pipeline requires validators", i))
			}
			if strings.TrimSpace(entry.ArtifactPath) == "" {
				issues = append(issues, fmt.Sprintf("entry %d validated_by_pipeline requires artifact_path", i))
			}
		case "formal_exemption_with_reason":
			if strings.TrimSpace(entry.Exemption) == "" {
				issues = append(issues, fmt.Sprintf("entry %d missing exemption reason", i))
			} else if !meaningfulPipelineExemption(entry.Exemption) {
				issues = append(issues, fmt.Sprintf("entry %d exemption reason is not specific enough", i))
			}
		case "not_artifact_producing", "legacy_untrusted_path_blocked":
		default:
			issues = append(issues, fmt.Sprintf("entry %d unknown status %q", i, entry.Status))
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

func ValidateBlockerReportFile(path string, kind string) error {
	var report BlockerReport
	if err := ReadStrictJSONFile(path, &report); err != nil {
		return err
	}
	return ValidateBlockerReport(report, kind)
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
			issues = append(issues, fmt.Sprintf("row %d missing site_id/function", i))
		}
		if !knownIntent(row.Intent) {
			issues = append(issues, fmt.Sprintf("row %d unknown intent %q", i, row.Intent))
		}
		if !knownPlacement(row.Placement) {
			issues = append(issues, fmt.Sprintf("row %d unknown placement %q", i, row.Placement))
		}
		if !knownGrade(row.ContractGrade) {
			issues = append(issues, fmt.Sprintf("row %d unknown contract_grade %q", i, row.ContractGrade))
		}
		if kind == "heap" && len(row.Blockers) == 0 {
			issues = append(issues, fmt.Sprintf("row %d heap blocker row requires blockers", i))
		}
		if kind == "copy" && strings.TrimSpace(row.CopyReason) == "" {
			issues = append(issues, fmt.Sprintf("row %d copy row requires copy_reason", i))
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
			issues = append(issues, prefix+" source_location_status available requires file and positive line")
		}
	case "unavailable", "generated", "internal":
	default:
		issues = append(issues, fmt.Sprintf("%s source_location_status %q is required and must be available, unavailable, generated, or internal", prefix, row.SourceLocationStatus))
	}
	if strings.TrimSpace(row.Symbol) == "" {
		issues = append(issues, prefix+" symbol is required")
	}
	switch row.Severity {
	case "P1", "P2", "P3":
	default:
		issues = append(issues, fmt.Sprintf("%s severity %q must be P1, P2, or P3", prefix, row.Severity))
	}
	if strings.TrimSpace(row.Reason) == "" {
		issues = append(issues, prefix+" reason is required")
	}
	if strings.TrimSpace(row.SuggestedFix) == "" {
		issues = append(issues, prefix+" suggested_fix is required")
	}
	if strings.TrimSpace(row.ProofID) == "" && strings.TrimSpace(row.EvidenceID) == "" {
		issues = append(issues, prefix+" proof_id or evidence_id is required")
	}
	if row.SafeToOptimize == nil {
		issues = append(issues, prefix+" safe_to_optimize is required")
	}
	if kind == "copy" {
		if !knownCopyKind(row.CopyKind) {
			issues = append(issues, fmt.Sprintf("%s copy_kind %q is not recognized", prefix, row.CopyKind))
		}
		if strings.TrimSpace(row.SourceValue) == "" {
			issues = append(issues, prefix+" source_value is required")
		}
		if strings.TrimSpace(row.DestinationValue) == "" {
			issues = append(issues, prefix+" destination_value is required")
		}
		if row.BytesEstimate < 0 {
			issues = append(issues, prefix+" bytes_estimate must not be negative")
		}
		if strings.TrimSpace(row.SafetyReason) == "" {
			issues = append(issues, prefix+" safety_reason is required")
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

func validateRow(i int, row Row, proofs map[string]ProofSummary) []string {
	var issues []string
	prefix := fmt.Sprintf("row %d", i)
	if strings.TrimSpace(row.SiteID) == "" {
		issues = append(issues, prefix+": site_id is required")
	}
	if strings.TrimSpace(row.ValueID) == "" || strings.TrimSpace(row.Function) == "" {
		issues = append(issues, prefix+": value_id and function are required")
	}
	if !knownIntent(row.Intent) {
		issues = append(issues, fmt.Sprintf("%s: unknown intent %q", prefix, row.Intent))
	}
	if !knownPlacement(row.Placement) {
		issues = append(issues, fmt.Sprintf("%s: unknown placement %q", prefix, row.Placement))
	}
	if !knownGrade(row.ContractGrade) {
		issues = append(issues, fmt.Sprintf("%s: unknown contract_grade %q", prefix, row.ContractGrade))
	} else if want := GradeForPlacement(row.Placement); row.ContractGrade != want {
		issues = append(issues, fmt.Sprintf("%s: contract_grade %q contradicts placement %q", prefix, row.ContractGrade, row.Placement))
	}
	if isHeap(row.Placement) && len(row.Blockers) == 0 {
		issues = append(issues, prefix+": heap placement requires blocker")
	}
	if isCopy(row.Intent) && row.CopyReason == "" {
		issues = append(issues, prefix+": copy row requires copy_reason")
	}
	issues = append(issues, validateMemoryDomain(prefix, row.Domain)...)
	if trusted(row.Placement) {
		if row.EscapeStatus != "no_escape" {
			issues = append(issues, fmt.Sprintf("%s: trusted placement %q requires no_escape escape_status, got %q", prefix, row.Placement, row.EscapeStatus))
		}
		if row.ValidationStatus != "validated" {
			issues = append(issues, fmt.Sprintf("%s: trusted placement %q requires validated no-escape proof, got validation_status %q", prefix, row.Placement, row.ValidationStatus))
		}
		if len(row.ProofIDs) == 0 {
			issues = append(issues, prefix+": proof_ids are required for validated trusted placement")
		}
	}
	for _, proofID := range row.ProofIDs {
		proof, ok := proofs[proofID]
		if !ok {
			issues = append(issues, fmt.Sprintf("%s: proof_id %q missing from proof summary", prefix, proofID))
			continue
		}
		if proof.Status == "rejected" || proof.Status == "unknown" {
			issues = append(issues, fmt.Sprintf("%s: proof_id %q has unusable status %s", prefix, proofID, proof.Status))
		}
		if trusted(row.Placement) && proof.Status != "proven" {
			issues = append(issues, fmt.Sprintf("%s: trusted placement proof_id %q must be proven, got %s", prefix, proofID, proof.Status))
		}
		issues = append(issues, validateScopedPlacementProof(prefix, row, proofID, proof)...)
	}
	return issues
}

func validateMemoryDomain(prefix string, domain *MemoryDomain) []string {
	if domain == nil {
		return nil
	}
	var issues []string
	if strings.TrimSpace(domain.DomainID) == "" {
		issues = append(issues, prefix+": domain_id is required")
	}
	if !knownDomainKind(domain.Kind) {
		issues = append(issues, fmt.Sprintf("%s: unknown domain kind %q", prefix, domain.Kind))
	}
	if strings.TrimSpace(domain.OwnerKind) == "" {
		issues = append(issues, prefix+": domain owner_kind is required")
	}
	if strings.TrimSpace(domain.OwnerID) == "" {
		issues = append(issues, prefix+": domain owner_id is required")
	}
	if strings.TrimSpace(domain.Lifetime) == "" {
		issues = append(issues, prefix+": domain lifetime is required")
	}
	for name, value := range map[string]int64{
		"budget_bytes":    domain.BudgetBytes,
		"requested_bytes": domain.RequestedBytes,
		"reserved_bytes":  domain.ReservedBytes,
		"committed_bytes": domain.CommittedBytes,
		"released_bytes":  domain.ReleasedBytes,
		"current_bytes":   domain.CurrentBytes,
		"peak_bytes":      domain.PeakBytes,
		"bytes_copied":    domain.BytesCopied,
	} {
		if value < 0 {
			issues = append(issues, fmt.Sprintf("%s: domain %s must not be negative", prefix, name))
		}
	}
	if domain.CopyCount < 0 {
		issues = append(issues, prefix+": domain copy_count must not be negative")
	}
	if domain.PeakBytes < domain.CurrentBytes {
		issues = append(issues, prefix+": domain peak_bytes must be >= current_bytes")
	}
	if domain.BytesCopied > 0 && domain.CopyCount == 0 {
		issues = append(issues, prefix+": domain bytes_copied requires copy_count")
	}
	return issues
}

func validateScopedPlacementProof(prefix string, row Row, proofID string, proof ProofSummary) []string {
	var issues []string
	wantKind := ""
	switch row.Placement {
	case "region":
		wantKind = "region_lifetime_placement"
	case "island":
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

func SummarizeRows(rows []Row) Summary {
	summary := Summary{ArtifactGrade: "M0"}
	domains := map[string]MemoryDomainSummary{}
	for _, row := range rows {
		summary.RowCount++
		summary.ArtifactGrade = MaxGrade(summary.ArtifactGrade, row.ContractGrade)
		if isHeap(row.Placement) {
			summary.HeapRows++
		}
		if isCopy(row.Intent) {
			summary.CopyRows++
		}
		if row.Placement == "heap_unbounded" || row.ContractGrade == "M5" || row.ContractGrade == "M6" {
			summary.UnboundedRows++
		}
		if row.RequestedBytes > 0 {
			summary.BudgetBytes += row.RequestedBytes
		}
		if row.Domain != nil {
			key := memoryDomainSummaryKey(*row.Domain)
			domain := domains[key]
			if domain.DomainID == "" {
				domain.DomainID = row.Domain.DomainID
				domain.ParentDomainID = row.Domain.ParentDomainID
				domain.Kind = row.Domain.Kind
				domain.OwnerKind = row.Domain.OwnerKind
				domain.OwnerID = row.Domain.OwnerID
				domain.Lifetime = row.Domain.Lifetime
			}
			domain.RowCount++
			domain.BudgetBytes += row.Domain.BudgetBytes
			domain.RequestedBytes += row.Domain.RequestedBytes
			domain.ReservedBytes += row.Domain.ReservedBytes
			domain.CommittedBytes += row.Domain.CommittedBytes
			domain.ReleasedBytes += row.Domain.ReleasedBytes
			domain.CurrentBytes += row.Domain.CurrentBytes
			if row.Domain.PeakBytes > domain.PeakBytes {
				domain.PeakBytes = row.Domain.PeakBytes
			}
			domain.CopyCount += row.Domain.CopyCount
			domain.BytesCopied += row.Domain.BytesCopied
			domains[key] = domain
		}
	}
	if len(domains) > 0 {
		keys := make([]string, 0, len(domains))
		for key := range domains {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			summary.Domains = append(summary.Domains, domains[key])
		}
	}
	return summary
}

func memoryDomainSummaryKey(domain MemoryDomain) string {
	return domain.Kind + "\x00" + domain.DomainID + "\x00" + domain.ParentDomainID + "\x00" + domain.OwnerKind + "\x00" + domain.OwnerID + "\x00" + domain.Lifetime
}

func GradeForPlacement(placement string) string {
	switch placement {
	case "eliminated":
		return "M0"
	case "register", "stack":
		return "M1"
	case "static", "interned":
		return "M2"
	case "island", "region":
		return "M3"
	case "heap_bounded":
		return "M4"
	case "heap_unbounded":
		return "M5"
	default:
		return "M6"
	}
}

func MaxGrade(a string, b string) string {
	if rank(b) > rank(a) {
		return b
	}
	return a
}

func rank(grade string) int {
	switch grade {
	case "M0":
		return 0
	case "M1":
		return 1
	case "M2":
		return 2
	case "M3":
		return 3
	case "M4":
		return 4
	case "M5":
		return 5
	case "M6":
		return 6
	default:
		return 7
	}
}

func knownGrade(grade string) bool { return rank(grade) <= 6 }

func knownIntent(intent string) bool {
	switch intent {
	case "allocation", "copy", "intern", "region_alloc", "heap_fallback", "copy_eliminated", "copy_stack_backed",
		"copy_heap_bounded", "copy_heap_unbounded", "copy_required_boundary", "copy_required_mutable_alias", "copy_into_no_allocation":
		return true
	default:
		return false
	}
}

func knownPlacement(placement string) bool {
	switch placement {
	case "eliminated", "register", "stack", "static", "interned", "island", "region", "heap_bounded", "heap_unbounded", "external", "rejected":
		return true
	default:
		return false
	}
}

func knownDomainKind(kind string) bool {
	switch kind {
	case "process", "task", "actor", "island", "request", "external":
		return true
	default:
		return false
	}
}

func trusted(placement string) bool {
	switch placement {
	case "register", "stack", "static", "interned", "island", "region":
		return true
	default:
		return false
	}
}

func isHeap(placement string) bool {
	return placement == "heap_bounded" || placement == "heap_unbounded"
}

func isCopy(intent string) bool {
	return strings.HasPrefix(intent, "copy")
}

func ValidateNonClaims(nonClaims []string) []string {
	if len(nonClaims) == 0 {
		return []string{"non_claims are required"}
	}
	var issues []string
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

func validateNonClaims(nonClaims []string) []string {
	return ValidateNonClaims(nonClaims)
}

func forbiddenClaimWithoutNegation(text string) bool {
	lower := strings.ToLower(text)
	for _, phrase := range []string{
		"memory 100%",
		"full formal proof",
		"official benchmark",
		"fastest language",
		"fastest-language",
		"faster than c",
		"faster than rust",
		"faster-than-c",
		"faster-than-rust",
		"c/rust parity",
		"broad zero-cost performance",
		"zero heap for all programs",
		"zero-copy for all programs",
		"all-target ram parity",
		"full target parity",
		"production actor runtime",
		"production object memory",
		"production persistent memory",
		"arbitrary unsafe external pointer safety",
	} {
		idx := strings.Index(lower, phrase)
		if idx < 0 {
			continue
		}
		prefix := strings.TrimSpace(lower[:idx])
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
