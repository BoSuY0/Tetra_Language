package surface

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ---- claims.go ----

type ClaimTier string

const (
	ClaimTierProdStableScoped ClaimTier = "PROD_STABLE_SCOPED"
	ClaimTierBetaTargetHost   ClaimTier = "BETA_TARGET_HOST"
	ClaimTierExperimental     ClaimTier = "EXPERIMENTAL"
	ClaimTierUnsupported      ClaimTier = "UNSUPPORTED"
	ClaimTierNonClaim         ClaimTier = "NONCLAIM"
)

var surfaceClaimTiers = []ClaimTier{
	ClaimTierProdStableScoped,
	ClaimTierBetaTargetHost,
	ClaimTierExperimental,
	ClaimTierUnsupported,
	ClaimTierNonClaim,
}

func SurfaceClaimTiers() []ClaimTier {
	return append([]ClaimTier(nil), surfaceClaimTiers...)
}

func ValidSurfaceClaimTier(value string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	for _, tier := range surfaceClaimTiers {
		if normalized == string(tier) {
			return true
		}
	}
	return false
}

// ---- common_validation.go ----

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
		if transition.Component == component && transition.Field == field &&
			transition.Before != transition.After {
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

func hasProcessNameAndPathMarkers(
	processes []ProcessReport,
	kind string,
	nameMarker string,
	pathMarkers ...string,
) bool {
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
		if process.ExitCode != nil && process.ExpectedExitCode != nil &&
			*process.ExitCode == exitCode &&
			*process.ExpectedExitCode == exitCode {
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
		if frame.Width == width && frame.Height == height && frame.Stride == stride &&
			frame.Presented &&
			strings.TrimSpace(frame.Checksum) != "" {
			return true
		}
	}
	return false
}

func hasFrameOrderDimensions(
	frames []FrameReport,
	order int,
	width int,
	height int,
	stride int,
) bool {
	for _, frame := range frames {
		if frame.Order == order && frame.Width == width && frame.Height == height &&
			frame.Stride == stride &&
			frame.Presented &&
			strings.TrimSpace(frame.Checksum) != "" {
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
		return []string{
			fmt.Sprintf(
				"source %q must be a Tetra source path with .tetra or .t4 extension",
				source,
			),
		}
	}

	var issues []string
	matched := false
	allowToolkitWidgets := isSurfaceToolkitFormSource(source) ||
		isSurfaceToolkitSettingsSource(source) ||
		isSurfaceReleaseFormSource(source) ||
		isSurfaceAccessibilitySettingsSource(source) ||
		isSurfaceReleaseAccessibilitySource(source)
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
		issues = append(
			issues,
			fmt.Sprintf(
				"component type %q does not match source module %q",
				componentType,
				sourceModule,
			),
		)
	}
	if len(components) > 0 && !matched {
		issues = append(
			issues,
			fmt.Sprintf(
				"source module %q is not represented by component type evidence",
				sourceModule,
			),
		)
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
			issues = append(
				issues,
				fmt.Sprintf("component %s missing measure ability evidence", component.ID),
			)
		}
		if !contains(component.Abilities, "layout") {
			issues = append(
				issues,
				fmt.Sprintf("component %s missing layout ability evidence", component.ID),
			)
		}
		if !contains(component.Abilities, "draw") {
			issues = append(
				issues,
				fmt.Sprintf("component %s missing draw ability evidence", component.ID),
			)
		}
		if !contains(component.Abilities, "event") {
			issues = append(
				issues,
				fmt.Sprintf("component %s missing event ability evidence", component.ID),
			)
		}
		if !contains(component.Abilities, "focus") {
			issues = append(
				issues,
				fmt.Sprintf("component %s missing focus ability evidence", component.ID),
			)
		}
		if !contains(component.Abilities, "text") {
			issues = append(
				issues,
				fmt.Sprintf("component %s missing text ability evidence", component.ID),
			)
		}
		if !contains(component.Abilities, "accessibility") {
			issues = append(
				issues,
				fmt.Sprintf("component %s missing accessibility ability evidence", component.ID),
			)
		}
		if component.Bounds.W <= 0 || component.Bounds.H <= 0 {
			issues = append(
				issues,
				fmt.Sprintf("component %s layout bounds are required", component.ID),
			)
		}
		if len(component.State) == 0 {
			issues = append(
				issues,
				fmt.Sprintf("component %s state evidence is required", component.ID),
			)
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
			issues = append(
				issues,
				fmt.Sprintf(
					"component %s parent %s is not in component evidence",
					component.ID,
					parent,
				),
			)
			continue
		}
		if !rectContainsRect(index[parent].Bounds, component.Bounds) {
			issues = append(
				issues,
				fmt.Sprintf(
					"component %s layout bounds must be inside parent %s bounds",
					component.ID,
					parent,
				),
			)
		}
	}
	if len(components) < 2 || !seenChild {
		issues = append(issues, "component hierarchy evidence is required")
	}
	return index, issues
}

