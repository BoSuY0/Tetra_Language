package memoryprod

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"strings"
)

const SchemaV1 = "tetra.memory.production.v1"

type Report struct {
	Schema     string            `json:"schema"`
	Status     string            `json:"status"`
	GitHead    string            `json:"git_head,omitempty"`
	Target     string            `json:"target"`
	Host       string            `json:"host"`
	Runtime    string            `json:"runtime"`
	Source     string            `json:"source"`
	Processes  []ProcessReport   `json:"processes"`
	Benchmarks []BenchmarkReport `json:"benchmarks"`
	Contracts  []ContractReport  `json:"contracts"`
	Cases      []CaseReport      `json:"cases"`
	Audit      []AuditReport     `json:"audit"`
}

type ProcessReport struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Ran      bool   `json:"ran"`
	Pass     bool   `json:"pass"`
	ExitCode *int   `json:"exit_code,omitempty"`
}

type ContractReport struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Evidence string `json:"evidence"`
}

type BenchmarkReport struct {
	Name                string  `json:"name"`
	Kind                string  `json:"kind"`
	Metric              string  `json:"metric"`
	Unit                string  `json:"unit"`
	EvidenceClass       string  `json:"evidence_class"`
	Method              string  `json:"method"`
	MeasurementArtifact string  `json:"measurement_artifact,omitempty"`
	BaselineValue       int     `json:"baseline_value"`
	MeasuredValue       int     `json:"measured_value"`
	ImprovementRatio    float64 `json:"improvement_ratio"`
	Evidence            string  `json:"evidence"`
	Ran                 bool    `json:"ran"`
	Pass                bool    `json:"pass"`
}

type CaseReport struct {
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	Ran           bool   `json:"ran"`
	Pass          bool   `json:"pass"`
	ExpectedError string `json:"expected_error,omitempty"`
	Error         string `json:"error,omitempty"`
}

type AuditReport struct {
	Requirement string `json:"requirement"`
	Artifact    string `json:"artifact"`
	Evidence    string `json:"evidence"`
	Result      string `json:"result"`
}

func ValidateReport(raw []byte) error {
	var report Report
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	issues = append(issues, rejectPaperEvidence(raw)...)
	issues = append(issues, validateReportFields(report)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func ValidateReportObject(report Report) error {
	var issues []string
	issues = append(issues, rejectPaperEvidenceStrings(reportStringFields(report))...)
	issues = append(issues, validateReportFields(report)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateReportFields(report Report) []string {
	var issues []string
	if report.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, SchemaV1))
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if report.Target != "linux-x64" {
		issues = append(issues, fmt.Sprintf("target is %q, want linux-x64", report.Target))
	}
	if report.Host != "linux-x64" {
		issues = append(issues, fmt.Sprintf("host is %q, want linux-x64", report.Host))
	}
	if report.Runtime != "memory-linux-x64" {
		issues = append(issues, fmt.Sprintf("runtime is %q, want memory-linux-x64", report.Runtime))
	}
	if strings.TrimSpace(report.Source) == "" {
		issues = append(issues, "source is required")
	}
	issues = append(issues, validateProcesses(report.Processes)...)
	issues = append(issues, validateBenchmarks(report.Benchmarks)...)
	issues = append(issues, validateContracts(report.Contracts)...)
	issues = append(issues, validateCases(report.Cases)...)
	issues = append(issues, validateAudit(report.Audit)...)
	return issues
}

func rejectPaperEvidence(raw []byte) []string {
	return rejectPaperEvidenceStrings([]string{string(raw)})
}

func rejectPaperEvidenceStrings(values []string) []string {
	forbidden := []string{
		"metadata-only",
		"build-only",
		"docs-only",
		"sidecar-only",
		" fake",
		"fake/",
		"\"fake\"",
		" mock",
		"mock/",
		"\"mock\"",
		"placeholder",
	}
	var issues []string
	for _, value := range values {
		lower := strings.ToLower(value)
		for _, marker := range forbidden {
			if strings.Contains(lower, marker) {
				issues = append(issues, fmt.Sprintf("report contains forbidden non-production evidence marker %q", strings.Trim(marker, " /\"")))
			}
		}
	}
	return issues
}

func reportStringFields(report Report) []string {
	var out []string
	collectStringFields(reflect.ValueOf(report), &out)
	return out
}

func collectStringFields(value reflect.Value, out *[]string) {
	if !value.IsValid() {
		return
	}
	if value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return
		}
		collectStringFields(value.Elem(), out)
		return
	}
	switch value.Kind() {
	case reflect.String:
		*out = append(*out, value.String())
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			collectStringFields(value.Field(i), out)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			collectStringFields(value.Index(i), out)
		}
	case reflect.Map:
		for _, key := range value.MapKeys() {
			collectStringFields(key, out)
			collectStringFields(value.MapIndex(key), out)
		}
	}
}

