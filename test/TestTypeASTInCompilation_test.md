### Test: I64 variable with TypeAST
```zong-program
{ var x I64; x = 42; print(x); }
```
```execute
42
```

### Test: Second I64 variable with TypeAST
```zong-program
{ var y I64; y = 7; print(y); }
```
```execute
7
```

### Test: Pointer variable with TypeAST
```zong-program
{ var ptr I64*; var x I64; x = 99; ptr = x&; print(ptr*); }
```
```execute
99
```

### Test: Multiple types with TypeAST
```zong-program
{ var x I64; var y I64; var ptr I64*; x = 10; y = 0; ptr = x&; print(x); print(y); print(ptr*); }
```
```execute
10
0
10
```

