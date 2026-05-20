package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"tetra_language/compiler"
)

func runInterface(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("interface", flag.ContinueOnError)
	fs.SetOutput(stderr)
	outPath := fs.String("o", "", "output .t4i path; stdout when empty")
	checkMode := fs.Bool("check", false, "check that the .t4i public API hash matches the source")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	if fs.NArg() != 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "interface requires exactly one input path")
		return 2
	}
	inputPath := fs.Arg(0)
	if *checkMode {
		path := *outPath
		if path == "" {
			path = compiler.InterfaceOutputPath(inputPath)
		}
		src, err := os.ReadFile(inputPath)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		iface, err := os.ReadFile(path)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		if err := compiler.ValidateInterfaceAgainstSource(src, iface, inputPath); err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		fmt.Fprintf(stdout, "Interface current: %s\n", path)
		return 0
	}
	docs, err := compiler.GenerateInterfaceFile(inputPath)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if *outPath == "" {
		fmt.Fprint(stdout, string(docs))
		return 0
	}
	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if err := os.WriteFile(*outPath, docs, 0o644); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	fmt.Fprintf(stdout, "Wrote interface: %s\n", *outPath)
	return 0
}
