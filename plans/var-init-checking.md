# Variable Initialization Checking Plan

## Overview

This document outlines the design and implementation plan for a control-flow-sensitive semantic analysis pass that ensures all local variables are initialized before use in Zong. This will replace the current simple initialization checking with a more robust, control-flow-aware analysis.

## Current Issues

The existing code has a critical bug in `BuildSymbolTable` (line 3951 in main.go):

```go
// Mark variable as assigned if it has an initializer or if it's a struct/slice
if hasInitializer || resolvedVarType.Kind == TypeStruct || resolvedVarType.Kind == TypeSlice {
    symbol.Assigned = true  // BUG: struct/slice should NOT be auto-initialized
}
```

**Problem**: Struct variables were incorrectly marked as initialized upon declaration, allowing uninitialized access.

**Correct Behavior**: Most variables (I64, Boolean, structs, pointers) require explicit assignment before use. Exception: slices are auto-initialized to empty slices for convenience.

## Requirements

1. **Safety**: Prevent access to uninitialized variables at compile time
2. **Precision**: Control-flow-sensitive analysis to minimize false positives
3. **Error Quality**: Clear error messages distinguishing definite vs. possible uninitialized use
4. **Compatibility**: Integrate with existing semantic analysis pipeline

## Architecture Design

### Key Design Decisions

**Why use `*SymbolInfo` instead of `string` for tracking variables?**
- **Robustness**: Avoids name conflicts between variables in different scopes
- **Precision**: Correctly handles variable shadowing (e.g., inner scope `x` vs outer scope `x`)
- **Integration**: Leverages existing symbol resolution from `BuildSymbolTable`
- **Performance**: Direct pointer comparison is faster than string comparison

**Why no `scopes` field in `InitializationAnalyzer`?**
The `scopes` field is unnecessary because:
1. **Symbol resolution already done**: `BuildSymbolTable` has already resolved which symbol each identifier refers to
2. **AST nodes contain `Symbol` pointers**: Each `NodeIdent` already points to the correct `SymbolInfo`
3. **Scope information embedded**: `SymbolInfo` contains all necessary scope information
4. **Simpler design**: Eliminates duplicate scope management logic

**Maintaining the invariant `definitely ∩ maybe = ∅`:**
- Variables are either definitely initialized, maybe initialized, or uninitialized
- The merge operation ensures no variable appears in both sets
- This makes error reporting clearer and logic simpler

### Core Data Structures

```go
// ControlFlowState tracks variable initialization at a program point
type ControlFlowState struct {
    definitelyInitialized map[*SymbolInfo]bool  // Variables guaranteed initialized
    maybeInitialized      map[*SymbolInfo]bool  // Variables initialized on some paths
}

// InitializationAnalyzer performs the analysis
type InitializationAnalyzer struct {
    errors       *ErrorCollection
    currentState *ControlFlowState
    // NOTE: No scopes field needed - we use existing SymbolInfo from BuildSymbolTable
    // The SymbolInfo already contains scope information and is more robust than strings
}
```

### State Operations

```go
func (s *ControlFlowState) Clone() *ControlFlowState {
    // Deep copy for branching control flow
    clone := &ControlFlowState{
        definitelyInitialized: make(map[*SymbolInfo]bool),
        maybeInitialized:      make(map[*SymbolInfo]bool),
    }
    for sym := range s.definitelyInitialized {
        clone.definitelyInitialized[sym] = true
    }
    for sym := range s.maybeInitialized {
        clone.maybeInitialized[sym] = true
    }
    return clone
}

func (s *ControlFlowState) Merge(other *ControlFlowState) *ControlFlowState {
    // Merge states from multiple paths:
    // - definitely = s.definitely ∩ other.definitely  
    // - maybe = (s.definitely ∪ s.maybe ∪ other.definitely ∪ other.maybe) \ merged.definitely
    //   (maintaining invariant: definitely ∩ maybe = ∅)
    
    merged := &ControlFlowState{
        definitelyInitialized: make(map[*SymbolInfo]bool),
        maybeInitialized:      make(map[*SymbolInfo]bool),
    }
    
    // Variables definitely initialized on ALL paths
    for sym := range s.definitelyInitialized {
        if other.definitelyInitialized[sym] {
            merged.definitelyInitialized[sym] = true
        }
    }
    
    // Variables maybe initialized = all initialized vars minus definitely initialized
    // This maintains the invariant: definitely ∩ maybe = ∅
    allMaybeInitialized := make(map[*SymbolInfo]bool)
    for sym := range s.definitelyInitialized {
        allMaybeInitialized[sym] = true
    }
    for sym := range s.maybeInitialized {
        allMaybeInitialized[sym] = true
    }
    for sym := range other.definitelyInitialized {
        allMaybeInitialized[sym] = true
    }
    for sym := range other.maybeInitialized {
        allMaybeInitialized[sym] = true
    }
    
    for sym := range allMaybeInitialized {
        if !merged.definitelyInitialized[sym] {
            merged.maybeInitialized[sym] = true
        }
    }
    
    return merged
}

func (s *ControlFlowState) MarkInitialized(symbol *SymbolInfo) {
    // Move variable from maybe/uninitialized to definitely initialized
    s.definitelyInitialized[symbol] = true
    delete(s.maybeInitialized, symbol)
}

func (s *ControlFlowState) IsDefinitelyInitialized(symbol *SymbolInfo) bool {
    return s.definitelyInitialized[symbol]
}

func (s *ControlFlowState) IsMaybeInitialized(symbol *SymbolInfo) bool {
    return s.maybeInitialized[symbol]
}
```

