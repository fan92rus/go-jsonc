// Package jsonc provides a concrete syntax tree (CST) parser, serializer,
// and formatter for JSONC (JSON with Comments).
package jsonc

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// NodeKind identifies the type of a CST node.
type NodeKind int

// Node kinds.
const (
	KindDocument NodeKind = iota
	KindObject
	KindArray
	KindMember // "key": value pair
	KindString
	KindNumber
	KindBoolean
	KindNull
	KindComment // line or block comment
	KindWhitespace
	KindComma
	KindColon
	KindLBrace   // {
	KindRBrace   // }
	KindLBracket // [
	KindRBracket // ]
	KindEOF      // end of input
	KindError    // parse error / unexpected token
)

func (k NodeKind) String() string {
	switch k {
	case KindDocument:
		return "Document"
	case KindObject:
		return "Object"
	case KindArray:
		return "Array"
	case KindMember:
		return "Member"
	case KindString:
		return "String"
	case KindNumber:
		return "Number"
	case KindBoolean:
		return "Boolean"
	case KindNull:
		return "Null"
	case KindComment:
		return "Comment"
	case KindWhitespace:
		return "Whitespace"
	case KindComma:
		return "Comma"
	case KindColon:
		return "Colon"
	case KindLBrace:
		return "LBrace"
	case KindRBrace:
		return "RBrace"
	case KindLBracket:
		return "LBracket"
	case KindRBracket:
		return "RBracket"
	case KindEOF:
		return "EOF"
	case KindError:
		return "Error"
	default:
		return fmt.Sprintf("Kind(%d)", int(k))
	}
}

// Position represents a byte-offset position in source text.
type Position struct {
	Offset int // 0-based byte offset from start of source
	Line   int // 0-based line number
	Column int // 0-based column (byte offset within the line)
}

// String formats a Position for display.
func (p Position) String() string {
	return fmt.Sprintf("%d:%d(%d)", p.Line+1, p.Column+1, p.Offset)
}

// CommentStyle distinguishes line comments from block comments.
type CommentStyle int

// Comment styles.
const (
	CommentLine  CommentStyle = iota // // ...
	CommentBlock                     // /* ... */
)

// Node is a single node in the Concrete Syntax Tree.
//
// A CST preserves ALL source information: values, structural tokens,
// comments, and whitespace. This enables lossless round-trip parsing
// (parse → serialize → identical output) and comment-aware formatting.
type Node struct {
	Kind     NodeKind
	Children []*Node // For container nodes (Document, Object, Array, Member)
	Value    string  // Raw source text for leaf nodes (tokens, comments, whitespace)

	// Comment-specific fields
	CommentStyle CommentStyle // Only valid when Kind == KindComment
	CommentBody  string       // Content without delimiters (// or /* */)

	Start Position // Start position in source (inclusive)
	End   Position // End position in source (exclusive)
}

// RawText reconstructs the original source text for this node.
// For leaf nodes, returns Value directly.
// For container nodes, concatenates children's RawText.
func (n *Node) RawText() string {
	if n == nil {
		return ""
	}
	if len(n.Children) > 0 {
		var sb strings.Builder
		for _, c := range n.Children {
			sb.WriteString(c.RawText())
		}
		return sb.String()
	}
	return n.Value
}

// IsContainer returns true for nodes that can have child value nodes.
func (n *Node) IsContainer() bool {
	switch n.Kind {
	case KindDocument, KindObject, KindArray, KindMember:
		return true
	}
	return false
}

// IsValue returns true for nodes that represent JSON values.
func (n *Node) IsValue() bool {
	switch n.Kind {
	case KindString, KindNumber, KindBoolean, KindNull, KindObject, KindArray:
		return true
	}
	return false
}

// IsTrivia returns true for whitespace and comment nodes.
func (n *Node) IsTrivia() bool {
	switch n.Kind {
	case KindWhitespace, KindComment:
		return true
	}
	return false
}

// Walk traverses the tree depth-first, calling fn for every node.
// If fn returns false, the walk stops descending into that node's children.
func (n *Node) Walk(fn func(*Node) bool) {
	if n == nil {
		return
	}
	if !fn(n) {
		return
	}
	for _, c := range n.Children {
		c.Walk(fn)
	}
}

// FindAll returns all nodes matching the given kind.
func (n *Node) FindAll(kind NodeKind) []*Node {
	var result []*Node
	n.Walk(func(node *Node) bool {
		if node.Kind == kind {
			result = append(result, node)
		}
		return true
	})
	return result
}

