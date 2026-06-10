package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"tetra_language/tools/validators/surfaceprod"
)

func main() {
	reportPath := flag.String("report", "", "path to tetra.surface.prod-claim.v1 report")
	flag.Parse()
	if strings.TrimSpace(*reportPath) == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateSurfaceProdClaim(*reportPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateSurfaceProdClaim(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return surfaceprod.ValidateClaim(raw)
}
