package surfacevisual

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateReportAcceptsSurfaceGoldenBaselines(t *testing.T) {
	report := validVisualReport()
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport(valid) = %v", err)
	}
}

func TestValidateReportRejectsScreenshotOnlyWithoutSceneHash(t *testing.T) {
	report := validVisualReport()
	report.Scenes[0].SceneSHA256 = ""
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	err = ValidateReport(raw)
	if err == nil {
		t.Fatal("ValidateReport accepted screenshot-only golden evidence without a scene hash")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "scene hash") {
		t.Fatalf("ValidateReport error = %v, want scene hash rejection", err)
	}
}

func TestValidateReportRejectsGoldenUpdateWithoutReviewMarker(t *testing.T) {
	report := validVisualReport()
	report.Scenes[0].CurrentFrameSHA256 = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	report.Scenes[0].CurrentPNG.SHA256 = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	report.Scenes[0].PixelDelta = 32
	report.Scenes[0].MaxChannelDelta = 8
	report.Scenes[0].Status = "changed"
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	err = ValidateReport(raw)
	if err == nil {
		t.Fatal("ValidateReport accepted a changed golden baseline without a review marker")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "review marker") {
		t.Fatalf("ValidateReport error = %v, want review marker rejection", err)
	}
}

func TestValidateReportRequiresProductionSceneCoverage(t *testing.T) {
	report := validVisualReport()
	report.Scenes = report.Scenes[:len(report.Scenes)-1]
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	err = ValidateReport(raw)
	if err == nil {
		t.Fatal("ValidateReport accepted incomplete visual scene coverage")
	}
	if !strings.Contains(err.Error(), "glass") {
		t.Fatalf("ValidateReport error = %v, want missing glass scene", err)
	}
}

func validVisualReport() Report {
	scenes := make([]SceneReport, 0, len(RequiredScenes()))
	for i, id := range RequiredScenes() {
		seed := strings.Repeat(string(rune('a'+i)), 64)
		scenes = append(scenes, SceneReport{
			ID:                  id,
			Name:                strings.ReplaceAll(id, "-", " "),
			SourceScene:         "testdata/surface-golden/scenes/" + id + ".json",
			SceneSHA256:         "sha256:" + seed,
			Target:              "headless",
			Renderer:            "software-rgba",
			RendererVersion:     "surface-software-rgba-golden-v1",
			Width:               320,
			Height:              200,
			Stride:              1280,
			Scale:               1,
			FrameOrder:          i + 1,
			BaselineFrameSHA256: "sha256:" + seed,
			CurrentFrameSHA256:  "sha256:" + seed,
			BaselinePNG: ArtifactRef{
				Path:   "baselines/" + id + ".png",
				SHA256: "sha256:" + seed,
				Size:   128,
			},
			CurrentPNG: ArtifactRef{
				Path:   "current/" + id + ".png",
				SHA256: "sha256:" + seed,
				Size:   128,
			},
			DiffPNG: ArtifactRef{
				Path:   "diff/" + id + ".png",
				SHA256: "sha256:" + strings.Repeat("f", 64),
				Size:   96,
			},
			PixelDelta:       0,
			MaxChannelDelta:  0,
			TolerancePixels:  0,
			ToleranceChannel: 0,
			Status:           "pass",
			FontManifestSHA256: "sha256:" +
				strings.Repeat("1", 64),
			AssetManifestSHA256: "sha256:" +
				strings.Repeat("2", 64),
		})
	}
	return Report{
		Schema:          SchemaV1,
		Status:          "pass",
		Level:           LevelVisualGoldenV1,
		Target:          "headless",
		Renderer:        "software-rgba",
		RendererVersion: "surface-software-rgba-golden-v1",
		BaselineRoot:    "testdata/surface-golden",
		DiffPolicy:      "fail-on-change-without-review",
		Tolerance: VisualTolerance{
			PixelDelta:      0,
			MaxChannelDelta: 0,
		},
		Assets: AssetEvidence{
			FontManifest: ArtifactRef{
				Path:   "manifests/fonts.json",
				SHA256: "sha256:" + strings.Repeat("1", 64),
				Size:   64,
			},
			AssetManifest: ArtifactRef{
				Path:   "manifests/assets.json",
				SHA256: "sha256:" + strings.Repeat("2", 64),
				Size:   64,
			},
		},
		Scenes: scenes,
		Operations: []Operation{
			{Name: "render baselines", Kind: "render", Ran: true, Pass: true},
			{Name: "compare visual diffs", Kind: "diff", Ran: true, Pass: true},
			{Name: "validate artifact checksums", Kind: "checksum", Ran: true, Pass: true},
		},
		NegativeGuards: NegativeGuards{
			ScreenshotOnlyWithoutSceneHashRejected: true,
			GoldenUpdateWithoutReviewRejected:      true,
			MissingBaselineRejected:                true,
			ChangedBaselineRejected:                true,
			MissingRendererVersionRejected:         true,
			MissingFontAssetHashesRejected:         true,
		},
		NonClaims: []string{
			"not Electron or Chromium pixel parity",
			"not CSS browser rendering parity",
			"not GPU compositor parity",
		},
		Cases: []CaseReport{
			{Name: "required scene baselines", Kind: "positive", Ran: true, Pass: true},
			{Name: "missing baseline rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "changed golden without review rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "screenshot without scene hash rejected", Kind: "negative", Ran: true, Pass: true},
		},
	}
}
