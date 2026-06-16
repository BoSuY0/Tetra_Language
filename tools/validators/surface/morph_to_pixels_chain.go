package surface

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

type MorphToPixelsChainReport struct {
	ChainID                 string   `json:"chain_id"`
	ReportPath              string   `json:"report_path"`
	Schema                  string   `json:"schema"`
	Status                  string   `json:"status"`
	SurfaceScope            string   `json:"surface_scope"`
	Source                  string   `json:"source"`
	SourceSHA256            string   `json:"source_sha256"`
	Target                  string   `json:"target"`
	ScenarioName            string   `json:"scenario_name"`
	GitHead                 string   `json:"git_head"`
	GitCommit               string   `json:"git_commit"`
	GitDirty                bool     `json:"git_dirty"`
	TokenGraphHash          string   `json:"token_graph_hash"`
	TokenCount              int      `json:"token_count"`
	TokenCategories         []string `json:"token_categories"`
	RecipeCount             int      `json:"recipe_count"`
	RecipeExpansionCount    int      `json:"recipe_expansion_count"`
	RecipeNames             []string `json:"recipe_names"`
	BlockSceneHash          string   `json:"block_scene_hash"`
	BlockSceneNodeCount     int      `json:"block_scene_node_count"`
	RenderCommandStreamHash string   `json:"render_command_stream_hash"`
	RenderCommandCount      int      `json:"render_command_count"`
	Renderer                string   `json:"renderer"`
	FrameArtifact           string   `json:"frame_artifact"`
	FrameArtifactSHA256     string   `json:"frame_artifact_sha256"`
	FrameChecksum           string   `json:"frame_checksum"`
	GoldenArtifact          string   `json:"golden_artifact"`
	GoldenArtifactSHA256    string   `json:"golden_artifact_sha256"`
	GoldenChecksum          string   `json:"golden_checksum"`
	DiffPixels              int      `json:"diff_pixels"`
	DiffRatioMilli          int      `json:"diff_ratio_milli"`
	MaxChannelDelta         int      `json:"max_channel_delta"`
	ProductClaim            bool     `json:"product_claim"`
	FinalSignoff            bool     `json:"final_signoff"`
	Pass                    bool     `json:"pass"`
}

func MorphToPixelsChainFromRenderedBeauty(reportPath string, report MorphRenderedBeautyReport) MorphToPixelsChainReport {
	chain := MorphToPixelsChainReport{
		ReportPath:              reportPath,
		Schema:                  report.Schema,
		Status:                  report.Status,
		SurfaceScope:            report.SurfaceScope,
		Source:                  report.MorphEvidence.Source,
		SourceSHA256:            report.MorphEvidence.SourceSHA256,
		Target:                  report.Target,
		ScenarioName:            report.ScenarioName,
		GitHead:                 report.GitHead,
		GitCommit:               report.GitCommit,
		GitDirty:                report.GitDirty,
		TokenGraphHash:          report.MorphEvidence.TokenGraphHash,
		TokenCount:              report.MorphEvidence.TokenCount,
		TokenCategories:         append([]string(nil), report.MorphEvidence.TokenCategories...),
		RecipeCount:             report.MorphEvidence.RecipeCount,
		RecipeExpansionCount:    report.MorphEvidence.RecipeExpansionCount,
		RecipeNames:             append([]string(nil), report.MorphEvidence.RecipeNames...),
		BlockSceneHash:          report.BlockSceneSnapshot.BlockSceneHash,
		BlockSceneNodeCount:     report.BlockSceneSnapshot.NodeCount,
		RenderCommandStreamHash: report.RenderCommandStream.CommandStreamHash,
		RenderCommandCount:      report.RenderCommandStream.CommandCount,
		Renderer:                report.RenderCommandStream.Renderer,
		FrameArtifact:           report.PixelEvidence.FrameArtifact,
		FrameArtifactSHA256:     report.PixelEvidence.FrameArtifactSHA256,
		FrameChecksum:           report.PixelEvidence.FrameChecksum,
		GoldenArtifact:          report.PixelEvidence.GoldenArtifact,
		GoldenArtifactSHA256:    report.PixelEvidence.GoldenArtifactSHA256,
		GoldenChecksum:          report.PixelEvidence.GoldenChecksum,
		DiffPixels:              report.PixelEvidence.DiffPixels,
		DiffRatioMilli:          report.PixelEvidence.DiffRatioMilli,
		MaxChannelDelta:         report.PixelEvidence.MaxChannelDelta,
		ProductClaim:            report.ProductClaim,
		FinalSignoff:            report.FinalSignoff,
		Pass:                    report.Status == "pass",
	}
	chain.ChainID = morphToPixelsChainID(chain)
	return chain
}

