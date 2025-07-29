# Slice parsing tests

## Test: slice variable declaration
```zong-program
var nums I64[];
```
```ast
[(var-decl "nums" "I64[]")]
```

## Test: slice subscript
```zong-expr
nums[0]
```
```ast
(idx (var "nums") 0)
```

## Test: slice assignment
```zong-expr
nums[1] = 42
```
```ast
(binary "=" (idx (var "nums") 1) 42)
```