## Algorithm Pseudocode

### Main Analysis Function

```
function AnalyzeInitialization(ast *ASTNode) *ErrorCollection:
    analyzer = new InitializationAnalyzer()
    analyzer.currentState = new ControlFlowState()
    
    AnalyzeStatement(ast, analyzer)
    return analyzer.errors
```

### Statement Analysis

```
function AnalyzeStatement(stmt *ASTNode, analyzer *InitializationAnalyzer):
    switch stmt.Kind:
        case NodeVar:
            // Variable declaration - symbol already exists from BuildSymbolTable
            // If has initializer, mark as initialized
            if len(stmt.Children) > 1:
                AnalyzeExpression(stmt.Children[1], analyzer)  // Check RHS first
                analyzer.currentState.MarkInitialized(stmt.Children[0].Symbol)
        
        case NodeBinary when stmt.Op == "=":
            // Assignment statement
            rhs = stmt.Children[1]
            lhs = stmt.Children[0]
            
            AnalyzeExpression(rhs, analyzer)  // Check RHS first
            
            if lhs.Kind == NodeIdent:
                analyzer.currentState.MarkInitialized(lhs.Symbol)
            else:
                AnalyzeExpression(lhs, analyzer)  // Check LHS (e.g., field access)
        
        case NodeBlock:
            // No scope management needed - symbols already contain scope info
            for i, child in stmt.Children:
                AnalyzeStatement(child, analyzer)
                // If this statement doesn't return, continue to next
                if not ReturnsControl(child):
                    continue
                // Otherwise, any subsequent statements are unreachable
                // No need to analyze them
                break
        
        case NodeIf:
            AnalyzeConditional(stmt, analyzer)
        
        case NodeLoop:
            AnalyzeLoop(stmt, analyzer)
        
        case NodeFunc:
            AnalyzeFunctionDeclaration(stmt, analyzer)
        
        case NodeBreak, NodeContinue:
            // These transfer control - no further analysis needed
            return
        
        case NodeReturn:
            // Analyze return expression if present
            if len(stmt.Children) > 0:
                AnalyzeExpression(stmt.Children[0], analyzer)
            // No further analysis after return
            return
        
        default:
            // Expression statement - analyze the expression
            AnalyzeExpression(stmt, analyzer)
```

### Expression Analysis

