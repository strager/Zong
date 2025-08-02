# Revised Plan: Fix Function Call Argument Evaluation Order

## Problem Analysis
The current implementation reorders arguments during type checking via `reorderCallArguments()`, which modifies the AST. This causes arguments to be evaluated in parameter declaration order instead of source code order, breaking expected side effect ordering.

## Key Complexities
1. **Mixed argument types**: Some arguments are simple values (I64, Boolean) while others are structs that require tstack allocation
2. **Stack management**: Struct copies require careful WASM stack manipulation
3. **Parameter matching**: Need to know final positions before evaluation starts

## Solution Strategy

Delete `reorderCallArguments()` and implement parameter matching during code generation:

1. **During Type Checking**:
   - Keep arguments in source order
   - Only validate that arguments match parameters
   - Don't modify the AST

2. **During Code Generation**:
   - Create a mapping: source position → parameter position
   - Allocate storage based on argument types:
     - WASM locals for simple types (I64, Boolean, pointers)
     - tstack space for struct copies
   - Evaluate arguments in source order, storing results
   - Push stored values onto WASM stack in parameter order

## Implementation Details

1. **Remove `reorderCallArguments()` call** from type checking

2. **Add parameter mapping logic** in function call code generation:
   ```go
   // Build mapping from source order to parameter order
   paramMapping := make([]int, len(args))
   for i, paramName := range node.ParameterNames {
       if paramName == "" {
           paramMapping[i] = i // positional arg
       } else {
           // Find parameter index by name
           for j, param := range function.Parameters {
               if param.Name == paramName {
                   paramMapping[i] = j
                   break
               }
           }
       }
   }
   ```

3. **Two-phase evaluation**:
   - Phase 1: Evaluate arguments in source order, storing results
   - Phase 2: Push results in parameter order

4. **Storage allocation**:
   - Before evaluation, determine storage needs for each argument
   - Allocate WASM locals for simple types
   - Reserve tstack space for structs

5. **Handle struct arguments specially**:
   - Still need tstack copies for struct parameters
   - But evaluate the struct expressions in source order

## Benefits
- Arguments evaluated in source order (preserves side effects)
- No AST modification during type checking
- Clean separation between validation and code generation
- Supports mixed positional and named parameters

## Testing
- Verify "function arguments execute in source code order" test passes
- Ensure all existing function tests still work
- Test mixed struct and non-struct arguments
- Test mixed positional and named parameters