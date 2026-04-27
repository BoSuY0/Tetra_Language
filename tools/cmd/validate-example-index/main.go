package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

type smokeList struct {
	Cases []smokeCase `json:"cases"`
}

type smokeCase struct {
	Name         string `json:"name"`
	SrcPath      string `json:"src_path"`
	TargetGroup  string `json:"target_group"`
	ExpectedExit int    `json:"expected_exit"`
}

type exampleIndexEntry struct {
	Purpose  string
	Target   string
	Expected string
}

func main() {
	smokeListPath := flag.String("smoke-list", "", "path to tetra smoke --list --format=json output")
	indexPath := flag.String("index", "docs/user/examples_index.md", "path to examples index markdown")
	flag.Parse()

	if *smokeListPath == "" {
		fmt.Fprintln(os.Stderr, "error: --smoke-list is required")
		os.Exit(2)
	}
	rawSmoke, err := os.ReadFile(*smokeListPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	rawIndex, err := os.ReadFile(*indexPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateExampleIndex(rawSmoke, string(rawIndex)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateExampleIndex(rawSmoke []byte, markdown string) error {
	var list smokeList
	if err := json.Unmarshal(rawSmoke, &list); err != nil {
		return err
	}
	entries, err := parseExampleIndex(markdown)
	if err != nil {
		return err
	}
	if len(list.Cases) == 0 {
		return fmt.Errorf("smoke list has no cases")
	}
	for _, c := range list.Cases {
		if c.Name == "" || c.SrcPath == "" {
			return fmt.Errorf("smoke case missing name or src_path")
		}
		entry, ok := entries[c.SrcPath]
		if !ok {
			return fmt.Errorf("example index missing %s", c.SrcPath)
		}
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

func splitMarkdownTableRow(line string) []string {
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, strings.TrimSpace(part))
	}
	return out
}
