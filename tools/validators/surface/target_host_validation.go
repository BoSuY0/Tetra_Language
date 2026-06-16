package surface

import (
	"errors"
	"fmt"
	"strings"
)

type TargetHostStatusReport struct {
	Schema             string                           `json:"schema"`
	Target             string                           `json:"target"`
	Status             string                           `json:"status"`
	Tier               string                           `json:"tier"`
	ReleaseScope       string                           `json:"release_scope"`
	Source             string                           `json:"source"`
	HostOS             string                           `json:"host_os"`
	HostArch           string                           `json:"host_arch"`
	Reason             string                           `json:"reason"`
	ProductionClaim    bool                             `json:"production_claim"`
	Experimental       bool                             `json:"experimental"`
	TargetHostEvidence bool                             `json:"target_host_evidence"`
	BuildOnlyEvidence  bool                             `json:"build_only_evidence"`
	BuildOnlyPromotion bool                             `json:"build_only_promotion"`
	LinuxSubstitute    bool                             `json:"linux_substitute"`
	CIArtifactRequired bool                             `json:"ci_artifact_required"`
	RequiredEvidence   TargetHostRequiredEvidenceReport `json:"required_evidence"`
	UnsupportedClaims  []string                         `json:"unsupported_claims"`
	NegativeGuards     TargetHostNegativeGuardsReport   `json:"negative_guards"`
}

type TargetHostRequiredEvidenceReport struct {
	RealWindow            bool `json:"real_window"`
	NativeInput           bool `json:"native_input"`
	Clipboard             bool `json:"clipboard"`
	DPIScaling            bool `json:"dpi_scaling"`
	AccessibilitySnapshot bool `json:"accessibility_snapshot"`
	AppShell              bool `json:"app_shell"`
}

type TargetHostNegativeGuardsReport struct {
	NoLinuxSubstitute    bool `json:"no_linux_substitute"`
	NoBuildOnlyPromotion bool `json:"no_build_only_promotion"`
	NoProductionClaim    bool `json:"no_production_claim"`
	NoDocsOnlyEvidence   bool `json:"no_docs_only_evidence"`
	NoCopiedReport       bool `json:"no_copied_report"`
	CIArtifactRequired   bool `json:"ci_artifact_required"`
}

func ValidateTargetHostStatus(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != TargetHostStatusSchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, TargetHostStatusSchemaV1)
	}

	var report TargetHostStatusReport
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: TargetHostStatusSchemaV1},
		{field: "release_scope", got: report.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("target_host_status %s is %q, want %q", check.field, check.got, check.want))
		}
	}
	if !isTargetHostStatusTarget(report.Target) {
		issues = append(issues, fmt.Sprintf("target_host_status target is %q, want windows-x64 or macos-x64", report.Target))
	}
	if strings.TrimSpace(report.Source) == "" {
		issues = append(issues, "target_host_status source is required")
	}
	if strings.TrimSpace(report.HostOS) == "" {
		issues = append(issues, "target_host_status host_os is required")
	}
	if strings.TrimSpace(report.HostArch) == "" {
		issues = append(issues, "target_host_status host_arch is required")
	}
	if strings.TrimSpace(report.Reason) == "" {
		issues = append(issues, "target_host_status reason is required")
	}
	if report.ProductionClaim {
		issues = append(issues, "target_host_status production_claim must be false without full production target-host gate evidence")
	}
	if report.BuildOnlyEvidence {
		issues = append(issues, "target_host_status build-only evidence must be false; build-only reports are not Surface runtime evidence")
	}
	if report.BuildOnlyPromotion {
		issues = append(issues, "target_host_status build-only promotion must be false")
	}
	if report.LinuxSubstitute {
		issues = append(issues, "target_host_status linux substitute must be false for non-Linux target-host evidence")
	}
	if !report.CIArtifactRequired {
		issues = append(issues, "target_host_status ci_artifact_required must be true")
	}
	issues = append(issues, validateTargetHostNegativeGuards(report.NegativeGuards)...)

	switch report.Status {
	case "unsupported":
		issues = append(issues, validateUnsupportedTargetHostStatus(report)...)
	case "beta_target_host":
		issues = append(issues, validateBetaTargetHostStatus(report)...)
	default:
		issues = append(issues, fmt.Sprintf("target_host_status status is %q, want unsupported or beta_target_host", report.Status))
	}

	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func isTargetHostStatusTarget(target string) bool {
	switch target {
	case "windows-x64", "macos-x64":
		return true
	default:
		return false
	}
}

