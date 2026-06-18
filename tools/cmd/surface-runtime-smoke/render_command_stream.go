package main

import (
	"tetra_language/tools/internal/surfacerender"
)

func attachRenderCommandStreamForScenario(source string, scenario *headlessScenario) {
	attachRenderCommandStreamForScenarioWithRenderer(source, "software-rgba-headless", scenario)
}

func attachRenderCommandStreamForScenarioWithRenderer(source string, renderer string, scenario *headlessScenario) {
	if scenario == nil || scenario.BlockSceneSnapshot == nil {
		return
	}
	scenario.RenderCommandStream = surfacerender.BuildCommandStream(
		source,
		renderer,
		scenario.BlockSceneSnapshot,
		scenario.Frames,
	)
}
