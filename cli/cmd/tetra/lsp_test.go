package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler"
	"tetra_language/internal/toon"
)

func TestLSPCommandSmoke(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	src := "func main() -> Int:\n    return 0\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"lsp", "--stdio-smoke", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"symbols"`) || !strings.Contains(stdout.String(), `"main"`) {
		t.Fatalf("lsp stdout = %q", stdout.String())
	}
}

func TestLSPCommandSmokeTOONFormat(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	src := "func main() -> Int:\n    return 0\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"lsp", "--stdio-smoke", srcPath, "--format=toon"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	jsonRaw, err := toon.ConvertTOONToJSON(stdout.Bytes(), toon.Options{Strict: true})
	if err != nil {
		t.Fatalf("TOON smoke output did not decode: %v\n%s", err, stdout.String())
	}
	var analysis compiler.LSPAnalysis
	if err := json.Unmarshal(jsonRaw, &analysis); err != nil {
		t.Fatalf("json.Unmarshal converted TOON: %v\nTOON:\n%s\nJSON:\n%s", err, stdout.String(), jsonRaw)
	}
	if len(analysis.Symbols) == 0 || analysis.Symbols[0].Name != "main" {
		t.Fatalf("TOON smoke analysis missing main symbol: %#v", analysis)
	}
}

func TestLSPStdioRejectsTOONFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"lsp", "--stdio", "--format=toon"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("lsp exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "stdio uses framed JSON-RPC") {
		t.Fatalf("stderr missing JSON-RPC boundary explanation: %q", stderr.String())
	}
}

func TestLSPSymbolKindMapsGlobals(t *testing.T) {
	if got := lspSymbolKind("const"); got != 14 {
		t.Fatalf("const symbol kind = %d, want 14", got)
	}
	if got := lspSymbolKind("val"); got != 13 {
		t.Fatalf("val symbol kind = %d, want 13", got)
	}
	if got := lspSymbolKind("var"); got != 13 {
		t.Fatalf("var symbol kind = %d, want 13", got)
	}
}

func TestLSPDocumentSymbolsIncludeDetail(t *testing.T) {
	got := lspDocumentSymbols(compiler.LSPAnalysis{
		Symbols: []compiler.LSPSymbol{{
			Name:   "answer",
			Kind:   "const",
			Line:   1,
			Column: 1,
			Detail: "const answer: Int",
		}},
	})
	if len(got) != 1 {
		t.Fatalf("symbols = %#v", got)
	}
	if got[0]["detail"] != "const answer: Int" {
		t.Fatalf("symbol = %#v", got[0])
	}
}

func TestLSPStdioInitializeAndDidOpen(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"func main() -> Int:\n    print(\"x\")\n    return 0\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `"id":1`) || !strings.Contains(out, `"capabilities"`) {
		t.Fatalf("initialize response missing: %q", out)
	}
	if !strings.Contains(out, `"completionProvider"`) {
		t.Fatalf("completion capability missing: %q", out)
	}
	if !strings.Contains(out, `"definitionProvider":true`) {
		t.Fatalf("definition capability missing: %q", out)
	}
	if !strings.Contains(out, `"referencesProvider":true`) {
		t.Fatalf("references capability missing: %q", out)
	}
	if !strings.Contains(out, `"renameProvider":true`) {
		t.Fatalf("rename capability missing: %q", out)
	}
	if !strings.Contains(out, `"documentFormattingProvider":true`) {
		t.Fatalf("document formatting capability missing: %q", out)
	}
	if !strings.Contains(out, `"codeActionProvider":true`) {
		t.Fatalf("code action capability missing: %q", out)
	}
	if !strings.Contains(out, `"method":"textDocument/publishDiagnostics"`) || !strings.Contains(out, `"diagnostics"`) {
		t.Fatalf("diagnostics notification missing: %q", out)
	}
	if !strings.Contains(out, `"id":2`) {
		t.Fatalf("shutdown response missing: %q", out)
	}
}

