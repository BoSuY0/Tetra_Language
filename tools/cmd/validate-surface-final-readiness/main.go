package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	surfaceFinalReadinessSchema     = "tetra.surface.final-readiness.v1"
	surfaceFinalReadinessScope      = "surface-v1-linux-web"
	surfaceFinalReadinessProducer   = "tools/cmd/validate-surface-final-readiness"
	surfaceFinalReadinessHashSchema = "tetra.release-artifact-hashes.v1alpha1"
)

type surfaceFinalReadinessOptions struct {
	ReportDir              string
	ProductReportDir       string
	ExpectedScope          string
	RequireClean           bool
	RequireCI              bool
	RequirePackage         bool
	Write                  bool
	ActionsPermissionsPath string
	CIRunsPath             string
	CurrentGitHead         string
	GitDirty               *bool
}

type surfaceFinalReadinessReport struct {
	Schema                  string   `json:"schema"`
	ReleaseScope            string   `json:"release_scope"`
	Status                  string   `json:"status"`
	Producer                string   `json:"producer"`
	GitHead                 string   `json:"git_head"`
	GitDirty                *bool    `json:"git_dirty"`
	ProductReportDir        string   `json:"product_report_dir"`
	ProductSummary          string   `json:"product_summary"`
	ProductSummaryGitHead   string   `json:"product_summary_git_head"`
	ProductGateStatus       string   `json:"product_gate_status"`
	ProductFinalVerdict     string   `json:"product_final_verdict"`
	ProductFinalSignoff     *bool    `json:"product_final_signoff"`
	CleanSameCommitRequired *bool    `json:"clean_same_commit_required"`
	CleanSameCommitProven   *bool    `json:"clean_same_commit_proven"`
	CIRequiredGate          *bool    `json:"ci_required_gate"`
	CIProofRequired         *bool    `json:"ci_proof_required"`
	CIProofStatus           string   `json:"ci_proof_status"`
	CIRunHead               string   `json:"ci_run_head,omitempty"`
	ActionsEnabled          *bool    `json:"actions_enabled"`
	PackageProofRequired    *bool    `json:"package_proof_required"`
	PackageProofStatus      string   `json:"package_proof_status"`
	PackageReport           string   `json:"package_report"`
	ArtifactHashManifest    string   `json:"artifact_hash_manifest"`
	ProductionClaim         *bool    `json:"production_claim"`
	FinalSignoff            *bool    `json:"final_signoff"`
	Nonclaims               []string `json:"nonclaims"`
	Blockers                []string `json:"blockers"`
}

type surfaceFinalProductSummary struct {
	Schema            string            `json:"schema"`
	ReleaseScope      string            `json:"release_scope"`
	Status            string            `json:"status"`
	GitHead           string            `json:"git_head"`
	GitDirty          *bool             `json:"git_dirty"`
	FinalVerdict      string            `json:"final_verdict"`
	ProductionClaim   *bool             `json:"production_claim"`
	FinalSignoff      *bool             `json:"final_signoff"`
	RequiredArtifacts map[string]string `json:"required_artifacts"`
}

type surfaceFinalHashManifest struct {
	Schema    string                     `json:"schema"`
	Artifacts []surfaceFinalHashArtifact `json:"artifacts"`
}

type surfaceFinalHashArtifact struct {
	Path string `json:"path"`
}

