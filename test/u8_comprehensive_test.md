# U8 Type Tests

## Test: U8 slice declaration
```zong-program
var data U8[];
```
```ast
[(var-decl "data" "U8[]")]
```

## Test: U8 variable declaration  
```zong-program
var b U8;
```
```ast
[(var-decl "b" "U8")]
```

## Test: U8 value in valid range
```zong-program
func main() {
    var b U8;
    b = 255;
    print(b);
}
```
```execute
255
```

## Test: U8 value zero
```zong-program
func main() {
    var b U8;
    b = 0;
    print(b);
}
```
```execute
0
```