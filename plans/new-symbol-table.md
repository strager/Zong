# New Symbol Table Implementation Plan

## Overview

This plan outlines the implementation of a hierarchical symbol table for Zong that supports proper scoping, variable shadowing, and forward references using a single-pass compilation approach with unresolved reference tracking.

## Current Problems

The existing flat symbol table (`main.go:2577`) has several critical issues:

1. **No Scoping**: All variables exist in a single flat list, causing conflicts between functions and blocks
2. **No Shadowing**: Inner scopes cannot redefine variables from outer scopes
3. **No Forward References**: Functions and types must be declared before use
4. **Linear Lookup**: Symbol resolution is O(n) and doesn't respect scope hierarchy
5. **Name Conflicts**: Function parameters, local variables, and struct fields all share the same namespace

## Design Goals

- **Hierarchical Scoping**: Support global, function, and block scopes with proper variable shadowing
- **Forward References**: Allow functions and types to be declared after they are referenced
- **Single-Pass Compilation**: Resolve symbols incrementally as declarations are encountered
- **Efficient Lookup**: Fast symbol resolution respecting scope chains
- **Clear Error Reporting**: Comprehensive error messages for undefined symbols

## Core Data Structures

### Scope Management

```go
type SymbolTable struct {
    currentScope    *Scope
    unresolvedRefs  []UnresolvedReference
}

type Scope struct {
    parent    *Scope                    // Parent scope for scope chain
    symbols   map[string]*SymbolInfo    // All symbols (variables, functions, structs) in this scope
}

// A reference to a struct, function, or variable.
// Eventually resolved to either a forward reference or a compilation error.
type UnresolvedReference struct {
    Name     string
    ASTNode  *ASTNode    // Node that needs symbol reference filled in
}
```

## Implementation Approach

### Single-Phase Complete Migration

**Goal**: Implement the complete hierarchical symbol table system in one phase, replacing the existing flat symbol table entirely to avoid maintaining dual code paths.

**Core Strategy**: Direct replacement approach. No testing in between steps because the code base will be broken during the migration.

**Key Methods**:
```go
func (st *SymbolTable) PushScope()
func (st *SymbolTable) PopScope()

func (st *SymbolTable) DeclareVariable(name string, varType *TypeNode) (*SymbolInfo, error)
func (st *SymbolTable) LookupVariable(name string) *SymbolInfo
func (st *SymbolTable) DeclareStruct(structType *TypeNode) error
func (st *SymbolTable) LookupStruct(name string) *TypeNode
func (st *SymbolTable) DeclareFunction(name string, parameters []FunctionParameter, returnType *TypeNode) error
func (st *SymbolTable) LookupFunction(name string) *FunctionInfo

func (st *SymbolTable) resolvePendingReferences(name string, symbol *Symbol)
func (st *SymbolTable) ReportUnresolvedSymbols() []error
```

**Resolution Algorithm**:
1. When encountering a symbol reference:
   - Try immediate resolution in scope chain
   - If found, link `ASTNode.Symbol` to the **same** `SymbolInfo` object from declaration
   - If not found, add to unresolved references list
2. When declaring a symbol:
   - Create **one** `SymbolInfo` object that will be the canonical identity
   - Add to current scope
   - Scan unresolved references for matches
   - Resolve matches by setting `ASTNode.Symbol` to the **same** `SymbolInfo` object
   - Remove resolved references from unresolved list
3. At end of compilation:
   - Report any remaining unresolved references as errors

**Symbol Identity Guarantee**: Every symbol reference (variable usage, function call, etc.) must have its `ASTNode.Symbol` field point to the exact same `SymbolInfo` object that was created during declaration. This ensures pointer equality works for symbol lookups in `LocalContext.FindVariable()`.

### Implementation Tasks

1. **Update SymbolTable struct and methods**: 
   - Update current `SymbolTable` (main.go:2577) with `Scope`-based implementation
   - Update existing methods: `DeclareVariable`, `AssignVariable`, `LookupVariable`, `DeclareStruct`, `LookupStruct`, `DeclareFunction`, `LookupFunction`
   - Maintain identical public interfaces initially to minimize parser changes
   - Call `ReportUnresolvedSymbols` after traversing AST in `BuildSymbolTable`
   - Run all tests after changes

2. **Scope management**:
   - Add scope push/pop in function parsing
   - Add scope push/pop for ALL block statements `{ ... }` (if statements, loops, standalone blocks)
   - Update variable declarations to add to current scope
   - Update identifier references to use new lookup system

3. **Test migration and validation**:
   - Update `locals_test.go` and `symboltable_test.go` to work with hierarchical scoping
   - Update tests to use symbol table APIs instead of directly constructing `SymbolInfo` structures
   - Tests should continue to work with canonical `SymbolInfo` objects returned by the new APIs
   - Add comprehensive tests for scoping behavior, shadowing, and forward references
   - Ensure all existing high-level functionality continues to work

4. **Critical considerations for test migration**:
   - Current tests directly construct `SymbolInfo` and `LocalVarInfo` structures - migrate to use APIs
   - Tests expect flat variable collection - update to respect scope boundaries
   - Ensure variable addressing and storage allocation works correctly with scoped variables

### Test Scenarios to Validate
```zong
// Variable shadowing
func main() {
    var x: I64 = 1;
    {
        var x: I64 = 2;  // Should shadow outer x
        print(x);       // Should print 2
    }
    print(x);           // Should print 1
}

// Forward function references
func main() {
    foo();              // Forward reference
}
func foo() { }          // Declaration resolves reference

// Forward type references
struct A {
    var b: B*;          // Forward reference to B
}
struct B {
    var a: A*;          // Mutual reference
}

// Same parameter names in different functions
func add(x: I64, y: I64): I64 { return x + y; }
func sub(x: I64, y: I64): I64 { return x - y; }  // Should not conflict
```

## Success Criteria

1. **Scoping Works**: Variables in different scopes don't conflict
2. **Shadowing Works**: Inner scopes can redefine outer variables
3. **Forward References Work**: Functions and types can be referenced before declaration
4. **Single Pass**: No multi-pass compilation required
5. **Error Reporting**: Clear messages for undefined symbols
6. **Symbol Identity Preserved**: All references to the same symbol use the exact same `SymbolInfo` object
7. **Address-of Operations Work**: Features that rely on `FindVariable()` pointer equality succeed
8. **All Tests Pass**: Existing functionality preserved

## Migration Strategy

1. **Direct Replacement**: Replace the existing flat symbol table entirely in one implementation
2. **Comprehensive Testing**: Run full test suite after each major component is updated
3. **Preserve Interfaces**: Maintain compatible method signatures where possible to minimize breakage
4. **Documentation**: Update CLAUDE.md with new scoping rules and testing approach
