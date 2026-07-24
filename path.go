package jsonc

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
)

// ---------------------------------------------------------------------------
// Document-level convenience
// ---------------------------------------------------------------------------

// Root returns the first non-trivia child of a Document, typically an
// Object or Array — the root value of the parsed JSONC document.
// Returns nil for nil, non-Document, or empty documents.
func (n *Node) Root() *Node {
	if n == nil || n.Kind != KindDocument {
		return nil
	}
	return n.FirstChild()
}

// ---------------------------------------------------------------------------
// Dot-path navigation
// ---------------------------------------------------------------------------

// GetPath navigates dot-separated keys from this node.
//
//	obj.GetPath("xkeen.speed_balancer.enabled")
//
// Each segment navigates into an Object via Get(). If any intermediate
// Object is nil or missing the segment key, returns nil.
// For single-segment paths, behaves identically to Get().
func (n *Node) GetPath(path string) *Node {
	if n == nil || path == "" {
		return nil
	}
	segments := strings.Split(path, ".")
	current := n
	for _, seg := range segments {
		if current == nil || current.Kind != KindObject {
			return nil
		}
		current = current.Get(seg)
	}
	return current
}

// SetPath sets a value at a dot-separated path, creating intermediate
// Objects as needed (auto-vivify).
//
//	obj.SetPath("xkeen.speed_balancer.enabled", true)
//
// The final segment uses Set() to update or create the member.
// Returns the receiver for fluent chaining, or nil on error.
func (n *Node) SetPath(path string, value any) *Node {
	if n == nil || path == "" {
		return nil
	}
	segments := strings.Split(path, ".")
	parent := n
	for i, seg := range segments[:len(segments)-1] {
		if parent == nil || parent.Kind != KindObject {
			return nil
		}
		child := parent.Get(seg)
		if child == nil || child.Kind != KindObject {
			// Auto-vivify
			newObj := &Node{Kind: KindObject,
				Children: []*Node{
					{Kind: KindLBrace, Value: "{"},
					{Kind: KindRBrace, Value: "}"},
				}}
			parent.Set(seg, newObj)
			child = newObj
		}
		parent = child
		_ = i
	}
	// Final segment — use Set on parent
	if parent != nil && parent.Kind == KindObject {
		parent.Set(segments[len(segments)-1], value)
		return n
	}
	return n
}

// DeletePath removes a member at a dot-separated path.
//
//	obj.DeletePath("xkeen.speed_balancer.interval")
//
// All intermediate keys must exist and be Objects. If the path is
// invalid (intermediate key missing, wrong type), this is a no-op.
// Returns the receiver for fluent chaining.
func (n *Node) DeletePath(path string) *Node {
	if n == nil || path == "" {
		return n
	}
	segments := strings.Split(path, ".")
	parent := n
	for _, seg := range segments[:len(segments)-1] {
		if parent == nil || parent.Kind != KindObject {
			return n
		}
		child := parent.Get(seg)
		if child == nil || child.Kind != KindObject {
			return n
		}
		parent = child
	}
	if parent != nil && parent.Kind == KindObject {
		parent.Delete(segments[len(segments)-1])
	}
	return n
}

// ---------------------------------------------------------------------------
// CST ↔ Go struct: UnmarshalPath & MarshalPath
// ---------------------------------------------------------------------------

// UnmarshalPath navigates to the subtree at path, serializes it to
// plain JSON (stripping comments), and unmarshals into v using
// encoding/json.
//
//	v := SpeedBalancerSettings{}
//	err := doc.Root().UnmarshalPath("xkeen.speed_balancer", &v)
//
// path "" means unmarshal from this node itself.
func (n *Node) UnmarshalPath(path string, v any) error {
	target := n
	if path != "" {
		target = n.GetPath(path)
	}
	if target == nil {
		return fmt.Errorf("jsonc: path %q not found", path)
	}
	// Serialize subtree to plain JSON (strip comments/whitespace)
	jsonBytes := serializePlainJSON(target)
	return json.Unmarshal(jsonBytes, v)
}

// MarshalPath serializes a Go struct v to a CST subtree and replaces
// the member at path.
//
//	doc.Root().MarshalPath("xkeen.speed_balancer", sbSettings)
//
// Struct fields are mapped by json tag; the optional jsonc tag provides
// a line comment that precedes the member. Example:
//
//	type Config struct {
//	    Enabled bool   `json:"enabled" jsonc:"Enable the feature"`
//	    Name    string `json:"name"`
//	}
//
// Path "" means replace this node itself (it must be an Object).
func (n *Node) MarshalPath(path string, v any) error {
	target := n
	if path != "" {
		target = n.GetPath(path)
	}
	if target == nil {
		return fmt.Errorf("jsonc: path %q not found", path)
	}
	if target.Kind != KindObject {
		return fmt.Errorf("jsonc: path %q is not an Object", path)
	}

	obj, err := structToObject(v)
	if err != nil {
		return err
	}

	// Replace target's children with new Object's children, preserving
	// the opening/closing braces from the original.
	if len(target.Children) >= 2 {
		lb := target.Children[0]
		rb := target.Children[len(target.Children)-1]
		obj.Children[0] = lb                   // replace with original {
		obj.Children[len(obj.Children)-1] = rb // replace with original }
		target.Children = obj.Children
		target.Start = obj.Start
		target.End = obj.End
	} else {
		// Empty or malformed — completely replace
		target.Children = obj.Children
		target.Start = obj.Start
		target.End = obj.End
	}

	return nil
}

