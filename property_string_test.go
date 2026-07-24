// Package jsonc tests — string handling and final coverage gaps
package jsonc

import (
	"strings"
	"testing"
)

// ============================================================================
// escapeJSON coverage: all escape branches
// ============================================================================

// propEscapeJSONQuote verifies double-quote encoding.
func TestProperty_EscapeJSONQuote(t *testing.T) {
	result := escapeJSON(`a"b`)
	if result != `a\"b` {
		t.Fatalf("expected a\\\"b, got %s", result)
	}
}

// propEscapeJSONBackslash verifies backslash encoding.
func TestProperty_EscapeJSONBackslash(t *testing.T) {
	result := escapeJSON(`a\b`)
	if result != `a\\b` {
		t.Fatalf("expected a\\\\b, got %s", result)
	}
}

// propEscapeJSONBackspace verifies backspace encoding.
func TestProperty_EscapeJSONBackspace(t *testing.T) {
	result := escapeJSON("a\bb")
	if result != "a\\bb" {
		t.Fatalf("expected a\\bb, got %s", result)
	}
}

// propEscapeJSONFormFeed verifies form-feed encoding.
func TestProperty_EscapeJSONFormFeed(t *testing.T) {
	result := escapeJSON("a\fb")
	if result != "a\\fb" {
		t.Fatalf("expected a\\fb, got %s", result)
	}
}

// propEscapeJSONNewline verifies newline encoding.
func TestProperty_EscapeJSONNewline(t *testing.T) {
	result := escapeJSON("a\nb")
	if result != "a\\nb" {
		t.Fatalf("expected a\\nb, got %s", result)
	}
}

// propEscapeJSONCarriageReturn verifies carriage-return encoding.
func TestProperty_EscapeJSONCarriageReturn(t *testing.T) {
	result := escapeJSON("a\rb")
	if result != "a\\rb" {
		t.Fatalf("expected a\\rb, got %s", result)
	}
}

// propEscapeJSONTab verifies tab encoding.
func TestProperty_EscapeJSONTab(t *testing.T) {
	result := escapeJSON("a\tb")
	if result != "a\\tb" {
		t.Fatalf("expected a\\tb, got %s", result)
	}
}

// propEscapeJSONControlChar verifies low control character encoding (< 0x20).
func TestProperty_EscapeJSONControlChar(t *testing.T) {
	result := escapeJSON("a\x01b")
	if result != "a\\u0001b" {
		t.Fatalf("expected a\\u0001b, got %s", result)
	}
}

// propEscapeJSONControlCharNull verifies null byte encoding.
func TestProperty_EscapeJSONControlCharNull(t *testing.T) {
	result := escapeJSON("a\x00b")
	if result != "a\\u0000b" {
		t.Fatalf("expected a\\u0000b, got %s", result)
	}
}

// propEscapeJSONControlCharBell verifies bell character encoding.
func TestProperty_EscapeJSONControlCharBell(t *testing.T) {
	result := escapeJSON("a\x07b")
	if result != "a\\u0007b" {
		t.Fatalf("expected a\\u0007b, got %s", result)
	}
}

// propEscapeJSONNormal verifies normal characters pass through.
func TestProperty_EscapeJSONNormal(t *testing.T) {
	result := escapeJSON("hello world 123 !@#$%^&*()")
	if result != "hello world 123 !@#$%^&*()" {
		t.Fatalf("expected no change, got %s", result)
	}
}

// propEscapeJSONUnicode verifies Unicode characters pass through unchanged.
func TestProperty_EscapeJSONUnicode(t *testing.T) {
	input := "Привет мир 🌍"
	result := escapeJSON(input)
	if result != input {
		t.Fatalf("expected %s, got %s", input, result)
	}
}

// propEscapeJSONAllEscapeChars verifies all JSON escape sequences together.
func TestProperty_EscapeJSONAllEscapeChars(t *testing.T) {
	input := "quote\"back\\bslash\bform\fnewline\ncr\rtab\t\x01\x00"
	result := escapeJSON(input)
	if !strings.Contains(result, `\"`) {
		t.Fatal("missing quote escape")
	}
	if !strings.Contains(result, `\\`) {
		t.Fatal("missing backslash escape")
	}
	if !strings.Contains(result, `\b`) {
		t.Fatal("missing backspace escape")
	}
	if !strings.Contains(result, `\f`) {
		t.Fatal("missing form-feed escape")
	}
	if !strings.Contains(result, `\n`) {
		t.Fatal("missing newline escape")
	}
	if !strings.Contains(result, `\r`) {
		t.Fatal("missing CR escape")
	}
	if !strings.Contains(result, `\t`) {
		t.Fatal("missing tab escape")
	}
	if !strings.Contains(result, `\u0001`) {
		t.Fatal("missing control char escape")
	}
	if !strings.Contains(result, `\u0000`) {
		t.Fatal("missing null escape")
	}
}

