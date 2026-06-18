package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

type projectDepsReport struct {
	Status       string                    `json:"status,omitempty"`
	Root         string                    `json:"root,omitempty"`
	CapsulePath  string                    `json:"capsule_path,omitempty"`
	Dependencies []projectDependencyReport `json:"dependencies"`
}

type projectDependencyReport struct {
	ID           string `json:"id"`
	Version      string `json:"version"`
	Path         string `json:"path,omitempty"`
	ResolvedPath string `json:"resolved_path,omitempty"`
	Status       string `json:"status"`
	Detail       string `json:"detail,omitempty"`
}

var semverPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

func main() {
	var path string
	flag.StringVar(
		&path,
		"report",
		"",
		"path to tetra project deps list/check --format=json output",
	)
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
	if err := validateProjectDepsReport(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateProjectDepsReport(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := decodeStrictJSON(raw, &fields); err != nil {
		return fmt.Errorf("invalid project deps JSON: %w", err)
	}
	if _, ok := fields["dependencies"]; !ok {
		return fmt.Errorf("project deps missing dependencies")
	}

	var report projectDepsReport
	if err := decodeStrictJSON(raw, &report); err != nil {
		return fmt.Errorf("invalid project deps JSON: %w", err)
	}
	if strings.TrimSpace(report.Root) == "" {
		return fmt.Errorf("project deps root is required")
	}
	if strings.TrimSpace(report.CapsulePath) == "" {
		return fmt.Errorf("project deps capsule_path is required")
	}
	if report.Dependencies == nil {
		return fmt.Errorf("project deps dependencies is required")
	}
	if report.Status != "" && report.Status != "pass" && report.Status != "fail" {
		return fmt.Errorf("project deps status = %q, want pass or fail", report.Status)
	}
	hasIssue := false
	seen := map[string]bool{}
	for i, dep := range report.Dependencies {
		issue, err := validateDependency(i, dep)
		if err != nil {
			return err
		}
		hasIssue = hasIssue || issue
		key := dep.ID + "\x00" + dep.Version + "\x00" + dep.Path
		if dep.ID != "" && seen[key] {
			return fmt.Errorf(
				"project deps dependency %s %s %s is duplicated",
				dep.ID,
				dep.Version,
				dep.Path,
			)
		}
		if dep.ID != "" {
			seen[key] = true
		}
	}
	if report.Status == "pass" && hasIssue {
		return fmt.Errorf("project deps status pass requires all dependencies to be ok")
	}
	if report.Status == "fail" && !hasIssue {
		return fmt.Errorf("project deps status fail requires at least one dependency issue")
	}
	return nil
}

func validateDependency(index int, dep projectDependencyReport) (bool, error) {
	allowedStatus := map[string]bool{
		"ok":       true,
		"missing":  true,
		"invalid":  true,
		"mismatch": true,
		"fail":     true,
	}
	if !allowedStatus[dep.Status] {
		return false, fmt.Errorf("project deps dependency[%d] invalid status %q", index, dep.Status)
	}
	if dep.ID == "" && dep.Status != "fail" {
		return false, fmt.Errorf("project deps dependency[%d] missing id", index)
	}
	if dep.ID != "" && !strings.HasPrefix(dep.ID, "tetra://") {
		return false, fmt.Errorf(
			"project deps dependency[%d] id %q must start with tetra://",
			index,
			dep.ID,
		)
	}
	if dep.Version == "" && dep.Status != "fail" {
		return false, fmt.Errorf("project deps dependency[%d] missing version", index)
	}
	if dep.Version != "" && !semverPattern.MatchString(dep.Version) {
		return false, fmt.Errorf(
			"project deps dependency[%d] version %q must use semver x.y.z",
			index,
			dep.Version,
		)
	}
	if dep.Status == "ok" {
		if strings.TrimSpace(dep.Path) == "" {
			return false, fmt.Errorf("project deps dependency[%d] ok status requires path", index)
		}
		if strings.TrimSpace(dep.ResolvedPath) == "" {
			return false, fmt.Errorf(
				"project deps dependency[%d] ok status requires resolved_path",
				index,
			)
		}
		if dep.Detail != "" {
			return false, fmt.Errorf(
				"project deps dependency[%d] ok status must not include detail",
				index,
			)
		}
		return false, nil
	}
	if strings.TrimSpace(dep.Detail) == "" {
		return false, fmt.Errorf(
			"project deps dependency[%d] %s status requires detail",
			index,
			dep.Status,
		)
	}
	return true, nil
}

func decodeStrictJSON(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}
