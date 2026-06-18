package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"tetra_language/tools/validators/surfaceinspector"
)

func runSurface(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printSurfaceUsage(stderr)
		return 2
	}
	if isHelpArgs(args) {
		printSurfaceUsage(stdout)
		return 0
	}
	switch args[0] {
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "run":
		return runRun(args[1:], stdout, stderr)
	case "dev":
		return runSurfaceDevCommand(args[1:], stdout, stderr)
	case "inspect":
		return runSurfaceInspectCommand(args[1:], stdout, stderr)
	case "package":
		return runSurfacePackageCommand(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown surface command %q\n", args[0])
		printSurfaceUsage(stderr)
		return 2
	}
}

func printSurfaceUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: tetra surface check [PROJECT]")
	fmt.Fprintln(w, "       tetra surface run [PROJECT]")
	fmt.Fprintln(w, "       tetra surface dev --project PROJECT [--once] [--state STATE] [--report REPORT]")
	fmt.Fprintln(w, "       tetra surface inspect --report REPORT [--out SNAPSHOT]")
	fmt.Fprintln(w, "       tetra surface package [PROJECT] [-o PACKAGE]")
}

func runSurfaceInspectCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("surface inspect", flag.ContinueOnError)
	fs.SetOutput(stderr)
	reportPath := fs.String("report", "", "Surface runtime report JSON")
	outPath := fs.String("out", "", "write inspector snapshot JSON to this path; defaults to stdout")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "surface inspect does not accept positional arguments")
		return 2
	}
	if *reportPath == "" {
		fmt.Fprintln(stderr, "--report is required")
		return 2
	}
	raw, err := os.ReadFile(*reportPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	snapshot, err := surfaceinspector.SnapshotFromReportRaw(raw, *reportPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	out, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	out = append(out, '\n')
	if *outPath == "" {
		if _, err := stdout.Write(out); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	if err := os.WriteFile(*outPath, out, 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote surface inspector snapshot to %s\n", *outPath)
	return 0
}
