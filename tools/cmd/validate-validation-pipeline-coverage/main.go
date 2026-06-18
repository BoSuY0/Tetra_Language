package main

import (
	"flag"
	"fmt"
	"os"

	"tetra_language/tools/internal/ramvalidate"
)

func main() {
	reportPath := flag.String(
		"report",
		"",
		"path to tetra.validation-pipeline-coverage.v1 JSON report",
	)
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateValidationPipelineCoverage(*reportPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateValidationPipelineCoverage(path string) error {
	return ramvalidate.ValidatePipelineCoverageFile(path)
}
