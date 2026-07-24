package jsonc_test

import (
	"fmt"
	"strings"

	"github.com/fan92rus/jsonc-cst"
)

// Compact constructor: Object takes key-value pairs.
func ExampleObject() {
	doc := jsonc.Object(
		"host", "localhost",
		"port", 8080,
	)
	fmt.Print(jsonc.Format(doc, &jsonc.FormatOptions{Indent: "  "}))
	// Output:
	// {
	//   "host": "localhost",
	//   "port": 8080
	// }
}

// Nested Object and Array work recursively.
func ExampleObject_nested() {
	doc := jsonc.Object(
		"server", jsonc.Object(
			"host", "example.com",
			"port", 443,
		),
		"tags", jsonc.Array("prod", "us-east"),
	)
	fmt.Print(jsonc.Format(doc, &jsonc.FormatOptions{Indent: "  "}))
	// Output:
	// {
	//   "server": {
	//     "host": "example.com",
	//     "port": 443
	//   },
	//   "tags": [
	//     "prod",
	//     "us-east"
	//   ]
	// }
}

// Compact array with mixed types.
func ExampleArray_mixed() {
	doc := jsonc.Array("hello", 42, true, nil, 3.14)
	fmt.Print(jsonc.Format(doc, &jsonc.FormatOptions{Indent: ""}))
	// Output:
	// [
	// "hello",
	// 42,
	// true,
	// null,
	// 3.14
	// ]
}

// Setter: add a member to an existing object.
func ExampleNode_Set_simple() {
	doc := jsonc.Object("name", "Alice")
	doc.Set("age", 30)
	fmt.Print(jsonc.Format(doc, &jsonc.FormatOptions{Indent: "  "}))
	// Output:
	// {
	//   "name": "Alice",
	//   "age": 30
	// }
}

// Setter with comment: trailing line comment.
func ExampleNode_Set_comment() {
	doc := jsonc.Object("host", "localhost")
	doc.Set("port", 8080, "default port")
	out := jsonc.Format(doc, &jsonc.FormatOptions{Indent: "  "})
	fmt.Print("comment count: ", strings.Count(out, "//"), "\n")
	fmt.Print("port value: ", strings.Contains(out, "8080"), "\n")
	// Output:
	// comment count: 1
	// port value: true
}

// Setter: update an existing key's value.
func ExampleNode_Set_update() {
	doc := jsonc.Object("mode", "strict")
	doc.Set("mode", "lax")
	fmt.Print(jsonc.Format(doc, &jsonc.FormatOptions{Indent: ""}))
	// Output:
	// {
	// "mode": "lax"
	// }
}

// Setter chain: fluent interface.
func ExampleNode_Set_chain() {
	doc := jsonc.Object("a", 1).
		Set("b", 2).
		Set("c", 3)
	fmt.Print(jsonc.Format(doc, &jsonc.FormatOptions{Indent: ""}))
	// Output:
	// {
	// "a": 1,
	// "b": 2,
	// "c": 3
	// }
}

// Getter: read a value by key.
func ExampleNode_Get() {
	doc := jsonc.Object("host", "localhost", "port", 8080)
	fmt.Println(doc.Get("host").Value)
	fmt.Println(doc.Get("port").Value)
	fmt.Println(doc.Get("missing"))
	// Output:
	// "localhost"
	// 8080
	// <nil>
}

// Delete: remove a member by key.
func ExampleNode_Delete() {
	doc := jsonc.Object("a", 1, "b", 2, "c", 3)
	doc.Delete("b")
	fmt.Print(jsonc.Format(doc, &jsonc.FormatOptions{Indent: ""}))
	// Output:
	// {
	// "a": 1,
	// "c": 3
	// }
}

// Delete chain: fluent interface.
func ExampleNode_Delete_chain() {
	doc := jsonc.Object("a", 1, "b", 2, "c", 3).
		Delete("a").
		Delete("c")
	fmt.Print(jsonc.Format(doc, &jsonc.FormatOptions{Indent: ""}))
	// Output:
	// {
	// "b": 2
	// }
}

// Len: count members in an Object.
func ExampleNode_Len() {
	empty := jsonc.Object()
	single := jsonc.Object("x", 1)
	multi := jsonc.Object("a", 1, "b", 2, "c", 3)
	nested := jsonc.Object("inner", jsonc.Object("y", 2))
	fmt.Println(empty.Len())
	fmt.Println(single.Len())
	fmt.Println(multi.Len())
	fmt.Println(nested.Len())
	fmt.Println((*jsonc.Node)(nil).Len())
	fmt.Println(jsonc.Array(1, 2).Len())
	// Output:
	// 0
	// 1
	// 3
	// 1
	// 0
	// 0
}

// Keys: list member keys in document order.
func ExampleNode_Keys() {
	doc := jsonc.Object("beta", 2, "alpha", 1, "gamma", 3)
	fmt.Println(doc.Keys())
	fmt.Println(jsonc.Object().Keys())
	fmt.Println((*jsonc.Node)(nil).Keys()) // nil slice prints as []
	fmt.Println(jsonc.Array(1, 2).Keys())  // non-Object prints as []
	// Output:
	// [beta alpha gamma]
	// []
	// []
	// []
}

// Has: check if a key exists.
func ExampleNode_Has() {
	doc := jsonc.Object("x", 10, "y", 20)
	fmt.Println(doc.Has("x"))
	fmt.Println(doc.Has("z"))
	fmt.Println(jsonc.Object().Has("x"))
	fmt.Println((*jsonc.Node)(nil).Has("x"))
	fmt.Println(jsonc.Array(1, 2).Has("x"))
	// Output:
	// true
	// false
	// false
	// false
	// false
}

// Values: list value nodes in document order.
func ExampleNode_Values() {
	doc := jsonc.Object("a", 10, "b", 20)
	for _, v := range doc.Values() {
		fmt.Println(v.Value)
	}
	fmt.Println(jsonc.Object().Values())
	// Output:
	// 10
	// 20
	// []
}
