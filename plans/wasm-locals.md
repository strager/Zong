# WebAssembly Local Variables ('var') Implementation Plan

## Overview

Extend the Zong WebAssembly backend to support local variables declared with `var` statements, including variable assignments. This implementation focuses only on the `I64` (integer 64-bit) type to keep the scope manageable while establishing the foundation for variable support.

Variables must be assigned before use, but this is not checked by the compiler right now.

## Current State Analysis

### Existing WebAssembly Backend Structure
- **Location**: Implemented in `main.go`
- **Binary encoding utilities**: `writeByte()`, `writeLEB128()`, `writeLEB128Signed()`
- **WASM section emitters**: `EmitWASMHeader()`, `EmitImportSection()`, `EmitTypeSection()`, `EmitFunctionSection()`, `EmitExportSection()`, `EmitCodeSection()`
- **Expression compiler**: `EmitExpression()` with support for integers, binary operations, and function calls
- **Current locals declaration**: `writeLEB128(&bodyBuf, 0) // 0 locals` in `EmitCodeSection()`

### Existing AST Support
- **NodeVar**: Already parsed by statement parser
- **Structure**: `(var (ident "varname") (ident "typename"))`
- **NodeIdent**: Used for variable references in expressions
- **Test coverage**: Basic var declaration tests exist in `parsestmt_test.go`

## WebAssembly Local Variables Mechanics

### Local Variable Instructions
- **local.get `<localidx>`** (0x20): Push local variable value onto stack
- **local.set `<localidx>`** (0x21): Pop stack value and store in local variable
- **local.tee `<localidx>`** (0x22): Set local variable and keep value on stack

### Function Locals Declaration Format
In WebAssembly binary format, locals are declared at the start of each function body:
```
locals ::= count:u32 type:valtype
```

For I64 variables:
- **count**: Number of consecutive locals of this type (LEB128 encoded)
- **type**: 0x7E for I64

## Implementation Plan

### Phase 1: AST Analysis and Local Variable Collection

#### 1.1 Add Local Variable Opcodes
```go
const (
    // ... existing opcodes ...
    LOCAL_GET = 0x20
    LOCAL_SET = 0x21
    LOCAL_TEE = 0x22
)
```

#### 1.2 Implement Local Variable Discovery
```go
// LocalVarInfo represents information about a local variable
type LocalVarInfo struct {
    Name    string
    Type    string  // "I64" only for this implementation
    Index   uint32  // Local variable index in WASM
}

// collectLocalVariables traverses AST to find all var declarations
func collectLocalVariables(node *ASTNode) []LocalVarInfo {
    var locals []LocalVarInfo
    var localIndex uint32 = 0
    
    collectLocalsRecursive(node, &locals, &localIndex)
    return locals
}

func collectLocalsRecursive(node *ASTNode, locals *[]LocalVarInfo, index *uint32) {
    if node == nil {
        return
    }
    
    switch node.Kind {
    case NodeVar:
        // Extract variable name and type
        varName := node.Children[0].String
        varType := node.Children[1].String
        
        // Only support I64 for now
        if varType == "I64" { // Zong 'int' maps to WASM I64
            *locals = append(*locals, LocalVarInfo{
                Name:  varName,
                Type:  "I64",
                Index: *index,
            })
            *index++
        }
    
    case NodeBlock, NodeIf, NodeLoop:
        // Recursively process child statements
        for _, child := range node.Children {
            collectLocalsRecursive(child, locals, index)
        }
    }
}
```

### Phase 2: WebAssembly Code Generation Updates

#### 2.1 Update EmitCodeSection to Support Locals
```go
func EmitCodeSection(buf *bytes.Buffer, ast *ASTNode) {
    writeByte(buf, 0x0A) // code section id
    
    // Collect local variables from AST
    locals := collectLocalVariables(ast)
    
    // Generate function body
    var bodyBuf bytes.Buffer
    
    // Emit locals declarations
    if len(locals) > 0 {
        // Group locals by type (all I64 in this implementation)
        i64Count := 0
        for _, local := range locals {
            if local.Type == "I64" {
                i64Count++
            }
        }
        
        writeLEB128(&bodyBuf, 1) // 1 local type group
        writeLEB128(&bodyBuf, uint32(i64Count)) // count of I64 locals
        writeByte(&bodyBuf, 0x7E) // I64 type
    } else {
        writeLEB128(&bodyBuf, 0) // 0 locals (existing behavior)
    }
    
    // Emit statement bytecode
    EmitStatement(&bodyBuf, ast, locals)
    writeByte(&bodyBuf, 0x0B) // end instruction
    
    // Write section size and content
    var sectionBuf bytes.Buffer
    writeLEB128(&sectionBuf, 1) // 1 function
    writeLEB128(&sectionBuf, uint32(bodyBuf.Len())) // function body size
    writeBytes(&sectionBuf, bodyBuf.Bytes())
    
    writeLEB128(buf, uint32(sectionBuf.Len()))
    writeBytes(buf, sectionBuf.Bytes())
}
```

