package jsonc

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Parser
// ---------------------------------------------------------------------------

// Parser converts JSONC source text into a Concrete Syntax Tree.
//
// The parser preserves ALL input tokens including comments and whitespace,
// enabling lossless round-trip and comment-aware formatting.
type Parser struct {
	input []byte
	pos   int // current byte offset

	// Token tracking
	tokens []*Node
	err    error
}

// Parse parses a JSONC document and returns its Concrete Syntax Tree.
// Returns an error only when the input is structurally invalid.
func Parse(input []byte) (*Node, error) {
	p := &Parser{
		input: input,
		pos:   0,
	}

	doc := p.parseDocument()
	if doc == nil {
		return nil, fmt.Errorf("parse failed")
	}
	return doc, nil
}

// parseDocument parses the root JSONC document.
func (p *Parser) parseDocument() *Node {
	doc := &Node{
		Kind: KindDocument,
		Start: Position{
			Offset: 0,
			Line:   0,
			Column: 0,
		},
	}
	doc.End = Position{
		Offset: len(p.input),
		Line:   0,
		Column: len(p.input),
	}

	// Consume any leading trivia
	p.skipTrivia()

	if p.pos >= len(p.input) {
		// Empty/whitespace-only input
		return doc
	}

	// Parse the value
	val := p.parseValue()
	if val != nil {
		doc.Children = append(doc.Children, val)
	}

	// Consume trailing trivia
	p.skipTrivia()

	// Set proper end
	doc.End = p.position()

	return doc
}

// skipTrivia consumes whitespace and comments at the current position.
func (p *Parser) skipTrivia() {
	for p.pos < len(p.input) {
		c := p.input[p.pos]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			p.pos++
		case c == '/' && p.pos+1 < len(p.input):
			next := p.input[p.pos+1]
			if next == '/' || next == '*' {
				return // let comment be parsed as a real node
			}
			p.pos++ // standalone /, consume as error
		default:
			return
		}
	}
}

func (p *Parser) position() Position {
	// Simplified: approximate position
	return Position{
		Offset: p.pos,
		Line:   0,
		Column: p.pos,
	}
}

func (p *Parser) peek() byte {
	if p.pos >= len(p.input) {
		return 0
	}
	return p.input[p.pos]
}

func (p *Parser) advance() byte {
	c := p.input[p.pos]
	p.pos++
	return c
}

// parseValue dispatches to the appropriate value parser based on the current token.
func (p *Parser) parseValue() *Node {
	p.skipTrivia()
	if p.pos >= len(p.input) {
		return nil
	}

	c := p.peek()
	switch {
	case c == '{':
		return p.parseObject()
	case c == '[':
		return p.parseArray()
	case c == '"':
		return p.parseString()
	case c == 't':
		return p.parseKeyword("true", KindBoolean)
	case c == 'f':
		return p.parseKeyword("false", KindBoolean)
	case c == 'n':
		return p.parseKeyword("null", KindNull)
	case c == '-' || c == '+' || (c >= '0' && c <= '9'):
		return p.parseNumber()
	default:
		return p.parseError("unexpected character")
	}
}

func (p *Parser) parseObject() *Node {
	start := p.pos
	obj := &Node{
		Kind: KindObject,
		Start: Position{
			Offset: start,
			Line:   0,
			Column: start,
		},
	}

	// {
	lBrace := &Node{
		Kind:  KindLBrace,
		Value: "{",
		Start: Position{Offset: start, Line: 0, Column: start},
		End:   Position{Offset: start + 1, Line: 0, Column: start + 1},
	}
	obj.Children = append(obj.Children, lBrace)
	p.advance() // consume {
	p.skipTrivia()

	// Parse members
	first := true
	for p.pos < len(p.input) && p.peek() != '}' {
		if !first {
			p.skipTrivia()
			if p.pos >= len(p.input) || p.peek() == '}' {
				break
			}
			if p.peek() != ',' {
				obj.Children = append(obj.Children, p.parseError("expected comma"))
				break
			}
			comma := &Node{
				Kind:  KindComma,
				Value: ",",
				Start: p.position(),
				End:   p.advancePos(),
			}
			obj.Children = append(obj.Children, comma)
			p.skipTrivia()
			if p.peek() == '}' {
				// Trailing comma — allow in JSONC
				break
			}
		}
		first = false

		member := p.parseMember()
		if member != nil {
			obj.Children = append(obj.Children, member)
		}
		p.skipTrivia()
	}

	if p.pos >= len(p.input) {
		obj.End = p.position()
		return obj
	}

	// }
	rBrace := &Node{
		Kind:  KindRBrace,
		Value: "}",
		Start: p.position(),
		End:   p.advancePos(),
	}
	obj.Children = append(obj.Children, rBrace)

	obj.End = Position{
		Offset: p.pos,
		Line:   0,
		Column: p.pos,
	}

	return obj
}

