package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"tetra_language/compiler"
	ctarget "tetra_language/compiler/target"
)

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

type smokeCaseReport struct {
	Name         string `json:"name"`
	SrcPath      string `json:"src_path"`
	OutPath      string `json:"out_path"`
	ExpectedExit int    `json:"expected_exit"`
	ActualExit   *int   `json:"actual_exit,omitempty"`
	Ran          bool   `json:"ran"`
	Pass         bool   `json:"pass"`
	Error        string `json:"error,omitempty"`
}

type smokeReport struct {
	Timestamp    string            `json:"timestamp"`
	Target       string            `json:"target"`
	Host         string            `json:"host"`
	Version      string            `json:"version"`
	GitHead      string            `json:"git_head,omitempty"`
	IslandsDebug bool              `json:"islands_debug"`
	Total        int               `json:"total"`
	Passed       int               `json:"passed"`
	Failed       int               `json:"failed"`
	Cases        []smokeCaseReport `json:"cases"`
}

type smokeCase struct {
	name         string
	srcPath      string
	expectedExit int
	debugOnly    bool
}

type smokeListCase struct {
	Name         string `json:"name"`
	SrcPath      string `json:"src_path"`
	ExpectedExit int    `json:"expected_exit"`
	DebugOnly    bool   `json:"debug_only,omitempty"`
}

type smokeListReport struct {
	Total        int             `json:"total"`
	IslandsDebug bool            `json:"islands_debug"`
	Cases        []smokeListCase `json:"cases"`
}

const supportedTargetsHelp = "linux-x64, windows-x64, macos-x64"

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}

