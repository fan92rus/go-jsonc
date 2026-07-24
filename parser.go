package jsonc

import (
	"fmt"
	"strings"
)

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
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			return l.scanWhitespace()
		case c == '/':
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
			goto scanValue
		}
	}
scanValue:
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
	case c == '-' || c == '+' || (c >= '0' && c <= '9'):
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
	return token{
		kind: tokNumber,
		text: string(l.input[start.Offset:l.pos]),
		pos:  start,
		end:  l.tokPos(),
	}
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

// ---------------------------------------------------------------------------
// Parser
// ---------------------------------------------------------------------------

type parser struct {
	lex    *lexer
	tokens []token
	pos    int // position in token buffer
}

// Parse parses a JSONC document and returns its CST.
func Parse(input []byte) (*Node, error) {
	p := &parser{lex: newLexer(input)}
	doc := p.parseDocument()
	if doc == nil {
		return nil, fmt.Errorf("parse failed")
	}
	return doc, nil
}

func (p *parser) peek() token {
	if p.pos >= len(p.tokens) {
		tok := p.lex.next()
		p.tokens = append(p.tokens, tok)
	}
	return p.tokens[p.pos]
}

func (p *parser) advance() token {
	tok := p.peek()
	p.pos++
	return tok
}

func (p *parser) parseDocument() *Node {
	doc := &Node{Kind: KindDocument, Start: Position{Offset: 0}}
	// Consume all tokens until EOF
	for p.peek().kind != tokEOF {
		tok := p.peek()
		switch tok.kind {
		case tokLBrace:
			doc.Children = append(doc.Children, p.parseObject())
		case tokLBracket:
			doc.Children = append(doc.Children, p.parseArray())
		case tokCommentLine, tokCommentBlock, tokWhitespace:
			doc.Children = append(doc.Children, p.tokToNode(p.advance()))
		default:
			// Single top-level value
			val := p.parseValue()
			if val != nil {
				doc.Children = append(doc.Children, val)
			} else {
				doc.Children = append(doc.Children, p.tokToNode(p.advance()))
			}
		}
	}
	if len(p.lex.input) > 0 {
		doc.End = Position{Offset: len(p.lex.input)}
	}
	return doc
}

func (p *parser) parseValue() *Node {
	tok := p.peek()
	switch tok.kind {
	case tokLBrace:
		return p.parseObject()
	case tokLBracket:
		return p.parseArray()
	case tokString, tokNumber, tokTrue, tokFalse, tokNull:
		return p.tokToNode(p.advance())
	case tokEOF:
		return nil
	default:
		return nil
	}
}

func (p *parser) parseObject() *Node {
	obj := &Node{Kind: KindObject}
	obj.Children = append(obj.Children, p.tokToNode(p.advance())) // {

	for {
		tok := p.peek()
		switch {
		case tok.kind == tokEOF:
			obj.End = Position{Offset: len(p.lex.input)}
			return obj
		case tok.kind == tokRBrace:
			obj.Children = append(obj.Children, p.tokToNode(p.advance()))
			obj.Start = obj.Children[0].Start
			obj.End = obj.Children[len(obj.Children)-1].End
			return obj
		case tok.kind == tokWhitespace || tok.kind == tokCommentLine || tok.kind == tokCommentBlock:
			obj.Children = append(obj.Children, p.tokToNode(p.advance()))
		case tok.kind == tokComma:
			obj.Children = append(obj.Children, p.tokToNode(p.advance()))
		case tok.kind == tokString:
			member := p.parseMember()
			if member != nil {
				obj.Children = append(obj.Children, member)
			}
		default:
			obj.Children = append(obj.Children, p.errNode("expected key or }"))
			obj.End = Position{Offset: len(p.lex.input)}
			return obj
		}
	}
}

func (p *parser) parseMember() *Node {
	member := &Node{Kind: KindMember}
	member.Children = append(member.Children, p.tokToNode(p.advance())) // key

	// Skip trivia before colon
	for p.peek().kind == tokWhitespace || p.peek().kind == tokCommentLine || p.peek().kind == tokCommentBlock {
		member.Children = append(member.Children, p.tokToNode(p.advance()))
	}

	// Colon
	if p.peek().kind == tokColon {
		member.Children = append(member.Children, p.tokToNode(p.advance()))
	} else {
		member.Children = append(member.Children, p.errNode("expected :"))
		return member
	}

	// Skip trivia after colon
	for p.peek().kind == tokWhitespace || p.peek().kind == tokCommentLine || p.peek().kind == tokCommentBlock {
		member.Children = append(member.Children, p.tokToNode(p.advance()))
	}

	// Value
	val := p.parseValue()
	if val != nil {
		member.Children = append(member.Children, val)
	} else {
		member.Children = append(member.Children, p.errNode("expected value"))
	}

	member.Start = member.Children[0].Start
	member.End = member.Children[len(member.Children)-1].End
	return member
}

