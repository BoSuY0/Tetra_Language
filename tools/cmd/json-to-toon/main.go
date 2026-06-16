package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"tetra_language/internal/toon"
)

func main() {
	inPath := flag.String("in", "", "input JSON path; stdin when empty")
	outPath := flag.String("out", "", "output TOON path; stdout when empty")
	flag.Parse()

	raw, err := readInput(*inPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	out, err := jsonToTOON(raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := writeOutput(*outPath, out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func readInput(path string) ([]byte, error) {
	if path == "" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

func writeOutput(path string, data []byte) error {
	data = append(data, '\n')
	if path == "" {
		_, err := os.Stdout.Write(data)
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func jsonToTOON(raw []byte) ([]byte, error) {
	return toon.ConvertJSONToTOON(raw, toon.Options{Deterministic: true, Strict: true})
}
