package compiler

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
	"tetra_language/compiler/internal/semantics"
)

type LSPSymbol struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Detail string `json:"detail,omitempty"`
}

type LSPHover struct {
	Name     string `json:"name"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Contents string `json:"contents"`
}

type LSPAnalysis struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Symbols     []LSPSymbol  `json:"symbols"`
	Hovers      []LSPHover   `json:"hovers"`
}

func AnalyzeLSPFile(path string) (LSPAnalysis, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return LSPAnalysis{}, err
	}
	out := AnalyzeLSPSource(raw, path)
	if len(out.Diagnostics) > 0 {
		return out, nil
	}
	file, err := frontend.ParseFile(raw, path)
	if err != nil {
		return out, nil
	}
	if len(file.Imports) == 0 {
		return out, nil
	}
	world, err := module.LoadWorld(path)
	if err != nil {
		out.Diagnostics = append(out.Diagnostics, DiagnosticFromError(err))
		return out, nil
	}
	if _, err := semantics.CheckWorldOpt(world, semantics.CheckOptions{RequireMain: false}); err != nil {
		out.Diagnostics = append(out.Diagnostics, DiagnosticFromError(err))
	}
	return out, nil
}

func AnalyzeLSPSource(src []byte, filename string) LSPAnalysis {
	out := LSPAnalysis{
		URI:         filename,
		Diagnostics: []Diagnostic{},
		Symbols:     []LSPSymbol{},
		Hovers:      []LSPHover{},
	}
	file, err := frontend.ParseFile(src, filename)
	if err != nil {
		out.Diagnostics = append(out.Diagnostics, DiagnosticFromError(err))
		return out
	}
	out.Symbols = collectLSPSymbols(file)
	out.Hovers = collectLSPHovers(file)
	if len(file.Imports) > 0 {
		return out
	}
	world := &module.World{
		EntryModule: "",
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{"": file},
	}
	if _, err := semantics.CheckWorldOpt(world, semantics.CheckOptions{RequireMain: false}); err != nil {
		out.Diagnostics = append(out.Diagnostics, DiagnosticFromError(err))
	}
	return out
}

func collectLSPSymbols(file *frontend.FileAST) []LSPSymbol {
	var symbols []LSPSymbol
	for _, st := range file.Structs {
		symbols = append(symbols, LSPSymbol{Name: st.Name, Kind: "struct", Line: st.At.Line, Column: st.At.Col})
	}
	for _, st := range file.States {
		symbols = append(symbols, LSPSymbol{Name: st.Name, Kind: "state", Line: st.At.Line, Column: st.At.Col, Detail: "state " + st.Name})
	}
	for _, view := range file.Views {
		symbols = append(symbols, LSPSymbol{Name: view.Name, Kind: "view", Line: view.At.Line, Column: view.At.Col, Detail: "view " + view.Name})
	}
	for _, en := range file.Enums {
		symbols = append(symbols, LSPSymbol{Name: en.Name, Kind: "enum", Line: en.At.Line, Column: en.At.Col})
	}
	for _, proto := range file.Protocols {
		symbols = append(symbols, LSPSymbol{Name: proto.Name, Kind: "protocol", Line: proto.At.Line, Column: proto.At.Col, Detail: "protocol " + proto.Name})
	}
	for _, glob := range file.Globals {
		symbols = append(symbols, LSPSymbol{Name: glob.Name, Kind: globalSymbolKind(glob), Line: glob.At.Line, Column: glob.At.Col, Detail: formatLSPGlobalDetail(glob)})
	}
	for _, impl := range file.Impls {
		symbols = append(symbols, LSPSymbol{Name: formatLSPImplName(impl), Kind: "impl", Line: impl.At.Line, Column: impl.At.Col, Detail: formatLSPImplDetail(impl)})
	}
	for _, fn := range file.Funcs {
		if fn.ExtensionOf != "" || fn.Synthetic {
			continue
		}
		symbols = append(symbols, LSPSymbol{Name: fn.Name, Kind: "function", Line: fn.Pos.Line, Column: fn.Pos.Col, Detail: formatLSPFuncDetail(fn)})
	}
	for _, ext := range file.Extensions {
		for _, fn := range ext.Methods {
			symbols = append(symbols, LSPSymbol{Name: fn.Name, Kind: "extension-method", Line: fn.Pos.Line, Column: fn.Pos.Col, Detail: formatLSPFuncDetail(fn)})
		}
	}
	for _, test := range file.Tests {
		symbols = append(symbols, LSPSymbol{Name: test.Name, Kind: "test", Line: test.At.Line, Column: test.At.Col})
	}
	return symbols
}

func collectLSPHovers(file *frontend.FileAST) []LSPHover {
	var hovers []LSPHover
	for _, fn := range file.Funcs {
		if fn.ExtensionOf != "" || fn.Synthetic {
			continue
		}
		hovers = append(hovers, LSPHover{
			Name:     fn.Name,
			Line:     fn.Pos.Line,
			Column:   fn.Pos.Col,
			Contents: formatLSPFuncDetail(fn),
		})
	}
	for _, ext := range file.Extensions {
		for _, fn := range ext.Methods {
			hovers = append(hovers, LSPHover{Name: fn.Name, Line: fn.Pos.Line, Column: fn.Pos.Col, Contents: formatLSPFuncDetail(fn)})
		}
	}
	for _, st := range file.Structs {
		hovers = append(hovers, LSPHover{Name: st.Name, Line: st.At.Line, Column: st.At.Col, Contents: "struct " + st.Name})
	}
	for _, st := range file.States {
		hovers = append(hovers, LSPHover{Name: st.Name, Line: st.At.Line, Column: st.At.Col, Contents: "state " + st.Name})
	}
	for _, view := range file.Views {
		hovers = append(hovers, LSPHover{Name: view.Name, Line: view.At.Line, Column: view.At.Col, Contents: "view " + view.Name})
	}
	for _, en := range file.Enums {
		hovers = append(hovers, LSPHover{Name: en.Name, Line: en.At.Line, Column: en.At.Col, Contents: "enum " + en.Name})
	}
	for _, proto := range file.Protocols {
		hovers = append(hovers, LSPHover{Name: proto.Name, Line: proto.At.Line, Column: proto.At.Col, Contents: "protocol " + proto.Name})
	}
	for _, glob := range file.Globals {
		hovers = append(hovers, LSPHover{Name: glob.Name, Line: glob.At.Line, Column: glob.At.Col, Contents: formatLSPGlobalDetail(glob)})
	}
	for _, impl := range file.Impls {
		hovers = append(hovers, LSPHover{Name: formatLSPImplName(impl), Line: impl.At.Line, Column: impl.At.Col, Contents: formatLSPImplDetail(impl)})
	}
	return hovers
}

func globalSymbolKind(glob *frontend.GlobalDecl) string {
	if glob.Mutable {
		return "var"
	}
	if glob.Const {
		return "const"
	}
	return "val"
}

func formatLSPGlobalDetail(glob *frontend.GlobalDecl) string {
	out := globalSymbolKind(glob) + " " + glob.Name
	if glob.Type.Name != "" || glob.Type.Elem != nil {
		out += ": " + formatLSPTypeRef(glob.Type)
	}
	return out
}

func formatLSPImplName(impl *frontend.ImplDecl) string {
	return formatLSPTypeRef(impl.Type) + ": " + formatLSPTypeRef(impl.Protocol)
}

func formatLSPImplDetail(impl *frontend.ImplDecl) string {
	return "impl " + formatLSPImplName(impl)
}

func formatLSPFuncDetail(fn *frontend.FuncDecl) string {
	params := make([]string, 0, len(fn.Params))
	for _, param := range fn.Params {
		typ := formatLSPTypeRef(param.Type)
		if param.Ownership != "" {
			typ = param.Ownership + " " + typ
		}
		params = append(params, param.Name+": "+typ)
	}
	prefix := "func"
	if fn.Async {
		prefix = "async func"
	}
	typeParams := ""
	if len(fn.TypeParams) > 0 {
		typeParams = "<" + strings.Join(fn.TypeParams, ", ") + ">"
	}
	detail := fmt.Sprintf("%s %s%s(%s) -> %s", prefix, fn.Name, typeParams, strings.Join(params, ", "), formatLSPTypeRef(fn.ReturnType))
	if fn.HasThrows {
		detail += " throws " + formatLSPTypeRef(fn.Throws)
	}
	if len(fn.Uses) > 0 {
		uses := append([]string(nil), fn.Uses...)
		sort.Strings(uses)
		detail += " uses " + strings.Join(uses, ", ")
	}
	return detail
}

func formatLSPTypeRef(ref frontend.TypeRef) string {
	switch ref.Kind {
	case frontend.TypeRefSlice:
		return "[]" + formatLSPTypeRef(*ref.Elem)
	case frontend.TypeRefArray:
		return fmt.Sprintf("[%d]%s", ref.Len, formatLSPTypeRef(*ref.Elem))
	case frontend.TypeRefOptional:
		return formatLSPTypeRef(*ref.Elem) + "?"
	default:
		return ref.Name
	}
}
