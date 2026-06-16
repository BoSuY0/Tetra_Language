package surface

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	MorphRenderedBeautyContractSchemaV1 = "tetra.surface.morph-rendered-beauty.contract.v1"
	MorphRenderedBeautyReportSchemaV1   = "tetra.surface.morph-rendered-beauty.v1"
	MorphRenderedBeautyScope            = "surface-morph-rendered-beauty-linux-web"
)

type MorphRenderedBeautyContract struct {
	Schema                  string                            `json:"schema"`
	Status                  string                            `json:"status"`
	ReportSchema            string                            `json:"report_schema"`
	SurfaceScope            string                            `json:"surface_scope"`
	Pipeline                []string                          `json:"pipeline"`
	CorePrimitives          []string                          `json:"core_primitives"`
	ForbiddenCorePrimitives []string                          `json:"forbidden_core_primitives"`
	SupportedTargets        []string                          `json:"supported_targets"`
	UnsupportedTargets      []string                          `json:"unsupported_targets"`
	RequiredEvidence        []string                          `json:"required_evidence"`
	NegativeGuards          MorphRenderedBeautyNegativeGuards `json:"negative_guards"`
	NonClaims               []string                          `json:"nonclaims"`
}

type MorphRenderedBeautyNegativeGuards struct {
	MetadataOnlyRejected             bool `json:"metadata_only_rejected"`
	SelfGoldenRejected               bool `json:"self_golden_rejected"`
	PrecomputedFrameRejected         bool `json:"precomputed_frame_rejected"`
	MissingFrameArtifactRejected     bool `json:"missing_frame_artifact_rejected"`
	NoDOMUI                          bool `json:"no_dom_ui"`
	NoCSSRuntime                     bool `json:"no_css_runtime"`
	NoReactRuntime                   bool `json:"no_react_runtime"`
	NoElectronRuntime                bool `json:"no_electron_runtime"`
	NoNativeWidgets                  bool `json:"no_native_widgets"`
	NoHiddenAppState                 bool `json:"no_hidden_app_state"`
	NonBlockOutputRejected           bool `json:"non_block_output_rejected"`
	DirtyCheckoutProductionRejected  bool `json:"dirty_checkout_production_rejected"`
	UnsupportedTargetRejected        bool `json:"unsupported_target_rejected"`
	RendererOwnedStableProofRequired bool `json:"renderer_owned_stable_proof_required"`
}

type MorphRenderedBeautyReport struct {
	Schema              string                                 `json:"schema"`
	Status              string                                 `json:"status"`
	SurfaceScope        string                                 `json:"surface_scope"`
	Target              string                                 `json:"target"`
	ScenarioName        string                                 `json:"scenario_name"`
	GitHead             string                                 `json:"git_head"`
	GitCommit           string                                 `json:"git_commit"`
	GitDirty            bool                                   `json:"git_dirty"`
	ProductClaim        bool                                   `json:"product_claim"`
	FinalSignoff        bool                                   `json:"final_signoff"`
	CorePrimitives      []string                               `json:"core_primitives"`
	MorphEvidence       MorphRenderedBeautyMorphEvidence       `json:"morph_evidence"`
	BlockSceneSnapshot  MorphRenderedBeautyBlockSceneSnapshot  `json:"block_scene_snapshot"`
	RenderEvidence      MorphRenderedBeautyRenderEvidence      `json:"render_evidence"`
	RendererStableProof MorphRenderedBeautyRendererStableProof `json:"renderer_stable_proof"`
	RenderCommandStream MorphRenderedBeautyRenderCommandStream `json:"render_command_stream"`
	PixelEvidence       MorphRenderedBeautyPixelEvidence       `json:"pixel_evidence"`
	NegativeGuards      MorphRenderedBeautyNegativeGuards      `json:"negative_guards"`
	NonClaims           []string                               `json:"nonclaims"`
}

type MorphRenderedBeautyMorphEvidence struct {
	Source                 string   `json:"source"`
	SourceSHA256           string   `json:"source_sha256"`
	CapsuleHash            string   `json:"capsule_hash"`
	TokenGraphHash         string   `json:"token_graph_hash"`
	TokenCount             int      `json:"token_count"`
	TokenCategories        []string `json:"token_categories"`
	RecipeCount            int      `json:"recipe_count"`
	RecipeExpansionCount   int      `json:"recipe_expansion_count"`
	RecipeNames            []string `json:"recipe_names"`
	ResolvedMorphSceneHash string   `json:"resolved_morph_scene_hash"`
	BlockSceneSnapshotHash string   `json:"block_scene_snapshot_hash"`
}

type MorphRenderedBeautyBlockSceneSnapshot struct {
	Schema               string                                    `json:"schema"`
	SurfaceScope         string                                    `json:"surface_scope"`
	Source               string                                    `json:"source"`
	QualityLevel         string                                    `json:"quality_level"`
	CorePrimitives       []string                                  `json:"core_primitives"`
	CompactPropsOnly     bool                                      `json:"compact_props_only"`
	RecipeExpansionCount int                                       `json:"recipe_expansion_count"`
	NodeCount            int                                       `json:"node_count"`
	RichSpecHash         string                                    `json:"rich_spec_hash"`
	BlockSceneHash       string                                    `json:"block_scene_hash"`
	SpecCoverage         MorphRenderedBeautyBlockSceneSpecCoverage `json:"spec_coverage"`
}

type MorphRenderedBeautyBlockSceneSpecCoverage struct {
	Layout        bool `json:"layout"`
	Paint         bool `json:"paint"`
	Text          bool `json:"text"`
	Image         bool `json:"image"`
	Input         bool `json:"input"`
	Event         bool `json:"event"`
	State         bool `json:"state"`
	Motion        bool `json:"motion"`
	Accessibility bool `json:"accessibility"`
}

