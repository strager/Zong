# Type Verification Tests

## Test: integer literal type verification
```zong-expr
42
```
```ast
42
```

## Test: binary arithmetic type verification
```zong-program
func main() {
    var result I64;
    result = 5 + 3;
    print(result);
}
```
```execute
8
```

## Test: binary comparison type verification  
```zong-program
func main() {
    var result Boolean;
    result = 42 == 10;
    print(result);
}
```
```execute
0
```

## Test: address-of type verification
```zong-program
func main() {
    var x I64;
    var ptr I64*;
    x = 42;
    ptr = x&;
    print(x);
}
```
```execute
42
```

## Test: function call type verification
```zong-program
func main() {
    print(42);
}
```
```execute
42
```