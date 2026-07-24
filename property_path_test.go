// Package jsonc tests — new API property-based tests
package jsonc

import (
	"os"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// ============================================================================
// Root API properties
// ============================================================================

// propRootReturnsFirstChild verifies Root() returns the first non-trivia child.
func TestProperty_RootReturnsFirstChild(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := genJSONWithoutComments.Draw(t, "json")
		doc, err := Parse(input)
		if err != nil {
			t.Skip("parse error")
		}
		root := doc.Root()
		first := doc.FirstChild()
		if root != first {
			t.Fatal("Root() should equal FirstChild()")
		}
		if root == nil {
			t.Fatal("Root() should not be nil for valid JSON")
		}
		if !root.IsValue() && root.Kind != KindObject && root.Kind != KindArray {
			t.Fatalf("Root() returned unexpected kind %s", root.Kind)
		}
	})
}

// propRootNilForNonDocument verifies Root() returns nil for non-document nodes.
func TestProperty_RootNilForNonDocument(t *testing.T) {
	n := (*Node)(nil)
	if n.Root() != nil {
		t.Fatal("nil node Root() should be nil")
	}
	obj := Object("x", 1)
	if obj.Root() != nil {
		t.Fatal("non-Document Root() should be nil")
	}
}

// ============================================================================
// GetPath properties
// ============================================================================

// propGetPathNested verifies GetPath navigates into nested objects.
func TestProperty_GetPathNested(t *testing.T) {
	doc := Object(
		"xkeen", Object(
			"speed_balancer", Object(
				"enabled", true,
				"interval", 10,
			),
		),
	)

	if v := doc.GetPath("xkeen"); v == nil || v.Kind != KindObject {
		t.Fatal("GetPath xkeen should return Object")
	}
	if v := doc.GetPath("xkeen.speed_balancer"); v == nil || v.Kind != KindObject {
		t.Fatal("GetPath xkeen.speed_balancer should return Object")
	}
	if v := doc.GetPath("xkeen.speed_balancer.enabled"); v == nil || v.Kind != KindBoolean {
		t.Fatal("GetPath xkeen.speed_balancer.enabled should return Boolean")
	}
	if v := doc.GetPath("xkeen.speed_balancer.interval"); v == nil || v.Kind != KindNumber {
		t.Fatal("GetPath xkeen.speed_balancer.interval should return Number")
	}
}

// propGetPathMissing verifies GetPath returns nil for missing paths.
func TestProperty_GetPathMissing(t *testing.T) {
	doc := Object("a", Object("b", 1))

	if v := doc.GetPath("a.missing"); v != nil {
		t.Fatal("GetPath for missing key should return nil")
	}
	if v := doc.GetPath("a.b.c"); v != nil {
		t.Fatal("GetPath into non-Object should return nil")
	}
	if v := doc.GetPath("x.y.z"); v != nil {
		t.Fatal("GetPath for completely missing path should return nil")
	}
	if v := (*Node)(nil).GetPath("a"); v != nil {
		t.Fatal("GetPath on nil should return nil")
	}
	if v := doc.GetPath(""); v != nil {
		t.Fatal("GetPath with empty path should return nil")
	}
}

// propGetPathSingle is equivalent to Get.
func TestProperty_GetPathSingle(t *testing.T) {
	doc := Object("name", "test", "value", 42)
	if v := doc.GetPath("name"); v == nil || v.Value != `"test"` {
		t.Fatal("GetPath name should return string")
	}
	if v := doc.GetPath("value"); v == nil || v.Kind != KindNumber {
		t.Fatal("GetPath value should return number")
	}
}

// ============================================================================
// SetPath properties
// ============================================================================

// propSetPathCreates verifies SetPath creates a new nested key.
func TestProperty_SetPathCreates(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		key := rapid.StringOf(lettersAZ).Draw(t, "key")
		if key == "" {
			key = "a"
		}
		val := rapid.Int64Range(0, 9999).Draw(t, "val")

		doc := Object("keep", true)
		doc.SetPath("xkeen.speed_balancer."+key, val)

		if v := doc.GetPath("keep"); v == nil {
			t.Fatal("existing key should be preserved")
		}
		if v := doc.GetPath("xkeen.speed_balancer." + key); v == nil {
			t.Fatal("SetPath should create deep path")
		}
	})
}