func TestLSPStdioDidOpenPublishesPrivacyConsentDiagnosticCode(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///privacy.tetra","languageId":"tetra","version":1,"text":"func seal(token: consent.token) -> secret.i32\nuses privacy:\n    return core.secret_seal_i32(1, token)\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	msgs := readLSPTestMessages(t, stdout.String())
	var publish map[string]any
	for _, msg := range msgs {
		if method, _ := msg["method"].(string); method == "textDocument/publishDiagnostics" {
			publish = msg
			break
		}
	}
	if publish == nil {
		t.Fatalf("publishDiagnostics notification missing: %#v", msgs)
	}
	params, ok := publish["params"].(map[string]any)
	if !ok {
		t.Fatalf("publishDiagnostics params missing: %#v", publish)
	}
	diagnostics, ok := params["diagnostics"].([]any)
	if !ok || len(diagnostics) == 0 {
		t.Fatalf("publishDiagnostics diagnostics missing: %#v", publish)
	}
	first, ok := diagnostics[0].(map[string]any)
	if !ok {
		t.Fatalf("diagnostic entry malformed: %#v", diagnostics[0])
	}
	if code, _ := first["code"].(string); code != compiler.DiagnosticCodeSafetyPrivacy {
		t.Fatalf("diagnostic code = %#v, want %q: %#v", first["code"], compiler.DiagnosticCodeSafetyPrivacy, publish)
	}
}

func TestLSPStdioDidOpenPublishesEffectPolicyDiagnosticCode(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///effect.tetra","languageId":"tetra","version":1,"text":"func main() -> Int:\n    print(\"x\")\n    return 0\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	msgs := readLSPTestMessages(t, stdout.String())
	var publish map[string]any
	for _, msg := range msgs {
		if method, _ := msg["method"].(string); method == "textDocument/publishDiagnostics" {
			publish = msg
			break
		}
	}
	if publish == nil {
		t.Fatalf("publishDiagnostics notification missing: %#v", msgs)
	}
	params, ok := publish["params"].(map[string]any)
	if !ok {
		t.Fatalf("publishDiagnostics params missing: %#v", publish)
	}
	diagnostics, ok := params["diagnostics"].([]any)
	if !ok || len(diagnostics) == 0 {
		t.Fatalf("publishDiagnostics diagnostics missing: %#v", publish)
	}
	first, ok := diagnostics[0].(map[string]any)
	if !ok {
		t.Fatalf("diagnostic entry malformed: %#v", diagnostics[0])
	}
	if code, _ := first["code"].(string); code != compiler.DiagnosticCodeSafetyEffect {
		t.Fatalf("diagnostic code = %#v, want %q: %#v", first["code"], compiler.DiagnosticCodeSafetyEffect, publish)
	}
}

func TestLSPStdioUnknownRequestMethodReturnsJSONRPCError(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":99,"method":"tetra/unknown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":100,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	msgs := readLSPTestMessages(t, stdout.String())
	assertLSPTestError(t, msgs[0], 99, -32601, "unknown method")
	if _, ok := msgs[0]["result"]; ok {
		t.Fatalf("unknown method returned success result: %#v", msgs[0])
	}
}

func TestLSPStdioStringRequestIDPreservesCorrelation(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":"init-1","method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"1.0","id":"bad-version","method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":"bad-1","method":"tetra/unknown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":"stop-1","method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	msgs := readLSPTestMessages(t, stdout.String())
	assertLSPTestResultObject(t, msgs[0], "init-1")
	assertLSPTestError(t, msgs[1], "bad-version", -32600, "jsonrpc")
	assertLSPTestError(t, msgs[2], "bad-1", -32601, "unknown method")
	assertLSPTestResultNil(t, msgs[3], "stop-1")
}

func TestLSPStdioInvalidJSONRPCVersionReturnsRequestError(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"1.0","id":7,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":8,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	msgs := readLSPTestMessages(t, stdout.String())
	assertLSPTestError(t, msgs[0], 7, -32600, "jsonrpc")
}

func TestLSPStdioMalformedRequestParamsReturnsInvalidParamsError(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"textDocument/hover","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":"bad","character":0}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	msgs := readLSPTestMessages(t, stdout.String())
	assertLSPTestError(t, msgs[1], 2, -32602, "invalid params")
}

