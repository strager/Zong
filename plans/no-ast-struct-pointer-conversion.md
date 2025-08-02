# Remove AST-Level Struct-to-Pointer Conversion

## Problem

The current code incorrectly converts struct parameters to pointer types at the AST level during parsing. This was introduced during the struct/function parameter consolidation (after commit ee9f7f) and violates separation of concerns.

**Current problematic behavior:**
- `func f(param: S)` gets stored in AST as `func f(param: S*)`
- The `convertPointers` parameter in `parseParameterList` forces this conversion
- Special case validation allows `S` arguments for `S*` parameters to maintain the illusion

**Why this is wrong:**
- Mixes parsing concerns with code generation implementation details
- AST should represent the source code, not implementation constraints
- Before consolidation (sl commit ee9f7f), this worked correctly without AST-level conversion
  - You can run `sl cat -r ee9f7f main.go` to review the old approach

## Solution Architecture

The correct architecture separates concerns properly:

1. **AST Level**: Store function parameters as their declared types (`S`, not `S*`)
2. **WASM Generation**: Handle struct parameters as pointers during code generation
3. **Function Calls**: Implement copy semantics in WASM emission
4. **Type System**: No special cases needed for struct parameter validation

## Evidence This Worked Before

From `sl diff -r ee9f7f`, the consolidation introduced:
- `parseParameterList` function with `convertPointers` parameter
- Struct-to-pointer conversion logic in parsing
- Special case validation for "struct argument for struct pointer parameter"

Before consolidation, there was NO `convertPointers` logic, and struct parameters worked correctly.

## Implementation Plan

### Phase 1: Remove AST-Level Conversion

1. **Remove convertPointers parameter**:
   ```go
   // Before: 
   func parseParameterList(endToken TokenType, allowPositional bool, convertPointers bool)
   // After:
   func parseParameterList(endToken TokenType, allowPositional bool)
   ```

2. **Remove conversion logic**:
   ```go
   // Remove this code:
   if convertPointers && (paramType.Kind == TypeStruct || paramType.Kind == TypeSlice) {
       finalParamType = &TypeNode{
           Kind:  TypePointer,
           Child: paramType,
       }
   }
   ```

3. **Update call sites**:
   ```go
   // Struct parsing:
   parseParameterList(RPAREN, false) // Remove third argument
   
   // Function parsing:
   parseParameterList(RPAREN, true) // Remove third argument
   ```

4. **Remove special case validation**:
   ```go
   // Remove the "Special case: allow struct argument for struct pointer parameter" logic
   // in validateCallArguments
   ```

### Phase 2: Fix WASM Generation

The key insight: **WASM generation already knows how to handle this correctly!**

1. **`wasmTypeByte` already handles struct parameters**:
   ```go
   if typeNode.Kind == TypeStruct {
       return 0x7F // i32 (struct parameters are passed as pointers)
   }
   ```

2. **Function calls already implement copy semantics**:
   - Code around line 1440-1459 copies struct arguments to temporary locations
   - Passes pointers to the copies

3. **Fix local context handling**:
   
   **Current problematic code** (line ~1097):
   ```go
   if targetLocal.Storage == VarStorageParameterLocal &&
       targetLocal.Symbol.Type.Kind == TypePointer &&  // Wrong: expects pointer
       targetLocal.Symbol.Type.Child.Kind == TypeStruct {
   ```
   
   **Should be**:
   ```go
   if targetLocal.Storage == VarStorageParameterLocal &&
       targetLocal.Symbol.Type.Kind == TypeStruct {  // Direct struct parameter
   ```

4. **Fix struct parameter field access**:
   
   In `EmitExpressionL` for struct parameter field access:
   ```go
   // For struct parameters, emit the parameter value (which is a pointer)
   if baseExpr.Symbol.Storage == VarStorageParameterLocal &&
       baseExpr.Symbol.Type.Kind == TypeStruct {
       // Emit LOCAL_GET to get the pointer value
       writeByte(buf, LOCAL_GET)
       writeLEB128(buf, targetLocal.Address)
   }
   ```

### Phase 3: Fix Symbol Resolution

1. **Update struct parameter type resolution**:
   
   **Current code** (only handles TypePointer):
   ```go
   // For pointer-to-struct parameters, resolve the child struct type
   if resolvedType.Kind == TypePointer && resolvedType.Child.Kind == TypeStruct {
   ```
   
   **Should also handle direct structs**:
   ```go
   // For struct parameters, resolve the struct type
   if resolvedType.Kind == TypeStruct {
       structDef := st.LookupStruct(resolvedType.String)
       if structDef != nil {
           node.Parameters[i].Type = structDef
       }
   }
   ```

### Phase 4: Update Tests

1. **Fix struct parameter parsing test**:
   ```markdown
   ## Test: struct parameter parsing
   ```zong-program
   func test(_ testP: Point): I64 { return 42; }
   ```
   ```ast
   [(func "test" [(param "testP" "Point" positional)] "I64" [(return 42)])]
   ```
   ```
   
   (Change from `"Point*"` back to `"Point"`)

2. **Verify all tests pass**, especially:
   - Function struct parameter copies test
   - Struct field access tests
   - Function call validation tests

## Expected Benefits

1. **Clean Separation of Concerns**:
   - AST represents source code accurately
   - WASM generation handles implementation details

2. **Simplified Type System**:
   - No special case validation needed
   - No artificial pointer conversion at parse time

3. **Maintainability**:
   - Code is easier to understand and modify
   - Implementation details are isolated to code generation

4. **Correctness**:
   - Matches the pre-consolidation architecture that worked correctly
   - Eliminates the confusion between surface syntax and internal representation

## Risk Mitigation

- **Incremental approach**: Make changes in phases and test after each phase
- **Comprehensive testing**: Run full test suite after each change
- **Rollback plan**: If issues arise, the current approach can be restored
- **Reference implementation**: The pre-consolidation code (ee9f7f) serves as a reference

## Success Criteria

1. All tests pass, including the "function struct param copies" test
2. AST for `func f(param: S)` shows `TypeStruct`, not `TypePointer`  
3. Function calls with struct arguments work correctly with copy semantics
4. Struct field access works inside functions (`param.field = value`)
5. No special case validation needed for struct parameters

This plan restores the clean architecture that existed before consolidation while maintaining all functionality.
