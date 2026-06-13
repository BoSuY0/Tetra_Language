package surface

import (
	"fmt"
	"strings"
)

type MorphReport struct {
	Schema           string                             `json:"schema"`
	QualityLevel     string                             `json:"quality_level"`
	Source           string                             `json:"source"`
	Module           string                             `json:"module"`
	SurfaceScope     string                             `json:"surface_scope"`
	Experimental     bool                               `json:"experimental"`
	ProductionClaim  bool                               `json:"production_claim"`
	GitHead          string                             `json:"git_head"`
	GitDirty         bool                               `json:"git_dirty"`
	CapsuleHash      string                             `json:"capsule_hash"`
	TokenGraphHash   string                             `json:"token_graph_hash"`
	Capsule          MorphCapsuleReport                 `json:"capsule"`
	TokenGraph       *MorphTokenGraphReport             `json:"token_graph,omitempty"`
	Materials        []MorphMaterialReport              `json:"materials,omitempty"`
	LayoutModes      []string                           `json:"layout_modes,omitempty"`
	TypographyRoles  []string                           `json:"typography_roles,omitempty"`
	AssetRefs        []MorphAssetRefReport              `json:"asset_refs,omitempty"`
	Affordances      []MorphAffordanceReport            `json:"affordances,omitempty"`
	StateLenses      []MorphStateLensReport             `json:"state_lenses,omitempty"`
	MotionPresets    []MorphMotionPresetReport          `json:"motion_presets,omitempty"`
	Recipes          []MorphRecipeReport                `json:"recipes,omitempty"`
	RecipeExpansions []MorphRecipeExpansionReport       `json:"recipe_expansions,omitempty"`
	RecipeApps       []MorphRecipeAppReport             `json:"recipe_apps,omitempty"`
	Accessibility    MorphAccessibilityProjectionReport `json:"accessibility"`
	EvidenceContract MorphEvidenceContractReport        `json:"evidence_contract"`
	MemoryBudget     MorphMemoryBudgetReport            `json:"memory_budget"`
	NegativeGuards   MorphNegativeGuardsReport          `json:"negative_guards"`
	NonClaims        []string                           `json:"nonclaims,omitempty"`
}

type MorphCapsuleReport struct {
	Namespace       string   `json:"namespace"`
	Version         string   `json:"version"`
	CapsuleHash     string   `json:"capsule_hash"`
	Imports         []string `json:"imports"`
	ExplicitImports bool     `json:"explicit_imports"`
	NoGlobalCascade bool     `json:"no_global_cascade"`
}

type MorphTokenGraphReport struct {
	Schema                     string                           `json:"schema"`
	Namespace                  string                           `json:"namespace"`
	Version                    string                           `json:"version"`
	Hash                       string                           `json:"hash"`
	SourceOfTruth              string                           `json:"source_of_truth,omitempty"`
	ExplicitImports            bool                             `json:"explicit_imports,omitempty"`
	NoGlobalCascade            bool                             `json:"no_global_cascade,omitempty"`
	FixedOverrideOrder         []string                         `json:"fixed_override_order,omitempty"`
	Categories                 []string                         `json:"categories"`
	Tokens                     []MorphTokenReport               `json:"tokens"`
	DensityDPI                 []MorphDensityDPIReport          `json:"density_dpi,omitempty"`
	Diagnostics                MorphTokenGraphDiagnosticsReport `json:"diagnostics,omitempty"`
	AliasCycleRejected         bool                             `json:"alias_cycle_rejected"`
	DuplicateSourceRejected    bool                             `json:"duplicate_source_rejected"`
	RawLiteralsInAppCode       bool                             `json:"raw_literals_in_app_code"`
	UnresolvedFallbackRejected bool                             `json:"unresolved_fallback_rejected"`
	FallbackToRandomDefault    bool                             `json:"fallback_to_random_default"`
}

type MorphDensityDPIReport struct {
	Target         string `json:"target"`
	Token          string `json:"token"`
	TargetDPI      int    `json:"target_dpi"`
	ScaleMilli     int    `json:"scale_milli"`
	RoundingPolicy string `json:"rounding_policy"`
}

type MorphTokenGraphDiagnosticsReport struct {
	AliasCycleRejected           bool `json:"alias_cycle_rejected,omitempty"`
	MissingTokenRejected         bool `json:"missing_token_rejected,omitempty"`
	DuplicateSourceRejected      bool `json:"duplicate_source_rejected,omitempty"`
	RawLiteralRejected           bool `json:"raw_literal_rejected,omitempty"`
	UnresolvedFallbackRejected   bool `json:"unresolved_fallback_rejected,omitempty"`
	CSSRuntimeRejected           bool `json:"css_runtime_rejected,omitempty"`
	MultipleColorSourcesRejected bool `json:"multiple_color_sources_rejected,omitempty"`
	OverrideOrderRejected        bool `json:"override_order_rejected,omitempty"`
	DensityDPIRejected           bool `json:"density_dpi_rejected,omitempty"`
}

type MorphTokenReport struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Kind     string `json:"kind"`
	Value    string `json:"value"`
	Source   string `json:"source"`
	Hash     string `json:"hash"`
}

type MorphMaterialReport struct {
	Name                    string   `json:"name"`
	PaintStack              []string `json:"paint_stack"`
	Fill                    string   `json:"fill"`
	Border                  string   `json:"border"`
	Radius                  string   `json:"radius"`
	Shadow                  string   `json:"shadow"`
	Overlay                 string   `json:"overlay"`
	UnsupportedBlur         bool     `json:"unsupported_blur"`
	UnsupportedBlurRejected bool     `json:"unsupported_blur_rejected"`
}

