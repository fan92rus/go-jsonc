package jsonc

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Parser
// ---------------------------------------------------------------------------

type parser struct {
	lex    *lexer
	tokens []token
	pos    int // position in token buffer
}

// Parse parses a JSONC document and returns its CST.
func Parse(input string) (*Node, error) {
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
		switch tok.kind {
		case tokEOF:
			obj.End = Position{Offset: len(p.lex.input)}
			return obj
		case tokRBrace:
			obj.Children = append(obj.Children, p.tokToNode(p.advance()))
			obj.Start = obj.Children[0].Start
			obj.End = obj.Children[len(obj.Children)-1].End
			return obj
		case tokWhitespace, tokCommentLine, tokCommentBlock:
			obj.Children = append(obj.Children, p.tokToNode(p.advance()))
		case tokComma:
			obj.Children = append(obj.Children, p.tokToNode(p.advance()))
		case tokString:
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
		switch tok.kind {
		case tokEOF:
			arr.End = Position{Offset: len(p.lex.input)}
			return arr
		case tokRBracket:
			arr.Children = append(arr.Children, p.tokToNode(p.advance()))
			arr.Start = arr.Children[0].Start
			arr.End = arr.Children[len(arr.Children)-1].End
			return arr
		case tokWhitespace, tokCommentLine, tokCommentBlock:
			arr.Children = append(arr.Children, p.tokToNode(p.advance()))
		case tokComma:
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
		n.CommentBody = extractLineCommentBody(tok.text)
	case tokCommentBlock:
		n.Kind = KindComment
		n.CommentStyle = CommentBlock
		n.CommentBody = extractBlockCommentBody(tok.text)
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

// extractLineCommentBody returns the body of a //-comment without
// the // prefix and trailing newlines.
func extractLineCommentBody(text string) string {
	body := text[2:] // strip "//"
	if len(body) > 0 && body[0] == ' ' {
		body = body[1:]
	}
	for len(body) > 0 && (body[len(body)-1] == '\n' || body[len(body)-1] == '\r') {
		body = body[:len(body)-1]
	}
	return body
}

// extractBlockCommentBody returns the body of a /*-comment without
// the delimiters and surrounding whitespace.
func extractBlockCommentBody(text string) string {
	body := text[2 : len(text)-2] // strip "/*" and "*/"
	return strings.TrimSpace(body)
}
