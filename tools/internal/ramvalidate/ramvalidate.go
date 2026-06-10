package ramvalidate

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
	SiteID           string   `json:"site_id"`
	ValueID          string   `json:"value_id"`
	Function         string   `json:"function"`
	SourceSpan       string   `json:"source_span,omitempty"`
	Intent           string   `json:"intent"`
	RequestedBytes   int64    `json:"requested_bytes"`
	Bounded          bool     `json:"bounded"`
	Owner            string   `json:"owner"`
	Lifetime         string   `json:"lifetime"`
	EscapeStatus     string   `json:"escape_status"`
	Placement        string   `json:"placement"`
	ProofIDs         []string `json:"proof_ids"`
	Blockers         []string `json:"blockers"`
	CopyReason       string   `json:"copy_reason,omitempty"`
	FreePoint        string   `json:"free_point,omitempty"`
	ContractGrade    string   `json:"contract_grade"`
	ValidationStatus string   `json:"validation_status"`
	SourceFactID     string   `json:"source_fact_id,omitempty"`
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
	RowCount      int    `json:"row_count"`
	ArtifactGrade string `json:"artifact_grade"`
	HeapRows      int    `json:"heap_rows"`
	CopyRows      int    `json:"copy_rows"`
	UnboundedRows int    `json:"unbounded_rows"`
	BudgetBytes   int64  `json:"budget_bytes"`
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
	SiteID        string   `json:"site_id"`
	Function      string   `json:"function"`
	Intent        string   `json:"intent"`
	Placement     string   `json:"placement"`
	Blockers      []string `json:"blockers,omitempty"`
	CopyReason    string   `json:"copy_reason,omitempty"`
	ContractGrade string   `json:"contract_grade"`
}

var fullGitHeadRE = regexp.MustCompile(`^[0-9a-f]{40}$`)

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
	for i, entry := range report.Entries {
		if strings.TrimSpace(entry.Entrypoint) == "" {
			issues = append(issues, fmt.Sprintf("entry %d missing entrypoint", i))
		}
		switch entry.Status {
		case "validated_by_pipeline":
			if len(entry.Validators) == 0 {
				issues = append(issues, fmt.Sprintf("entry %d validated_by_pipeline requires validators", i))
			}
		case "formal_exemption_with_reason":
			if strings.TrimSpace(entry.Exemption) == "" {
				issues = append(issues, fmt.Sprintf("entry %d missing exemption reason", i))
			}
		case "not_artifact_producing", "legacy_untrusted_path_blocked":
		default:
			issues = append(issues, fmt.Sprintf("entry %d unknown status %q", i, entry.Status))
		}
	}
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func ValidateBlockerReportFile(path string, kind string) error {
	var report BlockerReport
	if err := ReadStrictJSONFile(path, &report); err != nil {
		return err
	}
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
		if kind == "heap" && len(row.Blockers) == 0 {
			issues = append(issues, fmt.Sprintf("row %d heap blocker row requires blockers", i))
		}
		if kind == "copy" && strings.TrimSpace(row.CopyReason) == "" {
			issues = append(issues, fmt.Sprintf("row %d copy row requires copy_reason", i))
		}
	}
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
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
	if trusted(row.Placement) && row.ValidationStatus == "validated" && len(row.ProofIDs) == 0 {
		issues = append(issues, prefix+": proof_ids are required for validated trusted placement")
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
	}
	return issues
}

func SummarizeRows(rows []Row) Summary {
	summary := Summary{ArtifactGrade: "M0"}
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
	}
	return summary
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

func trusted(placement string) bool {
	switch placement {
	case "eliminated", "register", "stack", "static", "interned", "island", "region":
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

func validateNonClaims(nonClaims []string) []string {
	if len(nonClaims) == 0 {
		return []string{"non_claims are required"}
	}
	var issues []string
	for i, claim := range nonClaims {
		if strings.TrimSpace(claim) == "" {
			issues = append(issues, fmt.Sprintf("non_claim %d is empty", i))
		}
	}
	return issues
}
