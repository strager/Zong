# WebAssembly Backend

Create a WebAssembly code generator that compiles Zong AST nodes to WebAssembly binary format.

## Scope

This initial WASM backend will support:
- **Numbers only**: i64 integer arithmetic and comparison operations
- **Single function**: `print(n)` for output (no strings)
- **Expression evaluation**: Arithmetic expressions with operator precedence

## WebAssembly Module Structure

```
WASM Module:
├── Import Section
│   └── print: (i64) -> () from "env" module
├── Function Section  
│   └── main: () -> ()
├── Export Section
│   └── export main as "main"
└── Code Section
    └── main function bytecode
```

## AST to WASM Code Generation

### Code Generation Strategy

Use **recursive descent** over the AST to emit WASM instructions in **stack order**:

1. **Post-order traversal**: Children first, then parent operation
2. **Stack-based evaluation**: WASM uses an implicit operand stack
3. **Immediate instruction emission**: Generate bytecode during AST walk

### Node Translation Rules

| AST Node | WASM Instructions | Example |
|----------|------------------|---------|
| `NodeInteger` | `i64.const <value>` | `42` → `i64.const 42` |
| `NodeBinary "+"` | `<left> <right> i64.add` | `1+2` → `i64.const 1 i64.const 2 i64.add` |
| `NodeBinary "-"` | `<left> <right> i64.sub` | `5-3` → `i64.const 5 i64.const 3 i64.sub` |
| `NodeBinary "*"` | `<left> <right> i64.mul` | `2*3` → `i64.const 2 i64.const 3 i64.mul` |
| `NodeBinary "/"` | `<left> <right> i64.div_s` | `6/2` → `i64.const 6 i64.const 2 i64.div_s` |
| `NodeBinary "%"` | `<left> <right> i64.rem_s` | `7%3` → `i64.const 7 i64.const 3 i64.rem_s` |
| `NodeBinary "=="` | `<left> <right> i64.eq` | `x==y` → `<x> <y> i64.eq` |
| `NodeBinary "!="` | `<left> <right> i64.ne` | `x!=y` → `<x> <y> i64.ne` |
| `NodeBinary "<"` | `<left> <right> i64.lt_s` | `x<y` → `<x> <y> i64.lt_s` |
| `NodeBinary ">"` | `<left> <right> i64.gt_s` | `x>y` → `<x> <y> i64.gt_s` |
| `NodeBinary "<="` | `<left> <right> i64.le_s` | `x<=y` → `<x> <y> i64.le_s` |
| `NodeBinary ">="` | `<left> <right> i64.ge_s` | `x>=y` → `<x> <y> i64.ge_s` |
| `NodeCall "print"` | `<arg> call $print` | `print(42)` → `i64.const 42 call $print` |

### Code Generation Algorithm

```go
func CompileToWASM(ast *ASTNode) []byte {
    var buf bytes.Buffer
    
    // Emit WASM module header and sections in streaming fashion
    EmitWASMHeader(&buf)
    EmitImportSection(&buf)      // print function import
    EmitFunctionSection(&buf)    // declare main function
    EmitExportSection(&buf)      // export main function
    EmitCodeSection(&buf, ast)   // main function body with compiled expression
    
    return buf.Bytes()
}

func EmitCodeSection(buf *bytes.Buffer, ast *ASTNode) {
    // Emit code section header
    writeByte(buf, 0x0A)  // code section id
    
    // Calculate function body size (will be written after body generation)
    bodyStart := buf.Len() + 4  // reserve 4 bytes for section size
    
    writeLEB128(buf, 1)   // 1 function
    
    // Generate function body
    var bodyBuf bytes.Buffer
    writeLEB128(&bodyBuf, 0)  // 0 locals
    
    // Emit expression bytecode
    EmitExpression(&bodyBuf, ast)
    writeByte(&bodyBuf, 0x0B)  // end instruction
    
    // Write body size and body
    writeLEB128(buf, uint32(bodyBuf.Len()))
    buf.Write(bodyBuf.Bytes())
}

func EmitExpression(buf *bytes.Buffer, node *ASTNode) {
    switch node.Kind {
    case NodeInteger:
        writeByte(buf, 0x42)  // i64.const
        writeLEB128Signed(buf, node.Integer)
    
    case NodeBinary:
        EmitExpression(buf, node.Children[0])  // left operand
        EmitExpression(buf, node.Children[1])  // right operand
        writeByte(buf, getBinaryOpcode(node.Op))
    
    case NodeCall:
        if node.Children[0].String == "print" {
            EmitExpression(buf, node.Children[1])  // argument
            writeByte(buf, 0x10)  // call instruction
            writeLEB128(buf, 0)   // function index 0 (print import)
        }
    }
}
```

## Implementation Plan

