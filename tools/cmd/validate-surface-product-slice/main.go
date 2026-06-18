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
	productSliceSummarySchema = "tetra.surface.product-slice-summary.v1"
	productSliceReleaseScope  = "surface-v1-linux-web"
	productSliceProducer      = "scripts/release/surface/surface-product-slice-gate.sh"
	productSliceHashSchema    = "tetra.release-artifact-hashes.v1alpha1"
)

type productSliceOptions struct {
	ReportDir    string
	SummaryPath  string
	ManifestPath string
}

type productSliceSummary struct {
	Schema               string                  `json:"schema"`
	ReleaseScope         string                  `json:"release_scope"`
	Producer             string                  `json:"producer"`
	GitHead              string                  `json:"git_head"`
	GitCommit            string                  `json:"git_commit"`
	GitDirty             *bool                   `json:"git_dirty"`
	CommandLine          string                  `json:"command_line"`
	FlagshipSource       string                  `json:"flagship_source"`
	AppID                string                  `json:"app_id"`
	ArtifactHashManifest string                  `json:"artifact_hash_manifest"`
	ClaimScanner         string                  `json:"claim_scanner"`
	Manifest             string                  `json:"manifest"`
	Docs                 string                  `json:"docs"`
	MorphRenderedBeauty  string                  `json:"morph_rendered_beauty"`
	ProductClaim         *bool                   `json:"product_claim"`
	FinalSignoff         *bool                   `json:"final_signoff"`
	Categories           []productSliceCategory  `json:"categories"`
	RequiredArtifacts    map[string]string       `json:"required_artifacts"`
	Nonclaims            []string                `json:"nonclaims"`
	Validations          productSliceValidations `json:"validations"`
	Pass                 bool                    `json:"pass"`
}

type productSliceCategory struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	SourceReport string `json:"source_report"`
	Evidence     string `json:"evidence"`
	Pass         bool   `json:"pass"`
}

type productSliceValidations struct {
	FlagshipRuntime     string `json:"flagship_runtime"`
	DeveloperLoop       string `json:"developer_loop"`
	Package             string `json:"package"`
	MorphRenderedBeauty string `json:"morph_rendered_beauty"`
	Claims              string `json:"claims"`
	Manifest            string `json:"manifest"`
	Docs                string `json:"docs"`
	ArtifactHashes      string `json:"artifact_hashes"`
}

type productSliceMorphRenderedBeautyGate struct {
	Schema                     string                      `json:"schema"`
	Status                     string                      `json:"status"`
	Producer                   string                      `json:"producer"`
	GitHead                    string                      `json:"git_head"`
	GitCommit                  string                      `json:"git_commit"`
	GitDirty                   *bool                       `json:"git_dirty"`
	MorphRenderedBeauty        string                      `json:"morph_rendered_beauty_report"`
	MorphToPixels              string                      `json:"morph_to_pixels_report"`
	TargetMatrix               []productSliceMRBTargetGate `json:"target_matrix"`
	StablePromotionBlockers    []string                    `json:"stable_promotion_blockers"`
	RendererOwnedStableTargets []string                    `json:"renderer_owned_stable_targets"`
	BridgeOwnedStableTargets   []string                    `json:"bridge_owned_stable_targets"`
	ProductClaim               *bool                       `json:"product_claim"`
	FinalSignoff               *bool                       `json:"final_signoff"`
	Pass                       bool                        `json:"pass"`
}

type productSliceMRBTargetGate struct {
	Target                   string `json:"target"`
	Status                   string `json:"status"`
	RendererOwnedStableProof *bool  `json:"renderer_owned_stable_proof"`
	ProductClaim             *bool  `json:"product_claim"`
}

type productSliceHashManifest struct {
	Schema    string                     `json:"schema"`
	Artifacts []productSliceHashArtifact `json:"artifacts"`
}

type productSliceHashArtifact struct {
	Path string `json:"path"`
}

