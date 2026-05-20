package main

import (
	"encoding/json"
	"fmt"
	"regexp"

	"tetra_language/compiler"
)

type lspRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *lspID          `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type lspID struct {
	value any
}

func (id *lspID) UnmarshalJSON(raw []byte) error {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		id.value = s
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(raw, &n); err == nil {
		i, err := n.Int64()
		if err != nil {
			return fmt.Errorf("invalid JSON-RPC id: %w", err)
		}
		id.value = i
		return nil
	}
	return fmt.Errorf("invalid JSON-RPC id")
}

func (id lspID) JSONValue() any {
	return id.value
}

type lspTextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type lspDidOpenParams struct {
	TextDocument struct {
		URI  string `json:"uri"`
		Text string `json:"text"`
	} `json:"textDocument"`
}

type lspDidChangeParams struct {
	TextDocument   lspTextDocumentIdentifier `json:"textDocument"`
	ContentChanges []struct {
		Text string `json:"text"`
	} `json:"contentChanges"`
}

type lspTextDocumentParams struct {
	TextDocument lspTextDocumentIdentifier `json:"textDocument"`
}

type lspDidCloseParams struct {
	TextDocument lspTextDocumentIdentifier `json:"textDocument"`
}

type lspHoverParams struct {
	TextDocument lspTextDocumentIdentifier `json:"textDocument"`
	Position     struct {
		Line      int `json:"line"`
		Character int `json:"character"`
	} `json:"position"`
}

type lspDefinitionParams struct {
	TextDocument lspTextDocumentIdentifier `json:"textDocument"`
	Position     struct {
		Line      int `json:"line"`
		Character int `json:"character"`
	} `json:"position"`
}

type lspReferencesParams struct {
	TextDocument lspTextDocumentIdentifier `json:"textDocument"`
	Position     struct {
		Line      int `json:"line"`
		Character int `json:"character"`
	} `json:"position"`
	Context struct {
		IncludeDeclaration bool `json:"includeDeclaration"`
	} `json:"context"`
}

type lspRenameParams struct {
	TextDocument lspTextDocumentIdentifier `json:"textDocument"`
	Position     struct {
		Line      int `json:"line"`
		Character int `json:"character"`
	} `json:"position"`
	NewName string `json:"newName"`
}

type lspCodeActionParams struct {
	TextDocument lspTextDocumentIdentifier `json:"textDocument"`
	Context      struct {
		Diagnostics []lspCodeActionDiagnostic `json:"diagnostics"`
	} `json:"context"`
}

type lspCodeActionDiagnostic struct {
	Code    json.RawMessage `json:"code,omitempty"`
	Message string          `json:"message"`
}

type lspOpenDocument struct {
	Text     string
	Analysis compiler.LSPAnalysis
}

type lspReference struct {
	Line   int
	Column int
}

var lspMissingEffectDiagnosticRE = regexp.MustCompile(`^function '([^']+)' uses effect '([^']+)' but does not declare it$`)
