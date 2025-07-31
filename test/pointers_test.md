# Pointers Tests

Tests for address-of, dereference, and pointer operations.

## Basic Pointer Operations

## Test: basic pointer assign address dereference to read value
```zong-program
{ var x I64; var ptr I64*; x = 42; ptr = x&; print(ptr*); }
```
```execute
42
```

## Test: modify pointee via pointer read via original variable
```zong-program
{ var x I64; var ptr I64*; x = 10; ptr = x&; ptr* = 99; print(x); }
```
```execute
99
```

## Test: modify via variable read via pointer
```zong-program
{ var x I64; var ptr I64*; x = 25; ptr = x&; x = 77; print(ptr*); }
```
```execute
77
```

## Test: use pointer dereference in arithmetic expression
```zong-program
{ var x I64; var ptr I64*; x = 7; ptr = x&; print(ptr* + 3); }
```
```execute
10
```

## Test: multiple pointers to same variable modify via one read via another
```zong-program
{ var x I64; var ptr1 I64*; var ptr2 I64*; x = 123; ptr1 = x&; ptr2 = x&; print(ptr1*); print(ptr2*); ptr1* = 456; print(ptr2*); }
```
```execute
123
123
456
```

## Test: sequential pointer operations on same variable
```zong-program
{ var x I64; var ptr I64*; x = 100; ptr = x&; print(ptr*); ptr* = 200; print(x); }
```
```execute
100
200
```

## Test: use pointer dereference in complex expression
```zong-program
{ var x I64; var y I64; var ptr I64*; x = 8; y = 7; ptr = x&; print(ptr* * y + 6); }
```
```execute
62
```

## Test: sequential modifications via pointer
```zong-program
{ var x I64; var ptr I64*; x = 5; ptr = x&; ptr* = ptr* + 1; print(x); ptr* = ptr* * 2; print(x); }
```
```execute
6
12
```

## Address-of Operations

## Test: address of identifier
```zong-expr
x&
```
```ast
(unary "&" (var "x"))
```

## Test: address of expression
```zong-expr
(x + y)&
```
```ast
(unary "&" (binary "+" (var "x") (var "y")))
```

## Test: address of subscript
```zong-expr
arr[0]&
```
```ast
(unary "&" (idx (var "arr") 0))
```

## Test: address of with addition precedence
```zong-expr
x& + 1
```
```ast
(binary "+" (unary "&" (var "x")) 1)
```

## Test: addition with address of precedence
```zong-expr
1 + x&
```
```ast
(binary "+" 1 (unary "&" (var "x")))
```

## Test: parentheses with address of
```zong-expr
(x + 1)&
```
```ast
(unary "&" (binary "+" (var "x") 1))
```

## Test: address of variable execution
```zong-program
{ var x I64; x = 42; print(x&); }
```
```execute
0
```

## Test: multiple addressed variables execution
```zong-program
{ var x I64; var y I64; x = 10; y = 20; print(x&); print(y&); }
```
```execute
0
8
```

## Test: address of rvalue expression execution
```zong-program
{ var x I64; x = 5; print((x + 10)&); }
```
```execute
0
```

## Test: stack variable address access
```zong-program
{ var a I64; var b I64; a = 0; b = 0; print(a&); print(b&); print(a); print(b); }
```
```execute
0
8
0
0
```

## Dereference Operations

## Test: dereference pointer
```zong-expr
ptr*
```
```ast
(unary "*" (var "ptr"))
```

## Test: dereference expression
```zong-expr
(ptr + 1)*
```
```ast
(unary "*" (binary "+" (var "ptr") 1))
```

## Test: dereference with addition precedence
```zong-expr
ptr* + 1
```
```ast
(binary "+" (unary "*" (var "ptr")) 1)
```

## Test: addition with dereference precedence
```zong-expr
1 + ptr*
```
```ast
(binary "+" 1 (unary "*" (var "ptr")))
```

## Test: parentheses with dereference
```zong-expr
(ptr + 1)*
```
```ast
(unary "*" (binary "+" (var "ptr") 1))
```

## Advanced Pointer Scenarios

## Test: address of then dereference
```zong-expr
x&*
```
```ast
(unary "*" (unary "&" (var "x")))
```

## Test: dereference then address of
```zong-expr
x*&
```
```ast
(unary "&" (unary "*" (var "x")))
```

## Test: complex pointer operation with addition
```zong-expr
ptr*& + 1
```
```ast
(binary "+" (unary "&" (unary "*" (var "ptr"))) 1)
```

## Test: complex pointer operation with subscript
```zong-expr
arr[0]&*
```
```ast
(unary "*" (unary "&" (idx (var "arr") 0)))
```

## Additional Pointer Tests (from expressions_test.md)

## Test: dereference expression from expressions
```zong-expr
(ptr + 1)*
```
```ast
(unary "*" (binary "+" (var "ptr") 1))
```

## Test: parentheses with dereference from expressions
```zong-expr
(ptr + 1)*
```
```ast
(unary "*" (binary "+" (var "ptr") 1))
```

## Advanced Pointer Scenarios (from TestAdvancedPointerScenarios_test.md)

## Test: Address-of expressions stored on stack at different offsets
```zong-program
{ var x I64; x = 5; print((x + 10)&); print((x * 2)&); }
```
```execute
0
8
```

## Test: Pointer to complex expression results
```zong-program
{ var a I64; var b I64; var c I64; var ptr I64*; a = 1; b = 2; c = 3; ptr = (a + b)&; print(ptr*); ptr = (b * c)&; print(ptr*); }
```
```execute
3
6
```

## Test: Chain of pointer assignments - modify through second pointer
```zong-program
{ var x I64; var ptr1 I64*; var ptr2 I64*; x = 50; ptr1 = x&; ptr2 = ptr1; ptr2* = 75; print(x); print(ptr1*); }
```
```execute
75
75
```

## Pointer TypeAST Tests (from TestTypeASTInCompilation_test.md)

## Test: Pointer variable with TypeAST
```zong-program
{ var ptr I64*; var x I64; x = 99; ptr = x&; print(ptr*); }
```
```execute
99
```

## Phase3 Pointer Tests (from extracted_execution_test.md)

## Test: i64 pointer returns
```zong-program
func getPointer(): I64* {
	var x I64;
	x = 42;
	return x&;
}

func main() {
	var ptr I64*;
	ptr = getPointer();
	print(ptr*);
}
```
```execute
42
```

## Pointer Compile Error Tests (from compile_error_test.md)

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

## Pointer Type Verification Tests (from type_verification_test.md)

## Test: address-of type verification
```zong-program
func main() {
    var x I64;
    var ptr I64*;
    x = 42;
    ptr = x&;
    print(x);
}
```
```execute
42
```