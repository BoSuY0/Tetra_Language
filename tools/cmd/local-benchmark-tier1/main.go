package main

import (
	"flag"
	"fmt"
	"os"
	"tetra_language/tools/internal/localbenchmarktier1"
	"time"
)

func main() {
	outDir := flag.String(
		"out-dir",
		"reports/local-benchmark-tier1-v1",
		"output artifact directory",
	)
	iterations := flag.Int("iterations", 3, "run iterations per benchmark row")
	timeout := flag.Duration("timeout", 20*time.Second, "timeout per build/run command")
	flag.Parse()
	if err := localbenchmarktier1.Run(
		localbenchmarktier1.Options{OutDir: *outDir, Iterations: *iterations, Timeout: *timeout},
	); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
