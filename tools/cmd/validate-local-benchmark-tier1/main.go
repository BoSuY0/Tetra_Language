package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	schemaLocalBenchmarkTier1       = "tetra.local_benchmark_tier1.v1"
	scopeP25RealLocalBenchmark      = "p25.0_real_local_benchmark_execution_v1"
	schemaBenchmarkMemoryV1         = "tetra.local_benchmark.memory_evidence.v1"
	schemaLocalRSSBudgetPolicyV1    = "tetra.local_benchmark.rss_budget_policy.v1"
	runtimeFeatureEvidenceClass     = "lowered_ir_static_plan"
	runtimeFeatureEvidenceMethod    = "backend_report_lowered_ir_scan_v1"
	runtimeObjectPlanEvidenceClass  = "native_runtime_object_plan"
	runtimeObjectPlanEvidenceMethod = "native_link_runtime_object_plan_v1"
)

var allowedHeapReasonCodes = map[string]bool{
	"heap.required_escape_return":                true,
	"heap.required_unknown_call":                 true,
	"heap.required_actor_boundary":               true,
	"heap.required_task_boundary":                true,
	"heap.required_dynamic_lifetime":             true,
	"heap.required_large_object":                 true,
	"heap.required_ffi_external":                 true,
	"heap.required_backend_lowering_unavailable": true,
	"heap.required_region_lowering_unavailable":  true,
}

var requiredP20Categories = []string{
	"integer loops",
	"slice sum",
	"bounds-check loops",
	"function calls",
	"recursion",
	"matrix multiply",
	"hash table",
	"allocation",
	"region/island allocation",
	"JSON parse/stringify",
	"HTTP plaintext/json",
	"PostgreSQL single/multiple/update",
	"actor ping-pong",
	"parallel map/reduce",
	"startup time",
	"binary size",
	"compile time",
}

var zeroHeapRequiredCategories = map[string]bool{
	"integer loops":  true,
	"function calls": true,
	"hash table":     true,
	"startup time":   true,
}

var requiredLanguages = []string{"tetra", "c", "cpp", "rust"}

var allowedClassifications = map[string]bool{
	"faster than C/C++/Rust locally":      true,
	"comparable":                          true,
	"slower":                              true,
	"invalid/inconclusive":                true,
	"blocked by missing feature":          true,
	"blocked by fallback backend":         true,
	"blocked by heap allocation":          true,
	"blocked by bounds check":             true,
	"blocked by actor/runtime limitation": true,
}

var allowedStatuses = map[string]bool{
	"measured":     true,
	"build_failed": true,
	"run_failed":   true,
	"blocked":      true,
	"skipped":      true,
}

var allowedBackendPaths = map[string]bool{
	"register": true,
	"stack":    true,
	"fallback": true,
}

var allowedMemoryEvidenceClasses = map[string]bool{
	"runtime_measured":           true,
	"allocation_report_estimate": true,
	"unsupported":                true,
	"blocked":                    true,
}

var allowedMemoryBackendClasses = map[string]bool{
	"none":              true,
	"small_heap":        true,
	"region":            true,
	"large_backend":     true,
	"external":          true,
	"conservative_heap": true,
	"unknown":           true,
}

var allowedMemoryBackendOperations = map[string]bool{
	"reserve":   true,
	"commit":    true,
	"decommit":  true,
	"release":   true,
	"trim":      true,
	"footprint": true,
}

var allowedMemoryDomainKinds = map[string]bool{
	"process":  true,
	"task":     true,
	"actor":    true,
	"island":   true,
	"request":  true,
	"external": true,
}

type tier1Report struct {
	Schema              string              `json:"schema"`
	Scope               string              `json:"scope"`
	GeneratedAt         string              `json:"generated_at"`
	Host                tier1Host           `json:"host"`
	Policy              tier1Policy         `json:"policy"`
	NonClaims           []string            `json:"non_claims"`
	OptimizerValidation optimizerValidation `json:"optimizer_validation"`
	Results             []categoryResult    `json:"results"`
}

type tier1Host struct {
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
	CPUs      int    `json:"cpus"`
	TargetCPU string `json:"target_cpu"`
	GitCommit string `json:"git_commit"`
}

type tier1Policy struct {
	Tier                string  `json:"tier"`
	ComparableThreshold float64 `json:"comparable_threshold"`
	Iterations          int     `json:"iterations"`
}

