# Expression parsing tests

## Test: integer literal
```zong-expr
42
```
```ast
42
```

## Test: string literal
```zong-expr
"hello"
```
```ast
(string "hello")
```

## Test: identifier
```zong-expr
myVar
```
```ast
(var "myVar")
```

## Test: binary addition
```zong-expr
1 + 2
```
```ast
(binary "+" 1 2)
```

## Test: binary equality
```zong-expr
x == y
```
```ast
(binary "==" (var "x") (var "y"))
```

## Test: string concatenation
```zong-expr
"a" + "b"
```
```ast
(binary "+" (string "a") (string "b"))
```

## Test: subtraction
```zong-expr
5 - 3
```
```ast
(binary "-" 5 3)
```

## Test: division
```zong-expr
8 / 2
```
```ast
(binary "/" 8 2)
```

## Test: modulo
```zong-expr
10 % 3
```
```ast
(binary "%" 10 3)
```

## Test: inequality
```zong-expr
a != b
```
```ast
(binary "!=" (var "a") (var "b"))
```

## Test: operator precedence multiplication over addition
```zong-expr
1 + 2 * 3
```
```ast
(binary "+"
 1
 (binary "*" 2 3))
```

## Test: parentheses override precedence
```zong-expr
(1 + 2) * 3
```
```ast
(binary "*"
 (binary "+" 1 2)
 3)
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

## Test: function call no args
```zong-expr
f()
```
```ast
(call (var "f"))
```

## Test: function call with string arg
```zong-expr
print("hello")
```
```ast
(call (var "print") (string "hello"))
```

## Test: function call multiple args
```zong-expr
atan2(y, x)
```
```ast
(call (var "atan2") (var "y") (var "x"))
```

## Test: function call with named parameters
```zong-expr
Point(x: 1, y: 2)
```
```ast
(call (var "Point") "x" 1 "y" 2)
```

## Test: function call mixed parameters
```zong-expr
httpGet("http://example.com", headers: h)
```
```ast
(call (var "httpGet") (string "http://example.com") "headers" (var "h"))
```

## Test: parenthesized function call
```zong-expr
(foo)()
```
```ast
(call (var "foo"))
```

## Test: chained function call
```zong-expr
arr[0](x)
```
```ast
(call (idx (var "arr") 0) (var "x"))
```

## Test: array subscript variable
```zong-expr
x[y]
```
```ast
(idx (var "x") (var "y"))
```

## Test: array subscript integer
```zong-expr
arr[0]
```
```ast
(idx (var "arr") 0)
```

## Test: nested array subscript
```zong-expr
matrix[i][j]
```
```ast
(idx (idx (var "matrix") (var "i")) (var "j"))
```

## Test: array subscript with expression
```zong-expr
items[x + 1]
```
```ast
(idx (var "items") (binary "+" (var "x") 1))
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
(unary "!" (boolean true))
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

## Test: function call with subscript
```zong-expr
f(x)[0]
```
```ast
(idx (call (var "f") (var "x")) 0)
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
(binary "==" (unary "!" (call (var "f"))) (boolean true))
```

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

## Test: boolean literal true
```zong-expr
true
```
```ast
(boolean true)
```

## Test: boolean literal false
```zong-expr
false
```
```ast
(boolean false)
```

## Test: integer addition from wasmutil
```zong-expr
42 + 8
```
```ast
(binary "+" 42 8)
```