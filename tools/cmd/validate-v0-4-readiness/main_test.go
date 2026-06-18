package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/nativeui"
)

func TestValidateReadinessRejectsCurrentV030Evidence(t *testing.T) {
	err := validateReadiness(readinessInputs{
		ExpectedVersion: "v0.4.0",
		Manifest: []byte(`{
  "compiler_version": "v0.3.0"
}`),
		Features: []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.3.0",
  "features": [
    {"id":"language.callable-level1","status":"experimental"},
    {"id":"language.callable-level2","status":"planned"},
    {"id":"eco.distributed-network","status":"post-v1"}
  ]
}`),
		Targets: readinessTargetsJSON(
			readinessTarget{
				Triple:               "windows-x64",
				Status:               "supported",
				RunUnsupportedReason: "windows-x64 cannot run on host linux/amd64",
			},
			readinessTarget{
				Triple:               "wasm32-web",
				Status:               "build_only",
				BuildOnly:            true,
				RunUnsupportedReason: "browser smoke only",
			},
		),
		ScopeDecisions: readinessScopeDecisionsJSON(
			"full-production-scope-selected",
			featureDecisionWithoutEvidence("language.callable-level1"),
			featureDecisionWithoutEvidence("language.callable-level2"),
			featureDecisionWithoutEvidence("eco.distributed-network"),
			runtimeDecisionWithoutEvidence("windows-x64"),
			runtimeDecisionWithoutEvidence("wasm32-web"),
		),
	})
	if err == nil {
		t.Fatalf("expected current v0.3.0 evidence to fail readiness")
	}
	got := err.Error()
	for _, want := range []string{
		"manifest compiler_version = v0.3.0, want v0.4.0",
		"features version = v0.3.0, want v0.4.0",
		"feature language.callable-level1 status = experimental, want current",
		"feature language.callable-level2 status = planned, want current",
		"feature eco.distributed-network status = post-v1, want current",
		"target windows-x64 run_supported = false",
		"target wasm32-web build_only = true",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("readiness error missing %q:\n%s", want, got)
		}
	}
}

func TestValidateReadinessAcceptsCurrentFeaturesAndRuntimeTargets(t *testing.T) {
	chdirReadinessEvidenceRoot(t,
		"compiler/tests/callables/core/function_typed_callable_test.go",
		"cli/cmd/tetra/tetra_eco.go",
		"compiler/internal/backend/windows_x64",
		"compiler/internal/backend/wasm32_web",
		"docs/spec/core/current_supported_surface.md",
		"docs/user/platform/eco_package_guide.md",
		"docs/user/surface/wasm_ui_guide.md",
		"reports/v0.4.0/callable.json",
		"reports/v0.4.0/eco.json",
		"reports/v0.4.0/windows.json",
		"reports/v0.4.0/web.json",
	)

	err := validateReadiness(readinessInputs{
		ExpectedVersion: "v0.4.0",
		Manifest: []byte(`{
  "compiler_version": "v0.4.0"
}`),
		Features: []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.4.0",
  "features": [
    {"id":"language.callable-level1","status":"current","since":"v0.4.0"},
    {"id":"eco.distributed-network","status":"current","since":"v0.4.0"}
  ]
}`),
		Targets: readinessTargetsJSON(
			readinessTarget{Triple: "windows-x64", Status: "supported", RunSupported: true},
			readinessTarget{Triple: "wasm32-web", Status: "supported", RunSupported: true},
		),
		ScopeDecisions: readinessScopeDecisionsJSON(
			"full-production-scope-selected",
			featureDecision("language.callable-level1", decisionEvidence{
				Implementation: []string{
					"compiler/tests/callables/core/function_typed_callable_test.go",
				},
				Tests: []string{
					"go test ./compiler/... -run 'Closure|Callable|FunctionType' -count=1",
				},
				Docs:                []string{"docs/spec/core/current_supported_surface.md"},
				ReleaseGateEvidence: []string{"reports/v0.4.0/callable.json"},
			}),
			featureDecision("eco.distributed-network", decisionEvidence{
				Implementation:      []string{"cli/cmd/tetra/tetra_eco.go"},
				Tests:               []string{"go test ./cli/... -run Eco -count=1"},
				Docs:                []string{"docs/user/platform/eco_package_guide.md"},
				ReleaseGateEvidence: []string{"reports/v0.4.0/eco.json"},
			}),
			runtimeDecision("windows-x64", decisionEvidence{
				Implementation: []string{"compiler/internal/backend/windows_x64"},
				Tests: []string{
					"./tetra smoke --target windows-x64 --run=true " +
						"--report reports/v0.4.0/windows.json",
				},
				Docs:                []string{"docs/spec/core/current_supported_surface.md"},
				ReleaseGateEvidence: []string{"reports/v0.4.0/windows.json"},
			}),
			runtimeDecision("wasm32-web", decisionEvidence{
				Implementation: []string{"compiler/internal/backend/wasm32_web"},
				Tests: []string{
					"bash scripts/release/v1_0/web-smoke.sh --report " +
						"reports/v0.4.0/web.json",
				},
				Docs:                []string{"docs/user/surface/wasm_ui_guide.md"},
				ReleaseGateEvidence: []string{"reports/v0.4.0/web.json"},
			}),
		),
	})
	if err != nil {
		t.Fatalf("expected readiness to pass: %v", err)
	}
}