type localRSSBudgetPolicy struct {
	Schema      string                `json:"schema"`
	Target      string                `json:"target"`
	HostProfile localRSSBudgetHost    `json:"host_profile"`
	Budgets     []localRSSBudgetEntry `json:"budgets"`
	NonClaims   []string              `json:"non_claims"`
}

type localRSSBudgetHost struct {
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
	CPUs      int    `json:"cpus"`
	TargetCPU string `json:"target_cpu"`
	GitCommit string `json:"git_commit,omitempty"`
}

type localRSSBudgetEntry struct {
	Category               string  `json:"category"`
	Language               string  `json:"language"`
	RSSPeakBudgetBytes     uint64  `json:"rss_peak_budget_bytes"`
	AllowedVariancePercent float64 `json:"allowed_variance_percent"`
	Reason                 string  `json:"reason"`
}

type optimizerValidation struct {
	Status   string `json:"status"`
	Artifact string `json:"artifact"`
}

type categoryResult struct {
	Category             string         `json:"category"`
	AlgorithmID          string         `json:"algorithm_id"`
	InputDescription     string         `json:"input_description"`
	Classification       string         `json:"classification"`
	ClassificationReason string         `json:"classification_reason"`
	Rows                 []benchmarkRow `json:"rows"`
}

type benchmarkRow struct {
	Name               string         `json:"name"`
	Category           string         `json:"category"`
	Language           string         `json:"language"`
	Status             string         `json:"status"`
	CompilerVersion    string         `json:"compiler_version"`
	BuildCommand       []string       `json:"build_command"`
	RunCommand         []string       `json:"run_command"`
	SourcePath         string         `json:"source_path"`
	BinaryPath         string         `json:"binary_path"`
	BinarySizeBytes    int64          `json:"binary_size_bytes"`
	CompileTimeMS      float64        `json:"compile_time_ms"`
	RunMeasurementsMS  []float64      `json:"run_measurements_ms"`
	MedianRuntimeMS    float64        `json:"median_runtime_ms"`
	RawOutputArtifacts []string       `json:"raw_output_artifacts"`
	TetraMetadata      *tetraMetadata `json:"tetra_metadata,omitempty"`
	Error              string         `json:"error,omitempty"`
}

type tetraMetadata struct {
	ProofReport                 string                 `json:"proof_report"`
	BoundsReport                string                 `json:"bounds_report"`
	AllocationReport            string                 `json:"allocation_report"`
	PerfBlockerReport           string                 `json:"perf_blocker_report"`
	BackendReport               string                 `json:"backend_report"`
	BackendPath                 string                 `json:"backend_path"`
	RuntimeFeaturesRequired     []string               `json:"runtime_features_required"`
	RuntimeFeaturesLinked       []string               `json:"runtime_features_linked"`
	RuntimeFeaturesInitialized  []string               `json:"runtime_features_initialized"`
	RuntimeLazyInitBlockers     []string               `json:"runtime_lazy_init_blockers"`
	RuntimeFeatureEvidence      runtimeFeatureEvidence `json:"runtime_feature_evidence"`
	RuntimeObjectPlan           runtimeObjectPlan      `json:"runtime_object_plan"`
	BoundsLeft                  int                    `json:"bounds_left"`
	HeapAllocations             int                    `json:"heap_allocations"`
	HeapReasonCodes             []string               `json:"heap_reason_codes"`
	PerfBlockers                []string               `json:"perf_blockers"`
	OptimizerValidationMetadata optimizerValidation    `json:"optimizer_validation_metadata"`
	MemoryEvidence              *memoryEvidence        `json:"memory_evidence,omitempty"`
}

type runtimeFeatureEvidence struct {
	EvidenceClass     string `json:"evidence_class"`
	Method            string `json:"method"`
	SourceArtifact    string `json:"source_artifact,omitempty"`
	BlockedReason     string `json:"blocked_reason,omitempty"`
	UnsupportedReason string `json:"unsupported_reason,omitempty"`
}

type runtimeObjectPlan struct {
	EvidenceClass                    string   `json:"evidence_class"`
	EvidenceMethod                   string   `json:"evidence_method"`
	RuntimeUsed                      bool     `json:"runtime_used"`
	RuntimeObjectLinked              bool     `json:"runtime_object_linked"`
	RuntimeObjectInitialized         bool     `json:"runtime_object_initialized"`
	RuntimeObjectOverride            bool     `json:"runtime_object_override"`
	TimeOnlyRuntime                  bool     `json:"time_only_runtime"`
	LinuxMinimalRuntime              bool     `json:"linux_minimal_runtime"`
	RuntimeObjectFeaturesRequired    []string `json:"runtime_object_features_required"`
	RuntimeObjectFeaturesLinked      []string `json:"runtime_object_features_linked"`
	RuntimeObjectFeaturesInitialized []string `json:"runtime_object_features_initialized"`
	RuntimeObjectLazyInitBlockers    []string `json:"runtime_object_lazy_init_blockers"`
	BlockedReason                    string   `json:"blocked_reason,omitempty"`
	UnsupportedReason                string   `json:"unsupported_reason,omitempty"`
}

