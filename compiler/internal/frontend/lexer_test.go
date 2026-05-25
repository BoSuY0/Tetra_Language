package frontend

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func collectTokens(src string) ([]token, error) {
	return collectTokensFrom(src, "test")
}

func collectTokensFrom(src string, file string) ([]token, error) {
	l := newLexer([]byte(src), file)
	var tokens []token
	for {
		tok, err := l.nextToken()
		if err != nil {
			return tokens, err
		}
		tokens = append(tokens, tok)
		if tok.typ == TokenEOF {
			break
		}
	}
	return tokens, nil
}

func TestLexTokenTypes(t *testing.T) {
	tests := []struct {
		src  string
		want []TokenType
	}{
		{"", []TokenType{TokenEOF}},
		{"42", []TokenType{TokenNumber, TokenEOF}},
		{"foo", []TokenType{TokenIdent, TokenEOF}},
		{"fn", []TokenType{TokenFn, TokenEOF}},
		{"fun", []TokenType{TokenFun, TokenEOF}},
		{"let", []TokenType{TokenLet, TokenEOF}},
		{"var", []TokenType{TokenVar, TokenEOF}},
		{"val", []TokenType{TokenVal, TokenEOF}},
		{"module", []TokenType{TokenModule, TokenEOF}},
		{"import", []TokenType{TokenImport, TokenEOF}},
		{"pub", []TokenType{TokenPub, TokenEOF}},
		{"as", []TokenType{TokenAs, TokenEOF}},
		{"uses", []TokenType{TokenUses, TokenEOF}},
		{"struct", []TokenType{TokenStruct, TokenEOF}},
		{"const", []TokenType{TokenConst, TokenEOF}},
		{"if", []TokenType{TokenIf, TokenEOF}},
		{"else", []TokenType{TokenElse, TokenEOF}},
		{"while", []TokenType{TokenWhile, TokenEOF}},
		{"for", []TokenType{TokenFor, TokenEOF}},
		{"in", []TokenType{TokenIn, TokenEOF}},
		{"enum", []TokenType{TokenEnum, TokenEOF}},
		{"case", []TokenType{TokenCase, TokenEOF}},
		{"match", []TokenType{TokenMatch, TokenEOF}},
		{"true", []TokenType{TokenTrue, TokenEOF}},
		{"false", []TokenType{TokenFalse, TokenEOF}},
		{"none", []TokenType{TokenNone, TokenEOF}},
		{"throws", []TokenType{TokenThrows, TokenEOF}},
		{"try", []TokenType{TokenTry, TokenEOF}},
		{"throw", []TokenType{TokenThrow, TokenEOF}},
		{"catch", []TokenType{TokenCatch, TokenEOF}},
		{"async", []TokenType{TokenAsync, TokenEOF}},
		{"await", []TokenType{TokenAwait, TokenEOF}},
		{"break", []TokenType{TokenBreak, TokenEOF}},
		{"continue", []TokenType{TokenContinue, TokenEOF}},
		{"return", []TokenType{TokenReturn, TokenEOF}},
		{"print", []TokenType{TokenPrint, TokenEOF}},
		{"free", []TokenType{TokenFree, TokenEOF}},
		{"unsafe", []TokenType{TokenUnsafe, TokenEOF}},
		{"test", []TokenType{TokenTest, TokenEOF}},
		{"expect", []TokenType{TokenExpect, TokenEOF}},
		{"+", []TokenType{TokenPlus, TokenEOF}},
		{"-", []TokenType{TokenMinus, TokenEOF}},
		{"*", []TokenType{TokenStar, TokenEOF}},
		{"/", []TokenType{TokenSlash, TokenEOF}},
		{"%", []TokenType{TokenPercent, TokenEOF}},
		{"<", []TokenType{TokenLess, TokenEOF}},
		{">", []TokenType{TokenGreater, TokenEOF}},
		{">=", []TokenType{TokenGreaterEq, TokenEOF}},
		{"<=", []TokenType{TokenLessEq, TokenEOF}},
		{"!=", []TokenType{TokenBangEq, TokenEOF}},
		{"!", []TokenType{TokenBang, TokenEOF}},
		{"==", []TokenType{TokenEqEq, TokenEOF}},
		{"&&", []TokenType{TokenAmpAmp, TokenEOF}},
		{"||", []TokenType{TokenPipePipe, TokenEOF}},
		{"->", []TokenType{TokenArrow, TokenEOF}},
		{"=", []TokenType{TokenAssign, TokenEOF}},
		{":", []TokenType{TokenColon, TokenEOF}},
		{",", []TokenType{TokenComma, TokenEOF}},
		{".", []TokenType{TokenDot, TokenEOF}},
		{"@", []TokenType{TokenAt, TokenEOF}},
		{";", []TokenType{TokenSemicolon, TokenEOF}},
		{"(", []TokenType{TokenLParen, TokenEOF}},
		{")", []TokenType{TokenRParen, TokenEOF}},
		{"{", []TokenType{TokenLBrace, TokenEOF}},
		{"}", []TokenType{TokenRBrace, TokenEOF}},
		{"[", []TokenType{TokenLBracket, TokenEOF}},
		{"]", []TokenType{TokenRBracket, TokenEOF}},
		{"?", []TokenType{TokenQuestion, TokenEOF}},
		{"..<", []TokenType{TokenRangeUntil, TokenEOF}},
		{`"hello"`, []TokenType{TokenString, TokenEOF}},
		{"1 + 2 * 3", []TokenType{TokenNumber, TokenPlus, TokenNumber, TokenStar, TokenNumber, TokenEOF}},
		{"a >= b && c <= d", []TokenType{TokenIdent, TokenGreaterEq, TokenIdent, TokenAmpAmp, TokenIdent, TokenLessEq, TokenIdent, TokenEOF}},
		{"x != y || z == w", []TokenType{TokenIdent, TokenBangEq, TokenIdent, TokenPipePipe, TokenIdent, TokenEqEq, TokenIdent, TokenEOF}},
	}

	for _, tt := range tests {
		tokens, err := collectTokens(tt.src)
		if err != nil {
			t.Errorf("collectTokens(%q): %v", tt.src, err)
			continue
		}
		if len(tokens) != len(tt.want) {
			t.Errorf("collectTokens(%q): got %d tokens, want %d", tt.src, len(tokens), len(tt.want))
			continue
		}
		for i, tok := range tokens {
			if tok.typ != tt.want[i] {
				t.Errorf("collectTokens(%q): token[%d].typ = %s, want %s", tt.src, i, TokenName(tok.typ), TokenName(tt.want[i]))
			}
		}
	}
}

