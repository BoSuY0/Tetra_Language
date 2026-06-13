package surface

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const TokenGraphContractSchemaV1 = "tetra.surface.token-graph.contract.v1"

type TokenGraphValidationOptions struct {
	Root string
}

type TokenGraphContract struct {
	Schema                  string                         `json:"schema"`
	Status                  string                         `json:"status"`
	SurfaceScope            string                         `json:"surface_scope"`
	SourceOfTruth           TokenGraphSourceOfTruth        `json:"source_of_truth"`
	RequiredCategories      []string                       `json:"required_categories"`
	RequiredTokens          []string                       `json:"required_tokens"`
	ReferenceSources        []string                       `json:"reference_sources"`
	AllowedRawLiteralScopes []TokenGraphRawLiteralScope    `json:"allowed_raw_literal_scopes"`
	ForbiddenRuntimeModels  []string                       `json:"forbidden_runtime_models"`
	OverrideOrder           []string                       `json:"override_order"`
	DensityDPI              []MorphDensityDPIReport        `json:"density_dpi"`
	DiagnosticsRequired     []string                       `json:"diagnostics_required"`
	NegativeGuards          TokenGraphNegativeGuardsReport `json:"negative_guards"`
	NonClaims               []string                       `json:"nonclaims"`
}

type TokenGraphSourceOfTruth struct {
	Module               string `json:"module"`
	Namespace            string `json:"namespace"`
	Source               string `json:"source"`
	SingleTokenGraph     bool   `json:"single_token_graph"`
	ExplicitImports      bool   `json:"explicit_imports"`
	NoGlobalCascade      bool   `json:"no_global_cascade"`
	MultipleColorSources bool   `json:"multiple_color_sources"`
}

type TokenGraphRawLiteralScope struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type TokenGraphNegativeGuardsReport struct {
	AliasCycleRejected           bool `json:"alias_cycle_rejected"`
	MissingTokenRejected         bool `json:"missing_token_rejected"`
	DuplicateSourceRejected      bool `json:"duplicate_source_rejected"`
	RawLiteralRejected           bool `json:"raw_literal_rejected"`
	UnresolvedFallbackRejected   bool `json:"unresolved_fallback_rejected"`
	CSSRuntimeRejected           bool `json:"css_runtime_rejected"`
	MultipleColorSourcesRejected bool `json:"multiple_color_sources_rejected"`
	OverrideOrderRejected        bool `json:"override_order_rejected"`
	DensityDPIRejected           bool `json:"density_dpi_rejected"`
}

