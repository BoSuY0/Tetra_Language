package main

import "tetra_language/tools/validators/memoryprod"

type smokeOptions struct {
	ReportPath         string
	RAMMeasurementPath string
	TetraPath          string
	GitHead            string
	KeepWork           bool
}

type smokeRunner struct {
	opt          smokeOptions
	workDir      string
	sourceDir    string
	tetraPath    string
	processes    []memoryprod.ProcessReport
	benchmarks   []memoryprod.BenchmarkReport
	cases        []memoryprod.CaseReport
	ramSnapshots []ramMeasurementSnapshot
}

type processResult struct {
	exitCode int
	output   string
	err      error
}
