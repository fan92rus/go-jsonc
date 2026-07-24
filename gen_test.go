// Package jsonc tests.
package jsonc

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"pgregory.net/rapid"
)

// All generators defined at package level to avoid rapid "group did not use
// any data from bitstream" errors from nested Custom generators.

// lettersAZ generates a-z runes.
var lettersAZ = rapid.Custom(func(t *rapid.T) rune {
	return []rune("abcdefghijklmnopqrstuvwxyz")[rapid.IntRange(0, 25).Draw(t, "l")]
})

// lettersAZUp generates A-Z runes.
var lettersAZUp = rapid.Custom(func(t *rapid.T) rune {
	return []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")[rapid.IntRange(0, 25).Draw(t, "u")]
})

// digitsRune generates 0-9 runes.
var digitsRune = rapid.Custom(func(t *rapid.T) rune {
	return []rune("0123456789")[rapid.IntRange(0, 9).Draw(t, "d")]
})

// alnumRune combines letters + digits.
var alnumRune = rapid.OneOf(lettersAZ, lettersAZUp, digitsRune)

// spaceRune generates ' ' always.
var spaceRune = rapid.Custom(func(t *rapid.T) rune {
	rapid.IntRange(0, 0).Draw(t, "sp") // always draw
	return ' '
})

// punctRune generates punctuation chars.
var punctRune = rapid.Custom(func(t *rapid.T) rune {
	return []rune(".,!?-_")[rapid.IntRange(0, 5).Draw(t, "p")]
})

// alphaSpaceRune generates letters + space.
var alphaSpaceRune = rapid.OneOf(lettersAZ, lettersAZUp, spaceRune)

// unicodeRune generates Greek letters and symbols.
var unicodeRune = rapid.Custom(func(t *rapid.T) rune {
	return []rune{'α', 'β', 'γ', 'δ', 'ε', '★', '♦', '♣'}[rapid.IntRange(0, 7).Draw(t, "ui")]
})

// specialRune generates special chars that need JSON escaping.
var specialRune = rapid.Custom(func(t *rapid.T) rune {
	return []rune{'"', '\\', '\b', '\f', '\n', '\r', '\t', 'a', 'b', 'c', ' '}[rapid.IntRange(0, 10).Draw(t, "si")]
})

// commentSafeRune generates chars safe inside comments.
var commentSafeRune = rapid.OneOf(lettersAZ, lettersAZUp, digitsRune,
	rapid.Custom(func(t *rapid.T) rune {
		return []rune(" .,;!?-_")[rapid.IntRange(0, 7).Draw(t, "c")]
	}))

// whitespaceRune generates \s \t \n \r.
var whitespaceRune = rapid.Custom(func(t *rapid.T) rune {
	return []rune(" \t\n\r")[rapid.IntRange(0, 3).Draw(t, "w")]
})

// genJSONString generates valid JSON string literal with surrounding quotes.
var genJSONString = rapid.Custom(func(t *rapid.T) string {
	s := genJSONRawString.Draw(t, "raw")
	var b strings.Builder
	b.WriteByte('"')
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
		case r > 0x7E:
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
		i += sz
	}
	b.WriteByte('"')
	return b.String()
})

// genJSONRawString generates raw string content without quotes.
var genJSONRawString = rapid.Custom(func(t *rapid.T) string {
	kind := rapid.IntRange(0, 8).Draw(t, "kind")
	switch kind {
	case 0, 1, 2, 3:
		return rapid.StringOf(alnumRune).Draw(t, "ascii")
	case 4, 5:
		return rapid.StringOf(rapid.OneOf(alphaSpaceRune, punctRune)).Draw(t, "punct")
	case 6:
		return rapid.StringOf(unicodeRune).Draw(t, "unicode")
	case 7:
		return rapid.StringOf(specialRune).Draw(t, "specials")
	case 8:
		return "" // empty — draws already consumed by "kind"
	}
	return ""
})

// genJSONNumber generates valid JSON number literal.
var genJSONNumber = rapid.Custom(func(t *rapid.T) string {
	kind := rapid.IntRange(0, 6).Draw(t, "numKind")
	neg := rapid.Bool().Draw(t, "neg")
	switch kind {
	case 0:
		n := rapid.Int64Range(0, 9999999).Draw(t, "int")
		s := fmt.Sprintf("%d", n)
		if neg && n != 0 {
			s = "-" + s
		}
		return s
	case 1:
		return "0"
	case 2:
		whole := rapid.Int64Range(0, 9999).Draw(t, "whole")
		frac := rapid.Int64Range(1, 999999).Draw(t, "frac")
		s := fmt.Sprintf("%d.%d", whole, frac)
		if neg {
			s = "-" + s
		}
		return s
	case 3:
		mant := rapid.Int64Range(1, 9999).Draw(t, "mant")
		exp := rapid.Int64Range(0, 50).Draw(t, "exp")
		s := fmt.Sprintf("%de%d", mant, exp)
		if neg {
			s = "-" + s
		}
		return s
	case 4:
		mant := rapid.Float64Range(-1e6, 1e6).Draw(t, "fmant")
		exp := rapid.Int64Range(0, 20).Draw(t, "fexp")
		s := fmt.Sprintf("%.5fe%d", math.Abs(mant), exp)
		if mant < 0 {
			s = "-" + s
		}
		return s
	case 5:
		frac := rapid.Int64Range(1, 999999).Draw(t, "frac")
		s := fmt.Sprintf("0.%d", frac)
		if neg {
			s = "-" + s
		}
		return s
	case 6:
		whole := rapid.Int64Range(1, 9999).Draw(t, "whole")
		s := fmt.Sprintf("%d.0", whole)
		if neg {
			s = "-" + s
		}
		return s
	}
	return "0"
})

