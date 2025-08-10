# Compile-Time Integer Type Implementation Plan

## Overview

Currently, integer literals are hardcoded as I64 type during parsing, requiring special-case logic for implicit conversion to smaller types like U8. This design is inelegant and doesn't scale well.

**Goal**: Implement a compile-time-only 'Integer' type that represents unsized integer constants. This type automatically converts to the appropriate sized integer type (I64, U8, etc.) based on context during type checking.

## Current Problem

```zong
var x: U8 = 42;        // 42 is parsed as I64, requires special conversion
append(slice&, 255);  // 255 is parsed as I64, requires special conversion
```

Current approach:
1. Parse `42` as `NodeInteger` with `TypeI64`
2. During assignment/append, detect I64→U8 conversion and validate range
3. Change the node's TypeAST from I64 to U8

This approach requires special cases in:
- Variable assignment type checking
- Function parameter type checking  
- Append operation type checking
- Any future contexts where integers are used

## Proposed Solution

```zong
var x: U8 = 42;        // 42 has type 'Integer', converts to U8
var y: I64 = 42;       // 42 has type 'Integer', converts to I64
append(slice&, 255);  // 255 has type 'Integer', converts to U8 (slice element type)
```

New approach:
1. Parse `42` as `NodeInteger` with `TypeInteger` (compile-time only)
2. During type checking, resolve Integer→concrete type based on context
3. Validate range during resolution
4. No special cases needed - standard type checking handles everything

## Implementation Plan

### Phase 1: Add Integer Type

#### 1.1 Add TypeInteger to Type System
- Add `TypeInteger` to `TypeKind` enum
- Add `TypeInteger` to built-in types
- Update `TypeToString()` to handle TypeInteger
- Update `TypesEqual()` for TypeInteger comparisons

```go
const (
    TypeBuiltin TypeKind = "TypeBuiltin" // I64, U8, Bool
    TypeInteger TypeKind = "TypeInteger" // Compile-time integer constants
    TypePointer TypeKind = "TypePointer" // *T
    TypeStruct  TypeKind = "TypeStruct"  // MyStruct  
    TypeSlice   TypeKind = "TypeSlice"   // T[]
)

var (
    TypeI64     = &TypeNode{Kind: TypeBuiltin, String: "I64"}
    TypeU8      = &TypeNode{Kind: TypeBuiltin, String: "U8"}
    TypeInteger = &TypeNode{Kind: TypeInteger, String: "Integer"}
    TypeBool    = &TypeNode{Kind: TypeBuiltin, String: "Boolean"}
)
```

#### 1.2 Update Parser
- Modify `CheckExpression()` for `NodeInteger` to assign `TypeInteger` instead of `TypeI64`

```go
case NodeInteger:
    expr.TypeAST = TypeInteger  // Changed from TypeI64
    return nil
```

### Phase 2: Implement Type Resolution

#### 2.1 Add Type Resolution Function
Create `ResolveIntegerType()` function that converts Integer to concrete types:

```go
// ResolveIntegerType resolves an Integer type to a concrete type based on context
// Returns error if the integer value doesn't fit in the target type
//
// Precondition: node.Kind == NodeInteger
func ResolveIntegerType(node *ASTNode, targetType *TypeNode) error {
    if node.Kind != NodeInteger || node.TypeAST.Kind != TypeInteger {
        panic("ResolveIntegerType called with non-constant")
    }
    
    if !IsIntegerCompatible(node.Integer) {
        return fmt.Errorf("cannot convert integer %d to %s", node.Integer, TypeToString(targetType))
    }
    node.TypeAST = targetType
}
```

#### 2.2 Add Type Compatibility Function
Create `IsIntegerCompatible()` to check if Integer can convert to a type:

```go
// IsIntegerCompatible checks if an Integer type can be converted to targetType
func IsIntegerCompatible(integerValue int64, targetType *TypeNode) bool {
    switch targetType.Kind {
    case TypeBuiltin:
        switch targetType.String {
        case "I64":
            return true // I64 can hold any value we support
        case "U8":
            return integerValue >= 0 && integerValue <= 255
        case "Boolean":
            return false // No integer→Boolean conversion
        }
    }
    return false
}
```

### Phase 3: Update Type Checking

#### 3.1 Update Assignment Type Checking
Replace current special-case logic with generic Integer resolution:

```go
// In CheckAssignment function:
if !TypesEqual(lhsType, rhsType) {
    // Try to resolve Integer type to match LHS
    if rhsType.Kind == TypeInteger {
        err := ResolveIntegerType(rhs, lhsType)
        if err != nil {
            return err
        }
        // Type resolution succeeded, continue
    } else {
        return fmt.Errorf("error: cannot assign %s to %s",
            TypeToString(rhsType), TypeToString(lhsType))
    }
}
```

