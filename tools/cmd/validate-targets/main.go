package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type targetsReport struct {
	Supported []string `json:"supported"`
	BuildOnly []string `json:"build_only"`
	Planned   []string `json:"planned"`
}

func main() {
	var path string
	flag.StringVar(&path, "report", "", "path to tetra targets --format=json output")
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
	if err := validateTargetsReport(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateTargetsReport(raw []byte) error {
	var report targetsReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return fmt.Errorf("invalid targets JSON: %w", err)
	}
	if err := validateTargetList("supported", report.Supported, []string{"linux-x64", "windows-x64", "macos-x64"}); err != nil {
		return err
	}
	if err := validateTargetList("build_only", report.BuildOnly, []string{"wasm32-wasi", "wasm32-web"}); err != nil {
		return err
	}
	if err := validateTargetList("planned", report.Planned, []string{}); err != nil {
		return err
	}
	return nil
}

func validateTargetList(name string, got []string, want []string) error {
	if len(got) != len(want) {
		return fmt.Errorf("%s target count = %d, want %d", name, len(got), len(want))
	}
	seen := map[string]bool{}
	for i, target := range got {
		if target != want[i] {
			return fmt.Errorf("%s target[%d] = %q, want %q", name, i, target, want[i])
		}
		if seen[target] {
			return fmt.Errorf("%s target %q is duplicated", name, target)
		}
		seen[target] = true
	}
	return nil
}
