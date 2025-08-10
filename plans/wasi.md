# WASI Implementation Plan for Zong

## Overview

This document outlines the implementation plan for adding WASI (WebAssembly System Interface) support to the Zong programming language. The implementation is divided into three phases: runtime changes, compiler changes, and creating a WASI prelude.

## Phase 1: wasmruntime Changes

### 1.1 Update Dependencies

**File**: `wasmruntime/Cargo.toml`
- Add `wasmtime-wasi = "26.0"` dependency

### 1.2 Modify Runtime Implementation

**File**: `wasmruntime/src/main.rs`

Changes needed:
1. Import WASI modules:
   ```rust
   use wasmtime_wasi::preview1::{WasiP1Ctx, add_to_linker_sync};
   use wasmtime_wasi::WasiCtxBuilder;
   ```

2. Create WASI context before creating the store:
   ```rust
   let wasi = WasiCtxBuilder::new()
       .inherit_stdio()
       .inherit_env()
       .inherit_args()
       .build_p1();
   ```

3. Update store creation to include WASI context:
   ```rust
   let mut store = Store::new(&engine, wasi);
   ```

4. Create a Linker and add both custom functions and WASI:
   ```rust
   let mut linker = Linker::new(&engine);
   
   // Add WASI functions
   wasmtime_wasi::preview1::add_to_linker_sync(&mut linker, |ctx| ctx)?;
   
   // Add custom functions for backward compatibility
   linker.func_wrap("env", "print", |n: i64| {
       println!("{}", n);
   })?;
   
   // ... add print_bytes and read_line similarly
   ```

5. Use linker to instantiate module instead of direct imports array

### 1.3 Benefits
- Modules can now import from both "env" (legacy) and "wasi_snapshot_preview1"
- Full WASI preview1 API becomes available
- Maintains backward compatibility with existing Zong programs

## Phase 2: Compiler Changes

### 2.1 Add FFI/Extern Syntax Support

**Files to modify**:
- `lexer.go`: Add "extern" keyword
- `parser.go`: Add parsing for extern blocks
- `ast.go`: Add NodeExtern AST node type

**Proposed Syntax**:
```zong
extern "wasi_snapshot_preview1" {
    func fd_write(fd: I32, iovs: I32, iovs_len: I32, nwritten: I32*): I32;
    func fd_read(fd: I32, iovs: I32, iovs_len: I32, nread: I32*): I32;
    func random_get(buf: U8*, buf_len: I32): I32;
    func clock_time_get(id: I32, precision: I64, time: I64*): I32;
}
```

### 2.2 Update Symbol Table

**File**: `main.go` (symbol table section)

Changes:
1. Add `IsExtern bool` and `ModuleName string` fields to FunctionSymbol
2. Register extern functions during symbol table construction
3. Skip body compilation for extern functions

### 2.3 Modify Import Generation

**File**: `main.go` (EmitImportSection)

Changes:
1. Iterate through all extern functions in symbol table
2. Generate imports with appropriate module names (not just "env")
3. Update function indices to account for all imports

### 2.4 Update Function Call Handling

**File**: `main.go` (EmitExpression for NodeCall)

Changes:
1. Check if function is extern
2. Use correct function index based on import order
3. Handle module-specific calling conventions if needed

## Phase 3: WASI Prelude

### 3.1 Create wasi.zong

**File**: `lib/wasi.zong`

This file will contain:
1. WASI function declarations
2. Helper types and constants
3. Convenience wrappers for common operations

```zong
// WASI type definitions
struct IOVec(
    buf: U8*,
    buf_len: I32
);

// WASI errno constants
const ERRNO_SUCCESS: I32 = 0;
const ERRNO_BADF: I32 = 8;
const ERRNO_INVAL: I32 = 28;

// Clock IDs
const CLOCK_REALTIME: I32 = 0;
const CLOCK_MONOTONIC: I32 = 1;

// File descriptors
const STDIN_FD: I32 = 0;
const STDOUT_FD: I32 = 1;
const STDERR_FD: I32 = 2;

// WASI preview1 imports
extern "wasi_snapshot_preview1" {
    // File I/O
    func fd_write(fd: I32, iovs: IOVec*, iovs_len: I32, nwritten: I32*): I32;
    func fd_read(fd: I32, iovs: IOVec*, iovs_len: I32, nread: I32*): I32;
    func fd_close(fd: I32): I32;
    
    // Random
    func random_get(buf: U8*, buf_len: I32): I32;
    
    // Clock
    func clock_time_get(id: I32, precision: I64, time: I64*): I32;
    
    // Environment
    func environ_get(environ: I32*, environ_buf: U8*): I32;
    func environ_sizes_get(environ_count: I32*, environ_buf_size: I32*): I32;
    
    // Process
    func proc_exit(rval: I32);
}

// Convenience functions
func wasi_print_string(s: U8[]): I32 {
    var iov: IOVec = IOVec(buf: s.items, buf_len: I32(s.length));
    var written: I32;
    return fd_write(STDOUT_FD, iov&, 1, written&);
}

func wasi_get_time_ns(): I64 {
    var time: I64;
    var result: I32 = clock_time_get(CLOCK_REALTIME, 0, time&);
    if result == ERRNO_SUCCESS {
        return time;
    }
    return -1;
}

func wasi_fill_random(buffer: U8[], count: I32): Boolean {
    return random_get(buffer.items, count) == ERRNO_SUCCESS;
}
```

## Implementation Order

1. **Start with Phase 1**: Update runtime to support WASI
   - This enables manual testing with hand-written WASM
   - Ensures WASI functions work correctly

2. **Implement Phase 2**: Add compiler support
   - Start with lexer/parser changes
   - Then symbol table updates
   - Finally import generation

3. **Complete with Phase 3**: Create prelude
   - Write comprehensive WASI bindings
   - Add helper functions
   - Create examples and tests

## Testing Strategy

1. **Runtime tests**: Verify WASI functions work with test WASM modules
2. **Compiler tests**: Test extern syntax parsing and code generation
3. **Integration tests**: Full end-to-end tests using WASI functions
4. **Example programs**: Demonstrate real-world WASI usage

## Future Enhancements

1. **WASI Preview2 support**: Component model integration
2. **File I/O**: Add file open/read/write with capability-based security
3. **Networking**: Socket support when WASI adds it
4. **Advanced features**: Threading, shared memory, etc.
