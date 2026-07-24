// Package jsonc tests — array index access in path navigation
package jsonc

import (
	"testing"

	"pgregory.net/rapid"
)

// ============================================================================
// GetPath with array index access
// ============================================================================

// propGetPathArrayIndex verifies numeric segment indexes into Array.
func TestProperty_GetPathArrayIndex(t *testing.T) {
	doc := Array("a", "b", "c")
	v := doc.GetPath("0")
	if v == nil || v.Kind != KindString || v.Value != `"a"` {
		t.Fatalf("GetPath 0 should be 'a', got %v", v)
	}
	v = doc.GetPath("1")
	if v == nil || v.Kind != KindString || v.Value != `"b"` {
		t.Fatalf("GetPath 1 should be 'b', got %v", v)
	}
	v = doc.GetPath("2")
	if v == nil || v.Kind != KindString || v.Value != `"c"` {
		t.Fatalf("GetPath 2 should be 'c', got %v", v)
	}
}

// propGetPathArrayIndexOutOfBounds verifies out-of-range index returns nil.
func TestProperty_GetPathArrayIndexOutOfBounds(t *testing.T) {
	doc := Array(1, 2, 3)
	if v := doc.GetPath("3"); v != nil {
		t.Fatal("GetPath 3 on 3-element array should return nil")
	}
	if v := doc.GetPath("-1"); v != nil {
		t.Fatal("GetPath -1 should return nil")
	}
	// Empty array
	empty := NewArray()
	if v := empty.GetPath("0"); v != nil {
		t.Fatal("GetPath 0 on empty array should return nil")
	}
}

// propGetPathArrayOnNonArray verifies numeric segment on non-array returns nil.
func TestProperty_GetPathArrayOnNonArray(t *testing.T) {
	obj := Object("a", 1)
	if v := obj.GetPath("0"); v != nil {
		t.Fatal("GetPath with numeric index on Object should return nil")
	}
	str := NewString("hello")
	if v := str.GetPath("0"); v != nil {
		t.Fatal("GetPath with numeric index on String should return nil")
	}
}

// propGetPathMixedObjectArray verifies mixed object + array path.
func TestProperty_GetPathMixedObjectArray(t *testing.T) {
	doc := Object(
		"items", Array(
			Object("id", 1, "name", "first"),
			Object("id", 2, "name", "second"),
		),
	)

	v := doc.GetPath("items.0.id")
	if v == nil || v.Value != "1" {
		t.Fatalf("expected items.0.id=1, got %v", v)
	}
	v = doc.GetPath("items.1.name")
	if v == nil || v.Value != `"second"` {
		t.Fatalf("expected items.1.name=second, got %v", v)
	}
}

// ============================================================================
// SetPath with array index access
// ============================================================================

// propSetPathArrayIndex verifies SetPath replaces array elements by index.
func TestProperty_SetPathArrayIndex(t *testing.T) {
	doc := Array(1, 2, 3)
	doc.SetPath("0", 99)
	doc.SetPath("2", "replaced")

	v := doc.GetPath("0")
	if v == nil || v.Kind != KindNumber || v.Value != "99" {
		t.Fatalf("expected 99, got %v", v)
	}
	v = doc.GetPath("2")
	if v == nil || v.Kind != KindString || v.Value != `"replaced"` {
		t.Fatalf("expected 'replaced', got %v", v)
	}
	// Verify element 1 unchanged
	v = doc.GetPath("1")
	if v == nil || v.Kind != KindNumber || v.Value != "2" {
		t.Fatalf("expected 2, got %v", v)
	}

	// Round-trip through serialization
	out := Serialize(&Node{Kind: KindDocument, Children: []*Node{doc}})
	if _, err := Parse(out); err != nil {
		t.Fatalf("SetPath array produced invalid JSON: %s\n%s", err, out)
	}
}

// propSetPathArrayIndexOutOfBounds verifies out-of-range index is a no-op.
func TestProperty_SetPathArrayIndexOutOfBounds(t *testing.T) {
	doc := Array(1, 2, 3)
	doc.SetPath("5", 99)
	if doc.Elements()[0].Value != "1" {
		t.Fatal("out-of-range SetPath should be a no-op")
	}
	// Negative index
	doc.SetPath("-1", 99)
	if doc.Elements()[0].Value != "1" {
		t.Fatal("negative index SetPath should be a no-op")
	}
}

// ============================================================================
// DeletePath with array index access
// ============================================================================

// propDeletePathArrayIndex verifies DeletePath removes array elements by index.
func TestProperty_DeletePathArrayIndex(t *testing.T) {
	doc := Array(1, 2, 3, 4)
	doc.DeletePath("1") // remove second element

	v := doc.Elements()
	if len(v) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(v))
	}
	if v[0].Value != "1" || v[1].Value != "3" || v[2].Value != "4" {
		t.Fatalf("unexpected elements after delete: %v, %v, %v", v[0].Value, v[1].Value, v[2].Value)
	}

	// Round-trip through serialization
	out := Serialize(&Node{Kind: KindDocument, Children: []*Node{doc}})
	if _, err := Parse(out); err != nil {
		t.Fatalf("DeletePath array produced invalid JSON: %s\n%s", err, out)
	}
}

