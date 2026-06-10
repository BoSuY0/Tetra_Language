package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"tetra_language/tools/validators/surface"
	"tetra_language/tools/validators/surfaceinspector"
)

func main() {
	os.Exit(runSurfaceInspect(os.Args[1:], os.Stdout, os.Stderr))
}

func runSurfaceInspect(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("surface-inspect", flag.ContinueOnError)
	fs.SetOutput(stderr)
	reportPath := fs.String("report", "", "Surface runtime report JSON to inspect")
	outPath := fs.String("out", "", "write inspector snapshot JSON to this path; defaults to stdout")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "surface-inspect does not accept positional arguments")
		return 2
	}
	if *reportPath == "" {
		fmt.Fprintln(stderr, "--report is required")
		return 2
	}
	if err := inspectSurfaceReport(*reportPath, *outPath, stdout); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *outPath != "" {
		fmt.Fprintf(stdout, "wrote surface inspector snapshot to %s\n", *outPath)
	}
	return 0
}

func inspectSurfaceReport(reportPath string, outPath string, stdout io.Writer) error {
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		return err
	}
	snapshot, err := surfaceinspector.SnapshotFromReportRaw(raw, reportPath)
	if err != nil {
		return err
	}
	out, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	if outPath == "" {
		_, err = stdout.Write(out)
		return err
	}
	return os.WriteFile(outPath, out, 0o644)
}

func snapshotFromSurfaceReport(report surface.Report, reportPath string) surfaceinspector.Snapshot {
	return surfaceinspector.SnapshotFromReport(report, reportPath)
}
