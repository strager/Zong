### Test: Basic pointer: assign address, dereference to read value
```zong-program
{ var x I64; var ptr I64*; x = 42; ptr = x&; print(ptr*); }
```
```execute
42
```

### Test: Modify pointee via pointer, read via original variable
```zong-program
{ var x I64; var ptr I64*; x = 10; ptr = x&; ptr* = 99; print(x); }
```
```execute
99
```

### Test: Modify via variable, read via pointer
```zong-program
{ var x I64; var ptr I64*; x = 25; ptr = x&; x = 77; print(ptr*); }
```
```execute
77
```

### Test: Use pointer dereference in arithmetic expression
```zong-program
{ var x I64; var ptr I64*; x = 7; ptr = x&; print(ptr* + 3); }
```
```execute
10
```

### Test: Multiple pointers to same variable - modify via one, read via another
```zong-program
{ var x I64; var ptr1 I64*; var ptr2 I64*; x = 123; ptr1 = x&; ptr2 = x&; print(ptr1*); print(ptr2*); ptr1* = 456; print(ptr2*); }
```
```execute
123
123
456
```

### Test: Sequential pointer operations on same variable
```zong-program
{ var x I64; var ptr I64*; x = 100; ptr = x&; print(ptr*); ptr* = 200; print(x); }
```
```execute
100
200
```

### Test: Use pointer dereference in complex expression: (8 * 7 + 6)
```zong-program
{ var x I64; var y I64; var ptr I64*; x = 8; y = 7; ptr = x&; print(ptr* * y + 6); }
```
```execute
62
```

### Test: Sequential modifications via pointer
```zong-program
{ var x I64; var ptr I64*; x = 5; ptr = x&; ptr* = ptr* + 1; print(x); ptr* = ptr* * 2; print(x); }
```
```execute
6
12
```