func main() {
	var opt productSliceOptions
	flag.StringVar(&opt.ReportDir, "report-dir", "", "Surface product-slice report directory")
	flag.StringVar(
		&opt.SummaryPath,
		"summary",
		"",
		"summary path; defaults to <report-dir>/surface-product-slice-summary.json",
	)
	flag.StringVar(
		&opt.ManifestPath,
		"manifest",
		"",
		"artifact hash manifest path; defaults to <report-dir>/artifact-hashes.json",
	)
	flag.Parse()
	if err := validateProductSlice(opt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateProductSlice(opt productSliceOptions) error {
	reportDir := strings.TrimSpace(opt.ReportDir)
	if reportDir == "" {
		return errors.New("report-dir is required")
	}
	summaryPath := strings.TrimSpace(opt.SummaryPath)
	if summaryPath == "" {
		summaryPath = filepath.Join(reportDir, "surface-product-slice-summary.json")
	}
	manifestPath := strings.TrimSpace(opt.ManifestPath)
	if manifestPath == "" {
		manifestPath = filepath.Join(reportDir, "artifact-hashes.json")
	}

	raw, err := os.ReadFile(summaryPath)
	if err != nil {
		return fmt.Errorf("surface-product-slice-summary.json read failed: %w", err)
	}
	var summary productSliceSummary
	if err := json.Unmarshal(raw, &summary); err != nil {
		return fmt.Errorf("surface-product-slice-summary.json decode failed: %w", err)
	}
	covered, hashIssues := productSliceHashCoverage(manifestPath)
	var issues []string
	issues = append(issues, validateProductSliceFields(summary)...)
	issues = append(
		issues,
		validateProductSliceRequiredArtifacts(reportDir, summary.RequiredArtifacts, covered)...)
	issues = append(
		issues,
		validateProductSliceCategories(reportDir, summary.Categories, covered)...)
	issues = append(
		issues,
		validateProductSliceMorphRenderedBeautyGate(
			reportDir,
			summary.RequiredArtifacts,
			summary.ProductClaim,
			summary.FinalSignoff,
		)...)
	issues = append(issues, validateProductSliceNonclaims(summary.Nonclaims)...)
	issues = append(issues, hashIssues...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateProductSliceFields(summary productSliceSummary) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: summary.Schema, want: productSliceSummarySchema},
		{field: "release_scope", got: summary.ReleaseScope, want: productSliceReleaseScope},
		{field: "producer", got: summary.Producer, want: productSliceProducer},
		{
			field: "flagship_source",
			got:   summary.FlagshipSource,
			want:  "examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra",
		},
		{field: "app_id", got: summary.AppID, want: "studio-shell"},
		{
			field: "artifact_hash_manifest",
			got:   summary.ArtifactHashManifest,
			want:  "artifact-hashes.json",
		},
		{field: "claim_scanner", got: summary.ClaimScanner, want: "validated"},
		{field: "manifest", got: summary.Manifest, want: "validated"},
		{field: "docs", got: summary.Docs, want: "validated"},
		{field: "morph_rendered_beauty", got: summary.MorphRenderedBeauty, want: "validated"},
		{
			field: "validations.flagship_runtime",
			got:   summary.Validations.FlagshipRuntime,
			want:  "validated",
		},
		{field: "validations.developer_loop", got: summary.Validations.DeveloperLoop, want: "validated"},
		{field: "validations.package", got: summary.Validations.Package, want: "validated"},
		{
			field: "validations.morph_rendered_beauty",
			got:   summary.Validations.MorphRenderedBeauty,
			want:  "validated",
		},
		{field: "validations.claims", got: summary.Validations.Claims, want: "validated"},
		{field: "validations.manifest", got: summary.Validations.Manifest, want: "validated"},
		{field: "validations.docs", got: summary.Validations.Docs, want: "validated"},
		{
			field: "validations.artifact_hashes",
			got:   summary.Validations.ArtifactHashes,
			want:  "validated",
		},
	} {
		if check.got != check.want {
			issues = append(
				issues,
				fmt.Sprintf(
					"surface-product-slice-summary.json %s is %q, want %q",
					check.field,
					check.got,
					check.want,
				),
			)
		}
	}
	if strings.TrimSpace(summary.CommandLine) == "" {
		issues = append(issues, "surface-product-slice-summary.json command_line is required")
	}
	if !productSliceValidGitCommit(summary.GitHead) {
		issues = append(
			issues,
			"surface-product-slice-summary.json git_head must be 40 hex characters",
		)
	}
	if !productSliceValidGitCommit(summary.GitCommit) {
		issues = append(
			issues,
			"surface-product-slice-summary.json git_commit must be 40 hex characters",
		)
	}
	if productSliceValidGitCommit(summary.GitHead) &&
		productSliceValidGitCommit(summary.GitCommit) &&
		summary.GitHead != summary.GitCommit {
		issues = append(issues, "surface-product-slice-summary.json git_commit must match git_head")
	}
	if summary.GitDirty == nil {
		issues = append(issues, "surface-product-slice-summary.json git_dirty is required")
	}
	issues = append(
		issues,
		validateProductSliceClaimState(
			"surface-product-slice-summary.json",
			summary.ProductClaim,
			summary.FinalSignoff,
			summary.GitDirty,
		)...)
	if !summary.Pass {
		issues = append(issues, "surface-product-slice-summary.json pass must be true")
	}
	return issues
}

func validateProductSliceRequiredArtifacts(
	reportDir string,
	required map[string]string,
	covered map[string]bool,
) []string {
	want := productSliceRequiredArtifacts()
	var issues []string
	for key, path := range want {
		got := filepath.ToSlash(strings.TrimSpace(required[key]))
		if got != path {
			issues = append(
				issues,
				fmt.Sprintf(
					"surface-product-slice-summary.json required_artifacts.%s is %q, want %q",
					key,
					got,
					path,
				),
			)
			continue
		}
		if key == "artifact_hashes" {
			continue
		}
		issues = append(issues, productSlicePathExists(reportDir, path)...)
		if !covered[path] {
			issues = append(
				issues,
				fmt.Sprintf(
					"artifact-hashes.json missing required product-slice artifact %q",
					path,
				),
			)
		}
	}
	return issues
}

func validateProductSliceCategories(
	reportDir string,
	categories []productSliceCategory,
	covered map[string]bool,
) []string {
	required := map[string]string{
		"flagship-runtime":      "flagship/flagship-runtime-summary.json",
		"developer-loop":        "dev-workflow/surface-dev-workflow.json",
		"package-update":        "package/surface-package.json",
		"morph-rendered-beauty": "morph-rendered-beauty/morph-rendered-beauty-gate-summary.json",
		"claim-governance":      "claims/claim-governance-summary.json",
		"docs-manifest":         "docs-manifest/docs-manifest-summary.json",
	}
	seen := map[string]productSliceCategory{}
	var issues []string
	for _, category := range categories {
		name := strings.TrimSpace(category.Name)
		if name == "" {
			issues = append(issues, "surface-product-slice-summary.json category name is required")
			continue
		}
		if _, ok := seen[name]; ok {
			issues = append(
				issues,
				fmt.Sprintf("surface-product-slice-summary.json duplicate category %q", name),
			)
			continue
		}
		seen[name] = category
	}
	if len(categories) != len(required) {
		issues = append(
			issues,
			fmt.Sprintf(
				"surface-product-slice-summary.json categories length = %d, want %d",
				len(categories),
				len(required),
			),
		)
	}
	for name, sourceReport := range required {
		category, ok := seen[name]
		if !ok {
			issues = append(
				issues,
				fmt.Sprintf("surface-product-slice-summary.json categories missing %q", name),
			)
			continue
		}
		if category.Status != "validated" {
			issues = append(
				issues,
				fmt.Sprintf(
					"surface-product-slice-summary.json category %s status is %q, want validated",
					name,
					category.Status,
				),
			)
		}
		if filepath.ToSlash(category.SourceReport) != sourceReport {
			issues = append(
				issues,
				fmt.Sprintf(
					"surface-product-slice-summary.json category %s source_report is %q, want %q",
					name,
					category.SourceReport,
					sourceReport,
				),
			)
		}
		if strings.TrimSpace(category.Evidence) == "" {
			issues = append(
				issues,
				fmt.Sprintf(
					"surface-product-slice-summary.json category %s evidence is required",
					name,
				),
			)
		}
		if !category.Pass {
			issues = append(
				issues,
				fmt.Sprintf(
					"surface-product-slice-summary.json category %s pass must be true",
					name,
				),
			)
		}
		issues = append(issues, productSlicePathExists(reportDir, sourceReport)...)
		if !covered[sourceReport] {
			issues = append(
				issues,
				fmt.Sprintf("artifact-hashes.json missing category source_report %q", sourceReport),
			)
		}
	}
	return issues
}

func validateProductSliceMorphRenderedBeautyGate(
	reportDir string,
	required map[string]string,
	productClaim *bool,
	finalSignoff *bool,
) []string {
	const requiredPath = "morph-rendered-beauty/morph-rendered-beauty-gate-summary.json"
	path := filepath.ToSlash(strings.TrimSpace(required["morph_rendered_beauty_gate"]))
	if path != requiredPath {
		return []string{
			fmt.Sprintf(
				("surface-product-slice-summary.json required_artifacts.morph_" +
					"rendered_beauty_gate is %q, want %q"),
				path,
				requiredPath,
			),
		}
	}
	raw, err := os.ReadFile(filepath.Join(reportDir, filepath.FromSlash(path)))
	if err != nil {
		return []string{fmt.Sprintf("morph rendered beauty gate summary read failed: %v", err)}
	}
	var summary productSliceMorphRenderedBeautyGate
	if err := json.Unmarshal(raw, &summary); err != nil {
		return []string{fmt.Sprintf("morph rendered beauty gate summary decode failed: %v", err)}
	}
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: summary.Schema, want: "tetra.surface.morph-rendered-beauty.gate.v1"},
		{
			field: "producer",
			got:   summary.Producer,
			want:  "scripts/release/surface/morph-rendered-beauty-gate.sh",
		},
		{field: "morph_rendered_beauty_report", got: filepath.ToSlash(
			summary.MorphRenderedBeauty,
		), want: "morph-rendered-beauty.json"},
		{field: "morph_to_pixels_report", got: filepath.ToSlash(
			summary.MorphToPixels,
		), want: "morph-to-pixels.json"},
	} {
		if check.got != check.want {
			issues = append(
				issues,
				fmt.Sprintf(
					"morph rendered beauty gate summary %s is %q, want %q",
					check.field,
					check.got,
					check.want,
				),
			)
		}
	}
	if summary.Status != "validated" && summary.Status != "validated_with_target_blockers" {
		issues = append(
			issues,
			fmt.Sprintf(
				("morph rendered beauty gate summary status is %q, want validated "+
					"or validated_with_target_blockers"),
				summary.Status,
			),
		)
	}
	if !productSliceValidGitCommit(summary.GitHead) {
		issues = append(
			issues,
			"morph rendered beauty gate summary git_head must be 40 hex characters",
		)
	}
	if !productSliceValidGitCommit(summary.GitCommit) {
		issues = append(
			issues,
			"morph rendered beauty gate summary git_commit must be 40 hex characters",
		)
	}
	if productSliceValidGitCommit(summary.GitHead) &&
		productSliceValidGitCommit(summary.GitCommit) &&
		summary.GitHead != summary.GitCommit {
		issues = append(issues, "morph rendered beauty gate summary git_commit must match git_head")
	}
	issues = append(
		issues,
		validateProductSliceClaimState(
			"morph_rendered_beauty",
			summary.ProductClaim,
			summary.FinalSignoff,
			summary.GitDirty,
		)...)
	if productClaim != nil && summary.ProductClaim != nil &&
		*productClaim != *summary.ProductClaim {
		issues = append(
			issues,
			("surface-product-slice-summary.json product_claim must match " +
				"morph_rendered_beauty.product_claim"),
		)
	}
	if finalSignoff != nil && summary.FinalSignoff != nil &&
		*finalSignoff != *summary.FinalSignoff {
		issues = append(
			issues,
			("surface-product-slice-summary.json final_signoff must match " +
				"morph_rendered_beauty.final_signoff"),
		)
	}
	if summary.ProductClaim != nil && *summary.ProductClaim {
		issues = append(issues, validateProductSlicePromotedMRBGate(summary)...)
	}
	if !summary.Pass {
		issues = append(issues, "morph rendered beauty gate summary pass must be true")
	}
	return issues
}

