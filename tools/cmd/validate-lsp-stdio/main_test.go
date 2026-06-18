package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestValidateLSPStdioAcceptsExpectedTranscript(t *testing.T) {
	transcript := lspFrame(
		("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"capabilities\":" +
			"{\"textDocumentSync\":1,\"documentSymbolProvider\":true," +
			"\"hoverProvider\":true,\"definitionProvider\":true," +
			"\"referencesProvider\":true,\"renameProvider\":true," +
			"\"completionProvider\":{\"resolveProvider\":false}," +
			"\"documentFormattingProvider\":true,\"codeActionProvider\":" +
			"true}}}"),
	) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/publishDiagnostics\"," +
				"\"params\":{\"uri\":\"file:///sample.tetra\",\"diagnostics\":[]}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":2,\"result\":[{\"name\":\"answer\",\"kind\":" +
				"14,\"range\":{\"start\":{\"line\":0,\"character\":6},\"end\":{\"line\":" +
				"0,\"character\":12}},\"selectionRange\":{\"start\":{\"line\":0," +
				"\"character\":6},\"end\":{\"line\":0,\"character\":12}}}]}"),
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":3,"result":{"contents":{"kind":"markdown","value":"const answer: i32"}}}`,
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":4,"result":[{"label":"answer","kind":21,"detail":"const answer: i32"}]}`,
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":5,\"result\":[{\"uri\":\"file:" +
				"///sample.tetra\",\"range\":{\"start\":{\"line\":0,\"character\":6}," +
				"\"end\":{\"line\":0,\"character\":12}}}]}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":6,\"result\":[{\"uri\":\"file:" +
				"///sample.tetra\",\"range\":{\"start\":{\"line\":0,\"character\":6}," +
				"\"end\":{\"line\":0,\"character\":12}}},{\"uri\":\"file:" +
				"///sample.tetra\",\"range\":{\"start\":{\"line\":3,\"character\":11}," +
				"\"end\":{\"line\":3,\"character\":17}}}]}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":7,\"result\":{\"changes\":{\"file:" +
				"///sample.tetra\":[{\"range\":{\"start\":{\"line\":0,\"character\":" +
				"6},\"end\":{\"line\":0,\"character\":12}},\"newText\":\"value\"}]}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":8,\"result\":[{\"range\":{\"start\":{\"line\":" +
				"0,\"character\":0},\"end\":{\"line\":4,\"character\":0}},\"newText\":" +
				"\"const answer: Int = 42\\n\\nfunc main() -> Int:\\n    return " +
				"answer\\n\"}]}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":9,\"result\":[{\"title\":\"Add uses io to " +
				"function main\",\"kind\":\"quickfix\",\"edit\":{\"changes\":{\"file:" +
				"///sample.tetra\":[{\"range\":{\"start\":{\"line\":2,\"character\":" +
				"18},\"end\":{\"line\":2,\"character\":18}},\"newText\":\" uses io\"}]" +
				"}}}]}"),
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":10,"result":null}`,
		)
	out, err := runStdioValidator(t, transcript)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateLSPStdioAcceptsRequestResponseTranscript(t *testing.T) {
	transcript := validRequestResponseTranscript()
	out, err := runStdioValidator(t, transcript)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateLSPStdioAcceptsStringIDRequestResponseTranscript(t *testing.T) {
	transcript := stringIDRequestResponseTranscript()
	out, err := runStdioValidator(t, transcript)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateLSPStdioAcceptsNumericLookingStringIDCorrelation(t *testing.T) {
	transcript := numericLookingStringIDRequestResponseTranscript()
	out, err := runStdioValidator(t, transcript)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateLSPStdioRejectsSwappedRequestMethods(t *testing.T) {
	transcript := lspFrame(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"capabilities\":" +
				"{\"textDocumentSync\":1,\"documentSymbolProvider\":true," +
				"\"hoverProvider\":true,\"definitionProvider\":true," +
				"\"referencesProvider\":true,\"renameProvider\":true," +
				"\"completionProvider\":{\"resolveProvider\":false}," +
				"\"documentFormattingProvider\":true,\"codeActionProvider\":" +
				"true}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didOpen\",\"params\":" +
				"{\"textDocument\":{\"uri\":\"file:///sample.tetra\",\"languageId\":" +
				"\"tetra\",\"version\":1,\"text\":\"const answer: Int = 42\\n\\nfunc " +
				"main() -> Int:\\n  return answer\\n\"}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/publishDiagnostics\"," +
				"\"params\":{\"uri\":\"file:///sample.tetra\",\"diagnostics\":[]}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"textDocument/hover\"," +
				"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
				"\"position\":{\"line\":0,\"character\":6}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":" +
				"\"textDocument/documentSymbol\",\"params\":{\"textDocument\":" +
				"{\"uri\":\"file:///sample.tetra\"}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":2,\"result\":[{\"name\":\"answer\",\"kind\":" +
				"14,\"range\":{\"start\":{\"line\":0,\"character\":6},\"end\":{\"line\":" +
				"0,\"character\":12}},\"selectionRange\":{\"start\":{\"line\":0," +
				"\"character\":6},\"end\":{\"line\":0,\"character\":12}}}]}"),
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":3,"result":{"contents":{"kind":"markdown","value":"const answer: i32"}}}`,
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":4,\"method\":\"textDocument/completion\"," +
				"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
				"\"position\":{\"line\":3,\"character\":9}}}"),
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":4,"result":[{"label":"answer","kind":21,"detail":"const answer: i32"}]}`,
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":5,\"method\":\"textDocument/definition\"," +
				"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
				"\"position\":{\"line\":3,\"character\":9}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":5,\"result\":[{\"uri\":\"file:" +
				"///sample.tetra\",\"range\":{\"start\":{\"line\":0,\"character\":6}," +
				"\"end\":{\"line\":0,\"character\":12}}}]}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":6,\"method\":\"textDocument/references\"," +
				"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
				"\"position\":{\"line\":3,\"character\":9},\"context\":" +
				"{\"includeDeclaration\":true}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":6,\"result\":[{\"uri\":\"file:" +
				"///sample.tetra\",\"range\":{\"start\":{\"line\":0,\"character\":6}," +
				"\"end\":{\"line\":0,\"character\":12}}},{\"uri\":\"file:" +
				"///sample.tetra\",\"range\":{\"start\":{\"line\":3,\"character\":11}," +
				"\"end\":{\"line\":3,\"character\":17}}}]}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":7,\"method\":\"textDocument/rename\"," +
				"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
				"\"position\":{\"line\":3,\"character\":9},\"newName\":\"value\"}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":7,\"result\":{\"changes\":{\"file:" +
				"///sample.tetra\":[{\"range\":{\"start\":{\"line\":0,\"character\":" +
				"6},\"end\":{\"line\":0,\"character\":12}},\"newText\":\"value\"}]}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":8,\"method\":\"textDocument/formatting\"," +
				"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
				"\"options\":{\"tabSize\":4,\"insertSpaces\":true}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":8,\"result\":[{\"range\":{\"start\":{\"line\":" +
				"0,\"character\":0},\"end\":{\"line\":4,\"character\":0}},\"newText\":" +
				"\"const answer: Int = 42\\n\\nfunc main() -> Int:\\n    return " +
				"answer\\n\"}]}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didChange\",\"params\":" +
				"{\"textDocument\":{\"uri\":\"file:///sample.tetra\",\"version\":2}," +
				"\"contentChanges\":[{\"text\":\"const answer: Int = 42\\n\\nfunc " +
				"main() -> Int:\\n    print(\\\"x\\\")\\n    return answer\\n\"}]}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/publishDiagnostics\"," +
				"\"params\":{\"uri\":\"file:///sample.tetra\",\"diagnostics\":" +
				"[{\"range\":{\"start\":{\"line\":3,\"character\":4},\"end\":{\"line\":3," +
				"\"character\":9}},\"severity\":1,\"code\":\"TETRA2001\",\"source\":" +
				"\"tetra\",\"message\":\"function 'main' uses effect 'io' but " +
				"does not declare it\"}]}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":9,\"method\":\"textDocument/codeAction\"," +
				"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
				"\"range\":{\"start\":{\"line\":3,\"character\":4},\"end\":{\"line\":3," +
				"\"character\":9}},\"context\":{\"diagnostics\":[{\"range\":{\"start\":" +
				"{\"line\":3,\"character\":4},\"end\":{\"line\":3,\"character\":9}}," +
				"\"severity\":1,\"code\":\"TETRA2001\",\"source\":\"tetra\",\"message\":" +
				"\"function 'main' uses effect 'io' but does not declare it\"}]" +
				"}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":9,\"result\":[{\"title\":\"Add uses io to " +
				"function main\",\"kind\":\"quickfix\",\"edit\":{\"changes\":{\"file:" +
				"///sample.tetra\":[{\"range\":{\"start\":{\"line\":2,\"character\":" +
				"18},\"end\":{\"line\":2,\"character\":18}},\"newText\":\" uses io\"}]" +
				"}}}]}"),
		) +
		lspFrame(`{"jsonrpc":"2.0","id":10,"method":"shutdown","params":{}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":10,"result":null}`) +
		lspFrame(`{"jsonrpc":"2.0","method":"exit","params":{}}`)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(
		string(out),
		"request id 2 method textDocument/hover, expected textDocument/documentSymbol",
	) {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPStdioRejectsMissingDiagnosticsAfterDidChange(t *testing.T) {
	transcript := strings.Replace(
		validRequestResponseTranscript(),
		postChangeDiagnosticsFrame(),
		"",
		1,
	)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(
		string(out),
		"missing textDocument/publishDiagnostics notification after textDocument/didChange",
	) {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPStdioRejectsMissingHoverResponse(t *testing.T) {
	transcript := lspFrame(
		("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"capabilities\":" +
			"{\"textDocumentSync\":1,\"documentSymbolProvider\":true," +
			"\"hoverProvider\":true,\"definitionProvider\":true," +
			"\"referencesProvider\":true,\"renameProvider\":true," +
			"\"completionProvider\":{\"resolveProvider\":false}," +
			"\"documentFormattingProvider\":true,\"codeActionProvider\":" +
			"true}}}"),
	) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/publishDiagnostics\"," +
				"\"params\":{\"uri\":\"file:///sample.tetra\",\"diagnostics\":[]}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":2,\"result\":[{\"name\":\"answer\",\"kind\":" +
				"14,\"range\":{\"start\":{\"line\":0,\"character\":6},\"end\":{\"line\":" +
				"0,\"character\":12}},\"selectionRange\":{\"start\":{\"line\":0," +
				"\"character\":6},\"end\":{\"line\":0,\"character\":12}}}]}"),
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":4,"result":[{"label":"answer","kind":21}]}`,
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":5,"result":[{"uri":"file:///sample.tetra"}]}`,
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":6,"result":[{"uri":"file:///sample.tetra"}]}`,
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":7,"result":{"changes":{"file:///sample.tetra":[]}}}`,
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":8,"result":[{"newText":"formatted"}]}`,
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":9,"result":[{"title":"Add uses io to function main","kind":"quickfix"}]}`,
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":10,"result":null}`,
		)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing hover response") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPStdioRejectsMissingDiagnosticsNotification(t *testing.T) {
	transcript := lspFrame(
		("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"capabilities\":" +
			"{\"textDocumentSync\":1,\"documentSymbolProvider\":true," +
			"\"hoverProvider\":true,\"definitionProvider\":true," +
			"\"referencesProvider\":true,\"renameProvider\":true," +
			"\"completionProvider\":{\"resolveProvider\":false}," +
			"\"documentFormattingProvider\":true,\"codeActionProvider\":" +
			"true}}}"),
	) +
		lspFrame(
			`{"jsonrpc":"2.0","id":2,"result":null}`,
		)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing textDocument/publishDiagnostics notification") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPStdioRejectsMissingHoverCapability(t *testing.T) {
	transcript := lspFrame(
		("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"capabilities\":" +
			"{\"textDocumentSync\":1,\"documentSymbolProvider\":true," +
			"\"definitionProvider\":true,\"referencesProvider\":true," +
			"\"renameProvider\":true,\"completionProvider\":" +
			"{\"resolveProvider\":false},\"documentFormattingProvider\":true," +
			"\"codeActionProvider\":true}}}"),
	) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/publishDiagnostics\"," +
				"\"params\":{\"uri\":\"file:///sample.tetra\",\"diagnostics\":[]}}"),
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":2,"result":null}`,
		)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing hoverProvider") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPStdioRejectsMissingCompletionCapability(t *testing.T) {
	transcript := lspFrame(
		("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"capabilities\":" +
			"{\"textDocumentSync\":1,\"documentSymbolProvider\":true," +
			"\"hoverProvider\":true,\"definitionProvider\":true," +
			"\"referencesProvider\":true,\"renameProvider\":true," +
			"\"documentFormattingProvider\":true,\"codeActionProvider\":" +
			"true}}}"),
	) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/publishDiagnostics\"," +
				"\"params\":{\"uri\":\"file:///sample.tetra\",\"diagnostics\":[]}}"),
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":2,"result":null}`,
		)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing completionProvider") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPStdioRejectsMissingFormattingCapability(t *testing.T) {
	transcript := lspFrame(
		("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"capabilities\":" +
			"{\"textDocumentSync\":1,\"documentSymbolProvider\":true," +
			"\"hoverProvider\":true,\"definitionProvider\":true," +
			"\"referencesProvider\":true,\"renameProvider\":true," +
			"\"completionProvider\":{\"resolveProvider\":false}," +
			"\"codeActionProvider\":true}}}"),
	) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/publishDiagnostics\"," +
				"\"params\":{\"uri\":\"file:///sample.tetra\",\"diagnostics\":[]}}"),
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":2,"result":null}`,
		)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing documentFormattingProvider") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPStdioRejectsMissingCodeActionCapability(t *testing.T) {
	transcript := lspFrame(
		("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"capabilities\":" +
			"{\"textDocumentSync\":1,\"documentSymbolProvider\":true," +
			"\"hoverProvider\":true,\"definitionProvider\":true," +
			"\"referencesProvider\":true,\"renameProvider\":true," +
			"\"completionProvider\":{\"resolveProvider\":false}," +
			"\"documentFormattingProvider\":true}}}"),
	) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/publishDiagnostics\"," +
				"\"params\":{\"uri\":\"file:///sample.tetra\",\"diagnostics\":[]}}"),
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":2,"result":null}`,
		)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing codeActionProvider") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPStdioRejectsMissingDefinitionCapability(t *testing.T) {
	transcript := lspFrame(
		("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"capabilities\":" +
			"{\"textDocumentSync\":1,\"documentSymbolProvider\":true," +
			"\"hoverProvider\":true,\"referencesProvider\":true," +
			"\"renameProvider\":true,\"completionProvider\":" +
			"{\"resolveProvider\":false},\"documentFormattingProvider\":true," +
			"\"codeActionProvider\":true}}}"),
	) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/publishDiagnostics\"," +
				"\"params\":{\"uri\":\"file:///sample.tetra\",\"diagnostics\":[]}}"),
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":2,"result":null}`,
		)
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