```
function AnalyzeExpression(expr *ASTNode, analyzer *InitializationAnalyzer):
    switch expr.Kind:
        case NodeIdent:
            // Variable use - check if initialized
            symbol = expr.Symbol  // Symbol already resolved by BuildSymbolTable
            
            if symbol == nil:
                // This should have been caught by BuildSymbolTable, but just in case
                analyzer.AddError("variable '" + expr.String + "' used before declaration")
                return
            
            if not analyzer.currentState.IsDefinitelyInitialized(symbol):
                if analyzer.currentState.IsMaybeInitialized(symbol):
                    analyzer.AddError("variable '" + symbol.Name + "' may be used before assignment") 
                else:
                    analyzer.AddError("variable '" + symbol.Name + "' used before assignment")
                return
        
        case NodeBinary:
            AnalyzeExpression(expr.Children[0], analyzer)
            AnalyzeExpression(expr.Children[1], analyzer)
        
        case NodeUnary:
            AnalyzeExpression(expr.Children[0], analyzer)
        
        case NodeCall:
            // Function call - analyze all arguments
            for i = 1 to len(expr.Children):
                AnalyzeExpression(expr.Children[i], analyzer)
        
        case NodeDot:
            // Field access - check base object is initialized
            AnalyzeExpression(expr.Children[0], analyzer)
        
        case NodeIndex:
            // Array/slice access - check both base and index
            AnalyzeExpression(expr.Children[0], analyzer)
            AnalyzeExpression(expr.Children[1], analyzer)
        
        // Literals don't need initialization checking
        case NodeInteger, NodeBoolean, NodeString:
            return
```

### Conditional Analysis

```
function AnalyzeConditional(ifStmt *ASTNode, analyzer *InitializationAnalyzer):
    // Structure: [condition, then_block, condition2?, else_block2?, ...]
    
    AnalyzeExpression(ifStmt.Children[0], analyzer)  // Check condition
    
    originalState = analyzer.currentState.Clone()
    
    // Analyze then branch  
    AnalyzeStatement(ifStmt.Children[1], analyzer)
    thenState = analyzer.currentState.Clone()
    
    // Reset to original state for else branch
    analyzer.currentState = originalState.Clone()
    
    if len(ifStmt.Children) > 2:
        // Has else/else-if branches
        if ifStmt.Children[2] != nil:
            // else-if condition
            AnalyzeExpression(ifStmt.Children[2], analyzer)
        
        // Process remaining else/else-if branches recursively
        AnalyzeConditional(CreateIfFromTail(ifStmt, 2), analyzer)
        elseState = analyzer.currentState
    else:
        // No else branch - use original state
        elseState = originalState
    
    // Merge states from all branches
    analyzer.currentState = thenState.Merge(elseState)
```

### Loop Analysis

```
function AnalyzeLoop(loop *ASTNode, analyzer *InitializationAnalyzer):
    entryState = analyzer.currentState.Clone()
    
    // Analyze loop body once
    for stmt in loop.Children:
        AnalyzeStatement(stmt, analyzer)
    
    firstIterState = analyzer.currentState.Clone()
    
    // Conservative approach: merge entry state with loop body state
    // This handles cases where break statements prevent initialization
    // Variables are only definitely initialized if they were:
    // 1. Already initialized before the loop, OR
    // 2. Initialized on all paths through the loop (including early exits)
    analyzer.currentState = entryState.Merge(firstIterState)

// Helper function to determine if a statement transfers control
function ReturnsControl(stmt *ASTNode) bool:
    switch stmt.Kind:
        case NodeBreak, NodeContinue, NodeReturn:
            return true
        case NodeIf:
            // Check if all branches return
            return AllBranchesReturn(stmt)
        case NodeBlock:
            // Check if any statement in block returns
            for child in stmt.Children:
                if ReturnsControl(child):
                    return true
            return false
        default:
            return false
```

### Function Analysis

```
function AnalyzeFunctionDeclaration(funcDecl *ASTNode, analyzer *InitializationAnalyzer):
    // Create new analysis context for function
    savedState = analyzer.currentState
    analyzer.currentState = new ControlFlowState()
    
    // Mark function parameters as initialized (they get values from caller)
    for param in funcDecl.Parameters:
        analyzer.currentState.MarkInitialized(param.Symbol)
    
    // Analyze function body
    for stmt in funcDecl.Children:
        AnalyzeStatement(stmt, analyzer)
    
    // Restore previous state (functions don't affect outer scope initialization)
    analyzer.currentState = savedState
```

## Integration Strategy

### Phase 1: Fix Current Bug
1. **Remove incorrect auto-initialization** in `BuildSymbolTable`
2. **Update test expectations** for struct/slice variables
3. **Verify basic initialization checking** still works

### Phase 2: Enhanced Sequential Analysis  
1. **Implement `ControlFlowState` and `InitializationAnalyzer`**
2. **Add sequential statement analysis** (var declarations, assignments, uses)
3. **Integrate with existing `CheckProgram`** pipeline

### Phase 3: Control Flow Analysis
1. **Implement conditional analysis** (if/else statements)
2. **Add loop analysis** with conservative approach
3. **Handle break/continue statements** correctly

