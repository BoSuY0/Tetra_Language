package surface

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	SchemaV1                      = "tetra.surface.runtime.v1"
	ReleaseSchemaV1               = "tetra.surface.release.v1"
	TextInputSchemaV1             = "tetra.surface.text-input.v1"
	ReleaseScopeSurfaceV1LinuxWeb = "surface-v1-linux-web"
)

type Report struct {
	Schema            string                   `json:"schema"`
	Status            string                   `json:"status"`
	Target            string                   `json:"target"`
	Host              string                   `json:"host"`
	Runtime           string                   `json:"runtime"`
	SurfaceSchema     string                   `json:"surface_schema"`
	HostABI           string                   `json:"host_abi"`
	HostEvidence      HostEvidenceReport       `json:"host_evidence"`
	Source            string                   `json:"source"`
	Processes         []ProcessReport          `json:"processes"`
	Artifacts         []ArtifactReport         `json:"artifacts"`
	ArtifactScan      ArtifactScanReport       `json:"artifact_scan"`
	Components        []ComponentReport        `json:"components"`
	ComponentTree     *ComponentTreeReport     `json:"component_tree,omitempty"`
	ComponentTreeAPI  *ComponentTreeAPIReport  `json:"component_tree_api,omitempty"`
	Toolkit           *ToolkitReport           `json:"toolkit,omitempty"`
	AccessibilityTree *AccessibilityTreeReport `json:"accessibility_tree,omitempty"`
	Events            []EventReport            `json:"events"`
	Frames            []FrameReport            `json:"frames"`
	StateTransitions  []StateTransitionReport  `json:"state_transitions"`
	Cases             []CaseReport             `json:"cases"`
}

type ReleaseSummaryReport struct {
	Schema                  string   `json:"schema"`
	ReleaseScope            string   `json:"release_scope"`
	Status                  string   `json:"status"`
	ProductionClaim         bool     `json:"production_claim"`
	Experimental            bool     `json:"experimental"`
	Producer                string   `json:"producer"`
	GitHead                 string   `json:"git_head"`
	Version                 string   `json:"version"`
	GitDirty                bool     `json:"git_dirty"`
	HostOS                  string   `json:"host_os"`
	HostArch                string   `json:"host_arch"`
	GeneratedAtUTC          string   `json:"generated_at_utc"`
	CommandLine             string   `json:"command_line"`
	SupportedTargets        []string `json:"supported_targets"`
	RuntimeTargets          []string `json:"runtime_targets"`
	TestTargets             []string `json:"test_targets"`
	UnsupportedTargets      []string `json:"unsupported_targets"`
	HostABI                 string   `json:"host_abi"`
	Toolkit                 string   `json:"toolkit"`
	TextInput               string   `json:"text_input"`
	Clipboard               string   `json:"clipboard"`
	IME                     string   `json:"ime"`
	Accessibility           string   `json:"accessibility"`
	BrowserSurface          string   `json:"browser_surface"`
	LinuxSurface            string   `json:"linux_surface"`
	ArtifactHashesValidated bool     `json:"artifact_hashes_validated"`
	LegacySidecars          bool     `json:"legacy_sidecars"`
	DOMUI                   bool     `json:"dom_ui"`
	UserJS                  bool     `json:"user_js"`
	PlatformWidgets         bool     `json:"platform_widgets"`
}

type TextInputReport struct {
	Schema                  string                 `json:"schema"`
	Target                  string                 `json:"target"`
	Source                  string                 `json:"source"`
	Level                   string                 `json:"level"`
	Experimental            bool                   `json:"experimental"`
	ProductionClaim         bool                   `json:"production_claim"`
	Storage                 string                 `json:"storage"`
	UTF8Validation          bool                   `json:"utf8_validation"`
	Caret                   bool                   `json:"caret"`
	Selection               bool                   `json:"selection"`
	Backspace               bool                   `json:"backspace"`
	Delete                  bool                   `json:"delete"`
	HomeEnd                 bool                   `json:"home_end"`
	ArrowLeftRight          bool                   `json:"arrow_left_right"`
	CompositionEvents       bool                   `json:"composition_events"`
	CompositionCommit       bool                   `json:"composition_commit"`
	CompositionCancel       bool                   `json:"composition_cancel"`
	ClipboardRead           bool                   `json:"clipboard_read"`
	ClipboardWrite          bool                   `json:"clipboard_write"`
	ClipboardHostABI        bool                   `json:"clipboard_host_abi"`
	ClipboardOwnedCopy      bool                   `json:"clipboard_owned_copy"`
	CompositionTrace        CompositionTraceReport `json:"composition_trace"`
	BorrowedViewStorage     bool                   `json:"borrowed_view_storage"`
	SafeViewLifetimeChecked bool                   `json:"safe_view_lifetime_checked"`
	Processes               []ProcessReport        `json:"processes"`
	Artifacts               []ArtifactReport       `json:"artifacts"`
	ArtifactScan            ArtifactScanReport     `json:"artifact_scan"`
	Cases                   []CaseReport           `json:"cases"`
}

type CompositionTraceReport struct {
	Start  bool `json:"start"`
	Update bool `json:"update"`
	Commit bool `json:"commit"`
	Cancel bool `json:"cancel"`
}

type ProcessReport struct {
	Name             string `json:"name"`
	Kind             string `json:"kind"`
	Path             string `json:"path"`
	Ran              bool   `json:"ran"`
	Pass             bool   `json:"pass"`
	ExitCode         *int   `json:"exit_code,omitempty"`
	ExpectedExitCode *int   `json:"expected_exit_code,omitempty"`
}

type ArtifactReport struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type ArtifactScanReport struct {
	Root           string   `json:"root"`
	FilesChecked   int      `json:"files_checked"`
	ForbiddenPaths []string `json:"forbidden_paths"`
	Pass           bool     `json:"pass"`
}

type HostEvidenceReport struct {
	Level                        string `json:"level"`
	Backend                      string `json:"backend"`
	Framebuffer                  bool   `json:"framebuffer"`
	RealWindow                   bool   `json:"real_window"`
	NativeInput                  bool   `json:"native_input"`
	TextInput                    bool   `json:"text_input,omitempty"`
	Clipboard                    bool   `json:"clipboard,omitempty"`
	Composition                  bool   `json:"composition,omitempty"`
	AccessibilityBridge          bool   `json:"accessibility_bridge,omitempty"`
	BrowserCanvas                bool   `json:"browser_canvas,omitempty"`
	BrowserInput                 bool   `json:"browser_input,omitempty"`
	BrowserClipboard             bool   `json:"browser_clipboard,omitempty"`
	BrowserClipboardHarness      string `json:"browser_clipboard_harness,omitempty"`
	BrowserComposition           bool   `json:"browser_composition,omitempty"`
	BrowserAccessibilitySnapshot bool   `json:"browser_accessibility_snapshot,omitempty"`
	BrowserAccessibilityMirror   bool   `json:"browser_accessibility_mirror,omitempty"`
	UserFacingPlatformWidgets    bool   `json:"user_facing_platform_widgets"`
}

type ComponentReport struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Parent    string            `json:"parent,omitempty"`
	Bounds    RectReport        `json:"bounds"`
	Abilities []string          `json:"abilities"`
	State     map[string]string `json:"state"`
}

type RectReport struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type SizeReport struct {
	W int `json:"w"`
	H int `json:"h"`
}

type ComponentTreeReport struct {
	Schema        string                            `json:"schema"`
	DynamicLevel  string                            `json:"dynamic_level"`
	RootID        int                               `json:"root_id"`
	NodeCount     int                               `json:"node_count"`
	FocusedID     int                               `json:"focused_id"`
	Nodes         []ComponentTreeNodeReport         `json:"nodes"`
	LayoutPasses  []ComponentTreeLayoutPassReport   `json:"layout_passes"`
	DrawOrder     []int                             `json:"draw_order"`
	DispatchPaths []ComponentTreeDispatchPathReport `json:"dispatch_paths"`
	FocusOrder    []int                             `json:"focus_order"`
}

type ComponentTreeNodeReport struct {
	ID         int        `json:"id"`
	Name       string     `json:"name"`
	Kind       string     `json:"kind"`
	ParentID   int        `json:"parent_id"`
	ChildIndex int        `json:"child_index"`
	FirstChild int        `json:"first_child"`
	ChildCount int        `json:"child_count"`
	Focusable  bool       `json:"focusable"`
	Bounds     RectReport `json:"bounds"`
}

type ComponentTreeLayoutPassReport struct {
	ComponentID int        `json:"component_id"`
	Pass        string     `json:"pass"`
	Bounds      RectReport `json:"bounds"`
	Measured    SizeReport `json:"measured"`
}

type ComponentTreeDispatchPathReport struct {
	Event    string `json:"event"`
	TargetID int    `json:"target_id"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Path     []int  `json:"path"`
}

type ComponentTreeAPIReport struct {
	Schema            string                               `json:"schema"`
	APILevel          string                               `json:"api_level"`
	Source            string                               `json:"source"`
	ManualBookkeeping bool                                 `json:"manual_bookkeeping"`
	Builder           ComponentTreeAPIBuilderReport        `json:"builder"`
	Invariants        ComponentTreeAPIInvariantReport      `json:"invariants"`
	LayoutHelpers     []ComponentTreeAPILayoutHelperReport `json:"layout_helpers"`
	FocusHelpers      []ComponentTreeAPIFocusHelperReport  `json:"focus_helpers"`
	HitTests          []ComponentTreeAPIHitTestReport      `json:"hit_tests"`
	DispatchPaths     []ComponentTreeAPIDispatchPathReport `json:"dispatch_paths"`
}

type ComponentTreeAPIBuilderReport struct {
	RootCreatedBy     string `json:"root_created_by"`
	ChildrenCreatedBy string `json:"children_created_by"`
	NodeCount         int    `json:"node_count"`
	Capacity          int    `json:"capacity"`
	OverflowChecked   bool   `json:"overflow_checked"`
}

type ComponentTreeAPIInvariantReport struct {
	TreeValidateRan         bool `json:"tree_validate_ran"`
	TreeValidateStatus      int  `json:"tree_validate_status"`
	ParentChildLinksChecked bool `json:"parent_child_links_checked"`
	ChildIndicesChecked     bool `json:"child_indices_checked"`
	ChildCountChecked       bool `json:"child_count_checked"`
	FirstChildChecked       bool `json:"first_child_checked"`
}

type ComponentTreeAPILayoutHelperReport struct {
	Helper        string `json:"helper"`
	Target        string `json:"target"`
	Pass          string `json:"pass"`
	ChangedBounds bool   `json:"changed_bounds"`
}

type ComponentTreeAPIFocusHelperReport struct {
	Helper string `json:"helper"`
	Before string `json:"before"`
	After  string `json:"after"`
}

type ComponentTreeAPIHitTestReport struct {
	Helper string `json:"helper"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Target string `json:"target"`
	Path   []int  `json:"path"`
}

type ComponentTreeAPIDispatchPathReport struct {
	Helper string `json:"helper"`
	Target string `json:"target"`
	Path   []int  `json:"path"`
}

type ToolkitReport struct {
	Schema                    string                `json:"schema"`
	ToolkitLevel              string                `json:"toolkit_level"`
	ReuseLevel                string                `json:"reuse_level,omitempty"`
	ReleaseScope              string                `json:"release_scope,omitempty"`
	Source                    string                `json:"source"`
	Sources                   []string              `json:"sources,omitempty"`
	Module                    string                `json:"module"`
	StyleModule               string                `json:"style_module,omitempty"`
	Experimental              bool                  `json:"experimental"`
	ProductionClaim           bool                  `json:"production_claim"`
	UsesComponentTreeAPI      bool                  `json:"uses_component_tree_api"`
	ManualBookkeeping         bool                  `json:"manual_bookkeeping"`
	DemoSpecificWidgetStructs bool                  `json:"demo_specific_widget_structs"`
	NoMagicWidgets            bool                  `json:"no_magic_widgets"`
	NoPlatformWidgets         bool                  `json:"no_platform_widgets"`
	NoDOMUI                   bool                  `json:"no_dom_ui"`
	NoUserJS                  bool                  `json:"no_user_js"`
	ExampleCount              int                   `json:"example_count,omitempty"`
	TextBoxCount              int                   `json:"text_box_count,omitempty"`
	ButtonCount               int                   `json:"button_count,omitempty"`
	MultiTextBoxEvidence      bool                  `json:"multi_textbox_evidence,omitempty"`
	MultiFormEvidence         bool                  `json:"multi_form_evidence,omitempty"`
	WidgetSet                 []string              `json:"widget_set,omitempty"`
	StateSet                  []string              `json:"state_set,omitempty"`
	LayoutFeatures            []string              `json:"layout_features,omitempty"`
	Theme                     bool                  `json:"theme,omitempty"`
	SafeTextStorage           bool                  `json:"safe_text_storage,omitempty"`
	Widgets                   []ToolkitWidgetReport `json:"widgets"`
	ReusableSources           []string              `json:"reusable_sources"`
}

type ToolkitWidgetReport struct {
	Name                string `json:"name"`
	Kind                string `json:"kind"`
	NodeID              int    `json:"node_id"`
	Role                string `json:"role,omitempty"`
	Action              string `json:"action,omitempty"`
	Reusable            bool   `json:"reusable"`
	OrdinaryTetraStruct bool   `json:"ordinary_tetra_struct"`
	Editable            bool   `json:"editable,omitempty"`
}

type AccessibilityTreeReport struct {
	Schema                     string                            `json:"schema"`
	AccessibilityLevel         string                            `json:"accessibility_level"`
	ReleaseScope               string                            `json:"release_scope,omitempty"`
	Source                     string                            `json:"source"`
	Module                     string                            `json:"module"`
	WidgetModule               string                            `json:"widget_module"`
	Experimental               bool                              `json:"experimental"`
	ProductionClaim            bool                              `json:"production_claim"`
	PlatformHostIntegration    bool                              `json:"platform_host_integration"`
	DOMARIAIntegration         bool                              `json:"dom_aria_integration"`
	ScreenReaderEvidence       any                               `json:"screen_reader_evidence"`
	MetadataTree               bool                              `json:"metadata_tree,omitempty"`
	PlatformExport             bool                              `json:"platform_export,omitempty"`
	PlatformBridge             string                            `json:"platform_bridge,omitempty"`
	LinuxPlatformProbe         bool                              `json:"linux_platform_probe,omitempty"`
	LinuxProbeArtifact         string                            `json:"linux_probe_artifact,omitempty"`
	BrowserAccessibilitySnap   bool                              `json:"browser_accessibility_snapshot,omitempty"`
	BrowserAccessibilityMirror bool                              `json:"browser_accessibility_mirror,omitempty"`
	DerivedFromComponentTree   bool                              `json:"derived_from_component_tree"`
	UsesComponentTreeAPI       bool                              `json:"uses_component_tree_api"`
	UsesWidgetToolkit          bool                              `json:"uses_widget_toolkit"`
	ManualBookkeeping          bool                              `json:"manual_bookkeeping"`
	NoDOMUI                    bool                              `json:"no_dom_ui"`
	NoUserJS                   bool                              `json:"no_user_js"`
	NoPlatformWidgets          bool                              `json:"no_platform_widgets"`
	NoLegacySidecars           bool                              `json:"no_legacy_sidecars"`
	ComponentTreeSchema        string                            `json:"component_tree_schema"`
	ComponentTreeAPISchema     string                            `json:"component_tree_api_schema"`
	ToolkitSchema              string                            `json:"toolkit_schema"`
	NodeCount                  int                               `json:"node_count"`
	FocusableCount             int                               `json:"focusable_count"`
	LabelCount                 int                               `json:"label_count"`
	TextBoxCount               int                               `json:"textbox_count"`
	ButtonCount                int                               `json:"button_count"`
	StatusCount                int                               `json:"status_count"`
	RolesPresent               []string                          `json:"roles_present"`
	Nodes                      []AccessibilityNodeReport         `json:"nodes"`
	Relationships              []AccessibilityRelationshipReport `json:"relationships"`
	FocusOrder                 []string                          `json:"focus_order"`
	ReadingOrder               []string                          `json:"reading_order"`
	Actions                    []AccessibilityActionReport       `json:"actions"`
	Snapshots                  []AccessibilitySnapshotReport     `json:"snapshots"`
	NegativeGuards             AccessibilityNegativeGuardsReport `json:"negative_guards"`
}

type AccessibilityNodeReport struct {
	ID           int        `json:"id"`
	ComponentID  int        `json:"component_id"`
	ParentID     int        `json:"parent_id"`
	Name         string     `json:"name"`
	Role         string     `json:"role"`
	Bounds       RectReport `json:"bounds"`
	Visible      bool       `json:"visible"`
	Enabled      bool       `json:"enabled"`
	Focusable    bool       `json:"focusable"`
	Focused      bool       `json:"focused"`
	Editable     bool       `json:"editable"`
	Readonly     bool       `json:"readonly"`
	Required     bool       `json:"required"`
	Pressed      bool       `json:"pressed"`
	Invalid      bool       `json:"invalid"`
	LabelFor     string     `json:"label_for,omitempty"`
	LabelledBy   string     `json:"labelled_by,omitempty"`
	ValueKind    string     `json:"value_kind,omitempty"`
	ValueLen     int        `json:"value_len,omitempty"`
	Actions      []string   `json:"actions,omitempty"`
	FocusIndex   int        `json:"focus_index"`
	ReadingIndex int        `json:"reading_index"`
}

type AccessibilityRelationshipReport struct {
	Kind string `json:"kind"`
	From string `json:"from"`
	To   string `json:"to"`
}

type AccessibilityActionReport struct {
	Target   string `json:"target"`
	Action   string `json:"action"`
	Semantic string `json:"semantic"`
}

type AccessibilitySnapshotReport struct {
	Name                       string `json:"name"`
	Generation                 int    `json:"generation"`
	Focused                    string `json:"focused"`
	FocusedComponentID         int    `json:"focused_component_id"`
	FocusedAccessibilityNodeID int    `json:"focused_accessibility_node_id"`
	NameValueLen               int    `json:"name_value_len"`
	EmailValueLen              int    `json:"email_value_len"`
	StatusValue                string `json:"status_value"`
	BoundsChecksum             string `json:"bounds_checksum"`
	MetadataChecksum           string `json:"metadata_checksum"`
	FrameChecksum              string `json:"frame_checksum"`
}

type AccessibilityNegativeGuardsReport struct {
	NoBorrowedViewStorage       bool `json:"no_borrowed_view_storage"`
	ComponentIDAlignmentChecked bool `json:"component_id_alignment_checked"`
	BoundsAlignmentChecked      bool `json:"bounds_alignment_checked"`
	FocusOrderAlignmentChecked  bool `json:"focus_order_alignment_checked"`
	ReadingOrderChecked         bool `json:"reading_order_checked"`
	LabelRelationshipsChecked   bool `json:"label_relationships_checked"`
	StateUpdatesChecked         bool `json:"state_updates_checked"`
	ArtifactScanChecked         bool `json:"artifact_scan_checked"`
}

type EventReport struct {
	Order           int               `json:"order"`
	Kind            string            `json:"kind"`
	TargetComponent string            `json:"target_component"`
	DispatchPath    []string          `json:"dispatch_path"`
	Handled         bool              `json:"handled"`
	Pass            bool              `json:"pass"`
	X               int               `json:"x"`
	Y               int               `json:"y"`
	Key             int               `json:"key"`
	Width           int               `json:"width"`
	Height          int               `json:"height"`
	TimestampMS     int               `json:"timestamp_ms"`
	BufferSlots     []int             `json:"buffer_slots,omitempty"`
	TextLen         int               `json:"text_len,omitempty"`
	TextBytesHex    string            `json:"text_bytes_hex,omitempty"`
	BeforeState     map[string]string `json:"before_state"`
	AfterState      map[string]string `json:"after_state"`
}

