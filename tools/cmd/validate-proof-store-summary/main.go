package main

import (
	"flag"
	"fmt"
	"os"

	"tetra_language/tools/internal/ramvalidate"
)

func main() {
	reportPath := flag.String("report", "", "path to tetra.proof-store-summary.v1 JSON report")
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateProofStoreSummary(*reportPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateProofStoreSummary(path string) error {
	return ramvalidate.ValidateProofStoreSummaryFile(path)
}
