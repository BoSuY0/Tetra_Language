package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateLSPStdioAcceptsExpectedTranscript(t *testing.T) {
	transcript := lspFrame(`{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"textDocumentSync":1,"documentSymbolProvider":true,"hoverProvider":true,"completionProvider":{"resolveProvider":false}}}}`) +
		lspFrame(`{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///sample.tetra","diagnostics":[]}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":2,"result":null}`)
	out, err := runStdioValidator(t, transcript)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateLSPStdioRejectsMissingDiagnosticsNotification(t *testing.T) {
	transcript := lspFrame(`{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"textDocumentSync":1,"documentSymbolProvider":true,"hoverProvider":true,"completionProvider":{"resolveProvider":false}}}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":2,"result":null}`)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing textDocument/publishDiagnostics notification") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPStdioRejectsMissingHoverCapability(t *testing.T) {
	transcript := lspFrame(`{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"textDocumentSync":1,"documentSymbolProvider":true,"completionProvider":{"resolveProvider":false}}}}`) +
		lspFrame(`{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///sample.tetra","diagnostics":[]}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":2,"result":null}`)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing hoverProvider") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPStdioRejectsMissingCompletionCapability(t *testing.T) {
	transcript := lspFrame(`{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"textDocumentSync":1,"documentSymbolProvider":true,"hoverProvider":true}}}`) +
		lspFrame(`{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///sample.tetra","diagnostics":[]}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":2,"result":null}`)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing completionProvider") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPStdioRejectsMalformedFrameLength(t *testing.T) {
	transcript := "Content-Length: 99\r\n\r\n{}"
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "message body truncated") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func lspFrame(body string) string {
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
}

func runStdioValidator(t *testing.T, transcript string) ([]byte, error) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "lsp-stdio.out")
	if err := os.WriteFile(path, []byte(transcript), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".", "--transcript", path)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}