func TestLSPStdioUnopenedDocumentRequestsUseDocumentedEmptyPolicy(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"textDocument/documentSymbol","params":{"textDocument":{"uri":"file:///missing.tetra"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"textDocument/hover","params":{"textDocument":{"uri":"file:///missing.tetra"},"position":{"line":0,"character":0}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":4,"method":"textDocument/completion","params":{"textDocument":{"uri":"file:///missing.tetra"},"position":{"line":0,"character":0}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":5,"method":"textDocument/definition","params":{"textDocument":{"uri":"file:///missing.tetra"},"position":{"line":0,"character":0}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":6,"method":"textDocument/references","params":{"textDocument":{"uri":"file:///missing.tetra"},"position":{"line":0,"character":0},"context":{"includeDeclaration":true}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":7,"method":"textDocument/rename","params":{"textDocument":{"uri":"file:///missing.tetra"},"position":{"line":0,"character":0},"newName":"value"}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":8,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	msgs := readLSPTestMessages(t, stdout.String())
	assertLSPTestResultArrayLen(t, msgs[1], 2, 0)
	assertLSPTestResultNil(t, msgs[2], 3)
	assertLSPTestResultArrayLen(t, msgs[3], 4, 0)
	assertLSPTestResultNil(t, msgs[4], 5)
	assertLSPTestResultArrayLen(t, msgs[5], 6, 0)
	assertLSPTestResultNil(t, msgs[6], 7)
}

func TestLSPStdioTranscriptFixtureCoversEditingRequests(t *testing.T) {
	var input bytes.Buffer
	for _, body := range loadLSPTranscriptFixture(t, "full_session.jsonl") {
		writeLSPTestMessage(t, &input, body)
	}
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio fixture exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		`"id":1`,
		`"id":2`,
		`"selectionRange"`,
		`"id":3`,
		`"contents":{"kind":"markdown","value":"const answer: i32"}`,
		`"id":4`,
		`"label":"answer"`,
		`"id":5`,
		`"start":{"character":6,"line":0}`,
		`"id":6`,
		`"uri":"file:///fixture.tetra"`,
		`"id":7`,
		`"newText":"value"`,
		`"id":8`,
		`"newText":"const answer: Int = 42\n\nfunc main() -> Int:\n    return answer\n"`,
		`function 'main' uses effect 'io' but does not declare it`,
		`"id":9`,
		`"title":"Add uses io to function main"`,
		`"id":10`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("fixture transcript output missing %q:\n%s", want, out)
		}
	}
	if got := strings.Count(out, `"method":"textDocument/publishDiagnostics"`); got != 2 {
		t.Fatalf("publish diagnostics count = %d, stdout=%q", got, out)
	}
}

func TestLSPStdioCodeActionReturnsMissingUsesQuickFix(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"func main() -> Int:\n    print(\"x\")\n    return 0\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"textDocument/codeAction","params":{"textDocument":{"uri":"file:///sample.tetra"},"range":{"start":{"line":1,"character":4},"end":{"line":1,"character":9}},"context":{"diagnostics":[{"range":{"start":{"line":1,"character":4},"end":{"line":1,"character":9}},"severity":1,"code":"TETRA2001","source":"tetra","message":"function 'main' uses effect 'io' but does not declare it"}]}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)

	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, `"id":2`) {
		t.Fatalf("codeAction response missing: %q", out)
	}
	if !strings.Contains(out, `"title":"Add uses io to function main"`) {
		t.Fatalf("codeAction title missing: %q", out)
	}
	if !strings.Contains(out, `"kind":"quickfix"`) {
		t.Fatalf("codeAction kind missing: %q", out)
	}
	if !strings.Contains(out, `"newText":" uses io"`) {
		t.Fatalf("codeAction edit missing insertion text: %q", out)
	}
	if !strings.Contains(out, `"start":{"character":18,"line":0}`) || !strings.Contains(out, `"end":{"character":18,"line":0}`) {
		t.Fatalf("codeAction edit missing insertion range: %q", out)
	}
}

func TestLSPStdioCompletionReturnsOpenDocumentSymbols(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"const answer: Int = 42\n\nfunc main() -> Int:\n    return answer\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"textDocument/completion","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":3,"character":11}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `"id":2`) || !strings.Contains(out, `"label":"answer"`) || !strings.Contains(out, `"label":"main"`) {
		t.Fatalf("completion response missing expected symbols: %q", out)
	}
	if !strings.Contains(out, `"detail":"const answer: i32"`) {
		t.Fatalf("completion response missing detail: %q", out)
	}
}

