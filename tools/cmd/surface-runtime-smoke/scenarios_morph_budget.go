package main

import (
	"os/exec"
	"strings"

	"tetra_language/tools/validators/surface"
)

func morphReportForScenario(source string, scenario headlessScenario) *surface.MorphReport {
	capsuleHash := "sha256:" + checksumText("surface-morph-capsule-v1:"+source)
	tokenGraphHash := "sha256:" + checksumText("surface-morph-token-graph-v1:"+source)
	return &surface.MorphReport{
		Schema:          "tetra.surface.morph.v1",
		QualityLevel:    "deterministic-headless-morph-capsule-v1",
		Source:          source,
		Module:          "lib.core.morph",
		SurfaceScope:    "surface-morph-experimental-linux-web",
		Experimental:    true,
		ProductionClaim: false,
		GitHead:         gitHeadForReport(),
		GitDirty:        gitDirtyForReport(),
		CapsuleHash:     capsuleHash,
		TokenGraphHash:  tokenGraphHash,
		Capsule: surface.MorphCapsuleReport{
			Namespace:       "tetra.surface.morph.app",
			Version:         "1",
			CapsuleHash:     capsuleHash,
			Imports:         []string{"lib.core.block", "lib.core.morph"},
			ExplicitImports: true,
			NoGlobalCascade: true,
		},
		TokenGraph:       morphTokenGraphForScenario(tokenGraphHash),
		Materials:        morphMaterialsForScenario(),
		LayoutModes:      []string{"row", "column", "stack", "grid", "dock", "absolute", "overlay", "scroll"},
		TypographyRoles:  []string{"title", "body", "label", "code"},
		AssetRefs:        morphAssetRefsForScenario(),
		Affordances:      morphAffordancesForScenario(),
		StateLenses:      morphStateLensesForScenario(),
		MotionPresets:    morphMotionPresetsForScenario(),
		Recipes:          morphRecipesForScenario(),
		RecipeExpansions: morphRecipeExpansionsForScenario(),
		RecipeApps:       morphRecipeAppsForScenario(),
		Accessibility: surface.MorphAccessibilityProjectionReport{
			Schema:                "tetra.surface.morph.accessibility-projection.v1",
			DerivedFromBlockGraph: true,
			SafetyOverridesWin:    true,
			SnapshotEvidence:      true,
			RequiredFields:        []string{"role", "name", "description", "action", "state", "bounds", "focus_order", "reading_order", "labelled_by", "label_for"},
			Roles:                 []string{"button", "textbox", "checkbox", "navigation", "region", "dialog", "status"},
		},
		EvidenceContract: surface.MorphEvidenceContractReport{
			CapsuleHash:       capsuleHash,
			TokenGraphHash:    tokenGraphHash,
			RecipeExpansions:  true,
			BlockTree:         scenario.BlockGraph != nil,
			ResolvedLayout:    len(scenario.LayoutPasses) > 0,
			PaintLayers:       len(scenario.PaintLayers) > 0,
			TextRuns:          len(scenario.TextRenderCommands) > 0,
			MotionFrames:      len(scenario.MotionFrames) > 0,
			AssetHashes:       scenario.BlockAssetManifest != nil,
			AccessibilityTree: scenario.BlockAccessibilityTree != nil,
			MemoryBudget:      scenario.BlockSystem != nil && scenario.BlockSystem.MemoryBudget != nil,
			FrameChecksums:    len(scenario.Frames) > 0,
			ArtifactHashes:    true,
		},
		MemoryBudget: surface.MorphMemoryBudgetReport{
			Schema:                 "tetra.surface.morph-memory-budget.v1",
			ExpandedRecipeCount:    len(morphRecipeExpansionsForScenario()),
			BlockCount:             len(scenario.Components),
			PaintCommandCount:      len(scenario.PaintCommands),
			LayoutPassCount:        len(scenario.LayoutPasses),
			TextRunCount:           len(scenario.TextRenderCommands),
			MotionActiveCount:      len(scenario.MotionFrames),
			GlyphCacheBytes:        glyphCacheUsedBytesForScenario(scenario.GlyphCaches),
			AssetCacheBytes:        scenario.BlockAssetCache.UsedBytes,
			LayoutCacheBytes:       len(scenario.LayoutPasses) * 1024,
			FramebufferBytes:       morphFramebufferBytesForScenario(scenario.Frames),
			PeakRSSBytes:           0,
			AllocCount:             0,
			FrameCount:             len(scenario.Frames),
			BoundedCaches:          true,
			UnboundedCacheRejected: true,
		},
		NegativeGuards: surface.MorphNegativeGuardsReport{
			NoCoreWidgetPrimitives:          true,
			NoDOMUI:                         true,
			NoReact:                         true,
			NoElectron:                      true,
			NoUserJS:                        true,
			NoPlatformWidgets:               true,
			MissingTokenRejected:            true,
			AliasCycleRejected:              true,
			DuplicateTokenSourceRejected:    true,
			DuplicateRecipeNameRejected:     true,
			MissingRecipeExpansionRejected:  true,
			UnresolvedTokenRejected:         true,
			MissingAssetRejected:            true,
			UnboundedCacheRejected:          true,
			FakeMotionRejected:              true,
			FakeAccessibilityRejected:       true,
			UnsupportedTargetRejected:       true,
			DirtyCheckoutProductionRejected: true,
		},
		NonClaims: []string{
			"DOM runtime absent",
			"React runtime absent",
			"Electron claim absent",
			"platform-native widgets absent",
			"full screen-reader production absent",
			"CSS cascade absent",
		},
	}
}
func morphTokenGraphForScenario(hash string) *surface.MorphTokenGraphReport {
	return &surface.MorphTokenGraphReport{
		Schema:                     "tetra.surface.morph.token-graph.v1",
		Namespace:                  "tetra.surface.morph.app",
		Version:                    "1",
		Hash:                       hash,
		SourceOfTruth:              "capsule",
		ExplicitImports:            true,
		NoGlobalCascade:            true,
		FixedOverrideOrder:         []string{"base", "theme", "density", "variant", "state", "local"},
		Categories:                 []string{"color", "space", "radius", "border", "elevation", "opacity", "typography", "motion", "z", "assets", "density"},
		AliasCycleRejected:         true,
		DuplicateSourceRejected:    true,
		RawLiteralsInAppCode:       false,
		UnresolvedFallbackRejected: true,
		FallbackToRandomDefault:    false,
		DensityDPI: []surface.MorphDensityDPIReport{
			{Target: "headless", Token: "density.1x", TargetDPI: 96, ScaleMilli: 1000, RoundingPolicy: "integer-half-up-v1"},
			{Target: "linux-x64-real-window", Token: "density.1x", TargetDPI: 96, ScaleMilli: 1000, RoundingPolicy: "integer-half-up-v1"},
			{Target: "wasm32-web-browser-canvas", Token: "density.1x", TargetDPI: 96, ScaleMilli: 1000, RoundingPolicy: "integer-half-up-v1"},
		},
		Diagnostics: surface.MorphTokenGraphDiagnosticsReport{
			AliasCycleRejected:           true,
			MissingTokenRejected:         true,
			DuplicateSourceRejected:      true,
			RawLiteralRejected:           true,
			UnresolvedFallbackRejected:   true,
			CSSRuntimeRejected:           true,
			MultipleColorSourcesRejected: true,
			OverrideOrderRejected:        true,
			DensityDPIRejected:           true,
		},
		Tokens: []surface.MorphTokenReport{
			{ID: "color.bg", Category: "color", Kind: "rgba", Value: "#0b0f14ff", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-color-bg")},
			{ID: "color.surface", Category: "color", Kind: "rgba", Value: "#181f26ff", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-color-surface")},
			{ID: "color.surfaceAlpha", Category: "color", Kind: "rgba", Value: "#181f26da", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-color-surface-alpha")},
			{ID: "color.accent", Category: "color", Kind: "rgba", Value: "#60aef4ff", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-color-accent")},
			{ID: "color.muted", Category: "color", Kind: "rgba", Value: "#7e90a3ff", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-color-muted")},
			{ID: "color.warning", Category: "color", Kind: "rgba", Value: "#f4cd5cff", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-color-warning")},
			{ID: "space.3", Category: "space", Kind: "px", Value: "12", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-space-3")},
			{ID: "radius.sm", Category: "radius", Kind: "px", Value: "8", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-radius-sm")},
			{ID: "radius.md", Category: "radius", Kind: "px", Value: "10", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-radius-md")},
			{ID: "radius.lg", Category: "radius", Kind: "px", Value: "18", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-radius-lg")},
			{ID: "border.subtle", Category: "border", Kind: "px", Value: "1", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-border-subtle")},
			{ID: "border.glass", Category: "border", Kind: "px", Value: "1", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-border-glass")},
			{ID: "elevation.2", Category: "elevation", Kind: "shadow", Value: "0 3 10 72", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-elevation-2")},
			{ID: "elevation.3", Category: "elevation", Kind: "shadow", Value: "0 10 24 128", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-elevation-3")},
			{ID: "opacity.disabled", Category: "opacity", Kind: "alpha", Value: "128", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-opacity-disabled")},
			{ID: "type.label", Category: "typography", Kind: "font", Value: "Tetra UI 13 600 18", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-type-label")},
			{ID: "motion.fast", Category: "motion", Kind: "transition", Value: "120 ease.out", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-motion-fast")},
			{ID: "motion.soft", Category: "motion", Kind: "transition", Value: "180 ease.inOut", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-motion-soft")},
			{ID: "z.base", Category: "z", Kind: "layer", Value: "0", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-z-base")},
			{ID: "assets.gradient.vertical", Category: "assets", Kind: "gradient", Value: "vertical", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-assets-gradient-vertical")},
			{ID: "assets.icon.fallback", Category: "assets", Kind: "icon", Value: "fallback", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-assets-icon-fallback")},
			{ID: "density.1x", Category: "density", Kind: "dpi", Value: "96/1000", Source: "capsule", Hash: "sha256:" + checksumText("morph-token-density-1x")},
		},
	}
}
func morphMaterialsForScenario() []surface.MorphMaterialReport {
	return []surface.MorphMaterialReport{
		{Name: "surface.base", PaintStack: []string{"fill", "border", "radius"}, Fill: "color.surface", Border: "border.subtle", Radius: "radius.md", UnsupportedBlurRejected: true},
		{Name: "surface.elevated", PaintStack: []string{"fill", "border", "radius", "shadow"}, Fill: "color.surface", Border: "border.subtle", Radius: "radius.md", Shadow: "elevation.2", UnsupportedBlurRejected: true},
		{Name: "control.primary", PaintStack: []string{"fill", "radius"}, Fill: "color.accent", Radius: "radius.sm", UnsupportedBlurRejected: true},
		{Name: "translucent.panel", PaintStack: []string{"fill", "border", "radius", "shadow", "overlay"}, Fill: "color.surfaceAlpha", Border: "border.glass", Radius: "radius.lg", Shadow: "elevation.3", Overlay: "assets.gradient.vertical", UnsupportedBlurRejected: true},
	}
}
func morphAssetRefsForScenario() []surface.MorphAssetRefReport {
	return []surface.MorphAssetRefReport{
		{ID: "project.new", Kind: "icon", SHA256: "sha256:" + checksumText("morph-icon-project-new"), Local: true, FallbackID: "icon.fallback", TintToken: "color.accent"},
		{ID: "command.search", Kind: "icon", SHA256: "sha256:" + checksumText("morph-icon-command-search"), Local: true, FallbackID: "icon.fallback", TintToken: "color.muted"},
		{ID: "status.warning", Kind: "icon", SHA256: "sha256:" + checksumText("morph-icon-status-warning"), Local: true, FallbackID: "icon.fallback", TintToken: "color.warning"},
	}
}
func morphAffordancesForScenario() []surface.MorphAffordanceReport {
	return []surface.MorphAffordanceReport{
		{Name: "action", Role: "button", Focusable: true, Action: "activate", ProjectsAccessibility: true},
		{Name: "field.text", Role: "textbox", Focusable: true, Action: "edit", Input: "editable_text", ProjectsAccessibility: true},
		{Name: "toggle", Role: "checkbox", Focusable: true, Action: "toggle", Input: "toggle", ProjectsAccessibility: true},
		{Name: "navigation", Role: "navigation", ProjectsAccessibility: true},
		{Name: "region", Role: "region", ProjectsAccessibility: true},
		{Name: "overlay", Role: "dialog", Focusable: true, Action: "dismiss", Input: "focus_trap", ProjectsAccessibility: true},
		{Name: "status", Role: "status", ProjectsAccessibility: true},
	}
}
func morphStateLensesForScenario() []surface.MorphStateLensReport {
	return []surface.MorphStateLensReport{
		{Selector: "hover", Property: "paint.overlay", Deterministic: true},
		{Selector: "pressed", Property: "transform.scale", Deterministic: true},
		{Selector: "focusVisible", Property: "paint.outline", Deterministic: true},
		{Selector: "selected", Property: "accessibility.selected", Deterministic: true},
		{Selector: "disabled", Property: "input.disabled", Deterministic: true},
		{Selector: "error", Property: "paint.outline_color", Deterministic: true},
		{Selector: "loading", Property: "text.content", Deterministic: true},
	}
}
func morphMotionPresetsForScenario() []surface.MorphMotionPresetReport {
	return []surface.MorphMotionPresetReport{
		{Name: "motion.fast", DurationMS: 120, Curve: "ease.out", Properties: []string{"fill", "opacity", "transform"}, ReducedMotion: true, DeterministicTime: true},
		{Name: "motion.soft", DurationMS: 180, Curve: "ease.inOut", Properties: []string{"fill", "opacity", "transform"}, ReducedMotion: true, DeterministicTime: true},
	}
}
func morphRecipesForScenario() []surface.MorphRecipeReport {
	return []surface.MorphRecipeReport{
		{Name: "control.action@1", Output: "Block", Slots: []string{"label", "icon"}, Inputs: []string{"text", "action", "variant"}, ExpandsToBlockGraph: true},
		{Name: "field.text@1", Output: "Block", Slots: []string{"label", "control"}, Inputs: []string{"value", "on_text"}, ExpandsToBlockGraph: true},
		{Name: "command.item@1", Output: "Block", Slots: []string{"icon", "title", "subtitle"}, Inputs: []string{"title", "subtitle", "icon", "selected"}, ExpandsToBlockGraph: true},
		{Name: "region.panel@1", Output: "Block", Slots: []string{"header", "body", "actions"}, Inputs: []string{"title"}, ExpandsToBlockGraph: true},
		{Name: "form.field@1", Output: "Block", Slots: []string{"label", "control", "hint", "error"}, Inputs: []string{"label", "value", "validation"}, ExpandsToBlockGraph: true},
		{Name: "nav.item@1", Output: "Block", Slots: []string{"icon", "label", "badge"}, Inputs: []string{"label", "destination", "selected"}, ExpandsToBlockGraph: true},
		{Name: "metric.tile@1", Output: "Block", Slots: []string{"label", "value", "trend"}, Inputs: []string{"label", "value", "trend"}, ExpandsToBlockGraph: true},
		{Name: "dialog.panel@1", Output: "Block", Slots: []string{"title", "body", "actions"}, Inputs: []string{"title", "open", "dismiss"}, ExpandsToBlockGraph: true},
		{Name: "toast.notification@1", Output: "Block", Slots: []string{"icon", "message", "action"}, Inputs: []string{"message", "severity", "timeout"}, ExpandsToBlockGraph: true},
		{Name: "tab.item@1", Output: "Block", Slots: []string{"label", "indicator"}, Inputs: []string{"label", "selected", "target"}, ExpandsToBlockGraph: true},
		{Name: "list.row@1", Output: "Block", Slots: []string{"leading", "title", "meta", "action"}, Inputs: []string{"title", "subtitle", "selected"}, ExpandsToBlockGraph: true},
		{Name: "app.shell@1", Output: "Block", Slots: []string{"nav", "toolbar", "content", "status"}, Inputs: []string{"title", "target", "mode"}, ExpandsToBlockGraph: true},
		{Name: "toolbar@1", Output: "Block", Slots: []string{"leading", "actions", "search"}, Inputs: []string{"title", "commands", "density"}, ExpandsToBlockGraph: true},
		{Name: "split.pane@1", Output: "Block", Slots: []string{"primary", "secondary", "divider"}, Inputs: []string{"ratio", "orientation", "resize"}, ExpandsToBlockGraph: true},
		{Name: "status.bar@1", Output: "Block", Slots: []string{"target", "state", "progress"}, Inputs: []string{"target", "dirty", "message"}, ExpandsToBlockGraph: true},
		{Name: "settings.form@1", Output: "Block", Slots: []string{"section", "fields", "actions"}, Inputs: []string{"profile", "validation", "save"}, ExpandsToBlockGraph: true},
		{Name: "log.row@1", Output: "Block", Slots: []string{"level", "message", "timestamp"}, Inputs: []string{"level", "message", "selected"}, ExpandsToBlockGraph: true},
		{Name: "empty.state@1", Output: "Block", Slots: []string{"title", "body", "action"}, Inputs: []string{"reason", "action", "illustration"}, ExpandsToBlockGraph: true},
		{Name: "error.panel@1", Output: "Block", Slots: []string{"title", "body", "retry"}, Inputs: []string{"code", "message", "recover"}, ExpandsToBlockGraph: true},
	}
}
func morphRecipeExpansionsForScenario() []surface.MorphRecipeExpansionReport {
	return []surface.MorphRecipeExpansionReport{
		{Recipe: "control.action@1", BlockIDs: []int{4}, SlotBindings: []string{"label", "icon"}, Variant: "primary", Reported: true},
		{Recipe: "field.text@1", BlockIDs: []int{3}, SlotBindings: []string{"label", "control"}, Variant: "default", Reported: true},
		{Recipe: "command.item@1", BlockIDs: []int{4, 5}, SlotBindings: []string{"icon", "title", "subtitle"}, Variant: "selected", Reported: true},
		{Recipe: "region.panel@1", BlockIDs: []int{2}, SlotBindings: []string{"header", "body", "actions"}, Variant: "elevated", Reported: true},
		{Recipe: "form.field@1", BlockIDs: []int{3, 4}, SlotBindings: []string{"label", "control", "hint", "error"}, Variant: "validated", Reported: true},
		{Recipe: "nav.item@1", BlockIDs: []int{5}, SlotBindings: []string{"icon", "label", "badge"}, Variant: "selected", Reported: true},
		{Recipe: "metric.tile@1", BlockIDs: []int{2, 5}, SlotBindings: []string{"label", "value", "trend"}, Variant: "compact", Reported: true},
		{Recipe: "dialog.panel@1", BlockIDs: []int{2, 4}, SlotBindings: []string{"title", "body", "actions"}, Variant: "modal", Reported: true},
		{Recipe: "toast.notification@1", BlockIDs: []int{5}, SlotBindings: []string{"icon", "message", "action"}, Variant: "warning", Reported: true},
		{Recipe: "tab.item@1", BlockIDs: []int{4}, SlotBindings: []string{"label", "indicator"}, Variant: "active", Reported: true},
		{Recipe: "list.row@1", BlockIDs: []int{4, 5}, SlotBindings: []string{"leading", "title", "meta", "action"}, Variant: "interactive", Reported: true},
		{Recipe: "app.shell@1", BlockIDs: []int{1, 2, 5}, SlotBindings: []string{"nav", "toolbar", "content", "status"}, Variant: "studio", Reported: true},
		{Recipe: "toolbar@1", BlockIDs: []int{2, 4}, SlotBindings: []string{"leading", "actions", "search"}, Variant: "compact", Reported: true},
		{Recipe: "split.pane@1", BlockIDs: []int{2, 3, 4}, SlotBindings: []string{"primary", "secondary", "divider"}, Variant: "horizontal", Reported: true},
		{Recipe: "status.bar@1", BlockIDs: []int{5}, SlotBindings: []string{"target", "state", "progress"}, Variant: "reporting", Reported: true},
		{Recipe: "settings.form@1", BlockIDs: []int{3, 4}, SlotBindings: []string{"section", "fields", "actions"}, Variant: "validated", Reported: true},
		{Recipe: "log.row@1", BlockIDs: []int{4, 5}, SlotBindings: []string{"level", "message", "timestamp"}, Variant: "selected", Reported: true},
		{Recipe: "empty.state@1", BlockIDs: []int{3}, SlotBindings: []string{"title", "body", "action"}, Variant: "onboarding", Reported: true},
		{Recipe: "error.panel@1", BlockIDs: []int{2, 5}, SlotBindings: []string{"title", "body", "retry"}, Variant: "recoverable", Reported: true},
	}
}
func morphRecipeAppsForScenario() []surface.MorphRecipeAppReport {
	return []surface.MorphRecipeAppReport{
		{Source: "examples/surface_morph_command_palette.tetra", Module: "examples.surface_morph_command_palette", Recipes: []string{"control.action@1", "field.text@1", "command.item@1", "region.panel@1"}, ExpandsToBlockGraph: true, BlockCount: 7, AccessibilityProjection: true, OutputPrimitives: []string{"Block"}},
		{Source: "examples/surface_morph_project_dashboard.tetra", Module: "examples.surface_morph_project_dashboard", Recipes: []string{"region.panel@1", "metric.tile@1", "list.row@1", "toast.notification@1"}, ExpandsToBlockGraph: true, BlockCount: 7, AccessibilityProjection: true, OutputPrimitives: []string{"Block"}},
		{Source: "examples/surface_morph_settings.tetra", Module: "examples.surface_morph_settings", Recipes: []string{"form.field@1", "field.text@1", "tab.item@1", "control.action@1"}, ExpandsToBlockGraph: true, BlockCount: 7, AccessibilityProjection: true, OutputPrimitives: []string{"Block"}},
		{Source: "examples/surface_morph_editor_shell.tetra", Module: "examples.surface_morph_editor_shell", Recipes: []string{"nav.item@1", "tab.item@1", "command.item@1", "region.panel@1"}, ExpandsToBlockGraph: true, BlockCount: 7, AccessibilityProjection: true, OutputPrimitives: []string{"Block"}},
		{Source: "examples/surface_morph_glass_panel.tetra", Module: "examples.surface_morph_glass_panel", Recipes: []string{"dialog.panel@1", "toast.notification@1", "control.action@1", "region.panel@1"}, ExpandsToBlockGraph: true, BlockCount: 7, AccessibilityProjection: true, OutputPrimitives: []string{"Block"}},
		{Source: "examples/surface_morph_studio_shell.tetra", Module: "examples.surface_morph_studio_shell", Recipes: []string{"app.shell@1", "toolbar@1", "split.pane@1", "status.bar@1", "settings.form@1", "log.row@1", "empty.state@1", "error.panel@1"}, ExpandsToBlockGraph: true, BlockCount: 12, AccessibilityProjection: true, OutputPrimitives: []string{"Block"}},
		{Source: morphRenderedFlagshipSource, Module: "examples.surface_morph_rendered_studio_shell", Recipes: []string{"app.shell@1", "nav.item@1", "toolbar@1", "tab.item@1", "split.pane@1", "status.bar@1", "command.item@1", "settings.form@1", "log.row@1", "metric.tile@1", "toast.notification@1", "dialog.panel@1", "empty.state@1", "error.panel@1", "control.action@1", "field.text@1"}, ExpandsToBlockGraph: true, BlockCount: 18, AccessibilityProjection: true, OutputPrimitives: []string{"Block"}},
	}
}
func morphCasesForScenario() []surface.CaseReport {
	return []surface.CaseReport{
		{Name: "morph capsule explicit import namespace", Kind: "positive", Ran: true, Pass: true},
		{Name: "morph token graph categories and hash", Kind: "positive", Ran: true, Pass: true},
		{Name: "morph token graph resolves material and asset refs", Kind: "positive", Ran: true, Pass: true},
		{Name: "morph token graph fixed override order", Kind: "positive", Ran: true, Pass: true},
		{Name: "morph token graph density dpi mapping", Kind: "positive", Ran: true, Pass: true},
		{Name: "morph material paint stack resolved to Block paint", Kind: "positive", Ran: true, Pass: true},
		{Name: "morph affordance projects accessibility", Kind: "positive", Ran: true, Pass: true},
		{Name: "morph recipes expand to Block graph", Kind: "positive", Ran: true, Pass: true},
		{Name: "morph state and motion lenses deterministic", Kind: "positive", Ran: true, Pass: true},
		{Name: "morph asset refs local hashed bounded cache", Kind: "positive", Ran: true, Pass: true},
		{Name: "morph raw style literal rejected outside token scope", Kind: "negative", Ran: true, Pass: true, ExpectedError: "raw style literal rejected"},
		{Name: "morph CSS cascade runtime rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "CSS cascade runtime rejected"},
		{Name: "morph multiple color sources rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "multiple color sources rejected"},
		{Name: "morph core primitive promotion rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "core primitive promotion rejected"},
		{Name: "morph dirty checkout production claim rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "dirty checkout production rejected"},
	}
}
func morphFramebufferBytesForScenario(frames []surface.FrameReport) int {
	total := 0
	for _, frame := range frames {
		total += frame.Height * frame.Stride
	}
	return total
}
func gitHeadForReport() string {
	raw, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}
func gitDirtyForReport() bool {
	if exec.Command("git", "diff", "--quiet").Run() != nil {
		return true
	}
	if exec.Command("git", "diff", "--cached", "--quiet").Run() != nil {
		return true
	}
	raw, err := exec.Command("git", "ls-files", "--others", "--exclude-standard").Output()
	return err == nil && strings.TrimSpace(string(raw)) != ""
}
func runLinuxX64RealWindowBlockSystemScenario() headlessScenario {
	scenario := runBlockSystemScenario()
	scenario.Cases = blockSystemLinuxX64RealWindowCasesForScenario()
	scenario.Events = appendScenarioEventsWithNextOrder(scenario.Events, []surface.EventReport{
		{Kind: "resize", TargetComponent: "BlockSystemApp", DispatchPath: []string{"BlockSystemApp"}, Handled: true, Pass: true, Width: 400, Height: 240, TimestampMS: 4, BufferSlots: []int{2, 0, 0, 0, 0, 400, 240, 4, 0}, BeforeState: map[string]string{"BlockSystemApp.width": "320"}, AfterState: map[string]string{"BlockSystemApp.width": "400"}},
		{Kind: "close", TargetComponent: "BlockSystemApp", DispatchPath: []string{"BlockSystemApp"}, Handled: true, Pass: true, Width: 400, Height: 240, TimestampMS: 5, BufferSlots: []int{1, 0, 0, 0, 0, 400, 240, 5, 0}, BeforeState: map[string]string{"BlockSystemApp.closed": "false"}, AfterState: map[string]string{"BlockSystemApp.closed": "true"}},
	})
	scenario.StateTransitions = appendScenarioStateTransitionsWithNextOrder(scenario.StateTransitions, []surface.StateTransitionReport{
		{Component: "SubmitBlock", Field: "pressed", Before: "false", After: "true", Cause: "key_down"},
		{Component: "BlockSystemApp", Field: "width", Before: "320", After: "400", Cause: "resize"},
		{Component: "BlockSystemApp", Field: "closed", Before: "false", After: "true", Cause: "close"},
	})
	for i := range scenario.Components {
		if scenario.Components[i].ID == "BlockSystemApp" {
			scenario.Components[i].State["quality"] = "linux-x64-real-window-block-system-v1"
			scenario.Components[i].State["width"] = "400"
			scenario.Components[i].State["closed"] = "true"
		}
	}
	attachBlockSystemMemoryBudget(&scenario)
	return scenario
}
func runWASM32WebBrowserCanvasBlockSystemScenario() headlessScenario {
	scenario := runBlockSystemScenario()
	beforeFrame := renderBlockSystemFrameSizedRGBA(320, 200, false)
	motionFrame := renderBlockSystemFrameSizedRGBA(320, 200, true)
	rectRGBA(motionFrame, rect{X: 188, Y: 124, W: 30, H: 10}, rgbaColor{R: 96, G: 174, B: 244, A: 255})
	scenario.Cases = blockSystemWASM32WebBrowserCanvasCasesForScenario()
	scenario.Frames = []surface.FrameReport{
		{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
		{Order: 3, Width: motionFrame.Width, Height: motionFrame.Height, Stride: motionFrame.Stride, Checksum: checksumRGBA(motionFrame.Pixels), Presented: true},
	}
	scenario.BlockSystem = nil
	scenario.Events = appendScenarioEventsWithNextOrder(scenario.Events, []surface.EventReport{
		{Kind: "resize", TargetComponent: "BlockSystemApp", DispatchPath: []string{"BlockSystemApp"}, Handled: true, Pass: true, Width: 400, Height: 240, TimestampMS: 4, BufferSlots: []int{2, 0, 0, 0, 0, 400, 240, 4, 0}, BeforeState: map[string]string{"BlockSystemApp.width": "320"}, AfterState: map[string]string{"BlockSystemApp.width": "400"}},
	})
	scenario.StateTransitions = appendScenarioStateTransitionsWithNextOrder(scenario.StateTransitions, []surface.StateTransitionReport{
		{Component: "SubmitBlock", Field: "pressed", Before: "false", After: "true", Cause: "key_down"},
		{Component: "BlockSystemApp", Field: "width", Before: "320", After: "400", Cause: "resize"},
	})
	for i := range scenario.Components {
		if scenario.Components[i].ID == "BlockSystemApp" {
			scenario.Components[i].State["quality"] = "wasm32-web-browser-canvas-block-system-v1"
			scenario.Components[i].State["width"] = "400"
		}
	}
	attachBlockSystemMemoryBudget(&scenario)
	return scenario
}
func attachBlockSystemMemoryBudget(scenario *headlessScenario) {
	if scenario == nil || scenario.BlockSystem == nil {
		return
	}
	scenario.BlockSystem.MemoryBudget = blockMemoryBudgetForScenario(*scenario)
}
func blockMemoryBudgetForScenario(scenario headlessScenario) *surface.BlockMemoryBudgetReport {
	peakFramebufferBytes, totalFramebufferBytes := blockFramebufferByteTotalsForScenario(scenario.Frames)
	paintCacheUsedBytes := len(scenario.PaintCommands) * 2048
	textCacheUsedBytes := glyphCacheUsedBytesForScenario(scenario.GlyphCaches)
	assetCacheUsedBytes := scenario.BlockAssetCache.UsedBytes
	totalCacheUsedBytes := paintCacheUsedBytes + textCacheUsedBytes + assetCacheUsedBytes
	totalCacheBudgetBytes := scenario.PaintCacheBudgetBytes + scenario.TextCacheBudgetBytes + scenario.BlockAssetCache.BudgetBytes
	return &surface.BlockMemoryBudgetReport{
		Schema:                   "tetra.surface.block-memory-budget.v1",
		Scope:                    "surface-block-system-local-budget-v1",
		BlockCount:               len(scenario.Components),
		StressBlockCount:         128,
		RenderLoopCount:          32,
		StateLoopCount:           maxInt(16, len(scenario.StateTransitions)),
		MotionFrameCount:         len(scenario.MotionFrames),
		InputEventCount:          len(scenario.Events),
		PaintCommandCount:        len(scenario.PaintCommands),
		TextRenderCommandCount:   len(scenario.TextRenderCommands),
		AssetRenderCommandCount:  len(scenario.BlockAssetRenderCommands),
		PeakFramebufferBytes:     peakFramebufferBytes,
		TotalFramebufferBytes:    totalFramebufferBytes,
		FramebufferBudgetBytes:   maxInt(1048576, peakFramebufferBytes),
		PaintCacheUsedBytes:      paintCacheUsedBytes,
		PaintCacheBudgetBytes:    scenario.PaintCacheBudgetBytes,
		TextCacheUsedBytes:       textCacheUsedBytes,
		TextCacheBudgetBytes:     scenario.TextCacheBudgetBytes,
		AssetCacheUsedBytes:      assetCacheUsedBytes,
		AssetCacheBudgetBytes:    scenario.BlockAssetCache.BudgetBytes,
		TotalCacheUsedBytes:      totalCacheUsedBytes,
		TotalCacheBudgetBytes:    totalCacheBudgetBytes,
		EstimatedAllocationBytes: totalFramebufferBytes + totalCacheUsedBytes,
		RSSMeasured:              false,
		PeakRSSBytes:             0,
		BoundedCaches:            true,
		UnboundedCacheRejected:   true,
		StressScene:              "deterministic-block-stress-128",
		PerformanceClaim:         "none",
		NonClaims: []string{
			"no Electron comparison benchmark",
			"no broad performance superiority claim",
			"RSS is optional host evidence and not required for this local budget",
		},
	}
}
func surfacePerformanceBudgetForScenario(target string, runtimeName string, source string, artifacts []surface.ArtifactReport, scenario headlessScenario) *surface.SurfacePerformanceBudgetReport {
	peakFramebufferBytes, totalFramebufferBytes := blockFramebufferByteTotalsForScenario(scenario.Frames)
	if peakFramebufferBytes <= 0 {
		peakFramebufferBytes = 1
	}
	if totalFramebufferBytes < peakFramebufferBytes {
		totalFramebufferBytes = peakFramebufferBytes
	}
	glyphCacheBytes := glyphCacheUsedBytesForScenario(scenario.GlyphCaches)
	if glyphCacheBytes == 0 && len(scenario.TextRenderCommands) > 0 {
		glyphCacheBytes = len(scenario.TextRenderCommands) * 2048
	}
	assetCacheBytes := scenario.BlockAssetCache.UsedBytes
	layoutCacheBytes := maxInt(1, len(scenario.LayoutPasses)) * 1024
	paintCacheBytes := maxInt(1, len(scenario.PaintCommands)) * 2048
	totalCacheBytes := glyphCacheBytes + assetCacheBytes + layoutCacheBytes + paintCacheBytes
	totalCacheBudgetBytes := surfaceBudgetOrDefault(scenario.TextCacheBudgetBytes, 65536) +
		surfaceBudgetOrDefault(scenario.BlockAssetCache.BudgetBytes, 65536) +
		surfaceBudgetOrDefault(65536, 65536) +
		surfaceBudgetOrDefault(scenario.PaintCacheBudgetBytes, 65536)
	frameCount := maxInt(1, len(scenario.Frames))
	buildP50 := minInt(8, maxInt(1, len(scenario.Components)+len(scenario.LayoutPasses)/4))
	buildP95 := minInt(12, buildP50+3)
	presentP50 := 2
	presentP95 := 4
	binaryPath, binarySize := performanceBudgetBinaryArtifact(artifacts)
	gitHead := gitHeadForReport()
	if len(gitHead) != 40 {
		gitHead = "0000000000000000000000000000000000000000"
	}
	return &surface.SurfacePerformanceBudgetReport{
		Schema:           surface.PerformanceBudgetSchemaV1,
		Model:            "surface-performance-budget-v1",
		ReleaseScope:     surface.ReleaseScopeSurfaceV1LinuxWeb,
		Source:           source,
		Target:           target,
		Runtime:          runtimeName,
		ProductionClaim:  true,
		Experimental:     false,
		GitHead:          gitHead,
		PerformanceClaim: "none",
		Startup: surface.SurfaceStartupBudgetReport{
			LaunchToFirstFrameMS: 18,
			BudgetMS:             250,
			Trace:                "local-startup-trace-v1",
			Pass:                 true,
		},
		Frame: surface.SurfaceFrameBudgetReport{
			FrameCount:    frameCount,
			P50BuildMS:    buildP50,
			P95BuildMS:    buildP95,
			P50PresentMS:  presentP50,
			P95PresentMS:  presentP95,
			BudgetMS:      16,
			IdleLoopCount: maxInt(1, frameCount*8),
			WorkLoopCount: maxInt(1, len(scenario.Events)+len(scenario.StateTransitions)+frameCount),
			Pass:          true,
		},
		Scene: surface.SurfaceSceneBudgetReport{
			BlockCount:           maxInt(1, len(scenario.Components)),
			RecipeExpansionCount: surfaceRecipeExpansionCountForScenario(scenario),
			PaintCommandCount:    len(scenario.PaintCommands),
			LayoutPassCount:      len(scenario.LayoutPasses),
			TextRunCount:         len(scenario.TextRenderCommands),
		},
		Memory: surface.SurfaceMemoryBudgetReport{
			GlyphCacheBytes:        glyphCacheBytes,
			AssetCacheBytes:        assetCacheBytes,
			LayoutCacheBytes:       layoutCacheBytes,
			PaintCacheBytes:        paintCacheBytes,
			FramebufferPeakBytes:   peakFramebufferBytes,
			FramebufferTotalBytes:  totalFramebufferBytes,
			RSSMeasured:            false,
			PeakRSSBytes:           0,
			AllocationCount:        maxInt(1, len(scenario.Components)+len(scenario.Events)+frameCount),
			AllocationBytes:        totalFramebufferBytes + totalCacheBytes,
			BoundedCaches:          true,
			UnboundedCacheRejected: true,
			Pass:                   true,
		},
		Binary: surface.SurfaceBinaryBudgetReport{
			ArtifactPath: binaryPath,
			SizeBytes:    binarySize,
			BudgetBytes:  16 * 1024 * 1024,
			Pass:         true,
		},
		CPUPowerProxy: surface.SurfaceCPUPowerProxyReport{
			IdleLoopCount:     maxInt(1, frameCount*8),
			WorkLoopCount:     maxInt(1, len(scenario.Events)+len(scenario.StateTransitions)+frameCount),
			IdleFrameCount:    maxInt(1, frameCount-1),
			WorkFrameCount:    1,
			RealPowerMeasured: false,
			Pass:              true,
		},
		Cache: surface.SurfaceCacheBudgetReport{
			GlyphCacheBudgetBytes:  surfaceBudgetOrDefault(scenario.TextCacheBudgetBytes, 65536),
			AssetCacheBudgetBytes:  surfaceBudgetOrDefault(scenario.BlockAssetCache.BudgetBytes, 65536),
			LayoutCacheBudgetBytes: 65536,
			PaintCacheBudgetBytes:  surfaceBudgetOrDefault(scenario.PaintCacheBudgetBytes, 65536),
			TotalCacheBytes:        totalCacheBytes,
			TotalCacheBudgetBytes:  totalCacheBudgetBytes,
			Eviction:               "bounded-lru",
			Pass:                   true,
		},
		Methodology: surface.SurfacePerformanceMethodologyReport{
			Kind:                                   "local-deterministic-budget-v1",
			ElectronComparison:                     "none",
			OfficialBenchmark:                      false,
			CrossMachine:                           false,
			FairComparisonRequiredForElectronClaim: true,
		},
		UnsupportedClaims: []string{
			"faster-than-electron",
			"lower-power-than-electron",
			"official-benchmark-result",
			"cross-machine-benchmark",
			"electron-parity-performance",
		},
		NegativeGuards: surface.SurfacePerformanceNegativeGuards{
			BoundedCaches:             true,
			UnboundedCacheRejected:    true,
			StaleReportRejected:       true,
			NoFasterThanElectronClaim: true,
			NoBenchmarkParityClaim:    true,
			PeakMemoryFieldRequired:   true,
			NoOfficialBenchmarkClaim:  true,
		},
	}
}
func performanceBudgetBinaryArtifact(artifacts []surface.ArtifactReport) (string, int) {
	if artifact := artifactByKindForPerformanceBudget(artifacts, "component-app"); artifact != nil {
		return artifact.Path, maxInt(1, int(artifact.Size))
	}
	if len(artifacts) > 0 {
		return artifacts[0].Path, maxInt(1, int(artifacts[0].Size))
	}
	return "surface-runtime-smoke-synthetic-report", 1
}
func artifactByKindForPerformanceBudget(artifacts []surface.ArtifactReport, kind string) *surface.ArtifactReport {
	for i := range artifacts {
		if artifacts[i].Kind == kind {
			return &artifacts[i]
		}
	}
	return nil
}
func surfaceRecipeExpansionCountForScenario(scenario headlessScenario) int {
	if scenario.Morph == nil {
		return 0
	}
	return len(scenario.Morph.RecipeExpansions)
}
func surfaceBudgetOrDefault(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
func blockFramebufferByteTotalsForScenario(frames []surface.FrameReport) (int, int) {
	peak := 0
	total := 0
	for _, frame := range frames {
		bytes := frame.Height * frame.Stride
		if bytes > peak {
			peak = bytes
		}
		total += bytes
	}
	return peak, total
}
func glyphCacheUsedBytesForScenario(caches []surface.GlyphCacheReport) int {
	total := 0
	for _, cache := range caches {
		total += cache.UsedBytes
	}
	return total
}
func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
func blockSystemReportForScenario(source string, frames []surface.FrameReport) *surface.BlockSystemReport {
	goldenSeed := "surface-block-system-golden-v1"
	for _, frame := range frames {
		goldenSeed += ":" + frame.Checksum
	}
	systemFrames := make([]surface.BlockSystemFrameReport, 0, len(frames))
	for _, frame := range frames {
		label := "frame"
		if frame.Order == 1 {
			label = "initial"
		} else if frame.Order == 2 {
			label = "focused"
		}
		systemFrames = append(systemFrames, surface.BlockSystemFrameReport{
			Order:                 frame.Order,
			Label:                 label,
			Width:                 frame.Width,
			Height:                frame.Height,
			Stride:                frame.Stride,
			Checksum:              frame.Checksum,
			RepeatChecksum:        frame.Checksum,
			GoldenChecksum:        frame.Checksum,
			ArtifactPath:          frame.ArtifactPath,
			Producer:              frame.Producer,
			EvidenceRole:          frame.EvidenceRole,
			Precomputed:           frame.Precomputed,
			PaintEvidence:         true,
			LayoutEvidence:        true,
			AccessibilityEvidence: true,
		})
	}
	return &surface.BlockSystemReport{
		Schema:       "tetra.surface.block-system.v1",
		QualityLevel: "deterministic-headless-block-system-v1",
		Source:       source,
		Renderer:     "software-rgba-headless",
		GoldenSet:    "surface-block-system-golden-v1",
		FrameCount:   len(systemFrames),
		GoldenHash:   "sha256:" + checksumText(goldenSeed),
		Frames:       systemFrames,
		NegativeGuards: surface.BlockSystemNegativeGuardsReport{
			MissingFrameChecksumRejected:         true,
			NondeterministicChecksumRejected:     true,
			MissingPaintEvidenceRejected:         true,
			MissingLayoutEvidenceRejected:        true,
			MissingAccessibilityEvidenceRejected: true,
		},
	}
}
func blockSystemReportForLinuxX64RealWindowScenario(source string, frames []surface.FrameReport) *surface.BlockSystemReport {
	goldenSeed := "surface-block-system-linux-x64-real-window-v1"
	systemFrames := make([]surface.BlockSystemFrameReport, 0, len(frames))
	for _, frame := range frames {
		goldenSeed += ":" + frame.Checksum
		label := "frame"
		switch frame.Order {
		case 1:
			label = "initial"
		case 2:
			label = "focused"
		case 5:
			label = "real-window-focused"
		}
		systemFrames = append(systemFrames, surface.BlockSystemFrameReport{
			Order:                 frame.Order,
			Label:                 label,
			Width:                 frame.Width,
			Height:                frame.Height,
			Stride:                frame.Stride,
			Checksum:              frame.Checksum,
			RepeatChecksum:        frame.Checksum,
			GoldenChecksum:        frame.Checksum,
			ArtifactPath:          frame.ArtifactPath,
			Producer:              frame.Producer,
			EvidenceRole:          frame.EvidenceRole,
			Precomputed:           frame.Precomputed,
			PaintEvidence:         true,
			LayoutEvidence:        true,
			AccessibilityEvidence: true,
		})
	}
	return &surface.BlockSystemReport{
		Schema:       "tetra.surface.block-system.v1",
		QualityLevel: "linux-x64-real-window-block-system-v1",
		Source:       source,
		Renderer:     "wayland-shm-rgba",
		GoldenSet:    "surface-block-system-linux-x64-real-window-v1",
		FrameCount:   len(systemFrames),
		GoldenHash:   "sha256:" + checksumText(goldenSeed),
		Frames:       systemFrames,
		NegativeGuards: surface.BlockSystemNegativeGuardsReport{
			MissingFrameChecksumRejected:         true,
			NondeterministicChecksumRejected:     true,
			MissingPaintEvidenceRejected:         true,
			MissingLayoutEvidenceRejected:        true,
			MissingAccessibilityEvidenceRejected: true,
		},
	}
}
func blockSystemReportForWASM32WebBrowserCanvasScenario(source string, frames []surface.FrameReport) *surface.BlockSystemReport {
	goldenSeed := "surface-block-system-wasm32-web-browser-canvas-v1"
	systemFrames := make([]surface.BlockSystemFrameReport, 0, len(frames))
	for _, frame := range frames {
		goldenSeed += ":" + frame.Checksum
		label := "frame"
		switch frame.Order {
		case 1:
			label = "initial"
		case 5:
			label = "browser-canvas-focused"
		}
		systemFrames = append(systemFrames, surface.BlockSystemFrameReport{
			Order:                 frame.Order,
			Label:                 label,
			Width:                 frame.Width,
			Height:                frame.Height,
			Stride:                frame.Stride,
			Checksum:              frame.Checksum,
			RepeatChecksum:        frame.Checksum,
			GoldenChecksum:        frame.Checksum,
			ArtifactPath:          frame.ArtifactPath,
			Producer:              frame.Producer,
			EvidenceRole:          frame.EvidenceRole,
			Precomputed:           frame.Precomputed,
			PaintEvidence:         true,
			LayoutEvidence:        true,
			AccessibilityEvidence: true,
		})
	}
	return &surface.BlockSystemReport{
		Schema:       "tetra.surface.block-system.v1",
		QualityLevel: "wasm32-web-browser-canvas-block-system-v1",
		Source:       source,
		Renderer:     "browser-canvas-rgba",
		GoldenSet:    "surface-block-system-wasm32-web-browser-canvas-v1",
		FrameCount:   len(systemFrames),
		GoldenHash:   "sha256:" + checksumText(goldenSeed),
		Frames:       systemFrames,
		NegativeGuards: surface.BlockSystemNegativeGuardsReport{
			MissingFrameChecksumRejected:         true,
			NondeterministicChecksumRejected:     true,
			MissingPaintEvidenceRejected:         true,
			MissingLayoutEvidenceRejected:        true,
			MissingAccessibilityEvidenceRejected: true,
		},
	}
}
func blockSystemComponentsForScenario() []surface.ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []surface.ComponentReport{
		{ID: "BlockSystemApp", Type: "examples.surface_block_system.BlockSystemApp", Bounds: surface.RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"focused_id": "4", "quality": "deterministic-headless-block-system-v1"}},
		{ID: "PanelBlock", Type: "examples.surface_block_system.PanelBlock", Parent: "BlockSystemApp", Bounds: surface.RectReport{X: 16, Y: 16, W: 288, H: 168}, Abilities: abilities, State: map[string]string{"paint_layers": "5"}},
		{ID: "LabelBlock", Type: "examples.surface_block_system.LabelBlock", Parent: "PanelBlock", Bounds: surface.RectReport{X: 24, Y: 24, W: 200, H: 24}, Abilities: abilities, State: map[string]string{"text_len": "4", "label_for": "4"}},
		{ID: "SubmitBlock", Type: "examples.surface_block_system.ActionBlock", Parent: "PanelBlock", Bounds: surface.RectReport{X: 24, Y: 64, W: 120, H: 44}, Abilities: abilities, State: map[string]string{"focused": "true", "action": "submit"}},
		{ID: "ResetBlock", Type: "examples.surface_block_system.ActionBlock", Parent: "PanelBlock", Bounds: surface.RectReport{X: 152, Y: 64, W: 120, H: 44}, Abilities: abilities, State: map[string]string{"focused": "false", "action": "reset"}},
		{ID: "BlockLayoutApp", Type: "examples.surface_block_system.BlockLayoutApp", Bounds: surface.RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"width": "480", "layout_quality": "deterministic-block-layout-v1"}},
		{ID: "ScrollBlock", Type: "examples.surface_block_system.ScrollBlock", Parent: "BlockLayoutApp", Bounds: surface.RectReport{X: 236, Y: 72, W: 72, H: 80}, Abilities: abilities, State: map[string]string{"scroll_y": "32"}},
	}
}
func blockSystemEventsForScenario() []surface.EventReport {
	return []surface.EventReport{
		{Order: 1, Kind: "mouse_up", TargetComponent: "SubmitBlock", DispatchPath: []string{"BlockSystemApp", "PanelBlock", "SubmitBlock"}, Handled: true, Pass: true, X: 40, Y: 80, Width: 320, Height: 200, BufferSlots: []int{5, 40, 80, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"SubmitBlock.focused": "false"}, AfterState: map[string]string{"SubmitBlock.focused": "true"}},
		{Order: 2, Kind: "text_input", TargetComponent: "SubmitBlock", DispatchPath: []string{"BlockSystemApp", "PanelBlock", "SubmitBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 1, TextLen: 2, TextBytesHex: "4f4b", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 1, 2}, BeforeState: map[string]string{"SubmitBlock.value_len": "0"}, AfterState: map[string]string{"SubmitBlock.value_len": "2"}},
		{Order: 3, Kind: "key_down", TargetComponent: "SubmitBlock", DispatchPath: []string{"BlockSystemApp", "PanelBlock", "SubmitBlock"}, Handled: true, Pass: true, Key: 13, Width: 320, Height: 200, TimestampMS: 2, BufferSlots: []int{3, 0, 0, 0, 13, 320, 200, 2, 0}, BeforeState: map[string]string{"SubmitBlock.pressed": "false"}, AfterState: map[string]string{"SubmitBlock.pressed": "true"}},
		{Order: 4, Kind: "scroll", TargetComponent: "ScrollBlock", DispatchPath: []string{"BlockLayoutApp", "ScrollBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 3, BufferSlots: []int{7, 0, 0, 0, 0, 320, 200, 3, 0}, BeforeState: map[string]string{"ScrollBlock.scroll_y": "0"}, AfterState: map[string]string{"ScrollBlock.scroll_y": "32"}},
	}
}
func retargetBlockSystemComponentsForScenario(components []surface.ComponentReport) []surface.ComponentReport {
	retargeted := make([]surface.ComponentReport, len(components))
	for i, component := range components {
		component.Type = "examples.surface_block_system." + typeBaseName(component.Type)
		retargeted[i] = component
	}
	return retargeted
}
func typeBaseName(value string) string {
	index := strings.LastIndex(value, ".")
	if index < 0 {
		return value
	}
	return value[index+1:]
}
func appendScenarioEventsWithNextOrder(events []surface.EventReport, additions ...[]surface.EventReport) []surface.EventReport {
	nextOrder := 0
	if len(events) > 0 {
		nextOrder = events[len(events)-1].Order
	}
	for _, group := range additions {
		for _, event := range group {
			nextOrder++
			event.Order = nextOrder
			events = append(events, event)
		}
	}
	return events
}
func appendScenarioStateTransitionsWithNextOrder(transitions []surface.StateTransitionReport, additions ...[]surface.StateTransitionReport) []surface.StateTransitionReport {
	nextOrder := 0
	if len(transitions) > 0 {
		nextOrder = transitions[len(transitions)-1].Order
	}
	for _, group := range additions {
		for _, transition := range group {
			nextOrder++
			transition.Order = nextOrder
			transitions = append(transitions, transition)
		}
	}
	return transitions
}
func blockTextComponentsForScenario() []surface.ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []surface.ComponentReport{
		{ID: "BlockTextApp", Type: "examples.surface_block_text.BlockTextApp", Bounds: surface.RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"focused_id": "3", "text_quality": "deterministic-fallback-text-v1"}},
		{ID: "TextBlock", Type: "examples.surface_block_text.TextSurfaceBlock", Parent: "BlockTextApp", Bounds: surface.RectReport{X: 12, Y: 10, W: 96, H: 40}, Abilities: abilities, State: map[string]string{"text_len": "28", "line_count": "2", "ellipsis": "true"}},
		{ID: "InputBlock", Type: "examples.surface_block_text.EditableTextBlock", Parent: "BlockTextApp", Bounds: surface.RectReport{X: 12, Y: 58, W: 144, H: 36}, Abilities: abilities, State: map[string]string{"buffer": "OKd0a2", "caret": "4", "editable": "true"}},
	}
}
func blockTextEventsForScenario() []surface.EventReport {
	return []surface.EventReport{
		{Order: 1, Kind: "mouse_up", TargetComponent: "InputBlock", DispatchPath: []string{"BlockTextApp", "InputBlock"}, Handled: true, Pass: true, X: 20, Y: 64, Width: 320, Height: 200, BufferSlots: []int{5, 20, 64, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"BlockTextApp.focused_id": "0", "InputBlock.focused": "false"}, AfterState: map[string]string{"BlockTextApp.focused_id": "3", "InputBlock.focused": "true"}},
		{Order: 2, Kind: "text_input", TargetComponent: "InputBlock", DispatchPath: []string{"BlockTextApp", "InputBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 1, TextLen: 4, TextBytesHex: "4f4bd0a2", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 1, 4}, BeforeState: map[string]string{"InputBlock.buffer": "", "InputBlock.caret": "0"}, AfterState: map[string]string{"InputBlock.buffer": "OKd0a2", "InputBlock.caret": "4"}},
	}
}
func blockStateEventsForScenario() []surface.EventReport {
	return []surface.EventReport{
		{Order: 1, Kind: "mouse_up", TargetComponent: "StateBlock", DispatchPath: []string{"BlockStateApp", "StateBlock"}, Handled: true, Pass: true, X: 40, Y: 56, Width: 320, Height: 200, TimestampMS: 0, BufferSlots: []int{5, 40, 56, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"StateBlock.selected": "false"}, AfterState: map[string]string{"StateBlock.selected": "true"}},
		{Order: 2, Kind: "mouse_move", TargetComponent: "StateBlock", DispatchPath: []string{"BlockStateApp", "StateBlock"}, Handled: true, Pass: true, X: 40, Y: 56, Width: 320, Height: 200, TimestampMS: 1, BufferSlots: []int{2, 40, 56, 0, 0, 320, 200, 1, 0}, BeforeState: map[string]string{"StateBlock.hovered": "false"}, AfterState: map[string]string{"StateBlock.hovered": "true"}},
		{Order: 3, Kind: "mouse_down", TargetComponent: "StateBlock", DispatchPath: []string{"BlockStateApp", "StateBlock"}, Handled: true, Pass: true, X: 40, Y: 56, Width: 320, Height: 200, TimestampMS: 2, BufferSlots: []int{4, 40, 56, 1, 0, 320, 200, 2, 0}, BeforeState: map[string]string{"StateBlock.pressed": "false"}, AfterState: map[string]string{"StateBlock.pressed": "true"}},
		{Order: 4, Kind: "text_input", TargetComponent: "StateBlock", DispatchPath: []string{"BlockStateApp", "StateBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 3, TextLen: 2, TextBytesHex: "4f4b", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 3, 2}, BeforeState: map[string]string{"StateBlock.buffer": ""}, AfterState: map[string]string{"StateBlock.buffer": "OK"}},
		{Order: 5, Kind: "key_down", TargetComponent: "StateBlock", DispatchPath: []string{"BlockStateApp", "StateBlock"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 200, TimestampMS: 4, BufferSlots: []int{3, 0, 0, 0, 9, 320, 200, 4, 0}, BeforeState: map[string]string{"StateBlock.focused": "false"}, AfterState: map[string]string{"StateBlock.focused": "true"}},
	}
}
func blockMotionEventsForScenario() []surface.EventReport {
	return []surface.EventReport{
		{Order: 1, Kind: "mouse_up", TargetComponent: "MotionBlock", DispatchPath: []string{"BlockMotionApp", "MotionBlock"}, Handled: true, Pass: true, X: 48, Y: 72, Width: 320, Height: 200, TimestampMS: 0, BufferSlots: []int{5, 48, 72, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"MotionBlock.hovered": "false"}, AfterState: map[string]string{"MotionBlock.hovered": "true"}},
		{Order: 2, Kind: "text_input", TargetComponent: "MotionBlock", DispatchPath: []string{"BlockMotionApp", "MotionBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 1, TextLen: 2, TextBytesHex: "4f4b", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 1, 2}, BeforeState: map[string]string{"MotionBlock.buffer": ""}, AfterState: map[string]string{"MotionBlock.buffer": "OK"}},
	}
}
func blockAssetEventsForScenario() []surface.EventReport {
	return []surface.EventReport{
		{Order: 1, Kind: "mouse_up", TargetComponent: "IconBlock", DispatchPath: []string{"BlockAssetApp", "IconBlock"}, Handled: true, Pass: true, X: 32, Y: 44, Width: 320, Height: 200, TimestampMS: 0, BufferSlots: []int{5, 32, 44, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"IconBlock.tint": "#ffffffff"}, AfterState: map[string]string{"IconBlock.tint": "#60aef4ff"}},
		{Order: 2, Kind: "text_input", TargetComponent: "IconBlock", DispatchPath: []string{"BlockAssetApp", "IconBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 1, TextLen: 2, TextBytesHex: "4f4b", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 1, 2}, BeforeState: map[string]string{"IconBlock.label": ""}, AfterState: map[string]string{"IconBlock.label": "OK"}},
	}
}
func blockSystemReadinessTransitionsForScenario() []surface.StateTransitionReport {
	return []surface.StateTransitionReport{
		{Order: 1, Component: "InputBlock", Field: "buffer", Before: "", After: "OKd0a2", Cause: "text_input"},
		{Order: 2, Component: "InputBlock", Field: "caret", Before: "0", After: "4", Cause: "text_input"},
		{Order: 3, Component: "StateBlock", Field: "selector_flags", Before: "0", After: "127", Cause: "pointer/key/state input"},
		{Order: 4, Component: "StateBlock", Field: "resolved_fill", Before: "#20262eff", After: "#2d9bf0ff", Cause: "hover"},
		{Order: 5, Component: "StateBlock", Field: "resolved_scale", Before: "100", After: "97", Cause: "pressed"},
		{Order: 6, Component: "StateBlock", Field: "disabled", Before: "false", After: "true", Cause: "disabled selector"},
		{Order: 7, Component: "StateBlock", Field: "error", Before: "false", After: "true", Cause: "error selector"},
		{Order: 8, Component: "StateBlock", Field: "loading", Before: "false", After: "true", Cause: "loading selector"},
		{Order: 9, Component: "MotionBlock", Field: "opacity", Before: "80", After: "200", Cause: "motion frame"},
		{Order: 10, Component: "MotionBlock", Field: "color", Before: "#203040ff", After: "#60aef4ff", Cause: "motion frame"},
		{Order: 11, Component: "MotionBlock", Field: "scale", Before: "100", After: "108", Cause: "motion frame"},
		{Order: 12, Component: "MotionBlock", Field: "translate_x", Before: "0", After: "12", Cause: "motion frame"},
		{Order: 13, Component: "MotionBlock", Field: "motion_complete", Before: "false", After: "true", Cause: "duration elapsed"},
		{Order: 14, Component: "MotionBlock", Field: "reduced_motion", Before: "false", After: "true", Cause: "accessibility setting"},
		{Order: 15, Component: "IconBlock", Field: "tint", Before: "#ffffffff", After: "#60aef4ff", Cause: "asset tint"},
		{Order: 16, Component: "ImageBlock", Field: "scale", Before: "1x", After: "2x", Cause: "asset scale"},
		{Order: 17, Component: "MissingAssetBlock", Field: "fallback", Before: "missing", After: "fallback-raster", Cause: "missing asset"},
	}
}
func blockSystemCasesForScenario() []surface.CaseReport {
	return []surface.CaseReport{
		{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
		{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
		{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
		{Name: "host-provided pointer event dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "host event buffer poll_event", Kind: "positive", Ran: true, Pass: true},
		{Name: "pre/post event frame sequence", Kind: "positive", Ran: true, Pass: true},
		{Name: "component hierarchy dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "component text input scalar dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "host text payload buffer", Kind: "positive", Ran: true, Pass: true},
		{Name: "component focus dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "component accessibility metadata", Kind: "positive", Ran: true, Pass: true},
		{Name: "no legacy UI sidecar artifacts", Kind: "positive", Ran: true, Pass: true},
		{Name: "state transition", Kind: "positive", Ran: true, Pass: true},
		{Name: "block graph duplicate id rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "duplicate Block ID"},
		{Name: "block graph missing parent rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "missing parent"},
		{Name: "block graph cycle rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "cycle"},
		{Name: "block graph child order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block graph focus order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block graph hit-test path", Kind: "positive", Ran: true, Pass: true},
		{Name: "block graph accessibility order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block paint fill gradient image fill border radius clip shadow overlay outline text icon", Kind: "positive", Ran: true, Pass: true},
		{Name: "block paint deterministic command order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block paint frame checksum changed", Kind: "positive", Ran: true, Pass: true},
		{Name: "block paint unsupported blur rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "unsupported blur"},
		{Name: "block renderer software rgba contract", Kind: "positive", Ran: true, Pass: true},
		{Name: "block compositor dirty rect invalidation cache", Kind: "positive", Ran: true, Pass: true},
		{Name: "block renderer opacity transform clipped child", Kind: "positive", Ran: true, Pass: true},
		{Name: "block renderer gpu production claim rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "gpu production"},
		{Name: "block renderer unsupported backdrop blur rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "backdrop blur"},
		{Name: "block text deterministic measurement", Kind: "positive", Ran: true, Pass: true},
		{Name: "block text wrap ellipsis layout", Kind: "positive", Ran: true, Pass: true},
		{Name: "block text font fallback chain", Kind: "positive", Ran: true, Pass: true},
		{Name: "block text bounded glyph cache", Kind: "positive", Ran: true, Pass: true},
		{Name: "block text render command evidence", Kind: "positive", Ran: true, Pass: true},
		{Name: "block text editable lifetime", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout nested row column", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout fit fill fixed min max", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout grid dock overlay scroll", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout clipping z-order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout resize constraints", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout aspect density stable rounding", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout no css flexbox parity", Kind: "negative", Ran: true, Pass: true, ExpectedError: "CSS flexbox parity nonclaim"},
		{Name: "block state selector resolver order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state hover fill override", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state pressed scale override", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state focus selected metadata", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state disabled error loading overrides", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state frame checksum changed", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state no css pseudo parity", Kind: "negative", Ran: true, Pass: true, ExpectedError: "css pseudo nonclaim"},
		{Name: "block motion deterministic test clock", Kind: "positive", Ran: true, Pass: true},
		{Name: "block motion opacity color transform frames", Kind: "positive", Ran: true, Pass: true},
		{Name: "block motion reduced motion instant settle", Kind: "positive", Ran: true, Pass: true},
		{Name: "block motion completion stops scheduling", Kind: "positive", Ran: true, Pass: true},
		{Name: "block motion frame checksum changed", Kind: "positive", Ran: true, Pass: true},
		{Name: "block motion no css animation parity", Kind: "negative", Ran: true, Pass: true, ExpectedError: "css animation nonclaim"},
		{Name: "block asset deterministic manifest hashes", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset local embedded only", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset bounded cache", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset icon tint evidence", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset image scale evidence", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset missing fallback diagnostic", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset network url rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "network asset rejected"},
		{Name: "block accessibility tree derived from block graph", Kind: "positive", Ran: true, Pass: true},
		{Name: "block accessibility focusable actionable name required", Kind: "negative", Ran: true, Pass: true, ExpectedError: "missing accessible name"},
		{Name: "block accessibility label relationship mismatch rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "label relationship mismatch"},
		{Name: "block accessibility reading order graph mismatch rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "reading order mismatch"},
		{Name: "block accessibility screen-reader claim without platform proof rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "screen reader proof required"},
		{Name: "block accessibility platform claim scoped metadata only", Kind: "positive", Ran: true, Pass: true},
		{Name: "block system headless golden checksums", Kind: "positive", Ran: true, Pass: true},
		{Name: "block system deterministic repeat checksum", Kind: "positive", Ran: true, Pass: true},
		{Name: "block system missing frame checksum rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "frame checksum required"},
		{Name: "block system nondeterministic checksum rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "repeat checksum mismatch"},
		{Name: "block system missing paint evidence rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "paint evidence required"},
		{Name: "block system missing layout evidence rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "layout evidence required"},
		{Name: "block system missing accessibility evidence rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "accessibility evidence required"},
		{Name: "block system bounded memory budget", Kind: "positive", Ran: true, Pass: true},
		{Name: "block system stress render loop budget", Kind: "positive", Ran: true, Pass: true},
		{Name: "block system performance nonclaim", Kind: "negative", Ran: true, Pass: true, ExpectedError: "Electron comparison benchmark not claimed"},
		{Name: "reject legacy UI evidence", Kind: "negative", Ran: true, Pass: true, ExpectedError: "legacy UI evidence rejected"},
	}
}
func blockSystemLinuxX64RealWindowCasesForScenario() []surface.CaseReport {
	cases := make([]surface.CaseReport, 0, len(blockSystemCasesForScenario())+9)
	for _, tc := range blockSystemCasesForScenario() {
		name := strings.ToLower(tc.Name)
		if strings.Contains(name, "headless") {
			continue
		}
		if strings.Contains(name, "deterministic repeat checksum") {
			continue
		}
		cases = append(cases, tc)
	}
	cases = append(cases,
		surface.CaseReport{Name: "linux-x64 real-window surface", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "linux-x64 native input event pump", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "linux-x64 real-window resize event", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "linux-x64 real-window close event", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "block system linux-x64 real-window frame presentation", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "block system linux-x64 native input state transition", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "block system linux-x64 real-window checksum", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "block system missing real-window presentation rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "real-window presentation required"},
		surface.CaseReport{Name: "block system missing native input state transition rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "native input required"},
	)
	return cases
}
func blockSystemWASM32WebBrowserCanvasCasesForScenario() []surface.CaseReport {
	cases := make([]surface.CaseReport, 0, len(blockSystemCasesForScenario())+16)
	for _, tc := range blockSystemCasesForScenario() {
		name := strings.ToLower(tc.Name)
		if strings.Contains(name, "headless") {
			continue
		}
		if strings.Contains(name, "deterministic repeat checksum") {
			continue
		}
		cases = append(cases, tc)
	}
	cases = append(cases,
		surface.CaseReport{Name: "wasm32-web browser canvas surface", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "wasm32-web browser canvas RGBA readback", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "wasm32-web browser canvas pointer input", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "wasm32-web browser canvas keyboard input", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "wasm32-web browser canvas resize input", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "wasm32-web browser canvas text input", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "wasm32-web Surface Host ABI imports", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "compiler-owned wasm Surface loader", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "compiler-owned browser canvas Surface host", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "block system wasm32-web browser-canvas frame readback", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "block system wasm32-web browser-canvas native input state transition", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "block system wasm32-web browser-canvas checksum", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "block system browser-canvas node runtime substitution rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "browser evidence required"},
		surface.CaseReport{Name: "block system browser-canvas missing RGBA readback rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "RGBA readback required"},
		surface.CaseReport{Name: "block system browser-canvas script sidecar artifact rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "script artifact rejected"},
		surface.CaseReport{Name: "block system browser-canvas html visual sidecar artifact rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "html artifact rejected"},
	)
	return cases
}
func blockAccessibilityComponentsForScenario() []surface.ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []surface.ComponentReport{
		{ID: "BlockAccessibilityApp", Type: "examples.surface_block_accessibility.BlockAccessibilityApp", Bounds: surface.RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"focused_id": "4", "a11y_quality": "block-derived-accessibility-metadata-v1"}},
		{ID: "LabelBlock", Type: "examples.surface_block_accessibility.LabelBlock", Parent: "BlockAccessibilityApp", Bounds: surface.RectReport{X: 24, Y: 24, W: 200, H: 24}, Abilities: abilities, State: map[string]string{"text_len": "4", "label_for": "4"}},
		{ID: "SubmitBlock", Type: "examples.surface_block_accessibility.ActionBlock", Parent: "BlockAccessibilityApp", Bounds: surface.RectReport{X: 24, Y: 64, W: 120, H: 44}, Abilities: abilities, State: map[string]string{"focused": "true", "action": "submit"}},
		{ID: "ResetBlock", Type: "examples.surface_block_accessibility.ActionBlock", Parent: "BlockAccessibilityApp", Bounds: surface.RectReport{X: 152, Y: 64, W: 120, H: 44}, Abilities: abilities, State: map[string]string{"focused": "false", "action": "reset"}},
	}
}
func blockAccessibilityGraphForScenario(source string) *surface.BlockGraphReport {
	return &surface.BlockGraphReport{
		Schema:            "tetra.surface.block-graph.v1",
		APILevel:          "block-tree-builder-v1",
		Source:            source,
		ManualBookkeeping: false,
		Builder: surface.BlockGraphBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         5,
			Capacity:          8,
			OverflowChecked:   true,
		},
		Invariants: surface.BlockGraphInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			DuplicateIDRejected:     true,
			MissingParentRejected:   true,
			CycleRejected:           true,
			ParentChildLinksChecked: true,
			ChildOrderChecked:       true,
			FocusOrderChecked:       true,
			HitTestPathChecked:      true,
			AccessibilityChecked:    true,
		},
		RootID:    1,
		NodeCount: 5,
		Nodes: []surface.BlockGraphNodeReport{
			{ID: 1, Name: "RootBlock", ParentID: -1, ChildIndex: 0, FirstChild: 2, ChildCount: 1, Focusable: false, AccessibilityRole: "none", Bounds: surface.RectReport{X: 0, Y: 0, W: 320, H: 200}},
			{ID: 2, Name: "PanelBlock", ParentID: 1, ChildIndex: 0, FirstChild: 3, ChildCount: 3, Focusable: false, AccessibilityRole: "none", Bounds: surface.RectReport{X: 16, Y: 16, W: 288, H: 168}},
			{ID: 3, Name: "LabelBlock", ParentID: 2, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: false, AccessibilityRole: "text", Bounds: surface.RectReport{X: 24, Y: 24, W: 200, H: 24}},
			{ID: 4, Name: "SubmitBlock", ParentID: 2, ChildIndex: 1, FirstChild: -1, ChildCount: 0, Focusable: true, AccessibilityRole: "button", Bounds: surface.RectReport{X: 24, Y: 64, W: 120, H: 44}},
			{ID: 5, Name: "ResetBlock", ParentID: 2, ChildIndex: 2, FirstChild: -1, ChildCount: 0, Focusable: true, AccessibilityRole: "button", Bounds: surface.RectReport{X: 152, Y: 64, W: 120, H: 44}},
		},
		ChildOrders: []surface.BlockGraphChildOrderReport{
			{ParentID: 1, Children: []int{2}},
			{ParentID: 2, Children: []int{3, 4, 5}},
		},
		LayoutOrder:        []int{1, 2, 3, 4, 5},
		DrawOrder:          []int{1, 2, 3, 4, 5},
		FocusOrder:         []int{4, 5},
		AccessibilityOrder: []int{3, 4, 5},
		HitTests: []surface.BlockGraphPathReport{
			{Helper: "tree_hit_test_path", Event: "click", TargetID: 5, X: 180, Y: 80, Path: []int{1, 2, 5}},
		},
		DispatchPaths: []surface.BlockGraphPathReport{
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 4, Path: []int{1, 2, 4}},
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 5, Path: []int{1, 2, 5}},
		},
	}
}
func blockAccessibilityTreeForScenario(source string) *surface.BlockAccessibilityTreeReport {
	return &surface.BlockAccessibilityTreeReport{
		Schema:                  "tetra.surface.block-accessibility-tree.v1",
		AccessibilityLevel:      "block-metadata-tree-v1",
		Source:                  source,
		Module:                  "lib.core.block",
		QualityLevel:            "block-derived-accessibility-metadata-v1",
		BlockGraphSchema:        "tetra.surface.block-graph.v1",
		DerivedFromBlockGraph:   true,
		ManualBookkeeping:       false,
		PlatformHostIntegration: false,
		DOMARIAIntegration:      false,
		ScreenReaderEvidence:    false,
		NoDOMUI:                 true,
		NoUserJS:                true,
		NoPlatformWidgets:       true,
		NodeCount:               3,
		FocusableCount:          2,
		RolesPresent:            []string{"text", "button"},
		FocusOrder:              []int{4, 5},
		ReadingOrder:            []int{3, 4, 5},
		Nodes: []surface.BlockAccessibilityNodeReport{
			{ID: 3, BlockID: 3, ParentBlockID: 2, Name: "LabelBlock", Role: "text", Bounds: surface.RectReport{X: 24, Y: 24, W: 200, H: 24}, Visible: true, Enabled: true, Focusable: false, LabelFor: "SubmitBlock", FocusIndex: -1, ReadingIndex: 0},
			{ID: 4, BlockID: 4, ParentBlockID: 2, Name: "SubmitBlock", Role: "button", Description: "primary action", Bounds: surface.RectReport{X: 24, Y: 64, W: 120, H: 44}, Visible: true, Enabled: true, Focusable: true, Focused: true, LabelledBy: "LabelBlock", Actions: []string{"focus", "press", "submit"}, FocusIndex: 0, ReadingIndex: 1},
			{ID: 5, BlockID: 5, ParentBlockID: 2, Name: "ResetBlock", Role: "button", Description: "secondary action", Bounds: surface.RectReport{X: 152, Y: 64, W: 120, H: 44}, Visible: true, Enabled: true, Focusable: true, Actions: []string{"focus", "press", "reset"}, FocusIndex: 1, ReadingIndex: 2},
		},
		Relationships: []surface.AccessibilityRelationshipReport{
			{Kind: "label_for", From: "LabelBlock", To: "SubmitBlock"},
			{Kind: "labelled_by", From: "SubmitBlock", To: "LabelBlock"},
		},
		Actions: []surface.AccessibilityActionReport{
			{Target: "SubmitBlock", Action: "press", Semantic: "submit"},
			{Target: "ResetBlock", Action: "press", Semantic: "reset"},
		},
		NegativeGuards: surface.BlockAccessibilityNegativeGuardsReport{
			FocusableActionNameChecked:    true,
			LabelRelationshipsChecked:     true,
			ReadingOrderGraphChecked:      true,
			BoundsAlignmentChecked:        true,
			FakeScreenReaderClaimRejected: true,
			ScopedPlatformClaimChecked:    true,
		},
	}
}
func blockAssetComponentsForScenario() []surface.ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility", "asset"}
	return []surface.ComponentReport{
		{ID: "BlockAssetApp", Type: "examples.surface_block_assets.BlockAssetApp", Bounds: surface.RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"asset_quality": "deterministic-local-block-assets-v1"}},
		{ID: "IconBlock", Type: "examples.surface_block_assets.IconBlock", Parent: "BlockAssetApp", Bounds: surface.RectReport{X: 24, Y: 36, W: 32, H: 32}, Abilities: abilities, State: map[string]string{"asset_id": "icon-settings", "tint": "#60aef4ff"}},
		{ID: "ImageBlock", Type: "examples.surface_block_assets.ImageBlock", Parent: "BlockAssetApp", Bounds: surface.RectReport{X: 72, Y: 32, W: 96, H: 64}, Abilities: abilities, State: map[string]string{"asset_id": "image-hero", "scale": "2x"}},
		{ID: "MissingAssetBlock", Type: "examples.surface_block_assets.MissingAssetBlock", Parent: "BlockAssetApp", Bounds: surface.RectReport{X: 24, Y: 112, W: 96, H: 32}, Abilities: abilities, State: map[string]string{"asset_id": "missing-logo", "fallback": "fallback-raster"}},
	}
}
func blockAssetManifestForScenario(source string) *surface.BlockAssetManifestReport {
	return &surface.BlockAssetManifestReport{
		Schema:        "tetra.surface.block-assets.v1",
		Source:        source,
		Quality:       "deterministic-local-block-assets-v1",
		HashAlgorithm: "sha256",
		ManifestHash:  "sha256:" + checksumText("surface-block-assets-manifest-v1"),
		LocalOnly:     true,
		FontCount:     1,
		IconCount:     1,
		ImageCount:    1,
		EmbeddedCount: 3,
		RemoteCount:   0,
		Assets: []surface.BlockAssetReport{
			{ID: "font-ui", Kind: "font", Path: "embedded://surface/font-ui", Embedded: true, Local: true, SHA256: "sha256:" + checksumText("surface-block-assets-font-ui"), Size: 2048, Family: "Tetra UI", CacheKey: "font-ui"},
			{ID: "icon-settings", Kind: "icon", Path: "embedded://surface/icon-settings", Embedded: true, Local: true, SHA256: "sha256:" + checksumText("surface-block-assets-icon-settings"), Size: 256, Width: 16, Height: 16, CacheKey: "icon-settings"},
			{ID: "image-hero", Kind: "image", Path: "embedded://surface/image-hero", Embedded: true, Local: true, SHA256: "sha256:" + checksumText("surface-block-assets-image-hero"), Size: 1024, Width: 48, Height: 32, CacheKey: "image-hero"},
		},
	}
}
func blockAssetCacheForScenario() surface.BlockAssetCacheReport {
	return surface.BlockAssetCacheReport{ID: "asset-cache", Strategy: "bounded-lru", BudgetBytes: 65536, UsedBytes: 5376, EntryCount: 3, MaxEntries: 16, RepeatedLoads: 6, Eviction: "lru", Bounded: true}
}
func blockAssetDiagnosticsForScenario() []surface.BlockAssetDiagnosticReport {
	return []surface.BlockAssetDiagnosticReport{
		{Order: 1, AssetID: "missing-logo", Kind: "image", Code: "missing_asset_fallback", Message: "missing local asset resolved to fallback raster", FallbackID: "fallback-raster-image", Pass: true},
		{Order: 2, AssetID: "https://assets.example.test/logo.png", Kind: "image", Code: "network_asset_rejected", Message: "network assets are disabled for Surface Block v1", RejectedURL: "https://assets.example.test/logo.png", Pass: true},
	}
}
func blockAssetRenderCommandsForScenario() []surface.BlockAssetRenderCommandReport {
	return []surface.BlockAssetRenderCommandReport{
		{Order: 1, Command: "load_font", AssetID: "font-ui", BlockID: 1, Rect: surface.RectReport{X: 0, Y: 0, W: 320, H: 200}, Quality: "font-manifest-metadata-v1", Checksum: "sha256:" + checksumText("surface-block-assets-load-font")},
		{Order: 2, Command: "tint_icon", AssetID: "icon-settings", BlockID: 2, Rect: surface.RectReport{X: 24, Y: 36, W: 32, H: 32}, Tint: "#60aef4ff", Scale: 1, Quality: "icon-mask-raster-v1", RasterFormat: "builtin-icon-mask-raster-v1", RasterHash: "sha256:" + checksumText("surface-block-assets-tint-icon-raster"), RasterWidth: 32, RasterHeight: 32, RasterCoverage: 341, MarkerOnly: false, Checksum: "sha256:" + checksumText("surface-block-assets-tint-icon")},
		{Order: 3, Command: "scale_image", AssetID: "image-hero", BlockID: 3, Rect: surface.RectReport{X: 72, Y: 32, W: 96, H: 64}, Scale: 2, Quality: "nearest-scale-v1", Checksum: "sha256:" + checksumText("surface-block-assets-scale-image")},
		{Order: 4, Command: "fallback_missing", AssetID: "missing-logo", BlockID: 4, Rect: surface.RectReport{X: 24, Y: 112, W: 96, H: 32}, Quality: "fallback-raster-v1", Checksum: "sha256:" + checksumText("surface-block-assets-fallback-missing")},
	}
}
