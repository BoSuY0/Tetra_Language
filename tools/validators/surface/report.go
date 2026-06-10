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
	Schema                          string                          `json:"schema"`
	Status                          string                          `json:"status"`
	Target                          string                          `json:"target"`
	Host                            string                          `json:"host"`
	Runtime                         string                          `json:"runtime"`
	SurfaceSchema                   string                          `json:"surface_schema"`
	HostABI                         string                          `json:"host_abi"`
	HostEvidence                    HostEvidenceReport              `json:"host_evidence"`
	Source                          string                          `json:"source"`
	Processes                       []ProcessReport                 `json:"processes"`
	Artifacts                       []ArtifactReport                `json:"artifacts"`
	ArtifactScan                    ArtifactScanReport              `json:"artifact_scan"`
	Components                      []ComponentReport               `json:"components"`
	ComponentTree                   *ComponentTreeReport            `json:"component_tree,omitempty"`
	ComponentTreeAPI                *ComponentTreeAPIReport         `json:"component_tree_api,omitempty"`
	BlockGraph                      *BlockGraphReport               `json:"block_graph,omitempty"`
	PaintLayers                     []PaintLayerReport              `json:"paint_layers,omitempty"`
	PaintCommands                   []PaintCommandReport            `json:"paint_commands,omitempty"`
	VisualFeatures                  []string                        `json:"visual_features,omitempty"`
	PaintQualityLevel               string                          `json:"paint_quality_level,omitempty"`
	PaintCacheBudgetBytes           int                             `json:"paint_cache_budget_bytes,omitempty"`
	PaintUnsupportedBlur            bool                            `json:"paint_unsupported_blur,omitempty"`
	TextMeasurements                []TextMeasurementReport         `json:"text_measurements,omitempty"`
	FontFallbacks                   []FontFallbackReport            `json:"font_fallbacks,omitempty"`
	GlyphCaches                     []GlyphCacheReport              `json:"glyph_caches,omitempty"`
	TextRenderCommands              []TextRenderCommandReport       `json:"text_render_commands,omitempty"`
	TextQualityLevel                string                          `json:"text_quality_level,omitempty"`
	TextCacheBudgetBytes            int                             `json:"text_cache_budget_bytes,omitempty"`
	LayoutConstraints               []BlockLayoutConstraintReport   `json:"layout_constraints,omitempty"`
	LayoutPasses                    []BlockLayoutPassReport         `json:"layout_passes,omitempty"`
	LayoutScrolls                   []BlockLayoutScrollReport       `json:"layout_scrolls,omitempty"`
	LayoutFeatures                  []string                        `json:"layout_features,omitempty"`
	LayoutQualityLevel              string                          `json:"layout_quality_level,omitempty"`
	LayoutUnsupportedCSSFlexbox     bool                            `json:"layout_unsupported_css_flexbox,omitempty"`
	BlockEventRoutes                []BlockEventRouteReport         `json:"block_event_routes,omitempty"`
	BlockFocusTransitions           []BlockFocusTransitionReport    `json:"block_focus_transitions,omitempty"`
	BlockEventKinds                 []string                        `json:"block_event_kinds,omitempty"`
	BlockEventPolicy                string                          `json:"block_event_policy,omitempty"`
	BlockEventQualityLevel          string                          `json:"block_event_quality_level,omitempty"`
	BlockEventUnsupportedDragDrop   bool                            `json:"block_event_unsupported_drag_drop,omitempty"`
	BlockStateSelectors             []BlockStateSelectorReport      `json:"block_state_selectors,omitempty"`
	BlockStateResolutions           []BlockStateResolutionReport    `json:"block_state_resolutions,omitempty"`
	BlockStateResolverOrder         []string                        `json:"block_state_resolver_order,omitempty"`
	BlockStateQualityLevel          string                          `json:"block_state_quality_level,omitempty"`
	BlockStateUnsupportedCSSPseudos bool                            `json:"block_state_unsupported_css_pseudos,omitempty"`
	MotionFrames                    []MotionFrameReport             `json:"motion_frames,omitempty"`
	MotionQualityLevel              string                          `json:"motion_quality_level,omitempty"`
	MotionClock                     string                          `json:"motion_clock,omitempty"`
	MotionFrameBudget               int                             `json:"motion_frame_budget,omitempty"`
	MotionUnsupportedCSSAnimations  bool                            `json:"motion_unsupported_css_animations,omitempty"`
	BlockAssetManifest              *BlockAssetManifestReport       `json:"block_asset_manifest,omitempty"`
	BlockAssetCache                 BlockAssetCacheReport           `json:"block_asset_cache,omitempty"`
	BlockAssetDiagnostics           []BlockAssetDiagnosticReport    `json:"block_asset_diagnostics,omitempty"`
	BlockAssetRenderCommands        []BlockAssetRenderCommandReport `json:"block_asset_render_commands,omitempty"`
	BlockAssetQualityLevel          string                          `json:"block_asset_quality_level,omitempty"`
	BlockAssetNetworkFetchAllowed   bool                            `json:"block_asset_network_fetch_allowed,omitempty"`
	BlockAccessibilityTree          *BlockAccessibilityTreeReport   `json:"block_accessibility_tree,omitempty"`
	BlockSystem                     *BlockSystemReport              `json:"block_system,omitempty"`
	Morph                           *MorphReport                    `json:"morph,omitempty"`
	Toolkit                         *ToolkitReport                  `json:"toolkit,omitempty"`
	AccessibilityTree               *AccessibilityTreeReport        `json:"accessibility_tree,omitempty"`
	Events                          []EventReport                   `json:"events"`
	Frames                          []FrameReport                   `json:"frames"`
	StateTransitions                []StateTransitionReport         `json:"state_transitions"`
	Cases                           []CaseReport                    `json:"cases"`
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
	BlockSystem             string   `json:"block_system"`
	BlockSystemGate         string   `json:"block_system_gate"`
	Morph                   string   `json:"morph"`
	MorphGate               string   `json:"morph_gate"`
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

type BlockGraphReport struct {
	Schema             string                       `json:"schema"`
	APILevel           string                       `json:"api_level"`
	Source             string                       `json:"source"`
	ManualBookkeeping  bool                         `json:"manual_bookkeeping"`
	Builder            BlockGraphBuilderReport      `json:"builder"`
	Invariants         BlockGraphInvariantReport    `json:"invariants"`
	RootID             int                          `json:"root_id"`
	NodeCount          int                          `json:"node_count"`
	Nodes              []BlockGraphNodeReport       `json:"nodes"`
	ChildOrders        []BlockGraphChildOrderReport `json:"child_orders"`
	LayoutOrder        []int                        `json:"layout_order"`
	DrawOrder          []int                        `json:"draw_order"`
	FocusOrder         []int                        `json:"focus_order"`
	AccessibilityOrder []int                        `json:"accessibility_order"`
	HitTests           []BlockGraphPathReport       `json:"hit_tests"`
	DispatchPaths      []BlockGraphPathReport       `json:"dispatch_paths"`
}

type BlockGraphBuilderReport struct {
	RootCreatedBy     string `json:"root_created_by"`
	ChildrenCreatedBy string `json:"children_created_by"`
	NodeCount         int    `json:"node_count"`
	Capacity          int    `json:"capacity"`
	OverflowChecked   bool   `json:"overflow_checked"`
}

type BlockGraphInvariantReport struct {
	TreeValidateRan         bool `json:"tree_validate_ran"`
	TreeValidateStatus      int  `json:"tree_validate_status"`
	DuplicateIDRejected     bool `json:"duplicate_id_rejected"`
	MissingParentRejected   bool `json:"missing_parent_rejected"`
	CycleRejected           bool `json:"cycle_rejected"`
	ParentChildLinksChecked bool `json:"parent_child_links_checked"`
	ChildOrderChecked       bool `json:"child_order_checked"`
	FocusOrderChecked       bool `json:"focus_order_checked"`
	HitTestPathChecked      bool `json:"hit_test_path_checked"`
	AccessibilityChecked    bool `json:"accessibility_order_checked"`
}

type BlockGraphNodeReport struct {
	ID                int        `json:"id"`
	Name              string     `json:"name"`
	ParentID          int        `json:"parent_id"`
	ChildIndex        int        `json:"child_index"`
	FirstChild        int        `json:"first_child"`
	ChildCount        int        `json:"child_count"`
	Focusable         bool       `json:"focusable"`
	AccessibilityRole string     `json:"accessibility_role"`
	Bounds            RectReport `json:"bounds"`
}

type BlockGraphChildOrderReport struct {
	ParentID int   `json:"parent_id"`
	Children []int `json:"children"`
}

type BlockGraphPathReport struct {
	Helper   string `json:"helper"`
	Event    string `json:"event,omitempty"`
	TargetID int    `json:"target_id"`
	X        int    `json:"x,omitempty"`
	Y        int    `json:"y,omitempty"`
	Path     []int  `json:"path"`
}

type PaintLayerReport struct {
	ID      string `json:"id"`
	BlockID int    `json:"block_id"`
	Kind    string `json:"kind"`
	Color   string `json:"color,omitempty"`
	Radius  int    `json:"radius,omitempty"`
	Width   int    `json:"width,omitempty"`
	Blur    int    `json:"blur,omitempty"`
	OffsetX int    `json:"offset_x,omitempty"`
	OffsetY int    `json:"offset_y,omitempty"`
	Opacity int    `json:"opacity,omitempty"`
}

type PaintCommandReport struct {
	Order    int        `json:"order"`
	Command  string     `json:"command"`
	LayerID  string     `json:"layer_id"`
	BlockID  int        `json:"block_id"`
	Rect     RectReport `json:"rect"`
	Radius   int        `json:"radius,omitempty"`
	Quality  string     `json:"quality"`
	Checksum string     `json:"checksum"`
}

type TextMeasurementReport struct {
	ID                string     `json:"id"`
	BlockID           int        `json:"block_id"`
	TextLen           int        `json:"text_len"`
	FontFamily        string     `json:"font_family"`
	FontWeight        int        `json:"font_weight"`
	FontSize          int        `json:"font_size"`
	LineHeight        int        `json:"line_height"`
	MaxWidth          int        `json:"max_width"`
	Measured          SizeReport `json:"measured"`
	LineCount         int        `json:"line_count"`
	Wrap              string     `json:"wrap"`
	Overflow          string     `json:"overflow"`
	Ellipsis          bool       `json:"ellipsis"`
	EllipsizedTextLen int        `json:"ellipsized_text_len"`
	Align             string     `json:"align"`
	Quality           string     `json:"quality"`
	Checksum          string     `json:"checksum"`
}

type FontFallbackReport struct {
	ID              string   `json:"id"`
	RequestedFamily string   `json:"requested_family"`
	ResolvedFamily  string   `json:"resolved_family"`
	Chain           []string `json:"chain"`
	MissingGlyphs   int      `json:"missing_glyphs"`
	Coverage        string   `json:"coverage"`
}

type GlyphCacheReport struct {
	ID          string `json:"id"`
	Strategy    string `json:"strategy"`
	BudgetBytes int    `json:"budget_bytes"`
	UsedBytes   int    `json:"used_bytes"`
	EntryCount  int    `json:"entry_count"`
	Eviction    string `json:"eviction"`
	Bounded     bool   `json:"bounded"`
}

type TextRenderCommandReport struct {
	Order         int        `json:"order"`
	Command       string     `json:"command"`
	MeasurementID string     `json:"measurement_id"`
	BlockID       int        `json:"block_id"`
	Rect          RectReport `json:"rect"`
	Clip          RectReport `json:"clip"`
	Color         string     `json:"color"`
	Opacity       int        `json:"opacity"`
	Quality       string     `json:"quality"`
	Checksum      string     `json:"checksum"`
}

type BlockLayoutConstraintReport struct {
	ID           string     `json:"id"`
	BlockID      int        `json:"block_id"`
	Mode         string     `json:"mode"`
	WidthPolicy  string     `json:"width_policy"`
	HeightPolicy string     `json:"height_policy"`
	Min          SizeReport `json:"min"`
	Max          SizeReport `json:"max"`
	Padding      int        `json:"padding"`
	Margin       int        `json:"margin"`
	Gap          int        `json:"gap"`
	Align        string     `json:"align"`
	Justify      string     `json:"justify"`
	Overflow     string     `json:"overflow"`
	ZIndex       int        `json:"z_index"`
	Clip         bool       `json:"clip"`
}

type BlockLayoutPassReport struct {
	Order    int        `json:"order"`
	ParentID int        `json:"parent_id"`
	BlockID  int        `json:"block_id"`
	Mode     string     `json:"mode"`
	Input    RectReport `json:"input"`
	Resolved RectReport `json:"resolved"`
	Measured SizeReport `json:"measured"`
	Pass     string     `json:"pass"`
	Resize   bool       `json:"resize"`
	Clip     bool       `json:"clip"`
	ZIndex   int        `json:"z_index"`
	Checksum string     `json:"checksum"`
}

type BlockLayoutScrollReport struct {
	BlockID    int        `json:"block_id"`
	Viewport   RectReport `json:"viewport"`
	Content    SizeReport `json:"content"`
	OffsetY    int        `json:"offset_y"`
	MaxOffsetY int        `json:"max_offset_y"`
	Clipped    bool       `json:"clipped"`
	Checksum   string     `json:"checksum"`
}

type BlockEventRouteReport struct {
	Order          int    `json:"order"`
	Kind           string `json:"kind"`
	Policy         string `json:"policy"`
	TargetID       int    `json:"target_id"`
	TargetName     string `json:"target_name"`
	HitTestPath    []int  `json:"hit_test_path,omitempty"`
	DispatchPath   []int  `json:"dispatch_path"`
	CapturePath    []int  `json:"capture_path,omitempty"`
	BubblePath     []int  `json:"bubble_path,omitempty"`
	DirectTargetID int    `json:"direct_target_id"`
	Delivered      bool   `json:"delivered"`
	Rejected       bool   `json:"rejected"`
	RejectReason   string `json:"reject_reason,omitempty"`
	FocusedID      int    `json:"focused_id,omitempty"`
	Editable       bool   `json:"editable,omitempty"`
	Disabled       bool   `json:"disabled,omitempty"`
	TextLen        int    `json:"text_len,omitempty"`
	TextBytesHex   string `json:"text_bytes_hex,omitempty"`
}

type BlockFocusTransitionReport struct {
	Order        int    `json:"order"`
	Helper       string `json:"helper"`
	BeforeID     int    `json:"before_id"`
	AfterID      int    `json:"after_id"`
	Direction    string `json:"direction"`
	GraphDerived bool   `json:"graph_derived"`
	Wrapped      bool   `json:"wrapped"`
}

type BlockStateSelectorReport struct {
	Order    int    `json:"order"`
	Name     string `json:"name"`
	BlockID  int    `json:"block_id"`
	Flags    int    `json:"flags"`
	Hovered  bool   `json:"hovered,omitempty"`
	Pressed  bool   `json:"pressed,omitempty"`
	Focused  bool   `json:"focused,omitempty"`
	Selected bool   `json:"selected,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
	Error    bool   `json:"error,omitempty"`
	Loading  bool   `json:"loading,omitempty"`
}

type BlockStateResolutionReport struct {
	Order        int    `json:"order"`
	BlockID      int    `json:"block_id"`
	Selector     string `json:"selector"`
	ResolverStep string `json:"resolver_step"`
	Property     string `json:"property"`
	Before       string `json:"before"`
	After        string `json:"after"`
	Applied      bool   `json:"applied"`
}

type MotionFrameReport struct {
	Order         int    `json:"order"`
	BlockID       int    `json:"block_id"`
	Trigger       string `json:"trigger"`
	TimestampMS   int    `json:"timestamp_ms"`
	DurationMS    int    `json:"duration_ms"`
	DelayMS       int    `json:"delay_ms"`
	Progress      int    `json:"progress"`
	Easing        string `json:"easing"`
	Opacity       int    `json:"opacity"`
	Color         string `json:"color"`
	TranslateX    int    `json:"translate_x"`
	TranslateY    int    `json:"translate_y"`
	Scale         int    `json:"scale"`
	ReducedMotion bool   `json:"reduced_motion"`
	Scheduled     bool   `json:"scheduled"`
	Settled       bool   `json:"settled"`
	Checksum      string `json:"checksum"`
}

type BlockAssetManifestReport struct {
	Schema        string             `json:"schema"`
	Source        string             `json:"source"`
	Quality       string             `json:"quality"`
	HashAlgorithm string             `json:"hash_algorithm"`
	ManifestHash  string             `json:"manifest_hash"`
	LocalOnly     bool               `json:"local_only"`
	FontCount     int                `json:"font_count"`
	IconCount     int                `json:"icon_count"`
	ImageCount    int                `json:"image_count"`
	EmbeddedCount int                `json:"embedded_count"`
	RemoteCount   int                `json:"remote_count"`
	Assets        []BlockAssetReport `json:"assets"`
}

type BlockAssetReport struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Embedded bool   `json:"embedded"`
	Local    bool   `json:"local"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	Family   string `json:"family,omitempty"`
	CacheKey string `json:"cache_key"`
}

type BlockAssetCacheReport struct {
	ID            string `json:"id"`
	Strategy      string `json:"strategy"`
	BudgetBytes   int    `json:"budget_bytes"`
	UsedBytes     int    `json:"used_bytes"`
	EntryCount    int    `json:"entry_count"`
	MaxEntries    int    `json:"max_entries"`
	RepeatedLoads int    `json:"repeated_loads"`
	Eviction      string `json:"eviction"`
	Bounded       bool   `json:"bounded"`
}

type BlockAssetDiagnosticReport struct {
	Order       int    `json:"order"`
	AssetID     string `json:"asset_id"`
	Kind        string `json:"kind"`
	Code        string `json:"code"`
	Message     string `json:"message"`
	FallbackID  string `json:"fallback_id,omitempty"`
	RejectedURL string `json:"rejected_url,omitempty"`
	Pass        bool   `json:"pass"`
}

