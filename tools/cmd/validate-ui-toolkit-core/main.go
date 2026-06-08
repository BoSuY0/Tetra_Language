package main

import (
	"flag"
	"fmt"
	"os"

	"tetra_language/tools/validators/uitoolkit"
)

func main() {
	reportPath := flag.String("report", "", "path to tetra.ui.toolkit.v1 JSON report")
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateUIToolkitCoreReport(*reportPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateUIToolkitCoreReport(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return uitoolkit.ValidateReport(raw)
}
