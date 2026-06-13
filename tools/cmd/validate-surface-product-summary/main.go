package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	surfaceProductSummarySchema         = "tetra.surface.product-summary.v1"
	surfaceProductCategorySummarySchema = "tetra.surface.product-category-summary.v1"
	surfaceProductReleaseScope          = "surface-v1-linux-web"
	surfaceProductFinalVerdictOwner     = "SURFACE-BEAUTY-P29"
	surfaceProductHashManifestSchema    = "tetra.release-artifact-hashes.v1alpha1"
)

type surfaceProductSummaryOptions struct {
	ReportDir    string
	SummaryPath  string
	ManifestPath string
}

type surfaceProductSummaryReport struct {
	Schema                        string                 `json:"schema"`
	ReleaseScope                  string                 `json:"release_scope"`
	Status                        string                 `json:"status"`
	Producer                      string                 `json:"producer"`
	GitDirty                      *bool                  `json:"git_dirty"`
	ProductGateSummary            string                 `json:"product_gate_summary"`
	ReleaseGateReport             string                 `json:"release_gate_report"`
	ArtifactHashManifest          string                 `json:"artifact_hash_manifest"`
	ReleaseState                  string                 `json:"release_state"`
	ArtifactHashes                string                 `json:"artifact_hashes"`
	ClaimScanner                  string                 `json:"claim_scanner"`
	Manifest                      string                 `json:"manifest"`
	Docs                          string                 `json:"docs"`
	CIRequiredGate                *bool                  `json:"ci_required_gate"`
	ContinueOnErrorBypassAllowed  *bool                  `json:"continue_on_error_bypass_allowed"`
	FinalVerdictOwner             string                 `json:"final_verdict_owner"`
	FinalVerdict                  string                 `json:"final_verdict"`
	ProductionClaim               *bool                  `json:"production_claim"`
	FinalSignoff                  *bool                  `json:"final_signoff"`
	CanonicalFinalReadinessReport *bool                  `json:"canonical_final_readiness_report"`
	FinalReadinessSource          string                 `json:"final_readiness_source"`
	InnerReleaseSummaryRole       string                 `json:"inner_release_summary_role"`
	ReleaseGateReportFinalSignoff *bool                  `json:"release_gate_report_final_signoff"`
	CleanSameCommitRequired       *bool                  `json:"clean_same_commit_required"`
	CleanSameCommitProven         *bool                  `json:"clean_same_commit_proven"`
	TargetMatrix                  []surfaceProductTarget `json:"target_matrix"`
	RequiredArtifacts             map[string]string      `json:"required_artifacts"`
	Nonclaims                     []string               `json:"nonclaims"`
}

type surfaceProductTarget struct {
	Target          string `json:"target"`
	Status          string `json:"status"`
	Tier            string `json:"tier"`
	ProductionClaim *bool  `json:"production_claim"`
	Report          string `json:"report"`
}

type surfaceProductCategorySummary struct {
	Schema            string `json:"schema"`
	ReleaseScope      string `json:"release_scope"`
	Category          string `json:"category"`
	Status            string `json:"status"`
	SourceReport      string `json:"source_report"`
	Evidence          string `json:"evidence"`
	FinalVerdictOwner string `json:"final_verdict_owner"`
	FinalVerdict      string `json:"final_verdict"`
	ProductionClaim   *bool  `json:"production_claim"`
	FinalSignoff      *bool  `json:"final_signoff"`
}

type surfaceProductHashManifest struct {
	Schema    string                       `json:"schema"`
	Artifacts []surfaceProductHashArtifact `json:"artifacts"`
}

type surfaceProductHashArtifact struct {
	Path string `json:"path"`
}

