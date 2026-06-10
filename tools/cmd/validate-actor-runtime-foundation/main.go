package main

import (
	"flag"
	"fmt"
	"os"

	"tetra_language/tools/validators/actorprod"
)

func main() {
	reportDir := flag.String("report-dir", "", "path to actor runtime foundation report directory")
	currentGitHead := flag.String("current-git-head", "", "optional git head expected in the actor foundation manifest")
	flag.Parse()
	if *reportDir == "" {
		fmt.Fprintln(os.Stderr, "error: --report-dir is required")
		os.Exit(2)
	}
	if err := validateActorRuntimeFoundationReportDir(*reportDir, *currentGitHead); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateActorRuntimeFoundationReportDir(reportDir, currentGitHead string) error {
	return actorprod.ValidateReportDir(reportDir, actorprod.Options{CurrentGitHead: currentGitHead})
}
