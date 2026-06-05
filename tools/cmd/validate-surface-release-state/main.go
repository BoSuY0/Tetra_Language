package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/tools/validators/surface"
)

const surfaceArtifactHashSchema = "tetra.release-artifact-hashes.v1alpha1"

type surfaceReleaseStateOptions struct {
	ReportDir      string
	ExpectedStatus string
	Scope          string
	ManifestPath   string
}

type surfaceReleaseArtifactHashManifest struct {
	Schema    string                       `json:"schema"`
	Root      string                       `json:"root"`
	Artifacts []surfaceReleaseHashArtifact `json:"artifacts"`
}

type surfaceReleaseHashArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Schema string `json:"schema,omitempty"`
}

type surfaceReleaseRuntimeEnvelope struct {
	Schema       string                     `json:"schema"`
	Status       string                     `json:"status"`
	Target       string                     `json:"target"`
	Source       string                     `json:"source"`
	HostEvidence surface.HostEvidenceReport `json:"host_evidence"`
}

func main() {
	var opt surfaceReleaseStateOptions
	flag.StringVar(&opt.ReportDir, "report-dir", "", "Surface release report directory")
	flag.StringVar(&opt.ExpectedStatus, "expected-status", "current", "expected Surface release status")
	flag.StringVar(&opt.Scope, "scope", surface.ReleaseScopeSurfaceV1LinuxWeb, "expected Surface release scope")
	flag.StringVar(&opt.ManifestPath, "manifest", "docs/generated/manifest.json", "docs/generated manifest path")
	flag.Parse()
	if strings.TrimSpace(opt.ReportDir) == "" {
		fmt.Fprintln(os.Stderr, "error: --report-dir is required")
		os.Exit(2)
	}
	if err := validateSurfaceReleaseState(opt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateSurfaceReleaseState(opt surfaceReleaseStateOptions) error {
	reportDir := strings.TrimSpace(opt.ReportDir)
	if reportDir == "" {
		return errors.New("report-dir is required")
	}
	expectedStatus := strings.TrimSpace(opt.ExpectedStatus)
	if expectedStatus == "" {
		expectedStatus = "current"
	}
	scope := strings.TrimSpace(opt.Scope)
	if scope == "" {
		scope = surface.ReleaseScopeSurfaceV1LinuxWeb
	}
	var issues []string
	if expectedStatus != "current" {
		issues = append(issues, fmt.Sprintf("expected-status is %q, want current", expectedStatus))
	}
	if scope != surface.ReleaseScopeSurfaceV1LinuxWeb {
		issues = append(issues, fmt.Sprintf("scope is %q, want %q", scope, surface.ReleaseScopeSurfaceV1LinuxWeb))
	}
	issues = append(issues, validateReleaseSummaryFile(filepath.Join(reportDir, "surface-release-summary.json"), scope, expectedStatus)...)
	issues = append(issues, validateReleaseTextInputFile(filepath.Join(reportDir, "surface-headless-release-text-input.json"))...)
	issues = append(issues, validateReleaseRuntimeEnvelopeFile(filepath.Join(reportDir, "surface-wasm32-web-release-browser.json"), "wasm32-web")...)
	issues = append(issues, validateReleaseRuntimeEnvelopeFile(filepath.Join(reportDir, "surface-linux-x64-release-window.json"), "linux-x64")...)
	issues = append(issues, validateSurfaceArtifactHashes(filepath.Join(reportDir, "artifact-hashes.json"))...)
	issues = append(issues, validateSurfaceReleaseManifest(opt.ManifestPath, scope, expectedStatus)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateReleaseSummaryFile(path string, scope string, expectedStatus string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s read failed: %v", filepath.Base(path), err)}
	}
	if err := surface.ValidateReleaseSummary(raw); err != nil {
		return []string{fmt.Sprintf("%s invalid: %v", filepath.Base(path), err)}
	}
	var report surface.ReleaseSummaryReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return []string{fmt.Sprintf("%s decode failed: %v", filepath.Base(path), err)}
	}
	var issues []string
	if report.ReleaseScope != scope {
		issues = append(issues, fmt.Sprintf("%s release_scope is %q, want %q", filepath.Base(path), report.ReleaseScope, scope))
	}
	if report.Status != expectedStatus {
		issues = append(issues, fmt.Sprintf("%s status is %q, want %q", filepath.Base(path), report.Status, expectedStatus))
	}
	return issues
}

func validateReleaseTextInputFile(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s read failed: %v", filepath.Base(path), err)}
	}
	if err := surface.ValidateTextInputReport(raw); err != nil {
		return []string{fmt.Sprintf("%s invalid: %v", filepath.Base(path), err)}
	}
	return nil
}

func validateReleaseRuntimeEnvelopeFile(path string, target string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s read failed: %v", filepath.Base(path), err)}
	}
	var report surfaceReleaseRuntimeEnvelope
	if err := json.Unmarshal(raw, &report); err != nil {
		return []string{fmt.Sprintf("%s decode failed: %v", filepath.Base(path), err)}
	}
	var issues []string
	if report.Schema != surface.SchemaV1 {
		issues = append(issues, fmt.Sprintf("%s schema is %q, want %q", filepath.Base(path), report.Schema, surface.SchemaV1))
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("%s status is %q, want pass", filepath.Base(path), report.Status))
	}
	if report.Target != target {
		issues = append(issues, fmt.Sprintf("%s target is %q, want %q", filepath.Base(path), report.Target, target))
	}
	if report.Source != "examples/surface_release_form.tetra" {
		issues = append(issues, fmt.Sprintf("%s source is %q, want examples/surface_release_form.tetra", filepath.Base(path), report.Source))
	}
	if report.HostEvidence.UserFacingPlatformWidgets {
		issues = append(issues, fmt.Sprintf("%s must not claim user-facing platform widgets", filepath.Base(path)))
	}
	switch target {
	case "linux-x64":
		if report.HostEvidence.Level != "linux-x64-release-window-v1" {
			issues = append(issues, fmt.Sprintf("%s host_evidence.level is %q, want linux-x64-release-window-v1", filepath.Base(path), report.HostEvidence.Level))
		}
		if report.HostEvidence.Backend != "wayland-shm-rgba-release-v1" {
			issues = append(issues, fmt.Sprintf("%s host_evidence.backend is %q, want wayland-shm-rgba-release-v1", filepath.Base(path), report.HostEvidence.Backend))
		}
		for _, check := range []struct {
			name string
			ok   bool
		}{
			{name: "real_window", ok: report.HostEvidence.RealWindow},
			{name: "native_input", ok: report.HostEvidence.NativeInput},
			{name: "text_input", ok: report.HostEvidence.TextInput},
			{name: "clipboard", ok: report.HostEvidence.Clipboard},
			{name: "composition", ok: report.HostEvidence.Composition},
			{name: "accessibility_bridge", ok: report.HostEvidence.AccessibilityBridge},
		} {
			if !check.ok {
				issues = append(issues, fmt.Sprintf("%s host_evidence.%s must be true", filepath.Base(path), check.name))
			}
		}
	case "wasm32-web":
		if report.HostEvidence.Level != "wasm32-web-browser-canvas-release-v1" {
			issues = append(issues, fmt.Sprintf("%s host_evidence.level is %q, want wasm32-web-browser-canvas-release-v1", filepath.Base(path), report.HostEvidence.Level))
		}
		if report.HostEvidence.Backend != "browser-canvas-rgba-accessible" {
			issues = append(issues, fmt.Sprintf("%s host_evidence.backend is %q, want browser-canvas-rgba-accessible", filepath.Base(path), report.HostEvidence.Backend))
		}
		for _, check := range []struct {
			name string
			ok   bool
		}{
			{name: "browser_canvas", ok: report.HostEvidence.BrowserCanvas},
			{name: "browser_input", ok: report.HostEvidence.BrowserInput},
			{name: "browser_clipboard", ok: report.HostEvidence.BrowserClipboard},
			{name: "browser_composition", ok: report.HostEvidence.BrowserComposition},
			{name: "browser_accessibility_snapshot", ok: report.HostEvidence.BrowserAccessibilitySnapshot},
			{name: "browser_accessibility_mirror", ok: report.HostEvidence.BrowserAccessibilityMirror},
		} {
			if !check.ok {
				issues = append(issues, fmt.Sprintf("%s host_evidence.%s must be true", filepath.Base(path), check.name))
			}
		}
	}
	return issues
}