type MorphRenderedBeautyRenderEvidence struct {
	CommandStreamHash string `json:"command_stream_hash"`
	CommandCount      int    `json:"command_count"`
	Renderer          string `json:"renderer"`
}

type MorphRenderedBeautyRendererStableProof struct {
	Schema                         string `json:"schema"`
	PixelOwner                     string `json:"pixel_owner"`
	RendererOwned                  bool   `json:"renderer_owned"`
	BridgeOwnedPixels              bool   `json:"bridge_owned_pixels"`
	BlockFirst                     bool   `json:"block_first"`
	DerivedFromRenderCommandStream bool   `json:"derived_from_render_command_stream"`
	RenderCommandStreamHash        string `json:"render_command_stream_hash"`
	BlockSceneHash                 string `json:"block_scene_hash"`
	FrameChecksum                  string `json:"frame_checksum"`
	StablePromotionEligible        bool   `json:"stable_promotion_eligible"`
}

type MorphRenderedBeautyRenderCommandStream struct {
	Schema                        string                             `json:"schema"`
	Source                        string                             `json:"source"`
	SurfaceScope                  string                             `json:"surface_scope"`
	Producer                      string                             `json:"producer"`
	QualityLevel                  string                             `json:"quality_level"`
	Renderer                      string                             `json:"renderer"`
	DerivedFromBlockSceneSnapshot bool                               `json:"derived_from_block_scene_snapshot"`
	BlockSceneHash                string                             `json:"block_scene_hash"`
	FrameChecksum                 string                             `json:"frame_checksum"`
	CommandStreamHash             string                             `json:"command_stream_hash"`
	CommandCount                  int                                `json:"command_count"`
	SourceLinked                  bool                               `json:"source_linked"`
	HandcraftedFixture            bool                               `json:"handcrafted_fixture"`
	Commands                      []MorphRenderedBeautyRenderCommand `json:"commands"`
}

type MorphRenderedBeautyRenderCommand struct {
	Order          int    `json:"order"`
	Command        string `json:"command"`
	Source         string `json:"source"`
	SourceNodeID   string `json:"source_node_id"`
	Recipe         string `json:"recipe"`
	LayerID        string `json:"layer_id"`
	BlockID        int    `json:"block_id"`
	Quality        string `json:"quality"`
	Color          string `json:"color,omitempty"`
	Width          int    `json:"width,omitempty"`
	Blur           int    `json:"blur,omitempty"`
	OffsetX        int    `json:"offset_x,omitempty"`
	OffsetY        int    `json:"offset_y,omitempty"`
	RasterFormat   string `json:"raster_format,omitempty"`
	RasterHash     string `json:"raster_hash,omitempty"`
	RasterWidth    int    `json:"raster_width,omitempty"`
	RasterHeight   int    `json:"raster_height,omitempty"`
	RasterCoverage int    `json:"raster_coverage,omitempty"`
	MarkerOnly     bool   `json:"marker_only,omitempty"`
	Checksum       string `json:"checksum"`
}

type MorphRenderedBeautyPixelEvidence struct {
	FrameArtifact           string `json:"frame_artifact"`
	FrameArtifactSHA256     string `json:"frame_artifact_sha256"`
	FrameChecksum           string `json:"frame_checksum"`
	FrameProducer           string `json:"frame_producer"`
	AppSource               string `json:"app_source"`
	MorphRecipeHash         string `json:"morph_recipe_hash"`
	BlockSceneHash          string `json:"block_scene_hash"`
	RenderCommandStreamHash string `json:"render_command_stream_hash"`
	GoldenArtifact          string `json:"golden_artifact"`
	GoldenArtifactSHA256    string `json:"golden_artifact_sha256"`
	GoldenChecksum          string `json:"golden_checksum"`
	DiffPixels              int    `json:"diff_pixels"`
	DiffRatioMilli          int    `json:"diff_ratio_milli"`
	MaxChannelDelta         int    `json:"max_channel_delta"`
	PrecomputedFixtureFrame bool   `json:"precomputed_fixture_frame"`
}

func ValidateMorphRenderedBeautyContract(raw []byte) error {
	var contract MorphRenderedBeautyContract
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&contract); err != nil {
		return err
	}
	return ValidateMorphRenderedBeautyContractValue(contract)
}

func ValidateMorphRenderedBeautyReport(raw []byte) error {
	var report MorphRenderedBeautyReport
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return err
	}
	return ValidateMorphRenderedBeautyReportValue(report)
}

func ValidateMorphRenderedBeautyContractFile(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return ValidateMorphRenderedBeautyContract(raw)
}

func ValidateMorphRenderedBeautyReportFile(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return ValidateMorphRenderedBeautyReport(raw)
}

