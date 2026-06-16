package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	ctarget "tetra_language/compiler/target"
	"tetra_language/tools/internal/reportdecode"
)

type smokeCaseReport struct {
	Name               string `json:"name"`
	SrcPath            string `json:"src_path"`
	OutPath            string `json:"out_path,omitempty"`
	ExpectedExit       int    `json:"expected_exit"`
	Unsupported        bool   `json:"unsupported,omitempty"`
	ExpectedDiagnostic string `json:"expected_diagnostic,omitempty"`
	Diagnostic         string `json:"diagnostic,omitempty"`
	ActualExit         *int   `json:"actual_exit,omitempty"`
	Ran                bool   `json:"ran"`
	Pass               bool   `json:"pass"`
	Error              string `json:"error,omitempty"`
}

type islandsDebugScopeRow struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	CaseName string `json:"case_name,omitempty"`
	SrcPath  string `json:"src_path,omitempty"`
	Evidence string `json:"evidence"`
	Reason   string `json:"reason"`
}

type smokeReport struct {
	Timestamp         string                 `json:"timestamp"`
	Target            string                 `json:"target"`
	BuildOnly         bool                   `json:"build_only,omitempty"`
	Runner            string                 `json:"runner,omitempty"`
	Host              string                 `json:"host"`
	Version           string                 `json:"version"`
	GitHead           string                 `json:"git_head,omitempty"`
	IslandsDebug      bool                   `json:"islands_debug"`
	IslandsDebugScope []islandsDebugScopeRow `json:"islands_debug_scope,omitempty"`
	Total             *int                   `json:"total,omitempty"`
	Passed            *int                   `json:"passed,omitempty"`
	Failed            *int                   `json:"failed,omitempty"`
	Cases             []smokeCaseReport      `json:"cases"`
}

const smokeReportArtifact = "tetra.release.v0_2_0.smoke-report.v1"

func sectionHeadingForTarget(target string) (string, error) {
	switch target {
	case "windows-x64":
		return "## Windows x64", nil
	case "macos-x64":
		return "## macOS x64", nil
	case "linux-x64":
		return "## Linux x64 (sanity)", nil
	default:
		return "", fmt.Errorf("unsupported target %q", target)
	}
}

func setHeaderField(md string, key string, value string) string {
	lines := strings.Split(md, "\n")
	prefix := key + ":"
	for i := range lines {
		if strings.HasPrefix(lines[i], prefix) {
			if value == "" {
				lines[i] = prefix
			} else {
				lines[i] = prefix + " " + value
			}
			break
		}
	}
	return strings.Join(lines, "\n")
}

func extractSection(md string, heading string) (before string, section string, after string, err error) {
	idx := strings.Index(md, heading+"\n")
	if idx == -1 {
		return "", "", "", fmt.Errorf("missing heading %q", heading)
	}
	before = md[:idx]
	rest := md[idx:]
	nextIdx := strings.Index(rest[len(heading)+1:], "\n## ")
	if nextIdx == -1 {
		return before, rest, "", nil
	}
	nextIdx += len(heading) + 1
	section = rest[:nextIdx]
	after = rest[nextIdx:]
	return before, section, after, nil
}

func setCheckboxState(section string, contains string, checked bool) (string, bool) {
	lines := strings.Split(section, "\n")
	changed := false
	want := "- [ ]"
	if checked {
		want = "- [x]"
	}
	for i := range lines {
		if !strings.Contains(lines[i], contains) {
			continue
		}
		if strings.Contains(lines[i], "- [ ]") {
			if checked {
				lines[i] = strings.Replace(lines[i], "- [ ]", want, 1)
				changed = true
			}
			continue
		}
		if strings.Contains(lines[i], "- [x]") {
			if !checked {
				lines[i] = strings.Replace(lines[i], "- [x]", want, 1)
				changed = true
			}
			continue
		}
	}
	if changed {
		return strings.Join(lines, "\n"), true
	}
	return section, false
}

type checkboxUpdate struct {
	Contains string
	Checked  bool
}

