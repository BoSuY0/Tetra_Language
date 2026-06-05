package frontend

import (
	"bytes"
	"errors"
	"strings"
)

var errFlowTab = errors.New("tabs are not supported in Flow indentation")

type flowBridgeOptions struct {
	rewriteFuncKeyword bool
	rewriteLetKeyword  bool
	wrapIfWhileCond    bool
}

func canonicalizeFlowSyntax(src []byte, filename string) ([]byte, error) {
	return bridgeFlowSyntax(src, filename, flowBridgeOptions{})
}

// NormalizeFlowForMigration rewrites supported Flow syntax into migration
// compatibility form (fun/val keywords and wrapped control-flow conditions).
//
// This is not used by canonical parse paths; keep it for explicit migration
// tooling only.
func NormalizeFlowForMigration(src []byte, filename string) ([]byte, error) {
	return normalizeFlowSyntax(src, filename)
}

func normalizeFlowSyntax(src []byte, filename string) ([]byte, error) {
	return bridgeFlowSyntax(src, filename, flowBridgeOptions{
		rewriteFuncKeyword: true,
		rewriteLetKeyword:  true,
		wrapIfWhileCond:    true,
	})
}

func bridgeFlowSyntax(src []byte, filename string, opts flowBridgeOptions) ([]byte, error) {
	if !looksLikeFlowSyntax(src) {
		return src, nil
	}

	lines := strings.Split(string(src), "\n")
	var out bytes.Buffer
	type flowBlock struct {
		indent int
		kind   string
	}
	var blocks []flowBlock
	pendingBlockIndent := -1
	pendingBlockKind := ""
	for lineNo, line := range lines {
		trimmedRight := strings.TrimRight(line, " \t\r")
		content := strings.TrimSpace(trimmedRight)
		if content == "" {
			out.WriteByte('\n')
			continue
		}
		indent, col, err := flowIndent(line)
		if err != nil {
			return nil, diagnosticErrorf(Position{File: filename, Line: lineNo + 1, Col: col}, err.Error())
		}
		if strings.HasPrefix(content, "//") {
			out.WriteString(content)
			out.WriteByte('\n')
			continue
		}
		if pendingBlockIndent >= 0 {
			if !(isCaseBlockKind(pendingBlockKind) && indent == pendingBlockIndent && strings.HasPrefix(content, "case ")) && indent <= pendingBlockIndent {
				return nil, diagnosticErrorf(Position{File: filename, Line: lineNo + 1, Col: 1}, "expected indented block after ':'")
			}
			pendingBlockIndent = -1
			pendingBlockKind = ""
		}
		for len(blocks) > 0 {
			top := blocks[len(blocks)-1]
			if strings.HasPrefix(content, "case ") && isCaseBlockKind(top.kind) && indent == top.indent {
				break
			}
			if indent > top.indent {
				break
			}
			out.WriteString("} ")
			blocks = blocks[:len(blocks)-1]
		}

		content = flowRewriteLine(content, opts)
		if strings.HasSuffix(content, ":") {
			header := strings.TrimSpace(strings.TrimSuffix(content, ":"))
			content = flowRewriteBlockHeader(header, opts) + " {"
			kind := flowBlockKind(header)
			blocks = append(blocks, flowBlock{indent: indent, kind: kind})
			pendingBlockIndent = indent
			pendingBlockKind = kind
		}
		out.WriteString(strings.Repeat(" ", indent))
		out.WriteString(content)
		out.WriteByte('\n')
	}
	if pendingBlockIndent >= 0 {
		return nil, diagnosticErrorf(Position{File: filename, Line: len(lines), Col: 1}, "expected indented block after ':'")
	}
	for len(blocks) > 0 {
		out.WriteString("}\n")
		blocks = blocks[:len(blocks)-1]
	}
	return out.Bytes(), nil
}

