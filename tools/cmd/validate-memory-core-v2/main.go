package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"tetra_language/tools/validators/memorycorev2"
)

type stringList []string

func (l *stringList) String() string {
	if l == nil {
		return ""
	}
	return strings.Join(*l, ",")
}

func (l *stringList) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("empty path")
	}
	*l = append(*l, value)
	return nil
}

func main() {
	reportPath := flag.String("report", "", "path to tetra.memory-core-v2.evidence.v1 JSON report")
	currentGitHead := flag.String("current-git-head", "", "optional current git HEAD to require")
	var claimPaths stringList
	flag.Var(&claimPaths, "claim-path", "Markdown or text file to scan for unsupported Memory Core v2 claims; repeatable")
	flag.Parse()

	if strings.TrimSpace(*reportPath) == "" && len(claimPaths) == 0 {
		fmt.Fprintln(os.Stderr, "error: --report or --claim-path is required")
		os.Exit(2)
	}

	var issues []string
	if strings.TrimSpace(*reportPath) != "" {
		if err := memorycorev2.ValidateReportFile(
			*reportPath,
			memorycorev2.Options{CurrentGitHead: strings.TrimSpace(*currentGitHead)},
		); err != nil {
			issues = append(issues, err.Error())
		}
	}
	for _, path := range claimPaths {
		if err := memorycorev2.ValidateClaimFile(path); err != nil {
			issues = append(issues, err.Error())
		}
	}
	if len(issues) > 0 {
		fmt.Fprintln(os.Stderr, strings.Join(issues, "; "))
		os.Exit(1)
	}
}