type BlockAssetRenderCommandReport struct {
	Order    int        `json:"order"`
	Command  string     `json:"command"`
	AssetID  string     `json:"asset_id"`
	BlockID  int        `json:"block_id"`
	Rect     RectReport `json:"rect"`
	Tint     string     `json:"tint,omitempty"`
	Scale    int        `json:"scale,omitempty"`
	Quality  string     `json:"quality"`
	Checksum string     `json:"checksum"`
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

type BlockAccessibilityTreeReport struct {
	Schema                  string                                 `json:"schema"`
	AccessibilityLevel      string                                 `json:"accessibility_level"`
	Source                  string                                 `json:"source"`
	Module                  string                                 `json:"module"`
	QualityLevel            string                                 `json:"quality_level"`
	BlockGraphSchema        string                                 `json:"block_graph_schema"`
	DerivedFromBlockGraph   bool                                   `json:"derived_from_block_graph"`
	ManualBookkeeping       bool                                   `json:"manual_bookkeeping"`
	PlatformHostIntegration bool                                   `json:"platform_host_integration"`
	DOMARIAIntegration      bool                                   `json:"dom_aria_integration"`
	ScreenReaderEvidence    any                                    `json:"screen_reader_evidence"`
	NoDOMUI                 bool                                   `json:"no_dom_ui"`
	NoUserJS                bool                                   `json:"no_user_js"`
	NoPlatformWidgets       bool                                   `json:"no_platform_widgets"`
	NodeCount               int                                    `json:"node_count"`
	FocusableCount          int                                    `json:"focusable_count"`
	RolesPresent            []string                               `json:"roles_present"`
	Nodes                   []BlockAccessibilityNodeReport         `json:"nodes"`
	Relationships           []AccessibilityRelationshipReport      `json:"relationships"`
	FocusOrder              []int                                  `json:"focus_order"`
	ReadingOrder            []int                                  `json:"reading_order"`
	Actions                 []AccessibilityActionReport            `json:"actions"`
	NegativeGuards          BlockAccessibilityNegativeGuardsReport `json:"negative_guards"`
}

type BlockAccessibilityNodeReport struct {
	ID            int        `json:"id"`
	BlockID       int        `json:"block_id"`
	ParentBlockID int        `json:"parent_block_id"`
	Name          string     `json:"name"`
	Role          string     `json:"role"`
	Description   string     `json:"description,omitempty"`
	Value         string     `json:"value,omitempty"`
	State         string     `json:"state,omitempty"`
	Bounds        RectReport `json:"bounds"`
	Visible       bool       `json:"visible"`
	Enabled       bool       `json:"enabled"`
	Focusable     bool       `json:"focusable"`
	Focused       bool       `json:"focused"`
	Editable      bool       `json:"editable"`
	LabelFor      string     `json:"label_for,omitempty"`
	LabelledBy    string     `json:"labelled_by,omitempty"`
	Actions       []string   `json:"actions,omitempty"`
	FocusIndex    int        `json:"focus_index"`
	ReadingIndex  int        `json:"reading_index"`
}

type BlockAccessibilityNegativeGuardsReport struct {
	FocusableActionNameChecked    bool `json:"focusable_action_name_checked"`
	LabelRelationshipsChecked     bool `json:"label_relationships_checked"`
	ReadingOrderGraphChecked      bool `json:"reading_order_graph_checked"`
	BoundsAlignmentChecked        bool `json:"bounds_alignment_checked"`
	FakeScreenReaderClaimRejected bool `json:"fake_screen_reader_claim_rejected"`
	ScopedPlatformClaimChecked    bool `json:"scoped_platform_claim_checked"`
}

type BlockSystemReport struct {
	Schema         string                          `json:"schema"`
	QualityLevel   string                          `json:"quality_level"`
	Source         string                          `json:"source"`
	Renderer       string                          `json:"renderer"`
	GoldenSet      string                          `json:"golden_set"`
	FrameCount     int                             `json:"frame_count"`
	GoldenHash     string                          `json:"golden_hash"`
	Frames         []BlockSystemFrameReport        `json:"frames"`
	MemoryBudget   *BlockMemoryBudgetReport        `json:"memory_budget,omitempty"`
	NegativeGuards BlockSystemNegativeGuardsReport `json:"negative_guards"`
}

type BlockMemoryBudgetReport struct {
	Schema                   string   `json:"schema"`
	Scope                    string   `json:"scope"`
	BlockCount               int      `json:"block_count"`
	StressBlockCount         int      `json:"stress_block_count"`
	RenderLoopCount          int      `json:"render_loop_count"`
	StateLoopCount           int      `json:"state_loop_count"`
	MotionFrameCount         int      `json:"motion_frame_count"`
	InputEventCount          int      `json:"input_event_count"`
	PaintCommandCount        int      `json:"paint_command_count"`
	TextRenderCommandCount   int      `json:"text_render_command_count"`
	AssetRenderCommandCount  int      `json:"asset_render_command_count"`
	PeakFramebufferBytes     int      `json:"peak_framebuffer_bytes"`
	TotalFramebufferBytes    int      `json:"total_framebuffer_bytes"`
	FramebufferBudgetBytes   int      `json:"framebuffer_budget_bytes"`
	PaintCacheUsedBytes      int      `json:"paint_cache_used_bytes"`
	PaintCacheBudgetBytes    int      `json:"paint_cache_budget_bytes"`
	TextCacheUsedBytes       int      `json:"text_cache_used_bytes"`
	TextCacheBudgetBytes     int      `json:"text_cache_budget_bytes"`
	AssetCacheUsedBytes      int      `json:"asset_cache_used_bytes"`
	AssetCacheBudgetBytes    int      `json:"asset_cache_budget_bytes"`
	TotalCacheUsedBytes      int      `json:"total_cache_used_bytes"`
	TotalCacheBudgetBytes    int      `json:"total_cache_budget_bytes"`
	EstimatedAllocationBytes int      `json:"estimated_allocation_bytes"`
	RSSMeasured              bool     `json:"rss_measured"`
	PeakRSSBytes             int      `json:"peak_rss_bytes"`
	BoundedCaches            bool     `json:"bounded_caches"`
	UnboundedCacheRejected   bool     `json:"unbounded_cache_rejected"`
	StressScene              string   `json:"stress_scene"`
	PerformanceClaim         string   `json:"performance_claim"`
	NonClaims                []string `json:"nonclaims"`
}

type BlockSystemFrameReport struct {
	Order                 int    `json:"order"`
	Label                 string `json:"label"`
	Width                 int    `json:"width"`
	Height                int    `json:"height"`
	Stride                int    `json:"stride"`
	Checksum              string `json:"checksum"`
	RepeatChecksum        string `json:"repeat_checksum"`
	GoldenChecksum        string `json:"golden_checksum"`
	PaintEvidence         bool   `json:"paint_evidence"`
	LayoutEvidence        bool   `json:"layout_evidence"`
	AccessibilityEvidence bool   `json:"accessibility_evidence"`
}

type BlockSystemNegativeGuardsReport struct {
	MissingFrameChecksumRejected         bool `json:"missing_frame_checksum_rejected"`
	NondeterministicChecksumRejected     bool `json:"nondeterministic_checksum_rejected"`
	MissingPaintEvidenceRejected         bool `json:"missing_paint_evidence_rejected"`
	MissingLayoutEvidenceRejected        bool `json:"missing_layout_evidence_rejected"`
	MissingAccessibilityEvidenceRejected bool `json:"missing_accessibility_evidence_rejected"`
}

type MorphReport struct {
	Schema           string                             `json:"schema"`
	QualityLevel     string                             `json:"quality_level"`
	Source           string                             `json:"source"`
	Module           string                             `json:"module"`
	SurfaceScope     string                             `json:"surface_scope"`
	Experimental     bool                               `json:"experimental"`
	ProductionClaim  bool                               `json:"production_claim"`
	GitHead          string                             `json:"git_head"`
	GitDirty         bool                               `json:"git_dirty"`
	CapsuleHash      string                             `json:"capsule_hash"`
	TokenGraphHash   string                             `json:"token_graph_hash"`
	Capsule          MorphCapsuleReport                 `json:"capsule"`
	TokenGraph       *MorphTokenGraphReport             `json:"token_graph,omitempty"`
	Materials        []MorphMaterialReport              `json:"materials,omitempty"`
	LayoutModes      []string                           `json:"layout_modes,omitempty"`
	TypographyRoles  []string                           `json:"typography_roles,omitempty"`
	AssetRefs        []MorphAssetRefReport              `json:"asset_refs,omitempty"`
	Affordances      []MorphAffordanceReport            `json:"affordances,omitempty"`
	StateLenses      []MorphStateLensReport             `json:"state_lenses,omitempty"`
	MotionPresets    []MorphMotionPresetReport          `json:"motion_presets,omitempty"`
	Recipes          []MorphRecipeReport                `json:"recipes,omitempty"`
	RecipeExpansions []MorphRecipeExpansionReport       `json:"recipe_expansions,omitempty"`
	Accessibility    MorphAccessibilityProjectionReport `json:"accessibility"`
	EvidenceContract MorphEvidenceContractReport        `json:"evidence_contract"`
	MemoryBudget     MorphMemoryBudgetReport            `json:"memory_budget"`
	NegativeGuards   MorphNegativeGuardsReport          `json:"negative_guards"`
	NonClaims        []string                           `json:"nonclaims,omitempty"`
}

type MorphCapsuleReport struct {
	Namespace       string   `json:"namespace"`
	Version         string   `json:"version"`
	CapsuleHash     string   `json:"capsule_hash"`
	Imports         []string `json:"imports"`
	ExplicitImports bool     `json:"explicit_imports"`
	NoGlobalCascade bool     `json:"no_global_cascade"`
}

type MorphTokenGraphReport struct {
	Schema                     string             `json:"schema"`
	Namespace                  string             `json:"namespace"`
	Version                    string             `json:"version"`
	Hash                       string             `json:"hash"`
	Categories                 []string           `json:"categories"`
	Tokens                     []MorphTokenReport `json:"tokens"`
	AliasCycleRejected         bool               `json:"alias_cycle_rejected"`
	DuplicateSourceRejected    bool               `json:"duplicate_source_rejected"`
	RawLiteralsInAppCode       bool               `json:"raw_literals_in_app_code"`
	UnresolvedFallbackRejected bool               `json:"unresolved_fallback_rejected"`
	FallbackToRandomDefault    bool               `json:"fallback_to_random_default"`
}

type MorphTokenReport struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Kind     string `json:"kind"`
	Value    string `json:"value"`
	Source   string `json:"source"`
	Hash     string `json:"hash"`
}

type MorphMaterialReport struct {
	Name                    string   `json:"name"`
	PaintStack              []string `json:"paint_stack"`
	Fill                    string   `json:"fill"`
	Border                  string   `json:"border"`
	Radius                  string   `json:"radius"`
	Shadow                  string   `json:"shadow"`
	Overlay                 string   `json:"overlay"`
	UnsupportedBlur         bool     `json:"unsupported_blur"`
	UnsupportedBlurRejected bool     `json:"unsupported_blur_rejected"`
}

type MorphAssetRefReport struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	SHA256     string `json:"sha256"`
	Local      bool   `json:"local"`
	FallbackID string `json:"fallback_id"`
	TintToken  string `json:"tint_token"`
}

type MorphAffordanceReport struct {
	Name                  string `json:"name"`
	Role                  string `json:"role"`
	Focusable             bool   `json:"focusable"`
	Action                string `json:"action"`
	Input                 string `json:"input"`
	ProjectsAccessibility bool   `json:"projects_accessibility"`
}

type MorphStateLensReport struct {
	Selector      string `json:"selector"`
	Property      string `json:"property"`
	Deterministic bool   `json:"deterministic"`
}

type MorphMotionPresetReport struct {
	Name              string   `json:"name"`
	DurationMS        int      `json:"duration_ms"`
	Curve             string   `json:"curve"`
	Properties        []string `json:"properties"`
	ReducedMotion     bool     `json:"reduced_motion"`
	DeterministicTime bool     `json:"deterministic_time"`
}

type MorphRecipeReport struct {
	Name                   string   `json:"name"`
	Output                 string   `json:"output"`
	Slots                  []string `json:"slots"`
	Inputs                 []string `json:"inputs"`
	ExpandsToBlockGraph    bool     `json:"expands_to_block_graph"`
	HiddenAppState         bool     `json:"hidden_app_state"`
	PlatformWidgets        bool     `json:"platform_widgets"`
	CorePrimitivePromotion bool     `json:"core_primitive_promotion"`
}

type MorphRecipeExpansionReport struct {
	Recipe       string   `json:"recipe"`
	BlockIDs     []int    `json:"block_ids"`
	SlotBindings []string `json:"slot_bindings"`
	Variant      string   `json:"variant"`
	Reported     bool     `json:"reported"`
}

type MorphAccessibilityProjectionReport struct {
	Schema                string   `json:"schema"`
	DerivedFromBlockGraph bool     `json:"derived_from_block_graph"`
	SafetyOverridesWin    bool     `json:"safety_overrides_win"`
	SnapshotEvidence      bool     `json:"snapshot_evidence"`
	RequiredFields        []string `json:"required_fields"`
	Roles                 []string `json:"roles"`
}

type MorphEvidenceContractReport struct {
	CapsuleHash       string `json:"capsule_hash"`
	TokenGraphHash    string `json:"token_graph_hash"`
	RecipeExpansions  bool   `json:"recipe_expansions"`
	BlockTree         bool   `json:"block_tree"`
	ResolvedLayout    bool   `json:"resolved_layout"`
	PaintLayers       bool   `json:"paint_layers"`
	TextRuns          bool   `json:"text_runs"`
	MotionFrames      bool   `json:"motion_frames"`
	AssetHashes       bool   `json:"asset_hashes"`
	AccessibilityTree bool   `json:"accessibility_tree"`
	MemoryBudget      bool   `json:"memory_budget"`
	FrameChecksums    bool   `json:"frame_checksums"`
	ArtifactHashes    bool   `json:"artifact_hashes"`
}

type MorphMemoryBudgetReport struct {
	Schema                 string `json:"schema"`
	ExpandedRecipeCount    int    `json:"expanded_recipe_count"`
	BlockCount             int    `json:"block_count"`
	PaintCommandCount      int    `json:"paint_command_count"`
	LayoutPassCount        int    `json:"layout_pass_count"`
	TextRunCount           int    `json:"text_run_count"`
	MotionActiveCount      int    `json:"motion_active_count"`
	GlyphCacheBytes        int    `json:"glyph_cache_bytes"`
	AssetCacheBytes        int    `json:"asset_cache_bytes"`
	LayoutCacheBytes       int    `json:"layout_cache_bytes"`
	FramebufferBytes       int    `json:"framebuffer_bytes"`
	PeakRSSBytes           int    `json:"peak_rss_bytes"`
	AllocCount             int    `json:"alloc_count"`
	FrameCount             int    `json:"frame_count"`
	BoundedCaches          bool   `json:"bounded_caches"`
	UnboundedCacheRejected bool   `json:"unbounded_cache_rejected"`
}

type MorphNegativeGuardsReport struct {
	NoCoreWidgetPrimitives          bool `json:"no_core_widget_primitives"`
	NoDOMUI                         bool `json:"no_dom_ui"`
	NoReact                         bool `json:"no_react"`
	NoElectron                      bool `json:"no_electron"`
	NoUserJS                        bool `json:"no_user_js"`
	NoPlatformWidgets               bool `json:"no_platform_widgets"`
	MissingTokenRejected            bool `json:"missing_token_rejected"`
	AliasCycleRejected              bool `json:"alias_cycle_rejected"`
	DuplicateTokenSourceRejected    bool `json:"duplicate_token_source_rejected"`
	DuplicateRecipeNameRejected     bool `json:"duplicate_recipe_name_rejected"`
	MissingRecipeExpansionRejected  bool `json:"missing_recipe_expansion_rejected"`
	UnresolvedTokenRejected         bool `json:"unresolved_token_rejected"`
	MissingAssetRejected            bool `json:"missing_asset_rejected"`
	UnboundedCacheRejected          bool `json:"unbounded_cache_rejected"`
	FakeMotionRejected              bool `json:"fake_motion_rejected"`
	FakeAccessibilityRejected       bool `json:"fake_accessibility_rejected"`
	UnsupportedTargetRejected       bool `json:"unsupported_target_rejected"`
	DirtyCheckoutProductionRejected bool `json:"dirty_checkout_production_rejected"`
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
	issues = append(issues, validateBlockGraphEvidence(report)...)
	issues = append(issues, validateBlockCorePrimitiveEvidence(report)...)
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
		{field: "block_system", got: report.BlockSystem, want: "block-system"},
		{field: "block_system_gate", got: report.BlockSystemGate, want: "tetra.surface.block-system.gate.v1"},
		{field: "morph", got: report.Morph, want: "morph-capsule"},
		{field: "morph_gate", got: report.MorphGate, want: "tetra.surface.morph.gate.v1"},
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

func validateBlockGraphEvidence(report Report) []string {
	if report.BlockGraph == nil {
		return nil
	}

	graph := report.BlockGraph
	var issues []string
	if graph.Schema != "tetra.surface.block-graph.v1" {
		issues = append(issues, fmt.Sprintf("block_graph schema is %q, want tetra.surface.block-graph.v1", graph.Schema))
	}
	if graph.APILevel != "block-tree-builder-v1" {
		issues = append(issues, fmt.Sprintf("block_graph api_level is %q, want block-tree-builder-v1", graph.APILevel))
	}
	if normalizeEvidencePath(graph.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("block_graph source %q must match report source %q", graph.Source, report.Source))
	}
	if graph.ManualBookkeeping {
		issues = append(issues, "block_graph manual_bookkeeping must be false")
	}
	if graph.Builder.RootCreatedBy != "tree_add_root" {
		issues = append(issues, fmt.Sprintf("block_graph builder root_created_by is %q, want tree_add_root", graph.Builder.RootCreatedBy))
	}
	if graph.Builder.ChildrenCreatedBy != "tree_add_child" {
		issues = append(issues, fmt.Sprintf("block_graph builder children_created_by is %q, want tree_add_child", graph.Builder.ChildrenCreatedBy))
	}
	if graph.Builder.NodeCount != graph.NodeCount {
		issues = append(issues, fmt.Sprintf("block_graph builder node_count = %d, want block_graph node_count %d", graph.Builder.NodeCount, graph.NodeCount))
	}
	if graph.Builder.Capacity < graph.NodeCount {
		issues = append(issues, fmt.Sprintf("block_graph builder capacity = %d, want at least node_count %d", graph.Builder.Capacity, graph.NodeCount))
	}
	if !graph.Builder.OverflowChecked {
		issues = append(issues, "block_graph builder must prove overflow_checked")
	}
	if !graph.Invariants.TreeValidateRan {
		issues = append(issues, "block_graph invariants require tree_validate_ran")
	}
	if graph.Invariants.TreeValidateStatus != 0 {
		issues = append(issues, fmt.Sprintf("block_graph tree_validate_status = %d, want 0", graph.Invariants.TreeValidateStatus))
	}
	if !graph.Invariants.DuplicateIDRejected {
		issues = append(issues, "block_graph invariants require duplicate_id_rejected")
	}
	if !graph.Invariants.MissingParentRejected {
		issues = append(issues, "block_graph invariants require missing_parent_rejected")
	}
	if !graph.Invariants.CycleRejected {
		issues = append(issues, "block_graph invariants require cycle_rejected")
	}
	if !graph.Invariants.ParentChildLinksChecked || !graph.Invariants.ChildOrderChecked || !graph.Invariants.FocusOrderChecked ||
		!graph.Invariants.HitTestPathChecked || !graph.Invariants.AccessibilityChecked {
		issues = append(issues, "block_graph invariants must check parent/child links, child order, focus order, hit-test path, and accessibility order")
	}
	if graph.NodeCount != len(graph.Nodes) {
		issues = append(issues, fmt.Sprintf("block_graph node_count = %d, want len(nodes) %d", graph.NodeCount, len(graph.Nodes)))
	}
	if graph.NodeCount < 5 {
		issues = append(issues, fmt.Sprintf("block_graph node_count = %d, want at least 5", graph.NodeCount))
	}

	nodes := map[int]BlockGraphNodeReport{}
	childrenByParent := map[int][]BlockGraphNodeReport{}
	for _, node := range graph.Nodes {
		if _, exists := nodes[node.ID]; exists {
			issues = append(issues, fmt.Sprintf("block_graph duplicate node id %d", node.ID))
		}
		nodes[node.ID] = node
		if strings.TrimSpace(node.Name) == "" {
			issues = append(issues, fmt.Sprintf("block_graph node %d name is required", node.ID))
		}
		if node.Bounds.W < 0 || node.Bounds.H < 0 {
			issues = append(issues, fmt.Sprintf("block_graph node %d bounds must be non-negative", node.ID))
		}
		if node.ChildCount < 0 {
			issues = append(issues, fmt.Sprintf("block_graph node %d child_count must be non-negative", node.ID))
		}
		if node.ChildCount == 0 && node.FirstChild != -1 {
			issues = append(issues, fmt.Sprintf("block_graph leaf node %d first_child = %d, want -1", node.ID, node.FirstChild))
		}
		if node.ParentID >= 0 {
			childrenByParent[node.ParentID] = append(childrenByParent[node.ParentID], node)
		}
	}

	root, ok := nodes[graph.RootID]
	if !ok {
		issues = append(issues, fmt.Sprintf("block_graph root_id %d is not in nodes", graph.RootID))
	} else if root.ParentID != -1 {
		issues = append(issues, fmt.Sprintf("block_graph root %d parent_id = %d, want -1", root.ID, root.ParentID))
	}

	for _, node := range graph.Nodes {
		if node.ParentID < 0 {
			continue
		}
		parent, ok := nodes[node.ParentID]
		if !ok {
			issues = append(issues, fmt.Sprintf("block_graph node %d parent_id %d is unknown", node.ID, node.ParentID))
			continue
		}
		if !rectContainsRect(parent.Bounds, node.Bounds) {
			issues = append(issues, fmt.Sprintf("block_graph node %d bounds must be inside parent %d bounds", node.ID, parent.ID))
		}
		if _, ok := blockGraphPathToRoot(node.ID, nodes); !ok {
			issues = append(issues, fmt.Sprintf("block_graph node %d has missing parent or cycle in root path", node.ID))
		}
	}

	for parentID, children := range childrenByParent {
		parent := nodes[parentID]
		sort.Slice(children, func(i, j int) bool {
			return children[i].ChildIndex < children[j].ChildIndex
		})
		if parent.ChildCount != len(children) {
			issues = append(issues, fmt.Sprintf("block_graph node %d child_count = %d, want %d", parentID, parent.ChildCount, len(children)))
		}
		expectedChildren := make([]int, 0, len(children))
		seenIndex := map[int]int{}
		for _, child := range children {
			if child.ChildIndex < 0 || child.ChildIndex >= len(children) {
				issues = append(issues, fmt.Sprintf("block_graph child node %d child_index = %d, want 0..%d", child.ID, child.ChildIndex, len(children)-1))
			}
			if prev, exists := seenIndex[child.ChildIndex]; exists {
				issues = append(issues, fmt.Sprintf("block_graph sibling child_index %d is used by nodes %d and %d", child.ChildIndex, prev, child.ID))
			}
			seenIndex[child.ChildIndex] = child.ID
			expectedChildren = append(expectedChildren, child.ID)
		}
		if len(expectedChildren) > 0 && parent.FirstChild != expectedChildren[0] {
			issues = append(issues, fmt.Sprintf("block_graph node %d first_child = %d, want %d", parentID, parent.FirstChild, expectedChildren[0]))
		}
		if !hasBlockGraphChildOrder(graph.ChildOrders, parentID, expectedChildren) {
			issues = append(issues, fmt.Sprintf("block_graph child_orders require parent %d children %v", parentID, expectedChildren))
		}
	}

	if !blockGraphOrderCoversNodes(graph.LayoutOrder, nodes) {
		issues = append(issues, "block_graph layout_order must include every node exactly once")
	}
	if !blockGraphOrderCoversNodes(graph.DrawOrder, nodes) {
		issues = append(issues, "block_graph draw_order must include every node exactly once")
	}

	expectedFocus := blockGraphFocusOrder(graph.Nodes)
	if !intSlicesEqual(graph.FocusOrder, expectedFocus) {
		issues = append(issues, fmt.Sprintf("block_graph focus_order = %v, want focusable node order %v", graph.FocusOrder, expectedFocus))
	}
	expectedAccessibility := blockGraphAccessibilityOrder(graph.Nodes)
	if !intSlicesEqual(graph.AccessibilityOrder, expectedAccessibility) {
		issues = append(issues, fmt.Sprintf("block_graph accessibility_order = %v, want accessible node order %v", graph.AccessibilityOrder, expectedAccessibility))
	}
	issues = append(issues, validateBlockGraphPaths("hit_tests", "tree_hit_test_path", graph.HitTests, nodes)...)
	issues = append(issues, validateBlockGraphPaths("dispatch_paths", "tree_build_dispatch_path", graph.DispatchPaths, nodes)...)

	for _, required := range []string{
		"block graph duplicate id rejected",
		"block graph missing parent rejected",
		"block graph cycle rejected",
		"block graph child order",
		"block graph focus order",
		"block graph hit-test path",
		"block graph accessibility order",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("block_graph report requires %s evidence", required))
		}
	}
	return issues
}

