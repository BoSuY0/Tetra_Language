package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"tetra_language/compiler"
	"tetra_language/internal/outputformat"
)

func runLSP(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("lsp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	smokePath := fs.String("stdio-smoke", "", "analyze one .t4/.tetra file and print LSP-basic JSON")
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
			fmt.Fprintln(stderr, "lsp --stdio only supports --format=json because stdio uses framed JSON-RPC")
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
			if err := writeLSPErrorResponse(stdout, lspIDFromRawMessage(body), code, "invalid request: "+err.Error()); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			continue
		}
		if req.JSONRPC != "2.0" {
			if err := writeLSPRequestError(stdout, req, lspErrorInvalidRequest, `invalid request: jsonrpc must be "2.0"`); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			continue
		}
		if req.Method == "" {
			if err := writeLSPRequestError(stdout, req, lspErrorInvalidRequest, "invalid request: method is required"); err != nil {
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
			analysis := compiler.AnalyzeLSPSource([]byte(params.TextDocument.Text), params.TextDocument.URI)
			openDocs[params.TextDocument.URI] = lspOpenDocument{Text: params.TextDocument.Text, Analysis: analysis}
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
			if err := lspValidateTextDocumentPosition(params.TextDocument.URI, params.Position.Line, params.Position.Character); err != nil {
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
			if err := lspValidateTextDocumentPosition(params.TextDocument.URI, params.Position.Line, params.Position.Character); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if req.ID != nil {
				var result any
				if doc, ok := openDocs[params.TextDocument.URI]; ok {
					result = lspDefinitionLocations(doc, params.TextDocument.URI, params.Position.Line, params.Position.Character)
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
			if err := lspValidateTextDocumentPosition(params.TextDocument.URI, params.Position.Line, params.Position.Character); err != nil {
				if err := writeLSPRequestError(stdout, req, lspErrorInvalidParams, err.Error()); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				continue
			}
			if req.ID != nil {
				var result any = []map[string]any{}
				if doc, ok := openDocs[params.TextDocument.URI]; ok {
					result = lspReferenceLocations(doc, params.TextDocument.URI, params.Position.Line, params.Position.Character, params.Context.IncludeDeclaration)
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
			if err := lspValidateTextDocumentPosition(params.TextDocument.URI, params.Position.Line, params.Position.Character); err != nil {
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
					result = lspRenameWorkspaceEdit(doc, params.TextDocument.URI, params.Position.Line, params.Position.Character, params.NewName)
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
			if err := lspValidateTextDocumentPosition(params.TextDocument.URI, params.Position.Line, params.Position.Character); err != nil {
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
					actions = lspCodeActions(doc.Text, params.TextDocument.URI, params.Context.Diagnostics, doc.Analysis.Diagnostics)
				}
				if err := writeLSPResponse(stdout, req.ID.JSONValue(), actions); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		default:
			if req.ID != nil {
				if err := writeLSPErrorResponse(stdout, req.ID.JSONValue(), lspErrorMethodNotFound, "unknown method: "+req.Method); err != nil {
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
			return map[string]any{"contents": map[string]string{"kind": "markdown", "value": hover.Contents}}
		}
	}
	return nil
}

func lspDefinitionLocations(doc lspOpenDocument, uri string, zeroBasedLine int, zeroBasedCharacter int) any {
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
	if col >= 0 && col+len(sym.Name) <= len(lineText) && lineText[col:col+len(sym.Name)] == sym.Name {
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

func lspReferenceLocations(doc lspOpenDocument, uri string, zeroBasedLine int, zeroBasedCharacter int, includeDeclaration bool) any {
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

func lspRenameWorkspaceEdit(doc lspOpenDocument, uri string, zeroBasedLine int, zeroBasedCharacter int, newName string) any {
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

func lspCodeActions(text string, uri string, requestDiagnostics []lspCodeActionDiagnostic, analysisDiagnostics []compiler.Diagnostic) []map[string]any {
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

func lspMissingEffectCodeAction(text string, uri string, diag lspCodeActionDiagnostic) (map[string]any, bool) {
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