type MorphAssetRefReport struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	SHA256     string `json:"sha256"`
	Local      bool   `json:"local"`
	FallbackID string `json:"fallback_id"`
	TintToken  string `json:"tint_token"`
}

type MorphAffordanceReport struct {
	Name                  string `json:"name"`
	Role                  string `json:"role"`
	Focusable             bool   `json:"focusable"`
	Action                string `json:"action"`
	Input                 string `json:"input"`
	ProjectsAccessibility bool   `json:"projects_accessibility"`
}

type MorphStateLensReport struct {
	Selector      string `json:"selector"`
	Property      string `json:"property"`
	Deterministic bool   `json:"deterministic"`
}

type MorphMotionPresetReport struct {
	Name              string   `json:"name"`
	DurationMS        int      `json:"duration_ms"`
	Curve             string   `json:"curve"`
	Properties        []string `json:"properties"`
	ReducedMotion     bool     `json:"reduced_motion"`
	DeterministicTime bool     `json:"deterministic_time"`
}

type MorphRecipeReport struct {
	Name                   string   `json:"name"`
	Output                 string   `json:"output"`
	Slots                  []string `json:"slots"`
	Inputs                 []string `json:"inputs"`
	ExpandsToBlockGraph    bool     `json:"expands_to_block_graph"`
	HiddenAppState         bool     `json:"hidden_app_state"`
	PlatformWidgets        bool     `json:"platform_widgets"`
	CorePrimitivePromotion bool     `json:"core_primitive_promotion"`
}

type MorphRecipeExpansionReport struct {
	Recipe       string   `json:"recipe"`
	BlockIDs     []int    `json:"block_ids"`
	SlotBindings []string `json:"slot_bindings"`
	Variant      string   `json:"variant"`
	Reported     bool     `json:"reported"`
}

type MorphRecipeAppReport struct {
	Source                  string   `json:"source"`
	Module                  string   `json:"module"`
	Recipes                 []string `json:"recipes"`
	ExpandsToBlockGraph     bool     `json:"expands_to_block_graph"`
	BlockCount              int      `json:"block_count"`
	AccessibilityProjection bool     `json:"accessibility_projection"`
	HiddenAppState          bool     `json:"hidden_app_state"`
	ReactRuntime            bool     `json:"react_runtime"`
	ElectronRuntime         bool     `json:"electron_runtime"`
	DOMRuntime              bool     `json:"dom_runtime"`
	PlatformWidgets         bool     `json:"platform_widgets"`
	OutputPrimitives        []string `json:"output_primitives"`
}

type MorphAccessibilityProjectionReport struct {
	Schema                string   `json:"schema"`
	DerivedFromBlockGraph bool     `json:"derived_from_block_graph"`
	SafetyOverridesWin    bool     `json:"safety_overrides_win"`
	SnapshotEvidence      bool     `json:"snapshot_evidence"`
	RequiredFields        []string `json:"required_fields"`
	Roles                 []string `json:"roles"`
}

type MorphEvidenceContractReport struct {
	CapsuleHash       string `json:"capsule_hash"`
	TokenGraphHash    string `json:"token_graph_hash"`
	RecipeExpansions  bool   `json:"recipe_expansions"`
	BlockTree         bool   `json:"block_tree"`
	ResolvedLayout    bool   `json:"resolved_layout"`
	PaintLayers       bool   `json:"paint_layers"`
	TextRuns          bool   `json:"text_runs"`
	MotionFrames      bool   `json:"motion_frames"`
	AssetHashes       bool   `json:"asset_hashes"`
	AccessibilityTree bool   `json:"accessibility_tree"`
	MemoryBudget      bool   `json:"memory_budget"`
	FrameChecksums    bool   `json:"frame_checksums"`
	ArtifactHashes    bool   `json:"artifact_hashes"`
}

type MorphMemoryBudgetReport struct {
	Schema                 string `json:"schema"`
	ExpandedRecipeCount    int    `json:"expanded_recipe_count"`
	BlockCount             int    `json:"block_count"`
	PaintCommandCount      int    `json:"paint_command_count"`
	LayoutPassCount        int    `json:"layout_pass_count"`
	TextRunCount           int    `json:"text_run_count"`
	MotionActiveCount      int    `json:"motion_active_count"`
	GlyphCacheBytes        int    `json:"glyph_cache_bytes"`
	AssetCacheBytes        int    `json:"asset_cache_bytes"`
	LayoutCacheBytes       int    `json:"layout_cache_bytes"`
	FramebufferBytes       int    `json:"framebuffer_bytes"`
	PeakRSSBytes           int    `json:"peak_rss_bytes"`
	AllocCount             int    `json:"alloc_count"`
	FrameCount             int    `json:"frame_count"`
	BoundedCaches          bool   `json:"bounded_caches"`
	UnboundedCacheRejected bool   `json:"unbounded_cache_rejected"`
}

