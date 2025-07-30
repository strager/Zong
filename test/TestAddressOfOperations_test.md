### Test: address of variable
```zong-program
{ var x I64; x = 42; print(x&); }
```
```execute
0
```

### Test: multiple addressed variables
```zong-program
{ var x I64; var y I64; x = 10; y = 20; print(x&); print(y&); }
```
```execute
0
8
```

### Test: address of rvalue expression
```zong-program
{ var x I64; x = 5; print((x + 10)&); }
```
```execute
0
```

### Test: stack variable address access
```zong-program
{ var a I64; var b I64; a = 0; b = 0; print(a&); print(b&); print(a); print(b); }
```
```execute
0
8
0
0
```

