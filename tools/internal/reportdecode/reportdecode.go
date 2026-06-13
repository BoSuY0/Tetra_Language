package reportdecode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"tetra_language/internal/toon"
)

func DecodeStrict(raw []byte, out any) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return fmt.Errorf("empty report")
	}
	if trimmed[0] == '{' || trimmed[0] == '[' {
		return decodeStrictJSON(trimmed, out)
	}
	jsonData, err := toon.ConvertTOONToJSON(trimmed, toon.Options{Strict: true})
	if err != nil {
		return fmt.Errorf("invalid TOON: %w", err)
	}
	return decodeStrictJSON(jsonData, out)
}

func DecodeStrictFormat(raw []byte, format string, out any) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "auto":
		return DecodeStrict(raw, out)
	case "json":
		return decodeStrictJSON(bytes.TrimSpace(raw), out)
	case "toon":
		jsonData, err := toon.ConvertTOONToJSON(bytes.TrimSpace(raw), toon.Options{Strict: true})
		if err != nil {
			return fmt.Errorf("invalid TOON: %w", err)
		}
		return decodeStrictJSON(jsonData, out)
	default:
		return fmt.Errorf("unsupported report format %q", format)
	}
}

func decodeStrictJSON(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}