func TestLexStringEscapes(t *testing.T) {
	tests := []struct {
		src     string
		wantLit string
		wantErr bool
	}{
		{`"hello"`, "hello", false},
		{`"a\nb"`, "a\nb", false},
		{`"a\tb"`, "a\tb", false},
		{`"a\\b"`, "a\\b", false},
		{`"a\"b"`, "a\"b", false},
		{`"a\rb"`, "a\rb", false},
		{`"a\x00b"`, "a\x00b", false},
		{`"a\x1fb"`, "a\x1fb", false},
		{`"hello`, "", true},     // unterminated
		{`"bad\q"`, "", true},    // unsupported escape
		{`"trailing\`, "", true}, // unterminated escape
	}

	for _, tt := range tests {
		tokens, err := collectTokens(tt.src)
		if tt.wantErr {
			if err == nil {
				t.Errorf("collectTokens(%q): expected error", tt.src)
			}
			continue
		}
		if err != nil {
			t.Errorf("collectTokens(%q): %v", tt.src, err)
			continue
		}
		if len(tokens) < 1 || tokens[0].typ != TokenString {
			t.Errorf("collectTokens(%q): expected string token", tt.src)
			continue
		}
		if tokens[0].lit != tt.wantLit {
			t.Errorf("collectTokens(%q): lit = %q, want %q", tt.src, tokens[0].lit, tt.wantLit)
		}
	}
}

func TestLexNumbers(t *testing.T) {
	tests := []struct {
		src     string
		wantNum int64
	}{
		{"0", 0},
		{"42", 42},
		{"999999", 999999},
		{"100", 100},
	}

	for _, tt := range tests {
		tokens, err := collectTokens(tt.src)
		if err != nil {
			t.Errorf("collectTokens(%q): %v", tt.src, err)
			continue
		}
		if len(tokens) < 1 || tokens[0].typ != TokenNumber {
			t.Errorf("collectTokens(%q): expected number token", tt.src)
			continue
		}
		if tokens[0].num != tt.wantNum {
			t.Errorf("collectTokens(%q): num = %d, want %d", tt.src, tokens[0].num, tt.wantNum)
		}
	}
}

func TestLexComments(t *testing.T) {
	tests := []struct {
		src  string
		want []TokenType
	}{
		{"// comment\n42", []TokenType{TokenNumber, TokenEOF}},
		{"// comment", []TokenType{TokenEOF}},
		{"42 // trailing\n43", []TokenType{TokenNumber, TokenNumber, TokenEOF}},
	}

	for _, tt := range tests {
		tokens, err := collectTokens(tt.src)
		if err != nil {
			t.Errorf("collectTokens(%q): %v", tt.src, err)
			continue
		}
		if len(tokens) != len(tt.want) {
			t.Errorf("collectTokens(%q): got %d tokens, want %d", tt.src, len(tokens), len(tt.want))
			continue
		}
		for i, tok := range tokens {
			if tok.typ != tt.want[i] {
				t.Errorf("collectTokens(%q): token[%d].typ = %s, want %s", tt.src, i, TokenName(tok.typ), TokenName(tt.want[i]))
			}
		}
	}
}

func TestLexCRLFAndLFPositionTracking(t *testing.T) {
	tokens, err := collectTokens("a\r\nb\nc")
	if err != nil {
		t.Fatalf("collectTokens: %v", err)
	}
	expected := []struct {
		line int
		col  int
	}{
		{1, 1},
		{2, 1},
		{3, 1},
	}
	for i, exp := range expected {
		if tokens[i].pos.Line != exp.line || tokens[i].pos.Col != exp.col {
			t.Fatalf("token[%d] pos = %d:%d, want %d:%d", i, tokens[i].pos.Line, tokens[i].pos.Col, exp.line, exp.col)
		}
	}
}

func TestLexUnicodeInStringsAndComments(t *testing.T) {
	tokens, err := collectTokens("// привіт 🌊\n\"Привіт 🌊\"")
	if err != nil {
		t.Fatalf("collectTokens: %v", err)
	}
	if len(tokens) != 2 || tokens[0].typ != TokenString || tokens[1].typ != TokenEOF {
		t.Fatalf("tokens = %#v, want string/eof", tokens)
	}
	if tokens[0].lit != "Привіт 🌊" {
		t.Fatalf("string literal = %q", tokens[0].lit)
	}
}

func TestLexFlowTestBlockTokenCoverage(t *testing.T) {
	src := "test \"арифметика\":\n    expect 40 + 2 == 42\n"
	tokens, err := collectTokens(src)
	if err != nil {
		t.Fatalf("collectTokens: %v", err)
	}
	want := []TokenType{
		TokenTest, TokenString, TokenColon, TokenExpect,
		TokenNumber, TokenPlus, TokenNumber, TokenEqEq, TokenNumber, TokenEOF,
	}
	if len(tokens) != len(want) {
		t.Fatalf("tokens len = %d, want %d (%#v)", len(tokens), len(want), tokens)
	}
	for i, tt := range want {
		if tokens[i].typ != tt {
			t.Fatalf("token[%d] = %s, want %s", i, TokenName(tokens[i].typ), TokenName(tt))
		}
	}
	if tokens[1].lit != "арифметика" {
		t.Fatalf("test name literal = %q, want арифметика", tokens[1].lit)
	}
}

func TestLexFlowTestBlockUnsupportedEscapeDiagnostic(t *testing.T) {
	_, err := collectTokensFrom("test \"bad\\q\":\n    expect 1 == 1\n", "fixtures/flow_test_bad_escape.tetra")
	if err == nil {
		t.Fatalf("expected diagnostic")
	}
	diag, ok := DiagnosticForError(err)
	if !ok {
		t.Fatalf("expected structured diagnostic: %T %v", err, err)
	}
	if diag.Code != DiagnosticCodeParse || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v", diag)
	}
	if diag.File != "fixtures/flow_test_bad_escape.tetra" || diag.Line != 1 || diag.Column != 6 {
		t.Fatalf("diagnostic position = %q:%d:%d, want fixtures/flow_test_bad_escape.tetra:1:6", diag.File, diag.Line, diag.Column)
	}
	if diag.Message != "unsupported escape: \\q" {
		t.Fatalf("diagnostic message = %q", diag.Message)
	}
}

func TestLexFlowTestBlockSpanCRLFTabAndUnicode(t *testing.T) {
	tokens, err := collectTokens("test \"Привіт\"\r\n\texpect 1 == 1\n")
	if err != nil {
		t.Fatalf("collectTokens: %v", err)
	}
	if len(tokens) < 7 {
		t.Fatalf("token count = %d, want at least 7", len(tokens))
	}
	if tokens[0].typ != TokenTest || tokens[0].pos.Line != 1 || tokens[0].pos.Col != 1 {
		t.Fatalf("test token pos = %d:%d, want 1:1", tokens[0].pos.Line, tokens[0].pos.Col)
	}
	if tokens[2].typ != TokenExpect || tokens[2].pos.Line != 2 || tokens[2].pos.Col != 2 {
		t.Fatalf("expect token pos = %d:%d, want 2:2", tokens[2].pos.Line, tokens[2].pos.Col)
	}
	last := tokens[len(tokens)-1]
	if last.typ != TokenEOF || last.pos.Line != 3 || last.pos.Col != 1 {
		t.Fatalf("EOF token pos = %d:%d (%s), want 3:1 EOF", last.pos.Line, last.pos.Col, TokenName(last.typ))
	}
}

func TestLexInvalidUTF8Diagnostic(t *testing.T) {
	_, err := collectTokens(string([]byte{'f', 'n', ' ', 0xff, '\n'}))
	if err == nil {
		t.Fatalf("expected invalid UTF-8 diagnostic")
	}
	if !strings.Contains(err.Error(), "invalid UTF-8") {
		t.Fatalf("error = %v", err)
	}
}

func TestPlan250LexMixedNewlinesEOFAndDocComments(t *testing.T) {
	src := "/// module docs\r\nfunc main() -> Int:\n    /// return docs\r\n    return 0"
	tokens, err := collectTokensFrom(src, "doc_comments.tetra")
	if err != nil {
		t.Fatalf("collectTokens: %v", err)
	}
	want := []struct {
		typ  TokenType
		line int
		col  int
	}{
		{TokenFun, 2, 1},
		{TokenIdent, 2, 6},
		{TokenLParen, 2, 10},
		{TokenRParen, 2, 11},
		{TokenArrow, 2, 13},
		{TokenIdent, 2, 16},
		{TokenColon, 2, 19},
		{TokenReturn, 4, 5},
		{TokenNumber, 4, 12},
		{TokenEOF, 4, 13},
	}
	if len(tokens) != len(want) {
		t.Fatalf("tokens len = %d, want %d: %#v", len(tokens), len(want), tokens)
	}
	for i, exp := range want {
		if tokens[i].typ != exp.typ || tokens[i].pos.Line != exp.line || tokens[i].pos.Col != exp.col {
			t.Fatalf("token[%d] = %s at %d:%d, want %s at %d:%d", i, TokenName(tokens[i].typ), tokens[i].pos.Line, tokens[i].pos.Col, TokenName(exp.typ), exp.line, exp.col)
		}
	}
}

func TestPlan250LexInvalidUTF8PositionIsDeterministic(t *testing.T) {
	_, err := collectTokensFrom(string([]byte{'o', 'k', '\r', '\n', 0xff}), "bad_utf8.tetra")
	if err == nil {
		t.Fatalf("expected invalid UTF-8 diagnostic")
	}
	diag, ok := DiagnosticForError(err)
	if !ok {
		t.Fatalf("expected structured diagnostic: %T %v", err, err)
	}
	if diag.File != "bad_utf8.tetra" || diag.Line != 2 || diag.Column != 1 || diag.Message != "invalid UTF-8 encoding" {
		t.Fatalf("diagnostic = %#v, want bad_utf8.tetra:2:1 invalid UTF-8 encoding", diag)
	}
}

func TestPlan250LexStringEscapeCornerPositions(t *testing.T) {
	tests := []struct {
		name string
		src  string
		lit  string
		want string
		col  int
	}{
		{name: "valid escaped quote and slash", src: `"a\"\\b"`, lit: "a\"\\b"},
		{name: "valid carriage return escape", src: `"a\rb"`, lit: "a\rb"},
		{name: "valid hex control escape", src: `"a\x1fb"`, lit: "a\x1fb"},
		{name: "unsupported escape", src: `"bad\q"`, want: "unsupported escape: \\q", col: 1},
		{name: "unterminated escape", src: `"bad\`, want: "unterminated escape sequence", col: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := collectTokensFrom(tt.src, tt.name+".tetra")
			if tt.want == "" {
				if err != nil {
					t.Fatalf("collectTokens: %v", err)
				}
				if tokens[0].typ != TokenString || tokens[0].lit != tt.lit {
					t.Fatalf("token = %#v, want string %q", tokens[0], tt.lit)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected diagnostic")
			}
			diag, ok := DiagnosticForError(err)
			if !ok {
				t.Fatalf("expected structured diagnostic: %T %v", err, err)
			}
			if diag.Line != 1 || diag.Column != tt.col || diag.Message != tt.want {
				t.Fatalf("diagnostic = %#v, want 1:%d %q", diag, tt.col, tt.want)
			}
		})
	}
}

