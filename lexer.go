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
	input string
	pos   int // current byte offset
	line  int // current line (0-based)
	col   int // current column (0-based)
}

func newLexer(input string) *lexer {
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
			l.advance() // consume the '/'
			// Return error token positioned at the consumed '/'
			pos := l.tokPos()
			start := Position{Offset: pos.Offset - 1, Line: pos.Line, Column: pos.Column - 1}
			return token{kind: tokError, text: "unexpected character '/'", pos: start, end: pos}
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
		text: l.input[start.Offset:l.pos],
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
				text: l.input[start.Offset:l.pos],
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
	text := l.input[start.Offset:l.pos]
	if err := validateJSONNumber(text); err != "" {
		return token{
			kind: tokError,
			text: err,
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

// validateJSONNumber checks whether text is a valid JSON number per RFC 8259.
// Returns empty string for valid, or an error message for invalid.
func validateJSONNumber(text string) string {
	if text == "" {
		return "empty number"
	}
	s := text
	i := 0

	// Optional leading sign
	if s[i] == '-' || s[i] == '+' {
		i++
	}
	if i >= len(s) {
		return fmt.Sprintf("incomplete number %q", text)
	}

	// Integer part: "0" or non-zero-digit *digit
	if !scanIntPart(s, &i) {
		return fmt.Sprintf("invalid number %q", text)
	}

	// Optional fraction part
	if i < len(s) && s[i] == '.' {
		if !scanFracPart(s, &i) {
			return fmt.Sprintf("incomplete fraction in number %q", text)
		}
	}

	// Optional exponent part
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		if !scanExpPart(s, &i) {
			return fmt.Sprintf("incomplete exponent in number %q", text)
		}
	}

	if i < len(s) {
		return fmt.Sprintf("trailing characters in number %q", text)
	}
	return ""
}

// scanIntPart scans the integer portion of a JSON number.
// Expects s[*i] to be at the start of the integer part.
// Returns false if invalid, updates *i past the scanned digits.
// Leading zeros (e.g., "01") are rejected.
func scanIntPart(s string, i *int) bool {
	if s[*i] == '0' {
		*i++
		// Leading zero followed by another digit is not valid JSON
		if *i < len(s) && s[*i] >= '0' && s[*i] <= '9' {
			return false
		}
		return true
	}
	if s[*i] >= '1' && s[*i] <= '9' {
		*i++
		for *i < len(s) && s[*i] >= '0' && s[*i] <= '9' {
			*i++
		}
		return true
	}
	return false
}

func scanFracPart(s string, i *int) bool {
	*i++ // skip '.'
	if *i >= len(s) || s[*i] < '0' || s[*i] > '9' {
		return false
	}
	for *i < len(s) && s[*i] >= '0' && s[*i] <= '9' {
		*i++
	}
	return true
}

func scanExpPart(s string, i *int) bool {
	*i++ // skip 'e' or 'E'
	if *i < len(s) && (s[*i] == '-' || s[*i] == '+') {
		*i++
	}
	if *i >= len(s) || s[*i] < '0' || s[*i] > '9' {
		return false
	}
	for *i < len(s) && s[*i] >= '0' && s[*i] <= '9' {
		*i++
	}
	return true
}

func (l *lexer) scanKeyword(expected string, kind tokenKind, start Position) token {
	end := start.Offset + len(expected)
	if end > len(l.input) || l.input[start.Offset:end] != expected {
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
		text: l.input[start.Offset:l.pos],
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
		text: l.input[start.Offset:l.pos],
		pos:  start,
		end:  l.tokPos(),
	}
}

func (l *lexer) errorToken(msg string) token {
	pos := l.tokPos()
	return token{kind: tokError, text: msg, pos: pos, end: pos}
}