// propSetPathAutoVivify verifies intermediate objects are auto-created.
func TestProperty_SetPathAutoVivify(t *testing.T) {
	doc := Object()
	doc.SetPath("a.b.c", "deep")

	if v := doc.Get("a"); v == nil || v.Kind != KindObject {
		t.Fatal("SetPath should auto-create intermediate object 'a'")
	}
	if v := doc.GetPath("a.b"); v == nil || v.Kind != KindObject {
		t.Fatal("SetPath should auto-create intermediate object 'a.b'")
	}
	if v := doc.GetPath("a.b.c"); v == nil || v.Value != `"deep"` {
		t.Fatal("SetPath should set final key 'a.b.c'")
	}

	out := Serialize(&Node{Kind: KindDocument, Children: []*Node{doc}})
	if _, err := Parse(out); err != nil {
		t.Fatalf("SetPath produced invalid JSON: %s\n%s", err, out)
	}
}

// propSetPathDeepNesting creates 10 levels of nesting.
func TestProperty_SetPathDeepNesting(t *testing.T) {
	doc := Object()
	path := "l0.l1.l2.l3.l4.l5.l6.l7.l8.l9"
	doc.SetPath(path, 42)

	v := doc.GetPath(path)
	if v == nil || v.Kind != KindNumber {
		t.Fatal("SetPath should handle 10-level deep nesting")
	}

	out := Serialize(&Node{Kind: KindDocument, Children: []*Node{doc}})
	if _, err := Parse(out); err != nil {
		t.Fatalf("SetPath deep nesting produced invalid JSON: %s", err)
	}
}

// ============================================================================
// DeletePath properties
// ============================================================================

// propDeletePathRemoves verifies DeletePath removes a deeply nested key.
func TestProperty_DeletePathRemoves(t *testing.T) {
	doc := Object("x", Object("y", Object("z", 1, "w", 2)))
	doc.DeletePath("x.y.z")

	if v := doc.GetPath("x.y.z"); v != nil {
		t.Fatal("DeletePath should remove x.y.z")
	}
	if v := doc.GetPath("x.y.w"); v == nil {
		t.Fatal("DeletePath should preserve sibling x.y.w")
	}
	if v := doc.GetPath("x.y"); v == nil {
		t.Fatal("DeletePath should preserve parent x.y")
	}
}

// propDeletePathNoop verifies DeletePath on missing path is a no-op.
func TestProperty_DeletePathNoop(t *testing.T) {
	doc := Object("a", 1)
	before := doc.Len()

	doc.DeletePath("a.b.c")
	doc.DeletePath("missing")
	doc.DeletePath("")

	if doc.Len() != before {
		t.Fatal("DeletePath on missing path should not change object")
	}
}

// ============================================================================
// UnmarshalPath properties
// ============================================================================

type testUnmarshalTarget struct {
	Enabled  bool    `json:"enabled"`
	Name     string  `json:"name"`
	Interval int     `json:"interval"`
	Ratio    float64 `json:"ratio,omitempty"`
}

// propUnmarshalPathRoundTrip verifies struct -> CST -> struct round-trip.
func TestProperty_UnmarshalPathRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		src := testUnmarshalTarget{
			Enabled:  rapid.Bool().Draw(t, "enabled"),
			Name:     rapid.StringOf(lettersAZ).Draw(t, "name"),
			Interval: rapid.IntRange(0, 1000).Draw(t, "interval"),
			Ratio:    rapid.Float64Range(0, 1).Draw(t, "ratio"),
		}

		obj, err := structToObject(src)
		if err != nil {
			t.Fatalf("structToObject error: %v", err)
		}

		doc := &Node{Kind: KindDocument, Children: []*Node{obj}}

		var dst testUnmarshalTarget
		if err := doc.Root().UnmarshalPath("", &dst); err != nil {
			t.Fatalf("UnmarshalPath error: %v", err)
		}

		if dst.Enabled != src.Enabled {
			t.Fatalf("Enabled mismatch: %v != %v", dst.Enabled, src.Enabled)
		}
		if dst.Name != src.Name {
			t.Fatalf("Name mismatch: %q != %q", dst.Name, src.Name)
		}
		if dst.Interval != src.Interval {
			t.Fatalf("Interval mismatch: %d != %d", dst.Interval, src.Interval)
		}
	})
}

