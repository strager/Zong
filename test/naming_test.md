# Naming Convention Tests

## Test: struct name must start with uppercase
```zong-program
struct point(x: I64, y: I64);
```
```compile-error
expected struct name (must start with uppercase letter)
```

## Test: function name must start with lowercase
```zong-program
func Add(): I64 { 
    return 42; 
}
```
```compile-error
expected function name (must start with lowercase letter)
```

## Test: variable name must start with lowercase
```zong-program
func main() {
    var X: I64 = 42;
}
```
```compile-error
expected variable name (must start with lowercase letter)
```


## Test: type name in variable declaration must start with uppercase
```zong-program
func main() {
    var x: point = Point(x: 10, y: 20);
}
```
```compile-error
expected type name (must start with uppercase letter)
```

## Test: valid naming conventions work
```zong-program
struct Point(x: I64, y: I64);

func add(_ a: I64, _ b: I64): I64 { 
    return a + b; 
}

func main() {
    var x: I64 = 42;
    var p: Point = Point(x: 10, y: 20);
    print(add(p.x, p.y));
}
```
```execute
30
```

## Test: underscore names are allowed for functions and variables
```zong-program
func _helper(): I64 { 
    return 42; 
}

func main() {
    var _temp: I64 = _helper();
    print(_temp);
}
```
```execute
42
```