package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

type formatsReport struct {
	Formats []formatEntry `json:"formats"`
}

type formatEntry struct {
	Name        string `json:"name"`
	Extension   string `json:"extension,omitempty"`
	FileName    string `json:"file_name,omitempty"`
	Role        string `json:"role"`
	Description string `json:"description"`
	Primary     bool   `json:"primary,omitempty"`
	Legacy      bool   `json:"legacy,omitempty"`
}

func main() {
	var path string
	flag.StringVar(&path, "report", "", "path to tetra formats --format=json output")
	flag.Parse()
	if path == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateFormatsReport(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateFormatsReport(raw []byte) error {
	var report formatsReport
	if err := decodeStrictJSON(raw, &report); err != nil {
		return fmt.Errorf("invalid formats JSON: %w", err)
	}
	return validateFormats(report.Formats)
}

func decodeStrictJSON(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}

func validateFormats(formats []formatEntry) error {
	if len(formats) == 0 {
		return fmt.Errorf("formats must not be empty")
	}
	requiredRoles := map[string]string{
		".t4":        "source",
		".tetra":     "source",
		".tdx":       "todex-fragment",
		".t4s":       "offline-seed",
		".t4i":       "interface",
		".t4p":       "proof",
		".t4r":       "replay",
		".t4q":       "quest",
		".tneed":     "needmap",
		"Tetra.lock": "semantic-lock",
	}
	officialOrder := []string{".t4", ".tetra", ".tdx", ".t4s", ".t4i", ".t4p", ".t4r", ".t4q", ".tneed", "Tetra.lock"}
	seen := map[string]bool{}
	var order []string
	for _, format := range formats {
		key, err := validateFormatEntry(format)
		if err != nil {
			return err
		}
		if seen[key] {
			return fmt.Errorf("duplicate format %s", key)
		}
		seen[key] = true
		order = append(order, key)
		if wantRole, ok := requiredRoles[key]; ok && format.Role != wantRole {
			return fmt.Errorf("format %s role = %s, want %s", key, format.Role, wantRole)
		}
		switch key {
		case ".t4":
			if !format.Primary || format.Legacy {
				return fmt.Errorf(".t4 must be primary source format")
			}
		case ".tetra":
			if !format.Legacy || format.Primary {
				return fmt.Errorf(".tetra must be legacy source format")
			}
		default:
			if format.Primary || format.Legacy {
				return fmt.Errorf("format %s must not set primary or legacy", key)
			}
		}
	}
	for _, key := range officialOrder {
		if !seen[key] {
			return fmt.Errorf("formats missing %s", key)
		}
	}
	if len(order) >= len(officialOrder) && !sameStringSequence(order[:len(officialOrder)], officialOrder) {
		return fmt.Errorf("formats must start with official T4 order: got %s want %s", strings.Join(order[:len(officialOrder)], ", "), strings.Join(officialOrder, ", "))
	}
	return nil
}

func validateFormatEntry(format formatEntry) (string, error) {
	if format.Name == "" {
		return "", fmt.Errorf("format missing name")
	}
	if format.Role == "" {
		return "", fmt.Errorf("format %s missing role", format.Name)
	}
	if format.Description == "" {
		return "", fmt.Errorf("format %s missing description", format.Name)
	}
	if format.Extension != "" && format.FileName != "" {
		return "", fmt.Errorf("format %s must not set both extension and file_name", format.Name)
	}
	if format.Extension != "" {
		if !strings.HasPrefix(format.Extension, ".") {
			return "", fmt.Errorf("format %s extension must start with '.'", format.Name)
		}
		return format.Extension, nil
	}
	if format.FileName != "" {
		if strings.ContainsAny(format.FileName, `/\`) {
			return "", fmt.Errorf("format %s file_name must be a base name", format.Name)
		}
		return format.FileName, nil
	}
	return "", fmt.Errorf("format %s missing extension or file_name", format.Name)
}

func sameStringSequence(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
