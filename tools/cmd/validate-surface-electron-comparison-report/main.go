package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"tetra_language/tools/validators/surfaceelectron"
)

func main() {
	os.Exit(runValidateSurfaceElectronComparisonReport(os.Args[1:], os.Stdout, os.Stderr))
}

func runValidateSurfaceElectronComparisonReport(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate-surface-electron-comparison-report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	reportPath := fs.String("report", "", "Surface-vs-Electron comparison report JSON")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "validate-surface-electron-comparison-report does not accept positional arguments")
		return 2
	}
	if *reportPath == "" {
		fmt.Fprintln(stderr, "--report is required")
		return 2
	}
	raw, err := os.ReadFile(*reportPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := surfaceelectron.ValidateReport(raw); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "surface electron comparison report OK")
	return 0
}
