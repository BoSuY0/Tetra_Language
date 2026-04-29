package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type lspMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

func main() {
	var transcriptPath string
	flag.StringVar(&transcriptPath, "transcript", "", "path to captured tetra lsp --stdio output")
	flag.Parse()

	if transcriptPath == "" {
		fmt.Fprintln(os.Stderr, "error: --transcript is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(transcriptPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateLSPTranscript(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateLSPTranscript(raw []byte) error {
	messages, err := parseLSPTranscript(raw)
	if err != nil {
		return err
	}
	if len(messages) == 0 {
		return fmt.Errorf("empty LSP transcript")
	}
	sawInitialize := false
	sawDiagnostics := false
	sawShutdown := false
	editorResponses := map[int]json.RawMessage{}
	for _, msg := range messages {
		if msg.JSONRPC != "2.0" {
			return fmt.Errorf("message missing jsonrpc 2.0")
		}
		if len(msg.Error) > 0 && !bytes.Equal(bytes.TrimSpace(msg.Error), []byte("null")) {
			return fmt.Errorf("LSP error response present: %s", string(msg.Error))
		}
		if msg.ID != nil && *msg.ID == 1 {
			capabilities, ok := jsonObjectField(msg.Result, "capabilities")
			if !ok {
				return fmt.Errorf("initialize response missing capabilities")
			}
			if !jsonObjectHasKey(capabilities, "documentSymbolProvider") {
				return fmt.Errorf("initialize capabilities missing documentSymbolProvider")
			}
			if !jsonObjectHasKey(capabilities, "hoverProvider") {
				return fmt.Errorf("initialize capabilities missing hoverProvider")
			}
			if !jsonObjectHasKey(capabilities, "definitionProvider") {
				return fmt.Errorf("initialize capabilities missing definitionProvider")
			}
			if !jsonObjectHasKey(capabilities, "referencesProvider") {
				return fmt.Errorf("initialize capabilities missing referencesProvider")
			}
			if !jsonObjectHasKey(capabilities, "renameProvider") {
				return fmt.Errorf("initialize capabilities missing renameProvider")
			}
			if !jsonObjectHasKey(capabilities, "completionProvider") {
				return fmt.Errorf("initialize capabilities missing completionProvider")
			}
			if !jsonObjectHasKey(capabilities, "documentFormattingProvider") {
				return fmt.Errorf("initialize capabilities missing documentFormattingProvider")
			}
			if !jsonObjectHasKey(capabilities, "codeActionProvider") {
				return fmt.Errorf("initialize capabilities missing codeActionProvider")
			}
			sawInitialize = true
		}
		if msg.ID != nil {
			if *msg.ID >= 2 && *msg.ID <= 9 {
				editorResponses[*msg.ID] = msg.Result
			}
		}
		if msg.Method == "textDocument/publishDiagnostics" {
			if !jsonObjectHasKey(msg.Params, "diagnostics") {
				return fmt.Errorf("publishDiagnostics missing diagnostics")
			}
			if !jsonObjectHasKey(msg.Params, "uri") {
				return fmt.Errorf("publishDiagnostics missing uri")
			}
			sawDiagnostics = true
		}
		if msg.ID != nil && *msg.ID == 10 {
			sawShutdown = true
		}
	}
	if !sawInitialize {
		return fmt.Errorf("missing initialize response")
	}
	if !sawDiagnostics {
		return fmt.Errorf("missing textDocument/publishDiagnostics notification")
	}
	for _, expected := range []struct {
		id   int
		name string
	}{
		{2, "documentSymbol"},
		{3, "hover"},
		{4, "completion"},
		{5, "definition"},
		{6, "references"},
		{7, "rename"},
		{8, "formatting"},
		{9, "codeAction"},
	} {
		raw, ok := editorResponses[expected.id]
		if !ok {
			return fmt.Errorf("missing %s response", expected.name)
		}
		if err := validateEditorResponse(expected.id, raw); err != nil {
			return err
		}
	}
	if !sawShutdown {
		return fmt.Errorf("missing shutdown response")
	}
	return nil
}

func validateEditorResponse(id int, raw json.RawMessage) error {
	switch id {
	case 2:
		if !jsonArrayHasObjectField(raw, "name") {
			return fmt.Errorf("documentSymbol response missing symbol name")
		}
	case 3:
		contents, ok := jsonObjectField(raw, "contents")
		if !ok || !jsonObjectHasKey(contents, "value") {
			return fmt.Errorf("hover response missing markdown contents")
		}
	case 4:
		if !jsonArrayHasObjectField(raw, "label") {
			return fmt.Errorf("completion response missing item label")
		}
	case 5:
		if !jsonArrayHasObjectField(raw, "uri") {
			return fmt.Errorf("definition response missing location uri")
		}
	case 6:
		if jsonArrayLength(raw) < 2 {
			return fmt.Errorf("references response must include declaration and usage locations")
		}
	case 7:
		changes, ok := jsonObjectField(raw, "changes")
		if !ok || !jsonObjectHasAnyKey(changes) {
			return fmt.Errorf("rename response missing workspace edit changes")
		}
	case 8:
		if !jsonArrayHasObjectField(raw, "newText") {
			return fmt.Errorf("formatting response missing text edit")
		}
	case 9:
		if !jsonArrayHasObjectField(raw, "title") {
			return fmt.Errorf("codeAction response missing action title")
		}
	}
	return nil
}

func parseLSPTranscript(raw []byte) ([]lspMessage, error) {
	reader := bufio.NewReader(bytes.NewReader(raw))
	var messages []lspMessage
	for {
		length, err := readContentLength(reader)
		if err == io.EOF {
			return messages, nil
		}
		if err != nil {
			return nil, err
		}
		body := make([]byte, length)
		if _, err := io.ReadFull(reader, body); err != nil {
			if err == io.ErrUnexpectedEOF || err == io.EOF {
				return nil, fmt.Errorf("message body truncated")
			}
			return nil, err
		}
		var msg lspMessage
		if err := json.Unmarshal(body, &msg); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
}

func readContentLength(reader *bufio.Reader) (int, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return 0, err
	}
	line = strings.TrimRight(line, "\r\n")
	const prefix = "Content-Length:"
	if !strings.HasPrefix(line, prefix) {
		return 0, fmt.Errorf("expected Content-Length header, got %q", line)
	}
	lengthText := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	length, err := strconv.Atoi(lengthText)
	if err != nil || length < 0 {
		return 0, fmt.Errorf("invalid Content-Length %q", lengthText)
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return 0, err
		}
		if line == "\r\n" || line == "\n" {
			return length, nil
		}
	}
}

func jsonObjectHasKey(raw json.RawMessage, key string) bool {
	_, ok := jsonObjectField(raw, key)
	return ok
}

func jsonObjectHasAnyKey(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return false
	}
	return len(obj) > 0
}

func jsonObjectField(raw json.RawMessage, key string) (json.RawMessage, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, false
	}
	value, ok := obj[key]
	return value, ok
}

func jsonArrayHasObjectField(raw json.RawMessage, key string) bool {
	var values []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return false
	}
	for _, value := range values {
		if _, ok := value[key]; ok {
			return true
		}
	}
	return false
}

func jsonArrayLength(raw json.RawMessage) int {
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return 0
	}
	return len(values)
}
