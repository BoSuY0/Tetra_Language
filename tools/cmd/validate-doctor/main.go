package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type doctorReport struct {
	Status string        `json:"status"`
	Checks []doctorCheck `json:"checks"`
}

type doctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func main() {
	var path string
	flag.StringVar(&path, "report", "", "path to tetra doctor --format=json output")
	flag.Parse()
	if path == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateDoctorReport(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateDoctorReport(raw []byte) error {
	var report doctorReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return fmt.Errorf("invalid doctor JSON: %w", err)
	}
	if report.Status != "pass" {
		return fmt.Errorf("doctor status = %q, want pass", report.Status)
	}
	if len(report.Checks) == 0 {
		return fmt.Errorf("doctor checks must not be empty")
	}
	seen := map[string]bool{}
	required := map[string]bool{
		"version":                                false,
		"supported targets":                      false,
		"build-only targets":                     false,
		"planned targets":                        false,
		"repo root":                              false,
		"__rt/actors_sysv.tetra":                 false,
		"__rt/actors_win64.tetra":                false,
		"compiler/selfhostrt/actors_sysv.tetra":  false,
		"compiler/selfhostrt/actors_win64.tetra": false,
		"examples/flow_hello.tetra":              false,
		"docs/generated/manifest.json":           false,
		"docs manifest version":                  false,
		"docs manifest surface":                  false,
		"smoke sources":                          false,
		"runtime exports":                        false,
	}
	for _, check := range report.Checks {
		if check.Name == "" {
			return fmt.Errorf("doctor check missing name")
		}
		if seen[check.Name] {
			return fmt.Errorf("duplicate doctor check %s", check.Name)
		}
		seen[check.Name] = true
		if check.Status != "pass" {
			return fmt.Errorf("doctor check %s status = %q, want pass", check.Name, check.Status)
		}
		if _, ok := required[check.Name]; ok {
			required[check.Name] = true
		}
	}
	for name, ok := range required {
		if !ok {
			return fmt.Errorf("doctor missing required check %s", name)
		}
	}
	return nil
}
