package compiler

import (
	"tetra_language/compiler/internal/buildreports"
	"tetra_language/compiler/internal/semantics"
)

const p21LayoutPolicy = buildreports.P21LayoutPolicy

func buildLayoutReport(target string, checked *semantics.CheckedProgram) layoutReport {
	return buildreports.BuildLayoutReport(target, checked)
}

func ValidateLayoutReport(report layoutReport) error {
	return buildreports.ValidateLayoutReport(report)
}

func buildPerformanceReport(target string) perfReport {
	return buildreports.BuildPerformanceReport(target)
}

func ValidatePerformanceBlockerReport(report perfReport) error {
	return buildreports.ValidatePerformanceBlockerReport(report)
}