func applyToChecklist(path string, report *smokeReport, updates []checkboxUpdate) error {
	if report == nil {
		return fmt.Errorf("missing report")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	md := string(data)

	ts := report.Timestamp
	if ts == "" {
		ts = time.Now().UTC().Format(time.RFC3339)
	}
	date := strings.SplitN(ts, "T", 2)[0]

	md = setHeaderField(md, "Date", date)
	md = setHeaderField(md, "Target version", report.Target)
	md = setHeaderField(md, "Git HEAD", report.GitHead)
	md = setHeaderField(md, "Compiler version (compilerVersion)", report.Version)

	heading, err := sectionHeadingForTarget(report.Target)
	if err != nil {
		return err
	}
	before, section, after, err := extractSection(md, heading)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	for _, u := range updates {
		updated, ok := setCheckboxState(section, u.Contains, u.Checked)
		if ok {
			section = updated
		}
	}

	out := before + section + after
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(out), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func validateSmokeReportCounts(report *smokeReport) error {
	if report == nil {
		return fmt.Errorf("missing report")
	}
	if report.Total == nil && report.Passed == nil && report.Failed == nil {
		return nil
	}
	if report.Total == nil || report.Passed == nil || report.Failed == nil {
		return fmt.Errorf("smoke report counts incomplete: total, passed, and failed must appear together")
	}
	passed := 0
	for _, c := range report.Cases {
		if c.Pass {
			passed++
		}
	}
	total := len(report.Cases)
	failed := total - passed
	if *report.Total != total || *report.Passed != passed || *report.Failed != failed {
		return fmt.Errorf("smoke report counts mismatch: got total=%d passed=%d failed=%d, computed total=%d passed=%d failed=%d", *report.Total, *report.Passed, *report.Failed, total, passed, failed)
	}
	return nil
}

func parseSmokeReport(raw []byte) (*smokeReport, error) {
	return parseSmokeReportFormat(raw, "auto")
}

func parseSmokeReportFormat(raw []byte, format string) (*smokeReport, error) {
	var report smokeReport
	if err := reportdecode.DecodeStrictFormat(raw, format, &report); err != nil {
		return nil, err
	}
	return &report, nil
}

func validateSmokeReport(report *smokeReport) error {
	if err := validateSmokeReportCounts(report); err != nil {
		return err
	}
	if report.Total == nil && report.Passed == nil && report.Failed == nil {
		return nil
	}
	if report.Target == "" {
		return fmt.Errorf("smoke report missing target")
	}
	if !supportedTarget(report.Target) {
		return fmt.Errorf("smoke report unsupported target %s", report.Target)
	}
	if wantBuildOnly := ctarget.IsBuildOnlyTarget(report.Target); report.BuildOnly != wantBuildOnly {
		return fmt.Errorf("smoke report build_only = %v, want %v for target %s", report.BuildOnly, wantBuildOnly, report.Target)
	}
	if report.Host == "" {
		return fmt.Errorf("smoke report missing host")
	}
	if report.Version == "" {
		return fmt.Errorf("smoke report missing version")
	}
	if !strings.HasPrefix(report.Version, "v") {
		return fmt.Errorf("smoke report version must start with v")
	}
	seenNames := map[string]bool{}
	seenSources := map[string]bool{}
	for _, c := range report.Cases {
		if c.Name == "" {
			return fmt.Errorf("smoke report case missing name")
		}
		if seenNames[c.Name] {
			return fmt.Errorf("duplicate smoke report case %s", c.Name)
		}
		seenNames[c.Name] = true
		if c.SrcPath == "" {
			return fmt.Errorf("smoke report case %s missing src_path", c.Name)
		}
		if !strings.HasSuffix(c.SrcPath, ".tetra") {
			return fmt.Errorf("smoke report case %s src_path must be a .tetra file", c.Name)
		}
		if seenSources[c.SrcPath] {
			return fmt.Errorf("duplicate smoke report src_path %s", c.SrcPath)
		}
		seenSources[c.SrcPath] = true
		if c.ExpectedExit < 0 || c.ExpectedExit > 255 {
			return fmt.Errorf("smoke report case %s expected_exit = %d, want 0..255", c.Name, c.ExpectedExit)
		}
		if c.Unsupported {
			if c.ExpectedDiagnostic == "" {
				return fmt.Errorf("unsupported smoke report case %s missing expected_diagnostic", c.Name)
			}
			if c.Diagnostic == "" {
				return fmt.Errorf("unsupported smoke report case %s missing diagnostic", c.Name)
			}
			if !strings.Contains(c.Diagnostic, c.ExpectedDiagnostic) {
				return fmt.Errorf("unsupported smoke report case %s diagnostic = %q, want containing %q", c.Name, c.Diagnostic, c.ExpectedDiagnostic)
			}
			if c.Ran {
				return fmt.Errorf("unsupported smoke report case %s ran unexpectedly", c.Name)
			}
			if c.ActualExit != nil {
				return fmt.Errorf("unsupported smoke report case %s cannot include actual_exit", c.Name)
			}
			if c.OutPath != "" {
				return fmt.Errorf("unsupported smoke report case %s cannot include out_path", c.Name)
			}
		} else if c.ExpectedDiagnostic != "" || c.Diagnostic != "" {
			return fmt.Errorf("smoke report case %s has diagnostic metadata but is not marked unsupported", c.Name)
		}
		if c.Ran && c.ActualExit == nil {
			return fmt.Errorf("smoke report case %s ran without actual_exit", c.Name)
		}
		if report.BuildOnly && c.Ran && report.Runner == "" {
			return fmt.Errorf("smoke report case %s ran for build-only target %s", c.Name, report.Target)
		}
		if c.ActualExit != nil && (*c.ActualExit < 0 || *c.ActualExit > 255) {
			return fmt.Errorf("smoke report case %s actual_exit = %d, want 0..255", c.Name, *c.ActualExit)
		}
		if c.Ran && c.Pass && c.ActualExit != nil && *c.ActualExit != c.ExpectedExit {
			return fmt.Errorf("smoke report case %s passed with actual_exit %d, want %d", c.Name, *c.ActualExit, c.ExpectedExit)
		}
		if c.Pass && c.Error != "" {
			return fmt.Errorf("smoke report case %s passed with error text", c.Name)
		}
	}
	if err := validateRequiredSmokeCases(report.Target, seenNames); err != nil {
		return err
	}
	if report.IslandsDebug {
		if err := validateIslandsDebugTrapEvidence(report.Cases); err != nil {
			return err
		}
		if err := validateIslandsDebugScopeEvidence(report.Cases, report.IslandsDebugScope); err != nil {
			return err
		}
	}
	return nil
}

func validateIslandsDebugTrapEvidence(cases []smokeCaseReport) error {
	const trapName = "islands_overflow"
	for _, c := range cases {
		if c.Name != trapName {
			continue
		}
		if c.SrcPath != "examples/islands_overflow.tetra" {
			return fmt.Errorf("islands_debug trap case %s src_path = %q, want examples/islands_overflow.tetra", trapName, c.SrcPath)
		}
		if c.ExpectedExit == 0 {
			return fmt.Errorf("islands_debug trap case %s expected_exit must be non-zero", trapName)
		}
		if !c.Ran {
			return fmt.Errorf("islands_debug trap case %s did not run", trapName)
		}
		if c.ActualExit == nil {
			return fmt.Errorf("islands_debug trap case %s missing actual_exit", trapName)
		}
		if !c.Pass {
			return fmt.Errorf("islands_debug trap case %s did not pass", trapName)
		}
		if *c.ActualExit != c.ExpectedExit {
			return fmt.Errorf("islands_debug trap case %s actual_exit = %d, want %d", trapName, *c.ActualExit, c.ExpectedExit)
		}
		return nil
	}
	return fmt.Errorf("islands_debug smoke report missing runtime trap case %s", trapName)
}

type requiredIslandsDebugScope struct {
	status   string
	caseName string
	srcPath  string
}

var requiredIslandsDebugScopes = map[string]requiredIslandsDebugScope{
	"double_free":    {status: "static_only_nonclaim"},
	"use_after_free": {status: "static_only_nonclaim"},
	"stale_epoch":    {status: "static_only_nonclaim"},
	"wrong_island":   {status: "static_only_nonclaim"},
	"overflow_trap":  {status: "live_trap", caseName: "islands_overflow", srcPath: "examples/islands_overflow.tetra"},
}

var requiredIslandsDebugScopeOrder = []string{
	"double_free",
	"use_after_free",
	"stale_epoch",
	"wrong_island",
	"overflow_trap",
}

func validateIslandsDebugScopeEvidence(cases []smokeCaseReport, rows []islandsDebugScopeRow) error {
	byCase := make(map[string]smokeCaseReport, len(cases))
	for _, c := range cases {
		byCase[c.Name] = c
	}
	seen := make(map[string]islandsDebugScopeRow, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Name) == "" {
			return fmt.Errorf("islands_debug scope row missing name")
		}
		spec, ok := requiredIslandsDebugScopes[row.Name]
		if !ok {
			return fmt.Errorf("islands_debug scope row %q is not a supported scoped sanitizer evidence row", row.Name)
		}
		if _, exists := seen[row.Name]; exists {
			return fmt.Errorf("duplicate islands_debug scope row %s", row.Name)
		}
		if row.Status != spec.status {
			return fmt.Errorf("islands_debug scope row %s status = %q, want %q", row.Name, row.Status, spec.status)
		}
		if strings.TrimSpace(row.Evidence) == "" {
			return fmt.Errorf("islands_debug scope row %s missing evidence", row.Name)
		}
		if strings.TrimSpace(row.Reason) == "" {
			return fmt.Errorf("islands_debug scope row %s missing reason", row.Name)
		}
		switch spec.status {
		case "live_trap":
			if row.CaseName != spec.caseName || row.SrcPath != spec.srcPath {
				return fmt.Errorf("islands_debug live scope row %s must reference %s at %s", row.Name, spec.caseName, spec.srcPath)
			}
			c, ok := byCase[spec.caseName]
			if !ok {
				return fmt.Errorf("islands_debug live scope row %s references missing case %s", row.Name, spec.caseName)
			}
			if !c.Ran || c.ActualExit == nil || !c.Pass || *c.ActualExit == 0 {
				return fmt.Errorf("islands_debug live scope row %s requires ran/pass/non-zero trap evidence from %s", row.Name, spec.caseName)
			}
		case "static_only_nonclaim":
			reason := strings.ToLower(row.Reason)
			if !strings.Contains(reason, "static") || !strings.Contains(reason, "no live") || !strings.Contains(reason, "claimed") {
				return fmt.Errorf("islands_debug static-only scope row %s must explain static-only nonclaim status", row.Name)
			}
		}
		seen[row.Name] = row
	}
	for _, name := range requiredIslandsDebugScopeOrder {
		if _, ok := seen[name]; !ok {
			return fmt.Errorf("islands_debug scope missing %s evidence row", name)
		}
	}
	return nil
}

func validateRequiredSmokeCases(target string, seen map[string]bool) error {
	switch target {
	case "linux-x64", "windows-x64", "macos-x64":
		required := []string{
			"flow_hello",
			"flow_struct_smoke",
			"flow_islands_smoke",
			"flow_unsafe_cap_mem_smoke",
			"core_async_smoke",
			"core_capability_smoke",
			"core_collections_smoke",
			"core_component_smoke",
			"core_crypto_smoke",
			"core_filesystem_smoke",
			"core_io_smoke",
			"core_math_smoke",
			"core_memory_smoke",
			"core_networking_smoke",
			"core_serialization_smoke",
			"core_slices_smoke",
			"core_strings_smoke",
			"core_sync_smoke",
			"core_testing_smoke",
			"core_time_smoke",
		}
		for _, name := range required {
			if !seen[name] {
				return fmt.Errorf("smoke report missing required smoke case %s for target %s", name, target)
			}
		}
	case "wasm32-wasi", "wasm32-web":
		required := []string{"legacy_hello", "effects_io_smoke", "ui_web_smoke", "dogfood_wasi", "dogfood_web_ui"}
		for _, name := range required {
			if !seen[name] {
				return fmt.Errorf("smoke report missing required smoke profile for target %s", target)
			}
		}
	default:
		return nil
	}
	return nil
}

func supportedTarget(target string) bool {
	for _, triple := range ctarget.SupportedTriples() {
		if target == triple {
			return true
		}
	}
	for _, triple := range ctarget.BuildOnlyTriples() {
		if target == triple {
			return true
		}
	}
	return false
}

func main() {
	var reportPath string
	var islandsChecklist string
	var actorsChecklist string
	var validateOnly bool
	var format string

	flag.StringVar(&reportPath, "report", "", "path to tetra smoke structured report")
	flag.StringVar(&islandsChecklist, "islands-checklist", filepath.FromSlash("docs/checklists/islands_platform_smoke.md"), "path to islands platform checklist")
	flag.StringVar(&actorsChecklist, "actors-checklist", filepath.FromSlash("docs/checklists/actors_platform_smoke.md"), "path to actors platform checklist")
	flag.BoolVar(&validateOnly, "validate-only", false, "validate smoke report without updating checklists")
	flag.StringVar(&format, "format", "auto", "report format: auto, json, or toon")
	flag.Parse()

	if reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	report, err := parseSmokeReportFormat(raw, format)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateSmokeReport(report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if validateOnly {
		return
	}

	passed := make(map[string]bool, len(report.Cases))
	ran := make(map[string]bool, len(report.Cases))
	for _, c := range report.Cases {
		passed[c.Name] = c.Pass
		ran[c.Name] = c.Ran
	}

	var islandsUpdates []checkboxUpdate
	for _, name := range []string{"islands_hello", "islands_i32", "islands_overflow", "mmio_smoke", "cap_mem_smoke", "memset_smoke"} {
		if _, ok := passed[name]; !ok {
			continue
		}
		islandsUpdates = append(islandsUpdates, checkboxUpdate{
			Contains: fmt.Sprintf("examples/%s.tetra", name),
			Checked:  passed[name],
		})
		islandsUpdates = append(islandsUpdates, checkboxUpdate{
			Contains: fmt.Sprintf("./%s", name),
			Checked:  passed[name] && ran[name],
		})
	}
	if len(islandsUpdates) > 0 {
		if err := applyToChecklist(islandsChecklist, report, islandsUpdates); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	var actorsUpdates []checkboxUpdate
	if _, ok := passed["actors_pingpong"]; ok {
		actorsUpdates = append(actorsUpdates, checkboxUpdate{
			Contains: "examples/actors_pingpong.tetra",
			Checked:  passed["actors_pingpong"],
		})
		actorsUpdates = append(actorsUpdates, checkboxUpdate{
			Contains: "./actors_pingpong",
			Checked:  passed["actors_pingpong"] && ran["actors_pingpong"],
		})
	}
	if len(actorsUpdates) > 0 {
		if err := applyToChecklist(actorsChecklist, report, actorsUpdates); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
