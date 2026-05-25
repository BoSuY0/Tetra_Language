package main

import (
	"flag"
	"fmt"
	"os"

	"tetra_language/tools/validators/techempower"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("validate-techempower-report", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	reportPath := flags.String("report", "", "path to TechEmpower semantic or matrix JSON report")
	allowSkipDB := flags.Bool("allow-skip-db", false, "allow local smoke reports that intentionally skip database endpoints")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *reportPath == "" {
		return fmt.Errorf("--report is required")
	}
	raw, err := os.ReadFile(*reportPath)
	if err != nil {
		return err
	}
	return techempower.ValidateReport(raw, techempower.Options{AllowSkipDB: *allowSkipDB})
}
