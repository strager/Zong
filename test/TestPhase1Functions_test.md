### Test: simple function call
```zong-program
func add(_ addA5: I64, _ addB5: I64): I64 { return addA5 + addB5; }
					 func main() { print(add(5, 3)); }
```
```execute
8
```

### Test: void function
```zong-program
func printTwice(_ printTwiceX2: I64) { print(printTwiceX2); print(printTwiceX2); }
					 func main() { printTwice(42); }
```
```execute
42
42
```

### Test: multiple function calls
```zong-program
func double(_ doubleX2: I64): I64 { return doubleX2 * 2; }
					 func triple(_ tripleX2: I64): I64 { return tripleX2 * 3; }
					 func main() { print(double(5)); print(triple(4)); }
```
```execute
10
12
```

### Test: nested function calls
```zong-program
func add(_ addA6: I64, _ addB6: I64): I64 { return addA6 + addB6; }
					 func multiply(_ multiplyA2: I64, _ multiplyB2: I64): I64 { return multiplyA2 * multiplyB2; }
					 func main() { print(add(multiply(2, 3), multiply(4, 5))); }
```
```execute
26
```

### Test: function with complex expression
```zong-program
func compute(_ computeA2: I64, _ computeB2: I64, _ computeC2: I64): I64 { return (computeA2 + computeB2) * computeC2 - 10; }
					 func main() { print(compute(3, 4, 5)); }
```
```execute
25
```

