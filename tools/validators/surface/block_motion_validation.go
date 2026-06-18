package surface

import (
	"fmt"
	"strings"
)

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
	if isWASM32WebBrowserCanvasMorphRuntimeReport(report) || isLinuxX64RealWindowMorphRuntimeReport(report) {
		if len(report.Frames) < 2 || strings.TrimSpace(report.Frames[0].Checksum) == "" || report.Frames[0].Checksum == report.Frames[1].Checksum {
			issues = append(issues, "block motion frame checksum evidence must show motion-driven visual changes")
		}
	} else if len(report.Frames) < 3 || strings.TrimSpace(report.Frames[0].Checksum) == "" || report.Frames[0].Checksum == report.Frames[1].Checksum || report.Frames[1].Checksum == report.Frames[2].Checksum {
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
