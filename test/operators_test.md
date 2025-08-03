# Operators Tests

Tests for all operators (arithmetic, comparison, precedence).

## Arithmetic Operators

## Test: addition
```zong-expr
1 + 2
```
```ast
(binary "+" 1 2)
```

## Test: subtraction
```zong-expr
1 - 2
```
```ast
(binary "-" 1 2)
```

## Test: multiplication
```zong-expr
1 * 2
```
```ast
(binary "*" 1 2)
```

## Test: division
```zong-expr
x / y
```
```ast
(binary "/" (var "x") (var "y"))
```

## Test: modulo
```zong-expr
x % y
```
```ast
(binary "%" (var "x") (var "y"))
```

## Test: integer division
```zong-expr
10 / 3
```
```ast
(binary "/" 10 3)
```

## Test: integer modulo
```zong-expr
10 % 3
```
```ast
(binary "%" 10 3)
```

## Comparison Operators

## Test: equality
```zong-expr
x == y
```
```ast
(binary "==" (var "x") (var "y"))
```

## Test: not equal
```zong-expr
x != y
```
```ast
(binary "!=" (var "x") (var "y"))
```

## Test: less than
```zong-expr
x < y
```
```ast
(binary "<" (var "x") (var "y"))
```

## Test: greater than
```zong-expr
x > y
```
```ast
(binary ">" (var "x") (var "y"))
```

## Test: less than or equal
```zong-expr
x <= y
```
```ast
(binary "<=" (var "x") (var "y"))
```

## Test: greater than or equal
```zong-expr
x >= y
```
```ast
(binary ">=" (var "x") (var "y"))
```

## Test: integer comparison
```zong-expr
5 < 10
```
```ast
(binary "<" 5 10)
```

## Operator Precedence

## Test: operator precedence + *
```zong-expr
1 + 2 * 3
```
```ast
(binary "+"
 1
 (binary "*" 2 3))
```

## Test: operator precedence + * +
```zong-expr
1 + 2 * 3 + 4
```
```ast
(binary "+"
 (binary "+" 1 (binary "*" 2 3))
 4)
```

## Test: operator precedence with parens
```zong-expr
(1 + 2) * 3
```
```ast
(binary "*"
 (binary "+" 1 2)
 3)
```

## Test: division with multiplication precedence
```zong-expr
x * y / z
```
```ast
(binary "/" (binary "*" (var "x") (var "y")) (var "z"))
```

## Test: modulo with addition precedence
```zong-expr
x + y % z
```
```ast
(binary "+" (var "x") (binary "%" (var "y") (var "z")))
```

## Test: comparison with addition precedence
```zong-expr
x + 1 > y
```
```ast
(binary ">" (binary "+" (var "x") 1) (var "y"))
```

## Test: operator precedence == + *
```zong-expr
a == b + c * d
```
```ast
(binary "=="
 (var "a")
 (binary "+"
  (var "b")
  (binary "*" (var "c") (var "d"))))
```

## Arithmetic Execution Tests

## Test: addition execution
```zong-expr
print(10 + 5)
```
```execute
15
```

## Test: subtraction execution
```zong-expr
print(10 - 3)
```
```execute
7
```

## Test: multiplication execution
```zong-expr
print(6 * 7)
```
```execute
42
```

## Test: division execution
```zong-expr
print(20 / 4)
```
```execute
5
```

## Test: modulo execution
```zong-expr
print(23 % 5)
```
```execute
3
```

## Comparison Execution Tests

## Test: greater than true execution
```zong-expr
print(5 > 3)
```
```execute
1
```

## Test: greater than false execution
```zong-expr
print(3 > 5)
```
```execute
0
```

## Test: equality execution
```zong-expr
print(5 == 5)
```
```execute
1
```

## Test: not equal execution
```zong-expr
print(5 != 3)
```
```execute
1
```

## Test: less than execution
```zong-expr
print(3 < 5)
```
```execute
1
```

## Additional Operator Tests (from expressions_test.md)

## Test: binary addition
```zong-expr
1 + 2
```
```ast
(binary "+" 1 2)
```

## Test: complex expression with variables
```zong-expr
x + y * z
```
```ast
(binary "+"
 (var "x")
 (binary "*" (var "y") (var "z")))
```

## Test: equality with addition
```zong-expr
a == b + c
```
```ast
(binary "=="
 (var "a")
 (binary "+" (var "b") (var "c")))
```

## Test: nested parentheses
```zong-expr
((1 + 2))
```
```ast
(binary "+" 1 2)
```

## Test: complex parentheses
```zong-expr
(x + y) * (a - b)
```
```ast
(binary "*"
 (binary "+" (var "x") (var "y"))
 (binary "-" (var "a") (var "b")))
```

## Test: left associative addition
```zong-expr
1 + 2 + 3
```
```ast
(binary "+"
 (binary "+" 1 2)
 3)
```

## Test: left associative multiplication
```zong-expr
2 * 3 * 4
```
```ast
(binary "*"
 (binary "*" 2 3)
 4)
```

