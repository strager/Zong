### Test: Address-of expressions stored on stack at different offsets
```zong-program
{ var x I64; x = 5; print((x + 10)&); print((x * 2)&); }
```
```execute
0
8
```

### Test: Pointer to complex expression results
```zong-program
{ var a I64; var b I64; var c I64; var ptr I64*; a = 1; b = 2; c = 3; ptr = (a + b)&; print(ptr*); ptr = (b * c)&; print(ptr*); }
```
```execute
3
6
```

### Test: Chain of pointer assignments - modify through second pointer
```zong-program
{ var x I64; var ptr1 I64*; var ptr2 I64*; x = 50; ptr1 = x&; ptr2 = ptr1; ptr2* = 75; print(x); print(ptr1*); }
```
```execute
75
75
```

