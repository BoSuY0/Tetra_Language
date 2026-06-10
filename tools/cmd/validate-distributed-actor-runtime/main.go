package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tetra_language/tools/validators/actordist"
)

func main() {
	reportPath := flag.String("report", "", "path to tetra.actors.distributed-runtime.v1 JSON report")
	currentGitHead := flag.String("current-git-head", "", "optional git head expected in the distributed actor runtime report")
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateDistributedActorRuntimeReportWithCurrentHead(*reportPath, *currentGitHead); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateDistributedActorRuntimeReport(path string) error {
	return validateDistributedActorRuntimeReportWithCurrentHead(path, "")
}

func validateDistributedActorRuntimeReportWithCurrentHead(path, currentGitHead string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := actordist.ValidateReport(raw); err != nil {
		return err
	}
	if currentGitHead == "" {
		return nil
	}
	var report actordist.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		return err
	}
	if report.GitHead != currentGitHead {
		return fmt.Errorf("git_head is %q, want current-git-head %q", report.GitHead, currentGitHead)
	}
	return nil
}