func hasBlockGraphChildOrder(orders []BlockGraphChildOrderReport, parentID int, expected []int) bool {
	for _, order := range orders {
		if order.ParentID == parentID && intSlicesEqual(order.Children, expected) {
			return true
		}
	}
	return false
}

func blockGraphOrderCoversNodes(order []int, nodes map[int]BlockGraphNodeReport) bool {
	if len(order) != len(nodes) {
		return false
	}
	seen := map[int]bool{}
	for _, id := range order {
		if _, ok := nodes[id]; !ok || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}

func blockGraphFocusOrder(nodes []BlockGraphNodeReport) []int {
	var order []int
	for _, node := range nodes {
		if node.Focusable {
			order = append(order, node.ID)
		}
	}
	return order
}

func blockGraphAccessibilityOrder(nodes []BlockGraphNodeReport) []int {
	var order []int
	for _, node := range nodes {
		role := strings.TrimSpace(strings.ToLower(node.AccessibilityRole))
		if role != "" && role != "none" {
			order = append(order, node.ID)
		}
	}
	return order
}

func blockGraphPathToRoot(id int, nodes map[int]BlockGraphNodeReport) ([]int, bool) {
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

func validateBlockGraphPaths(field string, helper string, paths []BlockGraphPathReport, nodes map[int]BlockGraphNodeReport) []string {
	if len(paths) == 0 {
		return []string{fmt.Sprintf("block_graph %s evidence is required", field)}
	}
	var issues []string
	for _, path := range paths {
		if path.Helper != helper {
			issues = append(issues, fmt.Sprintf("block_graph %s helper is %q, want %s", field, path.Helper, helper))
		}
		wantPath, ok := blockGraphPathToRoot(path.TargetID, nodes)
		if !ok {
			issues = append(issues, fmt.Sprintf("block_graph %s target_id %d is not reachable from root", field, path.TargetID))
			continue
		}
		if !intSlicesEqual(path.Path, wantPath) {
			issues = append(issues, fmt.Sprintf("block_graph %s target_id %d path = %v, want %v", field, path.TargetID, path.Path, wantPath))
		}
	}
	return issues
}

func validateBlockAccessibilityEvidence(report Report) []string {
	if report.BlockAccessibilityTree == nil {
		return nil
	}

	tree := report.BlockAccessibilityTree
	var issues []string
	if report.BlockGraph == nil {
		return []string{"block_accessibility_tree requires block_graph evidence"}
	}
	graph := report.BlockGraph
	if !isSurfaceBlockAccessibilitySource(report.Source) {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree source path must match a Block accessibility/system example, got %q", report.Source))
	}
	if tree.Schema != "tetra.surface.block-accessibility-tree.v1" {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree schema is %q, want tetra.surface.block-accessibility-tree.v1", tree.Schema))
	}
	if tree.AccessibilityLevel != "block-metadata-tree-v1" {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree accessibility_level is %q, want block-metadata-tree-v1", tree.AccessibilityLevel))
	}
	if normalizeEvidencePath(tree.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree source %q must match report source %q", tree.Source, report.Source))
	}
	if tree.Module != "lib.core.block" {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree module is %q, want lib.core.block", tree.Module))
	}
	if tree.QualityLevel != "block-derived-accessibility-metadata-v1" {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree quality_level is %q, want block-derived-accessibility-metadata-v1", tree.QualityLevel))
	}
	if tree.BlockGraphSchema != "tetra.surface.block-graph.v1" {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree block_graph_schema is %q, want tetra.surface.block-graph.v1", tree.BlockGraphSchema))
	}
	if normalizeEvidencePath(graph.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree block_graph source %q must match report source %q", graph.Source, report.Source))
	}
	if !tree.DerivedFromBlockGraph {
		issues = append(issues, "block_accessibility_tree must declare derived_from_block_graph=true")
	}
	if tree.ManualBookkeeping {
		issues = append(issues, "block_accessibility_tree manual_bookkeeping must be false")
	}
	if tree.PlatformHostIntegration {
		issues = append(issues, "block_accessibility_tree platform_host_integration must be false for metadata-only Block evidence")
	}
	if tree.DOMARIAIntegration {
		issues = append(issues, "block_accessibility_tree dom_aria_integration must be false")
	}
	if screenReaderEvidenceTruthy(tree.ScreenReaderEvidence) {
		issues = append(issues, "block_accessibility_tree screen_reader_evidence must be false without platform assistive-tech proof")
	}
	if !tree.NoDOMUI || !tree.NoUserJS || !tree.NoPlatformWidgets {
		issues = append(issues, "block_accessibility_tree must prove no_dom_ui, no_user_js, and no_platform_widgets")
	}
	if tree.NodeCount != len(tree.Nodes) {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree node_count = %d, want len(nodes) %d", tree.NodeCount, len(tree.Nodes)))
	}
	if tree.NodeCount == 0 {
		issues = append(issues, "block_accessibility_tree nodes evidence is required")
	}

	graphNodes := map[int]BlockGraphNodeReport{}
	for _, node := range graph.Nodes {
		graphNodes[node.ID] = node
	}
	expectedFocus := graph.FocusOrder
	expectedReading := graph.AccessibilityOrder
	if !intSlicesEqual(tree.FocusOrder, expectedFocus) {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree focus_order = %v, want block_graph focus_order %v", tree.FocusOrder, expectedFocus))
	}
	if !intSlicesEqual(tree.ReadingOrder, expectedReading) {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree reading_order = %v, want block_graph accessibility_order %v", tree.ReadingOrder, expectedReading))
	}

	nodesByName := map[string]BlockAccessibilityNodeReport{}
	nodesByBlockID := map[int]BlockAccessibilityNodeReport{}
	roleCounts := map[string]int{}
	focusableCount := 0
	focusedCount := 0
	for _, node := range tree.Nodes {
		if node.ID != node.BlockID {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree node id %d must match block_id %d", node.ID, node.BlockID))
		}
		if strings.TrimSpace(node.Name) == "" {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree node %d name is required", node.BlockID))
		} else if _, exists := nodesByName[node.Name]; exists {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree duplicate node name %s", node.Name))
		}
		nodesByName[node.Name] = node
		if _, exists := nodesByBlockID[node.BlockID]; exists {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree duplicate block_id %d", node.BlockID))
		}
		nodesByBlockID[node.BlockID] = node
		graphNode, ok := graphNodes[node.BlockID]
		if !ok {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s block_id %d is not in block_graph", node.Name, node.BlockID))
		} else {
			if node.ParentBlockID != graphNode.ParentID {
				issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s parent_block_id = %d, want block_graph parent_id %d", node.Name, node.ParentBlockID, graphNode.ParentID))
			}
			if !rectsEqual(node.Bounds, graphNode.Bounds) {
				issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s bounds %+v do not match block_graph bounds %+v", node.Name, node.Bounds, graphNode.Bounds))
			}
			if normalizeAccessibilityRole(node.Role) != normalizeAccessibilityRole(graphNode.AccessibilityRole) {
				issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s role is %q, want block_graph role %q", node.Name, node.Role, graphNode.AccessibilityRole))
			}
			if node.Focusable != graphNode.Focusable {
				issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s focusable = %t, want block_graph focusable %t", node.Name, node.Focusable, graphNode.Focusable))
			}
		}
		role := normalizeAccessibilityRole(node.Role)
		if role == "" || role == "none" {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s role is required", node.Name))
		}
		roleCounts[role]++
		if !containsNormalized(tree.RolesPresent, role) {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree roles_present missing %s", role))
		}
		if node.Bounds.W <= 0 || node.Bounds.H <= 0 {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s bounds are required", node.Name))
		}
		if !node.Visible || !node.Enabled {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s must be visible and enabled", node.Name))
		}
		if node.Focusable {
			focusableCount++
			if node.FocusIndex < 0 {
				issues = append(issues, fmt.Sprintf("block_accessibility_tree focusable node %s requires focus_index", node.Name))
			}
		} else if node.FocusIndex >= 0 {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree non-focusable node %s focus_index = %d, want -1", node.Name, node.FocusIndex))
		}
		if node.Focused {
			focusedCount++
		}
		if node.ReadingIndex < 0 {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s requires reading_index", node.Name))
		}
		if (node.Focusable || len(node.Actions) > 0) && strings.TrimSpace(node.Name) == "" {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree actionable focusable block %d requires accessible name", node.BlockID))
		}
		if node.Focusable && len(node.Actions) == 0 {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree focusable node %s requires actions", node.Name))
		}
	}
	if tree.FocusableCount != focusableCount {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree focusable_count = %d, want %d", tree.FocusableCount, focusableCount))
	}
	if focusedCount > 1 {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree focused node count = %d, want at most 1", focusedCount))
	}
	for _, blockID := range expectedReading {
		if _, ok := nodesByBlockID[blockID]; !ok {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree missing block_graph accessibility node %d", blockID))
		}
	}
	for i, blockID := range expectedFocus {
		node, ok := nodesByBlockID[blockID]
		if !ok {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree missing block_graph focus node %d", blockID))
			continue
		}
		if node.FocusIndex != i {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s focus_index = %d, want %d", node.Name, node.FocusIndex, i))
		}
	}
	for i, blockID := range expectedReading {
		node, ok := nodesByBlockID[blockID]
		if !ok {
			continue
		}
		if node.ReadingIndex != i {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s reading_index = %d, want %d", node.Name, node.ReadingIndex, i))
		}
	}
	for role, count := range roleCounts {
		if count > 0 && !containsNormalized(tree.RolesPresent, role) {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree roles_present missing %s", role))
		}
	}
	issues = append(issues, validateBlockAccessibilityRelationships(tree.Relationships, nodesByName)...)
	issues = append(issues, validateBlockAccessibilityActions(tree.Actions, nodesByName)...)
	issues = append(issues, validateBlockAccessibilityNegativeGuards(tree.NegativeGuards)...)
	for _, required := range []string{
		"block accessibility tree derived from block graph",
		"block accessibility focusable actionable name required",
		"block accessibility label relationship mismatch rejected",
		"block accessibility reading order graph mismatch rejected",
		"block accessibility screen-reader claim without platform proof rejected",
		"block accessibility platform claim scoped metadata only",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree report requires %s evidence", required))
		}
	}
	return issues
}

func validateBlockAccessibilityRelationships(relationships []AccessibilityRelationshipReport, nodes map[string]BlockAccessibilityNodeReport) []string {
	if len(relationships) == 0 {
		return []string{"block_accessibility_tree label relationships evidence is required"}
	}
	var issues []string
	for _, relationship := range relationships {
		from, fromOK := nodes[relationship.From]
		to, toOK := nodes[relationship.To]
		if !fromOK {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree relationship from %q is not a node", relationship.From))
			continue
		}
		if !toOK {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree relationship to %q is not a node", relationship.To))
			continue
		}
		switch relationship.Kind {
		case "label_for":
			if from.LabelFor != to.Name || to.LabelledBy != from.Name {
				issues = append(issues, fmt.Sprintf("block_accessibility_tree label relationship mismatch: %s label_for %s must match reciprocal labelled_by", from.Name, to.Name))
			}
		case "labelled_by":
			if from.LabelledBy != to.Name || to.LabelFor != from.Name {
				issues = append(issues, fmt.Sprintf("block_accessibility_tree label relationship mismatch: %s labelled_by %s must match reciprocal label_for", from.Name, to.Name))
			}
		default:
			issues = append(issues, fmt.Sprintf("block_accessibility_tree relationship kind %q is unsupported", relationship.Kind))
		}
	}
	return issues
}

func validateBlockAccessibilityActions(actions []AccessibilityActionReport, nodes map[string]BlockAccessibilityNodeReport) []string {
	if len(actions) == 0 {
		return []string{"block_accessibility_tree actions evidence is required"}
	}
	var issues []string
	for _, action := range actions {
		node, ok := nodes[action.Target]
		if !ok {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree action target %q is not a node", action.Target))
			continue
		}
		if !contains(node.Actions, action.Action) {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree action %s missing from node %s actions", action.Action, action.Target))
		}
		if strings.TrimSpace(action.Semantic) == "" {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree action %s requires semantic", action.Target))
		}
	}
	return issues
}

func validateBlockAccessibilityNegativeGuards(guards BlockAccessibilityNegativeGuardsReport) []string {
	var missing []string
	if !guards.FocusableActionNameChecked {
		missing = append(missing, "focusable_action_name_checked")
	}
	if !guards.LabelRelationshipsChecked {
		missing = append(missing, "label_relationships_checked")
	}
	if !guards.ReadingOrderGraphChecked {
		missing = append(missing, "reading_order_graph_checked")
	}
	if !guards.BoundsAlignmentChecked {
		missing = append(missing, "bounds_alignment_checked")
	}
	if !guards.FakeScreenReaderClaimRejected {
		missing = append(missing, "fake_screen_reader_claim_rejected")
	}
	if !guards.ScopedPlatformClaimChecked {
		missing = append(missing, "scoped_platform_claim_checked")
	}
	if len(missing) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("block_accessibility_tree negative_guards missing %s", strings.Join(missing, ", "))}
}

func validateBlockCorePrimitiveEvidence(report Report) []string {
	if report.BlockSystem == nil {
		return nil
	}

	var issues []string
	check := func(location string, value string) {
		if token, ok := forbiddenBlockCorePrimitiveToken(value); ok {
			issues = append(issues, fmt.Sprintf("block_system fake core primitive %s rejected in %s %q; use Block parameters instead", token, location, value))
		}
	}
	for i, component := range report.Components {
		check(fmt.Sprintf("components[%d].id", i), component.ID)
		check(fmt.Sprintf("components[%d].type", i), component.Type)
	}
	if report.BlockGraph != nil {
		for i, node := range report.BlockGraph.Nodes {
			check(fmt.Sprintf("block_graph.nodes[%d].name", i), node.Name)
		}
	}
	if report.BlockAccessibilityTree != nil {
		for i, node := range report.BlockAccessibilityTree.Nodes {
			check(fmt.Sprintf("block_accessibility_tree.nodes[%d].name", i), node.Name)
		}
	}
	return issues
}

func forbiddenBlockCorePrimitiveToken(value string) (string, bool) {
	for _, field := range blockPrimitiveNameFields(value) {
		for _, token := range []string{"Button", "Card", "TextField", "TextBox", "Sidebar", "Modal"} {
			if strings.EqualFold(field, token) {
				return token, true
			}
		}
	}
	return "", false
}

func blockPrimitiveNameFields(value string) []string {
	replacer := strings.NewReplacer(
		".", " ",
		"/", " ",
		"\\", " ",
		"_", " ",
		"-", " ",
		":", " ",
	)
	return strings.Fields(replacer.Replace(strings.TrimSpace(value)))
}

func validateMorphEvidence(report Report) []string {
	if report.Morph == nil {
		return nil
	}

	morph := report.Morph
	var issues []string
	if morph.Schema != "tetra.surface.morph.v1" {
		issues = append(issues, fmt.Sprintf("morph schema is %q, want tetra.surface.morph.v1", morph.Schema))
	}
	if morph.QualityLevel != "deterministic-headless-morph-capsule-v1" {
		issues = append(issues, fmt.Sprintf("morph quality_level is %q, want deterministic-headless-morph-capsule-v1", morph.QualityLevel))
	}
	if report.Target != "headless" || report.Runtime != "surface-headless" || report.HostEvidence.Level != "deterministic-headless" {
		issues = append(issues, "morph evidence currently requires deterministic headless Surface runtime evidence")
	}
	if strings.TrimSpace(morph.Source) == "" {
		issues = append(issues, "morph source is required")
	}
	if normalizeEvidencePath(morph.Source) != "examples/surface_morph_command_palette.tetra" {
		issues = append(issues, fmt.Sprintf("morph source is %q, want examples/surface_morph_command_palette.tetra", morph.Source))
	}
	if morph.Module != "lib.core.morph" {
		issues = append(issues, fmt.Sprintf("morph module is %q, want lib.core.morph", morph.Module))
	}
	if morph.SurfaceScope != "surface-morph-experimental-linux-web" {
		issues = append(issues, fmt.Sprintf("morph surface_scope is %q, want surface-morph-experimental-linux-web", morph.SurfaceScope))
	}
	if !morph.Experimental {
		issues = append(issues, "morph experimental must be true until clean release-candidate signoff exists")
	}
	if morph.ProductionClaim && morph.GitDirty {
		issues = append(issues, "morph production claim rejected for dirty checkout")
	}
	if morph.ProductionClaim && !isGitHead(morph.GitHead) {
		issues = append(issues, "morph production claim requires git_head evidence")
	}
	if !validSHA256Digest(morph.CapsuleHash) {
		issues = append(issues, "morph capsule_hash must be sha256 evidence")
	}
	if !validSHA256Digest(morph.TokenGraphHash) {
		issues = append(issues, "morph token_graph_hash must be sha256 evidence")
	}
	if morph.Capsule.CapsuleHash != "" && morph.Capsule.CapsuleHash != morph.CapsuleHash {
		issues = append(issues, "morph capsule.capsule_hash must match morph capsule_hash")
	}
	issues = append(issues, validateMorphCapsule(morph.Capsule)...)
	issues = append(issues, validateMorphTokenGraph(morph)...)
	issues = append(issues, validateMorphMaterials(morph.Materials)...)
	issues = append(issues, validateMorphLayoutModes(morph.LayoutModes)...)
	issues = append(issues, validateMorphTypographyRoles(morph.TypographyRoles)...)
	issues = append(issues, validateMorphAssetRefs(morph.AssetRefs, report)...)
	issues = append(issues, validateMorphAffordances(morph.Affordances)...)
	issues = append(issues, validateMorphStateLenses(morph.StateLenses)...)
	issues = append(issues, validateMorphMotionPresets(morph.MotionPresets, report)...)
	issues = append(issues, validateMorphRecipes(morph.Recipes)...)
	issues = append(issues, validateMorphRecipeExpansions(morph.RecipeExpansions, report)...)
	issues = append(issues, validateMorphAccessibilityProjection(morph.Accessibility, report)...)
	issues = append(issues, validateMorphEvidenceContract(morph.EvidenceContract, morph, report)...)
	issues = append(issues, validateMorphMemoryBudget(morph.MemoryBudget, report)...)
	issues = append(issues, validateMorphNegativeGuards(morph.NegativeGuards)...)
	issues = append(issues, validateMorphNonClaims(morph.NonClaims)...)
	if report.BlockSystem == nil {
		issues = append(issues, "morph evidence requires block_system evidence")
	}
	if report.BlockGraph == nil || report.BlockAccessibilityTree == nil {
		issues = append(issues, "morph evidence requires Block graph and accessibility tree evidence")
	}
	if len(report.PaintLayers) == 0 || len(report.PaintCommands) == 0 {
		issues = append(issues, "morph evidence requires resolved paint layer evidence")
	}
	if len(report.LayoutPasses) == 0 || len(report.LayoutConstraints) == 0 {
		issues = append(issues, "morph evidence requires resolved layout evidence")
	}
	if !hasBlockTextEvidence(report) {
		issues = append(issues, "morph evidence requires text run evidence")
	}
	if !hasBlockMotionEvidence(report) {
		issues = append(issues, "morph evidence requires motion frame evidence")
	}
	if !hasBlockAssetEvidence(report) {
		issues = append(issues, "morph evidence requires asset hash/cache evidence")
	}
	return issues
}

func validateMorphCapsule(capsule MorphCapsuleReport) []string {
	var issues []string
	if capsule.Namespace != "tetra.surface.morph.app" {
		issues = append(issues, fmt.Sprintf("morph capsule namespace is %q, want tetra.surface.morph.app", capsule.Namespace))
	}
	if strings.TrimSpace(capsule.Version) == "" {
		issues = append(issues, "morph capsule version is required")
	}
	if !validSHA256Digest(capsule.CapsuleHash) {
		issues = append(issues, "morph capsule_hash must be sha256 evidence")
	}
	if !capsule.ExplicitImports {
		issues = append(issues, "morph capsule requires explicit imports")
	}
	if !capsule.NoGlobalCascade {
		issues = append(issues, "morph capsule must prove no global cascade")
	}
	for _, required := range []string{"lib.core.block", "lib.core.morph"} {
		if !contains(capsule.Imports, required) {
			issues = append(issues, fmt.Sprintf("morph capsule imports must include %s", required))
		}
	}
	return issues
}

