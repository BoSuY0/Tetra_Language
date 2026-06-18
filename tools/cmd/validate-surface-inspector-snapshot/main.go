package main

import (
	"flag"
	"fmt"
	"os"

	"tetra_language/tools/validators/surfaceinspector"
)

func main() {
	snapshotPath := flag.String("snapshot", "", "Surface inspector snapshot JSON")
	flag.Parse()
	if *snapshotPath == "" {
		fmt.Fprintln(os.Stderr, "--snapshot is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(*snapshotPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := surfaceinspector.ValidateSnapshot(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("surface inspector snapshot OK")
}
