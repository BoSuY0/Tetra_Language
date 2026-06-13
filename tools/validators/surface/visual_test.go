package surface

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateVisualReportAcceptsCompleteEvidence(t *testing.T) {
	raw := validSurfaceVisualReportJSON(t, nil)
	if err := ValidateVisualReport(raw); err != nil {
		t.Fatalf("ValidateVisualReport failed: %v\n%s", err, raw)
	}
}

func TestValidateVisualReportRejectsIncompleteEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*VisualRegressionReport)
		want   string
	}{
		{
			name: "screenshot only",
			mutate: func(report *VisualRegressionReport) {
				target := &report.Apps[0].Targets[0]
				target.ScreenshotOnly = true
				target.BlockGraphEvidence = false
				target.LayoutEvidence = false
				target.AccessibilityEvidence = false
				target.PerformanceEvidence = false
			},
			want: "screenshot-only",
		},
		{
			name: "stale golden",
			mutate: func(report *VisualRegressionReport) {
				report.Apps[0].Targets[0].GoldenGitHead = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			},
			want: "stale golden",
		},
		{
			name: "major drift",
			mutate: func(report *VisualRegressionReport) {
				frame := &report.Apps[0].Targets[0].Frames[0]
				frame.DiffPixels = 4096
				frame.DiffRatioMilli = 640
				frame.MaxChannelDelta = 64
				frame.Pass = false
			},
			want: "visual drift",
		},
		{
			name: "missing token conformance",
			mutate: func(report *VisualRegressionReport) {
				report.Apps[0].Targets[0].TokenThemeEvidence = false
			},
			want: "token/theme",
		},
		{
			name: "missing required target",
			mutate: func(report *VisualRegressionReport) {
				report.RequiredTargets = append(report.RequiredTargets, "linux-x64-real-window")
			},
			want: "required target",
		},
		{
			name: "missing required source",
			mutate: func(report *VisualRegressionReport) {
				report.RequiredSources = append(report.RequiredSources, "examples/surface_block_settings.tetra")
			},
			want: "required source",
		},
		{
			name: "negative guard missing",
			mutate: func(report *VisualRegressionReport) {
				report.NegativeGuards.ScreenshotOnlyRejected = false
			},
			want: "negative_guards",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validSurfaceVisualReportJSON(t, tc.mutate)
			err := ValidateVisualReport(raw)
			if err == nil {
				t.Fatalf("expected visual report %s evidence to fail", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func validSurfaceVisualReportJSON(t *testing.T, mutate func(*VisualRegressionReport)) []byte {
	t.Helper()
	report := VisualRegressionReport{
		Schema:          VisualRegressionSchemaV1,
		Status:          "pass",
		GitHead:         "c0258b63a636775b114d69d31cb7832fc3991b05",
		GoldenSet:       "surface-visual-regression-v1",
		GoldenHash:      "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		RequiredTargets: []string{"headless"},
		RequiredSources: []string{"examples/surface_block_system.tetra"},
		Apps: []VisualRegressionAppReport{
			{
				Name:         "surface-block-system",
				Source:       "examples/surface_block_system.tetra",
				ReferenceApp: true,
				Targets: []VisualRegressionTargetReport{
					{
						Target:                "headless",
						RuntimeReport:         "reports/surface-visual/headless/surface-headless-block-system.json",
						RuntimeSchema:         SchemaV1,
						GitHead:               "c0258b63a636775b114d69d31cb7832fc3991b05",
						GoldenGitHead:         "c0258b63a636775b114d69d31cb7832fc3991b05",
						Renderer:              "software-rgba",
						ScreenshotOnly:        false,
						BlockGraphEvidence:    true,
						TokenThemeEvidence:    true,
						LayoutEvidence:        true,
						AccessibilityEvidence: true,
						PerformanceEvidence:   true,
						Frames: []VisualRegressionFrameReport{
							{
								Order:                 1,
								Label:                 "initial",
								Width:                 320,
								Height:                200,
								Stride:                1280,
								Checksum:              "sha256:1111111111111111111111111111111111111111111111111111111111111111",
								GoldenChecksum:        "sha256:1111111111111111111111111111111111111111111111111111111111111111",
								DiffPixels:            0,
								DiffRatioMilli:        0,
								MaxChannelDelta:       0,
								TolerancePixels:       4,
								ToleranceRatioMilli:   1,
								ToleranceChannelDelta: 1,
								Pass:                  true,
							},
						},
					},
				},
			},
		},
		NegativeGuards: VisualRegressionNegativeGuardsReport{
			ScreenshotOnlyRejected:       true,
			StaleGoldenRejected:          true,
			MajorDriftRejected:           true,
			MissingBlockGraphRejected:    true,
			MissingLayoutRejected:        true,
			MissingAccessibilityRejected: true,
			MissingPerformanceRejected:   true,
		},
	}
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal visual report: %v", err)
	}
	return raw
}
