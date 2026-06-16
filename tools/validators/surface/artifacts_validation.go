package surface

import (
	"encoding/hex"
	"fmt"
	"strings"
)

func validateProcesses(source string, processes []ProcessReport) []string {
	var issues []string
	if len(processes) < 3 {
		issues = append(issues, fmt.Sprintf("process evidence has %d entries, want build, app, and runtime processes", len(processes)))
	}
	seen := map[string]bool{}
	seenBuild := false
	seenBuildForSource := false
	seenApp := false
	seenComponentApp := false
	seenRuntime := false
	for _, process := range processes {
		if strings.TrimSpace(process.Name) == "" {
			issues = append(issues, "process name is required")
		} else if seen[process.Name] {
			issues = append(issues, fmt.Sprintf("duplicate process %s", process.Name))
		}
		seen[process.Name] = true
		switch process.Kind {
		case "build":
			seenBuild = true
			if processReferencesSource(process.Path, source) {
				seenBuildForSource = true
			}
		case "app":
			seenApp = true
			if isSurfaceComponentAppProcess(source, process) {
				seenComponentApp = true
			}
		case "runtime":
			seenRuntime = true
		default:
			issues = append(issues, fmt.Sprintf("process %s kind is %q, want build, app, or runtime", process.Name, process.Kind))
		}
		if strings.TrimSpace(process.Path) == "" {
			issues = append(issues, fmt.Sprintf("process %s path is required", process.Name))
		} else if process.Kind == "app" && sourceLikeEvidencePath(process.Path) {
			issues = append(issues, fmt.Sprintf("process %s path %q is not executable Surface app process evidence", process.Name, process.Path))
		}
		if !process.Ran {
			issues = append(issues, fmt.Sprintf("process %s did not run", process.Name))
		}
		if !process.Pass {
			issues = append(issues, fmt.Sprintf("process %s did not pass", process.Name))
		}
		if process.ExitCode == nil {
			issues = append(issues, fmt.Sprintf("process %s missing exit_code", process.Name))
			continue
		}
		wantExit := 0
		if process.ExpectedExitCode != nil {
			wantExit = *process.ExpectedExitCode
		}
		if *process.ExitCode != wantExit {
			issues = append(issues, fmt.Sprintf("process %s exit_code = %d, want %d", process.Name, *process.ExitCode, wantExit))
		}
	}
	if !seenBuild {
		issues = append(issues, "process evidence missing build process")
	}
	if !seenBuildForSource {
		issues = append(issues, fmt.Sprintf("process evidence missing build process for reported source %q", source))
	}
	if !seenApp {
		issues = append(issues, "process evidence missing executable Surface app process")
	}
	if !seenComponentApp {
		issues = append(issues, "process evidence missing executable Surface component app process with expected app exit")
	}
	if !seenRuntime {
		issues = append(issues, "process evidence missing Surface runtime process")
	}
	return issues
}

func processReferencesSource(path string, source string) bool {
	source = normalizeEvidencePath(source)
	if source == "" {
		return false
	}
	path = normalizeEvidencePath(path)
	return strings.Contains(path, source)
}

func isSurfaceComponentAppProcess(source string, process ProcessReport) bool {
	name := strings.ToLower(strings.TrimSpace(process.Name))
	if process.Kind != "app" || !strings.Contains(name, "surface") || !strings.Contains(name, "component app") {
		return false
	}
	if process.ExitCode == nil || process.ExpectedExitCode == nil {
		return false
	}
	if *process.ExitCode == 1 && *process.ExpectedExitCode == 1 {
		return true
	}
	if isSurfaceProjectTemplateSource(source) && *process.ExitCode == 0 && *process.ExpectedExitCode == 0 {
		return true
	}
	if isSurfaceReferenceAppSource(source) && *process.ExitCode == 0 && *process.ExpectedExitCode == 0 {
		return true
	}
	if isBlockPaintValidationComponentApp(source, process) && *process.ExitCode == 0 && *process.ExpectedExitCode == 0 {
		return true
	}
	if isSurfaceFlagshipControlCenterSource(source) && *process.ExitCode == 5 && *process.ExpectedExitCode == 5 {
		return true
	}
	if isSurfaceMorphRenderedFlagshipSource(source) && *process.ExitCode == 0 && *process.ExpectedExitCode == 0 {
		return true
	}
	return strings.Contains(name, "browser canvas") && *process.ExitCode == 0 && *process.ExpectedExitCode == 0
}

