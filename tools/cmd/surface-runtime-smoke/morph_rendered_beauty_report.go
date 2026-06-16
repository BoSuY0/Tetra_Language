package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"tetra_language/tools/internal/surfacerender"
	"tetra_language/tools/validators/surface"
)

func readVisualRegressionReport(path string) (surface.VisualRegressionReport, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return surface.VisualRegressionReport{}, err
	}
	var report surface.VisualRegressionReport
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return surface.VisualRegressionReport{}, err
	}
	if err := surface.ValidateVisualReport(raw); err != nil {
		return surface.VisualRegressionReport{}, err
	}
	return report, nil
}

func buildMorphRenderedBeautyReport(runtimeReportPath string, runtime surface.Report, visual surface.VisualRegressionReport, scenarioName string) (surface.MorphRenderedBeautyReport, error) {
	if runtime.Morph == nil {
		return surface.MorphRenderedBeautyReport{}, fmt.Errorf("morph_evidence is required for Morph rendered beauty report")
	}
	if runtime.BlockSceneSnapshot == nil {
		return surface.MorphRenderedBeautyReport{}, fmt.Errorf("block_scene_snapshot is required for Morph rendered beauty report")
	}
	if runtime.RenderCommandStream == nil {
		return surface.MorphRenderedBeautyReport{}, fmt.Errorf("render_command_stream is required for Morph rendered beauty report")
	}
	if strings.TrimSpace(scenarioName) == "" {
		return surface.MorphRenderedBeautyReport{}, fmt.Errorf("scenario_name is required for Morph rendered beauty report")
	}
	source := strings.TrimSpace(runtime.Morph.Source)
	if source == "" {
		source = strings.TrimSpace(runtime.Source)
	}
	if source == "" {
		return surface.MorphRenderedBeautyReport{}, fmt.Errorf("morph source is required for Morph rendered beauty report")
	}
	if !sameEvidencePath(source, runtime.Source) {
		return surface.MorphRenderedBeautyReport{}, fmt.Errorf("morph source %q must match runtime report source %q", source, runtime.Source)
	}
	sourceSHA, err := prefixedSHA256File(source)
	if err != nil {
		return surface.MorphRenderedBeautyReport{}, fmt.Errorf("hash Morph source %s: %w", source, err)
	}
	target := morphRenderedBeautyTarget(runtime)
	visualTarget, visualFrame, err := morphRenderedBeautyVisualEvidence(runtimeReportPath, runtime, visual, source, target)
	if err != nil {
		return surface.MorphRenderedBeautyReport{}, err
	}
	if strings.TrimSpace(visualTarget.GitHead) != "" && strings.TrimSpace(runtime.Morph.GitHead) != "" && visualTarget.GitHead != runtime.Morph.GitHead {
		return surface.MorphRenderedBeautyReport{}, fmt.Errorf("visual git_head %q must match morph git_head %q", visualTarget.GitHead, runtime.Morph.GitHead)
	}
	if visualFrame.Checksum != normalizePrefixedSHA256(runtime.RenderCommandStream.FrameChecksum) {
		return surface.MorphRenderedBeautyReport{}, fmt.Errorf("pixel golden frame checksum %s must match render_command_stream.frame_checksum %s", visualFrame.Checksum, runtime.RenderCommandStream.FrameChecksum)
	}
	rendererStableProof := morphRenderedBeautyRendererStableProof(runtime, visualFrame)

	report := surface.MorphRenderedBeautyReport{
		Schema:         surface.MorphRenderedBeautyReportSchemaV1,
		Status:         "pass",
		SurfaceScope:   surface.MorphRenderedBeautyScope,
		Target:         target,
		ScenarioName:   scenarioName,
		GitHead:        runtime.Morph.GitHead,
		GitCommit:      runtime.Morph.GitHead,
		GitDirty:       runtime.Morph.GitDirty,
		ProductClaim:   false,
		FinalSignoff:   false,
		CorePrimitives: []string{"Block"},
		MorphEvidence: surface.MorphRenderedBeautyMorphEvidence{
			Source:                 source,
			SourceSHA256:           sourceSHA,
			CapsuleHash:            runtime.Morph.CapsuleHash,
			TokenGraphHash:         runtime.Morph.TokenGraphHash,
			TokenCount:             morphRenderedBeautyTokenCount(runtime.Morph),
			TokenCategories:        morphRenderedBeautyTokenCategories(runtime.Morph),
			RecipeCount:            len(runtime.Morph.Recipes),
			RecipeExpansionCount:   len(runtime.Morph.RecipeExpansions),
			RecipeNames:            morphRenderedBeautyRecipeNames(runtime.Morph),
			ResolvedMorphSceneHash: prefixedSHA256Text("resolved-morph-scene|" + source + "|" + runtime.Morph.CapsuleHash + "|" + runtime.Morph.TokenGraphHash + "|" + runtime.BlockSceneSnapshot.BlockSceneHash + "|" + runtime.RenderCommandStream.CommandStreamHash),
			BlockSceneSnapshotHash: runtime.BlockSceneSnapshot.BlockSceneHash,
		},
		BlockSceneSnapshot: surface.MorphRenderedBeautyBlockSceneSnapshot{
			Schema:               runtime.BlockSceneSnapshot.Schema,
			SurfaceScope:         runtime.BlockSceneSnapshot.SurfaceScope,
			Source:               runtime.BlockSceneSnapshot.Source,
			QualityLevel:         runtime.BlockSceneSnapshot.QualityLevel,
			CorePrimitives:       runtime.BlockSceneSnapshot.CorePrimitives,
			CompactPropsOnly:     runtime.BlockSceneSnapshot.CompactPropsOnly,
			RecipeExpansionCount: runtime.BlockSceneSnapshot.RecipeExpansionCount,
			NodeCount:            runtime.BlockSceneSnapshot.NodeCount,
			RichSpecHash:         runtime.BlockSceneSnapshot.RichSpecHash,
			BlockSceneHash:       runtime.BlockSceneSnapshot.BlockSceneHash,
			SpecCoverage: surface.MorphRenderedBeautyBlockSceneSpecCoverage{
				Layout:        runtime.BlockSceneSnapshot.SpecCoverage.Layout,
				Paint:         runtime.BlockSceneSnapshot.SpecCoverage.Paint,
				Text:          runtime.BlockSceneSnapshot.SpecCoverage.Text,
				Image:         runtime.BlockSceneSnapshot.SpecCoverage.Image,
				Input:         runtime.BlockSceneSnapshot.SpecCoverage.Input,
				Event:         runtime.BlockSceneSnapshot.SpecCoverage.Event,
				State:         runtime.BlockSceneSnapshot.SpecCoverage.State,
				Motion:        runtime.BlockSceneSnapshot.SpecCoverage.Motion,
				Accessibility: runtime.BlockSceneSnapshot.SpecCoverage.Accessibility,
			},
		},
		RenderEvidence: surface.MorphRenderedBeautyRenderEvidence{
			CommandStreamHash: runtime.RenderCommandStream.CommandStreamHash,
			CommandCount:      runtime.RenderCommandStream.CommandCount,
			Renderer:          runtime.RenderCommandStream.Renderer,
		},
		RendererStableProof: rendererStableProof,
		RenderCommandStream: morphRenderedBeautyCommandStream(runtime.RenderCommandStream),
		PixelEvidence: surface.MorphRenderedBeautyPixelEvidence{
			FrameArtifact:           visualFrame.ArtifactPath,
			FrameArtifactSHA256:     visualFrame.ArtifactSHA256,
			FrameChecksum:           visualFrame.Checksum,
			FrameProducer:           "app",
			AppSource:               source,
			MorphRecipeHash:         morphRenderedBeautyRecipeHash(runtime.Morph),
			BlockSceneHash:          runtime.BlockSceneSnapshot.BlockSceneHash,
			RenderCommandStreamHash: runtime.RenderCommandStream.CommandStreamHash,
			GoldenArtifact:          visualFrame.GoldenArtifactPath,
			GoldenArtifactSHA256:    visualFrame.GoldenArtifactSHA256,
			GoldenChecksum:          visualFrame.GoldenChecksum,
			DiffPixels:              visualFrame.DiffPixels,
			DiffRatioMilli:          visualFrame.DiffRatioMilli,
			MaxChannelDelta:         visualFrame.MaxChannelDelta,
			PrecomputedFixtureFrame: false,
		},
		NegativeGuards: surface.MorphRenderedBeautyNegativeGuards{
			MetadataOnlyRejected:             true,
			SelfGoldenRejected:               true,
			PrecomputedFrameRejected:         true,
			MissingFrameArtifactRejected:     true,
			NoDOMUI:                          true,
			NoCSSRuntime:                     true,
			NoReactRuntime:                   true,
			NoElectronRuntime:                true,
			NoNativeWidgets:                  true,
			NoHiddenAppState:                 true,
			NonBlockOutputRejected:           true,
			DirtyCheckoutProductionRejected:  true,
			UnsupportedTargetRejected:        true,
			RendererOwnedStableProofRequired: true,
		},
		NonClaims: []string{
			"no Electron runtime claim",
			"no React runtime claim",
			"no CSS runtime claim",
			"no DOM-authored UI claim",
			"no GPU renderer production claim",
			"no macOS production claim",
			"no Windows production claim",
		},
	}
	if err := surface.ValidateMorphRenderedBeautyReportValue(report); err != nil {
		return surface.MorphRenderedBeautyReport{}, err
	}
	return report, nil
}

