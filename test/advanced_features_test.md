# Advanced Features Tests

## Test: I64 pointer return type parsing
```zong-program
func getPointer(): I64* {
    return null;
}
```
```ast
[(func "getPointer" [] "I64*" [(return (var "null"))])]
```

## Test: struct parameter parsing
```zong-program
func test(_ testP: Point): I64 { return 42; }
```
```ast
[(func "test" [(param "testP" "Point*" positional)] "I64" [(return 42)])]
```