package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewSurfaceAppScaffoldCreatesDevLoopProject(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "SurfaceDesk")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"new", "surface-app", "--template", "surface-dashboard", appDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("new surface-app exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	for _, rel := range []string{"Capsule.t4", "src/main.t4", "tests/surface_smoke.t4", "surface.template.json", "README.md"} {
		if _, err := os.Stat(filepath.Join(appDir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected scaffold file %s: %v", rel, err)
		}
	}
	capsuleRaw, err := os.ReadFile(filepath.Join(appDir, "Capsule.t4"))
	if err != nil {
		t.Fatal(err)
	}
	capsuleText := string(capsuleRaw)
	for _, want := range []string{`capsule SurfaceDesk:`, `id "tetra://surface-apps/surfacedesk"`, `entry "src/main.t4"`, `source "src"`, `source "tests"`, `target "` + mustHostTarget(t) + `"`} {
		if !strings.Contains(capsuleText, want) {
			t.Fatalf("Capsule.t4 missing %q:\n%s", want, capsuleText)
		}
	}
	var metadata struct {
		Schema   string   `json:"schema"`
		Template string   `json:"template"`
		Commands []string `json:"commands"`
	}
	rawMetadata, err := os.ReadFile(filepath.Join(appDir, "surface.template.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(rawMetadata, &metadata); err != nil {
		t.Fatalf("unmarshal surface.template.json: %v\n%s", err, rawMetadata)
	}
	if metadata.Schema != "tetra.surface.template.v1" || metadata.Template != "surface-dashboard" {
		t.Fatalf("metadata = %#v", metadata)
	}
	for _, want := range []string{"tetra check .", "tetra surface dev --project .", "tetra surface inspect", "tetra surface package ."} {
		found := false
		for _, command := range metadata.Commands {
			found = found || strings.Contains(command, want)
		}
		if !found {
			t.Fatalf("metadata commands missing %q: %#v", want, metadata.Commands)
		}
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"check", appDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("scaffold check exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestSurfaceDevOnceWritesReloadReportAfterSourceChange(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "SurfaceDesk")
	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"new", "surface-app", appDir}, &stdout, &stderr); code != 0 {
		t.Fatalf("new surface-app exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	statePath := filepath.Join(appDir, ".tetra", "surface-dev-state.json")
	warmupReport := filepath.Join(appDir, ".tetra", "surface-dev-warmup.json")
	stdout.Reset()
	stderr.Reset()
	if code := runCLI([]string{"surface", "dev", "--project", appDir, "--once", "--state", statePath, "--report", warmupReport}, &stdout, &stderr); code != 0 {
		t.Fatalf("surface dev warmup exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	entryPath := filepath.Join(appDir, "src", "main.t4")
	f, err := os.OpenFile(entryPath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("\n// reload-change: dashboard title\n"); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	reportPath := filepath.Join(appDir, ".tetra", "surface-dev-report.json")
	stdout.Reset()
	stderr.Reset()
	code := runCLI([]string{"surface", "dev", "--project", appDir, "--once", "--state", statePath, "--report", reportPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("surface dev reload exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "surface dev reload report") {
		t.Fatalf("stdout = %q, want reload report confirmation", stdout.String())
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read dev report: %v", err)
	}
	var report struct {
		Schema  string `json:"schema"`
		Status  string `json:"status"`
		Level   string `json:"level"`
		Reloads []struct {
			Kind            string `json:"kind"`
			PreviousSHA256  string `json:"previous_sha256"`
			CurrentSHA256   string `json:"current_sha256"`
			ChangeDetected  bool   `json:"change_detected"`
			ReloadApplied   bool   `json:"reload_applied"`
			InspectorUpdate bool   `json:"inspector_updated"`
		} `json:"reloads"`
		StatePreservation struct {
			Policy           string `json:"policy"`
			Decision         string `json:"decision"`
			SchemaCompatible bool   `json:"schema_compatible"`
		} `json:"state_preservation"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("unmarshal dev report: %v\n%s", err, raw)
	}
	if report.Schema != "tetra.surface.dev-loop.v1" || report.Status != "pass" || report.Level != "surface-fast-dev-loop-v1" {
		t.Fatalf("report schema/status/level = %s/%s/%s", report.Schema, report.Status, report.Level)
	}
	if len(report.Reloads) != 1 {
		t.Fatalf("reload count = %d, want 1", len(report.Reloads))
	}
	reload := report.Reloads[0]
	if reload.Kind != "source-change-reload" || reload.PreviousSHA256 == reload.CurrentSHA256 || !reload.ChangeDetected || !reload.ReloadApplied || !reload.InspectorUpdate {
		t.Fatalf("reload trace = %#v, want end-to-end source change reload evidence", reload)
	}
	if report.StatePreservation.Policy == "" || report.StatePreservation.Decision == "" || !report.StatePreservation.SchemaCompatible {
		t.Fatalf("state preservation = %#v", report.StatePreservation)
	}
}

func TestSurfaceDevRejectsReportWithoutObservedChange(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "SurfaceDesk")
	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"new", "surface-app", appDir}, &stdout, &stderr); code != 0 {
		t.Fatalf("new surface-app exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	reportPath := filepath.Join(appDir, ".tetra", "surface-dev-report.json")
	code := runCLI([]string{"surface", "dev", "--project", appDir, "--once", "--report", reportPath, "--require-change"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("surface dev accepted missing change trace, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "source change trace") {
		t.Fatalf("stderr = %q, want source change trace rejection", stderr.String())
	}
}

func TestSurfacePackageAcceptsProjectBeforeOutputFlag(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "SurfaceDesk")
	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"new", "surface-app", appDir}, &stdout, &stderr); code != 0 {
		t.Fatalf("new surface-app exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	outPath := filepath.Join(dir, "surface-desk.tdx")
	stdout.Reset()
	stderr.Reset()
	code := runCLI([]string{"surface", "package", appDir, "-o", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("surface package exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected package %s: %v", outPath, err)
	}
}
