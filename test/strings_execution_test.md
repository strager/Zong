# String Execution Tests

## Test: string literal parsing
```zong-expr
"hello"
```
```ast
(string "hello")
```

## Test: string variable assignment parsing
```zong-program
var s U8[];
s = "hello";
```
```ast
[(var-decl "s" "U8[]") (binary "=" (var "s") (string "hello"))]
```