# Type Checking Implementation Plan

## Overview
Implement a two-pass type checking system for Zong: first build a symbol table, then perform type checking using that table.

## Phase 1: Symbol Table Implementation

### 1.1 Define Data Structures
```go
type SymbolInfo struct {
    Name     string
    Type     TypeNode
}

type SymbolTable struct {
    variables []SymbolInfo
}
```

### 1.2 Symbol Table Operations
- `NewSymbolTable() *SymbolTable`
- `DeclareVariable(name string, type TypeNode) error`
- `AssignVariable(name string)` - panics if the variable doesn't exist
- `LookupVariable(name string) *SymbolInfo` - returns nil if not found

### 1.3 Symbol Table Builder
- `BuildSymbolTable(ast *ASTNode) *SymbolTable`
- Traverse AST and populate symbol table with variable declarations
- Extend existing `collectLocalVariables()` pattern

### 1.4 Testing Symbol Table
- Test variable declaration tracking
- Test assignment tracking
- Test duplicate declaration errors
- Test variable lookup functionality
- Test with existing AST examples from parse tests

## Phase 2: Type Checking Implementation

### 2.1 Type Checker Structure
```go
type TypeChecker struct {
    symbolTable *SymbolTable
    errors      []string
}
```

### 2.2 Core Type Checking Functions
- `CheckProgram(ast *ASTNode, symbolTable *SymbolTable) error`
- `CheckStatement(stmt *ASTNode, tc *TypeChecker) error`
- `CheckExpression(expr *ASTNode, tc *TypeChecker) (TypeNode, error)`
- `CheckAssignment(lhs, rhs *ASTNode, tc *TypeChecker) error`

### 2.3 Statement Type Checking
**CheckStatement responsibilities:**
- Variable declarations: Validate type is supported (I64)
- Block statements: Check all contained statements
- Expression statements: Validate expression is well-formed
- Return statements: Validate return type (future)

### 2.4 Expression Type Checking
**CheckExpression responsibilities:**
- Variable references: Ensure declared and assigned before use
- Integer literals: Return TypeI64
- Binary operations: Ensure operand types match, return result type
- Function calls: Validate arguments and return function return type
- Assignment expressions: Validate and return assigned type

### 2.5 Assignment Validation
**CheckAssignment responsibilities:**
- Validate LHS is assignable (variable, not literal)
- Validate RHS type matches LHS declared type
- Mark variable as assigned in symbol table
- Ensure variable exists and is declared

## Phase 3: Integration with Existing Code

### 3.1 Modify Main Compilation Pipeline
Current flow:
```
Input -> Lexer -> Parser -> AST -> WASM Generator -> WASM Bytes
```

New flow:
```
Input -> Lexer -> Parser -> AST -> Symbol Table Builder -> Type Checker -> WASM Generator -> WASM Bytes
```

### 3.2 Update CompileToWASM Function
- Add symbol table building step
- Add type checking step
- Return early if type errors found
- Pass symbol table to existing WASM generation

### 3.3 Error Reporting
- Collect all type errors before stopping compilation
- Include variable names in error messages (no line numbers; we don't have those)
- Format: `"error: variable 'name' used before assignment"`

## Phase 4: Testing Strategy

### 4.1 Unit Tests
- `symboltable_test.go`: Test symbol table operations in isolation
  - uses the parser to create ASTNodes
- `typechecker_test.go`: Test type checking functions with known ASTs
  - uses the parser to create ASTNodes

### 4.2 Integration Tests
- Extend existing `compiler_test.go` with type error cases
- Test programs that should compile successfully
- Test programs that should fail type checking
- Verify error messages are helpful and accurate

### 4.3 Test Cases to Cover
- Variable used before declaration
- Variable used before assignment
- Type mismatches in arithmetic
- Invalid assignment targets
- Redeclaration of variables
- Valid programs that should pass all checks

## Phase 5: Future Extensions

## Implementation Order
1. Define data structures and basic symbol table operations
2. Implement and test symbol table builder
3. Add tests for symbol table
4. Implement and test basic type checking functions
5. Add tests for type checking
6. Integrate with compilation pipeline
7. Run existing tests
8. Add new compilation tests

## Success Criteria
- All existing valid Zong programs continue to compile
- Invalid programs are caught with clear error messages
- No performance regression in compilation speed
- Comprehensive test coverage for all type checking scenarios
- Foundation ready for future language extensions
