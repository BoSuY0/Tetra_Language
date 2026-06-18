package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"tetra_language/tools/validators/surfaceipc"
)

func main() {
	os.Exit(runValidateSurfaceIPCReport(os.Args[1:], os.Stdout, os.Stderr))
}

func runValidateSurfaceIPCReport(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate-surface-ipc-report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	reportPath := fs.String("report", "", "Surface IPC/lifecycle report JSON")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "validate-surface-ipc-report does not accept positional arguments")
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
	if err := surfaceipc.ValidateReport(raw); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "surface IPC/lifecycle report OK")
	return 0
}
