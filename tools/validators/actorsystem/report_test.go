package actorsystem

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testGitHead = "e2c19b8ee276158f8eb2c54cf61e11bd84952893"

func TestValidateReportAcceptsP01FixtureEvidence(t *testing.T) {
	raw := validActorSystemMessagesReport(t)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsProductionProducerForP01FixtureEvidence(t *testing.T) {
	raw := validActorSystemMessagesReportFrom(t, func(report *Report) {
		report.Producer = "production"
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected production producer rejection")
	}
	if !strings.Contains(err.Error(), "producer") {
		t.Fatalf("error = %v, want producer rejection", err)
	}
}

func TestValidateReportRejectsExportedTestInjector(t *testing.T) {
	raw := validActorSystemMessagesReportFrom(t, func(report *Report) {
		report.Security.ReleaseTestInjectorExported = true
		report.ReleaseSymbolScan.TestInjectorExported = true
		report.ReleaseSymbolScan.ExportedSymbols = []string{"__tetra_test_actor_system_inject"}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected test injector export rejection")
	}
	if !strings.Contains(err.Error(), "__tetra_test_actor_system_inject") {
		t.Fatalf("error = %v, want test injector symbol rejection", err)
	}
}

func TestValidateReportRejectsNodeDownProductionClaimInP01(t *testing.T) {
	raw := validActorSystemMessagesReportFrom(t, func(report *Report) {
		report.Claims = []string{"authenticated node-down producer is production ready"}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected node-down production claim rejection")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "node-down") {
		t.Fatalf("error = %v, want node-down claim rejection", err)
	}
}

func TestValidateReportRejectsMissingRequiredNonclaim(t *testing.T) {
	raw := validActorSystemMessagesReportFrom(t, func(report *Report) {
		report.NonClaims = []string{
			"real local link/monitor producers are completed in P06",
			"authenticated node-down producer is completed in P10",
			"no full Erlang/OTP actor runtime claim",
			"no cluster membership or reconnect/retry production claim",
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing distributed nonclaims rejection")
	}
	if !strings.Contains(err.Error(), "zero-copy") {
		t.Fatalf("error = %v, want zero-copy nonclaim rejection", err)
	}
}

func TestValidateReportDirCrossChecksHashManifest(t *testing.T) {
	dir := t.TempDir()
	writeActorSystemMessagesFixtureDir(t, dir)
	if err := ValidateReportDir(dir, Options{CurrentGitHead: testGitHead}); err != nil {
		t.Fatalf("ValidateReportDir failed: %v", err)
	}
}

func TestValidateReportDirRejectsMissingHashManifest(t *testing.T) {
	dir := t.TempDir()
	writeActorSystemMessagesFixtureDir(t, dir)
	if err := os.Remove(filepath.Join(dir, "artifact-hashes.json")); err != nil {
		t.Fatalf("remove hash manifest: %v", err)
	}
	err := ValidateReportDir(dir, Options{CurrentGitHead: testGitHead})
	if err == nil {
		t.Fatalf("expected missing artifact-hashes.json rejection")
	}
	if !strings.Contains(err.Error(), "artifact-hashes.json") {
		t.Fatalf("error = %v, want artifact-hashes.json", err)
	}
}

func TestValidateReportDirRejectsMissingLayoutReport(t *testing.T) {
	dir := t.TempDir()
	writeActorSystemMessagesFixtureDir(t, dir)
	if err := os.Remove(filepath.Join(dir, "actor-system-layout-linux-x64.json")); err != nil {
		t.Fatalf("remove layout report: %v", err)
	}
	writeArtifactHashManifest(t, dir, []testArtifact{
		{Path: "actor-system-messages-linux-x64.json", Schema: SchemaV1},
		{Path: "bin/system_user_queue_isolation"},
	})
	err := ValidateReportDir(dir, Options{CurrentGitHead: testGitHead})
	if err == nil {
		t.Fatalf("expected missing layout report rejection")
	}
	if !strings.Contains(err.Error(), "actor-system-layout-linux-x64.json") {
		t.Fatalf("error = %v, want actor-system-layout-linux-x64.json", err)
	}
}

func TestValidateReportDirRejectsStaleGitHead(t *testing.T) {
	dir := t.TempDir()
	writeActorSystemMessagesFixtureDir(t, dir)
	err := ValidateReportDir(dir, Options{CurrentGitHead: strings.Repeat("a", 40)})
	if err == nil {
		t.Fatalf("expected git head mismatch rejection")
	}
	if !strings.Contains(err.Error(), "git_head") {
		t.Fatalf("error = %v, want git_head rejection", err)
	}
}

func validActorSystemMessagesReport(t *testing.T) []byte {
	t.Helper()
	return validActorSystemMessagesReportFrom(t, func(*Report) {})
}

func validActorSystemMessagesReportFrom(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	report := Report{
		Schema:         SchemaV1,
		Pass:           true,
		Target:         "linux-x64",
		Host:           "linux-x64",
		Runtime:        "builtin-actor-runtime-v2",
		GitHead:        testGitHead,
		GitDirty:       true,
		Design:         "separate-system-lane-v1",
		Producer:       "test_hook",
		ReportDir:      ".",
		ArtifactHashes: "artifact-hashes.json",
		CommandLine:    "bash scripts/release/v1_0/actor-system-messages-linux-x64-smoke.sh --report-dir reports/v1-actor-system-messages/test",
		Claims: []string{
			"source-level system-message API and isolated runtime system lane implemented for Linux-x64 builtin runtime",
		},
		NonClaims: []string{
			"real local link/monitor producers are completed in P06",
			"authenticated node-down producer is completed in P10",
			"no full Erlang/OTP actor runtime claim",
			"no cluster membership or reconnect/retry production claim",
			"no non-Linux distributed actor runtime support claim",
			"no distributed zero-copy pointer or region transfer claim",
			"no formal race proof claim",
		},
		API: APICoverage{
			RecvSystem:      true,
			PollSystem:      true,
			RecvSystemUntil: true,
		},
		Isolation: IsolationReport{
			SeparateHeadsTails:         true,
			UserRecvSystemConsumptions: 0,
			SystemRecvUserConsumptions: 0,
			UserQueueFIFOViolations:    0,
			SystemQueueFIFOViolations:  0,
			SenderUnchanged:            true,
		},
		Security: SecurityReport{
			OrdinarySendForgeryRejected: true,
			RuntimeHandlesOpaque:        true,
			ReleaseTestInjectorExported: false,
		},
		Events: EventsReport{
			Exit:            1,
			Down:            1,
			NodeDownFixture: 1,
			DuplicateDown:   0,
			Producer:        "test_hook",
		},
		Memory: MemoryReport{
			Bounded:                true,
			ReservedCredits:        0,
			LiveBytesAfterShutdown: 0,
			SilentDrops:            0,
		},
		ReleaseSymbolScan: ReleaseSymbolScan{
			Scanned:              true,
			Binary:               "bin/system_user_queue_isolation",
			ForbiddenSymbols:     []string{"__tetra_test_actor_system_inject"},
			TestInjectorExported: false,
		},
		Commands: []CommandReport{
			{Name: "focused-validator-tests", Command: "go test -buildvcs=false ./tools/cmd/validate-actor-system-messages ./tools/validators/actorsystem -count=1", Status: "pass", Log: "logs/focused-validator-tests.log"},
			{Name: "actor-system-message-validator", Command: "go run -buildvcs=false ./tools/cmd/validate-actor-system-messages --root .", Status: "pass", Log: "logs/actor-system-message-validator.log"},
			{Name: "generated-examples-build-run", Command: "go run -buildvcs=false ./cli/cmd/tetra build --target linux-x64 examples/actors/system_messages/system_user_queue_isolation.tetra", Status: "pass", Log: "logs/generated-examples-build-run.log"},
			{Name: "negative-forgery-check", Command: "go run -buildvcs=false ./cli/cmd/tetra check examples/actors/system_messages/system_forgery_negative.tetra", Status: "pass", Log: "logs/negative-forgery-check.log"},
			{Name: "actor-system-layout-report", Command: "go run -buildvcs=false ./compiler/cmd/actor-system-layout-report --out reports/v1-actor-system-messages/test/actor-system-layout-linux-x64.json", Status: "pass", Log: "logs/actor-system-layout-report.log"},
			{Name: "release-symbol-scan", Command: "nm -a bin/system_user_queue_isolation", Status: "pass", Log: "logs/release-symbol-scan.log"},
			{Name: "artifact-hashes-write", Command: "go run ./tools/cmd/validate-artifact-hashes --write --root reports/v1-actor-system-messages/test --out reports/v1-actor-system-messages/test/artifact-hashes.json", Status: "pass", Log: "logs/artifact-hashes-write.log"},
			{Name: "artifact-hashes-validate", Command: "go run ./tools/cmd/validate-artifact-hashes --manifest reports/v1-actor-system-messages/test/artifact-hashes.json", Status: "pass", Log: "stdout"},
		},
		Artifacts: []ArtifactReport{
			{Path: "actor-system-messages-linux-x64.json", Kind: "actor_system_messages_report", Schema: SchemaV1},
			{Path: "actor-system-layout-linux-x64.json", Kind: "actor_system_layout_report", Schema: LayoutSchemaV1},
			{Path: "artifact-hashes.json", Kind: "artifact_hash_manifest", Schema: ArtifactHashSchema},
			{Path: "bin/system_user_queue_isolation", Kind: "native_binary"},
		},
	}
	mutate(&report)
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func writeActorSystemMessagesFixtureDir(t *testing.T, dir string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, "actor-system-messages-linux-x64.json"), string(validActorSystemMessagesReport(t)))
	writeFile(t, filepath.Join(dir, "actor-system-layout-linux-x64.json"), validLayoutReportJSON)
	writeFile(t, filepath.Join(dir, "bin", "system_user_queue_isolation"), "#!/bin/sh\nexit 0\n")
	writeArtifactHashManifest(t, dir, []testArtifact{
		{Path: "actor-system-messages-linux-x64.json", Schema: SchemaV1},
		{Path: "actor-system-layout-linux-x64.json", Schema: LayoutSchemaV1},
		{Path: "bin/system_user_queue_isolation"},
	})
}

const validLayoutReportJSON = `{
  "schema": "tetra.actor.system_layout.v1",
  "target": "linux-x64",
  "runtime": "builtin-actor-runtime-v2",
  "actor": {
    "name": "actor",
    "size": 512,
    "alignment": 64,
    "fields": [
      {"name":"system_mailbox_head","offset":272,"size":8,"end":280},
      {"name":"system_mailbox_tail","offset":280,"size":8,"end":288},
      {"name":"system_mailbox_count","offset":288,"size":4,"end":292},
      {"name":"system_recv_scratch","offset":328,"size":56,"end":384},
      {"name":"wait_kind","offset":392,"size":4,"end":396}
    ]
  },
  "scheduler": {
    "name": "scheduler",
    "size": 4960,
    "alignment": 64,
    "fields": [
      {"name":"system_event_base","offset":4872,"size":8,"end":4880},
      {"name":"system_event_live_bytes","offset":4912,"size":8,"end":4920},
      {"name":"system_event_reserved_credits","offset":4944,"size":8,"end":4952},
      {"name":"runtime_closing","offset":4952,"size":4,"end":4956}
    ]
  },
  "system_event": {
    "name": "system_event",
    "size": 64,
    "alignment": 8,
    "fields": [
      {"name":"kind","offset":8,"size":4,"end":12},
      {"name":"node_epoch","offset":40,"size":8,"end":48}
    ]
  },
  "raw_types": [
    {"name":"actor.node","slots":2,"runtime_owned":true,"user_constructible":false},
    {"name":"actor.system_recv_raw","slots":8,"runtime_owned":true,"user_constructible":false}
  ],
  "invariants": [
    {"name":"actor_system_mailbox_within_actor","pass":true},
    {"name":"scheduler_system_event_pool_fields_ordered","pass":true},
    {"name":"system_event_layout_separate_from_user_message","pass":true}
  ]
}`

type testArtifact struct {
	Path   string
	Schema string
}

func writeArtifactHashManifest(t *testing.T, root string, artifacts []testArtifact) {
	t.Helper()
	var rows []map[string]any
	for _, artifact := range artifacts {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(artifact.Path)))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		row := map[string]any{
			"path":   artifact.Path,
			"sha256": fmt.Sprintf("sha256:%x", sum),
			"size":   len(raw),
		}
		if artifact.Schema != "" {
			row["schema"] = artifact.Schema
		}
		rows = append(rows, row)
	}
	raw, err := json.MarshalIndent(map[string]any{
		"schema":    ArtifactHashSchema,
		"root":      ".",
		"artifacts": rows,
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "artifact-hashes.json"), string(raw)+"\n")
}

func writeFile(t *testing.T, path string, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}
