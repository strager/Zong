# Parser Robustness Tests

These tests verify that the parser handles malformed input gracefully without crashing.

## Test: if statement without brace
```zong-expr
if x == 1 ;
```
```compile-error

```

## Test: var statement without variable name
```zong-expr
var ;
```
```compile-error

```

## Test: var statement without type
```zong-expr
var x ;
```
```compile-error

```

## Test: loop statement without brace
```zong-expr
loop ;
```
```compile-error

```

## Test: var statement with invalid type
```zong-expr
var x 123;
```
```compile-error

```

## Test: malformed function call missing comma
```zong-expr
func(arg1 arg2)
```
```compile-error

```