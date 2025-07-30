# Test Extraction Scripts

## extract_tests.go

A Go program that parses all `*_test.go` files in the project to extract `CompileToWASM` usage patterns and convert them to Sexy execution test cases.

### Usage

```bash
cd /path/to/zong
go run scripts/extract_tests.go > test/extracted_execution_tests.md
```

### What it extracts

The program identifies these patterns:

1. **Direct execution tests**:
   ```go
   wasmBytes := CompileToWASM(ast)
   result, err := executeWasm(t, wasmBytes)
   be.Equal(t, result, "42\n")
   ```

2. **executeWasmAndVerify calls**:
   ```go
   wasmBytes := CompileToWASM(ast)
   executeWasmAndVerify(t, wasmBytes, "42\n")
   ```

3. **compileExpression wrapper**:
   ```go
   wasmBytes := compileExpression(t, "print(42)")
   executeWasmAndVerify(t, wasmBytes, "42\n")
   ```

### Features

- **AST-based parsing**: Uses `go/ast` for accurate Go code parsing
- **Input type detection**: Determines `zong-expr` vs `zong-program` based on parsing calls
- **String literal extraction**: Properly handles Go string literals and backticks
- **Output cleaning**: Removes Go-specific formatting like `\x00` terminators
- **Deduplication**: Avoids generating duplicate test cases
- **Error filtering**: Skips tests that expect compilation/runtime errors

### Output Format

Generates Sexy test cases in this format:

```markdown
## Test: descriptive name
```zong-program
func main() {
    print(42);
}
```
```execute
42
```
```

### Statistics

From the current Zong test suite, the extractor found **82 unique execution test cases** covering:

- Boolean operations and comparisons
- Arithmetic expressions and operator precedence  
- Function definitions, calls, and parameter passing
- Struct creation, field access, and nested structs
- Control flow (if/else, loops, break/continue)
- Variable declarations and initialization
- Array/slice operations
- Type system features (I64, U8, Boolean, pointers)
- Complex nested expressions and function compositions

These extracted tests provide comprehensive coverage of Zong's execution semantics and can be used to verify that the new Sexy execution test framework works correctly.