func ValidateMorphRenderedBeautyContractValue(contract MorphRenderedBeautyContract) error {
	var issues []string
	if contract.Schema != MorphRenderedBeautyContractSchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %s", contract.Schema, MorphRenderedBeautyContractSchemaV1))
	}
	if contract.Status != "experimental-contract" {
		issues = append(issues, fmt.Sprintf("status is %q, want experimental-contract", contract.Status))
	}
	if contract.ReportSchema != MorphRenderedBeautyReportSchemaV1 {
		issues = append(issues, fmt.Sprintf("report_schema is %q, want %s", contract.ReportSchema, MorphRenderedBeautyReportSchemaV1))
	}
	if contract.SurfaceScope != MorphRenderedBeautyScope {
		issues = append(issues, fmt.Sprintf("surface_scope is %q, want %s", contract.SurfaceScope, MorphRenderedBeautyScope))
	}
	issues = append(issues, requireTextSequence("pipeline", contract.Pipeline, []string{
		"morph_source",
		"token_graph",
		"recipe_expansions",
		"resolved_morph_scene",
		"block_scene_snapshot",
		"render_command_stream",
		"frame_artifact",
		"pixel_golden_comparison",
		"product_claim_gate",
	})...)
	issues = append(issues, validateMorphRenderedBeautyCorePrimitives(contract.CorePrimitives, contract.ForbiddenCorePrimitives)...)
	issues = append(issues, requireTextSet("supported_targets", contract.SupportedTargets, []string{"headless", "linux-x64-real-window", "wasm32-web-browser-canvas"})...)
	issues = append(issues, requireTextSet("unsupported_targets", contract.UnsupportedTargets, []string{"macos", "windows", "wasm32-wasi"})...)
	issues = append(issues, requireTextSet("required_evidence", contract.RequiredEvidence, []string{
		"morph_source_hash",
		"token_graph_hash",
		"token_coverage",
		"recipe_coverage",
		"recipe_expansions",
		"resolved_morph_scene_hash",
		"block_scene_snapshot_hash",
		"block_scene_snapshot_rich_specs",
		"render_command_stream_hash",
		"source_linked_render_command_stream",
		"text_icon_raster_evidence",
		"app_produced_frame",
		"morph_recipe_hash",
		"pixel_block_scene_hash",
		"pixel_render_command_stream_hash",
		"frame_artifact_sha256",
		"golden_artifact_sha256",
		"pixel_diff_metrics",
		"renderer_owned_stable_proof",
		"target_and_scenario_name",
		"same_commit_git_head",
		"same_commit_git_commit",
	})...)
	issues = append(issues, validateMorphRenderedBeautyNegativeGuards(contract.NegativeGuards)...)
	issues = append(issues, requireTextSet("nonclaims", contract.NonClaims, []string{
		"no Electron runtime claim",
		"no React runtime claim",
		"no CSS runtime claim",
		"no DOM-authored UI claim",
		"no GPU renderer production claim",
		"no macOS production claim",
		"no Windows production claim",
	})...)
	return combineMorphRenderedBeautyIssues(issues)
}

func ValidateMorphRenderedBeautyReportValue(report MorphRenderedBeautyReport) error {
	var issues []string
	if report.Schema != MorphRenderedBeautyReportSchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %s", report.Schema, MorphRenderedBeautyReportSchemaV1))
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if report.SurfaceScope != MorphRenderedBeautyScope {
		issues = append(issues, fmt.Sprintf("surface_scope is %q, want %s", report.SurfaceScope, MorphRenderedBeautyScope))
	}
	if !containsMorphRenderedBeautyText([]string{"headless", "linux-x64-real-window", "wasm32-web-browser-canvas"}, report.Target) {
		issues = append(issues, fmt.Sprintf("target %q is not supported for Morph rendered beauty evidence", report.Target))
	}
	if strings.TrimSpace(report.ScenarioName) == "" {
		issues = append(issues, "scenario_name is required")
	}
	if !isMorphRenderedBeautyGitHead(report.GitHead) {
		issues = append(issues, "git_head must be 40 hex characters")
	}
	if !isMorphRenderedBeautyGitHead(report.GitCommit) {
		issues = append(issues, "git_commit must be 40 hex characters")
	}
	if isMorphRenderedBeautyGitHead(report.GitHead) && isMorphRenderedBeautyGitHead(report.GitCommit) && report.GitHead != report.GitCommit {
		issues = append(issues, "git_commit must match git_head")
	}
	if report.ProductClaim && report.GitDirty {
		issues = append(issues, "dirty checkout production claim rejected")
	}
	if report.FinalSignoff && !report.ProductClaim {
		issues = append(issues, "final_signoff requires product_claim")
	}
	issues = append(issues, validateMorphRenderedBeautyCorePrimitives(report.CorePrimitives, []string{"Button", "Card", "TextField", "TextBox", "Sidebar", "Modal"})...)
	issues = append(issues, validateMorphRenderedBeautyMorphEvidence(report.MorphEvidence)...)
	issues = append(issues, validateMorphRenderedBeautyBlockSceneSnapshot(report.BlockSceneSnapshot, report.MorphEvidence)...)
	issues = append(issues, validateMorphRenderedBeautyRenderEvidence(report.RenderEvidence, report.RenderCommandStream)...)
	issues = append(issues, validateMorphRenderedBeautyRenderCommandStream(report.RenderCommandStream, report.BlockSceneSnapshot, report.MorphEvidence)...)
	issues = append(issues, validateMorphRenderedBeautyRendererStableProof(report.RendererStableProof, report)...)
	issues = append(issues, validateMorphRenderedBeautyPixelEvidence(report.PixelEvidence, report)...)
	issues = append(issues, validateMorphRenderedBeautyNegativeGuards(report.NegativeGuards)...)
	issues = append(issues, requireTextSet("nonclaims", report.NonClaims, []string{
		"no Electron runtime claim",
		"no React runtime claim",
		"no CSS runtime claim",
		"no DOM-authored UI claim",
		"no GPU renderer production claim",
		"no macOS production claim",
		"no Windows production claim",
	})...)
	return combineMorphRenderedBeautyIssues(issues)
}

func validateMorphRenderedBeautyCorePrimitives(core []string, forbidden []string) []string {
	var issues []string
	if !containsMorphRenderedBeautyText(core, "Block") {
		issues = append(issues, "core_primitives must include Block")
	}
	for _, primitive := range []string{"Button", "Card", "TextField", "TextBox", "Sidebar", "Modal"} {
		if !containsMorphRenderedBeautyText(forbidden, primitive) {
			issues = append(issues, fmt.Sprintf("forbidden_core_primitives missing %s", primitive))
		}
		if containsMorphRenderedBeautyText(core, primitive) {
			issues = append(issues, fmt.Sprintf("core_primitives must not include %s", primitive))
		}
	}
	return issues
}

