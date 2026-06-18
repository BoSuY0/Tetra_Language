package uitoolkit

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidateReportAcceptsToolkitCoreProductionEvidence(t *testing.T) {
	raw := validToolkitCoreReport(t)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsRuntimeLessEvidence(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.ui.toolkit.v1",
  "status": "pass",
  "target": "toolkit-core",
  "host": "linux-x64",
  "runtime": "toolkit-core",
  "ui_schema": "tetra.ui.toolkit.v1",
  "source": "docs-only-runtime-less-placeholder.md",
  "artifacts": [],
  "processes": [],
  "contracts": [],
  "widgets": [],
  "layouts": [],
  "events": [],
  "state_transitions": [],
  "cases": [],
  "audit": []
}`)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected runtime-less toolkit evidence to fail")
	}
	for _, want := range []string{"runtime-less", "process", "artifact", "widget", "event", "case"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateReportRejectsMissingToolkitWidgetCoverage(t *testing.T) {
	raw := mutateToolkitCoreReport(t, validToolkitCoreReport(t), func(report *Report) {
		report.Widgets = removeWidget(report.Widgets, "MenuItemOpen")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing menu-item widget coverage to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "menu-item") {
		t.Fatalf("error missing menu-item coverage:\n%v", err)
	}
}

func TestValidateReportRejectsMissingEventAndStateEvidence(t *testing.T) {
	raw := mutateToolkitCoreReport(t, validToolkitCoreReport(t), func(report *Report) {
		report.Events = removeEvent(report.Events, 8)
		report.StateTransitions = removeTransition(
			report.StateTransitions,
			"two-way input binding",
		)
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing input/state evidence to fail")
	}
	for _, want := range []string{"input", "two-way input binding"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func TestValidateReportRejectsMissingLayoutFocusAccessibility(t *testing.T) {
	raw := mutateToolkitCoreReport(t, validToolkitCoreReport(t), func(report *Report) {
		report.Layouts = removeLayout(report.Layouts, "grid")
		setWidgetFocusOrder(report.Widgets, "DataTable", 0)
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing layout/focus evidence to fail")
	}
	for _, want := range []string{"grid", "focus"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func validToolkitCoreReport(t *testing.T) []byte {
	t.Helper()
	reportPath := filepath.Join(t.TempDir(), "ui-toolkit-core.json")
	cmd := exec.Command(
		"go",
		"run",
		"-buildvcs=false",
		"./tools/cmd/ui-toolkit-core-smoke",
		"--report",
		reportPath,
	)
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generate toolkit report: %v\n%s", err, output)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read generated report: %v", err)
	}
	return raw
}

func mutateToolkitCoreReport(t *testing.T, raw []byte, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode generated report: %v", err)
	}
	mutate(&report)
	out, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("encode mutated report: %v", err)
	}
	return out
}

func removeWidget(widgets []WidgetReport, id string) []WidgetReport {
	out := widgets[:0]
	for _, widget := range widgets {
		if widget.ID != id {
			out = append(out, widget)
		}
	}
	return out
}

func removeEvent(events []EventReport, order int) []EventReport {
	out := events[:0]
	for _, event := range events {
		if event.Order != order {
			out = append(out, event)
		}
	}
	return out
}

func removeTransition(
	transitions []StateTransitionReport,
	name string,
) []StateTransitionReport {
	out := transitions[:0]
	for _, transition := range transitions {
		if transition.Name != name {
			out = append(out, transition)
		}
	}
	return out
}

func removeLayout(layouts []LayoutReport, kind string) []LayoutReport {
	out := layouts[:0]
	for _, layout := range layouts {
		if layout.Kind != kind {
			out = append(out, layout)
		}
	}
	return out
}

func setWidgetFocusOrder(widgets []WidgetReport, id string, focusOrder int) {
	for index := range widgets {
		if widgets[index].ID == id {
			widgets[index].Accessibility.FocusOrder = focusOrder
			return
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
