# Strategic Assessment — Iteration 13 (passing_tests 35→45)

## 1. Overfitting Check

**Verdict: NOT overfit.** Every gain maps to real code or test improvements:

| Gain | What changed | Real value |
|------|-------------|------------|
| 35→38 | Format idempotency PBT, cross-function round-trip, deep nesting | Catches actual formatter regressions |
| 38→42 | Formatter rewrite (minify fix, multi-line layout, line-comment \n, atLineStart) | Fixes real formatter bugs found in review |
| 42→44 | Member-comment preservation PBT, `+`-rejection PBT | Tests for bugs found in review; the `+` fix is a parser correctness improvement |
| 44→45 | Leading-zero validation + PBT | Real JSON-compliance fix — `01` should not parse |

None of these are benchmark-hacks. Each addition generalizes to any JSONC input, not just the test's generated patterns. The PBT generators are diverse (15+ character classes, random depth, random trivia types, random comment content).

## 2. Orthogonal Directions — What's Left?

### A. Architectural split (passing_tests neutral)
`parser.go` is 757 lines mixing lexer/parser/serializer/formatter. Splitting into separate files:
- `lexer.go` — token types + lexer (~200 lines)
- `parser.go` — parser only (~250 lines)
- `serialize.go` — Serialize (~30 lines)
- `format.go` — Format + FormatOptions + fmtNode/fmtContainer/fmtMember/fmtComment (~280 lines)

**Pros**: Maintainability, readability, diffs are cleaner
**Cons**: Doesn't change `passing_tests` metric at all
**Priority**: LOW for metric, HIGH for project health

### B. Error propagation (breaking change)
Currently `Parse()` never returns an error — all errors are `KindError` nodes in the tree. A stricter API could surface errors at the Go level. But this would break existing consumers and requires a `v2` module path. Not worth it for this project's stage.

### C. Performance optimization (different metric)
The parser is fast enough for config files (1.7s for full test suite). Benchmarking is not set up. Adding a `bench_µs` metric would enable perf work.

### D. Format string / pretty-print quality (subjective, no metric)
- `fmtMember` still puts `: ` on its own line when comments sit between colon and value (cosmetic)
- Compact mode with block comments produces `[1,/*x*/2]` (no space between comma and comment) — valid but ugly
- No way to configure brace placement (same-line vs next-line)

These are real formatting quality issues but have no associated metric.

## 3. Trade-off Analysis

| Improvement | Metric gain | Code complexity | Risk |
|------------|-------------|-----------------|------|
| Leading-zero validation | +1 | +5 lines | Low — lexer change, no existing test uses `01` |
| `+` rejection | +1 | +1 line deletion | Low — `+` is not valid JSON |
| Member comment PBT | +1 | +35 lines | Low — pure test |
| Architectural split | 0 | ~50 lines moved | Medium — git blame, import paths, no logic change |
| Strict JSON validation | +? | 20+ lines | Medium — might break lenient parsing use case |
| Format quality (style) | 0 | 50+ lines | Medium — subjective, idempotency must be preserved |

## 4. The Big Picture

**If restarting from scratch:**
1. Start with separate files (`lexer.go`, `parser.go`, `format.go`) from day one — prevents the current monolith
2. Add PBT generators in lockstep with parser features (already done, good)
3. Add leading-zero validation from the start — trivial once you know it's missing
4. Don't bother with `FormatOptions{}` zero-value minify — just document it
5. Add `go mod tidy` check to CI from the beginning

**What's NOT worth doing (diminishing returns):**
- Adding more edge-case PBT tests — the generators already cover diverse inputs; one more test won't catch a new class of bug
- Full JSON Schema validation — the parser is intentionally lenient for config file use
- Performance tuning — sub-2s for 91 test functions is already fine
- Error propagation — would require v2, low value for this audience

**What WOULD be worth doing:**
- Splitting `parser.go` — real maintainability win, long-term
- Adding a simple benchmark suite (`go test -bench`) — enables perf-aware development
- Documenting public API in godoc — already good but could add more examples
- Adding a `go.mod` version compatibility badge/check

## Recommendation

**Target: finalize.** The `passing_tests` metric has plateaued at 45/45 with diminishing gains. The most impactful code-quality improvement (architectural split) doesn't move the metric. Further test additions would be diminishing-returns edge cases. Remaining work is project health (docs, split), not optimization.

If continuing: split `parser.go` as a `startPhase` (metric-neutral, code health improvement) then finalize.
