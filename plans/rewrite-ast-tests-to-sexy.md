# Plan: Convert Go Tests to Sexy Tests

Based on my analysis, I found **44 uses of `ToSExpr()`** across **8 test files**. I will convert each test that uses `ToSExpr()` for AST assertions into a Sexy test format.

## Files to Create:
1. **parseexpr_test.go** (15 tests) → `test/expressions_test.md`
2. **parsestmt_test.go** (11 tests) → `test/statements_test.md` 
3. **func_test.go** (2 tests) → `test/functions_test.md`
4. **struct_test.go** (5 tests) → `test/structs_test.md`
5. **slice_test.go** (3 tests) → `test/slices_test.md`
6. **boolean_test.go** (1 test) → add to `test/expressions_test.md`
7. **phase3_test.go** (2 tests) → `test/symbolification_test.md`
8. **wasmutil_test.go** (1 test) → add to `test/expressions_test.md`

## Key Transformations:
- Convert Go table-driven tests to individual Sexy tests
- Map `zong-expr` input to `ast` assertions  
- Group related tests logically within files
- Handle both expression and statement tests appropriately
- Preserve test names as section headers with "Test: " prefix

## Example Conversion:
Go test:
```go
{"1 + 2\x00", "(binary \"+\" (integer 1) (integer 2))"}
```

Becomes Sexy test:
```markdown
## Test: basic addition
```zong-expr
1 + 2
```
```ast
(binary "+" (integer 1) (integer 2))
```
```

This will create a comprehensive test suite in the Sexy format covering expressions, statements, functions, structs, slices, and symbol resolution.

## Detailed Breakdown:

### expressions_test.md (19 tests total)
From parseexpr_test.go:
- Literals: integer, string, identifier
- Binary operations: +, -, *, /, %, ==, !=
- Operator precedence and parentheses
- Function calls with positional/named parameters
- Array subscripts
- Unary operators: !, &, *
- Complex combinations

From boolean_test.go:
- Boolean literal parsing (true/false in expressions)

From wasmutil_test.go:
- Additional binary expression test

### statements_test.md (11 tests)
From parsestmt_test.go:
- If statements (with/without else)
- Variable declarations
- Pointer variable declarations
- Loop statements
- Return statements
- Expression statements
- Block statements

### functions_test.md (2 tests)
From func_test.go:
- Function declarations with parameters
- Function declarations with return types

### structs_test.md (5 tests)
From struct_test.go:
- Struct declarations
- Struct variable declarations
- Field access expressions
- Field assignment expressions
- Complex struct expressions

### slices_test.md (3 tests)
From slice_test.go:
- Slice variable declarations
- Slice subscript expressions
- Slice assignment expressions

### symbolification_test.md (2 tests)
From phase3_test.go:
- Tests showing AST before and after symbol resolution
- Will use `ast-sym` assertion type for post-symbolification tests

Total: 42 individual Sexy tests organized across 6 files