type MorphNegativeGuardsReport struct {
	NoCoreWidgetPrimitives          bool `json:"no_core_widget_primitives"`
	NoDOMUI                         bool `json:"no_dom_ui"`
	NoReact                         bool `json:"no_react"`
	NoElectron                      bool `json:"no_electron"`
	NoUserJS                        bool `json:"no_user_js"`
	NoPlatformWidgets               bool `json:"no_platform_widgets"`
	MissingTokenRejected            bool `json:"missing_token_rejected"`
	AliasCycleRejected              bool `json:"alias_cycle_rejected"`
	DuplicateTokenSourceRejected    bool `json:"duplicate_token_source_rejected"`
	DuplicateRecipeNameRejected     bool `json:"duplicate_recipe_name_rejected"`
	MissingRecipeExpansionRejected  bool `json:"missing_recipe_expansion_rejected"`
	UnresolvedTokenRejected         bool `json:"unresolved_token_rejected"`
	MissingAssetRejected            bool `json:"missing_asset_rejected"`
	UnboundedCacheRejected          bool `json:"unbounded_cache_rejected"`
	FakeMotionRejected              bool `json:"fake_motion_rejected"`
	FakeAccessibilityRejected       bool `json:"fake_accessibility_rejected"`
	UnsupportedTargetRejected       bool `json:"unsupported_target_rejected"`
	DirtyCheckoutProductionRejected bool `json:"dirty_checkout_production_rejected"`
}

func validateMorphEvidence(report Report) []string {
	if report.Morph == nil {
		return nil
	}

	morph := report.Morph
	var issues []string
	if morph.Schema != "tetra.surface.morph.v1" {
		issues = append(issues, fmt.Sprintf("morph schema is %q, want tetra.surface.morph.v1", morph.Schema))
	}
	if morph.QualityLevel != "deterministic-headless-morph-capsule-v1" {
		issues = append(issues, fmt.Sprintf("morph quality_level is %q, want deterministic-headless-morph-capsule-v1", morph.QualityLevel))
	}
	if report.Target != "headless" || report.Runtime != "surface-headless" || report.HostEvidence.Level != "deterministic-headless" {
		issues = append(issues, "morph evidence currently requires deterministic headless Surface runtime evidence")
	}
	if strings.TrimSpace(morph.Source) == "" {
		issues = append(issues, "morph source is required")
	}
	if !isSurfaceMorphReportSource(morph.Source) {
		issues = append(issues, fmt.Sprintf("morph source is %q, want examples/surface_morph_command_palette.tetra or generated Surface project template source", morph.Source))
	}
	if morph.Module != "lib.core.morph" {
		issues = append(issues, fmt.Sprintf("morph module is %q, want lib.core.morph", morph.Module))
	}
	if morph.SurfaceScope != "surface-morph-experimental-linux-web" {
		issues = append(issues, fmt.Sprintf("morph surface_scope is %q, want surface-morph-experimental-linux-web", morph.SurfaceScope))
	}
	if !morph.Experimental {
		issues = append(issues, "morph experimental must be true until clean release-candidate signoff exists")
	}
	if morph.ProductionClaim && morph.GitDirty {
		issues = append(issues, "morph production claim rejected for dirty checkout")
	}
	if morph.ProductionClaim && !isGitHead(morph.GitHead) {
		issues = append(issues, "morph production claim requires git_head evidence")
	}
	if !validSHA256Digest(morph.CapsuleHash) {
		issues = append(issues, "morph capsule_hash must be sha256 evidence")
	}
	if !validSHA256Digest(morph.TokenGraphHash) {
		issues = append(issues, "morph token_graph_hash must be sha256 evidence")
	}
	if morph.Capsule.CapsuleHash != "" && morph.Capsule.CapsuleHash != morph.CapsuleHash {
		issues = append(issues, "morph capsule.capsule_hash must match morph capsule_hash")
	}
	issues = append(issues, validateMorphCapsule(morph.Capsule)...)
	issues = append(issues, validateMorphTokenGraph(morph)...)
	issues = append(issues, validateMorphMaterials(morph.Materials)...)
	issues = append(issues, validateMorphLayoutModes(morph.LayoutModes)...)
	issues = append(issues, validateMorphTypographyRoles(morph.TypographyRoles)...)
	issues = append(issues, validateMorphAssetRefs(morph.AssetRefs, report)...)
	issues = append(issues, validateMorphAffordances(morph.Affordances)...)
	issues = append(issues, validateMorphStateLenses(morph.StateLenses)...)
	issues = append(issues, validateMorphMotionPresets(morph.MotionPresets, report)...)
	issues = append(issues, validateMorphRecipes(morph.Recipes)...)
	issues = append(issues, validateMorphRecipeExpansions(morph.RecipeExpansions, report)...)
	issues = append(issues, validateMorphRecipeApps(morph.RecipeApps, morph.Recipes, morph.RecipeExpansions)...)
	issues = append(issues, validateMorphAccessibilityProjection(morph.Accessibility, report)...)
	issues = append(issues, validateMorphEvidenceContract(morph.EvidenceContract, morph, report)...)
	issues = append(issues, validateMorphMemoryBudget(morph.MemoryBudget, report)...)
	issues = append(issues, validateMorphNegativeGuards(morph.NegativeGuards)...)
	issues = append(issues, validateMorphNonClaims(morph.NonClaims)...)
	if report.BlockSystem == nil {
		issues = append(issues, "morph evidence requires block_system evidence")
	}
	if report.BlockGraph == nil || report.BlockAccessibilityTree == nil {
		issues = append(issues, "morph evidence requires Block graph and accessibility tree evidence")
	}
	if len(report.PaintLayers) == 0 || len(report.PaintCommands) == 0 {
		issues = append(issues, "morph evidence requires resolved paint layer evidence")
	}
	if len(report.LayoutPasses) == 0 || len(report.LayoutConstraints) == 0 {
		issues = append(issues, "morph evidence requires resolved layout evidence")
	}
	if !hasBlockTextEvidence(report) {
		issues = append(issues, "morph evidence requires text run evidence")
	}
	if !hasBlockMotionEvidence(report) {
		issues = append(issues, "morph evidence requires motion frame evidence")
	}
	if !hasBlockAssetEvidence(report) {
		issues = append(issues, "morph evidence requires asset hash/cache evidence")
	}
	return issues
}

