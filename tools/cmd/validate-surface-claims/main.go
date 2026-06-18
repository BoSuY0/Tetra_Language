package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"tetra_language/tools/validators/surface"
)

type surfaceClaimOptions struct {
	Root       string
	ReportDirs []string
	GitHead    string
}

type stringListFlag []string

func (f *stringListFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("empty report-dir")
	}
	*f = append(*f, value)
	return nil
}

type surfaceClaimIssue struct {
	Path    string
	Line    int
	Rule    string
	Snippet string
}

type surfaceClaimClause struct {
	Text string
	Line int
}

type surfaceClaimEvidence struct {
	WindowsProductionTargetHost bool
	MacOSProductionTargetHost   bool
	MorphRenderedBeautySameHead bool
	MorphRenderedBeautyProduct  bool
	NativeSurfaceHostStrict     bool
}

var defaultSurfaceClaimScanPaths = []string{
	"README.md",
	"docs/spec",
	"docs/release",
	"docs/user",
	"docs/audits",
	"docs/design",
	"docs/backend",
	"docs/checklists",
	"docs/generated/manifest.json",
	"docs/generated/v1_0/manifest.json",
	"compiler/compiler_facade.go",
	"examples",
	"lib/core",
	"scripts/release/surface",
	"tools/validators/surface",
	"tools/cmd/validate-surface-runtime",
	"tools/cmd/validate-surface-release-state",
	"tools/cmd/validate-surface-block-report",
	"tools/cmd/validate-surface-block-examples",
	"tools/cmd/validate-surface-morph-report",
	"tools/cmd/validate-surface-morph-rendered-beauty",
}

func main() {
	var opt surfaceClaimOptions
	var reportDirs stringListFlag
	flag.StringVar(&opt.Root, "root", ".", "repository root to scan")
	flag.Var(&reportDirs, "report-dir", "Surface report directory to scan; may be repeated")
	flag.Parse()
	opt.ReportDirs = []string(reportDirs)
	if err := validateSurfaceClaims(opt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateSurfaceClaims(opt surfaceClaimOptions) error {
	root := strings.TrimSpace(opt.Root)
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	files, err := surfaceClaimScanFiles(absRoot, opt.ReportDirs)
	if err != nil {
		return err
	}
	gitHead := strings.TrimSpace(opt.GitHead)
	if gitHead == "" {
		gitHead = currentSurfaceClaimGitHead(absRoot)
	}
	evidence := collectSurfaceClaimEvidence(files, gitHead)
	var issues []surfaceClaimIssue
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !utf8.Valid(raw) {
			continue
		}
		issues = append(issues, inspectSurfaceClaimText(absRoot, path, string(raw), evidence)...)
	}
	if len(issues) > 0 {
		return surfaceClaimIssuesError(issues)
	}
	return nil
}

func surfaceClaimScanFiles(root string, reportDirs []string) ([]string, error) {
	seen := map[string]bool{}
	var files []string
	addFile := func(path string) {
		clean := filepath.Clean(path)
		if seen[clean] {
			return
		}
		seen[clean] = true
		files = append(files, clean)
	}
	addPath := func(path string, required bool, reportRoot bool) error {
		info, err := os.Stat(path)
		if err != nil {
			if required {
				return err
			}
			return nil
		}
		if !info.IsDir() {
			if shouldScanSurfaceClaimFile(path) {
				addFile(path)
			}
			return nil
		}
		return filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if skipSurfaceClaimDir(path, reportRoot) {
					return filepath.SkipDir
				}
				return nil
			}
			if shouldScanSurfaceClaimFile(path) {
				addFile(path)
			}
			return nil
		})
	}
	for _, rel := range defaultSurfaceClaimScanPaths {
		if err := addPath(filepath.Join(root, filepath.FromSlash(rel)), false, false); err != nil {
			return nil, err
		}
	}
	for _, reportDir := range reportDirs {
		path := filepath.Clean(reportDir)
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		if err := addPath(path, true, true); err != nil {
			return nil, fmt.Errorf("scan report-dir %s: %w", reportDir, err)
		}
	}
	sort.Strings(files)
	return files, nil
}

