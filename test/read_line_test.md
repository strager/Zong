# read_line() builtin function tests

## Test: read_line basic functionality
```zong-program
func main() {
    var line U8[];
    line = read_line();
    print_bytes(line);
}
```
```input
hello world

```
```execute
hello world
```

## Test: read_line multiple calls
```zong-program
func main() {
    var line1 U8[];
    var line2 U8[];
    line1 = read_line();
    line2 = read_line();
    print_bytes(line2);
    print_bytes(line1);
}
```
```input
first line
second line
```
```execute
second line
first line
```

## Test: read_line empty input
```zong-program
func main() {
    var line U8[];
    line = read_line();
    print_bytes(line);
}
```
```input
```
```execute
```
