package toon

import (
	"math"
	"strings"
	"testing"
)

func TestConvertJSONToTOONRoundTripSupportedShapes(t *testing.T) {
	input := []byte(`{
		"empty_array": [],
		"empty_object": {},
		"mixed": [1, {"kind": "object"}, [true, false], {}],
		"nested": {"message": "hello: toon", "unicode": "Привіт"},
		"tags": ["compiler", "tools,ci", "true", ""],
		"users": [
			{"id": 1, "name": "Ada", "active": true},
			{"id": 2, "name": "Linus", "active": false}
		]
	}`)

	toonData, err := ConvertJSONToTOON(input, Options{Deterministic: true, Strict: true})
	if err != nil {
		t.Fatalf("ConvertJSONToTOON failed: %v", err)
	}

	toonText := string(toonData)
	required := []string{
		"empty_array: []",
		"empty_object:",
		"nested:",
		"  message: \"hello: toon\"",
		"  unicode: Привіт",
		"tags[4]: compiler,\"tools,ci\",\"true\",\"\"",
		"users[2]{active,id,name}:",
		"  true,1,Ada",
		"  false,2,Linus",
	}
	for _, want := range required {
		if !strings.Contains(toonText, want) {
			t.Fatalf("TOON output missing %q:\n%s", want, toonText)
		}
	}

	jsonData, err := ConvertTOONToJSON(toonData, Options{Strict: true})
	if err != nil {
		t.Fatalf("ConvertTOONToJSON failed: %v\nTOON:\n%s", err, toonText)
	}
	assertJSONEqual(t, jsonData, input)
}

func TestMarshalRejectsNonFiniteNumbers(t *testing.T) {
	_, err := Marshal(map[string]float64{"bad": math.Inf(1)})
	assertTOONErrorCode(t, err, CodeNonFiniteNumber)
}