func TestLSPStdioDefinitionReturnsOpenDocumentSymbolLocation(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"const answer: Int = 42\n\nfunc main() -> Int:\n    return answer\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"textDocument/definition","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":3,"character":11}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `"id":2`) || !strings.Contains(out, `"uri":"file:///sample.tetra"`) {
		t.Fatalf("definition response missing location uri: %q", out)
	}
	if !strings.Contains(out, `"start":{"character":6,"line":0}`) || !strings.Contains(out, `"end":{"character":12,"line":0}`) {
		t.Fatalf("definition response missing expected symbol range: %q", out)
	}
}

func TestLSPStdioReferencesReturnsOpenDocumentLocations(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"const answer: Int = 42\n\nfunc main() -> Int:\n    return answer + answer\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"textDocument/references","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":3,"character":11},"context":{"includeDeclaration":true}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `"id":2`) {
		t.Fatalf("references response missing: %q", out)
	}
	if got := strings.Count(out, `"uri":"file:///sample.tetra"`); got < 3 {
		t.Fatalf("references response missing locations: %q", out)
	}
	if !strings.Contains(out, `"start":{"character":6,"line":0}`) || !strings.Contains(out, `"end":{"character":12,"line":0}`) {
		t.Fatalf("references response missing declaration location: %q", out)
	}
	if !strings.Contains(out, `"start":{"character":11,"line":3}`) {
		t.Fatalf("references response missing first usage location: %q", out)
	}
	if !strings.Contains(out, `"start":{"character":20,"line":3}`) {
		t.Fatalf("references response missing second usage location: %q", out)
	}
}

func TestLSPStdioReferencesSkipsCommentsAndStrings(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"const answer: Int = 42\n\nfunc main() -> Int:\n    print(\"answer\")\n    // answer is documentation only\n    return answer\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"textDocument/references","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":5,"character":11},"context":{"includeDeclaration":true}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	msgs := readLSPTestMessages(t, stdout.String())
	response := lspTestMessageByID(t, msgs, 2)
	result, ok := response["result"].([]any)
	if !ok {
		t.Fatalf("references result is not an array: %#v", response)
	}
	if len(result) != 2 {
		t.Fatalf("references result len = %d, want 2: %#v", len(result), response)
	}
	assertLSPTestLocationsContainRange(t, result, 0, 6)
	assertLSPTestLocationsContainRange(t, result, 5, 11)
	assertLSPTestLocationsDoNotContainRange(t, result, 3, 11)
	assertLSPTestLocationsDoNotContainRange(t, result, 4, 7)
}

func TestLSPStdioRenameReturnsWorkspaceEditForOpenDocument(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"const answer: Int = 42\n\nfunc main() -> Int:\n    return answer + answer\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"textDocument/rename","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":3,"character":11},"newName":"value"}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `"id":2`) {
		t.Fatalf("rename response missing: %q", out)
	}
	if !strings.Contains(out, `"changes":{"file:///sample.tetra":[`) {
		t.Fatalf("rename workspace edit missing: %q", out)
	}
	if !strings.Contains(out, `"newText":"value"`) {
		t.Fatalf("rename edits missing newText: %q", out)
	}
	if !strings.Contains(out, `"start":{"character":6,"line":0}`) {
		t.Fatalf("rename edits missing declaration location: %q", out)
	}
	if !strings.Contains(out, `"start":{"character":11,"line":3}`) {
		t.Fatalf("rename edits missing first usage location: %q", out)
	}
	if !strings.Contains(out, `"start":{"character":20,"line":3}`) {
		t.Fatalf("rename edits missing second usage location: %q", out)
	}
}

func TestLSPStdioRenameSkipsCommentsAndStrings(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"const answer: Int = 42\n\nfunc main() -> Int:\n    print(\"answer\")\n    // answer is documentation only\n    return answer\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"textDocument/rename","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":5,"character":11},"newName":"value"}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	msgs := readLSPTestMessages(t, stdout.String())
	response := lspTestMessageByID(t, msgs, 2)
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("rename result is not an object: %#v", response)
	}
	changes, ok := result["changes"].(map[string]any)
	if !ok {
		t.Fatalf("rename result missing changes: %#v", response)
	}
	edits, ok := changes["file:///sample.tetra"].([]any)
	if !ok {
		t.Fatalf("rename result missing sample edits: %#v", response)
	}
	if len(edits) != 2 {
		t.Fatalf("rename edit len = %d, want 2: %#v", len(edits), response)
	}
	assertLSPTestLocationsContainRange(t, edits, 0, 6)
	assertLSPTestLocationsContainRange(t, edits, 5, 11)
	assertLSPTestLocationsDoNotContainRange(t, edits, 3, 11)
	assertLSPTestLocationsDoNotContainRange(t, edits, 4, 7)
}

