package toon

import (
	"encoding/json"
	"testing"
)

func TestConvertJSONToTOONRoundTripSimpleObject(t *testing.T) {
	input := []byte(`{"name":"Ada","active":true,"score":42}`)

	toonData, err := ConvertJSONToTOON(input, Options{Deterministic: true, Strict: true})
	if err != nil {
		t.Fatalf("ConvertJSONToTOON failed: %v", err)
	}
	if got, want := string(toonData), "active: true\nname: Ada\nscore: 42"; got != want {
		t.Fatalf("TOON output mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}

	jsonData, err := ConvertTOONToJSON(toonData, Options{Strict: true})
	if err != nil {
		t.Fatalf("ConvertTOONToJSON failed: %v", err)
	}
	assertJSONEqual(t, jsonData, input)
}

func assertJSONEqual(t *testing.T, got []byte, want []byte) {
	t.Helper()
	var gotValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("got invalid JSON: %v\n%s", err, got)
	}
	var wantValue any
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatalf("want invalid JSON: %v\n%s", err, want)
	}
	if !jsonValuesEqual(gotValue, wantValue) {
		t.Fatalf("JSON mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func jsonValuesEqual(a any, b any) bool {
	ab, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bb, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(ab) == string(bb)
}