func applyMorphRenderedBeautyProductSignoff(report *surface.MorphRenderedBeautyReport, productClaim bool, finalSignoff bool) error {
	if report == nil {
		return fmt.Errorf("Morph rendered beauty report is required for product signoff")
	}
	if !productClaim && !finalSignoff {
		return nil
	}
	if finalSignoff && !productClaim {
		return fmt.Errorf("Morph rendered beauty final_signoff requires product_claim")
	}
	if productClaim && !finalSignoff {
		return fmt.Errorf("Morph rendered beauty product_claim requires final_signoff")
	}
	if report.GitDirty {
		return fmt.Errorf("Morph rendered beauty product_claim requires clean checkout: git_dirty=true")
	}
	proof := report.RendererStableProof
	if proof.PixelOwner != "surface-renderer" || !proof.RendererOwned || proof.BridgeOwnedPixels || !proof.BlockFirst || !proof.DerivedFromRenderCommandStream || !proof.StablePromotionEligible {
		return fmt.Errorf("Morph rendered beauty product_claim requires renderer-owned stable proof")
	}
	report.ProductClaim = true
	report.FinalSignoff = true
	return nil
}

func morphRenderedBeautyScenarioName(opt smokeOptions) string {
	source := strings.TrimSpace(defaultSurfaceSourcePath(opt))
	if source == "" {
		return strings.TrimSpace(opt.Mode)
	}
	return strings.TrimSpace(opt.Mode) + ":" + source
}

