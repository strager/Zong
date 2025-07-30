# Compile Error Tests

## Test: undefined variable error
```zong-expr
undefinedVar
```
```compile-error
undefined symbol 'undefinedVar'
```

## Test: variable used before assignment
```zong-program
func main() {
    var x I64;
    var y Boolean;
    x = y;
}
```
```compile-error
error: variable 'y' used before assignment
```

## Test: undefined function call
```zong-expr
nonExistentFunction()
```
```compile-error
undefined symbol 'nonExistentFunction'
```

## Test: variable not declared in expression
```zong-expr
undefined
```
```compile-error
undefined symbol 'undefined'
```

## Test: variable not assigned in expression  
```zong-program
func main() {
    var x I64;
    print(x);
}
```
```compile-error
error: variable 'x' used before assignment
```

## Test: dereference non-pointer type
```zong-program
func main() {
    var x I64 = 42;
    print(*x);
}
```
```compile-error
error: unsupported expression type ''
```

## Test: unknown function in call
```zong-expr
unknown(42)
```
```compile-error
undefined symbol 'unknown'
```

## Test: assignment to undeclared variable
```zong-program
func main() {
    undefined = 42;
}
```
```compile-error
undefined symbol 'undefined'
```

## Test: program with variable used before assignment
```zong-program
func main() {
    var x I64;
    print(x);
}
```
```compile-error
error: variable 'x' used before assignment
```

## Test: U8 out of range value
```zong-program
func main() {
    var slice U8[];
    append(slice&, 256);
}
```
```compile-error
error: append() cannot convert integer 256 to U8
```

## Test: break outside of loop
```zong-program
func main() {
    break;
}
```
```compile-error
error: break statement outside of loop
```

## Test: continue outside of loop
```zong-program
func main() {
    continue;
}
```
```compile-error
error: continue statement outside of loop
```

## Test: dereference non-pointer in assignment
```zong-program
func main() {
    var x I64;
    x = 42;
    print(*x);
}
```
```compile-error
error: unsupported expression type ''
```

## Test: print with no arguments
```zong-expr
print()
```
```compile-error
error: print() function expects 1 argument
```

## Test: duplicate variable declaration
```zong-program
func main() {
    var x I64;
    var x I64;
}
```
```compile-error
error: variable 'x' already declared
```

## Test: invalid assignment target
```zong-expr
42 = 10
```
```compile-error
error: left side of assignment must be a variable, field access, or dereferenced pointer
```

## Test: field access on non-struct type
```zong-program
func main() {
    var x I64;
    x = 42;
    print(x.field);
}
```
```compile-error
error: cannot access field of non-struct type I64
```

## Test: type mismatch Boolean to I64
```zong-program
func main() {
    var x Boolean;
    x = true;
    var y I64;
    y = x;
}
```
```compile-error
error: cannot assign Boolean to I64
```

## Test: assignment to undefined variable
```zong-program
func main() {
    undefinedVar = 42;
}
```
```compile-error
undefined symbol 'undefinedVar'
```

## Test: assignment to invalid target
```zong-program
func main() {
    10 = 42;
}
```
```compile-error
error: left side of assignment must be a variable, field access, or dereferenced pointer
```

## Test: address of undefined variable
```zong-program
func main() {
    var ptr I64* = &undefinedVar;
}
```
```compile-error
undefined symbol 'undefinedVar'
```
