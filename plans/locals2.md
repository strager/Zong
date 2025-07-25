# Local Variable Management Unification Plan

## Overview

This plan addresses the architectural disconnect between legacy and function compilation paths in local variable management, specifically to fix struct return functionality and create a coherent WASM local allocation strategy.

## Problem Analysis

### Current Architecture Issues

1. **Dual Compilation Paths**: Legacy (`compileLegacyExpression`) and function (`emitSingleFunction`) use incompatible local management
2. **Inconsistent Frame Pointer Calculation**: Multiple competing formulas produce incorrect indices  
3. **Index Assignment Conflicts**: `collectLocalVariables()` and compilation paths make independent assumptions
4. **Broken Struct Returns**: Frame pointer access fails with "local index 32 out of bounds"

### Root Cause

The compiler has **two separate local variable systems** that don't communicate:
- **collectLocalVariables()**: Assigns indices to variables during AST analysis
- **Compilation paths**: Recalculate local counts and frame pointer indices independently

## Fix 1: Merge Legacy and Function Compilation Local Management

### Phase 1: Create Unified Local Variable Context

#### 1.1 Design Unified LocalContext Structure

```go
type LocalContext struct {
    // Variable registry
    Variables []LocalVarInfo
    
    // Layout configuration
    ParameterCount   uint32
    I32LocalCount    uint32  
    I64LocalCount    uint32
    FramePointerIndex uint32
    FrameSize        uint32
    
    // Compilation mode
    IsLegacyMode     bool
}
```

Also introduce a new VarStorage type: `VarStorageParameterLocal`. This is like
`VarStorageLocal` but is allocated differently.

#### 1.2 Replace collectLocalVariables() with Unified Builder

**Current:**
```go
func collectLocalVariables(node *ASTNode) ([]LocalVarInfo, uint32)
```

**New:**
```go
func BuildLocalContext(ast *ASTNode, params []FunctionParameter, isLegacy bool) *LocalContext {
    ctx := &LocalContext{
        IsLegacyMode: isLegacy,
    }
    
    // Phase 1: Add parameters (function mode only)
    if !isLegacy {
        ctx.addParameters(params)
    }
    
    // Phase 2: Collect body variables
    ctx.collectBodyVariables(ast)
    
    // Phase 3: Calculate frame pointer (if needed)
    ctx.calculateFramePointer()
    
    // Phase 4: Assign final WASM indices
    ctx.assignWASMIndices()
    
    return ctx
}
```

#### 1.3 Unified Index Assignment Algorithm

```go
func (ctx *LocalContext) assignWASMIndices() {
    wasmIndex := uint32(0)
    
    // WASM local layout: parameters first, then additional locals by type
    
    // Step 1: Assign parameter indices
    for i := range ctx.Variables {
        if ctx.Variables[i].Storage == VarStorageParameterLocal {
            ctx.Variables[i].Address = wasmIndex
            wasmIndex++
        }
    }
    
    // Step 2: Assign i32 body locals (including frame pointer)
    framePointerAssigned := false
    for i := range ctx.Variables {
        if ctx.Variables[i].Storage == VarStorageLocal && isWASMI32Type(ctx.Variables[i].Type) {
            ctx.Variables[i].Address = wasmIndex
            wasmIndex++
        }
    }
    
    // Step 3: Assign frame pointer if needed
    if ctx.FrameSize > 0 && !framePointerAssigned {
        ctx.FramePointerIndex = wasmIndex
        wasmIndex++
    }
    
    // Step 4: Assign i64 body locals
    for i := range ctx.Variables {
        if ctx.Variables[i].Storage == VarStorageLocal && isWASMI32Type(ctx.Variables[i].Type) {
            ctx.Variables[i].Address = wasmIndex
            wasmIndex++
        }
    }
}
```

### Phase 2: Update Compilation Paths

#### 2.1 Refactor emitSingleFunction()

**Before:**
```go
func emitSingleFunction(buf *bytes.Buffer, fn *ASTNode) {
    // Complex parameter and local management
    var locals []LocalVarInfo
    // ... parameter setup
    bodyLocals, frameSize := collectLocalVariables(fn.Body)
    // ... index adjustment
    framePointerIndex := ... // Complex calculation
    // ... rest of function
}
```

**After:**
```go
func emitSingleFunction(buf *bytes.Buffer, fn *ASTNode) {
    // Use unified local management
    localCtx := BuildLocalContext(fn.Body, fn.Parameters, false)
    
    // Generate WASM locals declaration
    emitLocalDeclarations(&bodyBuf, localCtx)
    
    // Generate frame setup if needed
    if localCtx.FrameSize > 0 {
        EmitFrameSetup(&bodyBuf, localCtx)
    }
    
    // Generate function body
    EmitStatement(&bodyBuf, fn.Body, localCtx)
    
    // ... rest unchanged
}
```

#### 2.2 Refactor compileLegacyExpression()

**Before:**
```go
func compileLegacyExpression(ast *ASTNode) []byte {
    locals, frameSize := collectLocalVariables(ast)
    // ... local counting and frame pointer calculation
}
```

**After:**
```go
func compileLegacyExpression(ast *ASTNode) []byte {
    // Use same unified system
    localCtx := BuildLocalContext(ast, nil, true)
    
    // Generate WASM with unified approach
    emitLocalDeclarations(&bodyBuf, localCtx)
    if localCtx.FrameSize > 0 {
        EmitFrameSetup(&bodyBuf, localCtx)
    }
    EmitStatement(&bodyBuf, ast, localCtx)
    
    // ... rest unchanged
}
```

