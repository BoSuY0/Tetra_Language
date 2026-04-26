package frontend

import (
	"testing"
)

func collectTokens(src string) ([]token, error) {
	l := newLexer([]byte(src), "test")
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
		{"as", []TokenType{TokenAs, TokenEOF}},
		{"struct", []TokenType{TokenStruct, TokenEOF}},
		{"if", []TokenType{TokenIf, TokenEOF}},
		{"else", []TokenType{TokenElse, TokenEOF}},
		{"while", []TokenType{TokenWhile, TokenEOF}},
		{"return", []TokenType{TokenReturn, TokenEOF}},
		{"print", []TokenType{TokenPrint, TokenEOF}},
		{"free", []TokenType{TokenFree, TokenEOF}},
		{"unsafe", []TokenType{TokenUnsafe, TokenEOF}},
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
