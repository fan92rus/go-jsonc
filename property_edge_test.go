// Package jsonc tests — reflectToNode and edge coverage for struct serialization
package jsonc

import (
	"reflect"
	"strings"
	"testing"
)

// ============================================================================
// reflectToNode: pointer fields
// ============================================================================

type ptrFieldsStruct struct {
	NilPtr *int    `json:"nil_ptr"`
	ValPtr *int    `json:"val_ptr"`
	NilStr *string `json:"nil_str"`
	ValStr *string `json:"val_str"`
}

// propMarshalNilPtr verifies nil pointer fields become null.
func TestProperty_MarshalNilPtr(t *testing.T) {
	v := 42
	s := "hello"
	src := ptrFieldsStruct{
		NilPtr: nil,
		ValPtr: &v,
		NilStr: nil,
		ValStr: &s,
	}
	obj, err := structToObject(src)
	if err != nil {
		t.Fatalf("structToObject error: %v", err)
	}
	out := Serialize(&Node{Kind: KindDocument, Children: []*Node{obj}})
	if !strings.Contains(out, `"nil_ptr": null`) {
		t.Fatalf("nil ptr should serialize to null:\n%s", out)
	}
	if !strings.Contains(out, `"val_ptr": 42`) {
		t.Fatalf("val ptr should serialize to 42:\n%s", out)
	}
	if !strings.Contains(out, `"nil_str": null`) {
		t.Fatalf("nil string ptr should serialize to null:\n%s", out)
	}
	if !strings.Contains(out, `"val_str": "hello"`) {
		t.Fatalf("val string ptr should serialize:\n%s", out)
	}

	// Round-trip through parse + UnmarshalPath
	doc, err := Parse(out)
	if err != nil {
		t.Fatalf("Parse produced invalid JSON: %s\n%s", err, out)
	}
	var dst ptrFieldsStruct
	_ = doc.Root().UnmarshalPath("", &dst)
	if dst.ValPtr == nil || *dst.ValPtr != 42 {
		t.Fatal("ValPtr should be 42 after round-trip")
	}
	if dst.ValStr == nil || *dst.ValStr != "hello" {
		t.Fatal("ValStr should be 'hello' after round-trip")
	}
}

// ============================================================================
// reflectToNode: nested struct field
// ============================================================================

type nestedInner struct {
	X int    `json:"x"`
	Y string `json:"y"`
}

type nestedOuter struct {
	Label string      `json:"label"`
	Inner nestedInner `json:"inner"`
}

// propMarshalNestedStruct verifies nested struct fields serialize and round-trip.
func TestProperty_MarshalNestedStruct(t *testing.T) {
	src := nestedOuter{
		Label: "test",
		Inner: nestedInner{X: 10, Y: "deep"},
	}
	obj, err := structToObject(src)
	if err != nil {
		t.Fatalf("structToObject error: %v", err)
	}
	out := Serialize(&Node{Kind: KindDocument, Children: []*Node{obj}})
	if !strings.Contains(out, `"x": 10`) {
		t.Fatalf("nested field x not found:\n%s", out)
	}
	if !strings.Contains(out, `"y": "deep"`) {
		t.Fatalf("nested field y not found:\n%s", out)
	}

	// MarshalPath into existing doc + UnmarshalPath round-trip
	var dst nestedOuter
	doc, err := Parse(`{"label": "x", "inner": {"x": 99, "y": "yy"}}`)
	if err != nil {
		t.Fatal(err)
	}
	_ = doc.Root().MarshalPath("", src)
	_ = doc.Root().UnmarshalPath("", &dst)
	if dst.Label != "test" {
		t.Fatalf("expected label=test, got %q", dst.Label)
	}
	if dst.Inner.X != 10 {
		t.Fatalf("expected inner.x=10, got %d", dst.Inner.X)
	}
	if dst.Inner.Y != "deep" {
		t.Fatalf("expected inner.y=deep, got %q", dst.Inner.Y)
	}
}

// ============================================================================
// reflectToNode: float32, uint8, int8
// ============================================================================

type typeRangeStruct struct {
	F32 float32 `json:"f32"`
	U8  uint8   `json:"u8"`
	I8  int8    `json:"i8"`
	Raw string  `json:"raw"`
}

// propMarshalTypeRanges verifies float32/uint8/int8 via struct serialization.
func TestProperty_MarshalTypeRanges(t *testing.T) {
	src := typeRangeStruct{
		F32: 3.14,
		U8:  255,
		I8:  -128,
		Raw: "text",
	}
	obj, err := structToObject(src)
	if err != nil {
		t.Fatalf("structToObject error: %v", err)
	}
	out := Serialize(&Node{Kind: KindDocument, Children: []*Node{obj}})
	if !strings.Contains(out, `"f32": 3.14`) && !strings.Contains(out, `"f32": 3.140000`) {
		t.Fatalf("float32 not found:\n%s", out)
	}
	if !strings.Contains(out, `"u8": 255`) {
		t.Fatalf("uint8 not found:\n%s", out)
	}
	if !strings.Contains(out, `"i8": -128`) {
		t.Fatalf("int8 not found:\n%s", out)
	}

	// Round-trip
	doc, err := Parse(out)
	if err != nil {
		t.Fatalf("Parse failed: %v\n%s", err, out)
	}
	var dst typeRangeStruct
	_ = doc.Root().UnmarshalPath("", &dst)
	if dst.F32 != 3.14 {
		t.Fatalf("F32 round-trip failed: got %v", dst.F32)
	}
	if dst.U8 != 255 {
		t.Fatalf("U8 round-trip failed: got %d", dst.U8)
	}
}