func main() {
	var opt surfaceFinalReadinessOptions
	var gitDirty string
	flag.StringVar(&opt.ReportDir, "report-dir", "", "Surface final readiness report directory")
	flag.StringVar(
		&opt.ProductReportDir,
		"product-report-dir",
		"",
		"Surface product report directory used as prerequisite evidence",
	)
	flag.StringVar(
		&opt.ExpectedScope,
		"expected-scope",
		surfaceFinalReadinessScope,
		"expected Surface release scope",
	)
	flag.BoolVar(&opt.RequireClean, "require-clean", false, "require clean same-commit evidence")
	flag.BoolVar(&opt.RequireCI, "require-ci", false, "require exact-head CI proof")
	flag.BoolVar(&opt.RequirePackage, "require-package", false, "require package proof")
	flag.BoolVar(&opt.Write, "write", false, "write final-readiness.json before validation")
	flag.StringVar(
		&opt.ActionsPermissionsPath,
		"actions-permissions",
		"",
		"optional GitHub Actions permissions JSON",
	)
	flag.StringVar(
		&opt.CIRunsPath,
		"ci-runs",
		"",
		"optional GitHub Actions runs JSON for the current head",
	)
	flag.StringVar(
		&opt.CurrentGitHead,
		"current-git-head",
		"",
		"override current git head when writing",
	)
	flag.StringVar(&gitDirty, "git-dirty", "", "override current git dirty state when writing")
	flag.Parse()

	if gitDirty != "" {
		parsed, err := strconv.ParseBool(gitDirty)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --git-dirty value %q\n", gitDirty)
			os.Exit(2)
		}
		opt.GitDirty = &parsed
	}
	if err := validateSurfaceFinalReadiness(opt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateSurfaceFinalReadiness(opt surfaceFinalReadinessOptions) error {
	if opt.Write {
		if err := writeSurfaceFinalReadiness(opt); err != nil {
			return err
		}
		return nil
	}

	reportDir := strings.TrimSpace(opt.ReportDir)
	if reportDir == "" {
		return errors.New("report-dir is required")
	}
	expectedScope := strings.TrimSpace(opt.ExpectedScope)
	if expectedScope == "" {
		expectedScope = surfaceFinalReadinessScope
	}
	raw, err := os.ReadFile(filepath.Join(reportDir, "final-readiness.json"))
	if err != nil {
		return fmt.Errorf("final-readiness.json read failed: %w", err)
	}
	var report surfaceFinalReadinessReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return fmt.Errorf("final-readiness.json decode failed: %w", err)
	}

	var issues []string
	issues = append(issues, validateSurfaceFinalReadinessFields(report, expectedScope)...)
	issues = append(issues, validateSurfaceFinalReadinessNonclaims(report.Nonclaims)...)
	issues = append(
		issues,
		validateSurfaceFinalReadinessHashManifest(
			filepath.Join(reportDir, "artifact-hashes.json"),
		)...)
	if opt.RequireClean {
		if report.GitDirty == nil || *report.GitDirty {
			issues = append(issues, "final-readiness.json --require-clean requires git_dirty=false")
		}
		if report.CleanSameCommitProven == nil || !*report.CleanSameCommitProven {
			issues = append(
				issues,
				"final-readiness.json --require-clean requires clean_same_commit_proven=true",
			)
		}
	}
	if opt.RequireCI && report.CIProofStatus != "validated" {
		issues = append(
			issues,
			fmt.Sprintf(
				"final-readiness.json ci_proof_status is %q, want validated because --require-ci was set",
				report.CIProofStatus,
			),
		)
	}
	if opt.RequirePackage && report.PackageProofStatus != "validated" {
		issues = append(
			issues,
			fmt.Sprintf(
				("final-readiness.json package_proof_status is %q, want validated "+
					"because --require-package was set"),
				report.PackageProofStatus,
			),
		)
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func writeSurfaceFinalReadiness(opt surfaceFinalReadinessOptions) error {
	reportDir := strings.TrimSpace(opt.ReportDir)
	if reportDir == "" {
		return errors.New("report-dir is required")
	}
	productDir := strings.TrimSpace(opt.ProductReportDir)
	if productDir == "" {
		return errors.New("product-report-dir is required when --write is set")
	}
	expectedScope := strings.TrimSpace(opt.ExpectedScope)
	if expectedScope == "" {
		expectedScope = surfaceFinalReadinessScope
	}
	product, err := readSurfaceFinalProductSummary(
		filepath.Join(productDir, "product-summary.json"),
	)
	if err != nil {
		return err
	}
	head := strings.TrimSpace(opt.CurrentGitHead)
	if head == "" {
		head, err = currentGitHead()
		if err != nil {
			return err
		}
	}
	gitDirty := false
	if opt.GitDirty != nil {
		gitDirty = *opt.GitDirty
	} else {
		gitDirty, err = currentGitDirty()
		if err != nil {
			return err
		}
	}

	cleanSameCommit := !gitDirty && product.GitDirty != nil && !*product.GitDirty &&
		head == product.GitHead
	packageStatus := packageProofStatus(productDir, product)
	ciStatus, actionsEnabled := ciProofStatus(opt.ActionsPermissionsPath, opt.CIRunsPath)
	status := "blocked_final_requirements"
	finalSignoff := false
	productionClaim := false
	var blockers []string
	if !cleanSameCommit {
		blockers = append(blockers, "clean-same-commit-not-proven")
	}
	if packageStatus != "validated" {
		blockers = append(blockers, "package-proof-not-validated")
	}
	if ciStatus != "validated" {
		blockers = append(blockers, "remote-ci-proof-not-validated")
	}
	if len(blockers) == 0 {
		status = "ready"
		finalSignoff = true
		productionClaim = true
	}

	report := surfaceFinalReadinessReport{
		Schema:                  surfaceFinalReadinessSchema,
		ReleaseScope:            expectedScope,
		Status:                  status,
		Producer:                surfaceFinalReadinessProducer,
		GitHead:                 head,
		GitDirty:                boolPtr(gitDirty),
		ProductReportDir:        productDir,
		ProductSummary:          "product-summary.json",
		ProductSummaryGitHead:   product.GitHead,
		ProductGateStatus:       product.Status,
		ProductFinalVerdict:     product.FinalVerdict,
		ProductFinalSignoff:     product.FinalSignoff,
		CleanSameCommitRequired: boolPtr(true),
		CleanSameCommitProven:   boolPtr(cleanSameCommit),
		CIRequiredGate:          boolPtr(true),
		CIProofRequired:         boolPtr(true),
		CIProofStatus:           ciStatus,
		ActionsEnabled:          actionsEnabled,
		PackageProofRequired:    boolPtr(true),
		PackageProofStatus:      packageStatus,
		PackageReport:           "surface-package.json",
		ArtifactHashManifest:    "artifact-hashes.json",
		ProductionClaim:         boolPtr(productionClaim),
		FinalSignoff:            boolPtr(finalSignoff),
		Nonclaims:               surfaceFinalReadinessNonclaims(),
		Blockers:                blockers,
	}
	if ciStatus == "validated" {
		report.CIRunHead = head
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("final-readiness.json marshal failed: %w", err)
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return fmt.Errorf("final report dir create failed: %w", err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "final-readiness.json"), raw, 0o644); err != nil {
		return fmt.Errorf("final-readiness.json write failed: %w", err)
	}
	return nil
}

func validateSurfaceFinalReadinessFields(
	report surfaceFinalReadinessReport,
	expectedScope string,
) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: surfaceFinalReadinessSchema},
		{field: "release_scope", got: report.ReleaseScope, want: expectedScope},
		{field: "producer", got: report.Producer, want: surfaceFinalReadinessProducer},
		{field: "product_summary", got: report.ProductSummary, want: "product-summary.json"},
		{field: "artifact_hash_manifest", got: report.ArtifactHashManifest, want: "artifact-hashes.json"},
	} {
		if check.got != check.want {
			issues = append(
				issues,
				fmt.Sprintf(
					"final-readiness.json %s is %q, want %q",
					check.field,
					check.got,
					check.want,
				),
			)
		}
	}
	for _, check := range []struct {
		field string
		got   string
	}{
		{field: "git_head", got: report.GitHead},
		{field: "product_report_dir", got: report.ProductReportDir},
		{field: "product_summary_git_head", got: report.ProductSummaryGitHead},
		{field: "product_gate_status", got: report.ProductGateStatus},
		{field: "product_final_verdict", got: report.ProductFinalVerdict},
		{field: "ci_proof_status", got: report.CIProofStatus},
		{field: "package_proof_status", got: report.PackageProofStatus},
		{field: "package_report", got: report.PackageReport},
	} {
		if strings.TrimSpace(check.got) == "" {
			issues = append(
				issues,
				fmt.Sprintf("final-readiness.json %s must not be empty", check.field),
			)
		}
	}
	if report.Status != "blocked_final_requirements" && report.Status != "ready" {
		issues = append(
			issues,
			fmt.Sprintf(
				"final-readiness.json status is %q, want blocked_final_requirements or ready",
				report.Status,
			),
		)
	}
	if report.ProductGateStatus != "product_gate_passed_p29_final_audit_required" &&
		report.ProductGateStatus != "product_gate_passed_clean_same_commit_blocked" {
		issues = append(
			issues,
			fmt.Sprintf(
				"final-readiness.json product_gate_status is %q, want product-gate final audit status",
				report.ProductGateStatus,
			),
		)
	}
	if report.ProductFinalVerdict != "P29_FINAL_AUDIT_REQUIRED" &&
		report.ProductFinalVerdict != "BLOCKED_DIRTY_CHECKOUT" {
		issues = append(
			issues,
			fmt.Sprintf(
				"final-readiness.json product_final_verdict is %q, want P29 final audit verdict",
				report.ProductFinalVerdict,
			),
		)
	}
	if report.GitHead != "" && report.ProductSummaryGitHead != "" &&
		report.GitHead != report.ProductSummaryGitHead {
		issues = append(issues, "final-readiness.json git_head must match product_summary_git_head")
	}
	issues = append(issues, requireSurfaceFinalBool("git_dirty", report.GitDirty)...)
	issues = append(
		issues,
		requireSurfaceFinalBoolValue("product_final_signoff", report.ProductFinalSignoff, false)...)
	issues = append(
		issues,
		requireSurfaceFinalBoolValue(
			"clean_same_commit_required",
			report.CleanSameCommitRequired,
			true,
		)...)
	issues = append(
		issues,
		requireSurfaceFinalBool("clean_same_commit_proven", report.CleanSameCommitProven)...)
	issues = append(
		issues,
		requireSurfaceFinalBoolValue("ci_required_gate", report.CIRequiredGate, true)...)
	issues = append(
		issues,
		requireSurfaceFinalBoolValue("ci_proof_required", report.CIProofRequired, true)...)
	issues = append(
		issues,
		requireSurfaceFinalBoolValue(
			"package_proof_required",
			report.PackageProofRequired,
			true,
		)...)
	issues = append(issues, requireSurfaceFinalBool("production_claim", report.ProductionClaim)...)
	issues = append(issues, requireSurfaceFinalBool("final_signoff", report.FinalSignoff)...)
	if report.CIProofStatus != "validated" && report.CIProofStatus != "missing" &&
		report.CIProofStatus != "blocked_actions_disabled" {
		issues = append(
			issues,
			fmt.Sprintf(
				("final-readiness.json ci_proof_status is %q, want validated, "+
					"missing, or blocked_actions_disabled"),
				report.CIProofStatus,
			),
		)
	}
	if report.PackageProofStatus != "validated" && report.PackageProofStatus != "missing" {
		issues = append(
			issues,
			fmt.Sprintf(
				"final-readiness.json package_proof_status is %q, want validated or missing",
				report.PackageProofStatus,
			),
		)
	}
	if report.FinalSignoff != nil && !*report.FinalSignoff {
		if report.ProductionClaim != nil && *report.ProductionClaim {
			issues = append(
				issues,
				"final-readiness.json production_claim must be false without final_signoff",
			)
		}
		if report.Status == "ready" {
			issues = append(issues, "final-readiness.json ready status requires final_signoff=true")
		}
	}
	if report.FinalSignoff != nil && *report.FinalSignoff {
		if report.Status != "ready" {
			issues = append(issues, "final-readiness.json final_signoff=true requires ready status")
		}
		if report.ProductionClaim == nil || !*report.ProductionClaim {
			issues = append(
				issues,
				"final-readiness.json final_signoff=true requires production_claim=true",
			)
		}
		if report.CleanSameCommitProven == nil || !*report.CleanSameCommitProven {
			issues = append(
				issues,
				"final-readiness.json final_signoff=true requires clean_same_commit_proven=true",
			)
		}
		if report.CIProofStatus != "validated" {
			issues = append(
				issues,
				"final-readiness.json final_signoff=true requires ci_proof_status=validated",
			)
		}
		if report.PackageProofStatus != "validated" {
			issues = append(
				issues,
				"final-readiness.json final_signoff=true requires package_proof_status=validated",
			)
		}
		if len(report.Blockers) > 0 {
			issues = append(
				issues,
				"final-readiness.json final_signoff=true requires empty blockers",
			)
		}
	}
	if report.Status == "blocked_final_requirements" && len(report.Blockers) == 0 {
		issues = append(issues, "final-readiness.json blocked_final_requirements requires blockers")
	}
	return issues
}

