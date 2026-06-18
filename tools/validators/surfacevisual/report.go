package surfacevisual

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	SchemaV1             = "tetra.surface.visual-regression.v1"
	LevelVisualGoldenV1  = "surface-visual-golden-v1"
	ReviewMarkerApproved = "surface-visual-review-approved"
)

var requiredScenes = []string{
	"command-palette",
	"dashboard",
	"settings",
	"editor",
	"glass",
}

type Report struct {
	Schema          string          `json:"schema"`
	Status          string          `json:"status"`
	Level           string          `json:"level"`
	Target          string          `json:"target"`
	Renderer        string          `json:"renderer"`
	RendererVersion string          `json:"renderer_version"`
	BaselineRoot    string          `json:"baseline_root"`
	DiffPolicy      string          `json:"diff_policy"`
	Tolerance       VisualTolerance `json:"tolerance"`
	Assets          AssetEvidence   `json:"assets"`
	Scenes          []SceneReport   `json:"scenes"`
	Operations      []Operation     `json:"operations"`
	NegativeGuards  NegativeGuards  `json:"negative_guards"`
	NonClaims       []string        `json:"nonclaims"`
	Cases           []CaseReport    `json:"cases"`
}

type VisualTolerance struct {
	PixelDelta      int `json:"pixel_delta"`
	MaxChannelDelta int `json:"max_channel_delta"`
}

type AssetEvidence struct {
	FontManifest  ArtifactRef `json:"font_manifest"`
	AssetManifest ArtifactRef `json:"asset_manifest"`
}

type SceneReport struct {
	ID                  string      `json:"id"`
	Name                string      `json:"name"`
	SourceScene         string      `json:"source_scene"`
	SceneSHA256         string      `json:"scene_sha256"`
	Target              string      `json:"target"`
	Renderer            string      `json:"renderer"`
	RendererVersion     string      `json:"renderer_version"`
	Width               int         `json:"width"`
	Height              int         `json:"height"`
	Stride              int         `json:"stride"`
	Scale               int         `json:"scale"`
	FrameOrder          int         `json:"frame_order"`
	BaselineFrameSHA256 string      `json:"baseline_frame_sha256"`
	CurrentFrameSHA256  string      `json:"current_frame_sha256"`
	BaselinePNG         ArtifactRef `json:"baseline_png"`
	CurrentPNG          ArtifactRef `json:"current_png"`
	DiffPNG             ArtifactRef `json:"diff_png"`
	PixelDelta          int         `json:"pixel_delta"`
	MaxChannelDelta     int         `json:"max_channel_delta"`
	TolerancePixels     int         `json:"tolerance_pixels"`
	ToleranceChannel    int         `json:"tolerance_channel"`
	Status              string      `json:"status"`
	ReviewMarker        string      `json:"review_marker,omitempty"`
	FontManifestSHA256  string      `json:"font_manifest_sha256"`
	AssetManifestSHA256 string      `json:"asset_manifest_sha256"`
}

type ArtifactRef struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type Operation struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Ran  bool   `json:"ran"`
	Pass bool   `json:"pass"`
}

type NegativeGuards struct {
	ScreenshotOnlyWithoutSceneHashRejected bool `json:"screenshot_only_without_scene_hash_rejected"`
	GoldenUpdateWithoutReviewRejected      bool `json:"golden_update_without_review_rejected"`
	MissingBaselineRejected                bool `json:"missing_baseline_rejected"`
	ChangedBaselineRejected                bool `json:"changed_baseline_rejected"`
	MissingRendererVersionRejected         bool `json:"missing_renderer_version_rejected"`
	MissingFontAssetHashesRejected         bool `json:"missing_font_asset_hashes_rejected"`
}

type CaseReport struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Ran  bool   `json:"ran"`
	Pass bool   `json:"pass"`
}

func RequiredScenes() []string {
	out := make([]string, len(requiredScenes))
	copy(out, requiredScenes)
	return out
}

func ValidateReport(raw []byte) error {
	report, err := decodeReport(raw)
	if err != nil {
		return err
	}
	return validateReport(report)
}

func ValidateReportWithRoot(raw []byte, root string) error {
	report, err := decodeReport(raw)
	if err != nil {
		return err
	}
	if err := validateReport(report); err != nil {
		return err
	}
	return validateArtifactFiles(report, root)
}

func decodeReport(raw []byte) (Report, error) {
	var report Report
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&report); err != nil {
		return Report{}, err
	}
	if err := ensureJSONEOF(dec); err != nil {
		return Report{}, err
	}
	return report, nil
}