### Phase 1: WASM Binary Streaming
1. **Binary Encoding Utilities**
   ```go
   func writeByte(buf *bytes.Buffer, b byte)
   func writeLEB128(buf *bytes.Buffer, val uint32)
   func writeLEB128Signed(buf *bytes.Buffer, val int64)
   func writeBytes(buf *bytes.Buffer, data []byte)
   ```

2. **Section Emitters**
   ```go
   func EmitWASMHeader(buf *bytes.Buffer)     // Magic + version
   func EmitImportSection(buf *bytes.Buffer)  // print function import
   func EmitFunctionSection(buf *bytes.Buffer) // main function signature
   func EmitExportSection(buf *bytes.Buffer)  // export "main"
   func EmitCodeSection(buf *bytes.Buffer, ast *ASTNode) // function bodies
   ```

3. **Opcode Constants**
   ```go
   const (
       I64_CONST = 0x42
       I64_ADD   = 0x7C
       I64_SUB   = 0x7D
       I64_MUL   = 0x7E
       I64_DIV_S = 0x7F
       I64_REM_S = 0x81
       I64_EQ    = 0x51
       I64_NE    = 0x52
       I64_LT_S  = 0x53
       I64_GT_S  = 0x55
       I64_LE_S  = 0x57
       I64_GE_S  = 0x59
       CALL      = 0x10
       END       = 0x0B
   )
   ```

### Phase 2: AST Integration
1. **Compiler Entry Point**
   ```go
   func CompileAST(ast *ASTNode) []byte
   ```

2. **Expression Compiler**
   - Recursive AST traversal with direct bytecode emission
   - No intermediate instruction representation needed
   - `panic()` for unsupported nodes

### Phase 3: Runtime Integration  
1. **Host Environment Setup**
   - Rust-based WASM runtime using `wasmtime`
   - Print function implementation: `fn print(n: i64) { println!("{n}"); }`

2. **Execution Pipeline**
   - ParseStatement()
   - CompileAST()
   - Write to .wasm file
   - Use `wasm-validate`, `wasm-objdump`, and `wasm2wat` tools to introspect (e.g. dump WAT if a test fails)
   - Execute .wasm file using wasmtime
   - In tests: Capture output (stdout)

## Test Cases

### Basic Arithmetic
```
Input:   42
AST:     (integer 42)
WASM:    i64.const 42
Output:  (none; no print calls)
```

```
Input:   1 + 2 * 3  
AST:     (binary "+" (integer 1) (binary "*" (integer 2) (integer 3)))
WASM:    i64.const 1 i64.const 2 i64.const 3 i64.mul i64.add
Output:  (none; no print calls)
```

### Comparisons
```
Input:   5 > 3
AST:     (binary ">" (integer 5) (integer 3))  
WASM:    i64.const 5 i64.const 3 i64.gt_s
Output:  (none; no print calls)
```

### Print Function
```
Input:   print(42 + 8)
AST:     (call (ident "print") (binary "+" (integer 42) (integer 8)))
WASM:    i64.const 42 i64.const 8 i64.add call $print  
Output:  50 (printed to console)
```

### Complex Expression  
```
Input:   print((10 + 5) * 2 - 3)
AST:     (call (ident "print") (binary "-" (binary "*" (binary "+" (integer 10) (integer 5)) (integer 2)) (integer 3)))
WASM:    i64.const 10 i64.const 5 i64.add i64.const 2 i64.mul i64.const 3 i64.sub call $print
Output:  27 (printed to console)
```

## File Structure

All code for the WASM backend, including utilities, goes in main.go.

All code for the Rust-based WASM runtime goes in the new `wasmruntime/` directory.

## Limitations

### Current Scope Exclusions
- **No variables**: Only literal numbers and expressions
- **No strings**: Print function only accepts numbers
- **No control flow**: No if/else, loops, or functions
- **No memory management**: Stack-based evaluation only

### Future Extensions
- Variable support with local.get/local.set
- String literals and string printing
- Control flow instructions (if, loop, br)
- Function definitions and calls
- Memory operations for complex data types

## Dependencies

- **Rust and Cargo** (stable if possible)
- **Rust WASM runtime**: wasmtime crate: https://crates.io/crates/wasmtime
- **WASM binary encoding**: No dependencies; custom streaming implementation with LEB128 encoding

## Success Criteria

1. **Compile simple arithmetic**: `42 + 8` generates valid WASM
2. **Execute expressions**: Generated WASM runs and produces correct results
3. **Print function works**: `print(result)` outputs to console
4. **Operator precedence preserved**: `1 + 2 * 3` evaluates as `7`, not `9`
5. **Comprehensive test coverage**: All supported operators and combinations tested

The WASM backend provides a foundation for compiling Zong to a portable, performant target while maintaining the language's expression semantics and operator precedence rules.
