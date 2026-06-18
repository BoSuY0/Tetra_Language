package parallelprod

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const SchemaV1 = "tetra.parallel.production.v1"

type Report struct {
	Schema             string                    `json:"schema"`
	Status             string                    `json:"status"`
	Target             string                    `json:"target"`
	Host               string                    `json:"host"`
	Runtime            string                    `json:"runtime"`
	Source             string                    `json:"source"`
	Processes          []ProcessReport           `json:"processes"`
	Benchmarks         []BenchmarkReport         `json:"benchmarks"`
	ActorMemoryDomains []ActorMemoryDomainReport `json:"actor_memory_domains"`
	Contracts          []ContractReport          `json:"contracts"`
	Cases              []CaseReport              `json:"cases"`
	Diagnostics        []DiagnosticReport        `json:"diagnostics,omitempty"`
	Audit              []AuditReport             `json:"audit"`
}

type ProcessReport struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Ran      bool   `json:"ran"`
	Pass     bool   `json:"pass"`
	ExitCode *int   `json:"exit_code,omitempty"`
}

type ContractReport struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Evidence string `json:"evidence"`
}

type BenchmarkReport struct {
	Name                  string                `json:"name"`
	Kind                  string                `json:"kind"`
	Metric                string                `json:"metric"`
	Unit                  string                `json:"unit"`
	BaselineValue         int                   `json:"baseline_value"`
	MeasuredValue         int                   `json:"measured_value"`
	ImprovementRatio      float64               `json:"improvement_ratio"`
	Evidence              string                `json:"evidence"`
	ClaimTier             string                `json:"claim_tier"`
	Claim                 string                `json:"claim"`
	RawOutputArtifacts    []string              `json:"raw_output_artifacts"`
	Environment           *BenchmarkEnvironment `json:"environment,omitempty"`
	ReproductionArtifacts []string              `json:"reproduction_artifacts,omitempty"`
	Ran                   bool                  `json:"ran"`
	Pass                  bool                  `json:"pass"`
}

type BenchmarkEnvironment struct {
	Host       string `json:"host,omitempty"`
	OS         string `json:"os,omitempty"`
	Arch       string `json:"arch,omitempty"`
	CPU        string `json:"cpu,omitempty"`
	Kernel     string `json:"kernel,omitempty"`
	Command    string `json:"command,omitempty"`
	Repeats    int    `json:"repeats,omitempty"`
	DurationMS int    `json:"duration_ms,omitempty"`
}

type CaseReport struct {
	Name              string `json:"name"`
	Kind              string `json:"kind"`
	Ran               bool   `json:"ran"`
	Pass              bool   `json:"pass"`
	ExpectedError     string `json:"expected_error,omitempty"`
	Iterations        int    `json:"iterations,omitempty"`
	DeterministicSeed string `json:"deterministic_seed,omitempty"`
	MaxDurationMS     int    `json:"max_duration_ms,omitempty"`
	Error             string `json:"error,omitempty"`
}

type DiagnosticReport struct {
	Case          string `json:"case"`
	Code          string `json:"code"`
	Severity      string `json:"severity"`
	Category      string `json:"category"`
	Position      string `json:"position"`
	ExpectedError string `json:"expected_error"`
}

type ActorMemoryDomainReport struct {
	SchemaVersion              string                   `json:"schema_version"`
	ActorID                    string                   `json:"actor_id"`
	EvidenceClass              string                   `json:"evidence_class"`
	EvidenceMethod             string                   `json:"evidence_method"`
	RuntimeMeasured            bool                     `json:"runtime_measured"`
	RuntimeBlockedReason       string                   `json:"runtime_blocked_reason,omitempty"`
	Domain                     MemoryDomainReport       `json:"domain"`
	Mailbox                    ActorMailboxMemoryReport `json:"mailbox"`
	MessagePool                ActorMessagePoolReport   `json:"message_pool"`
	OwnedRegions               []ActorOwnedRegionReport `json:"owned_regions,omitempty"`
	Backpressure               ActorBackpressureReport  `json:"backpressure"`
	NonClaims                  []string                 `json:"non_claims"`
	ProductionRuntimeClaimed   bool                     `json:"production_runtime_claimed"`
	DistributedZeroCopyClaimed bool                     `json:"distributed_zero_copy_claimed"`
}