func validateMorphRenderedBeautyMorphEvidence(e MorphRenderedBeautyMorphEvidence) []string {
	var issues []string
	if strings.TrimSpace(e.Source) == "" {
		issues = append(issues, "morph_evidence.source is required")
	}
	for _, check := range []struct {
		name  string
		value string
	}{
		{"morph_evidence.source_sha256", e.SourceSHA256},
		{"morph_evidence.capsule_hash", e.CapsuleHash},
		{"morph_evidence.token_graph_hash", e.TokenGraphHash},
		{"morph_evidence.resolved_morph_scene_hash", e.ResolvedMorphSceneHash},
		{"morph_evidence.block_scene_snapshot_hash", e.BlockSceneSnapshotHash},
	} {
		if !validMorphRenderedBeautySHA256(check.value) {
			issues = append(issues, check.name+" must be sha256 evidence")
		}
	}
	if e.RecipeExpansionCount <= 0 {
		issues = append(issues, "morph_evidence.recipe_expansion_count must be positive")
	}
	if e.TokenCount <= 0 {
		issues = append(issues, "morph_evidence.token_count must be positive")
	}
	issues = append(issues, requireTextSet("morph_evidence.token_categories", e.TokenCategories, []string{"color", "space", "radius", "typography", "motion", "assets"})...)
	if e.RecipeCount <= 0 {
		issues = append(issues, "morph_evidence.recipe_count must be positive")
	}
	if len(e.RecipeNames) == 0 {
		issues = append(issues, "morph_evidence.recipe_names coverage is required")
	}
	if e.RecipeCount > 0 && len(e.RecipeNames) != e.RecipeCount {
		issues = append(issues, fmt.Sprintf("morph_evidence.recipe_count = %d, want len(recipe_names) %d", e.RecipeCount, len(e.RecipeNames)))
	}
	if e.RecipeExpansionCount > 0 && e.RecipeCount > 0 && e.RecipeExpansionCount < e.RecipeCount {
		issues = append(issues, "morph_evidence.recipe_expansion_count must cover every reported recipe")
	}
	return issues
}

func validateMorphRenderedBeautyBlockSceneSnapshot(snapshot MorphRenderedBeautyBlockSceneSnapshot, morph MorphRenderedBeautyMorphEvidence) []string {
	var issues []string
	if snapshot.Schema != "tetra.surface.block-scene-snapshot.v1" {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot.schema is %q, want tetra.surface.block-scene-snapshot.v1", snapshot.Schema))
	}
	if snapshot.SurfaceScope != MorphRenderedBeautyScope {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot.surface_scope is %q, want %s", snapshot.SurfaceScope, MorphRenderedBeautyScope))
	}
	if strings.TrimSpace(snapshot.Source) == "" {
		issues = append(issues, "block_scene_snapshot.source is required")
	}
	if strings.TrimSpace(morph.Source) != "" && strings.TrimSpace(snapshot.Source) != strings.TrimSpace(morph.Source) {
		issues = append(issues, "block_scene_snapshot.source must match morph_evidence.source")
	}
	if snapshot.QualityLevel != "rich-renderable-block-scene-v1" {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot.quality_level is %q, want rich-renderable-block-scene-v1", snapshot.QualityLevel))
	}
	if len(snapshot.CorePrimitives) != 1 || !containsMorphRenderedBeautyText(snapshot.CorePrimitives, "Block") {
		issues = append(issues, "block_scene_snapshot.core_primitives must contain only Block")
	}
	for _, primitive := range snapshot.CorePrimitives {
		primitive = strings.TrimSpace(primitive)
		if !strings.EqualFold(primitive, "Block") {
			issues = append(issues, fmt.Sprintf("block_scene_snapshot.core_primitives must not include %s", primitive))
		}
	}
	if snapshot.CompactPropsOnly {
		issues = append(issues, "block_scene_snapshot compact_props_only must be false")
	}
	if snapshot.RecipeExpansionCount <= 0 {
		issues = append(issues, "block_scene_snapshot.recipe_expansion_count must be positive")
	}
	if snapshot.NodeCount <= 0 {
		issues = append(issues, "block_scene_snapshot.node_count must be positive")
	}
	if !validMorphRenderedBeautySHA256(snapshot.RichSpecHash) {
		issues = append(issues, "block_scene_snapshot.rich_spec_hash must be sha256 evidence")
	}
	if !validMorphRenderedBeautySHA256(snapshot.BlockSceneHash) {
		issues = append(issues, "block_scene_snapshot.block_scene_hash must be sha256 evidence")
	}
	if validMorphRenderedBeautySHA256(snapshot.BlockSceneHash) && snapshot.BlockSceneHash != morph.BlockSceneSnapshotHash {
		issues = append(issues, "block_scene_snapshot.block_scene_hash must match morph_evidence.block_scene_snapshot_hash")
	}
	for _, check := range []struct {
		name string
		ok   bool
	}{
		{"layout", snapshot.SpecCoverage.Layout},
		{"paint", snapshot.SpecCoverage.Paint},
		{"text", snapshot.SpecCoverage.Text},
		{"image", snapshot.SpecCoverage.Image},
		{"input", snapshot.SpecCoverage.Input},
		{"event", snapshot.SpecCoverage.Event},
		{"state", snapshot.SpecCoverage.State},
		{"motion", snapshot.SpecCoverage.Motion},
		{"accessibility", snapshot.SpecCoverage.Accessibility},
	} {
		if !check.ok {
			issues = append(issues, fmt.Sprintf("block_scene_snapshot spec_coverage missing %s", check.name))
		}
	}
	return issues
}

