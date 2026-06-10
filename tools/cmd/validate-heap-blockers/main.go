package main

import (
	"flag"
	"fmt"
	"os"

	"tetra_language/tools/internal/ramvalidate"
)

func main() {
	reportPath := flag.String("report", "", "path to heap blocker report")
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateHeapBlockers(*reportPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateHeapBlockers(path string) error {
	return ramvalidate.ValidateBlockerReportFile(path, "heap")
}
