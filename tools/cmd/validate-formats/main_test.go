package main

import (
	"strings"
	"testing"

	"tetra_language/internal/toon"
)

const validFormatsReport = `{
  "formats": [
    {"name":"T4 Source Format","extension":".t4","role":"source","description":"Tetra source file for capsules, apps, kernels, drivers, UI, games, and tests.","primary":true},
    {"name":"Legacy Tetra Source Format","extension":".tetra","role":"source","description":"Legacy Tetra source file kept for backward compatibility.","legacy":true},
    {"name":"Todex Fragment","extension":".tdx","role":"todex-fragment","description":"Todex encrypted semantic fragment."},
    {"name":"T4 Seed","extension":".t4s","role":"offline-seed","description":"Tetra Seed offline bundle."},
    {"name":"T4 Interface","extension":".t4i","role":"interface","description":"T4 interface file for fast type-checking without full source."},
    {"name":"T4 Proof","extension":".t4p","role":"proof","description":"T4 proof and verification file."},
    {"name":"T4 Replay","extension":".t4r","role":"replay","description":"T4 replay file for reproducible bugs and desync reports."},
    {"name":"T4 Quest","extension":".t4q","role":"quest","description":"T4 executable quest file."},
    {"name":"Tetra NeedMap","extension":".tneed","role":"needmap","description":"NeedMap file describing missing Todex fragments for offline builds."},
    {"name":"Tetra Semantic Lock","file_name":"Tetra.lock","role":"semantic-lock","description":"Tetra semantic lockfile for versions, policies, and reproducibility guarantees."}
  ]
}`

func TestValidateFormatsReportAcceptsExpectedShape(t *testing.T) {
	if err := validateFormatsReport([]byte(validFormatsReport)); err != nil {
		t.Fatalf("validate formats: %v", err)
	}
}

func TestValidateFormatsReportAcceptsTOON(t *testing.T) {
	raw, err := toon.ConvertJSONToTOON(
		[]byte(validFormatsReport),
		toon.Options{Deterministic: true, Strict: true},
	)
	if err != nil {
		t.Fatalf("json->toon: %v", err)
	}
	if err := validateFormatsReport(raw); err != nil {
		t.Fatalf("validate formats TOON: %v\n%s", err, raw)
	}
}

func TestValidateFormatsReportRejectsUnknownFields(t *testing.T) {
	raw := []byte(`{"formats":[],"extra":true}`)
	if err := validateFormatsReport(raw); err == nil ||
		!strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown top-level field failure, got %v", err)
	}

	raw = []byte(
		`{"formats":[{"name":"T4 Source Format","extension":".t4","role":"source","description":"Tetra source file.","primary":true,"extra":true}]}`,
	)
	if err := validateFormatsReport(raw); err == nil ||
		!strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown nested field failure, got %v", err)
	}
}

func TestValidateFormatsReportRejectsDuplicateKeys(t *testing.T) {
	raw := []byte(`{"formats":[
    {"name":"T4 Source Format","extension":".t4","role":"source","description":"Tetra source file.","primary":true},
    {"name":"Duplicate T4","extension":".t4","role":"source","description":"Duplicate source file."}
  ]}`)
	if err := validateFormatsReport(raw); err == nil ||
		!strings.Contains(err.Error(), "duplicate format .t4") {
		t.Fatalf("expected duplicate failure, got %v", err)
	}
}

func TestValidateFormatsReportRejectsMissingT4Primary(t *testing.T) {
	raw := []byte(strings.Replace(validFormatsReport, `"primary":true`, `"primary":false`, 1))
	if err := validateFormatsReport(raw); err == nil ||
		!strings.Contains(err.Error(), ".t4 must be primary source format") {
		t.Fatalf("expected missing .t4 primary failure, got %v", err)
	}
}

func TestValidateFormatsReportRejectsMissingTetraLegacy(t *testing.T) {
	raw := []byte(strings.Replace(validFormatsReport, `"legacy":true`, `"legacy":false`, 1))
	if err := validateFormatsReport(raw); err == nil ||
		!strings.Contains(err.Error(), ".tetra must be legacy source format") {
		t.Fatalf("expected missing .tetra legacy failure, got %v", err)
	}
}

func TestValidateFormatsReportRejectsExtensionAndFileNameTogether(t *testing.T) {
	raw := []byte(
		`{"formats":[{"name":"Broken","extension":".bad","file_name":"Broken.tetra","role":"source","description":"bad"}]}`,
	)
	if err := validateFormatsReport(raw); err == nil ||
		!strings.Contains(err.Error(), "must not set both extension and file_name") {
		t.Fatalf("expected extension/file_name exclusivity failure, got %v", err)
	}
}
