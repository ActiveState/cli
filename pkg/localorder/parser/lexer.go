package parser

import (
	"unicode/utf8"

	"github.com/ActiveState/cli/internal/errs"
)

type Lexer struct {
	input []byte
	pos   Position
	read  int
}

type Position struct {
	line   int
	column int
}

func NewLexer(input []byte) *Lexer {
	return &Lexer{
		input: input,
		pos:   Position{line: 1, column: 1},
		read:  0,
	}
}

func (l *Lexer) Scan() (Position, Token, string, error) {
	r := l.next()

	if r == 0 || l.read >= len(l.input) {
		return l.pos, EOF, "", nil
	}

	for r == ' ' || r == '\t' || r == '\n' {
		r = l.next()
	}

	if l.isLetStart(r) || l.isInStart(r) {
		return l.lexKeyword(r)
	}

	if l.isAlphanumeric(r) {
		return l.lexIdentifier(r)
	}

	switch r {
	case '=':
		return l.pos, EQUALS, "=", nil
	case '(':
		return l.pos, L_PAREN, "(", nil
	case ')':
		return l.pos, R_PAREN, ")", nil
	case '[':
		return l.pos, L_BRACKET, "[", nil
	case ']':
		return l.pos, R_BRACKET, "]", nil
	case '#':
		return l.lexComment(r)
	case ':':
		return l.pos, COLON, ":", nil
	case '"':
		return l.lexString(r)
	case ',':
		return l.pos, COMMA, ",", nil
	case '{':
		return l.pos, L_CURL, "{", nil
	case '}':
		return l.pos, R_CURL, "}", nil
	default:
		return l.pos, ILLEGAL, "", errs.New("unexpected rune: %s at %d:%d", string(r), l.pos.line, l.pos.column)
	}
}

func (l *Lexer) next() rune {
	if l.read >= len(l.input) {
		return 0
	}

	r, s := utf8.DecodeRune(l.input[l.read:])
	l.read += s
	l.pos.column += s

	if r == '\n' {
		l.pos.line++
		l.pos.column = 1
	}

	return r
}

func (l *Lexer) backtrack() {
	if l.read <= 0 {
		return
	}
	_, s := utf8.DecodeLastRune(l.input[:l.read])
	l.read -= s
	l.pos.column -= s
}

func (l *Lexer) peek() rune {
	if l.read >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRune(l.input[l.read:])
	return r
}

func (l *Lexer) peekN(n int) rune {
	if l.read+n >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRune(l.input[l.read+n:])
	return r
}

func (l *Lexer) isLetStart(r rune) bool {
	return r == 'l' && l.peek() == 'e' && l.peekN(1) == 't'
}

func (l *Lexer) isInStart(r rune) bool {
	return r == 'i' && l.peek() == 'n'
}

func (l *Lexer) lexKeyword(r rune) (Position, Token, string, error) {
	if l.read >= len(l.input) {
		return l.pos, ILLEGAL, "", errs.New("unexpected end of input lexing keyword")
	}
	start := l.read - 1
	for l.isAlphanumeric(r) {
		r = l.next()
	}

	l.backtrack()
	name := string(l.input[start:l.read])
	keyword, ok := keywordTokens[name]
	if !ok {
		return l.pos, ILLEGAL, "", errs.New("unexpected identifier: %s", string(l.input[start:l.read+1]))
	}

	// TODO: Should this be done in the parsing step?
	switch keyword {
	case LET, IN:
		if r != ':' {
			return l.pos, ILLEGAL, "", errs.New("expected ':'")
		}
	}

	return l.pos, keyword, name, nil
}

func (l *Lexer) isAlphanumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

func (l *Lexer) lexIdentifier(r rune) (Position, Token, string, error) {
	start := l.read - 1
	for l.isAlphanumeric(r) {
		r = l.next()
	}

	l.backtrack()
	name := string(l.input[start:l.read])
	keyword, ok := keywordTokens[name]
	if !ok {
		return l.pos, IDENTIFIER, name, nil
	}

	return l.pos, keyword, name, nil
}

func (l *Lexer) lexString(r rune) (Position, Token, string, error) {
	if r != '"' {
		return l.pos, ILLEGAL, "", errs.New("expected '\"'")
	}

	start := l.read
	r = l.next()
	// Slash is a workaround
	for r != '"' {
		r = l.next()
		if r == 0 {
			return l.pos, ILLEGAL, "", errs.New("unexpected end of input lexing string")
		}
	}

	l.backtrack()
	result := string(l.input[start:l.read])

	// Consume the last "
	l.next()
	return l.pos, STRING, result, nil
}

func (l *Lexer) lexComment(r rune) (Position, Token, string, error) {
	if r != '#' {
		return l.pos, ILLEGAL, "", errs.New("expected '#'")
	}

	start := l.read
	r = l.next()
	for r != '\n' {
		r = l.next()
		if r == 0 {
			return l.pos, ILLEGAL, "", errs.New("unexpected end of input lexing comment")
		}
	}

	l.backtrack()
	return l.pos, COMMENT, string(l.input[start:l.read]), nil
}
