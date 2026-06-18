package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateNativeUISmokeAcceptsDispatchAndWidgetTrace(t *testing.T) {
	report := validNativeUISmokeReport(t)
	if err := validateNativeUISmoke([]byte(report)); err != nil {
		t.Fatalf("validateNativeUISmoke failed: %v\n%s", err, report)
	}
}

func TestValidateNativeUISmokeRejectsMissingDispatchTrace(t *testing.T) {
	report := validNativeUISmokeReportFrom(t, func(report *nativeUISmokeReport) {
		report.Views[0].Events = nil
	})
	if err := validateNativeUISmoke([]byte(report)); err == nil {
		t.Fatalf("expected missing dispatch trace failure")
	} else if !strings.Contains(err.Error(), "view CounterView missing event dispatch trace") {
		t.Fatalf("error = %v, want missing event dispatch trace", err)
	}
}

func TestValidateNativeUISmokeRejectsMissingActionWidget(t *testing.T) {
	report := validNativeUISmokeReportFrom(t, func(report *nativeUISmokeReport) {
		report.Views[0].Widgets = report.Views[0].Widgets[:1]
	})
	if err := validateNativeUISmoke([]byte(report)); err == nil {
		t.Fatalf("expected missing action widget failure")
	} else if !strings.Contains(err.Error(), "view CounterView missing action widget") {
		t.Fatalf("error = %v, want missing action widget", err)
	}
}

func validNativeUISmokeReport(t *testing.T) string {
	t.Helper()
	return validNativeUISmokeReportFrom(t, func(*nativeUISmokeReport) {})
}

func validNativeUISmokeReportFrom(t *testing.T, mutate func(*nativeUISmokeReport)) string {
	t.Helper()
	report := nativeUISmokeReport{
		Schema:   nativeUISmokeSchemaV1,
		UISchema: uiBundleSchemaV1,
		Runtime:  nativeUIRuntimeDispatch,
		States: []nativeUIStateTrace{
			{
				Name: "CounterState",
				Fields: []nativeUIStateFieldTrace{
					{Name: "count", Type: "i32", Mutable: true, Value: "0"},
				},
			},
		},
		Views: []nativeUIViewTrace{
			{
				Name:      "CounterView",
				StateType: "CounterState",
				Bindings:  []nativeUIBindingTrace{{Name: "countValue", Type: "i32", Value: "0"}},
				Widgets: []nativeUIWidgetTrace{
					{
						ID:      "CounterView.countValue",
						Kind:    "value",
						Binding: "countValue",
						Type:    "i32",
						Value:   "0",
					},
					{ID: "CounterView.click", Kind: "action", Event: "click", Command: "increment"},
				},
				Events: []nativeUIEventTrace{
					{
						Name:    "click",
						Command: "increment",
						Operations: []nativeUIOperationTrace{
							{
								Kind:       "state_add",
								Target:     "state.count",
								Value:      "1",
								StateField: "count",
								StateValue: "1",
							},
						},
						Bindings: []nativeUIBindingTrace{
							{Name: "countValue", Type: "i32", Value: "1"},
						},
					},
				},
			},
		},
	}
	mutate(&report)
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