// FirstChild returns the first non-trivia child node, or nil.
// In a CST, the first child may be whitespace or a comment;
// FirstChild skips those and returns the first meaningful node.
func (n *Node) FirstChild() *Node {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if !c.IsTrivia() {
			return c
		}
	}
	return nil
}

// FirstChildOfKind returns the first child matching any of the given kinds.
func (n *Node) FirstChildOfKind(kinds ...NodeKind) *Node {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		for _, k := range kinds {
			if c.Kind == k {
				return c
			}
		}
	}
	return nil
}

// ValueNode returns the value child of a Member node, or nil.
// The value is the node after the colon — this skips the key (also a KindString).
func (n *Node) ValueNode() *Node {
	if n == nil || n.Kind != KindMember {
		return nil
	}
	pastColon := false
	for _, c := range n.Children {
		if c.Kind == KindColon {
			pastColon = true
			continue
		}
		if pastColon && c.IsValue() {
			return c
		}
	}
	return nil
}

// KeyNode returns the key (string) child of a Member node, or nil.
func (n *Node) KeyNode() *Node {
	if n == nil || n.Kind != KindMember {
		return nil
	}
	for _, c := range n.Children {
		if c.Kind == KindString {
			return c
		}
	}
	return nil
}

// Members returns the member nodes of an Object, or nil.
func (n *Node) Members() []*Node {
	if n == nil || n.Kind != KindObject {
		return nil
	}
	var members []*Node
	for _, c := range n.Children {
		if c.Kind == KindMember {
			members = append(members, c)
		}
	}
	return members
}

// Elements returns the value element nodes of an Array, or nil.
func (n *Node) Elements() []*Node {
	if n == nil || n.Kind != KindArray {
		return nil
	}
	var elems []*Node
	for _, c := range n.Children {
		if c.IsValue() {
			elems = append(elems, c)
		}
	}
	return elems
}

// DeepEqual checks structural equality: same kind, same value, same children.
func (n *Node) DeepEqual(other *Node) bool {
	if n == nil || other == nil {
		return n == other
	}
	if n.Kind != other.Kind {
		return false
	}
	if n.Value != other.Value {
		return false
	}
	if n.CommentStyle != other.CommentStyle {
		return false
	}
	if n.CommentBody != other.CommentBody {
		return false
	}
	if len(n.Children) != len(other.Children) {
		return false
	}
	for i := range n.Children {
		if !n.Children[i].DeepEqual(other.Children[i]) {
			return false
		}
	}
	return true
}

// String returns a human-readable debug representation of the node tree.
func (n *Node) String() string {
	var sb strings.Builder
	n.writeString(&sb, 0)
	return sb.String()
}

// ---------------------------------------------------------------------------
// Builder API — create and mutate CST nodes programmatically
// ---------------------------------------------------------------------------

// NewCommentLine creates a new line comment node.
// The body is the text after "// ". Leading/trailing whitespace is trimmed
// from the body to avoid double spacing.
func NewCommentLine(body string) *Node {
	body = strings.TrimSpace(body)
	val := "//"
	if body != "" {
		val += " " + body
	}
	return &Node{
		Kind:         KindComment,
		Value:        val,
		CommentStyle: CommentLine,
		CommentBody:  body,
	}
}

// NewCommentBlock creates a new block comment node.
// The body is the text inside "/*  */". Leading/trailing whitespace
// is trimmed from the body.
func NewCommentBlock(body string) *Node {
	body = strings.TrimSpace(body)
	val := "/**/"
	if body != "" {
		val = "/* " + body + " */"
	}
	return &Node{
		Kind:         KindComment,
		Value:        val,
		CommentStyle: CommentBlock,
		CommentBody:  body,
	}
}

// NewString creates a new JSON string value node.
// The value is the raw JSON literal including surrounding quotes.
func NewString(value string) *Node {
	if len(value) == 0 || value[0] != '"' {
		value = `"` + escapeJSON(value) + `"`
	}
	return &Node{Kind: KindString, Value: value}
}

// NewNumber creates a new JSON number value node.
func NewNumber(value string) *Node {
	return &Node{Kind: KindNumber, Value: value}
}

// NewBoolean creates a new JSON boolean value node.
func NewBoolean(val bool) *Node {
	if val {
		return &Node{Kind: KindBoolean, Value: "true"}
	}
	return &Node{Kind: KindBoolean, Value: "false"}
}

// NewNull creates a new JSON null value node.
func NewNull() *Node {
	return &Node{Kind: KindNull, Value: "null"}
}