func validateUnsupportedTargetHostStatus(report TargetHostStatusReport) []string {
	var issues []string
	if report.Tier != "UNSUPPORTED" {
		issues = append(issues, fmt.Sprintf("target_host_status tier is %q, want UNSUPPORTED", report.Tier))
	}
	if report.Experimental {
		issues = append(issues, "target_host_status experimental must be false for unsupported nonclaim status")
	}
	if report.TargetHostEvidence {
		issues = append(issues, "target_host_status target-host evidence must be false for unsupported nonclaim status")
	}
	if targetHostRequiredEvidenceAny(report.RequiredEvidence) {
		issues = append(issues, "target_host_status unsupported required_evidence entries must be false until real target-host evidence exists")
	}
	issues = append(issues, validateUnsupportedTargetHostClaims(report.Target, report.UnsupportedClaims)...)
	return issues
}

func validateBetaTargetHostStatus(report TargetHostStatusReport) []string {
	var issues []string
	if report.Tier != "BETA_TARGET_HOST" {
		issues = append(issues, fmt.Sprintf("target_host_status tier is %q, want BETA_TARGET_HOST", report.Tier))
	}
	if !report.Experimental {
		issues = append(issues, "target_host_status experimental must be true for beta target-host status")
	}
	if !report.TargetHostEvidence {
		issues = append(issues, "target_host_status target-host evidence must be true for beta target-host status")
	}
	required := report.RequiredEvidence
	for _, check := range []struct {
		field string
		ok    bool
	}{
		{field: "real_window", ok: required.RealWindow},
		{field: "native_input", ok: required.NativeInput},
		{field: "clipboard", ok: required.Clipboard},
		{field: "dpi_scaling", ok: required.DPIScaling},
		{field: "accessibility_snapshot", ok: required.AccessibilitySnapshot},
	} {
		if !check.ok {
			issues = append(issues, fmt.Sprintf("target_host_status beta target-host evidence requires required_evidence.%s", check.field))
		}
	}
	return issues
}

func validateTargetHostNegativeGuards(guards TargetHostNegativeGuardsReport) []string {
	var issues []string
	for _, check := range []struct {
		field string
		ok    bool
	}{
		{field: "no_linux_substitute", ok: guards.NoLinuxSubstitute},
		{field: "no_build_only_promotion", ok: guards.NoBuildOnlyPromotion},
		{field: "no_production_claim", ok: guards.NoProductionClaim},
		{field: "no_docs_only_evidence", ok: guards.NoDocsOnlyEvidence},
		{field: "no_copied_report", ok: guards.NoCopiedReport},
		{field: "ci_artifact_required", ok: guards.CIArtifactRequired},
	} {
		if !check.ok {
			issues = append(issues, fmt.Sprintf("target_host_status negative_guards.%s must be true", check.field))
		}
	}
	return issues
}

func targetHostRequiredEvidenceAny(evidence TargetHostRequiredEvidenceReport) bool {
	return evidence.RealWindow ||
		evidence.NativeInput ||
		evidence.Clipboard ||
		evidence.DPIScaling ||
		evidence.AccessibilitySnapshot ||
		evidence.AppShell
}

func validateUnsupportedTargetHostClaims(target string, claims []string) []string {
	var issues []string
	required := unsupportedTargetHostClaimSet(target)
	for _, want := range required {
		if !stringSliceContainsFold(claims, want) {
			issues = append(issues, fmt.Sprintf("target_host_status unsupported_claims requires %s", want))
		}
	}
	return issues
}

func unsupportedTargetHostClaimSet(target string) []string {
	switch target {
	case "windows-x64":
		return []string{
			"windows-real-window-surface",
			"windows-production-surface-nonclaim",
			"windows-target-host-runtime",
			"build-only-windows-surface-runtime",
			"linux-substitute-windows-surface-runtime",
		}
	case "macos-x64":
		return []string{
			"macos-real-window-surface",
			"macos-production-surface-nonclaim",
			"macos-target-host-runtime",
			"build-only-macos-surface-runtime",
			"linux-substitute-macos-surface-runtime",
		}
	default:
		return nil
	}
}

func isGitHead(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, ch := range value {
		if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F') {
			continue
		}
		return false
	}
	return true
}
