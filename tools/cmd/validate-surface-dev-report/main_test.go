package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surfacedev"
)

func TestValidateSurfaceDevReportCommand(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "surface-dev-report.json")
	raw, err := json.Marshal(validCommandDevLoopReport())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runValidateSurfaceDevReport([]string{"--report", reportPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "surface dev report OK") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func validCommandDevLoopReport() surfacedev.Report {
	return surfacedev.Report{
		Schema:       surfacedev.SchemaV1,
		Status:       "pass",
		Level:        surfacedev.LevelFastDevLoopV1,
		ProjectRoot:  "/work/demo",
		Template:     "surface-minimal",
		Entry:        "src/main.t4",
		Source:       "src/main.t4",
		ReleaseScope: "surface-v1-linux-web",
		Mode:         "once",
		Reloads: []surfacedev.ReloadTrace{
			{
				Order:                1,
				Kind:                 "source-change-reload",
				Source:               "src/main.t4",
				PreviousSHA256:       "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				CurrentSHA256:        "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				PreviousMTimeUnixNS:  10,
				CurrentMTimeUnixNS:   20,
				ChangeDetected:       true,
				RebuildTriggered:     true,
				ReloadApplied:        true,
				InspectorUpdated:     true,
				ErrorOverlay:         "surface-inspector-diagnostics",
				StatePreserved:       true,
				SourceMapEntryCount:  1,
				ComponentSnapshotIDs: []string{"Root"},
			},
		},
		Operations: []surfacedev.Operation{
			{Name: "template check", Kind: "check", Ran: true, Pass: true, Detail: "ok"},
			{Name: "headless dev run", Kind: "run", Ran: true, Pass: true, Detail: "ok"},
			{Name: "inspector snapshot", Kind: "inspect", Ran: true, Pass: true, Detail: "ok"},
			{Name: "dev package", Kind: "package", Ran: true, Pass: true, Detail: "ok"},
		},
		TemplateSmoke: surfacedev.TemplateSmoke{
			Templates:      surfacedev.RequiredTemplates(),
			CreatedProject: true,
			Checkable:      true,
			Runnable:       true,
			Inspectable:    true,
			Packageable:    true,
		},
		StatePreservation: surfacedev.StatePreservation{
			Policy:           "schema-compatible-owned-state-only",
			Decision:         "preserve",
			Reason:           "source hash changed without state schema change",
			SchemaCompatible: true,
			PreservedKeys:    []string{"app.query"},
		},
		NegativeGuards: surfacedev.NegativeGuards{
			SourceChangeTraceRequired: true,
			NoElectronDevServer:       true,
			NoReactFastRefresh:        true,
			NoCSSRuntimeInjection:     true,
			NoDOMHotReload:            true,
		},
		NonClaims: []string{
			"browser devtools parity",
			"React Fast Refresh compatibility",
			"CSS HMR runtime",
			"state preservation across incompatible schemas",
		},
	}
}
