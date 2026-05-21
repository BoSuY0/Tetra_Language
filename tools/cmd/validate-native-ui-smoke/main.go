package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	nativeUISmokeSchemaV1   = "tetra.ui.native-shell.v1"
	uiBundleSchemaV1        = "tetra.ui.v0.4.0"
	nativeUIRuntimeDispatch = "native shell command dispatch"
)

type nativeUISmokeReport struct {
	Schema   string               `json:"schema"`
	UISchema string               `json:"ui_schema"`
	Runtime  string               `json:"runtime"`
	States   []nativeUIStateTrace `json:"states,omitempty"`
	Views    []nativeUIViewTrace  `json:"views,omitempty"`
}

type nativeUIStateTrace struct {
	Name   string                    `json:"name"`
	Module string                    `json:"module,omitempty"`
	Fields []nativeUIStateFieldTrace `json:"fields,omitempty"`
}

type nativeUIStateFieldTrace struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Mutable bool   `json:"mutable,omitempty"`
	Const   bool   `json:"const,omitempty"`
	Value   string `json:"value"`
}

type nativeUIViewTrace struct {
	Name          string                  `json:"name"`
	Module        string                  `json:"module,omitempty"`
	StateType     string                  `json:"state_type"`
	Bindings      []nativeUIBindingTrace  `json:"bindings,omitempty"`
	Widgets       []nativeUIWidgetTrace   `json:"widgets,omitempty"`
	Events        []nativeUIEventTrace    `json:"events,omitempty"`
	Styles        []nativeUIPropertyTrace `json:"styles,omitempty"`
	Accessibility []nativeUIPropertyTrace `json:"accessibility,omitempty"`
}

type nativeUIWidgetTrace struct {
	ID            string                  `json:"id"`
	Kind          string                  `json:"kind"`
	Binding       string                  `json:"binding,omitempty"`
	Event         string                  `json:"event,omitempty"`
	Command       string                  `json:"command,omitempty"`
	Type          string                  `json:"type,omitempty"`
	Value         string                  `json:"value,omitempty"`
	Styles        []nativeUIPropertyTrace `json:"styles,omitempty"`
	Accessibility []nativeUIPropertyTrace `json:"accessibility,omitempty"`
}

type nativeUIEventTrace struct {
	Name       string                   `json:"name"`
	Command    string                   `json:"command"`
	Operations []nativeUIOperationTrace `json:"operations,omitempty"`
	Bindings   []nativeUIBindingTrace   `json:"bindings,omitempty"`
}

type nativeUIOperationTrace struct {
	Kind       string `json:"kind"`
	Target     string `json:"target"`
	Value      string `json:"value,omitempty"`
	StateField string `json:"state_field,omitempty"`
	StateValue string `json:"state_value,omitempty"`
}

type nativeUIBindingTrace struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type nativeUIPropertyTrace struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

func main() {
	var reportPath string
	flag.StringVar(&reportPath, "report", "", "path to tetra.ui.native-shell.v1 JSON trace")
	flag.Parse()
	if reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateNativeUISmoke(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateNativeUISmoke(raw []byte) error {
	var report nativeUISmokeReport
	if err := decodeStrictNativeUIJSON(raw, &report); err != nil {
		return err
	}
	if report.Schema != nativeUISmokeSchemaV1 {
		return fmt.Errorf("native UI schema = %q, want %q", report.Schema, nativeUISmokeSchemaV1)
	}
	if report.UISchema != uiBundleSchemaV1 {
		return fmt.Errorf("native UI ui_schema = %q, want %q", report.UISchema, uiBundleSchemaV1)
	}
	if report.Runtime != nativeUIRuntimeDispatch {
		return fmt.Errorf("native UI runtime = %q, want %q", report.Runtime, nativeUIRuntimeDispatch)
	}
	if len(report.States) == 0 {
		return fmt.Errorf("native UI report missing states")
	}
	stateNames := map[string]bool{}
	for _, state := range report.States {
		if strings.TrimSpace(state.Name) == "" {
			return fmt.Errorf("state name is required")
		}
		stateNames[state.Name] = true
		if len(state.Fields) == 0 {
			return fmt.Errorf("state %s missing fields", state.Name)
		}
		for _, field := range state.Fields {
			if strings.TrimSpace(field.Name) == "" || strings.TrimSpace(field.Type) == "" {
				return fmt.Errorf("state %s has field missing name or type", state.Name)
			}
		}
	}
	if len(report.Views) == 0 {
		return fmt.Errorf("native UI report missing views")
	}
	for _, view := range report.Views {
		if err := validateNativeUIView(view, stateNames); err != nil {
			return err
		}
	}
	return nil
}

func validateNativeUIView(view nativeUIViewTrace, stateNames map[string]bool) error {
	if strings.TrimSpace(view.Name) == "" {
		return fmt.Errorf("view name is required")
	}
	if !stateNames[view.StateType] {
		return fmt.Errorf("view %s references unknown state_type %q", view.Name, view.StateType)
	}
	if len(view.Bindings) == 0 {
		return fmt.Errorf("view %s missing initial bindings", view.Name)
	}
	if len(view.Events) == 0 {
		return fmt.Errorf("view %s missing event dispatch trace", view.Name)
	}
	if len(view.Widgets) == 0 {
		return fmt.Errorf("view %s missing widgets", view.Name)
	}
	bindingWidgets := map[string]bool{}
	actionWidgets := map[string]bool{}
	for _, widget := range view.Widgets {
		if strings.TrimSpace(widget.ID) == "" || strings.TrimSpace(widget.Kind) == "" {
			return fmt.Errorf("view %s has widget missing id or kind", view.Name)
		}
		switch widget.Kind {
		case "text", "value":
			if widget.Binding == "" || widget.Type == "" {
				return fmt.Errorf("view %s has binding widget missing binding or type", view.Name)
			}
			bindingWidgets[widget.Binding] = true
		case "action":
			if widget.Event == "" || widget.Command == "" {
				return fmt.Errorf("view %s has action widget missing event or command", view.Name)
			}
			actionWidgets[widget.Event+"->"+widget.Command] = true
		default:
			return fmt.Errorf("view %s has unsupported widget kind %q", view.Name, widget.Kind)
		}
	}
	for _, binding := range view.Bindings {
		if binding.Name == "" || binding.Type == "" {
			return fmt.Errorf("view %s has binding missing name or type", view.Name)
		}
		if !bindingWidgets[binding.Name] {
			return fmt.Errorf("view %s missing widget for binding %s", view.Name, binding.Name)
		}
	}
	for _, event := range view.Events {
		if event.Name == "" || event.Command == "" {
			return fmt.Errorf("view %s has event missing name or command", view.Name)
		}
		if !actionWidgets[event.Name+"->"+event.Command] {
			return fmt.Errorf("view %s missing action widget for event %s", view.Name, event.Name)
		}
		if len(event.Operations) == 0 {
			return fmt.Errorf("view %s event %s missing operations", view.Name, event.Name)
		}
		if len(event.Bindings) == 0 {
			return fmt.Errorf("view %s event %s missing post-dispatch bindings", view.Name, event.Name)
		}
		for _, op := range event.Operations {
			if op.Kind == "" || op.Target == "" || op.StateField == "" || op.StateValue == "" {
				return fmt.Errorf("view %s event %s has incomplete operation trace", view.Name, event.Name)
			}
		}
	}
	return nil
}

func decodeStrictNativeUIJSON(raw []byte, out any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err != nil {
			return err
		}
		return fmt.Errorf("multiple JSON values")
	}
	return nil
}
