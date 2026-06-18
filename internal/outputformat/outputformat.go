package outputformat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/internal/toon"
)

const (
	Auto = "auto"
	Text = "text"
	JSON = "json"
	TOON = "toon"
	Both = "both"

	ExtensionJSON = ".json"
	ExtensionTOON = ".toon"

	MediaTypeJSON = "application/json"
	MediaTypeTOON = "text/toon; charset=utf-8"
)

func Structured(format string) bool {
	format = Normalize(format)
	return format == JSON || format == TOON
}

func StructuredOrBoth(format string) bool {
	format = Normalize(format)
	return Structured(format) || format == Both
}

func Normalize(format string) string {
	return strings.ToLower(strings.TrimSpace(format))
}

type OutputFile struct {
	Path   string
	Format string
}

func WriteStructured(w io.Writer, format string, value any) error {
	raw, err := MarshalStructured(format, value)
	if err != nil {
		return err
	}
	_, err = w.Write(raw)
	return err
}

func DecodeStructured(raw []byte, format string, out any) error {
	jsonRaw, err := decodeInputJSON(raw, format)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonRaw, out)
}

func DecodeStructuredStrict(raw []byte, format string, out any) error {
	jsonRaw, err := decodeInputJSON(raw, format)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(jsonRaw))
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

func decodeInputJSON(raw []byte, format string) ([]byte, error) {
	format = Normalize(format)
	if format == "" {
		format = Auto
	}
	switch format {
	case Auto:
		trimmed := bytes.TrimSpace(raw)
		if len(trimmed) == 0 {
			return nil, fmt.Errorf("empty structured input")
		}
		if trimmed[0] == '{' || trimmed[0] == '[' {
			return trimmed, nil
		}
		return toon.ConvertTOONToJSON(trimmed, toon.Options{Strict: true})
	case JSON:
		return bytes.TrimSpace(raw), nil
	case TOON:
		return toon.ConvertTOONToJSON(bytes.TrimSpace(raw), toon.Options{Strict: true})
	default:
		return nil, unsupportedFormat(format)
	}
}

func MarshalStructured(format string, value any) ([]byte, error) {
	format = Normalize(format)
	switch format {
	case JSON:
		var b strings.Builder
		enc := json.NewEncoder(&b)
		enc.SetIndent("", "  ")
		if err := enc.Encode(value); err != nil {
			return nil, err
		}
		return []byte(b.String()), nil
	case TOON:
		raw, err := toon.MarshalIndent(value, toon.Options{Deterministic: true, Strict: true})
		if err != nil {
			return nil, err
		}
		return append(raw, '\n'), nil
	default:
		return nil, unsupportedFormat(format)
	}
}

func InferFormatFromPath(path string, fallback string) (string, error) {
	fallback = Normalize(fallback)
	if !StructuredOrBoth(fallback) {
		return "", unsupportedFormat(fallback)
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ExtensionJSON:
		return JSON, nil
	case ExtensionTOON:
		return TOON, nil
	default:
		return fallback, nil
	}
}

func OutputPathsForFormat(path string, format string) ([]OutputFile, error) {
	format = Normalize(format)
	switch format {
	case JSON, TOON:
		return []OutputFile{{Path: path, Format: format}}, nil
	case Both:
		jsonPath := replaceStructuredExtension(path, ExtensionJSON)
		toonPath := replaceStructuredExtension(path, ExtensionTOON)
		return []OutputFile{
			{Path: jsonPath, Format: JSON},
			{Path: toonPath, Format: TOON},
		}, nil
	default:
		return nil, unsupportedFormat(format)
	}
}

func WriteStructuredFile(path string, format string, value any) error {
	_, err := WriteStructuredFiles(path, format, value)
	return err
}

func WriteStructuredFiles(path string, format string, value any) ([]string, error) {
	files, err := OutputPathsForFormat(path, format)
	if err != nil {
		return nil, err
	}
	written := make([]string, 0, len(files))
	for _, file := range files {
		raw, err := MarshalStructured(file.Format, value)
		if err != nil {
			return written, err
		}
		if err := os.MkdirAll(filepath.Dir(file.Path), 0o755); err != nil {
			return written, err
		}
		if err := os.WriteFile(file.Path, raw, 0o644); err != nil {
			return written, err
		}
		written = append(written, file.Path)
	}
	return written, nil
}

func replaceStructuredExtension(path string, ext string) string {
	current := strings.ToLower(filepath.Ext(path))
	if current == ExtensionJSON || current == ExtensionTOON {
		return strings.TrimSuffix(path, filepath.Ext(path)) + ext
	}
	return path + ext
}

func unsupportedFormat(format string) error {
	return fmt.Errorf("unsupported structured output format %q", format)
}