// ============================================================================
// Parse/Serialize round-trip with escape sequences
// ============================================================================

// propStringEscapeParseSerialize verifies strings with escapes round-trip.
func TestProperty_StringEscapeParseSerialize(t *testing.T) {
	cases := []struct {
		name string
		val  string
	}{
		{"tab", "hello\tworld"},
		{"newline", "line1\nline2"},
		{"quote", "say \"hello\""},
		{"backslash", "path\\to\\file"},
		{"cr", "line1\rline2"},
		{"backspace", "a\bb"},
		{"formfeed", "a\fb"},
		{"unicode", "Привет 🌍"},
		{"mixed", "a\"b\\c\nd\te\x00"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := `{"value": "` + escapeJSON(tc.val) + `"}`
			doc, err := Parse(input)
			if err != nil {
				t.Fatalf("Parse error: %v\ninput: %s", err, input)
			}
			val := doc.Root().Get("value")
			if val == nil {
				t.Fatal("Get('value') returned nil")
			}
			// Parse should unescape string value, compare via serialized output
			out := Serialize(&Node{Kind: KindDocument, Children: []*Node{val}})
			expected := `"` + escapeJSON(tc.val) + `"`
			if out != expected {
				t.Fatalf("Serialize mismatch:\ngot:  %s\nwant: %s", out, expected)
			}
		})
	}
}

// ============================================================================
// DeepEqual edge cases
// ============================================================================

// propDeepEqualSameType verifies two identical nodes are equal.
func TestProperty_DeepEqualSameType(t *testing.T) {
	n1 := NewString("hello")
	n2 := NewString("hello")
	if !n1.DeepEqual(n2) {
		t.Fatal("identical string nodes should be DeepEqual")
	}
	if !n2.DeepEqual(n1) {
		t.Fatal("DeepEqual should be symmetric")
	}
}

// propDeepEqualDifferentType verifies different node kinds are not equal.
func TestProperty_DeepEqualDifferentType(t *testing.T) {
	n1 := NewString("hello")
	n2 := NewNumber("42")
	if n1.DeepEqual(n2) {
		t.Fatal("different kinds should not be DeepEqual")
	}
}

// propDeepEqualDifferentValue verifies same kind, different value.
func TestProperty_DeepEqualDifferentValue(t *testing.T) {
	n1 := NewString("hello")
	n2 := NewString("world")
	if n1.DeepEqual(n2) {
		t.Fatal("different values should not be DeepEqual")
	}
}

// propDeepEqualDifferentChildren verifies different children count.
func TestProperty_DeepEqualDifferentChildren(t *testing.T) {
	n1 := &Node{Kind: KindObject, Children: []*Node{
		{Kind: KindLBrace, Value: "{"},
		{Kind: KindRBrace, Value: "}"},
	}}
	n2 := &Node{Kind: KindObject, Children: []*Node{
		{Kind: KindLBrace, Value: "{"},
		{Kind: KindString, Value: `"x"`},
		{Kind: KindRBrace, Value: "}"},
	}}
	if n1.DeepEqual(n2) {
		t.Fatal("different children should not be DeepEqual")
	}
}

// propDeepEqualWithPositions verifies DeepEqual ignores position.
func TestProperty_DeepEqualWithPositions(t *testing.T) {
	n1 := &Node{Kind: KindNumber, Value: "42", Start: Position{Offset: 0, Line: 0, Column: 0}}
	n2 := &Node{Kind: KindNumber, Value: "42", Start: Position{Offset: 10, Line: 1, Column: 5}}
	if !n1.DeepEqual(n2) {
		t.Fatal("DeepEqual should ignore position differences")
	}
}

// ============================================================================
// FirstChild edge cases
// ============================================================================

// propFirstChildOnNil verifies FirstChild on nil returns nil.
func TestProperty_FirstChildOnNil(t *testing.T) {
	n := (*Node)(nil)
	if n.FirstChild() != nil {
		t.Fatal("nil.FirstChild() should be nil")
	}
}

// propFirstChildOnValue verifies FirstChild on value node returns nil.
func TestProperty_FirstChildOnValue(t *testing.T) {
	n := NewString("hello")
	if n.FirstChild() != nil {
		t.Fatal("string.FirstChild() should be nil")
	}
}

