package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"tetra_language/tools/validators/islandproof"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate-island-proof", flag.ContinueOnError)
	fs.SetOutput(stderr)
	proofPath := fs.String("proof", "", "path to tetra.island.proof.v1 JSON proof artifact")
	memoryReportPath := fs.String("memory-report", "", "path to tetra.memory-report.v1 JSON report")
	currentGitHead := fs.String(
		"current-git-head",
		"",
		"current git commit for --require-same-commit",
	)
	requireSameCommit := fs.Bool(
		"require-same-commit",
		false,
		"require proof git_head to match --current-git-head",
	)
	_ = fs.String(
		"plir",
		"",
		"optional PLIR artifact path; reserved for proof-carrying IR validation",
	)
	manifestPath := fs.String(
		"manifest",
		"",
		"optional release manifest path for proof artifact command metadata validation",
	)
	_ = fs.String("report-dir", "", "optional report directory; reserved for release integration")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *proofPath == "" {
		fmt.Fprintln(stderr, "error: --proof is required")
		return 2
	}
	if *memoryReportPath == "" {
		fmt.Fprintln(stderr, "error: --memory-report is required")
		return 2
	}
	if *requireSameCommit && *currentGitHead == "" {
		fmt.Fprintln(stderr, "error: --require-same-commit requires --current-git-head")
		return 2
	}
	proofRaw, err := os.ReadFile(*proofPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	memoryRaw, err := os.ReadFile(*memoryReportPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	var manifestRaw []byte
	if *manifestPath != "" {
		manifestRaw, err = os.ReadFile(*manifestPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	if err := islandproof.Validate(proofRaw, islandproof.Options{
		MemoryReport:      memoryRaw,
		Manifest:          manifestRaw,
		CurrentGitHead:    *currentGitHead,
		RequireSameCommit: *requireSameCommit,
	}); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "island proof validated")
	return 0
}