func runCLI(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "version":
		fmt.Fprintln(stdout, compiler.Version())
		return 0
	case "targets":
		return runTargets(args[1:], stdout, stderr)
	case "doctor":
		return runDoctor(args[1:], stdout, stderr)
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "build":
		return runBuild(args[1:], stdout, stderr)
	case "run":
		return runRun(args[1:], stdout, stderr)
	case "smoke":
		return runSmoke(args[1:], stdout, stderr)
	case "fmt":
		return runFmt(args[1:], stdout, stderr)
	case "test":
		return runTest(args[1:], stdout, stderr)
	case "doc":
		return runDoc(args[1:], stdout, stderr)
	case "clean":
		return runClean(args[1:], stdout, stderr)
	case "eco":
		return runEco(args[1:], stdout, stderr)
	case "lsp":
		return runLSP(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

type targetsReport struct {
	Supported []string `json:"supported"`
	Planned   []string `json:"planned"`
}

type doctorReport struct {
	Status string        `json:"status"`
	Checks []doctorCheck `json:"checks"`
}

type doctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func runTargets(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("targets", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "targets does not accept positional arguments")
		return 2
	}
	report := targetsReport{
		Supported: ctarget.SupportedTriples(),
		Planned:   ctarget.PlannedTriples(),
	}
	switch *format {
	case "text", "":
		fmt.Fprintln(stdout, "Supported targets:")
		for _, triple := range report.Supported {
			fmt.Fprintf(stdout, "  %s\n", triple)
		}
		fmt.Fprintln(stdout, "Planned targets:")
		for _, triple := range report.Planned {
			fmt.Fprintf(stdout, "  %s\n", triple)
		}
		return 0
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func runDoctor(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "doctor does not accept positional arguments")
		return 2
	}
	report := buildDoctorReport()
	switch *format {
	case "text", "":
		fmt.Fprintf(stdout, "Tetra doctor: %s\n", report.Status)
		for _, check := range report.Checks {
			if check.Detail == "" {
				fmt.Fprintf(stdout, "  %s: %s\n", check.Name, check.Status)
			} else {
				fmt.Fprintf(stdout, "  %s: %s (%s)\n", check.Name, check.Status, check.Detail)
			}
		}
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
	if report.Status != "pass" {
		return 1
	}
	return 0
}

func buildDoctorReport() doctorReport {
	checks := []doctorCheck{
		passCheck("version", compiler.Version()),
		passCheck("supported targets", strings.Join(ctarget.SupportedTriples(), ", ")),
		passCheck("planned targets", strings.Join(ctarget.PlannedTriples(), ", ")),
	}
	root, err := findRepoRoot()
	if err != nil {
		checks = append(checks, failCheck("repo root", err.Error()))
		return doctorReport{Status: doctorStatus(checks), Checks: checks}
	}
	checks = append(checks,
		passCheck("repo root", root),
		pathCheck(root, "__rt/actors_sysv.tetra"),
		pathCheck(root, "__rt/actors_win64.tetra"),
		pathCheck(root, "compiler/selfhostrt/actors_sysv.tetra"),
		pathCheck(root, "compiler/selfhostrt/actors_win64.tetra"),
		pathCheck(root, "examples/flow_hello.tetra"),
		pathCheck(root, "docs/generated/manifest.json"),
		manifestVersionCheck(root),
		manifestSurfaceCheck(root),
		smokeSourcesCheck(root),
		runtimeExportsCheck(root),
	)
	return doctorReport{Status: doctorStatus(checks), Checks: checks}
}

func manifestVersionCheck(root string) doctorCheck {
	path := filepath.Join(root, filepath.FromSlash("docs/generated/manifest.json"))
	raw, err := os.ReadFile(path)
	if err != nil {
		return failCheck("docs manifest version", err.Error())
	}
	var manifest struct {
		CompilerVersion string `json:"compiler_version"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return failCheck("docs manifest version", err.Error())
	}
	if manifest.CompilerVersion != compiler.Version() {
		return failCheck("docs manifest version", fmt.Sprintf("got %s want %s", manifest.CompilerVersion, compiler.Version()))
	}
	return passCheck("docs manifest version", manifest.CompilerVersion)
}

func manifestSurfaceCheck(root string) doctorCheck {
	path := filepath.Join(root, filepath.FromSlash("docs/generated/manifest.json"))
	raw, err := os.ReadFile(path)
	if err != nil {
		return failCheck("docs manifest surface", err.Error())
	}
	var manifest struct {
		Targets []struct {
			Triple string `json:"triple"`
		} `json:"targets"`
		RuntimeABI struct {
			ActorsSupportedTargets []string `json:"actors_supported_targets"`
			ActorsRequiredSymbols  []string `json:"actors_required_symbols"`
		} `json:"runtime_abi"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return failCheck("docs manifest surface", err.Error())
	}
	var targetTriples []string
	for _, target := range manifest.Targets {
		targetTriples = append(targetTriples, target.Triple)
	}
	if !sameStringSet(targetTriples, ctarget.SupportedTriples()) {
		return failCheck("docs manifest surface", fmt.Sprintf("targets got %s want %s", strings.Join(sortedDoctorStrings(targetTriples), ", "), strings.Join(sortedDoctorStrings(ctarget.SupportedTriples()), ", ")))
	}
	if !sameStringSet(manifest.RuntimeABI.ActorsSupportedTargets, ctarget.SupportedTriples()) {
		return failCheck("docs manifest surface", fmt.Sprintf("actors targets got %s want %s", strings.Join(sortedDoctorStrings(manifest.RuntimeABI.ActorsSupportedTargets), ", "), strings.Join(sortedDoctorStrings(ctarget.SupportedTriples()), ", ")))
	}
	if !sameStringSet(manifest.RuntimeABI.ActorsRequiredSymbols, actorRuntimeSymbols()) {
		return failCheck("docs manifest surface", fmt.Sprintf("runtime symbols got %s want %s", strings.Join(sortedDoctorStrings(manifest.RuntimeABI.ActorsRequiredSymbols), ", "), strings.Join(sortedDoctorStrings(actorRuntimeSymbols()), ", ")))
	}
	return passCheck("docs manifest surface", fmt.Sprintf("%d targets, %d runtime symbols", len(targetTriples), len(actorRuntimeSymbols())))
}

func smokeSourcesCheck(root string) doctorCheck {
	seenNames := map[string]bool{}
	seenSources := map[string]bool{}
	var missing []string
	var duplicates []string
	cases := smokeCases(true)
	for _, c := range cases {
		if seenNames[c.name] {
			duplicates = append(duplicates, "name:"+c.name)
		}
		seenNames[c.name] = true
		if seenSources[c.srcPath] {
			duplicates = append(duplicates, "src:"+c.srcPath)
		}
		seenSources[c.srcPath] = true
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(c.srcPath))); err != nil {
			missing = append(missing, c.srcPath)
		}
	}
	if len(missing) > 0 || len(duplicates) > 0 {
		sort.Strings(missing)
		sort.Strings(duplicates)
		parts := []string{}
		if len(missing) > 0 {
			parts = append(parts, "missing "+strings.Join(missing, ", "))
		}
		if len(duplicates) > 0 {
			parts = append(parts, "duplicate "+strings.Join(duplicates, ", "))
		}
		return failCheck("smoke sources", strings.Join(parts, "; "))
	}
	return passCheck("smoke sources", fmt.Sprintf("%d sources", len(cases)))
}