func main() {
	var opt surfaceProductSummaryOptions
	flag.StringVar(&opt.ReportDir, "report-dir", "", "Surface product report directory")
	flag.StringVar(&opt.SummaryPath, "summary", "", "Surface product summary path; defaults to <report-dir>/product-summary.json")
	flag.StringVar(&opt.ManifestPath, "manifest", "", "Surface product artifact hash manifest path; defaults to <report-dir>/artifact-hashes.json")
	flag.Parse()

	if err := validateSurfaceProductSummary(opt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateSurfaceProductSummary(opt surfaceProductSummaryOptions) error {
	reportDir := strings.TrimSpace(opt.ReportDir)
	if reportDir == "" {
		return errors.New("report-dir is required")
	}
	summaryPath := strings.TrimSpace(opt.SummaryPath)
	if summaryPath == "" {
		summaryPath = filepath.Join(reportDir, "product-summary.json")
	}
	manifestPath := strings.TrimSpace(opt.ManifestPath)
	if manifestPath == "" {
		manifestPath = filepath.Join(reportDir, "artifact-hashes.json")
	}

	raw, err := os.ReadFile(summaryPath)
	if err != nil {
		return fmt.Errorf("product-summary.json read failed: %w", err)
	}
	var report surfaceProductSummaryReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return fmt.Errorf("product-summary.json decode failed: %w", err)
	}

	var issues []string
	coveredArtifacts, hashIssues := loadProductHashCoverage(manifestPath)
	issues = append(issues, validateProductSummaryFields(report)...)
	issues = append(issues, validateProductSummaryTargets(report.TargetMatrix)...)
	issues = append(issues, validateProductSummaryRequiredArtifacts(report.RequiredArtifacts)...)
	issues = append(issues, validateProductSummaryNonclaims(report.Nonclaims)...)
	issues = append(issues, hashIssues...)
	issues = append(issues, validateProductCategorySummaries(reportDir, report.RequiredArtifacts, report.FinalVerdict, coveredArtifacts)...)
	issues = append(issues, validateProductSummaryHashCoverage(report.RequiredArtifacts, coveredArtifacts)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateProductSummaryFields(report surfaceProductSummaryReport) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: surfaceProductSummarySchema},
		{field: "release_scope", got: report.ReleaseScope, want: surfaceProductReleaseScope},
		{field: "producer", got: report.Producer, want: "scripts/release/surface/product-gate.sh"},
		{field: "product_gate_summary", got: report.ProductGateSummary, want: "surface-product-gate-summary.json"},
		{field: "release_gate_report", got: report.ReleaseGateReport, want: "surface-release-summary.json"},
		{field: "artifact_hash_manifest", got: report.ArtifactHashManifest, want: "artifact-hashes.json"},
		{field: "release_state", got: report.ReleaseState, want: "validated"},
		{field: "artifact_hashes", got: report.ArtifactHashes, want: "validated"},
		{field: "claim_scanner", got: report.ClaimScanner, want: "validated"},
		{field: "manifest", got: report.Manifest, want: "validated"},
		{field: "docs", got: report.Docs, want: "validated"},
		{field: "final_verdict_owner", got: report.FinalVerdictOwner, want: surfaceProductFinalVerdictOwner},
		{field: "final_readiness_source", got: report.FinalReadinessSource, want: "product-summary.json"},
		{field: "inner_release_summary_role", got: report.InnerReleaseSummaryRole, want: "prerequisite_evidence_not_final_signoff"},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("product-summary.json %s is %q, want %q", check.field, check.got, check.want))
		}
	}
	if report.Status != "product_gate_passed_clean_same_commit_blocked" && report.Status != "product_gate_passed_p29_final_audit_required" {
		issues = append(issues, fmt.Sprintf("product-summary.json status is %q, want P29 product-gate status", report.Status))
	}
	if report.FinalVerdict != "BLOCKED_DIRTY_CHECKOUT" && report.FinalVerdict != "P29_FINAL_AUDIT_REQUIRED" {
		issues = append(issues, fmt.Sprintf("product-summary.json final_verdict is %q, want P29 final-audit verdict", report.FinalVerdict))
	}
	if report.GitDirty == nil {
		issues = append(issues, "product-summary.json git_dirty is missing")
	}
	issues = append(issues, requireBool("ci_required_gate", report.CIRequiredGate, true)...)
	issues = append(issues, requireBool("continue_on_error_bypass_allowed", report.ContinueOnErrorBypassAllowed, false)...)
	issues = append(issues, requireBool("production_claim", report.ProductionClaim, false)...)
	issues = append(issues, requireBool("final_signoff", report.FinalSignoff, false)...)
	issues = append(issues, requireBool("canonical_final_readiness_report", report.CanonicalFinalReadinessReport, true)...)
	issues = append(issues, requireBool("release_gate_report_final_signoff", report.ReleaseGateReportFinalSignoff, false)...)
	issues = append(issues, requireBool("clean_same_commit_required", report.CleanSameCommitRequired, true)...)
	issues = append(issues, requireBool("clean_same_commit_proven", report.CleanSameCommitProven, false)...)
	if report.GitDirty != nil && *report.GitDirty {
		if report.Status != "product_gate_passed_clean_same_commit_blocked" {
			issues = append(issues, "product-summary.json dirty checkout must use product_gate_passed_clean_same_commit_blocked status")
		}
		if report.FinalVerdict != "BLOCKED_DIRTY_CHECKOUT" {
			issues = append(issues, "product-summary.json dirty checkout must use BLOCKED_DIRTY_CHECKOUT final_verdict")
		}
	} else if report.GitDirty != nil {
		if report.Status != "product_gate_passed_p29_final_audit_required" {
			issues = append(issues, "product-summary.json clean checkout must use product_gate_passed_p29_final_audit_required status before final signoff")
		}
		if report.FinalVerdict != "P29_FINAL_AUDIT_REQUIRED" {
			issues = append(issues, "product-summary.json clean checkout must use P29_FINAL_AUDIT_REQUIRED final_verdict before final signoff")
		}
	}
	return issues
}