// serializePlainJSON serializes a CST subtree to plain JSON (no comments,
// minimal whitespace).
func serializePlainJSON(n *Node) []byte {
	if n == nil {
		return nil
	}
	return []byte(Serialize(&Node{Kind: KindDocument, Children: []*Node{n}}))
}

// structToObject converts a Go struct value to an Object CST node.
// Uses json tags for field names and the optional jsonc tag for comments.
func structToObject(v any) (*Node, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("jsonc: expected struct, got %T", v)
	}
	rt := rv.Type()

	members := make([]*Node, 0)
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if !field.IsExported() {
			continue
		}

		// Read json tag for key name and options
		jsonTag := field.Tag.Get("json")
		jsonName, omitempty := parseJSONTag(jsonTag)
		if jsonName == "" || jsonName == "-" {
			continue
		}

		// Read jsonc tag for optional comment
		commentTag := field.Tag.Get("jsonc")

		fv := rv.Field(i)

		// Handle omitempty
		if omitempty && isZero(fv) {
			continue
		}

		// Convert field value to Node
		valNode := reflectToNode(fv)

		// Build member with optional comment
		children := make([]*Node, 0)
		if commentTag != "" {
			// Try commentStyle annotation: "// text" or "/* text */" or plain text
			tag := strings.TrimSpace(commentTag)
			if strings.HasPrefix(tag, "//") {
				children = append(children, NewCommentLine(strings.TrimPrefix(tag, "//")))
			} else if strings.HasPrefix(tag, "/*") {
				tag = strings.TrimSuffix(strings.TrimPrefix(tag, "/*"), "*/")
				children = append(children, NewCommentBlock(strings.TrimSpace(tag)))
			} else {
				children = append(children, NewCommentLine(tag))
			}
		}
		children = append(children, NewString(jsonName))
		children = append(children, &Node{Kind: KindColon, Value: ":"})
		children = append(children, &Node{Kind: KindWhitespace, Value: " "})
		children = append(children, valNode)

		members = append(members, &Node{Kind: KindMember, Children: children})
	}

	return ObjectFromMembers(members), nil
}

// ObjectFromMembers creates an Object CST node from a slice of Member nodes.
func ObjectFromMembers(members []*Node) *Node {
	nodes := make([]*Node, 0, len(members)*2+2)
	nodes = append(nodes, &Node{Kind: KindLBrace, Value: "{"})
	for i, m := range members {
		if i > 0 {
			nodes = append(nodes, &Node{Kind: KindComma, Value: ","})
		}
		nodes = append(nodes, m)
	}
	nodes = append(nodes, &Node{Kind: KindRBrace, Value: "}"})
	return &Node{Kind: KindObject, Children: nodes}
}

// parseJSONTag extracts name and omitempty from a json tag.
func parseJSONTag(tag string) (name string, omitempty bool) {
	parts := strings.Split(tag, ",")
	if len(parts) == 0 || parts[0] == "" {
		return "", false
	}
	name = parts[0]
	for _, p := range parts[1:] {
		if p == "omitempty" {
			omitempty = true
		}
	}
	return name, omitempty
}

// isZero checks if a reflect.Value is the zero value for its kind.
func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Ptr, reflect.Slice, reflect.Map:
		return v.IsNil()
	}
	return false
}

// reflectToNode converts a reflect.Value to a CST value Node.
func reflectToNode(v reflect.Value) *Node {
	if !v.IsValid() {
		return NewNull()
	}

	switch v.Kind() {
	case reflect.String:
		return NewString(v.String())
	case reflect.Bool:
		return NewBoolean(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return NewNumber(fmt.Sprintf("%d", v.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return NewNumber(fmt.Sprintf("%d", v.Uint()))
	case reflect.Float32, reflect.Float64:
		return NewNumber(fmt.Sprintf("%v", v.Interface()))
	case reflect.Ptr:
		if v.IsNil() {
			return NewNull()
		}
		return reflectToNode(v.Elem())
	case reflect.Slice:
		if v.IsNil() {
			return NewNull()
		}
		elems := make([]*Node, v.Len())
		for i := 0; i < v.Len(); i++ {
			elems[i] = reflectToNode(v.Index(i))
		}
		return NewArray(elems...)
	case reflect.Map:
		if v.IsNil() {
			return NewNull()
		}
		// Sort keys for deterministic output
		keys := v.MapKeys()
		members := make([]*Node, len(keys))
		for i, k := range keys {
			name := fmt.Sprintf("%v", k.Interface())
			members[i] = &Node{Kind: KindMember, Children: []*Node{
				NewString(name),
				{Kind: KindColon, Value: ":"},
				{Kind: KindWhitespace, Value: " "},
				reflectToNode(v.MapIndex(k)),
			}}
		}
		return ObjectFromMembers(members)
	case reflect.Struct:
		node, err := structToObject(v.Interface())
		if err != nil {
			return NewNull()
		}
		return node
	default:
		return NewString(fmt.Sprintf("%v", v.Interface()))
	}
}

// ---------------------------------------------------------------------------
// ParseFile / WriteFile
// ---------------------------------------------------------------------------

// ParseFile reads a file and parses its content as JSONC.
func ParseFile(path string) (*Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("jsonc: reading %q: %w", path, err)
	}
	doc, err := Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("jsonc: parsing %q: %w", path, err)
	}
	return doc, nil
}

// WriteFile serializes a Document and writes it to a file.
func (n *Node) WriteFile(path string) error {
	if n == nil {
		return fmt.Errorf("jsonc: cannot write nil document")
	}
	out := Serialize(n)
	return os.WriteFile(path, []byte(out), 0o600)
}