func isSurfaceMorphReportSource(source string) bool {
	source = normalizeEvidencePath(source)
	return source == "examples/surface_morph_command_palette.tetra" || isSurfaceProjectTemplateSource(source)
}

func validateMorphCapsule(capsule MorphCapsuleReport) []string {
	var issues []string
	if capsule.Namespace != "tetra.surface.morph.app" {
		issues = append(issues, fmt.Sprintf("morph capsule namespace is %q, want tetra.surface.morph.app", capsule.Namespace))
	}
	if strings.TrimSpace(capsule.Version) == "" {
		issues = append(issues, "morph capsule version is required")
	}
	if !validSHA256Digest(capsule.CapsuleHash) {
		issues = append(issues, "morph capsule_hash must be sha256 evidence")
	}
	if !capsule.ExplicitImports {
		issues = append(issues, "morph capsule requires explicit imports")
	}
	if !capsule.NoGlobalCascade {
		issues = append(issues, "morph capsule must prove no global cascade")
	}
	for _, required := range []string{"lib.core.block", "lib.core.morph"} {
		if !contains(capsule.Imports, required) {
			issues = append(issues, fmt.Sprintf("morph capsule imports must include %s", required))
		}
	}
	return issues
}

func validateMorphTokenGraph(morph *MorphReport) []string {
	var issues []string
	if morph.TokenGraph == nil {
		return []string{"morph token_graph is required"}
	}
	graph := morph.TokenGraph
	if graph.Schema != "tetra.surface.morph.token-graph.v1" {
		issues = append(issues, fmt.Sprintf("morph token_graph schema is %q, want tetra.surface.morph.token-graph.v1", graph.Schema))
	}
	if graph.Namespace != morph.Capsule.Namespace {
		issues = append(issues, "morph token_graph namespace must match capsule namespace")
	}
	if graph.Hash != morph.TokenGraphHash {
		issues = append(issues, "morph token_graph hash must match morph token_graph_hash")
	}
	if !validSHA256Digest(graph.Hash) {
		issues = append(issues, "morph token_graph hash must be sha256 evidence")
	}
	for _, category := range []string{"color", "space", "radius", "border", "elevation", "opacity", "typography", "motion", "z", "assets", "density"} {
		if !containsNormalized(graph.Categories, category) {
			issues = append(issues, fmt.Sprintf("morph token_graph categories require %s", category))
		}
	}
	if len(graph.Tokens) == 0 {
		issues = append(issues, "morph token_graph tokens are required")
	}
	seenIDs := map[string]string{}
	for i, token := range graph.Tokens {
		id := strings.TrimSpace(token.ID)
		if id == "" {
			issues = append(issues, fmt.Sprintf("morph token_graph tokens[%d].id is required", i))
		}
		if strings.TrimSpace(token.Category) == "" || !containsNormalized(graph.Categories, token.Category) {
			issues = append(issues, fmt.Sprintf("morph token_graph token %q category %q is not declared", token.ID, token.Category))
		}
		if strings.TrimSpace(token.Kind) == "" {
			issues = append(issues, fmt.Sprintf("morph token_graph token %q kind is required", token.ID))
		}
		if strings.TrimSpace(token.Value) == "" {
			issues = append(issues, fmt.Sprintf("morph token_graph token %q value is required", token.ID))
		}
		if strings.TrimSpace(token.Source) != "capsule" {
			issues = append(issues, fmt.Sprintf("morph token_graph token %q source is %q, want capsule", token.ID, token.Source))
		}
		if !validSHA256Digest(token.Hash) {
			issues = append(issues, fmt.Sprintf("morph token_graph token %q hash must be sha256 evidence", token.ID))
		}
		if previous, ok := seenIDs[id]; ok {
			issues = append(issues, fmt.Sprintf("morph token_graph duplicate token %q from %s and %s", id, previous, token.Source))
		}
		seenIDs[id] = token.Source
	}
	if graph.RawLiteralsInAppCode {
		issues = append(issues, "morph token_graph rejects raw literals in app code")
	}
	if graph.FallbackToRandomDefault {
		issues = append(issues, "morph token_graph rejects fallback-to-random-default")
	}
	if !graph.AliasCycleRejected || !graph.DuplicateSourceRejected || !graph.UnresolvedFallbackRejected {
		issues = append(issues, "morph token_graph negative guards require alias_cycle, duplicate_source, and unresolved_fallback rejection")
	}
	return issues
}

