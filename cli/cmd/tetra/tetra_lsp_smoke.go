package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"tetra_language/compiler"
	ctarget "tetra_language/compiler/target"
	"tetra_language/internal/outputformat"
	"time"
)

// ---- lsp.go ----

func runLSP(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("lsp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	smokePath := fs.String(
		"stdio-smoke",
		"",
		"analyze one .t4/.tetra file and print LSP-basic JSON",
	)
	stdio := fs.Bool("stdio", false, "run LSP-basic JSON-RPC over stdio")
	format := fs.String("format", outputformat.JSON, "stdio-smoke report format: json or toon")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "lsp does not accept positional arguments")
		return 2
	}
	if *stdio {
		if outputformat.Normalize(*format) != outputformat.JSON {
			fmt.Fprintln(
				stderr,
				"lsp --stdio only supports --format=json because stdio uses framed JSON-RPC",
			)
			return 2
		}
		return runLSPStdio(os.Stdin, stdout, stderr)
	}
	if *smokePath == "" {
		fmt.Fprintln(stderr, "lsp requires --stdio or --stdio-smoke <file>")
		return 2
	}
	if !outputformat.Structured(*format) {
		fmt.Fprintf(stderr, "unsupported lsp smoke report format %q\n", *format)
		return 2
	}
	analysis, err := compiler.AnalyzeLSPFile(*smokePath)
	if err != nil {
		writeDiagnostic(stderr, "json", err)
		return 1
	}
	if outputformat.Normalize(*format) == outputformat.TOON {
		if err := outputformat.WriteStructured(stdout, outputformat.TOON, analysis); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	enc := json.NewEncoder(stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(analysis); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

const (
	lspErrorParseError     = -32700
	lspErrorInvalidRequest = -32600
	lspErrorMethodNotFound = -32601
	lspErrorInvalidParams  = -32602
)

func runLSPStdio(stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	reader := bufio.NewReader(stdin)
	openDocs := map[string]lspOpenDocument{}
	shutdown := false
	for {
		body, err := readLSPMessage(reader)
		if err == io.EOF {
			return 0
		}
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		var req lspRequest
		if err := json.Unmarshal(body, &req); err != nil {
			code := lspErrorInvalidRequest
			if !json.Valid(body) {
				code = lspErrorParseError
			}
			if err := writeLSPErrorResponse(
				stdout,
				lspIDFromRawMessage(body),
				code,
				"invalid request: "+err.Error(),
			); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			continue
		}
		if req.JSONRPC != "2.0" {
			if err := writeLSPRequestError(
				stdout,
				req,
				lspErrorInvalidRequest,
				`invalid request: jsonrpc must be "2.0"`,
			); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			continue
		}
		if req.Method == "" {
			if err := writeLSPRequestError(
				stdout,
				req,
				lspErrorInvalidRequest,
				"invalid request: method is required",
			); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			continue
		}
		switch req.Method {
		case "initialize":
			if req.ID != nil {
				result := map[string]any{
					"capabilities": map[string]any{
						"textDocumentSync":           1,
						"documentSymbolProvider":     true,
						"hoverProvider":              true,
						"definitionProvider":         true,
						"referencesProvider":         true,
						"renameProvider":             true,
						"documentFormattingProvider": true,
						"codeActionProvider":         true,
						"completionProvider": map[string]any{
							"resolveProvider": false,
						},
					},
				}
				if err := writeLSPResponse(stdout, req.ID.JSONValue(), result); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		case "shutdown":
			shutdown = true
			if req.ID != nil {
				if err := writeLSPResponse(stdout, req.ID.JSONValue(), nil); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		case "exit":
			if shutdown {
				return 0
			}
			return 1
		case "textDocument/didOpen":
			var params lspDidOpenParams
			if err := lspDecodeRequestParams(req, &params); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if err := lspValidateTextDocumentURI(params.TextDocument.URI); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			analysis := compiler.AnalyzeLSPSource(
				[]byte(params.TextDocument.Text),
				params.TextDocument.URI,
			)
			openDocs[params.TextDocument.URI] = lspOpenDocument{
				Text:     params.TextDocument.Text,
				Analysis: analysis,
			}
			if err := writeLSPNotification(stdout, "textDocument/publishDiagnostics", map[string]any{
				"uri":         params.TextDocument.URI,
				"diagnostics": lspDiagnostics(analysis.Diagnostics),
			}); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
		case "textDocument/didChange":
			var params lspDidChangeParams
			if err := lspDecodeRequestParams(req, &params); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if err := lspValidateTextDocumentURI(params.TextDocument.URI); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if len(params.ContentChanges) == 0 {
				continue
			}
			text := params.ContentChanges[len(params.ContentChanges)-1].Text
			analysis := compiler.AnalyzeLSPSource([]byte(text), params.TextDocument.URI)
			openDocs[params.TextDocument.URI] = lspOpenDocument{Text: text, Analysis: analysis}
			if err := writeLSPNotification(stdout, "textDocument/publishDiagnostics", map[string]any{
				"uri":         params.TextDocument.URI,
				"diagnostics": lspDiagnostics(analysis.Diagnostics),
			}); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
		case "textDocument/didClose":
			var params lspDidCloseParams
			if err := lspDecodeRequestParams(req, &params); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if err := lspValidateTextDocumentURI(params.TextDocument.URI); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			delete(openDocs, params.TextDocument.URI)
			if err := writeLSPNotification(stdout, "textDocument/publishDiagnostics", map[string]any{
				"uri":         params.TextDocument.URI,
				"diagnostics": []any{},
			}); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
		case "textDocument/documentSymbol":
			var params lspTextDocumentParams
			if err := lspDecodeRequestParams(req, &params); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if err := lspValidateTextDocumentURI(params.TextDocument.URI); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if req.ID != nil {
				result := []map[string]any{}
				if doc, ok := openDocs[params.TextDocument.URI]; ok {
					result = lspDocumentSymbols(doc.Analysis)
				}
				if err := writeLSPResponse(stdout, req.ID.JSONValue(), result); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		case "textDocument/hover":
			var params lspHoverParams
			if err := lspDecodeRequestParams(req, &params); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if err := lspValidateTextDocumentPosition(
				params.TextDocument.URI,
				params.Position.Line,
				params.Position.Character,
			); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if req.ID != nil {
				var result any
				if doc, ok := openDocs[params.TextDocument.URI]; ok {
					result = lspHoverAt(doc.Analysis, params.Position.Line)
				}
				if err := writeLSPResponse(stdout, req.ID.JSONValue(), result); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		case "textDocument/definition":
			var params lspDefinitionParams
			if err := lspDecodeRequestParams(req, &params); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if err := lspValidateTextDocumentPosition(
				params.TextDocument.URI,
				params.Position.Line,
				params.Position.Character,
			); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if req.ID != nil {
				var result any
				if doc, ok := openDocs[params.TextDocument.URI]; ok {
					result = lspDefinitionLocations(
						doc,
						params.TextDocument.URI,
						params.Position.Line,
						params.Position.Character,
					)
				}
				if err := writeLSPResponse(stdout, req.ID.JSONValue(), result); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		case "textDocument/references":
			var params lspReferencesParams
			if err := lspDecodeRequestParams(req, &params); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if err := lspValidateTextDocumentPosition(
				params.TextDocument.URI,
				params.Position.Line,
				params.Position.Character,
			); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if req.ID != nil {
				var result any = []map[string]any{}
				if doc, ok := openDocs[params.TextDocument.URI]; ok {
					result = lspReferenceLocations(
						doc,
						params.TextDocument.URI,
						params.Position.Line,
						params.Position.Character,
						params.Context.IncludeDeclaration,
					)
				}
				if err := writeLSPResponse(stdout, req.ID.JSONValue(), result); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		case "textDocument/rename":
			var params lspRenameParams
			if err := lspDecodeRequestParams(req, &params); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if err := lspValidateTextDocumentPosition(
				params.TextDocument.URI,
				params.Position.Line,
				params.Position.Character,
			); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if err := lspValidateRenameNewName(params.NewName); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if req.ID != nil {
				var result any
				if doc, ok := openDocs[params.TextDocument.URI]; ok {
					result = lspRenameWorkspaceEdit(
						doc,
						params.TextDocument.URI,
						params.Position.Line,
						params.Position.Character,
						params.NewName,
					)
				}
				if err := writeLSPResponse(stdout, req.ID.JSONValue(), result); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		case "textDocument/completion":
			var params lspHoverParams
			if err := lspDecodeRequestParams(req, &params); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if err := lspValidateTextDocumentPosition(
				params.TextDocument.URI,
				params.Position.Line,
				params.Position.Character,
			); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if req.ID != nil {
				result := []map[string]any{}
				if doc, ok := openDocs[params.TextDocument.URI]; ok {
					result = lspCompletionItems(doc.Analysis)
				}
				if err := writeLSPResponse(stdout, req.ID.JSONValue(), result); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		case "textDocument/formatting":
			var params lspTextDocumentParams
			if err := lspDecodeRequestParams(req, &params); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if err := lspValidateTextDocumentURI(params.TextDocument.URI); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if req.ID != nil {
				edits := []map[string]any{}
				if doc, ok := openDocs[params.TextDocument.URI]; ok {
					var err error
					edits, err = lspFormattingEdits(doc.Text, params.TextDocument.URI)
					if err != nil {
						edits = []map[string]any{}
					}
				}
				if err := writeLSPResponse(stdout, req.ID.JSONValue(), edits); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		case "textDocument/codeAction":
			var params lspCodeActionParams
			if err := lspDecodeRequestParams(req, &params); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if err := lspValidateTextDocumentURI(params.TextDocument.URI); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if req.ID != nil {
				actions := []map[string]any{}
				if doc, ok := openDocs[params.TextDocument.URI]; ok {
					actions = lspCodeActions(
						doc.Text,
						params.TextDocument.URI,
						params.Context.Diagnostics,
						doc.Analysis.Diagnostics,
					)
				}
				if err := writeLSPResponse(stdout, req.ID.JSONValue(), actions); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		default:
			if req.ID != nil {
				if err := writeLSPErrorResponse(
					stdout,
					req.ID.JSONValue(),
					lspErrorMethodNotFound,
					"unknown method: "+req.Method,
				); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		}
	}
}

func lspDecodeRequestParams(req lspRequest, params any) error {
	if len(req.Params) == 0 {
		return fmt.Errorf("invalid params: params object is required")
	}
	if err := json.Unmarshal(req.Params, params); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	return nil
}

func lspValidateTextDocumentURI(uri string) error {
	if strings.TrimSpace(uri) == "" {
		return fmt.Errorf("invalid params: textDocument.uri is required")
	}
	return nil
}

func lspValidateTextDocumentPosition(uri string, line int, character int) error {
	if err := lspValidateTextDocumentURI(uri); err != nil {
		return err
	}
	if line < 0 || character < 0 {
		return fmt.Errorf("invalid params: position line and character must be non-negative")
	}
	return nil
}

func lspValidateRenameNewName(name string) error {
	if strings.TrimSpace(name) != name || name == "" {
		return fmt.Errorf("invalid params: rename newName must be a Tetra identifier")
	}
	if !isLSPIdentifierStart(name[0]) {
		return fmt.Errorf("invalid params: rename newName must be a Tetra identifier")
	}
	for i := 1; i < len(name); i++ {
		if !isLSPIdentifierChar(name[i]) {
			return fmt.Errorf("invalid params: rename newName must be a Tetra identifier")
		}
	}
	return nil
}

func lspIDFromRawMessage(body []byte) any {
	var raw struct {
		ID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(body, &raw); err != nil || len(raw.ID) == 0 {
		return nil
	}
	var id lspID
	if err := json.Unmarshal(raw.ID, &id); err == nil {
		return id.JSONValue()
	}
	return nil
}

func writeLSPRequestError(w io.Writer, req lspRequest, code int, message string) error {
	if req.ID == nil {
		return nil
	}
	return writeLSPErrorResponse(w, req.ID.JSONValue(), code, message)
}

func writeLSPErrorResponse(w io.Writer, id any, code int, message string) error {
	return writeLSPMessage(w, map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func lspDiagnostics(diags []compiler.Diagnostic) []map[string]any {
	out := make([]map[string]any, 0, len(diags))
	for _, diag := range diags {
		line := diag.Line
		if line > 0 {
			line--
		}
		col := diag.Column
		if col > 0 {
			col--
		}
		out = append(out, map[string]any{
			"range": map[string]any{
				"start": map[string]int{"line": line, "character": col},
				"end":   map[string]int{"line": line, "character": col + 1},
			},
			"severity": 1,
			"code":     diag.Code,
			"source":   "tetra",
			"message":  diag.Message,
		})
	}
	return out
}

func lspDocumentSymbols(analysis compiler.LSPAnalysis) []map[string]any {
	out := make([]map[string]any, 0, len(analysis.Symbols))
	for _, sym := range analysis.Symbols {
		line := maxInt(sym.Line-1, 0)
		col := maxInt(sym.Column-1, 0)
		item := map[string]any{
			"name": sym.Name,
			"kind": lspSymbolKind(sym.Kind),
			"range": map[string]any{
				"start": map[string]int{"line": line, "character": col},
				"end":   map[string]int{"line": line, "character": col + 1},
			},
			"selectionRange": map[string]any{
				"start": map[string]int{"line": line, "character": col},
				"end":   map[string]int{"line": line, "character": col + len(sym.Name)},
			},
		}
		if sym.Detail != "" {
			item["detail"] = sym.Detail
		}
		out = append(out, item)
	}
	return out
}

func lspHoverAt(analysis compiler.LSPAnalysis, zeroBasedLine int) any {
	line := zeroBasedLine + 1
	for _, hover := range analysis.Hovers {
		if hover.Line == line {
			return map[string]any{
				"contents": map[string]string{"kind": "markdown", "value": hover.Contents},
			}
		}
	}
	return nil
}

func lspDefinitionLocations(
	doc lspOpenDocument,
	uri string,
	zeroBasedLine int,
	zeroBasedCharacter int,
) any {
	name := lspIdentifierAt(doc.Text, zeroBasedLine, zeroBasedCharacter)
	if name == "" {
		return nil
	}
	line, col, ok := lspDefinitionPosition(doc, name)
	if !ok {
		return nil
	}
	return []map[string]any{{
		"uri": uri,
		"range": map[string]any{
			"start": map[string]int{"line": line, "character": col},
			"end":   map[string]int{"line": line, "character": col + len(name)},
		},
	}}
}

func lspIdentifierAt(text string, zeroBasedLine int, zeroBasedCharacter int) string {
	if zeroBasedLine < 0 || zeroBasedCharacter < 0 {
		return ""
	}
	lines := strings.Split(text, "\n")
	if zeroBasedLine >= len(lines) {
		return ""
	}
	line := lines[zeroBasedLine]
	if len(line) == 0 {
		return ""
	}
	idx := zeroBasedCharacter
	if idx >= len(line) {
		idx = len(line) - 1
	}
	if !isLSPIdentifierChar(line[idx]) {
		if idx > 0 && isLSPIdentifierChar(line[idx-1]) {
			idx--
		} else {
			return ""
		}
	}
	start := idx
	for start > 0 && isLSPIdentifierChar(line[start-1]) {
		start--
	}
	end := idx + 1
	for end < len(line) && isLSPIdentifierChar(line[end]) {
		end++
	}
	if !lspTextualRangeIsCode(lspTextualCodeMask(line), start, end) {
		return ""
	}
	return line[start:end]
}

func isLSPIdentifierChar(ch byte) bool {
	return isLSPIdentifierStart(ch) || ch >= '0' && ch <= '9'
}

func isLSPIdentifierStart(ch byte) bool {
	return ch == '_' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z'
}

func lspDefinitionColumn(text string, sym compiler.LSPSymbol) int {
	line := maxInt(sym.Line-1, 0)
	col := maxInt(sym.Column-1, 0)
	lines := strings.Split(text, "\n")
	if line < 0 || line >= len(lines) {
		return col
	}
	lineText := lines[line]
	if col >= 0 && col+len(sym.Name) <= len(lineText) &&
		lineText[col:col+len(sym.Name)] == sym.Name {
		return col
	}
	if idx := strings.Index(lineText, sym.Name); idx >= 0 {
		return idx
	}
	return col
}

func lspDefinitionPosition(doc lspOpenDocument, name string) (int, int, bool) {
	for _, sym := range doc.Analysis.Symbols {
		if sym.Name != name {
			continue
		}
		line := maxInt(sym.Line-1, 0)
		col := lspDefinitionColumn(doc.Text, sym)
		return line, col, true
	}
	return 0, 0, false
}

func lspReferenceLocations(
	doc lspOpenDocument,
	uri string,
	zeroBasedLine int,
	zeroBasedCharacter int,
	includeDeclaration bool,
) any {
	name := lspIdentifierAt(doc.Text, zeroBasedLine, zeroBasedCharacter)
	if name == "" {
		return nil
	}
	defLine, defCol, hasDefinition := lspDefinitionPosition(doc, name)
	locations := []map[string]any{}
	for _, ref := range lspIdentifierReferences(doc.Text, name) {
		if !includeDeclaration && hasDefinition && ref.Line == defLine && ref.Column == defCol {
			continue
		}
		locations = append(locations, map[string]any{
			"uri": uri,
			"range": map[string]any{
				"start": map[string]int{"line": ref.Line, "character": ref.Column},
				"end":   map[string]int{"line": ref.Line, "character": ref.Column + len(name)},
			},
		})
	}
	return locations
}

func lspIdentifierReferences(text string, name string) []lspReference {
	if name == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	refs := []lspReference{}
	for line, content := range lines {
		codeMask := lspTextualCodeMask(content)
		searchFrom := 0
		for searchFrom <= len(content)-len(name) {
			offset := strings.Index(content[searchFrom:], name)
			if offset < 0 {
				break
			}
			col := searchFrom + offset
			startOk := col == 0 || !isLSPIdentifierChar(content[col-1])
			end := col + len(name)
			endOk := end == len(content) || !isLSPIdentifierChar(content[end])
			searchFrom = end
			if !startOk || !endOk {
				continue
			}
			if !lspTextualRangeIsCode(codeMask, col, end) {
				continue
			}
			refs = append(refs, lspReference{Line: line, Column: col})
		}
	}
	return refs
}

func lspTextualCodeMask(line string) []bool {
	mask := make([]bool, len(line))
	for i := range mask {
		mask[i] = true
	}
	inString := false
	escaped := false
	for i := 0; i < len(line); i++ {
		if inString {
			mask[i] = false
			if escaped {
				escaped = false
				continue
			}
			switch line[i] {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}
		if line[i] == '"' {
			mask[i] = false
			inString = true
			continue
		}
		if line[i] == '#' || line[i] == '/' && i+1 < len(line) && line[i+1] == '/' {
			for j := i; j < len(line); j++ {
				mask[j] = false
			}
			break
		}
	}
	return mask
}

func lspTextualRangeIsCode(mask []bool, start int, end int) bool {
	if start < 0 || end > len(mask) || start >= end {
		return false
	}
	for i := start; i < end; i++ {
		if !mask[i] {
			return false
		}
	}
	return true
}

func lspRenameWorkspaceEdit(
	doc lspOpenDocument,
	uri string,
	zeroBasedLine int,
	zeroBasedCharacter int,
	newName string,
) any {
	if strings.TrimSpace(newName) == "" {
		return nil
	}
	oldName := lspIdentifierAt(doc.Text, zeroBasedLine, zeroBasedCharacter)
	if oldName == "" {
		return nil
	}
	if !lspHasTopLevelSymbol(doc, oldName) || lspHasLocalBindingConflict(doc.Text, oldName) {
		return nil
	}
	refs := lspIdentifierReferences(doc.Text, oldName)
	if len(refs) == 0 {
		return nil
	}
	edits := make([]map[string]any, 0, len(refs))
	for _, ref := range refs {
		edits = append(edits, map[string]any{
			"range": map[string]any{
				"start": map[string]int{"line": ref.Line, "character": ref.Column},
				"end":   map[string]int{"line": ref.Line, "character": ref.Column + len(oldName)},
			},
			"newText": newName,
		})
	}
	return map[string]any{
		"changes": map[string]any{
			uri: edits,
		},
	}
}

func lspHasTopLevelSymbol(doc lspOpenDocument, name string) bool {
	for _, sym := range doc.Analysis.Symbols {
		if sym.Name == name {
			return true
		}
	}
	return false
}

func lspHasLocalBindingConflict(text string, name string) bool {
	for _, line := range strings.Split(text, "\n") {
		if lspCodeLineDeclaresLocalName(line, name) {
			return true
		}
	}
	return false
}

func lspCodeLineDeclaresLocalName(line string, name string) bool {
	code := lspCodeOnly(line)
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return false
	}
	leadingLocal := len(code) > 0 && (code[0] == ' ' || code[0] == '\t')
	if leadingLocal {
		for _, keyword := range []string{"let", "var", "const", "for"} {
			if lspDeclarationStartsWithName(trimmed, keyword, name) {
				return true
			}
		}
	}
	return lspFunctionParamsDeclareName(trimmed, name)
}

func lspCodeOnly(line string) string {
	mask := lspTextualCodeMask(line)
	out := []byte(line)
	for i := range out {
		if !mask[i] {
			out[i] = ' '
		}
	}
	return string(out)
}

func lspDeclarationStartsWithName(trimmed string, keyword string, name string) bool {
	if !strings.HasPrefix(trimmed, keyword) {
		return false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(trimmed, keyword))
	if rest == "" {
		return false
	}
	declared := lspLeadingIdentifier(rest)
	return declared == name
}

func lspFunctionParamsDeclareName(trimmed string, name string) bool {
	if !strings.HasPrefix(trimmed, "func ") && !strings.HasPrefix(trimmed, "async func ") {
		return false
	}
	open := strings.Index(trimmed, "(")
	close := strings.Index(trimmed, ")")
	if open < 0 || close < 0 || close <= open {
		return false
	}
	for _, param := range strings.Split(trimmed[open+1:close], ",") {
		beforeType, _, _ := strings.Cut(param, ":")
		fields := strings.Fields(strings.TrimSpace(beforeType))
		if len(fields) == 0 {
			continue
		}
		if fields[len(fields)-1] == name {
			return true
		}
	}
	return false
}

func lspLeadingIdentifier(text string) string {
	if text == "" || !isLSPIdentifierStart(text[0]) {
		return ""
	}
	end := 1
	for end < len(text) && isLSPIdentifierChar(text[end]) {
		end++
	}
	return text[:end]
}

func lspCompletionItems(analysis compiler.LSPAnalysis) []map[string]any {
	out := make([]map[string]any, 0, len(analysis.Symbols))
	for _, sym := range analysis.Symbols {
		item := map[string]any{
			"label": sym.Name,
			"kind":  lspCompletionKind(sym.Kind),
		}
		if sym.Detail != "" {
			item["detail"] = sym.Detail
		}
		out = append(out, item)
	}
	return out
}

func lspCodeActions(
	text string,
	uri string,
	requestDiagnostics []lspCodeActionDiagnostic,
	analysisDiagnostics []compiler.Diagnostic,
) []map[string]any {
	diagnostics := requestDiagnostics
	if len(diagnostics) == 0 {
		diagnostics = lspCodeActionDiagnosticsFromCompiler(analysisDiagnostics)
	}
	actions := []map[string]any{}
	for _, diag := range diagnostics {
		action, ok := lspMissingEffectCodeAction(text, uri, diag)
		if ok {
			actions = append(actions, action)
		}
	}
	return actions
}

func lspCodeActionDiagnosticsFromCompiler(diags []compiler.Diagnostic) []lspCodeActionDiagnostic {
	out := make([]lspCodeActionDiagnostic, 0, len(diags))
	for _, diag := range diags {
		code, err := json.Marshal(diag.Code)
		if err != nil {
			continue
		}
		out = append(out, lspCodeActionDiagnostic{
			Code:    code,
			Message: diag.Message,
		})
	}
	return out
}

func lspMissingEffectCodeAction(
	text string,
	uri string,
	diag lspCodeActionDiagnostic,
) (map[string]any, bool) {
	code := lspDiagnosticCodeString(diag.Code)
	if code != "" && code != "TETRA2001" {
		return nil, false
	}
	match := lspMissingEffectDiagnosticRE.FindStringSubmatch(diag.Message)
	if len(match) != 3 {
		return nil, false
	}
	funcName := match[1]
	effect := match[2]
	line, character, newText, ok := lspFindUsesInsertion(text, funcName, effect)
	if !ok {
		return nil, false
	}
	edit := map[string]any{
		"range": map[string]any{
			"start": map[string]int{"line": line, "character": character},
			"end":   map[string]int{"line": line, "character": character},
		},
		"newText": newText,
	}
	return map[string]any{
		"title": fmt.Sprintf("Add uses %s to function %s", effect, funcName),
		"kind":  "quickfix",
		"edit": map[string]any{
			"changes": map[string]any{
				uri: []map[string]any{edit},
			},
		},
	}, true
}

func lspDiagnosticCodeString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var codeString string
	if err := json.Unmarshal(raw, &codeString); err == nil {
		return codeString
	}
	return ""
}

func lspFindUsesInsertion(text string, funcName string, effect string) (int, int, string, bool) {
	lines := strings.Split(text, "\n")
	prefix := "func " + funcName + "("
	for lineIdx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		if usesIdx := strings.Index(trimmed, " uses "); usesIdx >= 0 {
			colonIdx := strings.LastIndex(trimmed, ":")
			if colonIdx > usesIdx {
				existing := strings.TrimSpace(trimmed[usesIdx+len(" uses ") : colonIdx])
				if lspUsesContainsEffect(existing, effect) {
					return 0, 0, "", false
				}
				return lineIdx, lspLineIndent(line) + colonIdx, ", " + effect, true
			}
		}
		if colonIdx := strings.LastIndex(trimmed, ":"); colonIdx >= 0 {
			return lineIdx, lspLineIndent(line) + colonIdx, " uses " + effect, true
		}
		if lineIdx+1 >= len(lines) {
			return 0, 0, "", false
		}
		nextLine := lines[lineIdx+1]
		nextTrimmed := strings.TrimSpace(nextLine)
		if !strings.HasPrefix(nextTrimmed, "uses ") {
			return 0, 0, "", false
		}
		colonIdx := strings.LastIndex(nextTrimmed, ":")
		if colonIdx < 0 {
			return 0, 0, "", false
		}
		existing := strings.TrimSpace(nextTrimmed[len("uses "):colonIdx])
		if lspUsesContainsEffect(existing, effect) {
			return 0, 0, "", false
		}
		return lineIdx + 1, lspLineIndent(nextLine) + colonIdx, ", " + effect, true
	}
	return 0, 0, "", false
}

func lspLineIndent(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}

func lspUsesContainsEffect(existing string, effect string) bool {
	for _, item := range strings.Split(existing, ",") {
		if strings.TrimSpace(item) == effect {
			return true
		}
	}
	return false
}

func lspCompletionKind(kind string) int {
	switch kind {
	case "function", "extension-method":
		return 3
	case "const":
		return 21
	case "enum":
		return 13
	case "protocol":
		return 8
	case "struct":
		return 7
	default:
		return 6
	}
}

func lspFormattingEdits(text string, uri string) ([]map[string]any, error) {
	formatted, err := compiler.FormatSource([]byte(text), uri)
	if err != nil {
		return nil, err
	}
	if string(formatted) == text {
		return []map[string]any{}, nil
	}
	line, character := lspFullDocumentEnd(text)
	return []map[string]any{{
		"range": map[string]any{
			"start": map[string]int{"line": 0, "character": 0},
			"end":   map[string]int{"line": line, "character": character},
		},
		"newText": string(formatted),
	}}, nil
}

func lspFullDocumentEnd(text string) (int, int) {
	parts := strings.Split(text, "\n")
	return len(parts) - 1, len(parts[len(parts)-1])
}

func lspSymbolKind(kind string) int {
	switch kind {
	case "function":
		return 12
	case "extension-method":
		return 6
	case "const":
		return 14
	case "val", "var":
		return 13
	case "enum":
		return 10
	case "protocol":
		return 11
	case "struct":
		return 23
	default:
		return 13
	}
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

// ---- lsp_protocol.go ----

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

var lspMissingEffectDiagnosticRE = regexp.MustCompile(
	`^function '([^']+)' uses effect '([^']+)' but does not declare it$`,
)

// ---- lsp_wire.go ----

const maxLSPContentLength = 16 * 1024 * 1024

func readLSPMessage(reader *bufio.Reader) ([]byte, error) {
	length := -1
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("invalid LSP header %q", line)
		}
		if strings.EqualFold(strings.TrimSpace(key), "Content-Length") {
			parsed, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length")
			}
			if parsed > maxLSPContentLength {
				return nil, fmt.Errorf(
					"Content-Length too large: %d exceeds max %d",
					parsed,
					maxLSPContentLength,
				)
			}
			length = parsed
		}
	}
	if length < 0 {
		return nil, fmt.Errorf("missing Content-Length")
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, err
	}
	return body, nil
}

func writeLSPResponse(w io.Writer, id any, result any) error {
	return writeLSPMessage(w, map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

func writeLSPNotification(w io.Writer, method string, params any) error {
	return writeLSPMessage(w, map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

func writeLSPMessage(w io.Writer, msg any) error {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(msg); err != nil {
		return err
	}
	raw := bytes.TrimRight(b.Bytes(), "\n")
	_, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n%s", len(raw), raw)
	return err
}

// ---- smoke.go ----

type smokeCaseReport struct {
	Name               string `json:"name"`
	SrcPath            string `json:"src_path"`
	OutPath            string `json:"out_path"`
	ExpectedExit       int    `json:"expected_exit"`
	Unsupported        bool   `json:"unsupported,omitempty"`
	ExpectedDiagnostic string `json:"expected_diagnostic,omitempty"`
	Diagnostic         string `json:"diagnostic,omitempty"`
	ActualExit         *int   `json:"actual_exit,omitempty"`
	Ran                bool   `json:"ran"`
	Pass               bool   `json:"pass"`
	Error              string `json:"error,omitempty"`
}

type islandsDebugScopeRow struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	CaseName string `json:"case_name,omitempty"`
	SrcPath  string `json:"src_path,omitempty"`
	Evidence string `json:"evidence"`
	Reason   string `json:"reason"`
}

type smokeReport struct {
	Timestamp         string                 `json:"timestamp"`
	Target            string                 `json:"target"`
	BuildOnly         bool                   `json:"build_only"`
	Runner            string                 `json:"runner,omitempty"`
	Host              string                 `json:"host"`
	Version           string                 `json:"version"`
	GitHead           string                 `json:"git_head,omitempty"`
	IslandsDebug      bool                   `json:"islands_debug"`
	IslandsDebugScope []islandsDebugScopeRow `json:"islands_debug_scope,omitempty"`
	Total             int                    `json:"total"`
	Passed            int                    `json:"passed"`
	Failed            int                    `json:"failed"`
	Cases             []smokeCaseReport      `json:"cases"`
}

type smokeCase struct {
	name               string
	srcPath            string
	expectedExit       int
	debugOnly          bool
	expectedDiagnostic string
}

type smokeListCase struct {
	Name               string `json:"name"`
	SrcPath            string `json:"src_path"`
	TargetGroup        string `json:"target_group"`
	ExpectedExit       int    `json:"expected_exit"`
	Unsupported        bool   `json:"unsupported,omitempty"`
	ExpectedDiagnostic string `json:"expected_diagnostic,omitempty"`
	DebugOnly          bool   `json:"debug_only,omitempty"`
}

type smokeExcludedExample struct {
	SrcPath string `json:"src_path"`
	Reason  string `json:"reason"`
}

type smokeListReport struct {
	Target           string                 `json:"target"`
	BuildOnly        bool                   `json:"build_only"`
	RunSupported     bool                   `json:"run_supported"`
	Total            int                    `json:"total"`
	IslandsDebug     bool                   `json:"islands_debug"`
	Cases            []smokeListCase        `json:"cases"`
	ExcludedExamples []smokeExcludedExample `json:"excluded_examples,omitempty"`
}

func runSmoke(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("smoke", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", defaultTarget(), "target triple ("+supportedTargetsHelp+")")
	runBuilt := fs.Bool("run", true, "run built binaries when host matches target")
	reportPath := fs.String("report", "", "write smoke report")
	reportFormat := fs.String(
		"report-format",
		outputformat.JSON,
		"smoke report format: json, toon, or both",
	)
	listCases := fs.Bool("list", false, "list smoke cases without building")
	listFormat := fs.String("format", outputformat.Text, "smoke list format: text, json, or toon")
	islandsDebug := fs.Bool("islands-debug", false, "enable islands debug runtime checks")
	runtimeMode := fs.String("runtime", "auto", "actors runtime: auto, selfhost, or builtin")
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "smoke does not accept positional arguments")
		return 2
	}
	if *listCases {
		tgt, ok := parseBuildTargetOrReport(*target, "text", stderr)
		if !ok {
			return 2
		}
		return writeSmokeList(
			stdout,
			stderr,
			smokeCasesForTarget(*islandsDebug, tgt),
			*islandsDebug,
			*listFormat,
			tgt,
		)
	}
	if *listFormat != "text" {
		fmt.Fprintln(stderr, "--format is only supported with --list")
		return 2
	}
	tgt, ok := parseBuildTargetOrReport(*target, "text", stderr)
	if !ok {
		return 2
	}
	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	tmpDir, err := os.MkdirTemp("", "tetra-smoke-*")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer os.RemoveAll(tmpDir)
	outputDir := tmpDir
	if tgt.Arch == ctarget.ArchWASM32 && *reportPath != "" {
		outputDir = smokeArtifactDir(*reportPath)
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}

	host := ""
	hostTriple, hostOK := hostTarget()
	if hostOK {
		host = hostTriple
	}
	cases := smokeCasesForTarget(*islandsDebug, tgt)
	shouldRun := *runBuilt && hostOK && hostTriple == tgt.Triple
	runWASI := false
	var wasiRunner wasiRunner
	runWeb := false
	var webRunner string
	if *runBuilt && tgt.Triple == "wasm32-wasi" {
		runner, err := discoverWASIRunner(repoRoot)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		wasiRunner = runner
		runWASI = true
		shouldRun = true
	}
	if *runBuilt && tgt.Triple == "wasm32-web" {
		runner, err := discoverWebRunner()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		webRunner = runner
		runWeb = true
		shouldRun = true
	}
	opt, err := buildOptions("exe", *runtimeMode, *islandsDebug, "", nil, *jobs)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	report := smokeReport{
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Target:       tgt.Triple,
		BuildOnly:    ctarget.IsBuildOnlyTarget(tgt.Triple),
		Runner:       runnerName(wasiRunner.Name, webRunner),
		Host:         host,
		Version:      compiler.Version(),
		GitHead:      gitHead(repoRoot),
		IslandsDebug: *islandsDebug,
	}
	if *islandsDebug {
		report.IslandsDebugScope = islandsDebugScopeRows()
	}
	for _, c := range cases {
		outPath := filepath.Join(outputDir, c.name+tgt.ExeExt)
		srcAbs := filepath.Join(repoRoot, filepath.FromSlash(c.srcPath))
		caseReport := smokeCaseReport{
			Name:               c.name,
			SrcPath:            c.srcPath,
			OutPath:            outPath,
			ExpectedExit:       c.expectedExit,
			Unsupported:        c.expectedDiagnostic != "",
			ExpectedDiagnostic: c.expectedDiagnostic,
		}
		if _, err := compiler.BuildFileWithStatsOpt(srcAbs, outPath, tgt.Triple, opt); err != nil {
			if c.expectedDiagnostic != "" {
				caseReport.OutPath = ""
				caseReport.Diagnostic = err.Error()
				if strings.Contains(err.Error(), c.expectedDiagnostic) {
					caseReport.Pass = true
				} else {
					caseReport.Error = "build diagnostic mismatch: " + err.Error()
				}
				report.Cases = append(report.Cases, caseReport)
				continue
			}
			caseReport.Error = "build: " + err.Error()
			report.Cases = append(report.Cases, caseReport)
			continue
		}
		if c.expectedDiagnostic != "" {
			caseReport.Error = "build succeeded, want diagnostic containing " + c.expectedDiagnostic
			report.Cases = append(report.Cases, caseReport)
			continue
		}
		if shouldRun {
			caseReport.Ran = true
			var actual int
			if runWASI {
				actual, err = execWASMProgramWithRunner(outPath, wasiRunner, io.Discard, io.Discard)
				if err != nil {
					caseReport.Error = "run: " + err.Error()
					caseReport.Pass = false
					report.Cases = append(report.Cases, caseReport)
					continue
				}
			} else if runWeb {
				actual, err = execWebProgramWithBrowserRunner(outPath, webRunner, io.Discard, io.Discard)
				if err != nil {
					caseReport.Error = "run: " + err.Error()
					caseReport.Pass = false
					report.Cases = append(report.Cases, caseReport)
					continue
				}
			} else {
				actual = execProgram(outPath, io.Discard, io.Discard)
			}
			caseReport.ActualExit = &actual
			caseReport.Pass = actual == c.expectedExit
		} else {
			caseReport.Pass = true
		}
		report.Cases = append(report.Cases, caseReport)
	}

	passed := 0
	for _, c := range report.Cases {
		if c.Pass {
			passed++
		}
	}
	report.Total = len(report.Cases)
	report.Passed = passed
	report.Failed = report.Total - report.Passed
	fmt.Fprintf(stdout, "Smoke %s: %d/%d passed\n", tgt.Triple, passed, len(report.Cases))
	if *reportPath != "" {
		if _, err := outputformat.WriteStructuredFiles(*reportPath, *reportFormat, report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	if passed != len(report.Cases) {
		return 1
	}
	return 0
}

func islandsDebugScopeRows() []islandsDebugScopeRow {
	return []islandsDebugScopeRow{
		{
			Name:     "overflow_trap",
			Status:   "live_trap",
			CaseName: "islands_overflow",
			SrcPath:  "examples/memory/islands/islands_overflow.tetra",
			Evidence: ("tetra smoke --islands-debug executes islands_overflow and " +
				"observes non-zero trap exit"),
			Reason: "live sanitizer trap row for bounded island allocation overflow",
		},
		{
			Name:     "double_free",
			Status:   "static_only_nonclaim",
			CaseName: "islands_double_free",
			SrcPath:  "examples/memory/islands/islands_double_free.tetra",
			Evidence: ("compiler/tests/runtime/resource_finalization_test.go; compiler/" +
				"compiler_suite_test.go; compiler/internal/backend/x64abi/abi_test.go"),
			Reason: ("static semantics reject double-free before runtime; backend " +
				"freed-marker trap is covered, but no live double-free bypass is claimed"),
		},
		{
			Name:   "use_after_free",
			Status: "static_only_nonclaim",
			Evidence: ("compiler/internal/validation/validation_test.go; compiler/tests/" +
				"runtime/resource_finalization_test.go"),
			Reason: ("static validation rejects island use-after-free before runtime; " +
				"no live UAF sanitizer row is claimed"),
		},
		{
			Name:   "stale_epoch",
			Status: "static_only_nonclaim",
			Evidence: ("compiler/tests/runtime/resource_finalization_test.go; compiler/" +
				"internal/islandkernel/kernel_test.go; compiler/internal/memoryfacts_" +
				"test/report_test.go"),
			Reason: ("reset/stale-epoch misuse is covered by static/kernel/report " +
				"validators; no live stale-epoch sanitizer row is claimed"),
		},
		{
			Name:   "wrong_island",
			Status: "static_only_nonclaim",
			Evidence: ("compiler/internal/islandkernel/kernel_test.go; tools/validators/" +
				"islandproof/proof_test.go"),
			Reason: ("wrong-island proof/report misuse is covered by static verifier " +
				"evidence; no live wrong-island sanitizer row is claimed"),
		},
	}
}

func runnerName(names ...string) string {
	for _, name := range names {
		if name != "" {
			return name
		}
	}
	return ""
}

func smokeArtifactDir(reportPath string) string {
	base := filepath.Base(reportPath)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	if stem == "" || stem == "." {
		stem = "smoke"
	}
	return filepath.Join(filepath.Dir(reportPath), stem+"-artifacts")
}

func writeSmokeList(
	stdout io.Writer,
	stderr io.Writer,
	cases []smokeCase,
	islandsDebug bool,
	format string,
	tgt ctarget.Target,
) int {
	host, hostOK := hostTarget()
	runSupported, _, _ := targetRunSupport(tgt, host, hostOK)
	report := smokeListReport{
		Target:       tgt.Triple,
		BuildOnly:    ctarget.IsBuildOnlyTarget(tgt.Triple),
		RunSupported: runSupported,
		Total:        len(cases),
		IslandsDebug: islandsDebug,
		Cases:        make([]smokeListCase, 0, len(cases)),
	}
	for _, c := range cases {
		report.Cases = append(report.Cases, smokeListCase{
			Name:               c.name,
			SrcPath:            c.srcPath,
			TargetGroup:        smokeTargetGroup(tgt.Triple),
			ExpectedExit:       c.expectedExit,
			Unsupported:        c.expectedDiagnostic != "",
			ExpectedDiagnostic: c.expectedDiagnostic,
			DebugOnly:          c.debugOnly,
		})
	}
	if repoRoot, err := findRepoRoot(); err == nil {
		report.ExcludedExamples = smokeExampleExclusions(repoRoot, cases, tgt)
	}
	switch format {
	case "", "text":
		for _, c := range report.Cases {
			if c.DebugOnly {
				fmt.Fprintf(stdout, "%s %s exit=%d debug-only\n", c.Name, c.SrcPath, c.ExpectedExit)
			} else {
				fmt.Fprintf(stdout, "%s %s exit=%d\n", c.Name, c.SrcPath, c.ExpectedExit)
			}
		}
		return 0
	case outputformat.JSON, outputformat.TOON:
		if err := outputformat.WriteStructured(stdout, format, report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func smokeExampleExclusions(
	repoRoot string,
	cases []smokeCase,
	tgt ctarget.Target,
) []smokeExcludedExample {
	covered := map[string]bool{}
	for _, c := range cases {
		covered[filepath.ToSlash(filepath.Clean(c.srcPath))] = true
	}

	examplesRoot := filepath.Join(repoRoot, "examples")
	var out []smokeExcludedExample
	walkErr := filepath.WalkDir(examplesRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !compiler.IsSourceFile(path) {
			return nil
		}
		rel, err := filepath.Rel(examplesRoot, path)
		if err != nil {
			return nil
		}
		srcPath := "examples/" + filepath.ToSlash(rel)
		if covered[srcPath] {
			return nil
		}
		out = append(out, smokeExcludedExample{
			SrcPath: srcPath,
			Reason:  fmt.Sprintf("not part of %s smoke profile", tgt.Triple),
		})
		return nil
	})
	if walkErr != nil && len(out) == 0 {
		return out
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SrcPath < out[j].SrcPath })
	return out
}

func smokeTargetGroup(target string) string {
	if target == "wasm32-wasi" || target == "wasm32-web" {
		return "wasm"
	}
	return "native"
}

// ---- smoke_registry.go ----

type smokeSourceSet string

const (
	smokeSourceSetNative            smokeSourceSet = "native"
	smokeSourceSetWasmBuildOnly     smokeSourceSet = "wasm-build-only"
	smokeSourceSetWasmWASIBuildOnly smokeSourceSet = "wasm-wasi-build-only"
)

var smokeCaseRegistry = map[smokeSourceSet][]smokeCase{
	smokeSourceSetNative: {
		{
			name:         "islands_hello",
			srcPath:      "examples/memory/islands/islands_hello.tetra",
			expectedExit: 0,
		},
		{
			name:         "islands_i32",
			srcPath:      "examples/memory/islands/islands_i32.tetra",
			expectedExit: 55,
		},
		{
			name:         "islands_overflow",
			srcPath:      "examples/memory/islands/islands_overflow.tetra",
			expectedExit: 1,
		},
		{name: "mmio_smoke", srcPath: "examples/memory/raw/mmio_smoke.tetra", expectedExit: 123},
		{
			name:         "cap_mem_smoke",
			srcPath:      "examples/memory/raw/cap_mem_smoke.tetra",
			expectedExit: 77,
		},
		{name: "memset_smoke", srcPath: "examples/memory/raw/memset_smoke.tetra", expectedExit: 88},
		{
			name:         "actors_pingpong",
			srcPath:      "examples/actors/actors_pingpong.tetra",
			expectedExit: 0,
		},
		{
			name:         "actor_sleep_pingpong",
			srcPath:      "examples/actors/actor_sleep_pingpong.tetra",
			expectedExit: 0,
		},
		{name: "flow_hello", srcPath: "examples/flow/flow_hello.tetra", expectedExit: 0},
		{
			name:         "flow_struct_smoke",
			srcPath:      "examples/flow/flow_struct_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "flow_islands_smoke",
			srcPath:      "examples/flow/flow_islands_smoke.tetra",
			expectedExit: 0,
		},
		{
			name:         "flow_unsafe_cap_mem_smoke",
			srcPath:      "examples/flow/flow_unsafe_cap_mem_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "ui_native_shell_smoke",
			srcPath:      "examples/ui/ui_native_shell_smoke.tetra",
			expectedExit: 0,
		},
		{name: "bool_smoke", srcPath: "examples/smoke/scalars/bool_smoke.tetra", expectedExit: 42},
		{
			name:         "for_range_smoke",
			srcPath:      "examples/smoke/control/for_range_smoke.tetra",
			expectedExit: 55,
		},
		{
			name:         "for_collection_smoke",
			srcPath:      "examples/smoke/control/for_collection_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "for_collection_u8_smoke",
			srcPath:      "examples/smoke/control/for_collection_u8_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "loop_control_smoke",
			srcPath:      "examples/smoke/control/loop_control_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "complex_control_flow_smoke",
			srcPath:      "examples/smoke/control/complex_control_flow_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "unary_not_smoke",
			srcPath:      "examples/smoke/scalars/unary_not_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "const_smoke",
			srcPath:      "examples/smoke/scalars/const_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "const_bool_smoke",
			srcPath:      "examples/smoke/scalars/const_bool_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "local_const_smoke",
			srcPath:      "examples/smoke/scalars/local_const_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "compound_assignment_smoke",
			srcPath:      "examples/smoke/scalars/compound_assignment_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "else_if_smoke",
			srcPath:      "examples/smoke/control/else_if_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "enum_match_smoke",
			srcPath:      "examples/smoke/types/enum_match_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "enum_exhaustive_match_smoke",
			srcPath:      "examples/smoke/types/enum_exhaustive_match_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "effects_io_smoke",
			srcPath:      "examples/effects/effects_io_smoke.tetra",
			expectedExit: 0,
		},
		{
			name:         "effects_mem_smoke",
			srcPath:      "examples/effects/effects_mem_smoke.tetra",
			expectedExit: 17,
		},
		{
			name:         "effects_actors_smoke",
			srcPath:      "examples/effects/effects_actors_smoke.tetra",
			expectedExit: 0,
		},
		{
			name:         "optional_smoke",
			srcPath:      "examples/smoke/types/optional_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "optional_match_smoke",
			srcPath:      "examples/smoke/types/optional_match_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "optional_match_some_smoke",
			srcPath:      "examples/smoke/types/optional_match_some_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "ownership_smoke",
			srcPath:      "examples/memory/ownership/ownership_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "typed_errors_smoke",
			srcPath:      "examples/smoke/errors/typed_errors_smoke.tetra",
			expectedExit: 42,
		},
		{name: "async_smoke", srcPath: "examples/async/async_smoke.tetra", expectedExit: 42},
		{name: "task_smoke", srcPath: "examples/tasks/task_smoke.tetra", expectedExit: 42},
		{
			name:         "time_sleep_smoke",
			srcPath:      "examples/async/time_sleep_smoke.tetra",
			expectedExit: 0,
		},
		{
			name:         "task_sleep_deadline_smoke",
			srcPath:      "examples/tasks/task_sleep_deadline_smoke.tetra",
			expectedExit: 0,
		},
		{
			name:         "task_join_wait_smoke",
			srcPath:      "examples/tasks/task_join_wait_smoke.tetra",
			expectedExit: 5,
		},
		{
			name:         "task_group_cancel_smoke",
			srcPath:      "examples/tasks/task_group_cancel_smoke.tetra",
			expectedExit: 1,
		},
		{
			name:         "task_group_lifecycle_smoke",
			srcPath:      "examples/tasks/task_group_lifecycle_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "deadline_aware_waits_smoke",
			srcPath:      "examples/async/deadline_aware_waits_smoke.tetra",
			expectedExit: 0,
		},
		{
			name:         "wait_composition_smoke",
			srcPath:      "examples/async/wait_composition_smoke.tetra",
			expectedExit: 0,
		},
		{
			name:         "core_math_smoke",
			srcPath:      "examples/core/data/core_math_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "core_memory_smoke",
			srcPath:      "examples/core/memory/core_memory_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "core_strings_smoke",
			srcPath:      "examples/core/data/core_strings_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "core_slices_smoke",
			srcPath:      "examples/core/data/core_slices_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "core_io_smoke",
			srcPath:      "examples/core/platform/core_io_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "core_testing_smoke",
			srcPath:      "examples/core/runtime/core_testing_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "core_collections_smoke",
			srcPath:      "examples/core/data/core_collections_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "core_component_smoke",
			srcPath:      "examples/core/surface/core_component_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "core_serialization_smoke",
			srcPath:      "examples/core/data/core_serialization_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "core_filesystem_smoke",
			srcPath:      "examples/core/platform/core_filesystem_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "core_networking_smoke",
			srcPath:      "examples/core/platform/core_networking_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "core_async_smoke",
			srcPath:      "examples/async/core_async_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "core_sync_smoke",
			srcPath:      "examples/core/runtime/core_sync_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "core_time_smoke",
			srcPath:      "examples/core/platform/core_time_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "core_crypto_smoke",
			srcPath:      "examples/core/memory/core_crypto_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "core_capability_smoke",
			srcPath:      "examples/core/memory/core_capability_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "extension_smoke",
			srcPath:      "examples/smoke/language/extension_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "generic_smoke",
			srcPath:      "examples/smoke/language/generic_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "protocol_impl_smoke",
			srcPath:      "examples/smoke/language/protocol_impl_smoke.tetra",
			expectedExit: 42,
		},
		{
			name:         "surface_counter",
			srcPath:      "examples/surface/runtime/surface_counter.tetra",
			expectedExit: 1,
		},
		{
			name:         "surface_text_input",
			srcPath:      "examples/surface/runtime/surface_text_input.tetra",
			expectedExit: 42,
		},
		{
			name:         "surface_migration_ui_web_smoke",
			srcPath:      "examples/surface/migration/surface_migration_ui_web_smoke.tetra",
			expectedExit: 2,
		},
		{
			name:         "surface_migration_ui_native_shell_smoke",
			srcPath:      "examples/surface/migration/surface_migration_ui_native_shell_smoke.tetra",
			expectedExit: 11,
		},
		{
			name:         "surface_migration_dogfood_web_ui",
			srcPath:      "examples/surface/migration/surface_migration_dogfood_web_ui.tetra",
			expectedExit: 3,
		},
		{
			name:         "surface_migration_tetra_control_center",
			srcPath:      "examples/surface/migration/surface_migration_tetra_control_center.tetra",
			expectedExit: 5,
		},
		{
			name:         "dogfood_cli",
			srcPath:      "examples/projects/dogfood_cli/src/main.tetra",
			expectedExit: 0,
		},
		{
			name:         "dogfood_actor_task",
			srcPath:      "examples/projects/dogfood_actor_task/src/main.tetra",
			expectedExit: 0,
		},
	},
	smokeSourceSetWasmBuildOnly: {
		{name: "legacy_hello", srcPath: "examples/smoke/basic/hello.tetra", expectedExit: 0},
		{
			name:         "effects_io_smoke",
			srcPath:      "examples/effects/effects_io_smoke.tetra",
			expectedExit: 0,
		},
		{name: "ui_web_smoke", srcPath: "examples/ui/ui_web_smoke.tetra", expectedExit: 0},
		{
			name:         "core_slices_smoke",
			srcPath:      "examples/core/data/core_slices_smoke.tetra",
			expectedExit: 0,
		},
		{
			name:         "wasm_globals_smoke",
			srcPath:      "examples/wasm/wasm_globals_smoke.tetra",
			expectedExit: 0,
		},
		{
			name:         "surface_counter",
			srcPath:      "examples/surface/runtime/surface_counter.tetra",
			expectedExit: 1,
		},
		{
			name:         "surface_text_input",
			srcPath:      "examples/surface/runtime/surface_text_input.tetra",
			expectedExit: 42,
		},
		{
			name:         "wasm_multi_return_2_smoke",
			srcPath:      "examples/wasm/wasm_multi_return_2_smoke.tetra",
			expectedExit: 0,
		},
		{
			name:         "wasm_multi_return_3_smoke",
			srcPath:      "examples/wasm/wasm_multi_return_3_smoke.tetra",
			expectedExit: 0,
		},
		{
			name:         "wasm_multi_return_4_smoke",
			srcPath:      "examples/wasm/wasm_multi_return_4_smoke.tetra",
			expectedExit: 0,
		},
		{
			name:         "dogfood_wasi",
			srcPath:      "examples/projects/dogfood_wasi/src/main.tetra",
			expectedExit: 0,
		},
		{
			name:         "dogfood_web_ui",
			srcPath:      "examples/projects/dogfood_web_ui/src/main.tetra",
			expectedExit: 0,
		},
		{
			name:               "time_sleep_smoke",
			srcPath:            "examples/async/time_sleep_smoke.tetra",
			expectedExit:       0,
			expectedDiagnostic: "runtime not supported on wasm32",
		},
		{
			name:               "task_smoke",
			srcPath:            "examples/tasks/task_smoke.tetra",
			expectedExit:       42,
			expectedDiagnostic: "runtime not supported on wasm32",
		},
		{
			name:               "actors_pingpong",
			srcPath:            "examples/actors/actors_pingpong.tetra",
			expectedExit:       0,
			expectedDiagnostic: "runtime not supported on wasm32",
		},
	},
	smokeSourceSetWasmWASIBuildOnly: {
		{name: "legacy_hello", srcPath: "examples/smoke/basic/hello.tetra", expectedExit: 0},
		{
			name:         "effects_io_smoke",
			srcPath:      "examples/effects/effects_io_smoke.tetra",
			expectedExit: 0,
		},
		{name: "ui_web_smoke", srcPath: "examples/ui/ui_web_smoke.tetra", expectedExit: 0},
		{
			name:         "core_slices_smoke",
			srcPath:      "examples/core/data/core_slices_smoke.tetra",
			expectedExit: 0,
		},
		{
			name:         "wasm_globals_smoke",
			srcPath:      "examples/wasm/wasm_globals_smoke.tetra",
			expectedExit: 0,
		},
		{
			name:         "wasm_multi_return_2_smoke",
			srcPath:      "examples/wasm/wasm_multi_return_2_smoke.tetra",
			expectedExit: 0,
		},
		{
			name:         "wasm_multi_return_3_smoke",
			srcPath:      "examples/wasm/wasm_multi_return_3_smoke.tetra",
			expectedExit: 0,
		},
		{
			name:         "wasm_multi_return_4_smoke",
			srcPath:      "examples/wasm/wasm_multi_return_4_smoke.tetra",
			expectedExit: 0,
		},
		{
			name:         "dogfood_wasi",
			srcPath:      "examples/projects/dogfood_wasi/src/main.tetra",
			expectedExit: 0,
		},
		{
			name:         "dogfood_web_ui",
			srcPath:      "examples/projects/dogfood_web_ui/src/main.tetra",
			expectedExit: 0,
		},
		{
			name:               "time_sleep_smoke",
			srcPath:            "examples/async/time_sleep_smoke.tetra",
			expectedExit:       0,
			expectedDiagnostic: "runtime not supported on wasm32",
		},
		{
			name:               "task_smoke",
			srcPath:            "examples/tasks/task_smoke.tetra",
			expectedExit:       42,
			expectedDiagnostic: "runtime not supported on wasm32",
		},
		{
			name:               "actors_pingpong",
			srcPath:            "examples/actors/actors_pingpong.tetra",
			expectedExit:       0,
			expectedDiagnostic: "runtime not supported on wasm32",
		},
	},
}

func smokeRegistryCases(set smokeSourceSet) []smokeCase {
	cases := smokeCaseRegistry[set]
	out := make([]smokeCase, len(cases))
	copy(out, cases)
	return out
}

func smokeCases(islandsDebug bool) []smokeCase {
	return smokeRegistryCases(smokeSourceSetNative)
}

func smokeCasesForTarget(islandsDebug bool, tgt ctarget.Target) []smokeCase {
	if tgt.Triple == "wasm32-wasi" {
		return smokeRegistryCases(smokeSourceSetWasmWASIBuildOnly)
	}
	if tgt.Triple == "wasm32-web" {
		return smokeRegistryCases(smokeSourceSetWasmBuildOnly)
	}
	cases := smokeCases(islandsDebug)
	switch tgt.Triple {
	case "macos-x64", "windows-x64":
		for i := range cases {
			if cases[i].name == "core_filesystem_smoke" {
				cases[i].expectedDiagnostic = "filesystem runtime not supported on " + tgt.Triple
			}
			if smokeCaseUsesSurfaceRuntime(cases[i].name) {
				cases[i].expectedDiagnostic = "surface runtime not supported on " + tgt.Triple
			}
		}
	}
	return cases
}

func smokeCaseUsesSurfaceRuntime(name string) bool {
	switch name {
	case "core_component_smoke",
		"surface_counter",
		"surface_text_input",
		"surface_migration_ui_web_smoke",
		"surface_migration_ui_native_shell_smoke",
		"surface_migration_dogfood_web_ui",
		"surface_migration_tetra_control_center":
		return true
	default:
		return false
	}
}