// ============================================================================
// reflectToNode: structToObject error path (pass non-struct)
// ============================================================================

// propStructToObjectError verifies structToObject returns error for non-struct.
func TestProperty_StructToObjectError(t *testing.T) {
	_, err := structToObject(42)
	if err == nil {
		t.Fatal("expected error for non-struct")
	}
	_, err = structToObject("hello")
	if err == nil {
		t.Fatal("expected error for string")
	}
	_, err = structToObject(nil)
	if err == nil {
		t.Fatal("expected error for nil")
	}
}

// ============================================================================
// reflectToNode: unexported and json:"-" fields excluded
// ============================================================================

type mixedFieldsStruct struct {
	Exported   string `json:"exported"`
	unexported string
	Hidden     string `json:"-"`
}

// propMarshalUnexported verifies unexported and json:"-" fields are excluded.
func TestProperty_MarshalUnexported(t *testing.T) {
	src := mixedFieldsStruct{
		Exported:   "visible",
		unexported: "hidden",
		Hidden:     "also hidden",
	}
	obj, err := structToObject(src)
	if err != nil {
		t.Fatalf("structToObject error: %v", err)
	}
	out := Serialize(&Node{Kind: KindDocument, Children: []*Node{obj}})
	if !strings.Contains(out, `"exported": "visible"`) {
		t.Fatalf("exported field should be present:\n%s", out)
	}
	if strings.Contains(out, "hidden") {
		t.Fatal("unexported field should be excluded")
	}
	if strings.Contains(out, "also") {
		t.Fatal("json:\"-\" field should be excluded")
	}
}

// ============================================================================
// reflectToNode: invalid reflect.Value
// ============================================================================

// propReflectToNodeInvalid verifies invalid reflect.Value returns null.
func TestProperty_ReflectToNodeInvalid(t *testing.T) {
	var nilPtr *int
	invalidVal := reflect.ValueOf(nilPtr).Elem() // zero Value, invalid
	result := reflectToNode(invalidVal)
	if result.Kind != KindNull {
		t.Fatal("invalid reflect.Value should become null")
	}
}

// ============================================================================
// reflectToNode: nil and empty slice
// ============================================================================

type sliceEdgeStruct struct {
	NonNil  []int `json:"non_nil"`
	NilOmit []int `json:"nil_omit,omitempty"`
}

// propReflectToNodeSliceEdge verifies nil/empty slice handling.
func TestProperty_ReflectToNodeSliceEdge(t *testing.T) {
	src := sliceEdgeStruct{
		NonNil:  []int{1, 2, 3},
		NilOmit: nil,
	}
	obj, err := structToObject(src)
	if err != nil {
		t.Fatalf("structToObject error: %v", err)
	}
	out := Serialize(&Node{Kind: KindDocument, Children: []*Node{obj}})
	if !strings.Contains(out, `"non_nil"`) {
		t.Fatalf("non-nil slice should appear:\n%s", out)
	}
	if strings.Contains(out, "nil_omit") {
		t.Fatalf("nil slice with omitempty should not appear:\n%s", out)
	}
}

// ============================================================================
// reflectToNode: nil map
// ============================================================================

type mapEdgeStruct struct {
	NonNil  map[string]int `json:"non_nil_map"`
	NilOmit map[string]int `json:"nil_omit_map,omitempty"`
}

// propReflectToNodeMapEdge verifies nil map handling.
func TestProperty_ReflectToNodeMapEdge(t *testing.T) {
	src := mapEdgeStruct{
		NonNil:  map[string]int{"a": 1},
		NilOmit: nil,
	}
	obj, err := structToObject(src)
	if err != nil {
		t.Fatalf("structToObject error: %v", err)
	}
	out := Serialize(&Node{Kind: KindDocument, Children: []*Node{obj}})
	if !strings.Contains(out, `"non_nil_map"`) {
		t.Fatalf("non-nil map should appear:\n%s", out)
	}
	if strings.Contains(out, "nil_omit") {
		t.Fatalf("nil map with omitempty should not appear:\n%s", out)
	}
}

// ============================================================================
// serializePlainJSON with nil
// ============================================================================

// propSerializePlainJSONNil verifies nil input is handled.
func TestProperty_SerializePlainJSONNil(t *testing.T) {
	result := serializePlainJSON(nil)
	if result != nil {
		t.Fatal("serializePlainJSON(nil) should return nil")
	}
}

// ============================================================================
// Walk on non-container
// ============================================================================

// propWalkOnString verifies Walk on a string leaf node calls fn once.
func TestProperty_WalkOnString(t *testing.T) {
	n := NewString("hello")
	count := 0
	n.Walk(func(_ *Node) bool {
		count++
		return true
	})
	if count != 1 {
		t.Fatalf("Walk on string should call fn once, called %d", count)
	}
}

// ============================================================================
// NodeKind.String edge case
// ============================================================================

// propKindString verifies NodeKind.String for unknown kind.
func TestProperty_KindString(t *testing.T) {
	var uk NodeKind = 127
	if !strings.Contains(uk.String(), "127") {
		t.Fatalf("unknown kind should contain value, got %s", uk.String())
	}
}

// ============================================================================
// KeyNode edge cases
// ============================================================================

// propKeyNodeNonMember verifies KeyNode returns nil for non-member nodes.
func TestProperty_KeyNodeNonMember(t *testing.T) {
	n := NewString("hello")
	if n.KeyNode() != nil {
		t.Fatal("KeyNode on non-member should return nil")
	}
	obj := Object("a", 1)
	// Object itself is not a member
	if obj.KeyNode() != nil {
		t.Fatal("KeyNode on Object should return nil")
	}
}