func validateSurfaceArtifactHashes(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s read failed: %v", filepath.Base(path), err)}
	}
	var manifest surfaceReleaseArtifactHashManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return []string{fmt.Sprintf("%s decode failed: %v", filepath.Base(path), err)}
	}
	var issues []string
	if manifest.Schema != surfaceArtifactHashSchema {
		issues = append(issues, fmt.Sprintf("%s schema is %q, want %q", filepath.Base(path), manifest.Schema, surfaceArtifactHashSchema))
	}
	if strings.TrimSpace(manifest.Root) == "" || filepath.IsAbs(manifest.Root) || strings.Contains(manifest.Root, "..") {
		issues = append(issues, fmt.Sprintf("%s root is unsafe or empty", filepath.Base(path)))
	}
	if len(manifest.Artifacts) == 0 {
		issues = append(issues, fmt.Sprintf("%s artifacts must not be empty", filepath.Base(path)))
	}
	root := filepath.Join(filepath.Dir(path), filepath.FromSlash(manifest.Root))
	for _, artifact := range manifest.Artifacts {
		if artifact.Path == "" || filepath.IsAbs(artifact.Path) || strings.Contains(artifact.Path, "..") {
			issues = append(issues, fmt.Sprintf("%s contains unsafe artifact path %q", filepath.Base(path), artifact.Path))
			continue
		}
		size, digest, err := hashFile(filepath.Join(root, filepath.FromSlash(artifact.Path)))
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s artifact %s read failed: %v", filepath.Base(path), artifact.Path, err))
			continue
		}
		if size != artifact.Size {
			issues = append(issues, fmt.Sprintf("%s artifact %s size = %d, want %d", filepath.Base(path), artifact.Path, size, artifact.Size))
		}
		if digest != artifact.SHA256 {
			issues = append(issues, fmt.Sprintf("%s artifact %s sha256 = %s, want %s", filepath.Base(path), artifact.Path, digest, artifact.SHA256))
		}
	}
	return issues
}

