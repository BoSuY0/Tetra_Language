package main

import (
	"flag"
	"fmt"
	"os"

	"tetra_language/tools/internal/ramvalidate"
)

func main() {
	reportPath := flag.String("report", "", "path to tetra.memory-grade-report.v1 JSON report")
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateMemoryGradeReport(*reportPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateMemoryGradeReport(path string) error {
	return ramvalidate.ValidateGradeReportFile(path)
}
