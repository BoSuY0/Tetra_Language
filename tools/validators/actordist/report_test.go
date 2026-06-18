package actordist

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateReportAcceptsExecutableLinuxX64Evidence(t *testing.T) {
	raw := validDistributedActorRuntimeReport(t)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsThinPaperEvidence(t *testing.T) {
	raw := []byte(
		`{"schema":"tetra.actors.distributed-runtime.v1","status":"pass","runtime":"compiler/internal/actorsrt/distributed_runtime.go","cases":[{"name":"cross-node send/receive","pass":true},{"name":"failure/cancel/join diagnostics","pass":true}]}`,
	)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected thin report to fail")
	}
	for _, want := range []string{"target", "loopback", "process", "frame"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateReportRejectsMissingFailureCase(t *testing.T) {
	raw := validDistributedActorRuntimeReportFrom(t, func(report *Report) {
		report.Cases = report.Cases[:3]
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing failure case to fail")
	}
	if !strings.Contains(err.Error(), "missing required case missing-node failure/status") {
		t.Fatalf("error = %v, want missing failure case", err)
	}
}

func TestValidateReportRejectsMissingNegativeNetworkCases(t *testing.T) {
	raw := validDistributedActorRuntimeReportFrom(t, func(report *Report) {
		var kept []CaseReport
		for _, c := range report.Cases {
			if !isP11NegativeNetworkCase(c.Name) {
				kept = append(kept, c)
			}
		}
		report.Cases = kept
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing P11 negative network cases to fail")
	}
	for _, want := range []string{
		"malformed frame length",
		"duplicate node",
		"unknown frame type",
		"bad typed slot count",
		"broker close",
	} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateReportRejectsUnexpectedDecodeErrors(t *testing.T) {
	raw := validDistributedActorRuntimeReportFrom(t, func(report *Report) {
		report.Broker.ExpectedDecodeErrors = 0
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected undeclared decode errors to fail")
	}
	if !strings.Contains(err.Error(), "expected malformed-frame evidence") {
		t.Fatalf("error = %v, want expected malformed-frame evidence rejection", err)
	}
}

func TestValidateReportRejectsMissingSameCommitArtifactMetadata(t *testing.T) {
	var report map[string]any
	if err := json.Unmarshal(validDistributedActorRuntimeReport(t), &report); err != nil {
		t.Fatal(err)
	}
	delete(report, "git_head")
	delete(report, "artifact_hashes")
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}

	err = ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing same-commit artifact metadata to fail")
	}
	if !strings.Contains(err.Error(), "git_head") {
		t.Fatalf("error = %v, want missing git_head", err)
	}
	if !strings.Contains(err.Error(), "artifact_hashes") {
		t.Fatalf("error = %v, want missing artifact_hashes", err)
	}
}

func TestValidateReportForCurrentHeadRejectsStaleGitHead(t *testing.T) {
	raw := validDistributedActorRuntimeReport(t)
	err := ValidateReportForCurrentHead(raw, "c0258b63a636775b114d69d31cb7832fc3991b05")
	if err == nil {
		t.Fatalf("expected stale git_head to fail")
	}
	if !strings.Contains(err.Error(), "does not match current git head") {
		t.Fatalf("error = %v, want stale git_head rejection", err)
	}
	if err := ValidateReportForCurrentHead(
		raw,
		"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
	); err != nil {
		t.Fatalf("matching current head should pass: %v", err)
	}
}

func TestValidateReportRejectsMissingFrameOrder(t *testing.T) {
	var report map[string]any
	if err := json.Unmarshal(validDistributedActorRuntimeReport(t), &report); err != nil {
		t.Fatal(err)
	}
	delete(report, "frame_order")
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}

	err = ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing frame_order to fail")
	}
	if !strings.Contains(err.Error(), "frame_order") {
		t.Fatalf("error = %v, want missing frame_order", err)
	}
}

func TestValidateReportRejectsBadFrameOrder(t *testing.T) {
	raw := validDistributedActorRuntimeReportFrom(t, func(report *Report) {
		report.FrameOrder = []string{
			"send_typed",
			"send_msg",
			"send_i32",
			"spawn_ack",
			"spawn_req",
			"hello_ack",
			"hello",
			"node_down",
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected bad frame_order to fail")
	}
	if !strings.Contains(err.Error(), "frame_order") {
		t.Fatalf("error = %v, want frame_order rejection", err)
	}
}

func TestValidateReportRejectsMissingScopedNonClaims(t *testing.T) {
	var report map[string]any
	if err := json.Unmarshal(validDistributedActorRuntimeReport(t), &report); err != nil {
		t.Fatal(err)
	}
	delete(report, "nonclaims")
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}

	err = ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing nonclaims to fail")
	}
	if !strings.Contains(err.Error(), "nonclaims") {
		t.Fatalf("error = %v, want missing nonclaims", err)
	}
}

func TestValidateReportRejectsClusterRetryReconnectClaims(t *testing.T) {
	var report map[string]any
	if err := json.Unmarshal(validDistributedActorRuntimeReport(t), &report); err != nil {
		t.Fatal(err)
	}
	report["claims"] = []string{"cluster reconnect retry production for non-linux actors"}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}

	err = ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected forbidden distributed actor claim to fail")
	}
	if !strings.Contains(err.Error(), "forbidden distributed actor claim") {
		t.Fatalf("error = %v, want forbidden distributed actor claim", err)
	}
}

