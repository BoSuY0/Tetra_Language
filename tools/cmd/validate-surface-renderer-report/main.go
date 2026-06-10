package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"tetra_language/tools/validators/surfacerenderer"
)

func main() {
	reportPath := flag.String("report", "", "path to tetra.surface.renderer-backend.v1 report")
	flag.Parse()
	if strings.TrimSpace(*reportPath) == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateSurfaceRendererReport(*reportPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateSurfaceRendererReport(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return surfacerenderer.ValidateReport(raw)
}