func validateMorphMaterials(materials []MorphMaterialReport) []string {
	var issues []string
	if len(materials) == 0 {
		return []string{"morph materials are required"}
	}
	seenFeatures := map[string]bool{}
	for _, material := range materials {
		if strings.TrimSpace(material.Name) == "" {
			issues = append(issues, "morph material name is required")
		}
		if material.UnsupportedBlur {
			issues = append(issues, fmt.Sprintf("morph material %q must not claim unsupported blur", material.Name))
		}
		if !material.UnsupportedBlurRejected {
			issues = append(issues, fmt.Sprintf("morph material %q must prove unsupported blur diagnostics", material.Name))
		}
		for _, feature := range material.PaintStack {
			seenFeatures[normalizeStateToken(feature)] = true
		}
		if token, ok := forbiddenBlockCorePrimitiveToken(material.Name); ok {
			issues = append(issues, fmt.Sprintf("morph material fake core primitive %s rejected in %q", token, material.Name))
		}
	}
	for _, feature := range []string{"fill", "border", "radius", "shadow", "overlay"} {
		if !seenFeatures[feature] {
			issues = append(issues, fmt.Sprintf("morph materials require %s paint stack evidence", feature))
		}
	}
	return issues
}

func validateMorphLayoutModes(modes []string) []string {
	var issues []string
	for _, mode := range []string{"row", "column", "stack", "grid", "dock", "absolute", "overlay", "scroll"} {
		if !containsNormalized(modes, mode) {
			issues = append(issues, fmt.Sprintf("morph layout_modes require %s", mode))
		}
	}
	return issues
}

func validateMorphTypographyRoles(roles []string) []string {
	var issues []string
	for _, role := range []string{"title", "body", "label", "code"} {
		if !containsNormalized(roles, role) {
			issues = append(issues, fmt.Sprintf("morph typography_roles require %s", role))
		}
	}
	return issues
}

func validateMorphAssetRefs(refs []MorphAssetRefReport, report Report) []string {
	var issues []string
	if len(refs) == 0 {
		issues = append(issues, "morph asset_refs are required")
	}
	seen := map[string]bool{}
	for _, ref := range refs {
		if strings.TrimSpace(ref.ID) == "" {
			issues = append(issues, "morph asset_ref id is required")
		}
		seen[ref.ID] = true
		if ref.Kind != "icon" && ref.Kind != "font" && ref.Kind != "image" {
			issues = append(issues, fmt.Sprintf("morph asset_ref %q kind is %q, want icon, font, or image", ref.ID, ref.Kind))
		}
		if !validSHA256Digest(ref.SHA256) {
			issues = append(issues, fmt.Sprintf("morph asset_ref %q sha256 must be sha256 evidence", ref.ID))
		}
		if !ref.Local {
			issues = append(issues, fmt.Sprintf("morph asset_ref %q must be local", ref.ID))
		}
		if strings.TrimSpace(ref.FallbackID) == "" {
			issues = append(issues, fmt.Sprintf("morph asset_ref %q fallback_id is required", ref.ID))
		}
		if strings.TrimSpace(ref.TintToken) == "" {
			issues = append(issues, fmt.Sprintf("morph asset_ref %q tint_token is required", ref.ID))
		}
	}
	for _, required := range []string{"project.new", "command.search", "status.warning"} {
		if !seen[required] {
			issues = append(issues, fmt.Sprintf("morph asset_refs require %s", required))
		}
	}
	if report.BlockAssetNetworkFetchAllowed {
		issues = append(issues, "morph asset evidence requires network fetch disabled")
	}
	if !report.BlockAssetCache.Bounded {
		issues = append(issues, "morph asset evidence requires bounded asset cache")
	}
	return issues
}

func validateMorphAffordances(affordances []MorphAffordanceReport) []string {
	var issues []string
	seen := map[string]MorphAffordanceReport{}
	for _, affordance := range affordances {
		seen[normalizeStateToken(affordance.Name)] = affordance
		if strings.TrimSpace(affordance.Role) == "" {
			issues = append(issues, fmt.Sprintf("morph affordance %q role is required", affordance.Name))
		}
		if !affordance.ProjectsAccessibility {
			issues = append(issues, fmt.Sprintf("morph affordance %q must project accessibility", affordance.Name))
		}
	}
	for _, required := range []string{"action", "field.text", "toggle", "navigation", "region", "overlay", "status"} {
		key := normalizeStateToken(required)
		affordance, ok := seen[key]
		if !ok {
			issues = append(issues, fmt.Sprintf("morph affordances require %s", required))
			continue
		}
		if required == "action" && (!affordance.Focusable || strings.TrimSpace(affordance.Action) == "") {
			issues = append(issues, "morph action affordance requires focusable action evidence")
		}
		if required == "field.text" && (!affordance.Focusable || affordance.Input != "editable_text") {
			issues = append(issues, "morph field.text affordance requires editable text evidence")
		}
	}
	return issues
}

func validateMorphStateLenses(lenses []MorphStateLensReport) []string {
	var issues []string
	seen := map[string]bool{}
	for _, lens := range lenses {
		selector := normalizeStateToken(lens.Selector)
		seen[selector] = true
		if strings.TrimSpace(lens.Property) == "" {
			issues = append(issues, fmt.Sprintf("morph state_lens %q property is required", lens.Selector))
		}
		if !lens.Deterministic {
			issues = append(issues, fmt.Sprintf("morph state_lens %q must be deterministic", lens.Selector))
		}
	}
	for _, selector := range []string{"hover", "pressed", "focusvisible", "selected", "disabled", "error", "loading"} {
		if !seen[selector] {
			issues = append(issues, fmt.Sprintf("morph state_lenses require %s", selector))
		}
	}
	return issues
}