type memoryEvidence struct {
	Schema              string             `json:"schema"`
	HeapAllocBytes      memoryMetric       `json:"heap_alloc_bytes"`
	BytesRequested      memoryMetric       `json:"bytes_requested"`
	BytesReserved       memoryMetric       `json:"bytes_reserved"`
	BytesCommitted      memoryMetric       `json:"bytes_committed"`
	BytesReleased       memoryMetric       `json:"bytes_released"`
	BytesCopied         memoryMetric       `json:"bytes_copied"`
	RSSCurrent          memoryMetric       `json:"rss_current"`
	RSSPeak             memoryMetric       `json:"rss_peak"`
	DomainBytesEvidence memoryMetric       `json:"domain_bytes_evidence"`
	DomainBytes         []memoryDomainByte `json:"domain_bytes"`
}

type memoryMetric struct {
	Bytes             uint64 `json:"bytes,omitempty"`
	CurrentBytes      uint64 `json:"current_bytes,omitempty"`
	PeakBytes         uint64 `json:"peak_bytes,omitempty"`
	TotalAllocBytes   uint64 `json:"total_alloc_bytes,omitempty"`
	AllocationCount   uint64 `json:"allocation_count,omitempty"`
	EvidenceClass     string `json:"evidence_class"`
	Method            string `json:"method"`
	SourceArtifact    string `json:"source_artifact,omitempty"`
	UnsupportedReason string `json:"unsupported_reason,omitempty"`
	BlockedReason     string `json:"blocked_reason,omitempty"`
}

type memoryDomainByte struct {
	DomainID             string `json:"domain_id"`
	Kind                 string `json:"kind"`
	RequestedBytes       uint64 `json:"requested_bytes,omitempty"`
	ReservedBytes        uint64 `json:"reserved_bytes,omitempty"`
	CommittedBytes       uint64 `json:"committed_bytes,omitempty"`
	ReleasedBytes        uint64 `json:"released_bytes,omitempty"`
	CurrentBytes         uint64 `json:"current_bytes,omitempty"`
	PeakBytes            uint64 `json:"peak_bytes,omitempty"`
	BytesCopied          uint64 `json:"bytes_copied,omitempty"`
	MailboxCurrentBytes  uint64 `json:"mailbox_current_bytes,omitempty"`
	MailboxPeakBytes     uint64 `json:"mailbox_peak_bytes,omitempty"`
	StackLiveBytes       uint64 `json:"stack_live_bytes,omitempty"`
	StackReservedBytes   uint64 `json:"stack_reserved_bytes,omitempty"`
	StackRetainedBytes   uint64 `json:"stack_retained_bytes,omitempty"`
	StackReleasedBytes   uint64 `json:"stack_released_bytes,omitempty"`
	ByteBudget           uint64 `json:"byte_budget,omitempty"`
	OverBudgetCount      uint64 `json:"over_budget_count,omitempty"`
	BackpressureEvents   uint64 `json:"backpressure_events,omitempty"`
	ActorDomainFieldsSet bool   `json:"-"`
	EvidenceClass        string `json:"evidence_class"`
	Method               string `json:"method"`
	SourceArtifact       string `json:"source_artifact,omitempty"`
}

func (d *memoryDomainByte) UnmarshalJSON(data []byte) error {
	type memoryDomainByteAlias memoryDomainByte
	var alias memoryDomainByteAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*d = memoryDomainByte(alias)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	_, hasMailboxCurrent := raw["mailbox_current_bytes"]
	_, hasMailboxPeak := raw["mailbox_peak_bytes"]
	_, hasStackLive := raw["stack_live_bytes"]
	_, hasStackReserved := raw["stack_reserved_bytes"]
	_, hasStackRetained := raw["stack_retained_bytes"]
	_, hasStackReleased := raw["stack_released_bytes"]
	_, hasByteBudget := raw["byte_budget"]
	_, hasOverBudget := raw["over_budget_count"]
	_, hasBackpressure := raw["backpressure_events"]
	d.ActorDomainFieldsSet = hasMailboxCurrent && hasMailboxPeak &&
		hasStackLive && hasStackReserved && hasStackRetained && hasStackReleased &&
		hasByteBudget && hasOverBudget && hasBackpressure
	return nil
}

