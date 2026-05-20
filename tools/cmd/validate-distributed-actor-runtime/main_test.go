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
	if err := validateDistributedActorRuntimeReport(reportPath); err != nil {
		t.Fatalf("validateDistributedActorRuntimeReport failed: %v", err)
	}
}

func TestValidateDistributedActorRuntimeReportRejectsThinPaperEvidence(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "distributed-actors.json")
	raw := []byte(`{"schema":"tetra.actors.distributed-runtime.v1","status":"pass","runtime":"compiler/internal/actorsrt/distributed_runtime.go","cases":[{"name":"cross-node send/receive","pass":true}]}`)
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateDistributedActorRuntimeReport(reportPath)
	if err == nil {
		t.Fatalf("expected thin report to fail")
	}
	for _, want := range []string{"target", "loopback", "process", "frame"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
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
  "broker": {
    "runtime": "actornet",
    "transport": "loopback-tcp",
    "listen_addr": "127.0.0.1:47777",
    "accepted_connections": 3,
    "routed_frames": 5,
    "dropped_frames": 1
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
    "node_down": 1
  },
  "cases": [
    {"name":"cross-node i32 send/receive","ran":true,"pass":true,"expected_exit":0,"actual_exit":0,"node_processes":2},
    {"name":"cross-node tagged send/receive","ran":true,"pass":true,"expected_exit":0,"actual_exit":0,"node_processes":2},
    {"name":"cross-node typed send/receive","ran":true,"pass":true,"expected_exit":0,"actual_exit":0,"node_processes":2},
    {"name":"missing-node failure/status","ran":true,"pass":true,"expected_exit":0,"actual_exit":0,"node_processes":1},
    {"name":"task cancel/join compatibility","ran":true,"pass":true,"expected_exit":0,"actual_exit":0,"node_processes":1}
  ]
}
`)
}
