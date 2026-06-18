package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"tetra_language/compiler"
	"tetra_language/tools/validators/platformui"
)

func main() {
	reportPath := flag.String("report", "", "path to macOS UI runtime report")
	expectedVersion := flag.String(
		"expected-version",
		compiler.Version(),
		"expected compiler/runtime version",
	)
	expectedGitHead := flag.String(
		"expected-git-head",
		"",
		"expected git HEAD; defaults to current repository HEAD",
	)
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if strings.TrimSpace(*expectedGitHead) == "" {
		head, err := currentGitHead()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: could not determine expected git head: %v\n", err)
			os.Exit(2)
		}
		*expectedGitHead = head
	}
	raw, err := os.ReadFile(*reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := platformui.ValidateReportWithOptions(raw, platformui.ValidateOptions{
		ExpectedTarget:  "macos-x64",
		ExpectedVersion: *expectedVersion,
		ExpectedGitHead: *expectedGitHead,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func currentGitHead() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