func validateSurfaceFinalReadinessNonclaims(nonclaims []string) []string {
	set := map[string]bool{}
	for _, nonclaim := range nonclaims {
		set[nonclaim] = true
	}
	var issues []string
	for _, required := range surfaceFinalReadinessNonclaims() {
		if !set[required] {
			issues = append(
				issues,
				fmt.Sprintf("final-readiness.json nonclaims missing %q", required),
			)
		}
	}
	return issues
}

func validateSurfaceFinalReadinessHashManifest(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("artifact-hashes.json read failed: %v", err)}
	}
	var manifest surfaceFinalHashManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return []string{fmt.Sprintf("artifact-hashes.json decode failed: %v", err)}
	}
	var issues []string
	if manifest.Schema != surfaceFinalReadinessHashSchema {
		issues = append(
			issues,
			fmt.Sprintf(
				"artifact-hashes.json schema is %q, want %q",
				manifest.Schema,
				surfaceFinalReadinessHashSchema,
			),
		)
	}
	covered := false
	for _, artifact := range manifest.Artifacts {
		if filepath.ToSlash(artifact.Path) == "final-readiness.json" {
			covered = true
			break
		}
	}
	if !covered {
		issues = append(issues, "artifact-hashes.json missing final-readiness.json")
	}
	return issues
}