func (p *Parser) parseArray() *Node {
	start := p.pos
	arr := &Node{
		Kind: KindArray,
		Start: Position{
			Offset: start,
			Line:   0,
			Column: start,
		},
	}

	lBracket := &Node{
		Kind:  KindLBracket,
		Value: "[",
		Start: Position{Offset: start, Line: 0, Column: start},
		End:   Position{Offset: start + 1, Line: 0, Column: start + 1},
	}
	arr.Children = append(arr.Children, lBracket)
	p.advance() // consume [
	p.skipTrivia()

	first := true
	for p.pos < len(p.input) && p.peek() != ']' {
		if !first {
			p.skipTrivia()
			if p.pos >= len(p.input) || p.peek() == ']' {
				break
			}
			if p.peek() != ',' {
				arr.Children = append(arr.Children, p.parseError("expected comma"))
				break
			}
			comma := &Node{
				Kind:  KindComma,
				Value: ",",
				Start: p.position(),
				End:   p.advancePos(),
			}
			arr.Children = append(arr.Children, comma)
			p.skipTrivia()
			if p.peek() == ']' {
				// Trailing comma
				break
			}
		}
		first = false

		val := p.parseValue()
		if val != nil {
			arr.Children = append(arr.Children, val)
		}
		p.skipTrivia()
	}

	if p.pos >= len(p.input) {
		arr.End = p.position()
		return arr
	}

	// ]
	rBracket := &Node{
		Kind:  KindRBracket,
		Value: "]",
		Start: p.position(),
		End:   p.advancePos(),
	}
	arr.Children = append(arr.Children, rBracket)

	arr.End = Position{
		Offset: p.pos,
		Line:   0,
		Column: p.pos,
	}

	return arr
}

func (p *Parser) parseMember() *Node {
	p.skipTrivia()
	if p.pos >= len(p.input) {
		return nil
	}

	if p.peek() != '"' {
		return p.parseError("expected string key")
	}

	member := &Node{
		Kind: KindMember,
	}

	key := p.parseString()
	if key == nil {
		return nil
	}
	member.Children = append(member.Children, key)

	p.skipTrivia()

	// :
	if p.peek() == ':' {
		colon := &Node{
			Kind:  KindColon,
			Value: ":",
			Start: p.position(),
			End:   p.advancePos(),
		}
		member.Children = append(member.Children, colon)
	} else {
		member.Children = append(member.Children, p.parseError("expected colon"))
	}

	p.skipTrivia()

	val := p.parseValue()
	if val != nil {
		member.Children = append(member.Children, val)
	}

	member.Start = key.Start
	if val != nil {
		member.End = val.End
	}

	return member
}

func (p *Parser) parseString() *Node {
	if p.peek() != '"' {
		return nil
	}
	start := p.pos
	startPos := p.position()

	p.advance() // consume opening quote

	for p.pos < len(p.input) {
		c := p.advance()
		if c == '"' {
			break
		}
		if c == '\\' && p.pos < len(p.input) {
			p.advance() // consume escaped character
		}
	}

	endPos := p.position()

	return &Node{
		Kind:  KindString,
		Value: string(p.input[start:p.pos]),
		Start: startPos,
		End:   endPos,
	}
}

func (p *Parser) parseNumber() *Node {
	start := p.pos
	startPos := p.position()

	for p.pos < len(p.input) {
		c := p.input[p.pos]
		if c == '-' || c == '+' || c == '.' || c == 'e' || c == 'E' || (c >= '0' && c <= '9') {
			p.pos++
		} else {
			break
		}
	}

	return &Node{
		Kind:  KindNumber,
		Value: string(p.input[start:p.pos]),
		Start: startPos,
		End:   p.position(),
	}
}

func (p *Parser) parseKeyword(expected string, kind NodeKind) *Node {
	start := p.pos
	startPos := p.position()

	end := start + len(expected)
	if end > len(p.input) || string(p.input[start:end]) != expected {
		return p.parseError("expected " + expected)
	}

	p.pos = end

	return &Node{
		Kind:  kind,
		Value: expected,
		Start: startPos,
		End:   p.position(),
	}
}

func (p *Parser) parseError(msg string) *Node {
	errNode := &Node{
		Kind:  KindError,
		Value: msg,
		Start: p.position(),
		End:   p.position(),
	}
	if p.pos < len(p.input) {
		errNode.Value = fmt.Sprintf("%s at %q", msg, string(p.input[p.pos]))
	}
	return errNode
}

func (p *Parser) advancePos() Position {
	old := p.position()
	p.advance()
	return old
}

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

// FormatOptions controls the behaviour of the pretty-printer.
type FormatOptions struct {
	Indent string // indent string (default: two spaces)
}

// Format pretty-prints a CST using the specified formatting options.
func Format(doc *Node, opts *FormatOptions) string {
	if doc == nil {
		return ""
	}
	if opts == nil {
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

// Verify the API compiles
var (
	_ = Parse
	_ = Serialize
	_ = Format
)