func validateProcesses(processes []ProcessReport) []string {
	requiredProcesses := map[string][]string{
		"actornet close-without-cancel leak coverage": {
			"./cli/internal/actornet",
			"TestBrokerCloseWithoutCancelStopsServeWatcher",
		},
		"compiler resource finalization diagnostics": {
			"./compiler/tests/runtime",
			"TestTaskHandleFinalization",
			"TestTaskGroupFinalization",
			"TestIslandFinalization",
		},
	}
	var issues []string
	if len(processes) < 3 {
		issues = append(issues, fmt.Sprintf("process evidence has %d entries, want build, app, and stress processes", len(processes)))
	}
	seenBuild := false
	seenApp := false
	seenStress := false
	seenRequired := map[string]bool{}
	names := map[string]bool{}
	for _, p := range processes {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			issues = append(issues, "process name is required")
		} else if names[name] {
			issues = append(issues, fmt.Sprintf("duplicate process %s", name))
		}
		names[name] = true
		if requiredMarkers, ok := requiredProcesses[name]; ok {
			seenRequired[name] = true
			for _, marker := range requiredMarkers {
				if !strings.Contains(p.Path, marker) {
					issues = append(issues, fmt.Sprintf("process %s path must mention %s", name, marker))
				}
			}
		}
		switch p.Kind {
		case "build":
			seenBuild = true
		case "app":
			seenApp = true
		case "stress":
			seenStress = true
		case "benchmark":
		default:
			issues = append(issues, fmt.Sprintf("process %s kind is %q, want build, app, stress, or benchmark", p.Name, p.Kind))
		}
		if strings.TrimSpace(p.Path) == "" {
			issues = append(issues, fmt.Sprintf("process %s path is required", p.Name))
		}
		if !p.Ran {
			issues = append(issues, fmt.Sprintf("process %s did not run", p.Name))
		}
		if !p.Pass {
			issues = append(issues, fmt.Sprintf("process %s did not pass", p.Name))
		}
		if p.ExitCode == nil {
			issues = append(issues, fmt.Sprintf("process %s missing exit_code", p.Name))
		} else if *p.ExitCode != 0 {
			issues = append(issues, fmt.Sprintf("process %s exit_code = %d, want 0", p.Name, *p.ExitCode))
		}
	}
	if !seenBuild {
		issues = append(issues, "process evidence missing build process")
	}
	if !seenApp {
		issues = append(issues, "process evidence missing executable app process")
	}
	if !seenStress {
		issues = append(issues, "process evidence missing memory stress process")
	}
	for name := range requiredProcesses {
		if !seenRequired[name] {
			issues = append(issues, fmt.Sprintf("missing required memory process %q", name))
		}
	}
	return issues
}

