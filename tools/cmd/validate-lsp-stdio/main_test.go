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
	transcript := lspFrame(`{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"textDocumentSync":1,"documentSymbolProvider":true,"hoverProvider":true,"definitionProvider":true,"referencesProvider":true,"renameProvider":true,"completionProvider":{"resolveProvider":false},"documentFormattingProvider":true,"codeActionProvider":true}}}`) +
		lspFrame(`{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///sample.tetra","diagnostics":[]}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":2,"result":[{"name":"answer","kind":14,"range":{"start":{"line":0,"character":6},"end":{"line":0,"character":12}},"selectionRange":{"start":{"line":0,"character":6},"end":{"line":0,"character":12}}}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":3,"result":{"contents":{"kind":"markdown","value":"const answer: i32"}}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":4,"result":[{"label":"answer","kind":21,"detail":"const answer: i32"}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":5,"result":[{"uri":"file:///sample.tetra","range":{"start":{"line":0,"character":6},"end":{"line":0,"character":12}}}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":6,"result":[{"uri":"file:///sample.tetra","range":{"start":{"line":0,"character":6},"end":{"line":0,"character":12}}},{"uri":"file:///sample.tetra","range":{"start":{"line":3,"character":11},"end":{"line":3,"character":17}}}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":7,"result":{"changes":{"file:///sample.tetra":[{"range":{"start":{"line":0,"character":6},"end":{"line":0,"character":12}},"newText":"value"}]}}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":8,"result":[{"range":{"start":{"line":0,"character":0},"end":{"line":4,"character":0}},"newText":"const answer: Int = 42\n\nfunc main() -> Int:\n    return answer\n"}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":9,"result":[{"title":"Add uses io to function main","kind":"quickfix","edit":{"changes":{"file:///sample.tetra":[{"range":{"start":{"line":2,"character":18},"end":{"line":2,"character":18}},"newText":" uses io"}]}}}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":10,"result":null}`)
	out, err := runStdioValidator(t, transcript)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateLSPStdioRejectsMissingHoverResponse(t *testing.T) {
	transcript := lspFrame(`{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"textDocumentSync":1,"documentSymbolProvider":true,"hoverProvider":true,"definitionProvider":true,"referencesProvider":true,"renameProvider":true,"completionProvider":{"resolveProvider":false},"documentFormattingProvider":true,"codeActionProvider":true}}}`) +
		lspFrame(`{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///sample.tetra","diagnostics":[]}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":2,"result":[{"name":"answer","kind":14}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":4,"result":[{"label":"answer","kind":21}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":5,"result":[{"uri":"file:///sample.tetra"}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":6,"result":[{"uri":"file:///sample.tetra"}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":7,"result":{"changes":{"file:///sample.tetra":[]}}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":8,"result":[{"newText":"formatted"}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":9,"result":[{"title":"Add uses io to function main","kind":"quickfix"}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":10,"result":null}`)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing hover response") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPStdioRejectsMissingDiagnosticsNotification(t *testing.T) {
	transcript := lspFrame(`{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"textDocumentSync":1,"documentSymbolProvider":true,"hoverProvider":true,"definitionProvider":true,"referencesProvider":true,"renameProvider":true,"completionProvider":{"resolveProvider":false},"documentFormattingProvider":true,"codeActionProvider":true}}}`) +
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
	transcript := lspFrame(`{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"textDocumentSync":1,"documentSymbolProvider":true,"definitionProvider":true,"referencesProvider":true,"renameProvider":true,"completionProvider":{"resolveProvider":false},"documentFormattingProvider":true,"codeActionProvider":true}}}`) +
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
	transcript := lspFrame(`{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"textDocumentSync":1,"documentSymbolProvider":true,"hoverProvider":true,"definitionProvider":true,"referencesProvider":true,"renameProvider":true,"documentFormattingProvider":true,"codeActionProvider":true}}}`) +
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

func TestValidateLSPStdioRejectsMissingFormattingCapability(t *testing.T) {
	transcript := lspFrame(`{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"textDocumentSync":1,"documentSymbolProvider":true,"hoverProvider":true,"definitionProvider":true,"referencesProvider":true,"renameProvider":true,"completionProvider":{"resolveProvider":false},"codeActionProvider":true}}}`) +
		lspFrame(`{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///sample.tetra","diagnostics":[]}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":2,"result":null}`)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing documentFormattingProvider") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPStdioRejectsMissingCodeActionCapability(t *testing.T) {
	transcript := lspFrame(`{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"textDocumentSync":1,"documentSymbolProvider":true,"hoverProvider":true,"definitionProvider":true,"referencesProvider":true,"renameProvider":true,"completionProvider":{"resolveProvider":false},"documentFormattingProvider":true}}}`) +
		lspFrame(`{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///sample.tetra","diagnostics":[]}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":2,"result":null}`)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing codeActionProvider") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPStdioRejectsMissingDefinitionCapability(t *testing.T) {
	transcript := lspFrame(`{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"textDocumentSync":1,"documentSymbolProvider":true,"hoverProvider":true,"referencesProvider":true,"renameProvider":true,"completionProvider":{"resolveProvider":false},"documentFormattingProvider":true,"codeActionProvider":true}}}`) +
		lspFrame(`{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///sample.tetra","diagnostics":[]}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":2,"result":null}`)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing definitionProvider") {
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

func TestValidateLSPStdioRejectsMissingReferencesCapability(t *testing.T) {
	transcript := lspFrame(`{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"textDocumentSync":1,"documentSymbolProvider":true,"hoverProvider":true,"definitionProvider":true,"renameProvider":true,"completionProvider":{"resolveProvider":false},"documentFormattingProvider":true,"codeActionProvider":true}}}`) +
		lspFrame(`{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///sample.tetra","diagnostics":[]}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":2,"result":null}`)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing referencesProvider") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPStdioRejectsMissingRenameCapability(t *testing.T) {
	transcript := lspFrame(`{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"textDocumentSync":1,"documentSymbolProvider":true,"hoverProvider":true,"definitionProvider":true,"referencesProvider":true,"completionProvider":{"resolveProvider":false},"documentFormattingProvider":true,"codeActionProvider":true}}}`) +
		lspFrame(`{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///sample.tetra","diagnostics":[]}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":2,"result":null}`)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing renameProvider") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPStdioRejectsDuplicateDocumentSymbolResponse(t *testing.T) {
	transcript := lspFrame(`{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"textDocumentSync":1,"documentSymbolProvider":true,"hoverProvider":true,"definitionProvider":true,"referencesProvider":true,"renameProvider":true,"completionProvider":{"resolveProvider":false},"documentFormattingProvider":true,"codeActionProvider":true}}}`) +
		lspFrame(`{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///sample.tetra","diagnostics":[]}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":2,"result":[{"name":"answer","kind":14}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":2,"result":[{"name":"duplicate","kind":14}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":3,"result":{"contents":{"kind":"markdown","value":"const answer: i32"}}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":4,"result":[{"label":"answer","kind":21}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":5,"result":[{"uri":"file:///sample.tetra"}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":6,"result":[{"uri":"file:///sample.tetra"},{"uri":"file:///sample.tetra"}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":7,"result":{"changes":{"file:///sample.tetra":[]}}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":8,"result":[{"newText":"formatted"}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":9,"result":[{"title":"Add uses io to function main","kind":"quickfix"}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":10,"result":null}`)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "duplicate documentSymbol response") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPStdioRejectsDuplicateShutdownResponse(t *testing.T) {
	transcript := lspFrame(`{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"textDocumentSync":1,"documentSymbolProvider":true,"hoverProvider":true,"definitionProvider":true,"referencesProvider":true,"renameProvider":true,"completionProvider":{"resolveProvider":false},"documentFormattingProvider":true,"codeActionProvider":true}}}`) +
		lspFrame(`{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///sample.tetra","diagnostics":[]}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":2,"result":[{"name":"answer","kind":14}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":3,"result":{"contents":{"kind":"markdown","value":"const answer: i32"}}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":4,"result":[{"label":"answer","kind":21}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":5,"result":[{"uri":"file:///sample.tetra"}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":6,"result":[{"uri":"file:///sample.tetra"},{"uri":"file:///sample.tetra"}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":7,"result":{"changes":{"file:///sample.tetra":[]}}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":8,"result":[{"newText":"formatted"}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":9,"result":[{"title":"Add uses io to function main","kind":"quickfix"}]}`) +
		lspFrame(`{"jsonrpc":"2.0","id":10,"result":null}`) +
		lspFrame(`{"jsonrpc":"2.0","id":10,"result":null}`)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "duplicate shutdown response") {
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