func (p *parser) parseArray() *Node {
	arr := &Node{Kind: KindArray}
	arr.Children = append(arr.Children, p.tokToNode(p.advance())) // [

	for {
		tok := p.peek()
		switch {
		case tok.kind == tokEOF:
			arr.End = Position{Offset: len(p.lex.input)}
			return arr
		case tok.kind == tokRBracket:
			arr.Children = append(arr.Children, p.tokToNode(p.advance()))
			arr.Start = arr.Children[0].Start
			arr.End = arr.Children[len(arr.Children)-1].End
			return arr
		case tok.kind == tokWhitespace || tok.kind == tokCommentLine || tok.kind == tokCommentBlock:
			arr.Children = append(arr.Children, p.tokToNode(p.advance()))
		case tok.kind == tokComma:
			arr.Children = append(arr.Children, p.tokToNode(p.advance()))
		default:
			val := p.parseValue()
			if val != nil {
				arr.Children = append(arr.Children, val)
			} else if tok.kind == tokError {
				arr.Children = append(arr.Children, p.errNode(tok.text))
				p.advance()
			} else {
				arr.Children = append(arr.Children, p.errNode("expected value or ]"))
				return arr
			}
		}
	}
}

func (p *parser) tokToNode(tok token) *Node {
	n := &Node{
		Value: tok.text,
		Start: tok.pos,
		End:   tok.end,
	}
	switch tok.kind {
	case tokLBrace:
		n.Kind = KindLBrace
	case tokRBrace:
		n.Kind = KindRBrace
	case tokLBracket:
		n.Kind = KindLBracket
	case tokRBracket:
		n.Kind = KindRBracket
	case tokColon:
		n.Kind = KindColon
	case tokComma:
		n.Kind = KindComma
	case tokString:
		n.Kind = KindString
	case tokNumber:
		n.Kind = KindNumber
	case tokTrue, tokFalse:
		n.Kind = KindBoolean
	case tokNull:
		n.Kind = KindNull
	case tokWhitespace:
		n.Kind = KindWhitespace
	case tokCommentLine:
		n.Kind = KindComment
		n.CommentStyle = CommentLine
		body := strings.TrimPrefix(tok.text, "//")
		body = strings.TrimPrefix(body, " ")
		n.CommentBody = strings.TrimRight(body, "\n\r")
	case tokCommentBlock:
		n.Kind = KindComment
		n.CommentStyle = CommentBlock
		body := strings.TrimPrefix(tok.text, "/*")
		body = strings.TrimSuffix(body, "*/")
		n.CommentBody = strings.TrimSpace(body)
	case tokError:
		n.Kind = KindError
	}
	return n
}

func (p *parser) errNode(msg string) *Node {
	return &Node{
		Kind:  KindError,
		Value: msg,
		Start: Position{Offset: p.pos},
		End:   Position{Offset: p.pos},
	}
}

// ---------------------------------------------------------------------------
// Serialization
// ---------------------------------------------------------------------------

// Serialize converts a CST back to its source text.
func Serialize(doc *Node) string {
	if doc == nil {
		return ""
	}
	var sb strings.Builder
	serializeNode(doc, &sb)
	return sb.String()
}

func serializeNode(n *Node, sb *strings.Builder) {
	if n == nil {
		return
	}
	if len(n.Children) > 0 {
		for _, c := range n.Children {
			serializeNode(c, sb)
		}
	} else {
		sb.WriteString(n.Value)
	}
}

// ---------------------------------------------------------------------------
// Formatting
// ---------------------------------------------------------------------------

// FormatOptions controls pretty-printing behaviour.
type FormatOptions struct {
	Indent string // indent string (default: two spaces)
}

// Format pretty-prints a CST.
func Format(doc *Node, opts *FormatOptions) string {
	if doc == nil {
		return ""
	}
	if opts == nil || opts.Indent == "" {
		opts = &FormatOptions{Indent: "  "}
	}
	var sb strings.Builder
	formatNode(doc, &sb, opts, 0)
	return sb.String()
}

func formatNode(n *Node, sb *strings.Builder, opts *FormatOptions, depth int) {
	if n == nil {
		return
	}
	if len(n.Children) > 0 {
		for _, c := range n.Children {
			formatNode(c, sb, opts, depth)
		}
	} else {
		sb.WriteString(n.Value)
	}
}

// Exported API verification
var (
	_ = Parse
	_ = Serialize
	_ = Format
)