func validateReport(report Report) error {
	var issues []string
	issues = append(issues, validateIdentity(report)...)
	issues = append(issues, validateArtifactRef("font manifest", report.Assets.FontManifest, false)...)
	issues = append(issues, validateArtifactRef("asset manifest", report.Assets.AssetManifest, false)...)
	issues = append(issues, validateScenes(report)...)
	issues = append(issues, validateOperations(report.Operations)...)
	issues = append(issues, validateNegativeGuards(report.NegativeGuards)...)
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	issues = append(issues, validateCases(report.Cases)...)
	if len(issues) > 0 {
		sort.Strings(issues)
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateIdentity(report Report) []string {
	var issues []string
	if report.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, SchemaV1))
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if report.Level != LevelVisualGoldenV1 {
		issues = append(issues, fmt.Sprintf("level is %q, want %q", report.Level, LevelVisualGoldenV1))
	}
	if report.Target != "headless" && report.Target != "linux-x64" && report.Target != "wasm32-web" {
		issues = append(issues, fmt.Sprintf("target is %q, want headless, linux-x64, or wasm32-web", report.Target))
	}
	if report.Renderer != "software-rgba" {
		issues = append(issues, fmt.Sprintf("renderer is %q, want software-rgba", report.Renderer))
	}
	if strings.TrimSpace(report.RendererVersion) == "" {
		issues = append(issues, "renderer_version is required")
	}
	if strings.TrimSpace(report.BaselineRoot) == "" {
		issues = append(issues, "baseline_root is required")
	}
	if report.DiffPolicy != "fail-on-change-without-review" {
		issues = append(issues, fmt.Sprintf("diff_policy is %q, want fail-on-change-without-review", report.DiffPolicy))
	}
	if report.Tolerance.PixelDelta < 0 || report.Tolerance.MaxChannelDelta < 0 {
		issues = append(issues, "tolerance values must be non-negative")
	}
	return issues
}

func validateScenes(report Report) []string {
	var issues []string
	seen := map[string]bool{}
	required := map[string]bool{}
	for _, id := range requiredScenes {
		required[id] = false
	}
	lastOrder := 0
	for i, scene := range report.Scenes {
		id := strings.TrimSpace(scene.ID)
		if id == "" {
			issues = append(issues, fmt.Sprintf("scenes[%d].id is required", i))
			continue
		}
		if seen[id] {
			issues = append(issues, fmt.Sprintf("duplicate scene %s", id))
		}
		seen[id] = true
		if _, ok := required[id]; ok {
			required[id] = true
		} else {
			issues = append(issues, fmt.Sprintf("unexpected scene %s", id))
		}
		if strings.TrimSpace(scene.Name) == "" {
			issues = append(issues, fmt.Sprintf("scene %s name is required", id))
		}
		if err := validateSafeRelPath(scene.SourceScene); err != nil {
			issues = append(issues, fmt.Sprintf("scene %s source_scene: %v", id, err))
		}
		if strings.TrimSpace(scene.SceneSHA256) == "" {
			issues = append(issues, fmt.Sprintf("scene %s scene hash is required", id))
		} else if !validSHA256(scene.SceneSHA256) {
			issues = append(issues, fmt.Sprintf("scene %s scene hash must be sha256:<hex>", id))
		}
		if scene.Target != report.Target {
			issues = append(issues, fmt.Sprintf("scene %s target is %q, want %q", id, scene.Target, report.Target))
		}
		if scene.Renderer != report.Renderer {
			issues = append(issues, fmt.Sprintf("scene %s renderer is %q, want %q", id, scene.Renderer, report.Renderer))
		}
		if scene.RendererVersion != report.RendererVersion {
			issues = append(issues, fmt.Sprintf("scene %s renderer_version is %q, want %q", id, scene.RendererVersion, report.RendererVersion))
		}
		if scene.Width <= 0 || scene.Height <= 0 || scene.Stride < scene.Width*4 {
			issues = append(issues, fmt.Sprintf("scene %s frame dimensions/stride are invalid", id))
		}
		if scene.Scale <= 0 {
			issues = append(issues, fmt.Sprintf("scene %s scale must be positive", id))
		}
		if scene.FrameOrder <= lastOrder {
			issues = append(issues, fmt.Sprintf("scene %s frame_order %d is not strictly greater than %d", id, scene.FrameOrder, lastOrder))
		}
		lastOrder = scene.FrameOrder
		for name, value := range map[string]string{
			"baseline_frame_sha256": scene.BaselineFrameSHA256,
			"current_frame_sha256":  scene.CurrentFrameSHA256,
			"font_manifest_sha256":  scene.FontManifestSHA256,
			"asset_manifest_sha256": scene.AssetManifestSHA256,
		} {
			if !validSHA256(value) {
				issues = append(issues, fmt.Sprintf("scene %s %s must be sha256:<hex>", id, name))
			}
		}
		issues = append(issues, validateArtifactRef("scene "+id+" baseline png", scene.BaselinePNG, true)...)
		issues = append(issues, validateArtifactRef("scene "+id+" current png", scene.CurrentPNG, true)...)
		issues = append(issues, validateArtifactRef("scene "+id+" diff png", scene.DiffPNG, true)...)
		if scene.TolerancePixels < 0 || scene.ToleranceChannel < 0 || scene.PixelDelta < 0 || scene.MaxChannelDelta < 0 {
			issues = append(issues, fmt.Sprintf("scene %s diff/tolerance values must be non-negative", id))
		}
		changed := scene.BaselineFrameSHA256 != scene.CurrentFrameSHA256 || scene.BaselinePNG.SHA256 != scene.CurrentPNG.SHA256 || scene.PixelDelta > scene.TolerancePixels || scene.MaxChannelDelta > scene.ToleranceChannel
		if changed {
			if scene.ReviewMarker != ReviewMarkerApproved {
				issues = append(issues, fmt.Sprintf("scene %s changed golden requires review marker %q", id, ReviewMarkerApproved))
			}
			if scene.Status != "approved-update" {
				issues = append(issues, fmt.Sprintf("scene %s status is %q, want approved-update for reviewed change", id, scene.Status))
			}
		} else if scene.Status != "pass" {
			issues = append(issues, fmt.Sprintf("scene %s status is %q, want pass", id, scene.Status))
		}
	}
	for _, id := range requiredScenes {
		if !required[id] {
			issues = append(issues, fmt.Sprintf("missing required scene %s", id))
		}
	}
	return issues
}

