package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"tetra_language/tools/internal/surfacehost"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("tetra-surface-host", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	backendName := fs.String("backend", "wayland", "Surface host backend")
	socketPath := fs.String("socket", "", "absolute Unix socket path for tetra.surface.host-ipc.v1")
	reportPath := fs.String("report", "", "optional host-side JSON evidence report path")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "tetra-surface-host does not accept UI files or positional arguments")
		return 2
	}
	backend, err := surfacehost.NewBackend(*backendName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	var reporter *surfacehost.ReportingBackend
	if *reportPath != "" {
		reporter = surfacehost.NewReportingBackend(backend, *backendName, *socketPath)
		backend = reporter
		defer func() {
			if err := reporter.WriteReport(*reportPath); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}()
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := surfacehost.ListenAndServeUnix(ctx, *socketPath, backend); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