func validateBenchmarks(benchmarks []BenchmarkReport) []string {
	required := map[string]bool{
		"small heap allocation syscall reduction": false,
	}
	var issues []string
	if len(benchmarks) == 0 {
		issues = append(issues, "benchmark evidence is required")
	}
	seen := map[string]bool{}
	for _, b := range benchmarks {
		name := strings.TrimSpace(b.Name)
		if name == "" {
			issues = append(issues, "benchmark name is required")
			continue
		}
		if seen[name] {
			issues = append(issues, fmt.Sprintf("duplicate benchmark %s", name))
		}
		seen[name] = true
		if _, ok := required[name]; ok {
			required[name] = true
		}
		if b.Kind != "allocator" {
			issues = append(issues, fmt.Sprintf("benchmark %s kind is %q, want allocator", name, b.Kind))
		}
		for _, issue := range forbiddenPerformanceClaimIssues("benchmark "+name, b.Name, b.Kind, b.Metric, b.Unit, b.Evidence) {
			issues = append(issues, issue)
		}
		if strings.TrimSpace(b.Metric) == "" {
			issues = append(issues, fmt.Sprintf("benchmark %s metric is required", name))
		}
		if strings.TrimSpace(b.Unit) == "" {
			issues = append(issues, fmt.Sprintf("benchmark %s unit is required", name))
		}
		issues = append(issues, validateBenchmarkEvidenceClassification(name, b)...)
		if !b.Ran {
			issues = append(issues, fmt.Sprintf("benchmark %s did not run", name))
		}
		if !b.Pass {
			issues = append(issues, fmt.Sprintf("benchmark %s did not pass", name))
		}
		if b.BaselineValue <= 0 {
			issues = append(issues, fmt.Sprintf("benchmark %s baseline_value = %d, want positive", name, b.BaselineValue))
		}
		if b.MeasuredValue <= 0 {
			issues = append(issues, fmt.Sprintf("benchmark %s measured_value = %d, want positive", name, b.MeasuredValue))
		}
		if b.BaselineValue > 0 && b.MeasuredValue > 0 && b.MeasuredValue >= b.BaselineValue {
			issues = append(issues, fmt.Sprintf("benchmark %s measured_value = %d, want less than baseline_value %d", name, b.MeasuredValue, b.BaselineValue))
		}
		if b.ImprovementRatio <= 1 {
			issues = append(issues, fmt.Sprintf("benchmark %s improvement_ratio = %.3f, want > 1", name, b.ImprovementRatio))
		}
		evidence := strings.TrimSpace(b.Evidence)
		if evidence == "" {
			issues = append(issues, fmt.Sprintf("benchmark %s evidence is required", name))
		}
		if name == "small heap allocation syscall reduction" {
			if b.EvidenceClass != "allocation_report_estimate" {
				issues = append(issues, fmt.Sprintf("benchmark %s evidence_class is %q, want allocation_report_estimate", name, b.EvidenceClass))
			}
			if b.Metric != "estimated_os_syscalls" {
				issues = append(issues, fmt.Sprintf("benchmark %s metric is %q, want estimated_os_syscalls", name, b.Metric))
			}
			if b.Unit != "syscalls" {
				issues = append(issues, fmt.Sprintf("benchmark %s unit is %q, want syscalls", name, b.Unit))
			}
			for _, marker := range []string{"per_core_small_heap"} {
				if !strings.Contains(evidence, marker) {
					issues = append(issues, fmt.Sprintf("benchmark %s evidence must mention %s", name, marker))
				}
			}
			for _, marker := range []string{"allocation report schema v2", "64KiB chunk refill"} {
				if !strings.Contains(evidence, marker) {
					issues = append(issues, fmt.Sprintf("benchmark %s evidence must mention scoped local artifact marker %s", name, marker))
				}
			}
			for _, marker := range []string{"same_core_same_size_class_free_list", "free-list", "free list", "reuse policy"} {
				if strings.Contains(strings.ToLower(evidence), marker) {
					issues = append(issues, fmt.Sprintf("benchmark %s contains runtime free-list wording %q without runtime_measured evidence", name, marker))
				}
			}
		}
	}
	for name, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("missing required benchmark %q", name))
		}
	}
	return issues
}

func validateBenchmarkEvidenceClassification(name string, b BenchmarkReport) []string {
	var issues []string
	evidenceClass := strings.TrimSpace(b.EvidenceClass)
	method := strings.TrimSpace(b.Method)
	measurementArtifact := strings.TrimSpace(b.MeasurementArtifact)
	if evidenceClass == "" {
		issues = append(issues, fmt.Sprintf("benchmark %s evidence_class is required", name))
	}
	if method == "" {
		issues = append(issues, fmt.Sprintf("benchmark %s method is required", name))
	}
	switch evidenceClass {
	case "allocation_report_estimate":
		if method != "allocation_report_summary" {
			issues = append(issues, fmt.Sprintf("benchmark %s method is %q, want allocation_report_summary for allocation_report_estimate", name, b.Method))
		}
		if measurementArtifact != "" {
			issues = append(issues, fmt.Sprintf("benchmark %s measurement_artifact must be empty for allocation_report_estimate", name))
		}
		if !strings.HasPrefix(b.Metric, "estimated_") {
			issues = append(issues, fmt.Sprintf("benchmark %s metric %q must be explicitly estimated for allocation_report_estimate", name, b.Metric))
		}
	case "runtime_measured":
		if !isRuntimeMeasuredBenchmarkMethod(method) {
			issues = append(issues, fmt.Sprintf("benchmark %s method is %q, want one of time_v, strace, MemStats, pprof for runtime_measured", name, b.Method))
		}
		if measurementArtifact == "" {
			issues = append(issues, fmt.Sprintf("benchmark %s measurement_artifact is required for runtime_measured", name))
		} else if !isSafeRelativeArtifactPath(measurementArtifact) {
			issues = append(issues, fmt.Sprintf("benchmark %s measurement_artifact %q is not a safe relative artifact path", name, b.MeasurementArtifact))
		}
		if strings.HasPrefix(b.Metric, "estimated_") {
			issues = append(issues, fmt.Sprintf("benchmark %s metric %q must not be estimated for runtime_measured", name, b.Metric))
		}
	case "":
	default:
		issues = append(issues, fmt.Sprintf("benchmark %s evidence_class is %q, want allocation_report_estimate or runtime_measured", name, b.EvidenceClass))
	}
	return issues
}

