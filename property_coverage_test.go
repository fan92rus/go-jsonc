// Package jsonc tests — deep coverage for internal helpers
package jsonc

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// ============================================================================
// reflectToNode coverage: all branch types
// ============================================================================

// propSetPathString verifies SetPath with string value.
func TestProperty_SetPathString(t *testing.T) {
	doc := Object()
	doc.SetPath("a", "hello")
	v := doc.Get("a")
	if v == nil || v.Value != `"hello"` {
		t.Fatalf("expected string, got %v", v)
	}
}

// propSetPathInt verifies SetPath with int value.
func TestProperty_SetPathInt(t *testing.T) {
	doc := Object()
	doc.SetPath("a", 42)
	v := doc.Get("a")
	if v == nil || v.Value != "42" || v.Kind != KindNumber {
		t.Fatalf("expected number 42, got %v", v)
	}
}

// propSetPathFloat verifies SetPath with float value.
func TestProperty_SetPathFloat(t *testing.T) {
	doc := Object()
	doc.SetPath("a", 3.14)
	v := doc.Get("a")
	if v == nil || v.Kind != KindNumber {
		t.Fatalf("expected number, got %v", v)
	}
}

// propSetPathBoolean verifies SetPath with boolean value.
func TestProperty_SetPathBoolean(t *testing.T) {
	doc := Object()
	doc.SetPath("a", true)
	doc.SetPath("b", false)
	av := doc.Get("a")
	bv := doc.Get("b")
	if av == nil || av.Value != boolStr(true) || av.Kind != KindBoolean {
		t.Fatal("expected boolean true")
	}
	if bv == nil || bv.Value != boolStr(false) || bv.Kind != KindBoolean {
		t.Fatal("expected boolean false")
	}
}

// propSetPathNull verifies SetPath with nil value.
func TestProperty_SetPathNull(t *testing.T) {
	doc := Object()
	doc.SetPath("a", nil)
	v := doc.Get("a")
	if v == nil || v.Kind != KindNull {
		t.Fatalf("expected null, got %v", v)
	}
}

// propSetPathArray verifies SetPath with array value.
func TestProperty_SetPathArray(t *testing.T) {
	doc := Object()
	doc.SetPath("a", []any{1, "two", true})
	v := doc.Get("a")
	if v == nil || v.Kind != KindArray {
		t.Fatalf("expected array, got %v", v)
	}
	out := Serialize(&Node{Kind: KindDocument, Children: []*Node{doc}})
	if _, err := Parse(out); err != nil {
		t.Fatalf("SetPath array produced invalid JSON: %s\n%s", err, out)
	}
}

// propSetPathObject verifies SetPath with map (object) value.
func TestProperty_SetPathObject(t *testing.T) {
	doc := Object()
	doc.SetPath("a", map[string]any{"x": 1, "y": "z"})
	v := doc.Get("a")
	if v == nil || v.Kind != KindObject {
		t.Fatalf("expected object, got %v", v)
	}
	out := Serialize(&Node{Kind: KindDocument, Children: []*Node{doc}})
	if _, err := Parse(out); err != nil {
		t.Fatalf("SetPath object produced invalid JSON: %s\n%s", err, out)
	}
}

// propSetPathNegativeInt verifies negative int values.
func TestProperty_SetPathNegativeInt(t *testing.T) {
	doc := Object()
	doc.SetPath("a", -42)
	v := doc.Get("a")
	if v == nil || v.Value != "-42" || v.Kind != KindNumber {
		t.Fatalf("expected number -42, got %v", v)
	}
}

// ============================================================================
// AppendChild coverage
// ============================================================================

// propAppendChildObject adds a member to an object via AppendChild.
func TestProperty_AppendChildObject(t *testing.T) {
	obj := Object("a", 1)
	val := &Node{Kind: KindBoolean, Value: "true"}
	member := NewMember("b", val)
	obj.AppendChild(member)

	if obj.Len() != 2 {
		t.Fatalf("expected 2 members, got %d", obj.Len())
	}
	if v := obj.Get("b"); v == nil || v.Kind != KindBoolean {
		t.Fatal("AppendChild should add 'b' member")
	}
}

// propAppendChildArray adds an element to an array via AppendChild.
func TestProperty_AppendChildArray(t *testing.T) {
	arr := NewArray(NewNumber("1"), NewNumber("2"))
	arr.AppendChild(&Node{Kind: KindNumber, Value: "3"})
	elems := arr.Elements()
	if len(elems) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(elems))
	}
	if elems[2].Value != "3" {
		t.Fatalf("expected element '3', got %s", elems[2].Value)
	}
}

// propAppendChildToDocument verifies AppendChild on a Document.
func TestProperty_AppendChildToDocument(t *testing.T) {
	doc := &Node{Kind: KindDocument}
	obj := Object("a", 1)
	doc.AppendChild(obj)
	root := doc.Root()
	if root == nil || root.Kind != KindObject {
		t.Fatal("AppendChild to Document should make root available")
	}
}

