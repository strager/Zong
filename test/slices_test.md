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

## Test: address-of operator parsing
```zong-expr
nums&
```
```ast
(unary "&" (var "nums"))
```

## Test: slice append function call parsing
```zong-expr
append(nums&, 42)
```
```ast
(call (var "append") (unary "&" (var "nums")) 42)
```
