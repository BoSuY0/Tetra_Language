package frontend

import (
	"fmt"
	"strconv"
)

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdent
	TokenNumber
	TokenString
	TokenFn
	TokenFun
	TokenLet
	TokenVar
	TokenVal
	TokenModule
	TokenImport
	TokenAs
	TokenStruct
	TokenIf
	TokenElse
	TokenWhile
	TokenReturn
	TokenPrint
	TokenFree
	TokenUnsafe
	TokenAt
	TokenArrow
	TokenColon
	TokenAssign
	TokenEqEq
	TokenPlus
	TokenMinus
	TokenLess
	TokenComma
	TokenDot
	TokenLBracket
	TokenRBracket
	TokenLParen
	TokenRParen
	TokenLBrace
	TokenRBrace
	TokenSemicolon
)

type token struct {
	typ TokenType
	pos Position
	lit string
	str []byte
	num int64
}

type lexer struct {
	src  []byte
	idx  int
	line int
	col  int
	file string
}

func newLexer(src []byte, file string) *lexer {
	return &lexer{src: src, line: 1, col: 1, file: file}
}

func (l *lexer) nextToken() (token, error) {
	if err := l.skipSpaceAndComments(); err != nil {
		return token{}, err
	}
	if l.idx >= len(l.src) {
		return token{typ: TokenEOF, pos: l.pos()}, nil
	}

	ch := l.peek()
	pos := l.pos()

	if isIdentStart(ch) {
		start := l.idx
		l.advance()
		for l.idx < len(l.src) && isIdentPart(l.peek()) {
			l.advance()
		}
		lit := string(l.src[start:l.idx])
		switch lit {
		case "fn":
			return token{typ: TokenFn, pos: pos, lit: lit}, nil
		case "fun":
			return token{typ: TokenFun, pos: pos, lit: lit}, nil
		case "let":
			return token{typ: TokenLet, pos: pos, lit: lit}, nil
		case "var":
			return token{typ: TokenVar, pos: pos, lit: lit}, nil
		case "val":
			return token{typ: TokenVal, pos: pos, lit: lit}, nil
		case "module":
			return token{typ: TokenModule, pos: pos, lit: lit}, nil
		case "import":
			return token{typ: TokenImport, pos: pos, lit: lit}, nil
		case "as":
			return token{typ: TokenAs, pos: pos, lit: lit}, nil
		case "struct":
			return token{typ: TokenStruct, pos: pos, lit: lit}, nil
		case "if":
			return token{typ: TokenIf, pos: pos, lit: lit}, nil
		case "else":
			return token{typ: TokenElse, pos: pos, lit: lit}, nil
		case "while":
			return token{typ: TokenWhile, pos: pos, lit: lit}, nil
		case "return":
			return token{typ: TokenReturn, pos: pos, lit: lit}, nil
		case "print":
			return token{typ: TokenPrint, pos: pos, lit: lit}, nil
		case "free":
			return token{typ: TokenFree, pos: pos, lit: lit}, nil
		case "unsafe":
			return token{typ: TokenUnsafe, pos: pos, lit: lit}, nil
		default:
			return token{typ: TokenIdent, pos: pos, lit: lit}, nil
		}
	}

	if isDigit(ch) {
		start := l.idx
		l.advance()
		for l.idx < len(l.src) && isDigit(l.peek()) {
			l.advance()
		}
		lit := string(l.src[start:l.idx])
		val, err := strconv.ParseInt(lit, 10, 64)
		if err != nil {
			return token{}, l.errorf(pos, "invalid number: %s", lit)
		}
		return token{typ: TokenNumber, pos: pos, lit: lit, num: val}, nil
	}

	switch ch {
	case '"':
		return l.readString()
	case '-':
		if l.peekNext() == '>' {
			l.advance()
			l.advance()
			return token{typ: TokenArrow, pos: pos, lit: "->"}, nil
		}
		l.advance()
		return token{typ: TokenMinus, pos: pos, lit: "-"}, nil
	case '+':
		l.advance()
		return token{typ: TokenPlus, pos: pos, lit: "+"}, nil
	case ':':
		l.advance()
		return token{typ: TokenColon, pos: pos, lit: ":"}, nil
	case '=':
		if l.peekNext() == '=' {
			l.advance()
			l.advance()
			return token{typ: TokenEqEq, pos: pos, lit: "=="}, nil
		}
		l.advance()
		return token{typ: TokenAssign, pos: pos, lit: "="}, nil
	case '<':
		l.advance()
		return token{typ: TokenLess, pos: pos, lit: "<"}, nil
	case ',':
		l.advance()
		return token{typ: TokenComma, pos: pos, lit: ","}, nil
	case '.':
		l.advance()
		return token{typ: TokenDot, pos: pos, lit: "."}, nil
	case '[':
		l.advance()
		return token{typ: TokenLBracket, pos: pos, lit: "["}, nil
	case ']':
		l.advance()
		return token{typ: TokenRBracket, pos: pos, lit: "]"}, nil
	case '(':
		l.advance()
		return token{typ: TokenLParen, pos: pos, lit: "("}, nil
	case ')':
		l.advance()
		return token{typ: TokenRParen, pos: pos, lit: ")"}, nil
	case '{':
		l.advance()
		return token{typ: TokenLBrace, pos: pos, lit: "{"}, nil
	case '}':
		l.advance()
		return token{typ: TokenRBrace, pos: pos, lit: "}"}, nil
	case ';':
		l.advance()
		return token{typ: TokenSemicolon, pos: pos, lit: ";"}, nil
	case '@':
		l.advance()
		return token{typ: TokenAt, pos: pos, lit: "@"}, nil
	default:
		return token{}, l.errorf(pos, "unexpected character: %q", ch)
	}
}