#### 3.2 Update Function Call Type Checking
Handle Integer→parameter type resolution:

```go
// In function call validation:
for i, arg := range args {
    expectedType := function.Parameters[i].Type
    actualType := arg.TypeAST
    
    if actualType.Kind == TypeInteger {
        err := ResolveIntegerType(arg, expectedType)
        if err != nil {
            return fmt.Errorf("error: argument %d: %v", i+1, err)
        }
    } else if !TypesEqual(actualType, expectedType) {
        return fmt.Errorf("error: argument %d type mismatch", i+1)
    }
}
```

#### 3.3 Update Append Type Checking
Replace current special-case logic:

```go
// In append() validation:
elementType := slicePtrType.Child.Child
valueType := expr.Children[2].TypeAST

if !TypesEqual(valueType, elementType) {
    if valueType.Kind == TypeInteger {
        err := ResolveIntegerType(expr.Children[2], elementType)
        if err != nil {
            return fmt.Errorf("error: append() %v", err)
        }
    } else {
        return fmt.Errorf("error: append() value type %s does not match slice element type %s",
            TypeToString(valueType), TypeToString(elementType))
    }
}
```

#### 3.4 Update Binary Operation Type Checking
Handle Integer operands in arithmetic:

```go
// In binary operation validation:
leftType := expr.Children[0].TypeAST
rightType := expr.Children[1].TypeAST

// Resolve Integer types based on the other operand
if leftType.Kind == TypeInteger && rightType.Kind != TypeInteger {
    err := ResolveIntegerType(expr.Children[0], rightType)
    if err != nil {
        return err
    }
    leftType = expr.Children[0].TypeAST
    resultType = rightType
} else if rightType.Kind == TypeInteger && leftType.Kind != TypeInteger {
    err := ResolveIntegerType(expr.Children[1], leftType)
    if err != nil {
        return err
    }
    rightType = expr.Children[1].TypeAST
    resultType = leftType
} else if leftType.Kind == TypeInteger && rightType.Kind == TypeInteger {
    // Both are Integer - result is Integer
    resultType = TypeInteger
}
```

#### 3.5 Update Index Operation Type Checking
Handle Integer operands in slice indexing too.

### Phase 4: Update WASM Generation

#### 4.1 Ensure No TypeInteger in WASM
Add validation to ensure TypeInteger nodes are resolved before WASM generation:

```go
// In WASM generation functions:
func EmitExpressionR(buf *bytes.Buffer, node *ASTNode, localCtx *LocalContext) {
    if node.TypeAST != nil && node.TypeAST.Kind == TypeInteger {
        panic("Unresolved Integer type in WASM generation: " + ToSExpr(node))
    }
    // ... rest of function
}
```

#### 4.2 Update Type Functions
Ensure TypeInteger is not passed to WASM type functions:

```go
func wasmTypeByte(typeNode *TypeNode) byte {
    if typeNode.Kind == TypeInteger {
        panic("TypeInteger should be resolved before WASM generation")
    }
    // ... rest of function
}
```

### Phase 5: Clean Up Special Cases

#### 5.1 Remove Special Case Logic
Delete all the current I64→U8 conversion logic:
- Remove special cases in `CheckAssignment`
- Remove special cases in append type checking
- Remove special cases in variable initialization

#### 5.2 Update Tests
Modify tests to expect the new behavior:
- Integer literals should initially have TypeInteger
- After type checking, they should have concrete types
- Error messages should reflect the new type system

## Benefits of This Design

1. **Consistency**: No special cases - all type conversions go through the same resolution system
2. **Extensibility**: Easy to add new integer types without touching existing logic
3. **Clarity**: Intent is clear - integer constants adapt to their context
4. **Maintainability**: Centralized type resolution logic
5. **Error Messages**: Better error messages that reflect the actual type system

## Implementation Order

1. **Week 1**: Phase 1 (Add TypeInteger) + Phase 4 (WASM validation)
2. **Week 2**: Phase 2 (Type resolution functions) + basic tests
3. **Week 3**: Phase 3 (Update type checking) + comprehensive tests
4. **Week 4**: Phase 5 (Clean up) + Phase 6 (documentation)

## Risk Mitigation

- Extensive test coverage comparing old vs new behavior
- Validate all existing tests still pass
- Gradual rollout: assignments first, then function calls, then arithmetic

This design provides a much cleaner foundation for the type system and will make future integer type additions trivial.