func validateProductSliceClaimState(
	field string,
	productClaim *bool,
	finalSignoff *bool,
	gitDirty *bool,
) []string {
	var issues []string
	productClaimField := productSliceClaimField(field, "product_claim")
	finalSignoffField := productSliceClaimField(field, "final_signoff")
	gitDirtyField := productSliceClaimField(field, "git_dirty")
	if productClaim == nil {
		issues = append(issues, productClaimField+" is missing")
		return issues
	}
	if finalSignoff == nil {
		issues = append(issues, finalSignoffField+" is missing")
		return issues
	}
	if *finalSignoff && !*productClaim {
		issues = append(issues, finalSignoffField+" requires product_claim")
	}
	if *productClaim && !*finalSignoff {
		issues = append(issues, productClaimField+" requires final_signoff")
	}
	if *productClaim {
		if gitDirty == nil {
			issues = append(issues, gitDirtyField+" is required for product_claim")
		} else if *gitDirty {
			issues = append(issues, productClaimField+" requires git_dirty=false")
		}
	}
	return issues
}

func productSliceClaimField(prefix string, field string) string {
	if strings.HasSuffix(prefix, ".json") {
		return prefix + " " + field
	}
	return prefix + "." + field
}

func validateProductSlicePromotedMRBGate(summary productSliceMorphRenderedBeautyGate) []string {
	var issues []string
	if summary.Status != "validated" {
		issues = append(
			issues,
			fmt.Sprintf(
				"morph rendered beauty gate summary status is %q, want validated for product_claim",
				summary.Status,
			),
		)
	}
	if len(summary.StablePromotionBlockers) > 0 {
		issues = append(
			issues,
			"morph rendered beauty gate summary stable_promotion_blockers must be empty for product_claim",
		)
	}
	if len(summary.BridgeOwnedStableTargets) > 0 {
		issues = append(
			issues,
			"morph rendered beauty gate summary bridge_owned_stable_targets must be empty for product_claim",
		)
	}
	requiredTargets := []string{"headless", "linux-x64-real-window", "wasm32-web-browser-canvas"}
	for _, target := range requiredTargets {
		if !productSliceContainsText(summary.RendererOwnedStableTargets, target) {
			issues = append(
				issues,
				fmt.Sprintf(
					"morph rendered beauty gate summary renderer_owned_stable_targets missing %s",
					target,
				),
			)
		}
	}
	seen := map[string]productSliceMRBTargetGate{}
	for _, target := range summary.TargetMatrix {
		if strings.TrimSpace(target.Target) == "" {
			continue
		}
		seen[target.Target] = target
	}
	for _, required := range requiredTargets {
		target, ok := seen[required]
		if !ok {
			issues = append(
				issues,
				fmt.Sprintf(
					"morph rendered beauty gate summary target_matrix missing %s",
					required,
				),
			)
			continue
		}
		if target.Status != "validated" {
			issues = append(
				issues,
				fmt.Sprintf(
					"morph rendered beauty gate summary target_matrix %s status is %q, want validated",
					required,
					target.Status,
				),
			)
		}
		if target.RendererOwnedStableProof == nil || !*target.RendererOwnedStableProof {
			issues = append(
				issues,
				fmt.Sprintf(
					"morph rendered beauty gate summary target_matrix %s renderer_owned_stable_proof must be true",
					required,
				),
			)
		}
		if target.ProductClaim == nil || !*target.ProductClaim {
			issues = append(
				issues,
				fmt.Sprintf(
					"morph rendered beauty gate summary target_matrix %s product_claim must be true",
					required,
				),
			)
		}
	}
	return issues
}

