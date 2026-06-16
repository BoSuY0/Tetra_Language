package main

import (
	"flag"
	"fmt"
	"os"

	"tetra_language/compiler"
	"tetra_language/internal/outputformat"
)

func main() {
	outPath := flag.String("o", "docs/generated/manifest.json", "output path")
	format := flag.String("format", outputformat.JSON, "manifest output format: json, toon, or both")
	flag.Parse()

	manifest, err := compiler.GetManifest()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if _, err := outputformat.WriteStructuredFiles(*outPath, *format, manifest); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