func looksLikeFlowSyntax(src []byte) bool {
	for _, line := range strings.Split(string(src), "\n") {
		content := strings.TrimSpace(line)
		if content == "" || strings.HasPrefix(content, "//") {
			continue
		}
		header := strings.TrimPrefix(content, "pub ")
		if strings.HasPrefix(header, "func ") || strings.HasPrefix(header, "async func ") {
			return true
		}
		if strings.HasPrefix(header, "struct ") && strings.HasSuffix(header, ":") {
			return true
		}
		if strings.HasPrefix(header, "repr(C) struct ") && strings.HasSuffix(header, ":") {
			return true
		}
		if strings.HasPrefix(header, "enum ") && strings.HasSuffix(header, ":") {
			return true
		}
		if strings.HasPrefix(header, "extension ") && strings.HasSuffix(header, ":") {
			return true
		}
		if strings.HasPrefix(header, "protocol ") && strings.HasSuffix(header, ":") {
			return true
		}
		if strings.HasPrefix(header, "actor ") && strings.HasSuffix(header, ":") {
			return true
		}
		if strings.HasPrefix(header, "state ") && strings.HasSuffix(header, ":") {
			return true
		}
		if strings.HasPrefix(header, "view ") && strings.HasSuffix(header, ":") {
			return true
		}
		if strings.HasPrefix(header, "capsule ") && strings.HasSuffix(header, ":") {
			return true
		}
		if strings.HasPrefix(content, "command ") && strings.HasSuffix(content, ":") {
			return true
		}
		if strings.HasPrefix(content, "impl ") {
			return true
		}
		if strings.HasPrefix(content, "test ") && strings.HasSuffix(content, ":") {
			return true
		}
	}
	return false
}

func flowIndent(line string) (int, int, error) {
	indent := 0
	col := 1
	for _, r := range line {
		switch r {
		case ' ':
			indent++
		case '\t':
			return 0, col, errFlowTab
		default:
			return indent, col, nil
		}
		col++
	}
	return indent, col, nil
}

func flowRewriteLine(content string, opts flowBridgeOptions) string {
	if opts.rewriteFuncKeyword && strings.HasPrefix(content, "func ") {
		return "fun " + strings.TrimPrefix(content, "func ")
	}
	if opts.rewriteFuncKeyword && strings.HasPrefix(content, "async func ") {
		return "async fun " + strings.TrimPrefix(content, "async func ")
	}
	if opts.rewriteLetKeyword && strings.HasPrefix(content, "let ") {
		return "val " + strings.TrimPrefix(content, "let ")
	}
	return content
}

func flowRewriteBlockHeader(header string, opts flowBridgeOptions) string {
	if !opts.wrapIfWhileCond {
		return header
	}
	switch {
	case strings.HasPrefix(header, "if let "):
		return header
	case strings.HasPrefix(header, "else if let "):
		return header
	case strings.HasPrefix(header, "else if "):
		cond := strings.TrimSpace(strings.TrimPrefix(header, "else if "))
		if strings.HasPrefix(cond, "(") {
			return "else if " + cond
		}
		return "else if (" + cond + ")"
	case strings.HasPrefix(header, "if "):
		cond := strings.TrimSpace(strings.TrimPrefix(header, "if "))
		if strings.HasPrefix(cond, "(") {
			return "if " + cond
		}
		return "if (" + cond + ")"
	case strings.HasPrefix(header, "while "):
		cond := strings.TrimSpace(strings.TrimPrefix(header, "while "))
		if strings.HasPrefix(cond, "(") {
			return "while " + cond
		}
		return "while (" + cond + ")"
	case header == "else":
		return "else"
	default:
		return header
	}
}

func flowBlockKind(header string) string {
	header = strings.TrimPrefix(header, "pub ")
	switch {
	case strings.HasPrefix(header, "match "):
		return "match"
	case strings.Contains(header, " = match "):
		return "match"
	case strings.HasPrefix(header, "return match "):
		return "match"
	case strings.HasPrefix(header, "catch "):
		return "catch"
	case strings.Contains(header, " = catch "):
		return "catch"
	case strings.HasPrefix(header, "return catch "):
		return "catch"
	case strings.HasPrefix(header, "case "):
		return "case"
	default:
		return "block"
	}
}

func isCaseBlockKind(kind string) bool {
	return kind == "match" || kind == "catch"
}
