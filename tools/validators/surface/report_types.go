package surface

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
	BlockSceneSnapshot              *BlockSceneSnapshotReport       `json:"block_scene_snapshot,omitempty"`
	RenderCommandStream             *RenderCommandStreamReport      `json:"render_command_stream,omitempty"`
	PaintLayers                     []PaintLayerReport              `json:"paint_layers,omitempty"`
	PaintCommands                   []PaintCommandReport            `json:"paint_commands,omitempty"`
	VisualFeatures                  []string                        `json:"visual_features,omitempty"`
	PaintQualityLevel               string                          `json:"paint_quality_level,omitempty"`
	PaintCacheBudgetBytes           int                             `json:"paint_cache_budget_bytes,omitempty"`
	PaintUnsupportedBlur            bool                            `json:"paint_unsupported_blur,omitempty"`
	Renderer                        *RendererReport                 `json:"renderer,omitempty"`
	TextMeasurements                []TextMeasurementReport         `json:"text_measurements,omitempty"`
	FontFallbacks                   []FontFallbackReport            `json:"font_fallbacks,omitempty"`
	GlyphCaches                     []GlyphCacheReport              `json:"glyph_caches,omitempty"`
	TextRenderCommands              []TextRenderCommandReport       `json:"text_render_commands,omitempty"`
	TextQualityLevel                string                          `json:"text_quality_level,omitempty"`
	TextCacheBudgetBytes            int                             `json:"text_cache_budget_bytes,omitempty"`
	LayoutConstraints               []BlockLayoutConstraintReport   `json:"layout_constraints,omitempty"`
	LayoutPasses                    []BlockLayoutPassReport         `json:"layout_passes,omitempty"`
	LayoutScrolls                   []BlockLayoutScrollReport       `json:"layout_scrolls,omitempty"`
	LayoutDensity                   *BlockLayoutDensityReport       `json:"layout_density,omitempty"`
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
	AppModel                        *AppModelReport                 `json:"app_model,omitempty"`
	LinuxAppShell                   *LinuxAppShellReport            `json:"linux_app_shell,omitempty"`
	SecurityPermissions             *SecurityPermissionReport       `json:"security_permissions,omitempty"`
	SurfacePerformanceBudget        *SurfacePerformanceBudgetReport `json:"surface_performance_budget,omitempty"`
	BrowserSurface                  *BrowserSurfaceReport           `json:"browser_surface,omitempty"`
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
	AppModel                string   `json:"app_model"`
	LinuxAppShell           string   `json:"linux_app_shell"`
	AppShellFeatures        string   `json:"app_shell_features"`
	SecurityPermissions     string   `json:"security_permissions"`
	PerformanceBudget       string   `json:"performance_budget"`
	DeveloperFastLoop       string   `json:"developer_fast_loop"`
	Inspector               string   `json:"inspector"`
	ProjectTemplates        string   `json:"project_templates"`
	ReferenceApps           string   `json:"reference_apps"`
	SurfacePackage          string   `json:"surface_package"`
	CrashReporting          string   `json:"crash_reporting"`
	I18nLocalization        string   `json:"i18n_localization"`
	WidgetMigration         string   `json:"widget_migration"`
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
	Order                   int    `json:"order"`
	Width                   int    `json:"width"`
	Height                  int    `json:"height"`
	Stride                  int    `json:"stride"`
	Checksum                string `json:"checksum"`
	ArtifactPath            string `json:"artifact_path,omitempty"`
	Producer                string `json:"producer,omitempty"`
	EvidenceRole            string `json:"evidence_role,omitempty"`
	AppSource               string `json:"app_source,omitempty"`
	MorphRecipeHash         string `json:"morph_recipe_hash,omitempty"`
	BlockSceneHash          string `json:"block_scene_hash,omitempty"`
	RenderCommandStreamHash string `json:"render_command_stream_hash,omitempty"`
	Precomputed             bool   `json:"precomputed,omitempty"`
	Presented               bool   `json:"presented"`
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
