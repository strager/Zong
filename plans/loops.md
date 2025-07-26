# Loop Implementation Plan for Zong

## Overview

This plan outlines the implementation of loop control flow in Zong, starting with the basic infinite loop construct `loop { body; }` along with `break` and `continue` statements. The implementation follows Rust's `loop` semantics and Go's naked `for` loop behavior.

## Current State Analysis

### What's Already Implemented ✅
- **Tokens**: `LOOP`, `BREAK`, `CONTINUE` tokens are defined and recognized by the lexer
- **AST Nodes**: `NodeLoop`, `NodeBreak`, `NodeContinue` NodeKind constants exist
- **Parser**: Basic parsing for `loop { statements }`, `break;`, and `continue;` is implemented
- **Tests**: Parser tests exist in `parsestmt_test.go`

### What's Missing ❌
- **WASM Emission**: No code generation for loop constructs in `EmitStatement()`
- **Type Checking**: No validation for loop statements in `CheckStatement()`
- **Integration Tests**: No end-to-end WASM execution tests

## Language Design

### Syntax
```zong
// Infinite loop
loop {
    // statements
    if condition {
        break;
    }
    // more statements
    if other_condition {
        continue;
    }
}
```

### Semantics
- **`loop { body }`**: Infinite loop, executes body repeatedly until `break`
- **`break`**: Exits the innermost enclosing loop
- **`continue`**: Jumps to the beginning of the innermost enclosing loop
- **Nested loops**: Each `break`/`continue` targets the innermost loop
- **Type**: Loop statements have no return type (void)

### Error Conditions
- `break` outside of loop → compile error
- `continue` outside of loop → compile error
- Loops without `break` are allowed (infinite loops)

## Technical Implementation

### WASM Control Flow Mapping

WebAssembly uses structured control flow with `block`, `loop`, and `br` instructions:

```wasm
;; Zong: loop { body; }
;; WASM:
block                    ;; Outer block for break targets
  loop                   ;; Inner loop for continue targets
    ;; body statements
    br 1                 ;; break (exit outer block)
    br 0                 ;; continue (repeat inner loop)
  end
end
```

**Key WASM Opcodes**:
- `0x02` - `block` (for break targets)
- `0x03` - `loop` (for continue targets)  
- `0x0C` - `br` (unconditional branch)
- `0x0B` - `end`
- `0x40` - void block type

### Loop Context Management

Code generation does not need special state to track whether we're in a loop or not. `NodeContinue` and `NodeBreak` will always be valid.

## Implementation Steps

### Phase 1: WASM Code Generation

#### Step 1.1: Add Loop Context to LocalContext
```go
// Add to LocalContext struct in main.go
type LocalContext struct {
    // ... existing fields
    InLoop bool // Track if we're inside a loop for break/continue validation
}
```

#### Step 1.2: Implement EmitStatement Cases
Add to `EmitStatement()` function:

```go
case NodeLoop:
    // Save previous loop state and mark that we're in a loop
    prevInLoop := localCtx.InLoop
    localCtx.InLoop = true
    
    // Emit WASM: block (for break - outer block)
    writeByte(buf, 0x02) // block opcode
    writeByte(buf, 0x40) // void type
    
    // Emit WASM: loop (for continue - inner loop)
    writeByte(buf, 0x03) // loop opcode  
    writeByte(buf, 0x40) // void type
    
    // Emit loop body
    for _, stmt := range node.Children {
        EmitStatement(buf, stmt, localCtx)
    }
    
    // Emit WASM: end (loop)
    writeByte(buf, 0x0B) // end opcode
    
    // Emit WASM: end (block)  
    writeByte(buf, 0x0B) // end opcode
    
    // Restore previous loop state
    localCtx.InLoop = prevInLoop

case NodeBreak:
    if !localCtx.InLoop {
        panic("break statement outside of loop")
    }
    
    // Emit WASM: br 1 (always break to outer block)
    writeByte(buf, 0x0C) // br opcode
    writeLEB128(buf, 1)  // branch depth 1 (outer block)

case NodeContinue:
    if !localCtx.InLoop {
        panic("continue statement outside of loop")
    }
    
    // Emit WASM: br 0 (always continue to inner loop)  
    writeByte(buf, 0x0C) // br opcode
    writeLEB128(buf, 0)  // branch depth 0 (inner loop)
```


