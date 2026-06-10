package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surfacevisual"
)

func TestValidateSurfaceVisualReportCommand(t *testing.T) {
	dir := t.TempDir()
	report := validCommandVisualReport(t, dir)
	reportPath := writeCommandVisualReport(t, dir, report)

	var stdout, stderr bytes.Buffer
	code := runValidateSurfaceVisualReport([]string{"--report", reportPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "surface visual report OK") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestValidateSurfaceVisualReportCommandRejectsTamperedPNGHash(t *testing.T) {
	dir := t.TempDir()
	report := validCommandVisualReport(t, dir)
	reportPath := writeCommandVisualReport(t, dir, report)
	if err := os.WriteFile(filepath.Join(dir, report.Scenes[0].BaselinePNG.Path), []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := runValidateSurfaceVisualReport([]string{"--report", reportPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q; want tampered hash rejection", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(strings.ToLower(stderr.String()), "sha256 mismatch") {
		t.Fatalf("stderr = %q, want sha256 mismatch", stderr.String())
	}
}

func validCommandVisualReport(t *testing.T, root string) surfacevisual.Report {
	t.Helper()
	write := func(rel string, contents string) surfacevisual.ArtifactRef {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256([]byte(contents))
		return surfacevisual.ArtifactRef{
			Path:   rel,
			SHA256: "sha256:" + hex.EncodeToString(sum[:]),
			Size:   int64(len(contents)),
		}
	}

	fontManifest := write("manifests/fonts.json", `{"fonts":["Tetra UI"]}`)
	assetManifest := write("manifests/assets.json", `{"assets":["icons"]}`)
	report := surfacevisual.Report{
		Schema:          surfacevisual.SchemaV1,
		Status:          "pass",
		Level:           surfacevisual.LevelVisualGoldenV1,
		Target:          "headless",
		Renderer:        "software-rgba",
		RendererVersion: "surface-software-rgba-golden-v1",
		BaselineRoot:    ".",
		DiffPolicy:      "fail-on-change-without-review",
		Tolerance:       surfacevisual.VisualTolerance{PixelDelta: 0, MaxChannelDelta: 0},
		Assets:          surfacevisual.AssetEvidence{FontManifest: fontManifest, AssetManifest: assetManifest},
		Operations: []surfacevisual.Operation{
			{Name: "render baselines", Kind: "render", Ran: true, Pass: true},
			{Name: "compare visual diffs", Kind: "diff", Ran: true, Pass: true},
			{Name: "validate artifact checksums", Kind: "checksum", Ran: true, Pass: true},
		},
		NegativeGuards: surfacevisual.NegativeGuards{
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
		Cases: []surfacevisual.CaseReport{
			{Name: "required scene baselines", Kind: "positive", Ran: true, Pass: true},
			{Name: "missing baseline rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "changed golden without review rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "screenshot without scene hash rejected", Kind: "negative", Ran: true, Pass: true},
		},
	}
	for i, id := range surfacevisual.RequiredScenes() {
		source := write("scenes/"+id+".json", `{"id":"`+id+`"}`)
		baseline := write("baselines/"+id+".png", "png-"+id)
		current := write("current/"+id+".png", "png-"+id)
		diff := write("diff/"+id+".png", "diff-"+id)
		report.Scenes = append(report.Scenes, surfacevisual.SceneReport{
			ID:                  id,
			Name:                strings.ReplaceAll(id, "-", " "),
			SourceScene:         source.Path,
			SceneSHA256:         source.SHA256,
			Target:              "headless",
			Renderer:            "software-rgba",
			RendererVersion:     "surface-software-rgba-golden-v1",
			Width:               320,
			Height:              200,
			Stride:              1280,
			Scale:               1,
			FrameOrder:          i + 1,
			BaselineFrameSHA256: baseline.SHA256,
			CurrentFrameSHA256:  baseline.SHA256,
			BaselinePNG:         baseline,
			CurrentPNG:          current,
			DiffPNG:             diff,
			Status:              "pass",
			FontManifestSHA256:  fontManifest.SHA256,
			AssetManifestSHA256: assetManifest.SHA256,
		})
	}
	return report
}

func writeCommandVisualReport(t *testing.T, dir string, report surfacevisual.Report) string {
	t.Helper()
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "surface-visual-report.json")
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
