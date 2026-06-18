package main

import (
	"strings"
	"testing"
)

func TestValidateMorphRenderedBeautyReportRejectsMissingBlockSceneSnapshot(t *testing.T) {
	report := validMorphRenderedBeautyReportFixture()
	report.BlockSceneSnapshot = morphRenderedBeautyBlockSceneSnapshot{}

	err := validateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected missing block_scene_snapshot to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "block_scene_snapshot") {
		t.Fatalf("error = %v, want block_scene_snapshot diagnostic", err)
	}
}

func TestValidateMorphRenderedBeautyReportRejectsCompactOnlyBlockSceneSnapshot(t *testing.T) {
	report := validMorphRenderedBeautyReportFixture()
	report.BlockSceneSnapshot.CompactPropsOnly = true

	err := validateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected compact-only block_scene_snapshot to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "compact") {
		t.Fatalf("error = %v, want compact diagnostic", err)
	}
}