func validateMorphMotionPresets(presets []MorphMotionPresetReport, report Report) []string {
	var issues []string
	if len(presets) == 0 {
		return []string{"morph motion_presets are required"}
	}
	hasReduced := false
	for _, preset := range presets {
		if strings.TrimSpace(preset.Name) == "" {
			issues = append(issues, "morph motion_preset name is required")
		}
		if preset.DurationMS <= 0 {
			issues = append(issues, fmt.Sprintf("morph motion_preset %q duration_ms must be positive", preset.Name))
		}
		if strings.TrimSpace(preset.Curve) == "" {
			issues = append(issues, fmt.Sprintf("morph motion_preset %q curve is required", preset.Name))
		}
		for _, property := range []string{"fill", "opacity", "transform"} {
			if !containsNormalized(preset.Properties, property) {
				issues = append(issues, fmt.Sprintf("morph motion_preset %q properties require %s", preset.Name, property))
			}
		}
		if preset.ReducedMotion && preset.DeterministicTime {
			hasReduced = true
		}
	}
	if !hasReduced {
		issues = append(issues, "morph motion_presets require deterministic reduced-motion evidence")
	}
	if report.MotionUnsupportedCSSAnimations {
		issues = append(issues, "morph motion evidence must not claim CSS animation parity")
	}
	return issues
}

func requiredMorphRecipeNames() []string {
	return []string{
		"control.action@1",
		"field.text@1",
		"command.item@1",
		"region.panel@1",
		"form.field@1",
		"nav.item@1",
		"metric.tile@1",
		"dialog.panel@1",
		"toast.notification@1",
		"tab.item@1",
		"list.row@1",
	}
}

func requiredMorphRecipeAppSources() []string {
	return []string{
		"examples/surface_morph_command_palette.tetra",
		"examples/surface_morph_project_dashboard.tetra",
		"examples/surface_morph_settings.tetra",
		"examples/surface_morph_editor_shell.tetra",
		"examples/surface_morph_glass_panel.tetra",
	}
}

func validateMorphRecipes(recipes []MorphRecipeReport) []string {
	var issues []string
	if len(recipes) == 0 {
		return []string{"morph recipes are required"}
	}
	seen := map[string]bool{}
	for _, recipe := range recipes {
		name := strings.TrimSpace(recipe.Name)
		if name == "" {
			issues = append(issues, "morph recipe name is required")
		}
		if seen[name] {
			issues = append(issues, fmt.Sprintf("morph duplicate recipe name %q rejected", name))
		}
		seen[name] = true
		if token, ok := forbiddenBlockCorePrimitiveToken(name); ok {
			issues = append(issues, fmt.Sprintf("morph recipe fake core primitive %s rejected in name %q", token, name))
		}
		if token, ok := forbiddenBlockCorePrimitiveToken(recipe.Output); ok {
			issues = append(issues, fmt.Sprintf("morph recipe fake core primitive %s rejected in output %q", token, recipe.Output))
		}
		if recipe.Output != "Block" {
			issues = append(issues, fmt.Sprintf("morph recipe %q output is %q, want Block", name, recipe.Output))
		}
		if len(recipe.Slots) == 0 {
			issues = append(issues, fmt.Sprintf("morph recipe %q must declare slots", name))
		}
		if len(recipe.Inputs) == 0 {
			issues = append(issues, fmt.Sprintf("morph recipe %q must declare inputs", name))
		}
		if !recipe.ExpandsToBlockGraph {
			issues = append(issues, fmt.Sprintf("morph recipe %q must expand to Block graph", name))
		}
		if recipe.HiddenAppState {
			issues = append(issues, fmt.Sprintf("morph recipe %q must not allocate hidden app state", name))
		}
		if recipe.PlatformWidgets {
			issues = append(issues, fmt.Sprintf("morph recipe %q must not use platform widgets", name))
		}
		if recipe.CorePrimitivePromotion {
			issues = append(issues, fmt.Sprintf("morph recipe %q core primitive promotion rejected", name))
		}
	}
	for _, required := range requiredMorphRecipeNames() {
		if !seen[required] {
			issues = append(issues, fmt.Sprintf("morph recipes require %s", required))
		}
	}
	return issues
}

func validateMorphRecipeExpansions(expansions []MorphRecipeExpansionReport, report Report) []string {
	if len(expansions) == 0 {
		return []string{"morph recipe_expansions are required"}
	}
	var issues []string
	blockIDs := map[int]bool{}
	if report.BlockGraph != nil {
		for _, node := range report.BlockGraph.Nodes {
			blockIDs[node.ID] = true
		}
	}
	seenRecipe := map[string]bool{}
	for _, expansion := range expansions {
		if strings.TrimSpace(expansion.Recipe) == "" {
			issues = append(issues, "morph recipe_expansions recipe is required")
		}
		seenRecipe[expansion.Recipe] = true
		if len(expansion.BlockIDs) == 0 {
			issues = append(issues, fmt.Sprintf("morph recipe_expansions %q block_ids are required", expansion.Recipe))
		}
		for _, blockID := range expansion.BlockIDs {
			if blockID <= 0 {
				issues = append(issues, fmt.Sprintf("morph recipe_expansions %q block_id must be positive", expansion.Recipe))
			} else if len(blockIDs) > 0 && !blockIDs[blockID] {
				issues = append(issues, fmt.Sprintf("morph recipe_expansions %q references missing Block ID %d", expansion.Recipe, blockID))
			}
		}
		if len(expansion.SlotBindings) == 0 {
			issues = append(issues, fmt.Sprintf("morph recipe_expansions %q slot_bindings are required", expansion.Recipe))
		}
		if !expansion.Reported {
			issues = append(issues, fmt.Sprintf("morph recipe_expansions %q must be reported", expansion.Recipe))
		}
	}
	for _, required := range requiredMorphRecipeNames() {
		if !seenRecipe[required] {
			issues = append(issues, fmt.Sprintf("morph recipe_expansions require %s", required))
		}
	}
	return issues
}

