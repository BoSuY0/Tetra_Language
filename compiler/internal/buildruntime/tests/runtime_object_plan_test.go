package buildruntime_test

import (
	"testing"

	"tetra_language/compiler/internal/buildruntime"
)

func TestDecideRuntimeObjectPlan(t *testing.T) {
	tests := []struct {
		name            string
		target          string
		override        bool
		usage           buildruntime.RuntimeObjectPlanUsage
		wantRuntimeUsed bool
		wantTimeOnly    bool
		wantMinimal     bool
	}{
		{
			name:   "no runtime usage",
			target: "linux-x64",
		},
		{
			name:            "linux x86 time only runtime",
			target:          "linux-x86",
			usage:           buildruntime.RuntimeObjectPlanUsage{TimeRuntimeUsed: true},
			wantRuntimeUsed: true,
			wantTimeOnly:    true,
		},
		{
			name:            "override disables time only runtime",
			target:          "linux-x86",
			override:        true,
			usage:           buildruntime.RuntimeObjectPlanUsage{TimeRuntimeUsed: true},
			wantRuntimeUsed: true,
		},
		{
			name:   "actor usage disables time only runtime",
			target: "linux-x86",
			usage: buildruntime.RuntimeObjectPlanUsage{
				ActorRuntimeUsed: true,
				TimeRuntimeUsed:  true,
			},
			wantRuntimeUsed: true,
		},
		{
			name:   "linux x86 filesystem minimal runtime",
			target: "linux-x86",
			usage: buildruntime.RuntimeObjectPlanUsage{
				FilesystemRuntimeUsed: true,
				NetRuntimeSupported:   true,
			},
			wantRuntimeUsed: true,
			wantMinimal:     true,
		},
		{
			name:   "linux x32 networking minimal runtime",
			target: "linux-x32",
			usage: buildruntime.RuntimeObjectPlanUsage{
				NetRuntimeUsed:      true,
				NetRuntimeSupported: true,
			},
			wantRuntimeUsed: true,
			wantMinimal:     true,
		},
		{
			name:   "unsupported net usage disables minimal runtime",
			target: "linux-x86",
			usage: buildruntime.RuntimeObjectPlanUsage{
				NetRuntimeUsed:      true,
				NetRuntimeSupported: false,
			},
			wantRuntimeUsed: true,
		},
		{
			name:   "actor usage disables minimal runtime",
			target: "linux-x32",
			usage: buildruntime.RuntimeObjectPlanUsage{
				ActorsUsed:            true,
				FilesystemRuntimeUsed: true,
				NetRuntimeSupported:   true,
			},
			wantRuntimeUsed: true,
		},
		{
			name:   "linux x64 does not use minimal runtime",
			target: "linux-x64",
			usage: buildruntime.RuntimeObjectPlanUsage{
				FilesystemRuntimeUsed: true,
				NetRuntimeSupported:   true,
			},
			wantRuntimeUsed: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildruntime.DecideRuntimeObjectPlan(
				tt.target,
				tt.override,
				buildruntime.CapabilitiesForTarget(tt.target),
				tt.usage,
			)
			if got.RuntimeUsed != tt.wantRuntimeUsed || got.TimeOnlyRuntime != tt.wantTimeOnly ||
				got.LinuxMinimalRuntime != tt.wantMinimal {
				t.Fatalf(
					"DecideRuntimeObjectPlan = %+v, want runtime=%v timeOnly=%v minimal=%v",
					got,
					tt.wantRuntimeUsed,
					tt.wantTimeOnly,
					tt.wantMinimal,
				)
			}
		})
	}
}