func morphRenderedBeautyVisualEvidence(runtimeReportPath string, runtime surface.Report, visual surface.VisualRegressionReport, source string, target string) (surface.VisualRegressionTargetReport, surface.VisualRegressionFrameReport, error) {
	if len(visual.Apps) == 0 {
		return surface.VisualRegressionTargetReport{}, surface.VisualRegressionFrameReport{}, fmt.Errorf("pixel golden comparison is required for Morph rendered beauty report")
	}
	frameChecksum := normalizePrefixedSHA256(runtime.RenderCommandStream.FrameChecksum)
	for _, app := range visual.Apps {
		if !sameEvidencePath(app.Source, source) {
			continue
		}
		for _, visualTarget := range app.Targets {
			if normalizeTargetName(visualTarget.Target) != normalizeTargetName(target) {
				continue
			}
			if strings.TrimSpace(runtimeReportPath) != "" && strings.TrimSpace(visualTarget.RuntimeReport) != "" && visualTarget.RuntimeReport != runtimeReportPath {
				continue
			}
			for _, frame := range visualTarget.Frames {
				if !frame.Pass {
					continue
				}
				if frame.Checksum == frameChecksum {
					return visualTarget, frame, nil
				}
			}
		}
	}
	return surface.VisualRegressionTargetReport{}, surface.VisualRegressionFrameReport{}, fmt.Errorf("pixel golden comparison missing passing frame for source %s target %s checksum %s", source, target, frameChecksum)
}

func morphRenderedBeautyCommandStream(stream *surface.RenderCommandStreamReport) surface.MorphRenderedBeautyRenderCommandStream {
	out := surface.MorphRenderedBeautyRenderCommandStream{
		Schema:                        stream.Schema,
		Source:                        stream.Source,
		SurfaceScope:                  stream.SurfaceScope,
		Producer:                      stream.Producer,
		QualityLevel:                  stream.QualityLevel,
		Renderer:                      stream.Renderer,
		DerivedFromBlockSceneSnapshot: stream.DerivedFromBlockSceneSnapshot,
		BlockSceneHash:                stream.BlockSceneHash,
		FrameChecksum:                 normalizePrefixedSHA256(stream.FrameChecksum),
		CommandStreamHash:             stream.CommandStreamHash,
		CommandCount:                  stream.CommandCount,
		SourceLinked:                  stream.SourceLinked,
		HandcraftedFixture:            stream.HandcraftedFixture,
	}
	for _, command := range stream.Commands {
		out.Commands = append(out.Commands, surface.MorphRenderedBeautyRenderCommand{
			Order:          command.Order,
			Command:        command.Command,
			Source:         command.Source,
			SourceNodeID:   command.SourceNodeID,
			Recipe:         command.Recipe,
			LayerID:        command.LayerID,
			BlockID:        command.BlockID,
			Quality:        command.Quality,
			Color:          command.Color,
			Width:          command.Width,
			Blur:           command.Blur,
			OffsetX:        command.OffsetX,
			OffsetY:        command.OffsetY,
			RasterFormat:   command.RasterFormat,
			RasterHash:     command.RasterHash,
			RasterWidth:    command.RasterWidth,
			RasterHeight:   command.RasterHeight,
			RasterCoverage: command.RasterCoverage,
			MarkerOnly:     command.MarkerOnly,
			Checksum:       command.Checksum,
		})
	}
	return out
}