func isSurfaceProjectTemplateSource(source string) bool {
	source = normalizeEvidencePath(source)
	if !strings.HasSuffix(source, "/src/main.tetra") {
		return false
	}
	parts := strings.Split(source, "/")
	for i, part := range parts {
		if part != "templates" || i+3 != len(parts)-1 {
			continue
		}
		if parts[i+1] == "" || parts[i+2] != "src" || parts[i+3] != "main.tetra" {
			continue
		}
		for _, prefix := range parts[:i] {
			if prefix == "reports" {
				return true
			}
		}
	}
	return false
}

func isSurfaceReferenceAppSource(source string) bool {
	source = normalizeEvidencePath(source)
	if strings.HasPrefix(source, "examples/surface_reference_") && strings.HasSuffix(source, ".tetra") {
		return true
	}
	return strings.Contains(source, "/examples/surface_reference_") && strings.HasSuffix(source, ".tetra")
}

func isBlockPaintValidationComponentApp(source string, process ProcessReport) bool {
	return normalizeEvidencePath(source) == "examples/surface_block_paint_layers.tetra" &&
		strings.Contains(normalizeEvidencePath(process.Path), "surface-block-paint")
}

func isSurfaceFlagshipControlCenterSource(source string) bool {
	source = normalizeEvidencePath(source)
	return source == "examples/surface_migration_tetra_control_center.tetra" ||
		strings.HasSuffix(source, "/examples/surface_migration_tetra_control_center.tetra")
}

func isSurfaceMorphRenderedFlagshipSource(source string) bool {
	source = normalizeEvidencePath(source)
	return source == "examples/surface_morph_rendered_studio_shell.tetra" ||
		strings.HasSuffix(source, "/examples/surface_morph_rendered_studio_shell.tetra")
}

func normalizeEvidencePath(path string) string {
	path = strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	return path
}

func validateArtifacts(target string, source string, artifacts []ArtifactReport, processes []ProcessReport) []string {
	var issues []string
	if len(artifacts) == 0 {
		issues = append(issues, "artifact evidence is required")
	}
	seenPath := map[string]bool{}
	seenComponentAppArtifact := false
	seenCompilerOwnedLoaderArtifact := false
	seenRunnerTraceArtifact := false
	for _, artifact := range artifacts {
		kind := strings.TrimSpace(artifact.Kind)
		path := normalizeEvidencePath(artifact.Path)
		if kind == "" {
			issues = append(issues, "artifact kind is required")
		}
		if path == "" {
			issues = append(issues, fmt.Sprintf("artifact %s path is required", kind))
		} else if seenPath[path] {
			issues = append(issues, fmt.Sprintf("duplicate artifact path %s", artifact.Path))
		}
		seenPath[path] = true
		issues = append(issues, validateSurfaceArtifactPath(kind, path)...)
		if !validSHA256Digest(artifact.SHA256) {
			issues = append(issues, fmt.Sprintf("artifact %s sha256 must be sha256:<64 hex>", artifact.Path))
		}
		if artifact.Size <= 0 {
			issues = append(issues, fmt.Sprintf("artifact %s size must be positive", artifact.Path))
		}
		if kind == "component-app" && artifactReferencedByComponentAppProcess(source, path, processes) {
			seenComponentAppArtifact = true
		}
		if kind == "compiler-owned-loader" && strings.HasSuffix(strings.ToLower(path), ".mjs") {
			seenCompilerOwnedLoaderArtifact = true
		}
		if kind == "runner-trace" && strings.HasSuffix(strings.ToLower(path), "surface-runner-trace.json") {
			seenRunnerTraceArtifact = true
		}
	}
	if !seenComponentAppArtifact {
		issues = append(issues, "artifact evidence missing Surface component app artifact hash linked to Surface component app process")
	}
	if target == "wasm32-web" && !seenCompilerOwnedLoaderArtifact {
		issues = append(issues, "wasm32-web artifact evidence missing compiler-owned loader artifact")
	}
	if (target == "headless" || target == "wasm32-web") && !seenRunnerTraceArtifact {
		issues = append(issues, fmt.Sprintf("%s artifact evidence missing Surface runner trace artifact", target))
	}
	return issues
}

