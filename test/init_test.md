# Variable Initialization Tests

## Test: basic uninitialized use
```zong-program
func main() {
    var x: I64;
    print(x);
}
```
```compile-error
error: variable 'x' used before assignment
```

## Test: initialized variable is ok
```zong-program
func main() {
    var x: I64 = 42;
    print(x);
}
```
```execute
42
```

## Test: assignment then use is ok
```zong-program
func main() {
    var x: I64;
    x = 42;
    print(x);
}
```
```execute
42
```



## Test: struct initialized is ok
```zong-program
struct Point(x: I64, y: I64);

func main() {
    var p: Point = Point(x: 10, y: 20);
    print(p.x);
    print(p.y);
}
```
```execute
10
20
```


## Test: if both branches initialize
```zong-program
func main() {
    var x: I64;
    if true {
        x = 1;
    } else {
        x = 2;
    }
    print(x);
}
```
```execute
1
```

## Test: if missing else branch
```zong-program
func main() {
    var x: I64;
    if false {
        x = 1;
    }
    print(x);
}
```
```compile-error
error: variable 'x' may be used before assignment
```

## Test: nested if all paths initialize
```zong-program
func main() {
    var x: I64;
    if true {
        if true {
            x = 1;
        } else {
            x = 2;
        }
    } else {
        x = 3;
    }
    print(x);
}
```
```execute
1
```

## Test: loop with break before init
```zong-program
func main() {
    var x: I64;
    loop {
        break;
        x = 1;
    }
    print(x);
}
```
```compile-error
error: variable 'x' may be used before assignment
```


## Test: function parameter initialized
```zong-program
func test(x: I64): I64 {
    return x;
}

func main() {
    print(test(42));
}
```
```execute
42
```

## Test: return uninitialized variable
```zong-program
func test(): I64 {
    var x: I64;
    return x;
}

func main() {
    print(test());
}
```
```compile-error
error: variable 'x' used before assignment
```

## Test: address of uninitialized
```zong-program
func change(x: I64*) {
    x* = 42;
}

func main() {
    var x: I64;
    change(x&);
}
```
```compile-error
error: variable 'x' used before assignment
```

## Test: address of initialized
```zong-program
func change(x: I64*) {
    x* = 99;
}

func main() {
    var x: I64 = 42;
    change(x&);
    print(x);
}
```
```execute
99
```

## Test: multiple variables mixed init
```zong-program
func main() {
    var x: I64 = 10;
    var y: I64;
    var z: I64 = 30;
    print(x);
    y = 20;
    print(y);
    print(z);
}
```
```execute
10
20
30
```

## Test: variable shadowing
```zong-program
func main() {
    var x: I64 = 1;
    print(x);
    {
        var x: I64;
        x = 2;
        print(x);
    }
    print(x);
}
```
```execute
1
2
1
```

## Test: boolean uninitialized
```zong-program
func main() {
    var b: Boolean;
    if b {
        print(1);
    }
}
```
```compile-error
error: variable 'b' used before assignment
```

## Test: complex expression with uninitialized
```zong-program
func main() {
    var x: I64;
    var y: I64 = 10;
    print(x + y);
}
```
```compile-error
error: variable 'x' used before assignment
```

## Test: field access on uninitialized struct
```zong-program
struct Point(x: I64, y: I64);

func main() {
    var p: Point;
    var q: Point = Point(x: p.x, y: 10);
}
```
```compile-error
error: variable 'p' used before assignment
```


