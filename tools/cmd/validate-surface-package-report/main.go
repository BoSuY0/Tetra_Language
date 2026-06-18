package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"tetra_language/tools/validators/surfacepackage"
)

func main() {
	os.Exit(runValidateSurfacePackageReport(os.Args[1:], os.Stdout, os.Stderr))
}

func runValidateSurfacePackageReport(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate-surface-package-report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	reportPath := fs.String("report", "", "Surface package distribution report JSON")
	root := fs.String("root", "", "artifact root; defaults to the report directory")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "validate-surface-package-report does not accept positional arguments")
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
	artifactRoot := *root
	if artifactRoot == "" {
		artifactRoot = filepath.Dir(*reportPath)
	}
	if err := surfacepackage.ValidateReportWithRoot(raw, artifactRoot); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "surface package report OK")
	return 0
}