func isRuntimeMeasuredBenchmarkMethod(method string) bool {
	switch method {
	case "time_v", "strace", "MemStats", "pprof":
		return true
	default:
		return false
	}
}

func isSafeRelativeArtifactPath(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" || filepath.IsAbs(path) || strings.Contains(path, "\\") {
		return false
	}
	clean := filepath.Clean(path)
	return clean == path && clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}

func forbiddenPerformanceClaimIssues(label string, fields ...string) []string {
	var issues []string
	for _, field := range fields {
		lower := strings.ToLower(field)
		if explicitPerformanceNonClaimContext(lower) {
			continue
		}
		for _, claim := range []struct {
			label   string
			markers []string
		}{
			{label: "fastest language", markers: []string{"fastest language", "fastest-language"}},
			{label: "official benchmark", markers: []string{"official benchmark", "official techempower"}},
			{label: "target parity", markers: []string{"target parity", "target-parity"}},
			{label: "zero-cost performance", markers: []string{"zero-cost performance", "zero cost performance"}},
		} {
			for _, marker := range claim.markers {
				if strings.Contains(lower, marker) {
					issues = append(issues, fmt.Sprintf("%s contains forbidden %s claim", label, claim.label))
					break
				}
			}
		}
	}
	return issues
}

func explicitPerformanceNonClaimContext(lower string) bool {
	for _, marker := range []string{
		"does not claim",
		"do not claim",
		"does not prove",
		"do not prove",
		"does not promote",
		"do not promote",
		"must not use",
		"not an official",
		"not a fastest",
		"not fastest",
		"not target parity",
		"not a benchmark",
		"not a full",
		"not full",
		"not a runtime measurement",
		"no official",
		"no fastest",
		"no target parity",
		"no full",
		"makes no",
		"without an official",
		"without official",
		"forbid",
		"forbidden",
		"non-claim",
		"nonclaim",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func validateContracts(contracts []ContractReport) []string {
	required := map[string]bool{
		"allocator runtime model":                    false,
		"allocator failure semantics":                false,
		"ownership escape model":                     false,
		"unsafe cap.mem raw memory rules":            false,
		"runtime bounds diagnostics":                 false,
		"raw pointer bounds metadata":                false,
		"host resource leak and finalization checks": false,
		"actor task transfer rules":                  false,
	}
	var issues []string
	for _, c := range contracts {
		name := strings.TrimSpace(c.Name)
		if name == "" {
			issues = append(issues, "contract name is required")
			continue
		}
		if _, ok := required[name]; ok {
			required[name] = true
		}
		if c.Status != "pass" {
			issues = append(issues, fmt.Sprintf("contract %s status is %q, want pass", name, c.Status))
		}
		if strings.TrimSpace(c.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("contract %s evidence is required", name))
		}
	}
	for name, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("missing required memory contract %q", name))
		}
	}
	return issues
}

