# Print Bytes Function Tests

## Test: print_bytes with string literal
```zong-program
func main() {
    print_bytes("hello");
    print(42);
}
```
```execute
hello42
```

## Test: print_bytes with variable
```zong-program
func main() {
    var msg U8[] = "world";
    print_bytes(msg);
    print(999);
}
```
```execute
world999
```

## Test: print_bytes empty string
```zong-program
func main() {
    print_bytes("");
    print(0);
}
```
```execute
0
```