func validateArtifactRef(name string, artifact ArtifactRef, png bool) []string {
	var issues []string
	if err := validateSafeRelPath(artifact.Path); err != nil {
		issues = append(issues, fmt.Sprintf("%s path: %v", name, err))
	}
	if png && !strings.HasSuffix(strings.ToLower(artifact.Path), ".png") {
		issues = append(issues, fmt.Sprintf("%s path must end in .png", name))
	}
	if !validSHA256(artifact.SHA256) {
		issues = append(issues, fmt.Sprintf("%s sha256 must be sha256:<hex>", name))
	}
	if artifact.Size <= 0 {
		issues = append(issues, fmt.Sprintf("%s size must be positive", name))
	}
	return issues
}

func validateOperations(operations []Operation) []string {
	required := map[string]bool{"render": false, "diff": false, "checksum": false}
	var issues []string
	for _, operation := range operations {
		if strings.TrimSpace(operation.Name) == "" {
			issues = append(issues, "operation name is required")
		}
		if _, ok := required[operation.Kind]; ok {
			required[operation.Kind] = true
		}
		if operation.Kind == "" {
			issues = append(issues, fmt.Sprintf("operation %q kind is required", operation.Name))
		}
		if !operation.Ran {
			issues = append(issues, fmt.Sprintf("operation %s did not run", operation.Name))
		}
		if !operation.Pass {
			issues = append(issues, fmt.Sprintf("operation %s did not pass", operation.Name))
		}
	}
	for kind, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("missing required operation kind %s", kind))
		}
	}
	return issues
}

func validateNegativeGuards(guards NegativeGuards) []string {
	missing := []string{}
	if !guards.ScreenshotOnlyWithoutSceneHashRejected {
		missing = append(missing, "screenshot_only_without_scene_hash_rejected")
	}
	if !guards.GoldenUpdateWithoutReviewRejected {
		missing = append(missing, "golden_update_without_review_rejected")
	}
	if !guards.MissingBaselineRejected {
		missing = append(missing, "missing_baseline_rejected")
	}
	if !guards.ChangedBaselineRejected {
		missing = append(missing, "changed_baseline_rejected")
	}
	if !guards.MissingRendererVersionRejected {
		missing = append(missing, "missing_renderer_version_rejected")
	}
	if !guards.MissingFontAssetHashesRejected {
		missing = append(missing, "missing_font_asset_hashes_rejected")
	}
	if len(missing) == 0 {
		return nil
	}
	return []string{"negative_guards missing " + strings.Join(missing, ", ")}
}

