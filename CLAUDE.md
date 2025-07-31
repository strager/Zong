# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Project Overview

Zong is an experimental programming language implemented in Go that compiles to WebAssembly. It features static typing, structs, functions with named parameters, and manual memory management.

## Common Commands

### Building Rust WASM Runtime (if needed)
```bash
cd wasmruntime
cargo build --release
```

### Compiling a program and running it
```bash
go run . 'print(42);'
./wasmruntime/target/release/wasmruntime test.wasm
```

## Code Style

In tests, use the github.com/nalgeon/be package:

```go
func be.Equal[T any](tb testing.TB, got T, wants ...T)
func be.Err(tb testing.TB, got error, wants ...any)
func be.True(tb testing.TB, got bool)
```

## Architecture Overview

### Core Components

- **Lexer**: Global state lexer in main.go with `Init()` and `NextToken()`
- **Parser**: Precedence climbing parser with `ParseExpression()` and `ParseStatement()`
- **Type System**: `TypeNode` structures with support for I64, U8, Boolean, structs, pointers, and slices
- **Compiler**: Compiles typed, symbolified AST to WebAssembly
- **Runtime**: Rust-based WASM executor in `wasmruntime/` directory

### Key Types

- `ASTNode`
- `TypeNode`
- `LocalContext`
- `LocalVarInfo`

### Memory Management

- **Local Variables**: Stored in WASM locals (I64, pointers)
- **Structs**: Stored on stack frame using `tstack` global
- **Parameters**: Function parameters use copy semantics for structs
- **Address-of**: Variables can be addressed and stored on frame

## Language Features

### Current Syntax
```zong
// Struct definitions
struct Point { var x I64; var y I64; }

// Functions with named/positional parameters
func add(_ a: I64, _ b: I64): I64 { return a + b; }
func greet(name: I64, age: I64) { print(name); print(age); }

// Variable declarations with initialization
var x I64 = 42;
var flag Boolean = true;

// Control flow
if (condition) {
    // statements
}

loop {
    // infinite loop
}

// Slices
var numbers []I64;
numbers = append(numbers, 10);

// Main function
func main() {
    var p Point;
    p.x = 10;
    p.y = 20;
    print(add(p.x, p.y));
    greet(name: 42, age: 25);
    
    if (p.x > 5) {
        print(999);
    }
}
```

### Supported Types
- **I64**: 64-bit signed integers
- **U8**: 8-bit unsigned integers
- **Boolean**: `true` or `false`
- **Integer**: Compile-time integer type for type inference
- **Structs**: User-defined composite types
- **Pointers**: Address-of (`&`) and dereference (`*`) operations
- **Slices**: Dynamic arrays with `append()` function support

### Control Flow
- **If statements**: Conditional execution with `if (condition) { }`
- **Loops**: Infinite loops with `loop { }`

### Variable Features
- **Variable initialization**: `var x I64 = 42;` syntax
- **Nested struct field access**: Multi-level field access like `obj.field.subfield`
- **Slice operations**: Dynamic array operations with `append()`

### Key Functions

- **Parsing**: `ParseProgram()`, `ParseStatement()`, `ParseExpression()`
- **Compilation**: `CompileToWASM()`, `BuildLocalContext()`
- **Testing**: `executeWasm()`, `executeWasmAndVerify()`

## Sexy Test Framework

The Sexy test framework provides declarative AST testing for the Zong compiler using S-expression patterns.

### Test Structure

Tests are written in Markdown files in the `test/` directory with `.md` extension. Each test follows this format:

```markdown
## Test: test name
```zong-expr
input_code
```
```ast
(expected_ast_pattern)
```
```

### Test Types

- **Input Types**:
  - `zong-expr`: Single expression input
  - `zong-program`: Full program input

- **Assertion Types**:
  - `ast`: AST pattern matching using Sexy syntax
  - `execute`: Execution testing with expected output comparison

### Running Sexy Tests

```bash
go test -run TestSexyAllTests
```

This automatically discovers all `*_test.md` files in `test/` and runs each test case.

### Sexy Pattern Syntax

Sexy uses S-expression syntax to describe expected AST patterns:

- **Literals**: `42`, `true`, `(string "hello")`
- **Variables**: `(var "name")`
- **Binary operations**: `(binary "+" left_expr right_expr)`
- **Function calls**: `(call func_expr arg1 arg2)` or `(call func_expr "param" value)`
- **Variable declarations**: `(var-decl "name" "type" init_expr)`
- **Function definitions**: `(func "name" params return_type body)`
- **Struct definitions**: `(struct "Name" [(field "x" "I64")])`
- **Control flow**: `(if condition [then_stmts] nil [else_stmts])`
- **Arrays/blocks**: `[stmt1 stmt2 ...]`

### Execution Tests

Execution tests compile and run Zong code, comparing the actual output to expected output:

```markdown
## Test: simple expression execution
```zong-expr
print(2 + 3)
```
```execute
5
```

## Test: full program execution
```zong-program
func main() {
    var x I64 = 10;
    print(x * 2);
}
```
```execute
20
```

## Test: no output expected
```zong-expr
2 + 3
```
```execute

```
```

**Important**: Test code must explicitly call `print()` to generate output. Expressions are not automatically wrapped in print calls.

### Pattern Matching Features

- Exact AST structure matching with detailed error messages
- Path-based error reporting (e.g., "root.left.operand")
- Support for all Zong AST node types
- Automatic test discovery and execution

## Test Structure

The Zong compiler has a comprehensive test suite organized by compiler phases:

### Unit Test Files

1. **`parsing_test.go`** - Lexing and parsing (`source text → tokens → AST`)
2. **`sema_test.go`** - Semantic analysis: symbol tables and type checking (`AST → symbolified + typed AST`)
3. **`wasm_test.go`** - WASM code generation (`typed AST → executable WASM`)
4. **`sexy_test.go`** - End-to-end tests via Sexy framework (`source → execution`)

### Sexy Framework Test Files

Integration tests are written in Markdown files in `test/*.md` using the Sexy framework:
- `test/expressions_test.md`: Expression parsing and evaluation
- `test/functions_test.md`: Function declarations and calls
- `test/structs_test.md`: Struct definitions and field access
- `test/statements_test.md`: Variable declarations and control flow
- `test/binary_expr_test.md`: Binary operations and precedence
- And 20+ other feature-specific test files

### Running Tests

```bash
# Run all tests (unit + integration)
go test

# Run specific compiler phase
go test -run TestLexer          # parsing_test.go lexer tests
go test -run TestSymbolTable    # sema_test.go symbol tests
go test -run TestWASM           # wasm_test.go WASM tests
go test -run TestSexyAllTests   # sexy_test.go Sexy framework
```

## Development Notes

- Input must be null-terminated (`\x00`)
- Uses Go 1.23.5, requires Rust for runtime
- Test coverage: ~100 unit tests + 200+ Sexy integration tests
- WASM debugging tools: `wasm2wat`, `wasm-objdump` (optional)