func TestPlan250LexDocCommentsRemainTrivia(t *testing.T) {
	src := "/// file docs\r\n/// more docs\nfunc main() -> Int:\n    /// return docs\n    return 0\n"
	tokens, err := collectTokensFrom(src, "doc_trivia.tetra")
	if err != nil {
		t.Fatalf("collectTokens: %v", err)
	}
	if len(tokens) != 10 {
		t.Fatalf("tokens len = %d, want 10: %#v", len(tokens), tokens)
	}
	if tokens[0].typ != TokenFun || tokens[0].pos.Line != 3 || tokens[0].pos.Col != 1 {
		t.Fatalf("first token = %s at %d:%d, want func at 3:1", TokenName(tokens[0].typ), tokens[0].pos.Line, tokens[0].pos.Col)
	}
	if tokens[7].typ != TokenReturn || tokens[7].pos.Line != 5 || tokens[7].pos.Col != 5 {
		t.Fatalf("return token = %s at %d:%d, want return at 5:5", TokenName(tokens[7].typ), tokens[7].pos.Line, tokens[7].pos.Col)
	}
	if tokens[9].typ != TokenEOF || tokens[9].pos.Line != 6 || tokens[9].pos.Col != 1 {
		t.Fatalf("EOF token = %s at %d:%d, want EOF at 6:1", TokenName(tokens[9].typ), tokens[9].pos.Line, tokens[9].pos.Col)
	}
}

