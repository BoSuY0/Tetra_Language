package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"tetra_language/compiler"
)

func main() {
	outPath := flag.String("o", "", "output markdown path; stdout when empty")
	flag.Parse()
	docs, err := compiler.GenerateAPIDocs(flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if *outPath == "" {
		fmt.Print(string(docs))
		return
	}
	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outPath, docs, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