func validateCases(cases []CaseReport) []string {
	required := map[string]bool{
		"allocator alloc/free lifecycle":                        false,
		"allocator failure semantics":                           false,
		"allocator invalid size precondition":                   false,
		"cap.mem unsafe boundary":                               false,
		"memcpy/memset capability path":                         false,
		"runtime bounds check":                                  false,
		"raw ptr_add negative offset bounds":                    false,
		"raw ptr_add allocation upper bound":                    false,
		"raw allocation-base i32 access width":                  false,
		"raw allocation-base ptr access width":                  false,
		"raw slice negative length":                             false,
		"raw slice i32 length byte overflow":                    false,
		"raw pointer bounds metadata report":                    false,
		"memcpy/memset negative length":                         false,
		"reject use-after-free":                                 false,
		"reject double-free":                                    false,
		"reject borrow escape":                                  false,
		"reject aliasing violation":                             false,
		"callable mutable capture heap escape":                  false,
		"reject actor task transfer violation":                  false,
		"heap closure handle coverage":                          false,
		"slice struct borrow escape coverage":                   false,
		"function-typed slice aggregate borrow escape coverage": false,
		"actornet broker close-without-cancel leak smoke":       false,
		"compiler resource finalization diagnostics":            false,
		"real memory examples":                                  false,
		"stress allocator reuse":                                false,
		"deterministic memcpy/memset fuzz":                      false,
	}
	var issues []string
	seenPositive := false
	seenNegative := false
	seenStress := false
	for _, c := range cases {
		name := strings.TrimSpace(c.Name)
		if name == "" {
			issues = append(issues, "case name is required")
			continue
		}
		if _, ok := required[name]; ok {
			required[name] = true
		}
		switch c.Kind {
		case "positive":
			seenPositive = true
		case "negative":
			seenNegative = true
			if strings.TrimSpace(c.ExpectedError) == "" {
				issues = append(issues, fmt.Sprintf("negative case %s expected_error is required", name))
			}
		case "stress":
			seenStress = true
		default:
			issues = append(issues, fmt.Sprintf("case %s kind is %q, want positive, negative, or stress", name, c.Kind))
		}
		if !c.Ran {
			issues = append(issues, fmt.Sprintf("case %s did not run", name))
		}
		if !c.Pass {
			issues = append(issues, fmt.Sprintf("case %s did not pass", name))
		}
		if strings.TrimSpace(c.Error) != "" {
			issues = append(issues, fmt.Sprintf("case %s has unexpected error: %s", name, c.Error))
		}
	}
	if !seenPositive {
		issues = append(issues, "case evidence missing positive memory case")
	}
	if !seenNegative {
		issues = append(issues, "case evidence missing negative memory safety case")
	}
	if !seenStress {
		issues = append(issues, "case evidence missing memory stress case")
	}
	for name, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("missing required memory case %q", name))
		}
	}
	return issues
}

func validateAudit(audit []AuditReport) []string {
	required := map[string]bool{
		"stable allocator/runtime memory model":                           false,
		"ownership/borrow/consume escape model":                           false,
		"heap, slices, structs, and closures memory coverage":             false,
		"unsafe/cap.mem/raw memory/memcpy/memset rules":                   false,
		"runtime bounds checks and diagnostics":                           false,
		"raw pointer bounds metadata":                                     false,
		"stress/fuzz evidence":                                            false,
		"allocator benchmark evidence classification":                     false,
		"use-after-free, double-free, borrow escape, and aliasing safety": false,
		"actor/task transfer safety":                                      false,
		"leak/resource finalization evidence":                             false,
		"real memory examples":                                            false,
		"safe memory documentation":                                       false,
		"release-gate entrypoint":                                         false,
	}
	var issues []string
	if len(audit) == 0 {
		issues = append(issues, "completion audit is required")
	}
	seen := map[string]bool{}
	for _, row := range audit {
		requirement := strings.TrimSpace(row.Requirement)
		if requirement == "" {
			issues = append(issues, "completion audit row requirement is required")
			continue
		}
		if seen[requirement] {
			issues = append(issues, fmt.Sprintf("duplicate completion audit requirement %q", requirement))
		}
		seen[requirement] = true
		if _, ok := required[requirement]; ok {
			required[requirement] = true
		}
		if strings.TrimSpace(row.Artifact) == "" {
			issues = append(issues, fmt.Sprintf("completion audit requirement %q artifact is required", requirement))
		}
		if strings.TrimSpace(row.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("completion audit requirement %q evidence is required", requirement))
		}
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(row.Result)), "pass") {
			issues = append(issues, fmt.Sprintf("completion audit requirement %q result is %q, want pass", requirement, row.Result))
		}
	}
	for requirement, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("completion audit missing required requirement %q", requirement))
		}
	}
	return issues
}

func decodeStrict(raw []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("trailing JSON content")
	}
	return nil
}
