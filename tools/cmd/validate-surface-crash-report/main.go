package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"tetra_language/tools/validators/surfacecrash"
)

func main() {
	os.Exit(runValidateSurfaceCrashReport(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string) error {
	return validateSurfaceCrashReportArgs(args)
}

func runValidateSurfaceCrashReport(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateSurfaceCrashReportArgs(args); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "surface crash report OK")
	return 0
}

func validateSurfaceCrashReportArgs(args []string) error {
	fs := flag.NewFlagSet("validate-surface-crash-report", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	reportPath := fs.String("report", "", "path to tetra.surface.crash-report.v1 report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *reportPath == "" {
		return fmt.Errorf("--report is required")
	}
	raw, err := os.ReadFile(*reportPath)
	if err != nil {
		return err
	}
	return surfacecrash.ValidateReport(raw)
}