func validateMorphTokenGraph(morph *MorphReport) []string {
	var issues []string
	if morph.TokenGraph == nil {
		return []string{"morph token_graph is required"}
	}
	graph := morph.TokenGraph
	if graph.Schema != "tetra.surface.morph.token-graph.v1" {
		issues = append(issues, fmt.Sprintf("morph token_graph schema is %q, want tetra.surface.morph.token-graph.v1", graph.Schema))
	}
	if graph.Namespace != morph.Capsule.Namespace {
		issues = append(issues, "morph token_graph namespace must match capsule namespace")
	}
	if graph.Hash != morph.TokenGraphHash {
		issues = append(issues, "morph token_graph hash must match morph token_graph_hash")
	}
	if !validSHA256Digest(graph.Hash) {
		issues = append(issues, "morph token_graph hash must be sha256 evidence")
	}
	for _, category := range []string{"color", "space", "radius", "border", "elevation", "opacity", "typography", "motion", "z", "assets", "density"} {
		if !containsNormalized(graph.Categories, category) {
			issues = append(issues, fmt.Sprintf("morph token_graph categories require %s", category))
		}
	}
	if len(graph.Tokens) == 0 {
		issues = append(issues, "morph token_graph tokens are required")
	}
	seenIDs := map[string]string{}
	for i, token := range graph.Tokens {
		id := strings.TrimSpace(token.ID)
		if id == "" {
			issues = append(issues, fmt.Sprintf("morph token_graph tokens[%d].id is required", i))
		}
		if strings.TrimSpace(token.Category) == "" || !containsNormalized(graph.Categories, token.Category) {
			issues = append(issues, fmt.Sprintf("morph token_graph token %q category %q is not declared", token.ID, token.Category))
		}
		if strings.TrimSpace(token.Kind) == "" {
			issues = append(issues, fmt.Sprintf("morph token_graph token %q kind is required", token.ID))
		}
		if strings.TrimSpace(token.Value) == "" {
			issues = append(issues, fmt.Sprintf("morph token_graph token %q value is required", token.ID))
		}
		if strings.TrimSpace(token.Source) != "capsule" {
			issues = append(issues, fmt.Sprintf("morph token_graph token %q source is %q, want capsule", token.ID, token.Source))
		}
		if !validSHA256Digest(token.Hash) {
			issues = append(issues, fmt.Sprintf("morph token_graph token %q hash must be sha256 evidence", token.ID))
		}
		if previous, ok := seenIDs[id]; ok {
			issues = append(issues, fmt.Sprintf("morph token_graph duplicate token %q from %s and %s", id, previous, token.Source))
		}
		seenIDs[id] = token.Source
	}
	if graph.RawLiteralsInAppCode {
		issues = append(issues, "morph token_graph rejects raw literals in app code")
	}
	if graph.FallbackToRandomDefault {
		issues = append(issues, "morph token_graph rejects fallback-to-random-default")
	}
	if !graph.AliasCycleRejected || !graph.DuplicateSourceRejected || !graph.UnresolvedFallbackRejected {
		issues = append(issues, "morph token_graph negative guards require alias_cycle, duplicate_source, and unresolved_fallback rejection")
	}
	return issues
}

func validateMorphMaterials(materials []MorphMaterialReport) []string {
	var issues []string
	if len(materials) == 0 {
		return []string{"morph materials are required"}
	}
	seenFeatures := map[string]bool{}
	for _, material := range materials {
		if strings.TrimSpace(material.Name) == "" {
			issues = append(issues, "morph material name is required")
		}
		if material.UnsupportedBlur {
			issues = append(issues, fmt.Sprintf("morph material %q must not claim unsupported blur", material.Name))
		}
		if !material.UnsupportedBlurRejected {
			issues = append(issues, fmt.Sprintf("morph material %q must prove unsupported blur diagnostics", material.Name))
		}
		for _, feature := range material.PaintStack {
			seenFeatures[normalizeStateToken(feature)] = true
		}
		if token, ok := forbiddenBlockCorePrimitiveToken(material.Name); ok {
			issues = append(issues, fmt.Sprintf("morph material fake core primitive %s rejected in %q", token, material.Name))
		}
	}
	for _, feature := range []string{"fill", "border", "radius", "shadow", "overlay"} {
		if !seenFeatures[feature] {
			issues = append(issues, fmt.Sprintf("morph materials require %s paint stack evidence", feature))
		}
	}
	return issues
}

func validateMorphLayoutModes(modes []string) []string {
	var issues []string
	for _, mode := range []string{"row", "column", "stack", "grid", "dock", "absolute", "overlay", "scroll"} {
		if !containsNormalized(modes, mode) {
			issues = append(issues, fmt.Sprintf("morph layout_modes require %s", mode))
		}
	}
	return issues
}

func validateMorphTypographyRoles(roles []string) []string {
	var issues []string
	for _, role := range []string{"title", "body", "label", "code"} {
		if !containsNormalized(roles, role) {
			issues = append(issues, fmt.Sprintf("morph typography_roles require %s", role))
		}
	}
	return issues
}

func validateMorphAssetRefs(refs []MorphAssetRefReport, report Report) []string {
	var issues []string
	if len(refs) == 0 {
		issues = append(issues, "morph asset_refs are required")
	}
	seen := map[string]bool{}
	for _, ref := range refs {
		if strings.TrimSpace(ref.ID) == "" {
			issues = append(issues, "morph asset_ref id is required")
		}
		seen[ref.ID] = true
		if ref.Kind != "icon" && ref.Kind != "font" && ref.Kind != "image" {
			issues = append(issues, fmt.Sprintf("morph asset_ref %q kind is %q, want icon, font, or image", ref.ID, ref.Kind))
		}
		if !validSHA256Digest(ref.SHA256) {
			issues = append(issues, fmt.Sprintf("morph asset_ref %q sha256 must be sha256 evidence", ref.ID))
		}
		if !ref.Local {
			issues = append(issues, fmt.Sprintf("morph asset_ref %q must be local", ref.ID))
		}
		if strings.TrimSpace(ref.FallbackID) == "" {
			issues = append(issues, fmt.Sprintf("morph asset_ref %q fallback_id is required", ref.ID))
		}
		if strings.TrimSpace(ref.TintToken) == "" {
			issues = append(issues, fmt.Sprintf("morph asset_ref %q tint_token is required", ref.ID))
		}
	}
	for _, required := range []string{"project.new", "command.search", "status.warning"} {
		if !seen[required] {
			issues = append(issues, fmt.Sprintf("morph asset_refs require %s", required))
		}
	}
	if report.BlockAssetNetworkFetchAllowed {
		issues = append(issues, "morph asset evidence requires network fetch disabled")
	}
	if !report.BlockAssetCache.Bounded {
		issues = append(issues, "morph asset evidence requires bounded asset cache")
	}
	return issues
}

func validateMorphAffordances(affordances []MorphAffordanceReport) []string {
	var issues []string
	seen := map[string]MorphAffordanceReport{}
	for _, affordance := range affordances {
		seen[normalizeStateToken(affordance.Name)] = affordance
		if strings.TrimSpace(affordance.Role) == "" {
			issues = append(issues, fmt.Sprintf("morph affordance %q role is required", affordance.Name))
		}
		if !affordance.ProjectsAccessibility {
			issues = append(issues, fmt.Sprintf("morph affordance %q must project accessibility", affordance.Name))
		}
	}
	for _, required := range []string{"action", "field.text", "toggle", "navigation", "region", "overlay", "status"} {
		key := normalizeStateToken(required)
		affordance, ok := seen[key]
		if !ok {
			issues = append(issues, fmt.Sprintf("morph affordances require %s", required))
			continue
		}
		if required == "action" && (!affordance.Focusable || strings.TrimSpace(affordance.Action) == "") {
			issues = append(issues, "morph action affordance requires focusable action evidence")
		}
		if required == "field.text" && (!affordance.Focusable || affordance.Input != "editable_text") {
			issues = append(issues, "morph field.text affordance requires editable text evidence")
		}
	}
	return issues
}

func validateMorphStateLenses(lenses []MorphStateLensReport) []string {
	var issues []string
	seen := map[string]bool{}
	for _, lens := range lenses {
		selector := normalizeStateToken(lens.Selector)
		seen[selector] = true
		if strings.TrimSpace(lens.Property) == "" {
			issues = append(issues, fmt.Sprintf("morph state_lens %q property is required", lens.Selector))
		}
		if !lens.Deterministic {
			issues = append(issues, fmt.Sprintf("morph state_lens %q must be deterministic", lens.Selector))
		}
	}
	for _, selector := range []string{"hover", "pressed", "focusvisible", "selected", "disabled", "error", "loading"} {
		if !seen[selector] {
			issues = append(issues, fmt.Sprintf("morph state_lenses require %s", selector))
		}
	}
	return issues
}

func validateMorphMotionPresets(presets []MorphMotionPresetReport, report Report) []string {
	var issues []string
	if len(presets) == 0 {
		return []string{"morph motion_presets are required"}
	}
	hasReduced := false
	for _, preset := range presets {
		if strings.TrimSpace(preset.Name) == "" {
			issues = append(issues, "morph motion_preset name is required")
		}
		if preset.DurationMS <= 0 {
			issues = append(issues, fmt.Sprintf("morph motion_preset %q duration_ms must be positive", preset.Name))
		}
		if strings.TrimSpace(preset.Curve) == "" {
			issues = append(issues, fmt.Sprintf("morph motion_preset %q curve is required", preset.Name))
		}
		for _, property := range []string{"fill", "opacity", "transform"} {
			if !containsNormalized(preset.Properties, property) {
				issues = append(issues, fmt.Sprintf("morph motion_preset %q properties require %s", preset.Name, property))
			}
		}
		if preset.ReducedMotion && preset.DeterministicTime {
			hasReduced = true
		}
	}
	if !hasReduced {
		issues = append(issues, "morph motion_presets require deterministic reduced-motion evidence")
	}
	if report.MotionUnsupportedCSSAnimations {
		issues = append(issues, "morph motion evidence must not claim CSS animation parity")
	}
	return issues
}

func validateMorphRecipes(recipes []MorphRecipeReport) []string {
	var issues []string
	if len(recipes) == 0 {
		return []string{"morph recipes are required"}
	}
	seen := map[string]bool{}
	for _, recipe := range recipes {
		name := strings.TrimSpace(recipe.Name)
		if name == "" {
			issues = append(issues, "morph recipe name is required")
		}
		if seen[name] {
			issues = append(issues, fmt.Sprintf("morph duplicate recipe name %q rejected", name))
		}
		seen[name] = true
		if token, ok := forbiddenBlockCorePrimitiveToken(name); ok {
			issues = append(issues, fmt.Sprintf("morph recipe fake core primitive %s rejected in name %q", token, name))
		}
		if token, ok := forbiddenBlockCorePrimitiveToken(recipe.Output); ok {
			issues = append(issues, fmt.Sprintf("morph recipe fake core primitive %s rejected in output %q", token, recipe.Output))
		}
		if recipe.Output != "Block" {
			issues = append(issues, fmt.Sprintf("morph recipe %q output is %q, want Block", name, recipe.Output))
		}
		if len(recipe.Slots) == 0 {
			issues = append(issues, fmt.Sprintf("morph recipe %q must declare slots", name))
		}
		if len(recipe.Inputs) == 0 {
			issues = append(issues, fmt.Sprintf("morph recipe %q must declare inputs", name))
		}
		if !recipe.ExpandsToBlockGraph {
			issues = append(issues, fmt.Sprintf("morph recipe %q must expand to Block graph", name))
		}
		if recipe.HiddenAppState {
			issues = append(issues, fmt.Sprintf("morph recipe %q must not allocate hidden app state", name))
		}
		if recipe.PlatformWidgets {
			issues = append(issues, fmt.Sprintf("morph recipe %q must not use platform widgets", name))
		}
		if recipe.CorePrimitivePromotion {
			issues = append(issues, fmt.Sprintf("morph recipe %q core primitive promotion rejected", name))
		}
	}
	for _, required := range []string{"control.action@1", "field.text@1", "command.item@1", "region.panel@1"} {
		if !seen[required] {
			issues = append(issues, fmt.Sprintf("morph recipes require %s", required))
		}
	}
	return issues
}

func validateMorphRecipeExpansions(expansions []MorphRecipeExpansionReport, report Report) []string {
	if len(expansions) == 0 {
		return []string{"morph recipe_expansions are required"}
	}
	var issues []string
	blockIDs := map[int]bool{}
	if report.BlockGraph != nil {
		for _, node := range report.BlockGraph.Nodes {
			blockIDs[node.ID] = true
		}
	}
	seenRecipe := map[string]bool{}
	for _, expansion := range expansions {
		if strings.TrimSpace(expansion.Recipe) == "" {
			issues = append(issues, "morph recipe_expansions recipe is required")
		}
		seenRecipe[expansion.Recipe] = true
		if len(expansion.BlockIDs) == 0 {
			issues = append(issues, fmt.Sprintf("morph recipe_expansions %q block_ids are required", expansion.Recipe))
		}
		for _, blockID := range expansion.BlockIDs {
			if blockID <= 0 {
				issues = append(issues, fmt.Sprintf("morph recipe_expansions %q block_id must be positive", expansion.Recipe))
			} else if len(blockIDs) > 0 && !blockIDs[blockID] {
				issues = append(issues, fmt.Sprintf("morph recipe_expansions %q references missing Block ID %d", expansion.Recipe, blockID))
			}
		}
		if len(expansion.SlotBindings) == 0 {
			issues = append(issues, fmt.Sprintf("morph recipe_expansions %q slot_bindings are required", expansion.Recipe))
		}
		if !expansion.Reported {
			issues = append(issues, fmt.Sprintf("morph recipe_expansions %q must be reported", expansion.Recipe))
		}
	}
	for _, required := range []string{"control.action@1", "field.text@1", "command.item@1", "region.panel@1"} {
		if !seenRecipe[required] {
			issues = append(issues, fmt.Sprintf("morph recipe_expansions require %s", required))
		}
	}
	return issues
}

func validateMorphAccessibilityProjection(projection MorphAccessibilityProjectionReport, report Report) []string {
	var issues []string
	if projection.Schema != "tetra.surface.morph.accessibility-projection.v1" {
		issues = append(issues, fmt.Sprintf("morph accessibility schema is %q, want tetra.surface.morph.accessibility-projection.v1", projection.Schema))
	}
	if !projection.DerivedFromBlockGraph {
		issues = append(issues, "morph accessibility must be derived from Block graph")
	}
	if !projection.SafetyOverridesWin {
		issues = append(issues, "morph accessibility safety overrides must win")
	}
	if !projection.SnapshotEvidence {
		issues = append(issues, "morph accessibility snapshot evidence is required")
	}
	for _, field := range []string{"role", "name", "description", "action", "state", "bounds", "focus_order", "reading_order", "labelled_by", "label_for"} {
		if !containsNormalized(projection.RequiredFields, field) {
			issues = append(issues, fmt.Sprintf("morph accessibility required_fields require %s", field))
		}
	}
	for _, role := range []string{"button", "textbox", "checkbox", "navigation", "region", "dialog", "status"} {
		if !containsNormalized(projection.Roles, role) {
			issues = append(issues, fmt.Sprintf("morph accessibility roles require %s", role))
		}
	}
	if report.BlockAccessibilityTree == nil {
		issues = append(issues, "morph accessibility requires block_accessibility_tree")
	}
	return issues
}

func validateMorphEvidenceContract(contract MorphEvidenceContractReport, morph *MorphReport, report Report) []string {
	var issues []string
	if contract.CapsuleHash != morph.CapsuleHash {
		issues = append(issues, "morph evidence_contract capsule_hash must match morph capsule_hash")
	}
	if contract.TokenGraphHash != morph.TokenGraphHash {
		issues = append(issues, "morph evidence_contract token_graph_hash must match morph token_graph_hash")
	}
	required := []struct {
		name string
		ok   bool
	}{
		{"recipe_expansions", contract.RecipeExpansions},
		{"block_tree", contract.BlockTree && report.BlockGraph != nil},
		{"resolved_layout", contract.ResolvedLayout && len(report.LayoutPasses) > 0},
		{"paint_layers", contract.PaintLayers && len(report.PaintLayers) > 0},
		{"text_runs", contract.TextRuns && hasBlockTextEvidence(report)},
		{"motion_frames", contract.MotionFrames && len(report.MotionFrames) > 0},
		{"asset_hashes", contract.AssetHashes && report.BlockAssetManifest != nil},
		{"accessibility_tree", contract.AccessibilityTree && report.BlockAccessibilityTree != nil},
		{"memory_budget", contract.MemoryBudget && report.BlockSystem != nil && report.BlockSystem.MemoryBudget != nil},
		{"frame_checksums", contract.FrameChecksums && len(report.Frames) > 0},
		{"artifact_hashes", contract.ArtifactHashes && len(report.Artifacts) > 0},
	}
	for _, check := range required {
		if !check.ok {
			issues = append(issues, fmt.Sprintf("morph evidence_contract requires %s evidence", check.name))
		}
	}
	return issues
}

func validateMorphMemoryBudget(budget MorphMemoryBudgetReport, report Report) []string {
	var issues []string
	if budget.Schema != "tetra.surface.morph-memory-budget.v1" {
		issues = append(issues, fmt.Sprintf("morph memory_budget schema is %q, want tetra.surface.morph-memory-budget.v1", budget.Schema))
	}
	if budget.ExpandedRecipeCount <= 0 {
		issues = append(issues, "morph memory_budget expanded_recipe_count must be positive")
	}
	if report.BlockSystem != nil && report.BlockSystem.MemoryBudget != nil && budget.BlockCount < report.BlockSystem.MemoryBudget.BlockCount {
		issues = append(issues, fmt.Sprintf("morph memory_budget block_count = %d, want at least Block memory budget block_count %d", budget.BlockCount, report.BlockSystem.MemoryBudget.BlockCount))
	}
	if budget.PaintCommandCount <= 0 || budget.LayoutPassCount <= 0 || budget.TextRunCount <= 0 || budget.FrameCount <= 0 {
		issues = append(issues, "morph memory_budget requires paint_command_count, layout_pass_count, text_run_count, and frame_count")
	}
	if budget.GlyphCacheBytes < 0 || budget.AssetCacheBytes < 0 || budget.LayoutCacheBytes < 0 || budget.FramebufferBytes <= 0 {
		issues = append(issues, "morph memory_budget cache/framebuffer byte fields are invalid")
	}
	if !budget.BoundedCaches || !budget.UnboundedCacheRejected {
		issues = append(issues, "morph memory_budget requires bounded caches and unbounded cache rejection")
	}
	return issues
}

func validateMorphNegativeGuards(guards MorphNegativeGuardsReport) []string {
	missing := []string{}
	checks := []struct {
		name string
		ok   bool
	}{
		{"no_core_widget_primitives", guards.NoCoreWidgetPrimitives},
		{"no_dom_ui", guards.NoDOMUI},
		{"no_react", guards.NoReact},
		{"no_electron", guards.NoElectron},
		{"no_user_js", guards.NoUserJS},
		{"no_platform_widgets", guards.NoPlatformWidgets},
		{"missing_token_rejected", guards.MissingTokenRejected},
		{"alias_cycle_rejected", guards.AliasCycleRejected},
		{"duplicate_token_source_rejected", guards.DuplicateTokenSourceRejected},
		{"duplicate_recipe_name_rejected", guards.DuplicateRecipeNameRejected},
		{"missing_recipe_expansion_rejected", guards.MissingRecipeExpansionRejected},
		{"unresolved_token_rejected", guards.UnresolvedTokenRejected},
		{"missing_asset_rejected", guards.MissingAssetRejected},
		{"unbounded_cache_rejected", guards.UnboundedCacheRejected},
		{"fake_motion_rejected", guards.FakeMotionRejected},
		{"fake_accessibility_rejected", guards.FakeAccessibilityRejected},
		{"unsupported_target_rejected", guards.UnsupportedTargetRejected},
		{"dirty_checkout_production_rejected", guards.DirtyCheckoutProductionRejected},
	}
	for _, check := range checks {
		if !check.ok {
			missing = append(missing, check.name)
		}
	}
	if len(missing) > 0 {
		return []string{fmt.Sprintf("morph negative_guards missing %s", strings.Join(missing, ", "))}
	}
	return nil
}

func validateMorphNonClaims(nonclaims []string) []string {
	var issues []string
	for _, required := range []string{"DOM runtime absent", "React runtime absent", "Electron claim absent", "platform-native widgets absent", "CSS cascade absent"} {
		if !containsTextFold(nonclaims, required) {
			issues = append(issues, fmt.Sprintf("morph nonclaims require %q", required))
		}
	}
	return issues
}