func validateSurfaceArtifactPath(kind string, path string) []string {
	lower := strings.ToLower(path)
	var issues []string
	if strings.Contains(lower, ".ui.") {
		issues = append(issues, fmt.Sprintf("artifact %s must not be a legacy UI sidecar", path))
	}
	if strings.HasSuffix(lower, ".html") {
		issues = append(issues, fmt.Sprintf("artifact %s must not be generated HTML UI", path))
	}
	if strings.HasSuffix(lower, ".js") {
		issues = append(issues, fmt.Sprintf("artifact %s must not be generated JavaScript UI", path))
	}
	if strings.HasSuffix(lower, ".mjs") && kind != "compiler-owned-loader" {
		issues = append(issues, fmt.Sprintf("artifact %s .mjs is only allowed for compiler-owned-loader evidence", path))
	}
	for _, forbidden := range []struct {
		suffix string
		model  string
	}{
		{suffix: ".jsx", model: "React"},
		{suffix: ".tsx", model: "React"},
		{suffix: ".qml", model: "Qt"},
		{suffix: ".xaml", model: "WinUI"},
		{suffix: ".xib", model: "Cocoa"},
		{suffix: ".storyboard", model: "Cocoa"},
		{suffix: ".glade", model: "GTK"},
	} {
		if strings.HasSuffix(lower, forbidden.suffix) {
			issues = append(issues, fmt.Sprintf("artifact %s must not be %s user-facing UI evidence", path, forbidden.model))
		}
	}
	if kind == "compiler-owned-loader" && !strings.HasSuffix(lower, ".mjs") {
		issues = append(issues, fmt.Sprintf("compiler-owned loader artifact %s must be a .mjs loader", path))
	}
	return issues
}

func validateArtifactScan(scan ArtifactScanReport, artifacts []ArtifactReport) []string {
	var issues []string
	root := normalizeEvidencePath(scan.Root)
	if root == "" {
		issues = append(issues, "artifact_scan.root is required")
	}
	if scan.FilesChecked <= 0 {
		issues = append(issues, "artifact_scan.files_checked must be positive")
	}
	if len(artifacts) > 0 && scan.FilesChecked < len(artifacts) {
		issues = append(issues, fmt.Sprintf("artifact_scan.files_checked = %d, want at least %d reported artifacts", scan.FilesChecked, len(artifacts)))
	}
	if !scan.Pass {
		issues = append(issues, "artifact_scan.pass must be true")
	}
	if len(scan.ForbiddenPaths) > 0 {
		issues = append(issues, fmt.Sprintf("artifact_scan forbidden paths must be empty, got %d", len(scan.ForbiddenPaths)))
	}
	for _, path := range scan.ForbiddenPaths {
		if strings.TrimSpace(path) == "" {
			issues = append(issues, "artifact_scan forbidden path must not be empty")
		}
	}
	for _, artifact := range artifacts {
		path := normalizeEvidencePath(artifact.Path)
		if root == "" || path == "" {
			continue
		}
		if !evidencePathUnderRoot(path, root) {
			issues = append(issues, fmt.Sprintf("artifact %s is outside artifact_scan.root %s", artifact.Path, scan.Root))
		}
	}
	return issues
}

func evidencePathUnderRoot(path string, root string) bool {
	path = strings.TrimSuffix(normalizeEvidencePath(path), "/")
	root = strings.TrimSuffix(normalizeEvidencePath(root), "/")
	return path == root || strings.HasPrefix(path, root+"/")
}

func artifactReferencedByComponentAppProcess(source string, artifactPath string, processes []ProcessReport) bool {
	for _, process := range processes {
		if !isSurfaceComponentAppProcess(source, process) {
			continue
		}
		if strings.Contains(normalizeEvidencePath(process.Path), artifactPath) {
			return true
		}
	}
	return false
}

func validSHA256Digest(value string) bool {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "sha256:") {
		return false
	}
	hexDigest := strings.TrimPrefix(value, "sha256:")
	if len(hexDigest) != 64 {
		return false
	}
	_, err := hex.DecodeString(hexDigest)
	return err == nil
}
