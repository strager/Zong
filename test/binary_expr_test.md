# Binary expressions

## Test: +
```zong-expr
1 + 2
```
```ast
(binary "+" 1 2)
```

## Test: -
```zong-expr
1 - 2
```
```ast
(binary "-" 1 2)
```

## Test: *
```zong-expr
1 * 2
```
```ast
(binary "*" 1 2)
```

## Test: ==
```zong-expr
x == y
```
```ast
(binary "==" (var "x") (var "y"))
```

## Test: !=
```zong-expr
x != y
```
```ast
(binary "!=" (var "x") (var "y"))
```

## Test: + with strings
```zong-expr
"a" + "b"
```
```ast
(binary "+" (string "a") (string "b"))
```

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

## Test: operator precedence with parens
```zong-expr
(1 + 2) * 3
```
```ast
(binary "*"
 (binary "+" 1 2)
 3)
```
