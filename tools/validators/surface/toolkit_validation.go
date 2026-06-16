package surface

import (
	"fmt"
	"strings"
)

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

func isSurfaceToolkitFormSource(source string) bool {
	return strings.HasSuffix(normalizeEvidencePath(source), "examples/surface_toolkit_form.tetra")
}

func isSurfaceToolkitSettingsSource(source string) bool {
	return strings.HasSuffix(normalizeEvidencePath(source), "examples/surface_toolkit_settings.tetra")
}

func isSurfaceReleaseFormSource(source string) bool {
	return strings.HasSuffix(normalizeEvidencePath(source), "examples/surface_release_form.tetra")
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