// propFirstChildOnEmptyDocument verifies FirstChild on empty Document is nil.
func TestProperty_FirstChildOnEmptyDocument(t *testing.T) {
	doc := &Node{Kind: KindDocument}
	if doc.FirstChild() != nil {
		t.Fatal("empty Document.FirstChild() should be nil")
	}
}

// ============================================================================
// Walk on nil and stop (fn returns false)
// ============================================================================

// propWalkOnNil verifies Walk on nil node is a no-op.
func TestProperty_WalkOnNil(t *testing.T) {
	var n *Node
	count := 0
	n.Walk(func(_ *Node) bool {
		count++
		return true
	})
	if count != 0 {
		t.Fatal("Walk on nil should not call fn")
	}
}

// propWalkStopEarly verifies Walk stops when fn returns false.
func TestProperty_WalkStopEarly(t *testing.T) {
	obj := Object("a", Object("b", Object("c", 1)))
	visited := 0
	obj.Walk(func(_ *Node) bool {
		visited++
		return false // stop after first node
	})
	if visited != 1 {
		t.Fatalf("Walk should stop after first node, got %d", visited)
	}
}

// ============================================================================
// toIntValue default branch
// ============================================================================

// propToIntValueDefault verifies the default branch with an unsupported type.
func TestProperty_ToIntValueDefault(t *testing.T) {
	type customInt int
	var ci customInt = 42
	result := toValue(ci)
	// customInt is NOT int (distinct type), so it falls through to default → NewString
	if result == nil || result.Kind != KindString {
		t.Fatal("toValue with custom int should produce a string (not a number)")
	}
	if !strings.Contains(result.Value, "42") {
		t.Fatal("custom int should serialize containing 42")
	}
}

// propToIntValueDefaultBranch directly tests toIntValue with an unsupported type
// to cover the default case in its switch.
func TestProperty_ToIntValueDefaultBranch(t *testing.T) {
	type customInt int
	var ci customInt = 42
	result := toIntValue(ci)
	if result == nil || result.Kind != KindNumber {
		t.Fatal("toIntValue default branch should produce a number")
	}
	if !strings.Contains(result.Value, "42") {
		t.Fatal("toIntValue default should serialize containing 42")
	}
}

// ============================================================================
// toValue default branch (unreachable without new types)
// ============================================================================

type weirdType struct {
	val string
}

// propToValueDefault verifies toValue default branch for unsupported types.
func TestProperty_ToValueDefault(t *testing.T) {
	src := weirdType{val: "test"}
	result := toValue(src)
	if result == nil || result.Kind != KindString {
		t.Fatal("toValue with struct should produce a string")
	}
}

// ============================================================================
// Position.String coverage
// ============================================================================

// propPositionString verifies Position.String format.
func TestProperty_PositionString(t *testing.T) {
	p := Position{Offset: 0, Line: 0, Column: 0}
	s := p.String()
	if s != "1:1(0)" {
		t.Fatalf("expected '1:1(0)', got %q", s)
	}
	p2 := Position{Offset: 100, Line: 4, Column: 15}
	s2 := p2.String()
	if s2 != "5:16(100)" {
		t.Fatalf("expected '5:16(100)', got %q", s2)
	}
}

// ============================================================================
// ValueNode edge cases
// ============================================================================

// propValueNodeOnNonMember verifies ValueNode on non-member returns nil.
func TestProperty_ValueNodeOnNonMember(t *testing.T) {
	n := NewString("hello")
	if n.ValueNode() != nil {
		t.Fatal("ValueNode on non-member should return nil")
	}
}

// propValueNodeOnMemberWithoutColon verifies ValueNode when no colon exists.
func TestProperty_ValueNodeOnMemberWithoutColon(t *testing.T) {
	m := &Node{Kind: KindMember, Children: []*Node{
		NewString("key"),
		// No colon
		NewString("value"),
	}}
	v := m.ValueNode()
	if v != nil {
		t.Fatal("ValueNode without colon should return nil")
	}
}

// ============================================================================
// KeyNode edge cases
// ============================================================================

// propKeyNodeOnMemberWithoutStringKey verifies KeyNode when no string child exists.
func TestProperty_KeyNodeOnMemberWithoutStringKey(t *testing.T) {
	m := &Node{Kind: KindMember, Children: []*Node{
		{Kind: KindColon, Value: ":"},
		{Kind: KindBoolean, Value: "true"},
	}}
	kn := m.KeyNode()
	if kn != nil {
		t.Fatal("KeyNode on member without string should return nil")
	}
}
