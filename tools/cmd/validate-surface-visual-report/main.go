package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"tetra_language/tools/validators/surfacevisual"
)

func main() {
	os.Exit(runValidateSurfaceVisualReport(os.Args[1:], os.Stdout, os.Stderr))
}

func runValidateSurfaceVisualReport(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate-surface-visual-report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	reportPath := fs.String("report", "", "Surface visual regression report JSON")
	root := fs.String("root", "", "artifact root; defaults to the report directory")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "validate-surface-visual-report does not accept positional arguments")
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
	if err := surfacevisual.ValidateReportWithRoot(raw, artifactRoot); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "surface visual report OK")
	return 0
}