func TestValidateReadinessAcceptsLinuxOnlyNoEcoNetScope(t *testing.T) {
	chdirReadinessEvidenceRoot(t,
		"compiler/internal/frontend/frontend_core.go",
		"compiler/internal/semantics/semantics_checker.go",
		"compiler/internal/semantics/semantics_memory_resources.go",
		"compiler/internal/lower/lower_core.go",
		"compiler/internal/backend/native_shell/codegen.go",
		"tools/cmd/native-ui-runtime-smoke/main.go",
		"tools/validators/nativeui/report.go",
		"lib/experimental/base/math.tetra",
		"docs/spec/core/current_supported_surface.md",
		"docs/spec/flow/flow_syntax_v1.md",
		"docs/spec/policy/v1_feature_status.md",
		"docs/spec/runtime/ownership_v1.md",
		"docs/spec/standard_library/stdlib.md",
		"docs/spec/standard_library/stdlib_naming_versioning.md",
		"docs/user/platform/standard_library_guide.md",
		"docs/spec/ui/ui_v0.4.0.md",
		"docs/user/surface/wasm_ui_guide.md",
		"reports/v0.4.0/features.json",
		"reports/v0.4.0/native-ui-linux-x64.json",
	)
	writeReadinessEvidenceFile(
		t,
		"reports/v0.4.0/features.json",
		readinessFeaturesJSON("v0.4.0", readinessFeature{
			ID:     "language.callable-level1",
			Status: "current",
		}),
	)
	writeReadinessEvidenceFile(
		t,
		"reports/v0.4.0/native-ui-linux-x64.json",
		readinessNativeUIRuntimeJSON(),
	)

	err := validateReadiness(readinessInputs{
		ExpectedVersion: "v0.4.0",
		Manifest: []byte(`{
  "compiler_version": "v0.4.0"
}`),
		Features: []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.4.0",
  "features": [
    {"id":"language.callable-level1","status":"current","since":"v0.4.0"},
    {"id":"language.callable-level2","status":"current","since":"v0.4.0"},
    {"id":"language.full-first-class-callables","status":"current","since":"v0.4.0"},
    {"id":"language.lifetime-ssa","status":"current","since":"v0.4.0"},
    {"id":"stdlib.experimental-mirrors","status":"current","since":"v0.4.0"},
    {"id":"ui.metadata-v1","status":"current","since":"v0.4.0"},
    {"id":"ui.native-runtime","status":"current","since":"v0.4.0"},
    {"id":"wasm.runtime-execution","status":"current","since":"v0.4.0"},
    {"id":"eco.distributed-network","status":"post-v1"},
    {"id":"language.full-v1-guarantees","status":"planned"}
  ]
}`),
		Targets: readinessTargetsJSON(
			readinessTarget{Triple: "linux-x64", Status: "supported", RunSupported: true},
			readinessTarget{
				Triple:               "windows-x64",
				Status:               "supported",
				RunUnsupportedReason: "windows-x64 cannot run on host linux/amd64",
			},
			readinessTarget{
				Triple:               "macos-x64",
				Status:               "supported",
				RunUnsupportedReason: "macos-x64 cannot run on host linux/amd64",
			},
			readinessTarget{
				Triple:               "wasm32-wasi",
				Status:               "supported",
				RunUnsupportedReason: "runner unavailable",
			},
			readinessTarget{
				Triple:               "wasm32-web",
				Status:               "supported",
				RunUnsupportedReason: "runner unavailable",
			},
		),
		ScopeDecisions: linuxOnlyScopeDecisionsJSON(),
	})
	if err != nil {
		t.Fatalf("expected linux-only no-EcoNet scope to pass readiness: %v", err)
	}
}

func TestValidateReadinessRejectsActorDistributedRuntimeTransportOnlyEvidence(t *testing.T) {
	chdirReadinessEvidenceRoot(t,
		"tools/cmd/validate-actor-transport/main.go",
		"tools/cmd/validate-actor-transport/main_test.go",
		"docs/spec/runtime/actors.md",
		"reports/v0.4.0/actor-transport.json",
	)
	writeReadinessEvidenceFile(
		t,
		"reports/v0.4.0/actor-transport.json",
		readinessJSON(map[string]any{
			"schema": "tetra.actors.transport.v1",
			"status": "pass",
			"trace": []map[string]any{
				{"event": "send"},
				{"event": "receive"},
			},
		}),
	)

	err := validateReadiness(readinessInputs{
		ExpectedVersion: "v0.4.0",
		Manifest: []byte(`{
  "compiler_version": "v0.4.0"
}`),
		Features: []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.4.0",
  "features": [
    {"id":"actors.distributed-runtime","status":"current","since":"v0.4.0"}
  ]
}`),
		Targets: []byte(`{
  "targets": []
}`),
		ScopeDecisions: readinessScopeDecisionsJSON(
			"full-production-scope-selected",
			featureDecision("actors.distributed-runtime", decisionEvidence{
				Implementation:      []string{"tools/cmd/validate-actor-transport/main.go"},
				Tests:               []string{"go test ./tools/cmd/validate-actor-transport -count=1"},
				Docs:                []string{"docs/spec/runtime/actors.md"},
				ReleaseGateEvidence: []string{"reports/v0.4.0/actor-transport.json"},
			}),
		),
	})
	if err == nil {
		t.Fatalf("expected transport-only actor evidence to fail readiness")
	}
	got := err.Error()
	for _, want := range []string{
		"decision actors.distributed-runtime has only actor transport evidence",
		"requires real distributed actor runtime/lowering evidence",
		"tetra.actors.transport.v1",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("readiness error missing %q:\n%s", want, got)
		}
	}
}