func TestLSPStdioRenameRejectsInvalidNewName(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"const answer: Int = 42\n\nfunc main() -> Int:\n    return answer\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"textDocument/rename","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":3,"character":11},"newName":"bad-name"}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	msgs := readLSPTestMessages(t, stdout.String())
	response := lspTestMessageByID(t, msgs, 2)
	assertLSPTestError(t, response, 2, -32602, "rename newName must be a Tetra identifier")
	if _, ok := response["result"]; ok {
		t.Fatalf("invalid rename returned success result: %#v", response)
	}
}

func TestLSPStdioRenameRejectsLocalShadowing(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"const answer: Int = 42\n\nfunc main() -> Int:\n    let answer: Int = 7\n    return answer\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"textDocument/rename","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":0,"character":7},"newName":"value"}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	msgs := readLSPTestMessages(t, stdout.String())
	response := lspTestMessageByID(t, msgs, 2)
	assertLSPTestResultNil(t, response, 2)
}

func TestLSPStdioFormattingReturnsFullDocumentEdit(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"func main() -> Int:\n  return 0\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"textDocument/formatting","params":{"textDocument":{"uri":"file:///sample.tetra"},"options":{"tabSize":4,"insertSpaces":true}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `"id":2`) || !strings.Contains(out, `"newText"`) || !strings.Contains(out, `\n    return 0\n`) {
		t.Fatalf("formatting response missing formatted full-document edit: %q", out)
	}
	if !strings.Contains(out, `"end":{"character":0,"line":2}`) {
		t.Fatalf("formatting response missing full document range: %q", out)
	}
}

func TestLSPStdioDidChangePublishesUpdatedDiagnostics(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"func main() -> Int:\n    return 0\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file:///sample.tetra","version":2},"contentChanges":[{"text":"func main() -> Int:\n    print(\"x\")\n    return 0\n"}]}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	if got := strings.Count(out, `"method":"textDocument/publishDiagnostics"`); got != 2 {
		t.Fatalf("publish diagnostics count = %d, stdout=%q", got, out)
	}
	if !strings.Contains(out, `function 'main' uses effect 'io' but does not declare it`) {
		t.Fatalf("updated diagnostic missing: %q", out)
	}
}

