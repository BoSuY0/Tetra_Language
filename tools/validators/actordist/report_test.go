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
	raw := []byte(`{"schema":"tetra.actors.distributed-runtime.v1","status":"pass","runtime":"compiler/internal/actorsrt/distributed_runtime.go","cases":[{"name":"cross-node send/receive","pass":true},{"name":"failure/cancel/join diagnostics","pass":true}]}`)
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

func validDistributedActorRuntimeReport(t *testing.T) []byte {
	t.Helper()
	return validDistributedActorRuntimeReportFrom(t, func(*Report) {})
}

func validDistributedActorRuntimeReportFrom(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	zero := 0
	report := Report{
		Schema:    SchemaV1,
		Status:    "pass",
		Target:    "linux-x64",
		Host:      "linux-x64",
		Runtime:   "actornet",
		Transport: "loopback-tcp",
		Broker: BrokerReport{
			Runtime:             "actornet",
			Transport:           "loopback-tcp",
			ListenAddr:          "127.0.0.1:47777",
			AcceptedConnections: 3,
			RoutedFrames:        5,
			DroppedFrames:       1,
		},
		Processes: []ProcessReport{
			{Name: "broker", Kind: "broker", Path: "./tetra actor-net", Ran: true, Pass: true, ExitCode: &zero},
			{Name: "node-a", Kind: "node", Path: "reports/v0.4.0/bin/node-a", Ran: true, Pass: true, ExitCode: &zero},
			{Name: "node-b", Kind: "node", Path: "reports/v0.4.0/bin/node-b", Ran: true, Pass: true, ExitCode: &zero},
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
		},
		Cases: []CaseReport{
			validCase("cross-node i32 send/receive", 2),
			validCase("cross-node tagged send/receive", 2),
			validCase("cross-node typed send/receive", 2),
			validCase("missing-node failure/status", 1),
			validCase("task cancel/join compatibility", 1),
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