### Phase 4: Advanced Features
1. **Function parameter handling**
2. **Address-of operator handling** - require initialization before `&`
3. **Pointer dereference initialization**

### Phase 5: Testing & Polish
1. **Add comprehensive test cases** using Sexy framework
2. **Improve error messages** and locations
3. **Performance optimization** for large codebases

## Integration Points

### Compilation Pipeline
```go
func compileProgram(ast *ASTNode) []byte {
    // 1. Build symbol table
    symbolTable := BuildSymbolTable(ast)
    if symbolTable.Errors.HasErrors() {
        panic("symbol resolution failed: " + symbolTable.Errors.String())
    }
    
    // 2. NEW: Variable initialization analysis
    initErrors := AnalyzeInitialization(ast, symbolTable)
    if initErrors.HasErrors() {
        panic("initialization checking failed: " + initErrors.String())
    }
    
    // 3. Type checking  
    typeErrors := CheckProgram(ast, symbolTable.typeTable)
    if typeErrors.HasErrors() {
        panic("type checking failed: " + typeErrors.String())
    }
    
    // 4. Code generation
    return CompileToWASM(ast, symbolTable)
}
```

### Error Messages

- **Definite uninitialized use**: `"error: variable 'x' used before assignment"`
- **Possible uninitialized use**: `"error: variable 'x' may be used before assignment"`  
- **Use before declaration**: `"error: variable 'x' used before declaration"`

## Test Cases

### Basic Cases
```zong
// Should error - basic uninitialized use
func main() {
    var x: I64;
    print(x);  // Error: variable 'x' used before assignment
}

// Should pass - explicit initialization  
func main() {
    var x: I64;
    x = 42;
    print(x);  // OK
}

// Should error - struct not auto-initialized
func main() {
    var p: Point;
    print(p.x);  // Error: variable 'p' used before assignment  
}
```

### Control Flow Cases
```zong
// Should pass - both paths initialize
func main() {
    var x: I64;
    if condition {
        x = 1;
    } else {
        x = 2;
    }
    print(x);  // OK - definitely initialized
}

// Should error - missing else path
func main() {
    var x: I64;
    if condition {
        x = 1;
    }
    print(x);  // Error: variable 'x' may be used before assignment
}

// Should error - break prevents initialization
func main() {
    var x: I64;
    loop {
        break;
        x = 1;  // Never reached
    }
    print(x);  // Error: variable 'x' used before assignment
}

// Should pass - initialization before break
func main() {
    var x: I64;
    loop {
        x = 1;
        break;
    }
    print(x);  // OK - x initialized before break
}
```

### Function Cases
```zong
// Should pass - parameters are initialized
func test(x: I64): I64 {
    return x;  // OK - parameter is initialized
}

// Should error - local variable uninitialized
func test(): I64 {
    var x: I64;
    return x;  // Error: variable 'x' used before assignment
}

// Should error - address-of uninitialized variable
func main() {
    var x: I64;
    init_var(x&);  // Error: variable 'x' used before assignment
}

// Should pass - address-of initialized variable
func main() {
    var x: I64 = 42;
    init_var(x&);  // OK - x is initialized
}
```

## Success Criteria

1. **Primitive variables require explicit initialization** (I64, Boolean, pointers) ✓
2. **Control-flow sensitivity** reduces false positives ✓
3. **Clear error messages** distinguish definite vs. possible errors ✓
4. **Backward compatibility** with existing valid Zong programs ✓
5. **Performance** - analysis completes quickly even for large programs ✓

Note: Slices and structs are auto-initialized for backward compatibility and convenience.

## Known Limitations

1. **Conservative loop analysis**: The current implementation uses a conservative approach for loops that may report false positives. For example:
   ```zong
   var x: I64;
   loop {
       x = 1;
       break;
   }
   print(x);  // Reports "may be used before assignment" even though x is always initialized
   ```

2. **No inter-procedural analysis**: Function calls that initialize variables through pointers are not tracked.

3. **Slice literal parsing**: Slice type declarations like `var s: []I64;` have parsing issues unrelated to initialization checking.

## Future Enhancements

1. **Definite assignment analysis** for more complex control flow
2. **Inter-procedural analysis** for function calls
3. **Escape analysis integration** for stack vs. heap allocation
4. **Flow-sensitive type analysis** building on initialization state