func readSurfaceFinalProductSummary(path string) (surfaceFinalProductSummary, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return surfaceFinalProductSummary{}, fmt.Errorf("product-summary.json read failed: %w", err)
	}
	var summary surfaceFinalProductSummary
	if err := json.Unmarshal(raw, &summary); err != nil {
		return surfaceFinalProductSummary{}, fmt.Errorf(
			"product-summary.json decode failed: %w",
			err,
		)
	}
	if summary.ReleaseScope != surfaceFinalReadinessScope {
		return surfaceFinalProductSummary{}, fmt.Errorf(
			"product-summary.json release_scope is %q, want %q",
			summary.ReleaseScope,
			surfaceFinalReadinessScope,
		)
	}
	if strings.TrimSpace(summary.GitHead) == "" {
		return surfaceFinalProductSummary{}, errors.New(
			"product-summary.json git_head must not be empty",
		)
	}
	return summary, nil
}

func packageProofStatus(productDir string, product surfaceFinalProductSummary) string {
	if product.RequiredArtifacts["package"] == "" {
		return "missing"
	}
	for _, rel := range []string{product.RequiredArtifacts["package"], "surface-package.json"} {
		if _, err := os.ReadFile(filepath.Join(productDir, filepath.FromSlash(rel))); err != nil {
			return "missing"
		}
	}
	return "validated"
}

