# Autoresearch: JSONC CST Parser — full Concrete Syntax Tree with CommentNode

## Objective
Implement a complete JSONC (JSON with Comments) CST (Concrete Syntax Tree) parser
that preserves ALL tokens including comments and whitespace. The CST must support:
- Lossless round-trip (parse → serialize → identical output for well-formed input)
- Comment-aware formatting (pretty-printing that preserves comments)
- Query/traverse the tree

The implementation follows TDD with Property-Based Testing (PBT):
the PBT test suite defines the spec; each phase implements features
to make more tests pass.

## Metrics
- **Primary**: `passing_tests` (integer, higher is better) — number of passing PBT test invocations
- **Secondary**: `total_test_functions`, `failed_tests`, `panics`

## How to Run
`.auto/measure.sh` — runs the full PBT test suite and outputs METRIC lines.

## Files in Scope
| File | Purpose |
|------|---------|
| `node.go` | CST node types, positions, helper methods (Walk, DeepEqual, etc.) |
| `parser.go` | JSONC → CST parser + Serialize + Format (stubs now) |
| `gen_test.go` | Rapid generators for JSON/JSONC values |
| `property_test.go` | PBT tests defining ~30 properties |

## Off Limits
- External dependencies beyond `pgregory.net/rapid` (test-only)
- The test files (gen_test.go, property_test.go) — only modify to fix bugs in tests, not to make them pass more easily
- Encoding/json — only for verifying semantic equivalence, not for primary parsing

## Constraints
- All tests must pass for a `keep` on the final phase
- Panics always result in `crash`
- Tests must not be changed to make them easier to pass — the test suite is the spec
- Incremental implementation: each phase builds on the previous
- Use **phases** for multi-step work: `startPhase` → implement → `commitPhase`
- Do NOT overfit to specific test inputs — the tests are randomized

## Phases Plan

### Phase 1: Basic JSON parser (no comments)
Implement tokenizer + parser for standard JSON:
- Strings, numbers, booleans, null
- Objects and arrays
- Position tracking
- Aim: pass all JSON-only PBT tests (TestProperty_ParseValidJSON, TestProperty_ParseStringValues, etc.)

### Phase 2: Comment support
Add line (//) and block (/* */) comment parsing:
- CommentNode in CST with CommentStyle and CommentBody
- Comments in all positions (before values, after values, between members)
- Aim: pass all comment-related PBT tests

### Phase 3: Serialization round-trip
Implement full CST → text serialization:
- Parse → serialize → identical CST (idempotent)
- Position tracking ensures RawText matches input
- Aim: pass all serialization/position PBT tests

### Phase 4: Comment-aware formatting
Implement pretty-printer:
- Configurable indent
- Comment preservation during formatting
- Format is idempotent (format(format(x)) == format(x))
- Aim: pass all formatting PBT tests

### Phase 5: Edge cases and robustness
Polish remaining edge cases:
- Trailing commas, empty containers, deeply nested structures
- Unicode strings, escape sequences
- Error handling for invalid input (must not panic)
- Aim: 100% of property tests pass

## What's Been Tried
(Initial — bare-minimum parser stub exists. Most PBT tests fail.)

## Ideas
- Format can use a visitor pattern to walk the tree and emit text with proper indentation
- For round-trip fidelity, store raw text in each leaf node and reconstruct by concatenation
- Comment nodes between array elements need careful positioning
