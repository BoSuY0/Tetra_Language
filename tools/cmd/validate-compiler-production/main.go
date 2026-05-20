package main

import (
	"flag"
	"fmt"
	"os"

	"tetra_language/tools/validators/compilerprod"
)

func main() {
	reportPath := flag.String("report", "", "path to tetra.compiler.production.v1 JSON report")
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateCompilerProductionReport(*reportPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateCompilerProductionReport(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return compilerprod.ValidateReport(raw)
}