func validateProductSliceNonclaims(nonclaims []string) []string {
	set := map[string]bool{}
	for _, nonclaim := range nonclaims {
		set[strings.TrimSpace(nonclaim)] = true
	}
	var issues []string
	for _, required := range []string{
		"no-electron-api-compatibility",
		"no-react-runtime-claim",
		"no-css-runtime-claim",
		"no-dom-authored-application-ui",
		"nonclaim-macos-surface-production-support",
		"nonclaim-windows-surface-production-support",
		"no-gpu-renderer-parity",
		"no-native-widget-parity",
		"no-signing-or-notarization-claim",
		"no-automatic-network-update-claim",
	} {
		if !set[required] {
			issues = append(
				issues,
				fmt.Sprintf("surface-product-slice-summary.json nonclaims missing %q", required),
			)
		}
	}
	return issues
}

func productSliceRequiredArtifacts() map[string]string {
	return map[string]string{
		"product_slice_summary":      "surface-product-slice-summary.json",
		"artifact_hashes":            "artifact-hashes.json",
		"flagship_runtime_summary":   "flagship/flagship-runtime-summary.json",
		"flagship_headless":          "flagship/headless-block-system.json",
		"flagship_linux":             "flagship/linux-x64-real-window-block-system.json",
		"flagship_wasm":              "flagship/wasm32-web-browser-canvas-block-system.json",
		"developer_loop":             "dev-workflow/surface-dev-workflow.json",
		"package":                    "package/surface-package.json",
		"morph_rendered_beauty_gate": "morph-rendered-beauty/morph-rendered-beauty-gate-summary.json",
		"claim_governance":           "claims/claim-governance-summary.json",
		"docs_manifest":              "docs-manifest/docs-manifest-summary.json",
		"category_flagship_runtime":  "categories/flagship-runtime.json",
		"category_developer_loop":    "categories/developer-loop.json",
		"category_package_update":    "categories/package-update.json",
		"category_morph_beauty":      "categories/morph-rendered-beauty.json",
		"category_claim_governance":  "categories/claim-governance.json",
		"category_docs_manifest":     "categories/docs-manifest.json",
	}
}

