# Slice Implementation Plan

## Overview

Implement slices as synthesized struct types with subscript operators and `append()` builtin support.

## Design

### Slice Type Syntax
- `Foo[]` creates a slice type for element type `Foo`
  - Works for built-in types like `Boolean` and `I64` too
- Synthesizes: `struct Foo[] { var items Foo*; var length I64; }`
  - Not valid Zong syntax but compiler internal representation

### Type System Changes

#### 1. Add TypeSlice Kind
```go
const (
    TypeBuiltin TypeKind = "TypeBuiltin" // I64, Boolean
    TypePointer TypeKind = "TypePointer" // T*
    TypeStruct  TypeKind = "TypeStruct"  // MyStruct
    TypeSlice   TypeKind = "TypeSlice"   // T[]
)
```

#### 2. Extend TypeNode Structure
```go
type TypeNode struct {
    Kind TypeKind
    String string      // For TypeBuiltin, TypeStruct
    Child *TypeNode    // For TypePointer, TypeSlice (element type)
    Fields []StructField // For TypeStruct
}
```

#### 3. Update parseTypeExpression()
- Detect `[]` suffix after base type
- Create TypeSlice node with Child pointing to element type
- Example: `I64[]` → `&TypeNode{Kind: TypeSlice, Child: TypeI64}`

### Parser Changes

#### 1. Type Parsing Enhancement
```go
func parseTypeExpression() *TypeNode {
    // ... existing base type parsing ...
    
    // Handle slice suffix
    if CurrTokenType == LBRACKET {
        SkipToken(LBRACKET)
        if CurrTokenType == RBRACKET {
            SkipToken(RBRACKET)
            resultType = &TypeNode{
                Kind: TypeSlice,
                Child: resultType,
            }
        }
    }
    
    // ... existing pointer suffix handling ...
}
```

#### 2. Subscript Operator (Already Implemented)
- `NodeIndex` AST node exists at main.go:3486-3496
- Parsing: `left[index]` → `{Kind: NodeIndex, Children: [left, index]}`
- Need compilation support for slice subscripting

### Compilation Changes

#### 1. Type Registry Integration
```go
func synthesizeSliceStruct(elementType *TypeNode) *TypeNode {
    return &TypeNode{
        Kind: TypeStruct,
        String: fmt.Sprintf("%s[]", getTypeName(elementType)),
        Fields: []StructField{
            {Name: "items", Type: &TypeNode{Kind: TypePointer, Child: elementType}, Offset: 0},
            {Name: "length", Type: TypeI64, Offset: 8},
        },
    }
}
```

#### 2. Subscript Compilation
EmitExpressionL:
```go
case NodeIndex:
    // Get slice object and index
    // Compile: slice.items + (index * sizeof(elementType))
    // Put computed address on top of WASM stack
```
EmitExpressionR: Call EmitExpressionL (computes address) then perform the load.
EmitExpression (assignment): Call EmitExpressionL (computes address) then perform the store.

#### 3. Memory Layout
- Slice struct: 16 bytes (8-byte pointer + 8-byte length)
- Stored on tstack like other structs
- Elements stored consecutively at items pointer location

#### 4. Type checking

- Require subscript base to be a slice type
- Require subscript index to be an `I64`
- Require `append()`'s parameters to match

### Builtin Functions

#### 1. append() Function

For each synthesized slice type, an append() function is also synthesized that does the following at runtime:

`append(&slice, value)`:

1. Calculate new size: (length + 1) * elementSize
2. Allocate on tstack
3. Copy existing elements
4. Add new element (value)
5. Update slice.items and slice.length

When generating code for NodeCall, the matching synthesized function is looked up based on the slice type.

#### 2. Memory Management
- Use existing tstack for slice storage
- Elements allocated consecutively
- No automatic deallocation (manual memory management)

### Implementation Steps

1. **Type System**: Add TypeSlice kind and extend TypeNode
2. **Parser**: Update parseTypeExpression() for `[]` syntax
3. **Type Checking**: Ensure slice-related types are correct for subscripting and `append()`
4. **Type Registry**: Add slice struct synthesis
5. **Subscript Compilation**: Implement NodeIndex for slices
6. **append() Builtin**: Add special function handling
7. **Tests**: Comprehensive slice functionality tests

### Testing Strategy

Parser tests for slice types on variables and subscripts. Assert SExpr form.

Integration tests:

```go
func TestSliceBasics(t *testing.T) {
    input := `
    func main() {
        var nums I64[];
        append(&nums, 42);
        append(&nums, 100);
        print(nums[0]);
        print(nums[1]);
        print(nums.length);
    }`
    
    result := executeWasmAndVerify(t, input, "42\n100\n2\n")
}
```

### Edge Cases

1. **Empty slices**: length = 0, items = null pointer
2. **Out of bounds**: No bounds checking (manual memory management)
3. **Type safety**: Element type must match slice type
4. **Memory growth**: Each append reallocates and copies

### Integration Points

- **Lexer**: No changes needed (uses existing LBRACKET/RBRACKET tokens)
- **Symbol table**: Treat slice types as distinct from element types
- **Type checking**: Validate subscript operations and append() calls
- **WASM generation**: Use existing struct compilation patterns
