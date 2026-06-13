package surface

import (
	"fmt"
	"strings"
)

type LinuxAppShellReport struct {
	Schema            string                         `json:"schema"`
	AppShellLevel     string                         `json:"app_shell_level"`
	ReleaseScope      string                         `json:"release_scope"`
	Source            string                         `json:"source"`
	Module            string                         `json:"module"`
	HostAdapter       string                         `json:"host_adapter"`
	ProductionClaim   bool                           `json:"production_claim"`
	Experimental      bool                           `json:"experimental"`
	WindowLifecycle   []LinuxAppShellLifecycleReport `json:"window_lifecycle"`
	Windows           []LinuxAppShellWindowReport    `json:"windows"`
	ResizeDPI         []LinuxAppShellResizeDPIReport `json:"resize_dpi"`
	CursorTransitions []LinuxAppShellCursorReport    `json:"cursor_transitions"`
	Clipboard         LinuxAppShellCapabilityReport  `json:"clipboard"`
	IME               LinuxAppShellCapabilityReport  `json:"ime"`
	Accessibility     LinuxAppShellCapabilityReport  `json:"accessibility"`
	ShellFeatures     []LinuxAppShellFeatureReport   `json:"shell_features"`
	HostTraces        []LinuxAppShellHostTraceReport `json:"host_traces"`
	NegativeGuards    LinuxAppShellNegativeGuards    `json:"negative_guards"`
}

type LinuxAppShellLifecycleReport struct {
	Order     int    `json:"order"`
	WindowID  string `json:"window_id"`
	Operation string `json:"operation"`
	HostTrace bool   `json:"host_trace"`
	Pass      bool   `json:"pass"`
}