func productSliceHashCoverage(manifestPath string) (map[string]bool, []string) {
	covered := map[string]bool{}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return covered, []string{fmt.Sprintf("artifact-hashes.json read failed: %v", err)}
	}
	var manifest productSliceHashManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return covered, []string{fmt.Sprintf("artifact-hashes.json decode failed: %v", err)}
	}
	var issues []string
	if manifest.Schema != productSliceHashSchema {
		issues = append(
			issues,
			fmt.Sprintf(
				"artifact-hashes.json schema is %q, want %q",
				manifest.Schema,
				productSliceHashSchema,
			),
		)
	}
	for _, artifact := range manifest.Artifacts {
		covered[filepath.ToSlash(artifact.Path)] = true
	}
	return covered, issues
}

func productSlicePathExists(reportDir string, rel string) []string {
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == "." || filepath.IsAbs(clean) || clean == ".." ||
		strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return []string{fmt.Sprintf("unsafe product-slice artifact path %q", rel)}
	}
	if _, err := os.Stat(filepath.Join(reportDir, clean)); err != nil {
		return []string{fmt.Sprintf("product-slice artifact %q read failed: %v", rel, err)}
	}
	return nil
}

func productSliceRequireBool(field string, got *bool, want bool) []string {
	if got == nil {
		return []string{fmt.Sprintf("surface-product-slice-summary.json %s is missing", field)}
	}
	if *got != want {
		return []string{
			fmt.Sprintf("surface-product-slice-summary.json %s is %t, want %t", field, *got, want),
		}
	}
	return nil
}

func productSliceContainsText(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func productSliceValidGitCommit(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, ch := range value {
		if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') {
			continue
		}
		return false
	}
	return true
}
