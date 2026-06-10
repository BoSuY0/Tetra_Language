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

func runValidateSurfaceCrashReport(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate-surface-crash-report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	reportPath := fs.String("report", "", "Surface crash diagnostics report JSON")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "validate-surface-crash-report does not accept positional arguments")
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
	if err := surfacecrash.ValidateReport(raw); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "surface crash report OK")
	return 0
}
