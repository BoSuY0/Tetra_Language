package main

import (
	"flag"
	"fmt"
	"os"

	"tetra_language/tools/internal/ramvalidate"
)

func main() {
	reportPath := flag.String("report", "", "path to copy blocker report")
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateCopyBlockers(*reportPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateCopyBlockers(path string) error {
	return ramvalidate.ValidateBlockerReportFile(path, "copy")
}