// propDeletePathArrayIndexOutOfBounds verifies out-of-range array delete is no-op.
func TestProperty_DeletePathArrayIndexOutOfBounds(t *testing.T) {
	doc := Array(1, 2, 3)
	before := len(doc.Elements())
	doc.DeletePath("10")
	if len(doc.Elements()) != before {
		t.Fatal("out-of-range DeletePath should be a no-op")
	}
	doc.DeletePath("-1")
	if len(doc.Elements()) != before {
		t.Fatal("negative DeletePath should be a no-op")
	}
}

// ============================================================================
// PBT: array index path operations
// ============================================================================

// propArrayPathPBT verifies SetPath/GetPath/DeletePath on arrays with PBT.
func TestProperty_ArrayPathPBT(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		items := rapid.SliceOf(rapid.Int64Range(0, 1000)).Draw(t, "items")
		if len(items) == 0 {
			t.SkipNow()
		}

		// Build an array
		doc := Array()
		for _, it := range items {
			doc = doc.SetPath(intStr(len(doc.Elements())), it)
		}
		if len(doc.Elements()) != len(items) {
			t.Fatalf("expected %d elements, got %d", len(items), len(doc.Elements()))
		}

		// Verify all elements accessible by index
		for i := 0; i < len(items); i++ {
			v := doc.GetPath(intStr(i))
			if v == nil || v.Value != intStr(int(items[i])) {
				t.Fatalf("element %d mismatch: expected %d, got %v", i, items[i], v)
			}
		}

		// Remove first and last, verify
		if len(items) >= 2 {
			doc.DeletePath(intStr(0))
			doc.DeletePath(intStr(len(doc.Elements()) - 1))
			if len(doc.Elements()) != len(items)-2 {
				t.Fatalf("expected %d after delete, got %d", len(items)-2, len(doc.Elements()))
			}
		}
	})
}

// ============================================================================
// Mixed path with arrays and objects (realistic config)
// ============================================================================

// propMixedConfigPath verifies path navigation in a realistic nested config.
func TestProperty_MixedConfigPath(t *testing.T) {
	config := `{
  "outbounds": [
    {"tag": "proxy-1", "protocol": "vmess", "settings": {"port": 443}},
    {"tag": "proxy-2", "protocol": "trojan", "settings": {"port": 8443, "password": "secret"}}
  ],
  "routing": {
    "rules": [
      {"domain": "example.com", "outbound": "proxy-1"},
      {"domain": "test.org", "outbound": "proxy-2"}
    ]
  }
}`

	doc, err := Parse(config)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	root := doc.Root()

	// Object → Array → Object → key access
	assertPath(t, root, "outbounds.0.tag", `"proxy-1"`)
	assertPath(t, root, "outbounds.1.protocol", `"trojan"`)
	assertPath(t, root, "routing.rules.0.domain", `"example.com"`)

	// Modify via array path
	root.SetPath("outbounds.0.settings.port", 80)
	assertPath(t, root, "outbounds.0.settings.port", "80")

	// Delete array element and verify shift
	root.DeletePath("outbounds.0")
	assertPath(t, root, "outbounds.0.tag", `"proxy-2"`)

	// Serialize and re-parse
	out := Serialize(doc)
	doc2, err := Parse(out)
	if err != nil {
		t.Fatalf("Re-parse failed: %v\noutput:\n%s", err, out)
	}

	// After deleting element 0, port is 8443 (trojan)
	assertPath(t, doc2.Root(), "outbounds.0.settings.port", "8443")
}

// assertPath is a test helper: fail if GetPath(path) != expected.
func assertPath(t *testing.T, n *Node, path, expected string) {
	t.Helper()
	v := n.GetPath(path)
	if v == nil {
		t.Fatalf("GetPath(%q) returned nil, expected %q", path, expected)
	}
	if v.Value != expected {
		t.Fatalf("GetPath(%q) = %q, expected %q", path, v.Value, expected)
	}
}

// ============================================================================
// SetPath into first element of a pre-existing array
// ============================================================================

// propSetPathArrayFirstElem verifies SetPath into an existing array element.
func TestProperty_SetPathArrayFirstElem(t *testing.T) {
	doc := Object("items", Array(Object("name", "old")))
	doc.SetPath("items.0.name", "new")

	v := doc.GetPath("items.0.name")
	if v == nil || v.Value != `"new"` {
		t.Fatalf("expected 'new', got %v", v)
	}

	out := Serialize(&Node{Kind: KindDocument, Children: []*Node{doc}})
	if _, err := Parse(out); err != nil {
		t.Fatalf("invalid JSON after SetPath: %s\n%s", err, out)
	}
}

// ============================================================================
// Go-style helper for int to string
// ============================================================================

// intStr is defined in property_coverage_test.go