func skipSurfaceClaimDir(path string, reportRoot bool) bool {
	name := filepath.Base(path)
	switch name {
	case ".git", ".cache", ".workflow", "dumps", "graphify-out", "node_modules", "vendor":
		return true
	case "reports":
		return !reportRoot
	default:
		return false
	}
}

func shouldScanSurfaceClaimFile(path string) bool {
	base := filepath.Base(path)
	if strings.HasSuffix(base, "_test.go") {
		return false
	}
	switch strings.ToLower(filepath.Ext(base)) {
	case ".go", ".json", ".md", ".sh", ".tetra", ".txt", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func collectSurfaceClaimEvidence(files []string, gitHead string) surfaceClaimEvidence {
	var evidence surfaceClaimEvidence
	for _, path := range files {
		if !strings.Contains(filepath.ToSlash(path), "/reports/") {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil || !utf8.Valid(raw) {
			continue
		}
		lower := strings.ToLower(string(raw))
		if hasTargetHostProductionEvidence(lower, "windows") {
			evidence.WindowsProductionTargetHost = true
		}
		if hasTargetHostProductionEvidence(lower, "macos") {
			evidence.MacOSProductionTargetHost = true
		}
		if report, ok := morphRenderedBeautyClaimReport(raw); ok && report.GitHead == gitHead &&
			report.GitCommit == gitHead {
			evidence.MorphRenderedBeautySameHead = true
			if report.ProductClaim && report.FinalSignoff {
				evidence.MorphRenderedBeautyProduct = true
			}
		}
		if hasStrictNativeSurfaceHostEvidence(raw) {
			evidence.NativeSurfaceHostStrict = true
		}
	}
	return evidence
}

func currentSurfaceClaimGitHead(root string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	raw, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

func morphRenderedBeautyClaimReport(raw []byte) (surface.MorphRenderedBeautyReport, bool) {
	var meta struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return surface.MorphRenderedBeautyReport{}, false
	}
	if meta.Schema != surface.MorphRenderedBeautyReportSchemaV1 {
		return surface.MorphRenderedBeautyReport{}, false
	}
	var report surface.MorphRenderedBeautyReport
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return surface.MorphRenderedBeautyReport{}, false
	}
	if err := surface.ValidateMorphRenderedBeautyReportValue(report); err != nil {
		return surface.MorphRenderedBeautyReport{}, false
	}
	return report, true
}

func hasTargetHostProductionEvidence(lower string, target string) bool {
	return strings.Contains(lower, target) &&
		strings.Contains(lower, "real_window") &&
		strings.Contains(lower, "true") &&
		strings.Contains(lower, "production_claim") &&
		strings.Contains(lower, "true")
}

func inspectSurfaceClaimText(
	root string,
	path string,
	text string,
	evidence surfaceClaimEvidence,
) []surfaceClaimIssue {
	rel := relativeSurfaceClaimPath(root, path)
	var issues []surfaceClaimIssue
	if containsStaleProductionEvidence(text) {
		issues = append(issues, surfaceClaimIssue{
			Path:    rel,
			Line:    lineNumberForSurfaceClaimNeedle(text, "same_commit_validated"),
			Rule:    "stale production evidence claim",
			Snippet: "production_claim=true with same_commit_validated=false",
		})
	}
	for _, clause := range surfaceClaimClauses(text) {
		lower := normalizeSurfaceClaimText(clause.Text)
		if lower == "" || surfaceClaimClauseBoundaryOnly(lower) {
			continue
		}
		for _, runtime := range []string{"electron", "react", "css"} {
			if containsRuntimeReplacementClaim(lower, runtime) &&
				!surfaceClaimExplicitNonClaimContext(lower) {
				issues = append(issues, surfaceClaimIssue{
					Path: rel,
					Line: clause.Line,
					Rule: fmt.Sprintf(
						"%s replacement overclaim",
						surfaceClaimRuntimeLabel(runtime),
					),
					Snippet: strings.TrimSpace(clause.Text),
				})
			}
		}
		if containsSurfaceQualityClaim(lower) && !evidence.MorphRenderedBeautySameHead &&
			!surfaceClaimExplicitNonClaimContext(lower) {
			issues = append(issues, surfaceClaimIssue{
				Path:    rel,
				Line:    clause.Line,
				Rule:    "Surface beauty/quality claim without same-commit Morph rendered beauty evidence",
				Snippet: strings.TrimSpace(clause.Text),
			})
		}
		if containsProductionMorphClaim(lower) && !surfaceClaimExplicitNonClaimContext(lower) &&
			!evidence.MorphRenderedBeautyProduct {
			issues = append(issues, surfaceClaimIssue{
				Path:    rel,
				Line:    clause.Line,
				Rule:    "production Morph overclaim without same-commit Morph rendered beauty product signoff",
				Snippet: strings.TrimSpace(clause.Text),
			})
		}
		if containsTargetProductionClaim(lower, "windows") &&
			!evidence.WindowsProductionTargetHost &&
			!surfaceClaimExplicitNonClaimContext(lower) {
			issues = append(issues, surfaceClaimIssue{
				Path:    rel,
				Line:    clause.Line,
				Rule:    "Windows Surface production claim without target-host evidence",
				Snippet: strings.TrimSpace(clause.Text),
			})
		}
		if containsTargetProductionClaim(lower, "macos") && !evidence.MacOSProductionTargetHost &&
			!surfaceClaimExplicitNonClaimContext(lower) {
			issues = append(issues, surfaceClaimIssue{
				Path:    rel,
				Line:    clause.Line,
				Rule:    "macOS Surface production claim without target-host evidence",
				Snippet: strings.TrimSpace(clause.Text),
			})
		}
		if containsGPUProductionClaim(lower) && !surfaceClaimExplicitNonClaimContext(lower) {
			issues = append(issues, surfaceClaimIssue{
				Path:    rel,
				Line:    clause.Line,
				Rule:    "GPU Surface production claim without evidence",
				Snippet: strings.TrimSpace(clause.Text),
			})
		}
		if containsDocsOnlyProductionClaim(lower) && !surfaceClaimExplicitNonClaimContext(lower) {
			issues = append(issues, surfaceClaimIssue{
				Path:    rel,
				Line:    clause.Line,
				Rule:    "docs-only Surface production claim",
				Snippet: strings.TrimSpace(clause.Text),
			})
		}
		if containsNativeSurfaceHostPromotionClaim(lower) &&
			!surfaceClaimExplicitNonClaimContext(lower) &&
			!evidence.NativeSurfaceHostStrict {
			issues = append(issues, surfaceClaimIssue{
				Path:    rel,
				Line:    clause.Line,
				Rule:    "Native Surface Host v1 promotion without strict native-host evidence",
				Snippet: strings.TrimSpace(clause.Text),
			})
		}
	}
	return issues
}

func hasStrictNativeSurfaceHostEvidence(raw []byte) bool {
	var report struct {
		Schema       string `json:"schema"`
		Target       string `json:"target"`
		Runtime      string `json:"runtime"`
		HostEvidence struct {
			Level                     string `json:"level"`
			Backend                   string `json:"backend"`
			Framebuffer               bool   `json:"framebuffer"`
			RealWindow                bool   `json:"real_window"`
			NativeInput               bool   `json:"native_input"`
			UserFacingPlatformWidgets bool   `json:"user_facing_platform_widgets"`
		} `json:"host_evidence"`
		NativeSurfaceHost *surface.NativeSurfaceHostReport `json:"native_surface_host"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		return false
	}
	evidence := report.NativeSurfaceHost
	if report.Schema != surface.SchemaV1 || evidence == nil {
		return false
	}
	return report.Target == "linux-x64" &&
		report.Runtime == "surface-linux-x64" &&
		report.HostEvidence.Level == surface.NativeSurfaceHostLevelLinuxX64 &&
		report.HostEvidence.Backend == surface.NativeSurfaceHostBackendWayland &&
		report.HostEvidence.Framebuffer &&
		report.HostEvidence.RealWindow &&
		report.HostEvidence.NativeInput &&
		!report.HostEvidence.UserFacingPlatformWidgets &&
		evidence.Schema == surface.NativeSurfaceHostSchemaV1 &&
		evidence.Host == "wayland" &&
		evidence.Protocol == surface.NativeSurfaceHostProtocolV1 &&
		evidence.AppProcessKind == "compiled-linux-x64-tetra-app" &&
		evidence.HostProcessKind == "tetra-surface-host-wayland" &&
		evidence.AppPID > 0 &&
		evidence.HostPID > 0 &&
		evidence.AppPID != evidence.HostPID &&
		evidence.SurfaceOpenFromApp &&
		evidence.PollEventFromHost &&
		evidence.PresentFromAppRGBA &&
		evidence.AppLoopObserved &&
		evidence.RealWindow &&
		evidence.RealCloseEvent &&
		evidence.RealPointerEventCount > 0 &&
		evidence.RealKeyEventCount > 0 &&
		evidence.PresentedFrameCount >= 2 &&
		!evidence.PreRenderedFrameSource &&
		evidence.DeliveryPath == "compiled-tetra-app-to-wayland-surface"
}

func relativeSurfaceClaimPath(root string, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func surfaceClaimClauses(text string) []surfaceClaimClause {
	var clauses []surfaceClaimClause
	lineNo := 1
	for _, line := range strings.SplitAfter(text, "\n") {
		start := 0
		for i, r := range line {
			if r == '.' || r == ';' || r == '!' || r == '?' {
				if chunk := strings.TrimSpace(line[start:i]); chunk != "" {
					clauses = append(clauses, surfaceClaimClause{Text: chunk, Line: lineNo})
				}
				start = i + len(string(r))
			}
		}
		if chunk := strings.TrimSpace(line[start:]); chunk != "" {
			clauses = append(clauses, surfaceClaimClause{Text: chunk, Line: lineNo})
		}
		lineNo += strings.Count(line, "\n")
	}
	return clauses
}

func normalizeSurfaceClaimText(text string) string {
	return strings.ToLower(strings.Join(strings.Fields(text), " "))
}

func surfaceClaimClauseBoundaryOnly(lower string) bool {
	return surfaceClaimClauseSafe(lower) && !surfaceClaimPromotes(lower)
}

func surfaceClaimClauseSafe(lower string) bool {
	return containsAnySubstring(lower, []string{
		"future work",
		"remain future",
		"remains future",
		"unsupported",
		"outside",
		"without",
		"must not",
		"cannot",
		"forbid",
		"forbids",
		"reject",
		"rejected",
		"absent",
		"blocked",
		"blocker",
		"does not",
		"do not",
		"until",
		"requires real",
		"require real",
		"requires target-host",
		"require target-host",
		"only with target",
		"only with real",
		"only after",
		"only when",
		"evidence before",
		"must be false",
		"productionclaim",
		"production_claim\": false",
		"production_claim=false",
		"requires git_head evidence",
	})
}

func surfaceClaimExplicitNonClaimContext(lower string) bool {
	return containsAnySubstring(lower, []string{
		" not ",
		"not ",
		" no ",
		"no ",
		"nonclaim",
		"nonclaims",
		"non-claim",
		"non-goal",
		"must not",
		"cannot",
		"forbid",
		"forbids",
		"reject",
		"rejected",
		"absent",
		"until",
		"does not",
		"do not",
		"only after",
		"only when",
		"must be false",
		"productionclaim",
		"production_claim\": false",
		"production_claim=false",
		"requires git_head evidence",
	})
}

func containsRuntimeReplacementClaim(lower string, runtime string) bool {
	if !containsAnySubstring(lower, []string{"surface", "tetra", "electron/react/css"}) {
		return false
	}
	forms := []string{
		"full " + runtime + " replacement",
		runtime + " replacement",
		"replace " + runtime,
		"replaces " + runtime,
		"replacing " + runtime,
		"replacement for " + runtime,
	}
	if runtime == "electron" || runtime == "react" || runtime == "css" {
		forms = append(forms,
			"replace electron/react/css",
			"replaces electron/react/css",
			"replacing electron/react/css",
			"electron/react/css replacement",
		)
	}
	return containsAnySubstring(lower, forms)
}

func surfaceClaimRuntimeLabel(runtime string) string {
	switch runtime {
	case "css":
		return "CSS"
	case "electron":
		return "Electron"
	case "react":
		return "React"
	default:
		return runtime
	}
}

func containsGPUProductionClaim(lower string) bool {
	return strings.Contains(lower, "surface") &&
		strings.Contains(lower, "gpu") &&
		surfaceClaimPromotes(lower)
}

func containsDocsOnlyProductionClaim(lower string) bool {
	return strings.Contains(lower, "surface") &&
		strings.Contains(lower, "docs-only") &&
		surfaceClaimPromotes(lower)
}

func containsNativeSurfaceHostPromotionClaim(lower string) bool {
	if surfaceClaimArtifactContext(lower) {
		return false
	}
	if !containsNativeSurfaceHostReference(lower) {
		return false
	}
	return surfaceClaimPromotes(lower)
}

func containsNativeSurfaceHostReference(lower string) bool {
	return containsAnySubstring(lower, []string{
		"native surface host",
		"native-host",
		"native host v1",
		"tetra.surface.native-host.v1",
		"linux-x64-native-surface-host-v1",
		"linux-x64-native-host",
	})
}

func containsStaleProductionEvidence(text string) bool {
	compact := compactSurfaceClaimText(text)
	return strings.Contains(compact, `"production_claim":true`) &&
		(strings.Contains(compact, `"same_commit_validated":false`) ||
			strings.Contains(compact, `"stale_evidence":true`))
}

func compactSurfaceClaimText(text string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(text) {
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func lineNumberForSurfaceClaimNeedle(text string, needle string) int {
	lower := strings.ToLower(text)
	index := strings.Index(lower, strings.ToLower(needle))
	if index < 0 {
		return 1
	}
	return 1 + strings.Count(text[:index], "\n")
}

func containsProductionMorphClaim(lower string) bool {
	if !strings.Contains(lower, "morph") {
		return false
	}
	if containsAnySubstring(lower, []string{
		`"production_claim": true`,
		`"production_claim":true`,
		"production_claim=true",
		"production morph",
		"morph production",
		"production-ready morph",
		"morph production-ready",
		"morph is production-ready",
		"morph is ready for production",
		"morph ready for production",
		"morph production support",
		"production support for morph",
		"production-supported morph",
		"morph production-supported",
		"morph prod_stable_scoped",
	}) {
		return true
	}
	return false
}

func containsSurfaceQualityClaim(lower string) bool {
	if surfaceClaimArtifactContext(lower) {
		return false
	}
	if containsRuntimeQualityClaim(lower, "electron") ||
		containsRuntimeQualityClaim(lower, "react") {
		return true
	}
	if containsPixelPerfectSurfaceClaim(lower) {
		return true
	}
	if containsMorphBeautyClaim(lower) {
		return true
	}
	return false
}

func surfaceClaimArtifactContext(lower string) bool {
	return containsAnySubstring(lower, []string{
		`"path"`,
		`"root"`,
		`"command_line"`,
		"/reports/",
		"reports/",
		"--report-dir",
	})
}

func containsRuntimeQualityClaim(lower string, runtime string) bool {
	if !strings.Contains(lower, runtime) {
		return false
	}
	if !containsAnySubstring(lower, []string{
		runtime + "-quality",
		runtime + " quality",
		runtime + " grade",
		runtime + "-grade",
	}) {
		return false
	}
	return containsAnySubstring(lower, []string{
		"surface",
		"tetra",
		"ui",
		"user interface",
		"app shell",
	})
}

func containsPixelPerfectSurfaceClaim(lower string) bool {
	if !containsAnySubstring(lower, []string{"pixel-perfect", "pixel perfect"}) {
		return false
	}
	return containsAnySubstring(lower, []string{
		"surface",
		"tetra",
		"ui",
		"user interface",
		"morph",
		"rendering",
	})
}

func containsMorphBeautyClaim(lower string) bool {
	if !strings.Contains(lower, "morph") || !strings.Contains(lower, "beauty") {
		return false
	}
	return surfaceClaimPromotes(lower) ||
		containsAnySubstring(lower, []string{
			"beauty is proven",
			"beauty is guaranteed",
			"beauty is ready",
			"beauty layer is proven",
			"beauty layer is ready",
			"can be described as the rendered beauty layer",
			"is the rendered beauty layer",
			"has morph rendered beauty",
			"morph rendered beauty is proven",
			"morph rendered beauty is available",
			"production beauty",
			"product beauty",
		})
}

func containsTargetProductionClaim(lower string, target string) bool {
	return strings.Contains(lower, "surface") &&
		strings.Contains(lower, target) &&
		surfaceClaimPromotes(lower)
}

func surfaceClaimPromotes(lower string) bool {
	return containsAnySubstring(lower, []string{
		"release-ready",
		"release ready",
		"production-supported",
		"production supported",
		"production support",
		"prod_stable_scoped",
		"ready",
	}) ||
		containsSurfaceClaimWord(lower, "current") ||
		containsSurfaceClaimWord(lower, "production") ||
		containsSurfaceClaimWord(lower, "supported")
}

func containsSurfaceClaimWord(lower string, word string) bool {
	for _, field := range strings.FieldsFunc(lower, func(r rune) bool {
		return r < 'a' || r > 'z'
	}) {
		if field == word {
			return true
		}
	}
	return false
}

func containsAnySubstring(text string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func surfaceClaimIssuesError(issues []surfaceClaimIssue) error {
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Path != issues[j].Path {
			return issues[i].Path < issues[j].Path
		}
		if issues[i].Line != issues[j].Line {
			return issues[i].Line < issues[j].Line
		}
		return issues[i].Rule < issues[j].Rule
	})
	const maxIssues = 20
	var b strings.Builder
	fmt.Fprintf(&b, "surface claim validation failed: %d issue(s)", len(issues))
	limit := len(issues)
	if limit > maxIssues {
		limit = maxIssues
	}
	for i := 0; i < limit; i++ {
		issue := issues[i]
		fmt.Fprintf(&b, "; %s:%d: %s", issue.Path, issue.Line, issue.Rule)
		if issue.Snippet != "" {
			fmt.Fprintf(&b, ": %q", trimSurfaceClaimSnippet(issue.Snippet))
		}
	}
	if len(issues) > maxIssues {
		fmt.Fprintf(&b, "; and %d more", len(issues)-maxIssues)
	}
	return errors.New(b.String())
}

func trimSurfaceClaimSnippet(snippet string) string {
	snippet = strings.Join(strings.Fields(snippet), " ")
	if len(snippet) <= 180 {
		return snippet
	}
	return snippet[:177] + "..."
}
