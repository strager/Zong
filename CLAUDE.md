# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Zong is an experimental programming language in early development, implemented in Go. It is a statically typed, imperative language inspired by Go, featuring manual memory management, green threads, and named parameters. The language is designed to be self-hosted and targets application development.

The compiler currently supports compilation to WebAssembly (WASM) with local variable support, arithmetic expressions, and basic control structures.

## Common Commands

### Running Tests
```bash
go test
```

### Building
```bash
go build
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

## Code Style

In tests, use the github.com/nalgeon/be package:

    func be.Equal[T any](tb testing.TB, got T, wants ...T)
    func be.Err(tb testing.TB, got error, wants ...any)
    func be.True(tb testing.TB, got bool)

## Architecture

### Lexical Analyzer

- Implemented in main.go
- **Global state approach**: Uses global variables for lexer input state (`input`, `pos`) and current token state (`CurrTokenType`, `CurrLiteral`, `CurrIntValue`)
- **Key functions**:
  - `Init([]byte)`: Initializes lexer with input (must be null-terminated)
  - `NextToken()`: Advances lexer and updates global token state

### Expression Parser

- Implemented in main.go using **precedence climbing** algorithm
- **AST representation**: Uses `ASTNode` struct with `NodeKind` enum (NodeIdent, NodeString, NodeInteger, NodeBinary, NodeCall, NodeVar, NodeBlock, etc.)
- **Key functions**:
  - `ParseExpression()`: Main entry point for parsing expressions
  - `parseExpressionWithPrecedence(minPrec)`: Precedence-climbing recursive parser
  - `precedence(TokenType)`: Returns operator precedence levels (1=assignment, 2=comparison, 3=addition, 4=multiplication, 5=postfix)
  - `ToSExpr(*ASTNode)`: Converts AST to s-expression string for testing/debugging
- **Supported operators**: `=` (assignment), `+`, `-`, `*`, `/`, `%`, `==`, `!=`, `<`, `>`, `<=`, `>=`
- **Precedence levels**: Assignment (lowest) → Comparison → Addition/subtraction → Multiplication/division → Function calls/subscript (highest)
- **Assignment**: Right-associative with lowest precedence
- **Tests**: Comprehensive test suite in `parseexpr_test.go` using s-expression format

### Statement Parser

- Implemented in main.go for parsing statements and control structures
- **Key function**: `ParseStatement()`: Parses various statement types
- **Supported statements**:
  - Variable declarations: `var x I64;`
  - Block statements: `{ ... }`
  - If statements: `if condition { ... }`
  - Loop statements: `loop { ... }`
  - Return statements: `return value;`
  - Break/continue statements: `break;`, `continue;`
  - Expression statements: assignments, function calls
- **Tests**: Test suite in `parsestmt_test.go` with s-expression verification

### WebAssembly Backend

- **Full WASM compilation pipeline**: Compiles AST to executable WebAssembly bytecode
- **Key functions**:
  - `CompileToWASM(ast)`: Main compilation entry point, returns WASM bytes
  - `EmitWASMHeader()`: Generates WASM magic number and version
  - `EmitTypeSection()`: Defines function signatures
  - `EmitImportSection()`: Imports external functions (e.g., `print`)
  - `EmitFunctionSection()`: Declares function indices
  - `EmitExportSection()`: Exports main function
  - `EmitCodeSection()`: Generates function bodies with local variables and bytecode
  - `EmitStatement()`: Statement-level code generation
  - `EmitExpression()`: Expression-level code generation
- **Local variables support**:
  - `LocalVarInfo` struct tracks variable name, type, and WASM index
  - `collectLocalVariables()`: Traverses AST to find all variable declarations
  - Only I64 type supported currently
  - Function-scoped variables (WebAssembly limitation)
  - Generates `local.get`, `local.set` instructions
- **Supported features**:
  - Integer literals and arithmetic expressions
  - Variable declarations (`var x I64;`) and assignments (`x = value;`)
  - Function calls (currently `print()`)
  - All comparison and arithmetic operators
  - Nested expressions and statements
- **WASM instruction set**:
  - `I64_CONST`, `I64_ADD`, `I64_SUB`, `I64_MUL`, `I64_DIV_S`, `I64_REM_S`
  - `I64_EQ`, `I64_NE`, `I64_LT_S`, `I64_GT_S`, `I64_LE_S`, `I64_GE_S`
  - `LOCAL_GET`, `LOCAL_SET`, `LOCAL_TEE`
  - `CALL`, `END`
- **Tests**: Comprehensive test suite in `wasmutil_test.go`, `compiler_test.go`, `locals_test.go`, `locals_integration_test.go`

### WASM Runtime Environment

- **Rust-based helper program**: Located in `wasmruntime/` directory
- **Execution infrastructure**: Used by tests to execute generated WASM and verify outputs
- **Key test functions**:
  - `executeWasm()`: Executes WASM bytes and returns output
  - `executeWasmAndVerify()`: Executes and verifies expected output
  - `compileExpression()`: Helper to compile expressions to WASM
- **Runtime features**:
  - Imports `print` function to output I64 values
  - Automatic runtime building during tests
  - WAT (WebAssembly Text) format conversion for debugging

### Key Design Patterns

- **Null-terminated input**: All input must end with `\x00` byte
- **Global state lexer**: Current token information stored in globals rather than returned
- **Incremental parsing**: Call `NextToken()` repeatedly until `EOF`
- **Statement-first compilation**: Code generation operates on statement level, then expressions
- **Local variable tracking**: Variables collected before code generation for proper WASM indexing

## Development Notes

- The language uses Go's module system (`go 1.23.5`)
- No external dependencies beyond Go standard library for the compiler
- Requires Rust toolchain for WASM runtime (used by tests)
- Optional external tools for WASM debugging: `wasm2wat`, `wasm-objdump`

## Current Language Features

### Supported Syntax
```zong
// Variable declarations
var x I64;
var y I64;

// Assignments
x = 42;
y = x + 10;

// Expressions with all operators
var result I64;
result = (x * 2 + y) / 3;

// Comparisons (return 1 for true, 0 for false)
var isGreater I64;
isGreater = x > y;

// Block statements
{
    var local I64;
    local = 5;
    print(local);
}

// Function calls (currently only print)
print(result);
print(x + y * 2);
```

### Type System
- **I64**: 64-bit signed integers (only supported type currently)
- **Variables**: Must be declared before use with `var name I64;`
- **Assignments**: Use `=` operator (right-associative)
- **Scope**: All variables have function scope (WebAssembly limitation)

### Compilation Flow
1. **Lexical Analysis**: Input → Tokens
2. **Parsing**: Tokens → AST (using precedence climbing)
3. **Local Variable Collection**: AST traversal to find variable declarations
4. **Code Generation**: AST → WASM bytecode
5. **Execution**: WASM bytecode runs in Rust-based runtime
