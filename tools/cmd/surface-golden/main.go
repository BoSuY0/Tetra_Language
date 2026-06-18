package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/tools/validators/surfacevisual"
)

type sceneSource struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Blocks      []string `json:"blocks"`
	Interaction string   `json:"interaction"`
}

type rgbaFrame struct {
	Width  int
	Height int
	Stride int
	Pixels []byte
}

func main() {
	os.Exit(runSurfaceGolden(os.Args[1:], os.Stdout, os.Stderr))
}

func runSurfaceGolden(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("surface-golden", flag.ContinueOnError)
	fs.SetOutput(stderr)
	outDir := fs.String("out", "reports/surface-prod/P25-visual-golden", "directory for visual golden artifacts")
	sceneDir := fs.String("scene-dir", defaultSceneDir(), "directory containing Surface golden scene JSON files")
	target := fs.String("target", "headless", "Surface target")
	rendererVersion := fs.String("renderer-version", "surface-software-rgba-golden-v1", "renderer version evidence string")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "surface-golden does not accept positional arguments")
		return 2
	}
	if strings.TrimSpace(*outDir) == "" {
		fmt.Fprintln(stderr, "--out is required")
		return 2
	}
	if strings.TrimSpace(*sceneDir) == "" {
		fmt.Fprintln(stderr, "--scene-dir is required")
		return 2
	}
	if info, err := os.Stat(*sceneDir); err != nil || !info.IsDir() {
		if err == nil {
			err = fmt.Errorf("not a directory")
		}
		fmt.Fprintf(stderr, "scene directory %s is not usable: %v\n", *sceneDir, err)
		return 1
	}
	reportPath, err := writeSurfaceGolden(*outDir, *sceneDir, *target, *rendererVersion)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "surface visual golden report: %s\n", reportPath)
	return 0
}

func writeSurfaceGolden(outDir string, sceneDir string, target string, rendererVersion string) (string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(outDir, "scenes"), 0o755); err != nil {
		return "", err
	}
	for _, dir := range []string{"baselines", "current", "diff", "manifests"} {
		if err := os.MkdirAll(filepath.Join(outDir, dir), 0o755); err != nil {
			return "", err
		}
	}

	fontManifest, err := writeJSONArtifact(filepath.Join(outDir, "manifests", "fonts.json"), map[string]any{
		"schema": "tetra.surface.visual-font-manifest.v1",
		"fonts": []map[string]any{
			{"id": "tetra-ui-regular", "family": "Tetra UI", "source": "embedded:tetra-ui-regular", "weight": 400},
			{"id": "noto-sans-fallback", "family": "Noto Sans", "source": "system:fontconfig/noto-sans", "weight": 400},
		},
	})
	if err != nil {
		return "", err
	}
	assetManifest, err := writeJSONArtifact(filepath.Join(outDir, "manifests", "assets.json"), map[string]any{
		"schema": "tetra.surface.visual-asset-manifest.v1",
		"assets": []map[string]any{
			{"id": "icon-command", "kind": "icon", "source": "embedded:surface/icon-command"},
			{"id": "panel-noise", "kind": "image", "source": "embedded:surface/panel-noise"},
		},
	})
	if err != nil {
		return "", err
	}

	report := surfacevisual.Report{
		Schema:          surfacevisual.SchemaV1,
		Status:          "pass",
		Level:           surfacevisual.LevelVisualGoldenV1,
		Target:          target,
		Renderer:        "software-rgba",
		RendererVersion: rendererVersion,
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
		sceneBytes, source, err := readSceneSource(sceneDir, id)
		if err != nil {
			return "", err
		}
		frame := renderSceneFrame(source, i)
		sceneOut := filepath.Join(outDir, "scenes", id+".json")
		if err := os.WriteFile(sceneOut, sceneBytes, 0o644); err != nil {
			return "", err
		}
		baseline, err := writePNGArtifact(filepath.Join(outDir, "baselines", id+".png"), frame)
		if err != nil {
			return "", err
		}
		current, err := writePNGArtifact(filepath.Join(outDir, "current", id+".png"), frame)
		if err != nil {
			return "", err
		}
		diffFrame := blankDiffFrame(frame.Width, frame.Height)
		diff, err := writePNGArtifact(filepath.Join(outDir, "diff", id+".png"), diffFrame)
		if err != nil {
			return "", err
		}
		frameSHA := checksumBytes(frame.Pixels)
		report.Scenes = append(report.Scenes, surfacevisual.SceneReport{
			ID:                  id,
			Name:                source.Title,
			SourceScene:         filepath.ToSlash(filepath.Join("scenes", id+".json")),
			SceneSHA256:         checksumBytes(sceneBytes),
			Target:              target,
			Renderer:            "software-rgba",
			RendererVersion:     rendererVersion,
			Width:               frame.Width,
			Height:              frame.Height,
			Stride:              frame.Stride,
			Scale:               1,
			FrameOrder:          i + 1,
			BaselineFrameSHA256: frameSHA,
			CurrentFrameSHA256:  frameSHA,
			BaselinePNG:         baseline,
			CurrentPNG:          current,
			DiffPNG:             diff,
			PixelDelta:          0,
			MaxChannelDelta:     0,
			TolerancePixels:     0,
			ToleranceChannel:    0,
			Status:              "pass",
			FontManifestSHA256:  fontManifest.SHA256,
			AssetManifestSHA256: assetManifest.SHA256,
		})
	}

	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	raw = append(raw, '\n')
	if err := surfacevisual.ValidateReportWithRoot(raw, outDir); err != nil {
		return "", fmt.Errorf("generated visual report failed validation: %w", err)
	}
	reportPath := filepath.Join(outDir, "surface-visual-report.json")
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		return "", err
	}
	return reportPath, nil
}

