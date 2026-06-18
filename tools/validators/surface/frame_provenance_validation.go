package surface

import (
	"fmt"
	"strings"
)

func validateFrameProvenance(report Report) []string {
	var issues []string
	for i, frame := range report.Frames {
		role := normalizeFrameEvidenceToken(frame.EvidenceRole)
		producer := normalizeFrameEvidenceToken(frame.Producer)
		if role != "" && !isKnownFrameEvidenceRole(role) {
			issues = append(issues, fmt.Sprintf("frame %d evidence_role %q is not supported", frame.Order, frame.EvidenceRole))
		}
		if frame.Precomputed && !isHostProbeOnlyFrameRole(role) {
			issues = append(issues, fmt.Sprintf("frame %d precomputed evidence can only be host_probe_only infrastructure evidence", frame.Order))
		}
		if !isProductVisualFrameRole(role) {
			continue
		}
		location := fmt.Sprintf("frames[%d]", i)
		if frame.Precomputed {
			issues = append(issues, fmt.Sprintf("%s product visual frame %d must not be precomputed", location, frame.Order))
		}
		if producer != "app" {
			issues = append(issues, fmt.Sprintf("%s product visual frame %d producer is %q, want app", location, frame.Order, frame.Producer))
		}
		if strings.TrimSpace(frame.AppSource) == "" {
			issues = append(issues, fmt.Sprintf("%s product visual frame %d app_source is required", location, frame.Order))
		} else if normalizeEvidencePath(frame.AppSource) != normalizeEvidencePath(report.Source) {
			issues = append(issues, fmt.Sprintf("%s product visual frame %d app_source %q must match report source %q", location, frame.Order, frame.AppSource, report.Source))
		}
		if !validSHA256Digest(frame.MorphRecipeHash) {
			issues = append(issues, fmt.Sprintf("%s product visual frame %d morph_recipe_hash must be sha256 evidence", location, frame.Order))
		}
		if !validSHA256Digest(frame.BlockSceneHash) {
			issues = append(issues, fmt.Sprintf("%s product visual frame %d block_scene_hash must be sha256 evidence", location, frame.Order))
		} else if report.BlockSceneSnapshot == nil {
			issues = append(issues, fmt.Sprintf("%s product visual frame %d requires block_scene_snapshot evidence", location, frame.Order))
		} else if frame.BlockSceneHash != report.BlockSceneSnapshot.BlockSceneHash {
			issues = append(issues, fmt.Sprintf("%s product visual frame %d block_scene_hash must match block_scene_snapshot.block_scene_hash", location, frame.Order))
		}
		if !validSHA256Digest(frame.RenderCommandStreamHash) {
			issues = append(issues, fmt.Sprintf("%s product visual frame %d render_command_stream_hash must be sha256 evidence", location, frame.Order))
		} else if report.RenderCommandStream == nil {
			issues = append(issues, fmt.Sprintf("%s product visual frame %d requires render_command_stream evidence", location, frame.Order))
		} else if frame.RenderCommandStreamHash != report.RenderCommandStream.CommandStreamHash {
			issues = append(issues, fmt.Sprintf("%s product visual frame %d render_command_stream_hash must match render_command_stream.command_stream_hash", location, frame.Order))
		}
	}
	return issues
}

func validateBlockSystemFrameProvenance(frame BlockSystemFrameReport, runtimeFrame FrameReport) []string {
	var issues []string
	role := normalizeFrameEvidenceToken(frame.EvidenceRole)
	producer := normalizeFrameEvidenceToken(frame.Producer)
	if role != "" && !isKnownFrameEvidenceRole(role) {
		issues = append(issues, fmt.Sprintf("block_system frame %d evidence_role %q is not supported", frame.Order, frame.EvidenceRole))
	}
	if frame.Precomputed && !isHostProbeOnlyFrameRole(role) {
		issues = append(issues, fmt.Sprintf("block_system frame %d precomputed evidence can only be host_probe_only infrastructure evidence", frame.Order))
	}
	if isProductVisualFrameRole(role) {
		if frame.Precomputed {
			issues = append(issues, fmt.Sprintf("block_system frame %d product visual evidence must not be precomputed", frame.Order))
		}
		if producer != "app" {
			issues = append(issues, fmt.Sprintf("block_system frame %d product visual producer is %q, want app", frame.Order, frame.Producer))
		}
	}
	if strings.TrimSpace(frame.EvidenceRole) != "" && strings.TrimSpace(runtimeFrame.EvidenceRole) != "" &&
		role != normalizeFrameEvidenceToken(runtimeFrame.EvidenceRole) {
		issues = append(issues, fmt.Sprintf("block_system frame %d evidence_role must match runtime frame evidence_role", frame.Order))
	}
	if strings.TrimSpace(frame.Producer) != "" && strings.TrimSpace(runtimeFrame.Producer) != "" &&
		producer != normalizeFrameEvidenceToken(runtimeFrame.Producer) {
		issues = append(issues, fmt.Sprintf("block_system frame %d producer must match runtime frame producer", frame.Order))
	}
	return issues
}

func normalizeFrameEvidenceToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func isKnownFrameEvidenceRole(role string) bool {
	switch normalizeFrameEvidenceToken(role) {
	case "product_visual", "host_probe_only", "runtime_smoke", "infrastructure_probe":
		return true
	default:
		return false
	}
}

func isProductVisualFrameRole(role string) bool {
	return normalizeFrameEvidenceToken(role) == "product_visual"
}

func isHostProbeOnlyFrameRole(role string) bool {
	switch normalizeFrameEvidenceToken(role) {
	case "host_probe_only", "infrastructure_probe":
		return true
	default:
		return false
	}
}
