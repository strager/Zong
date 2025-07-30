# Variable Shadowing Tests

## Test: variable shadowing end to end
```zong-program
func main() {
    var x I64;
    x = 10;
    print(x);
    {
        var x I64;
        x = 20;
        print(x);
    }
    print(x);
}
```
```execute
10
20
10
```

## Test: function parameter shadowing end to end
```zong-program
func test(x: I64) {
    print(x);
    {
        var x I64;
        x = 99;
        print(x);
    }
    print(x);
}

func main() {
    test(x: 42);
}
```
```execute
42
99
42
```

## Test: deep nested shadowing end to end
```zong-program
func main() {
    var x I64;
    x = 1;
    print(x);
    {
        var x I64;
        x = 2;
        print(x);
        {
            var x I64;
            x = 3;
            print(x);
            {
                var x I64;
                x = 4;
                print(x);
            }
            print(x);
        }
        print(x);
    }
    print(x);
}
```
```execute
1
2
3
4
3
2
1
```

## Test: shadowing with different types
```zong-program
func main() {
    var x I64;
    x = 42;
    print(x);
    {
        var x Boolean;
        x = true;
        print(x);
    }
    print(x);
}
```
```execute
42
1
42
```