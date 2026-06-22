package memorycorev2

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

const SchemaV1 = "tetra.memory-core-v2.evidence.v1"

type Options struct {
	CurrentGitHead string
}

type Report struct {
	Schema                          string                    `json:"schema"`
	GitHead                         string                    `json:"git_head"`
	Target                          string                    `json:"target"`
	ProgramID                       string                    `json:"program_id"`
	MemoryGraphDigest               string                    `json:"memory_graph_digest"`
	ModulePlanDigests               map[string]string         `json:"module_plan_digests"`
	ModuleLoweringDigests           map[string]string         `json:"module_lowering_digests"`
	NormalBuildStateBuilt           bool                      `json:"normal_build_state_built"`
	ReportFlagDecisionParity        bool                      `json:"report_flag_decision_parity"`
	CacheAttestationChecked         bool                      `json:"cache_attestation_checked"`
	IslandRoutesTotal               int                       `json:"island_routes_total"`
	IslandRoutesDirect              int                       `json:"island_routes_direct"`
	MemoryModelOutcomesTotal        int                       `json:"memorymodel_outcomes_total"`
	MemoryModelOutcomesRealPipeline int                       `json:"memorymodel_outcomes_real_pipeline"`
	BackendOperationSupport         []BackendOperationSupport `json:"backend_operation_support"`
	OptimizerMemoryRewrites         int                       `json:"optimizer_memory_rewrites"`
	OptimizerRewritesWithProofIDs   int                       `json:"optimizer_rewrites_with_proof_ids"`
	NegativeGuards                  []NegativeGuard           `json:"negative_guards"`
	NonClaims                       []string                  `json:"nonclaims"`
	FinalSignoff                    bool                      `json:"final_signoff"`
}

type BackendOperationSupport struct {
	Target            string `json:"target"`
	Operation         string `json:"operation"`
	Supported         bool   `json:"supported"`
	Evidence          string `json:"evidence,omitempty"`
	UnsupportedReason string `json:"unsupported_reason,omitempty"`
}

type NegativeGuard struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Evidence string `json:"evidence"`
}

var (
	gitHeadPattern           = regexp.MustCompile(`^[0-9a-f]{40}$`)
	programIDPattern         = regexp.MustCompile(`^program:sha256:[0-9a-f]{64}$`)
	memoryGraphDigestPattern = regexp.MustCompile(`^memory-graph:sha256:[0-9a-f]{64}$`)
	memoryPlanDigestPattern  = regexp.MustCompile(`^memory-plan:sha256:[0-9a-f]{64}$`)
	loweringDigestPattern    = regexp.MustCompile(`^lowering:sha256:[0-9a-f]{64}$`)
)

func ValidateReport(raw []byte, opt Options) error {
	var report Report
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}
	var issues []string
	issues = append(issues, validateEnvelope(report, opt)...)
	issues = append(issues, validateDigests(report)...)
	issues = append(issues, validateBuildPathRequirements(report)...)
	issues = append(issues, validateBackendSupport(report.BackendOperationSupport)...)
	issues = append(issues, validateOptimizerProofs(report)...)
	issues = append(issues, validateNegativeGuards(report.NegativeGuards, report.FinalSignoff)...)
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	if !report.FinalSignoff {
		issues = append(issues, "final_signoff must be true for Memory Core v2 evidence")
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func ValidateReportFile(path string, opt Options) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return ValidateReport(raw, opt)
}

func ValidateClaimFile(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return ValidateClaimText(path, string(raw))
}

func ValidateClaimText(label string, text string) error {
	var issues []string
	for lineNo, line := range strings.Split(text, "\n") {
		if containsForbiddenClaim(line, true) {
			issues = append(
				issues,
				fmt.Sprintf("%s:%d contains forbidden broad Memory Core v2 claim: %q", label, lineNo+1, strings.TrimSpace(line)),
			)
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateEnvelope(report Report, opt Options) []string {
	var issues []string
	if report.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, SchemaV1))
	}
	if !gitHeadPattern.MatchString(report.GitHead) {
		issues = append(issues, "git_head must be a 40-character lowercase hex commit")
	}
	if head := strings.TrimSpace(opt.CurrentGitHead); head != "" && report.GitHead != head {
		issues = append(
			issues,
			fmt.Sprintf("git_head %s does not match current git head %s", report.GitHead, head),
		)
	}
	if strings.TrimSpace(report.Target) != "linux-x64" {
		issues = append(issues, fmt.Sprintf("target is %q, want linux-x64", report.Target))
	}
	if !programIDPattern.MatchString(report.ProgramID) {
		issues = append(issues, "program_id must match program:sha256:<64 hex>")
	}
	return issues
}

