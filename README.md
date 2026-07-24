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

### Compact builder (recommended)

```go
package main

import (
	"fmt"
	"github.com/fan92rus/jsonc-cst"
)

func main() {
	// Build a config with auto-typed values
	doc := jsonc.Object(
		"host", "localhost",
		"port", 8080,
		"debug", true,
		"tags", jsonc.Array("dev", "test"),
		"nested", jsonc.Object("timeout", 30),
	)
	doc.
		Set("mode", "strict", "override").
		Set("port", 9090)

	fmt.Println(jsonc.Format(doc, &jsonc.FormatOptions{Indent: "  "}))
}
```

Output:

```json
{
  "debug": true,
  "host": "localhost",
  "mode": "strict" // override
  ,
  "nested": {
    "timeout": 30
  },
  "port": 9090,
  "tags": [
    "dev",
    "test"
  ]
}
```

### Parse, edit, format

```go
src := `{"name": "Alice", "age": 30}`
doc, _ := jsonc.Parse(src)

doc.Set("age", 31)        // update
doc.Set("city", "Berlin") // add

fmt.Println(jsonc.Format(doc, nil))
// {
//   "age": 31,
//   "city": "Berlin",
//   "name": "Alice"
// }
```

### Read values

```go
doc, _ := jsonc.Parse(`{"host": "local", "port": 8080}`)

fmt.Println(doc.Get("host").Value)
fmt.Println(doc.Get("port").Value)
fmt.Println(doc.Has("host"))   // true
fmt.Println(doc.Keys())        // [host port]
fmt.Println(doc.Len())         // 2
// Output:
// "local"
// 8080
// true
// [host port]
// 2
```

## Building JSONC

### Compact constructors (simple, recommended)

`Object(key, val, key, val, ...)` wraps Go values automatically:

```go
doc := jsonc.Object(
	"name",   "Alice",
	"age",    30,
	"active", true,
	"data",   nil,         // → null
	"tags",   jsonc.Array("a", "b"),
	"meta",   jsonc.Object("key", "val"),
)
```

`Array(elem, elem, ...)` does the same for arrays:

```go
arr := jsonc.Array("hello", 42, true, nil, 3.14, jsonc.Object("x", 1))
```

### Mutation (fluent)

```go
doc := jsonc.Object("a", 1)
doc.
	Set("b", 2).              // add member
	Set("a", 10).             // update existing
	Set("c", 3, "comment")    // with trailing comment
	Delete("b")               // remove member

v := doc.Get("a")            // value node for "a"
fmt.Println(doc.Has("c"))    // true
fmt.Println(doc.Keys())      // [a c]
fmt.Println(doc.Len())       // 2
```

### Node tree navigation

```go
for _, key := range doc.Keys() {
	fmt.Println("key:", key)
}

for _, val := range doc.Values() {
	fmt.Println("value:", val.Value)
}

for _, m := range doc.Members() {
	fmt.Println(m.KeyNode().Value, "→", m.ValueNode().Value)
}
```

### Extended API (verbose, for fine control)

When you need full control over CST node placement:

```go
obj := jsonc.NewObject(
	jsonc.NewCommentLine(" Auto-generated"),
	jsonc.NewMember("host", jsonc.NewString("localhost")),
	jsonc.NewMember("port", jsonc.NewNumber("8080"),
		jsonc.NewCommentLine(" default"),
	),
)
```

The compact constructors `Object()` and `Array()` are built on top of these
primitives. Use the extended API when you need to insert comments, control
node order explicitly, or mix in whitespace/trivia nodes.

## Parsing

```go
doc, err := jsonc.Parse(src)
```

Parses a JSONC string into a CST [`*Node`](https://pkg.go.dev/github.com/fan92rus/jsonc-cst#Node).
Valid JSON and JSONC (with `//` and `/* */` comments) are both accepted. Malformed
input produces error nodes in the tree rather than failing catastrophically.

## Serialization

```go
text := jsonc.Serialize(doc)
```

Serializes a CST node tree back into source text. Produces identical output
to the original input (lossless round-trip). Works for both JSON and JSONC.

## Formatting (pretty-print)

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

## Node tree (lower-level API)

### Scalar nodes

```go
s  := jsonc.NewString("hello")   // → "hello"
n  := jsonc.NewNumber("42")      // → 42
b  := jsonc.NewBoolean(true)     // → true
nu := jsonc.NewNull()            // → null
lc := jsonc.NewCommentLine("hi") // → // hi
bc := jsonc.NewCommentBlock("x") // → /* x */
```

### Traversal

```go
// Walk all nodes depth-first
doc.Walk(func(n *jsonc.Node) bool {
	fmt.Println(n.Kind, n.Value)
	return true
})

// Find by kind
strings := doc.FindAll(jsonc.KindString)
comments := doc.FindAll(jsonc.KindComment)

// Container access
for _, m := range obj.Members() { /* ... */ }
for _, e := range arr.Elements() { /* ... */ }

// Comment body
fmt.Println(c.Body())        // "text without delimiters"
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

## Property-based testing

The test suite uses [rapid](https://pgregory.net/rapid) for property-based testing
with generators that produce random valid JSONC documents. Properties tested:

- ✅ All valid JSON/JSONC parses without errors
- ✅ Comment preservation (line, block, mixed, every position)
- ✅ Parse → serialize identity and idempotence
- ✅ Format preserves semantics across all indent styles
- ✅ Format idempotence (re-formatting produces identical output)
- ✅ Position tracking (monotonic, covers entire input)
- ✅ Deep nesting (500+ levels)
- ✅ Error recovery (truncated input, invalid constructs)
- ✅ Trailing commas, Unicode, escape sequences, number variations

Zero lint issues with 22+ linters (golangci-lint max-strict config).

## Project status

**Early production** — core API (parse, serialize, format, navigate) is
stable and well-tested. The Builder and Mutation API is new and under
active development. Backward compatibility is guaranteed within the v0.x series.

## Contributing

1. Fork the repo
2. Create a feature branch
3. Run `go test ./...` and `golangci-lint run ./...`
4. Submit a PR

## License

MIT — see [LICENSE](LICENSE).