type FrameReport struct {
	Order     int    `json:"order"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Stride    int    `json:"stride"`
	Checksum  string `json:"checksum"`
	Presented bool   `json:"presented"`
}

type StateTransitionReport struct {
	Order     int    `json:"order"`
	Component string `json:"component"`
	Field     string `json:"field"`
	Before    string `json:"before"`
	After     string `json:"after"`
	Cause     string `json:"cause"`
}

type CaseReport struct {
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	Ran           bool   `json:"ran"`
	Pass          bool   `json:"pass"`
	ExpectedError string `json:"expected_error,omitempty"`
	Error         string `json:"error,omitempty"`
}

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
	issues = append(issues, validateArtifacts(report.Target, report.Artifacts, report.Processes)...)
	issues = append(issues, validateArtifactScan(report.ArtifactScan, report.Artifacts)...)
	componentIndex, componentIssues := validateComponents(report.Components)
	issues = append(issues, componentIssues...)
	issues = append(issues, validateSourceComponentModel(report.Source, report.Components)...)
	issues = append(issues, validateEvents(report.Events, componentIndex)...)
	issues = append(issues, validateFrames(report.Frames)...)
	issues = append(issues, validateStateTransitions(report.StateTransitions, componentIndex)...)
	issues = append(issues, validateCases(report.Cases)...)
	issues = append(issues, validateTargetRuntimeEvidence(report)...)
	issues = append(issues, validateTextFocusInputEvidence(report, componentIndex)...)
	issues = append(issues, validateComponentTreeEvidence(report)...)
	issues = append(issues, validateProductionToolkitEvidence(report)...)
	issues = append(issues, validateBrowserReleaseEvidence(report)...)
	issues = append(issues, validateLinuxReleaseWindowEvidence(report)...)
	issues = append(issues, validateMinimalToolkitEvidence(report)...)
	issues = append(issues, validateAccessibilityTreeEvidence(report)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func ValidateReleaseSummary(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != ReleaseSchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, ReleaseSchemaV1)
	}

	var report ReleaseSummaryReport
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	issues = append(issues, rejectNonRuntimeEvidence(raw)...)
	if report.Schema != ReleaseSchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, ReleaseSchemaV1))
	}
	if report.ReleaseScope != ReleaseScopeSurfaceV1LinuxWeb {
		issues = append(issues, fmt.Sprintf("release_scope is %q, want %q", report.ReleaseScope, ReleaseScopeSurfaceV1LinuxWeb))
	}
	if report.Status != "current" {
		issues = append(issues, fmt.Sprintf("status is %q, want current", report.Status))
	}
	if !report.ProductionClaim {
		issues = append(issues, "production_claim must be true for Surface v1 release summaries")
	}
	if report.Experimental {
		issues = append(issues, "experimental must be false for Surface v1 release summaries")
	}
	if report.Producer != "scripts/release/surface/release-gate.sh" {
		issues = append(issues, fmt.Sprintf("producer is %q, want scripts/release/surface/release-gate.sh", report.Producer))
	}
	if !isGitHead(report.GitHead) {
		issues = append(issues, "git_head must be a 40-character hex commit")
	}
	if strings.TrimSpace(report.Version) == "" {
		issues = append(issues, "version is required")
	}
	if strings.TrimSpace(report.HostOS) == "" {
		issues = append(issues, "host_os is required")
	}
	if strings.TrimSpace(report.HostArch) == "" {
		issues = append(issues, "host_arch is required")
	}
	if strings.TrimSpace(report.GeneratedAtUTC) == "" || !strings.HasSuffix(report.GeneratedAtUTC, "Z") || !strings.Contains(report.GeneratedAtUTC, "T") {
		issues = append(issues, "generated_at_utc must be an RFC3339 UTC timestamp")
	}
	if !strings.Contains(report.CommandLine, "scripts/release/surface/release-gate.sh") {
		issues = append(issues, "command_line must include scripts/release/surface/release-gate.sh")
	}
	issues = append(issues, validateExactStringList("supported_targets", report.SupportedTargets, []string{"headless", "linux-x64", "wasm32-web"})...)
	issues = append(issues, validateExactStringList("runtime_targets", report.RuntimeTargets, []string{"linux-x64", "wasm32-web"})...)
	issues = append(issues, validateExactStringList("test_targets", report.TestTargets, []string{"headless"})...)
	issues = append(issues, validateExactStringList("unsupported_targets", report.UnsupportedTargets, []string{"macos-x64", "windows-x64", "wasm32-wasi"})...)
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "host_abi", got: report.HostABI, want: "tetra.surface.host.v1"},
		{field: "toolkit", got: report.Toolkit, want: "production-widgets-v1"},
		{field: "text_input", got: report.TextInput, want: "production-text-input-v1"},
		{field: "clipboard", got: report.Clipboard, want: "clipboard-text-v1"},
		{field: "ime", got: report.IME, want: "composition-baseline-v1"},
		{field: "accessibility", got: report.Accessibility, want: "platform-bridge-v1"},
		{field: "browser_surface", got: report.BrowserSurface, want: "browser-canvas-release-v1"},
		{field: "linux_surface", got: report.LinuxSurface, want: "linux-x64-release-window-v1"},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("%s is %q, want %q", check.field, check.got, check.want))
		}
	}
	if !report.ArtifactHashesValidated {
		issues = append(issues, "artifact_hashes_validated must be true")
	}
	if report.LegacySidecars {
		issues = append(issues, "legacy_sidecars must be false")
	}
	if report.DOMUI {
		issues = append(issues, "dom_ui must be false")
	}
	if report.UserJS {
		issues = append(issues, "user_js must be false")
	}
	if report.PlatformWidgets {
		issues = append(issues, "platform_widgets must be false")
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func isGitHead(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, ch := range value {
		if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F') {
			continue
		}
		return false
	}
	return true
}

func ValidateTextInputReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != TextInputSchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, TextInputSchemaV1)
	}

	var report TextInputReport
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	if report.Schema != TextInputSchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, TextInputSchemaV1))
	}
	switch report.Target {
	case "headless", "linux-x64", "wasm32-web":
	default:
		issues = append(issues, fmt.Sprintf("target is %q, want headless, linux-x64, or wasm32-web", report.Target))
	}
	if normalizeEvidencePath(report.Source) != "examples/surface_release_text_input.tetra" {
		issues = append(issues, fmt.Sprintf("source is %q, want examples/surface_release_text_input.tetra", report.Source))
	}
	if report.Level != "production-text-input-v1" {
		issues = append(issues, fmt.Sprintf("level is %q, want production-text-input-v1", report.Level))
	}
	if report.Experimental {
		issues = append(issues, "experimental must be false for production text-input reports")
	}
	if !report.ProductionClaim {
		issues = append(issues, "production_claim must be true for production text-input reports")
	}
	if report.Storage != "owned-utf8-byte-buffer" {
		issues = append(issues, fmt.Sprintf("storage is %q, want owned-utf8-byte-buffer", report.Storage))
	}
	for _, check := range []struct {
		field string
		ok    bool
	}{
		{field: "utf8_validation", ok: report.UTF8Validation},
		{field: "caret", ok: report.Caret},
		{field: "selection", ok: report.Selection},
		{field: "backspace", ok: report.Backspace},
		{field: "delete", ok: report.Delete},
		{field: "home_end", ok: report.HomeEnd},
		{field: "arrow_left_right", ok: report.ArrowLeftRight},
		{field: "composition_events", ok: report.CompositionEvents},
		{field: "composition_commit", ok: report.CompositionCommit},
		{field: "composition_cancel", ok: report.CompositionCancel},
		{field: "clipboard_read", ok: report.ClipboardRead},
		{field: "clipboard_write", ok: report.ClipboardWrite},
		{field: "clipboard_host_abi", ok: report.ClipboardHostABI},
		{field: "clipboard_owned_copy", ok: report.ClipboardOwnedCopy},
		{field: "composition_trace.start", ok: report.CompositionTrace.Start},
		{field: "composition_trace.update", ok: report.CompositionTrace.Update},
		{field: "composition_trace.commit", ok: report.CompositionTrace.Commit},
		{field: "composition_trace.cancel", ok: report.CompositionTrace.Cancel},
		{field: "safe_view_lifetime_checked", ok: report.SafeViewLifetimeChecked},
	} {
		if !check.ok {
			issues = append(issues, fmt.Sprintf("%s must be true", check.field))
		}
	}
	if report.BorrowedViewStorage {
		issues = append(issues, "borrowed_view_storage must be false")
	}
	issues = append(issues, validateProcesses(report.Source, report.Processes)...)
	issues = append(issues, validateArtifacts(report.Target, report.Artifacts, report.Processes)...)
	issues = append(issues, validateArtifactScan(report.ArtifactScan, report.Artifacts)...)
	issues = append(issues, validateCases(report.Cases)...)
	for _, required := range []string{
		"release text input ASCII insertion",
		"release text input UTF-8 insertion",
		"release text input caret home end arrows",
		"release text input selection replacement",
		"release text input backspace delete",
		"release text input clipboard owned copy transfer",
		"release text input composition start update",
		"release text input composition commit",
		"release text input composition cancel",
		"release text input safe view lifetime checked",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("text-input report requires %s evidence", required))
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateExactStringList(field string, got []string, want []string) []string {
	if len(got) != len(want) {
		return []string{fmt.Sprintf("%s = %v, want exactly %v", field, got, want)}
	}
	for i := range want {
		if got[i] != want[i] {
			return []string{fmt.Sprintf("%s = %v, want exactly %v", field, got, want)}
		}
	}
	return nil
}

func validateHostEvidence(report Report) []string {
	var issues []string
	evidence := report.HostEvidence
	if strings.TrimSpace(evidence.Level) == "" {
		issues = append(issues, "host_evidence.level is required")
	}
	if strings.TrimSpace(evidence.Backend) == "" {
		issues = append(issues, "host_evidence.backend is required")
	}
	if evidence.UserFacingPlatformWidgets {
		issues = append(issues, "host_evidence must not expose user-facing platform widgets")
	}

	switch report.Target {
	case "headless":
		if evidence.Level != "deterministic-headless" {
			issues = append(issues, fmt.Sprintf("headless host_evidence.level is %q, want deterministic-headless", evidence.Level))
		}
		if evidence.Backend != "software-rgba" {
			issues = append(issues, fmt.Sprintf("headless host_evidence.backend is %q, want software-rgba", evidence.Backend))
		}
		if !evidence.Framebuffer {
			issues = append(issues, "headless host_evidence requires framebuffer=true")
		}
		if evidence.RealWindow || evidence.NativeInput {
			issues = append(issues, "headless host_evidence must not claim real_window or native_input")
		}
	case "linux-x64":
		switch evidence.Level {
		case "linux-x64-memfd-starter":
			if evidence.Backend != "memfd-rgba" {
				issues = append(issues, fmt.Sprintf("linux-x64 memfd starter host_evidence.backend is %q, want memfd-rgba", evidence.Backend))
			}
			if !evidence.Framebuffer {
				issues = append(issues, "linux-x64 memfd starter host_evidence requires framebuffer=true")
			}
			if evidence.RealWindow || evidence.NativeInput {
				issues = append(issues, "linux-x64 memfd starter host_evidence must not claim real_window or native_input")
			}
		case "linux-x64-real-window":
			if !evidence.Framebuffer || !evidence.RealWindow || !evidence.NativeInput {
				issues = append(issues, "linux-x64 real-window host_evidence requires framebuffer=true, real_window=true, and native_input=true")
			}
			if evidence.Backend == "memfd-rgba" || evidence.Backend == "software-rgba" || evidence.Backend == "node-surface-host" {
				issues = append(issues, fmt.Sprintf("linux-x64 real-window host_evidence.backend %q is not real-window evidence", evidence.Backend))
			}
			if !hasAppProcessWithExpectedExit(report.Processes, "surface linux-x64 real-window probe", 42) {
				issues = append(issues, "linux-x64 real-window host_evidence requires a Surface real-window probe app exiting 42")
			}
			if !caseNameContains(report.Cases, "linux-x64 real-window surface") {
				issues = append(issues, "linux-x64 real-window host_evidence requires linux-x64 real-window surface case evidence")
			}
			if !caseNameContains(report.Cases, "linux-x64 native input event pump") {
				issues = append(issues, "linux-x64 real-window host_evidence requires linux-x64 native input event pump case evidence")
			}
		case "linux-x64-release-window-v1":
			if evidence.Backend != "wayland-shm-rgba-release-v1" {
				issues = append(issues, fmt.Sprintf("linux release host_evidence.backend is %q, want wayland-shm-rgba-release-v1", evidence.Backend))
			}
			if !evidence.Framebuffer || !evidence.RealWindow || !evidence.NativeInput {
				issues = append(issues, "linux release host_evidence requires framebuffer=true, real_window=true, and native_input=true")
			}
			if !evidence.TextInput {
				issues = append(issues, "linux release host_evidence.text_input must be true")
			}
			if !evidence.Clipboard {
				issues = append(issues, "linux release host_evidence.clipboard must be true")
			}
			if !evidence.Composition {
				issues = append(issues, "linux release host_evidence.composition must be true")
			}
			if !evidence.AccessibilityBridge {
				issues = append(issues, "linux release host_evidence.accessibility_bridge must be true")
			}
			if !caseNameContains(report.Cases, "linux release real window presented frame") {
				issues = append(issues, "linux release host_evidence requires real window presented frame case evidence")
			}
			if !caseNameContains(report.Cases, "linux release accessibility bridge probe") {
				issues = append(issues, "linux release host_evidence requires accessibility bridge probe case evidence")
			}
		default:
			issues = append(issues, fmt.Sprintf("linux-x64 host_evidence.level is %q, want linux-x64-memfd-starter, linux-x64-real-window, or linux-x64-release-window-v1", evidence.Level))
		}
	case "wasm32-web":
		switch evidence.Level {
		case "wasm32-web-compiler-owned-loader":
			if evidence.Backend != "node-surface-host" {
				issues = append(issues, fmt.Sprintf("wasm32-web starter host_evidence.backend is %q, want node-surface-host", evidence.Backend))
			}
			if !evidence.Framebuffer {
				issues = append(issues, "wasm32-web starter host_evidence requires framebuffer=true")
			}
			if evidence.RealWindow || evidence.NativeInput {
				issues = append(issues, "wasm32-web starter host_evidence must not claim browser canvas native input")
			}
		case "wasm32-web-browser-canvas-input":
			if evidence.Backend != "browser-canvas-rgba" {
				issues = append(issues, fmt.Sprintf("wasm32-web browser canvas host_evidence.backend is %q, want browser-canvas-rgba", evidence.Backend))
			}
			if !evidence.Framebuffer || !evidence.NativeInput {
				issues = append(issues, "wasm32-web browser canvas host_evidence requires framebuffer=true and native_input=true")
			}
			if evidence.RealWindow {
				issues = append(issues, "wasm32-web browser canvas host_evidence must not claim OS real_window")
			}
			if !caseNameContains(report.Cases, "wasm32-web browser canvas surface") {
				issues = append(issues, "wasm32-web browser canvas host_evidence requires browser canvas surface case evidence")
			}
			if !caseNameContains(report.Cases, "wasm32-web browser canvas RGBA readback") {
				issues = append(issues, "wasm32-web browser canvas host_evidence requires canvas RGBA readback case evidence")
			}
		case "wasm32-web-browser-canvas-release-v1":
			if evidence.Backend != "browser-canvas-rgba-accessible" {
				issues = append(issues, fmt.Sprintf("browser release host_evidence.backend is %q, want browser-canvas-rgba-accessible", evidence.Backend))
			}
			if !evidence.Framebuffer || !evidence.NativeInput {
				issues = append(issues, "browser release host_evidence requires framebuffer=true and native_input=true")
			}
			if evidence.RealWindow {
				issues = append(issues, "browser release host_evidence must not claim OS real_window")
			}
			if !evidence.BrowserCanvas {
				issues = append(issues, "browser release host_evidence.browser_canvas must be true")
			}
			if !evidence.BrowserInput {
				issues = append(issues, "browser release host_evidence.browser_input must be true")
			}
			if !evidence.BrowserClipboard {
				issues = append(issues, "browser release host_evidence.browser_clipboard must be true")
			}
			if evidence.BrowserClipboardHarness != "deterministic-browser-clipboard-v1" {
				issues = append(issues, fmt.Sprintf("browser release host_evidence.browser_clipboard_harness is %q, want deterministic-browser-clipboard-v1", evidence.BrowserClipboardHarness))
			}
			if !evidence.BrowserComposition {
				issues = append(issues, "browser release host_evidence.browser_composition must be true")
			}
			if !evidence.BrowserAccessibilitySnapshot {
				issues = append(issues, "browser release host_evidence.browser_accessibility_snapshot must be true")
			}
			if !evidence.BrowserAccessibilityMirror {
				issues = append(issues, "browser release host_evidence.browser_accessibility_mirror must be true")
			}
			if !caseNameContains(report.Cases, "wasm32-web browser canvas surface") {
				issues = append(issues, "browser release host_evidence requires browser canvas surface case evidence")
			}
			if !caseNameContains(report.Cases, "wasm32-web browser canvas RGBA readback") {
				issues = append(issues, "browser release host_evidence requires canvas RGBA readback case evidence")
			}
		default:
			issues = append(issues, fmt.Sprintf("wasm32-web host_evidence.level is %q, want wasm32-web-compiler-owned-loader, wasm32-web-browser-canvas-input, or wasm32-web-browser-canvas-release-v1", evidence.Level))
		}
	}
	return issues
}

func decodeSchema(raw []byte) (string, error) {
	var header struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return "", err
	}
	if strings.TrimSpace(header.Schema) == "" {
		return "", errors.New("schema is required")
	}
	return header.Schema, nil
}

func decodeStrict(raw []byte, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("unexpected trailing JSON")
	}
	return nil
}

func rejectNonRuntimeEvidence(raw []byte) []string {
	lower := strings.ToLower(string(raw))
	forbidden := []string{
		"metadata-only",
		"node-only",
		"web-only",
		"dom-only",
		"sidecar-only",
		"docs-only",
		"build-only",
		"stale",
		" fake",
		"fake/",
		"\"fake\"",
		" mock",
		"mock/",
		"\"mock\"",
		"placeholder",
		".ui.html",
		".ui.web.mjs",
		".ui.json",
		"tetra.ui.v1",
		"dom ui",
		"html ui",
		"user javascript",
		"user js",
		"react component",
		"gtk widget",
		"qt widget",
		"winui",
		"cocoa",
	}
	var issues []string
	for _, marker := range forbidden {
		if strings.Contains(lower, marker) {
			issues = append(issues, fmt.Sprintf("report contains forbidden non-runtime evidence marker %q", strings.Trim(marker, " /\"")))
		}
	}
	return issues
}

func validateProcesses(source string, processes []ProcessReport) []string {
	var issues []string
	if len(processes) < 3 {
		issues = append(issues, fmt.Sprintf("process evidence has %d entries, want build, app, and runtime processes", len(processes)))
	}
	seen := map[string]bool{}
	seenBuild := false
	seenBuildForSource := false
	seenApp := false
	seenComponentApp := false
	seenRuntime := false
	for _, process := range processes {
		if strings.TrimSpace(process.Name) == "" {
			issues = append(issues, "process name is required")
		} else if seen[process.Name] {
			issues = append(issues, fmt.Sprintf("duplicate process %s", process.Name))
		}
		seen[process.Name] = true
		switch process.Kind {
		case "build":
			seenBuild = true
			if processReferencesSource(process.Path, source) {
				seenBuildForSource = true
			}
		case "app":
			seenApp = true
			if isSurfaceComponentAppProcess(process) {
				seenComponentApp = true
			}
		case "runtime":
			seenRuntime = true
		default:
			issues = append(issues, fmt.Sprintf("process %s kind is %q, want build, app, or runtime", process.Name, process.Kind))
		}
		if strings.TrimSpace(process.Path) == "" {
			issues = append(issues, fmt.Sprintf("process %s path is required", process.Name))
		} else if process.Kind == "app" && sourceLikeEvidencePath(process.Path) {
			issues = append(issues, fmt.Sprintf("process %s path %q is not executable Surface app process evidence", process.Name, process.Path))
		}
		if !process.Ran {
			issues = append(issues, fmt.Sprintf("process %s did not run", process.Name))
		}
		if !process.Pass {
			issues = append(issues, fmt.Sprintf("process %s did not pass", process.Name))
		}
		if process.ExitCode == nil {
			issues = append(issues, fmt.Sprintf("process %s missing exit_code", process.Name))
			continue
		}
		wantExit := 0
		if process.ExpectedExitCode != nil {
			wantExit = *process.ExpectedExitCode
		}
		if *process.ExitCode != wantExit {
			issues = append(issues, fmt.Sprintf("process %s exit_code = %d, want %d", process.Name, *process.ExitCode, wantExit))
		}
	}
	if !seenBuild {
		issues = append(issues, "process evidence missing build process")
	}
	if !seenBuildForSource {
		issues = append(issues, fmt.Sprintf("process evidence missing build process for reported source %q", source))
	}
	if !seenApp {
		issues = append(issues, "process evidence missing executable Surface app process")
	}
	if !seenComponentApp {
		issues = append(issues, "process evidence missing executable Surface component app process with expected exit code 1")
	}
	if !seenRuntime {
		issues = append(issues, "process evidence missing Surface runtime process")
	}
	return issues
}

func processReferencesSource(path string, source string) bool {
	source = normalizeEvidencePath(source)
	if source == "" {
		return false
	}
	path = normalizeEvidencePath(path)
	return strings.Contains(path, source)
}

func isSurfaceComponentAppProcess(process ProcessReport) bool {
	name := strings.ToLower(strings.TrimSpace(process.Name))
	if process.Kind != "app" || !strings.Contains(name, "surface") || !strings.Contains(name, "component app") {
		return false
	}
	if process.ExitCode == nil || process.ExpectedExitCode == nil {
		return false
	}
	if *process.ExitCode == 1 && *process.ExpectedExitCode == 1 {
		return true
	}
	return strings.Contains(name, "browser canvas") && *process.ExitCode == 0 && *process.ExpectedExitCode == 0
}

func normalizeEvidencePath(path string) string {
	path = strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	return path
}

func validateArtifacts(target string, artifacts []ArtifactReport, processes []ProcessReport) []string {
	var issues []string
	if len(artifacts) == 0 {
		issues = append(issues, "artifact evidence is required")
	}
	seenPath := map[string]bool{}
	seenComponentAppArtifact := false
	seenCompilerOwnedLoaderArtifact := false
	seenRunnerTraceArtifact := false
	for _, artifact := range artifacts {
		kind := strings.TrimSpace(artifact.Kind)
		path := normalizeEvidencePath(artifact.Path)
		if kind == "" {
			issues = append(issues, "artifact kind is required")
		}
		if path == "" {
			issues = append(issues, fmt.Sprintf("artifact %s path is required", kind))
		} else if seenPath[path] {
			issues = append(issues, fmt.Sprintf("duplicate artifact path %s", artifact.Path))
		}
		seenPath[path] = true
		issues = append(issues, validateSurfaceArtifactPath(kind, path)...)
		if !validSHA256Digest(artifact.SHA256) {
			issues = append(issues, fmt.Sprintf("artifact %s sha256 must be sha256:<64 hex>", artifact.Path))
		}
		if artifact.Size <= 0 {
			issues = append(issues, fmt.Sprintf("artifact %s size must be positive", artifact.Path))
		}
		if kind == "component-app" && artifactReferencedByComponentAppProcess(path, processes) {
			seenComponentAppArtifact = true
		}
		if kind == "compiler-owned-loader" && strings.HasSuffix(strings.ToLower(path), ".mjs") {
			seenCompilerOwnedLoaderArtifact = true
		}
		if kind == "runner-trace" && strings.HasSuffix(strings.ToLower(path), "surface-runner-trace.json") {
			seenRunnerTraceArtifact = true
		}
	}
	if !seenComponentAppArtifact {
		issues = append(issues, "artifact evidence missing Surface component app artifact hash linked to Surface component app process")
	}
	if target == "wasm32-web" && !seenCompilerOwnedLoaderArtifact {
		issues = append(issues, "wasm32-web artifact evidence missing compiler-owned loader artifact")
	}
	if (target == "headless" || target == "wasm32-web") && !seenRunnerTraceArtifact {
		issues = append(issues, fmt.Sprintf("%s artifact evidence missing Surface runner trace artifact", target))
	}
	return issues
}

func validateSurfaceArtifactPath(kind string, path string) []string {
	lower := strings.ToLower(path)
	var issues []string
	if strings.Contains(lower, ".ui.") {
		issues = append(issues, fmt.Sprintf("artifact %s must not be a legacy UI sidecar", path))
	}
	if strings.HasSuffix(lower, ".html") {
		issues = append(issues, fmt.Sprintf("artifact %s must not be generated HTML UI", path))
	}
	if strings.HasSuffix(lower, ".js") {
		issues = append(issues, fmt.Sprintf("artifact %s must not be generated JavaScript UI", path))
	}
	if strings.HasSuffix(lower, ".mjs") && kind != "compiler-owned-loader" {
		issues = append(issues, fmt.Sprintf("artifact %s .mjs is only allowed for compiler-owned-loader evidence", path))
	}
	for _, forbidden := range []struct {
		suffix string
		model  string
	}{
		{suffix: ".jsx", model: "React"},
		{suffix: ".tsx", model: "React"},
		{suffix: ".qml", model: "Qt"},
		{suffix: ".xaml", model: "WinUI"},
		{suffix: ".xib", model: "Cocoa"},
		{suffix: ".storyboard", model: "Cocoa"},
		{suffix: ".glade", model: "GTK"},
	} {
		if strings.HasSuffix(lower, forbidden.suffix) {
			issues = append(issues, fmt.Sprintf("artifact %s must not be %s user-facing UI evidence", path, forbidden.model))
		}
	}
	if kind == "compiler-owned-loader" && !strings.HasSuffix(lower, ".mjs") {
		issues = append(issues, fmt.Sprintf("compiler-owned loader artifact %s must be a .mjs loader", path))
	}
	return issues
}

func validateArtifactScan(scan ArtifactScanReport, artifacts []ArtifactReport) []string {
	var issues []string
	root := normalizeEvidencePath(scan.Root)
	if root == "" {
		issues = append(issues, "artifact_scan.root is required")
	}
	if scan.FilesChecked <= 0 {
		issues = append(issues, "artifact_scan.files_checked must be positive")
	}
	if len(artifacts) > 0 && scan.FilesChecked < len(artifacts) {
		issues = append(issues, fmt.Sprintf("artifact_scan.files_checked = %d, want at least %d reported artifacts", scan.FilesChecked, len(artifacts)))
	}
	if !scan.Pass {
		issues = append(issues, "artifact_scan.pass must be true")
	}
	if len(scan.ForbiddenPaths) > 0 {
		issues = append(issues, fmt.Sprintf("artifact_scan forbidden paths must be empty, got %d", len(scan.ForbiddenPaths)))
	}
	for _, path := range scan.ForbiddenPaths {
		if strings.TrimSpace(path) == "" {
			issues = append(issues, "artifact_scan forbidden path must not be empty")
		}
	}
	for _, artifact := range artifacts {
		path := normalizeEvidencePath(artifact.Path)
		if root == "" || path == "" {
			continue
		}
		if !evidencePathUnderRoot(path, root) {
			issues = append(issues, fmt.Sprintf("artifact %s is outside artifact_scan.root %s", artifact.Path, scan.Root))
		}
	}
	return issues
}

func evidencePathUnderRoot(path string, root string) bool {
	path = strings.TrimSuffix(normalizeEvidencePath(path), "/")
	root = strings.TrimSuffix(normalizeEvidencePath(root), "/")
	return path == root || strings.HasPrefix(path, root+"/")
}

func artifactReferencedByComponentAppProcess(artifactPath string, processes []ProcessReport) bool {
	for _, process := range processes {
		if !isSurfaceComponentAppProcess(process) {
			continue
		}
		if strings.Contains(normalizeEvidencePath(process.Path), artifactPath) {
			return true
		}
	}
	return false
}

func validSHA256Digest(value string) bool {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "sha256:") {
		return false
	}
	hexDigest := strings.TrimPrefix(value, "sha256:")
	if len(hexDigest) != 64 {
		return false
	}
	_, err := hex.DecodeString(hexDigest)
	return err == nil
}

func validateTargetRuntimeEvidence(report Report) []string {
	var issues []string
	switch report.Target {
	case "headless":
		if report.Runtime != "surface-headless" {
			issues = append(issues, fmt.Sprintf("headless target runtime is %q, want surface-headless", report.Runtime))
		}
		if !hasRuntimeProcessName(report.Processes, "headless") {
			issues = append(issues, "headless target requires a headless Surface runtime process")
		}
		if !caseNameContains(report.Cases, "headless event dispatch") {
			issues = append(issues, "headless target requires headless event dispatch evidence")
		}
		if !caseNameContains(report.Cases, "headless actual runner trace") {
			issues = append(issues, "headless target requires headless actual runner trace evidence")
		}
		if isAccessibilityMetadataReport(report) {
			if !hasFrameOrderDimensions(report.Frames, 5, 480, 320, 1920) {
				issues = append(issues, "headless accessibility metadata target requires order-5 480x320 resized headless runner trace frame evidence")
			}
		} else if isProductionToolkitReport(report) {
			if !hasFrameOrderDimensions(report.Frames, 5, 560, 420, 2240) {
				issues = append(issues, "headless production toolkit target requires order-5 560x420 resized headless runner trace frame evidence")
			}
		} else if isToolkitReuseReport(report) {
			if !hasFrameOrderDimensions(report.Frames, 5, 480, 320, 1920) {
				issues = append(issues, "headless toolkit reuse target requires order-5 480x320 resized headless runner trace frame evidence")
			}
		} else if isMinimalToolkitReport(report) {
			if !hasFrameOrderDimensions(report.Frames, 4, 400, 240, 1600) {
				issues = append(issues, "headless minimal toolkit target requires order-4 400x240 resized headless runner trace frame evidence")
			}
		} else if isComponentTreeReport(report) {
			if !hasFrameOrderDimensions(report.Frames, 2, 400, 240, 1600) {
				issues = append(issues, "headless component tree target requires order-2 400x240 resized headless runner trace frame evidence")
			}
		} else if isTextFocusInputReport(report) {
			if !hasFrameOrderDimensions(report.Frames, 2, 400, 240, 1600) {
				issues = append(issues, "headless text focus input target requires order-2 400x240 resized headless runner trace frame evidence")
			}
		} else if !hasFrameOrderDimensions(report.Frames, 2, 320, 200, 1280) {
			issues = append(issues, "headless target requires order-2 320x200 headless runner trace frame evidence")
		}
	case "linux-x64":
		if report.Runtime != "surface-linux-x64" {
			issues = append(issues, fmt.Sprintf("linux-x64 target runtime is %q, want surface-linux-x64", report.Runtime))
		}
		if !hasRuntimeProcessName(report.Processes, "linux-x64") {
			issues = append(issues, "linux-x64 target requires a linux-x64 Surface runtime process")
		}
		if isLinuxRealWindowHostEvidenceLevel(report.HostEvidence.Level) {
			if !hasAppProcessWithExpectedExit(report.Processes, "surface linux-x64 real-window probe", 42) {
				issues = append(issues, "linux-x64 real-window target requires a Surface real-window probe app exiting 42")
			}
			if !caseNameContains(report.Cases, "linux-x64 real-window surface") {
				issues = append(issues, "linux-x64 real-window target requires real-window surface evidence")
			}
			if !caseNameContains(report.Cases, "linux-x64 native input event pump") {
				issues = append(issues, "linux-x64 real-window target requires native input event pump evidence")
			}
			if !caseNameContains(report.Cases, "linux-x64 real-window resize event") {
				issues = append(issues, "linux-x64 real-window target requires resize event evidence")
			}
			if !caseNameContains(report.Cases, "linux-x64 real-window close event") {
				issues = append(issues, "linux-x64 real-window target requires close event evidence")
			}
			if isLinuxReleaseWindowReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 5, 560, 420, 2240) {
					issues = append(issues, "linux-x64 release-window target requires order-5 560x420 presented window frame evidence")
				}
			} else if isAccessibilityMetadataReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 5, 480, 320, 1920) {
					issues = append(issues, "linux-x64 real-window accessibility metadata target requires order-5 480x320 presented window frame evidence")
				}
			} else if isProductionToolkitReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 5, 560, 420, 2240) {
					issues = append(issues, "linux-x64 real-window production toolkit target requires order-5 560x420 presented window frame evidence")
				}
			} else if isToolkitReuseReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 5, 480, 320, 1920) {
					issues = append(issues, "linux-x64 real-window toolkit reuse target requires order-5 480x320 presented window frame evidence")
				}
			} else if !hasFrameOrderDimensions(report.Frames, 5, 400, 240, 1600) {
				issues = append(issues, "linux-x64 real-window target requires order-5 400x240 presented window frame evidence")
			}
		} else {
			if !hasAppProcessWithExpectedExit(report.Processes, "surface linux-x64 host probe", 42) {
				issues = append(issues, "linux-x64 target requires a Surface Host ABI probe app exiting 42")
			}
			if !caseNameContains(report.Cases, "linux-x64 Surface Host ABI") {
				issues = append(issues, "linux-x64 target requires linux-x64 Surface Host ABI evidence")
			}
			if !hasAppProcessWithExpectedExit(report.Processes, "surface linux-x64 event sequence probe", 42) {
				issues = append(issues, "linux-x64 target requires a Surface event sequence probe app exiting 42")
			}
			if !caseNameContains(report.Cases, "linux-x64 host event sequence") {
				issues = append(issues, "linux-x64 target requires linux-x64 host event sequence evidence")
			}
			if !caseNameContains(report.Cases, "linux-x64 app-presented RGBA checksum") {
				issues = append(issues, "linux-x64 target requires app-presented RGBA checksum evidence")
			}
			if !hasFrameDimensions(report.Frames, 2, 2, 8) {
				issues = append(issues, "linux-x64 target requires a 2x2 app-presented RGBA checksum frame")
			}
			if !hasAppProcessWithExpectedExit(report.Processes, "surface linux-x64 counter app presented frame probe", -1) {
				issues = append(issues, "linux-x64 target requires a counter component app-presented frame probe process")
			}
			if !caseNameContains(report.Cases, "linux-x64 counter component app-presented frame") {
				issues = append(issues, "linux-x64 target requires counter component app-presented frame evidence")
			}
			if !hasFrameOrderDimensions(report.Frames, 4, 320, 200, 1280) {
				issues = append(issues, "linux-x64 target requires order-4 320x200 counter component app-presented frame evidence")
			}
		}
		if caseNameContains(report.Cases, "headless") {
			issues = append(issues, "linux-x64 target must not use headless runtime case evidence")
		}
	case "wasm32-web":
		if report.Runtime != "surface-wasm32-web" {
			issues = append(issues, fmt.Sprintf("wasm32-web target runtime is %q, want surface-wasm32-web", report.Runtime))
		}
		if !hasRuntimeProcessName(report.Processes, "wasm32-web") {
			issues = append(issues, "wasm32-web target requires a wasm32-web Surface runtime process")
		}
		if !hasProcessNameAndPathMarkers(report.Processes, "runtime", "surface wasm32-web import validator", "validate-wasm-imports", "--target wasm32-web") {
			issues = append(issues, "wasm32-web target requires validate-wasm-imports runtime process for Surface Host ABI import allowlist")
		}
		if !caseNameContains(report.Cases, "wasm32-web Surface Host ABI imports") {
			issues = append(issues, "wasm32-web target requires wasm32-web Surface Host ABI import evidence")
		}
		if !caseNameContains(report.Cases, "compiler-owned wasm Surface loader") {
			issues = append(issues, "wasm32-web target requires compiler-owned wasm Surface loader evidence")
		}
		if isBrowserCanvasHostEvidenceLevel(report.HostEvidence.Level) {
			for _, required := range []string{
				"wasm32-web browser canvas surface",
				"wasm32-web browser canvas RGBA readback",
				"wasm32-web browser canvas pointer input",
				"wasm32-web browser canvas keyboard input",
				"wasm32-web browser canvas resize input",
				"wasm32-web browser canvas text input",
				"compiler-owned browser canvas Surface host",
			} {
				if !caseNameContains(report.Cases, required) {
					issues = append(issues, fmt.Sprintf("wasm32-web browser canvas target requires %s evidence", required))
				}
			}
			if !hasProcessNameAndPathMarkers(report.Processes, "app", "surface wasm32-web browser canvas component app", "chromium") &&
				!hasProcessNameAndPathMarkers(report.Processes, "app", "surface wasm32-web browser canvas component app", "chrome") {
				issues = append(issues, "wasm32-web browser canvas target requires Chromium-compatible browser app process evidence")
			}
			for _, kind := range []string{"mouse_up", "key_down", "resize", "text_input"} {
				if !eventKindContains(report.Events, kind) {
					issues = append(issues, fmt.Sprintf("wasm32-web browser canvas target requires %s event evidence", kind))
				}
			}
			if isAccessibilityMetadataReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 5, 480, 320, 1920) {
					issues = append(issues, "wasm32-web browser canvas accessibility metadata target requires order-5 480x320 canvas readback frame evidence")
				}
			} else if isProductionToolkitReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 5, 560, 420, 2240) {
					issues = append(issues, "wasm32-web browser canvas production toolkit target requires order-5 560x420 canvas readback frame evidence")
				}
			} else if isToolkitReuseReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 5, 480, 320, 1920) {
					issues = append(issues, "wasm32-web browser canvas toolkit reuse target requires order-5 480x320 canvas readback frame evidence")
				}
			} else if !hasFrameOrderDimensions(report.Frames, 5, 400, 240, 1600) {
				issues = append(issues, "wasm32-web browser canvas target requires order-5 400x240 canvas readback frame evidence")
			}
		} else {
			if !caseNameContains(report.Cases, "wasm32-web actual presented frame trace") {
				issues = append(issues, "wasm32-web target requires actual presented frame trace evidence")
			}
			if !hasFrameOrderDimensions(report.Frames, 4, 320, 200, 1280) {
				issues = append(issues, "wasm32-web target requires order-4 320x200 actual presented frame trace evidence")
			}
		}
		if caseNameContains(report.Cases, "headless") {
			issues = append(issues, "wasm32-web target must not use headless runtime case evidence")
		}
	}
	return issues
}

func validateTextFocusInputEvidence(report Report, components map[string]ComponentReport) []string {
	if !isTextFocusInputReport(report) {
		return nil
	}
	var issues []string
	for _, required := range []string{
		"text focus input click focuses TextBox",
		"text focus input Tab changes focus",
		"text focus input keyboard routes only focused component",
		"text focus input text insertion",
		"text focus input caret movement",
		"text focus input backspace delete",
		"text focus input resize preserves focus",
		"text focus input rendered frame update",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("text focus input report requires %s evidence", required))
		}
	}
	textBox, ok := components["TextBox"]
	if !ok {
		issues = append(issues, "text focus input report requires TextBox component evidence")
	} else {
		if textBox.State["buffer"] == "" {
			issues = append(issues, "TextBox state requires edited component-owned buffer")
		}
		if textBox.State["caret"] == "" {
			issues = append(issues, "TextBox state requires caret evidence")
		}
		if textBox.State["backspace_count"] == "" || textBox.State["delete_count"] == "" {
			issues = append(issues, "TextBox state requires backspace/delete evidence")
		}
	}
	button, ok := components["SubmitButton"]
	if !ok {
		issues = append(issues, "text focus input report requires SubmitButton component evidence")
	} else if button.State["press_count"] == "" {
		issues = append(issues, "SubmitButton state requires focused keyboard press evidence")
	}
	if !hasEventTargetKind(report.Events, "TextBox", "mouse_up") {
		issues = append(issues, "text focus input report requires mouse_up targeted to TextBox")
	}
	if !hasEventTargetKind(report.Events, "TextBox", "text_input") {
		issues = append(issues, "text focus input report requires text_input targeted to TextBox")
	}
	if !hasKeyEvent(report.Events, 9) {
		issues = append(issues, "text focus input report requires Tab key focus routing evidence")
	}
	if !hasKeyEvent(report.Events, 37) && !hasKeyEvent(report.Events, 39) {
		issues = append(issues, "text focus input report requires caret movement key evidence")
	}
	if !hasKeyEvent(report.Events, 8) || !hasKeyEvent(report.Events, 46) {
		issues = append(issues, "text focus input report requires backspace and delete key evidence")
	}
	if !hasEventTargetKind(report.Events, "SubmitButton", "key_down") {
		issues = append(issues, "text focus input report requires keyboard event routed to focused SubmitButton")
	}
	if !hasResizePreservingFocus(report.Events) {
		issues = append(issues, "text focus input report requires resize preserving focused component state")
	}
	if !hasTransition(report.StateTransitions, "TextBox", "buffer") || !hasTransition(report.StateTransitions, "TextBox", "caret") {
		issues = append(issues, "text focus input report requires TextBox buffer and caret state transitions")
	}
	if !hasTransition(report.StateTransitions, "TextInputApp", "focused_component") {
		issues = append(issues, "text focus input report requires focus manager state transition")
	}
	return issues
}

func isTextFocusInputReport(report Report) bool {
	if strings.Contains(strings.ToLower(report.Source), "surface_textbox_app") {
		return true
	}
	return caseNameContains(report.Cases, "text focus input")
}

func validateComponentTreeEvidence(report Report) []string {
	if !isComponentTreeReport(report) {
		return nil
	}
	var issues []string
	accessibility := isAccessibilityMetadataReport(report) && !isLinuxReleaseWindowReport(report)
	releaseAccessibility := isSurfaceReleaseAccessibilitySource(report.Source) || isPlatformBridgeAccessibilityReport(report)
	productionToolkit := isProductionToolkitReport(report)
	minimalToolkit := isMinimalToolkitReport(report)
	toolkitReuse := isToolkitReuseReport(report)
	if accessibility {
		if !isSurfaceAccessibilitySettingsSource(report.Source) && !isSurfaceReleaseAccessibilitySource(report.Source) {
			issues = append(issues, fmt.Sprintf("component_tree accessibility source path must match examples/surface_accessibility_settings.tetra or examples/surface_release_accessibility.tetra, got %q", report.Source))
		}
	} else if productionToolkit {
		if !isSurfaceReleaseFormSource(report.Source) {
			issues = append(issues, fmt.Sprintf("component_tree production toolkit source path must match examples/surface_release_form.tetra, got %q", report.Source))
		}
	} else if toolkitReuse {
		if !isSurfaceToolkitSettingsSource(report.Source) {
			issues = append(issues, fmt.Sprintf("component_tree toolkit reuse source path must match examples/surface_toolkit_settings.tetra, got %q", report.Source))
		}
	} else if minimalToolkit {
		if !isSurfaceToolkitFormSource(report.Source) {
			issues = append(issues, fmt.Sprintf("component_tree toolkit source path must match examples/surface_toolkit_form.tetra, got %q", report.Source))
		}
	} else if !isSurfaceTreeAppSource(report.Source) {
		issues = append(issues, fmt.Sprintf("component_tree source path must match examples/surface_tree_app.tetra, got %q", report.Source))
	}
	if report.ComponentTree == nil {
		if accessibility {
			return append(issues, "component_tree evidence is required for examples/surface_accessibility_settings.tetra")
		}
		if productionToolkit {
			return append(issues, "component_tree evidence is required for examples/surface_release_form.tetra")
		}
		if minimalToolkit {
			return append(issues, "component_tree evidence is required for examples/surface_toolkit_form.tetra")
		}
		return append(issues, "component_tree evidence is required for examples/surface_tree_app.tetra")
	}

	tree := report.ComponentTree
	if tree.Schema != "tetra.surface.component-tree.v1" {
		issues = append(issues, fmt.Sprintf("component_tree schema is %q, want tetra.surface.component-tree.v1", tree.Schema))
	}
	if accessibility {
		wantLevel := "accessibility-metadata-tree-v1"
		if releaseAccessibility {
			wantLevel = "platform-bridge-v1"
		}
		if tree.DynamicLevel != wantLevel {
			issues = append(issues, fmt.Sprintf("component_tree dynamic_level is %q, want %s", tree.DynamicLevel, wantLevel))
		}
	} else if productionToolkit {
		if tree.DynamicLevel != "production-widgets-v1" {
			issues = append(issues, fmt.Sprintf("component_tree dynamic_level is %q, want production-widgets-v1", tree.DynamicLevel))
		}
	} else if toolkitReuse {
		if tree.DynamicLevel != "toolkit-reuse-widget-tree" {
			issues = append(issues, fmt.Sprintf("component_tree dynamic_level is %q, want toolkit-reuse-widget-tree", tree.DynamicLevel))
		}
	} else if minimalToolkit {
		if tree.DynamicLevel != "minimal-toolkit-widget-tree" {
			issues = append(issues, fmt.Sprintf("component_tree dynamic_level is %q, want minimal-toolkit-widget-tree", tree.DynamicLevel))
		}
	} else if tree.DynamicLevel != "semi-dynamic-child-list" {
		issues = append(issues, fmt.Sprintf("component_tree dynamic_level is %q, want semi-dynamic-child-list", tree.DynamicLevel))
	}
	if tree.NodeCount != len(tree.Nodes) {
		issues = append(issues, fmt.Sprintf("component_tree node_count = %d, want len(nodes) %d", tree.NodeCount, len(tree.Nodes)))
	}
	if tree.NodeCount < 7 || len(tree.Nodes) < 7 {
		issues = append(issues, fmt.Sprintf("component_tree node_count = %d, want at least 7", tree.NodeCount))
	}

	nodes := map[int]ComponentTreeNodeReport{}
	childrenByParent := map[int][]ComponentTreeNodeReport{}
	focusableCount := 0
	for _, node := range tree.Nodes {
		if _, exists := nodes[node.ID]; exists {
			issues = append(issues, fmt.Sprintf("component_tree duplicate node id %d", node.ID))
		}
		nodes[node.ID] = node
		if strings.TrimSpace(node.Name) == "" {
			issues = append(issues, fmt.Sprintf("component_tree node %d name is required", node.ID))
		}
		if strings.TrimSpace(node.Kind) == "" {
			issues = append(issues, fmt.Sprintf("component_tree node %d kind is required", node.ID))
		}
		if node.Bounds.W <= 0 || node.Bounds.H <= 0 {
			issues = append(issues, fmt.Sprintf("component_tree node %d layout bounds are required", node.ID))
		}
		if node.Focusable {
			focusableCount++
		}
		if node.ParentID >= 0 {
			childrenByParent[node.ParentID] = append(childrenByParent[node.ParentID], node)
		}
		if node.ChildCount == 0 && node.FirstChild != -1 {
			issues = append(issues, fmt.Sprintf("component_tree leaf node %d first_child = %d, want -1", node.ID, node.FirstChild))
		}
		if node.ChildCount < 0 {
			issues = append(issues, fmt.Sprintf("component_tree node %d child_count must be non-negative", node.ID))
		}
	}

	root, ok := nodes[tree.RootID]
	if !ok {
		issues = append(issues, fmt.Sprintf("component_tree root_id %d is not in nodes", tree.RootID))
	} else {
		if root.ParentID != -1 {
			issues = append(issues, fmt.Sprintf("component_tree root %d parent_id = %d, want -1", root.ID, root.ParentID))
		}
		if root.ChildCount < 1 {
			issues = append(issues, "component_tree root must have at least one child")
		}
	}
	if _, ok := nodes[tree.FocusedID]; !ok {
		issues = append(issues, fmt.Sprintf("component_tree focused_id %d is not in nodes", tree.FocusedID))
	}

	for _, node := range tree.Nodes {
		if node.ParentID < 0 {
			continue
		}
		parent, ok := nodes[node.ParentID]
		if !ok {
			issues = append(issues, fmt.Sprintf("component_tree node %d parent_id %d is unknown", node.ID, node.ParentID))
			continue
		}
		if !rectContainsRect(parent.Bounds, node.Bounds) {
			issues = append(issues, fmt.Sprintf("component_tree node %d bounds must be inside parent %d bounds", node.ID, parent.ID))
		}
	}
	for parentID, children := range childrenByParent {
		parent := nodes[parentID]
		if parent.ChildCount != len(children) {
			issues = append(issues, fmt.Sprintf("component_tree node %d child_count = %d, want %d", parentID, parent.ChildCount, len(children)))
		}
		seenChildIndex := map[int]int{}
		firstID := -1
		for _, child := range children {
			if child.ChildIndex < 0 || child.ChildIndex >= len(children) {
				issues = append(issues, fmt.Sprintf("component_tree child node %d child_index = %d, want 0..%d", child.ID, child.ChildIndex, len(children)-1))
			}
			if prev, exists := seenChildIndex[child.ChildIndex]; exists {
				issues = append(issues, fmt.Sprintf("component_tree sibling child_index %d is used by nodes %d and %d", child.ChildIndex, prev, child.ID))
			}
			seenChildIndex[child.ChildIndex] = child.ID
			if child.ChildIndex == 0 {
				firstID = child.ID
			}
		}
		if len(children) > 0 && parent.FirstChild != firstID {
			issues = append(issues, fmt.Sprintf("component_tree node %d first_child = %d, want child_index 0 node %d", parentID, parent.FirstChild, firstID))
		}
	}
	issues = append(issues, validateComponentTreeSiblingLayout(nodes, childrenByParent)...)

	column, hasColumn := componentTreeNodeByKind(tree.Nodes, "column")
	row, hasRow := componentTreeNodeByKind(tree.Nodes, "row")
	textBoxName := "TextBox"
	secondTextBoxName := ""
	submitName := "SubmitButton"
	if accessibility || productionToolkit || toolkitReuse {
		textBoxName = "NameTextBox"
		secondTextBoxName = "EmailTextBox"
		submitName = "SaveButton"
	}
	textBox, hasTextBox := componentTreeNodeByName(tree.Nodes, textBoxName)
	secondTextBox, hasSecondTextBox := componentTreeNodeByName(tree.Nodes, secondTextBoxName)
	checkbox, hasCheckbox := componentTreeNodeByName(tree.Nodes, "SubscribeCheckbox")
	submit, hasSubmit := componentTreeNodeByName(tree.Nodes, submitName)
	reset, hasReset := componentTreeNodeByName(tree.Nodes, "ResetButton")
	if !hasColumn || column.ChildCount < 3 {
		issues = append(issues, "component_tree requires a Column node with at least 3 children")
	}
	if !hasRow || row.ChildCount < 2 {
		issues = append(issues, "component_tree requires a Row node with at least 2 children")
	}
	if !hasTextBox {
		issues = append(issues, fmt.Sprintf("component_tree requires %s node", textBoxName))
	}
	if productionToolkit && !hasCheckbox {
		issues = append(issues, "component_tree requires SubscribeCheckbox node for production toolkit")
	}
	if (accessibility || productionToolkit || toolkitReuse) && !hasSecondTextBox {
		if accessibility {
			issues = append(issues, "component_tree requires EmailTextBox node for accessibility metadata")
		} else if productionToolkit {
			issues = append(issues, "component_tree requires EmailTextBox node for production toolkit")
		} else {
			issues = append(issues, "component_tree requires EmailTextBox node for toolkit reuse")
		}
	}
	if !hasSubmit {
		issues = append(issues, fmt.Sprintf("component_tree requires %s node", submitName))
	}
	if !hasReset {
		issues = append(issues, "component_tree requires ResetButton node")
	}
	if focusableCount < 3 {
		issues = append(issues, fmt.Sprintf("component_tree focusable node count = %d, want at least 3", focusableCount))
	}

	if !componentTreeDrawOrderCoversNodes(tree.DrawOrder, nodes) {
		issues = append(issues, "component_tree draw_order must include every node exactly once")
	}
	if hasTextBox && !containsInt(tree.FocusOrder, textBox.ID) {
		issues = append(issues, fmt.Sprintf("component_tree focus_order missing %s", textBoxName))
	}
	if (accessibility || productionToolkit || toolkitReuse) && hasSecondTextBox && !containsInt(tree.FocusOrder, secondTextBox.ID) {
		issues = append(issues, "component_tree focus_order missing EmailTextBox")
	}
	if hasSubmit && !containsInt(tree.FocusOrder, submit.ID) {
		issues = append(issues, fmt.Sprintf("component_tree focus_order missing %s", submitName))
	}
	if hasReset && !containsInt(tree.FocusOrder, reset.ID) {
		issues = append(issues, "component_tree focus_order missing ResetButton")
	}
	if hasTextBox && hasSubmit && hasReset {
		wantFocusOrder := []int{textBox.ID, submit.ID, reset.ID}
		if (accessibility || productionToolkit || toolkitReuse) && hasSecondTextBox {
			wantFocusOrder = []int{textBox.ID, secondTextBox.ID, submit.ID, reset.ID}
		}
		if productionToolkit && hasSecondTextBox && hasCheckbox {
			wantFocusOrder = []int{textBox.ID, secondTextBox.ID, checkbox.ID, submit.ID, reset.ID}
		}
		if !intSlicesEqual(tree.FocusOrder, wantFocusOrder) {
			if productionToolkit {
				issues = append(issues, fmt.Sprintf("component_tree focus_order = %v, want NameTextBox -> EmailTextBox -> SubscribeCheckbox -> SaveButton -> ResetButton (%v)", tree.FocusOrder, wantFocusOrder))
			} else if accessibility || toolkitReuse {
				issues = append(issues, fmt.Sprintf("component_tree focus_order = %v, want NameTextBox -> EmailTextBox -> SaveButton -> ResetButton (%v)", tree.FocusOrder, wantFocusOrder))
			} else {
				issues = append(issues, fmt.Sprintf("component_tree focus_order = %v, want TextBox -> SubmitButton -> ResetButton (%v)", tree.FocusOrder, wantFocusOrder))
			}
		}
	}

	if len(tree.LayoutPasses) == 0 {
		issues = append(issues, "component_tree layout_passes evidence is required")
	}
	if hasTextBox && !componentTreeHasResizeLayoutPass(tree.LayoutPasses, textBox.ID) {
		issues = append(issues, "component_tree layout_passes require TextBox initial and resize bounds")
	}
	if (accessibility || productionToolkit || toolkitReuse) && hasSecondTextBox && !componentTreeHasResizeLayoutPass(tree.LayoutPasses, secondTextBox.ID) {
		issues = append(issues, "component_tree layout_passes require EmailTextBox initial and resize bounds")
	}

	expectedPaths := map[int][]int{}
	if hasTextBox {
		if path, ok := componentTreePathToRoot(textBox.ID, nodes); ok {
			expectedPaths[textBox.ID] = path
		}
	}
	if hasSubmit {
		if path, ok := componentTreePathToRoot(submit.ID, nodes); ok {
			expectedPaths[submit.ID] = path
		}
	}
	if (accessibility || productionToolkit || toolkitReuse) && hasSecondTextBox {
		if path, ok := componentTreePathToRoot(secondTextBox.ID, nodes); ok {
			expectedPaths[secondTextBox.ID] = path
		}
	}
	if productionToolkit && hasCheckbox {
		if path, ok := componentTreePathToRoot(checkbox.ID, nodes); ok {
			expectedPaths[checkbox.ID] = path
		}
	}
	if hasReset {
		if path, ok := componentTreePathToRoot(reset.ID, nodes); ok {
			expectedPaths[reset.ID] = path
		}
	}
	issues = append(issues, validateComponentTreeDispatchPaths(tree.DispatchPaths, expectedPaths, nodes)...)
	issues = append(issues, validateComponentTreeAPIEvidence(report, tree, expectedPaths, nodes)...)

	for _, required := range []string{
		"component tree node count",
		"component tree parent child links",
		"component tree layout bounds",
		"component tree draw traversal",
		"component tree pointer dispatch path",
		"component tree focus traversal",
		"component tree text routed to focused TextBox",
		"component tree button action dispatch",
		"component tree resize relayout",
		"component tree rendered frame update",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("component_tree report requires %s evidence", required))
		}
	}

	if hasTextBox && hasSubmit && hasReset {
		if productionToolkit && hasSecondTextBox && hasCheckbox {
			if !hasComponentTreeTabFocus(report.Events, fmt.Sprint(textBox.ID), fmt.Sprint(secondTextBox.ID)) ||
				!hasComponentTreeTabFocus(report.Events, fmt.Sprint(secondTextBox.ID), fmt.Sprint(checkbox.ID)) ||
				!hasComponentTreeTabFocus(report.Events, fmt.Sprint(checkbox.ID), fmt.Sprint(submit.ID)) ||
				!hasComponentTreeTabFocus(report.Events, fmt.Sprint(submit.ID), fmt.Sprint(reset.ID)) ||
				!hasComponentTreeTabFocus(report.Events, fmt.Sprint(reset.ID), fmt.Sprint(textBox.ID)) {
				issues = append(issues, "component_tree requires Tab focus traversal NameTextBox -> EmailTextBox -> SubscribeCheckbox -> SaveButton -> ResetButton -> NameTextBox")
			}
		} else if (accessibility || toolkitReuse) && hasSecondTextBox {
			if !hasComponentTreeTabFocus(report.Events, fmt.Sprint(textBox.ID), fmt.Sprint(secondTextBox.ID)) ||
				!hasComponentTreeTabFocus(report.Events, fmt.Sprint(secondTextBox.ID), fmt.Sprint(submit.ID)) ||
				!hasComponentTreeTabFocus(report.Events, fmt.Sprint(submit.ID), fmt.Sprint(reset.ID)) ||
				!hasComponentTreeTabFocus(report.Events, fmt.Sprint(reset.ID), fmt.Sprint(textBox.ID)) {
				issues = append(issues, "component_tree requires Tab focus traversal NameTextBox -> EmailTextBox -> SaveButton -> ResetButton -> NameTextBox")
			}
		} else if !hasComponentTreeTabFocus(report.Events, fmt.Sprint(textBox.ID), fmt.Sprint(submit.ID)) ||
			!hasComponentTreeTabFocus(report.Events, fmt.Sprint(submit.ID), fmt.Sprint(reset.ID)) ||
			!hasComponentTreeTabFocus(report.Events, fmt.Sprint(reset.ID), fmt.Sprint(textBox.ID)) {
			issues = append(issues, "component_tree requires Tab focus traversal TextBox -> SubmitButton -> ResetButton -> TextBox including ResetButton -> TextBox wrap")
		}
	}
	if !hasComponentTreeTextInsertion(report.Events) {
		issues = append(issues, "component_tree requires text input routed to focused TextBox")
	}
	if componentTreeTextMutatedWhileButtonFocused(report.Events) {
		issues = append(issues, "component_tree unfocused TextBox mutated while Button focused")
	}
	if !hasComponentTreeButtonAction(report.Events) {
		issues = append(issues, "component_tree requires keyboard button action dispatch through tree path")
	}
	if !hasComponentTreeResizeRelayout(report.Events, report.StateTransitions) {
		issues = append(issues, "component_tree resize relayout requires changed TextBox bounds while preserving focused_id")
	}
	if len(report.Frames) >= 2 && report.Frames[0].Checksum == report.Frames[len(report.Frames)-1].Checksum {
		issues = append(issues, "component_tree rendered frame update requires changed frame checksum")
	}
	return issues
}

func validateMinimalToolkitEvidence(report Report) []string {
	if !isMinimalToolkitReport(report) {
		return nil
	}
	if isToolkitReuseReport(report) {
		return validateToolkitReuseEvidence(report)
	}
	var issues []string
	if !isSurfaceToolkitFormSource(report.Source) {
		issues = append(issues, fmt.Sprintf("minimal toolkit source path must match examples/surface_toolkit_form.tetra, got %q", report.Source))
	}
	if report.Toolkit == nil {
		return append(issues, "toolkit evidence is required for examples/surface_toolkit_form.tetra")
	}
	toolkit := report.Toolkit
	if toolkit.Schema != "tetra.surface.toolkit.v1" {
		issues = append(issues, fmt.Sprintf("toolkit schema is %q, want tetra.surface.toolkit.v1", toolkit.Schema))
	}
	if toolkit.ToolkitLevel != "minimal-widgets-v1" {
		issues = append(issues, fmt.Sprintf("toolkit_level is %q, want minimal-widgets-v1", toolkit.ToolkitLevel))
	}
	if normalizeEvidencePath(toolkit.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("toolkit source %q must match report source %q", toolkit.Source, report.Source))
	}
	if toolkit.Module != "lib.core.widgets" {
		issues = append(issues, fmt.Sprintf("toolkit module is %q, want lib.core.widgets", toolkit.Module))
	}
	if !toolkit.Experimental {
		issues = append(issues, "toolkit must declare experimental=true")
	}
	if toolkit.ProductionClaim {
		issues = append(issues, "toolkit production_claim must be false for experimental minimal toolkit evidence")
	}
	if !toolkit.UsesComponentTreeAPI {
		issues = append(issues, "toolkit uses_component_tree_api must be true")
	}
	if toolkit.ManualBookkeeping {
		issues = append(issues, "toolkit manual_bookkeeping must be false")
	}
	if toolkit.DemoSpecificWidgetStructs {
		issues = append(issues, "toolkit demo_specific_widget_structs must be false")
	}
	if !toolkit.NoMagicWidgets || !toolkit.NoPlatformWidgets || !toolkit.NoDOMUI || !toolkit.NoUserJS {
		issues = append(issues, "toolkit must prove no_magic_widgets, no_platform_widgets, no_dom_ui, and no_user_js")
	}

	widgets := map[string]ToolkitWidgetReport{}
	for _, widget := range toolkit.Widgets {
		if strings.TrimSpace(widget.Name) == "" {
			issues = append(issues, "toolkit widget name is required")
			continue
		}
		if _, exists := widgets[widget.Name]; exists {
			issues = append(issues, fmt.Sprintf("toolkit duplicate widget %s", widget.Name))
		}
		widgets[widget.Name] = widget
		if strings.TrimSpace(widget.Kind) == "" {
			issues = append(issues, fmt.Sprintf("toolkit widget %s kind is required", widget.Name))
		}
		if widget.NodeID < 0 {
			issues = append(issues, fmt.Sprintf("toolkit widget %s node_id must be non-negative", widget.Name))
		}
		if !widget.Reusable || !widget.OrdinaryTetraStruct {
			issues = append(issues, fmt.Sprintf("toolkit widget %s must be reusable ordinary Tetra struct evidence", widget.Name))
		}
	}
	for _, required := range []struct {
		name   string
		kind   string
		nodeID int
		role   string
		action string
	}{
		{name: "Panel", kind: "Panel", nodeID: 1},
		{name: "Column", kind: "Column", nodeID: 2},
		{name: "NameLabel", kind: "Text", nodeID: 3, role: "label"},
		{name: "TextBox", kind: "TextBox", nodeID: 4},
		{name: "ButtonRow", kind: "Row", nodeID: 5},
		{name: "SubmitButton", kind: "Button", nodeID: 6, action: "submit"},
		{name: "ResetButton", kind: "Button", nodeID: 7, action: "reset"},
		{name: "StatusText", kind: "Text", nodeID: 8, role: "status"},
	} {
		widget, ok := widgets[required.name]
		if !ok {
			issues = append(issues, fmt.Sprintf("toolkit widget evidence missing %s", required.name))
			continue
		}
		if widget.Kind != required.kind {
			issues = append(issues, fmt.Sprintf("toolkit widget %s kind is %q, want %q", required.name, widget.Kind, required.kind))
		}
		if widget.NodeID != required.nodeID {
			issues = append(issues, fmt.Sprintf("toolkit widget %s node_id = %d, want %d", required.name, widget.NodeID, required.nodeID))
		}
		if required.role != "" && widget.Role != required.role {
			issues = append(issues, fmt.Sprintf("toolkit widget %s role is %q, want %q", required.name, widget.Role, required.role))
		}
		if required.action != "" && widget.Action != required.action {
			issues = append(issues, fmt.Sprintf("toolkit widget %s action is %q, want %q", required.name, widget.Action, required.action))
		}
		if required.kind == "TextBox" && !widget.Editable {
			issues = append(issues, "toolkit widget TextBox must prove editable=true")
		}
	}
	for _, source := range []string{
		"lib/core/widgets.tetra:panel_init",
		"lib/core/widgets.tetra:column_init",
		"lib/core/widgets.tetra:text_init",
		"lib/core/widgets.tetra:textbox_init",
		"lib/core/widgets.tetra:row_init",
		"lib/core/widgets.tetra:button_init",
	} {
		if !contains(toolkit.ReusableSources, source) {
			issues = append(issues, fmt.Sprintf("toolkit reusable_sources missing %s", source))
		}
	}

	treeNodes := map[int]ComponentTreeNodeReport{}
	if report.ComponentTree == nil {
		issues = append(issues, "minimal toolkit requires component_tree evidence")
	} else {
		for _, node := range report.ComponentTree.Nodes {
			treeNodes[node.ID] = node
		}
		for name, widget := range widgets {
			node, ok := treeNodes[widget.NodeID]
			if !ok {
				issues = append(issues, fmt.Sprintf("toolkit widget %s node_id %d is not in component_tree", name, widget.NodeID))
				continue
			}
			if node.Name != widget.Name {
				issues = append(issues, fmt.Sprintf("toolkit widget %s node_id %d names component_tree node %s", name, widget.NodeID, node.Name))
			}
		}
	}

	for _, required := range []string{
		"minimal toolkit reusable widgets",
		"minimal toolkit Text widget evidence",
		"minimal toolkit Button widget evidence",
		"minimal toolkit TextBox widget evidence",
		"minimal toolkit Row Column Panel layout",
		"minimal toolkit tree api reuse",
		"minimal toolkit TextBox focus input editing",
		"minimal toolkit Submit action routed",
		"minimal toolkit Reset action routed",
		"minimal toolkit status text update",
		"minimal toolkit resize relayout",
		"minimal toolkit rendered frame update",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("toolkit report requires %s evidence", required))
		}
	}
	if !hasEventTargetKind(report.Events, "TextBox", "mouse_up") {
		issues = append(issues, "toolkit TextBox requires mouse_up focus evidence")
	}
	if !hasEventTargetKind(report.Events, "TextBox", "text_input") || !hasKeyEvent(report.Events, 37) ||
		!hasKeyEvent(report.Events, 8) || !hasKeyEvent(report.Events, 46) {
		issues = append(issues, "toolkit TextBox requires OK text insertion plus caret, backspace, and delete evidence")
	}
	if !hasMinimalToolkitButtonAction(report.Events, "SubmitButton", "submit_count", "StatusText.status_code") {
		issues = append(issues, "minimal toolkit SubmitButton action requires focused root-to-leaf dispatch path and status update")
	}
	if !hasMinimalToolkitButtonAction(report.Events, "ResetButton", "reset_count", "StatusText.status_code") {
		issues = append(issues, "minimal toolkit ResetButton action requires focused root-to-leaf dispatch path and status update")
	}
	if !hasTransition(report.StateTransitions, "StatusText", "status_code") {
		issues = append(issues, "minimal toolkit requires StatusText status_code transition evidence")
	}
	if !hasTransition(report.StateTransitions, "ToolkitFormApp", "TextBox.bounds.w") {
		issues = append(issues, "minimal toolkit requires resize relayout evidence for TextBox.bounds.w")
	}
	if len(report.Frames) >= 2 && report.Frames[0].Checksum == report.Frames[len(report.Frames)-1].Checksum {
		issues = append(issues, "minimal toolkit rendered frame update requires changed frame checksum")
	}
	return issues
}

func validateToolkitReuseEvidence(report Report) []string {
	var issues []string
	if !isSurfaceToolkitSettingsSource(report.Source) {
		issues = append(issues, fmt.Sprintf("toolkit reuse source path must match examples/surface_toolkit_settings.tetra, got %q", report.Source))
	}
	if report.Toolkit == nil {
		return append(issues, "toolkit evidence is required for examples/surface_toolkit_settings.tetra")
	}
	toolkit := report.Toolkit
	if toolkit.Schema != "tetra.surface.toolkit.v1" {
		issues = append(issues, fmt.Sprintf("toolkit schema is %q, want tetra.surface.toolkit.v1", toolkit.Schema))
	}
	if toolkit.ToolkitLevel != "toolkit-reuse-v1" {
		issues = append(issues, fmt.Sprintf("toolkit_level is %q, want toolkit-reuse-v1", toolkit.ToolkitLevel))
	}
	if toolkit.ReuseLevel != "multi-form-widget-reuse-v1" {
		issues = append(issues, fmt.Sprintf("toolkit reuse_level is %q, want multi-form-widget-reuse-v1", toolkit.ReuseLevel))
	}
	if normalizeEvidencePath(toolkit.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("toolkit source %q must match report source %q", toolkit.Source, report.Source))
	}
	if toolkit.Module != "lib.core.widgets" {
		issues = append(issues, fmt.Sprintf("toolkit module is %q, want lib.core.widgets", toolkit.Module))
	}
	if !toolkit.Experimental {
		issues = append(issues, "toolkit must declare experimental=true")
	}
	if toolkit.ProductionClaim {
		issues = append(issues, "toolkit production_claim must be false for experimental toolkit reuse evidence")
	}
	if !toolkit.UsesComponentTreeAPI {
		issues = append(issues, "toolkit uses_component_tree_api must be true")
	}
	if toolkit.ManualBookkeeping {
		issues = append(issues, "toolkit manual_bookkeeping must be false")
	}
	if toolkit.DemoSpecificWidgetStructs {
		issues = append(issues, "toolkit demo_specific_widget_structs must be false")
	}
	if !toolkit.NoMagicWidgets || !toolkit.NoPlatformWidgets || !toolkit.NoDOMUI || !toolkit.NoUserJS {
		issues = append(issues, "toolkit must prove no_magic_widgets, no_platform_widgets, no_dom_ui, and no_user_js")
	}
	if toolkit.ExampleCount < 2 || !contains(toolkit.Sources, "examples/surface_toolkit_form.tetra") || !contains(toolkit.Sources, "examples/surface_toolkit_settings.tetra") {
		issues = append(issues, "toolkit reuse example_count must cover both surface_toolkit_form and surface_toolkit_settings examples")
	}
	if toolkit.TextBoxCount < 2 || !toolkit.MultiTextBoxEvidence {
		issues = append(issues, "toolkit reuse requires text_box_count >= 2 with multi_textbox_evidence")
	}
	if toolkit.ButtonCount < 2 || !toolkit.MultiFormEvidence {
		issues = append(issues, "toolkit reuse requires button_count >= 2 with multi_form_evidence")
	}

	widgets := map[string]ToolkitWidgetReport{}
	kindCounts := map[string]int{}
	ids := map[int]string{}
	for _, widget := range toolkit.Widgets {
		if strings.TrimSpace(widget.Name) == "" {
			issues = append(issues, "toolkit widget name is required")
			continue
		}
		if _, exists := widgets[widget.Name]; exists {
			issues = append(issues, fmt.Sprintf("toolkit duplicate widget %s", widget.Name))
		}
		widgets[widget.Name] = widget
		kindCounts[widget.Kind]++
		if prev, exists := ids[widget.NodeID]; exists {
			issues = append(issues, fmt.Sprintf("toolkit duplicate widget node_id %d used by %s and %s", widget.NodeID, prev, widget.Name))
		}
		ids[widget.NodeID] = widget.Name
		if !widget.Reusable || !widget.OrdinaryTetraStruct {
			issues = append(issues, fmt.Sprintf("toolkit widget %s must be reusable ordinary Tetra struct evidence", widget.Name))
		}
	}
	for _, requiredKind := range []string{"Panel", "Column", "Row", "Text", "TextBox", "Button"} {
		if kindCounts[requiredKind] == 0 {
			issues = append(issues, fmt.Sprintf("toolkit widget set missing %s", requiredKind))
		}
	}
	if kindCounts["TextBox"] < 2 {
		issues = append(issues, "toolkit reuse requires at least two TextBox widgets")
	}
	if kindCounts["Button"] < 2 {
		issues = append(issues, "toolkit reuse requires at least two Button widgets")
	}
	for _, required := range []struct {
		name   string
		kind   string
		nodeID int
		role   string
		action string
	}{
		{name: "Panel", kind: "Panel", nodeID: 1},
		{name: "Column", kind: "Column", nodeID: 2},
		{name: "TitleText", kind: "Text", nodeID: 3, role: "label"},
		{name: "NameTextBox", kind: "TextBox", nodeID: 4},
		{name: "NameLabel", kind: "Text", nodeID: 5, role: "label"},
		{name: "EmailTextBox", kind: "TextBox", nodeID: 6},
		{name: "ButtonRow", kind: "Row", nodeID: 7},
		{name: "SaveButton", kind: "Button", nodeID: 8, action: "save"},
		{name: "ResetButton", kind: "Button", nodeID: 9, action: "reset"},
		{name: "StatusText", kind: "Text", nodeID: 10, role: "status"},
	} {
		widget, ok := widgets[required.name]
		if !ok {
			issues = append(issues, fmt.Sprintf("toolkit widget evidence missing %s", required.name))
			continue
		}
		if widget.Kind != required.kind {
			issues = append(issues, fmt.Sprintf("toolkit widget %s kind is %q, want %q", required.name, widget.Kind, required.kind))
		}
		if widget.NodeID != required.nodeID {
			issues = append(issues, fmt.Sprintf("toolkit widget %s node_id = %d, want %d", required.name, widget.NodeID, required.nodeID))
		}
		if required.role != "" && widget.Role != required.role {
			issues = append(issues, fmt.Sprintf("toolkit widget %s role is %q, want %q", required.name, widget.Role, required.role))
		}
		if required.action != "" && widget.Action != required.action {
			issues = append(issues, fmt.Sprintf("toolkit widget %s action is %q, want %q", required.name, widget.Action, required.action))
		}
		if required.kind == "TextBox" && !widget.Editable {
			issues = append(issues, fmt.Sprintf("toolkit widget %s must prove editable=true", required.name))
		}
	}
	treeNodes := map[int]ComponentTreeNodeReport{}
	if report.ComponentTree == nil {
		issues = append(issues, "toolkit reuse requires component_tree evidence")
	} else {
		for _, node := range report.ComponentTree.Nodes {
			treeNodes[node.ID] = node
		}
		for name, widget := range widgets {
			node, ok := treeNodes[widget.NodeID]
			if !ok {
				issues = append(issues, fmt.Sprintf("toolkit widget %s node_id %d is not in component_tree", name, widget.NodeID))
				continue
			}
			if node.Name != widget.Name {
				issues = append(issues, fmt.Sprintf("toolkit widget %s node_id %d names component_tree node %s", name, widget.NodeID, node.Name))
			}
		}
	}
	for _, source := range []string{
		"lib/core/widgets.tetra:panel_init",
		"lib/core/widgets.tetra:column_init",
		"lib/core/widgets.tetra:text_init",
		"lib/core/widgets.tetra:textbox_init",
		"lib/core/widgets.tetra:row_init",
		"lib/core/widgets.tetra:button_init",
		"lib/core/widgets.tetra:hit_test",
		"lib/core/widgets.tetra:textbox_text_input",
		"lib/core/widgets.tetra:button_key_event",
	} {
		if !contains(toolkit.ReusableSources, source) {
			issues = append(issues, fmt.Sprintf("toolkit reusable_sources missing %s", source))
		}
	}
	for _, required := range []string{
		"toolkit reuse second example evidence",
		"toolkit reuse widgets module evidence",
		"toolkit reuse multi TextBox routing",
		"toolkit reuse focused TextBox only mutates",
		"toolkit reuse Save action routed",
		"toolkit reuse Reset action routed",
		"toolkit reuse StatusText updates",
		"toolkit reuse resize relayout",
		"toolkit reuse changed frame checksums",
		"toolkit reuse no demo-local widget structs",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("toolkit reuse report requires %s evidence", required))
		}
	}
	if !hasEventTargetKind(report.Events, "NameTextBox", "mouse_up") {
		issues = append(issues, "toolkit reuse requires NameTextBox mouse_up focus evidence")
	}
	if !hasEventTargetKind(report.Events, "NameTextBox", "text_input") || !hasEventTargetKind(report.Events, "EmailTextBox", "text_input") {
		issues = append(issues, "toolkit reuse requires NameTextBox and EmailTextBox text_input routing evidence")
	}
	if !toolkitTextInputMutatesOnlyFocused(report.Events, "NameTextBox") || !toolkitTextInputMutatesOnlyFocused(report.Events, "EmailTextBox") {
		issues = append(issues, "toolkit reuse requires focused TextBox only mutation evidence")
	}
	if !hasToolkitReuseButtonAction(report.Events, "SaveButton", ".save_count", false) {
		issues = append(issues, "toolkit reuse SaveButton action requires focused root-to-leaf dispatch path and status update")
	}
	if !hasToolkitReuseButtonAction(report.Events, "ResetButton", ".reset_count", true) {
		issues = append(issues, "toolkit reuse ResetButton action requires focused root-to-leaf dispatch path and TextBox clears")
	}
	if !hasTransition(report.StateTransitions, "StatusText", "status_code") {
		issues = append(issues, "toolkit reuse requires StatusText status_code transition evidence")
	}
	if !hasTransition(report.StateTransitions, "ToolkitSettingsApp", "NameTextBox.bounds.w") || !hasTransition(report.StateTransitions, "ToolkitSettingsApp", "EmailTextBox.bounds.w") {
		issues = append(issues, "toolkit reuse requires resize relayout evidence for both TextBoxes")
	}
	if len(report.Frames) >= 2 && report.Frames[0].Checksum == report.Frames[len(report.Frames)-1].Checksum {
		issues = append(issues, "toolkit reuse rendered frame update requires changed frame checksum")
	}
	if report.Target == "wasm32-web" && !isBrowserCanvasHostEvidenceLevel(report.HostEvidence.Level) {
		issues = append(issues, "toolkit reuse browser evidence must be browser-canvas input, not Node-only wasm32-web evidence")
	}
	return issues
}

func validateProductionToolkitEvidence(report Report) []string {
	if !isProductionToolkitReport(report) {
		return nil
	}
	var issues []string
	if !isSurfaceReleaseFormSource(report.Source) {
		issues = append(issues, fmt.Sprintf("production toolkit source path must match examples/surface_release_form.tetra, got %q", report.Source))
	}
	if report.Toolkit == nil {
		return append(issues, "toolkit evidence is required for examples/surface_release_form.tetra")
	}
	toolkit := report.Toolkit
	if toolkit.Schema != "tetra.surface.toolkit.v1" {
		issues = append(issues, fmt.Sprintf("toolkit schema is %q, want tetra.surface.toolkit.v1", toolkit.Schema))
	}
	if toolkit.ToolkitLevel != "production-widgets-v1" {
		issues = append(issues, fmt.Sprintf("toolkit_level is %q, want production-widgets-v1", toolkit.ToolkitLevel))
	}
	if toolkit.ReleaseScope != "surface-v1-linux-web" {
		issues = append(issues, fmt.Sprintf("toolkit release_scope is %q, want surface-v1-linux-web", toolkit.ReleaseScope))
	}
	if normalizeEvidencePath(toolkit.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("toolkit source %q must match report source %q", toolkit.Source, report.Source))
	}
	if toolkit.Module != "lib.core.widgets" {
		issues = append(issues, fmt.Sprintf("toolkit module is %q, want lib.core.widgets", toolkit.Module))
	}
	if toolkit.StyleModule != "lib.core.style" {
		issues = append(issues, fmt.Sprintf("toolkit style_module is %q, want lib.core.style", toolkit.StyleModule))
	}
	if toolkit.Experimental {
		issues = append(issues, "production toolkit must declare experimental=false")
	}
	if !toolkit.ProductionClaim {
		issues = append(issues, "production toolkit production_claim must be true")
	}
	if !toolkit.UsesComponentTreeAPI {
		issues = append(issues, "production toolkit uses_component_tree_api must be true")
	}
	if toolkit.ManualBookkeeping {
		issues = append(issues, "production toolkit manual_bookkeeping must be false")
	}
	if toolkit.DemoSpecificWidgetStructs {
		issues = append(issues, "production toolkit demo_specific_widget_structs must be false")
	}
	if !toolkit.NoMagicWidgets || !toolkit.NoPlatformWidgets || !toolkit.NoDOMUI || !toolkit.NoUserJS {
		issues = append(issues, "production toolkit must prove no_magic_widgets, no_platform_widgets, no_dom_ui, and no_user_js")
	}
	requiredSources := []string{
		"examples/surface_release_form.tetra",
		"examples/surface_toolkit_form.tetra",
		"examples/surface_toolkit_settings.tetra",
	}
	if toolkit.ExampleCount < len(requiredSources) {
		issues = append(issues, fmt.Sprintf("production toolkit example_count = %d, want at least %d scoped examples", toolkit.ExampleCount, len(requiredSources)))
	}
	for _, source := range requiredSources {
		if !contains(toolkit.Sources, source) {
			issues = append(issues, fmt.Sprintf("production toolkit sources missing %s", source))
		}
	}
	if toolkit.TextBoxCount < 2 || !toolkit.MultiTextBoxEvidence {
		issues = append(issues, "production toolkit requires two TextBox widgets with multi_textbox_evidence")
	}
	if toolkit.ButtonCount < 2 || !toolkit.MultiFormEvidence {
		issues = append(issues, "production toolkit requires at least two Button widgets with multi_form_evidence")
	}
	for _, required := range []string{"Text", "Label", "StatusText", "Button", "TextBox", "Checkbox", "Row", "Column", "Panel", "Stack", "Scroll", "Spacer"} {
		if !contains(toolkit.WidgetSet, required) {
			issues = append(issues, fmt.Sprintf("production toolkit widget_set missing %s", required))
		}
	}
	for _, required := range []string{"normal", "focused", "hovered", "pressed", "disabled", "error"} {
		if !contains(toolkit.StateSet, required) {
			issues = append(issues, fmt.Sprintf("production toolkit state_set missing %s", required))
		}
	}
	for _, required := range []string{"padding", "margin", "spacing", "min_size", "max_size", "fill", "scroll_offset"} {
		if !contains(toolkit.LayoutFeatures, required) {
			issues = append(issues, fmt.Sprintf("production toolkit layout_features missing %s", required))
		}
	}
	if !toolkit.Theme {
		issues = append(issues, "production toolkit theme must be true")
	}
	if !toolkit.SafeTextStorage {
		issues = append(issues, "production toolkit safe_text_storage must be true")
	}

	widgets := map[string]ToolkitWidgetReport{}
	kindCounts := map[string]int{}
	ids := map[int]string{}
	for _, widget := range toolkit.Widgets {
		if strings.TrimSpace(widget.Name) == "" {
			issues = append(issues, "production toolkit widget name is required")
			continue
		}
		if _, exists := widgets[widget.Name]; exists {
			issues = append(issues, fmt.Sprintf("production toolkit duplicate widget %s", widget.Name))
		}
		widgets[widget.Name] = widget
		kindCounts[widget.Kind]++
		if prev, exists := ids[widget.NodeID]; exists {
			issues = append(issues, fmt.Sprintf("production toolkit duplicate widget node_id %d used by %s and %s", widget.NodeID, prev, widget.Name))
		}
		ids[widget.NodeID] = widget.Name
		if !widget.Reusable || !widget.OrdinaryTetraStruct {
			issues = append(issues, fmt.Sprintf("production toolkit widget %s must be reusable ordinary Tetra struct evidence", widget.Name))
		}
	}
	for _, requiredKind := range []string{"Text", "Label", "StatusText", "Button", "TextBox", "Checkbox", "Row", "Column", "Panel", "Stack", "Scroll", "Spacer"} {
		if kindCounts[requiredKind] == 0 {
			issues = append(issues, fmt.Sprintf("production toolkit widget evidence missing kind %s", requiredKind))
		}
	}
	for _, required := range []struct {
		name   string
		kind   string
		nodeID int
		role   string
		action string
	}{
		{name: "Panel", kind: "Panel", nodeID: 1},
		{name: "Stack", kind: "Stack", nodeID: 2},
		{name: "Column", kind: "Column", nodeID: 3},
		{name: "TitleText", kind: "Text", nodeID: 4, role: "label"},
		{name: "DescriptionText", kind: "Text", nodeID: 5, role: "description"},
		{name: "NameLabel", kind: "Label", nodeID: 6, role: "label"},
		{name: "NameTextBox", kind: "TextBox", nodeID: 7},
		{name: "EmailLabel", kind: "Label", nodeID: 8, role: "label"},
		{name: "EmailTextBox", kind: "TextBox", nodeID: 9},
		{name: "SubscribeCheckbox", kind: "Checkbox", nodeID: 10},
		{name: "TermsScroll", kind: "Scroll", nodeID: 11},
		{name: "TermsText", kind: "Text", nodeID: 12, role: "description"},
		{name: "ButtonRow", kind: "Row", nodeID: 13},
		{name: "SaveButton", kind: "Button", nodeID: 14, action: "save"},
		{name: "ResetButton", kind: "Button", nodeID: 15, action: "reset"},
		{name: "Spacer", kind: "Spacer", nodeID: 16},
		{name: "StatusText", kind: "StatusText", nodeID: 17, role: "status"},
	} {
		widget, ok := widgets[required.name]
		if !ok {
			issues = append(issues, fmt.Sprintf("production toolkit widget evidence missing %s", required.name))
			continue
		}
		if widget.Kind != required.kind {
			issues = append(issues, fmt.Sprintf("production toolkit widget %s kind is %q, want %q", required.name, widget.Kind, required.kind))
		}
		if widget.NodeID != required.nodeID {
			issues = append(issues, fmt.Sprintf("production toolkit widget %s node_id = %d, want %d", required.name, widget.NodeID, required.nodeID))
		}
		if required.role != "" && widget.Role != required.role {
			issues = append(issues, fmt.Sprintf("production toolkit widget %s role is %q, want %q", required.name, widget.Role, required.role))
		}
		if required.action != "" && widget.Action != required.action {
			issues = append(issues, fmt.Sprintf("production toolkit widget %s action is %q, want %q", required.name, widget.Action, required.action))
		}
		if required.kind == "TextBox" && !widget.Editable {
			issues = append(issues, fmt.Sprintf("production toolkit widget %s must prove editable=true", required.name))
		}
	}
	if report.ComponentTree == nil {
		issues = append(issues, "production toolkit requires component_tree evidence")
	} else {
		treeNodes := map[int]ComponentTreeNodeReport{}
		for _, node := range report.ComponentTree.Nodes {
			treeNodes[node.ID] = node
		}
		for name, widget := range widgets {
			node, ok := treeNodes[widget.NodeID]
			if !ok {
				issues = append(issues, fmt.Sprintf("production toolkit widget %s node_id %d is not in component_tree", name, widget.NodeID))
				continue
			}
			if node.Name != widget.Name {
				issues = append(issues, fmt.Sprintf("production toolkit widget %s node_id %d names component_tree node %s", name, widget.NodeID, node.Name))
			}
		}
	}
	for _, source := range []string{
		"lib/core/widgets.tetra:panel_init",
		"lib/core/widgets.tetra:column_init",
		"lib/core/widgets.tetra:text_init",
		"lib/core/widgets.tetra:label_init",
		"lib/core/widgets.tetra:status_text_init",
		"lib/core/widgets.tetra:textbox_init",
		"lib/core/widgets.tetra:checkbox_init",
		"lib/core/widgets.tetra:checkbox_toggle",
		"lib/core/widgets.tetra:row_init",
		"lib/core/widgets.tetra:stack_init",
		"lib/core/widgets.tetra:scroll_init",
		"lib/core/widgets.tetra:scroll_set_offset",
		"lib/core/widgets.tetra:spacer_init",
		"lib/core/widgets.tetra:button_init",
		"lib/core/widgets.tetra:hit_test_release_form",
		"lib/core/style.tetra:default_theme",
		"lib/core/style.tetra:style_for_state",
	} {
		if !contains(toolkit.ReusableSources, source) {
			issues = append(issues, fmt.Sprintf("production toolkit reusable_sources missing %s", source))
		}
	}
	for _, required := range []string{
		"production toolkit required widget set",
		"production toolkit style module default theme",
		"production toolkit style states normal focused hovered pressed disabled error",
		"production toolkit Text Label StatusText evidence",
		"production toolkit Button TextBox Checkbox evidence",
		"production toolkit Row Column Panel Stack Scroll Spacer layout",
		"production toolkit component tree api reuse",
		"production toolkit TextBox focus input editing",
		"production toolkit Checkbox toggle routed",
		"production toolkit Scroll offset routed",
		"production toolkit Save action routed",
		"production toolkit Reset action routed",
		"production toolkit StatusText updates",
		"production toolkit safe text storage",
		"production toolkit no demo-local widget structs",
		"production toolkit browser host separation",
		"production toolkit rendered frame update",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("production toolkit report requires %s evidence", required))
		}
	}
	if !hasEventTargetKind(report.Events, "NameTextBox", "mouse_up") {
		issues = append(issues, "production toolkit requires NameTextBox mouse_up focus evidence")
	}
	if !hasEventTargetKind(report.Events, "NameTextBox", "text_input") || !hasEventTargetKind(report.Events, "EmailTextBox", "text_input") {
		issues = append(issues, "production toolkit requires NameTextBox and EmailTextBox text_input routing evidence")
	}
	if !hasEventTargetKind(report.Events, "SubscribeCheckbox", "key_down") || !hasTransition(report.StateTransitions, "SubscribeCheckbox", "checked") {
		issues = append(issues, "production toolkit requires Checkbox keyboard toggle and checked transition evidence")
	}
	if !eventKindContains(report.Events, "scroll") || !hasTransition(report.StateTransitions, "TermsScroll", "offset_y") {
		issues = append(issues, "production toolkit requires Scroll event and offset transition evidence")
	}
	if !hasToolkitReuseButtonAction(report.Events, "SaveButton", ".save_count", false) {
		issues = append(issues, "production toolkit SaveButton action requires focused root-to-leaf dispatch path and status update")
	}
	if !hasToolkitReuseButtonAction(report.Events, "ResetButton", ".reset_count", true) {
		issues = append(issues, "production toolkit ResetButton action requires focused root-to-leaf dispatch path and TextBox clears")
	}
	if !hasTransition(report.StateTransitions, "StatusText", "status_code") {
		issues = append(issues, "production toolkit requires StatusText status_code transition evidence")
	}
	if !hasTransition(report.StateTransitions, "SurfaceReleaseFormApp", "NameTextBox.bounds.w") || !hasTransition(report.StateTransitions, "SurfaceReleaseFormApp", "EmailTextBox.bounds.w") {
		issues = append(issues, "production toolkit requires resize relayout evidence for both TextBoxes")
	}
	if len(report.Frames) >= 2 && report.Frames[0].Checksum == report.Frames[len(report.Frames)-1].Checksum {
		issues = append(issues, "production toolkit rendered frame update requires changed frame checksum")
	}
	if report.Target == "wasm32-web" && !isBrowserCanvasHostEvidenceLevel(report.HostEvidence.Level) {
		issues = append(issues, "production toolkit browser evidence must be browser-canvas input, not Node-only wasm32-web evidence")
	}
	return issues
}

func validateBrowserReleaseEvidence(report Report) []string {
	if !isBrowserReleaseReport(report) {
		return nil
	}
	var issues []string
	if report.Target != "wasm32-web" {
		issues = append(issues, fmt.Sprintf("browser release target is %q, want wasm32-web", report.Target))
	}
	if !isSurfaceReleaseFormSource(report.Source) {
		issues = append(issues, fmt.Sprintf("browser release source path must match examples/surface_release_form.tetra, got %q", report.Source))
	}
	if report.HostEvidence.Level != "wasm32-web-browser-canvas-release-v1" {
		issues = append(issues, fmt.Sprintf("browser release host_evidence.level is %q, want wasm32-web-browser-canvas-release-v1", report.HostEvidence.Level))
	}
	if report.HostEvidence.Backend != "browser-canvas-rgba-accessible" {
		issues = append(issues, fmt.Sprintf("browser release host_evidence.backend is %q, want browser-canvas-rgba-accessible", report.HostEvidence.Backend))
	}
	if !report.HostEvidence.BrowserCanvas {
		issues = append(issues, "browser release host_evidence.browser_canvas must be true")
	}
	if !report.HostEvidence.BrowserInput {
		issues = append(issues, "browser release host_evidence.browser_input must be true")
	}
	if !report.HostEvidence.BrowserClipboard {
		issues = append(issues, "browser release host_evidence.browser_clipboard must be true")
	}
	if report.HostEvidence.BrowserClipboardHarness != "deterministic-browser-clipboard-v1" {
		issues = append(issues, fmt.Sprintf("browser release host_evidence.browser_clipboard_harness is %q, want deterministic-browser-clipboard-v1", report.HostEvidence.BrowserClipboardHarness))
	}
	if !report.HostEvidence.BrowserComposition {
		issues = append(issues, "browser release host_evidence.browser_composition must be true")
	}
	if !report.HostEvidence.BrowserAccessibilitySnapshot {
		issues = append(issues, "browser release host_evidence.browser_accessibility_snapshot must be true")
	}
	if !report.HostEvidence.BrowserAccessibilityMirror {
		issues = append(issues, "browser release host_evidence.browser_accessibility_mirror must be true")
	}
	if !hasFrameOrderDimensions(report.Frames, 5, 560, 420, 2240) {
		issues = append(issues, "browser release requires order-5 560x420 canvas readback frame evidence")
	}
	for _, required := range []string{
		"browser release Surface v1 schema",
		"browser release Chromium canvas readback",
		"browser release native pointer keyboard text resize",
		"browser release deterministic clipboard harness",
		"browser release composition trace",
		"browser release accessibility snapshot mirror",
		"browser release forbidden web sidecar rejection",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("browser release report requires %s evidence", required))
		}
	}
	if report.Toolkit == nil || report.Toolkit.ToolkitLevel != "production-widgets-v1" {
		issues = append(issues, "browser release requires production-widgets-v1 toolkit evidence")
	}
	if report.ComponentTree == nil || report.ComponentTree.DynamicLevel != "production-widgets-v1" {
		issues = append(issues, "browser release requires production-widgets-v1 component tree evidence")
	}
	return issues
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

func validateAccessibilityTreeEvidence(report Report) []string {
	if !isAccessibilityMetadataReport(report) {
		return nil
	}
	var issues []string
	releaseWindow := isLinuxReleaseWindowReport(report)
	releaseAccessibility := isSurfaceReleaseAccessibilitySource(report.Source)
	if !isSurfaceAccessibilitySettingsSource(report.Source) && !releaseAccessibility && !releaseWindow {
		issues = append(issues, fmt.Sprintf("accessibility_tree source path must match examples/surface_accessibility_settings.tetra or examples/surface_release_accessibility.tetra, got %q", report.Source))
	}
	if report.Target == "wasm32-web" && !isBrowserCanvasHostEvidenceLevel(report.HostEvidence.Level) {
		issues = append(issues, "accessibility_tree browser evidence must use browser canvas native input, not Node-only wasm32-web evidence")
	}
	if report.AccessibilityTree == nil {
		return append(issues, "accessibility_tree evidence is required for examples/surface_accessibility_settings.tetra")
	}

	tree := report.AccessibilityTree
	if releaseWindow {
		issues = append(issues, validateLinuxReleaseWindowAccessibilityBridgeEvidence(report, tree)...)
		return issues
	}
	if tree.Schema != "tetra.surface.accessibility-tree.v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree schema is %q, want tetra.surface.accessibility-tree.v1", tree.Schema))
	}
	if tree.AccessibilityLevel != "metadata-tree-v1" && tree.AccessibilityLevel != "platform-bridge-v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree accessibility_level is %q, want metadata-tree-v1 or platform-bridge-v1", tree.AccessibilityLevel))
	}
	releaseAccessibility = releaseAccessibility || tree.AccessibilityLevel == "platform-bridge-v1"
	if normalizeEvidencePath(tree.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("accessibility_tree source %q must match report source %q", tree.Source, report.Source))
	}
	if tree.Module != "lib.core.accessibility" {
		issues = append(issues, fmt.Sprintf("accessibility_tree module is %q, want lib.core.accessibility", tree.Module))
	}
	if tree.WidgetModule != "lib.core.widgets" {
		issues = append(issues, fmt.Sprintf("accessibility_tree widget_module is %q, want lib.core.widgets", tree.WidgetModule))
	}
	if releaseAccessibility {
		issues = append(issues, validateReleaseAccessibilityBridgeEvidence(report, tree)...)
	} else {
		if tree.AccessibilityLevel != "metadata-tree-v1" {
			issues = append(issues, fmt.Sprintf("accessibility_tree accessibility_level is %q, want metadata-tree-v1", tree.AccessibilityLevel))
		}
		if !tree.Experimental {
			issues = append(issues, "accessibility_tree must declare experimental=true")
		}
		if tree.ProductionClaim {
			issues = append(issues, "accessibility_tree production_claim must be false")
		}
		if tree.PlatformHostIntegration {
			issues = append(issues, "accessibility_tree platform_host_integration must be false for metadata-only Surface evidence")
		}
		if tree.DOMARIAIntegration {
			issues = append(issues, "accessibility_tree dom_aria_integration must be false")
		}
		if screenReaderEvidenceTruthy(tree.ScreenReaderEvidence) {
			issues = append(issues, "accessibility_tree screen_reader_evidence must be false")
		}
	}
	if !tree.DerivedFromComponentTree || !tree.UsesComponentTreeAPI || !tree.UsesWidgetToolkit {
		issues = append(issues, "accessibility_tree must be derived from component_tree, component_tree_api, and widget toolkit evidence")
	}
	if tree.ManualBookkeeping {
		issues = append(issues, "accessibility_tree manual_bookkeeping must be false")
	}
	if !tree.NoDOMUI || !tree.NoUserJS || !tree.NoPlatformWidgets || !tree.NoLegacySidecars {
		issues = append(issues, "accessibility_tree must prove no_dom_ui, no_user_js, no_platform_widgets, and no_legacy_sidecars")
	}
	if tree.ComponentTreeSchema != "tetra.surface.component-tree.v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree component_tree_schema is %q, want tetra.surface.component-tree.v1", tree.ComponentTreeSchema))
	}
	if tree.ComponentTreeAPISchema != "tetra.surface.component-tree-api.v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree component_tree_api_schema is %q, want tetra.surface.component-tree-api.v1", tree.ComponentTreeAPISchema))
	}
	if tree.ToolkitSchema != "tetra.surface.toolkit.v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree toolkit_schema is %q, want tetra.surface.toolkit.v1", tree.ToolkitSchema))
	}

	issues = append(issues, validateAccessibilityToolkitEvidence(report)...)

	componentNodes := map[int]ComponentTreeNodeReport{}
	if report.ComponentTree == nil {
		issues = append(issues, "accessibility_tree requires component_tree evidence")
	} else {
		wantLevel := "accessibility-metadata-tree-v1"
		if releaseAccessibility {
			wantLevel = "platform-bridge-v1"
		}
		if report.ComponentTree.DynamicLevel != wantLevel {
			issues = append(issues, fmt.Sprintf("accessibility_tree component_tree dynamic_level is %q, want %s", report.ComponentTree.DynamicLevel, wantLevel))
		}
		if !intSlicesEqual(report.ComponentTree.FocusOrder, []int{5, 7, 9, 10}) {
			issues = append(issues, fmt.Sprintf("accessibility_tree component_tree focus_order = %v, want [5 7 9 10]", report.ComponentTree.FocusOrder))
		}
		for _, node := range report.ComponentTree.Nodes {
			componentNodes[node.ID] = node
		}
	}

	if tree.NodeCount != len(tree.Nodes) {
		issues = append(issues, fmt.Sprintf("accessibility_tree node_count = %d, want len(nodes) %d", tree.NodeCount, len(tree.Nodes)))
	}
	if tree.NodeCount != 12 || len(tree.Nodes) != 12 {
		issues = append(issues, fmt.Sprintf("accessibility_tree node_count = %d, want 12", tree.NodeCount))
	}

	allowedRoles := map[string]bool{
		"root": true, "panel": true, "column": true, "text": true, "label": true,
		"textbox": true, "row": true, "button": true, "status": true,
	}
	for role := range allowedRoles {
		if !contains(tree.RolesPresent, role) {
			issues = append(issues, fmt.Sprintf("accessibility_tree roles_present missing %s", role))
		}
	}

	nodesByID := map[int]AccessibilityNodeReport{}
	nodesByName := map[string]AccessibilityNodeReport{}
	componentIDs := map[int]string{}
	roleCounts := map[string]int{}
	focusedCount := 0
	for _, node := range tree.Nodes {
		if _, exists := nodesByID[node.ID]; exists {
			issues = append(issues, fmt.Sprintf("accessibility_tree duplicate node id %d", node.ID))
		}
		nodesByID[node.ID] = node
		if strings.TrimSpace(node.Name) == "" {
			issues = append(issues, fmt.Sprintf("accessibility_tree node %d name is required", node.ID))
		} else if _, exists := nodesByName[node.Name]; exists {
			issues = append(issues, fmt.Sprintf("accessibility_tree duplicate node name %s", node.Name))
		}
		nodesByName[node.Name] = node
		if !allowedRoles[node.Role] {
			issues = append(issues, fmt.Sprintf("accessibility_tree node %s has unknown role %q", node.Name, node.Role))
		}
		roleCounts[node.Role]++
		if prev, exists := componentIDs[node.ComponentID]; exists {
			issues = append(issues, fmt.Sprintf("accessibility_tree duplicate component_id %d used by %s and %s", node.ComponentID, prev, node.Name))
		}
		componentIDs[node.ComponentID] = node.Name
		componentNode, ok := componentNodes[node.ComponentID]
		if !ok {
			issues = append(issues, fmt.Sprintf("accessibility_tree node %s component_id %d is not in component_tree", node.Name, node.ComponentID))
		} else {
			if componentNode.Name != node.Name {
				issues = append(issues, fmt.Sprintf("accessibility_tree node %s component_id %d names component_tree node %s", node.Name, node.ComponentID, componentNode.Name))
			}
			if componentNode.ParentID != node.ParentID {
				issues = append(issues, fmt.Sprintf("accessibility_tree node %s parent_id = %d, want component_tree parent_id %d", node.Name, node.ParentID, componentNode.ParentID))
			}
			if !rectsEqual(node.Bounds, componentNode.Bounds) {
				issues = append(issues, fmt.Sprintf("accessibility_tree node %s bounds %+v do not match component_tree bounds %+v", node.Name, node.Bounds, componentNode.Bounds))
			}
		}
		if node.Bounds.W <= 0 || node.Bounds.H <= 0 {
			issues = append(issues, fmt.Sprintf("accessibility_tree node %s bounds are required", node.Name))
		}
		if !node.Visible || !node.Enabled {
			issues = append(issues, fmt.Sprintf("accessibility_tree node %s must be visible and enabled", node.Name))
		}
		if node.Focused {
			focusedCount++
		}
		if node.Role == "textbox" && !node.Editable {
			issues = append(issues, fmt.Sprintf("accessibility_tree textbox %s must be editable", node.Name))
		}
		if node.Role != "textbox" && node.Editable {
			issues = append(issues, fmt.Sprintf("accessibility_tree non-textbox node %s must not be editable", node.Name))
		}
		if node.Readonly || node.Required || node.Pressed || node.Invalid {
			issues = append(issues, fmt.Sprintf("accessibility_tree node %s unexpected readonly/required/pressed/invalid state", node.Name))
		}
	}
	if focusedCount != 1 {
		issues = append(issues, fmt.Sprintf("accessibility_tree focused node count = %d, want exactly 1", focusedCount))
	}
	for _, count := range []struct {
		name string
		got  int
		want int
	}{
		{name: "focusable_count", got: tree.FocusableCount, want: 4},
		{name: "label_count", got: tree.LabelCount, want: 2},
		{name: "textbox_count", got: tree.TextBoxCount, want: 2},
		{name: "button_count", got: tree.ButtonCount, want: 2},
		{name: "status_count", got: tree.StatusCount, want: 1},
	} {
		if count.got != count.want {
			issues = append(issues, fmt.Sprintf("accessibility_tree %s = %d, want %d", count.name, count.got, count.want))
		}
	}
	if roleCounts["label"] != tree.LabelCount || roleCounts["textbox"] != tree.TextBoxCount || roleCounts["button"] != tree.ButtonCount || roleCounts["status"] != tree.StatusCount {
		issues = append(issues, "accessibility_tree role counts must match node roles")
	}

	for _, expected := range []struct {
		id      int
		parent  int
		name    string
		role    string
		focus   int
		reading int
		actions []string
	}{
		{id: 0, parent: -1, name: "AccessibilitySettingsApp", role: "root", focus: -1, reading: 0},
		{id: 1, parent: 0, name: "Panel", role: "panel", focus: -1, reading: 1},
		{id: 2, parent: 1, name: "Column", role: "column", focus: -1, reading: 2},
		{id: 3, parent: 2, name: "TitleText", role: "text", focus: -1, reading: 3},
		{id: 4, parent: 2, name: "NameLabel", role: "label", focus: -1, reading: 4},
		{id: 5, parent: 2, name: "NameTextBox", role: "textbox", focus: 0, reading: 5, actions: []string{"focus", "edit"}},
		{id: 6, parent: 2, name: "EmailLabel", role: "label", focus: -1, reading: 6},
		{id: 7, parent: 2, name: "EmailTextBox", role: "textbox", focus: 1, reading: 7, actions: []string{"focus", "edit"}},
		{id: 8, parent: 2, name: "ButtonRow", role: "row", focus: -1, reading: 8},
		{id: 9, parent: 8, name: "SaveButton", role: "button", focus: 2, reading: 9, actions: []string{"focus", "press", "save"}},
		{id: 10, parent: 8, name: "ResetButton", role: "button", focus: 3, reading: 10, actions: []string{"focus", "press", "reset"}},
		{id: 11, parent: 2, name: "StatusText", role: "status", focus: -1, reading: 11},
	} {
		node, ok := nodesByID[expected.id]
		if !ok {
			issues = append(issues, fmt.Sprintf("accessibility_tree missing node id %d (%s)", expected.id, expected.name))
			continue
		}
		if node.Name != expected.name || node.Role != expected.role || node.ParentID != expected.parent {
			issues = append(issues, fmt.Sprintf("accessibility_tree node %d = %s/%s parent %d, want %s/%s parent %d", expected.id, node.Name, node.Role, node.ParentID, expected.name, expected.role, expected.parent))
		}
		if node.FocusIndex != expected.focus {
			issues = append(issues, fmt.Sprintf("accessibility_tree node %s focus_index = %d, want %d", expected.name, node.FocusIndex, expected.focus))
		}
		if node.ReadingIndex != expected.reading {
			issues = append(issues, fmt.Sprintf("accessibility_tree node %s reading_index = %d, want %d", expected.name, node.ReadingIndex, expected.reading))
		}
		for _, action := range expected.actions {
			if !contains(node.Actions, action) {
				issues = append(issues, fmt.Sprintf("accessibility_tree node %s actions missing %s", expected.name, action))
			}
		}
	}

	for _, relation := range []struct {
		kind string
		from string
		to   string
	}{
		{kind: "label_for", from: "NameLabel", to: "NameTextBox"},
		{kind: "labelled_by", from: "NameTextBox", to: "NameLabel"},
		{kind: "label_for", from: "EmailLabel", to: "EmailTextBox"},
		{kind: "labelled_by", from: "EmailTextBox", to: "EmailLabel"},
	} {
		if !hasAccessibilityRelationship(tree.Relationships, relation.kind, relation.from, relation.to) {
			issues = append(issues, fmt.Sprintf("accessibility_tree relationships missing %s %s %s", relation.from, relation.kind, relation.to))
		}
	}
	if nameBox, ok := nodesByName["NameTextBox"]; ok && nameBox.LabelledBy != "NameLabel" {
		issues = append(issues, fmt.Sprintf("accessibility_tree NameTextBox labelled_by is %q, want NameLabel", nameBox.LabelledBy))
	}
	if emailBox, ok := nodesByName["EmailTextBox"]; ok && emailBox.LabelledBy != "EmailLabel" {
		issues = append(issues, fmt.Sprintf("accessibility_tree EmailTextBox labelled_by is %q, want EmailLabel", emailBox.LabelledBy))
	}
	if nameLabel, ok := nodesByName["NameLabel"]; ok && nameLabel.LabelFor != "NameTextBox" {
		issues = append(issues, fmt.Sprintf("accessibility_tree NameLabel label_for is %q, want NameTextBox", nameLabel.LabelFor))
	}
	if emailLabel, ok := nodesByName["EmailLabel"]; ok && emailLabel.LabelFor != "EmailTextBox" {
		issues = append(issues, fmt.Sprintf("accessibility_tree EmailLabel label_for is %q, want EmailTextBox", emailLabel.LabelFor))
	}

	wantFocusOrder := []string{"NameTextBox", "EmailTextBox", "SaveButton", "ResetButton"}
	if !stringSlicesEqual(tree.FocusOrder, wantFocusOrder) {
		issues = append(issues, fmt.Sprintf("accessibility_tree focus_order = %v, want %v", tree.FocusOrder, wantFocusOrder))
	}
	wantReadingOrder := []string{"TitleText", "NameLabel", "NameTextBox", "EmailLabel", "EmailTextBox", "SaveButton", "ResetButton", "StatusText"}
	if !stringSlicesEqual(tree.ReadingOrder, wantReadingOrder) {
		issues = append(issues, fmt.Sprintf("accessibility_tree reading_order = %v, want %v", tree.ReadingOrder, wantReadingOrder))
	}

	for _, action := range []struct {
		target   string
		action   string
		semantic string
	}{
		{target: "NameTextBox", action: "edit", semantic: "text-input"},
		{target: "EmailTextBox", action: "edit", semantic: "text-input"},
		{target: "SaveButton", action: "press", semantic: "save"},
		{target: "ResetButton", action: "press", semantic: "reset"},
	} {
		if !hasAccessibilityAction(tree.Actions, action.target, action.action, action.semantic) {
			issues = append(issues, fmt.Sprintf("accessibility_tree actions missing %s %s/%s", action.target, action.action, action.semantic))
		}
	}

	issues = append(issues, validateAccessibilitySnapshots(tree.Snapshots)...)
	issues = append(issues, validateAccessibilityNegativeGuards(tree.NegativeGuards)...)

	for _, required := range []string{
		"accessibility metadata tree schema",
		"accessibility metadata roles labels values states",
		"accessibility metadata component tree alignment",
		"accessibility metadata focus order alignment",
		"accessibility metadata reading order",
		"accessibility metadata snapshots update",
		"accessibility metadata no DOM ARIA platform host claim",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("accessibility_tree report requires %s evidence", required))
		}
	}
	if releaseAccessibility {
		for _, required := range []string{
			"accessibility platform bridge v1 schema",
			"linux accessibility host bridge export",
			"accessibility release honest screen reader evidence",
		} {
			if !caseNameContains(report.Cases, required) {
				issues = append(issues, fmt.Sprintf("accessibility_tree release report requires %s evidence", required))
			}
		}
	}
	return issues
}

func validateReleaseAccessibilityBridgeEvidence(report Report, tree *AccessibilityTreeReport) []string {
	var issues []string
	if tree.AccessibilityLevel != "platform-bridge-v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree accessibility_level is %q, want platform-bridge-v1", tree.AccessibilityLevel))
	}
	if tree.ReleaseScope != "surface-v1-linux-web" {
		issues = append(issues, fmt.Sprintf("accessibility_tree release_scope is %q, want surface-v1-linux-web", tree.ReleaseScope))
	}
	if tree.Experimental {
		issues = append(issues, "accessibility_tree experimental must be false for release accessibility reports")
	}
	if !tree.ProductionClaim {
		issues = append(issues, "accessibility_tree production_claim must be true for release accessibility reports")
	}
	if !tree.MetadataTree {
		issues = append(issues, "accessibility_tree metadata_tree must be true for platform-bridge-v1")
	}
	if !tree.PlatformExport {
		issues = append(issues, "accessibility_tree platform_export must be true for platform-bridge-v1")
	}
	screenReaderEvidence, ok := screenReaderEvidenceString(tree.ScreenReaderEvidence)
	if !ok || screenReaderEvidence == "" {
		issues = append(issues, "accessibility_tree screen_reader_evidence must name the exact platform-tree evidence")
	}
	if strings.Contains(screenReaderEvidence, "full") || strings.Contains(screenReaderEvidence, "screen-reader-support") {
		issues = append(issues, "accessibility_tree screen_reader_evidence must not claim full screen-reader support without a real probe")
	}
	switch report.Target {
	case "linux-x64":
		if report.HostEvidence.Level != "linux-x64-real-window" {
			issues = append(issues, "accessibility_tree linux release evidence must use linux-x64 real-window host evidence")
		}
		if !report.HostEvidence.AccessibilityBridge {
			issues = append(issues, "accessibility_tree linux release host_evidence.accessibility_bridge must be true")
		}
		if !tree.PlatformHostIntegration {
			issues = append(issues, "accessibility_tree platform_host_integration must be true for linux platform-bridge-v1")
		}
		if tree.PlatformBridge != "linux_accessibility_host_bridge_v1" {
			issues = append(issues, fmt.Sprintf("accessibility_tree platform_bridge is %q, want linux_accessibility_host_bridge_v1", tree.PlatformBridge))
		}
		if !tree.LinuxPlatformProbe {
			issues = append(issues, "accessibility_tree linux_platform_probe must be true for linux platform-bridge-v1")
		}
		if strings.TrimSpace(tree.LinuxProbeArtifact) == "" {
			issues = append(issues, "accessibility_tree linux_probe_artifact is required for linux platform-bridge-v1")
		}
		if screenReaderEvidence != "linux_accessibility_host_bridge_v1" {
			issues = append(issues, fmt.Sprintf("accessibility_tree screen_reader_evidence is %q, want linux_accessibility_host_bridge_v1", screenReaderEvidence))
		}
		if !hasRuntimeProcessName(report.Processes, "surface linux accessibility platform probe") {
			issues = append(issues, "accessibility_tree linux release requires surface linux accessibility platform probe process evidence")
		}
		if !artifactKindContains(report.Artifacts, "linux-accessibility-platform-probe") {
			issues = append(issues, "accessibility_tree linux release requires linux-accessibility-platform-probe artifact")
		}
	case "wasm32-web":
		if !isBrowserCanvasHostEvidenceLevel(report.HostEvidence.Level) {
			issues = append(issues, "accessibility_tree browser release evidence must be browser-canvas input, not Node-only wasm32-web evidence")
		}
		if !report.HostEvidence.BrowserAccessibilitySnapshot {
			issues = append(issues, "accessibility_tree browser release host_evidence.browser_accessibility_snapshot must be true")
		}
		if !report.HostEvidence.BrowserAccessibilityMirror {
			issues = append(issues, "accessibility_tree browser release host_evidence.browser_accessibility_mirror must be true")
		}
		if tree.PlatformBridge != "browser_accessibility_mirror_v1" {
			issues = append(issues, fmt.Sprintf("accessibility_tree platform_bridge is %q, want browser_accessibility_mirror_v1", tree.PlatformBridge))
		}
		if !tree.BrowserAccessibilitySnap {
			issues = append(issues, "accessibility_tree browser_accessibility_snapshot must be true for browser platform-bridge-v1")
		}
		if !tree.BrowserAccessibilityMirror {
			issues = append(issues, "accessibility_tree browser_accessibility_mirror must be true for browser platform-bridge-v1")
		}
		if screenReaderEvidence != "browser_accessibility_snapshot_v1" {
			issues = append(issues, fmt.Sprintf("accessibility_tree screen_reader_evidence is %q, want browser_accessibility_snapshot_v1", screenReaderEvidence))
		}
		if !hasRuntimeProcessName(report.Processes, "surface wasm32-web browser canvas trace") {
			issues = append(issues, "accessibility_tree browser release requires browser canvas trace process evidence")
		}
		if !artifactKindContains(report.Artifacts, "runner-trace") {
			issues = append(issues, "accessibility_tree browser release requires runner-trace artifact")
		}
	case "headless":
		if tree.PlatformBridge == "" {
			issues = append(issues, "accessibility_tree platform_bridge is required for headless platform-bridge-v1")
		}
		if tree.PlatformHostIntegration || tree.LinuxPlatformProbe || strings.TrimSpace(tree.LinuxProbeArtifact) != "" || tree.BrowserAccessibilitySnap || tree.BrowserAccessibilityMirror {
			issues = append(issues, "accessibility_tree headless platform bridge must not claim linux platform probe or browser accessibility mirror")
		}
	default:
		issues = append(issues, fmt.Sprintf("accessibility_tree release target %q is unsupported", report.Target))
	}
	return issues
}

func validateLinuxReleaseWindowAccessibilityBridgeEvidence(report Report, tree *AccessibilityTreeReport) []string {
	var issues []string
	if tree.Schema != "tetra.surface.accessibility-tree.v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree schema is %q, want tetra.surface.accessibility-tree.v1", tree.Schema))
	}
	if tree.AccessibilityLevel != "platform-bridge-v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree accessibility_level is %q, want platform-bridge-v1", tree.AccessibilityLevel))
	}
	if tree.ReleaseScope != "surface-v1-linux-web" {
		issues = append(issues, fmt.Sprintf("accessibility_tree release_scope is %q, want surface-v1-linux-web", tree.ReleaseScope))
	}
	if normalizeEvidencePath(tree.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("accessibility_tree source %q must match report source %q", tree.Source, report.Source))
	}
	if tree.Module != "lib.core.accessibility" {
		issues = append(issues, fmt.Sprintf("accessibility_tree module is %q, want lib.core.accessibility", tree.Module))
	}
	if tree.WidgetModule != "lib.core.widgets" {
		issues = append(issues, fmt.Sprintf("accessibility_tree widget_module is %q, want lib.core.widgets", tree.WidgetModule))
	}
	if tree.Experimental || !tree.ProductionClaim {
		issues = append(issues, "accessibility_tree linux release window requires experimental=false and production_claim=true")
	}
	if !tree.PlatformHostIntegration || !tree.MetadataTree || !tree.PlatformExport {
		issues = append(issues, "accessibility_tree linux release window requires platform_host_integration, metadata_tree, and platform_export")
	}
	if tree.PlatformBridge != "linux_accessibility_host_bridge_v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree platform_bridge is %q, want linux_accessibility_host_bridge_v1", tree.PlatformBridge))
	}
	if !tree.LinuxPlatformProbe || strings.TrimSpace(tree.LinuxProbeArtifact) == "" {
		issues = append(issues, "accessibility_tree linux release window requires linux_platform_probe and linux_probe_artifact")
	}
	screenReaderEvidence, ok := screenReaderEvidenceString(tree.ScreenReaderEvidence)
	if !ok || screenReaderEvidence != "linux_accessibility_host_bridge_v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree screen_reader_evidence is %q, want linux_accessibility_host_bridge_v1", screenReaderEvidence))
	}
	if tree.DOMARIAIntegration || !tree.NoDOMUI || !tree.NoUserJS || !tree.NoPlatformWidgets || !tree.NoLegacySidecars {
		issues = append(issues, "accessibility_tree linux release window must not claim DOM ARIA, DOM UI, user JS, platform widgets, or legacy sidecars")
	}
	if !tree.DerivedFromComponentTree || !tree.UsesComponentTreeAPI || !tree.UsesWidgetToolkit {
		issues = append(issues, "accessibility_tree linux release window must derive from component_tree, component_tree_api, and toolkit evidence")
	}
	if tree.ComponentTreeSchema != "tetra.surface.component-tree.v1" ||
		tree.ComponentTreeAPISchema != "tetra.surface.component-tree-api.v1" ||
		tree.ToolkitSchema != "tetra.surface.toolkit.v1" {
		issues = append(issues, "accessibility_tree linux release window requires component-tree, component-tree-api, and toolkit schemas")
	}
	if report.ComponentTree == nil || report.ComponentTree.DynamicLevel != "production-widgets-v1" {
		issues = append(issues, "accessibility_tree linux release window requires production-widgets-v1 component_tree evidence")
	}
	if report.Toolkit == nil || report.Toolkit.ToolkitLevel != "production-widgets-v1" {
		issues = append(issues, "accessibility_tree linux release window requires production-widgets-v1 toolkit evidence")
	}
	for _, role := range []string{"root", "textbox", "checkbox", "button", "status"} {
		if !contains(tree.RolesPresent, role) {
			issues = append(issues, fmt.Sprintf("accessibility_tree linux release window roles_present missing %s", role))
		}
	}
	for _, focus := range []string{"NameTextBox", "EmailTextBox", "SubscribeCheckbox", "SaveButton", "ResetButton"} {
		if !contains(tree.FocusOrder, focus) {
			issues = append(issues, fmt.Sprintf("accessibility_tree linux release window focus_order missing %s", focus))
		}
	}
	return issues
}

func validateAccessibilityToolkitEvidence(report Report) []string {
	var issues []string
	if report.Toolkit == nil {
		return []string{"accessibility_tree requires toolkit evidence"}
	}
	toolkit := report.Toolkit
	if toolkit.Schema != "tetra.surface.toolkit.v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree toolkit schema is %q, want tetra.surface.toolkit.v1", toolkit.Schema))
	}
	if toolkit.ToolkitLevel != "toolkit-reuse-v1" || toolkit.ReuseLevel != "multi-form-widget-reuse-v1" {
		issues = append(issues, "accessibility_tree toolkit must reuse toolkit-reuse-v1 multi-form widgets")
	}
	if normalizeEvidencePath(toolkit.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("accessibility_tree toolkit source %q must match report source %q", toolkit.Source, report.Source))
	}
	if toolkit.Module != "lib.core.widgets" {
		issues = append(issues, fmt.Sprintf("accessibility_tree toolkit module is %q, want lib.core.widgets", toolkit.Module))
	}
	if !toolkit.Experimental || toolkit.ProductionClaim || !toolkit.UsesComponentTreeAPI || toolkit.ManualBookkeeping || toolkit.DemoSpecificWidgetStructs {
		issues = append(issues, "accessibility_tree toolkit must be experimental reusable component-tree evidence without production/manual/demo widget claims")
	}
	if !toolkit.NoMagicWidgets || !toolkit.NoPlatformWidgets || !toolkit.NoDOMUI || !toolkit.NoUserJS {
		issues = append(issues, "accessibility_tree toolkit must prove no magic, platform, DOM, or user-JS widgets")
	}
	if toolkit.ExampleCount < 3 || !contains(toolkit.Sources, "examples/surface_accessibility_settings.tetra") {
		issues = append(issues, "accessibility_tree toolkit example_count must include surface_accessibility_settings")
	}
	if toolkit.TextBoxCount < 2 || toolkit.ButtonCount < 2 || !toolkit.MultiTextBoxEvidence || !toolkit.MultiFormEvidence {
		issues = append(issues, "accessibility_tree toolkit requires multi TextBox and multi Button reuse evidence")
	}

	widgets := map[string]ToolkitWidgetReport{}
	for _, widget := range toolkit.Widgets {
		widgets[widget.Name] = widget
	}
	for _, required := range []struct {
		name     string
		kind     string
		nodeID   int
		role     string
		action   string
		editable bool
	}{
		{name: "Panel", kind: "Panel", nodeID: 1},
		{name: "Column", kind: "Column", nodeID: 2},
		{name: "TitleText", kind: "Text", nodeID: 3, role: "text"},
		{name: "NameLabel", kind: "Text", nodeID: 4, role: "label"},
		{name: "NameTextBox", kind: "TextBox", nodeID: 5, editable: true},
		{name: "EmailLabel", kind: "Text", nodeID: 6, role: "label"},
		{name: "EmailTextBox", kind: "TextBox", nodeID: 7, editable: true},
		{name: "ButtonRow", kind: "Row", nodeID: 8},
		{name: "SaveButton", kind: "Button", nodeID: 9, action: "save"},
		{name: "ResetButton", kind: "Button", nodeID: 10, action: "reset"},
		{name: "StatusText", kind: "Text", nodeID: 11, role: "status"},
	} {
		widget, ok := widgets[required.name]
		if !ok {
			issues = append(issues, fmt.Sprintf("accessibility_tree toolkit widget evidence missing %s", required.name))
			continue
		}
		if widget.Kind != required.kind || widget.NodeID != required.nodeID {
			issues = append(issues, fmt.Sprintf("accessibility_tree toolkit widget %s = %s/%d, want %s/%d", required.name, widget.Kind, widget.NodeID, required.kind, required.nodeID))
		}
		if required.role != "" && widget.Role != required.role {
			issues = append(issues, fmt.Sprintf("accessibility_tree toolkit widget %s role is %q, want %q", required.name, widget.Role, required.role))
		}
		if required.action != "" && widget.Action != required.action {
			issues = append(issues, fmt.Sprintf("accessibility_tree toolkit widget %s action is %q, want %q", required.name, widget.Action, required.action))
		}
		if required.editable && !widget.Editable {
			issues = append(issues, fmt.Sprintf("accessibility_tree toolkit widget %s must prove editable=true", required.name))
		}
		if !widget.Reusable || !widget.OrdinaryTetraStruct {
			issues = append(issues, fmt.Sprintf("accessibility_tree toolkit widget %s must be reusable ordinary Tetra struct evidence", required.name))
		}
	}
	for _, source := range []string{
		"lib/core/widgets.tetra:add_accessible_textbox",
		"lib/core/widgets.tetra:add_accessible_button",
		"lib/core/widgets.tetra:add_accessible_status",
	} {
		if !contains(toolkit.ReusableSources, source) {
			issues = append(issues, fmt.Sprintf("accessibility_tree toolkit reusable_sources missing %s", source))
		}
	}
	return issues
}

func validateAccessibilitySnapshots(snapshots []AccessibilitySnapshotReport) []string {
	var issues []string
	if len(snapshots) == 0 {
		return []string{"accessibility_tree snapshots are required"}
	}
	byName := map[string]AccessibilitySnapshotReport{}
	lastGeneration := 0
	for _, snapshot := range snapshots {
		if strings.TrimSpace(snapshot.Name) == "" {
			issues = append(issues, "accessibility_tree snapshot name is required")
		} else if _, exists := byName[snapshot.Name]; exists {
			issues = append(issues, fmt.Sprintf("accessibility_tree duplicate snapshot %s", snapshot.Name))
		}
		byName[snapshot.Name] = snapshot
		if snapshot.Generation <= lastGeneration {
			issues = append(issues, fmt.Sprintf("accessibility_tree snapshot %s generation = %d, want greater than %d", snapshot.Name, snapshot.Generation, lastGeneration))
		}
		lastGeneration = snapshot.Generation
		if !validChecksumLike(snapshot.BoundsChecksum) {
			issues = append(issues, fmt.Sprintf("accessibility_tree snapshot %s bounds_checksum is invalid", snapshot.Name))
		}
		if !validChecksumLike(snapshot.MetadataChecksum) {
			issues = append(issues, fmt.Sprintf("accessibility_tree snapshot %s metadata_checksum is invalid", snapshot.Name))
		}
		if !validChecksumLike(snapshot.FrameChecksum) {
			issues = append(issues, fmt.Sprintf("accessibility_tree snapshot %s frame_checksum is invalid", snapshot.Name))
		}
	}
	for _, expected := range []struct {
		name        string
		focused     string
		componentID int
		nodeID      int
		nameLen     int
		emailLen    int
		status      string
	}{
		{name: "initial", focused: "", componentID: -1, nodeID: -1, nameLen: 0, emailLen: 0, status: "idle"},
		{name: "after_name_focus", focused: "NameTextBox", componentID: 5, nodeID: 5, nameLen: 0, emailLen: 0, status: "idle"},
		{name: "after_name_text", focused: "NameTextBox", componentID: 5, nodeID: 5, nameLen: 3, emailLen: 0, status: "idle"},
		{name: "after_email_focus", focused: "EmailTextBox", componentID: 7, nodeID: 7, nameLen: 3, emailLen: 0, status: "idle"},
		{name: "after_email_text", focused: "EmailTextBox", componentID: 7, nodeID: 7, nameLen: 3, emailLen: 5, status: "idle"},
		{name: "after_save", focused: "SaveButton", componentID: 9, nodeID: 9, nameLen: 3, emailLen: 5, status: "saved"},
		{name: "after_reset", focused: "ResetButton", componentID: 10, nodeID: 10, nameLen: 0, emailLen: 0, status: "reset"},
		{name: "after_resize", focused: "NameTextBox", componentID: 5, nodeID: 5, nameLen: 0, emailLen: 0, status: "reset"},
	} {
		snapshot, ok := byName[expected.name]
		if !ok {
			issues = append(issues, fmt.Sprintf("accessibility_tree snapshots missing %s", expected.name))
			continue
		}
		if snapshot.Focused != expected.focused || snapshot.FocusedComponentID != expected.componentID || snapshot.FocusedAccessibilityNodeID != expected.nodeID {
			issues = append(issues, fmt.Sprintf("accessibility_tree snapshot %s focused = %s/%d/%d, want %s/%d/%d", expected.name, snapshot.Focused, snapshot.FocusedComponentID, snapshot.FocusedAccessibilityNodeID, expected.focused, expected.componentID, expected.nodeID))
		}
		if snapshot.NameValueLen != expected.nameLen || snapshot.EmailValueLen != expected.emailLen {
			issues = append(issues, fmt.Sprintf("accessibility_tree snapshot %s value lengths = %d/%d, want %d/%d", expected.name, snapshot.NameValueLen, snapshot.EmailValueLen, expected.nameLen, expected.emailLen))
		}
		if snapshot.StatusValue != expected.status {
			issues = append(issues, fmt.Sprintf("accessibility_tree snapshot %s status_value = %q, want %q", expected.name, snapshot.StatusValue, expected.status))
		}
	}
	if afterNameText, ok := byName["after_name_text"]; ok {
		if afterNameFocus, ok := byName["after_name_focus"]; ok && afterNameText.MetadataChecksum == afterNameFocus.MetadataChecksum {
			issues = append(issues, "accessibility_tree metadata_checksum must change after_name_text")
		}
	}
	if afterEmailText, ok := byName["after_email_text"]; ok {
		if afterEmailFocus, ok := byName["after_email_focus"]; ok && afterEmailText.MetadataChecksum == afterEmailFocus.MetadataChecksum {
			issues = append(issues, "accessibility_tree metadata_checksum must change after_email_text")
		}
	}
	if afterResize, ok := byName["after_resize"]; ok {
		if afterReset, ok := byName["after_reset"]; ok {
			if afterResize.BoundsChecksum == afterReset.BoundsChecksum {
				issues = append(issues, "accessibility_tree bounds_checksum must change after_resize")
			}
			if afterResize.FrameChecksum == afterReset.FrameChecksum {
				issues = append(issues, "accessibility_tree frame_checksum must change after_resize")
			}
		}
	}
	return issues
}

func validateAccessibilityNegativeGuards(guards AccessibilityNegativeGuardsReport) []string {
	var missing []string
	if !guards.NoBorrowedViewStorage {
		missing = append(missing, "no_borrowed_view_storage")
	}
	if !guards.ComponentIDAlignmentChecked {
		missing = append(missing, "component_id_alignment_checked")
	}
	if !guards.BoundsAlignmentChecked {
		missing = append(missing, "bounds_alignment_checked")
	}
	if !guards.FocusOrderAlignmentChecked {
		missing = append(missing, "focus_order_alignment_checked")
	}
	if !guards.ReadingOrderChecked {
		missing = append(missing, "reading_order_checked")
	}
	if !guards.LabelRelationshipsChecked {
		missing = append(missing, "label_relationships_checked")
	}
	if !guards.StateUpdatesChecked {
		missing = append(missing, "state_updates_checked")
	}
	if !guards.ArtifactScanChecked {
		missing = append(missing, "artifact_scan_checked")
	}
	if len(missing) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("accessibility_tree negative_guards missing %s", strings.Join(missing, ", "))}
}

func hasAccessibilityRelationship(relationships []AccessibilityRelationshipReport, kind string, from string, to string) bool {
	for _, relationship := range relationships {
		if relationship.Kind == kind && relationship.From == from && relationship.To == to {
			return true
		}
	}
	return false
}

func hasAccessibilityAction(actions []AccessibilityActionReport, target string, action string, semantic string) bool {
	for _, item := range actions {
		if item.Target == target && item.Action == action && item.Semantic == semantic {
			return true
		}
	}
	return false
}

func rectsEqual(a RectReport, b RectReport) bool {
	return a.X == b.X && a.Y == b.Y && a.W == b.W && a.H == b.H
}

func validChecksumLike(value string) bool {
	value = strings.TrimSpace(value)
	if validSHA256Digest(value) {
		return true
	}
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func hasMinimalToolkitButtonAction(events []EventReport, target string, countSuffix string, statusKey string) bool {
	for _, event := range events {
		if event.TargetComponent != target || event.Kind != "key_down" || !event.Handled || !event.Pass {
			continue
		}
		if event.Key != 32 && event.Key != 13 {
			continue
		}
		if !dispatchPathHasSuffix(event.DispatchPath, "Panel", "Column", "ButtonRow", target) {
			continue
		}
		if !stateChangedBySuffix(event.BeforeState, event.AfterState, "."+countSuffix) {
			continue
		}
		if event.BeforeState[statusKey] == event.AfterState[statusKey] {
			continue
		}
		return true
	}
	return false
}

func validateComponentTreeSiblingLayout(nodes map[int]ComponentTreeNodeReport, childrenByParent map[int][]ComponentTreeNodeReport) []string {
	var issues []string
	for parentID, children := range childrenByParent {
		parent := nodes[parentID]
		kind := strings.ToLower(strings.TrimSpace(parent.Kind))
		if kind != "column" && kind != "row" {
			continue
		}
		ordered := append([]ComponentTreeNodeReport(nil), children...)
		sort.SliceStable(ordered, func(i int, j int) bool {
			return ordered[i].ChildIndex < ordered[j].ChildIndex
		})
		for i := 1; i < len(ordered); i++ {
			prev := ordered[i-1]
			child := ordered[i]
			switch kind {
			case "column":
				if child.Bounds.Y < prev.Bounds.Y+prev.Bounds.H {
					issues = append(issues, fmt.Sprintf("component_tree Column node %d child_index %d node %d overlaps or precedes child_index %d node %d", parentID, child.ChildIndex, child.ID, prev.ChildIndex, prev.ID))
				}
			case "row":
				if child.Bounds.X < prev.Bounds.X+prev.Bounds.W {
					issues = append(issues, fmt.Sprintf("component_tree Row node %d child_index %d node %d overlaps child_index %d node %d", parentID, child.ChildIndex, child.ID, prev.ChildIndex, prev.ID))
				}
			}
		}
	}
	return issues
}

func validateComponentTreeAPIEvidence(report Report, tree *ComponentTreeReport, expectedPaths map[int][]int, nodes map[int]ComponentTreeNodeReport) []string {
	if tree == nil {
		return nil
	}
	api := report.ComponentTreeAPI
	if api == nil {
		return []string{"component_tree_api evidence is required for component tree API hardening reports"}
	}
	var issues []string
	accessibility := isAccessibilityMetadataReport(report)
	if api.Schema != "tetra.surface.component-tree-api.v1" {
		issues = append(issues, fmt.Sprintf("component_tree_api schema is %q, want tetra.surface.component-tree-api.v1", api.Schema))
	}
	if api.APILevel != "builder-layout-dispatch-v1" {
		issues = append(issues, fmt.Sprintf("component_tree_api api_level is %q, want builder-layout-dispatch-v1", api.APILevel))
	}
	if normalizeEvidencePath(api.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("component_tree_api source %q must match report source %q", api.Source, report.Source))
	}
	if api.ManualBookkeeping {
		issues = append(issues, "component_tree_api manual_bookkeeping must be false")
	}
	if api.Builder.RootCreatedBy != "tree_add_root" {
		issues = append(issues, fmt.Sprintf("component_tree_api builder root_created_by is %q, want tree_add_root", api.Builder.RootCreatedBy))
	}
	if api.Builder.ChildrenCreatedBy != "tree_add_child" {
		issues = append(issues, fmt.Sprintf("component_tree_api builder children_created_by is %q, want tree_add_child", api.Builder.ChildrenCreatedBy))
	}
	if api.Builder.NodeCount != tree.NodeCount {
		issues = append(issues, fmt.Sprintf("component_tree_api builder node_count = %d, want component_tree node_count %d", api.Builder.NodeCount, tree.NodeCount))
	}
	if api.Builder.Capacity < tree.NodeCount {
		issues = append(issues, fmt.Sprintf("component_tree_api builder capacity = %d, want at least node_count %d", api.Builder.Capacity, tree.NodeCount))
	}
	if !api.Builder.OverflowChecked {
		issues = append(issues, "component_tree_api builder must prove overflow_checked")
	}
	if !api.Invariants.TreeValidateRan {
		issues = append(issues, "component_tree_api invariants require tree_validate_ran")
	}
	if api.Invariants.TreeValidateStatus != 0 {
		issues = append(issues, fmt.Sprintf("component_tree_api tree_validate_status = %d, want 0", api.Invariants.TreeValidateStatus))
	}
	if !api.Invariants.ParentChildLinksChecked || !api.Invariants.ChildIndicesChecked || !api.Invariants.ChildCountChecked || !api.Invariants.FirstChildChecked {
		issues = append(issues, "component_tree_api invariants must check parent/child links, child indices, child_count, and first_child")
	}
	if !hasComponentTreeAPILayout(api.LayoutHelpers, []string{"tree_layout_column", "widgets.column_layout"}, "Column") {
		issues = append(issues, "component_tree_api layout_helpers require changed tree_layout_column or widgets.column_layout evidence for Column")
	}
	if !hasComponentTreeAPILayout(api.LayoutHelpers, []string{"tree_layout_row", "widgets.row_layout"}, "ButtonRow") {
		issues = append(issues, "component_tree_api layout_helpers require changed tree_layout_row or widgets.row_layout evidence for ButtonRow")
	}
	focusPairs := [][2]string{
		{"TextBox", "SubmitButton"},
		{"SubmitButton", "ResetButton"},
		{"ResetButton", "TextBox"},
	}
	dispatchTargets := []string{"TextBox", "SubmitButton", "ResetButton"}
	if isProductionToolkitReport(report) {
		focusPairs = [][2]string{
			{"NameTextBox", "EmailTextBox"},
			{"EmailTextBox", "SubscribeCheckbox"},
			{"SubscribeCheckbox", "SaveButton"},
			{"SaveButton", "ResetButton"},
			{"ResetButton", "NameTextBox"},
		}
		dispatchTargets = []string{"NameTextBox", "EmailTextBox", "SubscribeCheckbox", "SaveButton", "ResetButton"}
	} else if accessibility || isToolkitReuseReport(report) {
		focusPairs = [][2]string{
			{"NameTextBox", "EmailTextBox"},
			{"EmailTextBox", "SaveButton"},
			{"SaveButton", "ResetButton"},
			{"ResetButton", "NameTextBox"},
		}
		dispatchTargets = []string{"NameTextBox", "EmailTextBox", "SaveButton", "ResetButton"}
	}
	for _, pair := range focusPairs {
		if !hasComponentTreeAPIFocusTransition(api.FocusHelpers, pair[0], pair[1]) {
			issues = append(issues, fmt.Sprintf("component_tree_api focus_helpers require %s -> %s", pair[0], pair[1]))
		}
	}
	primaryTextBox := dispatchTargets[0]
	if !hasComponentTreeAPIHitTest(api.HitTests, primaryTextBox, expectedPathForComponentTreeTarget(nodes, expectedPaths, primaryTextBox)) {
		issues = append(issues, fmt.Sprintf("component_tree_api hit_tests require %s path evidence", primaryTextBox))
	}
	if (accessibility || isProductionToolkitReport(report) || isToolkitReuseReport(report)) && !hasComponentTreeAPIHitTest(api.HitTests, "EmailTextBox", expectedPathForComponentTreeTarget(nodes, expectedPaths, "EmailTextBox")) {
		issues = append(issues, "component_tree_api hit_tests require EmailTextBox path evidence")
	}
	if isProductionToolkitReport(report) && !hasComponentTreeAPIHitTest(api.HitTests, "SubscribeCheckbox", expectedPathForComponentTreeTarget(nodes, expectedPaths, "SubscribeCheckbox")) {
		issues = append(issues, "component_tree_api hit_tests require SubscribeCheckbox path evidence")
	}
	if !hasComponentTreeAPIHitTest(api.HitTests, "ResetButton", expectedPathForComponentTreeTarget(nodes, expectedPaths, "ResetButton")) &&
		!hasComponentTreeAPIHitTest(api.HitTests, "SubmitButton", expectedPathForComponentTreeTarget(nodes, expectedPaths, "SubmitButton")) &&
		!hasComponentTreeAPIHitTest(api.HitTests, "SaveButton", expectedPathForComponentTreeTarget(nodes, expectedPaths, "SaveButton")) {
		issues = append(issues, "component_tree_api hit_tests require Button path evidence")
	}
	for _, target := range dispatchTargets {
		wantPath := expectedPathForComponentTreeTarget(nodes, expectedPaths, target)
		if len(wantPath) == 0 {
			continue
		}
		if !hasComponentTreeAPIDispatchPath(api.DispatchPaths, target, wantPath) {
			issues = append(issues, fmt.Sprintf("component_tree_api dispatch_paths require tree_build_dispatch_path %s path %v", target, wantPath))
		}
	}
	for _, required := range []string{
		"component tree api builder node creation",
		"component tree api parent child invariants",
		"component tree api layout helper dispatch",
		"component tree api hit test helper",
		"component tree api focus helper traversal",
		"component tree api dispatch path helper",
		"component tree api no manual bookkeeping",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("component_tree_api report requires %s evidence", required))
		}
	}
	return issues
}

func hasComponentTreeAPILayout(layouts []ComponentTreeAPILayoutHelperReport, helpers []string, target string) bool {
	for _, layout := range layouts {
		if layout.Target != target || !layout.ChangedBounds {
			continue
		}
		for _, helper := range helpers {
			if layout.Helper == helper {
				return true
			}
		}
	}
	return false
}

func hasComponentTreeAPIFocusTransition(focus []ComponentTreeAPIFocusHelperReport, before string, after string) bool {
	for _, item := range focus {
		if item.Helper == "tree_focus_next" && item.Before == before && item.After == after {
			return true
		}
	}
	return false
}

func hasComponentTreeAPIHitTest(hits []ComponentTreeAPIHitTestReport, target string, wantPath []int) bool {
	if len(wantPath) == 0 {
		return false
	}
	for _, hit := range hits {
		if (hit.Helper == "tree_hit_test" || hit.Helper == "widgets.hit_test" || strings.HasPrefix(hit.Helper, "widgets.hit_test_")) && hit.Target == target && intSlicesEqual(hit.Path, wantPath) {
			return true
		}
	}
	return false
}

func hasComponentTreeAPIDispatchPath(paths []ComponentTreeAPIDispatchPathReport, target string, wantPath []int) bool {
	for _, path := range paths {
		if path.Helper == "tree_build_dispatch_path" && path.Target == target && intSlicesEqual(path.Path, wantPath) {
			return true
		}
	}
	return false
}

func expectedPathForComponentTreeTarget(nodes map[int]ComponentTreeNodeReport, expectedPaths map[int][]int, target string) []int {
	for id, node := range nodes {
		if node.Name == target {
			return expectedPaths[id]
		}
	}
	return nil
}

func isComponentTreeReport(report Report) bool {
	if isAccessibilityMetadataReport(report) {
		return true
	}
	if isProductionToolkitReport(report) {
		return true
	}
	if isSurfaceTreeAppSource(report.Source) {
		return true
	}
	if isMinimalToolkitReport(report) {
		return true
	}
	if caseNameContains(report.Cases, "component tree") {
		return true
	}
	return report.ComponentTree != nil
}

func isSurfaceTreeAppSource(source string) bool {
	return strings.HasSuffix(normalizeEvidencePath(source), "examples/surface_tree_app.tetra")
}

func isMinimalToolkitReport(report Report) bool {
	if isAccessibilityMetadataReport(report) {
		return false
	}
	if isProductionToolkitReport(report) {
		return false
	}
	if isToolkitReuseReport(report) {
		return true
	}
	if isSurfaceToolkitFormSource(report.Source) {
		return true
	}
	if report.Toolkit != nil {
		return true
	}
	return caseNameContains(report.Cases, "minimal toolkit")
}

func isToolkitReuseReport(report Report) bool {
	if isAccessibilityMetadataReport(report) {
		return false
	}
	if isProductionToolkitReport(report) {
		return false
	}
	if isSurfaceToolkitSettingsSource(report.Source) {
		return true
	}
	if report.Toolkit != nil && (report.Toolkit.ToolkitLevel == "toolkit-reuse-v1" || report.Toolkit.ReuseLevel == "multi-form-widget-reuse-v1") {
		return true
	}
	return caseNameContains(report.Cases, "toolkit reuse")
}

func isProductionToolkitReport(report Report) bool {
	if isLinuxReleaseWindowReport(report) {
		return true
	}
	if isAccessibilityMetadataReport(report) {
		return false
	}
	if isSurfaceReleaseFormSource(report.Source) {
		return true
	}
	if report.Toolkit != nil && report.Toolkit.ToolkitLevel == "production-widgets-v1" {
		return true
	}
	return caseNameContains(report.Cases, "production toolkit")
}

func isBrowserCanvasHostEvidenceLevel(level string) bool {
	return level == "wasm32-web-browser-canvas-input" ||
		level == "wasm32-web-browser-canvas-release-v1"
}

func isLinuxRealWindowHostEvidenceLevel(level string) bool {
	return level == "linux-x64-real-window" ||
		level == "linux-x64-release-window-v1"
}

func isBrowserReleaseReport(report Report) bool {
	if report.HostEvidence.Level == "wasm32-web-browser-canvas-release-v1" {
		return true
	}
	return caseNameContains(report.Cases, "browser release Surface v1 schema")
}

func isLinuxReleaseWindowReport(report Report) bool {
	if report.HostEvidence.Level == "linux-x64-release-window-v1" {
		return true
	}
	return caseNameContains(report.Cases, "linux release window v1 schema")
}

func isAccessibilityMetadataReport(report Report) bool {
	if isSurfaceAccessibilitySettingsSource(report.Source) || isSurfaceReleaseAccessibilitySource(report.Source) {
		return true
	}
	if isLinuxReleaseWindowReport(report) {
		return true
	}
	if report.AccessibilityTree != nil {
		return true
	}
	return caseNameContains(report.Cases, "accessibility metadata tree") || caseNameContains(report.Cases, "accessibility platform bridge")
}

func isSurfaceToolkitFormSource(source string) bool {
	return strings.HasSuffix(normalizeEvidencePath(source), "examples/surface_toolkit_form.tetra")
}

func isSurfaceToolkitSettingsSource(source string) bool {
	return strings.HasSuffix(normalizeEvidencePath(source), "examples/surface_toolkit_settings.tetra")
}

func isSurfaceReleaseFormSource(source string) bool {
	return strings.HasSuffix(normalizeEvidencePath(source), "examples/surface_release_form.tetra")
}

func isSurfaceAccessibilitySettingsSource(source string) bool {
	return strings.HasSuffix(normalizeEvidencePath(source), "examples/surface_accessibility_settings.tetra")
}

func isSurfaceReleaseAccessibilitySource(source string) bool {
	return strings.HasSuffix(normalizeEvidencePath(source), "examples/surface_release_accessibility.tetra")
}

func isPlatformBridgeAccessibilityReport(report Report) bool {
	return report.AccessibilityTree != nil && report.AccessibilityTree.AccessibilityLevel == "platform-bridge-v1"
}

func componentTreeNodeByName(nodes []ComponentTreeNodeReport, name string) (ComponentTreeNodeReport, bool) {
	for _, node := range nodes {
		if node.Name == name {
			return node, true
		}
	}
	return ComponentTreeNodeReport{}, false
}

func componentTreeNodeByKind(nodes []ComponentTreeNodeReport, kind string) (ComponentTreeNodeReport, bool) {
	for _, node := range nodes {
		if strings.EqualFold(node.Kind, kind) {
			return node, true
		}
	}
	return ComponentTreeNodeReport{}, false
}

func componentTreeDrawOrderCoversNodes(drawOrder []int, nodes map[int]ComponentTreeNodeReport) bool {
	if len(drawOrder) != len(nodes) {
		return false
	}
	seen := map[int]bool{}
	for _, id := range drawOrder {
		if _, ok := nodes[id]; !ok || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}

func componentTreeHasResizeLayoutPass(passes []ComponentTreeLayoutPassReport, id int) bool {
	initialWidth := -1
	resizeWidth := -1
	for _, pass := range passes {
		if pass.ComponentID != id || pass.Bounds.W <= 0 || pass.Bounds.H <= 0 || pass.Measured.W <= 0 || pass.Measured.H <= 0 {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(pass.Pass)) {
		case "initial":
			initialWidth = pass.Bounds.W
		case "resize":
			resizeWidth = pass.Bounds.W
		}
	}
	return initialWidth > 0 && resizeWidth > 0 && initialWidth != resizeWidth
}

func validateComponentTreeDispatchPaths(paths []ComponentTreeDispatchPathReport, expected map[int][]int, nodes map[int]ComponentTreeNodeReport) []string {
	var issues []string
	if len(paths) == 0 {
		return []string{"component_tree dispatch paths are required"}
	}
	uniqueLeafTargets := map[int]bool{}
	for _, path := range paths {
		target, ok := nodes[path.TargetID]
		if !ok {
			issues = append(issues, fmt.Sprintf("component_tree dispatch path target_id %d is unknown", path.TargetID))
			continue
		}
		if target.ChildCount != 0 {
			issues = append(issues, fmt.Sprintf("component_tree dispatch path target_id %d must be a leaf", path.TargetID))
		}
		if strings.TrimSpace(path.Event) == "" {
			issues = append(issues, fmt.Sprintf("component_tree dispatch path target_id %d event is required", path.TargetID))
		}
		for _, id := range path.Path {
			if _, ok := nodes[id]; !ok {
				issues = append(issues, fmt.Sprintf("component_tree dispatch path for target_id %d contains unknown node id %d", path.TargetID, id))
			}
		}
		parentPath, ok := componentTreePathToRoot(path.TargetID, nodes)
		if !ok {
			issues = append(issues, fmt.Sprintf("component_tree dispatch path cannot resolve parent chain for target_id %d", path.TargetID))
			continue
		}
		if !intSlicesEqual(path.Path, parentPath) {
			issues = append(issues, fmt.Sprintf("component_tree dispatch path for target_id %d = %v, want parent chain %v", path.TargetID, path.Path, parentPath))
		}
		if want, ok := expected[path.TargetID]; ok && !intSlicesEqual(path.Path, want) {
			issues = append(issues, fmt.Sprintf("component_tree dispatch path for target_id %d = %v, want %v", path.TargetID, path.Path, want))
		}
		if !rectContainsPoint(target.Bounds, path.X, path.Y) {
			issues = append(issues, fmt.Sprintf("component_tree dispatch path target_id %d coordinates %d,%d are outside target bounds", path.TargetID, path.X, path.Y))
		} else if hitID, ok := componentTreeHitTest(nodes, path.X, path.Y); !ok || hitID != path.TargetID {
			issues = append(issues, fmt.Sprintf("component_tree dispatch path coordinates %d,%d hit node %d, want target_id %d", path.X, path.Y, hitID, path.TargetID))
		}
		uniqueLeafTargets[path.TargetID] = true
	}
	for targetID := range expected {
		found := false
		for _, path := range paths {
			if path.TargetID == targetID {
				found = true
				break
			}
		}
		if !found {
			issues = append(issues, fmt.Sprintf("component_tree dispatch path missing target_id %d", targetID))
		}
	}
	if len(uniqueLeafTargets) < 2 {
		issues = append(issues, "component_tree dispatch paths require at least two different leaf targets")
	}
	return issues
}

func componentTreePathToRoot(id int, nodes map[int]ComponentTreeNodeReport) ([]int, bool) {
	var reversed []int
	seen := map[int]bool{}
	for {
		node, ok := nodes[id]
		if !ok || seen[id] {
			return nil, false
		}
		seen[id] = true
		reversed = append(reversed, id)
		if node.ParentID < 0 {
			break
		}
		id = node.ParentID
	}
	path := make([]int, len(reversed))
	for i := range reversed {
		path[i] = reversed[len(reversed)-1-i]
	}
	return path, true
}

func componentTreeHitTest(nodes map[int]ComponentTreeNodeReport, x int, y int) (int, bool) {
	bestID := -1
	bestDepth := -1
	for id, node := range nodes {
		if !rectContainsPoint(node.Bounds, x, y) {
			continue
		}
		path, ok := componentTreePathToRoot(id, nodes)
		if !ok {
			continue
		}
		if len(path) > bestDepth {
			bestDepth = len(path)
			bestID = id
		}
	}
	return bestID, bestID >= 0
}

func hasComponentTreeTabFocus(events []EventReport, before string, after string) bool {
	for _, event := range events {
		if event.Kind != "key_down" || event.Key != 9 || !event.Handled || !event.Pass {
			continue
		}
		beforeFocus, beforeOK := stateValueWithSuffix(event.BeforeState, ".focused_id")
		afterFocus, afterOK := stateValueWithSuffix(event.AfterState, ".focused_id")
		if beforeOK && afterOK && beforeFocus == before && afterFocus == after {
			return true
		}
	}
	return false
}

func hasComponentTreeTextInsertion(events []EventReport) bool {
	for _, event := range events {
		if event.Kind != "text_input" || !event.Handled || !event.Pass || !strings.HasSuffix(event.TargetComponent, "TextBox") {
			continue
		}
		key := event.TargetComponent + ".buffer"
		if event.BeforeState[key] != event.AfterState[key] && event.AfterState[key] != "" {
			return true
		}
	}
	return false
}

func componentTreeTextMutatedWhileButtonFocused(events []EventReport) bool {
	for _, event := range events {
		if event.Kind != "text_input" {
			continue
		}
		for key, before := range event.BeforeState {
			if !strings.HasSuffix(key, "TextBox.buffer") {
				continue
			}
			after, afterOK := event.AfterState[key]
			if !afterOK || before == after {
				continue
			}
			if !strings.HasPrefix(key, event.TargetComponent+".") {
				return true
			}
		}
	}
	return false
}

func hasComponentTreeButtonAction(events []EventReport) bool {
	seenSubmit := false
	seenReset := false
	for _, event := range events {
		if !event.Handled || !event.Pass {
			continue
		}
		if event.Kind != "key_down" || (event.Key != 32 && event.Key != 13) {
			continue
		}
		if (event.TargetComponent == "SubmitButton" || event.TargetComponent == "SaveButton") &&
			dispatchPathHasSuffix(event.DispatchPath, "ButtonRow", event.TargetComponent) &&
			(stateChangedBySuffix(event.BeforeState, event.AfterState, ".submitted_count") ||
				stateChangedBySuffix(event.BeforeState, event.AfterState, ".submit_count") ||
				stateChangedBySuffix(event.BeforeState, event.AfterState, ".save_count")) {
			seenSubmit = true
		}
		if event.TargetComponent == "ResetButton" &&
			dispatchPathHasSuffix(event.DispatchPath, "ButtonRow", "ResetButton") &&
			stateChangedBySuffix(event.BeforeState, event.AfterState, ".reset_count") &&
			textBoxBufferChanged(event.BeforeState, event.AfterState) {
			seenReset = true
		}
	}
	return seenSubmit && seenReset
}

func hasComponentTreeResizeRelayout(events []EventReport, transitions []StateTransitionReport) bool {
	seenEvent := false
	for _, event := range events {
		if event.Kind != "resize" || !event.Handled || !event.Pass {
			continue
		}
		beforeFocus, beforeOK := stateValueWithSuffix(event.BeforeState, ".focused_id")
		afterFocus, afterOK := stateValueWithSuffix(event.AfterState, ".focused_id")
		if beforeOK && afterOK && beforeFocus == afterFocus && textBoxBoundsChanged(event.BeforeState, event.AfterState) {
			seenEvent = true
		}
	}
	seenTransition := false
	for _, transition := range transitions {
		if transition.Cause == "resize" && strings.HasSuffix(transition.Field, "TextBox.bounds.w") && transition.Before != transition.After {
			seenTransition = true
		}
	}
	return seenEvent && seenTransition
}

func textBoxBufferChanged(before map[string]string, after map[string]string) bool {
	for key, beforeValue := range before {
		if !strings.HasSuffix(key, "TextBox.buffer") {
			continue
		}
		if afterValue, ok := after[key]; ok && beforeValue != afterValue {
			return true
		}
	}
	return false
}

func textBoxBoundsChanged(before map[string]string, after map[string]string) bool {
	for key, beforeValue := range before {
		if !strings.HasSuffix(key, "TextBox.bounds.w") {
			continue
		}
		if afterValue, ok := after[key]; ok && beforeValue != afterValue {
			return true
		}
	}
	return false
}

func toolkitTextInputMutatesOnlyFocused(events []EventReport, target string) bool {
	for _, event := range events {
		if event.Kind != "text_input" || event.TargetComponent != target || !event.Handled || !event.Pass {
			continue
		}
		targetChanged := false
		for key, before := range event.BeforeState {
			if !strings.HasSuffix(key, "TextBox.buffer") {
				continue
			}
			after, ok := event.AfterState[key]
			if !ok || before == after {
				continue
			}
			if !strings.HasPrefix(key, target+".") {
				return false
			}
			targetChanged = true
		}
		if targetChanged {
			return true
		}
	}
	return false
}

func hasToolkitReuseButtonAction(events []EventReport, target string, countSuffix string, clearsTextBoxes bool) bool {
	for _, event := range events {
		if event.TargetComponent != target || event.Kind != "key_down" || !event.Handled || !event.Pass {
			continue
		}
		if event.Key != 32 && event.Key != 13 {
			continue
		}
		if !dispatchPathHasSuffix(event.DispatchPath, "ButtonRow", target) {
			continue
		}
		if !stateChangedBySuffix(event.BeforeState, event.AfterState, countSuffix) {
			continue
		}
		if event.BeforeState["StatusText.status_code"] == event.AfterState["StatusText.status_code"] {
			continue
		}
		if clearsTextBoxes && (event.AfterState["NameTextBox.buffer"] != "" || event.AfterState["EmailTextBox.buffer"] != "") {
			continue
		}
		return true
	}
	return false
}

func stateValueWithSuffix(state map[string]string, suffix string) (string, bool) {
	for key, value := range state {
		if strings.HasSuffix(key, suffix) {
			return value, true
		}
	}
	return "", false
}

func stateChangedBySuffix(before map[string]string, after map[string]string, suffix string) bool {
	for key, beforeValue := range before {
		if !strings.HasSuffix(key, suffix) {
			continue
		}
		if afterValue, ok := after[key]; ok && beforeValue != afterValue {
			return true
		}
	}
	return false
}

func dispatchPathHasSuffix(path []string, suffix ...string) bool {
	if len(path) < len(suffix) {
		return false
	}
	start := len(path) - len(suffix)
	for i, want := range suffix {
		if path[start+i] != want {
			return false
		}
	}
	return true
}

func containsInt(values []int, want int) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func intSlicesEqual(a []int, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func hasEventTargetKind(events []EventReport, target string, kind string) bool {
	for _, event := range events {
		if event.TargetComponent == target && event.Kind == kind && event.Handled && event.Pass {
			return true
		}
	}
	return false
}

func hasKeyEvent(events []EventReport, key int) bool {
	for _, event := range events {
		if event.Kind == "key_down" && event.Key == key && event.Handled && event.Pass {
			return true
		}
	}
	return false
}

func hasResizePreservingFocus(events []EventReport) bool {
	for _, event := range events {
		if event.Kind != "resize" || !event.Handled || !event.Pass {
			continue
		}
		before := event.BeforeState["TextInputApp.focused_component"]
		after := event.AfterState["TextInputApp.focused_component"]
		if before != "" && before == after {
			return true
		}
	}
	return false
}

func hasTransition(transitions []StateTransitionReport, component string, field string) bool {
	for _, transition := range transitions {
		if transition.Component == component && transition.Field == field && transition.Before != transition.After {
			return true
		}
	}
	return false
}

func eventKindContains(events []EventReport, kind string) bool {
	kind = strings.ToLower(strings.TrimSpace(kind))
	for _, event := range events {
		if strings.ToLower(strings.TrimSpace(event.Kind)) == kind {
			return true
		}
	}
	return false
}

func hasRuntimeProcessName(processes []ProcessReport, marker string) bool {
	marker = strings.ToLower(marker)
	for _, process := range processes {
		if process.Kind == "runtime" && strings.Contains(strings.ToLower(process.Name), marker) {
			return true
		}
	}
	return false
}

func artifactKindContains(artifacts []ArtifactReport, marker string) bool {
	marker = strings.ToLower(strings.TrimSpace(marker))
	for _, artifact := range artifacts {
		if strings.Contains(strings.ToLower(strings.TrimSpace(artifact.Kind)), marker) {
			return true
		}
	}
	return false
}

func screenReaderEvidenceTruthy(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.TrimSpace(typed) != ""
	default:
		return false
	}
}

func screenReaderEvidenceString(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed), true
	case bool:
		if typed {
			return "true", true
		}
		return "", false
	default:
		return "", false
	}
}

func hasProcessNameAndPathMarkers(processes []ProcessReport, kind string, nameMarker string, pathMarkers ...string) bool {
	kind = strings.ToLower(strings.TrimSpace(kind))
	nameMarker = strings.ToLower(strings.TrimSpace(nameMarker))
	for i := range pathMarkers {
		pathMarkers[i] = strings.ToLower(strings.TrimSpace(pathMarkers[i]))
	}
	for _, process := range processes {
		if kind != "" && strings.ToLower(strings.TrimSpace(process.Kind)) != kind {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(process.Name))
		if nameMarker != "" && !strings.Contains(name, nameMarker) {
			continue
		}
		path := strings.ToLower(strings.TrimSpace(process.Path))
		missing := false
		for _, marker := range pathMarkers {
			if marker != "" && !strings.Contains(path, marker) {
				missing = true
				break
			}
		}
		if !missing {
			return true
		}
	}
	return false
}

func hasAppProcessWithExpectedExit(processes []ProcessReport, marker string, exitCode int) bool {
	marker = strings.ToLower(marker)
	for _, process := range processes {
		if process.Kind != "app" || !strings.Contains(strings.ToLower(process.Name), marker) {
			continue
		}
		if process.ExitCode != nil && process.ExpectedExitCode != nil && *process.ExitCode == exitCode && *process.ExpectedExitCode == exitCode {
			return true
		}
	}
	return false
}

func caseNameContains(cases []CaseReport, marker string) bool {
	marker = strings.ToLower(marker)
	for _, c := range cases {
		if strings.Contains(strings.ToLower(c.Name), marker) {
			return true
		}
	}
	return false
}

func hasFrameDimensions(frames []FrameReport, width int, height int, stride int) bool {
	for _, frame := range frames {
		if frame.Width == width && frame.Height == height && frame.Stride == stride && frame.Presented && strings.TrimSpace(frame.Checksum) != "" {
			return true
		}
	}
	return false
}

func hasFrameOrderDimensions(frames []FrameReport, order int, width int, height int, stride int) bool {
	for _, frame := range frames {
		if frame.Order == order && frame.Width == width && frame.Height == height && frame.Stride == stride && frame.Presented && strings.TrimSpace(frame.Checksum) != "" {
			return true
		}
	}
	return false
}

func sourceLikeEvidencePath(path string) bool {
	lower := strings.ToLower(strings.TrimSpace(path))
	for _, suffix := range []string{".tetra", ".t4", ".md", ".json", ".html", ".mjs", ".js"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return strings.Contains(lower, ".ui.") || strings.Contains(lower, "tetra.ui.v1")
}

func validateSourceComponentModel(source string, components []ComponentReport) []string {
	sourceModule, ok := sourceModuleFromPath(source)
	if !ok {
		if strings.TrimSpace(source) == "" {
			return nil
		}
		return []string{fmt.Sprintf("source %q must be a Tetra source path with .tetra or .t4 extension", source)}
	}

	var issues []string
	matched := false
	allowToolkitWidgets := isSurfaceToolkitFormSource(source) || isSurfaceToolkitSettingsSource(source) || isSurfaceReleaseFormSource(source) || isSurfaceAccessibilitySettingsSource(source) || isSurfaceReleaseAccessibilitySource(source)
	for _, component := range components {
		componentType := strings.TrimSpace(component.Type)
		if componentType == "" {
			continue
		}
		if strings.HasPrefix(componentType, sourceModule+".") {
			matched = true
			continue
		}
		if allowToolkitWidgets && strings.HasPrefix(componentType, "lib.core.widgets.") {
			continue
		}
		issues = append(issues, fmt.Sprintf("component type %q does not match source module %q", componentType, sourceModule))
	}
	if len(components) > 0 && !matched {
		issues = append(issues, fmt.Sprintf("source module %q is not represented by component type evidence", sourceModule))
	}
	return issues
}

func sourceModuleFromPath(source string) (string, bool) {
	normalized := strings.ReplaceAll(strings.TrimSpace(source), "\\", "/")
	lower := strings.ToLower(normalized)
	switch {
	case strings.HasSuffix(lower, ".tetra"):
		normalized = normalized[:len(normalized)-len(".tetra")]
	case strings.HasSuffix(lower, ".t4"):
		normalized = normalized[:len(normalized)-len(".t4")]
	default:
		return "", false
	}

	parts := strings.Split(normalized, "/")
	start := len(parts) - 1
	for i, part := range parts {
		switch part {
		case "examples", "app", "lib":
			start = i
		}
	}
	moduleParts := make([]string, 0, len(parts)-start)
	for _, part := range parts[start:] {
		part = strings.TrimSpace(part)
		if part == "" || part == "." || part == ".." {
			return "", false
		}
		moduleParts = append(moduleParts, part)
	}
	if len(moduleParts) == 0 {
		return "", false
	}
	return strings.Join(moduleParts, "."), true
}

func validateComponents(components []ComponentReport) (map[string]ComponentReport, []string) {
	var issues []string
	if len(components) == 0 {
		issues = append(issues, "component evidence is required")
	}
	index := map[string]ComponentReport{}
	seenChild := false
	for _, component := range components {
		if strings.TrimSpace(component.ID) == "" {
			issues = append(issues, "component id is required")
			continue
		}
		if _, exists := index[component.ID]; exists {
			issues = append(issues, fmt.Sprintf("duplicate component %s", component.ID))
		}
		index[component.ID] = component
		if strings.TrimSpace(component.Type) == "" {
			issues = append(issues, fmt.Sprintf("component %s type is required", component.ID))
		}
		if !contains(component.Abilities, "measure") {
			issues = append(issues, fmt.Sprintf("component %s missing measure ability evidence", component.ID))
		}
		if !contains(component.Abilities, "layout") {
			issues = append(issues, fmt.Sprintf("component %s missing layout ability evidence", component.ID))
		}
		if !contains(component.Abilities, "draw") {
			issues = append(issues, fmt.Sprintf("component %s missing draw ability evidence", component.ID))
		}
		if !contains(component.Abilities, "event") {
			issues = append(issues, fmt.Sprintf("component %s missing event ability evidence", component.ID))
		}
		if !contains(component.Abilities, "focus") {
			issues = append(issues, fmt.Sprintf("component %s missing focus ability evidence", component.ID))
		}
		if !contains(component.Abilities, "text") {
			issues = append(issues, fmt.Sprintf("component %s missing text ability evidence", component.ID))
		}
		if !contains(component.Abilities, "accessibility") {
			issues = append(issues, fmt.Sprintf("component %s missing accessibility ability evidence", component.ID))
		}
		if component.Bounds.W <= 0 || component.Bounds.H <= 0 {
			issues = append(issues, fmt.Sprintf("component %s layout bounds are required", component.ID))
		}
		if len(component.State) == 0 {
			issues = append(issues, fmt.Sprintf("component %s state evidence is required", component.ID))
		}
	}
	for _, component := range components {
		if strings.TrimSpace(component.ID) == "" {
			continue
		}
		parent := strings.TrimSpace(component.Parent)
		if parent == "" {
			continue
		}
		seenChild = true
		if parent == component.ID {
			issues = append(issues, fmt.Sprintf("component %s cannot parent itself", component.ID))
			continue
		}
		if _, ok := index[parent]; !ok {
			issues = append(issues, fmt.Sprintf("component %s parent %s is not in component evidence", component.ID, parent))
			continue
		}
		if !rectContainsRect(index[parent].Bounds, component.Bounds) {
			issues = append(issues, fmt.Sprintf("component %s layout bounds must be inside parent %s bounds", component.ID, parent))
		}
	}
	if len(components) < 2 || !seenChild {
		issues = append(issues, "component hierarchy evidence is required")
	}
	return index, issues
}

func validateEvents(events []EventReport, components map[string]ComponentReport) []string {
	var issues []string
	if len(events) == 0 {
		issues = append(issues, "event evidence is required")
	}
	lastOrder := 0
	handledStateChange := false
	handledChildDispatch := false
	handledEventBuffer := false
	handledEventBufferSequence := false
	handledTextInput := false
	handledTextPayload := false
	pointerBufferOrder := 0
	for _, event := range events {
		if event.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("event order %d is not strictly greater than previous order %d", event.Order, lastOrder))
		}
		lastOrder = event.Order
		if strings.TrimSpace(event.Kind) == "" {
			issues = append(issues, fmt.Sprintf("event %d kind is required", event.Order))
		}
		if strings.TrimSpace(event.TargetComponent) == "" {
			issues = append(issues, fmt.Sprintf("event %d target_component is required", event.Order))
		} else if _, ok := components[event.TargetComponent]; !ok {
			issues = append(issues, fmt.Sprintf("event %d target_component %s is not in component evidence", event.Order, event.TargetComponent))
		}
		if !validateDispatchPath(event, components, &issues) {
			continue
		}
		if event.Handled && isPointerEvent(event.Kind) {
			target := components[event.TargetComponent]
			if !rectContainsPoint(target.Bounds, event.X, event.Y) {
				issues = append(issues, fmt.Sprintf("event %d pointer dispatch point %d,%d is outside target bounds for %s", event.Order, event.X, event.Y, event.TargetComponent))
			}
		}
		if !event.Pass {
			issues = append(issues, fmt.Sprintf("event %d did not pass", event.Order))
		}
		if len(event.BeforeState) == 0 || len(event.AfterState) == 0 {
			issues = append(issues, fmt.Sprintf("event %d must include before_state and after_state", event.Order))
		}
		if event.Handled && stateChanged(event.BeforeState, event.AfterState) {
			handledStateChange = true
		}
		if event.Handled {
			if component, ok := components[event.TargetComponent]; ok && strings.TrimSpace(component.Parent) != "" {
				handledChildDispatch = true
			}
			if validateEventBuffer(event, &issues) {
				handledEventBuffer = true
				if event.Kind == "mouse_up" && event.BufferSlots[0] == 5 && event.BufferSlots[7] == 0 {
					pointerBufferOrder = event.Order
				}
				if event.Kind == "text_input" && pointerBufferOrder > 0 && event.Order > pointerBufferOrder && event.BufferSlots[0] == 8 && event.BufferSlots[7] > 0 {
					handledEventBufferSequence = true
				}
			}
			if event.Kind == "text_input" && stateChanged(event.BeforeState, event.AfterState) {
				handledTextInput = true
				if validateTextPayloadEvent(event, &issues) {
					handledTextPayload = true
				}
			}
		}
	}
	if !handledStateChange {
		issues = append(issues, "event evidence missing handled state transition")
	}
	if !handledChildDispatch {
		issues = append(issues, "event evidence missing child component dispatch")
	}
	if !handledEventBuffer {
		issues = append(issues, "event evidence missing host event buffer")
	}
	if !handledEventBufferSequence {
		issues = append(issues, "event evidence missing host event buffer pointer/text sequence")
	}
	if !handledTextInput {
		issues = append(issues, "event evidence missing handled text_input scalar dispatch")
	}
	if !handledTextPayload {
		issues = append(issues, "event evidence missing host text payload buffer")
	}
	return issues
}

func validateDispatchPath(event EventReport, components map[string]ComponentReport, issues *[]string) bool {
	if len(event.DispatchPath) == 0 {
		*issues = append(*issues, fmt.Sprintf("event %d dispatch_path is required", event.Order))
		return false
	}
	for _, id := range event.DispatchPath {
		if strings.TrimSpace(id) == "" {
			*issues = append(*issues, fmt.Sprintf("event %d dispatch_path contains an empty component id", event.Order))
			return false
		}
		if _, ok := components[id]; !ok {
			*issues = append(*issues, fmt.Sprintf("event %d dispatch_path component %s is not in component evidence", event.Order, id))
			return false
		}
	}
	if event.DispatchPath[len(event.DispatchPath)-1] != event.TargetComponent {
		*issues = append(*issues, fmt.Sprintf("event %d dispatch_path ends at %s, want target_component %s", event.Order, event.DispatchPath[len(event.DispatchPath)-1], event.TargetComponent))
		return false
	}
	want, ok := componentPathToRoot(event.TargetComponent, components)
	if !ok {
		*issues = append(*issues, fmt.Sprintf("event %d dispatch_path cannot resolve parent chain for %s", event.Order, event.TargetComponent))
		return false
	}
	if !stringSlicesEqual(event.DispatchPath, want) {
		*issues = append(*issues, fmt.Sprintf("event %d dispatch_path = %v, want parent chain %v", event.Order, event.DispatchPath, want))
		return false
	}
	return true
}

func componentPathToRoot(id string, components map[string]ComponentReport) ([]string, bool) {
	var reversed []string
	seen := map[string]bool{}
	for {
		component, ok := components[id]
		if !ok {
			return nil, false
		}
		if seen[id] {
			return nil, false
		}
		seen[id] = true
		reversed = append(reversed, id)
		parent := strings.TrimSpace(component.Parent)
		if parent == "" {
			break
		}
		id = parent
	}
	path := make([]string, len(reversed))
	for i := range reversed {
		path[i] = reversed[len(reversed)-1-i]
	}
	return path, true
}

func stringSlicesEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isPointerEvent(kind string) bool {
	return kind == "mouse_move" || kind == "mouse_down" || kind == "mouse_up"
}

func rectContainsPoint(rect RectReport, x int, y int) bool {
	return x >= rect.X && y >= rect.Y && x < rect.X+rect.W && y < rect.Y+rect.H
}

func rectContainsRect(parent RectReport, child RectReport) bool {
	return child.X >= parent.X && child.Y >= parent.Y && child.X+child.W <= parent.X+parent.W && child.Y+child.H <= parent.Y+parent.H
}

func validateEventBuffer(event EventReport, issues *[]string) bool {
	if len(event.BufferSlots) == 0 {
		return false
	}
	if len(event.BufferSlots) < 9 {
		*issues = append(*issues, fmt.Sprintf("event %d event buffer has %d slots, want at least 9", event.Order, len(event.BufferSlots)))
		return false
	}
	if event.BufferSlots[1] != event.X || event.BufferSlots[2] != event.Y {
		*issues = append(*issues, fmt.Sprintf("event %d event buffer coordinates = %d,%d, want %d,%d", event.Order, event.BufferSlots[1], event.BufferSlots[2], event.X, event.Y))
		return false
	}
	if event.BufferSlots[4] != event.Key || event.BufferSlots[5] != event.Width || event.BufferSlots[6] != event.Height || event.BufferSlots[7] != event.TimestampMS || event.BufferSlots[8] != event.TextLen {
		*issues = append(*issues, fmt.Sprintf("event %d event buffer record = %v, want key/width/height/timestamp/text_len = %d/%d/%d/%d/%d", event.Order, event.BufferSlots, event.Key, event.Width, event.Height, event.TimestampMS, event.TextLen))
		return false
	}
	if event.Kind == "mouse_up" && (event.BufferSlots[0] != 5 || event.BufferSlots[3] != 1) {
		*issues = append(*issues, fmt.Sprintf("event %d mouse_up event buffer slots = %v, want kind 5 and button 1", event.Order, event.BufferSlots))
		return false
	}
	if event.Kind == "text_input" && event.BufferSlots[0] != 8 {
		*issues = append(*issues, fmt.Sprintf("event %d text_input event buffer slots = %v, want kind 8", event.Order, event.BufferSlots))
		return false
	}
	return true
}

func validateTextPayloadEvent(event EventReport, issues *[]string) bool {
	if event.TextLen <= 0 {
		*issues = append(*issues, fmt.Sprintf("event %d text payload length must be positive", event.Order))
		return false
	}
	if strings.TrimSpace(event.TextBytesHex) == "" {
		*issues = append(*issues, fmt.Sprintf("event %d text payload bytes are required", event.Order))
		return false
	}
	payload, err := hex.DecodeString(event.TextBytesHex)
	if err != nil {
		*issues = append(*issues, fmt.Sprintf("event %d text payload bytes are not valid hex", event.Order))
		return false
	}
	if len(payload) != event.TextLen {
		*issues = append(*issues, fmt.Sprintf("event %d text payload length = %d, want %d bytes", event.Order, event.TextLen, len(payload)))
		return false
	}
	return true
}

func validateFrames(frames []FrameReport) []string {
	var issues []string
	if len(frames) == 0 {
		issues = append(issues, "frame evidence is required")
	}
	lastOrder := 0
	for _, frame := range frames {
		if frame.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("frame order %d is not strictly greater than previous order %d", frame.Order, lastOrder))
		}
		lastOrder = frame.Order
		if frame.Width <= 0 || frame.Height <= 0 || frame.Stride <= 0 {
			issues = append(issues, fmt.Sprintf("frame %d dimensions and stride must be positive", frame.Order))
		}
		if strings.TrimSpace(frame.Checksum) == "" {
			issues = append(issues, fmt.Sprintf("frame %d checksum is required", frame.Order))
		} else if !strings.HasPrefix(frame.Checksum, "sha256:") && len(frame.Checksum) != 64 {
			issues = append(issues, fmt.Sprintf("frame %d checksum must be sha256 hex or sha256:<hex>", frame.Order))
		}
		if !frame.Presented {
			issues = append(issues, fmt.Sprintf("frame %d was not presented", frame.Order))
		}
	}
	if len(frames) < 2 {
		issues = append(issues, "frame evidence missing pre/post event frame sequence")
	} else if frames[0].Checksum == frames[1].Checksum {
		issues = append(issues, "frame evidence pre/post event checksums must differ")
	}
	return issues
}

func validateStateTransitions(transitions []StateTransitionReport, components map[string]ComponentReport) []string {
	var issues []string
	if len(transitions) == 0 {
		issues = append(issues, "state transition evidence is required")
	}
	lastOrder := 0
	for _, transition := range transitions {
		if transition.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("state transition order %d is not strictly greater than previous order %d", transition.Order, lastOrder))
		}
		lastOrder = transition.Order
		if strings.TrimSpace(transition.Component) == "" {
			issues = append(issues, "state transition component is required")
		} else if _, ok := components[transition.Component]; !ok {
			issues = append(issues, fmt.Sprintf("state transition component %s is not in component evidence", transition.Component))
		}
		if strings.TrimSpace(transition.Field) == "" {
			issues = append(issues, fmt.Sprintf("state transition %d field is required", transition.Order))
		}
		if transition.Before == transition.After {
			issues = append(issues, fmt.Sprintf("state transition %d must change value", transition.Order))
		}
		if strings.TrimSpace(transition.Cause) == "" {
			issues = append(issues, fmt.Sprintf("state transition %d cause is required", transition.Order))
		}
	}
	return issues
}

func validateCases(cases []CaseReport) []string {
	var issues []string
	if len(cases) == 0 {
		issues = append(issues, "case evidence is required")
	}
	seenPositive := false
	seenNegative := false
	seenNoLegacyUISidecars := false
	seenHostProvidedPointerEvent := false
	seenHostEventBufferPoll := false
	seenPrePostEventFrameSequence := false
	seenComponentHierarchyDispatch := false
	seenComponentTextInputScalarDispatch := false
	seenHostTextPayloadBuffer := false
	seenComponentFocusDispatch := false
	seenComponentAccessibilityMetadata := false
	for _, tc := range cases {
		if strings.TrimSpace(tc.Name) == "" {
			issues = append(issues, "case name is required")
		}
		switch tc.Kind {
		case "positive":
			seenPositive = true
		case "negative":
			seenNegative = true
			if strings.TrimSpace(tc.ExpectedError) == "" {
				issues = append(issues, fmt.Sprintf("negative case %s expected_error is required", tc.Name))
			}
		default:
			issues = append(issues, fmt.Sprintf("case %s kind is %q, want positive or negative", tc.Name, tc.Kind))
		}
		if !tc.Ran {
			issues = append(issues, fmt.Sprintf("case %s did not run", tc.Name))
		}
		if !tc.Pass {
			issues = append(issues, fmt.Sprintf("case %s did not pass", tc.Name))
		}
		if strings.Contains(strings.ToLower(tc.Name), "no legacy ui sidecar artifacts") && tc.Kind == "positive" && tc.Ran && tc.Pass {
			seenNoLegacyUISidecars = true
		}
		if strings.Contains(strings.ToLower(tc.Name), "host-provided pointer event dispatch") && tc.Kind == "positive" && tc.Ran && tc.Pass {
			seenHostProvidedPointerEvent = true
		}
		if strings.Contains(strings.ToLower(tc.Name), "host event buffer poll_event") && tc.Kind == "positive" && tc.Ran && tc.Pass {
			seenHostEventBufferPoll = true
		}
		if strings.Contains(strings.ToLower(tc.Name), "pre/post event frame sequence") && tc.Kind == "positive" && tc.Ran && tc.Pass {
			seenPrePostEventFrameSequence = true
		}
		if strings.Contains(strings.ToLower(tc.Name), "component hierarchy dispatch") && tc.Kind == "positive" && tc.Ran && tc.Pass {
			seenComponentHierarchyDispatch = true
		}
		if strings.Contains(strings.ToLower(tc.Name), "component text input scalar dispatch") && tc.Kind == "positive" && tc.Ran && tc.Pass {
			seenComponentTextInputScalarDispatch = true
		}
		if strings.Contains(strings.ToLower(tc.Name), "host text payload buffer") && tc.Kind == "positive" && tc.Ran && tc.Pass {
			seenHostTextPayloadBuffer = true
		}
		if strings.Contains(strings.ToLower(tc.Name), "component focus dispatch") && tc.Kind == "positive" && tc.Ran && tc.Pass {
			seenComponentFocusDispatch = true
		}
		if strings.Contains(strings.ToLower(tc.Name), "component accessibility metadata") && tc.Kind == "positive" && tc.Ran && tc.Pass {
			seenComponentAccessibilityMetadata = true
		}
	}
	if !seenPositive {
		issues = append(issues, "case evidence missing positive case")
	}
	if !seenNegative {
		issues = append(issues, "case evidence missing negative rejection case")
	}
	if !seenNoLegacyUISidecars {
		issues = append(issues, "case evidence missing no legacy UI sidecar artifacts case")
	}
	if !seenHostProvidedPointerEvent {
		issues = append(issues, "case evidence missing host-provided pointer event dispatch case")
	}
	if !seenHostEventBufferPoll {
		issues = append(issues, "case evidence missing host event buffer poll_event case")
	}
	if !seenPrePostEventFrameSequence {
		issues = append(issues, "case evidence missing pre/post event frame sequence case")
	}
	if !seenComponentHierarchyDispatch {
		issues = append(issues, "case evidence missing component hierarchy dispatch case")
	}
	if !seenComponentTextInputScalarDispatch {
		issues = append(issues, "case evidence missing component text input scalar dispatch case")
	}
	if !seenHostTextPayloadBuffer {
		issues = append(issues, "case evidence missing host text payload buffer case")
	}
	if !seenComponentFocusDispatch {
		issues = append(issues, "case evidence missing component focus dispatch case")
	}
	if !seenComponentAccessibilityMetadata {
		issues = append(issues, "case evidence missing component accessibility metadata case")
	}
	return issues
}

func stateChanged(before, after map[string]string) bool {
	for key, beforeValue := range before {
		if afterValue, ok := after[key]; ok && afterValue != beforeValue {
			return true
		}
	}
	for key := range after {
		if _, ok := before[key]; !ok {
			return true
		}
	}
	return false
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