func ValidateTokenGraphContract(contractRaw []byte, reportRaw []byte, options TokenGraphValidationOptions) error {
	var contract TokenGraphContract
	if err := decodeStrict(contractRaw, &contract); err != nil {
		return err
	}
	var report Report
	if err := decodeStrict(reportRaw, &report); err != nil {
		return err
	}
	issues := validateTokenGraphContractFields(contract)
	issues = append(issues, validateTokenGraphReport(contract, report)...)
	issues = append(issues, validateTokenGraphReferenceSources(contract, options.Root)...)
	if len(issues) > 0 {
		sort.Strings(issues)
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateTokenGraphContractFields(contract TokenGraphContract) []string {
	var issues []string
	if contract.Schema != TokenGraphContractSchemaV1 {
		issues = append(issues, fmt.Sprintf("token_graph contract schema is %q, want %s", contract.Schema, TokenGraphContractSchemaV1))
	}
	if contract.Status != "current" {
		issues = append(issues, fmt.Sprintf("token_graph contract status is %q, want current", contract.Status))
	}
	if contract.SurfaceScope != "surface-token-graph-linux-web" {
		issues = append(issues, fmt.Sprintf("token_graph contract surface_scope is %q, want surface-token-graph-linux-web", contract.SurfaceScope))
	}
	if contract.SourceOfTruth.Module != "lib.core.morph" {
		issues = append(issues, fmt.Sprintf("token_graph source_of_truth.module is %q, want lib.core.morph", contract.SourceOfTruth.Module))
	}
	if contract.SourceOfTruth.Namespace != "tetra.surface.morph.app" {
		issues = append(issues, fmt.Sprintf("token_graph source_of_truth.namespace is %q, want tetra.surface.morph.app", contract.SourceOfTruth.Namespace))
	}
	if contract.SourceOfTruth.Source != "capsule" {
		issues = append(issues, fmt.Sprintf("token_graph source_of_truth.source is %q, want capsule", contract.SourceOfTruth.Source))
	}
	if !contract.SourceOfTruth.SingleTokenGraph {
		issues = append(issues, "token_graph source_of_truth requires single_token_graph")
	}
	if !contract.SourceOfTruth.ExplicitImports {
		issues = append(issues, "token_graph source_of_truth requires explicit_imports")
	}
	if !contract.SourceOfTruth.NoGlobalCascade {
		issues = append(issues, "token_graph source_of_truth requires no_global_cascade")
	}
	if contract.SourceOfTruth.MultipleColorSources {
		issues = append(issues, "token_graph source_of_truth rejects multiple_color_sources")
	}
	for _, category := range requiredTokenGraphCategories() {
		if !containsNormalized(contract.RequiredCategories, category) {
			issues = append(issues, fmt.Sprintf("token_graph contract required_categories missing %s", category))
		}
	}
	for _, token := range requiredTokenGraphTokens() {
		if !containsNormalized(contract.RequiredTokens, token) {
			issues = append(issues, fmt.Sprintf("token_graph contract required_tokens missing %s", token))
		}
	}
	if len(contract.ReferenceSources) == 0 {
		issues = append(issues, "token_graph contract reference_sources are required")
	}
	if len(contract.AllowedRawLiteralScopes) == 0 {
		issues = append(issues, "token_graph contract allowed_raw_literal_scopes are required")
	}
	for _, runtime := range []string{"CSS cascade runtime", "DOM style runtime", "React runtime", "Electron runtime", "platform-native widgets"} {
		if !containsTextFoldTokenGraph(contract.ForbiddenRuntimeModels, runtime) {
			issues = append(issues, fmt.Sprintf("token_graph contract forbidden_runtime_models missing %s", runtime))
		}
	}
	if !sameStringSetFoldTokenGraph(contract.OverrideOrder, requiredTokenGraphOverrideOrder()) {
		issues = append(issues, "token_graph contract override_order must be [base theme density variant state local]")
	}
	issues = append(issues, validateTokenGraphDensityMappings(contract.DensityDPI, contract.RequiredTokens, "contract")...)
	issues = append(issues, validateTokenGraphDiagnostics(contract.DiagnosticsRequired, contract.NegativeGuards, "contract")...)
	for _, nonclaim := range []string{"no CSS cascade runtime", "no React runtime", "no Electron runtime", "no DOM style runtime", "no platform-native widgets"} {
		if !containsTextFoldTokenGraph(contract.NonClaims, nonclaim) {
			issues = append(issues, fmt.Sprintf("token_graph contract nonclaims missing %q", nonclaim))
		}
	}
	return issues
}

func validateTokenGraphReport(contract TokenGraphContract, report Report) []string {
	var issues []string
	if report.Morph == nil {
		return []string{"token_graph report requires morph evidence"}
	}
	morph := report.Morph
	if morph.Module != contract.SourceOfTruth.Module {
		issues = append(issues, fmt.Sprintf("token_graph report morph.module is %q, want %s", morph.Module, contract.SourceOfTruth.Module))
	}
	if morph.Capsule.Namespace != contract.SourceOfTruth.Namespace {
		issues = append(issues, fmt.Sprintf("token_graph report capsule namespace is %q, want %s", morph.Capsule.Namespace, contract.SourceOfTruth.Namespace))
	}
	if !morph.Capsule.ExplicitImports || !morph.Capsule.NoGlobalCascade {
		issues = append(issues, "token_graph report capsule must prove explicit imports and no global cascade")
	}
	if morph.TokenGraph == nil {
		return append(issues, "token_graph report morph token_graph is required")
	}
	graph := morph.TokenGraph
	if graph.Schema != "tetra.surface.morph.token-graph.v1" {
		issues = append(issues, fmt.Sprintf("token_graph report schema is %q, want tetra.surface.morph.token-graph.v1", graph.Schema))
	}
	if graph.Namespace != contract.SourceOfTruth.Namespace {
		issues = append(issues, fmt.Sprintf("token_graph report namespace is %q, want %s", graph.Namespace, contract.SourceOfTruth.Namespace))
	}
	if graph.SourceOfTruth != contract.SourceOfTruth.Source {
		issues = append(issues, fmt.Sprintf("token_graph report source_of_truth is %q, want %s", graph.SourceOfTruth, contract.SourceOfTruth.Source))
	}
	if !graph.ExplicitImports || !graph.NoGlobalCascade {
		issues = append(issues, "token_graph report requires explicit_imports and no_global_cascade")
	}
	if !sameStringSetFoldTokenGraph(graph.FixedOverrideOrder, requiredTokenGraphOverrideOrder()) {
		issues = append(issues, "token_graph report fixed_override_order must be [base theme density variant state local]")
	}
	if graph.Hash != morph.TokenGraphHash || !validSHA256Digest(graph.Hash) {
		issues = append(issues, "token_graph report hash must match morph token_graph_hash and be sha256 evidence")
	}
	for _, category := range contract.RequiredCategories {
		if !containsNormalized(graph.Categories, category) {
			issues = append(issues, fmt.Sprintf("token_graph report categories missing %s", category))
		}
	}
	tokenIDs := map[string]MorphTokenReport{}
	for _, token := range graph.Tokens {
		id := strings.TrimSpace(token.ID)
		if id == "" {
			issues = append(issues, "token_graph report token id is required")
			continue
		}
		if previous, ok := tokenIDs[id]; ok {
			issues = append(issues, fmt.Sprintf("token_graph report duplicate token %s from %s and %s", id, previous.Source, token.Source))
		}
		tokenIDs[id] = token
		if token.Source != contract.SourceOfTruth.Source {
			issues = append(issues, fmt.Sprintf("token_graph report token %s source is %q, want %s", id, token.Source, contract.SourceOfTruth.Source))
		}
		if !containsNormalized(graph.Categories, token.Category) {
			issues = append(issues, fmt.Sprintf("token_graph report token %s category %q is not declared", id, token.Category))
		}
		if !validSHA256Digest(token.Hash) {
			issues = append(issues, fmt.Sprintf("token_graph report token %s hash must be sha256 evidence", id))
		}
	}
	for _, token := range contract.RequiredTokens {
		if _, ok := tokenIDs[token]; !ok {
			issues = append(issues, fmt.Sprintf("token_graph report required token missing %s", token))
		}
	}
	issues = append(issues, validateTokenGraphMaterialRefs(morph.Materials, tokenIDs)...)
	issues = append(issues, validateTokenGraphAssetRefs(morph.AssetRefs, tokenIDs)...)
	if graph.RawLiteralsInAppCode {
		issues = append(issues, "token_graph report rejects raw literals in app code")
	}
	if graph.FallbackToRandomDefault {
		issues = append(issues, "token_graph report rejects fallback-to-random-default")
	}
	if !graph.AliasCycleRejected || !graph.DuplicateSourceRejected || !graph.UnresolvedFallbackRejected {
		issues = append(issues, "token_graph report requires alias_cycle, duplicate_source, and unresolved_fallback rejection")
	}
	issues = append(issues, validateTokenGraphDensityMappings(graph.DensityDPI, mapKeys(tokenIDs), "report")...)
	issues = append(issues, validateMorphTokenGraphDiagnostics(graph.Diagnostics)...)
	return issues
}

func validateTokenGraphMaterialRefs(materials []MorphMaterialReport, tokens map[string]MorphTokenReport) []string {
	var issues []string
	if len(materials) == 0 {
		return []string{"token_graph report materials are required"}
	}
	for _, material := range materials {
		for field, token := range map[string]string{
			"fill":    material.Fill,
			"border":  material.Border,
			"radius":  material.Radius,
			"shadow":  material.Shadow,
			"overlay": material.Overlay,
		} {
			token = strings.TrimSpace(token)
			if token == "" {
				continue
			}
			if _, ok := tokens[token]; !ok {
				issues = append(issues, fmt.Sprintf("token_graph report material %s missing token %s for %s", material.Name, token, field))
			}
		}
	}
	return issues
}

func validateTokenGraphAssetRefs(refs []MorphAssetRefReport, tokens map[string]MorphTokenReport) []string {
	var issues []string
	for _, ref := range refs {
		if strings.TrimSpace(ref.TintToken) != "" {
			if _, ok := tokens[ref.TintToken]; !ok {
				issues = append(issues, fmt.Sprintf("token_graph report asset_ref %s missing tint token %s", ref.ID, ref.TintToken))
			}
		}
		if strings.TrimSpace(ref.FallbackID) != "" {
			fallbackToken := "assets." + strings.TrimSpace(ref.FallbackID)
			if _, ok := tokens[fallbackToken]; !ok {
				issues = append(issues, fmt.Sprintf("token_graph report asset_ref %s missing fallback token %s", ref.ID, fallbackToken))
			}
		}
	}
	return issues
}

func validateTokenGraphReferenceSources(contract TokenGraphContract, root string) []string {
	if strings.TrimSpace(root) == "" {
		return nil
	}
	var issues []string
	for _, source := range contract.ReferenceSources {
		clean := normalizeEvidencePath(source)
		path := filepath.Join(root, filepath.FromSlash(clean))
		raw, err := os.ReadFile(path)
		if err != nil {
			issues = append(issues, fmt.Sprintf("token_graph reference source %s cannot be read: %v", clean, err))
			continue
		}
		if tokenGraphPathAllowsRawLiterals(clean, contract.AllowedRawLiteralScopes) {
			continue
		}
		if sourceHasRawStyleLiteral(string(raw)) {
			issues = append(issues, fmt.Sprintf("token_graph reference source %s contains raw literal outside allowed scopes", clean))
		}
	}
	return issues
}

func validateTokenGraphDensityMappings(mappings []MorphDensityDPIReport, tokenIDs []string, label string) []string {
	var issues []string
	if len(mappings) == 0 {
		return []string{fmt.Sprintf("token_graph %s density_dpi mappings are required", label)}
	}
	for _, target := range []string{"headless", "linux-x64-real-window", "wasm32-web-browser-canvas"} {
		found := false
		for _, mapping := range mappings {
			if mapping.Target != target {
				continue
			}
			found = true
			if !containsNormalized(tokenIDs, mapping.Token) {
				issues = append(issues, fmt.Sprintf("token_graph %s density_dpi target %s token %s is not declared", label, target, mapping.Token))
			}
			if mapping.TargetDPI < 96 {
				issues = append(issues, fmt.Sprintf("token_graph %s density_dpi target %s target_dpi is %d, want >= 96", label, target, mapping.TargetDPI))
			}
			if mapping.ScaleMilli < 1000 || mapping.ScaleMilli > 4000 {
				issues = append(issues, fmt.Sprintf("token_graph %s density_dpi target %s scale_milli is %d, want 1000..4000", label, target, mapping.ScaleMilli))
			}
			if normalizeTokenGraphName(mapping.RoundingPolicy) != "integer_half_up_v1" {
				issues = append(issues, fmt.Sprintf("token_graph %s density_dpi target %s rounding_policy is %q, want integer-half-up-v1", label, target, mapping.RoundingPolicy))
			}
		}
		if !found {
			issues = append(issues, fmt.Sprintf("token_graph %s density_dpi missing target %s", label, target))
		}
	}
	return issues
}

func validateTokenGraphDiagnostics(required []string, guards TokenGraphNegativeGuardsReport, label string) []string {
	var issues []string
	requiredNames := []string{"alias_cycle", "missing_token", "duplicate_source", "raw_literal", "unresolved_fallback", "css_runtime", "multiple_color_sources", "override_order", "density_dpi"}
	for _, name := range requiredNames {
		if !containsNormalized(required, name) {
			issues = append(issues, fmt.Sprintf("token_graph %s diagnostics_required missing %s", label, name))
		}
	}
	checks := []struct {
		name string
		ok   bool
	}{
		{"alias_cycle_rejected", guards.AliasCycleRejected},
		{"missing_token_rejected", guards.MissingTokenRejected},
		{"duplicate_source_rejected", guards.DuplicateSourceRejected},
		{"raw_literal_rejected", guards.RawLiteralRejected},
		{"unresolved_fallback_rejected", guards.UnresolvedFallbackRejected},
		{"css_runtime_rejected", guards.CSSRuntimeRejected},
		{"multiple_color_sources_rejected", guards.MultipleColorSourcesRejected},
		{"override_order_rejected", guards.OverrideOrderRejected},
		{"density_dpi_rejected", guards.DensityDPIRejected},
	}
	for _, check := range checks {
		if !check.ok {
			issues = append(issues, fmt.Sprintf("token_graph %s negative_guards missing %s", label, check.name))
		}
	}
	return issues
}

func validateMorphTokenGraphDiagnostics(diagnostics MorphTokenGraphDiagnosticsReport) []string {
	guards := TokenGraphNegativeGuardsReport{
		AliasCycleRejected:           diagnostics.AliasCycleRejected,
		MissingTokenRejected:         diagnostics.MissingTokenRejected,
		DuplicateSourceRejected:      diagnostics.DuplicateSourceRejected,
		RawLiteralRejected:           diagnostics.RawLiteralRejected,
		UnresolvedFallbackRejected:   diagnostics.UnresolvedFallbackRejected,
		CSSRuntimeRejected:           diagnostics.CSSRuntimeRejected,
		MultipleColorSourcesRejected: diagnostics.MultipleColorSourcesRejected,
		OverrideOrderRejected:        diagnostics.OverrideOrderRejected,
		DensityDPIRejected:           diagnostics.DensityDPIRejected,
	}
	return validateTokenGraphDiagnostics([]string{"alias_cycle", "missing_token", "duplicate_source", "raw_literal", "unresolved_fallback", "css_runtime", "multiple_color_sources", "override_order", "density_dpi"}, guards, "report")
}

func tokenGraphPathAllowsRawLiterals(path string, scopes []TokenGraphRawLiteralScope) bool {
	path = normalizeEvidencePath(path)
	for _, scope := range scopes {
		pattern := normalizeEvidencePath(scope.Path)
		if pattern == "" {
			continue
		}
		if ok, _ := filepath.Match(pattern, path); ok {
			return true
		}
		if pattern == path {
			return true
		}
	}
	return false
}

func sourceHasRawStyleLiteral(source string) bool {
	rawNeedles := []string{"surface.Color(", "draw.Color(", "Color(r:", "#", "rgba(", "rgb("}
	for _, needle := range rawNeedles {
		if strings.Contains(source, needle) {
			return true
		}
	}
	return false
}

func requiredTokenGraphCategories() []string {
	return []string{"color", "space", "radius", "border", "elevation", "opacity", "typography", "motion", "z", "assets", "density"}
}

func requiredTokenGraphTokens() []string {
	return []string{"color.bg", "color.surface", "color.surfaceAlpha", "color.accent", "color.muted", "color.warning", "space.3", "radius.sm", "radius.md", "radius.lg", "border.subtle", "border.glass", "elevation.2", "elevation.3", "opacity.disabled", "type.label", "motion.fast", "motion.soft", "z.base", "assets.gradient.vertical", "assets.icon.fallback", "density.1x"}
}

func requiredTokenGraphOverrideOrder() []string {
	return []string{"base", "theme", "density", "variant", "state", "local"}
}

func normalizeTokenGraphName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, ".", "_")
	return value
}

func containsTextFoldTokenGraph(values []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == want {
			return true
		}
	}
	return false
}

func sameStringSetFoldTokenGraph(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if strings.ToLower(strings.TrimSpace(got[i])) != strings.ToLower(strings.TrimSpace(want[i])) {
			return false
		}
	}
	return true
}

func mapKeys(tokens map[string]MorphTokenReport) []string {
	keys := make([]string, 0, len(tokens))
	for key := range tokens {
		keys = append(keys, key)
	}
	return keys
}
