# Execution tests

## Test: simple expression execution
```zong-expr
print(42)
```
```execute
42
```

## Test: arithmetic expression execution
```zong-expr
print(2 + 3)
```
```execute
5
```

## Test: full program execution
```zong-program
func main() {
    var x I64 = 10;
    print(x * 2);
}
```
```execute
20
```

## Test: no output expected
```zong-program
func main() {
    var x I64 = 10;
    x = x * 2;
}
```
```execute

```

## Test: multiple print statements
```zong-program
func main() {
    print(1);
    print(2);
    print(3);
}
```
```execute
1
2
3
```