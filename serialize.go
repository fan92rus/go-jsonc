package jsonc

import "strings"

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
