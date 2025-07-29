# Arithmetic operators tests

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

## Test: complex arithmetic expression
```zong-expr
(a + b) / (c - d)
```
```ast
(binary "/" (binary "+" (var "a") (var "b")) (binary "-" (var "c") (var "d")))
```

## Test: modulo in function call
```zong-expr
print(x % 10)
```
```ast
(call (var "print") (binary "%" (var "x") 10))
```