func validateSurfaceReleaseManifest(path string, scope string, expectedStatus string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("manifest %s read failed: %v", path, err)}
	}
	text := string(raw)
	var issues []string
	for _, want := range []string{
		scope,
		expectedStatus,
		"docs/spec/surface_v1.md",
		"docs/user/surface_guide.md",
		"docs/user/examples_index.md",
	} {
		if !strings.Contains(text, want) {
			issues = append(issues, fmt.Sprintf("manifest %s missing %q", path, want))
		}
	}
	var manifest struct {
		Features []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"features"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		issues = append(issues, fmt.Sprintf("manifest %s decode failed: %v", path, err))
		return issues
	}
	requiredSurfaceFeatures := map[string]string{
		"ui.surface-core":             expectedStatus,
		"ui.surface-headless":         expectedStatus,
		"ui.surface-linux-x64":        expectedStatus,
		"ui.surface-web-wasm":         expectedStatus,
		"ui.surface-component-model":  expectedStatus,
		"ui.surface-toolkit-v1":       expectedStatus,
		"ui.surface-text-input-v1":    expectedStatus,
		"ui.surface-accessibility-v1": expectedStatus,
		"ui.surface-macos-x64":        "unsupported",
		"ui.surface-windows-x64":      "unsupported",
		"ui.surface-wasm32-wasi":      "unsupported",
	}
	seen := map[string]string{}
	for _, feature := range manifest.Features {
		if _, ok := requiredSurfaceFeatures[feature.ID]; ok {
			seen[feature.ID] = feature.Status
		}
	}
	for id, wantStatus := range requiredSurfaceFeatures {
		if gotStatus, ok := seen[id]; !ok {
			issues = append(issues, fmt.Sprintf("manifest %s missing Surface release feature %s", path, id))
		} else if gotStatus != wantStatus {
			issues = append(issues, fmt.Sprintf("manifest %s Surface release feature %s status is %q, want %q", path, id, gotStatus, wantStatus))
		}
	}
	return issues
}

func hashFile(path string) (int64, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer file.Close()
	h := sha256.New()
	size, err := io.Copy(h, file)
	if err != nil {
		return 0, "", err
	}
	return size, "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}