func TestValidateReadinessAcceptsActorDistributedRuntimeEvidenceShape(t *testing.T) {
	chdirReadinessEvidenceRoot(t,
		"compiler/internal/actorsrt/actorsrt_core.go",
		"compiler/tests/runtime/distributed_actor_runtime_test.go",
		"docs/spec/core/current_supported_surface.md",
		"docs/spec/runtime/actors.md",
		"reports/v0.4.0/actors-distributed-runtime.json",
	)
	writeReadinessEvidenceFile(
		t,
		"compiler/internal/actorsrt/actorsrt_core.go",
		[]byte(
			"func emitActorNodeConnect() {}\nfunc emitActorSpawnRemote() {}\nfunc emitActorNetPump() {}\n",
		),
	)
	writeReadinessEvidenceFile(
		t,
		"reports/v0.4.0/actors-distributed-runtime.json",
		readinessDistributedActorRuntimeJSON(),
	)

	err := validateReadiness(readinessInputs{
		ExpectedVersion: "v0.4.0",
		Manifest: []byte(`{
  "compiler_version": "v0.4.0"
}`),
		Features: []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.4.0",
  "features": [
    {"id":"actors.distributed-runtime","status":"current","since":"v0.4.0"}
  ]
}`),
		Targets: []byte(`{
  "targets": []
}`),
		ScopeDecisions: readinessScopeDecisionsJSON(
			"full-production-scope-selected",
			featureDecision("actors.distributed-runtime", decisionEvidence{
				Implementation: []string{"compiler/internal/actorsrt/actorsrt_core.go"},
				Tests:          []string{"compiler/tests/runtime/distributed_actor_runtime_test.go"},
				Docs: []string{
					"docs/spec/core/current_supported_surface.md",
					"docs/spec/runtime/actors.md",
				},
				ReleaseGateEvidence: []string{
					"reports/v0.4.0/actors-distributed-runtime.json",
				},
			}),
		),
	})
	if err != nil {
		t.Fatalf("expected distributed actor runtime-shaped evidence to pass readiness: %v", err)
	}
}

func TestValidateReadinessRejectsActorDistributedRuntimeThinReport(t *testing.T) {
	chdirReadinessEvidenceRoot(t,
		"compiler/internal/actorsrt/distributed_runtime.go",
		"compiler/tests/runtime/distributed_actor_runtime_test.go",
		"docs/spec/core/current_supported_surface.md",
		"docs/spec/runtime/actors.md",
		"reports/v0.4.0/actors-distributed-runtime.json",
	)
	writeReadinessEvidenceFile(
		t,
		"reports/v0.4.0/actors-distributed-runtime.json",
		readinessJSON(map[string]any{
			"schema":  "tetra.actors.distributed-runtime.v1",
			"status":  "pass",
			"runtime": "compiler/internal/actorsrt/distributed_runtime.go",
			"cases": []map[string]any{
				{"name": "cross-node send/receive", "pass": true},
				{"name": "failure/cancel/join diagnostics", "pass": true},
			},
		}),
	)

	err := validateReadiness(readinessInputs{
		ExpectedVersion: "v0.4.0",
		Manifest: []byte(`{
  "compiler_version": "v0.4.0"
}`),
		Features: []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.4.0",
  "features": [
    {"id":"actors.distributed-runtime","status":"current","since":"v0.4.0"}
  ]
}`),
		Targets: []byte(`{
  "targets": []
}`),
		ScopeDecisions: readinessScopeDecisionsJSON(
			"full-production-scope-selected",
			featureDecision("actors.distributed-runtime", decisionEvidence{
				Implementation: []string{"compiler/internal/actorsrt/distributed_runtime.go"},
				Tests:          []string{"compiler/tests/runtime/distributed_actor_runtime_test.go"},
				Docs: []string{
					"docs/spec/core/current_supported_surface.md",
					"docs/spec/runtime/actors.md",
				},
				ReleaseGateEvidence: []string{
					"reports/v0.4.0/actors-distributed-runtime.json",
				},
			}),
		),
	})
	if err == nil {
		t.Fatalf("expected thin distributed actor runtime report to fail readiness")
	}
	got := strings.ToLower(err.Error())
	for _, want := range []string{"target", "loopback", "process", "frame"} {
		if !strings.Contains(got, want) {
			t.Fatalf("readiness error missing %q:\n%s", want, err.Error())
		}
	}
}

func TestValidateReadinessRejectsNativeUISidecarOnlyEvidence(t *testing.T) {
	chdirReadinessEvidenceRoot(t,
		"compiler/internal/backend/native_shell/codegen.go",
		"tools/cmd/validate-native-ui-smoke/main.go",
		"docs/spec/core/current_supported_surface.md",
		"docs/spec/ui/ui_v0.4.0.md",
		"reports/v0.4.0/native-ui-sidecar.json",
	)
	writeReadinessEvidenceFile(
		t,
		"reports/v0.4.0/native-ui-sidecar.json",
		readinessJSON(map[string]any{
			"schema":    "tetra.ui.native-shell.v1",
			"ui_schema": "tetra.ui.v0.4.0",
			"runtime":   "native shell command dispatch",
		}),
	)

	err := validateReadiness(readinessInputs{
		ExpectedVersion: "v0.4.0",
		Manifest: []byte(`{
  "compiler_version": "v0.4.0"
}`),
		Features: []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.4.0",
  "features": [
    {"id":"ui.native-runtime","status":"current","since":"v0.4.0"}
  ]
}`),
		Targets: []byte(`{
  "targets": []
}`),
		ScopeDecisions: readinessScopeDecisionsJSON(
			"full-production-scope-selected",
			featureDecision("ui.native-runtime", decisionEvidence{
				Implementation: []string{"compiler/internal/backend/native_shell/codegen.go"},
				Tests:          []string{"go test ./tools/cmd/validate-native-ui-smoke -count=1"},
				Docs: []string{
					"docs/spec/core/current_supported_surface.md",
					"docs/spec/ui/ui_v0.4.0.md",
				},
				ReleaseGateEvidence: []string{"reports/v0.4.0/native-ui-sidecar.json"},
			}),
		),
	})
	if err == nil {
		t.Fatalf("expected native UI sidecar-only evidence to fail readiness")
	}
	got := err.Error()
	for _, want := range []string{
		"decision ui.native-runtime has only metadata/web/native-shell sidecar evidence",
		"requires real Linux-x64 native UI runtime",
		"tetra.ui.native-shell.v1",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("readiness error missing %q:\n%s", want, got)
		}
	}
}

func TestValidateReadinessAcceptsNativeUIRuntimeEvidenceShape(t *testing.T) {
	chdirReadinessEvidenceRoot(t,
		"tools/cmd/native-ui-runtime-smoke/main.go",
		"tools/validators/nativeui/report.go",
		"docs/spec/core/current_supported_surface.md",
		"docs/spec/ui/ui_v0.4.0.md",
		"reports/v0.4.0/native-ui-linux-x64.json",
	)
	writeReadinessEvidenceFile(
		t,
		"tools/cmd/native-ui-runtime-smoke/main.go",
		[]byte("func runRuntimeScenario() {}\nconst runtime = \"native-ui-linux-x64\"\n"),
	)
	writeReadinessEvidenceFile(
		t,
		"tools/validators/nativeui/report.go",
		[]byte("const SchemaV1 = \"tetra.ui.native-runtime.v1\"\n"),
	)
	writeReadinessEvidenceFile(
		t,
		"reports/v0.4.0/native-ui-linux-x64.json",
		readinessNativeUIRuntimeJSON(),
	)

	err := validateReadiness(readinessInputs{
		ExpectedVersion: "v0.4.0",
		Manifest: []byte(`{
  "compiler_version": "v0.4.0"
}`),
		Features: []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.4.0",
  "features": [
    {"id":"ui.native-runtime","status":"current","since":"v0.4.0"}
  ]
}`),
		Targets: []byte(`{
  "targets": []
}`),
		ScopeDecisions: readinessScopeDecisionsJSON(
			"full-production-scope-selected",
			featureDecision("ui.native-runtime", nativeUIRuntimeEvidence()),
		),
	})
	if err != nil {
		t.Fatalf("expected native UI runtime-shaped evidence to pass readiness: %v", err)
	}
}

