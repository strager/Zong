# Struct parsing tests

## Test: struct declaration
```zong-program
struct Point { var x I64; var y I64; }
```
```ast
[(struct "Point"
  [(field "x" "I64")
   (field "y" "I64")])]
```

## Test: struct type in variable declaration
```zong-program
var p Point;
```
```ast
[(var-decl "p" "Point")]
```

## Test: field access
```zong-expr
p.x
```
```ast
(dot (var "p") "x")
```

## Test: field assignment
```zong-expr
p.x = 42
```
```ast
(binary "=" (dot (var "p") "x") 42)
```

## Test: complex struct expression
```zong-expr
p.x + q.y
```
```ast
(binary "+" (dot (var "p") "x") (dot (var "q") "y"))
```

## Test: struct field access and assignment execution
```zong-program
struct Point { var x I64; var y I64; }
func main() {
    var p Point;
    p.x = 42;
    p.y = 24;
    print(p.x);
    print(p.y);
}
```
```execute
42
24
```