func TestValidateReportRejectsTransportOnlyClaim(t *testing.T) {
	raw := validDistributedActorRuntimeReportFrom(t, func(report *Report) {
		report.Claims = []string{"linux-x64 transport-only actor wire evidence"}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected transport-only claim to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "transport-only") {
		t.Fatalf("error = %v, want transport-only rejection", err)
	}
}

func validDistributedActorRuntimeReport(t *testing.T) []byte {
	t.Helper()
	return validDistributedActorRuntimeReportFrom(t, func(*Report) {})
}

func validDistributedActorRuntimeReportFrom(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	zero := 0
	report := Report{
		Schema:         SchemaV1,
		Status:         "pass",
		Target:         "linux-x64",
		Host:           "linux-x64",
		Runtime:        "actornet",
		Transport:      "loopback-tcp",
		GitHead:        "e2c19b8ee276158f8eb2c54cf61e11bd84952893",
		ArtifactHashes: "artifact-hashes.json",
		Claims:         []string{"linux-x64 loopback tcp distributed actor runtime evidence"},
		NonClaims: []string{
			"no cluster membership",
			"no reconnect/retry production",
			"no non-linux distributed actor runtime support",
		},
		Broker: BrokerReport{
			Runtime:              "actornet",
			Transport:            "loopback-tcp",
			ListenAddr:           "127.0.0.1:47777",
			AcceptedConnections:  8,
			RoutedFrames:         5,
			DroppedFrames:        3,
			DecodeErrors:         3,
			ExpectedDecodeErrors: 3,
			LastError:            "actor wire: invalid slot count: 9",
		},
		Processes: []ProcessReport{
			{
				Name:     "broker",
				Kind:     "broker",
				Path:     "./tetra actor-net",
				Ran:      true,
				Pass:     true,
				ExitCode: &zero,
			},
			{
				Name:     "node-a",
				Kind:     "node",
				Path:     "reports/v0.4.0/bin/node-a",
				Ran:      true,
				Pass:     true,
				ExitCode: &zero,
			},
			{
				Name:     "node-b",
				Kind:     "node",
				Path:     "reports/v0.4.0/bin/node-b",
				Ran:      true,
				Pass:     true,
				ExitCode: &zero,
			},
		},
		FrameCounts: FrameCounts{
			Hello:     2,
			HelloAck:  2,
			SpawnReq:  1,
			SpawnAck:  1,
			SendI32:   1,
			SendMsg:   1,
			SendTyped: 1,
			NodeDown:  1,
			Error:     2,
		},
		FrameOrder: []string{
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
		Cases: []CaseReport{
			validCase("cross-node i32 send/receive", 2),
			validCase("cross-node tagged send/receive", 2),
			validCase("cross-node typed send/receive", 2),
			validCase("missing-node failure/status", 1),
			validCase("task cancel/join compatibility", 1),
			validNetworkNegativeCase("malformed frame length rejected"),
			validNetworkNegativeCase("duplicate node rejected"),
			validNetworkNegativeCase("unknown frame type rejected"),
			validNetworkNegativeCase("bad typed slot count rejected"),
			validNetworkNegativeCase("missing-node send after broker close"),
			validNetworkNegativeCase("forged source node rejected"),
		},
	}
	mutate(&report)
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func validCase(name string, nodeProcesses int) CaseReport {
	zero := 0
	return CaseReport{
		Name:          name,
		Ran:           true,
		Pass:          true,
		ExpectedExit:  0,
		ActualExit:    &zero,
		NodeProcesses: nodeProcesses,
	}
}

func validNetworkNegativeCase(name string) CaseReport {
	zero := 0
	return CaseReport{
		Name:          name,
		Kind:          "network_negative",
		Ran:           true,
		Pass:          true,
		ExpectedExit:  0,
		ActualExit:    &zero,
		NodeProcesses: 0,
	}
}

func isP11NegativeNetworkCase(name string) bool {
	switch name {
	case "malformed frame length rejected",
		"duplicate node rejected",
		"unknown frame type rejected",
		"bad typed slot count rejected",
		"missing-node send after broker close",
		"forged source node rejected":
		return true
	default:
		return false
	}
}