func ciProofStatus(actionsPermissionsPath string, ciRunsPath string) (string, *bool) {
	actionsEnabled := boolPtr(true)
	if strings.TrimSpace(actionsPermissionsPath) != "" {
		raw, err := os.ReadFile(actionsPermissionsPath)
		if err == nil {
			var permissions struct {
				Enabled *bool `json:"enabled"`
			}
			if json.Unmarshal(raw, &permissions) == nil && permissions.Enabled != nil {
				actionsEnabled = permissions.Enabled
			}
		}
	}
	if strings.TrimSpace(ciRunsPath) != "" {
		raw, err := os.ReadFile(ciRunsPath)
		if err == nil {
			var runs []json.RawMessage
			if json.Unmarshal(raw, &runs) == nil && len(runs) > 0 {
				return "validated", actionsEnabled
			}
		}
	}
	if actionsEnabled != nil && !*actionsEnabled {
		return "blocked_actions_disabled", actionsEnabled
	}
	return "missing", actionsEnabled
}

func currentGitHead() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func currentGitDirty() (bool, error) {
	if err := exec.Command("git", "diff", "--quiet").Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return true, nil
		}
		return false, fmt.Errorf("git diff --quiet failed: %w", err)
	}
	if err := exec.Command("git", "diff", "--cached", "--quiet").Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return true, nil
		}
		return false, fmt.Errorf("git diff --cached --quiet failed: %w", err)
	}
	out, err := exec.Command("git", "ls-files", "--others", "--exclude-standard").Output()
	if err != nil {
		return false, fmt.Errorf("git ls-files --others failed: %w", err)
	}
	return strings.TrimSpace(string(out)) != "", nil
}

func requireSurfaceFinalBool(field string, got *bool) []string {
	if got == nil {
		return []string{fmt.Sprintf("final-readiness.json %s is missing", field)}
	}
	return nil
}

func requireSurfaceFinalBoolValue(field string, got *bool, want bool) []string {
	if got == nil {
		return []string{fmt.Sprintf("final-readiness.json %s is missing", field)}
	}
	if *got != want {
		return []string{fmt.Sprintf("final-readiness.json %s is %t, want %t", field, *got, want)}
	}
	return nil
}

func boolPtr(value bool) *bool {
	return &value
}

func surfaceFinalReadinessNonclaims() []string {
	return []string{
		"all-platform-surface-parity",
		"nonclaim-macos-surface-production-support",
		"nonclaim-windows-surface-production-support",
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
		"prod-stable-scoped-without-clean-same-commit-ci-package-proof",
	}
}