type MemoryDomainReport struct {
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

type ActorMailboxMemoryReport struct {
	CapacityMessages int    `json:"capacity_messages"`
	QueuedMessages   int    `json:"queued_messages"`
	CapacityBytes    int    `json:"capacity_bytes"`
	QueuedBytes      int    `json:"queued_bytes"`
	PeakQueuedBytes  int    `json:"peak_queued_bytes"`
	ReclaimedBytes   int    `json:"reclaimed_bytes"`
	MessageBytes     int    `json:"message_bytes"`
	BackpressureMode string `json:"backpressure_mode"`
}

type ActorMessagePoolReport struct {
	SlabBytes         int `json:"slab_bytes"`
	LiveBytes         int `json:"live_bytes"`
	ReclaimedBytes    int `json:"reclaimed_bytes"`
	CapacityBytes     int `json:"capacity_bytes"`
	MessageSlotsLive  int `json:"message_slots_live"`
	MessageSlotsLimit int `json:"message_slots_limit"`
}

type ActorOwnedRegionReport struct {
	RegionName string `json:"region_name"`
	DomainID   string `json:"domain_id"`
	OwnerID    string `json:"owner_id"`
	Bytes      int    `json:"bytes"`
}

type ActorBackpressureReport struct {
	Mode   string `json:"mode"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type AuditReport struct {
	Requirement string `json:"requirement"`
	Artifact    string `json:"artifact"`
	Evidence    string `json:"evidence"`
	Result      string `json:"result"`
}

func ValidateReport(raw []byte) error {
	var report Report
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	issues = append(issues, rejectPaperEvidence(raw)...)
	if report.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, SchemaV1))
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if report.Target != "linux-x64" {
		issues = append(issues, fmt.Sprintf("target is %q, want linux-x64", report.Target))
	}
	if report.Host != "linux-x64" {
		issues = append(issues, fmt.Sprintf("host is %q, want linux-x64", report.Host))
	}
	if report.Runtime != "parallel-linux-x64" {
		issues = append(
			issues,
			fmt.Sprintf("runtime is %q, want parallel-linux-x64", report.Runtime),
		)
	}
	if strings.TrimSpace(report.Source) == "" {
		issues = append(issues, "source is required")
	}
	issues = append(issues, validateProcesses(report.Processes)...)
	issues = append(issues, validateBenchmarks(report.Benchmarks)...)
	issues = append(issues, validateActorMemoryDomains(report.ActorMemoryDomains)...)
	issues = append(issues, validateContracts(report.Contracts)...)
	issues = append(issues, validateCases(report.Cases)...)
	issues = append(issues, validateDiagnostics(report.Cases, report.Diagnostics)...)
	issues = append(issues, validateAudit(report.Audit)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func rejectPaperEvidence(raw []byte) []string {
	lower := strings.ToLower(string(raw))
	forbidden := []string{
		"metadata-only",
		"build-only",
		"docs-only",
		"sidecar-only",
		" fake",
		"fake/",
		"\"fake\"",
		" mock",
		"mock/",
		"\"mock\"",
		"placeholder",
	}
	var issues []string
	for _, marker := range forbidden {
		if strings.Contains(lower, marker) {
			issues = append(
				issues,
				fmt.Sprintf(
					"report contains forbidden non-production evidence marker %q",
					strings.Trim(marker, " /\""),
				),
			)
		}
	}
	return issues
}

func validateProcesses(processes []ProcessReport) []string {
	var issues []string
	if len(processes) < 3 {
		issues = append(
			issues,
			fmt.Sprintf(
				"process evidence has %d entries, want build, app, and stress processes",
				len(processes),
			),
		)
	}
	seenBuild := false
	seenApp := false
	seenStress := false
	names := map[string]bool{}
	for _, p := range processes {
		if strings.TrimSpace(p.Name) == "" {
			issues = append(issues, "process name is required")
		} else if names[p.Name] {
			issues = append(issues, fmt.Sprintf("duplicate process %s", p.Name))
		}
		names[p.Name] = true
		switch p.Kind {
		case "build":
			seenBuild = true
		case "app":
			seenApp = true
		case "stress":
			seenStress = true
		case "benchmark":
		default:
			issues = append(
				issues,
				fmt.Sprintf(
					"process %s kind is %q, want build, app, stress, or benchmark",
					p.Name,
					p.Kind,
				),
			)
		}
		if strings.TrimSpace(p.Path) == "" {
			issues = append(issues, fmt.Sprintf("process %s path is required", p.Name))
		}
		if !p.Ran {
			issues = append(issues, fmt.Sprintf("process %s did not run", p.Name))
		}
		if !p.Pass {
			issues = append(issues, fmt.Sprintf("process %s did not pass", p.Name))
		}
		if p.ExitCode == nil {
			issues = append(issues, fmt.Sprintf("process %s missing exit_code", p.Name))
		} else if *p.ExitCode != 0 {
			issues = append(issues, fmt.Sprintf("process %s exit_code = %d, want 0", p.Name, *p.ExitCode))
		}
	}
	if !seenBuild {
		issues = append(issues, "process evidence missing build process")
	}
	if !seenApp {
		issues = append(issues, "process evidence missing executable app process")
	}
	if !seenStress {
		issues = append(issues, "process evidence missing parallel stress process")
	}
	return issues
}

func validateBenchmarks(benchmarks []BenchmarkReport) []string {
	required := map[string]string{
		"actor ping-pong benchmark prep":                    "actor_benchmark_prep",
		"actor fanout/fanin benchmark prep":                 "actor_benchmark_prep",
		"actor mailbox throughput benchmark prep":           "actor_benchmark_prep",
		"actor backpressure latency benchmark prep":         "actor_benchmark_prep",
		"zero_copy_move local typed mailbox benchmark prep": "actor_transfer_prep",
	}
	var issues []string
	if len(benchmarks) == 0 {
		issues = append(issues, "benchmark evidence is required")
	}
	seen := map[string]bool{}
	for _, b := range benchmarks {
		name := strings.TrimSpace(b.Name)
		if name == "" {
			issues = append(issues, "benchmark name is required")
			continue
		}
		if seen[name] {
			issues = append(issues, fmt.Sprintf("duplicate benchmark %s", name))
		}
		seen[name] = true
		if wantKind, ok := required[name]; ok {
			required[name] = ""
			if b.Kind != wantKind {
				issues = append(
					issues,
					fmt.Sprintf("benchmark %s kind is %q, want %s", name, b.Kind, wantKind),
				)
			}
		} else if b.Kind != "scheduler" && b.Kind != "transfer" && b.Kind != "actor_benchmark_prep" && b.Kind != "actor_transfer_prep" {
			issues = append(
				issues,
				fmt.Sprintf(("benchmark %s kind is %q, want scheduler, transfer, actor_"+
					"benchmark_prep, or actor_transfer_prep"), name, b.Kind),
			)
		}
		if strings.TrimSpace(b.Metric) == "" {
			issues = append(issues, fmt.Sprintf("benchmark %s metric is required", name))
		}
		if strings.TrimSpace(b.Unit) == "" {
			issues = append(issues, fmt.Sprintf("benchmark %s unit is required", name))
		}
		if !b.Pass {
			issues = append(issues, fmt.Sprintf("benchmark %s did not pass", name))
		}
		switch b.ClaimTier {
		case "tier0_local_smoke_only":
			if b.Ran {
				issues = append(
					issues,
					fmt.Sprintf("benchmark %s ran=true, want dry-run Tier 0 prep", name),
				)
			}
			if b.BaselineValue != 0 || b.MeasuredValue != 0 || b.ImprovementRatio != 0 {
				issues = append(
					issues,
					fmt.Sprintf(
						"benchmark %s Tier 0 prep must not record measured improvement values",
						name,
					),
				)
			}
		case "tier1_local_benchmark_evidence":
			if !b.Ran {
				issues = append(
					issues,
					fmt.Sprintf("benchmark %s did not run for Tier 1 local evidence", name),
				)
			}
			if b.BaselineValue <= 0 {
				issues = append(
					issues,
					fmt.Sprintf(
						"benchmark %s baseline_value = %d, want positive for Tier 1",
						name,
						b.BaselineValue,
					),
				)
			}
			if b.MeasuredValue < 0 {
				issues = append(
					issues,
					fmt.Sprintf(
						"benchmark %s measured_value = %d, want non-negative",
						name,
						b.MeasuredValue,
					),
				)
			}
		case "tier2_reproducible_cross_machine",
			"tier3_independent_reproduction",
			"tier4_official_upstream_submission":
			issues = append(issues, validateUnsupportedBenchmarkPromotion(name, b)...)
		default:
			issues = append(
				issues,
				fmt.Sprintf(
					"benchmark %s claim_tier is %q, want tier0_local_smoke_only or tier1_local_benchmark_evidence",
					name,
					b.ClaimTier,
				),
			)
		}
		evidence := strings.TrimSpace(b.Evidence)
		if evidence == "" {
			issues = append(issues, fmt.Sprintf("benchmark %s evidence is required", name))
		}
		if len(b.RawOutputArtifacts) == 0 {
			issues = append(
				issues,
				fmt.Sprintf("benchmark %s raw output artifacts are required", name),
			)
		}
		for _, artifact := range b.RawOutputArtifacts {
			if strings.TrimSpace(artifact) == "" {
				issues = append(
					issues,
					fmt.Sprintf("benchmark %s has empty raw output artifact path", name),
				)
			}
		}
		if err := validateBenchmarkClaim(name, b.Claim, evidence); err != nil {
			issues = append(issues, err.Error())
		}
		if name == "actor fanout/fanin benchmark prep" &&
			!strings.Contains(strings.ToLower(evidence), "work stealing") {
			issues = append(
				issues,
				fmt.Sprintf("benchmark %s evidence must mention work stealing", name),
			)
		}
		if name == "zero_copy_move local typed mailbox benchmark prep" &&
			!strings.Contains(evidence, "zero_copy_move") {
			issues = append(
				issues,
				fmt.Sprintf("benchmark %s evidence must mention zero_copy_move", name),
			)
		}
	}
	for name, wantKind := range required {
		if wantKind != "" {
			issues = append(issues, fmt.Sprintf("missing required benchmark %q", name))
		}
	}
	return issues
}

func validateUnsupportedBenchmarkPromotion(name string, b BenchmarkReport) []string {
	tier := benchmarkTierLabel(b.ClaimTier)
	issues := []string{
		fmt.Sprintf(
			("benchmark %s claim_tier %q is beyond actor benchmark readiness " +
				"scope; %s requires a separate reproducible benchmark gate"),
			name,
			b.ClaimTier,
			tier,
		),
	}
	if benchmarkEnvironmentMissing(b.Environment) {
		issues = append(
			issues,
			fmt.Sprintf("benchmark %s %s promotion missing environment metadata", name, tier),
		)
	}
	if len(b.ReproductionArtifacts) == 0 {
		issues = append(
			issues,
			fmt.Sprintf("benchmark %s %s promotion missing reproduction_artifacts", name, tier),
		)
	} else {
		for _, artifact := range b.ReproductionArtifacts {
			if strings.TrimSpace(artifact) == "" {
				issues = append(
					issues,
					fmt.Sprintf("benchmark %s %s promotion has empty reproduction_artifacts entry", name, tier),
				)
			}
		}
	}
	return issues
}

func benchmarkTierLabel(claimTier string) string {
	switch claimTier {
	case "tier2_reproducible_cross_machine":
		return "Tier 2"
	case "tier3_independent_reproduction":
		return "Tier 3"
	case "tier4_official_upstream_submission":
		return "Tier 4"
	default:
		return claimTier
	}
}

func benchmarkEnvironmentMissing(env *BenchmarkEnvironment) bool {
	if env == nil {
		return true
	}
	return strings.TrimSpace(env.Host) == "" &&
		strings.TrimSpace(env.OS) == "" &&
		strings.TrimSpace(env.Arch) == "" &&
		strings.TrimSpace(env.CPU) == "" &&
		strings.TrimSpace(env.Kernel) == "" &&
		strings.TrimSpace(env.Command) == "" &&
		env.Repeats == 0 &&
		env.DurationMS == 0
}

func validateBenchmarkClaim(name string, claim string, evidence string) error {
	claim = strings.TrimSpace(claim)
	if claim == "" {
		return fmt.Errorf("benchmark %s claim is required", name)
	}
	lower := strings.ToLower(claim + "\n" + evidence)
	actorBenchmarkContext := strings.Contains(lower, "actor") ||
		strings.Contains(lower, "mailbox") ||
		strings.Contains(lower, "zero_copy_move")
	if actorBenchmarkContext {
		for _, phrase := range []string{
			"faster than rust/c++",
			"faster than c++/rust",
			"faster than rust",
			"faster than c++",
			"fastest",
			"faster than",
			"benchmark superiority",
			"superiority",
			"outperforms",
			"beats rust",
			"beats c++",
			"beats go",
			"beats erlang",
			"official benchmark result",
			"official benchmark",
			"official upstream benchmark",
			"c++/rust parity",
			"rust/c++ parity",
			"rust parity",
			"c++ parity",
			"parity with rust",
			"parity with c++",
			"measured speed comparison",
			"throughput advantage",
			"latency advantage",
			"production throughput guarantee",
			"real-world sla",
		} {
			if containsUnsafeBenchmarkPhrase(lower, phrase) {
				return fmt.Errorf(
					"benchmark %s actor benchmark claim uses forbidden wording %q",
					name,
					phrase,
				)
			}
		}
	}
	if strings.Contains(lower, "zero_copy_move") {
		if err := validateZeroCopyPromotionText(
			fmt.Sprintf("benchmark %s zero_copy_move claim", name),
			lower,
		); err != nil {
			return err
		}
	}
	schedulerPrototypeContext := strings.Contains(lower, "scheduler") &&
		strings.Contains(lower, "prototype")
	if schedulerPrototypeContext {
		for _, phrase := range []string{
			"production runtime",
			"production scheduler",
			"production actor runtime",
			"full production",
		} {
			if containsUnsafeBenchmarkPhrase(lower, phrase) {
				return fmt.Errorf(
					"benchmark %s scheduler prototype claim uses forbidden wording %q",
					name,
					phrase,
				)
			}
		}
	}
	return nil
}

func validateZeroCopyPromotionText(context string, text string) error {
	rawLower := strings.ToLower(text)
	normalized := normalizeClaimText(rawLower)
	zeroCopyContext := strings.Contains(rawLower, "zero_copy_move") ||
		strings.Contains(normalized, "zero copy") ||
		strings.Contains(normalized, "copy free")
	if !zeroCopyContext {
		return nil
	}
	for _, phrase := range []string{
		"production runtime",
		"distributed zero copy",
		"network zero copy",
		"cross machine zero copy",
		"cross node zero copy",
		"inter node zero copy",
		"remote node zero copy",
		"remote nodes zero copy",
		"distributed copy free",
		"cross node copy free",
		"across nodes",
		"across node",
		"remote nodes",
		"remote node",
	} {
		if containsUnsafeBenchmarkPhrase(normalized, phrase) {
			return fmt.Errorf("%s uses forbidden wording %q", context, phrase)
		}
	}
	return nil
}

func validateActorMemoryDomains(domains []ActorMemoryDomainReport) []string {
	if len(domains) == 0 {
		return []string{"actor_memory_domains evidence is required"}
	}
	var issues []string
	seen := map[string]bool{}
	hasByteBackpressure := false
	hasOwnedRegion := false
	for _, domain := range domains {
		context := "actor_memory_domains"
		if strings.TrimSpace(domain.ActorID) != "" {
			context = fmt.Sprintf("actor_memory_domains[%s]", domain.ActorID)
		}
		if domain.SchemaVersion != "tetra.actors.memory-domain.v1" {
			issues = append(
				issues,
				fmt.Sprintf(
					"%s schema_version is %q, want tetra.actors.memory-domain.v1",
					context,
					domain.SchemaVersion,
				),
			)
		}
		if strings.TrimSpace(domain.ActorID) == "" {
			issues = append(issues, fmt.Sprintf("%s actor_id is required", context))
		}
		if seen[domain.ActorID] {
			issues = append(
				issues,
				fmt.Sprintf("duplicate actor memory domain actor_id %s", domain.ActorID),
			)
		}
		seen[domain.ActorID] = true
		if strings.TrimSpace(domain.EvidenceClass) == "" {
			issues = append(issues, fmt.Sprintf("%s evidence_class is required", context))
		}
		if domain.EvidenceClass == "allocation_report_estimate" {
			issues = append(
				issues,
				fmt.Sprintf("%s evidence_class must not be allocation_report_estimate", context),
			)
		}
		if strings.TrimSpace(domain.EvidenceMethod) == "" {
			issues = append(issues, fmt.Sprintf("%s evidence_method is required", context))
		}
		if !domain.RuntimeMeasured && strings.TrimSpace(domain.RuntimeBlockedReason) == "" {
			issues = append(
				issues,
				fmt.Sprintf(
					"%s runtime_blocked_reason is required when runtime_measured=false",
					context,
				),
			)
		}
		if domain.ProductionRuntimeClaimed {
			issues = append(
				issues,
				fmt.Sprintf("%s production actor runtime claim is forbidden", context),
			)
		}
		if domain.DistributedZeroCopyClaimed {
			issues = append(
				issues,
				fmt.Sprintf("%s distributed actor zero-copy claim is forbidden", context),
			)
		}
		issues = append(issues, validateActorMemoryDomainBytes(context, domain)...)
		if domain.Backpressure.Status == "byte_limit_reached" {
			hasByteBackpressure = true
		}
		if len(domain.OwnedRegions) > 0 {
			hasOwnedRegion = true
		}
		if !containsText(domain.NonClaims, "production actor runtime is not claimed") {
			issues = append(
				issues,
				fmt.Sprintf("%s missing production actor runtime nonclaim", context),
			)
		}
		if !containsText(domain.NonClaims, "distributed actor zero-copy is not claimed") {
			issues = append(
				issues,
				fmt.Sprintf("%s missing distributed actor zero-copy nonclaim", context),
			)
		}
	}
	if !hasByteBackpressure {
		issues = append(
			issues,
			"actor_memory_domains missing byte_limit_reached backpressure evidence",
		)
	}
	if !hasOwnedRegion {
		issues = append(issues, "actor_memory_domains missing owned region byte evidence")
	}
	return issues
}

func validateActorMemoryDomainBytes(context string, report ActorMemoryDomainReport) []string {
	var issues []string
	domain := report.Domain
	if strings.TrimSpace(domain.DomainID) == "" {
		issues = append(issues, fmt.Sprintf("%s domain_id is required", context))
	}
	if domain.Kind != "actor" {
		issues = append(
			issues,
			fmt.Sprintf("%s domain kind is %q, want actor", context, domain.Kind),
		)
	}
	if domain.OwnerKind != "actor" {
		issues = append(
			issues,
			fmt.Sprintf("%s owner_kind is %q, want actor", context, domain.OwnerKind),
		)
	}
	if domain.OwnerID != report.ActorID {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s owner_id is %q, want actor_id %q",
				context,
				domain.OwnerID,
				report.ActorID,
			),
		)
	}
	if strings.TrimSpace(domain.Lifetime) == "" {
		issues = append(issues, fmt.Sprintf("%s lifetime is required", context))
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
			issues = append(issues, fmt.Sprintf("%s %s must not be negative", context, name))
		}
	}
	if domain.CopyCount < 0 {
		issues = append(issues, fmt.Sprintf("%s copy_count must not be negative", context))
	}
	if domain.BytesCopied > 0 && domain.CopyCount == 0 {
		issues = append(issues, fmt.Sprintf("%s bytes_copied requires copy_count", context))
	}
	if domain.PeakBytes < domain.CurrentBytes {
		issues = append(issues, fmt.Sprintf("%s peak_bytes must be >= current_bytes", context))
	}
	if report.Mailbox.CapacityMessages <= 0 {
		issues = append(
			issues,
			fmt.Sprintf("%s mailbox capacity_messages must be positive", context),
		)
	}
	if report.Mailbox.CapacityBytes <= 0 {
		issues = append(issues, fmt.Sprintf("%s mailbox capacity_bytes must be positive", context))
	}
	if report.Mailbox.QueuedMessages < 0 || report.Mailbox.QueuedBytes < 0 ||
		report.Mailbox.PeakQueuedBytes < 0 ||
		report.Mailbox.ReclaimedBytes < 0 {
		issues = append(issues, fmt.Sprintf("%s mailbox counts must not be negative", context))
	}
	if report.Mailbox.QueuedMessages > report.Mailbox.CapacityMessages {
		issues = append(issues, fmt.Sprintf("%s queued messages exceed capacity", context))
	}
	if report.Mailbox.QueuedBytes > report.Mailbox.CapacityBytes {
		issues = append(issues, fmt.Sprintf("%s queued bytes exceed capacity", context))
	}
	if report.Mailbox.PeakQueuedBytes < report.Mailbox.QueuedBytes {
		issues = append(
			issues,
			fmt.Sprintf("%s peak queued bytes must be >= queued bytes", context),
		)
	}
	if report.MessagePool.LiveBytes != report.Mailbox.QueuedBytes {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s message_pool live_bytes = %d, want queued_bytes %d",
				context,
				report.MessagePool.LiveBytes,
				report.Mailbox.QueuedBytes,
			),
		)
	}
	if report.MessagePool.ReclaimedBytes != report.Mailbox.ReclaimedBytes {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s message_pool reclaimed_bytes = %d, want mailbox reclaimed_bytes %d",
				context,
				report.MessagePool.ReclaimedBytes,
				report.Mailbox.ReclaimedBytes,
			),
		)
	}
	if report.MessagePool.CapacityBytes != report.Mailbox.CapacityBytes {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s message_pool capacity_bytes = %d, want mailbox capacity_bytes %d",
				context,
				report.MessagePool.CapacityBytes,
				report.Mailbox.CapacityBytes,
			),
		)
	}
	ownedBytes := 0
	for _, owned := range report.OwnedRegions {
		if strings.TrimSpace(owned.RegionName) == "" {
			issues = append(issues, fmt.Sprintf("%s owned region name is required", context))
		}
		if owned.DomainID != domain.DomainID {
			issues = append(
				issues,
				fmt.Sprintf(
					"%s owned region %s domain_id = %q, want %q",
					context,
					owned.RegionName,
					owned.DomainID,
					domain.DomainID,
				),
			)
		}
		if owned.OwnerID != report.ActorID {
			issues = append(
				issues,
				fmt.Sprintf(
					"%s owned region %s owner_id = %q, want %q",
					context,
					owned.RegionName,
					owned.OwnerID,
					report.ActorID,
				),
			)
		}
		if owned.Bytes <= 0 {
			issues = append(
				issues,
				fmt.Sprintf("%s owned region %s bytes must be positive", context, owned.RegionName),
			)
		}
		ownedBytes += positiveInt(owned.Bytes)
	}
	if ownedBytes > report.Mailbox.QueuedBytes {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s owned region bytes = %d, want <= queued bytes %d",
				context,
				ownedBytes,
				report.Mailbox.QueuedBytes,
			),
		)
	}
	if domain.CurrentBytes != int64(report.Mailbox.QueuedBytes) {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s current_bytes = %d, want queued_bytes %d",
				context,
				domain.CurrentBytes,
				report.Mailbox.QueuedBytes,
			),
		)
	}
	if domain.CommittedBytes != int64(report.Mailbox.CapacityBytes) {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s committed_bytes = %d, want mailbox capacity bytes %d",
				context,
				domain.CommittedBytes,
				report.Mailbox.CapacityBytes,
			),
		)
	}
	if domain.ReleasedBytes != int64(report.Mailbox.ReclaimedBytes) {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s released_bytes = %d, want reclaimed bytes %d",
				context,
				domain.ReleasedBytes,
				report.Mailbox.ReclaimedBytes,
			),
		)
	}
	switch report.Backpressure.Status {
	case "available", "message_limit_reached", "byte_limit_reached":
	default:
		issues = append(
			issues,
			fmt.Sprintf("%s backpressure status is %q", context, report.Backpressure.Status),
		)
	}
	if report.Backpressure.Status == "byte_limit_reached" &&
		!strings.Contains(strings.ToLower(report.Backpressure.Reason), "byte") {
		issues = append(
			issues,
			fmt.Sprintf("%s byte_limit_reached backpressure must include byte reason", context),
		)
	}
	if strings.TrimSpace(report.Backpressure.Mode) == "" {
		issues = append(issues, fmt.Sprintf("%s backpressure mode is required", context))
	}
	return issues
}

func containsText(values []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), want) {
			return true
		}
	}
	return false
}

func positiveInt(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func normalizeClaimText(text string) string {
	replacer := strings.NewReplacer("-", " ", "_", " ", "/", " ")
	return strings.Join(strings.Fields(replacer.Replace(strings.ToLower(text))), " ")
}

func containsUnsafeBenchmarkPhrase(lower string, phrase string) bool {
	start := 0
	for {
		idx := strings.Index(lower[start:], phrase)
		if idx < 0 {
			return false
		}
		idx += start
		if !benchmarkPhraseContextIsSafeNonClaim(lower, idx, phrase) {
			return true
		}
		start = idx + len(phrase)
	}
}

func benchmarkPhraseContextIsSafeNonClaim(lower string, idx int, phrase string) bool {
	prefixStart := idx - 56
	if prefixStart < 0 {
		prefixStart = 0
	}
	prefix := strings.TrimSpace(lower[prefixStart:idx])
	for _, safePrefix := range []string{"no", "not", "without"} {
		if strings.HasSuffix(prefix, safePrefix) {
			return true
		}
	}
	for _, safeBefore := range []string{
		"does not claim",
		"not claimed",
		"not proven",
		"not implied",
		"without claiming",
	} {
		if strings.Contains(prefix, safeBefore) {
			return true
		}
	}
	suffixEnd := idx + len(phrase) + 96
	if suffixEnd > len(lower) {
		suffixEnd = len(lower)
	}
	suffix := lower[idx:suffixEnd]
	for _, safeAfter := range []string{
		"not claimed",
		"not proven",
		"not implied",
		"is not claimed",
		"claim is made",
	} {
		if strings.Contains(suffix, safeAfter) {
			return true
		}
	}
	return false
}

func validateContracts(contracts []ContractReport) []string {
	required := map[string]bool{
		"production task scheduler":                       false,
		"join cancel deadline select group lifecycle":     false,
		"actor mailbox backpressure and failure handling": false,
		"task actor thread boundary transfer rules":       false,
		"race safety model":                               false,
		"safe unsafe forbidden parallelism boundary":      false,
	}
	var issues []string
	for _, c := range contracts {
		name := strings.TrimSpace(c.Name)
		if name == "" {
			issues = append(issues, "contract name is required")
			continue
		}
		if _, ok := required[name]; ok {
			required[name] = true
		}
		if c.Status != "pass" {
			issues = append(
				issues,
				fmt.Sprintf("contract %s status is %q, want pass", name, c.Status),
			)
		}
		if strings.TrimSpace(c.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("contract %s evidence is required", name))
		} else if err := validateZeroCopyPromotionText(
			fmt.Sprintf("contract %s", name),
			c.Evidence,
		); err != nil {
			issues = append(issues, err.Error())
		}
	}
	for name, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("missing required parallel contract %q", name))
		}
	}
	return issues
}

func validateCases(cases []CaseReport) []string {
	required := map[string]bool{
		"scheduler fairness":                       false,
		"task join lifecycle":                      false,
		"task cancellation":                        false,
		"deadline timeout":                         false,
		"select readiness":                         false,
		"task group lifecycle":                     false,
		"task group cancel wakes deadline join":    false,
		"actor recv cancel wake":                   false,
		"nested cancellation propagation":          false,
		"task actor mailbox handoff":               false,
		"actor mailbox backpressure":               false,
		"message pool exhaustion":                  false,
		"invalid actor handle send":                false,
		"done actor send":                          false,
		"actor failure handling":                   false,
		"invalid handle diagnostics":               false,
		"resource double join diagnostic":          false,
		"task group use-after-close diagnostic":    false,
		"ownership transfer across task boundary":  false,
		"ownership transfer across actor boundary": false,
		"race-safety shared mutable rejection":     false,
		"race-safety rejection matrix":             false,
		"actor island boundary proof":              false,
		"actor broker leak cleanup":                false,
		"safe unsafe forbidden boundary coverage":  false,
		"actor fanout mailbox drain soak":          false,
		"many tasks stress":                        false,
		"many actor messages stress":               false,
		"cancellation storm":                       false,
		"timeouts stress":                          false,
	}
	var issues []string
	seenPositive := false
	seenNegative := false
	seenStress := false
	for _, c := range cases {
		name := strings.TrimSpace(c.Name)
		if name == "" {
			issues = append(issues, "case name is required")
			continue
		}
		if _, ok := required[name]; ok {
			required[name] = true
		}
		switch c.Kind {
		case "positive":
			seenPositive = true
		case "negative":
			seenNegative = true
			if strings.TrimSpace(c.ExpectedError) == "" {
				issues = append(
					issues,
					fmt.Sprintf("negative case %s expected_error is required", name),
				)
			}
		case "stress":
			seenStress = true
			issues = append(issues, validateStressCaseMetadata(c)...)
		default:
			issues = append(
				issues,
				fmt.Sprintf("case %s kind is %q, want positive, negative, or stress", name, c.Kind),
			)
		}
		if !c.Ran {
			issues = append(issues, fmt.Sprintf("case %s did not run", name))
		}
		if !c.Pass {
			issues = append(issues, fmt.Sprintf("case %s did not pass", name))
		}
		if strings.TrimSpace(c.Error) != "" {
			issues = append(issues, fmt.Sprintf("case %s has unexpected error: %s", name, c.Error))
		}
	}
	if !seenPositive {
		issues = append(issues, "case evidence missing positive parallel case")
	}
	if !seenNegative {
		issues = append(issues, "case evidence missing negative parallel safety case")
	}
	if !seenStress {
		issues = append(issues, "case evidence missing parallel stress case")
	}
	for name, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("missing required parallel case %q", name))
		}
	}
	return issues
}

func validateStressCaseMetadata(c CaseReport) []string {
	name := strings.TrimSpace(c.Name)
	var issues []string
	if c.Iterations <= 0 {
		issues = append(
			issues,
			fmt.Sprintf(
				"stress case %s iterations = %d, want positive bounded iteration count",
				name,
				c.Iterations,
			),
		)
	}
	if strings.TrimSpace(c.DeterministicSeed) == "" {
		issues = append(issues, fmt.Sprintf("stress case %s deterministic_seed is required", name))
	}
	if c.MaxDurationMS <= 0 {
		issues = append(
			issues,
			fmt.Sprintf(
				"stress case %s max_duration_ms = %d, want positive bounded duration cap",
				name,
				c.MaxDurationMS,
			),
		)
	} else if c.MaxDurationMS > 600000 {
		issues = append(
			issues,
			fmt.Sprintf("stress case %s max_duration_ms = %d, want <= 600000", name, c.MaxDurationMS),
		)
	}
	return issues
}

func validateDiagnostics(cases []CaseReport, diagnostics []DiagnosticReport) []string {
	var issues []string
	byCase := map[string]DiagnosticReport{}
	for _, d := range diagnostics {
		name := strings.TrimSpace(d.Case)
		if name == "" {
			issues = append(issues, "diagnostic case is required")
			continue
		}
		if _, ok := byCase[name]; ok {
			issues = append(issues, fmt.Sprintf("duplicate diagnostic for case %s", name))
		}
		byCase[name] = d
	}
	for _, c := range cases {
		if c.Kind != "negative" {
			continue
		}
		name := strings.TrimSpace(c.Name)
		d, ok := byCase[name]
		if !ok {
			issues = append(
				issues,
				fmt.Sprintf(
					"negative case %s diagnostic with code, severity, category, and position is required",
					name,
				),
			)
			continue
		}
		code := strings.TrimSpace(d.Code)
		if code == "" {
			issues = append(
				issues,
				fmt.Sprintf("negative case %s diagnostic code is required", name),
			)
		}
		lowerCode := strings.ToLower(code)
		if strings.Contains(lowerCode, "generic") || strings.Contains(lowerCode, "placeholder") ||
			strings.Contains(lowerCode, "backend") {
			issues = append(
				issues,
				fmt.Sprintf("negative case %s diagnostic code %q is not stable", name, d.Code),
			)
		}
		if strings.TrimSpace(d.Severity) != "error" {
			issues = append(
				issues,
				fmt.Sprintf(
					"negative case %s diagnostic severity is %q, want error",
					name,
					d.Severity,
				),
			)
		}
		if strings.TrimSpace(d.Category) == "" {
			issues = append(
				issues,
				fmt.Sprintf("negative case %s diagnostic category is required", name),
			)
		}
		if strings.TrimSpace(d.Position) == "" {
			issues = append(
				issues,
				fmt.Sprintf("negative case %s diagnostic position is required", name),
			)
		}
		if strings.TrimSpace(d.ExpectedError) == "" {
			issues = append(
				issues,
				fmt.Sprintf("negative case %s diagnostic expected_error is required", name),
			)
		} else if d.ExpectedError != c.ExpectedError {
			issues = append(
				issues,
				fmt.Sprintf("negative case %s diagnostic expected_error = %q, want %q", name, d.ExpectedError, c.ExpectedError),
			)
		}
	}
	for name := range byCase {
		found := false
		for _, c := range cases {
			if c.Name == name && c.Kind == "negative" {
				found = true
				break
			}
		}
		if !found {
			issues = append(issues, fmt.Sprintf("diagnostic for unknown negative case %s", name))
		}
	}
	return issues
}

func validateAudit(audit []AuditReport) []string {
	required := map[string]bool{
		"production task scheduler":                                                    false,
		"join/cancel/deadline/select/group lifecycle":                                  false,
		"actor mailbox backpressure and failure handling":                              false,
		"task/actor/thread-boundary transfer rules":                                    false,
		"race-safety model or conservative rejections":                                 false,
		"stress evidence for tasks, actor messages, cancellation storms, and timeouts": false,
		"safe/unsafe/forbidden parallelism documentation":                              false,
		"stable parallel diagnostics":                                                  false,
		"actor benchmark Tier 0/Tier 1 preparation":                                    false,
		"release-gate entrypoint":                                                      false,
	}
	var issues []string
	if len(audit) == 0 {
		issues = append(issues, "completion audit is required")
	}
	seen := map[string]bool{}
	for _, row := range audit {
		requirement := strings.TrimSpace(row.Requirement)
		if requirement == "" {
			issues = append(issues, "completion audit row requirement is required")
			continue
		}
		if seen[requirement] {
			issues = append(
				issues,
				fmt.Sprintf("duplicate completion audit requirement %q", requirement),
			)
		}
		seen[requirement] = true
		if _, ok := required[requirement]; ok {
			required[requirement] = true
		}
		if strings.TrimSpace(row.Artifact) == "" {
			issues = append(
				issues,
				fmt.Sprintf("completion audit requirement %q artifact is required", requirement),
			)
		}
		if strings.TrimSpace(row.Evidence) == "" {
			issues = append(
				issues,
				fmt.Sprintf("completion audit requirement %q evidence is required", requirement),
			)
		} else if err := validateZeroCopyPromotionText(
			fmt.Sprintf("completion audit requirement %s", requirement),
			row.Evidence+"\n"+row.Result,
		); err != nil {
			issues = append(issues, err.Error())
		}
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(row.Result)), "pass") {
			issues = append(
				issues,
				fmt.Sprintf(
					"completion audit requirement %q result is %q, want pass",
					requirement,
					row.Result,
				),
			)
		}
		if requirement == "actor benchmark Tier 0/Tier 1 preparation" {
			issues = append(issues, validateActorBenchmarkAuditNonClaims(row)...)
		}
	}
	for requirement, ok := range required {
		if !ok {
			issues = append(
				issues,
				fmt.Sprintf("completion audit missing required requirement %q", requirement),
			)
		}
	}
	return issues
}

func validateActorBenchmarkAuditNonClaims(row AuditReport) []string {
	text := strings.ToLower(row.Evidence + "\n" + row.Result)
	var issues []string
	for _, phrase := range []string{
		"tier 0",
		"tier 1",
		"no benchmark superiority",
		"no c++/rust parity",
		"no official benchmark",
	} {
		if !strings.Contains(text, phrase) {
			issues = append(issues, fmt.Sprintf("actor benchmark audit missing %q", phrase))
		}
	}
	return issues
}

func decodeStrict(raw []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("trailing JSON content")
	}
	return nil
}