// propUnmarshalPathNested verifies nested path unmarshaling from a JSONC parse.
func TestProperty_UnmarshalPathNested(t *testing.T) {
	src := `{
	"config": {
		"enabled": true,
		"name": "test",
		"interval": 42
	}
}`
	doc, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	var dst testUnmarshalTarget
	if err := doc.Root().UnmarshalPath("config", &dst); err != nil {
		t.Fatalf("UnmarshalPath error: %v", err)
	}

	if !dst.Enabled {
		t.Fatal("expected enabled=true")
	}
	if dst.Name != "test" {
		t.Fatalf("expected name=test, got %q", dst.Name)
	}
	if dst.Interval != 42 {
		t.Fatalf("expected interval=42, got %d", dst.Interval)
	}
}

// ============================================================================
// MarshalPath properties
// ============================================================================

type testMarshalTarget struct {
	Enabled  bool   `json:"enabled" jsonc:"Enable the feature"`
	Name     string `json:"name"    jsonc:"Display name"`
	Internal string `json:"-"`
	Secret   string `json:"secret,omitempty"`
}

// propMarshalPathComments verifies jsonc tag creates comments for each field.
func TestProperty_MarshalPathComments(t *testing.T) {
	src := testMarshalTarget{
		Enabled: true,
		Name:    "hello",
	}

	obj, err := structToObject(src)
	if err != nil {
		t.Fatalf("structToObject error: %v", err)
	}

	out := Serialize(&Node{Kind: KindDocument, Children: []*Node{obj}})

	if !strings.Contains(out, "// Enable the feature") {
		t.Fatalf("MarshalPath should add comment from jsonc tag:\n%s", out)
	}
	if !strings.Contains(out, "// Display name") {
		t.Fatalf("MarshalPath should add comment from jsonc tag:\n%s", out)
	}
	if !strings.Contains(out, `"enabled": true`) {
		t.Fatalf("MarshalPath should set enabled:\n%s", out)
	}
	if !strings.Contains(out, `"name": "hello"`) {
		t.Fatalf("MarshalPath should set name:\n%s", out)
	}
	if strings.Contains(out, "Internal") {
		t.Fatal("MarshalPath should exclude json:\"-\" fields")
	}
	if strings.Contains(out, "secret") {
		t.Fatal("MarshalPath should omit empty secret field")
	}
}

// propMarshalPathOmitemptyNotEmpty verifies omitempty does NOT omit a non-zero value.
func TestProperty_MarshalPathOmitemptyNotEmpty(t *testing.T) {
	src := testMarshalTarget{
		Enabled: false,
		Name:    "test",
		Secret:  "s3cret",
	}

	obj, err := structToObject(src)
	if err != nil {
		t.Fatalf("structToObject error: %v", err)
	}

	out := Serialize(&Node{Kind: KindDocument, Children: []*Node{obj}})
	if !strings.Contains(out, `"secret": "s3cret"`) {
		t.Fatalf("omitempty should NOT omit non-empty secret:\n%s", out)
	}
}

// propMarshalPathIntoExisting verifies MarshalPath replaces inside an existing Object.
func TestProperty_MarshalPathIntoExisting(t *testing.T) {
	doc := Object("keep_me", 1, "target", Object("old", "data"))
	if err := doc.MarshalPath("target", testUnmarshalTarget{
		Enabled:  true,
		Name:     "new",
		Interval: 99,
	}); err != nil {
		t.Fatalf("MarshalPath error: %v", err)
	}

	if v := doc.Get("keep_me"); v == nil || v.Kind != KindNumber {
		t.Fatal("MarshalPath should preserve sibling keys")
	}
	if v := doc.GetPath("target.enabled"); v == nil || v.Value != "true" {
		t.Fatal("MarshalPath should set enabled")
	}
	if v := doc.GetPath("target.name"); v == nil || v.Value != `"new"` {
		t.Fatal("MarshalPath should set name")
	}
	if v := doc.GetPath("target.old"); v != nil {
		t.Fatal("MarshalPath should remove old keys not in the struct")
	}
}