func main() {
	reportPath := flag.String("report", "", "P25.0 local Tier 1 benchmark report JSON")
	rssBudgetPolicyPath := flag.String(
		"rss-budget-policy",
		"",
		"optional local RSS budget policy JSON",
	)
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(
			os.Stderr,
			("usage: validate-local-benchmark-tier1 --report reports/local-" +
				"benchmark-tier1-v1/report.json [--rss-budget-policy docs/spec/local-rss-" +
				"budget-policy.local.json]"),
		)
		os.Exit(2)
	}
	raw, err := os.ReadFile(*reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var validateErr error
	if *rssBudgetPolicyPath != "" {
		policyRaw, err := os.ReadFile(*rssBudgetPolicyPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		validateErr = ValidateReportBytesWithRSSBudgetPolicy(raw, root, policyRaw)
	} else {
		validateErr = ValidateReportBytes(raw, root)
	}
	if validateErr != nil {
		fmt.Fprintln(os.Stderr, validateErr)
		os.Exit(1)
	}
}

func ValidateReportBytes(raw []byte, root string) error {
	var report tier1Report
	if err := json.Unmarshal(raw, &report); err != nil {
		return err
	}
	return validateReport(report, root)
}

func ValidateReportBytesWithRSSBudgetPolicy(raw []byte, root string, policyRaw []byte) error {
	var report tier1Report
	if err := json.Unmarshal(raw, &report); err != nil {
		return err
	}
	if err := validateReport(report, root); err != nil {
		return err
	}
	var policy localRSSBudgetPolicy
	if err := json.Unmarshal(policyRaw, &policy); err != nil {
		return err
	}
	return validateLocalRSSBudgetPolicy(report, root, policy)
}

func validateReport(report tier1Report, root string) error {
	if report.Schema != schemaLocalBenchmarkTier1 {
		return fmt.Errorf("schema = %q, want %q", report.Schema, schemaLocalBenchmarkTier1)
	}
	if report.Scope != scopeP25RealLocalBenchmark {
		return fmt.Errorf("scope = %q, want %q", report.Scope, scopeP25RealLocalBenchmark)
	}
	if strings.TrimSpace(report.GeneratedAt) == "" {
		return fmt.Errorf("generated_at is required")
	}
	if err := validateHost(report.Host); err != nil {
		return err
	}
	if err := validateReportGitHead(report.Host.GitCommit, root); err != nil {
		return err
	}
	if err := validatePolicy(report.Policy); err != nil {
		return err
	}
	if err := validateNonClaims(report.NonClaims); err != nil {
		return err
	}
	if err := validateOptimizerValidation(
		"top-level optimizer validation",
		report.OptimizerValidation,
		root,
	); err != nil {
		return err
	}
	seenCategories := map[string]bool{}
	resultsByCategory := map[string]categoryResult{}
	for _, result := range report.Results {
		if strings.TrimSpace(result.Category) == "" {
			return fmt.Errorf("result missing category")
		}
		if seenCategories[result.Category] {
			return fmt.Errorf("duplicate category %q", result.Category)
		}
		seenCategories[result.Category] = true
		resultsByCategory[result.Category] = result
		if err := validateCategoryResult(result, root); err != nil {
			return err
		}
	}
	for _, category := range requiredP20Categories {
		if !seenCategories[category] {
			return fmt.Errorf("missing category %q", category)
		}
	}
	if len(resultsByCategory) != len(requiredP20Categories) {
		return fmt.Errorf(
			"report has %d categories, want %d",
			len(resultsByCategory),
			len(requiredP20Categories),
		)
	}
	return nil
}

func validateHost(host tier1Host) error {
	if host.GOOS == "" || host.GOARCH == "" || host.CPUs <= 0 ||
		strings.TrimSpace(host.TargetCPU) == "" ||
		strings.TrimSpace(host.GitCommit) == "" {
		return fmt.Errorf("host metadata is incomplete: %+v", host)
	}
	return nil
}

func validateReportGitHead(reportHead string, root string) error {
	current, ok := currentGitHead(root)
	if !ok {
		return nil
	}
	if reportHead != current {
		return fmt.Errorf("stale report git_commit = %q, current HEAD = %q", reportHead, current)
	}
	return nil
}

func currentGitHead(root string) (string, bool) {
	cmd := exec.Command("git", "-C", root, "rev-parse", "--verify", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	head := strings.TrimSpace(string(out))
	return head, head != ""
}

func validatePolicy(policy tier1Policy) error {
	if policy.Tier != "tier1_local_benchmark_evidence" {
		return fmt.Errorf("policy tier = %q, want tier1_local_benchmark_evidence", policy.Tier)
	}
	if policy.ComparableThreshold <= 0 || policy.ComparableThreshold > 1 {
		return fmt.Errorf(
			"policy comparable_threshold = %f, want 0 < threshold <= 1",
			policy.ComparableThreshold,
		)
	}
	if policy.Iterations <= 0 {
		return fmt.Errorf("policy iterations = %d, want positive", policy.Iterations)
	}
	return nil
}

func validateNonClaims(nonClaims []string) error {
	required := []string{
		"no fastest-language claim",
		"no official benchmark claim",
		"no cross-machine claim",
		"no TechEmpower claim",
		"no production claim",
	}
	seen := map[string]bool{}
	for _, claim := range nonClaims {
		lower := strings.ToLower(strings.TrimSpace(claim))
		if lower == "" {
			return fmt.Errorf("empty non-claim")
		}
		switch {
		case strings.Contains(lower, "fastest") && !strings.Contains(lower, "no fastest-language"):
			return fmt.Errorf("fastest-language claim is forbidden: %q", claim)
		case strings.Contains(lower, "official") && !strings.Contains(lower, "no official"):
			return fmt.Errorf("official benchmark claim is forbidden: %q", claim)
		case strings.Contains(lower, "cross-machine") && !strings.Contains(lower, "no cross-machine"):
			return fmt.Errorf("cross-machine claim is forbidden: %q", claim)
		case strings.Contains(lower, "techempower") && !strings.Contains(lower, "no techempower"):
			return fmt.Errorf("TechEmpower claim is forbidden: %q", claim)
		case strings.Contains(lower, "production") && !strings.Contains(lower, "no production"):
			return fmt.Errorf("production claim is forbidden: %q", claim)
		}
		seen[lower] = true
	}
	for _, claim := range required {
		if !seen[strings.ToLower(claim)] {
			return fmt.Errorf("missing non-claim %q", claim)
		}
	}
	return nil
}

func validateLocalRSSBudgetPolicy(
	report tier1Report,
	root string,
	policy localRSSBudgetPolicy,
) error {
	if policy.Schema != schemaLocalRSSBudgetPolicyV1 {
		return fmt.Errorf(
			"rss budget policy schema = %q, want %q",
			policy.Schema,
			schemaLocalRSSBudgetPolicyV1,
		)
	}
	if strings.TrimSpace(policy.Target) == "" {
		return fmt.Errorf("rss budget policy target is required")
	}
	if err := validateLocalRSSBudgetNonClaims(policy.NonClaims); err != nil {
		return err
	}
	if err := validateLocalRSSBudgetHost(policy.HostProfile); err != nil {
		return err
	}
	if len(policy.Budgets) == 0 {
		return fmt.Errorf("rss budget policy requires at least one budget")
	}
	for i, budget := range policy.Budgets {
		if err := validateLocalRSSBudgetEntry(i, budget); err != nil {
			return err
		}
	}
	if !localRSSBudgetAppliesToReport(report, policy) {
		return nil
	}
	for _, budget := range policy.Budgets {
		if err := validateLocalRSSBudgetForEntry(report, root, budget); err != nil {
			return err
		}
	}
	return nil
}

func validateLocalRSSBudgetHost(host localRSSBudgetHost) error {
	if strings.TrimSpace(host.GOOS) == "" {
		return fmt.Errorf("rss budget policy host_profile missing goos")
	}
	if strings.TrimSpace(host.GOARCH) == "" {
		return fmt.Errorf("rss budget policy host_profile missing goarch")
	}
	if host.CPUs <= 0 {
		return fmt.Errorf("rss budget policy host_profile cpus must be positive")
	}
	if strings.TrimSpace(host.TargetCPU) == "" {
		return fmt.Errorf("rss budget policy host_profile missing target_cpu")
	}
	return nil
}

func validateLocalRSSBudgetNonClaims(nonClaims []string) error {
	required := []string{
		"local rss budget only",
		"no cross-machine rss claim",
		"no official benchmark claim",
	}
	seen := map[string]bool{}
	for _, claim := range nonClaims {
		lower := strings.ToLower(strings.TrimSpace(claim))
		if lower == "" {
			return fmt.Errorf("rss budget policy non_claims contains an empty entry")
		}
		if strings.Contains(lower, "cross-machine") &&
			!strings.Contains(lower, "no cross-machine") {
			return fmt.Errorf(
				"rss budget policy non_claims contains forbidden cross-machine claim: %q",
				claim,
			)
		}
		if strings.Contains(lower, "official") && !strings.Contains(lower, "no official") {
			return fmt.Errorf(
				"rss budget policy non_claims contains forbidden official claim: %q",
				claim,
			)
		}
		seen[lower] = true
	}
	for _, claim := range required {
		if !seen[claim] {
			return fmt.Errorf("rss budget policy missing non_claim %q", claim)
		}
	}
	return nil
}

func validateLocalRSSBudgetEntry(index int, budget localRSSBudgetEntry) error {
	if strings.TrimSpace(budget.Category) == "" {
		return fmt.Errorf("rss budget policy budgets[%d] missing category", index)
	}
	if budget.Language == "" {
		budget.Language = "tetra"
	}
	if strings.TrimSpace(budget.Language) == "" {
		return fmt.Errorf("rss budget policy budgets[%d] missing language", index)
	}
	if budget.RSSPeakBudgetBytes == 0 {
		return fmt.Errorf(
			"rss budget policy budgets[%d] rss_peak_budget_bytes must be positive",
			index,
		)
	}
	if budget.AllowedVariancePercent < 0 {
		return fmt.Errorf(
			"rss budget policy budgets[%d] allowed_variance_percent must be non-negative",
			index,
		)
	}
	if strings.TrimSpace(budget.Reason) == "" {
		return fmt.Errorf("rss budget policy budgets[%d] missing reason", index)
	}
	return nil
}

func localRSSBudgetAppliesToReport(report tier1Report, policy localRSSBudgetPolicy) bool {
	if policy.Target != targetFromHost(report.Host) {
		return false
	}
	host := policy.HostProfile
	if host.GOOS != report.Host.GOOS || host.GOARCH != report.Host.GOARCH ||
		host.CPUs != report.Host.CPUs ||
		host.TargetCPU != report.Host.TargetCPU {
		return false
	}
	if strings.TrimSpace(host.GitCommit) != "" && host.GitCommit != report.Host.GitCommit {
		return false
	}
	return true
}

func targetFromHost(host tier1Host) string {
	arch := host.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	if host.GOOS == "" || arch == "" {
		return ""
	}
	return host.GOOS + "-" + arch
}

func validateLocalRSSBudgetForEntry(
	report tier1Report,
	root string,
	budget localRSSBudgetEntry,
) error {
	language := budget.Language
	if language == "" {
		language = "tetra"
	}
	row, ok := findBudgetRow(report, budget.Category, language)
	if !ok {
		return fmt.Errorf(
			"rss budget policy category %q language %q has no matching benchmark row",
			budget.Category,
			language,
		)
	}
	if row.Status != "measured" {
		return fmt.Errorf(
			"rss budget policy category %q language %q requires measured row, got %q",
			budget.Category,
			language,
			row.Status,
		)
	}
	if row.TetraMetadata == nil || row.TetraMetadata.MemoryEvidence == nil {
		return fmt.Errorf(
			"rss budget policy category %q language %q missing memory evidence",
			budget.Category,
			language,
		)
	}
	sample, err := validateRSSPeakMetric(row.Name, row.TetraMetadata.MemoryEvidence.RSSPeak, root)
	if err != nil {
		return fmt.Errorf(
			"rss budget policy category %q language %q rss_peak: %w",
			budget.Category,
			language,
			err,
		)
	}
	allowed := localRSSBudgetAllowedPeak(budget)
	if sample.RSSPeakBytes > allowed {
		return fmt.Errorf(
			("rss budget category %q language %q rss_peak = %d bytes exceeds " +
				"local budget %d bytes (allowed %d bytes with %.2f%% variance): %s"),
			budget.Category,
			language,
			sample.RSSPeakBytes,
			budget.RSSPeakBudgetBytes,
			allowed,
			budget.AllowedVariancePercent,
			budget.Reason,
		)
	}
	return nil
}

func findBudgetRow(report tier1Report, category string, language string) (benchmarkRow, bool) {
	for _, result := range report.Results {
		if result.Category != category {
			continue
		}
		for _, row := range result.Rows {
			if row.Language == language {
				return row, true
			}
		}
	}
	return benchmarkRow{}, false
}

func localRSSBudgetAllowedPeak(budget localRSSBudgetEntry) uint64 {
	allowed := float64(budget.RSSPeakBudgetBytes) * (1 + budget.AllowedVariancePercent/100)
	return uint64(math.Ceil(allowed))
}

func validateOptimizerValidation(label string, metadata optimizerValidation, root string) error {
	if strings.TrimSpace(metadata.Status) == "" {
		return fmt.Errorf("%s missing status", label)
	}
	if strings.TrimSpace(metadata.Artifact) == "" {
		return fmt.Errorf("%s missing artifact", label)
	}
	if err := requireExistingPath(root, metadata.Artifact); err != nil {
		return fmt.Errorf("%s artifact: %w", label, err)
	}
	return nil
}

func validateCategoryResult(result categoryResult, root string) error {
	if strings.TrimSpace(result.AlgorithmID) == "" {
		return fmt.Errorf("category %q missing algorithm_id", result.Category)
	}
	if strings.TrimSpace(result.InputDescription) == "" {
		return fmt.Errorf("category %q missing input_description", result.Category)
	}
	if !allowedClassifications[result.Classification] {
		return fmt.Errorf(
			"category %q classification %q is not allowed",
			result.Category,
			result.Classification,
		)
	}
	if strings.TrimSpace(result.ClassificationReason) == "" {
		return fmt.Errorf("category %q missing classification_reason", result.Category)
	}
	seenLanguages := map[string]bool{}
	for _, row := range result.Rows {
		if row.Category != result.Category {
			return fmt.Errorf(
				"row %q category = %q, want %q",
				row.Name,
				row.Category,
				result.Category,
			)
		}
		if seenLanguages[row.Language] {
			return fmt.Errorf(
				"duplicate matrix row for category %q language %q",
				result.Category,
				row.Language,
			)
		}
		seenLanguages[row.Language] = true
		if err := validateBenchmarkRow(row, root); err != nil {
			return err
		}
	}
	for _, language := range requiredLanguages {
		if !seenLanguages[language] {
			return fmt.Errorf(
				"missing matrix row for category %q language %q",
				result.Category,
				language,
			)
		}
	}
	if len(seenLanguages) != len(requiredLanguages) {
		return fmt.Errorf(
			"category %q has %d language rows, want %d",
			result.Category,
			len(seenLanguages),
			len(requiredLanguages),
		)
	}
	return nil
}

func validateBenchmarkRow(row benchmarkRow, root string) error {
	if strings.TrimSpace(row.Name) == "" {
		return fmt.Errorf("benchmark row missing name")
	}
	if !languageAllowed(row.Language) {
		return fmt.Errorf("benchmark %s language %q is not allowed", row.Name, row.Language)
	}
	if !allowedStatuses[row.Status] {
		return fmt.Errorf("benchmark %s status %q is not allowed", row.Name, row.Status)
	}
	if strings.TrimSpace(row.CompilerVersion) == "" {
		return fmt.Errorf("benchmark %s missing compiler_version", row.Name)
	}
	if len(row.BuildCommand) == 0 || strings.TrimSpace(row.BuildCommand[0]) == "" {
		return fmt.Errorf("benchmark %s missing build_command", row.Name)
	}
	if len(row.RunCommand) == 0 || strings.TrimSpace(row.RunCommand[0]) == "" {
		return fmt.Errorf("benchmark %s missing run_command", row.Name)
	}
	if err := requireExistingPath(root, row.SourcePath); err != nil {
		return fmt.Errorf("benchmark %s source_path: %w", row.Name, err)
	}
	if len(row.RawOutputArtifacts) == 0 {
		return fmt.Errorf("benchmark %s missing raw_output_artifacts", row.Name)
	}
	for _, artifact := range row.RawOutputArtifacts {
		if err := requireExistingPath(root, artifact); err != nil {
			return fmt.Errorf("benchmark %s raw artifact: %w", row.Name, err)
		}
	}
	if row.CompileTimeMS < 0 {
		return fmt.Errorf("benchmark %s compile_time_ms is negative", row.Name)
	}
	if row.Status == "measured" {
		if err := requireExistingPath(root, row.BinaryPath); err != nil {
			return fmt.Errorf("benchmark %s binary_path: %w", row.Name, err)
		}
		if row.BinarySizeBytes <= 0 {
			return fmt.Errorf(
				"benchmark %s binary_size_bytes = %d, want positive",
				row.Name,
				row.BinarySizeBytes,
			)
		}
		if len(row.RunMeasurementsMS) == 0 || row.MedianRuntimeMS <= 0 {
			return fmt.Errorf("benchmark %s missing positive runtime measurements", row.Name)
		}
		for _, ms := range row.RunMeasurementsMS {
			if ms < 0 {
				return fmt.Errorf("benchmark %s has negative runtime measurement", row.Name)
			}
		}
	}
	if row.Language == "tetra" {
		if row.TetraMetadata == nil {
			return fmt.Errorf("benchmark %s missing tetra metadata", row.Name)
		}
		if err := validateTetraMetadata(
			row.Name,
			row.Category,
			row.Status,
			*row.TetraMetadata,
			root,
		); err != nil {
			return err
		}
	} else if row.TetraMetadata != nil {
		return fmt.Errorf("benchmark %s non-Tetra row carries tetra metadata", row.Name)
	}
	return nil
}

func validateTetraMetadata(
	name string,
	category string,
	rowStatus string,
	metadata tetraMetadata,
	root string,
) error {
	if rowStatus == "build_failed" &&
		metadata.OptimizerValidationMetadata.Status == "missing_build_artifacts" {
		if !allowedBackendPaths[metadata.BackendPath] {
			return fmt.Errorf(
				"benchmark %s tetra metadata backend_path %q is not allowed",
				name,
				metadata.BackendPath,
			)
		}
		if metadata.BoundsLeft < 0 || metadata.HeapAllocations < 0 {
			return fmt.Errorf("benchmark %s tetra metadata has negative bounds/heap totals", name)
		}
		if err := validateOptimizerValidation(
			"benchmark "+name+" optimizer validation",
			metadata.OptimizerValidationMetadata,
			root,
		); err != nil {
			return err
		}
		if err := validateRuntimeFeatureEvidence(name, rowStatus, metadata, root); err != nil {
			return err
		}
		if err := validateHeapReasonEvidence(name, rowStatus, metadata, root); err != nil {
			return err
		}
		if err := validateAllocationMemoryBackendEvidence(name, rowStatus, metadata, root); err != nil {
			return err
		}
		if err := validateMemoryEvidence(name, rowStatus, metadata.MemoryEvidence, root); err != nil {
			return err
		}
		return validateZeroHeapRequirement(name, category, rowStatus, metadata, root)
	}
	for label, path := range map[string]string{
		"proof report":        metadata.ProofReport,
		"bounds report":       metadata.BoundsReport,
		"allocation report":   metadata.AllocationReport,
		"perf blocker report": metadata.PerfBlockerReport,
		"backend report":      metadata.BackendReport,
	} {
		if err := requireExistingPath(root, path); err != nil {
			return fmt.Errorf("benchmark %s tetra metadata %s: %w", name, label, err)
		}
	}
	if !allowedBackendPaths[metadata.BackendPath] {
		return fmt.Errorf(
			"benchmark %s tetra metadata backend_path %q is not allowed",
			name,
			metadata.BackendPath,
		)
	}
	if metadata.BoundsLeft < 0 || metadata.HeapAllocations < 0 {
		return fmt.Errorf("benchmark %s tetra metadata has negative bounds/heap totals", name)
	}
	if err := validateOptimizerValidation(
		"benchmark "+name+" optimizer validation",
		metadata.OptimizerValidationMetadata,
		root,
	); err != nil {
		return err
	}
	if err := validateRuntimeFeatureEvidence(name, rowStatus, metadata, root); err != nil {
		return err
	}
	if err := validateHeapReasonEvidence(name, rowStatus, metadata, root); err != nil {
		return err
	}
	if err := validateAllocationMemoryBackendEvidence(name, rowStatus, metadata, root); err != nil {
		return err
	}
	if err := validateMemoryEvidence(name, rowStatus, metadata.MemoryEvidence, root); err != nil {
		return err
	}
	return validateZeroHeapRequirement(name, category, rowStatus, metadata, root)
}

func requireExistingPath(root string, path string) error {
	_, err := resolveExistingPath(root, path)
	return err
}

func resolveExistingPath(root string, path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("path is required")
	}
	resolved := path
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(root, path)
	}
	if _, err := os.Stat(resolved); err != nil {
		return "", fmt.Errorf("%s does not exist", path)
	}
	return resolved, nil
}

func languageAllowed(language string) bool {
	for _, supported := range requiredLanguages {
		if language == supported {
			return true
		}
	}
	return false
}

func slug(value string) string {
	replacer := strings.NewReplacer("/", "_", "-", "_")
	return strings.Join(strings.Fields(replacer.Replace(strings.ToLower(value))), "_")
}

func hostDefaults() tier1Host {
	return tier1Host{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, CPUs: runtime.NumCPU()}
}
