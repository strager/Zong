# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Zong is an experimental programming language in early development, implemented in Go. It is a statically typed, imperative language inspired by Go, featuring manual memory management, green threads, and named parameters. The language is designed to be self-hosted and targets application development.

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

## Architecture

### Lexical Analyzer

- Implemented in main.go
- **Global state approach**: Uses global variables for lexer input state (`input`, `pos`) and current token state (`CurrTokenType`, `CurrLiteral`, `CurrIntValue`)
- **Key functions**:
  - `Init([]byte)`: Initializes lexer with input (must be null-terminated)
  - `NextToken()`: Advances lexer and updates global token state

### Expression Parser

- Implemented in main.go using **precedence climbing** algorithm
- **AST representation**: Uses `ASTNode` struct with `NodeKind` enum (NodeIdent, NodeString, NodeInteger, NodeBinary)
- **Key functions**:
  - `ParseExpression()`: Main entry point for parsing expressions
  - `parseExpressionWithPrecedence(minPrec)`: Precedence-climbing recursive parser
  - `precedence(TokenType)`: Returns operator precedence levels (1=comparison, 2=addition, 3=multiplication)
  - `ToSExpr(*ASTNode)`: Converts AST to s-expression string for testing/debugging
- **Supported operators**: `+`, `-`, `*`, `/`, `%`, `==`, `!=`
- **Precedence levels**: Multiplication/division (highest) → Addition/subtraction → Comparison (lowest)
- **Tests**: Comprehensive test suite in `parseexpr_test.go` using s-expression format

### Key Design Patterns

- **Null-terminated input**: All input must end with `\x00` byte
- **Global state lexer**: Current token information stored in globals rather than returned
- **Incremental parsing**: Call `NextToken()` repeatedly until `EOF`

## Development Notes

- The language uses Go's module system (`go 1.23.5`)
- No external dependencies beyond Go standard library
