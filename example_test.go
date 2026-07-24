package jsonc_test

import (
	"fmt"
	"log"
	"strings"

	"github.com/fan92rus/go-jsonc"
)

func ExampleParse() {
	src := `{"name": "hello", "value": 42}`
	doc, err := jsonc.Parse(src)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(doc.Kind)
	// Output: Document
}

func ExampleParse_jsonc() {
	src := `{
  // user profile
  "name": "Alice",
  "age": 30 /* years */
}`
	doc, err := jsonc.Parse(src)
	if err != nil {
		log.Fatal(err)
	}
	comments := doc.FindAll(jsonc.KindComment)
	fmt.Println(len(comments))
	// Output: 2
}

func ExampleFormat() {
	src := `{"a":1,"b":2}`
	doc, err := jsonc.Parse(src)
	if err != nil {
		log.Fatal(err)
	}
	formatted := jsonc.Format(doc, &jsonc.FormatOptions{Indent: "  "})
	fmt.Println(formatted)
	// Output:
	// {
	//   "a": 1,
	//   "b": 2
	// }
}

func ExampleNode_Walk() {
	src := `[true, false, null]`
	doc, err := jsonc.Parse(src)
	if err != nil {
		log.Fatal(err)
	}
	var kinds []string
	doc.Walk(func(n *jsonc.Node) bool {
		if n.IsValue() {
			kinds = append(kinds, n.Kind.String())
		}
		return true
	})
	fmt.Println(kinds)
	// Output: [Array Boolean Boolean Null]
}

func ExampleNewObject() {
	doc := jsonc.NewObject(
		jsonc.NewMember("host", jsonc.NewString("localhost")),
		jsonc.NewMember("port", jsonc.NewNumber("8080")),
	)
	formatted := jsonc.Format(doc, &jsonc.FormatOptions{Indent: "  "})
	fmt.Print(formatted)
	// Output:
	// {
	//   "host": "localhost",
	//   "port": 8080
	// }
}

func ExampleNode_Body() {
	src := `{"a": 1 /* important */}`
	doc, _ := jsonc.Parse(src)
	for _, c := range doc.FindAll(jsonc.KindComment) {
		fmt.Println(c.Body())
	}
	// Output: important
}

func ExampleNode_SetValue() {
	src := `{"name": "Alice"}`
	doc, _ := jsonc.Parse(src)
	obj := doc.FirstChild()
	obj.Members()[0].ValueNode().SetValue(`"Bob"`)
	fmt.Println(jsonc.Serialize(doc))
	// Output: {"name": "Bob"}
}

func ExampleNewArray() {
	doc := jsonc.NewArray(
		jsonc.NewNumber("1"),
		jsonc.NewNumber("2"),
		jsonc.NewNumber("3"),
	)
	fmt.Println(jsonc.Format(doc, nil))
	// Output:
	// [
	//   1,
	//   2,
	//   3
	// ]
}

func ExampleNewCommentLine_builder() {
	doc := jsonc.NewObject(
		jsonc.NewMember("key", jsonc.NewString("value"),
			jsonc.NewCommentLine("note"),
		),
	)
	out := jsonc.Format(doc, &jsonc.FormatOptions{Indent: "  "})
	fmt.Println(strings.Count(out, "//"))
	// Output: 1
}

func ExampleNewCommentBlock_builder() {
	doc := jsonc.NewObject(
		jsonc.NewMember("count", jsonc.NewNumber("42"),
			jsonc.NewCommentBlock("answer"),
		),
	)
	out := jsonc.Format(doc, &jsonc.FormatOptions{Indent: "  "})
	fmt.Println(strings.Contains(out, "answer"))
	// Output: true
}
