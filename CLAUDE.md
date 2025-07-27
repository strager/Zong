# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Project Overview

Zong is an experimental programming language implemented in Go that compiles to WebAssembly. It features static typing, structs, functions with named parameters, and manual memory management.

## Common Commands

### Running Tests
```bash
go test
```

### Running Single Tests
```bash
go test -run TestFunctionName
```

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

## Development Notes

- Input must be null-terminated (`\x00`)
- Uses Go 1.23.5, requires Rust for runtime
- Test files: `*_test.go` with comprehensive end-to-end testing
- WASM debugging tools: `wasm2wat`, `wasm-objdump` (optional)
