package main

import (
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
	script := filepath.Join(root, "scripts", "ci", "test-all.sh")

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
