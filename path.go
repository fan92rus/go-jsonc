package jsonc

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
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
//	arr.GetPath("0")         // first element of array
//	obj.GetPath("items.2")   // third element of items array
//
// Numeric segments index into Arrays via Elements(). Object segments
// use Get() for key lookup. Mixed paths are supported.
// Returns nil when any intermediate segment is missing or wrong type.
func (n *Node) GetPath(path string) *Node {
	if n == nil || path == "" {
		return nil
	}
	segments := strings.Split(path, ".")
	current := n
	for _, seg := range segments {
		if current == nil {
			return nil
		}
		if idx, err := strconv.Atoi(seg); err == nil {
			// Array index access
			if current.Kind != KindArray {
				return nil
			}
			elems := current.Elements()
			if idx < 0 || idx >= len(elems) {
				return nil
			}
			current = elems[idx]
		} else {
			// Object key access
			if current.Kind != KindObject {
				return nil
			}
			current = current.Get(seg)
		}
	}
	return current
}

// SetPath sets a value at a dot-separated path, creating intermediate
// Objects as needed (auto-vivify).
//
//	obj.SetPath("xkeen.speed_balancer.enabled", true)
//	arr.SetPath("0", "replaced")   // replace first array element
//
// Numeric intermediate segments index into Arrays; non-numeric
// segments access Object keys. Auto-vivify creates Objects for
// missing keys.
// Returns the receiver for fluent chaining.
func (n *Node) SetPath(path string, value any) *Node {
	if n == nil || path == "" {
		return nil
	}
	segments := strings.Split(path, ".")
	parent := n
	for _, seg := range segments[:len(segments)-1] {
		parent = setPathNavigate(parent, seg)
		if parent == nil {
			return n
		}
	}
	finalSeg := segments[len(segments)-1]
	return setPathFinal(parent, finalSeg, value)
}

// setPathNavigate resolves one intermediate path segment, auto-vivifying if needed.
func setPathNavigate(parent *Node, seg string) *Node {
	if parent == nil {
		return nil
	}
	if idx, err := strconv.Atoi(seg); err == nil {
		return setPathNavigateArray(parent, idx)
	}
	return setPathNavigateObject(parent, seg)
}

func setPathNavigateArray(parent *Node, idx int) *Node {
	if parent.Kind != KindArray {
		// Auto-vivify array container
		return &Node{Kind: KindArray, Children: []*Node{
			{Kind: KindLBracket, Value: "["},
			{Kind: KindRBracket, Value: "]"},
		}}
	}
	elems := parent.Elements()
	if idx >= 0 && idx < len(elems) {
		return elems[idx]
	}
	if idx == len(elems) {
		empty := &Node{Kind: KindObject, Children: []*Node{
			{Kind: KindLBrace, Value: "{"},
			{Kind: KindRBrace, Value: "}"},
		}}
		parent.appendElement(empty)
		return empty
	}
	return nil
}

func setPathNavigateObject(parent *Node, seg string) *Node {
	if parent.Kind != KindObject {
		return nil
	}
	child := parent.Get(seg)
	if child == nil || (child.Kind != KindObject && child.Kind != KindArray) {
		newObj := &Node{Kind: KindObject,
			Children: []*Node{
				{Kind: KindLBrace, Value: "{"},
				{Kind: KindRBrace, Value: "}"},
			}}
		parent.Set(seg, newObj)
		child = newObj
	}
	return child
}

// setPathFinal applies the value at the last path segment.
func setPathFinal(parent *Node, seg string, value any) *Node {
	if parent == nil {
		return parent
	}
	if idx, err := strconv.Atoi(seg); err == nil {
		return setPathFinalArray(parent, idx, value)
	}
	if parent.Kind == KindObject {
		parent.Set(seg, value)
	}
	return parent
}

func setPathFinalArray(parent *Node, idx int, value any) *Node {
	if parent.Kind != KindArray {
		return parent
	}
	val := toValue(value)
	elems := parent.Elements()
	if idx >= 0 && idx < len(elems) {
		replaceNode(elems[idx], val)
	} else if idx == len(elems) {
		parent.appendElement(val)
	}
	return parent
}

// replaceNode overwrites the type, value, and children of dst with src.
func replaceNode(dst, src *Node) {
	dst.Kind = src.Kind
	dst.Value = src.Value
	dst.Children = src.Children
	dst.Start = src.Start
	dst.End = src.End
}

// appendElement appends a value node as an Array element, inserting a
// comma before it if the array already has elements.
func (n *Node) appendElement(val *Node) {
	if n.Kind == KindArray {
		// Insert comma before closing bracket if we have existing elements
		if len(n.Elements()) > 0 {
			// Insert comma before the last child (the closing bracket)
			n.Children = append(n.Children[:len(n.Children)-1], append(
				[]*Node{{Kind: KindComma, Value: ","}},
				n.Children[len(n.Children)-1:]...,
			)...)
		}
		// Insert the value before the closing bracket
		n.Children = append(n.Children[:len(n.Children)-1], append(
			[]*Node{val},
			n.Children[len(n.Children)-1:]...,
		)...)
	}
}

