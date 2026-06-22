package model

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestOptionsExposeLoweringFlags(t *testing.T) {
	opt := Options{
		StackAllocationLowering:    true,
		FunctionTempRegionLowering: true,
		OwnedAllocDropLowering:     true,
	}

	if !opt.StackAllocationLowering {
		t.Fatalf("StackAllocationLowering was not retained")
	}
	if !opt.FunctionTempRegionLowering {
		t.Fatalf("FunctionTempRegionLowering was not retained")
	}
	if !opt.OwnedAllocDropLowering {
		t.Fatalf("OwnedAllocDropLowering was not retained")
	}
}

func TestUILoweredJSONContract(t *testing.T) {
	if UIBundleSchema != "tetra.ui.v0.4.0" {
		t.Fatalf("unexpected UI bundle schema %q", UIBundleSchema)
	}

	bundle := UILoweredBundle{
		Schema: UIBundleSchema,
		States: []UILoweredState{{
			Name:   "CounterState",
			Module: "app",
			Fields: []UILoweredStateField{{
				Name:    "count",
				Type:    "i32",
				Mutable: true,
				Init:    "0",
			}},
		}},
		Views: []UILoweredView{{
			Name:      "CounterView",
			Module:    "app",
			StateType: "CounterState",
			Commands: []UILoweredCommand{{
				Name:           "inc",
				StatementCount: 1,
				Operations: []UILoweredCommandOperation{{
					Kind:   "state_add",
					Target: "state.count",
					Value:  "1",
				}},
			}},
		}},
	}

	encoded, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("marshal UI bundle: %v", err)
	}
	jsonText := string(encoded)
	for _, want := range []string{
		`"schema":"tetra.ui.v0.4.0"`,
		`"state_type":"CounterState"`,
		`"statement_count":1`,
		`"operations":[`,
	} {
		if !strings.Contains(jsonText, want) {
			t.Fatalf("encoded UI bundle missing %s in %s", want, jsonText)
		}
	}

	assertJSONTag(t, reflect.TypeOf(UILoweredCommand{}), "Operations", "operations,omitempty")
	assertJSONTag(t, reflect.TypeOf(UILoweredCommandOperation{}), "Value", "value,omitempty")
}

func TestUIToolkitJSONContract(t *testing.T) {
	if UIToolkitSchema != "tetra.ui.toolkit.v1" {
		t.Fatalf("unexpected UI toolkit schema %q", UIToolkitSchema)
	}

	bundle := UIToolkitBundle{
		Schema:              UIToolkitSchema,
		CompatibilitySchema: UIBundleSchema,
		Views: []UIToolkitView{{
			Name:        "CounterView",
			Module:      "app",
			StateType:   "CounterState",
			WidgetKinds: []string{"button"},
			LayoutKinds: []string{"row"},
			Widgets: []UIToolkitWidget{{
				ID:     "CounterView.inc",
				Kind:   "button",
				Layout: UIToolkitLayout{Kind: "row", Order: 1},
			}},
			Commands: []UIToolkitCommand{{
				Name:           "inc",
				StatementCount: 1,
				Operations: []UILoweredCommandOperation{{
					Kind:   "state_add",
					Target: "state.count",
					Value:  "1",
				}},
			}},
		}},
	}

	encoded, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("marshal UI toolkit bundle: %v", err)
	}
	jsonText := string(encoded)
	for _, want := range []string{
		`"schema":"tetra.ui.toolkit.v1"`,
		`"compatibility_schema":"tetra.ui.v0.4.0"`,
		`"widget_kinds":["button"]`,
		`"operations":[`,
	} {
		if !strings.Contains(jsonText, want) {
			t.Fatalf("encoded UI toolkit bundle missing %s in %s", want, jsonText)
		}
	}

	stateFields, ok := reflect.TypeOf(UIToolkitState{}).FieldByName("Fields")
	if !ok {
		t.Fatalf("UIToolkitState.Fields missing")
	}
	if stateFields.Type != reflect.TypeOf([]UILoweredStateField{}) {
		t.Fatalf("UIToolkitState.Fields type = %s", stateFields.Type)
	}
	assertJSONTag(t, reflect.TypeOf(UIToolkitBundle{}), "States", "states,omitempty")
	assertJSONTag(t, reflect.TypeOf(UIToolkitCommand{}), "Operations", "operations")
}

func assertJSONTag(t *testing.T, typ reflect.Type, fieldName, want string) {
	t.Helper()
	field, ok := typ.FieldByName(fieldName)
	if !ok {
		t.Fatalf("%s.%s missing", typ.Name(), fieldName)
	}
	if got := field.Tag.Get("json"); got != want {
		t.Fatalf("%s.%s json tag = %q, want %q", typ.Name(), fieldName, got, want)
	}
}