#### 2.3 Update EmitExpression Signature

**Before:**
```go
func EmitExpression(buf *bytes.Buffer, node *ASTNode, locals []LocalVarInfo, framePointerIndex uint32)
```

**After:**
```go  
func EmitExpression(buf *bytes.Buffer, node *ASTNode, localCtx *LocalContext)
```

**Impact:**
- Single source of truth for all local variable information
- Eliminates framePointerIndex parameter passing
- Consistent variable lookup across all contexts

## Fix 3: Create Coherent Local Variable Allocation Strategy

### Phase 3: WASM Local Allocation Strategy

#### 3.1 Standardized WASM Local Layout

```
WASM Function Local Layout:
┌─────────────────────────────────────────────────────────────┐
│ Index │ Type │ Category          │ Purpose                   │
├───────┼──────┼───────────────────┼───────────────────────────┤
│ 0..N  │ *    │ Parameters        │ Function parameters       │
│ N+1.. │ i32  │ Body Locals       │ i32 user variables        │
│ ...   │ i32  │ Frame Pointer     │ Stack frame base (if req) │
│ ...   │ i64  │ Body Locals       │ i64 user variables        │
└─────────────────────────────────────────────────────────────┘
```

#### 3.2 Local Declaration Generation

```go
func emitLocalDeclarations(buf *bytes.Buffer, localCtx *LocalContext) {
    // Count locals by type (excluding parameters)
    i32Count := localCtx.countBodyLocalsByType(TypeI32)
    i64Count := localCtx.countBodyLocalsByType(TypeI64)
    
    // Add frame pointer to i32 count if needed
    if localCtx.FrameSize > 0 {
        i32Count++
    }
    
    // Emit local declarations
    groupCount := 0
    if i32Count > 0 { groupCount++ }
    if i64Count > 0 { groupCount++ }
    
    writeLEB128(buf, uint32(groupCount))
    
    if i32Count > 0 {
        writeLEB128(buf, uint32(i32Count))
        writeByte(buf, 0x7F) // i32
    }
    
    if i64Count > 0 {
        writeLEB128(buf, uint32(i64Count))
        writeByte(buf, 0x7E) // i64  
    }
}
```

#### 3.3 Variable Access Standardization

```go
func (ctx *LocalContext) EmitVariableAccess(buf *bytes.Buffer, varName string) {
    local := ctx.FindVariable(varName)
    if local == nil {
        panic("Undefined variable: " + varName)
    }
    
    switch local.Storage {
    case VarStorageLocal, VarStorageParameterLocal:
        // Direct local access
        writeByte(buf, LOCAL_GET)
        writeLEB128(buf, local.WASMIndex)
        
    case VarStorageTStack:
        // Frame-relative access
        writeByte(buf, LOCAL_GET)
        writeLEB128(buf, ctx.FramePointerIndex)
        
        if local.Address > 0 {
            writeByte(buf, I32_CONST)
            writeLEB128Signed(buf, int64(local.Address))
            writeByte(buf, I32_ADD)
        }
    }
}
```

### Phase 4: Struct Return Implementation

#### 4.1 Coherent Struct Return Strategy

```go
func (ctx *LocalContext) EmitStructReturn(buf *bytes.Buffer, structVar string) {
    local := ctx.FindVariable(structVar)
    if local == nil || local.Type.Kind != TypeStruct {
        panic("Invalid struct return variable: " + structVar)
    }
    
    // Emit struct address calculation
    ctx.EmitVariableAccess(buf, structVar)
    // Address is now on stack - return it
    writeByte(buf, RETURN)
}
```

#### 4.2 Frame Pointer Access Validation

```go
func (ctx *LocalContext) validateFramePointerAccess() error {
    if ctx.FrameSize > 0 && ctx.FramePointerIndex >= ctx.getTotalLocalCount() {
        return fmt.Errorf("frame pointer index %d out of bounds (total locals: %d)", 
                         ctx.FramePointerIndex, ctx.getTotalLocalCount())
    }
    return nil
}
```

## Implementation Phases

### Phase 1: Infrastructure (Week 1-2)
1. Create `LocalContext` struct and methods
2. Implement `BuildLocalContext()` function  
3. Create unified index assignment algorithm
4. Add validation and error checking

### Phase 2: Legacy Path Migration (Week 3)
1. Update `compileLegacyExpression()` to use `LocalContext`
2. Update `EmitExpression` signature and calls
3. Test legacy compilation path compatibility
4. Ensure existing tests pass

### Phase 3: Function Path Migration (Week 4) 
1. Update `emitSingleFunction()` to use `LocalContext`
2. Remove duplicate local management code
3. Update all `EmitExpression` calls
4. Test function compilation path

### Phase 4: Struct Return Implementation (Week 5)
1. Implement coherent struct return logic
2. Fix `TestFunctionReturningStruct()`
3. Add comprehensive struct return tests
4. Performance optimization and cleanup

## Testing Strategy

### Unit Tests
- `LocalContext` creation and index assignment

### Regression Tests
- All existing tests must continue to pass

## Success Criteria

1. **Architectural Unification**: Single `LocalContext` system used by both compilation paths
2. **Bug Resolution**: `TestFunctionReturningStruct()` passes consistently
3. **No Regressions**: All existing tests continue to pass
4. **Code Clarity**: Elimination of duplicate local management logic
5. **Performance**: No significant performance degradation

## Conclusion

This plan addresses the root architectural issues causing struct return failures by creating a unified, coherent local variable management system. The phased approach minimizes risk while systematically eliminating the current dual-system architecture that creates index calculation conflicts.