func TestLSPStdioDidCloseClearsDiagnostics(t *testing.T) {
	var input bytes.Buffer
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"func main() -> Int:\n    print(\"x\")\n    return 0\n"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"textDocument/didClose","params":{"textDocument":{"uri":"file:///sample.tetra"}}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","id":2,"method":"shutdown","params":{}}`)
	writeLSPTestMessage(t, &input, `{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var stdout, stderr bytes.Buffer
	code := runLSPStdio(&input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lsp stdio exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	if got := strings.Count(out, `"method":"textDocument/publishDiagnostics"`); got != 2 {
		t.Fatalf("publish diagnostics count = %d, stdout=%q", got, out)
	}
	if !strings.Contains(out, `function 'main' uses effect 'io' but does not declare it`) {
		t.Fatalf("initial diagnostic missing: %q", out)
	}
	if !strings.Contains(out, `"diagnostics":[]`) {
		t.Fatalf("didClose did not publish empty diagnostics: %q", out)
	}
}

func writeLSPTestMessage(t *testing.T, w *bytes.Buffer, body string) {
	t.Helper()
	fmt.Fprintf(w, "Content-Length: %d\r\n\r\n%s", len(body), body)
}

func readLSPTestMessages(t *testing.T, transcript string) []map[string]any {
	t.Helper()
	reader := bufio.NewReader(strings.NewReader(transcript))
	var msgs []map[string]any
	for {
		body, err := readLSPMessage(reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read LSP response: %v\n%s", err, transcript)
		}
		var msg map[string]any
		if err := json.Unmarshal(body, &msg); err != nil {
			t.Fatalf("decode LSP response %q: %v", string(body), err)
		}
		msgs = append(msgs, msg)
	}
	return msgs
}

func assertLSPTestError(t *testing.T, msg map[string]any, id any, code int, messagePart string) {
	t.Helper()
	assertLSPTestID(t, msg, id)
	errObj, ok := msg["error"].(map[string]any)
	if !ok {
		t.Fatalf("message error missing: %#v", msg)
	}
	if got := int(errObj["code"].(float64)); got != code {
		t.Fatalf("error code = %d, want %d: %#v", got, code, msg)
	}
	if got, _ := errObj["message"].(string); !strings.Contains(got, messagePart) {
		t.Fatalf("error message = %q, want containing %q: %#v", got, messagePart, msg)
	}
}

func assertLSPTestResultArrayLen(t *testing.T, msg map[string]any, id any, want int) {
	t.Helper()
	assertLSPTestID(t, msg, id)
	result, ok := msg["result"].([]any)
	if !ok {
		t.Fatalf("message result is not an array: %#v", msg)
	}
	if len(result) != want {
		t.Fatalf("result len = %d, want %d: %#v", len(result), want, msg)
	}
}

func assertLSPTestResultNil(t *testing.T, msg map[string]any, id any) {
	t.Helper()
	assertLSPTestID(t, msg, id)
	if result, ok := msg["result"]; !ok || result != nil {
		t.Fatalf("message result = %#v, want nil: %#v", result, msg)
	}
}

func assertLSPTestResultObject(t *testing.T, msg map[string]any, id any) {
	t.Helper()
	assertLSPTestID(t, msg, id)
	if _, ok := msg["result"].(map[string]any); !ok {
		t.Fatalf("message result is not an object: %#v", msg)
	}
}

func lspTestMessageByID(t *testing.T, msgs []map[string]any, id any) map[string]any {
	t.Helper()
	for _, msg := range msgs {
		switch want := id.(type) {
		case int:
			got, ok := msg["id"].(float64)
			if ok && int(got) == want {
				return msg
			}
		case string:
			got, ok := msg["id"].(string)
			if ok && got == want {
				return msg
			}
		default:
			t.Fatalf("unsupported test id type %T", id)
		}
	}
	t.Fatalf("message id %v not found in %#v", id, msgs)
	return nil
}

func assertLSPTestLocationsContainRange(t *testing.T, locations []any, line int, character int) {
	t.Helper()
	if !lspTestLocationsContainRange(locations, line, character) {
		t.Fatalf("locations missing range start line=%d character=%d: %#v", line, character, locations)
	}
}

func assertLSPTestLocationsDoNotContainRange(t *testing.T, locations []any, line int, character int) {
	t.Helper()
	if lspTestLocationsContainRange(locations, line, character) {
		t.Fatalf("locations unexpectedly include range start line=%d character=%d: %#v", line, character, locations)
	}
}

func lspTestLocationsContainRange(locations []any, line int, character int) bool {
	for _, item := range locations {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		rangeObj, ok := obj["range"].(map[string]any)
		if !ok {
			continue
		}
		start, ok := rangeObj["start"].(map[string]any)
		if !ok {
			continue
		}
		gotLine, lineOK := start["line"].(float64)
		gotCharacter, characterOK := start["character"].(float64)
		if lineOK && characterOK && int(gotLine) == line && int(gotCharacter) == character {
			return true
		}
	}
	return false
}

func assertLSPTestID(t *testing.T, msg map[string]any, id any) {
	t.Helper()
	switch want := id.(type) {
	case int:
		got, ok := msg["id"].(float64)
		if !ok || int(got) != want {
			t.Fatalf("message id = %#v, want %d: %#v", msg["id"], want, msg)
		}
	case string:
		got, ok := msg["id"].(string)
		if !ok || got != want {
			t.Fatalf("message id = %#v, want %q: %#v", msg["id"], want, msg)
		}
	default:
		t.Fatalf("unsupported test id type %T", id)
	}
}

func loadLSPTranscriptFixture(t *testing.T, name string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "lsp", name))
	if err != nil {
		t.Fatalf("read LSP fixture: %v", err)
	}
	var bodies []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		bodies = append(bodies, line)
	}
	if len(bodies) == 0 {
		t.Fatalf("LSP fixture %s is empty", name)
	}
	return bodies
}