func validateMorphRenderedBeautyRenderEvidence(e MorphRenderedBeautyRenderEvidence, stream MorphRenderedBeautyRenderCommandStream) []string {
	var issues []string
	if !validMorphRenderedBeautySHA256(e.CommandStreamHash) {
		issues = append(issues, "render_evidence.command_stream_hash must be sha256 evidence")
	}
	if e.CommandCount <= 0 {
		issues = append(issues, "render_evidence.command_count must be positive")
	}
	if !containsMorphRenderedBeautyText([]string{"software-rgba-headless", "wayland-shm-rgba", "browser-canvas-rgba"}, e.Renderer) {
		issues = append(issues, fmt.Sprintf("render_evidence.renderer %q is not allowed", e.Renderer))
	}
	if strings.TrimSpace(stream.CommandStreamHash) != "" && e.CommandStreamHash != stream.CommandStreamHash {
		issues = append(issues, "render_evidence.command_stream_hash must match render_command_stream.command_stream_hash")
	}
	if stream.CommandCount != 0 && e.CommandCount != stream.CommandCount {
		issues = append(issues, "render_evidence.command_count must match render_command_stream.command_count")
	}
	if strings.TrimSpace(stream.Renderer) != "" && e.Renderer != stream.Renderer {
		issues = append(issues, "render_evidence.renderer must match render_command_stream.renderer")
	}
	return issues
}

func validateMorphRenderedBeautyRenderCommandStream(stream MorphRenderedBeautyRenderCommandStream, snapshot MorphRenderedBeautyBlockSceneSnapshot, morph MorphRenderedBeautyMorphEvidence) []string {
	var issues []string
	if stream.Schema != "tetra.surface.render-command-stream.v1" {
		issues = append(issues, fmt.Sprintf("render_command_stream.schema is %q, want tetra.surface.render-command-stream.v1", stream.Schema))
	}
	if strings.TrimSpace(stream.Source) == "" {
		issues = append(issues, "render_command_stream.source is required")
	}
	if strings.TrimSpace(morph.Source) != "" && strings.TrimSpace(stream.Source) != strings.TrimSpace(morph.Source) {
		issues = append(issues, "render_command_stream.source must match morph_evidence.source")
	}
	if strings.TrimSpace(snapshot.Source) != "" && strings.TrimSpace(stream.Source) != strings.TrimSpace(snapshot.Source) {
		issues = append(issues, "render_command_stream.source must match block_scene_snapshot.source")
	}
	if stream.SurfaceScope != MorphRenderedBeautyScope {
		issues = append(issues, fmt.Sprintf("render_command_stream.surface_scope is %q, want %s", stream.SurfaceScope, MorphRenderedBeautyScope))
	}
	if strings.TrimSpace(stream.Producer) == "" {
		issues = append(issues, "render_command_stream.producer is required")
	}
	if stream.QualityLevel != "deterministic-render-command-stream-v1" {
		issues = append(issues, fmt.Sprintf("render_command_stream.quality_level is %q, want deterministic-render-command-stream-v1", stream.QualityLevel))
	}
	if !containsMorphRenderedBeautyText([]string{"software-rgba-headless", "wayland-shm-rgba", "browser-canvas-rgba"}, stream.Renderer) {
		issues = append(issues, fmt.Sprintf("render_command_stream.renderer %q is not allowed", stream.Renderer))
	}
	if !stream.DerivedFromBlockSceneSnapshot {
		issues = append(issues, "render_command_stream.derived_from_block_scene_snapshot must be true")
	}
	if !validMorphRenderedBeautySHA256(stream.BlockSceneHash) {
		issues = append(issues, "render_command_stream.block_scene_hash must be sha256 evidence")
	}
	if validMorphRenderedBeautySHA256(snapshot.BlockSceneHash) && stream.BlockSceneHash != snapshot.BlockSceneHash {
		issues = append(issues, "render_command_stream.block_scene_hash must match block_scene_snapshot.block_scene_hash")
	}
	if validMorphRenderedBeautySHA256(morph.BlockSceneSnapshotHash) && stream.BlockSceneHash != morph.BlockSceneSnapshotHash {
		issues = append(issues, "render_command_stream.block_scene_hash must match morph_evidence.block_scene_snapshot_hash")
	}
	if !validMorphRenderedBeautySHA256(stream.FrameChecksum) {
		issues = append(issues, "render_command_stream.frame_checksum must be sha256 evidence")
	}
	if !validMorphRenderedBeautySHA256(stream.CommandStreamHash) {
		issues = append(issues, "render_command_stream.command_stream_hash must be sha256 evidence")
	}
	if stream.CommandCount <= 0 {
		issues = append(issues, "render_command_stream.command_count must be positive")
	}
	if stream.CommandCount != len(stream.Commands) {
		issues = append(issues, fmt.Sprintf("render_command_stream.command_count = %d, want len(commands) %d", stream.CommandCount, len(stream.Commands)))
	}
	if !stream.SourceLinked {
		issues = append(issues, "render_command_stream.source_linked must be true")
	}
	if stream.HandcraftedFixture {
		issues = append(issues, "render_command_stream.handcrafted_fixture must be false")
	}
	issues = append(issues, validateMorphRenderedBeautyRenderCommands(stream.Commands, morph.Source)...)
	return issues
}

