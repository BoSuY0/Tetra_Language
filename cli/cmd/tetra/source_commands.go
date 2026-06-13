package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"tetra_language/compiler"
)

func runDoc(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("doc", flag.ContinueOnError)
	fs.SetOutput(stderr)
	outPath := fs.String("o", "", "output markdown path; stdout when empty")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text, json, or toon")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	paths := fs.Args()
	if len(paths) == 0 {
		ctx, err := discoverCLIProject(".")
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		if ctx != nil && ctx.Found {
			paths = existingProjectSourcePaths(ctx)
		}
	}
	docs, err := compiler.GenerateAPIDocs(paths)
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
	fmt.Fprintf(stdout, "Wrote docs: %s\n", *outPath)
	return 0
}

func runCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	interfaceOnly := fs.Bool("interface-only", false, "check interface/API surface without requiring executable output")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text, json, or toon")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	input := ""
	if fs.NArg() > 0 {
		input = fs.Arg(0)
	}
	if fs.NArg() > 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "check accepts at most one input path")
		return 2
	}
	input, worldOpt, projectCtx, err := resolveCLIInput(input)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if err := validateDiscoveredProjectLock(projectCtx, ""); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	world, err := compiler.LoadWorldOpt(input, worldOpt)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	checkOpt := compiler.CheckOptions{RequireMain: true}
	if *interfaceOnly {
		checkOpt.RequireMain = false
	}
	if _, err := compiler.CheckWorldOpt(world, checkOpt); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	fmt.Fprintf(stdout, "Checked: %s\n", input)
	return 0
}

func runFmt(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("fmt", flag.ContinueOnError)
	fs.SetOutput(stderr)
	check := fs.Bool("check", false, "check whether files are formatted")
	write := fs.Bool("write", false, "rewrite files in place")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text, json, or toon")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	if *check && *write {
		writeValidationDiagnostic(stderr, *diagnostics, "fmt accepts only one of --check or --write")
		return 2
	}
	paths := fs.Args()
	if len(paths) == 0 {
		writeValidationDiagnostic(stderr, *diagnostics, "fmt requires at least one path")
		return 2
	}
	files, err := collectTetraFiles(paths)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if !*check && !*write && len(files) != 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "fmt stdout mode accepts exactly one file")
		return 2
	}
	dirty := false
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		formatted, err := compiler.FormatSource(raw, path)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		if *check {
			if string(raw) != string(formatted) {
				dirty = true
				if *diagnostics == "json" || *diagnostics == "toon" {
					line, column := firstFormatterDiffPosition(raw, formatted)
					writeDiagnosticObject(stderr, *diagnostics, compiler.Diagnostic{
						Code:     compiler.DiagnosticCodeFormatterCheck,
						Message:  "not formatted",
						File:     path,
						Line:     line,
						Column:   column,
						Severity: "error",
						Hint:     "Run tetra fmt --write to update the file.",
					})
				} else {
					fmt.Fprintf(stderr, "%s: not formatted\n", path)
				}
			}
			continue
		}
		if *write {
			if string(raw) != string(formatted) {
				if err := os.WriteFile(path, formatted, 0o644); err != nil {
					writeDiagnostic(stderr, *diagnostics, err)
					return 1
				}
			}
			continue
		}
		fmt.Fprint(stdout, string(formatted))
	}
	if dirty {
		return 1
	}
	return 0
}

func firstFormatterDiffPosition(raw []byte, formatted []byte) (int, int) {
	line := 1
	column := 1
	limit := len(raw)
	if len(formatted) < limit {
		limit = len(formatted)
	}
	for i := 0; i < limit; i++ {
		if raw[i] != formatted[i] {
			return line, column
		}
		if raw[i] == '\n' {
			line++
			column = 1
			continue
		}
		column++
	}
	return line, column
}
