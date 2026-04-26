package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestTestAllScriptInterface(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	script := filepath.Join(root, "scripts", "test_all.sh")

	if out, err := exec.Command("bash", "-n", script).CombinedOutput(); err != nil {
		t.Fatalf("bash -n failed: %v\n%s", err, string(out))
	}

	help := exec.Command("bash", script, "--help")
	help.Dir = root
	helpOut, err := help.CombinedOutput()
	if err != nil {
		t.Fatalf("--help failed: %v\n%s", err, string(helpOut))
	}
	for _, want := range []string{"--keep-going", "--json-only", "Exit codes", "--report-dir"} {
		if !strings.Contains(string(helpOut), want) {
			t.Fatalf("help missing %q:\n%s", want, string(helpOut))
		}
	}

	bad := exec.Command("bash", script, "--definitely-not-a-real-option")
	bad.Dir = root
	badOut, err := bad.CombinedOutput()
	if err == nil {
		t.Fatalf("invalid option unexpectedly succeeded:\n%s", string(badOut))
	}
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 2 {
		t.Fatalf("invalid option exit = %v, output:\n%s", err, string(badOut))
	}
}

func TestTestAllScriptKeepGoingJSONOnly(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	scriptRaw, err := os.ReadFile(filepath.Join(root, "scripts", "test_all.sh"))
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scripts", "test_all.sh"), scriptRaw, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scripts", "bootstrap.sh"), []byte("#!/usr/bin/env bash\ncp ./tetra ./t\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "go"), []byte("#!/usr/bin/env bash\nif [[ \"$1\" == \"test\" ]]; then exit 1; fi\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tetra"), []byte(`#!/usr/bin/env bash
case "$1" in
  version) echo "v0.6.99"; exit 0 ;;
  fmt|test|smoke) exit 0 ;;
  check)
    for arg in "$@"; do
      if [[ "$arg" == "--diagnostics=json" ]]; then
        case "$*" in
          *missing-effect-diagnostic.tetra*) echo '{"code":"TETRA2001","message":"function main uses effect '\''io'\'' but does not declare it","severity":"error"}' >&2 ;;
          *tabs-diagnostic.tetra*) echo '{"code":"TETRA0001","message":"tabs are not supported in Flow indentation","severity":"error"}' >&2 ;;
          *planned-actor-diagnostic.tetra*) echo '{"code":"TETRA0001","message":"planned feature '\''actor'\'' is not implemented","severity":"error"}' >&2 ;;
          *) echo '{"code":"TETRA2001","message":"unknown function missing_call","severity":"error"}' >&2 ;;
        esac
        exit 1
      fi
    done
    exit 0
    ;;
  build)
    for arg in "$@"; do
      if [[ "$arg" == "wasm32-wasi" ]]; then
        echo '{"code":"TETRA0001","message":"planned target not implemented: wasm32-wasi","severity":"error"}' >&2
        exit 2
      fi
    done
    exit 0
    ;;
  targets)
    echo '{"supported":["linux-x64","windows-x64","macos-x64"],"planned":["wasm32-wasi","wasm32-web"]}'
    exit 0
    ;;
  doctor)
    echo '{"status":"pass","checks":[{"name":"version","status":"pass"},{"name":"supported targets","status":"pass"},{"name":"planned targets","status":"pass"},{"name":"repo root","status":"pass"},{"name":"__rt/actors_sysv.tetra","status":"pass"},{"name":"__rt/actors_win64.tetra","status":"pass"},{"name":"compiler/selfhostrt/actors_sysv.tetra","status":"pass"},{"name":"compiler/selfhostrt/actors_win64.tetra","status":"pass"},{"name":"examples/flow_hello.tetra","status":"pass"},{"name":"docs/generated/manifest.json","status":"pass"},{"name":"docs manifest version","status":"pass"},{"name":"docs manifest surface","status":"pass"},{"name":"smoke sources","status":"pass"},{"name":"runtime exports","status":"pass"}]}'
    exit 0
    ;;
  *) exit 2 ;;
esac
`), 0o755); err != nil {
		t.Fatal(err)
	}

	reportDir := filepath.Join(dir, "report")
	cmd := exec.Command("bash", "scripts/test_all.sh", "--quick", "--keep-going", "--json-only", "--report-dir", reportDir)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected failing keep-going run, got success:\n%s", string(out))
	}
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
		t.Fatalf("exit = %v, output:\n%s", err, string(out))
	}

	var summary struct {
		Status string `json:"status"`
		Steps  []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(out, &summary); err != nil {
		t.Fatalf("summary JSON: %v\n%s", err, string(out))
	}
	if summary.Status != "fail" || len(summary.Steps) != 13 {
		t.Fatalf("summary = %#v", summary)
	}
	if summary.Steps[0].Name != "go test all packages" || summary.Steps[0].Status != "fail" {
		t.Fatalf("first step = %#v", summary.Steps[0])
	}
	if summary.Steps[len(summary.Steps)-1].Name != "host smoke linux-x64" || summary.Steps[len(summary.Steps)-1].Status != "pass" {
		t.Fatalf("last step = %#v", summary.Steps[len(summary.Steps)-1])
	}
	if _, err := os.Stat(filepath.Join(reportDir, "summary.md")); err != nil {
		t.Fatalf("missing summary.md: %v", err)
	}
}
