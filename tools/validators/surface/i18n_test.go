package surface

import (
	"strings"
	"testing"
)

func TestValidateI18nReportAcceptsLocalizedFormEvidence(t *testing.T) {
	raw := validI18nReportJSON()
	if err := ValidateI18nReport([]byte(raw)); err != nil {
		t.Fatalf("ValidateI18nReport failed: %v\n%s", err, raw)
	}
}

func TestValidateI18nReportRejectsMissingKeyWithoutDiagnostic(t *testing.T) {
	raw := strings.Replace(validI18nReportJSON(), `"missing_key_diagnostic": true`, `"missing_key_diagnostic": false`, 1)
	err := ValidateI18nReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing key diagnostic absence to fail")
	}
	if !strings.Contains(err.Error(), "missing_key") {
		t.Fatalf("error = %v, want missing_key diagnostic", err)
	}
}

func TestValidateI18nReportRejectsMissingFallbackLanguage(t *testing.T) {
	raw := strings.Replace(validI18nReportJSON(), `"fallback_locale": "en-US"`, `"fallback_locale": ""`, 1)
	err := ValidateI18nReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing fallback language to fail")
	}
	if !strings.Contains(err.Error(), "fallback") {
		t.Fatalf("error = %v, want fallback diagnostic", err)
	}
}

func TestValidateI18nReportRejectsFullBidiClaimWithoutShapingProof(t *testing.T) {
	raw := strings.Replace(validI18nReportJSON(), `"full_bidi_claim": false`, `"full_bidi_claim": true`, 1)
	err := ValidateI18nReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected full bidi claim without shaping proof to fail")
	}
	if !strings.Contains(err.Error(), "bidi") {
		t.Fatalf("error = %v, want bidi diagnostic", err)
	}
}

func validI18nReportJSON() string {
	return `{
  "schema": "tetra.surface.i18n.v1",
  "model": "surface-i18n-v1",
  "release_scope": "surface-v1-linux-web",
  "producer": "scripts/release/surface/surface-i18n-smoke.sh",
  "source": "examples/surface_reference_localized_form.tetra",
  "reference_app": "localized-form",
  "target": "linux-x64",
  "string_tables": [
    {"locale":"en-US","entry_count":5,"checksum":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","primary":true,"fallback":false,"pass":true},
    {"locale":"uk-UA","entry_count":4,"checksum":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","primary":false,"fallback":true,"pass":true}
  ],
  "locale_selection": {
    "requested_locale": "uk-UA",
    "selected_locale": "uk-UA",
    "fallback_locale": "en-US",
    "fallback_used": true,
    "unsupported_locale_rejected": true,
    "pass": true
  },
  "lookups": [
    {"key":"form.title","locale":"uk-UA","resolved_locale":"uk-UA","source":"primary","missing_key":false,"fallback_used":false,"diagnostic_code":0,"pass":true},
    {"key":"form.secondary","locale":"uk-UA","resolved_locale":"en-US","source":"fallback","missing_key":false,"fallback_used":true,"diagnostic_code":0,"pass":true},
    {"key":"form.unknown","locale":"uk-UA","resolved_locale":"en-US","source":"missing","missing_key":true,"fallback_used":true,"diagnostic_code":2001,"pass":true}
  ],
  "format_hooks": [
    {"kind":"date","locale":"uk-UA","input":"2026-06-12","output":"2026-06-12","deterministic":true,"icu_claim":false,"pass":true},
    {"kind":"number","locale":"uk-UA","input":"4200","output":"4200","deterministic":true,"icu_claim":false,"pass":true}
  ],
  "text_direction": {
    "default_direction": "ltr",
    "rtl_placeholder": true,
    "full_bidi_supported": false,
    "full_bidi_claim": false,
    "shaping_proof": false,
    "nonclaim": "rtl-placeholder-without-full-bidi-shaping-v1",
    "pass": true
  },
  "localized_form": {
    "shape": "localized-form",
    "source": "examples/surface_reference_localized_form.tetra",
    "imports": ["lib.core.surface","lib.core.block","lib.core.morph","lib.core.i18n"],
    "compiles": true,
    "runs": true,
    "exit_code": 0,
    "localized_strings": true,
    "fallback_evidence": true,
    "missing_key_diagnostic": true,
    "format_hook_evidence": true,
    "resolves_to_block": true,
    "pass": true
  },
  "negative_guards": {
    "no_full_icu_claim": true,
    "no_full_bidi_claim": true,
    "no_rtl_production_claim": true,
    "no_missing_key_silent_fallback": true,
    "no_docs_only_i18n_claim": true,
    "no_react_intl_runtime": true,
    "no_platform_locale_dependency": true
  },
  "pass": true
}
`
}