### Phase 2: Type Checking

#### Step 2.1: Add CheckStatement Cases
Add to `CheckStatement()` function:

```go
case NodeLoop:
    // Check all statements in loop body
    for _, stmt := range stmt.Children {
        err := CheckStatement(stmt, tc)
        if err != nil {
            return err
        }
    }
    return nil

case NodeBreak:
    if !tc.InLoop() {
        return fmt.Errorf("error: break statement outside of loop")
    }
    return nil
    
case NodeContinue:
    if !tc.InLoop() {
        return fmt.Errorf("error: continue statement outside of loop") 
    }
    return nil
```

#### Step 2.2: Add Loop Context to TypeChecker
```go
// Add to TypeChecker struct
type TypeChecker struct {
    // ... existing fields
    LoopDepth int // Track loop nesting for break/continue validation
}

func (tc *TypeChecker) EnterLoop() {
    tc.LoopDepth++
}

func (tc *TypeChecker) ExitLoop() {
    tc.LoopDepth--
}

func (tc *TypeChecker) InLoop() bool {
    return tc.LoopDepth > 0
}
```

### Phase 3: Testing

#### Step 3.1: Integration Tests
Create comprehensive tests in `loop_test.go`:

```go
func TestBasicLoop(t *testing.T) {
    source := `
        func main() {
            var i I64;
            i = 0;
            loop {
                print(i);
                i = i + 1;
                if i >= 3 {
                    break;
                }
            }
        }
    `
    // Expected output: 0\n1\n2\n
}

func TestNestedLoops(t *testing.T) {
    source := `
        func main() {
            var i I64;
            var j I64;
            i = 0;
            loop {
                j = 0;
                loop {
                    print(j);
                    j = j + 1;
                    if j >= 2 {
                        break;
                    }
                }
                i = i + 1; 
                if i >= 2 {
                    break;
                }
            }
        }
    `
    // Expected output: 0\n1\n0\n1\n
}

func TestContinueStatement(t *testing.T) {
    source := `
        func main() {
            var i I64;
            i = 0;
            loop {
                i = i + 1;
                if i == 2 {
                    continue;
                }
                print(i);
                if i >= 3 {
                    break;
                }
            }
        }
    `
    // Expected output: 1\n3\n (skips 2)
}
```

#### Step 3.2: Error Handling Tests
```go
func TestBreakOutsideLoop(t *testing.T) {
    source := `
        func main() {
            break;
        }
    `
    // Should fail type checking
}

func TestContinueOutsideLoop(t *testing.T) {
    source := `
        func main() {
            continue;
        }
    `
    // Should fail type checking  
}
```

### Phase 4: Integration and Documentation

#### Step 4.1: Update ToSExpr for Debugging
```go
case NodeLoop:
    result := "(loop"
    for _, child := range node.Children {
        result += " " + ToSExpr(child)
    }
    result += ")"
    return result
    
case NodeBreak:
    return "(break)"
    
case NodeContinue:
    return "(continue)"
```

#### Step 4.2: Update CLAUDE.md
Add loop examples to the language features section:
```zong
// Infinite loop with break
loop {
    var input I64;
    // ... get input
    if input == 0 {
        break;
    }
    print(input);
}
```

## Potential Issues and Solutions

### Issue 1: Infinite Loops Without Break
**Problem**: Infinite loops may cause WASM execution to hang.
**Solution**: This is expected behavior (matches Rust/Go).

### Issue 2: Multiple Break/Continue Statements  
**Problem**: Dead code after break/continue may confuse users.
**Solution**: Consider adding dead code detection in future type checking phases.

## Success Criteria

✅ **Functionality**: 
- Basic infinite loops work correctly
- Break exits loops properly  
- Continue restarts loops properly
- Nested loops behave correctly

✅ **Error Handling**:
- Break/continue outside loops caught at compile time
- Clear error messages for invalid usage

✅ **Testing**:
- Comprehensive test coverage (>95%)
- Integration tests with WASM execution
- Edge cases covered (nested loops, early breaks)

This implementation provides a solid foundation for Zong's loop system while maintaining consistency with the existing codebase architecture and following WebAssembly best practices.