func validateDigests(report Report) []string {
	var issues []string
	if !memoryGraphDigestPattern.MatchString(report.MemoryGraphDigest) {
		issues = append(
			issues,
			"memory_graph_digest must match memory-graph:sha256:<64 hex>",
		)
	}
	if len(report.ModulePlanDigests) == 0 {
		issues = append(issues, "module_plan_digests must not be empty")
	}
	if len(report.ModuleLoweringDigests) == 0 {
		issues = append(issues, "module_lowering_digests must not be empty")
	}
	for module, digest := range report.ModulePlanDigests {
		module = strings.TrimSpace(module)
		if module == "" {
			issues = append(issues, "module_plan_digests contains empty module name")
			continue
		}
		if !memoryPlanDigestPattern.MatchString(digest) {
			issues = append(
				issues,
				fmt.Sprintf("module_plan_digests[%s] must match memory-plan:sha256:<64 hex>", module),
			)
		}
		if strings.TrimSpace(report.ModuleLoweringDigests[module]) == "" {
			issues = append(
				issues,
				fmt.Sprintf("module_lowering_digests missing module %s from module_plan_digests", module),
			)
		}
	}
	for module, digest := range report.ModuleLoweringDigests {
		module = strings.TrimSpace(module)
		if module == "" {
			issues = append(issues, "module_lowering_digests contains empty module name")
			continue
		}
		if !loweringDigestPattern.MatchString(digest) {
			issues = append(
				issues,
				fmt.Sprintf("module_lowering_digests[%s] must match lowering:sha256:<64 hex>", module),
			)
		}
		if strings.TrimSpace(report.ModulePlanDigests[module]) == "" {
			issues = append(
				issues,
				fmt.Sprintf("module_plan_digests missing module %s from module_lowering_digests", module),
			)
		}
	}
	return issues
}

func validateBuildPathRequirements(report Report) []string {
	var issues []string
	if !report.NormalBuildStateBuilt {
		issues = append(issues, "normal_build_state_built must be true")
	}
	if !report.ReportFlagDecisionParity {
		issues = append(issues, "report_flag_decision_parity must be true")
	}
	if !report.CacheAttestationChecked {
		issues = append(issues, "cache_attestation_checked must be true")
	}
	if report.IslandRoutesTotal < 16 {
		issues = append(issues, "island_routes_total must be at least 16")
	}
	if report.IslandRoutesDirect != report.IslandRoutesTotal {
		issues = append(
			issues,
			fmt.Sprintf(
				"island route mismatch: direct=%d total=%d",
				report.IslandRoutesDirect,
				report.IslandRoutesTotal,
			),
		)
	}
	if report.MemoryModelOutcomesTotal < 50 {
		issues = append(issues, "memorymodel_outcomes_total must be at least 50")
	}
	if report.MemoryModelOutcomesRealPipeline != report.MemoryModelOutcomesTotal {
		issues = append(
			issues,
			fmt.Sprintf(
				"memorymodel parity incomplete: real_pipeline=%d total=%d",
				report.MemoryModelOutcomesRealPipeline,
				report.MemoryModelOutcomesTotal,
			),
		)
	}
	return issues
}

func validateBackendSupport(rows []BackendOperationSupport) []string {
	var issues []string
	if len(rows) == 0 {
		return []string{"backend_operation_support must not be empty"}
	}
	seen := map[string]bool{}
	linuxSupported := map[string]bool{}
	wasmUnsupported := false
	for i, row := range rows {
		target := strings.TrimSpace(row.Target)
		operation := strings.TrimSpace(row.Operation)
		label := fmt.Sprintf("backend_operation_support[%d]", i)
		if target == "" {
			issues = append(issues, label+" target is required")
		}
		if operation == "" {
			issues = append(issues, label+" operation is required")
		}
		key := target + "/" + operation
		if seen[key] {
			issues = append(issues, fmt.Sprintf("duplicate backend operation support row %s", key))
		}
		seen[key] = true
		if row.Supported {
			if strings.TrimSpace(row.Evidence) == "" {
				issues = append(issues, label+" supported operation evidence is required")
			}
			if !isSupportedBackendOperation(target, operation) {
				issues = append(
					issues,
					fmt.Sprintf("unsupported backend operation %s marked supported", key),
				)
			}
			if target == "linux-x64" {
				linuxSupported[operation] = true
			}
		} else {
			if strings.TrimSpace(row.UnsupportedReason) == "" {
				issues = append(issues, label+" unsupported_reason is required when supported=false")
			}
			if strings.HasPrefix(target, "wasm32-") {
				wasmUnsupported = true
			}
			if isSupportedBackendOperation(target, operation) {
				issues = append(
					issues,
					fmt.Sprintf("required backend operation %s is marked unsupported", key),
				)
			}
		}
	}
	for _, operation := range requiredLinuxOperations() {
		if !linuxSupported[operation] {
			issues = append(
				issues,
				fmt.Sprintf("linux-x64 backend operation %s must be marked supported", operation),
			)
		}
	}
	if !wasmUnsupported {
		issues = append(issues, "backend_operation_support must include an unsupported wasm32 target row")
	}
	return issues
}