func validateMorphRecipeApps(apps []MorphRecipeAppReport, recipes []MorphRecipeReport, expansions []MorphRecipeExpansionReport) []string {
	if len(apps) == 0 {
		return []string{"morph recipe_apps are required"}
	}
	var issues []string
	knownRecipes := map[string]bool{}
	for _, recipe := range recipes {
		knownRecipes[recipe.Name] = true
	}
	expandedRecipes := map[string]bool{}
	for _, expansion := range expansions {
		expandedRecipes[expansion.Recipe] = true
	}
	seenSources := map[string]bool{}
	for _, app := range apps {
		source := normalizeEvidencePath(app.Source)
		seenSources[source] = true
		if !strings.HasPrefix(source, "examples/surface_morph_") || !strings.HasSuffix(source, ".tetra") {
			issues = append(issues, fmt.Sprintf("morph recipe_apps source %q must be a Surface Morph example", app.Source))
		}
		if strings.TrimSpace(app.Module) == "" {
			issues = append(issues, fmt.Sprintf("morph recipe_apps %q module is required", app.Source))
		}
		if len(app.Recipes) == 0 {
			issues = append(issues, fmt.Sprintf("morph recipe_apps %q recipes are required", app.Source))
		}
		for _, recipe := range app.Recipes {
			if !knownRecipes[recipe] {
				issues = append(issues, fmt.Sprintf("morph recipe_apps %q references undeclared recipe %s", app.Source, recipe))
			}
			if !expandedRecipes[recipe] {
				issues = append(issues, fmt.Sprintf("morph recipe_apps %q references recipe %s without expansion report", app.Source, recipe))
			}
		}
		if !app.ExpandsToBlockGraph {
			issues = append(issues, fmt.Sprintf("morph recipe_apps %q must expand to Block graph", app.Source))
		}
		if app.BlockCount <= 0 {
			issues = append(issues, fmt.Sprintf("morph recipe_apps %q block_count must be positive", app.Source))
		}
		if !app.AccessibilityProjection {
			issues = append(issues, fmt.Sprintf("morph recipe_apps %q requires accessibility projection", app.Source))
		}
		if app.HiddenAppState {
			issues = append(issues, fmt.Sprintf("morph recipe_apps %q must not allocate hidden app state", app.Source))
		}
		if app.ReactRuntime {
			issues = append(issues, fmt.Sprintf("morph recipe_apps %q must not use React runtime", app.Source))
		}
		if app.ElectronRuntime {
			issues = append(issues, fmt.Sprintf("morph recipe_apps %q must not use Electron runtime", app.Source))
		}
		if app.DOMRuntime {
			issues = append(issues, fmt.Sprintf("morph recipe_apps %q must not use DOM runtime", app.Source))
		}
		if app.PlatformWidgets {
			issues = append(issues, fmt.Sprintf("morph recipe_apps %q must not use platform widgets", app.Source))
		}
		if !contains(app.OutputPrimitives, "Block") {
			issues = append(issues, fmt.Sprintf("morph recipe_apps %q output_primitives require Block", app.Source))
		}
		for _, primitive := range app.OutputPrimitives {
			if primitive != "Block" {
				issues = append(issues, fmt.Sprintf("morph recipe_apps %q fake output primitive %s rejected", app.Source, primitive))
			}
		}
	}
	for _, required := range requiredMorphRecipeAppSources() {
		if !seenSources[required] {
			issues = append(issues, fmt.Sprintf("morph recipe_apps require %s", required))
		}
	}
	return issues
}

func validateMorphAccessibilityProjection(projection MorphAccessibilityProjectionReport, report Report) []string {
	var issues []string
	if projection.Schema != "tetra.surface.morph.accessibility-projection.v1" {
		issues = append(issues, fmt.Sprintf("morph accessibility schema is %q, want tetra.surface.morph.accessibility-projection.v1", projection.Schema))
	}
	if !projection.DerivedFromBlockGraph {
		issues = append(issues, "morph accessibility must be derived from Block graph")
	}
	if !projection.SafetyOverridesWin {
		issues = append(issues, "morph accessibility safety overrides must win")
	}
	if !projection.SnapshotEvidence {
		issues = append(issues, "morph accessibility snapshot evidence is required")
	}
	for _, field := range []string{"role", "name", "description", "action", "state", "bounds", "focus_order", "reading_order", "labelled_by", "label_for"} {
		if !containsNormalized(projection.RequiredFields, field) {
			issues = append(issues, fmt.Sprintf("morph accessibility required_fields require %s", field))
		}
	}
	for _, role := range []string{"button", "textbox", "checkbox", "navigation", "region", "dialog", "status"} {
		if !containsNormalized(projection.Roles, role) {
			issues = append(issues, fmt.Sprintf("morph accessibility roles require %s", role))
		}
	}
	if report.BlockAccessibilityTree == nil {
		issues = append(issues, "morph accessibility requires block_accessibility_tree")
	}
	return issues
}

