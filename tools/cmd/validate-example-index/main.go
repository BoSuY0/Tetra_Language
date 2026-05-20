package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type smokeList struct {
	Target           string          `json:"target"`
	BuildOnly        bool            `json:"build_only"`
	RunSupported     bool            `json:"run_supported"`
	IslandsDebug     bool            `json:"islands_debug"`
	Total            int             `json:"total"`
	Cases            []smokeCase     `json:"cases"`
	ExcludedExamples []smokeExcluded `json:"excluded_examples,omitempty"`
}

type smokeCase struct {
	Name         string `json:"name"`
	SrcPath      string `json:"src_path"`
	TargetGroup  string `json:"target_group"`
	ExpectedExit int    `json:"expected_exit"`
}

type smokeExcluded struct {
	SrcPath string `json:"src_path"`
	Reason  string `json:"reason"`
}

type exampleIndexEntry struct {
	Purpose  string
	Target   string
	Expected string
}

const exampleIndexArtifact = "tetra.release.v0_2_0.examples-index.v1"

func main() {
	os.Exit(runValidateExampleIndex(os.Args[1:], os.Stdout, os.Stderr))
}

func runValidateExampleIndex(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("validate-example-index", flag.ContinueOnError)
	flags.SetOutput(stderr)
	smokeListPath := flags.String("smoke-list", "", "path to tetra smoke --list --format=json output")
	indexPath := flags.String("index", "docs/user/examples_index.md", "path to examples index markdown")
	docsPath := flags.String("docs", "", "path to examples index markdown for docs-only validation")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	_ = stdout
	if *docsPath != "" {
		rawIndex, err := os.ReadFile(*docsPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if err := validateExampleDocs(string(rawIndex)); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}

	if *smokeListPath == "" {
		fmt.Fprintln(stderr, "error: --smoke-list is required")
		return 2
	}
	rawSmoke, err := os.ReadFile(*smokeListPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	rawIndex, err := os.ReadFile(*indexPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := validateExampleIndex(rawSmoke, string(rawIndex)); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func validateExampleIndex(rawSmoke []byte, markdown string) error {
	var list smokeList
	dec := json.NewDecoder(bytes.NewReader(rawSmoke))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&list); err != nil {
		return fmt.Errorf("invalid smoke list JSON: %w", err)
	}
	if list.Total != 0 && list.Total != len(list.Cases) {
		return fmt.Errorf("smoke list total = %d, want %d", list.Total, len(list.Cases))
	}
	entries, err := parseExampleIndex(markdown)
	if err != nil {
		return err
	}
	if len(list.Cases) == 0 {
		return fmt.Errorf("smoke list has no cases")
	}
	coveredPaths := map[string]bool{}
	for _, c := range list.Cases {
		if c.Name == "" || c.SrcPath == "" {
			return fmt.Errorf("smoke case missing name or src_path")
		}
		if err := validateExamplePath(c.SrcPath); err != nil {
			return fmt.Errorf("smoke case %s: %w", c.Name, err)
		}
		switch c.TargetGroup {
		case "native", "wasm":
		default:
			return fmt.Errorf("smoke case %s has invalid target_group %q", c.Name, c.TargetGroup)
		}
		if c.ExpectedExit < 0 || c.ExpectedExit > 255 {
			return fmt.Errorf("smoke case %s expected_exit = %d, want 0..255", c.Name, c.ExpectedExit)
		}
		entry, ok := entries[c.SrcPath]
		if !ok {
			return fmt.Errorf("example index missing %s", c.SrcPath)
		}
		coveredPaths[c.SrcPath] = true
		if entry.Purpose == "" {
			return fmt.Errorf("example index %s missing purpose", c.SrcPath)
		}
		if entry.Target == "" || !strings.Contains(entry.Target, c.TargetGroup) {
			return fmt.Errorf("example index %s target group %q does not mention %q", c.SrcPath, entry.Target, c.TargetGroup)
		}
		wantExit := fmt.Sprintf("exit %d", c.ExpectedExit)
		wantExits := fmt.Sprintf("exits %d", c.ExpectedExit)
		expected := strings.ToLower(entry.Expected)
		if !strings.Contains(expected, wantExit) && !strings.Contains(expected, wantExits) && !strings.Contains(expected, "build-only") {
			return fmt.Errorf("example index %s expected behavior must mention %s or build-only", c.SrcPath, wantExit)
		}
	}
	for _, exclusion := range list.ExcludedExamples {
		if err := validateExamplePath(exclusion.SrcPath); err != nil {
			return fmt.Errorf("excluded example: %w", err)
		}
		if strings.TrimSpace(exclusion.Reason) == "" {
			return fmt.Errorf("excluded example %s missing reason", exclusion.SrcPath)
		}
		coveredPaths[exclusion.SrcPath] = true
	}
	for path := range entries {
		if !coveredPaths[path] {
			return fmt.Errorf("example index includes %s but smoke list does not cover or exclude it", path)
		}
	}
	return nil
}

func validateExampleDocs(markdown string) error {
	entries, err := parseExampleIndex(markdown)
	if err != nil {
		return err
	}
	for path, entry := range entries {
		if entry.Purpose == "" {
			return fmt.Errorf("example index %s missing purpose", path)
		}
		if entry.Target == "" {
			return fmt.Errorf("example index %s missing target group", path)
		}
		if !strings.Contains(entry.Target, "native") && !strings.Contains(entry.Target, "wasm") {
			return fmt.Errorf("example index %s target group %q must mention native or wasm", path, entry.Target)
		}
		if entry.Expected == "" {
			return fmt.Errorf("example index %s missing expected behavior", path)
		}
	}
	return nil
}

func parseExampleIndex(markdown string) (map[string]exampleIndexEntry, error) {
	entries := map[string]exampleIndexEntry{}
	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") || strings.Contains(trimmed, "---") {
			continue
		}
		cols := splitMarkdownTableRow(trimmed)
		if len(cols) != 4 {
			continue
		}
		if strings.EqualFold(cols[0], "Example") {
			continue
		}
		path := strings.Trim(cols[0], "` ")
		if path == "" {
			continue
		}
		if _, exists := entries[path]; exists {
			return nil, fmt.Errorf("example index has duplicate entry %s", path)
		}
		if err := validateExamplePath(path); err != nil {
			return nil, fmt.Errorf("example index entry %q: %w", path, err)
		}
		entries[path] = exampleIndexEntry{
			Purpose:  strings.TrimSpace(cols[1]),
			Target:   strings.TrimSpace(cols[2]),
			Expected: strings.TrimSpace(cols[3]),
		}
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("example index has no table entries")
	}
	return entries, nil
}

func validateExamplePath(path string) error {
	if path == "" {
		return fmt.Errorf("missing src_path")
	}
	if strings.Contains(path, "\\") {
		return fmt.Errorf("src_path %q must use forward slashes", path)
	}
	if filepath.IsAbs(path) {
		return fmt.Errorf("src_path %q must be relative", path)
	}
	if !strings.HasPrefix(path, "examples/") {
		return fmt.Errorf("src_path %q must start with examples/", path)
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("src_path %q must not contain ..", path)
	}
	if !strings.HasSuffix(path, ".tetra") && !strings.HasSuffix(path, ".t4") {
		return fmt.Errorf("src_path %q must point to a .tetra or .t4 file", path)
	}
	return nil
}

func splitMarkdownTableRow(line string) []string {
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, strings.TrimSpace(part))
	}
	return out
}