func TestLexLineCommentsDoNotNest(t *testing.T) {
	tokens, err := collectTokens("// looks like /* nested */ comment\n42")
	if err != nil {
		t.Fatalf("collectTokens: %v", err)
	}
	if len(tokens) != 2 || tokens[0].typ != TokenNumber || tokens[1].typ != TokenEOF {
		t.Fatalf("tokens = %#v, want number/eof", tokens)
	}
}

func TestLexFuzzRegressionCorpus(t *testing.T) {
	tests := []struct {
		name    string
		src     []byte
		wantErr string
	}{
		{name: "invalid utf8 after keyword", src: []byte{'f', 'n', ' ', 0xff, '\n'}, wantErr: "invalid UTF-8 encoding"},
		{name: "nul byte", src: []byte{0x00}, wantErr: "unexpected character"},
		{name: "lone ampersand", src: []byte("&"), wantErr: "did you mean '&&'?"},
		{name: "lone pipe", src: []byte("|"), wantErr: "did you mean '||'?"},
		{name: "unterminated escape", src: []byte(`"trailing\`), wantErr: "unterminated escape sequence"},
		{name: "unicode comment then string", src: []byte("// Привіт 🌊\n\"ok\"")},
		{name: "flow test block", src: []byte("test \"math\":\n    expect 40 + 2 == 42\n")},
		{name: "flow test bad escape", src: []byte("test \"bad\\q\":\n    expect 1 == 1\n"), wantErr: "unsupported escape: \\q"},
		{name: "flow test crlf tab unicode", src: []byte("test \"Привіт\"\r\n\texpect 1 == 1\r\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := collectTokens(string(tt.src))
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("collectTokens: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestLexArchivedFuzzCrashers(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "lexer", "crashers", "*.tetra"))
	if err != nil {
		t.Fatalf("glob crashers: %v", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatalf("expected archived lexer crashers")
	}
	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read crasher: %v", err)
			}
			_, _ = collectTokens(string(raw))
		})
	}
}

func TestLexLongLineAndLargeFileSmoke(t *testing.T) {
	longIdent := strings.Repeat("a", 32*1024)
	tokens, err := collectTokens(longIdent)
	if err != nil {
		t.Fatalf("collect long identifier: %v", err)
	}
	if len(tokens) != 2 || tokens[0].typ != TokenIdent || tokens[1].typ != TokenEOF {
		t.Fatalf("tokens = %#v, want ident/eof", tokens)
	}
	if tokens[0].lit != longIdent {
		t.Fatalf("long identifier length = %d, want %d", len(tokens[0].lit), len(longIdent))
	}
	if tokens[1].pos.Line != 1 || tokens[1].pos.Col != len(longIdent)+1 {
		t.Fatalf("EOF position = %d:%d, want 1:%d", tokens[1].pos.Line, tokens[1].pos.Col, len(longIdent)+1)
	}

	var large strings.Builder
	for i := 0; i < 4096; i++ {
		large.WriteString("let value")
		large.WriteString("0")
		large.WriteString(": i32 = 42\n")
	}
	tokens, err = collectTokens(large.String())
	if err != nil {
		t.Fatalf("collect large file: %v", err)
	}
	if tokens[len(tokens)-1].typ != TokenEOF {
		t.Fatalf("last token = %s, want EOF", TokenName(tokens[len(tokens)-1].typ))
	}
	if tokens[len(tokens)-1].pos.Line != 4097 || tokens[len(tokens)-1].pos.Col != 1 {
		t.Fatalf("large-file EOF position = %d:%d, want 4097:1", tokens[len(tokens)-1].pos.Line, tokens[len(tokens)-1].pos.Col)
	}
}

func TestLexPositionTracking(t *testing.T) {
	src := "a\nb c"
	tokens, err := collectTokens(src)
	if err != nil {
		t.Fatalf("collectTokens: %v", err)
	}

	// tokens: a (1:1), b (2:1), c (2:3), EOF
	expected := []struct {
		line int
		col  int
	}{
		{1, 1}, // a
		{2, 1}, // b
		{2, 3}, // c
	}

	for i, exp := range expected {
		if i >= len(tokens) {
			t.Fatalf("not enough tokens: got %d, need at least %d", len(tokens), i+1)
		}
		if tokens[i].pos.Line != exp.line || tokens[i].pos.Col != exp.col {
			t.Errorf("token[%d] pos = %d:%d, want %d:%d", i, tokens[i].pos.Line, tokens[i].pos.Col, exp.line, exp.col)
		}
	}
}

func TestLexMultiCharOperators(t *testing.T) {
	tests := []struct {
		src  string
		want TokenType
		lit  string
	}{
		{"==", TokenEqEq, "=="},
		{"!=", TokenBangEq, "!="},
		{"!", TokenBang, "!"},
		{">=", TokenGreaterEq, ">="},
		{"<=", TokenLessEq, "<="},
		{"->", TokenArrow, "->"},
		{"&&", TokenAmpAmp, "&&"},
		{"||", TokenPipePipe, "||"},
	}

	for _, tt := range tests {
		tokens, err := collectTokens(tt.src)
		if err != nil {
			t.Errorf("collectTokens(%q): %v", tt.src, err)
			continue
		}
		if len(tokens) < 1 {
			t.Errorf("collectTokens(%q): no tokens", tt.src)
			continue
		}
		if tokens[0].typ != tt.want {
			t.Errorf("collectTokens(%q): typ = %s, want %s", tt.src, TokenName(tokens[0].typ), TokenName(tt.want))
		}
		if tokens[0].lit != tt.lit {
			t.Errorf("collectTokens(%q): lit = %q, want %q", tt.src, tokens[0].lit, tt.lit)
		}
	}
}

func TestLexErrors(t *testing.T) {
	tests := []struct {
		src string
	}{
		{"&"}, // lone &
		{"|"}, // lone |
		{"`"}, // unknown char
		{"~"}, // unknown char
	}

	for _, tt := range tests {
		_, err := collectTokens(tt.src)
		if err == nil {
			t.Errorf("collectTokens(%q): expected error", tt.src)
		}
	}
}
