# Comprehensive Parsing Tests

## Test: right associativity of assignment
```zong-expr
a = b = c
```
```ast
(binary "=" (var "a") (binary "=" (var "b") (var "c")))
```

## Test: empty block statement
```zong-program
{ }
```
```ast
[(block [])]
```

## Test: block with single expression
```zong-program
{ x; }
```
```ast
[(block [(var "x")])]
```

## Test: block with multiple expressions
```zong-program
{ 1; 2; }
```
```ast
[(block [1 2])]
```

## Test: block with variable and return
```zong-program
{ var x int; return x; }
```
```ast
[(block [(var-decl "x" "int") (return (var "x"))])]
```

## Test: nested empty blocks
```zong-program
{ { } }
```
```ast
[(block [(block [])])]
```


## Test: expression statement with variable
```zong-program
x;
```
```ast
[(var "x")]
```

## Test: expression statement with integer
```zong-program
42;
```
```ast
[42]
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

## Test: complex if statement with nested blocks
```zong-program
if x > 0 { var y int; return y + 1; }
```
```ast
[(if (binary ">" (var "x") 0) [(var-decl "y" "int") (return (binary "+" (var "y") 1))])]
```

## Test: loop with break and continue
```zong-program
loop { if done { break; } continue; }
```
```ast
[(loop [(if (var "done") [break]) continue])]
```

## Test: deeply nested blocks
```zong-program
{ if a { { b; } } }
```
```ast
[(block [(if (var "a") [(block [(var "b")])])])]
```