func morphRenderedBeautyRendererStableProof(runtime surface.Report, visualFrame surface.VisualRegressionFrameReport) surface.MorphRenderedBeautyRendererStableProof {
	proof := surface.MorphRenderedBeautyRendererStableProof{
		Schema:                         "tetra.surface.renderer-stable-proof.v1",
		PixelOwner:                     "morph-evidence-bridge",
		RendererOwned:                  false,
		BridgeOwnedPixels:              true,
		BlockFirst:                     true,
		DerivedFromRenderCommandStream: false,
		RenderCommandStreamHash:        runtime.RenderCommandStream.CommandStreamHash,
		BlockSceneHash:                 runtime.BlockSceneSnapshot.BlockSceneHash,
		FrameChecksum:                  normalizePrefixedSHA256(runtime.RenderCommandStream.FrameChecksum),
		StablePromotionEligible:        false,
	}
	if runtime.RenderCommandStream == nil || runtime.BlockSceneSnapshot == nil {
		return proof
	}
	rendered, err := surfacerender.RenderCommandStreamRGBA(runtime.RenderCommandStream, visualFrame.Width, visualFrame.Height)
	if err != nil {
		return proof
	}
	if normalizePrefixedSHA256(rendered.Checksum) != normalizePrefixedSHA256(visualFrame.Checksum) {
		return proof
	}
	proof.PixelOwner = "surface-renderer"
	proof.RendererOwned = true
	proof.BridgeOwnedPixels = false
	proof.DerivedFromRenderCommandStream = true
	proof.StablePromotionEligible = true
	proof.FrameChecksum = normalizePrefixedSHA256(rendered.Checksum)
	return proof
}

func morphRenderedBeautyTarget(report surface.Report) string {
	switch {
	case report.Target == "linux-x64" && report.HostEvidence.RealWindow:
		return "linux-x64-real-window"
	case report.Target == "wasm32-web" && report.HostEvidence.BrowserCanvas:
		return "wasm32-web-browser-canvas"
	default:
		return strings.TrimSpace(report.Target)
	}
}

func morphRenderedBeautyTokenCount(morph *surface.MorphReport) int {
	if morph == nil || morph.TokenGraph == nil {
		return 0
	}
	return len(morph.TokenGraph.Tokens)
}

func morphRenderedBeautyTokenCategories(morph *surface.MorphReport) []string {
	if morph == nil || morph.TokenGraph == nil {
		return nil
	}
	values := append([]string{}, morph.TokenGraph.Categories...)
	if len(values) == 0 {
		for _, token := range morph.TokenGraph.Tokens {
			values = append(values, token.Category)
		}
	}
	return uniqueSortedStrings(values)
}

func morphRenderedBeautyRecipeNames(morph *surface.MorphReport) []string {
	if morph == nil {
		return nil
	}
	values := make([]string, 0, len(morph.Recipes))
	for _, recipe := range morph.Recipes {
		values = append(values, recipe.Name)
	}
	if len(values) == 0 {
		for _, expansion := range morph.RecipeExpansions {
			values = append(values, expansion.Recipe)
		}
	}
	return uniqueSortedStrings(values)
}

func morphRenderedBeautyRecipeHash(morph *surface.MorphReport) string {
	var builder strings.Builder
	for _, name := range morphRenderedBeautyRecipeNames(morph) {
		builder.WriteString(name)
		builder.WriteByte('\n')
	}
	if morph != nil {
		for _, expansion := range morph.RecipeExpansions {
			builder.WriteString(expansion.Recipe)
			builder.WriteString(fmt.Sprint(expansion.BlockIDs))
			builder.WriteByte('\n')
		}
	}
	return prefixedSHA256Text(builder.String())
}

func uniqueSortedStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func sameEvidencePath(a string, b string) bool {
	return strings.TrimSpace(strings.ReplaceAll(a, "\\", "/")) == strings.TrimSpace(strings.ReplaceAll(b, "\\", "/"))
}

func normalizeTargetName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	return value
}

func normalizePrefixedSHA256(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 64 && isHexSHA256(value) {
		return "sha256:" + strings.ToLower(value)
	}
	if strings.HasPrefix(value, "sha256:") {
		digest := strings.TrimPrefix(value, "sha256:")
		if len(digest) == 64 && isHexSHA256(digest) {
			return "sha256:" + strings.ToLower(digest)
		}
	}
	return value
}

func isHexSHA256(value string) bool {
	for _, r := range value {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			continue
		}
		return false
	}
	return true
}

func prefixedSHA256Text(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func prefixedSHA256File(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		resolved, resolveErr := resolveSurfaceSourcePath(path)
		if resolveErr != nil {
			return "", err
		}
		raw, err = os.ReadFile(resolved)
	}
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