func TestParseLSPTranscriptRejectsTooLargeContentLength(t *testing.T) {
	transcript := "Content-Length: 4194305\r\n\r\n{}"
	_, err := parseLSPTranscript([]byte(transcript))
	if err == nil {
		t.Fatal("expected parser failure")
	}
	if !strings.Contains(err.Error(), "Content-Length 4194305 too large") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseLSPTranscriptAcceptsNormalContentLength(t *testing.T) {
	body := `{"jsonrpc":"2.0","method":"exit","params":{}}`
	messages, err := parseLSPTranscript([]byte(lspFrame(body)))
	if err != nil {
		t.Fatalf("parser failed: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(messages))
	}
	if messages[0].Method != "exit" {
		t.Fatalf("got method %q, want exit", messages[0].Method)
	}
}

func TestValidateLSPStdioRejectsMissingReferencesCapability(t *testing.T) {
	transcript := lspFrame(
		("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"capabilities\":" +
			"{\"textDocumentSync\":1,\"documentSymbolProvider\":true," +
			"\"hoverProvider\":true,\"definitionProvider\":true," +
			"\"renameProvider\":true,\"completionProvider\":" +
			"{\"resolveProvider\":false},\"documentFormattingProvider\":true," +
			"\"codeActionProvider\":true}}}"),
	) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/publishDiagnostics\"," +
				"\"params\":{\"uri\":\"file:///sample.tetra\",\"diagnostics\":[]}}"),
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":2,"result":null}`,
		)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing referencesProvider") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPStdioRejectsMissingRenameCapability(t *testing.T) {
	transcript := lspFrame(
		("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"capabilities\":" +
			"{\"textDocumentSync\":1,\"documentSymbolProvider\":true," +
			"\"hoverProvider\":true,\"definitionProvider\":true," +
			"\"referencesProvider\":true,\"completionProvider\":" +
			"{\"resolveProvider\":false},\"documentFormattingProvider\":true," +
			"\"codeActionProvider\":true}}}"),
	) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/publishDiagnostics\"," +
				"\"params\":{\"uri\":\"file:///sample.tetra\",\"diagnostics\":[]}}"),
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":2,"result":null}`,
		)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing renameProvider") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPStdioRejectsDuplicateDocumentSymbolResponse(t *testing.T) {
	transcript := lspFrame(
		("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"capabilities\":" +
			"{\"textDocumentSync\":1,\"documentSymbolProvider\":true," +
			"\"hoverProvider\":true,\"definitionProvider\":true," +
			"\"referencesProvider\":true,\"renameProvider\":true," +
			"\"completionProvider\":{\"resolveProvider\":false}," +
			"\"documentFormattingProvider\":true,\"codeActionProvider\":" +
			"true}}}"),
	) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/publishDiagnostics\"," +
				"\"params\":{\"uri\":\"file:///sample.tetra\",\"diagnostics\":[]}}"),
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":2,"result":[{"name":"answer","kind":14}]}`,
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":2,"result":[{"name":"duplicate","kind":14}]}`,
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":3,"result":{"contents":{"kind":"markdown","value":"const answer: i32"}}}`,
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":4,"result":[{"label":"answer","kind":21}]}`,
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":5,"result":[{"uri":"file:///sample.tetra"}]}`,
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":6,\"result\":[{\"uri\":\"file:" +
				"///sample.tetra\"},{\"uri\":\"file:///sample.tetra\"}]}"),
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":7,"result":{"changes":{"file:///sample.tetra":[]}}}`,
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":8,"result":[{"newText":"formatted"}]}`,
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":9,"result":[{"title":"Add uses io to function main","kind":"quickfix"}]}`,
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":10,"result":null}`,
		)
	out, err := runStdioValidator(t, transcript)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "duplicate documentSymbol response") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPStdioRejectsDuplicateShutdownResponse(t *testing.T) {
	transcript := lspFrame(
		("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"capabilities\":" +
			"{\"textDocumentSync\":1,\"documentSymbolProvider\":true," +
			"\"hoverProvider\":true,\"definitionProvider\":true," +
			"\"referencesProvider\":true,\"renameProvider\":true," +
			"\"completionProvider\":{\"resolveProvider\":false}," +
			"\"documentFormattingProvider\":true,\"codeActionProvider\":" +
			"true}}}"),
	) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/publishDiagnostics\"," +
				"\"params\":{\"uri\":\"file:///sample.tetra\",\"diagnostics\":[]}}"),
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":2,"result":[{"name":"answer","kind":14}]}`,
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":3,"result":{"contents":{"kind":"markdown","value":"const answer: i32"}}}`,
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":4,"result":[{"label":"answer","kind":21}]}`,
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":5,"result":[{"uri":"file:///sample.tetra"}]}`,
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":6,\"result\":[{\"uri\":\"file:" +
				"///sample.tetra\"},{\"uri\":\"file:///sample.tetra\"}]}"),
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":7,"result":{"changes":{"file:///sample.tetra":[]}}}`,
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":8,"result":[{"newText":"formatted"}]}`,
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":9,"result":[{"title":"Add uses io to function main","kind":"quickfix"}]}`,
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":10,"result":null}`,
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":10,"result":null}`,
		)
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

