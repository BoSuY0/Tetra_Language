package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

const webUISmokeSchema = "tetra.web-ui-smoke.v1alpha1"

type webUISmokeReport struct {
	Schema             string `json:"schema"`
	GeneratedAt        string `json:"generated_at"`
	Target             string `json:"target"`
	UIScopeActive      bool   `json:"ui_scope_active"`
	Source             string `json:"source"`
	UsedFallbackSource bool   `json:"used_fallback_source"`
	Automation         string `json:"automation"`
	Status             string `json:"status"`
	Result             string `json:"result"`
	Blocker            string `json:"blocker"`
	DOMSnapshot        string `json:"dom_snapshot"`
	ChromiumStderr     string `json:"chromium_stderr"`
}

func main() {
	var reportPath string
	flag.StringVar(&reportPath, "report", "", "path to web UI smoke JSON report")
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
	var report webUISmokeReport
	if err := decodeStrictJSON(raw, &report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateWebUISmokeReport(report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateWebUISmokeReport(report webUISmokeReport) error {
	if report.Schema != webUISmokeSchema {
		return fmt.Errorf("web UI smoke schema = %q, want %q", report.Schema, webUISmokeSchema)
	}
	if report.GeneratedAt == "" {
		return fmt.Errorf("web UI smoke missing generated_at")
	}
	if _, err := time.Parse(time.RFC3339, report.GeneratedAt); err != nil {
		return fmt.Errorf("web UI smoke generated_at is not RFC3339: %w", err)
	}
	if report.Target != "wasm32-web" {
		return fmt.Errorf("web UI smoke target = %q, want wasm32-web", report.Target)
	}
	if report.Source == "" || !strings.HasSuffix(report.Source, ".tetra") {
		return fmt.Errorf("web UI smoke source must be a .tetra file")
	}
	if report.Automation == "" {
		return fmt.Errorf("web UI smoke missing automation")
	}
	switch report.Status {
	case "pass":
		if report.UsedFallbackSource {
			return fmt.Errorf("web UI smoke pass cannot use fallback source")
		}
		if !report.UIScopeActive {
			return fmt.Errorf("web UI smoke pass cannot use inactive UI scope")
		}
		if !strings.HasPrefix(report.Result, "ok:") {
			return fmt.Errorf("web UI smoke pass result must start with ok:")
		}
		if report.Blocker != "" {
			return fmt.Errorf("web UI smoke pass cannot include blocker")
		}
	case "blocked":
		if report.Blocker == "" {
			return fmt.Errorf("web UI smoke blocked report missing blocker")
		}
	case "fail":
		if report.Blocker == "" {
			return fmt.Errorf("web UI smoke failure missing blocker")
		}
	default:
		return fmt.Errorf("web UI smoke status = %q, want pass, blocked, or fail", report.Status)
	}
	return nil
}

func decodeStrictJSON(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}
