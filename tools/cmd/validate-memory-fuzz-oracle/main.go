package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tetra_language/compiler"
)

func main() {
	var reportPath string
	flag.StringVar(&reportPath, "report", "", "path to tetra.memory-fuzz.oracle.v1 report")
	flag.Parse()
	if reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateMemoryFuzzOracleReportFile(reportPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateMemoryFuzzOracleReportFile(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var report compiler.MemoryFuzzOracleReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return fmt.Errorf("memory fuzz oracle report is malformed: %w", err)
	}
	return compiler.ValidateMemoryFuzzOracleReport(report)
}
