package main

import (
	"flag"
	"fmt"
	"os"

	"tetra_language/tools/validators/surfacemacos"
)

func main() {
	reportPath := flag.String("report", "", "path to Surface macOS target boundary report")
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(*reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := surfacemacos.ValidateReport(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
