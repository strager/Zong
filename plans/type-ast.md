# Type System AST Design

## Current State

The Zong compiler currently has a primitive type system:
- Types are represented as strings in `LocalVarInfo.Type` (e.g., "I64", "I64*")
- Parser handles pointer syntax by appending "*" to type names in `ParseStatement` (main.go:1561)
- No dedicated type AST representation
- Only I64 and pointer types are supported

## Proposed TypeNode Design

### Core TypeNode Structure

```go
type TypeKind string

const (
    TypeBuiltin TypeKind = "TypeBuiltin"    // I64, Bool
    TypePointer TypeKind = "TypePointer"    // *T
)

type TypeNode struct {
    Kind TypeKind
    
    // For TypeBuiltin
    String string // "I64", "Bool"
    
    // For TypePointer
    Child *TypeNode
}
```

### Built-in Types

```go
// Built-in types
var (
    TypeI64 = &TypeNode{Kind: TypeBuiltin, String: "I64"}
    TypeBool = &TypeNode{Kind: TypeBuiltin, String: "Bool"}
)
```

### Pointer Type Examples

```go
// I64* (pointer to I64)
var TypeI64Ptr = &TypeNode{
    Kind: TypePointer,
    Child: TypeI64,
}

// I64** (pointer to pointer to I64)
var TypeI64PtrPtr = &TypeNode{
    Kind: TypePointer,
    Child: TypeI64Ptr,
}

// Bool* (pointer to Bool)
var TypeBoolPtr = &TypeNode{
    Kind: TypePointer,
    Child: TypeBool,
}
```

### Type Parsing Integration

Update the parser to build TypeNode AST:

```go
// In ParseStatement for var declarations
func parseTypeExpression() *TypeNode {
    if CurrTokenType != IDENT {
        return nil
    }
    
    // Parse base type
    baseTypeName := CurrLiteral
    SkipToken(IDENT)
    
    baseType := getBuiltinType(baseTypeName)
    if baseType == nil {
        // TODO: Handle user-defined types
        return nil
    }
    
    // Handle pointer suffixes
    resultType := baseType
    for CurrTokenType == ASTERISK {
        SkipToken(ASTERISK)
        resultType = &TypeNode{
            Kind: TypePointer,
            Child: resultType,
        }
    }
    
    return resultType
}

func getBuiltinType(name string) *TypeNode {
    switch name {
    case "I64": return TypeI64
    case "Bool": return TypeBool
    default: return nil
    }
}
```

### Local Variable Updates

Replace string-based types with TypeNode:

```go
type LocalVarInfo struct {
    Name    string
    Type    *TypeNode  // Changed from string
    Storage VarStorage
    Address uint32
}
```

### Type Operations

```go
// Type equality checking
func TypesEqual(a, b *TypeNode) bool {
    if a.Kind != b.Kind {
        return false
    }
    
    switch a.Kind {
    case TypeBuiltin:
        return a.String == b.String
    case TypePointer:
        return TypesEqual(a.Child, b.Child)
    }
    return false
}

// Get size in bytes for WASM code generation
func GetTypeSize(t *TypeNode) int {
    switch t.Kind {
    case TypeBuiltin:
        switch t.String {
        case "I64": return 8
        case "Bool": return 1
        default: return 8 // default to 8 bytes
        }
    case TypePointer:
        return 8 // pointers are always 64-bit
    // ... other cases
    }
    return 8
}

// Convert TypeNode to string for display/debugging
func TypeToString(t *TypeNode) string {
    switch t.Kind {
    case TypeBuiltin:
        return t.String
    case TypePointer:
        return TypeToString(t.Child) + "*"
    }
    return ""
}
```

### WASM Backend Integration

Update code generation to use TypeNode:

```go
func emitWasmType(t *TypeNode) byte {
    switch t.Kind {
    case TypeBuiltin:
        switch t.String {
        case "I64", "Bool": return 0x7E // i64
        default: panic("unknown type")
        }
    case TypePointer:
        return 0x7E // pointers are i64 in WASM
    }
    return 0x7E
}
```

## Migration Strategy

1. **Phase 1**: Add TypeNode definitions alongside existing string types
2. **Phase 2**: Update parser to generate TypeNode alongside strings  
3. **Phase 3**: Update LocalVarInfo to include both Type (string) and TypeAST (*TypeNode)
4. **Phase 4**: Update code generation to use TypeNode instead of strings
5. **Phase 5**: Remove string-based Type field and rename TypeAST to Type
