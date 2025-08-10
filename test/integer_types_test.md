# Integer Types Test Cases

## Test: I64 type declaration and usage
```zong-program
func main() {
    var x: I64 = 42;
    print(x);
}
```
```execute
42
```

## Test: I32 type declaration and usage
```zong-program
func main() {
    var x: I32 = 42;
    print(x);
}
```
```execute
42
```

## Test: I16 type declaration and usage
```zong-program
func main() {
    var x: I16 = 100;
    print(x);
}
```
```execute
100
```

## Test: I8 type declaration and usage
```zong-program
func main() {
    var x: I8 = 127;
    print(x);
}
```
```execute
127
```

## Test: U64 type declaration and usage
```zong-program
func main() {
    var x: U64 = 1000000000;
    print(x);
}
```
```execute
1000000000
```

## Test: U32 type declaration and usage
```zong-program
func main() {
    var x: U32 = 100000;
    print(x);
}
```
```execute
100000
```

## Test: U16 type declaration and usage
```zong-program
func main() {
    var x: U16 = 65535;
    print(x);
}
```
```execute
65535
```

## Test: U8 type declaration and usage
```zong-program
func main() {
    var x: U8 = 255;
    print(x);
}
```
```execute
255
```

## Test: Mixed integer operations
```zong-program
func main() {
    var a: I32 = 10;
    var b: I32 = 20;
    var c: I32 = a + b;
    print(c);
}
```
```execute
30
```

## Test: Integer type boundaries - I8
```zong-program
func main() {
    var min: I8 = -128;
    var max: I8 = 127;
    print(min);
    print(max);
}
```
```execute
-128
127
```

## Test: Integer type boundaries - U8
```zong-program
func main() {
    var min: U8 = 0;
    var max: U8 = 255;
    print(min);
    print(max);
}
```
```execute
0
255
```

## Test: Array of I32
```zong-program
func main() {
    var nums: I32[];
    append(nums&, 10);
    append(nums&, 20);
    append(nums&, 30);
    print(nums[0]);
    print(nums[1]);
    print(nums[2]);
}
```
```execute
10
20
30
```

## Test: Struct with integer types
```zong-program
struct Data(a: I32, b: U16, c: I8);

func main() {
    var d: Data = Data(a: 100, b: 200, c: 50);
    print(d.a);
    print(d.b);
    print(d.c);
}
```
```execute
100
200
50
```

## Test: Function with integer type parameters
```zong-program
func add32(_ a: I32, _ b: I32): I32 {
    return a + b;
}

func main() {
    var result: I32 = add32(15, 25);
    print(result);
}
```
```execute
40
```

## Test: Type compatibility - Integer literal to I16
```zong-program
func main() {
    var x: I16 = 1000;
    print(x);
}
```
```execute
1000
```

## Test: Negative values in signed types
```zong-program
func main() {
    var a: I32 = -42;
    var b: I16 = -1000;
    var c: I8 = -50;
    print(a);
    print(b);
    print(c);
}
```
```execute
-42
-1000
-50
```
