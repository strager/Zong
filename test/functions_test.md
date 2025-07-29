# Function parsing tests

## Test: void function no parameters
```zong-program
func test() {}
```
```ast
[(func "test" [] nil [])]
```

## Test: function with I64 return type
```zong-program
func add(): I64 {}
```
```ast
[(func "add" [] "I64" [])]
```

## Test: function with positional parameters
```zong-program
func add(_ addA: I64, _ addB: I64): I64 {}
```
```ast
[(func "add"
  [(param "addA" "I64" positional)
   (param "addB" "I64" positional)]
  "I64"
  [])]
```

## Test: function with named parameters
```zong-program
func test(testX: I64, testY: I64) {}
```
```ast
[(func "test"
  [(param "testX" "I64" named)
   (param "testY" "I64" named)]
  nil
  [])]
```

## Test: function with body
```zong-program
func test() { var x I64; }
```
```ast
[(func "test"
  []
  nil
  [(var-decl "x" "I64")])]
```
