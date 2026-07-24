package jsonc

import (
	"fmt"
	"strings"
)

// NodeKind identifies the type of a CST node.
type NodeKind int

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

// FindAllComments returns all comment nodes in the tree.
func (n *Node) FindAllComments() []*Node {
	return n.FindAll(KindComment)
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
func (n *Node) ValueNode() *Node {
	if n == nil || n.Kind != KindMember {
		return nil
	}
	for _, c := range n.Children {
		if c.IsValue() {
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
	sb.WriteString(fmt.Sprintf("%s%s%s @%s\n", pad, n.Kind, extra, n.Start))
	for _, c := range n.Children {
		c.writeString(sb, indent+1)
	}
}