func validRequestResponseTranscript() string {
	return lspFrame(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"capabilities\":" +
				"{\"textDocumentSync\":1,\"documentSymbolProvider\":true," +
				"\"hoverProvider\":true,\"definitionProvider\":true," +
				"\"referencesProvider\":true,\"renameProvider\":true," +
				"\"completionProvider\":{\"resolveProvider\":false}," +
				"\"documentFormattingProvider\":true,\"codeActionProvider\":" +
				"true}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didOpen\",\"params\":" +
				"{\"textDocument\":{\"uri\":\"file:///sample.tetra\",\"languageId\":" +
				"\"tetra\",\"version\":1,\"text\":\"const answer: Int = 42\\n\\nfunc " +
				"main() -> Int:\\n  return answer\\n\"}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/publishDiagnostics\"," +
				"\"params\":{\"uri\":\"file:///sample.tetra\",\"diagnostics\":[]}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":" +
				"\"textDocument/documentSymbol\",\"params\":{\"textDocument\":" +
				"{\"uri\":\"file:///sample.tetra\"}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":2,\"result\":[{\"name\":\"answer\",\"kind\":" +
				"14,\"range\":{\"start\":{\"line\":0,\"character\":6},\"end\":{\"line\":" +
				"0,\"character\":12}},\"selectionRange\":{\"start\":{\"line\":0," +
				"\"character\":6},\"end\":{\"line\":0,\"character\":12}}}]}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"textDocument/hover\"," +
				"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
				"\"position\":{\"line\":0,\"character\":6}}}"),
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":3,"result":{"contents":{"kind":"markdown","value":"const answer: i32"}}}`,
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":4,\"method\":\"textDocument/completion\"," +
				"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
				"\"position\":{\"line\":3,\"character\":9}}}"),
		) +
		lspFrame(
			`{"jsonrpc":"2.0","id":4,"result":[{"label":"answer","kind":21,"detail":"const answer: i32"}]}`,
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":5,\"method\":\"textDocument/definition\"," +
				"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
				"\"position\":{\"line\":3,\"character\":9}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":5,\"result\":[{\"uri\":\"file:" +
				"///sample.tetra\",\"range\":{\"start\":{\"line\":0,\"character\":6}," +
				"\"end\":{\"line\":0,\"character\":12}}}]}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":6,\"method\":\"textDocument/references\"," +
				"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
				"\"position\":{\"line\":3,\"character\":9},\"context\":" +
				"{\"includeDeclaration\":true}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":6,\"result\":[{\"uri\":\"file:" +
				"///sample.tetra\",\"range\":{\"start\":{\"line\":0,\"character\":6}," +
				"\"end\":{\"line\":0,\"character\":12}}},{\"uri\":\"file:" +
				"///sample.tetra\",\"range\":{\"start\":{\"line\":3,\"character\":11}," +
				"\"end\":{\"line\":3,\"character\":17}}}]}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":7,\"method\":\"textDocument/rename\"," +
				"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
				"\"position\":{\"line\":3,\"character\":9},\"newName\":\"value\"}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":7,\"result\":{\"changes\":{\"file:" +
				"///sample.tetra\":[{\"range\":{\"start\":{\"line\":0,\"character\":" +
				"6},\"end\":{\"line\":0,\"character\":12}},\"newText\":\"value\"}]}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":8,\"method\":\"textDocument/formatting\"," +
				"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
				"\"options\":{\"tabSize\":4,\"insertSpaces\":true}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":8,\"result\":[{\"range\":{\"start\":{\"line\":" +
				"0,\"character\":0},\"end\":{\"line\":4,\"character\":0}},\"newText\":" +
				"\"const answer: Int = 42\\n\\nfunc main() -> Int:\\n    return " +
				"answer\\n\"}]}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didChange\",\"params\":" +
				"{\"textDocument\":{\"uri\":\"file:///sample.tetra\",\"version\":2}," +
				"\"contentChanges\":[{\"text\":\"const answer: Int = 42\\n\\nfunc " +
				"main() -> Int:\\n    print(\\\"x\\\")\\n    return answer\\n\"}]}}"),
		) +
		postChangeDiagnosticsFrame() +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":9,\"method\":\"textDocument/codeAction\"," +
				"\"params\":{\"textDocument\":{\"uri\":\"file:///sample.tetra\"}," +
				"\"range\":{\"start\":{\"line\":3,\"character\":4},\"end\":{\"line\":3," +
				"\"character\":9}},\"context\":{\"diagnostics\":[{\"range\":{\"start\":" +
				"{\"line\":3,\"character\":4},\"end\":{\"line\":3,\"character\":9}}," +
				"\"severity\":1,\"code\":\"TETRA2001\",\"source\":\"tetra\",\"message\":" +
				"\"function 'main' uses effect 'io' but does not declare it\"}]" +
				"}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":9,\"result\":[{\"title\":\"Add uses io to " +
				"function main\",\"kind\":\"quickfix\",\"edit\":{\"changes\":{\"file:" +
				"///sample.tetra\":[{\"range\":{\"start\":{\"line\":2,\"character\":" +
				"18},\"end\":{\"line\":2,\"character\":18}},\"newText\":\" uses io\"}]" +
				"}}}]}"),
		) +
		lspFrame(`{"jsonrpc":"2.0","id":10,"method":"shutdown","params":{}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":10,"result":null}`) +
		lspFrame(`{"jsonrpc":"2.0","method":"exit","params":{}}`)
}