// propAppendChildOnValueNode verifies AppendChild on a value node with nil children.
func TestProperty_AppendChildOnValueNode(t *testing.T) {
	n := NewString("hello")
	initial := len(n.Children)
	n.AppendChild(&Node{Kind: KindWhitespace, Value: " "})
	if len(n.Children) != initial+1 {
		t.Fatalf("AppendChild on value should add child")
	}
}

// ============================================================================
// SetCommentBody coverage
// ============================================================================

// propSetCommentBodyOnLineComment verifies SetCommentBody modifies a line comment.
func TestProperty_SetCommentBodyOnLineComment(t *testing.T) {
	c := NewCommentLine("old")
	c.SetCommentBody("new body")
	if c.Value != "// new body" {
		t.Fatalf("expected '// new body', got %q", c.Value)
	}
	if c.CommentBody != "new body" {
		t.Fatalf("expected CommentBody 'new body', got %q", c.CommentBody)
	}
}

// propSetCommentBodyOnBlockComment verifies SetCommentBody modifies a block comment.
func TestProperty_SetCommentBodyOnBlockComment(t *testing.T) {
	c := NewCommentBlock("old")
	c.SetCommentBody("new body")
	if c.Value != "/* new body */" {
		t.Fatalf("expected '/* new body */', got %q", c.Value)
	}
	if c.CommentBody != "new body" {
		t.Fatalf("expected CommentBody 'new body', got %q", c.CommentBody)
	}
}

// propSetCommentBodyOnNonComment verifies SetCommentBody on non-comment is no-op.
func TestProperty_SetCommentBodyOnNonComment(t *testing.T) {
	n := NewString("hello")
	n.SetCommentBody("world")
	if n.Value != `"hello"` {
		t.Fatal("SetCommentBody on non-comment should be no-op")
	}
}

// ============================================================================
// Body() deeper coverage
// ============================================================================

// propBodyOnEmpty verifies Body returns empty for nil/non-comment nodes.
func TestProperty_BodyOnEmpty(t *testing.T) {
	n := &Node{Kind: KindNumber, Value: "42"}
	if body := n.Body(); body != "" {
		t.Fatal("value node Body() should be empty")
	}
}

// propBodyOnComment verifies Body returns the comment text.
func TestProperty_BodyOnComment(t *testing.T) {
	lc := NewCommentLine("some note")
	body := lc.Body()
	if body != "some note" {
		t.Fatalf("expected 'some note', got %q", body)
	}

	bc := NewCommentBlock("block note")
	body = bc.Body()
	if body != "block note" {
		t.Fatalf("expected 'block note', got %q", body)
	}
}

// ============================================================================
// isZero deeper coverage via struct omitempty
// ============================================================================

type testOmitemptyFields struct {
	A string  `json:"a,omitempty"`
	B bool    `json:"b,omitempty"`
	C int     `json:"c,omitempty"`
	D uint    `json:"d,omitempty"`
	E float64 `json:"e,omitempty"`
}

// propOmitemptyAllZero verifies MarshalPath omits all zero-value fields.
func TestProperty_OmitemptyAllZero(t *testing.T) {
	src := testOmitemptyFields{}
	obj, err := structToObject(src)
	if err != nil {
		t.Fatalf("structToObject error: %v", err)
	}
	out := Serialize(&Node{Kind: KindDocument, Children: []*Node{obj}})
	if out != "{}" {
		t.Fatalf("expected empty object {}, got %s", out)
	}
}

// propOmitemptyNonZero verifies MarshalPath includes non-zero omitempty fields.
func TestProperty_OmitemptyNonZero(t *testing.T) {
	src := testOmitemptyFields{
		A: "hello",
		B: true,
		C: 42,
		D: 7,
		E: 3.5,
	}
	obj, err := structToObject(src)
	if err != nil {
		t.Fatalf("structToObject error: %v", err)
	}
	out := Serialize(&Node{Kind: KindDocument, Children: []*Node{obj}})
	for _, field := range []string{`"a": "hello"`, `"b": true`, `"c": 42`, `"d": 7`} {
		if !strings.Contains(out, field) {
			t.Fatalf("expected %s in output:\n%s", field, out)
		}
	}
}

// ============================================================================
// PBT: SetPath/GetPath/DeletePath round-trip with various types
// ============================================================================

// propSetGetDeletePathPBT verifies SetPath → GetPath → DeletePath on random data.
func TestProperty_SetGetDeletePathPBT(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		key1 := rapid.StringOf(lettersAZ).Draw(t, "key1")
		key2 := rapid.StringOf(lettersAZ).Draw(t, "key2")
		val := rapid.Int64Range(-1000, 1000).Draw(t, "val")

		if key1 == "" || key2 == "" {
			t.SkipNow()
		}

		doc := Object()
		path := key1 + "." + key2
		doc.SetPath(path, val)
		got := doc.GetPath(path)
		if got == nil || got.Kind != KindNumber {
			t.Fatalf("expected number at %s", path)
		}
		doc.DeletePath(path)
		if doc.GetPath(path) != nil {
			t.Fatalf("DeletePath should remove %s", path)
		}
		if doc.Get(key1) == nil || doc.Get(key1).Kind != KindObject {
			t.Fatalf("parent %s should still be an object", key1)
		}
	})
}