// NewObject creates a new JSON object node with the given children.
// NewObject creates a new JSON object CST node.
// Commas are automatically inserted between the children.
func NewObject(items ...*Node) *Node {
	nodes := make([]*Node, 0, len(items)*2+2)
	nodes = append(nodes, &Node{Kind: KindLBrace, Value: "{"})
	for i, item := range items {
		if i > 0 {
			nodes = append(nodes, &Node{Kind: KindComma, Value: ","})
		}
		nodes = append(nodes, item)
	}
	nodes = append(nodes, &Node{Kind: KindRBrace, Value: "}"})
	return &Node{Kind: KindObject, Children: nodes}
}

// NewArray creates a new JSON array CST node.
// Commas are automatically inserted between the elements.
func NewArray(elements ...*Node) *Node {
	nodes := make([]*Node, 0, len(elements)*2+2)
	nodes = append(nodes, &Node{Kind: KindLBracket, Value: "["})
	for i, elem := range elements {
		if i > 0 {
			nodes = append(nodes, &Node{Kind: KindComma, Value: ","})
		}
		nodes = append(nodes, elem)
	}
	nodes = append(nodes, &Node{Kind: KindRBracket, Value: "]"})
	return &Node{Kind: KindArray, Children: nodes}
}

// NewMember creates a new JSON object member CST node ("key": value).
// key must be a KindString node or a string literal.
// val must be a value node (KindString, KindNumber, KindBoolean, KindNull,
// KindObject, KindArray).
// Extra nodes (comments, whitespace) are placed between the colon and the
// value — useful for inline comments.
func NewMember(key interface{}, val *Node, extra ...*Node) *Node {
	var keyNode *Node
	switch k := key.(type) {
	case *Node:
		keyNode = k
	case string:
		keyNode = NewString(k)
	}
	children := []*Node{keyNode}
	children = append(children, &Node{Kind: KindColon, Value: ":"})
	children = append(children, &Node{Kind: KindWhitespace, Value: " "})
	children = append(children, extra...)
	children = append(children, val)
	return &Node{Kind: KindMember, Children: children}
}

// SetValue sets a leaf node's text value and updates the End position.
func (n *Node) SetValue(text string) {
	n.Value = text
	if len(text) > 0 {
		n.End = Position{Offset: n.Start.Offset + len(text)}
	}
}

// SetCommentBody updates the body of a comment node and reconstructs
// its raw text (Value). Leading/trailing whitespace is trimmed.
func (n *Node) SetCommentBody(body string) {
	if n.Kind != KindComment {
		return
	}
	body = strings.TrimSpace(body)
	n.CommentBody = body
	switch n.CommentStyle {
	case CommentLine:
		val := "//"
		if body != "" {
			val += " " + body
		}
		n.Value = val
	case CommentBlock:
		val := "/**/"
		if body != "" {
			val = "/* " + body + " */"
		}
		n.Value = val
	}
}

// AppendChild appends a child node and updates Start/End positions.
func (n *Node) AppendChild(child *Node) {
	n.Children = append(n.Children, child)
	if len(n.Children) == 1 {
		n.Start = child.Start
	}
	n.End = child.End
}

// Body returns the body of a comment node.
// Returns empty string for non-comment nodes.
func (n *Node) Body() string {
	if n.Kind != KindComment {
		return ""
	}
	return n.CommentBody
}

// escapeJSON escapes a plain string for use as a JSON string literal.
func escapeJSON(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		r, sz := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == '"' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\b':
			b.WriteString("\\b")
		case r == '\f':
			b.WriteString("\\f")
		case r == '\n':
			b.WriteString("\\n")
		case r == '\r':
			b.WriteString("\\r")
		case r == '\t':
			b.WriteString("\\t")
		case r < 0x20:
			fmt.Fprintf(&b, "\\u%04x", r)
		default:
			b.WriteRune(r)
		}
		i += sz
	}
	return b.String()
}

func (n *Node) writeString(sb *strings.Builder, indent int) {
	pad := strings.Repeat("  ", indent)
	if n == nil {
		sb.WriteString(pad + "<nil>\n")
		return
	}
	extra := ""
	if n.Value != "" && len(n.Children) == 0 {
		extra = fmt.Sprintf(" %q", n.Value)
	}
	if n.Kind == KindComment {
		extra = fmt.Sprintf(" style=%d %q", n.CommentStyle, n.CommentBody)
	}
	fmt.Fprintf(sb, "%s%s%s @%s\n", pad, n.Kind, extra, n.Start)
	for _, c := range n.Children {
		c.writeString(sb, indent+1)
	}
}