func validateMorphRenderedBeautyRendererStableProof(proof MorphRenderedBeautyRendererStableProof, report MorphRenderedBeautyReport) []string {
	var issues []string
	if proof.Schema != "tetra.surface.renderer-stable-proof.v1" {
		issues = append(issues, fmt.Sprintf("renderer_stable_proof.schema is %q, want tetra.surface.renderer-stable-proof.v1", proof.Schema))
	}
	if !containsMorphRenderedBeautyText([]string{"surface-renderer", "morph-evidence-bridge"}, proof.PixelOwner) {
		issues = append(issues, fmt.Sprintf("renderer_stable_proof.pixel_owner is %q, want surface-renderer or morph-evidence-bridge", proof.PixelOwner))
	}
	for _, check := range []struct {
		name  string
		value string
	}{
		{"renderer_stable_proof.render_command_stream_hash", proof.RenderCommandStreamHash},
		{"renderer_stable_proof.block_scene_hash", proof.BlockSceneHash},
		{"renderer_stable_proof.frame_checksum", proof.FrameChecksum},
	} {
		if !validMorphRenderedBeautySHA256(check.value) {
			issues = append(issues, check.name+" must be sha256 evidence")
		}
	}
	if validMorphRenderedBeautySHA256(proof.RenderCommandStreamHash) && validMorphRenderedBeautySHA256(report.RenderCommandStream.CommandStreamHash) && proof.RenderCommandStreamHash != report.RenderCommandStream.CommandStreamHash {
		issues = append(issues, "renderer_stable_proof.render_command_stream_hash must match render_command_stream.command_stream_hash")
	}
	if validMorphRenderedBeautySHA256(proof.BlockSceneHash) && validMorphRenderedBeautySHA256(report.BlockSceneSnapshot.BlockSceneHash) && proof.BlockSceneHash != report.BlockSceneSnapshot.BlockSceneHash {
		issues = append(issues, "renderer_stable_proof.block_scene_hash must match block_scene_snapshot.block_scene_hash")
	}
	if validMorphRenderedBeautySHA256(proof.FrameChecksum) && validMorphRenderedBeautySHA256(report.RenderCommandStream.FrameChecksum) && proof.FrameChecksum != report.RenderCommandStream.FrameChecksum {
		issues = append(issues, "renderer_stable_proof.frame_checksum must match render_command_stream.frame_checksum")
	}
	if proof.RendererOwned && proof.BridgeOwnedPixels {
		issues = append(issues, "renderer_stable_proof cannot be both renderer_owned and bridge_owned_pixels")
	}
	if proof.StablePromotionEligible && (!proof.RendererOwned || proof.BridgeOwnedPixels || !proof.BlockFirst || !proof.DerivedFromRenderCommandStream || proof.PixelOwner != "surface-renderer") {
		issues = append(issues, "renderer_stable_proof.stable_promotion_eligible requires renderer-owned stable proof")
	}
	if (report.ProductClaim || report.FinalSignoff) && (!proof.StablePromotionEligible || !proof.RendererOwned || proof.BridgeOwnedPixels || !proof.BlockFirst || !proof.DerivedFromRenderCommandStream || proof.PixelOwner != "surface-renderer") {
		issues = append(issues, "product_claim requires renderer_owned stable proof")
	}
	return issues
}

func validateMorphRenderedBeautyRenderCommands(commands []MorphRenderedBeautyRenderCommand, source string) []string {
	var issues []string
	seenCommands := map[string]bool{}
	lastOrder := 0
	for i, command := range commands {
		name := normalizeMorphRenderedBeautyRenderCommand(command.Command)
		if command.Order != i+1 {
			issues = append(issues, fmt.Sprintf("render_command_stream.commands[%d].order = %d, want %d", i, command.Order, i+1))
		}
		if command.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("render_command_stream command order %d is not strictly greater than previous order %d", command.Order, lastOrder))
		}
		lastOrder = command.Order
		if !isMorphRenderedBeautyRenderCommand(name) {
			issues = append(issues, fmt.Sprintf("render_command_stream.commands[%d].command %q is not supported", i, command.Command))
		}
		if strings.TrimSpace(source) != "" && strings.TrimSpace(command.Source) != strings.TrimSpace(source) {
			issues = append(issues, fmt.Sprintf("render_command_stream.commands[%d].source must match morph_evidence.source", i))
		}
		if strings.TrimSpace(command.SourceNodeID) == "" {
			issues = append(issues, fmt.Sprintf("render_command_stream.commands[%d].source_node_id is required", i))
		}
		if strings.TrimSpace(command.Recipe) == "" {
			issues = append(issues, fmt.Sprintf("render_command_stream.commands[%d].recipe is required", i))
		}
		if command.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("render_command_stream.commands[%d].block_id must be positive", i))
		}
		if strings.TrimSpace(command.LayerID) == "" {
			issues = append(issues, fmt.Sprintf("render_command_stream.commands[%d].layer_id is required", i))
		}
		if strings.TrimSpace(command.Quality) == "" {
			issues = append(issues, fmt.Sprintf("render_command_stream.commands[%d].quality is required", i))
		}
		if name != "radius_clip" && strings.TrimSpace(command.Color) == "" {
			issues = append(issues, fmt.Sprintf("render_command_stream.commands[%d].color is required for renderer-owned pixels", i))
		}
		if name == "text" {
			issues = append(issues, validateMorphRenderedBeautyRasterProof(
				fmt.Sprintf("render_command_stream.commands[%d]", i),
				"builtin-5x7-alpha-mask-v1",
				command.RasterFormat,
				command.RasterHash,
				command.RasterWidth,
				command.RasterHeight,
				command.RasterCoverage,
				command.MarkerOnly,
			)...)
		}
		if name == "icon" {
			issues = append(issues, validateMorphRenderedBeautyRasterProof(
				fmt.Sprintf("render_command_stream.commands[%d]", i),
				"builtin-icon-mask-raster-v1",
				command.RasterFormat,
				command.RasterHash,
				command.RasterWidth,
				command.RasterHeight,
				command.RasterCoverage,
				command.MarkerOnly,
			)...)
		}
		if !validMorphRenderedBeautySHA256(command.Checksum) {
			issues = append(issues, fmt.Sprintf("render_command_stream.commands[%d].checksum must be sha256 evidence", i))
		}
		seenCommands[name] = true
	}
	for _, required := range []string{"fill", "gradient", "image_fill", "border", "radius_clip", "shadow", "overlay", "outline", "text", "icon"} {
		if !seenCommands[required] {
			issues = append(issues, fmt.Sprintf("render_command_stream.commands require %s command", required))
		}
	}
	return issues
}