func stringIDRequestResponseTranscript() string {
	return lspFrame(`{"jsonrpc":"2.0","id":"init-1","method":"initialize","params":{}}`) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":\"init-1\",\"result\":{\"capabilities\":" +
				"{\"textDocumentSync\":1,\"documentSymbolProvider\":true," +
				"\"hoverProvider\":true,\"definitionProvider\":true," +
				"\"referencesProvider\":true,\"renameProvider\":true," +
				"\"completionProvider\":{\"resolveProvider\":false}," +
				"\"documentFormattingProvider\":true,\"codeActionProvider\":" +
				"true}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didOpen\",\"params\":" +
				"{\"textDocument\":{\"uri\":\"file:///sample.tetra\",\"languageId\":" +
				"\"tetra\",\"version\":1,\"text\":\"const answer: Int = 42\\n\\nfunc " +
				"main() -> Int:\\n  return answer\\n\"}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/publishDiagnostics\"," +
				"\"params\":{\"uri\":\"file:///sample.tetra\",\"diagnostics\":[]}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":\"symbols-1\",\"method\":" +
				"\"textDocument/documentSymbol\",\"params\":{\"textDocument\":" +
				"{\"uri\":\"file:///sample.tetra\"}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":\"symbols-1\",\"result\":[{\"name\":" +
				"\"answer\",\"kind\":14,\"range\":{\"start\":{\"line\":0,\"character\":" +
				"6},\"end\":{\"line\":0,\"character\":12}},\"selectionRange\":" +
				"{\"start\":{\"line\":0,\"character\":6},\"end\":{\"line\":0," +
				"\"character\":12}}}]}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":\"hover-1\",\"method\":" +
				"\"textDocument/hover\",\"params\":{\"textDocument\":{\"uri\":\"file:" +
				"///sample.tetra\"},\"position\":{\"line\":0,\"character\":6}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":\"hover-1\",\"result\":{\"contents\":" +
				"{\"kind\":\"markdown\",\"value\":\"const answer: i32\"}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":\"completion-1\",\"method\":" +
				"\"textDocument/completion\",\"params\":{\"textDocument\":{\"uri\":" +
				"\"file:///sample.tetra\"},\"position\":{\"line\":3,\"character\":" +
				"9}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":\"completion-1\",\"result\":[{\"label\":" +
				"\"answer\",\"kind\":21,\"detail\":\"const answer: i32\"}]}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":\"definition-1\",\"method\":" +
				"\"textDocument/definition\",\"params\":{\"textDocument\":{\"uri\":" +
				"\"file:///sample.tetra\"},\"position\":{\"line\":3,\"character\":" +
				"9}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":\"definition-1\",\"result\":[{\"uri\":\"file:" +
				"///sample.tetra\",\"range\":{\"start\":{\"line\":0,\"character\":6}," +
				"\"end\":{\"line\":0,\"character\":12}}}]}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":\"references-1\",\"method\":" +
				"\"textDocument/references\",\"params\":{\"textDocument\":{\"uri\":" +
				"\"file:///sample.tetra\"},\"position\":{\"line\":3,\"character\":9}," +
				"\"context\":{\"includeDeclaration\":true}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":\"references-1\",\"result\":[{\"uri\":\"file:" +
				"///sample.tetra\",\"range\":{\"start\":{\"line\":0,\"character\":6}," +
				"\"end\":{\"line\":0,\"character\":12}}},{\"uri\":\"file:" +
				"///sample.tetra\",\"range\":{\"start\":{\"line\":3,\"character\":11}," +
				"\"end\":{\"line\":3,\"character\":17}}}]}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":\"rename-1\",\"method\":" +
				"\"textDocument/rename\",\"params\":{\"textDocument\":{\"uri\":\"file:" +
				"///sample.tetra\"},\"position\":{\"line\":3,\"character\":9}," +
				"\"newName\":\"value\"}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":\"rename-1\",\"result\":{\"changes\":{\"file:" +
				"///sample.tetra\":[{\"range\":{\"start\":{\"line\":0,\"character\":" +
				"6},\"end\":{\"line\":0,\"character\":12}},\"newText\":\"value\"}]}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":\"formatting-1\",\"method\":" +
				"\"textDocument/formatting\",\"params\":{\"textDocument\":{\"uri\":" +
				"\"file:///sample.tetra\"},\"options\":{\"tabSize\":4," +
				"\"insertSpaces\":true}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":\"formatting-1\",\"result\":[{\"range\":" +
				"{\"start\":{\"line\":0,\"character\":0},\"end\":{\"line\":4," +
				"\"character\":0}},\"newText\":\"const answer: Int = 42\\n\\nfunc " +
				"main() -> Int:\\n    return answer\\n\"}]}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/didChange\",\"params\":" +
				"{\"textDocument\":{\"uri\":\"file:///sample.tetra\",\"version\":2}," +
				"\"contentChanges\":[{\"text\":\"const answer: Int = 42\\n\\nfunc " +
				"main() -> Int:\\n    print(\\\"x\\\")\\n    return answer\\n\"}]}}"),
		) +
		postChangeDiagnosticsFrame() +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":\"code-action-1\",\"method\":" +
				"\"textDocument/codeAction\",\"params\":{\"textDocument\":{\"uri\":" +
				"\"file:///sample.tetra\"},\"range\":{\"start\":{\"line\":3," +
				"\"character\":4},\"end\":{\"line\":3,\"character\":9}},\"context\":" +
				"{\"diagnostics\":[{\"range\":{\"start\":{\"line\":3,\"character\":4}," +
				"\"end\":{\"line\":3,\"character\":9}},\"severity\":1,\"code\":" +
				"\"TETRA2001\",\"source\":\"tetra\",\"message\":\"function 'main' " +
				"uses effect 'io' but does not declare it\"}]}}}"),
		) +
		lspFrame(
			("{\"jsonrpc\":\"2.0\",\"id\":\"code-action-1\",\"result\":[{\"title\":" +
				"\"Add uses io to function main\",\"kind\":\"quickfix\",\"edit\":" +
				"{\"changes\":{\"file:///sample.tetra\":[{\"range\":{\"start\":" +
				"{\"line\":2,\"character\":18},\"end\":{\"line\":2,\"character\":18}}," +
				"\"newText\":\" uses io\"}]}}}]}"),
		) +
		lspFrame(`{"jsonrpc":"2.0","id":"shutdown-1","method":"shutdown","params":{}}`) +
		lspFrame(`{"jsonrpc":"2.0","id":"shutdown-1","result":null}`) +
		lspFrame(`{"jsonrpc":"2.0","method":"exit","params":{}}`)
}

