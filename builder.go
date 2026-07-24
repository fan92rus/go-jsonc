package jsonc

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// Compact constructors — auto-wrap Go types into CST nodes.
// ---------------------------------------------------------------------------

// Object builds a JSON object from key-value pairs.
// Each argument pair is (string key, any value).
//
// Supported value types:
//   - string          → quoted JSON string
//   - int, int64      → number
//   - float64         → number
//   - bool            → true/false
//   - nil             → null
//   - *Node           → used as-is (for pre-built / nested structures)
//   - []any           → Array (recursive)
//   - map[string]any  → Object (recursive, keys sorted)
func Object(keyvals ...any) *Node {
	if len(keyvals)%2 != 0 {
		panic("jsonc.Object: odd number of arguments (must be key-value pairs)")
	}
	members := make([]*Node, len(keyvals)/2)
	for i := 0; i < len(keyvals); i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			panic(fmt.Sprintf("jsonc.Object: key must be string, got %T", keyvals[i]))
		}
		members[i/2] = NewMember(key, toValue(keyvals[i+1]))
	}
	return NewObject(members...)
}

// Array builds a JSON array from Go values.
func Array(values ...any) *Node {
	elems := make([]*Node, len(values))
	for i, v := range values {
		elems[i] = toValue(v)
	}
	return NewArray(elems...)
}

// ---------------------------------------------------------------------------
// Type conversion
// ---------------------------------------------------------------------------

func toValue(v any) *Node {
	switch x := v.(type) {
	case *Node:
		return x
	case string:
		return NewString(x)
	case bool:
		return NewBoolean(x)
	case nil:
		return NewNull()
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return toIntValue(x)
	case float32:
		return NewNumber(strconv.FormatFloat(float64(x), 'f', -1, 32))
	case float64:
		return NewNumber(strconv.FormatFloat(x, 'f', -1, 64))
	case []any:
		vals := make([]any, len(x))
		for i, e := range x {
			vals[i] = toValue(e)
		}
		return Array(vals...)
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		members := make([]*Node, len(keys))
		for i, k := range keys {
			members[i] = NewMember(k, toValue(x[k]))
		}
		return NewObject(members...)
	default:
		return NewString(fmt.Sprint(x))
	}
}

func toIntValue(v any) *Node {
	switch x := v.(type) {
	case int:
		return NewNumber(strconv.Itoa(x))
	case int8:
		return NewNumber(strconv.FormatInt(int64(x), 10))
	case int16:
		return NewNumber(strconv.FormatInt(int64(x), 10))
	case int32:
		return NewNumber(strconv.FormatInt(int64(x), 10))
	case int64:
		return NewNumber(strconv.FormatInt(x, 10))
	case uint:
		return NewNumber(strconv.FormatUint(uint64(x), 10))
	case uint8:
		return NewNumber(strconv.FormatUint(uint64(x), 10))
	case uint16:
		return NewNumber(strconv.FormatUint(uint64(x), 10))
	case uint32:
		return NewNumber(strconv.FormatUint(uint64(x), 10))
	case uint64:
		return NewNumber(strconv.FormatUint(x, 10))
	default:
		return NewNumber(fmt.Sprint(x))
	}
}

// ---------------------------------------------------------------------------
// Mutable setter — add or update members on an Object.
// ---------------------------------------------------------------------------

// Get returns the value node for a given key in an Object, or nil if not found.
func (n *Node) Get(key string) *Node {
	if n == nil || n.Kind != KindObject {
		return nil
	}
	for _, c := range n.Children {
		if c.Kind == KindMember && memberKeyEquals(c, key) {
			return c.ValueNode()
		}
	}
	return nil
}

// Delete removes a member by key from an Object.
// Returns the receiver for fluent chaining.
func (n *Node) Delete(key string) *Node {
	if n == nil || n.Kind != KindObject {
		return n
	}
	for i, c := range n.Children {
		if c.Kind == KindMember && memberKeyEquals(c, key) {
			// Remove the member and one adjacent comma.
			removeStart, removeEnd := i, i+1
			if i > 0 && n.Children[i-1].Kind == KindComma {
				removeStart = i - 1 // comma before member
			} else if i+1 < len(n.Children) && n.Children[i+1].Kind == KindComma {
				removeEnd = i + 2 // comma after member
			}
			n.Children = append(n.Children[:removeStart], n.Children[removeEnd:]...)
			return n
		}
	}
	return n
}

// Len returns the number of members in an Object.
// Returns 0 for nil, non-Object, or empty objects.
func (n *Node) Len() int {
	if n == nil || n.Kind != KindObject {
		return 0
	}
	count := 0
	for _, c := range n.Children {
		if c.Kind == KindMember {
			count++
		}
	}
	return count
}

// Keys returns the member keys of an Object in document order.
// Returns nil for nil, non-Object, or empty objects.
func (n *Node) Keys() []string {
	if n == nil || n.Kind != KindObject {
		return nil
	}
	keys := make([]string, 0, n.Len())
	for _, c := range n.Children {
		if c.Kind == KindMember {
			if kn := c.KeyNode(); kn != nil {
				keys = append(keys, strings.Trim(kn.Value, `"`))
			}
		}
	}
	return keys
}

// Values returns the value nodes of an Object in document order.
// Returns nil for nil, non-Object, or empty objects.
func (n *Node) Values() []*Node {
	if n == nil || n.Kind != KindObject {
		return nil
	}
	vals := make([]*Node, 0, n.Len())
	for _, c := range n.Children {
		if c.Kind == KindMember {
			vals = append(vals, c.ValueNode())
		}
	}
	return vals
}

// Has reports whether a member with the given key exists in an Object.
func (n *Node) Has(key string) bool {
	if n == nil || n.Kind != KindObject {
		return false
	}
	for _, c := range n.Children {
		if c.Kind == KindMember && memberKeyEquals(c, key) {
			return true
		}
	}
	return false
}

func memberKeyEquals(n *Node, key string) bool {
	if kn := n.KeyNode(); kn != nil {
		return strings.Trim(kn.Value, `"`) == key
	}
	return false
}

// Set adds or updates a member on an Object node.
//   - If key already exists, its value (and optional comment) is replaced.
//   - If key does not exist, a new member is appended.
//   - comments (optional) are added as trailing line comments after the value.
//
// Returns the receiver for fluent chaining:
//
//	obj := jsonc.Object("a", 1).Set("b", 2).Set("c", 3)
func (n *Node) Set(key string, value any, comments ...string) *Node {
	if n == nil || n.Kind != KindObject {
		return n
	}

	val := toValue(value)

	// Build the member node.
	children := []*Node{
		NewString(key),
		{Kind: KindColon, Value: ":"},
		{Kind: KindWhitespace, Value: " "},
		val,
	}
	for _, c := range comments {
		children = append(children, NewCommentLine(c))
	}
	member := &Node{Kind: KindMember, Children: children}

	// Try to update an existing member with the same key.
	for i, c := range n.Children {
		if c.Kind == KindMember && memberKeyEquals(c, key) {
			n.Children[i] = member
			return n
		}
	}

	// Append new member before the closing brace.
	tail := n.Children[len(n.Children)-1]
	comma := &Node{Kind: KindComma, Value: ","}
	n.Children = append(n.Children[:len(n.Children)-1], comma, member, tail)
	return n
}
