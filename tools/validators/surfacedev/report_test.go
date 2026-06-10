package surfacedev

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateReportAcceptsEndToEndReloadTrace(t *testing.T) {
	report := validDevLoopReport()
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport(valid) = %v", err)
	}
}

func TestValidateReportRejectsHotReloadWithoutSourceChangeTrace(t *testing.T) {
	report := validDevLoopReport()
	report.Reloads[0].PreviousSHA256 = report.Reloads[0].CurrentSHA256
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	err = ValidateReport(raw)
	if err == nil {
		t.Fatal("ValidateReport accepted reload report without a source hash delta")
	}
	if !strings.Contains(err.Error(), "source change trace") {
		t.Fatalf("ValidateReport error = %v, want source change trace rejection", err)
	}
}

func TestValidateReportRequiresSurfaceTemplateCoverage(t *testing.T) {
	report := validDevLoopReport()
	report.TemplateSmoke.Templates = []string{"surface-minimal"}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	err = ValidateReport(raw)
	if err == nil {
		t.Fatal("ValidateReport accepted incomplete template coverage")
	}
	if !strings.Contains(err.Error(), "surface-dashboard") {
		t.Fatalf("ValidateReport error = %v, want missing template name", err)
	}
}

func validDevLoopReport() Report {
	return Report{
		Schema:       SchemaV1,
		Status:       "pass",
		Level:        LevelFastDevLoopV1,
		ProjectRoot:  "/work/demo",
		Template:     "surface-dashboard",
		Entry:        "src/main.t4",
		Source:       "src/main.t4",
		ReleaseScope: "surface-v1-linux-web",
		Mode:         "once",
		Reloads: []ReloadTrace{
			{
				Order:                1,
				Kind:                 "source-change-reload",
				Source:               "src/main.t4",
				PreviousSHA256:       "1111111111111111111111111111111111111111111111111111111111111111",
				CurrentSHA256:        "2222222222222222222222222222222222222222222222222222222222222222",
				PreviousMTimeUnixNS:  1000,
				CurrentMTimeUnixNS:   2000,
				ChangeDetected:       true,
				RebuildTriggered:     true,
				ReloadApplied:        true,
				InspectorUpdated:     true,
				ErrorOverlay:         "surface-inspector-diagnostics",
				StatePreserved:       true,
				SourceMapEntryCount:  2,
				ComponentSnapshotIDs: []string{"Root", "CommandList"},
			},
		},
		Operations: []Operation{
			{Name: "template check", Kind: "check", Ran: true, Pass: true, Detail: "compiler check passed"},
			{Name: "headless dev run", Kind: "run", Ran: true, Pass: true, Detail: "reload scheduler applied frame"},
			{Name: "inspector snapshot", Kind: "inspect", Ran: true, Pass: true, Detail: "source locations present"},
			{Name: "dev package", Kind: "package", Ran: true, Pass: true, Detail: "project package command resolved"},
		},
		TemplateSmoke: TemplateSmoke{
			Templates:      RequiredTemplates(),
			CreatedProject: true,
			Checkable:      true,
			Runnable:       true,
			Inspectable:    true,
			Packageable:    true,
		},
		StatePreservation: StatePreservation{
			Policy:           "schema-compatible-owned-state-only",
			Decision:         "preserve",
			Reason:           "source hash changed without state schema change",
			SchemaCompatible: true,
			PreservedKeys:    []string{"app.query", "panel.scroll_y"},
			ResetKeys:        []string{},
		},
		NegativeGuards: NegativeGuards{
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