func validateBlockSystemEvidence(report Report) []string {
	if report.BlockSystem == nil {
		return nil
	}

	system := report.BlockSystem
	var issues []string
	if system.Schema != "tetra.surface.block-system.v1" {
		issues = append(issues, fmt.Sprintf("block_system schema is %q, want tetra.surface.block-system.v1", system.Schema))
	}
	expectedQuality := ""
	expectedRenderer := ""
	requiredCases := []string{}
	linuxRealWindowBlockSystem := system.QualityLevel == "linux-x64-real-window-block-system-v1"
	wasmBrowserCanvasBlockSystem := system.QualityLevel == "wasm32-web-browser-canvas-block-system-v1"
	if linuxRealWindowBlockSystem &&
		(report.Target != "linux-x64" || report.Runtime != "surface-linux-x64" || !isLinuxRealWindowHostEvidenceLevel(report.HostEvidence.Level)) {
		issues = append(issues, "linux-x64 real-window block_system requires linux-x64 real-window runtime evidence")
	}
	if wasmBrowserCanvasBlockSystem &&
		(report.Target != "wasm32-web" || report.Runtime != "surface-wasm32-web" || report.HostEvidence.Level != "wasm32-web-browser-canvas-input") {
		issues = append(issues, "wasm32-web browser-canvas block_system requires wasm32-web browser-canvas runtime evidence")
	}
	switch {
	case report.Target == "headless" && report.Runtime == "surface-headless" && !linuxRealWindowBlockSystem && !wasmBrowserCanvasBlockSystem:
		expectedQuality = "deterministic-headless-block-system-v1"
		expectedRenderer = "software-rgba-headless"
		requiredCases = []string{
			"block system headless golden checksums",
			"block system deterministic repeat checksum",
			"block system missing frame checksum rejected",
			"block system nondeterministic checksum rejected",
			"block system missing paint evidence rejected",
			"block system missing layout evidence rejected",
			"block system missing accessibility evidence rejected",
		}
	case report.Target == "linux-x64" && report.Runtime == "surface-linux-x64" && isLinuxRealWindowHostEvidenceLevel(report.HostEvidence.Level):
		expectedQuality = "linux-x64-real-window-block-system-v1"
		expectedRenderer = "wayland-shm-rgba"
		requiredCases = []string{
			"linux-x64 real-window surface",
			"linux-x64 native input event pump",
			"linux-x64 real-window resize event",
			"linux-x64 real-window close event",
			"block system linux-x64 real-window frame presentation",
			"block system linux-x64 native input state transition",
			"block system linux-x64 real-window checksum",
			"block system missing frame checksum rejected",
			"block system nondeterministic checksum rejected",
			"block system missing paint evidence rejected",
			"block system missing layout evidence rejected",
			"block system missing accessibility evidence rejected",
			"block system missing real-window presentation rejected",
			"block system missing native input state transition rejected",
		}
		if !report.HostEvidence.RealWindow || !report.HostEvidence.NativeInput || !report.HostEvidence.Framebuffer {
			issues = append(issues, "linux-x64 real-window block_system requires framebuffer, real_window, and native_input host evidence")
		}
		if !hasFrameOrderDimensions(report.Frames, 5, 400, 240, 1600) {
			issues = append(issues, "linux-x64 real-window block_system requires order-5 400x240 presented frame evidence")
		}
		if !eventKindContains(report.Events, "mouse_up") || !eventKindContains(report.Events, "key_down") ||
			!eventKindContains(report.Events, "resize") || !eventKindContains(report.Events, "close") {
			issues = append(issues, "linux-x64 real-window block_system requires native input, resize, and close event evidence")
		}
		if !hasTransition(report.StateTransitions, "SubmitBlock", "pressed") {
			issues = append(issues, "linux-x64 real-window block_system requires native input state transition evidence")
		}
	case report.Target == "wasm32-web" && report.Runtime == "surface-wasm32-web" && report.HostEvidence.Level == "wasm32-web-browser-canvas-input":
		expectedQuality = "wasm32-web-browser-canvas-block-system-v1"
		expectedRenderer = "browser-canvas-rgba"
		requiredCases = []string{
			"wasm32-web browser canvas surface",
			"wasm32-web browser canvas RGBA readback",
			"wasm32-web browser canvas pointer input",
			"wasm32-web browser canvas keyboard input",
			"wasm32-web browser canvas resize input",
			"wasm32-web browser canvas text input",
			"wasm32-web Surface Host ABI imports",
			"compiler-owned wasm Surface loader",
			"compiler-owned browser canvas Surface host",
			"block system wasm32-web browser-canvas frame readback",
			"block system wasm32-web browser-canvas native input state transition",
			"block system wasm32-web browser-canvas checksum",
			"block system missing frame checksum rejected",
			"block system nondeterministic checksum rejected",
			"block system missing paint evidence rejected",
			"block system missing layout evidence rejected",
			"block system missing accessibility evidence rejected",
			"block system browser-canvas node runtime substitution rejected",
			"block system browser-canvas missing RGBA readback rejected",
			"block system browser-canvas script sidecar artifact rejected",
			"block system browser-canvas html visual sidecar artifact rejected",
		}
		if !report.HostEvidence.Framebuffer || !report.HostEvidence.NativeInput || !report.HostEvidence.BrowserCanvas || !report.HostEvidence.BrowserInput {
			issues = append(issues, "wasm32-web browser-canvas block_system requires framebuffer, native_input, browser_canvas, and browser_input host evidence")
		}
		if report.HostEvidence.RealWindow {
			issues = append(issues, "wasm32-web browser-canvas block_system must not claim OS real_window evidence")
		}
		if !hasFrameOrderDimensions(report.Frames, 5, 400, 240, 1600) {
			issues = append(issues, "wasm32-web browser-canvas block_system requires order-5 400x240 RGBA readback frame evidence")
		}
		if !eventKindContains(report.Events, "mouse_up") || !eventKindContains(report.Events, "key_down") ||
			!eventKindContains(report.Events, "resize") || !eventKindContains(report.Events, "text_input") {
			issues = append(issues, "wasm32-web browser-canvas block_system requires browser input, resize, and text input event evidence")
		}
		if !hasTransition(report.StateTransitions, "SubmitBlock", "pressed") {
			issues = append(issues, "wasm32-web browser-canvas block_system requires browser native input state transition evidence")
		}
		if !hasProcessNameAndPathMarkers(report.Processes, "app", "surface wasm32-web browser canvas component app", "chrom", "scenario=block-system") {
			issues = append(issues, "wasm32-web browser-canvas block_system requires Chromium-compatible browser-canvas app process evidence")
		}
		issues = append(issues, validateBlockSystemBrowserCanvasArtifacts(report.Artifacts)...)
	default:
		issues = append(issues, "block_system requires deterministic headless, linux-x64 real-window, or wasm32-web browser-canvas runtime evidence")
	}
	if expectedQuality != "" && system.QualityLevel != expectedQuality {
		issues = append(issues, fmt.Sprintf("block_system quality_level is %q, want %s", system.QualityLevel, expectedQuality))
	}
	if normalizeEvidencePath(system.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("block_system source %q must match report source %q", system.Source, report.Source))
	}
	if expectedRenderer != "" && system.Renderer != expectedRenderer {
		issues = append(issues, fmt.Sprintf("block_system renderer is %q, want %s", system.Renderer, expectedRenderer))
	}
	if strings.TrimSpace(system.GoldenSet) == "" {
		issues = append(issues, "block_system golden_set is required")
	}
	if !validChecksumLike(system.GoldenHash) {
		issues = append(issues, "block_system golden_hash must be sha256 evidence")
	}
	if system.FrameCount != len(system.Frames) {
		issues = append(issues, fmt.Sprintf("block_system frame_count = %d, want len(frames) %d", system.FrameCount, len(system.Frames)))
	}
	if len(system.Frames) == 0 {
		issues = append(issues, "block_system frame golden evidence is required")
	}
	if system.MemoryBudget == nil {
		issues = append(issues, "block_system memory_budget is required")
	} else {
		issues = append(issues, validateBlockMemoryBudgetEvidence(report, *system.MemoryBudget)...)
	}

	if len(report.PaintLayers) == 0 || len(report.PaintCommands) == 0 || len(report.VisualFeatures) == 0 {
		issues = append(issues, "block_system requires paint evidence")
	}
	if len(report.LayoutConstraints) == 0 || len(report.LayoutPasses) == 0 || len(report.LayoutScrolls) == 0 {
		issues = append(issues, "block_system requires layout evidence")
	}
	if !hasBlockTextEvidence(report) {
		issues = append(issues, "block_system requires text measurement evidence")
	}
	if !hasBlockStateEvidence(report) {
		issues = append(issues, "block_system requires state selector evidence")
	}
	if !hasBlockMotionEvidence(report) {
		issues = append(issues, "block_system requires motion frame evidence")
	}
	if !hasBlockAssetEvidence(report) {
		issues = append(issues, "block_system requires asset manifest/cache evidence")
	}
	if report.BlockGraph == nil || report.BlockAccessibilityTree == nil {
		issues = append(issues, "block_system requires accessibility evidence")
	}

	reportFrames := map[int]FrameReport{}
	for _, frame := range report.Frames {
		reportFrames[frame.Order] = frame
	}
	lastOrder := 0
	for i, frame := range system.Frames {
		if frame.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("block_system frames order %d is not strictly greater than previous order %d", frame.Order, lastOrder))
		}
		lastOrder = frame.Order
		if strings.TrimSpace(frame.Label) == "" {
			issues = append(issues, fmt.Sprintf("block_system frames[%d] label is required", i))
		}
		if frame.Width <= 0 || frame.Height <= 0 || frame.Stride <= 0 {
			issues = append(issues, fmt.Sprintf("block_system frame %d dimensions and stride must be positive", frame.Order))
		}
		if !validChecksumLike(frame.Checksum) {
			issues = append(issues, fmt.Sprintf("block_system frame %d checksum must be sha256 evidence", frame.Order))
		}
		if !validChecksumLike(frame.RepeatChecksum) {
			issues = append(issues, fmt.Sprintf("block_system frame %d repeat_checksum must be sha256 evidence", frame.Order))
		}
		if !validChecksumLike(frame.GoldenChecksum) {
			issues = append(issues, fmt.Sprintf("block_system frame %d golden_checksum must be sha256 evidence", frame.Order))
		}
		if strings.TrimSpace(frame.Checksum) != "" && strings.TrimSpace(frame.RepeatChecksum) != "" && frame.Checksum != frame.RepeatChecksum {
			issues = append(issues, fmt.Sprintf("block_system frame %d nondeterministic repeat checksum %q, want %q", frame.Order, frame.RepeatChecksum, frame.Checksum))
		}
		if strings.TrimSpace(frame.Checksum) != "" && strings.TrimSpace(frame.GoldenChecksum) != "" && frame.Checksum != frame.GoldenChecksum {
			issues = append(issues, fmt.Sprintf("block_system frame %d golden checksum %q, want %q", frame.Order, frame.GoldenChecksum, frame.Checksum))
		}
		reportFrame, ok := reportFrames[frame.Order]
		if !ok {
			issues = append(issues, fmt.Sprintf("block_system frame %d is missing from runtime frame evidence", frame.Order))
		} else {
			if reportFrame.Width != frame.Width || reportFrame.Height != frame.Height || reportFrame.Stride != frame.Stride {
				issues = append(issues, fmt.Sprintf("block_system frame %d dimensions do not match runtime frame evidence", frame.Order))
			}
			if reportFrame.Checksum != frame.Checksum {
				issues = append(issues, fmt.Sprintf("block_system frame %d checksum %q must match runtime frame checksum %q", frame.Order, frame.Checksum, reportFrame.Checksum))
			}
			if !reportFrame.Presented {
				issues = append(issues, fmt.Sprintf("block_system frame %d requires presented runtime frame evidence", frame.Order))
			}
		}
		if !frame.PaintEvidence {
			issues = append(issues, fmt.Sprintf("block_system frame %d missing paint evidence", frame.Order))
		}
		if !frame.LayoutEvidence {
			issues = append(issues, fmt.Sprintf("block_system frame %d missing layout evidence", frame.Order))
		}
		if !frame.AccessibilityEvidence {
			issues = append(issues, fmt.Sprintf("block_system frame %d missing accessibility evidence", frame.Order))
		}
	}
	if len(system.Frames) != len(report.Frames) {
		issues = append(issues, fmt.Sprintf("block_system frames length %d must match runtime frames length %d", len(system.Frames), len(report.Frames)))
	}

	issues = append(issues, validateBlockSystemNegativeGuards(system.NegativeGuards)...)
	requiredCases = append(requiredCases,
		"block system bounded memory budget",
		"block system stress render loop budget",
		"block system performance nonclaim",
	)
	for _, required := range requiredCases {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("block_system report requires %s evidence", required))
		}
	}
	return issues
}

func validateBlockMemoryBudgetEvidence(report Report, budget BlockMemoryBudgetReport) []string {
	var issues []string
	if budget.Schema != "tetra.surface.block-memory-budget.v1" {
		issues = append(issues, fmt.Sprintf("block_system memory_budget schema is %q, want tetra.surface.block-memory-budget.v1", budget.Schema))
	}
	if budget.Scope != "surface-block-system-local-budget-v1" {
		issues = append(issues, fmt.Sprintf("block_system memory_budget scope is %q, want surface-block-system-local-budget-v1", budget.Scope))
	}
	if budget.BlockCount != len(report.Components) {
		issues = append(issues, fmt.Sprintf("block_system memory_budget block_count = %d, want component count %d", budget.BlockCount, len(report.Components)))
	}
	if budget.StressBlockCount < 128 {
		issues = append(issues, fmt.Sprintf("block_system memory_budget stress_block_count = %d, want at least 128", budget.StressBlockCount))
	}
	if budget.RenderLoopCount < 16 {
		issues = append(issues, fmt.Sprintf("block_system memory_budget render_loop_count = %d, want at least 16", budget.RenderLoopCount))
	}
	if budget.StateLoopCount < len(report.StateTransitions) {
		issues = append(issues, fmt.Sprintf("block_system memory_budget state_loop_count = %d, want at least state transition count %d", budget.StateLoopCount, len(report.StateTransitions)))
	}
	if budget.MotionFrameCount != len(report.MotionFrames) {
		issues = append(issues, fmt.Sprintf("block_system memory_budget motion_frame_count = %d, want %d", budget.MotionFrameCount, len(report.MotionFrames)))
	}
	if budget.InputEventCount != len(report.Events) {
		issues = append(issues, fmt.Sprintf("block_system memory_budget input_event_count = %d, want %d", budget.InputEventCount, len(report.Events)))
	}
	if budget.PaintCommandCount != len(report.PaintCommands) {
		issues = append(issues, fmt.Sprintf("block_system memory_budget paint_command_count = %d, want %d", budget.PaintCommandCount, len(report.PaintCommands)))
	}
	if budget.TextRenderCommandCount != len(report.TextRenderCommands) {
		issues = append(issues, fmt.Sprintf("block_system memory_budget text_render_command_count = %d, want %d", budget.TextRenderCommandCount, len(report.TextRenderCommands)))
	}
	if budget.AssetRenderCommandCount != len(report.BlockAssetRenderCommands) {
		issues = append(issues, fmt.Sprintf("block_system memory_budget asset_render_command_count = %d, want %d", budget.AssetRenderCommandCount, len(report.BlockAssetRenderCommands)))
	}
	peakFramebufferBytes, totalFramebufferBytes := blockFramebufferByteTotals(report.Frames)
	if budget.PeakFramebufferBytes != peakFramebufferBytes {
		issues = append(issues, fmt.Sprintf("block_system memory_budget peak_framebuffer_bytes = %d, want %d", budget.PeakFramebufferBytes, peakFramebufferBytes))
	}
	if budget.TotalFramebufferBytes != totalFramebufferBytes {
		issues = append(issues, fmt.Sprintf("block_system memory_budget total_framebuffer_bytes = %d, want %d", budget.TotalFramebufferBytes, totalFramebufferBytes))
	}
	if budget.FramebufferBudgetBytes < peakFramebufferBytes || budget.FramebufferBudgetBytes > 16*1024*1024 {
		issues = append(issues, fmt.Sprintf("block_system memory_budget framebuffer_budget_bytes = %d outside bounded range for peak %d", budget.FramebufferBudgetBytes, peakFramebufferBytes))
	}
	expectedPaintUsed := len(report.PaintCommands) * 2048
	expectedTextUsed := blockGlyphCacheUsedBytes(report.GlyphCaches)
	expectedAssetUsed := report.BlockAssetCache.UsedBytes
	if budget.PaintCacheUsedBytes != expectedPaintUsed {
		issues = append(issues, fmt.Sprintf("block_system memory_budget paint_cache_used_bytes = %d, want %d", budget.PaintCacheUsedBytes, expectedPaintUsed))
	}
	if budget.PaintCacheBudgetBytes != report.PaintCacheBudgetBytes {
		issues = append(issues, fmt.Sprintf("block_system memory_budget paint_cache_budget_bytes = %d, want %d", budget.PaintCacheBudgetBytes, report.PaintCacheBudgetBytes))
	}
	if budget.TextCacheUsedBytes != expectedTextUsed {
		issues = append(issues, fmt.Sprintf("block_system memory_budget text_cache_used_bytes = %d, want %d", budget.TextCacheUsedBytes, expectedTextUsed))
	}
	if budget.TextCacheBudgetBytes != report.TextCacheBudgetBytes {
		issues = append(issues, fmt.Sprintf("block_system memory_budget text_cache_budget_bytes = %d, want %d", budget.TextCacheBudgetBytes, report.TextCacheBudgetBytes))
	}
	if budget.AssetCacheUsedBytes != expectedAssetUsed {
		issues = append(issues, fmt.Sprintf("block_system memory_budget asset_cache_used_bytes = %d, want %d", budget.AssetCacheUsedBytes, expectedAssetUsed))
	}
	if budget.AssetCacheBudgetBytes != report.BlockAssetCache.BudgetBytes {
		issues = append(issues, fmt.Sprintf("block_system memory_budget asset_cache_budget_bytes = %d, want %d", budget.AssetCacheBudgetBytes, report.BlockAssetCache.BudgetBytes))
	}
	expectedTotalCacheUsed := expectedPaintUsed + expectedTextUsed + expectedAssetUsed
	expectedTotalCacheBudget := report.PaintCacheBudgetBytes + report.TextCacheBudgetBytes + report.BlockAssetCache.BudgetBytes
	if budget.TotalCacheUsedBytes != expectedTotalCacheUsed {
		issues = append(issues, fmt.Sprintf("block_system memory_budget total_cache_used_bytes = %d, want %d", budget.TotalCacheUsedBytes, expectedTotalCacheUsed))
	}
	if budget.TotalCacheBudgetBytes != expectedTotalCacheBudget {
		issues = append(issues, fmt.Sprintf("block_system memory_budget total_cache_budget_bytes = %d, want %d", budget.TotalCacheBudgetBytes, expectedTotalCacheBudget))
	}
	if expectedTotalCacheBudget <= 0 || expectedTotalCacheUsed < 0 || expectedTotalCacheUsed > expectedTotalCacheBudget {
		issues = append(issues, "block_system memory_budget cache totals must be bounded and within budget")
	}
	if budget.EstimatedAllocationBytes < totalFramebufferBytes+expectedTotalCacheUsed {
		issues = append(issues, fmt.Sprintf("block_system memory_budget estimated_allocation_bytes = %d, want at least framebuffer+cache %d", budget.EstimatedAllocationBytes, totalFramebufferBytes+expectedTotalCacheUsed))
	}
	if budget.RSSMeasured {
		if budget.PeakRSSBytes <= 0 || budget.PeakRSSBytes > 512*1024*1024 {
			issues = append(issues, fmt.Sprintf("block_system memory_budget peak_rss_bytes = %d outside scoped RSS range", budget.PeakRSSBytes))
		}
	} else if budget.PeakRSSBytes != 0 {
		issues = append(issues, "block_system memory_budget peak_rss_bytes must be 0 when rss_measured is false")
	}
	if !budget.BoundedCaches {
		issues = append(issues, "block_system memory_budget bounded_caches must be true")
	}
	if !budget.UnboundedCacheRejected {
		issues = append(issues, "block_system memory_budget unbounded_cache_rejected must be true")
	}
	if strings.TrimSpace(budget.StressScene) != "deterministic-block-stress-128" {
		issues = append(issues, fmt.Sprintf("block_system memory_budget stress_scene is %q, want deterministic-block-stress-128", budget.StressScene))
	}
	if strings.TrimSpace(budget.PerformanceClaim) != "none" {
		issues = append(issues, fmt.Sprintf("block_system memory_budget performance_claim is %q, want none", budget.PerformanceClaim))
	}
	issues = append(issues, forbiddenBlockPerformanceClaimIssues("block_system memory_budget", budget.PerformanceClaim)...)
	for _, claim := range budget.NonClaims {
		issues = append(issues, forbiddenBlockPerformanceClaimIssues("block_system memory_budget nonclaims", claim)...)
	}
	for _, required := range []string{
		"no Electron comparison benchmark",
		"no broad performance superiority claim",
		"RSS is optional host evidence",
	} {
		if !containsTextFold(budget.NonClaims, required) {
			issues = append(issues, fmt.Sprintf("block_system memory_budget nonclaims missing %q", required))
		}
	}
	return issues
}

func blockFramebufferByteTotals(frames []FrameReport) (int, int) {
	peak := 0
	total := 0
	for _, frame := range frames {
		bytes := frame.Height * frame.Stride
		if bytes > peak {
			peak = bytes
		}
		total += bytes
	}
	return peak, total
}

func blockGlyphCacheUsedBytes(caches []GlyphCacheReport) int {
	total := 0
	for _, cache := range caches {
		total += cache.UsedBytes
	}
	return total
}

func containsTextFold(values []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, value := range values {
		if strings.Contains(strings.ToLower(strings.TrimSpace(value)), want) {
			return true
		}
	}
	return false
}

func forbiddenBlockPerformanceClaimIssues(label string, fields ...string) []string {
	var issues []string
	for _, field := range fields {
		lower := strings.ToLower(field)
		for _, marker := range []string{
			"faster than " + "electron",
			"zero " + "overhead",
			"zero-cost " + "ui",
			"zero cost " + "ui",
			"fastest " + "ui",
		} {
			if strings.Contains(lower, marker) {
				issues = append(issues, fmt.Sprintf("%s contains forbidden performance claim %q", label, marker))
			}
		}
	}
	return issues
}

func validateBlockSystemBrowserCanvasArtifacts(artifacts []ArtifactReport) []string {
	var issues []string
	for _, artifact := range artifacts {
		kind := strings.ToLower(strings.TrimSpace(artifact.Kind))
		path := strings.ToLower(normalizeEvidencePath(artifact.Path))
		if strings.Contains(kind, "user-js") || strings.Contains(path, ".user.js") || strings.HasSuffix(path, "/user.js") {
			issues = append(issues, fmt.Sprintf("wasm32-web browser-canvas block_system must not include user JS artifact %q", artifact.Path))
		}
		if strings.Contains(kind, "dom-ui") || strings.Contains(path, ".dom.") || strings.HasSuffix(path, ".html") {
			issues = append(issues, fmt.Sprintf("wasm32-web browser-canvas block_system must not include DOM UI artifact %q", artifact.Path))
		}
		if strings.HasSuffix(path, ".js") && !strings.HasSuffix(path, ".mjs") {
			issues = append(issues, fmt.Sprintf("wasm32-web browser-canvas block_system must not include user JS artifact %q", artifact.Path))
		}
	}
	return issues
}

