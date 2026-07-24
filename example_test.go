package jsonc_test

import (
	"fmt"
	"log"

	"github.com/fan92rus/jsonc-cst"
)

func ExampleParse() {
	src := `{"name": "hello", "value": 42}`
	doc, err := jsonc.Parse([]byte(src))
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
	doc, err := jsonc.Parse([]byte(src))
	if err != nil {
		log.Fatal(err)
	}
	comments := doc.FindAllComments()
	fmt.Println(len(comments))
	// Output: 2
}

func ExampleFormat() {
	src := `{"a":1,"b":2}`
	doc, err := jsonc.Parse([]byte(src))
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
	doc, err := jsonc.Parse([]byte(src))
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