// genWhitespace generates space/tab/newline.
var genWhitespace = rapid.Custom(func(t *rapid.T) string {
	return rapid.StringOf(whitespaceRune).Draw(t, "ws")
})

// genCommentText generates text safe for comment bodies.
var genCommentText = rapid.Custom(func(t *rapid.T) string {
	return rapid.StringOf(commentSafeRune).Draw(t, "body")
})

// genLineComment generates // ... comments (always includes trailing \n
// to prevent the comment from consuming structural tokens on the same line).
var genLineComment = rapid.Custom(func(t *rapid.T) string {
	body := ""
	if rapid.Bool().Draw(t, "hasContent") {
		body = " " + genCommentText.Draw(t, "body")
	}
	return "//" + body + "\n"
})

// genBlockComment generates /* ... */ comments.
var genBlockComment = rapid.Custom(func(t *rapid.T) string {
	body := ""
	if rapid.Bool().Draw(t, "hasContent") {
		body = " " + genCommentText.Draw(t, "body") + " "
	}
	return "/*" + body + "*/"
})

// genComment generates either line or block comment.
var genComment = rapid.Custom(func(t *rapid.T) string {
	if rapid.Bool().Draw(t, "commentMode") {
		return genLineComment.Draw(t, "lineComment")
	}
	return genBlockComment.Draw(t, "blockComment")
})

// genTrivia generates optional whitespace and/or comment.
func genTrivia(t *rapid.T) string {
	switch rapid.IntRange(0, 3).Draw(t, "triviaMode") {
	case 0:
		return genWhitespace.Draw(t, "ws")
	case 1:
		return genComment.Draw(t, "comment")
	case 2:
		return genWhitespace.Draw(t, "ws1") + genComment.Draw(t, "comment")
	case 3:
		return genComment.Draw(t, "comment") + genWhitespace.Draw(t, "ws2")
	}
	return ""
}

// genValue generates a JSON value with optional trivia.
func genValue(t *rapid.T) string {
	pre := genTrivia(t)
	post := genTrivia(t)
	var body string
	switch rapid.IntRange(0, 5).Draw(t, "valKind") {
	case 0:
		body = genJSONString.Draw(t, "str")
	case 1:
		body = genJSONNumber.Draw(t, "num")
	case 2:
		body = "true"
	case 3:
		body = "false"
	case 4:
		body = "null"
	case 5:
		body = genContainerValue(t)
	}
	return pre + body + post
}

// genContainerValue generates object or array.
func genContainerValue(t *rapid.T) string {
	if rapid.Bool().Draw(t, "containerKind") {
		return genObject(t)
	}
	return genArray(t)
}

// genMember generates "key": value.
func genMember(t *rapid.T) string {
	return genTrivia(t) + genJSONString.Draw(t, "key") +
		genTrivia(t) + ":" + genTrivia(t) +
		genValue(t) + genTrivia(t)
}

// genContainer generates a JSON container with the given brackets.
// genItem produces one element.
func genContainer(t *rapid.T, open, closeTok string, n int, genItem func(*rapid.T) string) string {
	allowTrailing := rapid.Bool().Draw(t, "trailingComma")
	var sb strings.Builder
	sb.WriteString(genTrivia(t))
	sb.WriteString(open)
	sb.WriteString(genTrivia(t))
	for i := 0; i < n; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(genItem(t))
	}
	if n > 0 && allowTrailing {
		sb.WriteString(",")
	}
	sb.WriteString(genTrivia(t))
	sb.WriteString(closeTok)
	return sb.String()
}

// genObject generates { ... }.
func genObject(t *rapid.T) string {
	n := rapid.IntRange(0, 4).Draw(t, "numMembers")
	return genContainer(t, "{", "}", n, func(t *rapid.T) string { return genMember(t) })
}

// genArray generates [ ... ].
func genArray(t *rapid.T) string {
	n := rapid.IntRange(0, 5).Draw(t, "numElems")
	return genContainer(t, "[", "]", n, func(t *rapid.T) string { return genValue(t) })
}

// genFullJSONC generates a full JSONC document with comments.
func genFullJSONC(t *rapid.T) string {
	return genTrivia(t) + genContainerValue(t) + genTrivia(t)
}

// genJSONWithoutComments generates JSON without comments (always draws).
var genJSONWithoutComments = rapid.Custom(func(t *rapid.T) string {
	return genObject(t)
})

// genNumbersString is a numeric JSON value generator.
var genNumbersString = rapid.Custom(func(t *rapid.T) string {
	return genJSONNumber.Draw(t, "num")
})
