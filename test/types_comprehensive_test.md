# Types Comprehensive Tests

## Test: boolean parsing
```zong-program
func main() {
    var x Boolean;
    x = true;
}
```
```ast
[(func "main" [] nil [(var-decl "x" "Boolean") (binary "=" (var "x") true)])]
```

## Test: boolean literal true
```zong-expr
true
```
```ast
true
```

## Test: boolean literal false
```zong-expr
false
```
```ast
false
```