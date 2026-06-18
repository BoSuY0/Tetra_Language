package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"tetra_language/tools/validators/surfaceelectron"
)

func main() {
	os.Exit(runSurfaceElectronComparison(os.Args[1:], os.Stdout, os.Stderr))
}

func runSurfaceElectronComparison(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("surface-electron-comparison", flag.ContinueOnError)
	fs.SetOutput(stderr)
	outPath := fs.String("out", "", "output Surface-vs-Electron comparison report JSON")
	claimOfficial := fs.Bool("claim-official-benchmark", false, "intentionally request an unsupported official benchmark claim")
	claimFaster := fs.Bool("claim-faster-than-electron", false, "intentionally request an unsupported faster-than-Electron claim")
	sampleCount := fs.Int("sample-count", 7, "sample count to record in the deterministic comparison report")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "surface-electron-comparison does not accept positional arguments")
		return 2
	}
	if *outPath == "" {
		fmt.Fprintln(stderr, "--out is required")
		return 2
	}

	report := surfaceelectron.ValidFixtureReport()
	report.GitHead = currentGitHead()
	report.Method.SampleCount = *sampleCount
	for i := range report.Metrics {
		report.Metrics[i].SampleCount = *sampleCount
	}
	if *claimOfficial {
		report.Positioning.OfficialBenchmarkClaim = true
		report.Positioning.Claim = "Surface has official benchmark superiority over Electron."
	}
	if *claimFaster {
		report.Positioning.FasterThanElectronClaim = true
		report.Positioning.Claim = "Surface is faster than Electron."
	}
	if err := surfaceelectron.Validate(report); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.WriteFile(*outPath, raw, 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "surface electron comparison report: %s\n", *outPath)
	return 0
}

func currentGitHead() string {
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "0123456789abcdef0123456789abcdef01234567"
	}
	head := strings.TrimSpace(string(out))
	if len(head) != 40 {
		return "0123456789abcdef0123456789abcdef01234567"
	}
	return head
}
