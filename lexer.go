package jsonc

import "fmt"

// ---------------------------------------------------------------------------
// Token types
// ---------------------------------------------------------------------------

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokError
	tokLBrace
	tokRBrace
	tokLBracket
	tokRBracket
	tokColon
	tokComma
	tokString
	tokNumber
	tokTrue
	tokFalse
	tokNull
	tokCommentLine  // // ...
	tokCommentBlock // /* ... */
	tokWhitespace
)

// token with proper source position.
type token struct {
	kind tokenKind
	text string
	pos  Position // start position of the token
	end  Position // end position (exclusive)
}

// ---------------------------------------------------------------------------
// Lexer
// ---------------------------------------------------------------------------

type lexer struct {
	input []byte
	pos   int // current byte offset
	line  int // current line (0-based)
	col   int // current column (0-based)
}

func newLexer(input []byte) *lexer {
	return &lexer{
		input: input,
		pos:   0,
		line:  0,
		col:   0,
	}
}

func (l *lexer) tokPos() Position {
	return Position{Offset: l.pos, Line: l.line, Column: l.col}
}

func (l *lexer) advance() byte {
	c := l.input[l.pos]
	if c == '\n' {
		l.line++
		l.col = 0
	} else {
		l.col++
	}
	l.pos++
	return c
}

func (l *lexer) peek() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

// next returns the next token from the input.
func (l *lexer) next() token {
	// Whitespace and comments are always scanned as separate tokens
	for l.pos < len(l.input) {
		c := l.peek()
		switch c {
		case ' ', '\t', '\n', '\r':
			return l.scanWhitespace()
		case '/':
			if l.pos+1 < len(l.input) {
				next := l.input[l.pos+1]
				if next == '/' {
					return l.scanLineComment()
				}
				if next == '*' {
					return l.scanBlockComment()
				}
			}
			return l.errorToken("unexpected character '/'")
		default:
			return l.scanValueToken()
		}
	}
	return token{kind: tokEOF, pos: l.tokPos(), end: l.tokPos()}
}

// scanValueToken scans a JSON value token (structural character, string,
// number, keyword) starting from the current position.
func (l *lexer) scanValueToken() token {
	if l.pos >= len(l.input) {
		return token{kind: tokEOF, pos: l.tokPos(), end: l.tokPos()}
	}

	start := l.tokPos()
	c := l.advance()

	switch {
	case c == '{':
		return token{kind: tokLBrace, text: "{", pos: start, end: l.tokPos()}
	case c == '}':
		return token{kind: tokRBrace, text: "}", pos: start, end: l.tokPos()}
	case c == '[':
		return token{kind: tokLBracket, text: "[", pos: start, end: l.tokPos()}
	case c == ']':
		return token{kind: tokRBracket, text: "]", pos: start, end: l.tokPos()}
	case c == ':':
		return token{kind: tokColon, text: ":", pos: start, end: l.tokPos()}
	case c == ',':
		return token{kind: tokComma, text: ",", pos: start, end: l.tokPos()}
	case c == '"':
		return l.scanString(start)
	case c == 't':
		return l.scanKeyword("true", tokTrue, start)
	case c == 'f':
		return l.scanKeyword("false", tokFalse, start)
	case c == 'n':
		return l.scanKeyword("null", tokNull, start)
	case c == '-' || (c >= '0' && c <= '9'):
		return l.scanNumber(start)
	default:
		return l.errorToken(fmt.Sprintf("unexpected character %q", c))
	}
}

func (l *lexer) scanWhitespace() token {
	start := l.tokPos()
	for l.pos < len(l.input) {
		c := l.peek()
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			l.advance()
		} else {
			break
		}
	}
	return token{
		kind: tokWhitespace,
		text: string(l.input[start.Offset:l.pos]),
		pos:  start,
		end:  l.tokPos(),
	}
}

func (l *lexer) scanString(start Position) token {
	for l.pos < len(l.input) {
		c := l.advance()
		if c == '"' {
			return token{
				kind: tokString,
				text: string(l.input[start.Offset:l.pos]),
				pos:  start,
				end:  l.tokPos(),
			}
		}
		if c == '\\' && l.pos < len(l.input) {
			l.advance()
		}
	}
	return token{kind: tokError, text: "unterminated string", pos: start, end: l.tokPos()}
}

func (l *lexer) scanNumber(start Position) token {
	for l.pos < len(l.input) {
		c := l.input[l.pos]
		if c == '-' || c == '+' || c == '.' || c == 'e' || c == 'E' || (c >= '0' && c <= '9') {
			l.advance()
		} else {
			break
		}
	}
	text := string(l.input[start.Offset:l.pos])
	if hasLeadingZero(text) {
		return token{
			kind: tokError,
			text: fmt.Sprintf("leading zero in number %q", text),
			pos:  start,
			end:  l.tokPos(),
		}
	}
	return token{
		kind: tokNumber,
		text: text,
		pos:  start,
		end:  l.tokPos(),
	}
}

// hasLeadingZero reports whether a JSON number string has a leading zero,
// which is not valid in JSON except for the literal "0" itself, "0." (fraction),
// and "0e"/"0E" (exponent).
func hasLeadingZero(text string) bool {
	s := text
	if len(s) > 0 && (s[0] == '-' || s[0] == '+') {
		s = s[1:]
	}
	return len(s) >= 2 && s[0] == '0' && s[1] >= '0' && s[1] <= '9'
}

func (l *lexer) scanKeyword(expected string, kind tokenKind, start Position) token {
	end := start.Offset + len(expected)
	if end > len(l.input) || string(l.input[start.Offset:end]) != expected {
		return token{kind: tokError, text: fmt.Sprintf("expected %q", expected), pos: start, end: l.tokPos()}
	}
	for l.pos < end {
		l.advance()
	}
	return token{kind: kind, text: expected, pos: start, end: l.tokPos()}
}

func (l *lexer) scanLineComment() token {
	start := l.tokPos()
	l.advance() // first /
	l.advance() // second /
	// Consume until newline or EOF — stop BEFORE consuming the newline
	// so the newline can be its own whitespace token.
	for l.pos < len(l.input) && l.input[l.pos] != '\n' {
		l.advance()
	}
	// Do NOT consume the \n — leave it for the whitespace scanner
	return token{
		kind: tokCommentLine,
		text: string(l.input[start.Offset:l.pos]),
		pos:  start,
		end:  l.tokPos(),
	}
}

func (l *lexer) scanBlockComment() token {
	start := l.tokPos()
	l.advance() // /
	l.advance() // *
	for l.pos < len(l.input) {
		c := l.advance()
		if c == '*' && l.pos < len(l.input) && l.input[l.pos] == '/' {
			l.advance() // /
			break
		}
	}
	return token{
		kind: tokCommentBlock,
		text: string(l.input[start.Offset:l.pos]),
		pos:  start,
		end:  l.tokPos(),
	}
}

func (l *lexer) errorToken(msg string) token {
	pos := l.tokPos()
	return token{kind: tokError, text: msg, pos: pos, end: pos}
}
