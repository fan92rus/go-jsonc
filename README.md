# go-jsonc

[![Go Reference](https://pkg.go.dev/badge/github.com/fan92rus/go-jsonc.svg)](https://pkg.go.dev/github.com/fan92rus/go-jsonc)
[![Go Report Card](https://goreportcard.com/badge/github.com/fan92rus/go-jsonc)](https://goreportcard.com/report/github.com/fan92rus/go-jsonc)
[![CI](https://github.com/fan92rus/go-jsonc/actions/workflows/ci.yml/badge.svg)](https://github.com/fan92rus/go-jsonc/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A **Concrete Syntax Tree (CST)** parser, serializer, formatter, and builder
for **JSONC (JSON with Comments)** in Go.

```
go get github.com/fan92rus/go-jsonc
```

## Why CST, not AST?

A CST preserves **everything** — every comment, every space, every formatting
choice. When you parse a file, edit a value, and serialize it back, the
original formatting and comments are preserved. This is essential for config
file management where comments carry meaning.

---

## Quick start

### Build JSONC from scratch

```go
doc := jsonc.Object(
	"host", "localhost",
	"port", 8080,
	"debug", true,
	"tags", jsonc.Array("dev", "test"),
)
doc.Set("mode", "strict").Set("port", 9090)

fmt.Println(jsonc.Format(doc, &jsonc.FormatOptions{Indent: "  "}))
```

Output:

```json
{
  "host": "localhost",
  "port": 9090,
  "debug": true,
  "tags": [
    "dev",
    "test"
  ],
  "mode": "strict"
}
```

### Parse, edit, format

```go
src := `{"name": "Alice", "age": 30}`
doc, _ := jsonc.Parse(src)
obj := doc.Root() // skip Document → root Object

obj.Set("age", 31)
obj.Set("city", "Berlin")

fmt.Println(jsonc.Format(obj, nil))
// {
//   "name": "Alice",
//   "age": 31,
//   "city": "Berlin"
// }
```

---

## Path navigation (dot paths)

Navigate into nested objects and arrays with dot-separated paths.
**Numeric segments index into Arrays** — ideal for real config files.

```go
doc, _ := jsonc.Parse(`{
  "outbounds": [
    {"tag": "proxy-1", "port": 443, "settings": {"tls": true}},
    {"tag": "proxy-2", "port": 8443}
  ]
}`)
root := doc.Root()

// Read
fmt.Println(root.GetPath("outbounds.0.tag").Value)       // "proxy-1"
fmt.Println(root.GetPath("outbounds.1.port").Value)       // 8443
fmt.Println(root.GetPath("outbounds.0.settings.tls").Value) // true

// Write
root.SetPath("outbounds.0.port", 8080)
root.SetPath("outbounds.0.settings.tls", false)

// Delete
root.DeletePath("outbounds.1")           // removes second element
fmt.Println(root.GetPath("outbounds"))   // only proxy-1 remains

// Struct bindings
type Outbound struct {
	Tag string `json:"tag"`
	Port int   `json:"port"`
}
var ob Outbound
root.UnmarshalPath("outbounds.0", &ob)   // → {proxy-1 8080}
```

### Path operations cheat sheet

| Expression | Result |
|---|---|
| `GetPath("port")` | Value of member `"port"` |
| `GetPath("outbounds.0.tag")` | Value at Object→Array→Object→key |
| `SetPath("timeout", 30)` | Add or update a top-level member |
| `SetPath("items.0.port", 80)` | Replace array element's nested field |
| `DeletePath("items.1")` | Remove an array element by index |
| `DeletePath("feature")` | Remove a member |
| `UnmarshalPath("outbound", &v)` | Navigate + deserialize into Go struct |
| `MarshalPath("outbound", v)` | Serialize Go struct → replace subtree |

Missing keys auto-vivify as Objects. Arrays must exist before indexing into
them — `SetPath("items.0", val)` works when `items` is already an Array.

---

## File operations

```go
// Read a JSONC config file
doc, err := jsonc.ParseFile("/opt/etc/xkeen/xray/config.json")

// Navigate and modify
doc.Root().SetPath("outbounds.0.port", 8080)

// Write back (preserves comments and formatting)
err = doc.WriteFile("/opt/etc/xkeen/xray/config.json")
```

---

## Struct binding (MarshalPath / UnmarshalPath)

Convert between JSONC subtrees and Go structs. The optional `jsonc` tag
adds a line comment before the JSON member.

```go
type Config struct {
	Host    string `json:"host"`              // required
	Port    int    `json:"port" jsonc:"Listen port"`
	Debug   bool   `json:"debug,omitempty"`
}

cfg := Config{Host: "localhost", Port: 9090}
doc := jsonc.Object()
doc.MarshalPath("server", cfg)

fmt.Println(jsonc.Format(doc, &jsonc.FormatOptions{Indent: "  "}))
```

Output:

```json
{
  "server": {
    // Listen port
    "port": 9090,
    "host": "localhost"
  }
}
```

Read back:

```go
var got Config
doc.UnmarshalPath("server", &got)
fmt.Println(got.Host) // localhost
```

### jsonc tag styles

| Tag | Result |
|---|---|
| `jsonc:"My comment"` | `// My comment` before the member |
| `jsonc:"// Important"` | `// Important` |
| `jsonc:"/* block */"` | `/* block */` before the member |

---

## Building JSONC

### Compact constructors (recommended)

`Object(key, val, key, val, ...)` wraps Go values automatically:

```go
doc := jsonc.Object(
	"name",   "Alice",
	"age",    30,
	"active", true,
	"data",   nil,                     // → null
	"tags",   jsonc.Array("a", "b"),
	"meta",   jsonc.Object("key", "val"),
)
```

`Array(elem, elem, ...)` does the same for arrays:

```go
arr := jsonc.Array("hello", 42, true, nil, 3.14, jsonc.Object("x", 1))
```

Supported value types: `string`, `int`/`int64`/`float64`, `bool`, `nil`,
`*Node` (for pre-built or nested structures), `[]any`, `map[string]any`.

### Mutation API (fluent)

```go
doc := jsonc.Object("a", 1)
doc.
	Set("b", 2).              // add member
	Set("a", 10).             // update existing
	Set("c", 3, "comment").   // with trailing comment
	Delete("b")               // remove member

v := doc.Get("a")             // value node for "a"
fmt.Println(doc.Has("c"))     // true
fmt.Println(doc.Keys())       // [a c]
fmt.Println(doc.Len())        // 2
```

### Iteration

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

### Extended API (verbose control)

When you need fine control over CST node placement:

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

---

## Parsing

```go
doc, err := jsonc.Parse(src)
```

Parses a JSONC string into a CST [`*Node`](https://pkg.go.dev/github.com/fan92rus/go-jsonc#Node).
The root is always `KindDocument`. Access the root via `doc.Root()`.

Valid JSON and JSONC (with `//` and `/* */` comments) are both accepted.
Invalid input produces `KindError` nodes in the tree rather than panicking.

---

## Serialization

```go
text := jsonc.Serialize(doc)
```

Serializes a CST back into source text. **Lossless round-trip** — identical
to the original input for valid JSON/JSONC.

---

## Formatting (pretty-print)

```go
formatted := jsonc.Format(doc, &jsonc.FormatOptions{Indent: "  "})
```

| `Indent` | Style |
|----------|-------|
| `"  "` (or `nil`) | Two-space indent (default) |
| `"\t"` | Tab indent |
| `"    "` | Four-space indent |
| `""` | Compact / unindented |

Comments are preserved and properly positioned. **Idempotent** — re-formatting
produces identical output.

---

## Lower-level API

### Scalar nodes

```go
s  := jsonc.NewString("hello")   // → "hello"
n  := jsonc.NewNumber("42")      // → 42
b  := jsonc.NewBoolean(true)     // → true
nu := jsonc.NewNull()            // → null
lc := jsonc.NewCommentLine("hi") // → // hi
bc := jsonc.NewCommentBlock("x") // → /* x */
```

### Tree traversal

```go
// Walk depth-first (stop early by returning false)
doc.Walk(func(n *jsonc.Node) bool {
	fmt.Println(n.Kind, n.Value)
	return true
})

// Find by kind
strs := doc.FindAll(jsonc.KindString)
cmts := doc.FindAll(jsonc.KindComment)

// Compare subtrees
jsonc.DeepEqual(a, b) // true if same structure ignoring positions
```

### Container children

```go
for _, m := range obj.Members() { /* ... */ }
for _, e := range arr.Elements() { /* ... */ }

// Comment body (without delimiters)
fmt.Println(c.Body()) // "text without // or /* */"
```

---

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

---

## Property-based testing

The test suite uses [rapid](https://pgregory.net/rapid) for property-based testing
with generators that produce random valid JSONC documents. Properties tested:

- All valid JSON/JSONC parses without errors
- Comment preservation (line, block, mixed, every position)
- Parse → serialize identity and idempotence
- Format preserves semantics across all indent styles
- Format idempotence (re-formatting produces identical output)
- Path navigation get/set/delete with PBT
- Array index access with PBT round-trip
- Struct serialization round-trip
- Deep nesting (500+ levels)
- Error recovery (truncated input, random bytes)
- Trailing commas, Unicode, escape sequences, number variations

**167+ passing tests, 0 lint issues** (golangci-lint max-strict config, 22+ linters).

---

## Project status

**Stable v0.x** — core API (parse, serialize, format, navigation, path access,
struct binding, file I/O) is fully implemented and well-tested. Backward
compatibility is guaranteed within the v0.x series.

---

## Contributing

1. Fork the repo
2. Create a feature branch
3. Run `go test ./...` and `golangci-lint run ./...`
4. Submit a PR

---

## License

MIT — see [LICENSE](LICENSE).