func validateNonClaims(nonclaims []string) []string {
	var issues []string
	for _, want := range []string{"Electron", "CSS", "GPU"} {
		if !containsSubstring(nonclaims, want) {
			issues = append(issues, fmt.Sprintf("nonclaims missing %s boundary", want))
		}
	}
	return issues
}

func validateCases(cases []CaseReport) []string {
	required := map[string]bool{
		"required scene baselines":               false,
		"missing baseline rejected":              false,
		"changed golden without review rejected": false,
		"screenshot without scene hash rejected": false,
	}
	var issues []string
	for _, c := range cases {
		name := strings.TrimSpace(c.Name)
		if name == "" {
			issues = append(issues, "case name is required")
			continue
		}
		if _, ok := required[name]; ok {
			required[name] = true
		}
		if c.Kind != "positive" && c.Kind != "negative" {
			issues = append(issues, fmt.Sprintf("case %s kind is %q, want positive or negative", name, c.Kind))
		}
		if !c.Ran {
			issues = append(issues, fmt.Sprintf("case %s did not run", name))
		}
		if !c.Pass {
			issues = append(issues, fmt.Sprintf("case %s did not pass", name))
		}
	}
	for name, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("missing required case %q", name))
		}
	}
	return issues
}

func validateArtifactFiles(report Report, root string) error {
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	root = filepath.Clean(root)
	var issues []string
	for label, artifact := range map[string]ArtifactRef{
		"font manifest":  report.Assets.FontManifest,
		"asset manifest": report.Assets.AssetManifest,
	} {
		if err := validateArtifactFile(root, label, artifact); err != nil {
			issues = append(issues, err.Error())
		}
	}
	for _, scene := range report.Scenes {
		if err := validateSourceFileHash(root, "scene "+scene.ID+" source scene", scene.SourceScene, scene.SceneSHA256); err != nil {
			issues = append(issues, err.Error())
		}
		for label, artifact := range map[string]ArtifactRef{
			"scene " + scene.ID + " baseline png": scene.BaselinePNG,
			"scene " + scene.ID + " current png":  scene.CurrentPNG,
			"scene " + scene.ID + " diff png":     scene.DiffPNG,
		} {
			if err := validateArtifactFile(root, label, artifact); err != nil {
				issues = append(issues, err.Error())
			}
		}
	}
	if len(issues) > 0 {
		sort.Strings(issues)
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateArtifactFile(root string, label string, artifact ArtifactRef) error {
	if err := validateSafeRelPath(artifact.Path); err != nil {
		return fmt.Errorf("%s path: %w", label, err)
	}
	actual, size, err := hashFile(filepath.Join(root, filepath.FromSlash(artifact.Path)))
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	var issues []string
	if actual != artifact.SHA256 {
		issues = append(issues, fmt.Sprintf("sha256 mismatch for %s: got %s want %s", artifact.Path, actual, artifact.SHA256))
	}
	if size != artifact.Size {
		issues = append(issues, fmt.Sprintf("size mismatch for %s: got %d want %d", artifact.Path, size, artifact.Size))
	}
	if len(issues) > 0 {
		return fmt.Errorf("%s %s", label, strings.Join(issues, ", "))
	}
	return nil
}

func validateSourceFileHash(root string, label string, rel string, want string) error {
	if err := validateSafeRelPath(rel); err != nil {
		return fmt.Errorf("%s path: %w", label, err)
	}
	actual, _, err := hashFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	if actual != want {
		return fmt.Errorf("%s sha256 mismatch for %s: got %s want %s", label, rel, actual, want)
	}
	return nil
}

func hashFile(path string) (string, int64, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", 0, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", 0, fmt.Errorf("symlink artifact is not allowed")
	}
	if info.IsDir() {
		return "", 0, fmt.Errorf("directory artifact is not allowed")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", 0, err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), info.Size(), nil
}

func validateSafeRelPath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("is required")
	}
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute path %q is not allowed", path)
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return fmt.Errorf("unsafe relative path %q", path)
	}
	if strings.HasPrefix(clean, "-") {
		return fmt.Errorf("dash-prefixed path %q is not allowed", path)
	}
	return nil
}

func validSHA256(value string) bool {
	return regexp.MustCompile(`^sha256:[0-9a-f]{64}$`).MatchString(value)
}

func containsSubstring(values []string, want string) bool {
	want = strings.ToLower(want)
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), want) {
			return true
		}
	}
	return false
}

func ensureJSONEOF(dec *json.Decoder) error {
	var extra any
	if err := dec.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("unexpected trailing JSON value")
}
