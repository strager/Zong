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

### Modern CLI Usage

The Zong compiler now provides a modern CLI with subcommands:

```bash
# File-based operations
go run . run examples/prime.zong           # Compile and execute
go run . build examples/prime.zong         # Compile to WASM only  
go run . build -o myapp.wasm prime.zong    # Custom output name

# Inline evaluation
go run . eval 'print(42)'                 # Evaluate expressions
go run . eval 'func main() { print(123); }' # Evaluate full programs

# Development workflow
go run . check examples/prime.zong         # Parse and type-check only

# Verbose output
go run . run -v examples/prime.zong        # Show compilation details
```

### Building and Installing

To create a standalone binary:
```bash
go build -o zong .
./zong run examples/prime.zong
```

## Code Style

In tests, use the github.com/nalgeon/be package:

```go
func be.Equal[T any](tb testing.TB, got T, wants T)
func be.Err(tb testing.TB, got error, wants any)
func be.True(tb testing.TB, got bool)
```

## Architecture Overview

### Core Components

- **Parser**: Precedence climbing parser with `ParseExpression()` and `ParseStatement()`, with split lexer+parser
- **Type System**: `TypeNode` structures with support for I64, U8, Boolean, structs, pointers, and slices
- **Compiler**: Compiles typed, symbolified AST to WebAssembly
- **Runtime**: Rust-based WASM executor in `wasmruntime/` directory

### Memory Management

- **Local Variables**: Stored in WASM locals (I64, pointers)
- **Structs**: Stored on stack frame using `tstack` global
- **Parameters**: Function parameters use copy semantics for structs
- **Address-of**: Variables can be addressed and stored on frame

## Language Features

### Current Syntax
```zong
// Struct definitions
struct Point(x: I64, y: I64);

// Functions with named/positional parameters
func add(_ a: I64, _ b: I64): I64 { return a + b; }
func update(p: Point*) {
    p.x = 0;
}

// Variable declarations with initialization
var x: I64 = 42;
var flag: Boolean = true;

// Control flow
if condition {
    // statements
}

loop {
    // infinite loop
}

// Slices
var numbers: []I64;
numbers = append(numbers, 10);

// Main function
func main() {
    var p: Point = Point(x: 10, y: 20);
    print(add(p.x, p.y));

    greet(p&);
    print(p.x);
    print(p.y);
    
    if p.x > 5 {
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
- **Variable initialization**: `var x: I64 = 42;` syntax
- **Nested struct field access**: Multi-level field access like `obj.field.subfield`
- **Slice operations**: Dynamic array operations with `append()`

### Foreign Function Interface (FFI)
Zong supports calling external functions through WASI (WebAssembly System Interface) using `extern` blocks:

```zong
// WASI function declarations
extern "wasi_snapshot_preview1" {
    func random_get(buf: U8*, buf_len: I32): I32;
    func clock_time_get(id: I32, precision: I64, time: I64*): I32;
    func proc_exit(code: I32);
}

func main() {
    var randomByte: U8 = 0;
    var result: I32 = random_get(randomByte&, 1);
    if result == 0 {
        print(randomByte);
    }
}
```

**FFI Features:**
- **WASI preview1 support**: Access to system functions like file I/O, random, time, and process control
- **Multiple module imports**: Can import from different WASM modules (e.g., "wasi_snapshot_preview1", "env")
- **Type safety**: All extern functions are type-checked with proper parameter and return types
- **Comprehensive WASI prelude**: `lib/wasi.zong` provides complete WASI bindings and helper functions

**Available WASI Functions:**
- File I/O: `fd_write`, `fd_read`, `fd_close`
- Random: `random_get`
- Time: `clock_time_get`
- Environment: `environ_get`, `environ_sizes_get`
- Process: `proc_exit`

See `examples/random.zong` for a complete example using WASI functions.

## Sexy Test Framework

The Sexy test framework provides:

- AST testing using S-expression patterns
- Compile error testing
- Execution testing

### Test Structure

Tests are written in Markdown files in the `test/` directory with `_test.md` extension. Each test follows this format:

    ## Test: test name
    ```zong-expr
    input_code
    ```
    ```ast
    (expected_ast_pattern)
    ```

### Test Types

- **Input Types**:
  - `zong-expr`: Single expression input
  - `zong-program`: Full program input
  - `input`: Data to feed to the program for `execute` tests

- **Assertion Types**:
  - `ast`: AST pattern matching using Sexy syntax
  - `compile-error`: Test errors found during compilation
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
- **Extern blocks**: `(extern "module_name" (func1 func2 ...))`
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
    var x: I64 = 10;
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

**Important**: Test code must explicitly call `print()` to generate output. Expressions are not automatically wrapped in print calls.

## Test Structure

The Zong compiler has a comprehensive test suite organized by compiler phases:

### Unit Test Files

1. **`parsing_test.go`** - Lexing and parsing (`source text → tokens → AST`)
2. **`sema_test.go`** - Semantic analysis: symbol tables and type checking (`AST → symbolified + typed AST`)
3. **`wasm_test.go`** - WASM code generation (`typed AST → executable WASM`)
4. **`sexy_test.go`** - End-to-end tests via Sexy framework (`source → execution`)

### Sexy Framework Test Files

Integration tests are written in Markdown files in `test/*.md` using the Sexy framework.

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

- Uses Go 1.23.5, requires Rust for runtime
- WASM debugging tools: `wasm2wat`, `wasm-objdump` (optional)
