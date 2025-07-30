# Local Variables Collection Tests

## Test: single local variable
```zong-program
func main() { var x I64; }
```
```wasm-locals
[(local "x" "I64" local 0)]
```

## Test: multiple local variables
```zong-program
func main() { var x I64; var y I64; }
```
```wasm-locals
[(local "x" "I64" local 0) (local "y" "I64" local 1)]
```

## Test: nested block variables
```zong-program
func main() { var a I64; { var b I64; } }
```
```wasm-locals
[(local "a" "I64" local 0) (local "b" "I64" local 1)]
```

## Test: no variables
```zong-program
func main() { print(42); }
```
```wasm-locals
[]
```

## Test: undefined variable reference
```zong-program
func main() { print(undefined_var); }
```
```compile-error
undefined symbol 'undefined_var'
```

## Test: single pointer variable
```zong-program
func main() { var ptr I64*; }
```
```wasm-locals
[(local "ptr" "I64*" local 0)]
```

## Test: mixed pointer and regular variables
```zong-program
func main() { var x I64; var ptr I64*; var y I64; }
```
```wasm-locals
[(local "x" "I64" local 1) (local "ptr" "I64*" local 0) (local "y" "I64" local 2)]
```

## Test: multiple pointer variables
```zong-program
func main() { var ptr1 I64*; var ptr2 I64*; }
```
```wasm-locals
[(local "ptr1" "I64*" local 0) (local "ptr2" "I64*" local 1)]
```

## Test: nested block pointer variables
```zong-program
func main() { var a I64; { var ptr I64*; } var b I64*; }
```
```wasm-locals
[(local "a" "I64" local 2) (local "ptr" "I64*" local 0) (local "b" "I64*" local 1)]
```

## Test: pointer variables in WASM code generation
```zong-program
func main() { var x I64; var ptr I64*; x = 42; print(x); }
```
```execute
42
```

## Test: addressed single variable
```zong-program
func main() { var x I64; print(x&); }
```
```wasm-locals
[(local "x" "I64" tstack 0)]
```

## Test: addressed multiple variables
```zong-program
func main() { var x I64; var y I64; print(x&); print(y&); }
```
```wasm-locals
[(local "x" "I64" tstack 0) (local "y" "I64" tstack 8)]
```

## Test: mixed addressed and non-addressed variables
```zong-program
func main() { var a I64; var b I64; var c I64; print(b&); }
```
```wasm-locals
[(local "a" "I64" local 0) (local "b" "I64" tstack 0) (local "c" "I64" local 1)]
```

## Test: addressed variable frame offset calculation
```zong-program
func main() { var a I64; var b I64; var c I64; var d I64; print(a&); print(c&); print(d&); }
```
```wasm-locals
[(local "a" "I64" tstack 0) (local "b" "I64" local 0) (local "c" "I64" tstack 8) (local "d" "I64" tstack 16)]
```

## Test: address of rvalue
```zong-program
func main() { var x I64; print((x + 1)&); }
```
```wasm-locals
[(local "x" "I64" local 0)]
```