// ============================================================================
// End-to-end workflow: parse → modify via path → write → re-parse
// ============================================================================

// propConfigWorkflowPBT simulates the xkeen-ui config workflow with PBT.
func TestProperty_ConfigWorkflowPBT(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		enabled := rapid.Bool().Draw(t, "enabled")
		interval := rapid.IntRange(1, 3600).Draw(t, "interval")
		port := rapid.IntRange(1024, 65535).Draw(t, "port")

		template := `{
  // XKeen global config
  "xkeen": {
    "speed_balancer": {
      // Speed balancer settings
      "enabled": false,
      "interval": 10
    }
  }
}`
		doc, err := Parse(template)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		root := doc.Root()

		// Mutate via path API
		root.SetPath("xkeen.speed_balancer.enabled", enabled)
		root.SetPath("xkeen.speed_balancer.interval", interval)
		root.SetPath("xkeen.speed_balancer.port", port)

		// Verify via GetPath
		e := root.GetPath("xkeen.speed_balancer.enabled")
		if e == nil || (e.Kind == KindBoolean && e.Value != boolStr(enabled)) {
			t.Fatalf("enabled mismatch: got %s", e.Value)
		}
		i := root.GetPath("xkeen.speed_balancer.interval")
		if i == nil || i.Value != intStr(interval) {
			t.Fatalf("interval mismatch: got %s, expected %d", i.Value, interval)
		}

		// Verify comments preserved
		out := Serialize(doc)
		if !strings.Contains(out, "// XKeen global config") {
			t.Fatalf("top-level comment lost:\n%s", out)
		}
		if !strings.Contains(out, "// Speed balancer settings") {
			t.Fatalf("nested comment lost:\n%s", out)
		}

		// Round-trip through file
		dir, _ := os.MkdirTemp("", "jsonc-workflow-*")
		defer func() { _ = os.RemoveAll(dir) }()
		f := dir + "/config.json"
		if err := doc.WriteFile(f); err != nil {
			t.Fatalf("WriteFile error: %v", err)
		}
		doc2, err := ParseFile(f)
		if err != nil {
			t.Fatalf("ParseFile error: %v", err)
		}
		root2 := doc2.Root()
		e2 := root2.GetPath("xkeen.speed_balancer.enabled")
		if e2 == nil || e2.Value != boolStr(enabled) {
			t.Fatalf("round-trip enabled mismatch")
		}
	})
}

// boolStr converts a bool to the JSON string representation.
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// intStr converts an int to JSON number string.
func intStr(i int) string {
	return fmt.Sprintf("%d", i)
}

// ============================================================================
// serializePlainJSON coverage
// ============================================================================

// propUnmarshalPathInvalid verifies UnmarshalPath returns error for bad paths.
func TestProperty_UnmarshalPathInvalid(t *testing.T) {
	doc := Object("a", 1)
	err := doc.UnmarshalPath("missing", nil)
	if err == nil {
		t.Fatal("expected error for missing path")
	}

	var target testUnmarshalTarget
	err = doc.UnmarshalPath("", &target)
	if err != nil {
		t.Fatal("empty path should unmarshal from self, got:", err)
	}
}

// propMarshalPathInvalid verifies MarshalPath returns error for bad paths.
func TestProperty_MarshalPathInvalid(t *testing.T) {
	doc := Object("a", 1)
	err := doc.MarshalPath("missing", struct{ X int }{X: 42})
	if err == nil {
		t.Fatal("expected error for missing path")
	}

	err = doc.MarshalPath("a", struct{ X int }{X: 42})
	if err == nil {
		t.Fatal("expected error for scalar path (not an Object)")
	}
}

// propParseFileErrors verifies ParseFile handles file errors.
func TestProperty_ParseFileErrors(t *testing.T) {
	_, err := ParseFile("/nonexistent/path/file.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

// propSetPathToUint verifies uint values via SetPath.
func TestProperty_SetPathToUint(t *testing.T) {
	var ui uint = 255
	n := &Node{Kind: KindDocument}
	n.AppendChild(Object())
	doc := n.FirstChild()
	doc.SetPath("max_uint", ui)
	v := doc.Get("max_uint")
	if v == nil || v.Kind != KindNumber || v.Value != "255" {
		t.Fatalf("expected uint 255, got %v", v)
	}
}

// propSetPathToFloat verifies float64 values via SetPath.
func TestProperty_SetPathToFloat(t *testing.T) {
	f := 3.14159
	doc := Object()
	doc.SetPath("pi", f)
	v := doc.Get("pi")
	if v == nil || v.Kind != KindNumber {
		t.Fatalf("expected number, got %v", v)
	}
	out := Serialize(&Node{Kind: KindDocument, Children: []*Node{doc}})
	if !strings.Contains(out, "3.14159") {
		t.Fatalf("expected 3.14159 in output:\n%s", out)
	}
}
