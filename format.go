package jsonc

import "strings"

// ---------------------------------------------------------------------------
// Formatting
// ---------------------------------------------------------------------------

// FormatOptions controls pretty-printing behaviour.
//
// Indent specifies the per-level indentation string. Examples:
//   - "  "  — two-space indent (default)
//   - "\t"  — tab indent
//   - "    " — four-space indent
//   - ""    — compact/minified output (no indentation)
//
// Zero value (FormatOptions{}) produces compact output because Indent="" is
// the empty string. To get the default two-space indent, pass
//
//	&FormatOptions{Indent: "  "}
//
// or nil (Format(nil) uses two spaces).
type FormatOptions struct {
	Indent string
}

// Format pretty-prints a CST.
func Format(doc *Node, opts *FormatOptions) string {
	if doc == nil {
		return ""
	}
	if opts == nil {
		opts = &FormatOptions{Indent: "  "}
	}
	var sb strings.Builder
	fmtNode(doc, &sb, opts, 0, false, false)
	return sb.String()
}

// fmtNode recursively formats a CST node.
// inLine: if true, don't add newlines for this container.
// afterComma: true if we just emitted a comma (no leading indent needed).
func fmtNode(n *Node, sb *strings.Builder, opts *FormatOptions, depth int, inLine bool, afterComma bool) {
	if n == nil {
		return
	}

	switch n.Kind {
	case KindObject, KindArray:
		fmtContainer(n, sb, opts, depth, inLine, afterComma)
	case KindMember:
		fmtMember(n, sb, opts, depth, afterComma, inLine)
	case KindDocument:
		// Document: render children directly
		for _, c := range n.Children {
			fmtNode(c, sb, opts, depth, false, false)
		}
	case KindLBrace, KindRBrace, KindLBracket, KindRBracket:
		sb.WriteString(n.Value)
	case KindComma:
		sb.WriteString(n.Value)
	case KindColon:
		sb.WriteString(": ")
	case KindString, KindNumber, KindBoolean, KindNull:
		sb.WriteString(n.Value)
	case KindWhitespace:
		// Strip original whitespace; the formatter manages its own spacing.
		// Comments handle their own newlines.
	case KindComment:
		fmtComment(n, sb, opts, depth, afterComma)
	case KindError:
		sb.WriteString(n.Value)
	default:
		// Leaf: pass through
		if len(n.Children) == 0 {
			sb.WriteString(n.Value)
		} else {
			for _, c := range n.Children {
				fmtNode(c, sb, opts, depth, false, false)
			}
		}
	}
}

func fmtContainer(n *Node, sb *strings.Builder, opts *FormatOptions, depth int, inLine bool, _ bool) {
	children := filterNonTriviaCST(n.Children)
	hasMembers := false
	hasComments := false
	for _, c := range children {
		if c.Kind == KindMember || c.IsValue() {
			hasMembers = true
		}
	}
	for _, c := range n.Children {
		if c.Kind == KindComment {
			hasComments = true
			break
		}
	}

	singleLine := !hasComments && (!hasMembers || inLine)

	// Opening bracket
	lb := n.FirstChildOfKind(KindLBrace, KindLBracket)
	if lb != nil {
		sb.WriteString(lb.Value)
	}

	if singleLine {
		fmtContainerCompact(n, sb, opts, depth)
	} else {
		fmtContainerMulti(n, sb, opts, depth)
	}

	// Closing bracket
	rb := n.FirstChildOfKind(KindRBrace, KindRBracket)
	if rb != nil {
		sb.WriteString(rb.Value)
	}
}

// fmtContainerCompact renders a container on a single line.
func fmtContainerCompact(n *Node, sb *strings.Builder, opts *FormatOptions, depth int) {
	needSep := false
	for _, c := range n.Children {
		switch c.Kind {
		case KindLBrace, KindLBracket:
			continue
		case KindRBrace, KindRBracket:
			continue
		case KindWhitespace:
			continue
		case KindComma:
			needSep = false // comma already includes text
		default:
			if needSep {
				sb.WriteString(", ")
			}
			fmtNode(c, sb, opts, depth+1, true, needSep)
			needSep = true
		}
	}
}

