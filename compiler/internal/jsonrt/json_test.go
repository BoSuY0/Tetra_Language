package jsonrt

import (
	"encoding/json"
	"testing"
	"unicode/utf8"
)

func TestAppendStringEscapesJSONControlCharacters(t *testing.T) {
	input := "quote \" slash \\ backspace \b tab \t newline \n formfeed \f carriage \r nul " + string(
		[]byte{0x00, 0x1f},
	)
	got := string(AppendString(nil, input))
	want := `"quote \" slash \\ backspace \b tab \t newline \n formfeed \f carriage \r nul \u0000\u001f"`
	if got != want {
		t.Fatalf("AppendString mismatch:\ngot  %q\nwant %q", got, want)
	}
	assertValidJSONString(t, got)
}

func TestAppendStringPreservesValidUTF8AndRepairsInvalidUTF8(t *testing.T) {
	valid := "Hello, Привіт, 世界"
	if got := string(AppendString(nil, valid)); got != `"`+valid+`"` {
		t.Fatalf("valid UTF-8 string = %q, want quoted original", got)
	}

	invalid := string([]byte{'o', 'k', 0xff, 'x'})
	got := string(AppendString(nil, invalid))
	if got != `"ok\ufffdx"` {
		t.Fatalf("invalid UTF-8 string = %q, want replacement escape", got)
	}
	assertValidJSONString(t, got)
}

func TestAppendMessageObjectWritesTechEmpowerPayload(t *testing.T) {
	got := string(AppendMessageObject(nil, "Hello, World!"))
	want := `{"message":"Hello, World!"}`
	if got != want {
		t.Fatalf("AppendMessageObject = %q, want %q", got, want)
	}
	var decoded struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if decoded.Message != "Hello, World!" {
		t.Fatalf("decoded message = %q", decoded.Message)
	}
}

func TestAppendWorldObjectAndArray(t *testing.T) {
	if got, want := string(
		AppendWorldObject(nil, 3217, 2149),
	), `{"id":3217,"randomNumber":2149}`; got != want {
		t.Fatalf("AppendWorldObject = %q, want %q", got, want)
	}
	worlds := []World{
		{ID: 1, RandomNumber: 42},
		{ID: 2, RandomNumber: 84},
	}
	got := string(AppendWorldArray(nil, worlds))
	want := `[{"id":1,"randomNumber":42},{"id":2,"randomNumber":84}]`
	if got != want {
		t.Fatalf("AppendWorldArray = %q, want %q", got, want)
	}
	var decoded []struct {
		ID           int `json:"id"`
		RandomNumber int `json:"randomNumber"`
	}
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("json.Unmarshal world array: %v", err)
	}
	if len(decoded) != 2 || decoded[0].ID != 1 || decoded[1].RandomNumber != 84 {
		t.Fatalf("decoded worlds = %#v", decoded)
	}
}

func TestParseAndAppendGenericJSONValueDeterministically(t *testing.T) {
	value, err := ParseValue([]byte(`{"b":2,"a":[true,null,"x\n"],"nested":{"z":false}}`))
	if err != nil {
		t.Fatalf("ParseValue: %v", err)
	}
	got, err := AppendValue(nil, value)
	if err != nil {
		t.Fatalf("AppendValue: %v", err)
	}
	want := `{"a":[true,null,"x\n"],"b":2,"nested":{"z":false}}`
	if string(got) != want {
		t.Fatalf("AppendValue deterministic output:\ngot  %s\nwant %s", got, want)
	}
	if !json.Valid(got) {
		t.Fatalf("AppendValue produced invalid JSON: %s", got)
	}
}

func TestGenericJSONValueRejectsMalformedAndUnsupportedValues(t *testing.T) {
	if _, err := ParseValue([]byte(`{"a":`)); err == nil {
		t.Fatalf("ParseValue accepted malformed JSON")
	}
	if _, err := AppendValue(nil, Value{Kind: NumberKind, Number: "01"}); err == nil {
		t.Fatalf("AppendValue accepted invalid JSON number")
	}
	if _, err := AppendValue(nil, Value{Kind: Kind(99)}); err == nil {
		t.Fatalf("AppendValue accepted unsupported kind")
	}
}

func FuzzAppendStringProducesValidJSON(f *testing.F) {
	for _, seed := range []string{
		"Hello, World!",
		"quote \" slash \\",
		"control \x00 \x1f",
		"Привіт",
		string([]byte{'x', 0xff, 'y'}),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		raw := AppendString(nil, input)
		if !json.Valid(raw) {
			t.Fatalf("AppendString produced invalid JSON for %q: %q", input, raw)
		}
		var decoded string
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", raw, err)
		}
		want := repairInvalidUTF8Bytewise(input)
		if decoded != want {
			t.Fatalf("decoded = %q, want %q from raw %q", decoded, want, raw)
		}
	})
}

func assertValidJSONString(t *testing.T, raw string) {
	t.Helper()
	if !json.Valid([]byte(raw)) {
		t.Fatalf("invalid JSON string: %q", raw)
	}
	var decoded string
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(%q): %v", raw, err)
	}
}

func repairInvalidUTF8Bytewise(input string) string {
	var out []rune
	for len(input) > 0 {
		r, size := utf8.DecodeRuneInString(input)
		if r == utf8.RuneError && size == 1 {
			out = append(out, utf8.RuneError)
			input = input[1:]
			continue
		}
		out = append(out, r)
		input = input[size:]
	}
	return string(out)
}