// DeletePath removes a member at a dot-separated path.
//
//	obj.DeletePath("xkeen.speed_balancer.interval")
//	arr.DeletePath("0")          // remove first array element
//
// Numeric segments target Array elements by index. Intermediate
// access follows the same object/array rules as GetPath.
// Missing or out-of-range paths are a silent no-op.
// Returns the receiver for fluent chaining.
func (n *Node) DeletePath(path string) *Node {
	if n == nil || path == "" {
		return n
	}
	segments := strings.Split(path, ".")
	parent := n
	for _, seg := range segments[:len(segments)-1] {
		if parent == nil {
			return n
		}
		if idx, err := strconv.Atoi(seg); err == nil {
			if parent.Kind != KindArray {
				return n
			}
			elems := parent.Elements()
			if idx < 0 || idx >= len(elems) {
				return n
			}
			parent = elems[idx]
		} else {
			if parent.Kind != KindObject {
				return n
			}
			child := parent.Get(seg)
			if child == nil {
				return n
			}
			parent = child
		}
	}
	if parent == nil {
		return n
	}
	lastSeg := segments[len(segments)-1]
	if idx, err := strconv.Atoi(lastSeg); err == nil {
		// Array element removal
		if parent.Kind != KindArray {
			return n
		}
		parent.deleteElement(idx)
		return n
	}
	if parent.Kind == KindObject {
		parent.Delete(lastSeg)
	}
	return n
}

// deleteElement removes the i-th value element from an Array.
func (n *Node) deleteElement(idx int) {
	if n == nil || n.Kind != KindArray {
		return
	}
	elems := n.Elements()
	if idx < 0 || idx >= len(elems) {
		return
	}
	target := elems[idx]
	// Find and remove the element node and one adjacent comma
	for i, c := range n.Children {
		if c == target {
			removeStart, removeEnd := i, i+1
			if i > 0 && n.Children[i-1].Kind == KindComma {
				removeStart = i - 1
			} else if i+1 < len(n.Children) && n.Children[i+1].Kind == KindComma {
				removeEnd = i + 2
			}
			n.Children = append(n.Children[:removeStart], n.Children[removeEnd:]...)
			return
		}
	}
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

		topComments, bottomComments := splitObjectComments(target.Children)

		obj.Children[0] = lb                   // replace with original {
		obj.Children[len(obj.Children)-1] = rb // replace with original }

		// Inject top comments after the opening brace
		if len(topComments) > 0 {
			head := []*Node{obj.Children[0]}
			head = append(head, topComments...)
			head = append(head, &Node{Kind: KindWhitespace, Value: "\n"})
			obj.Children = append(head, obj.Children[1:]...)
		}
		// Inject bottom comments before the closing brace
		if len(bottomComments) > 0 {
			last := len(obj.Children) - 1
			obj.Children = append(obj.Children[:last], append(bottomComments, obj.Children[last])...)
		}

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

// splitObjectComments separates top and bottom comment groups (with their
// preceding whitespace) from an Object's children array. Comments before
// the first Member are "top"; comments after the last Member are "bottom".
// Returns two slices that can be injected after { and before }.
func splitObjectComments(children []*Node) (top, bottom []*Node) {
	firstMember, lastMember := -1, -1
	for i, c := range children {
		if c.Kind == KindMember {
			if firstMember < 0 {
				firstMember = i
			}
			lastMember = i
		}
	}
	for i := 1; i < len(children)-1; i++ {
		c := children[i]
		if c.Kind == KindComment {
			group := []*Node{c}
			if i > 0 && children[i-1].Kind == KindWhitespace {
				group = append([]*Node{children[i-1]}, group...)
			}
			if firstMember < 0 || i < firstMember {
				top = append(top, group...)
			} else if i > lastMember {
				bottom = append(bottom, group...)
			}
		}
	}
	return top, bottom
}

// serializePlainJSON serializes a CST subtree to plain JSON (no comments,
// no whitespace, minimal output).
func serializePlainJSON(n *Node) []byte {
	if n == nil {
		return nil
	}
	var sb strings.Builder
	serializePlainNode(n, &sb)
	return []byte(sb.String())
}

// serializePlainNode writes the plain JSON representation of a CST node,
// skipping comment and whitespace nodes entirely.
func serializePlainNode(n *Node, sb *strings.Builder) {
	if n == nil {
		return
	}
	if n.Kind == KindComment || n.Kind == KindWhitespace {
		return
	}
	if len(n.Children) > 0 {
		for _, c := range n.Children {
			serializePlainNode(c, sb)
		}
	} else {
		sb.WriteString(n.Value)
	}
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