func validateBlockSystemNegativeGuards(guards BlockSystemNegativeGuardsReport) []string {
	var missing []string
	if !guards.MissingFrameChecksumRejected {
		missing = append(missing, "missing_frame_checksum_rejected")
	}
	if !guards.NondeterministicChecksumRejected {
		missing = append(missing, "nondeterministic_checksum_rejected")
	}
	if !guards.MissingPaintEvidenceRejected {
		missing = append(missing, "missing_paint_evidence_rejected")
	}
	if !guards.MissingLayoutEvidenceRejected {
		missing = append(missing, "missing_layout_evidence_rejected")
	}
	if !guards.MissingAccessibilityEvidenceRejected {
		missing = append(missing, "missing_accessibility_evidence_rejected")
	}
	if len(missing) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("block_system negative_guards missing %s", strings.Join(missing, ", "))}
}

func normalizeAccessibilityRole(role string) string {
	return strings.ToLower(strings.TrimSpace(role))
}

func containsNormalized(values []string, value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, current := range values {
		if strings.ToLower(strings.TrimSpace(current)) == value {
			return true
		}
	}
	return false
}

func validateBlockPaintEvidence(report Report) []string {
	if !hasBlockPaintEvidence(report) {
		return nil
	}

	var issues []string
	if report.PaintQualityLevel != "deterministic-software-paint-v1" {
		issues = append(issues, fmt.Sprintf("paint_quality_level is %q, want deterministic-software-paint-v1", report.PaintQualityLevel))
	}
	if report.PaintCacheBudgetBytes <= 0 || report.PaintCacheBudgetBytes > 1024*1024 {
		issues = append(issues, fmt.Sprintf("paint_cache_budget_bytes = %d, want 1..1048576", report.PaintCacheBudgetBytes))
	}
	if report.PaintUnsupportedBlur {
		issues = append(issues, "paint unsupported blur claim must be false")
	}
	if len(report.PaintLayers) == 0 {
		issues = append(issues, "paint_layers evidence is required")
	}
	if len(report.PaintCommands) == 0 {
		issues = append(issues, "paint_commands evidence is required")
	}
	if len(report.VisualFeatures) == 0 {
		issues = append(issues, "visual_features evidence is required")
	}

	for _, feature := range []string{"fill", "gradient", "border", "radius", "shadow", "outline"} {
		if !visualFeatureContains(report.VisualFeatures, feature) {
			issues = append(issues, fmt.Sprintf("visual_features require %s", feature))
		}
	}

	layerByKind := map[string]PaintLayerReport{}
	layerIDs := map[string]bool{}
	hasLayerRadius := false
	for _, layer := range report.PaintLayers {
		kind := normalizePaintToken(layer.Kind)
		if strings.TrimSpace(layer.ID) == "" {
			issues = append(issues, "paint_layers id is required")
		} else if layerIDs[layer.ID] {
			issues = append(issues, fmt.Sprintf("paint_layers duplicate id %q", layer.ID))
		}
		layerIDs[layer.ID] = true
		if layer.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("paint_layers %q block_id must be positive", layer.ID))
		}
		if kind == "" {
			issues = append(issues, fmt.Sprintf("paint_layers %q kind is required", layer.ID))
		}
		if layer.Radius > 0 {
			hasLayerRadius = true
		}
		if (kind == "border" || kind == "outline") && layer.Width <= 0 {
			issues = append(issues, fmt.Sprintf("paint_layers %q width must be positive for %s", layer.ID, kind))
		}
		if kind == "shadow" && layer.Blur <= 0 {
			issues = append(issues, fmt.Sprintf("paint_layers %q blur must be positive for shadow approximation", layer.ID))
		}
		if kind == "blur" || kind == "backdrop_blur" {
			issues = append(issues, "paint_layers unsupported blur/backdrop_blur layer is not allowed")
		}
		if _, exists := layerByKind[kind]; !exists {
			layerByKind[kind] = layer
		}
	}
	for _, kind := range []string{"fill", "gradient", "border", "shadow", "outline"} {
		if _, ok := layerByKind[kind]; !ok {
			issues = append(issues, fmt.Sprintf("paint_layers require %s layer", kind))
		}
	}
	if !hasLayerRadius {
		issues = append(issues, "paint_layers require radius evidence")
	}

	expectedCommands := []string{"fill", "gradient", "border", "shadow", "outline"}
	seenCommands := map[string]bool{}
	lastOrder := 0
	for i, command := range report.PaintCommands {
		name := normalizePaintToken(command.Command)
		if command.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("paint_commands order %d is not strictly greater than previous order %d", command.Order, lastOrder))
		}
		lastOrder = command.Order
		if i < len(expectedCommands) && name != expectedCommands[i] {
			issues = append(issues, fmt.Sprintf("paint_commands[%d] command is %q, want deterministic %q", i, command.Command, expectedCommands[i]))
		}
		if command.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("paint_commands[%d] block_id must be positive", i))
		}
		if command.Rect.W <= 0 || command.Rect.H <= 0 {
			issues = append(issues, fmt.Sprintf("paint_commands[%d] rect dimensions must be positive", i))
		}
		if strings.TrimSpace(command.LayerID) == "" {
			issues = append(issues, fmt.Sprintf("paint_commands[%d] layer_id is required", i))
		} else if !layerIDs[command.LayerID] {
			issues = append(issues, fmt.Sprintf("paint_commands[%d] layer_id %q is not in paint_layers", i, command.LayerID))
		}
		if strings.TrimSpace(command.Quality) == "" {
			issues = append(issues, fmt.Sprintf("paint_commands[%d] quality is required", i))
		}
		if !validChecksumLike(command.Checksum) {
			issues = append(issues, fmt.Sprintf("paint_commands[%d] checksum must be sha256 evidence", i))
		}
		if name == "fill" || name == "gradient" || name == "border" || name == "outline" {
			if command.Radius <= 0 {
				issues = append(issues, fmt.Sprintf("paint_commands[%d] %s radius must be positive", i, name))
			}
		}
		seenCommands[name] = true
	}
	for _, command := range expectedCommands {
		if !seenCommands[command] {
			issues = append(issues, fmt.Sprintf("paint_commands require %s command", command))
		}
	}

	if len(report.Frames) < 2 || strings.TrimSpace(report.Frames[0].Checksum) == "" || report.Frames[0].Checksum == report.Frames[1].Checksum {
		issues = append(issues, "paint frame checksum evidence must show visual change")
	}
	for _, required := range []string{
		"block paint fill border radius shadow outline",
		"block paint deterministic command order",
		"block paint frame checksum changed",
		"block paint unsupported blur rejected",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("paint report requires %s evidence", required))
		}
	}
	return issues
}

func hasBlockPaintEvidence(report Report) bool {
	return len(report.PaintLayers) > 0 ||
		len(report.PaintCommands) > 0 ||
		len(report.VisualFeatures) > 0 ||
		strings.TrimSpace(report.PaintQualityLevel) != "" ||
		report.PaintCacheBudgetBytes != 0 ||
		report.PaintUnsupportedBlur
}

func validateBlockTextEvidence(report Report) []string {
	if !hasBlockTextEvidence(report) {
		return nil
	}

	var issues []string
	if report.TextQualityLevel != "deterministic-fallback-text-v1" {
		issues = append(issues, fmt.Sprintf("text_quality_level is %q, want deterministic-fallback-text-v1", report.TextQualityLevel))
	}
	if report.TextCacheBudgetBytes <= 0 || report.TextCacheBudgetBytes > 1024*1024 {
		issues = append(issues, fmt.Sprintf("text_cache_budget_bytes = %d, want 1..1048576", report.TextCacheBudgetBytes))
	}
	if len(report.TextMeasurements) == 0 {
		issues = append(issues, "text_measurements evidence is required")
	}
	if len(report.FontFallbacks) == 0 {
		issues = append(issues, "font_fallback evidence is required")
	}
	if len(report.GlyphCaches) == 0 {
		issues = append(issues, "glyph cache evidence is required")
	}
	if len(report.TextRenderCommands) == 0 {
		issues = append(issues, "text render command evidence is required")
	}

	measurementIDs := map[string]TextMeasurementReport{}
	hasWrapEllipsis := false
	for _, measurement := range report.TextMeasurements {
		if strings.TrimSpace(measurement.ID) == "" {
			issues = append(issues, "text_measurements id is required")
		} else if _, exists := measurementIDs[measurement.ID]; exists {
			issues = append(issues, fmt.Sprintf("text_measurements duplicate id %q", measurement.ID))
		}
		measurementIDs[measurement.ID] = measurement
		if measurement.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("text_measurements %q block_id must be positive", measurement.ID))
		}
		if measurement.TextLen <= 0 {
			issues = append(issues, fmt.Sprintf("text_measurements %q text_len must be positive", measurement.ID))
		}
		if strings.TrimSpace(measurement.FontFamily) == "" {
			issues = append(issues, fmt.Sprintf("text_measurements %q font_family is required", measurement.ID))
		}
		if measurement.FontSize <= 0 || measurement.LineHeight <= 0 || measurement.FontWeight <= 0 {
			issues = append(issues, fmt.Sprintf("text_measurements %q font metrics must be positive", measurement.ID))
		}
		if measurement.MaxWidth <= 0 || measurement.Measured.W <= 0 || measurement.Measured.H <= 0 || measurement.LineCount <= 0 {
			issues = append(issues, fmt.Sprintf("text_measurements %q measured dimensions and line_count must be positive", measurement.ID))
		}
		if measurement.MaxWidth > 0 && measurement.Measured.W > measurement.MaxWidth {
			issues = append(issues, fmt.Sprintf("text_measurements %q measured width %d exceeds max_width %d", measurement.ID, measurement.Measured.W, measurement.MaxWidth))
		}
		wrap := normalizeTextToken(measurement.Wrap)
		overflow := normalizeTextToken(measurement.Overflow)
		align := normalizeTextToken(measurement.Align)
		if wrap != "none" && wrap != "word" && wrap != "char" {
			issues = append(issues, fmt.Sprintf("text_measurements %q wrap is %q, want none, word, or char", measurement.ID, measurement.Wrap))
		}
		if overflow != "clip" && overflow != "ellipsis" {
			issues = append(issues, fmt.Sprintf("text_measurements %q overflow is %q, want clip or ellipsis", measurement.ID, measurement.Overflow))
		}
		if align != "start" && align != "center" && align != "end" {
			issues = append(issues, fmt.Sprintf("text_measurements %q align is %q, want start, center, or end", measurement.ID, measurement.Align))
		}
		if strings.TrimSpace(measurement.Quality) == "" {
			issues = append(issues, fmt.Sprintf("text_measurements %q quality is required", measurement.ID))
		}
		if !validChecksumLike(measurement.Checksum) {
			issues = append(issues, fmt.Sprintf("text_measurements %q checksum must be sha256 evidence", measurement.ID))
		}
		if measurement.Ellipsis || overflow == "ellipsis" {
			if measurement.EllipsizedTextLen <= 0 || measurement.EllipsizedTextLen >= measurement.TextLen {
				issues = append(issues, fmt.Sprintf("text_measurements %q ellipsis requires ellipsized_text_len between 1 and text_len-1", measurement.ID))
			}
		}
		if (wrap == "word" || wrap == "char") && overflow == "ellipsis" && measurement.Ellipsis && measurement.LineCount >= 1 && measurement.EllipsizedTextLen > 0 && measurement.EllipsizedTextLen < measurement.TextLen {
			hasWrapEllipsis = true
		}
	}
	if !hasWrapEllipsis {
		issues = append(issues, "text_measurements require wrap/ellipsis layout evidence")
	}

	for _, fallback := range report.FontFallbacks {
		if strings.TrimSpace(fallback.ID) == "" {
			issues = append(issues, "font_fallback id is required")
		}
		if strings.TrimSpace(fallback.RequestedFamily) == "" || strings.TrimSpace(fallback.ResolvedFamily) == "" {
			issues = append(issues, "font_fallback requested_family and resolved_family are required")
		}
		if len(fallback.Chain) < 2 {
			issues = append(issues, "font_fallback chain must include at least requested and fallback families")
		}
		if fallback.MissingGlyphs != 0 {
			issues = append(issues, fmt.Sprintf("font_fallback %q missing_glyphs = %d, want 0 for smoke coverage", fallback.ID, fallback.MissingGlyphs))
		}
		if strings.TrimSpace(fallback.Coverage) == "" {
			issues = append(issues, "font_fallback coverage is required")
		}
	}

	for _, cache := range report.GlyphCaches {
		if strings.TrimSpace(cache.ID) == "" {
			issues = append(issues, "glyph cache id is required")
		}
		if !cache.Bounded {
			issues = append(issues, "glyph cache must prove bounded=true")
		}
		if cache.BudgetBytes <= 0 || cache.BudgetBytes > report.TextCacheBudgetBytes {
			issues = append(issues, fmt.Sprintf("glyph cache %q budget_bytes = %d outside text cache budget %d", cache.ID, cache.BudgetBytes, report.TextCacheBudgetBytes))
		}
		if cache.UsedBytes < 0 || cache.UsedBytes > cache.BudgetBytes {
			issues = append(issues, fmt.Sprintf("glyph cache %q used_bytes = %d outside budget %d", cache.ID, cache.UsedBytes, cache.BudgetBytes))
		}
		if cache.EntryCount <= 0 {
			issues = append(issues, fmt.Sprintf("glyph cache %q entry_count must be positive", cache.ID))
		}
		if strings.TrimSpace(cache.Strategy) == "" || strings.TrimSpace(cache.Eviction) == "" {
			issues = append(issues, "glyph cache strategy and eviction are required")
		}
	}

	seenCommands := map[string]bool{}
	lastOrder := 0
	for i, command := range report.TextRenderCommands {
		name := normalizeTextToken(command.Command)
		if command.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("text_render_commands order %d is not strictly greater than previous order %d", command.Order, lastOrder))
		}
		lastOrder = command.Order
		if command.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("text_render_commands[%d] block_id must be positive", i))
		}
		if command.Rect.W <= 0 || command.Rect.H <= 0 || command.Clip.W <= 0 || command.Clip.H <= 0 {
			issues = append(issues, fmt.Sprintf("text_render_commands[%d] rect and clip dimensions must be positive", i))
		}
		if !rectContainsRect(command.Clip, command.Rect) && !rectContainsRect(command.Rect, command.Clip) {
			issues = append(issues, fmt.Sprintf("text_render_commands[%d] rect/clip must overlap as scoped text bounds", i))
		}
		if _, ok := measurementIDs[command.MeasurementID]; !ok {
			issues = append(issues, fmt.Sprintf("text_render_commands[%d] measurement_id %q is not in text_measurements", i, command.MeasurementID))
		}
		if name != "measure" && name != "render_glyphs" && name != "render_caret" {
			issues = append(issues, fmt.Sprintf("text_render_commands[%d] command is %q, want measure, render_glyphs, or render_caret", i, command.Command))
		}
		if strings.TrimSpace(command.Color) == "" || command.Opacity <= 0 {
			issues = append(issues, fmt.Sprintf("text_render_commands[%d] color and opacity are required", i))
		}
		if strings.TrimSpace(command.Quality) == "" {
			issues = append(issues, fmt.Sprintf("text_render_commands[%d] quality is required", i))
		}
		if !validChecksumLike(command.Checksum) {
			issues = append(issues, fmt.Sprintf("text_render_commands[%d] checksum must be sha256 evidence", i))
		}
		seenCommands[name] = true
	}
	for _, command := range []string{"measure", "render_glyphs"} {
		if !seenCommands[command] {
			issues = append(issues, fmt.Sprintf("text render commands require %s command", command))
		}
	}
	if !hasEventTargetKind(report.Events, "InputBlock", "text_input") {
		issues = append(issues, "block text editable input requires text_input targeted to InputBlock")
	}
	if !hasTransition(report.StateTransitions, "InputBlock", "buffer") || !hasTransition(report.StateTransitions, "InputBlock", "caret") {
		issues = append(issues, "block text editable input requires InputBlock buffer and caret state transitions")
	}
	if len(report.Frames) < 2 || strings.TrimSpace(report.Frames[0].Checksum) == "" || report.Frames[0].Checksum == report.Frames[1].Checksum {
		issues = append(issues, "text frame checksum evidence must show rendered text change")
	}
	for _, required := range []string{
		"block text deterministic measurement",
		"block text wrap ellipsis layout",
		"block text font fallback chain",
		"block text bounded glyph cache",
		"block text render command evidence",
		"block text editable lifetime",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("text report requires %s evidence", required))
		}
	}
	return issues
}

func hasBlockTextEvidence(report Report) bool {
	return len(report.TextMeasurements) > 0 ||
		len(report.FontFallbacks) > 0 ||
		len(report.GlyphCaches) > 0 ||
		len(report.TextRenderCommands) > 0 ||
		strings.TrimSpace(report.TextQualityLevel) != "" ||
		report.TextCacheBudgetBytes != 0
}

func validateBlockLayoutEvidence(report Report) []string {
	if !hasBlockLayoutEvidence(report) {
		return nil
	}

	var issues []string
	if report.LayoutQualityLevel != "deterministic-block-layout-v1" {
		issues = append(issues, fmt.Sprintf("layout_quality_level is %q, want deterministic-block-layout-v1", report.LayoutQualityLevel))
	}
	if report.LayoutUnsupportedCSSFlexbox {
		issues = append(issues, "layout unsupported CSS flexbox claim must be false")
	}
	if len(report.LayoutFeatures) == 0 {
		issues = append(issues, "layout_features evidence is required")
	}
	if len(report.LayoutConstraints) == 0 {
		issues = append(issues, "layout_constraints evidence is required")
	}
	if len(report.LayoutPasses) == 0 {
		issues = append(issues, "layout_passes evidence is required")
	}
	if len(report.LayoutScrolls) == 0 {
		issues = append(issues, "layout_scrolls evidence is required")
	}

	for _, feature := range []string{"stack", "row", "column", "absolute", "overlay", "grid", "dock", "scroll", "fit", "fill", "fixed", "min", "max", "spacing", "alignment", "z-order", "clipping", "resize"} {
		if !layoutFeatureContains(report.LayoutFeatures, feature) {
			issues = append(issues, fmt.Sprintf("layout_features require %s", feature))
		}
	}

	hasPolicy := map[string]bool{}
	hasMin := false
	hasMax := false
	hasSpacing := false
	hasAlignment := false
	hasClip := false
	hasZ := false
	constraintIDs := map[string]bool{}
	for _, constraint := range report.LayoutConstraints {
		id := strings.TrimSpace(constraint.ID)
		if id == "" {
			issues = append(issues, "layout_constraints id is required")
		} else if constraintIDs[id] {
			issues = append(issues, fmt.Sprintf("layout_constraints duplicate id %q", id))
		}
		constraintIDs[id] = true
		if constraint.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("layout_constraints %q block_id must be positive", id))
		}
		if !validLayoutMode(constraint.Mode) {
			issues = append(issues, fmt.Sprintf("layout_constraints %q mode is %q, want supported Block layout mode", id, constraint.Mode))
		}
		widthPolicy := normalizeLayoutToken(constraint.WidthPolicy)
		heightPolicy := normalizeLayoutToken(constraint.HeightPolicy)
		if !validLayoutPolicy(widthPolicy) {
			issues = append(issues, fmt.Sprintf("layout_constraints %q width_policy is %q, want fixed, fit, or fill", id, constraint.WidthPolicy))
		}
		if !validLayoutPolicy(heightPolicy) {
			issues = append(issues, fmt.Sprintf("layout_constraints %q height_policy is %q, want fixed, fit, or fill", id, constraint.HeightPolicy))
		}
		hasPolicy[widthPolicy] = true
		hasPolicy[heightPolicy] = true
		if constraint.Min.W > 0 && constraint.Min.H > 0 {
			hasMin = true
		} else {
			issues = append(issues, fmt.Sprintf("layout_constraints %q min dimensions must be positive", id))
		}
		if constraint.Max.W > 0 && constraint.Max.H > 0 {
			hasMax = true
		} else {
			issues = append(issues, fmt.Sprintf("layout_constraints %q max dimensions must be positive", id))
		}
		if constraint.Max.W > 0 && constraint.Min.W > constraint.Max.W {
			issues = append(issues, fmt.Sprintf("layout_constraints %q min.w exceeds max.w", id))
		}
		if constraint.Max.H > 0 && constraint.Min.H > constraint.Max.H {
			issues = append(issues, fmt.Sprintf("layout_constraints %q min.h exceeds max.h", id))
		}
		if constraint.Padding < 0 || constraint.Margin < 0 || constraint.Gap < 0 {
			issues = append(issues, fmt.Sprintf("layout_constraints %q spacing values must be non-negative", id))
		}
		if constraint.Padding > 0 || constraint.Margin > 0 || constraint.Gap > 0 {
			hasSpacing = true
		}
		if !validLayoutAlign(constraint.Align) {
			issues = append(issues, fmt.Sprintf("layout_constraints %q align is %q, want start, center, end, or stretch", id, constraint.Align))
		} else {
			hasAlignment = true
		}
		if !validLayoutJustify(constraint.Justify) {
			issues = append(issues, fmt.Sprintf("layout_constraints %q justify is %q, want start, center, end, or space-between", id, constraint.Justify))
		}
		if !validLayoutOverflow(constraint.Overflow) {
			issues = append(issues, fmt.Sprintf("layout_constraints %q overflow is %q, want visible, clip, or scroll", id, constraint.Overflow))
		}
		if constraint.ZIndex > 0 {
			hasZ = true
		}
		if constraint.Clip || normalizeLayoutToken(constraint.Overflow) == "clip" || normalizeLayoutToken(constraint.Overflow) == "scroll" {
			hasClip = true
		}
	}
	for _, policy := range []string{"fixed", "fit", "fill"} {
		if !hasPolicy[policy] {
			issues = append(issues, fmt.Sprintf("layout_constraints require %s sizing policy evidence", policy))
		}
	}
	if !hasMin {
		issues = append(issues, "layout_constraints require min sizing evidence")
	}
	if !hasMax {
		issues = append(issues, "layout_constraints require max sizing evidence")
	}
	if !hasSpacing {
		issues = append(issues, "layout_constraints require spacing evidence")
	}
	if !hasAlignment {
		issues = append(issues, "layout_constraints require alignment evidence")
	}
	if !hasZ {
		issues = append(issues, "layout_constraints require z-order evidence")
	}
	if !hasClip {
		issues = append(issues, "layout_constraints require clipping evidence")
	}

	passModes := map[string]bool{}
	lastOrder := 0
	hasResize := false
	for i, pass := range report.LayoutPasses {
		if pass.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("layout_passes order %d is not strictly greater than previous order %d", pass.Order, lastOrder))
		}
		lastOrder = pass.Order
		mode := normalizeLayoutToken(pass.Mode)
		if !validLayoutMode(mode) {
			issues = append(issues, fmt.Sprintf("layout_passes[%d] mode is %q, want supported Block layout mode", i, pass.Mode))
		}
		passModes[mode] = true
		if pass.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("layout_passes[%d] block_id must be positive", i))
		}
		if pass.Resolved.W <= 0 || pass.Resolved.H <= 0 || pass.Measured.W <= 0 || pass.Measured.H <= 0 {
			issues = append(issues, fmt.Sprintf("layout_passes[%d] resolved and measured dimensions must be positive", i))
		}
		if strings.TrimSpace(pass.Pass) == "" {
			issues = append(issues, fmt.Sprintf("layout_passes[%d] pass is required", i))
		}
		if !validChecksumLike(pass.Checksum) {
			issues = append(issues, fmt.Sprintf("layout_passes[%d] checksum must be sha256 evidence", i))
		}
		if pass.Resize && (pass.Input.W != pass.Resolved.W || pass.Input.H != pass.Resolved.H) {
			hasResize = true
		}
	}
	for _, mode := range []string{"stack", "row", "column", "absolute", "overlay", "grid", "dock", "scroll"} {
		if !passModes[mode] {
			issues = append(issues, fmt.Sprintf("layout_passes require %s mode evidence", mode))
		}
	}
	if !hasResize {
		issues = append(issues, "layout_passes require resize evidence with changed bounds")
	}

	hasScroll := false
	for i, scroll := range report.LayoutScrolls {
		if scroll.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("layout_scrolls[%d] block_id must be positive", i))
		}
		if scroll.Viewport.W <= 0 || scroll.Viewport.H <= 0 || scroll.Content.W <= 0 || scroll.Content.H <= 0 {
			issues = append(issues, fmt.Sprintf("layout_scrolls[%d] viewport and content dimensions must be positive", i))
		}
		if scroll.Content.H <= scroll.Viewport.H {
			issues = append(issues, fmt.Sprintf("layout_scrolls[%d] content.h must exceed viewport.h for scroll evidence", i))
		}
		if scroll.OffsetY < 0 || scroll.MaxOffsetY < 0 || scroll.OffsetY > scroll.MaxOffsetY {
			issues = append(issues, fmt.Sprintf("layout_scrolls[%d] offset_y must be within max_offset_y", i))
		}
		if !scroll.Clipped {
			issues = append(issues, fmt.Sprintf("layout_scrolls[%d] clipped must be true", i))
		}
		if !validChecksumLike(scroll.Checksum) {
			issues = append(issues, fmt.Sprintf("layout_scrolls[%d] checksum must be sha256 evidence", i))
		}
		if scroll.Clipped && scroll.Content.H > scroll.Viewport.H && scroll.OffsetY <= scroll.MaxOffsetY {
			hasScroll = true
		}
	}
	if !hasScroll {
		issues = append(issues, "layout_scrolls require clipped scroll bounds evidence")
	}

	if !hasEventTargetKind(report.Events, "ScrollBlock", "scroll") {
		issues = append(issues, "block layout scroll requires scroll event targeted to ScrollBlock")
	}
	if !hasTransition(report.StateTransitions, "BlockLayoutApp", "width") || !hasTransition(report.StateTransitions, "ScrollBlock", "scroll_y") {
		issues = append(issues, "block layout requires resize width and scroll_y state transitions")
	}
	if len(report.Frames) < 2 || strings.TrimSpace(report.Frames[0].Checksum) == "" || report.Frames[0].Checksum == report.Frames[1].Checksum {
		issues = append(issues, "layout frame checksum evidence must show responsive layout change")
	}
	for _, required := range []string{
		"block layout nested row column",
		"block layout fit fill fixed min max",
		"block layout grid dock overlay scroll",
		"block layout clipping z-order",
		"block layout resize constraints",
		"block layout no css flexbox parity",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("layout report requires %s evidence", required))
		}
	}
	return issues
}