func runtimeExportsCheck(root string) doctorCheck {
	paths := []string{
		"__rt/actors_sysv.tetra",
		"__rt/actors_win64.tetra",
		"compiler/selfhostrt/actors_sysv.tetra",
		"compiler/selfhostrt/actors_win64.tetra",
	}
	required := actorRuntimeSymbols()
	var missing []string
	for _, rel := range paths {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			missing = append(missing, rel+": "+err.Error())
			continue
		}
		text := string(raw)
		for _, symbol := range required {
			if !strings.Contains(text, "@export("+strconv.Quote(symbol)+")") {
				missing = append(missing, rel+":"+symbol)
			}
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return failCheck("runtime exports", strings.Join(missing, ", "))
	}
	return passCheck("runtime exports", fmt.Sprintf("%d files, %d symbols", len(paths), len(required)))
}

func actorRuntimeSymbols() []string {
	return []string{
		"__tetra_entry",
		"__tetra_actor_spawn",
		"__tetra_actor_send",
		"__tetra_actor_recv",
		"__tetra_actor_self",
		"__tetra_actor_sender",
	}
}

func sameStringSet(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[string]int{}
	for _, s := range a {
		seen[s]++
	}
	for _, s := range b {
		seen[s]--
		if seen[s] < 0 {
			return false
		}
	}
	for _, n := range seen {
		if n != 0 {
			return false
		}
	}
	return true
}

func sortedDoctorStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func pathCheck(root string, rel string) doctorCheck {
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
		return failCheck(rel, err.Error())
	}
	return passCheck(rel, "found")
}

func passCheck(name string, detail string) doctorCheck {
	return doctorCheck{Name: name, Status: "pass", Detail: detail}
}

func failCheck(name string, detail string) doctorCheck {
	return doctorCheck{Name: name, Status: "fail", Detail: detail}
}

func doctorStatus(checks []doctorCheck) string {
	for _, check := range checks {
		if check.Status != "pass" {
			return "fail"
		}
	}
	return "pass"
}

func runDoc(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("doc", flag.ContinueOnError)
	fs.SetOutput(stderr)
	outPath := fs.String("o", "", "output markdown path; stdout when empty")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text or json")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	docs, err := compiler.GenerateAPIDocs(fs.Args())
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if *outPath == "" {
		fmt.Fprint(stdout, string(docs))
		return 0
	}
	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if err := os.WriteFile(*outPath, docs, 0o644); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	fmt.Fprintf(stdout, "Wrote docs: %s\n", *outPath)
	return 0
}

func runCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text or json")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	input := "main.tetra"
	if fs.NArg() > 0 {
		input = fs.Arg(0)
	}
	if fs.NArg() > 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "check accepts at most one input path")
		return 2
	}
	world, err := compiler.LoadWorld(input)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	fmt.Fprintf(stdout, "Checked: %s\n", input)
	return 0
}

