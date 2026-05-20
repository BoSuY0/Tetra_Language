package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestActorNetCommandHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"actor-net", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("actor-net --help exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "usage: tetra actor-net") {
		t.Fatalf("actor-net help = %q, want usage", stdout.String())
	}
}

func TestActorNetCommandRejectsInvalidArgs(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"actor-net", "--definitely-invalid"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("actor-net invalid arg exit code = %d, stderr=%q", code, stderr.String())
	}
}