func hasBlockLayoutEvidence(report Report) bool {
	return len(report.LayoutConstraints) > 0 ||
		len(report.LayoutPasses) > 0 ||
		len(report.LayoutScrolls) > 0 ||
		len(report.LayoutFeatures) > 0 ||
		strings.TrimSpace(report.LayoutQualityLevel) != "" ||
		report.LayoutUnsupportedCSSFlexbox
}

func validateBlockEventFocusEvidence(report Report) []string {
	if !hasBlockEventFocusEvidence(report) {
		return nil
	}

	var issues []string
	if report.BlockEventQualityLevel != "deterministic-block-events-v1" {
		issues = append(issues, fmt.Sprintf("block_event_quality_level is %q, want deterministic-block-events-v1", report.BlockEventQualityLevel))
	}
	if report.BlockEventPolicy != "capture-bubble-direct-v1" {
		issues = append(issues, fmt.Sprintf("block_event_policy is %q, want capture-bubble-direct-v1", report.BlockEventPolicy))
	}
	if report.BlockEventUnsupportedDragDrop {
		issues = append(issues, "block_event unsupported drag-and-drop claim must be false")
	}
	if report.BlockGraph == nil {
		issues = append(issues, "block_event evidence requires block_graph")
		return issues
	}

	nodes := map[int]BlockGraphNodeReport{}
	for _, node := range report.BlockGraph.Nodes {
		nodes[node.ID] = node
	}
	for _, kind := range []string{"pointer_enter", "pointer_leave", "pointer_move", "pointer_down", "pointer_up", "click", "double_click", "key", "text", "focus", "blur", "scroll", "resize", "close", "frame"} {
		if !containsNormalizedEventKind(report.BlockEventKinds, kind) {
			issues = append(issues, fmt.Sprintf("block_event_kinds require %s", kind))
		}
	}
	if len(report.BlockEventRoutes) == 0 {
		issues = append(issues, "block_event_routes evidence is required")
	}
	if len(report.BlockFocusTransitions) == 0 {
		issues = append(issues, "block_focus_transitions evidence is required")
	}

	lastOrder := 0
	hasNestedHit := false
	hasCaptureBubbleDirect := false
	hasDisabledReject := false
	hasUnfocusedTextReject := false
	hasFocusedTextDeliver := false
	for _, route := range report.BlockEventRoutes {
		if route.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("block_event_routes order %d is not strictly greater than previous order %d", route.Order, lastOrder))
		}
		lastOrder = route.Order
		kind := normalizeEventToken(route.Kind)
		if kind == "" {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] kind is required", route.Order))
		}
		if !validBlockEventKind(kind) {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] kind is %q, want supported Block event kind", route.Order, route.Kind))
		}
		node, ok := nodes[route.TargetID]
		if !ok {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] target_id %d is not in block_graph", route.Order, route.TargetID))
			continue
		}
		if strings.TrimSpace(route.TargetName) == "" {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] target_name is required", route.Order))
		} else if route.TargetName != node.Name {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] target_name is %q, want block_graph node name %q", route.Order, route.TargetName, node.Name))
		}
		wantPath, ok := blockGraphPathToRoot(route.TargetID, nodes)
		if !ok {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] target_id %d is not reachable from root", route.Order, route.TargetID))
			continue
		}
		if !intSlicesEqual(route.DispatchPath, wantPath) {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] dispatch_path = %v, want %v", route.Order, route.DispatchPath, wantPath))
		}
		if len(route.HitTestPath) > 0 {
			if !intSlicesEqual(route.HitTestPath, wantPath) {
				issues = append(issues, fmt.Sprintf("block_event_routes[%d] hit_test_path = %v, want %v", route.Order, route.HitTestPath, wantPath))
			}
			if len(route.HitTestPath) >= 3 {
				hasNestedHit = true
			}
		}
		if route.DirectTargetID != route.TargetID {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] direct_target_id = %d, want target_id %d", route.Order, route.DirectTargetID, route.TargetID))
		}
		if route.Delivered == route.Rejected {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] must be exactly one of delivered or rejected", route.Order))
		}
		if route.Rejected && strings.TrimSpace(route.RejectReason) == "" {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] rejected route requires reject_reason", route.Order))
		}
		if strings.TrimSpace(route.Policy) == "" {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] policy is required", route.Order))
		}
		if normalizeEventPolicy(route.Policy) == "capture-bubble-direct-v1" {
			if !blockEventCapturePathMatches(route.CapturePath, wantPath) {
				issues = append(issues, fmt.Sprintf("block_event_routes[%d] capture_path = %v, want ancestors", route.Order, route.CapturePath))
			}
			if !blockEventBubblePathMatches(route.BubblePath, wantPath) {
				issues = append(issues, fmt.Sprintf("block_event_routes[%d] bubble_path = %v, want reverse ancestors", route.Order, route.BubblePath))
			}
			hasCaptureBubbleDirect = true
		}
		reason := strings.ToLower(route.RejectReason)
		if kind == "click" && route.Disabled && route.Rejected && !route.Delivered && strings.Contains(reason, "disabled") {
			hasDisabledReject = true
		}
		if kind == "text" && route.Editable && route.Rejected && !route.Delivered && strings.Contains(reason, "unfocused") && route.FocusedID != route.TargetID {
			hasUnfocusedTextReject = true
		}
		if kind == "text" && route.Editable && route.Delivered && !route.Rejected && route.FocusedID == route.TargetID && route.TextLen > 0 && strings.TrimSpace(route.TextBytesHex) != "" {
			payload, err := hex.DecodeString(route.TextBytesHex)
			if err != nil {
				issues = append(issues, fmt.Sprintf("block_event_routes[%d] text_bytes_hex is not valid hex", route.Order))
			} else if len(payload) != route.TextLen {
				issues = append(issues, fmt.Sprintf("block_event_routes[%d] text_len = %d, want decoded payload length %d", route.Order, route.TextLen, len(payload)))
			}
			hasFocusedTextDeliver = true
		}
	}
	if !hasNestedHit {
		issues = append(issues, "block_event_routes require nested hit_test_path evidence")
	}
	if !hasCaptureBubbleDirect {
		issues = append(issues, "block_event_routes require capture-bubble-direct policy evidence")
	}
	if !hasDisabledReject {
		issues = append(issues, "block_event_routes require disabled click rejection evidence")
	}
	if !hasUnfocusedTextReject {
		issues = append(issues, "block_event_routes require unfocused text rejection evidence")
	}
	if !hasFocusedTextDeliver {
		issues = append(issues, "block_event_routes require focused editable text delivery evidence")
	}

	expectedFocus := report.BlockGraph.FocusOrder
	if len(expectedFocus) < 2 {
		issues = append(issues, "block_focus_transitions require at least two focusable Block IDs")
	}
	hasGraphDerived := false
	hasWrap := false
	lastFocusOrder := 0
	for _, transition := range report.BlockFocusTransitions {
		if transition.Order <= lastFocusOrder {
			issues = append(issues, fmt.Sprintf("block_focus_transitions order %d is not strictly greater than previous order %d", transition.Order, lastFocusOrder))
		}
		lastFocusOrder = transition.Order
		if transition.Helper != "tree_focus_next" && transition.Helper != "tree_focus_prev" {
			issues = append(issues, fmt.Sprintf("block_focus_transitions[%d] helper is %q, want tree_focus_next or tree_focus_prev", transition.Order, transition.Helper))
		}
		if !transition.GraphDerived {
			issues = append(issues, fmt.Sprintf("block_focus_transitions[%d] must prove graph_derived", transition.Order))
		}
		if !containsInt(expectedFocus, transition.BeforeID) || !containsInt(expectedFocus, transition.AfterID) {
			issues = append(issues, fmt.Sprintf("block_focus_transitions[%d] before/after must be in block_graph focus_order %v", transition.Order, expectedFocus))
		}
		if transition.GraphDerived {
			hasGraphDerived = true
		}
		if transition.Wrapped && len(expectedFocus) >= 2 {
			if transition.BeforeID != expectedFocus[len(expectedFocus)-1] || transition.AfterID != expectedFocus[0] {
				issues = append(issues, fmt.Sprintf("block_focus_transitions[%d] wrap = %d -> %d, want %d -> %d", transition.Order, transition.BeforeID, transition.AfterID, expectedFocus[len(expectedFocus)-1], expectedFocus[0]))
			}
			hasWrap = true
		}
	}
	if !hasGraphDerived || !hasWrap {
		issues = append(issues, "block_focus_transitions require graph-derived tab wrap evidence")
	}

	for _, required := range []string{
		"block event nested hit-test path",
		"block event capture bubble direct policy",
		"block event disabled click rejected",
		"block event text input focused only",
		"block focus tab order graph-derived",
		"block event no complex drag claim",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("block_event report requires %s evidence", required))
		}
	}
	return issues
}

func hasBlockEventFocusEvidence(report Report) bool {
	return len(report.BlockEventRoutes) > 0 ||
		len(report.BlockFocusTransitions) > 0 ||
		len(report.BlockEventKinds) > 0 ||
		strings.TrimSpace(report.BlockEventPolicy) != "" ||
		strings.TrimSpace(report.BlockEventQualityLevel) != "" ||
		report.BlockEventUnsupportedDragDrop
}

func validateBlockStateEvidence(report Report) []string {
	if !hasBlockStateEvidence(report) {
		return nil
	}

	var issues []string
	if report.BlockStateQualityLevel != "deterministic-block-state-resolver-v1" {
		issues = append(issues, fmt.Sprintf("block_state_quality_level is %q, want deterministic-block-state-resolver-v1", report.BlockStateQualityLevel))
	}
	if report.BlockStateUnsupportedCSSPseudos {
		issues = append(issues, "block_state unsupported css pseudo parity claim must be false")
	}
	expectedOrder := []string{"base", "variant", "hover", "pressed", "focused", "selected", "disabled", "error", "loading", "motion"}
	if !normalizedStringListEqual(report.BlockStateResolverOrder, expectedOrder) {
		issues = append(issues, fmt.Sprintf("block_state resolver order = %v, want %v", report.BlockStateResolverOrder, expectedOrder))
	}
	if len(report.BlockStateSelectors) == 0 {
		issues = append(issues, "block_state_selectors evidence is required")
	}
	if len(report.BlockStateResolutions) == 0 {
		issues = append(issues, "block_state_resolutions evidence is required")
	}

	expectedSelectors := map[string]struct {
		flag  int
		check func(BlockStateSelectorReport) bool
	}{
		"hover":    {flag: 1, check: func(selector BlockStateSelectorReport) bool { return selector.Hovered }},
		"pressed":  {flag: 2, check: func(selector BlockStateSelectorReport) bool { return selector.Pressed }},
		"focused":  {flag: 4, check: func(selector BlockStateSelectorReport) bool { return selector.Focused }},
		"selected": {flag: 8, check: func(selector BlockStateSelectorReport) bool { return selector.Selected }},
		"disabled": {flag: 16, check: func(selector BlockStateSelectorReport) bool { return selector.Disabled }},
		"error":    {flag: 32, check: func(selector BlockStateSelectorReport) bool { return selector.Error }},
		"loading":  {flag: 64, check: func(selector BlockStateSelectorReport) bool { return selector.Loading }},
	}
	seenSelectors := map[string]bool{}
	lastSelectorOrder := 0
	for _, selector := range report.BlockStateSelectors {
		if selector.Order <= lastSelectorOrder {
			issues = append(issues, fmt.Sprintf("block_state_selectors order %d is not strictly greater than previous order %d", selector.Order, lastSelectorOrder))
		}
		lastSelectorOrder = selector.Order
		name := normalizeStateToken(selector.Name)
		spec, ok := expectedSelectors[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("block_state_selectors[%d] name is %q, want supported Block state selector", selector.Order, selector.Name))
			continue
		}
		if selector.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("block_state_selectors[%d] block_id must be positive", selector.Order))
		}
		if selector.Flags != spec.flag {
			issues = append(issues, fmt.Sprintf("block_state_selectors[%d] %s flags = %d, want %d", selector.Order, name, selector.Flags, spec.flag))
		}
		if !spec.check(selector) {
			issues = append(issues, fmt.Sprintf("block_state_selectors[%d] %s selector boolean evidence is missing", selector.Order, name))
		}
		seenSelectors[name] = true
	}
	for name := range expectedSelectors {
		if !seenSelectors[name] {
			issues = append(issues, fmt.Sprintf("block_state_selectors require %s selector evidence", name))
		}
	}

	requiredProperties := map[string]map[string]bool{
		"hover":    {"paint.fill": false},
		"pressed":  {"layout.scale": false},
		"focused":  {"paint.outline": false},
		"selected": {"accessibility.selected": false},
		"disabled": {"input.disabled": false, "text.opacity": false},
		"error":    {"paint.outline_color": false},
		"loading":  {"text.content": false},
		"motion":   {"motion.transition_ms": false},
	}
	lastResolutionOrder := 0
	for _, resolution := range report.BlockStateResolutions {
		if resolution.Order <= lastResolutionOrder {
			issues = append(issues, fmt.Sprintf("block_state_resolutions order %d is not strictly greater than previous order %d", resolution.Order, lastResolutionOrder))
		}
		lastResolutionOrder = resolution.Order
		selector := normalizeStateToken(resolution.Selector)
		step := normalizeStateToken(resolution.ResolverStep)
		property := normalizeStateProperty(resolution.Property)
		properties, ok := requiredProperties[selector]
		if !ok {
			issues = append(issues, fmt.Sprintf("block_state_resolutions[%d] selector is %q, want supported selector or motion", resolution.Order, resolution.Selector))
			continue
		}
		if resolution.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("block_state_resolutions[%d] block_id must be positive", resolution.Order))
		}
		if step == "" {
			issues = append(issues, fmt.Sprintf("block_state_resolutions[%d] resolver_step is required", resolution.Order))
		} else if step != selector {
			issues = append(issues, fmt.Sprintf("block_state_resolutions[%d] resolver_step is %q, want selector %q", resolution.Order, resolution.ResolverStep, resolution.Selector))
		}
		if property == "" {
			issues = append(issues, fmt.Sprintf("block_state_resolutions[%d] property is required", resolution.Order))
		}
		if strings.TrimSpace(resolution.Before) == "" || strings.TrimSpace(resolution.After) == "" {
			issues = append(issues, fmt.Sprintf("block_state_resolutions[%d] before and after values are required", resolution.Order))
		}
		if !resolution.Applied {
			issues = append(issues, fmt.Sprintf("block_state_resolutions[%d] %s %s override must be applied", resolution.Order, selector, property))
		}
		if resolution.Applied && strings.TrimSpace(resolution.Before) == strings.TrimSpace(resolution.After) {
			issues = append(issues, fmt.Sprintf("block_state_resolutions[%d] %s %s before/after must change", resolution.Order, selector, property))
		}
		if _, want := properties[property]; want {
			properties[property] = true
		}
	}
	for selector, properties := range requiredProperties {
		for property, seen := range properties {
			if !seen {
				issues = append(issues, fmt.Sprintf("block_state_resolutions require %s %s evidence", selector, property))
			}
		}
	}

	if !hasEventTargetKind(report.Events, "StateBlock", "mouse_move") ||
		!hasEventTargetKind(report.Events, "StateBlock", "mouse_down") ||
		!hasEventTargetKind(report.Events, "StateBlock", "key_down") {
		issues = append(issues, "block_state evidence requires StateBlock hover/press/focus events")
	}
	for _, field := range []string{"selector_flags", "resolved_fill", "resolved_scale", "disabled", "error", "loading"} {
		if !hasTransition(report.StateTransitions, "StateBlock", field) {
			issues = append(issues, fmt.Sprintf("block_state evidence requires StateBlock %s state transition", field))
		}
	}
	if len(report.Frames) < 2 || strings.TrimSpace(report.Frames[0].Checksum) == "" || report.Frames[0].Checksum == report.Frames[1].Checksum {
		issues = append(issues, "block_state frame checksum evidence must show state-driven visual change")
	}
	for _, required := range []string{
		"block state selector resolver order",
		"block state hover fill override",
		"block state pressed scale override",
		"block state focus selected metadata",
		"block state disabled error loading overrides",
		"block state frame checksum changed",
		"block state no css pseudo parity",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("block_state report requires %s evidence", required))
		}
	}
	return issues
}

func hasBlockStateEvidence(report Report) bool {
	return len(report.BlockStateSelectors) > 0 ||
		len(report.BlockStateResolutions) > 0 ||
		len(report.BlockStateResolverOrder) > 0 ||
		strings.TrimSpace(report.BlockStateQualityLevel) != "" ||
		report.BlockStateUnsupportedCSSPseudos
}

