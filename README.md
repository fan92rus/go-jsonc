# jsonc-cst

[![Go Reference](https://pkg.go.dev/badge/github.com/fan92rus/jsonc-cst.svg)](https://pkg.go.dev/github.com/fan92rus/jsonc-cst)
[![Go Report Card](https://goreportcard.com/badge/github.com/fan92rus/jsonc-cst)](https://goreportcard.com/report/github.com/fan92rus/jsonc-cst)
[![CI](https://github.com/fan92rus/jsonc-cst/actions/workflows/ci.yml/badge.svg)](https://github.com/fan92rus/jsonc-cst/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A **Concrete Syntax Tree (CST)** parser, serializer, and pretty-printer for **JSONC (JSON with Comments)** in Go.

```
go get github.com/fan92rus/jsonc-cst
```

## Why CST, not AST?

A CST preserves **everything** — every comment, every space, every formatting choice.
When you parse a file, edit a comment, and serialize it back, the original formatting
is preserved. This is essential for config file management where comments carry meaning.

## Quick start

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

	// Pretty-print
	formatted := jsonc.Format(doc, &jsonc.FormatOptions{Indent: "  "})
	fmt.Println(formatted)

	// Serialize back to text (preserving original formatting)
	text := jsonc.Serialize(doc)
	fmt.Println(text)
}
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

Serializes a CST node tree back into its source text. Produces identical output
to the original input for valid JSON (without comments). For JSONC, the
parse→serialize round-trip is structure-preserving.

### Formatting

```go
formatted := jsonc.Format(doc, &jsonc.FormatOptions{Indent: "  "})
```

Pretty-prints a CST node tree. The `Indent` field controls indentation:
- `"  "` — two-space indent (default)
- `"\t"` — tab indent
- `"    "` — four-space indent
- `""` — no indent (compact)

Comments are preserved and properly positioned.

### Node tree navigation

```go
// Walk all nodes
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
- ✅ Parse→serialize identity and idempotence
- ✅ Format preserves semantics across all indent styles
- ✅ Format idempotence (re-formatting produces identical output)
- ✅ Position tracking (monotonic, covers entire input)
- ✅ Deep nesting (500+ levels)
- ✅ Error recovery (truncated input, invalid constructs)
- ✅ Trailing commas, Unicode, escape sequences, number variations

## Contributing

1. Fork the repo
2. Create a feature branch
3. Run `go test ./...` and `golangci-lint run ./...`
4. Submit a PR

## License

MIT — see [LICENSE](LICENSE).
