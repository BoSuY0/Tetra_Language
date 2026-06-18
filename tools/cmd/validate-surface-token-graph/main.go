package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"tetra_language/tools/validators/surface"
)

type tokenGraphCLIOptions struct {
	ContractPath string
	ReportPath   string
	Root         string
}

func main() {
	contractPath := flag.String(
		"contract",
		"",
		"path to tetra.surface.token-graph.contract.v1 JSON",
	)
	reportPath := flag.String(
		"report",
		"",
		"path to tetra.surface.runtime.v1 report with morph token graph evidence",
	)
	root := flag.String("root", ".", "repo root used to scan token graph reference sources")
	flag.Parse()
	if err := validateSurfaceTokenGraph(
		tokenGraphCLIOptions{ContractPath: *contractPath, ReportPath: *reportPath, Root: *root},
	); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateSurfaceTokenGraph(options tokenGraphCLIOptions) error {
	if strings.TrimSpace(options.ContractPath) == "" {
		return fmt.Errorf("--contract is required")
	}
	if strings.TrimSpace(options.ReportPath) == "" {
		return fmt.Errorf("--report is required")
	}
	contractRaw, err := os.ReadFile(options.ContractPath)
	if err != nil {
		return fmt.Errorf("read token graph contract: %w", err)
	}
	reportRaw, err := os.ReadFile(options.ReportPath)
	if err != nil {
		return fmt.Errorf("read token graph report: %w", err)
	}
	return surface.ValidateTokenGraphContract(
		contractRaw,
		reportRaw,
		surface.TokenGraphValidationOptions{Root: options.Root},
	)
}