## Test: mixed precedence complex
```zong-expr
1 + 2 * 3 + 4
```
```ast
(binary "+"
 (binary "+" 1 (binary "*" 2 3))
 4)
```

## Test: equality with multiplication precedence
```zong-expr
a == b + c * d
```
```ast
(binary "=="
 (var "a")
 (binary "+"
  (var "b")
  (binary "*" (var "c") (var "d"))))
```

## Test: unary not identifier
```zong-expr
!x
```
```ast
(unary "!" (var "x"))
```

## Test: unary not boolean
```zong-expr
!true
```
```ast
(unary "!" true)
```

## Test: double negation
```zong-expr
!!x
```
```ast
(unary "!" (unary "!" (var "x")))
```

## Test: unary not with parentheses
```zong-expr
!(x == y)
```
```ast
(unary "!" (binary "==" (var "x") (var "y")))
```

## Test: unary not with subscript
```zong-expr
!arr[i]
```
```ast
(unary "!" (idx (var "arr") (var "i")))
```

## Test: subscript with addition
```zong-expr
x[y] + z
```
```ast
(binary "+" (idx (var "x") (var "y")) (var "z"))
```

## Test: complex boolean expression
```zong-expr
!f() == true
```
```ast
(binary "==" (unary "!" (call (var "f"))) true)
```

## Test: integer addition from wasmutil
```zong-expr
42 + 8
```
```ast
(binary "+" 42 8)
```

## Test: string concatenation
```zong-expr
"a" + "b"
```
```ast
(binary "+" (string "a") (string "b"))
```

## Test: subtraction from expressions
```zong-expr
5 - 3
```
```ast
(binary "-" 5 3)
```

## Test: division from expressions
```zong-expr
8 / 2
```
```ast
(binary "/" 8 2)
```

## Test: modulo from expressions
```zong-expr
10 % 3
```
```ast
(binary "%" 10 3)
```

## Test: inequality from expressions
```zong-expr
a != b
```
```ast
(binary "!=" (var "a") (var "b"))
```

## Test: operator precedence multiplication over addition from expressions
```zong-expr
1 + 2 * 3
```
```ast
(binary "+"
 1
 (binary "*" 2 3))
```

## Test: parentheses override precedence from expressions
```zong-expr
(1 + 2) * 3
```
```ast
(binary "*"
 (binary "+" 1 2)
 3)
```

## Arithmetic Execution Tests (from extracted_execution_test.md)

## Test: arithmetic print
```zong-expr
print(42 + 8)
```
```execute
50
```

## Test: complex arithmetic
```zong-expr
print((10 + 5) * 2 - 3)
```
```execute
27
```

## Test: nested expressions
```zong-expr
print(((2 + 3) * 4 - 8) / 2 + 1)
```
```execute
7
```

## Test: operator precedence
```zong-expr
print(1 + 2 * 3)
```
```execute
7
```

## Additional Arithmetic Execution Tests (from execution_test.md)

## Test: arithmetic expression execution
```zong-expr
print(2 + 3)
```
```execute
5
```

## Assignment Operator Compile Error Tests (from compile_error_test.md)

## Test: invalid assignment target
```zong-expr
42 = 10
```
```compile-error
error: left side of assignment must be a variable, field access, or dereferenced pointer
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

## Comprehensive Assignment Tests (from parsing_comprehensive_test.md)

## Test: right associativity of assignment
```zong-expr
a = b = c
```
```ast
(binary "=" (var "a") (binary "=" (var "b") (var "c")))
```

## Test: expression statement with binary operation
```zong-program
a + b;
```
```ast
[(binary "+" (var "a") (var "b"))]
```

## Test: complex expression statement with precedence
```zong-program
x * y + z;
```
```ast
[(binary "+" (binary "*" (var "x") (var "y")) (var "z"))]
```

## Binary Operator Type Verification Tests (from type_verification_test.md)

## Test: binary arithmetic type verification
```zong-program
func main() {
    var result I64;
    result = 5 + 3;
    print(result);
}
```
```execute
8
```

## Test: binary comparison type verification  
```zong-program
func main() {
    var result Boolean;
    result = 42 == 10;
    print(result);
}
```
```execute
0
```

# Assignment Error Tests

Tests for various invalid assignment target scenarios.

### Test: function call as assignment target
```zong-expr
print(42) = 10
```
```compile-error
error: left side of assignment must be a variable, field access, or dereferenced pointer
```

### Test: binary expression as assignment target
```zong-expr
(1 + 2) = 10
```
```compile-error
error: left side of assignment must be a variable, field access, or dereferenced pointer
```

### Test: address-of as assignment target
```zong-program
func main() {
    var x I64;
    x& = 42;
}
```
```compile-error
error: left side of assignment must be a variable, field access, or dereferenced pointer
```

### Test: logical NOT as assignment target
```zong-program
func main() {
    var x Boolean;
    !x = true;
}
```
```compile-error
error: left side of assignment must be a variable, field access, or dereferenced pointer
```
