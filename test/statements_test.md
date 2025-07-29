# Statement parsing tests

## Test: if statement simple
```zong-program
if x { y; }
```
```ast
[(if (var "x") [(var "y")])]
```

## Test: if statement with expression
```zong-program
if 1 + 2 { 3; }
```
```ast
[(if (binary "+" 1 2) [3])]
```

## Test: if statement with equality
```zong-program
if foo == bar { return 42; }
```
```ast
[(if (binary "==" (var "foo") (var "bar"))
 [(return 42)])]
```

## Test: if else statement
```zong-program
if x { y; } else { z; }
```
```ast
[(if (var "x")
  [(var "y")]
  nil
  [(var "z")])]
```

## Test: if else with expressions
```zong-program
if x == 1 { print(1); } else { print(0); }
```
```ast
[(if (binary "==" (var "x") 1)
  [(call (var "print") 1)]
  nil
  [(call (var "print") 0)])]
```

## Test: if else if else chain
```zong-program
if x > 0 { print(1); } else if x < 0 { print(2); } else { print(0); }
```
```ast
[(if (binary ">" (var "x") 0)
  [(call (var "print") 1)]
  (binary "<" (var "x") 0)
  [(call (var "print") 2)]
  nil
  [(call (var "print") 0)])]
```

## Test: variable declaration with type
```zong-program
var x int;
```
```ast
[(var-decl "x" "int")]
```

## Test: variable declaration string type
```zong-program
var name string;
```
```ast
[(var-decl "name" "string")]
```

## Test: variable declaration custom type
```zong-program
var count MyType;
```
```ast
[(var-decl "count" "MyType")]
```

## Test: pointer variable declaration
```zong-program
var ptr I64*;
```
```ast
[(var-decl "ptr" "I64*")]
```

## Test: slice variable declaration
```zong-program
var data I64[];
```
```ast
[(var-decl "data" "I64[]")]
```