func validateMorphEvidenceContract(contract MorphEvidenceContractReport, morph *MorphReport, report Report) []string {
	var issues []string
	if contract.CapsuleHash != morph.CapsuleHash {
		issues = append(issues, "morph evidence_contract capsule_hash must match morph capsule_hash")
	}
	if contract.TokenGraphHash != morph.TokenGraphHash {
		issues = append(issues, "morph evidence_contract token_graph_hash must match morph token_graph_hash")
	}
	required := []struct {
		name string
		ok   bool
	}{
		{"recipe_expansions", contract.RecipeExpansions},
		{"block_tree", contract.BlockTree && report.BlockGraph != nil},
		{"resolved_layout", contract.ResolvedLayout && len(report.LayoutPasses) > 0},
		{"paint_layers", contract.PaintLayers && len(report.PaintLayers) > 0},
		{"text_runs", contract.TextRuns && hasBlockTextEvidence(report)},
		{"motion_frames", contract.MotionFrames && len(report.MotionFrames) > 0},
		{"asset_hashes", contract.AssetHashes && report.BlockAssetManifest != nil},
		{"accessibility_tree", contract.AccessibilityTree && report.BlockAccessibilityTree != nil},
		{"memory_budget", contract.MemoryBudget && report.BlockSystem != nil && report.BlockSystem.MemoryBudget != nil},
		{"frame_checksums", contract.FrameChecksums && len(report.Frames) > 0},
		{"artifact_hashes", contract.ArtifactHashes && len(report.Artifacts) > 0},
	}
	for _, check := range required {
		if !check.ok {
			issues = append(issues, fmt.Sprintf("morph evidence_contract requires %s evidence", check.name))
		}
	}
	return issues
}

func validateMorphMemoryBudget(budget MorphMemoryBudgetReport, report Report) []string {
	var issues []string
	if budget.Schema != "tetra.surface.morph-memory-budget.v1" {
		issues = append(issues, fmt.Sprintf("morph memory_budget schema is %q, want tetra.surface.morph-memory-budget.v1", budget.Schema))
	}
	if budget.ExpandedRecipeCount <= 0 {
		issues = append(issues, "morph memory_budget expanded_recipe_count must be positive")
	}
	if report.BlockSystem != nil && report.BlockSystem.MemoryBudget != nil && budget.BlockCount < report.BlockSystem.MemoryBudget.BlockCount {
		issues = append(issues, fmt.Sprintf("morph memory_budget block_count = %d, want at least Block memory budget block_count %d", budget.BlockCount, report.BlockSystem.MemoryBudget.BlockCount))
	}
	if budget.PaintCommandCount <= 0 || budget.LayoutPassCount <= 0 || budget.TextRunCount <= 0 || budget.FrameCount <= 0 {
		issues = append(issues, "morph memory_budget requires paint_command_count, layout_pass_count, text_run_count, and frame_count")
	}
	if budget.GlyphCacheBytes < 0 || budget.AssetCacheBytes < 0 || budget.LayoutCacheBytes < 0 || budget.FramebufferBytes <= 0 {
		issues = append(issues, "morph memory_budget cache/framebuffer byte fields are invalid")
	}
	if !budget.BoundedCaches || !budget.UnboundedCacheRejected {
		issues = append(issues, "morph memory_budget requires bounded caches and unbounded cache rejection")
	}
	return issues
}

func validateMorphNegativeGuards(guards MorphNegativeGuardsReport) []string {
	missing := []string{}
	checks := []struct {
		name string
		ok   bool
	}{
		{"no_core_widget_primitives", guards.NoCoreWidgetPrimitives},
		{"no_dom_ui", guards.NoDOMUI},
		{"no_react", guards.NoReact},
		{"no_electron", guards.NoElectron},
		{"no_user_js", guards.NoUserJS},
		{"no_platform_widgets", guards.NoPlatformWidgets},
		{"missing_token_rejected", guards.MissingTokenRejected},
		{"alias_cycle_rejected", guards.AliasCycleRejected},
		{"duplicate_token_source_rejected", guards.DuplicateTokenSourceRejected},
		{"duplicate_recipe_name_rejected", guards.DuplicateRecipeNameRejected},
		{"missing_recipe_expansion_rejected", guards.MissingRecipeExpansionRejected},
		{"unresolved_token_rejected", guards.UnresolvedTokenRejected},
		{"missing_asset_rejected", guards.MissingAssetRejected},
		{"unbounded_cache_rejected", guards.UnboundedCacheRejected},
		{"fake_motion_rejected", guards.FakeMotionRejected},
		{"fake_accessibility_rejected", guards.FakeAccessibilityRejected},
		{"unsupported_target_rejected", guards.UnsupportedTargetRejected},
		{"dirty_checkout_production_rejected", guards.DirtyCheckoutProductionRejected},
	}
	for _, check := range checks {
		if !check.ok {
			missing = append(missing, check.name)
		}
	}
	if len(missing) > 0 {
		return []string{fmt.Sprintf("morph negative_guards missing %s", strings.Join(missing, ", "))}
	}
	return nil
}

func validateMorphNonClaims(nonclaims []string) []string {
	var issues []string
	for _, required := range []string{"DOM runtime absent", "React runtime absent", "Electron claim absent", "platform-native widgets absent", "CSS cascade absent"} {
		if !containsTextFold(nonclaims, required) {
			issues = append(issues, fmt.Sprintf("morph nonclaims require %q", required))
		}
	}
	return issues
}
