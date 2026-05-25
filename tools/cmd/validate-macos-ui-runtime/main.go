package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"tetra_language/tools/validators/uiplatform"
)

func main() {
	reportPath := flag.String("report", "", "path to tetra.ui.platform.v1 macOS UI runtime report")
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateMacOSUIRuntimeReport(*reportPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateMacOSUIRuntimeReport(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return uiplatform.ValidateReport(raw, uiplatform.Options{
		Target:  "macos-x64",
		Host:    "macos-x64",
		Runtime: "platform-ui-macos-x64",
		Now:     time.Now().UTC(),
		MaxAge:  uiplatform.DefaultMaxEvidenceAge,
	})
}