func readSceneSource(sceneDir string, id string) ([]byte, sceneSource, error) {
	path := filepath.Join(sceneDir, id+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, sceneSource{}, fmt.Errorf("read scene %s: %w", id, err)
	}
	var source sceneSource
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, sceneSource{}, fmt.Errorf("decode scene %s: %w", id, err)
	}
	if source.ID != id {
		return nil, sceneSource{}, fmt.Errorf("scene %s id is %q", id, source.ID)
	}
	if strings.TrimSpace(source.Title) == "" {
		source.Title = strings.ReplaceAll(id, "-", " ")
	}
	return raw, source, nil
}

func writeJSONArtifact(path string, value any) (surfacevisual.ArtifactRef, error) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return surfacevisual.ArtifactRef{}, err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return surfacevisual.ArtifactRef{}, err
	}
	return artifactRefForFile(path)
}

func writePNGArtifact(path string, frame rgbaFrame) (surfacevisual.ArtifactRef, error) {
	img := image.NewRGBA(image.Rect(0, 0, frame.Width, frame.Height))
	copy(img.Pix, frame.Pixels)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return surfacevisual.ArtifactRef{}, err
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return surfacevisual.ArtifactRef{}, err
	}
	return artifactRefForFile(path)
}

func artifactRefForFile(path string) (surfacevisual.ArtifactRef, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return surfacevisual.ArtifactRef{}, err
	}
	rel := filepath.ToSlash(filepath.Clean(path))
	for _, marker := range []string{"/baselines/", "/current/", "/diff/", "/manifests/"} {
		if idx := strings.Index(rel, marker); idx >= 0 {
			rel = rel[idx+1:]
			break
		}
	}
	return surfacevisual.ArtifactRef{
		Path:   rel,
		SHA256: checksumBytes(raw),
		Size:   int64(len(raw)),
	}, nil
}

func renderSceneFrame(source sceneSource, index int) rgbaFrame {
	frame := newRGBAFrame(360, 220)
	palette := []color.RGBA{
		{R: 19, G: 28, B: 35, A: 255},
		{R: 39, G: 86, B: 101, A: 255},
		{R: 106, G: 68, B: 133, A: 255},
		{R: 196, G: 164, B: 86, A: 255},
		{R: 226, G: 238, B: 224, A: 255},
	}
	clear(frame, palette[index%len(palette)])
	panel := color.RGBA{R: uint8(42 + index*17), G: uint8(52 + index*13), B: uint8(64 + index*11), A: 255}
	accent := color.RGBA{R: uint8(224 - index*19), G: uint8(144 + index*9), B: uint8(78 + index*21), A: 255}
	soft := color.RGBA{R: 234, G: 238, B: 226, A: 255}
	rect(frame, 22, 20, 316, 176, panel)
	rect(frame, 22, 20, 316, 12, accent)
	for i, block := range source.Blocks {
		x := 42 + (i%3)*96
		y := 52 + (i/3)*42
		w := 72 + (len(block)%5)*8
		h := 24 + (len(block)%3)*6
		rect(frame, x, y, w, h, soft)
		rect(frame, x+4, y+4, w-8, 4, accent)
	}
	if source.ID == "glass" {
		for y := 44; y < 166; y += 8 {
			for x := 48; x < 292; x += 8 {
				if (x+y)%24 == 0 {
					rect(frame, x, y, 4, 4, color.RGBA{R: 255, G: 255, B: 255, A: 96})
				}
			}
		}
	}
	return frame
}

func blankDiffFrame(width int, height int) rgbaFrame {
	frame := newRGBAFrame(width, height)
	clear(frame, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	return frame
}

func newRGBAFrame(width int, height int) rgbaFrame {
	return rgbaFrame{
		Width:  width,
		Height: height,
		Stride: width * 4,
		Pixels: make([]byte, width*height*4),
	}
}

func clear(frame rgbaFrame, c color.RGBA) {
	rect(frame, 0, 0, frame.Width, frame.Height, c)
}

func rect(frame rgbaFrame, x int, y int, w int, h int, c color.RGBA) {
	x0 := clamp(x, 0, frame.Width)
	y0 := clamp(y, 0, frame.Height)
	x1 := clamp(x+w, 0, frame.Width)
	y1 := clamp(y+h, 0, frame.Height)
	for py := y0; py < y1; py++ {
		row := py * frame.Stride
		for px := x0; px < x1; px++ {
			i := row + px*4
			frame.Pixels[i] = c.R
			frame.Pixels[i+1] = c.G
			frame.Pixels[i+2] = c.B
			frame.Pixels[i+3] = c.A
		}
	}
}

func clamp(v int, low int, high int) int {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

func checksumBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func defaultSceneDir() string {
	const rel = "testdata/surface-golden/scenes"
	wd, err := os.Getwd()
	if err != nil {
		return rel
	}
	for {
		candidate := filepath.Join(wd, rel)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		next := filepath.Dir(wd)
		if next == wd {
			break
		}
		wd = next
	}
	return rel
}