func requiredLinuxOperations() []string {
	return []string{"commit", "decommit", "release", "reserve"}
}

func isSupportedBackendOperation(target string, operation string) bool {
	if target != "linux-x64" {
		return false
	}
	switch operation {
	case "reserve", "commit", "decommit", "release":
		return true
	default:
		return false
	}
}

func validateOptimizerProofs(report Report) []string {
	var issues []string
	if report.OptimizerMemoryRewrites < 0 {
		issues = append(issues, "optimizer_memory_rewrites must be non-negative")
	}
	if report.OptimizerRewritesWithProofIDs < 0 {
		issues = append(issues, "optimizer_rewrites_with_proof_ids must be non-negative")
	}
	if report.OptimizerRewritesWithProofIDs < report.OptimizerMemoryRewrites {
		issues = append(
			issues,
			fmt.Sprintf(
				"optimizer proof mismatch: rewrites=%d proof_ids=%d",
				report.OptimizerMemoryRewrites,
				report.OptimizerRewritesWithProofIDs,
			),
		)
	}
	return issues
}

func validateNegativeGuards(guards []NegativeGuard, finalSignoff bool) []string {
	required := map[string]bool{
		"missing-digest":                       false,
		"report-only-state":                    false,
		"route-count-mismatch":                 false,
		"proofless-optimizer-rewrite":          false,
		"unsupported-backend-marked-supported": false,
		"memorymodel-parity-incomplete":        false,
		"broad-claim":                          false,
		"final-signoff-failed-requirement":     false,
	}
	var issues []string
	seen := map[string]bool{}
	failed := false
	for i, guard := range guards {
		name := strings.TrimSpace(guard.Name)
		status := strings.TrimSpace(guard.Status)
		label := fmt.Sprintf("negative_guards[%d]", i)
		if name == "" {
			issues = append(issues, label+" name is required")
			failed = true
			continue
		}
		if seen[name] {
			issues = append(issues, fmt.Sprintf("duplicate negative guard %s", name))
			failed = true
		}
		seen[name] = true
		if _, ok := required[name]; ok {
			required[name] = true
		} else {
			issues = append(issues, fmt.Sprintf("unexpected negative guard %s", name))
			failed = true
		}
		if status != "pass" {
			issues = append(issues, fmt.Sprintf("negative guard %s status is %q, want pass", name, status))
			failed = true
		}
		if strings.TrimSpace(guard.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("negative guard %s evidence is required", name))
			failed = true
		}
	}
	var missing []string
	for name, ok := range required {
		if !ok {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	for _, name := range missing {
		issues = append(issues, fmt.Sprintf("missing negative guard %s", name))
		failed = true
	}
	if finalSignoff && failed {
		issues = append(issues, "final_signoff=true is invalid while a negative guard requirement failed")
	}
	return issues
}

func validateNonClaims(values []string) []string {
	var issues []string
	if len(nonEmptyStrings(values)) == 0 {
		return []string{"nonclaims must not be empty"}
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			issues = append(issues, "nonclaims contains empty entry")
			continue
		}
		if containsForbiddenClaim(value, false) {
			issues = append(issues, fmt.Sprintf("nonclaims contains forbidden broad claim: %q", value))
		}
	}
	required := []string{
		"no universal memory safety",
		"no universal performance",
		"no zero heap",
		"no all target",
	}
	joined := strings.Join(normalizedStrings(values), "\n")
	for _, want := range required {
		if !strings.Contains(joined, want) {
			issues = append(issues, fmt.Sprintf("nonclaims missing required boundary %q", want))
		}
	}
	return issues
}

func containsForbiddenClaim(value string, scanText bool) bool {
	lower := strings.ToLower(value)
	if hasNegation(lower) {
		return false
	}
	phrases := []string{
		"memory is 100% ready",
		"universal memory safety",
		"fully proven memory safety",
		"full formal proof",
		"all-target memory safety",
		"all target memory safety",
		"all targets memory stable",
		"all-target memory support",
		"all target memory support",
		"all-target backend runtime",
		"zero heap for all programs",
		"zero-heap for all programs",
		"zero copy for all programs",
		"zero-copy for all programs",
		"universal performance",
		"faster than c",
		"faster than rust",
	}
	if scanText {
		phrases = append(phrases, "guarantee for every program")
	}
	for _, phrase := range phrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func hasNegation(lower string) bool {
	for _, marker := range []string{
		"no ",
		"not ",
		"does not ",
		"without ",
		"nonclaim",
		"non-claim",
		"nonclaims",
		"unsupported ",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func nonEmptyStrings(values []string) []string {
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func normalizedStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		value = strings.ReplaceAll(value, "-", " ")
		out = append(out, value)
	}
	return out
}

func decodeStrict(raw []byte, out any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("trailing data after Memory Core v2 evidence JSON")
		}
		return fmt.Errorf("trailing data after Memory Core v2 evidence JSON: %w", err)
	}
	return nil
}
