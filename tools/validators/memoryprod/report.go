package memoryprod

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const SchemaV1 = "tetra.memory.production.v1"

type Report struct {
	Schema    string           `json:"schema"`
	Status    string           `json:"status"`
	Target    string           `json:"target"`
	Host      string           `json:"host"`
	Runtime   string           `json:"runtime"`
	Source    string           `json:"source"`
	Processes []ProcessReport  `json:"processes"`
	Contracts []ContractReport `json:"contracts"`
	Cases     []CaseReport     `json:"cases"`
	Audit     []AuditReport    `json:"audit"`
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
	issues = append(issues, validateContracts(report.Contracts)...)
	issues = append(issues, validateCases(report.Cases)...)
	issues = append(issues, validateAudit(report.Audit)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func rejectPaperEvidence(raw []byte) []string {
	lower := strings.ToLower(string(raw))
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
	for _, marker := range forbidden {
		if strings.Contains(lower, marker) {
			issues = append(issues, fmt.Sprintf("report contains forbidden non-production evidence marker %q", strings.Trim(marker, " /\"")))
		}
	}
	return issues
}

func validateProcesses(processes []ProcessReport) []string {
	var issues []string
	if len(processes) < 3 {
		issues = append(issues, fmt.Sprintf("process evidence has %d entries, want build, app, and stress processes", len(processes)))
	}
	seenBuild := false
	seenApp := false
	seenStress := false
	names := map[string]bool{}
	for _, p := range processes {
		if strings.TrimSpace(p.Name) == "" {
			issues = append(issues, "process name is required")
		} else if names[p.Name] {
			issues = append(issues, fmt.Sprintf("duplicate process %s", p.Name))
		}
		names[p.Name] = true
		switch p.Kind {
		case "build":
			seenBuild = true
		case "app":
			seenApp = true
		case "stress":
			seenStress = true
		default:
			issues = append(issues, fmt.Sprintf("process %s kind is %q, want build, app, or stress", p.Name, p.Kind))
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
	return issues
}

func validateContracts(contracts []ContractReport) []string {
	required := map[string]bool{
		"allocator runtime model":         false,
		"allocator failure semantics":     false,
		"ownership escape model":          false,
		"unsafe cap.mem raw memory rules": false,
		"runtime bounds diagnostics":      false,
		"actor task transfer rules":       false,
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
		"stress/fuzz evidence":                                            false,
		"use-after-free, double-free, borrow escape, and aliasing safety": false,
		"actor/task transfer safety":                                      false,
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