#### 2.2 Implement Statement-Level Code Generation
```go
// EmitStatement generates WASM bytecode for statements
func EmitStatement(buf *bytes.Buffer, node *ASTNode, locals []LocalVarInfo) {
    if node == nil {
        return
    }
    
    switch node.Kind {
    case NodeVar:
        // Variable declarations don't generate runtime code
        // (locals are declared in function header)
        break
        
    case NodeBlock:
        // Emit all statements in the block
        for _, stmt := range node.Children {
            EmitStatement(buf, stmt, locals)
        }
        
    case NodeCall:
        // Handle expression statements (e.g., print calls)
        EmitExpression(buf, node, locals)
        
    default:
        // For now, treat unknown statements as expressions
        EmitExpression(buf, node, locals)
    }
}
```

#### 2.3 Update EmitExpression for Variable References
```go
func EmitExpression(buf *bytes.Buffer, node *ASTNode, locals []LocalVarInfo) {
    switch node.Kind {
    case NodeInteger:
        writeByte(buf, I64_CONST)
        writeLEB128Signed(buf, node.Integer)
        
    case NodeIdent:
        // Variable reference - emit local.get
        var localIndex uint32
        found := false
        for _, local := range locals {
            if local.Name == node.String {
                localIndex = local.Index
                found = true
                break
            }
        }
        if !found {
            panic("Undefined variable: " + node.String)
        }
        writeByte(buf, LOCAL_GET)
        writeLEB128(buf, localIndex)
        
    case NodeBinary:
        EmitExpression(buf, node.Children[0], locals) // left operand
        EmitExpression(buf, node.Children[1], locals) // right operand
        writeByte(buf, getBinaryOpcode(node.Op))
        
    case NodeCall:
        if len(node.Children) > 0 && node.Children[0].Kind == NodeIdent && node.Children[0].String == "print" {
            if len(node.Children) > 1 {
                EmitExpression(buf, node.Children[1], locals) // argument
            }
            writeByte(buf, CALL)
            writeLEB128(buf, 0) // function index 0 (print import)
        }
    }
}
```

### 2.4 Variable Assignment Support

```go
case NodeBinary:
    // Emit RHS expression
    EmitExpression(buf, node.Children[1], locals) // value
    
    if node.Op == "=" {
        // Get variable name and emit local.set
        varName := node.Children[0].String
        var localIndex uint32
        found := false
        for _, local := range locals {
            if local.Name == varName {
                localIndex = local.Index
                found = true
                break
            }
        }
        if !found {
            panic("Undefined variable: " + varName)
        }
        writeByte(buf, LOCAL_SET)
        writeLEB128(buf, localIndex)
    } else {
        // ...
    }
```

## Test Cases

### Unit Tests for Local Variable Collection

#### Test 1: Single Variable Declaration
```go
func TestCollectSingleLocalVariable(t *testing.T) {
    input := []byte("var x I64;\x00")
    Init(input)
    ast := ParseStatement()
    
    locals := collectLocalVariables(ast)
    
    expected := []LocalVarInfo{
        {Name: "x", Type: "I64", Index: 0},
    }
    
    assert.Equal(t, expected, locals)
}
```

#### Test 2: Multiple Variable Declarations
```go
func TestCollectMultipleLocalVariables(t *testing.T) {
    input := []byte("{ var x I64; var y I64; }\x00")
    Init(input)
    ast := ParseStatement()
    
    locals := collectLocalVariables(ast)
    
    expected := []LocalVarInfo{
        {Name: "x", Type: "I64", Index: 0},
        {Name: "y", Type: "I64", Index: 1},
    }
    
    assert.Equal(t, expected, locals)
}
```

