package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"tetra_language/tools/validators/postv04prod"
)

func main() {
	var reportDir string
	var auditPath string
	var outPath string
	var write bool
	flag.StringVar(&reportDir, "report-dir", "", "post-v0.4 production report directory")
	flag.StringVar(
		&auditPath,
		"audit",
		"",
		"path to post-v0.4 production audit JSON; defaults inside --report-dir",
	)
	flag.BoolVar(&write, "write", false, "write a fresh post-v0.4 production audit")
	flag.StringVar(
		&outPath,
		"out",
		"",
		"audit output path for --write; defaults inside --report-dir",
	)
	flag.Parse()

	if reportDir == "" {
		fmt.Fprintln(os.Stderr, "error: --report-dir is required")
		os.Exit(2)
	}
	var err error
	if write {
		err = writePostV04ProductionAudit(reportDir, outPath)
	} else {
		err = validatePostV04ProductionAudit(reportDir, auditPath)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func writePostV04ProductionAudit(reportDir, outPath string) error {
	if outPath == "" {
		outPath = filepath.Join(reportDir, postv04prod.DefaultAuditFilename)
	}
	report, err := postv04prod.BuildReport(reportDir)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outPath, raw, 0o644)
}

func validatePostV04ProductionAudit(reportDir, auditPath string) error {
	if auditPath == "" {
		return postv04prod.ValidateReportDir(reportDir)
	}
	raw, err := os.ReadFile(auditPath)
	if err != nil {
		return err
	}
	var report postv04prod.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		return err
	}
	if err := postv04prod.ValidateReport(report); err != nil {
		return err
	}
	return postv04prod.ValidateReportDir(reportDir)
}
