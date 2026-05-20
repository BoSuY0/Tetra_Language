package main

import (
	"flag"
	"fmt"
	"os"

	"tetra_language/tools/validators/nativeui"
)

func main() {
	reportPath := flag.String("report", "", "path to tetra.ui.native-runtime.v1 JSON report")
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateNativeUIRuntimeReport(*reportPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateNativeUIRuntimeReport(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return nativeui.ValidateReport(raw)
}
