package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"tetra_language/tools/internal/verifydocs"
)

func main() {
	manifestPath := flag.String(
		"manifest",
		"docs/generated/manifest.json",
		"path to generated manifest json",
	)
	flag.Parse()

	if err := verifydocs.Run("", *manifestPath); err != nil {
		if strings.HasPrefix(err.Error(), "verify-docs:") {
			fmt.Fprintln(os.Stderr, err)
		} else {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
		os.Exit(1)
	}
}
