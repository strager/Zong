# Comparison operators tests

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

## Test: comparison with addition precedence
```zong-expr
x + 1 > y
```
```ast
(binary ">" (binary "+" (var "x") 1) (var "y"))
```

## Test: comparison in parentheses
```zong-expr
(x < y) == true
```
```ast
(binary "==" (binary "<" (var "x") (var "y")) true)
```

## Test: comparison with parentheses and equality
```zong-expr
(a < b) == (c > d)
```
```ast
(binary "==" (binary "<" (var "a") (var "b")) (binary ">" (var "c") (var "d")))
```