// ============================================================================
// ParseFile / WriteFile properties
// ============================================================================

// propParseWriteFileRoundTrip verifies ParseFile -> WriteFile -> ParseFile.
func TestProperty_ParseWriteFileRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := genJSONWithoutComments.Draw(t, "json")
		if input == "" {
			t.SkipNow()
			return
		}

		tmpDir, tmpErr := os.MkdirTemp("", "jsonc-test-*")
		if tmpErr != nil {
			t.Fatalf("MkdirTemp error: %v", tmpErr)
		}
		f := tmpDir + "/test.json"
		if err := os.WriteFile(f, []byte(input), 0o644); err != nil {
			t.Fatalf("WriteFile error: %v", err)
		}

		doc, err := ParseFile(f)
		if err != nil {
			t.Skipf("ParseFile error: %v", err)
			return
		}
		if doc == nil {
			t.Fatal("ParseFile returned nil")
			return
		}

		if err := doc.WriteFile(f); err != nil {
			t.Fatalf("WriteFile error: %v", err)
		}

		doc2, err := ParseFile(f)
		if err != nil {
			t.Fatalf("second ParseFile error: %v", err)
		}

		if !doc.DeepEqual(doc2) && doc.RawText() != doc2.RawText() {
			r1 := doc.RawText()
			r2 := doc2.RawText()
			if r1 == r2 {
				return
			}
			if len(r1) > 0 && len(r2) > 0 {
				root1 := doc.FirstChild()
				root2 := doc2.FirstChild()
				if root1 != nil && root2 != nil && root1.RawText() == root2.RawText() {
					return
				}
			}
			t.Fatalf("round-trip produced different output:\nbefore: %s\nafter:  %s", r1, r2)
		}
	})
}

// propMarshalPathPreservesCommentsDuringRoundTrip verifies MarshalPath preserves
// comments outside the replaced subtree.
func TestProperty_MarshalPathPreservesComments(t *testing.T) {
	input := `{
  // Top-level comment
  "keep": 1,
  "xkeen": {
    "speed_balancer": {
      // This comment should be lost (inside replaced subtree)
      "enabled": false
    }
  }
}`
	doc, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	root := doc.Root()
	_ = root.MarshalPath("xkeen.speed_balancer", testUnmarshalTarget{
		Enabled:  true,
		Name:     "test",
		Interval: 10,
	})

	out := Serialize(doc)

	if !strings.Contains(out, "// Top-level comment") {
		t.Fatalf("Top-level comment not preserved:\n%s", out)
	}
	if !strings.Contains(out, `"keep": 1`) {
		t.Fatalf("Sibling key not preserved:\n%s", out)
	}
	if !strings.Contains(out, `"enabled": true`) {
		t.Fatalf("enabled should be true:\n%s", out)
	}
	if !strings.Contains(out, `"interval": 10`) {
		t.Fatalf("interval should be present:\n%s", out)
	}
}

// propMarshalPathWithBlockComment verifies block comment syntax in jsonc tag.
func TestProperty_MarshalPathBlockComment(t *testing.T) {
	type blockCommentStruct struct {
		Value int `json:"value" jsonc:"/* this is a block comment */"`
	}

	obj, err := structToObject(blockCommentStruct{Value: 42})
	if err != nil {
		t.Fatalf("structToObject error: %v", err)
	}

	out := Serialize(&Node{Kind: KindDocument, Children: []*Node{obj}})
	if !strings.Contains(out, "/* this is a block comment */") {
		t.Fatalf("block comment not found in output:\n%s", out)
	}
	if !strings.Contains(out, `"value": 42`) {
		t.Fatalf("value not found in output:\n%s", out)
	}
}