func TestValidateReadinessRejectsNativeUIRuntimeThinReport(t *testing.T) {
	chdirReadinessEvidenceRoot(t,
		"tools/cmd/native-ui-runtime-smoke/main.go",
		"tools/validators/nativeui/report.go",
		"docs/spec/core/current_supported_surface.md",
		"docs/spec/ui/ui_v0.4.0.md",
		"reports/v0.4.0/native-ui-linux-x64.json",
	)
	writeReadinessEvidenceFile(
		t,
		"reports/v0.4.0/native-ui-linux-x64.json",
		[]byte(
			`{"schema":"tetra.ui.native-runtime.v1","status":"pass","runtime":"native-ui-linux-x64"}`+"\n",
		),
	)

	err := validateReadiness(readinessInputs{
		ExpectedVersion: "v0.4.0",
		Manifest: []byte(`{
  "compiler_version": "v0.4.0"
}`),
		Features: []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.4.0",
  "features": [
    {"id":"ui.native-runtime","status":"current","since":"v0.4.0"}
  ]
}`),
		Targets: []byte(`{
  "targets": []
}`),
		ScopeDecisions: readinessScopeDecisionsJSON(
			"full-production-scope-selected",
			featureDecision("ui.native-runtime", decisionEvidence{
				Implementation: []string{
					"tools/cmd/native-ui-runtime-smoke/main.go",
					"tools/validators/nativeui/report.go",
				},
				Tests: []string{
					"go test ./tools/cmd/native-ui-runtime-smoke " +
						"./tools/validators/nativeui -count=1",
				},
				Docs: []string{
					"docs/spec/core/current_supported_surface.md",
					"docs/spec/ui/ui_v0.4.0.md",
				},
				ReleaseGateEvidence: []string{"reports/v0.4.0/native-ui-linux-x64.json"},
			}),
		),
	})
	if err == nil {
		t.Fatalf("expected thin native UI runtime report to fail readiness")
	}
	got := strings.ToLower(err.Error())
	for _, want := range []string{"target", "process", "event", "state"} {
		if !strings.Contains(got, want) {
			t.Fatalf("readiness error missing %q:\n%s", want, err.Error())
		}
	}
}

func TestValidateReadinessAcceptsCrossHostRuntimeEvidenceReport(t *testing.T) {
	chdirReadinessEvidenceRoot(t,
		"compiler/internal/backend/windows_x64",
		"docs/spec/core/current_supported_surface.md",
		"reports/v0.4.0/windows-runtime-smoke.json",
	)

	err := validateReadiness(readinessInputs{
		ExpectedVersion: "v0.4.0",
		Manifest: []byte(`{
  "compiler_version": "v0.4.0"
}`),
		Features: []byte(`{
  "version": "v0.4.0",
  "features": []
}`),
		Targets: readinessTargetsJSON(windowsUnsupportedTarget()),
		ScopeDecisions: readinessScopeDecisionsJSON(
			"full-production-scope-selected",
			runtimeDecision("windows-x64", windowsRuntimeEvidence()),
		),
		RuntimeReports: map[string][]byte{
			"windows-x64": readinessRuntimeSmokeJSON("windows-x64", "v0.4.0", true),
		},
	})
	if err != nil {
		t.Fatalf("expected cross-host runtime evidence report to satisfy readiness: %v", err)
	}
}

func TestValidateReadinessRejectsRuntimeEvidenceReportFromWrongHost(t *testing.T) {
	chdirReadinessEvidenceRoot(t,
		"compiler/internal/backend/windows_x64",
		"docs/spec/core/current_supported_surface.md",
		"reports/v0.4.0/windows-runtime-smoke.json",
	)

	err := validateReadiness(readinessInputs{
		ExpectedVersion: "v0.4.0",
		Manifest: []byte(`{
  "compiler_version": "v0.4.0"
}`),
		Features: []byte(`{
  "version": "v0.4.0",
  "features": []
}`),
		Targets: readinessTargetsJSON(windowsUnsupportedTarget()),
		ScopeDecisions: readinessScopeDecisionsJSON(
			"full-production-scope-selected",
			runtimeDecision("windows-x64", windowsRuntimeEvidence()),
		),
		RuntimeReports: map[string][]byte{
			"windows-x64": readinessRuntimeSmokeJSONWithHost(
				"windows-x64",
				"linux-x64",
				"v0.4.0",
				true,
			),
		},
	})
	if err == nil {
		t.Fatalf("expected runtime evidence from the wrong host to fail readiness")
	}
	if !strings.Contains(
		err.Error(),
		`windows-x64 runtime report host is "linux-x64", want "windows-x64"`,
	) {
		t.Fatalf("unexpected readiness error: %v", err)
	}
}

func TestValidateReadinessRejectsImplementedDecisionWithoutEvidence(t *testing.T) {
	err := validateReadiness(readinessInputs{
		ExpectedVersion: "v0.4.0",
		Manifest: []byte(`{
  "compiler_version": "v0.4.0"
}`),
		Features: []byte(`{
  "version": "v0.4.0",
  "features": [
    {"id":"language.callable-level1","status":"current","since":"v0.4.0"}
  ]
}`),
		Targets: readinessTargetsJSON(readinessTarget{
			Triple:       "wasm32-web",
			Status:       "supported",
			RunSupported: true,
		}),
		ScopeDecisions: readinessScopeDecisionsJSON(
			"full-production-scope-selected",
			featureDecisionWithoutEvidence("language.callable-level1"),
			runtimeDecision("wasm32-web", wasmRuntimeEvidence()),
		),
	})
	if err == nil {
		t.Fatalf("expected implemented decision without evidence to fail readiness")
	}
	if !strings.Contains(
		err.Error(),
		"decision language.callable-level1 missing evidence.implementation",
	) {
		t.Fatalf("unexpected readiness error: %v", err)
	}
}

func TestValidateReadinessRejectsMissingImplementationOrDocsEvidencePaths(t *testing.T) {
	err := validateReadiness(readinessInputs{
		ExpectedVersion: "v0.4.0",
		Manifest: []byte(`{
  "compiler_version": "v0.4.0"
}`),
		Features: []byte(`{
  "version": "v0.4.0",
  "features": [
    {"id":"language.callable-level1","status":"current","since":"v0.4.0"}
  ]
}`),
		Targets: []byte(`{
  "targets": []
}`),
		ScopeDecisions: readinessScopeDecisionsJSON(
			"full-production-scope-selected",
			featureDecision("language.callable-level1", decisionEvidence{
				Implementation: []string{"../outside.go", "compiler/does-not-exist.go"},
				Tests:          []string{"go test ./compiler/... -run Callable -count=1"},
				Docs:           []string{"README.md", "docs/spec/does-not-exist.md"},
				ReleaseGateEvidence: []string{
					"reports/v0.4.0/callable.json",
				},
			}),
		),
	})
	if err == nil {
		t.Fatalf("expected unsafe/missing implementation or docs evidence to fail readiness")
	}
	got := err.Error()
	for _, want := range []string{
		"decision language.callable-level1 evidence.implementation path ../outside.go is unsafe",
		("decision language.callable-level1 evidence.implementation path " +
			"compiler/does-not-exist.go is not readable"),
		"decision language.callable-level1 evidence.docs path README.md must be under docs/",
		("decision language.callable-level1 evidence.docs path docs/spec/" +
			"does-not-exist.md is not readable"),
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("readiness error missing %q:\n%s", want, got)
		}
	}
}

func TestValidateReadinessRejectsMissingTestOrReleaseGateEvidencePaths(t *testing.T) {
	chdirReadinessEvidenceRoot(t,
		"compiler/internal/frontend/frontend_core.go",
		"docs/spec/core/current_supported_surface.md",
	)

	err := validateReadiness(readinessInputs{
		ExpectedVersion: "v0.4.0",
		Manifest: []byte(`{
  "compiler_version": "v0.4.0"
}`),
		Features: []byte(`{
  "version": "v0.4.0",
  "features": [
    {"id":"language.callable-level1","status":"current","since":"v0.4.0"}
  ]
}`),
		Targets: []byte(`{
  "targets": []
}`),
		ScopeDecisions: readinessScopeDecisionsJSON(
			"full-production-scope-selected",
			featureDecision("language.callable-level1", decisionEvidence{
				Implementation:      []string{"compiler/internal/frontend/frontend_core.go"},
				Tests:               []string{"compiler/tests/does-not-exist.go"},
				Docs:                []string{"docs/spec/core/current_supported_surface.md"},
				ReleaseGateEvidence: []string{"reports/v0.4.0/missing.json"},
			}),
		),
	})
	if err == nil {
		t.Fatalf("expected missing test and release-gate evidence paths to fail readiness")
	}
	got := err.Error()
	for _, want := range []string{
		("decision language.callable-level1 evidence.tests path compiler/" +
			"tests/does-not-exist.go is not readable"),
		("decision language.callable-level1 evidence.release_gate_" +
			"evidence path reports/v0.4.0/missing.json is not readable"),
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("readiness error missing %q:\n%s", want, got)
		}
	}
}

func TestValidateReadinessRejectsReleaseGateEvidenceWithoutReportArtifact(t *testing.T) {
	chdirReadinessEvidenceRoot(t,
		"compiler/internal/frontend/frontend_core.go",
		"docs/spec/core/current_supported_surface.md",
		"docs/checklists/v0_4_0_release_gate.md",
	)

	err := validateReadiness(readinessInputs{
		ExpectedVersion: "v0.4.0",
		Manifest: []byte(`{
  "compiler_version": "v0.4.0"
}`),
		Features: []byte(`{
  "version": "v0.4.0",
  "features": [
    {"id":"language.callable-level1","status":"current","since":"v0.4.0"}
  ]
}`),
		Targets: []byte(`{
  "targets": []
}`),
		ScopeDecisions: readinessScopeDecisionsJSON(
			"full-production-scope-selected",
			featureDecision("language.callable-level1", decisionEvidence{
				Implementation: []string{"compiler/internal/frontend/frontend_core.go"},
				Tests:          []string{"go test ./compiler -run Callable -count=1"},
				Docs:           []string{"docs/spec/core/current_supported_surface.md"},
				ReleaseGateEvidence: []string{
					"docs/checklists/v0_4_0_release_gate.md",
				},
			}),
		),
	})
	if err == nil {
		t.Fatalf("expected release-gate evidence without a report artifact to fail readiness")
	}
	if !strings.Contains(
		err.Error(),
		("decision language.callable-level1 missing evidence.release_gate_" +
			"evidence report artifact under reports/"),
	) {
		t.Fatalf("unexpected readiness error: %v", err)
	}
}

func TestValidateReadinessRejectsFakeReleaseGateReport(t *testing.T) {
	chdirReadinessEvidenceRoot(t,
		"compiler/internal/frontend/frontend_core.go",
		"docs/spec/core/current_supported_surface.md",
	)
	writeReadinessEvidenceFile(t, "reports/v0.4.0/evidence.json", []byte(`{"status":"fake"}`+"\n"))

	err := validateReadiness(readinessInputs{
		ExpectedVersion: "v0.4.0",
		Manifest: []byte(`{
  "compiler_version": "v0.4.0"
}`),
		Features: []byte(`{
  "version": "v0.4.0",
  "features": [
    {"id":"language.callable-level1","status":"current","since":"v0.4.0"}
  ]
}`),
		Targets: []byte(`{
  "targets": []
}`),
		ScopeDecisions: callableScopeWithReleaseGate("reports/v0.4.0/evidence.json"),
	})
	if err == nil {
		t.Fatalf("expected fake release-gate report to fail readiness")
	}
	if !strings.Contains(
		err.Error(),
		("decision language.callable-level1 evidence.release_gate_" +
			"evidence path reports/v0.4.0/evidence.json contains forbidden filler " +
			"wording"),
	) {
		t.Fatalf("unexpected readiness error: %v", err)
	}
}

func TestValidateReadinessRejectsEmptyReleaseGateReport(t *testing.T) {
	chdirReadinessEvidenceRoot(t,
		"compiler/internal/frontend/frontend_core.go",
		"docs/spec/core/current_supported_surface.md",
	)
	writeReadinessEvidenceFile(t, "reports/v0.4.0/evidence.json", []byte(`{}`+"\n"))

	err := validateReadiness(readinessInputs{
		ExpectedVersion: "v0.4.0",
		Manifest: []byte(`{
  "compiler_version": "v0.4.0"
}`),
		Features: []byte(`{
  "version": "v0.4.0",
  "features": [
    {"id":"language.callable-level1","status":"current","since":"v0.4.0"}
  ]
}`),
		Targets: []byte(`{
  "targets": []
}`),
		ScopeDecisions: callableScopeWithReleaseGate("reports/v0.4.0/evidence.json"),
	})
	if err == nil {
		t.Fatalf("expected empty release-gate report to fail readiness")
	}
	if !strings.Contains(
		err.Error(),
		("decision language.callable-level1 evidence.release_gate_" +
			"evidence path reports/v0.4.0/evidence.json is incomplete"),
	) {
		t.Fatalf("unexpected readiness error: %v", err)
	}
}

func TestValidateReadinessRejectsRuntimeTargetWithNonSupportedStatus(t *testing.T) {
	err := validateReadiness(readinessInputs{
		ExpectedVersion: "v0.4.0",
		Manifest: []byte(`{
  "compiler_version": "v0.4.0"
}`),
		Features: []byte(`{
  "version": "v0.4.0",
  "features": [
    {"id":"language.callable-level1","status":"current","since":"v0.4.0"}
  ]
}`),
		Targets: readinessTargetsJSON(readinessTarget{
			Triple:       "wasm32-web",
			Status:       "planned",
			RunSupported: true,
		}),
		ScopeDecisions: readinessScopeDecisionsJSON(
			"full-production-scope-selected",
			featureDecisionWithoutEvidence("language.callable-level1"),
			runtimeDecisionWithoutEvidence("wasm32-web"),
		),
	})
	if err == nil {
		t.Fatalf("expected non-supported runtime target status to fail readiness")
	}
	if !strings.Contains(err.Error(), "target wasm32-web status = planned, want supported") {
		t.Fatalf("unexpected readiness error: %v", err)
	}
}

type readinessFeature struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Since  string `json:"since,omitempty"`
}

type readinessTarget struct {
	Triple               string `json:"triple"`
	Status               string `json:"status"`
	BuildOnly            bool   `json:"build_only"`
	RunSupported         bool   `json:"run_supported"`
	RunUnsupportedReason string `json:"run_unsupported_reason,omitempty"`
}

type readinessDecision struct {
	Kind     string            `json:"kind"`
	ID       string            `json:"id"`
	Decision string            `json:"decision"`
	Evidence *decisionEvidence `json:"evidence,omitempty"`
}

func readinessFeaturesJSON(version string, features ...readinessFeature) []byte {
	return readinessJSON(map[string]any{
		"schema":   "tetra.features.v1",
		"version":  version,
		"features": features,
	})
}

func readinessTargetsJSON(targets ...readinessTarget) []byte {
	return readinessJSON(map[string]any{"targets": targets})
}

func readinessScopeDecisionsJSON(status string, decisions ...readinessDecision) []byte {
	return readinessJSON(map[string]any{
		"release_version": "v0.4.0",
		"status":          status,
		"decisions":       decisions,
	})
}

func currentFeature(id string) readinessFeature {
	return readinessFeature{ID: id, Status: "current", Since: "v0.4.0"}
}

func featureDecision(id string, evidence decisionEvidence) readinessDecision {
	return readinessDecision{
		Kind:     "feature",
		ID:       id,
		Decision: "implement",
		Evidence: &evidence,
	}
}

func featureDecisionWithoutEvidence(id string) readinessDecision {
	return readinessDecision{Kind: "feature", ID: id, Decision: "implement"}
}

func excludedFeatureDecision(id string) readinessDecision {
	return readinessDecision{
		Kind:     "feature",
		ID:       id,
		Decision: "exclude-from-v0.4.0-prod",
	}
}

func runtimeDecision(id string, evidence decisionEvidence) readinessDecision {
	return readinessDecision{
		Kind:     "target-runtime",
		ID:       id,
		Decision: "implement-production-runtime",
		Evidence: &evidence,
	}
}

func runtimeDecisionWithoutEvidence(id string) readinessDecision {
	return readinessDecision{
		Kind:     "target-runtime",
		ID:       id,
		Decision: "implement-production-runtime",
	}
}

func excludedRuntimeDecision(id string) readinessDecision {
	return readinessDecision{
		Kind:     "target-runtime",
		ID:       id,
		Decision: "exclude-from-v0.4.0-prod",
	}
}

func readinessJSON(value any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}

func linuxOnlyScopeDecisionsJSON() []byte {
	return readinessScopeDecisionsJSON(
		"linux-x64-production-scope-selected",
		featureDecision("language.callable-level1", decisionEvidence{
			Implementation: []string{"compiler/internal/frontend/frontend_core.go"},
			Tests:          []string{"go test ./compiler -run Callable -count=1"},
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
			},
			ReleaseGateEvidence: []string{"reports/v0.4.0/features.json"},
		}),
		featureDecision("language.callable-level2", decisionEvidence{
			Implementation: []string{"compiler/internal/semantics/semantics_checker.go"},
			Tests:          []string{"go test ./compiler -run Closure -count=1"},
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/policy/v1_feature_status.md",
			},
			ReleaseGateEvidence: []string{"reports/v0.4.0/features.json"},
		}),
		featureDecision("language.full-first-class-callables", decisionEvidence{
			Implementation: []string{"compiler/internal/lower/lower_core.go"},
			Tests:          []string{"go test ./compiler -run FullCallable -count=1"},
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/policy/v1_feature_status.md",
			},
			ReleaseGateEvidence: []string{"reports/v0.4.0/features.json"},
		}),
		featureDecision("language.lifetime-ssa", decisionEvidence{
			Implementation: []string{
				"compiler/internal/semantics/semantics_memory_resources.go",
			},
			Tests: []string{"go test ./compiler -run Lifetime -count=1"},
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/ownership_v1.md",
			},
			ReleaseGateEvidence: []string{"reports/v0.4.0/features.json"},
		}),
		featureDecision("stdlib.experimental-mirrors", decisionEvidence{
			Implementation: []string{"lib/experimental/base/math.tetra"},
			Tests:          []string{"go test ./compiler -run Experimental -count=1"},
			Docs: []string{
				"docs/spec/standard_library/stdlib.md",
				"docs/spec/standard_library/stdlib_naming_versioning.md",
				"docs/user/platform/standard_library_guide.md",
			},
			ReleaseGateEvidence: []string{"reports/v0.4.0/features.json"},
		}),
		featureDecision("ui.metadata-v1", decisionEvidence{
			Implementation: []string{"compiler/internal/backend/native_shell/codegen.go"},
			Tests:          []string{"go test ./compiler -run UI -count=1"},
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/ui/ui_v0.4.0.md",
			},
			ReleaseGateEvidence: []string{"reports/v0.4.0/native-ui-linux-x64.json"},
		}),
		featureDecision("ui.native-runtime", nativeUIRuntimeEvidence()),
		excludedFeatureDecision("language.full-v1-guarantees"),
		excludedFeatureDecision("eco.distributed-network"),
		excludedFeatureDecision("wasm.runtime-execution"),
		runtimeDecision("linux-x64", decisionEvidence{
			Implementation: []string{"compiler/internal/backend/native_shell/codegen.go"},
			Tests: []string{
				"./tetra smoke --target linux-x64 --run=true " +
					"--report reports/v0.4.0/linux-host-smoke.json",
			},
			Docs:                []string{"docs/spec/core/current_supported_surface.md"},
			ReleaseGateEvidence: []string{"reports/v0.4.0/features.json"},
		}),
		excludedRuntimeDecision("windows-x64"),
		excludedRuntimeDecision("macos-x64"),
		excludedRuntimeDecision("wasm32-wasi"),
		excludedRuntimeDecision("wasm32-web"),
	)
}

func nativeUIRuntimeEvidence() decisionEvidence {
	return decisionEvidence{
		Implementation: []string{
			"tools/cmd/native-ui-runtime-smoke/main.go",
			"tools/validators/nativeui/report.go",
		},
		Tests: []string{
			"go test ./tools/cmd/native-ui-runtime-smoke ./tools/validators/nativeui -count=1",
			"bash scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh",
		},
		Docs: []string{
			"docs/spec/core/current_supported_surface.md",
			"docs/spec/ui/ui_v0.4.0.md",
			"docs/user/surface/wasm_ui_guide.md",
		},
		ReleaseGateEvidence: []string{"reports/v0.4.0/native-ui-linux-x64.json"},
	}
}

func windowsUnsupportedTarget() readinessTarget {
	return readinessTarget{
		Triple:               "windows-x64",
		Status:               "supported",
		RunUnsupportedReason: "windows-x64 cannot run on host linux/amd64",
	}
}

func windowsRuntimeEvidence() decisionEvidence {
	return decisionEvidence{
		Implementation: []string{"compiler/internal/backend/windows_x64"},
		Tests: []string{
			"./tetra smoke --target windows-x64 --run=true " +
				"--report reports/v0.4.0/windows-runtime-smoke.json",
		},
		Docs:                []string{"docs/spec/core/current_supported_surface.md"},
		ReleaseGateEvidence: []string{"reports/v0.4.0/windows-runtime-smoke.json"},
	}
}

func wasmRuntimeEvidence() decisionEvidence {
	return decisionEvidence{
		Implementation: []string{"compiler/internal/backend/wasm32_web"},
		Tests: []string{
			"bash scripts/release/v1_0/web-smoke.sh --report reports/v0.4.0/web.json",
		},
		Docs:                []string{"docs/user/surface/wasm_ui_guide.md"},
		ReleaseGateEvidence: []string{"reports/v0.4.0/web.json"},
	}
}

func callableScopeWithReleaseGate(releaseGatePath string) []byte {
	return readinessScopeDecisionsJSON(
		"full-production-scope-selected",
		featureDecision("language.callable-level1", decisionEvidence{
			Implementation: []string{"compiler/internal/frontend/frontend_core.go"},
			Tests:          []string{"go test ./compiler -run Callable -count=1"},
			Docs:           []string{"docs/spec/core/current_supported_surface.md"},
			ReleaseGateEvidence: []string{
				releaseGatePath,
			},
		}),
	)
}

func readinessDistributedActorRuntimeJSON() []byte {
	zero := 0
	cases := []map[string]any{}
	for _, name := range []string{
		"cross-node i32 send/receive",
		"cross-node tagged send/receive",
		"cross-node typed send/receive",
		"missing-node failure/status",
		"task cancel/join compatibility",
	} {
		nodeProcesses := 2
		if name == "missing-node failure/status" || name == "task cancel/join compatibility" {
			nodeProcesses = 1
		}
		cases = append(cases, map[string]any{
			"name":           name,
			"ran":            true,
			"pass":           true,
			"expected_exit":  0,
			"actual_exit":    0,
			"node_processes": nodeProcesses,
		})
	}
	for _, name := range []string{
		"malformed frame length rejected",
		"duplicate node rejected",
		"unknown frame type rejected",
		"bad typed slot count rejected",
		"missing-node send after broker close",
		"forged source node rejected",
	} {
		cases = append(cases, map[string]any{
			"name":           name,
			"kind":           "network_negative",
			"ran":            true,
			"pass":           true,
			"expected_exit":  0,
			"actual_exit":    0,
			"node_processes": 0,
		})
	}
	raw, err := json.Marshal(map[string]any{
		"schema":          "tetra.actors.distributed-runtime.v1",
		"status":          "pass",
		"target":          "linux-x64",
		"host":            "linux-x64",
		"runtime":         "actornet",
		"transport":       "loopback-tcp",
		"git_head":        "e2c19b8ee276158f8eb2c54cf61e11bd84952893",
		"artifact_hashes": "artifact-hashes.json",
		"claims":          []string{"linux-x64 loopback tcp distributed actor runtime evidence"},
		"nonclaims": []string{
			"no cluster membership",
			"no reconnect/retry production",
			"no non-linux distributed actor runtime support",
		},
		"broker": map[string]any{
			"runtime":                "actornet",
			"transport":              "loopback-tcp",
			"listen_addr":            "127.0.0.1:47777",
			"accepted_connections":   8,
			"routed_frames":          5,
			"dropped_frames":         3,
			"decode_errors":          3,
			"expected_decode_errors": 3,
			"last_error":             "actor wire: invalid slot count: 9",
		},
		"processes": []map[string]any{
			{
				"name":      "broker",
				"kind":      "broker",
				"path":      "./tetra actor-net",
				"ran":       true,
				"pass":      true,
				"exit_code": zero,
			},
			{
				"name":      "node-a",
				"kind":      "node",
				"path":      "reports/v0.4.0/bin/node-a",
				"ran":       true,
				"pass":      true,
				"exit_code": zero,
			},
			{
				"name":      "node-b",
				"kind":      "node",
				"path":      "reports/v0.4.0/bin/node-b",
				"ran":       true,
				"pass":      true,
				"exit_code": zero,
			},
		},
		"frame_counts": map[string]any{
			"hello":      2,
			"hello_ack":  2,
			"spawn_req":  1,
			"spawn_ack":  1,
			"send_i32":   1,
			"send_msg":   1,
			"send_typed": 1,
			"node_down":  1,
			"error":      2,
		},
		"frame_order": []string{
			"hello",
			"hello_ack",
			"spawn_req",
			"spawn_ack",
			"send_i32",
			"send_msg",
			"send_typed",
			"node_down",
			"error",
			"error",
		},
		"cases": cases,
	})
	if err != nil {
		panic(err)
	}
	return raw
}

func readinessRuntimeSmokeJSON(target string, version string, ran bool) []byte {
	return readinessRuntimeSmokeJSONWithHost(target, target, version, ran)
}

func readinessRuntimeSmokeJSONWithHost(
	target string,
	host string,
	version string,
	ran bool,
) []byte {
	cases := []map[string]any{}
	for _, tc := range []struct {
		name string
		src  string
		exit int
	}{
		{name: "actors_pingpong", src: "examples/actors/actors_pingpong.tetra", exit: 0},
		{name: "actor_sleep_pingpong", src: "examples/actors/actor_sleep_pingpong.tetra", exit: 0},
		{name: "task_smoke", src: "examples/tasks/task_smoke.tetra", exit: 42},
		{name: "time_sleep_smoke", src: "examples/async/time_sleep_smoke.tetra", exit: 0},
		{
			name: "task_sleep_deadline_smoke",
			src:  "examples/tasks/task_sleep_deadline_smoke.tetra",
			exit: 0,
		},
		{name: "task_join_wait_smoke", src: "examples/tasks/task_join_wait_smoke.tetra", exit: 5},
		{
			name: "deadline_aware_waits_smoke",
			src:  "examples/async/deadline_aware_waits_smoke.tetra",
			exit: 0,
		},
		{name: "wait_composition_smoke", src: "examples/async/wait_composition_smoke.tetra", exit: 0},
	} {
		cases = append(cases, map[string]any{
			"name":          tc.name,
			"src_path":      tc.src,
			"out_path":      "/tmp/" + tc.name,
			"expected_exit": tc.exit,
			"actual_exit":   tc.exit,
			"ran":           ran,
			"pass":          true,
		})
	}
	raw, err := json.Marshal(map[string]any{
		"timestamp":     "2026-05-05T12:00:00Z",
		"target":        target,
		"host":          host,
		"version":       version,
		"git_head":      "abcdef0",
		"islands_debug": false,
		"total":         len(cases),
		"passed":        len(cases),
		"failed":        0,
		"cases":         cases,
	})
	if err != nil {
		panic(err)
	}
	return raw
}

func readinessNativeUIRuntimeJSON() []byte {
	raw, err := json.Marshal(map[string]any{
		"schema":    nativeui.SchemaV1,
		"status":    "pass",
		"target":    "linux-x64",
		"host":      "linux-x64",
		"runtime":   "native-ui-linux-x64",
		"ui_schema": "tetra.ui.v0.4.0",
		"source":    "examples/ui/ui_native_shell_smoke.tetra",
		"processes": []map[string]any{
			{
				"name":      "tetra build",
				"kind":      "build",
				"path":      "/tmp/tetra",
				"ran":       true,
				"pass":      true,
				"exit_code": 0,
			},
			{
				"name":      "native app",
				"kind":      "app",
				"path":      "/tmp/ui-native",
				"ran":       true,
				"pass":      true,
				"exit_code": 0,
			},
			{
				"name":      "native ui runtime",
				"kind":      "runtime",
				"path":      "tools/cmd/native-ui-runtime-smoke",
				"ran":       true,
				"pass":      true,
				"exit_code": 0,
			},
		},
		"widgets": []map[string]any{
			{
				"id":      "ShellView",
				"kind":    "view",
				"parent":  "",
				"enabled": true,
				"visible": true,
				"bounds":  map[string]any{"x": 0, "y": 0, "width": 320, "height": 96},
			},
			{
				"id":      "ShellView.toggles",
				"kind":    "value",
				"parent":  "ShellView",
				"binding": "toggles",
				"value":   "0",
				"enabled": true,
				"visible": true,
				"bounds":  map[string]any{"x": 8, "y": 8, "width": 304, "height": 24},
			},
			{
				"id":      "ShellView.submit",
				"kind":    "action",
				"parent":  "ShellView",
				"event":   "submit",
				"command": "toggle",
				"enabled": true,
				"visible": true,
				"bounds":  map[string]any{"x": 8, "y": 40, "width": 304, "height": 24},
			},
		},
		"events": []map[string]any{
			{
				"order":        1,
				"widget_id":    "ShellView.submit",
				"event":        "click",
				"command":      "toggle",
				"pass":         true,
				"before_state": map[string]any{"ShellState.toggles": "0"},
				"after_state":  map[string]any{"ShellState.toggles": "1"},
				"operations": []map[string]any{
					{
						"kind":        "state_add",
						"target":      "state.toggles",
						"value":       "1",
						"state_field": "toggles",
						"state_value": "1",
					},
				},
				"widget_updates": []map[string]any{
					{"id": "ShellView.toggles", "before": "0", "after": "1"},
				},
			},
			{
				"order":        2,
				"widget_id":    "ShellView.submit",
				"event":        "click",
				"command":      "toggle",
				"pass":         true,
				"before_state": map[string]any{"ShellState.toggles": "1"},
				"after_state":  map[string]any{"ShellState.toggles": "2"},
				"operations": []map[string]any{
					{
						"kind":        "state_add",
						"target":      "state.toggles",
						"value":       "1",
						"state_field": "toggles",
						"state_value": "2",
					},
				},
				"widget_updates": []map[string]any{
					{"id": "ShellView.toggles", "before": "1", "after": "2"},
				},
			},
		},
		"cases": []map[string]any{
			{"name": "load widget tree", "ran": true, "pass": true},
			{"name": "dispatch click command", "ran": true, "pass": true},
			{"name": "propagate state update", "ran": true, "pass": true},
			{"name": "dispatch multiple ordered events", "ran": true, "pass": true},
			{
				"name":           "reject invalid widget id",
				"ran":            true,
				"pass":           true,
				"expected_error": "unknown widget",
			},
			{
				"name":           "reject malformed metadata",
				"ran":            true,
				"pass":           true,
				"expected_error": "malformed metadata",
			},
			{
				"name":           "reject unsupported event kind",
				"ran":            true,
				"pass":           true,
				"expected_error": "unsupported event",
			},
			{
				"name":           "reject command failure",
				"ran":            true,
				"pass":           true,
				"expected_error": "unknown command",
			},
			{"name": "close runtime", "ran": true, "pass": true},
		},
	})
	if err != nil {
		panic(err)
	}
	return raw
}

func chdirReadinessEvidenceRoot(t *testing.T, paths ...string) {
	t.Helper()
	root := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir test evidence root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})
	for _, path := range paths {
		writeReadinessEvidenceFile(t, path, defaultReadinessEvidenceContent(path))
	}
}

func writeReadinessEvidenceFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if filepath.IsAbs(path) {
		t.Fatalf("test evidence path must be relative: %s", path)
	}
	fullPath := filepath.FromSlash(path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir test evidence: %v", err)
	}
	if err := os.WriteFile(fullPath, content, 0o644); err != nil {
		t.Fatalf("write test evidence: %v", err)
	}
}

func defaultReadinessEvidenceContent(path string) []byte {
	if strings.HasSuffix(path, ".json") {
		return []byte(`{"status":"pass"}` + "\n")
	}
	return []byte("ok\n")
}
