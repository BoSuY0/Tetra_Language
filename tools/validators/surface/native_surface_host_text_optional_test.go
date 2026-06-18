package surface

import (
	"strings"
	"testing"
)

func TestValidateReportAcceptsNativeSurfaceHostWithoutTextInputEvidence(t *testing.T) {
	raw := validLinuxX64NativeSurfaceHostReportJSON(t, func(report map[string]any) {
		var events []any
		for _, item := range report["events"].([]any) {
			event := item.(map[string]any)
			if event["kind"] != "text_input" {
				events = append(events, event)
			}
		}
		report["events"] = events

		var transitions []any
		for _, item := range report["state_transitions"].([]any) {
			transition := item.(map[string]any)
			if transition["cause"] != "text_input" {
				transitions = append(transitions, transition)
			}
		}
		report["state_transitions"] = transitions

		var cases []any
		for _, item := range report["cases"].([]any) {
			tc := item.(map[string]any)
			name := strings.ToLower(tc["name"].(string))
			if strings.Contains(name, "component text input scalar dispatch") ||
				strings.Contains(name, "host text payload buffer") {
				continue
			}
			cases = append(cases, tc)
		}
		report["cases"] = cases
	})
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed for native host without text input evidence: %v\n%s", err, raw)
	}
}

func TestValidateReportAcceptsNativeSurfaceLiveFrameRole(t *testing.T) {
	raw := validLinuxX64NativeSurfaceHostReportJSON(t, func(report map[string]any) {
		for _, item := range report["frames"].([]any) {
			frame := item.(map[string]any)
			frame["evidence_role"] = "native-surface-live-frame"
		}
	})
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed for native live frame role: %v\n%s", err, raw)
	}
}