func validateEvents(report Report, components map[string]ComponentReport) []string {
	events := report.Events
	textInputRequired := report.HostEvidence.Level != NativeSurfaceHostLevelLinuxX64
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
			issues = append(
				issues,
				fmt.Sprintf(
					"event order %d is not strictly greater than previous order %d",
					event.Order,
					lastOrder,
				),
			)
		}
		lastOrder = event.Order
		if strings.TrimSpace(event.Kind) == "" {
			issues = append(issues, fmt.Sprintf("event %d kind is required", event.Order))
		}
		if strings.TrimSpace(event.TargetComponent) == "" {
			issues = append(
				issues,
				fmt.Sprintf("event %d target_component is required", event.Order),
			)
		} else if _, ok := components[event.TargetComponent]; !ok {
			issues = append(
				issues,
				fmt.Sprintf(
					"event %d target_component %s is not in component evidence",
					event.Order,
					event.TargetComponent,
				),
			)
		}
		if !validateDispatchPath(event, components, &issues) {
			continue
		}
		if event.Handled && isPointerEvent(event.Kind) {
			target := components[event.TargetComponent]
			if !rectContainsPoint(target.Bounds, event.X, event.Y) {
				issues = append(
					issues,
					fmt.Sprintf(
						"event %d pointer dispatch point %d,%d is outside target bounds for %s",
						event.Order,
						event.X,
						event.Y,
						event.TargetComponent,
					),
				)
			}
		}
		if !event.Pass {
			issues = append(issues, fmt.Sprintf("event %d did not pass", event.Order))
		}
		if len(event.BeforeState) == 0 || len(event.AfterState) == 0 {
			issues = append(
				issues,
				fmt.Sprintf("event %d must include before_state and after_state", event.Order),
			)
		}
		if event.Handled && stateChanged(event.BeforeState, event.AfterState) {
			handledStateChange = true
		}
		if event.Handled {
			if component, ok := components[event.TargetComponent]; ok &&
				strings.TrimSpace(component.Parent) != "" {
				handledChildDispatch = true
			}
			if validateEventBuffer(event, &issues) {
				handledEventBuffer = true
				if event.Kind == "mouse_up" && event.BufferSlots[0] == 5 &&
					event.BufferSlots[7] == 0 {
					pointerBufferOrder = event.Order
				}
				if event.Kind == "text_input" && pointerBufferOrder > 0 &&
					event.Order > pointerBufferOrder &&
					event.BufferSlots[0] == 8 &&
					event.BufferSlots[7] > 0 {
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
	if !handledEventBufferSequence && textInputRequired {
		issues = append(issues, "event evidence missing host event buffer pointer/text sequence")
	}
	if !handledTextInput && textInputRequired {
		issues = append(issues, "event evidence missing handled text_input scalar dispatch")
	}
	if !handledTextPayload && textInputRequired {
		issues = append(issues, "event evidence missing host text payload buffer")
	}
	return issues
}

func validateDispatchPath(
	event EventReport,
	components map[string]ComponentReport,
	issues *[]string,
) bool {
	if len(event.DispatchPath) == 0 {
		*issues = append(*issues, fmt.Sprintf("event %d dispatch_path is required", event.Order))
		return false
	}
	for _, id := range event.DispatchPath {
		if strings.TrimSpace(id) == "" {
			*issues = append(
				*issues,
				fmt.Sprintf("event %d dispatch_path contains an empty component id", event.Order),
			)
			return false
		}
		if _, ok := components[id]; !ok {
			*issues = append(
				*issues,
				fmt.Sprintf(
					"event %d dispatch_path component %s is not in component evidence",
					event.Order,
					id,
				),
			)
			return false
		}
	}
	if event.DispatchPath[len(event.DispatchPath)-1] != event.TargetComponent {
		*issues = append(
			*issues,
			fmt.Sprintf(
				"event %d dispatch_path ends at %s, want target_component %s",
				event.Order,
				event.DispatchPath[len(event.DispatchPath)-1],
				event.TargetComponent,
			),
		)
		return false
	}
	want, ok := componentPathToRoot(event.TargetComponent, components)
	if !ok {
		*issues = append(
			*issues,
			fmt.Sprintf(
				"event %d dispatch_path cannot resolve parent chain for %s",
				event.Order,
				event.TargetComponent,
			),
		)
		return false
	}
	if !stringSlicesEqual(event.DispatchPath, want) {
		*issues = append(
			*issues,
			fmt.Sprintf(
				"event %d dispatch_path = %v, want parent chain %v",
				event.Order,
				event.DispatchPath,
				want,
			),
		)
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
	return child.X >= parent.X && child.Y >= parent.Y && child.X+child.W <= parent.X+parent.W &&
		child.Y+child.H <= parent.Y+parent.H
}

func validateEventBuffer(event EventReport, issues *[]string) bool {
	if len(event.BufferSlots) == 0 {
		return false
	}
	if len(event.BufferSlots) < 9 {
		*issues = append(
			*issues,
			fmt.Sprintf(
				"event %d event buffer has %d slots, want at least 9",
				event.Order,
				len(event.BufferSlots),
			),
		)
		return false
	}
	if event.BufferSlots[1] != event.X || event.BufferSlots[2] != event.Y {
		*issues = append(
			*issues,
			fmt.Sprintf(
				"event %d event buffer coordinates = %d,%d, want %d,%d",
				event.Order,
				event.BufferSlots[1],
				event.BufferSlots[2],
				event.X,
				event.Y,
			),
		)
		return false
	}
	if event.BufferSlots[4] != event.Key || event.BufferSlots[5] != event.Width ||
		event.BufferSlots[6] != event.Height ||
		event.BufferSlots[7] != event.TimestampMS ||
		event.BufferSlots[8] != event.TextLen {
		*issues = append(
			*issues,
			fmt.Sprintf(
				"event %d event buffer record = %v, want key/width/height/timestamp/text_len = %d/%d/%d/%d/%d",
				event.Order,
				event.BufferSlots,
				event.Key,
				event.Width,
				event.Height,
				event.TimestampMS,
				event.TextLen,
			),
		)
		return false
	}
	if event.Kind == "mouse_up" && (event.BufferSlots[0] != 5 || event.BufferSlots[3] != 1) {
		*issues = append(
			*issues,
			fmt.Sprintf(
				"event %d mouse_up event buffer slots = %v, want kind 5 and button 1",
				event.Order,
				event.BufferSlots,
			),
		)
		return false
	}
	if event.Kind == "text_input" && event.BufferSlots[0] != 8 {
		*issues = append(
			*issues,
			fmt.Sprintf(
				"event %d text_input event buffer slots = %v, want kind 8",
				event.Order,
				event.BufferSlots,
			),
		)
		return false
	}
	return true
}

func validateTextPayloadEvent(event EventReport, issues *[]string) bool {
	if event.TextLen <= 0 {
		*issues = append(
			*issues,
			fmt.Sprintf("event %d text payload length must be positive", event.Order),
		)
		return false
	}
	if strings.TrimSpace(event.TextBytesHex) == "" {
		*issues = append(
			*issues,
			fmt.Sprintf("event %d text payload bytes are required", event.Order),
		)
		return false
	}
	payload, err := hex.DecodeString(event.TextBytesHex)
	if err != nil {
		*issues = append(
			*issues,
			fmt.Sprintf("event %d text payload bytes are not valid hex", event.Order),
		)
		return false
	}
	if len(payload) != event.TextLen {
		*issues = append(
			*issues,
			fmt.Sprintf(
				"event %d text payload length = %d, want %d bytes",
				event.Order,
				event.TextLen,
				len(payload),
			),
		)
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
			issues = append(
				issues,
				fmt.Sprintf(
					"frame order %d is not strictly greater than previous order %d",
					frame.Order,
					lastOrder,
				),
			)
		}
		lastOrder = frame.Order
		if frame.Width <= 0 || frame.Height <= 0 || frame.Stride <= 0 {
			issues = append(
				issues,
				fmt.Sprintf("frame %d dimensions and stride must be positive", frame.Order),
			)
		}
		if strings.TrimSpace(frame.Checksum) == "" {
			issues = append(issues, fmt.Sprintf("frame %d checksum is required", frame.Order))
		} else if !strings.HasPrefix(frame.Checksum, "sha256:") && len(frame.Checksum) != 64 {
			issues = append(
				issues,
				fmt.Sprintf("frame %d checksum must be sha256 hex or sha256:<hex>", frame.Order),
			)
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

func validateStateTransitions(
	transitions []StateTransitionReport,
	components map[string]ComponentReport,
) []string {
	var issues []string
	if len(transitions) == 0 {
		issues = append(issues, "state transition evidence is required")
	}
	lastOrder := 0
	for _, transition := range transitions {
		if transition.Order <= lastOrder {
			issues = append(
				issues,
				fmt.Sprintf(
					"state transition order %d is not strictly greater than previous order %d",
					transition.Order,
					lastOrder,
				),
			)
		}
		lastOrder = transition.Order
		if strings.TrimSpace(transition.Component) == "" {
			issues = append(issues, "state transition component is required")
		} else if _, ok := components[transition.Component]; !ok {
			issues = append(
				issues,
				fmt.Sprintf("state transition component %s is not in component evidence", transition.Component),
			)
		}
		if strings.TrimSpace(transition.Field) == "" {
			issues = append(
				issues,
				fmt.Sprintf("state transition %d field is required", transition.Order),
			)
		}
		if transition.Before == transition.After {
			issues = append(
				issues,
				fmt.Sprintf("state transition %d must change value", transition.Order),
			)
		}
		if strings.TrimSpace(transition.Cause) == "" {
			issues = append(
				issues,
				fmt.Sprintf("state transition %d cause is required", transition.Order),
			)
		}
	}
	return issues
}

func validateCases(report Report) []string {
	cases := report.Cases
	textInputRequired := report.HostEvidence.Level != NativeSurfaceHostLevelLinuxX64
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
				issues = append(
					issues,
					fmt.Sprintf("negative case %s expected_error is required", tc.Name),
				)
			}
		default:
			issues = append(
				issues,
				fmt.Sprintf("case %s kind is %q, want positive or negative", tc.Name, tc.Kind),
			)
		}
		if !tc.Ran {
			issues = append(issues, fmt.Sprintf("case %s did not run", tc.Name))
		}
		if !tc.Pass {
			issues = append(issues, fmt.Sprintf("case %s did not pass", tc.Name))
		}
		if strings.Contains(strings.ToLower(tc.Name), "no legacy ui sidecar artifacts") &&
			tc.Kind == "positive" &&
			tc.Ran &&
			tc.Pass {
			seenNoLegacyUISidecars = true
		}
		if strings.Contains(strings.ToLower(tc.Name), "host-provided pointer event dispatch") &&
			tc.Kind == "positive" &&
			tc.Ran &&
			tc.Pass {
			seenHostProvidedPointerEvent = true
		}
		if strings.Contains(strings.ToLower(tc.Name), "host event buffer poll_event") &&
			tc.Kind == "positive" &&
			tc.Ran &&
			tc.Pass {
			seenHostEventBufferPoll = true
		}
		if strings.Contains(strings.ToLower(tc.Name), "pre/post event frame sequence") &&
			tc.Kind == "positive" &&
			tc.Ran &&
			tc.Pass {
			seenPrePostEventFrameSequence = true
		}
		if strings.Contains(strings.ToLower(tc.Name), "component hierarchy dispatch") &&
			tc.Kind == "positive" &&
			tc.Ran &&
			tc.Pass {
			seenComponentHierarchyDispatch = true
		}
		if strings.Contains(strings.ToLower(tc.Name), "component text input scalar dispatch") &&
			tc.Kind == "positive" &&
			tc.Ran &&
			tc.Pass {
			seenComponentTextInputScalarDispatch = true
		}
		if strings.Contains(strings.ToLower(tc.Name), "host text payload buffer") &&
			tc.Kind == "positive" &&
			tc.Ran &&
			tc.Pass {
			seenHostTextPayloadBuffer = true
		}
		if strings.Contains(strings.ToLower(tc.Name), "component focus dispatch") &&
			tc.Kind == "positive" &&
			tc.Ran &&
			tc.Pass {
			seenComponentFocusDispatch = true
		}
		if strings.Contains(strings.ToLower(tc.Name), "component accessibility metadata") &&
			tc.Kind == "positive" &&
			tc.Ran &&
			tc.Pass {
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
	if !seenComponentTextInputScalarDispatch && textInputRequired {
		issues = append(issues, "case evidence missing component text input scalar dispatch case")
	}
	if !seenHostTextPayloadBuffer && textInputRequired {
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

// ---- report.go ----

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
	if report.Target != "headless" && report.Target != "linux-x64" &&
		report.Target != "wasm32-web" {
		issues = append(
			issues,
			fmt.Sprintf("target is %q, want headless, linux-x64, or wasm32-web", report.Target),
		)
	}
	if report.Host != "linux-x64" {
		issues = append(issues, fmt.Sprintf("host is %q, want linux-x64", report.Host))
	}
	if report.Runtime != "surface-headless" && report.Runtime != "surface-linux-x64" &&
		report.Runtime != "surface-wasm32-web" {
		issues = append(
			issues,
			fmt.Sprintf(
				"runtime is %q, want surface-headless, surface-linux-x64, or surface-wasm32-web",
				report.Runtime,
			),
		)
	}
	if report.SurfaceSchema != "tetra.surface.v1" {
		issues = append(
			issues,
			fmt.Sprintf("surface_schema is %q, want tetra.surface.v1", report.SurfaceSchema),
		)
	}
	if report.HostABI != "tetra.surface.host-abi.v1" {
		issues = append(
			issues,
			fmt.Sprintf("host_abi is %q, want tetra.surface.host-abi.v1", report.HostABI),
		)
	}
	issues = append(issues, validateHostEvidence(report)...)
	issues = append(issues, validateNativeSurfaceHostEvidence(report)...)
	if strings.TrimSpace(report.Source) == "" {
		issues = append(issues, "source is required")
	}
	issues = append(issues, validateProcesses(report.Source, report.Processes)...)
	issues = append(
		issues,
		validateArtifacts(report.Target, report.Source, report.Artifacts, report.Processes)...)
	issues = append(issues, validateArtifactScan(report.ArtifactScan, report.Artifacts)...)
	componentIndex, componentIssues := validateComponents(report.Components)
	issues = append(issues, componentIssues...)
	issues = append(issues, validateSourceComponentModel(report.Source, report.Components)...)
	issues = append(issues, validateEvents(report, componentIndex)...)
	issues = append(issues, validateFrames(report.Frames)...)
	issues = append(issues, validateFrameProvenance(report)...)
	issues = append(issues, validateStateTransitions(report.StateTransitions, componentIndex)...)
	issues = append(issues, validateCases(report)...)
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
		issues = append(
			issues,
			"surface_performance_budget memory peak_rss_bytes field is required",
		)
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

// ---- report_types.go ----

type SurfacePerfBudgetReport = SurfacePerformanceBudgetReport
type BlockAssetRenderCmdReport = BlockAssetRenderCommandReport

type Report struct {
	Schema        string             `json:"schema"`
	Status        string             `json:"status"`
	Target        string             `json:"target"`
	Host          string             `json:"host"`
	Runtime       string             `json:"runtime"`
	SurfaceSchema string             `json:"surface_schema"`
	HostABI       string             `json:"host_abi"`
	HostEvidence  HostEvidenceReport `json:"host_evidence"`
	Source        string             `json:"source"`

	NativeSurfaceHost *NativeSurfaceHostReport `json:"native_surface_host,omitempty"`
	Processes         []ProcessReport          `json:"processes"`
	Artifacts         []ArtifactReport         `json:"artifacts"`
	ArtifactScan      ArtifactScanReport       `json:"artifact_scan"`
	Components        []ComponentReport        `json:"components"`
	ComponentTree     *ComponentTreeReport     `json:"component_tree,omitempty"`
	ComponentTreeAPI  *ComponentTreeAPIReport  `json:"component_tree_api,omitempty"`

	BlockGraph          *BlockGraphReport          `json:"block_graph,omitempty"`
	BlockSceneSnapshot  *BlockSceneSnapshotReport  `json:"block_scene_snapshot,omitempty"`
	RenderCommandStream *RenderCommandStreamReport `json:"render_command_stream,omitempty"`

	PaintLayers           []PaintLayerReport   `json:"paint_layers,omitempty"`
	PaintCommands         []PaintCommandReport `json:"paint_commands,omitempty"`
	VisualFeatures        []string             `json:"visual_features,omitempty"`
	PaintQualityLevel     string               `json:"paint_quality_level,omitempty"`
	PaintCacheBudgetBytes int                  `json:"paint_cache_budget_bytes,omitempty"`
	PaintUnsupportedBlur  bool                 `json:"paint_unsupported_blur,omitempty"`
	Renderer              *RendererReport      `json:"renderer,omitempty"`

	TextMeasurements     []TextMeasurementReport   `json:"text_measurements,omitempty"`
	FontFallbacks        []FontFallbackReport      `json:"font_fallbacks,omitempty"`
	GlyphCaches          []GlyphCacheReport        `json:"glyph_caches,omitempty"`
	TextRenderCommands   []TextRenderCommandReport `json:"text_render_commands,omitempty"`
	TextQualityLevel     string                    `json:"text_quality_level,omitempty"`
	TextCacheBudgetBytes int                       `json:"text_cache_budget_bytes,omitempty"`

	LayoutConstraints  []BlockLayoutConstraintReport `json:"layout_constraints,omitempty"`
	LayoutPasses       []BlockLayoutPassReport       `json:"layout_passes,omitempty"`
	LayoutScrolls      []BlockLayoutScrollReport     `json:"layout_scrolls,omitempty"`
	LayoutDensity      *BlockLayoutDensityReport     `json:"layout_density,omitempty"`
	LayoutFeatures     []string                      `json:"layout_features,omitempty"`
	LayoutQualityLevel string                        `json:"layout_quality_level,omitempty"`

	LayoutUnsupportedCSSFlexbox bool `json:"layout_unsupported_css_flexbox,omitempty"`

	BlockEventRoutes []BlockEventRouteReport `json:"block_event_routes,omitempty"`

	BlockFocusTransitions []BlockFocusTransitionReport `json:"block_focus_transitions,omitempty"`

	BlockEventKinds  []string `json:"block_event_kinds,omitempty"`
	BlockEventPolicy string   `json:"block_event_policy,omitempty"`

	BlockEventQualityLevel string `json:"block_event_quality_level,omitempty"`

	BlockEventUnsupportedDragDrop bool `json:"block_event_unsupported_drag_drop,omitempty"`

	BlockStateSelectors []BlockStateSelectorReport `json:"block_state_selectors,omitempty"`

	BlockStateResolutions []BlockStateResolutionReport `json:"block_state_resolutions,omitempty"`

	BlockStateResolverOrder []string `json:"block_state_resolver_order,omitempty"`

	BlockStateQualityLevel string `json:"block_state_quality_level,omitempty"`

	BlockStateUnsupportedCSSPseudos bool `json:"block_state_unsupported_css_pseudos,omitempty"`

	MotionFrames       []MotionFrameReport `json:"motion_frames,omitempty"`
	MotionQualityLevel string              `json:"motion_quality_level,omitempty"`
	MotionClock        string              `json:"motion_clock,omitempty"`
	MotionFrameBudget  int                 `json:"motion_frame_budget,omitempty"`

	MotionUnsupportedCSSAnimations bool `json:"motion_unsupported_css_animations,omitempty"`

	BlockAssetManifest *BlockAssetManifestReport `json:"block_asset_manifest,omitempty"`

	BlockAssetCache BlockAssetCacheReport `json:"block_asset_cache,omitempty"`

	BlockAssetDiagnostics []BlockAssetDiagnosticReport `json:"block_asset_diagnostics,omitempty"`

	BlockAssetRenderCommands []BlockAssetRenderCmdReport `json:"block_asset_render_commands,omitempty"`

	BlockAssetQualityLevel string `json:"block_asset_quality_level,omitempty"`

	BlockAssetNetworkFetchAllowed bool `json:"block_asset_network_fetch_allowed,omitempty"`

	BlockAccessibilityTree *BlockAccessibilityTreeReport `json:"block_accessibility_tree,omitempty"`
	BlockSystem            *BlockSystemReport            `json:"block_system,omitempty"`
	Morph                  *MorphReport                  `json:"morph,omitempty"`
	Toolkit                *ToolkitReport                `json:"toolkit,omitempty"`
	AccessibilityTree      *AccessibilityTreeReport      `json:"accessibility_tree,omitempty"`
	AppModel               *AppModelReport               `json:"app_model,omitempty"`
	LinuxAppShell          *LinuxAppShellReport          `json:"linux_app_shell,omitempty"`
	SecurityPermissions    *SecurityPermissionReport     `json:"security_permissions,omitempty"`
	BrowserSurface         *BrowserSurfaceReport         `json:"browser_surface,omitempty"`

	SurfacePerformanceBudget *SurfacePerfBudgetReport `json:"surface_performance_budget,omitempty"`

	Events           []EventReport           `json:"events"`
	Frames           []FrameReport           `json:"frames"`
	StateTransitions []StateTransitionReport `json:"state_transitions"`
	Cases            []CaseReport            `json:"cases"`
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

type NativeSurfaceHostReport struct {
	Schema                 string `json:"schema"`
	Host                   string `json:"host"`
	Protocol               string `json:"protocol"`
	AppProcessKind         string `json:"app_process_kind"`
	HostProcessKind        string `json:"host_process_kind"`
	AppPID                 int    `json:"app_pid"`
	HostPID                int    `json:"host_pid"`
	SurfaceOpenFromApp     bool   `json:"surface_open_from_app"`
	PollEventFromHost      bool   `json:"poll_event_from_host"`
	PresentFromAppRGBA     bool   `json:"present_from_app_rgba"`
	AppLoopObserved        bool   `json:"app_loop_observed"`
	RealWindow             bool   `json:"real_window"`
	RealCloseEvent         bool   `json:"real_close_event"`
	RealPointerEventCount  int    `json:"real_pointer_event_count"`
	RealKeyEventCount      int    `json:"real_key_event_count"`
	PresentedFrameCount    int    `json:"presented_frame_count"`
	PreRenderedFrameSource bool   `json:"pre_rendered_frame_source"`
	DeliveryPath           string `json:"delivery_path"`
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

// ---- schemas.go ----

const (
	SchemaV1                        = "tetra.surface.runtime.v1"
	ReleaseSchemaV1                 = "tetra.surface.release.v1"
	RendererFeatureSchemaV1         = "tetra.surface.renderer-feature.v1"
	TextInputSchemaV1               = "tetra.surface.text-input.v1"
	LinuxAppShellSchemaV1           = "tetra.surface.linux-app-shell.v1"
	BrowserSurfaceSchemaV1          = "tetra.surface.browser-surface.v1"
	SecurityPermissionSchemaV1      = "tetra.surface.security-permission.v1"
	PerformanceBudgetSchemaV1       = "tetra.surface.performance-budget.v1"
	TargetHostStatusSchemaV1        = "tetra.surface.target-host-status.v1"
	ReleaseScopeSurfaceV1LinuxWeb   = "surface-v1-linux-web"
	NativeSurfaceHostSchemaV1       = "tetra.surface.native-host.v1"
	NativeSurfaceHostLevelLinuxX64  = "linux-x64-native-surface-host-v1"
	NativeSurfaceHostBackendWayland = "wayland-surface-host-v1"
	NativeSurfaceHostProtocolV1     = "tetra.surface.host-ipc.v1"
)

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

// ---- release_summary.go ----

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
		issues = append(
			issues,
			fmt.Sprintf("schema is %q, want %q", report.Schema, ReleaseSchemaV1),
		)
	}
	if report.ReleaseScope != ReleaseScopeSurfaceV1LinuxWeb {
		issues = append(
			issues,
			fmt.Sprintf(
				"release_scope is %q, want %q",
				report.ReleaseScope,
				ReleaseScopeSurfaceV1LinuxWeb,
			),
		)
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
		issues = append(
			issues,
			fmt.Sprintf(
				"producer is %q, want scripts/release/surface/release-gate.sh",
				report.Producer,
			),
		)
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
	if strings.TrimSpace(report.GeneratedAtUTC) == "" ||
		!strings.HasSuffix(report.GeneratedAtUTC, "Z") ||
		!strings.Contains(report.GeneratedAtUTC, "T") {
		issues = append(issues, "generated_at_utc must be an RFC3339 UTC timestamp")
	}
	if !strings.Contains(report.CommandLine, "scripts/release/surface/release-gate.sh") {
		issues = append(issues, "command_line must include scripts/release/surface/release-gate.sh")
	}
	issues = append(
		issues,
		validateExactStringList(
			"supported_targets",
			report.SupportedTargets,
			[]string{"headless", "linux-x64", "wasm32-web"},
		)...)
	issues = append(
		issues,
		validateExactStringList(
			"runtime_targets",
			report.RuntimeTargets,
			[]string{"linux-x64", "wasm32-web"},
		)...)
	issues = append(
		issues,
		validateExactStringList("test_targets", report.TestTargets, []string{"headless"})...)
	issues = append(
		issues,
		validateExactStringList(
			"unsupported_targets",
			report.UnsupportedTargets,
			[]string{"macos-x64", "windows-x64", "wasm32-wasi"},
		)...)
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
		{field: "app_model", got: report.AppModel, want: "explicit-command-reducer-v1"},
		{field: "linux_app_shell", got: report.LinuxAppShell, want: "linux-app-shell-subset-v1"},
		{field: "app_shell_features", got: report.AppShellFeatures, want: "electron-feature-ledger-v1"},
		{
			field: "security_permissions",
			got:   report.SecurityPermissions,
			want:  "surface-security-permission-v1",
		},
		{
			field: "performance_budget",
			got:   report.PerformanceBudget,
			want:  "surface-performance-budget-v1",
		},
		{field: "developer_fast_loop", got: report.DeveloperFastLoop, want: "surface-dev-workflow-v1"},
		{field: "inspector", got: report.Inspector, want: "surface-inspector-v1"},
		{field: "project_templates", got: report.ProjectTemplates, want: "surface-template-smoke-v1"},
		{field: "reference_apps", got: report.ReferenceApps, want: "surface-reference-app-suite-v1"},
		{field: "surface_package", got: report.SurfacePackage, want: "surface-package-v1"},
		{field: "crash_reporting", got: report.CrashReporting, want: "surface-crash-report-v1"},
		{field: "i18n_localization", got: report.I18nLocalization, want: "surface-i18n-v1"},
		{field: "widget_migration", got: report.WidgetMigration, want: "surface-widget-migration-v1"},
		{field: "browser_surface", got: report.BrowserSurface, want: "browser-canvas-release-v1"},
		{field: "linux_surface", got: report.LinuxSurface, want: "linux-x64-release-window-v1"},
		{field: "block_system", got: report.BlockSystem, want: "block-system"},
		{
			field: "block_system_gate",
			got:   report.BlockSystemGate,
			want:  "tetra.surface.block-system.gate.v1",
		},
		{field: "morph", got: report.Morph, want: "morph-capsule"},
		{field: "morph_gate", got: report.MorphGate, want: "tetra.surface.morph.gate.v1"},
	} {
		if check.got != check.want {
			issues = append(
				issues,
				fmt.Sprintf("%s is %q, want %q", check.field, check.got, check.want),
			)
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
