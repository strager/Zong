# String Literal Implementation Plan

## Overview

Implement string literals for Zong with the following specifications:
- **Syntax**: `"hello world"` (no escape characters)
- **Type**: `U8[]` (slice of U8 bytes)
- **Encoding**: ASCII-only
- **Storage**: WASM data section starting at linear address 0
- **Memory Layout**: tstack pointer initialized after data section

## Memory Layout Design

```
WASM Linear Memory:
┌─────────────────────────────────┐ Address 0
│     Data Section                │
│  ┌─────────────────────────────┐│
│  │ "hello"                     ││ Addresses 0-4
│  │ "world"                     ││ Addresses 5-9  
│  │ "foo"                       ││ Addresses 10-12
│  └─────────────────────────────┘│
├─────────────────────────────────┤ Address N (after all strings)
│     tstack region               │ ← tstack pointer starts here
│  ┌─────────────────────────────┐│
│  │ Runtime allocations         ││
│  └─────────────────────────────┘│
└─────────────────────────────────┘
```

## Implementation Phases

### Phase 1: Lexer and Parser Support

#### 1.1 Lexer Changes
- Add `STRING` token type to TokenKind enum
- Implement string tokenization in `NextToken()`
- Store string content (without quotes) in token

```go
const (
    // ... existing tokens
    STRING   TokenKind = "STRING"
)

// In NextToken():
case '"':
    // parse a string literal (no escape sequences supported)
```

#### 1.2 AST Support
- Add `NodeString` to NodeKind enum
- Add string storage to ASTNode struct
- Update `ToSExpr()` for debugging

```go
const (
    // ... existing nodes
    NodeString NodeKind = "NodeString"
)

type ASTNode struct {
    // ... existing fields
    String  string  // For NodeString (and NodeIdent)
}
```

#### 1.3 Parser Integration
- Add string literal parsing to `ParsePrimary()`
- Create NodeString AST nodes

### Phase 2: Type System Integration

#### 2.1 Type Checking
- String literals have type `U8[]` (slice of U8)
- Integrate with existing slice type system
- Update `CheckExpression()` for NodeString

```go
case NodeString:
    // String literal has type U8[]
    expr.TypeAST = &TypeNode{
        Kind: TypeSlice,
        Child: TypeU8,
    }
    return nil
```

#### 2.2 String Operations
- new builtin function `print_bytes(s)` prints string to stdout (no trailing newline) (new runtime feature)
- String indexing: `str[i]` returns U8
- String assignment to U8[] variables
- String parameters in function calls

### Phase 3: Data Section Management

#### 3.1 String Collection
- During compilation, collect all unique string literals
- Assign addresses in data section
- Track string addresses for slice creation

```go
type StringLiteral struct {
    Content string
    Address uint32
    Length  uint32
}

type DataSection struct {
    Strings []StringLiteral
    TotalSize uint32
}
```

#### 3.2 WASM Data Section Emission
- Emit WASM data section with string bytes
- Update existing `EmitDataSection()` or create new function
- Place all string data starting at address 0

```go
func EmitDataSection(buf *bytes.Buffer, dataSection *DataSection) {
    writeByte(buf, 0x0B) // data section ID
    
    // Emit each string as a data segment
    for _, str := range dataSection.Strings {
        // Emit data segment at specific address
        emitDataSegment(buf, str.Address, []byte(str.Content))
    }
}
```

#### 3.3 Memory Initialization
- Calculate total data section size during compilation
- Update `tstack` global initialization in WASM file (not runtime)
- Set `tstack` initial value to `dataSection.TotalSize` 
- No runtime computation needed - value computed at compile time

### Phase 4: Runtime Representation

#### 4.1 String Slice Creation
- Create slice objects pointing to string data
- Length field = string length
- Data pointer = address in data section

```go
// In EmitExpressionR for NodeString:
case NodeString:
    // Create slice structure on tstack
    // 1. Allocate space for slice struct
    // 2. Set length field
    // 3. Set data pointer to string address
    // 4. Return pointer to slice
```

#### 4.2 Integration with Slice Operations
- String indexing uses existing slice indexing
- String assignment uses existing slice assignment
- String parameters use existing slice parameter passing

### Phase 5: WASM Integration

#### 5.1 Global tstack Update
- Modify tstack initialization in import section  
- Set initial value to `dataSection.TotalSize` (computed at compile time)
- No runtime computation needed

```go
func EmitImportSection(buf *bytes.Buffer, dataSize uint32) {
    // ... existing imports
    
    // tstack global with compile-time computed initial value
    writeByte(buf, 0x7F) // i32 type
    writeByte(buf, 0x01) // mutable
    writeByte(buf, I32_CONST)
    writeLEB128(buf, dataSize) // computed by compiler
    writeByte(buf, END)
}
```

#### 5.2 Compilation Pipeline Updates
- Update `CompileToWASM()` to collect strings first
- Pass data section info through compilation pipeline
- Ensure data section is emitted before code section

## Design Decisions and Alternatives

### 1. String Deduplication
Should identical string literals share memory? Answer: Yes! Deduplicate identical strings (saves memory) using a simple algorithm

### 2. String Mutability
Are strings mutable through slice operations? Answer: Yes! For implementation simplicity

### 3. String Length Representation
How to handle string length in slice structure? Answer: Store byte length only

### 4. String Concatenation
How to handle string concatenation? Answer: No built-in concatenation (write it yourself)

### 5. Empty String Handling
How to represent empty strings `""`? Answer: Zero-length slice with null pointer

### 6. Character Encoding Validation
Should we validate ASCII during parsing? Answer: fail on non-ASCII

## Implementation Order

### Week 1: Foundation
1. Add STRING token and NodeString AST support
2. Implement basic string literal parsing
3. Add type checking for string literals
4. Write basic parsing tests

### Week 2: Data Section
1. Implement string collection during compilation
2. Add data section emission to WASM output
3. Update tstack initialization
4. Test data section generation

### Week 3: Runtime Integration
1. Implement string slice creation in code generation
2. Integrate with existing slice operations
3. Add string indexing and assignment tests
4. Test string parameters in function calls

### Week 4: Polish and Testing
1. Add comprehensive test coverage
2. Optimize string handling if needed
3. Document string literal behavior
4. Add example programs using strings

## Testing Strategy

### Unit Tests
- String literal parsing edge cases
- Type checking for string operations
- Data section emission correctness
- String slice creation

### Integration Tests
- String literals in expressions
- String parameters in function calls
- String indexing and length access
- Mixed string and slice operations

### Example Programs
```zong
func main() {
    var greeting: U8[] = "Hello";
    var name: U8[] = "World";
    
    print(greeting[0]); // Should print 72 ('H')
    print(greeting.length); // Should print 5
    print_bytes(greeting); // Should print 'Hello' (no newline)
}
```

## Risk Mitigation

1. **Type Safety**: Validate string operations through type system
2. **WASM Compatibility**: Test data section emission with WASM tools

This implementation provides a solid foundation for string literals while maintaining consistency with Zong's existing slice and memory management systems.