func validateProductSummaryTargets(targets []surfaceProductTarget) []string {
	required := map[string]surfaceProductTarget{
		"headless":    {Status: "release-test-evidence", Tier: "evidence-target", Report: "surface-headless-release.json"},
		"linux-x64":   {Status: "current", Tier: "bounded-linux-web-scope", Report: "surface-linux-x64-release-app-shell.json"},
		"wasm32-web":  {Status: "current", Tier: "bounded-linux-web-scope", Report: "surface-wasm32-web-release-browser.json"},
		"macos-x64":   {Status: "unsupported", Tier: "UNSUPPORTED", Report: "surface-macos-x64-target-host-status.json"},
		"windows-x64": {Status: "unsupported", Tier: "UNSUPPORTED", Report: "surface-windows-x64-target-host-status.json"},
		"wasm32-wasi": {Status: "unsupported", Tier: "UNSUPPORTED", Report: "surface-release-summary.json"},
	}
	claimByTarget := map[string]bool{
		"headless":    false,
		"linux-x64":   true,
		"wasm32-web":  true,
		"macos-x64":   false,
		"windows-x64": false,
		"wasm32-wasi": false,
	}
	byTarget := map[string]surfaceProductTarget{}
	seenTargets := map[string]bool{}
	var issues []string
	for _, target := range targets {
		name := strings.TrimSpace(target.Target)
		if name == "" {
			issues = append(issues, "product-summary.json target_matrix contains empty target")
			continue
		}
		if seenTargets[name] {
			issues = append(issues, fmt.Sprintf("product-summary.json duplicate target_matrix target %q", name))
			continue
		}
		seenTargets[name] = true
		if _, ok := required[name]; !ok {
			issues = append(issues, fmt.Sprintf("product-summary.json unexpected target_matrix target %q", name))
			continue
		}
		byTarget[name] = target
	}
	if len(targets) != len(required) {
		issues = append(issues, fmt.Sprintf("product-summary.json target_matrix has %d entries, want %d", len(targets), len(required)))
	}
	for name, want := range required {
		got, ok := byTarget[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("product-summary.json target_matrix missing %q", name))
			continue
		}
		for _, check := range []struct {
			field string
			got   string
			want  string
		}{
			{field: "status", got: got.Status, want: want.Status},
			{field: "tier", got: got.Tier, want: want.Tier},
			{field: "report", got: got.Report, want: want.Report},
		} {
			if check.got != check.want {
				issues = append(issues, fmt.Sprintf("product-summary.json target_matrix[%s].%s is %q, want %q", name, check.field, check.got, check.want))
			}
		}
		issues = append(issues, requireBool(fmt.Sprintf("target_matrix[%s].production_claim", name), got.ProductionClaim, claimByTarget[name])...)
	}
	return issues
}

func validateProductSummaryRequiredArtifacts(required map[string]string) []string {
	want := productSummaryRequiredArtifacts()
	var issues []string
	if len(required) == 0 {
		return []string{"product-summary.json required_artifacts must not be empty"}
	}
	for key, value := range want {
		if required[key] != value {
			issues = append(issues, fmt.Sprintf("product-summary.json required_artifacts.%s is %q, want %q", key, required[key], value))
		}
	}
	return issues
}

func validateProductSummaryNonclaims(nonclaims []string) []string {
	set := map[string]bool{}
	for _, nonclaim := range nonclaims {
		set[nonclaim] = true
	}
	var issues []string
	for _, required := range productSummaryRequiredNonclaims() {
		if !set[required] {
			issues = append(issues, fmt.Sprintf("product-summary.json nonclaims missing %q", required))
		}
	}
	return issues
}