func (l *lexer) readString() (token, error) {
	pos := l.pos()
	l.advance()
	var out []byte
	for {
		if l.idx >= len(l.src) {
			return token{}, l.errorf(pos, "unterminated string literal")
		}
		ch := l.advance()
		if ch == '"' {
			break
		}
		if ch == '\\' {
			if l.idx >= len(l.src) {
				return token{}, l.errorf(pos, "unterminated escape sequence")
			}
			esc := l.advance()
			switch esc {
			case 'n':
				out = append(out, '\n')
			case 'r':
				out = append(out, '\r')
			case 't':
				out = append(out, '\t')
			case '\\':
				out = append(out, '\\')
			case '"':
				out = append(out, '"')
			default:
				return token{}, l.errorf(pos, "unsupported escape: \\%c", esc)
			}
			continue
		}
		out = append(out, ch)
	}

	return token{typ: TokenString, pos: pos, lit: string(out), str: out}, nil
}

func (l *lexer) skipSpaceAndComments() error {
	for {
		if l.idx >= len(l.src) {
			return nil
		}
		ch := l.peek()
		switch ch {
		case ' ', '\t', '\r':
			l.advance()
			continue
		case '\n':
			l.advance()
			continue
		case '/':
			if l.peekNext() == '/' {
				l.advance()
				l.advance()
				for l.idx < len(l.src) && l.peek() != '\n' {
					l.advance()
				}
				continue
			}
		}
		return nil
	}
}

func (l *lexer) advance() byte {
	ch := l.src[l.idx]
	l.idx++
	if ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return ch
}

func (l *lexer) peek() byte {
	if l.idx >= len(l.src) {
		return 0
	}
	return l.src[l.idx]
}

func (l *lexer) peekNext() byte {
	if l.idx+1 >= len(l.src) {
		return 0
	}
	return l.src[l.idx+1]
}

func (l *lexer) pos() Position {
	return Position{File: l.file, Line: l.line, Col: l.col}
}

func (l *lexer) errorf(pos Position, format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("%s: %s", FormatPos(pos), msg)
}

func isIdentStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isIdentPart(ch byte) bool {
	return isIdentStart(ch) || isDigit(ch)
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
