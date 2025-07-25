# Function

This document describes Zong's functions feature (to be implemented).

## Syntax

Function declaration+definition:

```
// Note: `_` is required for positional parameters, like in Swift.
// Note: Return type cannot be inferred and must be specified in this example.
func add(_ a: I64, _ b: I64): I64 {
    return a+b
}

// Note: x and y are named parameters. Callers must write "x:" and "y:" for arguments (in any order).
// Note: No return type.
func print_point(x: I64, y: I64) {
    print(x)
    print(y)
}
```

Function call:

```
print(add(1, 2))  // prints "3"
print_point(x: 1, y: 2)
print_point(y: 2, x: 1) // same as above
```

## Memory Semantics

A Zong function lowers to a WebAssembly function.

### Parameter passing

I64 parameters are passed as i64 parameters in WASM.

I64* (pointer) parameters are passed as i32 parameters in WASM.

User-declared struct type parameters are passed by pointer to copy.
- In a call:
  1. Space is allocated on the tstack for a copy.
  2. A copy of the argument is written to the new space.
  3. The WASM function is called with a pointer to the copy.
- Inside a function:
  1. The parameter is treated as a pointer to the struct.

### Return values

I64 return values are passed as i64 returns in WASM.

I64* (pointer) return values are passed as i32 returns in WASM.

User-declared struct type returns are allocated by the caller and filled in by the callee.
- In a call:
  1. Space is allocated on the tstack for the result.
  2. The WASM function is called with a hidden parameter, a pointer to the space.
  3. When the function returns, the allocated space is interpreted as the function expression's result.
- Inside a function:
  1. When returning, the return value is copied into the memory specified by the hidden parameter pointer.

## Scoping and Declaration

### Function Visibility
- Functions are **global declarations only** - they cannot be declared inside blocks or nested within other functions.
- Functions must be declared before use (no hoisting).
- Functions are not first-class values and cannot be assigned to variables or passed as parameters.

### Parameter and Variable Scoping
- Function parameters create a new scope for the function body.
- Parameters cannot have the same name as local variables within the function.
- Functions cannot access variables from outer scopes (no closures).
- Each function gets its own stack frame; memory is not freed on function return.

## Parameter Matching Rules

### Named vs Positional Parameters
- **Named parameters** (declared with `name: Type`) can only be called with named arguments: `func(name: value)`
- **Positional parameters** (declared with `_ name: Type`) can only be called with positional arguments: `func(value)`
- **Mixing is allowed**: A function can have both named and positional parameters
  - Positional parameters must be written before named parameters
  - Positional arguments must be written before named arguments
- **Call-site validation**: Named parameters can be provided in any order at call site

### Parameter Name Validation
- Parameter names must be unique within a function signature
- Named parameter calls must exactly match declared parameter names
- Unknown parameter names are compile-time errors
- Duplicate parameter names in calls are compile-time errors

## Return Statement Semantics
- Functions with explicit return types must have return statements on all code paths
- Missing return statements result in **undefined behavior** (not checked by compiler)
- Multiple return statements are allowed within a function
- Early returns are supported and properly handle tstack frame cleanup

## AST and Symbol Table Representation

### AST Nodes
- New `NodeFunc` kind for function declarations
- Function calls use existing `NodeCall` with enhanced parameter matching
- Return statements use existing `NodeReturn`

### Symbol Management
- Functions are stored in a global function symbol table in a slice separate from variables
- Function symbols contain signature information (parameters, return type, WASM index)
- Parameter symbols are stored in function-local symbol tables during compilation

### Type System Integration
- Function signatures are represented as composite types for type checking
- Hidden parameters for struct returns are not exposed in the type system
- Function call type checking validates parameter types and return type compatibility

## WASM Code Generation

### Function Index Management
- Built-in functions (like `print`) use reserved indices starting at 0
- User-defined functions get sequential indices starting after built-ins
- Function indices are assigned during the symbol table building phase

### Multiple Function Support
- `EmitTypeSection()` generates type signatures for all functions
- `EmitFunctionSection()` declares all user-defined functions
- `EmitExportSection()` exports the `main` function only (user functions are internal)
- Each function gets its own `EmitCodeSection()` call

### Parameter and Local Management
- WASM function parameters come first in the local index space
- Function-local variables follow parameters in local index assignment
- Frame pointer (if needed) is allocated as the last local
- Hidden struct return parameters are the first WASM parameter but not visible in Zong source

## Implementation Phases

### Phase 1: Basic Functions
- Positional parameters only with I64 types
- Simple return values (I64 or void)
- No struct parameters or returns

### Phase 2: Enhanced Parameters
- Add named parameter support
- Add I64* (pointer) parameter support
- Implement parameter validation and matching

### Phase 3: Advanced Returns
- Add I64* return support
- Implement struct parameter passing (by copy)
- Implement struct return values with hidden parameters

## Error Handling
- **Compile-time errors**: Undefined functions, signature mismatches, parameter name conflicts, duplicate parameters
- **Type checking**: Parameter and return type validation
- **Undefined behavior**: Missing return statements, stack overflow, memory leaks from unreturned tstack allocations
