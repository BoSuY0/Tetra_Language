package main

import (
	"encoding/json"
	"fmt"
	"os"

	"tetra_language/compiler/internal/parallelrt"
)

func main() {
	rows, err := parallelrt.PrototypeBenchmarks()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rows); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
