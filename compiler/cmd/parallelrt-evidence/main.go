package main

import (
	"encoding/json"
	"fmt"
	"os"

	"tetra_language/compiler/internal/parallelrt"
)

func main() {
	evidence, err := parallelrt.CollectPrototypeEvidence()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(evidence); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
