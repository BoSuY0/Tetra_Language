package main

import (
	"flag"
	"fmt"
	"os"

	"tetra_language/tools/validators/surface"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("validate-surface-dev-workflow", flag.ContinueOnError)
	reportPath := fs.String("report", "", "path to tetra.surface.dev-workflow.v1 report")
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
	return surface.ValidateDevWorkflowReport(raw)
}