func validateBlockMotionEvidence(report Report) []string {
	if !hasBlockMotionEvidence(report) {
		return nil
	}

	var issues []string
	if report.MotionQualityLevel != "deterministic-block-motion-v1" {
		issues = append(issues, fmt.Sprintf("motion_quality_level is %q, want deterministic-block-motion-v1", report.MotionQualityLevel))
	}
	if report.MotionClock != "deterministic-test-clock-v1" {
		issues = append(issues, fmt.Sprintf("motion_clock is %q, want deterministic-test-clock-v1", report.MotionClock))
	}
	if report.MotionUnsupportedCSSAnimations {
		issues = append(issues, "motion unsupported css animation parity claim must be false")
	}
	if report.MotionFrameBudget <= 0 || report.MotionFrameBudget > 16 {
		issues = append(issues, fmt.Sprintf("motion_frame_budget = %d, want 1..16", report.MotionFrameBudget))
	}
	if len(report.MotionFrames) == 0 {
		issues = append(issues, "motion_frames evidence is required")
	}
	if report.MotionFrameBudget > 0 && len(report.MotionFrames) > report.MotionFrameBudget {
		issues = append(issues, fmt.Sprintf("motion_frames length %d exceeds motion_frame_budget %d", len(report.MotionFrames), report.MotionFrameBudget))
	}

	var startFrame *MotionFrameReport
	var midFrame *MotionFrameReport
	var finalFrame *MotionFrameReport
	var reducedFrame *MotionFrameReport
	lastOrder := 0
	lastTimestamp := -1
	for i := range report.MotionFrames {
		frame := &report.MotionFrames[i]
		if frame.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("motion_frames order %d is not strictly greater than previous order %d", frame.Order, lastOrder))
		}
		lastOrder = frame.Order
		if frame.TimestampMS < lastTimestamp {
			issues = append(issues, fmt.Sprintf("motion_frames[%d] timestamp_ms %d is before previous timestamp %d", frame.Order, frame.TimestampMS, lastTimestamp))
		}
		lastTimestamp = frame.TimestampMS
		if frame.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("motion_frames[%d] block_id must be positive", frame.Order))
		}
		if strings.TrimSpace(frame.Trigger) == "" {
			issues = append(issues, fmt.Sprintf("motion_frames[%d] trigger is required", frame.Order))
		}
		if frame.DurationMS <= 0 {
			issues = append(issues, fmt.Sprintf("motion_frames[%d] duration_ms must be positive", frame.Order))
		}
		if frame.DelayMS < 0 {
			issues = append(issues, fmt.Sprintf("motion_frames[%d] delay_ms must be non-negative", frame.Order))
		}
		if frame.Progress < 0 || frame.Progress > 1000 {
			issues = append(issues, fmt.Sprintf("motion_frames[%d] progress = %d, want 0..1000", frame.Order, frame.Progress))
		}
		if normalizeStateToken(frame.Easing) == "" {
			issues = append(issues, fmt.Sprintf("motion_frames[%d] easing is required", frame.Order))
		}
		if frame.Opacity <= 0 || frame.Opacity > 255 {
			issues = append(issues, fmt.Sprintf("motion_frames[%d] opacity = %d, want 1..255", frame.Order, frame.Opacity))
		}
		if strings.TrimSpace(frame.Color) == "" {
			issues = append(issues, fmt.Sprintf("motion_frames[%d] color is required", frame.Order))
		}
		if frame.Scale <= 0 {
			issues = append(issues, fmt.Sprintf("motion_frames[%d] scale must be positive", frame.Order))
		}
		if !validChecksumLike(frame.Checksum) {
			issues = append(issues, fmt.Sprintf("motion_frames[%d] checksum must be sha256 evidence", frame.Order))
		}
		if frame.Settled && frame.Scheduled {
			issues = append(issues, fmt.Sprintf("motion_frames[%d] settled frame must not keep scheduling", frame.Order))
		}
		if frame.ReducedMotion {
			if frame.Progress != 1000 || frame.Scheduled || !frame.Settled {
				issues = append(issues, fmt.Sprintf("motion_frames[%d] reduced motion must instantly settle without scheduling", frame.Order))
			}
			if reducedFrame == nil {
				reducedFrame = frame
			}
			continue
		}
		if frame.Progress == 0 && startFrame == nil {
			startFrame = frame
		}
		if frame.Progress > 0 && frame.Progress < 1000 && midFrame == nil {
			midFrame = frame
		}
		if frame.Progress == 1000 && finalFrame == nil {
			finalFrame = frame
		}
		if frame.Progress == 1000 && (!frame.Settled || frame.Scheduled) {
			issues = append(issues, fmt.Sprintf("motion_frames[%d] completed motion must be settled and stop scheduling", frame.Order))
		}
	}
	if startFrame == nil || midFrame == nil || finalFrame == nil {
		issues = append(issues, "motion_frames require start, interpolated middle, and settled final frames")
	}
	if reducedFrame == nil {
		issues = append(issues, "motion_frames require reduced motion instant-settle evidence")
	}
	if startFrame != nil && midFrame != nil && finalFrame != nil {
		if startFrame.Opacity == finalFrame.Opacity || midFrame.Opacity == startFrame.Opacity || midFrame.Opacity == finalFrame.Opacity {
			issues = append(issues, "motion_frames require opacity interpolation evidence")
		}
		if startFrame.Color == finalFrame.Color || midFrame.Color == startFrame.Color || midFrame.Color == finalFrame.Color {
			issues = append(issues, "motion_frames require color interpolation evidence")
		}
		if startFrame.Scale == finalFrame.Scale || midFrame.Scale == startFrame.Scale || midFrame.Scale == finalFrame.Scale {
			issues = append(issues, "motion_frames require scale interpolation evidence")
		}
		if startFrame.TranslateX == finalFrame.TranslateX || midFrame.TranslateX == startFrame.TranslateX || midFrame.TranslateX == finalFrame.TranslateX {
			issues = append(issues, "motion_frames require translate interpolation evidence")
		}
	}

	if !hasEventTargetKind(report.Events, "MotionBlock", "mouse_up") {
		issues = append(issues, "block motion evidence requires MotionBlock trigger event")
	}
	for _, field := range []string{"opacity", "color", "scale", "translate_x", "motion_complete", "reduced_motion"} {
		if !hasTransition(report.StateTransitions, "MotionBlock", field) {
			issues = append(issues, fmt.Sprintf("block motion evidence requires MotionBlock %s state transition", field))
		}
	}
	if len(report.Frames) < 3 || strings.TrimSpace(report.Frames[0].Checksum) == "" || report.Frames[0].Checksum == report.Frames[1].Checksum || report.Frames[1].Checksum == report.Frames[2].Checksum {
		issues = append(issues, "block motion frame checksum evidence must show motion-driven visual changes")
	}
	for _, required := range []string{
		"block motion deterministic test clock",
		"block motion opacity color transform frames",
		"block motion reduced motion instant settle",
		"block motion completion stops scheduling",
		"block motion frame checksum changed",
		"block motion no css animation parity",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("block motion report requires %s evidence", required))
		}
	}
	return issues
}

func hasBlockMotionEvidence(report Report) bool {
	return len(report.MotionFrames) > 0 ||
		strings.TrimSpace(report.MotionQualityLevel) != "" ||
		strings.TrimSpace(report.MotionClock) != "" ||
		report.MotionFrameBudget != 0 ||
		report.MotionUnsupportedCSSAnimations
}

func validateBlockAssetEvidence(report Report) []string {
	if !hasBlockAssetEvidence(report) {
		return nil
	}

	var issues []string
	if report.BlockAssetQualityLevel != "deterministic-local-block-assets-v1" {
		issues = append(issues, fmt.Sprintf("block_asset_quality_level is %q, want deterministic-local-block-assets-v1", report.BlockAssetQualityLevel))
	}
	if report.BlockAssetNetworkFetchAllowed {
		issues = append(issues, "block asset network fetch must be disabled")
	}

	assetIDs := map[string]BlockAssetReport{}
	if report.BlockAssetManifest == nil {
		issues = append(issues, "block_asset_manifest evidence is required")
	} else {
		manifest := report.BlockAssetManifest
		if manifest.Schema != "tetra.surface.block-assets.v1" {
			issues = append(issues, fmt.Sprintf("block_asset_manifest schema is %q, want tetra.surface.block-assets.v1", manifest.Schema))
		}
		if manifest.Quality != "deterministic-local-block-assets-v1" {
			issues = append(issues, fmt.Sprintf("block_asset_manifest quality is %q, want deterministic-local-block-assets-v1", manifest.Quality))
		}
		if manifest.HashAlgorithm != "sha256" {
			issues = append(issues, fmt.Sprintf("block_asset_manifest hash_algorithm is %q, want sha256", manifest.HashAlgorithm))
		}
		if !validSHA256Digest(manifest.ManifestHash) {
			issues = append(issues, "block_asset_manifest manifest_hash must be sha256 evidence")
		}
		if strings.TrimSpace(manifest.Source) == "" || normalizeEvidencePath(manifest.Source) != normalizeEvidencePath(report.Source) {
			issues = append(issues, "block_asset_manifest source must match report source")
		}
		if !manifest.LocalOnly || manifest.RemoteCount != 0 {
			issues = append(issues, "block_asset_manifest must be local-only with remote_count 0")
		}
		if manifest.FontCount <= 0 || manifest.IconCount <= 0 || manifest.ImageCount <= 0 {
			issues = append(issues, "block_asset_manifest requires font, icon, and image counts")
		}
		if manifest.EmbeddedCount <= 0 {
			issues = append(issues, "block_asset_manifest requires embedded/local sample asset evidence")
		}
		if len(manifest.Assets) < 3 {
			issues = append(issues, "block_asset_manifest assets require font/icon/image asset hashes")
		}
		kindCounts := map[string]int{}
		for i, asset := range manifest.Assets {
			id := strings.TrimSpace(asset.ID)
			kind := normalizeStateToken(asset.Kind)
			if id == "" {
				issues = append(issues, fmt.Sprintf("block_asset_manifest assets[%d] id is required", i))
			} else if _, exists := assetIDs[id]; exists {
				issues = append(issues, fmt.Sprintf("block_asset_manifest duplicate asset id %q", id))
			}
			assetIDs[id] = asset
			if !validBlockAssetKind(kind) {
				issues = append(issues, fmt.Sprintf("block_asset_manifest asset %q kind is %q, want font, icon, or image", id, asset.Kind))
			}
			kindCounts[kind]++
			if strings.TrimSpace(asset.Path) == "" {
				issues = append(issues, fmt.Sprintf("block_asset_manifest asset %q path is required", id))
			}
			if isNetworkAssetPath(asset.Path) {
				issues = append(issues, fmt.Sprintf("block_asset_manifest asset %q uses network path %q", id, asset.Path))
			}
			if !asset.Local && !asset.Embedded {
				issues = append(issues, fmt.Sprintf("block_asset_manifest asset %q must be local or embedded", id))
			}
			if !validSHA256Digest(asset.SHA256) {
				issues = append(issues, fmt.Sprintf("block_asset_manifest asset %q sha256 must be present", id))
			}
			if asset.Size <= 0 {
				issues = append(issues, fmt.Sprintf("block_asset_manifest asset %q size must be positive", id))
			}
			if strings.TrimSpace(asset.CacheKey) == "" {
				issues = append(issues, fmt.Sprintf("block_asset_manifest asset %q cache_key is required", id))
			}
			if kind == "font" && strings.TrimSpace(asset.Family) == "" {
				issues = append(issues, fmt.Sprintf("block_asset_manifest font asset %q family is required", id))
			}
			if (kind == "icon" || kind == "image") && (asset.Width <= 0 || asset.Height <= 0) {
				issues = append(issues, fmt.Sprintf("block_asset_manifest %s asset %q width/height must be positive", kind, id))
			}
		}
		if kindCounts["font"] < manifest.FontCount || kindCounts["icon"] < manifest.IconCount || kindCounts["image"] < manifest.ImageCount {
			issues = append(issues, "block_asset_manifest counts must be backed by matching asset entries")
		}
	}

	cache := report.BlockAssetCache
	if strings.TrimSpace(cache.ID) == "" {
		issues = append(issues, "block_asset_cache evidence is required")
	}
	if normalizeStateToken(cache.Strategy) != "bounded_lru" {
		issues = append(issues, fmt.Sprintf("block_asset_cache strategy is %q, want bounded-lru", cache.Strategy))
	}
	if !cache.Bounded {
		issues = append(issues, "block_asset_cache must be bounded")
	}
	if cache.BudgetBytes <= 0 || cache.BudgetBytes > 1<<20 {
		issues = append(issues, fmt.Sprintf("block_asset_cache budget_bytes = %d, want 1..1048576", cache.BudgetBytes))
	}
	if cache.UsedBytes < 0 || (cache.BudgetBytes > 0 && cache.UsedBytes > cache.BudgetBytes) {
		issues = append(issues, "block_asset_cache used_bytes must be within budget")
	}
	if cache.MaxEntries <= 0 || cache.EntryCount < 0 || cache.EntryCount > cache.MaxEntries {
		issues = append(issues, "block_asset_cache entry_count must be within max_entries")
	}
	if cache.RepeatedLoads <= cache.EntryCount {
		issues = append(issues, "block_asset_cache repeated_loads must exceed entry_count to prove reuse")
	}
	if strings.TrimSpace(cache.Eviction) == "" {
		issues = append(issues, "block_asset_cache eviction policy is required")
	}

	if len(report.BlockAssetDiagnostics) == 0 {
		issues = append(issues, "block_asset_diagnostics evidence is required")
	}
	lastDiagnosticOrder := 0
	hasMissingDiagnostic := false
	hasNetworkDiagnostic := false
	for i, diagnostic := range report.BlockAssetDiagnostics {
		if diagnostic.Order <= lastDiagnosticOrder {
			issues = append(issues, fmt.Sprintf("block_asset_diagnostics order %d is not strictly greater than previous order %d", diagnostic.Order, lastDiagnosticOrder))
		}
		lastDiagnosticOrder = diagnostic.Order
		if strings.TrimSpace(diagnostic.AssetID) == "" || strings.TrimSpace(diagnostic.Kind) == "" || strings.TrimSpace(diagnostic.Code) == "" || strings.TrimSpace(diagnostic.Message) == "" {
			issues = append(issues, fmt.Sprintf("block_asset_diagnostics[%d] asset_id, kind, code, and message are required", i))
		}
		if !diagnostic.Pass {
			issues = append(issues, fmt.Sprintf("block_asset_diagnostics[%d] pass must be true", i))
		}
		code := normalizeStateToken(diagnostic.Code)
		if code == "missing_asset_fallback" {
			hasMissingDiagnostic = true
			if strings.TrimSpace(diagnostic.FallbackID) == "" {
				issues = append(issues, "block_asset_diagnostics missing asset fallback requires fallback_id")
			}
		}
		if code == "network_asset_rejected" {
			hasNetworkDiagnostic = true
			if !isNetworkAssetPath(diagnostic.RejectedURL) {
				issues = append(issues, "block_asset_diagnostics network rejection requires rejected_url")
			}
		}
	}
	if !hasMissingDiagnostic {
		issues = append(issues, "block_asset_diagnostics require missing asset fallback diagnostic")
	}
	if !hasNetworkDiagnostic {
		issues = append(issues, "block_asset_diagnostics require network asset rejection diagnostic")
	}

	if len(report.BlockAssetRenderCommands) == 0 {
		issues = append(issues, "block_asset_render_commands evidence is required")
	}
	lastCommandOrder := 0
	commands := map[string]bool{}
	for i, command := range report.BlockAssetRenderCommands {
		if command.Order <= lastCommandOrder {
			issues = append(issues, fmt.Sprintf("block_asset_render_commands order %d is not strictly greater than previous order %d", command.Order, lastCommandOrder))
		}
		lastCommandOrder = command.Order
		name := normalizeStateToken(command.Command)
		commands[name] = true
		if strings.TrimSpace(command.AssetID) == "" {
			issues = append(issues, fmt.Sprintf("block_asset_render_commands[%d] asset_id is required", i))
		}
		if command.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("block_asset_render_commands[%d] block_id must be positive", i))
		}
		if command.Rect.W <= 0 || command.Rect.H <= 0 {
			issues = append(issues, fmt.Sprintf("block_asset_render_commands[%d] rect dimensions must be positive", i))
		}
		if strings.TrimSpace(command.Quality) == "" {
			issues = append(issues, fmt.Sprintf("block_asset_render_commands[%d] quality is required", i))
		}
		if !validChecksumLike(command.Checksum) {
			issues = append(issues, fmt.Sprintf("block_asset_render_commands[%d] checksum must be sha256 evidence", i))
		}
		if name == "tint_icon" && strings.TrimSpace(command.Tint) == "" {
			issues = append(issues, "block_asset_render_commands tint_icon requires tint evidence")
		}
		if name == "scale_image" && command.Scale < 2 {
			issues = append(issues, "block_asset_render_commands scale_image requires scale evidence")
		}
	}
	for _, required := range []string{"load_font", "tint_icon", "scale_image", "fallback_missing"} {
		if !commands[required] {
			issues = append(issues, fmt.Sprintf("block_asset_render_commands require %s command", required))
		}
	}

	if !hasEventTargetKind(report.Events, "IconBlock", "mouse_up") {
		issues = append(issues, "block asset evidence requires IconBlock tint trigger event")
	}
	for _, requirement := range []struct {
		component string
		field     string
	}{
		{"IconBlock", "tint"},
		{"ImageBlock", "scale"},
		{"MissingAssetBlock", "fallback"},
	} {
		if !hasTransition(report.StateTransitions, requirement.component, requirement.field) {
			issues = append(issues, fmt.Sprintf("block asset evidence requires %s %s state transition", requirement.component, requirement.field))
		}
	}
	if len(report.Frames) < 2 || strings.TrimSpace(report.Frames[0].Checksum) == "" || report.Frames[0].Checksum == report.Frames[1].Checksum {
		issues = append(issues, "block asset frame checksum evidence must show asset-driven visual change")
	}
	for _, required := range []string{
		"block asset deterministic manifest hashes",
		"block asset local embedded only",
		"block asset bounded cache",
		"block asset icon tint evidence",
		"block asset image scale evidence",
		"block asset missing fallback diagnostic",
		"block asset network url rejected",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("block asset report requires %s evidence", required))
		}
	}
	return issues
}

func hasBlockAssetEvidence(report Report) bool {
	return report.BlockAssetManifest != nil ||
		strings.TrimSpace(report.BlockAssetQualityLevel) != "" ||
		report.BlockAssetNetworkFetchAllowed ||
		strings.TrimSpace(report.BlockAssetCache.ID) != "" ||
		len(report.BlockAssetDiagnostics) > 0 ||
		len(report.BlockAssetRenderCommands) > 0
}

func validBlockAssetKind(kind string) bool {
	return kind == "font" || kind == "icon" || kind == "image"
}

func isNetworkAssetPath(path string) bool {
	value := strings.ToLower(strings.TrimSpace(path))
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

func normalizeStateToken(value string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(value, "-", "_")))
}

func normalizeStateProperty(value string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(value, "-", "_")))
}

func normalizedStringListEqual(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if normalizeStateToken(got[i]) != normalizeStateToken(want[i]) {
			return false
		}
	}
	return true
}

func normalizeEventToken(value string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(value, "-", "_")))
}

func normalizeEventPolicy(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func validBlockEventKind(value string) bool {
	switch normalizeEventToken(value) {
	case "pointer_enter", "pointer_leave", "pointer_move", "pointer_down", "pointer_up", "click", "double_click", "key", "text", "focus", "blur", "scroll", "resize", "close", "frame":
		return true
	default:
		return false
	}
}

func containsNormalizedEventKind(values []string, want string) bool {
	want = normalizeEventToken(want)
	for _, value := range values {
		if normalizeEventToken(value) == want {
			return true
		}
	}
	return false
}

func blockEventCapturePathMatches(got []int, fullPath []int) bool {
	if len(fullPath) < 2 {
		return false
	}
	return intSlicesEqual(got, fullPath[:len(fullPath)-1])
}

func blockEventBubblePathMatches(got []int, fullPath []int) bool {
	if len(fullPath) < 2 || len(got) != len(fullPath)-1 {
		return false
	}
	for i := range got {
		if got[i] != fullPath[len(fullPath)-2-i] {
			return false
		}
	}
	return true
}

func validLayoutMode(value string) bool {
	switch normalizeLayoutToken(value) {
	case "fixed", "stack", "row", "column", "absolute", "overlay", "grid", "dock", "scroll":
		return true
	default:
		return false
	}
}

func validLayoutPolicy(value string) bool {
	switch normalizeLayoutToken(value) {
	case "fixed", "fit", "fill":
		return true
	default:
		return false
	}
}

func validLayoutAlign(value string) bool {
	switch normalizeLayoutToken(value) {
	case "start", "center", "end", "stretch":
		return true
	default:
		return false
	}
}

func validLayoutJustify(value string) bool {
	switch normalizeLayoutToken(value) {
	case "start", "center", "end", "space_between":
		return true
	default:
		return false
	}
}

func validLayoutOverflow(value string) bool {
	switch normalizeLayoutToken(value) {
	case "visible", "clip", "scroll":
		return true
	default:
		return false
	}
}

func layoutFeatureContains(features []string, want string) bool {
	want = normalizeLayoutToken(want)
	for _, feature := range features {
		if normalizeLayoutToken(feature) == want {
			return true
		}
	}
	return false
}

func normalizePaintToken(value string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(value, "-", "_")))
}

func normalizeTextToken(value string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(value, "-", "_")))
}

func normalizeLayoutToken(value string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(value, "-", "_")))
}

func visualFeatureContains(features []string, want string) bool {
	want = normalizePaintToken(want)
	for _, feature := range features {
		if normalizePaintToken(feature) == want {
			return true
		}
	}
	return false
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

func isSurfaceBlockAccessibilitySource(source string) bool {
	source = normalizeEvidencePath(source)
	return strings.HasSuffix(source, "examples/surface_block_accessibility.tetra") ||
		strings.HasSuffix(source, "examples/surface_block_system.tetra") ||
		strings.HasSuffix(source, "examples/surface_morph_command_palette.tetra")
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