func runLSP(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("lsp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	smokePath := fs.String("stdio-smoke", "", "analyze one .tetra file and print LSP-basic JSON")
	stdio := fs.Bool("stdio", false, "run LSP-basic JSON-RPC over stdio")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "lsp does not accept positional arguments")
		return 2
	}
	if *stdio {
		return runLSPStdio(os.Stdin, stdout, stderr)
	}
	if *smokePath == "" {
		fmt.Fprintln(stderr, "lsp requires --stdio or --stdio-smoke <file>")
		return 2
	}
	analysis, err := compiler.AnalyzeLSPFile(*smokePath)
	if err != nil {
		writeDiagnostic(stderr, "json", err)
		return 1
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

type lspRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
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

func runLSPStdio(stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	reader := bufio.NewReader(stdin)
	openDocs := map[string]compiler.LSPAnalysis{}
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
			fmt.Fprintln(stderr, err)
			return 1
		}
		switch req.Method {
		case "initialize":
			if req.ID != nil {
				result := map[string]any{
					"capabilities": map[string]any{
						"textDocumentSync":       1,
						"documentSymbolProvider": true,
						"hoverProvider":          true,
					},
				}
				if err := writeLSPResponse(stdout, *req.ID, result); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		case "shutdown":
			shutdown = true
			if req.ID != nil {
				if err := writeLSPResponse(stdout, *req.ID, nil); err != nil {
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
			if err := json.Unmarshal(req.Params, &params); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			analysis := compiler.AnalyzeLSPSource([]byte(params.TextDocument.Text), params.TextDocument.URI)
			openDocs[params.TextDocument.URI] = analysis
			if err := writeLSPNotification(stdout, "textDocument/publishDiagnostics", map[string]any{
				"uri":         params.TextDocument.URI,
				"diagnostics": lspDiagnostics(analysis.Diagnostics),
			}); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
		case "textDocument/didChange":
			var params lspDidChangeParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			if len(params.ContentChanges) == 0 {
				continue
			}
			text := params.ContentChanges[len(params.ContentChanges)-1].Text
			analysis := compiler.AnalyzeLSPSource([]byte(text), params.TextDocument.URI)
			openDocs[params.TextDocument.URI] = analysis
			if err := writeLSPNotification(stdout, "textDocument/publishDiagnostics", map[string]any{
				"uri":         params.TextDocument.URI,
				"diagnostics": lspDiagnostics(analysis.Diagnostics),
			}); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
		case "textDocument/didClose":
			var params lspDidCloseParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
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
			if err := json.Unmarshal(req.Params, &params); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			if req.ID != nil {
				if err := writeLSPResponse(stdout, *req.ID, lspDocumentSymbols(openDocs[params.TextDocument.URI])); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		case "textDocument/hover":
			var params lspHoverParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			if req.ID != nil {
				if err := writeLSPResponse(stdout, *req.ID, lspHoverAt(openDocs[params.TextDocument.URI], params.Position.Line)); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		default:
			if req.ID != nil {
				if err := writeLSPResponse(stdout, *req.ID, nil); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}
		}
	}
}

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

func writeLSPResponse(w io.Writer, id int, result any) error {
	return writeLSPMessage(w, map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

func writeLSPNotification(w io.Writer, method string, params any) error {
	return writeLSPMessage(w, map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

func writeLSPMessage(w io.Writer, msg any) error {
	raw, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "Content-Length: %d\r\n\r\n%s", len(raw), raw)
	return err
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

func runBuild(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", defaultTarget(), "target triple ("+supportedTargetsHelp+")")
	out := fs.String("o", "", "output path")
	islandsDebug := fs.Bool("islands-debug", false, "enable islands debug runtime checks")
	emit := fs.String("emit", "exe", "emit mode: exe, object, or library")
	runtimeMode := fs.String("runtime", "auto", "actors runtime: auto, selfhost, or builtin")
	runtimeObject := fs.String("runtime-object", "", "actors runtime object override")
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text or json")
	var linkObjects multiFlag
	fs.Var(&linkObjects, "link-object", "extra TOBJ object to link")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}

	input := "main.tetra"
	if fs.NArg() > 0 {
		input = fs.Arg(0)
	}
	if fs.NArg() > 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "build accepts at most one input path")
		return 2
	}

	tgt, err := ctarget.Parse(*target)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 2
	}
	output := *out
	if output == "" {
		output = defaultOutput(tgt, *emit)
	}

	opt, err := buildOptions(*emit, *runtimeMode, *islandsDebug, *runtimeObject, []string(linkObjects), *jobs)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 2
	}
	if _, err := compiler.BuildFileWithStatsOpt(input, output, tgt.Triple, opt); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	fmt.Fprintf(stdout, "Built: %s\n", output)
	return 0
}

func runRun(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", defaultTarget(), "target triple ("+supportedTargetsHelp+")")
	out := fs.String("o", "", "output path")
	islandsDebug := fs.Bool("islands-debug", false, "enable islands debug runtime checks")
	runtimeMode := fs.String("runtime", "auto", "actors runtime: auto, selfhost, or builtin")
	runtimeObject := fs.String("runtime-object", "", "actors runtime object override")
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text or json")
	var linkObjects multiFlag
	fs.Var(&linkObjects, "link-object", "extra TOBJ object to link")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	input := "main.tetra"
	if fs.NArg() > 0 {
		input = fs.Arg(0)
	}
	if fs.NArg() > 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "run accepts at most one input path")
		return 2
	}
	tgt, err := ctarget.Parse(*target)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 2
	}
	host, ok := hostTarget()
	if !ok || host != tgt.Triple {
		writeDiagnostic(stderr, *diagnostics, fmt.Errorf("cannot run target %s on host %s/%s", tgt.Triple, runtime.GOOS, runtime.GOARCH))
		return 2
	}
	output := *out
	tmpDir := ""
	if output == "" {
		tmpDir, err = os.MkdirTemp("", "tetra-run-*")
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		defer os.RemoveAll(tmpDir)
		output = filepath.Join(tmpDir, defaultOutput(tgt, "exe"))
	}
	opt, err := buildOptions("exe", *runtimeMode, *islandsDebug, *runtimeObject, []string(linkObjects), *jobs)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 2
	}
	if _, err := compiler.BuildFileWithStatsOpt(input, output, tgt.Triple, opt); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	return execProgram(output, stdout, stderr)
}

