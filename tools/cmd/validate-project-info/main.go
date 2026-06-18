package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

type projectInfoReport struct {
	Found              bool           `json:"found"`
	Root               string         `json:"root,omitempty"`
	CapsulePath        string         `json:"capsule_path,omitempty"`
	LockPath           string         `json:"lock_path,omitempty"`
	EntryPath          string         `json:"entry_path,omitempty"`
	SourceRoots        []string       `json:"source_roots,omitempty"`
	Targets            []string       `json:"targets,omitempty"`
	DependencyRoots    []string       `json:"dependency_roots,omitempty"`
	ArtifactCounts     map[string]int `json:"artifact_counts,omitempty"`
	DependencyCapsules []string       `json:"dependency_capsules,omitempty"`
}

func main() {
	var path string
	flag.StringVar(&path, "report", "", "path to tetra project info --format=json output")
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
	if err := validateProjectInfoReport(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateProjectInfoReport(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := decodeStrictJSON(raw, &fields); err != nil {
		return fmt.Errorf("invalid project info JSON: %w", err)
	}
	if _, ok := fields["found"]; !ok {
		return fmt.Errorf("project info missing found")
	}

	var report projectInfoReport
	if err := decodeStrictJSON(raw, &report); err != nil {
		return fmt.Errorf("invalid project info JSON: %w", err)
	}
	if !report.Found {
		return validateProjectInfoNotFound(report)
	}
	if strings.TrimSpace(report.Root) == "" {
		return fmt.Errorf("project info root is required when found is true")
	}
	if strings.TrimSpace(report.CapsulePath) == "" {
		return fmt.Errorf("project info capsule_path is required when found is true")
	}
	if strings.TrimSpace(report.EntryPath) == "" {
		return fmt.Errorf("project info entry_path is required when found is true")
	}
	if report.SourceRoots == nil {
		return fmt.Errorf("project info source_roots is required when found is true")
	}
	if report.Targets == nil {
		return fmt.Errorf("project info targets is required when found is true")
	}
	if err := validateStringList("source_roots", report.SourceRoots, false); err != nil {
		return err
	}
	if err := validateStringList("targets", report.Targets, true); err != nil {
		return err
	}
	if err := validateStringList("dependency_roots", report.DependencyRoots, true); err != nil {
		return err
	}
	if err := validateStringList("dependency_capsules", report.DependencyCapsules, true); err != nil {
		return err
	}
	for kind, count := range report.ArtifactCounts {
		if strings.TrimSpace(kind) == "" {
			return fmt.Errorf("project info artifact_counts has empty kind")
		}
		if count < 0 {
			return fmt.Errorf(
				"project info artifact_counts[%s] = %d, want non-negative",
				kind,
				count,
			)
		}
	}
	return nil
}

func validateProjectInfoNotFound(report projectInfoReport) error {
	if report.Root != "" || report.CapsulePath != "" || report.LockPath != "" ||
		report.EntryPath != "" {
		return fmt.Errorf("project info not found report must not include project paths")
	}
	if len(report.SourceRoots) != 0 || len(report.Targets) != 0 ||
		len(report.DependencyRoots) != 0 ||
		len(report.DependencyCapsules) != 0 {
		return fmt.Errorf("project info not found report must not include project lists")
	}
	if len(report.ArtifactCounts) != 0 {
		return fmt.Errorf("project info not found report must not include artifact_counts")
	}
	return nil
}

func validateStringList(name string, values []string, allowEmpty bool) error {
	if !allowEmpty && len(values) == 0 {
		return fmt.Errorf("project info %s must not be empty", name)
	}
	seen := map[string]bool{}
	for i, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("project info %s[%d] is empty", name, i)
		}
		if seen[value] {
			return fmt.Errorf("project info %s value %q is duplicated", name, value)
		}
		seen[value] = true
	}
	return nil
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
