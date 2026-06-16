package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateDistributedActorRuntimeReportAcceptsExecutableEvidence(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "distributed-actors.json")
	if err := os.WriteFile(reportPath, validDistributedActorRuntimeReportJSON(), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := validateDistributedActorRuntimeReport(reportPath, ""); err != nil {
		t.Fatalf("validateDistributedActorRuntimeReport failed: %v", err)
	}
	if err := validateDistributedActorRuntimeReport(reportPath, "e2c19b8ee276158f8eb2c54cf61e11bd84952893"); err != nil {
		t.Fatalf("validateDistributedActorRuntimeReport with matching head failed: %v", err)
	}
}

func TestValidateDistributedActorRuntimeReportRejectsThinPaperEvidence(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "distributed-actors.json")
	raw := []byte(`{"schema":"tetra.actors.distributed-runtime.v1","status":"pass","runtime":"compiler/internal/actorsrt/distributed_runtime.go","cases":[{"name":"cross-node send/receive","pass":true}]}`)
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateDistributedActorRuntimeReport(reportPath, "")
	if err == nil {
		t.Fatalf("expected thin report to fail")
	}
	for _, want := range []string{"target", "loopback", "process", "frame"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateDistributedActorRuntimeReportRejectsStaleCurrentGitHead(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "distributed-actors.json")
	if err := os.WriteFile(reportPath, validDistributedActorRuntimeReportJSON(), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateDistributedActorRuntimeReport(reportPath, "c0258b63a636775b114d69d31cb7832fc3991b05")
	if err == nil {
		t.Fatalf("expected stale current git head to fail")
	}
	if !strings.Contains(err.Error(), "does not match current git head") {
		t.Fatalf("error = %v, want stale git_head rejection", err)
	}
}

func validDistributedActorRuntimeReportJSON() []byte {
	return []byte(`{
  "schema": "tetra.actors.distributed-runtime.v1",
  "status": "pass",
  "target": "linux-x64",
  "host": "linux-x64",
  "runtime": "actornet",
  "transport": "loopback-tcp",
  "git_head": "e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "artifact_hashes": "artifact-hashes.json",
  "claims": ["linux-x64 loopback tcp distributed actor runtime evidence"],
  "nonclaims": [
    "no cluster membership",
    "no reconnect/retry production",
    "no non-linux distributed actor runtime support"
  ],
  "broker": {
    "runtime": "actornet",
    "transport": "loopback-tcp",
    "listen_addr": "127.0.0.1:47777",
    "accepted_connections": 8,
    "routed_frames": 5,
    "dropped_frames": 3,
    "decode_errors": 3,
    "expected_decode_errors": 3,
    "last_error": "actor wire: invalid slot count: 9"
  },
  "processes": [
    {"name":"broker","kind":"broker","path":"./tetra actor-net","ran":true,"pass":true,"exit_code":0},
    {"name":"node-a","kind":"node","path":"reports/v0.4.0/bin/node-a","ran":true,"pass":true,"exit_code":0},
    {"name":"node-b","kind":"node","path":"reports/v0.4.0/bin/node-b","ran":true,"pass":true,"exit_code":0}
  ],
  "frame_counts": {
    "hello": 2,
    "hello_ack": 2,
    "spawn_req": 1,
    "spawn_ack": 1,
    "send_i32": 1,
    "send_msg": 1,
    "send_typed": 1,
    "node_down": 1,
    "error": 2
  },
  "frame_order": ["hello","hello_ack","spawn_req","spawn_ack","send_i32","send_msg","send_typed","node_down","error","error"],
  "cases": [
    {"name":"cross-node i32 send/receive","ran":true,"pass":true,"expected_exit":0,"actual_exit":0,"node_processes":2},
    {"name":"cross-node tagged send/receive","ran":true,"pass":true,"expected_exit":0,"actual_exit":0,"node_processes":2},
    {"name":"cross-node typed send/receive","ran":true,"pass":true,"expected_exit":0,"actual_exit":0,"node_processes":2},
    {"name":"missing-node failure/status","ran":true,"pass":true,"expected_exit":0,"actual_exit":0,"node_processes":1},
    {"name":"task cancel/join compatibility","ran":true,"pass":true,"expected_exit":0,"actual_exit":0,"node_processes":1},
    {"name":"malformed frame length rejected","kind":"network_negative","ran":true,"pass":true,"expected_exit":0,"actual_exit":0,"node_processes":0},
    {"name":"duplicate node rejected","kind":"network_negative","ran":true,"pass":true,"expected_exit":0,"actual_exit":0,"node_processes":0},
    {"name":"unknown frame type rejected","kind":"network_negative","ran":true,"pass":true,"expected_exit":0,"actual_exit":0,"node_processes":0},
    {"name":"bad typed slot count rejected","kind":"network_negative","ran":true,"pass":true,"expected_exit":0,"actual_exit":0,"node_processes":0},
    {"name":"missing-node send after broker close","kind":"network_negative","ran":true,"pass":true,"expected_exit":0,"actual_exit":0,"node_processes":0},
    {"name":"forged source node rejected","kind":"network_negative","ran":true,"pass":true,"expected_exit":0,"actual_exit":0,"node_processes":0}
  ]
}
`)
}