#### Test 3: Nested Block Variable Collection
```go
func TestCollectNestedBlockVariables(t *testing.T) {
    input := []byte("{ var a I64; { var b I64; } }\x00")
    Init(input)
    ast := ParseStatement()
    
    locals := collectLocalVariables(ast)
    
    expected := []LocalVarInfo{
        {Name: "a", Type: "I64", Index: 0},
        {Name: "b", Type: "I64", Index: 1},
    }
    
    assert.Equal(t, expected, locals)
}
```

### Error Handling Tests

#### Test 7: Undefined Variable Reference
```go
func TestUndefinedVariableReference(t *testing.T) {
    input := []byte("print(undefined_var);\x00")
    Init(input)
    ast := ParseStatement()
    
    var buf bytes.Buffer
    locals := []LocalVarInfo{} // No locals defined
    
    defer func() {
        if r := recover(); r != nil {
            assert.Contains(t, r.(string), "Undefined variable: undefined_var")
        } else {
            t.Fatal("Expected panic for undefined variable")
        }
    }()
    
    // Extract undefined_var from print(undefined_var)
    printArg := ast.Children[1] // the undefined_var argument
    EmitExpression(&buf, printArg, locals)
}
```

### End-to-End WASM Tests

#### Test 8: Complete WASM Module with Variables
```go
func TestCompleteWASMModuleWithVariables(t *testing.T) {
    input := []byte("{ var x I64; x = 40; var y I64; y = 2; print(x + y); }\x00")
    Init(input)
    ast := ParseStatement()
    
    // Compile to WASM
    wasmBytes := CompileToWASM(ast)
    
    /* TODO: execute the program and verify it prints 42. */
}
```

#### Test 12: Empty Program (No Variables)
```go
func TestNoVariables(t *testing.T) {
    input := []byte("print(42);\x00")
    Init(input)
    ast := ParseStatement()
    
    locals := collectLocalVariables(ast)
    assert.Equal(t, 0, len(locals))
    
    var buf bytes.Buffer
    EmitCodeSection(&buf, ast)
    
    // Should emit 0 locals (existing behavior)
    bytes := buf.Bytes()
    // Verify locals count is 0 in the generated WASM
}
```

## Implementation Steps

### Step 1: Core Infrastructure
1. Add local variable opcodes constants
2. Implement `LocalVarInfo` struct and collection functions
3. Add variable map building functionality

### Step 2: Code Generation Updates
1. Update `EmitCodeSection` to collect locals and emit locals declarations
2. Create `EmitStatement` function for statement-level code generation
3. Update `EmitExpression` to handle `NodeIdent` as variable references
4. Update `EmitExpression` to handle `NodeBinary` variable assignment statements

### Step 3: Integration
1. Update `CompileToWASM` to use new statement-based emission
2. Add error handling (panic()) for undefined variables
3. Add comprehensive test cases

### Step 4: Testing and Validation
1. Add unit tests for local variable collection
2. Add integration tests for variable usage in expressions
3. Test with existing WASM runtime
4. Validate generated WASM with external tools (`wasm-validate`, `wasm-objdump`)

## Constraints and Limitations

### Current Scope
- **I64 only**: Only integer variables supported
- **No default values**: Variables must be assigned after declaration (unchecked)
- **Function scope**: All locals have function-wide scope (WebAssembly limitation)

### WebAssembly Limitations
- **No block scoping**: WebAssembly locals are function-scoped, not block-scoped
- **No variable shadowing**: Each variable name must be unique within a function
- **Index-based access**: Variables accessed by numeric index, not name at runtime

### Future Extensions
- Support for other data types (F64, I32)
- Variable initialization syntax (`var x I64 = 42`)
- Better error reporting for undefined variables

## Success Criteria

1. **Variable declaration**: `var x I64;` generates valid WASM with proper locals declaration
2. **Variable reference**: `print(x)` correctly emits `local.get` instruction
3. **Multiple variables**: Multiple var declarations create correct local indices
4. **Expression integration**: Variables work correctly in arithmetic expressions
5. **WASM validation**: Generated bytecode passes `wasm-validate` checks
6. **Runtime execution**: Generated WASM runs correctly in wasmtime runtime

This implementation provides a solid foundation for variable support in the Zong WebAssembly backend while maintaining compatibility with the existing expression-based compilation model.
