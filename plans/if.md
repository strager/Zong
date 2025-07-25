# If Statement Implementation Plan

## Overview

Implement if statements with the syntax: `if cond { ... } else { ... }`. The else clause is optional, and else-if chains are supported with separate 'else' and 'if' keywords.

## Boolean Type Integration

### Type System Changes

The Boolean type is already defined in the type system:
- `TypeBoolean = &TypeNode{Kind: TypeBuiltin, String: "Boolean"}` (main.go:1587)
- Boolean type integrated into type checking and WASM lowering

### Comparison Operators Return Boolean

Comparison operators (`==`, `!=`, `<`, `>`, `<=`, `>=`) should return `TypeBoolean` instead of implicit I64. This requires:

1. **Type Checker Updates** (main.go:2240-2300 area)
   - Update `TypeCheckBinary()` to return `TypeBoolean` for comparison operators
   - Ensure comparison operands are type-compatible

2. **WASM Backend** (existing comparison opcodes)
   - `i64.eq`, `i64.ne`, `i64.lt_s`, `i64.gt_s`, `i64.le_s`, `i64.ge_s` already emit i32 (Boolean-compatible)
   - No changes needed for existing comparison WASM generation

## Parsing Changes

### Current If Statement Parsing

The basic if statement parsing already exists (main.go:3358-3376):
```go
case IF:
    SkipToken(IF)
    cond := ParseExpression()
    if CurrTokenType != LBRACE {
        return &ASTNode{} // error
    }
    // ... parse block
    return &ASTNode{
        Kind:     NodeIf,
        Children: children, // [condition, statements...]
    }
```

### Required Parsing Enhancements

1. **Add Else Clause Support**
   - After parsing the if block, check for `ELSE` token
   - Parse else block as either:
     - `else { ... }` (simple else)
     - `else if ...` (else-if chain)

2. **Update AST Structure**
   - Modify `NodeIf` to support else clauses
   - Children array structure:
     - solo 'if': `[condition, then_block]`
     - if-else: `[condition, then_block, nil, else_block]`
     - if-elseif: `[condition, then_block, condition, elseif_block]`
     - if-elseif-else: `[condition, then_block, condition, elseif_block, nil, else_block]`

3. **Enhanced ParseStatement() Case**
```go
case IF:
    SkipToken(IF)
    children := []*ASTNode{}
    children = append(children, ParseExpression()) // if condition
    if CurrTokenType != LBRACE {
        return &ASTNode{} // error
    }
    
    children = append(children, parseBlockStatements()) // then block
    
    for CurrTokenType == ELSE {
        SkipToken(ELSE)
        if CurrTokenType == IF {
            // else-if block
            // ...
            children = append(children, ParseExpression()) // else condition
            children = append(children, parseBlockStatements()) // else block
        } else if CurrTokenType == LBRACE {
            // else block
            children = append(children, nil) // else condition
            children = append(children, parseBlockStatements()) // else block
        } else {
            return &ASTNode{} // error: expected { after else
        }
    }
    
    return &ASTNode{
        Kind:     NodeIf,
        Children: children,
    }
```

## WASM Code Generation

### TypeBoolean

For the new Boolean type, use i32 in WASM. Update isWASMI32Type() to reflect this.

### Control Flow Instructions

Use WASM's structured control flow instructions:

1. **Basic If Statement** (condition + then block)
   ```wasm
   <condition>          ;; Push condition (i32) onto stack
   if (result i32)      ;; If condition != 0
     <then_statements>
   end
   ```

2. **If-Else Statement** (condition + then + else)
   ```wasm
   <condition>          ;; Push condition (i32) onto stack
   if (result i32)      ;; If condition != 0
     <then_statements>
   else
     <else_statements>
   end
   ```

3. **Else-If Chain**
   ```wasm
   <condition1>
   if (result i32)
     <then1_statements>
   else
     <condition2>       ;; Nested if in else block
     if (result i32)
       <then2_statements>
     else
       <else_statements>
     end
   end
   ```

### Code Generation Updates

1. **Extend CompileNode()** (main.go:2920 area)
```go
case NodeIf:
    // Compile condition
    compileNode(node.Children[0], wasm, context)
    
    // Emit if instruction
    wasm.WriteByte(0x04) // if opcode
    wasm.WriteByte(0x40) // block type: void
    
    // Compile then block (Children[1])
    compileBlockStatements(node.Children[1], wasm, context)
    
    // Check for else clause (Children[2])
    if len(node.Children) > 2 {
        wasm.WriteByte(0x05) // else opcode
        // ...
    }
    
    wasm.WriteByte(0x0B) // end opcode
```

2. **Boolean Condition Type Check**
   - Ensure condition expression has `TypeBoolean`
   - WASM expects i32 for condition (0 = false, non-zero = true)
   - Boolean values already represented as i32 in WASM

## Implementation Steps

### Phase 1: Boolean Type Integration
1. Update `TypeCheckBinary()` to return `TypeBoolean` for comparison operators
2. Add tests for Boolean type checking
3. Verify existing WASM comparison opcodes work correctly

### Phase 2: Parsing Enhancements  
1. Extend `ParseStatement()` IF case to handle else clauses
2. Create helper function `parseBlockStatements()` for code reuse
3. Add comprehensive parsing tests for:
   - Basic if statements
   - If-else statements  
   - Else-if chains
   - Nested if statements
4. Use s-expr for testing parsing

### Phase 3: WASM Code Generation
1. Extend `CompileNode()` NodeIf case for else support
2. Implement proper WASM control flow instruction emission
2. Test complex else-if chains and nested conditions
3. Add end-to-end tests (combining parsing, type checking, and execution; see compiler_test.go) verifying WASM execution

## Test Cases

### Basic If Statement
```zong
func main() {
    var x I64;
    x = 42;
    if x == 42 {
        print(1);
    }
}
// should print 1
```

```zong
func main() {
    var x I64;
    x = 420;
    if x == 42 {
        print(1);
    }
}
// should print nothing
```

### If-Else Statement
```zong
func main() {
    var x I64;
    x = 10;
    if x > 20 {
        print(1);
    } else {
        print(0);
    }
}
// should print 0
```

### Else-If Chain
```zong
func main() {
    var score I64;
    score = 85;
    if score >= 90 {
        print(4); // A
    } else if score >= 80 {
        print(3); // B
    } else if score >= 70 {
        print(2); // C
    } else {
        print(1); // F
    }
}
// should print 3
```

### Nested If Statements
```zong
func main() {
    var x I64;
    var y I64;
    x = 5;
    y = 10;
    if x > 0 {
        if y > 0 {
            print(x + y);
        }
    }
}
// should print 15
```

## File Locations

### Primary Changes
- **main.go:3358-3376**: Extend IF parsing case
- **main.go:2920**: Add/extend NodeIf compilation case  
- **main.go:2240-2300**: Update type checking for Boolean returns

### Test Files
- **parsestmt_test.go**: Add if statement parsing tests
- **typechecker_test.go**: Add Boolean type checking tests
- **compiler_test.go**: Add if statement compilation and execution tests

## Dependencies

- Boolean type system (partially implemented)
- Expression parsing (already implemented)
- Block statement parsing (already implemented)
- WASM control flow instructions (need implementation)
- Type checking framework (already implemented)