type LinuxAppShellWindowReport struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Role          string `json:"role"`
	BlockRoot     string `json:"block_root"`
	RealWindow    bool   `json:"real_window"`
	Presented     bool   `json:"presented"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	DPIScaleMilli int    `json:"dpi_scale_milli"`
}

type LinuxAppShellResizeDPIReport struct {
	WindowID      string `json:"window_id"`
	Operation     string `json:"operation"`
	BeforeWidth   int    `json:"before_width"`
	BeforeHeight  int    `json:"before_height"`
	AfterWidth    int    `json:"after_width"`
	AfterHeight   int    `json:"after_height"`
	DPIScaleMilli int    `json:"dpi_scale_milli"`
	HostTrace     bool   `json:"host_trace"`
	Pass          bool   `json:"pass"`
}

type LinuxAppShellCursorReport struct {
	WindowID  string `json:"window_id"`
	Cursor    string `json:"cursor"`
	Target    string `json:"target"`
	HostTrace bool   `json:"host_trace"`
	Pass      bool   `json:"pass"`
}

type LinuxAppShellCapabilityReport struct {
	Level          string `json:"level"`
	HostTrace      bool   `json:"host_trace"`
	ArtifactKind   string `json:"artifact_kind"`
	Read           bool   `json:"read,omitempty"`
	Write          bool   `json:"write,omitempty"`
	Start          bool   `json:"start,omitempty"`
	Update         bool   `json:"update,omitempty"`
	Commit         bool   `json:"commit,omitempty"`
	Cancel         bool   `json:"cancel,omitempty"`
	MetadataTree   bool   `json:"metadata_tree,omitempty"`
	PlatformExport bool   `json:"platform_export,omitempty"`
	Pass           bool   `json:"pass"`
}

type LinuxAppShellFeatureReport struct {
	Name             string `json:"name"`
	Status           string `json:"status"`
	Claimed          bool   `json:"claimed"`
	HostTrace        bool   `json:"host_trace"`
	BlockedReason    string `json:"blocked_reason"`
	NoNativeWidgetUI bool   `json:"no_native_widget_ui"`
	Pass             bool   `json:"pass"`
}

type LinuxAppShellHostTraceReport struct {
	Name         string `json:"name"`
	ArtifactKind string `json:"artifact_kind"`
	Path         string `json:"path"`
	Pass         bool   `json:"pass"`
}

type LinuxAppShellNegativeGuards struct {
	NoGTK             bool `json:"no_gtk"`
	NoQT              bool `json:"no_qt"`
	NoNativeWidgets   bool `json:"no_native_widgets"`
	NoElectronRuntime bool `json:"no_electron_runtime"`
	NoReactRuntime    bool `json:"no_react_runtime"`
	NoDOMUI           bool `json:"no_dom_ui"`
	NoUserJS          bool `json:"no_user_js"`
	NoPlatformWidgets bool `json:"no_platform_widgets"`
}

func validateLinuxReleaseWindowEvidence(report Report) []string {
	if !isLinuxReleaseWindowReport(report) {
		return nil
	}
	var issues []string
	if report.Target != "linux-x64" {
		issues = append(issues, fmt.Sprintf("linux release target is %q, want linux-x64", report.Target))
	}
	if !isSurfaceReleaseFormSource(report.Source) {
		issues = append(issues, fmt.Sprintf("linux release source path must match examples/surface_release_form.tetra, got %q", report.Source))
	}
	if report.HostEvidence.Level != "linux-x64-release-window-v1" {
		issues = append(issues, fmt.Sprintf("linux release host_evidence.level is %q, want linux-x64-release-window-v1", report.HostEvidence.Level))
	}
	if report.HostEvidence.Backend != "wayland-shm-rgba-release-v1" {
		issues = append(issues, fmt.Sprintf("linux release host_evidence.backend is %q, want wayland-shm-rgba-release-v1", report.HostEvidence.Backend))
	}
	if !report.HostEvidence.TextInput {
		issues = append(issues, "linux release host_evidence.text_input must be true")
	}
	if !report.HostEvidence.Clipboard {
		issues = append(issues, "linux release host_evidence.clipboard must be true")
	}
	if !report.HostEvidence.Composition {
		issues = append(issues, "linux release host_evidence.composition must be true")
	}
	if !report.HostEvidence.AccessibilityBridge {
		issues = append(issues, "linux release host_evidence.accessibility_bridge must be true")
	}
	if !hasFrameOrderDimensions(report.Frames, 5, 560, 420, 2240) {
		issues = append(issues, "linux release requires order-5 560x420 real-window presented frame evidence")
	}
	for _, kind := range []string{"mouse_up", "key_down", "text_input", "resize", "close"} {
		if !eventKindContains(report.Events, kind) {
			issues = append(issues, fmt.Sprintf("linux release requires %s event evidence", kind))
		}
	}
	for _, required := range []string{
		"linux release window v1 schema",
		"linux release real window presented frame",
		"linux release native pointer key text resize close",
		"linux release clipboard harness",
		"linux release composition harness",
		"linux release accessibility bridge probe",
		"linux release forbids memfd starter promotion",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("linux release report requires %s evidence", required))
		}
	}
	if !hasRuntimeProcessName(report.Processes, "surface linux-x64 release clipboard harness") {
		issues = append(issues, "linux release requires clipboard harness process evidence")
	}
	if !hasRuntimeProcessName(report.Processes, "surface linux-x64 release composition harness") {
		issues = append(issues, "linux release requires composition harness process evidence")
	}
	if !hasRuntimeProcessName(report.Processes, "surface linux accessibility platform probe") {
		issues = append(issues, "linux release requires accessibility platform probe process evidence")
	}
	if !artifactKindContains(report.Artifacts, "linux-accessibility-platform-probe") {
		issues = append(issues, "linux release requires linux-accessibility-platform-probe artifact")
	}
	if report.Toolkit == nil || report.Toolkit.ToolkitLevel != "production-widgets-v1" {
		issues = append(issues, "linux release requires production-widgets-v1 toolkit evidence")
	}
	if report.AccessibilityTree == nil || report.AccessibilityTree.AccessibilityLevel != "platform-bridge-v1" {
		issues = append(issues, "linux release requires platform-bridge-v1 accessibility_tree evidence")
	}
	return issues
}

func validateLinuxAppShellEvidence(report Report) []string {
	if !isLinuxAppShellReport(report) {
		return nil
	}
	var issues []string
	if report.Target != "linux-x64" {
		issues = append(issues, fmt.Sprintf("linux app-shell target is %q, want linux-x64", report.Target))
	}
	if !isSurfaceLinuxAppShellNotesSource(report.Source) {
		issues = append(issues, fmt.Sprintf("linux app-shell source path must match examples/surface_linux_app_shell_notes.tetra, got %q", report.Source))
	}
	if report.HostEvidence.Level != "linux-x64-release-window-v1" {
		issues = append(issues, fmt.Sprintf("linux app-shell host_evidence.level is %q, want linux-x64-release-window-v1", report.HostEvidence.Level))
	}
	if report.HostEvidence.Backend != "wayland-shm-rgba-release-v1" {
		issues = append(issues, fmt.Sprintf("linux app-shell host_evidence.backend is %q, want wayland-shm-rgba-release-v1", report.HostEvidence.Backend))
	}
	for _, check := range []struct {
		name string
		ok   bool
	}{
		{name: "framebuffer", ok: report.HostEvidence.Framebuffer},
		{name: "real_window", ok: report.HostEvidence.RealWindow},
		{name: "native_input", ok: report.HostEvidence.NativeInput},
		{name: "text_input", ok: report.HostEvidence.TextInput},
		{name: "clipboard", ok: report.HostEvidence.Clipboard},
		{name: "composition", ok: report.HostEvidence.Composition},
		{name: "accessibility_bridge", ok: report.HostEvidence.AccessibilityBridge},
	} {
		if !check.ok {
			issues = append(issues, fmt.Sprintf("linux app-shell host_evidence.%s must be true", check.name))
		}
	}
	if report.HostEvidence.UserFacingPlatformWidgets {
		issues = append(issues, "linux app-shell must not expose GTK/Qt/native widget UI or user-facing platform widgets")
	}
	if !hasFrameOrderDimensions(report.Frames, 6, 720, 540, 2880) {
		issues = append(issues, "linux app-shell requires order-6 720x540 presented resize/DPI frame evidence")
	}
	for _, kind := range []string{"mouse_up", "key_down", "text_input", "resize", "close"} {
		if !eventKindContains(report.Events, kind) {
			issues = append(issues, fmt.Sprintf("linux app-shell requires %s event evidence", kind))
		}
	}
	for _, process := range []string{
		"surface linux app-shell host trace",
		"surface linux app-shell window trace",
		"surface linux-x64 release clipboard harness",
		"surface linux-x64 release composition harness",
		"surface linux accessibility platform probe",
	} {
		if !hasRuntimeProcessName(report.Processes, process) {
			issues = append(issues, fmt.Sprintf("linux app-shell requires %s process evidence", process))
		}
	}
	for _, kind := range []string{"linux-app-shell-host-trace", "linux-app-shell-window-trace", "linux-accessibility-platform-probe"} {
		if !artifactKindContains(report.Artifacts, kind) {
			issues = append(issues, fmt.Sprintf("linux app-shell requires %s artifact", kind))
		}
	}
	for _, required := range []string{
		"linux app-shell v1 schema",
		"linux app-shell lifecycle open close reopen",
		"linux app-shell multi-window notes reference",
		"linux app-shell resize dpi cursor trace",
		"linux app-shell clipboard ime accessibility adapters",
		"linux app-shell file dialog notification blocked-pass",
		"linux app-shell electron feature ledger",
		"linux app-shell dialog file picker tray blocked-pass",
		"linux app-shell crash error report scoped adapters",
		"linux app-shell rejects GTK Qt native widget UI",
		"linux app-shell no Electron React DOM application scripting",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("linux app-shell report requires %s evidence", required))
		}
	}
	if report.LinuxAppShell == nil {
		return append(issues, "linux_app_shell evidence is required for examples/surface_linux_app_shell_notes.tetra")
	}
	app := report.LinuxAppShell
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: app.Schema, want: LinuxAppShellSchemaV1},
		{field: "app_shell_level", got: app.AppShellLevel, want: "linux-app-shell-subset-v1"},
		{field: "release_scope", got: app.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
		{field: "module", got: app.Module, want: "lib.core.surface_app_shell"},
		{field: "host_adapter", got: app.HostAdapter, want: "wayland-shm-rgba-release-v1"},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("linux_app_shell %s is %q, want %q", check.field, check.got, check.want))
		}
	}
	if normalizeEvidencePath(app.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("linux_app_shell source %q must match report source %q", app.Source, report.Source))
	}
	if !app.ProductionClaim {
		issues = append(issues, "linux_app_shell production_claim must be true")
	}
	if app.Experimental {
		issues = append(issues, "linux_app_shell experimental must be false")
	}
	issues = append(issues, validateLinuxAppShellLifecycle(app.WindowLifecycle)...)
	issues = append(issues, validateLinuxAppShellWindows(app.Windows)...)
	issues = append(issues, validateLinuxAppShellResizeDPI(app.ResizeDPI)...)
	issues = append(issues, validateLinuxAppShellCursors(app.CursorTransitions)...)
	issues = append(issues, validateLinuxAppShellCapability("clipboard", app.Clipboard, "clipboard-text-v1", "linux-app-shell-host-trace", []string{"read", "write"})...)
	issues = append(issues, validateLinuxAppShellCapability("ime", app.IME, "composition-baseline-v1", "linux-app-shell-host-trace", []string{"start", "update", "commit", "cancel"})...)
	issues = append(issues, validateLinuxAppShellCapability("accessibility", app.Accessibility, "platform-bridge-v1", "linux-accessibility-platform-probe", []string{"metadata_tree", "platform_export"})...)
	issues = append(issues, validateLinuxAppShellFeatures(app.ShellFeatures)...)
	issues = append(issues, validateLinuxAppShellHostTraces(app.HostTraces)...)
	issues = append(issues, validateLinuxAppShellNegativeGuards(app.NegativeGuards)...)
	return issues
}

func validateLinuxAppShellLifecycle(rows []LinuxAppShellLifecycleReport) []string {
	var issues []string
	if len(rows) < 3 {
		issues = append(issues, "linux_app_shell window_lifecycle requires open, close, and reopen evidence")
	}
	seen := map[string]bool{}
	for _, row := range rows {
		if strings.TrimSpace(row.WindowID) == "" {
			issues = append(issues, "linux_app_shell window_lifecycle window_id is required")
		}
		if !row.HostTrace || !row.Pass {
			issues = append(issues, fmt.Sprintf("linux_app_shell lifecycle %s must have host_trace=true and pass=true", row.Operation))
		}
		seen[row.Operation] = true
	}
	for _, op := range []string{"open", "close", "reopen"} {
		if !seen[op] {
			issues = append(issues, fmt.Sprintf("linux_app_shell window_lifecycle missing %s", op))
		}
	}
	return issues
}

func validateLinuxAppShellWindows(rows []LinuxAppShellWindowReport) []string {
	var issues []string
	if len(rows) < 2 {
		issues = append(issues, "linux_app_shell windows requires at least two real windows for multi-window notes")
	}
	seen := map[string]bool{}
	for _, row := range rows {
		if strings.TrimSpace(row.ID) == "" || strings.TrimSpace(row.BlockRoot) == "" {
			issues = append(issues, "linux_app_shell windows require id and block_root")
		}
		if !row.RealWindow || !row.Presented {
			issues = append(issues, fmt.Sprintf("linux_app_shell window %s must be real_window=true and presented=true", row.ID))
		}
		if row.Width <= 0 || row.Height <= 0 || row.DPIScaleMilli <= 0 {
			issues = append(issues, fmt.Sprintf("linux_app_shell window %s requires positive size and dpi_scale_milli", row.ID))
		}
		seen[row.ID] = true
	}
	for _, id := range []string{"notes-main", "notes-inspector"} {
		if !seen[id] {
			issues = append(issues, fmt.Sprintf("linux_app_shell windows missing %s multi-window notes evidence", id))
		}
	}
	return issues
}

func validateLinuxAppShellResizeDPI(rows []LinuxAppShellResizeDPIReport) []string {
	var issues []string
	seen := map[string]bool{}
	for _, row := range rows {
		if !row.HostTrace || !row.Pass {
			issues = append(issues, fmt.Sprintf("linux_app_shell resize_dpi %s must have host_trace=true and pass=true", row.Operation))
		}
		if row.BeforeWidth <= 0 || row.BeforeHeight <= 0 || row.AfterWidth <= 0 || row.AfterHeight <= 0 || row.DPIScaleMilli <= 0 {
			issues = append(issues, fmt.Sprintf("linux_app_shell resize_dpi %s requires positive dimensions and dpi_scale_milli", row.Operation))
		}
		seen[row.Operation] = true
	}
	for _, op := range []string{"resize", "dpi_scale"} {
		if !seen[op] {
			issues = append(issues, fmt.Sprintf("linux_app_shell resize_dpi missing %s", op))
		}
	}
	return issues
}

func validateLinuxAppShellCursors(rows []LinuxAppShellCursorReport) []string {
	var issues []string
	seen := map[string]bool{}
	for _, row := range rows {
		if !row.HostTrace || !row.Pass {
			issues = append(issues, fmt.Sprintf("linux_app_shell cursor %s must have host_trace=true and pass=true", row.Cursor))
		}
		if strings.TrimSpace(row.Target) == "" {
			issues = append(issues, fmt.Sprintf("linux_app_shell cursor %s target is required", row.Cursor))
		}
		seen[row.Cursor] = true
	}
	for _, cursor := range []string{"pointer", "text", "resize"} {
		if !seen[cursor] {
			issues = append(issues, fmt.Sprintf("linux_app_shell cursor_transitions missing %s", cursor))
		}
	}
	return issues
}

func validateLinuxAppShellCapability(name string, cap LinuxAppShellCapabilityReport, wantLevel string, wantArtifactKind string, required []string) []string {
	var issues []string
	if cap.Level != wantLevel {
		issues = append(issues, fmt.Sprintf("linux_app_shell %s level is %q, want %q", name, cap.Level, wantLevel))
	}
	if !cap.HostTrace || !cap.Pass {
		issues = append(issues, fmt.Sprintf("linux_app_shell %s requires host_trace=true and pass=true", name))
	}
	if cap.ArtifactKind != wantArtifactKind {
		issues = append(issues, fmt.Sprintf("linux_app_shell %s artifact_kind is %q, want %q", name, cap.ArtifactKind, wantArtifactKind))
	}
	checks := map[string]bool{
		"read":            cap.Read,
		"write":           cap.Write,
		"start":           cap.Start,
		"update":          cap.Update,
		"commit":          cap.Commit,
		"cancel":          cap.Cancel,
		"metadata_tree":   cap.MetadataTree,
		"platform_export": cap.PlatformExport,
	}
	for _, field := range required {
		if !checks[field] {
			issues = append(issues, fmt.Sprintf("linux_app_shell %s requires %s=true", name, field))
		}
	}
	return issues
}

func validateLinuxAppShellFeatures(rows []LinuxAppShellFeatureReport) []string {
	var issues []string
	features := map[string]LinuxAppShellFeatureReport{}
	for _, row := range rows {
		if strings.TrimSpace(row.Name) == "" {
			issues = append(issues, "linux_app_shell shell_feature name is required")
			continue
		}
		features[row.Name] = row
		if !row.Pass || !row.HostTrace || !row.NoNativeWidgetUI {
			issues = append(issues, fmt.Sprintf("linux_app_shell shell_feature %s requires host_trace=true, no_native_widget_ui=true, and pass=true", row.Name))
		}
		if !linuxAppShellKnownFeature(row.Name) {
			issues = append(issues, fmt.Sprintf("linux_app_shell shell_feature %s is not part of the supported P16 feature ledger", row.Name))
		}
	}
	for _, name := range []string{"window_lifecycle", "multi_window", "clipboard", "ime", "accessibility_bridge"} {
		feature, ok := features[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("linux_app_shell shell_features missing %s", name))
			continue
		}
		if feature.Status != "target_evidenced" || !feature.Claimed || strings.TrimSpace(feature.BlockedReason) != "" {
			issues = append(issues, fmt.Sprintf("linux_app_shell %s must be target_evidenced, claimed=true, and carry no blocked_reason", name))
		}
	}
	for _, name := range []string{"app_menu", "crash_recovery", "error_report"} {
		feature, ok := features[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("linux_app_shell shell_features missing %s", name))
			continue
		}
		if feature.Status != "scoped_adapter" || !feature.Claimed || strings.TrimSpace(feature.BlockedReason) != "" {
			issues = append(issues, fmt.Sprintf("linux_app_shell %s must be a claimed scoped_adapter with target evidence, not GTK/Qt/native widget UI", name))
		}
	}
	for _, name := range []string{"dialog", "file_dialog", "file_picker", "notification", "tray", "deep_link"} {
		feature, ok := features[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("linux_app_shell shell_features missing %s", name))
			continue
		}
		if feature.Status != "blocked_pass" || feature.Claimed || strings.TrimSpace(feature.BlockedReason) == "" {
			issues = append(issues, fmt.Sprintf("linux_app_shell %s requires blocked_pass nonclaim until target evidence exists", name))
		}
	}
	return issues
}

func linuxAppShellKnownFeature(name string) bool {
	switch name {
	case "app_menu",
		"window_lifecycle",
		"multi_window",
		"clipboard",
		"ime",
		"accessibility_bridge",
		"crash_recovery",
		"error_report",
		"dialog",
		"file_dialog",
		"file_picker",
		"notification",
		"tray",
		"deep_link":
		return true
	default:
		return false
	}
}

func validateLinuxAppShellHostTraces(rows []LinuxAppShellHostTraceReport) []string {
	var issues []string
	seen := map[string]bool{}
	for _, row := range rows {
		if !row.Pass || strings.TrimSpace(row.Path) == "" {
			issues = append(issues, fmt.Sprintf("linux_app_shell host_trace %s requires path and pass=true", row.Name))
		}
		seen[row.ArtifactKind] = true
	}
	for _, kind := range []string{"linux-app-shell-host-trace", "linux-app-shell-window-trace", "linux-accessibility-platform-probe"} {
		if !seen[kind] {
			issues = append(issues, fmt.Sprintf("linux_app_shell host_traces missing %s", kind))
		}
	}
	return issues
}

func validateLinuxAppShellNegativeGuards(guards LinuxAppShellNegativeGuards) []string {
	if guards.NoGTK &&
		guards.NoQT &&
		guards.NoNativeWidgets &&
		guards.NoElectronRuntime &&
		guards.NoReactRuntime &&
		guards.NoDOMUI &&
		guards.NoUserJS &&
		guards.NoPlatformWidgets {
		return nil
	}
	return []string{"linux_app_shell negative_guards must reject GTK/Qt/native widget UI, Electron/React runtimes, DOM UI, user JS, and platform widgets"}
}

func isLinuxRealWindowHostEvidenceLevel(level string) bool {
	return level == "linux-x64-real-window" ||
		level == "linux-x64-release-window-v1"
}

func isLinuxReleaseWindowReport(report Report) bool {
	if isLinuxAppShellReport(report) {
		return false
	}
	if report.HostEvidence.Level == "linux-x64-release-window-v1" {
		return true
	}
	return caseNameContains(report.Cases, "linux release window v1 schema")
}

func isLinuxAppShellReport(report Report) bool {
	if isSurfaceLinuxAppShellNotesSource(report.Source) {
		return true
	}
	if report.LinuxAppShell != nil {
		return true
	}
	return caseNameContains(report.Cases, "linux app-shell")
}

func isSurfaceLinuxAppShellNotesSource(source string) bool {
	return strings.HasSuffix(normalizeEvidencePath(source), "examples/surface_linux_app_shell_notes.tetra")
}
