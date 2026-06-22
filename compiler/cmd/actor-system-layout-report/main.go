package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"tetra_language/compiler/internal/actorsrt"
)

func main() {
	out := flag.String("out", "", "optional path to write the actor system layout report")
	flag.Parse()

	raw, err := json.MarshalIndent(actorsrt.ActorSystemLayoutReport(), "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	raw = append(raw, '\n')
	if *out == "" {
		_, _ = os.Stdout.Write(raw)
		return
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, raw, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
