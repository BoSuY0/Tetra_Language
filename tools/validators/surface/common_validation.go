package surface

import (
	"encoding/hex"
	"fmt"
	"strings"
)

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

func stringSliceContainsFold(values []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == want {
			return true
		}
	}
	return false
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
