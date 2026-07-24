// Package jsonc tests — comment preservation in path/struct operations
package jsonc

import (
	"strings"
	"testing"
)

type sbConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// TestCommentUnmarshal_PreservesCommentsInValue verifies that UnmarshalPath
// works when the target subtree contains JSONC comments (line and block).
func TestComment_UnmarshalPathWithComments(t *testing.T) {
	src := `{
  "host": "localhost", // this is the host
  "port": 8080 /* web port */
}`
	doc, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	var cfg sbConfig
	if err := doc.Root().UnmarshalPath("", &cfg); err != nil {
		t.Fatalf("UnmarshalPath error: %v", err)
	}
	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "localhost")
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8080)
	}
}

// TestComment_UnmarshalLeafValueWithComment verifies that a leaf value
// with trailing comments unmarshals correctly.
func TestComment_UnmarshalLeafValueWithComments(t *testing.T) {
	src := `{
  "items": [
    1, /* first */
    2, // second
    3  /* third */
  ]
}`
	doc, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Deserialize entire array
	var nums []int
	if err := doc.Root().UnmarshalPath("items", &nums); err != nil {
		t.Fatalf("UnmarshalPath items: %v", err)
	}
	if len(nums) != 3 || nums[0] != 1 || nums[1] != 2 || nums[2] != 3 {
		t.Fatalf("got %v, want [1 2 3]", nums)
	}

	// Deserialize a single element
	var item int
	if err := doc.Root().UnmarshalPath("items.0", &item); err != nil {
		t.Fatalf("UnmarshalPath items.0: %v", err)
	}
	if item != 1 {
		t.Errorf("item = %d, want 1", item)
	}
}

// TestComment_UnmarshalArrayWithComments verifies array with comments
// between elements unmarshals correctly.
func TestComment_UnmarshalArrayWithComments(t *testing.T) {
	src := `{
  "ports": [
    // first
    100,
    // second
    200,
    /* third */
    300
    // last
  ]
}`
	doc, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	var ports []int
	if err := doc.Root().UnmarshalPath("ports", &ports); err != nil {
		t.Fatalf("UnmarshalPath ports: %v", err)
	}
	if len(ports) != 3 || ports[0] != 100 || ports[2] != 300 {
		t.Fatalf("got %v, want [100 200 300]", ports)
	}
}

// TestComment_MarshalPreservesTopLevelComments verifies that MarshalPath
// preserves top-level comments when replacing an Object's contents.
func TestComment_MarshalPreservesTopLevelComments(t *testing.T) {
	src := `{
  // server config
  "host": "old-host",
  "port": 1234
}`
	doc, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	newCfg := sbConfig{Host: "new-host", Port: 8080}
	if err := doc.Root().MarshalPath("", newCfg); err != nil {
		t.Fatalf("MarshalPath error: %v", err)
	}

	out := Serialize(doc)
	if !strings.Contains(out, "server config") {
		t.Errorf("top-level comment lost after MarshalPath:\n%s", out)
	}
	if !strings.Contains(out, `"host": "new-host"`) {
		t.Errorf("new host not found in output:\n%s", out)
	}
	if !strings.Contains(out, `"port": 8080`) {
		t.Errorf("new port not found in output:\n%s", out)
	}

	// Re-parse to ensure valid JSONC
	doc2, err := Parse(out)
	if err != nil {
		t.Fatalf("Re-parse error after MarshalPath: %v\n%s", err, out)
	}
	var got sbConfig
	if err := doc2.Root().UnmarshalPath("", &got); err != nil {
		t.Fatalf("UnmarshalPath after MarshalPath: %v", err)
	}
	if got.Host != "new-host" || got.Port != 8080 {
		t.Fatalf("round-trip: got %+v, want {new-host 8080}", got)
	}
}

// TestComment_SetPathOnValueWithComments verifies that SetPath preserves
// comments inside the replaced structure.
func TestComment_SetPathOnValueWithComments(t *testing.T) {
	src := `{
  "outbounds": [
    {"tag": "proxy-1" /* first */},
    {"tag": "proxy-2" /* second */}
  ]
}`
	doc, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// SetPath on the tag value — replaces just the string node
	doc.Root().SetPath("outbounds.0.tag", "changed")
	out := Serialize(doc)

	if !strings.Contains(out, "/* first */") {
		t.Errorf("comment after tag value lost:\n%s", out)
	}
	if !strings.Contains(out, `"tag": "changed"`) {
		t.Errorf("new tag value not found:\n%s", out)
	}
}

// TestComment_UnmarshalPathSingleFieldFromMemberWithComment verifies
// that UnmarshalPath on a single key works even with value comments.
func TestComment_UnmarshalSingleFieldWithComment(t *testing.T) {
	const wantHost = "test"
	src := `{"host": "` + wantHost + `" /* hostname */}`
	doc, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	var host string
	if err := doc.Root().UnmarshalPath("host", &host); err != nil {
		t.Fatalf("UnmarshalPath host: %v", err)
	}
	if host != wantHost {
		t.Errorf("host = %q, want %q", host, wantHost)
	}
}
