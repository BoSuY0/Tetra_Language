package main

import (
	"flag"
	"fmt"
	"os"

	"tetra_language/tools/validators/actordist"
)

func main() {
	reportPath := flag.String("report", "", "path to tetra.actors.distributed-runtime.v1 JSON report")
	currentGitHead := flag.String("current-git-head", "", "optional current git HEAD required to match report git_head")
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateDistributedActorRuntimeReport(*reportPath, *currentGitHead); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateDistributedActorRuntimeReport(path string, currentGitHead string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if currentGitHead != "" {
		return actordist.ValidateReportForCurrentHead(raw, currentGitHead)
	}
	return actordist.ValidateReport(raw)
}
