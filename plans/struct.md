# Structs in Zong

## Syntax

Declaration:

```
struct MyStruct {
    var field: I64;
    var field2: I64*;
}
```

Type use:

```
var s: MyStruct;
```

Field access:

```
print(s.field);  // read/load
s.field2* = 42;  // write/store
```

Copying:

```
var s2: MyStruct;
// Equivalent to copying all fields or a memcpy.
s2 = s;
```

## Memory Model

Structs are always stored in memory (tstack), not in WASM locals or on the WASM stack.

Struct fields are stored in order of declaration, like in C.

Currently, there is no padding between or after fields, but this might be implemented in the future.

Field offsets are calculated as cumulative byte sizes: first field at offset 0, second field at offset 8 (assuming I64 = 8 bytes), etc.

## Implementation Details

### Parser Integration

New AST node types:
- `NodeStruct`: Represents struct declaration with field list
- `NodeDot`: Represents `s.field` access with base expression and field name
- `NodeStructType`: Represents struct type usage in variable declarations

### Lexer Tokens

New tokens needed:
- `STRUCT`: `struct` keyword
- `DOT`: `.` operator for field access

### Type System

- Struct types stored in global symbol table with field information
- Each struct type contains
  - list of field names, types, and byte offsets
- Type checking validates field existence and types during compilation

### WASM Compilation

- Struct variables allocated as memory offsets in linear memory
- Field access compiles to memory load/store operations:
  - `s.field` → `i32.load offset=field_offset` (base address + field offset)
  - `s.field = value` → `i32.store offset=field_offset`
- Struct assignment compiles to memory copy operation

### Memory Management

- Structs allocated on tstack like other variables
- Base address stored in WASM local variable or computed from tstack pointer
- Struct size calculated as byte offset of last field + size of last field

### Error Handling

- Parse error for invalid struct syntax
- Type error for accessing non-existent fields
- Type error for field type mismatches
- Error for using undeclared struct types
