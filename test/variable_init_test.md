# Variable initialization tests

## Test: integer initialization
```zong-program
var x I64 = 42;
```
```ast
[(var-decl "x" "I64" 42)]
```

## Test: boolean initialization
```zong-program
var flag Boolean = true;
```
```ast
[(var-decl "flag" "Boolean" (boolean true))]
```

## Test: string initialization
```zong-program
var name string = "hello";
```
```ast
[(var-decl "name" "string" (string "hello"))]
```

## Test: variable initialization with expression
```zong-program
var result I64 = x + y;
```
```ast
[(var-decl "result" "I64" (binary "+" (var "x") (var "y")))]
```

## Test: initialization with function call
```zong-program
var value I64 = getValue();
```
```ast
[(var-decl "value" "I64" (call (var "getValue")))]
```

## Test: initialization without semicolon
```zong-program
var count I64 = 0
```
```ast
[(var-decl "count" "I64" 0)]
```

## Test: pointer initialization
```zong-program
var ptr I64* = ptr&;
```
```ast
[(var-decl "ptr" "I64*" (unary "&" (var "ptr")))]
```