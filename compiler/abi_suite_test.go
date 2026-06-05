package compiler

import (
	"strings"
	"testing"
)

func TestRunTargetABIChecksCoversP21Targets(t *testing.T) {
	tests := []struct {
		target string
		names  []string
	}{
		{
			target: "x86",
			names:  []string{"x86 target model", "x86 i386 SysV classifier", "x86 varargs and sret ABI", "x86 pointer FFI object smoke", "x86 c_int FFI object smoke", "x86 c_uint FFI object smoke", "x86 ILP32 native/libc FFI object smoke", "x86 ref FFI null-return diagnostics", "x86 function-pointer FFI diagnostics", "x86 source native scalar diagnostics", "x86 stdout executable smoke", "x86 stderr fd runtime smoke", "x86 allocator executable smoke", "x86 allocator failure executable smoke", "x86 raw memory bounds executable smoke", "x86 raw pointer slot executable smoke", "x86 raw pointer offset slot executable smoke", "x86 island free executable smoke", "x86 stdlib runtime boundary diagnostics", "x86 filesystem runtime smoke", "x86 filesystem scheduler composition smoke", "x86 time runtime smoke", "x86 single-actor self-host runtime smoke", "x86 single-task self-host runtime smoke", "x86 typed-task self-host runtime smoke", "x86 staged typed-task self-host runtime smoke", "x86 task-group self-host runtime smoke", "x86 typed-task-group self-host runtime smoke", "x86 actor-state self-host runtime smoke", "x86 ctx_switch object smoke", "x86 target runtime boundary diagnostics", "x86 networking runtime boundary diagnostics", "x86 networking lifecycle runtime smoke", "x86 surface/distributed runtime boundary diagnostics", "x86 pointer atomic ABI width"},
		},
		{
			target: "x64",
			names:  []string{"x64 target model", "x64 SysV classifier", "x64 SysV varargs and aggregates", "x64 source native scalar diagnostics", "x64 pointer FFI regression smoke", "x64 c_int FFI object smoke", "x64 c_uint FFI object smoke", "x64 filesystem scheduler composition smoke", "x64 networking runtime smoke", "x64 scheduler restriction regression smoke", "x64 pointer atomic ABI width"},
		},
		{
			target: "windows-x64",
			names:  []string{"windows-x64 target model", "windows-x64 Win64 classifier", "windows-x64 Win64 varargs and aggregates", "windows-x64 object ABI smoke", "windows-x64 source native scalar diagnostics", "windows-x64 pointer atomic ABI width"},
		},
		{
			target: "macos-x64",
			names:  []string{"macos-x64 target model", "macos-x64 SysV classifier", "macos-x64 SysV varargs and aggregates", "macos-x64 object ABI smoke", "macos-x64 source native scalar diagnostics", "macos-x64 pointer atomic ABI width"},
		},
		{
			target: "x32",
			names:  []string{"x32 target model", "x32 SysV classifier", "x32 SysV varargs and aggregates", "x32 pointer FFI object smoke", "x32 c_int FFI object smoke", "x32 c_uint FFI object smoke", "x32 ILP32 native/libc FFI object smoke", "x32 ref FFI null-return diagnostics", "x32 function-pointer FFI diagnostics", "x32 source native scalar diagnostics", "x32 stdout executable smoke", "x32 stderr fd runtime smoke", "x32 allocator executable smoke", "x32 allocator failure executable smoke", "x32 raw memory bounds executable smoke", "x32 raw pointer slot executable smoke", "x32 raw pointer offset slot executable smoke", "x32 island free executable smoke", "x32 stdlib runtime boundary diagnostics", "x32 time runtime smoke", "x32 filesystem runtime smoke", "x32 filesystem scheduler composition smoke", "x32 single-actor self-host runtime smoke", "x32 single-task self-host runtime smoke", "x32 typed-task self-host runtime smoke", "x32 staged typed-task self-host runtime smoke", "x32 task-group self-host runtime smoke", "x32 typed-task-group self-host runtime smoke", "x32 actor-state self-host runtime smoke", "x32 ctx_switch object smoke", "x32 target runtime boundary diagnostics", "x32 networking runtime boundary diagnostics", "x32 networking lifecycle runtime smoke", "x32 surface/distributed runtime boundary diagnostics", "x32 pointer atomic ABI width"},
		},
		{
			target: "wasm32-wasi",
			names:  []string{"wasm32-wasi target model", "wasm32-wasi slot ABI metadata", "wasm32-wasi struct/enum/slice/String return layout", "wasm32-wasi call boundary validation", "wasm32-wasi FFI repr(C) boundary policy"},
		},
		{
			target: "wasm32-web",
			names:  []string{"wasm32-web target model", "wasm32-web slot ABI metadata", "wasm32-web struct/enum/slice/String return layout", "wasm32-web call boundary validation", "wasm32-web FFI repr(C) boundary policy"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			checks, err := RunTargetABIChecks(tt.target)
			if err != nil {
				t.Fatalf("RunTargetABIChecks(%s): %v", tt.target, err)
			}
			if len(checks) != len(tt.names) {
				t.Fatalf("checks = %#v, want %d checks", checks, len(tt.names))
			}
			for i, want := range tt.names {
				if checks[i].Name != want || checks[i].Error != "" {
					t.Fatalf("check[%d] = %#v, want passing %q", i, checks[i], want)
				}
			}
		})
	}
}