func validateMorphRenderedBeautyRasterProof(prefix string, want string, format string, hash string, width int, height int, coverage int, markerOnly bool) []string {
	var issues []string
	if markerOnly {
		issues = append(issues, fmt.Sprintf("%s.marker_only must be false for raster evidence", prefix))
	}
	if strings.Contains(strings.ToLower(strings.TrimSpace(format)), "marker") {
		issues = append(issues, fmt.Sprintf("%s.raster_format must not be marker evidence", prefix))
	}
	if format != want {
		issues = append(issues, fmt.Sprintf("%s.raster_format is %q, want %s", prefix, format, want))
	}
	if !validMorphRenderedBeautySHA256(hash) {
		issues = append(issues, fmt.Sprintf("%s.raster_hash must be sha256 evidence", prefix))
	}
	if width <= 0 || height <= 0 {
		issues = append(issues, fmt.Sprintf("%s raster dimensions must be positive", prefix))
	}
	if coverage <= 0 {
		issues = append(issues, fmt.Sprintf("%s.raster_coverage must be positive", prefix))
	}
	if width > 0 && height > 0 && coverage > width*height {
		issues = append(issues, fmt.Sprintf("%s.raster_coverage exceeds raster dimensions", prefix))
	}
	return issues
}

func normalizeMorphRenderedBeautyRenderCommand(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func isMorphRenderedBeautyRenderCommand(value string) bool {
	switch normalizeMorphRenderedBeautyRenderCommand(value) {
	case "fill", "gradient", "image_fill", "border", "radius_clip", "shadow", "overlay", "outline", "text", "icon":
		return true
	default:
		return false
	}
}

func validateMorphRenderedBeautyPixelEvidence(e MorphRenderedBeautyPixelEvidence, report MorphRenderedBeautyReport) []string {
	var issues []string
	if strings.TrimSpace(e.FrameArtifact) == "" {
		issues = append(issues, "pixel_evidence.frame_artifact is required")
	}
	if strings.TrimSpace(e.GoldenArtifact) == "" {
		issues = append(issues, "pixel_evidence.golden_artifact is required")
	}
	if strings.TrimSpace(e.FrameProducer) != "app" {
		issues = append(issues, fmt.Sprintf("pixel_evidence.frame_producer is %q, want app", e.FrameProducer))
	}
	if strings.TrimSpace(e.AppSource) == "" {
		issues = append(issues, "pixel_evidence.app_source is required")
	} else if e.AppSource != report.MorphEvidence.Source {
		issues = append(issues, "pixel_evidence.app_source must match morph_evidence.source")
	} else if e.AppSource != report.BlockSceneSnapshot.Source {
		issues = append(issues, "pixel_evidence.app_source must match block_scene_snapshot.source")
	} else if e.AppSource != report.RenderCommandStream.Source {
		issues = append(issues, "pixel_evidence.app_source must match render_command_stream.source")
	}
	for _, check := range []struct {
		name  string
		value string
	}{
		{"pixel_evidence.frame_artifact_sha256", e.FrameArtifactSHA256},
		{"pixel_evidence.frame_checksum", e.FrameChecksum},
		{"pixel_evidence.morph_recipe_hash", e.MorphRecipeHash},
		{"pixel_evidence.block_scene_hash", e.BlockSceneHash},
		{"pixel_evidence.render_command_stream_hash", e.RenderCommandStreamHash},
		{"pixel_evidence.golden_artifact_sha256", e.GoldenArtifactSHA256},
		{"pixel_evidence.golden_checksum", e.GoldenChecksum},
	} {
		if !validMorphRenderedBeautySHA256(check.value) {
			issues = append(issues, check.name+" must be sha256 evidence")
		}
	}
	if strings.TrimSpace(e.FrameArtifact) != "" && e.FrameArtifact == e.GoldenArtifact {
		issues = append(issues, "self-golden pixel evidence rejected: frame_artifact equals golden_artifact")
	}
	if validMorphRenderedBeautySHA256(e.FrameArtifactSHA256) && e.FrameArtifactSHA256 == e.GoldenArtifactSHA256 {
		issues = append(issues, "self-golden pixel evidence rejected: frame artifact hash equals golden artifact hash")
	}
	if validMorphRenderedBeautySHA256(e.FrameChecksum) && e.FrameChecksum == e.GoldenChecksum {
		issues = append(issues, "self-golden pixel evidence rejected: frame checksum equals golden checksum")
	}
	if validMorphRenderedBeautySHA256(e.FrameChecksum) && validMorphRenderedBeautySHA256(report.RenderCommandStream.FrameChecksum) && e.FrameChecksum != report.RenderCommandStream.FrameChecksum {
		issues = append(issues, "pixel_evidence.frame_checksum must match render_command_stream.frame_checksum")
	}
	if validMorphRenderedBeautySHA256(e.BlockSceneHash) && validMorphRenderedBeautySHA256(report.BlockSceneSnapshot.BlockSceneHash) && e.BlockSceneHash != report.BlockSceneSnapshot.BlockSceneHash {
		issues = append(issues, "pixel_evidence.block_scene_hash must match block_scene_snapshot.block_scene_hash")
	}
	if validMorphRenderedBeautySHA256(e.BlockSceneHash) && validMorphRenderedBeautySHA256(report.MorphEvidence.BlockSceneSnapshotHash) && e.BlockSceneHash != report.MorphEvidence.BlockSceneSnapshotHash {
		issues = append(issues, "pixel_evidence.block_scene_hash must match morph_evidence.block_scene_snapshot_hash")
	}
	if validMorphRenderedBeautySHA256(e.BlockSceneHash) && validMorphRenderedBeautySHA256(report.RenderCommandStream.BlockSceneHash) && e.BlockSceneHash != report.RenderCommandStream.BlockSceneHash {
		issues = append(issues, "pixel_evidence.block_scene_hash must match render_command_stream.block_scene_hash")
	}
	if validMorphRenderedBeautySHA256(e.RenderCommandStreamHash) && validMorphRenderedBeautySHA256(report.RenderEvidence.CommandStreamHash) && e.RenderCommandStreamHash != report.RenderEvidence.CommandStreamHash {
		issues = append(issues, "pixel_evidence.render_command_stream_hash must match render_evidence.command_stream_hash")
	}
	if validMorphRenderedBeautySHA256(e.RenderCommandStreamHash) && validMorphRenderedBeautySHA256(report.RenderCommandStream.CommandStreamHash) && e.RenderCommandStreamHash != report.RenderCommandStream.CommandStreamHash {
		issues = append(issues, "pixel_evidence.render_command_stream_hash must match render_command_stream.command_stream_hash")
	}
	if e.PrecomputedFixtureFrame {
		issues = append(issues, "precomputed fixture frame cannot be product visual evidence")
	}
	if morphRenderedBeautyFixtureFrameArtifactPath(e.FrameArtifact) {
		issues = append(issues, "fixture or precomputed frame artifact cannot be product visual evidence")
	}
	if e.DiffPixels < 0 || e.DiffRatioMilli < 0 || e.MaxChannelDelta < 0 {
		issues = append(issues, "pixel diff metrics must be non-negative")
	}
	return issues
}

func morphRenderedBeautyFixtureFrameArtifactPath(path string) bool {
	clean := strings.ToLower(strings.TrimSpace(path))
	clean = strings.ReplaceAll(clean, "\\", "/")
	for _, marker := range []string{
		"/fixtures/",
		"fixtures/",
		"/fixture/",
		"fixture/",
		"/testdata/",
		"testdata/",
		"precomputed",
		"synthetic",
		"renderblocksystemframesizedrgba",
	} {
		if strings.Contains(clean, marker) {
			return true
		}
	}
	return false
}

func validateMorphRenderedBeautyNegativeGuards(guards MorphRenderedBeautyNegativeGuards) []string {
	var missing []string
	checks := []struct {
		name string
		ok   bool
	}{
		{"metadata_only_rejected", guards.MetadataOnlyRejected},
		{"self_golden_rejected", guards.SelfGoldenRejected},
		{"precomputed_frame_rejected", guards.PrecomputedFrameRejected},
		{"missing_frame_artifact_rejected", guards.MissingFrameArtifactRejected},
		{"no_dom_ui", guards.NoDOMUI},
		{"no_css_runtime", guards.NoCSSRuntime},
		{"no_react_runtime", guards.NoReactRuntime},
		{"no_electron_runtime", guards.NoElectronRuntime},
		{"no_native_widgets", guards.NoNativeWidgets},
		{"no_hidden_app_state", guards.NoHiddenAppState},
		{"non_block_output_rejected", guards.NonBlockOutputRejected},
		{"dirty_checkout_production_rejected", guards.DirtyCheckoutProductionRejected},
		{"unsupported_target_rejected", guards.UnsupportedTargetRejected},
		{"renderer_owned_stable_proof_required", guards.RendererOwnedStableProofRequired},
	}
	for _, check := range checks {
		if !check.ok {
			missing = append(missing, check.name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("negative_guards missing %s", strings.Join(missing, ", "))}
}

func requireTextSequence(field string, got []string, want []string) []string {
	var issues []string
	if len(got) != len(want) {
		issues = append(issues, fmt.Sprintf("%s length is %d, want %d", field, len(got), len(want)))
	}
	for i, value := range want {
		if i >= len(got) {
			issues = append(issues, fmt.Sprintf("%s missing %s", field, value))
			continue
		}
		if strings.TrimSpace(got[i]) != value {
			issues = append(issues, fmt.Sprintf("%s[%d] is %q, want %s", field, i, got[i], value))
		}
	}
	return issues
}

func requireTextSet(field string, got []string, want []string) []string {
	var issues []string
	for _, value := range want {
		if !containsMorphRenderedBeautyText(got, value) {
			issues = append(issues, fmt.Sprintf("%s missing %s", field, value))
		}
	}
	return issues
}

func containsMorphRenderedBeautyText(values []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == want {
			return true
		}
	}
	return false
}

func validMorphRenderedBeautySHA256(value string) bool {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "sha256:") {
		return false
	}
	digest := strings.TrimPrefix(value, "sha256:")
	if len(digest) != 64 {
		return false
	}
	for _, r := range digest {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

func isMorphRenderedBeautyGitHead(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 40 {
		return false
	}
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

func combineMorphRenderedBeautyIssues(issues []string) error {
	if len(issues) == 0 {
		return nil
	}
	sort.Strings(issues)
	return errors.New(strings.Join(issues, "; "))
}
