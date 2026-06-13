package main

import (
	"flag"
	"fmt"
	"os"

	"tetra_language/tools/validators/surface"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "validate-surface-visual-report: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("validate-surface-visual-report", flag.ContinueOnError)
	var reportPath string
	fs.StringVar(&reportPath, "report", "", "path to tetra.surface.visual-regression.v1 report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if reportPath == "" {
		return fmt.Errorf("--report is required")
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", reportPath, err)
	}
	if err := surface.ValidateVisualReport(raw); err != nil {
		return err
	}
	return nil
}