func validateProductCategorySummaries(reportDir string, required map[string]string, finalVerdict string, coveredArtifacts map[string]bool) []string {
	categoryByKey := map[string]string{
		"visual":           "visual",
		"accessibility":    "accessibility",
		"performance":      "performance",
		"app_shell":        "app-shell",
		"package":          "package",
		"reference_apps":   "reference-apps",
		"claim_governance": "claim-governance",
	}
	var issues []string
	for key, category := range categoryByKey {
		path := required[key]
		if path == "" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(reportDir, filepath.FromSlash(path)))
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s read failed: %v", path, err))
			continue
		}
		var summary surfaceProductCategorySummary
		if err := json.Unmarshal(raw, &summary); err != nil {
			issues = append(issues, fmt.Sprintf("%s decode failed: %v", path, err))
			continue
		}
		for _, check := range []struct {
			field string
			got   string
			want  string
		}{
			{field: "schema", got: summary.Schema, want: surfaceProductCategorySummarySchema},
			{field: "release_scope", got: summary.ReleaseScope, want: surfaceProductReleaseScope},
			{field: "category", got: summary.Category, want: category},
			{field: "final_verdict_owner", got: summary.FinalVerdictOwner, want: surfaceProductFinalVerdictOwner},
			{field: "final_verdict", got: summary.FinalVerdict, want: finalVerdict},
		} {
			if check.got != check.want {
				issues = append(issues, fmt.Sprintf("%s %s is %q, want %q", path, check.field, check.got, check.want))
			}
		}
		if strings.TrimSpace(summary.Status) == "" {
			issues = append(issues, fmt.Sprintf("%s status must not be empty", path))
		}
		sourceReport := filepath.ToSlash(strings.TrimSpace(summary.SourceReport))
		if sourceReport == "" {
			issues = append(issues, fmt.Sprintf("%s source_report must not be empty", path))
		} else if relPath, pathIssues := productReportRelPath(sourceReport); len(pathIssues) > 0 {
			for _, issue := range pathIssues {
				issues = append(issues, fmt.Sprintf("%s source_report %q %s", path, sourceReport, issue))
			}
		} else {
			if _, err := os.ReadFile(filepath.Join(reportDir, filepath.FromSlash(relPath))); err != nil {
				issues = append(issues, fmt.Sprintf("%s source_report %q read failed: %v", path, sourceReport, err))
			}
			if !coveredArtifacts[relPath] {
				issues = append(issues, fmt.Sprintf("%s source_report %q is not hash-covered", path, sourceReport))
			}
		}
		if strings.TrimSpace(summary.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("%s evidence must not be empty", path))
		}
		issues = append(issues, requireBool(path+" production_claim", summary.ProductionClaim, false)...)
		issues = append(issues, requireBool(path+" final_signoff", summary.FinalSignoff, false)...)
	}
	return issues
}

func loadProductHashCoverage(manifestPath string) (map[string]bool, []string) {
	covered := map[string]bool{}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return covered, []string{fmt.Sprintf("artifact-hashes.json read failed: %v", err)}
	}
	var manifest surfaceProductHashManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return covered, []string{fmt.Sprintf("artifact-hashes.json decode failed: %v", err)}
	}
	var issues []string
	if manifest.Schema != surfaceProductHashManifestSchema {
		issues = append(issues, fmt.Sprintf("artifact-hashes.json schema is %q, want %q", manifest.Schema, surfaceProductHashManifestSchema))
	}
	for _, artifact := range manifest.Artifacts {
		covered[filepath.ToSlash(artifact.Path)] = true
	}
	return covered, issues
}

func validateProductSummaryHashCoverage(required map[string]string, covered map[string]bool) []string {
	var issues []string
	for key, path := range required {
		if key == "artifact_hashes" {
			continue
		}
		if !covered[filepath.ToSlash(path)] {
			issues = append(issues, fmt.Sprintf("artifact-hashes.json missing required product artifact %q", path))
		}
	}
	return issues
}

func productReportRelPath(path string) (string, []string) {
	clean := filepath.Clean(filepath.FromSlash(path))
	if clean == "." || strings.TrimSpace(path) == "" {
		return "", []string{"must not be empty"}
	}
	if filepath.IsAbs(clean) {
		return "", []string{"must be relative to the report directory"}
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", []string{"must not escape the report directory"}
	}
	return filepath.ToSlash(clean), nil
}

func requireBool(field string, got *bool, want bool) []string {
	if got == nil {
		return []string{fmt.Sprintf("product-summary.json %s is missing", field)}
	}
	if *got != want {
		return []string{fmt.Sprintf("product-summary.json %s is %t, want %t", field, *got, want)}
	}
	return nil
}

func productSummaryRequiredArtifacts() map[string]string {
	return map[string]string{
		"product_summary":  "product-summary.json",
		"artifact_hashes":  "artifact-hashes.json",
		"visual":           "visual/visual-summary.json",
		"accessibility":    "accessibility/accessibility-summary.json",
		"performance":      "performance/performance-budget.json",
		"app_shell":        "app-shell/app-shell-summary.json",
		"package":          "package/package-summary.json",
		"reference_apps":   "reference-apps/reference-apps-summary.json",
		"claim_governance": "claim-governance/claims-summary.json",
	}
}

func productSummaryRequiredNonclaims() []string {
	return []string{
		"all-platform-surface-parity",
		"macos-surface-production-nonclaim",
		"windows-surface-production-nonclaim",
		"wasm32-wasi-surface-ui-runtime",
		"gpu-renderer",
		"full-rich-text-editor",
		"full-screen-reader-support",
		"official-benchmark-superiority",
		"electron-api-compatibility",
		"react-api-compatibility",
		"css-cascade-compatibility",
		"dom-authored-application-ui",
		"user-javascript-application-logic",
	}
}