func numericLookingStringIDRequestResponseTranscript() string {
	replacer := strings.NewReplacer(
		`"init-1"`, `"1"`,
		`"symbols-1"`, `"3"`,
		`"hover-1"`, `"2"`,
		`"completion-1"`, `"4"`,
		`"definition-1"`, `"5"`,
		`"references-1"`, `"6"`,
		`"rename-1"`, `"7"`,
		`"formatting-1"`, `"8"`,
		`"code-action-1"`, `"9"`,
		`"shutdown-1"`, `"10"`,
	)
	return replaceLSPFrameBodies(stringIDRequestResponseTranscript(), replacer)
}

func postChangeDiagnosticsFrame() string {
	return lspFrame(
		("{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/publishDiagnostics\"," +
			"\"params\":{\"uri\":\"file:///sample.tetra\",\"diagnostics\":" +
			"[{\"range\":{\"start\":{\"line\":3,\"character\":4},\"end\":{\"line\":3," +
			"\"character\":9}},\"severity\":1,\"code\":\"TETRA2001\",\"source\":" +
			"\"tetra\",\"message\":\"function 'main' uses effect 'io' but " +
			"does not declare it\"}]}}"),
	)
}

func replaceLSPFrameBodies(transcript string, replacer *strings.Replacer) string {
	var out strings.Builder
	for transcript != "" {
		const prefix = "Content-Length: "
		if !strings.HasPrefix(transcript, prefix) {
			panic("test transcript missing Content-Length header")
		}
		headerEnd := strings.Index(transcript, "\r\n\r\n")
		if headerEnd < 0 {
			panic("test transcript missing frame separator")
		}
		lengthText := strings.TrimSpace(strings.TrimPrefix(transcript[:headerEnd], prefix))
		length, err := strconv.Atoi(lengthText)
		if err != nil {
			panic("test transcript has invalid Content-Length")
		}
		bodyStart := headerEnd + len("\r\n\r\n")
		bodyEnd := bodyStart + length
		if bodyEnd > len(transcript) {
			panic("test transcript body is truncated")
		}
		out.WriteString(lspFrame(replacer.Replace(transcript[bodyStart:bodyEnd])))
		transcript = transcript[bodyEnd:]
	}
	return out.String()
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
