# Memory Model Implementation Plan for Zong

This document outlines the implementation plan for the memory model described in `memory.md` for the Zong compiler.

## Overview

The memory model introduces:
- **tstack** (thread stack): Linear memory region for stack allocation
- **tstack pointer**: Points to top of tstack (initialized to address 0)
- **frame pointer**: Points within tstack, computed per function
- **address-of operator (`&`)**: Takes addresses of lvalues and rvalues

## Current Compiler State Analysis

The Zong compiler currently supports:
- I64 local variables with WASM local indexing (`LocalVarInfo` struct)
- Expression parsing with precedence climbing
- WASM code generation for arithmetic and assignments
- Function-scoped variables (WebAssembly limitation)

**Missing for memory model:**
- Stack pointer management in WASM
- Frame-based memory allocation

## Implementation Plan

### Phase 2: AST and Semantic Analysis

#### 2.1 Variable Classification
- Add field to track which variables need stack allocation (`isAddressed` flag)
- Scan AST for address-of operations to mark variables as addressed
- Distinguish between lvalue `variable&` and rvalue `(expression+1)&` cases

#### 2.2 Stack Layout Planning  
- Calculate stack frame size based on addressed variables
- Assign frame offsets to addressed variables during `collectLocalVariables()`
- Track both WASM local indices and frame offsets in `LocalVarInfo`

### Phase 3: WASM Code Generation

#### 3.1 Runtime Stack Management
- Add WASM global for tstack pointer (import from runtime)
- Add WASM local for frame pointer in each function
- Generate frame setup code on function entry:
  ```wasm
  global.get $tstack_pointer
  local.set $frame_pointer
  global.get $tstack_pointer
  i64.const [frame_size]
  i64.add
  global.set $tstack_pointer
  ```

#### 3.2 Address-of Code Generation
- For lvalue `variable&`:
  - Load frame pointer: `local.get $frame_pointer`
  - Add variable offset: `i64.const [offset]`, `i64.add`
- For rvalue `expression&`:
  - Emit expression code to compute value
  - Store at tstack pointer: `global.get $tstack_pointer`, `i64.store`
  - Increment tstack pointer: `global.get $tstack_pointer`, `i64.const 8`, `i64.add`, `global.set $tstack_pointer`
  - Return old tstack pointer value

#### 3.3 Memory Operations
- Add WASM opcodes: `I64_STORE = 0x37`, `I64_LOAD = 0x29`
- Implement store/load operations for pointer dereferencing
- Update `EmitExpression()` to handle `NodeUnary` `&` nodes

### Phase 4: Runtime Environment

- Add tstack WASM global (initialized to 0 on module instantiation)

### Phase 5: Testing

#### Local TStack Allocation Tests
in locals_test.go:

- Test `&variable` with local variables, should allocate space
- Test multiple variables; allocated parts of frame shouldn't overlap

#### End-to-End Tests
in compiler_test.go:

- Compile and execute programs using address-of
- Verify correct stack allocation and pointer values
- Test interaction with existing features (assignments, arithmetic)

## Implementation Order

1. **AST Analysis**: Mark addressed variables and calculate frame sizes
2. **Testing** of AST analysis
3. **WASM Generation**: Implement stack management, address-of emission, and tstack setup
4. **Testing** of WASM generation

## Files to Modify

- `main.go`: All lexer, parser, and WASM generation changes
- `compiler_test.go` and `locals_test.go`
- `CLAUDE.md`: Update with new memory model features

## Technical Notes

- Stack grows upward (increment pointer to allocate)
- All values are 8 bytes (I64 and I64* both fit in i64)
- Frame pointer remains constant within function scope
- WASM linear memory limitation: need to use globals for stack pointers
- Address-of rvalues create temporary stack allocations
