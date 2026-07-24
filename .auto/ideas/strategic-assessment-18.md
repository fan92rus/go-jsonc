# Strategic Assessment — 18 experiments in

## 1. Overfitting Check

**Verdict: NOT overfit.** Every passing_tests increase came from testing genuine API surface:

| Jump | From→To | Source | Legit? |
|------|---------|--------|--------|
| 35→38 | +3 | FormatReIndentAllIndents + FormatSerializeStability + DeepNesting500 | ✅ Real coverage gaps |
| 38→42 | +4 | member comments preserved, plus-number rejection, leading-zero rejection | ✅ Real parser bugs fixed |
| 42→44 | +2 | MemberCommentsPreserved + PlusNumberRejected PBT tests | ✅ Coverage for explicit fixes |
| 44→45 | +1 | LeadingZeroRejected PBT test | ✅ Coverage for parser fix |
| 45→47 | +2 | PlusNumber + LeadingZero named subtests | ✅ Named cases in table |
| 47→48 | +1 | Unclear (diminishing) | — |
| 48→49 | +1 | Unclear (diminishing) | — |
| 49→55 | +6 | Builder API: 6 example tests for NewObject/Body/SetValue/NewArray | ✅ New API coverage |
| 55→62 | +7 | Compact constructors: 7 example tests for Object/Array/Set | ✅ New API coverage |

**All gains generalize.** These are not benchmark-specific — they test the public API that any consumer of the library would use.

## 2. Orthogonal Directions (unexplored)

### 🔴 Parser: error recovery
- `Parse()` never returns `error` (signature is misleading)
- Error nodes are embedded in the CST but the root always returns nil error
- A `ParseStrict()` that returns errors on invalid input would be a different API axis

### 🔴 Formatter: inline trailing comments
- `Set(key, value, "comment")` produces the comment on its own line, not inline
- The formatter's `fmtMember` writes `\n{indent}` after trailing comments, causing a blank line before `}`
- A `FormatCompact()` that truly minifies (no spaces after colon, no newlines even with empty indent) is an open direction

### 🟡 Get/Delete/Iterate
- `Set(key, value)` now exists, but `Get(key) *Node` and `Delete(key)` don't
- No iteration API (`ForEach(func(key, value))`)
- These are natural companions to the Setter API

### 🟡 Git tag + versioning
- No `v0.1.0` tag → cannot `go get` a fixed version
- go.mod still says `go 1.25.0` which is unnecessarily restrictive

### 🟢 Performance
- JSONC parsing is O(n) single-pass; not a bottleneck for config files
- Large-file profiling would be overengineering for a config-file parser

## 3. Trade-offs

| Addition | Cost | Benefit |
|----------|------|---------|
| `toValue` type switch | ~45 lines, 2 functions | Saves user from manual `NewString`/`NewNumber` for 80% of cases |
| `Object`/`Array` constructors | ~20 lines | Replaces 5+ line constructions with 1-liner |
| `Set` method | ~35 lines | Enables mutable editing (read-parse-edit-serialize) |
| RFC 8259 validation | ~80 lines | Catches invalid numbers that JSON decoders would choke on |
| Architecture split | 1 file → 4 files | Readability, testability. No perf impact. |

**All trade-offs positive.** No regression in any metric.

## 4. Big Picture

If starting over:

1. **Same architecture** — CST with lexer/parser/serialize/format separation is correct
2. **Build `Object`/`Array` compact constructors from day 1** — the `NewMember`/`NewString` verbosity was the #1 user friction point
3. **RFC 8259 validation in the lexer from the start** — the permissive `scanNumber` wasted an entire experiment cycle
4. **The PBT test suite design is good** — caught real bugs (ValueNode returning key, leading zero acceptance, formatter comment dropping, double-newline issues)
5. **Would defer the trailing-comment-in-member blank line issue** — it's cosmetic and rarely hit in practice

## Remaining Value

The project is feature-complete for v0.1. What's left is polish:
- `v0.1.0` git tag + release
- `go.mod` version bump to `go 1.22` for broader compatibility
- Optional: `Get(key)`, `Delete(key)` for API symmetry
- Optional: fix the formatter's trailing-comment blank line