func TestP21ABIVerificationReportCoversTargetsTasksAndNonClaims(t *testing.T) {
	report := BuildP21ABIVerificationReport()
	if report.Schema != abiVerificationSchemaV1 {
		t.Fatalf("ABI report schema = %q, want %q", report.Schema, abiVerificationSchemaV1)
	}
	if report.Scope != abiVerificationScopeP211 {
		t.Fatalf("ABI report scope = %q, want %q", report.Scope, abiVerificationScopeP211)
	}
	if err := ValidateP21ABIVerificationReport(report); err != nil {
		t.Fatalf("ValidateP21ABIVerificationReport: %v", err)
	}

	targetRows := map[string]ABIVerificationTargetRow{}
	for _, row := range report.Targets {
		if row.Target == "" || row.ABI == "" || row.Status == "" || len(row.Evidence) == 0 {
			t.Fatalf("ABI target row missing required metadata: %#v", row)
		}
		targetRows[row.Target] = row
	}
	for _, target := range []string{"linux-x64", "linux-x86", "linux-x32", "macos-x64", "windows-x64", "wasm32-wasi", "wasm32-web"} {
		row, ok := targetRows[target]
		if !ok {
			t.Fatalf("ABI report missing target %s: %#v", target, report.Targets)
		}
		for _, task := range p21ABIVerificationTaskIDs() {
			if !p21ABIHasString(row.TaskCoverage, task) {
				t.Fatalf("ABI target %s missing task %s coverage: %#v", target, task, row)
			}
		}
	}

	taskRows := map[string]ABIVerificationTaskRow{}
	for _, row := range report.Tasks {
		if row.ID == "" || row.Name == "" || len(row.Targets) == 0 || len(row.Evidence) == 0 {
			t.Fatalf("ABI task row missing required metadata: %#v", row)
		}
		taskRows[row.ID] = row
	}
	for _, task := range p21ABIVerificationTaskIDs() {
		if _, ok := taskRows[task]; !ok {
			t.Fatalf("ABI report missing task row %s: %#v", task, report.Tasks)
		}
	}
	for _, nonClaim := range []string{
		"no runtime execution claim for build-only or wasm targets",
		"no C ABI claim for default structs",
		"no native C aggregate ABI claim for wasm targets",
		"no performance claim",
		"no safe-program semantics change",
	} {
		if !p21ABIHasString(report.NonClaims, nonClaim) {
			t.Fatalf("ABI report missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP21ABIVerificationReportRejectsFakeClaims(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ABIVerificationReport)
		want   string
	}{
		{
			name: "missing target",
			mutate: func(report *ABIVerificationReport) {
				report.Targets = report.Targets[1:]
			},
			want: "missing target",
		},
		{
			name: "missing task",
			mutate: func(report *ABIVerificationReport) {
				report.Tasks = report.Tasks[1:]
			},
			want: "missing task",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *ABIVerificationReport) {
				report.Targets[0].Evidence = []string{"placeholder evidence"}
			},
			want: "placeholder",
		},
		{
			name: "fake full runtime",
			mutate: func(report *ABIVerificationReport) {
				report.Claims = append(report.Claims, "full runtime execution verified for wasm32-wasi and linux-x86")
			},
			want: "runtime execution",
		},
		{
			name: "fake wasm C aggregate ABI",
			mutate: func(report *ABIVerificationReport) {
				report.Claims = append(report.Claims, "wasm32-web native C aggregate ABI verified")
			},
			want: "wasm",
		},
		{
			name: "fake default struct C ABI",
			mutate: func(report *ABIVerificationReport) {
				report.Claims = append(report.Claims, "default structs have C ABI")
			},
			want: "default structs",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := cloneABIVerificationReport(BuildP21ABIVerificationReport())
			tc.mutate(&report)
			err := ValidateP21ABIVerificationReport(report)
			if err == nil {
				t.Fatalf("ValidateP21ABIVerificationReport accepted %#v", report)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func cloneABIVerificationReport(report ABIVerificationReport) ABIVerificationReport {
	report.Claims = append([]string{}, report.Claims...)
	report.NonClaims = append([]string{}, report.NonClaims...)
	report.Targets = append([]ABIVerificationTargetRow{}, report.Targets...)
	for i := range report.Targets {
		report.Targets[i].TaskCoverage = append([]string{}, report.Targets[i].TaskCoverage...)
		report.Targets[i].Evidence = append([]string{}, report.Targets[i].Evidence...)
		report.Targets[i].Claims = append([]string{}, report.Targets[i].Claims...)
	}
	report.Tasks = append([]ABIVerificationTaskRow{}, report.Tasks...)
	for i := range report.Tasks {
		report.Tasks[i].Targets = append([]string{}, report.Tasks[i].Targets...)
		report.Tasks[i].Evidence = append([]string{}, report.Tasks[i].Evidence...)
	}
	return report
}

func p21ABIHasString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
