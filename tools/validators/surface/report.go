package surface

import (
	"errors"
	"fmt"
	"strings"
)

func ValidateReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != SchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, SchemaV1)
	}

	var report Report
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	issues = append(issues, rejectNonRuntimeEvidence(raw)...)
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if report.Target != "headless" && report.Target != "linux-x64" && report.Target != "wasm32-web" {
		issues = append(issues, fmt.Sprintf("target is %q, want headless, linux-x64, or wasm32-web", report.Target))
	}
	if report.Host != "linux-x64" {
		issues = append(issues, fmt.Sprintf("host is %q, want linux-x64", report.Host))
	}
	if report.Runtime != "surface-headless" && report.Runtime != "surface-linux-x64" && report.Runtime != "surface-wasm32-web" {
		issues = append(issues, fmt.Sprintf("runtime is %q, want surface-headless, surface-linux-x64, or surface-wasm32-web", report.Runtime))
	}
	if report.SurfaceSchema != "tetra.surface.v1" {
		issues = append(issues, fmt.Sprintf("surface_schema is %q, want tetra.surface.v1", report.SurfaceSchema))
	}
	if report.HostABI != "tetra.surface.host-abi.v1" {
		issues = append(issues, fmt.Sprintf("host_abi is %q, want tetra.surface.host-abi.v1", report.HostABI))
	}
	issues = append(issues, validateHostEvidence(report)...)
	if strings.TrimSpace(report.Source) == "" {
		issues = append(issues, "source is required")
	}
	issues = append(issues, validateProcesses(report.Source, report.Processes)...)
	issues = append(issues, validateArtifacts(report.Target, report.Source, report.Artifacts, report.Processes)...)
	issues = append(issues, validateArtifactScan(report.ArtifactScan, report.Artifacts)...)
	componentIndex, componentIssues := validateComponents(report.Components)
	issues = append(issues, componentIssues...)
	issues = append(issues, validateSourceComponentModel(report.Source, report.Components)...)
	issues = append(issues, validateEvents(report.Events, componentIndex)...)
	issues = append(issues, validateFrames(report.Frames)...)
	issues = append(issues, validateFrameProvenance(report)...)
	issues = append(issues, validateStateTransitions(report.StateTransitions, componentIndex)...)
	issues = append(issues, validateCases(report.Cases)...)
	issues = append(issues, validateTargetRuntimeEvidence(report)...)
	issues = append(issues, validateTextFocusInputEvidence(report, componentIndex)...)
	issues = append(issues, validateComponentTreeEvidence(report)...)
	issues = append(issues, validateBlockGraphEvidence(report)...)
	issues = append(issues, validateBlockCorePrimitiveEvidence(report)...)
	issues = append(issues, validateBlockSceneSnapshotEvidence(report)...)
	issues = append(issues, validateRenderCommandStreamEvidence(report)...)
	issues = append(issues, validateBlockPaintEvidence(report)...)
	issues = append(issues, validateBlockTextEvidence(report)...)
	issues = append(issues, validateBlockLayoutEvidence(report)...)
	issues = append(issues, validateBlockEventFocusEvidence(report)...)
	issues = append(issues, validateBlockStateEvidence(report)...)
	issues = append(issues, validateBlockMotionEvidence(report)...)
	issues = append(issues, validateBlockAssetEvidence(report)...)
	issues = append(issues, validateBlockAccessibilityEvidence(report)...)
	issues = append(issues, validateBlockSystemEvidence(report)...)
	issues = append(issues, validateMorphEvidence(report)...)
	issues = append(issues, validateProductionToolkitEvidence(report)...)
	issues = append(issues, validateBrowserReleaseEvidence(report)...)
	issues = append(issues, validateBrowserSurfaceEvidence(report)...)
	issues = append(issues, validateLinuxReleaseWindowEvidence(report)...)
	issues = append(issues, validateMinimalToolkitEvidence(report)...)
	issues = append(issues, validateAccessibilityTreeEvidence(report)...)
	issues = append(issues, validateAppModelEvidence(report)...)
	issues = append(issues, validateLinuxAppShellEvidence(report)...)
	issues = append(issues, validateSecurityPermissionEvidence(report)...)
	issues = append(issues, validateSurfacePerformanceBudgetEvidence(report)...)
	if report.SurfacePerformanceBudget != nil && !performanceBudgetPeakRSSFieldPresent(raw, true) {
		issues = append(issues, "surface_performance_budget memory peak_rss_bytes field is required")
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}