func ValidateMorphToPixelsChainReport(chain MorphToPixelsChainReport, expectedSource string) error {
	issues := validateMorphToPixelsChain("morph_to_pixels", chain, expectedSource)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateMorphToPixelsChain(field string, chain MorphToPixelsChainReport, expectedSource string) []string {
	var issues []string
	if strings.TrimSpace(chain.ChainID) == "" {
		issues = append(issues, field+".chain_id is required")
	}
	if strings.TrimSpace(chain.ReportPath) == "" {
		issues = append(issues, field+".report_path is required")
	}
	if chain.Schema != MorphRenderedBeautyReportSchemaV1 {
		issues = append(issues, fmt.Sprintf("%s.schema is %q, want %s", field, chain.Schema, MorphRenderedBeautyReportSchemaV1))
	}
	if chain.Status != "pass" {
		issues = append(issues, field+".status must be pass")
	}
	if chain.SurfaceScope != MorphRenderedBeautyScope {
		issues = append(issues, fmt.Sprintf("%s.surface_scope is %q, want %s", field, chain.SurfaceScope, MorphRenderedBeautyScope))
	}
	if strings.TrimSpace(chain.Source) == "" {
		issues = append(issues, field+".source is required")
	}
	if strings.TrimSpace(expectedSource) != "" && normalizeEvidencePath(chain.Source) != normalizeEvidencePath(expectedSource) {
		issues = append(issues, field+".source must match the inspected Surface source")
	}
	if !validChecksumLike(chain.SourceSHA256) {
		issues = append(issues, field+".source_sha256 must be sha256 evidence")
	}
	if !containsMorphRenderedBeautyText([]string{"headless", "linux-x64-real-window", "wasm32-web-browser-canvas"}, chain.Target) {
		issues = append(issues, fmt.Sprintf("%s.target %q is not supported", field, chain.Target))
	}
	if strings.TrimSpace(chain.ScenarioName) == "" {
		issues = append(issues, field+".scenario_name is required")
	}
	if !isMorphRenderedBeautyGitHead(chain.GitHead) {
		issues = append(issues, field+".git_head must be 40 hex characters")
	}
	if !isMorphRenderedBeautyGitHead(chain.GitCommit) {
		issues = append(issues, field+".git_commit must be 40 hex characters")
	}
	if isMorphRenderedBeautyGitHead(chain.GitHead) && isMorphRenderedBeautyGitHead(chain.GitCommit) && chain.GitHead != chain.GitCommit {
		issues = append(issues, field+".git_commit must match git_head")
	}
	for _, check := range []struct {
		name  string
		value string
	}{
		{"token_graph_hash", chain.TokenGraphHash},
		{"block_scene_hash", chain.BlockSceneHash},
		{"render_command_stream_hash", chain.RenderCommandStreamHash},
		{"frame_artifact_sha256", chain.FrameArtifactSHA256},
		{"frame_checksum", chain.FrameChecksum},
		{"golden_artifact_sha256", chain.GoldenArtifactSHA256},
		{"golden_checksum", chain.GoldenChecksum},
	} {
		if !validChecksumLike(check.value) {
			issues = append(issues, fmt.Sprintf("%s.%s must be sha256 evidence", field, check.name))
		}
	}
	if chain.TokenCount <= 0 {
		issues = append(issues, field+".token_count must be positive")
	}
	issues = append(issues, requireTextSet(field+".token_categories", chain.TokenCategories, []string{"color", "space", "radius", "typography", "motion", "assets"})...)
	if chain.RecipeCount <= 0 {
		issues = append(issues, field+".recipe_count must be positive")
	}
	if len(chain.RecipeNames) != chain.RecipeCount {
		issues = append(issues, fmt.Sprintf("%s.recipe_names length = %d, want recipe_count %d", field, len(chain.RecipeNames), chain.RecipeCount))
	}
	if chain.RecipeExpansionCount < chain.RecipeCount || chain.RecipeExpansionCount <= 0 {
		issues = append(issues, field+".recipe_expansion_count must cover every recipe")
	}
	if chain.BlockSceneNodeCount <= 0 {
		issues = append(issues, field+".block_scene_node_count must be positive")
	}
	if chain.RenderCommandCount <= 0 {
		issues = append(issues, field+".render_command_count must be positive")
	}
	if strings.TrimSpace(chain.Renderer) == "" {
		issues = append(issues, field+".renderer is required")
	}
	if strings.TrimSpace(chain.FrameArtifact) == "" {
		issues = append(issues, field+".frame_artifact is required")
	}
	if strings.TrimSpace(chain.GoldenArtifact) == "" {
		issues = append(issues, field+".golden_artifact is required")
	}
	if normalizeEvidencePath(chain.FrameArtifact) == normalizeEvidencePath(chain.GoldenArtifact) {
		issues = append(issues, field+" self-golden artifact rejected")
	}
	if validChecksumLike(chain.FrameArtifactSHA256) && chain.FrameArtifactSHA256 == chain.GoldenArtifactSHA256 {
		issues = append(issues, field+" self-golden artifact hash rejected")
	}
	if validChecksumLike(chain.FrameChecksum) && chain.FrameChecksum == chain.GoldenChecksum {
		issues = append(issues, field+" self-golden frame checksum rejected")
	}
	if chain.DiffPixels < 0 || chain.DiffRatioMilli < 0 || chain.MaxChannelDelta < 0 {
		issues = append(issues, field+" diff metrics must be non-negative")
	}
	if chain.FinalSignoff && !chain.ProductClaim {
		issues = append(issues, field+".final_signoff requires product_claim")
	}
	if !chain.Pass {
		issues = append(issues, field+".pass must be true")
	}
	return issues
}

func morphToPixelsChainID(chain MorphToPixelsChainReport) string {
	parts := []string{
		normalizeEvidencePath(chain.Source),
		chain.SourceSHA256,
		chain.GitCommit,
		chain.TokenGraphHash,
		chain.BlockSceneHash,
		chain.RenderCommandStreamHash,
		chain.FrameChecksum,
		chain.GoldenChecksum,
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return "sha256:" + hex.EncodeToString(sum[:])
}
