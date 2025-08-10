# Pointer Implementation Plan for Zong

Implement pointer syntax and semantics in Zong with suffix '*' operator (unlike Go's prefix '*'). Pointers are represented as i64 in WebAssembly.

## Overview

Zong pointer syntax differs from Go by using **suffix** operators instead of prefix:
- **Pointer types**: `I64*` (pointer to I64) instead of Go's `*I64`  
- **Dereference**: `ptr*` (get value at ptr) instead of Go's `*ptr`
- **Address-of**: `expr&` (get pointer to expr) instead of Go's `ptr&`

In WebAssembly, pointers are implemented as i64 values representing memory addresses.

## Lexer Changes

**No lexer changes required** - existing `ASTERISK` and `BIT_AND` tokens cover pointer operations.

## Parser Changes

### 1. Type System Extensions

Currently only `I64` type exists. Add pointer type parsing:

```go
// In ParseStatement() for variable declarations
// Current: var x: I64;
// New:     var ptr: I64*;

func parseType() string {
    typeName := CurrLiteral  // "I64"
    NextToken()
    
    // Check for pointer suffix
    if CurrTokenType == ASTERISK {
        NextToken()
        return typeName + "*"  // "I64*"
    }
    
    return typeName  // "I64"
}
```

### 2. Expression Parsing Extensions

#### Address-of Operator (&) and Dereference Operator (*) - Suffix
Add `&` as a unary suffix operator with highest precedence:

Add `*` as a postfix operator at highest precedence level too.

```go
// In parseExpressionWithPrecedence() after parsing left operand
// Check for postfix operators (similar to function calls)
for {
    if CurrTokenType == ASTERISK {
        // Postfix dereference: expr*
        NextToken()
        left = &ASTNode{
            Kind:     NodeUnary,
            Op:       "*",
            Children: []*ASTNode{left},
        }
    } else if CurrTokenType == BIT_AND {
        // Postfix address-of: expr&
        NextToken()
        left = &ASTNode{
            Kind:     NodeUnary,
            Op:       "&",
            Children: []*ASTNode{left},
        }
    } else if CurrTokenType == LBRACKET {
        // Existing array indexing...
    } else if CurrTokenType == LPAREN {
        // Existing function calls...
    } else {
        break
    }
}
```

### 3. AST Representation

**No new NodeKind needed** - use existing `NodeUnary` for both address-of and dereference:

```go
// Address-of: &x
&ASTNode{
    Kind:     NodeUnary,
    Op:       "&",
    Children: []*ASTNode{identNode},
}

// Dereference: ptr*
&ASTNode{
    Kind:     NodeUnary,
    Op:       "*",
    Children: []*ASTNode{ptrNode},
}
```

## WebAssembly Backend Changes

### 1. Local Variable Types
Extend `LocalVarInfo` to support pointer types:

```go
type LocalVarInfo struct {
    Name  string
    Type  string // "I64", "I64*", etc.
    Index uint32
}

func collectLocalVariables(ast *ASTNode) []LocalVarInfo {
    // ... existing logic
    // For "I64*" types, still use i64 in WASM (pointers are addresses)
    wasmType := "I64"  // All pointers become i64 in WASM
}
```

### 2. Code Generation for Pointer Operations

#### Address-of (&)
```go
// In EmitExpression()
case NodeUnary:
    if node.Op == "&" {
        // Generate address of variable
        // For now, since we don't have heap allocation,
        // we can use the local variable index as a fake address
        // This is a placeholder implementation
        panic("Address-of operator not yet implemented - requires memory model")
    }
```

#### Dereference (*)
```go
// In EmitExpression()  
case NodeUnary:
    if node.Op == "*" {
        // Emit pointer value onto stack
        EmitExpression(buf, node.Children[0])
        // In future: add i64.load instruction to dereference
        // For now: no-op since we don't have memory model yet
        panic("Dereference operator not yet implemented - requires memory model")
    }
```

#### Pointer Arithmetic
Since we don't have type checking yet, allow arithmetic on pointer values:

```go
// Existing binary operator handling works for pointer arithmetic
// ptr + 1, ptr - offset, etc. - just use i64.add, i64.sub
```

## Memory Model Considerations

### Current Limitations
- **No heap allocation**: Cannot create meaningful pointer addresses yet
- **No memory operations**: WebAssembly memory operations not implemented
- **Function-scoped variables only**: Due to WebAssembly local variables

### Future Implementation Path
1. **Phase 1 (This Implementation)**: Syntax and parsing only
2. **Phase 2**: Add WebAssembly linear memory support
3. **Phase 3**: Implement heap allocation
4. **Phase 4**: Add stack variable addressing

### Placeholder Implementation Strategy
For initial implementation, pointer operations can:
- **Parse successfully**: Full syntax support
- **Panic at build time**: With helpful "not implemented" messages
- **Enable testing**: Syntax and AST generation can be tested

## Test Cases

### 1. Type Declaration Tests
```go
// parsestmt_test.go
func TestParsePointerVariableDeclaration(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"var ptr: I64*;\x00", "(var ptr: I64*)"},
        {"{ var x: I64; var ptr: I64*; }\x00", "(block (var x: I64) (var ptr: I64*))"},
    }
    // ... test implementation
}
```

### 2. Address-of Expression Tests  
```go
// parseexpr_test.go
func TestParseAddressOfOperator(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"x&\x00", "(unary \"&\" (ident \"x\"))"},
        {"(x + y)&\x00", "(unary \"&\" (binary \"+\" (ident \"x\") (ident \"y\")))"},
    }
    // ... test implementation
}
```

### 3. Dereference Expression Tests
```go
func TestParseDereferenceOperator(t *testing.T) {
    tests := []struct {
        input    string 
        expected string
    }{
        {"ptr*\x00", "(unary \"*\" (ident \"ptr\"))"},
        {"(ptr + 1)*\x00", "(unary \"*\" (binary \"+\" (ident \"ptr\") (integer 1)))"},
        {"ptr* + 1\x00", "(binary \"+\" (unary \"*\" (ident \"ptr\")) (integer 1))"},
    }
    // ... test implementation
}
```

### 4. Precedence Tests
```go
func TestPointerOperatorPrecedence(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"x& + 1\x00", "(binary \"+\" (unary \"&\" (ident \"x\")) (integer 1))"},
        {"1 + x&\x00", "(binary \"+\" (integer 1) (unary \"&\" (ident \"x\")))"},
        {"(x + 1)&\x00", "(unary \"&\" (binary \"+\" (ident \"x\") (integer 1)))"},
        {"ptr* + 1\x00", "(binary \"+\" (unary \"*\" (ident \"ptr\")) (integer 1))"},
        {"1 + ptr*\x00", "(binary \"+\" (integer 1) (unary \"*\" (ident \"ptr\")))"},
        {"(ptr + 1)*\x00", "(unary \"*\" (binary \"+\" (ident \"ptr\") (integer 1)))"},
    }
    // ... test implementation
}
```

### 5. Complex Expression Tests
```go
func TestComplexPointerExpressions(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"x&*\x00", "(unary \"*\" (unary \"&\" (ident \"x\")))"},  // x& then dereference
        {"x*&\x00", "(unary \"&\" (unary \"*\" (ident \"x\")))"},  // x* then take address
    }
    // ... test implementation
}
```

## S-Expression Representation

**No changes needed** - existing unary handling supports pointer operators.

## Implementation Steps

### Phase 1: Parser Extensions
1. **Update type parsing** in `ParseStatement()` for variable declarations
2. **Add address-of parsing** in `parseExpressionWithPrecedence()` 
3. **Add dereference parsing** as postfix operator
4. **Update precedence handling** for `&` operator

### Phase 2: Testing Infrastructure
1. **Write type declaration tests** in `parsestmt_test.go`
2. **Write address-of expression tests** in `parseexpr_test.go`  
3. **Write dereference expression tests** in `parseexpr_test.go`
4. **Write precedence tests** for operator interactions

### Phase 3: WebAssembly Placeholder
1. **Update `collectLocalVariables()`** to handle pointer types
2. Code generation doesn't work due to missing semantics for dereference and address-of, so don't test this yet

## Success Criteria

1. **Syntax parsing works**: All pointer expressions parse to correct AST
2. **Type declarations work**: `var ptr: I64*;` parses successfully
3. **Operator precedence correct**: `1 + x&` vs `(1 + x)&` parse differently
4. **Test coverage complete**: All pointer operations have tests
5. **WebAssembly compilation**: Code generates (even if runtime panics)
6. **S-expression output**: AST converts to readable s-expressions

## Future Extensions

### Memory Model
- Linear memory allocation and management
- Stack variable addressing
- Heap allocation
- Global variable addressing

### Advanced Pointer Features
- Pointer arithmetic with type-aware scaling
- Null pointer checking
- Pointer-to-pointer types (`I64**`)

### WebAssembly Integration
- Memory load/store instructions
- Bounds checking
- Memory growth operations

This implementation provides the foundation for pointer support while acknowledging current memory model limitations. The syntax and parsing infrastructure enables future memory management features.
