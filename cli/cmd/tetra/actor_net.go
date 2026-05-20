package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"tetra_language/cli/internal/actornet"
)

func runActorNet(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("actor-net", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addr := fs.String("addr", "127.0.0.1:0", "loopback address to listen on")
	report := fs.String("report", "", "optional JSON runtime report path")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Fprintln(stdout, "usage: tetra actor-net [--addr 127.0.0.1:PORT] [--report PATH]")
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "actor-net does not accept positional arguments")
		return 2
	}

	broker, err := actornet.NewBroker(actornet.Config{
		Addr:       *addr,
		ReportPath: *report,
	})
	if err != nil {
		fmt.Fprintf(stderr, "actor-net: %v\n", err)
		return 1
	}
	defer broker.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Fprintf(stdout, "Actor network broker listening on %s\n", broker.Addr())
	if err := broker.Serve(ctx); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintf(stderr, "actor-net: %v\n", err)
		return 1
	}
	return 0
}