// writeIndentIfNeeded writes a newline and indentation for the multi-line
// container path, skipping the leading newline if we are already at the
// start of a line (avoids double newlines from line-comment trailing \n).
func writeIndentIfNeeded(sb *strings.Builder, indent string, depth int, needNewline bool) {
	if !needNewline {
		return
	}
	atLineStart := sb.Len() > 0 && sb.String()[sb.Len()-1] == '\n'
	if !atLineStart {
		sb.WriteString("\n")
	}
	sb.WriteString(strings.Repeat(indent, depth))
}

// fmtContainerMulti renders a container with multi-line layout.
func fmtContainerMulti(n *Node, sb *strings.Builder, opts *FormatOptions, depth int) {
	needNewline := true
	for _, c := range n.Children {
		switch c.Kind {
		case KindLBrace, KindLBracket:
			continue
		case KindRBrace, KindRBracket:
			continue
		case KindWhitespace:
			continue
		case KindComma:
			sb.WriteString(",")
			needNewline = true
		case KindComment:
			writeIndentIfNeeded(sb, opts.Indent, depth+1, needNewline)
			fmtComment(c, sb, opts, depth+1, needNewline)
			needNewline = true
		default:
			writeIndentIfNeeded(sb, opts.Indent, depth+1, needNewline)
			fmtNode(c, sb, opts, depth+1, false, false)
			needNewline = true
		}
	}
	if needNewline {
		atLineStart := sb.Len() > 0 && sb.String()[sb.Len()-1] == '\n'
		if !atLineStart {
			sb.WriteString("\n")
		}
		sb.WriteString(strings.Repeat(opts.Indent, depth))
	}
}

func fmtMember(n *Node, sb *strings.Builder, opts *FormatOptions, depth int, _ bool, inLine bool) {
	// Container calls us after writing the newline + indent (multi-line)
	// or directly in compact mode (no indent needed). Either way, we write
	// the key-value pair without additional indentation.
	for _, c := range n.Children {
		switch c.Kind {
		case KindString:
			sb.WriteString(c.Value)
		case KindColon:
			sb.WriteString(": ")
		case KindWhitespace:
			// skip original whitespace
		case KindComment:
			if !inLine {
				atLineStart := sb.Len() > 0 && sb.String()[sb.Len()-1] == '\n'
				if !atLineStart {
					sb.WriteString("\n")
				}
				sb.WriteString(strings.Repeat(opts.Indent, depth))
			}
			fmtComment(c, sb, opts, depth, false)
			if !inLine {
				atLineStart := sb.Len() > 0 && sb.String()[sb.Len()-1] == '\n'
				if !atLineStart {
					sb.WriteString("\n")
				}
				sb.WriteString(strings.Repeat(opts.Indent, depth))
			}
		default:
			if c.IsValue() {
				fmtNode(c, sb, opts, depth, false, false)
			} else if len(c.Children) > 0 {
				fmtNode(c, sb, opts, depth, false, false)
			} else {
				sb.WriteString(c.Value)
			}
		}
	}
}

func fmtComment(n *Node, sb *strings.Builder, _ *FormatOptions, _ int, _ bool) {
	if n.Kind != KindComment {
		return
	}
	// Line comments MUST be followed by a newline in valid JSONC,
	// otherwise the next token becomes part of the comment body.
	sb.WriteString(n.Value)
	if n.CommentStyle == CommentLine && !strings.HasSuffix(n.Value, "\n") {
		sb.WriteString("\n")
	}
}

func filterNonTriviaCST(nodes []*Node) []*Node {
	var result []*Node
	for _, n := range nodes {
		if !n.IsTrivia() {
			result = append(result, n)
		}
	}
	return result
}
