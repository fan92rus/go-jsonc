# jsonc-cst

[![Go Reference](https://pkg.go.dev/badge/github.com/fan92rus/jsonc-cst.svg)](https://pkg.go.dev/github.com/fan92rus/jsonc-cst)
[![Go Report Card](https://goreportcard.com/badge/github.com/fan92rus/jsonc-cst)](https://goreportcard.com/report/github.com/fan92rus/jsonc-cst)
[![CI](https://github.com/fan92rus/jsonc-cst/actions/workflows/ci.yml/badge.svg)](https://github.com/fan92rus/jsonc-cst/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A **Concrete Syntax Tree (CST)** parser, serializer, formatter, and builder for **JSONC (JSON with Comments)** in Go.

```
go get github.com/fan92rus/jsonc-cst
```

## Why CST, not AST?

A CST preserves **everything** — every comment, every space, every formatting choice.
When you parse a file, edit a comment, and serialize it back, the original formatting
is preserved. This is essential for config file management where comments carry meaning.

## Quick start

### Parse JSONC with comments

```go
package main

import (
	"fmt"
	"log"

	"github.com/fan92rus/jsonc-cst"
)

func main() {
	src := `{
  // Connection settings
  "host": "localhost",
  "port": 8080  // default port
}`
	doc, err := jsonc.Parse([]byte(src))
	if err != nil {
		log.Fatal(err)
	}

	// Find all comments
	for _, c := range doc.FindAllComments() {
		fmt.Println("Comment:", c.Body())
	}

	// Pretty-print with two-space indent
	formatted := jsonc.Format(doc, &jsonc.FormatOptions{Indent: "  "})
	fmt.Println(formatted)

	// Serialize back to text (preserving original formatting)
	text := jsonc.Serialize(doc)
	fmt.Println(text)
}
```

### Build JSON from scratch (Builder API)

```go
package main

import (
	"fmt"

	"github.com/fan92rus/jsonc-cst"
)

func main() {
	// Build a JSONC document programmatically
	doc := jsonc.NewObject(
		jsonc.NewCommentLine(" Auto-generated config"),
		jsonc.NewMember("host", jsonc.NewString("localhost")),
		jsonc.NewMember("port", jsonc.NewNumber("8080"),
			jsonc.NewCommentLine(" default port"),
		),
	)

	fmt.Println(jsonc.Format(doc, &jsonc.FormatOptions{Indent: "  "}))
	// Output:
	// {
	//   // Auto-generated config
	//   "host": "localhost",
	//   "port": 8080 // default port
	// }
}
```

### Read and edit values

```go
src := `{"name": "Alice", "age": 30}`
doc, _ := jsonc.Parse([]byte(src))

obj := doc.FirstChild()
for _, m := range obj.Members() {
    fmt.Println("Key:", m.KeyNode().Value, "→ Value:", m.ValueNode().Value)
}

// Change a value
obj.Members()[0].ValueNode().SetValue(`"Bob"`)

fmt.Println(jsonc.Serialize(doc))
// Output: {"name": "Bob", "age": 30}
```

## API

### Parsing

```go
doc, err := jsonc.Parse([]byte(input))
```

Parses a JSONC byte slice into a CST [`*Node`](https://pkg.go.dev/github.com/fan92rus/jsonc-cst#Node).
Valid JSON and JSONC (with `//` and `/* */` comments) are both accepted. Malformed
input produces error nodes in the tree rather than failing catastrophically.

### Serialization

```go
text := jsonc.Serialize(doc)
```

Serializes a CST node tree back into source text. Produces identical output
to the original input (lossless round-trip). Works for both JSON and JSONC.

### Formatting (pretty-print)

```go
formatted := jsonc.Format(doc, &jsonc.FormatOptions{Indent: "  "})
```

Pretty-prints a CST node tree. The `Indent` field controls indentation:

| `Indent` | Style |
|----------|-------|
| `"  "` (or `nil`) | Two-space indent (default) |
| `"\t"` | Tab indent |
| `"    "` | Four-space indent |
| `""` | Compact/minified (no indentation) |

Comments are preserved and properly positioned.

### Builder API (programmatic creation)

Create JSONC structures from code without parsing:

```go
// Scalar values
s := jsonc.NewString("hello")         // → "hello"
n := jsonc.NewNumber("42")            // → 42
b := jsonc.NewBoolean(true)           // → true
nu := jsonc.NewNull()                 // → null

// Comments
lc := jsonc.NewCommentLine(" note")   // → // note
bc := jsonc.NewCommentBlock("cfg")    // → /* cfg */

// Containers
arr := jsonc.NewArray(e1, e2, e3)
obj := jsonc.NewObject(
    jsonc.NewMember("key", jsonc.NewString("value")),
    jsonc.NewMember("count", jsonc.NewNumber("3")),
)
```

### Mutation API

Modify parsed or built trees:

```go
node.SetValue(`"new text"`)           // Change a leaf node's text
node.SetCommentBody("updated text")   // Update a comment's body text
node.AppendChild(child)               // Add a child to a container
```

### Node tree navigation

```go
// Walk all nodes depth-first
doc.Walk(func(n *jsonc.Node) bool {
    fmt.Println(n.Kind, n.Value)
    return true
})

// Find nodes by type
strings := doc.FindAll(jsonc.KindString)
comments := doc.FindAllComments()

// Navigate structure
for _, member := range obj.Members() {
    key := member.KeyNode()
    val := member.ValueNode()
}

for _, elem := range arr.Elements() {
    // ...
}

// Get comment body
for _, c := range comments {
    fmt.Println(c.Body())     // text without delimiters
    fmt.Println(c.CommentBody) // same field, direct access
}
```

## Node kinds

| Kind | Description |
|------|-------------|
| `KindDocument` | Root node |
| `KindObject` | `{ }` |
| `KindArray` | `[ ]` |
| `KindMember` | `"key": value` |
| `KindString` | `"..."` |
| `KindNumber` | `123`, `-1.5e10` |
| `KindBoolean` | `true`, `false` |
| `KindNull` | `null` |
| `KindComment` | `//…` or `/*…*/` |
| `KindWhitespace` | Spaces, tabs, newlines |
| `KindComma` | `,` |
| `KindColon` | `:` |
| `KindLBrace` / `KindRBrace` | `{` / `}` |
| `KindLBracket` / `KindRBracket` | `[` / `]` |
| `KindError` | Error recovery node |

## Project status

**Early production** — the core API (parse, serialize, format, navigate) is
stable and well-tested. The Builder and Mutation API (v0.2) is new and under
active development. Backward compatibility is guaranteed within the v0.x series.

See [docs/REVIEW.md](docs/REVIEW.md) for the full architectural and code review.

## Property-based testing

The test suite uses [rapid](https://pgregory.net/rapid) for property-based testing
with generators that produce random valid JSONC documents. Properties tested:

- ✅ All valid JSON/JSONC parses without errors
- ✅ Comment preservation (line, block, mixed, every position)
- ✅ Parse→serialize identity and idempotence
- ✅ Format preserves semantics across all indent styles
- ✅ Format idempotence (re-formatting produces identical output)
- ✅ Position tracking (monotonic, covers entire input)
- ✅ Deep nesting (500+ levels)
- ✅ Error recovery (truncated input, invalid constructs)
- ✅ Trailing commas, Unicode, escape sequences, number variations

Zero lint issues with 22+ linters (golangci-lint max-strict config).

## Contributing

1. Fork the repo
2. Create a feature branch
3. Run `go test ./...` and `golangci-lint run ./...`
4. Submit a PR

## License

MIT — see [LICENSE](LICENSE).
