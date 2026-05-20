package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler"
)

func TestVersionCommand(t *testing.T) {
	var stdout bytes.Buffer
	code := runCLI([]string{"version"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("version exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), compiler.Version()) {
		t.Fatalf("version output = %q, want compiler version", stdout.String())
	}
}

func TestCLIContractDocumentedCommandsHaveHelpAndInvalidArgBehavior(t *testing.T) {
	commands := documentedCLICommands(t)
	if len(commands) == 0 {
		t.Fatal("no documented CLI commands found")
	}
	for _, command := range commands {
		t.Run(command+"_help", func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runCLI([]string{command, "--help"}, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("%s --help exit code = %d, stdout=%q stderr=%q", command, code, stdout.String(), stderr.String())
			}
			combined := stdout.String() + stderr.String()
			if !strings.Contains(strings.ToLower(combined), command) && !strings.Contains(strings.ToLower(combined), "usage") {
				t.Fatalf("%s --help output does not describe the command: stdout=%q stderr=%q", command, stdout.String(), stderr.String())
			}
		})
		t.Run(command+"_invalid_arg", func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runCLI([]string{command, "--definitely-invalid"}, &stdout, &stderr)
			if code != 2 {
				t.Fatalf("%s invalid arg exit code = %d, stdout=%q stderr=%q", command, code, stdout.String(), stderr.String())
			}
		})
	}
}

func documentedCLICommands(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "spec", "cli_contracts.md"))
	if err != nil {
		t.Fatalf("read cli contracts: %v", err)
	}
	seen := map[string]bool{}
	var commands []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		rest := strings.TrimPrefix(line, "| `")
		command, _, ok := strings.Cut(rest, "`")
		if !ok || command == "tetra" || strings.Contains(command, " ") || command == "" || command[0] < 'a' || command[0] > 'z' {
			continue
		}
		if !seen[command] {
			seen[command] = true
			commands = append(commands, command)
		}
	}
	return commands
}