func runFmt(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("fmt", flag.ContinueOnError)
	fs.SetOutput(stderr)
	check := fs.Bool("check", false, "check whether files are formatted")
	write := fs.Bool("write", false, "rewrite files in place")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text or json")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	if *check && *write {
		writeValidationDiagnostic(stderr, *diagnostics, "fmt accepts only one of --check or --write")
		return 2
	}
	paths := fs.Args()
	if len(paths) == 0 {
		writeValidationDiagnostic(stderr, *diagnostics, "fmt requires at least one path")
		return 2
	}
	files, err := collectTetraFiles(paths)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if !*check && !*write && len(files) != 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "fmt stdout mode accepts exactly one file")
		return 2
	}
	dirty := false
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		formatted, err := compiler.FormatSource(raw, path)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		if *check {
			if string(raw) != string(formatted) {
				dirty = true
				if *diagnostics == "json" {
					writeDiagnosticObject(stderr, compiler.Diagnostic{
						Code:     "TETRA_FMT002",
						Message:  "not formatted",
						File:     path,
						Severity: "error",
						Hint:     "Run tetra fmt --write to update the file.",
					})
				} else {
					fmt.Fprintf(stderr, "%s: not formatted\n", path)
				}
			}
			continue
		}
		if *write {
			if string(raw) != string(formatted) {
				if err := os.WriteFile(path, formatted, 0o644); err != nil {
					writeDiagnostic(stderr, *diagnostics, err)
					return 1
				}
			}
			continue
		}
		fmt.Fprint(stdout, string(formatted))
	}
	if dirty {
		return 1
	}
	return 0
}

func runTest(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", defaultTarget(), "target triple ("+supportedTargetsHelp+")")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text or json")
	reportFormat := fs.String("report", "text", "report format: text or json")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	if *reportFormat != "text" && *reportFormat != "json" {
		writeValidationDiagnostic(stderr, *diagnostics, "unsupported --report format")
		return 2
	}
	paths := fs.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}
	tgt, err := ctarget.Parse(*target)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 2
	}
	host, ok := hostTarget()
	if !ok || host != tgt.Triple {
		writeDiagnostic(stderr, *diagnostics, fmt.Errorf("cannot run tests for target %s on host %s/%s", tgt.Triple, runtime.GOOS, runtime.GOARCH))
		return 2
	}
	files, err := collectTetraFiles(paths)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	tmpDir, err := os.MkdirTemp("", "tetra-test-*")
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	defer os.RemoveAll(tmpDir)
	total := 0
	passed := 0
	var results []compiler.TestRunnerResult
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		runners, err := compiler.TestRunnerSources(raw, file)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		for i, runner := range runners {
			total++
			start := time.Now()
			srcPath := filepath.Join(tmpDir, fmt.Sprintf("test_%d.tetra", total))
			outPath := filepath.Join(tmpDir, fmt.Sprintf("test_%d%s", total, tgt.ExeExt))
			if err := os.WriteFile(srcPath, runner.Source, 0o644); err != nil {
				writeDiagnostic(stderr, *diagnostics, err)
				return 1
			}
			if err := compiler.BuildFile(srcPath, outPath, tgt.Triple); err != nil {
				writeDiagnostic(stderr, *diagnostics, err)
				return 1
			}
			code := execProgram(outPath, io.Discard, io.Discard)
			name := runner.Name
			if name == "" {
				name = fmt.Sprintf("%s#%d", file, i+1)
			}
			result := runner.ResultWithDuration(code, nil, elapsedMillis(time.Since(start)))
			results = append(results, result)
			if code == 0 {
				passed++
				if *reportFormat == "text" {
					fmt.Fprintf(stdout, "PASS %s\n", name)
				}
			} else {
				if *reportFormat == "text" {
					if result.Error != "" {
						fmt.Fprintf(stdout, "FAIL %s (%s)\n", name, result.Error)
					} else {
						fmt.Fprintf(stdout, "FAIL %s\n", name)
					}
				}
			}
		}
	}
	if *reportFormat == "json" {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(compiler.NewTestRunnerReport(results)); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	} else {
		fmt.Fprintf(stdout, "Tetra tests: %d/%d passed\n", passed, total)
	}
	if passed != total {
		return 1
	}
	return 0
}

func elapsedMillis(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	ms := d.Milliseconds()
	if ms == 0 {
		return 1
	}
	return ms
}

func runSmoke(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("smoke", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", defaultTarget(), "target triple ("+supportedTargetsHelp+")")
	runBuilt := fs.Bool("run", true, "run built binaries when host matches target")
	reportPath := fs.String("report", "", "write JSON smoke report")
	listCases := fs.Bool("list", false, "list smoke cases without building")
	listFormat := fs.String("format", "text", "smoke list format: text or json")
	islandsDebug := fs.Bool("islands-debug", false, "enable islands debug runtime checks")
	runtimeMode := fs.String("runtime", "auto", "actors runtime: auto, selfhost, or builtin")
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "smoke does not accept positional arguments")
		return 2
	}
	if *listCases {
		return writeSmokeList(stdout, stderr, smokeCases(*islandsDebug), *islandsDebug, *listFormat)
	}
	if *listFormat != "text" {
		fmt.Fprintln(stderr, "--format is only supported with --list")
		return 2
	}
	tgt, err := ctarget.Parse(*target)
	if err != nil {
		fmt.Fprintln(stderr, err)
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

	host := ""
	hostTriple, hostOK := hostTarget()
	if hostOK {
		host = hostTriple
	}
	shouldRun := *runBuilt && hostOK && hostTriple == tgt.Triple
	opt, err := buildOptions("exe", *runtimeMode, *islandsDebug, "", nil, *jobs)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	report := smokeReport{
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Target:       tgt.Triple,
		Host:         host,
		Version:      compiler.Version(),
		GitHead:      gitHead(repoRoot),
		IslandsDebug: *islandsDebug,
	}
	for _, c := range smokeCases(*islandsDebug) {
		outPath := filepath.Join(tmpDir, c.name+tgt.ExeExt)
		srcAbs := filepath.Join(repoRoot, filepath.FromSlash(c.srcPath))
		caseReport := smokeCaseReport{
			Name:         c.name,
			SrcPath:      c.srcPath,
			OutPath:      outPath,
			ExpectedExit: c.expectedExit,
		}
		if _, err := compiler.BuildFileWithStatsOpt(srcAbs, outPath, tgt.Triple, opt); err != nil {
			caseReport.Error = "build: " + err.Error()
			report.Cases = append(report.Cases, caseReport)
			continue
		}
		if shouldRun {
			actual := execProgram(outPath, io.Discard, io.Discard)
			caseReport.ActualExit = &actual
			caseReport.Ran = true
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
		if err := writeJSON(*reportPath, report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	if passed != len(report.Cases) {
		return 1
	}
	return 0
}

func writeSmokeList(stdout io.Writer, stderr io.Writer, cases []smokeCase, islandsDebug bool, format string) int {
	report := smokeListReport{
		Total:        len(cases),
		IslandsDebug: islandsDebug,
		Cases:        make([]smokeListCase, 0, len(cases)),
	}
	for _, c := range cases {
		report.Cases = append(report.Cases, smokeListCase{
			Name:         c.name,
			SrcPath:      c.srcPath,
			ExpectedExit: c.expectedExit,
			DebugOnly:    c.debugOnly,
		})
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
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func runClean(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("clean", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "clean does not accept positional arguments")
		return 2
	}
	for _, path := range []string{".tetra_cache", "tetra_cache"} {
		if err := os.RemoveAll(path); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	fmt.Fprintln(stdout, "Cleaned Tetra cache")
	return 0
}

func buildOptions(emit string, runtimeMode string, islandsDebug bool, runtimeObject string, linkObjects []string, jobs int) (compiler.BuildOptions, error) {
	opt := compiler.BuildOptions{
		Jobs:              jobs,
		IslandsDebug:      islandsDebug,
		RuntimeObjectPath: runtimeObject,
		LinkObjectPaths:   linkObjects,
	}
	switch emit {
	case "", "exe":
		opt.Emit = compiler.EmitExe
	case "object":
		opt.Emit = compiler.EmitObject
	case "library":
		opt.Emit = compiler.EmitLibrary
	default:
		return opt, fmt.Errorf("unsupported --emit %q", emit)
	}
	switch runtimeMode {
	case "", "auto":
		opt.Runtime = compiler.RuntimeAuto
	case "selfhost":
		opt.Runtime = compiler.RuntimeSelfHost
	case "builtin":
		opt.Runtime = compiler.RuntimeBuiltin
	default:
		return opt, fmt.Errorf("unsupported --runtime %q", runtimeMode)
	}
	return opt, nil
}

func smokeCases(islandsDebug bool) []smokeCase {
	cases := []smokeCase{
		{name: "islands_hello", srcPath: "examples/islands_hello.tetra", expectedExit: 0},
		{name: "islands_i32", srcPath: "examples/islands_i32.tetra", expectedExit: 55},
		{name: "islands_overflow", srcPath: "examples/islands_overflow.tetra", expectedExit: 1},
		{name: "mmio_smoke", srcPath: "examples/mmio_smoke.tetra", expectedExit: 123},
		{name: "cap_mem_smoke", srcPath: "examples/cap_mem_smoke.tetra", expectedExit: 77},
		{name: "memset_smoke", srcPath: "examples/memset_smoke.tetra", expectedExit: 88},
		{name: "actors_pingpong", srcPath: "examples/actors_pingpong.tetra", expectedExit: 0},
		{name: "flow_hello", srcPath: "examples/flow_hello.tetra", expectedExit: 0},
		{name: "flow_struct_smoke", srcPath: "examples/flow_struct_smoke.tetra", expectedExit: 42},
		{name: "flow_islands_smoke", srcPath: "examples/flow_islands_smoke.tetra", expectedExit: 0},
		{name: "flow_unsafe_cap_mem_smoke", srcPath: "examples/flow_unsafe_cap_mem_smoke.tetra", expectedExit: 42},
		{name: "bool_smoke", srcPath: "examples/bool_smoke.tetra", expectedExit: 42},
		{name: "for_range_smoke", srcPath: "examples/for_range_smoke.tetra", expectedExit: 55},
		{name: "for_collection_smoke", srcPath: "examples/for_collection_smoke.tetra", expectedExit: 42},
		{name: "for_collection_u8_smoke", srcPath: "examples/for_collection_u8_smoke.tetra", expectedExit: 42},
		{name: "loop_control_smoke", srcPath: "examples/loop_control_smoke.tetra", expectedExit: 42},
		{name: "unary_not_smoke", srcPath: "examples/unary_not_smoke.tetra", expectedExit: 42},
		{name: "const_smoke", srcPath: "examples/const_smoke.tetra", expectedExit: 42},
		{name: "const_bool_smoke", srcPath: "examples/const_bool_smoke.tetra", expectedExit: 42},
		{name: "local_const_smoke", srcPath: "examples/local_const_smoke.tetra", expectedExit: 42},
		{name: "compound_assignment_smoke", srcPath: "examples/compound_assignment_smoke.tetra", expectedExit: 42},
		{name: "else_if_smoke", srcPath: "examples/else_if_smoke.tetra", expectedExit: 42},
		{name: "enum_match_smoke", srcPath: "examples/enum_match_smoke.tetra", expectedExit: 42},
		{name: "enum_exhaustive_match_smoke", srcPath: "examples/enum_exhaustive_match_smoke.tetra", expectedExit: 42},
		{name: "effects_io_smoke", srcPath: "examples/effects_io_smoke.tetra", expectedExit: 0},
		{name: "effects_mem_smoke", srcPath: "examples/effects_mem_smoke.tetra", expectedExit: 17},
		{name: "effects_actors_smoke", srcPath: "examples/effects_actors_smoke.tetra", expectedExit: 0},
		{name: "optional_smoke", srcPath: "examples/optional_smoke.tetra", expectedExit: 42},
		{name: "optional_match_smoke", srcPath: "examples/optional_match_smoke.tetra", expectedExit: 42},
		{name: "optional_match_some_smoke", srcPath: "examples/optional_match_some_smoke.tetra", expectedExit: 42},
		{name: "ownership_smoke", srcPath: "examples/ownership_smoke.tetra", expectedExit: 42},
		{name: "typed_errors_smoke", srcPath: "examples/typed_errors_smoke.tetra", expectedExit: 42},
		{name: "async_smoke", srcPath: "examples/async_smoke.tetra", expectedExit: 42},
		{name: "task_smoke", srcPath: "examples/task_smoke.tetra", expectedExit: 42},
		{name: "core_math_smoke", srcPath: "examples/core_math_smoke.tetra", expectedExit: 42},
		{name: "core_memory_smoke", srcPath: "examples/core_memory_smoke.tetra", expectedExit: 42},
		{name: "extension_smoke", srcPath: "examples/extension_smoke.tetra", expectedExit: 42},
		{name: "generic_smoke", srcPath: "examples/generic_smoke.tetra", expectedExit: 42},
		{name: "protocol_impl_smoke", srcPath: "examples/protocol_impl_smoke.tetra", expectedExit: 42},
	}
	if islandsDebug {
		cases = append(cases, smokeCase{name: "islands_double_free", srcPath: "examples/islands_double_free.tetra", expectedExit: 2, debugOnly: true})
	}
	return cases
}

func defaultTarget() string {
	if target, ok := hostTarget(); ok {
		return target
	}
	return "linux-x64"
}

func hostTarget() (string, bool) {
	tgt, ok := ctarget.Host()
	if !ok {
		return "", false
	}
	return tgt.Triple, true
}

func defaultOutput(tgt ctarget.Target, emit string) string {
	switch emit {
	case "object", "library":
		return "app.tobj"
	default:
		return "app" + tgt.ExeExt
	}
}

func execProgram(path string, stdout io.Writer, stderr io.Writer) int {
	cmd := exec.Command(path)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode()
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if fileExists(filepath.Join(dir, "go.work")) && fileExists(filepath.Join(dir, "examples")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repository root")
		}
		dir = parent
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func gitHead(root string) string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func collectTetraFiles(paths []string) ([]string, error) {
	seen := map[string]struct{}{}
	var files []string
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if strings.HasSuffix(path, ".tetra") {
				if _, ok := seen[path]; !ok {
					seen[path] = struct{}{}
					files = append(files, path)
				}
			}
			continue
		}
		err = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if strings.HasPrefix(d.Name(), ".") && p != path {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(p, ".tetra") {
				if _, ok := seen[p]; !ok {
					seen[p] = struct{}{}
					files = append(files, p)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(files)
	return files, nil
}

func writeDiagnostic(w io.Writer, mode string, err error) {
	if mode == "json" {
		writeDiagnosticObject(w, compiler.DiagnosticFromError(err))
		return
	}
	fmt.Fprintln(w, err)
}

func writeValidationDiagnostic(w io.Writer, mode string, message string) {
	writeDiagnostic(w, mode, fmt.Errorf("%s", message))
}

func validateDiagnosticsMode(w io.Writer, mode string) bool {
	if mode == "text" || mode == "json" {
		return true
	}
	fmt.Fprintln(w, "unsupported --diagnostics format")
	return false
}

func writeDiagnosticObject(w io.Writer, diag compiler.Diagnostic) {
	raw, err := json.Marshal(diag)
	if err == nil {
		fmt.Fprintln(w, string(raw))
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: tetra <version|targets|doctor|check|build|run|smoke|fmt|test|doc|clean|eco|lsp> [options]")
}
